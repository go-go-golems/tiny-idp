package idpcontinuation

import (
	"context"
	"errors"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

var (
	ErrNotFound        = errors.New("workflow continuation not found")
	ErrConflict        = errors.New("workflow continuation revision conflict")
	ErrAlreadyTerminal = errors.New("workflow continuation already terminal")
	ErrExpired         = errors.New("workflow continuation expired")
	ErrRevoked         = errors.New("workflow continuation revoked")
)

// Store provides atomic one-use transition operations. Advance must mark the
// current record advanced and insert next in one transaction.
type Store interface {
	Create(context.Context, WorkflowContinuation) error
	Load(context.Context, []byte, time.Time) (WorkflowContinuation, error)
	Advance(context.Context, []byte, uint64, WorkflowContinuation, time.Time) error
	Consume(context.Context, []byte, uint64, TerminalOutcome, time.Time) (WorkflowContinuation, error)
	Revoke(context.Context, []byte, uint64, time.Time) error
	ListExpired(context.Context, time.Time, int) ([]WorkflowContinuation, error)
	DeleteExpired(context.Context, []byte, time.Time) error
}

type GenerationResolver interface {
	ResolveProgram(context.Context, string) (idpprogram.Program, error)
}

type AttachmentCleaner interface {
	// DeleteContinuationAttachments must be idempotent. Cleanup may call it
	// again after a process failure between attachment deletion and record
	// deletion.
	DeleteContinuationAttachments(context.Context, []SecretReference, []EvidenceReference) error
}
