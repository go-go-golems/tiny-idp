package idpemailchallenge_test

import (
	"encoding/json"
	"github.com/go-go-golems/tiny-idp/pkg/idpcontinuation"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestEvidenceProjectionAndContinuationBindingAreNativeOnly(t *testing.T) {
	c := idpcontinuation.WorkflowContinuation{WorkflowID: "signup", ResumeHandlerID: "verified", ProgramFingerprint: "fp", ClientID: "app", ClientGeneration: "g", BrowserBindingHash: []byte("browser")}
	b := idpemailchallenge.BindingsFromContinuation(c)
	assert.Equal(t, "verified", b.ResumeHandlerID)
	p, err := idpemailchallenge.EvidenceProjection(idpemailchallenge.VerifiedEmailEvidence{Version: 1, ChallengeID: "c", Address: "a@example.test", Method: "email_code", VerifiedAt: time.Now()})
	require.NoError(t, err)
	var v map[string]any
	require.NoError(t, json.Unmarshal(p["email"], &v))
	assert.Equal(t, "a@example.test", v["address"])
	assert.Equal(t, true, v["verified"])
}
