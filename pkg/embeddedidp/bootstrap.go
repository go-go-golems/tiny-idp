package embeddedidp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
)

type ClientProfile string

const (
	ClientProfileBrowser ClientProfile = "browser"
	ClientProfileDevice  ClientProfile = "device"
	ClientProfileGeneric ClientProfile = "generic"
)

type ClientSpec struct {
	Profile ClientProfile
	Client  idpstore.Client
}

func BrowserClient(id string, redirectURIs, postLogoutRedirectURIs, scopes []string) ClientSpec {
	return ClientSpec{Profile: ClientProfileBrowser, Client: idpstore.Client{
		ID: id, Public: true, RequirePKCE: true, RedirectURIs: redirectURIs,
		PostLogoutRedirectURIs: postLogoutRedirectURIs, AllowedScopes: scopes,
		AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour,
	}}
}

func DeviceClient(id string, scopes []string) ClientSpec {
	return ClientSpec{Profile: ClientProfileDevice, Client: idpstore.Client{
		ID: id, Public: true, RequirePKCE: true, AllowedScopes: scopes,
		AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour,
	}}
}

type BootstrapConfig struct {
	Mode         idpstore.Mode
	Clients      []ClientSpec
	SigningKeyID string
	Clock        func() time.Time
	Audit        idp.Sink
}

type BootstrapReport struct {
	ClientsCreated    []string `json:"clients_created,omitempty"`
	ClientsValidated  []string `json:"clients_validated,omitempty"`
	SigningKeyCreated bool     `json:"signing_key_created"`
	ActiveSigningKey  string   `json:"active_signing_key"`
}

var ErrBootstrapConflict = errors.New("embedding bootstrap conflict")

type ClientConflictError struct {
	ClientID string
	Fields   []string
}

func (e *ClientConflictError) Error() string {
	return fmt.Sprintf("client %q conflicts in fields: %s", e.ClientID, strings.Join(e.Fields, ", "))
}

func (e *ClientConflictError) Unwrap() error { return ErrBootstrapConflict }

// tinyidp:development-default -- production hosts inject a durable bootstrap audit sink.
func Bootstrap(ctx context.Context, store idpstore.Store, cfg BootstrapConfig) (BootstrapReport, error) {
	report := BootstrapReport{}
	if ctx == nil {
		return report, fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if store == nil {
		return report, fmt.Errorf("store is required")
	}
	mode := cfg.Mode
	if mode == "" {
		mode = idpstore.DevMode
	}
	now := time.Now
	if cfg.Clock != nil {
		now = cfg.Clock
	}
	audit := cfg.Audit
	if audit == nil {
		audit = idp.NoopSink{}
	}

	clients := make([]idpstore.Client, 0, len(cfg.Clients))
	seen := make(map[string]struct{}, len(cfg.Clients))
	for _, spec := range cfg.Clients {
		client, err := normalizeClientSpec(spec, mode)
		if err != nil {
			return report, err
		}
		if _, duplicate := seen[client.ID]; duplicate {
			return report, fmt.Errorf("%w: duplicate client id %q", ErrBootstrapConflict, client.ID)
		}
		seen[client.ID] = struct{}{}
		clients = append(clients, client)
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].ID < clients[j].ID })

	for _, desired := range clients {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		existing, err := store.GetClient(ctx, desired.ID)
		switch {
		case errors.Is(err, idpstore.ErrNotFound):
			stamp := now().UTC()
			desired.CreatedAt = stamp
			desired.UpdatedAt = stamp
			if err := store.PutClient(ctx, desired); err != nil {
				return report, fmt.Errorf("create client %q: %w", desired.ID, err)
			}
			report.ClientsCreated = append(report.ClientsCreated, desired.ID)
			if err := emitBootstrapAudit(ctx, audit, stamp, "identity.bootstrap.client_created", map[string]string{"client_id": desired.ID}); err != nil {
				return report, err
			}
		case err != nil:
			return report, fmt.Errorf("load client %q: %w", desired.ID, err)
		default:
			fields := clientConflictFields(normalizeClient(existing), desired)
			if len(fields) != 0 {
				return report, &ClientConflictError{ClientID: desired.ID, Fields: fields}
			}
			report.ClientsValidated = append(report.ClientsValidated, desired.ID)
		}
	}

	if err := ctx.Err(); err != nil {
		return report, err
	}
	active, err := store.ActiveSigningKey(ctx)
	if err == nil {
		if err := validateBootstrapSigningKey(active, now().UTC()); err != nil {
			return report, err
		}
		report.ActiveSigningKey = active.ID
		return report, nil
	}
	if !errors.Is(err, idpstore.ErrNotFound) {
		return report, fmt.Errorf("load active signing key: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	stamp := now().UTC()
	kid := strings.TrimSpace(cfg.SigningKeyID)
	if kid == "" {
		var err error
		kid, err = generatedSigningKeyID(stamp)
		if err != nil {
			return report, fmt.Errorf("generate signing key id: %w", err)
		}
	}
	key, err := keys.GenerateRSA(kid, stamp)
	if err != nil {
		return report, fmt.Errorf("generate signing key: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		return report, fmt.Errorf("create signing key %q: %w", kid, err)
	}
	report.SigningKeyCreated = true
	report.ActiveSigningKey = kid
	if err := emitBootstrapAudit(ctx, audit, stamp, "identity.bootstrap.signing_key_created", map[string]string{"key_id": kid}); err != nil {
		return report, err
	}
	return report, nil
}

func normalizeClientSpec(spec ClientSpec, mode idpstore.Mode) (idpstore.Client, error) {
	client := normalizeClient(spec.Client)
	switch spec.Profile {
	case ClientProfileBrowser:
		if len(client.RedirectURIs) == 0 {
			return idpstore.Client{}, fmt.Errorf("browser client %q requires a redirect URI", client.ID)
		}
		if !contains(client.AllowedScopes, "openid") {
			return idpstore.Client{}, fmt.Errorf("browser client %q requires openid scope", client.ID)
		}
		if !client.Public || !client.RequirePKCE || len(client.SecretHash) != 0 {
			return idpstore.Client{}, fmt.Errorf("browser client %q must be public and require PKCE", client.ID)
		}
	case ClientProfileDevice:
		if len(client.RedirectURIs) != 0 || len(client.PostLogoutRedirectURIs) != 0 {
			return idpstore.Client{}, fmt.Errorf("device client %q must not declare redirect URIs", client.ID)
		}
		if !client.Public || !client.RequirePKCE || len(client.SecretHash) != 0 {
			return idpstore.Client{}, fmt.Errorf("device client %q must be public", client.ID)
		}
	case ClientProfileGeneric:
	default:
		return idpstore.Client{}, fmt.Errorf("client %q has unknown profile %q", client.ID, spec.Profile)
	}
	if err := client.Validate(mode); err != nil {
		return idpstore.Client{}, fmt.Errorf("client %q: %w", client.ID, err)
	}
	return client, nil
}

func normalizeClient(client idpstore.Client) idpstore.Client {
	client.ID = strings.TrimSpace(client.ID)
	client.RedirectURIs = normalizedStrings(client.RedirectURIs)
	client.PostLogoutRedirectURIs = normalizedStrings(client.PostLogoutRedirectURIs)
	client.AllowedScopes = normalizedStrings(client.AllowedScopes)
	if client.AccessTokenTTL == 0 {
		client.AccessTokenTTL = time.Hour
	}
	if client.IDTokenTTL == 0 {
		client.IDTokenTTL = time.Hour
	}
	if client.RefreshTokenTTL == 0 {
		client.RefreshTokenTTL = 24 * time.Hour
	}
	client.CreatedAt = time.Time{}
	client.UpdatedAt = time.Time{}
	return client
}

func normalizedStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func clientConflictFields(existing, desired idpstore.Client) []string {
	fields := make([]string, 0, 11)
	if existing.ID != desired.ID {
		fields = append(fields, "id")
	}
	if existing.Public != desired.Public {
		fields = append(fields, "public")
	}
	if !bytes.Equal(existing.SecretHash, desired.SecretHash) {
		fields = append(fields, "secret_hash")
	}
	if !equalStrings(existing.RedirectURIs, desired.RedirectURIs) {
		fields = append(fields, "redirect_uris")
	}
	if !equalStrings(existing.PostLogoutRedirectURIs, desired.PostLogoutRedirectURIs) {
		fields = append(fields, "post_logout_redirect_uris")
	}
	if !equalStrings(existing.AllowedScopes, desired.AllowedScopes) {
		fields = append(fields, "allowed_scopes")
	}
	if existing.RequirePKCE != desired.RequirePKCE {
		fields = append(fields, "require_pkce")
	}
	if existing.AccessTokenTTL != desired.AccessTokenTTL {
		fields = append(fields, "access_token_ttl")
	}
	if existing.IDTokenTTL != desired.IDTokenTTL {
		fields = append(fields, "id_token_ttl")
	}
	if existing.RefreshTokenTTL != desired.RefreshTokenTTL {
		fields = append(fields, "refresh_token_ttl")
	}
	if existing.Disabled != desired.Disabled {
		fields = append(fields, "disabled")
	}
	return fields
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateBootstrapSigningKey(key idpstore.SigningKey, now time.Time) error {
	if !key.Active || key.ID == "" || key.Algorithm != "RS256" || now.Before(key.NotBefore) || (!key.NotAfter.IsZero() && !now.Before(key.NotAfter)) {
		return fmt.Errorf("active signing key metadata is invalid")
	}
	privateKey, err := keys.ParseRSAPrivateKey(key)
	if err != nil || privateKey.N.BitLen() < 2048 {
		return fmt.Errorf("active signing key %q is invalid", key.ID)
	}
	return nil
}

func generatedSigningKeyID(now time.Time) (string, error) {
	random := make([]byte, 9)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "bootstrap-" + now.Format("20060102T150405Z") + "-" + base64.RawURLEncoding.EncodeToString(random), nil
}

func emitBootstrapAudit(ctx context.Context, sink idp.Sink, at time.Time, name string, fields map[string]string) error {
	if err := sink.Emit(ctx, idp.Event{Time: at, Name: name, Result: "accepted", Fields: fields}); err != nil {
		return fmt.Errorf("%w: %v", idp.ErrAuditDelivery, err)
	}
	return nil
}
