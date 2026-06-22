package client

import (
	"testing"
)

func TestNewRegistryPreloadsBuiltinClients(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"dev-client", "public-spa", "web-app"} {
		c, ok := r.Lookup(id)
		if !ok {
			t.Fatalf("builtin client %q missing", id)
		}
		if c.ID != id {
			t.Fatalf("client ID mismatch: got %q want %q", c.ID, id)
		}
	}
}

func TestPublicSpaRequiresPKCE(t *testing.T) {
	r := NewRegistry()
	c, _ := r.Lookup("public-spa")
	if !c.RequirePKCE {
		t.Fatal("public-spa must require PKCE")
	}
	if c.Secret != "" {
		t.Fatal("public-spa must be a public client (no secret)")
	}
}

func TestWebAppIsConfidential(t *testing.T) {
	r := NewRegistry()
	c, _ := r.Lookup("web-app")
	if c.Secret == "" {
		t.Fatal("web-app must be a confidential client (secret required)")
	}
	if c.RequirePKCE {
		t.Fatal("web-app should not require PKCE")
	}
}

func TestDevClientIsPermissive(t *testing.T) {
	r := NewRegistry()
	c, _ := r.Lookup("dev-client")
	if c.Secret != "" {
		t.Fatal("dev-client should be public (no secret) by default")
	}
	if c.RequirePKCE {
		t.Fatal("dev-client should not require PKCE")
	}
	// Empty AllowedScopes means all scopes allowed (permissive).
	if !c.AllowsScope("openid profile email some-extra") {
		t.Fatal("dev-client should allow any scope")
	}
}

func TestAllowsRedirectURI(t *testing.T) {
	c := Client{
		ID:           "x",
		RedirectURIs: []string{"http://localhost:8080/cb", "http://127.0.0.1:8080/cb"},
	}
	if !c.AllowsRedirectURI("http://localhost:8080/cb") {
		t.Fatal("expected allowed redirect URI")
	}
	if c.AllowsRedirectURI("https://evil.test/cb") {
		t.Fatal("disallowed redirect URI must be rejected")
	}
}

func TestAllowsScopeWithAllowlist(t *testing.T) {
	c := Client{
		ID:            "x",
		AllowedScopes: []string{"openid", "profile", "email"},
	}
	if !c.AllowsScope("openid profile") {
		t.Fatal("allowed scopes should be accepted")
	}
	if c.AllowsScope("openid offline_access") {
		t.Fatal("offline_access should be rejected when not in allowlist")
	}
}

func TestRegisterAddsOrReplaces(t *testing.T) {
	r := NewRegistry()
	r.Register(Client{ID: "dev-client", Secret: "overridden"})
	c, _ := r.Lookup("dev-client")
	if c.Secret != "overridden" {
		t.Fatalf("Register should replace: secret = %q", c.Secret)
	}
	r.Register(Client{ID: "custom", RedirectURIs: []string{"http://x/cb"}})
	if _, ok := r.Lookup("custom"); !ok {
		t.Fatal("Register should add a new client")
	}
}
