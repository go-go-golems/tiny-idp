package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var errRegistrationAttemptUnavailable = errors.New("registration attempt is unavailable")

type registrationAttempt struct {
	CSRFSecret []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ConsumedAt time.Time
}

func (s *appStore) createRegistrationAttempt(ctx context.Context, rawToken string, attempt registrationAttempt) error {
	if strings.TrimSpace(rawToken) == "" || len(rawToken) > 1024 {
		return errors.New("registration token is required and must be bounded")
	}
	if len(attempt.CSRFSecret) != sha256.Size || attempt.CreatedAt.IsZero() || !attempt.ExpiresAt.After(attempt.CreatedAt) {
		return errors.New("registration attempt is incomplete")
	}
	hash := sha256.Sum256([]byte(rawToken))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO registration_attempts(token_hash, csrf_secret, created_at, expires_at)
VALUES(?, ?, ?, ?)`, hash[:], attempt.CSRFSecret, formatAppTime(attempt.CreatedAt), formatAppTime(attempt.ExpiresAt))
	return errors.Wrap(err, "create registration attempt")
}

func (s *appStore) consumeRegistrationAttempt(ctx context.Context, rawToken string, now time.Time) (registrationAttempt, error) {
	if strings.TrimSpace(rawToken) == "" || len(rawToken) > 1024 {
		return registrationAttempt{}, errRegistrationAttemptUnavailable
	}
	hash := sha256.Sum256([]byte(rawToken))
	var attempt registrationAttempt
	var createdAt, expiresAt, consumedAt string
	err := s.db.QueryRowContext(ctx, `
UPDATE registration_attempts
SET consumed_at = ?
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?
RETURNING csrf_secret, created_at, expires_at, consumed_at`,
		formatAppTime(now), hash[:], formatAppTime(now),
	).Scan(&attempt.CSRFSecret, &createdAt, &expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return registrationAttempt{}, errRegistrationAttemptUnavailable
	}
	if err != nil {
		return registrationAttempt{}, errors.Wrap(err, "consume registration attempt")
	}
	if attempt.CreatedAt, err = parseAppTime(createdAt); err != nil {
		return registrationAttempt{}, err
	}
	if attempt.ExpiresAt, err = parseAppTime(expiresAt); err != nil {
		return registrationAttempt{}, err
	}
	if attempt.ConsumedAt, err = parseAppTime(consumedAt); err != nil {
		return registrationAttempt{}, err
	}
	return attempt, nil
}
