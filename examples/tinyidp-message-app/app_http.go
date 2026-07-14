package main

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type messageApp struct {
	store        *appStore
	oidc         *oidcClient
	provider     http.Handler
	cookieSecure bool
	now          func() time.Time
	mux          *http.ServeMux
}

var _ http.Handler = (*messageApp)(nil)

func newMessageApp(store *appStore, oidcClient *oidcClient, provider http.Handler, cookieSecure bool) *messageApp {
	app := &messageApp{store: store, oidc: oidcClient, provider: provider, cookieSecure: cookieSecure, now: time.Now, mux: http.NewServeMux()}
	app.mux.HandleFunc("GET /auth/login", app.handleLogin)
	app.mux.HandleFunc("GET /auth/callback", app.handleCallback)
	app.mux.HandleFunc("GET /api/session", app.handleSession)
	app.mux.HandleFunc("POST /auth/logout", app.handleLogout)
	if provider != nil {
		app.mux.Handle("/idp/", provider)
	}
	return app
}

func (a *messageApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	a.mux.ServeHTTP(w, r)
}

func (a *messageApp) handleLogin(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil {
		http.Error(w, "identity login is unavailable", http.StatusServiceUnavailable)
		return
	}
	location, err := a.oidc.beginLogin(r.Context(), a.store, r.URL.Query().Get("return_to"))
	if err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (a *messageApp) handleCallback(w http.ResponseWriter, r *http.Request) {
	if a.oidc == nil || r.URL.Query().Get("error") != "" {
		http.Error(w, "identity login was not accepted", http.StatusBadRequest)
		return
	}
	completion, err := a.oidc.finishLogin(r.Context(), a.store, r.URL.Query().Get("state"), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "identity login could not be completed", http.StatusBadGateway)
		return
	}
	a.setSessionCookie(w, completion.SessionToken)
	http.Redirect(w, r, completion.ReturnTo, http.StatusSeeOther)
}

func (a *messageApp) handleSession(w http.ResponseWriter, r *http.Request) {
	session, ok := a.currentSession(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true, "subject": session.Subject, "displayName": session.DisplayName,
		"csrfToken": base64.RawURLEncoding.EncodeToString(session.CSRFSecret),
	})
}

func (a *messageApp) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := a.currentSession(r)
	if !ok || !csrfEqual(r.Header.Get("X-CSRF-Token"), session.CSRFSecret) {
		http.Error(w, "invalid logout request", http.StatusForbidden)
		return
	}
	cookie, _ := r.Cookie(appCookieName)
	if err := a.store.revokeAppSession(r.Context(), cookie.Value, a.now().UTC()); err != nil {
		http.Error(w, "invalid logout request", http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: appCookieName, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cookieSecure})
	w.WriteHeader(http.StatusNoContent)
}

func (a *messageApp) currentSession(r *http.Request) (appSession, bool) {
	cookie, err := r.Cookie(appCookieName)
	if err != nil || len(cookie.Value) > 1024 {
		return appSession{}, false
	}
	session, err := a.store.getAppSession(r.Context(), cookie.Value, a.now().UTC())
	return session, err == nil
}

func (a *messageApp) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: appCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int((8 * time.Hour).Seconds())})
}

func csrfEqual(raw string, expected []byte) bool {
	provided, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(raw))
	if err != nil || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(provided, expected) == 1
}
