package cmds

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	cmd_sources "github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/stretchr/testify/require"
)

func TestAdminInvitationIssueAndRevoke(t *testing.T) {
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "tinyidp.sqlite")
	keyPath := filepath.Join(directory, "invitation.key")
	require.NoError(t, os.WriteFile(keyPath, []byte("0123456789abcdef0123456789abcdef"), 0o600))

	issue, err := newAdminInvitationIssueCommand(&dbPath)
	require.NoError(t, err)
	issue.now = func() time.Time { return time.Date(2026, time.July, 21, 18, 0, 0, 0, time.UTC) }
	issueValues := values.New()
	require.NoError(t, cmd_sources.Execute(issue.Schema, issueValues, cmd_sources.FromMap(map[string]map[string]any{
		"default": {"lookup-key-file": keyPath, "audience": "goja-client", "policy-version": "v1", "ttl": "1h"},
	})))
	processor := &captureProcessor{}
	require.NoError(t, issue.RunIntoGlazeProcessor(context.Background(), issueValues, processor))
	require.Len(t, processor.rows, 1)
	code := rowVal(processor.rows[0], "code")
	require.NotEmpty(t, code)
	require.NotEmpty(t, rowVal(processor.rows[0], "invitation_id"))

	service, closeFn, err := openInvitationService(dbPath, keyPath)
	require.NoError(t, err)
	_, err = service.Inspect(t.Context(), code, "goja-client", issue.now())
	closeFn()
	require.NoError(t, err)

	codePath := filepath.Join(directory, "invitation-code")
	require.NoError(t, os.WriteFile(codePath, []byte(code), 0o600))
	revoke, err := newAdminInvitationRevokeCommand(&dbPath)
	require.NoError(t, err)
	revoke.now = func() time.Time { return issue.now().Add(time.Minute) }
	revokeValues := values.New()
	require.NoError(t, cmd_sources.Execute(revoke.Schema, revokeValues, cmd_sources.FromMap(map[string]map[string]any{
		"default": {"lookup-key-file": keyPath, "code-file": codePath},
	})))
	processor = &captureProcessor{}
	require.NoError(t, revoke.RunIntoGlazeProcessor(context.Background(), revokeValues, processor))
	require.Equal(t, "revoked", rowVal(processor.rows[0], "status"))
	require.Equal(t, "goja-client", rowVal(processor.rows[0], "audience"))
	require.Equal(t, "v1", rowVal(processor.rows[0], "policy_version"))

	service, closeFn, err = openInvitationService(dbPath, keyPath)
	require.NoError(t, err)
	defer closeFn()
	_, err = service.Inspect(t.Context(), code, "goja-client", revoke.now())
	require.Error(t, err)
	audit, err := os.ReadFile(dbPath + ".audit.jsonl")
	require.NoError(t, err)
	require.Contains(t, string(audit), `"name":"signup_invitation.revoked","client_id":"goja-client"`)
}

func TestAdminInvitationIssueDeliversCodeBeforeAuditFailure(t *testing.T) {
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "tinyidp.sqlite")
	keyPath := filepath.Join(directory, "invitation.key")
	require.NoError(t, os.WriteFile(keyPath, []byte("0123456789abcdef0123456789abcdef"), 0o600))
	require.NoError(t, os.Mkdir(dbPath+".audit.jsonl", 0o700))

	issue, err := newAdminInvitationIssueCommand(&dbPath)
	require.NoError(t, err)
	issuedAt := time.Date(2026, time.July, 22, 20, 0, 0, 0, time.UTC)
	issue.now = func() time.Time { return issuedAt }
	issueValues := values.New()
	require.NoError(t, cmd_sources.Execute(issue.Schema, issueValues, cmd_sources.FromMap(map[string]map[string]any{
		"default": {"lookup-key-file": keyPath, "audience": "goja-client", "policy-version": "v1", "ttl": "1h"},
	})))
	processor := &captureProcessor{}
	err = issue.RunIntoGlazeProcessor(context.Background(), issueValues, processor)
	require.Error(t, err)
	require.Len(t, processor.rows, 1)
	code := rowVal(processor.rows[0], "code")
	require.NotEmpty(t, code)
	require.NotContains(t, err.Error(), code)

	service, closeFn, openErr := openInvitationService(dbPath, keyPath)
	require.NoError(t, openErr)
	defer closeFn()
	_, inspectErr := service.Inspect(t.Context(), code, "goja-client", issuedAt)
	require.NoError(t, inspectErr)
}

func TestAdminCommandBuildsGlazedInvitationChildren(t *testing.T) {
	admin, err := NewAdminCommand()
	require.NoError(t, err)
	invitation, _, err := admin.Find([]string{"invitation"})
	require.NoError(t, err)
	require.NotNil(t, invitation)
	require.Len(t, invitation.Commands(), 2)
}
