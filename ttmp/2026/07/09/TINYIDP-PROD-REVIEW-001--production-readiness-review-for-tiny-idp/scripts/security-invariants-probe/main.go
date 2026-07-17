package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/go-go-golems/tiny-idp/internal/admin"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func main() {
	rounds := flag.Int("rounds", 25, "concurrent lockout rounds")
	attempts := flag.Int("attempts", 5, "simultaneous failed logins per round")
	logLevel := flag.String("log-level", "info", "zerolog level")
	flag.Parse()
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --log-level: %v\n", err)
		os.Exit(2)
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	if *rounds < 1 || *attempts < 1 {
		log.Fatal().Msg("--rounds and --attempts must be positive")
	}
	if err := runSecurityProbe(context.Background(), *rounds, *attempts); err != nil {
		log.Fatal().Err(err).Msg("security invariant probe failed")
	}
}

func runSecurityProbe(ctx context.Context, rounds, attempts int) error {
	dir, err := os.MkdirTemp("", "tinyidp-security-probe-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Warn().Err(err).Str("path", dir).Msg("remove security probe directory")
		}
	}()
	dbPath := filepath.Join(dir, "idp.db")

	previousUmask := syscall.Umask(0)
	store, openErr := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	syscall.Umask(previousUmask)
	if openErr != nil {
		return fmt.Errorf("open SQLite: %w", openErr)
	}
	defer store.Close()
	info, err := os.Stat(dbPath)
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if mode != 0o600 {
		return fmt.Errorf("SQLite database mode under umask 000 = %04o, want 0600", mode)
	}
	fmt.Println("PASS: SQLite database remains 0600 under umask 000")

	service, err := admin.NewService(store, admin.Options{})
	if err != nil {
		return err
	}
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		return err
	}
	_, err = accounts.Create(ctx, idpaccounts.CreateRequest{Login: "short-password-user", Password: []byte("x"), Email: "short@example.test"})
	if !errors.Is(err, idp.ErrPasswordRejected) {
		return fmt.Errorf("one-character password error = %v, want password rejection", err)
	}
	fmt.Println("PASS: one-character password is rejected by the establishment policy")
	_, err = accounts.Create(ctx, idpaccounts.CreateRequest{Login: "probe-user", Password: []byte("a valid password phrase"), Email: "probe@example.test"})
	if err != nil {
		return fmt.Errorf("create valid probe user: %w", err)
	}

	_, _, err = service.CreateClient(ctx, admin.CreateClientRequest{
		ID:                "probe-spa",
		Public:            true,
		RequirePKCE:       true,
		RedirectURIs:      []string{"https://client.example.test/callback"},
		AllowedScopes:     []string{"openid"},
		AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken},
	})
	if err != nil {
		return err
	}
	expiredKey, err := keys.GenerateRSA("expired-active-key", time.Now().Add(-48*time.Hour))
	if err != nil {
		return err
	}
	expiredKey.NotAfter = time.Now().Add(-24 * time.Hour)
	if err := store.CreateSigningKey(ctx, expiredKey); err != nil {
		return err
	}
	provider, err := embeddedidp.New(context.Background(), embeddedidp.Options{
		Issuer: "https://id.example.test",
		Mode:   embeddedidp.ProductionMode,
		Store:  store,
		Cookie: embeddedidp.CookieConfig{Secure: true},
		Token:  embeddedidp.TokenConfig{SecretKey: []byte("security-probe-secret-key-32-bytes-minimum")},
	})
	if err == nil {
		_ = provider.Close(ctx)
		return fmt.Errorf("production provider accepted expired key and nil controls")
	}
	fmt.Println("PASS: production construction rejects expired keys and missing controls")

	lostRound, observedCount, locked, err := probeConcurrentLockout(ctx, store, accounts, rounds, attempts)
	if err != nil {
		return err
	}
	if lostRound >= 0 {
		return fmt.Errorf("concurrent failed-login invariant failed in round %d: attempts=%d stored_count=%d locked=%t", lostRound, attempts, observedCount, locked)
	}
	fmt.Printf("PASS: %d rounds of %d concurrent failures reached the lockout threshold without lost updates\n", rounds, attempts)
	return nil
}

func probeConcurrentLockout(ctx context.Context, store *sqlitestore.Store, passwords *idpaccounts.Service, rounds, attempts int) (int, int, bool, error) {
	user, err := store.GetUserByLogin(ctx, "probe-user")
	if err != nil {
		return 0, 0, false, err
	}
	for round := 0; round < rounds; round++ {
		if err := store.ResetAccountSecurityState(ctx, user.ID, time.Now().UTC()); err != nil {
			return 0, 0, false, err
		}
		start := make(chan struct{})
		group, groupCtx := errgroup.WithContext(ctx)
		for attempt := 0; attempt < attempts; attempt++ {
			group.Go(func() error {
				select {
				case <-start:
				case <-groupCtx.Done():
					return groupCtx.Err()
				}
				_, authErr := passwords.AuthenticatePassword(groupCtx, "probe-user", "wrong", idp.LoginMetadata{ClientID: "probe"})
				if authErr == nil || (!errors.Is(authErr, idpaccounts.ErrInvalidCredentials) && !errors.Is(authErr, idpaccounts.ErrAccountLocked)) {
					return fmt.Errorf("unexpected authentication result: %v", authErr)
				}
				return nil
			})
		}
		close(start)
		if err := group.Wait(); err != nil {
			return 0, 0, false, err
		}
		state, err := store.GetAccountSecurityState(ctx, user.ID)
		if err != nil {
			return 0, 0, false, err
		}
		locked := state.LockedUntil != nil && time.Now().UTC().Before(*state.LockedUntil)
		wantCount := attempts
		if wantCount > 5 {
			wantCount = 5
		}
		if state.FailedLoginCount < wantCount || !locked {
			return round, state.FailedLoginCount, locked, nil
		}
	}
	return -1, attempts, true, nil
}
