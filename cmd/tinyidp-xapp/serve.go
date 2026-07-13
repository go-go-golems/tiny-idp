package main

import (
	"context"
	stderrors "errors"
	"net"
	"net/http"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	Listen         string `glazed:"listen"`
	PublicBaseURL  string `glazed:"public-base-url"`
	StateRoot      string `glazed:"state-root"`
	Login          string `glazed:"login"`
	Password       string `glazed:"password"`
	SecondLogin    string `glazed:"second-login"`
	SecondPassword string `glazed:"second-password"`
}

var _ cmds.BareCommand = (*ServeCommand)(nil)

func NewServeCommand() (*ServeCommand, error) {
	return &ServeCommand{CommandDescription: cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Run the self-contained development identity/object application"),
		cmds.WithLong(`Start the development vertical slice with an embedded tiny-idp,
in-process OIDC back channel, application session, trusted xgoja routes, and an
actor-bound SQLite Durable Object. This command is development-only; production
initialization and persistent identity/auth stores are added in later phases.`),
		cmds.WithFlags(
			fields.New("listen", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("TCP listen address")),
			fields.New("public-base-url", fields.TypeString, fields.WithDefault("http://127.0.0.1:8787"), fields.WithHelp("Browser-visible application origin")),
			fields.New("state-root", fields.TypeString, fields.WithDefault("./var/tinyidp-xapp-dev"), fields.WithHelp("Development object and secret state root")),
			fields.New("login", fields.TypeString, fields.WithDefault("alice"), fields.WithHelp("Seed development login")),
			fields.New("password", fields.TypeString, fields.WithDefault("correct horse battery staple"), fields.WithHelp("Seed development password; do not use this command for production")),
			fields.New("second-login", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional second development login for account-isolation testing")),
			fields.New("second-password", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional second development password; do not use this command for production")),
		),
	)}, nil
}

func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings ServeSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode serve settings")
	}
	application, err := NewDevelopmentApplication(ctx, DevelopmentApplicationConfig{
		PublicBaseURL:  settings.PublicBaseURL,
		StateRoot:      settings.StateRoot,
		Login:          settings.Login,
		Password:       settings.Password,
		SecondLogin:    settings.SecondLogin,
		SecondPassword: settings.SecondPassword,
	})
	if err != nil {
		return err
	}
	defer func() { _ = application.Close(context.Background()) }()

	listener, err := net.Listen("tcp", settings.Listen)
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	server := &http.Server{
		Addr:              settings.Listen,
		Handler:           application.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Info().Str("listen", settings.Listen).Str("public_base_url", settings.PublicBaseURL).Msg("tinyidp-xapp development server started")

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err := server.Serve(listener)
		if err != nil && !stderrors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "serve HTTP")
		}
		return nil
	})
	group.Go(func() error {
		<-groupCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Wrap(err, "shutdown HTTP server")
		}
		return nil
	})
	return group.Wait()
}
