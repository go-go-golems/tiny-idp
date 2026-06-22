// Command tinyidp runs the mock OIDC Identity Provider.
//
// It is a local development and integration testing tool, NOT production
// grade (no real login, consent, persistent keys, refresh tokens, or TLS
// enforcement). Bind to loopback (the default) and never expose it publicly.
// See the design doc in ttmp/ (ticket MOCK-OIDC-IDP) for full scope.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/manuel/tinyidp/internal/server"
)

func main() {
	srv, err := server.New(server.Options{
		Issuer:       strings.TrimRight(env("OIDC_ISSUER", "http://localhost:5556"), "/"),
		ClientID:     env("OIDC_CLIENT_ID", "dev-client"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURIs: parseCSV(env("OIDC_REDIRECT_URIS", "http://localhost:3000/callback,http://127.0.0.1:3000/callback")),
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	addr := env("OIDC_ADDR", "127.0.0.1:5556")
	log.Printf("tinyidp listening on %s; issuer=%s client_id=%s", addr, srv.Issuer(), srv.ClientID())
	log.Fatal(http.ListenAndServe(addr, server.WithCORS(mux)))
}

// env returns the value of k, or dflt if empty/unset.
func env(k, dflt string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return dflt
}

// parseCSV splits a comma-separated string into a trimmed slice, dropping
// empty entries. Used for OIDC_REDIRECT_URIS.
func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
