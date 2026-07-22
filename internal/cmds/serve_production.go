package cmds

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"golang.org/x/sync/errgroup"

	"github.com/go-go-golems/tiny-idp/internal/productionconfig"
	"github.com/go-go-golems/tiny-idp/internal/productionui"
	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge/smtpmailer"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

type ServeProductionCommand struct {
	*cmds.CommandDescription
}

type serveProductionSettings struct {
	Addr                    string   `glazed:"addr"`
	ListenerMode            string   `glazed:"listener-mode"`
	Issuer                  string   `glazed:"issuer"`
	ClientsFile             string   `glazed:"clients-file"`
	ThemeDir                string   `glazed:"theme-dir"`
	ThemeCatalogFile        string   `glazed:"theme-catalog-file"`
	SignupProgramFile       string   `glazed:"signup-program-file"`
	DBPath                  string   `glazed:"db"`
	AuditPath               string   `glazed:"audit-path"`
	TokenSecretFile         string   `glazed:"token-secret-file"`
	InvitationKeyFile       string   `glazed:"invitation-lookup-key-file"`
	EmailChallengeKeyFile   string   `glazed:"email-challenge-key-file"`
	EmailSMTPAddress        string   `glazed:"email-smtp-address"`
	EmailSMTPTLSMode        string   `glazed:"email-smtp-tls-mode"`
	EmailSMTPServerName     string   `glazed:"email-smtp-server-name"`
	EmailSMTPUsername       string   `glazed:"email-smtp-username"`
	EmailSMTPPasswordFile   string   `glazed:"email-smtp-password-file"`
	EmailFromAddress        string   `glazed:"email-from-address"`
	EmailFromName           string   `glazed:"email-from-name"`
	EmailSMTPConnectTimeout string   `glazed:"email-smtp-connect-timeout"`
	EmailSMTPSendTimeout    string   `glazed:"email-smtp-send-timeout"`
	TLSCertFile             string   `glazed:"tls-cert"`
	TLSKeyFile              string   `glazed:"tls-key"`
	TrustedProxyCIDRs       []string `glazed:"trusted-proxy-cidrs"`
	MaxProxyHops            int      `glazed:"max-proxy-hops"`
	AccountChooser          bool     `glazed:"account-chooser"`
	RateLimit               int      `glazed:"rate-limit"`
	RateWindow              string   `glazed:"rate-window"`
	MaintenanceInterval     string   `glazed:"maintenance-interval"`
	ReadHeaderTimeout       string   `glazed:"read-header-timeout"`
	ReadTimeout             string   `glazed:"read-timeout"`
	WriteTimeout            string   `glazed:"write-timeout"`
	IdleTimeout             string   `glazed:"idle-timeout"`
	ShutdownTimeout         string   `glazed:"shutdown-timeout"`
	MaxRequestBytes         int      `glazed:"max-request-bytes"`
}

const maxProductionSignupProgramBytes = 256 << 10

func NewServeProductionCommand() (*ServeProductionCommand, error) {
	commandSettings, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	description := cmds.NewCommandDescription(
		"serve-production",
		cmds.WithShort("Run the durable production embedding host"),
		cmds.WithLong(`Run tiny-idp with the public embedded API, durable SQLite and audit stores,
bounded requests, an explicit listener mode, maintenance, and graceful shutdown.

This command intentionally reads no token secret from an environment variable
or command-line value. Put at least 32 random bytes in an owner-only file and
pass its path with --token-secret-file. Provision the database with the admin
commands before startup. The required --signup-program-file is reviewed,
non-secret JavaScript; startup checks and warms it before accepting traffic.

Example:
  tinyidp serve-production --addr :8443 --issuer https://idp.example.test \
    --db /var/lib/tinyidp/idp.db --audit-path /var/log/tinyidp/audit.jsonl \
    --token-secret-file /run/secrets/tinyidp-token \
    --clients-file /etc/tinyidp/catalog/clients.json \
    --theme-dir /etc/tinyidp/themes \
    --theme-catalog-file /etc/tinyidp/themes/themes.json \
    --signup-program-file /etc/tinyidp/signup.js \
    --tls-cert /run/tls/tls.crt --tls-key /run/tls/tls.key
`),
		cmds.WithFlags(
			fields.New("addr", fields.TypeString, fields.WithDefault(":8443"), fields.WithHelp("Listener address")),
			fields.New("listener-mode", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Required listener mode: direct-tls or trusted-proxy-http")),
			fields.New("issuer", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Canonical HTTPS issuer URL")),
			fields.New("clients-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Reviewed non-secret JSON catalog of exact production browser clients")),
			fields.New("theme-dir", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Read-only root containing reviewed production theme CSS")),
			fields.New("theme-catalog-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Reviewed non-secret JSON theme catalog inside --theme-dir")),
			fields.New("signup-program-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Reviewed non-secret JavaScript signup program; checked and activated before listening")),
			fields.New("db", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Provisioned SQLite database path")),
			fields.New("audit-path", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Synchronous JSONL audit path")),
			fields.New("token-secret-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing at least 32 random bytes")),
			fields.New("invitation-lookup-key-file", fields.TypeString, fields.WithHelp("Owner-only 32-byte HMAC key; required when the signup program declares a durable invitation provider")),
			fields.New("email-challenge-key-file", fields.TypeString, fields.WithHelp("Owner-only 32-byte HMAC key; required when the signup program declares an email challenge")),
			fields.New("email-smtp-address", fields.TypeString, fields.WithHelp("SMTP submission host:port; required for email-challenge signup")),
			fields.New("email-smtp-tls-mode", fields.TypeString, fields.WithHelp("SMTP transport: starttls, implicit, or private-plaintext")),
			fields.New("email-smtp-server-name", fields.TypeString, fields.WithHelp("Optional TLS server name; defaults to the SMTP address host")),
			fields.New("email-smtp-username", fields.TypeString, fields.WithHelp("Optional SMTP username; requires --email-smtp-password-file and TLS")),
			fields.New("email-smtp-password-file", fields.TypeString, fields.WithHelp("Owner-only SMTP password file; required with --email-smtp-username")),
			fields.New("email-from-address", fields.TypeString, fields.WithHelp("Fixed sender mailbox for email challenges")),
			fields.New("email-from-name", fields.TypeString, fields.WithDefault("TinyIDP"), fields.WithHelp("Fixed sender display name")),
			fields.New("email-smtp-connect-timeout", fields.TypeString, fields.WithDefault("5s"), fields.WithHelp("SMTP connection timeout")),
			fields.New("email-smtp-send-timeout", fields.TypeString, fields.WithDefault("15s"), fields.WithHelp("Complete SMTP exchange timeout")),
			fields.New("tls-cert", fields.TypeString, fields.WithHelp("TLS certificate PEM path; required only for direct-tls")),
			fields.New("tls-key", fields.TypeString, fields.WithHelp("TLS private-key PEM path; required only for direct-tls")),
			fields.New("trusted-proxy-cidrs", fields.TypeStringList, fields.WithHelp("Required only for trusted-proxy-http; narrow CIDRs allowed to supply forwarded metadata")),
			fields.New("max-proxy-hops", fields.TypeInteger, fields.WithDefault(8), fields.WithHelp("Maximum accepted forwarded-address hops")),
			fields.New("account-chooser", fields.TypeBool, fields.WithHelp("Offer remembered signed-in accounts when an OIDC client requests prompt=select_account; uses each account's display name as its deliberate browser-visible label")),
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
	listenerMode, err := parseProductionListenerMode(settings.ListenerMode)
	if err != nil {
		return err
	}
	if err := validateProductionListenerSettings(listenerMode, settings); err != nil {
		return err
	}
	secret, err := readOwnerOnlySecret(settings.TokenSecretFile)
	if err != nil {
		return err
	}
	defer clearProductionSecret(secret)
	signupSource, err := readProductionSignupProgram(settings.SignupProgramFile)
	if err != nil {
		return err
	}
	signupArtifact, err := idpsignup.Compile(ctx, signupSource)
	if err != nil {
		return fmt.Errorf("check signup program: %w", err)
	}
	signupProgram := signupArtifact.Program()
	var invitationLookupKey []byte
	if productionProgramRequiresDurableInvitations(signupProgram) {
		invitationLookupKey, err = readOwnerOnlySecret(settings.InvitationKeyFile)
		if err != nil {
			return fmt.Errorf("read invitation lookup key: %w", err)
		}
		defer clearProductionSecret(invitationLookupKey)
	}
	clientCatalog, err := productionconfig.LoadClientCatalog(settings.ClientsFile)
	if err != nil {
		return err
	}
	themeCatalog, err := productionui.LoadCatalog(settings.ThemeDir, settings.ThemeCatalogFile, clientCatalog)
	if err != nil {
		return err
	}
	interactionUI, err := productionui.NewRenderer(themeCatalog)
	if err != nil {
		return err
	}
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(settings.DBPath))
	if err != nil {
		return err
	}
	var durableInvitations *idpinvite.DurableService
	if len(invitationLookupKey) != 0 {
		durableInvitations, err = idpinvite.NewDurableService(store, invitationLookupKey)
		if err != nil {
			_ = store.Close()
			return fmt.Errorf("construct durable invitation service: %w", err)
		}
		clearProductionSecret(invitationLookupKey)
	}
	emailChallenges, err := newProductionEmailChallenges(settings, store, signupProgram)
	if err != nil {
		_ = store.Close()
		return err
	}
	audit, err := idp.NewFileAuditSink(settings.AuditPath)
	if err != nil {
		_ = store.Close()
		return err
	}
	signupManager, err := newProductionSignupManager(ctx, signupSource, audit, productionSignupServices{EmailChallenges: emailChallenges != nil, DisplayNameLookup: true})
	if err != nil {
		_ = audit.Close()
		_ = store.Close()
		return err
	}
	if _, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{Mode: idpstore.ProductionMode, Audit: audit, Clients: clientCatalog.Specs()}); err != nil {
		_ = signupManager.Close(context.Background())
		_ = audit.Close()
		_ = store.Close()
		return fmt.Errorf("bootstrap production browser clients: %w", err)
	}
	addressResolver := idp.ClientAddressResolver(idp.DirectClientAddressResolver{})
	var proxyResolver *idp.TrustedProxyResolver
	if listenerMode == productionListenerTrustedProxyHTTP {
		proxyResolver, err = idp.NewTrustedProxyResolver(idp.TrustedProxyConfig{TrustedCIDRs: settings.TrustedProxyCIDRs, MaxHops: settings.MaxProxyHops})
		if err != nil {
			_ = signupManager.Close(context.Background())
			_ = audit.Close()
			_ = store.Close()
			return err
		}
		addressResolver = proxyResolver
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
		ScriptedSignup: embeddedidp.ScriptedSignupConfig{
			GenerationManager:  signupManager,
			DurableInvitations: durableInvitations,
			EmailChallenges:    emailChallenges,
		},
		Maintenance: embeddedidp.MaintenanceConfig{Interval: maintenanceInterval},
		UI:          embeddedidp.UIConfig{Renderer: interactionUI, WorkflowRenderer: interactionUI},
		AccountChooser: embeddedidp.AccountChooserConfig{
			Enabled:                 settings.AccountChooser,
			RememberOnPasswordLogin: settings.AccountChooser,
			DisplayLabel: func(user idpstore.User) (string, error) {
				if label := strings.TrimSpace(user.Name); label != "" {
					return label, nil
				}
				return user.PreferredUsername, nil
			},
		},
	})
	clearProductionSecret(secret)
	if err != nil {
		_ = signupManager.Close(context.Background())
		_ = audit.Close()
		_ = store.Close()
		return err
	}
	if _, err := provider.RunMaintenance(ctx); err != nil {
		_ = provider.Close(context.Background())
		_ = signupManager.Close(context.Background())
		_ = audit.Close()
		_ = store.Close()
		return fmt.Errorf("initial maintenance: %w", err)
	}
	handler := productionHTTPHandler(provider.Handler(), themeCatalog.AssetsHandler(), settings.MaxRequestBytes)
	if listenerMode == productionListenerTrustedProxyHTTP {
		publicOrigin, originErr := issuerOrigin(settings.Issuer)
		if originErr != nil {
			_ = provider.Close(context.Background())
			_ = signupManager.Close(context.Background())
			_ = audit.Close()
			_ = store.Close()
			return originErr
		}
		handler, originErr = idp.NewTrustedProxyHTTPHandler(idp.TrustedProxyHTTPConfig{PublicOrigin: publicOrigin, Resolver: proxyResolver}, handler)
		if originErr != nil {
			_ = provider.Close(context.Background())
			_ = audit.Close()
			_ = store.Close()
			return originErr
		}
	}
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
		log.Info().Str("addr", settings.Addr).Str("issuer", settings.Issuer).Str("listener_mode", string(listenerMode)).Msg("tinyidp production host listening")
		var serveErr error
		if listenerMode == productionListenerDirectTLS {
			serveErr = httpServer.ListenAndServeTLS(settings.TLSCertFile, settings.TLSKeyFile)
		} else {
			serveErr = httpServer.ListenAndServe()
		}
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve production listener: %w", serveErr)
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
	closeErr := errors.Join(provider.Close(context.Background()), signupManager.Close(context.Background()), audit.Close(), store.Close())
	return errors.Join(runErr, closeErr)
}

// productionHTTPHandler keeps reviewed interaction assets on the provider's
// own origin. Every OAuth/OIDC and form route remains owned by the embedded
// provider handler.
func productionHTTPHandler(providerHandler, assetsHandler http.Handler, maxRequestBytes int) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/themes/", assetsHandler)
	mux.Handle("/", providerHandler)
	return http.MaxBytesHandler(mux, int64(maxRequestBytes))
}

func clearProductionSecret(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}

func readProductionSignupProgram(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("--signup-program-file is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open signup program file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat signup program file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("signup program file must be a regular file")
	}
	if info.Size() > maxProductionSignupProgramBytes {
		return "", fmt.Errorf("signup program file exceeds %d bytes", maxProductionSignupProgramBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxProductionSignupProgramBytes+1))
	if err != nil {
		return "", fmt.Errorf("read signup program file: %w", err)
	}
	if len(data) > maxProductionSignupProgramBytes {
		return "", fmt.Errorf("signup program file exceeds %d bytes", maxProductionSignupProgramBytes)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return "", fmt.Errorf("signup program file must not be empty")
	}
	return string(data), nil
}

type productionSignupServices struct {
	EmailChallenges   bool
	DisplayNameLookup bool
}

func newProductionSignupManager(ctx context.Context, source string, audit idp.Sink, services productionSignupServices) (*idpsignup.GenerationManager, error) {
	_, err := checkProductionSignupProgram(ctx, source, services)
	if err != nil {
		return nil, err
	}
	manager, err := idpsignup.NewGenerationManagerWithOptions(ctx, source, 1, 1, idpsignup.GenerationManagerOptions{Audit: audit})
	if err != nil {
		return nil, fmt.Errorf("activate signup program: %w", err)
	}
	if err := manager.Ready(); err != nil {
		_ = manager.Close(context.Background())
		return nil, fmt.Errorf("active signup program is unavailable: %w", err)
	}
	return manager, nil
}

func checkProductionSignupProgram(ctx context.Context, source string, services productionSignupServices) (idpprogram.Program, error) {
	artifact, err := idpsignup.Compile(ctx, source)
	if err != nil {
		return idpprogram.Program{}, fmt.Errorf("check signup program: %w", err)
	}
	program := artifact.Program()
	if err := validateProductionSignupProgram(program, services); err != nil {
		return idpprogram.Program{}, err
	}
	return program, nil
}

func validateProductionSignupProgram(program idpprogram.Program, services productionSignupServices) error {
	unsupportedCapabilities := make([]string, 0)
	for id, requirement := range program.Capabilities {
		supported := id == idpinvite.LookupCapabilityID && requirement.Version == idpinvite.LookupCapabilityVersion ||
			id == idpaccounts.DisplayNameLookupCapabilityID && requirement.Version == idpaccounts.DisplayNameLookupCapabilityVersion && services.DisplayNameLookup
		if !supported {
			unsupportedCapabilities = append(unsupportedCapabilities, id)
		}
	}
	if len(unsupportedCapabilities) != 0 {
		sort.Strings(unsupportedCapabilities)
		return fmt.Errorf("signup program declares unsupported native capabilities: %s", strings.Join(unsupportedCapabilities, ", "))
	}
	if err := validateProductionSignupCapabilityBindings(program, services); err != nil {
		return err
	}
	durableProvider := false
	for _, provider := range program.Providers {
		if provider.Kind != idpprogram.ProviderKindInvitation || provider.State != idpprogram.ProviderStateDurable {
			continue
		}
		durableProvider = true
		handler, ok := provider.Handlers[idpprogram.InvitationValidateHandler]
		lambda, lambdaOK := program.Lambdas[handler.LambdaID]
		if !ok || !lambdaOK || !lambdaRequiresCapability(lambda, idpinvite.LookupCapabilityID, idpinvite.LookupCapabilityVersion) {
			return fmt.Errorf("durable invitation provider %q must bind validate to invitation.lookup@v1", provider.ID)
		}
	}
	if _, declared := program.Capabilities[idpinvite.LookupCapabilityID]; declared && !durableProvider {
		return fmt.Errorf("signup program declares invitation.lookup without a durable invitation provider")
	}
	unsupported := map[string]struct{}{}
	usesInvitationEffect := false
	for _, lambda := range program.Lambdas {
		for _, outcome := range lambda.AllowedOutcomes {
			if outcome == idpprogram.OutcomeChallenge && !services.EmailChallenges {
				unsupported["email_challenge"] = struct{}{}
			}
		}
		for _, effect := range lambda.AllowedEffects {
			if effect == idpprogram.EffectConsumeInvitation {
				usesInvitationEffect = true
				continue
			}
			if effect != idpprogram.EffectCreateLocalIdentity && effect != idpprogram.EffectAttachPasswordCredential {
				unsupported["effect:"+string(effect)] = struct{}{}
			}
		}
	}
	if usesInvitationEffect && !durableProvider {
		unsupported["effect:consumeInvitation_without_durable_provider"] = struct{}{}
	}
	if durableProvider && !usesInvitationEffect {
		unsupported["durable_invitation_provider_without_consumeInvitation"] = struct{}{}
	}
	if len(unsupported) == 0 {
		return nil
	}
	ids := make([]string, 0, len(unsupported))
	for id := range unsupported {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return fmt.Errorf("signup program declares unsupported native services: %s", strings.Join(ids, ", "))
}

func validateProductionSignupCapabilityBindings(program idpprogram.Program, services productionSignupServices) error {
	unsupported := make([]string, 0)
	for workflowID, workflow := range program.Workflows {
		for handlerID, handler := range workflow.Handlers {
			lambda := program.Lambdas[handler.LambdaID]
			for _, requirement := range lambda.RequiredCapabilities {
				if handlerID == workflow.EntryHandler || requirement.ID != idpaccounts.DisplayNameLookupCapabilityID || requirement.Version != idpaccounts.DisplayNameLookupCapabilityVersion || !services.DisplayNameLookup {
					unsupported = append(unsupported, fmt.Sprintf("workflow %s handler %s: %s@v%d", workflowID, handlerID, requirement.ID, requirement.Version))
				}
			}
		}
	}
	for providerID, provider := range program.Providers {
		for handlerID, handler := range provider.Handlers {
			lambda := program.Lambdas[handler.LambdaID]
			for _, requirement := range lambda.RequiredCapabilities {
				if provider.Kind != idpprogram.ProviderKindInvitation || provider.State != idpprogram.ProviderStateDurable || requirement.ID != idpinvite.LookupCapabilityID || requirement.Version != idpinvite.LookupCapabilityVersion {
					unsupported = append(unsupported, fmt.Sprintf("provider %s handler %s: %s@v%d", providerID, handlerID, requirement.ID, requirement.Version))
				}
			}
		}
	}
	if len(unsupported) == 0 {
		return nil
	}
	sort.Strings(unsupported)
	return fmt.Errorf("signup program requires capabilities unavailable on their invocation paths: %s", strings.Join(unsupported, ", "))
}

func productionProgramRequiresDurableInvitations(program idpprogram.Program) bool {
	for _, provider := range program.Providers {
		if provider.Kind == idpprogram.ProviderKindInvitation && provider.State == idpprogram.ProviderStateDurable {
			return true
		}
	}
	return false
}

func productionProgramRequiresEmailChallenges(program idpprogram.Program) bool {
	for _, lambda := range program.Lambdas {
		for _, outcome := range lambda.AllowedOutcomes {
			if outcome == idpprogram.OutcomeChallenge {
				return true
			}
		}
	}
	return false
}

func newProductionEmailChallenges(settings *serveProductionSettings, store idpemailchallenge.Store, program idpprogram.Program) (*idpemailchallenge.Service, error) {
	required := productionProgramRequiresEmailChallenges(program)
	configured := strings.TrimSpace(settings.EmailChallengeKeyFile) != "" || strings.TrimSpace(settings.EmailSMTPAddress) != "" || strings.TrimSpace(settings.EmailSMTPTLSMode) != "" || strings.TrimSpace(settings.EmailSMTPServerName) != "" || strings.TrimSpace(settings.EmailSMTPUsername) != "" || strings.TrimSpace(settings.EmailSMTPPasswordFile) != "" || strings.TrimSpace(settings.EmailFromAddress) != ""
	if !required {
		if configured {
			return nil, errors.New("email delivery flags require a signup program that declares an email challenge")
		}
		return nil, nil
	}
	if store == nil {
		return nil, errors.New("email challenge signup requires a durable challenge store")
	}
	if strings.TrimSpace(settings.EmailChallengeKeyFile) == "" || strings.TrimSpace(settings.EmailSMTPAddress) == "" || strings.TrimSpace(settings.EmailSMTPTLSMode) == "" || strings.TrimSpace(settings.EmailFromAddress) == "" {
		return nil, errors.New("email challenge signup requires --email-challenge-key-file, --email-smtp-address, --email-smtp-tls-mode, and --email-from-address")
	}
	connectTimeout, err := positiveDurationFlag("email-smtp-connect-timeout", settings.EmailSMTPConnectTimeout)
	if err != nil {
		return nil, err
	}
	sendTimeout, err := positiveDurationFlag("email-smtp-send-timeout", settings.EmailSMTPSendTimeout)
	if err != nil {
		return nil, err
	}
	key, err := readOwnerOnlyFile(settings.EmailChallengeKeyFile, "email challenge key", 32)
	if err != nil {
		return nil, err
	}
	defer clearProductionSecret(key)
	var password []byte
	if strings.TrimSpace(settings.EmailSMTPUsername) != "" || strings.TrimSpace(settings.EmailSMTPPasswordFile) != "" {
		if strings.TrimSpace(settings.EmailSMTPUsername) == "" || strings.TrimSpace(settings.EmailSMTPPasswordFile) == "" {
			return nil, errors.New("--email-smtp-username and --email-smtp-password-file must be configured together")
		}
		password, err = readOwnerOnlyFile(settings.EmailSMTPPasswordFile, "SMTP password", 1)
		if err != nil {
			return nil, err
		}
		defer clearProductionSecret(password)
	}
	mailer, err := smtpmailer.New(smtpmailer.Config{
		Address: settings.EmailSMTPAddress, TLSMode: smtpmailer.TLSMode(settings.EmailSMTPTLSMode), ServerName: settings.EmailSMTPServerName,
		Username: settings.EmailSMTPUsername, Password: password, FromAddress: settings.EmailFromAddress, FromName: settings.EmailFromName,
		ConnectTimeout: connectTimeout, SendTimeout: sendTimeout, Templates: smtpmailer.SignupTemplates(),
	})
	if err != nil {
		return nil, fmt.Errorf("construct SMTP email challenge mailer: %w", err)
	}
	service, err := idpemailchallenge.NewService(store, mailer, key)
	if err != nil {
		return nil, fmt.Errorf("construct durable email challenge service: %w", err)
	}
	return service, nil
}

func positiveDurationFlag(name, raw string) (time.Duration, error) {
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid --%s duration %q", name, raw)
	}
	return duration, nil
}

func lambdaRequiresCapability(lambda idpprogram.LambdaSpec, id string, version uint32) bool {
	for _, requirement := range lambda.RequiredCapabilities {
		if requirement.ID == id && requirement.Version == version {
			return true
		}
	}
	return false
}

type productionListenerMode string

const (
	productionListenerDirectTLS        productionListenerMode = "direct-tls"
	productionListenerTrustedProxyHTTP productionListenerMode = "trusted-proxy-http"
)

func parseProductionListenerMode(raw string) (productionListenerMode, error) {
	mode := productionListenerMode(strings.TrimSpace(raw))
	if mode != productionListenerDirectTLS && mode != productionListenerTrustedProxyHTTP {
		return "", fmt.Errorf("--listener-mode must be direct-tls or trusted-proxy-http")
	}
	return mode, nil
}

func validateProductionListenerSettings(mode productionListenerMode, settings *serveProductionSettings) error {
	if mode == productionListenerDirectTLS {
		if settings.TLSCertFile == "" || settings.TLSKeyFile == "" || len(settings.TrustedProxyCIDRs) != 0 {
			return fmt.Errorf("direct-tls requires --tls-cert and --tls-key and forbids --trusted-proxy-cidrs")
		}
		return nil
	}
	issuer, err := url.Parse(settings.Issuer)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" || len(settings.TrustedProxyCIDRs) == 0 || settings.TLSCertFile != "" || settings.TLSKeyFile != "" {
		return fmt.Errorf("trusted-proxy-http requires an HTTPS issuer and --trusted-proxy-cidrs and forbids TLS certificate flags")
	}
	return nil
}

func issuerOrigin(raw string) (string, error) {
	issuer, err := url.Parse(raw)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" {
		return "", fmt.Errorf("issuer must have an HTTPS origin")
	}
	return issuer.Scheme + "://" + issuer.Host, nil
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
	return readOwnerOnlyFile(path, "token secret", 32)
}

func readOwnerOnlyFile(path, label string, minimumBytes int) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%s file is required", label)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s file: %w", label, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%s file must be regular and owner-only (0600 or 0400)", label)
	}
	secret, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s file: %w", label, err)
	}
	secret = bytes.TrimSuffix(secret, []byte("\n"))
	if len(secret) < minimumBytes {
		return nil, fmt.Errorf("%s file must contain at least %d bytes", label, minimumBytes)
	}
	return secret, nil
}
