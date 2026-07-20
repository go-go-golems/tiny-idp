package idpprogram

import "time"

// LambdaKind separates workflow handlers from later typed provider callbacks.
type LambdaKind string

const (
	LambdaKindWorkflow LambdaKind = "workflow"
	LambdaKindProvider LambdaKind = "provider"
)

// Valid reports whether k is a supported lambda category.
func (k LambdaKind) Valid() bool {
	return k == LambdaKindWorkflow || k == LambdaKindProvider
}

// LambdaSpec is the activation and invocation contract for a named JavaScript
// callback. It never contains the callback function itself.
type LambdaSpec struct {
	ID                   string                  `json:"id"`
	Kind                 LambdaKind              `json:"kind"`
	InputSchema          string                  `json:"inputSchema"`
	OutputSchema         string                  `json:"outputSchema"`
	AllowedOutcomes      []OutcomeKind           `json:"allowedOutcomes"`
	RequiredCapabilities []CapabilityRequirement `json:"requiredCapabilities,omitempty"`
	AllowedEffects       []EffectKind            `json:"allowedEffects,omitempty"`
	Budget               InvocationBudget        `json:"budget"`
	SourceLocation       SourceLocation          `json:"sourceLocation,omitempty"`
}

// InvocationBudget bounds the host resources available to one callback.
type InvocationBudget struct {
	Timeout            time.Duration `json:"timeoutNanos"`
	MaxCapabilityCalls int           `json:"maxCapabilityCalls"`
	MaxOutputBytes     int           `json:"maxOutputBytes"`
}

// SourceLocation is safe compiler metadata used for deterministic diagnostics.
type SourceLocation struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}
