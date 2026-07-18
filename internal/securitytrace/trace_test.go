package securitytrace

import (
	"testing"

	"pgregory.net/rapid"
)

func TestMonitorAcceptsRequiredAuthenticationConsentAndArtifactOrder(t *testing.T) {
	monitor := NewMonitor()
	for _, event := range []Event{
		{Version: SchemaVersion, Kind: InteractionCreated, InteractionID: "i1", RequiredActions: requireFreshLogin | requireConsent},
		{Version: SchemaVersion, Kind: AuthenticationSatisfied, InteractionID: "i1"},
		{Version: SchemaVersion, Kind: ConsentApproved, InteractionID: "i1"},
		{Version: SchemaVersion, Kind: InteractionTerminal, InteractionID: "i1", Outcome: "approved"},
		{Version: SchemaVersion, Kind: AuthorizationArtifactsDone, InteractionID: "i1"},
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
		events := []Event{{Version: SchemaVersion, Kind: InteractionCreated, InteractionID: "generated", RequiredActions: required}}
		if required&requireFreshLogin != 0 {
			events = append(events, Event{Version: SchemaVersion, Kind: AuthenticationSatisfied, InteractionID: "generated"})
		}
		if approved && required&requireConsent != 0 {
			events = append(events, Event{Version: SchemaVersion, Kind: ConsentApproved, InteractionID: "generated"})
		}
		outcome := "approved"
		if !approved {
			outcome = "denied"
			events = append(events, Event{Version: SchemaVersion, Kind: ConsentDenied, InteractionID: "generated"})
		}
		events = append(events, Event{Version: SchemaVersion, Kind: InteractionTerminal, InteractionID: "generated", Outcome: outcome})
		if approved {
			events = append(events, Event{Version: SchemaVersion, Kind: AuthorizationArtifactsDone, InteractionID: "generated"})
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
			outcome := "approved"
			if value&1 != 0 {
				outcome = "denied"
			}
			version := SchemaVersion
			if value == 255 {
				version = 99
			}
			monitor.Observe(Event{Version: version, Kind: kind, InteractionID: string(rune('a' + index%4)), RequiredActions: uint32(value) & 7, Outcome: outcome})
		}
		_ = monitor.Violations()
	})
}

func TestMonitorRejectsMissingActionsDuplicateTerminalAndArtifact(t *testing.T) {
	monitor := NewMonitor()
	for _, event := range []Event{
		{Version: SchemaVersion, Kind: InteractionCreated, InteractionID: "i1", RequiredActions: requireLogin | requireConsent},
		{Version: SchemaVersion, Kind: InteractionTerminal, InteractionID: "i1", Outcome: "approved"},
		{Version: SchemaVersion, Kind: InteractionTerminal, InteractionID: "i1", Outcome: "denied"},
		{Version: SchemaVersion, Kind: AuthorizationArtifactsDone, InteractionID: "i1"},
		{Version: SchemaVersion, Kind: AuthorizationArtifactsDone, InteractionID: "i1"},
	} {
		monitor.Observe(event)
	}
	if violations := monitor.Violations(); len(violations) != 4 {
		t.Fatalf("violations=%d want 4: %v", len(violations), violations)
	}
}
