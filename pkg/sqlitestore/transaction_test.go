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

func openTestStore(t *testing.T) *sqlitestore.Store {
	t.Helper()
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
