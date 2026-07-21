package idpaccounts

import (
	"context"
	"fmt"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

const (
	DisplayNameLookupCapabilityID      = "identity.displayName.lookup"
	DisplayNameLookupCapabilityVersion = 1
)

// DisplayNameAvailable reports whether an optional unique-name claim has
// already reserved the canonical representation. It is useful for a bounded
// signup preflight but is not an authority for account creation: callers must
// still request a transactional reservation through CreateRequest.
func (s *Service) DisplayNameAvailable(ctx context.Context, displayName string) (bool, error) {
	if s == nil || s.store == nil {
		return false, fmt.Errorf("account service is unavailable")
	}
	claims, ok := s.store.(idpstore.DisplayNameStore)
	if !ok {
		return false, fmt.Errorf("account store does not support display-name claims")
	}
	key := NormalizeDisplayName(displayName)
	if key == "" {
		return false, nil
	}
	return claims.DisplayNameAvailable(ctx, key)
}
