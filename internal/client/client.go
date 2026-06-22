// Package client defines the OIDC client registry for the mock IdP.
//
// A Client is a configured relying party: an ID, an optional secret, an
// allowlist of redirect URIs, whether PKCE is required, and the scopes it may
// request. The registry maps client IDs to Client structs.
//
// Phase 0-4 used a single hardcoded client (the OIDC_* / TINYIDP_CLIENT_ID
// env var). Phase 5 introduces a registry so a single running provider can
// serve multiple relying parties — a public SPA client (PKCE-only), a
// confidential web-app client (secret required), and a permissive default
// client for quick testing.
package client

import (
	"strings"
)

// Client is a configured relying party.
type Client struct {
	// ID is the client_id accepted at /authorize and /token.
	ID string
	// Secret is the client_secret. If empty, the client is treated as public
	// (no secret check at /token); if non-empty, /token enforces it via
	// client_secret_basic or client_secret_post.
	Secret string
	// RedirectURIs is the allowlist of redirect URIs. A request with a
	// redirect_uri not in this set is rejected before any redirect.
	RedirectURIs []string
	// RequirePKCE, when true, forces /token to reject a code exchange that
	// did not include a code_challenge at /authorize and a code_verifier at
	// /token. Public clients should set this.
	RequirePKCE bool
	// AllowedScopes is the set of scopes this client may request. If empty,
	// all scopes the provider supports are allowed (permissive). When
	// non-empty, /authorize rejects a scope request containing a scope not in
	// this set.
	AllowedScopes []string
	// PostLogoutRedirectURIs is the allowlist of URIs the RP may redirect to
	// after RP-initiated logout (Phase 11). If empty, no post-logout redirect
	// is permitted for this client and /end-session returns a logged-out page.
	PostLogoutRedirectURIs []string
}

// AllowsRedirectURI reports whether uri is in the client's allowlist.
func (c *Client) AllowsRedirectURI(uri string) bool {
	for _, allowed := range c.RedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// AllowsScope reports whether every space-separated scope in requested is
// permitted for this client. An empty AllowedScopes means all scopes are
// allowed (permissive), which preserves the Phase 0-4 default behavior.
func (c *Client) AllowsScope(requested string) bool {
	if len(c.AllowedScopes) == 0 {
		return true
	}
	for _, s := range strings.Fields(requested) {
		if !contains(c.AllowedScopes, s) {
			return false
		}
	}
	return true
}

// AllowsPostLogoutRedirectURI reports whether uri is in the client's
// post-logout redirect allowlist (Phase 11).
func (c *Client) AllowsPostLogoutRedirectURI(uri string) bool {
	for _, allowed := range c.PostLogoutRedirectURIs {
		if allowed == uri {
			return true
		}
	}
	return false
}

// Registry maps client IDs to Client structs.
type Registry struct {
	clients map[string]Client
}

// NewRegistry returns a Registry preloaded with the built-in clients:
//
//   - dev-client: permissive default (no secret, PKCE optional, all scopes).
//   - public-spa: public client, PKCE required, no secret.
//   - web-app: confidential client, secret required, PKCE optional.
//
// Additional clients can be registered with Register.
func NewRegistry() *Registry {
	r := &Registry{clients: map[string]Client{}}
	for _, c := range BuiltinClients() {
		r.clients[c.ID] = c
	}
	return r
}

// Lookup returns the Client for a client ID and whether it exists.
func (r *Registry) Lookup(id string) (Client, bool) {
	c, ok := r.clients[id]
	return c, ok
}

// Register adds or replaces a client keyed by its ID.
func (r *Registry) Register(c Client) {
	r.clients[c.ID] = c
}

// All returns every registered client.
func (r *Registry) All() []Client {
	out := make([]Client, 0, len(r.clients))
	for _, c := range r.clients {
		out = append(out, c)
	}
	return out
}

// BuiltinClients returns the default clients the mock IdP ships with.
func BuiltinClients() []Client {
	return []Client{
		{
			ID:                     "dev-client",
			Secret:                 "",
			RedirectURIs:           []string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback"},
			PostLogoutRedirectURIs: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
			RequirePKCE:            false,
		},
		{
			ID:                     "public-spa",
			Secret:                 "",
			RedirectURIs:           []string{"http://localhost:8080/callback"},
			PostLogoutRedirectURIs: []string{"http://localhost:8080"},
			RequirePKCE:            true,
		},
		{
			ID:                     "web-app",
			Secret:                 "dev-secret",
			RedirectURIs:           []string{"http://localhost:8080/callback"},
			PostLogoutRedirectURIs: []string{"http://localhost:8080"},
			RequirePKCE:            false,
		},
	}
}

// Merge returns a new Client that starts from base and applies non-empty
// values from override. RedirectURIs are unioned and deduplicated.
//
// This is the merge semantics used when a configured client ID matches a
// builtin: the builtin's properties (RequirePKCE, Secret, AllowedScopes) are
// preserved, the configured redirect URIs are added to the builtin's, and a
// non-empty configured Secret overrides the builtin's. RequirePKCE and
// AllowedScopes have no configured override in the OIDC section, so they are
// always taken from base.
//
// The override's ID wins (it is the configured client_id).
func Merge(base, override Client) Client {
	out := base
	out.ID = override.ID
	if override.Secret != "" {
		out.Secret = override.Secret
	}
	out.RedirectURIs = unionStrings(base.RedirectURIs, override.RedirectURIs)
	out.PostLogoutRedirectURIs = unionStrings(base.PostLogoutRedirectURIs, override.PostLogoutRedirectURIs)
	// RequirePKCE and AllowedScopes are kept from base (no override fields).
	return out
}

func contains(slice []string, s string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}
	return false
}

// unionStrings returns the deduplicated union of a and b, preserving
// first-seen order (a's elements first, then b's new elements).
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
