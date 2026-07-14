package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
	"golang.org/x/oauth2"
)

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
	app := newMessageApp(store, nil, nil, false)
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
	app := newMessageApp(store, &oidcClient{config: oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "http://issuer/authorize"}}, now: time.Now}, nil, false)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auth/login?return_to=//attacker.test", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("login status = %d", response.Code)
	}
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
	if _, err := accounts.Create(ctx, idpaccounts.CreateRequest{Login: "alice", Password: []byte("correct horse battery staple"), Name: "Alice"}); err != nil {
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
	app := newMessageApp(appStore, oidc, provider.Handler(), false)
	server.Config.Handler = app
	server.Start()
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	browser := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
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
	form := url.Values{"login": {"alice"}, "password": {"correct horse battery staple"}, "action": {"approve"}}
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
