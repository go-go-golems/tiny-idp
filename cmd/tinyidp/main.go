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
	"context"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"

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
			AppName:           "tinyidp",                  // enables TINYIDP_* env loading
			ConfigPlanBuilder: cmds.ConfigFilePlanBuilder, // makes --config-file actually load
			// Replace the default parser chain with one that inserts profile
			// resolution at the right precedence layer (defaults < profiles <
			// config < env < args < flags). Without this, --profile is a flag
			// that never resolves a profiles.yaml.
			MiddlewaresFunc: cmds.ProfileMiddlewaresFunc("tinyidp", cmds.ConfigFilePlanBuilder),
		}),
		// Adds --profile / --profile-file (and TINYIDP_PROFILE /
		// TINYIDP_PROFILE_FILE). The MiddlewaresFunc above reads these and
		// loads ~/.config/tinyidp/profiles.yaml. See `tinyidp help reference`.
		cli.WithProfileSettingsSection(),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(serveCobraCmd)

	productionCmd, err := cmds.NewServeProductionCommand()
	cobra.CheckErr(err)
	productionCobraCmd, err := cli.BuildCobraCommand(productionCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(productionCobraCmd)

	// `tinyidp print-config` — print the resolved OIDC configuration. Composes
	// the same reusable oidc section as serve, so it is both a debugging tool
	// and the second consumer that proves the section is reusable.
	printConfigCmd, err := cmds.NewPrintConfigCommand()
	cobra.CheckErr(err)
	printConfigCobraCmd, err := cli.BuildCobraCommand(printConfigCmd,
		cli.WithParserConfig(cli.CobraParserConfig{
			AppName:           "tinyidp",
			ConfigPlanBuilder: cmds.ConfigFilePlanBuilder,
			MiddlewaresFunc:   cmds.ProfileMiddlewaresFunc("tinyidp", cmds.ConfigFilePlanBuilder),
		}),
		cli.WithProfileSettingsSection(),
	)
	cobra.CheckErr(err)
	rootCmd.AddCommand(printConfigCobraCmd)

	// `tinyidp admin` — operational user/password administration commands.
	rootCmd.AddCommand(cmds.NewAdminCommand())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cobra.CheckErr(rootCmd.ExecuteContext(ctx))
}
