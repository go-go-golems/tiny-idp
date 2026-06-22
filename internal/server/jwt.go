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
)

// discovery returns the OIDC provider metadata. A compliant RP only needs the
// issuer and endpoint URLs to configure itself; the rest advertises what the
// mock supports (RS256 only, authorization_code only, S256/plain PKCE).
func (s *Server) discovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/authorize",
		"token_endpoint":                        s.issuer + "/token",
		"userinfo_endpoint":                     s.issuer + "/userinfo",
		"jwks_uri":                              s.issuer + "/jwks",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "email", "email_verified", "name"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
	})
}

// jwks returns the public half of the signing key as a JWK set. Clients fetch
// this to verify ID token signatures. The private key is never exposed.
func (s *Server) jwks(w http.ResponseWriter, r *http.Request) {
	pub := s.key.PublicKey
	e := big.NewInt(int64(pub.E)).Bytes()

	writeJSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"use": "sig",
			"kid": s.kid,
			"alg": "RS256",
			"n":   b64(pub.N.Bytes()),
			"e":   b64(e),
		}},
	})
}

// signJWT produces an RS256-signed JWT for the given claims.
//
//	header  = {"typ":"JWT","alg":"RS256","kid": s.kid}
//	input   = base64url(header) + "." + base64url(claims)
//	sig     = RS256(input, private_key)   // PKCS#1v15 over SHA-256(input)
//	jwt     = input + "." + base64url(sig)
func (s *Server) signJWT(claims map[string]any) (string, error) {
	header := map[string]any{
		"typ": "JWT",
		"alg": "RS256",
		"kid": s.kid,
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

	sig, err := rsa.SignPKCS1v15(cryptoRandReader, s.key, crypto.SHA256, sum[:])
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
