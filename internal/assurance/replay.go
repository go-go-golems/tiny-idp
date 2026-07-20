package assurance

import (
	"fmt"

	"github.com/go-go-golems/tiny-idp/pkg/verifyplan"
)

const NormalizedCounterexampleSchemaVersion = "tinyidp.assurance.counterexample/v1"

// NormalizedCounterexample is a finite model counterexample after a model
// exporter has removed model-tool syntax and unbounded values. Each action is
// already a stable native StepID. This is deliberately a data record: it does
// not include a model checker, a driver, or a JavaScript runtime.
type NormalizedCounterexample struct {
	SchemaVersion string         `json:"schemaVersion"`
	Name          string         `json:"name"`
	Steps         []ScenarioStep `json:"steps"`
}

func (c NormalizedCounterexample) Validate(catalog TransitionCatalog) error {
	if c.SchemaVersion != NormalizedCounterexampleSchemaVersion || c.Name == "" {
		return fmt.Errorf("normalized counterexample is invalid")
	}
	return (ScenarioRecord{SchemaVersion: ScenarioTraceSchemaVersion, Steps: c.Steps}).Validate(catalog)
}

// VerificationPlan copies normalized StepID values directly to registered
// VerificationPlan kinds. There is intentionally no action-name adapter: a
// counterexample that names an unregistered or malformed native step fails at
// the runner's StepRegistry materialization boundary.
func (c NormalizedCounterexample) VerificationPlan(catalog TransitionCatalog) (verifyplan.Plan, error) {
	if err := c.Validate(catalog); err != nil {
		return verifyplan.Plan{}, err
	}
	steps := make([]verifyplan.Step, 0, len(c.Steps))
	for _, step := range c.Steps {
		steps = append(steps, verifyplan.Step{Kind: string(step.Step), Parameters: append([]byte(nil), step.Parameters...)})
	}
	return verifyplan.Plan{SchemaVersion: verifyplan.SchemaVersion, Suites: []verifyplan.Suite{{Name: "normalized-model-counterexamples", Scenarios: []verifyplan.Scenario{{Name: c.Name, Steps: steps}}}}}, nil
}
