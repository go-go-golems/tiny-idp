package idpworkflow

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const DefaultMaximumContinuationTTL = 30 * time.Minute

type FieldErrorCode string

const (
	ErrorRequired FieldErrorCode = "required"
	ErrorInvalid  FieldErrorCode = "invalid"
	ErrorMismatch FieldErrorCode = "mismatch"
	ErrorRejected FieldErrorCode = "rejected"
	// ErrorExpired identifies a time-bounded value that can no longer be used.
	ErrorExpired FieldErrorCode = "expired"
	// ErrorAttemptsExceeded identifies a verifier that rejected too many values.
	ErrorAttemptsExceeded FieldErrorCode = "attempts_exceeded"
	// ErrorResendLimited identifies a challenge whose approved resend budget is spent.
	ErrorResendLimited FieldErrorCode = "resend_limited"
)

func (c FieldErrorCode) Valid() bool {
	return c == ErrorRequired || c == ErrorInvalid || c == ErrorMismatch || c == ErrorRejected || c == ErrorExpired || c == ErrorAttemptsExceeded || c == ErrorResendLimited
}

type FieldError struct {
	Field FieldID        `json:"field"`
	Code  FieldErrorCode `json:"code"`
}

// Presentation is the copied data-only result of ctx.present.form. It contains
// stable registry IDs and public text, never HTML or a secret field value.
type Presentation struct {
	Title         string             `json:"title"`
	ResumeHandler string             `json:"resumeHandler"`
	Fields        []FieldID          `json:"fields"`
	Actions       []ActionID         `json:"actions"`
	PublicValues  map[FieldID]string `json:"publicValues,omitempty"`
	Errors        []FieldError       `json:"errors,omitempty"`
	Carry         json.RawMessage    `json:"carry"`
	ExpiresIn     time.Duration      `json:"expiresIn"`
}

type ValidatedPresentation struct {
	Presentation Presentation
	Fields       []FieldDescriptor
	Actions      []ActionDescriptor
	InputSchema  string
}

// ValidatePresentation proves that a presentation uses one declared browser
// edge and only host-registered fields/actions before persistence or rendering.
func ValidatePresentation(program idpprogram.Program, workflowID, sourceHandler string, presentation Presentation, registry *Registry, maximumTTL time.Duration) (ValidatedPresentation, error) {
	if registry == nil {
		return ValidatedPresentation{}, errors.New("workflow presentation registry is required")
	}
	if maximumTTL <= 0 {
		maximumTTL = DefaultMaximumContinuationTTL
	}
	if strings.TrimSpace(presentation.Title) == "" || utf8.RuneCountInString(presentation.Title) > 120 {
		return ValidatedPresentation{}, errors.New("workflow presentation title must contain 1..120 characters")
	}
	if presentation.ExpiresIn <= 0 || presentation.ExpiresIn > maximumTTL {
		return ValidatedPresentation{}, errors.Errorf("workflow presentation expiry must be within 1..%s", maximumTTL)
	}
	workflow, ok := program.Workflows[workflowID]
	if !ok {
		return ValidatedPresentation{}, errors.Errorf("workflow %q is not registered", workflowID)
	}
	source, ok := workflow.Handlers[sourceHandler]
	if !ok {
		return ValidatedPresentation{}, errors.Errorf("source handler %q is not registered", sourceHandler)
	}
	lambda, ok := program.Lambdas[source.LambdaID]
	if !ok || !containsOutcome(lambda.AllowedOutcomes, idpprogram.OutcomePresent) {
		return ValidatedPresentation{}, errors.Errorf("source handler %q may not return present", sourceHandler)
	}
	inputSchema := ""
	for _, edge := range source.ContinuationEdges {
		if edge.OutcomeKind == idpprogram.OutcomePresent && edge.HandlerID == presentation.ResumeHandler {
			inputSchema = edge.InputSchema
			break
		}
	}
	if inputSchema == "" {
		return ValidatedPresentation{}, errors.Errorf("handler %q has no present edge to %q", sourceHandler, presentation.ResumeHandler)
	}
	destination, ok := workflow.Handlers[presentation.ResumeHandler]
	if !ok {
		return ValidatedPresentation{}, errors.Errorf("resume handler %q is not registered", presentation.ResumeHandler)
	}
	destinationLambda, ok := program.Lambdas[destination.LambdaID]
	if !ok || destinationLambda.InputSchema != inputSchema {
		return ValidatedPresentation{}, errors.New("presentation edge input schema is incompatible with resume handler")
	}
	if err := idpprogram.ValidatePublicJSON(program.Schemas, inputSchema, presentation.Carry); err != nil {
		return ValidatedPresentation{}, errors.Wrap(err, "validate presentation carry")
	}

	validated := ValidatedPresentation{Presentation: clonePresentation(presentation), InputSchema: inputSchema}
	seenFields := map[FieldID]bool{}
	for _, id := range presentation.Fields {
		if seenFields[id] {
			return ValidatedPresentation{}, errors.Errorf("duplicate presentation field %q", id)
		}
		seenFields[id] = true
		field, ok := registry.Field(id)
		if !ok {
			return ValidatedPresentation{}, errors.Errorf("presentation field %q is not registered", id)
		}
		validated.Fields = append(validated.Fields, field)
	}
	if len(validated.Fields) == 0 {
		return ValidatedPresentation{}, errors.New("workflow presentation requires at least one field")
	}
	seenActions := map[ActionID]bool{}
	for _, id := range presentation.Actions {
		if seenActions[id] {
			return ValidatedPresentation{}, errors.Errorf("duplicate presentation action %q", id)
		}
		seenActions[id] = true
		action, ok := registry.Action(id)
		if !ok {
			return ValidatedPresentation{}, errors.Errorf("presentation action %q is not registered", id)
		}
		validated.Actions = append(validated.Actions, action)
	}
	if len(validated.Actions) == 0 {
		return ValidatedPresentation{}, errors.New("workflow presentation requires at least one action")
	}
	for id, value := range presentation.PublicValues {
		field, selected := registry.Field(id)
		if !selected || !seenFields[id] {
			return ValidatedPresentation{}, errors.Errorf("public value field %q is not selected", id)
		}
		if field.Sensitive || field.Redisplay != RedisplayPublic {
			return ValidatedPresentation{}, errors.Errorf("field %q may not be publicly redisplayed", id)
		}
		length := utf8.RuneCountInString(value)
		if length < field.MinLength || length > field.MaxLength {
			return ValidatedPresentation{}, errors.Errorf("public value field %q is outside length bounds", id)
		}
	}
	for _, fieldError := range presentation.Errors {
		if !seenFields[fieldError.Field] || !fieldError.Code.Valid() {
			return ValidatedPresentation{}, errors.New("presentation contains an invalid field error")
		}
	}
	return validated, nil
}

func clonePresentation(presentation Presentation) Presentation {
	presentation.Fields = append([]FieldID(nil), presentation.Fields...)
	presentation.Actions = append([]ActionID(nil), presentation.Actions...)
	presentation.Carry = append([]byte(nil), presentation.Carry...)
	presentation.Errors = append([]FieldError(nil), presentation.Errors...)
	if presentation.PublicValues != nil {
		values := make(map[FieldID]string, len(presentation.PublicValues))
		for key, value := range presentation.PublicValues {
			values[key] = value
		}
		presentation.PublicValues = values
	}
	return presentation
}

func containsOutcome(outcomes []idpprogram.OutcomeKind, expected idpprogram.OutcomeKind) bool {
	for _, outcome := range outcomes {
		if outcome == expected {
			return true
		}
	}
	return false
}
