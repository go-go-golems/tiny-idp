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

	// UserInfo-error scenarios simulate failures at the userinfo endpoint.
	switch at.Scenario.UserInfoError {
	case "401":
		http.Error(w, "simulated invalid bearer token", http.StatusUnauthorized)
		return
	case "500":
		http.Error(w, "simulated userinfo server error", http.StatusInternalServerError)
		return
	case "sub_mismatch":
		writeJSON(w, http.StatusOK, map[string]any{
			"sub":            at.User.Sub + "-different",
			"email":          at.User.Email,
			"email_verified": true,
			"name":           at.User.Name,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sub":            at.User.Sub,
		"email":          at.User.Email,
		"email_verified": true,
		"name":           at.User.Name,
	})
}
