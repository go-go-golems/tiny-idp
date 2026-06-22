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

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/rs/zerolog/log"

	"github.com/manuel/tinyidp/internal/sections/oidc"
	"github.com/manuel/tinyidp/internal/server"
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

Introspect the resolved configuration:
  tinyidp serve --print-parsed-fields

Use a named profile (requires profiles.yaml, see `+"`tinyidp help profiles`"+`):
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

	srv, err := server.New(server.Options{
		Issuer:       cfg.Issuer,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURIs: cfg.RedirectURIs,
	})
	if err != nil {
		return fmt.Errorf("build server: %w", err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Info().
		Str("addr", cfg.Addr).
		Str("issuer", srv.Issuer()).
		Str("client_id", srv.ClientID()).
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
