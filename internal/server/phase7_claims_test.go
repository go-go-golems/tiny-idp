package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runClaimFlow drives authorize (GET form -> POST login) -> token for the
// given login, returning the verified ID token claims and the userinfo
// response body. Used by Phase 7 claim-variant tests.
func runClaimFlow(t *testing.T, ts *httptest.Server, login string) (map[string]any, map[string]any) {
	t.Helper()
	var idClaims map[string]any
	var userinfo map[string]any

	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid profile email")
	auth.Set("state", "st")
	auth.Set("nonce", "n")
	loc := authorizePostRedirect(t, ts, authorizeForm(login, auth))
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "token exchange failed: %s", body)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(body, &tok))
	idClaims = verifyIDTokenSignature(t, ts, tok["id_token"].(string))

	access := tok["access_token"].(string)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	uResp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer uResp.Body.Close()
	uBody, _ := io.ReadAll(uResp.Body)
	require.Equal(t, http.StatusOK, uResp.StatusCode, "userinfo failed: %s", uBody)
	require.NoError(t, json.Unmarshal(uBody, &userinfo))
	return idClaims, userinfo
}

// TestPhase7_AdminClaims verifies the admin scenario emits groups, roles, and
// preferred_username in both the ID token and userinfo, and that the two agree.
func TestPhase7_AdminClaims(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "admin")

	assert.Equal(t, []any{"admin", "engineering"}, idClaims["groups"], "ID token groups")
	assert.Equal(t, []any{"owner"}, idClaims["roles"], "ID token roles")
	assert.Equal(t, "admin", idClaims["preferred_username"])

	// Userinfo must agree with the ID token on the extra claims.
	assert.Equal(t, idClaims["groups"], ui["groups"], "userinfo groups must match ID token")
	assert.Equal(t, idClaims["roles"], ui["roles"], "userinfo roles must match ID token")
	assert.Equal(t, idClaims["preferred_username"], ui["preferred_username"])
}

// TestPhase7_ViewerClaims verifies the viewer scenario.
func TestPhase7_ViewerClaims(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "viewer")
	assert.Equal(t, []any{"viewer"}, idClaims["groups"])
	assert.Equal(t, []any{"reader"}, idClaims["roles"])
	assert.Equal(t, idClaims["groups"], ui["groups"])
}

// TestPhase7_NoGroupsOmitsClaims verifies that the no-groups scenario emits no
// groups/roles claims at all, so RPs that require them can be tested.
func TestPhase7_NoGroupsOmitsClaims(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "no-groups")
	_, hasGroups := idClaims["groups"]
	assert.False(t, hasGroups, "no-groups must not emit a groups claim")
	_, hasRoles := idClaims["roles"]
	assert.False(t, hasRoles, "no-groups must not emit a roles claim")
	_, uiHasGroups := ui["groups"]
	assert.False(t, uiHasGroups, "userinfo for no-groups must not emit groups")
}

// TestPhase7_ManyGroups verifies the many-groups scenario emits all groups.
func TestPhase7_ManyGroups(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, _ := runClaimFlow(t, ts, "many-groups")
	groups, _ := idClaims["groups"].([]any)
	assert.Len(t, groups, 8, "many-groups should emit 8 groups")
}

// TestPhase7_TenantClaims verifies tenant-a-admin and tenant-b-viewer carry
// the right tenant and group, with distinct tenants.
func TestPhase7_TenantClaims(t *testing.T) {
	_, ts := newTestServer(t)
	aClaims, aUI := runClaimFlow(t, ts, "tenant-a-admin")
	bClaims, bUI := runClaimFlow(t, ts, "tenant-b-viewer")

	assert.Equal(t, "tenant-a", aClaims["tenant"])
	assert.Equal(t, []any{"admin"}, aClaims["groups"])
	assert.Equal(t, "tenant-b", bClaims["tenant"])
	assert.Equal(t, []any{"viewer"}, bClaims["groups"])
	// Distinct tenants.
	assert.NotEqual(t, aClaims["tenant"], bClaims["tenant"])
	// Userinfo agrees.
	assert.Equal(t, aClaims["tenant"], aUI["tenant"])
	assert.Equal(t, bClaims["tenant"], bUI["tenant"])
}

// TestPhase7_UnicodeName verifies the unicode-name scenario overrides the
// display name and sets locale, and that userinfo carries the same.
func TestPhase7_UnicodeName(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "unicode-name")
	assert.Equal(t, "Müller Frédéric", idClaims["name"], "unicode name must override the derived name")
	assert.Equal(t, "de-DE", idClaims["locale"])
	assert.Equal(t, "Müller Frédéric", ui["name"], "userinfo must carry the unicode name")
	assert.Equal(t, "de-DE", ui["locale"])
}

// TestPhase7_NoEmailDeletesClaims verifies the no-email scenario omits email
// and email_verified from BOTH the ID token and userinfo (distinct from the
// Phase 4 id-missing-email mutation, which only affects the ID token).
func TestPhase7_NoEmailDeletesClaims(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "no-email")
	_, idHasEmail := idClaims["email"]
	assert.False(t, idHasEmail, "no-email ID token must omit email")
	_, idHasVerified := idClaims["email_verified"]
	assert.False(t, idHasVerified, "no-email ID token must omit email_verified")
	_, uiHasEmail := ui["email"]
	assert.False(t, uiHasEmail, "no-email userinfo must omit email")
}

// TestPhase7_UnverifiedEmail verifies the unverified-email scenario sets
// email_verified = false in both the ID token and userinfo.
func TestPhase7_UnverifiedEmail(t *testing.T) {
	_, ts := newTestServer(t)
	idClaims, ui := runClaimFlow(t, ts, "unverified-email")
	assert.Equal(t, false, idClaims["email_verified"], "ID token email_verified must be false")
	assert.Equal(t, false, ui["email_verified"], "userinfo email_verified must be false")
}
