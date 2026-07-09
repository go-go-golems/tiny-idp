//go:build ignore

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
)

func main() {
	ctx := context.Background()
	st := memory.New()
	secretKey := []byte("example-secret-key-32-bytes-minimum")
	_ = st.PutClient(ctx, idpstore.Client{
		ID: "example-app", Public: true,
		RedirectURIs:  []string{"http://localhost:8080/callback"},
		AllowedScopes: []string{"openid", "profile", "email", "offline_access"},
		RequirePKCE:   true,
	})
	_ = st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", EmailVerified: true, Name: "Alice"})
	key, err := keys.GenerateRSA("example-key", time.Now())
	if err != nil {
		log.Fatal(err)
	}
	_ = st.CreateSigningKey(ctx, key)

	provider, err := embeddedidp.New(embeddedidp.Options{
		Issuer: "http://127.0.0.1:5556",
		Mode:   embeddedidp.DevMode,
		Store:  st,
		Token:  embeddedidp.TokenConfig{SecretKey: secretKey},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", provider.Handler())
	log.Fatal(http.ListenAndServe("127.0.0.1:5556", mux))
}
