package main

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestOIDCClientDiscoversEmbeddedIssuerInProcess(t *testing.T) {
	ctx := context.Background()
	baseURL := "http://127.0.0.1:8090"
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "identity.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{Mode: embeddedidp.DevMode,
		Clients:      []embeddedidp.ClientSpec{embeddedidp.BrowserClient(clientID, []string{baseURL + callbackPath}, []string{baseURL + "/"}, []string{"openid", "profile"})},
		SigningKeyID: "message-app-test-key"}); err != nil {
		t.Fatal(err)
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: baseURL + issuerPath, Mode: embeddedidp.DevMode, Store: store,
		Token: embeddedidp.TokenConfig{SecretKey: []byte("message-app-test-token-secret-32bytes")}, Authenticator: accounts})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close(context.Background())
	transport, err := embeddedidp.NewInProcessIssuerTransport(baseURL+issuerPath, provider.Handler(), embeddedidp.InProcessTransportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newOIDCClient(ctx, baseURL+issuerPath, baseURL, &http.Client{Transport: transport, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if client.publicOrigin != baseURL || client.config.Endpoint.AuthURL != baseURL+issuerPath+"/authorize" || client.config.RedirectURL != baseURL+callbackPath {
		t.Fatalf("unexpected discovered client config: %#v", client.config)
	}
}
