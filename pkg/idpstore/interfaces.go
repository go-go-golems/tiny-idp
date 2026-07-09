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

type Store interface {
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

// PersistentReporter lets startup validation distinguish production-capable
// durable stores from development-only stores without depending on concrete
// package names.
type PersistentReporter interface {
	Persistent() bool
}
