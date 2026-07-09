package idp_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
)

func TestDefaultPasswordAcceptancePolicy(t *testing.T) {
	ctx := context.Background()
	policy := idp.DefaultPasswordAcceptancePolicy()
	if _, err := policy.NormalizeAndValidatePassword(ctx, []byte("short-password"), "alice"); !errors.Is(err, idp.ErrPasswordRejected) {
		t.Fatalf("short password error = %v", err)
	}
	accepted := []byte("a sufficiently long passphrase")
	normalized, err := policy.NormalizeAndValidatePassword(ctx, accepted, "alice")
	if err != nil || string(normalized) != string(accepted) {
		t.Fatalf("accepted password = %q, err=%v", normalized, err)
	}
	if _, err := policy.NormalizeAndValidatePassword(ctx, []byte("temporarypassword"), "alice"); !errors.Is(err, idp.ErrPasswordRejected) {
		t.Fatalf("blocklisted password error = %v", err)
	}
	if _, err := policy.NormalizeAndValidatePassword(ctx, []byte("alice123456789!"), "alice"); !errors.Is(err, idp.ErrPasswordRejected) {
		t.Fatalf("context-derived password error = %v", err)
	}
}

func TestTrustedProxyResolver(t *testing.T) {
	resolver, err := idp.NewTrustedProxyResolver(idp.TrustedProxyConfig{TrustedCIDRs: []string{"10.0.0.0/8", "192.0.2.0/24"}, MaxHops: 4})
	if err != nil {
		t.Fatal(err)
	}
	untrusted := &http.Request{RemoteAddr: "203.0.113.9:4444", Header: http.Header{"X-Forwarded-For": []string{"198.51.100.4"}}}
	got, err := resolver.ResolveClientAddress(untrusted)
	if err != nil || got != "203.0.113.9" {
		t.Fatalf("untrusted peer resolved to %q, err=%v", got, err)
	}
	trusted := &http.Request{RemoteAddr: "10.0.0.2:443", Header: http.Header{"X-Forwarded-For": []string{"198.51.100.7, 192.0.2.8"}}}
	got, err = resolver.ResolveClientAddress(trusted)
	if err != nil || got != "198.51.100.7" {
		t.Fatalf("trusted chain resolved to %q, err=%v", got, err)
	}
	malformed := &http.Request{RemoteAddr: "10.0.0.2:443", Header: http.Header{"X-Forwarded-For": []string{"not-an-ip"}}}
	if _, err := resolver.ResolveClientAddress(malformed); err == nil {
		t.Fatal("malformed forwarded address was accepted")
	}
}

func TestFixedWindowLimiterFailsClosed(t *testing.T) {
	ctx := context.Background()
	limiter := idp.NewFixedWindowRateLimiter(2, time.Hour)
	for attempt := 0; attempt < 2; attempt++ {
		if !limiter.Allow(ctx, "login:account:a") {
			t.Fatalf("valid request %d rejected before limit", attempt)
		}
	}
	if limiter.Allow(ctx, "login:account:a") {
		t.Fatal("request accepted after limit")
	}
	stats := limiter.Stats()
	if stats.Accepted != 2 || stats.Rejected != 1 || stats.Buckets != 1 {
		t.Fatalf("limiter stats = %#v", stats)
	}
}
