package server

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flowIDToken drives authorize (GET form -> POST login) -> token for the
// given login and returns the raw, unverified id_token string. It does NOT
// verify the signature (Phase 10 tests must inspect tokens whose signatures
// are intentionally broken).
func flowIDToken(t *testing.T, ts *httptest.Server, login string) string {
	t.Helper()
	auth := url.Values{}
	auth.Set("response_type", "code")
	auth.Set("client_id", "dev-client")
	auth.Set("redirect_uri", "https://app.test/cb")
	auth.Set("scope", "openid")
	auth.Set("state", "s")
	loc := authorizePostRedirect(t, ts, authorizeForm(login, auth))
	code := loc.Query().Get("code")

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", "https://app.test/cb")
	form.Set("client_id", "dev-client")
	resp, err := ts.Client().PostForm(ts.URL+"/token", form)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, "token exchange: %s", body)
	var tok map[string]any
	require.NoError(t, json.Unmarshal(body, &tok))
	idt, ok := tok["id_token"].(string)
	require.True(t, ok, "no id_token in response: %v", tok)
	return idt
}

// idTokenHeader returns the parsed JWT header (kid, alg).
func idTokenHeader(t *testing.T, idToken string) map[string]any {
	t.Helper()
	parts := strings.Split(idToken, ".")
	require.Len(t, parts, 3)
	hJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var hdr map[string]any
	require.NoError(t, json.Unmarshal(hJSON, &hdr))
	return hdr
}

// tryVerifyIDToken fetches JWKS and attempts to verify the id_token's
// signature. It returns the parsed claims on success, or an error describing
// why verification failed (no key for the kid, or signature mismatch). It
// never fails the test, so Phase 10 failure scenarios can assert on the error.
func tryVerifyIDToken(t *testing.T, ts *httptest.Server, idToken string) (map[string]any, error) {
	t.Helper()
	parts := strings.Split(idToken, ".")
	require.Len(t, parts, 3)
	hdr := idTokenHeader(t, idToken)
	kid, _ := hdr["kid"].(string)

	jwksResp, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer jwksResp.Body.Close()
	if jwksResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks status %d", jwksResp.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			Kid, N, E string
		}
	}
	require.NoError(t, json.NewDecoder(jwksResp.Body).Decode(&jwks))

	pubByKey := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		nBytes, _ := base64.RawURLEncoding.DecodeString(k.N)
		eBytes, _ := base64.RawURLEncoding.DecodeString(k.E)
		pubByKey[k.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}
	}
	pub, ok := pubByKey[kid]
	if !ok {
		return nil, fmt.Errorf("no key for kid %q (available: %v)", kid, keysOf(pubByKey))
	}

	pJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(pJSON, &claims))

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	signed := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signed))
	return claims, rsaVerifyErr(pub, sum[:], sig)
}

// rsaVerifyErr wraps rsa.VerifyPKCS1v15 to return an error instead of
// fataling, so Phase 10 failure scenarios can assert on the error message.
func rsaVerifyErr(pub *rsa.PublicKey, hashed, sig []byte) error {
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed, sig)
}

// TestPhase10_JWKSPublishesMultipleKids verifies that /jwks advertises the
// active key plus the rotated and bad-sig keys, so RPs see a multi-key set.
func TestPhase10_JWKSPublishesMultipleKids(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct{ Kid string }
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&jwks))
	kids := map[string]bool{}
	for _, k := range jwks.Keys {
		kids[k.Kid] = true
	}
	assert.True(t, kids["dev-key-1"])
	assert.True(t, kids["rotated-key-2"])
	assert.True(t, kids["bad-sig-key"])
	assert.Len(t, jwks.Keys, 3)
}

// TestPhase10_KeyRotatedTokenVerifies verifies that a token signed with the
// rotated key (kid rotated-key-2) verifies against the JWKS entry for that
// kid — i.e. the RP must look up the kid, not assume a single key.
func TestPhase10_KeyRotatedTokenVerifies(t *testing.T) {
	_, ts := newTestServer(t)
	idt := flowIDToken(t, ts, "key-rotated")
	assert.Equal(t, "rotated-key-2", idTokenHeader(t, idt)["kid"])
	claims, err := tryVerifyIDToken(t, ts, idt)
	require.NoError(t, err, "rotated token must verify against JWKS")
	assert.NotEmpty(t, claims["sub"])
}

// TestPhase10_KidNotFound verifies that the kid-not-found scenario produces a
// token whose kid is not present in JWKS, so verification fails because no
// key can be found.
func TestPhase10_KidNotFound(t *testing.T) {
	_, ts := newTestServer(t)
	idt := flowIDToken(t, ts, "kid-not-found")
	assert.Equal(t, "unknown-key", idTokenHeader(t, idt)["kid"])
	_, err := tryVerifyIDToken(t, ts, idt)
	require.Error(t, err, "verification must fail: kid not in JWKS")
	assert.Contains(t, err.Error(), "no key for kid")
}

// TestPhase10_BadSignature verifies that the bad-signature scenario produces a
// token whose kid IS in JWKS but whose signature does not verify, because the
// signing key differs from the key published under that kid.
func TestPhase10_BadSignature(t *testing.T) {
	_, ts := newTestServer(t)
	idt := flowIDToken(t, ts, "bad-signature")
	assert.Equal(t, "bad-sig-key", idTokenHeader(t, idt)["kid"])
	_, err := tryVerifyIDToken(t, ts, idt)
	require.Error(t, err, "verification must fail: signature made with a different key")
	assert.Contains(t, err.Error(), "verification error")
}

// TestPhase10_JWKS500 verifies that SetJWKSMode("500") makes /jwks return 500.
func TestPhase10_JWKS500(t *testing.T) {
	s, ts := newTestServer(t)
	s.SetJWKSMode("500")
	resp, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestPhase10_JWKSEmpty verifies that SetJWKSMode("empty") makes /jwks return
// an empty key set.
func TestPhase10_JWKSEmpty(t *testing.T) {
	s, ts := newTestServer(t)
	s.SetJWKSMode("empty")
	resp, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer resp.Body.Close()
	var jwks struct {
		Keys []struct{ Kid string }
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&jwks))
	assert.Empty(t, jwks.Keys)
}

// TestPhase10_JWKSSlow verifies that SetJWKSMode("slow") delays the /jwks
// response. The delay is shortened for the test via the jwksSlowDelay var.
func TestPhase10_JWKSSlow(t *testing.T) {
	s, ts := newTestServer(t)
	prev := jwksSlowDelay
	jwksSlowDelay = 80 * time.Millisecond
	t.Cleanup(func() { jwksSlowDelay = prev })
	s.SetJWKSMode("slow")
	start := time.Now()
	resp, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer resp.Body.Close()
	elapsed := time.Since(start)
	assert.GreaterOrEqual(t, elapsed, 70*time.Millisecond, "jwks must be delayed")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestPhase10_DebugJWKSMode verifies that POST /debug/jwks-mode sets the mode
// and GET returns it, and that reset restores normal.
func TestPhase10_DebugJWKSMode(t *testing.T) {
	s, ts := newTestServer(t)

	// GET initial mode.
	resp, err := ts.Client().Get(ts.URL + "/debug/jwks-mode")
	require.NoError(t, err)
	defer resp.Body.Close()
	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, "normal", got["mode"])

	// POST to set "500".
	post, err := ts.Client().Post(ts.URL+"/debug/jwks-mode", "application/json",
		strings.NewReader(`{"mode":"500"}`))
	require.NoError(t, err)
	defer post.Body.Close()
	assert.Equal(t, http.StatusOK, post.StatusCode)

	// The index reports the mode.
	idx, err := ts.Client().Get(ts.URL + "/debug")
	require.NoError(t, err)
	defer idx.Body.Close()
	var idxBody map[string]any
	require.NoError(t, json.NewDecoder(idx.Body).Decode(&idxBody))
	assert.Equal(t, "500", idxBody["jwks_mode"])

	// /jwks is now 500.
	jw, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	jw.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, jw.StatusCode)

	// Reset restores normal.
	_ = s // satisfy unused var; reset via HTTP below
	reset, err := ts.Client().Post(ts.URL+"/debug/reset", "application/json", nil)
	require.NoError(t, err)
	defer reset.Body.Close()
	resp2, err := ts.Client().Get(ts.URL + "/jwks")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}
