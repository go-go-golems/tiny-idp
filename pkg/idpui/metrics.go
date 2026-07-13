package idpui

import "time"

// RenderStats contains process-local, non-sensitive interaction-rendering
// counters. It deliberately has no client, user, route, error-text, or other
// high-cardinality labels.
type RenderStats struct {
	Attempts              uint64
	Successes             uint64
	Failures              uint64
	OversizedDocuments    uint64
	EmptyDocuments        uint64
	ResponseWriteFailures uint64
	TotalLatency          time.Duration
	MaxLatency            time.Duration
}
