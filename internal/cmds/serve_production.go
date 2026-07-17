package cmds

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"golang.org/x/sync/errgroup"

	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

type ServeProductionCommand struct {
	*cmds.CommandDescription
}

type serveProductionSettings struct {
	Addr                string   `glazed:"addr"`
	Issuer              string   `glazed:"issuer"`
	DBPath              string   `glazed:"db"`
	AuditPath           string   `glazed:"audit-path"`
	TokenSecretFile     string   `glazed:"token-secret-file"`
	TLSCertFile         string   `glazed:"tls-cert"`
	TLSKeyFile          string   `glazed:"tls-key"`
	TrustedProxyCIDRs   []string `glazed:"trusted-proxy-cidrs"`
	MaxProxyHops        int      `glazed:"max-proxy-hops"`
	RateLimit           int      `glazed:"rate-limit"`
	RateWindow          string   `glazed:"rate-window"`
	MaintenanceInterval string   `glazed:"maintenance-interval"`
	ReadHeaderTimeout   string   `glazed:"read-header-timeout"`
	ReadTimeout         string   `glazed:"read-timeout"`
	WriteTimeout        string   `glazed:"write-timeout"`
	IdleTimeout         string   `glazed:"idle-timeout"`
	ShutdownTimeout     string   `glazed:"shutdown-timeout"`
	MaxRequestBytes     int      `glazed:"max-request-bytes"`
}

func NewServeProductionCommand() (*ServeProductionCommand, error) {
	commandSettings, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	description := cmds.NewCommandDescription(
		"serve-production",
		cmds.WithShort("Run the durable production embedding host"),
		cmds.WithLong(`Run tiny-idp with the public embedded API, durable SQLite and audit stores,
bounded requests, explicit proxy trust, TLS, maintenance, and graceful shutdown.

This command intentionally reads no token secret from an environment variable
or command-line value. Put at least 32 random bytes in an owner-only file and
pass its path with --token-secret-file. Provision the database with the admin
commands before startup.

Example:
  tinyidp serve-production --addr :8443 --issuer https://idp.example.test \
    --db /var/lib/tinyidp/idp.db --audit-path /var/log/tinyidp/audit.jsonl \
    --token-secret-file /run/secrets/tinyidp-token \
    --tls-cert /run/tls/tls.crt --tls-key /run/tls/tls.key
`),
		cmds.WithFlags(
			fields.New("addr", fields.TypeString, fields.WithDefault(":8443"), fields.WithHelp("HTTPS listen address")),
			fields.New("issuer", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Canonical HTTPS issuer URL")),
			fields.New("db", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Provisioned SQLite database path")),
			fields.New("audit-path", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Synchronous JSONL audit path")),
			fields.New("token-secret-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing at least 32 random bytes")),
			fields.New("tls-cert", fields.TypeString, fields.WithRequired(true), fields.WithHelp("TLS certificate PEM path")),
			fields.New("tls-key", fields.TypeString, fields.WithRequired(true), fields.WithHelp("TLS private-key PEM path")),
			fields.New("trusted-proxy-cidrs", fields.TypeStringList, fields.WithHelp("CIDRs allowed to supply X-Forwarded-For")),
			fields.New("max-proxy-hops", fields.TypeInteger, fields.WithDefault(8), fields.WithHelp("Maximum accepted forwarded-address hops")),
			fields.New("rate-limit", fields.TypeInteger, fields.WithDefault(30), fields.WithHelp("Login attempts per account/client/address bucket and window")),
			fields.New("rate-window", fields.TypeString, fields.WithDefault("1m"), fields.WithHelp("Login rate-limit window")),
			fields.New("maintenance-interval", fields.TypeString, fields.WithDefault("15m"), fields.WithHelp("Retention maintenance interval")),
			fields.New("read-header-timeout", fields.TypeString, fields.WithDefault("5s"), fields.WithHelp("HTTP header read timeout")),
			fields.New("read-timeout", fields.TypeString, fields.WithDefault("15s"), fields.WithHelp("HTTP request read timeout")),
			fields.New("write-timeout", fields.TypeString, fields.WithDefault("30s"), fields.WithHelp("HTTP response write timeout")),
			fields.New("idle-timeout", fields.TypeString, fields.WithDefault("1m"), fields.WithHelp("HTTP keep-alive idle timeout")),
			fields.New("shutdown-timeout", fields.TypeString, fields.WithDefault("20s"), fields.WithHelp("Graceful shutdown deadline")),
			fields.New("max-request-bytes", fields.TypeInteger, fields.WithDefault(1<<20), fields.WithHelp("Maximum request body size")),
		),
		cmds.WithSections(commandSettings),
	)
	return &ServeProductionCommand{CommandDescription: description}, nil
}

func (c *ServeProductionCommand) Run(ctx context.Context, vals *values.Values) error {
	settings := &serveProductionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	return runProductionHost(ctx, settings)
}

func runProductionHost(ctx context.Context, settings *serveProductionSettings) error {
	if settings == nil {
		return fmt.Errorf("settings are required")
	}
	rateWindow, maintenanceInterval, readHeaderTimeout, readTimeout, writeTimeout, idleTimeout, shutdownTimeout, err := parseProductionDurations(settings)
	if err != nil {
		return err
	}
	if settings.RateLimit <= 0 || settings.MaxRequestBytes <= 0 {
		return fmt.Errorf("rate-limit and max-request-bytes must be positive")
	}
	secret, err := readOwnerOnlySecret(settings.TokenSecretFile)
	if err != nil {
		return err
	}
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(settings.DBPath))
	if err != nil {
		return err
	}
	audit, err := idp.NewFileAuditSink(settings.AuditPath)
	if err != nil {
		_ = store.Close()
		return err
	}
	addressResolver := idp.ClientAddressResolver(idp.DirectClientAddressResolver{})
	if len(settings.TrustedProxyCIDRs) > 0 {
		addressResolver, err = idp.NewTrustedProxyResolver(idp.TrustedProxyConfig{TrustedCIDRs: settings.TrustedProxyCIDRs, MaxHops: settings.MaxProxyHops})
		if err != nil {
			_ = audit.Close()
			_ = store.Close()
			return err
		}
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer:        settings.Issuer,
		Mode:          embeddedidp.ProductionMode,
		Store:         store,
		Cookie:        embeddedidp.CookieConfig{Secure: true, SameSite: http.SameSiteLaxMode},
		Token:         embeddedidp.TokenConfig{SecretKey: secret},
		Audit:         audit,
		RateLimiter:   idp.NewFixedWindowRateLimiter(settings.RateLimit, rateWindow),
		ClientAddress: addressResolver,
		Maintenance:   embeddedidp.MaintenanceConfig{Interval: maintenanceInterval},
	})
	for i := range secret {
		secret[i] = 0
	}
	if err != nil {
		_ = audit.Close()
		_ = store.Close()
		return err
	}
	if _, err := provider.RunMaintenance(ctx); err != nil {
		_ = provider.Close(context.Background())
		_ = audit.Close()
		_ = store.Close()
		return fmt.Errorf("initial maintenance: %w", err)
	}
	handler := http.MaxBytesHandler(provider.Handler(), int64(settings.MaxRequestBytes))
	httpServer := &http.Server{
		Addr:              settings.Addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    1 << 20,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info().Str("addr", settings.Addr).Str("issuer", settings.Issuer).Msg("tinyidp production host listening")
		if err := httpServer.ListenAndServeTLS(settings.TLSCertFile, settings.TLSKeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve production HTTPS: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		ticker := time.NewTicker(maintenanceInterval)
		defer ticker.Stop()
		for {
			select {
			case <-groupCtx.Done():
				return nil
			case <-ticker.C:
				if _, err := provider.RunMaintenance(groupCtx); err != nil && groupCtx.Err() == nil {
					log.Error().Err(err).Msg("retention maintenance failed; readiness is degraded")
				}
			}
		}
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	})
	runErr := group.Wait()
	closeErr := errors.Join(provider.Close(context.Background()), audit.Close(), store.Close())
	return errors.Join(runErr, closeErr)
}

func parseProductionDurations(settings *serveProductionSettings) (time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, time.Duration, error) {
	values := []struct {
		name string
		raw  string
	}{
		{"rate-window", settings.RateWindow}, {"maintenance-interval", settings.MaintenanceInterval},
		{"read-header-timeout", settings.ReadHeaderTimeout}, {"read-timeout", settings.ReadTimeout},
		{"write-timeout", settings.WriteTimeout}, {"idle-timeout", settings.IdleTimeout},
		{"shutdown-timeout", settings.ShutdownTimeout},
	}
	parsed := make([]time.Duration, len(values))
	for index, value := range values {
		duration, err := time.ParseDuration(value.raw)
		if err != nil || duration <= 0 {
			return 0, 0, 0, 0, 0, 0, 0, fmt.Errorf("invalid --%s duration %q", value.name, value.raw)
		}
		parsed[index] = duration
	}
	return parsed[0], parsed[1], parsed[2], parsed[3], parsed[4], parsed[5], parsed[6], nil
}

func readOwnerOnlySecret(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--token-secret-file is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat token secret file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("token secret file must be regular and owner-only (0600 or 0400)")
	}
	secret, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token secret file: %w", err)
	}
	secret = bytes.TrimSuffix(secret, []byte("\n"))
	if len(secret) < 32 {
		return nil, fmt.Errorf("token secret file must contain at least 32 bytes")
	}
	return secret, nil
}
