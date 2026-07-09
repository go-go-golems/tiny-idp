package oidcmeta_test

import (
	"testing"

	"github.com/manuel/tinyidp/internal/oidcmeta"
)

func TestProductionDiscoveryOmitsUnimplementedEndSessionEndpoint(t *testing.T) {
	discovery, err := oidcmeta.ProductionDiscovery("https://issuer.example.test")
	if err != nil {
		t.Fatal(err)
	}
	if discovery.EndSessionEndpoint != "" {
		t.Fatalf("end_session_endpoint = %q, want omitted until strict adapter implements /end-session", discovery.EndSessionEndpoint)
	}
}
