package cmds

import (
	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/internal/admin"
)

func newAdminBackupCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "backup", Short: "Create and verify SQLite backups"}
	cmd.AddCommand(newAdminBackupCreateCommand(dbPath))
	cmd.AddCommand(newAdminBackupVerifyCommand())
	return cmd
}

func newAdminBackupCreateCommand(dbPath *string) *cobra.Command {
	var out string
	cmd := &cobra.Command{Use: "create", Short: "Copy the SQLite database to a backup file", RunE: func(cmd *cobra.Command, _ []string) error {
		result, err := admin.CreateSQLiteBackup(cmd.Context(), *dbPath, out)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "created", "backup": result})
	}}
	cmd.Flags().StringVar(&out, "out", "", "Backup output path")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}

func newAdminBackupVerifyCommand() *cobra.Command {
	var path string
	cmd := &cobra.Command{Use: "verify", Short: "Open a backup database and run basic store checks", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := admin.VerifySQLiteBackup(cmd.Context(), path); err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "verified", "path": path})
	}}
	cmd.Flags().StringVar(&path, "path", "", "Backup database path")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}
