package fositeadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestStoredConsentPersistsScopeApproval(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	policy := fositeadapter.NewStoredConsent(st, time.Hour)
	user := idpstore.User{ID: "u1", Sub: "sub-1"}
	client := idpstore.Client{ID: "client-1"}

	require, err := policy.RequireConsent(ctx, user, client, []string{"openid", "email"})
	if err != nil {
		t.Fatal(err)
	}
	if !require {
		t.Fatalf("new scope set should require consent")
	}
	if err := policy.RecordConsent(ctx, user, client, []string{"email", "openid", "email"}); err != nil {
		t.Fatal(err)
	}
	require, err = policy.RequireConsent(ctx, user, client, []string{"openid", "email"})
	if err != nil {
		t.Fatal(err)
	}
	if require {
		t.Fatalf("recorded normalized scope set should not require consent")
	}

	if err := st.RevokeConsent(ctx, user.ID, client.ID, []string{"email", "openid"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	require, err = policy.RequireConsent(ctx, user, client, []string{"openid", "email"})
	if err != nil {
		t.Fatal(err)
	}
	if !require {
		t.Fatalf("revoked consent should require consent again")
	}
}

func TestPromptNoneReturnsConsentRequiredWhenNewScopesNeedConsent(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	user := idpstore.User{ID: "u1", Sub: "sub-1"}
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "email"}}
	_ = st.PutClient(ctx, client)
	_ = st.PutUser(ctx, "alice", user)
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("prompt-none-consent-secret-32"), Consent: fositeadapter.NewStoredConsent(st, 0)})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	form := authorizeForm(verifier)
	form.Set("scope", "openid")
	form.Set("consent_approved", "true")
	csrf, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrf)
	noRedirect := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	resp, err := noRedirect.Do(req)
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
	q.Set("scope", "openid email")
	q.Set("prompt", "none")
	q.Set("state", "state-consent-required")
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+q.Encode(), nil)
	getReq.AddCookie(sessionCookie)
	silent, err := noRedirect.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Body.Close()
	if silent.StatusCode != http.StatusFound && silent.StatusCode != http.StatusSeeOther {
		t.Fatalf("silent status=%d, want OAuth error redirect", silent.StatusCode)
	}
	loc, err := url.Parse(silent.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Query().Get("error") != "consent_required" {
		t.Fatalf("prompt=none error=%q location=%s", loc.Query().Get("error"), loc.String())
	}
}

func TestProductionProviderDefaultsToStoredConsent(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	user := idpstore.User{ID: "u1", Sub: "sub-1"}
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "email"}}
	_ = st.PutClient(ctx, client)
	_ = st.PutUser(ctx, "alice", user)
	svc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := svc.HashCredential(ctx, "u1", "alice", []byte("alice-password-long"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.PutPasswordCredential(ctx, credential)
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "https://issuer.example.test", Store: st, Mode: idpstore.ProductionMode, SecretKey: []byte("stored-consent-secret-32-bytes!!!"), Authenticator: svc})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	form := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	form.Set("scope", "openid email")
	form.Set("password", "alice-password-long")
	csrf, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrf)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("authorize without consent status=%d, want 403", resp.StatusCode)
	}
}
