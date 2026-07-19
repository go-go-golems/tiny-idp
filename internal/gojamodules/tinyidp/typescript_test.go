package tinyidp

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/stretchr/testify/require"
)

func TestTypeScriptDeclarationsCoverPhase0Surface(t *testing.T) {
	declarations, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{(&module{}).TypeScriptModule()}})
	require.NoError(t, err)
	for _, fragment := range []string{
		`declare module "tinyidp"`,
		"interface InvocationContext",
		"interface PresentationBuilders",
		"interface FieldBuilders",
		"interface ActionBuilders",
		"interface SecretHandle",
		"interface CommitBuilders",
		"Promise<Outcome>",
		"interface ProgramBuilder",
		"interface ProgramTestSpec",
		"readonly result: ResultBuilders",
	} {
		require.True(t, strings.Contains(declarations, fragment), "declarations missing %q:\n%s", fragment, declarations)
	}
}
