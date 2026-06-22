// Package scenario models the behavior of a login in the mock IdP.
//
// A Scenario bundles a synthetic user with optional failure hooks for each
// OIDC stage (authorization, token, ID-token claims, userinfo). The mock
// IdP resolves a typed login to a Scenario via a Registry, then each
// handler consults the relevant field on that Scenario instead of branching
// on the login string. This is what makes adding a new failure case a
// one-entry change rather than edits in every handler.
//
// Normal logins (e.g. "alice", "bob") resolve to a Scenario whose only
// non-zero field is User. Failure logins (e.g. "id-expired", "userinfo-401")
// set the matching failure fields.
package scenario

import (
	"time"

	"github.com/manuel/tinyidp/internal/user"
)

// Scenario describes what should happen (or fail) for a given login.
type Scenario struct {
	Name        string
	Description string
	Category    string // used by the login page to group buttons (Phase 3)
	User        user.User

	// AuthError, when non-empty, is the OAuth error code returned at the
	// /authorize endpoint via redirect (e.g. "access_denied"). The code is
	// never issued.
	AuthError string

	// TokenError, when non-empty, selects a failure at /token:
	//   "invalid_grant"  -> 400 invalid_grant
	//   "server_error"   -> 500 server_error
	//   "slow"            -> sleep 10s, then succeed normally
	TokenError string

	// UserInfoError, when non-empty, selects a failure at /userinfo:
	//   "401"            -> 401
	//   "500"            -> 500
	//   "sub_mismatch"   -> 200 with a different sub than the ID token
	UserInfoError string

	// MutateClaims, when non-nil, mutates the ID token claims after they are
	// built (e.g. set exp in the past for id-expired, wrong aud, etc.).
	MutateClaims func(claims map[string]any, now time.Time)
}

// Registry maps a normalized login to a Scenario. The zero value is not
// usable; construct with New.
type Registry struct {
	m     map[string]Scenario
	fallback func(string) Scenario
}

// Lookup returns the Scenario for a login and whether an explicit scenario
// matched. When no explicit scenario matches, a fallback normal user is
// derived from the login (so any arbitrary username still works).
func (r *Registry) Lookup(login string) (Scenario, bool) {
	if sc, ok := r.m[login]; ok {
		return sc, true
	}
	return r.fallback(login), false
}

// All returns every explicitly registered scenario, in insertion order. Used
// by the login page (Phase 3) to render selectable buttons.
func (r *Registry) All() []Scenario {
	out := make([]Scenario, 0, len(r.m))
	for _, sc := range r.m {
		out = append(out, sc)
	}
	return out
}

// Register adds or replaces a scenario keyed by its Name. This is the single
// entry point for extending the mock's behavior: callers (including tests)
// add a scenario here rather than editing handlers.
func (r *Registry) Register(sc Scenario) {
	r.m[sc.Name] = sc
}

// New returns a Registry preloaded with the built-in scenarios. Phase 4 will
// add the failure scenarios; for now this registers the normal users and
// sets up the fallback so arbitrary logins keep working.
func New() *Registry {
	r := &Registry{
		m: map[string]Scenario{},
		fallback: func(login string) Scenario {
			return Scenario{
				Name:        login,
				Description: "synthetic user derived from login",
				User:        user.FromLogin(login),
			}
		},
	}
	for _, sc := range builtinScenarios() {
		r.m[sc.Name] = sc
	}
	return r
}

// builtinScenarios returns the explicitly-registered scenarios. Phase 1 ships
// only the normal users; Phase 4 appends the failure scenarios.
func builtinScenarios() []Scenario {
	return []Scenario{
		// --- Normal users ---
		{
			Name:        "alice",
			Description: "normal user",
			Category:    "Normal users",
			User:        user.FromLogin("alice"),
		},
		{
			Name:        "bob",
			Description: "normal user",
			Category:    "Normal users",
			User:        user.FromLogin("bob"),
		},

		// --- Authorization endpoint failures (redirect back with OAuth error) ---
		{
			Name:        "fail-access-denied",
			Description: "authorization error: access_denied",
			Category:    "Authorization failures",
			User:        user.FromLogin("fail-access-denied"),
			AuthError:   "access_denied",
		},
		{
			Name:        "fail-login-required",
			Description: "authorization error: login_required",
			Category:    "Authorization failures",
			User:        user.FromLogin("fail-login-required"),
			AuthError:   "login_required",
		},
		{
			Name:        "fail-consent-required",
			Description: "authorization error: consent_required",
			Category:    "Authorization failures",
			User:        user.FromLogin("fail-consent-required"),
			AuthError:   "consent_required",
		},
		{
			Name:        "fail-server-error",
			Description: "authorization error: server_error",
			Category:    "Authorization failures",
			User:        user.FromLogin("fail-server-error"),
			AuthError:   "server_error",
		},

		// --- Token endpoint failures ---
		{
			Name:        "token-invalid-grant",
			Description: "token exchange fails: invalid_grant",
			Category:    "Token failures",
			User:        user.FromLogin("token-invalid-grant"),
			TokenError:  "invalid_grant",
		},
		{
			Name:        "token-server-error",
			Description: "token endpoint returns 500: server_error",
			Category:    "Token failures",
			User:        user.FromLogin("token-server-error"),
			TokenError:  "server_error",
		},
		// token-slow intentionally omitted from quick-pick (10s sleep is
		// painful in manual UI); it still resolves via Lookup.
		{
			Name:        "token-slow",
			Description: "token endpoint sleeps 10s then succeeds",
			User:        user.FromLogin("token-slow"),
			TokenError:  "slow",
		},

		// --- ID token claim mutations ---
		{
			Name:        "id-expired",
			Description: "expired ID token (exp in the past)",
			Category:    "ID token failures",
			User:        user.FromLogin("id-expired"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				claims["exp"] = now.Add(-1 * time.Hour).Unix()
			},
		},
		{
			Name:        "id-wrong-aud",
			Description: "wrong audience in ID token",
			Category:    "ID token failures",
			User:        user.FromLogin("id-wrong-aud"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				claims["aud"] = "some-other-client"
			},
		},
		{
			Name:        "id-wrong-iss",
			Description: "wrong issuer in ID token",
			Category:    "ID token failures",
			User:        user.FromLogin("id-wrong-iss"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				claims["iss"] = claims["iss"].(string) + "/wrong"
			},
		},
		{
			Name:        "id-missing-email",
			Description: "ID token omits email claims",
			Category:    "ID token failures",
			User:        user.FromLogin("id-missing-email"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				delete(claims, "email")
				delete(claims, "email_verified")
			},
		},
		{
			Name:        "id-email-unverified",
			Description: "email_verified = false",
			Category:    "ID token failures",
			User:        user.FromLogin("id-email-unverified"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				claims["email_verified"] = false
			},
		},
		{
			Name:        "id-bad-nonce",
			Description: "nonce mismatch in ID token",
			Category:    "ID token failures",
			User:        user.FromLogin("id-bad-nonce"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				if _, ok := claims["nonce"]; ok {
					claims["nonce"] = "wrong-nonce"
				}
			},
		},
		{
			Name:        "id-future-iat",
			Description: "iat/auth_time in the future",
			Category:    "ID token failures",
			User:        user.FromLogin("id-future-iat"),
			MutateClaims: func(claims map[string]any, now time.Time) {
				future := now.Add(10 * time.Minute).Unix()
				claims["iat"] = future
				claims["auth_time"] = future
			},
		},

		// --- UserInfo endpoint failures ---
		{
			Name:         "userinfo-401",
			Description:  "userinfo returns 401",
			Category:     "UserInfo failures",
			User:         user.FromLogin("userinfo-401"),
			UserInfoError: "401",
		},
		{
			Name:         "userinfo-500",
			Description:  "userinfo returns 500",
			Category:     "UserInfo failures",
			User:         user.FromLogin("userinfo-500"),
			UserInfoError: "500",
		},
		{
			Name:         "userinfo-sub-mismatch",
			Description:  "userinfo sub differs from ID token",
			Category:     "UserInfo failures",
			User:         user.FromLogin("userinfo-sub-mismatch"),
			UserInfoError: "sub_mismatch",
		},
	}
}

// CategoryGroup is a labeled set of scenarios, used by the login page to
// render grouped quick-pick buttons.
type CategoryGroup struct {
	Label string
	Items []Scenario
}

// Grouped returns all registered scenarios bucketed by Category, preserving
// the first-seen order of categories. The login page renders one section per
// group. Scenarios with an empty Category are skipped (they are not
// quick-pickable; the fallback covers arbitrary logins via the text input).
func (r *Registry) Grouped() []CategoryGroup {
	var order []string
	buckets := map[string][]Scenario{}
	for _, sc := range r.m {
		if sc.Category == "" {
			continue
		}
		if _, seen := buckets[sc.Category]; !seen {
			order = append(order, sc.Category)
		}
		buckets[sc.Category] = append(buckets[sc.Category], sc)
	}
	out := make([]CategoryGroup, 0, len(order))
	for _, label := range order {
		out = append(out, CategoryGroup{Label: label, Items: buckets[label]})
	}
	return out
}
