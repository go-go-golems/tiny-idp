package main

import (
	"context"
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/tiny-idp/examples/tinyidp-message-app/loginui"
	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type commandFactory func() (cmds.BareCommand, error)

type InitCommand struct{ *cmds.CommandDescription }

type initSettings struct {
	StateRoot     string `glazed:"state-root"`
	PublicBaseURL string `glazed:"public-base-url"`
}

var _ cmds.BareCommand = (*InitCommand)(nil)

func NewInitCommand() (cmds.BareCommand, error) {
	return &InitCommand{CommandDescription: cmds.NewCommandDescription(
		"init",
		cmds.WithShort("Initialize an owner-only message application state root"),
		cmds.WithLong("Create the durable state directory, owner-only token material, and canonical-origin manifest. It does not read configuration from the environment."),
		cmds.WithFlags(
			fields.New("state-root", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only durable application state directory")),
			fields.New("public-base-url", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Canonical browser-visible HTTP(S) application origin")),
		),
	)}, nil
}

func (c *InitCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings initSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode init settings")
	}
	manifest, err := initializeStateRoot(ctx, settings.StateRoot, settings.PublicBaseURL, time.Now())
	if err != nil {
		return errors.Wrap(err, "initialize message application state")
	}
	log.Info().Str("state_root", resolveStatePaths(settings.StateRoot).Root).Str("issuer", manifest.Issuer).Msg("message application state initialized")
	return nil
}

type ServeCommand struct{ *cmds.CommandDescription }

type serveSettings struct {
	StateRoot           string `glazed:"state-root"`
	Addr                string `glazed:"addr"`
	TLSCertificate      string `glazed:"tls-cert"`
	TLSKey              string `glazed:"tls-key"`
	MaintenanceInterval string `glazed:"maintenance-interval"`
	ShutdownTimeout     string `glazed:"shutdown-timeout"`
	ReadHeaderTimeout   string `glazed:"read-header-timeout"`
	ReadTimeout         string `glazed:"read-timeout"`
	WriteTimeout        string `glazed:"write-timeout"`
	IdleTimeout         string `glazed:"idle-timeout"`
	MaxRequestBytes     int    `glazed:"max-request-bytes"`
	ExternalIssuer      string `glazed:"external-issuer"`
	ExternalBackchannel string `glazed:"external-backchannel-url"`
}

var _ cmds.BareCommand = (*ServeCommand)(nil)

func NewServeCommand() (cmds.BareCommand, error) {
	return &ServeCommand{CommandDescription: cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run the initialized self-contained message application"),
		cmds.WithLong("Open the durable identity and message stores, reconcile the public PKCE client and signing key, run initial maintenance, then bind the application listener. HTTP public origins are accepted only on loopback for local development."),
		cmds.WithFlags(
			fields.New("state-root", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initialized state directory")),
			fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:8090"), fields.WithHelp("HTTP listen address")),
			fields.New("tls-cert", fields.TypeString, fields.WithHelp("TLS certificate PEM; required when the public origin is HTTPS")),
			fields.New("tls-key", fields.TypeString, fields.WithHelp("TLS private key PEM; required when the public origin is HTTPS")),
			fields.New("maintenance-interval", fields.TypeString, fields.WithDefault("15m")),
			fields.New("shutdown-timeout", fields.TypeString, fields.WithDefault("20s")),
			fields.New("read-header-timeout", fields.TypeString, fields.WithDefault("5s")),
			fields.New("read-timeout", fields.TypeString, fields.WithDefault("15s")),
			fields.New("write-timeout", fields.TypeString, fields.WithDefault("30s")),
			fields.New("idle-timeout", fields.TypeString, fields.WithDefault("1m")),
			fields.New("max-request-bytes", fields.TypeInteger, fields.WithDefault(1<<20)),
			fields.New("external-issuer", fields.TypeString, fields.WithHelp("Run as an external OIDC relying party against this issuer instead of embedding tiny-idp")),
			fields.New("external-backchannel-url", fields.TypeString, fields.WithHelp("Optional container-only URL for OIDC discovery, JWKS, and token requests; preserves the public issuer in protocol validation")),
		),
	)}, nil
}

func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings serveSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode serve settings")
	}
	return runMessageApplication(ctx, settings)
}

type DoctorCommand struct{ *cmds.CommandDescription }

type doctorSettings struct {
	StateRoot string `glazed:"state-root"`
}

var _ cmds.BareCommand = (*DoctorCommand)(nil)

func NewDoctorCommand() (cmds.BareCommand, error) {
	return &DoctorCommand{CommandDescription: cmds.NewCommandDescription(
		"doctor",
		cmds.WithShort("Inspect an initialized message application state root"),
		cmds.WithLong("Validate the manifest and owner-only secrets, then open both SQLite databases to check migrations and integrity-relevant startup conditions without exposing secret material."),
		cmds.WithFlags(fields.New("state-root", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initialized state directory"))),
	)}, nil
}

func (c *DoctorCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings doctorSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode doctor settings")
	}
	app, err := openInitializedMessageApplication(ctx, settings.StateRoot)
	if err != nil {
		return errors.Wrap(err, "open initialized message application")
	}
	defer func() { _ = app.Close(context.Background()) }()
	if report := app.provider.Readiness(ctx); !report.Ready {
		return fmt.Errorf("identity provider is not ready: %#v", report.Checks)
	}
	if err := app.application.db.PingContext(ctx); err != nil {
		return errors.Wrap(err, "ping application database")
	}
	log.Info().Str("state_root", app.paths.Root).Str("issuer", app.manifest.Issuer).Msg("message application doctor passed")
	return nil
}

type initializedMessageApplication struct {
	manifest    stateManifest
	paths       statePaths
	identity    *sqlitestore.Store
	application *appStore
	audit       *idp.FileAuditSink
	provider    *embeddedidp.Provider
	handler     *messageApp
}

func openInitializedMessageApplication(ctx context.Context, stateRoot string) (*initializedMessageApplication, error) {
	manifest, paths, err := validateStateRoot(stateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "validate state root")
	}
	tokenSecret, err := os.ReadFile(paths.TokenSecret)
	if err != nil {
		return nil, errors.Wrap(err, "read token secret")
	}
	identity, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(paths.IdentityDatabase))
	if err != nil {
		return nil, errors.Wrap(err, "open identity database")
	}
	closeIdentity := true
	defer func() {
		if closeIdentity {
			_ = identity.Close()
		}
	}()
	application, err := openAppStore(ctx, paths.ApplicationDatabase)
	if err != nil {
		return nil, errors.Wrap(err, "open application database")
	}
	closeApplication := true
	defer func() {
		if closeApplication {
			_ = application.Close()
		}
	}()
	audit, err := idp.NewFileAuditSink(paths.AuditLog)
	if err != nil {
		return nil, errors.Wrap(err, "open audit log")
	}
	closeAudit := true
	defer func() {
		if closeAudit {
			_ = audit.Close()
		}
	}()
	accounts, err := idpaccounts.NewService(identity, idpaccounts.Options{Audit: audit})
	if err != nil {
		return nil, errors.Wrap(err, "create account service")
	}
	mode := embeddedidp.DevMode
	cookieSecure := false
	if strings.HasPrefix(manifest.PublicBaseURL, "https://") {
		mode, cookieSecure = embeddedidp.ProductionMode, true
	}
	if _, err := embeddedidp.Bootstrap(ctx, identity, embeddedidp.BootstrapConfig{
		Mode: mode, Audit: audit,
		Clients: []embeddedidp.ClientSpec{embeddedidp.BrowserClient(clientID, []string{manifest.PublicBaseURL + callbackPath}, []string{manifest.PublicBaseURL + "/"}, []string{"openid", "profile"})},
	}); err != nil {
		return nil, errors.Wrap(err, "bootstrap identity client and signing key")
	}
	renderer, err := loginui.New()
	if err != nil {
		return nil, errors.Wrap(err, "create login interaction renderer")
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer: manifest.Issuer, Mode: mode, Store: identity,
		Cookie: embeddedidp.CookieConfig{Secure: cookieSecure, SameSite: http.SameSiteLaxMode, Path: "/"},
		Token:  embeddedidp.TokenConfig{SecretKey: tokenSecret}, Audit: audit,
		RateLimiter: idp.NewFixedWindowRateLimiter(30, time.Minute), ClientAddress: idp.DirectClientAddressResolver{},
		Authenticator: accounts, UI: embeddedidp.UIConfig{Renderer: renderer},
		AccountChooser: embeddedidp.AccountChooserConfig{
			Enabled:                 true,
			RememberOnPasswordLogin: true,
			DisplayLabel: func(user idpstore.User) (string, error) {
				if label := strings.TrimSpace(user.Name); label != "" {
					return label, nil
				}
				return user.PreferredUsername, nil
			},
		},
	})
	for index := range tokenSecret {
		tokenSecret[index] = 0
	}
	if err != nil {
		return nil, errors.Wrap(err, "create embedded identity provider")
	}
	transport, err := embeddedidp.NewInProcessIssuerTransport(manifest.Issuer, provider.Handler(), embeddedidp.InProcessTransportOptions{})
	if err != nil {
		_ = provider.Close(context.Background())
		return nil, errors.Wrap(err, "create issuer-only back-channel transport")
	}
	oidcClient, err := newOIDCClient(ctx, manifest.Issuer, manifest.PublicBaseURL, &http.Client{Transport: transport, Timeout: 10 * time.Second})
	if err != nil {
		_ = provider.Close(context.Background())
		return nil, errors.Wrap(err, "create application OIDC client")
	}
	handler := newMessageApp(application, oidcClient, accounts, provider.Handler(), cookieSecure)
	handler.audit = audit
	handler.liveness = provider.Liveness
	handler.readiness = provider.Readiness
	closeAudit, closeApplication, closeIdentity = false, false, false
	return &initializedMessageApplication{manifest: manifest, paths: paths, identity: identity, application: application, audit: audit, provider: provider, handler: handler}, nil
}

func (a *initializedMessageApplication) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	var providerErr, auditErr, applicationErr, identityErr error
	if a.provider != nil {
		providerErr = a.provider.Close(ctx)
	}
	if a.audit != nil {
		auditErr = a.audit.Close()
	}
	if a.application != nil {
		applicationErr = a.application.Close()
	}
	if a.identity != nil {
		identityErr = a.identity.Close()
	}
	return stderrors.Join(providerErr, auditErr, applicationErr, identityErr)
}

func runMessageApplication(ctx context.Context, settings serveSettings) error {
	durations, err := parseServeDurations(settings)
	if err != nil {
		return err
	}
	if settings.MaxRequestBytes < 1 {
		return errors.New("max-request-bytes must be positive")
	}
	app, err := openInitializedMessageApplication(ctx, settings.StateRoot)
	if settings.ExternalIssuer != "" {
		app, err = openExternalMessageApplication(ctx, settings.StateRoot, settings.ExternalIssuer, settings.ExternalBackchannel)
	}
	if err != nil {
		return err
	}
	defer func() { _ = app.Close(context.Background()) }()
	if app.provider != nil {
		if _, err := app.provider.RunMaintenance(ctx); err != nil {
			return errors.Wrap(err, "run initial identity maintenance")
		}
	}
	if _, err := app.application.cleanup(ctx, time.Now().UTC(), 24*time.Hour); err != nil {
		return errors.Wrap(err, "run initial application protocol-state cleanup")
	}
	if app.provider != nil {
		if report := app.provider.Readiness(ctx); !report.Ready {
			return fmt.Errorf("refuse listener while provider is not ready: %#v", report.Checks)
		}
	}
	server := &http.Server{Addr: settings.Addr, Handler: http.MaxBytesHandler(app.handler, int64(settings.MaxRequestBytes)), ReadHeaderTimeout: durations.readHeader, ReadTimeout: durations.read, WriteTimeout: durations.write, IdleTimeout: durations.idle, MaxHeaderBytes: 1 << 20, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	secureOrigin := strings.HasPrefix(app.manifest.PublicBaseURL, "https://")
	if secureOrigin != (settings.TLSCertificate != "" || settings.TLSKey != "") {
		return errors.New("--tls-cert and --tls-key are both required exactly when the public origin is HTTPS")
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		log.Info().Str("addr", settings.Addr).Str("origin", app.manifest.PublicBaseURL).Msg("message application listening")
		var serveErr error
		if secureOrigin {
			serveErr = server.ListenAndServeTLS(settings.TLSCertificate, settings.TLSKey)
		} else {
			serveErr = server.ListenAndServe()
		}
		if serveErr != nil && !stderrors.Is(serveErr, http.ErrServerClosed) {
			return errors.Wrap(serveErr, "serve message application")
		}
		return nil
	})
	group.Go(func() error {
		ticker := time.NewTicker(durations.maintenance)
		defer ticker.Stop()
		for {
			select {
			case <-groupCtx.Done():
				return nil
			case <-ticker.C:
				if app.provider != nil {
					if _, err := app.provider.RunMaintenance(groupCtx); err != nil && groupCtx.Err() == nil {
						log.Error().Err(err).Msg("identity maintenance failed; readiness degraded")
					}
				}
				if _, err := app.application.cleanup(groupCtx, time.Now().UTC(), 24*time.Hour); err != nil && groupCtx.Err() == nil {
					log.Error().Err(err).Msg("application protocol-state cleanup failed")
				}
			}
		}
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), durations.shutdown)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	})
	return group.Wait()
}

type serveDurations struct{ maintenance, shutdown, readHeader, read, write, idle time.Duration }

func parseServeDurations(settings serveSettings) (serveDurations, error) {
	values := []struct{ name, raw string }{{"maintenance-interval", settings.MaintenanceInterval}, {"shutdown-timeout", settings.ShutdownTimeout}, {"read-header-timeout", settings.ReadHeaderTimeout}, {"read-timeout", settings.ReadTimeout}, {"write-timeout", settings.WriteTimeout}, {"idle-timeout", settings.IdleTimeout}}
	parsed := make([]time.Duration, len(values))
	for index, value := range values {
		duration, err := time.ParseDuration(value.raw)
		if err != nil || duration <= 0 {
			return serveDurations{}, errors.Errorf("invalid positive --%s duration %q", value.name, value.raw)
		}
		parsed[index] = duration
	}
	return serveDurations{parsed[0], parsed[1], parsed[2], parsed[3], parsed[4], parsed[5]}, nil
}
