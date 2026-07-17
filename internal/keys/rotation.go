package keys

import (
	"context"
	"fmt"
	"time"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

// RotateRSA creates a new RSA signing key, activates it, and retires the
// previously active key while leaving it available through VerificationKeys.
// Keeping the retired key published lets relying parties validate ID Tokens
// issued immediately before rotation until their token lifetime has elapsed.
func RotateRSA(ctx context.Context, store idpstore.AtomicStore, kid string, now time.Time) (idpstore.SigningKey, *idpstore.SigningKey, error) {
	if kid == "" {
		return idpstore.SigningKey{}, nil, fmt.Errorf("kid is required")
	}
	key, err := GenerateRSA(kid, now)
	if err != nil {
		return idpstore.SigningKey{}, nil, err
	}
	result, err := store.RotateSigningKey(ctx, key, now)
	if err != nil {
		return idpstore.SigningKey{}, nil, fmt.Errorf("rotate signing key: %w", err)
	}
	return result.Active, result.Retired, nil
}
