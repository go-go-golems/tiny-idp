package idpscript_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

func TestPoolInvokesSynchronousLambdaWithFrozenInput(t *testing.T) {
	pool := newInvocationPool(t, 1)

	outcome, err := pool.Invoke(context.Background(), "test.sync", input("Ada"), nil)
	require.NoError(t, err)
	assert.Equal(t, "Ada", outcomeValue(t, outcome))
	assert.Equal(t, idpscript.PoolStats{Capacity: 1, Idle: 1, WorkersCreated: 1}, pool.Stats())
}

func TestPoolAwaitsPromiseCapability(t *testing.T) {
	pool := newInvocationPool(t, 1)

	outcome, err := pool.Invoke(context.Background(), "test.async", input("Ada"), map[string]idpscript.CapabilityBinding{
		"test.lookup": lookupCapability(func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
			var value map[string]string
			require.NoError(t, json.Unmarshal(input, &value))
			return json.RawMessage(fmt.Sprintf(`{"value":%q}`, "cap:"+value["value"])), nil
		}),
	})
	require.NoError(t, err)
	assert.Equal(t, "cap:Ada", outcomeValue(t, outcome))
}

func TestPoolBuildsDataOnlyPresentationOutcome(t *testing.T) {
	pool := newInvocationPool(t, 1)

	outcome, err := pool.Invoke(context.Background(), "test.present", input("Ada"), nil)
	require.NoError(t, err)
	require.Equal(t, idpprogram.OutcomePresent, outcome.Kind)
	require.NotNil(t, outcome.Continuation)
	assert.Equal(t, "submitted", outcome.Continuation.HandlerID)
	assert.JSONEq(t, `{"value":"Ada"}`, string(outcome.Continuation.Carry))
	assert.JSONEq(t, `{
		"title":"Create account",
		"resumeHandler":"submitted",
		"fields":["displayName","email"],
		"actions":["submit","deny"],
		"publicValues":{"displayName":"Ada"},
		"errors":[{"field":"email","code":"invalid"}],
		"carry":{"value":"Ada"},
		"expiresInSeconds":300
	}`, string(outcome.Presentation))
}

func TestInvocationCapabilityBudgetAndExpiredBindingFailClosed(t *testing.T) {
	pool := newInvocationPool(t, 1)
	capability := lookupCapability(func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"value":"ok"}`), nil
	})

	outcome, err := pool.Invoke(context.Background(), "test.budget", input("Ada"), map[string]idpscript.CapabilityBinding{"test.lookup": capability})
	require.NoError(t, err)
	assert.Equal(t, "budget_blocked", outcomeValue(t, outcome))

	_, err = pool.Invoke(context.Background(), "test.steal", input("Ada"), map[string]idpscript.CapabilityBinding{"test.lookup": capability})
	require.NoError(t, err)
	outcome, err = pool.Invoke(context.Background(), "test.reuse", input("Ada"), nil)
	require.NoError(t, err)
	assert.Equal(t, "expired_blocked", outcomeValue(t, outcome))
}

func TestCapabilityPanicBecomesCatchableRejection(t *testing.T) {
	pool := newInvocationPool(t, 1)
	capability := lookupCapability(func(context.Context, json.RawMessage) (json.RawMessage, error) {
		panic("backend exploded")
	})

	outcome, err := pool.Invoke(context.Background(), "test.capFailure", input("Ada"), map[string]idpscript.CapabilityBinding{"test.lookup": capability})
	require.NoError(t, err)
	assert.Equal(t, "capability_blocked", outcomeValue(t, outcome))
}

func TestUndeclaredAndMissingCapabilitiesFailClosed(t *testing.T) {
	pool := newInvocationPool(t, 1)
	capability := lookupCapability(func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
		return append(json.RawMessage(nil), input...), nil
	})

	_, err := pool.Invoke(context.Background(), "test.async", input("Ada"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `required capability "test.lookup" is not bound`)
	assert.Zero(t, pool.Stats().Discarded, "a pre-invocation binding error leaves the worker safe")

	outcome, err := pool.Invoke(context.Background(), "test.undeclared", input("Ada"), map[string]idpscript.CapabilityBinding{"test.lookup": capability})
	require.NoError(t, err)
	assert.Equal(t, "undeclared_blocked", outcomeValue(t, outcome))
}

func TestCallerCancellationDiscardsAndReplacesWorker(t *testing.T) {
	pool := newInvocationPool(t, 1)
	started := make(chan struct{})
	slow := slowCapability(func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := pool.Invoke(ctx, "test.slow", input("Ada"), map[string]idpscript.CapabilityBinding{"test.slow": slow})
		done <- err
	}()
	<-started
	cancel()
	err := <-done
	require.Error(t, err)
	assert.ErrorIs(t, err, idpscript.ErrInvocationCanceled)
	assert.Equal(t, uint64(1), pool.Stats().Discarded)
	assert.Equal(t, uint64(2), pool.Stats().WorkersCreated)
}

func TestActiveJavaScriptDeadlineInterruptDiscardsAndReplacesWorker(t *testing.T) {
	pool := newInvocationPool(t, 1)

	_, err := pool.Invoke(context.Background(), "test.spin", input("Ada"), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, idpscript.ErrInvocationTimeout)
	assert.Equal(t, uint64(1), pool.Stats().Discarded)
	assert.Equal(t, uint64(2), pool.Stats().WorkersCreated)

	outcome, err := pool.Invoke(context.Background(), "test.safe", input("Ada"), nil)
	require.NoError(t, err)
	assert.Equal(t, "safe", outcomeValue(t, outcome))
}

func TestTimeoutDiscardsWorkerAndLateSettlementCannotReachReplacement(t *testing.T) {
	pool := newInvocationPool(t, 1)
	release := make(chan struct{})
	slow := slowCapability(func(context.Context, json.RawMessage) (json.RawMessage, error) {
		<-release // Deliberately ignore cancellation to exercise late completion.
		return json.RawMessage(`{"value":"too-late"}`), nil
	})

	_, err := pool.Invoke(context.Background(), "test.timeoutCapability", input("Ada"), map[string]idpscript.CapabilityBinding{"test.slow": slow})
	require.Error(t, err)
	assert.ErrorIs(t, err, idpscript.ErrInvocationTimeout)
	assert.Equal(t, uint64(1), pool.Stats().Discarded)
	assert.Equal(t, uint64(2), pool.Stats().WorkersCreated)

	close(release)
	time.Sleep(5 * time.Millisecond)
	outcome, err := pool.Invoke(context.Background(), "test.safe", input("Ada"), nil)
	require.NoError(t, err)
	assert.Equal(t, "safe", outcomeValue(t, outcome))
}

func TestThrownAndInvalidOutputsDiscardWorkers(t *testing.T) {
	pool := newInvocationPool(t, 1)

	_, err := pool.Invoke(context.Background(), "test.throw", input("Ada"), nil)
	require.Error(t, err)
	assert.Equal(t, uint64(1), pool.Stats().Discarded)

	_, err = pool.Invoke(context.Background(), "test.invalid", input("Ada"), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, idpscript.ErrInvalidOutput)
	assert.Equal(t, uint64(2), pool.Stats().Discarded)
	assert.Equal(t, 1, pool.Stats().Capacity)
}

func TestPoolSaturationFailsClosed(t *testing.T) {
	pool := newInvocationPool(t, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	slow := slowCapability(func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		once.Do(func() { close(started) })
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return json.RawMessage(`{"value":"done"}`), nil
		}
	})

	group, groupCtx := errgroup.WithContext(context.Background())
	group.Go(func() error {
		_, err := pool.Invoke(groupCtx, "test.slow", input("Ada"), map[string]idpscript.CapabilityBinding{"test.slow": slow})
		return err
	})
	<-started
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, err := pool.Invoke(waitCtx, "test.safe", input("Ada"), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, idpscript.ErrRuntimeSaturated)
	close(release)
	require.NoError(t, group.Wait())
}

func TestPoolConcurrentInvocationsRemainExclusiveAndRaceSafe(t *testing.T) {
	pool := newInvocationPool(t, 2)
	capability := lookupCapability(func(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
		return append(json.RawMessage(nil), input...), nil
	})
	group, groupCtx := errgroup.WithContext(context.Background())
	for i := 0; i < 40; i++ {
		i := i
		group.Go(func() error {
			expected := fmt.Sprintf("user-%d", i)
			outcome, err := pool.Invoke(groupCtx, "test.async", input(expected), map[string]idpscript.CapabilityBinding{"test.lookup": capability})
			if err != nil {
				return err
			}
			if got := outcomeValueRaw(outcome); got != expected {
				return errors.Errorf("outcome value %q, want %q", got, expected)
			}
			return nil
		})
	}
	require.NoError(t, group.Wait())
	assert.Equal(t, idpscript.PoolStats{Capacity: 2, Idle: 2, WorkersCreated: 2}, pool.Stats())
}

func TestCompilerRejectsAllAmbientModuleFamilies(t *testing.T) {
	forbidden := []string{
		"fs", "node:fs", "exec", "database", "db", "os", "node:os",
		"process", "node:process", "fetch", "http", "https", "net",
		"child_process", "arbitrary/project/module",
	}
	for _, moduleName := range forbidden {
		moduleName := moduleName
		t.Run(moduleName, func(t *testing.T) {
			source := fmt.Sprintf(`require(%q);`, moduleName)
			_, err := idpscript.Compile(context.Background(), source, invocationCompileOptions())
			require.Error(t, err)
			message := err.Error()
			assert.True(t,
				strings.Contains(message, "ambient module") && strings.Contains(message, "disabled") ||
					strings.Contains(message, "No such built-in module"),
				"unexpected module-denial diagnostic: %s", message,
			)
		})
	}
}

func newInvocationPool(t *testing.T, size int) *idpscript.Pool {
	t.Helper()
	options := invocationCompileOptions()
	artifact, err := idpscript.Compile(context.Background(), invocationSource, options)
	require.NoError(t, err)
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	require.NoError(t, err)
	pool, err := idpscript.NewPool(context.Background(), artifact, factory, size)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close(context.Background())) })
	return pool
}

func invocationCompileOptions() idpscript.CompileOptions {
	options := idpscript.DefaultCompileOptions()
	options.Schemas = map[string]idpprogram.Schema{
		"input": {
			ID:       "input",
			Kind:     idpprogram.SchemaKindObject,
			MaxBytes: 1024,
			Fields: map[string]idpprogram.SchemaField{
				"value": {Ref: "text", Required: true},
			},
		},
		"result": {
			ID:       "result",
			Kind:     idpprogram.SchemaKindObject,
			MaxBytes: 1024,
			Fields: map[string]idpprogram.SchemaField{
				"value": {Ref: "text", Required: true},
			},
		},
		"text": {
			ID:        "text",
			Kind:      idpprogram.SchemaKindString,
			MaxBytes:  512,
			MaxLength: 128,
		},
	}
	return options
}

func input(value string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"value":%q}`, value))
}

func lookupCapability(invoke func(context.Context, json.RawMessage) (json.RawMessage, error)) idpscript.CapabilityBinding {
	return idpscript.CapabilityBinding{
		Requirement:    idpprogram.CapabilityRequirement{ID: "test.lookup", Version: 1},
		MaxInputBytes:  1024,
		MaxOutputBytes: 1024,
		Invoke:         invoke,
	}
}

func slowCapability(invoke func(context.Context, json.RawMessage) (json.RawMessage, error)) idpscript.CapabilityBinding {
	return idpscript.CapabilityBinding{
		Requirement:    idpprogram.CapabilityRequirement{ID: "test.slow", Version: 1},
		MaxInputBytes:  1024,
		MaxOutputBytes: 1024,
		Invoke:         invoke,
	}
}

func outcomeValue(t *testing.T, outcome idpprogram.Outcome) string {
	t.Helper()
	var value map[string]string
	require.NoError(t, json.Unmarshal(outcome.Value, &value))
	return value["value"]
}

func outcomeValueRaw(outcome idpprogram.Outcome) string {
	var value map[string]string
	if err := json.Unmarshal(outcome.Value, &value); err != nil {
		return ""
	}
	return value["value"]
}

const invocationSource = `
const A = require("tinyidp").v1;
let stolenLookup;
module.exports = A.program("invocation-tests", program => {
  program.capabilities({"test.lookup": {version: 1}, "test.slow": {version: 1}});

  function lambda(id, run, options = {}) {
    return A.lambda(id, {
      input: "input", output: "result", outcomes: ["complete"],
      effects: [], capabilities: options.capabilities || [],
      timeoutMs: options.timeoutMs || 200,
      maxCapabilityCalls: options.maxCapabilityCalls === undefined ? 0 : options.maxCapabilityCalls,
      maxOutputBytes: 1024, run,
    });
  }
  function workflow(name, handler) {
    program.workflow(name, {version: 1, entry: "run", handlers: {run: handler}, edges: []});
  }

  workflow("sync", lambda("test.sync", ctx => {
    const frozen = Object.isFrozen(ctx) && Object.isFrozen(ctx.input) && Object.isFrozen(ctx.cap);
    ctx.input.value = "mutated";
    return A.result.complete({value: frozen ? ctx.input.value : "not_frozen"});
  }));
  workflow("async", lambda("test.async", async ctx => {
    const found = await ctx.cap.test.lookup(ctx.input);
    return A.result.complete(found);
  }, {capabilities: ["test.lookup"], maxCapabilityCalls: 1}));
  workflow("budget", lambda("test.budget", async ctx => {
    await ctx.cap.test.lookup(ctx.input);
    try { await ctx.cap.test.lookup(ctx.input); }
    catch (_) { return A.result.complete({value: "budget_blocked"}); }
    return A.result.complete({value: "budget_bypassed"});
  }, {capabilities: ["test.lookup"], maxCapabilityCalls: 1}));
  workflow("steal", lambda("test.steal", async ctx => {
    stolenLookup = ctx.cap.test.lookup;
    await stolenLookup(ctx.input);
    return A.result.complete({value: "stored"});
  }, {capabilities: ["test.lookup"], maxCapabilityCalls: 1}));
  workflow("reuse", lambda("test.reuse", async ctx => {
    try { await stolenLookup(ctx.input); }
    catch (_) { return A.result.complete({value: "expired_blocked"}); }
    return A.result.complete({value: "expired_bypassed"});
  }));
  workflow("capFailure", lambda("test.capFailure", async ctx => {
    try { await ctx.cap.test.lookup(ctx.input); }
    catch (_) { return A.result.complete({value: "capability_blocked"}); }
    return A.result.complete({value: "capability_bypassed"});
  }, {capabilities: ["test.lookup"], maxCapabilityCalls: 1}));
  workflow("undeclared", lambda("test.undeclared", async ctx => {
    try { await ctx.cap.test.lookup(ctx.input); }
    catch (_) { return A.result.complete({value: "undeclared_blocked"}); }
    return A.result.complete({value: "undeclared_bypassed"});
  }));
  workflow("timeoutCapability", lambda("test.timeoutCapability", async ctx => {
    const value = await ctx.cap.test.slow(ctx.input);
    return A.result.complete(value);
  }, {capabilities: ["test.slow"], maxCapabilityCalls: 1, timeoutMs: 15}));
  workflow("slow", lambda("test.slow", async ctx => {
    const value = await ctx.cap.test.slow(ctx.input);
    return A.result.complete(value);
  }, {capabilities: ["test.slow"], maxCapabilityCalls: 1, timeoutMs: 500}));
  workflow("safe", lambda("test.safe", _ => A.result.complete({value: "safe"})));
  const present = A.lambda("test.present", {
    input: "input", output: "result", outcomes: ["present"], effects: [], capabilities: [],
    timeoutMs: 200, maxCapabilityCalls: 0, maxOutputBytes: 1024,
    run: ctx => ctx.present.form({
      title: "Create account",
      resume: "submitted",
      fields: [A.field.displayName(), A.field.email()],
      actions: [A.action.submit(), A.action.deny()],
      values: {displayName: ctx.input.value},
      errors: [{field: A.field.email(), code: "invalid"}],
      carry: ctx.input,
      expiresInSeconds: 300,
    }),
  });
  const submitted = A.lambda("test.present.submitted", {
    input: "input", output: "result", outcomes: ["complete"], effects: [], capabilities: [],
    timeoutMs: 200, maxCapabilityCalls: 0, maxOutputBytes: 1024,
    run: ctx => A.result.complete(ctx.input),
  });
  program.workflow("present", {
    version: 1, entry: "start", handlers: {start: present, submitted},
    edges: [{from: "start", outcome: "present", to: "submitted", input: "input"}],
  });
  workflow("spin", lambda("test.spin", _ => { for (;;) {} }, {timeoutMs: 15}));
  workflow("throw", lambda("test.throw", _ => { throw new Error("secret exception text"); }));
  workflow("invalid", lambda("test.invalid", _ => undefined));
});`
