// Command tinyidp is a mock OpenID Connect Identity Provider for local
// development and integration testing. It is NOT production grade (no real
// login, consent, persistent keys, refresh tokens, or TLS enforcement).
// Bind to loopback (the default) and never expose it publicly.
//
// The CLI is built on the Glazed command framework: the root command owns
// logging and help initialization, and child commands (currently `serve`)
// compose reusable field sections such as the `oidc` provider-config
// section. See `tinyidp help` for topics.
package main

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/cmd/tinyidp/doc"
	"github.com/manuel/tinyidp/internal/cmds"
)

// version is overridden at link time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "tinyidp",
		Short:   "tinyidp is a mock OIDC Identity Provider for local testing",
		Version: version,
		// PersistentPreRunE initializes structured logging from the logging
		// section flags (--log-level, --log-format, ...) added below.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	// Add the reusable Glazed logging section to the root so every child
	// command inherits --log-level / --log-format / --log-file / --verbose.
	if err := logging.AddLoggingSectionToRootCommand(rootCmd, "tinyidp"); err != nil {
		cobra.CheckErr(err)
	}

	// Load embedded help pages and wire `tinyidp help` / `tinyidp help <slug>`.
	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		cobra.CheckErr(err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	// `tinyidp serve` — run the mock IdP HTTP server.
	serveCmd, err := cmds.NewServeCommand()
	cobra.CheckErr(err)
	serveCobraCmd, err := cli.BuildCobraCommand(serveCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			AppName:          "tinyidp", // enables TINYIDP_* env loading on the default parser path
			ConfigPlanBuilder: cmds.ConfigFilePlanBuilder, // makes --config-file actually load
		}),
		// Profile-ready out of the box: adds --profile / --profile-file and
		// TINYIDP_PROFILE / TINYIDP_PROFILE_FILE. See `tinyidp help profiles`.
		cli.WithProfileSettingsSection(),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(serveCobraCmd)

	cobra.CheckErr(rootCmd.Execute())
}
