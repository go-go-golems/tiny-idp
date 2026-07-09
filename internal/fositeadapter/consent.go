package fositeadapter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type AlwaysSkipConsent struct{}

var _ idp.ConsentPolicy = AlwaysSkipConsent{}

func (AlwaysSkipConsent) RequireConsent(context.Context, idpstore.User, idpstore.Client, []string) (bool, error) {
	return false, nil
}
func (AlwaysSkipConsent) RecordConsent(context.Context, idpstore.User, idpstore.Client, []string) error {
	return nil
}

type StoredConsent struct {
	store idpstore.ConsentStore
	ttl   time.Duration
}

var _ idp.ConsentPolicy = (*StoredConsent)(nil)

func NewStoredConsent(store idpstore.ConsentStore, ttl time.Duration) *StoredConsent {
	return &StoredConsent{store: store, ttl: ttl}
}

func (p *StoredConsent) RequireConsent(ctx context.Context, user idpstore.User, client idpstore.Client, scopes []string) (bool, error) {
	consent, err := p.store.GetConsent(ctx, user.ID, client.ID, scopes)
	if errors.Is(err, idpstore.ErrNotFound) {
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

func (p *StoredConsent) RecordConsent(ctx context.Context, user idpstore.User, client idpstore.Client, scopes []string) error {
	now := time.Now().UTC()
	consent := idpstore.Consent{UserID: user.ID, ClientID: client.ID, Scope: idpstore.NormalizeScopes(scopes), GrantedAt: now}
	if p.ttl > 0 {
		consent.ExpiresAt = now.Add(p.ttl)
	}
	return p.store.PutConsent(ctx, consent)
}

type RememberConsent struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

var _ idp.ConsentPolicy = (*RememberConsent)(nil)

func NewRememberConsent() *RememberConsent { return &RememberConsent{seen: map[string]struct{}{}} }
func (p *RememberConsent) RequireConsent(_ context.Context, user idpstore.User, client idpstore.Client, scopes []string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.seen[consentKey(user, client, scopes)]
	return !ok, nil
}
func (p *RememberConsent) RecordConsent(_ context.Context, user idpstore.User, client idpstore.Client, scopes []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen[consentKey(user, client, scopes)] = struct{}{}
	return nil
}
func consentKey(user idpstore.User, client idpstore.Client, scopes []string) string {
	key := user.ID + "\x00" + client.ID
	for _, s := range idpstore.NormalizeScopes(scopes) {
		key += "\x00" + s
	}
	return key
}
