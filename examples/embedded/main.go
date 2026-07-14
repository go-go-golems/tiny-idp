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

func main() {
	ctx := context.Background()
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig("tinyidp.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	accounts, err := idpaccounts.NewService(st, idpaccounts.Options{})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := accounts.Create(ctx, idpaccounts.CreateRequest{
		Login: "alice", Password: []byte("correct horse battery staple"),
		Email: "alice@example.test", EmailVerified: true, Name: "Alice Example",
	}); err != nil && !errors.Is(err, idpstore.ErrDuplicate) {
		log.Fatal(err)
	}
	if _, err := embeddedidp.Bootstrap(ctx, st, embeddedidp.BootstrapConfig{
		Mode: embeddedidp.DevMode,
		Clients: []embeddedidp.ClientSpec{embeddedidp.BrowserClient(
			"embedded-example",
			[]string{"http://127.0.0.1:8080/auth/callback"},
			[]string{"http://127.0.0.1:8080/"},
			[]string{"openid", "profile", "email"},
		)},
		SigningKeyID: "embedded-example-key",
	}); err != nil {
		log.Fatal(err)
	}

	secretKey := []byte("example-secret-key-32-bytes-minimum")

	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer:        "http://127.0.0.1:5556/idp",
		Mode:          embeddedidp.DevMode,
		Store:         st,
		Token:         embeddedidp.TokenConfig{SecretKey: secretKey},
		Authenticator: accounts,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = provider.Close(context.Background()) }()

	mux := http.NewServeMux()
	mux.Handle("/", provider.Handler())
	server := &http.Server{Addr: "127.0.0.1:5556", Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("embedded issuer listening at http://127.0.0.1:5556/idp (alice / correct horse battery staple)")
	log.Fatal(server.ListenAndServe())
}
