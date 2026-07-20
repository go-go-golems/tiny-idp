package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestInitializedMessageApplicationExposesReadyHealthEndpoints(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	if _, err := initializeStateRoot(ctx, root, "http://127.0.0.1:8090", time.Now()); err != nil {
		t.Fatal(err)
	}
	app, err := openInitializedMessageApplication(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Close(context.Background()) }()
	if _, err := app.provider.RunMaintenance(ctx); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		response := httptest.NewRecorder()
		app.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
		}
	}
}

func TestParseServeDurationsRejectsNonPositiveValues(t *testing.T) {
	t.Parallel()
	_, err := parseServeDurations(serveSettings{MaintenanceInterval: "0s", ShutdownTimeout: "20s", ReadHeaderTimeout: "5s", ReadTimeout: "15s", WriteTimeout: "30s", IdleTimeout: "1m"})
	if err == nil {
		t.Fatal("parseServeDurations accepted zero maintenance interval")
	}
}

func TestMessageListenerModesAreExplicitAndMutuallyExclusive(t *testing.T) {
	if _, err := parseMessageListenerMode(""); err == nil {
		t.Fatal("empty listener mode accepted")
	}
	direct, err := parseMessageListenerMode("direct-tls")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateMessageListenerSettings(direct, serveSettings{TLSCertificate: "cert.pem", TLSKey: "key.pem"}, "https://message.example.test"); err != nil {
		t.Fatalf("direct TLS rejected: %v", err)
	}
	proxy, err := parseMessageListenerMode("trusted-proxy-http")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateMessageListenerSettings(proxy, serveSettings{TrustedProxyCIDRs: []string{"10.42.0.0/24"}}, "https://message.example.test"); err != nil {
		t.Fatalf("trusted proxy rejected: %v", err)
	}
	if err := validateMessageListenerSettings(proxy, serveSettings{TrustedProxyCIDRs: []string{"10.42.0.0/24"}, TLSCertificate: "cert.pem"}, "https://message.example.test"); err == nil {
		t.Fatal("trusted proxy accepted certificate")
	}
	if err := validateMessageListenerSettings(proxy, serveSettings{TrustedProxyCIDRs: []string{"10.42.0.0/24"}}, "http://127.0.0.1:8090"); err == nil {
		t.Fatal("trusted proxy accepted HTTP public origin")
	}
}
