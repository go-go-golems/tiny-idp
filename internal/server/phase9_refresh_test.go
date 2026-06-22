package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/manuel/tinyidp/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flowWithRefresh runs an authorize+token flow with the offline_access scope
// and returns the token response (which should include a refresh_token).
func flowWithRefresh(t *testing.T, ts *httptest.Server, login string) map[string]any {
	t.Helper()
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid offline_access")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm(login, auth))
	code := loc.Query().Get("code")

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "token exchange: %s", body)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(body, &tok))
	return tok
}

// refresh exchanges a refresh token and returns the parsed response + status.
func refresh(t *testing.T, ts *httptest.Server, refreshToken string) (int, map[string]any) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var v map[string]any
	_ = json.Unmarshal(body, &v)
	return resp.StatusCode, v
}

// TestPhase9_OfflineAccessIssuesRefreshToken verifies that requesting the
// offline_access scope produces a refresh_token in the token response.
func TestPhase9_OfflineAccessIssuesRefreshToken(t *testing.T) {
	_, ts := newTestServer(t)
	tok := flowWithRefresh(t, ts, "alice")
	assert.NotEmpty(t, tok["refresh_token"], "offline_access scope must produce a refresh_token")
	assert.NotEmpty(t, tok["access_token"])
	assert.NotEmpty(t, tok["id_token"])
}

// TestPhase9_NoOfflineAccessNoRefreshToken verifies that without offline_access,
// no refresh_token is issued.
func TestPhase9_NoOfflineAccessNoRefreshToken(t *testing.T) {
	_, ts := newTestServer(t)
	// Standard flow without offline_access.
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	var tok map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&tok))
	_, hasRT := tok["refresh_token"]
	assert.False(t, hasRT, "no offline_access scope should not produce a refresh_token")
}

// TestPhase9_RefreshIssuesNewAccessToken verifies that a refresh_token grant
// produces a new, different access token.
func TestPhase9_RefreshIssuesNewAccessToken(t *testing.T) {
	_, ts := newTestServer(t)
	tok := flowWithRefresh(t, ts, "alice")
	oldAccess := tok["access_token"].(string)
	rt := tok["refresh_token"].(string)

	status, resp := refresh(t, ts, rt)
	require.Equal(t, http.StatusOK, status, "refresh should succeed: %v", resp)
	newAccess := resp["access_token"].(string)
	assert.NotEqual(t, oldAccess, newAccess, "refresh must issue a new access token")
	assert.NotEmpty(t, resp["refresh_token"], "refresh must issue a new (rotated) refresh token")
}

// TestPhase9_RefreshRotationRejectsReuse verifies that after rotation, the
// old refresh token cannot be used again (reuse detection).
func TestPhase9_RefreshRotationRejectsReuse(t *testing.T) {
	_, ts := newTestServer(t)
	tok := flowWithRefresh(t, ts, "alice")
	rt := tok["refresh_token"].(string)

	// First refresh succeeds and rotates.
	status, _ := refresh(t, ts, rt)
	require.Equal(t, http.StatusOK, status)

	// Reuse of the old (rotated) token must fail.
	status2, resp2 := refresh(t, ts, rt)
	assert.Equal(t, http.StatusBadRequest, status2, "rotated refresh token must be rejected on reuse")
	assert.Equal(t, "invalid_grant", resp2["error"])
}

// TestPhase9_RefreshWrongClientRejected verifies that a refresh token cannot
// be redeemed by a different client.
func TestPhase9_RefreshWrongClientRejected(t *testing.T) {
	s, ts := newTestServer(t)
	// Issue a refresh token for dev-client.
	tok := flowWithRefresh(t, ts, "alice")
	rt := tok["refresh_token"].(string)

	// Try to redeem as a different client. Register a second client first.
	s.clients.Register(testSecondClient())
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", rt)
	form.Set("client_id", "other-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "cross-client refresh must be rejected")
}

// testSecondClient returns a client for cross-client tests.
func testSecondClient() client.Client {
	return client.Client{ID: "other-client", RedirectURIs: []string{"https://app.test/cb"}}
}
