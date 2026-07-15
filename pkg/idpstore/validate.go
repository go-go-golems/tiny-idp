package idpstore

import (
	"net"
	"net/url"
	"strings"
)

// Validate checks client invariants under the given mode.
func (c Client) Validate(mode Mode) error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrEmptyClientID
	}
	if c.Public {
		if len(c.SecretHash) > 0 {
			return ErrPublicClientHasSecret
		}
		if !c.RequirePKCE {
			return ErrPublicClientRequiresPKCE
		}
	} else if mode == ProductionMode && len(c.SecretHash) == 0 {
		return ErrConfidentialMissingSecret
	}
	if len(c.AllowedGrantTypes) == 0 {
		return ErrClientMissingGrantTypes
	}
	seenGrantTypes := make(map[string]struct{}, len(c.AllowedGrantTypes))
	for _, grantType := range c.AllowedGrantTypes {
		if !supportedGrantType(grantType) {
			return ErrClientGrantTypeInvalid
		}
		if _, seen := seenGrantTypes[grantType]; seen {
			return ErrClientGrantTypeDuplicate
		}
		seenGrantTypes[grantType] = struct{}{}
	}
	for _, ru := range c.RedirectURIs {
		if err := ValidateRedirectURI(ru, mode); err != nil {
			return err
		}
	}
	for _, ru := range c.PostLogoutRedirectURIs {
		if err := ValidateRedirectURI(ru, mode); err != nil {
			return err
		}
	}
	return nil
}

// AllowsGrantType reports whether the client has explicitly been granted use
// of the OAuth grant type. Empty client grant lists never authorize a grant.
func (c Client) AllowsGrantType(grantType string) bool {
	for _, allowed := range c.AllowedGrantTypes {
		if allowed == grantType {
			return true
		}
	}
	return false
}

func supportedGrantType(grantType string) bool {
	switch grantType {
	case GrantAuthorizationCode, GrantRefreshToken, GrantDeviceCode:
		return true
	default:
		return false
	}
}

// AllowsRedirectURI reports whether uri exactly matches one registered URI.
func (c Client) AllowsRedirectURI(uri string) bool {
	for _, allowed := range c.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// AllowsScope reports whether every requested scope is allowed for the client.
// Empty AllowedScopes means no scopes are allowed in production domain code; the
// mock engine can retain its own permissive legacy behavior separately.
func (c Client) AllowsScope(requested []string) bool {
	allowed := make(map[string]struct{}, len(c.AllowedScopes))
	for _, s := range c.AllowedScopes {
		allowed[s] = struct{}{}
	}
	for _, s := range requested {
		if _, ok := allowed[s]; !ok {
			return false
		}
	}
	return true
}

func ValidateRedirectURI(raw string, mode Mode) error {
	if strings.TrimSpace(raw) == "" {
		return ErrEmptyRedirectURI
	}
	if strings.Contains(raw, "*") {
		return ErrWildcardRedirectURI
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrInvalidRedirectURI
	}
	if u.Fragment != "" {
		return ErrRedirectURIFragment
	}
	if mode == ProductionMode && u.Scheme != "https" && !isLoopbackHost(u.Hostname()) {
		return ErrProductionRedirectHTTP
	}
	return nil
}

func (u User) Validate() error {
	if strings.TrimSpace(u.Sub) == "" {
		return ErrEmptySubject
	}
	if u.Email != "" && strings.EqualFold(strings.TrimSpace(u.Sub), strings.TrimSpace(u.Email)) {
		return ErrSubjectUsesEmail
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
