package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
)

// jwksSlowDelay is how long the "slow" JWKS mode sleeps before responding.
// It is a package var so tests can shorten it; the default mirrors the
// token-slow scenario (a realistic latency a developer would notice in the UI).
var jwksSlowDelay = 10 * time.Second

// rotatedKey and badSigKey are the second and third published signing keys
// (Phase 10). They are generated once and shared across all Servers because
// they exist only to demonstrate key rotation and signature failure; the
// active key (s.key) remains per-Server for process isolation. Generating
// them lazily keeps Server construction at a single RSA keygen, so the test
// suite does not pay for keys it may never use.
var (
	rotatedOnce sync.Once
	rotatedKey  *rsa.PrivateKey

	badSigOnce sync.Once
	badSigKey  *rsa.PrivateKey // published under kid "bad-sig-key"; tokens are signed with s.key (a different key), so verification fails
)

const (
	kidActive  = "dev-key-1"
	kidRotated = "rotated-key-2"
	kidBadSig  = "bad-sig-key"
	kidUnknown = "unknown-key" // never published; used by the kid-not-found scenario
)

func sharedRotatedKey() *rsa.PrivateKey {
	rotatedOnce.Do(func() {
		k, err := rsa.GenerateKey(cryptoRandReader, 2048)
		if err != nil {
			panic("tinyidp: failed to generate rotated key: " + err.Error())
		}
		rotatedKey = k
	})
	return rotatedKey
}

func sharedBadSigKey() *rsa.PrivateKey {
	badSigOnce.Do(func() {
		k, err := rsa.GenerateKey(cryptoRandReader, 2048)
		if err != nil {
			panic("tinyidp: failed to generate bad-sig key: " + err.Error())
		}
		badSigKey = k
	})
	return badSigKey
}

// discovery returns the OIDC provider metadata. A compliant RP only needs the
// issuer and endpoint URLs to configure itself; the rest advertises what the
// mock supports (RS256 only, authorization_code + refresh_token grants,
// S256/plain PKCE, prompt none/login).
func (s *Server) discovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/authorize",
		"token_endpoint":                        s.issuer + "/token",
		"device_authorization_endpoint":         s.issuer + "/device_authorization",
		"userinfo_endpoint":                     s.issuer + "/userinfo",
		"jwks_uri":                              s.issuer + "/jwks",
		"end_session_endpoint":                  s.issuer + "/end-session",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", deviceGrantType},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "email", "email_verified", "name", "groups", "roles", "tenant", "preferred_username", "locale"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
		"prompt_values_supported":               []string{"none", "login"},
	})
}

// jwks returns the public half of the signing keys as a JWK set. Clients fetch
// this to verify ID token signatures. Private keys are never exposed.
//
// Phase 10 publishes three kids (the active key plus a rotated key plus a
// bad-sig key) so RPs can be tested against multi-key JWKS. A server-level
// mode (SetJWKSMode) can make the endpoint return 500, sleep, or return an
// empty key set, to exercise RP error handling and key caching.
func (s *Server) jwks(w http.ResponseWriter, r *http.Request) {
	switch s.JWKSMode() {
	case "500":
		http.Error(w, "jwks error (simulated)", http.StatusInternalServerError)
		return
	case "slow":
		time.Sleep(jwksSlowDelay)
	case "empty":
		writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{}})
		return
	}

	keys := []map[string]string{
		jwkFor(kidActive, &s.key.PublicKey),
		jwkFor(kidRotated, &sharedRotatedKey().PublicKey),
		jwkFor(kidBadSig, &sharedBadSigKey().PublicKey),
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

// jwkFor builds a single JWK (public RSA key) for the given kid.
func jwkFor(kid string, pub *rsa.PublicKey) map[string]string {
	e := big.NewInt(int64(pub.E)).Bytes()
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": kid,
		"alg": "RS256",
		"n":   b64(pub.N.Bytes()),
		"e":   b64(e),
	}
}

// signJWT produces an RS256-signed JWT for the given claims. The signing key
// and kid are selected by the scenario's SignKey (Phase 10):
//
//	default       -> active key (dev-key-1), published
//	"rotated"     -> shared rotated key (rotated-key-2), published; verifies
//	"unknown-kid" -> active key but kid "unknown-key", NOT published (kid-not-found)
//	"bad-sig"     -> active key but kid "bad-sig-key"; JWKS publishes a DIFFERENT
//	                  key under that kid, so signature verification fails
//
//	header  = {"typ":"JWT","alg":"RS256","kid": kid}
//	input   = base64url(header) + "." + base64url(claims)
//	sig     = RS256(input, signer)   // PKCS#1v15 over SHA-256(input)
//	jwt     = input + "." + base64url(sig)
func (s *Server) signJWT(claims map[string]any, sc *scenario.Scenario) (string, error) {
	kid := s.kid
	signer := s.key
	if sc != nil {
		switch sc.SignKey {
		case "rotated":
			kid, signer = kidRotated, sharedRotatedKey()
		case "unknown-kid":
			kid = kidUnknown // signer stays s.key, but the kid is not published
		case "bad-sig":
			// Sign with the active key but claim (via header) to be bad-sig-key,
			// whose published public half is a different key. Verification fails.
			kid = kidBadSig
		}
	}

	header := map[string]any{
		"typ": "JWT",
		"alg": "RS256",
		"kid": kid,
	}

	h, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	c, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	input := b64(h) + "." + b64(c)
	sum := sha256.Sum256([]byte(input))

	sig, err := rsa.SignPKCS1v15(cryptoRandReader, signer, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return input + "." + b64(sig), nil
}

// verifyPKCE checks a code_verifier against the stored challenge.
//
//	empty challenge  -> PKCE not used, accepted
//	S256             -> base64url(sha256(verifier)) == challenge
//	"" or "plain"    -> verifier == challenge
//	anything else    -> rejected
func verifyPKCE(challenge, method, verifier string) bool {
	if challenge == "" {
		return true
	}
	if verifier == "" {
		return false
	}
	switch method {
	case "", "plain":
		return verifier == challenge
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		return b64(sum[:]) == challenge
	default:
		return false
	}
}

// cryptoRandReader is the reader used for key generation and signature
// nonces. It is a package-level indirection so tests could substitute a
// deterministic source if needed (currently they do not).
var cryptoRandReader = rand.Reader

// b64 is raw URL-safe base64 (no padding), used for JWT segments and JWK
// integer fields.
func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}
