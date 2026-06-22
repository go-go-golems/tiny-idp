package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// endSession implements RP-initiated logout (OIDC Session Management 1.0,
// RP-Initiated Logout). The RP redirects the user here with optional
// parameters:
//
//   - id_token_hint:           a previously issued ID token identifying the
//     session to end. Treated as a hint: the payload
//     is decoded (signature not re-verified) to find
//     the subject, and any IdP session for that
//     subject is deleted.
//   - post_logout_redirect_uri: a URI to redirect to after logout. Must be
//     registered for the client (see client.Allow
//     sPostLogoutRedirectURI). If absent, a simple
//     "logged out" page is shown.
//   - client_id:               the RP's client_id. When present, the
//     post_logout_redirect_uri is validated against
//     that client only; otherwise against all
//     registered clients.
//   - state:                   an opaque value forwarded back to the
//     post_logout_redirect_uri as a query parameter.
//
// The IdP session cookie is always cleared on the response, regardless of
// whether a hint matched. This means a logout without a hint still ends the
// caller's own session (the cookie they sent).
func (s *Server) endSession(w http.ResponseWriter, r *http.Request) {
	postLogout := r.FormValue("post_logout_redirect_uri")
	state := r.FormValue("state")
	clientID := r.FormValue("client_id")
	hint := r.FormValue("id_token_hint")

	// 1. Validate post_logout_redirect_uri against the client registry.
	if postLogout != "" {
		if !s.postLogoutAllowed(clientID, postLogout) {
			http.Error(w, "post_logout_redirect_uri not registered for this client", http.StatusBadRequest)
			return
		}
	}

	// 2. End the session. An id_token_hint deletes any session whose subject
	//    matches the hint; without a hint, the caller's own session (from the
	//    cookie) is deleted. The cookie is always cleared on the response.
	if hint != "" {
		if sub := subFromIDTokenHint(hint); sub != "" {
			s.deleteSessionsBySub(sub)
		}
	} else if sess := s.readSession(r); sess != nil {
		s.deleteSession(sess.ID)
	}
	clearSessionCookie(w)

	// 3. Redirect or show the logged-out page.
	if postLogout != "" {
		target := postLogout
		if state != "" {
			target = withQuery(target, "state", state)
		}
		http.Redirect(w, r, target, http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(loggedOutHTML))
}

// postLogoutAllowed reports whether postLogout is registered for the given
// client (if clientID is non-empty) or for any registered client.
func (s *Server) postLogoutAllowed(clientID, postLogout string) bool {
	if clientID != "" {
		c, ok := s.clients.Lookup(clientID)
		return ok && c.AllowsPostLogoutRedirectURI(postLogout)
	}
	for _, c := range s.clients.All() {
		if c.AllowsPostLogoutRedirectURI(postLogout) {
			return true
		}
	}
	return false
}

// deleteSession removes a single session by id.
func (s *Server) deleteSession(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// deleteSessionsBySub removes every session whose subject matches sub. This
// is how an id_token_hint ends a session identified by subject rather than by
// cookie: a hint issued in one browser/tab may be presented in another.
func (s *Server) deleteSessionsBySub(sub string) {
	s.mu.Lock()
	for id, sess := range s.sessions {
		if sess.User.Sub == sub {
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()
}

// clearSessionCookie expires the session cookie on the client by setting a
// MaxAge < 0. The cookie value is emptied so the browser drops it.
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// subFromIDTokenHint decodes the ID token payload (without verifying the
// signature) and returns its "sub" claim. The hint is advisory; the IdP is
// not required to verify it. An unparseable token yields an empty string.
func subFromIDTokenHint(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

// withQuery appends a key=value query parameter to rawURL, preserving any
// existing query string. It is used to forward `state` on a post-logout
// redirect.
func withQuery(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

// loggedOutHTML is the page shown when logout completes without a
// post_logout_redirect_uri. It is minimal and intentionally unstyled.
const loggedOutHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Logged out</title></head>
<body>
<h1>Logged out</h1>
<p>Your tinyidp session has ended.</p>
</body>
</html>`
