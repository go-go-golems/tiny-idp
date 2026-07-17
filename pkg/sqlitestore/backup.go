package sqlitestore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
)

// Manifest is the verified logical identity of a database snapshot.
type Manifest struct {
	SchemaVersion      int              `json:"schema_version"`
	MigrationChecksums map[int]string   `json:"migration_checksums"`
	TableCounts        map[string]int64 `json:"table_counts"`
	ActiveSigningKeys  []string         `json:"active_signing_keys"`
}

// BackupResult describes an atomically published online backup.
type BackupResult struct {
	Source   string   `json:"source"`
	Path     string   `json:"path"`
	Bytes    int64    `json:"bytes"`
	Manifest Manifest `json:"manifest"`
}

// RestoreResult describes an installed database and its preserved rollback
// copy, when a previous destination existed.
type RestoreResult struct {
	Path         string   `json:"path"`
	RollbackPath string   `json:"rollback_path,omitempty"`
	Manifest     Manifest `json:"manifest"`
}

var manifestTables = []string{
	"clients", "users", "password_credentials", "account_security_states",
	"grants", "authorization_codes", "access_tokens", "refresh_tokens",
	"consents", "sessions", "signing_keys", "fosite_authorize_codes",
	"browser_contexts", "remembered_browser_sessions",
	"device_grants",
	"fosite_pkces", "fosite_oidc_sessions", "fosite_access_tokens",
	"fosite_refresh_tokens", "fosite_jtis",
}

// Backup creates a consistent SQLite online backup, verifies it against a
// source manifest, fsyncs it, and atomically publishes it at dest.
func (s *Store) Backup(ctx context.Context, dest string) (BackupResult, error) {
	if strings.TrimSpace(dest) == "" {
		return BackupResult{}, fmt.Errorf("backup destination is required")
	}
	if samePath(s.path, dest) {
		return BackupResult{}, fmt.Errorf("backup destination must differ from source database")
	}
	dir := filepath.Dir(dest)
	if err := ensureOwnerOnlyDirectory(dir); err != nil {
		return BackupResult{}, err
	}
	temp, err := os.CreateTemp(dir, ".tinyidp-backup-*.db")
	if err != nil {
		return BackupResult{}, fmt.Errorf("create temporary backup: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return BackupResult{}, fmt.Errorf("set temporary backup permissions: %w", err)
	}
	if err := temp.Close(); err != nil {
		return BackupResult{}, fmt.Errorf("close temporary backup: %w", err)
	}

	sourceConn, err := s.db.Conn(ctx)
	if err != nil {
		return BackupResult{}, fmt.Errorf("reserve source connection: %w", err)
	}
	defer sourceConn.Close()
	manifest, err := readManifest(ctx, sourceConn)
	if err != nil {
		return BackupResult{}, fmt.Errorf("read source manifest: %w", err)
	}
	copyBackup := onlineBackup
	if s.backupCopy != nil {
		copyBackup = s.backupCopy
	}
	if err := copyBackup(ctx, sourceConn, tempPath); err != nil {
		return BackupResult{}, err
	}
	verified, err := VerifyBackup(ctx, tempPath, &manifest)
	if err != nil {
		return BackupResult{}, fmt.Errorf("verify temporary backup: %w", err)
	}
	if err := syncFile(tempPath); err != nil {
		return BackupResult{}, err
	}
	if err := syncDirectory(dir); err != nil {
		return BackupResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return BackupResult{}, err
	}
	if err := os.Rename(tempPath, dest); err != nil {
		return BackupResult{}, fmt.Errorf("publish backup: %w", err)
	}
	if err := os.Chmod(dest, 0o600); err != nil {
		return BackupResult{}, fmt.Errorf("set published backup permissions: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return BackupResult{}, err
	}
	info, err := os.Stat(dest)
	if err != nil {
		return BackupResult{}, err
	}
	return BackupResult{Source: s.path, Path: dest, Bytes: info.Size(), Manifest: verified}, nil
}

func onlineBackup(ctx context.Context, sourceConn *sql.Conn, dest string) error {
	destDB, err := sql.Open("sqlite3", dest)
	if err != nil {
		return fmt.Errorf("open temporary backup database: %w", err)
	}
	destDB.SetMaxOpenConns(1)
	defer destDB.Close()
	destConn, err := destDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve backup connection: %w", err)
	}
	defer destConn.Close()
	var backup *sqlite3.SQLiteBackup
	err = destConn.Raw(func(destDriver any) error {
		destination, ok := destDriver.(*sqlite3.SQLiteConn)
		if !ok {
			return fmt.Errorf("unexpected destination SQLite driver %T", destDriver)
		}
		return sourceConn.Raw(func(sourceDriver any) error {
			source, ok := sourceDriver.(*sqlite3.SQLiteConn)
			if !ok {
				return fmt.Errorf("unexpected source SQLite driver %T", sourceDriver)
			}
			var backupErr error
			backup, backupErr = destination.Backup("main", source, "main")
			return backupErr
		})
	})
	if err != nil {
		return fmt.Errorf("start SQLite online backup: %w", err)
	}
	finished := false
	defer func() {
		if !finished {
			_ = backup.Close()
		}
	}()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done, err := backup.Step(128)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "busy") || strings.Contains(strings.ToLower(err.Error()), "locked") {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}
			return fmt.Errorf("copy SQLite backup pages: %w", err)
		}
		if done {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
	if err := backup.Finish(); err != nil {
		return fmt.Errorf("finish SQLite online backup: %w", err)
	}
	finished = true
	return nil
}

// VerifyBackup opens an artifact read-only and checks integrity, schema,
// migration checksums, and, when supplied, the expected source manifest.
func VerifyBackup(ctx context.Context, path string, expected *Manifest) (Manifest, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Manifest{}, err
	}
	u := &url.URL{Scheme: "file", Path: abs, RawQuery: "mode=ro&immutable=1"}
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return Manifest{}, fmt.Errorf("open backup read-only: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return Manifest{}, fmt.Errorf("run integrity_check: %w", err)
	}
	if integrity != "ok" {
		return Manifest{}, fmt.Errorf("SQLite integrity_check failed: %s", integrity)
	}
	manifest, err := readManifest(ctx, db)
	if err != nil {
		return Manifest{}, err
	}
	names, err := MigrationNames()
	if err != nil {
		return Manifest{}, err
	}
	if manifest.SchemaVersion != len(names) {
		return Manifest{}, fmt.Errorf("unsupported backup schema version %d; expected %d", manifest.SchemaVersion, len(names))
	}
	embedded, err := embeddedMigrationChecksums()
	if err != nil {
		return Manifest{}, err
	}
	if !reflect.DeepEqual(manifest.MigrationChecksums, embedded) {
		return Manifest{}, fmt.Errorf("backup migration checksums do not match this binary")
	}
	if expected != nil && !reflect.DeepEqual(manifest, *expected) {
		return Manifest{}, fmt.Errorf("backup manifest differs from source snapshot")
	}
	return manifest, nil
}

// Restore verifies and atomically installs a backup while the provider is
// stopped. It refuses destinations with live WAL/SHM sidecars and preserves an
// owner-only rollback copy of an existing database.
func Restore(ctx context.Context, backupPath, destPath string) (RestoreResult, error) {
	if strings.TrimSpace(backupPath) == "" || strings.TrimSpace(destPath) == "" {
		return RestoreResult{}, fmt.Errorf("backup and destination paths are required")
	}
	if samePath(backupPath, destPath) {
		return RestoreResult{}, fmt.Errorf("restore destination must differ from backup")
	}
	manifest, err := VerifyBackup(ctx, backupPath, nil)
	if err != nil {
		return RestoreResult{}, fmt.Errorf("refuse unverified backup: %w", err)
	}
	for _, sidecar := range []string{destPath + "-wal", destPath + "-shm"} {
		if _, err := os.Stat(sidecar); err == nil {
			return RestoreResult{}, fmt.Errorf("refuse restore while SQLite sidecar exists: %s", sidecar)
		} else if !errors.Is(err, os.ErrNotExist) {
			return RestoreResult{}, err
		}
	}
	dir := filepath.Dir(destPath)
	if err := ensureOwnerOnlyDirectory(dir); err != nil {
		return RestoreResult{}, err
	}
	temp, err := os.CreateTemp(dir, ".tinyidp-restore-*.db")
	if err != nil {
		return RestoreResult{}, err
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return RestoreResult{}, err
	}
	in, err := os.Open(backupPath)
	if err != nil {
		_ = temp.Close()
		return RestoreResult{}, err
	}
	_, copyErr := copyContext(ctx, temp, in)
	closeInErr := in.Close()
	syncErr := temp.Sync()
	closeErr := temp.Close()
	for _, candidate := range []error{copyErr, closeInErr, syncErr, closeErr} {
		if candidate != nil {
			return RestoreResult{}, fmt.Errorf("stage restore: %w", candidate)
		}
	}
	if _, err := VerifyBackup(ctx, tempPath, &manifest); err != nil {
		return RestoreResult{}, fmt.Errorf("verify staged restore: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return RestoreResult{}, err
	}
	rollbackPath := ""
	if _, err := os.Stat(destPath); err == nil {
		rollbackPath = destPath + ".pre-restore-" + time.Now().UTC().Format("20060102T150405.000000000Z")
		if err := copyOwnerOnlyFile(ctx, destPath, rollbackPath); err != nil {
			return RestoreResult{}, fmt.Errorf("preserve rollback database: %w", err)
		}
		if err := syncDirectory(dir); err != nil {
			return RestoreResult{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return RestoreResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return RestoreResult{}, err
	}
	if err := os.Rename(tempPath, destPath); err != nil {
		return RestoreResult{}, fmt.Errorf("install restored database: %w", err)
	}
	if err := os.Chmod(destPath, 0o600); err != nil {
		return RestoreResult{}, err
	}
	if err := syncDirectory(dir); err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{Path: destPath, RollbackPath: rollbackPath, Manifest: manifest}, nil
}

func readManifest(ctx context.Context, runner sqlRunner) (Manifest, error) {
	manifest := Manifest{MigrationChecksums: map[int]string{}, TableCounts: map[string]int64{}}
	rows, err := runner.QueryContext(ctx, `SELECT version,checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return Manifest{}, fmt.Errorf("read migration ledger: %w", err)
	}
	for rows.Next() {
		var version int
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			_ = rows.Close()
			return Manifest{}, err
		}
		manifest.MigrationChecksums[version] = checksum
		if version > manifest.SchemaVersion {
			manifest.SchemaVersion = version
		}
	}
	if err := rows.Close(); err != nil {
		return Manifest{}, err
	}
	for _, table := range manifestTables {
		var count int64
		if err := runner.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			return Manifest{}, fmt.Errorf("count %s: %w", table, err)
		}
		manifest.TableCounts[table] = count
	}
	rows, err = runner.QueryContext(ctx, `SELECT id FROM signing_keys WHERE active=1 ORDER BY id`)
	if err != nil {
		return Manifest{}, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return Manifest{}, err
		}
		manifest.ActiveSigningKeys = append(manifest.ActiveSigningKeys, id)
	}
	if err := rows.Close(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func embeddedMigrationChecksums() (map[int]string, error) {
	names, err := MigrationNames()
	if err != nil {
		return nil, err
	}
	out := make(map[int]string, len(names))
	for _, name := range names {
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", name, err)
		}
		body, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		sum := sha256Sum(body)
		out[version] = hex.EncodeToString(sum)
	}
	return out, nil
}

func sha256Sum(body []byte) []byte {
	sum := sha256.Sum256(body)
	return sum[:]
}

func ensureOwnerOnlyDirectory(path string) error {
	clean := filepath.Clean(path)
	if clean == string(filepath.Separator) || clean == filepath.Clean(os.TempDir()) {
		return fmt.Errorf("backup and restore require a dedicated owner-only directory, not %s", clean)
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create owner-only directory: %w", err)
	}
	// #nosec G302 -- directories need owner search permission; 0700 is owner-only.
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("set owner-only directory permissions: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("directory %s must be owner-only (0700)", path)
	}
	return nil
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbs == rightAbs
}

func syncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("fsync %s: %w", path, err)
	}
	return nil
}

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("fsync directory %s: %w", path, err)
	}
	return nil
}

func copyOwnerOnlyFile(ctx context.Context, source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := copyContext(ctx, out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	for _, candidate := range []error{copyErr, syncErr, closeErr} {
		if candidate != nil {
			_ = os.Remove(dest)
			return candidate
		}
	}
	return nil
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 128*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		read, readErr := source.Read(buffer)
		if read > 0 {
			n, writeErr := destination.Write(buffer[:read])
			written += int64(n)
			if writeErr != nil {
				return written, writeErr
			}
			if n != read {
				return written, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return written, nil
		}
		if readErr != nil {
			return written, readErr
		}
	}
}
