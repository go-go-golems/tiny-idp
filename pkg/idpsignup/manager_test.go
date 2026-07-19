package idpsignup_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
)

func TestGenerationManagerSwapsOnlyWarmedCandidatesAndRetainsPriorGeneration(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	first := manager.Snapshot()
	require.True(t, first.Ready)

	_, err = manager.Activate(ctx, "not valid JavaScript")
	require.Error(t, err)
	assert.Equal(t, first.ActiveFingerprint, manager.Snapshot().ActiveFingerprint)

	secondSource := strings.Replace(idpsignup.DefaultSource, "Create an account", "Create your account", 1)
	secondFingerprint, err := manager.Activate(ctx, secondSource)
	require.NoError(t, err)
	assert.NotEqual(t, first.ActiveFingerprint, secondFingerprint)
	second := manager.Snapshot()
	assert.Equal(t, secondFingerprint, second.ActiveFingerprint)
	assert.ElementsMatch(t, []string{first.ActiveFingerprint, secondFingerprint}, second.Retained)
	_, err = manager.ExecutorFor(first.ActiveFingerprint)
	require.NoError(t, err)
}

func TestGenerationManagerEvictsOnlyBeyondRetention(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	first := manager.Snapshot().ActiveFingerprint
	second, err := manager.Activate(ctx, strings.Replace(idpsignup.DefaultSource, "Create an account", "Create your account", 1))
	require.NoError(t, err)
	third, err := manager.Activate(ctx, strings.Replace(idpsignup.DefaultSource, "Create an account", "Create another account", 1))
	require.NoError(t, err)
	assert.NotEqual(t, second, third)
	_, err = manager.ExecutorFor(first)
	require.Error(t, err)
	_, err = manager.ExecutorFor(second)
	require.NoError(t, err)
	_, err = manager.ExecutorFor(third)
	require.NoError(t, err)
}

func TestGenerationManagerKeepsActiveGenerationWhenEmbeddedTestFails(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.EmailVerifiedSource, 1, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	active := manager.Snapshot().ActiveFingerprint
	failing := strings.Replace(idpsignup.EmailVerifiedSource, `expectedKind:"present"`, `expectedKind:"deny"`, 1)
	_, err = manager.Activate(ctx, failing)
	require.Error(t, err)
	assert.Equal(t, active, manager.Snapshot().ActiveFingerprint)
}

func TestGenerationManagerReadinessTracksClosedActivePool(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 1)
	require.NoError(t, err)
	require.NoError(t, manager.Ready())
	snapshot := manager.Snapshot()
	assert.True(t, snapshot.Ready)
	assert.Equal(t, 1, snapshot.Pool.Capacity)

	require.NoError(t, manager.Close(ctx))
	assert.Error(t, manager.Ready())
	assert.False(t, manager.Snapshot().Ready)
}

func TestGenerationManagerDrainsEvictedAndRemainingWorkerPools(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 1)
	require.NoError(t, err)
	first, err := manager.Active()
	require.NoError(t, err)

	secondSource := strings.Replace(idpsignup.DefaultSource, "Create an account", "Create your account", 1)
	_, err = manager.Activate(ctx, secondSource)
	require.NoError(t, err)
	second, err := manager.Active()
	require.NoError(t, err)
	thirdSource := strings.Replace(idpsignup.DefaultSource, "Create an account", "Create another account", 1)
	_, err = manager.Activate(ctx, thirdSource)
	require.NoError(t, err)
	third, err := manager.Active()
	require.NoError(t, err)

	assert.True(t, first.PoolStats().Closed, "evicted generation must drain its worker pool")
	assert.False(t, second.PoolStats().Closed)
	assert.False(t, third.PoolStats().Closed)
	assert.Len(t, manager.Snapshot().Retained, 2)
	require.NoError(t, manager.Close(ctx))
	assert.True(t, second.PoolStats().Closed)
	assert.True(t, third.PoolStats().Closed)
}

func TestGenerationManagerRepeatedReloadsKeepRetentionBounded(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 2)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })

	for index := 0; index < 12; index++ {
		source := strings.Replace(idpsignup.DefaultSource, "Create an account", fmt.Sprintf("Create account %d", index), 1)
		_, err := manager.Activate(ctx, source)
		require.NoError(t, err)
		snapshot := manager.Snapshot()
		assert.True(t, snapshot.Ready)
		assert.LessOrEqual(t, len(snapshot.Retained), 3)
	}
}

func TestGenerationManagerReportsBoundedOperationalMetrics(t *testing.T) {
	ctx := context.Background()
	manager, err := idpsignup.NewGenerationManager(ctx, idpsignup.DefaultSource, 1, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	_, err = manager.Activate(ctx, "not valid JavaScript")
	require.Error(t, err)
	_, err = manager.Activate(ctx, strings.Replace(idpsignup.DefaultSource, "Create an account", "Create your account", 1))
	require.NoError(t, err)
	metrics := manager.Metrics()
	assert.Equal(t, uint64(1), metrics.Activations)
	assert.Equal(t, uint64(1), metrics.ActivationFailures)
	assert.Equal(t, uint64(1), metrics.Evicted)
	assert.Equal(t, 1, metrics.Retained)
	assert.Equal(t, 1, metrics.PoolCapacity)
	assert.Equal(t, 0, metrics.PoolActive)
}

func TestGenerationManagerActivationAuditIsRedacted(t *testing.T) {
	ctx := context.Background()
	sink := idp.NewMemorySink()
	manager, err := idpsignup.NewGenerationManagerWithOptions(ctx, idpsignup.DefaultSource, 1, 1, idpsignup.GenerationManagerOptions{Audit: sink})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close(context.Background())) })
	_, err = manager.Activate(ctx, strings.Replace(idpsignup.DefaultSource, "Create an account", "Create your account", 1))
	require.NoError(t, err)
	events := sink.Events()
	require.Len(t, events, 1)
	event := events[0]
	assert.Equal(t, "script.signup.activation", event.Name)
	assert.Equal(t, "accepted", event.Result)
	assert.Empty(t, event.Subject)
	assert.Empty(t, event.ClientID)
	assert.NotEmpty(t, event.Fields["source_fingerprint"])
	assert.NotEmpty(t, event.Fields["program_fingerprint"])
	assert.NotContains(t, event.Fields["source_fingerprint"], "Create your account")
}
