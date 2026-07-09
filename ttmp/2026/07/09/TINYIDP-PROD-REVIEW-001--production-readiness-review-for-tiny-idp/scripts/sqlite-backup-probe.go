package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/storage"
	"github.com/manuel/tinyidp/internal/store/sqlite"
)

func main() {
	logLevel := flag.String("log-level", "info", "zerolog level")
	flag.Parse()
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
		os.Exit(2)
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	if err := run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("backup probe failed")
	}
}

func run(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "tinyidp-backup-probe-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Warn().Err(err).Str("path", dir).Msg("remove backup probe directory")
		}
	}()

	source := filepath.Join(dir, "source.db")
	backup := filepath.Join(dir, "backup.db")
	store, err := sqlite.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer store.Close()
	if _, err := store.SQLDB().ExecContext(ctx, `PRAGMA journal_mode=WAL`); err != nil {
		return fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := store.SQLDB().ExecContext(ctx, `PRAGMA wal_autocheckpoint=0`); err != nil {
		return fmt.Errorf("disable auto-checkpoint: %w", err)
	}

	client := domain.Client{
		ID:            "committed-after-checkpoint",
		Public:        true,
		RequirePKCE:   true,
		RedirectURIs:  []string{"https://client.example.test/callback"},
		AllowedScopes: []string{"openid"},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.PutClient(ctx, client); err != nil {
		return fmt.Errorf("commit client: %w", err)
	}
	walInfo, err := os.Stat(source + "-wal")
	if err != nil {
		return fmt.Errorf("stat WAL: %w", err)
	}
	log.Info().Int64("wal_bytes", walInfo.Size()).Msg("committed client is represented in live WAL")

	result, err := admin.CreateSQLiteBackup(ctx, source, backup)
	if err != nil {
		return fmt.Errorf("product backup: %w", err)
	}
	log.Info().Int64("backup_bytes", result.Bytes).Msg("product backup copied only the main database file")

	copyStore, err := sqlite.Open(backup)
	if err != nil {
		return fmt.Errorf("backup did not open: %w", err)
	}
	defer copyStore.Close()
	_, lookupErr := copyStore.GetClient(ctx, client.ID)
	switch lookupErr {
	case nil:
		return fmt.Errorf("probe assumption failed: copied file unexpectedly contains WAL-only client")
	case storage.ErrNotFound:
		fmt.Println("CONFIRMED: backup opens successfully but omits a committed client stored in the source WAL")
		return nil
	default:
		return fmt.Errorf("lookup copied client: %w", lookupErr)
	}
}
