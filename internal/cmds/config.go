package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/config"
	"github.com/spf13/cobra"
)

// ConfigFilePlanBuilder is a cli.ConfigPlanBuilder that loads the file named
// by the --config-file flag (added to every command by the Glazed
// command-settings section). Without a builder, --config-file is a no-op
// flag; with it, the file is loaded as an explicit config layer.
//
// Precedence (low to high, set by the built-in parser chain when both
// AppName and this builder are configured on the CobraParserConfig):
//
//	defaults < config files (--config-file) < env (TINYIDP_*) < args < flags
//
// Returning an empty plan (no sources) when --config-file is unset is a
// clean no-op: no files are loaded, no other layers are touched.
func ConfigFilePlanBuilder(_ *values.Values, cmd *cobra.Command, _ []string) (*config.Plan, error) {
	cfgFile, err := cmd.Flags().GetString("config-file")
	if err != nil || cfgFile == "" {
		return config.NewPlan(config.WithLayerOrder(config.LayerExplicit)), nil
	}
	return config.NewPlan(
		config.WithLayerOrder(config.LayerExplicit),
	).Add(
		config.ExplicitFile(cfgFile).Named("config-file"),
	), nil
}
