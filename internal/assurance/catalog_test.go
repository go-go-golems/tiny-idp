package assurance

import "testing"

func TestCurrentAuthorizationCatalogGeneratesStaticAndModelMetadata(t *testing.T) {
	catalog := CurrentAuthorizationCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
	metadata, err := GenerateStaticAnalysisMetadata(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Authorities) != len(catalog.Transitions) {
		t.Fatalf("authority metadata=%d want %d", len(metadata.Authorities), len(catalog.Transitions))
	}
	artifactSinks := 0
	for _, authority := range metadata.Authorities {
		if authority.ArtifactSink {
			artifactSinks++
			if authority.Handler != HandlerAuthorizationArtifact && authority.Handler != HandlerTokenIssue {
				t.Fatalf("unexpected artifact authority=%q", authority.Handler)
			}
		}
	}
	if artifactSinks != 2 {
		t.Fatalf("artifact sinks=%d want 2", artifactSinks)
	}
	vocabulary, err := GenerateFormalModelVocabulary(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(vocabulary.Actions) != len(catalog.Transitions) || len(vocabulary.Resources) == 0 || len(vocabulary.Outcomes) == 0 {
		t.Fatalf("incomplete model vocabulary=%#v", vocabulary)
	}
	for _, action := range vocabulary.Actions {
		if action.Step == StepAuthorizationCommit && !hasOutcome(action.Outcomes, TransitionApplied) {
			t.Fatalf("authorization commit lost applied outcome: %#v", action)
		}
	}
}

func hasOutcome(values []OutcomeID, want OutcomeID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCatalogProjectionRejectsMissingAuthority(t *testing.T) {
	catalog := TransitionCatalog{SchemaVersion: TransitionSchemaVersion, Transitions: []TransitionDescriptor{{ID: StepAuthorizationBegin, Outcomes: []OutcomeID{TransitionApplied}}}}
	if _, err := GenerateStaticAnalysisMetadata(catalog); err == nil {
		t.Fatal("catalog without a native authority was accepted")
	}
	if _, err := GenerateFormalModelVocabulary(catalog); err == nil {
		t.Fatal("model vocabulary accepted catalog without a native authority")
	}
}
