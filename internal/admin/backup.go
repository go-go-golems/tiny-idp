package admin

import (
	"context"
	"fmt"

	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

type BackupResult = sqlitestore.BackupResult
type RestoreResult = sqlitestore.RestoreResult

func CreateSQLiteBackup(ctx context.Context, source, dest string) (BackupResult, error) {
	if source == "" || dest == "" {
		return BackupResult{}, fmt.Errorf("source and destination are required")
	}
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(source))
	if err != nil {
		return BackupResult{}, err
	}
	defer st.Close()
	return st.Backup(ctx, dest)
}

func VerifySQLiteBackup(ctx context.Context, path string) error {
	_, err := sqlitestore.VerifyBackup(ctx, path, nil)
	return err
}

func RestoreSQLiteBackup(ctx context.Context, backupPath, destPath string) (RestoreResult, error) {
	return sqlitestore.Restore(ctx, backupPath, destPath)
}
