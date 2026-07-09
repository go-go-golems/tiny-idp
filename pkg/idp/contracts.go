// Package idp defines the public runtime and policy contracts used to embed
// tiny-idp. The package deliberately contains no Fosite types.
package idp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/manuel/tinyidp/pkg/idpstore"
)

// ConsentPolicy decides whether a user must approve requested scopes and
// records approval after the browser interaction succeeds.
type ConsentPolicy interface {
	RequireConsent(ctx context.Context, user idpstore.User, client idpstore.Client, scopes []string) (bool, error)
	RecordConsent(ctx context.Context, user idpstore.User, client idpstore.Client, scopes []string) error
}

// RateLimiter admits or rejects one operation identified by a stable,
// non-secret key. Production construction requires an explicit implementation.
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
}

// ClientAddressResolver returns a normalized client address from an HTTP
// request according to the host's trusted-proxy policy.
type ClientAddressResolver interface {
	ResolveClientAddress(r *http.Request) (string, error)
}

// DirectClientAddressResolver trusts only the immediate TCP peer and ignores
// forwarded headers. It is the safe choice when no trusted reverse proxy is in
// front of the provider.
type DirectClientAddressResolver struct{}

func (DirectClientAddressResolver) ResolveClientAddress(r *http.Request) (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is required")
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("parse remote address: %w", err)
	}
	if net.ParseIP(host) == nil {
		return "", fmt.Errorf("remote address is not an IP")
	}
	return host, nil
}

// LoginMetadata contains request context safe for authentication policy and
// audit. RemoteAddr is the immediate peer until Phase 3 introduces the trusted
// client-address resolver.
type LoginMetadata struct {
	RemoteAddr string
	UserAgent  string
	ClientID   string
}

// AuthResult is the successful result of password authentication.
type AuthResult struct {
	User               idpstore.User
	MustChangePassword bool
	AMR                []string
}

// PasswordAuthenticator verifies a login without exposing the concrete
// password service or Fosite request/session types.
type PasswordAuthenticator interface {
	AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (AuthResult, error)
}

// ReadinessCheck is one stable, non-secret provider preflight result.
type ReadinessCheck struct {
	Name      string
	Ready     bool
	Degraded  bool
	Reason    string
	CheckedAt time.Time
}

// ReadinessReport aggregates provider preflight checks.
type ReadinessReport struct {
	Ready  bool
	Checks []ReadinessCheck
}
