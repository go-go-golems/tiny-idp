package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/oidcauth"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/loginui"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
	"github.com/pkg/errors"
)

func NewInitializedApplication(ctx context.Context, stateRoot string) (_ *DevelopmentApplication, retErr error) {
	manifest, err := ValidateInitializedState(stateRoot)
	if err != nil {
		return nil, errors.Wrap(err, "refuse uninitialized product state")
	}
	paths := ResolveStatePaths(stateRoot)
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(paths.IdentityDatabase))
	if err != nil {
		return nil, errors.Wrap(err, "open initialized identity database")
	}
	audit, err := idp.NewFileAuditSink(paths.AuditLog)
	if err != nil {
		_ = store.Close()
		return nil, errors.Wrap(err, "open initialized audit log")
	}
	app := &DevelopmentApplication{
		publicBaseURL: manifest.PublicBaseURL,
		extras: []func(context.Context) error{
			func(context.Context) error { return audit.Close() },
			func(context.Context) error { return store.Close() },
		},
	}
	defer func() {
		if retErr != nil {
			_ = app.Close(context.Background())
		}
	}()
	interactionUI, err := loginui.New(loginui.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "create production interaction renderer")
	}
	app.loginUI = interactionUI
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{Audit: audit})
	if err != nil {
		return nil, errors.Wrap(err, "create production password service")
	}
	tokenSecret, err := os.ReadFile(paths.TokenSecret)
	if err != nil {
		return nil, errors.Wrap(err, "read token secret")
	}
	defer func() {
		for index := range tokenSecret {
			tokenSecret[index] = 0
		}
	}()
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer:        manifest.Issuer,
		Mode:          embeddedidp.ProductionMode,
		Store:         store,
		Authenticator: accounts,
		Cookie: embeddedidp.CookieConfig{
			Secure:      true,
			SameSite:    http.SameSiteLaxMode,
			SessionName: "xapp_idp_session",
			CSRFName:    "xapp_idp_csrf",
		},
		Token:         embeddedidp.TokenConfig{SecretKey: tokenSecret},
		Audit:         audit,
		RateLimiter:   idp.NewFixedWindowRateLimiter(30, time.Minute),
		ClientAddress: idp.DirectClientAddressResolver{},
		UI:            embeddedidp.UIConfig{Renderer: interactionUI},
	})
	if err != nil {
		return nil, errors.Wrap(err, "construct production embedded IdP")
	}
	app.idp = provider
	if _, err := provider.RunMaintenance(ctx); err != nil {
		return nil, errors.Wrap(err, "run initial retention maintenance")
	}

	transport, err := oidcauth.NewInProcessIssuerTransport(manifest.Issuer, provider.Handler())
	if err != nil {
		return nil, errors.Wrap(err, "create production in-process issuer transport")
	}
	observedTransport := &observedRoundTripper{base: transport}
	app.oidc = observedTransport
	if err := os.MkdirAll(filepath.Dir(paths.AppAuthDatabase), 0o700); err != nil {
		return nil, errors.Wrap(err, "create application auth directory")
	}
	applySchema := true
	authFactory := hostauth.NewServiceFactory(hostauth.BuilderOptions{
		Config: hostauth.Config{
			Mode: hostauth.ModeOIDC,
			Session: hostauth.SessionConfig{Cookie: hostauth.CookieConfig{
				Name:     "xapp_session",
				Path:     "/",
				SameSite: "lax",
			}},
			Stores: hostauth.StoresConfig{Default: hostauth.StoreConfig{
				Driver:      string(hostauth.StoreDriverSQLite),
				DSN:         paths.AppAuthDatabase,
				ApplySchema: &applySchema,
			}},
			OIDC: hostauth.OIDCConfig{
				IssuerURL:      manifest.Issuer,
				ClientID:       manifest.ClientID,
				PublicBaseURL:  manifest.PublicBaseURL,
				Scopes:         []string{"profile", "email"},
				AfterLoginURL:  "/",
				AfterLogoutURL: "/",
			},
		},
		OIDCHTTPClient: &http.Client{Transport: observedTransport, Timeout: 10 * time.Second},
	})
	authServices, err := authFactory.BuildHostAuthServices(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build persistent application authentication")
	}
	app.auth = authServices
	if err := composeApplication(ctx, app, authFactory, stateRoot); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *DevelopmentApplication) Ready(ctx context.Context) error {
	if a == nil || a.idp == nil || a.auth == nil || a.objects == nil || a.runtime == nil || a.handler == nil {
		return errors.New("combined application is not fully constructed")
	}
	report := a.idp.Readiness(ctx)
	if !report.Ready {
		return errors.Errorf("embedded identity provider is not ready: %#v", report.Checks)
	}
	return nil
}
