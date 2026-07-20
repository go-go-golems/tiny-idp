package idpscript

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

// ProviderInvoker routes one typed provider operation through the same bounded
// worker pool used by workflow lambdas. It exposes no Goja runtime, loader, or
// ambient host authority to callers.
type ProviderInvoker struct {
	pool    *Pool
	program idpprogram.Program
}

// NewProviderInvoker binds an invoker to the exact immutable artifact loaded
// by pool. The program is copied so later callers cannot mutate routing or
// schema metadata.
func NewProviderInvoker(pool *Pool) (*ProviderInvoker, error) {
	if pool == nil || pool.artifact == nil {
		return nil, errors.New("provider worker pool is required")
	}
	return &ProviderInvoker{pool: pool, program: pool.artifact.Program()}, nil
}

// Invoke validates provider and handler selection against the compiled
// contract, then delegates schema, capability, budget, Promise, and worker
// ownership enforcement to Pool.Invoke.
func (i *ProviderInvoker) Invoke(ctx context.Context, providerID, handlerID string, input json.RawMessage, capabilities map[string]CapabilityBinding) (idpprogram.Outcome, error) {
	if i == nil || i.pool == nil {
		return idpprogram.Outcome{}, errors.New("provider invoker is unavailable")
	}
	provider, ok := i.program.Providers[providerID]
	if !ok {
		return idpprogram.Outcome{}, errors.Errorf("unknown provider %q", providerID)
	}
	handler, ok := provider.Handlers[handlerID]
	if !ok {
		return idpprogram.Outcome{}, errors.Errorf("provider %q has no handler %q", providerID, handlerID)
	}
	return i.pool.Invoke(ctx, handler.LambdaID, input, capabilities)
}

// Provider returns a defensive copy of a provider contract for native callers
// that need to bind a typed capability or effect committer.
func (i *ProviderInvoker) Provider(providerID string) (idpprogram.Provider, bool) {
	if i == nil {
		return idpprogram.Provider{}, false
	}
	provider, ok := i.program.Providers[providerID]
	if !ok {
		return idpprogram.Provider{}, false
	}
	encoded, err := json.Marshal(provider)
	if err != nil {
		return idpprogram.Provider{}, false
	}
	var providerCopy idpprogram.Provider
	if err := json.Unmarshal(encoded, &providerCopy); err != nil {
		return idpprogram.Provider{}, false
	}
	return providerCopy, true
}
