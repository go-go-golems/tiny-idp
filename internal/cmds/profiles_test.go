package cmds

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	cmd_sources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/tiny-idp/internal/sections/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildProfileSchema builds a schema containing the OIDC section plus the
// profile-settings section, mirroring what BuildCobraCommand assembles.
func buildProfileSchema(t *testing.T) *schema.Schema {
	t.Helper()
	oidcSection, err := oidc.NewSection()
	require.NoError(t, err)
	profileSection, err := cli.NewProfileSettingsSection()
	require.NoError(t, err)
	return schema.NewSchema(schema.WithSections(oidcSection, profileSection))
}

// writeProfiles writes a profiles.yaml with the given content to a temp dir
// and returns its path.
func writeProfiles(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "profiles.yaml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

// setProfileSelection populates the profile-settings section in parsed as if
// the bootstrap parse (cobra + defaults) had resolved --profile / --profile-file.
// This mirrors what cli.ParseCommandSettingsSection produces before our
// MiddlewaresFunc runs.
func setProfileSelection(t *testing.T, parsed *values.Values, schema_ *schema.Schema, profile, profileFile string) {
	t.Helper()
	profSection, ok := schema_.Get(cli.ProfileSettingsSlug)
	require.True(t, ok, "profile-settings section must be in schema")
	sv := parsed.GetOrCreate(profSection)
	setField(t, sv, profSection, "profile", profile)
	setField(t, sv, profSection, "profile-file", profileFile)
}

// setField sets a single field value on a SectionValues, looking up the
// definition on the section so the FieldValue carries the right type info.
func setField(t *testing.T, sv *values.SectionValues, section schema.Section, name, val string) {
	t.Helper()
	def, ok := section.GetDefinitions().Get(name)
	require.True(t, ok, "definition %s missing", name)
	fv := &fields.FieldValue{Definition: def}
	require.NoError(t, fv.Update(val))
	sv.Fields.Set(name, fv)
}

// TestProfileOverridesDefaults is the core profile contract: a field set in a
// profile wins over the section default. This pins the "profiles sit above
// defaults" placement in the middleware chain.
func TestProfileOverridesDefaults(t *testing.T) {
	profileFile := writeProfiles(t, `
dev:
  oidc:
    client-id: dev-profile-client
`)
	parsed := values.New()
	pls := buildProfileSchema(t)
	setProfileSelection(t, parsed, pls, "dev", profileFile)

	mws, err := ProfileMiddlewaresFunc("tinyidp", nil)(parsed, nil, nil)
	require.NoError(t, err)
	require.NoError(t, cmd_sources.Execute(pls, parsed, mws...))

	s, err := oidc.GetSettings(parsed)
	require.NoError(t, err)
	assert.Equal(t, "dev-profile-client", s.ClientID, "profile should override default")
}

// TestProfileEnvOverridesProfile pins the second tier of precedence: an env
// var (TINYIDP_*) overrides the profile value, because env is applied after
// profiles in the chain.
func TestProfileEnvOverridesProfile(t *testing.T) {
	profileFile := writeProfiles(t, `
dev:
  oidc:
    client-id: dev-profile-client
`)
	t.Setenv("TINYIDP_CLIENT_ID", "env-wins")

	parsed := values.New()
	pls := buildProfileSchema(t)
	setProfileSelection(t, parsed, pls, "dev", profileFile)

	mws, err := ProfileMiddlewaresFunc("tinyidp", nil)(parsed, nil, nil)
	require.NoError(t, err)
	require.NoError(t, cmd_sources.Execute(pls, parsed, mws...))

	s, err := oidc.GetSettings(parsed)
	require.NoError(t, err)
	assert.Equal(t, "env-wins", s.ClientID, "env should override profile")
}

// TestProfileMissingDefaultFileSkipsSilently verifies the no-profiles.yaml
// case: when the default profile file is missing and the requested profile is
// "default", profile loading is skipped silently rather than erroring. This is
// what makes `tinyidp serve` work out of the box with no profiles.yaml.
func TestProfileMissingDefaultFileSkipsSilently(t *testing.T) {
	// Point the default file at a non-existent path inside a temp dir. Because
	// the requested profile file equals the default and the profile name is
	// "default", GatherFlagsFromProfiles must skip rather than error.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	parsed := values.New()
	pls := buildProfileSchema(t)
	// No --profile / --profile-file set: resolveProfileSelection falls back to
	// "default" + the (missing) default file.
	setProfileSelection(t, parsed, pls, "", "")

	mws, err := ProfileMiddlewaresFunc("tinyidp", nil)(parsed, nil, nil)
	require.NoError(t, err)
	// Execute must not error despite the missing default file.
	require.NoError(t, cmd_sources.Execute(pls, parsed, mws...))

	s, err := oidc.GetSettings(parsed)
	require.NoError(t, err)
	// Defaults preserved (no profile overrode them).
	assert.Equal(t, "dev-client", s.ClientID)
}

// TestProfileExplicitMissingFileErrors verifies that an explicitly-requested
// profile file that does not exist produces an error (not a silent skip).
func TestProfileExplicitMissingFileErrors(t *testing.T) {
	parsed := values.New()
	pls := buildProfileSchema(t)
	setProfileSelection(t, parsed, pls, "dev", "/nonexistent/path/profiles.yaml")

	mws, err := ProfileMiddlewaresFunc("tinyidp", nil)(parsed, nil, nil)
	require.NoError(t, err)
	err = cmd_sources.Execute(pls, parsed, mws...)
	require.Error(t, err, "explicitly missing profile file should error")
}
