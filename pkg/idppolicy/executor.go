// Package idppolicy binds typed authorization and claims provider callbacks to
// the bounded Tiny-IDP Goja runtime. It intentionally exposes no HTTP,
// Fosite, cookie, credential, signing-key, session, or store authority.
package idppolicy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

const (
	defaultAuthorizationProvider = "authorization.default"
	defaultClaimsProvider        = "claims.default"
)

type Config struct {
	AuthorizationProvider string
	ClaimsProvider        string
}

type Executor struct {
	pool     *idpscript.Pool
	invoker  *idpscript.ProviderInvoker
	config   Config
	artifact *idpscript.Artifact
}

// New compiles a policy source against the small host-owned policy schema
// catalog and warms a fixed number of owned runtimes. A source may declare
// either or both provider kinds; invocation of an undeclared kind fails closed.
func New(ctx context.Context, source string, workers int, config Config) (*Executor, error) {
	if workers <= 0 {
		return nil, errors.New("policy executor worker count must be positive")
	}
	config = normalizeConfig(config)
	options := idpscript.DefaultCompileOptions()
	options.SourceName = "tinyidp-policy.js"
	options.Schemas = schemas()
	artifact, err := idpscript.Compile(ctx, source, options)
	if err != nil {
		return nil, errors.Wrap(err, "compile policy program")
	}
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	if err != nil {
		return nil, errors.Wrap(err, "create policy runtime factory")
	}
	pool, err := idpscript.NewPool(ctx, artifact, factory, workers)
	if err != nil {
		return nil, errors.Wrap(err, "create policy runtime pool")
	}
	invoker, err := idpscript.NewProviderInvoker(pool)
	if err != nil {
		_ = pool.Close(context.Background())
		return nil, errors.Wrap(err, "create policy provider invoker")
	}
	return &Executor{pool: pool, invoker: invoker, config: config, artifact: artifact}, nil
}

func (e *Executor) Close(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return nil
	}
	return e.pool.Close(ctx)
}

func (e *Executor) Fingerprints() idpprogram.Fingerprints {
	if e == nil || e.artifact == nil {
		return idpprogram.Fingerprints{}
	}
	return e.artifact.Fingerprints()
}

func (e *Executor) PoolStats() idpscript.PoolStats {
	if e == nil || e.pool == nil {
		return idpscript.PoolStats{Closed: true}
	}
	return e.pool.Stats()
}

var _ idp.AuthorizationPolicy = (*Executor)(nil)
var _ idp.ClaimsPolicy = (*Executor)(nil)

// Authorize projects a cloned immutable input into the named provider handler.
// Complete returns a validated allow/deny/skip decision. A JavaScript deny or
// skip outcome is normalized to the equivalent native decision; any runtime,
// malformed-output, or explicit error outcome is returned for the provider to
// fail closed before consent/code issuance.
func (e *Executor) Authorize(ctx context.Context, input idp.AuthorizationInput) (idp.AuthorizationDecision, error) {
	raw, err := json.Marshal(input.Clone())
	if err != nil {
		return idp.AuthorizationDecision{}, errors.Wrap(err, "encode authorization policy input")
	}
	outcome, err := e.invoke(ctx, e.config.AuthorizationProvider, idpprogram.AuthorizationDecideHandler, raw)
	if err != nil {
		return idp.AuthorizationDecision{}, err
	}
	decision := idp.AuthorizationDecision{}
	switch outcome.Kind {
	case idpprogram.OutcomeComplete:
		if err := json.Unmarshal(outcome.Value, &decision); err != nil {
			return idp.AuthorizationDecision{}, errors.Wrap(err, "decode authorization policy decision")
		}
	case idpprogram.OutcomeDeny:
		decision = idp.AuthorizationDecision{Kind: idp.AuthorizationDeny, DiagnosticID: outcome.Code}
	case idpprogram.OutcomeSkip:
		decision = idp.AuthorizationDecision{Kind: idp.AuthorizationSkip}
	case idpprogram.OutcomeError:
		return idp.AuthorizationDecision{}, errors.Errorf("authorization policy returned error %q", outcome.Code)
	case idpprogram.OutcomeContinue, idpprogram.OutcomePresent, idpprogram.OutcomeChallenge, idpprogram.OutcomeCommit:
		return idp.AuthorizationDecision{}, errors.Errorf("authorization policy returned unsupported outcome %q", outcome.Kind)
	default:
		return idp.AuthorizationDecision{}, errors.Errorf("authorization policy returned unsupported outcome %q", outcome.Kind)
	}
	return idp.NormalizeAuthorizationDecision(decision)
}

// Claims projects a cloned native claims view to a provider callback. The
// callback can return only its additional claim map; native MergeClaims still
// protects issuer/subject/time/nonce and existing scope-filtered claims.
func (e *Executor) Claims(ctx context.Context, input idp.ClaimsInput) (idp.ClaimsOutput, error) {
	raw, err := json.Marshal(input.Clone())
	if err != nil {
		return idp.ClaimsOutput{}, errors.Wrap(err, "encode claims policy input")
	}
	outcome, err := e.invoke(ctx, e.config.ClaimsProvider, idpprogram.ClaimsAdditionalHandler, raw)
	if err != nil {
		return idp.ClaimsOutput{}, err
	}
	if outcome.Kind == idpprogram.OutcomeError {
		return idp.ClaimsOutput{}, errors.Errorf("claims policy returned error %q", outcome.Code)
	}
	if outcome.Kind != idpprogram.OutcomeComplete {
		return idp.ClaimsOutput{}, errors.Errorf("claims policy returned unsupported outcome %q", outcome.Kind)
	}
	var output idp.ClaimsOutput
	if err := json.Unmarshal(outcome.Value, &output); err != nil {
		return idp.ClaimsOutput{}, errors.Wrap(err, "decode claims policy output")
	}
	if err := output.Validate(input.Base); err != nil {
		return idp.ClaimsOutput{}, errors.Wrap(err, "validate claims policy output")
	}
	return output, nil
}

func (e *Executor) invoke(ctx context.Context, provider, handler string, input json.RawMessage) (idpprogram.Outcome, error) {
	if e == nil || e.invoker == nil {
		return idpprogram.Outcome{}, errors.New("policy executor is unavailable")
	}
	outcome, err := e.invoker.Invoke(ctx, provider, handler, input, nil)
	if err != nil {
		return idpprogram.Outcome{}, errors.Wrap(err, "invoke policy provider")
	}
	return outcome, nil
}

func normalizeConfig(config Config) Config {
	if config.AuthorizationProvider == "" {
		config.AuthorizationProvider = defaultAuthorizationProvider
	}
	if config.ClaimsProvider == "" {
		config.ClaimsProvider = defaultClaimsProvider
	}
	return config
}

func schemas() map[string]idpprogram.Schema {
	// Native policy contracts validate semantics after decoding. The program
	// schema catalog supplies a separate bounded JSON crossing for Goja and is
	// intentionally permissive enough to carry copied scope/group/claim maps.
	return map[string]idpprogram.Schema{
		"authorizationInput":  {ID: "authorizationInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 16 << 10, Additional: true},
		"authorizationOutput": {ID: "authorizationOutput", Kind: idpprogram.SchemaKindObject, MaxBytes: 8 << 10, Additional: true},
		"claimsInput":         {ID: "claimsInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 32 << 10, Additional: true},
		"claimsOutput":        {ID: "claimsOutput", Kind: idpprogram.SchemaKindObject, MaxBytes: 64 << 10, Additional: true},
	}
}

func (e *Executor) String() string {
	fingerprints := e.Fingerprints()
	return fmt.Sprintf("idppolicy.Executor{%s}", fingerprints.Program)
}
