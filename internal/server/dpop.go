package server

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const dpopProofMaxAge = 5 * time.Minute

type dpopProof struct {
	JKT string
	JTI string
	ATH string
}

type dpopHeader struct {
	Typ string         `json:"typ"`
	Alg string         `json:"alg"`
	JWK map[string]any `json:"jwk"`
}

type dpopClaims struct {
	JTI string `json:"jti"`
	HTM string `json:"htm"`
	HTU string `json:"htu"`
	IAT int64  `json:"iat"`
	ATH string `json:"ath,omitempty"`
}

func tokenTypeForJKT(jkt string) string {
	if jkt != "" {
		return "DPoP"
	}
	return "Bearer"
}

func (s *Server) dpopProofForTokenRequest(w http.ResponseWriter, r *http.Request) (dpopProof, bool) {
	raw := strings.TrimSpace(r.Header.Get("DPoP"))
	if raw == "" {
		return dpopProof{}, true
	}
	proof, err := s.validateDPoPProof(r, raw, "")
	if err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_dpop_proof", err.Error())
		return dpopProof{}, false
	}
	return proof, true
}

func (s *Server) validateDPoPProof(r *http.Request, raw, accessToken string) (dpopProof, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return dpopProof{}, errors.New("DPoP proof must be a compact JWT")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return dpopProof{}, errors.New("invalid DPoP proof header encoding")
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return dpopProof{}, errors.New("invalid DPoP proof payload encoding")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return dpopProof{}, errors.New("invalid DPoP proof signature encoding")
	}

	var header dpopHeader
	dec := json.NewDecoder(strings.NewReader(string(headerBytes)))
	dec.UseNumber()
	if err := dec.Decode(&header); err != nil {
		return dpopProof{}, errors.New("invalid DPoP proof header JSON")
	}
	if strings.ToLower(header.Typ) != "dpop+jwt" {
		return dpopProof{}, errors.New("DPoP proof typ must be dpop+jwt")
	}
	if header.Alg != "ES256" && header.Alg != "RS256" {
		return dpopProof{}, errors.New("unsupported DPoP proof alg")
	}
	if hasPrivateJWKMembers(header.JWK) {
		return dpopProof{}, errors.New("DPoP proof jwk must not contain private key material")
	}

	var claims dpopClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return dpopProof{}, errors.New("invalid DPoP proof payload JSON")
	}
	if claims.JTI == "" {
		return dpopProof{}, errors.New("DPoP proof missing jti")
	}
	if strings.ToUpper(claims.HTM) != r.Method {
		return dpopProof{}, errors.New("DPoP proof htm mismatch")
	}
	if claims.HTU != requestURLWithoutQuery(r) {
		return dpopProof{}, errors.New("DPoP proof htu mismatch")
	}
	if claims.IAT == 0 {
		return dpopProof{}, errors.New("DPoP proof missing iat")
	}
	iat := time.Unix(claims.IAT, 0)
	now := time.Now()
	if iat.Before(now.Add(-dpopProofMaxAge)) || iat.After(now.Add(dpopProofMaxAge)) {
		return dpopProof{}, errors.New("DPoP proof iat is outside the allowed freshness window")
	}
	if accessToken != "" {
		wantATH := accessTokenHash(accessToken)
		if claims.ATH == "" {
			return dpopProof{}, errors.New("DPoP proof missing ath")
		}
		if claims.ATH != wantATH {
			return dpopProof{}, errors.New("DPoP proof ath mismatch")
		}
	}

	input := parts[0] + "." + parts[1]
	jkt, err := verifyDPoPSignatureAndThumbprint(header, []byte(input), sig)
	if err != nil {
		return dpopProof{}, err
	}
	if !s.rememberDPoPJTI(jkt, claims.JTI, iat.Add(dpopProofMaxAge)) {
		return dpopProof{}, errors.New("DPoP proof replay detected")
	}
	return dpopProof{JKT: jkt, JTI: claims.JTI, ATH: claims.ATH}, nil
}

func verifyDPoPSignatureAndThumbprint(header dpopHeader, input, sig []byte) (string, error) {
	sum := sha256.Sum256(input)
	switch header.Alg {
	case "ES256":
		pub, x, y, err := ecJWK(header.JWK)
		if err != nil {
			return "", err
		}
		if len(sig) != 64 {
			return "", errors.New("invalid ES256 DPoP proof signature length")
		}
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		if !ecdsa.Verify(pub, sum[:], r, s) {
			return "", errors.New("invalid DPoP proof signature")
		}
		return jwkThumbprint(map[string]string{
			"crv": "P-256",
			"kty": "EC",
			"x":   x,
			"y":   y,
		})
	case "RS256":
		pub, n, e, err := rsaJWK(header.JWK)
		if err != nil {
			return "", err
		}
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
			return "", errors.New("invalid DPoP proof signature")
		}
		return jwkThumbprint(map[string]string{
			"e":   e,
			"kty": "RSA",
			"n":   n,
		})
	default:
		return "", errors.New("unsupported DPoP proof alg")
	}
}

func ecJWK(jwk map[string]any) (*ecdsa.PublicKey, string, string, error) {
	if jwkString(jwk, "kty") != "EC" || jwkString(jwk, "crv") != "P-256" {
		return nil, "", "", errors.New("DPoP ES256 proof requires P-256 EC jwk")
	}
	xs, ys := jwkString(jwk, "x"), jwkString(jwk, "y")
	if xs == "" || ys == "" {
		return nil, "", "", errors.New("DPoP EC jwk missing x or y")
	}
	xb, err := base64.RawURLEncoding.DecodeString(xs)
	if err != nil {
		return nil, "", "", errors.New("invalid DPoP EC jwk x")
	}
	yb, err := base64.RawURLEncoding.DecodeString(ys)
	if err != nil {
		return nil, "", "", errors.New("invalid DPoP EC jwk y")
	}
	x, y := new(big.Int).SetBytes(xb), new(big.Int).SetBytes(yb)
	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return nil, "", "", errors.New("DPoP EC jwk point is not on P-256")
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, xs, ys, nil
}

func rsaJWK(jwk map[string]any) (*rsa.PublicKey, string, string, error) {
	if jwkString(jwk, "kty") != "RSA" {
		return nil, "", "", errors.New("DPoP RS256 proof requires RSA jwk")
	}
	ns, es := jwkString(jwk, "n"), jwkString(jwk, "e")
	if ns == "" || es == "" {
		return nil, "", "", errors.New("DPoP RSA jwk missing n or e")
	}
	nb, err := base64.RawURLEncoding.DecodeString(ns)
	if err != nil {
		return nil, "", "", errors.New("invalid DPoP RSA jwk n")
	}
	eb, err := base64.RawURLEncoding.DecodeString(es)
	if err != nil {
		return nil, "", "", errors.New("invalid DPoP RSA jwk e")
	}
	e := 0
	for _, b := range eb {
		e = e<<8 + int(b)
	}
	if e < 3 {
		return nil, "", "", errors.New("invalid DPoP RSA exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, ns, es, nil
}

func jwkString(jwk map[string]any, key string) string {
	v, _ := jwk[key].(string)
	return v
}

func hasPrivateJWKMembers(jwk map[string]any) bool {
	for _, key := range []string{"d", "p", "q", "dp", "dq", "qi", "oth"} {
		if _, ok := jwk[key]; ok {
			return true
		}
	}
	return false
}

func jwkThumbprint(fields map[string]string) (string, error) {
	b, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return b64(sum[:]), nil
}

func accessTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return b64(sum[:])
}

func requestURLWithoutQuery(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.EscapedPath())
}

func (s *Server) rememberDPoPJTI(jkt, jti string, expires time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, exp := range s.dpopReplay {
		if now.After(exp) {
			delete(s.dpopReplay, key)
		}
	}
	key := jkt + "\x00" + jti
	if _, exists := s.dpopReplay[key]; exists {
		return false
	}
	s.dpopReplay[key] = expires
	return true
}
