package idpemailchallenge_test

import (
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPendingChallengeContractsBindEvidenceToItsWorkflow(t *testing.T) {
	now := time.Date(2026, time.July, 19, 20, 0, 0, 0, time.UTC)
	challenge := idpemailchallenge.PendingChallenge{Version: 1, ID: "challenge-1", CodeHash: []byte("0123456789abcdef0123456789abcdef"), Email: "ada@example.test", Template: "signup-code", WorkflowID: "signup", ResumeHandlerID: "emailVerified", ProgramFingerprint: "fingerprint", ClientID: "message-app", ClientGeneration: "generation", BrowserBindingHash: []byte("browser"), CreatedAt: now, ExpiresAt: now.Add(time.Minute), MaximumAttempts: 5, MaximumResends: 2, Status: idpemailchallenge.StatusPending}
	require.NoError(t, challenge.ValidateForCreate())
	assert.Equal(t, idpemailchallenge.Reference{ID: "challenge-1", Version: 1}, challenge.Reference())
	assert.NoError(t, challenge.VerifyBindings(idpemailchallenge.VerificationBindings{WorkflowID: "signup", ResumeHandlerID: "emailVerified", ProgramFingerprint: "fingerprint", ClientID: "message-app", ClientGeneration: "generation", BrowserBindingHash: []byte("browser")}))
	assert.ErrorIs(t, challenge.VerifyBindings(idpemailchallenge.VerificationBindings{WorkflowID: "signup", ResumeHandlerID: "emailVerified", ProgramFingerprint: "old", ClientID: "message-app", ClientGeneration: "generation", BrowserBindingHash: []byte("browser")}), idpemailchallenge.ErrBinding)
}
