package embeddedidp

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type Provider struct {
	handler http.Handler
	store   idpstore.Store
	closed  atomic.Bool
}

func New(ctx context.Context, opts Options) (*Provider, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if opts.Mode == "" {
		opts.Mode = idpstore.DevMode
	}
	if err := opts.Validate(ctx); err != nil {
		return nil, err
	}
	adapter, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: opts.Issuer, Store: opts.Store, SecretKey: opts.Token.SecretKey, Mode: opts.Mode, CookieSecure: opts.Cookie.Secure, Audit: opts.Audit, Consent: opts.Consent, RateLimiter: opts.RateLimiter, Authenticator: opts.Authenticator})
	if err != nil {
		return nil, err
	}
	return &Provider{handler: adapter.Handler(), store: opts.Store}, nil
}

func (p *Provider) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.closed.Load() {
			http.Error(w, "provider closed", http.StatusServiceUnavailable)
			return
		}
		p.handler.ServeHTTP(w, r)
	})
}

func (p *Provider) Readiness(ctx context.Context) idp.ReadinessReport {
	now := time.Now().UTC()
	report := idp.ReadinessReport{Ready: true}
	add := func(name, reason string, err error) {
		check := idp.ReadinessCheck{Name: name, Ready: err == nil, CheckedAt: now}
		if err != nil {
			check.Reason = reason
			report.Ready = false
		}
		report.Checks = append(report.Checks, check)
	}
	if ctx == nil {
		add("context", "context_required", fmt.Errorf("context is required"))
		return report
	}
	if p.closed.Load() {
		add("lifecycle", "provider_closed", fmt.Errorf("provider is closed"))
		return report
	}
	add("lifecycle", "", nil)
	_, err := p.store.ListClients(ctx)
	add("store", "store_unavailable", err)
	_, err = p.store.ActiveSigningKey(ctx)
	add("signing_key", "signing_key_unavailable", err)
	return report
}

func (p *Provider) Close(_ context.Context) error {
	p.closed.Store(true)
	return nil
}
