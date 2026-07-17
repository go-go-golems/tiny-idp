package scenario

import (
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/user"
)

func TestNewPreloadsBuiltinScenarios(t *testing.T) {
	r := New()
	sc, ok := r.Lookup("alice")
	if !ok {
		t.Fatal("alice should be a builtin scenario")
	}
	if sc.User.Sub != user.FromLogin("alice").Sub {
		t.Fatalf("alice sub = %v", sc.User.Sub)
	}
	if sc.AuthError != "" || sc.TokenError != "" || sc.UserInfoError != "" || sc.MutateClaims != nil {
		t.Fatalf("alice should be a normal scenario, got %+v", sc)
	}
}

func TestFallbackForArbitraryLogin(t *testing.T) {
	r := New()
	sc, ok := r.Lookup("nobody-defined-this")
	if ok {
		t.Fatal("arbitrary login should not be an explicit scenario")
	}
	if sc.User.Sub != user.FromLogin("nobody-defined-this").Sub {
		t.Fatalf("fallback sub = %v", sc.User.Sub)
	}
}

func TestLookupAppliesNormalize(t *testing.T) {
	r := New()
	// The registry is keyed on normalized logins (lowercased/trimmed). The
	// caller is expected to normalize before lookup; verify lowercase works.
	sc, ok := r.Lookup("alice")
	if !ok || sc.Name != "alice" {
		t.Fatalf("Lookup(alice) = %+v ok=%v", sc, ok)
	}
}

func TestAllReturnsEveryBuiltin(t *testing.T) {
	r := New()
	all := r.All()
	if len(all) < 2 {
		t.Fatalf("expected at least 2 builtin scenarios, got %d", len(all))
	}
	names := map[string]bool{}
	for _, sc := range all {
		names[sc.Name] = true
	}
	for _, want := range []string{"alice", "bob"} {
		if !names[want] {
			t.Fatalf("builtin missing %q", want)
		}
	}
}

func TestMutateClaimsHookIsCallable(t *testing.T) {
	// A scenario with a MutateClaims hook should mutate the claims map in
	// place. This is the contract token() relies on.
	sc := Scenario{
		Name: "test-mutator",
		User: user.FromLogin("test-mutator"),
		MutateClaims: func(claims map[string]any, now time.Time) {
			claims["custom"] = "injected"
			claims["exp"] = now.Add(-time.Hour).Unix()
		},
	}
	claims := map[string]any{"sub": "x"}
	sc.MutateClaims(claims, time.Now())
	if claims["custom"] != "injected" {
		t.Fatal("MutateClaims did not set custom")
	}
	exp, _ := claims["exp"].(int64)
	if exp >= time.Now().Unix() {
		t.Fatalf("exp should be in the past, got %d", exp)
	}
}
