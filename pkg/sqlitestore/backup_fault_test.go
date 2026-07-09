package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestBackupDiskFullLeavesPublishedFileUntouched(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	st, err := Open(ctx, DefaultConfig(filepath.Join(root, "source", "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	dir := filepath.Join(root, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "idp.db")
	if err := os.WriteFile(dest, []byte("last good backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	st.backupCopy = func(context.Context, *sql.Conn, string) error { return syscall.ENOSPC }
	if _, err := st.Backup(ctx, dest); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("disk-full backup error = %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "last good backup" {
		t.Fatalf("published file = %q, err=%v", got, err)
	}
}
