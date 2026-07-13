package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestInitializedApplicationRefusesMissingOrIncompleteState(t *testing.T) {
	root := t.TempDir()
	if _, err := NewInitializedApplication(context.Background(), root); err == nil {
		t.Fatal("expected missing manifest refusal")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := InitializeStateConfig{StateRoot: root, PublicBaseURL: "https://app.example.test", Login: "alice", Password: []byte("a unique production password phrase 2026")}
	if _, err := InitializeState(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(ResolveStatePaths(root).ObjectBindingKey); err != nil {
		t.Fatal(err)
	}
	if _, err := NewInitializedApplication(context.Background(), root); err == nil {
		t.Fatal("expected missing binding key refusal")
	}
}

func TestInitializedApplicationUsesPersistentStoresAndIsReady(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config := InitializeStateConfig{StateRoot: root, PublicBaseURL: "https://app.example.test", Login: "alice", Password: []byte("a unique production password phrase 2026")}
	if _, err := InitializeState(context.Background(), config); err != nil {
		t.Fatal(err)
	}
	app, err := NewInitializedApplication(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Ready(context.Background()); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{"/healthz": http.StatusOK, "/readyz": http.StatusOK} {
		recorder := httptest.NewRecorder()
		initializedHandler(app, 1024).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://app.example.test"+path, nil))
		if recorder.Code != want {
			t.Fatalf("%s status=%d", path, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	app.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://app.example.test/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("root status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Header().Get("Set-Cookie"), "go_go_goja_session=") {
		t.Fatal("initialized product emitted unused gojahttp lightweight session cookie")
	}
	if err := app.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	paths := ResolveStatePaths(root)
	if _, err := os.Stat(paths.AppAuthDatabase); err != nil {
		t.Fatalf("persistent application auth database: %v", err)
	}
	if _, err := os.Stat(paths.ObjectRoot); err != nil {
		t.Fatalf("persistent object root: %v", err)
	}
}
