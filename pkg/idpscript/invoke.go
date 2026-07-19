package idpscript

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	tinyidpmodule "github.com/go-go-golems/tiny-idp/internal/gojamodules/tinyidp"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

var (
	ErrUnknownLambda      = errors.New("unknown lambda")
	ErrInvocationTimeout  = errors.New("lambda invocation timed out")
	ErrInvocationCanceled = errors.New("lambda invocation canceled")
	ErrInvalidOutput      = errors.New("invalid lambda output")
	ErrPromiseRejected    = errors.New("lambda promise rejected")
)

type worker struct {
	id    uint64
	image *RuntimeImage
}

type invocationStart struct {
	promise *goja.Promise
	encoded []byte
}

type promiseState struct {
	state   goja.PromiseState
	encoded []byte
}

// Named results let the interrupt-cleanup defer force discard and preserve a
// cleanup failure without obscuring an earlier invocation error.
func (w *worker) invoke(ctx context.Context, lambdaID string, input json.RawMessage, supplied map[string]CapabilityBinding) (outcome idpprogram.Outcome, safe bool, err error) { //nolint:nonamedreturns
	if w == nil || w.image == nil || w.image.runtime == nil {
		return idpprogram.Outcome{}, false, errors.New("runtime worker is not initialized")
	}
	program := w.image.program
	spec, ok := program.Lambdas[lambdaID]
	if !ok {
		return idpprogram.Outcome{}, true, errors.Wrapf(ErrUnknownLambda, "%q", lambdaID)
	}
	err = validateInput(program, spec, input)
	if err != nil {
		return idpprogram.Outcome{}, true, err
	}
	if err := validateCapabilityBindings(supplied); err != nil {
		return idpprogram.Outcome{}, true, err
	}

	invocationCtx, cancel := context.WithTimeout(ctx, spec.Budget.Timeout)
	defer cancel()
	bindings, err := newInvocationBindings(invocationCtx, w.image, spec, supplied)
	if err != nil {
		return idpprogram.Outcome{}, true, err
	}
	defer bindings.close()

	var interrupted atomic.Bool
	stopInterrupt := context.AfterFunc(invocationCtx, func() {
		interrupted.Store(true)
		w.image.runtime.VM.Interrupt(ErrInvocationTimeout)
	})
	defer func() {
		stopped := stopInterrupt()
		if !stopped || interrupted.Load() {
			safe = false
			clearCtx, clearCancel := context.WithTimeout(context.Background(), time.Second)
			defer clearCancel()
			_, clearErr := w.image.runtime.Owner.Call(clearCtx, "tinyidp.clear-invocation-interrupt", func(_ context.Context, vm *goja.Runtime) (any, error) {
				vm.ClearInterrupt()
				return nil, nil
			})
			if clearErr != nil && err == nil {
				err = errors.Wrap(clearErr, "clear invocation interrupt")
			}
		}
	}()

	ret, callErr := w.image.runtime.Owner.Call(invocationCtx, "tinyidp.invoke."+lambdaID, func(_ context.Context, vm *goja.Runtime) (any, error) {
		callback, ok := w.image.collector.Callback(lambdaID)
		if !ok {
			return nil, errors.Wrapf(ErrUnknownLambda, "%q", lambdaID)
		}
		capabilityObject, err := bindings.capabilityObject(vm)
		if err != nil {
			return nil, err
		}
		ctxObject := vm.NewObject()
		jsInput, err := parseJSONValue(vm, input)
		if err != nil {
			return nil, errors.Wrap(err, "create native JavaScript lambda input")
		}
		if err := ctxObject.Set("input", jsInput); err != nil {
			return nil, errors.Wrap(err, "set lambda input")
		}
		if err := ctxObject.Set("cap", capabilityObject); err != nil {
			return nil, errors.Wrap(err, "set lambda capabilities")
		}
		if err := ctxObject.Set("present", tinyidpmodule.NewPresentationContext(vm, w.image.collector)); err != nil {
			return nil, errors.Wrap(err, "set lambda presentation context")
		}
		if err := deepFreeze(vm, ctxObject); err != nil {
			return nil, errors.Wrap(err, "freeze lambda context")
		}
		value, err := callback(goja.Undefined(), ctxObject)
		if err != nil {
			return nil, errors.Wrap(err, "execute lambda")
		}
		if promise, ok := value.Export().(*goja.Promise); ok {
			return invocationStart{promise: promise}, nil
		}
		encoded, err := json.Marshal(value.Export())
		if err != nil {
			return nil, errors.Wrap(err, "encode synchronous lambda output")
		}
		return invocationStart{encoded: encoded}, nil
	})
	if callErr != nil {
		if invocationCtx.Err() != nil {
			return idpprogram.Outcome{}, false, invocationContextError(invocationCtx)
		}
		return idpprogram.Outcome{}, false, callErr
	}
	start := ret.(invocationStart)
	encoded := start.encoded
	if start.promise != nil {
		encoded, err = w.awaitPromise(invocationCtx, start.promise)
		if err != nil {
			if invocationCtx.Err() != nil {
				return idpprogram.Outcome{}, false, invocationContextError(invocationCtx)
			}
			return idpprogram.Outcome{}, false, err
		}
	}
	if err := bindings.waitSettled(invocationCtx); err != nil {
		return idpprogram.Outcome{}, false, err
	}
	outcome, err = decodeOutcome(program, spec, encoded)
	if err != nil {
		return idpprogram.Outcome{}, false, errors.Wrap(ErrInvalidOutput, err.Error())
	}
	return outcome, true, nil
}

func invocationContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return errors.Wrap(ErrInvocationCanceled, ctx.Err().Error())
	}
	return errors.Wrap(ErrInvocationTimeout, ctx.Err().Error())
}

func (w *worker) awaitPromise(ctx context.Context, promise *goja.Promise) ([]byte, error) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		ret, err := w.image.runtime.Owner.Call(ctx, "tinyidp.promise-state", func(_ context.Context, _ *goja.Runtime) (any, error) {
			snapshot := promiseState{state: promise.State()}
			if snapshot.state == goja.PromiseStateFulfilled {
				encoded, err := json.Marshal(promise.Result().Export())
				if err != nil {
					return nil, errors.Wrap(err, "encode fulfilled lambda output")
				}
				snapshot.encoded = encoded
			}
			return snapshot, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot := ret.(promiseState)
		switch snapshot.state {
		case goja.PromiseStateFulfilled:
			return snapshot.encoded, nil
		case goja.PromiseStateRejected:
			return nil, ErrPromiseRejected
		case goja.PromiseStatePending:
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// parseJSONValue crosses the host/guest boundary through JSON.parse so objects
// and arrays are ordinary JavaScript values rather than Goja host objects. This
// matters for both isolation and Object.freeze: host object fields cannot be
// made read-only by JavaScript.
func parseJSONValue(vm *goja.Runtime, encoded json.RawMessage) (goja.Value, error) {
	jsonObject := vm.Get("JSON").ToObject(vm)
	parse, ok := goja.AssertFunction(jsonObject.Get("parse"))
	if !ok {
		return nil, errors.New("JSON.parse is unavailable")
	}
	value, err := parse(jsonObject, vm.ToValue(string(encoded)))
	if err != nil {
		return nil, errors.Wrap(err, "parse JSON in JavaScript runtime")
	}
	return value, nil
}

func deepFreeze(vm *goja.Runtime, value goja.Value) error {
	object, ok := value.(*goja.Object)
	if !ok {
		return nil
	}
	for _, key := range object.Keys() {
		if err := deepFreeze(vm, object.Get(key)); err != nil {
			return err
		}
	}
	objectConstructor := vm.Get("Object").ToObject(vm)
	freeze, ok := goja.AssertFunction(objectConstructor.Get("freeze"))
	if !ok {
		return errors.New("Object.freeze is unavailable")
	}
	_, err := freeze(objectConstructor, object)
	return err
}
