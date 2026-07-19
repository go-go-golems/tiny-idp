package idpprogram_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAcceptsBoundedWorkflow(t *testing.T) {
	program := validProgram()

	diagnostics := idpprogram.Validate(program)

	assert.Empty(t, diagnostics)
}

func TestValidateReturnsStableDiagnosticsForMalformedProgram(t *testing.T) {
	program := validProgram()
	program.APIVersion = "tinyidp/v99"
	program.Schemas["signupInput"] = idpprogram.Schema{
		ID:       "wrong",
		Kind:     idpprogram.SchemaKindObject,
		MaxBytes: 0,
		Fields: map[string]idpprogram.SchemaField{
			"self": {Ref: "signupInput"},
		},
	}
	lambda := program.Lambdas["signup.start"]
	lambda.AllowedOutcomes = append(lambda.AllowedOutcomes, idpprogram.OutcomePresent)
	lambda.RequiredCapabilities = []idpprogram.CapabilityRequirement{{ID: "missing", Version: 1}}
	program.Lambdas[lambda.ID] = lambda
	workflow := program.Workflows["signup"]
	workflow.Handlers["orphan"] = idpprogram.HandlerSpec{ID: "orphan", LambdaID: "missing"}
	program.Workflows[workflow.ID] = workflow

	diagnostics := idpprogram.Validate(program)

	require.True(t, diagnostics.HasErrors())
	ids := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		ids = append(ids, diagnostic.ID)
	}
	assert.Contains(t, ids, "program.api_version")
	assert.Contains(t, ids, "schema.id_mismatch")
	assert.Contains(t, ids, "schema.max_bytes")
	assert.Contains(t, ids, "schema.reference_cycle")
	assert.Contains(t, ids, "lambda.outcome_duplicate")
	assert.Contains(t, ids, "lambda.capability")
	assert.Contains(t, ids, "handler.lambda")
	assert.Contains(t, ids, "handler.unreachable")
	for i := 1; i < len(diagnostics); i++ {
		previous := diagnostics[i-1]
		current := diagnostics[i]
		assert.True(t,
			previous.Path < current.Path ||
				(previous.Path == current.Path && previous.ID <= current.ID),
			"diagnostics are not sorted at %d: %#v then %#v", i, previous, current,
		)
	}
}

func TestValidateRejectsIncompatibleContinuationEdge(t *testing.T) {
	program := validProgram()
	workflow := program.Workflows["signup"]
	handler := workflow.Handlers["start"]
	handler.ContinuationEdges[0].InputSchema = "terminalValue"
	workflow.Handlers[handler.ID] = handler
	program.Workflows[workflow.ID] = workflow

	diagnostics := idpprogram.Validate(program)

	require.True(t, diagnostics.HasErrors())
	assert.Contains(t, diagnosticIDs(diagnostics), "edge.schema")
}

func TestCanonicalJSONAndFingerprintsAreDeterministic(t *testing.T) {
	first := validProgram()
	second := validProgram()
	// Rebuild maps in a deliberately different insertion order.
	second.Schemas = map[string]idpprogram.Schema{
		"terminalValue": second.Schemas["terminalValue"],
		"signupInput":   second.Schemas["signupInput"],
	}
	second.Lambdas = map[string]idpprogram.LambdaSpec{
		"signup.submitted": second.Lambdas["signup.submitted"],
		"signup.start":     second.Lambdas["signup.start"],
	}

	firstJSON, err := idpprogram.CanonicalJSON(first)
	require.NoError(t, err)
	secondJSON, err := idpprogram.CanonicalJSON(second)
	require.NoError(t, err)
	assert.Equal(t, string(firstJSON), string(secondJSON))

	firstHashes, err := idpprogram.ComputeFingerprints([]byte("module.exports = 1"), first)
	require.NoError(t, err)
	secondHashes, err := idpprogram.ComputeFingerprints([]byte("module.exports = 1"), second)
	require.NoError(t, err)
	assert.Equal(t, firstHashes, secondHashes)
	assert.Len(t, firstHashes.Program, 64)
	assert.NotEqual(t, firstHashes.Program, firstHashes.CallbackRegistry)

	changedHashes, err := idpprogram.ComputeFingerprints([]byte("module.exports = 2"), second)
	require.NoError(t, err)
	assert.NotEqual(t, firstHashes.Source, changedHashes.Source)
	assert.Equal(t, firstHashes.Program, changedHashes.Program)
}

func TestValidateOutcomeEnforcesDeclaredOutcomeAndEffects(t *testing.T) {
	spec := validProgram().Lambdas["signup.submitted"]

	err := idpprogram.ValidateOutcome(spec, idpprogram.Outcome{
		Kind: idpprogram.OutcomeCommit,
		Effects: []idpprogram.EffectPlan{{
			Kind:    idpprogram.EffectCreateLocalIdentity,
			Payload: json.RawMessage(`{"login":"ada"}`),
		}},
	})
	require.NoError(t, err)

	err = idpprogram.ValidateOutcome(spec, idpprogram.Outcome{
		Kind: idpprogram.OutcomeCommit,
		Effects: []idpprogram.EffectPlan{{
			Kind: idpprogram.EffectConsumeInvitation,
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undeclared effect")

	err = idpprogram.ValidateOutcome(spec, idpprogram.Outcome{Kind: idpprogram.OutcomeSkip})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undeclared outcome")
}

func TestValidateOutcomeRequiresExplicitBrowserContinuation(t *testing.T) {
	spec := validProgram().Lambdas["signup.start"]

	err := idpprogram.ValidateOutcome(spec, idpprogram.Outcome{Kind: idpprogram.OutcomePresent})
	require.Error(t, err)

	err = idpprogram.ValidateOutcome(spec, idpprogram.Outcome{
		Kind: idpprogram.OutcomePresent,
		Continuation: &idpprogram.BrowserContinuation{
			HandlerID: "submitted",
			ExpiresIn: 300,
		},
	})
	require.NoError(t, err)
}

func validProgram() idpprogram.Program {
	return idpprogram.Program{
		APIVersion: idpprogram.APIVersionV1,
		Name:       "community",
		Schemas: map[string]idpprogram.Schema{
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
		},
		Capabilities: map[string]idpprogram.CapabilityRequirement{
			"directory.lookup": {ID: "directory.lookup", Version: 1},
		},
		Lambdas: map[string]idpprogram.LambdaSpec{
			"signup.start": {
				ID:              "signup.start",
				Kind:            idpprogram.LambdaKindWorkflow,
				InputSchema:     "signupInput",
				OutputSchema:    "terminalValue",
				AllowedOutcomes: []idpprogram.OutcomeKind{idpprogram.OutcomePresent},
				Budget: idpprogram.InvocationBudget{
					Timeout:            250 * time.Millisecond,
					MaxCapabilityCalls: 0,
					MaxOutputBytes:     4096,
				},
			},
			"signup.submitted": {
				ID:              "signup.submitted",
				Kind:            idpprogram.LambdaKindWorkflow,
				InputSchema:     "signupInput",
				OutputSchema:    "terminalValue",
				AllowedOutcomes: []idpprogram.OutcomeKind{idpprogram.OutcomeCommit, idpprogram.OutcomeDeny},
				RequiredCapabilities: []idpprogram.CapabilityRequirement{
					{ID: "directory.lookup", Version: 1},
				},
				AllowedEffects: []idpprogram.EffectKind{
					idpprogram.EffectCreateLocalIdentity,
					idpprogram.EffectAttachPasswordCredential,
				},
				Budget: idpprogram.InvocationBudget{
					Timeout:            time.Second,
					MaxCapabilityCalls: 1,
					MaxOutputBytes:     4096,
				},
			},
		},
		Workflows: map[string]idpprogram.Workflow{
			"signup": {
				ID:           "signup",
				Version:      1,
				EntryHandler: "start",
				Handlers: map[string]idpprogram.HandlerSpec{
					"start": {
						ID:       "start",
						LambdaID: "signup.start",
						ContinuationEdges: []idpprogram.ContinuationEdge{{
							OutcomeKind: idpprogram.OutcomePresent,
							HandlerID:   "submitted",
							InputSchema: "signupInput",
						}},
					},
					"submitted": {
						ID:       "submitted",
						LambdaID: "signup.submitted",
					},
				},
			},
		},
	}
}

func diagnosticIDs(diagnostics idpprogram.Diagnostics) []string {
	ret := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		ret = append(ret, diagnostic.ID)
	}
	return ret
}
