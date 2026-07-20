package idp

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// TrustedProxyHTTPConfig describes an internal HTTP listener whose public
// HTTPS transport is terminated by a specific, trusted reverse proxy.
// PublicOrigin is deliberately an origin (not an arbitrary URL): protocol
// identity is configured once and is never derived from forwarding headers.
type TrustedProxyHTTPConfig struct {
	PublicOrigin string
	Resolver     *TrustedProxyResolver
}

// NewTrustedProxyHTTPHandler validates the reverse-proxy transport contract
// before passing a request to next. It rejects untrusted peers, non-HTTPS
// forwarded transport, and a Host/X-Forwarded-Host that differs from the
// configured public origin. On success it presents the request to application
// code as TLS-backed, allowing same-origin checks to use the public scheme.
func NewTrustedProxyHTTPHandler(cfg TrustedProxyHTTPConfig, next http.Handler) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("next handler is required")
	}
	origin, err := canonicalHTTPSOrigin(cfg.PublicOrigin)
	if err != nil {
		return nil, err
	}
	if cfg.Resolver == nil || !cfg.Resolver.ProductionReady() {
		return nil, fmt.Errorf("trusted proxy resolver is required")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trusted, peerErr := cfg.Resolver.TrustsRequestPeer(r)
		if peerErr != nil || !trusted {
			http.Error(w, "untrusted proxy peer", http.StatusBadRequest)
			return
		}
		if values := r.Header.Values("X-Forwarded-Proto"); len(values) != 1 || strings.TrimSpace(values[0]) != "https" {
			http.Error(w, "trusted proxy must forward HTTPS transport", http.StatusBadRequest)
			return
		}
		if r.Host != origin.Host || !forwardedHostMatches(r.Header.Values("X-Forwarded-Host"), origin.Host) {
			http.Error(w, "request host does not match configured public origin", http.StatusBadRequest)
			return
		}
		clone := r.Clone(r.Context())
		clone.URL.Scheme = "https"
		clone.URL.Host = origin.Host
		clone.TLS = &tls.ConnectionState{}
		next.ServeHTTP(w, clone)
	}), nil
}

func canonicalHTTPSOrigin(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("public origin must be a canonical HTTPS origin")
	}
	parsed.Path = ""
	return parsed, nil
}

func forwardedHostMatches(values []string, expected string) bool {
	if len(values) == 0 {
		return true
	}
	return len(values) == 1 && strings.TrimSpace(values[0]) == expected
}
