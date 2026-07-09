package cmds

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/sections/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildClientRegistryMergesBuiltin verifies the merge behavior of
// buildClientRegistry: when the configured client_id matches a builtin, the
// builtin's properties (RequirePKCE, Secret) are preserved and the configured
// redirect URIs are added. This is the resolution to the open question from
// diary Step 10.
func TestBuildClientRegistryMergesBuiltin(t *testing.T) {
	t.Run("public-spa keeps RequirePKCE and adds configured redirect URI", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "public-spa",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, ok := r.Lookup("public-spa")
		require.True(t, ok)
		assert.True(t, c.RequirePKCE, "RequirePKCE must be preserved from the builtin")
		assert.Equal(t, "", c.Secret, "public-spa must stay public")
		// Builtin's redirect URI + configured one.
		assert.Contains(t, c.RedirectURIs, "http://localhost:8080/callback")
		assert.Contains(t, c.RedirectURIs, "http://localhost:9090/cb")
	})

	t.Run("web-app keeps builtin secret when none configured", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "web-app",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, _ := r.Lookup("web-app")
		assert.Equal(t, "dev-secret", c.Secret, "web-app must keep builtin dev-secret")
	})

	t.Run("web-app configured secret overrides builtin", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "web-app",
			ClientSecret: "custom-secret",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, _ := r.Lookup("web-app")
		assert.Equal(t, "custom-secret", c.Secret)
	})
}

// TestBuildClientRegistryRegistersNewPermissiveClient verifies that a
// configured client_id that does NOT match a builtin registers a new
// permissive client (the Phase 0-4 single-client behavior for custom IDs).
func TestBuildClientRegistryRegistersNewPermissiveClient(t *testing.T) {
	cfg := &oidc.Settings{
		ClientID:     "my-custom-app",
		ClientSecret: "s",
		RedirectURIs: []string{"http://localhost:8080/cb"},
	}
	r := buildClientRegistry(cfg)
	c, ok := r.Lookup("my-custom-app")
	require.True(t, ok)
	assert.False(t, c.RequirePKCE, "a new configured client is permissive (no PKCE requirement)")
	assert.Equal(t, "s", c.Secret)
	assert.Equal(t, []string{"http://localhost:8080/cb"}, c.RedirectURIs)
}

// TestBuildClientRegistryDefaultKeepsBuiltins verifies the default case
// (dev-client, no overrides) still has all three builtins present and dev-client
// unchanged by the merge (its configured defaults equal its builtin defaults).
func TestBuildClientRegistryRegistersExtraClients(t *testing.T) {
	cfg := &oidc.Settings{
		ClientID:     "web-app",
		RedirectURIs: []string{"http://localhost:9090/cb"},
		ExtraClients: []string{"web-app-2|dev-secret-2|http://localhost:9090/cb|http://localhost:9090/cb2"},
	}
	r := buildClientRegistry(cfg)
	c, ok := r.Lookup("web-app-2")
	require.True(t, ok)
	assert.Equal(t, "dev-secret-2", c.Secret)
	assert.Equal(t, []string{"http://localhost:9090/cb", "http://localhost:9090/cb2"}, c.RedirectURIs)
}

func TestBuildClientRegistryDefaultKeepsBuiltins(t *testing.T) {
	cfg := &oidc.Settings{
		ClientID:     "dev-client",
		RedirectURIs: []string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback"},
	}
	r := buildClientRegistry(cfg)
	for _, id := range []string{"dev-client", "public-spa", "web-app"} {
		_, ok := r.Lookup(id)
		assert.True(t, ok, "builtin %q must remain in registry", id)
	}
}

func TestStrictProviderHonorsSeededUserPassword(t *testing.T) {
	seeded, err := scenario.SeededUsersToScenarios([]scenario.SeededUser{{Login: "alice", Password: "alice-password", Sub: "user-alice-fixed"}})
	require.NoError(t, err)
	registry := scenario.New()
	registry.RegisterAll(seeded)
	cfg := &oidc.Settings{Issuer: "http://127.0.0.1:5556", ClientID: "public-spa", RedirectURIs: []string{"http://localhost/callback"}}
	provider, err := buildStrictProvider(cfg, buildClientRegistry(cfg), registry)
	require.NoError(t, err)
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	form := strictAuthorizeForm(verifier)
	form.Set("password", "wrong-password")
	resp := postStrictAuthorize(t, ts.URL, form)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	form = strictAuthorizeForm(verifier)
	form.Set("password", "alice-password")
	resp = postStrictAuthorize(t, ts.URL, form)
	defer resp.Body.Close()
	assert.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, resp.StatusCode)
	loc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	assert.NotEmpty(t, loc.Query().Get("code"))
}

func strictAuthorizeForm(verifier string) url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {"public-spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {strictS256(verifier)},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
	}
}

func postStrictAuthorize(t *testing.T, baseURL string, form url.Values) *http.Response {
	t.Helper()
	csrfToken, csrfCookie := fetchStrictCSRF(t, baseURL, form)
	form.Set("csrf_token", csrfToken)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func fetchStrictCSRF(t *testing.T, baseURL string, form url.Values) (string, *http.Cookie) {
	t.Helper()
	q := cloneStrictValues(form)
	q.Del("login")
	q.Del("password")
	resp, err := http.Get(baseURL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	m := re.FindStringSubmatch(string(body))
	require.Len(t, m, 2, "csrf token not found in %s", body)
	for _, c := range resp.Cookies() {
		if c.Name == "tinyidp_csrf" {
			return m[1], c
		}
	}
	t.Fatal("csrf cookie not found")
	return "", nil
}

func cloneStrictValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, values := range v {
		out[k] = append([]string(nil), values...)
	}
	return out
}

func strictS256(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
