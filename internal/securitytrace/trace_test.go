package securitytrace

import "testing"

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
