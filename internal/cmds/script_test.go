package cmds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmd_sources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scriptTestValues(t *testing.T, command *ScriptTestCommand, source string) *values.Values {
	t.Helper()
	parsed := values.New()
	require.NoError(t, cmd_sources.Execute(command.Schema, parsed,
		cmd_sources.FromMap(map[string]map[string]interface{}{
			"default": {"source": source, "profile": "signup"},
		}),
	))
	return parsed
}

func writeScriptTestSource(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "signup.js")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	return path
}

func rowBool(t *testing.T, rowKey string, row interface {
	Get(string) (interface{}, bool)
}) bool {
	t.Helper()
	value, ok := row.Get(rowKey)
	require.True(t, ok, "row is missing %q", rowKey)
	parsed, ok := value.(bool)
	require.True(t, ok, "row %q is %T, want bool", rowKey, value)
	return parsed
}

func TestScriptTestCommandEmitsStableSuccessRow(t *testing.T) {
	command, err := NewScriptTestCommand()
	require.NoError(t, err)
	path := writeScriptTestSource(t, idpsignup.EmailVerifiedSource)
	processor := &captureProcessor{}

	require.NoError(t, command.RunIntoGlazeProcessor(context.Background(), scriptTestValues(t, command, path), processor))
	require.Len(t, processor.rows, 1)
	assert.Equal(t, "signup-start-presents-identity", rowVal(processor.rows[0], "id"))
	assert.True(t, rowBool(t, "passed", processor.rows[0]))
	assert.Equal(t, "present", rowVal(processor.rows[0], "expected_kind"))
	assert.Equal(t, "present", rowVal(processor.rows[0], "actual_kind"))
}

func TestScriptTestCommandEmitsFailureRowThenReturnsError(t *testing.T) {
	command, err := NewScriptTestCommand()
	require.NoError(t, err)
	failing := strings.Replace(idpsignup.EmailVerifiedSource, `outcomes: ["present"]`, `outcomes: ["present", "deny"]`, 1)
	failing = strings.Replace(failing, `expectedKind:"present"`, `expectedKind:"deny"`, 1)
	path := writeScriptTestSource(t, failing)
	processor := &captureProcessor{}

	err = command.RunIntoGlazeProcessor(context.Background(), scriptTestValues(t, command, path), processor)
	require.EqualError(t, err, `script test "signup-start-presents-identity" failed: expected outcome "deny", got "present"`)
	require.Len(t, processor.rows, 1)
	assert.False(t, rowBool(t, "passed", processor.rows[0]))
	assert.Equal(t, "deny", rowVal(processor.rows[0], "expected_kind"))
	assert.Equal(t, "present", rowVal(processor.rows[0], "actual_kind"))
}
