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
}
