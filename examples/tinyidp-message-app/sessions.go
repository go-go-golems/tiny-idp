package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var errSessionUnavailable = errors.New("application session is unavailable")

type appSession struct {
	Subject     string
	DisplayName string
	CSRFSecret  []byte
	CreatedAt   time.Time
	LastSeenAt  time.Time
	ExpiresAt   time.Time
	RevokedAt   time.Time
}

func (s *appStore) createAppSession(ctx context.Context, rawToken string, session appSession) error {
	if strings.TrimSpace(rawToken) == "" || len(rawToken) > 1024 {
		return errors.New("application session token is required and must be bounded")
	}
	if session.Subject == "" || session.DisplayName == "" || session.CreatedAt.IsZero() ||
		!session.ExpiresAt.After(session.CreatedAt) || len(session.CSRFSecret) != sha256.Size {
		return errors.New("application session is incomplete")
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = session.CreatedAt
	}
	hash := sha256.Sum256([]byte(rawToken))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO app_sessions(token_hash, subject, display_name, csrf_secret, created_at, last_seen_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?, ?)`, hash[:], session.Subject, session.DisplayName, session.CSRFSecret,
		formatAppTime(session.CreatedAt), formatAppTime(session.LastSeenAt), formatAppTime(session.ExpiresAt))
	return errors.Wrap(err, "create application session")
}

func (s *appStore) getAppSession(ctx context.Context, rawToken string, now time.Time) (appSession, error) {
	if strings.TrimSpace(rawToken) == "" || len(rawToken) > 1024 {
		return appSession{}, errSessionUnavailable
	}
	hash := sha256.Sum256([]byte(rawToken))
	var session appSession
	var createdAt, lastSeenAt, expiresAt string
	err := s.db.QueryRowContext(ctx, `
SELECT subject, display_name, csrf_secret, created_at, last_seen_at, expires_at
FROM app_sessions
WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ?`, hash[:], formatAppTime(now)).
		Scan(&session.Subject, &session.DisplayName, &session.CSRFSecret, &createdAt, &lastSeenAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return appSession{}, errSessionUnavailable
	}
	if err != nil {
		return appSession{}, errors.Wrap(err, "load application session")
	}
	if session.CreatedAt, err = parseAppTime(createdAt); err != nil {
		return appSession{}, err
	}
	if session.LastSeenAt, err = parseAppTime(lastSeenAt); err != nil {
		return appSession{}, err
	}
	if session.ExpiresAt, err = parseAppTime(expiresAt); err != nil {
		return appSession{}, err
	}
	return session, nil
}

func (s *appStore) revokeAppSession(ctx context.Context, rawToken string, now time.Time) error {
	if strings.TrimSpace(rawToken) == "" || len(rawToken) > 1024 {
		return errSessionUnavailable
	}
	hash := sha256.Sum256([]byte(rawToken))
	result, err := s.db.ExecContext(ctx, `
UPDATE app_sessions SET revoked_at = ?
WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ?`,
		formatAppTime(now), hash[:], formatAppTime(now))
	if err != nil {
		return errors.Wrap(err, "revoke application session")
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "read application session revocation result")
	}
	if changed != 1 {
		return errSessionUnavailable
	}
	return nil
}

func (s *appStore) touchAppSession(ctx context.Context, rawToken string, previous, now time.Time) error {
	hash := sha256.Sum256([]byte(rawToken))
	_, err := s.db.ExecContext(ctx, `
UPDATE app_sessions SET last_seen_at = ?
WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ? AND last_seen_at = ?`,
		formatAppTime(now), hash[:], formatAppTime(now), formatAppTime(previous))
	return errors.Wrap(err, "touch application session")
}
