package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var errLoginAttemptUnavailable = errors.New("login attempt is unavailable")

type loginAttempt struct {
	Nonce        string
	PKCEVerifier string
	ReturnTo     string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	ConsumedAt   time.Time
}

func (s *appStore) createLoginAttempt(ctx context.Context, state string, attempt loginAttempt) error {
	if strings.TrimSpace(state) == "" || len(state) > 1024 {
		return errors.New("login state is required and must be bounded")
	}
	if attempt.Nonce == "" || attempt.PKCEVerifier == "" || attempt.ReturnTo == "" ||
		attempt.CreatedAt.IsZero() || !attempt.ExpiresAt.After(attempt.CreatedAt) {
		return errors.New("login attempt is incomplete")
	}
	hash := sha256.Sum256([]byte(state))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO oidc_login_attempts(state_hash, nonce, pkce_verifier, return_to, created_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?)`,
		hash[:], attempt.Nonce, attempt.PKCEVerifier, attempt.ReturnTo,
		formatAppTime(attempt.CreatedAt), formatAppTime(attempt.ExpiresAt),
	)
	return errors.Wrap(err, "create OIDC login attempt")
}

func (s *appStore) consumeLoginAttempt(ctx context.Context, state string, now time.Time) (loginAttempt, error) {
	if strings.TrimSpace(state) == "" || len(state) > 1024 {
		return loginAttempt{}, errLoginAttemptUnavailable
	}
	hash := sha256.Sum256([]byte(state))
	var attempt loginAttempt
	var createdAt, expiresAt, consumedAt string
	err := s.db.QueryRowContext(ctx, `
UPDATE oidc_login_attempts
SET consumed_at = ?
WHERE state_hash = ? AND consumed_at IS NULL AND expires_at > ?
RETURNING nonce, pkce_verifier, return_to, created_at, expires_at, consumed_at`,
		formatAppTime(now), hash[:], formatAppTime(now),
	).Scan(&attempt.Nonce, &attempt.PKCEVerifier, &attempt.ReturnTo, &createdAt, &expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return loginAttempt{}, errLoginAttemptUnavailable
	}
	if err != nil {
		return loginAttempt{}, errors.Wrap(err, "consume OIDC login attempt")
	}
	if attempt.CreatedAt, err = parseAppTime(createdAt); err != nil {
		return loginAttempt{}, err
	}
	if attempt.ExpiresAt, err = parseAppTime(expiresAt); err != nil {
		return loginAttempt{}, err
	}
	if attempt.ConsumedAt, err = parseAppTime(consumedAt); err != nil {
		return loginAttempt{}, err
	}
	return attempt, nil
}
