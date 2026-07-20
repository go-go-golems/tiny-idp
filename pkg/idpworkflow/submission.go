package idpworkflow

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const (
	interactionFieldName  = "interaction"
	csrfFieldName         = "csrf_token"
	actionFieldName       = "action"
	continuationFieldName = "workflow_continuation"
)

// Submission is the native projection of one workflow form POST. PublicValues
// contain only fields that may be redisplayed. SecretValues are byte slices so
// callers cannot accidentally mix them with normal JavaScript input strings.
type Submission struct {
	Interaction  string
	Continuation string
	CSRFToken    string
	Action       ActionID
	PublicValues map[FieldID]string
	Secrets      map[FieldID]SecretHandle
	secretSet    *SecretSet
}

// ParseSubmission accepts exactly the fields declared by a validated
// presentation. It rejects duplicate singleton values, extra parameters,
// missing selected fields, unsupported actions, invalid UTF-8, and invalid
// normalization before any workflow lambda is invoked.
func ParseSubmission(fields []FieldDescriptor, actions []ActionDescriptor, form url.Values) (Submission, error) {
	if len(fields) == 0 || len(actions) == 0 {
		return Submission{}, errors.New("workflow submission requires fields and actions")
	}
	expected := map[string]FieldDescriptor{}
	for _, field := range fields {
		if err := field.Validate(); err != nil {
			return Submission{}, errors.Wrap(err, "invalid workflow field")
		}
		if _, exists := expected[field.InputName]; exists {
			return Submission{}, errors.Errorf("duplicate workflow field name %q", field.InputName)
		}
		expected[field.InputName] = field
	}
	allowedActions := map[ActionID]ActionDescriptor{}
	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return Submission{}, errors.Wrap(err, "invalid workflow action")
		}
		if _, exists := allowedActions[action.ID]; exists {
			return Submission{}, errors.Errorf("duplicate workflow action %q", action.ID)
		}
		allowedActions[action.ID] = action
	}

	for name := range form {
		if name == interactionFieldName || name == csrfFieldName || name == actionFieldName || name == continuationFieldName {
			continue
		}
		if _, ok := expected[name]; !ok {
			return Submission{}, errors.Errorf("workflow submission contains unexpected field %q", name)
		}
	}
	interaction, err := singleton(form, interactionFieldName, true)
	if err != nil {
		return Submission{}, err
	}
	csrfToken, err := singleton(form, csrfFieldName, true)
	if err != nil {
		return Submission{}, err
	}
	continuation, err := singleton(form, continuationFieldName, true)
	if err != nil {
		return Submission{}, err
	}
	actionValue, err := singleton(form, actionFieldName, true)
	if err != nil {
		return Submission{}, err
	}
	action, ok := allowedActions[ActionID(actionValue)]
	if !ok {
		return Submission{}, errors.Errorf("workflow submission has unsupported action %q", actionValue)
	}

	result := Submission{Interaction: interaction, Continuation: continuation, CSRFToken: csrfToken, Action: action.ID, PublicValues: map[FieldID]string{}}
	secretValues := map[FieldID][]byte{}
	defer func() {
		for field, value := range secretValues {
			clear(value)
			delete(secretValues, field)
		}
	}()
	for inputName, field := range expected {
		raw, err := singleton(form, inputName, false)
		if err != nil {
			return Submission{}, err
		}
		if !utf8.ValidString(raw) {
			return Submission{}, errors.Errorf("workflow field %q is not valid UTF-8", field.ID)
		}
		if field.Sensitive {
			if !action.SkipFormValidation && field.Required && raw == "" {
				return Submission{}, errors.Errorf("workflow field %q is required", field.ID)
			}
			if !action.SkipFormValidation && utf8.RuneCountInString(raw) > field.MaxLength {
				return Submission{}, errors.Errorf("workflow field %q exceeds maximum length", field.ID)
			}
			secretValues[field.ID] = append([]byte(nil), raw...)
			continue
		}
		normalized, err := normalizeField(field, raw)
		if err != nil {
			return Submission{}, err
		}
		if !action.SkipFormValidation && field.Required && normalized == "" {
			return Submission{}, errors.Errorf("workflow field %q is required", field.ID)
		}
		if !action.SkipFormValidation && (utf8.RuneCountInString(normalized) < field.MinLength || utf8.RuneCountInString(normalized) > field.MaxLength) {
			return Submission{}, errors.Errorf("workflow field %q is outside length bounds", field.ID)
		}
		if !action.SkipFormValidation && field.Kind == ValueEmail && normalized != "" {
			address, parseErr := mail.ParseAddress(normalized)
			if parseErr != nil || address.Address != normalized {
				return Submission{}, errors.Errorf("workflow field %q is not a valid email address", field.ID)
			}
		}
		if field.Redisplay == RedisplayPublic {
			result.PublicValues[field.ID] = normalized
		}
	}
	secretSet, handles, err := newSecretSet(secretValues)
	if err != nil {
		return Submission{}, err
	}
	result.secretSet = secretSet
	result.Secrets = handles
	return result, nil
}

// ResolveSecret supplies a clone for immediate trusted native work. A secret
// can never be recovered from a serialized workflow outcome or presentation.
func (s Submission) ResolveSecret(handle SecretHandle) ([]byte, bool) {
	return s.secretSet.Resolve(handle)
}

// DestroySecrets erases parsed secret bytes once the request's native commit
// work has completed or failed.
func (s Submission) DestroySecrets() { s.secretSet.Destroy() }

func singleton(values url.Values, name string, nonEmpty bool) (string, error) {
	entries, ok := values[name]
	if !ok {
		return "", errors.Errorf("workflow submission is missing field %q", name)
	}
	if len(entries) != 1 {
		return "", errors.Errorf("workflow submission field %q must occur exactly once", name)
	}
	if !utf8.ValidString(entries[0]) || (nonEmpty && entries[0] == "") {
		return "", errors.Errorf("workflow submission field %q is invalid", name)
	}
	return entries[0], nil
}

func normalizeField(field FieldDescriptor, raw string) (string, error) {
	switch field.Normalize {
	case NormalizeTrim:
		return strings.TrimSpace(raw), nil
	case NormalizeTrimLower:
		return strings.ToLower(strings.TrimSpace(raw)), nil
	case NormalizeNone:
		return raw, nil
	default:
		return "", errors.Errorf("workflow field %q has unsupported normalization", field.ID)
	}
}
