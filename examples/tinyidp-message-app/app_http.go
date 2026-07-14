package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/pkg/errors"
)

const (
	registrationAttemptLifetime = 10 * time.Minute
	maxRegistrationRequestBytes = 64 << 10
	registrationRetryAfter      = 60
)

type messageApp struct {
	store               *appStore
	oidc                *oidcClient
	accounts            *idpaccounts.Service
	provider            http.Handler
	publicOrigin        string
	addressResolver     idp.ClientAddressResolver
	registrationLimiter idp.RateLimiter
	audit               idp.Sink
	cookieSecure        bool
	now                 func() time.Time
	mux                 *http.ServeMux
}

var _ http.Handler = (*messageApp)(nil)

func newMessageApp(store *appStore, oidcClient *oidcClient, accounts *idpaccounts.Service, provider http.Handler, cookieSecure bool) *messageApp {
	publicOrigin := ""
	if oidcClient != nil {
		publicOrigin = oidcClient.publicOrigin
	}
	app := &messageApp{store: store, oidc: oidcClient, accounts: accounts, provider: provider, publicOrigin: publicOrigin,
		addressResolver: idp.DirectClientAddressResolver{}, registrationLimiter: idp.NewFixedWindowRateLimiter(5, time.Minute),
		audit: idp.NoopSink{}, cookieSecure: cookieSecure, now: time.Now, mux: http.NewServeMux()}
	app.mux.HandleFunc("GET /auth/login", app.handleLogin)
	app.mux.HandleFunc("GET /auth/callback", app.handleCallback)
	app.mux.HandleFunc("GET /api/session", app.handleSession)
	app.mux.HandleFunc("GET /api/registration", app.handleRegistration)
	app.mux.HandleFunc("POST /api/accounts", app.handleCreateAccount)
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

func (a *messageApp) handleRegistration(w http.ResponseWriter, r *http.Request) {
	token, err := randomURLToken(32)
	if err != nil {
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	csrfSecret := make([]byte, sha256.Size)
	if _, err := rand.Read(csrfSecret); err != nil {
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	now := a.now().UTC()
	if err := a.store.createRegistrationAttempt(r.Context(), token, registrationAttempt{
		CSRFSecret: csrfSecret, CreatedAt: now, ExpiresAt: now.Add(registrationAttemptLifetime),
	}); err != nil {
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: registerCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: a.cookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(registrationAttemptLifetime.Seconds())})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"csrfToken": base64.RawURLEncoding.EncodeToString(csrfSecret)})
}

type createAccountRequest struct {
	Login                string `json:"login"`
	DisplayName          string `json:"displayName"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"passwordConfirmation"`
}

func (a *messageApp) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if a.accounts == nil {
		a.recordRegistration(r.Context(), "rejected", "unavailable", "")
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if !a.isSameOriginRegistrationRequest(r) {
		a.recordRegistration(r.Context(), "rejected", "origin_rejected", "")
		http.Error(w, "invalid registration request", http.StatusForbidden)
		return
	}
	attempt, ok := a.currentRegistrationAttempt(r)
	if !ok || !csrfEqual(r.Header.Get("X-CSRF-Token"), attempt.CSRFSecret) {
		a.recordRegistration(r.Context(), "rejected", "csrf_rejected", "")
		http.Error(w, "invalid registration request", http.StatusForbidden)
		return
	}
	request, err := decodeCreateAccountRequest(w, r)
	if err != nil || strings.TrimSpace(request.Login) == "" || strings.TrimSpace(request.DisplayName) == "" || request.Password != request.PasswordConfirmation {
		a.recordRegistration(r.Context(), "rejected", "invalid_request", "")
		writeAccountCreationError(w, http.StatusUnprocessableEntity)
		return
	}
	if !a.allowRegistration(r, request.Login) {
		a.recordRegistration(r.Context(), "rejected", "rate_limited", "")
		w.Header().Set("Retry-After", strconv.Itoa(registrationRetryAfter))
		writeAccountCreationError(w, http.StatusTooManyRequests)
		return
	}
	password := []byte(request.Password)
	defer clearBytes(password)
	user, err := a.accounts.Create(r.Context(), idpaccounts.CreateRequest{
		Login: request.Login, Name: request.DisplayName, PreferredUsername: request.Login, Password: password,
	})
	if err != nil {
		if errors.Is(err, idpstore.ErrDuplicate) || errors.Is(err, idp.ErrPasswordRejected) {
			a.recordRegistration(r.Context(), "rejected", "account_rejected", "")
			writeAccountCreationError(w, http.StatusUnprocessableEntity)
			return
		}
		a.recordRegistration(r.Context(), "rejected", "unavailable", "")
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	a.recordRegistration(r.Context(), "accepted", "", user.Sub)
	http.SetCookie(w, &http.Cookie{Name: registerCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cookieSecure})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"next": "/auth/login"})
}

func (a *messageApp) recordRegistration(ctx context.Context, result, reason, subject string) {
	if a.audit == nil {
		return
	}
	_ = a.audit.Emit(ctx, idp.Event{Time: a.now().UTC(), Name: "account.self_registration", Subject: subject, Result: result, Reason: reason})
}

func (a *messageApp) isSameOriginRegistrationRequest(r *http.Request) bool {
	if a.publicOrigin == "" || r.Header.Get("Origin") != a.publicOrigin {
		return false
	}
	return strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))) != "cross-site"
}

func (a *messageApp) allowRegistration(r *http.Request, login string) bool {
	if a.addressResolver == nil || a.registrationLimiter == nil {
		return false
	}
	address, err := a.addressResolver.ResolveClientAddress(r)
	if err != nil {
		return false
	}
	normalizedLogin := idpaccounts.NormalizeLogin(login)
	if normalizedLogin == "" {
		return false
	}
	loginHash := sha256.Sum256([]byte(normalizedLogin))
	return a.registrationLimiter.Allow(r.Context(), "registration:address:"+address) &&
		a.registrationLimiter.Allow(r.Context(), "registration:login:"+base64.RawURLEncoding.EncodeToString(loginHash[:]))
}

func (a *messageApp) currentRegistrationAttempt(r *http.Request) (registrationAttempt, bool) {
	cookie, err := r.Cookie(registerCookie)
	if err != nil || len(cookie.Value) > 1024 {
		return registrationAttempt{}, false
	}
	attempt, err := a.store.consumeRegistrationAttempt(r.Context(), cookie.Value, a.now().UTC())
	return attempt, err == nil
}

func decodeCreateAccountRequest(w http.ResponseWriter, r *http.Request) (createAccountRequest, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return createAccountRequest{}, errors.New("registration request must be JSON")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRegistrationRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request createAccountRequest
	if err := decoder.Decode(&request); err != nil {
		return createAccountRequest{}, errors.Wrap(err, "decode registration request")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return createAccountRequest{}, errors.New("registration request must contain one JSON value")
	}
	return request, nil
}

func writeAccountCreationError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "account could not be created"})
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
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
