package authn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func TestPasswordServiceAuthenticatesAndResetsFailures(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	now := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	sink := idp.NewMemorySink()
	svc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Clock: func() time.Time { return now }, Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	cred, err := svc.HashCredential("u1", "alice", []byte("alice-password"), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutPasswordCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}
	lockedUntil := now.Add(time.Minute)
	if err := st.PutAccountSecurityState(ctx, idpstore.AccountSecurityState{UserID: "u1", FailedLoginCount: 2, LockedUntil: &lockedUntil}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticatePassword(ctx, "alice", "alice-password", idp.LoginMetadata{ClientID: "spa"}); !errors.Is(err, authn.ErrAccountLocked) {
		t.Fatalf("locked err=%v", err)
	}
	later := now.Add(2 * time.Minute)
	svc, err = authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Clock: func() time.Time { return later }, Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.AuthenticatePassword(ctx, "alice", "alice-password", idp.LoginMetadata{ClientID: "spa"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Sub != "user-alice" || len(result.AMR) != 1 || result.AMR[0] != "pwd" {
		t.Fatalf("bad auth result: %#v", result)
	}
	state, err := st.GetAccountSecurityState(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if state.FailedLoginCount != 0 || state.LockedUntil != nil || state.LastSuccessfulLoginAt == nil {
		t.Fatalf("security state not reset: %#v", state)
	}
}

func TestPasswordServiceRejectsWrongPasswordAndLocks(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	now := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	policy := authn.DefaultPasswordPolicy()
	policy.LockoutThreshold = 2
	policy.LockoutDuration = time.Minute
	svc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Policy: policy, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	cred, err := svc.HashCredential("u1", "alice", []byte("right-password"), now)
	if err != nil {
		t.Fatal(err)
	}
	_ = st.PutPasswordCredential(ctx, cred)
	for i := 0; i < 2; i++ {
		if _, err := svc.AuthenticatePassword(ctx, "alice", "wrong", idp.LoginMetadata{}); !errors.Is(err, authn.ErrInvalidCredentials) {
			t.Fatalf("attempt %d err=%v", i, err)
		}
	}
	state, err := st.GetAccountSecurityState(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if state.LockedUntil == nil {
		t.Fatalf("expected lockout: %#v", state)
	}
	if _, err := svc.AuthenticatePassword(ctx, "alice", "right-password", idp.LoginMetadata{}); !errors.Is(err, authn.ErrAccountLocked) {
		t.Fatalf("locked err=%v", err)
	}
}

func TestPasswordServiceAllowsPasswordlessOnlyWhenPolicyAllows(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"})
	svc, err := authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AuthenticatePassword(ctx, "alice", "anything", idp.LoginMetadata{}); !errors.Is(err, authn.ErrInvalidCredentials) {
		t.Fatalf("err=%v, want invalid credentials", err)
	}
	policy := authn.DefaultPasswordPolicy()
	policy.AllowPasswordless = true
	svc, err = authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.AuthenticatePassword(ctx, "alice", "anything", idp.LoginMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Sub != "user-alice" {
		t.Fatalf("bad result: %#v", result)
	}
}
