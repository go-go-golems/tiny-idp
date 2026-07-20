// Package idpinvite contains native invitation evidence primitives. Scripts
// may choose which validated evidence to accept, but cannot verify arbitrary
// signatures, declare an expiry valid, or mark an invite consumed.
package idpinvite

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	signedVersion = "v1"
	maxTokenBytes = 8 << 10
)

// SignedClaims are native-verified stateless invitation restrictions. A signed
// invitation has expiry and audience binding but deliberately makes no
// one-time-use or immediate-individual-revocation promise.
type SignedClaims struct {
	ID            string    `json:"id"`
	Issuer        string    `json:"issuer"`
	Audience      string    `json:"audience"`
	ExpiresAt     time.Time `json:"expiresAt"`
	NotBefore     time.Time `json:"notBefore,omitempty"`
	PolicyVersion string    `json:"policyVersion"`
	Subject       string    `json:"subject,omitempty"`
	Email         string    `json:"email,omitempty"`
}

// Verified is safe provider input/evidence after signature and policy checks.
// Raw token text and signing material are intentionally absent.
type Verified struct {
	ID            string
	Issuer        string
	Audience      string
	ExpiresAt     time.Time
	PolicyVersion string
	Subject       string
	Email         string
}

// KeyRing maps a bounded key identifier to an HMAC key. Rotating/removing a
// key revokes every token signed by it; this is the explicit revocation model
// for stateless signed invitations.
type KeyRing struct {
	keys map[string][]byte
}

func NewKeyRing(keys map[string][]byte) (*KeyRing, error) {
	if len(keys) == 0 {
		return nil, errors.New("signed invitation key ring is required")
	}
	result := &KeyRing{keys: make(map[string][]byte, len(keys))}
	for id, key := range keys {
		if !validText(id) || len(key) < 32 {
			return nil, errors.New("signed invitation key identifiers and keys are invalid")
		}
		result.keys[id] = append([]byte(nil), key...)
	}
	return result, nil
}

// Sign exists for native operator tooling and deterministic tests. Browser
// code and JavaScript providers receive only Verify.
func (r *KeyRing) Sign(keyID string, claims SignedClaims) (string, error) {
	if r == nil {
		return "", errors.New("signed invitation key ring is unavailable")
	}
	key, ok := r.keys[keyID]
	if !ok {
		return "", errors.New("signed invitation signing key is unavailable")
	}
	if err := validateClaims(claims, time.Time{}); err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", errors.Wrap(err, "encode signed invitation claims")
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	prefix := signedVersion + "." + keyID + "." + encoded
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(prefix))
	return prefix + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Verify validates signature, issuer, audience, expiry, policy version, and
// optional subject/email constraints. A caller supplies those expectations;
// none are derived from the untrusted token alone.
func (r *KeyRing) Verify(raw string, expected Verification) (Verified, error) {
	if r == nil {
		return Verified{}, errors.New("signed invitation key ring is unavailable")
	}
	if len(raw) == 0 || len(raw) > maxTokenBytes {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 4 || parts[0] != signedVersion {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	key, ok := r.keys[parts[1]]
	if !ok {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	prefix := strings.Join(parts[:3], ".")
	signature, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(prefix))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(payload) == 0 {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	var claims SignedClaims
	if err := decoder.Decode(&claims); err != nil {
		return Verified{}, errors.New("signed invitation token is invalid")
	}
	if err := validateClaims(claims, expected.Now); err != nil {
		return Verified{}, err
	}
	if claims.Issuer != expected.Issuer || claims.Audience != expected.Audience || claims.PolicyVersion != expected.PolicyVersion || (expected.Subject != "" && claims.Subject != expected.Subject) || (expected.Email != "" && !strings.EqualFold(claims.Email, expected.Email)) {
		return Verified{}, errors.New("signed invitation is not accepted")
	}
	return Verified{ID: claims.ID, Issuer: claims.Issuer, Audience: claims.Audience, ExpiresAt: claims.ExpiresAt.UTC(), PolicyVersion: claims.PolicyVersion, Subject: claims.Subject, Email: claims.Email}, nil
}

// Verification is host-owned expected policy. Now is injected for tests;
// production callers pass time.Now().UTC().
type Verification struct {
	Issuer        string
	Audience      string
	PolicyVersion string
	Subject       string
	Email         string
	Now           time.Time
}

func validateClaims(claims SignedClaims, now time.Time) error {
	if !validText(claims.ID) || !validText(claims.Issuer) || !validText(claims.Audience) || !validText(claims.PolicyVersion) || claims.ExpiresAt.IsZero() {
		return errors.New("signed invitation claims are invalid")
	}
	if claims.Subject != "" && !validText(claims.Subject) || claims.Email != "" && !validText(claims.Email) {
		return errors.New("signed invitation claims are invalid")
	}
	if !now.IsZero() && (!claims.ExpiresAt.After(now) || (!claims.NotBefore.IsZero() && claims.NotBefore.After(now))) {
		return errors.New("signed invitation is expired or not active")
	}
	return nil
}

func validText(value string) bool { return strings.TrimSpace(value) != "" && len(value) <= 512 }
