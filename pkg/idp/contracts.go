// Package idp defines the public runtime and policy contracts used to embed
// tiny-idp. The package deliberately contains no Fosite types.
package idp

import (
	"context"
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
