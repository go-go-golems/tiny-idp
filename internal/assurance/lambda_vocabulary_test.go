package assurance

import (
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func TestVocabularyForProgramProjectsDeclaredLambdaContracts(t *testing.T) {
	program := idpprogram.Program{
		APIVersion: idpprogram.APIVersionV1,
		Name:       "assurance-test",
		Schemas: map[string]idpprogram.Schema{
			"input":  {ID: "input", Kind: idpprogram.SchemaKindObject, MaxBytes: 128},
			"output": {ID: "output", Kind: idpprogram.SchemaKindObject, MaxBytes: 128},
		},
		Capabilities: map[string]idpprogram.CapabilityRequirement{
			"directory.lookup": {ID: "directory.lookup", Version: 1},
		},
		Lambdas: map[string]idpprogram.LambdaSpec{
			"signup": {
				ID:                   "signup",
				Kind:                 idpprogram.LambdaKindWorkflow,
				InputSchema:          "input",
				OutputSchema:         "output",
				AllowedOutcomes:      []idpprogram.OutcomeKind{idpprogram.OutcomePresent, idpprogram.OutcomeCommit},
				AllowedEffects:       []idpprogram.EffectKind{idpprogram.EffectCreateLocalIdentity, idpprogram.EffectAttachPasswordCredential},
				RequiredCapabilities: []idpprogram.CapabilityRequirement{{ID: "directory.lookup", Version: 1}},
				Budget:               idpprogram.InvocationBudget{Timeout: time.Second, MaxOutputBytes: 128},
			},
		},
		Workflows: map[string]idpprogram.Workflow{
			"signup": {ID: "signup", Version: 1, EntryHandler: "start", Handlers: map[string]idpprogram.HandlerSpec{"start": {ID: "start", LambdaID: "signup"}}},
		},
	}

	vocabulary, err := VocabularyForProgram(program)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := vocabulary.Outcomes, []OutcomeID{LambdaOutcomeCommit, LambdaOutcomePresent}; !equalOutcomes(got, want) {
		t.Fatalf("outcomes=%v want %v", got, want)
	}
	if got, want := vocabulary.Effects, []EffectID{EffectAttachPasswordCredential, EffectCreateLocalIdentity}; !equalEffects(got, want) {
		t.Fatalf("effects=%v want %v", got, want)
	}
	if got, want := vocabulary.Capabilities, []CapabilityID{"directory.lookup"}; !equalCapabilities(got, want) {
		t.Fatalf("capabilities=%v want %v", got, want)
	}
}

func TestLambdaVocabularyRejectsUnknownContractValues(t *testing.T) {
	if _, err := LambdaOutcomeID("surprise"); err == nil {
		t.Fatal("unknown outcome accepted")
	}
	if _, err := LambdaEffectID("surprise"); err == nil {
		t.Fatal("unknown effect accepted")
	}
}

func equalOutcomes(a, b []OutcomeID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalEffects(a, b []EffectID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCapabilities(a, b []CapabilityID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
