package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
	"golang.org/x/oauth2"
)

const registrationTestOrigin = "https://app.example.test"

func TestSessionEndpointAndLogoutUseIndependentAppSession(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	csrf := bytes.Repeat([]byte{9}, sha256.Size)
	if err := store.createAppSession(ctx, "browser-token", appSession{Subject: "subject", DisplayName: "Alice", CSRFSecret: csrf, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(store, nil, nil, nil, false)
	app.now = func() time.Time { return now }
	request := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	request.AddCookie(&http.Cookie{Name: appCookieName, Value: "browser-token"})
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"authenticated":true`) || !strings.Contains(response.Body.String(), base64.RawURLEncoding.EncodeToString(csrf)) {
		t.Fatalf("session response = %d %s", response.Code, response.Body.String())
	}
	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.AddCookie(&http.Cookie{Name: appCookieName, Value: "browser-token"})
	logout.Header.Set("X-CSRF-Token", base64.RawURLEncoding.EncodeToString(csrf))
	logoutResponse := httptest.NewRecorder()
	app.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d: %s", logoutResponse.Code, logoutResponse.Body.String())
	}
	if _, err := store.getAppSession(ctx, "browser-token", now.Add(time.Second)); err == nil {
		t.Fatal("logout did not revoke application session")
	}
}

func TestLoginRejectsAmbiguousReturnTo(t *testing.T) {
	store, err := openAppStore(context.Background(), filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app := newMessageApp(store, &oidcClient{config: oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "http://issuer/authorize"}}, now: time.Now}, nil, nil, false)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login?return_to=//attacker.test", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("login status = %d", response.Code)
	}
}

func TestRegistrationEndpointCreatesOneTimePreSession(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	app := newMessageApp(store, nil, nil, nil, false)
	app.now = func() time.Time { return now }
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/registration", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("registration status = %d: %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != registerCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Value == "" {
		t.Fatalf("registration cookie = %#v", cookies)
	}
	var payload struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	csrf, err := base64.RawURLEncoding.DecodeString(payload.CSRFToken)
	if err != nil || len(csrf) != sha256.Size {
		t.Fatalf("registration CSRF = %q, %v", payload.CSRFToken, err)
	}
	attempt, err := store.consumeRegistrationAttempt(ctx, cookies[0].Value, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(attempt.CSRFSecret, csrf) || !attempt.ExpiresAt.Equal(now.Add(registrationAttemptLifetime)) {
		t.Fatalf("stored registration attempt = %#v", attempt)
	}
}

func TestMessageFeedUsesCursorAndDoesNotExposeSubject(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	stamp := time.Now().UTC()
	for _, body := range []string{"first", "second", "third"} {
		if _, err := store.createMessage(ctx, message{AuthorSubject: "private-subject", AuthorName: "Alice", Body: body, CreatedAt: stamp}); err != nil {
			t.Fatal(err)
		}
	}
	app := newMessageApp(store, nil, nil, nil, false)
	first := httptest.NewRecorder()
	app.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/messages?limit=2", nil))
	var page struct {
		Messages   []messageResponse `json:"messages"`
		NextCursor string            `json:"nextCursor"`
	}
	if first.Code != http.StatusOK || json.Unmarshal(first.Body.Bytes(), &page) != nil || len(page.Messages) != 2 || page.Messages[0].Body != "third" || page.NextCursor == "" || strings.Contains(first.Body.String(), "private-subject") {
		t.Fatalf("first page = %d: %s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	app.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/messages?before="+url.QueryEscape(page.NextCursor)+"&limit=2", nil))
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"body":"first"`) {
		t.Fatalf("second page = %d: %s", second.Code, second.Body.String())
	}
}

func TestCreateMessageUsesVerifiedSessionAuthorAndCSRF(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	csrf := bytes.Repeat([]byte{4}, sha256.Size)
	if err := store.createAppSession(ctx, "session-token", appSession{Subject: "verified-subject", DisplayName: "Verified Alice", CSRFSecret: csrf, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(store, nil, nil, nil, false)
	app.publicOrigin = registrationTestOrigin
	for name, mutate := range map[string]func(*http.Request){
		"missing csrf":   func(r *http.Request) { r.Header.Del("X-CSRF-Token") },
		"foreign origin": func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example.test") },
	} {
		t.Run(name, func(t *testing.T) {
			request := newMessageRequest(t, `{"body":"hello"}`, "session-token", csrf)
			mutate(request)
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("message create = %d: %s", response.Code, response.Body.String())
			}
		})
	}
	request := newMessageRequest(t, `{"body":"hello","authorSubject":"attacker"}`, "session-token", csrf)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("spoofed author status = %d: %s", response.Code, response.Body.String())
	}
	created := httptest.NewRecorder()
	app.ServeHTTP(created, newMessageRequest(t, `{"body":"hello"}`, "session-token", csrf))
	if created.Code != http.StatusCreated || strings.Contains(created.Body.String(), "verified-subject") {
		t.Fatalf("message create = %d: %s", created.Code, created.Body.String())
	}
	values, err := store.listMessages(ctx, nil, 10)
	if err != nil || len(values) != 1 || values[0].AuthorSubject != "verified-subject" || values[0].AuthorName != "Verified Alice" {
		t.Fatalf("stored message = %#v, %v", values, err)
	}
}

func newMessageRequest(t *testing.T, body, token string, csrf []byte) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", registrationTestOrigin)
	request.Header.Set("X-CSRF-Token", base64.RawURLEncoding.EncodeToString(csrf))
	request.AddCookie(&http.Cookie{Name: appCookieName, Value: token})
	return request
}

func TestCreateAccountRequiresPreSessionCSRFAndUsesPublicAccountService(t *testing.T) {
	ctx := context.Background()
	appStore, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer appStore.Close()
	identityStore, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "identity.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer identityStore.Close()
	accounts, err := idpaccounts.NewService(identityStore, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(appStore, nil, accounts, nil, false)
	app.publicOrigin = registrationTestOrigin
	audit := idp.NewMemorySink()
	app.audit = audit
	noPreSession := httptest.NewRecorder()
	app.ServeHTTP(noPreSession, newAccountRequest(t, createAccountRequest{Login: "alice", DisplayName: "Alice", Password: "this is a long enough password", PasswordConfirmation: "this is a long enough password"}, "", nil))
	if noPreSession.Code != http.StatusForbidden {
		t.Fatalf("account creation without pre-session = %d", noPreSession.Code)
	}

	cookie, csrf := registrationContext(t, app)
	created := httptest.NewRecorder()
	app.ServeHTTP(created, newAccountRequest(t, createAccountRequest{Login: "alice", DisplayName: "Alice", Password: "this is a long enough password", PasswordConfirmation: "this is a long enough password"}, csrf, cookie))
	if created.Code != http.StatusCreated || !strings.Contains(created.Body.String(), `"next":"/auth/login"`) {
		t.Fatalf("account creation = %d: %s", created.Code, created.Body.String())
	}
	if _, err := accounts.AuthenticatePassword(ctx, "alice", "this is a long enough password", idp.LoginMetadata{}); err != nil {
		t.Fatalf("new account cannot authenticate: %v", err)
	}
	for _, setCookie := range created.Result().Cookies() {
		if setCookie.Name == appCookieName {
			t.Fatal("registration unexpectedly created an application session")
		}
	}
	events := audit.Events()
	if len(events) != 2 || events[0].Name != "account.self_registration" || events[0].Result != "rejected" || events[0].Reason != "csrf_rejected" ||
		events[1].Name != "account.self_registration" || events[1].Result != "accepted" || events[1].Reason != "" || events[1].Subject == "" {
		t.Fatalf("registration audit events = %#v", events)
	}
	for _, event := range events {
		if len(event.Fields) != 0 {
			t.Fatalf("registration audit fields leak request data: %#v", event)
		}
	}
}

func TestCreateAccountRejectsUnknownAndMultipleJSONValues(t *testing.T) {
	ctx := context.Background()
	appStore, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer appStore.Close()
	identityStore, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "identity.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer identityStore.Close()
	accounts, err := idpaccounts.NewService(identityStore, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(appStore, nil, accounts, nil, false)
	app.publicOrigin = registrationTestOrigin
	for name, body := range map[string]string{
		"unknown field": `{"login":"alice","displayName":"Alice","password":"this is a long enough password","passwordConfirmation":"this is a long enough password","admin":true}`,
		"two values":    `{"login":"alice","displayName":"Alice","password":"this is a long enough password","passwordConfirmation":"this is a long enough password"} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			cookie, csrf := registrationContext(t, app)
			request := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-CSRF-Token", csrf)
			request.Header.Set("Origin", registrationTestOrigin)
			request.AddCookie(cookie)
			response := httptest.NewRecorder()
			app.ServeHTTP(response, request)
			if response.Code != http.StatusUnprocessableEntity || response.Body.String() != "{\"error\":\"account could not be created\"}\n" {
				t.Fatalf("account creation = %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestCreateAccountRejectsForeignOriginBeforeConsumingPreSession(t *testing.T) {
	ctx := context.Background()
	appStore, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer appStore.Close()
	identityStore, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "identity.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer identityStore.Close()
	accounts, err := idpaccounts.NewService(identityStore, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	app := newMessageApp(appStore, nil, accounts, nil, false)
	app.publicOrigin = registrationTestOrigin
	cookie, csrf := registrationContext(t, app)
	payload := createAccountRequest{Login: "alice", DisplayName: "Alice", Password: "this is a long enough password", PasswordConfirmation: "this is a long enough password"}
	foreign := httptest.NewRecorder()
	request := newAccountRequest(t, payload, csrf, cookie)
	request.Header.Set("Origin", "https://attacker.example.test")
	app.ServeHTTP(foreign, request)
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("foreign origin status = %d", foreign.Code)
	}
	fetchCrossSite := httptest.NewRecorder()
	request = newAccountRequest(t, payload, csrf, cookie)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	app.ServeHTTP(fetchCrossSite, request)
	if fetchCrossSite.Code != http.StatusForbidden {
		t.Fatalf("cross-site fetch status = %d", fetchCrossSite.Code)
	}
	created := httptest.NewRecorder()
	app.ServeHTTP(created, newAccountRequest(t, payload, csrf, cookie))
	if created.Code != http.StatusCreated {
		t.Fatalf("valid retry after foreign origin = %d: %s", created.Code, created.Body.String())
	}
}

func TestRegistrationRateLimitsAddressAndCanonicalLogin(t *testing.T) {
	app := &messageApp{addressResolver: idp.DirectClientAddressResolver{}, registrationLimiter: idp.NewFixedWindowRateLimiter(1, time.Hour)}
	firstAddress := httptest.NewRequest(http.MethodPost, "/api/accounts", nil)
	firstAddress.RemoteAddr = "192.0.2.1:1234"
	if !app.allowRegistration(firstAddress, "Alice") {
		t.Fatal("first registration was rate limited")
	}
	secondAddress := httptest.NewRequest(http.MethodPost, "/api/accounts", nil)
	secondAddress.RemoteAddr = "192.0.2.2:1234"
	if app.allowRegistration(secondAddress, " alice ") {
		t.Fatal("canonical login did not share a rate-limit key")
	}
	app.registrationLimiter = idp.NewFixedWindowRateLimiter(1, time.Hour)
	if !app.allowRegistration(firstAddress, "alice") {
		t.Fatal("first address-limit registration was rate limited")
	}
	if app.allowRegistration(firstAddress, "bob") {
		t.Fatal("address did not share a rate-limit key")
	}
}

func registrationContext(t *testing.T, app *messageApp) (*http.Cookie, string) {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/registration", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("registration context status = %d", response.Code)
	}
	var payload struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("registration cookies = %#v", cookies)
	}
	return cookies[0], payload.CSRFToken
}

func newAccountRequest(t *testing.T, payload createAccountRequest, csrf string, cookie *http.Cookie) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", registrationTestOrigin)
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return request
}

func TestBrowserLoginCompletesAgainstEmbeddedProvider(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewUnstartedServer(nil)
	baseURL := "http://" + server.Listener.Addr().String()

	identityStore, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "identity.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer identityStore.Close()
	accounts, err := idpaccounts.NewService(identityStore, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	issuer := baseURL + issuerPath
	if _, err := embeddedidp.Bootstrap(ctx, identityStore, embeddedidp.BootstrapConfig{Mode: embeddedidp.DevMode,
		Clients:      []embeddedidp.ClientSpec{embeddedidp.BrowserClient(clientID, []string{baseURL + callbackPath}, []string{baseURL + "/"}, []string{"openid", "profile"})},
		SigningKeyID: "message-app-browser-test-key"}); err != nil {
		t.Fatal(err)
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: issuer, Mode: embeddedidp.DevMode, Store: identityStore,
		Token: embeddedidp.TokenConfig{SecretKey: []byte("message-app-browser-test-secret-key-32")}, Authenticator: accounts})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close(context.Background())
	transport, err := embeddedidp.NewInProcessIssuerTransport(issuer, provider.Handler(), embeddedidp.InProcessTransportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	oidc, err := newOIDCClient(ctx, issuer, baseURL, &http.Client{Transport: transport, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	appStore, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer appStore.Close()
	app := newMessageApp(appStore, oidc, accounts, provider.Handler(), false)
	server.Config.Handler = app
	server.Start()
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	browser := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	registrationResponse, err := browser.Get(server.URL + "/api/registration")
	if err != nil {
		t.Fatal(err)
	}
	var registration struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.NewDecoder(registrationResponse.Body).Decode(&registration); err != nil {
		registrationResponse.Body.Close()
		t.Fatal(err)
	}
	registrationResponse.Body.Close()
	registrationBody, err := json.Marshal(createAccountRequest{Login: "alice", DisplayName: "Alice", Password: "correct horse battery staple 2026", PasswordConfirmation: "correct horse battery staple 2026"})
	if err != nil {
		t.Fatal(err)
	}
	registrationRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/accounts", bytes.NewReader(registrationBody))
	if err != nil {
		t.Fatal(err)
	}
	registrationRequest.Header.Set("Content-Type", "application/json")
	registrationRequest.Header.Set("Origin", server.URL)
	registrationRequest.Header.Set("X-CSRF-Token", registration.CSRFToken)
	registrationResponse, err = browser.Do(registrationRequest)
	if err != nil {
		t.Fatal(err)
	}
	if registrationResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registrationResponse.Body)
		registrationResponse.Body.Close()
		t.Fatalf("registration status = %d: %s", registrationResponse.StatusCode, body)
	}
	registrationResponse.Body.Close()
	loginResponse, err := browser.Get(server.URL + "/auth/login?return_to=/messages")
	if err != nil {
		t.Fatal(err)
	}
	defer loginResponse.Body.Close()
	if loginResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status = %d", loginResponse.StatusCode)
	}
	authorizeURL := loginResponse.Header.Get("Location")
	if !strings.HasPrefix(authorizeURL, issuer+"/authorize?") {
		t.Fatalf("login location = %q", authorizeURL)
	}

	pageResponse, err := browser.Get(authorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	page, readErr := io.ReadAll(pageResponse.Body)
	pageResponse.Body.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if pageResponse.StatusCode != http.StatusOK {
		t.Fatalf("authorize page status = %d: %s", pageResponse.StatusCode, page)
	}
	form := url.Values{"login": {"alice"}, "password": {"correct horse battery staple 2026"}, "action": {"approve"}}
	form.Set("csrf_token", hiddenFormValue(t, page, "csrf_token"))
	form.Set("interaction", hiddenFormValue(t, page, "interaction"))
	postResponse, err := browser.PostForm(issuer+"/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	defer postResponse.Body.Close()
	if postResponse.StatusCode != http.StatusFound && postResponse.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(postResponse.Body)
		t.Fatalf("authorize submit status = %d: %s", postResponse.StatusCode, body)
	}
	callbackURL := postResponse.Header.Get("Location")
	if !strings.HasPrefix(callbackURL, baseURL+callbackPath+"?") {
		t.Fatalf("callback location = %q", callbackURL)
	}

	callbackResponse, err := browser.Get(callbackURL)
	if err != nil {
		t.Fatal(err)
	}
	defer callbackResponse.Body.Close()
	if callbackResponse.StatusCode != http.StatusSeeOther || callbackResponse.Header.Get("Location") != "/messages" {
		body, _ := io.ReadAll(callbackResponse.Body)
		t.Fatalf("callback = %d %q: %s", callbackResponse.StatusCode, callbackResponse.Header.Get("Location"), body)
	}
	sessionResponse, err := browser.Get(server.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	defer sessionResponse.Body.Close()
	sessionBody, _ := io.ReadAll(sessionResponse.Body)
	if sessionResponse.StatusCode != http.StatusOK || !strings.Contains(string(sessionBody), `"authenticated":true`) || !strings.Contains(string(sessionBody), `"subject"`) {
		t.Fatalf("session = %d: %s", sessionResponse.StatusCode, sessionBody)
	}

	replayResponse, err := browser.Get(callbackURL)
	if err != nil {
		t.Fatal(err)
	}
	defer replayResponse.Body.Close()
	if replayResponse.StatusCode != http.StatusBadGateway {
		t.Fatalf("replayed callback status = %d", replayResponse.StatusCode)
	}
}

func hiddenFormValue(t *testing.T, page []byte, name string) string {
	t.Helper()
	re := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]+)"`)
	matches := re.FindStringSubmatch(string(page))
	if len(matches) != 2 {
		t.Fatalf("hidden %q not found in: %s", name, page)
	}
	return matches[1]
}
