package oidcmeta_test

import (
	"testing"

	"github.com/manuel/tinyidp/internal/oidcmeta"
)

func TestProductionDiscoveryIncludesEndSessionEndpoint(t *testing.T) {
	discovery, err := oidcmeta.ProductionDiscovery("https://issuer.example.test")
	if err != nil {
		t.Fatal(err)
	}
	if discovery.EndSessionEndpoint != "https://issuer.example.test/end-session" {
		t.Fatalf("end_session_endpoint = %q", discovery.EndSessionEndpoint)
	}
	if discovery.IntrospectionEndpoint != "https://issuer.example.test/introspect" || len(discovery.IntrospectionEndpointAuthMethodsSupported) != 1 || discovery.IntrospectionEndpointAuthMethodsSupported[0] != "client_secret_basic" {
		t.Fatalf("introspection discovery = %#v", discovery)
	}
}
