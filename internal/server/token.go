package server

import (
	"net/http"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// refreshTokenTTL is how long a refresh token stays valid. Longer than the
// access token (1h) so an RP can renew across a session without re-login.
const refreshTokenTTL = 24 * time.Hour

// token implements the token endpoint. It supports two grant types:
//
//   - authorization_code: exchange a one-time code for ID + access tokens
//     (and a refresh token when the scope includes offline_access).
//   - refresh_token: exchange a refresh token for a new access + refresh
//     token pair (rotation). Reuse of a rotated refresh token is rejected.
//
// Client auth (client_secret_basic / client_secret_post) is shared across
// both grants.
func (s *Server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
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
	// Confidential clients must present their secret; public clients skip.
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

	switch r.Form.Get("grant_type") {
	case "authorization_code":
		s.tokenAuthorizationCode(w, r, clientID, c)
	case "refresh_token":
		s.tokenRefresh(w, r, clientID, c)
	default:
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code and refresh_token are supported")
	}
}

// tokenAuthorizationCode exchanges a one-time authorization code for ID +
// access tokens. If the scope includes offline_access, a refresh token is
// also issued.
func (s *Server) tokenAuthorizationCode(w http.ResponseWriter, r *http.Request, clientID string, _ interface{}) {
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

	access := randomB64(32)
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
	// Phase 7: merge the scenario's declarative extra claims (groups, roles,
	// tenant, etc.) before MutateClaims so a mutator can still override.
	for k, v := range ac.Scenario.ExtraClaims {
		claims[k] = v
	}
	// Phase 7: omit claims the scenario marks as absent (e.g. no-email).
	for _, k := range ac.Scenario.OmitClaims {
		delete(claims, k)
	}
	if ac.Scenario.MutateClaims != nil {
		ac.Scenario.MutateClaims(claims, now)
	}

	idToken, err := s.signJWT(claims)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
		return
	}

	resp := map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        ac.Scope,
		"id_token":     idToken,
	}

	// Phase 9: issue a refresh token when the RP requested offline_access.
	if hasScope(ac.Scope, "offline_access") {
		rt := s.issueRefreshToken(ac.User, ac.Scenario, ac.ClientID, ac.Scope, now)
		resp["refresh_token"] = rt
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

// tokenRefresh exchanges a refresh token for a new access + refresh token
// pair. Rotation: the old refresh token is deleted, and a new one is issued.
// Reuse of a rotated (deleted) token fails with invalid_grant, which is the
// standard OAuth refresh-token-rotation reuse signal.
func (s *Server) tokenRefresh(w http.ResponseWriter, r *http.Request, clientID string, _ interface{}) {
	rt := r.Form.Get("refresh_token")

	s.mu.Lock()
	rtok, ok := s.refreshTokens[rt]
	// Rotation: delete the presented token so it cannot be reused. If it was
	// already rotated (absent), this is a reuse attempt.
	delete(s.refreshTokens, rt)
	s.mu.Unlock()

	if !ok {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown refresh token (rotated, revoked, or never issued)")
		return
	}
	if time.Now().After(rtok.Expires) {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "expired refresh token")
		return
	}
	if rtok.ClientID != clientID {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
		return
	}

	now := time.Now()
	access := randomB64(32)
	s.mu.Lock()
	s.tokens[access] = accessToken{
		User:     rtok.User,
		Expires:  now.Add(time.Hour),
		Scenario: rtok.Scenario,
	}
	s.mu.Unlock()

	// Issue a rotated refresh token.
	newRT := s.issueRefreshToken(rtok.User, rtok.Scenario, rtok.ClientID, rtok.Scope, now)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"scope":         rtok.Scope,
		"refresh_token": newRT,
	})
}

// issueRefreshToken stores a new refresh token and returns its opaque value.
func (s *Server) issueRefreshToken(u user.User, sc *scenario.Scenario, clientID, scope string, now time.Time) string {
	rt := randomB64(32)
	s.mu.Lock()
	s.refreshTokens[rt] = refreshToken{
		User:     u,
		Scenario: sc,
		ClientID: clientID,
		Scope:    scope,
		Expires:  now.Add(refreshTokenTTL),
	}
	s.mu.Unlock()
	return rt
}
