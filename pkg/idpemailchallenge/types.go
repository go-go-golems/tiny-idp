// Package idpemailchallenge defines the native, restart-safe contracts for
// email-code verification. Scripts receive references and verified evidence,
// never a raw code, mail transport, or mutable challenge record.
package idpemailchallenge

import (
	"strings"
	"time"

	"github.com/pkg/errors"
)

const RecordVersionV1 uint32 = 1

type Status string

const (
	StatusPending  Status = "pending"
	StatusVerified Status = "verified"
	StatusRevoked  Status = "revoked"
)

// Reference is safe to carry into a browser continuation or mailer request.
// It deliberately omits the submitted code and its lookup hash.
type Reference struct {
	ID      string `json:"id"`
	Version uint32 `json:"version"`
}

// PendingChallenge is durable native state. CodeHash is a domain-separated
// keyed hash. Every binding is rechecked before verification or resend.
type PendingChallenge struct {
	Version            uint32
	ID                 string
	CodeHash           []byte
	Email              string
	Template           string
	WorkflowID         string
	ResumeHandlerID    string
	ProgramFingerprint string
	ClientID           string
	ClientGeneration   string
	BrowserBindingHash []byte
	CreatedAt          time.Time
	ExpiresAt          time.Time
	LastSentAt         time.Time
	Attempts           uint32
	MaximumAttempts    uint32
	Resends            uint32
	MaximumResends     uint32
	ResendNotBefore    time.Time
	Status             Status
	VerifiedAt         *time.Time
}

// VerificationBindings bind the browser request to the exact challenge and
// workflow generation that issued it.
type VerificationBindings struct {
	WorkflowID, ResumeHandlerID, ProgramFingerprint, ClientID, ClientGeneration string
	BrowserBindingHash                                                          []byte
}

// VerifiedEmailEvidence is created only by a successful native consume. It is
// the unforgeable value later projected into a resumed lambda.
type VerifiedEmailEvidence struct {
	Version     uint32
	ChallengeID string
	Address     string
	Method      string
	VerifiedAt  time.Time
}

type AttemptResult struct {
	Accepted          bool
	RemainingAttempts uint32
	Terminal          bool
}
type ResendResult struct {
	Allowed          bool
	RemainingResends uint32
	NotBefore        time.Time
}

var (
	ErrNotFound         = errors.New("email challenge not found")
	ErrConflict         = errors.New("email challenge conflict")
	ErrExpired          = errors.New("email challenge expired")
	ErrAlreadyTerminal  = errors.New("email challenge already terminal")
	ErrAttemptsExceeded = errors.New("email challenge attempts exceeded")
	ErrResendLimited    = errors.New("email challenge resend limited")
	ErrBinding          = errors.New("email challenge binding rejected")
)

func (c PendingChallenge) Reference() Reference { return Reference{ID: c.ID, Version: c.Version} }

func (c PendingChallenge) ValidateForCreate() error {
	if c.Version != RecordVersionV1 || !valid(c.ID) || len(c.CodeHash) < 32 || !valid(c.Email) || !strings.Contains(c.Email, "@") || !valid(c.Template) || !valid(c.WorkflowID) || !valid(c.ResumeHandlerID) || !valid(c.ProgramFingerprint) || !valid(c.ClientID) || !valid(c.ClientGeneration) || len(c.BrowserBindingHash) == 0 || c.CreatedAt.IsZero() || c.ExpiresAt.IsZero() || !c.ExpiresAt.After(c.CreatedAt) || c.MaximumAttempts == 0 || c.MaximumResends == 0 || c.Status != StatusPending || c.VerifiedAt != nil {
		return errors.New("email challenge create contract is invalid")
	}
	return nil
}

func (c PendingChallenge) VerifyBindings(bindings VerificationBindings) error {
	if c.WorkflowID != bindings.WorkflowID || c.ResumeHandlerID != bindings.ResumeHandlerID || c.ProgramFingerprint != bindings.ProgramFingerprint || c.ClientID != bindings.ClientID || c.ClientGeneration != bindings.ClientGeneration || string(c.BrowserBindingHash) != string(bindings.BrowserBindingHash) {
		return ErrBinding
	}
	return nil
}

func valid(value string) bool { return strings.TrimSpace(value) != "" && len(value) <= 512 }
