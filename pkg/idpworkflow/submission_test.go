package idpworkflow_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestParseSubmissionProjectsPublicValuesAndSecretsSeparately(t *testing.T) {
	registry := idpworkflow.DefaultRegistry()
	fields := selectedFields(t, registry, idpworkflow.FieldDisplayName, idpworkflow.FieldEmail, idpworkflow.FieldPassword)
	actions := selectedActions(t, registry, idpworkflow.ActionSubmit, idpworkflow.ActionDeny)
	result, err := idpworkflow.ParseSubmission(fields, actions, validForm())
	require.NoError(t, err)
	assert.Equal(t, idpworkflow.ActionSubmit, result.Action)
	assert.Equal(t, "Ada", result.PublicValues[idpworkflow.FieldDisplayName])
	assert.Equal(t, "ada@example.test", result.PublicValues[idpworkflow.FieldEmail])
	password, ok := result.ResolveSecret(result.Secrets[idpworkflow.FieldPassword])
	require.True(t, ok)
	assert.Equal(t, []byte("correct horse battery staple"), password)
	result.DestroySecrets()
	_, ok = result.ResolveSecret(result.Secrets[idpworkflow.FieldPassword])
	assert.False(t, ok, "destroyed secrets must not remain resolvable")
	_, present := result.PublicValues[idpworkflow.FieldPassword]
	assert.False(t, present, "a secret must not enter the public projection")
}

func TestParseSubmissionDoesNotRedisplayEmailVerificationCodes(t *testing.T) {
	registry := idpworkflow.DefaultRegistry()
	fields := selectedFields(t, registry, idpworkflow.FieldEmailCode)
	actions := selectedActions(t, registry, idpworkflow.ActionSubmit, idpworkflow.ActionDeny, idpworkflow.ActionResend)
	result, err := idpworkflow.ParseSubmission(fields, actions, url.Values{
		idpui.InteractionFieldName:          {"interaction"},
		idpui.WorkflowContinuationFieldName: {"continuation"},
		idpui.CSRFFieldName:                 {"csrf"},
		idpui.ActionFieldName:               {"submit"},
		"email_code":                        {"ABCDEFGH"},
	})
	require.NoError(t, err)
	_, present := result.PublicValues[idpworkflow.FieldEmailCode]
	assert.False(t, present, "email verification code must not enter the redisplay projection")
	emailCode, ok := result.ResolveSecret(result.Secrets[idpworkflow.FieldEmailCode])
	require.True(t, ok)
	assert.Equal(t, "ABCDEFGH", string(emailCode))
	clear(emailCode)
	result.DestroySecrets()
}

func TestParseSubmissionRejectsMalformedShapeAndValues(t *testing.T) {
	registry := idpworkflow.DefaultRegistry()
	fields := selectedFields(t, registry, idpworkflow.FieldDisplayName, idpworkflow.FieldEmail, idpworkflow.FieldPassword)
	actions := selectedActions(t, registry, idpworkflow.ActionSubmit, idpworkflow.ActionDeny)
	tests := []struct {
		name string
		edit func(url.Values)
		want string
	}{
		{name: "duplicate action", edit: func(v url.Values) { v[idpui.ActionFieldName] = []string{"submit", "deny"} }, want: "exactly once"},
		{name: "missing field", edit: func(v url.Values) { delete(v, "email") }, want: "missing field"},
		{name: "extra field", edit: func(v url.Values) { v.Set("admin", "true") }, want: "unexpected"},
		{name: "unknown action", edit: func(v url.Values) { v.Set(idpui.ActionFieldName, "admin") }, want: "unsupported action"},
		{name: "required empty", edit: func(v url.Values) { v.Set("display_name", "  ") }, want: "required"},
		{name: "bad email", edit: func(v url.Values) { v.Set("email", "not an email") }, want: "valid email"},
		{name: "short password", edit: func(v url.Values) { v.Set("password", "too short") }, want: "length bounds"},
		{name: "long password", edit: func(v url.Values) { v.Set("password", string(make([]byte, 1025))) }, want: "length bounds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			form := validForm()
			test.edit(form)
			_, err := idpworkflow.ParseSubmission(fields, actions, form)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.want)
		})
	}
}

func TestParseSubmissionDenyStillRequiresExactShapeButSkipsValueValidation(t *testing.T) {
	registry := idpworkflow.DefaultRegistry()
	fields := selectedFields(t, registry, idpworkflow.FieldDisplayName, idpworkflow.FieldEmail, idpworkflow.FieldPassword)
	actions := selectedActions(t, registry, idpworkflow.ActionSubmit, idpworkflow.ActionDeny)
	form := validForm()
	form.Set(idpui.ActionFieldName, "deny")
	form.Set("display_name", "")
	form.Set("email", "not-an-email")
	form.Set("password", "")
	_, err := idpworkflow.ParseSubmission(fields, actions, form)
	require.NoError(t, err)
}

func validForm() url.Values {
	return url.Values{
		idpui.InteractionFieldName:          {"interaction"},
		idpui.WorkflowContinuationFieldName: {"continuation"},
		idpui.CSRFFieldName:                 {"csrf"},
		idpui.ActionFieldName:               {"submit"},
		"display_name":                      {" Ada "},
		"email":                             {" ADA@EXAMPLE.TEST "},
		"password":                          {"correct horse battery staple"},
	}
}

func selectedFields(t *testing.T, registry *idpworkflow.Registry, ids ...idpworkflow.FieldID) []idpworkflow.FieldDescriptor {
	t.Helper()
	fields := make([]idpworkflow.FieldDescriptor, 0, len(ids))
	for _, id := range ids {
		field, ok := registry.Field(id)
		require.True(t, ok)
		fields = append(fields, field)
	}
	return fields
}

func selectedActions(t *testing.T, registry *idpworkflow.Registry, ids ...idpworkflow.ActionID) []idpworkflow.ActionDescriptor {
	t.Helper()
	actions := make([]idpworkflow.ActionDescriptor, 0, len(ids))
	for _, id := range ids {
		action, ok := registry.Action(id)
		require.True(t, ok)
		actions = append(actions, action)
	}
	return actions
}
