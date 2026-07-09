package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/manuel/tinyidp/internal/audit"
	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/storage"
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
	RequirePKCE            bool
	AccessTokenTTL         time.Duration
	IDTokenTTL             time.Duration
	RefreshTokenTTL        time.Duration
	Disabled               bool
}

func (s *Service) CreateClient(ctx context.Context, req CreateClientRequest) (domain.Client, SecretResult, error) {
	now := s.Clock().UTC()
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return domain.Client{}, SecretResult{}, fmt.Errorf("client id is required")
	}
	if _, err := s.Store.GetClient(ctx, id); err == nil {
		return domain.Client{}, SecretResult{}, storage.ErrDuplicate
	} else if err != storage.ErrNotFound {
		return domain.Client{}, SecretResult{}, err
	}
	secret := strings.TrimSpace(req.Secret)
	generated := false
	if !req.Public && secret == "" && req.GenerateSecret {
		var err error
		secret, err = randomSecret(32)
		if err != nil {
			return domain.Client{}, SecretResult{}, err
		}
		generated = true
	}
	c := domain.Client{ID: id, Public: req.Public, RedirectURIs: cleanList(req.RedirectURIs), PostLogoutRedirectURIs: cleanList(req.PostLogoutRedirectURIs), AllowedScopes: cleanList(req.AllowedScopes), RequirePKCE: req.RequirePKCE || req.Public, AccessTokenTTL: defaultDuration(req.AccessTokenTTL, time.Hour), IDTokenTTL: defaultDuration(req.IDTokenTTL, time.Hour), RefreshTokenTTL: defaultDuration(req.RefreshTokenTTL, 30*24*time.Hour), CreatedAt: now, UpdatedAt: now, Disabled: req.Disabled}
	if !c.Public {
		if secret == "" {
			return domain.Client{}, SecretResult{}, fmt.Errorf("client secret is required for confidential clients")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
		if err != nil {
			return domain.Client{}, SecretResult{}, err
		}
		c.SecretHash = hash
	}
	if err := c.Validate(domain.ProductionMode); err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	if err := s.Store.PutClient(ctx, c); err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	_ = s.Audit.Emit(ctx, audit.Event{Time: now, Name: "admin.client.created", ClientID: c.ID, Result: "accepted"})
	result := SecretResult{Generated: generated}
	if generated {
		result.Secret = secret
	}
	return c, result, nil
}

func (s *Service) ListClients(ctx context.Context) ([]domain.Client, error) {
	return s.Store.ListClients(ctx)
}

func (s *Service) GetClient(ctx context.Context, id string) (domain.Client, error) {
	return s.Store.GetClient(ctx, strings.TrimSpace(id))
}

func (s *Service) SetClientDisabled(ctx context.Context, id string, disabled bool) (domain.Client, error) {
	c, err := s.GetClient(ctx, id)
	if err != nil {
		return domain.Client{}, err
	}
	c.Disabled = disabled
	c.UpdatedAt = s.Clock().UTC()
	if err := s.Store.PutClient(ctx, c); err != nil {
		return domain.Client{}, err
	}
	name := "admin.client.enabled"
	if disabled {
		name = "admin.client.disabled"
	}
	_ = s.Audit.Emit(ctx, audit.Event{Time: c.UpdatedAt, Name: name, ClientID: c.ID, Result: "accepted"})
	return c, nil
}

func (s *Service) RotateClientSecret(ctx context.Context, id string) (domain.Client, SecretResult, error) {
	c, err := s.GetClient(ctx, id)
	if err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	if c.Public {
		return domain.Client{}, SecretResult{}, fmt.Errorf("public clients do not have secrets")
	}
	secret, err := randomSecret(32)
	if err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	c.SecretHash = hash
	c.UpdatedAt = s.Clock().UTC()
	if err := c.Validate(domain.ProductionMode); err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	if err := s.Store.PutClient(ctx, c); err != nil {
		return domain.Client{}, SecretResult{}, err
	}
	_ = s.Audit.Emit(ctx, audit.Event{Time: c.UpdatedAt, Name: "admin.client.secret_rotated", ClientID: c.ID, Result: "accepted"})
	return c, SecretResult{Generated: true, Secret: secret}, nil
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
