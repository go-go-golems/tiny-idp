package idpinvite_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

func TestComputedEligibilityCapabilityOnlyExposesValidatedDecision(t *testing.T) {
	called := false
	capability, err := idpinvite.NewEligibilityCapability(func(_ context.Context, probe idpinvite.EligibilityProbe) (idpinvite.EligibilityDecision, error) {
		called = true
		assert.Equal(t, "member@example.test", probe.Email)
		assert.Equal(t, "message-app", probe.Audience)
		return idpinvite.EligibilityDecision{Accepted: true, EvidenceID: "directory:membership-42"}, nil
	})
	require.NoError(t, err)
	result, err := capability.Invoke(context.Background(), json.RawMessage(`{"email":"member@example.test","audience":"message-app"}`))
	require.NoError(t, err)
	assert.True(t, called)
	assert.JSONEq(t, `{"accepted":true,"evidenceId":"directory:membership-42"}`, string(result))
}

func TestComputedEligibilityCapabilityRejectsUnboundedOrInvalidValues(t *testing.T) {
	called := false
	capability, err := idpinvite.NewEligibilityCapability(func(context.Context, idpinvite.EligibilityProbe) (idpinvite.EligibilityDecision, error) {
		called = true
		return idpinvite.EligibilityDecision{}, nil
	})
	require.NoError(t, err)
	_, err = capability.Invoke(context.Background(), json.RawMessage(`{"email":"member@example.test","audience":"message-app","database":"forged"}`))
	require.Error(t, err)
	assert.False(t, called)
	_, err = idpinvite.NewEligibilityCapability(nil)
	require.Error(t, err)
}

func TestComputedInvitationProviderReceivesCapabilityDecisionNotHostAuthority(t *testing.T) {
	options := idpscript.DefaultCompileOptions()
	options.Schemas = map[string]idpprogram.Schema{
		"probe": {ID: "probe", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Fields: map[string]idpprogram.SchemaField{
			"email": {Ref: "text", Required: true}, "audience": {Ref: "text", Required: true},
		}},
		"decision": {ID: "decision", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Fields: map[string]idpprogram.SchemaField{
			"accepted": {Ref: "bool", Required: true}, "evidenceId": {Ref: "text"},
		}},
		"text": {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 512, MaxLength: 128},
		"bool": {ID: "bool", Kind: idpprogram.SchemaKindBoolean, MaxBytes: 8},
	}
	artifact, err := idpscript.Compile(context.Background(), computedProviderSource, options)
	require.NoError(t, err)
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	require.NoError(t, err)
	pool, err := idpscript.NewPool(context.Background(), artifact, factory, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close(context.Background())) })
	invoker, err := idpscript.NewProviderInvoker(pool)
	require.NoError(t, err)
	capability, err := idpinvite.NewEligibilityCapability(func(_ context.Context, probe idpinvite.EligibilityProbe) (idpinvite.EligibilityDecision, error) {
		return idpinvite.EligibilityDecision{Accepted: probe.Email == "member@example.test", EvidenceID: "directory:42"}, nil
	})
	require.NoError(t, err)
	outcome, err := invoker.Invoke(context.Background(), "invitation.community", idpprogram.InvitationValidateHandler, json.RawMessage(`{"email":"member@example.test","audience":"message-app"}`), map[string]idpscript.CapabilityBinding{idpinvite.EligibilityCapabilityID: capability})
	require.NoError(t, err)
	assert.Equal(t, idpprogram.OutcomeComplete, outcome.Kind)
	assert.JSONEq(t, `{"accepted":true,"evidenceId":"directory:42"}`, string(outcome.Value))
}

const computedProviderSource = `
const A = require("tinyidp").v1;
module.exports = A.program("computed-invitation", program => {
  program.capabilities({"invitation.eligibility": {version: 1}});
  const validate = A.lambda("community.validate", {
    kind: "provider", input: "probe", output: "decision",
    outcomes: ["complete"], effects: [], capabilities: ["invitation.eligibility"],
    timeoutMs: 250, maxCapabilityCalls: 1, maxOutputBytes: 1024,
    run: async ctx => A.result.complete(await ctx.cap.invitation.eligibility(ctx.input)),
  });
  program.provider("invitation", "community", {
    version: 1, state: "virtual", replayProtection: "none", revocation: "none",
    handlers: {validate},
  });
});`
