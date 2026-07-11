// Package gojaverify compiles verification-plan JavaScript in an isolated Goja
// runtime. Native Go executes the resulting plan; JavaScript never receives a
// provider, store, network, filesystem, clock, or assertion implementation.
package gojaverify

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules"

	_ "github.com/manuel/tinyidp/internal/gojamodules/verify"
	"github.com/manuel/tinyidp/pkg/verifyplan"
)

type Options struct {
	MaxSourceBytes int
	Timeout        time.Duration
	Limits         verifyplan.Limits
}

func DefaultOptions() Options {
	return Options{MaxSourceBytes: 64 << 10, Timeout: 250 * time.Millisecond, Limits: verifyplan.DefaultLimits()}
}

func Compile(ctx context.Context, source string, options Options) (verifyplan.Plan, error) {
	if ctx == nil {
		return verifyplan.Plan{}, fmt.Errorf("context is required")
	}
	defaults := DefaultOptions()
	if options.MaxSourceBytes == 0 {
		options.MaxSourceBytes = defaults.MaxSourceBytes
	}
	if options.Timeout == 0 {
		options.Timeout = defaults.Timeout
	}
	if options.Limits.MaxSuites == 0 {
		options.Limits = defaults.Limits
	}
	if len(source) == 0 || len(source) > options.MaxSourceBytes {
		return verifyplan.Plan{}, fmt.Errorf("verification source size %d outside 1..%d", len(source), options.MaxSourceBytes)
	}
	compileContext, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	vm := goja.New()
	registry := require.NewRegistry(require.WithLoader(func(path string) ([]byte, error) {
		return nil, fmt.Errorf("ambient module %q is disabled", path)
	}))
	nativeModule := modules.GetModule("tinyidp/verify")
	if nativeModule == nil {
		return verifyplan.Plan{}, fmt.Errorf("tinyidp/verify native module is not registered")
	}
	registry.RegisterNativeModule(nativeModule.Name(), nativeModule.Loader)
	registry.Enable(vm)
	moduleObject := vm.NewObject()
	_ = moduleObject.Set("exports", vm.NewObject())
	if err := vm.Set("module", moduleObject); err != nil {
		return verifyplan.Plan{}, err
	}
	if err := vm.Set("exports", moduleObject.Get("exports")); err != nil {
		return verifyplan.Plan{}, err
	}
	stopInterrupt := context.AfterFunc(compileContext, func() { vm.Interrupt(compileContext.Err()) })
	defer stopInterrupt()
	if _, err := vm.RunString(source); err != nil {
		return verifyplan.Plan{}, fmt.Errorf("compile verification JavaScript: %w", err)
	}
	encoded, err := json.Marshal(moduleObject.Get("exports").Export())
	if err != nil {
		return verifyplan.Plan{}, fmt.Errorf("encode compiled verification plan: %w", err)
	}
	var plan verifyplan.Plan
	if err := json.Unmarshal(encoded, &plan); err != nil {
		return verifyplan.Plan{}, fmt.Errorf("decode compiled verification plan: %w", err)
	}
	plan.BindSource([]byte(source))
	if err := plan.Validate(options.Limits); err != nil {
		return verifyplan.Plan{}, err
	}
	return plan, nil
}
