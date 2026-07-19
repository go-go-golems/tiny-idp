// Package idpidentity contains native identity candidates that may be durable
// or virtual. A virtual candidate is an authenticated subject, not a database
// row and not a script-controlled OIDC session.
package idpidentity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const (
	maxSubjectBytes = 255
	maxClaimBytes   = 512
	subjectDomain   = "tiny-idp/virtual-subject/v1\x00"
)

// Kind identifies how a candidate is materialized. Only a durable identity
// has a local user row; virtual candidates are derived from verified evidence.
type Kind string

const (
	KindDurable Kind = "durable"
	KindVirtual Kind = "virtual"
)

// Candidate is the bounded native representation of an identity selected by a
// provider. It intentionally has no arbitrary claim map, token fields, or
// protocol-owned claims.
type Candidate struct {
	Kind              Kind
	Subject           string
	DisplayName       string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Groups            []string
	Roles             []string
	Tenant            string
	Locale            string
	Version           string
}

// VirtualRequest describes a provider-approved virtual identity. Namespace
// and seed are used only by DeriveSubject; seed is never returned in a subject
// or projected claim.
type VirtualRequest struct {
	Namespace         string
	Seed              string
	DisplayName       string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Groups            []string
	Roles             []string
	Tenant            string
	Locale            string
	Version           string
}

// SubjectDeriver derives stable pairwise subject identifiers from a host-held
// key. JavaScript may name a namespace and provide verified evidence, but it
// never receives this key or chooses the final subject string.
type SubjectDeriver struct{ key []byte }

func NewSubjectDeriver(key []byte) (*SubjectDeriver, error) {
	if len(key) < 32 {
		return nil, errors.New("virtual subject derivation key must be at least 32 bytes")
	}
	return &SubjectDeriver{key: append([]byte(nil), key...)}, nil
}

func (d *SubjectDeriver) Derive(namespace, seed string) (string, error) {
	if d == nil || len(d.key) < 32 {
		return "", errors.New("virtual subject deriver is unavailable")
	}
	namespace = strings.TrimSpace(namespace)
	seed = strings.TrimSpace(seed)
	if !validBounded(namespace) || !validBounded(seed) {
		return "", errors.New("virtual subject namespace and seed are required bounded UTF-8 strings")
	}
	mac := hmac.New(sha256.New, d.key)
	_, _ = mac.Write([]byte(subjectDomain))
	_, _ = mac.Write([]byte(namespace))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(seed))
	return "v1_" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// NewVirtual validates a provider-approved request and derives a subject. It
// cannot create a local account or issue protocol artifacts.
func NewVirtual(deriver *SubjectDeriver, request VirtualRequest) (Candidate, error) {
	subject, err := deriver.Derive(request.Namespace, request.Seed)
	if err != nil {
		return Candidate{}, err
	}
	candidate := Candidate{Kind: KindVirtual, Subject: subject, DisplayName: strings.TrimSpace(request.DisplayName), Email: strings.TrimSpace(request.Email), EmailVerified: request.EmailVerified, PreferredUsername: strings.TrimSpace(request.PreferredUsername), Groups: clean(request.Groups), Roles: clean(request.Roles), Tenant: strings.TrimSpace(request.Tenant), Locale: strings.TrimSpace(request.Locale), Version: strings.TrimSpace(request.Version)}
	if err := candidate.Validate(); err != nil {
		return Candidate{}, err
	}
	return candidate, nil
}

func (c Candidate) Validate() error {
	if c.Kind != KindDurable && c.Kind != KindVirtual {
		return errors.New("identity candidate has invalid kind")
	}
	if len(c.Subject) == 0 || len(c.Subject) > maxSubjectBytes || !utf8.ValidString(c.Subject) {
		return errors.New("identity candidate subject is invalid")
	}
	for _, value := range []string{c.DisplayName, c.Email, c.PreferredUsername, c.Tenant, c.Locale, c.Version} {
		if len(value) > maxClaimBytes || !utf8.ValidString(value) {
			return errors.New("identity candidate contains an invalid claim value")
		}
	}
	return nil
}

// ProfileClaims projects only provider-owned profile claims. OIDC protocol
// claims such as iss, sub, aud, exp, iat, nonce, auth_time, acr, and amr stay
// in the Fosite adapter and cannot be overwritten by a virtual provider.
func (c Candidate) ProfileClaims() map[string]any {
	claims := map[string]any{}
	if c.DisplayName != "" {
		claims["name"] = c.DisplayName
	}
	if c.Email != "" {
		claims["email"] = c.Email
		claims["email_verified"] = c.EmailVerified
	}
	if c.PreferredUsername != "" {
		claims["preferred_username"] = c.PreferredUsername
	}
	if len(c.Groups) != 0 {
		claims["groups"] = append([]string(nil), c.Groups...)
	}
	if len(c.Roles) != 0 {
		claims["roles"] = append([]string(nil), c.Roles...)
	}
	if c.Tenant != "" {
		claims["tenant"] = c.Tenant
	}
	if c.Locale != "" {
		claims["locale"] = c.Locale
	}
	return claims
}

func validBounded(value string) bool {
	return len(value) != 0 && len(value) <= maxClaimBytes && utf8.ValidString(value)
}

func clean(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && len(value) <= maxClaimBytes && utf8.ValidString(value) {
			result = append(result, value)
		}
	}
	return result
}
