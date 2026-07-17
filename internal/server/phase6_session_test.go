package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jarClient returns an http.Client bound to ts's transport with a real cookie
// jar, so the IdP session cookie set by POST /authorize persists across
// subsequent requests on the same client. ts.Client() has a nil jar, so
// session behavior is invisible without this.
func jarClient(t *testing.T, ts *httptest.Server) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &http.Client{
		Transport: ts.Client().Transport,
		Jar:       jar,
		// Don't follow redirects: tests need to inspect the 302 Location.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// establishSession performs a login on the given client, leaving a session
// cookie in the client's jar. Returns nothing; the side effect (cookie) is
// what subsequent requests rely on.
func establishSession(t *testing.T, ts *httptest.Server, c *http.Client, login string) {
	t.Helper()
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "first")
	// Use a client that DOES follow redirects here is unnecessary; the POST
	// returns a 302 we don't need to follow. authorizePostRedirect uses its
	// own client, but we need the cookie to land in c's jar. So drive the
	// POST directly on c.
	form := authorizeForm(login, auth)
	resp, err := c.PostForm(ts.URL+"/authorize", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusFound, resp.StatusCode, "login POST should redirect (set session cookie)")
}

// silentIssue performs a prompt=none authorize on c and returns the redirect
// Location (which carries the code for a valid session, or the error for an
// invalid one).
func silentIssue(t *testing.T, ts *httptest.Server, c *http.Client, state string) *url.URL {
	t.Helper()
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("state", state)
	q.Set("prompt", "none")
	resp, err := c.Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	loc, err := resp.Location()
	require.NoError(t, err)
	return loc
}

// TestPhase6_PromptNoneNoSessionReturnsLoginRequired verifies that prompt=none
// with no IdP session redirects back to the RP with error=login_required
// rather than showing a login form.
func TestPhase6_PromptNoneNoSessionReturnsLoginRequired(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	loc := silentIssue(t, ts, c, "st")
	assert.Equal(t, "login_required", loc.Query().Get("error"))
	assert.Equal(t, "st", loc.Query().Get("state"))
	assert.Empty(t, loc.Query().Get("code"), "prompt=none with no session must not issue a code")
}

// TestPhase6_PromptNoneWithSessionSilentlyIssues verifies that after a login
// (which sets a session cookie), a subsequent authorize with prompt=none
// silently issues a code without showing the login form.
func TestPhase6_PromptNoneWithSessionSilentlyIssues(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")
	loc := silentIssue(t, ts, c, "second")
	assert.Equal(t, "second", loc.Query().Get("state"))
	assert.NotEmpty(t, loc.Query().Get("code"), "prompt=none with a valid session must silently issue a code")
}

// TestPhase6_SilentIssueUsesSessionUser verifies that a silent issue (valid
// session, no prompt) produces an ID token whose sub is the session user's,
// proving the session's User is threaded through silent issuance.
func TestPhase6_SilentIssueUsesSessionUser(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "bob")

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("state", "second")
	q.Set("prompt", "none")
	resp, err := c.Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	loc, err := resp.Location()
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	defer tresp.Body.Close()
	body, _ := io.ReadAll(tresp.Body)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(body, &tok))
	claims := verifyIDTokenSignature(t, ts, tok["id_token"].(string))
	assert.Equal(t, user.FromLogin("bob").Sub, claims["sub"], "silent issue must use the session user")
}

// TestPhase6_PromptLoginForcesFormEvenWithSession verifies that prompt=login
// shows the login form even when a valid session exists (re-authentication),
// rather than silently issuing.
func TestPhase6_PromptLoginForcesFormEvenWithSession(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("prompt", "login")
	resp, err := c.Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "prompt=login must render the form (200), not silently issue (302)")
	body, _ := io.ReadAll(resp.Body)
	assert.True(t, strings.Contains(string(body), `name="login"`), "prompt=login must render the login form even with a session")
}

// TestPhase6_LoginHintPrefillsForm verifies that login_hint is rendered into
// the login input's value attribute.
func TestPhase6_LoginHintPrefillsForm(t *testing.T) {
	_, ts := newTestServer(t)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("login_hint", "alice@example.test")
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `value="alice@example.test"`, "login_hint must prefill the login field")
}

// TestPhase6_SilentIssuePreservesAuthTime verifies that a silent issue
// (prompt=none) produces an ID token whose auth_time is the *original* login
// time, not the silent-issue time. We sleep briefly so a now()-based
// auth_time would be observably later than the login time.
func TestPhase6_SilentIssuePreservesAuthTime(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")
	loginAuthTime := time.Now().Unix()

	// Sleep so a now()-based auth_time would be observably later.
	time.Sleep(1500 * time.Millisecond)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("state", "second")
	q.Set("prompt", "none")
	resp, err := c.Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	loc, err := resp.Location()
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	defer tresp.Body.Close()
	body, _ := io.ReadAll(tresp.Body)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(body, &tok))
	claims := verifyIDTokenSignature(t, ts, tok["id_token"].(string))
	authTime, _ := claims["auth_time"].(float64)

	// auth_time must be the original login time (within a small tolerance),
	// NOT the silent-issue time (~1.5s later).
	assert.Less(t, int64(authTime), loginAuthTime+1, "auth_time should be the original login time, not the silent-issue time")
	assert.GreaterOrEqual(t, int64(authTime), loginAuthTime-1, "auth_time should be the original login time")
}

// TestPhase6_MaxAgeExceedsForcesReauth verifies that a valid session with
// max_age smaller than the session's age forces re-authentication (shows the
// form), because the session is no longer "fresh enough".
func TestPhase6_MaxAgeExceedsForcesReauth(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	// Wait so the session is older than max_age=1.
	time.Sleep(1100 * time.Millisecond)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	q.Set("max_age", "1")
	resp, err := c.Get(ts.URL + "/authorize?" + q.Encode())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "max_age exceeded should force the login form")
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `name="login"`)
}
