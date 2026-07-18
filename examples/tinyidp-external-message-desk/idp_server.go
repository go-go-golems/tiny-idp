package externalmessagedesk

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

// StandaloneIDPConfig names only the capabilities a standalone IdP process
// needs. Applications never receive the resulting Store or account service.
type StandaloneIDPConfig struct {
	Issuer       string
	Mode         embeddedidp.Mode
	Store        idpstore.Store
	Accounts     *idpaccounts.Service
	Seed         SeedManifest
	TokenSecret  []byte
	CookieSecure bool
	Audit        idp.Sink
	Renderer     idpui.InteractionRenderer
}

func NewStandaloneIDP(ctx context.Context, cfg StandaloneIDPConfig) (*embeddedidp.Provider, error) {
	issuer, err := normalizeIssuer(cfg.Issuer)
	if err != nil {
		return nil, err
	}
	if cfg.Store == nil || cfg.Accounts == nil || len(cfg.TokenSecret) < 32 {
		return nil, errors.New("standalone IdP store, account service, and 32-byte token secret are required")
	}
	if err := cfg.Seed.Bootstrap(ctx, cfg.Store, cfg.Accounts, cfg.Mode); err != nil {
		return nil, err
	}
	renderer := cfg.Renderer
	if renderer == nil {
		renderer, err = idpui.NewDefaultRenderer()
		if err != nil {
			return nil, err
		}
	}
	return embeddedidp.New(ctx, embeddedidp.Options{
		Issuer: issuer, Mode: cfg.Mode, Store: cfg.Store,
		Cookie: embeddedidp.CookieConfig{Secure: cfg.CookieSecure, SameSite: http.SameSiteLaxMode, Path: "/"},
		Token:  embeddedidp.TokenConfig{SecretKey: cfg.TokenSecret}, Audit: cfg.Audit,
		RateLimiter: idp.NewFixedWindowRateLimiter(30, time.Minute), ClientAddress: idp.DirectClientAddressResolver{},
		Authenticator: cfg.Accounts, UI: embeddedidp.UIConfig{Renderer: renderer},
		AccountChooser: embeddedidp.AccountChooserConfig{Enabled: true, RememberOnPasswordLogin: true, DisplayLabel: func(user idpstore.User) (string, error) {
			if name := strings.TrimSpace(user.Name); name != "" {
				return name, nil
			}
			return user.PreferredUsername, nil
		}},
	})
}

func normalizeIssuer(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("issuer must be an absolute HTTP(S) URL without query, fragment, or credentials")
	}
	if parsed.Scheme == "http" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return "", errors.New("plain HTTP issuer is allowed only for local development")
	}
	path := strings.TrimSuffix(parsed.EscapedPath(), "/")
	if strings.Contains(path, "//") {
		return "", errors.New("issuer path must be canonical")
	}
	return parsed.Scheme + "://" + parsed.Host + path, nil
}
