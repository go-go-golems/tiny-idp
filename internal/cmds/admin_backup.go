package cmds

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/go-go-golems/tiny-idp/internal/admin"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

func newAdminBackupCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "backup", Short: "Create, verify, and restore SQLite online backups"}
	cmd.AddCommand(newAdminBackupCreateCommand(dbPath))
	cmd.AddCommand(newAdminBackupVerifyCommand())
	cmd.AddCommand(newAdminBackupRestoreCommand(dbPath))
	return cmd
}

func newAdminBackupRestoreCommand(dbPath *string) *cobra.Command {
	var path string
	cmd := &cobra.Command{Use: "restore", Short: "Verify and atomically restore a stopped SQLite database", RunE: func(cmd *cobra.Command, _ []string) error {
		result, err := admin.RestoreSQLiteBackup(cmd.Context(), path, *dbPath)
		if err != nil {
			return err
		}
		if err := emitAdminAudit(cmd.Context(), *dbPath, idp.Event{Time: time.Now().UTC(), Name: "admin.backup.restored", Result: "accepted", Fields: map[string]string{"backup_path": path, "rollback_path": result.RollbackPath}}); err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "restored", "restore": result})
	}}
	cmd.Flags().StringVar(&path, "path", "", "Verified backup database path")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

func newAdminBackupCreateCommand(dbPath *string) *cobra.Command {
	var out string
	cmd := &cobra.Command{Use: "create", Short: "Create and atomically publish a verified SQLite online backup", RunE: func(cmd *cobra.Command, _ []string) error {
		result, err := admin.CreateSQLiteBackup(cmd.Context(), *dbPath, out)
		if err != nil {
			return err
		}
		if err := emitAdminAudit(cmd.Context(), *dbPath, idp.Event{Time: time.Now().UTC(), Name: "admin.backup.created", Result: "accepted", Fields: map[string]string{"backup_path": result.Path}}); err != nil {
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
	cmd := &cobra.Command{Use: "verify", Short: "Verify a backup read-only without migrating it", RunE: func(cmd *cobra.Command, _ []string) error {
		if err := admin.VerifySQLiteBackup(cmd.Context(), path); err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "verified", "path": path})
	}}
	cmd.Flags().StringVar(&path, "path", "", "Backup database path")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}
