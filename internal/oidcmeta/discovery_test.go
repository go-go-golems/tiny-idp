package oidcmeta

import "testing"

func TestDiscoveryPath(t *testing.T) {
	iss, err := ParseIssuer("https://example.com/idp/")
	if err != nil {
		t.Fatal(err)
	}
	if got := iss.DiscoveryPath(); got != "/idp/.well-known/openid-configuration" {
		t.Fatalf("got %s", got)
	}
	d, err := ProductionDiscovery("https://example.com/idp")
	if err != nil {
		t.Fatal(err)
	}
	if d.Issuer != "https://example.com/idp" || d.AuthorizationEndpoint != "https://example.com/idp/authorize" {
		t.Fatalf("bad discovery: %#v", d)
	}
	for _, m := range d.CodeChallengeMethodsSupported {
		if m == "plain" {
			t.Fatal("production discovery must not advertise plain PKCE")
		}
	}
}
