// Package securitytrace defines secret-free, versioned security transition
// events and a deterministic offline invariant monitor.
package securitytrace

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/assurance"
)

const SchemaVersion = 1

type Kind string

const (
	InteractionCreated         Kind = "interaction.created"
	AuthenticationSatisfied    Kind = "authentication.satisfied"
	ConsentApproved            Kind = "consent.approved"
	ConsentDenied              Kind = "consent.denied"
	InteractionTerminal        Kind = "interaction.terminal"
	AuthorizationArtifactsDone Kind = "authorization.artifacts_committed"
	TokenLifecycleDone         Kind = "token.lifecycle_committed"
)

type Event struct {
	Version         int                 `json:"version"`
	Time            time.Time           `json:"time"`
	Kind            Kind                `json:"kind"`
	InteractionID   string              `json:"interaction_id,omitempty"`
	Transition      assurance.StepID    `json:"transition"`
	RequiredActions uint32              `json:"required_actions,omitempty"`
	Outcome         assurance.OutcomeID `json:"outcome"`
}

// TransitionResult is the assurance representation of a recorded native
// transition. It deliberately contains no request, client, subject, handle,
// token, credential, or grant value.
type TransitionResult struct {
	Step        assurance.StepID
	Observation assurance.ObservationID
	Outcome     assurance.OutcomeID
}

// Validate ensures that a trace event has only bounded, protocol-independent
// dimensions. InteractionID is an HMAC-derived correlation reference and must
// be a lower-case SHA-256 hex digest; it is not a user, client, or handle.
func (e Event) Validate() error {
	if e.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d", e.Version)
	}
	if e.Kind == TokenLifecycleDone {
		if e.InteractionID != "" {
			return fmt.Errorf("token lifecycle event must not carry an interaction reference")
		}
	} else if !validInteractionReference(e.InteractionID) {
		return fmt.Errorf("invalid interaction reference")
	}
	result, ok := transitionResults[e.Kind]
	if !ok {
		return fmt.Errorf("unknown event kind %q", e.Kind)
	}
	if !containsStep(result.steps, e.Transition) {
		return fmt.Errorf("event kind %q cannot report transition %q", e.Kind, e.Transition)
	}
	if !containsOutcome(result.outcomes, e.Outcome) {
		return fmt.Errorf("event kind %q cannot report outcome %q", e.Kind, e.Outcome)
	}
	return nil
}

// Result maps a validated runtime event to one declared native transition
// result. Callers must not infer a transition from arbitrary audit data.
func (e Event) Result() (TransitionResult, error) {
	if err := e.Validate(); err != nil {
		return TransitionResult{}, err
	}
	contract := transitionResults[e.Kind]
	return TransitionResult{Step: e.Transition, Observation: contract.observation, Outcome: e.Outcome}, nil
}

type eventContract struct {
	observation assurance.ObservationID
	steps       []assurance.StepID
	outcomes    []assurance.OutcomeID
}

var transitionResults = map[Kind]eventContract{
	InteractionCreated:         {assurance.ObservationInteractionCreated, []assurance.StepID{assurance.StepInteractionCreate}, []assurance.OutcomeID{assurance.TransitionApplied}},
	AuthenticationSatisfied:    {assurance.ObservationAuthenticationSatisfied, []assurance.StepID{assurance.StepPasswordAuthenticate, assurance.StepDeviceApprove}, []assurance.OutcomeID{assurance.TransitionApplied}},
	ConsentApproved:            {assurance.ObservationConsentApproved, []assurance.StepID{assurance.StepConsentGrant}, []assurance.OutcomeID{assurance.TransitionApplied}},
	ConsentDenied:              {assurance.ObservationConsentDenied, []assurance.StepID{assurance.StepInteractionDeny}, []assurance.OutcomeID{assurance.TransitionDenied}},
	InteractionTerminal:        {assurance.ObservationInteractionTerminal, []assurance.StepID{assurance.StepInteractionApprove, assurance.StepInteractionDeny, assurance.StepAccountSelection, assurance.StepDeviceApprove}, []assurance.OutcomeID{assurance.TransitionApproved, assurance.TransitionDenied}},
	AuthorizationArtifactsDone: {assurance.ObservationAuthorizationArtifacts, []assurance.StepID{assurance.StepAuthorizationCommit}, []assurance.OutcomeID{assurance.TransitionApplied}},
	TokenLifecycleDone:         {assurance.ObservationTokenLifecycle, []assurance.StepID{assurance.StepTokenIssue}, []assurance.OutcomeID{assurance.TransitionApplied}},
}

func validInteractionReference(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func containsStep(values []assurance.StepID, want assurance.StepID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsOutcome(values []assurance.OutcomeID, want assurance.OutcomeID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type Sink interface {
	EmitSecurity(ctx context.Context, event Event) error
}

type NoopSink struct{}

func (NoopSink) EmitSecurity(context.Context, Event) error { return nil }

var _ Sink = NoopSink{}

type Recorder struct {
	mu     sync.Mutex
	events []Event
}

var _ Sink = (*Recorder)(nil)

func (r *Recorder) EmitSecurity(_ context.Context, event Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

func (r *Recorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

const (
	requireLogin      uint32 = 1 << 0
	requireFreshLogin uint32 = 1 << 1
	requireConsent    uint32 = 1 << 2
)

type interactionState struct {
	required uint32
	authed   bool
	consent  bool
	terminal string
	artifact bool
}

type Monitor struct {
	interactions map[string]*interactionState
	violations   []error
}

func NewMonitor() *Monitor {
	return &Monitor{interactions: map[string]*interactionState{}}
}

func (m *Monitor) Observe(event Event) {
	if err := event.Validate(); err != nil {
		m.violate(event, "%s", err)
		return
	}
	if event.InteractionID == "" {
		return
	}
	state := m.interactions[event.InteractionID]
	switch event.Kind {
	case InteractionCreated:
		if state != nil {
			m.violate(event, "interaction was created more than once")
			return
		}
		m.interactions[event.InteractionID] = &interactionState{required: event.RequiredActions}
		return
	case AuthenticationSatisfied, ConsentApproved, ConsentDenied, InteractionTerminal, AuthorizationArtifactsDone:
		if state == nil {
			m.violate(event, "event occurred before interaction creation")
			return
		}
	case TokenLifecycleDone:
		return
	}

	switch event.Kind {
	case AuthenticationSatisfied:
		state.authed = true
	case ConsentApproved:
		state.consent = true
	case ConsentDenied:
		state.consent = false
	case InteractionTerminal:
		if state.terminal != "" {
			m.violate(event, "interaction has multiple terminal outcomes (%s then %s)", state.terminal, event.Outcome)
			return
		}
		if event.Outcome == assurance.TransitionApproved {
			if state.required&(requireLogin|requireFreshLogin) != 0 && !state.authed {
				m.violate(event, "approved terminal outcome lacks required authentication")
			}
			if state.required&requireConsent != 0 && !state.consent {
				m.violate(event, "approved terminal outcome lacks required consent")
			}
		}
		state.terminal = string(event.Outcome)
	case AuthorizationArtifactsDone:
		if state.terminal != string(assurance.TransitionApproved) {
			m.violate(event, "authorization artifacts committed before approved terminal outcome")
		}
		if state.artifact {
			m.violate(event, "authorization artifacts committed more than once")
		}
		state.artifact = true
	case InteractionCreated, TokenLifecycleDone:
		return
	}
}

func (e Event) String() string {
	return strings.Join([]string{string(e.Kind), string(e.Transition), string(e.Outcome)}, "/")
}

func (m *Monitor) Violations() []error { return append([]error(nil), m.violations...) }

func (m *Monitor) violate(event Event, format string, args ...any) {
	m.violations = append(m.violations, fmt.Errorf("%s %s: %s", event.InteractionID, event.Kind, fmt.Sprintf(format, args...)))
}
