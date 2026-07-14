package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistrationAttemptConsumesOnce(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if err := store.createRegistrationAttempt(ctx, "registration-token", registrationAttempt{
		CSRFSecret: make([]byte, sha256.Size), CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.consumeRegistrationAttempt(ctx, "wrong", now); !errors.Is(err, errRegistrationAttemptUnavailable) {
		t.Fatalf("wrong token error = %v", err)
	}
	if _, err := store.consumeRegistrationAttempt(ctx, "registration-token", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.consumeRegistrationAttempt(ctx, "registration-token", now.Add(2*time.Second)); !errors.Is(err, errRegistrationAttemptUnavailable) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestRegistrationAttemptConcurrentConsumeHasOneWinner(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if err := store.createRegistrationAttempt(ctx, "contended", registrationAttempt{
		CSRFSecret: make([]byte, sha256.Size), CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var unexpected atomic.Value
	var group sync.WaitGroup
	for range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := store.consumeRegistrationAttempt(ctx, "contended", now.Add(time.Second))
			switch {
			case err == nil:
				winners.Add(1)
			case errors.Is(err, errRegistrationAttemptUnavailable):
			default:
				unexpected.Store(err)
			}
		}()
	}
	group.Wait()
	if value := unexpected.Load(); value != nil {
		t.Fatalf("unexpected concurrent error: %v", value)
	}
	if winners.Load() != 1 {
		t.Fatalf("consume winners = %d, want 1", winners.Load())
	}
}
