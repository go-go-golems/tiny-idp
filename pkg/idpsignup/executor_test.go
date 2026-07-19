package idpsignup_test

import (
	"context"
	"net/url"
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

	presentation, err := executor.Start(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "submitted", presentation.Presentation.ResumeHandler)
	require.Len(t, presentation.Fields, 4)

	submission, err := idpworkflow.ParseSubmission(presentation.Fields, presentation.Actions, url.Values{
		idpui.InteractionFieldName: {"interaction"},
		idpui.CSRFFieldName:        {"csrf"},
		idpui.ActionFieldName:      {"submit"},
		"display_name":             {"Ada"},
		"email":                    {"ada@example.test"},
		"password":                 {"correct horse battery staple"},
		"password_confirmation":    {"correct horse battery staple"},
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
