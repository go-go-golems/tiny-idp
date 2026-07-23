package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

type SecretResult struct {
	Generated bool   `json:"generated"`
	Secret    string `json:"secret,omitempty"`
}

type CreateClientRequest struct {
	ID                     string
	Public                 bool
	Secret                 string
	GenerateSecret         bool
	RedirectURIs           []string
	PostLogoutRedirectURIs []string
	AllowedScopes          []string
	AllowedGrantTypes      []string
	AllowedAudiences       []string
	CanIntrospect          bool
	RequirePKCE            bool
	AccessTokenTTL         time.Duration
	IDTokenTTL             time.Duration
	RefreshTokenTTL        time.Duration
	Disabled               bool
}

func (s *Service) CreateClient(ctx context.Context, req CreateClientRequest) (idpstore.Client, SecretResult, error) {
	now := s.Clock().UTC()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return idpstore.Client{}, SecretResult{}, fmt.Errorf("client id is required")
	}
	if _, err := s.Store.GetClient(ctx, id); err == nil {
		return idpstore.Client{}, SecretResult{}, idpstore.ErrDuplicate
	} else if err != idpstore.ErrNotFound {
		return idpstore.Client{}, SecretResult{}, err
	}
	secret := strings.TrimSpace(req.Secret)
	generated := false
	if !req.Public && secret == "" && req.GenerateSecret {
		var err error
		secret, err = randomSecret(32)
		if err != nil {
			return idpstore.Client{}, SecretResult{}, err
		}
		generated = true
	}
	c := idpstore.Client{ID: id, Public: req.Public, RedirectURIs: cleanList(req.RedirectURIs), PostLogoutRedirectURIs: cleanList(req.PostLogoutRedirectURIs), AllowedScopes: cleanList(req.AllowedScopes), AllowedGrantTypes: cleanList(req.AllowedGrantTypes), AllowedAudiences: cleanList(req.AllowedAudiences), CanIntrospect: req.CanIntrospect, RequirePKCE: req.RequirePKCE || req.Public, AccessTokenTTL: defaultDuration(req.AccessTokenTTL, time.Hour), IDTokenTTL: defaultDuration(req.IDTokenTTL, time.Hour), RefreshTokenTTL: defaultDuration(req.RefreshTokenTTL, 30*24*time.Hour), CreatedAt: now, UpdatedAt: now, Disabled: req.Disabled}
	if !c.Public {
		if secret == "" {
			return idpstore.Client{}, SecretResult{}, fmt.Errorf("client secret is required for confidential clients")
		}
		if len([]byte(secret)) > 72 {
			return idpstore.Client{}, SecretResult{}, fmt.Errorf("client secret must not exceed 72 bytes")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			return idpstore.Client{}, SecretResult{}, err
		}
		c.SecretHash = hash
	}
	if err := c.Validate(idpstore.ProductionMode); err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	if err := s.Store.PutClient(ctx, c); err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	result := SecretResult{Generated: generated}
	if generated {
		result.Secret = secret
	}
	err := s.auditCommitted(ctx, idp.Event{Time: now, Name: "admin.client.created", ClientID: c.ID, Result: "accepted"})
	return c, result, err
}

func (s *Service) ListClients(ctx context.Context) ([]idpstore.Client, error) {
	return s.Store.ListClients(ctx)
}

func (s *Service) GetClient(ctx context.Context, id string) (idpstore.Client, error) {
	return s.Store.GetClient(ctx, strings.TrimSpace(id))
}

func (s *Service) SetClientDisabled(ctx context.Context, id string, disabled bool) (idpstore.Client, error) {
	c, err := s.GetClient(ctx, id)
	if err != nil {
		return idpstore.Client{}, err
	}
	c.Disabled = disabled
	c.UpdatedAt = s.Clock().UTC()
	if err := s.Store.PutClient(ctx, c); err != nil {
		return idpstore.Client{}, err
	}
	name := "admin.client.enabled"
	if disabled {
		name = "admin.client.disabled"
	}
	err = s.auditCommitted(ctx, idp.Event{Time: c.UpdatedAt, Name: name, ClientID: c.ID, Result: "accepted"})
	return c, err
}

func (s *Service) RotateClientSecret(ctx context.Context, id string) (idpstore.Client, SecretResult, error) {
	c, err := s.GetClient(ctx, id)
	if err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	if c.Public {
		return idpstore.Client{}, SecretResult{}, fmt.Errorf("public clients do not have secrets")
	}
	secret, err := randomSecret(32)
	if err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	c.SecretHash = hash
	c.UpdatedAt = s.Clock().UTC()
	if err := c.Validate(idpstore.ProductionMode); err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	if err := s.Store.PutClient(ctx, c); err != nil {
		return idpstore.Client{}, SecretResult{}, err
	}
	err = s.auditCommitted(ctx, idp.Event{Time: c.UpdatedAt, Name: "admin.client.secret_rotated", ClientID: c.ID, Result: "accepted"})
	return c, SecretResult{Generated: true, Secret: secret}, err
}

func randomSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
