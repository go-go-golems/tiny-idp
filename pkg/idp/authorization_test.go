package idp_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

func TestAuthorizationInputCloneDoesNotShareProviderSlices(t *testing.T) {
	input := idp.AuthorizationInput{
		Subject:        idp.AuthorizationSubject{Groups: []string{"staff"}, Roles: []string{"reader"}},
		Request:        idp.AuthorizationRequest{Scopes: []string{"openid"}, Audience: []string{"api"}, Prompt: []string{"login"}},
		Authentication: idp.AuthenticationView{AuthenticatedAt: time.Now(), AMR: []string{"pwd"}},
	}
	cloned := input.Clone()
	cloned.Subject.Groups[0] = "changed"
	cloned.Request.Scopes[0] = "email"
	cloned.Authentication.AMR[0] = "passkey"
	assert.Equal(t, "staff", input.Subject.Groups[0])
	assert.Equal(t, "openid", input.Request.Scopes[0])
	assert.Equal(t, "pwd", input.Authentication.AMR[0])
}

func TestAuthorizationDecisionAcceptsOnlyBoundedStableValues(t *testing.T) {
	decision, err := idp.NormalizeAuthorizationDecision(idp.AuthorizationDecision{
		Kind:         idp.AuthorizationDeny,
		DiagnosticID: "policy.member_required",
		Evidence:     []idp.AuthorizationEvidence{{ID: "email_verified"}, {ID: "membership.v1"}},
	})
	require.NoError(t, err)
	assert.Equal(t, []idp.AuthorizationEvidence{{ID: "email_verified"}, {ID: "membership.v1"}}, decision.Evidence)

	for _, invalid := range []idp.AuthorizationDecision{
		{Kind: "redirect"},
		{Kind: idp.AuthorizationAllow, DiagnosticID: "not_allowed"},
		{Kind: idp.AuthorizationDeny, DiagnosticID: "raw exception: secret"},
		{Kind: idp.AuthorizationDeny, DiagnosticID: "denied", Evidence: []idp.AuthorizationEvidence{{ID: "same"}, {ID: "same"}}},
	} {
		assert.Error(t, invalid.Validate(), "%+v", invalid)
	}
}
