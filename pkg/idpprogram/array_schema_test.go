package idpprogram_test

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func TestArraySchemaValidatesItemTypeAndBound(t *testing.T) {
	schemas := map[string]idpprogram.Schema{
		"text": {ID: "text", Kind: idpprogram.SchemaKindString, MaxBytes: 32, MaxLength: 8},
		"list": {ID: "list", Kind: idpprogram.SchemaKindArray, MaxBytes: 64, Items: "text", MaxItems: 2},
	}
	if err := idpprogram.ValidateJSON(schemas, "list", json.RawMessage(`["one","two"]`)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`["one","two","three"]`, `["one",2]`, `{}`} {
		if err := idpprogram.ValidateJSON(schemas, "list", json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid array %s accepted", raw)
		}
	}
}
