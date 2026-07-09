package idp

import (
	"context"
	"sync"
	"time"
)

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

type NoopSink struct{}

func (NoopSink) Emit(context.Context, Event) error { return nil }

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

func New(name string) Event { return Event{Time: time.Now().UTC(), Name: name} }
