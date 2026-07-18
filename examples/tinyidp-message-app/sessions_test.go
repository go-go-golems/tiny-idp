package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStoresOnlyTokenHash(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	rawToken := "browser-session-secret"
	desired := appSession{
		Subject: "user-123", DisplayName: "Alice", CSRFSecret: bytes.Repeat([]byte{7}, sha256.Size),
		CreatedAt: now, ExpiresAt: now.Add(8 * time.Hour),
	}
	if err := store.createAppSession(ctx, rawToken, desired); err != nil {
		t.Fatal(err)
	}
	var storedHash []byte
	if err := store.db.QueryRowContext(ctx, "SELECT token_hash FROM app_sessions").Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte(rawToken))
	if !bytes.Equal(storedHash, expected[:]) || bytes.Equal(storedHash, []byte(rawToken)) {
		t.Fatal("database did not store only the expected session-token hash")
	}
	loaded, err := store.getAppSession(ctx, rawToken, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Subject != desired.Subject || loaded.DisplayName != desired.DisplayName || !bytes.Equal(loaded.CSRFSecret, desired.CSRFSecret) {
		t.Fatalf("unexpected session: %#v", loaded)
	}
}

func TestSessionRevocationAndExpiry(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	create := func(token string, expiry time.Time) {
		t.Helper()
		if err := store.createAppSession(ctx, token, appSession{
			Subject: "subject", DisplayName: "Name", CSRFSecret: make([]byte, 32), CreatedAt: now, ExpiresAt: expiry,
		}); err != nil {
			t.Fatal(err)
		}
	}
	create("revoked", now.Add(time.Hour))
	if err := store.revokeAppSession(ctx, "revoked", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.getAppSession(ctx, "revoked", now.Add(2*time.Minute)); !errors.Is(err, errSessionUnavailable) {
		t.Fatalf("revoked lookup error = %v", err)
	}
	create("expired", now.Add(time.Second))
	if _, err := store.getAppSession(ctx, "expired", now.Add(time.Second)); !errors.Is(err, errSessionUnavailable) {
		t.Fatalf("expired lookup error = %v", err)
	}
}

func TestApplicationSessionSurvivesStoreRestart(t *testing.T) {
	ctx := context.Background()
	file := filepath.Join(t.TempDir(), "messages.sqlite")
	store, err := openAppStore(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := store.createAppSession(ctx, "restart-token", appSession{
		Subject: "subject", DisplayName: "Name", CSRFSecret: make([]byte, 32), CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = openAppStore(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.getAppSession(ctx, "restart-token", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
}
