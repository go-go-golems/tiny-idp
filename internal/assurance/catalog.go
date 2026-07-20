package assurance

import (
	"sort"

	"github.com/pkg/errors"
)

const (
	StaticAnalysisSchemaVersion = "tinyidp.assurance.static-analysis/v1"
	FormalModelSchemaVersion    = "tinyidp.assurance.formal-model/v1"
)

// StaticAnalysisMetadata is a deterministic, data-only projection of native
// transition authority. It tells an analyzer which stable handler is allowed
// to implement a transition and which security-relevant effects it declares;
// it does not dispatch handlers or grant runtime authority.
type StaticAnalysisMetadata struct {
	SchemaVersion string              `json:"schemaVersion"`
	Authorities   []AuthorityMetadata `json:"authorities"`
}

type AuthorityMetadata struct {
	Handler      HandlerID       `json:"handler"`
	Step         StepID          `json:"step"`
	Effects      []EffectID      `json:"effects"`
	ArtifactSink bool            `json:"artifactSink"`
	Observations []ObservationID `json:"observations"`
}

// FormalModelVocabulary is a finite action/resource/fact/obligation projection
// from which a model author can start a bounded model. It is intentionally not
// a formal model: it supplies no scheduler, fairness, state representation, or
// protocol implementation.
type FormalModelVocabulary struct {
	SchemaVersion string         `json:"schemaVersion"`
	Resources     []ResourceID   `json:"resources"`
	Facts         []FactID       `json:"facts"`
	Obligations   []ObligationID `json:"obligations"`
	Outcomes      []OutcomeID    `json:"outcomes"`
	Actions       []ModelAction  `json:"actions"`
}

type ModelAction struct {
	Step       StepID         `json:"step"`
	Requires   []FactID       `json:"requires,omitempty"`
	Produces   []FactID       `json:"produces,omitempty"`
	Discharges []ObligationID `json:"discharges,omitempty"`
	MayCreate  []ObligationID `json:"mayCreate,omitempty"`
	Outcomes   []OutcomeID    `json:"outcomes"`
}

// GenerateStaticAnalysisMetadata projects each catalog authority into a
// reviewable effect summary. Duplicate (handler, step) pairs are rejected,
// because they would make a downstream source-symbol binding ambiguous.
func GenerateStaticAnalysisMetadata(catalog TransitionCatalog) (StaticAnalysisMetadata, error) {
	if err := catalog.Validate(); err != nil {
		return StaticAnalysisMetadata{}, errors.Wrap(err, "validate transition catalog")
	}
	metadata := StaticAnalysisMetadata{SchemaVersion: StaticAnalysisSchemaVersion}
	seen := map[string]struct{}{}
	for _, transition := range catalog.Transitions {
		for _, handler := range transition.Authorities {
			key := string(handler) + "\x00" + string(transition.ID)
			if _, duplicate := seen[key]; duplicate {
				return StaticAnalysisMetadata{}, errors.Errorf("duplicate authority %q for transition %q", handler, transition.ID)
			}
			seen[key] = struct{}{}
			metadata.Authorities = append(metadata.Authorities, AuthorityMetadata{
				Handler:      handler,
				Step:         transition.ID,
				Effects:      sortedCopy(transition.Effects),
				ArtifactSink: containsEffect(transition.Effects, EffectIssueArtifact),
				Observations: sortedCopy(transition.Observations),
			})
		}
	}
	sort.Slice(metadata.Authorities, func(i, j int) bool {
		if metadata.Authorities[i].Handler == metadata.Authorities[j].Handler {
			return metadata.Authorities[i].Step < metadata.Authorities[j].Step
		}
		return metadata.Authorities[i].Handler < metadata.Authorities[j].Handler
	})
	return metadata, nil
}

// GenerateFormalModelVocabulary derives finite vocabulary sets and one action
// summary per transition. It deliberately erases source symbols, request data,
// users, credentials, tokens, and other unbounded implementation details.
func GenerateFormalModelVocabulary(catalog TransitionCatalog) (FormalModelVocabulary, error) {
	if err := catalog.Validate(); err != nil {
		return FormalModelVocabulary{}, errors.Wrap(err, "validate transition catalog")
	}
	vocabulary := FormalModelVocabulary{SchemaVersion: FormalModelSchemaVersion}
	resources := map[ResourceID]struct{}{}
	facts := map[FactID]struct{}{}
	obligations := map[ObligationID]struct{}{}
	outcomes := map[OutcomeID]struct{}{}
	for _, transition := range catalog.Transitions {
		for _, resource := range append(append([]ResourceID(nil), transition.Reads...), transition.Writes...) {
			resources[resource] = struct{}{}
		}
		for _, fact := range append(append([]FactID(nil), transition.Requires...), transition.Produces...) {
			facts[fact] = struct{}{}
		}
		for _, obligation := range append(append([]ObligationID(nil), transition.Discharges...), transition.MayCreate...) {
			obligations[obligation] = struct{}{}
		}
		for _, outcome := range transition.Outcomes {
			outcomes[outcome] = struct{}{}
		}
		vocabulary.Actions = append(vocabulary.Actions, ModelAction{Step: transition.ID, Requires: sortedCopy(transition.Requires), Produces: sortedCopy(transition.Produces), Discharges: sortedCopy(transition.Discharges), MayCreate: sortedCopy(transition.MayCreate), Outcomes: sortedCopy(transition.Outcomes)})
	}
	vocabulary.Resources = sortedSet(resources)
	vocabulary.Facts = sortedSet(facts)
	vocabulary.Obligations = sortedSet(obligations)
	vocabulary.Outcomes = sortedSet(outcomes)
	sort.Slice(vocabulary.Actions, func(i, j int) bool { return vocabulary.Actions[i].Step < vocabulary.Actions[j].Step })
	return vocabulary, nil
}

// CurrentAuthorizationCatalog describes the native authorization transitions
// presently implemented by the Fosite adapter. Its IDs are contracts for
// analysis and modeling, not a replacement for the handler control flow.
func CurrentAuthorizationCatalog() TransitionCatalog {
	return TransitionCatalog{SchemaVersion: TransitionSchemaVersion, Transitions: []TransitionDescriptor{
		{ID: StepInteractionCreate, Authorities: []HandlerID{HandlerAuthorizeBegin}, Reads: []ResourceID{ResourceClient, ResourceBrowserSession, ResourceBrowserContext}, Writes: []ResourceID{ResourceInteraction}, Produces: []FactID{FactRequestValidated, FactInteractionPending, FactBrowserBound}, MayCreate: []ObligationID{ObligationLogin, ObligationFreshLogin, ObligationConsent, ObligationAccountSelection, ObligationRegistration}, Effects: []EffectID{EffectCreateResource, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected}, Observations: []ObservationID{ObservationInteractionCreated}},
		{ID: StepPasswordAuthenticate, Authorities: []HandlerID{HandlerAuthorizeResume}, Reads: []ResourceID{ResourceInteraction, ResourceCredential, ResourceIdentity}, Writes: []ResourceID{ResourceBrowserSession, ResourceCredential}, Produces: []FactID{FactPrincipalAuthenticated}, Discharges: []ObligationID{ObligationLogin, ObligationFreshLogin}, Effects: []EffectID{EffectReadResource, EffectUpdateResource, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected}, Observations: []ObservationID{ObservationAuthenticationSatisfied}},
		{ID: StepConsentGrant, Authorities: []HandlerID{HandlerAuthorizeResume}, Reads: []ResourceID{ResourceInteraction, ResourceClient, ResourceIdentity}, Writes: []ResourceID{ResourceConsent}, Produces: []FactID{FactConsentGranted}, Discharges: []ObligationID{ObligationConsent}, Effects: []EffectID{EffectReadResource, EffectUpdateResource, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected}, Observations: []ObservationID{ObservationConsentApproved}},
		{ID: StepInteractionDeny, Authorities: []HandlerID{HandlerAuthorizeResume}, Reads: []ResourceID{ResourceInteraction}, Writes: []ResourceID{ResourceInteraction}, Effects: []EffectID{EffectConsumeOnce, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionDenied, TransitionConflict}, Observations: []ObservationID{ObservationConsentDenied, ObservationInteractionTerminal}},
		{ID: StepAccountSelection, Authorities: []HandlerID{HandlerAuthorizeResume}, Reads: []ResourceID{ResourceInteraction, ResourceBrowserContext}, Writes: []ResourceID{ResourceInteraction, ResourceBrowserSession}, MayCreate: []ObligationID{ObligationFreshLogin, ObligationConsent}, Effects: []EffectID{EffectReadResource, EffectUpdateResource, EffectConsumeOnce, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected, TransitionConflict}, Observations: []ObservationID{ObservationInteractionTerminal}},
		{ID: StepDeviceApprove, Authorities: []HandlerID{HandlerDeviceVerification}, Reads: []ResourceID{ResourceDeviceGrant, ResourceCredential, ResourceInteraction}, Writes: []ResourceID{ResourceDeviceGrant, ResourceInteraction}, Produces: []FactID{FactPrincipalAuthenticated}, Effects: []EffectID{EffectReadResource, EffectUpdateResource, EffectConsumeOnce, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionDenied, TransitionConflict}, Observations: []ObservationID{ObservationAuthenticationSatisfied, ObservationInteractionTerminal}},
		{ID: StepAuthorizationCommit, Authorities: []HandlerID{HandlerAuthorizationArtifact}, Reads: []ResourceID{ResourceInteraction, ResourceClient, ResourceConsent, ResourceIdentity}, Writes: []ResourceID{ResourceInteraction, ResourceAuthorizationCode}, Requires: []FactID{FactRequestValidated, FactInteractionPending}, Effects: []EffectID{EffectConsumeOnce, EffectIssueArtifact, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected, TransitionConflict}, Observations: []ObservationID{ObservationInteractionTerminal, ObservationAuthorizationArtifacts}},
		{ID: StepTokenIssue, Authorities: []HandlerID{HandlerTokenIssue}, Reads: []ResourceID{ResourceAuthorizationCode, ResourceTokenFamily}, Writes: []ResourceID{ResourceAuthorizationCode, ResourceTokenFamily}, Effects: []EffectID{EffectConsumeOnce, EffectIssueArtifact, EffectEmitSecurityObservation}, Outcomes: []OutcomeID{TransitionApplied, TransitionRejected, TransitionConflict}, Observations: []ObservationID{ObservationTokenLifecycle}},
	}}
}

func containsEffect(values []EffectID, want EffectID) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sortedCopy[T ~string](values []T) []T {
	result := append([]T(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sortedSet[T ~string](values map[T]struct{}) []T {
	result := make([]T, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
