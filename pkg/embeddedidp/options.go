package embeddedidp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/oidcmeta"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

type Mode = idpstore.Mode

const (
	DevMode        = idpstore.DevMode
	ProductionMode = idpstore.ProductionMode
)

type CookieConfig struct {
	Secure      bool
	SameSite    http.SameSite
	SessionName string
	CSRFName    string
	Path        string
}

type TokenConfig struct {
	SecretKey []byte
}

type UIConfig struct {
	Renderer                   idpui.InteractionRenderer
	WorkflowRenderer           idpui.WorkflowRenderer
	DeviceVerificationRenderer idpui.DeviceVerificationRenderer
}

// ScriptedSignupConfig threads an activated, bounded signup generation and
// its continuation store into the embedded provider. It has no listener,
// TLS, OAuth, cookie, key, or general storage authority.
type ScriptedSignupConfig struct {
	Executor           *idpsignup.Executor
	GenerationManager  *idpsignup.GenerationManager
	Continuations      idpcontinuation.Store
	DurableInvitations *idpinvite.DurableService
	EmailChallenges    *idpemailchallenge.Service
}

// RegistrationConfig enables provider-owned self-registration as a continuation
// of an already-validated authorization request. The provider remains the only
// component with access to the password/account-creation service.
type RegistrationConfig struct {
	Enabled  bool
	Accounts *idpaccounts.Service
}

// AccountChooserConfig opts a host into provider-owned remembered-account
// state. Remembering stays disabled unless explicitly selected because labels
// can reveal prior browser use. When remembering is enabled, DisplayLabel must
// return a deliberate, safe label for each user.
type AccountChooserConfig struct {
	Enabled                 bool
	ContextCookieName       string
	ContextTTL              time.Duration
	MaxRememberedAccounts   int
	RememberOnPasswordLogin bool
	DisplayLabel            func(idpstore.User) (string, error)
}

// MaintenanceConfig makes retention and the host scheduling contract explicit.
// Zero values select conservative defaults derived from client token lifetimes.
type MaintenanceConfig struct {
	Interval               time.Duration
	RetainExpiredFor       time.Duration
	ProtocolStateRetention time.Duration
	SigningKeyRetention    time.Duration
}

type Options struct {
	Issuer         string
	Mode           Mode
	Store          idpstore.Store
	Cookie         CookieConfig
	Token          TokenConfig
	Audit          idp.Sink
	Consent        idp.ConsentPolicy
	RateLimiter    idp.RateLimiter
	ClientAddress  idp.ClientAddressResolver
	Authenticator  idp.PasswordAuthenticator
	PasswordPolicy idp.PasswordAcceptancePolicy
	PasswordWork   idp.PasswordWorkConfig
	Maintenance    MaintenanceConfig
	UI             UIConfig
	AccountChooser AccountChooserConfig
	Registration   RegistrationConfig
	ScriptedSignup ScriptedSignupConfig
}

func (o Options) Validate(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mode := o.Mode
	if mode == "" {
		mode = DevMode
	}
	if err := oidcmeta.ValidateIssuer(o.Issuer, mode); err != nil {
		return err
	}
	issuerURL, err := url.Parse(o.Issuer)
	if err != nil {
		return fmt.Errorf("parse issuer for cookie configuration: %w", err)
	}
	if o.Cookie.Path != "" && !cookiePathCoversIssuer(o.Cookie.Path, issuerURL.EscapedPath()) {
		return fmt.Errorf("cookie path %q does not cover issuer path %q", o.Cookie.Path, issuerURL.EscapedPath())
	}
	if o.Store == nil {
		return fmt.Errorf("store is required")
	}
	if err := validateCookieConfig(o.Cookie); err != nil {
		return err
	}
	if err := validateAccountChooserConfig(o.Cookie, o.AccountChooser); err != nil {
		return err
	}
	clients, err := o.Store.ListClients(ctx)
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}
	for _, c := range clients {
		if err := c.Validate(mode); err != nil {
			return fmt.Errorf("client %q: %w", c.ID, err)
		}
	}
	maintenance, err := normalizeMaintenance(o.Maintenance, clients)
	if err != nil {
		return err
	}
	if mode == ProductionMode {
		if len(o.Token.SecretKey) < 32 {
			return fmt.Errorf("production mode requires token secret key of at least 32 bytes")
		}
		if !o.Cookie.Secure {
			return fmt.Errorf("production cookies must be secure")
		}
		sameSite := o.Cookie.SameSite
		if sameSite == 0 {
			sameSite = http.SameSiteLaxMode
		}
		if sameSite != http.SameSiteLaxMode && sameSite != http.SameSiteStrictMode && sameSite != http.SameSiteNoneMode {
			return fmt.Errorf("production cookies require an explicit supported SameSite policy")
		}
		if o.Audit == nil {
			return fmt.Errorf("production mode requires an audit sink")
		}
		audit, ok := o.Audit.(idp.AuditReporter)
		if !ok || !audit.ProductionReady() {
			return fmt.Errorf("production mode requires a durable audit reporter")
		}
		if health := audit.AuditHealth(ctx); !health.Ready {
			return fmt.Errorf("production audit is not ready: %s", health.Reason)
		}
		if o.RateLimiter == nil {
			return fmt.Errorf("production mode requires a rate limiter")
		}
		if o.ClientAddress == nil {
			return fmt.Errorf("production mode requires a client address resolver")
		}
		if reporter, ok := o.RateLimiter.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
			return fmt.Errorf("production mode requires a production-ready rate limiter")
		}
		if reporter, ok := o.ClientAddress.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
			return fmt.Errorf("production mode requires a production-ready client address resolver")
		}
		policy := o.PasswordPolicy
		if policy.MinCharacters == 0 {
			policy = idp.DefaultPasswordAcceptancePolicy()
		}
		if policy.MinCharacters < 15 || policy.MaxCharacters < 64 || policy.Blocklist == nil {
			return fmt.Errorf("production mode requires NIST-aligned password acceptance policy")
		}
		work := o.PasswordWork
		if work.MaxConcurrent == 0 {
			work = idp.DefaultPasswordWorkConfig()
		}
		if work.MaxConcurrent < 1 {
			return fmt.Errorf("production mode requires bounded password work")
		}
		if o.Authenticator != nil {
			if reporter, ok := o.Authenticator.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
				return fmt.Errorf("production mode requires a production-ready password authenticator")
			}
			if _, ok := o.Authenticator.(idp.PasswordWorkReporter); !ok {
				return fmt.Errorf("production mode requires password work metrics")
			}
		}
		reporter, ok := o.Store.(idpstore.PersistentReporter)
		if !ok || !reporter.Persistent() {
			return fmt.Errorf("production mode requires persistent stores")
		}
		schema, ok := o.Store.(idpstore.SchemaReporter)
		if !ok {
			return fmt.Errorf("production mode requires schema reporting")
		}
		version, err := schema.SchemaVersion(ctx)
		if err != nil || version != schema.SupportedSchemaVersion() || version <= 0 {
			return fmt.Errorf("production mode requires a supported schema")
		}
		if _, ok := o.Store.(idpstore.MaintenanceStore); !ok {
			return fmt.Errorf("production mode requires store maintenance support")
		}
		_ = maintenance
		key, err := o.Store.ActiveSigningKey(ctx)
		if err != nil {
			return fmt.Errorf("production mode requires active signing key: %w", err)
		}
		now := time.Now().UTC()
		if key.Algorithm != "RS256" || key.ID == "" || now.Before(key.NotBefore) || (!key.NotAfter.IsZero() && !now.Before(key.NotAfter)) {
			return fmt.Errorf("production mode requires a currently usable RS256 signing key")
		}
		privateKey, err := keys.ParseRSAPrivateKey(key)
		if err != nil || privateKey.N.BitLen() < 2048 {
			return fmt.Errorf("production mode requires a valid RSA signing key of at least 2048 bits")
		}
		verificationKeys, err := o.Store.VerificationKeys(ctx)
		if err != nil {
			return fmt.Errorf("production mode requires verification keys: %w", err)
		}
		active := 0
		for _, verificationKey := range verificationKeys {
			if verificationKey.Active {
				active++
			}
			if verificationKey.ID == "" || verificationKey.Algorithm != "RS256" {
				return fmt.Errorf("production verification key metadata is invalid")
			}
			parsed, err := keys.ParseRSAPrivateKey(verificationKey)
			if err != nil || parsed.N.BitLen() < 2048 {
				return fmt.Errorf("production verification key %q is invalid", verificationKey.ID)
			}
			if !verificationKey.Active && verificationKey.NotAfter.IsZero() {
				return fmt.Errorf("retired verification key %q has no retirement time", verificationKey.ID)
			}
		}
		if active != 1 {
			return fmt.Errorf("production mode requires exactly one active signing key")
		}
	}
	return nil
}

func validateCookieConfig(cfg CookieConfig) error {
	for label, name := range map[string]string{"session": cfg.SessionName, "csrf": cfg.CSRFName} {
		if name == "" {
			continue
		}
		if strings.TrimSpace(name) != name || strings.ContainsAny(name, "\x00\r\n\t ;,=") {
			return fmt.Errorf("%s cookie name is invalid", label)
		}
	}
	if cfg.SessionName != "" && cfg.SessionName == cfg.CSRFName {
		return fmt.Errorf("session and csrf cookie names must differ")
	}
	if cfg.Path != "" && (!strings.HasPrefix(cfg.Path, "/") || strings.ContainsAny(cfg.Path, "\x00\r\n;")) {
		return fmt.Errorf("cookie path must be an absolute HTTP path")
	}
	return nil
}

func validateAccountChooserConfig(cookie CookieConfig, cfg AccountChooserConfig) error {
	if !cfg.Enabled {
		return nil
	}
	sessionName := cookie.SessionName
	if sessionName == "" {
		sessionName = "tinyidp_session"
	}
	csrfName := cookie.CSRFName
	if csrfName == "" {
		csrfName = "tinyidp_csrf"
	}
	if cfg.ContextCookieName != "" {
		if strings.TrimSpace(cfg.ContextCookieName) != cfg.ContextCookieName || strings.ContainsAny(cfg.ContextCookieName, "\x00\r\n\t ;,=") {
			return fmt.Errorf("browser context cookie name is invalid")
		}
		if cfg.ContextCookieName == sessionName || cfg.ContextCookieName == csrfName {
			return fmt.Errorf("browser context cookie name must differ from session and csrf cookie names")
		}
	}
	if cfg.ContextTTL < 0 {
		return fmt.Errorf("browser context TTL must be positive")
	}
	if cfg.MaxRememberedAccounts < 0 || cfg.MaxRememberedAccounts > 20 {
		return fmt.Errorf("maximum remembered accounts must be between 1 and 20")
	}
	if cfg.RememberOnPasswordLogin && cfg.DisplayLabel == nil {
		return fmt.Errorf("remembered password logins require an account display-label policy")
	}
	return nil
}

func cookiePathCoversIssuer(cookiePath, issuerPath string) bool {
	if cookiePath == "/" {
		return true
	}
	issuerPath = strings.TrimSuffix(issuerPath, "/")
	return issuerPath == cookiePath || strings.HasPrefix(issuerPath, strings.TrimSuffix(cookiePath, "/")+"/")
}

func normalizeMaintenance(cfg MaintenanceConfig, clients []idpstore.Client) (MaintenanceConfig, error) {
	if cfg.Interval == 0 {
		cfg.Interval = 15 * time.Minute
	}
	if cfg.RetainExpiredFor == 0 {
		cfg.RetainExpiredFor = 24 * time.Hour
	}
	if cfg.Interval <= 0 || cfg.RetainExpiredFor < 0 {
		return MaintenanceConfig{}, fmt.Errorf("maintenance interval must be positive and expired retention non-negative")
	}
	maxRefresh := time.Duration(0)
	maxID := time.Duration(0)
	for _, client := range clients {
		if client.RefreshTokenTTL > maxRefresh {
			maxRefresh = client.RefreshTokenTTL
		}
		if client.IDTokenTTL > maxID {
			maxID = client.IDTokenTTL
		}
	}
	minimumProtocol := maxRefresh + cfg.RetainExpiredFor
	if minimumProtocol == cfg.RetainExpiredFor {
		minimumProtocol += 30 * 24 * time.Hour
	}
	if cfg.ProtocolStateRetention == 0 {
		cfg.ProtocolStateRetention = minimumProtocol
	}
	minimumKey := maxID + 5*time.Minute
	if minimumKey == 5*time.Minute {
		minimumKey += time.Hour
	}
	if cfg.SigningKeyRetention == 0 {
		cfg.SigningKeyRetention = minimumKey
	}
	if cfg.ProtocolStateRetention < minimumProtocol {
		return MaintenanceConfig{}, fmt.Errorf("protocol state retention %s is shorter than maximum refresh-token lifetime plus expired retention %s", cfg.ProtocolStateRetention, minimumProtocol)
	}
	if cfg.SigningKeyRetention < minimumKey {
		return MaintenanceConfig{}, fmt.Errorf("signing-key retention %s is shorter than maximum ID-token lifetime plus clock skew %s", cfg.SigningKeyRetention, minimumKey)
	}
	return cfg, nil
}
