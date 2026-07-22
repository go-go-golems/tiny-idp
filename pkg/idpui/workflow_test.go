package idpui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestDefaultRendererRendersOnlyValidatedWorkflowDescriptors(t *testing.T) {
	page := workflowPage(t)
	renderer, err := idpui.NewDefaultRenderer()
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, renderer.RenderWorkflow(context.Background(), &output, page))
	html := output.String()
	assert.Contains(t, html, `name="display_name"`)
	assert.Contains(t, html, `name="password" type="password" value=""`)
	assert.Contains(t, html, `minlength="15" maxlength="1024"`)
	assert.Contains(t, html, `Use at least 15 characters.`)
	assert.Contains(t, html, `value="submit"`)
	assert.Contains(t, html, `value="deny" formnovalidate`)
	assert.NotContains(t, html, "secret-value")
	assert.NotContains(t, html, "<script>")
}

func TestWorkflowPasswordRejectionHasActionablePublicCopy(t *testing.T) {
	errorMessage := idpui.WorkflowFieldError{Field: idpworkflow.FieldPassword, Code: idpworkflow.ErrorRejected}
	assert.Equal(t, "Use at least 15 characters and choose a password that is difficult to guess.", errorMessage.Summary())
}

func TestWorkflowEmailCodeFailureHasSpecificPublicCopy(t *testing.T) {
	assert.Equal(t, "This verification code has expired. Restart registration to receive a new code.", (idpui.WorkflowFieldError{Field: idpworkflow.FieldEmailCode, Code: idpworkflow.ErrorExpired}).Summary())
	assert.Equal(t, "Too many incorrect verification codes were entered. Request a new code to try again.", (idpui.WorkflowFieldError{Field: idpworkflow.FieldEmailCode, Code: idpworkflow.ErrorAttemptsExceeded}).Summary())
	assert.Equal(t, "No more verification codes can be sent for this registration. Enter the most recent code or restart registration.", (idpui.WorkflowFieldError{Field: idpworkflow.FieldEmailCode, Code: idpworkflow.ErrorResendLimited}).Summary())
}

func TestWorkflowDuplicateIdentityHasSafeGlobalRecoveryCopy(t *testing.T) {
	errorMessage := idpui.WorkflowGlobalError{Code: idpui.WorkflowErrorDuplicateIdentity}
	assert.Equal(t, "An account already uses this email address. Return to the application to sign in, or restart signup with a different email address.", errorMessage.Summary())
	page := workflowPage(t)
	page.Form.RedirectOrigin = "https://app.example"
	page.Error = &errorMessage
	renderer, err := idpui.NewDefaultRenderer()
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, renderer.RenderWorkflow(context.Background(), &output, page))
	assert.Contains(t, output.String(), "An account already uses this email address.")
	assert.Contains(t, output.String(), "Return to application")
}

func TestWorkflowDuplicateDisplayNameHasSafeRecoveryCopy(t *testing.T) {
	errorMessage := idpui.WorkflowGlobalError{Code: idpui.WorkflowErrorDuplicateDisplayName}
	assert.Equal(t, "That display name was claimed while your signup was in progress. Return to the application and restart signup with a different display name.", errorMessage.Summary())
	assert.Equal(t, "That display name is already in use. Choose another.", (idpui.WorkflowFieldError{Field: idpworkflow.FieldDisplayName, Code: idpworkflow.ErrorRejected}).Summary())
}

func TestWorkflowPageRejectsRenderedSecretAndInvalidDescriptor(t *testing.T) {
	page := workflowPage(t)
	page.Fields[1].Value = "secret-value"
	require.Error(t, page.Validate())
	page = workflowPage(t)
	page.Fields[0].Descriptor.InputName = `"><script>`
	page.Fields[0].Descriptor.Kind = "html"
	require.Error(t, page.Validate())

	page = workflowPage(t)
	page.Form.RedirectOrigin = "https://app.example/path"
	require.Error(t, page.Validate())
}

func workflowPage(t *testing.T) idpui.WorkflowPage {
	t.Helper()
	registry := idpworkflow.DefaultRegistry()
	displayName, ok := registry.Field(idpworkflow.FieldDisplayName)
	require.True(t, ok)
	password, ok := registry.Field(idpworkflow.FieldPassword)
	require.True(t, ok)
	submit, ok := registry.Action(idpworkflow.ActionSubmit)
	require.True(t, ok)
	deny, ok := registry.Action(idpworkflow.ActionDeny)
	require.True(t, ok)
	return idpui.WorkflowPage{
		DocumentTitle: "Create <script> account",
		ClientID:      "example-client",
		Form:          idpui.WorkflowForm{ActionURL: "https://idp.example/authorize", InteractionField: idpui.InteractionFieldName, Interaction: strings.Repeat("a", 32), ContinuationField: idpui.WorkflowContinuationFieldName, Continuation: strings.Repeat("c", 32), CSRFField: idpui.CSRFFieldName, CSRFToken: strings.Repeat("b", 32), ActionField: idpui.ActionFieldName},
		Fields:        []idpui.WorkflowField{{Descriptor: displayName, Value: "Ada"}, {Descriptor: password}},
		Actions:       []idpui.WorkflowAction{{Descriptor: submit}, {Descriptor: deny}},
		Errors:        []idpui.WorkflowFieldError{{Field: idpworkflow.FieldDisplayName, Code: idpworkflow.ErrorInvalid}},
	}
}
