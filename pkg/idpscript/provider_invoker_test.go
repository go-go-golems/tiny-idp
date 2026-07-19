package idpscript_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

func TestProviderInvokerUsesPinnedHandlerSchema(t *testing.T) {
	options := idpscript.DefaultCompileOptions()
	options.Schemas = map[string]idpprogram.Schema{
		"providerInput":  {ID: "providerInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 128, Additional: false, Fields: map[string]idpprogram.SchemaField{"subject": {Ref: "text", Required: true}}},
		"providerOutput": {ID: "providerOutput", Kind: idpprogram.SchemaKindObject, MaxBytes: 128, Additional: false, Fields: map[string]idpprogram.SchemaField{"member": {Ref: "bool", Required: true}}},
		"text":           {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 64, MaxLength: 32},
		"bool":           {ID: "bool", Kind: idpprogram.SchemaKindBoolean, MaxBytes: 5},
	}
	source := `
const A = require("tinyidp").v1;
const establish = A.lambda("identity.establish", {
  kind:"provider", input:"providerInput", output:"providerOutput",
  outcomes:["complete","deny"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:128,
  run: ctx => ctx.input.subject === "ada" ? A.result.complete({member:true}) : A.result.deny("policy.not_member")
});
module.exports = A.program("provider-test", p => p.provider("identity", "community", {
  version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{establish}
}));`
	artifact, err := idpscript.Compile(context.Background(), source, options)
	require.NoError(t, err)
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	require.NoError(t, err)
	pool, err := idpscript.NewPool(context.Background(), artifact, factory, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close(context.Background())) })
	invoker, err := idpscript.NewProviderInvoker(pool)
	require.NoError(t, err)

	complete, err := invoker.Invoke(context.Background(), "identity.community", "establish", json.RawMessage(`{"subject":"ada"}`), nil)
	require.NoError(t, err)
	assert.Equal(t, idpprogram.OutcomeComplete, complete.Kind)
	assert.JSONEq(t, `{"member":true}`, string(complete.Value))

	denied, err := invoker.Invoke(context.Background(), "identity.community", "establish", json.RawMessage(`{"subject":"ben"}`), nil)
	require.NoError(t, err)
	assert.Equal(t, idpprogram.OutcomeDeny, denied.Kind)
	assert.Equal(t, "policy.not_member", denied.Code)

	_, err = invoker.Invoke(context.Background(), "identity.community", "establish", json.RawMessage(`{"unexpected":true}`), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `schema "providerInput" requires field "subject"`)
}
