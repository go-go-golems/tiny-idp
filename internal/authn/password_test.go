package authn_test

import (
	"context"
	"errors"
	"sync"
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
	cred, err := svc.HashCredential(ctx, "u1", "alice", []byte("alice-password-long"), now)
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
	if _, err := svc.AuthenticatePassword(ctx, "alice", "alice-password-long", idp.LoginMetadata{ClientID: "spa"}); !errors.Is(err, authn.ErrAccountLocked) {
		t.Fatalf("locked err=%v", err)
	}
	later := now.Add(2 * time.Minute)
	svc, err = authn.NewPasswordService(st, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams()), Clock: func() time.Time { return later }, Audit: sink})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.AuthenticatePassword(ctx, "alice", "alice-password-long", idp.LoginMetadata{ClientID: "spa"})
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

type failingSecurityStore struct {
	idpstore.Store
	failFailure bool
	failSuccess bool
}

func (s *failingSecurityStore) RecordFailedLogin(ctx context.Context, userID string, now time.Time, policy idpstore.LockoutPolicy) (idpstore.AccountSecurityState, error) {
	if s.failFailure {
		return idpstore.AccountSecurityState{}, errors.New("injected failed-login storage error")
	}
	return s.Store.RecordFailedLogin(ctx, userID, now, policy)
}

func (s *failingSecurityStore) RecordSuccessfulLogin(ctx context.Context, userID string, now time.Time, session *idpstore.Session) error {
	if s.failSuccess {
		return errors.New("injected success-reset storage error")
	}
	return s.Store.RecordSuccessfulLogin(ctx, userID, now, session)
}

func TestPasswordServiceStorageFailuresFailClosed(t *testing.T) {
	ctx := context.Background()
	base := memory.New()
	store := &failingSecurityStore{Store: base}
	now := time.Now().UTC()
	if err := base.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "subject-1"}); err != nil {
		t.Fatal(err)
	}
	service, err := authn.NewPasswordService(store, authn.Options{Hasher: passwordhash.New(passwordhash.TestParams())})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := service.HashCredential(ctx, "u1", "alice", []byte("a valid password phrase"), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.PutPasswordCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	store.failFailure = true
	if _, err := service.AuthenticatePassword(ctx, "alice", "wrong password phrase", idp.LoginMetadata{}); !errors.Is(err, authn.ErrAuthenticationUnavailable) {
		t.Fatalf("failed-login storage error = %v", err)
	}
	store.failFailure = false
	store.failSuccess = true
	if _, err := service.AuthenticatePassword(ctx, "alice", "a valid password phrase", idp.LoginMetadata{}); !errors.Is(err, authn.ErrAuthenticationUnavailable) {
		t.Fatalf("success-reset storage error = %v", err)
	}
}

func TestPasswordWorkIsBoundedAndObservableAtProductionParams(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Now().UTC()
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "subject-1"}); err != nil {
		t.Fatal(err)
	}
	service, err := authn.NewPasswordService(store, authn.Options{
		Hasher: passwordhash.New(passwordhash.DefaultParams()),
		Work:   idp.PasswordWorkConfig{MaxConcurrent: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	credential, err := service.HashCredential(ctx, "u1", "alice", []byte("a valid password phrase"), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutPasswordCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	const attempts = 6
	start := make(chan struct{})
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := service.AuthenticatePassword(ctx, "alice", "wrong password phrase", idp.LoginMetadata{})
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if !errors.Is(err, authn.ErrInvalidCredentials) {
			t.Fatalf("authentication error = %v", err)
		}
	}
	stats := service.PasswordWorkStats()
	if stats.Capacity != 1 || stats.InFlight != 0 || stats.Saturations == 0 || stats.Completed < attempts+1 || stats.TotalDuration == 0 {
		t.Fatalf("password work stats = %#v", stats)
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
	cred, err := svc.HashCredential(ctx, "u1", "alice", []byte("right-password-long"), now)
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
	if _, err := svc.AuthenticatePassword(ctx, "alice", "right-password-long", idp.LoginMetadata{}); !errors.Is(err, authn.ErrAccountLocked) {
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
