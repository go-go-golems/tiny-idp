package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idpaccounts"
	"github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

const (
	listenAddress = "127.0.0.1:5556"
	publicBaseURL = "http://127.0.0.1:5556"
	issuerURL     = publicBaseURL + "/idp"
	clientID      = "embedded-example"
)

func main() {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig("tinyidp.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	accounts, err := idpaccounts.NewService(store, idpaccounts.Options{})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := accounts.Create(ctx, idpaccounts.CreateRequest{
		Login: "alice", Password: []byte("correct horse battery staple"),
		Email: "alice@example.test", EmailVerified: true, Name: "Alice Example",
	}); err != nil && !errors.Is(err, idpstore.ErrDuplicate) {
		log.Fatal(err)
	}
	if _, err := embeddedidp.Bootstrap(ctx, store, embeddedidp.BootstrapConfig{
		Mode: embeddedidp.DevMode,
		Clients: []embeddedidp.ClientSpec{embeddedidp.BrowserClient(
			clientID,
			[]string{publicBaseURL + "/auth/callback"},
			[]string{publicBaseURL + "/"},
			[]string{"openid", "profile", "email"},
		)},
		SigningKeyID: "embedded-example-key",
	}); err != nil {
		log.Fatal(err)
	}

	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer: issuerURL, Mode: embeddedidp.DevMode, Store: store,
		Token:         embeddedidp.TokenConfig{SecretKey: []byte("example-secret-key-32-bytes-minimum")},
		Authenticator: accounts,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = provider.Close(context.Background()) }()

	transport, err := embeddedidp.NewInProcessIssuerTransport(
		issuerURL, provider.Handler(), embeddedidp.InProcessTransportOptions{},
	)
	if err != nil {
		log.Fatal(err)
	}
	relyingParty, err := newRelyingParty(rpOptions{
		PublicBaseURL: publicBaseURL,
		Issuer:        issuerURL,
		ClientID:      clientID,
		HTTPClient:    &http.Client{Transport: transport, Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/idp/", provider.Handler())
	mux.Handle("/", relyingParty)
	server := &http.Server{
		Addr: listenAddress, Handler: mux,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	log.Printf("self-contained app listening at %s (alice / correct horse battery staple)", publicBaseURL)
	log.Fatal(server.ListenAndServe())
}
