package assurance

import "testing"

func TestVocabularyUsesBoundedStableIdentifiers(t *testing.T) {
	for _, value := range []string{
		string(ResourceInteraction),
		string(FactRequestValidated),
		string(ObligationConsent),
		string(StepAuthorizationCommit),
		string(EffectIssueArtifact),
		string(TransitionApplied),
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

func TestVocabularyHasNoDuplicateContractIdentifiers(t *testing.T) {
	values := []string{
		string(ResourceClient), string(ResourceInteraction), string(ResourceBrowserSession), string(ResourceBrowserContext), string(ResourceConsent), string(ResourceCredential), string(ResourceIdentity), string(ResourceAuthorizationCode), string(ResourceTokenFamily), string(ResourceDeviceGrant),
		string(FactRequestValidated), string(FactInteractionPending), string(FactBrowserBound), string(FactPrincipalAuthenticated), string(FactConsentGranted), string(FactEmailVerified), string(FactAuthorizationCommitted),
		string(ObligationLogin), string(ObligationFreshLogin), string(ObligationConsent), string(ObligationStepUp), string(ObligationAccountSelection), string(ObligationRegistration),
		string(StepInteractionCreate), string(StepInteractionLoad), string(StepInteractionApprove), string(StepInteractionDeny), string(StepInteractionCopyMutation), string(StepAuthorizationBegin), string(StepAuthorizationCommit), string(StepPasswordAuthenticate), string(StepConsentGrant), string(StepDeviceApprove),
		string(EffectReadResource), string(EffectCreateResource), string(EffectUpdateResource), string(EffectConsumeOnce), string(EffectIssueArtifact), string(EffectEmitSecurityObservation), string(EffectInvokePolicy), string(EffectInvokeBoundedCapability),
		string(TransitionApplied), string(TransitionRejected), string(TransitionDenied), string(TransitionExpired), string(TransitionConflict),
		string(ObservationInteractionCreated), string(ObservationAuthenticationSatisfied), string(ObservationConsentApproved), string(ObservationConsentDenied), string(ObservationInteractionTerminal), string(ObservationAuthorizationArtifacts), string(ObservationTokenLifecycle), string(ObservationLambdaStarted), string(ObservationLambdaCompleted), string(ObservationLambdaRejected), string(ObservationContinuationCreated), string(ObservationContinuationTerminal), string(ObservationEvidenceVerified), string(ObservationNativeEffectCommitted),
		string(PropertyInteractionCreatedOnce), string(PropertyRequiredAuthBeforeApproval), string(PropertyRequiredConsentBeforeApproval), string(PropertyInteractionSingleTerminal), string(PropertyArtifactsAfterApproval), string(PropertyArtifactsOnce), string(PropertyAuthorizationRequiresNativeCommit),
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if !ValidStableID(value) {
			t.Fatalf("contract identifier %q is invalid", value)
		}
		if _, duplicate := seen[value]; duplicate {
			t.Fatalf("contract identifier %q is duplicated", value)
		}
		seen[value] = struct{}{}
	}
}
