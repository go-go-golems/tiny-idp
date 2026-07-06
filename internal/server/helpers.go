package server

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"
)

// writeJSON serializes v as JSON with the given status. Errors are ignored
// because the response is already being written; callers in tests should
// assert on the decoded body instead.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// tokenError writes an OAuth 2.0 token-endpoint error response
// ({"error":..., "error_description":...}) with no-store cache headers.
func tokenError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": desc,
	})
}

// randomB64 returns n bytes of crypto-random data as raw URL-safe base64.
// Used for authorization codes and opaque access tokens.
func randomB64(n int) string {
	return b64(randomBytes(n))
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// rand.Read failing indicates a broken system CSPRNG; panicking is
		// correct because nothing the server can do afterward is safe.
		panic(err)
	}
	return b
}

// hasScope reports whether the space-separated scope string contains wanted.
func hasScope(scope, wanted string) bool {
	for _, s := range strings.Fields(scope) {
		if s == wanted {
			return true
		}
	}
	return false
}

// WithCORS wraps an http.Handler with permissive CORS headers for local
// browser-based RPs. This is intentionally open because the server binds to
// loopback by default; do not use it behind a public-facing deployment.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, DPoP")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
