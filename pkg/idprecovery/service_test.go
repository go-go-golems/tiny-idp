package idprecovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
	"github.com/go-go-golems/tiny-idp/pkg/idprecovery"
)

type captureMailer struct {
	requests []idpemailchallenge.MailRequest
}

func (m *captureMailer) SendEmailChallenge(_ context.Context, request idpemailchallenge.MailRequest) error {
	m.requests = append(m.requests, request)
	return nil
}

func TestResetPasswordRequiresVerifiedRecoveryTemplate(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{PasswordPolicy: idp.DevelopmentPasswordAcceptancePolicy()})
	require.NoError(t, err)
	_, err = accounts.Create(ctx, idpaccounts.CreateRequest{Login: "ada@example.test", Email: "ada@example.test", Password: []byte("original password phrase")})
	require.NoError(t, err)
	mailer := &captureMailer{}
	challenges, err := idpemailchallenge.NewService(idpemailchallenge.NewMemoryStore(), mailer, []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	recovery, err := idprecovery.NewService(challenges, accounts)
	require.NoError(t, err)
	bindings := idpemailchallenge.VerificationBindings{WorkflowID: "recovery", ResumeHandlerID: "reset", ProgramFingerprint: "native-recovery-v1", ClientID: "spa", ClientGeneration: "client-v1", BrowserBindingHash: []byte("browser")}
	ref, err := challenges.CreateAndSend(ctx, idpemailchallenge.CreateRequest{ID: "recovery-code", Email: "ada@example.test", Template: idprecovery.EmailTemplate, Bindings: bindings, ExpiresAt: time.Now().Add(time.Hour), MaximumAttempts: 3, MaximumResends: 1})
	require.NoError(t, err)
	_, err = challenges.Verify(ctx, ref, mailer.requests[0].Code, bindings)
	require.NoError(t, err)
	require.NoError(t, recovery.ResetPassword(ctx, ref, bindings, []byte("replacement password phrase")))
	_, err = accounts.AuthenticatePassword(ctx, "ada@example.test", "replacement password phrase", idp.LoginMetadata{})
	require.NoError(t, err)
}
