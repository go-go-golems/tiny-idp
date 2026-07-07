package embeddedidp

import (
	"net/http"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/fositeadapter"
)

type Provider struct{ handler http.Handler }

func New(opts Options) (*Provider, error) {
	if opts.Mode == "" {
		opts.Mode = domain.DevMode
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	adapter, err := fositeadapter.NewProvider(fositeadapter.Options{Issuer: opts.Issuer, Store: opts.Store, SecretKey: opts.Token.SecretKey, Mode: opts.Mode, CookieSecure: opts.Cookie.Secure, Audit: opts.Audit, Consent: opts.Consent})
	if err != nil {
		return nil, err
	}
	return &Provider{handler: adapter.Handler()}, nil
}

func (p *Provider) Handler() http.Handler { return p.handler }
