package fositeadapter

import (
	"context"
	"sync"

	"github.com/manuel/tinyidp/internal/domain"
)

type ConsentPolicy interface {
	RequireConsent(ctx context.Context, user domain.User, client domain.Client, scopes []string) (bool, error)
	RecordConsent(ctx context.Context, user domain.User, client domain.Client, scopes []string) error
}

type AlwaysSkipConsent struct{}

func (AlwaysSkipConsent) RequireConsent(context.Context, domain.User, domain.Client, []string) (bool, error) {
	return false, nil
}
func (AlwaysSkipConsent) RecordConsent(context.Context, domain.User, domain.Client, []string) error {
	return nil
}

type RememberConsent struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewRememberConsent() *RememberConsent { return &RememberConsent{seen: map[string]struct{}{}} }
func (p *RememberConsent) RequireConsent(_ context.Context, user domain.User, client domain.Client, scopes []string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.seen[consentKey(user, client, scopes)]
	return !ok, nil
}
func (p *RememberConsent) RecordConsent(_ context.Context, user domain.User, client domain.Client, scopes []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen[consentKey(user, client, scopes)] = struct{}{}
	return nil
}
func consentKey(user domain.User, client domain.Client, scopes []string) string {
	key := user.ID + "\x00" + client.ID
	for _, s := range scopes {
		key += "\x00" + s
	}
	return key
}
