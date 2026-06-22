package client

import (
	"reflect"
	"testing"
)

// TestMergePreservesBuiltinProperties is the core merge contract: when a
// configured client matches a builtin, the builtin's RequirePKCE, Secret,
// and AllowedScopes are preserved, and the configured redirect URIs are
// unioned onto the builtin's.
func TestMergePreservesBuiltinProperties(t *testing.T) {
	base := Client{
		ID:           "public-spa",
		Secret:       "",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		RequirePKCE:  true,
		AllowedScopes: []string{"openid", "profile", "email"},
	}
	override := Client{
		ID:           "public-spa",
		Secret:       "",
		RedirectURIs: []string{"http://localhost:9090/cb"},
	}
	got := Merge(base, override)

	if !got.RequirePKCE {
		t.Fatal("Merge must preserve base.RequirePKCE")
	}
	if got.Secret != "" {
		t.Fatalf("Merge preserved wrong secret: %q", got.Secret)
	}
	if !reflect.DeepEqual(got.AllowedScopes, base.AllowedScopes) {
		t.Fatalf("Merge must preserve base.AllowedScopes: got %v", got.AllowedScopes)
	}
	// Redirect URIs unioned: builtin's first, then configured (deduplicated).
	want := []string{"http://localhost:8080/callback", "http://localhost:9090/cb"}
	if !reflect.DeepEqual(got.RedirectURIs, want) {
		t.Fatalf("RedirectURIs = %v, want %v", got.RedirectURIs, want)
	}
}

// TestMergeDeduplicatesRedirectURIs verifies that a configured redirect URI
// already present in the base is not duplicated.
func TestMergeDeduplicatesRedirectURIs(t *testing.T) {
	base := Client{ID: "dev-client", RedirectURIs: []string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback"}}
	override := Client{
		ID:           "dev-client",
		RedirectURIs: []string{"http://localhost:3000/callback", "http://localhost:8080/cb"},
	}
	got := Merge(base, override)
	want := []string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback", "http://localhost:8080/cb"}
	if !reflect.DeepEqual(got.RedirectURIs, want) {
		t.Fatalf("RedirectURIs = %v, want %v", got.RedirectURIs, want)
	}
}

// TestMergeNonEmptySecretOverrides verifies that a configured non-empty
// secret overrides the builtin's, so --client-id web-app --client-secret X
// yields a web-app client with secret X (not the builtin dev-secret).
func TestMergeNonEmptySecretOverrides(t *testing.T) {
	base := Client{ID: "web-app", Secret: "dev-secret"}
	override := Client{ID: "web-app", Secret: "custom-secret"}
	got := Merge(base, override)
	if got.Secret != "custom-secret" {
		t.Fatalf("Secret = %q, want custom-secret", got.Secret)
	}
}

// TestMergeEmptySecretKeepsBuiltin verifies that an empty configured secret
// (the common case: --client-id web-app with no --client-secret) keeps the
// builtin's secret, so web-app stays confidential with dev-secret.
func TestMergeEmptySecretKeepsBuiltin(t *testing.T) {
	base := Client{ID: "web-app", Secret: "dev-secret"}
	override := Client{ID: "web-app", Secret: ""}
	got := Merge(base, override)
	if got.Secret != "dev-secret" {
		t.Fatalf("Secret = %q, want dev-secret (kept from base)", got.Secret)
	}
}

// TestMergeTakesOverrideID verifies the override's ID wins (it is the
// configured client_id).
func TestMergeTakesOverrideID(t *testing.T) {
	base := Client{ID: "public-spa"}
	override := Client{ID: "public-spa"}
	got := Merge(base, override)
	if got.ID != "public-spa" {
		t.Fatalf("ID = %q, want public-spa", got.ID)
	}
}
