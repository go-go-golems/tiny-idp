package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type KeyRotationResult struct {
	Active  idpstore.SigningKey  `json:"active"`
	Retired *idpstore.SigningKey `json:"retired,omitempty"`
}

func (s *Service) GenerateSigningKey(ctx context.Context, kid string, active bool) (idpstore.SigningKey, error) {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		kid = "rsa-" + s.Clock().UTC().Format("20060102-150405")
	}
	key, err := keys.GenerateRSA(kid, s.Clock().UTC())
	if err != nil {
		return idpstore.SigningKey{}, err
	}
	key.Active = active
	if active {
		result, err := s.Store.RotateSigningKey(ctx, key, s.Clock().UTC())
		if err != nil {
			return idpstore.SigningKey{}, err
		}
		key = result.Active
	} else if err := s.Store.CreateSigningKey(ctx, key); err != nil {
		return idpstore.SigningKey{}, err
	}
	err = s.auditCommitted(ctx, idp.Event{Time: key.CreatedAt, Name: "admin.key.generated", Result: "accepted", Fields: map[string]string{"kid": key.ID}})
	return key, err
}

func (s *Service) RotateSigningKey(ctx context.Context, kid string) (KeyRotationResult, error) {
	key, retired, err := keys.RotateRSA(ctx, s.Store, strings.TrimSpace(kid), s.Clock().UTC())
	if err != nil {
		return KeyRotationResult{}, err
	}
	err = s.auditCommitted(ctx, idp.Event{Time: s.Clock().UTC(), Name: "admin.key.rotated", Result: "accepted", Fields: map[string]string{"kid": key.ID}})
	return KeyRotationResult{Active: key, Retired: retired}, err
}

func (s *Service) ListSigningKeys(ctx context.Context) ([]idpstore.SigningKey, error) {
	return s.Store.VerificationKeys(ctx)
}

func (s *Service) RetireSigningKey(ctx context.Context, kid string) error {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return fmt.Errorf("kid is required")
	}
	if err := s.Store.RetireSigningKey(ctx, kid); err != nil {
		return err
	}
	return s.auditCommitted(ctx, idp.Event{Time: s.Clock().UTC(), Name: "admin.key.retired", Result: "accepted", Fields: map[string]string{"kid": kid}})
}

func (s *Service) PurgeRetiredSigningKey(ctx context.Context, kid string) error {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return fmt.Errorf("kid is required")
	}
	if err := s.Store.DeleteRetiredSigningKey(ctx, kid); err != nil {
		return err
	}
	return s.auditCommitted(ctx, idp.Event{Time: s.Clock().UTC(), Name: "admin.key.retired_purged", Result: "accepted", Fields: map[string]string{"kid": kid, "emergency": "true"}})
}

func RedactSigningKey(key idpstore.SigningKey) idpstore.SigningKey {
	key.PrivateKeyPEM = nil
	return key
}

func RedactSigningKeys(keys []idpstore.SigningKey) []idpstore.SigningKey {
	out := make([]idpstore.SigningKey, len(keys))
	for i, key := range keys {
		out[i] = RedactSigningKey(key)
	}
	return out
}

func RedactRotationResult(result KeyRotationResult) KeyRotationResult {
	result.Active = RedactSigningKey(result.Active)
	if result.Retired != nil {
		retired := RedactSigningKey(*result.Retired)
		result.Retired = &retired
	}
	return result
}
