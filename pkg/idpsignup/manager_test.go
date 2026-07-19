package idpsignup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
