package sqlitestore_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestOnlineBackupVerifyAndRestore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "source", "idp.db")
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutClient(ctx, idpstore.Client{ID: "wal-client", RedirectURIs: []string{"https://rp.example/cb"}, AllowedScopes: []string{"openid"}, RequirePKCE: true}); err != nil {
		t.Fatal(err)
	}
	deviceGrant := idpstore.DeviceGrant{ID: "backup-device", DeviceCodeHash: []byte("backup-device-hash"), UserCodeHash: []byte("backup-user-hash"), ClientID: "wal-client", Status: idpstore.DeviceGrantPending, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour), PollInterval: 5 * time.Second, NextPollAt: time.Now().UTC()}
	if err := st.CreateDeviceGrant(ctx, deviceGrant); err != nil {
		t.Fatal(err)
	}
	wal, err := os.Stat(source + "-wal")
	if err != nil || wal.Size() == 0 {
		t.Fatalf("expected committed WAL content before backup: info=%v err=%v", wal, err)
	}
	assertMode(t, source, 0o600)
	assertMode(t, source+"-wal", 0o600)
	assertMode(t, source+"-shm", 0o600)
	backupDir := filepath.Join(root, "backups")
	backupPath := filepath.Join(backupDir, "idp.db")
	result, err := st.Backup(ctx, backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.TableCounts["clients"] != 1 {
		t.Fatalf("backup manifest = %#v", result.Manifest)
	}
	if result.Manifest.TableCounts["device_grants"] != 1 {
		t.Fatalf("device grant count = %#v", result.Manifest)
	}
	assertMode(t, backupDir, 0o700)
	assertMode(t, backupPath, 0o600)
	before, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := sqlitestore.VerifyBackup(ctx, backupPath, &result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) || verified.SchemaVersion != result.Manifest.SchemaVersion {
		t.Fatal("read-only verification mutated the artifact or changed its manifest")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(root, "restore")
	restorePath := filepath.Join(restoreDir, "idp.db")
	restored, err := sqlitestore.Restore(ctx, backupPath, restorePath)
	if err != nil {
		t.Fatal(err)
	}
	if restored.RollbackPath != "" {
		t.Fatalf("unexpected rollback path %q", restored.RollbackPath)
	}
	copyStore, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(restorePath))
	if err != nil {
		t.Fatal(err)
	}
	defer copyStore.Close()
	client, err := copyStore.GetClient(ctx, "wal-client")
	if err != nil || client.ID != "wal-client" {
		t.Fatalf("restored client = %#v, err=%v", client, err)
	}
	if restoredGrant, err := copyStore.InspectDeviceGrantByDeviceCodeHash(ctx, deviceGrant.DeviceCodeHash, deviceGrant.ClientID); err != nil || restoredGrant.ID != deviceGrant.ID {
		t.Fatalf("restored device grant = %#v, err=%v", restoredGrant, err)
	}
}

func TestRestorePreservesRollbackAndRejectsCorruption(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := filepath.Join(root, "source", "idp.db")
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(source))
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(root, "backup", "idp.db")
	if _, err := st.Backup(ctx, backupPath); err != nil {
		t.Fatal(err)
	}
	_ = st.Close()

	restoreDir := filepath.Join(root, "restore")
	if err := os.MkdirAll(restoreDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(restoreDir, "idp.db")
	if err := os.WriteFile(dest, []byte("previous database"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := sqlitestore.Restore(ctx, backupPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if result.RollbackPath == "" {
		t.Fatal("restore did not preserve the previous database")
	}
	rollback, err := os.ReadFile(result.RollbackPath)
	if err != nil || string(rollback) != "previous database" {
		t.Fatalf("rollback contents = %q, err=%v", rollback, err)
	}
	corrupt := filepath.Join(root, "backup", "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("not sqlite"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlitestore.VerifyBackup(ctx, corrupt, nil); err == nil {
		t.Fatal("corrupt backup verified successfully")
	}
}

func TestCanceledBackupLeavesPublishedFileUntouched(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(root, "source", "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	dir := filepath.Join(root, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "idp.db")
	if err := os.WriteFile(dest, []byte("existing artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := st.Backup(canceled, dest); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled backup error = %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "existing artifact" {
		t.Fatalf("published file = %q, err=%v", got, err)
	}
	temps, err := filepath.Glob(filepath.Join(dir, ".tinyidp-backup-*"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("temporary backups = %v, err=%v", temps, err)
	}
}

func TestBusyConnectionHonorsContextDeadline(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	tx, err := st.SQLDB().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	deadline, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	_, err = st.RecordFailedLogin(deadline, "u1", time.Now(), idpstore.LockoutPolicy{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("busy write error = %v", err)
	}
}

func TestBackupDuringConcurrentWritesIsSelfConsistent(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(root, "source", "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- st.PutClient(ctx, idpstore.Client{ID: string(rune('a' + i)), RedirectURIs: []string{"https://rp.example/cb"}, AllowedScopes: []string{"openid"}, RequirePKCE: true})
		}(i)
	}
	result, err := st.Backup(ctx, filepath.Join(root, "backup", "idp.db"))
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sqlitestore.VerifyBackup(ctx, result.Path, &result.Manifest); err != nil {
		t.Fatal(err)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode(%s) = %o, want %o", path, got, want)
	}
}
