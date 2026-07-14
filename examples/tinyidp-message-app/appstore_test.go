package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationMigrationCreatesExpectedSchema(t *testing.T) {
	file := filepath.Join(t.TempDir(), "application", "messages.sqlite")
	store, err := openAppStore(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	version, err := store.schemaVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if version != len(appMigrations) {
		t.Fatalf("schema version = %d, want %d", version, len(appMigrations))
	}
	for _, table := range []string{"app_schema_migrations", "oidc_login_attempts", "app_sessions", "registration_attempts", "messages"} {
		var count int
		if err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("table %s count = %d", table, count)
		}
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions = %#o, want 0600", info.Mode().Perm())
	}
}

func TestMigrationChecksumMismatchFailsStartup(t *testing.T) {
	ctx := context.Background()
	file := filepath.Join(t.TempDir(), "messages.sqlite")
	store, err := openAppStore(ctx, file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE app_schema_migrations SET checksum = 'tampered' WHERE version = 1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := openAppStore(ctx, file); err == nil {
		t.Fatal("expected checksum mismatch to fail startup")
	}
}

func TestApplicationStoreEnablesRequiredPragmas(t *testing.T) {
	store, err := openAppStore(context.Background(), filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var journalMode, synchronous string
	var foreignKeys, busyTimeout int
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" || synchronous != "2" || foreignKeys != 1 || busyTimeout != 5000 {
		t.Fatalf("pragmas journal=%q synchronous=%q foreign_keys=%d busy_timeout=%d", journalMode, synchronous, foreignKeys, busyTimeout)
	}
}
