package cmds

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/manuel/tinyidp/internal/admin"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

func newAdminClientCommand(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "client", Short: "Manage OAuth clients"}
	cmd.AddCommand(newAdminClientCreateCommand(dbPath))
	cmd.AddCommand(newAdminClientListCommand(dbPath))
	cmd.AddCommand(newAdminClientGetCommand(dbPath))
	cmd.AddCommand(newAdminClientDisableCommand(dbPath, true))
	cmd.AddCommand(newAdminClientDisableCommand(dbPath, false))
	cmd.AddCommand(newAdminClientRotateSecretCommand(dbPath))
	return cmd
}

func newAdminClientCreateCommand(dbPath *string) *cobra.Command {
	var id, secret string
	var public, generateSecret, requirePKCE bool
	var redirectURIs, scopes, grantTypes, postLogout []string
	var accessTTL, idTTL, refreshTTL time.Duration
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an OAuth client",
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, closeFn, err := openAdminService(*dbPath)
			if err != nil {
				return err
			}
			defer closeFn()
			client, secretResult, err := svc.CreateClient(cmd.Context(), admin.CreateClientRequest{ID: id, Public: public, Secret: secret, GenerateSecret: generateSecret, RedirectURIs: redirectURIs, PostLogoutRedirectURIs: postLogout, AllowedScopes: scopes, AllowedGrantTypes: grantTypes, RequirePKCE: requirePKCE, AccessTokenTTL: accessTTL, IDTokenTTL: idTTL, RefreshTokenTTL: refreshTTL})
			if err != nil {
				return err
			}
			return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "created", "client": redactClient(client), "secret": secretResult})
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Client ID")
	cmd.Flags().BoolVar(&public, "public", false, "Create a public client")
	cmd.Flags().StringVar(&secret, "secret", "", "Client secret (prefer --generate-secret outside tests)")
	cmd.Flags().BoolVar(&generateSecret, "generate-secret", false, "Generate a one-time client secret")
	cmd.Flags().StringArrayVar(&redirectURIs, "redirect-uri", nil, "Allowed redirect URI (repeatable)")
	cmd.Flags().StringArrayVar(&postLogout, "post-logout-redirect-uri", nil, "Allowed post-logout redirect URI (repeatable)")
	cmd.Flags().StringArrayVar(&scopes, "scope", []string{"openid", "profile", "email"}, "Allowed scope (repeatable)")
	cmd.Flags().StringArrayVar(&grantTypes, "grant-type", nil, "Allowed OAuth grant type (repeatable)")
	cmd.Flags().BoolVar(&requirePKCE, "require-pkce", true, "Require PKCE")
	cmd.Flags().DurationVar(&accessTTL, "access-token-ttl", time.Hour, "Access token TTL")
	cmd.Flags().DurationVar(&idTTL, "id-token-ttl", time.Hour, "ID token TTL")
	cmd.Flags().DurationVar(&refreshTTL, "refresh-token-ttl", 30*24*time.Hour, "Refresh token TTL")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("grant-type")
	return cmd
}

func newAdminClientListCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List clients", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		clients, err := svc.ListClients(cmd.Context())
		if err != nil {
			return err
		}
		out := make([]any, 0, len(clients))
		for _, c := range clients {
			out = append(out, redactClient(c))
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"clients": out})
	}}
}

func newAdminClientGetCommand(dbPath *string) *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "get", Short: "Get client", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		client, err := svc.GetClient(cmd.Context(), id)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"client": redactClient(client)})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Client ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAdminClientDisableCommand(dbPath *string, disabled bool) *cobra.Command {
	name := "enable"
	status := "enabled"
	if disabled {
		name = "disable"
		status = "disabled"
	}
	var id string
	cmd := &cobra.Command{Use: name, Short: name + " client", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		client, err := svc.SetClientDisabled(cmd.Context(), id, disabled)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": status, "client": redactClient(client)})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Client ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newAdminClientRotateSecretCommand(dbPath *string) *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "rotate-secret", Short: "Rotate a confidential client secret", RunE: func(cmd *cobra.Command, _ []string) error {
		svc, closeFn, err := openAdminService(*dbPath)
		if err != nil {
			return err
		}
		defer closeFn()
		client, secret, err := svc.RotateClientSecret(cmd.Context(), id)
		if err != nil {
			return err
		}
		return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "secret-rotated", "client": redactClient(client), "secret": secret})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Client ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func redactClient(client idpstore.Client) map[string]any {
	return map[string]any{
		"id":                        client.ID,
		"public":                    client.Public,
		"has_secret":                len(client.SecretHash) > 0,
		"redirect_uris":             client.RedirectURIs,
		"post_logout_redirect_uris": client.PostLogoutRedirectURIs,
		"allowed_scopes":            client.AllowedScopes,
		"require_pkce":              client.RequirePKCE,
		"access_token_ttl":          client.AccessTokenTTL.String(),
		"id_token_ttl":              client.IDTokenTTL.String(),
		"refresh_token_ttl":         client.RefreshTokenTTL.String(),
		"disabled":                  client.Disabled,
		"created_at":                client.CreatedAt,
		"updated_at":                client.UpdatedAt,
	}
}
