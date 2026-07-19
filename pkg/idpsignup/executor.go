// Package idpsignup contains the intentionally narrow native binding for the
// first scripted Tiny-IDP workflow: open local-account signup.
package idpsignup

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

//go:embed open_signup.js
var DefaultSource string

//go:embed email_verified_signup.js
var EmailVerifiedSource string

const (
	WorkflowID       = "signup"
	StartHandler     = "start"
	SubmittedHandler = "submitted"
)

type Executor struct {
	artifact *idpscript.Artifact
	pool     *idpscript.Pool
	metrics  executorMetrics
}

type executorMetrics struct{ invocations, failures, present, challenge, commit, other, latencyNanos, discarded atomic.Uint64 }
type ExecutorMetrics struct{ Invocations, Failures, Present, Challenge, Commit, Other, LatencyNanos, Discarded uint64 }

type TestResult struct {
	ID       string
	Passed   bool
	Expected idpprogram.OutcomeKind
	Actual   idpprogram.OutcomeKind
	Err      error
}

var deterministicTestFakeOutputs = map[string]json.RawMessage{
	"clock.now":         json.RawMessage(`{"unixMillis":0}`),
	"random.bytes":      json.RawMessage(`{"base64":""}`),
	"mailer.send":       json.RawMessage(`{"accepted":true}`),
	"identity.lookup":   json.RawMessage(`{"found":false}`),
	"invitation.lookup": json.RawMessage(`{"valid":false}`),
	"store.get":         json.RawMessage(`{"found":false}`),
}

// StartInput is the immutable, redacted view of a validated authorization
// interaction available to the signup-start lambda. It deliberately contains
// no Fosite request, HTTP request, cookie, browser handle, session identifier,
// or mutable store object.
type StartInput struct {
	ClientID          string `json:"clientId"`
	RedirectURI       string `json:"redirectUri"`
	RequestedScope    string `json:"requestedScope"`
	InteractionID     string `json:"interactionId"`
	HasBrowserSession bool   `json:"hasBrowserSession"`
}

var _ idpcontinuation.GenerationResolver = (*Executor)(nil)

func New(ctx context.Context, source string, workers int) (*Executor, error) {
	if source == "" {
		source = DefaultSource
	}
	if workers <= 0 {
		return nil, errors.New("signup executor worker count must be positive")
	}
	artifact, err := Compile(ctx, source)
	if err != nil {
		return nil, errors.Wrap(err, "compile signup program")
	}
	factory, err := idpscript.NewRuntimeFactory(schemas())
	if err != nil {
		return nil, errors.Wrap(err, "create signup runtime factory")
	}
	pool, err := idpscript.NewPool(ctx, artifact, factory, workers)
	if err != nil {
		return nil, errors.Wrap(err, "create signup runtime pool")
	}
	return &Executor{artifact: artifact, pool: pool}, nil
}

// Compile compiles a signup program against the host-owned signup input
// schemas without creating workers. Operational tools use this seam to
// validate and explain exactly the artifact a signup executor would activate.
func Compile(ctx context.Context, source string) (*idpscript.Artifact, error) {
	if source == "" {
		source = DefaultSource
	}
	options := idpscript.DefaultCompileOptions()
	options.SourceName = "open_signup.js"
	options.Schemas = schemas()
	artifact, err := idpscript.Compile(ctx, source, options)
	if err != nil {
		return nil, errors.Wrap(err, "compile signup program")
	}
	return artifact, nil
}

func (e *Executor) Close(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return nil
	}
	return e.pool.Close(ctx)
}

// PoolStats returns a bounded, non-secret operational snapshot. Hosts use it
// for readiness and metrics; it intentionally exposes neither JavaScript
// source nor any request-specific values.
func (e *Executor) PoolStats() idpscript.PoolStats {
	if e == nil || e.pool == nil {
		return idpscript.PoolStats{Closed: true}
	}
	return e.pool.Stats()
}

// Ready reports whether this executor still owns a warmed worker pool. A pool
// may have no idle workers while it is serving traffic, so saturation is an
// operational metric rather than a readiness failure.
func (e *Executor) Ready() bool {
	stats := e.PoolStats()
	return !stats.Closed && stats.Capacity > 0 && stats.WorkersCreated >= uint64(stats.Capacity)
}

func (e *Executor) Metrics() ExecutorMetrics {
	if e == nil {
		return ExecutorMetrics{}
	}
	return ExecutorMetrics{Invocations: e.metrics.invocations.Load(), Failures: e.metrics.failures.Load(), Present: e.metrics.present.Load(), Challenge: e.metrics.challenge.Load(), Commit: e.metrics.commit.Load(), Other: e.metrics.other.Load(), LatencyNanos: e.metrics.latencyNanos.Load(), Discarded: e.metrics.discarded.Load()}
}

func (e *Executor) Program() idpprogram.Program { return e.artifact.Program() }

// Fingerprint identifies the executable generation, not merely its declared
// program contract. A lambda body can change while handlers/schemas/effects
// stay identical, so both source and program fingerprints are persisted into
// continuations and used for generation routing.
func (e *Executor) Fingerprint() string {
	if e == nil || e.artifact == nil {
		return ""
	}
	fingerprints := e.artifact.Fingerprints()
	return strings.Join([]string{fingerprints.Source, fingerprints.Program}, ":")
}

func (e *Executor) ResolveProgram(_ context.Context, fingerprint string) (idpprogram.Program, error) {
	if e == nil || e.artifact == nil || fingerprint != e.Fingerprint() {
		return idpprogram.Program{}, errors.New("signup program generation is unavailable")
	}
	return e.Program(), nil
}

// RunTests executes declarative tests with no secrets and only six fixed,
// deterministic test fakes. Fakes are test data, never production capability
// bindings: a lambda must declare the matching capability and an invocation
// outside this runner never receives them.
func (e *Executor) RunTests(ctx context.Context) []TestResult {
	if e == nil || e.pool == nil {
		return []TestResult{{ID: "runner", Err: errors.New("signup executor is unavailable")}}
	}
	results := make([]TestResult, 0, len(e.Program().Tests))
	for _, test := range e.Program().Tests {
		capabilities, err := e.testCapabilities(test)
		if err != nil {
			results = append(results, TestResult{ID: test.ID, Expected: test.ExpectedKind, Err: err})
			continue
		}
		outcome, err := e.pool.Invoke(ctx, test.LambdaID, test.Input, capabilities)
		result := TestResult{ID: test.ID, Expected: test.ExpectedKind, Actual: outcome.Kind, Err: err}
		result.Passed = err == nil && outcome.Kind == test.ExpectedKind
		results = append(results, result)
	}
	return results
}

func (e *Executor) testCapabilities(test idpprogram.ProgramTest) (map[string]idpscript.CapabilityBinding, error) {
	lambda, ok := e.Program().Lambdas[test.LambdaID]
	if !ok {
		return nil, errors.Errorf("test lambda %q is unavailable", test.LambdaID)
	}
	bindings := map[string]idpscript.CapabilityBinding{}
	for _, requirement := range lambda.RequiredCapabilities {
		defaultOutput, known := deterministicTestFakeOutputs[requirement.ID]
		if !known {
			return nil, errors.Errorf("test capability %q has no deterministic fake", requirement.ID)
		}
		output := defaultOutput
		if configured, ok := test.Fakes[requirement.ID]; ok {
			output = configured
		}
		output = append(json.RawMessage(nil), output...)
		bindings[requirement.ID] = idpscript.CapabilityBinding{
			Requirement: requirement,
			Invoke: func(context.Context, json.RawMessage) (json.RawMessage, error) {
				return append(json.RawMessage(nil), output...), nil
			},
		}
	}
	return bindings, nil
}

func (e *Executor) Start(ctx context.Context, input StartInput) (idpworkflow.ValidatedPresentation, error) {
	if e == nil || e.pool == nil {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup executor is unavailable")
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return idpworkflow.ValidatedPresentation{}, errors.Wrap(err, "encode signup start input")
	}
	outcome, err := e.pool.Invoke(ctx, "signup.start", encoded, nil)
	if err != nil {
		return idpworkflow.ValidatedPresentation{}, errors.Wrap(err, "invoke signup start")
	}
	if outcome.Kind != idpprogram.OutcomePresent || outcome.Continuation == nil {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup start did not return a presentation")
	}
	presentation, err := idpworkflow.DecodePresentation(outcome.Presentation)
	if err != nil {
		return idpworkflow.ValidatedPresentation{}, err
	}
	if presentation.ResumeHandler != outcome.Continuation.HandlerID || presentation.ExpiresIn.Milliseconds() != outcome.Continuation.ExpiresIn*1000 {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup presentation continuation does not agree with outcome")
	}
	return idpworkflow.ValidatePresentation(e.Program(), WorkflowID, StartHandler, presentation, idpworkflow.DefaultRegistry(), idpworkflow.DefaultMaximumContinuationTTL)
}

func (e *Executor) Submit(ctx context.Context, values map[idpworkflow.FieldID]string, secrets map[string]idpworkflow.SecretHandle) (idpprogram.Outcome, error) {
	return e.SubmitWithEvidence(ctx, values, secrets, nil)
}

// SubmitWithEvidence is the only executor entry point that can project native
// verified evidence into a terminal signup handler. Callers supply JSON made
// by a native verifier; scripts cannot construct this invocation context.
func (e *Executor) SubmitWithEvidence(ctx context.Context, values map[idpworkflow.FieldID]string, secrets map[string]idpworkflow.SecretHandle, evidence map[string]json.RawMessage) (idpprogram.Outcome, error) {
	input, err := e.SubmissionInput(SubmittedHandler, values)
	if err != nil {
		return idpprogram.Outcome{}, err
	}
	return e.InvokeSubmission(ctx, SubmittedHandler, input, secrets, evidence)
}

// SubmissionInput projects only fields named by the selected handler's pinned
// object schema. A form can never smuggle a later-step password or code into
// a handler that did not declare it.
func (e *Executor) SubmissionInput(handler string, values map[idpworkflow.FieldID]string) (json.RawMessage, error) {
	if e == nil || e.artifact == nil {
		return nil, errors.New("signup executor is unavailable")
	}
	program := e.Program()
	workflow, ok := program.Workflows[WorkflowID]
	if !ok {
		return nil, errors.New("signup workflow is unavailable")
	}
	handlerSpec, ok := workflow.Handlers[handler]
	if !ok {
		return nil, errors.New("signup handler is unavailable")
	}
	lambda, ok := program.Lambdas[handlerSpec.LambdaID]
	if !ok {
		return nil, errors.New("signup handler lambda is unavailable")
	}
	schema, ok := program.Schemas[lambda.InputSchema]
	if !ok || schema.Kind != idpprogram.SchemaKindObject {
		return nil, errors.New("signup handler input schema is unavailable")
	}
	input := map[string]string{}
	for field := range schema.Fields {
		if value, ok := values[idpworkflow.FieldID(field)]; ok {
			input[field] = value
		}
	}
	return json.Marshal(input)
}

func (e *Executor) InvokeSubmission(ctx context.Context, handler string, input json.RawMessage, secrets map[string]idpworkflow.SecretHandle, evidence map[string]json.RawMessage) (idpprogram.Outcome, error) {
	if e == nil || e.pool == nil {
		return idpprogram.Outcome{}, errors.New("signup executor is unavailable")
	}
	workflow, ok := e.Program().Workflows[WorkflowID]
	if !ok {
		return idpprogram.Outcome{}, errors.New("signup workflow is unavailable")
	}
	handlerSpec, ok := workflow.Handlers[handler]
	if !ok {
		return idpprogram.Outcome{}, errors.New("signup handler is unavailable")
	}
	before := e.pool.Stats().Discarded
	started := time.Now()
	outcome, err := e.pool.InvokeWithSecretsAndEvidence(ctx, handlerSpec.LambdaID, input, nil, secrets, evidence)
	e.metrics.invocations.Add(1)
	e.metrics.latencyNanos.Add(uint64(time.Since(started)))
	after := e.pool.Stats().Discarded
	if after > before {
		e.metrics.discarded.Add(after - before)
	}
	if err != nil {
		e.metrics.failures.Add(1)
		return outcome, err
	}
	switch outcome.Kind {
	case idpprogram.OutcomePresent:
		e.metrics.present.Add(1)
	case idpprogram.OutcomeChallenge:
		e.metrics.challenge.Add(1)
	case idpprogram.OutcomeCommit:
		e.metrics.commit.Add(1)
	case idpprogram.OutcomeContinue, idpprogram.OutcomeComplete, idpprogram.OutcomeDeny, idpprogram.OutcomeSkip, idpprogram.OutcomeError:
		e.metrics.other.Add(1)
	}
	return outcome, nil
}

// Resume invokes the exact handler named by a validated continuation. The
// provider obtains that name from idpcontinuation, never from a browser form.
func (e *Executor) Resume(ctx context.Context, handler string, input json.RawMessage, evidence map[string]json.RawMessage) (idpprogram.Outcome, error) {
	if e == nil || e.pool == nil || handler == "" {
		return idpprogram.Outcome{}, errors.New("signup resume is unavailable")
	}
	return e.InvokeSubmission(ctx, handler, input, nil, evidence)
}

func schemas() map[string]idpprogram.Schema {
	return map[string]idpprogram.Schema{
		"signupStartInput": {ID: "signupStartInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 2048, Additional: false, Fields: map[string]idpprogram.SchemaField{
			"clientId":          {Ref: "signupText", Required: true},
			"redirectUri":       {Ref: "signupText", Required: true},
			"requestedScope":    {Ref: "signupText", Required: true},
			"interactionId":     {Ref: "signupText", Required: true},
			"hasBrowserSession": {Ref: "signupBool", Required: true},
		}},
		"signupSubmittedInput": {ID: "signupSubmittedInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Additional: false, Fields: map[string]idpprogram.SchemaField{
			"displayName": {Ref: "signupText"}, "email": {Ref: "signupEmail"}, "inviteCode": {Ref: "signupText"},
		}},
		"signupText":   {ID: "signupText", Kind: idpprogram.SchemaKindString, MaxBytes: 512, MaxLength: 120},
		"signupEmail":  {ID: "signupEmail", Kind: idpprogram.SchemaKindString, MaxBytes: 512, MaxLength: 320},
		"signupBool":   {ID: "signupBool", Kind: idpprogram.SchemaKindBoolean, MaxBytes: 5},
		"signupResult": {ID: "signupResult", Kind: idpprogram.SchemaKindObject, MaxBytes: 64, Additional: false, Fields: map[string]idpprogram.SchemaField{}},
	}
}
