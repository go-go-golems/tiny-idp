package admin

import (
	"context"
	"fmt"

	"github.com/go-go-golems/tiny-idp/internal/user"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

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
	err = s.auditCommitted(ctx, idp.Event{Time: u.UpdatedAt, Name: name, Subject: u.Sub, Result: "accepted"})
	return u, err
}

func (s *Service) GetUserByLogin(ctx context.Context, login string) (idpstore.User, error) {
	return s.Store.GetUserByLogin(ctx, user.Normalize(login))
}
