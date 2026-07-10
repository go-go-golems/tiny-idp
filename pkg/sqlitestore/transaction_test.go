package sqlitestore_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestUpdateRollsBackAllWrites(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	want := errors.New("injected callback failure")
	err := st.Update(ctx, func(tx idpstore.TxStore) error {
		if err := tx.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "u1"}); err != nil {
			return err
		}
		if err := tx.PutPasswordCredential(ctx, idpstore.PasswordCredential{UserID: "u1", Login: "alice"}); err != nil {
			return err
		}
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Update error = %v, want %v", err, want)
	}
	if _, err := st.GetUser(ctx, "u1"); !errors.Is(err, idpstore.ErrNotFound) {
		t.Fatalf("rolled-back user lookup error = %v", err)
	}
	if _, err := st.GetPasswordCredentialByUserID(ctx, "u1"); !errors.Is(err, idpstore.ErrNotFound) {
		t.Fatalf("rolled-back credential lookup error = %v", err)
	}
}

func TestCreateUserWithCredentialRollsBackOnCredentialConflict(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	if err := st.CreateUserWithCredential(ctx, "alice", idpstore.User{ID: "u1", Sub: "u1"}, idpstore.PasswordCredential{UserID: "u1", Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	err := st.CreateUserWithCredential(ctx, "alice-alias", idpstore.User{ID: "u2", Sub: "u2"}, idpstore.PasswordCredential{UserID: "u2", Login: "alice"})
	if !errors.Is(err, idpstore.ErrDuplicate) {
		t.Fatalf("conflicting create error = %v", err)
	}
	if _, err := st.GetUser(ctx, "u2"); !errors.Is(err, idpstore.ErrNotFound) {
		t.Fatalf("partially committed user lookup error = %v", err)
	}
}

func TestConcurrentFailedLoginsLoseNoUpdates(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	const writers = 24
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := st.RecordFailedLogin(ctx, "u1", time.Now().UTC(), idpstore.LockoutPolicy{Threshold: writers, Window: time.Hour, Duration: time.Minute})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err := st.GetAccountSecurityState(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if state.FailedLoginCount != writers || state.LockedUntil == nil {
		t.Fatalf("security state = %#v, want count=%d and lock", state, writers)
	}
}

func TestFailedMigrationIsNotRecorded(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "migration-failure.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE signing_keys (id TEXT PRIMARY KEY, active INTEGER NOT NULL DEFAULT 0, data BLOB NOT NULL);
INSERT INTO signing_keys(id,active,data) VALUES ('one',1,'{}'),('two',1,'{}');`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	if st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path)); err == nil {
		_ = st.Close()
		t.Fatal("Open succeeded despite duplicate active signing keys")
	}
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=3`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed migration ledger count = %d", count)
	}
}

func TestMigrationChecksumMismatchRefusesOpen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "checksum.db")
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SQLDB().ExecContext(ctx, `UPDATE schema_migrations SET checksum='tampered' WHERE version=1`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path)); err == nil {
		_ = reopened.Close()
		t.Fatal("Open accepted a migration checksum mismatch")
	}
}

func TestNewerSchemaVersionRefusesOpen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "future.db")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	if err != nil {
		t.Fatal(err)
	}
	future := store.SupportedSchemaVersion() + 1
	if _, err := store.SQLDB().ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)`, future, "future.sql", "future", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path)); err == nil {
		t.Fatal("expected newer schema refusal")
	}
}

func TestPasswordReplacementRevokesDomainAndProtocolArtifacts(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	now := time.Now().UTC()
	if err := st.CreateUserWithCredential(ctx, "alice", idpstore.User{ID: "u1", Sub: "subject-1"}, idpstore.PasswordCredential{UserID: "u1", Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(ctx, idpstore.Session{IDHash: []byte("session"), UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	protocolRows := []string{
		`INSERT INTO fosite_authorize_codes(signature,active,subject,request_json) VALUES('code',1,'subject-1','{}')`,
		`INSERT INTO fosite_pkces(signature,subject,request_json) VALUES('pkce','subject-1','{}')`,
		`INSERT INTO fosite_oidc_sessions(signature,subject,request_json) VALUES('oidc','subject-1','{}')`,
		`INSERT INTO fosite_access_tokens(signature,request_id,subject,request_json) VALUES('access','request','subject-1','{}')`,
		`INSERT INTO fosite_refresh_tokens(signature,request_id,active,access_token_signature,subject,request_json) VALUES('refresh','request',1,'access','subject-1','{}')`,
	}
	for _, statement := range protocolRows {
		if _, err := st.SQLDB().ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	credential := idpstore.PasswordCredential{UserID: "u1", Login: "alice", PasswordChangedAt: now}
	state := idpstore.AccountSecurityState{UserID: "u1", LastSuccessfulLoginAt: &now}
	if err := st.ReplacePasswordAndSecurityState(ctx, credential, state); err != nil {
		t.Fatal(err)
	}
	session, err := st.GetSession(ctx, []byte("session"))
	if err != nil || session.RevokedAt == nil {
		t.Fatalf("session after password replacement = %#v, err=%v", session, err)
	}
	checks := []struct {
		query string
		want  int
	}{
		{`SELECT active FROM fosite_authorize_codes WHERE signature='code'`, 0},
		{`SELECT COUNT(*) FROM fosite_pkces WHERE signature='pkce'`, 0},
		{`SELECT COUNT(*) FROM fosite_oidc_sessions WHERE signature='oidc'`, 0},
		{`SELECT COUNT(*) FROM fosite_access_tokens WHERE signature='access'`, 0},
		{`SELECT active FROM fosite_refresh_tokens WHERE signature='refresh'`, 0},
	}
	for _, check := range checks {
		var got int
		if err := st.SQLDB().QueryRowContext(ctx, check.query).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != check.want {
			t.Fatalf("query %q = %d, want %d", check.query, got, check.want)
		}
	}
}

func openTestStore(t *testing.T) *sqlitestore.Store {
	t.Helper()
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
