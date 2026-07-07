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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	fositememory "github.com/ory/fosite/storage"
	fositejwt "github.com/ory/fosite/token/jwt"

	"github.com/manuel/tinyidp/internal/audit"
	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/oidcmeta"
	"github.com/manuel/tinyidp/internal/storage"
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
	Store           storage.Store
	SecretKey       []byte
	Mode            domain.Mode
	CodeTTL         time.Duration
	AccessTokenTTL  time.Duration
	IDTokenTTL      time.Duration
	RefreshTokenTTL time.Duration
	SessionTTL      time.Duration
	// ClientSecrets optionally supplies plaintext client secrets for callers that
	// are converting legacy/dev config into Fosite's BCrypt client store. The
	// production embedded API should prefer BCrypt hashes in domain.Client.SecretHash.
	ClientSecrets map[string]string
	CookieSecure  bool
	Audit         audit.Sink
	Consent       ConsentPolicy
	RateLimiter   RateLimiter
}

type Provider struct {
	issuer       oidcmeta.Issuer
	store        storage.Store
	fositeStore  *fositememory.MemoryStore
	oauth2       fosite.OAuth2Provider
	config       *fosite.Config
	mode         domain.Mode
	csrfKey      []byte
	cookieSecure bool
	audit        audit.Sink
	consent      ConsentPolicy
	rateLimiter  RateLimiter
	sessionTTL   time.Duration
}

func NewProvider(opts Options) (*Provider, error) {
	iss, err := oidcmeta.ParseIssuer(opts.Issuer)
	if err != nil {
		return nil, err
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if opts.Mode == "" {
		opts.Mode = domain.DevMode
	}
	if len(opts.SecretKey) == 0 {
		opts.SecretKey = []byte("tinyidp-dev-secret-key-at-least-32-bytes")
	}
	if opts.Audit == nil {
		opts.Audit = audit.NoopSink{}
	}
	if opts.Consent == nil {
		if opts.Mode == domain.ProductionMode {
			opts.Consent = NewStoredConsent(opts.Store, 0)
		} else {
			opts.Consent = AlwaysSkipConsent{}
		}
	}
	if opts.RateLimiter == nil {
		opts.RateLimiter = AllowAllRateLimiter{}
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

	sendDebug := opts.Mode != domain.ProductionMode
	cfg := &fosite.Config{
		GlobalSecret:                   opts.SecretKey,
		SendDebugMessagesToClients:     sendDebug,
		AccessTokenLifespan:            opts.AccessTokenTTL,
		RefreshTokenLifespan:           opts.RefreshTokenTTL,
		AuthorizeCodeLifespan:          opts.CodeTTL,
		IDTokenLifespan:                opts.IDTokenTTL,
		IDTokenIssuer:                  iss.String(),
		EnforcePKCE:                    true,
		EnforcePKCEForPublicClients:    true,
		EnablePKCEPlainChallengeMethod: false,
		ScopeStrategy:                  fosite.ExactScopeStrategy,
		RefreshTokenScopes:             []string{"offline_access"},
		MinParameterEntropy:            8,
		RedirectSecureChecker: func(_ context.Context, u *url.URL) bool {
			return u.Scheme == "https" || u.Hostname() == "localhost" || strings.HasPrefix(u.Hostname(), "127.")
		},
	}

	fs, err := buildFositeStore(opts.Store, cfg, opts.ClientSecrets)
	if err != nil {
		return nil, err
	}
	p := &Provider{issuer: iss, store: opts.Store, fositeStore: fs.memoryStore, config: cfg, mode: opts.Mode, csrfKey: opts.SecretKey, cookieSecure: opts.CookieSecure, audit: opts.Audit, consent: opts.Consent, rateLimiter: opts.RateLimiter, sessionTTL: opts.SessionTTL}

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

type composedFositeStore struct {
	store       interface{}
	memoryStore *fositememory.MemoryStore
}

func buildFositeStore(st storage.Store, cfg *fosite.Config, plainSecrets map[string]string) (*composedFositeStore, error) {
	if sqlProvider, ok := st.(sqlDBProvider); ok {
		s, err := newSQLFositeStore(sqlProvider.SQLDB(), st, cfg, plainSecrets)
		if err != nil {
			return nil, err
		}
		return &composedFositeStore{store: s}, nil
	}
	fs := fositememory.NewMemoryStore()
	clients, err := st.ListClients(context.Background())
	if err != nil {
		return nil, err
	}
	hasher := &fosite.BCrypt{Config: cfg}
	for _, c := range clients {
		fc := &fosite.DefaultClient{
			ID:            c.ID,
			Public:        c.Public,
			RedirectURIs:  append([]string(nil), c.RedirectURIs...),
			ResponseTypes: []string{"code"},
			GrantTypes:    []string{"authorization_code", "refresh_token"},
			Scopes:        append([]string(nil), c.AllowedScopes...),
		}
		if len(fc.Scopes) == 0 {
			fc.Scopes = []string{"openid", "profile", "email", "offline_access"}
		}
		if !c.Public {
			if secret, ok := plainSecrets[c.ID]; ok {
				hashed, err := hasher.Hash(context.Background(), []byte(secret))
				if err != nil {
					return nil, err
				}
				fc.Secret = hashed
			} else if len(c.SecretHash) > 0 && strings.HasPrefix(string(c.SecretHash), "$2") {
				fc.Secret = append([]byte(nil), c.SecretHash...)
			}
		}
		fs.Clients[c.ID] = fc
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
	p.registerAt(mux, "")
	if prefix != "" && prefix != "/" {
		p.registerAt(mux, prefix)
	}
	return p.securityHeaders(mux)
}

func (p *Provider) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'")
		next.ServeHTTP(w, r)
	})
}

func (p *Provider) registerAt(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/.well-known/openid-configuration", p.discovery)
	mux.HandleFunc(prefix+"/jwks", p.jwks)
	mux.HandleFunc(prefix+"/authorize", p.authorize)
	mux.HandleFunc(prefix+"/token", p.token)
	mux.HandleFunc(prefix+"/userinfo", p.userinfo)
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
		ar, err := p.oauth2.NewAuthorizeRequest(fosite.NewContext(), r)
		if err != nil {
			p.emit(r.Context(), audit.New("authorize.request.rejected"), ar, "rejected", err.Error())
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
			return
		}
		u, sess, hasSession := p.readBrowserSession(r)
		if hasSession && !promptHas(ar.GetRequestForm().Get("prompt"), "login") {
			client, _ := p.store.GetClient(r.Context(), ar.GetClient().GetID())
			requireConsent, err := p.consent.RequireConsent(r.Context(), u, client, []string(ar.GetRequestedScopes()))
			if err != nil {
				http.Error(w, "consent policy failed", http.StatusInternalServerError)
				return
			}
			if !requireConsent {
				p.finishAuthorize(w, r, ar, u, sess.AuthTime, false)
				return
			}
			p.renderInteraction(w, ar, false, true)
			return
		}
		if promptHas(ar.GetRequestForm().Get("prompt"), "none") {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrLoginRequired)
			return
		}
		p.renderInteraction(w, ar, true, true)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if !p.rateLimiter.Allow(r.Context(), "authorize:"+r.RemoteAddr) {
			_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "login.rate_limited", Result: "rejected", Reason: "rate_limited"})
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		if !p.validateCSRF(r) {
			_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "login.csrf_rejected", Result: "rejected", Reason: "invalid_csrf"})
			http.Error(w, "invalid csrf token", http.StatusBadRequest)
			return
		}
		ar, err := p.oauth2.NewAuthorizeRequest(fosite.NewContext(), r)
		if err != nil {
			p.emit(r.Context(), audit.New("authorize.request.rejected"), ar, "rejected", err.Error())
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
			return
		}
		u, sess, hasSession := p.readBrowserSession(r)
		authTime := sess.AuthTime
		login := strings.ToLower(strings.TrimSpace(r.PostForm.Get("login")))
		if login != "" {
			u, err = p.store.GetUserByLogin(r.Context(), login)
			if err != nil {
				_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "login.failure", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "invalid_login"})
				http.Error(w, "invalid login", http.StatusUnauthorized)
				return
			}
			authTime = time.Now().UTC()
			if err := p.createBrowserSession(w, r, u, authTime); err != nil {
				http.Error(w, "create session failed", http.StatusInternalServerError)
				return
			}
			_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "login.success", ClientID: ar.GetClient().GetID(), Subject: u.Sub, Result: "accepted"})
		} else if !hasSession {
			_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "login.failure", ClientID: ar.GetClient().GetID(), Result: "rejected", Reason: "missing_login"})
			http.Error(w, "login is required", http.StatusBadRequest)
			return
		}
		p.finishAuthorize(w, r, ar, u, authTime, r.PostForm.Get("consent_approved") == "true")
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if r.Method != http.MethodPost {
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.rejected", Result: "rejected", Reason: "method_not_allowed"})
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.rejected", Result: "rejected", Reason: "invalid_form"})
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	if !p.rateLimiter.Allow(r.Context(), "token:"+r.Form.Get("client_id")+":"+r.RemoteAddr) {
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.rejected", ClientID: r.Form.Get("client_id"), Result: "rejected", Reason: "rate_limited"})
		tokenError(w, http.StatusTooManyRequests, "temporarily_unavailable", "rate limited")
		return
	}
	accessRequest, err := p.oauth2.NewAccessRequest(fosite.NewContext(), r, openid.NewDefaultSession())
	if err != nil {
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.rejected", ClientID: r.Form.Get("client_id"), Result: "rejected", Reason: err.Error()})
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
	p.grantRequestedAccessScopes(accessRequest)
	response, err := p.oauth2.NewAccessResponse(fosite.NewContext(), accessRequest)
	if err != nil {
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.rejected", ClientID: accessRequest.GetClient().GetID(), Subject: accessRequest.GetSession().GetSubject(), Result: "rejected", Reason: err.Error()})
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
	_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "token.request.accepted", ClientID: accessRequest.GetClient().GetID(), Subject: accessRequest.GetSession().GetSubject(), Result: "accepted", Fields: map[string]string{"grant_type": strings.Join(accessRequest.GetGrantTypes(), " ")}})
	p.oauth2.WriteAccessResponse(r.Context(), w, accessRequest, response)
}

func (p *Provider) userinfo(w http.ResponseWriter, r *http.Request) {
	session := openid.NewDefaultSession()
	_, requester, err := p.oauth2.IntrospectToken(fosite.NewContext(), fosite.AccessTokenFromRequest(r), fosite.AccessToken, session)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
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

func (p *Provider) newOIDCSession(u domain.User, ar fosite.AuthorizeRequester, authTime time.Time) *openid.DefaultSession {
	now := time.Now().UTC()
	claims := &fositejwt.IDTokenClaims{
		Issuer:      p.issuer.String(),
		Subject:     u.Sub,
		Audience:    []string{ar.GetClient().GetID()},
		Nonce:       ar.GetRequestForm().Get("nonce"),
		IssuedAt:    now,
		RequestedAt: ar.GetRequestedAt(),
		AuthTime:    authTime.UTC(),
		Extra:       map[string]interface{}{},
	}
	for k, v := range domain.ClaimsForScopes(u, []string{"profile", "email"}) {
		if k != "sub" {
			claims.Extra[k] = v
		}
	}
	return &openid.DefaultSession{Claims: claims, Headers: &fositejwt.Headers{}, Subject: u.Sub, Username: u.PreferredUsername, ExpiresAt: map[fosite.TokenType]time.Time{}}
}

func hidden(ar fosite.AuthorizeRequester) string {
	fields := map[string]string{
		"response_type":         strings.Join(ar.GetResponseTypes(), " "),
		"client_id":             ar.GetClient().GetID(),
		"redirect_uri":          ar.GetRedirectURI().String(),
		"scope":                 strings.Join(ar.GetRequestedScopes(), " "),
		"state":                 ar.GetState(),
		"nonce":                 ar.GetRequestForm().Get("nonce"),
		"code_challenge":        ar.GetRequestForm().Get("code_challenge"),
		"code_challenge_method": ar.GetRequestForm().Get("code_challenge_method"),
	}
	var b strings.Builder
	for k, v := range fields {
		_, _ = fmt.Fprintf(&b, `<input type="hidden" name="%s" value="%s">`, k, htmlEscape(v))
	}
	return b.String()
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "\"", "&quot;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func randomB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func tokenError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}

func (p *Provider) emit(ctx context.Context, e audit.Event, ar fosite.AuthorizeRequester, result, reason string) {
	e.Result = result
	e.Reason = reason
	if ar != nil {
		if ar.GetClient() != nil {
			e.ClientID = ar.GetClient().GetID()
		}
		if ar.GetID() != "" {
			e.RequestID = ar.GetID()
		}
	}
	_ = p.audit.Emit(ctx, e)
}

func (p *Provider) renderInteraction(w http.ResponseWriter, ar fosite.AuthorizeRequester, needLogin, includeConsent bool) {
	csrf := p.issueCSRF(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	loginFields := ""
	button := "Continue"
	if needLogin {
		loginFields = `<input name="login" autocomplete="username"><input name="password" type="password" autocomplete="current-password">`
		button = "Login"
	}
	consent := ""
	if includeConsent {
		consent = `<label><input type="checkbox" name="consent_approved" value="true"> Approve requested access</label>`
	}
	_, _ = fmt.Fprintf(w, `<html><body><form method="post">%s<input type="hidden" name="csrf_token" value="%s">%s%s<button type="submit">%s</button></form></body></html>`, loginFields, htmlEscape(csrf), consent, hidden(ar), button)
}

func (p *Provider) finishAuthorize(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, u domain.User, authTime time.Time, consentApproved bool) {
	client, err := p.store.GetClient(r.Context(), ar.GetClient().GetID())
	if err != nil {
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
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "consent.required", ClientID: client.ID, Subject: u.Sub, Result: "rejected", Reason: "not_approved"})
		http.Error(w, "consent required", http.StatusForbidden)
		return
	}
	if requireConsent {
		if err := p.consent.RecordConsent(r.Context(), u, client, scopes); err != nil {
			http.Error(w, "record consent failed", http.StatusInternalServerError)
			return
		}
		_ = p.audit.Emit(r.Context(), audit.Event{Time: time.Now().UTC(), Name: "consent.granted", ClientID: client.ID, Subject: u.Sub, Result: "accepted"})
	}
	p.grantRequestedScopes(ar)
	p.grantRequestedAudience(ar)
	session := p.newOIDCSession(u, ar, authTime)
	response, err := p.oauth2.NewAuthorizeResponse(fosite.NewContext(), ar, session)
	if err != nil {
		p.emit(r.Context(), audit.New("authorize.request.rejected"), ar, "rejected", err.Error())
		p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
		return
	}
	p.clearCSRF(w)
	p.emit(r.Context(), audit.New("authorize.request.accepted"), ar, "accepted", "")
	p.oauth2.WriteAuthorizeResponse(r.Context(), w, ar, response)
}
