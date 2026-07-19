package sqlitestore_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation/idpcontinuationtest"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func TestWorkflowContinuationStoreSuite(t *testing.T) {
	idpcontinuationtest.RunStoreSuite(t, func(t *testing.T) idpcontinuation.Store {
		store, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "continuations.db")))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, store.Close()) })
		return store
	})
}

func TestWorkflowContinuationSurvivesStoreAndServiceRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "restart.db")
	now := time.Date(2026, time.July, 19, 18, 0, 0, 0, time.UTC)
	key := []byte("0123456789abcdef0123456789abcdef")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	service := newRestartService(t, store, key, &now)
	handle, stored, err := service.Create(ctx, restartRecord(now))
	require.NoError(t, err)
	require.NoError(t, store.Close())

	store, err = sqlitestore.Open(ctx, sqlitestore.DefaultConfig(path))
	require.NoError(t, err)
	defer store.Close()
	service = newRestartService(t, store, key, &now)
	loaded, err := service.Load(ctx, handle, restartBindings())
	require.NoError(t, err)
	assert.Equal(t, stored, loaded)

	consumed, err := service.Consume(ctx, handle, loaded.Revision, restartBindings(), idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalComplete})
	require.NoError(t, err)
	assert.Equal(t, idpcontinuation.StatusConsumed, consumed.Status)
	_, err = service.Load(ctx, handle, restartBindings())
	var failure *idpcontinuation.Failure
	require.True(t, errors.As(err, &failure))
	assert.Equal(t, idpcontinuation.FailureReplayed, failure.Class)
}

func newRestartService(t *testing.T, store idpcontinuation.Store, key []byte, now *time.Time) *idpcontinuation.Service {
	t.Helper()
	service, err := idpcontinuation.NewService(store, idpcontinuation.Config{
		HashKey: key,
		Clock:   func() time.Time { return *now },
		Resolver: restartResolver{
			program: restartProgram(),
		},
	})
	require.NoError(t, err)
	return service
}

type restartResolver struct{ program idpprogram.Program }

func (r restartResolver) ResolveProgram(_ context.Context, fingerprint string) (idpprogram.Program, error) {
	if fingerprint != "restart-program" {
		return idpprogram.Program{}, errors.New("generation unavailable")
	}
	return r.program, nil
}

func restartProgram() idpprogram.Program {
	return idpprogram.Program{
		Schemas: map[string]idpprogram.Schema{
			"input": {ID: "input", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024},
		},
		Lambdas: map[string]idpprogram.LambdaSpec{
			"resume": {ID: "resume", InputSchema: "input"},
		},
		Workflows: map[string]idpprogram.Workflow{
			"signup": {ID: "signup", Version: 1, EntryHandler: "resume", Handlers: map[string]idpprogram.HandlerSpec{
				"resume": {ID: "resume", LambdaID: "resume"},
			}},
		},
	}
}

func restartRecord(now time.Time) idpcontinuation.WorkflowContinuation {
	return idpcontinuation.WorkflowContinuation{
		WorkflowID:         "signup",
		ResumeHandlerID:    "resume",
		ProgramFingerprint: "restart-program",
		SchemaVersion:      "schema-1",
		WorkflowVersion:    1,
		RequestDigest:      []byte("request"),
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-1",
		BrowserBindingHash: []byte("browser"),
		BrowserContextHash: []byte("context"),
		InputSchema:        "input",
		Carry:              []byte(`{}`),
		ExpiresAt:          now.Add(time.Hour),
		SecretReferences:   []idpcontinuation.SecretReference{},
		EvidenceReferences: []idpcontinuation.EvidenceReference{},
	}
}

func restartBindings() idpcontinuation.Bindings {
	return idpcontinuation.Bindings{
		WorkflowID:         "signup",
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-1",
		ProgramFingerprint: "restart-program",
		RequestDigest:      []byte("request"),
		BrowserBindingHash: []byte("browser"),
		BrowserContextHash: []byte("context"),
	}
}
