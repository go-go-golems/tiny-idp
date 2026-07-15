// Package externalmessagedesk contains the public-API bootstrap pieces shared
// by the standalone tiny-idp container and its Docker initialization job.
package externalmessagedesk

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type SeedAccount struct {
	ID       string
	Subject  string
	Login    string
	Password string
	Name     string
	Email    string
}

type SeedManifest struct {
	ClientID               string
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	Accounts               []SeedAccount
}

func (m SeedManifest) Bootstrap(ctx context.Context, store idpstore.Store, accounts *idpaccounts.Service, mode embeddedidp.Mode) error {
	if store == nil || accounts == nil {
		return errors.New("identity store and account service are required")
	}
	if strings.TrimSpace(m.ClientID) == "" || len(m.RedirectURIs) == 0 || len(m.PostLogoutRedirectURIs) == 0 {
		return errors.New("seed client ID, redirect URIs, and post-logout redirect URIs are required")
	}
	if _, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{Mode: mode, Clients: []embeddedidp.ClientSpec{embeddedidp.BrowserClient(m.ClientID, m.RedirectURIs, m.PostLogoutRedirectURIs, []string{"openid", "profile"})}}); err != nil {
		return errors.Wrap(err, "bootstrap seeded browser client")
	}
	seen := make(map[string]struct{}, len(m.Accounts))
	for _, seed := range m.Accounts {
		login := idpaccounts.NormalizeLogin(seed.Login)
		if login == "" || strings.TrimSpace(seed.ID) == "" || strings.TrimSpace(seed.Subject) == "" || seed.Password == "" {
			return errors.New("each seed account requires ID, subject, login, and password")
		}
		if _, duplicate := seen[login]; duplicate {
			return errors.Errorf("duplicate seed login %q", login)
		}
		seen[login] = struct{}{}
		if err := reconcileSeedAccount(ctx, store, accounts, m.ClientID, seed); err != nil {
			return err
		}
	}
	return nil
}

func reconcileSeedAccount(ctx context.Context, store idpstore.Store, accounts *idpaccounts.Service, clientID string, seed SeedAccount) error {
	_, err := accounts.Create(ctx, idpaccounts.CreateRequest{ID: seed.ID, Subject: seed.Subject, Login: seed.Login, Password: []byte(seed.Password), Email: seed.Email, EmailVerified: true, Name: seed.Name, PreferredUsername: seed.Login})
	if err == nil {
		return nil
	}
	if !errors.Is(err, idpstore.ErrDuplicate) {
		return errors.Wrap(err, "create seed account")
	}
	existing, err := store.GetUserByLogin(ctx, seed.Login)
	if err != nil {
		return errors.Wrap(err, "load duplicate seed account")
	}
	if existing.ID != seed.ID || existing.Sub != seed.Subject || existing.Email != seed.Email || existing.Name != seed.Name {
		return errors.Errorf("seed account %q conflicts with persisted identity state", seed.Login)
	}
	if _, err := accounts.AuthenticatePassword(ctx, seed.Login, seed.Password, idp.LoginMetadata{ClientID: clientID}); err != nil {
		return errors.Wrapf(err, "seed account %q password conflicts with persisted identity state", seed.Login)
	}
	return nil
}
