package admin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/manuel/tinyidp/internal/store/sqlite"
)

type BackupResult struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
}

func CreateSQLiteBackup(_ context.Context, source, dest string) (BackupResult, error) {
	if source == "" || dest == "" {
		return BackupResult{}, fmt.Errorf("source and destination are required")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return BackupResult{}, err
	}
	in, err := os.Open(source)
	if err != nil {
		return BackupResult{}, err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return BackupResult{}, err
	}
	bytes, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return BackupResult{}, copyErr
	}
	if closeErr != nil {
		return BackupResult{}, closeErr
	}
	return BackupResult{Source: source, Path: dest, Bytes: bytes}, nil
}

func VerifySQLiteBackup(ctx context.Context, path string) error {
	st, err := sqlite.Open(path)
	if err != nil {
		return err
	}
	defer st.Close()
	_, err = st.ListClients(ctx)
	return err
}
