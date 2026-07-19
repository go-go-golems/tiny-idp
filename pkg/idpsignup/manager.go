package idpsignup

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

// GenerationManager atomically publishes warmed signup executors. It retains
// a bounded number of prior fingerprints so a host can route compatible
// browser continuations explicitly rather than resuming them on whichever
// source happened to become active last.
type GenerationManager struct {
	mu       sync.RWMutex
	workers  int
	retained int
	active   *Executor
	byHash   map[string]*Executor
	order    []string
	closed   bool
	metrics  generationMetrics
}

type generationMetrics struct{ activations, activationFailures, evicted, closed atomic.Uint64 }

// GenerationMetrics contains bounded, secret-free operational counters.
type GenerationMetrics struct {
	Activations, ActivationFailures, Evicted, Closed uint64
	Retained, PoolCapacity, PoolActive               int
}

type GenerationSnapshot struct {
	ActiveFingerprint string
	Retained          []string
	Ready             bool
	Pool              idpscript.PoolStats
	Metrics           GenerationMetrics
}

func NewGenerationManager(ctx context.Context, source string, workers, retained int) (*GenerationManager, error) {
	if workers <= 0 {
		return nil, errors.New("generation manager worker count must be positive")
	}
	if retained < 0 {
		return nil, errors.New("generation manager retained generation count must not be negative")
	}
	candidate, err := warmGeneration(ctx, source, workers)
	if err != nil {
		return nil, errors.Wrap(err, "warm initial signup generation")
	}
	fingerprint := candidate.Fingerprint()
	return &GenerationManager{workers: workers, retained: retained, active: candidate, byHash: map[string]*Executor{fingerprint: candidate}, order: []string{fingerprint}}, nil
}

// Activate compiles and warms a candidate before taking the publication lock.
// On every candidate failure the active generation remains unchanged.
func (m *GenerationManager) Activate(ctx context.Context, source string) (string, error) {
	if m == nil {
		return "", errors.New("generation manager is unavailable")
	}
	candidate, err := warmGeneration(ctx, source, m.workers)
	if err != nil {
		m.metrics.activationFailures.Add(1)
		return "", errors.Wrap(err, "warm candidate signup generation")
	}
	fingerprint := candidate.Fingerprint()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		_ = candidate.Close(context.Background())
		m.metrics.activationFailures.Add(1)
		return "", errors.New("generation manager is closed")
	}
	if existing := m.byHash[fingerprint]; existing != nil {
		m.active = existing
		m.metrics.activations.Add(1)
		m.mu.Unlock()
		_ = candidate.Close(context.Background())
		return fingerprint, nil
	}
	m.active = candidate
	m.metrics.activations.Add(1)
	m.byHash[fingerprint] = candidate
	m.order = append(m.order, fingerprint)
	toClose := m.evictLocked()
	m.mu.Unlock()
	for _, executor := range toClose {
		if closeErr := executor.Close(context.Background()); closeErr != nil {
			return "", errors.Wrap(closeErr, "close drained signup generation")
		}
	}
	m.metrics.evicted.Add(uint64(len(toClose)))
	return fingerprint, nil
}

func warmGeneration(ctx context.Context, source string, workers int) (*Executor, error) {
	candidate, err := New(ctx, source, workers)
	if err != nil {
		return nil, err
	}
	for _, result := range candidate.RunTests(ctx) {
		if result.Passed {
			continue
		}
		_ = candidate.Close(context.Background())
		if result.Err != nil {
			return nil, errors.Wrapf(result.Err, "embedded test %q", result.ID)
		}
		return nil, errors.Errorf("embedded test %q expected outcome %q, got %q", result.ID, result.Expected, result.Actual)
	}
	return candidate, nil
}

// Active returns the executor for new browser interactions. Callers that
// resume a continuation must use ExecutorFor with its persisted fingerprint.
func (m *GenerationManager) Active() (*Executor, error) {
	if m == nil {
		return nil, errors.New("generation manager is unavailable")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed || m.active == nil {
		return nil, errors.New("active signup generation is unavailable")
	}
	return m.active, nil
}

func (m *GenerationManager) ExecutorFor(fingerprint string) (*Executor, error) {
	if m == nil || fingerprint == "" {
		return nil, errors.New("signup generation is unavailable")
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return nil, errors.New("generation manager is closed")
	}
	executor := m.byHash[fingerprint]
	if executor == nil {
		return nil, errors.New("signup generation is unavailable")
	}
	return executor, nil
}

func (m *GenerationManager) ResolveProgram(ctx context.Context, fingerprint string) (idpprogram.Program, error) {
	executor, err := m.ExecutorFor(fingerprint)
	if err != nil {
		return idpprogram.Program{}, err
	}
	return executor.ResolveProgram(ctx, fingerprint)
}

func (m *GenerationManager) Snapshot() GenerationSnapshot {
	if m == nil {
		return GenerationSnapshot{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	snapshot := GenerationSnapshot{Retained: append([]string(nil), m.order...)}
	if m.active != nil {
		snapshot.ActiveFingerprint = m.active.Fingerprint()
		snapshot.Pool = m.active.PoolStats()
	}
	snapshot.Ready = !m.closed && m.active != nil && m.active.Ready()
	snapshot.Metrics = m.Metrics()
	return snapshot
}

func (m *GenerationManager) Metrics() GenerationMetrics {
	if m == nil {
		return GenerationMetrics{}
	}
	m.mu.RLock()
	retained := len(m.order)
	var capacity, active int
	if m.active != nil {
		stats := m.active.PoolStats()
		capacity, active = stats.Capacity, stats.Active
	}
	m.mu.RUnlock()
	return GenerationMetrics{Activations: m.metrics.activations.Load(), ActivationFailures: m.metrics.activationFailures.Load(), Evicted: m.metrics.evicted.Load(), Closed: m.metrics.closed.Load(), Retained: retained, PoolCapacity: capacity, PoolActive: active}
}

// Ready returns an operator-safe readiness error for hosts that have opted
// into scripted signup. It distinguishes a missing/closed active generation
// from pool saturation: saturated but warmed workers can still become
// available, whereas a closed or empty pool cannot serve a continuation.
func (m *GenerationManager) Ready() error {
	if m == nil {
		return errors.New("signup generation manager is unavailable")
	}
	snapshot := m.Snapshot()
	if !snapshot.Ready {
		return errors.New("active signup generation is unavailable")
	}
	return nil
}

func (m *GenerationManager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	m.metrics.closed.Add(1)
	executors := make([]*Executor, 0, len(m.byHash))
	for _, executor := range m.byHash {
		executors = append(executors, executor)
	}
	m.byHash = nil
	m.active = nil
	m.order = nil
	m.mu.Unlock()
	for _, executor := range executors {
		if err := executor.Close(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *GenerationManager) evictLocked() []*Executor {
	limit := m.retained + 1 // active generation plus retained predecessors
	toClose := []*Executor{}
	for len(m.order) > limit {
		fingerprint := m.order[0]
		m.order = m.order[1:]
		executor := m.byHash[fingerprint]
		if executor == m.active {
			m.order = append(m.order, fingerprint)
			continue
		}
		delete(m.byHash, fingerprint)
		toClose = append(toClose, executor)
	}
	return toClose
}
