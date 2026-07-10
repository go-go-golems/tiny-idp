// Package idp defines the public runtime and policy contracts used to embed
// tiny-idp. The package deliberately contains no Fosite types.
package idp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
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

func (DirectClientAddressResolver) ProductionReady() bool { return true }

// TrustedProxyConfig defines which immediate/intermediate proxies may supply
// X-Forwarded-For. Untrusted peers never influence the resolved address.
type TrustedProxyConfig struct {
	TrustedCIDRs []string
	MaxHops      int
}

type TrustedProxyResolver struct {
	trusted []*net.IPNet
	maxHops int
}

var _ ClientAddressResolver = (*TrustedProxyResolver)(nil)
var _ ProductionReadyReporter = (*TrustedProxyResolver)(nil)

func NewTrustedProxyResolver(cfg TrustedProxyConfig) (*TrustedProxyResolver, error) {
	if len(cfg.TrustedCIDRs) == 0 {
		return nil, fmt.Errorf("at least one trusted proxy CIDR is required")
	}
	if cfg.MaxHops <= 0 {
		cfg.MaxHops = 8
	}
	resolver := &TrustedProxyResolver{maxHops: cfg.MaxHops}
	for _, raw := range cfg.TrustedCIDRs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy CIDR %q: %w", raw, err)
		}
		resolver.trusted = append(resolver.trusted, network)
	}
	return resolver, nil
}

func (r *TrustedProxyResolver) ResolveClientAddress(req *http.Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is required")
	}
	peer, err := remoteIP(req.RemoteAddr)
	if err != nil {
		return "", err
	}
	if r == nil || !r.isTrusted(peer) {
		return peer.String(), nil
	}
	values := strings.Split(req.Header.Get("X-Forwarded-For"), ",")
	if len(values) == 1 && strings.TrimSpace(values[0]) == "" {
		return peer.String(), nil
	}
	if len(values) > r.maxHops {
		return "", fmt.Errorf("forwarded address chain exceeds %d hops", r.maxHops)
	}
	chain := make([]net.IP, 0, len(values)+1)
	for _, value := range values {
		ip := net.ParseIP(strings.TrimSpace(value))
		if ip == nil {
			return "", fmt.Errorf("invalid forwarded client address")
		}
		chain = append(chain, ip)
	}
	chain = append(chain, peer)
	for i := len(chain) - 1; i >= 0; i-- {
		if !r.isTrusted(chain[i]) {
			return chain[i].String(), nil
		}
	}
	return chain[0].String(), nil
}

func (r *TrustedProxyResolver) ProductionReady() bool {
	return r != nil && len(r.trusted) > 0 && r.maxHops > 0
}

func (r *TrustedProxyResolver) isTrusted(ip net.IP) bool {
	for _, network := range r.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func remoteIP(address string) (net.IP, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse remote address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("remote address is not an IP")
	}
	return ip, nil
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
	User idpstore.User
	AMR  []string
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

// MaintenanceStatus is the host-visible lifecycle state of retention work.
type MaintenanceStatus struct {
	LastStartedAt  time.Time
	LastFinishedAt time.Time
	LastSuccessAt  time.Time
	LastError      string
	LastReport     idpstore.MaintenanceReport
}
