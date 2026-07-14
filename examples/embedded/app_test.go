package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestRelyingPartyCompletesAuthorizationCodeFlow(t *testing.T) {
	const (
		base   = "http://app.example.test"
		issuer = base + "/idp"
		client = "embedded-example"
	)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	var expectedNonce string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/idp/.well-known/openid-configuration":
			return jsonResponse(request, map[string]any{
				"issuer": issuer, "token_endpoint": issuer + "/token",
				"userinfo_endpoint": issuer + "/userinfo", "jwks_uri": issuer + "/jwks.json",
			}), nil
		case "/idp/token":
			if err := request.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if request.Form.Get("code") != "one-time-code" || request.Form.Get("code_verifier") == "" {
				t.Fatalf("unexpected token request: %v", request.Form)
			}
			return jsonResponse(request, map[string]any{
				"access_token": "access-token", "token_type": "Bearer",
				"id_token": signedIDToken(t, key, expectedNonce, now, issuer, client),
			}), nil
		case "/idp/jwks.json":
			return jsonResponse(request, map[string]any{"keys": []any{publicJWK(&key.PublicKey)}}), nil
		case "/idp/userinfo":
			if request.Header.Get("Authorization") != "Bearer access-token" {
				t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
			}
			return jsonResponse(request, userInfo{Subject: "user-123", Name: "Alice", Email: "alice@example.test"}), nil
		default:
			t.Fatalf("unexpected back-channel path: %s", request.URL.Path)
			return nil, nil
		}
	})
	rp, err := newRelyingParty(rpOptions{
		PublicBaseURL: base, Issuer: issuer, ClientID: client,
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	rp.now = func() time.Time { return now }

	loginResponse := httptest.NewRecorder()
	rp.ServeHTTP(loginResponse, httptest.NewRequest(http.MethodGet, base+"/login", nil))
	if loginResponse.Code != http.StatusFound {
		t.Fatalf("login status = %d", loginResponse.Code)
	}
	authorizeURL, err := url.Parse(loginResponse.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	flowCookie := responseCookie(t, loginResponse.Result(), flowCookieName)
	rp.mu.Lock()
	flow := rp.flows[flowCookie.Value]
	rp.mu.Unlock()
	expectedNonce = flow.Nonce
	if authorizeURL.Query().Get("state") != flow.State || authorizeURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("authorization request was not bound to the stored PKCE flow: %s", authorizeURL)
	}

	callbackURL := base + "/auth/callback?code=one-time-code&state=" + url.QueryEscape(flow.State)
	callbackRequest := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	callbackRequest.AddCookie(flowCookie)
	callbackResponse := httptest.NewRecorder()
	rp.ServeHTTP(callbackResponse, callbackRequest)
	if callbackResponse.Code != http.StatusSeeOther || callbackResponse.Header().Get("Location") != "/" {
		t.Fatalf("callback response = %d %q: %s", callbackResponse.Code, callbackResponse.Header().Get("Location"), callbackResponse.Body.String())
	}
	sessionCookie := responseCookie(t, callbackResponse.Result(), sessionCookieName)

	homeRequest := httptest.NewRequest(http.MethodGet, base+"/", nil)
	homeRequest.AddCookie(sessionCookie)
	homeResponse := httptest.NewRecorder()
	rp.ServeHTTP(homeResponse, homeRequest)
	if homeResponse.Code != http.StatusOK || !strings.Contains(homeResponse.Body.String(), "Alice") || !strings.Contains(homeResponse.Body.String(), "user-123") {
		t.Fatalf("authenticated home page is incomplete: %s", homeResponse.Body.String())
	}
}

func TestCallbackRejectsStateMismatchAndConsumesFlow(t *testing.T) {
	rp, err := newRelyingParty(rpOptions{
		PublicBaseURL: "http://app.example.test", Issuer: "http://app.example.test/idp", ClientID: "client",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("state mismatch must fail before a back-channel request")
			return nil, nil
		})},
	})
	if err != nil {
		t.Fatal(err)
	}
	loginResponse := httptest.NewRecorder()
	rp.ServeHTTP(loginResponse, httptest.NewRequest(http.MethodGet, "http://app.example.test/login", nil))
	flowCookie := responseCookie(t, loginResponse.Result(), flowCookieName)
	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodGet, "http://app.example.test/auth/callback?code=code&state=wrong", nil)
		request.AddCookie(flowCookie)
		response := httptest.NewRecorder()
		rp.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d status = %d", attempt, response.Code)
		}
	}
}

func jsonResponse(request *http.Request, value any) *http.Response {
	contents, _ := json.Marshal(value)
	return &http.Response{
		StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(string(contents))), Request: request,
	}
}

func signedIDToken(t *testing.T, key *rsa.PrivateKey, nonce string, now time.Time, issuer, audience string) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "test-key", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss": issuer, "sub": "user-123", "aud": audience,
		"exp": now.Add(time.Hour).Unix(), "iat": now.Unix(), "nonce": nonce,
	})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := crypto.SHA256.New()
	_, _ = digest.Write([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func publicJWK(key *rsa.PublicKey) map[string]any {
	exponent := make([]byte, 4)
	binary.BigEndian.PutUint32(exponent, uint32(key.E))
	for len(exponent) > 1 && exponent[0] == 0 {
		exponent = exponent[1:]
	}
	return map[string]any{
		"kty": "RSA", "kid": "test-key", "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func responseCookie(t *testing.T, response *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie
		}
	}
	t.Fatalf("response has no %s cookie", name)
	return nil
}
