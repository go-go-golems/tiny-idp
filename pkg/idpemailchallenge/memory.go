package idpemailchallenge

import (
	"bytes"
	"context"
	"sync"
	"time"
)

// MemoryStore is concurrency-safe and is intended for tests/development.
type MemoryStore struct {
	mu      sync.Mutex
	records map[string]PendingChallenge
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: map[string]PendingChallenge{}} }

var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) Create(_ context.Context, c PendingChallenge) error {
	if err := c.ValidateForCreate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[c.ID]; ok {
		return ErrConflict
	}
	s.records[c.ID] = clone(c)
	return nil
}
func (s *MemoryStore) Load(_ context.Context, id string, now time.Time) (PendingChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.records[id]
	if !ok {
		return PendingChallenge{}, ErrNotFound
	}
	if !c.ExpiresAt.After(now) {
		return PendingChallenge{}, ErrExpired
	}
	return clone(c), nil
}
func (s *MemoryStore) Verify(_ context.Context, id string, codeHash []byte, b VerificationBindings, now time.Time) (VerifiedEmailEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.checked(id, b, now)
	if err != nil {
		return VerifiedEmailEvidence{}, err
	}
	if c.Attempts >= c.MaximumAttempts {
		return VerifiedEmailEvidence{}, ErrAttemptsExceeded
	}
	if !bytes.Equal(c.CodeHash, codeHash) {
		c.Attempts++
		s.records[id] = c
		if c.Attempts >= c.MaximumAttempts {
			return VerifiedEmailEvidence{}, ErrAttemptsExceeded
		}
		return VerifiedEmailEvidence{}, ErrConflict
	}
	c.Status = StatusVerified
	at := now.UTC()
	c.VerifiedAt = &at
	s.records[id] = c
	return VerifiedEmailEvidence{Version: RecordVersionV1, ChallengeID: c.ID, Address: c.Email, Method: "email_code", VerifiedAt: at}, nil
}
func (s *MemoryStore) RecordAttempt(_ context.Context, id string, b VerificationBindings, now time.Time) (AttemptResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.checked(id, b, now)
	if err != nil {
		return AttemptResult{}, err
	}
	c.Attempts++
	s.records[id] = c
	return AttemptResult{RemainingAttempts: c.MaximumAttempts - c.Attempts, Terminal: c.Attempts >= c.MaximumAttempts}, nil
}
func (s *MemoryStore) ReserveResend(_ context.Context, id string, b VerificationBindings, now time.Time) (ResendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.checked(id, b, now)
	if err != nil {
		return ResendResult{}, err
	}
	if c.Resends >= c.MaximumResends || now.Before(c.ResendNotBefore) {
		return ResendResult{}, ErrResendLimited
	}
	c.Resends++
	c.LastSentAt = now.UTC()
	s.records[id] = c
	return ResendResult{Allowed: true, RemainingResends: c.MaximumResends - c.Resends, NotBefore: c.ResendNotBefore}, nil
}
func (s *MemoryStore) ListExpired(_ context.Context, now time.Time, limit int) ([]PendingChallenge, error) {
	if limit <= 0 {
		return nil, ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := []PendingChallenge{}
	for _, c := range s.records {
		if !c.ExpiresAt.After(now) && len(r) < limit {
			r = append(r, clone(c))
		}
	}
	return r, nil
}
func (s *MemoryStore) DeleteExpired(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.records[id]
	if !ok {
		return ErrNotFound
	}
	if c.ExpiresAt.After(now) {
		return ErrConflict
	}
	delete(s.records, id)
	return nil
}
func (s *MemoryStore) checked(id string, b VerificationBindings, now time.Time) (PendingChallenge, error) {
	c, ok := s.records[id]
	if !ok {
		return PendingChallenge{}, ErrNotFound
	}
	if !c.ExpiresAt.After(now) {
		return PendingChallenge{}, ErrExpired
	}
	if c.Status != StatusPending {
		return PendingChallenge{}, ErrAlreadyTerminal
	}
	if err := c.VerifyBindings(b); err != nil {
		return PendingChallenge{}, err
	}
	return c, nil
}
func clone(c PendingChallenge) PendingChallenge {
	c.CodeHash = append([]byte(nil), c.CodeHash...)
	c.BrowserBindingHash = append([]byte(nil), c.BrowserBindingHash...)
	if c.VerifiedAt != nil {
		v := *c.VerifiedAt
		c.VerifiedAt = &v
	}
	return c
}
