package storage

import (
	"context"
	"errors"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
)

var (
	ErrNotFound             = errors.New("not found")
	ErrAlreadyConsumed      = errors.New("already consumed")
	ErrExpired              = errors.New("expired")
	ErrRefreshReuseDetected = errors.New("refresh token reuse detected")
	ErrDuplicate            = errors.New("duplicate")
)

type ClientStore interface {
	GetClient(ctx context.Context, id string) (domain.Client, error)
	ListClients(ctx context.Context) ([]domain.Client, error)
	PutClient(ctx context.Context, c domain.Client) error
}

type UserStore interface {
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
	PutUser(ctx context.Context, login string, u domain.User) error
}

type GrantStore interface {
	CreateGrant(ctx context.Context, grant domain.Grant) error
	GetGrant(ctx context.Context, id string) (domain.Grant, error)
	RevokeGrant(ctx context.Context, id string, at time.Time) error
}

type AuthorizationCodeStore interface {
	CreateAuthorizationCode(ctx context.Context, code domain.AuthorizationCode) error
	ConsumeAuthorizationCode(ctx context.Context, codeHash []byte, now time.Time) (domain.AuthorizationCode, error)
}

type AccessTokenStore interface {
	CreateAccessToken(ctx context.Context, token domain.AccessToken) error
	GetAccessToken(ctx context.Context, tokenHash []byte) (domain.AccessToken, error)
	RevokeAccessToken(ctx context.Context, tokenHash []byte, at time.Time) error
}

type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, token domain.RefreshToken) error
	RotateRefreshToken(ctx context.Context, oldHash []byte, next domain.RefreshToken, now time.Time) (domain.RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash []byte) (domain.RefreshToken, error)
	RevokeRefreshTokenFamily(ctx context.Context, tokenHash []byte, at time.Time) error
}

type SessionStore interface {
	CreateSession(ctx context.Context, session domain.Session) error
	GetSession(ctx context.Context, idHash []byte) (domain.Session, error)
	RevokeSession(ctx context.Context, idHash []byte, at time.Time) error
}

type KeyStore interface {
	ActiveSigningKey(ctx context.Context) (domain.SigningKey, error)
	VerificationKeys(ctx context.Context) ([]domain.SigningKey, error)
	CreateSigningKey(ctx context.Context, key domain.SigningKey) error
	ActivateSigningKey(ctx context.Context, kid string) error
	RetireSigningKey(ctx context.Context, kid string) error
}

type Store interface {
	ClientStore
	UserStore
	GrantStore
	AuthorizationCodeStore
	AccessTokenStore
	RefreshTokenStore
	SessionStore
	KeyStore
}
