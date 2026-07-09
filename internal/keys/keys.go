package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func GenerateRSA(kid string, now time.Time) (idpstore.SigningKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return idpstore.SigningKey{}, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return idpstore.SigningKey{ID: kid, Algorithm: "RS256", PrivateKeyPEM: pemBytes, CreatedAt: now, NotBefore: now, Active: true}, nil
}

func ParseRSAPrivateKey(key idpstore.SigningKey) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(key.PrivateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("missing PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use,omitempty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func PublicJWK(key idpstore.SigningKey) (JWK, error) {
	priv, err := ParseRSAPrivateKey(key)
	if err != nil {
		return JWK{}, err
	}
	pub := &priv.PublicKey
	return JWK{Kty: "RSA", Use: "sig", Kid: key.ID, Alg: key.Algorithm, N: b64(pub.N.Bytes()), E: b64(big.NewInt(int64(pub.E)).Bytes())}, nil
}

func PublicJWKS(signingKeys []idpstore.SigningKey) (JWKS, error) {
	out := JWKS{Keys: make([]JWK, 0, len(signingKeys))}
	for _, k := range signingKeys {
		jwk, err := PublicJWK(k)
		if err != nil {
			return JWKS{}, err
		}
		out.Keys = append(out.Keys, jwk)
	}
	return out, nil
}

func ThumbprintJWK(j JWK) string {
	sum := sha256.Sum256([]byte(j.Kty + ":" + j.Kid + ":" + j.N + ":" + j.E))
	return b64(sum[:])
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// SignJWT signs a compact JWT with RS256 using the given signing key.
func SignJWT(key idpstore.SigningKey, claims map[string]any) (string, error) {
	priv, err := ParseRSAPrivateKey(key)
	if err != nil {
		return "", err
	}
	header := map[string]any{"typ": "JWT", "alg": "RS256", "kid": key.ID}
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
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return input + "." + b64(sig), nil
}
