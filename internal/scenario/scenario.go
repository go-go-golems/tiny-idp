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

	// ExtraClaims are merged into the ID token claims (and the userinfo
	// response) after the base claims are built and before MutateClaims runs.
	// This is the declarative way to emit claim variants (groups, roles,
	// tenant, preferred_username, locale) without writing a MutateClaims
	// closure for each. MutateClaims (if also set) runs after ExtraClaims and
	// can override them.
	ExtraClaims map[string]any

	// OmitClaims is a list of claim names to delete from both the ID token
	// and the userinfo response after ExtraClaims are merged. This is the
	// declarative way to model "this user has no X claim" as a positive
	// scenario (distinct from the Phase 4 id-* MutateClaims mutators, which
	// affect the ID token only and are meant to test ID-token validation).
	// MutateClaims runs after OmitClaims and can re-add a claim if needed.
	OmitClaims []string

	// SignKey selects which signing key the ID token is signed with (Phase 10):
	//   ""             -> the active key (kid dev-key-1), published in JWKS
	//   "rotated"      -> a second published key (kid rotated-key-2); the token
	//                     verifies, which tests that the RP looks up the kid
	//                     rather than assuming a single key
	//   "unknown-kid"  -> signs with a kid that is NOT published in JWKS, so the
	//                     RP cannot find a key to verify against (kid-not-found)
	//   "bad-sig"      -> signs with a key whose private half does NOT match the
	//                     public half published under that kid, so signature
	//                     verification fails (the kid is found but the sig is wrong)
	// This is distinct from MutateClaims: MutateClaims corrupts claim *values*,
	// while SignKey corrupts the *signature/key binding*.
	SignKey string
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

		// --- Phase 7: claim variants (groups, roles, tenant, etc.) ---
		// These emit extra claims so relying parties can test role/tenant
		// mapping and weird claim shapes. ExtraClaims are merged into both the
		// ID token and the userinfo response, so the two agree under normal
		// scenarios.
		{
			Name:        "admin",
			Description: "user with groups:[admin, engineering] and roles:[owner]",
			Category:    "Claim variants",
			User:        user.FromLogin("admin"),
			ExtraClaims: map[string]any{
				"groups":              []string{"admin", "engineering"},
				"roles":               []string{"owner"},
				"preferred_username": "admin",
			},
		},
		{
			Name:        "viewer",
			Description: "user with groups:[viewer] and roles:[reader]",
			Category:    "Claim variants",
			User:        user.FromLogin("viewer"),
			ExtraClaims: map[string]any{
				"groups":              []string{"viewer"},
				"roles":               []string{"reader"},
				"preferred_username": "viewer",
			},
		},
		{
			Name:        "no-groups",
			Description: "user with no groups or roles claims",
			Category:    "Claim variants",
			User:        user.FromLogin("no-groups"),
			// No groups/roles claims at all — tests RPs that require them.
		},
		{
			Name:        "many-groups",
			Description: "user with many groups (stress claim parsing)",
			Category:    "Claim variants",
			User:        user.FromLogin("many-groups"),
			ExtraClaims: map[string]any{
				"groups": []string{"g1", "g2", "g3", "g4", "g5", "g6", "g7", "g8"},
			},
		},
		{
			Name:        "tenant-a-admin",
			Description: "user in tenant-a with admin group",
			Category:    "Claim variants",
			User:        user.FromLogin("tenant-a-admin"),
			ExtraClaims: map[string]any{
				"groups":  []string{"admin"},
				"tenant": "tenant-a",
			},
		},
		{
			Name:        "tenant-b-viewer",
			Description: "user in tenant-b with viewer group",
			Category:    "Claim variants",
			User:        user.FromLogin("tenant-b-viewer"),
			ExtraClaims: map[string]any{
				"groups":  []string{"viewer"},
				"tenant": "tenant-b",
			},
		},
		{
			Name:        "unicode-name",
			Description: "user with a unicode display name and locale",
			Category:    "Claim variants",
			User:        user.FromLogin("unicode-name"),
			ExtraClaims: map[string]any{
				"name":  "Müller Frédéric",
				"locale": "de-DE",
			},
		},
		{
			Name:        "no-email",
			Description: "user with no email/email_verified claims (positive scenario)",
			Category:    "Claim variants",
			User:        user.FromLogin("no-email"),
			// Omit email claims from BOTH the ID token and userinfo, to test RPs
			// that handle missing email. (Distinct from id-missing-email, which
			// is an ID-token-only mutation via MutateClaims and tests ID-token
			// validation specifically.)
			OmitClaims: []string{"email", "email_verified"},
		},
		{
			Name:        "unverified-email",
			Description: "user with email_verified = false (positive scenario)",
			Category:    "Claim variants",
			User:        user.FromLogin("unverified-email"),
			ExtraClaims: map[string]any{
				"email_verified": false,
			},
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

		// --- JWKS / key rotation (Phase 10) ---
		// These corrupt the signature/key binding, not claim values. The JWKS
		// endpoint publishes three kids (dev-key-1, rotated-key-2, bad-sig-key);
		// each scenario picks a different signing behavior. JWKS-level failures
		// (500/slow/empty) are server modes toggled via the debug UI, not
		// per-user scenarios, because /jwks is global and not tied to a login.
		{
			Name:        "key-rotated",
			Description: "ID token signed with a second published key (kid rotated-key-2); verifies against JWKS",
			Category:    "JWKS / key rotation",
			User:        user.FromLogin("key-rotated"),
			SignKey:     "rotated",
		},
		{
			Name:        "kid-not-found",
			Description: "ID token kid is not present in JWKS (RP cannot find a key)",
			Category:    "JWKS / key rotation",
			User:        user.FromLogin("kid-not-found"),
			SignKey:     "unknown-kid",
		},
		{
			Name:        "bad-signature",
			Description: "ID token kid is in JWKS but signature was made with a different key",
			Category:    "JWKS / key rotation",
			User:        user.FromLogin("bad-signature"),
			SignKey:     "bad-sig",
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
