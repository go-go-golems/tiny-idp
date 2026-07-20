package verifyplan

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type echoDriver struct{}

func (echoDriver) Execute(_ context.Context, step Step) (Observation, error) {
	return Observation{Kind: step.Kind}, nil
}

func TestRunnerUsesOnlyRegisteredNativeAssertions(t *testing.T) {
	plan := Plan{SchemaVersion: SchemaVersion, Suites: []Suite{{Name: "suite", Scenarios: []Scenario{{Name: "scenario", Steps: []Step{{Kind: "begin"}}, Assertions: []Assertion{{ID: "observed", Version: "v1"}}}}}}}
	runner := Runner{Driver: echoDriver{}, Steps: StepRegistry{"begin": ExactObjectValidator}, Assertions: map[string]AssertionFunc{
		"observed@v1": func(_ context.Context, _ json.RawMessage, observations []Observation) error {
			if len(observations) != 1 || observations[0].Kind != "begin" {
				return fmt.Errorf("unexpected observations")
			}
			return nil
		},
	}}
	results, err := runner.Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results=%#v", results)
	}
}

func TestRunnerRejectsUnknownStepBeforeCallingDriver(t *testing.T) {
	plan := Plan{SchemaVersion: SchemaVersion, Suites: []Suite{{Name: "suite", Scenarios: []Scenario{{Name: "scenario", Steps: []Step{{Kind: "unknown"}}}}}}}
	_, err := (Runner{Driver: echoDriver{}, Steps: StepRegistry{"begin": ExactObjectValidator}}).Run(context.Background(), plan)
	if err == nil {
		t.Fatal("unknown step accepted")
	}
}

func TestRunnerRejectsMalformedParametersBeforeCallingDriver(t *testing.T) {
	plan := Plan{SchemaVersion: SchemaVersion, Suites: []Suite{{Name: "suite", Scenarios: []Scenario{{Name: "scenario", Steps: []Step{{Kind: "begin", Parameters: json.RawMessage(`{"forged":true}`)}}}}}}}
	_, err := (Runner{Driver: echoDriver{}, Steps: StepRegistry{"begin": ExactObjectValidator}}).Run(context.Background(), plan)
	if err == nil {
		t.Fatal("malformed step parameters accepted")
	}
}

func TestPlanValidationRejectsUnknownSchemaAndEmptyStep(t *testing.T) {
	plan := Plan{SchemaVersion: "unknown", Suites: []Suite{{Name: "suite"}}}
	if err := plan.Validate(DefaultLimits()); err == nil {
		t.Fatal("unknown schema accepted")
	}
	plan = Plan{SchemaVersion: SchemaVersion, Suites: []Suite{{Name: "suite", Scenarios: []Scenario{{Name: "scenario", Steps: []Step{{}}}}}}}
	if err := plan.Validate(DefaultLimits()); err == nil {
		t.Fatal("empty step kind accepted")
	}
}
