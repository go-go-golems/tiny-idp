// Package cmds holds the Glazed command implementations for the tinyidp CLI.
//
// Each verb is one file. serve.go implements `tinyidp serve`, which runs the
// mock OIDC IdP HTTP server. Future verbs (e.g. print-config, gen-key) would
// live alongside it as siblings.
package cmds

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"

	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/client"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/sections/oidc"
	"github.com/manuel/tinyidp/internal/server"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

// ServeCommand runs the mock OIDC IdP HTTP server. It implements
// cmds.BareCommand (Run returns an error, no row emission) because a
// long-running server has no tabular output.
type ServeCommand struct {
	*cmds.CommandDescription
}

// NewServeCommand builds the `serve` command, composing the reusable OIDC
// section (issuer/client/redirect config) and the Glazed command-settings
// section (--print-parsed-fields / --print-schema / --print-yaml for
// introspection). Profile support is enabled at the root via
// cli.WithProfileSettingsSection().
func NewServeCommand() (*ServeCommand, error) {
	oidcSection, err := oidc.NewSection()
	if err != nil {
		return nil, fmt.Errorf("build oidc section: %w", err)
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, fmt.Errorf("build command-settings section: %w", err)
	}

	cmdDesc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run the mock OIDC IdP HTTP server"),
		cmds.WithLong(`Run the mock OIDC Identity Provider.

This is a local development and integration testing tool, NOT production
grade. It binds to loopback (127.0.0.1:5556) by default; never expose it
publicly.

Configuration precedence (low to high): section defaults < profiles <
config files < environment variables (TINYIDP_*) < CLI flags.

Examples:
  tinyidp serve
  tinyidp serve --issuer http://localhost:5556 --client-id dev-client
  tinyidp serve --redirect-uris http://localhost:8080/callback
  tinyidp serve --client-secret dev-secret
  tinyidp serve --users-file ./users.yaml

Introspect the resolved configuration:
  tinyidp serve --print-parsed-fields

Use a named profile (requires profiles.yaml, see `+"`tinyidp help reference`"+`):
  tinyidp serve --profile dev
`),
		cmds.WithSections(oidcSection, commandSettingsSection),
	)
	return &ServeCommand{CommandDescription: cmdDesc}, nil
}

// Run starts the HTTP server and blocks until the context is cancelled
// (e.g. Ctrl+C) or the server stops. It is the BareCommand entry point.
func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
	cfg, err := oidc.GetSettings(vals)
	if err != nil {
		return err
	}

	registry, err := buildScenarioRegistry(cfg)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	clientRegistry := buildClientRegistry(cfg)
	clientCount := len(clientRegistry.All())
	engine := cfg.Engine
	if engine == "" {
		engine = "mock"
	}

	switch engine {
	case "mock":
		srv, err := server.New(server.Options{
			Issuer:   cfg.Issuer,
			Clients:  clientRegistry,
			Registry: registry,
		})
		if err != nil {
			return fmt.Errorf("build server: %w", err)
		}
		srv.RegisterRoutes(mux)
	case "fosite":
		strict, err := buildStrictProvider(cfg, clientRegistry, registry)
		if err != nil {
			return fmt.Errorf("build strict engine: %w", err)
		}
		mux.Handle("/", strict.Handler())
	default:
		return fmt.Errorf("unknown engine %q (want mock or fosite)", engine)
	}

	log.Info().
		Str("addr", cfg.Addr).
		Str("issuer", cfg.Issuer).
		Str("engine", engine).
		Int("clients", clientCount).
		Msg("tinyidp listening")

	errCh := make(chan error, 1)
	go func() {
		errCh <- http.ListenAndServe(cfg.Addr, server.WithCORS(mux))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
	}
	return nil
}

// buildClientRegistry returns a client registry seeded with the built-in
// clients (dev-client, public-spa, web-app) plus a client configured from
// the OIDC section's --client-id / --client-secret / --redirect-uris.
//
// If the configured client_id matches a builtin, the configured client is
// MERGED into the builtin: the builtin's RequirePKCE, Secret, and
// AllowedScopes are preserved, and the configured redirect URIs are added
// (deduplicated) to the builtin's. A non-empty configured --client-secret
// overrides the builtin's. So `--client-id public-spa --redirect-uris X`
// yields a public-spa client that still requires PKCE but now also accepts X.
//
// If the configured client_id does NOT match a builtin, a new permissive
// client is registered (the Phase 0-4 single-client behavior for custom IDs).
func buildClientRegistry(cfg *oidc.Settings) *client.Registry {
	r := client.NewRegistry()
	configured := client.Client{
		ID:           cfg.ClientID,
		Secret:       cfg.ClientSecret,
		RedirectURIs: cfg.RedirectURIs,
	}
	if base, ok := r.Lookup(cfg.ClientID); ok {
		// Merge: keep builtin properties, add configured redirect URIs.
		r.Register(client.Merge(base, configured))
	} else {
		// New permissive client (no secret, PKCE optional, all scopes).
		r.Register(configured)
	}
	for _, spec := range cfg.ExtraClients {
		if c, ok := parseExtraClientSpec(spec); ok {
			r.Register(c)
		}
	}
	return r
}

func parseExtraClientSpec(spec string) (client.Client, bool) {
	parts := strings.Split(spec, "|")
	if len(parts) < 3 || strings.TrimSpace(parts[0]) == "" {
		return client.Client{}, false
	}
	redirects := make([]string, 0, len(parts)-2)
	for _, redirect := range parts[2:] {
		redirect = strings.TrimSpace(redirect)
		if redirect != "" {
			redirects = append(redirects, redirect)
		}
	}
	if len(redirects) == 0 {
		return client.Client{}, false
	}
	return client.Client{ID: strings.TrimSpace(parts[0]), Secret: strings.TrimSpace(parts[1]), RedirectURIs: redirects}, true
}

func buildScenarioRegistry(cfg *oidc.Settings) (*scenario.Registry, error) {
	r := scenario.New()
	if cfg.UsersFile == "" {
		return r, nil
	}
	seeded, err := scenario.LoadSeededUsers(cfg.UsersFile)
	if err != nil {
		return nil, err
	}
	r.RegisterAll(seeded)
	return r, nil
}

var strictDevSecretKey = []byte("tinyidp-strict-dev-secret-key-32-bytes-min")

func buildStrictProvider(cfg *oidc.Settings, clients *client.Registry, scenarios *scenario.Registry) (*fositeadapter.Provider, error) {
	st := memory.New()
	plainClientSecrets := map[string]string{}
	for _, c := range clients.All() {
		dc := idpstore.Client{
			ID:                     c.ID,
			Public:                 c.Secret == "",
			RedirectURIs:           c.RedirectURIs,
			PostLogoutRedirectURIs: c.PostLogoutRedirectURIs,
			AllowedScopes:          c.AllowedScopes,
			RequirePKCE:            true,
			AccessTokenTTL:         time.Hour,
			IDTokenTTL:             time.Hour,
			RefreshTokenTTL:        24 * time.Hour,
		}
		if len(dc.AllowedScopes) == 0 {
			dc.AllowedScopes = []string{"openid", "profile", "email", "offline_access"}
		}
		if c.Secret != "" {
			dc.Public = false
			plainClientSecrets[c.ID] = c.Secret
		}
		if err := st.PutClient(context.Background(), dc); err != nil {
			return nil, err
		}
	}
	passwords, err := authn.NewPasswordService(st, authn.Options{Acceptance: idp.DevelopmentPasswordAcceptancePolicy()})
	if err != nil {
		return nil, err
	}
	for _, sc := range scenarios.All() {
		u := idpstore.User{ID: sc.User.Sub, Sub: sc.User.Sub, Email: sc.User.Email, Name: sc.User.Name, EmailVerified: true}
		applyScenarioClaims(&u, sc.ExtraClaims)
		if err := st.PutUser(context.Background(), sc.Name, u); err != nil {
			return nil, err
		}
		if sc.Password != "" {
			credential, err := passwords.HashCredential(context.Background(), u.ID, sc.Name, []byte(sc.Password), time.Now().UTC())
			if err != nil {
				return nil, err
			}
			if err := st.PutPasswordCredential(context.Background(), credential); err != nil {
				return nil, err
			}
		}
	}
	key, err := keys.GenerateRSA("strict-dev-key-1", time.Now())
	if err != nil {
		return nil, err
	}
	if err := st.CreateSigningKey(context.Background(), key); err != nil {
		return nil, err
	}
	return fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: cfg.Issuer, Store: st, SecretKey: strictDevSecretKey, Mode: idpstore.DevMode, ClientSecrets: plainClientSecrets})
}

func applyScenarioClaims(u *idpstore.User, claims map[string]any) {
	for k, v := range claims {
		switch k {
		case "email_verified":
			if b, ok := v.(bool); ok {
				u.EmailVerified = b
			}
		case "preferred_username":
			if s, ok := v.(string); ok {
				u.PreferredUsername = s
			}
		case "tenant":
			if s, ok := v.(string); ok {
				u.Tenant = s
			}
		case "locale":
			if s, ok := v.(string); ok {
				u.Locale = s
			}
		case "groups":
			u.Groups = stringSlice(v)
		case "roles":
			u.Roles = stringSlice(v)
		case "name":
			if s, ok := v.(string); ok {
				u.Name = s
			}
		}
	}
}

func stringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
