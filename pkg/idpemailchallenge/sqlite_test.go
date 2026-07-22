package idpemailchallenge_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestSQLiteChallengeSurvivesRestartAndAllowsOneVerification(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	path := filepath.Join(t.TempDir(), "idp.db")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	challenge := testChallenge(now)
	require.NoError(t, store.CreateEmailChallenge(ctx, challenge))
	require.NoError(t, store.Close())
	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, challenge.CodeHash, testBindings(), now)
	require.NoError(t, err)
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, challenge.CodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAlreadyTerminal)
}

func TestSQLiteChallengeConsumesVerifiedEvidenceOnce(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	challenge := testChallenge(now)
	require.NoError(t, store.CreateEmailChallenge(ctx, challenge))
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, challenge.CodeHash, testBindings(), now)
	require.NoError(t, err)
	_, err = store.ConsumeVerifiedEmailChallenge(ctx, challenge.ID, testBindings(), now)
	require.NoError(t, err)
	_, err = store.ConsumeVerifiedEmailChallenge(ctx, challenge.ID, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAlreadyTerminal)
}

func TestChallengeAttemptsExpiryResendAndCleanupFailClosed(t *testing.T) {
	now := time.Now().UTC()
	store := idpemailchallenge.NewMemoryStore()
	challenge := testChallenge(now)
	challenge.MaximumAttempts = 2
	challenge.ResendNotBefore = now.Add(time.Minute)
	require.NoError(t, store.CreateEmailChallenge(context.Background(), challenge))
	_, err := store.ResendEmailChallenge(context.Background(), challenge.ID, []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"), testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrResendLimited)
	_, err = store.VerifyEmailChallenge(context.Background(), challenge.ID, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrConflict)
	_, err = store.VerifyEmailChallenge(context.Background(), challenge.ID, []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAttemptsExceeded)
	expired := testChallenge(now.Add(-time.Hour))
	expired.ID = "expired"
	expired.ExpiresAt = now.Add(-time.Minute)
	expired.CreatedAt = now.Add(-2 * time.Hour)
	require.NoError(t, store.CreateEmailChallenge(context.Background(), expired))
	_, err = store.LoadEmailChallenge(context.Background(), expired.ID, now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrExpired)
	expiredRows, err := store.ListExpiredEmailChallenges(context.Background(), now, 10)
	require.NoError(t, err)
	require.Len(t, expiredRows, 1)
	require.NoError(t, store.DeleteExpiredEmailChallenge(context.Background(), expired.ID, now))
}

func TestSQLiteChallengeResendRestoresAttemptBudgetAndInvalidatesOldCode(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	challenge := testChallenge(now)
	challenge.MaximumAttempts = 2
	challenge.MaximumResends = 1
	require.NoError(t, store.CreateEmailChallenge(ctx, challenge))
	wrongCodeHash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, wrongCodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrConflict)
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, wrongCodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAttemptsExceeded)

	newCodeHash := []byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	resend, err := store.ResendEmailChallenge(ctx, challenge.ID, newCodeHash, testBindings(), now)
	require.NoError(t, err)
	assert.Zero(t, resend.Attempts)
	assert.Equal(t, uint32(1), resend.Resends)
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, challenge.CodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrConflict)
	_, err = store.VerifyEmailChallenge(ctx, challenge.ID, newCodeHash, testBindings(), now)
	require.NoError(t, err)
}
