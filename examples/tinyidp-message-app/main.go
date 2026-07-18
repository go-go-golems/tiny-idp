// Command tinyidp-message-app runs the self-contained tiny-idp message-board
// example. It deliberately keeps browser-facing identity, relying-party, and
// application routes on one canonical origin.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     applicationName,
		Short:   "Self-contained tiny-idp SQLite message application",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}
	if err := logging.AddLoggingSectionToRootCommand(root, applicationName); err != nil {
		cobra.CheckErr(err)
	}
	for _, command := range []commandFactory{NewInitCommand, NewServeCommand, NewDoctorCommand} {
		glazedCommand, err := command()
		cobra.CheckErr(err)
		cobraCommand, err := cli.BuildCobraCommandFromCommand(glazedCommand,
			cli.WithParserConfig(cli.CobraParserConfig{AppName: applicationName, ShortHelpSections: []string{"default"}, MiddlewaresFunc: cli.CobraCommandDefaultMiddlewares}),
		)
		cobra.CheckErr(err)
		root.AddCommand(cobraCommand)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
