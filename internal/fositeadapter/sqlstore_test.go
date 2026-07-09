package fositeadapter_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/sqlite"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestFositeSQLiteRefreshTokenReuseIsRejected(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-reuse", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts.URL, verifier)
	tokens := exchangeCode(t, ts.URL, code, verifier)
	oldRefresh := tokens["refresh_token"].(string)
	firstRefresh := refreshToken(t, ts.URL, oldRefresh)
	if firstRefresh["refresh_token"] == "" || firstRefresh["refresh_token"] == oldRefresh {
		t.Fatalf("refresh token was not rotated: %#v", firstRefresh)
	}
	refreshTokenMustFail(t, ts.URL, oldRefresh)
}

func TestFositeSQLiteClientWithEmptyScopesRejectsRequestedScope(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-empty-scopes", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("sqlite-empty-scopes-secret-32")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status=%d body=%s", resp.StatusCode, b)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Query().Get("error") != "invalid_scope" {
		t.Fatalf("error=%q location=%s", loc.Query().Get("error"), loc.String())
	}
}

func TestFositeSQLiteDisabledClientRejectsPersistedAuthorizationCode(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}}
	if err := st.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-disabled", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts.URL, verifier)
	client.Disabled = true
	if err := st.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	exchangeCodeMustFail(t, ts.URL, code, verifier)
}

func TestFositeSQLiteStoreSurvivesProviderRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")

	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", EmailVerified: true, Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-sqlite", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider1, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts1 := httptest.NewServer(provider1.Handler())

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts1.URL, verifier)
	ts1.Close()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st2, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	provider2, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st2, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts2 := httptest.NewServer(provider2.Handler())
	tokens := exchangeCode(t, ts2.URL, code, verifier)
	ts2.Close()
	if err := st2.Close(); err != nil {
		t.Fatal(err)
	}

	st3, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st3.Close()
	provider3, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st3, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts3 := httptest.NewServer(provider3.Handler())
	defer ts3.Close()
	refreshed := refreshToken(t, ts3.URL, tokens["refresh_token"].(string))
	if refreshed["access_token"] == "" || refreshed["refresh_token"] == "" {
		t.Fatalf("missing refreshed token fields: %#v", refreshed)
	}
}

func authorizeForCode(t *testing.T, baseURL, verifier string) string {
	t.Helper()
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
	}
	csrfToken, csrfCookie := fetchCSRF(t, baseURL, form)
	form.Set("csrf_token", csrfToken)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	reqAuth, _ := http.NewRequest(http.MethodPost, baseURL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err := client.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status = %d body=%s", resp.StatusCode, b)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in location %s", loc.String())
	}
	return code
}

func exchangeCodeMustFail(t *testing.T, baseURL, code, verifier string) {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("token exchange unexpectedly succeeded: %s", b)
	}
}

func exchangeCode(t *testing.T, baseURL, code, verifier string) map[string]any {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("token status = %d body=%s", resp.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func refreshTokenMustFail(t *testing.T, baseURL, token string) {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {token},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("refresh reuse unexpectedly succeeded: %s", b)
	}
}

func refreshToken(t *testing.T, baseURL, token string) map[string]any {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {token},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("refresh status = %d body=%s", resp.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
