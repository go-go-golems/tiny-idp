// Package fositeadapter contains the strict production-like OAuth/OIDC adapter.
// Fosite owns protocol request parsing, authorization-code persistence, PKCE
// validation, token exchange, refresh-token handling, and response writing. The
// surrounding package owns product behavior: discovery/JWKS, login lookup,
// scope granting policy, and UserInfo claim rendering.
package fositeadapter

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	fositememory "github.com/ory/fosite/storage"
	fositejwt "github.com/ory/fosite/token/jwt"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/oidcmeta"
	"github.com/manuel/tinyidp/internal/securitytrace"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/idpui"
)

var ProductionHandlerFactories = []string{
	"OAuth2AuthorizeExplicitFactory",
	"OAuth2PKCEFactory",
	"OAuth2RefreshTokenGrantFactory",
	"OpenIDConnectExplicitFactory",
	"OpenIDConnectRefreshFactory",
	"OAuth2TokenIntrospectionFactory",
}

type Options struct {
	Issuer          string
	Store           idpstore.Store
	SecretKey       []byte
	Mode            idpstore.Mode
	CodeTTL         time.Duration
	AccessTokenTTL  time.Duration
	IDTokenTTL      time.Duration
	RefreshTokenTTL time.Duration
	SessionTTL      time.Duration
	InteractionTTL  time.Duration
	Clock           func() time.Time
	// ClientSecrets optionally supplies plaintext client secrets for callers that
	// are converting legacy/dev config into Fosite's BCrypt client store. The
	// production embedded API should prefer BCrypt hashes in idpstore.Client.SecretHash.
	ClientSecrets     map[string]string
	CookieSecure      bool
	CookieSameSite    http.SameSite
	SessionCookieName string
	CSRFCookieName    string
	CookiePath        string
	AccountChooser    AccountChooserConfig
	Audit             idp.Sink
	Consent           idp.ConsentPolicy
	RateLimiter       idp.RateLimiter
	ClientAddress     idp.ClientAddressResolver
	Authenticator     idp.PasswordAuthenticator
	PasswordPolicy    idp.PasswordAcceptancePolicy
	PasswordWork      idp.PasswordWorkConfig
	// AuthorizePersistenceHook is a test-only failpoint hook for the durable
	// authorization response lifecycle. Production callers should leave it nil.
	AuthorizePersistenceHook func(point string) error
	// TokenPersistenceHook is a test-only failpoint hook for authorization-code
	// redemption and refresh-token rotation storage lifecycles.
	TokenPersistenceHook func(point string) error
	SecurityEvents       securitytrace.Sink
	InteractionRenderer  idpui.InteractionRenderer
}

type Provider struct {
	issuer            oidcmeta.Issuer
	store             idpstore.Store
	fositeStore       *fositememory.MemoryStore
	sqlStore          *sqlFositeStore
	oauth2            fosite.OAuth2Provider
	config            *fosite.Config
	mode              idpstore.Mode
	csrfKey           []byte
	cookieSecure      bool
	cookieSameSite    http.SameSite
	sessionCookieName string
	csrfCookieName    string
	chooser           AccountChooserConfig
	cookiePathValue   string
	audit             idp.Sink
	securityEvents    securitytrace.Sink
	consent           idp.ConsentPolicy
	rateLimiter       idp.RateLimiter
	clientAddress     idp.ClientAddressResolver
	authenticator     idp.PasswordAuthenticator
	auditFailures     atomic.Uint64
	securityFailures  atomic.Uint64
	sessionTTL        time.Duration
	interactionTTL    time.Duration
	clock             func() time.Time
	interactionUI     idpui.InteractionRenderer
	renderMetrics     interactionRenderMetrics
}

func (p *Provider) PasswordWorkStats() (idp.PasswordWorkStats, bool) {
	reporter, ok := p.authenticator.(idp.PasswordWorkReporter)
	if !ok {
		return idp.PasswordWorkStats{}, false
	}
	return reporter.PasswordWorkStats(), true
}

// InteractionRenderStats reports process-local renderer health without
// exposing interaction, user, client, or error text as metric labels.
func (p *Provider) InteractionRenderStats() idpui.RenderStats {
	if p == nil {
		return idpui.RenderStats{}
	}
	return p.renderMetrics.snapshot()
}

func (p *Provider) AuditDeliveryFailures() uint64 { return p.auditFailures.Load() }

func (p *Provider) SecurityEventDeliveryFailures() uint64 { return p.securityFailures.Load() }

func (p *Provider) recordAudit(ctx context.Context, event idp.Event) {
	if err := p.audit.Emit(ctx, event); err != nil {
		p.auditFailures.Add(1)
	}
}

func (p *Provider) recordSecurity(ctx context.Context, event securitytrace.Event) {
	if event.Version == 0 {
		event.Version = securitytrace.SchemaVersion
	}
	if event.Time.IsZero() {
		event.Time = p.now()
	}
	if err := p.securityEvents.EmitSecurity(ctx, event); err != nil {
		p.securityFailures.Add(1)
	}
}

// tinyidp:development-default -- production mode rejects missing security controls upstream.
func NewProvider(ctx context.Context, opts Options) (*Provider, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	iss, err := oidcmeta.ParseIssuer(opts.Issuer)
	if err != nil {
		return nil, err
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if opts.Mode == "" {
		opts.Mode = idpstore.DevMode
	}
	if opts.Mode == idpstore.ProductionMode && len(opts.SecretKey) < 32 {
		return nil, fmt.Errorf("production mode requires a token secret key of at least 32 bytes")
	}
	if len(opts.SecretKey) == 0 {
		opts.SecretKey = []byte("tinyidp-dev-secret-key-at-least-32-bytes")
	}
	if opts.Audit == nil {
		opts.Audit = idp.NoopSink{}
	}
	if opts.SecurityEvents == nil {
		opts.SecurityEvents = securitytrace.NoopSink{}
	}
	if opts.InteractionRenderer == nil {
		renderer, rendererErr := idpui.NewDefaultRenderer()
		if rendererErr != nil {
			return nil, fmt.Errorf("build default interaction renderer: %w", rendererErr)
		}
		opts.InteractionRenderer = renderer
	}
	if opts.Consent == nil {
		if opts.Mode == idpstore.ProductionMode {
			opts.Consent = NewStoredConsent(opts.Store, 0)
		} else {
			opts.Consent = AlwaysSkipConsent{}
		}
	}
	if opts.RateLimiter == nil {
		opts.RateLimiter = AllowAllRateLimiter{}
	}
	if opts.ClientAddress == nil {
		opts.ClientAddress = idp.DirectClientAddressResolver{}
	}
	if opts.Authenticator == nil {
		policy := idpaccounts.DefaultLoginPolicy()
		if opts.Mode != idpstore.ProductionMode {
			policy.AllowPasswordless = true
			policy.LockoutThreshold = 0
			if opts.PasswordPolicy.MinCharacters == 0 {
				opts.PasswordPolicy = idp.DevelopmentPasswordAcceptancePolicy()
			}
		}
		authenticator, err := idpaccounts.NewService(opts.Store, idpaccounts.Options{Audit: opts.Audit, LoginPolicy: policy, PasswordPolicy: opts.PasswordPolicy, PasswordWork: opts.PasswordWork})
		if err != nil {
			return nil, fmt.Errorf("build password authenticator: %w", err)
		}
		opts.Authenticator = authenticator
	}
	if opts.CodeTTL == 0 {
		opts.CodeTTL = 5 * time.Minute
	}
	if opts.AccessTokenTTL == 0 {
		opts.AccessTokenTTL = time.Hour
	}
	if opts.IDTokenTTL == 0 {
		opts.IDTokenTTL = time.Hour
	}
	if opts.RefreshTokenTTL == 0 {
		opts.RefreshTokenTTL = 24 * time.Hour
	}
	if opts.SessionTTL == 0 {
		opts.SessionTTL = 24 * time.Hour
	}
	if opts.InteractionTTL == 0 {
		opts.InteractionTTL = 10 * time.Minute
	}
	if opts.InteractionTTL < 0 {
		return nil, fmt.Errorf("interaction TTL must be positive")
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.CookieSameSite == 0 {
		opts.CookieSameSite = http.SameSiteLaxMode
	}
	if opts.SessionCookieName == "" {
		opts.SessionCookieName = defaultSessionCookieName
	}
	if opts.CSRFCookieName == "" {
		opts.CSRFCookieName = defaultCSRFCookieName
	}
	if err := opts.AccountChooser.normalize(); err != nil {
		return nil, err
	}
	if !validCookieName(opts.SessionCookieName) || !validCookieName(opts.CSRFCookieName) || opts.SessionCookieName == opts.CSRFCookieName || (opts.AccountChooser.Enabled && (opts.AccountChooser.ContextCookieName == opts.SessionCookieName || opts.AccountChooser.ContextCookieName == opts.CSRFCookieName)) {
		return nil, fmt.Errorf("fositeadapter: session and csrf cookie names must be distinct valid cookie names")
	}
	if opts.CookiePath != "" && (!strings.HasPrefix(opts.CookiePath, "/") || strings.ContainsAny(opts.CookiePath, "\x00\r\n;")) {
		return nil, fmt.Errorf("fositeadapter: cookie path must be an absolute HTTP path")
	}

	sendDebug := opts.Mode != idpstore.ProductionMode
	cfg := &fosite.Config{
		GlobalSecret:                   opts.SecretKey,
		SendDebugMessagesToClients:     sendDebug,
		AccessTokenLifespan:            opts.AccessTokenTTL,
		RefreshTokenLifespan:           opts.RefreshTokenTTL,
		AuthorizeCodeLifespan:          opts.CodeTTL,
		IDTokenLifespan:                opts.IDTokenTTL,
		IDTokenIssuer:                  iss.String(),
		EnforcePKCE:                    opts.Mode == idpstore.ProductionMode,
		EnforcePKCEForPublicClients:    true,
		EnablePKCEPlainChallengeMethod: false,
		ScopeStrategy:                  fosite.ExactScopeStrategy,
		RefreshTokenScopes:             []string{"offline_access"},
		MinParameterEntropy:            8,
		RedirectSecureChecker: func(_ context.Context, u *url.URL) bool {
			return u.Scheme == "https" || u.Hostname() == "localhost" || strings.HasPrefix(u.Hostname(), "127.")
		},
	}

	fs, err := buildFositeStore(ctx, opts.Store, cfg, opts.ClientSecrets, opts.AuthorizePersistenceHook, opts.TokenPersistenceHook)
	if err != nil {
		return nil, err
	}
	p := &Provider{issuer: iss, store: opts.Store, fositeStore: fs.memoryStore, sqlStore: fs.sqlStore, config: cfg, mode: opts.Mode, csrfKey: opts.SecretKey, cookieSecure: opts.CookieSecure, cookieSameSite: opts.CookieSameSite, sessionCookieName: opts.SessionCookieName, csrfCookieName: opts.CSRFCookieName, chooser: opts.AccountChooser, cookiePathValue: opts.CookiePath, audit: opts.Audit, securityEvents: opts.SecurityEvents, consent: opts.Consent, rateLimiter: opts.RateLimiter, clientAddress: opts.ClientAddress, authenticator: opts.Authenticator, sessionTTL: opts.SessionTTL, interactionTTL: opts.InteractionTTL, clock: opts.Clock, interactionUI: opts.InteractionRenderer}

	core := compose.NewOAuth2HMACStrategy(cfg)
	oidc := compose.NewOpenIDConnectStrategy(p.activePrivateKey, cfg)
	strategy := compose.CommonStrategy{CoreStrategy: core, OpenIDConnectTokenStrategy: oidc, Signer: oidc.Signer}
	p.oauth2 = compose.Compose(
		cfg,
		fs.store,
		strategy,
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2PKCEFactory,
		compose.OAuth2RefreshTokenGrantFactory,
		compose.OpenIDConnectExplicitFactory,
		compose.OpenIDConnectRefreshFactory,
		compose.OAuth2TokenIntrospectionFactory,
	)
	return p, nil
}

func validCookieName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "\x00\r\n\t ;,=")
}

func (p *Provider) now() time.Time { return p.clock().UTC() }

type composedFositeStore struct {
	store       interface{}
	memoryStore *fositememory.MemoryStore
	sqlStore    *sqlFositeStore
}

func buildFositeStore(ctx context.Context, st idpstore.Store, cfg *fosite.Config, plainSecrets map[string]string, authorizeHook, tokenHook func(string) error) (*composedFositeStore, error) {
	if sqlProvider, ok := st.(sqlDBProvider); ok {
		s, err := newSQLFositeStore(sqlProvider.SQLDB(), st, cfg, plainSecrets, authorizeHook, tokenHook)
		if err != nil {
			return nil, err
		}
		return &composedFositeStore{store: s, sqlStore: s}, nil
	}
	fs := fositememory.NewMemoryStore()
	clients, err := st.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	hasher := &fosite.BCrypt{Config: cfg}
	for _, c := range clients {
		if c.Disabled {
			continue
		}
		fc := &fosite.DefaultClient{
			ID:            c.ID,
			Public:        c.Public,
			RedirectURIs:  append([]string(nil), c.RedirectURIs...),
			ResponseTypes: []string{"code"},
			GrantTypes:    []string{"authorization_code", "refresh_token"},
			Scopes:        append([]string(nil), c.AllowedScopes...),
		}
		if !c.Public {
			if secret, ok := plainSecrets[c.ID]; ok {
				hashed, err := hasher.Hash(ctx, []byte(secret))
				if err != nil {
					return nil, err
				}
				fc.Secret = hashed
			} else if len(c.SecretHash) > 0 && strings.HasPrefix(string(c.SecretHash), "$2") {
				fc.Secret = append([]byte(nil), c.SecretHash...)
			}
		}
		fs.Clients[c.ID] = clientWithLifespans(fc, c)
	}
	return &composedFositeStore{store: fs, memoryStore: fs}, nil
}

func (p *Provider) activePrivateKey(ctx context.Context) (interface{}, error) {
	key, err := p.store.ActiveSigningKey(ctx)
	if err != nil {
		return nil, err
	}
	priv, err := keys.ParseRSAPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return (*rsa.PrivateKey)(priv), nil
}

func (p *Provider) Handler() http.Handler {
	mux := http.NewServeMux()
	prefix := strings.TrimRight(p.issuer.URL.EscapedPath(), "/")
	p.registerAt(mux, prefix)
	return p.securityHeaders(mux)
}

func (p *Provider) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

func (p *Provider) registerAt(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/.well-known/openid-configuration", p.discovery)
	mux.HandleFunc(prefix+"/jwks", p.jwks)
	mux.HandleFunc(prefix+"/authorize", p.authorize)
	mux.HandleFunc(prefix+"/token", p.token)
	mux.HandleFunc(prefix+"/userinfo", p.userinfo)
	mux.HandleFunc(prefix+"/end-session", p.endSession)
	mux.HandleFunc(prefix+"/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc(prefix+"/readyz", p.readyz)
}

func (p *Provider) discovery(w http.ResponseWriter, _ *http.Request) {
	d, err := oidcmeta.ProductionDiscovery(p.issuer.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (p *Provider) jwks(w http.ResponseWriter, r *http.Request) {
	verificationKeys, err := p.store.VerificationKeys(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jwks, err := keys.PublicJWKS(verificationKeys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, jwks)
}

func (p *Provider) readyz(w http.ResponseWriter, r *http.Request) {
	if _, err := p.store.ActiveSigningKey(r.Context()); err != nil {
		http.Error(w, "active signing key missing", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ready\n"))
}

func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p.beginAuthorize(w, r)
	case http.MethodPost:
		p.resumeAuthorize(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Provider) beginAuthorize(w http.ResponseWriter, r *http.Request) {
	if p.rejectUnsupportedRequestObject(w, r) {
		return
	}
	ar, err := p.oauth2.NewAuthorizeRequest(fosite.NewContext(), r)
	if err != nil {
		p.emit(r.Context(), idp.New("authorize.request.rejected"), ar, "rejected", auditReason(err))
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	maxAge, hasMaxAge, err := parseMaxAge(ar.GetRequestForm().Get("max_age"))
	if err != nil {
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrInvalidRequest.WithHint("max_age must be a non-negative decimal integer"))
		return
	}
	u, sess, sessionState, sessionErr := p.readBrowserSession(r)
	if sessionErr != nil {
		http.Error(w, "browser session storage unavailable", http.StatusServiceUnavailable)
		return
	}
	hasSession := sessionState == browserSessionActive
	actions := idpstore.InteractionRequiredAction(0)
	needLogin := !hasSession
	if needLogin {
		actions |= idpstore.InteractionRequireLogin
	} else if promptHas(ar.GetRequestForm().Get("prompt"), "login") || !sessionSatisfiesMaxAge(sess.AuthTime, p.now(), maxAge, hasMaxAge) {
		needLogin = true
		actions |= idpstore.InteractionRequireFreshLogin
	}
	client, clientErr := p.store.GetClient(r.Context(), ar.GetClient().GetID())
	if clientErr != nil || client.Disabled {
		http.Error(w, "unknown or disabled client", http.StatusBadRequest)
		return
	}
	selectAccount := p.chooser.Enabled && promptHas(ar.GetRequestForm().Get("prompt"), "select_account") && !needLogin
	var chooserEntries []idpstore.RememberedBrowserSession
	if selectAccount {
		contextHash := p.browserContextHash(r)
		if len(contextHash) != 0 {
			chooserEntries, err = p.store.ListRememberedBrowserSessions(r.Context(), contextHash, p.now())
			if err != nil && !errors.Is(err, idpstore.ErrNotFound) {
				http.Error(w, "browser context storage unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		if len(chooserEntries) == 0 {
			needLogin = true
			actions |= idpstore.InteractionRequireLogin
		} else {
			actions |= idpstore.InteractionRequireAccountSelection
		}
	}
	requireConsent := false
	if hasSession && !needLogin && !actions.Has(idpstore.InteractionRequireAccountSelection) {
		requireConsent, err = p.consent.RequireConsent(r.Context(), u, client, []string(ar.GetRequestedScopes()))
		if err != nil {
			http.Error(w, "consent policy failed", http.StatusInternalServerError)
			return
		}
		if requireConsent {
			actions |= idpstore.InteractionRequireConsent
		}
	}
	if promptHas(ar.GetRequestForm().Get("prompt"), "none") {
		if actions.Has(idpstore.InteractionRequireAccountSelection) || (selectAccount && needLogin) {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, accountSelectionRequiredError())
			return
		}
		if needLogin {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrLoginRequired)
			return
		}
		if requireConsent {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrConsentRequired)
			return
		}
	}
	if !needLogin && !requireConsent && !actions.Has(idpstore.InteractionRequireAccountSelection) {
		p.finishAuthorize(w, r, ar, u, sess.AuthTime, false, nil)
		return
	}
	handle, csrfToken, err := p.createInteraction(w, r, ar, actions)
	if err != nil {
		http.Error(w, "create authorization interaction failed", http.StatusInternalServerError)
		return
	}
	page := p.newInteractionPage(handle, csrfToken, actions, ar.GetRequestForm(), !actions.Has(idpstore.InteractionRequireAccountSelection), client.ID, []string(ar.GetRequestedScopes()), strings.TrimSpace(ar.GetRequestForm().Get("login_hint")), nil)
	if actions.Has(idpstore.InteractionRequireAccountSelection) {
		page.DocumentTitle = "Choose an account"
		page.AccountChooser = chooserPrompt(chooserEntries)
	}
	p.renderInteraction(w, r, http.StatusOK, page)
}

func accountSelectionRequiredError() *fosite.RFC6749Error {
	return &fosite.RFC6749Error{ErrorField: "account_selection_required", DescriptionField: "The Authorization Server requires End-User account selection.", CodeField: http.StatusBadRequest}
}

func chooserPrompt(entries []idpstore.RememberedBrowserSession) *idpui.AccountChooserPrompt {
	prompt := &idpui.AccountChooserPrompt{AccountField: idpui.AccountFieldName, Entries: make([]idpui.AccountChooserEntry, 0, len(entries))}
	for _, entry := range entries {
		prompt.Entries = append(prompt.Entries, idpui.AccountChooserEntry{Value: base64.RawURLEncoding.EncodeToString(entry.IDHash), Label: entry.DisplayLabel})
	}
	return prompt
}

func (p *Provider) resumeAuthorize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	clientAddress, err := p.clientAddress.ResolveClientAddress(r)
	if err != nil {
		http.Error(w, "resolve client address failed", http.StatusInternalServerError)
		return
	}
	if !p.rateLimiter.Allow(r.Context(), "authorize:"+clientAddress) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.rate_limited", Result: "rejected", Reason: "rate_limited"})
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}
	handle := r.PostForm.Get(interactionFieldName)
	if !p.validateCSRF(r, handle) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.csrf_rejected", Result: "rejected", Reason: "invalid_csrf"})
		http.Error(w, "invalid csrf token", http.StatusBadRequest)
		return
	}
	record, err := p.store.GetInteraction(r.Context(), idpstore.HashSecret(p.csrfKey, handle))
	if err != nil || record.ConsumedAt != nil || !p.now().Before(record.ExpiresAt) {
		http.Error(w, "authorization interaction is invalid or expired", http.StatusBadRequest)
		return
	}
	if !equalBytes(record.BrowserBindingHash, p.browserBindingHash(r)) {
		http.Error(w, "authorization interaction browser mismatch", http.StatusBadRequest)
		return
	}
	if len(record.SessionIDHash) != 0 && !equalBytes(record.SessionIDHash, p.browserSessionHash(r)) {
		http.Error(w, "authorization interaction session mismatch", http.StatusBadRequest)
		return
	}
	if len(record.BrowserContextHash) != 0 && !equalBytes(record.BrowserContextHash, p.browserContextHash(r)) {
		http.Error(w, "authorization interaction browser context mismatch", http.StatusBadRequest)
		return
	}
	ar, err := p.reconstructAuthorizeRequest(r, record)
	if err != nil {
		p.emit(r.Context(), idp.New("authorize.request.rejected"), ar, "rejected", auditReason(err))
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	client, err := p.store.GetClient(r.Context(), record.ClientID)
	if err != nil || client.Disabled || !equalBytes(record.GenerationHash, clientGenerationHash(client)) || !clientAllowsScopeAndRedirect(client, url.Values(record.CanonicalRequest).Get("scope"), record.RedirectURI) {
		http.Error(w, "authorization client changed or is disabled", http.StatusBadRequest)
		return
	}
	if _, err := p.store.ActiveSigningKey(r.Context()); err != nil {
		http.Error(w, "signing key unavailable", http.StatusServiceUnavailable)
		return
	}
	action := idpui.Action(r.PostForm.Get(idpui.ActionFieldName))
	selectionRequired := record.RequiredActions.Has(idpstore.InteractionRequireAccountSelection)
	if (selectionRequired && action != idpui.ActionContinue && action != idpui.ActionDeny) || (!selectionRequired && action != idpui.ActionApprove && action != idpui.ActionDeny) {
		http.Error(w, "invalid interaction action", http.StatusBadRequest)
		return
	}
	if action == idpui.ActionDeny {
		if _, err := p.store.ConsumeInteraction(r.Context(), record.IDHash, p.now(), idpstore.InteractionOutcomeDenied); err != nil {
			http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
			return
		}
		traceID := interactionTraceID(record)
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ConsentDenied, InteractionID: traceID, ClientID: record.ClientID})
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: traceID, ClientID: record.ClientID, Outcome: string(idpstore.InteractionOutcomeDenied)})
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrAccessDenied)
		return
	}
	u, sess, sessionState, sessionErr := p.readBrowserSession(r)
	if sessionErr != nil {
		http.Error(w, "browser session storage unavailable", http.StatusServiceUnavailable)
		return
	}
	hasSession := sessionState == browserSessionActive
	authTime := sess.AuthTime
	if selectionRequired {
		entryHash, decodeErr := base64.RawURLEncoding.DecodeString(r.PostForm.Get(idpui.AccountFieldName))
		if decodeErr != nil || len(entryHash) == 0 {
			http.Error(w, "invalid account selection", http.StatusBadRequest)
			return
		}
		newHandle, randomErr := randomB64(32)
		if randomErr != nil {
			http.Error(w, "create browser session failed", http.StatusInternalServerError)
			return
		}
		selectedSession, selectedUser, activateErr := p.store.ActivateRememberedSession(r.Context(), p.browserContextHash(r), entryHash, idpstore.HashSecret(p.csrfKey, newHandle), p.now())
		if activateErr != nil {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account_selection.rejected", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "invalid_selection"})
			http.Error(w, "invalid account selection", http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: p.sessionCookieName, Value: newHandle, Path: p.cookiePath(), HttpOnly: true, Secure: p.cookieSecure, SameSite: p.cookieSameSite, MaxAge: int(p.sessionTTL.Seconds())})
		u, sess, hasSession, authTime = selectedUser, selectedSession, true, selectedSession.AuthTime
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "account_selection.success", ClientID: ar.GetClient().GetID(), Subject: u.Sub, Result: "accepted"})
	}
	requiresLogin := record.RequiredActions.Has(idpstore.InteractionRequireLogin) || record.RequiredActions.Has(idpstore.InteractionRequireFreshLogin)
	login := strings.ToLower(strings.TrimSpace(r.PostForm.Get("login")))
	if requiresLogin && login == "" {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.failure", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "missing_login"})
		page := p.newInteractionPage(handle, r.PostForm.Get(idpui.CSRFFieldName), record.RequiredActions, url.Values(record.CanonicalRequest), true, client.ID, []string(ar.GetRequestedScopes()), "", &idpui.PublicError{Code: idpui.ErrorMissingLogin, Field: idpui.FieldCredentials, Summary: "Enter your username and password."})
		p.renderInteraction(w, r, http.StatusBadRequest, page)
		return
	}
	if login != "" {
		if !p.allowLogin(r.Context(), ar.GetClient().GetID(), clientAddress, login) {
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.rate_limited", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "rate_limited"})
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		result, authErr := p.authenticator.AuthenticatePassword(r.Context(), login, r.PostForm.Get("password"), idp.LoginMetadata{RemoteAddr: clientAddress, UserAgent: r.UserAgent(), ClientID: ar.GetClient().GetID()})
		if authErr != nil {
			if errors.Is(authErr, idpaccounts.ErrAuthenticationUnavailable) || errors.Is(authErr, idpaccounts.ErrPasswordWorkRejected) {
				p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.unavailable", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "authentication_unavailable"})
				http.Error(w, "authentication temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.failure", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: idpaccounts.AuditReason(authErr)})
			page := p.newInteractionPage(handle, r.PostForm.Get(idpui.CSRFFieldName), record.RequiredActions, url.Values(record.CanonicalRequest), true, client.ID, []string(ar.GetRequestedScopes()), login, &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: "Invalid login or password."})
			p.renderInteraction(w, r, http.StatusUnauthorized, page)
			return
		}
		u = result.User
		authTime = p.now()
		hasSession = true
		if err := p.createBrowserSession(w, r, u, authTime); err != nil {
			http.Error(w, "create session failed", http.StatusInternalServerError)
			return
		}
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "login.success", ClientID: ar.GetClient().GetID(), Subject: u.Sub, Result: "accepted"})
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.AuthenticationSatisfied, InteractionID: interactionTraceID(record), ClientID: record.ClientID})
	}
	if !hasSession {
		http.Error(w, "login is required", http.StatusBadRequest)
		return
	}
	u, err = p.store.GetUser(r.Context(), u.ID)
	if err != nil || u.Disabled {
		http.Error(w, "user is unavailable", http.StatusForbidden)
		return
	}
	requireConsent, err := p.consent.RequireConsent(r.Context(), u, client, []string(ar.GetRequestedScopes()))
	if err != nil {
		http.Error(w, "consent policy failed", http.StatusInternalServerError)
		return
	}
	approved := action == idpui.ActionApprove
	if requireConsent && !approved {
		page := p.newInteractionPage(handle, r.PostForm.Get(idpui.CSRFFieldName), record.RequiredActions, url.Values(record.CanonicalRequest), true, client.ID, []string(ar.GetRequestedScopes()), login, &idpui.PublicError{Code: idpui.ErrorConsentRequired, Field: idpui.FieldConsent, Summary: "Approve or deny the requested access."})
		p.renderInteraction(w, r, http.StatusBadRequest, page)
		return
	}
	if requireConsent && approved {
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.ConsentApproved, InteractionID: interactionTraceID(record), ClientID: record.ClientID})
	}
	if p.sqlStore == nil {
		if _, err := p.store.ConsumeInteraction(r.Context(), record.IDHash, p.now(), idpstore.InteractionOutcomeApproved); err != nil {
			http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
			return
		}
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: interactionTraceID(record), ClientID: record.ClientID, Outcome: string(idpstore.InteractionOutcomeApproved)})
	}
	p.finishAuthorize(w, r, ar, u, authTime, approved, &record)
}

func clientAllowsScopeAndRedirect(client idpstore.Client, scope, redirectURI string) bool {
	redirectAllowed := false
	for _, candidate := range client.RedirectURIs {
		if candidate == redirectURI {
			redirectAllowed = true
			break
		}
	}
	if !redirectAllowed {
		return false
	}
	allowed := make(map[string]struct{}, len(client.AllowedScopes))
	for _, candidate := range client.AllowedScopes {
		allowed[candidate] = struct{}{}
	}
	for _, requested := range strings.Fields(scope) {
		if _, ok := allowed[requested]; !ok {
			return false
		}
	}
	return true
}

func (p *Provider) allowLogin(ctx context.Context, clientID, clientAddress, login string) bool {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(login))))
	account := hex.EncodeToString(sum[:])
	keys := []string{
		"login:account:" + account,
		"login:client:" + clientID,
		"login:address:" + clientAddress,
	}
	allowed := true
	for _, key := range keys {
		if !p.rateLimiter.Allow(ctx, key) {
			allowed = false
		}
	}
	return allowed
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if r.Method != http.MethodPost {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", Result: "rejected", Reason: "method_not_allowed"})
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", Result: "rejected", Reason: "invalid_form"})
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	clientAddress, err := p.clientAddress.ResolveClientAddress(r)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "resolve client address failed")
		return
	}
	claimedClientID := r.Form.Get("client_id")
	if basicClientID, _, ok := r.BasicAuth(); ok {
		claimedClientID = basicClientID
	}
	if !p.allowTokenPreAuthentication(r.Context(), clientAddress) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", ClientID: claimedClientID, Result: "rejected", Reason: "rate_limited"})
		tokenError(w, http.StatusTooManyRequests, "temporarily_unavailable", "rate limited")
		return
	}
	accessRequest, err := p.oauth2.NewAccessRequest(fosite.NewContext(), r, openid.NewDefaultSession())
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", ClientID: claimedClientID, Result: "rejected", Reason: auditReason(err)})
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
	authenticatedClientID := accessRequest.GetClient().GetID()
	if !p.rateLimiter.Allow(r.Context(), "token:client:"+authenticatedClientID) {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", ClientID: authenticatedClientID, Result: "rejected", Reason: "rate_limited"})
		tokenError(w, http.StatusTooManyRequests, "temporarily_unavailable", "rate limited")
		return
	}
	p.grantRequestedAccessScopes(accessRequest)
	response, err := p.oauth2.NewAccessResponse(fosite.NewContext(), accessRequest)
	if err != nil {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.rejected", ClientID: accessRequest.GetClient().GetID(), Subject: accessRequest.GetSession().GetSubject(), Result: "rejected", Reason: auditReason(err)})
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
	p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.TokenLifecycleDone, RequestID: accessRequest.GetID(), ClientID: accessRequest.GetClient().GetID(), GrantType: strings.Join(accessRequest.GetGrantTypes(), " ")})
	p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "token.request.accepted", ClientID: accessRequest.GetClient().GetID(), Subject: accessRequest.GetSession().GetSubject(), Result: "accepted", Fields: map[string]string{"grant_type": strings.Join(accessRequest.GetGrantTypes(), " ")}})
	p.oauth2.WriteAccessResponse(r.Context(), w, accessRequest, response)
}

func (p *Provider) allowTokenPreAuthentication(ctx context.Context, clientAddress string) bool {
	return p.rateLimiter.Allow(ctx, "token:address:"+clientAddress)
}

func (p *Provider) userinfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(r.Header.Values("Authorization")) > 1 || r.URL.Query().Has("access_token") {
		p.userinfoInvalidRequest(w, "duplicate Authorization headers and query bearer transport are forbidden")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			p.userinfoInvalidRequest(w, "invalid form body")
			return
		}
		if r.PostForm.Has("access_token") {
			p.userinfoInvalidRequest(w, "form bearer transport is forbidden")
			return
		}
	}
	authorization := r.Header.Get("Authorization")
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		p.userinfoUnauthorized(w, "missing or malformed bearer token")
		return
	}
	session := openid.NewDefaultSession()
	_, requester, err := p.oauth2.IntrospectToken(fosite.NewContext(), parts[1], fosite.AccessToken, session)
	if err != nil {
		p.userinfoUnauthorized(w, "invalid bearer token")
		return
	}
	claims := map[string]any{"sub": requester.GetSession().GetSubject()}
	if oidcSession, ok := requester.GetSession().(*openid.DefaultSession); ok && oidcSession.Claims != nil {
		for k, v := range oidcSession.Claims.Extra {
			claims[k] = v
		}
	}
	writeJSON(w, http.StatusOK, claims)
}

func (p *Provider) userinfoUnauthorized(w http.ResponseWriter, description string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="tiny-idp", error="invalid_token"`)
	tokenError(w, http.StatusUnauthorized, "invalid_token", description)
}

func (p *Provider) userinfoInvalidRequest(w http.ResponseWriter, description string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="tiny-idp", error="invalid_request"`)
	tokenError(w, http.StatusBadRequest, "invalid_request", description)
}

func (p *Provider) grantRequestedScopes(ar fosite.AuthorizeRequester) {
	clientScopes := map[string]struct{}{}
	for _, s := range ar.GetClient().GetScopes() {
		clientScopes[s] = struct{}{}
	}
	for _, s := range ar.GetRequestedScopes() {
		if _, ok := clientScopes[s]; ok {
			ar.GrantScope(s)
		}
	}
}

func (p *Provider) grantRequestedAudience(ar fosite.AuthorizeRequester) {
	for _, a := range ar.GetRequestedAudience() {
		ar.GrantAudience(a)
	}
}

func (p *Provider) grantRequestedAccessScopes(ar fosite.AccessRequester) {
	clientScopes := map[string]struct{}{}
	for _, s := range ar.GetClient().GetScopes() {
		clientScopes[s] = struct{}{}
	}
	for _, s := range ar.GetRequestedScopes() {
		if _, ok := clientScopes[s]; ok {
			ar.GrantScope(s)
		}
	}
}

func (p *Provider) newOIDCSession(ctx context.Context, u idpstore.User, ar fosite.AuthorizeRequester, authTime time.Time) *openid.DefaultSession {
	now := p.now()
	claims := &fositejwt.IDTokenClaims{
		Issuer:   p.issuer.String(),
		Subject:  u.Sub,
		Audience: []string{ar.GetClient().GetID()},
		Nonce:    ar.GetRequestForm().Get("nonce"),
		IssuedAt: now,
		AuthTime: authTime.UTC(),
		Extra:    map[string]interface{}{},
	}
	prompt := ar.GetRequestForm().Get("prompt")
	if promptHas(prompt, "none") || promptHas(prompt, "login") || ar.GetRequestForm().Get("max_age") != "" {
		claims.RequestedAt = now
	}
	for k, v := range idpstore.ClaimsForScopes(u, []string(ar.GetGrantedScopes())) {
		if k != "sub" {
			claims.Extra[k] = v
		}
	}
	headers := fositejwt.NewHeaders()
	if key, err := p.store.ActiveSigningKey(ctx); err == nil && key.ID != "" {
		headers.Add("kid", key.ID)
	}
	return &openid.DefaultSession{Claims: claims, Headers: headers, Subject: u.Sub, Username: u.PreferredUsername, ExpiresAt: map[fosite.TokenType]time.Time{}}
}

func (p *Provider) rejectUnsupportedRequestObject(w http.ResponseWriter, r *http.Request) bool {
	requestObject := r.URL.Query().Get("request")
	if requestObject == "" {
		return false
	}
	claims := requestObjectClaims(requestObject)
	clientID := firstNonEmpty(r.URL.Query().Get("client_id"), stringClaim(claims, "client_id"))
	queryRedirectURI := r.URL.Query().Get("redirect_uri")
	if queryRedirectURI != "" && !p.clientAllowsRedirect(r.Context(), clientID, queryRedirectURI) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return true
	}
	redirectURI := firstNonEmpty(queryRedirectURI, stringClaim(claims, "redirect_uri"))
	if clientID == "" || redirectURI == "" || !p.clientAllowsRedirect(r.Context(), clientID, redirectURI) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"request_not_supported","error_description":"The OP does not support use of the request parameter."}`))
		return true
	}
	loc, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return true
	}
	q := loc.Query()
	q.Set("error", "request_not_supported")
	q.Set("error_description", "The OP does not support use of the request parameter.")
	if state := stringClaim(claims, "state"); state != "" {
		q.Set("state", state)
	}
	loc.RawQuery = q.Encode()
	http.Redirect(w, r, loc.String(), http.StatusFound)
	return true
}

func requestObjectClaims(requestObject string) map[string]any {
	parts := strings.Split(requestObject, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

func stringClaim(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	v, _ := claims[key].(string)
	return v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (p *Provider) clientAllowsRedirect(ctx context.Context, clientID, redirectURI string) bool {
	client, err := p.store.GetClient(ctx, clientID)
	if err != nil || client.Disabled {
		return false
	}
	for _, allowed := range client.RedirectURIs {
		if allowed == redirectURI {
			return true
		}
	}
	return false
}

func randomB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func tokenError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func (p *Provider) emit(ctx context.Context, e idp.Event, ar fosite.AuthorizeRequester, result, reason string) {
	e.Result = result
	e.Reason = cleanAuditReason(reason)
	if ar != nil {
		if ar.GetClient() != nil {
			e.ClientID = ar.GetClient().GetID()
		}
		if ar.GetID() != "" {
			e.RequestID = ar.GetID()
		}
	}
	p.recordAudit(ctx, e)
}

func (p *Provider) finishAuthorize(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, u idpstore.User, authTime time.Time, consentApproved bool, interaction *idpstore.InteractionRecord) {
	client, err := p.store.GetClient(r.Context(), ar.GetClient().GetID())
	if err != nil || client.Disabled {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}
	scopes := []string(ar.GetRequestedScopes())
	requireConsent, err := p.consent.RequireConsent(r.Context(), u, client, scopes)
	if err != nil {
		http.Error(w, "consent policy failed", http.StatusInternalServerError)
		return
	}
	if requireConsent && !consentApproved {
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "consent.required", ClientID: client.ID, Subject: u.Sub, Result: "rejected", Reason: "not_approved"})
		http.Error(w, "consent required", http.StatusForbidden)
		return
	}
	if requireConsent {
		if err := p.consent.RecordConsent(r.Context(), u, client, scopes); err != nil {
			http.Error(w, "record consent failed", http.StatusInternalServerError)
			return
		}
		p.recordAudit(r.Context(), idp.Event{Time: p.now(), Name: "consent.granted", ClientID: client.ID, Subject: u.Sub, Result: "accepted"})
	}
	p.grantRequestedScopes(ar)
	p.grantRequestedAudience(ar)
	session := p.newOIDCSession(r.Context(), u, ar, authTime)
	protocolContext := fosite.NewContext()
	var finishLifecycle func(bool) error
	if p.sqlStore != nil {
		protocolContext, finishLifecycle, err = p.sqlStore.beginAuthorizeLifecycle(protocolContext, interaction, p.now())
		if err != nil {
			http.Error(w, "authorization interaction already completed", http.StatusBadRequest)
			return
		}
	}
	response, err := p.oauth2.NewAuthorizeResponse(protocolContext, ar, session)
	if err != nil {
		if finishLifecycle != nil {
			_ = finishLifecycle(false)
		}
		p.emit(r.Context(), idp.New("authorize.request.rejected"), ar, "rejected", auditReason(err))
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	if finishLifecycle != nil {
		if err := finishLifecycle(true); err != nil {
			p.emit(r.Context(), idp.New("authorize.request.rejected"), ar, "rejected", auditReason(err))
			http.Error(w, "authorization persistence failed", http.StatusInternalServerError)
			return
		}
	}
	if interaction != nil {
		traceID := interactionTraceID(*interaction)
		if p.sqlStore != nil {
			p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.InteractionTerminal, InteractionID: traceID, ClientID: client.ID, Outcome: string(idpstore.InteractionOutcomeApproved)})
		}
		p.recordSecurity(r.Context(), securitytrace.Event{Kind: securitytrace.AuthorizationArtifactsDone, InteractionID: traceID, RequestID: ar.GetID(), ClientID: client.ID})
	}
	p.emit(r.Context(), idp.New("authorize.request.accepted"), ar, "accepted", "")
	responseWriter := w
	if r.Method == http.MethodPost {
		responseWriter = seeOtherRedirectWriter{ResponseWriter: w}
	}
	p.oauth2.WriteAuthorizeResponse(r.Context(), responseWriter, ar, response)
}
