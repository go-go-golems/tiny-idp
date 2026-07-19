// Package idpscript compiles and loads isolated Tiny-IDP JavaScript programs.
package idpscript

import (
	"context"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const (
	defaultSourceName = "tinyidp-policy.js"
	defaultMaxSource  = 64 << 10
)

// CompileOptions bounds compilation and supplies the host-owned schema catalog.
type CompileOptions struct {
	SourceName     string
	MaxSourceBytes int
	Timeout        time.Duration
	Schemas        map[string]idpprogram.Schema
}

// DefaultCompileOptions returns production-safe Phase 0 compiler defaults.
func DefaultCompileOptions() CompileOptions {
	return CompileOptions{
		SourceName:     defaultSourceName,
		MaxSourceBytes: defaultMaxSource,
		Timeout:        250 * time.Millisecond,
	}
}

// Compile compiles and materializes one source in an isolated owned runtime.
func Compile(ctx context.Context, source string, options CompileOptions) (*Artifact, error) {
	if ctx == nil {
		return nil, errors.New("compile context is required")
	}
	options = normalizeCompileOptions(options)
	if len(source) == 0 || len(source) > options.MaxSourceBytes {
		return nil, errors.Errorf("scripting source size %d outside 1..%d", len(source), options.MaxSourceBytes)
	}

	compiled, err := goja.Compile(options.SourceName, source, false)
	if err != nil {
		return nil, errors.Wrap(err, "compile Tiny-IDP JavaScript")
	}
	factory, err := NewRuntimeFactory(options.Schemas)
	if err != nil {
		return nil, errors.Wrap(err, "create isolated runtime factory")
	}

	compileCtx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	image, err := loadProgram(compileCtx, factory, compiled, nil)
	if err != nil {
		return nil, errors.Wrap(err, "materialize Tiny-IDP program")
	}
	defer image.Close(context.Background()) //nolint:errcheck // compile result is already copied; close failure cannot make it valid.

	program := image.Program()
	diagnostics := idpprogram.Validate(program)
	if diagnostics.HasErrors() {
		return nil, &ValidationError{Diagnostics: diagnostics}
	}
	fingerprints, err := idpprogram.ComputeFingerprints([]byte(source), program)
	if err != nil {
		return nil, errors.Wrap(err, "fingerprint Tiny-IDP program")
	}
	if image.Fingerprints().Program != fingerprints.Program || image.Fingerprints().CallbackRegistry != fingerprints.CallbackRegistry {
		return nil, errors.New("materialized program fingerprints changed during compilation")
	}

	return newArtifact(source, compiled, program, fingerprints), nil
}

func normalizeCompileOptions(options CompileOptions) CompileOptions {
	defaults := DefaultCompileOptions()
	if options.SourceName == "" {
		options.SourceName = defaults.SourceName
	}
	if options.MaxSourceBytes == 0 {
		options.MaxSourceBytes = defaults.MaxSourceBytes
	}
	if options.Timeout == 0 {
		options.Timeout = defaults.Timeout
	}
	return options
}

// ValidationError preserves stable diagnostics for CLI and activation callers.
type ValidationError struct {
	Diagnostics idpprogram.Diagnostics
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "invalid Tiny-IDP program"
	}
	return errors.Errorf("invalid Tiny-IDP program: %s at %s", e.Diagnostics[0].ID, e.Diagnostics[0].Path).Error()
}
