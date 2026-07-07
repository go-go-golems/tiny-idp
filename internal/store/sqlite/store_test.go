package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/storage"
	"github.com/manuel/tinyidp/internal/store/sqlite"
)

func TestStoreSuite(t *testing.T) {
	storage.RunStoreSuite(t, func(t *testing.T) storage.Store {
		st, err := sqlite.Open(filepath.Join(t.TempDir(), "idp.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		return st
	})
}

func TestSigningKeyRotationPersistsRetiredVerificationKey(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	old, err := keys.GenerateRSA("old", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, _, err := keys.RotateRSA(ctx, st, "new", time.Now()); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st2, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	active, err := st2.ActiveSigningKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "new" {
		t.Fatalf("active = %s", active.ID)
	}
	keysForVerify, err := st2.VerificationKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, k := range keysForVerify {
		seen[k.ID] = true
	}
	if !seen["old"] || !seen["new"] {
		t.Fatalf("verification keys = %#v", seen)
	}
}

func TestSigningKeyPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-restart", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	st2, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	active, err := st2.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if active.ID != "kid-restart" {
		t.Fatalf("active key = %s", active.ID)
	}
}
