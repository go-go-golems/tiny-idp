package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/manuel/tinyidp/internal/audit"
	"github.com/manuel/tinyidp/internal/authn"
	"github.com/manuel/tinyidp/internal/passwordhash"
	"github.com/manuel/tinyidp/internal/user"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type Service struct {
	Store     idpstore.Store
	Passwords *authn.PasswordService
	Clock     func() time.Time
	Audit     audit.Sink
}

type Options struct {
	Hasher passwordhash.Hasher
	Clock  func() time.Time
	Audit  audit.Sink
}

func NewService(store idpstore.Store, opts Options) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	sink := opts.Audit
	if sink == nil {
		sink = audit.NoopSink{}
	}
	passwords, err := authn.NewPasswordService(store, authn.Options{Hasher: opts.Hasher, Clock: clock, Audit: sink})
	if err != nil {
		return nil, err
	}
	return &Service{Store: store, Passwords: passwords, Clock: clock, Audit: sink}, nil
}

type CreateUserRequest struct {
	Login             string
	Password          []byte
	ID                string
	Sub               string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	Groups            []string
	Roles             []string
	Tenant            string
	Locale            string
	MustChangeAtLogin bool
}

func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (idpstore.User, error) {
	login := user.Normalize(req.Login)
	if login == "" {
		return idpstore.User{}, fmt.Errorf("login is required")
	}
	if len(req.Password) == 0 {
		return idpstore.User{}, fmt.Errorf("password is required")
	}
	if _, err := s.Store.GetUserByLogin(ctx, login); err == nil {
		return idpstore.User{}, idpstore.ErrDuplicate
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, err
	}
	now := s.Clock().UTC()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = newID("user")
	}
	if _, err := s.Store.GetUser(ctx, id); err == nil {
		return idpstore.User{}, idpstore.ErrDuplicate
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.User{}, err
	}
	sub := strings.TrimSpace(req.Sub)
	if sub == "" {
		sub = id
	}
	u := idpstore.User{ID: id, Sub: sub, Email: strings.TrimSpace(req.Email), EmailVerified: req.EmailVerified, Name: strings.TrimSpace(req.Name), PreferredUsername: firstNonEmpty(strings.TrimSpace(req.PreferredUsername), login), Groups: cleanList(req.Groups), Roles: cleanList(req.Roles), Tenant: strings.TrimSpace(req.Tenant), Locale: strings.TrimSpace(req.Locale), CreatedAt: now, UpdatedAt: now}
	if err := u.Validate(); err != nil {
		return idpstore.User{}, err
	}
	cred, err := s.Passwords.HashCredential(u.ID, login, req.Password, now)
	if err != nil {
		return idpstore.User{}, err
	}
	cred.MustChangeAtLogin = req.MustChangeAtLogin
	if err := s.Store.PutUser(ctx, login, u); err != nil {
		return idpstore.User{}, err
	}
	if err := s.Store.PutPasswordCredential(ctx, cred); err != nil {
		return idpstore.User{}, err
	}
	_ = s.Audit.Emit(ctx, audit.Event{Time: now, Name: "admin.user.created", Subject: u.Sub, Result: "accepted"})
	return u, nil
}

type SetPasswordRequest struct {
	Login             string
	Password          []byte
	MustChangeAtLogin bool
}

func (s *Service) SetPassword(ctx context.Context, req SetPasswordRequest) error {
	login := user.Normalize(req.Login)
	if login == "" {
		return fmt.Errorf("login is required")
	}
	if len(req.Password) == 0 {
		return fmt.Errorf("password is required")
	}
	u, err := s.Store.GetUserByLogin(ctx, login)
	if err != nil {
		return err
	}
	now := s.Clock().UTC()
	cred, err := s.Passwords.HashCredential(u.ID, login, req.Password, now)
	if err != nil {
		return err
	}
	cred.MustChangeAtLogin = req.MustChangeAtLogin
	if err := s.Store.PutPasswordCredential(ctx, cred); err != nil {
		return err
	}
	_ = s.Store.ResetAccountSecurityState(ctx, u.ID, now)
	_ = s.Audit.Emit(ctx, audit.Event{Time: now, Name: "admin.user.password_changed", Subject: u.Sub, Result: "accepted"})
	return nil
}

func (s *Service) SetUserDisabled(ctx context.Context, login string, disabled bool) (idpstore.User, error) {
	login = user.Normalize(login)
	if login == "" {
		return idpstore.User{}, fmt.Errorf("login is required")
	}
	u, err := s.Store.GetUserByLogin(ctx, login)
	if err != nil {
		return idpstore.User{}, err
	}
	u.Disabled = disabled
	u.UpdatedAt = s.Clock().UTC()
	if err := s.Store.PutUser(ctx, login, u); err != nil {
		return idpstore.User{}, err
	}
	name := "admin.user.enabled"
	if disabled {
		name = "admin.user.disabled"
	}
	_ = s.Audit.Emit(ctx, audit.Event{Time: u.UpdatedAt, Name: name, Subject: u.Sub, Result: "accepted"})
	return u, nil
}

func (s *Service) GetUserByLogin(ctx context.Context, login string) (idpstore.User, error) {
	return s.Store.GetUserByLogin(ctx, user.Normalize(login))
}

func newID(prefix string) string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(b)
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
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
