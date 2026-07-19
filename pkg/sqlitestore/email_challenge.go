package sqlitestore

import (
	"bytes"
	"context"
	"database/sql"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/pkg/errors"
)

var _ idpemailchallenge.Store = (*Store)(nil)

func (s *Store) CreateEmailChallenge(ctx context.Context, c idpemailchallenge.PendingChallenge) error {
	if err := c.ValidateForCreate(); err != nil {
		return err
	}
	b, err := enc(c)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO email_challenges(id,expires_at_ns,data) VALUES(?,?,?)`, c.ID, c.ExpiresAt.UnixNano(), b)
	if err != nil {
		return idpemailchallenge.ErrConflict
	}
	return nil
}
func (s *Store) LoadEmailChallenge(ctx context.Context, id string, now time.Time) (idpemailchallenge.PendingChallenge, error) {
	return s.load(ctx, s.db, id, now)
}
func (s *Store) VerifyEmailChallenge(ctx context.Context, id string, hash []byte, b idpemailchallenge.VerificationBindings, now time.Time) (idpemailchallenge.VerifiedEmailEvidence, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return idpemailchallenge.VerifiedEmailEvidence{}, err
	}
	defer tx.Rollback()
	c, err := s.checked(ctx, tx, id, b, now)
	if err != nil {
		return idpemailchallenge.VerifiedEmailEvidence{}, err
	}
	if c.Attempts >= c.MaximumAttempts {
		return idpemailchallenge.VerifiedEmailEvidence{}, idpemailchallenge.ErrAttemptsExceeded
	}
	if !bytes.Equal(c.CodeHash, hash) {
		c.Attempts++
		if err := s.save(ctx, tx, c); err != nil {
			return idpemailchallenge.VerifiedEmailEvidence{}, err
		}
		if c.Attempts >= c.MaximumAttempts {
			return idpemailchallenge.VerifiedEmailEvidence{}, idpemailchallenge.ErrAttemptsExceeded
		}
		return idpemailchallenge.VerifiedEmailEvidence{}, idpemailchallenge.ErrConflict
	}
	at := now.UTC()
	c.Status = idpemailchallenge.StatusVerified
	c.VerifiedAt = &at
	if err := s.save(ctx, tx, c); err != nil {
		return idpemailchallenge.VerifiedEmailEvidence{}, err
	}
	if err := tx.Commit(); err != nil {
		return idpemailchallenge.VerifiedEmailEvidence{}, err
	}
	return idpemailchallenge.VerifiedEmailEvidence{Version: 1, ChallengeID: c.ID, Address: c.Email, Method: "email_code", VerifiedAt: at}, nil
}
func (s *Store) RecordEmailChallengeAttempt(ctx context.Context, id string, b idpemailchallenge.VerificationBindings, now time.Time) (idpemailchallenge.AttemptResult, error) {
	c, err := s.checked(ctx, s.db, id, b, now)
	if err != nil {
		return idpemailchallenge.AttemptResult{}, err
	}
	c.Attempts++
	if err = s.save(ctx, s.db, c); err != nil {
		return idpemailchallenge.AttemptResult{}, err
	}
	return idpemailchallenge.AttemptResult{RemainingAttempts: c.MaximumAttempts - c.Attempts, Terminal: c.Attempts >= c.MaximumAttempts}, nil
}
func (s *Store) ResendEmailChallenge(ctx context.Context, id string, hash []byte, b idpemailchallenge.VerificationBindings, now time.Time) (idpemailchallenge.PendingChallenge, error) {
	if len(hash) < 32 {
		return idpemailchallenge.PendingChallenge{}, idpemailchallenge.ErrConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return idpemailchallenge.PendingChallenge{}, err
	}
	defer tx.Rollback()
	c, err := s.checked(ctx, tx, id, b, now)
	if err != nil {
		return idpemailchallenge.PendingChallenge{}, err
	}
	if c.Resends >= c.MaximumResends || now.Before(c.ResendNotBefore) {
		return idpemailchallenge.PendingChallenge{}, idpemailchallenge.ErrResendLimited
	}
	c.CodeHash = append([]byte(nil), hash...)
	c.Resends++
	c.LastSentAt = now.UTC()
	if err = s.save(ctx, tx, c); err != nil {
		return idpemailchallenge.PendingChallenge{}, err
	}
	if err = tx.Commit(); err != nil {
		return idpemailchallenge.PendingChallenge{}, err
	}
	return c, nil
}
func (s *Store) ListExpiredEmailChallenges(ctx context.Context, now time.Time, limit int) ([]idpemailchallenge.PendingChallenge, error) {
	if limit <= 0 {
		return nil, idpemailchallenge.ErrConflict
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM email_challenges WHERE expires_at_ns<=? LIMIT ?`, now.UnixNano(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := []idpemailchallenge.PendingChallenge{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		c, e := dec[idpemailchallenge.PendingChallenge](b)
		if e != nil {
			return nil, e
		}
		r = append(r, c)
	}
	return r, rows.Err()
}
func (s *Store) DeleteExpiredEmailChallenge(ctx context.Context, id string, now time.Time) error {
	r, err := s.db.ExecContext(ctx, `DELETE FROM email_challenges WHERE id=? AND expires_at_ns<=?`, id, now.UnixNano())
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return idpemailchallenge.ErrConflict
	}
	return nil
}

type runner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (s *Store) load(ctx context.Context, q runner, id string, now time.Time) (idpemailchallenge.PendingChallenge, error) {
	var raw []byte
	err := q.QueryRowContext(ctx, `SELECT data FROM email_challenges WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return idpemailchallenge.PendingChallenge{}, idpemailchallenge.ErrNotFound
	}
	if err != nil {
		return idpemailchallenge.PendingChallenge{}, err
	}
	c, err := dec[idpemailchallenge.PendingChallenge](raw)
	if err != nil {
		return c, err
	}
	if !c.ExpiresAt.After(now) {
		return c, idpemailchallenge.ErrExpired
	}
	return c, nil
}
func (s *Store) checked(ctx context.Context, q runner, id string, b idpemailchallenge.VerificationBindings, now time.Time) (idpemailchallenge.PendingChallenge, error) {
	c, err := s.load(ctx, q, id, now)
	if err != nil {
		return c, err
	}
	if c.Status != idpemailchallenge.StatusPending {
		return c, idpemailchallenge.ErrAlreadyTerminal
	}
	if err = c.VerifyBindings(b); err != nil {
		return c, err
	}
	return c, nil
}
func (s *Store) save(ctx context.Context, q runner, c idpemailchallenge.PendingChallenge) error {
	raw, err := enc(c)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `UPDATE email_challenges SET data=? WHERE id=?`, raw, c.ID)
	return err
}
