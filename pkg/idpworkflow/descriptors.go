// Package idpworkflow defines provider-owned workflow presentation and browser
// projection contracts. JavaScript selects registered descriptors; it cannot
// define HTML, input names, secret handling, or normalization code.
package idpworkflow

import (
	"sort"

	"github.com/pkg/errors"
)

type FieldID string

const (
	FieldDisplayName          FieldID = "displayName"
	FieldEmail                FieldID = "email"
	FieldPassword             FieldID = "password"
	FieldPasswordConfirmation FieldID = "passwordConfirmation"
	FieldInviteCode           FieldID = "inviteCode"
	FieldEmailCode            FieldID = "emailCode"
)

type ValueKind string

const (
	ValueText   ValueKind = "text"
	ValueEmail  ValueKind = "email"
	ValueSecret ValueKind = "secret"
)

type Normalization string

const (
	NormalizeTrim      Normalization = "trim"
	NormalizeTrimLower Normalization = "trimLower"
	NormalizeNone      Normalization = "none"
)

type RedisplayPolicy string

const (
	RedisplayPublic RedisplayPolicy = "public"
	RedisplayNever  RedisplayPolicy = "never"
)

type FieldDescriptor struct {
	ID           FieldID
	InputName    string
	Label        string
	Kind         ValueKind
	Normalize    Normalization
	Required     bool
	MinLength    int
	MaxLength    int
	Sensitive    bool
	Autocomplete string
	Redisplay    RedisplayPolicy
}

func (d FieldDescriptor) Validate() error {
	if d.ID == "" || d.InputName == "" || d.Label == "" {
		return errors.New("field ID, input name, and label are required")
	}
	if d.Kind != ValueText && d.Kind != ValueEmail && d.Kind != ValueSecret {
		return errors.Errorf("field %q has invalid value kind %q", d.ID, d.Kind)
	}
	if d.Normalize != NormalizeTrim && d.Normalize != NormalizeTrimLower && d.Normalize != NormalizeNone {
		return errors.Errorf("field %q has invalid normalization %q", d.ID, d.Normalize)
	}
	if d.MinLength < 0 || d.MaxLength <= 0 || d.MinLength > d.MaxLength {
		return errors.Errorf("field %q has invalid length bounds", d.ID)
	}
	if d.Sensitive != (d.Kind == ValueSecret) {
		return errors.Errorf("field %q sensitivity must match secret value kind", d.ID)
	}
	if d.Sensitive && (d.Redisplay != RedisplayNever || d.Normalize != NormalizeNone) {
		return errors.Errorf("secret field %q must never redisplay or normalize", d.ID)
	}
	if !d.Sensitive && d.Redisplay != RedisplayPublic {
		return errors.Errorf("public field %q must use public redisplay policy", d.ID)
	}
	return nil
}

type ActionID string

const (
	ActionSubmit ActionID = "submit"
	ActionDeny   ActionID = "deny"
)

type ActionDescriptor struct {
	ID                 ActionID
	Label              string
	SkipFormValidation bool
}

func (d ActionDescriptor) Validate() error {
	if d.ID != ActionSubmit && d.ID != ActionDeny {
		return errors.Errorf("invalid workflow action %q", d.ID)
	}
	if d.Label == "" {
		return errors.Errorf("workflow action %q label is required", d.ID)
	}
	if d.SkipFormValidation != (d.ID == ActionDeny) {
		return errors.Errorf("workflow action %q has invalid form-validation policy", d.ID)
	}
	return nil
}

type Registry struct {
	fields  map[FieldID]FieldDescriptor
	actions map[ActionID]ActionDescriptor
}

func NewRegistry(fields []FieldDescriptor, actions []ActionDescriptor) (*Registry, error) {
	registry := &Registry{fields: map[FieldID]FieldDescriptor{}, actions: map[ActionID]ActionDescriptor{}}
	for _, field := range fields {
		if err := field.Validate(); err != nil {
			return nil, err
		}
		if _, exists := registry.fields[field.ID]; exists {
			return nil, errors.Errorf("duplicate field descriptor %q", field.ID)
		}
		for _, existing := range registry.fields {
			if existing.InputName == field.InputName {
				return nil, errors.Errorf("duplicate field input name %q", field.InputName)
			}
		}
		registry.fields[field.ID] = field
	}
	for _, action := range actions {
		if err := action.Validate(); err != nil {
			return nil, err
		}
		if _, exists := registry.actions[action.ID]; exists {
			return nil, errors.Errorf("duplicate action descriptor %q", action.ID)
		}
		registry.actions[action.ID] = action
	}
	return registry, nil
}

func DefaultRegistry() *Registry {
	registry, err := NewRegistry([]FieldDescriptor{
		{ID: FieldDisplayName, InputName: "display_name", Label: "Display name", Kind: ValueText, Normalize: NormalizeTrim, Required: true, MinLength: 1, MaxLength: 120, Autocomplete: "name", Redisplay: RedisplayPublic},
		{ID: FieldEmail, InputName: "email", Label: "Email", Kind: ValueEmail, Normalize: NormalizeTrimLower, Required: true, MinLength: 3, MaxLength: 320, Autocomplete: "email", Redisplay: RedisplayPublic},
		{ID: FieldPassword, InputName: "password", Label: "Password", Kind: ValueSecret, Normalize: NormalizeNone, Required: true, MinLength: 12, MaxLength: 1024, Sensitive: true, Autocomplete: "new-password", Redisplay: RedisplayNever},
		{ID: FieldPasswordConfirmation, InputName: "password_confirmation", Label: "Confirm password", Kind: ValueSecret, Normalize: NormalizeNone, Required: true, MinLength: 12, MaxLength: 1024, Sensitive: true, Autocomplete: "new-password", Redisplay: RedisplayNever},
		{ID: FieldInviteCode, InputName: "invite_code", Label: "Invite code", Kind: ValueText, Normalize: NormalizeTrim, Required: false, MinLength: 0, MaxLength: 128, Autocomplete: "off", Redisplay: RedisplayPublic},
		{ID: FieldEmailCode, InputName: "email_code", Label: "Email verification code", Kind: ValueText, Normalize: NormalizeTrim, Required: true, MinLength: 6, MaxLength: 32, Autocomplete: "one-time-code", Redisplay: RedisplayPublic},
	}, []ActionDescriptor{
		{ID: ActionSubmit, Label: "Create account"},
		{ID: ActionDeny, Label: "Cancel", SkipFormValidation: true},
	})
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Field(id FieldID) (FieldDescriptor, bool) {
	if r == nil {
		return FieldDescriptor{}, false
	}
	field, ok := r.fields[id]
	return field, ok
}

func (r *Registry) Action(id ActionID) (ActionDescriptor, bool) {
	if r == nil {
		return ActionDescriptor{}, false
	}
	action, ok := r.actions[id]
	return action, ok
}

func (r *Registry) FieldIDs() []FieldID {
	ids := make([]FieldID, 0, len(r.fields))
	for id := range r.fields {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (r *Registry) ActionIDs() []ActionID {
	ids := make([]ActionID, 0, len(r.actions))
	for id := range r.actions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
