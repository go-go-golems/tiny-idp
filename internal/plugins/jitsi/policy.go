package jitsi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

const (
	PolicyLambdaID     = "integration.jitsi.authorize@v1"
	policyInputSchema  = "integration.jitsi.authorize.input.v1"
	policyOutputSchema = "integration.jitsi.authorize.output.v1"
)

const PolicyTypeScript = `export interface JitsiAuthorizeInput {
  integrationId: string;
  room: string;
  tenant: string;
  identity: {
    subject: string;
    displayName: string;
    preferredUsername: string;
    email: string;
    emailVerified: boolean;
    roles: string[];
    groups: string[];
    authTime: string;
  };
}
export type JitsiAuthorizeResult =
  | { kind: "complete"; claims: { displayName: string; includeEmail: boolean; moderator: boolean } }
  | { kind: "deny"; diagnosticId: "verified_email_required" | "meeting_access_denied" };
`

var allowedPolicyDenials = map[string]struct{}{
	"verified_email_required": {},
	"meeting_access_denied":   {},
}

type PolicyInput struct {
	IntegrationID string         `json:"integrationId"`
	Room          string         `json:"room"`
	Tenant        string         `json:"tenant"`
	Identity      PolicyIdentity `json:"identity"`
}

type PolicyIdentity struct {
	Subject           string   `json:"subject"`
	DisplayName       string   `json:"displayName"`
	PreferredUsername string   `json:"preferredUsername"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Roles             []string `json:"roles"`
	Groups            []string `json:"groups"`
	AuthTime          string   `json:"authTime"`
}

type Decision struct {
	Allowed      bool
	DiagnosticID string
	DisplayName  string
	IncludeEmail bool
	Moderator    bool
}

type PolicyStats struct {
	Invocations uint64
	Allowed     uint64
	Denied      uint64
	Failures    uint64
	Pool        idpscript.PoolStats
}

type PolicyExecutor struct {
	pool        *idpscript.Pool
	invocations atomic.Uint64
	allowed     atomic.Uint64
	denied      atomic.Uint64
	failures    atomic.Uint64
}

func NewPolicyExecutor(ctx context.Context, source string, poolSize int) (*PolicyExecutor, error) {
	if strings.TrimSpace(source) == "" {
		return nil, errors.New("jitsi policy source is required")
	}
	schemas := policySchemas()
	options := idpscript.DefaultCompileOptions()
	options.SourceName = "jitsi-policy.js"
	options.Schemas = schemas
	artifact, err := idpscript.Compile(ctx, source, options)
	if err != nil {
		return nil, fmt.Errorf("compile Jitsi policy: %w", err)
	}
	program := artifact.Program()
	provider, ok := program.Providers["authorization.jitsi"]
	handler, handlerOK := provider.Handlers[idpprogram.AuthorizationDecideHandler]
	if !ok || !handlerOK || handler.LambdaID != PolicyLambdaID ||
		handler.InputSchema != policyInputSchema || handler.OutputSchema != policyOutputSchema {
		return nil, errors.New("jitsi policy must declare authorization.jitsi decide at integration.jitsi.authorize@v1")
	}
	lambda := program.Lambdas[PolicyLambdaID]
	if len(lambda.RequiredCapabilities) != 0 || len(lambda.AllowedEffects) != 0 {
		return nil, errors.New("jitsi policy version 1 does not permit capabilities or effects")
	}
	factory, err := idpscript.NewRuntimeFactory(schemas)
	if err != nil {
		return nil, fmt.Errorf("construct Jitsi policy runtime: %w", err)
	}
	pool, err := idpscript.NewPool(ctx, artifact, factory, poolSize)
	if err != nil {
		return nil, fmt.Errorf("warm Jitsi policy pool: %w", err)
	}
	return &PolicyExecutor{pool: pool}, nil
}

func (e *PolicyExecutor) Authorize(ctx context.Context, input PolicyInput) (Decision, error) {
	if e == nil || e.pool == nil {
		return Decision{}, errors.New("jitsi policy executor is unavailable")
	}
	e.invocations.Add(1)
	encoded, err := json.Marshal(input)
	if err != nil {
		e.failures.Add(1)
		return Decision{}, err
	}
	outcome, err := e.pool.Invoke(ctx, PolicyLambdaID, encoded, nil)
	if err != nil {
		e.failures.Add(1)
		return Decision{}, err
	}
	switch outcome.Kind {
	case idpprogram.OutcomeDeny:
		if _, ok := allowedPolicyDenials[outcome.Code]; !ok {
			e.failures.Add(1)
			return Decision{}, errors.New("jitsi policy returned an unsupported denial code")
		}
		e.denied.Add(1)
		return Decision{DiagnosticID: outcome.Code}, nil
	case idpprogram.OutcomeComplete:
		var value struct {
			Kind   string `json:"kind"`
			Claims struct {
				DisplayName  string `json:"displayName"`
				IncludeEmail bool   `json:"includeEmail"`
				Moderator    bool   `json:"moderator"`
			} `json:"claims"`
		}
		if err := json.Unmarshal(outcome.Value, &value); err != nil || value.Kind != "complete" ||
			strings.TrimSpace(value.Claims.DisplayName) == "" {
			e.failures.Add(1)
			return Decision{}, errors.New("jitsi policy completion was not accepted")
		}
		e.allowed.Add(1)
		return Decision{
			Allowed: true, DisplayName: value.Claims.DisplayName,
			IncludeEmail: value.Claims.IncludeEmail, Moderator: value.Claims.Moderator,
		}, nil
	case idpprogram.OutcomeContinue, idpprogram.OutcomePresent, idpprogram.OutcomeChallenge,
		idpprogram.OutcomeCommit, idpprogram.OutcomeSkip, idpprogram.OutcomeError:
		e.failures.Add(1)
		return Decision{}, errors.New("jitsi policy returned an unsupported outcome")
	}
	return Decision{}, errors.New("jitsi policy returned an unknown outcome")
}

func (e *PolicyExecutor) Ready() bool {
	return e != nil && e.pool != nil && !e.pool.Stats().Closed && e.pool.Stats().Capacity > 0
}

func (e *PolicyExecutor) Stats() PolicyStats {
	if e == nil {
		return PolicyStats{}
	}
	return PolicyStats{
		Invocations: e.invocations.Load(), Allowed: e.allowed.Load(), Denied: e.denied.Load(),
		Failures: e.failures.Load(), Pool: e.pool.Stats(),
	}
}

func (e *PolicyExecutor) Close(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return nil
	}
	return e.pool.Close(ctx)
}

func PolicyInputFromIdentity(identity pluginapi.Identity, integrationID, room, tenant string) PolicyInput {
	displayName := identity.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = identity.PreferredUsername
	}
	authTime := ""
	if !identity.AuthTime.IsZero() {
		authTime = identity.AuthTime.UTC().Format(time.RFC3339)
	}
	return PolicyInput{
		IntegrationID: integrationID, Room: room, Tenant: tenant,
		Identity: PolicyIdentity{
			Subject: identity.Subject, DisplayName: displayName, PreferredUsername: identity.PreferredUsername,
			Email: identity.Email, EmailVerified: identity.EmailVerified,
			Roles: append([]string(nil), identity.Roles...), Groups: append([]string(nil), identity.Groups...), AuthTime: authTime,
		},
	}
}

func policySchemas() map[string]idpprogram.Schema {
	stringSchema := func(id string, maxLength int) idpprogram.Schema {
		return idpprogram.Schema{ID: id, Kind: idpprogram.SchemaKindString, MaxBytes: maxLength + 2, MaxLength: maxLength}
	}
	return map[string]idpprogram.Schema{
		"jitsi.string.128": stringSchema("jitsi.string.128", 128),
		"jitsi.string.256": stringSchema("jitsi.string.256", 256),
		"jitsi.bool":       {ID: "jitsi.bool", Kind: idpprogram.SchemaKindBoolean, MaxBytes: 5},
		"jitsi.strings":    {ID: "jitsi.strings", Kind: idpprogram.SchemaKindArray, MaxBytes: 4096, Items: "jitsi.string.128", MaxItems: 64},
		"jitsi.identity": {
			ID: "jitsi.identity", Kind: idpprogram.SchemaKindObject, MaxBytes: 8192,
			Fields: map[string]idpprogram.SchemaField{
				"subject": {Ref: "jitsi.string.256", Required: true}, "displayName": {Ref: "jitsi.string.256", Required: true},
				"preferredUsername": {Ref: "jitsi.string.256", Required: true}, "email": {Ref: "jitsi.string.256", Required: true},
				"emailVerified": {Ref: "jitsi.bool", Required: true}, "roles": {Ref: "jitsi.strings", Required: true},
				"groups": {Ref: "jitsi.strings", Required: true}, "authTime": {Ref: "jitsi.string.128", Required: true},
			},
		},
		policyInputSchema: {
			ID: policyInputSchema, Kind: idpprogram.SchemaKindObject, MaxBytes: 16384,
			Fields: map[string]idpprogram.SchemaField{
				"integrationId": {Ref: "jitsi.string.128", Required: true}, "room": {Ref: "jitsi.string.128", Required: true},
				"tenant": {Ref: "jitsi.string.128", Required: true}, "identity": {Ref: "jitsi.identity", Required: true},
			},
		},
		"jitsi.claims": {
			ID: "jitsi.claims", Kind: idpprogram.SchemaKindObject, MaxBytes: 2048,
			Fields: map[string]idpprogram.SchemaField{
				"displayName": {Ref: "jitsi.string.256", Required: true}, "includeEmail": {Ref: "jitsi.bool", Required: true},
				"moderator": {Ref: "jitsi.bool", Required: true},
			},
		},
		policyOutputSchema: {
			ID: policyOutputSchema, Kind: idpprogram.SchemaKindObject, MaxBytes: 4096,
			Fields: map[string]idpprogram.SchemaField{
				"kind": {Ref: "jitsi.string.128", Required: true}, "claims": {Ref: "jitsi.claims"},
				"diagnosticId": {Ref: "jitsi.string.128"},
			},
		},
	}
}
