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
