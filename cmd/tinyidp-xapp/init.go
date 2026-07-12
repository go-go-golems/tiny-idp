package main

import (
	"context"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type InitCommand struct {
	*cmds.CommandDescription
}

type InitSettings struct {
	StateRoot     string `glazed:"state-root"`
	PublicBaseURL string `glazed:"public-base-url"`
	Login         string `glazed:"login"`
	PasswordFile  string `glazed:"password-file"`
	Email         string `glazed:"email"`
	Name          string `glazed:"name"`
}

var _ cmds.BareCommand = (*InitCommand)(nil)

func NewInitCommand() (*InitCommand, error) {
	return &InitCommand{CommandDescription: cmds.NewCommandDescription(
		"init",
		cmds.WithShort("Initialize persistent identity and object security state"),
		cmds.WithLong(`Create or reconcile the single-node product state root.

The manifest is written only after SQLite migrations, owner-only secrets, the
exact public PKCE client, first password credential, and signing key exist.
Reruns preserve existing credentials and keys and reject conflicting identity
or public URL configuration.`),
		cmds.WithFlags(
			fields.New("state-root", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only product state directory")),
			fields.New("public-base-url", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Canonical browser-visible application origin")),
			fields.New("login", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initial administrator login")),
			fields.New("password-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing the initial password")),
			fields.New("email", fields.TypeString, fields.WithHelp("Initial verified email address")),
			fields.New("name", fields.TypeString, fields.WithHelp("Initial display name")),
		),
	)}, nil
}

func (c *InitCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings InitSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode init settings")
	}
	password, err := readOwnerOnlyPassword(settings.PasswordFile)
	if err != nil {
		return err
	}
	defer func() {
		for index := range password {
			password[index] = 0
		}
	}()
	manifest, err := InitializeState(ctx, InitializeStateConfig{
		StateRoot:     settings.StateRoot,
		PublicBaseURL: settings.PublicBaseURL,
		Login:         settings.Login,
		Password:      password,
		Email:         settings.Email,
		Name:          settings.Name,
	})
	if err != nil {
		return err
	}
	log.Info().Str("state_root", settings.StateRoot).Str("issuer", manifest.Issuer).Str("client_id", manifest.ClientID).Msg("tinyidp-xapp state initialized")
	return nil
}

func readOwnerOnlyPassword(file string) ([]byte, error) {
	info, err := os.Stat(file)
	if err != nil {
		return nil, errors.Wrap(err, "stat password file")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return nil, errors.Errorf("password file must be regular and mode 0600; got %s", info.Mode())
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "read password file")
	}
	if len(contents) > 0 && contents[len(contents)-1] == '\n' {
		contents = contents[:len(contents)-1]
		if len(contents) > 0 && contents[len(contents)-1] == '\r' {
			contents = contents[:len(contents)-1]
		}
	}
	if len(contents) == 0 {
		return nil, errors.New("password file is empty")
	}
	return contents, nil
}
