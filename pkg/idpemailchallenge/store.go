package idpemailchallenge

import (
	"context"
	"time"
)

// Store exposes only lifecycle transitions. Implementations must make Verify
// and RecordAttempt atomic; callers cannot mutate a challenge record directly.
type Store interface {
	CreateEmailChallenge(context.Context, PendingChallenge) error
	LoadEmailChallenge(context.Context, string, time.Time) (PendingChallenge, error)
	VerifyEmailChallenge(context.Context, string, []byte, VerificationBindings, time.Time) (VerifiedEmailEvidence, error)
	RecordEmailChallengeAttempt(context.Context, string, VerificationBindings, time.Time) (AttemptResult, error)
	ResendEmailChallenge(context.Context, string, []byte, VerificationBindings, time.Time) (PendingChallenge, error)
	ListExpiredEmailChallenges(context.Context, time.Time, int) ([]PendingChallenge, error)
	DeleteExpiredEmailChallenge(context.Context, string, time.Time) error
}
