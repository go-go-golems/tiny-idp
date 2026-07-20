package assurance

import (
	"fmt"
	"sort"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

// The lambda outcome IDs are the assurance names for the fixed wire outcomes
// defined by idpprogram. The wire values remain the public JavaScript API;
// analysis and traces use these versioned names instead of giving arbitrary
// configuration text protocol authority.
const (
	LambdaOutcomeContinue  OutcomeID = "lambda.continue@v1"
	LambdaOutcomePresent   OutcomeID = "lambda.present@v1"
	LambdaOutcomeChallenge OutcomeID = "lambda.challenge@v1"
	LambdaOutcomeCommit    OutcomeID = "lambda.commit@v1"
	LambdaOutcomeComplete  OutcomeID = "lambda.complete@v1"
	LambdaOutcomeDeny      OutcomeID = "lambda.deny@v1"
	LambdaOutcomeSkip      OutcomeID = "lambda.skip@v1"
	LambdaOutcomeError     OutcomeID = "lambda.error@v1"

	EffectCreateLocalIdentity      EffectID = "identity.local.create@v1"
	EffectAttachPasswordCredential EffectID = "credential.password.attach@v1"
	EffectConsumeInvitation        EffectID = "invitation.consume@v1"
	EffectEstablishBrowserSession  EffectID = "browser_session.establish@v1"
	EffectEstablishVirtualIdentity EffectID = "identity.virtual.establish@v1"
	EffectSendEmailChallenge       EffectID = "email_challenge.send@v1"
)

// ProgramVocabulary is the typed assurance projection of a compiled,
// runtime-independent Goja program. It is analysis data, not a second program
// representation and not an activation or dispatch mechanism.
type ProgramVocabulary struct {
	Schemas      []SchemaID
	Capabilities []CapabilityID
	Outcomes     []OutcomeID
	Effects      []EffectID
}

// VocabularyForProgram validates the configuration grammar first, then maps
// its finite declared schema, capability, outcome, and effect vocabulary to
// the shared assurance identifier types. Configuration IDs are intentionally
// projected, never promoted to native HandlerID authority.
func VocabularyForProgram(program idpprogram.Program) (ProgramVocabulary, error) {
	if diagnostics := idpprogram.Validate(program); diagnostics.HasErrors() {
		return ProgramVocabulary{}, fmt.Errorf("invalid program cannot produce assurance vocabulary: %v", diagnostics)
	}

	schemas := make([]SchemaID, 0, len(program.Schemas))
	for id := range program.Schemas {
		if !ValidStableID(id) {
			return ProgramVocabulary{}, fmt.Errorf("invalid configuration schema ID %q", id)
		}
		schemas = append(schemas, SchemaID(id))
	}
	capabilities := make([]CapabilityID, 0, len(program.Capabilities))
	for id := range program.Capabilities {
		if !ValidStableID(id) {
			return ProgramVocabulary{}, fmt.Errorf("invalid configuration capability ID %q", id)
		}
		capabilities = append(capabilities, CapabilityID(id))
	}
	outcomes := map[OutcomeID]struct{}{}
	effects := map[EffectID]struct{}{}
	for _, lambda := range program.Lambdas {
		for _, outcome := range lambda.AllowedOutcomes {
			mapped, err := LambdaOutcomeID(outcome)
			if err != nil {
				return ProgramVocabulary{}, err
			}
			outcomes[mapped] = struct{}{}
		}
		for _, effect := range lambda.AllowedEffects {
			mapped, err := LambdaEffectID(effect)
			if err != nil {
				return ProgramVocabulary{}, err
			}
			effects[mapped] = struct{}{}
		}
	}

	return ProgramVocabulary{
		Schemas:      sortedSchemaIDs(schemas),
		Capabilities: sortedCapabilityIDs(capabilities),
		Outcomes:     sortedOutcomeSet(outcomes),
		Effects:      sortedEffectSet(effects),
	}, nil
}

// LambdaOutcomeID gives a Goja outcome's versioned assurance identity.
func LambdaOutcomeID(outcome idpprogram.OutcomeKind) (OutcomeID, error) {
	switch outcome {
	case idpprogram.OutcomeContinue:
		return LambdaOutcomeContinue, nil
	case idpprogram.OutcomePresent:
		return LambdaOutcomePresent, nil
	case idpprogram.OutcomeChallenge:
		return LambdaOutcomeChallenge, nil
	case idpprogram.OutcomeCommit:
		return LambdaOutcomeCommit, nil
	case idpprogram.OutcomeComplete:
		return LambdaOutcomeComplete, nil
	case idpprogram.OutcomeDeny:
		return LambdaOutcomeDeny, nil
	case idpprogram.OutcomeSkip:
		return LambdaOutcomeSkip, nil
	case idpprogram.OutcomeError:
		return LambdaOutcomeError, nil
	default:
		return "", fmt.Errorf("unknown lambda outcome %q", outcome)
	}
}

// LambdaEffectID gives a declared Goja effect's versioned native assurance
// identity. It does not authorize the effect: native effect validation and
// commit remain the only authorities that can act on this declaration.
func LambdaEffectID(effect idpprogram.EffectKind) (EffectID, error) {
	switch effect {
	case idpprogram.EffectRead:
		return EffectReadResource, nil
	case idpprogram.EffectCreateLocalIdentity:
		return EffectCreateLocalIdentity, nil
	case idpprogram.EffectAttachPasswordCredential:
		return EffectAttachPasswordCredential, nil
	case idpprogram.EffectConsumeInvitation:
		return EffectConsumeInvitation, nil
	case idpprogram.EffectEstablishBrowserSession:
		return EffectEstablishBrowserSession, nil
	case idpprogram.EffectEstablishVirtualIdentity:
		return EffectEstablishVirtualIdentity, nil
	case idpprogram.EffectSendEmailChallenge:
		return EffectSendEmailChallenge, nil
	default:
		return "", fmt.Errorf("unknown lambda effect %q", effect)
	}
}

// DiagnosticIDs projects deterministic compiler diagnostics into their typed
// assurance form. The compiler owns their spelling; this function makes a
// malformed or user-text diagnostic impossible to use as analysis metadata.
func DiagnosticIDs(diagnostics idpprogram.Diagnostics) ([]DiagnosticID, error) {
	result := make([]DiagnosticID, 0, len(diagnostics))
	seen := map[DiagnosticID]struct{}{}
	for _, diagnostic := range diagnostics {
		id := DiagnosticID(diagnostic.ID)
		if !ValidStableID(diagnostic.ID) {
			return nil, fmt.Errorf("invalid program diagnostic ID %q", diagnostic.ID)
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

// AuthorizationEvidenceIDs validates and projects a policy decision's bounded
// native evidence names. Evidence is metadata only; no credential, claim, or
// proof value crosses this boundary.
func AuthorizationEvidenceIDs(decision idp.AuthorizationDecision) ([]EvidenceID, error) {
	normalized, err := idp.NormalizeAuthorizationDecision(decision)
	if err != nil {
		return nil, err
	}
	result := make([]EvidenceID, 0, len(normalized.Evidence))
	for _, evidence := range normalized.Evidence {
		if !ValidStableID(evidence.ID) {
			return nil, fmt.Errorf("invalid authorization evidence ID %q", evidence.ID)
		}
		result = append(result, EvidenceID(evidence.ID))
	}
	return result, nil
}

func sortedSchemaIDs(values []SchemaID) []SchemaID {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func sortedCapabilityIDs(values []CapabilityID) []CapabilityID {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}

func sortedOutcomeSet(values map[OutcomeID]struct{}) []OutcomeID {
	result := make([]OutcomeID, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func sortedEffectSet(values map[EffectID]struct{}) []EffectID {
	result := make([]EffectID, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
