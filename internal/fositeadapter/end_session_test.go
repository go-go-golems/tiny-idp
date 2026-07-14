package fositeadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestEndSessionRevokesCurrentBrowserSessionAndRedirects(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{
		ID: "tinyidp-xapp", Public: true, RequirePKCE: true,
		RedirectURIs:           []string{"https://app.example.test/auth/callback"},
		PostLogoutRedirectURIs: []string{"https://app.example.test/"},
		AllowedScopes:          []string{"openid"},
	}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	secret := []byte("end-session-secret-key-32-bytes!!")
	handle := "current-browser-session"
	sessionHash := idpstore.HashSecret(secret, handle)
	if err := store.CreateSession(ctx, idpstore.Session{IDHash: sessionHash, UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	sink := idp.NewMemorySink()
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:            "https://issuer.example.test/idp",
		Store:             store,
		SecretKey:         secret,
		SessionCookieName: "xapp_idp_session",
		CSRFCookieName:    "xapp_idp_csrf",
		CookieSecure:      true,
		Audit:             sink,
	})
	if err != nil {
		t.Fatal(err)
	}

	target := "/idp/end-session?client_id=tinyidp-xapp&post_logout_redirect_uri=" + url.QueryEscape("https://app.example.test/") + "&state=signed-out-everywhere"
	request := httptest.NewRequest(http.MethodGet, "https://issuer.example.test"+target, nil)
	request.AddCookie(&http.Cookie{Name: "xapp_idp_session", Value: handle, Path: "/idp"})
	recorder := httptest.NewRecorder()
	provider.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.String() != "https://app.example.test/?state=signed-out-everywhere" {
		t.Fatalf("location=%s", location.String())
	}
	session, err := store.GetSession(ctx, sessionHash)
	if err != nil || session.RevokedAt == nil {
		t.Fatalf("session not revoked: session=%#v err=%v", session, err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cleared cookies=%#v", cookies)
	}
	for _, cookie := range cookies {
		if cookie.MaxAge >= 0 || cookie.Path != "/idp" || !cookie.HttpOnly || !cookie.Secure {
			t.Fatalf("unsafe cleared cookie: %#v", cookie)
		}
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control=%q", recorder.Header().Get("Cache-Control"))
	}
	found := false
	for _, event := range sink.Events() {
		if event.Name == "logout.success" && event.ClientID == "tinyidp-xapp" && event.Result == "accepted" {
			found = true
		}
	}
	if !found {
		t.Fatalf("logout audit missing: %#v", sink.Events())
	}
}

func TestEndSessionRejectsUnregisteredRedirectBeforeRevocation(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	_ = store.PutClient(ctx, idpstore.Client{ID: "tinyidp-xapp", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/auth/callback"}, PostLogoutRedirectURIs: []string{"https://app.example.test/"}, AllowedScopes: []string{"openid"}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = store.CreateSigningKey(ctx, key)
	secret := []byte("end-session-secret-key-32-bytes!!")
	handle := "current-browser-session"
	sessionHash := idpstore.HashSecret(secret, handle)
	_ = store.CreateSession(ctx, idpstore.Session{IDHash: sessionHash, UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "https://issuer.example.test/idp", Store: store, SecretKey: secret})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://issuer.example.test/idp/end-session?client_id=tinyidp-xapp&post_logout_redirect_uri=https%3A%2F%2Fevil.example%2F", nil)
	request.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: handle})
	recorder := httptest.NewRecorder()
	provider.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
	}
	session, err := store.GetSession(ctx, sessionHash)
	if err != nil || session.RevokedAt != nil {
		t.Fatalf("invalid redirect changed session: session=%#v err=%v", session, err)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("invalid redirect cleared cookies: %#v", recorder.Result().Cookies())
	}
}

func TestEndSessionRevokesBrowserContextAndClearsItsCookie(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	key, err := keys.GenerateRSA("chooser-logout-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	secret := []byte("account-chooser-logout-secret-key-32")
	sessionHandle := "active-browser-session"
	contextHandle := "browser-context"
	sessionHash := idpstore.HashSecret(secret, sessionHandle)
	contextHash := idpstore.HashSecret(secret, contextHandle)
	now := time.Now().UTC()
	if err := store.CreateSession(ctx, idpstore.Session{IDHash: sessionHash, UserID: "u1", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateBrowserContext(ctx, idpstore.BrowserContext{IDHash: contextHash, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:    "https://issuer.example.test/idp",
		Store:     store,
		SecretKey: secret,
		AccountChooser: fositeadapter.AccountChooserConfig{
			Enabled: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://issuer.example.test/idp/end-session", nil)
	request.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: sessionHandle})
	request.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: contextHandle})
	recorder := httptest.NewRecorder()
	provider.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	session, err := store.GetSession(ctx, sessionHash)
	if err != nil || session.RevokedAt == nil {
		t.Fatalf("session not revoked: %#v err=%v", session, err)
	}
	browserContext, err := store.GetBrowserContext(ctx, contextHash)
	if err != nil || browserContext.RevokedAt == nil {
		t.Fatalf("browser context not revoked: %#v err=%v", browserContext, err)
	}
	cleared := map[string]bool{}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.MaxAge < 0 {
			cleared[cookie.Name] = true
		}
	}
	for _, name := range []string{"tinyidp_session", "tinyidp_csrf", "tinyidp_browser_context"} {
		if !cleared[name] {
			t.Fatalf("logout did not clear %q: %#v", name, recorder.Result().Cookies())
		}
	}
}
