package idpcontinuation_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/memorystore"
)

func TestServicePinsGenerationAndClassifiesUnavailableGeneration(t *testing.T) {
	resolver := &toggleResolver{program: continuationProgram()}
	service := newService(t, resolver)
	handle, _, err := service.Create(context.Background(), continuationRecord())
	require.NoError(t, err)
	resolver.unavailable.Store(true)

	_, err = service.Load(context.Background(), handle, continuationBindings())
	assertFailureClass(t, err, idpcontinuation.FailureGenerationUnavailable)
}

func TestServiceValidatesDestinationAndSensitiveStateBoundaries(t *testing.T) {
	service := newService(t, &toggleResolver{program: continuationProgram()})
	record := continuationRecord()
	record.Carry = []byte(`{"email":"a@example.test","password":"secret"}`)
	_, _, err := service.Create(context.Background(), record)
	assertFailureClass(t, err, idpcontinuation.FailureInvalid)
	assert.Contains(t, err.Error(), "sensitive")

	record = continuationRecord()
	record.ResumeHandlerID = "unknown"
	_, _, err = service.Create(context.Background(), record)
	assertFailureClass(t, err, idpcontinuation.FailureInvalid)
	assert.Contains(t, err.Error(), "unknown")

	record = continuationRecord()
	handle, stored, err := service.Create(context.Background(), record)
	require.NoError(t, err)
	require.NoError(t, service.ValidateResumeInput(context.Background(), stored, []byte(`{"email":"a@example.test","password":"secret"}`)))
	err = service.ValidateResumeInput(context.Background(), stored, []byte(`{"email":"`+strings.Repeat("x", 200)+`"}`))
	assertFailureClass(t, err, idpcontinuation.FailureInvalid)

	_, err = service.Load(context.Background(), handle, idpcontinuation.Bindings{})
	assertFailureClass(t, err, idpcontinuation.FailureInvalid)
}

func TestServiceClassifiesMissingAndGenerationMismatchWithoutLeakingHandle(t *testing.T) {
	service := newService(t, &toggleResolver{program: continuationProgram()})
	_, err := service.Load(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", continuationBindings())
	assertFailureClass(t, err, idpcontinuation.FailureMissing)

	handle, _, err := service.Create(context.Background(), continuationRecord())
	require.NoError(t, err)
	bindings := continuationBindings()
	bindings.ProgramFingerprint = "other-generation"
	_, err = service.Load(context.Background(), handle, bindings)
	assertFailureClass(t, err, idpcontinuation.FailureGenerationMismatch)
}

func TestSafeTerminalKeepsPublicResponseUniformAndAuditClassSpecific(t *testing.T) {
	missing := idpcontinuation.ClassifyTerminal(&idpcontinuation.Failure{Class: idpcontinuation.FailureMissing, Err: idpcontinuation.ErrNotFound})
	replayed := idpcontinuation.ClassifyTerminal(&idpcontinuation.Failure{Class: idpcontinuation.FailureReplayed, Err: idpcontinuation.ErrConflict})
	assert.Equal(t, "interaction_unavailable", missing.PublicCode)
	assert.Equal(t, missing.PublicCode, replayed.PublicCode)
	assert.Equal(t, idpcontinuation.FailureMissing, missing.AuditClass)
	assert.Equal(t, idpcontinuation.FailureReplayed, replayed.AuditClass)
}

func newService(t *testing.T, resolver idpcontinuation.GenerationResolver) *idpcontinuation.Service {
	t.Helper()
	service, err := idpcontinuation.NewService(memorystore.NewContinuationStore(), idpcontinuation.Config{
		HashKey:  []byte("0123456789abcdef0123456789abcdef"),
		Clock:    func() time.Time { return time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC) },
		Resolver: resolver,
	})
	require.NoError(t, err)
	return service
}

type toggleResolver struct {
	program     idpprogram.Program
	unavailable atomic.Bool
}

func (r *toggleResolver) ResolveProgram(context.Context, string) (idpprogram.Program, error) {
	if r.unavailable.Load() {
		return idpprogram.Program{}, errors.New("not retained")
	}
	return r.program, nil
}

func continuationProgram() idpprogram.Program {
	return idpprogram.Program{
		Schemas: map[string]idpprogram.Schema{
			"input": {
				ID: "input", Kind: idpprogram.SchemaKindObject, MaxBytes: 128,
				Fields: map[string]idpprogram.SchemaField{
					"email":    {Ref: "text", Required: true},
					"password": {Ref: "text", Sensitive: true},
				},
			},
			"text": {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 64, MaxLength: 32},
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

func continuationRecord() idpcontinuation.WorkflowContinuation {
	return idpcontinuation.WorkflowContinuation{
		WorkflowID:         "signup",
		ResumeHandlerID:    "resume",
		ProgramFingerprint: "program-1",
		SchemaVersion:      "schema-1",
		WorkflowVersion:    1,
		RequestDigest:      []byte("request"),
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-1",
		BrowserBindingHash: []byte("browser"),
		BrowserContextHash: []byte("context"),
		InputSchema:        "input",
		Carry:              []byte(`{"email":"a@example.test"}`),
		ExpiresAt:          time.Date(2026, time.July, 19, 21, 0, 0, 0, time.UTC),
	}
}

func continuationBindings() idpcontinuation.Bindings {
	return idpcontinuation.Bindings{
		WorkflowID:         "signup",
		ClientID:           "app",
		RedirectURI:        "https://app.example/callback",
		ClientGeneration:   "client-1",
		ProgramFingerprint: "program-1",
		RequestDigest:      []byte("request"),
		BrowserBindingHash: []byte("browser"),
		BrowserContextHash: []byte("context"),
	}
}

func assertFailureClass(t *testing.T, err error, expected idpcontinuation.FailureClass) {
	t.Helper()
	require.Error(t, err)
	var failure *idpcontinuation.Failure
	require.True(t, errors.As(err, &failure), "expected classified failure, got %v", err)
	assert.Equal(t, expected, failure.Class)
}
