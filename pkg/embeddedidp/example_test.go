package embeddedidp_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idpaccounts"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

func Example_browserComposition() {
	ctx := context.Background()
	directory, err := os.MkdirTemp("", "tinyidp-browser-example-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(directory, "identity.sqlite")))
	if err != nil {
		panic(err)
	}
	defer store.Close()

	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		panic(err)
	}
	_, err = accounts.Create(ctx, idpaccounts.CreateRequest{
		Login: "alice", Password: []byte("correct horse battery staple"), Name: "Alice",
	})
	if err != nil {
		panic(err)
	}
	report, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{
		Clients: []embeddedidp.ClientSpec{embeddedidp.BrowserClient(
			"message-app",
			[]string{"http://127.0.0.1:8080/auth/callback"},
			[]string{"http://127.0.0.1:8080/"},
			[]string{"openid", "profile"},
		)},
		SigningKeyID: "example-browser-key",
	})
	if err != nil {
		panic(err)
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer: "http://127.0.0.1:5556/idp", Mode: embeddedidp.DevMode, Store: store,
		Token: embeddedidp.TokenConfig{SecretKey: []byte("example-secret-key-32-bytes-minimum")}, Authenticator: accounts,
	})
	if err != nil {
		panic(err)
	}
	defer provider.Close(context.Background())
	transport, err := embeddedidp.NewInProcessIssuerTransport(
		"http://127.0.0.1:5556/idp", provider.Handler(), embeddedidp.InProcessTransportOptions{},
	)
	if err != nil {
		panic(err)
	}
	request, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:5556/idp/.well-known/openid-configuration", nil)
	response, err := transport.RoundTrip(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	fmt.Println(report.ClientsCreated[0], report.SigningKeyCreated, response.StatusCode)
	// Output: message-app true 200
}

func Example_deviceClientBootstrap() {
	ctx := context.Background()
	directory, err := os.MkdirTemp("", "tinyidp-device-example-")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(directory, "identity.sqlite")))
	if err != nil {
		panic(err)
	}
	defer store.Close()
	report, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{
		Clients:      []embeddedidp.ClientSpec{embeddedidp.DeviceClient("example-cli", []string{"openid", "profile"})},
		SigningKeyID: "example-device-key",
	})
	if err != nil {
		panic(err)
	}
	client, err := store.GetClient(ctx, "example-cli")
	if err != nil {
		panic(err)
	}
	fmt.Println(report.ClientsCreated[0], client.Public, client.RequirePKCE, len(client.RedirectURIs))
	// Output: example-cli true true 0
}
