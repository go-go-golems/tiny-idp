// Package idpcontinuation owns restart-safe browser workflow state. Its values
// are pure Go data and must never contain Goja values, functions, Promises, or
// goroutine-local state.
package idpcontinuation

import (
	"encoding/json"
	"time"
)

const RecordVersionV1 uint32 = 1

type Status string

const (
	StatusActive   Status = "active"
	StatusAdvanced Status = "advanced"
	StatusConsumed Status = "consumed"
	StatusRevoked  Status = "revoked"
)

type TerminalKind string

const (
	TerminalComplete TerminalKind = "complete"
	TerminalDeny     TerminalKind = "deny"
	TerminalError    TerminalKind = "error"
)

type TerminalOutcome struct {
	Kind TerminalKind    `json:"kind"`
	Code string          `json:"code,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
	At   time.Time       `json:"at"`
}

type SecretReference struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type EvidenceReference struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type PresentationState struct {
	ID             string          `json:"id,omitempty"`
	AllowedActions []string        `json:"allowedActions,omitempty"`
	PublicValues   json.RawMessage `json:"publicValues,omitempty"`
}

// WorkflowContinuation is the complete durable state required to resume one
// browser workflow. HandleHash is keyed and domain-separated; a raw handle is
// never stored in this record.
type WorkflowContinuation struct {
	Version            uint32              `json:"version"`
	HandleHash         []byte              `json:"handleHash"`
	WorkflowID         string              `json:"workflowId"`
	ResumeHandlerID    string              `json:"resumeHandlerId"`
	ProgramFingerprint string              `json:"programFingerprint"`
	SchemaVersion      string              `json:"schemaVersion"`
	WorkflowVersion    uint32              `json:"workflowVersion"`
	RequestDigest      []byte              `json:"requestDigest"`
	ClientID           string              `json:"clientId"`
	RedirectURI        string              `json:"redirectUri"`
	ClientGeneration   string              `json:"clientGeneration"`
	BrowserBindingHash []byte              `json:"browserBindingHash"`
	SessionIDHash      []byte              `json:"sessionIdHash,omitempty"`
	BrowserContextHash []byte              `json:"browserContextHash,omitempty"`
	Presentation       PresentationState   `json:"presentation"`
	InputSchema        string              `json:"inputSchema"`
	Carry              json.RawMessage     `json:"carry"`
	SecretReferences   []SecretReference   `json:"secretReferences,omitempty"`
	EvidenceReferences []EvidenceReference `json:"evidenceReferences,omitempty"`
	Revision           uint64              `json:"revision"`
	CreatedAt          time.Time           `json:"createdAt"`
	ExpiresAt          time.Time           `json:"expiresAt"`
	Status             Status              `json:"status"`
	Terminal           *TerminalOutcome    `json:"terminal,omitempty"`
}

type Bindings struct {
	WorkflowID         string
	ClientID           string
	RedirectURI        string
	ClientGeneration   string
	ProgramFingerprint string
	RequestDigest      []byte
	BrowserBindingHash []byte
	SessionIDHash      []byte
	BrowserContextHash []byte
}
