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
