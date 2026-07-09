package embeddedidp

import (
	"context"
	"fmt"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/oidcmeta"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type Mode = idpstore.Mode

const (
	DevMode        = idpstore.DevMode
	ProductionMode = idpstore.ProductionMode
)

type CookieConfig struct {
	Secure bool
}

type TokenConfig struct {
	SecretKey []byte
}

type Options struct {
	Issuer         string
	Mode           Mode
	Store          idpstore.Store
	Cookie         CookieConfig
	Token          TokenConfig
	Audit          idp.Sink
	Consent        idp.ConsentPolicy
	RateLimiter    idp.RateLimiter
	ClientAddress  idp.ClientAddressResolver
	Authenticator  idp.PasswordAuthenticator
	PasswordPolicy idp.PasswordAcceptancePolicy
	PasswordWork   idp.PasswordWorkConfig
}

func (o Options) Validate(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
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
	clients, err := o.Store.ListClients(ctx)
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
		if o.Audit == nil {
			return fmt.Errorf("production mode requires an audit sink")
		}
		if o.RateLimiter == nil {
			return fmt.Errorf("production mode requires a rate limiter")
		}
		if o.ClientAddress == nil {
			return fmt.Errorf("production mode requires a client address resolver")
		}
		if reporter, ok := o.RateLimiter.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
			return fmt.Errorf("production mode requires a production-ready rate limiter")
		}
		if reporter, ok := o.ClientAddress.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
			return fmt.Errorf("production mode requires a production-ready client address resolver")
		}
		policy := o.PasswordPolicy
		if policy.MinCharacters == 0 {
			policy = idp.DefaultPasswordAcceptancePolicy()
		}
		if policy.MinCharacters < 15 || policy.MaxCharacters < 64 || policy.Blocklist == nil {
			return fmt.Errorf("production mode requires NIST-aligned password acceptance policy")
		}
		work := o.PasswordWork
		if work.MaxConcurrent == 0 {
			work = idp.DefaultPasswordWorkConfig()
		}
		if work.MaxConcurrent < 1 {
			return fmt.Errorf("production mode requires bounded password work")
		}
		if o.Authenticator != nil {
			if reporter, ok := o.Authenticator.(idp.ProductionReadyReporter); !ok || !reporter.ProductionReady() {
				return fmt.Errorf("production mode requires a production-ready password authenticator")
			}
			if _, ok := o.Authenticator.(idp.PasswordWorkReporter); !ok {
				return fmt.Errorf("production mode requires password work metrics")
			}
		}
		reporter, ok := o.Store.(idpstore.PersistentReporter)
		if !ok || !reporter.Persistent() {
			return fmt.Errorf("production mode requires persistent stores")
		}
		schema, ok := o.Store.(idpstore.SchemaReporter)
		if !ok {
			return fmt.Errorf("production mode requires schema reporting")
		}
		if version, err := schema.SchemaVersion(ctx); err != nil || version <= 0 {
			return fmt.Errorf("production mode requires a supported schema")
		}
		key, err := o.Store.ActiveSigningKey(ctx)
		if err != nil {
			return fmt.Errorf("production mode requires active signing key: %w", err)
		}
		now := time.Now().UTC()
		if key.Algorithm != "RS256" || key.ID == "" || now.Before(key.NotBefore) || (!key.NotAfter.IsZero() && !now.Before(key.NotAfter)) {
			return fmt.Errorf("production mode requires a currently usable RS256 signing key")
		}
		privateKey, err := keys.ParseRSAPrivateKey(key)
		if err != nil || privateKey.N.BitLen() < 2048 {
			return fmt.Errorf("production mode requires a valid RSA signing key of at least 2048 bits")
		}
		verificationKeys, err := o.Store.VerificationKeys(ctx)
		if err != nil {
			return fmt.Errorf("production mode requires verification keys: %w", err)
		}
		active := 0
		for _, verificationKey := range verificationKeys {
			if verificationKey.Active {
				active++
			}
		}
		if active != 1 {
			return fmt.Errorf("production mode requires exactly one active signing key")
		}
	}
	return nil
}
