package idpemailchallenge_test

import (
	"context"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type mailer struct {
	requests []idpemailchallenge.MailRequest
	err      error
}

func (m *mailer) SendEmailChallenge(_ context.Context, r idpemailchallenge.MailRequest) error {
	m.requests = append(m.requests, r)
	return m.err
}
func TestServiceCreatesMailAndNativeEvidence(t *testing.T) {
	now := time.Now().UTC()
	m := &mailer{}
	s, err := idpemailchallenge.NewService(idpemailchallenge.NewMemoryStore(), m, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	b := testBindings()
	ref, err := s.CreateAndSend(context.Background(), idpemailchallenge.CreateRequest{ID: "challenge", Email: "ada@example.test", Template: "signup", Bindings: b, ExpiresAt: now.Add(time.Hour), MaximumAttempts: 3, MaximumResends: 2})
	require.NoError(t, err)
	require.Len(t, m.requests, 1)
	assert.Equal(t, ref, m.requests[0].Challenge)
	e, err := s.Verify(context.Background(), ref, m.requests[0].Code, b)
	require.NoError(t, err)
	assert.Equal(t, "ada@example.test", e.Address)
	rehydrated, err := s.Evidence(context.Background(), ref, b)
	require.NoError(t, err)
	assert.Equal(t, e, rehydrated)
	_, err = s.Evidence(context.Background(), ref, idpemailchallenge.VerificationBindings{WorkflowID: "other"})
	assert.ErrorIs(t, err, idpemailchallenge.ErrBinding)
}
