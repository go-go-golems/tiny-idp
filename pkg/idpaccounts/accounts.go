package idpaccounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/go-go-golems/tiny-idp/internal/user"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

// CreateRequest describes a local account and its initial password.
type CreateRequest struct {
	Login             string
	Password          []byte
	ID                string
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Groups            []string
	Roles             []string
	Tenant            string
	Locale            string
}

// NormalizeLogin returns the canonical login representation used by account
// creation and password authentication. Embedding applications can use this
// value for non-secret correlation keys such as rate limits; they must not use
// it as proof that an account exists.
func NormalizeLogin(login string) string {
	return user.Normalize(login)
}

// Create atomically persists a user and its initial password credential.
func (s *Service) Create(ctx context.Context, req CreateRequest) (idpstore.User, error) {
	login := NormalizeLogin(req.Login)
	if login == "" {
		return idpstore.User{}, fmt.Errorf("login is required")
	}
	if len(req.Password) == 0 {
		return idpstore.User{}, fmt.Errorf("password is required")
	}
	if _, err := s.store.GetUserByLogin(ctx, login); err == nil {
		return idpstore.User{}, idpstore.ErrDuplicate
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, err
	}
	now := s.clock().UTC()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		generatedID, err := newID("user")
		if err != nil {
			return idpstore.User{}, fmt.Errorf("generate user id: %w", err)
		}
		id = generatedID
	}
	if _, err := s.store.GetUser(ctx, id); err == nil {
		return idpstore.User{}, idpstore.ErrDuplicate
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, err
	}
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = id
	}
	if _, err := s.store.GetUserBySubject(ctx, subject); err == nil {
		return idpstore.User{}, idpstore.ErrDuplicate
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, err
	}
	u := idpstore.User{
		ID: id, Sub: subject, Email: strings.TrimSpace(req.Email), EmailVerified: req.EmailVerified,
		Name: strings.TrimSpace(req.Name), PreferredUsername: firstNonEmpty(strings.TrimSpace(req.PreferredUsername), login),
		Groups: cleanList(req.Groups), Roles: cleanList(req.Roles), Tenant: strings.TrimSpace(req.Tenant),
		Locale: strings.TrimSpace(req.Locale), CreatedAt: now, UpdatedAt: now,
	}
	if err := u.Validate(); err != nil {
		return idpstore.User{}, err
	}
	credential, err := s.hashCredential(ctx, u.ID, login, req.Password, now)
	if err != nil {
		return idpstore.User{}, err
	}
	if err := s.store.CreateUserWithCredential(ctx, login, u, credential); err != nil {
		return idpstore.User{}, err
	}
	err = s.auditCommitted(ctx, idp.Event{Time: now, Name: "identity.account.created", Subject: u.Sub, Result: "accepted"})
	return u, err
}

// SetPasswordRequest identifies an account and its replacement password.
type SetPasswordRequest struct {
	Login    string
	Password []byte
}

// SetPassword atomically replaces a credential and clears account lockout state.
func (s *Service) SetPassword(ctx context.Context, req SetPasswordRequest) error {
	login := user.Normalize(req.Login)
	if login == "" {
		return fmt.Errorf("login is required")
	}
	if len(req.Password) == 0 {
		return fmt.Errorf("password is required")
	}
	u, err := s.store.GetUserByLogin(ctx, login)
	if err != nil {
		return err
	}
	now := s.clock().UTC()
	credential, err := s.hashCredential(ctx, u.ID, login, req.Password, now)
	if err != nil {
		return err
	}
	state := idpstore.AccountSecurityState{UserID: u.ID}
	if err := s.store.ReplacePasswordAndSecurityState(ctx, credential, state); err != nil {
		return err
	}
	return s.auditCommitted(ctx, idp.Event{Time: now, Name: "identity.account.password_changed", Subject: u.Sub, Result: "accepted"})
}

func (s *Service) auditCommitted(ctx context.Context, event idp.Event) error {
	if err := s.audit.Emit(ctx, event); err != nil {
		return fmt.Errorf("%w: %v", idp.ErrAuditDelivery, err)
	}
	return nil
}

func newID(prefix string) (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(b), nil
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
