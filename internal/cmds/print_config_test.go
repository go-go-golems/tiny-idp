package cmds

import (
	"context"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	cmd_sources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/manuel/tinyidp/internal/sections/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureProcessor collects emitted rows so a test can assert on the output
// of a GlazeCommand without a real formatter.
type captureProcessor struct {
	rows []types.Row
}

func (c *captureProcessor) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *captureProcessor) Close(_ context.Context) error { return nil }

// rowVal reads a field from a row, returning "" if absent.
func rowVal(row types.Row, key string) string {
	v, ok := row.Get(key)
	if !ok || v == nil {
		return ""
	}
	return v.(string)
}

// buildOidcSchema builds a schema containing only the OIDC section, used to
// resolve values for print-config tests.
func buildOidcSchema(t *testing.T) *schema.Schema {
	t.Helper()
	section, err := oidc.NewSection()
	require.NoError(t, err)
	return schema.NewSchema(schema.WithSections(section))
}

// TestPrintConfigEmitsResolvedDefaults verifies print-config composes the OIDC
// section, resolves defaults, and emits a single row with the resolved config.
// This is the "reusable section" contract: a second command composes the same
// oidc section as serve and produces the same resolved config.
func TestPrintConfigEmitsResolvedDefaults(t *testing.T) {
	cmd, err := NewPrintConfigCommand()
	require.NoError(t, err)

	pls := buildOidcSchema(t)
	parsed := values.New()
	require.NoError(t, cmd_sources.Execute(pls, parsed, cmd_sources.FromDefaults()))

	proc := &captureProcessor{}
	require.NoError(t, cmd.RunIntoGlazeProcessor(context.Background(), parsed, proc))

	require.Len(t, proc.rows, 1)
	row := proc.rows[0]
	assert.Equal(t, "http://localhost:5556", rowVal(row, "issuer"))
	assert.Equal(t, "127.0.0.1:5556", rowVal(row, "addr"))
	assert.Equal(t, "dev-client", rowVal(row, "client_id"))
	assert.Equal(t, "", rowVal(row, "users_file"))
	assert.Equal(t, "mock", rowVal(row, "engine"))
}

// TestPrintConfigReflectsEnvOverride verifies that values resolved above
// defaults (here: env) are reflected in the emitted row. This pins the
// contract that print-config shows what serve would actually use, not just
// the defaults.
func TestPrintConfigReflectsEnvOverride(t *testing.T) {
	t.Setenv("TINYIDP_CLIENT_ID", "seen-by-print-config")
	t.Setenv("TINYIDP_USERS_FILE", "/tmp/tinyidp-users.yaml")

	cmd, err := NewPrintConfigCommand()
	require.NoError(t, err)

	pls := buildOidcSchema(t)
	parsed := values.New()
	require.NoError(t, cmd_sources.Execute(pls, parsed,
		cmd_sources.FromEnv("TINYIDP"),
		cmd_sources.FromDefaults(),
	))

	proc := &captureProcessor{}
	require.NoError(t, cmd.RunIntoGlazeProcessor(context.Background(), parsed, proc))

	require.Len(t, proc.rows, 1)
	assert.Equal(t, "seen-by-print-config", rowVal(proc.rows[0], "client_id"))
	assert.Equal(t, "/tmp/tinyidp-users.yaml", rowVal(proc.rows[0], "users_file"))
}

// TestPrintConfigReflectsProfileOverride verifies that a profile value
// surfaces in print-config output. This is the full chain: the same oidc
// section, resolved with a profile, emitted as a row.
func TestPrintConfigReflectsProfileOverride(t *testing.T) {
	// Use the profile machinery from profiles_test.go (writeProfiles +
	// ProfileMiddlewaresFunc) to resolve the oidc section, then run
	// print-config against the resolved values.
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

	cmd, err := NewPrintConfigCommand()
	require.NoError(t, err)
	proc := &captureProcessor{}
	require.NoError(t, cmd.RunIntoGlazeProcessor(context.Background(), parsed, proc))

	require.Len(t, proc.rows, 1)
	assert.Equal(t, "dev-profile-client", rowVal(proc.rows[0], "client_id"))
}

// Ensure captureProcessor satisfies middlewares.Processor at compile time.
var _ middlewares.Processor = (*captureProcessor)(nil)
