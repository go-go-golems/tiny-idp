package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/tiny-idp/examples/tinyidp-message-app/loginui"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

//go:embed static/app/*
var messageAppAssets embed.FS

const (
	registrationAttemptLifetime = 10 * time.Minute
	maxRegistrationRequestBytes = 64 << 10
	registrationRetryAfter      = 60
)

type messageApp struct {
	store               *appStore
	oidc                *oidcClient
	accounts            *idpaccounts.Service
	registrationEnabled bool
	provider            http.Handler
	publicOrigin        string
	addressResolver     idp.ClientAddressResolver
	registrationLimiter idp.RateLimiter
	audit               idp.Sink
	interactionUI       *loginui.Renderer
	liveness            func(context.Context) idp.ReadinessReport
	readiness           func(context.Context) idp.ReadinessReport
	auditFailures       atomic.Uint64
	cookieSecure        bool
	now                 func() time.Time
	mux                 *http.ServeMux
}

var _ http.Handler = (*messageApp)(nil)

// tinyidp:development-default -- production construction replaces this test
// fallback with the initialized state root's synchronous file audit sink.
func newMessageApp(store *appStore, oidcClient *oidcClient, accounts *idpaccounts.Service, provider http.Handler, cookieSecure bool) *messageApp {
	publicOrigin := ""
	if oidcClient != nil {
		publicOrigin = oidcClient.publicOrigin
	}
	interactionUI, err := loginui.New()
	if err != nil {
		panic(err)
	}
	app := &messageApp{store: store, oidc: oidcClient, accounts: accounts, provider: provider, publicOrigin: publicOrigin, interactionUI: interactionUI,
		addressResolver: idp.DirectClientAddressResolver{}, registrationEnabled: true, registrationLimiter: idp.NewFixedWindowRateLimiter(5, time.Minute),
		audit: idp.NoopSink{}, cookieSecure: cookieSecure, now: time.Now, mux: http.NewServeMux()}
	app.mux.HandleFunc("GET /auth/login", app.handleLogin)
	app.mux.HandleFunc("GET /auth/callback", app.handleCallback)
	app.mux.HandleFunc("GET /api/session", app.handleSession)
	app.mux.HandleFunc("GET /api/registration", app.handleRegistration)
	app.mux.HandleFunc("POST /api/accounts", app.handleCreateAccount)
	app.mux.HandleFunc("GET /api/messages", app.handleListMessages)
	app.mux.HandleFunc("GET /healthz", app.handleHealth)
	app.mux.HandleFunc("GET /readyz", app.handleReady)
	app.mux.HandleFunc("POST /api/messages", app.handleCreateMessage)
	app.mux.HandleFunc("POST /auth/logout/local", app.handleLocalLogout)
	app.mux.HandleFunc("POST /auth/logout", app.handleLogout)
	app.mux.Handle("GET /static/app/", http.StripPrefix("/static/app/", http.FileServer(http.FS(messageAppAssetFS()))))
	app.mux.Handle("/static/tinyidp/", interactionUI.AssetsHandler())
	// The root fallback must not be method-qualified: /idp/ accepts both GET
	// and POST, and ServeMux rejects an overlapping GET-only root pattern.
	app.mux.HandleFunc("/", app.handleIndex)
	if provider != nil {
		app.mux.Handle("/idp/", provider)
	}
	return app
}

func (a *messageApp) handleHealth(w http.ResponseWriter, r *http.Request) {
	report := idp.ReadinessReport{Ready: true}
	if a.liveness != nil {
		report = a.liveness(r.Context())
	}
	writeApplicationReadiness(w, report)
}

func (a *messageApp) handleReady(w http.ResponseWriter, r *http.Request) {
	report := idp.ReadinessReport{Ready: true}
	if a.readiness != nil {
		report = a.readiness(r.Context())
	}
	if a.auditFailures.Load() != 0 {
		report.Ready = false
		report.Checks = append(report.Checks, idp.ReadinessCheck{Name: "application_audit", Ready: false, Reason: "audit_delivery_failed", CheckedAt: a.now().UTC()})
	}
	if a.store == nil || a.store.db == nil {
		report.Ready = false
		report.Checks = append(report.Checks, idp.ReadinessCheck{Name: "application_store", Ready: false, Reason: "store_unavailable", CheckedAt: a.now().UTC()})
	} else if err := a.store.db.PingContext(r.Context()); err != nil {
		report.Ready = false
		report.Checks = append(report.Checks, idp.ReadinessCheck{Name: "application_store", Ready: false, Reason: "store_unavailable", CheckedAt: a.now().UTC()})
	} else {
		report.Checks = append(report.Checks, idp.ReadinessCheck{Name: "application_store", Ready: true, CheckedAt: a.now().UTC()})
	}
	writeApplicationReadiness(w, report)
}

func writeApplicationReadiness(w http.ResponseWriter, report idp.ReadinessReport) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !report.Ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(report)
}

func messageAppAssetFS() fs.FS {
	assets, err := fs.Sub(messageAppAssets, "static/app")
	if err != nil {
		panic(err)
	}
	return assets
}

func (a *messageApp) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	contents, err := fs.ReadFile(messageAppAssetFS(), "index.html")
	if err != nil {
		http.Error(w, "application UI is unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	_, _ = w.Write(contents)
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
	// The existing frontend directs a successful registration through this
	// endpoint. If registration has just established the application session,
	// return to the local page instead of needlessly starting a second OIDC
	// transaction. A host can still provide an explicit account-switch control
	// that starts its own prompt=select_account request.
	if _, authenticated := a.currentSession(r); authenticated && r.URL.Query().Get("switch_account") != "1" {
		returnTo, err := normalizeReturnTo(r.URL.Query().Get("return_to"))
		if err != nil {
			http.Error(w, "invalid login request", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, returnTo, http.StatusSeeOther)
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
		_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false, "registrationEnabled": a.registrationEnabled})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"authenticated": true, "subject": session.Subject, "displayName": session.DisplayName,
		"csrfToken": base64.RawURLEncoding.EncodeToString(session.CSRFSecret), "registrationEnabled": a.registrationEnabled,
	})
}

func (a *messageApp) handleRegistration(w http.ResponseWriter, r *http.Request) {
	if !a.registrationEnabled {
		http.NotFound(w, r)
		return
	}
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

type messageResponse struct {
	ID         int64     `json:"id"`
	AuthorName string    `json:"authorName"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (a *messageApp) handleListMessages(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "invalid message page", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	before, err := decodeMessageCursor(r.URL.Query().Get("before"))
	if err != nil {
		http.Error(w, "invalid message page", http.StatusBadRequest)
		return
	}
	values, err := a.store.listMessages(r.Context(), before, limit)
	if err != nil {
		http.Error(w, "messages are temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	response := struct {
		Messages   []messageResponse `json:"messages"`
		NextCursor string            `json:"nextCursor,omitempty"`
	}{Messages: make([]messageResponse, 0, len(values))}
	for _, value := range values {
		response.Messages = append(response.Messages, messageResponse{ID: value.ID, AuthorName: value.AuthorName, Body: value.Body, CreatedAt: value.CreatedAt})
	}
	if len(values) == limit {
		response.NextCursor, err = encodeMessageCursor(messageCursor{CreatedAt: values[len(values)-1].CreatedAt, ID: values[len(values)-1].ID})
		if err != nil {
			http.Error(w, "messages are temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(response)
}

type createMessageRequest struct {
	Body string `json:"body"`
}

func (a *messageApp) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	if !a.isSameOriginUnsafeRequest(r) {
		http.Error(w, "invalid message request", http.StatusForbidden)
		return
	}
	session, ok := a.currentSession(r)
	if !ok || !csrfEqual(r.Header.Get("X-CSRF-Token"), session.CSRFSecret) {
		http.Error(w, "invalid message request", http.StatusForbidden)
		return
	}
	request, err := decodeCreateMessageRequest(w, r)
	if err != nil {
		http.Error(w, "invalid message request", http.StatusUnprocessableEntity)
		return
	}
	created, err := a.store.createMessage(r.Context(), message{AuthorSubject: session.Subject, AuthorName: session.DisplayName, Body: request.Body, CreatedAt: a.now().UTC()})
	if err != nil {
		http.Error(w, "invalid message request", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(messageResponse{ID: created.ID, AuthorName: created.AuthorName, Body: created.Body, CreatedAt: created.CreatedAt})
}

func decodeCreateMessageRequest(w http.ResponseWriter, r *http.Request) (createMessageRequest, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return createMessageRequest{}, errors.New("message request must be JSON")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRegistrationRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request createMessageRequest
	if err := decoder.Decode(&request); err != nil {
		return createMessageRequest{}, errors.Wrap(err, "decode message request")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return createMessageRequest{}, errors.New("message request must contain one JSON value")
	}
	return request, nil
}

type createAccountRequest struct {
	Login                string `json:"login"`
	DisplayName          string `json:"displayName"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"passwordConfirmation"`
}

func (a *messageApp) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !a.registrationEnabled {
		http.NotFound(w, r)
		return
	}
	if a.accounts == nil {
		a.recordRegistration(r.Context(), "rejected", "unavailable", "")
		http.Error(w, "registration is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if !a.isSameOriginUnsafeRequest(r) {
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
	if err := a.establishAppSession(r.Context(), w, user.Sub, user.Name); err != nil {
		a.recordRegistration(r.Context(), "accepted", "auto_login_unavailable", user.Sub)
		http.Error(w, "account was created but automatic sign-in is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	a.recordRegistration(r.Context(), "accepted", "", user.Sub)
	http.SetCookie(w, &http.Cookie{Name: registerCookie, Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.cookieSecure})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"next": "/"})
}

// establishAppSession creates the relying-party session after a successful
// OIDC callback or a freshly completed local registration. Registration has
// just accepted the account's initial password, so issuing the Message Desk
// session does not bypass an authentication decision. It deliberately does
// not mint an IdP browser cookie: that remains owned by the OIDC provider.
func (a *messageApp) establishAppSession(ctx context.Context, w http.ResponseWriter, subject, displayName string) error {
	if a == nil || a.store == nil || strings.TrimSpace(subject) == "" {
		return errors.New("application session dependencies are unavailable")
	}
	token, err := randomURLToken(32)
	if err != nil {
		return err
	}
	csrfSecret := make([]byte, sha256.Size)
	if _, err := rand.Read(csrfSecret); err != nil {
		return errors.Wrap(err, "generate application session CSRF secret")
	}
	now := a.now().UTC()
	if err := a.store.createAppSession(ctx, token, appSession{
		Subject: subject, DisplayName: firstNonEmpty(strings.TrimSpace(displayName), subject), CSRFSecret: csrfSecret,
		CreatedAt: now, ExpiresAt: now.Add(8 * time.Hour),
	}); err != nil {
		return err
	}
	a.setSessionCookie(w, token)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (a *messageApp) recordRegistration(ctx context.Context, result, reason, subject string) {
	if a.audit == nil {
		return
	}
	if err := a.audit.Emit(ctx, idp.Event{Time: a.now().UTC(), Name: "account.self_registration", Subject: subject, Result: result, Reason: reason}); err != nil {
		a.auditFailures.Add(1)
		log.Error().Err(err).Str("event", "account.self_registration").Msg("registration audit delivery failed; readiness degraded")
	}
}

func (a *messageApp) isSameOriginUnsafeRequest(r *http.Request) bool {
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
	a.logoutApplicationSession(w, r, true)
}

// handleLocalLogout ends only the Message Desk relying-party session. The
// provider browser session and remembered account context remain intact, so a
// later normal Sign in request may render prompt=select_account.
func (a *messageApp) handleLocalLogout(w http.ResponseWriter, r *http.Request) {
	a.logoutApplicationSession(w, r, false)
}

func (a *messageApp) logoutApplicationSession(w http.ResponseWriter, r *http.Request, endProviderSession bool) {
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
	if !endProviderSession {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	endSessionURL, err := a.endSessionURL()
	if err != nil {
		http.Error(w, "logout is temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if endSessionURL == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"endSessionUrl": endSessionURL})
}

// endSessionURL is derived exclusively from the canonical public origin and
// bootstrap-owned client registration. A browser navigation, not a server-side
// request, carries the provider's session cookies to RP-initiated logout.
func (a *messageApp) endSessionURL() (string, error) {
	if a.oidc != nil && a.oidc.endSessionEndpoint != "" {
		endpoint, err := url.Parse(a.oidc.endSessionEndpoint)
		if err != nil {
			return "", errors.Wrap(err, "parse identity end-session URL")
		}
		query := endpoint.Query()
		query.Set("client_id", clientID)
		query.Set("post_logout_redirect_uri", a.publicOrigin+"/")
		endpoint.RawQuery = query.Encode()
		return endpoint.String(), nil
	}
	if a.publicOrigin == "" {
		return "", nil
	}
	endpoint, err := url.Parse(a.publicOrigin + issuerPath + "/end-session")
	if err != nil {
		return "", errors.Wrap(err, "construct identity end-session URL")
	}
	query := endpoint.Query()
	query.Set("client_id", clientID)
	query.Set("post_logout_redirect_uri", a.publicOrigin+"/")
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
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
