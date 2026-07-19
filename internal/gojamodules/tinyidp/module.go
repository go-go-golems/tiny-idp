// Package tinyidp registers the isolated require("tinyidp") native module.
package tinyidp

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

// Name is the only native module exposed by the production policy compiler.
const Name = "tinyidp"

type module struct{}

var _ modules.NativeModule = (*module)(nil)

func (*module) Name() string { return Name }

func (*module) Doc() string {
	return "Build a bounded Tiny-IDP v1 program and register named lambdas. This module exposes no ambient host authority."
}

// Loader creates a standalone collector. Production compilation uses
// NewLoader so the host can retrieve the runtime-scoped callback registry.
func (*module) Loader(vm *goja.Runtime, moduleObject *goja.Object) {
	NewLoader(NewCollector(nil))(vm, moduleObject)
}

func init() { modules.Register(&module{}) }

// Collector owns the program drafts and VM-local callbacks registered by one
// execution of one source artifact.
type Collector struct {
	schemas   map[string]idpprogram.Schema
	program   *idpprogram.Program
	lambdas   map[string]lambdaDraft
	callbacks map[string]goja.Callable
	handles   map[*goja.Object]string
	fields    map[*goja.Object]idpworkflow.FieldID
	actions   map[*goja.Object]idpworkflow.ActionID
}

type lambdaDraft struct {
	spec          idpprogram.LambdaSpec
	capabilityIDs []string
}

// NewCollector returns an empty runtime-scoped collector. Schemas are copied so
// caller mutation cannot change the materialized program.
func NewCollector(schemas map[string]idpprogram.Schema) *Collector {
	return &Collector{
		schemas:   cloneSchemas(schemas),
		lambdas:   map[string]lambdaDraft{},
		callbacks: map[string]goja.Callable{},
		handles:   map[*goja.Object]string{},
		fields:    map[*goja.Object]idpworkflow.FieldID{},
		actions:   map[*goja.Object]idpworkflow.ActionID{},
	}
}

// Callback returns a VM-owned callback. Callers must invoke this only while
// owning the collector's runtime.
func (c *Collector) Callback(id string) (goja.Callable, bool) {
	callback, ok := c.callbacks[id]
	return callback, ok
}

// Program returns a deep, VM-independent copy after A.program has completed.
func (c *Collector) Program() (idpprogram.Program, error) {
	if c == nil || c.program == nil {
		return idpprogram.Program{}, errors.New("tinyidp program was not registered")
	}
	encoded, err := json.Marshal(c.program)
	if err != nil {
		return idpprogram.Program{}, errors.Wrap(err, "encode collected program")
	}
	var ret idpprogram.Program
	if err := json.Unmarshal(encoded, &ret); err != nil {
		return idpprogram.Program{}, errors.Wrap(err, "decode collected program")
	}
	return ret, nil
}

// CallbackIDs returns a copy of the registered callback IDs.
func (c *Collector) CallbackIDs() []string {
	ret := make([]string, 0, len(c.callbacks))
	for id := range c.callbacks {
		ret = append(ret, id)
	}
	return ret
}

// NewLoader returns a native module loader bound to exactly one runtime-scoped
// collector.
func NewLoader(collector *Collector) func(*goja.Runtime, *goja.Object) {
	if collector == nil {
		panic("tinyidp module collector is nil")
	}
	return func(vm *goja.Runtime, moduleObject *goja.Object) {
		exports := moduleObject.Get("exports").(*goja.Object)
		v1 := vm.NewObject()
		lambdaFunction := newLambdaFunction(vm, collector)
		mustSet(vm, v1, "lambda", lambdaFunction)
		mustSet(vm, v1, "program", newProgramFunction(vm, collector, lambdaFunction))
		mustSet(vm, v1, "result", newResultBuilders(vm))
		mustSet(vm, v1, "field", newFieldBuilders(vm, collector))
		mustSet(vm, v1, "action", newActionBuilders(vm, collector))
		mustSet(vm, exports, "v1", v1)
	}
}

func newLambdaFunction(vm *goja.Runtime, collector *Collector) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 2, "lambda(id, spec)")
		id := requireString(vm, call.Argument(0), "lambda id")
		if _, exists := collector.lambdas[id]; exists {
			panic(vm.NewTypeError("duplicate lambda %q", id))
		}
		specObject := requireObject(vm, call.Argument(1), "lambda spec")
		run, ok := goja.AssertFunction(specObject.Get("run"))
		if !ok {
			panic(vm.NewTypeError("lambda %q run must be a function", id))
		}
		draft := lambdaDraft{
			spec: idpprogram.LambdaSpec{
				ID:              id,
				Kind:            parseLambdaKind(vm, specObject.Get("kind")),
				InputSchema:     requireString(vm, specObject.Get("input"), "lambda input schema"),
				OutputSchema:    requireString(vm, specObject.Get("output"), "lambda output schema"),
				AllowedOutcomes: parseOutcomeKinds(vm, specObject.Get("outcomes")),
				AllowedEffects:  parseEffectKinds(vm, specObject.Get("effects")),
				Budget: idpprogram.InvocationBudget{
					Timeout:            time.Duration(requirePositiveInteger(vm, specObject.Get("timeoutMs"), "timeoutMs")) * time.Millisecond,
					MaxCapabilityCalls: requireNonNegativeInteger(vm, specObject.Get("maxCapabilityCalls"), "maxCapabilityCalls"),
					MaxOutputBytes:     requirePositiveInteger(vm, specObject.Get("maxOutputBytes"), "maxOutputBytes"),
				},
			},
			capabilityIDs: parseStringArray(vm, specObject.Get("capabilities"), "capabilities", true),
		}
		collector.lambdas[id] = draft
		collector.callbacks[id] = run
		handle := vm.NewObject()
		collector.handles[handle] = id
		return handle
	}
}

func newProgramFunction(vm *goja.Runtime, collector *Collector, lambdaFunction func(goja.FunctionCall) goja.Value) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 2, "program(name, define)")
		if collector.program != nil {
			panic(vm.NewTypeError("only one tinyidp program may be registered"))
		}
		name := requireString(vm, call.Argument(0), "program name")
		define, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			panic(vm.NewTypeError("program define must be a function"))
		}

		program := &idpprogram.Program{
			APIVersion:   idpprogram.APIVersionV1,
			Name:         name,
			Workflows:    map[string]idpprogram.Workflow{},
			Providers:    map[string]idpprogram.Provider{},
			Lambdas:      map[string]idpprogram.LambdaSpec{},
			Schemas:      cloneSchemas(collector.schemas),
			Capabilities: map[string]idpprogram.CapabilityRequirement{},
		}
		builder := vm.NewObject()
		mustSet(vm, builder, "lambda", lambdaFunction)
		mustSet(vm, builder, "capabilities", newCapabilitiesFunction(vm, program))
		mustSet(vm, builder, "workflow", newWorkflowFunction(vm, collector, program))
		mustSet(vm, builder, "provider", newProviderFunction(vm, collector, program))
		if _, err := define(goja.Undefined(), builder); err != nil {
			panic(err)
		}

		for id, draft := range collector.lambdas {
			for _, capabilityID := range draft.capabilityIDs {
				capability, ok := program.Capabilities[capabilityID]
				if !ok {
					panic(vm.NewTypeError("lambda %q references undeclared capability %q", id, capabilityID))
				}
				draft.spec.RequiredCapabilities = append(draft.spec.RequiredCapabilities, capability)
			}
			program.Lambdas[id] = draft.spec
		}
		collector.program = program
		return normalizedValue(vm, program)
	}
}

func newProviderFunction(vm *goja.Runtime, collector *Collector, program *idpprogram.Program) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 3, "provider(kind, name, spec)")
		kind := idpprogram.ProviderKind(requireString(vm, call.Argument(0), "provider kind"))
		name := requireString(vm, call.Argument(1), "provider name")
		if !kind.Valid() {
			panic(vm.NewTypeError("unsupported provider kind %q", kind))
		}
		providerID := string(kind) + "." + name
		if _, exists := program.Providers[providerID]; exists {
			panic(vm.NewTypeError("duplicate provider %q", providerID))
		}
		spec := requireObject(vm, call.Argument(2), "provider spec")
		handlers := requireObject(vm, spec.Get("handlers"), "provider handlers")
		provider := idpprogram.Provider{
			ID:               providerID,
			Kind:             kind,
			Version:          requirePositiveUint32(vm, spec.Get("version"), "provider version"),
			State:            idpprogram.ProviderState(requireString(vm, spec.Get("state"), "provider state")),
			ReplayProtection: idpprogram.ReplayProtection(requireString(vm, spec.Get("replayProtection"), "provider replay protection")),
			Revocation:       idpprogram.RevocationMode(requireString(vm, spec.Get("revocation"), "provider revocation")),
			Handlers:         map[string]idpprogram.ProviderHandler{},
		}
		for _, handlerID := range handlers.Keys() {
			handle := requireObject(vm, handlers.Get(handlerID), "provider handler")
			lambdaID, ok := collector.handles[handle]
			if !ok {
				panic(vm.NewTypeError("provider handler %q is not a lambda returned by this module", handlerID))
			}
			draft, ok := collector.lambdas[lambdaID]
			if !ok {
				panic(vm.NewTypeError("provider handler %q has no registered lambda", handlerID))
			}
			provider.Handlers[handlerID] = idpprogram.ProviderHandler{ID: handlerID, LambdaID: lambdaID, InputSchema: draft.spec.InputSchema, OutputSchema: draft.spec.OutputSchema}
		}
		program.Providers[providerID] = provider
		return goja.Undefined()
	}
}

func newCapabilitiesFunction(vm *goja.Runtime, program *idpprogram.Program) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 1, "capabilities(requirements)")
		requirements := requireObject(vm, call.Argument(0), "capability requirements")
		for _, id := range requirements.Keys() {
			if _, exists := program.Capabilities[id]; exists {
				panic(vm.NewTypeError("duplicate capability %q", id))
			}
			spec := requireObject(vm, requirements.Get(id), "capability requirement")
			version := requirePositiveUint32(vm, spec.Get("version"), "capability version")
			program.Capabilities[id] = idpprogram.CapabilityRequirement{ID: id, Version: version}
		}
		return goja.Undefined()
	}
}

func newWorkflowFunction(vm *goja.Runtime, collector *Collector, program *idpprogram.Program) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 2, "workflow(name, spec)")
		id := requireString(vm, call.Argument(0), "workflow name")
		if _, exists := program.Workflows[id]; exists {
			panic(vm.NewTypeError("duplicate workflow %q", id))
		}
		spec := requireObject(vm, call.Argument(1), "workflow spec")
		handlersObject := requireObject(vm, spec.Get("handlers"), "workflow handlers")
		workflow := idpprogram.Workflow{
			ID:           id,
			Version:      requirePositiveUint32(vm, spec.Get("version"), "workflow version"),
			EntryHandler: requireString(vm, spec.Get("entry"), "workflow entry"),
			Handlers:     map[string]idpprogram.HandlerSpec{},
		}
		for _, handlerID := range handlersObject.Keys() {
			handle := requireObject(vm, handlersObject.Get(handlerID), "workflow handler")
			lambdaID, ok := collector.handles[handle]
			if !ok {
				panic(vm.NewTypeError("workflow handler %q is not a lambda returned by this module", handlerID))
			}
			workflow.Handlers[handlerID] = idpprogram.HandlerSpec{ID: handlerID, LambdaID: lambdaID}
		}
		parseWorkflowEdges(vm, spec.Get("edges"), &workflow)
		program.Workflows[id] = workflow
		return goja.Undefined()
	}
}

func parseWorkflowEdges(vm *goja.Runtime, value goja.Value, workflow *idpprogram.Workflow) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return
	}
	array := requireArray(vm, value, "workflow edges")
	for i := int64(0); i < array.Get("length").ToInteger(); i++ {
		edgeObject := requireObject(vm, array.Get(fmt.Sprintf("%d", i)), "workflow edge")
		from := requireString(vm, edgeObject.Get("from"), "edge from")
		handler, ok := workflow.Handlers[from]
		if !ok {
			panic(vm.NewTypeError("edge references unknown source handler %q", from))
		}
		handler.ContinuationEdges = append(handler.ContinuationEdges, idpprogram.ContinuationEdge{
			OutcomeKind: idpprogram.OutcomeKind(requireString(vm, edgeObject.Get("outcome"), "edge outcome")),
			HandlerID:   requireString(vm, edgeObject.Get("to"), "edge destination"),
			InputSchema: requireString(vm, edgeObject.Get("input"), "edge input schema"),
		})
		workflow.Handlers[from] = handler
	}
}

func newResultBuilders(vm *goja.Runtime) *goja.Object {
	result := vm.NewObject()
	plain := func(kind idpprogram.OutcomeKind) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			value := map[string]any{"kind": kind}
			if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) {
				value["value"] = call.Argument(0).Export()
			}
			return vm.ToValue(value)
		}
	}
	code := func(kind idpprogram.OutcomeKind, optional bool) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			if optional && len(call.Arguments) == 0 {
				return vm.ToValue(map[string]any{"kind": kind})
			}
			requireArgumentCount(vm, call, 1, string(kind)+"(code)")
			return vm.ToValue(map[string]any{"kind": kind, "code": requireString(vm, call.Argument(0), "result code")})
		}
	}
	browser := func(kind idpprogram.OutcomeKind) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			requireArgumentCount(vm, call, 1, string(kind)+"(spec)")
			spec := requireObject(vm, call.Argument(0), string(kind)+" spec")
			continuation := map[string]any{
				"handlerId":        requireString(vm, spec.Get("handler"), "continuation handler"),
				"expiresInSeconds": requirePositiveInteger(vm, spec.Get("expiresInSeconds"), "continuation expiry"),
			}
			if carry := spec.Get("carry"); !goja.IsUndefined(carry) {
				continuation["carry"] = carry.Export()
			}
			return vm.ToValue(map[string]any{"kind": kind, "continuation": continuation})
		}
	}
	mustSet(vm, result, "continue", func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 1, "continue(handler)")
		return vm.ToValue(map[string]any{"kind": idpprogram.OutcomeContinue, "nextHandler": requireString(vm, call.Argument(0), "next handler")})
	})
	mustSet(vm, result, "present", browser(idpprogram.OutcomePresent))
	mustSet(vm, result, "challenge", browser(idpprogram.OutcomeChallenge))
	mustSet(vm, result, "commit", func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 1, "commit(effects)")
		effects := requireArray(vm, call.Argument(0), "commit effects")
		return vm.ToValue(map[string]any{"kind": idpprogram.OutcomeCommit, "effects": effects.Export()})
	})
	mustSet(vm, result, "complete", plain(idpprogram.OutcomeComplete))
	mustSet(vm, result, "deny", code(idpprogram.OutcomeDeny, false))
	mustSet(vm, result, "skip", code(idpprogram.OutcomeSkip, true))
	mustSet(vm, result, "error", code(idpprogram.OutcomeError, false))
	return result
}

// newFieldBuilders exposes only host-defined descriptor identities. A script
// may select a field for a presentation but cannot define an input name, HTML,
// sensitive-data policy, or normalizer.
func newFieldBuilders(vm *goja.Runtime, collector *Collector) *goja.Object {
	builders := vm.NewObject()
	for _, id := range []idpworkflow.FieldID{
		idpworkflow.FieldDisplayName,
		idpworkflow.FieldEmail,
		idpworkflow.FieldPassword,
		idpworkflow.FieldPasswordConfirmation,
		idpworkflow.FieldInviteCode,
	} {
		fieldID := id
		mustSet(vm, builders, string(fieldID), func(call goja.FunctionCall) goja.Value {
			requireArgumentCount(vm, call, 0, "field."+string(fieldID)+"()")
			handle := vm.NewObject()
			collector.fields[handle] = fieldID
			return handle
		})
	}
	return builders
}

// newActionBuilders exposes only host-defined action identities. The host
// retains the action label and its form-validation policy.
func newActionBuilders(vm *goja.Runtime, collector *Collector) *goja.Object {
	builders := vm.NewObject()
	for _, id := range []idpworkflow.ActionID{idpworkflow.ActionSubmit, idpworkflow.ActionDeny} {
		actionID := id
		mustSet(vm, builders, string(actionID), func(call goja.FunctionCall) goja.Value {
			requireArgumentCount(vm, call, 0, "action."+string(actionID)+"()")
			handle := vm.NewObject()
			collector.actions[handle] = actionID
			return handle
		})
	}
	return builders
}

// NewPresentationContext creates the invocation-scoped browser boundary. It
// deliberately returns data, not a response writer, request, template, or
// redirect mutator. Native HTTP code validates and projects the result later.
func NewPresentationContext(vm *goja.Runtime, collector *Collector) *goja.Object {
	if collector == nil {
		panic("tinyidp presentation collector is nil")
	}
	present := vm.NewObject()
	mustSet(vm, present, "form", func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 1, "ctx.present.form(spec)")
		spec := requireObject(vm, call.Argument(0), "presentation spec")
		carry := spec.Get("carry")
		if goja.IsUndefined(carry) {
			panic(vm.NewTypeError("presentation carry is required"))
		}
		fields := presentationFieldIDs(vm, collector, spec.Get("fields"))
		actions := presentationActionIDs(vm, collector, spec.Get("actions"))
		publicValues := presentationPublicValues(vm, spec.Get("values"))
		errors := presentationErrors(vm, collector, spec.Get("errors"))
		expiresInSeconds := requirePositiveInteger(vm, spec.Get("expiresInSeconds"), "presentation expiry")
		resumeHandler := requireString(vm, spec.Get("resume"), "presentation resume handler")
		title := requireString(vm, spec.Get("title"), "presentation title")
		presentation := map[string]any{
			"title":            title,
			"resumeHandler":    resumeHandler,
			"fields":           fields,
			"actions":          actions,
			"carry":            carry.Export(),
			"expiresInSeconds": expiresInSeconds,
		}
		if len(publicValues) != 0 {
			presentation["publicValues"] = publicValues
		}
		if len(errors) != 0 {
			presentation["errors"] = errors
		}
		return vm.ToValue(map[string]any{
			"kind": idpprogram.OutcomePresent,
			"continuation": map[string]any{
				"handlerId":        resumeHandler,
				"carry":            carry.Export(),
				"expiresInSeconds": expiresInSeconds,
			},
			"presentation": presentation,
		})
	})
	return present
}

// InvocationSecrets binds opaque native secret handles to one owned VM call.
// Its objects have no serializable properties and are accepted only by the
// matching ctx.commit builder.
type InvocationSecrets struct {
	context *goja.Object
	handles map[*goja.Object]idpworkflow.SecretHandle
}

func NewInvocationSecrets(vm *goja.Runtime, secrets map[string]idpworkflow.SecretHandle) *InvocationSecrets {
	bindings := &InvocationSecrets{context: vm.NewObject(), handles: map[*goja.Object]idpworkflow.SecretHandle{}}
	for name, handle := range secrets {
		if handle.Token() == "" {
			panic(vm.NewTypeError("secret handle %q is invalid", name))
		}
		object := vm.NewObject()
		bindings.handles[object] = handle
		mustSet(vm, bindings.context, name, object)
	}
	return bindings
}

func (s *InvocationSecrets) Context() *goja.Object {
	if s == nil || s.context == nil {
		panic("tinyidp invocation secrets are nil")
	}
	return s.context
}

// NewCommitContext exposes the closed native signup commit request. It accepts
// opaque secret object identities, not string handles or secret bytes, and
// returns effect plans for trusted Go to validate and commit atomically.
func NewCommitContext(vm *goja.Runtime, secrets *InvocationSecrets) *goja.Object {
	commit := vm.NewObject()
	mustSet(vm, commit, "signup", func(call goja.FunctionCall) goja.Value {
		requireArgumentCount(vm, call, 1, "ctx.commit.signup(spec)")
		spec := requireObject(vm, call.Argument(0), "signup commit spec")
		password := requireSecretHandle(vm, secrets, spec.Get("password"), "signup password")
		confirmation := requireSecretHandle(vm, secrets, spec.Get("passwordConfirmation"), "signup password confirmation")
		return vm.ToValue(map[string]any{
			"kind": idpprogram.OutcomeCommit,
			"effects": []map[string]any{
				{"kind": idpprogram.EffectCreateLocalIdentity, "payload": map[string]any{
					"login":       requireString(vm, spec.Get("login"), "signup login"),
					"displayName": requireString(vm, spec.Get("displayName"), "signup display name"),
				}},
				{"kind": idpprogram.EffectAttachPasswordCredential, "payload": map[string]any{
					"passwordHandle":             password.Token(),
					"passwordConfirmationHandle": confirmation.Token(),
				}},
			},
		})
	})
	return commit
}

func requireSecretHandle(vm *goja.Runtime, secrets *InvocationSecrets, value goja.Value, name string) idpworkflow.SecretHandle {
	if secrets == nil {
		panic(vm.NewTypeError("%s is unavailable", name))
	}
	object := requireObject(vm, value, name)
	handle, ok := secrets.handles[object]
	if !ok {
		panic(vm.NewTypeError("%s must be an invocation-scoped secret handle", name))
	}
	return handle
}

func presentationFieldIDs(vm *goja.Runtime, collector *Collector, value goja.Value) []string {
	array := requireArray(vm, value, "presentation fields")
	ret := make([]string, 0, array.Get("length").ToInteger())
	for i := int64(0); i < array.Get("length").ToInteger(); i++ {
		handle := requireObject(vm, array.Get(fmt.Sprintf("%d", i)), "presentation field")
		id, ok := collector.fields[handle]
		if !ok {
			panic(vm.NewTypeError("presentation field is not a field returned by this module"))
		}
		ret = append(ret, string(id))
	}
	return ret
}

func presentationActionIDs(vm *goja.Runtime, collector *Collector, value goja.Value) []string {
	array := requireArray(vm, value, "presentation actions")
	ret := make([]string, 0, array.Get("length").ToInteger())
	for i := int64(0); i < array.Get("length").ToInteger(); i++ {
		handle := requireObject(vm, array.Get(fmt.Sprintf("%d", i)), "presentation action")
		id, ok := collector.actions[handle]
		if !ok {
			panic(vm.NewTypeError("presentation action is not an action returned by this module"))
		}
		ret = append(ret, string(id))
	}
	return ret
}

func presentationPublicValues(vm *goja.Runtime, value goja.Value) map[string]string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	object := requireObject(vm, value, "presentation values")
	ret := make(map[string]string, len(object.Keys()))
	for _, key := range object.Keys() {
		ret[key] = requireString(vm, object.Get(key), "presentation public value")
	}
	return ret
}

func presentationErrors(vm *goja.Runtime, collector *Collector, value goja.Value) []map[string]string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	array := requireArray(vm, value, "presentation errors")
	ret := make([]map[string]string, 0, array.Get("length").ToInteger())
	for i := int64(0); i < array.Get("length").ToInteger(); i++ {
		errorObject := requireObject(vm, array.Get(fmt.Sprintf("%d", i)), "presentation error")
		field := requireObject(vm, errorObject.Get("field"), "presentation error field")
		fieldID, ok := collector.fields[field]
		if !ok {
			panic(vm.NewTypeError("presentation error field is not a field returned by this module"))
		}
		ret = append(ret, map[string]string{
			"field": string(fieldID),
			"code":  requireString(vm, errorObject.Get("code"), "presentation error code"),
		})
	}
	return ret
}

func parseLambdaKind(vm *goja.Runtime, value goja.Value) idpprogram.LambdaKind {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return idpprogram.LambdaKindWorkflow
	}
	return idpprogram.LambdaKind(requireString(vm, value, "lambda kind"))
}

func parseOutcomeKinds(vm *goja.Runtime, value goja.Value) []idpprogram.OutcomeKind {
	values := parseStringArray(vm, value, "outcomes", false)
	ret := make([]idpprogram.OutcomeKind, 0, len(values))
	for _, value := range values {
		ret = append(ret, idpprogram.OutcomeKind(value))
	}
	return ret
}

func parseEffectKinds(vm *goja.Runtime, value goja.Value) []idpprogram.EffectKind {
	values := parseStringArray(vm, value, "effects", true)
	ret := make([]idpprogram.EffectKind, 0, len(values))
	for _, value := range values {
		ret = append(ret, idpprogram.EffectKind(value))
	}
	return ret
}

func parseStringArray(vm *goja.Runtime, value goja.Value, name string, optional bool) []string {
	if optional && (value == nil || goja.IsUndefined(value) || goja.IsNull(value)) {
		return nil
	}
	array := requireArray(vm, value, name)
	length := array.Get("length").ToInteger()
	ret := make([]string, 0, length)
	for i := int64(0); i < length; i++ {
		ret = append(ret, requireString(vm, array.Get(fmt.Sprintf("%d", i)), name+" item"))
	}
	return ret
}

func requireArray(vm *goja.Runtime, value goja.Value, name string) *goja.Object {
	object := requireObject(vm, value, name)
	if object.ClassName() != "Array" {
		panic(vm.NewTypeError("%s must be an array", name))
	}
	return object
}

func requireObject(vm *goja.Runtime, value goja.Value, name string) *goja.Object {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("%s is required", name))
	}
	object := value.ToObject(vm)
	if object == nil {
		panic(vm.NewTypeError("%s must be an object", name))
	}
	return object
}

func requireString(vm *goja.Runtime, value goja.Value, name string) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("%s must be a non-empty string", name))
	}
	ret, ok := value.Export().(string)
	if !ok || ret == "" {
		panic(vm.NewTypeError("%s must be a non-empty string", name))
	}
	return ret
}

func requirePositiveInteger(vm *goja.Runtime, value goja.Value, name string) int {
	ret := value.ToFloat()
	if math.IsNaN(ret) || math.IsInf(ret, 0) || math.Trunc(ret) != ret || ret <= 0 || ret > float64(^uint(0)>>1) {
		panic(vm.NewTypeError("%s must be a positive integer", name))
	}
	return int(ret) // #nosec G115 -- bounds are checked above.
}

func requireNonNegativeInteger(vm *goja.Runtime, value goja.Value, name string) int {
	ret := value.ToFloat()
	if math.IsNaN(ret) || math.IsInf(ret, 0) || math.Trunc(ret) != ret || ret < 0 || ret > float64(^uint(0)>>1) {
		panic(vm.NewTypeError("%s must be a non-negative integer", name))
	}
	return int(ret) // #nosec G115 -- bounds are checked above.
}

func requirePositiveUint32(vm *goja.Runtime, value goja.Value, name string) uint32 {
	ret := value.ToFloat()
	if math.IsNaN(ret) || math.IsInf(ret, 0) || math.Trunc(ret) != ret || ret <= 0 || ret > math.MaxUint32 {
		panic(vm.NewTypeError("%s must be a positive 32-bit integer", name))
	}
	return uint32(ret) // #nosec G115 -- bounds are checked above.
}

func requireArgumentCount(vm *goja.Runtime, call goja.FunctionCall, count int, usage string) {
	if len(call.Arguments) != count {
		panic(vm.NewTypeError("%s requires exactly %d argument(s)", usage, count))
	}
}

func normalizedValue(vm *goja.Runtime, value any) goja.Value {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(vm.NewTypeError("normalize value: %s", err))
	}
	var plain any
	if err := json.Unmarshal(encoded, &plain); err != nil {
		panic(vm.NewTypeError("normalize value: %s", err))
	}
	return vm.ToValue(plain)
}

func mustSet(vm *goja.Runtime, object *goja.Object, name string, value any) {
	if err := object.Set(name, value); err != nil {
		panic(vm.NewTypeError("set %s: %s", name, err))
	}
}

func cloneSchemas(schemas map[string]idpprogram.Schema) map[string]idpprogram.Schema {
	if len(schemas) == 0 {
		return map[string]idpprogram.Schema{}
	}
	encoded, err := json.Marshal(schemas)
	if err != nil {
		panic(fmt.Sprintf("clone schemas: %v", err))
	}
	var ret map[string]idpprogram.Schema
	if err := json.Unmarshal(encoded, &ret); err != nil {
		panic(fmt.Sprintf("clone schemas: %v", err))
	}
	return ret
}
