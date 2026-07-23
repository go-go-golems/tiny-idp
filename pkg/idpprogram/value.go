package idpprogram

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// ValidateJSON validates one bounded JSON value against a named program
// schema. It is runtime-independent and is shared by invocation and durable
// continuation boundaries.
func ValidateJSON(schemas map[string]Schema, schemaID string, encoded json.RawMessage) error {
	schema, ok := schemas[schemaID]
	if !ok {
		return errors.Errorf("schema %q is not registered", schemaID)
	}
	if len(encoded) == 0 || len(encoded) > schema.MaxBytes {
		return errors.Errorf("schema %q value size %d outside 1..%d", schemaID, len(encoded), schema.MaxBytes)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return errors.Wrap(err, "decode schema value")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return validateJSONValue(schemas, schema, value, map[string]bool{}, false)
}

// ValidatePublicJSON is ValidateJSON plus a prohibition on values in fields
// marked Sensitive. Continuation carry must use native secret references
// instead of persisting such fields directly.
func ValidatePublicJSON(schemas map[string]Schema, schemaID string, encoded json.RawMessage) error {
	schema, ok := schemas[schemaID]
	if !ok {
		return errors.Errorf("schema %q is not registered", schemaID)
	}
	if len(encoded) == 0 || len(encoded) > schema.MaxBytes {
		return errors.Errorf("schema %q value size %d outside 1..%d", schemaID, len(encoded), schema.MaxBytes)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return errors.Wrap(err, "decode public schema value")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return validateJSONValue(schemas, schema, value, map[string]bool{}, true)
}

func validateJSONValue(schemas map[string]Schema, schema Schema, value any, stack map[string]bool, publicOnly bool) error {
	if stack[schema.ID] {
		return errors.Errorf("recursive schema %q", schema.ID)
	}
	stack[schema.ID] = true
	defer delete(stack, schema.ID)

	switch schema.Kind {
	case SchemaKindObject:
		object, ok := value.(map[string]any)
		if !ok {
			return errors.Errorf("schema %q requires an object", schema.ID)
		}
		for name, field := range schema.Fields {
			fieldValue, exists := object[name]
			if field.Required && !exists {
				return errors.Errorf("schema %q requires field %q", schema.ID, name)
			}
			if !exists {
				continue
			}
			if publicOnly && field.Sensitive {
				return errors.Errorf("schema %q field %q is sensitive and cannot be persisted as public carry", schema.ID, name)
			}
			fieldSchema, ok := schemas[field.Ref]
			if !ok {
				return errors.Errorf("schema %q field %q references unknown schema %q", schema.ID, name, field.Ref)
			}
			if err := validateJSONValue(schemas, fieldSchema, fieldValue, stack, publicOnly); err != nil {
				return errors.Wrapf(err, "field %q", name)
			}
		}
		if !schema.Additional {
			for name := range object {
				if _, ok := schema.Fields[name]; !ok {
					return errors.Errorf("schema %q rejects additional field %q", schema.ID, name)
				}
			}
		}
	case SchemaKindString:
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) {
			return errors.Errorf("schema %q requires a UTF-8 string", schema.ID)
		}
		if schema.MaxLength > 0 && utf8.RuneCountInString(text) > schema.MaxLength {
			return errors.Errorf("schema %q string exceeds length %d", schema.ID, schema.MaxLength)
		}
	case SchemaKindBoolean:
		if _, ok := value.(bool); !ok {
			return errors.Errorf("schema %q requires a boolean", schema.ID)
		}
	case SchemaKindInteger:
		number, ok := value.(json.Number)
		if !ok {
			return errors.Errorf("schema %q requires an integer", schema.ID)
		}
		integer, err := number.Int64()
		if err != nil {
			return errors.Errorf("schema %q requires an integer", schema.ID)
		}
		if schema.Minimum != nil && integer < *schema.Minimum {
			return errors.Errorf("schema %q integer is below minimum", schema.ID)
		}
		if schema.Maximum != nil && integer > *schema.Maximum {
			return errors.Errorf("schema %q integer is above maximum", schema.ID)
		}
	case SchemaKindBytes:
		text, ok := value.(string)
		if !ok {
			return errors.Errorf("schema %q requires base64 bytes", schema.ID)
		}
		decoded, err := base64.RawURLEncoding.DecodeString(text)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(text)
		}
		if err != nil {
			return errors.Errorf("schema %q requires valid base64 bytes", schema.ID)
		}
		if schema.MaxLength > 0 && len(decoded) > schema.MaxLength {
			return errors.Errorf("schema %q bytes exceed length %d", schema.ID, schema.MaxLength)
		}
	case SchemaKindArray:
		items, ok := value.([]any)
		if !ok {
			return errors.Errorf("schema %q requires an array", schema.ID)
		}
		if schema.MaxItems > 0 && len(items) > schema.MaxItems {
			return errors.Errorf("schema %q array exceeds item count %d", schema.ID, schema.MaxItems)
		}
		itemSchema, ok := schemas[schema.Items]
		if !ok {
			return errors.Errorf("schema %q references unknown item schema %q", schema.ID, schema.Items)
		}
		for index, item := range items {
			if err := validateJSONValue(schemas, itemSchema, item, stack, publicOnly); err != nil {
				return errors.Wrapf(err, "item %d", index)
			}
		}
	default:
		return errors.Errorf("schema %q has unsupported kind %q", schema.ID, schema.Kind)
	}
	return nil
}
