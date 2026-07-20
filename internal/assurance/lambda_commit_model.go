package assurance

import "fmt"

// DeclaredLambdaModel is a small pure safety model for one native workflow
// invocation history. Lambda outcomes are deliberately nondeterministic: any
// member of DeclaredOutcomes may occur. Only the native boundaries can turn a
// declared commit request into identity/session/artifact-relevant completion.
type DeclaredLambdaModel struct {
	declared         map[OutcomeID]struct{}
	evidenceRequired bool
	evidenceVerified bool
	commitRequested  bool
	effectsValidated bool
	nativeCommitted  bool
}

// NewDeclaredLambdaModel creates a model with the exact compiled lambda
// outcome vocabulary. Empty, invalid, or duplicate values fail closed rather
// than widening the outcome set with a default.
func NewDeclaredLambdaModel(declaredOutcomes []OutcomeID, evidenceRequired bool) (*DeclaredLambdaModel, error) {
	if len(declaredOutcomes) == 0 {
		return nil, fmt.Errorf("declared lambda outcomes are required")
	}
	declared := make(map[OutcomeID]struct{}, len(declaredOutcomes))
	for _, outcome := range declaredOutcomes {
		if !ValidStableID(string(outcome)) {
			return nil, fmt.Errorf("invalid declared lambda outcome %q", outcome)
		}
		if _, duplicate := declared[outcome]; duplicate {
			return nil, fmt.Errorf("duplicate declared lambda outcome %q", outcome)
		}
		declared[outcome] = struct{}{}
	}
	return &DeclaredLambdaModel{declared: declared, evidenceRequired: evidenceRequired}, nil
}

// Apply consumes a normalized observed native transition. It returns all
// violations attributable to that observation while preserving the model state
// for counterexample inspection. It neither executes a lambda nor commits a
// store transaction.
func (m *DeclaredLambdaModel) Apply(observation TraceObservation) []error {
	if m == nil {
		return []error{fmt.Errorf("declared lambda model is nil")}
	}
	switch observation.Kind {
	case ObservationLambdaStarted, ObservationLambdaRejected, ObservationContinuationCreated:
		return nil
	case ObservationLambdaCompleted:
		if observation.Step != StepLambdaInvoke {
			return []error{fmt.Errorf("lambda completion used step %q", observation.Step)}
		}
		if _, ok := m.declared[observation.Outcome]; !ok {
			return []error{fmt.Errorf("lambda completed with undeclared outcome %q", observation.Outcome)}
		}
		if observation.Outcome == LambdaOutcomeCommit {
			m.commitRequested = true
		}
		return nil
	case ObservationEvidenceVerified:
		if observation.Step != StepEvidenceVerify {
			return []error{fmt.Errorf("evidence observation used step %q", observation.Step)}
		}
		if observation.Outcome == TransitionApplied {
			m.evidenceVerified = true
			return nil
		}
		if observation.Outcome == TransitionRejected {
			return nil
		}
		return []error{fmt.Errorf("invalid evidence outcome %q", observation.Outcome)}
	case ObservationNativeEffectCommitted:
		if observation.Step == StepEffectValidate {
			if observation.Outcome == TransitionRejected {
				return nil
			}
			if observation.Outcome != TransitionApplied {
				return []error{fmt.Errorf("invalid effect-validation outcome %q", observation.Outcome)}
			}
			if !m.commitRequested {
				return []error{fmt.Errorf("native effect validation occurred without declared lambda commit")}
			}
			m.effectsValidated = true
			return nil
		}
		if observation.Step == StepSignupCommit {
			if observation.Outcome == TransitionRejected {
				return nil
			}
			if observation.Outcome != TransitionApplied {
				return []error{fmt.Errorf("invalid native commit outcome %q", observation.Outcome)}
			}
			violations := m.commitViolations()
			if len(violations) != 0 {
				return violations
			}
			if m.nativeCommitted {
				return []error{fmt.Errorf("native signup commit occurred more than once")}
			}
			m.nativeCommitted = true
			return nil
		}
		return []error{fmt.Errorf("native effect observation used step %q", observation.Step)}
	case ObservationContinuationTerminal:
		if observation.Step != StepContinuationConsume {
			return []error{fmt.Errorf("continuation terminal used step %q", observation.Step)}
		}
		if observation.Outcome == TransitionApplied && m.commitRequested && !m.nativeCommitted {
			return []error{fmt.Errorf("continuation completed before native signup commit")}
		}
		return nil
	case ObservationInteractionTerminal:
		if observation.Outcome == TransitionApproved && m.commitRequested && !m.nativeCommitted {
			return []error{fmt.Errorf("approved interaction completed before native signup commit")}
		}
		return nil
	case ObservationAuthorizationArtifacts, ObservationTokenLifecycle:
		if m.commitRequested && !m.nativeCommitted {
			return []error{fmt.Errorf("artifact or token lifecycle completed before native signup commit")}
		}
		return nil
	case ObservationInteractionCreated, ObservationAuthenticationSatisfied, ObservationConsentApproved, ObservationConsentDenied:
		return nil
	default:
		return []error{fmt.Errorf("unknown lambda-model observation %q", observation.Kind)}
	}
}

func (m *DeclaredLambdaModel) commitViolations() []error {
	violations := []error(nil)
	if !m.commitRequested {
		violations = append(violations, fmt.Errorf("native signup commit occurred without declared lambda commit"))
	}
	if m.evidenceRequired && !m.evidenceVerified {
		violations = append(violations, fmt.Errorf("native signup commit lacks required verified evidence"))
	}
	if !m.effectsValidated {
		violations = append(violations, fmt.Errorf("native signup commit lacks validated effect plan"))
	}
	return violations
}
