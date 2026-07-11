// Package verify registers the compile-only tinyidp/verify native module.
package verify

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"

	"github.com/manuel/tinyidp/pkg/verifyplan"
)

const Name = "tinyidp/verify"

type module struct{}

var _ modules.NativeModule = (*module)(nil)

func (*module) Name() string { return Name }

func (*module) Doc() string {
	return "Compile lower-camel JavaScript data into a validated tiny-idp verification plan. The module exposes no live provider capabilities."
}

func (*module) Loader(vm *goja.Runtime, moduleObject *goja.Object) {
	exports := moduleObject.Get("exports").(*goja.Object)
	v1 := vm.NewObject()
	_ = v1.Set("plan", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) != 1 {
			panic(vm.NewTypeError("plan(spec) requires exactly one argument"))
		}
		encoded, err := json.Marshal(call.Argument(0).Export())
		if err != nil {
			panic(vm.NewTypeError("encode verification plan: %s", err))
		}
		var plan verifyplan.Plan
		if err := json.Unmarshal(encoded, &plan); err != nil {
			panic(vm.NewTypeError("decode verification plan: %s", err))
		}
		if plan.SchemaVersion == "" {
			plan.SchemaVersion = verifyplan.SchemaVersion
		}
		if err := plan.Validate(verifyplan.DefaultLimits()); err != nil {
			panic(vm.NewTypeError("invalid verification plan: %s", err))
		}
		normalized, err := json.Marshal(plan)
		if err != nil {
			panic(vm.NewTypeError("normalize verification plan: %s", err))
		}
		var plain any
		if err := json.Unmarshal(normalized, &plain); err != nil {
			panic(vm.NewTypeError("normalize verification plan object: %s", err))
		}
		return vm.ToValue(plain)
	})
	if err := exports.Set("v1", v1); err != nil {
		panic(fmt.Errorf("set tinyidp/verify exports: %w", err))
	}
}

func init() { modules.Register(&module{}) }
