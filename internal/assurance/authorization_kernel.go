package assurance

import (
	"fmt"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

// AuthorizationKernel is the deliberately small pure transition kernel for
// one authorization interaction's terminal ordering invariants. It has no
// HTTP, Fosite, database, clock, Goja, goroutine, or secret dependency.
type AuthorizationKernel struct {
	required map[ObligationID]struct{}
	authed   bool
	consent  bool
	terminal OutcomeID
	artifact bool
}

// NewAuthorizationKernel decodes the persisted required-action bitset through
// the fail-closed obligation codec before creating the pure model state.
func NewAuthorizationKernel(actions uint32) (*AuthorizationKernel, error) {
	obligations, err := ObligationsFromActions(idpstore.InteractionRequiredAction(actions))
	if err != nil {
		return nil, err
	}
	required := make(map[ObligationID]struct{}, len(obligations))
	for _, obligation := range obligations {
		required[obligation] = struct{}{}
	}
	return &AuthorizationKernel{required: required}, nil
}

// Apply consumes one already-validated assurance observation. It returns every
// invariant violation caused by that observation rather than choosing one.
// State transitions remain deterministic, allowing a monitor to report the
// complete diagnostic set without inventing separate transition semantics.
func (k *AuthorizationKernel) Apply(observation ObservationID, outcome OutcomeID) []error {
	if k == nil {
		return []error{fmt.Errorf("authorization kernel is nil")}
	}
	switch observation {
	case ObservationAuthenticationSatisfied:
		k.authed = true
	case ObservationConsentApproved:
		k.consent = true
	case ObservationConsentDenied:
		k.consent = false
	case ObservationInteractionTerminal:
		if k.terminal != "" {
			return []error{fmt.Errorf("interaction has multiple terminal outcomes (%s then %s)", k.terminal, outcome)}
		}
		violations := []error(nil)
		if outcome == TransitionApproved {
			if k.requires(ObligationLogin) || k.requires(ObligationFreshLogin) {
				if !k.authed {
					violations = append(violations, fmt.Errorf("approved terminal outcome lacks required authentication"))
				}
			}
			if k.requires(ObligationConsent) && !k.consent {
				violations = append(violations, fmt.Errorf("approved terminal outcome lacks required consent"))
			}
		}
		k.terminal = outcome
		return violations
	case ObservationAuthorizationArtifacts:
		if k.terminal != TransitionApproved {
			return []error{fmt.Errorf("authorization artifacts committed before approved terminal outcome")}
		}
		if k.artifact {
			return []error{fmt.Errorf("authorization artifacts committed more than once")}
		}
		k.artifact = true
	case ObservationInteractionCreated,
		ObservationTokenLifecycle,
		ObservationLambdaStarted,
		ObservationLambdaCompleted,
		ObservationLambdaRejected,
		ObservationContinuationCreated,
		ObservationContinuationTerminal,
		ObservationEvidenceVerified,
		ObservationNativeEffectCommitted:
		return []error{fmt.Errorf("observation %q is outside authorization kernel", observation)}
	}
	return nil
}

func (k *AuthorizationKernel) requires(obligation ObligationID) bool {
	_, ok := k.required[obligation]
	return ok
}
