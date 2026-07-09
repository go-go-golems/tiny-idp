package idpstore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// RunStoreSuite verifies invariants every store implementation must satisfy.
func RunStoreSuite(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()
	t.Run("authorization code can be consumed once", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		codeHash := []byte("code-1")
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
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
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ClientID: "c", ExpiresAt: now.Add(time.Minute)}); err != nil {
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
		if err := st.CreateAuthorizationCode(ctx, AuthorizationCode{CodeHash: codeHash, ExpiresAt: now.Add(-time.Minute)}); err != nil {
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
		if err := st.CreateRefreshToken(ctx, RefreshToken{TokenHash: oldHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatalf("create refresh: %v", err)
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, RefreshToken{TokenHash: newHash, GrantID: "g", ClientID: "c", UserID: "u", ExpiresAt: now.Add(time.Hour)}, now); err != nil {
			t.Fatalf("rotate refresh: %v", err)
		}
		old, err := st.GetRefreshToken(ctx, oldHash)
		if err != nil {
			t.Fatalf("get old: %v", err)
		}
		if string(old.ReplacedByHash) != string(newHash) {
			t.Fatalf("old token not linked to replacement")
		}
		if _, err := st.RotateRefreshToken(ctx, oldHash, RefreshToken{TokenHash: []byte("other"), GrantID: "g"}, now); !errors.Is(err, ErrRefreshReuseDetected) {
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

	t.Run("consent is normalized and revocable", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now()
		consent := Consent{UserID: "u", ClientID: "c", Scope: []string{"email", "openid", "email"}, GrantedAt: now}
		if err := st.PutConsent(ctx, consent); err != nil {
			t.Fatalf("put consent: %v", err)
		}
		got, err := st.GetConsent(ctx, "u", "c", []string{"openid", "email"})
		if err != nil {
			t.Fatalf("get consent: %v", err)
		}
		if len(got.Scope) != 2 || got.Scope[0] != "email" || got.Scope[1] != "openid" {
			t.Fatalf("scope not normalized: %#v", got.Scope)
		}
		if err := st.RevokeConsent(ctx, "u", "c", []string{"email", "openid"}, now); err != nil {
			t.Fatalf("revoke consent: %v", err)
		}
		revoked, err := st.GetConsent(ctx, "u", "c", []string{"openid", "email"})
		if err != nil {
			t.Fatalf("get revoked consent: %v", err)
		}
		if revoked.RevokedAt == nil {
			t.Fatalf("consent was not revoked")
		}
	})

	t.Run("password credentials and account security state", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		now := time.Now().UTC()
		credential := PasswordCredential{UserID: "u1", Login: "alice", PasswordHash: []byte("encoded-hash"), HashAlgorithm: "argon2id-v1", CreatedAt: now, UpdatedAt: now, PasswordChangedAt: now}
		if err := st.PutPasswordCredential(ctx, credential); err != nil {
			t.Fatalf("put credential: %v", err)
		}
		byLogin, err := st.GetPasswordCredentialByLogin(ctx, "alice")
		if err != nil {
			t.Fatalf("get by login: %v", err)
		}
		if byLogin.UserID != "u1" || string(byLogin.PasswordHash) != "encoded-hash" {
			t.Fatalf("bad credential by login: %#v", byLogin)
		}
		byUser, err := st.GetPasswordCredentialByUserID(ctx, "u1")
		if err != nil {
			t.Fatalf("get by user: %v", err)
		}
		if byUser.Login != "alice" {
			t.Fatalf("bad credential by user: %#v", byUser)
		}
		if err := st.PutPasswordCredential(ctx, PasswordCredential{UserID: "u2", Login: "alice", PasswordHash: []byte("other")}); !errors.Is(err, ErrDuplicate) {
			t.Fatalf("duplicate login got %v, want %v", err, ErrDuplicate)
		}
		lockedUntil := now.Add(time.Minute)
		state := AccountSecurityState{UserID: "u1", FailedLoginCount: 2, LockedUntil: &lockedUntil}
		if err := st.PutAccountSecurityState(ctx, state); err != nil {
			t.Fatalf("put security state: %v", err)
		}
		gotState, err := st.GetAccountSecurityState(ctx, "u1")
		if err != nil {
			t.Fatalf("get security state: %v", err)
		}
		if gotState.FailedLoginCount != 2 || gotState.LockedUntil == nil {
			t.Fatalf("bad security state: %#v", gotState)
		}
		if err := st.ResetAccountSecurityState(ctx, "u1", now); err != nil {
			t.Fatalf("reset security state: %v", err)
		}
		reset, err := st.GetAccountSecurityState(ctx, "u1")
		if err != nil {
			t.Fatalf("get reset state: %v", err)
		}
		if reset.FailedLoginCount != 0 || reset.LockedUntil != nil || reset.LastSuccessfulLoginAt == nil {
			t.Fatalf("bad reset state: %#v", reset)
		}
		if err := st.DeletePasswordCredential(ctx, "u1"); err != nil {
			t.Fatalf("delete credential: %v", err)
		}
		if _, err := st.GetPasswordCredentialByUserID(ctx, "u1"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("deleted credential got %v, want not found", err)
		}
	})

	t.Run("active signing key and verification keys", func(t *testing.T) {
		ctx := context.Background()
		st := newStore(t)
		if err := st.CreateSigningKey(ctx, SigningKey{ID: "k1", Algorithm: "RS256"}); err != nil {
			t.Fatal(err)
		}
		if err := st.CreateSigningKey(ctx, SigningKey{ID: "k2", Algorithm: "RS256"}); err != nil {
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
