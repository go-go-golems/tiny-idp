package embeddedidp

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

func TestBootstrapCreatesBrowserDeviceAndKeyIdempotently(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 7, 13, 20, 0, 0, 0, time.UTC)
	cfg := BootstrapConfig{
		Mode: idpstore.ProductionMode,
		Clients: []ClientSpec{
			DeviceClient("device-cli", []string{"profile", "openid", "profile"}),
			BrowserClient("browser-app", []string{"https://app.example.test/callback"}, []string{"https://app.example.test/"}, []string{"email", "openid"}),
		},
		SigningKeyID: "initial-key",
		Clock:        func() time.Time { return now },
	}
	report, err := Bootstrap(ctx, store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.ClientsCreated, []string{"browser-app", "device-cli"}) || len(report.ClientsValidated) != 0 || !report.SigningKeyCreated || report.ActiveSigningKey != "initial-key" {
		t.Fatalf("first report = %#v", report)
	}
	device, err := store.GetClient(ctx, "device-cli")
	if err != nil {
		t.Fatal(err)
	}
	if len(device.RedirectURIs) != 0 || !device.Public || !device.RequirePKCE || !reflect.DeepEqual(device.AllowedScopes, []string{"openid", "profile"}) || !reflect.DeepEqual(device.AllowedGrantTypes, []string{idpstore.GrantDeviceCode}) {
		t.Fatalf("device client = %#v", device)
	}
	browser, err := store.GetClient(ctx, "browser-app")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(browser.AllowedGrantTypes, []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}) {
		t.Fatalf("browser grant types = %#v", browser.AllowedGrantTypes)
	}

	report, err = Bootstrap(ctx, store, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ClientsCreated) != 0 || !reflect.DeepEqual(report.ClientsValidated, []string{"browser-app", "device-cli"}) || report.SigningKeyCreated || report.ActiveSigningKey != "initial-key" {
		t.Fatalf("second report = %#v", report)
	}
}

func TestBootstrapRejectsAllInvalidDeclarationsBeforeWrites(t *testing.T) {
	tests := map[string][]ClientSpec{
		"browser without redirect": {BrowserClient("browser", nil, nil, []string{"openid"})},
		"browser without openid":   {BrowserClient("browser", []string{"https://app.example/callback"}, nil, []string{"profile"})},
		"device with redirect":     {{Profile: ClientProfileDevice, Client: idpstore.Client{ID: "device", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example/callback"}}}},
		"duplicate normalized ids": {DeviceClient(" same ", []string{"openid"}), DeviceClient("same", []string{"openid"})},
	}
	for name, specs := range tests {
		t.Run(name, func(t *testing.T) {
			store := memory.New()
			if _, err := Bootstrap(context.Background(), store, BootstrapConfig{Clients: specs}); err == nil {
				t.Fatal("Bootstrap returned nil error")
			}
			clients, err := store.ListClients(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(clients) != 0 {
				t.Fatalf("clients were written: %#v", clients)
			}
			if _, err := store.ActiveSigningKey(context.Background()); !errors.Is(err, idpstore.ErrNotFound) {
				t.Fatalf("active key error = %v", err)
			}
		})
	}
}

func TestBootstrapGenericClientValidation(t *testing.T) {
	store := memory.New()
	_, err := Bootstrap(context.Background(), store, BootstrapConfig{Mode: idpstore.ProductionMode, Clients: []ClientSpec{{Profile: ClientProfileGeneric, Client: idpstore.Client{ID: "confidential", RedirectURIs: []string{"https://app.example/callback"}}}}})
	if !errors.Is(err, idpstore.ErrConfidentialMissingSecret) {
		t.Fatalf("generic validation error = %v", err)
	}
}

func TestBootstrapReportsOnlyConflictFieldNames(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	existing := BrowserClient("app", []string{"https://app.example/old"}, nil, []string{"openid"}).Client
	existing.SecretHash = []byte("stored-secret-material")
	if err := store.PutClient(ctx, existing); err != nil {
		t.Fatal(err)
	}
	_, err := Bootstrap(ctx, store, BootstrapConfig{Clients: []ClientSpec{BrowserClient("app", []string{"https://app.example/new"}, nil, []string{"openid"})}})
	var conflict *ClientConflictError
	if !errors.As(err, &conflict) || !errors.Is(err, ErrBootstrapConflict) {
		t.Fatalf("conflict error = %v", err)
	}
	if !reflect.DeepEqual(conflict.Fields, []string{"secret_hash", "redirect_uris"}) {
		t.Fatalf("conflict fields = %#v", conflict.Fields)
	}
	if strings.Contains(err.Error(), "stored-secret-material") {
		t.Fatalf("conflict leaked secret: %v", err)
	}
}

func TestBootstrapTreatsListOrderAndTimestampsAsEquivalent(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	existing := BrowserClient("app", []string{"https://app.example/b", "https://app.example/a"}, nil, []string{"profile", "openid"}).Client
	existing.CreatedAt = time.Unix(1, 0)
	existing.UpdatedAt = time.Unix(2, 0)
	if err := store.PutClient(ctx, existing); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("existing-key", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	report, err := Bootstrap(ctx, store, BootstrapConfig{Clients: []ClientSpec{BrowserClient("app", []string{"https://app.example/a", "https://app.example/b", "https://app.example/a"}, nil, []string{"openid", "profile"})}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.ClientsValidated, []string{"app"}) || report.ActiveSigningKey != "existing-key" || report.SigningKeyCreated {
		t.Fatalf("report = %#v", report)
	}
}

func TestBootstrapFailsOnCorruptOrExpiredActiveKey(t *testing.T) {
	for name, key := range map[string]idpstore.SigningKey{
		"corrupt": {ID: "bad", Algorithm: "RS256", Active: true, PrivateKeyPEM: []byte("not PEM")},
		"expired": {ID: "expired", Algorithm: "RS256", Active: true, PrivateKeyPEM: []byte("not relevant"), NotAfter: time.Now().Add(-time.Minute)},
	} {
		t.Run(name, func(t *testing.T) {
			store := memory.New()
			if err := store.CreateSigningKey(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			report, err := Bootstrap(context.Background(), store, BootstrapConfig{})
			if err == nil || report.ActiveSigningKey != "" || strings.Contains(err.Error(), string(key.PrivateKeyPEM)) {
				t.Fatalf("report = %#v, error = %v", report, err)
			}
		})
	}
}

type failingBootstrapAudit struct{ failName string }

func (s failingBootstrapAudit) Emit(_ context.Context, event idp.Event) error {
	if event.Name == s.failName {
		return errors.New("audit unavailable")
	}
	return nil
}

func TestBootstrapReturnsCommittedReportOnAuditFailure(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	report, err := Bootstrap(ctx, store, BootstrapConfig{
		Clients: []ClientSpec{DeviceClient("device", []string{"openid"})},
		Audit:   failingBootstrapAudit{failName: "identity.bootstrap.client_created"},
	})
	if !errors.Is(err, idp.ErrAuditDelivery) || !reflect.DeepEqual(report.ClientsCreated, []string{"device"}) {
		t.Fatalf("client audit report = %#v, error = %v", report, err)
	}
	if _, err := store.GetClient(ctx, "device"); err != nil {
		t.Fatalf("client did not commit: %v", err)
	}

	store = memory.New()
	report, err = Bootstrap(ctx, store, BootstrapConfig{SigningKeyID: "audit-key", Audit: failingBootstrapAudit{failName: "identity.bootstrap.signing_key_created"}})
	if !errors.Is(err, idp.ErrAuditDelivery) || !report.SigningKeyCreated || report.ActiveSigningKey != "audit-key" {
		t.Fatalf("key audit report = %#v, error = %v", report, err)
	}
}

func TestBootstrapHonorsCanceledContextBeforeWrites(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := memory.New()
	if _, err := Bootstrap(ctx, store, BootstrapConfig{Clients: []ClientSpec{DeviceClient("device", []string{"openid"})}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	clients, err := store.ListClients(context.Background())
	if err != nil || len(clients) != 0 {
		t.Fatalf("clients = %#v, error = %v", clients, err)
	}
}
