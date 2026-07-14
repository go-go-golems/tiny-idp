package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupDeletesOnlyExpiredTerminalState(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	old := now.Add(-2 * time.Hour)
	future := now.Add(time.Hour)
	if err := store.createLoginAttempt(ctx, "old-login", loginAttempt{Nonce: "n", PKCEVerifier: "v", ReturnTo: "/", CreatedAt: old.Add(-time.Minute), ExpiresAt: old}); err != nil {
		t.Fatal(err)
	}
	if err := store.createLoginAttempt(ctx, "fresh-login", loginAttempt{Nonce: "n", PKCEVerifier: "v", ReturnTo: "/", CreatedAt: now, ExpiresAt: future}); err != nil {
		t.Fatal(err)
	}
	if err := store.createRegistrationAttempt(ctx, "old-registration", registrationAttempt{CSRFSecret: make([]byte, 32), CreatedAt: old.Add(-time.Minute), ExpiresAt: old}); err != nil {
		t.Fatal(err)
	}
	if err := store.createAppSession(ctx, "old-session", appSession{Subject: "s", DisplayName: "n", CSRFSecret: make([]byte, 32), CreatedAt: old.Add(-time.Hour), ExpiresAt: old}); err != nil {
		t.Fatal(err)
	}
	report, err := store.cleanup(ctx, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if report.LoginAttempts != 1 || report.RegistrationAttempts != 1 || report.Sessions != 1 {
		t.Fatalf("cleanup report = %#v", report)
	}
	if _, err := store.consumeLoginAttempt(ctx, "fresh-login", now); err != nil {
		t.Fatalf("fresh login attempt was deleted: %v", err)
	}
}
