package assurance

import "testing"

func TestThreeSchemasValidateOnlyTheirOwnAuthority(t *testing.T) {
	catalog := TransitionCatalog{SchemaVersion: TransitionSchemaVersion, Transitions: []TransitionDescriptor{{
		ID: StepAuthorizationCommit, Reads: []ResourceID{ResourceInteraction, ResourceConsent}, Writes: []ResourceID{ResourceAuthorizationCode}, Requires: []FactID{FactRequestValidated, FactConsentGranted}, Effects: []EffectID{EffectIssueArtifact, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionDenied}, Observations: []ObservationID{ObservationAuthorizationArtifacts},
	}}}
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (ConfigurationReference{SchemaVersion: ConfigurationSchemaVersion, ProgramFingerprint: "sha256.program", SourceFingerprint: "sha256.source"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (ScenarioRecord{SchemaVersion: ScenarioTraceSchemaVersion, Steps: []ScenarioStep{{Step: StepAuthorizationCommit, Parameters: []byte(`{"prompt":"none"}`)}}}).Validate(catalog); err != nil {
		t.Fatal(err)
	}
	if err := (TraceRecord{SchemaVersion: ScenarioTraceSchemaVersion, Observations: []TraceObservation{{Step: StepAuthorizationCommit, Kind: ObservationAuthorizationArtifacts, Outcome: TransitionApplied}}}).Validate(catalog); err != nil {
		t.Fatal(err)
	}
}

func TestScenarioAndTraceRejectUnregisteredOrMalformedValues(t *testing.T) {
	catalog := TransitionCatalog{SchemaVersion: TransitionSchemaVersion, Transitions: []TransitionDescriptor{{ID: StepAuthorizationBegin, Outcomes: []OutcomeID{TransitionApplied}}}}
	if err := (ScenarioRecord{SchemaVersion: ScenarioTraceSchemaVersion, Steps: []ScenarioStep{{Step: "unknown@v1"}}}).Validate(catalog); err == nil {
		t.Fatal("unregistered scenario step accepted")
	}
	if err := (TraceRecord{SchemaVersion: ScenarioTraceSchemaVersion, Observations: []TraceObservation{{Step: StepAuthorizationBegin, Kind: "has space", Outcome: TransitionApplied}}}).Validate(catalog); err == nil {
		t.Fatal("malformed trace observation accepted")
	}
}
