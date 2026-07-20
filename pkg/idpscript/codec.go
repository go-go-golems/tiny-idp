package idpscript

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

func validateInput(program idpprogram.Program, spec idpprogram.LambdaSpec, input json.RawMessage) error {
	if err := idpprogram.ValidateJSON(program.Schemas, spec.InputSchema, input); err != nil {
		return errors.Wrap(err, "validate lambda input")
	}
	return nil
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
		if err := idpprogram.ValidateJSON(program.Schemas, spec.OutputSchema, outcome.Value); err != nil {
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
