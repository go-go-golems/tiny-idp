package server

import (
	"net/http"
	"strings"
	"time"
)

// userinfo implements the UserInfo endpoint: given a bearer access token,
// return the authenticated user's claims.
//
// The access token is opaque (a random string mapped in-memory to a user),
// not a JWT. It is looked up under the mutex; expired tokens are rejected.
func (s *Server) userinfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))

	s.mu.Lock()
	at, ok := s.tokens[token]
	s.mu.Unlock()

	if !ok || time.Now().After(at.Expires) {
		http.Error(w, "invalid bearer token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sub":            at.User.Sub,
		"email":          at.User.Email,
		"email_verified": true,
		"name":           at.User.Name,
	})
}
