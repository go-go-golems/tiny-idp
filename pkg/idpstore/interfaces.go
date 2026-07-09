package idpstore

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrAlreadyConsumed      = errors.New("already consumed")
	ErrExpired              = errors.New("expired")
	ErrRefreshReuseDetected = errors.New("refresh token reuse detected")
	ErrDuplicate            = errors.New("duplicate")
	ErrLastSigningKey       = errors.New("cannot retire the final active signing key")
	ErrNestedTransaction    = errors.New("nested store transactions are not supported")
)

type ClientStore interface {
	GetClient(ctx context.Context, id string) (Client, error)
	ListClients(ctx context.Context) ([]Client, error)
	PutClient(ctx context.Context, c Client) error
}

type UserStore interface {
	GetUser(ctx context.Context, id string) (User, error)
	GetUserByLogin(ctx context.Context, login string) (User, error)
	PutUser(ctx context.Context, login string, u User) error
}

type PasswordCredentialStore interface {
	PutPasswordCredential(ctx context.Context, credential PasswordCredential) error
	GetPasswordCredentialByLogin(ctx context.Context, login string) (PasswordCredential, error)
	GetPasswordCredentialByUserID(ctx context.Context, userID string) (PasswordCredential, error)
	DeletePasswordCredential(ctx context.Context, userID string) error
}

type AccountSecurityStore interface {
	GetAccountSecurityState(ctx context.Context, userID string) (AccountSecurityState, error)
	PutAccountSecurityState(ctx context.Context, state AccountSecurityState) error
	ResetAccountSecurityState(ctx context.Context, userID string, now time.Time) error
}

type GrantStore interface {
	CreateGrant(ctx context.Context, grant Grant) error
	GetGrant(ctx context.Context, id string) (Grant, error)
	RevokeGrant(ctx context.Context, id string, at time.Time) error
}

type AuthorizationCodeStore interface {
	CreateAuthorizationCode(ctx context.Context, code AuthorizationCode) error
	ConsumeAuthorizationCode(ctx context.Context, codeHash []byte, now time.Time) (AuthorizationCode, error)
}

type AccessTokenStore interface {
	CreateAccessToken(ctx context.Context, token AccessToken) error
	GetAccessToken(ctx context.Context, tokenHash []byte) (AccessToken, error)
	RevokeAccessToken(ctx context.Context, tokenHash []byte, at time.Time) error
}

type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	RotateRefreshToken(ctx context.Context, oldHash []byte, next RefreshToken, now time.Time) (RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash []byte) (RefreshToken, error)
	RevokeRefreshTokenFamily(ctx context.Context, tokenHash []byte, at time.Time) error
}

type ConsentStore interface {
	PutConsent(ctx context.Context, consent Consent) error
	GetConsent(ctx context.Context, userID, clientID string, scopes []string) (Consent, error)
	RevokeConsent(ctx context.Context, userID, clientID string, scopes []string, at time.Time) error
}

type SessionStore interface {
	CreateSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, idHash []byte) (Session, error)
	RevokeSession(ctx context.Context, idHash []byte, at time.Time) error
}

type KeyStore interface {
	ActiveSigningKey(ctx context.Context) (SigningKey, error)
	VerificationKeys(ctx context.Context) ([]SigningKey, error)
	CreateSigningKey(ctx context.Context, key SigningKey) error
	ActivateSigningKey(ctx context.Context, kid string) error
	RetireSigningKey(ctx context.Context, kid string) error
}

type StoreOperations interface {
	ClientStore
	UserStore
	PasswordCredentialStore
	AccountSecurityStore
	GrantStore
	AuthorizationCodeStore
	AccessTokenStore
	RefreshTokenStore
	ConsentStore
	SessionStore
	KeyStore
}

// ReadStore is the store view supplied to a read-only transaction callback.
// Implementations may also expose mutation methods internally, but callers
// receive only this read contract.
type ReadStore interface {
	GetClient(ctx context.Context, id string) (Client, error)
	ListClients(ctx context.Context) ([]Client, error)
	GetUser(ctx context.Context, id string) (User, error)
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetPasswordCredentialByLogin(ctx context.Context, login string) (PasswordCredential, error)
	GetPasswordCredentialByUserID(ctx context.Context, userID string) (PasswordCredential, error)
	GetAccountSecurityState(ctx context.Context, userID string) (AccountSecurityState, error)
	GetGrant(ctx context.Context, id string) (Grant, error)
	GetAccessToken(ctx context.Context, tokenHash []byte) (AccessToken, error)
	GetRefreshToken(ctx context.Context, tokenHash []byte) (RefreshToken, error)
	GetConsent(ctx context.Context, userID, clientID string, scopes []string) (Consent, error)
	GetSession(ctx context.Context, idHash []byte) (Session, error)
	ActiveSigningKey(ctx context.Context) (SigningKey, error)
	VerificationKeys(ctx context.Context) ([]SigningKey, error)
}

// TxStore is the mutation surface scoped to one implementation transaction.
type TxStore interface {
	StoreOperations
}

// LockoutPolicy controls the atomic failed-login window and lock duration.
type LockoutPolicy struct {
	Threshold int
	Window    time.Duration
	Duration  time.Duration
}

// RotationResult reports the newly active key and the previous key, if any.
type RotationResult struct {
	Active  SigningKey
	Retired *SigningKey
}

// AtomicStore exposes transaction callbacks and named security invariants.
// Callback-scoped stores must not be retained; nested transactions fail with
// ErrNestedTransaction.
type AtomicStore interface {
	View(ctx context.Context, fn func(ReadStore) error) error
	Update(ctx context.Context, fn func(TxStore) error) error
	CreateUserWithCredential(ctx context.Context, login string, user User, credential PasswordCredential) error
	ReplacePasswordAndSecurityState(ctx context.Context, credential PasswordCredential, state AccountSecurityState) error
	RecordFailedLogin(ctx context.Context, userID string, now time.Time, policy LockoutPolicy) (AccountSecurityState, error)
	RecordSuccessfulLogin(ctx context.Context, userID string, now time.Time, session *Session) error
	RotateSigningKey(ctx context.Context, next SigningKey, now time.Time) (RotationResult, error)
}

// Store is the complete persistence contract consumed by the embedded IdP.
type Store interface {
	StoreOperations
	AtomicStore
}

// PersistentReporter lets startup validation distinguish production-capable
// durable stores from development-only stores without depending on concrete
// package names.
type PersistentReporter interface {
	Persistent() bool
}

// SchemaReporter exposes the durable schema version for production preflight.
type SchemaReporter interface {
	SchemaVersion(ctx context.Context) (int, error)
}
