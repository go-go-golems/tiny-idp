package fositeadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestBrowserSessionSilentAuthorizeAndPromptNone(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("session-secret-key-32-bytes!!!!!")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	form := authorizeForm(verifier)
	csrf, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrf)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status=%d", resp.StatusCode)
	}
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "tinyidp_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie missing")
	}

	q := authorizeForm(verifier)
	q.Del("login")
	q.Set("state", "state-2234567890")
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	getReq.AddCookie(sessionCookie)
	silent, err := client.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Body.Close()
	if silent.StatusCode != http.StatusFound && silent.StatusCode != http.StatusSeeOther {
		t.Fatalf("silent status=%d", silent.StatusCode)
	}
	loc, _ := url.Parse(silent.Header.Get("Location"))
	if loc.Query().Get("code") == "" {
		t.Fatalf("silent authorize did not issue code: %s", loc.String())
	}

	time.Sleep(2 * time.Second)
	q.Set("state", "state-max-age-1234567890")
	q.Set("max_age", "1")
	maxAgeReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	maxAgeReq.AddCookie(sessionCookie)
	maxAge, err := http.DefaultClient.Do(maxAgeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer maxAge.Body.Close()
	if maxAge.StatusCode != http.StatusOK {
		t.Fatalf("expired max_age status=%d, want login form", maxAge.StatusCode)
	}
	q.Set("state", "state-max-age-zero")
	q.Set("max_age", "0")
	maxAgeZeroReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	maxAgeZeroReq.AddCookie(sessionCookie)
	maxAgeZero, err := http.DefaultClient.Do(maxAgeZeroReq)
	if err != nil {
		t.Fatal(err)
	}
	defer maxAgeZero.Body.Close()
	if maxAgeZero.StatusCode != http.StatusOK {
		t.Fatalf("max_age=0 status=%d, want login form", maxAgeZero.StatusCode)
	}
	q.Del("max_age")

	q.Set("prompt", "none")
	q.Set("state", "state-3234567890")
	withCookieReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	withCookieReq.AddCookie(sessionCookie)
	withCookie, err := client.Do(withCookieReq)
	if err != nil {
		t.Fatal(err)
	}
	defer withCookie.Body.Close()
	loc, _ = url.Parse(withCookie.Header.Get("Location"))
	if loc.Query().Get("code") == "" || loc.Query().Get("error") != "" {
		t.Fatalf("prompt=none with session did not issue code: %s", loc.String())
	}

	q.Set("state", "state-4234567890")
	noCookieReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	noCookie, err := client.Do(noCookieReq)
	if err != nil {
		t.Fatal(err)
	}
	defer noCookie.Body.Close()
	loc, _ = url.Parse(noCookie.Header.Get("Location"))
	if loc.Query().Get("error") != "login_required" {
		t.Fatalf("prompt=none error=%q location=%s", loc.Query().Get("error"), loc.String())
	}
}

func TestConfiguredCookiesCoexistWithHostApplicationCookie(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:            "http://127.0.0.1:5556/idp",
		Store:             st,
		SecretKey:         []byte("session-secret-key-32-bytes!!!!!"),
		SessionCookieName: "xapp_idp_session",
		CSRFCookieName:    "xapp_idp_csrf",
		CookiePath:        "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	form := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	idpBaseURL := ts.URL + "/idp"
	csrf, csrfCookie := fetchCSRFNamed(t, idpBaseURL, form, "xapp_idp_csrf")
	if csrfCookie.Path != "/" {
		t.Fatalf("csrf cookie path = %q, want /", csrfCookie.Path)
	}
	form.Set("csrf_token", csrf)

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodPost, idpBaseURL+"/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "xapp_session", Value: "host-owned-session"})
	req.AddCookie(csrfCookie)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status=%d", resp.StatusCode)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "tinyidp_session" || cookie.Name == "tinyidp_csrf" {
			t.Fatalf("provider emitted default cookie %q despite explicit names", cookie.Name)
		}
		if cookie.Name == "xapp_idp_session" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("configured session cookie missing")
	}
	if sessionCookie.Path != "/" {
		t.Fatalf("session cookie path = %q, want /", sessionCookie.Path)
	}

	q := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	q.Del("login")
	q.Set("state", "coexistence-state-1234567890")
	silentReq, _ := http.NewRequest(http.MethodGet, idpBaseURL+"/authorize?"+q.Encode(), nil)
	silentReq.AddCookie(&http.Cookie{Name: "xapp_session", Value: "host-owned-session"})
	silentReq.AddCookie(sessionCookie)
	silent, err := client.Do(silentReq)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Body.Close()
	location, _ := url.Parse(silent.Header.Get("Location"))
	if (silent.StatusCode != http.StatusFound && silent.StatusCode != http.StatusSeeOther) || location.Query().Get("code") == "" {
		t.Fatalf("silent authorize with coexisting cookies failed: status=%d location=%s", silent.StatusCode, location.String())
	}
}

func TestOptInPasswordLoginCreatesRememberedBrowserSession(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Name: "Alice Example"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("chooser-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	secret := []byte("account-chooser-secret-key-32-bytes")
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:    "http://127.0.0.1:5556",
		Store:     store,
		SecretKey: secret,
		AccountChooser: fositeadapter.AccountChooserConfig{
			Enabled:                 true,
			RememberOnPasswordLogin: true,
			DisplayLabel: func(user idpstore.User) (string, error) {
				return user.Name, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()

	form := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	csrf, csrfCookie := fetchCSRF(t, server.URL, form)
	form.Set("csrf_token", csrf)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/authorize", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(csrfCookie)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusFound && response.StatusCode != http.StatusSeeOther {
		t.Fatalf("login status=%d", response.StatusCode)
	}
	var contextCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "tinyidp_browser_context" {
			contextCookie = cookie
		}
	}
	if contextCookie == nil || !contextCookie.HttpOnly || contextCookie.Value == "" {
		t.Fatalf("browser context cookie missing or unsafe: %#v", contextCookie)
	}
	entries, err := store.ListRememberedBrowserSessions(ctx, idpstore.HashSecret(secret, contextCookie.Value), time.Now().UTC())
	if err != nil || len(entries) != 1 || entries[0].DisplayLabel != "Alice Example" || entries[0].UserID != "u1" {
		t.Fatalf("remembered entries=%#v err=%v", entries, err)
	}
}

func TestAccountChooserRememberingRequiresLabelPolicy(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	key, err := keys.GenerateRSA("chooser-config-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	_, err = fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:    "http://127.0.0.1:5556",
		Store:     store,
		SecretKey: []byte("account-chooser-secret-key-32-bytes"),
		AccountChooser: fositeadapter.AccountChooserConfig{
			Enabled:                 true,
			RememberOnPasswordLogin: true,
		},
	})
	if err == nil {
		t.Fatal("expected missing display-label policy rejection")
	}
}

func authorizeForm(verifier string) url.Values {
	return url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256(verifier)}, "code_challenge_method": {"S256"}, "login": {"alice"}}
}
