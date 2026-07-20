package idp_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

func TestMergeClaimsPreservesNativeAndRejectsProtectedNames(t *testing.T) {
	base := map[string]json.RawMessage{"email": json.RawMessage(`"ada@example.test"`)}
	merged, err := idp.MergeClaims(base, idp.ClaimsOutput{Additional: map[string]json.RawMessage{"community_role": json.RawMessage(`"member"`)}})
	require.NoError(t, err)
	assert.JSONEq(t, `"ada@example.test"`, string(merged["email"]))
	assert.JSONEq(t, `"member"`, string(merged["community_role"]))

	for _, output := range []idp.ClaimsOutput{
		{Additional: map[string]json.RawMessage{"sub": json.RawMessage(`"other"`)}},
		{Additional: map[string]json.RawMessage{"email": json.RawMessage(`"other@example.test"`)}},
		{Additional: map[string]json.RawMessage{"bad claim": json.RawMessage(`true`)}},
		{Additional: map[string]json.RawMessage{"invalid": json.RawMessage(`{`)}},
	} {
		assert.Error(t, output.Validate(base), "%+v", output)
	}
}
