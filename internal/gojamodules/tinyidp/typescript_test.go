package tinyidp

import (
	"strings"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/stretchr/testify/require"
)

func TestTypeScriptDeclarationsCoverImplementedV1Surface(t *testing.T) {
	declarations, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{(&module{}).TypeScriptModule()}})
	require.NoError(t, err)
	for _, fragment := range []string{
		`declare module "tinyidp"`,
		`export type OutcomeKind = "continue" | "present" | "challenge" | "commit" | "complete" | "deny" | "skip" | "error";`,
		`export type EffectKind = "read" | "createLocalIdentity" | "attachPasswordCredential" | "consumeInvitation" | "establishBrowserSession" | "establishVirtualIdentity" | "sendEmailChallenge";`,
		"interface InvocationContext",
		"interface PresentationBuilders",
		"form(spec: PresentationSpec): Outcome",
		"interface ChallengeBuilders",
		"emailCode(spec: EmailChallengeSpec): Outcome",
		"interface FieldBuilders",
		"interface ActionBuilders",
		"interface SecretHandle",
		"interface CommitBuilders",
		"signup(spec: SignupCommitSpec): Outcome",
		"interface WorkflowSpec",
		"interface ProviderSpec",
		"Promise<Outcome>",
		"interface ProgramBuilder",
		"interface ProgramTestSpec",
		"readonly result: ResultBuilders",
	} {
		require.True(t, strings.Contains(declarations, fragment), "declarations missing %q:\n%s", fragment, declarations)
	}
}
