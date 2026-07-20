package idppolicy_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idppolicy"
)

func TestExecutorRunsBoundedAuthorizationAndClaimsProviders(t *testing.T) {
	executor, err := idppolicy.New(context.Background(), policySource, 1, idppolicy.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })

	decision, err := executor.Authorize(context.Background(), idp.AuthorizationInput{Subject: idp.AuthorizationSubject{Subject: "ada"}})
	require.NoError(t, err)
	assert.Equal(t, idp.AuthorizationDeny, decision.Kind)
	assert.Equal(t, "policy.member_required", decision.DiagnosticID)
	assert.Equal(t, []idp.AuthorizationEvidence{{ID: "evidence.community_membership"}}, decision.Evidence)

	claims, err := executor.Claims(context.Background(), idp.ClaimsInput{Subject: idp.AuthorizationSubject{Subject: "ada"}, Base: map[string]json.RawMessage{"sub": json.RawMessage(`"ada"`)}})
	require.NoError(t, err)
	assert.JSONEq(t, `"member"`, string(claims.Additional["community_role"]))
}

func TestExecutorRetainsNativePolicyValidation(t *testing.T) {
	executor, err := idppolicy.New(context.Background(), protectedClaimsSource, 1, idppolicy.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })

	_, err = executor.Claims(context.Background(), idp.ClaimsInput{Base: map[string]json.RawMessage{"sub": json.RawMessage(`"ada"`)}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "protocol-owned")
}

func TestExecutorRunsBoundedPresentationProvider(t *testing.T) {
	executor, err := idppolicy.New(context.Background(), presentationSource, 1, idppolicy.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
	output, err := executor.Present(context.Background(), idp.PresentationInput{Kind: idp.PresentationConsent, ClientID: "message-app", RequestedScope: []string{"openid"}})
	require.NoError(t, err)
	assert.Equal(t, "Review message-app access", output.DocumentTitle)
}

const policySource = `
const A = require("tinyidp").v1;
module.exports = A.program("policy", p => {
  const decide = A.lambda("authorization.decide", {
    kind:"provider", input:"authorizationInput", output:"authorizationOutput",
    outcomes:["complete","error"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:8192,
    run: ctx => A.result.complete({Kind:"deny", DiagnosticID:"policy.member_required", Evidence:[{ID:"evidence.community_membership"}]})
  });
  const additional = A.lambda("claims.additional", {
    kind:"provider", input:"claimsInput", output:"claimsOutput",
    outcomes:["complete","error"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:8192,
    run: ctx => A.result.complete({Additional:{community_role:"member"}})
  });
  p.provider("authorization", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{decide}});
  p.provider("claims", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{additional}});
});`

const protectedClaimsSource = `
const A = require("tinyidp").v1;
module.exports = A.program("claims", p => {
  const additional = A.lambda("claims.additional", {
    kind:"provider", input:"claimsInput", output:"claimsOutput",
    outcomes:["complete"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:8192,
    run: ctx => A.result.complete({Additional:{sub:"other"}})
  });
  p.provider("claims", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{additional}});
});`

const presentationSource = `
const A = require("tinyidp").v1;
module.exports = A.program("presentation", p => {
  const render = A.lambda("presentation.render", {
    kind:"provider", input:"presentationInput", output:"presentationOutput",
    outcomes:["complete"], effects:[], capabilities:[], timeoutMs:100, maxCapabilityCalls:0, maxOutputBytes:1024,
    run: ctx => A.result.complete({DocumentTitle:"Review message-app access"})
  });
  p.provider("presentation", "default", {version:1, state:"virtual", replayProtection:"none", revocation:"none", handlers:{render}});
});`
