// Package memorystore provides concurrency-safe ephemeral implementations of
// Tiny-IDP domain stores for tests and explicitly non-durable deployments.
package memorystore

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
)

type ContinuationStore struct {
	mu      sync.Mutex
	records map[string]idpcontinuation.WorkflowContinuation
}

var _ idpcontinuation.Store = (*ContinuationStore)(nil)

func NewContinuationStore() *ContinuationStore {
	return &ContinuationStore{records: map[string]idpcontinuation.WorkflowContinuation{}}
}

func (s *ContinuationStore) Create(_ context.Context, record idpcontinuation.WorkflowContinuation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(record.HandleHash)
	if _, exists := s.records[key]; exists {
		return idpcontinuation.ErrConflict
	}
	s.records[key] = cloneContinuation(record)
	return nil
}

func (s *ContinuationStore) Load(_ context.Context, hash []byte, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[hashKey(hash)]
	if !ok {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrNotFound
	}
	if err := loadStateError(record, now); err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	return cloneContinuation(record), nil
}

func (s *ContinuationStore) Advance(_ context.Context, hash []byte, expectedRevision uint64, next idpcontinuation.WorkflowContinuation, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	current, ok := s.records[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if err := loadStateError(current, now); err != nil {
		return err
	}
	if current.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	nextKey := hashKey(next.HandleHash)
	if _, exists := s.records[nextKey]; exists {
		return idpcontinuation.ErrConflict
	}
	current.Status = idpcontinuation.StatusAdvanced
	current.Revision++
	s.records[key] = current
	s.records[nextKey] = cloneContinuation(next)
	return nil
}

func (s *ContinuationStore) Consume(_ context.Context, hash []byte, expectedRevision uint64, outcome idpcontinuation.TerminalOutcome, now time.Time) (idpcontinuation.WorkflowContinuation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.records[key]
	if !ok {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrNotFound
	}
	if err := loadStateError(record, now); err != nil {
		return idpcontinuation.WorkflowContinuation{}, err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.WorkflowContinuation{}, idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusConsumed
	record.Revision++
	record.Terminal = &outcome
	s.records[key] = record
	return cloneContinuation(record), nil
}

func (s *ContinuationStore) Revoke(_ context.Context, hash []byte, expectedRevision uint64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.records[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if err := loadStateError(record, now); err != nil {
		return err
	}
	if record.Revision != expectedRevision {
		return idpcontinuation.ErrConflict
	}
	record.Status = idpcontinuation.StatusRevoked
	record.Revision++
	s.records[key] = record
	return nil
}

func (s *ContinuationStore) ListExpired(_ context.Context, now time.Time, limit int) ([]idpcontinuation.WorkflowContinuation, error) {
	if limit <= 0 {
		return nil, errors.New("cleanup limit must be greater than zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expired := make([]idpcontinuation.WorkflowContinuation, 0, limit)
	for _, record := range s.records {
		if len(expired) == limit {
			break
		}
		if record.ExpiresAt.After(now) {
			continue
		}
		expired = append(expired, cloneContinuation(record))
	}
	return expired, nil
}

func (s *ContinuationStore) DeleteExpired(_ context.Context, hash []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := hashKey(hash)
	record, ok := s.records[key]
	if !ok {
		return idpcontinuation.ErrNotFound
	}
	if record.ExpiresAt.After(now) {
		return idpcontinuation.ErrConflict
	}
	delete(s.records, key)
	return nil
}

func loadStateError(record idpcontinuation.WorkflowContinuation, now time.Time) error {
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
		return errors.Errorf("invalid continuation status %q", record.Status)
	}
}

func hashKey(hash []byte) string { return hex.EncodeToString(hash) }

func cloneContinuation(record idpcontinuation.WorkflowContinuation) idpcontinuation.WorkflowContinuation {
	record.HandleHash = append([]byte(nil), record.HandleHash...)
	record.RequestDigest = append([]byte(nil), record.RequestDigest...)
	record.BrowserBindingHash = append([]byte(nil), record.BrowserBindingHash...)
	record.SessionIDHash = append([]byte(nil), record.SessionIDHash...)
	record.BrowserContextHash = append([]byte(nil), record.BrowserContextHash...)
	record.Carry = append([]byte(nil), record.Carry...)
	record.Presentation.AllowedActions = append([]string(nil), record.Presentation.AllowedActions...)
	record.Presentation.PublicValues = append([]byte(nil), record.Presentation.PublicValues...)
	record.SecretReferences = append([]idpcontinuation.SecretReference(nil), record.SecretReferences...)
	record.EvidenceReferences = append([]idpcontinuation.EvidenceReference(nil), record.EvidenceReferences...)
	if record.Terminal != nil {
		terminal := *record.Terminal
		terminal.Data = append([]byte(nil), terminal.Data...)
		record.Terminal = &terminal
	}
	return record
}
