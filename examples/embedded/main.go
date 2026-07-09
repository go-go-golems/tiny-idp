//go:build ignore

package main

import (
	"context"
	"log"
	"net/http"

	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func main() {
	ctx := context.Background()
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig("tinyidp.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	// Provision clients, users, credentials, and an active signing key with the
	// tinyidp admin commands before starting the embedded provider.
	secretKey := []byte("example-secret-key-32-bytes-minimum")

	provider, err := embeddedidp.New(ctx, embeddedidp.Options{
		Issuer: "http://127.0.0.1:5556",
		Mode:   embeddedidp.DevMode,
		Store:  st,
		Token:  embeddedidp.TokenConfig{SecretKey: secretKey},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = provider.Close(context.Background()) }()

	mux := http.NewServeMux()
	mux.Handle("/", provider.Handler())
	log.Fatal(http.ListenAndServe("127.0.0.1:5556", mux))
}
