package assurance

import (
	"encoding/json"
	"sort"

	"github.com/pkg/errors"
)

const (
	ConfigurationSchemaVersion = "tinyidp.assurance.configuration/v1"
	TransitionSchemaVersion    = "tinyidp.assurance.transitions/v1"
	ScenarioTraceSchemaVersion = "tinyidp.assurance.scenario-trace/v1"
)

// ConfigurationReference identifies an already compiled script generation.
// It describes desired configuration only; it cannot claim that a native step
// ran or that an assurance property held at runtime.
type ConfigurationReference struct {
	SchemaVersion      string `json:"schemaVersion"`
	ProgramFingerprint string `json:"programFingerprint"`
	SourceFingerprint  string `json:"sourceFingerprint"`
}

func (r ConfigurationReference) Validate() error {
	if r.SchemaVersion != ConfigurationSchemaVersion || !ValidStableID(r.ProgramFingerprint) || !ValidStableID(r.SourceFingerprint) {
		return errors.New("configuration reference is invalid")
	}
	return nil
}

// TransitionDescriptor describes a host-owned native state transition. It is
// metadata for review, testing, analysis, and model export; it never dispatches
// production HTTP requests or grants script authority.
type TransitionDescriptor struct {
	ID           StepID          `json:"id"`
	Authorities  []HandlerID     `json:"authorities"`
	Reads        []ResourceID    `json:"reads,omitempty"`
	Writes       []ResourceID    `json:"writes,omitempty"`
	Requires     []FactID        `json:"requires,omitempty"`
	Produces     []FactID        `json:"produces,omitempty"`
	Discharges   []ObligationID  `json:"discharges,omitempty"`
	MayCreate    []ObligationID  `json:"mayCreate,omitempty"`
	Effects      []EffectID      `json:"effects,omitempty"`
	Outcomes     []OutcomeID     `json:"outcomes"`
	Observations []ObservationID `json:"observations,omitempty"`
}

type TransitionCatalog struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Transitions   []TransitionDescriptor `json:"transitions"`
}

func (c TransitionCatalog) Validate() error {
	if c.SchemaVersion != TransitionSchemaVersion || len(c.Transitions) == 0 {
		return errors.New("transition catalog is invalid")
	}
	seen := map[StepID]struct{}{}
	for _, transition := range c.Transitions {
		if !ValidStableID(string(transition.ID)) || len(transition.Outcomes) == 0 {
			return errors.New("transition descriptor is invalid")
		}
		if _, duplicate := seen[transition.ID]; duplicate {
			return errors.Errorf("transition descriptor %q is duplicated", transition.ID)
		}
		seen[transition.ID] = struct{}{}
		if len(transition.Authorities) == 0 {
			return errors.Errorf("transition descriptor %q has no native authority", transition.ID)
		}
		if err := validateIDs(stringIDs(transition.Authorities), stringIDs(transition.Reads), stringIDs(transition.Writes), stringIDs(transition.Requires), stringIDs(transition.Produces), stringIDs(transition.Discharges), stringIDs(transition.MayCreate), stringIDs(transition.Effects), stringIDs(transition.Outcomes), stringIDs(transition.Observations)); err != nil {
			return errors.Wrapf(err, "transition descriptor %q", transition.ID)
		}
	}
	return nil
}

// ScenarioRecord requests a sequence of registered native steps using bounded
// public parameters. TraceRecord records actual observations; it never infers
// an observation merely because the catalog says a step ought to emit one.
type ScenarioRecord struct {
	SchemaVersion string         `json:"schemaVersion"`
	Steps         []ScenarioStep `json:"steps"`
}

type ScenarioStep struct {
	Step       StepID          `json:"step"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type TraceRecord struct {
	SchemaVersion string             `json:"schemaVersion"`
	Observations  []TraceObservation `json:"observations"`
}

type TraceObservation struct {
	Step    StepID        `json:"step"`
	Kind    ObservationID `json:"kind"`
	Outcome OutcomeID     `json:"outcome"`
}

func (r ScenarioRecord) Validate(catalog TransitionCatalog) error {
	if r.SchemaVersion != ScenarioTraceSchemaVersion || len(r.Steps) == 0 {
		return errors.New("scenario record is invalid")
	}
	if err := catalog.Validate(); err != nil {
		return errors.Wrap(err, "validate transition catalog")
	}
	registered := catalog.stepSet()
	for _, step := range r.Steps {
		if _, ok := registered[step.Step]; !ok || !ValidStableID(string(step.Step)) || (len(step.Parameters) != 0 && !json.Valid(step.Parameters)) {
			return errors.New("scenario step is invalid")
		}
	}
	return nil
}

func (r TraceRecord) Validate(catalog TransitionCatalog) error {
	if r.SchemaVersion != ScenarioTraceSchemaVersion || len(r.Observations) == 0 {
		return errors.New("trace record is invalid")
	}
	if err := catalog.Validate(); err != nil {
		return errors.Wrap(err, "validate transition catalog")
	}
	registered := catalog.stepSet()
	for _, observation := range r.Observations {
		if _, ok := registered[observation.Step]; !ok || !ValidStableID(string(observation.Kind)) || !ValidStableID(string(observation.Outcome)) {
			return errors.New("trace observation is invalid")
		}
	}
	return nil
}

func (c TransitionCatalog) stepSet() map[StepID]struct{} {
	steps := make(map[StepID]struct{}, len(c.Transitions))
	for _, transition := range c.Transitions {
		steps[transition.ID] = struct{}{}
	}
	return steps
}

func validateIDs(groups ...[]string) error {
	for _, group := range groups {
		for _, value := range group {
			if !ValidStableID(value) {
				return errors.Errorf("invalid identifier %q", value)
			}
		}
	}
	return nil
}

func stringIDs[T ~string](values []T) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

// SortedTransitions returns a deterministic copy for canonical review output.
func (c TransitionCatalog) SortedTransitions() []TransitionDescriptor {
	result := append([]TransitionDescriptor(nil), c.Transitions...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
