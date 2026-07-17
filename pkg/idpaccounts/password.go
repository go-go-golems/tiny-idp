package idpaccounts

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/passwordhash"
	"github.com/go-go-golems/tiny-idp/internal/user"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

var (
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrAccountDisabled           = errors.New("account disabled")
	ErrAccountLocked             = errors.New("account locked")
	ErrAuthenticationUnavailable = errors.New("authentication unavailable")
	ErrPasswordWorkRejected      = errors.New("password work rejected")
)

// LoginPolicy controls lockout behavior and development-only passwordless use.
type LoginPolicy struct {
	LockoutThreshold  int
	LockoutWindow     time.Duration
	LockoutDuration   time.Duration
	AllowPasswordless bool
	DummyHash         []byte
}

func DefaultLoginPolicy() LoginPolicy {
	return LoginPolicy{LockoutThreshold: 5, LockoutWindow: 15 * time.Minute, LockoutDuration: 15 * time.Minute}
}

// Options configures account lifecycle and password authentication behavior.
// Password hashing parameters deliberately remain an implementation detail.
type Options struct {
	LoginPolicy    LoginPolicy
	PasswordPolicy idp.PasswordAcceptancePolicy
	PasswordWork   idp.PasswordWorkConfig
	Clock          func() time.Time
	Audit          idp.Sink
}

// Service owns account creation, password replacement, and password authentication.
type Service struct {
	store         idpstore.Store
	hasher        passwordhash.Hasher
	policy        LoginPolicy
	clock         func() time.Time
	audit         idp.Sink
	acceptance    idp.PasswordAcceptancePolicy
	work          chan struct{}
	metrics       passwordWorkMetrics
	auditFailures atomic.Uint64
}

type passwordWorkMetrics struct {
	inFlight      atomic.Int64
	waiting       atomic.Int64
	saturations   atomic.Uint64
	rejected      atomic.Uint64
	completed     atomic.Uint64
	totalWait     atomic.Int64
	totalDuration atomic.Int64
}

var _ idp.PasswordAuthenticator = (*Service)(nil)
var _ idp.PasswordWorkReporter = (*Service)(nil)
var _ idp.ProductionReadyReporter = (*Service)(nil)

// tinyidp:development-default -- production construction validates the injected sink.
func NewService(store idpstore.Store, opts Options) (*Service, error) {
	return newService(store, opts, passwordhash.Hasher{})
}

// tinyidp:development-default -- production hosts inject the provider's durable audit sink.
func newService(store idpstore.Store, opts Options, hasher passwordhash.Hasher) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	if hasher.Params.MemoryKiB == 0 {
		hasher = passwordhash.New(passwordhash.DefaultParams())
	}
	policy := opts.LoginPolicy
	if policy.LockoutThreshold == 0 && policy.LockoutWindow == 0 && policy.LockoutDuration == 0 && len(policy.DummyHash) == 0 {
		policy = DefaultLoginPolicy()
	}
	acceptance := opts.PasswordPolicy
	if acceptance.MinCharacters == 0 {
		acceptance = idp.DefaultPasswordAcceptancePolicy()
	}
	workConfig := opts.PasswordWork
	if workConfig.MaxConcurrent == 0 {
		workConfig = idp.DefaultPasswordWorkConfig()
	}
	if workConfig.MaxConcurrent < 1 {
		return nil, fmt.Errorf("password work max concurrency must be positive")
	}
	if len(policy.DummyHash) == 0 {
		dummy, err := hasher.HashPassword([]byte("tinyidp dummy password verifier"))
		if err != nil {
			return nil, err
		}
		policy.DummyHash = dummy
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	sink := opts.Audit
	if sink == nil {
		sink = idp.NoopSink{}
	}
	return &Service{store: store, hasher: hasher, policy: policy, clock: clock, audit: sink, acceptance: acceptance, work: make(chan struct{}, workConfig.MaxConcurrent)}, nil
}

func (s *Service) AuthenticatePassword(ctx context.Context, login, password string, meta idp.LoginMetadata) (idp.AuthResult, error) {
	now := s.clock().UTC()
	normalized := NormalizeLogin(login)
	passwordBytes, normalizeErr := s.acceptance.NormalizePassword([]byte(password))
	if normalized == "" || normalizeErr != nil || (password == "" && !s.policy.AllowPasswordless) {
		if err := s.dummyVerify(ctx, passwordBytes); err != nil {
			return idp.AuthResult{}, err
		}
		s.emit(ctx, "password.login.failure", meta, "", "invalid_credentials")
		return idp.AuthResult{}, ErrInvalidCredentials
	}

	u, userErr := s.store.GetUserByLogin(ctx, normalized)
	if userErr != nil {
		if !errors.Is(userErr, idpstore.ErrNotFound) {
			s.emit(ctx, "password.login.unavailable", meta, "", "store_error")
			return idp.AuthResult{}, fmt.Errorf("%w: load user", ErrAuthenticationUnavailable)
		}
		if err := s.dummyVerify(ctx, passwordBytes); err != nil {
			return idp.AuthResult{}, err
		}
		s.emit(ctx, "password.login.failure", meta, "", "invalid_credentials")
		return idp.AuthResult{}, ErrInvalidCredentials
	}
	credential, credErr := s.store.GetPasswordCredentialByLogin(ctx, normalized)
	if credErr != nil {
		if errors.Is(credErr, idpstore.ErrNotFound) && s.policy.AllowPasswordless {
			if err := s.checkAccountState(ctx, u, idpstore.PasswordCredential{}, now, meta); err != nil {
				return idp.AuthResult{}, err
			}
			if err := s.store.RecordSuccessfulLogin(ctx, u.ID, now, nil); err != nil {
				return idp.AuthResult{}, fmt.Errorf("%w: reset successful login", ErrAuthenticationUnavailable)
			}
			s.emit(ctx, "password.login.success", meta, u.Sub, "")
			return idp.AuthResult{User: u, AMR: []string{"pwd"}}, nil
		}
		if !errors.Is(credErr, idpstore.ErrNotFound) {
			s.emit(ctx, "password.login.unavailable", meta, u.Sub, "store_error")
			return idp.AuthResult{}, fmt.Errorf("%w: load credential", ErrAuthenticationUnavailable)
		}
		if err := s.dummyVerify(ctx, passwordBytes); err != nil {
			return idp.AuthResult{}, err
		}
		s.emit(ctx, "password.login.failure", meta, u.Sub, "invalid_credentials")
		return idp.AuthResult{}, ErrInvalidCredentials
	}
	if err := s.checkAccountState(ctx, u, credential, now, meta); err != nil {
		if !errors.Is(err, ErrAccountDisabled) && !errors.Is(err, ErrAccountLocked) {
			return idp.AuthResult{}, fmt.Errorf("%w: account state", ErrAuthenticationUnavailable)
		}
		return idp.AuthResult{}, err
	}

	release, err := s.beginPasswordWork(ctx)
	if err != nil {
		return idp.AuthResult{}, err
	}
	needsRehash, err := s.hasher.VerifyPassword(passwordBytes, credential.PasswordHash)
	release()
	if err != nil {
		if !errors.Is(err, passwordhash.ErrPasswordMismatch) {
			s.emit(ctx, "password.login.unavailable", meta, u.Sub, "credential_error")
			return idp.AuthResult{}, fmt.Errorf("%w: verify credential", ErrAuthenticationUnavailable)
		}
		if failureErr := s.recordFailure(ctx, u.ID, now); failureErr != nil {
			s.emit(ctx, "password.login.unavailable", meta, u.Sub, "store_error")
			return idp.AuthResult{}, fmt.Errorf("%w: record failed login", ErrAuthenticationUnavailable)
		}
		s.emit(ctx, "password.login.failure", meta, u.Sub, "invalid_credentials")
		return idp.AuthResult{}, ErrInvalidCredentials
	}
	if needsRehash {
		updated, err := s.rehashCredential(ctx, credential, passwordBytes, now)
		if err != nil {
			return idp.AuthResult{}, fmt.Errorf("%w: rehash credential", ErrAuthenticationUnavailable)
		}
		if err := s.store.PutPasswordCredential(ctx, updated); err != nil {
			return idp.AuthResult{}, fmt.Errorf("%w: persist rehash", ErrAuthenticationUnavailable)
		}
	}
	if err := s.store.RecordSuccessfulLogin(ctx, u.ID, now, nil); err != nil {
		s.emit(ctx, "password.login.unavailable", meta, u.Sub, "store_error")
		return idp.AuthResult{}, fmt.Errorf("%w: reset successful login", ErrAuthenticationUnavailable)
	}
	s.emit(ctx, "password.login.success", meta, u.Sub, "")
	return idp.AuthResult{User: u, AMR: []string{"pwd"}}, nil
}

func (s *Service) hashCredential(ctx context.Context, userID, login string, password []byte, now time.Time) (idpstore.PasswordCredential, error) {
	normalized, err := s.acceptance.NormalizeAndValidatePassword(ctx, password, login, userID)
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	release, err := s.beginPasswordWork(ctx)
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	encoded, err := s.hasher.HashPassword(normalized)
	release()
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	params := idpstore.PasswordHashParams{}
	if parsed, err := passwordhash.Parse(encoded); err == nil {
		params = idpstore.PasswordHashParams(parsed.Params)
	}
	return idpstore.PasswordCredential{UserID: userID, Login: user.Normalize(login), PasswordHash: encoded, HashAlgorithm: passwordhash.AlgorithmArgon2id, HashParams: params, CreatedAt: now, UpdatedAt: now, PasswordChangedAt: now}, nil
}

func (s *Service) checkAccountState(ctx context.Context, u idpstore.User, credential idpstore.PasswordCredential, now time.Time, meta idp.LoginMetadata) error {
	if u.Disabled || credential.Disabled {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_disabled")
		return ErrAccountDisabled
	}
	if u.LockedUntil != nil && now.Before(*u.LockedUntil) {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_locked")
		return ErrAccountLocked
	}
	state, err := s.store.GetAccountSecurityState(ctx, u.ID)
	if err != nil && !errors.Is(err, idpstore.ErrNotFound) {
		return err
	}
	if state.LockedUntil != nil && now.Before(*state.LockedUntil) {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_locked")
		return ErrAccountLocked
	}
	return nil
}

func (s *Service) recordFailure(ctx context.Context, userID string, now time.Time) error {
	_, err := s.store.RecordFailedLogin(ctx, userID, now, idpstore.LockoutPolicy{
		Threshold: s.policy.LockoutThreshold,
		Window:    s.policy.LockoutWindow,
		Duration:  s.policy.LockoutDuration,
	})
	return err
}

func (s *Service) rehashCredential(ctx context.Context, credential idpstore.PasswordCredential, password []byte, now time.Time) (idpstore.PasswordCredential, error) {
	release, err := s.beginPasswordWork(ctx)
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	encoded, err := s.hasher.HashPassword(password)
	release()
	if err != nil {
		return idpstore.PasswordCredential{}, err
	}
	credential.PasswordHash = encoded
	credential.HashAlgorithm = passwordhash.AlgorithmArgon2id
	if parsed, err := passwordhash.Parse(encoded); err == nil {
		credential.HashParams = idpstore.PasswordHashParams(parsed.Params)
	}
	credential.UpdatedAt = now
	return credential, nil
}

func (s *Service) dummyVerify(ctx context.Context, password []byte) error {
	release, err := s.beginPasswordWork(ctx)
	if err != nil {
		return err
	}
	_, _ = s.hasher.VerifyPassword(password, s.policy.DummyHash)
	release()
	return nil
}

func (s *Service) beginPasswordWork(ctx context.Context) (func(), error) {
	waitStart := time.Now()
	waited := false
	select {
	case s.work <- struct{}{}:
	default:
		waited = true
		s.metrics.saturations.Add(1)
		s.metrics.waiting.Add(1)
		select {
		case s.work <- struct{}{}:
			s.metrics.waiting.Add(-1)
		case <-ctx.Done():
			s.metrics.waiting.Add(-1)
			s.metrics.rejected.Add(1)
			return nil, fmt.Errorf("%w: %w", ErrPasswordWorkRejected, ctx.Err())
		}
	}
	if waited {
		s.metrics.totalWait.Add(time.Since(waitStart).Nanoseconds())
	}
	s.metrics.inFlight.Add(1)
	started := time.Now()
	return func() {
		<-s.work
		s.metrics.inFlight.Add(-1)
		s.metrics.completed.Add(1)
		s.metrics.totalDuration.Add(time.Since(started).Nanoseconds())
	}, nil
}

func (s *Service) PasswordWorkStats() idp.PasswordWorkStats {
	return idp.PasswordWorkStats{
		Capacity:      cap(s.work),
		InFlight:      s.metrics.inFlight.Load(),
		Waiting:       s.metrics.waiting.Load(),
		Saturations:   s.metrics.saturations.Load(),
		Rejected:      s.metrics.rejected.Load(),
		Completed:     s.metrics.completed.Load(),
		TotalWait:     s.metrics.totalWait.Load(),
		TotalDuration: s.metrics.totalDuration.Load(),
	}
}

func (s *Service) ProductionReady() bool {
	return s != nil && cap(s.work) > 0 && s.acceptance.MinCharacters >= 15 && s.acceptance.MaxCharacters >= 64 && s.acceptance.Blocklist != nil
}

func (s *Service) emit(ctx context.Context, name string, meta idp.LoginMetadata, subject, reason string) {
	result := "accepted"
	if reason != "" {
		result = "rejected"
	}
	if err := s.audit.Emit(ctx, idp.Event{Time: s.clock().UTC(), Name: name, ClientID: meta.ClientID, Subject: subject, Result: result, Reason: reason, Fields: map[string]string{"remote_addr": meta.RemoteAddr}}); err != nil {
		s.auditFailures.Add(1)
	}
}

func AuditReason(err error) string {
	switch {
	case errors.Is(err, ErrAccountDisabled):
		return "account_disabled"
	case errors.Is(err, ErrAccountLocked):
		return "account_locked"
	default:
		return "invalid_credentials"
	}
}
