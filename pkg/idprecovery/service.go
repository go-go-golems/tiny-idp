// Package idprecovery contains the native password-recovery effect boundary.
// It accepts only an already verified, bound email challenge and delegates
// password hashing and credential replacement to idpaccounts.Service.
package idprecovery

import (
	"context"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/idpemailchallenge"
)

const EmailTemplate = "password-recovery"

type Service struct {
	challenges *idpemailchallenge.Service
	accounts   *idpaccounts.Service
}

func NewService(challenges *idpemailchallenge.Service, accounts *idpaccounts.Service) (*Service, error) {
	if challenges == nil || accounts == nil {
		return nil, errors.New("email challenges and accounts are required")
	}
	return &Service{challenges: challenges, accounts: accounts}, nil
}

// ResetPassword rehydrates native evidence with its original bindings before
// replacing the credential. Browser/script callers cannot provide an address,
// account ID, password hash, or synthetic verified marker.
func (s *Service) ResetPassword(ctx context.Context, ref idpemailchallenge.Reference, bindings idpemailchallenge.VerificationBindings, password []byte) error {
	if s == nil {
		return errors.New("password recovery service is unavailable")
	}
	evidence, err := s.challenges.Evidence(ctx, ref, bindings)
	if err != nil {
		return errors.Wrap(err, "load verified recovery evidence")
	}
	if evidence.Template != EmailTemplate {
		return errors.New("verified email challenge is not a password recovery challenge")
	}
	// Consume before the credential mutation. This deliberately fails closed:
	// an unavailable credential store requires a new recovery request instead
	// of leaving a reusable verified reset capability behind.
	if _, err := s.challenges.ConsumeEvidence(ctx, ref, bindings); err != nil {
		return errors.Wrap(err, "consume verified recovery evidence")
	}
	if err := s.accounts.SetPassword(ctx, idpaccounts.SetPasswordRequest{Login: evidence.Address, Password: password}); err != nil {
		return errors.Wrap(err, "replace recovered password")
	}
	return nil
}
