package server

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// authorizeRequest carries the OAuth/OIDC authorize params across the GET
// (render form) and POST (submit login) steps.
type authorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string

	// Phase 6: OIDC re-authentication / session parameters.
	Prompt    string // space-separated: none, login, consent
	MaxAge    int    // seconds; 0 means unset
	LoginHint string // prefill for the login field
}

// authorize implements the authorization endpoint.
//
//	GET  -> validate params, render the login form (echoing params as hidden
//	         fields so the POST reconstructs the request verbatim)
//	POST -> validate params again, derive a synthetic user from the typed
//	         login, store an auth code, redirect to redirect_uri?code=...
//
// Validation happens in parseAuthorizeRequest on both methods, so a bad
// client_id / redirect_uri / scope never reaches the redirect path.
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ar, err := s.parseAuthorizeRequest(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.authorizeGET(w, r, ar)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		ar, err := s.parseAuthorizeRequest(r.PostForm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		login := user.Normalize(r.PostForm.Get("login"))
		if login == "" {
			http.Error(w, "login is required", http.StatusBadRequest)
			return
		}
		sc, _ := s.registry.Lookup(login)

		// Auth-error scenarios redirect back to the RP with an OAuth error
		// instead of issuing a code (no session is created).
		if sc.AuthError != "" {
			redirectOAuthError(w, r, ar.RedirectURI, ar.State, sc.AuthError, "simulated "+sc.AuthError)
			return
		}

		// Successful login: create an IdP session. The session carries the
		// scenario and AuthTime so that silent re-issuance (prompt=none) and
		// the ID token's auth_time behave correctly.
		sess := newSession(login, sc.User, &sc)
		s.setSessionCookie(w, sess)
		s.issueCodeAndRedirect(w, r, ar, sc.User, &sc, sess.AuthTime)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// authorizeGET implements the OIDC session / re-authentication rules for the
// GET branch of /authorize:
//
//   - prompt=none + no valid session   -> redirect with error=login_required
//   - prompt=none + valid, fresh session -> silently issue a code (no UI)
//   - prompt=login or max_age exceeded    -> show the login form (re-auth)
//   - valid, fresh session (no forced)    -> silently issue a code
//   - no session (normal flow)            -> show the login form
//
// prompt=none forbids any UI, so if re-authentication is required it must
// surface as a login_required error rather than the form.
func (s *Server) authorizeGET(w http.ResponseWriter, r *http.Request, ar authorizeRequest) {
	sess := s.readSession(r)
	forceReauth := promptHas(ar.Prompt, "login") || (sess != nil && !sess.freshEnough(ar.MaxAge))

	if promptHas(ar.Prompt, "none") {
		if sess == nil || forceReauth {
			redirectOAuthError(w, r, ar.RedirectURI, ar.State, "login_required", "simulated login_required (no session or re-auth required)")
			return
		}
		// Valid session and prompt=none: silently issue a code.
		s.issueCodeAndRedirect(w, r, ar, sess.User, sess.Scenario, sess.AuthTime)
		return
	}

	if sess != nil && !forceReauth {
		// Valid session, no forced re-auth: silently issue a code.
		s.issueCodeAndRedirect(w, r, ar, sess.User, sess.Scenario, sess.AuthTime)
		return
	}

	// Show the login form. login_hint prefills the login field.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = loginPage.Execute(w, loginPageData{
		Hidden:    hiddenAuthorizeFields(ar),
		Scenarios: s.scenarioGroups(),
		LoginHint: ar.LoginHint,
	})
}

// parseAuthorizeRequest validates the OAuth/OIDC authorize params from either
// a query string (GET) or a posted form (POST). It is the single validation
// chokepoint: disallowed redirect URIs and unknown clients never reach the
// redirect path (they get a 400 instead, never an open redirect).
func (s *Server) parseAuthorizeRequest(v url.Values) (authorizeRequest, error) {
	ar := authorizeRequest{
		ResponseType:        v.Get("response_type"),
		ClientID:            v.Get("client_id"),
		RedirectURI:         v.Get("redirect_uri"),
		Scope:               v.Get("scope"),
		State:               v.Get("state"),
		Nonce:               v.Get("nonce"),
		CodeChallenge:       v.Get("code_challenge"),
		CodeChallengeMethod: v.Get("code_challenge_method"),
		Prompt:              v.Get("prompt"),
		LoginHint:           v.Get("login_hint"),
	}
	// max_age is optional; parse to seconds. Invalid/unset = 0 (no constraint).
	if ma := v.Get("max_age"); ma != "" {
		if n, err := strconv.Atoi(ma); err == nil {
			ar.MaxAge = n
		}
	}
	if ar.ResponseType != "code" {
		return ar, errText("unsupported response_type")
	}
	c, ok := s.clients.Lookup(ar.ClientID)
	if !ok {
		return ar, errText("unknown client_id")
	}
	if !c.AllowsRedirectURI(ar.RedirectURI) {
		return ar, errText("redirect_uri not allowed for this client")
	}
	if !hasScope(ar.Scope, "openid") {
		return ar, errText("scope must include openid")
	}
	if !c.AllowsScope(ar.Scope) {
		return ar, errText("scope not allowed for this client")
	}
	if c.RequirePKCE && ar.CodeChallenge == "" {
		return ar, errText("this client requires PKCE (code_challenge required)")
	}
	return ar, nil
}

func hiddenAuthorizeFields(ar authorizeRequest) []hiddenField {
	return []hiddenField{
		{"response_type", ar.ResponseType},
		{"client_id", ar.ClientID},
		{"redirect_uri", ar.RedirectURI},
		{"scope", ar.Scope},
		{"state", ar.State},
		{"nonce", ar.Nonce},
		{"code_challenge", ar.CodeChallenge},
		{"code_challenge_method", ar.CodeChallengeMethod},
		// Phase 6: carry prompt/max_age/login_hint through the form so the
		// POST reconstructs the original request verbatim.
		{"prompt", ar.Prompt},
		{"max_age", strconv.Itoa(ar.MaxAge)},
		{"login_hint", ar.LoginHint},
	}
}

// issueCodeAndRedirect stores an auth code for the user + scenario and
// redirects back to the RP with code + state. authTime is the moment the user
// authenticated (from the session or the fresh login); it is carried on the
// code so the token endpoint can set the ID token's auth_time claim to the
// real authentication time rather than the token-issuance time.
func (s *Server) issueCodeAndRedirect(w http.ResponseWriter, r *http.Request, ar authorizeRequest, u user.User, sc *scenario.Scenario, authTime time.Time) {
	code := randomB64(32)

	s.mu.Lock()
	s.codes[code] = authCode{
		ClientID:            ar.ClientID,
		RedirectURI:         ar.RedirectURI,
		Scope:               ar.Scope,
		Nonce:               ar.Nonce,
		CodeChallenge:       ar.CodeChallenge,
		CodeChallengeMethod: ar.CodeChallengeMethod,
		Expires:             time.Now().Add(5 * time.Minute),
		User:                u,
		Scenario:            sc,
		AuthTime:            authTime,
	}
	s.mu.Unlock()

	redirectURL, err := url.Parse(ar.RedirectURI)
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}

	q := redirectURL.Query()
	q.Set("code", code)
	if ar.State != "" {
		q.Set("state", ar.State)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// redirectOAuthError sends an OAuth authorization error back to the RP via
// redirect (error + error_description + state). Used by AuthError scenarios.
func redirectOAuthError(w http.ResponseWriter, r *http.Request, redirectURI, state, code, desc string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("error", code)
	q.Set("error_description", desc)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// errText is a string that satisfies error, used for validation messages.
type errText string

func (e errText) Error() string { return string(e) }
