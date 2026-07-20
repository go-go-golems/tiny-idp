package idpemailchallenge

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/pkg/errors"
)

const codeDomain = "tiny-idp/email-challenge/v1\x00"

// Mailer is the only delivery authority accepted by the challenge service.
// It receives a challenge reference and approved template fields, never an
// SMTP client, arbitrary recipient list, arbitrary subject, or raw state.
type Mailer interface {
	SendEmailChallenge(context.Context, MailRequest) error
}
type MailRequest struct {
	Challenge Reference
	Recipient string
	Template  string
	Code      string
	ExpiresAt time.Time
}
type RetryClass string

const (
	RetryNone      RetryClass = "none"
	RetryTransient RetryClass = "transient"
	RetryPermanent RetryClass = "permanent"
)

type MailFailure interface {
	error
	RetryClass() RetryClass
}

type CreateRequest struct {
	ID, Email, Template             string
	Bindings                        VerificationBindings
	ExpiresAt                       time.Time
	MaximumAttempts, MaximumResends uint32
	ResendNotBefore                 time.Time
}
type Service struct {
	store   Store
	mailer  Mailer
	key     []byte
	now     func() time.Time
	newCode func() (string, error)
}

func NewService(store Store, mailer Mailer, key []byte) (*Service, error) {
	if store == nil || mailer == nil || len(key) < 32 {
		return nil, errors.New("email challenge store, mailer, and 32-byte key are required")
	}
	return &Service{store: store, mailer: mailer, key: append([]byte(nil), key...), now: func() time.Time { return time.Now().UTC() }, newCode: randomCode}, nil
}
func (s *Service) CreateAndSend(ctx context.Context, r CreateRequest) (Reference, error) {
	if s == nil {
		return Reference{}, errors.New("email challenge service unavailable")
	}
	now := s.now()
	code, err := s.newCode()
	if err != nil {
		return Reference{}, errors.Wrap(err, "generate email challenge code")
	}
	c := PendingChallenge{Version: 1, ID: r.ID, CodeHash: s.hash(code), Email: r.Email, Template: r.Template, WorkflowID: r.Bindings.WorkflowID, ResumeHandlerID: r.Bindings.ResumeHandlerID, ProgramFingerprint: r.Bindings.ProgramFingerprint, ClientID: r.Bindings.ClientID, ClientGeneration: r.Bindings.ClientGeneration, BrowserBindingHash: append([]byte(nil), r.Bindings.BrowserBindingHash...), CreatedAt: now, ExpiresAt: r.ExpiresAt, LastSentAt: now, MaximumAttempts: r.MaximumAttempts, MaximumResends: r.MaximumResends, ResendNotBefore: r.ResendNotBefore, Status: StatusPending}
	if err := s.store.CreateEmailChallenge(ctx, c); err != nil {
		return Reference{}, err
	}
	if err := s.mailer.SendEmailChallenge(ctx, MailRequest{Challenge: c.Reference(), Recipient: c.Email, Template: c.Template, Code: code, ExpiresAt: c.ExpiresAt}); err != nil {
		return Reference{}, err
	}
	return c.Reference(), nil
}
func (s *Service) Verify(ctx context.Context, ref Reference, code string, b VerificationBindings) (VerifiedEmailEvidence, error) {
	if s == nil || ref.Version != RecordVersionV1 || !valid(code) {
		return VerifiedEmailEvidence{}, ErrConflict
	}
	return s.store.VerifyEmailChallenge(ctx, ref.ID, s.hash(code), b, s.now())
}

// Resend rotates the code through one atomic native store transition before
// delivering it through the typed mailer. The browser supplies only the
// durable reference; it cannot choose a recipient, message, or code.
func (s *Service) Resend(ctx context.Context, ref Reference, b VerificationBindings) error {
	if s == nil || ref.Version != RecordVersionV1 || !valid(ref.ID) {
		return ErrConflict
	}
	code, err := s.newCode()
	if err != nil {
		return errors.Wrap(err, "generate email challenge code")
	}
	c, err := s.store.ResendEmailChallenge(ctx, ref.ID, s.hash(code), b, s.now())
	if err != nil {
		return err
	}
	if err := s.mailer.SendEmailChallenge(ctx, MailRequest{Challenge: c.Reference(), Recipient: c.Email, Template: c.Template, Code: code, ExpiresAt: c.ExpiresAt}); err != nil {
		return errors.Wrap(err, "deliver resent email challenge")
	}
	return nil
}

// Evidence rehydrates evidence that was previously created by Verify. It is
// deliberately a native operation: a continuation may retain only a typed
// challenge reference, never a script-created "verified" marker or email
// address. The original binding and expiry are checked again on every use.
func (s *Service) Evidence(ctx context.Context, ref Reference, b VerificationBindings) (VerifiedEmailEvidence, error) {
	if s == nil || ref.Version != RecordVersionV1 || !valid(ref.ID) {
		return VerifiedEmailEvidence{}, ErrConflict
	}
	c, err := s.store.LoadEmailChallenge(ctx, ref.ID, s.now())
	if err != nil {
		return VerifiedEmailEvidence{}, err
	}
	if err := c.VerifyEvidenceBindings(b); err != nil {
		return VerifiedEmailEvidence{}, err
	}
	if c.Status != StatusVerified || c.VerifiedAt == nil {
		return VerifiedEmailEvidence{}, ErrAlreadyTerminal
	}
	return VerifiedEmailEvidence{Version: RecordVersionV1, ChallengeID: c.ID, Address: c.Email, Template: c.Template, Method: "email_code", VerifiedAt: c.VerifiedAt.UTC()}, nil
}

// ConsumeEvidence performs the one-time native transition from verified
// evidence to a consumed terminal record. It is deliberately separate from
// Evidence: ordinary signup continuations can rehydrate verified evidence
// while progressing through their declared presentation steps, whereas a
// credential-recovery effect must be replay-proof.
func (s *Service) ConsumeEvidence(ctx context.Context, ref Reference, b VerificationBindings) (VerifiedEmailEvidence, error) {
	if s == nil || ref.Version != RecordVersionV1 || !valid(ref.ID) {
		return VerifiedEmailEvidence{}, ErrConflict
	}
	return s.store.ConsumeVerifiedEmailChallenge(ctx, ref.ID, b, s.now())
}
func (s *Service) hash(code string) []byte {
	m := hmac.New(sha256.New, s.key)
	_, _ = m.Write([]byte(codeDomain))
	_, _ = m.Write([]byte(code))
	return m.Sum(nil)
}
func randomCode() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}
