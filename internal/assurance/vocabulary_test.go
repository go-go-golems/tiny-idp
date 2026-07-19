package assurance

import "testing"

func TestVocabularyUsesBoundedStableIdentifiers(t *testing.T) {
	for _, value := range []string{
		string(OutcomePresent),
		string(ObservationLambdaCompleted),
		string(PropertyAuthorizationRequiresNativeCommit),
		"signup.submitted",
		"createLocalIdentity",
	} {
		if !ValidStableID(value) {
			t.Fatalf("stable identifier %q rejected", value)
		}
	}
	for _, value := range []string{"", "has space", "user@example.com", "line\nbreak"} {
		if ValidStableID(value) {
			t.Fatalf("unstable identifier %q accepted", value)
		}
	}
}
