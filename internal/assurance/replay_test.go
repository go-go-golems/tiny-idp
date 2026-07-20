package assurance

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/verifyplan"
)

type recordingReplayDriver struct{ kinds []string }

func (d *recordingReplayDriver) Execute(_ context.Context, step verifyplan.Step) (verifyplan.Observation, error) {
	d.kinds = append(d.kinds, step.Kind)
	return verifyplan.Observation{Kind: "replayed"}, nil
}

func TestNormalizedCounterexampleUsesStableStepIDsWithoutAdapter(t *testing.T) {
	catalog := CurrentAuthorizationCatalog()
	counterexample := NormalizedCounterexample{
		SchemaVersion: NormalizedCounterexampleSchemaVersion,
		Name:          "duplicate terminal consume",
		Steps: []ScenarioStep{
			{Step: StepInteractionCreate, Parameters: json.RawMessage(`{}`)},
			{Step: StepInteractionApprove, Parameters: json.RawMessage(`{}`)},
			{Step: StepInteractionApprove, Parameters: json.RawMessage(`{}`)},
		},
	}
	plan, err := counterexample.VerificationPlan(catalog)
	if err != nil {
		t.Fatal(err)
	}
	driver := &recordingReplayDriver{}
	runner := verifyplan.Runner{Driver: driver, Steps: verifyplan.StepRegistry{
		string(StepInteractionCreate):  verifyplan.ExactObjectValidator,
		string(StepInteractionApprove): verifyplan.ExactObjectValidator,
	}}
	results, err := runner.Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results=%#v", results)
	}
	want := []string{string(StepInteractionCreate), string(StepInteractionApprove), string(StepInteractionApprove)}
	if len(driver.kinds) != len(want) {
		t.Fatalf("replayed kinds=%v want %v", driver.kinds, want)
	}
	for index := range want {
		if driver.kinds[index] != want[index] {
			t.Fatalf("replayed kind %d=%q want %q", index, driver.kinds[index], want[index])
		}
	}
}
