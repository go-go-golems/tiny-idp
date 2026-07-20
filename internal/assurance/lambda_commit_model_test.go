package assurance

import "testing"

func TestDeclaredLambdaModelAcceptsDeclaredNondeterministicPathAndNativeCommit(t *testing.T) {
	model, err := NewDeclaredLambdaModel([]OutcomeID{LambdaOutcomePresent, LambdaOutcomeChallenge, LambdaOutcomeCommit}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, observation := range []TraceObservation{
		{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomePresent},
		{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeChallenge},
		{Step: StepEvidenceVerify, Kind: ObservationEvidenceVerified, Outcome: TransitionApplied},
		{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeCommit},
		{Step: StepEffectValidate, Kind: ObservationNativeEffectCommitted, Outcome: TransitionApplied},
		{Step: StepSignupCommit, Kind: ObservationNativeEffectCommitted, Outcome: TransitionApplied},
		{Step: StepContinuationConsume, Kind: ObservationContinuationTerminal, Outcome: TransitionApplied},
		{Step: StepInteractionApprove, Kind: ObservationInteractionTerminal, Outcome: TransitionApproved},
		{Step: StepAuthorizationCommit, Kind: ObservationAuthorizationArtifacts, Outcome: TransitionApplied},
		{Step: StepTokenIssue, Kind: ObservationTokenLifecycle, Outcome: TransitionApplied},
	} {
		if violations := model.Apply(observation); len(violations) != 0 {
			t.Fatalf("observation=%#v violations=%v", observation, violations)
		}
	}
}

func TestDeclaredLambdaModelRejectsUndeclaredOutcomeAndEarlyCompletion(t *testing.T) {
	model, err := NewDeclaredLambdaModel([]OutcomeID{LambdaOutcomeCommit}, true)
	if err != nil {
		t.Fatal(err)
	}
	if violations := model.Apply(TraceObservation{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeDeny}); len(violations) == 0 {
		t.Fatal("undeclared lambda outcome accepted")
	}
	if violations := model.Apply(TraceObservation{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeCommit}); len(violations) != 0 {
		t.Fatal(violations)
	}
	if violations := model.Apply(TraceObservation{Step: StepSignupCommit, Kind: ObservationNativeEffectCommitted, Outcome: TransitionApplied}); len(violations) != 2 {
		t.Fatalf("native commit violations=%v want evidence and effect-plan failures", violations)
	}
	if violations := model.Apply(TraceObservation{Step: StepAuthorizationCommit, Kind: ObservationAuthorizationArtifacts, Outcome: TransitionApplied}); len(violations) == 0 {
		t.Fatal("artifact completion before native commit accepted")
	}
}

func FuzzDeclaredLambdaModelNeverPanics(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6})
	f.Add([]byte{2, 5, 3, 4, 6})
	f.Fuzz(func(t *testing.T, encoded []byte) {
		if len(encoded) > 256 {
			encoded = encoded[:256]
		}
		model, err := NewDeclaredLambdaModel([]OutcomeID{LambdaOutcomePresent, LambdaOutcomeChallenge, LambdaOutcomeCommit}, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range encoded {
			_ = model.Apply(fuzzLambdaObservation(value))
		}
	})
}

func fuzzLambdaObservation(value byte) TraceObservation {
	switch value % 8 {
	case 0:
		return TraceObservation{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomePresent}
	case 1:
		return TraceObservation{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeChallenge}
	case 2:
		return TraceObservation{Step: StepLambdaInvoke, Kind: ObservationLambdaCompleted, Outcome: LambdaOutcomeCommit}
	case 3:
		return TraceObservation{Step: StepEvidenceVerify, Kind: ObservationEvidenceVerified, Outcome: TransitionApplied}
	case 4:
		return TraceObservation{Step: StepEffectValidate, Kind: ObservationNativeEffectCommitted, Outcome: TransitionApplied}
	case 5:
		return TraceObservation{Step: StepSignupCommit, Kind: ObservationNativeEffectCommitted, Outcome: TransitionApplied}
	case 6:
		return TraceObservation{Step: StepContinuationConsume, Kind: ObservationContinuationTerminal, Outcome: TransitionApplied}
	default:
		return TraceObservation{Step: StepAuthorizationCommit, Kind: ObservationAuthorizationArtifacts, Outcome: TransitionApplied}
	}
}
