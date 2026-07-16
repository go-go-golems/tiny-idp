package fositeadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestAuthorizeRequiresCSRFAndEmitsAudit(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	sink := idp.NewMemorySink()
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("hardening-secret-key-32-bytes!!!!"), Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
	resp, err := http.Post(ts.URL+"/authorize", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	found := false
	for _, e := range sink.Events() {
		if e.Name == "login.csrf_rejected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("csrf audit event not found: %#v", sink.Events())
	}
}

func TestAuditReasonsUseStableCodes(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	sink := idp.NewMemorySink()
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("audit-reason-secret-32-bytes!!!!"), Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.PostForm(ts.URL+"/token", url.Values{"grant_type": {"authorization_code"}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("token request unexpectedly succeeded")
	}
	for _, e := range sink.Events() {
		if e.Name == "token.request.rejected" {
			if e.Reason == "" || strings.Contains(e.Reason, " ") || strings.Contains(e.Reason, "(") {
				t.Fatalf("unstable audit reason: %#v", e)
			}
			return
		}
	}
	t.Fatalf("token rejection audit event not found: %#v", sink.Events())
}

func TestSecurityHeadersOnDiscovery(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("headers-secret-key-32-bytes!!!!!!")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame deny header")
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatalf("missing CSP")
	}
}

func TestDiscoveryPublishesIntrospectionAtRootAndPathIssuer(t *testing.T) {
	for _, tc := range []struct {
		name              string
		issuer            string
		discoveryPath     string
		wantIntrospection string
	}{
		{name: "root issuer", issuer: "https://issuer.example.test", discoveryPath: "/.well-known/openid-configuration", wantIntrospection: "https://issuer.example.test/introspect"},
		{name: "path issuer", issuer: "https://issuer.example.test/idp", discoveryPath: "/idp/.well-known/openid-configuration", wantIntrospection: "https://issuer.example.test/idp/introspect"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := memory.New()
			key, err := keys.GenerateRSA("kid-discovery", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := st.CreateSigningKey(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			provider, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: tc.issuer, Store: st, SecretKey: []byte("discovery-introspection-key-32-bytes")})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(provider.Handler())
			defer server.Close()
			response, err := http.Get(server.URL + tc.discoveryPath)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("discovery status=%d", response.StatusCode)
			}
			var discovery struct {
				IntrospectionEndpoint string   `json:"introspection_endpoint"`
				AuthMethods           []string `json:"introspection_endpoint_auth_methods_supported"`
			}
			if err := json.NewDecoder(response.Body).Decode(&discovery); err != nil {
				t.Fatal(err)
			}
			if discovery.IntrospectionEndpoint != tc.wantIntrospection || len(discovery.AuthMethods) != 1 || discovery.AuthMethods[0] != "client_secret_basic" {
				t.Fatalf("discovery introspection contract=%#v", discovery)
			}
		})
	}
}
