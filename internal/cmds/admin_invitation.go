package cmds

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/spf13/cobra"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

type adminInvitationIssueSettings struct {
	Audience      string `glazed:"audience"`
	PolicyVersion string `glazed:"policy-version"`
	TTL           string `glazed:"ttl"`
	LookupKeyFile string `glazed:"lookup-key-file"`
}

type adminInvitationRevokeSettings struct {
	Audience      string `glazed:"audience"`
	LookupKeyFile string `glazed:"lookup-key-file"`
	CodeFile      string `glazed:"code-file"`
}

type AdminInvitationIssueCommand struct {
	*cmds.CommandDescription
	dbPath *string
	now    func() time.Time
}

type AdminInvitationRevokeCommand struct {
	*cmds.CommandDescription
	dbPath *string
	now    func() time.Time
}

func newAdminInvitationCommand(dbPath *string) (*cobra.Command, error) {
	root := &cobra.Command{Use: "invitation", Short: "Issue and revoke durable signup invitations"}
	issue, err := newAdminInvitationIssueCommand(dbPath)
	if err != nil {
		return nil, err
	}
	issueCobra, err := cli.BuildCobraCommand(issue)
	if err != nil {
		return nil, err
	}
	revoke, err := newAdminInvitationRevokeCommand(dbPath)
	if err != nil {
		return nil, err
	}
	revokeCobra, err := cli.BuildCobraCommand(revoke)
	if err != nil {
		return nil, err
	}
	root.AddCommand(issueCobra, revokeCobra)
	return root, nil
}

func newAdminInvitationSections() (schema.Section, schema.Section, error) {
	output, err := settings.NewGlazedSchema(settings.WithOutputSectionOptions(schema.WithDefaults(map[string]any{"output": "json"})))
	if err != nil {
		return nil, nil, err
	}
	command, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, nil, err
	}
	return output, command, nil
}

func newAdminInvitationIssueCommand(dbPath *string) (*AdminInvitationIssueCommand, error) {
	output, command, err := newAdminInvitationSections()
	if err != nil {
		return nil, err
	}
	description := cmds.NewCommandDescription("issue",
		cmds.WithShort("Issue a one-time signup invitation and print its raw code once"),
		cmds.WithFlags(
			fields.New("audience", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Exact OIDC client ID allowed to redeem the invitation")),
			fields.New("policy-version", fields.TypeString, fields.WithDefault("signup-invite-v1"), fields.WithHelp("Reviewed signup invitation policy version")),
			fields.New("ttl", fields.TypeString, fields.WithDefault("24h"), fields.WithHelp("Invitation lifetime")),
			fields.New("lookup-key-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing the durable invitation HMAC lookup key")),
		),
		cmds.WithSections(output, command),
	)
	return &AdminInvitationIssueCommand{CommandDescription: description, dbPath: dbPath, now: time.Now}, nil
}

func newAdminInvitationRevokeCommand(dbPath *string) (*AdminInvitationRevokeCommand, error) {
	output, command, err := newAdminInvitationSections()
	if err != nil {
		return nil, err
	}
	description := cmds.NewCommandDescription("revoke",
		cmds.WithShort("Revoke one unused signup invitation read from an owner-only file"),
		cmds.WithFlags(
			fields.New("audience", fields.TypeString, fields.WithRequired(true), fields.WithHelp("OIDC client ID recorded in the revocation audit event")),
			fields.New("lookup-key-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing the durable invitation HMAC lookup key")),
			fields.New("code-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Owner-only file containing exactly the invitation code to revoke")),
		),
		cmds.WithSections(output, command),
	)
	return &AdminInvitationRevokeCommand{CommandDescription: description, dbPath: dbPath, now: time.Now}, nil
}

func (c *AdminInvitationIssueCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, processor middlewares.Processor) error {
	var cfg adminInvitationIssueSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &cfg); err != nil {
		return err
	}
	ttl, err := time.ParseDuration(cfg.TTL)
	if err != nil || ttl <= 0 || strings.TrimSpace(cfg.PolicyVersion) == "" {
		return fmt.Errorf("invitation ttl must be positive and policy-version is required")
	}
	service, closeFn, err := openInvitationService(valueOf(c.dbPath), cfg.LookupKeyFile)
	if err != nil {
		return err
	}
	defer closeFn()
	id, err := randomAdminToken(18)
	if err != nil {
		return err
	}
	code, err := randomAdminToken(32)
	if err != nil {
		return err
	}
	now := c.now().UTC()
	expiresAt := now.Add(ttl)
	if err := service.Issue(ctx, idpinvite.DurableIssue{Code: code, ID: id, Audience: cfg.Audience, PolicyVersion: cfg.PolicyVersion, ExpiresAt: expiresAt}); err != nil {
		return err
	}
	if err := emitAdminAudit(ctx, valueOf(c.dbPath), idp.Event{Time: now, Name: "signup_invitation.issued", ClientID: cfg.Audience, Result: "accepted", Fields: map[string]string{"invitation_id": id, "policy_version": cfg.PolicyVersion, "expires_at": expiresAt.Format(time.RFC3339)}}); err != nil {
		return err
	}
	return processor.AddRow(ctx, types.NewRow(
		types.MRP("status", "issued"), types.MRP("invitation_id", id), types.MRP("audience", cfg.Audience),
		types.MRP("policy_version", cfg.PolicyVersion), types.MRP("expires_at", expiresAt), types.MRP("code", code),
	))
}

func (c *AdminInvitationRevokeCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, processor middlewares.Processor) error {
	var cfg adminInvitationRevokeSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &cfg); err != nil {
		return err
	}
	code, err := readOwnerOnlySecret(cfg.CodeFile)
	if err != nil {
		return fmt.Errorf("read invitation code file: %w", err)
	}
	defer clearProductionSecret(code)
	service, closeFn, err := openInvitationService(valueOf(c.dbPath), cfg.LookupKeyFile)
	if err != nil {
		return err
	}
	defer closeFn()
	now := c.now().UTC()
	if err := service.Revoke(ctx, string(code), now); err != nil {
		return err
	}
	if err := emitAdminAudit(ctx, valueOf(c.dbPath), idp.Event{Time: now, Name: "signup_invitation.revoked", ClientID: cfg.Audience, Result: "accepted"}); err != nil {
		return err
	}
	return processor.AddRow(ctx, types.NewRow(types.MRP("status", "revoked"), types.MRP("audience", cfg.Audience)))
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func openInvitationService(dbPath, lookupKeyFile string) (*idpinvite.DurableService, func(), error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, nil, fmt.Errorf("--db is required")
	}
	key, err := readOwnerOnlySecret(lookupKeyFile)
	if err != nil {
		return nil, nil, err
	}
	defer clearProductionSecret(key)
	store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		return nil, nil, err
	}
	service, err := idpinvite.NewDurableService(store, key)
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return service, func() { _ = store.Close() }, nil
}

func randomAdminToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("random token size must be positive")
	}
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate invitation token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
