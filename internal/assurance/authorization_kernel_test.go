package assurance

import (
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestAuthorizationKernelEnforcesPureTerminalOrdering(t *testing.T) {
	kernel, err := NewAuthorizationKernel(uint32(idpstore.InteractionRequireFreshLogin | idpstore.InteractionRequireConsent))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []struct {
		observation ObservationID
		outcome     OutcomeID
	}{
		{ObservationAuthenticationSatisfied, TransitionApplied},
		{ObservationConsentApproved, TransitionApplied},
		{ObservationInteractionTerminal, TransitionApproved},
		{ObservationAuthorizationArtifacts, TransitionApplied},
	} {
		if violations := kernel.Apply(event.observation, event.outcome); len(violations) != 0 {
			t.Fatal(violations)
		}
	}
}

func TestAuthorizationKernelRejectsMissingEvidenceAndDuplicateArtifact(t *testing.T) {
	kernel, err := NewAuthorizationKernel(uint32(idpstore.InteractionRequireLogin))
	if err != nil {
		t.Fatal(err)
	}
	if violations := kernel.Apply(ObservationInteractionTerminal, TransitionApproved); len(violations) == 0 {
		t.Fatal("missing authentication accepted")
	}
	kernel, err = NewAuthorizationKernel(0)
	if err != nil {
		t.Fatal(err)
	}
	if violations := kernel.Apply(ObservationInteractionTerminal, TransitionApproved); len(violations) != 0 {
		t.Fatal(violations)
	}
	if violations := kernel.Apply(ObservationAuthorizationArtifacts, TransitionApplied); len(violations) != 0 {
		t.Fatal(violations)
	}
	if violations := kernel.Apply(ObservationAuthorizationArtifacts, TransitionApplied); len(violations) == 0 {
		t.Fatal("duplicate artifact accepted")
	}
}
