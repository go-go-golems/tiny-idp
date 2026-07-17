package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/scenario"
	"github.com/go-go-golems/tiny-idp/internal/user"
)

// sessionCookieName is the cookie used to track IdP sessions across
// authorize requests. HttpOnly + SameSite=Lax keeps it out of JavaScript and
// scopes it to top-level navigations, which is the right shape for an OIDC
// IdP session cookie.
const sessionCookieName = "tinyidp_session"

// sessionTTL is how long an IdP session stays valid after the last login.
// A long TTL (24h) suits a test tool: a developer logging in as alice
// repeatedly across a session doesn't have to re-authenticate each time
// unless they ask to (prompt=login) or the RP requests re-auth (max_age).
const sessionTTL = 24 * time.Hour

// session is an authenticated IdP session. It remembers who logged in, the
// scenario they resolved to (so silent re-issuance reproduces the same
// failure behavior), and when authentication happened (auth_time).
type session struct {
	ID       string
	Login    string
	User     user.User
	Scenario *scenario.Scenario
	AuthTime time.Time
	Expires  time.Time
}

// newSession creates a session for a successful login. AuthTime is the moment
// of authentication; Expires is AuthTime + sessionTTL.
func newSession(login string, u user.User, sc *scenario.Scenario) *session {
	now := time.Now()
	return &session{
		ID:       randomB64(32),
		Login:    login,
		User:     u,
		Scenario: sc,
		AuthTime: now,
		Expires:  now.Add(sessionTTL),
	}
}

// readSession looks up the session cookie in the request and returns the
// matching non-expired session, or nil. Expired sessions are treated as
// absent so a stale cookie triggers a normal login.
func (s *Server) readSession(r *http.Request) *session {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[c.Value]
	if !ok || time.Now().After(sess.Expires) {
		return nil
	}
	return sess
}

// setSessionCookie stores the session and writes the cookie to the response.
// The cookie is HttpOnly and SameSite=Lax; it is not Secure because the mock
// IdP serves plain HTTP on loopback (a Secure cookie would not be sent over
// http://localhost).
func (s *Server) setSessionCookie(w http.ResponseWriter, sess *session) {
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// freshEnough reports whether the session's auth_time is within maxAge
// seconds of now. A maxAge <= 0 means "no max_age constraint", so any
// non-expired session is fresh enough.
func (sess *session) freshEnough(maxAge int) bool {
	if maxAge <= 0 {
		return true
	}
	return time.Since(sess.AuthTime) <= time.Duration(maxAge)*time.Second
}

// promptHas reports whether the space-separated prompt parameter contains
// the given value. OIDC prompt is a space-delimited list (e.g. "login none"),
// so a substring check would false-match.
func promptHas(prompt, want string) bool {
	for _, p := range strings.Fields(prompt) {
		if p == want {
			return true
		}
	}
	return false
}
