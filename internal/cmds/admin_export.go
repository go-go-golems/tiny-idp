package cmds

import (
	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/internal/admin"
)

func newAdminExportCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "export", Short: "Export sanitized diagnostics"}
	cmd.AddCommand(newAdminExportDiagnosticsCommand(dbPath))
	return cmd
}

func newAdminExportDiagnosticsCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{Use: "diagnostics", Short: "Export redacted admin diagnostics", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		clients, err := svc.ListClients(cmd.Context())
		if err != nil {
			return err
		}
		clientOut := make([]any, 0, len(clients))
		for _, c := range clients {
			clientOut = append(clientOut, redactClient(c))
		}
		keys, err := svc.ListSigningKeys(cmd.Context())
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{
			"doctor":  svc.Doctor(cmd.Context()),
			"clients": clientOut,
			"keys":    admin.RedactSigningKeys(keys),
		})
	}}
}
