package idp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrAuditDelivery marks an operation whose state mutation committed but whose
// required audit record could not be durably delivered. Callers must not retry
// the mutation blindly; they should reconcile state and alert an operator.
var ErrAuditDelivery = errors.New("audit delivery failed after operation committed")

type Event struct {
	Time      time.Time         `json:"time"`
	Name      string            `json:"name"`
	ClientID  string            `json:"client_id,omitempty"`
	Subject   string            `json:"subject,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Result    string            `json:"result,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
}

type Sink interface {
	Emit(ctx context.Context, event Event) error
}

type AuditHealth struct {
	Ready     bool   `json:"ready"`
	Reason    string `json:"reason,omitempty"`
	Delivered uint64 `json:"delivered"`
	Failed    uint64 `json:"failed"`
	Dropped   uint64 `json:"dropped"`
	Policy    string `json:"policy"`
}

type AuditReporter interface {
	Sink
	ProductionReadyReporter
	AuditHealth(ctx context.Context) AuditHealth
}

type NoopSink struct{}

func (NoopSink) Emit(context.Context, Event) error { return nil }
func (NoopSink) ProductionReady() bool             { return false }
func (NoopSink) AuditHealth(context.Context) AuditHealth {
	return AuditHealth{Ready: false, Reason: "audit_disabled", Dropped: 1, Policy: "drop-all"}
}

var _ Sink = NoopSink{}

type MemorySink struct {
	mu     sync.Mutex
	events []Event
}

var _ Sink = (*MemorySink)(nil)

func NewMemorySink() *MemorySink { return &MemorySink{} }
func (s *MemorySink) Emit(_ context.Context, e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}
func (s *MemorySink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *MemorySink) ProductionReady() bool { return false }
func (s *MemorySink) AuditHealth(context.Context) AuditHealth {
	if s == nil {
		return AuditHealth{Ready: false, Reason: "audit_nil", Policy: "memory"}
	}
	s.mu.Lock()
	delivered := len(s.events)
	s.mu.Unlock()
	return AuditHealth{Ready: false, Reason: "audit_not_durable", Delivered: uint64(delivered), Policy: "memory"}
}

// FileAuditSink synchronously appends one JSON event and fsyncs it before Emit
// returns. It applies backpressure to callers, has no buffer, and never drops.
type FileAuditSink struct {
	mu        sync.Mutex
	file      *os.File
	delivered uint64
	failed    uint64
	lastError string
	closed    bool
}

var _ AuditReporter = (*FileAuditSink)(nil)

func NewFileAuditSink(path string) (*FileAuditSink, error) {
	if path == "" {
		return nil, fmt.Errorf("audit path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create audit directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("set audit file permissions: %w", err)
	}
	return &FileAuditSink{file: file}, nil
}

func (s *FileAuditSink) Emit(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.Name == "" {
		return fmt.Errorf("audit event name is required")
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.file == nil {
		s.failed++
		s.lastError = "audit_closed"
		return fmt.Errorf("audit sink is closed")
	}
	written, err := s.file.Write(encoded)
	if err != nil || written != len(encoded) {
		s.failed++
		s.lastError = "audit_write_failed"
		if err != nil {
			return fmt.Errorf("write audit event: %w", err)
		}
		return fmt.Errorf("write audit event: short write (%d of %d bytes)", written, len(encoded))
	}
	if err := s.file.Sync(); err != nil {
		s.failed++
		s.lastError = "audit_sync_failed"
		return fmt.Errorf("sync audit event: %w", err)
	}
	s.delivered++
	s.lastError = ""
	return nil
}

func (s *FileAuditSink) ProductionReady() bool { return s != nil }

func (s *FileAuditSink) AuditHealth(ctx context.Context) AuditHealth {
	if err := ctx.Err(); err != nil {
		return AuditHealth{Ready: false, Reason: "context_canceled", Policy: "synchronous-fsync"}
	}
	if s == nil {
		return AuditHealth{Ready: false, Reason: "audit_nil", Policy: "synchronous-fsync"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ready := !s.closed && s.file != nil && s.lastError == ""
	reason := s.lastError
	if s.closed {
		reason = "audit_closed"
	}
	return AuditHealth{Ready: ready, Reason: reason, Delivered: s.delivered, Failed: s.failed, Policy: "synchronous-fsync"}
}

func (s *FileAuditSink) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.file == nil {
		return nil
	}
	return s.file.Close()
}

func New(name string) Event { return Event{Time: time.Now().UTC(), Name: name} }
