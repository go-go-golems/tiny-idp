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
		// Return the same claims the ID token would, but with a different sub,
		// so the RP catches the disagreement between ID token and userinfo.
		resp := userinfoClaims(at)
		resp["sub"] = at.User.Sub + "-different"
		writeJSON(w, http.StatusOK, resp)
		return
	}

	writeJSON(w, http.StatusOK, userinfoClaims(at))
}

// userinfoClaims builds the userinfo response body for an access token. It
// mirrors the ID token's user-facing claims (sub/email/name + the scenario's
// ExtraClaims), so that under normal scenarios the ID token and userinfo
// agree. The sub_mismatch scenario overrides sub after calling this.

// userinfoClaims builds the userinfo response body for an access token: the
// base user claims (sub/email/email_verified/name) merged with the scenario's
// ExtraClaims and with OmitClaims deleted. This mirrors what the ID token
// carries, so under normal scenarios the ID token and userinfo agree on the
// user's attributes.
func userinfoClaims(at accessToken) map[string]any {
	resp := map[string]any{
		"sub":            at.User.Sub,
		"email":          at.User.Email,
		"email_verified": true,
		"name":           at.User.Name,
	}
	for k, v := range at.Scenario.ExtraClaims {
		resp[k] = v
	}
	for _, k := range at.Scenario.OmitClaims {
		delete(resp, k)
	}
	return resp
}
