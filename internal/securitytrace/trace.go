// Package securitytrace defines secret-free, versioned security transition
// events and a deterministic offline invariant monitor.
package securitytrace

import (
	"context"
	"fmt"
	"sync"
	"time"
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
	Version         int       `json:"version"`
	Time            time.Time `json:"time"`
	Kind            Kind      `json:"kind"`
	InteractionID   string    `json:"interaction_id,omitempty"`
	RequestID       string    `json:"request_id,omitempty"`
	ClientID        string    `json:"client_id,omitempty"`
	RequiredActions uint32    `json:"required_actions,omitempty"`
	Outcome         string    `json:"outcome,omitempty"`
	GrantType       string    `json:"grant_type,omitempty"`
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
	if event.Version != SchemaVersion {
		m.violate(event, "unsupported schema version %d", event.Version)
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
		if event.Outcome == "approved" {
			if state.required&(requireLogin|requireFreshLogin) != 0 && !state.authed {
				m.violate(event, "approved terminal outcome lacks required authentication")
			}
			if state.required&requireConsent != 0 && !state.consent {
				m.violate(event, "approved terminal outcome lacks required consent")
			}
		}
		state.terminal = event.Outcome
	case AuthorizationArtifactsDone:
		if state.terminal != "approved" {
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

func (m *Monitor) Violations() []error { return append([]error(nil), m.violations...) }

func (m *Monitor) violate(event Event, format string, args ...any) {
	m.violations = append(m.violations, fmt.Errorf("%s %s: %s", event.InteractionID, event.Kind, fmt.Sprintf(format, args...)))
}
