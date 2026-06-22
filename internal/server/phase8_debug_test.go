package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/manuel/tinyidp/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// debugGet fetches a debug endpoint and returns the parsed JSON. httptest
// servers have loopback RemoteAddrs, so the loopback guard passes.
func debugGet(t *testing.T, ts *httptest.Server, path string) map[string]any {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + path)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "debug %s", path)
	var v map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v))
	return v
}

// TestPhase8_DebugIndexListsCounts verifies /debug returns counts and the
// endpoint map.
func TestPhase8_DebugIndexListsCounts(t *testing.T) {
	_, ts := newTestServer(t)
	v := debugGet(t, ts, "/debug")
	counts, ok := v["counts"].(map[string]any)
	require.True(t, ok, "counts missing")
	assert.Equal(t, float64(0), counts["sessions"])
	assert.Equal(t, float64(0), counts["codes"])
	assert.Equal(t, float64(0), counts["tokens"])
	endpoints, ok := v["endpoints"].(map[string]any)
	require.True(t, ok, "endpoints missing")
	assert.Contains(t, endpoints, "debug/sessions")
	assert.Contains(t, endpoints, "debug/reset")
}

// TestPhase8_DebugTokensListsIssuedToken verifies that after a flow, the
// issued access token appears in /debug/tokens with the right sub.
func TestPhase8_DebugTokensListsIssuedToken(t *testing.T) {
	_, ts := newTestServer(t)
	// Run a flow to populate state.
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")
	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	tresp.Body.Close()
	require.Equal(t, http.StatusOK, tresp.StatusCode)

	// /debug/tokens should now list one token with alice's sub.
	resp, err := ts.Client().Get(ts.URL + "/debug/tokens")
	require.NoError(t, err)
	defer resp.Body.Close()
	var list []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.NotEmpty(t, list[0]["prefix"], "token prefix should be present")
	assert.Equal(t, user.FromLogin("alice").Sub, list[0]["sub"])
}

// TestPhase8_DebugSessionsListsSession verifies that after a login (POST
// /authorize, which sets a session cookie), the session appears in
// /debug/sessions. Note: the POST itself sets the session server-side
// regardless of cookie jar.
func TestPhase8_DebugSessionsListsSession(t *testing.T) {
	_, ts := newTestServer(t)
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	c := ts.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.PostForm(ts.URL+"/authorize", authorizeForm("alice", auth))
	require.NoError(t, err)
	resp.Body.Close()

	counts := debugGet(t, ts, "/debug")["counts"].(map[string]any)
	assert.Equal(t, float64(1), counts["sessions"], "one session should exist after login")
	// /debug/sessions returns a list (not a map), so fetch it directly.
	resp2, err := ts.Client().Get(ts.URL + "/debug/sessions")
	require.NoError(t, err)
	defer resp2.Body.Close()
	var list []map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&list))
	require.Len(t, list, 1)
	assert.Equal(t, "alice", list[0]["login"])
}

// TestPhase8_DebugResetClearsState verifies that POST /debug/reset wipes
// sessions, codes, and tokens.
func TestPhase8_DebugResetClearsState(t *testing.T) {
	_, ts := newTestServer(t)
	// Populate: a login (session) + a code (from the authorize redirect) +
	// a token (exchange). The simplest single flow produces a session + token.
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")
	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	tresp.Body.Close()

	// Before reset: state is non-empty.
	before := debugGet(t, ts, "/debug")["counts"].(map[string]any)
	assert.Greater(t, before["sessions"].(float64), float64(0))
	assert.Greater(t, before["tokens"].(float64), float64(0))

	// POST /debug/reset.
	resp, err := ts.Client().PostForm(ts.URL+"/debug/reset", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// After reset: all zero.
	after := debugGet(t, ts, "/debug")["counts"].(map[string]any)
	assert.Equal(t, float64(0), after["sessions"])
	assert.Equal(t, float64(0), after["codes"])
	assert.Equal(t, float64(0), after["tokens"])
}

// TestPhase8_DebugResetIsPostOnly verifies a GET to /debug/reset is rejected
// (405), so a stray GET cannot wipe state.
func TestPhase8_DebugResetIsPostOnly(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/debug/reset")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestPhase8_DebugCodesListsOutstandingCode verifies that an unexchanged auth
// code appears in /debug/codes, and disappears once exchanged.
func TestPhase8_DebugCodesListsOutstandingCode(t *testing.T) {
	_, ts := newTestServer(t)
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")

	// Before exchange: code is present.
	resp, err := ts.Client().Get(ts.URL + "/debug/codes")
	require.NoError(t, err)
	var list []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	resp.Body.Close()
	require.Len(t, list, 1)
	assert.Equal(t, "dev-client", list[0]["client_id"])
	assert.Equal(t, "https://app.test/cb", list[0]["redirect_uri"])

	// Exchange the code (one-time use).
	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	tresp.Body.Close()

	// After exchange: code is gone.
	resp2, err := ts.Client().Get(ts.URL + "/debug/codes")
	require.NoError(t, err)
	var list2 []map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&list2))
	resp2.Body.Close()
	assert.Empty(t, list2, "exchanged code should be removed from /debug/codes")
}
