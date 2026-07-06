package server

import (
	"net/http"
	"time"

	"github.com/manuel/tinyidp/internal/client"
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

	clientID, c, ok := s.authenticateOAuthClient(w, r)
	if !ok {
		return
	}

	switch r.Form.Get("grant_type") {
	case "authorization_code":
		s.tokenAuthorizationCode(w, r, clientID, c)
	case "refresh_token":
		s.tokenRefresh(w, r, clientID, c)
	case deviceGrantType:
		s.tokenDeviceCode(w, r, clientID)
	default:
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code, refresh_token, and device_code are supported")
	}
}

func (s *Server) authenticateOAuthClient(w http.ResponseWriter, r *http.Request) (string, client.Client, bool) {
	clientID, basicSecret, hasBasic := r.BasicAuth()
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	c, ok := s.clients.Lookup(clientID)
	if !ok {
		tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_id")
		return "", client.Client{}, false
	}
	// Confidential clients must present their secret; public clients skip.
	if c.Secret != "" {
		secret := r.Form.Get("client_secret")
		if hasBasic {
			secret = basicSecret
		}
		if secret != c.Secret {
			tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_secret")
			return "", client.Client{}, false
		}
	}
	return clientID, c, true
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

	proof, ok := s.dpopProofForTokenRequest(w, r)
	if !ok {
		return
	}

	access := s.issueAccessToken(ac.User, ac.Scenario, now, proof.JKT)

	idToken, err := s.issueIDToken(ac.User, ac.Scenario, ac.ClientID, ac.Nonce, ac.AuthTime, now)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
		return
	}

	resp := map[string]any{
		"access_token": access,
		"token_type":   tokenTypeForJKT(proof.JKT),
		"expires_in":   3600,
		"scope":        ac.Scope,
		"id_token":     idToken,
	}

	// Phase 9: issue a refresh token when the RP requested offline_access.
	if hasScope(ac.Scope, "offline_access") {
		rt := s.issueRefreshToken(ac.User, ac.Scenario, ac.ClientID, ac.Scope, now, proof.JKT)
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

	proof, ok := s.dpopProofForTokenRequest(w, r)
	if !ok {
		return
	}
	newDPoPJKT := proof.JKT
	if rtok.DPoPJKT != "" {
		if proof.JKT == "" {
			tokenError(w, http.StatusBadRequest, "invalid_dpop_proof", "refresh token requires DPoP proof")
			return
		}
		if proof.JKT != rtok.DPoPJKT {
			tokenError(w, http.StatusBadRequest, "invalid_dpop_proof", "DPoP proof key does not match refresh token")
			return
		}
		newDPoPJKT = rtok.DPoPJKT
	}

	// Rotation: delete the presented token so it cannot be reused. If it was
	// already rotated (absent), this is a reuse attempt.
	s.mu.Lock()
	latest, ok := s.refreshTokens[rt]
	if ok {
		delete(s.refreshTokens, rt)
		rtok = latest
	}
	s.mu.Unlock()
	if !ok {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown refresh token (rotated, revoked, or never issued)")
		return
	}

	now := time.Now()
	access := s.issueAccessToken(rtok.User, rtok.Scenario, now, newDPoPJKT)

	// Issue a rotated refresh token.
	newRT := s.issueRefreshToken(rtok.User, rtok.Scenario, rtok.ClientID, rtok.Scope, now, newDPoPJKT)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"token_type":    tokenTypeForJKT(newDPoPJKT),
		"expires_in":    3600,
		"scope":         rtok.Scope,
		"refresh_token": newRT,
	})
}

func (s *Server) tokenDeviceCode(w http.ResponseWriter, r *http.Request, clientID string) {
	deviceCode := r.Form.Get("device_code")
	now := time.Now()

	s.mu.Lock()
	grant, ok := s.deviceGrants[deviceCode]
	s.mu.Unlock()

	if !ok {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "unknown device code")
		return
	}
	if now.After(grant.Expires) {
		tokenError(w, http.StatusBadRequest, "expired_token", "device code expired")
		return
	}
	if grant.ClientID != clientID {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "client_id mismatch")
		return
	}

	s.mu.Lock()
	grant = s.deviceGrants[deviceCode]
	if !grant.LastPoll.IsZero() && now.Sub(grant.LastPoll) < grant.Interval {
		grant.SlowDownCount++
		grant.Interval += 5 * time.Second
		grant.LastPoll = now
		s.deviceGrants[deviceCode] = grant
		s.mu.Unlock()
		tokenError(w, http.StatusBadRequest, "slow_down", "polling too quickly")
		return
	}
	grant.LastPoll = now
	s.deviceGrants[deviceCode] = grant
	s.mu.Unlock()

	switch grant.Status {
	case devicePending:
		tokenError(w, http.StatusBadRequest, "authorization_pending", "device authorization is pending")
		return
	case deviceDenied:
		tokenError(w, http.StatusBadRequest, "access_denied", "device authorization denied")
		return
	case deviceApproved:
		// Continue below.
	default:
		tokenError(w, http.StatusBadRequest, "invalid_grant", "invalid device grant state")
		return
	}

	proof, ok := s.dpopProofForTokenRequest(w, r)
	if !ok {
		return
	}

	s.mu.Lock()
	latest, ok := s.deviceGrants[deviceCode]
	if ok && latest.Status == deviceApproved && latest.ClientID == clientID {
		delete(s.deviceGrants, deviceCode)
		grant = latest
	}
	s.mu.Unlock()
	if !ok || grant.Status != deviceApproved {
		tokenError(w, http.StatusBadRequest, "invalid_grant", "device grant already used")
		return
	}

	access := s.issueAccessToken(grant.User, grant.Scenario, now, proof.JKT)
	resp := map[string]any{
		"access_token": access,
		"token_type":   tokenTypeForJKT(proof.JKT),
		"expires_in":   3600,
		"scope":        grant.Scope,
	}
	if hasScope(grant.Scope, "openid") {
		idToken, err := s.issueIDToken(grant.User, grant.Scenario, grant.ClientID, "", grant.AuthTime, now)
		if err != nil {
			tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
			return
		}
		resp["id_token"] = idToken
	}
	if hasScope(grant.Scope, "offline_access") {
		resp["refresh_token"] = s.issueRefreshToken(grant.User, grant.Scenario, grant.ClientID, grant.Scope, now, proof.JKT)
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) issueAccessToken(u user.User, sc *scenario.Scenario, now time.Time, dpopJKT string) string {
	access := randomB64(32)
	s.mu.Lock()
	s.tokens[access] = accessToken{
		User:     u,
		Expires:  now.Add(time.Hour),
		Scenario: sc,
		DPoPJKT:  dpopJKT,
	}
	s.mu.Unlock()
	return access
}

func (s *Server) issueIDToken(u user.User, sc *scenario.Scenario, clientID, nonce string, authTime, now time.Time) (string, error) {
	claims := map[string]any{
		"iss":            s.issuer,
		"sub":            u.Sub,
		"aud":            clientID,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
		"auth_time":      authTime.Unix(),
		"email":          u.Email,
		"email_verified": true,
		"name":           u.Name,
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	for k, v := range sc.ExtraClaims {
		claims[k] = v
	}
	for _, k := range sc.OmitClaims {
		delete(claims, k)
	}
	if sc.MutateClaims != nil {
		sc.MutateClaims(claims, now)
	}
	return s.signJWT(claims, sc)
}

// issueRefreshToken stores a new refresh token and returns its opaque value.
func (s *Server) issueRefreshToken(u user.User, sc *scenario.Scenario, clientID, scope string, now time.Time, dpopJKT string) string {
	rt := randomB64(32)
	s.mu.Lock()
	s.refreshTokens[rt] = refreshToken{
		User:     u,
		Scenario: sc,
		ClientID: clientID,
		Scope:    scope,
		Expires:  now.Add(refreshTokenTTL),
		DPoPJKT:  dpopJKT,
	}
	s.mu.Unlock()
	return rt
}
