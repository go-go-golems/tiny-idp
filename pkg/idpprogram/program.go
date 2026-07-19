// Package idpprogram defines the runtime-independent contracts produced by a
// Tiny-IDP JavaScript program. It deliberately contains no Goja dependency.
package idpprogram

import "encoding/json"

// Program is the complete serializable contract for one compiled scripting
// generation. JavaScript callbacks are registered separately by Lambda.ID;
// functions and other VM-owned values never enter this structure.
type Program struct {
	APIVersion   string                           `json:"apiVersion"`
	Name         string                           `json:"name"`
	Workflows    map[string]Workflow              `json:"workflows"`
	Providers    map[string]Provider              `json:"providers,omitempty"`
	Lambdas      map[string]LambdaSpec            `json:"lambdas"`
	Schemas      map[string]Schema                `json:"schemas"`
	Capabilities map[string]CapabilityRequirement `json:"capabilities,omitempty"`
	Tests        []ProgramTest                    `json:"tests,omitempty"`
}

// ProgramTest is a bounded, declarative lambda test. It never contains a
// callback, capability implementation, browser object, or host authority;
// the Go runner creates those deterministic native test bindings.
type ProgramTest struct {
	ID           string          `json:"id"`
	LambdaID     string          `json:"lambdaId"`
	Input        json.RawMessage `json:"input"`
	ExpectedKind OutcomeKind     `json:"expectedKind"`
}

// Workflow is a named set of handlers with one native entry point.
type Workflow struct {
	ID           string                 `json:"id"`
	Version      uint32                 `json:"version"`
	EntryHandler string                 `json:"entryHandler"`
	Handlers     map[string]HandlerSpec `json:"handlers"`
}

// HandlerSpec binds a workflow handler name to a registered lambda and records
// every statically permitted immediate or browser-continuation edge.
type HandlerSpec struct {
	ID                string             `json:"id"`
	LambdaID          string             `json:"lambdaId"`
	ContinuationEdges []ContinuationEdge `json:"continuationEdges,omitempty"`
}

// ContinuationEdge declares a legal continue/present/challenge destination. The
// input schema is repeated here so compilation can reject incompatible edges
// before a request exists.
type ContinuationEdge struct {
	OutcomeKind OutcomeKind `json:"outcomeKind"`
	HandlerID   string      `json:"handlerId"`
	InputSchema string      `json:"inputSchema"`
}
