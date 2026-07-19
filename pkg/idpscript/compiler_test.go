package idpscript_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileMaterializesImmutableProgram(t *testing.T) {
	artifact, err := idpscript.Compile(context.Background(), validSource, compileOptions())
	require.NoError(t, err)

	program := artifact.Program()
	assert.Equal(t, "community", program.Name)
	assert.Equal(t, "start", program.Workflows["signup"].EntryHandler)
	assert.Equal(t, []idpprogram.OutcomeKind{idpprogram.OutcomePresent}, program.Lambdas["signup.start"].AllowedOutcomes)
	assert.Equal(t, uint32(1), program.Capabilities["directory.lookup"].Version)
	assert.Len(t, artifact.Fingerprints().Source, 64)
	assert.Len(t, artifact.Fingerprints().CallbackRegistry, 64)

	delete(program.Lambdas, "signup.start")
	assert.Contains(t, artifact.Program().Lambdas, "signup.start", "Program must return a defensive copy")
}

func TestArtifactLoadsIntoIndependentOwnedRuntimes(t *testing.T) {
	options := compileOptions()
	artifact, err := idpscript.Compile(context.Background(), validSource, options)
	require.NoError(t, err)
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	require.NoError(t, err)

	first, err := artifact.Load(context.Background(), factory)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, first.Close(context.Background())) })
	second, err := artifact.Load(context.Background(), factory)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Close(context.Background())) })

	assert.Equal(t, []string{"signup.start", "signup.submitted"}, first.CallbackIDs())
	assert.Equal(t, first.CallbackIDs(), second.CallbackIDs())
	assert.Equal(t, artifact.Fingerprints(), first.Fingerprints())
	assert.Equal(t, first.Fingerprints(), second.Fingerprints())
}

func TestArtifactRejectsRuntimeSchemaDrift(t *testing.T) {
	options := compileOptions()
	artifact, err := idpscript.Compile(context.Background(), validSource, options)
	require.NoError(t, err)
	drifted := compileSchemas()
	schema := drifted["signupInput"]
	schema.MaxBytes++
	drifted["signupInput"] = schema
	factory, err := idpscript.NewRuntimeFactory(drifted)
	require.NoError(t, err)

	_, err = artifact.Load(context.Background(), factory)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fingerprint does not match")
}

func TestCompileRejectsAmbientModules(t *testing.T) {
	_, err := idpscript.Compile(context.Background(), `require("fs");`, compileOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambient module")
	assert.Contains(t, err.Error(), "disabled")
}

func TestCompileRejectsForgedLambdaHandle(t *testing.T) {
	source := `
const A = require("tinyidp").v1;
module.exports = A.program("forged", program => {
  program.workflow("signup", {
    version: 1,
    entry: "start",
    handlers: {start: {}},
    edges: [],
  });
});`

	_, err := idpscript.Compile(context.Background(), source, compileOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a lambda returned by this module")
}

func TestCompileRequiresProgramExport(t *testing.T) {
	source := strings.Replace(validSource, "module.exports = A.program", "const ignored = A.program", 1)

	_, err := idpscript.Compile(context.Background(), source, compileOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module.exports must be the value")
}

func TestCompileRejectsUnboundedSource(t *testing.T) {
	options := compileOptions()
	options.MaxSourceBytes = 8

	_, err := idpscript.Compile(context.Background(), validSource, options)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside 1..8")
}

func TestCompileInterruptsUnboundedDefinition(t *testing.T) {
	options := compileOptions()
	options.Timeout = 10 * time.Millisecond

	_, err := idpscript.Compile(context.Background(), `for (;;) {}`, options)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}

func compileOptions() idpscript.CompileOptions {
	options := idpscript.DefaultCompileOptions()
	options.Schemas = compileSchemas()
	return options
}

func compileSchemas() map[string]idpprogram.Schema {
	return map[string]idpprogram.Schema{
		"signupInput": {
			ID:       "signupInput",
			Kind:     idpprogram.SchemaKindObject,
			MaxBytes: 4096,
		},
		"terminalValue": {
			ID:        "terminalValue",
			Kind:      idpprogram.SchemaKindString,
			MaxBytes:  1024,
			MaxLength: 256,
		},
	}
}

const validSource = `
const A = require("tinyidp").v1;
module.exports = A.program("community", program => {
  program.capabilities({
    "directory.lookup": {version: 1},
  });
  const start = A.lambda("signup.start", {
    input: "signupInput",
    output: "terminalValue",
    outcomes: ["present"],
    effects: [],
    capabilities: [],
    timeoutMs: 250,
    maxCapabilityCalls: 0,
    maxOutputBytes: 4096,
    run: ctx => A.result.present({
      handler: "submitted",
      carry: {clientId: ctx.input.clientId},
      expiresInSeconds: 300,
    }),
  });
  const submitted = program.lambda("signup.submitted", {
    input: "signupInput",
    output: "terminalValue",
    outcomes: ["complete", "deny"],
    effects: ["read"],
    capabilities: ["directory.lookup"],
    timeoutMs: 1000,
    maxCapabilityCalls: 1,
    maxOutputBytes: 4096,
    run: ctx => A.result.complete(ctx.input),
  });
  program.workflow("signup", {
    version: 1,
    entry: "start",
    handlers: {start, submitted},
    edges: [{from: "start", outcome: "present", to: "submitted", input: "signupInput"}],
  });
});`
