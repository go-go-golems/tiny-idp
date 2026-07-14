package fositeadapter_test

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestPromptSelectAccountActivatesSelectedRememberedSession(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}}); err != nil {
		t.Fatal(err)
	}
	for _, user := range []struct {
		login string
		user  idpstore.User
	}{{"one", idpstore.User{ID: "u1", Sub: "subject-one", Name: "One"}}, {"two", idpstore.User{ID: "u2", Sub: "subject-two", Name: "Two"}}} {
		if err := store.PutUser(ctx, user.login, user.user); err != nil {
			t.Fatal(err)
		}
	}
	key, err := keys.GenerateRSA("select-account-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	secret := []byte("select-account-test-secret-key-32-bytes")
	now := time.Now().UTC()
	activeHandle, contextHandle := "active-account-one", "browser-context"
	activeHash := idpstore.HashSecret(secret, activeHandle)
	contextHash := idpstore.HashSecret(secret, contextHandle)
	selectedHash := idpstore.HashSecret(secret, "remembered-account-two")
	entryHash := idpstore.HashSecret(secret, "account-entry-two")
	if err := store.CreateSession(ctx, idpstore.Session{IDHash: activeHash, UserID: "u1", AuthTime: now, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSession(ctx, idpstore.Session{IDHash: selectedHash, UserID: "u2", AuthTime: now, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateBrowserContext(ctx, idpstore.BrowserContext{IDHash: contextHash, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateRememberedBrowserSession(ctx, idpstore.RememberedBrowserSession{IDHash: entryHash, ContextIDHash: contextHash, SessionIDHash: selectedHash, UserID: "u2", DisplayLabel: "Two", CreatedAt: now, LastUsedAt: now}); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: secret, AccountChooser: fositeadapter.AccountChooserConfig{Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	query := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	query.Del("login")
	query.Set("prompt", "select_account")
	request, _ := http.NewRequest(http.MethodGet, server.URL+"/authorize?"+query.Encode(), nil)
	request.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: activeHandle})
	request.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: contextHandle})
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := noRedirect.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Choose an account") {
		t.Fatalf("chooser status=%d location=%q body=%s", response.StatusCode, response.Header.Get("Location"), body)
	}
	interaction := chooserInput(t, body, "interaction")
	csrf := chooserInput(t, body, "csrf_token")
	var csrfCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "tinyidp_csrf" {
			csrfCookie = cookie
		}
	}
	if csrfCookie == nil {
		t.Fatal("csrf cookie missing")
	}
	form := url.Values{"interaction": {interaction}, "csrf_token": {csrf}, "action": {"continue"}, "account": {base64.RawURLEncoding.EncodeToString(entryHash)}}
	post, _ := http.NewRequest(http.MethodPost, server.URL+"/authorize", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: activeHandle})
	post.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: contextHandle})
	post.AddCookie(csrfCookie)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	completed, err := client.Do(post)
	if err != nil {
		t.Fatal(err)
	}
	defer completed.Body.Close()
	if completed.StatusCode != http.StatusSeeOther {
		t.Fatalf("selection status=%d", completed.StatusCode)
	}
	location, _ := url.Parse(completed.Header.Get("Location"))
	if location.Query().Get("code") == "" {
		t.Fatalf("selection did not issue code: %s", location)
	}
	var fresh *http.Cookie
	for _, cookie := range completed.Cookies() {
		if cookie.Name == "tinyidp_session" {
			fresh = cookie
		}
	}
	if fresh == nil || fresh.Value == activeHandle {
		t.Fatalf("fresh session cookie=%#v", fresh)
	}
	activated, err := store.GetSession(ctx, idpstore.HashSecret(secret, fresh.Value))
	if err != nil || activated.UserID != "u2" {
		t.Fatalf("activated session=%#v err=%v", activated, err)
	}
}

func chooserInput(t *testing.T, body []byte, name string) string {
	t.Helper()
	re := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]+)"`)
	matches := re.FindSubmatch(body)
	if len(matches) != 2 {
		t.Fatalf("missing %s in %s", name, body)
	}
	return string(matches[1])
}
