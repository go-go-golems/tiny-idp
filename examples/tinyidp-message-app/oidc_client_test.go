package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestBeginLoginPersistsPKCEStateAndNonce(t *testing.T) {
	store, err := openAppStore(context.Background(), filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	client := &oidcClient{config: oauth2.Config{
		ClientID: clientID, RedirectURL: "http://127.0.0.1:8090" + callbackPath,
		Endpoint: oauth2.Endpoint{AuthURL: "http://127.0.0.1:8090/idp/authorize"},
	}, now: func() time.Time { return now }}
	authorizationURL, err := client.beginLogin(context.Background(), store, "/messages")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("state") == "" || query.Get("nonce") == "" || query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("incomplete authorization URL: %s", authorizationURL)
	}
	if query.Get("prompt") != "select_account" {
		t.Fatalf("authorization URL prompt = %q, want select_account", query.Get("prompt"))
	}
	attempt, err := store.consumeLoginAttempt(context.Background(), query.Get("state"), now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Nonce != query.Get("nonce") || attempt.PKCEVerifier == "" || attempt.ReturnTo != "/messages" {
		t.Fatalf("stored attempt does not match authorization URL: %#v", attempt)
	}
}

func TestReturnToRejectsExternalAndAmbiguousPaths(t *testing.T) {
	for _, raw := range []string{"https://example.test", "//example.test", "/a/../b", "/%2fetc", "/%2e%2e/x", "/a\\b", "/?next=x"} {
		if _, err := normalizeReturnTo(raw); err == nil {
			t.Errorf("normalizeReturnTo(%q) succeeded", raw)
		}
	}
	for _, raw := range []string{"", "/", "/messages", "/account/settings"} {
		if _, err := normalizeReturnTo(raw); err != nil {
			t.Errorf("normalizeReturnTo(%q): %v", raw, err)
		}
	}
}

func TestIssuerRewriteTransportRoutesOnlyTheNetworkDestination(t *testing.T) {
	issuer, err := url.Parse("https://issuer.example.test/idp")
	if err != nil {
		t.Fatal(err)
	}
	backchannel, err := url.Parse("http://idp:8081/private-idp")
	if err != nil {
		t.Fatal(err)
	}
	var captured *http.Request
	transport := &issuerRewriteTransport{
		issuer: issuer, backchannel: backchannel,
		base: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}")), Request: request}, nil
		}),
	}
	request, err := http.NewRequest(http.MethodGet, "https://issuer.example.test/idp/.well-known/openid-configuration?probe=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if captured == nil {
		t.Fatal("base transport did not receive a request")
	}
	if got, want := captured.URL.String(), "http://idp:8081/private-idp/.well-known/openid-configuration?probe=1"; got != want {
		t.Fatalf("backchannel request URL = %q, want %q", got, want)
	}
	if got, want := captured.Host, "issuer.example.test"; got != want {
		t.Fatalf("backchannel request Host = %q, want public issuer host %q", got, want)
	}
	if got, want := request.URL.String(), "https://issuer.example.test/idp/.well-known/openid-configuration?probe=1"; got != want {
		t.Fatalf("original request mutated to %q, want %q", got, want)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
