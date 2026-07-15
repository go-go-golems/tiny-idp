package fositeadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestDeviceAuthorizationCreatesHashedGrantAndNoStoreResponse(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, sink := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	response, err := http.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("response status/headers = %d %q %q", response.StatusCode, response.Header.Get("Cache-Control"), response.Header.Get("Pragma"))
	}
	var body deviceAuthorizationResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.DeviceCode != "device-code-one" || body.UserCode != "ABCD-EFGH" || body.VerificationURI != "http://127.0.0.1:5556/device" || body.ExpiresIn != 600 || body.Interval != 5 {
		t.Fatalf("response = %#v", body)
	}
	grant, err := store.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), body.DeviceCode), "device-cli")
	if err != nil {
		t.Fatal(err)
	}
	if grant.Status != idpstore.DeviceGrantPending || strings.Contains(grant.ID, body.DeviceCode) || !containsScope(grant.RequestedScopes, "openid") || grant.Version != 1 {
		t.Fatalf("stored grant = %#v", grant)
	}
	for _, event := range sink.Events() {
		serialized := fmt.Sprintf("%#v", event)
		if strings.Contains(serialized, body.DeviceCode) || strings.Contains(serialized, body.UserCode) {
			t.Fatalf("audit event leaked device material: %#v", event)
		}
	}
}

func TestDeviceAuthorizationRejectsMalformedUnauthorizedAndInvalidScopeRequests(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, _, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	cases := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCode    string
	}{
		{name: "method", method: http.MethodGet, wantCode: "invalid_request"},
		{name: "content type", method: http.MethodPost, contentType: "application/json", body: `{}`, wantCode: "invalid_request"},
		{name: "duplicate client", method: http.MethodPost, contentType: "application/x-www-form-urlencoded", body: "client_id=device-cli&client_id=device-cli&scope=openid", wantCode: "invalid_request"},
		{name: "not device capable", method: http.MethodPost, contentType: "application/x-www-form-urlencoded", body: "client_id=browser-only&scope=openid", wantCode: "unauthorized_client"},
		{name: "invalid scope", method: http.MethodPost, contentType: "application/x-www-form-urlencoded", body: "client_id=device-cli&scope=profile", wantCode: "invalid_scope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(tc.method, server.URL+"/device_authorization", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			var body map[string]string
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["error"] != tc.wantCode || response.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("response = status %d body %#v", response.StatusCode, body)
			}
		})
	}
}

func TestDeviceAuthorizationRetriesHashCollisionsWithinBound(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	calls := 0
	provider, store, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) {
		calls++
		if calls == 1 {
			return "device-code-collision", "ABCD-EFGH", nil
		}
		return "device-code-unique", "JKLM-NPQR", nil
	}, now)
	if err := store.CreateDeviceGrant(context.Background(), idpstore.DeviceGrant{ID: "existing", DeviceCodeHash: deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), "device-code-collision"), UserCodeHash: userCodeHash([]byte("device-auth-test-secret-key-32-bytes"), "ABCD-EFGH"), ClientID: "device-cli", Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour), PollInterval: time.Second, NextPollAt: now}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	response, err := http.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var body deviceAuthorizationResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || calls != 2 || body.DeviceCode != "device-code-unique" {
		t.Fatalf("collision response = %d %#v calls=%d", response.StatusCode, body, calls)
	}
}

func TestDeviceAuthorizationAuthenticatesConfidentialDeviceClients(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "confidential-device-code", "ABCD-EFGH", nil }, now)
	secretHash, err := bcrypt.GenerateFromPassword([]byte("device-client-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(context.Background(), idpstore.Client{ID: "protected-device", SecretHash: secretHash, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/device_authorization", strings.NewReader("scope=openid"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth("protected-device", "wrong")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized || response.Header.Get("WWW-Authenticate") == "" {
		t.Fatalf("wrong-secret response = %d %q", response.StatusCode, response.Header.Get("WWW-Authenticate"))
	}
	_ = response.Body.Close()
	request, err = http.NewRequest(http.MethodPost, server.URL+"/device_authorization", strings.NewReader("scope=openid"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth("protected-device", "device-client-secret")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("correct-secret response = %d", response.StatusCode)
	}
}

func TestDeviceCodeHelpersAreCanonicalAndDomainSeparated(t *testing.T) {
	key := []byte("device-auth-test-secret-key-32-bytes")
	if got := normalizeUserCode(" abcd efgh "); got != "ABCD-EFGH" {
		t.Fatalf("normalized code = %q", got)
	}
	if got := normalizeUserCode("ABCD-0EFG"); got != "" {
		t.Fatalf("ambiguous user code normalized to %q", got)
	}
	if string(deviceCodeHash(key, "same")) == string(userCodeHash(key, "SAME")) {
		t.Fatal("device and user code hash domains overlap")
	}
	for range 20 {
		deviceCode, userCode, err := generateDeviceCodes()
		if err != nil || len(deviceCode) < 40 || normalizeUserCode(userCode) != userCode {
			t.Fatalf("generated codes = %q %q %v", deviceCode, userCode, err)
		}
	}
}

func newDeviceAuthorizationProvider(t *testing.T, generator func() (string, string, error), now time.Time) (*Provider, *memory.Store, *idp.MemorySink) {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-cli", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid", "profile"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(ctx, idpstore.Client{ID: "browser-only", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("device-auth-key", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	sink := idp.NewMemorySink()
	provider, err := NewProvider(ctx, Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("device-auth-test-secret-key-32-bytes"), Clock: func() time.Time { return now }, Audit: sink, deviceCodeGenerator: generator})
	if err != nil {
		t.Fatal(err)
	}
	return provider, store, sink
}
