package main

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// externalOIDCConfig is the security boundary for a Message Desk deployment
// that treats tiny-idp as an independent HTTP service. It deliberately does
// not contain identity-store, account, or provider-handler references.
type externalOIDCConfig struct {
	PublicBaseURL      string
	Issuer             string
	ClientID           string
	EndSessionEndpoint string
	CookieSecure       bool
}

func (c externalOIDCConfig) validate() error {
	publicBaseURL, err := normalizePublicBaseURL(c.PublicBaseURL)
	if err != nil {
		return errors.Wrap(err, "external relying-party public base URL")
	}
	issuer, err := normalizeExternalIssuer(c.Issuer)
	if err != nil {
		return err
	}
	if publicBaseURL == issuer {
		return errors.New("external issuer must use a distinct origin from the relying party")
	}
	if strings.TrimSpace(c.ClientID) == "" || len(c.ClientID) > 256 {
		return errors.New("external OIDC client ID is required and must be at most 256 bytes")
	}
	if strings.HasPrefix(publicBaseURL, "https://") != c.CookieSecure {
		return errors.New("external relying-party cookie security must match the public URL scheme")
	}
	if c.EndSessionEndpoint == "" {
		return nil
	}
	endpoint, err := normalizeExternalIssuer(c.EndSessionEndpoint)
	if err != nil {
		return errors.Wrap(err, "external end-session endpoint")
	}
	issuerURL, _ := url.Parse(issuer)
	endpointURL, _ := url.Parse(endpoint)
	if issuerURL.Scheme != endpointURL.Scheme || issuerURL.Host != endpointURL.Host {
		return errors.New("external end-session endpoint must share the issuer origin")
	}
	return nil
}

func normalizeExternalIssuer(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.Wrap(err, "parse external issuer")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", errors.New("external issuer must be an absolute HTTP(S) URL without query, fragment, or credentials")
	}
	origin, err := normalizePublicBaseURL(parsed.Scheme + "://" + parsed.Host)
	if err != nil {
		return "", errors.Wrap(err, "external issuer origin")
	}
	path := strings.TrimSuffix(parsed.EscapedPath(), "/")
	if path == "/" {
		path = ""
	}
	if strings.Contains(path, "//") {
		return "", errors.New("external issuer path must be canonical")
	}
	return origin + path, nil
}
