package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	flowCookieName    = "embedded_example_flow"
	sessionCookieName = "embedded_example_session"
	flowLifetime      = 5 * time.Minute
	sessionLifetime   = 8 * time.Hour
)

type rpOptions struct {
	PublicBaseURL string
	Issuer        string
	ClientID      string
	HTTPClient    *http.Client
}

type loginFlow struct {
	State        string
	Nonce        string
	CodeVerifier string
	ExpiresAt    time.Time
}

type appSession struct {
	Subject   string
	Name      string
	Email     string
	IDToken   string
	CSRFToken string
	ExpiresAt time.Time
}

type relyingParty struct {
	opts     rpOptions
	mux      *http.ServeMux
	now      func() time.Time
	mu       sync.Mutex
	flows    map[string]loginFlow
	sessions map[string]appSession
}

var _ http.Handler = (*relyingParty)(nil)

func newRelyingParty(opts rpOptions) (*relyingParty, error) {
	base, err := url.Parse(opts.PublicBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" || base.Path != "" {
		return nil, fmt.Errorf("public base URL must be an absolute origin")
	}
	issuer, err := url.Parse(opts.Issuer)
	if err != nil || issuer.Scheme != base.Scheme || issuer.Host != base.Host || issuer.Path == "" {
		return nil, fmt.Errorf("issuer must be a path on the public application origin")
	}
	if strings.TrimSpace(opts.ClientID) == "" || opts.HTTPClient == nil {
		return nil, fmt.Errorf("client ID and back-channel HTTP client are required")
	}
	rp := &relyingParty{
		opts: opts, mux: http.NewServeMux(), now: func() time.Time { return time.Now().UTC() },
		flows: make(map[string]loginFlow), sessions: make(map[string]appSession),
	}
	rp.mux.HandleFunc("GET /", rp.home)
	rp.mux.HandleFunc("GET /login", rp.login)
	rp.mux.HandleFunc("GET /auth/callback", rp.callback)
	rp.mux.HandleFunc("POST /logout", rp.logout)
	return rp, nil
}

func (rp *relyingParty) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	rp.mux.ServeHTTP(w, r)
}

func (rp *relyingParty) home(w http.ResponseWriter, r *http.Request) {
	session, authenticated := rp.currentSession(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTemplate.Execute(w, struct {
		Authenticated bool
		Name          string
		Email         string
		Subject       string
		CSRFToken     string
	}{authenticated, session.Name, session.Email, session.Subject, session.CSRFToken})
}

func (rp *relyingParty) login(w http.ResponseWriter, r *http.Request) {
	flowID, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not begin login", http.StatusInternalServerError)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not begin login", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not begin login", http.StatusInternalServerError)
		return
	}
	verifier, err := randomToken(48)
	if err != nil {
		http.Error(w, "could not begin login", http.StatusInternalServerError)
		return
	}
	now := rp.now()
	rp.mu.Lock()
	rp.pruneLocked(now)
	rp.flows[flowID] = loginFlow{State: state, Nonce: nonce, CodeVerifier: verifier, ExpiresAt: now.Add(flowLifetime)}
	rp.mu.Unlock()
	rp.setCookie(w, flowCookieName, flowID, flowLifetime)

	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {rp.opts.ClientID},
		"redirect_uri":          {rp.opts.PublicBaseURL + "/auth/callback"},
		"scope":                 {"openid profile email"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	http.Redirect(w, r, rp.opts.Issuer+"/authorize?"+query.Encode(), http.StatusFound)
}

func (rp *relyingParty) callback(w http.ResponseWriter, r *http.Request) {
	flow, ok := rp.takeFlow(r)
	if !ok || r.URL.Query().Get("state") != flow.State {
		http.Error(w, "login response did not match the initiating browser", http.StatusBadRequest)
		return
	}
	if protocolError := r.URL.Query().Get("error"); protocolError != "" {
		http.Error(w, "identity provider rejected login: "+protocolError, http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "identity provider returned no authorization code", http.StatusBadRequest)
		return
	}
	tokens, err := exchangeCode(r.Context(), rp.opts, code, flow.CodeVerifier)
	if err != nil {
		http.Error(w, "authorization code exchange failed", http.StatusBadGateway)
		return
	}
	claims, err := verifyIDToken(r.Context(), rp.opts, tokens.IDToken, flow.Nonce)
	if err != nil {
		http.Error(w, "identity token verification failed", http.StatusBadGateway)
		return
	}
	profile, err := fetchUserInfo(r.Context(), rp.opts, tokens.AccessToken)
	if err != nil || profile.Subject == "" || profile.Subject != claims.Subject {
		http.Error(w, "UserInfo verification failed", http.StatusBadGateway)
		return
	}
	sessionID, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not establish application session", http.StatusInternalServerError)
		return
	}
	csrf, err := randomToken(32)
	if err != nil {
		http.Error(w, "could not establish application session", http.StatusInternalServerError)
		return
	}
	now := rp.now()
	rp.mu.Lock()
	rp.pruneLocked(now)
	rp.sessions[sessionID] = appSession{
		Subject: profile.Subject, Name: profile.Name, Email: profile.Email,
		IDToken: tokens.IDToken, CSRFToken: csrf, ExpiresAt: now.Add(sessionLifetime),
	}
	rp.mu.Unlock()
	rp.clearCookie(w, flowCookieName)
	rp.setCookie(w, sessionCookieName, sessionID, sessionLifetime)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (rp *relyingParty) logout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	rp.mu.Lock()
	session, ok := rp.sessions[cookie.Value]
	if ok && session.CSRFToken == r.PostForm.Get("csrf_token") {
		delete(rp.sessions, cookie.Value)
	}
	rp.mu.Unlock()
	if !ok || session.CSRFToken != r.PostForm.Get("csrf_token") {
		http.Error(w, "invalid logout request", http.StatusForbidden)
		return
	}
	rp.clearCookie(w, sessionCookieName)
	query := url.Values{
		"id_token_hint":            {session.IDToken},
		"post_logout_redirect_uri": {rp.opts.PublicBaseURL + "/"},
		"client_id":                {rp.opts.ClientID},
	}
	http.Redirect(w, r, rp.opts.Issuer+"/end-session?"+query.Encode(), http.StatusSeeOther)
}

func (rp *relyingParty) takeFlow(r *http.Request) (loginFlow, bool) {
	cookie, err := r.Cookie(flowCookieName)
	if err != nil {
		return loginFlow{}, false
	}
	now := rp.now()
	rp.mu.Lock()
	defer rp.mu.Unlock()
	flow, ok := rp.flows[cookie.Value]
	delete(rp.flows, cookie.Value)
	return flow, ok && now.Before(flow.ExpiresAt)
}

func (rp *relyingParty) currentSession(r *http.Request) (appSession, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return appSession{}, false
	}
	now := rp.now()
	rp.mu.Lock()
	defer rp.mu.Unlock()
	session, ok := rp.sessions[cookie.Value]
	if !ok || !now.Before(session.ExpiresAt) {
		delete(rp.sessions, cookie.Value)
		return appSession{}, false
	}
	return session, true
}

func (rp *relyingParty) pruneLocked(now time.Time) {
	for id, flow := range rp.flows {
		if !now.Before(flow.ExpiresAt) {
			delete(rp.flows, id)
		}
	}
	for id, session := range rp.sessions {
		if !now.Before(session.ExpiresAt) {
			delete(rp.sessions, id)
		}
	}
}

func (rp *relyingParty) setCookie(w http.ResponseWriter, name, value string, lifetime time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", HttpOnly: true,
		Secure: strings.HasPrefix(rp.opts.PublicBaseURL, "https://"), SameSite: http.SameSiteLaxMode,
		MaxAge: int(lifetime.Seconds()),
	})
}

func (rp *relyingParty) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
}

func randomToken(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

var pageTemplate = template.Must(template.New("home").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>tiny-idp embedded application</title>
<style>body{font:16px/1.55 system-ui,sans-serif;max-width:46rem;margin:5rem auto;padding:0 1.5rem;color:#17202a;background:#f6f8fa}main{background:white;border:1px solid #d8dee4;border-radius:.7rem;padding:2rem}a,button{display:inline-block;border:0;border-radius:.35rem;padding:.65rem 1rem;background:#285f8f;color:white;text-decoration:none;font:inherit;cursor:pointer}dl{display:grid;grid-template-columns:8rem 1fr;gap:.5rem}dt{font-weight:650}code{overflow-wrap:anywhere}</style>
</head><body><main><h1>Self-contained tiny-idp application</h1>
{{if .Authenticated}}<p>The application has established its own session after verifying an OIDC response.</p>
<dl><dt>Name</dt><dd>{{.Name}}</dd><dt>Email</dt><dd>{{.Email}}</dd><dt>Subject</dt><dd><code>{{.Subject}}</code></dd></dl>
<form method="post" action="/logout"><input type="hidden" name="csrf_token" value="{{.CSRFToken}}"><button type="submit">Sign out of app and IdP</button></form>
{{else}}<p>This process serves both the relying party and its identity provider. The callback, token exchange, ID-token verification, UserInfo request, and application session are implemented here.</p><p><a href="/login">Sign in with the embedded IdP</a></p>{{end}}
</main></body></html>`))
