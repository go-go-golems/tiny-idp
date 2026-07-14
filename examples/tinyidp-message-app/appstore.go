package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type appStore struct {
	db *sql.DB
}

var _ io.Closer = (*appStore)(nil)

type appMigration struct {
	Version  int
	Filename string
	SQL      string
}

var appMigrations = []appMigration{{
	Version: 1, Filename: "001_initial_schema.sql", SQL: `
CREATE TABLE oidc_login_attempts (
    state_hash BLOB PRIMARY KEY CHECK(length(state_hash) = 32),
    nonce TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    return_to TEXT NOT NULL,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT
);
CREATE INDEX oidc_login_attempts_expiry ON oidc_login_attempts(expires_at);

CREATE TABLE app_sessions (
    token_hash BLOB PRIMARY KEY CHECK(length(token_hash) = 32),
    subject TEXT NOT NULL,
    display_name TEXT NOT NULL,
    csrf_secret BLOB NOT NULL CHECK(length(csrf_secret) = 32),
    created_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked_at TEXT
);
CREATE INDEX app_sessions_subject ON app_sessions(subject);
CREATE INDEX app_sessions_expiry ON app_sessions(expires_at);

CREATE TABLE registration_attempts (
    token_hash BLOB PRIMARY KEY CHECK(length(token_hash) = 32),
    csrf_secret BLOB NOT NULL CHECK(length(csrf_secret) = 32),
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    consumed_at TEXT
);
CREATE INDEX registration_attempts_expiry ON registration_attempts(expires_at);

CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_subject TEXT NOT NULL CHECK(length(author_subject) BETWEEN 1 AND 512),
    author_name TEXT NOT NULL CHECK(length(author_name) BETWEEN 1 AND 256),
    body TEXT NOT NULL CHECK(length(body) BETWEEN 1 AND 4096),
    created_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE INDEX messages_feed ON messages(created_at DESC, id DESC) WHERE deleted_at IS NULL;
`,
}}

func openAppStore(ctx context.Context, file string) (*appStore, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(filepath.Clean(file))
	if err != nil {
		return nil, errors.Wrap(err, "resolve application database path")
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, errors.Wrap(err, "create application database directory")
	}
	if err := os.Chmod(filepath.Dir(absolute), 0o700); err != nil {
		return nil, errors.Wrap(err, "protect application database directory")
	}
	query := url.Values{
		"_busy_timeout": {"5000"}, "_foreign_keys": {"on"}, "_journal_mode": {"WAL"},
		"_synchronous": {"FULL"}, "_txlock": {"immediate"},
	}
	dsn := (&url.URL{Scheme: "file", Path: absolute, RawQuery: query.Encode()}).String()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "open application database")
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)
	store := &appStore{db: db}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "ping application database")
	}
	if err := os.Chmod(absolute, 0o600); err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "protect application database")
	}
	if err := store.applyMigrations(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *appStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *appStore) applyMigrations(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS app_schema_migrations (
    version INTEGER PRIMARY KEY,
    filename TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at TEXT NOT NULL
)`); err != nil {
		return errors.Wrap(err, "create application migration ledger")
	}
	for _, migration := range appMigrations {
		if err := s.applyMigration(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

func (s *appStore) applyMigration(ctx context.Context, migration appMigration) (retErr error) {
	checksum := appMigrationChecksum(migration)
	var existingFilename, existingChecksum string
	err := s.db.QueryRowContext(ctx,
		"SELECT filename, checksum FROM app_schema_migrations WHERE version = ?", migration.Version,
	).Scan(&existingFilename, &existingChecksum)
	switch {
	case err == nil:
		if existingFilename != migration.Filename || existingChecksum != checksum {
			return errors.Errorf("application migration %d checksum or filename mismatch", migration.Version)
		}
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return errors.Wrapf(err, "read application migration %d", migration.Version)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "begin application migration")
	}
	defer func() {
		if retErr != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return errors.Wrapf(err, "apply application migration %s", migration.Filename)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO app_schema_migrations(version, filename, checksum, applied_at) VALUES(?, ?, ?, ?)",
		migration.Version, migration.Filename, checksum, formatAppTime(time.Now().UTC()),
	); err != nil {
		return errors.Wrap(err, "record application migration")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "commit application migration")
	}
	return nil
}

func appMigrationChecksum(migration appMigration) string {
	digest := sha256.Sum256([]byte(strconv.Itoa(migration.Version) + "\x00" + migration.Filename + "\x00" + migration.SQL))
	return hex.EncodeToString(digest[:])
}

func (s *appStore) schemaVersion(ctx context.Context) (int, error) {
	var version sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(version) FROM app_schema_migrations").Scan(&version); err != nil {
		return 0, errors.Wrap(err, "read application schema version")
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func formatAppTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseAppTime(raw string) (time.Time, error) {
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "parse application timestamp %q", raw)
	}
	return value.UTC(), nil
}

func requireHash32(value []byte, label string) error {
	if len(value) != sha256.Size {
		return fmt.Errorf("%s must be exactly %d bytes", label, sha256.Size)
	}
	return nil
}
