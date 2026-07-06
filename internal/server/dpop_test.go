package server

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestDPoPProofValidation(t *testing.T) {
	s := &Server{dpopReplay: map[string]time.Time{}}
	key := newDPoPECKey(t)
	req := httptest.NewRequest(http.MethodPost, "http://issuer.test/token?ignored=true", nil)

	proof := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/token", "", "proof-1", time.Now())
	parsed, err := s.validateDPoPProof(req, proof, "")
	if err != nil {
		t.Fatalf("valid proof rejected: %v", err)
	}
	if parsed.JKT == "" || parsed.JTI != "proof-1" {
		t.Fatalf("unexpected parsed proof: %#v", parsed)
	}
	if _, err := s.validateDPoPProof(req, proof, ""); err == nil {
		t.Fatal("replayed proof accepted")
	}

	wrongMethod := makeDPoPProof(t, key, http.MethodGet, "http://issuer.test/token", "", "proof-2", time.Now())
	if _, err := s.validateDPoPProof(req, wrongMethod, ""); err == nil {
		t.Fatal("wrong htm accepted")
	}

	wrongURL := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/other", "", "proof-3", time.Now())
	if _, err := s.validateDPoPProof(req, wrongURL, ""); err == nil {
		t.Fatal("wrong htu accepted")
	}

	stale := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/token", "", "proof-4", time.Now().Add(-10*time.Minute))
	if _, err := s.validateDPoPProof(req, stale, ""); err == nil {
		t.Fatal("stale iat accepted")
	}

	access := "access-token-value"
	missingATH := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/token", "", "proof-5", time.Now())
	if _, err := s.validateDPoPProof(req, missingATH, access); err == nil {
		t.Fatal("missing ath accepted")
	}
	wrongATH := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/token", "wrong", "proof-6", time.Now())
	if _, err := s.validateDPoPProof(req, wrongATH, access); err == nil {
		t.Fatal("wrong ath accepted")
	}
	withATH := makeDPoPProof(t, key, http.MethodPost, "http://issuer.test/token", accessTokenHash(access), "proof-7", time.Now())
	if _, err := s.validateDPoPProof(req, withATH, access); err != nil {
		t.Fatalf("valid ath proof rejected: %v", err)
	}
}

func TestDPoPRS256ProofValidation(t *testing.T) {
	s := &Server{dpopReplay: map[string]time.Time{}}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa keygen: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://issuer.test/token", nil)
	proof := makeDPoPRSProof(t, key, http.MethodPost, "http://issuer.test/token", "", "rsa-proof-1", time.Now())
	parsed, err := s.validateDPoPProof(req, proof, "")
	if err != nil {
		t.Fatalf("valid RS256 proof rejected: %v", err)
	}
	if parsed.JKT == "" {
		t.Fatalf("missing jkt: %#v", parsed)
	}
}

func TestDPoPAuthorizationCodeUserinfoAndRefresh(t *testing.T) {
	s, ts := newTestServer(t)
	key := newDPoPECKey(t)

	loc := authorizePostRedirect(t, ts, authorizeForm("alice", url.Values{
		"response_type": {"code"},
		"client_id":     {"dev-client"},
		"redirect_uri":  {"https://app.test/cb"},
		"scope":         {"openid profile email offline_access"},
		"state":         {"state-dpop"},
		"nonce":         {"nonce-dpop"},
	}))
	body := postTokenWithDPoP(t, ts, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {loc.Query().Get("code")},
		"redirect_uri": {"https://app.test/cb"},
		"client_id":    {"dev-client"},
	}, makeDPoPProof(t, key, http.MethodPost, ts.URL+"/token", "", "token-proof-1", time.Now()), http.StatusOK)
	if body["token_type"] != "DPoP" {
		t.Fatalf("token_type = %v", body["token_type"])
	}
	access := body["access_token"].(string)
	refresh := body["refresh_token"].(string)
	s.mu.Lock()
	at := s.tokens[access]
	rt := s.refreshTokens[refresh]
	s.mu.Unlock()
	if at.DPoPJKT == "" || rt.DPoPJKT == "" || at.DPoPJKT != rt.DPoPJKT {
		t.Fatalf("tokens not consistently DPoP-bound: access=%q refresh=%q", at.DPoPJKT, rt.DPoPJKT)
	}

	userinfoStatus(t, ts, "Bearer "+access, "", http.StatusUnauthorized)
	userinfoStatus(t, ts, "DPoP "+access, makeDPoPProof(t, key, http.MethodGet, ts.URL+"/userinfo", accessTokenHash(access), "userinfo-proof-1", time.Now()), http.StatusOK)
	userinfoStatus(t, ts, "DPoP "+access, makeDPoPProof(t, key, http.MethodGet, ts.URL+"/userinfo", "wrong-ath", "userinfo-proof-2", time.Now()), http.StatusUnauthorized)
	wrongKey := newDPoPECKey(t)
	userinfoStatus(t, ts, "DPoP "+access, makeDPoPProof(t, wrongKey, http.MethodGet, ts.URL+"/userinfo", accessTokenHash(access), "userinfo-proof-3", time.Now()), http.StatusUnauthorized)
	replay := makeDPoPProof(t, key, http.MethodGet, ts.URL+"/userinfo", accessTokenHash(access), "userinfo-proof-4", time.Now())
	userinfoStatus(t, ts, "DPoP "+access, replay, http.StatusOK)
	userinfoStatus(t, ts, "DPoP "+access, replay, http.StatusUnauthorized)

	postTokenWithDPoP(t, ts, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"dev-client"},
	}, "", http.StatusBadRequest)
	postTokenWithDPoP(t, ts, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"dev-client"},
	}, makeDPoPProof(t, wrongKey, http.MethodPost, ts.URL+"/token", "", "refresh-proof-wrong-key", time.Now()), http.StatusBadRequest)
	rotated := postTokenWithDPoP(t, ts, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {"dev-client"},
	}, makeDPoPProof(t, key, http.MethodPost, ts.URL+"/token", "", "refresh-proof-1", time.Now()), http.StatusOK)
	if rotated["token_type"] != "DPoP" || rotated["refresh_token"] == "" {
		t.Fatalf("unexpected refresh response: %#v", rotated)
	}
}

func TestDPoPDeviceCodeToken(t *testing.T) {
	s, ts := newTestServer(t)
	key := newDPoPECKey(t)
	started := deviceAuthorization(t, ts, "dev-client", "openid profile email")
	approveDevice(t, ts, started["user_code"].(string), "alice", "", "Device request approved")

	body := postTokenWithDPoP(t, ts, url.Values{
		"grant_type":  {deviceGrantType},
		"device_code": {started["device_code"].(string)},
		"client_id":   {"dev-client"},
	}, makeDPoPProof(t, key, http.MethodPost, ts.URL+"/token", "", "device-token-proof-1", time.Now()), http.StatusOK)
	if body["token_type"] != "DPoP" || body["access_token"] == "" {
		t.Fatalf("unexpected device token response: %#v", body)
	}
	s.mu.Lock()
	at := s.tokens[body["access_token"].(string)]
	s.mu.Unlock()
	if at.DPoPJKT == "" {
		t.Fatal("device access token was not DPoP-bound")
	}
}

func newDPoPECKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa keygen: %v", err)
	}
	return key
}

func makeDPoPProof(t *testing.T, key *ecdsa.PrivateKey, method, htu, ath, jti string, iat time.Time) string {
	t.Helper()
	header := map[string]any{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": map[string]any{
			"kty": "EC",
			"crv": "P-256",
			"x":   b64(padBigInt(key.PublicKey.X, 32)),
			"y":   b64(padBigInt(key.PublicKey.Y, 32)),
		},
	}
	return signDPoPProof(t, header, method, htu, ath, jti, iat, func(input []byte) []byte {
		sum := sha256.Sum256(input)
		r, s, err := ecdsa.Sign(rand.Reader, key, sum[:])
		if err != nil {
			t.Fatalf("ecdsa sign: %v", err)
		}
		return append(padBigInt(r, 32), padBigInt(s, 32)...)
	})
}

func makeDPoPRSProof(t *testing.T, key *rsa.PrivateKey, method, htu, ath, jti string, iat time.Time) string {
	t.Helper()
	header := map[string]any{
		"typ": "dpop+jwt",
		"alg": "RS256",
		"jwk": map[string]any{
			"kty": "RSA",
			"n":   b64(key.PublicKey.N.Bytes()),
			"e":   b64(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		},
	}
	return signDPoPProof(t, header, method, htu, ath, jti, iat, func(input []byte) []byte {
		sum := sha256.Sum256(input)
		sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
		if err != nil {
			t.Fatalf("rsa sign: %v", err)
		}
		return sig
	})
}

func signDPoPProof(t *testing.T, header map[string]any, method, htu, ath, jti string, iat time.Time, sign func([]byte) []byte) string {
	t.Helper()
	claims := map[string]any{
		"jti": jti,
		"htm": method,
		"htu": htu,
		"iat": iat.Unix(),
	}
	if ath != "" {
		claims["ath"] = ath
	}
	h, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	c, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	input := []byte(b64(h) + "." + b64(c))
	return string(input) + "." + b64(sign(input))
}

func padBigInt(v *big.Int, size int) []byte {
	b := v.Bytes()
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func postTokenWithDPoP(t *testing.T, ts *httptest.Server, form url.Values, proof string, wantStatus int) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		t.Fatalf("new token request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("token status = %d, want %d: %s", resp.StatusCode, wantStatus, body)
	}
	var out map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decode token response status %d: %v body=%s", resp.StatusCode, err, body)
		}
	}
	return out
}

func userinfoStatus(t *testing.T, ts *httptest.Server, auth, proof string, wantStatus int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/userinfo", nil)
	if err != nil {
		t.Fatalf("new userinfo request: %v", err)
	}
	req.Header.Set("Authorization", auth)
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("userinfo request: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("userinfo status = %d, want %d (%s proof=%t)", resp.StatusCode, wantStatus, auth, proof != "")
	}
}

func uniqueDPoPJTI(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
