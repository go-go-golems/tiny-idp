package idpinvite_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
)

func TestLookupCapabilityInspectsWithoutConsumingAndFixesAudienceNatively(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.July, 21, 18, 0, 0, 0, time.UTC)
	service, err := idpinvite.NewDurableService(memory.New(), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	require.NoError(t, service.Issue(ctx, idpinvite.DurableIssue{Code: "secret-code", ID: "invite-1", Audience: "goja-client", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))

	binding, err := idpinvite.NewLookupCapability(service, "goja-client", func() time.Time { return now })
	require.NoError(t, err)
	for range 2 {
		raw, invokeErr := binding.Invoke(ctx, json.RawMessage(`{"code":"secret-code"}`))
		require.NoError(t, invokeErr)
		var decision idpinvite.LookupDecision
		require.NoError(t, json.Unmarshal(raw, &decision))
		assert.True(t, decision.Valid)
		assert.Equal(t, "invite-1", decision.InvitationID)
		assert.NotContains(t, string(raw), "secret-code")
	}

	_, err = binding.Invoke(ctx, json.RawMessage(`{"code":"secret-code","audience":"other"}`))
	require.Error(t, err)

	wrongAudience, err := idpinvite.NewLookupCapability(service, "other-client", func() time.Time { return now })
	require.NoError(t, err)
	raw, err := wrongAudience.Invoke(ctx, json.RawMessage(`{"code":"secret-code"}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"valid":false}`, string(raw))

	_, err = service.Redeem(ctx, "secret-code", "goja-client", now)
	require.NoError(t, err)
	raw, err = binding.Invoke(ctx, json.RawMessage(`{"code":"secret-code"}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"valid":false}`, string(raw))
}
