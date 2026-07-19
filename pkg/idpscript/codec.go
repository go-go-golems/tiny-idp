package idpscript

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func validateInput(program idpprogram.Program, spec idpprogram.LambdaSpec, input json.RawMessage) (any, error) {
	schema, ok := program.Schemas[spec.InputSchema]
	if !ok {
		return nil, errors.Errorf("input schema %q is not registered", spec.InputSchema)
	}
	if len(input) == 0 || len(input) > schema.MaxBytes {
		return nil, errors.Errorf("lambda input size %d outside 1..%d", len(input), schema.MaxBytes)
	}
	var plain any
	if err := decodeSingleJSON(input, &plain); err != nil {
		return nil, errors.Wrap(err, "decode lambda input")
	}
	if err := validateSchemaValue(program.Schemas, schema, plain, map[string]bool{}); err != nil {
		return nil, errors.Wrap(err, "validate lambda input")
	}
	return plain, nil
}

func decodeOutcome(program idpprogram.Program, spec idpprogram.LambdaSpec, encoded []byte) (idpprogram.Outcome, error) {
	if len(encoded) == 0 || len(encoded) > spec.Budget.MaxOutputBytes {
		return idpprogram.Outcome{}, errors.Errorf("lambda output size %d outside 1..%d", len(encoded), spec.Budget.MaxOutputBytes)
	}
	var outcome idpprogram.Outcome
	if err := decodeSingleJSON(encoded, &outcome); err != nil {
		return idpprogram.Outcome{}, errors.Wrap(err, "decode lambda outcome")
	}
	if err := idpprogram.ValidateOutcome(spec, outcome); err != nil {
		return idpprogram.Outcome{}, errors.Wrap(err, "validate lambda outcome")
	}
	if len(outcome.Value) != 0 {
		schema, ok := program.Schemas[spec.OutputSchema]
		if !ok {
			return idpprogram.Outcome{}, errors.Errorf("output schema %q is not registered", spec.OutputSchema)
		}
		if len(outcome.Value) > schema.MaxBytes {
			return idpprogram.Outcome{}, errors.Errorf("lambda value exceeds schema byte limit %d", schema.MaxBytes)
		}
		var plain any
		if err := decodeSingleJSON(outcome.Value, &plain); err != nil {
			return idpprogram.Outcome{}, errors.Wrap(err, "decode lambda outcome value")
		}
		if err := validateSchemaValue(program.Schemas, schema, plain, map[string]bool{}); err != nil {
			return idpprogram.Outcome{}, errors.Wrap(err, "validate lambda outcome value")
		}
	}
	return outcome, nil
}

func decodeSingleJSON(encoded []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func validateSchemaValue(schemas map[string]idpprogram.Schema, schema idpprogram.Schema, value any, stack map[string]bool) error {
	if stack[schema.ID] {
		return errors.Errorf("recursive schema %q", schema.ID)
	}
	stack[schema.ID] = true
	defer delete(stack, schema.ID)

	switch schema.Kind {
	case idpprogram.SchemaKindObject:
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
			fieldSchema, ok := schemas[field.Ref]
			if !ok {
				return errors.Errorf("schema %q field %q references unknown schema %q", schema.ID, name, field.Ref)
			}
			if err := validateSchemaValue(schemas, fieldSchema, fieldValue, stack); err != nil {
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
	case idpprogram.SchemaKindString:
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) {
			return errors.Errorf("schema %q requires a UTF-8 string", schema.ID)
		}
		if schema.MaxLength > 0 && utf8.RuneCountInString(text) > schema.MaxLength {
			return errors.Errorf("schema %q string exceeds length %d", schema.ID, schema.MaxLength)
		}
	case idpprogram.SchemaKindBoolean:
		if _, ok := value.(bool); !ok {
			return errors.Errorf("schema %q requires a boolean", schema.ID)
		}
	case idpprogram.SchemaKindInteger:
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
	case idpprogram.SchemaKindBytes:
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
	default:
		return errors.Errorf("schema %q has unsupported kind %q", schema.ID, schema.Kind)
	}
	return nil
}
