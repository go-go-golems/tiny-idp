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
	// ClientSecrets optionally supplies plaintext client secrets for callers that
	// are converting legacy/dev config into Fosite's BCrypt client store. The
	// production embedded API should prefer BCrypt hashes in domain.Client.SecretHash.
	ClientSecrets map[string]string
}

type Provider struct {
	issuer      oidcmeta.Issuer
	store       storage.Store
	fositeStore *fositememory.MemoryStore
	oauth2      fosite.OAuth2Provider
	config      *fosite.Config
}

func NewProvider(opts Options) (*Provider, error) {
	iss, err := oidcmeta.ParseIssuer(opts.Issuer)
	if err != nil {
		return nil, err
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if len(opts.SecretKey) == 0 {
		opts.SecretKey = []byte("tinyidp-dev-secret-key-at-least-32-bytes")
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

	cfg := &fosite.Config{
		GlobalSecret:                   opts.SecretKey,
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
	p := &Provider{issuer: iss, store: opts.Store, fositeStore: fs.memoryStore, config: cfg}

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
	return mux
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
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<html><body><form method="post"><input name="login" autocomplete="username"><input name="password" type="password" autocomplete="current-password">%s<button type="submit">Login</button></form></body></html>`, hidden(ar))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		ar, err := p.oauth2.NewAuthorizeRequest(fosite.NewContext(), r)
		if err != nil {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
			return
		}
		login := strings.ToLower(strings.TrimSpace(r.PostForm.Get("login")))
		if login == "" {
			http.Error(w, "login is required", http.StatusBadRequest)
			return
		}
		u, err := p.store.GetUserByLogin(r.Context(), login)
		if err != nil {
			http.Error(w, "invalid login", http.StatusUnauthorized)
			return
		}
		p.grantRequestedScopes(ar)
		p.grantRequestedAudience(ar)
		session := p.newOIDCSession(u, ar)
		response, err := p.oauth2.NewAuthorizeResponse(fosite.NewContext(), ar, session)
		if err != nil {
			p.oauth2.WriteAuthorizeError(r.Context(), w, ar, err)
			return
		}
		p.oauth2.WriteAuthorizeResponse(r.Context(), w, ar, response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	accessRequest, err := p.oauth2.NewAccessRequest(fosite.NewContext(), r, openid.NewDefaultSession())
	if err != nil {
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
	p.grantRequestedAccessScopes(accessRequest)
	response, err := p.oauth2.NewAccessResponse(fosite.NewContext(), accessRequest)
	if err != nil {
		p.oauth2.WriteAccessError(r.Context(), w, accessRequest, err)
		return
	}
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

func (p *Provider) newOIDCSession(u domain.User, ar fosite.AuthorizeRequester) *openid.DefaultSession {
	now := time.Now().UTC()
	claims := &fositejwt.IDTokenClaims{
		Issuer:      p.issuer.String(),
		Subject:     u.Sub,
		Audience:    []string{ar.GetClient().GetID()},
		Nonce:       ar.GetRequestForm().Get("nonce"),
		IssuedAt:    now,
		RequestedAt: ar.GetRequestedAt(),
		AuthTime:    now,
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
