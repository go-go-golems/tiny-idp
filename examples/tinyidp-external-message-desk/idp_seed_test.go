package externalmessagedesk

import (
	"context"
	"testing"

	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
)

func TestSeedManifestBootstrapIsIdempotentAndRejectsDrift(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	manifest := SeedManifest{ClientID: "message-desk", RedirectURIs: []string{"http://127.0.0.1:8080/auth/callback"}, PostLogoutRedirectURIs: []string{"http://127.0.0.1:8080/"}, Accounts: []SeedAccount{{ID: "demo-amelie", Subject: "demo-amelie", Login: "amelie", Password: "long-demo-password", Name: "Amelie", Email: "amelie@example.test"}}}
	if err := manifest.Bootstrap(ctx, store, accounts, embeddedidp.DevMode); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Bootstrap(ctx, store, accounts, embeddedidp.DevMode); err != nil {
		t.Fatalf("idempotent bootstrap: %v", err)
	}
	drift := manifest
	drift.Accounts = append([]SeedAccount(nil), manifest.Accounts...)
	drift.Accounts[0].Name = "Different"
	if err := drift.Bootstrap(ctx, store, accounts, embeddedidp.DevMode); err == nil {
		t.Fatal("seed identity drift was accepted")
	}
}
