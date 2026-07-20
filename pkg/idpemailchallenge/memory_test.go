package idpemailchallenge_test

import (
	"context"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreConsumesCorrectCodeOnce(t *testing.T) {
	now := time.Now().UTC()
	s := idpemailchallenge.NewMemoryStore()
	c := testChallenge(now)
	require.NoError(t, s.CreateEmailChallenge(context.Background(), c))
	e, err := s.VerifyEmailChallenge(context.Background(), c.ID, c.CodeHash, testBindings(), now)
	require.NoError(t, err)
	assert.Equal(t, c.Email, e.Address)
	_, err = s.VerifyEmailChallenge(context.Background(), c.ID, c.CodeHash, testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrAlreadyTerminal)
}

func TestMemoryStoreAllowsExactlyOneConcurrentCorrectVerification(t *testing.T) {
	now := time.Now().UTC()
	s := idpemailchallenge.NewMemoryStore()
	c := testChallenge(now)
	require.NoError(t, s.CreateEmailChallenge(context.Background(), c))
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := s.VerifyEmailChallenge(context.Background(), c.ID, c.CodeHash, testBindings(), now)
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		assert.ErrorIs(t, err, idpemailchallenge.ErrAlreadyTerminal)
	}
	assert.Equal(t, 1, successes)
}
func TestMemoryStoreRejectsWrongCodeAndBindings(t *testing.T) {
	now := time.Now().UTC()
	s := idpemailchallenge.NewMemoryStore()
	c := testChallenge(now)
	require.NoError(t, s.CreateEmailChallenge(context.Background(), c))
	_, err := s.VerifyEmailChallenge(context.Background(), c.ID, []byte("badbadbadbadbadbadbadbadbadbadbadb"), testBindings(), now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrConflict)
	_, err = s.VerifyEmailChallenge(context.Background(), c.ID, c.CodeHash, idpemailchallenge.VerificationBindings{}, now)
	assert.ErrorIs(t, err, idpemailchallenge.ErrBinding)
}
func testChallenge(now time.Time) idpemailchallenge.PendingChallenge {
	return idpemailchallenge.PendingChallenge{Version: 1, ID: "c", CodeHash: []byte("0123456789abcdef0123456789abcdef"), Email: "a@example.test", Template: "signup", WorkflowID: "signup", ResumeHandlerID: "verified", ProgramFingerprint: "p", ClientID: "app", ClientGeneration: "g", BrowserBindingHash: []byte("b"), CreatedAt: now, ExpiresAt: now.Add(time.Hour), MaximumAttempts: 3, MaximumResends: 2, Status: idpemailchallenge.StatusPending}
}
func testBindings() idpemailchallenge.VerificationBindings {
	return idpemailchallenge.VerificationBindings{WorkflowID: "signup", ResumeHandlerID: "verified", ProgramFingerprint: "p", ClientID: "app", ClientGeneration: "g", BrowserBindingHash: []byte("b")}
}
