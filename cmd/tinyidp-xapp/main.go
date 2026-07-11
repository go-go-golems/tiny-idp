// Command tinyidp-xapp is the custom lifecycle host for the self-contained
// tiny-idp, xgoja Express, and actor-bound Durable Objects application.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "tinyidp-xapp",
		Short:   "Self-contained identity and private Durable Object application",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	if err := logging.AddLoggingSectionToRootCommand(root, "tinyidp-xapp"); err != nil {
		cobra.CheckErr(err)
	}
	help_cmd.SetupCobraRootCommand(help.NewHelpSystem(), root)

	doctor, err := NewDoctorCommand()
	cobra.CheckErr(err)
	doctorCobra, err := cli.BuildCobraCommandFromCommand(doctor,
		cli.WithParserConfig(cli.CobraParserConfig{
			AppName:           "tinyidp-xapp",
			ShortHelpSections: []string{"default"},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	cobra.CheckErr(err)
	root.AddCommand(doctorCobra)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
