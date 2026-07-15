package main

import "testing"

func TestExternalOIDCConfigValidation(t *testing.T) {
	valid := externalOIDCConfig{PublicBaseURL: "http://127.0.0.1:8080", Issuer: "http://127.0.0.1:8081", ClientID: "message-desk", EndSessionEndpoint: "http://127.0.0.1:8081/end-session"}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid external configuration: %v", err)
	}
	for _, config := range []externalOIDCConfig{
		{PublicBaseURL: valid.PublicBaseURL, Issuer: valid.PublicBaseURL, ClientID: valid.ClientID},
		{PublicBaseURL: valid.PublicBaseURL, Issuer: valid.Issuer, ClientID: ""},
		{PublicBaseURL: valid.PublicBaseURL, Issuer: valid.Issuer, ClientID: valid.ClientID, CookieSecure: true},
		{PublicBaseURL: valid.PublicBaseURL, Issuer: valid.Issuer, ClientID: valid.ClientID, EndSessionEndpoint: "http://127.0.0.1:8090/end-session"},
		{PublicBaseURL: valid.PublicBaseURL, Issuer: valid.Issuer, ClientID: valid.ClientID, BackchannelURL: "http://idp:8081/other"},
	} {
		if err := config.validate(); err == nil {
			t.Fatalf("invalid external configuration accepted: %#v", config)
		}
	}
}

func TestNormalizeExternalIssuer(t *testing.T) {
	if got, err := normalizeExternalIssuer("http://127.0.0.1:8081/idp/"); err != nil || got != "http://127.0.0.1:8081/idp" {
		t.Fatalf("normalize issuer = %q, %v", got, err)
	}
	for _, raw := range []string{"https://issuer.example.test/?query=x", "https://user@issuer.example.test", "https://issuer.example.test//idp"} {
		if _, err := normalizeExternalIssuer(raw); err == nil {
			t.Fatalf("invalid issuer %q accepted", raw)
		}
	}
}
