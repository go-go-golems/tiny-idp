package securitytrace

import (
	"fmt"
	"testing"

	"github.com/go-go-golems/tiny-idp/internal/assurance"
	"pgregory.net/rapid"
)

func traceEvent(kind Kind, interactionID string, required uint32, outcome assurance.OutcomeID) Event {
	event := Event{Version: SchemaVersion, Kind: kind, InteractionID: interactionID, RequiredActions: required, Outcome: outcome}
	switch kind {
	case InteractionCreated:
		event.Transition = assurance.StepInteractionCreate
		event.Outcome = assurance.TransitionApplied
	case AuthenticationSatisfied:
		event.Transition = assurance.StepPasswordAuthenticate
		event.Outcome = assurance.TransitionApplied
	case ConsentApproved:
		event.Transition = assurance.StepConsentGrant
		event.Outcome = assurance.TransitionApplied
	case ConsentDenied:
		event.Transition = assurance.StepInteractionDeny
		event.Outcome = assurance.TransitionDenied
	case InteractionTerminal:
		if outcome == assurance.TransitionDenied {
			event.Transition = assurance.StepInteractionDeny
		} else {
			event.Transition = assurance.StepInteractionApprove
		}
	case AuthorizationArtifactsDone:
		event.Transition = assurance.StepAuthorizationCommit
		event.Outcome = assurance.TransitionApplied
	case TokenLifecycleDone:
		event.Transition = assurance.StepTokenIssue
		event.Outcome = assurance.TransitionApplied
	}
	return event
}

func traceID(index int) string { return fmt.Sprintf("%064x", index) }

func TestMonitorAcceptsRequiredAuthenticationConsentAndArtifactOrder(t *testing.T) {
	monitor := NewMonitor()
	for _, event := range []Event{
		traceEvent(InteractionCreated, traceID(1), requireFreshLogin|requireConsent, assurance.TransitionApplied),
		traceEvent(AuthenticationSatisfied, traceID(1), 0, assurance.TransitionApplied),
		traceEvent(ConsentApproved, traceID(1), 0, assurance.TransitionApplied),
		traceEvent(InteractionTerminal, traceID(1), 0, assurance.TransitionApproved),
		traceEvent(AuthorizationArtifactsDone, traceID(1), 0, assurance.TransitionApplied),
	} {
		monitor.Observe(event)
	}
	if violations := monitor.Violations(); len(violations) != 0 {
		t.Fatalf("valid trace violations=%v", violations)
	}
}

func TestMonitorAcceptsGeneratedValidTraces(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		required := uint32(0)
		if rapid.Bool().Draw(t, "require_auth") {
			required |= requireFreshLogin
		}
		if rapid.Bool().Draw(t, "require_consent") {
			required |= requireConsent
		}
		approved := rapid.Bool().Draw(t, "approved")
		events := []Event{traceEvent(InteractionCreated, traceID(2), required, assurance.TransitionApplied)}
		if required&requireFreshLogin != 0 {
			events = append(events, traceEvent(AuthenticationSatisfied, traceID(2), 0, assurance.TransitionApplied))
		}
		if approved && required&requireConsent != 0 {
			events = append(events, traceEvent(ConsentApproved, traceID(2), 0, assurance.TransitionApplied))
		}
		outcome := assurance.TransitionApproved
		if !approved {
			outcome = assurance.TransitionDenied
			events = append(events, traceEvent(ConsentDenied, traceID(2), 0, assurance.TransitionDenied))
		}
		events = append(events, traceEvent(InteractionTerminal, traceID(2), 0, outcome))
		if approved {
			events = append(events, traceEvent(AuthorizationArtifactsDone, traceID(2), 0, assurance.TransitionApplied))
		}
		monitor := NewMonitor()
		for _, event := range events {
			monitor.Observe(event)
		}
		if violations := monitor.Violations(); len(violations) != 0 {
			t.Fatalf("valid generated trace violations=%v events=%#v", violations, events)
		}
	})
}

func FuzzMonitorEventSequences(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4})
	f.Add([]byte{4, 0, 3, 3, 4})
	kinds := []Kind{InteractionCreated, AuthenticationSatisfied, ConsentApproved, ConsentDenied, InteractionTerminal, AuthorizationArtifactsDone}
	f.Fuzz(func(t *testing.T, sequence []byte) {
		if len(sequence) > 256 {
			sequence = sequence[:256]
		}
		monitor := NewMonitor()
		for index, value := range sequence {
			kind := kinds[int(value)%len(kinds)]
			outcome := assurance.TransitionApproved
			if value&1 != 0 {
				outcome = assurance.TransitionDenied
			}
			event := traceEvent(kind, traceID(index%4), uint32(value)&7, outcome)
			if value == 255 {
				event.Version = 99
			}
			monitor.Observe(event)
		}
		_ = monitor.Violations()
	})
}

func TestMonitorRejectsMissingActionsDuplicateTerminalAndArtifact(t *testing.T) {
	monitor := NewMonitor()
	for _, event := range []Event{
		traceEvent(InteractionCreated, traceID(3), requireLogin|requireConsent, assurance.TransitionApplied),
		traceEvent(InteractionTerminal, traceID(3), 0, assurance.TransitionApproved),
		traceEvent(InteractionTerminal, traceID(3), 0, assurance.TransitionDenied),
		traceEvent(AuthorizationArtifactsDone, traceID(3), 0, assurance.TransitionApplied),
		traceEvent(AuthorizationArtifactsDone, traceID(3), 0, assurance.TransitionApplied),
	} {
		monitor.Observe(event)
	}
	if violations := monitor.Violations(); len(violations) != 4 {
		t.Fatalf("violations=%d want 4: %v", len(violations), violations)
	}
}
