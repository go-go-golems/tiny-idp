// Package idpcontinuationtest contains the shared continuation-store contract.
package idpcontinuationtest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

type Factory func(*testing.T) idpcontinuation.Store

func RunStoreSuite(t *testing.T, factory Factory) {
	t.Helper()
	t.Run("create-load-and-no-raw-handle", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, stored := harness.create(t, time.Hour)
		loaded, err := harness.service.Load(context.Background(), handle, harness.bindings())
		require.NoError(t, err)
		assert.Equal(t, stored, loaded)
		raw, err := base64.RawURLEncoding.DecodeString(handle)
		require.NoError(t, err)
		assert.NotEqual(t, raw, stored.HandleHash)
		assert.Len(t, stored.HandleHash, sha256.Size)
	})

	t.Run("advance-is-one-use", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, current := harness.create(t, time.Hour)
		next := harness.record("submitted", time.Hour)
		nextHandle, nextStored, err := harness.service.Advance(context.Background(), handle, current.Revision, harness.bindings(), next)
		require.NoError(t, err)
		assert.NotEqual(t, handle, nextHandle)
		_, err = harness.service.Load(context.Background(), handle, harness.bindings())
		assertFailure(t, err, idpcontinuation.FailureReplayed)
		loaded, err := harness.service.Load(context.Background(), nextHandle, harness.bindings())
		require.NoError(t, err)
		assert.Equal(t, nextStored, loaded)
	})

	t.Run("terminal-consume-is-one-use", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, current := harness.create(t, time.Hour)
		consumed, err := harness.service.Consume(context.Background(), handle, current.Revision, harness.bindings(), idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalComplete})
		require.NoError(t, err)
		assert.Equal(t, idpcontinuation.StatusConsumed, consumed.Status)
		assert.Equal(t, current.Revision+1, consumed.Revision)
		_, err = harness.service.Consume(context.Background(), handle, current.Revision, harness.bindings(), idpcontinuation.TerminalOutcome{Kind: idpcontinuation.TerminalComplete})
		assertFailure(t, err, idpcontinuation.FailureReplayed)
	})

	t.Run("revision-conflict-and-revocation", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, current := harness.create(t, time.Hour)
		_, _, err := harness.service.Advance(context.Background(), handle, current.Revision+1, harness.bindings(), harness.record("submitted", time.Hour))
		assertFailure(t, err, idpcontinuation.FailureReplayed)
		require.NoError(t, harness.service.Revoke(context.Background(), handle, current.Revision))
		_, err = harness.service.Load(context.Background(), handle, harness.bindings())
		assertFailure(t, err, idpcontinuation.FailureRevoked)
	})

	t.Run("expiry-and-binding-failures", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, _ := harness.create(t, time.Minute)
		wrongBrowser := harness.bindings()
		wrongBrowser.BrowserBindingHash = []byte("other-browser")
		_, err := harness.service.Load(context.Background(), handle, wrongBrowser)
		assertFailure(t, err, idpcontinuation.FailureBrowserMismatch)
		wrongClient := harness.bindings()
		wrongClient.ClientGeneration = "client-generation-2"
		_, err = harness.service.Load(context.Background(), handle, wrongClient)
		assertFailure(t, err, idpcontinuation.FailureClientMismatch)
		harness.now = harness.now.Add(2 * time.Minute)
		_, err = harness.service.Load(context.Background(), handle, harness.bindings())
		assertFailure(t, err, idpcontinuation.FailureExpired)
	})

	t.Run("concurrent-advance-has-one-winner", func(t *testing.T) {
		harness := newHarness(t, factory(t))
		handle, current := harness.create(t, time.Hour)
		const workers = 24
		errs := make(chan error, workers)
		var wait sync.WaitGroup
		for i := 0; i < workers; i++ {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				next := harness.record("submitted", time.Hour)
				next.Carry = []byte(fmt.Sprintf(`{"value":"next-%d"}`, index))
				_, _, err := harness.service.Advance(context.Background(), handle, current.Revision, harness.bindings(), next)
				errs <- err
			}(i)
		}
		wait.Wait()
		close(errs)
		winners := 0
		for err := range errs {
			if err == nil {
				winners++
				continue
			}
			var failure *idpcontinuation.Failure
			require.True(t, errors.As(err, &failure), "unexpected error: %v", err)
			assert.Equal(t, idpcontinuation.FailureReplayed, failure.Class)
		}
		assert.Equal(t, 1, winners)
	})

	t.Run("cleanup-removes-expired-attachments", func(t *testing.T) {
		store := factory(t)
		cleaner := &recordingCleaner{}
		harness := newHarnessWithCleaner(t, store, cleaner)
		_, _ = harness.create(t, time.Minute)
		harness.now = harness.now.Add(2 * time.Minute)
		count, err := harness.service.Cleanup(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, []string{"pending-secret", "invite-evidence"}, cleaner.ids)
	})

	t.Run("cleanup-failure-retains-retry-intent", func(t *testing.T) {
		store := factory(t)
		cleaner := &recordingCleaner{failOnce: true}
		harness := newHarnessWithCleaner(t, store, cleaner)
		_, _ = harness.create(t, time.Minute)
		harness.now = harness.now.Add(2 * time.Minute)
		count, err := harness.service.Cleanup(context.Background(), 10)
		require.Error(t, err)
		assert.Zero(t, count)
		expired, listErr := store.ListExpired(context.Background(), harness.now, 10)
		require.NoError(t, listErr)
		assert.Len(t, expired, 1, "failed attachment cleanup must retain a retryable record")
		count, err = harness.service.Cleanup(context.Background(), 10)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

type harness struct {
	now     time.Time
	service *idpcontinuation.Service
}

func newHarness(t *testing.T, store idpcontinuation.Store) *harness {
	return newHarnessWithCleaner(t, store, nil)
}

func newHarnessWithCleaner(t *testing.T, store idpcontinuation.Store, cleaner idpcontinuation.AttachmentCleaner) *harness {
	t.Helper()
	h := &harness{now: time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)}
	service, err := idpcontinuation.NewService(store, idpcontinuation.Config{
		HashKey:  []byte("0123456789abcdef0123456789abcdef"),
		Clock:    func() time.Time { return h.now },
		Resolver: staticResolver{program: testProgram()},
		Cleaner:  cleaner,
	})
	require.NoError(t, err)
	h.service = service
	return h
}

func (h *harness) create(t *testing.T, ttl time.Duration) (string, idpcontinuation.WorkflowContinuation) {
	t.Helper()
	return mustCreate(t, h.service, h.record("start", ttl))
}

func (h *harness) record(handler string, ttl time.Duration) idpcontinuation.WorkflowContinuation {
	return idpcontinuation.WorkflowContinuation{
		WorkflowID:         "signup",
		ResumeHandlerID:    handler,
		ProgramFingerprint: "program-1",
		SchemaVersion:      "schema-1",
		WorkflowVersion:    1,
		RequestDigest:      []byte("request-digest"),
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-generation-1",
		BrowserBindingHash: []byte("browser-binding"),
		SessionIDHash:      []byte("session"),
		BrowserContextHash: []byte("browser-context"),
		InputSchema:        "input",
		Carry:              []byte(`{"value":"public"}`),
		SecretReferences:   []idpcontinuation.SecretReference{{Kind: "password", ID: "pending-secret"}},
		EvidenceReferences: []idpcontinuation.EvidenceReference{{Kind: "invite", ID: "invite-evidence"}},
		ExpiresAt:          h.now.Add(ttl),
	}
}

func (h *harness) bindings() idpcontinuation.Bindings {
	return idpcontinuation.Bindings{
		WorkflowID:         "signup",
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-generation-1",
		ProgramFingerprint: "program-1",
		RequestDigest:      []byte("request-digest"),
		BrowserBindingHash: []byte("browser-binding"),
		SessionIDHash:      []byte("session"),
		BrowserContextHash: []byte("browser-context"),
	}
}

func mustCreate(t *testing.T, service *idpcontinuation.Service, record idpcontinuation.WorkflowContinuation) (string, idpcontinuation.WorkflowContinuation) {
	t.Helper()
	handle, stored, err := service.Create(context.Background(), record)
	require.NoError(t, err)
	return handle, stored
}

func assertFailure(t *testing.T, err error, class idpcontinuation.FailureClass) {
	t.Helper()
	require.Error(t, err)
	var failure *idpcontinuation.Failure
	require.True(t, errors.As(err, &failure), "expected classified failure, got %v", err)
	assert.Equal(t, class, failure.Class)
}

type staticResolver struct{ program idpprogram.Program }

func (r staticResolver) ResolveProgram(_ context.Context, fingerprint string) (idpprogram.Program, error) {
	if fingerprint != "program-1" {
		return idpprogram.Program{}, errors.New("not retained")
	}
	return r.program, nil
}

func testProgram() idpprogram.Program {
	return idpprogram.Program{
		APIVersion: idpprogram.APIVersionV1,
		Schemas: map[string]idpprogram.Schema{
			"input": {ID: "input", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Fields: map[string]idpprogram.SchemaField{"value": {Ref: "text", Required: true}}},
			"text":  {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 256, MaxLength: 128},
		},
		Lambdas: map[string]idpprogram.LambdaSpec{
			"start":     {ID: "start", InputSchema: "input"},
			"submitted": {ID: "submitted", InputSchema: "input"},
		},
		Workflows: map[string]idpprogram.Workflow{
			"signup": {
				ID: "signup", Version: 1, EntryHandler: "start",
				Handlers: map[string]idpprogram.HandlerSpec{
					"start":     {ID: "start", LambdaID: "start"},
					"submitted": {ID: "submitted", LambdaID: "submitted"},
				},
			},
		},
	}
}

type recordingCleaner struct {
	ids      []string
	failOnce bool
}

func (c *recordingCleaner) DeleteContinuationAttachments(_ context.Context, secrets []idpcontinuation.SecretReference, evidence []idpcontinuation.EvidenceReference) error {
	if c.failOnce {
		c.failOnce = false
		return errors.New("injected attachment cleanup failure")
	}
	for _, reference := range secrets {
		c.ids = append(c.ids, reference.ID)
	}
	for _, reference := range evidence {
		c.ids = append(c.ids, reference.ID)
	}
	return nil
}
