package idpprogram

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// OutcomeKind is the closed family of results a lambda may return.
type OutcomeKind string

const (
	OutcomeContinue  OutcomeKind = "continue"
	OutcomePresent   OutcomeKind = "present"
	OutcomeChallenge OutcomeKind = "challenge"
	OutcomeCommit    OutcomeKind = "commit"
	OutcomeComplete  OutcomeKind = "complete"
	OutcomeDeny      OutcomeKind = "deny"
	OutcomeSkip      OutcomeKind = "skip"
	OutcomeError     OutcomeKind = "error"
)

// Valid reports whether k is a supported outcome family.
func (k OutcomeKind) Valid() bool {
	switch k {
	case OutcomeContinue, OutcomePresent, OutcomeChallenge, OutcomeCommit,
		OutcomeComplete, OutcomeDeny, OutcomeSkip, OutcomeError:
		return true
	default:
		return false
	}
}

// BrowserContinuation names the native handler that will receive a later
// browser request. Carry must satisfy the destination schema and is bounded by
// the compiler/runtime.
type BrowserContinuation struct {
	HandlerID string          `json:"handlerId"`
	Carry     json.RawMessage `json:"carry,omitempty"`
	ExpiresIn int64           `json:"expiresInSeconds"`
}

// EffectPlan is a native operation request; JavaScript never executes it.
type EffectPlan struct {
	Kind    EffectKind      `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Outcome is the copied, VM-independent value returned by one invocation.
type Outcome struct {
	Kind         OutcomeKind          `json:"kind"`
	Code         string               `json:"code,omitempty"`
	NextHandler  string               `json:"nextHandler,omitempty"`
	Continuation *BrowserContinuation `json:"continuation,omitempty"`
	// Presentation is a data-only browser page descriptor. It is populated by
	// ctx.present.* and decoded by the native workflow boundary before a
	// continuation is persisted or a page is rendered.
	Presentation json.RawMessage `json:"presentation,omitempty"`
	// Challenge is a typed native challenge request emitted only by ctx.challenge.
	Challenge json.RawMessage `json:"challenge,omitempty"`
	Effects   []EffectPlan    `json:"effects,omitempty"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// ValidateOutcome checks the dynamic parts of an outcome against a lambda's
// declared contract. Workflow-edge compatibility is checked by Program.Validate.
func ValidateOutcome(spec LambdaSpec, outcome Outcome) error {
	if !outcome.Kind.Valid() {
		return errors.Errorf("invalid outcome kind %q", outcome.Kind)
	}
	if !containsOutcome(spec.AllowedOutcomes, outcome.Kind) {
		return errors.Errorf("lambda %q returned undeclared outcome %q", spec.ID, outcome.Kind)
	}

	if outcome.Kind == OutcomePresent || outcome.Kind == OutcomeChallenge {
		if outcome.Continuation == nil || outcome.Continuation.HandlerID == "" {
			return errors.Errorf("outcome %q requires a continuation handler", outcome.Kind)
		}
	} else if outcome.Continuation != nil {
		return errors.Errorf("outcome %q must not contain a browser continuation", outcome.Kind)
	}
	if len(outcome.Presentation) != 0 && outcome.Kind != OutcomePresent {
		return errors.Errorf("outcome %q must not contain a presentation", outcome.Kind)
	}
	if len(outcome.Challenge) != 0 && outcome.Kind != OutcomeChallenge {
		return errors.Errorf("outcome %q must not contain a challenge request", outcome.Kind)
	}
	if outcome.Kind == OutcomeChallenge && len(outcome.Challenge) == 0 {
		return errors.New("challenge outcome requires a challenge request")
	}

	if outcome.Kind == OutcomeContinue && outcome.NextHandler == "" {
		return errors.New("continue outcome requires nextHandler")
	}
	if outcome.Kind != OutcomeContinue && outcome.NextHandler != "" {
		return errors.Errorf("outcome %q must not contain nextHandler", outcome.Kind)
	}

	if outcome.Kind == OutcomeCommit {
		if len(outcome.Effects) == 0 {
			return errors.New("commit outcome requires at least one effect")
		}
		for _, effect := range outcome.Effects {
			if !effect.Kind.Valid() {
				return errors.Errorf("invalid effect kind %q", effect.Kind)
			}
			if !containsEffect(spec.AllowedEffects, effect.Kind) {
				return errors.Errorf("lambda %q returned undeclared effect %q", spec.ID, effect.Kind)
			}
		}
	} else if len(outcome.Effects) != 0 {
		return errors.Errorf("outcome %q must not contain effects", outcome.Kind)
	}

	return nil
}

func containsOutcome(values []OutcomeKind, candidate OutcomeKind) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func containsEffect(values []EffectKind, candidate EffectKind) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
