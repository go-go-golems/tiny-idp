package authn

import (
	"context"
	"errors"
	"time"

	"github.com/manuel/tinyidp/internal/audit"
	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/storage"
	"github.com/manuel/tinyidp/internal/user"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrAccountLocked      = errors.New("account locked")
)

type LoginMetadata struct {
	RemoteAddr string
	UserAgent  string
	ClientID   string
}

type AuthResult struct {
	User               domain.User
	MustChangePassword bool
	AMR                []string
}

type PasswordPolicy struct {
	MinLength         int
	MaxLength         int
	LockoutThreshold  int
	LockoutWindow     time.Duration
	LockoutDuration   time.Duration
	AllowPasswordless bool
	DummyHash         []byte
}

func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{MinLength: 8, MaxLength: 1024, LockoutThreshold: 5, LockoutWindow: 15 * time.Minute, LockoutDuration: 15 * time.Minute}
}

type Options struct {
	Hasher passwordhash.Hasher
	Policy PasswordPolicy
	Clock  func() time.Time
	Audit  audit.Sink
}

type PasswordService struct {
	store  storage.Store
	hasher passwordhash.Hasher
	policy PasswordPolicy
	clock  func() time.Time
	audit  audit.Sink
}

func NewPasswordService(store storage.Store, opts Options) (*PasswordService, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	hasher := opts.Hasher
	if hasher.Params.MemoryKiB == 0 {
		hasher = passwordhash.New(passwordhash.DefaultParams())
	}
	policy := opts.Policy
	if policy.MaxLength == 0 && policy.MinLength == 0 && policy.LockoutThreshold == 0 && policy.LockoutWindow == 0 && policy.LockoutDuration == 0 && len(policy.DummyHash) == 0 {
		policy = DefaultPasswordPolicy()
	}
	if policy.MaxLength == 0 {
		policy.MaxLength = 1024
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
		sink = audit.NoopSink{}
	}
	return &PasswordService{store: store, hasher: hasher, policy: policy, clock: clock, audit: sink}, nil
}

func (s *PasswordService) AuthenticatePassword(ctx context.Context, login, password string, meta LoginMetadata) (AuthResult, error) {
	now := s.clock().UTC()
	normalized := user.Normalize(login)
	if normalized == "" || (password == "" && !s.policy.AllowPasswordless) {
		s.dummyVerify(password)
		s.emit(ctx, "password.login.failure", meta, "", "invalid_credentials")
		return AuthResult{}, ErrInvalidCredentials
	}
	if len(password) > s.policy.MaxLength {
		s.dummyVerify(password)
		s.emit(ctx, "password.login.failure", meta, "", "invalid_credentials")
		return AuthResult{}, ErrInvalidCredentials
	}

	u, userErr := s.store.GetUserByLogin(ctx, normalized)
	if userErr != nil {
		s.dummyVerify(password)
		s.emit(ctx, "password.login.failure", meta, "", "invalid_credentials")
		return AuthResult{}, ErrInvalidCredentials
	}
	credential, credErr := s.store.GetPasswordCredentialByLogin(ctx, normalized)
	if credErr != nil {
		if errors.Is(credErr, storage.ErrNotFound) && s.policy.AllowPasswordless {
			if err := s.checkAccountState(ctx, u, domain.PasswordCredential{}, now, meta); err != nil {
				return AuthResult{}, err
			}
			_ = s.store.ResetAccountSecurityState(ctx, u.ID, now)
			s.emit(ctx, "password.login.success", meta, u.Sub, "")
			return AuthResult{User: u, AMR: []string{"pwd"}}, nil
		}
		s.dummyVerify(password)
		s.emit(ctx, "password.login.failure", meta, u.Sub, "invalid_credentials")
		return AuthResult{}, ErrInvalidCredentials
	}
	if err := s.checkAccountState(ctx, u, credential, now, meta); err != nil {
		return AuthResult{}, err
	}

	needsRehash, err := s.hasher.VerifyPassword([]byte(password), credential.PasswordHash)
	if err != nil {
		_ = s.recordFailure(ctx, u.ID, now)
		s.emit(ctx, "password.login.failure", meta, u.Sub, "invalid_credentials")
		return AuthResult{}, ErrInvalidCredentials
	}
	if needsRehash {
		if updated, err := s.rehashCredential(credential, []byte(password), now); err == nil {
			_ = s.store.PutPasswordCredential(ctx, updated)
		}
	}
	_ = s.store.ResetAccountSecurityState(ctx, u.ID, now)
	s.emit(ctx, "password.login.success", meta, u.Sub, "")
	return AuthResult{User: u, MustChangePassword: credential.MustChangeAtLogin, AMR: []string{"pwd"}}, nil
}

func (s *PasswordService) HashCredential(userID, login string, password []byte, now time.Time) (domain.PasswordCredential, error) {
	encoded, err := s.hasher.HashPassword(password)
	if err != nil {
		return domain.PasswordCredential{}, err
	}
	params := domain.PasswordHashParams{}
	if parsed, err := passwordhash.Parse(encoded); err == nil {
		params = domain.PasswordHashParams(parsed.Params)
	}
	return domain.PasswordCredential{UserID: userID, Login: user.Normalize(login), PasswordHash: encoded, HashAlgorithm: passwordhash.AlgorithmArgon2id, HashParams: params, CreatedAt: now, UpdatedAt: now, PasswordChangedAt: now}, nil
}

func (s *PasswordService) checkAccountState(ctx context.Context, u domain.User, credential domain.PasswordCredential, now time.Time, meta LoginMetadata) error {
	if u.Disabled || credential.Disabled {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_disabled")
		return ErrAccountDisabled
	}
	if u.LockedUntil != nil && now.Before(*u.LockedUntil) {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_locked")
		return ErrAccountLocked
	}
	state, err := s.store.GetAccountSecurityState(ctx, u.ID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	if state.LockedUntil != nil && now.Before(*state.LockedUntil) {
		s.emit(ctx, "password.login.failure", meta, u.Sub, "account_locked")
		return ErrAccountLocked
	}
	return nil
}

func (s *PasswordService) recordFailure(ctx context.Context, userID string, now time.Time) error {
	state, err := s.store.GetAccountSecurityState(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return err
	}
	state.UserID = userID
	if state.FirstFailedLoginAt == nil || (s.policy.LockoutWindow > 0 && now.Sub(*state.FirstFailedLoginAt) > s.policy.LockoutWindow) {
		state.FailedLoginCount = 0
		first := now
		state.FirstFailedLoginAt = &first
	}
	state.FailedLoginCount++
	last := now
	state.LastFailedLoginAt = &last
	if s.policy.LockoutThreshold > 0 && state.FailedLoginCount >= s.policy.LockoutThreshold {
		lockedUntil := now.Add(s.policy.LockoutDuration)
		state.LockedUntil = &lockedUntil
	}
	return s.store.PutAccountSecurityState(ctx, state)
}

func (s *PasswordService) rehashCredential(credential domain.PasswordCredential, password []byte, now time.Time) (domain.PasswordCredential, error) {
	encoded, err := s.hasher.HashPassword(password)
	if err != nil {
		return domain.PasswordCredential{}, err
	}
	credential.PasswordHash = encoded
	credential.HashAlgorithm = passwordhash.AlgorithmArgon2id
	if parsed, err := passwordhash.Parse(encoded); err == nil {
		credential.HashParams = domain.PasswordHashParams(parsed.Params)
	}
	credential.UpdatedAt = now
	return credential, nil
}

func (s *PasswordService) dummyVerify(password string) {
	_, _ = s.hasher.VerifyPassword([]byte(password), s.policy.DummyHash)
}

func (s *PasswordService) emit(ctx context.Context, name string, meta LoginMetadata, subject, reason string) {
	result := "accepted"
	if reason != "" {
		result = "rejected"
	}
	_ = s.audit.Emit(ctx, audit.Event{Time: s.clock().UTC(), Name: name, ClientID: meta.ClientID, Subject: subject, Result: result, Reason: reason, Fields: map[string]string{"remote_addr": meta.RemoteAddr}})
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
