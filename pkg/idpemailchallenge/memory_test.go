package idpemailchallenge_test

import (
	"context"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMemoryStoreConsumesCorrectCodeOnce(t *testing.T) {
	now := time.Now().UTC()
	s := idpemailchallenge.NewMemoryStore()
	c := testChallenge(now)
	require.NoError(t, s.Create(context.Background(), c))
	e, err := s.Verify(context.Background(), c.ID, c.CodeHash, testBindings(), now)
	require.NoError(t, err)
	assert.Equal(t, c.Email, e.Address)
	_, err = s.Verify(context.Background(), c.ID, c.CodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAlreadyTerminal)
}
func TestMemoryStoreRejectsWrongCodeAndBindings(t *testing.T) {
	now := time.Now().UTC()
	s := idpemailchallenge.NewMemoryStore()
	c := testChallenge(now)
	require.NoError(t, s.Create(context.Background(), c))
	_, err := s.Verify(context.Background(), c.ID, []byte("badbadbadbadbadbadbadbadbadbadbadb"), testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrConflict)
	_, err = s.Verify(context.Background(), c.ID, c.CodeHash, idpemailchallenge.VerificationBindings{}, now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrBinding)
}
func testChallenge(now time.Time) idpemailchallenge.PendingChallenge {
	return idpemailchallenge.PendingChallenge{Version: 1, ID: "c", CodeHash: []byte("0123456789abcdef0123456789abcdef"), Email: "a@example.test", Template: "signup", WorkflowID: "signup", ResumeHandlerID: "verified", ProgramFingerprint: "p", ClientID: "app", ClientGeneration: "g", BrowserBindingHash: []byte("b"), CreatedAt: now, ExpiresAt: now.Add(time.Hour), MaximumAttempts: 3, MaximumResends: 2, Status: idpemailchallenge.StatusPending}
}
func testBindings() idpemailchallenge.VerificationBindings {
	return idpemailchallenge.VerificationBindings{WorkflowID: "signup", ResumeHandlerID: "verified", ProgramFingerprint: "p", ClientID: "app", ClientGeneration: "g", BrowserBindingHash: []byte("b")}
}
