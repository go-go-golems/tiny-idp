package idpworkflow_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestDefaultRegistryFreezesSignupFieldAndActionPolicies(t *testing.T) {
	registry := idpworkflow.DefaultRegistry()
	assert.Equal(t, []idpworkflow.FieldID{
		idpworkflow.FieldDisplayName,
		idpworkflow.FieldEmail,
		idpworkflow.FieldInviteCode,
		idpworkflow.FieldPassword,
		idpworkflow.FieldPasswordConfirmation,
	}, registry.FieldIDs())
	password, ok := registry.Field(idpworkflow.FieldPassword)
	require.True(t, ok)
	assert.True(t, password.Sensitive)
	assert.Equal(t, idpworkflow.RedisplayNever, password.Redisplay)
	assert.Equal(t, idpworkflow.NormalizeNone, password.Normalize)
	email, ok := registry.Field(idpworkflow.FieldEmail)
	require.True(t, ok)
	assert.Equal(t, idpworkflow.NormalizeTrimLower, email.Normalize)
	assert.Equal(t, 320, email.MaxLength)
	assert.Equal(t, []idpworkflow.ActionID{idpworkflow.ActionDeny, idpworkflow.ActionSubmit}, registry.ActionIDs())
	deny, ok := registry.Action(idpworkflow.ActionDeny)
	require.True(t, ok)
	assert.True(t, deny.SkipFormValidation)
}

func TestRegistryRejectsAuthorityChangingDescriptors(t *testing.T) {
	_, err := idpworkflow.NewRegistry([]idpworkflow.FieldDescriptor{{
		ID: "password", InputName: "password", Label: "Password", Kind: idpworkflow.ValueSecret,
		Normalize: idpworkflow.NormalizeTrim, MinLength: 1, MaxLength: 20,
		Sensitive: true, Redisplay: idpworkflow.RedisplayPublic,
	}}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret")

	_, err = idpworkflow.NewRegistry(nil, []idpworkflow.ActionDescriptor{{ID: idpworkflow.ActionSubmit, Label: "Submit", SkipFormValidation: true}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "form-validation")
}
