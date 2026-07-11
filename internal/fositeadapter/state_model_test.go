package fositeadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/store/memory"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"pgregory.net/rapid"
)

type interactionReferenceState struct {
	created  bool
	consumed bool
	expired  bool
}

type interactionModelAction uint8

const (
	modelCreate interactionModelAction = iota
	modelGet
	modelApprove
	modelDeny
	modelAdvancePastExpiry
	modelMutateReturnedCopy
)

type interactionModelObservation struct {
	Action   interactionModelAction
	Accepted bool
	Reason   string
}

func (s *interactionReferenceState) Apply(action interactionModelAction) interactionModelObservation {
	observation := interactionModelObservation{Action: action}
	switch action {
	case modelCreate:
		if s.created {
			observation.Reason = "duplicate"
			return observation
		}
		s.created = true
		observation.Accepted = true
	case modelGet, modelMutateReturnedCopy:
		observation.Accepted = s.created
		if !s.created {
			observation.Reason = "not_found"
		}
	case modelApprove, modelDeny:
		switch {
		case !s.created:
			observation.Reason = "not_found"
		case s.consumed:
			observation.Reason = "already_consumed"
		case s.expired:
			observation.Reason = "expired"
		default:
			s.consumed = true
			observation.Accepted = true
		}
	case modelAdvancePastExpiry:
		if s.created {
			s.expired = true
		}
		observation.Accepted = true
	default:
		observation.Reason = "unknown_action"
	}
	return observation
}

func TestInteractionStoreStateMachine(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		store := memory.New()
		ctx := context.Background()
		now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
		hash := []byte("rapid-interaction-hash")
		model := interactionReferenceState{}
		steps := rapid.IntRange(1, 80).Draw(t, "steps")
		for step := 0; step < steps; step++ {
			switch rapid.IntRange(0, 5).Draw(t, "operation") {
			case 0:
				err := store.CreateInteraction(ctx, idpstore.InteractionRecord{
					IDHash:           hash,
					CanonicalRequest: map[string][]string{"state": {"original"}},
					CreatedAt:        now,
					ExpiresAt:        now.Add(time.Minute),
				})
				if model.created {
					if !errors.Is(err, idpstore.ErrDuplicate) {
						t.Fatalf("duplicate create error=%v", err)
					}
				} else {
					if err != nil {
						t.Fatalf("initial create: %v", err)
					}
					model.created = true
				}
			case 1:
				_, err := store.GetInteraction(ctx, hash)
				if model.created && err != nil {
					t.Fatalf("get created interaction: %v", err)
				}
				if !model.created && !errors.Is(err, idpstore.ErrNotFound) {
					t.Fatalf("get absent interaction error=%v", err)
				}
			case 2, 3:
				outcome := idpstore.InteractionOutcomeApproved
				if rapid.Bool().Draw(t, "deny") {
					outcome = idpstore.InteractionOutcomeDenied
				}
				_, err := store.ConsumeInteraction(ctx, hash, now, outcome)
				switch {
				case !model.created:
					if !errors.Is(err, idpstore.ErrNotFound) {
						t.Fatalf("consume absent error=%v", err)
					}
				case model.consumed:
					if !errors.Is(err, idpstore.ErrAlreadyConsumed) {
						t.Fatalf("consume terminal error=%v", err)
					}
				case model.expired:
					if !errors.Is(err, idpstore.ErrExpired) {
						t.Fatalf("consume expired error=%v", err)
					}
				default:
					if err != nil {
						t.Fatalf("consume pending: %v", err)
					}
					model.consumed = true
				}
			case 4:
				now = now.Add(2 * time.Minute)
				if model.created {
					model.expired = true
				}
			case 5:
				if !model.created {
					continue
				}
				record, err := store.GetInteraction(ctx, hash)
				if err != nil {
					t.Fatal(err)
				}
				record.CanonicalRequest["state"][0] = "mutated"
				again, err := store.GetInteraction(ctx, hash)
				if err != nil {
					t.Fatal(err)
				}
				if again.CanonicalRequest["state"][0] != "original" {
					t.Fatal("read result mutated stored canonical request")
				}
			}
		}
	})
}

func TestInteractionModelReplaysShrunkRegressionSequences(t *testing.T) {
	tests := []struct {
		name              string
		actions           []interactionModelAction
		acceptedTerminals int
	}{
		{name: "sequential replay", actions: []interactionModelAction{modelCreate, modelApprove, modelApprove}, acceptedTerminals: 1},
		{name: "deny then approve", actions: []interactionModelAction{modelCreate, modelDeny, modelApprove}, acceptedTerminals: 1},
		{name: "expired interaction", actions: []interactionModelAction{modelCreate, modelAdvancePastExpiry, modelApprove}, acceptedTerminals: 0},
		{name: "consume absent", actions: []interactionModelAction{modelApprove}, acceptedTerminals: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := interactionReferenceState{}
			acceptedTerminals := 0
			for _, action := range test.actions {
				observation := state.Apply(action)
				if observation.Accepted && (action == modelApprove || action == modelDeny) {
					acceptedTerminals++
				}
			}
			if acceptedTerminals != test.acceptedTerminals {
				t.Fatalf("accepted terminals=%d, want %d", acceptedTerminals, test.acceptedTerminals)
			}
		})
	}
}

func FuzzInteractionModelActionSequences(f *testing.F) {
	// These committed seeds are minimized replays for duplicate terminal,
	// denial/approval competition, expiry, and absent-consume histories.
	f.Add([]byte{byte(modelCreate), byte(modelApprove), byte(modelApprove)})
	f.Add([]byte{byte(modelCreate), byte(modelDeny), byte(modelApprove)})
	f.Add([]byte{byte(modelCreate), byte(modelAdvancePastExpiry), byte(modelApprove)})
	f.Add([]byte{byte(modelApprove)})
	f.Fuzz(func(t *testing.T, encoded []byte) {
		if len(encoded) > 256 {
			encoded = encoded[:256]
		}
		state := interactionReferenceState{}
		acceptedTerminals := 0
		for _, value := range encoded {
			action := interactionModelAction(value % byte(modelMutateReturnedCopy+1))
			observation := state.Apply(action)
			if observation.Accepted && (action == modelApprove || action == modelDeny) {
				acceptedTerminals++
			}
			if acceptedTerminals > 1 {
				t.Fatalf("more than one accepted terminal in %v", encoded)
			}
		}
	})
}
