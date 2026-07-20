package verifyplan

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StepValidator validates the bounded JSON parameters for one native scenario
// step. It decodes no authority and executes no transition.
type StepValidator func(json.RawMessage) error

// StepRegistry is an explicit native allow-list. A verification plan may name
// only registered steps; this is intentionally separate from the production
// policy capability registry.
type StepRegistry map[string]StepValidator

func (r StepRegistry) Validate(step Step) error {
	validator := r[step.Kind]
	if validator == nil {
		return fmt.Errorf("unknown native verification step %q", step.Kind)
	}
	raw := step.Parameters
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return fmt.Errorf("verification step %q has invalid parameters", step.Kind)
	}
	if err := validator(raw); err != nil {
		return fmt.Errorf("verification step %q parameters: %w", step.Kind, err)
	}
	return nil
}

// ExactObjectValidator returns a validator for parameter-free steps. It rejects
// all non-object, nonempty, or trailing JSON values.
func ExactObjectValidator(raw json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var value map[string]json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return fmt.Errorf("multiple JSON values")
	}
	if len(value) != 0 {
		return fmt.Errorf("parameters are not allowed")
	}
	return nil
}
