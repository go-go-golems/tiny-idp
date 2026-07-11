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
