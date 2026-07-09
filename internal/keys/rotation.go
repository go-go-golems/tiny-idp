package keys

import (
	"context"
	"errors"
	"fmt"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

// RotateRSA creates a new RSA signing key, activates it, and retires the
// previously active key while leaving it available through VerificationKeys.
// Keeping the retired key published lets relying parties validate ID Tokens
// issued immediately before rotation until their token lifetime has elapsed.
func RotateRSA(ctx context.Context, store idpstore.KeyStore, kid string, now time.Time) (idpstore.SigningKey, *idpstore.SigningKey, error) {
	if kid == "" {
		return idpstore.SigningKey{}, nil, fmt.Errorf("kid is required")
	}
	old, err := store.ActiveSigningKey(ctx)
	var retired *idpstore.SigningKey
	if err == nil {
		oldCopy := old
		retired = &oldCopy
	} else if !errors.Is(err, idpstore.ErrNotFound) {
		return idpstore.SigningKey{}, nil, fmt.Errorf("load active signing key: %w", err)
	}

	key, err := GenerateRSA(kid, now)
	if err != nil {
		return idpstore.SigningKey{}, nil, err
	}
	if retired != nil {
		key.Active = false
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		return idpstore.SigningKey{}, nil, fmt.Errorf("create signing key: %w", err)
	}
	if retired != nil {
		if err := store.ActivateSigningKey(ctx, key.ID); err != nil {
			return idpstore.SigningKey{}, nil, fmt.Errorf("activate signing key: %w", err)
		}
		if err := store.RetireSigningKey(ctx, retired.ID); err != nil {
			return idpstore.SigningKey{}, nil, fmt.Errorf("retire old signing key: %w", err)
		}
		updated, err := store.ActiveSigningKey(ctx)
		if err != nil {
			return idpstore.SigningKey{}, nil, err
		}
		key = updated
	}
	return key, retired, nil
}
