package idpprogram

// SchemaKind identifies the top-level representation accepted by a named
// schema. Phase 0 intentionally supports a small bounded data vocabulary.
type SchemaKind string

const (
	SchemaKindObject  SchemaKind = "object"
	SchemaKindString  SchemaKind = "string"
	SchemaKindBoolean SchemaKind = "boolean"
	SchemaKindInteger SchemaKind = "integer"
	SchemaKindBytes   SchemaKind = "bytes"
	SchemaKindArray   SchemaKind = "array"
)

// Valid reports whether k belongs to the Phase 0 schema vocabulary.
func (k SchemaKind) Valid() bool {
	switch k {
	case SchemaKindObject, SchemaKindString, SchemaKindBoolean, SchemaKindInteger, SchemaKindBytes, SchemaKindArray:
		return true
	default:
		return false
	}
}

// Schema is a named bounded input or output contract. Object fields are
// deterministic because canonical JSON sorts map keys.
type Schema struct {
	ID         string                 `json:"id"`
	Kind       SchemaKind             `json:"kind"`
	MaxBytes   int                    `json:"maxBytes"`
	Fields     map[string]SchemaField `json:"fields,omitempty"`
	MaxLength  int                    `json:"maxLength,omitempty"`
	Minimum    *int64                 `json:"minimum,omitempty"`
	Maximum    *int64                 `json:"maximum,omitempty"`
	Items      string                 `json:"items,omitempty"`
	MaxItems   int                    `json:"maxItems,omitempty"`
	Additional bool                   `json:"additional"`
}

// SchemaField describes one object property. Ref names another registered
// schema; inline recursive schemas are intentionally unsupported.
type SchemaField struct {
	Ref       string `json:"ref"`
	Required  bool   `json:"required,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}
