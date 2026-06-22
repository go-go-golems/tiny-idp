package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newTestServer builds an in-process server with a fresh RSA key and mounts it
// on an httptest.Server. The issuer is set to the test server's base URL so
// discovery and JWT iss claims are consistent.
func newTestServer(t *testing.T) (*server, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa keygen: %v", err)
	}
	s := &server{
		issuer:       "", // filled in after we know the test server URL
		clientID:     "dev-client",
		clientSecret: "",
		redirectURIs: map[string]bool{"https://app.test/cb": true},
		key:          key,
		kid:          "dev-key-1",
		codes:        map[string]authCode{},
		tokens:       map[string]accessToken{},
		user: user{
			Sub:   "user-123",
			Email: "dev@example.test",
			Name:  "Dev User",
		},
	}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	ts := httptest.NewServer(withCORS(mux))
	t.Cleanup(ts.Close)
	s.issuer = ts.URL
	return s, ts
}

// fullFlow drives authorize -> token -> userinfo for the given extra authorize
// params (code_challenge / code_verifier / etc.) and returns the parsed token
// response, the verified ID token claims, and the userinfo body. It verifies
// the ID token signature against the server's JWKS.
func fullFlow(t *testing.T, ts *httptest.Server, extra url.Values) (tokenResp map[string]any, idClaims map[string]any, userinfo map[string]any) {
	t.Helper()

	// 1. authorize
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid profile email")
	auth.Set("state", "st-123")
	auth.Set("nonce", "nonce-xyz")
	for k, vs := range extra {
		for _, v := range vs {
			auth.Set(k, v)
		}
	}
	loc := authorizeRedirect(t, ts, auth)
	q := loc.Query()
	if q.Get("state") != "st-123" {
		t.Fatalf("state not echoed: got %q", q.Get("state"))
	}
	code := q.Get("code")
	if code == "" {
		t.Fatalf("no code in redirect: %s", loc.String())
	}

	// 2. token
	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	if v := extra.Get("code_verifier"); v != "" {
		tokForm.Set("code_verifier", v)
	}
	resp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token exchange status %d: %s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	// 3. verify ID token signature against JWKS public key
	idToken, _ := tokenResp["id_token"].(string)
	if idToken == "" {
		t.Fatal("missing id_token")
	}
	idClaims = verifyIDTokenSignature(t, ts, idToken)

	if idClaims["iss"] != ts.URL {
		t.Fatalf("iss mismatch: got %v want %s", idClaims["iss"], ts.URL)
	}
	if idClaims["aud"] != "dev-client" {
		t.Fatalf("aud mismatch: got %v", idClaims["aud"])
	}
	if idClaims["nonce"] != "nonce-xyz" {
		t.Fatalf("nonce mismatch: got %v", idClaims["nonce"])
	}

	// 4. userinfo
	access, _ := tokenResp["access_token"].(string)
	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req2.Header.Set("Authorization", "Bearer "+access)
	resp2, err := ts.Client().Do(req2)
	if err != nil {
		t.Fatalf("userinfo: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("userinfo status %d: %s", resp2.StatusCode, body2)
	}
	if err := json.Unmarshal(body2, &userinfo); err != nil {
		t.Fatalf("decode userinfo: %v", err)
	}
	return tokenResp, idClaims, userinfo
}

// authorizeRedirect performs an authorize request without following redirects
// and returns the Location URL.
func authorizeRedirect(t *testing.T, ts *httptest.Server, auth url.Values) *url.URL {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/authorize?"+auth.Encode(), nil)
	c := ts.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d, want 302", resp.StatusCode)
	}
	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("no Location: %v", err)
	}
	return loc
}

func verifyIDTokenSignature(t *testing.T, ts *httptest.Server, idToken string) map[string]any {
	t.Helper()
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		t.Fatalf("id_token is not 3 parts: %d", len(parts))
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var hdr map[string]any
	_ = json.Unmarshal(headerJSON, &hdr)
	if hdr["alg"] != "RS256" {
		t.Fatalf("alg = %v, want RS256", hdr["alg"])
	}
	if hdr["kid"] != "dev-key-1" {
		t.Fatalf("kid = %v, want dev-key-1", hdr["kid"])
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}

	jwksResp, err := ts.Client().Get(ts.URL + "/jwks")
	if err != nil {
		t.Fatalf("jwks: %v", err)
	}
	defer jwksResp.Body.Close()
	var jwks struct {
		Keys []struct {
			Kty, Use, Kid, Alg, N, E string
		}
	}
	if err := json.NewDecoder(jwksResp.Body).Decode(&jwks); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	if len(jwks.Keys) != 1 || jwks.Keys[0].Kid != "dev-key-1" {
		t.Fatalf("jwks unexpected: %+v", jwks)
	}
	nBytes, _ := base64.RawURLEncoding.DecodeString(jwks.Keys[0].N)
	eBytes, _ := base64.RawURLEncoding.DecodeString(jwks.Keys[0].E)
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes).Int64()
	pub := &rsa.PublicKey{N: n, E: int(e)}

	signed := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("id_token signature verification failed: %v", err)
	}
	return claims
}

func TestDiscoveryContainsRequiredFields(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var disc map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&disc)
	for _, k := range []string{"issuer", "authorization_endpoint", "token_endpoint", "userinfo_endpoint", "jwks_uri", "response_types_supported", "id_token_signing_alg_values_supported", "code_challenge_methods_supported"} {
		if _, ok := disc[k]; !ok {
			t.Fatalf("discovery missing %q", k)
		}
	}
	if disc["issuer"] != ts.URL {
		t.Fatalf("issuer = %v want %s", disc["issuer"], ts.URL)
	}
}

func TestAuthorizeRejectsBadClient(t *testing.T) {
	_, ts := newTestServer(t)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "wrong")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad client_id", resp.StatusCode)
	}
}

func TestAuthorizeRejectsDisallowedRedirectURI(t *testing.T) {
	_, ts := newTestServer(t)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://evil.test/cb")
	q.Set("scope", "openid")
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for disallowed redirect_uri", resp.StatusCode)
	}
}

func TestHappyPathNoPKCE(t *testing.T) {
	_, ts := newTestServer(t)
	_, claims, ui := fullFlow(t, ts, nil)
	if claims["sub"] != "user-123" {
		t.Fatalf("sub = %v", claims["sub"])
	}
	if ui["email"] != "dev@example.test" {
		t.Fatalf("userinfo email = %v", ui["email"])
	}
}

func TestHappyPathWithPKCE(t *testing.T) {
	_, ts := newTestServer(t)
	verifier := "verifier-abc-123-very-long-random-string-for-pkce"
	challenge := b64(sha256sum(verifier))
	_, claims, _ := fullFlow(t, ts, url.Values{
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"code_verifier":         {verifier},
	})
	if claims["sub"] != "user-123" {
		t.Fatalf("sub = %v", claims["sub"])
	}
}

func TestPKCEVerifierMismatchRejected(t *testing.T) {
	_, ts := newTestServer(t)
	wrong := "wrong-verifier-value-here-that-does-not-match"
	verifier := "verifier-abc-123-very-long-random-string-for-pkce"
	challenge := b64(sha256sum(verifier))

	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	auth.Set("code_challenge", challenge)
	auth.Set("code_challenge_method", "S256")
	loc := authorizeRedirect(t, ts, auth)
	code := loc.Query().Get("code")

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tokForm.Set("code_verifier", wrong)
	tresp, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	if err != nil {
		t.Fatal(err)
	}
	defer tresp.Body.Close()
	if tresp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for PKCE mismatch", tresp.StatusCode)
	}
}

func TestCodeIsOneTimeUse(t *testing.T) {
	_, ts := newTestServer(t)
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizeRedirect(t, ts, auth)
	code := loc.Query().Get("code")

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	first, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	if err != nil {
		t.Fatal(err)
	}
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first exchange = %d", first.StatusCode)
	}
	second, err := ts.Client().PostForm(ts.URL+"/token", tokForm)
	if err != nil {
		t.Fatal(err)
	}
	second.Body.Close()
	if second.StatusCode != http.StatusBadRequest {
		t.Fatalf("reuse of code = %d, want 400", second.StatusCode)
	}
}

func TestUserInfoRejectsBadToken(t *testing.T) {
	_, ts := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func sha256sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
