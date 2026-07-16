package resourceauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthenticateValidatesAndCachesConstrainedPrincipal(t *testing.T) {
	now := time.Date(2026, 7, 16, 19, 0, 0, 0, time.UTC)
	var introspectionCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, discoveryDocument{Issuer: serverURL(r), IntrospectionEndpoint: serverURL(r) + "/introspect", IntrospectionAuthMethodsSupported: []string{"client_secret_basic"}})
		case "/introspect":
			introspectionCalls.Add(1)
			clientID, secret, ok := r.BasicAuth()
			if !ok || clientID != "xapp-api" || secret != "resource-secret" {
				t.Fatalf("introspection Basic authentication = %q/%q/%v", clientID, secret, ok)
			}
			if err := r.ParseForm(); err != nil || r.Form.Get("token") != "opaque-good" {
				t.Fatalf("introspection form = %#v, err = %v", r.Form, err)
			}
			writeJSON(t, w, introspectionResponse{Active: true, Issuer: serverURL(r), Subject: "subject-alice", ClientID: "xapp-cli", Scope: "bbs.read bbs.post.create", Audience: []string{"https://app.example.test/api"}, Expires: now.Add(time.Hour).Unix(), TokenType: "Bearer"})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	authenticator := newTestAuthenticator(t, server.URL, now, server.Client())
	first := authenticator.Authenticate(context.Background(), []string{"Bearer opaque-good"}, []string{"bbs.read"})
	if first.Outcome != OutcomeAuthenticated || first.Principal.Subject != "subject-alice" {
		t.Fatalf("first Authenticate = %#v", first)
	}
	first.Principal.Scopes[0] = "mutated"
	second := authenticator.Authenticate(context.Background(), []string{"Bearer opaque-good"}, []string{"bbs.post.create"})
	if second.Outcome != OutcomeAuthenticated || second.Principal.Scopes[0] != "bbs.read" {
		t.Fatalf("cached Authenticate = %#v", second)
	}
	if got := introspectionCalls.Load(); got != 1 {
		t.Fatalf("introspection calls = %d, want 1", got)
	}
}

func TestAuthenticateRejectsSecurityInvariantFailures(t *testing.T) {
	now := time.Date(2026, 7, 16, 19, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		authorization []string
		response      introspectionResponse
		required      []string
		want          Outcome
	}{
		{name: "missing bearer", authorization: nil, want: OutcomeUnauthorized},
		{name: "multiple authorization headers", authorization: []string{"Bearer one", "Bearer two"}, want: OutcomeUnauthorized},
		{name: "wrong scheme", authorization: []string{"Basic opaque"}, want: OutcomeUnauthorized},
		{name: "inactive", authorization: []string{"Bearer opaque-inactive"}, response: introspectionResponse{Active: false}, want: OutcomeUnauthorized},
		{name: "wrong issuer", authorization: []string{"Bearer opaque-issuer"}, response: activeResponse("https://other.example.test/idp", now), want: OutcomeUnauthorized},
		{name: "wrong audience", authorization: []string{"Bearer opaque-audience"}, response: withAudience(activeResponse("", now), "https://other.example.test/api"), want: OutcomeUnauthorized},
		{name: "wrong token type", authorization: []string{"Bearer opaque-type"}, response: withTokenType(activeResponse("", now), "DPoP"), want: OutcomeUnauthorized},
		{name: "expired", authorization: []string{"Bearer opaque-expired"}, response: withExpiry(activeResponse("", now), now.Add(-time.Second)), want: OutcomeUnauthorized},
		{name: "missing scope", authorization: []string{"Bearer opaque-scope"}, response: activeResponse("", now), required: []string{"bbs.post.create"}, want: OutcomeForbidden},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var issuer string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/.well-known/openid-configuration":
					writeJSON(t, w, discoveryDocument{Issuer: issuer, IntrospectionEndpoint: issuer + "/introspect", IntrospectionAuthMethodsSupported: []string{"client_secret_basic"}})
				case "/introspect":
					response := testCase.response
					if response.Issuer == "" {
						response.Issuer = issuer
					}
					writeJSON(t, w, response)
				default:
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
			}))
			defer server.Close()
			issuer = server.URL
			authenticator := newTestAuthenticator(t, issuer, now, server.Client())
			result := authenticator.Authenticate(context.Background(), testCase.authorization, testCase.required)
			if result.Outcome != testCase.want {
				t.Fatalf("Authenticate outcome = %q, want %q", result.Outcome, testCase.want)
			}
		})
	}
}

func TestAuthenticatorTreatsProviderFailuresAsUnavailableAndDoesNotCacheThem(t *testing.T) {
	now := time.Date(2026, 7, 16, 19, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	var issuer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, discoveryDocument{Issuer: issuer, IntrospectionEndpoint: issuer + "/introspect", IntrospectionAuthMethodsSupported: []string{"client_secret_basic"}})
		case "/introspect":
			calls.Add(1)
			http.Error(w, `{"error":"temporarily_unavailable"}`, http.StatusTooManyRequests)
		}
	}))
	defer server.Close()
	issuer = server.URL
	authenticator := newTestAuthenticator(t, issuer, now, server.Client())
	for range 2 {
		result := authenticator.Authenticate(context.Background(), []string{"Bearer opaque"}, nil)
		if result.Outcome != OutcomeUnavailable {
			t.Fatalf("Authenticate outcome = %q, want unavailable", result.Outcome)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("provider failures were cached: calls = %d, want 2", got)
	}
}

func TestNewRejectsWrongDiscoveryContract(t *testing.T) {
	cases := []struct {
		name     string
		document discoveryDocument
	}{
		{name: "wrong issuer", document: discoveryDocument{Issuer: "https://wrong.example.test", IntrospectionEndpoint: "https://wrong.example.test/introspect", IntrospectionAuthMethodsSupported: []string{"client_secret_basic"}}},
		{name: "no basic", document: discoveryDocument{IntrospectionAuthMethodsSupported: []string{"client_secret_post"}}},
		{name: "foreign endpoint", document: discoveryDocument{IntrospectionEndpoint: "https://foreign.example.test/introspect", IntrospectionAuthMethodsSupported: []string{"client_secret_basic"}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				document := testCase.document
				if document.Issuer == "" {
					document.Issuer = serverURL(r)
				}
				if document.IntrospectionEndpoint == "" {
					document.IntrospectionEndpoint = serverURL(r) + "/introspect"
				}
				writeJSON(t, w, document)
			}))
			defer server.Close()
			_, err := New(context.Background(), Config{IssuerURL: server.URL, ClientID: "api", ClientSecret: []byte("secret"), Audience: "https://app.example.test/api", HTTPClient: server.Client()})
			if err == nil {
				t.Fatal("New succeeded with invalid discovery")
			}
		})
	}
}

func TestParseBearer(t *testing.T) {
	cases := []struct {
		values []string
		want   string
		ok     bool
	}{
		{values: []string{"Bearer opaque"}, want: "opaque", ok: true},
		{values: []string{"bearer opaque"}, want: "opaque", ok: true},
		{values: []string{"Bearer"}, ok: false},
		{values: []string{"Bearer one two"}, ok: false},
		{values: []string{"Bearer one", "Bearer two"}, ok: false},
	}
	for _, testCase := range cases {
		got, ok := parseBearer(testCase.values)
		if got != testCase.want || ok != testCase.ok {
			t.Fatalf("parseBearer(%q) = %q, %v; want %q, %v", testCase.values, got, ok, testCase.want, testCase.ok)
		}
	}
}

func newTestAuthenticator(t *testing.T, issuer string, now time.Time, client *http.Client) *Authenticator {
	t.Helper()
	authenticator, err := New(context.Background(), Config{
		IssuerURL: issuer, ClientID: "xapp-api", ClientSecret: []byte("resource-secret"), Audience: "https://app.example.test/api",
		HTTPClient: client, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return authenticator
}

func activeResponse(issuer string, now time.Time) introspectionResponse {
	return introspectionResponse{Active: true, Issuer: issuer, Subject: "subject", ClientID: "cli", Scope: "bbs.read", Audience: []string{"https://app.example.test/api"}, Expires: now.Add(time.Hour).Unix(), TokenType: "Bearer"}
}

func withAudience(response introspectionResponse, audience string) introspectionResponse {
	response.Audience = []string{audience}
	return response
}

func withTokenType(response introspectionResponse, tokenType string) introspectionResponse {
	response.TokenType = tokenType
	return response
}

func withExpiry(response introspectionResponse, expiry time.Time) introspectionResponse {
	response.Expires = expiry.Unix()
	return response
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return (&url.URL{Scheme: scheme, Host: r.Host}).String()
}

var _ = strings.TrimSpace
