package idpscript

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/pkg/errors"

	tinyidpmodule "github.com/go-go-golems/tiny-idp/internal/gojamodules/tinyidp"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const collectorRuntimeKey = "tinyidp.scripting.collector.v1"

type tinyIDPModuleRegistrar struct {
	schemas map[string]idpprogram.Schema
}

var _ engine.RuntimeModuleRegistrar = (*tinyIDPModuleRegistrar)(nil)

func (*tinyIDPModuleRegistrar) ID() string { return "tinyidp:scripting:v1" }

func (r *tinyIDPModuleRegistrar) RegisterRuntimeModule(ctx *engine.RuntimeModuleRegistrationContext, registry *require.Registry) error {
	if ctx == nil || registry == nil {
		return errors.New("Tiny-IDP runtime module context and registry are required")
	}
	collector := tinyidpmodule.NewCollector(r.schemas)
	ctx.SetValue(collectorRuntimeKey, collector)
	registry.RegisterNativeModule(tinyidpmodule.Name, tinyidpmodule.NewLoader(collector))
	return nil
}

// NewRuntimeFactory creates an immutable engine factory that exposes only
// require("tinyidp"). Data-only defaults and ambient file loaders are disabled.
func NewRuntimeFactory(schemas map[string]idpprogram.Schema) (*engine.RuntimeFactory, error) {
	return engine.NewRuntimeFactoryBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(false),
		engine.WithRequireOptions(require.WithLoader(func(path string) ([]byte, error) {
			return nil, errors.Errorf("ambient module %q is disabled", path)
		})),
	).WithModules(&tinyIDPModuleRegistrar{schemas: schemas}).Build()
}

// RuntimeImage is one owned VM loaded from an artifact. Callback values remain
// private and may only be accessed inside Owner.Call.
type RuntimeImage struct {
	runtime      *engine.Runtime
	collector    *tinyidpmodule.Collector
	callbackIDs  []string
	program      idpprogram.Program
	fingerprints idpprogram.Fingerprints
}

// Load creates and verifies a fresh owned runtime for this artifact.
func (a *Artifact) Load(ctx context.Context, factory *engine.RuntimeFactory) (*RuntimeImage, error) {
	if a == nil || a.compiled == nil {
		return nil, errors.New("compiled artifact is required")
	}
	if factory == nil {
		return nil, errors.New("runtime factory is required")
	}
	return loadProgram(ctx, factory, a.compiled, &a.fingerprints)
}

func loadProgram(ctx context.Context, factory *engine.RuntimeFactory, compiled *goja.Program, expected *idpprogram.Fingerprints) (*RuntimeImage, error) {
	if ctx == nil {
		return nil, errors.New("runtime load context is required")
	}
	runtime, err := factory.NewRuntime(engine.WithStartupContext(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create owned Tiny-IDP runtime")
	}
	failed := true
	defer func() {
		if failed {
			_ = runtime.Close(context.Background())
		}
	}()

	collectorValue, ok := runtime.Value(collectorRuntimeKey)
	if !ok {
		return nil, errors.New("Tiny-IDP runtime collector was not installed")
	}
	collector, ok := collectorValue.(*tinyidpmodule.Collector)
	if !ok || collector == nil {
		return nil, errors.New("Tiny-IDP runtime collector has invalid type")
	}

	var exportedJSON []byte
	stopInterrupt := context.AfterFunc(ctx, func() {
		runtime.VM.Interrupt(ctx.Err())
	})
	_, runErr := runtime.Owner.Call(ctx, "tinyidp.load-artifact", func(_ context.Context, vm *goja.Runtime) (any, error) {
		moduleObject := vm.NewObject()
		if err := moduleObject.Set("exports", vm.NewObject()); err != nil {
			return nil, errors.Wrap(err, "set CommonJS exports")
		}
		if err := vm.Set("module", moduleObject); err != nil {
			return nil, errors.Wrap(err, "set CommonJS module")
		}
		if err := vm.Set("exports", moduleObject.Get("exports")); err != nil {
			return nil, errors.Wrap(err, "set CommonJS exports global")
		}
		if _, err := vm.RunProgram(compiled); err != nil {
			return nil, errors.Wrap(err, "run compiled Tiny-IDP program")
		}
		encoded, err := json.Marshal(moduleObject.Get("exports").Export())
		if err != nil {
			return nil, errors.Wrap(err, "encode module.exports")
		}
		exportedJSON = encoded
		return nil, nil
	})
	interrupted := !stopInterrupt()
	if interrupted {
		_, _ = runtime.Owner.Call(context.Background(), "tinyidp.clear-load-interrupt", func(_ context.Context, vm *goja.Runtime) (any, error) {
			vm.ClearInterrupt()
			return nil, nil
		})
	}
	if runErr != nil {
		return nil, errors.Wrap(runErr, "load Tiny-IDP artifact")
	}

	program, err := collector.Program()
	if err != nil {
		return nil, err
	}
	canonicalProgram, err := idpprogram.CanonicalJSON(program)
	if err != nil {
		return nil, err
	}
	var exportedProgram idpprogram.Program
	if err := json.Unmarshal(exportedJSON, &exportedProgram); err != nil {
		return nil, errors.Wrap(err, "decode module.exports as Tiny-IDP program")
	}
	canonicalExport, err := idpprogram.CanonicalJSON(exportedProgram)
	if err != nil {
		return nil, err
	}
	if string(canonicalExport) != string(canonicalProgram) {
		return nil, errors.New("module.exports must be the value returned by tinyidp.v1.program")
	}
	if diagnostics := idpprogram.Validate(program); diagnostics.HasErrors() {
		return nil, &ValidationError{Diagnostics: diagnostics}
	}
	fingerprints, err := idpprogram.ComputeFingerprints(nil, program)
	if err != nil {
		return nil, err
	}
	// Source is not available to a generic runtime load; preserve the artifact's
	// source identity when comparing and exposing the image.
	if expected != nil {
		fingerprints.Source = expected.Source
		if fingerprints.Program != expected.Program ||
			fingerprints.CallbackRegistry != expected.CallbackRegistry ||
			fingerprints.Schemas != expected.Schemas {
			return nil, errors.New("runtime callback registry or program fingerprint does not match artifact")
		}
	}

	callbackIDs := collector.CallbackIDs()
	sort.Strings(callbackIDs)
	if len(callbackIDs) != len(program.Lambdas) {
		return nil, errors.Errorf("runtime registered %d callbacks for %d lambda specs", len(callbackIDs), len(program.Lambdas))
	}
	for _, id := range callbackIDs {
		if _, ok := program.Lambdas[id]; !ok {
			return nil, errors.Errorf("runtime registered callback %q without a lambda spec", id)
		}
	}

	failed = false
	return &RuntimeImage{
		runtime:      runtime,
		collector:    collector,
		callbackIDs:  append([]string(nil), callbackIDs...),
		program:      program,
		fingerprints: fingerprints,
	}, nil
}

// Program returns a deep VM-independent copy.
func (r *RuntimeImage) Program() idpprogram.Program {
	if r == nil {
		return idpprogram.Program{}
	}
	encoded, err := idpprogram.CanonicalJSON(r.program)
	if err != nil {
		panic(err)
	}
	var ret idpprogram.Program
	if err := json.Unmarshal(encoded, &ret); err != nil {
		panic(err)
	}
	return ret
}

// Fingerprints returns the verified runtime identities.
func (r *RuntimeImage) Fingerprints() idpprogram.Fingerprints {
	if r == nil {
		return idpprogram.Fingerprints{}
	}
	return r.fingerprints
}

// CallbackIDs returns the VM-local callback names without exposing functions.
func (r *RuntimeImage) CallbackIDs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.callbackIDs...)
}

// Close terminates the owned runtime and its event loop.
func (r *RuntimeImage) Close(ctx context.Context) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	closeCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return r.runtime.Close(closeCtx)
}
