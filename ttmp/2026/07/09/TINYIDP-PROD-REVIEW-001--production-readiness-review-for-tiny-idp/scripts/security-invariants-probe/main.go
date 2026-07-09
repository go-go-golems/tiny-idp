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

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/sqlite"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
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
	store, openErr := sqlite.Open(dbPath)
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
	fmt.Printf("OBSERVED: SQLite database mode under umask 000 is %04o\n", mode)
	if mode&0o077 != 0 {
		fmt.Println("CONFIRMED: SQLite store creation does not enforce owner-only permissions")
	}

	hasher := passwordhash.New(passwordhash.TestParams())
	service, err := admin.NewService(store, admin.Options{Hasher: hasher})
	if err != nil {
		return err
	}
	_, err = service.CreateUser(ctx, admin.CreateUserRequest{
		Login:             "short-password-user",
		Password:          []byte("x"),
		Email:             "short@example.test",
		MustChangeAtLogin: true,
	})
	if err != nil {
		return fmt.Errorf("one-character password was unexpectedly rejected: %w", err)
	}
	authResult, err := service.Passwords.AuthenticatePassword(ctx, "short-password-user", "x", idp.LoginMetadata{ClientID: "probe"})
	if err != nil {
		return fmt.Errorf("authenticate accepted one-character password: %w", err)
	}
	fmt.Printf("CONFIRMED: one-character password is accepted; MustChangePassword=%t\n", authResult.MustChangePassword)

	_, _, err = service.CreateClient(ctx, admin.CreateClientRequest{
		ID:            "probe-spa",
		Public:        true,
		RequirePKCE:   true,
		RedirectURIs:  []string{"https://client.example.test/callback"},
		AllowedScopes: []string{"openid"},
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
	provider, err := embeddedidp.New(embeddedidp.Options{
		Issuer: "https://id.example.test",
		Mode:   embeddedidp.ProductionMode,
		Store:  store,
		Cookie: embeddedidp.CookieConfig{Secure: true},
		Token:  embeddedidp.TokenConfig{SecretKey: []byte("security-probe-secret-key-32-bytes-minimum")},
	})
	if err != nil {
		return fmt.Errorf("production provider rejected expired key or nil controls: %w", err)
	}
	if provider.Handler() == nil {
		return fmt.Errorf("production provider returned nil handler")
	}
	fmt.Println("CONFIRMED: production construction accepts an expired active key plus nil audit and rate-limit controls")

	lostRound, observedCount, locked, err := probeConcurrentLockout(ctx, store, service.Passwords, rounds, attempts)
	if err != nil {
		return err
	}
	if lostRound >= 0 {
		fmt.Printf("CONFIRMED: concurrent failed-login accounting lost updates in round %d: attempts=%d stored_count=%d locked=%t\n", lostRound, attempts, observedCount, locked)
	} else {
		fmt.Printf("NOT REPRODUCED: %d rounds of %d concurrent failures all reached the lockout threshold\n", rounds, attempts)
	}
	return nil
}

func probeConcurrentLockout(ctx context.Context, store *sqlite.Store, passwords *authn.PasswordService, rounds, attempts int) (int, int, bool, error) {
	user, err := store.GetUserByLogin(ctx, "short-password-user")
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
				_, authErr := passwords.AuthenticatePassword(groupCtx, "short-password-user", "wrong", idp.LoginMetadata{ClientID: "probe"})
				if authErr == nil || (!errors.Is(authErr, authn.ErrInvalidCredentials) && !errors.Is(authErr, authn.ErrAccountLocked)) {
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
		if state.FailedLoginCount < attempts || !locked {
			return round, state.FailedLoginCount, locked, nil
		}
	}
	return -1, attempts, true, nil
}
