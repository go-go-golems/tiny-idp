package idpstore

import (
	"errors"
	"testing"
)

func TestClientValidateRedirectURIs(t *testing.T) {
	valid := Client{ID: "web", SecretHash: []byte("hash"), RedirectURIs: []string{"https://app.example/callback"}, AllowedScopes: []string{"openid"}}
	if err := valid.Validate(ProductionMode); err != nil {
		t.Fatalf("valid client rejected: %v", err)
	}

	cases := []struct {
		name string
		uri  string
		want error
	}{
		{"wildcard", "https://*.example/callback", ErrWildcardRedirectURI},
		{"fragment", "https://app.example/callback#frag", ErrRedirectURIFragment},
		{"invalid", "://bad", ErrInvalidRedirectURI},
		{"http production", "http://app.example/callback", ErrProductionRedirectHTTP},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Client{ID: "web", SecretHash: []byte("hash"), RedirectURIs: []string{tc.uri}}
			if err := c.Validate(ProductionMode); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestClientExactRedirectMatch(t *testing.T) {
	c := Client{RedirectURIs: []string{"https://app.example/callback"}}
	if !c.AllowsRedirectURI("https://app.example/callback") {
		t.Fatal("exact URI should match")
	}
	if c.AllowsRedirectURI("https://app.example/callback/extra") {
		t.Fatal("prefix attack should not match")
	}
}

func TestPublicClientRequiresPKCE(t *testing.T) {
	c := Client{ID: "spa", Public: true, RedirectURIs: []string{"https://app.example/callback"}}
	if err := c.Validate(ProductionMode); !errors.Is(err, ErrPublicClientRequiresPKCE) {
		t.Fatalf("got %v", err)
	}
}

func TestParseScopesAndClaims(t *testing.T) {
	scopes := ParseScopes("openid email profile email")
	if len(scopes) != 3 {
		t.Fatalf("expected deduped scopes, got %#v", scopes)
	}
	u := User{Sub: "user-1", Email: "a@example.test", EmailVerified: true, Name: "Alice", Groups: []string{"admin"}}
	claims := ClaimsForScopes(u, scopes)
	if claims["email"] != "a@example.test" || claims["name"] != "Alice" {
		t.Fatalf("claims not filtered as expected: %#v", claims)
	}
	claims = ClaimsForScopes(u, []string{"openid"})
	if _, ok := claims["email"]; ok {
		t.Fatalf("email should require email scope: %#v", claims)
	}
}

func TestUserSubjectStableNotEmail(t *testing.T) {
	if err := (User{Sub: "alice@example.test", Email: "alice@example.test"}).Validate(); !errors.Is(err, ErrSubjectUsesEmail) {
		t.Fatalf("got %v", err)
	}
	if err := (User{Sub: "user-123", Email: "alice@example.test"}).Validate(); err != nil {
		t.Fatalf("valid user rejected: %v", err)
	}
}
