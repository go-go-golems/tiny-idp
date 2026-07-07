package fositeadapter_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
)

func TestStrictAuthorizationCodeFlow(t *testing.T) {
	ctx := context.Background()
	secretKey := []byte("test-secret-key-32-bytes-minimum!!")
	st := memory.New()
	if err := st.PutClient(ctx, domain.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", domain.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", EmailVerified: true, Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}

	p, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	challenge := s256(verifier)
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
	}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	noRedirect := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	reqAuth, _ := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err := noRedirect.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("authorize status = %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	cb, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	code := cb.Query().Get("code")
	if code == "" || cb.Query().Get("state") != "state-1234567890" {
		t.Fatalf("bad callback location: %s", loc)
	}

	tokResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tokResp.Body.Close()
	if tokResp.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d", tokResp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(tokResp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["id_token"] == "" || body["access_token"] == "" || body["refresh_token"] == "" {
		t.Fatalf("missing token fields: %#v", body)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+body["access_token"].(string))
	uiResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer uiResp.Body.Close()
	if uiResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(uiResp.Body)
		t.Fatalf("userinfo status = %d body=%s", uiResp.StatusCode, b)
	}
	var claims map[string]any
	if err := json.NewDecoder(uiResp.Body).Decode(&claims); err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "user-alice" || claims["email"] != "alice@example.test" {
		t.Fatalf("bad userinfo: %#v", claims)
	}

	refreshResp, err := http.PostForm(ts.URL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {body["refresh_token"].(string)},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(refreshResp.Body)
		t.Fatalf("refresh status = %d body=%s", refreshResp.StatusCode, b)
	}
	var refreshed map[string]any
	if err := json.NewDecoder(refreshResp.Body).Decode(&refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed["access_token"] == "" || refreshed["refresh_token"] == "" {
		t.Fatalf("missing refreshed token fields: %#v", refreshed)
	}
}

func TestStrictProviderHasNoDebugRoute(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	p, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/debug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/debug status = %d, want 404", resp.StatusCode)
	}
}

func fetchCSRF(t *testing.T, baseURL string, form url.Values) (string, *http.Cookie) {
	t.Helper()
	q := cloneValues(form)
	q.Del("login")
	resp, err := http.Get(baseURL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	m := re.FindStringSubmatch(string(body))
	if len(m) != 2 {
		t.Fatalf("csrf token not found in %s", body)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "tinyidp_csrf" {
			return m[1], c
		}
	}
	t.Fatal("csrf cookie not found")
	return "", nil
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vv := range v {
		out[k] = append([]string(nil), vv...)
	}
	return out
}

func s256(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
