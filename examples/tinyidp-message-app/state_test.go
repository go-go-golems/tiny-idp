package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateRootPermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	now := time.Date(2026, 7, 14, 19, 0, 0, 0, time.UTC)
	manifest, err := initializeStateRoot(context.Background(), root, "http://127.0.0.1:8090/", now)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.PublicBaseURL != "http://127.0.0.1:8090" || manifest.Issuer != "http://127.0.0.1:8090/idp" || manifest.CreatedAt != now {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	paths := resolveStatePaths(root)
	for _, file := range []string{paths.Manifest, paths.TokenSecret, paths.SessionSecret} {
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s permissions = %#o, want 0600", file, info.Mode().Perm())
		}
	}
	for _, directory := range []string{paths.Root, filepath.Dir(paths.IdentityDatabase), filepath.Dir(paths.ApplicationDatabase), filepath.Dir(paths.AuditLog)} {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("%s permissions = %#o, want 0700", directory, info.Mode().Perm())
		}
	}
	if _, _, err := validateStateRoot(root); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeStateRootIsIdempotentAndDetectsConflict(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	first, err := initializeStateRoot(context.Background(), root, "https://messages.example.test", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	paths := resolveStatePaths(root)
	firstToken, _ := os.ReadFile(paths.TokenSecret)
	second, err := initializeStateRoot(context.Background(), root, "https://messages.example.test/", time.Unix(200, 0))
	if err != nil {
		t.Fatal(err)
	}
	secondToken, _ := os.ReadFile(paths.TokenSecret)
	if first.CreatedAt != second.CreatedAt || string(firstToken) != string(secondToken) {
		t.Fatal("idempotent initialization changed creation time or secret")
	}
	if _, err := initializeStateRoot(context.Background(), root, "https://other.example.test", time.Unix(300, 0)); err == nil {
		t.Fatal("expected conflicting origin to fail")
	}
}

func TestNormalizePublicBaseURLRejectsUnsafeOrigins(t *testing.T) {
	for _, raw := range []string{
		"", "http://example.test", "https://user@example.test", "https://example.test/path",
		"https://example.test/?query=1", "https://example.test/#fragment",
	} {
		if _, err := normalizePublicBaseURL(raw); err == nil {
			t.Errorf("normalizePublicBaseURL(%q) succeeded", raw)
		}
	}
	for _, raw := range []string{"http://localhost:8080", "http://[::1]:8080", "https://example.test"} {
		if _, err := normalizePublicBaseURL(raw); err != nil {
			t.Errorf("normalizePublicBaseURL(%q): %v", raw, err)
		}
	}
}

func TestValidateStateRootRejectsDamagedSecret(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	if _, err := initializeStateRoot(context.Background(), root, "http://localhost:8090", time.Now()); err != nil {
		t.Fatal(err)
	}
	paths := resolveStatePaths(root)
	if err := os.WriteFile(paths.SessionSecret, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := validateStateRoot(root); err == nil {
		t.Fatal("expected damaged secret validation failure")
	}
}
