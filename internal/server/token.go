package server

import (
	"net/http"
	"time"
)

// token implements the token endpoint: exchange an authorization code for an
// ID token + access token.
//
// Client auth supports both client_secret_basic (HTTP Basic) and
// client_secret_post. Public clients (empty client secret) skip the secret
// check. The auth code is popped atomically under the mutex because
// authorization codes are one-time use.
func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported")
		return
	}

	clientID, basicSecret, hasBasic := r.BasicAuth()
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	c, ok := s.clients.Lookup(clientID)
	if !ok {
		tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_id")
		return
	}

	// Confidential clients (non-empty secret) must present it; public clients
	// (empty secret) skip the check. This matches the client_secret_basic and
	// client_secret_post methods advertised in discovery.
	if c.Secret != "" {
		secret := r.Form.Get("client_secret")
		if hasBasic {
			secret = basicSecret
		}
		if secret != c.Secret {
			tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_secret")
			return
		}
	}

	// The client_id on the code (set at /authorize) must match the client
	// authenticating at /token. A code issued to one client cannot be
	// redeemed by another.
	code := r.Form.Get("code")

	// Pop the code atomically: authorization codes are one-time use, so the
	// read and delete must share one critical section to avoid a code-reuse
	// race between two concurrent token exchanges.
	s.mu.Lock()
	ac, ok := s.codes[code]
	delete(s.codes, code)
	s.mu.Unlock()

	if !ok || time.Now().After(ac.Expires) {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown or expired code")
		return
	}
	if ac.ClientID != clientID || ac.RedirectURI != r.Form.Get("redirect_uri") {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "client_id or redirect_uri mismatch")
		return
	}
	if !verifyPKCE(ac.CodeChallenge, ac.CodeChallengeMethod, r.Form.Get("code_verifier")) {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	now := time.Now()
	access := randomB64(32)

	// Token-error scenarios simulate failures at the token endpoint.
	switch ac.Scenario.TokenError {
	case "invalid_grant":
		tokenError(w, http.StatusBadRequest, "invalid_grant", "simulated invalid_grant during token exchange")
		return
	case "server_error":
		tokenError(w, http.StatusInternalServerError, "server_error", "simulated token endpoint failure")
		return
	case "slow":
		time.Sleep(10 * time.Second)
	}

	s.mu.Lock()
	s.tokens[access] = accessToken{
		User:     ac.User,
		Expires:  now.Add(time.Hour),
		Scenario: ac.Scenario,
	}
	s.mu.Unlock()

	claims := map[string]any{
		"iss":            s.issuer,
		"sub":            ac.User.Sub,
		"aud":            ac.ClientID,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
		"auth_time":      ac.AuthTime.Unix(),
		"email":          ac.User.Email,
		"email_verified": true,
		"name":           ac.User.Name,
	}
	if ac.Nonce != "" {
		claims["nonce"] = ac.Nonce
	}
	if ac.Scenario.MutateClaims != nil {
		ac.Scenario.MutateClaims(claims, now)
	}

	idToken, err := s.signJWT(claims)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        ac.Scope,
		"id_token":     idToken,
	})
}
