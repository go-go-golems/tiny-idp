package cmds

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/internal/admin"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func newAdminInitCommand(dbPath *string) *cobra.Command {
	var generateKey bool
	var kid string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a tinyidp SQLite database",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, closeFn, err := openAdminService(*dbPath)
			if err != nil {
				return err
			}
			defer closeFn()
			result := map[string]any{"status": "initialized", "db": *dbPath}
			if generateKey {
				key, err := svc.GenerateSigningKey(cmd.Context(), kid, true)
				if err != nil {
					return err
				}
				result["signing_key"] = admin.RedactSigningKey(key)
			}
			return writeJSONLine(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().BoolVar(&generateKey, "generate-signing-key", false, "Generate and activate an initial RSA signing key")
	cmd.Flags().StringVar(&kid, "kid", "", "Key ID for --generate-signing-key")
	return cmd
}

func newAdminMigrateCommand(dbPath *string) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply SQLite migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := sqlitestore.MigrationNames()
			if err != nil {
				return err
			}
			if dryRun {
				return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "dry-run", "migrations": names})
			}
			st, closeFn, err := openAdminStore(*dbPath)
			if err != nil {
				return err
			}
			defer closeFn()
			if err := st.Migrate(cmd.Context()); err != nil {
				return err
			}
			return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "migrated", "migrations": names})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "List migrations without opening or changing the database")
	return cmd
}

func newAdminDoctorCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run admin preflight checks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, closeFn, err := openAdminService(*dbPath)
			if err != nil {
				return err
			}
			defer closeFn()
			report := svc.Doctor(cmd.Context())
			if err := writeJSONLine(cmd.OutOrStdout(), report); err != nil {
				return err
			}
			if !report.OK {
				return fmt.Errorf("doctor checks failed")
			}
			return nil
		},
	}
	return cmd
}

func openAdminStore(dbPath string) (*sqlitestore.Store, func(), error) {
	if dbPath == "" {
		return nil, nil, fmt.Errorf("--db is required")
	}
	st, err := sqlitestore.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}
	return st, func() { _ = st.Close() }, nil
}
