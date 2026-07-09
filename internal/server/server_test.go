package server

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
	"time"

	"github.com/manuel/tinyidp/internal/client"
	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// newTestServer builds a Server with a fresh RSA key and mounts it on an
// httptest.Server. The issuer is set to the test server's base URL so
// discovery and JWT iss claims are consistent.
func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa keygen: %v", err)
	}
	s := &Server{
		issuer:        "", // filled in after we know the test server URL
		clients:       client.NewRegistry(),
		key:           key,
		kid:           "dev-key-1",
		registry:      scenario.New(),
		codes:         map[string]authCode{},
		tokens:        map[string]accessToken{},
		sessions:      map[string]*session{},
		refreshTokens: map[string]refreshToken{},
		deviceGrants:  map[string]deviceGrant{},
		dpopReplay:    map[string]time.Time{},
	}
	// Register a permissive test client that allows the test's redirect URI.
	// The built-in dev-client uses localhost:3000; tests use https://app.test/cb.
	s.clients.Register(client.Client{
		ID:                     "dev-client",
		RedirectURIs:           []string{"https://app.test/cb"},
		PostLogoutRedirectURIs: []string{"https://app.test/logout"},
	})
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	ts := httptest.NewServer(WithCORS(mux))
	t.Cleanup(ts.Close)
	s.issuer = ts.URL
	return s, ts
}

// TestIssuerPathPrefixRoutes proves tinyidp can serve Keycloak-shaped issuer
// URLs such as /realms/<name> while still deriving discovery metadata and ID
// token issuer claims from the configured path-based issuer.
func TestIssuerPathPrefixRoutes(t *testing.T) {
	const prefix = "/realms/personal-inbox"

	s, err := New(Options{Issuer: "http://issuer.test" + prefix})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	s.clients.Register(client.Client{
		ID:                     "dev-client",
		RedirectURIs:           []string{"https://app.test/cb"},
		PostLogoutRedirectURIs: []string{"https://app.test/logout"},
	})
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	ts := httptest.NewServer(WithCORS(mux))
	t.Cleanup(ts.Close)
	s.issuer = ts.URL + prefix

	resp, err := ts.Client().Get(ts.URL + prefix + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("prefixed discovery: %v", err)
	}
	var discovery map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		resp.Body.Close()
		t.Fatalf("decode discovery: %v", err)
	}
	resp.Body.Close()
	if discovery["issuer"] != s.issuer {
		t.Fatalf("issuer = %v want %s", discovery["issuer"], s.issuer)
	}
	if discovery["authorization_endpoint"] != s.issuer+"/authorize" {
		t.Fatalf("authorization_endpoint = %v", discovery["authorization_endpoint"])
	}
	if discovery["token_endpoint"] != s.issuer+"/token" {
		t.Fatalf("token_endpoint = %v", discovery["token_endpoint"])
	}
	if discovery["jwks_uri"] != s.issuer+"/jwks" {
		t.Fatalf("jwks_uri = %v", discovery["jwks_uri"])
	}

	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid profile email")
	auth.Set("state", "prefixed-state")
	auth.Set("nonce", "prefixed-nonce")

	formResp, err := ts.Client().Get(ts.URL + prefix + "/authorize?" + auth.Encode())
	if err != nil {
		t.Fatalf("prefixed authorize GET: %v", err)
	}
	formBody, _ := io.ReadAll(formResp.Body)
	formResp.Body.Close()
	if formResp.StatusCode != http.StatusOK || !strings.Contains(string(formBody), `action="authorize"`) {
		t.Fatalf("prefixed authorize form = %d %q", formResp.StatusCode, formBody)
	}

	c := ts.Client()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	postResp, err := c.PostForm(ts.URL+prefix+"/authorize", authorizeForm("alice", auth))
	if err != nil {
		t.Fatalf("prefixed authorize POST: %v", err)
	}
	loc, err := postResp.Location()
	postResp.Body.Close()
	if err != nil {
		t.Fatalf("prefixed authorize redirect location: %v", err)
	}
	if loc.Query().Get("state") != "prefixed-state" {
		t.Fatalf("state not echoed: %s", loc.String())
	}
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in prefixed redirect: %s", loc.String())
	}

	tokForm := url.Values{}
	tokForm.Set("grant_type", "authorization_code")
	tokForm.Set("code", code)
	tokForm.Set("redirect_uri", "https://app.test/cb")
	tokForm.Set("client_id", "dev-client")
	tokenResp, err := ts.Client().PostForm(ts.URL+prefix+"/token", tokForm)
	if err != nil {
		t.Fatalf("prefixed token exchange: %v", err)
	}
	defer tokenResp.Body.Close()
	var tokenBody map[string]any
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenBody); err != nil {
		t.Fatalf("decode prefixed token response: %v", err)
	}
	if tokenResp.StatusCode != http.StatusOK {
		t.Fatalf("prefixed token status %d: %v", tokenResp.StatusCode, tokenBody)
	}
	claims := verifyIDTokenSignature(t, ts, tokenBody["id_token"].(string))
	if claims["iss"] != s.issuer {
		t.Fatalf("id token iss = %v want %s", claims["iss"], s.issuer)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+prefix+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenBody["access_token"].(string))
	uiResp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("prefixed userinfo: %v", err)
	}
	defer uiResp.Body.Close()
	if uiResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(uiResp.Body)
		t.Fatalf("prefixed userinfo status %d: %s", uiResp.StatusCode, body)
	}
}

// fullFlow drives authorize (GET form -> POST login) -> token -> userinfo for
// the given login and extra authorize params (code_challenge / code_verifier /
// etc.) and returns the parsed token response, the verified ID token claims,
// and the userinfo body. It verifies the ID token signature against JWKS.
func fullFlow(t *testing.T, ts *httptest.Server, login string, extra url.Values) (map[string]any, map[string]any, map[string]any) {
	t.Helper()
	var tokenResp map[string]any
	var idClaims map[string]any
	var userinfo map[string]any

	// 1. authorize: GET renders the form; POST submits login + hidden params.
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
	// Sanity: the form renders and carries the hidden login field.
	formResp, err := ts.Client().Get(ts.URL + "/authorize?" + auth.Encode())
	if err != nil {
		t.Fatalf("authorize GET: %v", err)
	}
	formBody, _ := io.ReadAll(formResp.Body)
	formResp.Body.Close()
	if formResp.StatusCode != http.StatusOK || !strings.Contains(string(formBody), `name="login"`) {
		t.Fatalf("authorize GET did not render login form: %d %q", formResp.StatusCode, formBody)
	}

	loc := authorizePostRedirect(t, ts, authorizeForm(login, auth))
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

// authorizePostRedirect POSTs the login form without following redirects and
// returns the Location URL.
func authorizePostRedirect(t *testing.T, ts *httptest.Server, form url.Values) *url.URL {
	t.Helper()
	c := ts.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.PostForm(ts.URL+"/authorize", form)
	if err != nil {
		t.Fatalf("authorize POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("authorize POST status = %d, want 302", resp.StatusCode)
	}
	loc, err := resp.Location()
	if err != nil {
		t.Fatalf("no Location: %v", err)
	}
	return loc
}

func postAuthorizeNoRedirect(t *testing.T, ts *httptest.Server, form url.Values) *http.Response {
	t.Helper()
	c := ts.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.PostForm(ts.URL+"/authorize", form)
	if err != nil {
		t.Fatalf("authorize POST: %v", err)
	}
	return resp
}

// authorizeForm builds a POST form for the given login + authorize params.
func authorizeForm(login string, auth url.Values) url.Values {
	f := url.Values{}
	f.Set("login", login)
	f.Set("password", "ignored")
	for k := range auth {
		f.Set(k, auth.Get(k))
	}
	return f
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
	kid, _ := hdr["kid"].(string)
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

	// JWKS may publish multiple kids (Phase 10). Build a kid -> public key map
	// and look up the one the token header claims to be signed by.
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
	pubByKey := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		nBytes, _ := base64.RawURLEncoding.DecodeString(k.N)
		eBytes, _ := base64.RawURLEncoding.DecodeString(k.E)
		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes).Int64()
		pubByKey[k.Kid] = &rsa.PublicKey{N: n, E: int(e)}
	}
	pub, ok := pubByKey[kid]
	if !ok {
		t.Fatalf("jwks has no key for kid %q (available: %v)", kid, keysOf(pubByKey))
	}

	signed := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signed))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		t.Fatalf("id_token signature verification failed: %v", err)
	}
	return claims
}

// keysOf returns the kids of a kid->public-key map, for error messages.
func keysOf(m map[string]*rsa.PublicKey) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
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

func TestAuthorizeGETRejectsBadClient(t *testing.T) {
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

func TestAuthorizeGETRejectsDisallowedRedirectURI(t *testing.T) {
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

func TestAuthorizePOSTRequiresLogin(t *testing.T) {
	_, ts := newTestServer(t)
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	form := authorizeForm("", auth) // empty login
	resp, err := ts.Client().PostForm(ts.URL+"/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for empty login", resp.StatusCode)
	}
}

func TestHappyPathNoPKCE(t *testing.T) {
	_, ts := newTestServer(t)
	_, claims, ui := fullFlow(t, ts, "alice", nil)
	if claims["sub"] != user.FromLogin("alice").Sub {
		t.Fatalf("sub = %v", claims["sub"])
	}
	if ui["email"] != "alice@example.test" {
		t.Fatalf("userinfo email = %v", ui["email"])
	}
	if ui["name"] != "alice" {
		t.Fatalf("userinfo name = %v", ui["name"])
	}
}

func TestHappyPathWithPKCE(t *testing.T) {
	_, ts := newTestServer(t)
	verifier := "verifier-abc-123-very-long-random-string-for-pkce"
	challenge := b64(sha256sum(verifier))
	_, claims, _ := fullFlow(t, ts, "alice", url.Values{
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"code_verifier":         {verifier},
	})
	if claims["sub"] != user.FromLogin("alice").Sub {
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
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
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
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
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

// --- Phase 1: multiple synthetic users ---

func TestDistinctUsersHaveDistinctSubs(t *testing.T) {
	_, ts := newTestServer(t)
	_, alice, _ := fullFlow(t, ts, "alice", nil)
	_, bob, _ := fullFlow(t, ts, "bob", nil)
	if alice["sub"] == bob["sub"] {
		t.Fatalf("alice and bob share sub %v", alice["sub"])
	}
	if alice["sub"] != user.FromLogin("alice").Sub {
		t.Fatalf("alice sub = %v, want %s", alice["sub"], user.FromLogin("alice").Sub)
	}
	if bob["sub"] != user.FromLogin("bob").Sub {
		t.Fatalf("bob sub = %v, want %s", bob["sub"], user.FromLogin("bob").Sub)
	}
}

func TestSubIsStableAcrossLogins(t *testing.T) {
	_, ts := newTestServer(t)
	_, first, _ := fullFlow(t, ts, "alice", nil)
	_, second, _ := fullFlow(t, ts, "alice", nil)
	if first["sub"] != second["sub"] {
		t.Fatalf("alice sub not stable: %v vs %v", first["sub"], second["sub"])
	}
}

func TestArbitraryEmailLogin(t *testing.T) {
	_, ts := newTestServer(t)
	_, claims, ui := fullFlow(t, ts, "carol@example.test", nil)
	want := user.FromLogin("carol@example.test")
	if claims["email"] != want.Email {
		t.Fatalf("email = %v, want %s", claims["email"], want.Email)
	}
	if claims["name"] != want.Name {
		t.Fatalf("name = %v, want %s", claims["name"], want.Name)
	}
	if ui["email"] != "carol@example.test" {
		t.Fatalf("userinfo email = %v", ui["email"])
	}
	if ui["name"] != "carol" {
		t.Fatalf("userinfo name = %v", ui["name"])
	}
}
func sha256sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

// --- Phase 2: scenario registry ---

// TestScenarioHookIsThreadedThroughFlow proves the core Phase 2 property:
// a scenario's MutateClaims hook actually mutates the issued ID token. This
// is the foundation Phase 4 builds on (id-expired, id-wrong-aud, ...).
func TestScenarioHookIsThreadedThroughFlow(t *testing.T) {
	s, ts := newTestServer(t)
	// Inject a scenario that adds a custom claim. No handler code changed —
	// only the registry entry. This is the "one-file add" guarantee.
	s.registry.Register(scenario.Scenario{
		Name:        "custom-claim",
		Description: "injects a custom claim",
		User:        user.FromLogin("dave"),
		MutateClaims: func(claims map[string]any, now time.Time) {
			claims["custom"] = "from-scenario"
		},
	})
	_, claims, _ := fullFlow(t, ts, "custom-claim", nil)
	if claims["custom"] != "from-scenario" {
		t.Fatalf("scenario MutateClaims did not run: claims = %+v", claims)
	}
	if claims["sub"] != user.FromLogin("dave").Sub {
		t.Fatalf("sub = %v, want dave's sub", claims["sub"])
	}
}

func TestSeededUserPasswordValidation(t *testing.T) {
	s, ts := newTestServer(t)
	seeded, err := scenario.SeededUsersToScenarios([]scenario.SeededUser{
		{Login: "alice", Password: "alice-password", Sub: "user-alice-fixed"},
		{Login: "bob", Sub: "user-bob-fixed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.registry.RegisterAll(seeded)

	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid profile email")
	auth.Set("state", "pw-state")
	auth.Set("nonce", "pw-nonce")

	form := authorizeForm("alice", auth)
	form.Set("password", "alice-password")
	loc := authorizePostRedirect(t, ts, form)
	if loc.Query().Get("code") == "" {
		t.Fatalf("correct password did not issue code: %s", loc.String())
	}

	wrong := authorizeForm("alice", auth)
	wrong.Set("password", "wrong-password")
	resp := postAuthorizeNoRedirect(t, ts, wrong)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "invalid login or password") {
		t.Fatalf("wrong password body = %q", body)
	}

	missing := authorizeForm("alice", auth)
	missing.Del("password")
	resp = postAuthorizeNoRedirect(t, ts, missing)
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing password status = %d body=%s", resp.StatusCode, body)
	}

	s.mu.Lock()
	sessionCount := len(s.sessions)
	codeCount := len(s.codes)
	s.mu.Unlock()
	if sessionCount != 1 || codeCount != 1 {
		t.Fatalf("wrong/missing passwords created state: sessions=%d codes=%d", sessionCount, codeCount)
	}

	permissive := authorizeForm("bob", auth)
	permissive.Set("password", "anything")
	loc = authorizePostRedirect(t, ts, permissive)
	if loc.Query().Get("code") == "" {
		t.Fatalf("unprotected seeded user did not issue code: %s", loc.String())
	}

	builtin := authorizeForm("viewer", auth)
	builtin.Set("password", "anything")
	loc = authorizePostRedirect(t, ts, builtin)
	if loc.Query().Get("code") == "" {
		t.Fatalf("builtin user did not issue code: %s", loc.String())
	}
}

func TestSeededUserScenarioIsThreadedThroughFlow(t *testing.T) {
	s, ts := newTestServer(t)
	seeded, err := scenario.SeededUsersToScenarios([]scenario.SeededUser{
		{
			Login:             "alice",
			Sub:               "user-alice-fixed",
			Email:             "alice@inbox.test",
			Name:              "Alice Inbox",
			Groups:            []string{"inbox-users"},
			Roles:             []string{"writer"},
			Tenant:            "personal",
			PreferredUsername: "alice",
			Locale:            "en-US",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.registry.RegisterAll(seeded)

	_, claims, ui := fullFlow(t, ts, "alice", nil)
	if claims["sub"] != "user-alice-fixed" || ui["sub"] != "user-alice-fixed" {
		t.Fatalf("seeded sub not used: id=%v userinfo=%v", claims["sub"], ui["sub"])
	}
	if claims["email"] != "alice@inbox.test" || ui["email"] != "alice@inbox.test" {
		t.Fatalf("seeded email not used: id=%v userinfo=%v", claims["email"], ui["email"])
	}
	if claims["tenant"] != "personal" || ui["tenant"] != "personal" {
		t.Fatalf("seeded tenant not used: id=%v userinfo=%v", claims["tenant"], ui["tenant"])
	}
	if claims["preferred_username"] != "alice" || ui["preferred_username"] != "alice" {
		t.Fatalf("seeded preferred_username not used: id=%v userinfo=%v", claims["preferred_username"], ui["preferred_username"])
	}
	if claims["locale"] != "en-US" || ui["locale"] != "en-US" {
		t.Fatalf("seeded locale not used: id=%v userinfo=%v", claims["locale"], ui["locale"])
	}
	groups, ok := claims["groups"].([]any)
	if !ok || len(groups) != 1 || groups[0] != "inbox-users" {
		t.Fatalf("seeded groups not used in ID token: %#v", claims["groups"])
	}
	uiGroups, ok := ui["groups"].([]any)
	if !ok || len(uiGroups) != 1 || uiGroups[0] != "inbox-users" {
		t.Fatalf("seeded groups not used in userinfo: %#v", ui["groups"])
	}
	roles, ok := claims["roles"].([]any)
	if !ok || len(roles) != 1 || roles[0] != "writer" {
		t.Fatalf("seeded roles not used in ID token: %#v", claims["roles"])
	}
	uiRoles, ok := ui["roles"].([]any)
	if !ok || len(uiRoles) != 1 || uiRoles[0] != "writer" {
		t.Fatalf("seeded roles not used in userinfo: %#v", ui["roles"])
	}
}

// --- Phase 3: self-documenting login page ---

// TestLoginPageListsBuiltinScenarios verifies the login page renders the
// scenario registry as quick-pick buttons, so the page is always in sync
// with what Lookup accepts.
func TestLoginPageListsBuiltinScenarios(t *testing.T) {
	_, ts := newTestServer(t)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	page := string(body)
	if !strings.Contains(page, "Quick picks") {
		t.Fatal("login page missing 'Quick picks' section")
	}
	if !strings.Contains(page, "Normal users") {
		t.Fatal("login page missing 'Normal users' group")
	}
	// Every registered scenario with a Category should appear as a button.
	for _, name := range []string{"alice", "bob"} {
		if !strings.Contains(page, `data-login="`+name+`"`) {
			t.Fatalf("login page missing quick-pick button for %q", name)
		}
	}
}

// --- Phase 4: high-value failure scenarios (matrix) ---

// runAuthorizeLogin does the GET form + POST login and returns the redirect
// Location (without exchanging the code). Used by failure-scenario tests
// that stop at /authorize.
func runAuthorizeLogin(t *testing.T, ts *httptest.Server, login string) *url.URL {
	t.Helper()
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "st")
	auth.Set("nonce", "nonce-xyz")
	// GET renders the form (sanity).
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + auth.Encode())
	if err != nil {
		t.Fatalf("authorize GET: %v", err)
	}
	resp.Body.Close()
	return authorizePostRedirect(t, ts, authorizeForm(login, auth))
}

// exchangeCode POSTs the code to /token and returns (status, parsedBody).
func exchangeCode(t *testing.T, ts *httptest.Server, code string) (int, map[string]any) {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", form)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	return resp.StatusCode, parsed
}

func TestPhase4_AuthErrorScenariosRedirectWithError(t *testing.T) {
	cases := map[string]string{
		"fail-access-denied":    "access_denied",
		"fail-login-required":   "login_required",
		"fail-consent-required": "consent_required",
		"fail-server-error":     "server_error",
	}
	for login, wantErr := range cases {
		t.Run(login, func(t *testing.T) {
			_, ts := newTestServer(t)
			loc := runAuthorizeLogin(t, ts, login)
			q := loc.Query()
			if q.Get("error") != wantErr {
				t.Fatalf("error = %q, want %q (redirect=%s)", q.Get("error"), wantErr, loc)
			}
			if q.Get("state") != "st" {
				t.Fatalf("state = %q, want st", q.Get("state"))
			}
			if q.Get("code") != "" {
				t.Fatalf("auth-error scenario must not issue a code, got %q", q.Get("code"))
			}
		})
	}
}

func TestPhase4_TokenErrorScenarios(t *testing.T) {
	t.Run("token-invalid-grant", func(t *testing.T) {
		_, ts := newTestServer(t)
		loc := runAuthorizeLogin(t, ts, "token-invalid-grant")
		status, body := exchangeCode(t, ts, loc.Query().Get("code"))
		if status != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", status)
		}
		if body["error"] != "invalid_grant" {
			t.Fatalf("error = %v, want invalid_grant", body["error"])
		}
	})
	t.Run("token-server-error", func(t *testing.T) {
		_, ts := newTestServer(t)
		loc := runAuthorizeLogin(t, ts, "token-server-error")
		status, body := exchangeCode(t, ts, loc.Query().Get("code"))
		if status != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", status)
		}
		if body["error"] != "server_error" {
			t.Fatalf("error = %v, want server_error", body["error"])
		}
	})
}

func TestPhase4_IDTokenMutations(t *testing.T) {
	_, ts := newTestServer(t)

	// id-expired -> exp in the past.
	loc := runAuthorizeLogin(t, ts, "id-expired")
	status, body := exchangeCode(t, ts, loc.Query().Get("code"))
	if status != http.StatusOK {
		t.Fatalf("id-expired exchange status = %d", status)
	}
	claims := verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if exp, _ := claims["exp"].(float64); exp >= float64(time.Now().Unix()) {
		t.Fatalf("id-expired: exp %v not in the past", claims["exp"])
	}

	// id-wrong-aud -> aud != dev-client.
	loc = runAuthorizeLogin(t, ts, "id-wrong-aud")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if claims["aud"] != "some-other-client" {
		t.Fatalf("id-wrong-aud: aud = %v", claims["aud"])
	}

	// id-wrong-iss -> iss ends with /wrong.
	loc = runAuthorizeLogin(t, ts, "id-wrong-iss")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if !strings.HasSuffix(claims["iss"].(string), "/wrong") {
		t.Fatalf("id-wrong-iss: iss = %v", claims["iss"])
	}

	// id-missing-email -> email absent.
	loc = runAuthorizeLogin(t, ts, "id-missing-email")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if _, ok := claims["email"]; ok {
		t.Fatalf("id-missing-email: email present = %v", claims["email"])
	}

	// id-email-unverified -> email_verified = false.
	loc = runAuthorizeLogin(t, ts, "id-email-unverified")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if claims["email_verified"] != false {
		t.Fatalf("id-email-unverified: email_verified = %v", claims["email_verified"])
	}

	// id-bad-nonce -> nonce != nonce-xyz.
	loc = runAuthorizeLogin(t, ts, "id-bad-nonce")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if claims["nonce"] == "nonce-xyz" {
		t.Fatalf("id-bad-nonce: nonce still matches: %v", claims["nonce"])
	}

	// id-future-iat -> iat in the future.
	loc = runAuthorizeLogin(t, ts, "id-future-iat")
	_, body = exchangeCode(t, ts, loc.Query().Get("code"))
	claims = verifyIDTokenSignature(t, ts, body["id_token"].(string))
	if iat, _ := claims["iat"].(float64); iat <= float64(time.Now().Unix()) {
		t.Fatalf("id-future-iat: iat %v not in the future", claims["iat"])
	}
}

func TestPhase4_UserInfoFailures(t *testing.T) {
	// The full matrix for userinfo failures is exercised with a dedicated
	// helper, since fullFlow asserts a 200 at userinfo.
	_, ts := newTestServer(t)
	for _, tc := range []struct {
		login      string
		wantStatus int
	}{
		{"userinfo-401", http.StatusUnauthorized},
		{"userinfo-500", http.StatusInternalServerError},
	} {
		t.Run(tc.login, func(t *testing.T) {
			loc := runAuthorizeLogin(t, ts, tc.login)
			status, body := exchangeCode(t, ts, loc.Query().Get("code"))
			if status != http.StatusOK {
				t.Fatalf("token exchange status = %d", status)
			}
			access := body["access_token"].(string)
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
			req.Header.Set("Authorization", "Bearer "+access)
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("userinfo status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestPhase4_UserInfoSubMismatch(t *testing.T) {
	_, ts := newTestServer(t)
	loc := runAuthorizeLogin(t, ts, "userinfo-sub-mismatch")
	status, body := exchangeCode(t, ts, loc.Query().Get("code"))
	if status != http.StatusOK {
		t.Fatalf("token exchange status = %d", status)
	}
	idClaims := verifyIDTokenSignature(t, ts, body["id_token"].(string))
	access := body["access_token"].(string)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var ui map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ui)
	if ui["sub"] == idClaims["sub"] {
		t.Fatalf("userinfo-sub-mismatch: sub matched ID token (%v)", ui["sub"])
	}
	if !strings.HasSuffix(ui["sub"].(string), "-different") {
		t.Fatalf("userinfo-sub-mismatch: sub = %v, want suffix -different", ui["sub"])
	}
}

// --- Phase 5: multiple clients ---

// newTestServerWithBuiltinClients builds a server whose client registry is the
// builtins (dev-client, public-spa, web-app) plus the test's https://app.test/cb
// redirect URI on dev-client. Used by Phase 5 multi-client tests.
func newTestServerWithBuiltinClients(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa keygen: %v", err)
	}
	clients := client.NewRegistry()
	// Allow the test's redirect URI on each builtin client so they can all
	// complete a flow against the test harness.
	for _, id := range []string{"dev-client", "public-spa", "web-app"} {
		c, _ := clients.Lookup(id)
		c.RedirectURIs = append(c.RedirectURIs, "https://app.test/cb")
		clients.Register(c)
	}
	s := &Server{
		issuer:        "",
		clients:       clients,
		key:           key,
		kid:           "dev-key-1",
		registry:      scenario.New(),
		codes:         map[string]authCode{},
		tokens:        map[string]accessToken{},
		sessions:      map[string]*session{},
		refreshTokens: map[string]refreshToken{},
		deviceGrants:  map[string]deviceGrant{},
		dpopReplay:    map[string]time.Time{},
	}
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	ts := httptest.NewServer(WithCORS(mux))
	t.Cleanup(ts.Close)
	s.issuer = ts.URL
	return s, ts
}

// TestPhase5_PublicSpaRequiresPKCE verifies that the public-spa client rejects
// an authorize request with no code_challenge.
func TestPhase5_PublicSpaRequiresPKCE(t *testing.T) {
	_, ts := newTestServerWithBuiltinClients(t)
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", "public-spa")
	q.Set("redirect_uri", "https://app.test/cb")
	q.Set("scope", "openid")
	resp, err := ts.Client().Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("public-spa without PKCE: status = %d, want 400", resp.StatusCode)
	}
}

// TestPhase5_PublicSpaAcceptsWithPKCE verifies the public-spa client accepts
// an authorize POST that includes a code_challenge.
func TestPhase5_PublicSpaAcceptsWithPKCE(t *testing.T) {
	_, ts := newTestServerWithBuiltinClients(t)
	verifier := "a-verifier-long-enough-for-pkce-testing-1234567890"
	challenge := b64(sha256sum(verifier))
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "public-spa")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	auth.Set("code_challenge", challenge)
	auth.Set("code_challenge_method", "S256")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	if loc.Query().Get("code") == "" {
		t.Fatalf("public-spa with PKCE should issue a code (redirect=%s)", loc)
	}
}

// TestPhase5_WebAppRequiresSecret verifies that the confidential web-app
// client rejects a token exchange without the correct client_secret.
func TestPhase5_WebAppRequiresSecret(t *testing.T) {
	_, ts := newTestServerWithBuiltinClients(t)
	// Authorize as web-app (no secret needed at /authorize).
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "web-app")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")

	// Exchange without the secret -> 401 invalid_client.
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "web-app")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	tresp.Body.Close()
	if tresp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("web-app without secret: status = %d, want 401", tresp.StatusCode)
	}
}

// TestPhase5_WebAppSucceedsWithSecret verifies the web-app client succeeds when
// the correct secret is presented via client_secret_post.
func TestPhase5_WebAppSucceedsWithSecret(t *testing.T) {
	_, ts := newTestServerWithBuiltinClients(t)
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "web-app")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "web-app")
	form.Set("client_secret", "dev-secret")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer tresp.Body.Close()
	if tresp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tresp.Body)
		t.Fatalf("web-app with secret: status = %d, want 200: %s", tresp.StatusCode, body)
	}
}

// TestPhase5_CrossClientCodeRejection verifies that a code issued to one
// client cannot be redeemed by another.
func TestPhase5_CrossClientCodeRejection(t *testing.T) {
	_, ts := newTestServerWithBuiltinClients(t)
	// Issue a code as dev-client.
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm("alice", auth))
	code := loc.Query().Get("code")

	// Try to redeem it as web-app (different client).
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "web-app")
	form.Set("client_secret", "dev-secret")
	tresp, err := ts.Client().PostForm(ts.URL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer tresp.Body.Close()
	if tresp.StatusCode != http.StatusBadRequest {
		t.Fatalf("cross-client redemption: status = %d, want 400", tresp.StatusCode)
	}
}
