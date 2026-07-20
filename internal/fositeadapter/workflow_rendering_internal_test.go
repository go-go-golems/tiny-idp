package fositeadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestRenderWorkflowPreservesInteractionSecurityEnvelope(t *testing.T) {
	renderer, err := idpui.NewDefaultRenderer()
	require.NoError(t, err)
	provider := &Provider{workflowUI: renderer, audit: idp.NoopSink{}, clock: time.Now}
	request := httptest.NewRequest(http.MethodGet, "https://idp.example/authorize", nil)
	response := httptest.NewRecorder()

	provider.renderWorkflow(response, request, http.StatusOK, workflowRenderingPage(t))

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Equal(t, "no-cache", response.Header().Get("Pragma"))
	require.Equal(t, "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self' https://app.example; base-uri 'none'", response.Header().Get("Content-Security-Policy"))
	require.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
	body := response.Body.String()
	require.Contains(t, body, `action="https://idp.example/authorize"`)
	require.Contains(t, body, `name="interaction"`)
	require.Contains(t, body, `name="workflow_continuation"`)
	require.Contains(t, body, `name="csrf_token"`)
}

func workflowRenderingPage(t *testing.T) idpui.WorkflowPage {
	t.Helper()
	registry := idpworkflow.DefaultRegistry()
	email, ok := registry.Field(idpworkflow.FieldEmail)
	require.True(t, ok)
	password, ok := registry.Field(idpworkflow.FieldPassword)
	require.True(t, ok)
	submit, ok := registry.Action(idpworkflow.ActionSubmit)
	require.True(t, ok)

	return idpui.WorkflowPage{
		DocumentTitle: "Create an account",
		Form: idpui.WorkflowForm{
			ActionURL:         "https://idp.example/authorize",
			RedirectOrigin:    "https://app.example",
			InteractionField:  idpui.InteractionFieldName,
			Interaction:       strings.Repeat("a", 32),
			ContinuationField: idpui.WorkflowContinuationFieldName,
			Continuation:      strings.Repeat("c", 32),
			CSRFField:         idpui.CSRFFieldName,
			CSRFToken:         strings.Repeat("b", 32),
			ActionField:       idpui.ActionFieldName,
		},
		Fields:  []idpui.WorkflowField{{Descriptor: email, Value: "ada@example.test"}, {Descriptor: password}},
		Actions: []idpui.WorkflowAction{{Descriptor: submit}},
	}
}
