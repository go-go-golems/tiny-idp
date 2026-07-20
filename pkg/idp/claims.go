package idp

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/pkg/errors"
)

// ClaimsPolicy may add bounded, JSON-safe application claims to an already
// native-filtered claim set. It cannot set protocol-owned or existing native
// claims, and it has no token, session, Fosite, or signing-key authority.
type ClaimsPolicy interface {
	Claims(context.Context, ClaimsInput) (ClaimsOutput, error)
}

type ClaimsInput struct {
	Subject       AuthorizationSubject
	Client        AuthorizationClient
	GrantedScopes []string
	Base          map[string]json.RawMessage
}

type ClaimsOutput struct{ Additional map[string]json.RawMessage }

const maxAdditionalClaims = 16

var protectedClaimNames = map[string]struct{}{
	"iss": {}, "sub": {}, "aud": {}, "exp": {}, "nbf": {}, "iat": {}, "jti": {}, "nonce": {}, "auth_time": {}, "acr": {}, "amr": {}, "at_hash": {}, "c_hash": {}, "s_hash": {},
}

func (in ClaimsInput) Clone() ClaimsInput {
	in.Subject.Groups = append([]string(nil), in.Subject.Groups...)
	in.Subject.Roles = append([]string(nil), in.Subject.Roles...)
	in.GrantedScopes = append([]string(nil), in.GrantedScopes...)
	in.Base = cloneJSONClaims(in.Base)
	return in
}

func (out ClaimsOutput) Validate(base map[string]json.RawMessage) error {
	if len(out.Additional) > maxAdditionalClaims {
		return errors.Errorf("additional claims exceed maximum of %d", maxAdditionalClaims)
	}
	for name, value := range out.Additional {
		if !stableDiagnosticID(name) {
			return errors.Errorf("claim name %q is not a stable identifier", name)
		}
		if _, protected := protectedClaimNames[name]; protected {
			return errors.Errorf("claim %q is protocol-owned", name)
		}
		if _, exists := base[name]; exists {
			return errors.Errorf("claim %q is native-owned", name)
		}
		if len(value) == 0 || len(value) > 4096 || !json.Valid(value) {
			return errors.Errorf("claim %q must contain bounded valid JSON", name)
		}
	}
	return nil
}

func MergeClaims(base map[string]json.RawMessage, output ClaimsOutput) (map[string]json.RawMessage, error) {
	if err := output.Validate(base); err != nil {
		return nil, err
	}
	merged := cloneJSONClaims(base)
	for name, value := range output.Additional {
		merged[name] = append(json.RawMessage(nil), value...)
	}
	return merged, nil
}

func cloneJSONClaims(source map[string]json.RawMessage) map[string]json.RawMessage {
	cloned := make(map[string]json.RawMessage, len(source))
	for name, value := range source {
		cloned[name] = append(json.RawMessage(nil), value...)
	}
	return cloned
}

func SortedClaimNames(claims map[string]json.RawMessage) []string {
	names := make([]string, 0, len(claims))
	for name := range claims {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
