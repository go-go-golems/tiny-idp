package idpscript

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

var (
	ErrRuntimeSaturated = errors.New("scripting runtime pool is saturated")
	ErrPoolClosed       = errors.New("scripting runtime pool is closed")
)

// Pool owns a bounded number of exclusive runtime workers for one artifact.
type Pool struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	artifact  *Artifact
	factory   *engine.RuntimeFactory
	idle      chan *worker
	workers   map[uint64]*worker
	nextID    uint64
	discarded uint64
	active    int
	closed    bool
	period    time.Duration
}

// PoolStats is a race-safe operational snapshot.
type PoolStats struct {
	Capacity       int
	Idle           int
	Active         int
	WorkersCreated uint64
	Discarded      uint64
	Closed         bool
}

// NewPool loads and verifies size independent workers.
func NewPool(ctx context.Context, artifact *Artifact, factory *engine.RuntimeFactory, size int) (*Pool, error) {
	if ctx == nil {
		return nil, errors.New("pool context is required")
	}
	if artifact == nil || factory == nil {
		return nil, errors.New("pool artifact and runtime factory are required")
	}
	if size <= 0 {
		return nil, errors.New("pool size must be greater than zero")
	}
	poolCtx, cancel := context.WithCancel(ctx)
	p := &Pool{
		ctx:      poolCtx,
		cancel:   cancel,
		artifact: artifact,
		factory:  factory,
		idle:     make(chan *worker, size),
		workers:  map[uint64]*worker{},
		period:   time.Millisecond,
	}
	for i := 0; i < size; i++ {
		worker, err := p.loadWorker(poolCtx)
		if err != nil {
			_ = p.Close(context.Background())
			return nil, errors.Wrapf(err, "load runtime worker %d", i)
		}
		p.workers[worker.id] = worker
		p.idle <- worker
	}
	return p, nil
}

// Invoke acquires one worker, runs a lambda, and releases only a safe worker.
func (p *Pool) Invoke(ctx context.Context, lambdaID string, input json.RawMessage, capabilities map[string]CapabilityBinding) (idpprogram.Outcome, error) {
	return p.InvokeWithSecrets(ctx, lambdaID, input, capabilities, nil)
}

// InvokeWithSecrets adds request-scoped opaque secret handles to an invocation.
// The JSON input remains public data; secret values are never encoded into it.
func (p *Pool) InvokeWithSecrets(ctx context.Context, lambdaID string, input json.RawMessage, capabilities map[string]CapabilityBinding, secrets map[string]idpworkflow.SecretHandle) (idpprogram.Outcome, error) {
	return p.invoke(ctx, lambdaID, input, capabilities, secrets, nil)
}

// InvokeWithSecretsAndEvidence projects native-verified evidence into exactly
// one owned invocation. Callers cannot install evidence on a worker or reuse
// it after the invocation returns.
func (p *Pool) InvokeWithSecretsAndEvidence(ctx context.Context, lambdaID string, input json.RawMessage, capabilities map[string]CapabilityBinding, secrets map[string]idpworkflow.SecretHandle, evidence map[string]json.RawMessage) (idpprogram.Outcome, error) {
	return p.invoke(ctx, lambdaID, input, capabilities, secrets, evidence)
}

func (p *Pool) invoke(ctx context.Context, lambdaID string, input json.RawMessage, capabilities map[string]CapabilityBinding, secrets map[string]idpworkflow.SecretHandle, evidence map[string]json.RawMessage) (idpprogram.Outcome, error) {
	worker, err := p.acquire(ctx)
	if err != nil {
		return idpprogram.Outcome{}, err
	}
	outcome, safe, invokeErr := worker.invokeWithSecrets(ctx, lambdaID, input, capabilities, secrets, evidence)
	if safe {
		p.release(worker)
		return outcome, invokeErr
	}
	replaceErr := p.discardAndReplace(worker)
	if replaceErr != nil {
		if invokeErr == nil {
			return idpprogram.Outcome{}, replaceErr
		}
		return idpprogram.Outcome{}, errors.Wrapf(invokeErr, "replace discarded worker: %v", replaceErr)
	}
	return idpprogram.Outcome{}, invokeErr
}

func (p *Pool) acquire(ctx context.Context) (*worker, error) {
	if ctx == nil {
		return nil, errors.New("invocation context is required")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}
	p.mu.Unlock()
	select {
	case worker := <-p.idle:
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, ErrPoolClosed
		}
		p.active++
		p.mu.Unlock()
		return worker, nil
	case <-ctx.Done():
		return nil, errors.Wrap(ErrRuntimeSaturated, ctx.Err().Error())
	case <-p.ctx.Done():
		return nil, ErrPoolClosed
	}
}

func (p *Pool) release(worker *worker) {
	p.mu.Lock()
	p.active--
	closed := p.closed
	p.mu.Unlock()
	if closed {
		_ = worker.image.Close(context.Background())
		return
	}
	p.idle <- worker
}

func (p *Pool) discardAndReplace(worker *worker) error {
	p.mu.Lock()
	delete(p.workers, worker.id)
	p.discarded++
	p.active--
	closed := p.closed
	p.mu.Unlock()
	_ = worker.image.Close(context.Background())
	if closed {
		return nil
	}
	replacement, err := p.loadWorker(p.ctx)
	if err != nil {
		return errors.Wrap(err, "load replacement runtime worker")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = replacement.image.Close(context.Background())
		return nil
	}
	p.workers[replacement.id] = replacement
	p.mu.Unlock()
	p.idle <- replacement
	return nil
}

func (p *Pool) loadWorker(ctx context.Context) (*worker, error) {
	image, err := p.artifact.Load(ctx, p.factory)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	p.nextID++
	id := p.nextID
	p.mu.Unlock()
	return &worker{id: id, image: image}, nil
}

// Stats returns a bounded-cardinality pool snapshot.
func (p *Pool) Stats() PoolStats {
	if p == nil {
		return PoolStats{Closed: true}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return PoolStats{
		Capacity:       len(p.workers),
		Idle:           len(p.idle),
		Active:         p.active,
		WorkersCreated: p.nextID,
		Discarded:      p.discarded,
		Closed:         p.closed,
	}
}

// Close waits for acquired workers, cancels the pool lifetime, and closes all
// runtime/event-loop resources.
func (p *Pool) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	ticker := time.NewTicker(p.period)
	defer ticker.Stop()
	for {
		p.mu.Lock()
		active := p.active
		p.mu.Unlock()
		if active == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "wait for active scripting workers")
		case <-ticker.C:
		}
	}
	p.cancel()
	p.mu.Lock()
	workers := make([]*worker, 0, len(p.workers))
	for _, worker := range p.workers {
		workers = append(workers, worker)
	}
	p.workers = map[uint64]*worker{}
	p.mu.Unlock()
	var closeErr error
	for _, worker := range workers {
		if err := worker.image.Close(ctx); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}
