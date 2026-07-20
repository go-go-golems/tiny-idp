package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
)

var _ idpcontinuation.Store = (*Store)(nil)

func (s *Store) Create(ctx context.Context, record idpcontinuation.WorkflowContinuation) error {
	encoded, err := json.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "encode workflow continuation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.db.ExecContext(ctx, `INSERT INTO workflow_continuations(handle_hash, revision, status, expires_at_ns, data) VALUES(?, ?, ?, ?, ?)`,
		record.HandleHash, record.Revision, record.Status, record.ExpiresAt.UnixNano(), encoded)
	if err != nil {
		return errors.Wrap(idpcontinuation.ErrConflict, err.Error())
	}
	return nil
}

func (s *Store) Load(ctx context.Context, hash []byte, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return loadContinuation(ctx, s.db, hash, now)
}

func (s *Store) Advance(ctx context.Context, hash []byte, expectedRevision uint64, next idpcontinuation.WorkflowContinuation, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "begin workflow continuation advance")
	}
	defer tx.Rollback() //nolint:errcheck // rollback is a no-op after commit.
	current, err := loadContinuation(ctx, tx, hash, now)
	if err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	nextEncoded, err := json.Marshal(next)
	if err != nil {
		return errors.Wrap(err, "encode next workflow continuation")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workflow_continuations(handle_hash, revision, status, expires_at_ns, data) VALUES(?, ?, ?, ?, ?)`,
		next.HandleHash, next.Revision, next.Status, next.ExpiresAt.UnixNano(), nextEncoded); err != nil {
		return errors.Wrap(idpcontinuation.ErrConflict, err.Error())
	}
	current.Status = idpcontinuation.StatusAdvanced
	current.Revision++
	currentEncoded, err := json.Marshal(current)
	if err != nil {
		return errors.Wrap(err, "encode advanced workflow continuation")
	}
	result, err := tx.ExecContext(ctx, `UPDATE workflow_continuations SET revision=?, status=?, data=? WHERE handle_hash=? AND revision=? AND status=?`,
		current.Revision, current.Status, currentEncoded, hash, expectedRevision, idpcontinuation.StatusActive)
	if err != nil {
		return errors.Wrap(err, "advance workflow continuation")
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	return errors.Wrap(tx.Commit(), "commit workflow continuation advance")
}

func (s *Store) Consume(ctx context.Context, hash []byte, expectedRevision uint64, outcome idpcontinuation.TerminalOutcome, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runner != nil {
		return consumeContinuation(ctx, s.conn(), hash, expectedRevision, outcome, now)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "begin workflow continuation consume")
	}
	defer tx.Rollback() //nolint:errcheck // rollback is a no-op after commit.
	record, err := consumeContinuation(ctx, tx, hash, expectedRevision, outcome, now)
	if err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	if err := tx.Commit(); err != nil {
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "commit workflow continuation consume")
	}
	return record, nil
}

func consumeContinuation(ctx context.Context, query sqlRunner, hash []byte, expectedRevision uint64, outcome idpcontinuation.TerminalOutcome, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	record, err := loadContinuation(ctx, query, hash, now)
	if err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusConsumed
	record.Revision++
	record.Terminal = &outcome
	encoded, err := json.Marshal(record)
	if err != nil {
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "encode consumed workflow continuation")
	}
	result, err := query.ExecContext(ctx, `UPDATE workflow_continuations SET revision=?, status=?, data=? WHERE handle_hash=? AND revision=? AND status=?`,
		record.Revision, record.Status, encoded, hash, expectedRevision, idpcontinuation.StatusActive)
	if err != nil {
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "consume workflow continuation")
	}
	if err := requireOneRow(result); err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	return record, nil
}

func (s *Store) Revoke(ctx context.Context, hash []byte, expectedRevision uint64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := loadContinuation(ctx, s.db, hash, now)
	if err != nil {
		return err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusRevoked
	record.Revision++
	encoded, err := json.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "encode revoked workflow continuation")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE workflow_continuations SET revision=?, status=?, data=? WHERE handle_hash=? AND revision=? AND status=?`,
		record.Revision, record.Status, encoded, hash, expectedRevision, idpcontinuation.StatusActive)
	if err != nil {
		return errors.Wrap(err, "revoke workflow continuation")
	}
	return requireOneRow(result)
}

func (s *Store) ListExpired(ctx context.Context, now time.Time, limit int) ([]idpcontinuation.WorkflowContinuation, error) {
	if limit <= 0 {
		return nil, errors.New("cleanup limit must be greater than zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM workflow_continuations WHERE expires_at_ns <= ? ORDER BY expires_at_ns, handle_hash LIMIT ?`, now.UnixNano(), limit)
	if err != nil {
		return nil, errors.Wrap(err, "select expired workflow continuations")
	}
	var expired []idpcontinuation.WorkflowContinuation
	for rows.Next() {
		var encoded []byte
		if err := rows.Scan(&encoded); err != nil {
			_ = rows.Close()
			return nil, errors.Wrap(err, "scan expired workflow continuation")
		}
		var record idpcontinuation.WorkflowContinuation
		if err := json.Unmarshal(encoded, &record); err != nil {
			_ = rows.Close()
			return nil, errors.Wrap(err, "decode expired workflow continuation")
		}
		expired = append(expired, record)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Wrap(err, "close expired continuation rows")
	}
	return expired, nil
}

func (s *Store) DeleteExpired(ctx context.Context, hash []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.db.ExecContext(ctx, `DELETE FROM workflow_continuations WHERE handle_hash=? AND expires_at_ns <= ?`, hash, now.UnixNano())
	if err != nil {
		return errors.Wrap(err, "delete expired workflow continuation")
	}
	return requireOneRow(result)
}

type continuationQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadContinuation(ctx context.Context, query continuationQuery, hash []byte, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	var encoded []byte
	if err := query.QueryRowContext(ctx, `SELECT data FROM workflow_continuations WHERE handle_hash=?`, hash).Scan(&encoded); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrNotFound
		}
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "load workflow continuation")
	}
	var record idpcontinuation.WorkflowContinuation
	if err := json.Unmarshal(encoded, &record); err != nil {
		return idpcontinuation.WorkflowContinuation{}, errors.Wrap(err, "decode workflow continuation")
	}
	if !record.ExpiresAt.After(now) {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrExpired
	}
	switch record.Status {
	case idpcontinuation.StatusActive:
		return record, nil
	case idpcontinuation.StatusRevoked:
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrRevoked
	case idpcontinuation.StatusAdvanced, idpcontinuation.StatusConsumed:
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrAlreadyTerminal
	default:
		return idpcontinuation.WorkflowContinuation{}, errors.Errorf("invalid continuation status %q", record.Status)
	}
}

func requireOneRow(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "read continuation rows affected")
	}
	if count != 1 {
		return idpcontinuation.ErrConflict
	}
	return nil
}
