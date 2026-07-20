// Package assurance defines the versioned, dependency-free identifiers shared
// by Tiny-IDP configuration, native transitions, verification scenarios, and
// secret-free runtime observations. It deliberately imports no protocol,
// persistence, Goja, or HTTP package.
package assurance

import "strings"

const VocabularyVersion = "tinyidp.assurance/v1"

type (
	ResourceID    string
	FactID        string
	ObligationID  string
	HandlerID     string
	SchemaID      string
	CapabilityID  string
	EffectID      string
	EvidenceID    string
	DiagnosticID  string
	ObservationID string
	OutcomeID     string
	StepID        string
	PropertyID    string
)

const (
	ResourceClient            ResourceID = "client@v1"
	ResourceInteraction       ResourceID = "interaction@v1"
	ResourceBrowserSession    ResourceID = "browser_session@v1"
	ResourceBrowserContext    ResourceID = "browser_context@v1"
	ResourceConsent           ResourceID = "consent@v1"
	ResourceCredential        ResourceID = "credential@v1"
	ResourceIdentity          ResourceID = "identity@v1"
	ResourceAuthorizationCode ResourceID = "authorization_code@v1"
	ResourceTokenFamily       ResourceID = "token_family@v1"
	ResourceDeviceGrant       ResourceID = "device_grant@v1"

	FactRequestValidated       FactID = "request.validated@v1"
	FactInteractionPending     FactID = "interaction.pending@v1"
	FactBrowserBound           FactID = "browser.bound@v1"
	FactPrincipalAuthenticated FactID = "principal.authenticated@v1"
	FactConsentGranted         FactID = "consent.granted@v1"
	FactEmailVerified          FactID = "email.verified@v1"
	FactAuthorizationCommitted FactID = "authorization.committed@v1"

	ObligationLogin            ObligationID = "authn.login.required@v1"
	ObligationFreshLogin       ObligationID = "authn.fresh.required@v1"
	ObligationConsent          ObligationID = "consent.required@v1"
	ObligationStepUp           ObligationID = "authn.step_up.required@v1"
	ObligationAccountSelection ObligationID = "account.selection.required@v1"
	ObligationRegistration     ObligationID = "registration.required@v1"

	StepInteractionCreate       StepID = "interaction.create@v1"
	StepInteractionLoad         StepID = "interaction.load@v1"
	StepInteractionApprove      StepID = "interaction.approve@v1"
	StepInteractionDeny         StepID = "interaction.deny@v1"
	StepInteractionCopyMutation StepID = "interaction.copy_mutation@v1"
	StepAccountSelection        StepID = "account.selection@v1"
	StepAuthorizationBegin      StepID = "authorize.begin@v1"
	StepAuthorizationCommit     StepID = "authorize.commit@v1"
	StepPasswordAuthenticate    StepID = "authn.password.verify@v1"
	StepConsentGrant            StepID = "consent.grant@v1"
	StepDeviceApprove           StepID = "device.approve@v1"
	StepTokenIssue              StepID = "token.issue@v1"

	EffectReadResource            EffectID = "read.resource@v1"
	EffectCreateResource          EffectID = "create.resource@v1"
	EffectUpdateResource          EffectID = "update.resource@v1"
	EffectConsumeOnce             EffectID = "consume_once@v1"
	EffectIssueArtifact           EffectID = "issue.artifact@v1"
	EffectEmitSecurityObservation EffectID = "emit.security_observation@v1"
	EffectInvokePolicy            EffectID = "invoke.policy@v1"
	EffectInvokeBoundedCapability EffectID = "invoke.bounded_capability@v1"

	TransitionApplied  OutcomeID = "applied"
	TransitionRejected OutcomeID = "rejected"
	TransitionDenied   OutcomeID = "denied"
	TransitionApproved OutcomeID = "approved"
	TransitionExpired  OutcomeID = "expired"
	TransitionConflict OutcomeID = "conflict"

	OutcomeContinue  OutcomeID = "continue"
	OutcomePresent   OutcomeID = "present"
	OutcomeChallenge OutcomeID = "challenge"
	OutcomeCommit    OutcomeID = "commit"
	OutcomeComplete  OutcomeID = "complete"
	OutcomeDeny      OutcomeID = "deny"
	OutcomeSkip      OutcomeID = "skip"
	OutcomeError     OutcomeID = "error"

	ObservationInteractionCreated      ObservationID = "interaction.created@v1"
	ObservationAuthenticationSatisfied ObservationID = "authentication.satisfied@v1"
	ObservationConsentApproved         ObservationID = "consent.approved@v1"
	ObservationConsentDenied           ObservationID = "consent.denied@v1"
	ObservationInteractionTerminal     ObservationID = "interaction.terminal@v1"
	ObservationAuthorizationArtifacts  ObservationID = "authorization.artifacts_committed@v1"
	ObservationTokenLifecycle          ObservationID = "token.lifecycle_committed@v1"
	ObservationLambdaStarted           ObservationID = "lambda.started@v1"
	ObservationLambdaCompleted         ObservationID = "lambda.completed@v1"
	ObservationLambdaRejected          ObservationID = "lambda.rejected@v1"
	ObservationContinuationCreated     ObservationID = "continuation.created@v1"
	ObservationContinuationTerminal    ObservationID = "continuation.terminal@v1"
	ObservationEvidenceVerified        ObservationID = "evidence.verified@v1"
	ObservationNativeEffectCommitted   ObservationID = "effect.committed@v1"

	PropertyInteractionCreatedOnce            PropertyID = "interaction.created_once@v1"
	PropertyRequiredAuthBeforeApproval        PropertyID = "authorization.required_auth_before_approval@v1"
	PropertyRequiredConsentBeforeApproval     PropertyID = "authorization.required_consent_before_approval@v1"
	PropertyInteractionSingleTerminal         PropertyID = "interaction.single_terminal@v1"
	PropertyArtifactsAfterApproval            PropertyID = "authorization.artifacts_after_approval@v1"
	PropertyArtifactsOnce                     PropertyID = "authorization.artifacts_once@v1"
	PropertyAuthorizationRequiresNativeCommit PropertyID = "authorization.requires_native_commit@v1"
)

// ValidStableID accepts the bounded ASCII identifier vocabulary used in
// diagnostics, evidence, handler names, and machine-readable observations. It
// deliberately excludes user-controlled free text and whitespace.
func ValidStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	versionAt := strings.IndexByte(value, '@')
	if versionAt >= 0 {
		if strings.Count(value, "@") != 1 || versionAt == 0 || versionAt+2 >= len(value) || value[versionAt+1] != 'v' {
			return false
		}
		for _, r := range value[versionAt+2:] {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' || r == '@' {
			continue
		}
		return false
	}
	return true
}
