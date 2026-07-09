package embeddedidp

import (
	"context"
	"fmt"

	"github.com/manuel/tinyidp/internal/audit"
	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/oidcmeta"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type Mode = idpstore.Mode

const (
	DevMode        = idpstore.DevMode
	ProductionMode = idpstore.ProductionMode
)

type CookieConfig struct {
	Secure   bool
	SameSite string
}

type TokenConfig struct {
	SecretKey []byte
}

type Options struct {
	Issuer                          string
	Mode                            Mode
	Store                           idpstore.Store
	Cookie                          CookieConfig
	Token                           TokenConfig
	Audit                           audit.Sink
	Consent                         fositeadapter.ConsentPolicy
	RateLimiter                     fositeadapter.RateLimiter
	Authenticator                   fositeadapter.PasswordAuthenticator
	AllowInMemoryStoresInProduction bool
}

func (o Options) Validate() error {
	mode := o.Mode
	if mode == "" {
		mode = DevMode
	}
	if err := oidcmeta.ValidateIssuer(o.Issuer, mode); err != nil {
		return err
	}
	if o.Store == nil {
		return fmt.Errorf("store is required")
	}
	clients, err := o.Store.ListClients(context.Background())
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}
	for _, c := range clients {
		if err := c.Validate(mode); err != nil {
			return fmt.Errorf("client %q: %w", c.ID, err)
		}
	}
	if mode == ProductionMode {
		if len(o.Token.SecretKey) < 32 {
			return fmt.Errorf("production mode requires token secret key of at least 32 bytes")
		}
		if !o.Cookie.Secure {
			return fmt.Errorf("production cookies must be secure")
		}
		if reporter, ok := o.Store.(idpstore.PersistentReporter); ok && !reporter.Persistent() && !o.AllowInMemoryStoresInProduction {
			return fmt.Errorf("production mode requires persistent stores")
		}
		if _, err := o.Store.ActiveSigningKey(context.Background()); err != nil {
			return fmt.Errorf("production mode requires active signing key: %w", err)
		}
	}
	return nil
}
