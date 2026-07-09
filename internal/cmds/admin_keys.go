package cmds

import (
	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/internal/admin"
)

func newAdminKeysCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "keys", Short: "Manage signing keys"}
	cmd.AddCommand(newAdminKeysGenerateCommand(dbPath))
	cmd.AddCommand(newAdminKeysRotateCommand(dbPath))
	cmd.AddCommand(newAdminKeysListCommand(dbPath))
	cmd.AddCommand(newAdminKeysRetireCommand(dbPath))
	return cmd
}

func newAdminKeysGenerateCommand(dbPath *string) *cobra.Command {
	var kid string
	var active bool
	cmd := &cobra.Command{Use: "generate", Short: "Generate an RSA signing key", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		key, err := svc.GenerateSigningKey(cmd.Context(), kid, active)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "generated", "key": admin.RedactSigningKey(key)})
	}}
	cmd.Flags().StringVar(&kid, "kid", "", "Key ID (default generated from current time)")
	cmd.Flags().BoolVar(&active, "active", true, "Activate the generated key")
	return cmd
}

func newAdminKeysRotateCommand(dbPath *string) *cobra.Command {
	var kid string
	cmd := &cobra.Command{Use: "rotate", Short: "Rotate the active RSA signing key", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		result, err := svc.RotateSigningKey(cmd.Context(), kid)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "rotated", "result": admin.RedactRotationResult(result)})
	}}
	cmd.Flags().StringVar(&kid, "kid", "", "New key ID")
	_ = cmd.MarkFlagRequired("kid")
	return cmd
}

func newAdminKeysListCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List verification keys", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		keys, err := svc.ListSigningKeys(cmd.Context())
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"keys": admin.RedactSigningKeys(keys)})
	}}
}

func newAdminKeysRetireCommand(dbPath *string) *cobra.Command {
	var kid string
	cmd := &cobra.Command{Use: "retire", Short: "Retire a signing key", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		if err := svc.RetireSigningKey(cmd.Context(), kid); err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "retired", "kid": kid})
	}}
	cmd.Flags().StringVar(&kid, "kid", "", "Key ID")
	_ = cmd.MarkFlagRequired("kid")
	return cmd
}
