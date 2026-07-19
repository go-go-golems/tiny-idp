package memory

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
)

// The continuation operations intentionally use the same Store and transaction
// snapshots as account and interaction state. This lets a signup committer use
// one native transaction in development and test configurations too.
func (s *Store) Create(_ context.Context, record idpcontinuation.WorkflowContinuation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(record.HandleHash)
	if _, exists := s.continuations[key]; exists {
		return idpcontinuation.ErrConflict
	}
	s.continuations[key] = cloneContinuation(record)
	return nil
}

func (s *Store) Load(_ context.Context, hash []byte, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.continuations[hashKey(hash)]
	if !ok {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrNotFound
	}
	if err := continuationLoadState(record, now); err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	return cloneContinuation(record), nil
}

func (s *Store) Advance(_ context.Context, hash []byte, expectedRevision uint64, next idpcontinuation.WorkflowContinuation, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	current, ok := s.continuations[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if err := continuationLoadState(current, now); err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	nextKey := hashKey(next.HandleHash)
	if _, exists := s.continuations[nextKey]; exists {
		return idpcontinuation.ErrConflict
	}
	current.Status = idpcontinuation.StatusAdvanced
	current.Revision++
	s.continuations[key] = current
	s.continuations[nextKey] = cloneContinuation(next)
	return nil
}

func (s *Store) Consume(_ context.Context, hash []byte, expectedRevision uint64, outcome idpcontinuation.TerminalOutcome, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.continuations[key]
	if !ok {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrNotFound
	}
	if err := continuationLoadState(record, now); err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusConsumed
	record.Revision++
	record.Terminal = &outcome
	s.continuations[key] = record
	return cloneContinuation(record), nil
}

func (s *Store) Revoke(_ context.Context, hash []byte, expectedRevision uint64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.continuations[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if err := continuationLoadState(record, now); err != nil {
		return err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusRevoked
	record.Revision++
	s.continuations[key] = record
	return nil
}

func (s *Store) ListExpired(_ context.Context, now time.Time, limit int) ([]idpcontinuation.WorkflowContinuation, error) {
	if limit <= 0 {
		return nil, idpcontinuation.ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]idpcontinuation.WorkflowContinuation, 0, limit)
	for _, record := range s.continuations {
		if len(result) == limit {
			break
		}
		if !record.ExpiresAt.After(now) {
			result = append(result, cloneContinuation(record))
		}
	}
	return result, nil
}

func (s *Store) DeleteExpired(_ context.Context, hash []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.continuations[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if record.ExpiresAt.After(now) {
		return idpcontinuation.ErrConflict
	}
	delete(s.continuations, key)
	return nil
}

func continuationLoadState(record idpcontinuation.WorkflowContinuation, now time.Time) error {
	if !record.ExpiresAt.After(now) {
		return idpcontinuation.ErrExpired
	}
	switch record.Status {
	case idpcontinuation.StatusActive:
		return nil
	case idpcontinuation.StatusRevoked:
		return idpcontinuation.ErrRevoked
	case idpcontinuation.StatusAdvanced, idpcontinuation.StatusConsumed:
		return idpcontinuation.ErrAlreadyTerminal
	default:
		return idpcontinuation.ErrConflict
	}
}

func cloneContinuation(record idpcontinuation.WorkflowContinuation) idpcontinuation.WorkflowContinuation {
	encoded, _ := json.Marshal(record)
	var clone idpcontinuation.WorkflowContinuation
	_ = json.Unmarshal(encoded, &clone)
	return clone
}

func cloneContinuationMap(source map[string]idpcontinuation.WorkflowContinuation) map[string]idpcontinuation.WorkflowContinuation {
	clone := make(map[string]idpcontinuation.WorkflowContinuation, len(source))
	for key, record := range source {
		clone[key] = cloneContinuation(record)
	}
	return clone
}
