package server

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
)

// debugRoutes registers the /debug/* introspection endpoints. These are
// read-only views of in-memory state (sessions, codes, tokens) plus a reset
// endpoint, intended for manual debugging of flows. They are guarded to
// loopback: a request whose RemoteAddr is not 127.0.0.1/::1 is rejected with
// 403, so exposing the server to a LAN (OIDC_ADDR=0.0.0.0:...) does not also
// expose debug state.
func (s *Server) debugRoutesAt(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/debug", s.debugIndex)
	mux.HandleFunc(prefix+"/debug/sessions", s.debugSessions)
	mux.HandleFunc(prefix+"/debug/codes", s.debugCodes)
	mux.HandleFunc(prefix+"/debug/tokens", s.debugTokens)
	mux.HandleFunc(prefix+"/debug/device-grants", s.debugDeviceGrants)
	mux.HandleFunc(prefix+"/debug/jwks-mode", s.debugJWKSMode)
	mux.HandleFunc(prefix+"/debug/reset", s.debugReset)
}

// requireLoopback rejects non-loopback requests. Returns true if the request
// may proceed.
func requireLoopback(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		http.Error(w, "debug endpoints are loopback-only", http.StatusForbidden)
		return false
	}
	return true
}

// debugEntry is one row in a debug listing. Prefix is the first 8 chars of the
// secret (code/token/session id) — enough to identify it in a flow log without
// exposing the full secret in a listing.
type debugEntry struct {
	Prefix    string `json:"prefix"`
	Expires   string `json:"expires"`
	ExpiresIn int    `json:"expires_in_sec"`
}

type debugSessionEntry struct {
	debugEntry
	Login    string `json:"login"`
	Sub      string `json:"sub"`
	AuthTime string `json:"auth_time"`
}

type debugCodeEntry struct {
	debugEntry
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Sub         string `json:"sub"`
	Scenario    string `json:"scenario"`
}

type debugTokenEntry struct {
	debugEntry
	Sub      string `json:"sub"`
	Scenario string `json:"scenario"`
}

type debugDeviceGrantEntry struct {
	debugEntry
	UserCode      string `json:"user_code"`
	ClientID      string `json:"client_id"`
	Scope         string `json:"scope"`
	Status        string `json:"status"`
	Sub           string `json:"sub,omitempty"`
	Scenario      string `json:"scenario,omitempty"`
	LastPoll      string `json:"last_poll,omitempty"`
	SlowDownCount int    `json:"slow_down_count"`
}

func (s *Server) debugIndex(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	s.mu.Lock()
	counts := map[string]int{
		"sessions":      len(s.sessions),
		"codes":         len(s.codes),
		"tokens":        len(s.tokens),
		"device_grants": len(s.deviceGrants),
	}
	mode := s.jwksMode
	if mode == "" {
		mode = "normal"
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"endpoints": map[string]string{
			"debug":               "this index",
			"debug/sessions":      "list active IdP sessions",
			"debug/codes":         "list outstanding authorization codes",
			"debug/tokens":        "list issued access tokens",
			"debug/device-grants": "list pending/approved/denied device grants",
			"debug/jwks-mode":     "GET current /jwks mode; POST {\"mode\":\"normal|500|slow|empty\"} to set",
			"debug/reset":         "POST to clear all sessions/codes/tokens/refresh-tokens/device-grants",
		},
		"counts":    counts,
		"jwks_mode": mode,
	})
}

func (s *Server) debugSessions(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	s.mu.Lock()
	out := make([]debugSessionEntry, 0, len(s.sessions))
	for id, sess := range s.sessions {
		out = append(out, debugSessionEntry{
			debugEntry: debugEntry{
				Prefix:    prefix(id, 8),
				Expires:   sess.Expires.Format(time.RFC3339),
				ExpiresIn: int(time.Until(sess.Expires).Seconds()),
			},
			Login:    sess.Login,
			Sub:      sess.User.Sub,
			AuthTime: sess.AuthTime.Format(time.RFC3339),
		})
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) debugCodes(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	s.mu.Lock()
	out := make([]debugCodeEntry, 0, len(s.codes))
	for code, ac := range s.codes {
		out = append(out, debugCodeEntry{
			debugEntry: debugEntry{
				Prefix:    prefix(code, 8),
				Expires:   ac.Expires.Format(time.RFC3339),
				ExpiresIn: int(time.Until(ac.Expires).Seconds()),
			},
			ClientID:    ac.ClientID,
			RedirectURI: ac.RedirectURI,
			Sub:         ac.User.Sub,
			Scenario:    scenarioName(ac.Scenario),
		})
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) debugTokens(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	s.mu.Lock()
	out := make([]debugTokenEntry, 0, len(s.tokens))
	for tok, at := range s.tokens {
		out = append(out, debugTokenEntry{
			debugEntry: debugEntry{
				Prefix:    prefix(tok, 8),
				Expires:   at.Expires.Format(time.RFC3339),
				ExpiresIn: int(time.Until(at.Expires).Seconds()),
			},
			Sub:      at.User.Sub,
			Scenario: scenarioName(at.Scenario),
		})
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) debugDeviceGrants(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	s.mu.Lock()
	out := make([]debugDeviceGrantEntry, 0, len(s.deviceGrants))
	for code, grant := range s.deviceGrants {
		entry := debugDeviceGrantEntry{
			debugEntry: debugEntry{
				Prefix:    prefix(code, 8),
				Expires:   grant.Expires.Format(time.RFC3339),
				ExpiresIn: int(time.Until(grant.Expires).Seconds()),
			},
			UserCode:      grant.UserCode,
			ClientID:      grant.ClientID,
			Scope:         grant.Scope,
			Status:        string(grant.Status),
			Sub:           grant.User.Sub,
			Scenario:      scenarioName(grant.Scenario),
			SlowDownCount: grant.SlowDownCount,
		}
		if !grant.LastPoll.IsZero() {
			entry.LastPoll = grant.LastPoll.Format(time.RFC3339)
		}
		out = append(out, entry)
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

// debugJWKSMode gets or sets the /jwks failure mode (Phase 10). GET returns
// the current mode; POST with JSON {"mode":"normal|500|slow|empty"} sets it.
func (s *Server) debugJWKSMode(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"mode": s.JWKSMode()})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed (GET to read, POST to set)", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	switch body.Mode {
	case "normal", "500", "slow", "empty":
	default:
		http.Error(w, "unknown mode (want normal|500|slow|empty)", http.StatusBadRequest)
		return
	}
	s.SetJWKSMode(body.Mode)
	writeJSON(w, http.StatusOK, map[string]any{"mode": body.Mode})
}

// debugReset clears all in-memory sessions, codes, and tokens. POST-only so a
// stray GET (e.g. a browser prefetch) cannot wipe state.
func (s *Server) debugReset(w http.ResponseWriter, r *http.Request) {
	if !requireLoopback(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed (POST to reset)", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	n := len(s.sessions) + len(s.codes) + len(s.tokens) + len(s.refreshTokens) + len(s.deviceGrants) + len(s.dpopReplay)
	s.sessions = map[string]*session{}
	s.codes = map[string]authCode{}
	s.tokens = map[string]accessToken{}
	s.refreshTokens = map[string]refreshToken{}
	s.deviceGrants = map[string]deviceGrant{}
	s.dpopReplay = map[string]time.Time{}
	s.jwksMode = "normal"
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"reset": true,
		"count": n,
	})
}

// scenarioName returns the scenario's Name, or "" for a nil scenario (the
// nil case should not occur in practice but is guarded).
func scenarioName(sc *scenario.Scenario) string {
	if sc == nil {
		return ""
	}
	return sc.Name
}

// prefix returns the first n chars of s, or s if shorter.
func prefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
