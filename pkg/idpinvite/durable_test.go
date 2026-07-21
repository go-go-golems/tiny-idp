package idpinvite_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idpinvite"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestDurableInvitationHashesCodesAndRedeemsOnlyOnce(t *testing.T) {
	service, err := idpinvite.NewDurableService(memory.New(), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	require.NoError(t, service.Issue(context.Background(), idpinvite.DurableIssue{Code: "invite-secret", ID: "invite-42", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	evidence, err := service.Redeem(context.Background(), "invite-secret", "message-app", now)
	require.NoError(t, err)
	assert.Equal(t, "invite-42", evidence.InvitationID)
	assert.NotContains(t, evidence.InvitationID, "secret")
	_, err = service.Redeem(context.Background(), "invite-secret", "message-app", now)
	assert.ErrorIs(t, err, idpstore.ErrAlreadyConsumed)
}

func TestDurableInvitationRejectsExpiryRevocationAndAudienceMismatch(t *testing.T) {
	service, err := idpinvite.NewDurableService(memory.New(), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	require.NoError(t, service.Issue(context.Background(), idpinvite.DurableIssue{Code: "revoked", ID: "invite-r", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	_, err = service.Redeem(context.Background(), "revoked", "other-app", now)
	assert.ErrorIs(t, err, idpstore.ErrNotFound)
	require.NoError(t, service.Revoke(context.Background(), "revoked", now))
	_, err = service.Redeem(context.Background(), "revoked", "message-app", now)
	assert.ErrorIs(t, err, idpstore.ErrInvitationRevoked)
	require.NoError(t, service.Issue(context.Background(), idpinvite.DurableIssue{Code: "expired", ID: "invite-expired", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Second)}))
	_, err = service.Redeem(context.Background(), "expired", "message-app", now.Add(time.Second))
	assert.ErrorIs(t, err, idpstore.ErrExpired)
}

func TestDurableInvitationConcurrentRedemptionHasOneWinner(t *testing.T) {
	service, err := idpinvite.NewDurableService(memory.New(), []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	require.NoError(t, service.Issue(context.Background(), idpinvite.DurableIssue{Code: "race", ID: "invite-race", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	var successes int
	var mu sync.Mutex
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := service.Redeem(context.Background(), "race", "message-app", now)
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
				return
			}
			assert.True(t, errors.Is(err, idpstore.ErrAlreadyConsumed))
		}()
	}
	group.Wait()
	assert.Equal(t, 1, successes)
}

func TestDurableInvitationSurvivesSQLiteRestartAndRemainsOneTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idp.db")
	store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	service, err := idpinvite.NewDurableService(store, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	require.NoError(t, service.Issue(context.Background(), idpinvite.DurableIssue{Code: "restart", ID: "invite-restart", Audience: "message-app", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, store.Close())
	store, err = sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	service, err = idpinvite.NewDurableService(store, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	_, err = service.Redeem(context.Background(), "restart", "message-app", now)
	require.NoError(t, err)
	_, err = service.Redeem(context.Background(), "restart", "message-app", now)
	assert.ErrorIs(t, err, idpstore.ErrAlreadyConsumed)
}

func TestDurableInvitationRedemptionRollsBackWithCallerTransaction(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	service, err := idpinvite.NewDurableService(store, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 21, 20, 0, 0, 0, time.UTC)
	require.NoError(t, service.Issue(ctx, idpinvite.DurableIssue{Code: "rollback", ID: "invite-rollback", Audience: "goja-client", PolicyVersion: "v1", ExpiresAt: now.Add(time.Hour)}))

	injected := errors.New("injected account commit failure")
	err = store.Update(ctx, func(tx idpstore.TxStore) error {
		if _, redeemErr := service.RedeemInTransaction(ctx, tx, "rollback", "goja-client", now); redeemErr != nil {
			return redeemErr
		}
		return injected
	})
	require.ErrorIs(t, err, injected)
	inspection, err := service.Inspect(ctx, "rollback", "goja-client", now)
	require.NoError(t, err)
	assert.Equal(t, "invite-rollback", inspection.InvitationID)
}
