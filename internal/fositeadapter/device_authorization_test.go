package fositeadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idppolicy"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestDeviceAuthorizationCreatesHashedGrantAndNoStoreResponse(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, sink := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	response, err := http.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}, "resource": {"https://inbox.example.test/api"}})
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
	if grant.Status != idpstore.DeviceGrantPending || strings.Contains(grant.ID, body.DeviceCode) || !containsScope(grant.RequestedScopes, "openid") || !grantAudienceContains(grant.RequestedAudiences, "https://inbox.example.test/api") || grant.Version != 1 {
		t.Fatalf("stored grant = %#v", grant)
	}
	for _, event := range sink.Events() {
		serialized := fmt.Sprintf("%#v", event)
		if strings.Contains(serialized, body.DeviceCode) || strings.Contains(serialized, body.UserCode) {
			t.Fatalf("audit event leaked device material: %#v", event)
		}
	}
}

func TestDeviceAuthorizationAudienceCompatibilityRejectsAmbiguousInputs(t *testing.T) {
	resource := "https://inbox.example.test/api"
	got, err := deviceAuthorizationAudiences(url.Values{"resource": {resource}})
	if err != nil || len(got) != 1 || got[0] != resource {
		t.Fatalf("RFC 8707 resource = %#v, %v", got, err)
	}
	got, err = deviceAuthorizationAudiences(url.Values{"audience": {resource}})
	if err != nil || len(got) != 1 || got[0] != resource {
		t.Fatalf("legacy audience = %#v, %v", got, err)
	}
	if _, err := deviceAuthorizationAudiences(url.Values{"resource": {resource}, "audience": {resource}}); err == nil {
		t.Fatal("combined resource and audience accepted")
	}
	if _, err := deviceAuthorizationAudiences(url.Values{"resource": {"relative"}}); err == nil {
		t.Fatal("relative resource accepted")
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
		{name: "invalid audience", method: http.MethodPost, contentType: "application/x-www-form-urlencoded", body: "client_id=device-cli&scope=openid&audience=https%3A%2F%2Fother.example.test%2Fapi", wantCode: "invalid_target"},
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

func TestDeviceVerificationApprovesADeviceGrantWithFreshPasswordAndOneUse(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, sink := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}, "audience": {"https://inbox.example.test/api"}})
	if err != nil {
		t.Fatal(err)
	}
	var response deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	confirmation := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+url.QueryEscape(response.UserCode), http.StatusOK)
	if strings.Contains(confirmation, response.DeviceCode) || !strings.Contains(confirmation, "device-cli") {
		t.Fatalf("confirmation leaked device material or omitted client: %q", confirmation)
	}
	values := deviceVerificationHiddenFields(t, confirmation)
	values.Set(idpui.ActionFieldName, string(idpui.ActionApprove))
	values.Set(idpui.LoginFieldName, "alice")
	values.Set(idpui.PasswordFieldName, "password")
	decision, err := client.PostForm(server.URL+"/device", values)
	if err != nil {
		t.Fatal(err)
	}
	body := readAndClose(t, decision)
	if decision.StatusCode != http.StatusOK || !strings.Contains(body, "approved") || decision.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("approval response = %d %q", decision.StatusCode, body)
	}
	grant, err := store.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), response.DeviceCode), "device-cli")
	if err != nil {
		t.Fatal(err)
	}
	if grant.Status != idpstore.DeviceGrantApproved || grant.Subject != "user-alice" || grant.AuthTime != now || !containsScope(grant.ApprovedScopes, "profile") || !grantAudienceContains(grant.ApprovedAudiences, "https://inbox.example.test/api") {
		t.Fatalf("approved grant = %#v", grant)
	}
	replay, err := client.PostForm(server.URL+"/device", values)
	if err != nil {
		t.Fatal(err)
	}
	if replay.StatusCode != http.StatusBadRequest {
		t.Fatalf("replay status = %d", replay.StatusCode)
	}
	_ = replay.Body.Close()
	if cookieNamed(decision.Cookies(), provider.sessionCookieName) {
		t.Fatal("device verification unexpectedly created a browser session")
	}
	for _, event := range sink.Events() {
		if event.Name == "device.verification.approved" && event.ClientID == "device-cli" {
			return
		}
	}
	t.Fatal("approval audit event not recorded")
}

func TestDeviceVerificationRequiresBoundCSRFAndKeepsInvalidCredentialsRetryable(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	var response deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	page := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+response.UserCode, http.StatusOK)
	values := deviceVerificationHiddenFields(t, page)
	values.Set(idpui.ActionFieldName, string(idpui.ActionDeny))
	values.Set(idpui.LoginFieldName, "alice")
	values.Set(idpui.PasswordFieldName, "wrong")
	wrongPassword, err := client.PostForm(server.URL+"/device", values)
	if err != nil {
		t.Fatal(err)
	}
	if wrongPassword.StatusCode != http.StatusUnauthorized || !strings.Contains(readAndClose(t, wrongPassword), "Invalid login or password") {
		t.Fatalf("wrong-password response = %d", wrongPassword.StatusCode)
	}
	grant, err := store.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), response.DeviceCode), "device-cli")
	if err != nil || grant.Status != idpstore.DeviceGrantPending {
		t.Fatalf("wrong-password altered grant = %#v %v", grant, err)
	}
	otherBrowser, err := http.PostForm(server.URL+"/device", values)
	if err != nil {
		t.Fatal(err)
	}
	if otherBrowser.StatusCode != http.StatusBadRequest {
		t.Fatalf("cross-browser submission status = %d", otherBrowser.StatusCode)
	}
	_ = otherBrowser.Body.Close()
	values.Set(idpui.PasswordFieldName, "password")
	denied, err := client.PostForm(server.URL+"/device", values)
	if err != nil {
		t.Fatal(err)
	}
	if denied.StatusCode != http.StatusOK || !strings.Contains(readAndClose(t, denied), "denied") {
		t.Fatalf("denial response = %d", denied.StatusCode)
	}
	grant, err = store.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), response.DeviceCode), "device-cli")
	if err != nil || grant.Status != idpstore.DeviceGrantDenied {
		t.Fatalf("denied grant = %#v %v", grant, err)
	}
}

func TestDeviceVerificationDoesNotRevealUnknownOrTerminalCodeState(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	var response deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	if _, err := store.DecideDeviceGrant(context.Background(), idpstore.DeviceDecisionRequest{UserCodeHash: userCodeHash([]byte("device-auth-test-secret-key-32-bytes"), response.UserCode), Decision: idpstore.DeviceGrantDeny, Now: now}); err != nil {
		t.Fatal(err)
	}
	unknown := getDeviceVerificationPage(t, client, server.URL+"/device?user_code=JKLM-NPQR", http.StatusBadRequest)
	terminal := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+response.UserCode, http.StatusBadRequest)
	if unknown != terminal || strings.Contains(terminal, "device-cli") || strings.Contains(terminal, response.UserCode) {
		t.Fatalf("device-code state oracle: unknown=%q terminal=%q", unknown, terminal)
	}
}

func TestDeviceVerificationDecisionRaceHasOneTerminalWinner(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, store, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid"}})
	if err != nil {
		t.Fatal(err)
	}
	var response deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	page := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+response.UserCode, http.StatusOK)
	values := deviceVerificationHiddenFields(t, page)
	values.Set(idpui.ActionFieldName, string(idpui.ActionApprove))
	values.Set(idpui.LoginFieldName, "alice")
	values.Set(idpui.PasswordFieldName, "password")
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cookies := client.Jar.Cookies(serverURL)
	statuses := make(chan int, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			request, err := http.NewRequest(http.MethodPost, server.URL+"/device", strings.NewReader(values.Encode()))
			if err != nil {
				statuses <- 0
				return
			}
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for _, cookie := range cookies {
				request.AddCookie(cookie)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				statuses <- 0
				return
			}
			statuses <- response.StatusCode
			_ = response.Body.Close()
		}()
	}
	group.Wait()
	close(statuses)
	accepted, rejected := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			accepted++
		case http.StatusBadRequest:
			rejected++
		default:
			t.Fatalf("unexpected concurrent status %d", status)
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("decision winners accepted=%d rejected=%d", accepted, rejected)
	}
	grant, err := store.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), response.DeviceCode), "device-cli")
	if err != nil || grant.Status != idpstore.DeviceGrantApproved {
		t.Fatalf("race final grant = %#v %v", grant, err)
	}
}

func TestDeviceVerificationRendererFailureFailsClosed(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, _, sink := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	provider.deviceVerificationUI = failingDeviceVerificationRenderer{}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	response, err := http.Get(server.URL + "/device")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("renderer failure status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
	for _, event := range sink.Events() {
		if event.Name == "device.verification.render_failed" && event.Reason == "renderer_failed" {
			return
		}
	}
	t.Fatal("renderer failure was not audited")
}

func TestDeviceVerificationPresentationPolicyDecoratesNativeConfirmation(t *testing.T) {
	now := time.Date(2026, 7, 15, 17, 0, 0, 0, time.UTC)
	provider, _, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-presentation", "WXYZ-ABCD", nil }, now)
	provider.presentation = devicePresentationTitlePolicy{}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}})
	if err != nil {
		t.Fatal(err)
	}
	var device deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&device); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	page := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+device.UserCode, http.StatusOK)
	if !strings.Contains(page, "Review coding-agent access") {
		t.Fatalf("device presentation title missing from native confirmation: %s", page)
	}
}

func TestDeviceTokenExchangeIssuesOIDCTokensConsumesOnceAndSupportsUserInfo(t *testing.T) {
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	provider, _, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-one", "ABCD-EFGH", nil }, now)
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}})
	if err != nil {
		t.Fatal(err)
	}
	var device deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&device); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	pending := postDeviceToken(t, client, server.URL, device.DeviceCode, "device-cli")
	if pending.StatusCode != http.StatusBadRequest || tokenErrorCode(t, pending) != "authorization_pending" {
		t.Fatalf("pending token response = %d", pending.StatusCode)
	}
	// A pending poll advances its durable next-poll time. Exercise the successful
	// redemption on an independent grant so this test does not bypass that
	// protocol rule with test-only store mutation.
	readyProvider, readyStore, readySink := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-ready", "JKLM-NPQR", nil }, now)
	readyServer := httptest.NewServer(readyProvider.Handler())
	defer readyServer.Close()
	readyClient := newDeviceVerificationHTTPClient(t)
	secondStart, err := readyClient.PostForm(readyServer.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}, "audience": {"https://inbox.example.test/api"}})
	if err != nil {
		t.Fatal(err)
	}
	var readyDevice deviceAuthorizationResponse
	if err := json.NewDecoder(secondStart.Body).Decode(&readyDevice); err != nil {
		t.Fatal(err)
	}
	_ = secondStart.Body.Close()
	readyPage := getDeviceVerificationPage(t, readyClient, readyServer.URL+"/device?user_code="+readyDevice.UserCode, http.StatusOK)
	readyDecision := deviceVerificationHiddenFields(t, readyPage)
	readyDecision.Set(idpui.ActionFieldName, string(idpui.ActionApprove))
	readyDecision.Set(idpui.LoginFieldName, "alice")
	readyDecision.Set(idpui.PasswordFieldName, "password")
	readyApproval, err := readyClient.PostForm(readyServer.URL+"/device", readyDecision)
	if err != nil {
		t.Fatal(err)
	}
	_ = readyApproval.Body.Close()
	token := postDeviceToken(t, readyClient, readyServer.URL, readyDevice.DeviceCode, "device-cli")
	if token.StatusCode != http.StatusOK {
		t.Fatalf("token status = %d body=%q audit=%#v", token.StatusCode, readAndClose(t, token), readySink.Events())
	}
	var body map[string]any
	if err := json.NewDecoder(token.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	_ = token.Body.Close()
	accessToken, _ := body["access_token"].(string)
	if accessToken == "" || body["token_type"] != "bearer" || body["id_token"] == "" || body["scope"] != "openid profile" {
		t.Fatalf("token body = %#v", body)
	}
	introspection, err := http.NewRequest(http.MethodPost, readyServer.URL+"/introspect", strings.NewReader(url.Values{"token": {accessToken}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	introspection.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introspection.SetBasicAuth("inbox-api", "inbox-api-secret")
	introspectionHTTPResponse, err := readyClient.Do(introspection)
	if err != nil {
		t.Fatal(err)
	}
	var introspectionBody introspectionResponse
	if err := json.NewDecoder(introspectionHTTPResponse.Body).Decode(&introspectionBody); err != nil {
		t.Fatal(err)
	}
	_ = introspectionHTTPResponse.Body.Close()
	if introspectionHTTPResponse.StatusCode != http.StatusOK || !introspectionBody.Active || !grantAudienceContains(introspectionBody.Audience, "https://inbox.example.test/api") {
		t.Fatalf("introspection response=%d %#v", introspectionHTTPResponse.StatusCode, introspectionBody)
	}
	userinfo, err := http.NewRequest(http.MethodGet, readyServer.URL+"/userinfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	userinfo.Header.Set("Authorization", "Bearer "+accessToken)
	userResponse, err := readyClient.Do(userinfo)
	if err != nil {
		t.Fatal(err)
	}
	if userResponse.StatusCode != http.StatusOK {
		t.Fatalf("userinfo status = %d body=%q", userResponse.StatusCode, readAndClose(t, userResponse))
	}
	var claims map[string]any
	if err := json.NewDecoder(userResponse.Body).Decode(&claims); err != nil {
		t.Fatal(err)
	}
	_ = userResponse.Body.Close()
	if claims["sub"] != "user-alice" || claims["name"] != "Alice" {
		t.Fatalf("userinfo claims = %#v", claims)
	}
	grant, err := readyStore.InspectDeviceGrantByDeviceCodeHash(context.Background(), deviceCodeHash([]byte("device-auth-test-secret-key-32-bytes"), readyDevice.DeviceCode), "device-cli")
	if err != nil || grant.Status != idpstore.DeviceGrantConsumed {
		t.Fatalf("consumed grant = %#v %v", grant, err)
	}
	replay := postDeviceToken(t, readyClient, readyServer.URL, readyDevice.DeviceCode, "device-cli")
	if replay.StatusCode != http.StatusBadRequest || tokenErrorCode(t, replay) != "invalid_grant" {
		t.Fatalf("replay token response = %d", replay.StatusCode)
	}
}

func TestDeviceClaimsPolicyPersistsAdditionalClaimToUserInfo(t *testing.T) {
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	provider, _, _ := newDeviceAuthorizationProvider(t, func() (string, string, error) { return "device-code-claims", "QRST-UVWX", nil }, now)
	policy, err := idppolicy.New(context.Background(), gojaAdditionalClaimsSource, 1, idppolicy.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = policy.Close(context.Background()) })
	provider.claims = policy
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)

	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}})
	if err != nil {
		t.Fatal(err)
	}
	var device deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&device); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	page := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+device.UserCode, http.StatusOK)
	decision := deviceVerificationHiddenFields(t, page)
	decision.Set(idpui.ActionFieldName, string(idpui.ActionApprove))
	decision.Set(idpui.LoginFieldName, "alice")
	decision.Set(idpui.PasswordFieldName, "password")
	approval, err := client.PostForm(server.URL+"/device", decision)
	if err != nil {
		t.Fatal(err)
	}
	_ = approval.Body.Close()
	token := postDeviceToken(t, client, server.URL, device.DeviceCode, "device-cli")
	if token.StatusCode != http.StatusOK {
		t.Fatalf("token status=%d body=%q", token.StatusCode, readAndClose(t, token))
	}
	var tokens map[string]any
	if err := json.NewDecoder(token.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}
	_ = token.Body.Close()
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("token response=%#v", tokens)
	}
	userinfo, err := http.NewRequest(http.MethodGet, server.URL+"/userinfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	userinfo.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.Do(userinfo)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("userinfo status=%d body=%q", response.StatusCode, readAndClose(t, response))
	}
	var claims map[string]any
	if err := json.NewDecoder(response.Body).Decode(&claims); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if claims["sub"] != "user-alice" || claims["community_role"] != "member" {
		t.Fatalf("userinfo claims=%#v", claims)
	}
}

const gojaAdditionalClaimsSource = `
const A = require("tinyidp").v1;
module.exports = A.program("claims-policy", p => {
  const additional = A.lambda("claims.additional", {
    kind:"provider", input:"claimsInput", output:"claimsOutput",
    outcomes:["complete"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:8192,
    run: ctx => A.result.complete({Additional:{community_role:"member"}})
  });
  p.provider("claims", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{additional}});
});`

func TestSQLiteDeviceBrowserApprovalTokenUserInfoAndReplay(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	secret := []byte("sqlite-device-browser-flow-secret-key")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-cli", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid", "profile"}, AllowedAudiences: []string{"https://inbox.example.test/api"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("sqlite-device-browser-key", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := NewProvider(ctx, Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: secret, Clock: func() time.Time { return now }, Authenticator: deviceTestAuthenticator{}, deviceCodeGenerator: func() (string, string, error) { return "sqlite-browser-device-code", "JKLM-NPQR", nil }})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	client := newDeviceVerificationHTTPClient(t)
	client.Timeout = 2 * time.Second
	start, err := client.PostForm(server.URL+"/device_authorization", url.Values{"client_id": {"device-cli"}, "scope": {"openid profile"}})
	if err != nil {
		t.Fatal(err)
	}
	var device deviceAuthorizationResponse
	if err := json.NewDecoder(start.Body).Decode(&device); err != nil {
		t.Fatal(err)
	}
	_ = start.Body.Close()
	page := getDeviceVerificationPage(t, client, server.URL+"/device?user_code="+device.UserCode, http.StatusOK)
	decision := deviceVerificationHiddenFields(t, page)
	decision.Set(idpui.ActionFieldName, string(idpui.ActionApprove))
	decision.Set(idpui.LoginFieldName, "alice")
	decision.Set(idpui.PasswordFieldName, "password")
	approved, err := client.PostForm(server.URL+"/device", decision)
	if err != nil {
		t.Fatal(err)
	}
	if approved.StatusCode != http.StatusOK {
		t.Fatalf("approval status=%d body=%q", approved.StatusCode, readAndClose(t, approved))
	}
	_ = approved.Body.Close()
	token := postDeviceToken(t, client, server.URL, device.DeviceCode, "device-cli")
	if token.StatusCode != http.StatusOK {
		t.Fatalf("token status=%d body=%q", token.StatusCode, readAndClose(t, token))
	}
	var tokens map[string]any
	if err := json.NewDecoder(token.Body).Decode(&tokens); err != nil {
		t.Fatal(err)
	}
	_ = token.Body.Close()
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" || tokens["id_token"] == "" {
		t.Fatalf("token response=%#v", tokens)
	}
	userinfo, err := http.NewRequest(http.MethodGet, server.URL+"/userinfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	userinfo.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := client.Do(userinfo)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("userinfo status=%d body=%q", response.StatusCode, readAndClose(t, response))
	}
	_ = response.Body.Close()
	replay := postDeviceToken(t, client, server.URL, device.DeviceCode, "device-cli")
	if replay.StatusCode != http.StatusBadRequest || tokenErrorCode(t, replay) != "invalid_grant" {
		t.Fatalf("replay response=%d", replay.StatusCode)
	}
}

func newDeviceAuthorizationProvider(t *testing.T, generator func() (string, string, error), now time.Time) (*Provider, *memory.Store, *idp.MemorySink) {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-cli", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid", "profile"}, AllowedAudiences: []string{"https://inbox.example.test/api"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(ctx, idpstore.Client{ID: "browser-only", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	resourceSecret, err := bcrypt.GenerateFromPassword([]byte("inbox-api-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(ctx, idpstore.Client{ID: "inbox-api", SecretHash: resourceSecret, AllowedAudiences: []string{"https://inbox.example.test/api"}, CanIntrospect: true, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Name: "Alice"}); err != nil {
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
	provider, err := NewProvider(ctx, Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("device-auth-test-secret-key-32-bytes"), Clock: func() time.Time { return now }, Audit: sink, Authenticator: deviceTestAuthenticator{}, ClientSecrets: map[string]string{"inbox-api": "inbox-api-secret"}, deviceCodeGenerator: generator})
	if err != nil {
		t.Fatal(err)
	}
	return provider, store, sink
}

type deviceTestAuthenticator struct{}

type devicePresentationTitlePolicy struct{}

func (devicePresentationTitlePolicy) Present(_ context.Context, input idp.PresentationInput) (idp.PresentationOutput, error) {
	if input.Kind != idp.PresentationDeviceVerify || input.ClientID != "device-cli" {
		return idp.PresentationOutput{}, fmt.Errorf("unexpected presentation input: %#v", input)
	}
	return idp.PresentationOutput{DocumentTitle: "Review coding-agent access"}, nil
}

type failingDeviceVerificationRenderer struct{}

func (failingDeviceVerificationRenderer) RenderDeviceVerification(context.Context, io.Writer, idpui.DeviceVerificationPage) error {
	return fmt.Errorf("synthetic device renderer failure")
}

func (deviceTestAuthenticator) AuthenticatePassword(_ context.Context, login, password string, _ idp.LoginMetadata) (idp.AuthResult, error) {
	if login != "alice" || password != "password" {
		return idp.AuthResult{}, idpaccounts.ErrInvalidCredentials
	}
	return idp.AuthResult{User: idpstore.User{ID: "u1", Sub: "user-alice", Name: "Alice"}, AMR: []string{"pwd"}}, nil
}

func newDeviceVerificationHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar}
}

func getDeviceVerificationPage(t *testing.T, client *http.Client, rawURL string, wantStatus int) string {
	t.Helper()
	response, err := client.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	body := readAndClose(t, response)
	if response.StatusCode != wantStatus {
		t.Fatalf("GET %s status=%d want=%d body=%q", rawURL, response.StatusCode, wantStatus, body)
	}
	return body
}

func deviceVerificationHiddenFields(t *testing.T, body string) url.Values {
	t.Helper()
	values := make(url.Values)
	for _, match := range regexp.MustCompile(`<input type="hidden" name="([^"]+)" value="([^"]+)">`).FindAllStringSubmatch(body, -1) {
		values.Set(match[1], match[2])
	}
	if values.Get(idpui.InteractionFieldName) == "" || values.Get(idpui.CSRFFieldName) == "" {
		t.Fatalf("verification page did not contain interaction and CSRF fields: %q", body)
	}
	return values
}

func readAndClose(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return string(body)
}

func cookieNamed(cookies []*http.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return true
		}
	}
	return false
}

func postDeviceToken(t *testing.T, client *http.Client, serverURL, deviceCode, clientID string) *http.Response {
	t.Helper()
	response, err := client.PostForm(serverURL+"/token", url.Values{"grant_type": {idpstore.GrantDeviceCode}, "device_code": {deviceCode}, "client_id": {clientID}})
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func tokenErrorCode(t *testing.T, response *http.Response) string {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	code, _ := body["error"].(string)
	return code
}

func grantAudienceContains(audiences []string, want string) bool {
	for _, audience := range audiences {
		if audience == want {
			return true
		}
	}
	return false
}
