package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-go-golems/tiny-idp/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// endSession issues a GET /end-session on the given client (which must not
// follow redirects) and returns the response. Using the caller's client means
// the session cookie is sent and, on logout, cleared in that client's jar.
func endSession(t *testing.T, ts *httptest.Server, c *http.Client, q url.Values) *http.Response {
	t.Helper()
	u := ts.URL + "/end-session"
	if q != nil {
		u += "?" + q.Encode()
	}
	resp, err := c.Get(u)
	require.NoError(t, err)
	return resp
}

// TestPhase11_DiscoveryAdvertisesEndSession verifies the discovery document
// exposes end_session_endpoint.
func TestPhase11_DiscoveryAdvertisesEndSession(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/.well-known/openid-configuration")
	require.NoError(t, err)
	defer resp.Body.Close()
	var disc map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&disc))
	assert.Equal(t, ts.URL+"/end-session", disc["end_session_endpoint"])
}

// TestPhase11_LogoutWithoutRedirectShowsPage verifies that /end-session with
// no post_logout_redirect_uri clears the session and returns a 200 page.
func TestPhase11_LogoutWithoutRedirectShowsPage(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	resp := endSession(t, ts, c, nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Logged out")
}

// TestPhase11_LogoutClearsSession verifies that after logout, a subsequent
// prompt=none authorize request returns login_required (the session is gone).
func TestPhase11_LogoutClearsSession(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	// Before logout: prompt=none silently issues a code.
	pre := silentIssue(t, ts, c, "pre")
	assert.NotEmpty(t, pre.Query().Get("code"), "before logout, prompt=none must issue a code")

	// Logout (no redirect URI -> logged-out page).
	resp := endSession(t, ts, c, nil)
	resp.Body.Close()

	// After logout: prompt=none returns login_required.
	post := silentIssue(t, ts, c, "post")
	assert.Equal(t, "login_required", post.Query().Get("error"))
	assert.Empty(t, post.Query().Get("code"), "after logout, no code must be issued")
}

// TestPhase11_PostLogoutRedirect verifies that a valid post_logout_redirect_uri
// results in a 302 to that URI.
func TestPhase11_PostLogoutRedirect(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	q := url.Values{}
	q.Set("post_logout_redirect_uri", "https://app.test/logout")
	resp := endSession(t, ts, c, q)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	loc, err := resp.Location()
	require.NoError(t, err)
	assert.Equal(t, "app.test", loc.Host)
	assert.Equal(t, "/logout", loc.Path)
}

// TestPhase11_StateForwarded verifies that the state parameter is appended to
// the post-logout redirect.
func TestPhase11_StateForwarded(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	q := url.Values{}
	q.Set("post_logout_redirect_uri", "https://app.test/logout")
	q.Set("state", "xyz123")
	resp := endSession(t, ts, c, q)
	defer resp.Body.Close()
	loc, err := resp.Location()
	require.NoError(t, err)
	assert.Equal(t, "xyz123", loc.Query().Get("state"))
}

// TestPhase11_InvalidPostLogoutRedirectRejected verifies that an unregistered
// post_logout_redirect_uri is rejected with 400.
func TestPhase11_InvalidPostLogoutRedirectRejected(t *testing.T) {
	_, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	q := url.Values{}
	q.Set("post_logout_redirect_uri", "https://evil.test/cb")
	resp := endSession(t, ts, c, q)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestPhase11_ClientScopedPostLogout verifies that a post_logout_redirect_uri
// valid for one client is rejected when client_id names a different client
// that does not allow it.
func TestPhase11_ClientScopedPostLogout(t *testing.T) {
	s, ts := newTestServer(t)
	c := jarClient(t, ts)
	establishSession(t, ts, c, "alice")

	// Register a second client with a different post-logout URI.
	s.clients.Register(client.Client{
		ID:                     "other-client",
		RedirectURIs:           []string{"https://app.test/cb"},
		PostLogoutRedirectURIs: []string{"https://app.test/other-logout"},
	})

	// dev-client's logout URI, but client_id=other-client: must be rejected.
	q := url.Values{}
	q.Set("client_id", "other-client")
	q.Set("post_logout_redirect_uri", "https://app.test/logout")
	resp := endSession(t, ts, c, q)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"post_logout_redirect_uri not registered for other-client must be rejected")
}

// TestPhase11_IDTokenHintDeletesSession verifies that passing id_token_hint
// ends the session identified by that token's sub — even from a different
// client that has no session cookie. This proves the hint deletes by subject
// rather than merely clearing the caller's cookie.
func TestPhase11_IDTokenHintDeletesSession(t *testing.T) {
	_, ts := newTestServer(t)
	c1 := jarClient(t, ts)
	establishSession(t, ts, c1, "alice")

	// Before logout: prompt=none on c1 issues a code.
	pre := silentIssue(t, ts, c1, "pre")
	require.NotEmpty(t, pre.Query().Get("code"), "session should exist before logout")

	// Obtain an ID token for alice (separate flow; sub is stable per login).
	idToken := flowIDToken(t, ts, "alice")

	// Logout from a *different* client (no session cookie) using the hint.
	c2 := jarClient(t, ts)
	q := url.Values{}
	q.Set("id_token_hint", idToken)
	resp := endSession(t, ts, c2, q)
	resp.Body.Close()

	// c1 still holds its cookie, but the server-side session for alice's sub
	// was deleted by the hint, so prompt=none now returns login_required.
	post := silentIssue(t, ts, c1, "post")
	assert.Equal(t, "login_required", post.Query().Get("error"),
		"id_token_hint must delete the session by subject, not just clear a cookie")
}
