package idpworkflow_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestValidatePresentationBindsRegistryAndDeclaredEdge(t *testing.T) {
	presentation := validPresentation()
	validated, err := idpworkflow.ValidatePresentation(presentationProgram(), "signup", "start", presentation, idpworkflow.DefaultRegistry(), 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "submittedInput", validated.InputSchema)
	assert.Equal(t, idpworkflow.FieldDisplayName, validated.Fields[0].ID)
	assert.Equal(t, idpworkflow.ActionSubmit, validated.Actions[0].ID)

	presentation.PublicValues[idpworkflow.FieldDisplayName] = "changed"
	assert.Equal(t, "Ada", validated.Presentation.PublicValues[idpworkflow.FieldDisplayName], "validated result must be defensive")
}

func TestValidatePresentationRejectsAuthorityAndSecretViolations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*idpworkflow.Presentation)
		want string
	}{
		{name: "unknown field", edit: func(p *idpworkflow.Presentation) { p.Fields[0] = "html" }, want: "not registered"},
		{name: "unknown action", edit: func(p *idpworkflow.Presentation) { p.Actions[0] = "admin" }, want: "not registered"},
		{name: "wrong edge", edit: func(p *idpworkflow.Presentation) { p.ResumeHandler = "start" }, want: "no present edge"},
		{name: "long expiry", edit: func(p *idpworkflow.Presentation) { p.ExpiresIn = time.Hour }, want: "expiry"},
		{name: "secret redisplay", edit: func(p *idpworkflow.Presentation) { p.PublicValues[idpworkflow.FieldPassword] = "secret" }, want: "redisplayed"},
		{name: "unknown error", edit: func(p *idpworkflow.Presentation) {
			p.Errors = []idpworkflow.FieldError{{Field: idpworkflow.FieldEmail, Code: "backend_text"}}
		}, want: "invalid field error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			presentation := validPresentation()
			test.edit(&presentation)
			_, err := idpworkflow.ValidatePresentation(presentationProgram(), "signup", "start", presentation, idpworkflow.DefaultRegistry(), 10*time.Minute)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.want)
		})
	}
}

func validPresentation() idpworkflow.Presentation {
	return idpworkflow.Presentation{
		Title: "Create account", ResumeHandler: "submitted",
		Fields:       []idpworkflow.FieldID{idpworkflow.FieldDisplayName, idpworkflow.FieldEmail, idpworkflow.FieldPassword},
		Actions:      []idpworkflow.ActionID{idpworkflow.ActionSubmit, idpworkflow.ActionDeny},
		PublicValues: map[idpworkflow.FieldID]string{idpworkflow.FieldDisplayName: "Ada"},
		Carry:        []byte(`{"displayName":"Ada"}`), ExpiresIn: 5 * time.Minute,
	}
}

func presentationProgram() idpprogram.Program {
	return idpprogram.Program{
		Schemas: map[string]idpprogram.Schema{
			"submittedInput": {ID: "submittedInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024, Fields: map[string]idpprogram.SchemaField{"displayName": {Ref: "text", Required: true}}},
			"text":           {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 128, MaxLength: 120},
		},
		Lambdas: map[string]idpprogram.LambdaSpec{
			"start":     {ID: "start", InputSchema: "submittedInput", AllowedOutcomes: []idpprogram.OutcomeKind{idpprogram.OutcomePresent}},
			"submitted": {ID: "submitted", InputSchema: "submittedInput"},
		},
		Workflows: map[string]idpprogram.Workflow{
			"signup": {ID: "signup", Version: 1, EntryHandler: "start", Handlers: map[string]idpprogram.HandlerSpec{
				"start":     {ID: "start", LambdaID: "start", ContinuationEdges: []idpprogram.ContinuationEdge{{OutcomeKind: idpprogram.OutcomePresent, HandlerID: "submitted", InputSchema: "submittedInput"}}},
				"submitted": {ID: "submitted", LambdaID: "submitted"},
			}},
		},
	}
}
