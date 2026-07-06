package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
)

func TestDeviceUserCodeHelpers(t *testing.T) {
	code := generateUserCode()
	if len(code) != 9 || code[4] != '-' {
		t.Fatalf("user code format = %q", code)
	}
	for _, r := range strings.ReplaceAll(code, "-", "") {
		if !strings.ContainsRune(userCodeAlphabet, r) {
			t.Fatalf("user code contains unexpected rune %q in %q", r, code)
		}
	}
	if got := normalizeUserCode(" abcd efgh "); got != "ABCDEFGH" {
		t.Fatalf("normalize = %q", got)
	}
	if got := displayUserCode("abcd efgh"); got != "ABCD-EFGH" {
		t.Fatalf("display = %q", got)
	}
}

func TestDeviceDiscoveryAndAuthorizationEndpoint(t *testing.T) {
	_, ts := newTestServer(t)

	resp, err := ts.Client().Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	var discovery map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		resp.Body.Close()
		t.Fatalf("decode discovery: %v", err)
	}
	resp.Body.Close()
	if discovery["device_authorization_endpoint"] != ts.URL+"/device_authorization" {
		t.Fatalf("device endpoint = %v", discovery["device_authorization_endpoint"])
	}
	grants, ok := discovery["grant_types_supported"].([]any)
	if !ok || !containsAny(grants, deviceGrantType) {
		t.Fatalf("grant_types_supported missing device grant: %#v", discovery["grant_types_supported"])
	}

	body := deviceAuthorization(t, ts, "dev-client", "openid profile email")
	if body["device_code"] == "" || body["user_code"] == "" {
		t.Fatalf("missing device/user code: %#v", body)
	}
	if body["verification_uri"] != ts.URL+"/device" {
		t.Fatalf("verification_uri = %#v", body["verification_uri"])
	}
	if !strings.HasPrefix(body["verification_uri_complete"].(string), ts.URL+"/device?user_code=") {
		t.Fatalf("verification_uri_complete = %#v", body["verification_uri_complete"])
	}
	if body["expires_in"].(float64) != 600 || body["interval"].(float64) != 5 {
		t.Fatalf("unexpected timing fields: %#v", body)
	}
}

func TestDeviceAuthorizationRejectsUnknownClientAndScope(t *testing.T) {
	_, ts := newTestServer(t)

	resp := postForm(t, ts, "/device_authorization", url.Values{
		"client_id": {"missing-client"},
		"scope":     {"openid"},
	})
	assertOAuthError(t, resp, http.StatusBadRequest, "invalid_client")

	resp = postForm(t, ts, "/device_authorization", url.Values{
		"client_id": {"public-spa"},
		"scope":     {"profile email"},
	})
	assertOAuthError(t, resp, http.StatusBadRequest, "invalid_scope")
}

func TestDeviceTokenPollingApprovalAndOneTimeUse(t *testing.T) {
	s, ts := newTestServer(t)
	seeded, err := scenario.SeededUsersToScenarios([]scenario.SeededUser{
		{
			Login:             "alice",
			Password:          "alice-password",
			Sub:               "user-alice-fixed",
			Email:             "alice@example.test",
			Name:              "Alice Inbox",
			Groups:            []string{"inbox-users"},
			Roles:             []string{"writer"},
			Tenant:            "personal",
			PreferredUsername: "alice",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.registry.RegisterAll(seeded)

	started := deviceAuthorization(t, ts, "dev-client", "openid profile email offline_access")
	deviceCode := started["device_code"].(string)
	userCode := started["user_code"].(string)

	pending := pollDeviceToken(t, ts, deviceCode, "dev-client")
	assertOAuthError(t, pending, http.StatusBadRequest, "authorization_pending")

	slow := pollDeviceToken(t, ts, deviceCode, "dev-client")
	assertOAuthError(t, slow, http.StatusBadRequest, "slow_down")

	approveDevice(t, ts, userCode, "alice", "wrong-password", "invalid login or password")
	s.mu.Lock()
	grant := s.deviceGrants[deviceCode]
	if grant.Status != devicePending {
		t.Fatalf("wrong password changed status to %s", grant.Status)
	}
	grant.LastPoll = time.Now().Add(-10 * time.Second)
	s.deviceGrants[deviceCode] = grant
	s.mu.Unlock()

	approveDevice(t, ts, userCode, "alice", "alice-password", "Device request approved")
	s.mu.Lock()
	grant = s.deviceGrants[deviceCode]
	grant.LastPoll = time.Now().Add(-10 * time.Second)
	s.deviceGrants[deviceCode] = grant
	s.mu.Unlock()

	resp := pollDeviceToken(t, ts, deviceCode, "dev-client")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("approved poll status = %d: %s", resp.StatusCode, body)
	}
	var token map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	if token["access_token"] == "" || token["id_token"] == "" || token["refresh_token"] == "" {
		t.Fatalf("missing tokens: %#v", token)
	}
	claims := verifyIDTokenSignature(t, ts, token["id_token"].(string))
	if claims["sub"] != "user-alice-fixed" || claims["tenant"] != "personal" {
		t.Fatalf("unexpected ID token claims: %#v", claims)
	}

	reuse := pollDeviceToken(t, ts, deviceCode, "dev-client")
	assertOAuthError(t, reuse, http.StatusBadRequest, "invalid_grant")
}

func TestDeviceDeniedExpiredClientMismatchAndPathRoutes(t *testing.T) {
	s, ts := newTestServer(t)

	denied := deviceAuthorization(t, ts, "dev-client", "openid profile email")
	approveDenyDevice(t, ts, denied["user_code"].(string))
	resp := pollDeviceToken(t, ts, denied["device_code"].(string), "dev-client")
	assertOAuthError(t, resp, http.StatusBadRequest, "access_denied")

	expired := deviceAuthorization(t, ts, "dev-client", "openid profile email")
	expiredCode := expired["device_code"].(string)
	s.mu.Lock()
	grant := s.deviceGrants[expiredCode]
	grant.Expires = time.Now().Add(-time.Minute)
	s.deviceGrants[expiredCode] = grant
	s.mu.Unlock()
	resp = pollDeviceToken(t, ts, expiredCode, "dev-client")
	assertOAuthError(t, resp, http.StatusBadRequest, "expired_token")

	mismatch := deviceAuthorization(t, ts, "dev-client", "openid profile email")
	resp = pollDeviceToken(t, ts, mismatch["device_code"].(string), "public-spa")
	assertOAuthError(t, resp, http.StatusBadRequest, "invalid_grant")

	const prefix = "/realms/device-test"
	ps, err := New(Options{Issuer: "http://issuer.test" + prefix})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	ps.RegisterRoutes(mux)
	pts := httptest.NewServer(WithCORS(mux))
	t.Cleanup(pts.Close)
	ps.issuer = pts.URL + prefix
	prefixed := postFormURL(t, pts.URL+prefix+"/device_authorization", url.Values{
		"client_id": {"dev-client"},
		"scope":     {"openid profile email"},
	})
	defer prefixed.Body.Close()
	if prefixed.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(prefixed.Body)
		t.Fatalf("prefixed device authorization status = %d: %s", prefixed.StatusCode, body)
	}
	form, err := pts.Client().Get(pts.URL + prefix + "/device")
	if err != nil {
		t.Fatalf("prefixed device GET: %v", err)
	}
	body, _ := io.ReadAll(form.Body)
	form.Body.Close()
	if form.StatusCode != http.StatusOK || !strings.Contains(string(body), "Tiny IdP Device Approval") {
		t.Fatalf("prefixed device form = %d %q", form.StatusCode, body)
	}
}

func containsAny(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func deviceAuthorization(t *testing.T, ts *httptest.Server, clientID, scope string) map[string]any {
	t.Helper()
	resp := postFormURL(t, ts.URL+"/device_authorization", url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("device authorization status = %d: %s", resp.StatusCode, body)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode device authorization: %v", err)
	}
	return out
}

func postForm(t *testing.T, ts *httptest.Server, path string, form url.Values) *http.Response {
	t.Helper()
	return postFormURL(t, ts.URL+path, form)
}

func postFormURL(t *testing.T, target string, form url.Values) *http.Response {
	t.Helper()
	resp, err := http.PostForm(target, form)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	return resp
}

func assertOAuthError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode oauth error: %v", err)
	}
	if resp.StatusCode != wantStatus || body["error"] != wantCode {
		t.Fatalf("oauth error status/code = %d/%v, want %d/%s body=%#v", resp.StatusCode, body["error"], wantStatus, wantCode, body)
	}
}

func pollDeviceToken(t *testing.T, ts *httptest.Server, deviceCode, clientID string) *http.Response {
	t.Helper()
	return postForm(t, ts, "/token", url.Values{
		"grant_type":  {deviceGrantType},
		"client_id":   {clientID},
		"device_code": {deviceCode},
	})
}

func approveDevice(t *testing.T, ts *httptest.Server, userCode, login, password, wantText string) {
	t.Helper()
	resp := postForm(t, ts, "/device", url.Values{
		"user_code": {userCode},
		"login":     {login},
		"password":  {password},
		"action":    {"approve"},
	})
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), wantText) {
		t.Fatalf("approve response = %d %q, want text %q", resp.StatusCode, body, wantText)
	}
}

func approveDenyDevice(t *testing.T, ts *httptest.Server, userCode string) {
	t.Helper()
	resp := postForm(t, ts, "/device", url.Values{
		"user_code": {userCode},
		"action":    {"deny"},
	})
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Device request denied") {
		t.Fatalf("deny response = %d %q", resp.StatusCode, body)
	}
}
