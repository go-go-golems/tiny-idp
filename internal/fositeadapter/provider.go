// Package fositeadapter contains the strict production-like OAuth/OIDC adapter
// seam. The first implementation is intentionally small and dependency-light so
// domain, storage, metadata, keys, embedded API, and CLI wiring can be tested
// before binding the codebase to a concrete Fosite version. The exported handler
// shape and factory list match the planned Fosite composition boundary.
package fositeadapter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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
}

type Provider struct {
	issuer     oidcmeta.Issuer
	store      storage.Store
	secretKey  []byte
	codeTTL    time.Duration
	accessTTL  time.Duration
	idTTL      time.Duration
	refreshTTL time.Duration
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
		opts.SecretKey = []byte("tinyidp-dev-secret-key")
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
	return &Provider{issuer: iss, store: opts.Store, secretKey: opts.SecretKey, codeTTL: opts.CodeTTL, accessTTL: opts.AccessTokenTTL, idTTL: opts.IDTokenTTL, refreshTTL: opts.RefreshTokenTTL}, nil
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

type authRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ar, err := p.parseAuthorize(r.Context(), r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<html><body><form method="post"><input name="login" autocomplete="username"><input name="password" type="password" autocomplete="current-password">%s<button type="submit">Login</button></form></body></html>`, hidden(ar))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		ar, err := p.parseAuthorize(r.Context(), r.PostForm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		p.issueCodeAndRedirect(w, r, ar, u)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Provider) parseAuthorize(ctx context.Context, v url.Values) (authRequest, error) {
	ar := authRequest{ResponseType: v.Get("response_type"), ClientID: v.Get("client_id"), RedirectURI: v.Get("redirect_uri"), Scope: v.Get("scope"), State: v.Get("state"), Nonce: v.Get("nonce"), CodeChallenge: v.Get("code_challenge"), CodeChallengeMethod: v.Get("code_challenge_method")}
	if ar.ResponseType != "code" {
		return ar, fmt.Errorf("unsupported response_type")
	}
	client, err := p.store.GetClient(ctx, ar.ClientID)
	if err != nil || client.Disabled {
		return ar, fmt.Errorf("unknown client_id")
	}
	if !client.AllowsRedirectURI(ar.RedirectURI) {
		return ar, fmt.Errorf("redirect_uri not allowed for this client")
	}
	scopes := domain.ParseScopes(ar.Scope)
	if !domain.HasScope(scopes, "openid") {
		return ar, fmt.Errorf("scope must include openid")
	}
	if !client.AllowsScope(scopes) {
		return ar, fmt.Errorf("scope not allowed for this client")
	}
	if ar.CodeChallenge == "" || ar.CodeChallengeMethod != "S256" {
		return ar, fmt.Errorf("S256 PKCE is required")
	}
	return ar, nil
}

func (p *Provider) issueCodeAndRedirect(w http.ResponseWriter, r *http.Request, ar authRequest, u domain.User) {
	now := time.Now()
	code := randomB64(32)
	codeHash := domain.HashSecret(p.secretKey, code)
	grantID := "grant-" + randomB64(16)
	_ = p.store.CreateGrant(r.Context(), domain.Grant{ID: grantID, UserID: u.ID, ClientID: ar.ClientID, Scope: domain.ParseScopes(ar.Scope), AuthTime: now, CreatedAt: now, ExpiresAt: now.Add(p.refreshTTL)})
	if err := p.store.CreateAuthorizationCode(r.Context(), domain.AuthorizationCode{CodeHash: codeHash, ClientID: ar.ClientID, UserID: u.ID, GrantID: grantID, RedirectURI: ar.RedirectURI, Scope: domain.ParseScopes(ar.Scope), Nonce: ar.Nonce, PKCEChallenge: ar.CodeChallenge, PKCEMethod: ar.CodeChallengeMethod, AuthTime: now, ExpiresAt: now.Add(p.codeTTL)}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ru, _ := url.Parse(ar.RedirectURI)
	q := ru.Query()
	q.Set("code", code)
	if ar.State != "" {
		q.Set("state", ar.State)
	}
	ru.RawQuery = q.Encode()
	http.Redirect(w, r, ru.String(), http.StatusFound)
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
	clientID, client, ok := p.authenticateClient(w, r)
	if !ok {
		return
	}
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		p.tokenCode(w, r, clientID, client)
	case "refresh_token":
		p.tokenRefresh(w, r, clientID)
	default:
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
	}
}

func (p *Provider) authenticateClient(w http.ResponseWriter, r *http.Request) (string, domain.Client, bool) {
	clientID, basicSecret, hasBasic := r.BasicAuth()
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	client, err := p.store.GetClient(r.Context(), clientID)
	if err != nil || client.Disabled {
		tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_id")
		return "", domain.Client{}, false
	}
	if !client.Public {
		secret := r.Form.Get("client_secret")
		if hasBasic {
			secret = basicSecret
		}
		if len(client.SecretHash) == 0 || string(domain.HashSecret(p.secretKey, secret)) != string(client.SecretHash) {
			tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_secret")
			return "", domain.Client{}, false
		}
	}
	return clientID, client, true
}

func (p *Provider) tokenCode(w http.ResponseWriter, r *http.Request, clientID string, _ domain.Client) {
	now := time.Now()
	codeHash := domain.HashSecret(p.secretKey, r.Form.Get("code"))
	ac, err := p.store.ConsumeAuthorizationCode(r.Context(), codeHash, now)
	if err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown, expired, or consumed code")
		return
	}
	if ac.ClientID != clientID || ac.RedirectURI != r.Form.Get("redirect_uri") {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "client_id or redirect_uri mismatch")
		return
	}
	if !verifyS256(ac.PKCEChallenge, r.Form.Get("code_verifier")) {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}
	u, err := p.store.GetUser(r.Context(), ac.UserID)
	if err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown user")
		return
	}
	idToken, err := p.issueIDToken(r.Context(), u, ac, now)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
		return
	}
	access := randomB64(32)
	_ = p.store.CreateAccessToken(r.Context(), domain.AccessToken{TokenHash: domain.HashSecret(p.secretKey, access), GrantID: ac.GrantID, ClientID: ac.ClientID, UserID: ac.UserID, Scope: ac.Scope, CreatedAt: now, ExpiresAt: now.Add(p.accessTTL)})
	resp := map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": int(p.accessTTL.Seconds()), "scope": strings.Join(ac.Scope, " "), "id_token": idToken}
	if domain.HasScope(ac.Scope, "offline_access") {
		rt := randomB64(32)
		_ = p.store.CreateRefreshToken(r.Context(), domain.RefreshToken{TokenHash: domain.HashSecret(p.secretKey, rt), GrantID: ac.GrantID, ClientID: ac.ClientID, UserID: ac.UserID, Scope: ac.Scope, CreatedAt: now, ExpiresAt: now.Add(p.refreshTTL)})
		resp["refresh_token"] = rt
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

func (p *Provider) tokenRefresh(w http.ResponseWriter, r *http.Request, clientID string) {
	now := time.Now()
	oldHash := domain.HashSecret(p.secretKey, r.Form.Get("refresh_token"))
	old, err := p.store.GetRefreshToken(r.Context(), oldHash)
	if err != nil || old.ClientID != clientID {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown refresh token")
		return
	}
	newRT := randomB64(32)
	next := domain.RefreshToken{TokenHash: domain.HashSecret(p.secretKey, newRT), GrantID: old.GrantID, ClientID: old.ClientID, UserID: old.UserID, Scope: old.Scope, CreatedAt: now, ExpiresAt: now.Add(p.refreshTTL)}
	if _, err := p.store.RotateRefreshToken(r.Context(), oldHash, next, now); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "refresh token rejected")
		return
	}
	access := randomB64(32)
	_ = p.store.CreateAccessToken(r.Context(), domain.AccessToken{TokenHash: domain.HashSecret(p.secretKey, access), GrantID: old.GrantID, ClientID: old.ClientID, UserID: old.UserID, Scope: old.Scope, CreatedAt: now, ExpiresAt: now.Add(p.accessTTL)})
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": int(p.accessTTL.Seconds()), "scope": strings.Join(old.Scope, " "), "refresh_token": newRT})
}

func (p *Provider) issueIDToken(ctx context.Context, u domain.User, ac domain.AuthorizationCode, now time.Time) (string, error) {
	key, err := p.store.ActiveSigningKey(ctx)
	if err != nil {
		return "", err
	}
	claims := map[string]any{"iss": p.issuer.String(), "sub": u.Sub, "aud": ac.ClientID, "exp": now.Add(p.idTTL).Unix(), "iat": now.Unix(), "auth_time": ac.AuthTime.Unix()}
	if ac.Nonce != "" {
		claims["nonce"] = ac.Nonce
	}
	for k, v := range domain.ClaimsForScopes(u, ac.Scope) {
		if k != "sub" {
			claims[k] = v
		}
	}
	return keys.SignJWT(key, claims)
}

func (p *Provider) userinfo(w http.ResponseWriter, r *http.Request) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tok == "" {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	at, err := p.store.GetAccessToken(r.Context(), domain.HashSecret(p.secretKey, tok))
	if err != nil || (!at.ExpiresAt.IsZero() && time.Now().After(at.ExpiresAt)) || at.RevokedAt != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	u, err := p.store.GetUser(r.Context(), at.UserID)
	if err != nil {
		http.Error(w, "unknown user", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, domain.ClaimsForScopes(u, at.Scope))
}

func hidden(ar authRequest) string {
	fields := map[string]string{"response_type": ar.ResponseType, "client_id": ar.ClientID, "redirect_uri": ar.RedirectURI, "scope": ar.Scope, "state": ar.State, "nonce": ar.Nonce, "code_challenge": ar.CodeChallenge, "code_challenge_method": ar.CodeChallengeMethod}
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

func verifyS256(challenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:]) == challenge
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func tokenError(w http.ResponseWriter, status int, code, desc string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": desc})
}
