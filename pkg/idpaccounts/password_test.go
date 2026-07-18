package idpaccounts

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/passwordhash"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func testService(t *testing.T, store idpstore.Store, opts Options) *Service {
	t.Helper()
	service, err := newService(store, opts, passwordhash.New(passwordhash.TestParams()))
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func seedCredential(t *testing.T, service *Service, store idpstore.Store, login, userID, password string, now time.Time) {
	t.Helper()
	credential, err := service.hashCredential(context.Background(), userID, login, []byte(password), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutPasswordCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
}

func TestServiceAuthenticatesAndResetsFailures(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 7, 8, 1, 0, 0, 0, time.UTC)
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	service := testService(t, store, Options{Clock: func() time.Time { return now }})
	seedCredential(t, service, store, "alice", "u1", "alice-password-long", now)
	lockedUntil := now.Add(time.Minute)
	if err := store.PutAccountSecurityState(ctx, idpstore.AccountSecurityState{UserID: "u1", FailedLoginCount: 2, LockedUntil: &lockedUntil}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticatePassword(ctx, "alice", "alice-password-long", idp.LoginMetadata{}); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("locked error = %v", err)
	}
	later := now.Add(2 * time.Minute)
	service = testService(t, store, Options{Clock: func() time.Time { return later }})
	result, err := service.AuthenticatePassword(ctx, "alice", "alice-password-long", idp.LoginMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.Sub != "user-alice" || len(result.AMR) != 1 || result.AMR[0] != "pwd" {
		t.Fatalf("authentication result = %#v", result)
	}
	state, err := store.GetAccountSecurityState(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if state.FailedLoginCount != 0 || state.LockedUntil != nil || state.LastSuccessfulLoginAt == nil {
		t.Fatalf("security state = %#v", state)
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
		return errors.New("injected successful-login storage error")
	}
	return s.Store.RecordSuccessfulLogin(ctx, userID, now, session)
}

func TestServiceFailsClosedOnSecurityStateStorageErrors(t *testing.T) {
	ctx := context.Background()
	base := memory.New()
	store := &failingSecurityStore{Store: base}
	now := time.Now().UTC()
	if err := base.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "subject-1"}); err != nil {
		t.Fatal(err)
	}
	service := testService(t, store, Options{})
	seedCredential(t, service, base, "alice", "u1", "a valid password phrase", now)
	store.failFailure = true
	if _, err := service.AuthenticatePassword(ctx, "alice", "wrong password phrase", idp.LoginMetadata{}); !errors.Is(err, ErrAuthenticationUnavailable) {
		t.Fatalf("failed-login storage error = %v", err)
	}
	store.failFailure = false
	store.failSuccess = true
	if _, err := service.AuthenticatePassword(ctx, "alice", "a valid password phrase", idp.LoginMetadata{}); !errors.Is(err, ErrAuthenticationUnavailable) {
		t.Fatalf("successful-login storage error = %v", err)
	}
}

func TestPasswordWorkIsBoundedAndObservable(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Now().UTC()
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "subject-1"}); err != nil {
		t.Fatal(err)
	}
	service := testService(t, store, Options{PasswordWork: idp.PasswordWorkConfig{MaxConcurrent: 1}})
	seedCredential(t, service, store, "alice", "u1", "a valid password phrase", now)
	const attempts = 6
	start := make(chan struct{})
	errs := make(chan error, attempts)
	var group sync.WaitGroup
	for range attempts {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := service.AuthenticatePassword(ctx, "alice", "wrong password phrase", idp.LoginMetadata{})
			errs <- err
		}()
	}
	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("authentication error = %v", err)
		}
	}
	stats := service.PasswordWorkStats()
	if stats.Capacity != 1 || stats.InFlight != 0 || stats.Saturations == 0 || stats.Completed < attempts+1 {
		t.Fatalf("password work stats = %#v", stats)
	}
}

func TestServiceLocksAndPasswordlessRequiresExplicitPolicy(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Now().UTC()
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	policy := DefaultLoginPolicy()
	policy.LockoutThreshold = 2
	service := testService(t, store, Options{LoginPolicy: policy, Clock: func() time.Time { return now }})
	seedCredential(t, service, store, "alice", "u1", "right-password-long", now)
	for range 2 {
		if _, err := service.AuthenticatePassword(ctx, "alice", "wrong", idp.LoginMetadata{}); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("wrong-password error = %v", err)
		}
	}
	if _, err := service.AuthenticatePassword(ctx, "alice", "right-password-long", idp.LoginMetadata{}); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("locked error = %v", err)
	}

	passwordlessStore := memory.New()
	if err := passwordlessStore.PutUser(ctx, "bob", idpstore.User{ID: "u2", Sub: "user-bob"}); err != nil {
		t.Fatal(err)
	}
	service = testService(t, passwordlessStore, Options{})
	if _, err := service.AuthenticatePassword(ctx, "bob", "anything", idp.LoginMetadata{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("default passwordless error = %v", err)
	}
	passwordless := DefaultLoginPolicy()
	passwordless.AllowPasswordless = true
	service = testService(t, passwordlessStore, Options{LoginPolicy: passwordless})
	if _, err := service.AuthenticatePassword(ctx, "bob", "anything", idp.LoginMetadata{}); err != nil {
		t.Fatal(err)
	}
}
