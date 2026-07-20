package fositeadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/fositeadapter"
	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idppolicy"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestAuthorizeRequiresCSRFAndEmitsAudit(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	sink := idp.NewMemorySink()
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("hardening-secret-key-32-bytes!!!!"), Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
	resp, err := http.Post(ts.URL+"/authorize", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	found := false
	for _, e := range sink.Events() {
		if e.Name == "login.csrf_rejected" {
			found = true
		}
	}
	if !found {
		t.Fatalf("csrf audit event not found: %#v", sink.Events())
	}
}

func TestAuthorizationPolicyDeniesAfterNativeValidationBeforeCodeIssuance(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	require.NoError(t, st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}))
	require.NoError(t, st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Roles: []string{"member"}}))
	key, err := keys.GenerateRSA("kid-1", time.Now())
	require.NoError(t, err)
	require.NoError(t, st.CreateSigningKey(ctx, key))
	sink := idp.NewMemorySink()
	p, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("authorization-policy-secret-key-32-b"), Audit: sink, Authorization: denyAuthorizationPolicy{}})
	require.NoError(t, err)
	ts := httptest.NewServer(p.Handler())
	t.Cleanup(ts.Close)

	form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	require.NoError(t, err)
	response.Body.Close()
	require.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, response.StatusCode)
	redirect, err := url.Parse(response.Header.Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "access_denied", redirect.Query().Get("error"))
	assert.Empty(t, redirect.Query().Get("code"))

	events := sink.Events()
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, "authorization.policy.denied", last.Name)
	assert.Equal(t, "policy.member_required", last.Reason)
}

func TestGojaAuthorizationProviderDeniesAfterNativeValidationBeforeCodeIssuance(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	require.NoError(t, st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}))
	require.NoError(t, st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}))
	key, err := keys.GenerateRSA("kid-1", time.Now())
	require.NoError(t, err)
	require.NoError(t, st.CreateSigningKey(ctx, key))
	policy, err := idppolicy.New(ctx, gojaDenyAuthorizationSource, 1, idppolicy.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, policy.Close(context.Background())) })
	p, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("goja-authorization-policy-secret-32"), Authorization: policy})
	require.NoError(t, err)
	ts := httptest.NewServer(p.Handler())
	t.Cleanup(ts.Close)

	form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	request, err := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(csrfCookie)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	require.NoError(t, err)
	response.Body.Close()
	require.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, response.StatusCode)
	redirect, err := url.Parse(response.Header.Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "access_denied", redirect.Query().Get("error"))
	assert.Empty(t, redirect.Query().Get("code"))
}

func TestAuthorizationPolicyDeniesPromptNoneWithExistingSessionBeforeCodeIssuance(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	require.NoError(t, st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}))
	require.NoError(t, st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}))
	key, err := keys.GenerateRSA("kid-1", time.Now())
	require.NoError(t, err)
	require.NoError(t, st.CreateSigningKey(ctx, key))
	p, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("authorization-policy-prompt-none-32"), Authorization: denyPromptNoneAuthorizationPolicy{}})
	require.NoError(t, err)
	ts := httptest.NewServer(p.Handler())
	t.Cleanup(ts.Close)

	form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
	csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
	form.Set("csrf_token", csrfToken)
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(csrfCookie)
	response, err := noRedirect.Do(request)
	require.NoError(t, err)
	response.Body.Close()
	require.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, response.StatusCode)
	var sessionCookie *http.Cookie
	for _, cookie := range response.Cookies() {
		if cookie.Name == "tinyidp_session" {
			sessionCookie = cookie
		}
	}
	require.NotNil(t, sessionCookie)

	silent := url.Values{}
	for key, values := range form {
		silent[key] = append([]string(nil), values...)
	}
	silent.Del("login")
	silent.Del("csrf_token")
	silent.Set("prompt", "none")
	silent.Set("state", "state-prompt-none-policy")
	request, err = http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+silent.Encode(), nil)
	require.NoError(t, err)
	request.AddCookie(sessionCookie)
	response, err = noRedirect.Do(request)
	require.NoError(t, err)
	response.Body.Close()
	require.Contains(t, []int{http.StatusFound, http.StatusSeeOther}, response.StatusCode)
	redirect, err := url.Parse(response.Header.Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "access_denied", redirect.Query().Get("error"))
	assert.Empty(t, redirect.Query().Get("code"))
}

func TestAuthorizationAndClaimsPolicyFailuresDoNotIssueCode(t *testing.T) {
	for name, options := range map[string]fositeadapter.Options{
		"authorization": {Authorization: failingAuthorizationPolicy{}},
		"claims":        {Claims: failingClaimsPolicy{}},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			st := memory.New()
			require.NoError(t, st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}))
			require.NoError(t, st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}))
			key, err := keys.GenerateRSA("kid-1", time.Now())
			require.NoError(t, err)
			require.NoError(t, st.CreateSigningKey(ctx, key))
			options.Issuer = "http://127.0.0.1:5556"
			options.Store = st
			options.SecretKey = []byte("authorization-claims-failure-key-32")
			p, err := fositeadapter.NewProvider(ctx, options)
			require.NoError(t, err)
			ts := httptest.NewServer(p.Handler())
			t.Cleanup(ts.Close)

			form := url.Values{"response_type": {"code"}, "client_id": {"spa"}, "redirect_uri": {"http://localhost/callback"}, "scope": {"openid"}, "state": {"state-1234567890"}, "nonce": {"nonce-1234567890"}, "code_challenge": {s256("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")}, "code_challenge_method": {"S256"}, "login": {"alice"}}
			csrfToken, csrfCookie := fetchCSRF(t, ts.URL, form)
			form.Set("csrf_token", csrfToken)
			request, err := http.NewRequest(http.MethodPost, ts.URL+"/authorize", strings.NewReader(form.Encode()))
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			request.AddCookie(csrfCookie)
			response, err := http.DefaultClient.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()
			assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
			assert.Empty(t, response.Header.Get("Location"))
		})
	}
}

type denyAuthorizationPolicy struct{}

func (denyAuthorizationPolicy) Authorize(context.Context, idp.AuthorizationInput) (idp.AuthorizationDecision, error) {
	return idp.AuthorizationDecision{Kind: idp.AuthorizationDeny, DiagnosticID: "policy.member_required"}, nil
}

const gojaDenyAuthorizationSource = `
const A = require("tinyidp").v1;
module.exports = A.program("authorization-policy", p => {
  const decide = A.lambda("authorization.decide", {
    kind:"provider", input:"authorizationInput", output:"authorizationOutput",
    outcomes:["complete"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:8192,
    run: ctx => A.result.complete({Kind:"deny", DiagnosticID:"policy.member_required"})
  });
  p.provider("authorization", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{decide}});
});`

type denyPromptNoneAuthorizationPolicy struct{}

func (denyPromptNoneAuthorizationPolicy) Authorize(_ context.Context, input idp.AuthorizationInput) (idp.AuthorizationDecision, error) {
	for _, prompt := range input.Request.Prompt {
		if prompt == "none" {
			return idp.AuthorizationDecision{Kind: idp.AuthorizationDeny, DiagnosticID: "policy.silent_access_denied"}, nil
		}
	}
	return idp.AuthorizationDecision{Kind: idp.AuthorizationAllow}, nil
}

type failingAuthorizationPolicy struct{}

func (failingAuthorizationPolicy) Authorize(context.Context, idp.AuthorizationInput) (idp.AuthorizationDecision, error) {
	return idp.AuthorizationDecision{}, errors.New("synthetic authorization policy failure")
}

type failingClaimsPolicy struct{}

func (failingClaimsPolicy) Claims(context.Context, idp.ClaimsInput) (idp.ClaimsOutput, error) {
	return idp.ClaimsOutput{}, errors.New("synthetic claims policy failure")
}

func TestAuditReasonsUseStableCodes(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	sink := idp.NewMemorySink()
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("audit-reason-secret-32-bytes!!!!"), Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.PostForm(ts.URL+"/token", url.Values{"grant_type": {"authorization_code"}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("token request unexpectedly succeeded")
	}
	for _, e := range sink.Events() {
		if e.Name == "token.request.rejected" {
			if e.Reason == "" || strings.Contains(e.Reason, " ") || strings.Contains(e.Reason, "(") {
				t.Fatalf("unstable audit reason: %#v", e)
			}
			return
		}
	}
	t.Fatalf("token rejection audit event not found: %#v", sink.Events())
}

func TestSecurityHeadersOnDiscovery(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	p, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("headers-secret-key-32-bytes!!!!!!")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing frame deny header")
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing nosniff header")
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatalf("missing CSP")
	}
}

func TestDiscoveryPublishesIntrospectionAtRootAndPathIssuer(t *testing.T) {
	for _, tc := range []struct {
		name              string
		issuer            string
		discoveryPaths    []string
		wantIntrospection string
	}{
		{name: "root issuer", issuer: "https://issuer.example.test", discoveryPaths: []string{"/.well-known/openid-configuration"}, wantIntrospection: "https://issuer.example.test/introspect"},
		{name: "path issuer", issuer: "https://issuer.example.test/idp", discoveryPaths: []string{"/.well-known/openid-configuration/idp", "/idp/.well-known/openid-configuration"}, wantIntrospection: "https://issuer.example.test/idp/introspect"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := memory.New()
			key, err := keys.GenerateRSA("kid-discovery", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := st.CreateSigningKey(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			provider, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: tc.issuer, Store: st, SecretKey: []byte("discovery-introspection-key-32-bytes")})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(provider.Handler())
			defer server.Close()
			for _, path := range tc.discoveryPaths {
				response, err := http.Get(server.URL + path)
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != http.StatusOK {
					_ = response.Body.Close()
					t.Fatalf("discovery path %q status=%d", path, response.StatusCode)
				}
				var discovery struct {
					IntrospectionEndpoint       string   `json:"introspection_endpoint"`
					DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
					AuthMethods                 []string `json:"introspection_endpoint_auth_methods_supported"`
					GrantTypes                  []string `json:"grant_types_supported"`
				}
				if err := json.NewDecoder(response.Body).Decode(&discovery); err != nil {
					_ = response.Body.Close()
					t.Fatal(err)
				}
				_ = response.Body.Close()
				wantDeviceAuthorization := strings.TrimSuffix(tc.wantIntrospection, "/introspect") + "/device_authorization"
				if discovery.IntrospectionEndpoint != tc.wantIntrospection || discovery.DeviceAuthorizationEndpoint != wantDeviceAuthorization || len(discovery.AuthMethods) != 1 || discovery.AuthMethods[0] != "client_secret_basic" || !containsString(discovery.GrantTypes, "urn:ietf:params:oauth:grant-type:device_code") {
					t.Fatalf("discovery path %q contract=%#v", path, discovery)
				}
			}
		})
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
