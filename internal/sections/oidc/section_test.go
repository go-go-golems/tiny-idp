package oidc

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildSchema builds a Glazed schema containing only the OIDC section, used
// to drive the defaults/env/flag precedence chain in tests.
func buildSchema(t *testing.T) *schema.Schema {
	t.Helper()
	section, err := NewSection()
	require.NoError(t, err)
	return schema.NewSchema(schema.WithSections(section))
}

// TestSectionShape verifies the reusable section declares the expected
// fields with the expected defaults. A downstream command composing this
// section relies on these names/types existing.
func TestSectionShape(t *testing.T) {
	section, err := NewSection()
	require.NoError(t, err)
	assert.Equal(t, Slug, section.GetSlug())
	assert.Equal(t, "OIDC Provider Configuration", section.GetName())

	defs := section.GetDefinitions()
	for _, name := range []string{"issuer", "addr", "client-id", "client-secret", "redirect-uris", "users-file"} {
		_, present := defs.Get(name)
		assert.True(t, present, "missing field %q", name)
	}
}

// TestDefaultsRoundTrip exercises the full Glazed parse chain with only
// defaults, then decodes into Settings. This proves the section's struct
// tags line up with the field names (a mismatch would surface as a zero
// value here).
func TestDefaultsRoundTrip(t *testing.T) {
	pls := buildSchema(t)
	parsed := values.New()
	require.NoError(t, sources.Execute(pls, parsed, sources.FromDefaults()))

	s, err := GetSettings(parsed)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:5556", s.Issuer)
	assert.Equal(t, "127.0.0.1:5556", s.Addr)
	assert.Equal(t, "dev-client", s.ClientID)
	assert.Equal(t, "", s.ClientSecret)
	assert.Equal(t,
		[]string{"http://localhost:3000/callback", "http://127.0.0.1:3000/callback"},
		s.RedirectURIs,
	)
	assert.Equal(t, "", s.UsersFile)
}

// TestEnvOverridesDefaults proves the AppName-based env loading (TINYIDP_*)
// overrides defaults, which is the precedence contract the README and help
// pages document.
func TestEnvOverridesDefaults(t *testing.T) {
	t.Setenv("TINYIDP_ISSUER", "http://example.test")
	t.Setenv("TINYIDP_CLIENT_ID", "env-client")
	t.Setenv("TINYIDP_ADDR", "0.0.0.0:9999")

	pls := buildSchema(t)
	parsed := values.New()
	require.NoError(t, sources.Execute(pls, parsed,
		sources.FromDefaults(),
		sources.FromEnv("tinyidp"),
	))

	s, err := GetSettings(parsed)
	require.NoError(t, err)
	assert.Equal(t, "http://example.test", s.Issuer)
	assert.Equal(t, "env-client", s.ClientID)
	assert.Equal(t, "0.0.0.0:9999", s.Addr)
}

// TestIssuerTrailingSlashStripped confirms GetSettings normalizes the
// issuer so discovery/JWKS URLs don't accumulate stray slashes.
func TestIssuerTrailingSlashStripped(t *testing.T) {
	t.Setenv("TINYIDP_ISSUER", "http://localhost:5556/")
	pls := buildSchema(t)
	parsed := values.New()
	require.NoError(t, sources.Execute(pls, parsed,
		sources.FromDefaults(),
		sources.FromEnv("tinyidp"),
	))
	s, err := GetSettings(parsed)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:5556", s.Issuer)
}
