package idpprogram_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func TestValidatePublicJSONRejectsSensitiveCarryButResumeAllowsIt(t *testing.T) {
	schemas := map[string]idpprogram.Schema{
		"form": {
			ID: "form", Kind: idpprogram.SchemaKindObject, MaxBytes: 128,
			Fields: map[string]idpprogram.SchemaField{
				"email":      {Ref: "text", Required: true},
				"credential": {Ref: "text", Required: true, Sensitive: true},
			},
		},
		"text": {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 64, MaxLength: 32},
	}
	value := []byte(`{"email":"a@example.test","credential":"secret"}`)
	require.NoError(t, idpprogram.ValidateJSON(schemas, "form", value))
	err := idpprogram.ValidatePublicJSON(schemas, "form", value)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sensitive")
}

func TestValidateJSONRejectsOversizedUnknownAndMultipleValues(t *testing.T) {
	schemas := map[string]idpprogram.Schema{
		"value": {ID: "value", Kind: idpprogram.SchemaKindString, MaxBytes: 5, MaxLength: 3},
	}
	for _, input := range [][]byte{[]byte(`"long"`), []byte(`"ok" "no"`)} {
		require.Error(t, idpprogram.ValidateJSON(schemas, "value", input))
	}
	require.Error(t, idpprogram.ValidateJSON(schemas, "missing", []byte(`"ok"`)))
}
