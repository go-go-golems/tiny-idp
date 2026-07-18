package fositeadapter

import (
	"context"
	"strings"
	"testing"
)

type recordingLimiter struct {
	keys   []string
	reject string
}

func (l *recordingLimiter) Allow(_ context.Context, key string) bool {
	l.keys = append(l.keys, key)
	return key != l.reject
}

func TestLoginRateLimitUsesAccountClientAndAddressWithoutLoginDisclosure(t *testing.T) {
	limiter := &recordingLimiter{reject: "login:client:client-1"}
	provider := &Provider{rateLimiter: limiter}
	if provider.allowLogin(context.Background(), "client-1", "192.0.2.10", "Alice") {
		t.Fatal("login accepted despite rejected client bucket")
	}
	if len(limiter.keys) != 3 {
		t.Fatalf("rate-limit keys = %#v", limiter.keys)
	}
	wantPrefixes := []string{"login:account:", "login:client:client-1", "login:address:192.0.2.10"}
	for index, prefix := range wantPrefixes {
		if !strings.HasPrefix(limiter.keys[index], prefix) {
			t.Fatalf("key[%d] = %q, want prefix %q", index, limiter.keys[index], prefix)
		}
		if strings.Contains(strings.ToLower(limiter.keys[index]), "alice") {
			t.Fatalf("rate-limit key leaks login: %q", limiter.keys[index])
		}
	}
}

func TestTokenPreAuthenticationRateLimitUsesOnlyStableAddress(t *testing.T) {
	limiter := &recordingLimiter{}
	provider := &Provider{rateLimiter: limiter}
	if !provider.allowTokenPreAuthentication(context.Background(), "192.0.2.10") {
		t.Fatal("pre-authentication limiter unexpectedly rejected request")
	}
	want := []string{"token:address:192.0.2.10"}
	if len(limiter.keys) != len(want) {
		t.Fatalf("rate-limit keys = %#v", limiter.keys)
	}
	for i := range want {
		if limiter.keys[i] != want[i] {
			t.Fatalf("key[%d] = %q, want %q", i, limiter.keys[i], want[i])
		}
	}
}
