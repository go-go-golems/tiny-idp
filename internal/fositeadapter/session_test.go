package fositeadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestBrowserSessionSilentAuthorizeAndPromptNone(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, domain.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}})
	_ = st.PutUser(ctx, "alice", domain.User{ID: "u1", Sub: "user-alice"})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("session-secret-key-32-bytes!!!!!")})
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

	q.Set("prompt", "none")
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

func authorizeForm(verifier string) url.Values {
	return url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256(verifier)}, "code_challenge_method": {"S256"}, "login": {"alice"}}
}
