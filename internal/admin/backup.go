package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/manuel/tinyidp/pkg/sqlitestore"
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
	same, err := sameFile(source, dest)
	if err != nil {
		return BackupResult{}, err
	}
	if same {
		return BackupResult{}, fmt.Errorf("backup destination must differ from source database")
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

func sameFile(source, dest string) (bool, error) {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return false, err
	}
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return false, err
	}
	if sourceAbs == destAbs {
		return true, nil
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return false, err
	}
	destInfo, err := os.Stat(dest)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return os.SameFile(sourceInfo, destInfo), nil
}

func VerifySQLiteBackup(ctx context.Context, path string) error {
	st, err := sqlitestore.Open(path)
	if err != nil {
		return err
	}
	defer st.Close()
	_, err = st.ListClients(ctx)
	return err
}
