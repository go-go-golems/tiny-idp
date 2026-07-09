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
	if err := s.Store.CreateSigningKey(ctx, key); err != nil {
		return idpstore.SigningKey{}, err
	}
	if active {
		if err := s.Store.ActivateSigningKey(ctx, key.ID); err != nil {
			return idpstore.SigningKey{}, err
		}
		key.Active = true
	}
	_ = s.Audit.Emit(ctx, idp.Event{Time: key.CreatedAt, Name: "admin.key.generated", Result: "accepted", Fields: map[string]string{"kid": key.ID}})
	return key, nil
}

func (s *Service) RotateSigningKey(ctx context.Context, kid string) (KeyRotationResult, error) {
	key, retired, err := keys.RotateRSA(ctx, s.Store, strings.TrimSpace(kid), s.Clock().UTC())
	if err != nil {
		return KeyRotationResult{}, err
	}
	_ = s.Audit.Emit(ctx, idp.Event{Time: s.Clock().UTC(), Name: "admin.key.rotated", Result: "accepted", Fields: map[string]string{"kid": key.ID}})
	return KeyRotationResult{Active: key, Retired: retired}, nil
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
	_ = s.Audit.Emit(ctx, idp.Event{Time: s.Clock().UTC(), Name: "admin.key.retired", Result: "accepted", Fields: map[string]string{"kid": kid}})
	return nil
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
