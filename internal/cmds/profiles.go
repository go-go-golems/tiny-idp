package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	cmd_sources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
)

// defaultProfileFile is the well-known location Glazed's profile middleware
// treats specially: if it is missing AND the requested profile is the default
// profile name, profile loading is skipped silently (rather than erroring).
// This is what lets `tinyidp serve` work out of the box with no profiles.yaml.
//
//	~/.config/tinyidp/profiles.yaml
//
// (Resolved from os.UserConfigDir, so $XDG_CONFIG_HOME on Linux.)
func defaultProfileFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tinyidp", "profiles.yaml"), nil
}

// ProfileMiddlewaresFunc returns a cli.CobraMiddlewaresFunc that builds the
// full source chain with profile resolution inserted at the correct precedence
// layer. Pass it to cli.WithParserConfig as MiddlewaresFunc.
//
// The chain it produces (reverse precedence; last applied wins):
//
//	flags (cobra) > args > env (TINYIDP_*) > config (--config-file) > profiles > defaults
//
// so the effective precedence (low to high) is:
//
//	defaults < profiles < config < env < args < flags
//
// This matches the precedence documented in `tinyidp help reference` and in
// the README. Profiles sit above defaults (a profile overrides the built-in
// defaults) but below config/env/flags (local overrides win), which is the
// Glazed-recommended placement for environment presets.
//
// Because profile selection itself can come from env (TINYIDP_PROFILE) or a
// config file, this function performs a bootstrap resolution of the
// profile-settings section (cobra + env) before constructing the profile
// middleware, exactly as the Glazed "implementing-profile-middleware" topic
// prescribes.
func ProfileMiddlewaresFunc(appName string, configPlanBuilder cli.ConfigPlanBuilder) cli.CobraMiddlewaresFunc {
	return func(parsedCommandSections *values.Values, cmd *cobra.Command, args []string) ([]cmd_sources.Middleware, error) {
		// 1. Bootstrap-resolve profile selection. The parser calls
		// ParseCommandSettingsSection (defaults + cobra) before us, populating
		// parsedCommandSections for the profile-settings section. That pass
		// does NOT apply env, so we honor TINYIDP_PROFILE / TINYIDP_PROFILE_FILE
		// explicitly here.
		profile, profileFile, err := resolveProfileSelection(appName, parsedCommandSections)
		if err != nil {
			return nil, err
		}

		defFile, err := defaultProfileFile()
		if err != nil {
			return nil, fmt.Errorf("resolve default profile file: %w", err)
		}

		// 2. Build the main chain (reverse precedence; last applied wins).
		var mws []cmd_sources.Middleware

		// flags (highest). Skip when there is no cobra command (test paths).
		if cmd != nil {
			mws = append(mws, cmd_sources.FromCobra(cmd, fields.WithSource("cobra")))
		}
		// positional args
		mws = append(mws, cmd_sources.FromArgs(args, fields.WithSource("arguments")))
		// env (TINYIDP_*)
		if appName != "" {
			mws = append(mws, cmd_sources.FromEnv(strings.ToUpper(appName), fields.WithSource("env")))
		}
		// config files (--config-file)
		if configPlanBuilder != nil {
			builder := configPlanBuilder // capture to avoid closure-loop warning
			mws = append(mws, cmd_sources.FromConfigPlanBuilder(
				func(ctx context.Context, vals *values.Values) (*config.Plan, error) {
					return builder(vals, cmd, args)
				},
				cmd_sources.WithParseOptions(fields.WithSource("config")),
			))
		}
		// profiles (above defaults, below config/env/flags)
		mws = append(mws, cmd_sources.GatherFlagsFromProfiles(
			defFile,
			profileFile,
			profile,
			"default",
			fields.WithSource("profiles"),
		))
		// defaults (lowest)
		mws = append(mws, cmd_sources.FromDefaults(fields.WithSource(fields.SourceDefaults)))

		return mws, nil
	}
}

// resolveProfileSelection reads --profile / --profile-file from the bootstrap
// parse, then applies env (TINYIDP_PROFILE / TINYIDP_PROFILE_FILE) on top so
// env selection works even though the bootstrap parse didn't run env. Falls
// back to the default profile file and "default" profile name when unset.
func resolveProfileSelection(appName string, parsedCommandSections *values.Values) (profile, profileFile string, err error) {
	ps := &cli.ProfileSettings{}
	if err := parsedCommandSections.DecodeSectionInto(cli.ProfileSettingsSlug, ps); err != nil {
		return "", "", fmt.Errorf("decode profile-settings: %w", err)
	}

	// Env overrides the bootstrap-resolved values (the bootstrap parse does
	// not apply env to profile-settings, so we honor it here explicitly).
	envPrefix := strings.ToUpper(appName)
	if v := os.Getenv(envPrefix + "_PROFILE_FILE"); v != "" {
		ps.ProfileFile = v
	}
	if v := os.Getenv(envPrefix + "_PROFILE"); v != "" {
		ps.Profile = v
	}

	defFile, err := defaultProfileFile()
	if err != nil {
		return "", "", err
	}
	if ps.ProfileFile == "" {
		ps.ProfileFile = defFile
	}
	if ps.Profile == "" {
		ps.Profile = "default"
	}
	return ps.Profile, ps.ProfileFile, nil
}
