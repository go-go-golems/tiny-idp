package idpstore

import (
	"errors"
	"testing"
)

func TestClientValidateRedirectURIs(t *testing.T) {
	valid := Client{ID: "web", SecretHash: []byte("hash"), RedirectURIs: []string{"https://app.example/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{GrantAuthorizationCode, GrantRefreshToken}}
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
			c := Client{ID: "web", SecretHash: []byte("hash"), RedirectURIs: []string{tc.uri}, AllowedGrantTypes: []string{GrantAuthorizationCode}}
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
	c := Client{ID: "spa", Public: true, RedirectURIs: []string{"https://app.example/callback"}, AllowedGrantTypes: []string{GrantAuthorizationCode}}
	if err := c.Validate(ProductionMode); !errors.Is(err, ErrPublicClientRequiresPKCE) {
		t.Fatalf("got %v", err)
	}
}

func TestClientGrantCapabilitiesRequireExplicitKnownUniqueValues(t *testing.T) {
	base := Client{ID: "client", SecretHash: []byte("hash"), AllowedGrantTypes: []string{GrantAuthorizationCode}}
	if !base.AllowsGrantType(GrantAuthorizationCode) || base.AllowsGrantType(GrantDeviceCode) {
		t.Fatalf("grant capability lookup returned unexpected result")
	}
	cases := []struct {
		name   string
		grants []string
		want   error
	}{
		{name: "missing", want: ErrClientMissingGrantTypes},
		{name: "unknown", grants: []string{"client_credentials"}, want: ErrClientGrantTypeInvalid},
		{name: "duplicate", grants: []string{GrantAuthorizationCode, GrantAuthorizationCode}, want: ErrClientGrantTypeDuplicate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := base
			client.AllowedGrantTypes = tc.grants
			if err := client.Validate(ProductionMode); !errors.Is(err, tc.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestClientAudienceAndIntrospectionCapabilitiesFailClosed(t *testing.T) {
	client := Client{ID: "resource", SecretHash: []byte("hash"), AllowedGrantTypes: []string{GrantAuthorizationCode}, AllowedAudiences: []string{"https://api.example.test"}, CanIntrospect: true}
	if err := client.Validate(ProductionMode); err != nil {
		t.Fatalf("valid resource client rejected: %v", err)
	}
	if !client.AllowsAudience([]string{"https://api.example.test"}) || client.AllowsAudience([]string{"https://other.example.test"}) {
		t.Fatal("audience capability did not fail closed")
	}
	public := client
	public.Public = true
	public.RequirePKCE = true
	public.SecretHash = nil
	if err := public.Validate(ProductionMode); err == nil {
		t.Fatal("public introspection client was accepted")
	}
	noAudience := client
	noAudience.AllowedAudiences = nil
	if err := noAudience.Validate(ProductionMode); err == nil {
		t.Fatal("introspection client without audience was accepted")
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
