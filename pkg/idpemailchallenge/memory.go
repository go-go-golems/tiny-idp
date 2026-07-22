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

func (s *MemoryStore) CreateEmailChallenge(_ context.Context, c PendingChallenge) error {
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
func (s *MemoryStore) LoadEmailChallenge(_ context.Context, id string, now time.Time) (PendingChallenge, error) {
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
func (s *MemoryStore) VerifyEmailChallenge(_ context.Context, id string, codeHash []byte, b VerificationBindings, now time.Time) (VerifiedEmailEvidence, error) {
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
	return VerifiedEmailEvidence{Version: RecordVersionV1, ChallengeID: c.ID, Address: c.Email, Template: c.Template, Method: "email_code", VerifiedAt: at}, nil
}

func (s *MemoryStore) ConsumeVerifiedEmailChallenge(_ context.Context, id string, b VerificationBindings, now time.Time) (VerifiedEmailEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.records[id]
	if !ok {
		return VerifiedEmailEvidence{}, ErrNotFound
	}
	if !c.ExpiresAt.After(now) {
		return VerifiedEmailEvidence{}, ErrExpired
	}
	if err := c.VerifyEvidenceBindings(b); err != nil {
		return VerifiedEmailEvidence{}, err
	}
	if c.Status != StatusVerified || c.VerifiedAt == nil {
		return VerifiedEmailEvidence{}, ErrAlreadyTerminal
	}
	c.Status = StatusConsumed
	s.records[id] = c
	return VerifiedEmailEvidence{Version: RecordVersionV1, ChallengeID: c.ID, Address: c.Email, Template: c.Template, Method: "email_code", VerifiedAt: c.VerifiedAt.UTC()}, nil
}
func (s *MemoryStore) RecordEmailChallengeAttempt(_ context.Context, id string, b VerificationBindings, now time.Time) (AttemptResult, error) {
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
func (s *MemoryStore) ResendEmailChallenge(_ context.Context, id string, codeHash []byte, b VerificationBindings, now time.Time) (PendingChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.checked(id, b, now)
	if err != nil {
		return PendingChallenge{}, err
	}
	if len(codeHash) < 32 || c.Resends >= c.MaximumResends || now.Before(c.ResendNotBefore) {
		return PendingChallenge{}, ErrResendLimited
	}
	c.CodeHash = append([]byte(nil), codeHash...)
	// A resend starts a fresh code generation. A challenge that reached its
	// incorrect-code limit remains bound to the same browser and workflow, but
	// the newly delivered secret gets its own attempt budget. Without this
	// reset, the UI could offer a resend which could never be verified.
	c.Attempts = 0
	c.Resends++
	c.LastSentAt = now.UTC()
	s.records[id] = c
	return clone(c), nil
}
func (s *MemoryStore) ListExpiredEmailChallenges(_ context.Context, now time.Time, limit int) ([]PendingChallenge, error) {
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
func (s *MemoryStore) DeleteExpiredEmailChallenge(_ context.Context, id string, now time.Time) error {
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
