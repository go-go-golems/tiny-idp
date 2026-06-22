package cmds

import (
	"testing"

	"github.com/manuel/tinyidp/internal/sections/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildClientRegistryMergesBuiltin verifies the merge behavior of
// buildClientRegistry: when the configured client_id matches a builtin, the
// builtin's properties (RequirePKCE, Secret) are preserved and the configured
// redirect URIs are added. This is the resolution to the open question from
// diary Step 10.
func TestBuildClientRegistryMergesBuiltin(t *testing.T) {
	t.Run("public-spa keeps RequirePKCE and adds configured redirect URI", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "public-spa",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, ok := r.Lookup("public-spa")
		require.True(t, ok)
		assert.True(t, c.RequirePKCE, "RequirePKCE must be preserved from the builtin")
		assert.Equal(t, "", c.Secret, "public-spa must stay public")
		// Builtin's redirect URI + configured one.
		assert.Contains(t, c.RedirectURIs, "http://localhost:8080/callback")
		assert.Contains(t, c.RedirectURIs, "http://localhost:9090/cb")
	})

	t.Run("web-app keeps builtin secret when none configured", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "web-app",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, _ := r.Lookup("web-app")
		assert.Equal(t, "dev-secret", c.Secret, "web-app must keep builtin dev-secret")
	})

	t.Run("web-app configured secret overrides builtin", func(t *testing.T) {
		cfg := &oidc.Settings{
			ClientID:     "web-app",
			ClientSecret: "custom-secret",
			RedirectURIs: []string{"http://localhost:9090/cb"},
		}
		r := buildClientRegistry(cfg)
		c, _ := r.Lookup("web-app")
		assert.Equal(t, "custom-secret", c.Secret)
	})
}

// TestBuildClientRegistryRegistersNewPermissiveClient verifies that a
// configured client_id that does NOT match a builtin registers a new
// permissive client (the Phase 0-4 single-client behavior for custom IDs).
func TestBuildClientRegistryRegistersNewPermissiveClient(t *testing.T) {
	cfg := &oidc.Settings{
		ClientID:     "my-custom-app",
		ClientSecret: "s",
		RedirectURIs: []string{"http://localhost:8080/cb"},
	}
	r := buildClientRegistry(cfg)
	c, ok := r.Lookup("my-custom-app")
	require.True(t, ok)
	assert.False(t, c.RequirePKCE, "a new configured client is permissive (no PKCE requirement)")
	assert.Equal(t, "s", c.Secret)
	assert.Equal(t, []string{"http://localhost:8080/cb"}, c.RedirectURIs)
}

// TestBuildClientRegistryDefaultKeepsBuiltins verifies the default case
// (dev-client, no overrides) still has all three builtins present and dev-client
// unchanged by the merge (its configured defaults equal its builtin defaults).
func TestBuildClientRegistryDefaultKeepsBuiltins(t *testing.T) {
	cfg := &oidc.Settings{
		ClientID:     "dev-client",
		RedirectURIs: []string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback"},
	}
	r := buildClientRegistry(cfg)
	for _, id := range []string{"dev-client", "public-spa", "web-app"} {
		_, ok := r.Lookup(id)
		assert.True(t, ok, "builtin %q must remain in registry", id)
	}
}
