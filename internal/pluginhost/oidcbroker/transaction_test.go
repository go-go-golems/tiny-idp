package oidcbroker

import (
	"context"
	"crypto/rand"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestTransactionPersistsAcrossRestartAndConsumesOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "idp.db")
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	key := []byte("0123456789abcdef0123456789abcdef")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewTransactionManager(store.SQLDB(), key, rand.Reader, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	created, err := manager.Create(ctx, NewTransaction{
		PluginID: "jitsi", ClientID: "jitsi-client", CallbackPath: "/integrations/jitsi/callback",
		PluginState: []byte(`{"room":"engineering"}`), BrowserBinding: "browser-one", TTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.State == "" || created.Nonce == "" || created.PKCEVerifier == "" || created.PKCEChallenge == "" {
		t.Fatalf("created transaction = %#v", created)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	manager, err = NewTransactionManager(reopened.SQLDB(), key, rand.Reader, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	consumed, err := manager.Consume(ctx, "jitsi", "browser-one", created.State)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.ClientID != "jitsi-client" || consumed.PKCEVerifier != created.PKCEVerifier ||
		string(consumed.PluginState) != `{"room":"engineering"}` {
		t.Fatalf("consumed transaction = %#v", consumed)
	}
	if _, err := manager.Consume(ctx, "jitsi", "browser-one", created.State); !errors.Is(err, ErrTransactionConsumed) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestTransactionFailsClosedForBindingPluginExpiryAndMalformedState(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	manager, err := NewTransactionManager(store.SQLDB(), []byte("0123456789abcdef0123456789abcdef"), rand.Reader, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	create := func() CreatedTransaction {
		value, createErr := manager.Create(ctx, NewTransaction{
			PluginID: "jitsi", ClientID: "jitsi-client", CallbackPath: "/integrations/jitsi/callback",
			BrowserBinding: "browser-one", TTL: time.Minute,
		})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return value
	}
	wrongBinding := create()
	if _, err := manager.Consume(ctx, "jitsi", "browser-two", wrongBinding.State); !errors.Is(err, ErrTransactionBinding) {
		t.Fatalf("binding error = %v", err)
	}
	wrongPlugin := create()
	if _, err := manager.Consume(ctx, "other", "browser-one", wrongPlugin.State); !errors.Is(err, ErrTransactionPlugin) {
		t.Fatalf("plugin error = %v", err)
	}
	expired := create()
	manager.now = func() time.Time { return now.Add(2 * time.Minute) }
	if _, err := manager.Consume(ctx, "jitsi", "browser-one", expired.State); !errors.Is(err, ErrTransactionExpired) {
		t.Fatalf("expiry error = %v", err)
	}
	if _, err := manager.Consume(ctx, "jitsi", "browser-one", "not-base64"); !errors.Is(err, ErrTransactionStateMalformed) {
		t.Fatalf("malformed error = %v", err)
	}
}

func TestTransactionCiphertextDoesNotStoreVerifierOrPluginState(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := NewTransactionManager(store.SQLDB(), []byte("0123456789abcdef0123456789abcdef"), rand.Reader, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	created, err := manager.Create(ctx, NewTransaction{
		PluginID: "jitsi", ClientID: "jitsi-client", CallbackPath: "/integrations/jitsi/callback",
		PluginState: []byte("private-room-state"), BrowserBinding: "browser-one", TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	var verifierBox, stateBox []byte
	if err := store.SQLDB().QueryRowContext(ctx, `SELECT pkce_verifier_box,plugin_state_box FROM integration_transactions`).Scan(&verifierBox, &stateBox); err != nil {
		t.Fatal(err)
	}
	if string(verifierBox) == created.PKCEVerifier || string(stateBox) == "private-room-state" {
		t.Fatal("transaction secret was stored in plaintext")
	}
}
