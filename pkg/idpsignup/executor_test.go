package idpsignup_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestDefaultProgramPresentsAndRequestsNativeSignupCommit(t *testing.T) {
	executor, err := idpsignup.New(context.Background(), "", 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })

	presentation, err := executor.Start(context.Background(), idpsignup.StartInput{ClientID: "test-client", RedirectURI: "https://client.example.test/callback", RequestedScope: "openid profile", InteractionID: "test-interaction"})
	require.NoError(t, err)
	assert.Equal(t, "submitted", presentation.Presentation.ResumeHandler)
	require.Len(t, presentation.Fields, 4)

	submission, err := idpworkflow.ParseSubmission(presentation.Fields, presentation.Actions, url.Values{
		idpui.InteractionFieldName:          {"interaction"},
		idpui.WorkflowContinuationFieldName: {"continuation"},
		idpui.CSRFFieldName:                 {"csrf"},
		idpui.ActionFieldName:               {"submit"},
		"display_name":                      {"Ada"},
		"email":                             {"ada@example.test"},
		"password":                          {"correct horse battery staple"},
		"password_confirmation":             {"correct horse battery staple"},
	})
	require.NoError(t, err)
	defer submission.DestroySecrets()
	outcome, err := executor.Submit(context.Background(), submission.PublicValues, map[string]idpworkflow.SecretHandle{
		"password":             submission.Secrets[idpworkflow.FieldPassword],
		"passwordConfirmation": submission.Secrets[idpworkflow.FieldPasswordConfirmation],
	})
	require.NoError(t, err)
	assert.Equal(t, idpprogram.OutcomeCommit, outcome.Kind)
	assert.Len(t, outcome.Effects, 2)
}

func TestInvitationProgramRequestsNativeInvitationConsumption(t *testing.T) {
	source := strings.ReplaceAll(idpsignup.DefaultSource,
		"A.field.password(), A.field.passwordConfirmation()",
		"A.field.password(), A.field.passwordConfirmation(), A.field.inviteCode()")
	source = strings.ReplaceAll(source,
		`effects: ["createLocalIdentity", "attachPasswordCredential"]`,
		`effects: ["createLocalIdentity", "attachPasswordCredential", "consumeInvitation"]`)
	source = strings.ReplaceAll(source,
		"passwordConfirmation: ctx.secret.passwordConfirmation,",
		"passwordConfirmation: ctx.secret.passwordConfirmation, inviteCode: ctx.input.inviteCode,")
	executor, err := idpsignup.New(context.Background(), source, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
	presentation, err := executor.Start(context.Background(), idpsignup.StartInput{ClientID: "message-app", RedirectURI: "https://client.example.test/callback", RequestedScope: "openid", InteractionID: "test-interaction"})
	require.NoError(t, err)
	submission, err := idpworkflow.ParseSubmission(presentation.Fields, presentation.Actions, url.Values{
		idpui.InteractionFieldName: {"interaction"}, idpui.WorkflowContinuationFieldName: {"continuation"}, idpui.CSRFFieldName: {"csrf"}, idpui.ActionFieldName: {"submit"},
		"display_name": {"Ada"}, "email": {"ada@example.test"}, "password": {"correct horse battery staple"}, "password_confirmation": {"correct horse battery staple"}, "invite_code": {"invite-code"},
	})
	require.NoError(t, err)
	defer submission.DestroySecrets()
	outcome, err := executor.Submit(context.Background(), submission.PublicValues, map[string]idpworkflow.SecretHandle{"password": submission.Secrets[idpworkflow.FieldPassword], "passwordConfirmation": submission.Secrets[idpworkflow.FieldPasswordConfirmation]})
	require.NoError(t, err)
	require.Len(t, outcome.Effects, 3)
	assert.Equal(t, idpprogram.EffectConsumeInvitation, outcome.Effects[2].Kind)
	assert.JSONEq(t, `{"code":"invite-code"}`, string(outcome.Effects[2].Payload))
}

func TestEmailVerifiedProgramDeclaresChallengeThenPasswordWorkflow(t *testing.T) {
	executor, err := idpsignup.New(context.Background(), idpsignup.EmailVerifiedSource, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
	workflow := executor.Program().Workflows[idpsignup.WorkflowID]
	assert.Equal(t, "start", workflow.EntryHandler)
	assert.Equal(t, idpprogram.OutcomeChallenge, workflow.Handlers["submitted"].ContinuationEdges[0].OutcomeKind)
	assert.Equal(t, "passwordSubmitted", workflow.Handlers["emailVerified"].ContinuationEdges[0].HandlerID)
	outcome, err := executor.Submit(context.Background(), map[idpworkflow.FieldID]string{idpworkflow.FieldDisplayName: "Ada", idpworkflow.FieldEmail: "ada@example.test"}, nil)
	require.NoError(t, err)
	assert.Equal(t, idpprogram.OutcomeChallenge, outcome.Kind)
	assert.NotEmpty(t, outcome.Challenge)
}

func TestExecutorRunsDeclarativeEmbeddedTests(t *testing.T) {
	source := `const A = require("tinyidp").v1;
module.exports = A.program("embedded-tests", p => {
  const start = A.lambda("signup.start", { input:"signupStartInput", output:"signupResult", outcomes:["complete"], effects:[], capabilities:[], timeoutMs:250, maxCapabilityCalls:0, maxOutputBytes:1024, run: ctx => A.result.complete() });
  p.workflow("signup", { version:1, entry:"start", handlers:{start}, edges:[] });
  p.test("start-completes", { lambda:"signup.start", input:{ clientId:"client", redirectUri:"https://client.example.test/callback", requestedScope:"openid", interactionId:"interaction", hasBrowserSession:false }, expectedKind:"complete" });
});`
	executor, err := idpsignup.New(context.Background(), source, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
	results := executor.RunTests(context.Background())
	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "%+v", results[0])
}

func TestExecutorRunsDeclarativeTestsWithConfiguredDeterministicFake(t *testing.T) {
	source := `const A = require("tinyidp").v1;
module.exports = A.program("fake-tests", p => {
  p.capabilities({"clock.now": {version:1}});
  const start = A.lambda("signup.start", { input:"signupStartInput", output:"signupResult", outcomes:["complete","deny"], effects:[], capabilities:["clock.now"], timeoutMs:250, maxCapabilityCalls:1, maxOutputBytes:1024, run: async ctx => { const now = await ctx.cap.clock.now({}); return now.unixMillis === 42 ? A.result.complete() : A.result.deny(); } });
  p.workflow("signup", { version:1, entry:"start", handlers:{start}, edges:[] });
  p.test("configured-clock", { lambda:"signup.start", input:{ clientId:"client", redirectUri:"https://client.example.test/callback", requestedScope:"openid", interactionId:"interaction", hasBrowserSession:false }, expectedKind:"complete", fakes:{"clock.now": {unixMillis:42}} });
});`
	executor, err := idpsignup.New(context.Background(), source, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })

	results := executor.RunTests(context.Background())
	require.Len(t, results, 1)
	assert.True(t, results[0].Passed, "%+v", results[0])
}

func TestExecutorReportsBoundedInvocationMetrics(t *testing.T) {
	executor, err := idpsignup.New(context.Background(), idpsignup.EmailVerifiedSource, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, executor.Close(context.Background())) })
	_, err = executor.Submit(context.Background(), map[idpworkflow.FieldID]string{idpworkflow.FieldDisplayName: "Ada", idpworkflow.FieldEmail: "ada@example.test"}, nil)
	require.NoError(t, err)
	metrics := executor.Metrics()
	assert.Equal(t, uint64(1), metrics.Invocations)
	assert.Equal(t, uint64(1), metrics.Challenge)
	assert.Zero(t, metrics.Failures)
	assert.Greater(t, metrics.LatencyNanos, uint64(0))
}
