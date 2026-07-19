package idp

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// AuthorizationPolicy decides whether a request that has already passed native
// OAuth/OIDC validation may proceed. It never receives a Fosite request, an
// HTTP request, browser state, credentials, signing keys, or token mutators.
type AuthorizationPolicy interface {
	Authorize(context.Context, AuthorizationInput) (AuthorizationDecision, error)
}

// AuthorizationInput is the immutable, transport-neutral view supplied after
// native request, client, redirect URI, scope, PKCE, prompt, session, and
// authentication validation. Slices are copied by Clone before a policy is
// invoked so a policy cannot mutate provider-owned state.
type AuthorizationInput struct {
	Subject        AuthorizationSubject
	Client         AuthorizationClient
	Request        AuthorizationRequest
	Authentication AuthenticationView
}

type AuthorizationSubject struct {
	Subject       string
	Tenant        string
	Groups        []string
	Roles         []string
	EmailVerified bool
}

type AuthorizationClient struct {
	ID     string
	Public bool
}

type AuthorizationRequest struct {
	Scopes   []string
	Audience []string
	Prompt   []string
}

type AuthenticationView struct {
	AuthenticatedAt time.Time
	AMR             []string
	ACR             string
}

// AuthorizationDecisionKind is closed so scripts and native policies cannot
// invent a new protocol transition. Skip means the policy abstains; the native
// provider still performs consent and token issuance decisions.
type AuthorizationDecisionKind string

const (
	AuthorizationAllow AuthorizationDecisionKind = "allow"
	AuthorizationDeny  AuthorizationDecisionKind = "deny"
	AuthorizationSkip  AuthorizationDecisionKind = "skip"
)

// AuthorizationEvidence is a declared, stable native fact identifier. It is
// intentionally metadata only: a policy cannot provide a credential, claim,
// or mutable proof value through this contract.
type AuthorizationEvidence struct {
	ID string
}

// AuthorizationDecision contains only a closed transition, a stable denial
// diagnostic, and declared evidence identifiers. DiagnosticID is accepted only
// for deny and must be a bounded identifier, never exception text.
type AuthorizationDecision struct {
	Kind         AuthorizationDecisionKind
	DiagnosticID string
	Evidence     []AuthorizationEvidence
}

const maxAuthorizationEvidence = 16

func (in AuthorizationInput) Clone() AuthorizationInput {
	in.Subject.Groups = append([]string(nil), in.Subject.Groups...)
	in.Subject.Roles = append([]string(nil), in.Subject.Roles...)
	in.Request.Scopes = append([]string(nil), in.Request.Scopes...)
	in.Request.Audience = append([]string(nil), in.Request.Audience...)
	in.Request.Prompt = append([]string(nil), in.Request.Prompt...)
	in.Authentication.AMR = append([]string(nil), in.Authentication.AMR...)
	return in
}

func (d AuthorizationDecision) Validate() error {
	switch d.Kind {
	case AuthorizationAllow, AuthorizationDeny, AuthorizationSkip:
	default:
		return errors.Errorf("unsupported authorization decision %q", d.Kind)
	}
	if d.Kind == AuthorizationDeny {
		if !stableDiagnosticID(d.DiagnosticID) {
			return errors.New("authorization denial requires a stable diagnostic ID")
		}
	} else if d.DiagnosticID != "" {
		return errors.New("only authorization denial may include a diagnostic ID")
	}
	if len(d.Evidence) > maxAuthorizationEvidence {
		return errors.Errorf("authorization evidence exceeds maximum of %d", maxAuthorizationEvidence)
	}
	seen := map[string]struct{}{}
	for _, evidence := range d.Evidence {
		if !stableDiagnosticID(evidence.ID) {
			return errors.New("authorization evidence requires a stable ID")
		}
		if _, duplicate := seen[evidence.ID]; duplicate {
			return errors.Errorf("duplicate authorization evidence %q", evidence.ID)
		}
		seen[evidence.ID] = struct{}{}
	}
	return nil
}

func (d AuthorizationDecision) Clone() AuthorizationDecision {
	d.Evidence = append([]AuthorizationEvidence(nil), d.Evidence...)
	return d
}

func stableDiagnosticID(value string) bool {
	if len(value) == 0 || len(value) > 96 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return strings.TrimSpace(value) == value
}

// NormalizeAuthorizationDecision returns a deterministic value for audit and
// persistence boundaries. Evidence is sorted by its stable ID.
func NormalizeAuthorizationDecision(decision AuthorizationDecision) (AuthorizationDecision, error) {
	if err := decision.Validate(); err != nil {
		return AuthorizationDecision{}, err
	}
	decision = decision.Clone()
	sort.Slice(decision.Evidence, func(i, j int) bool { return decision.Evidence[i].ID < decision.Evidence[j].ID })
	return decision, nil
}
