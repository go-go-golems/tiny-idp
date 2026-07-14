package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type Service struct {
	Store idpstore.Store
	Clock func() time.Time
	Audit idp.Sink
}

type Options struct {
	Clock func() time.Time
	Audit idp.Sink
}

// tinyidp:development-default -- production callers inject the provider audit sink.
func NewService(store idpstore.Store, opts Options) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	sink := opts.Audit
	if sink == nil {
		sink = idp.NoopSink{}
	}
	return &Service{Store: store, Clock: clock, Audit: sink}, nil
}

func (s *Service) auditCommitted(ctx context.Context, event idp.Event) error {
	if err := s.Audit.Emit(ctx, event); err != nil {
		return fmt.Errorf("%w: %v", idp.ErrAuditDelivery, err)
	}
	return nil
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
