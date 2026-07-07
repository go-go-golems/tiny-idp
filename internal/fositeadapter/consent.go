package fositeadapter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/storage"
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

type StoredConsent struct {
	store storage.ConsentStore
	ttl   time.Duration
}

func NewStoredConsent(store storage.ConsentStore, ttl time.Duration) *StoredConsent {
	return &StoredConsent{store: store, ttl: ttl}
}

func (p *StoredConsent) RequireConsent(ctx context.Context, user domain.User, client domain.Client, scopes []string) (bool, error) {
	consent, err := p.store.GetConsent(ctx, user.ID, client.ID, scopes)
	if errors.Is(err, storage.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	if consent.RevokedAt != nil {
		return true, nil
	}
	if !consent.ExpiresAt.IsZero() && now.After(consent.ExpiresAt) {
		return true, nil
	}
	return false, nil
}

func (p *StoredConsent) RecordConsent(ctx context.Context, user domain.User, client domain.Client, scopes []string) error {
	now := time.Now().UTC()
	consent := domain.Consent{UserID: user.ID, ClientID: client.ID, Scope: domain.NormalizeScopes(scopes), GrantedAt: now}
	if p.ttl > 0 {
		consent.ExpiresAt = now.Add(p.ttl)
	}
	return p.store.PutConsent(ctx, consent)
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
	for _, s := range domain.NormalizeScopes(scopes) {
		key += "\x00" + s
	}
	return key
}
