package cmds

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"

	"github.com/go-go-golems/tiny-idp/internal/admin"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
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
	var id, secret, secretFile string
	var public, generateSecret, requirePKCE, canIntrospect bool
	var redirectURIs, scopes, grantTypes, audiences, postLogout []string
	var accessTTL, idTTL, refreshTTL time.Duration
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an OAuth client",
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedSecret, err := resolveClientSecret(secret, secretFile, generateSecret)
			if err != nil {
				return err
			}
			svc, closeFn, err := openAdminService(*dbPath)
			if err != nil {
				return err
			}
			defer closeFn()
			client, secretResult, err := svc.CreateClient(cmd.Context(), admin.CreateClientRequest{ID: id, Public: public, Secret: resolvedSecret, GenerateSecret: generateSecret, RedirectURIs: redirectURIs, PostLogoutRedirectURIs: postLogout, AllowedScopes: scopes, AllowedGrantTypes: grantTypes, AllowedAudiences: audiences, CanIntrospect: canIntrospect, RequirePKCE: requirePKCE, AccessTokenTTL: accessTTL, IDTokenTTL: idTTL, RefreshTokenTTL: refreshTTL})
			if err != nil {
				return err
			}
			return writeJSONLine(cmd.OutOrStdout(), map[string]any{"status": "created", "client": redactClient(client), "secret": secretResult})
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Client ID")
	cmd.Flags().BoolVar(&public, "public", false, "Create a public client")
	cmd.Flags().StringVar(&secret, "secret", "", "Client secret (prefer --secret-file or --generate-secret outside tests)")
	cmd.Flags().StringVar(&secretFile, "secret-file", "", "File containing the client secret")
	cmd.Flags().BoolVar(&generateSecret, "generate-secret", false, "Generate a one-time client secret")
	cmd.Flags().StringArrayVar(&redirectURIs, "redirect-uri", nil, "Allowed redirect URI (repeatable)")
	cmd.Flags().StringArrayVar(&postLogout, "post-logout-redirect-uri", nil, "Allowed post-logout redirect URI (repeatable)")
	cmd.Flags().StringArrayVar(&scopes, "scope", []string{"openid", "profile", "email"}, "Allowed scope (repeatable)")
	cmd.Flags().StringArrayVar(&grantTypes, "grant-type", nil, "Allowed OAuth grant type (repeatable)")
	cmd.Flags().StringArrayVar(&audiences, "audience", nil, "Allowed OAuth resource indicator (repeatable)")
	cmd.Flags().BoolVar(&canIntrospect, "can-introspect", false, "Authorize this confidential client as an OAuth resource server")
	cmd.Flags().BoolVar(&requirePKCE, "require-pkce", true, "Require PKCE")
	cmd.Flags().DurationVar(&accessTTL, "access-token-ttl", time.Hour, "Access token TTL")
	cmd.Flags().DurationVar(&idTTL, "id-token-ttl", time.Hour, "ID token TTL")
	cmd.Flags().DurationVar(&refreshTTL, "refresh-token-ttl", 30*24*time.Hour, "Refresh token TTL")
	cmd.MarkFlagsMutuallyExclusive("secret", "secret-file", "generate-secret")
	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("grant-type")
	return cmd
}

func resolveClientSecret(value, path string, generate bool) (string, error) {
	selected := 0
	if value != "" {
		selected++
	}
	if path != "" {
		selected++
	}
	if generate {
		selected++
	}
	if selected > 1 {
		return "", errors.New("--secret, --secret-file, and --generate-secret are mutually exclusive")
	}
	if path == "" {
		return value, nil
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", errors.Wrap(err, "inspect client secret file")
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("client secret file must be regular and not a symlink")
	}
	if info.Size() > 4096 {
		return "", errors.New("client secret file is too large")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "read client secret file")
	}
	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", errors.New("client secret file is empty")
	}
	return secret, nil
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
		"allowed_audiences":         client.AllowedAudiences,
		"can_introspect":            client.CanIntrospect,
		"require_pkce":              client.RequirePKCE,
		"access_token_ttl":          client.AccessTokenTTL.String(),
		"id_token_ttl":              client.IDTokenTTL.String(),
		"refresh_token_ttl":         client.RefreshTokenTTL.String(),
		"disabled":                  client.Disabled,
		"created_at":                client.CreatedAt,
		"updated_at":                client.UpdatedAt,
	}
}
