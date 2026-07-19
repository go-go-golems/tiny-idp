package idpscript

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const defaultCapabilityPayloadBytes = 16 << 10

// CapabilityBinding is one explicitly host-supplied bounded operation.
type CapabilityBinding struct {
	Requirement    idpprogram.CapabilityRequirement
	MaxInputBytes  int
	MaxOutputBytes int
	Invoke         func(context.Context, json.RawMessage) (json.RawMessage, error)
}

type invocationBindings struct {
	ctx          context.Context
	cancel       context.CancelFunc
	image        *RuntimeImage
	allowed      map[string]CapabilityBinding
	maxCalls     int64
	calls        atomic.Int64
	pending      atomic.Int64
	active       atomic.Bool
	settlements  errgroup.Group
	settlePeriod time.Duration
}

func newInvocationBindings(ctx context.Context, image *RuntimeImage, spec idpprogram.LambdaSpec, supplied map[string]CapabilityBinding) (*invocationBindings, error) {
	bindingCtx, cancel := context.WithCancel(ctx)
	bindings := &invocationBindings{
		ctx:          bindingCtx,
		cancel:       cancel,
		image:        image,
		allowed:      map[string]CapabilityBinding{},
		maxCalls:     int64(spec.Budget.MaxCapabilityCalls),
		settlePeriod: time.Millisecond,
	}
	bindings.active.Store(true)
	for _, requirement := range spec.RequiredCapabilities {
		binding, ok := supplied[requirement.ID]
		if !ok {
			bindings.close()
			return nil, errors.Errorf("required capability %q is not bound", requirement.ID)
		}
		if binding.Requirement.ID != requirement.ID || binding.Requirement.Version != requirement.Version {
			bindings.close()
			return nil, errors.Errorf("capability %q binding does not satisfy version %d", requirement.ID, requirement.Version)
		}
		if binding.Invoke == nil {
			bindings.close()
			return nil, errors.Errorf("capability %q invoke function is nil", requirement.ID)
		}
		if binding.MaxInputBytes <= 0 {
			binding.MaxInputBytes = defaultCapabilityPayloadBytes
		}
		if binding.MaxOutputBytes <= 0 {
			binding.MaxOutputBytes = defaultCapabilityPayloadBytes
		}
		bindings.allowed[requirement.ID] = binding
	}
	return bindings, nil
}

func (b *invocationBindings) close() {
	if b == nil {
		return
	}
	b.active.Store(false)
	if b.cancel != nil {
		b.cancel()
	}
}

func (b *invocationBindings) waitSettled(ctx context.Context) error {
	if b == nil {
		return nil
	}
	ticker := time.NewTicker(b.settlePeriod)
	defer ticker.Stop()
	for b.pending.Load() != 0 {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "wait for capability settlement")
		case <-ticker.C:
		}
	}
	return b.settlements.Wait()
}

func (b *invocationBindings) capabilityObject(vm *goja.Runtime) (*goja.Object, error) {
	root := vm.NewObject()
	for id, binding := range b.allowed {
		parts := strings.Split(id, ".")
		current := root
		for _, part := range parts[:len(parts)-1] {
			existing := current.Get(part)
			if existing == nil || goja.IsUndefined(existing) {
				nested := vm.NewObject()
				if err := current.Set(part, nested); err != nil {
					return nil, errors.Wrapf(err, "set capability namespace %q", part)
				}
				current = nested
				continue
			}
			object, ok := existing.(*goja.Object)
			if !ok {
				return nil, errors.Errorf("capability namespace collision at %q", part)
			}
			current = object
		}
		leaf := parts[len(parts)-1]
		bindingCopy := binding
		if err := current.Set(leaf, b.capabilityFunction(vm, bindingCopy)); err != nil {
			return nil, errors.Wrapf(err, "set capability %q", id)
		}
	}
	return root, nil
}

func (b *invocationBindings) capabilityFunction(vm *goja.Runtime, binding CapabilityBinding) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if !b.active.Load() {
			panic(vm.NewTypeError("capability binding is no longer active"))
		}
		if len(call.Arguments) != 1 {
			panic(vm.NewTypeError("capability %q requires exactly one argument", binding.Requirement.ID))
		}
		callNumber := b.calls.Add(1)
		if callNumber > b.maxCalls {
			panic(vm.NewTypeError("capability call budget exceeded"))
		}
		input, err := json.Marshal(call.Argument(0).Export())
		if err != nil {
			panic(vm.NewTypeError("encode capability input"))
		}
		if len(input) > binding.MaxInputBytes {
			panic(vm.NewTypeError("capability input exceeds byte limit"))
		}

		promise, resolve, reject := vm.NewPromise()
		b.pending.Add(1)
		b.settlements.Go(func() error {
			defer b.pending.Add(-1)
			output, invokeErr := invokeCapability(binding, b.ctx, append(json.RawMessage(nil), input...))
			if !b.active.Load() {
				return nil
			}
			if len(output) > binding.MaxOutputBytes {
				invokeErr = errors.New("capability output exceeds byte limit")
			}
			if invokeErr == nil {
				if len(output) == 0 {
					output = json.RawMessage("null")
				}
				var plain any
				if err := decodeSingleJSON(output, &plain); err != nil {
					invokeErr = errors.Wrap(err, "decode capability output")
				}
			}
			outputCopy := append(json.RawMessage(nil), output...)

			settled := make(chan struct{})
			postErr := b.image.runtime.Owner.Post(b.ctx, "tinyidp.capability.settle."+binding.Requirement.ID, func(_ context.Context, ownerVM *goja.Runtime) {
				defer close(settled)
				if !b.active.Load() {
					return
				}
				if invokeErr != nil {
					_ = reject(ownerVM.ToValue("capability_failed"))
					return
				}
				value, err := parseJSONValue(ownerVM, outputCopy)
				if err != nil {
					_ = reject(ownerVM.ToValue("capability_failed"))
					return
				}
				_ = resolve(value)
			})
			if postErr != nil {
				return nil
			}
			select {
			case <-settled:
			case <-b.ctx.Done():
			}
			return nil
		})
		return vm.ToValue(promise)
	}
}

// Named results allow panic recovery to convert a host capability panic into a
// rejected Promise without letting it cross the runtime ownership boundary.
func invokeCapability(binding CapabilityBinding, ctx context.Context, input json.RawMessage) (output json.RawMessage, err error) { //nolint:nonamedreturns
	defer func() {
		if recovered := recover(); recovered != nil {
			output = nil
			err = errors.Errorf("capability panicked: %v", recovered)
		}
	}()
	return binding.Invoke(ctx, input)
}

func validateCapabilityID(id string) error {
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return errors.Errorf("capability %q must contain a namespace", id)
	}
	for _, part := range parts {
		if part == "" {
			return errors.Errorf("capability %q contains an empty namespace segment", id)
		}
	}
	return nil
}

func validateCapabilityBindings(bindings map[string]CapabilityBinding) error {
	for id, binding := range bindings {
		if err := validateCapabilityID(id); err != nil {
			return err
		}
		if binding.Requirement.ID != id {
			return errors.Errorf("capability map key %q does not match requirement ID %q", id, binding.Requirement.ID)
		}
		if binding.Requirement.Version == 0 {
			return errors.Errorf("capability %q version must be greater than zero", id)
		}
	}
	return nil
}
