package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginAttemptConsumesOnce(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	desired := loginAttempt{
		Nonce: "nonce", PKCEVerifier: "verifier", ReturnTo: "/messages",
		CreatedAt: now, ExpiresAt: now.Add(5 * time.Minute),
	}
	if err := store.createLoginAttempt(ctx, "raw-state", desired); err != nil {
		t.Fatal(err)
	}
	consumed, err := store.consumeLoginAttempt(ctx, "raw-state", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if consumed.Nonce != desired.Nonce || consumed.PKCEVerifier != desired.PKCEVerifier || consumed.ReturnTo != desired.ReturnTo || consumed.ConsumedAt.IsZero() {
		t.Fatalf("unexpected consumed attempt: %#v", consumed)
	}
	if _, err := store.consumeLoginAttempt(ctx, "raw-state", now.Add(2*time.Second)); !errors.Is(err, errLoginAttemptUnavailable) {
		t.Fatalf("replay error = %v", err)
	}
}

func TestLoginAttemptRejectsWrongAndExpiredState(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if err := store.createLoginAttempt(ctx, "state", loginAttempt{
		Nonce: "nonce", PKCEVerifier: "verifier", ReturnTo: "/", CreatedAt: now, ExpiresAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		state string
		now   time.Time
	}{{"wrong", now}, {"state", now.Add(time.Second)}} {
		if _, err := store.consumeLoginAttempt(ctx, test.state, test.now); !errors.Is(err, errLoginAttemptUnavailable) {
			t.Errorf("consume(%q, %s) error = %v", test.state, test.now, err)
		}
	}
}

func TestLoginAttemptConcurrentConsumeHasOneWinner(t *testing.T) {
	ctx := context.Background()
	store, err := openAppStore(ctx, filepath.Join(t.TempDir(), "messages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	if err := store.createLoginAttempt(ctx, "contended-state", loginAttempt{
		Nonce: "nonce", PKCEVerifier: "verifier", ReturnTo: "/", CreatedAt: now, ExpiresAt: now.Add(time.Minute),
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
			_, err := store.consumeLoginAttempt(ctx, "contended-state", now.Add(time.Second))
			switch {
			case err == nil:
				winners.Add(1)
			case errors.Is(err, errLoginAttemptUnavailable):
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
