package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
)

// RunStoreSuite verifies invariants every store implementation must satisfy.
func RunStoreSuite(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()
	t.Run("authorization code can be consumed once", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-1")
		if err := st.CreateAuthorizationCode(ctx, domain.AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); err != nil {
			t.Fatalf("consume code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); !errors.Is(err, ErrAlreadyConsumed) {
			t.Fatalf("second consume got %v, want %v", err, ErrAlreadyConsumed)
		}
	})

	t.Run("parallel authorization code consumption has one winner", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-race")
		if err := st.CreateAuthorizationCode(ctx, domain.AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		success := 0
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); err == nil {
					mu.Lock()
					success++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if success != 1 {
			t.Fatalf("success count = %d, want 1", success)
		}
	})

	t.Run("expired authorization code is rejected", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-expired")
		if err := st.CreateAuthorizationCode(ctx, domain.AuthorizationCode{CodeHash: codeHash, ExpiresAt: now.Add(-time.Minute)}); err != nil {
			t.Fatalf("create code: %v", err)
		}
		if _, err := st.ConsumeAuthorizationCode(ctx, codeHash, now); !errors.Is(err, ErrExpired) {
			t.Fatalf("got %v, want expired", err)
		}
	})

	t.Run("refresh token rotation and reuse detection", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		oldHash := []byte("refresh-old")
		newHash := []byte("refresh-new")
		if err := st.CreateRefreshToken(ctx, domain.RefreshToken{TokenHash: oldHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatalf("create refresh: %v", err)
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, domain.RefreshToken{TokenHash: newHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}, now); err != nil {
			t.Fatalf("rotate refresh: %v", err)
		}
		old, err := st.GetRefreshToken(ctx, oldHash)
		if err != nil {
			t.Fatalf("get old: %v", err)
		}
		if string(old.ReplacedByHash) != string(newHash) {
			t.Fatalf("old token not linked to replacement")
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, domain.RefreshToken{TokenHash: []byte("other"), GrantID: "g"}, now); !errors.Is(err, ErrRefreshReuseDetected) {
			t.Fatalf("reuse got %v, want reuse detected", err)
		}
		newToken, err := st.GetRefreshToken(ctx, newHash)
		if err != nil {
			t.Fatalf("get new: %v", err)
		}
		if newToken.RevokedAt == nil {
			t.Fatalf("reuse should revoke token family")
		}
	})

	t.Run("active signing key and verification keys", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		if err := st.CreateSigningKey(ctx, domain.SigningKey{ID: "k1", Algorithm: "RS256"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateSigningKey(ctx, domain.SigningKey{ID: "k2", Algorithm: "RS256"}); err != nil {
			t.Fatal(err)
		}
		if err := st.ActivateSigningKey(ctx, "k2"); err != nil {
			t.Fatal(err)
		}
		active, err := st.ActiveSigningKey(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if active.ID != "k2" {
			t.Fatalf("active = %s", active.ID)
		}
		keys, err := st.VerificationKeys(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 1 || keys[0].ID != "k2" {
			t.Fatalf("verification keys = %#v", keys)
		}
	})
}
