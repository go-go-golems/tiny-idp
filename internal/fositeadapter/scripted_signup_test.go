package fositeadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestWorkflowFieldErrorsProjectsValidatedScriptErrors(t *testing.T) {
	got := workflowFieldErrors([]idpworkflow.FieldError{{Field: idpworkflow.FieldDisplayName, Code: idpworkflow.ErrorRejected}})
	assert.Equal(t, []idpui.WorkflowFieldError{{Field: idpworkflow.FieldDisplayName, Code: idpworkflow.ErrorRejected}}, got)
	assert.Nil(t, workflowFieldErrors(nil))
}
