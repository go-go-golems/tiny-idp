// Package verifyplan defines immutable, data-only verification plans and a
// native runner. It deliberately has no JavaScript or identity-provider runtime
// dependency.
package verifyplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const SchemaVersion = "tinyidp.verify/v1"

type Plan struct {
	SchemaVersion string  `json:"schemaVersion"`
	SourceHash    string  `json:"sourceHash,omitempty"`
	Suites        []Suite `json:"suites"`
}

type Suite struct {
	Name      string     `json:"name"`
	Scenarios []Scenario `json:"scenarios"`
}

type Scenario struct {
	Name       string      `json:"name"`
	Steps      []Step      `json:"steps"`
	Assertions []Assertion `json:"assertions"`
}

type Step struct {
	Kind       string          `json:"kind"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type Assertion struct {
	ID      string          `json:"id"`
	Version string          `json:"version"`
	Config  json.RawMessage `json:"config,omitempty"`
}

type Limits struct {
	MaxSuites     int
	MaxScenarios  int
	MaxSteps      int
	MaxAssertions int
	MaxJSONBytes  int
}

func DefaultLimits() Limits {
	return Limits{MaxSuites: 32, MaxScenarios: 256, MaxSteps: 4096, MaxAssertions: 1024, MaxJSONBytes: 1 << 20}
}

func (p Plan) Validate(limits Limits) error {
	if p.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported verification plan schema %q", p.SchemaVersion)
	}
	if len(p.Suites) == 0 || len(p.Suites) > limits.MaxSuites {
		return fmt.Errorf("suite count %d outside 1..%d", len(p.Suites), limits.MaxSuites)
	}
	scenarios, steps, assertions := 0, 0, 0
	for suiteIndex, suite := range p.Suites {
		if strings.TrimSpace(suite.Name) == "" {
			return fmt.Errorf("suite %d has empty name", suiteIndex)
		}
		for scenarioIndex, scenario := range suite.Scenarios {
			scenarios++
			if strings.TrimSpace(scenario.Name) == "" {
				return fmt.Errorf("suite %q scenario %d has empty name", suite.Name, scenarioIndex)
			}
			for stepIndex, step := range scenario.Steps {
				steps++
				if strings.TrimSpace(step.Kind) == "" {
					return fmt.Errorf("scenario %q step %d has empty kind", scenario.Name, stepIndex)
				}
			}
			for assertionIndex, assertion := range scenario.Assertions {
				assertions++
				if strings.TrimSpace(assertion.ID) == "" || strings.TrimSpace(assertion.Version) == "" {
					return fmt.Errorf("scenario %q assertion %d requires id and version", scenario.Name, assertionIndex)
				}
			}
		}
	}
	if scenarios > limits.MaxScenarios || steps > limits.MaxSteps || assertions > limits.MaxAssertions {
		return fmt.Errorf("plan exceeds limits: scenarios=%d/%d steps=%d/%d assertions=%d/%d", scenarios, limits.MaxScenarios, steps, limits.MaxSteps, assertions, limits.MaxAssertions)
	}
	b, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshal verification plan: %w", err)
	}
	if len(b) > limits.MaxJSONBytes {
		return fmt.Errorf("plan JSON size %d exceeds %d", len(b), limits.MaxJSONBytes)
	}
	return nil
}

// ValidateWithSteps materializes a plan against the native driver's explicit
// step registry. It is the required boundary before a driver may execute any
// user-authored scenario action.
func (p Plan) ValidateWithSteps(limits Limits, steps StepRegistry) error {
	if len(steps) == 0 {
		return fmt.Errorf("verification step registry is required")
	}
	if err := p.Validate(limits); err != nil {
		return err
	}
	for _, suite := range p.Suites {
		for _, scenario := range suite.Scenarios {
			for _, step := range scenario.Steps {
				if err := steps.Validate(step); err != nil {
					return fmt.Errorf("suite %q scenario %q: %w", suite.Name, scenario.Name, err)
				}
			}
		}
	}
	return nil
}

func (p *Plan) BindSource(source []byte) {
	digest := sha256.Sum256(source)
	p.SourceHash = hex.EncodeToString(digest[:])
}

type Observation struct {
	Kind string         `json:"kind"`
	Data map[string]any `json:"data,omitempty"`
}

type Driver interface {
	Execute(ctx context.Context, step Step) (Observation, error)
}

type AssertionFunc func(ctx context.Context, config json.RawMessage, observations []Observation) error

type Runner struct {
	Driver     Driver
	Steps      StepRegistry
	Assertions map[string]AssertionFunc
}

type ScenarioResult struct {
	Suite        string
	Scenario     string
	Passed       bool
	Error        string
	Observations []Observation
}

func (r Runner) Run(ctx context.Context, plan Plan) ([]ScenarioResult, error) {
	if r.Driver == nil {
		return nil, fmt.Errorf("verification driver is required")
	}
	if err := plan.ValidateWithSteps(DefaultLimits(), r.Steps); err != nil {
		return nil, err
	}
	results := make([]ScenarioResult, 0)
	for _, suite := range plan.Suites {
		for _, scenario := range suite.Scenarios {
			result := ScenarioResult{Suite: suite.Name, Scenario: scenario.Name, Passed: true}
			for _, step := range scenario.Steps {
				observation, err := r.Driver.Execute(ctx, step)
				if err != nil {
					result.Passed = false
					result.Error = err.Error()
					break
				}
				result.Observations = append(result.Observations, observation)
			}
			if result.Passed {
				for _, assertion := range scenario.Assertions {
					key := assertion.ID + "@" + assertion.Version
					check := r.Assertions[key]
					if check == nil {
						result.Passed = false
						result.Error = "unknown native assertion " + key
						break
					}
					if err := check(ctx, assertion.Config, result.Observations); err != nil {
						result.Passed = false
						result.Error = err.Error()
						break
					}
				}
			}
			results = append(results, result)
		}
	}
	return results, nil
}
