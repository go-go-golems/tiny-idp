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

	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

type selectAccountFixture struct {
	ctx           context.Context
	store         *memory.Store
	secret        []byte
	activeHandle  string
	contextHandle string
	entryHash     []byte
	server        *httptest.Server
}

func newSelectAccountFixture(t *testing.T, consentFactory func(*memory.Store) idp.ConsentPolicy) *selectAccountFixture {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
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
	var consent idp.ConsentPolicy
	if consentFactory != nil {
		consent = consentFactory(store)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: secret, Consent: consent, AccountChooser: fositeadapter.AccountChooserConfig{Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	t.Cleanup(server.Close)
	return &selectAccountFixture{ctx: ctx, store: store, secret: secret, activeHandle: activeHandle, contextHandle: contextHandle, entryHash: entryHash, server: server}
}

func (f *selectAccountFixture) begin(t *testing.T) (url.Values, *http.Cookie) {
	t.Helper()
	query := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	query.Del("login")
	query.Set("prompt", "select_account")
	request, _ := http.NewRequest(http.MethodGet, f.server.URL+"/authorize?"+query.Encode(), nil)
	request.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: f.activeHandle})
	request.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: f.contextHandle})
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
	return url.Values{"interaction": {interaction}, "csrf_token": {csrf}, "action": {"continue"}, "account": {base64.RawURLEncoding.EncodeToString(f.entryHash)}}, csrfCookie
}

func (f *selectAccountFixture) submit(t *testing.T, form url.Values, csrfCookie *http.Cookie) *http.Response {
	t.Helper()
	post, _ := http.NewRequest(http.MethodPost, f.server.URL+"/authorize", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: "tinyidp_session", Value: f.activeHandle})
	post.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: f.contextHandle})
	post.AddCookie(csrfCookie)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	completed, err := client.Do(post)
	if err != nil {
		t.Fatal(err)
	}
	return completed
}

func TestPromptSelectAccountActivatesSelectedRememberedSession(t *testing.T) {
	fixture := newSelectAccountFixture(t, nil)
	form, csrfCookie := fixture.begin(t)
	completed := fixture.submit(t, form, csrfCookie)
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
	if fresh == nil || fresh.Value == fixture.activeHandle {
		t.Fatalf("fresh session cookie=%#v", fresh)
	}
	activated, err := fixture.store.GetSession(fixture.ctx, idpstore.HashSecret(fixture.secret, fresh.Value))
	if err != nil || activated.UserID != "u2" {
		t.Fatalf("activated session=%#v err=%v", activated, err)
	}
}

func TestPromptSelectAccountCreatesFreshConsentInteraction(t *testing.T) {
	fixture := newSelectAccountFixture(t, func(store *memory.Store) idp.ConsentPolicy {
		return fositeadapter.NewStoredConsent(store, 0)
	})
	form, csrfCookie := fixture.begin(t)
	completed := fixture.submit(t, form, csrfCookie)
	defer completed.Body.Close()
	body, err := io.ReadAll(completed.Body)
	if err != nil {
		t.Fatal(err)
	}
	if completed.StatusCode != http.StatusOK || !strings.Contains(string(body), "Approve access") || strings.Contains(string(body), "Choose an account") {
		t.Fatalf("selection should transition to consent status=%d body=%s", completed.StatusCode, body)
	}
	consentHandle := chooserInput(t, body, "interaction")
	consentCSRF := chooserInput(t, body, "csrf_token")
	var fresh *http.Cookie
	for _, cookie := range completed.Cookies() {
		if cookie.Name == "tinyidp_session" {
			fresh = cookie
		}
	}
	if fresh == nil || fresh.Value == fixture.activeHandle {
		t.Fatalf("fresh session cookie=%#v", fresh)
	}
	approval := url.Values{"interaction": {consentHandle}, "csrf_token": {consentCSRF}, "action": {"approve"}}
	post, _ := http.NewRequest(http.MethodPost, fixture.server.URL+"/authorize", strings.NewReader(approval.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(fresh)
	post.AddCookie(&http.Cookie{Name: "tinyidp_browser_context", Value: fixture.contextHandle})
	for _, cookie := range completed.Cookies() {
		if cookie.Name == "tinyidp_csrf" {
			post.AddCookie(cookie)
		}
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	approved, err := client.Do(post)
	if err != nil {
		t.Fatal(err)
	}
	defer approved.Body.Close()
	if approved.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(approved.Body)
		t.Fatalf("consent approval status=%d body=%s", approved.StatusCode, body)
	}
	location, _ := url.Parse(approved.Header.Get("Location"))
	if location.Query().Get("code") == "" {
		t.Fatalf("consent approval did not issue code: %s", location)
	}
}

func TestPromptSelectAccountUseAnotherAccountRequiresCredentials(t *testing.T) {
	fixture := newSelectAccountFixture(t, nil)
	form, csrfCookie := fixture.begin(t)
	form.Del("account")
	form.Set("action", string(idpui.ActionUseAnotherAccount))
	response := fixture.submit(t, form, csrfCookie)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="password"`) || strings.Contains(string(body), "Choose an account") {
		t.Fatalf("use-another should render a credential prompt status=%d body=%s", response.StatusCode, body)
	}
	if !strings.Contains(string(body), `value="approve"`) || !strings.Contains(string(body), `value="deny"`) {
		t.Fatalf("credential prompt does not have terminal actions: %s", body)
	}
}

func TestPromptSelectAccountRemovesOnlySelectedRememberedMembership(t *testing.T) {
	fixture := newSelectAccountFixture(t, nil)
	form, csrfCookie := fixture.begin(t)
	form.Set("action", string(idpui.ActionRemoveAccount))
	response := fixture.submit(t, form, csrfCookie)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `name="password"`) || strings.Contains(string(body), "Choose an account") {
		t.Fatalf("removing final account should require credentials status=%d body=%s", response.StatusCode, body)
	}
	contextHash := idpstore.HashSecret(fixture.secret, fixture.contextHandle)
	if _, _, err := fixture.store.ActivateRememberedSession(fixture.ctx, contextHash, fixture.entryHash, idpstore.HashSecret(fixture.secret, "must-not-activate"), time.Now().UTC()); err == nil {
		t.Fatal("removed remembered account could still be activated")
	}
	if _, err := fixture.store.GetUser(fixture.ctx, "u2"); err != nil {
		t.Fatalf("removing remembered membership removed the account: %v", err)
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
