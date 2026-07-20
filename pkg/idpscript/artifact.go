package idpscript

import (
	"encoding/json"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

// Artifact is immutable compiled source plus its VM-independent contract.
// *goja.Program is safe to execute in multiple independent runtimes.
type Artifact struct {
	source       string
	compiled     *goja.Program
	programJSON  []byte
	fingerprints idpprogram.Fingerprints
}

func newArtifact(source string, compiled *goja.Program, program idpprogram.Program, fingerprints idpprogram.Fingerprints) *Artifact {
	programJSON, err := idpprogram.CanonicalJSON(program)
	if err != nil {
		panic(errors.Wrap(err, "canonicalize validated artifact"))
	}
	return &Artifact{
		source:       source,
		compiled:     compiled,
		programJSON:  append([]byte(nil), programJSON...),
		fingerprints: fingerprints,
	}
}

// Source returns the immutable source text used for this artifact.
func (a *Artifact) Source() string {
	if a == nil {
		return ""
	}
	return a.source
}

// Program returns a deep copy of the materialized contract.
func (a *Artifact) Program() idpprogram.Program {
	if a == nil {
		return idpprogram.Program{}
	}
	var ret idpprogram.Program
	if err := json.Unmarshal(a.programJSON, &ret); err != nil {
		panic(errors.Wrap(err, "decode validated artifact"))
	}
	return ret
}

// Fingerprints returns the immutable activation identities.
func (a *Artifact) Fingerprints() idpprogram.Fingerprints {
	if a == nil {
		return idpprogram.Fingerprints{}
	}
	return a.fingerprints
}
