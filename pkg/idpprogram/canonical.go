package idpprogram

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/pkg/errors"
)

// Fingerprints are the stable identities used during compile, activation, and
// cross-worker registry verification.
type Fingerprints struct {
	Source           string `json:"source"`
	Program          string `json:"program"`
	CallbackRegistry string `json:"callbackRegistry"`
	Schemas          string `json:"schemas"`
}

// CanonicalJSON returns deterministic JSON for a program. Program contracts
// intentionally exclude floats, interfaces, functions, and VM-owned values.
func CanonicalJSON(program Program) ([]byte, error) {
	ret, err := json.Marshal(program)
	if err != nil {
		return nil, errors.Wrap(err, "marshal canonical program")
	}
	return ret, nil
}

// ComputeFingerprints hashes source and the deterministic contract registries.
func ComputeFingerprints(source []byte, program Program) (Fingerprints, error) {
	programJSON, err := CanonicalJSON(program)
	if err != nil {
		return Fingerprints{}, err
	}

	callbackIDs := make([]string, 0, len(program.Lambdas))
	for id := range program.Lambdas {
		callbackIDs = append(callbackIDs, id)
	}
	sort.Strings(callbackIDs)
	callbackJSON, err := json.Marshal(callbackIDs)
	if err != nil {
		return Fingerprints{}, errors.Wrap(err, "marshal callback registry")
	}

	schemaJSON, err := json.Marshal(program.Schemas)
	if err != nil {
		return Fingerprints{}, errors.Wrap(err, "marshal schema registry")
	}

	return Fingerprints{
		Source:           hashHex(source),
		Program:          hashHex(programJSON),
		CallbackRegistry: hashHex(callbackJSON),
		Schemas:          hashHex(schemaJSON),
	}, nil
}

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
