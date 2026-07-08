package fositeadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestStoredConsentPersistsScopeApproval(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	policy := fositeadapter.NewStoredConsent(st, time.Hour)
	user := domain.User{ID: "u1", Sub: "sub-1"}
	client := domain.Client{ID: "client-1"}

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

func TestProductionProviderDefaultsToStoredConsent(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	user := domain.User{ID: "u1", Sub: "sub-1"}
	client := domain.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "email"}}
	_ = st.PutClient(ctx, client)
	_ = st.PutUser(ctx, "alice", user)
	svc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := svc.HashCredential("u1", "alice", []byte("alice-password"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.PutPasswordCredential(ctx, credential)
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_ = st.CreateSigningKey(ctx, key)
	p, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "https://issuer.example.test", Store: st, Mode: domain.ProductionMode, SecretKey: []byte("stored-consent-secret-32-bytes!!!"), Authenticator: svc})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	form := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	form.Set("scope", "openid email")
	form.Set("password", "alice-password")
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
