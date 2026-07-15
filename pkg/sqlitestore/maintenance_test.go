package sqlitestore_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestMaintenanceHonorsRetentionAndVerificationOverlap(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	oldSession := idpstore.Session{IDHash: []byte("old"), UserID: "u", ExpiresAt: now.Add(-49 * time.Hour)}
	freshSession := idpstore.Session{IDHash: []byte("fresh"), UserID: "u", ExpiresAt: now.Add(-23 * time.Hour)}
	if err := store.CreateSession(ctx, oldSession); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, freshSession); err != nil {
		t.Fatal(err)
	}

	active, err := keys.GenerateRSA("active", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, active); err != nil {
		t.Fatal(err)
	}
	oldKey, err := keys.GenerateRSA("old", now.Add(-72*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	oldKey.Active = false
	oldKey.NotAfter = now.Add(-3 * time.Hour)
	if err := store.CreateSigningKey(ctx, oldKey); err != nil {
		t.Fatal(err)
	}
	recentKey, err := keys.GenerateRSA("recent", now.Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	recentKey.Active = false
	recentKey.NotAfter = now.Add(-30 * time.Minute)
	if err := store.CreateSigningKey(ctx, recentKey); err != nil {
		t.Fatal(err)
	}

	oldCreated := now.Add(-32 * 24 * time.Hour)
	if _, err := store.SQLDB().ExecContext(ctx, `INSERT INTO fosite_pkces(signature,subject,request_json,created_at) VALUES(?,?,?,?)`, "old-protocol", "u", []byte(`{}`), oldCreated); err != nil {
		t.Fatal(err)
	}
	report, err := store.Maintain(ctx, now, idpstore.MaintenancePolicy{RetainExpiredFor: 24 * time.Hour, ProtocolStateRetention: 31 * 24 * time.Hour, SigningKeyRetention: 2 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	if report.DomainRecords != 1 || report.ProtocolRecords != 1 || report.RetiredSigningKeys != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if _, err := store.GetSession(ctx, oldSession.IDHash); !errors.Is(err, idpstore.ErrNotFound) {
		t.Fatalf("old session error = %v", err)
	}
	if _, err := store.GetSession(ctx, freshSession.IDHash); err != nil {
		t.Fatalf("fresh session: %v", err)
	}
	verification, err := store.VerificationKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, key := range verification {
		seen[key.ID] = true
	}
	if seen["old"] || !seen["recent"] || !seen["active"] {
		t.Fatalf("verification keys = %#v", seen)
	}
}

func TestMaintenanceRejectsUnsafePolicyWithoutMutation(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.Maintain(ctx, time.Now(), idpstore.MaintenancePolicy{}); err == nil {
		t.Fatal("expected invalid maintenance policy")
	}
}

func TestMaintenanceDeletesExpiredDeviceGrantsAfterRetention(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Date(2026, 7, 15, 16, 0, 0, 0, time.UTC)
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-client"}); err != nil {
		t.Fatal(err)
	}
	grant := idpstore.DeviceGrant{ID: "expired-device", DeviceCodeHash: []byte("expired-device-hash"), UserCodeHash: []byte("expired-user-hash"), ClientID: "device-client", Status: idpstore.DeviceGrantPending, CreatedAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(-48 * time.Hour), PollInterval: time.Second, NextPollAt: now.Add(-72 * time.Hour)}
	if err := store.CreateDeviceGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	report, err := store.Maintain(ctx, now, idpstore.MaintenancePolicy{RetainExpiredFor: 24 * time.Hour, ProtocolStateRetention: time.Hour, SigningKeyRetention: time.Hour})
	if err != nil || report.DomainRecords != 1 {
		t.Fatalf("maintenance = %#v, %v", report, err)
	}
	if _, err := store.InspectDeviceGrantByDeviceCodeHash(ctx, grant.DeviceCodeHash, grant.ClientID); !errors.Is(err, idpstore.ErrNotFound) {
		t.Fatalf("expired device grant survived maintenance: %v", err)
	}
}
