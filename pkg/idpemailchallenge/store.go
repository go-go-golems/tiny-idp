package idpemailchallenge

import (
	"context"
	"time"
)

// Store exposes only lifecycle transitions. Implementations must make Verify
// and RecordAttempt atomic; callers cannot mutate a challenge record directly.
type Store interface {
	Create(context.Context, PendingChallenge) error
	Load(context.Context, string, time.Time) (PendingChallenge, error)
	Verify(context.Context, string, []byte, VerificationBindings, time.Time) (VerifiedEmailEvidence, error)
	RecordAttempt(context.Context, string, VerificationBindings, time.Time) (AttemptResult, error)
	ReserveResend(context.Context, string, VerificationBindings, time.Time) (ResendResult, error)
	ListExpired(context.Context, time.Time, int) ([]PendingChallenge, error)
	DeleteExpired(context.Context, string, time.Time) error
}
