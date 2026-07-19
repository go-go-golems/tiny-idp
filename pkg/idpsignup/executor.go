// Package idpsignup contains the intentionally narrow native binding for the
// first scripted Tiny-IDP workflow: open local-account signup.
package idpsignup

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

//go:embed open_signup.js
var DefaultSource string

const (
	WorkflowID       = "signup"
	StartHandler     = "start"
	SubmittedHandler = "submitted"
)

type Executor struct {
	artifact *idpscript.Artifact
	pool     *idpscript.Pool
}

var _ idpcontinuation.GenerationResolver = (*Executor)(nil)

func New(ctx context.Context, source string, workers int) (*Executor, error) {
	if source == "" {
		source = DefaultSource
	}
	if workers <= 0 {
		return nil, errors.New("signup executor worker count must be positive")
	}
	options := idpscript.DefaultCompileOptions()
	options.SourceName = "open_signup.js"
	options.Schemas = schemas()
	artifact, err := idpscript.Compile(ctx, source, options)
	if err != nil {
		return nil, errors.Wrap(err, "compile signup program")
	}
	factory, err := idpscript.NewRuntimeFactory(options.Schemas)
	if err != nil {
		return nil, errors.Wrap(err, "create signup runtime factory")
	}
	pool, err := idpscript.NewPool(ctx, artifact, factory, workers)
	if err != nil {
		return nil, errors.Wrap(err, "create signup runtime pool")
	}
	return &Executor{artifact: artifact, pool: pool}, nil
}

func (e *Executor) Close(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return nil
	}
	return e.pool.Close(ctx)
}

func (e *Executor) Program() idpprogram.Program { return e.artifact.Program() }

func (e *Executor) Fingerprint() string { return e.artifact.Fingerprints().Program }

func (e *Executor) ResolveProgram(_ context.Context, fingerprint string) (idpprogram.Program, error) {
	if e == nil || e.artifact == nil || fingerprint != e.Fingerprint() {
		return idpprogram.Program{}, errors.New("signup program generation is unavailable")
	}
	return e.Program(), nil
}

func (e *Executor) Start(ctx context.Context) (idpworkflow.ValidatedPresentation, error) {
	if e == nil || e.pool == nil {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup executor is unavailable")
	}
	outcome, err := e.pool.Invoke(ctx, "signup.start", json.RawMessage(`{}`), nil)
	if err != nil {
		return idpworkflow.ValidatedPresentation{}, errors.Wrap(err, "invoke signup start")
	}
	if outcome.Kind != idpprogram.OutcomePresent || outcome.Continuation == nil {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup start did not return a presentation")
	}
	presentation, err := idpworkflow.DecodePresentation(outcome.Presentation)
	if err != nil {
		return idpworkflow.ValidatedPresentation{}, err
	}
	if presentation.ResumeHandler != outcome.Continuation.HandlerID || presentation.ExpiresIn.Milliseconds() != outcome.Continuation.ExpiresIn*1000 {
		return idpworkflow.ValidatedPresentation{}, errors.New("signup presentation continuation does not agree with outcome")
	}
	return idpworkflow.ValidatePresentation(e.Program(), WorkflowID, StartHandler, presentation, idpworkflow.DefaultRegistry(), idpworkflow.DefaultMaximumContinuationTTL)
}

func (e *Executor) Submit(ctx context.Context, values map[idpworkflow.FieldID]string, secrets map[string]idpworkflow.SecretHandle) (idpprogram.Outcome, error) {
	if e == nil || e.pool == nil {
		return idpprogram.Outcome{}, errors.New("signup executor is unavailable")
	}
	input, err := json.Marshal(map[string]string{
		"displayName": values[idpworkflow.FieldDisplayName],
		"email":       values[idpworkflow.FieldEmail],
	})
	if err != nil {
		return idpprogram.Outcome{}, errors.Wrap(err, "encode signup submission")
	}
	return e.pool.InvokeWithSecrets(ctx, "signup.submitted", input, nil, secrets)
}

func schemas() map[string]idpprogram.Schema {
	return map[string]idpprogram.Schema{
		"signupStartInput": {ID: "signupStartInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 64, Additional: false, Fields: map[string]idpprogram.SchemaField{}},
		"signupSubmittedInput": {ID: "signupSubmittedInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Additional: false, Fields: map[string]idpprogram.SchemaField{
			"displayName": {Ref: "signupText"}, "email": {Ref: "signupEmail"},
		}},
		"signupText":   {ID: "signupText", Kind: idpprogram.SchemaKindString, MaxBytes: 512, MaxLength: 120},
		"signupEmail":  {ID: "signupEmail", Kind: idpprogram.SchemaKindString, MaxBytes: 512, MaxLength: 320},
		"signupResult": {ID: "signupResult", Kind: idpprogram.SchemaKindObject, MaxBytes: 64, Additional: false, Fields: map[string]idpprogram.SchemaField{}},
	}
}
