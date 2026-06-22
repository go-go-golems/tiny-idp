package server

import (
	"net/http"
	"net/url"
	"time"

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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = loginPage.Execute(w, loginPageData{
			Hidden: hiddenAuthorizeFields(ar),
			// Scenarios left nil until Phase 3 wires the registry in.
		})

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
		u := user.FromLogin(login)
		s.issueCodeAndRedirect(w, r, ar, u)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
	}
	if ar.ResponseType != "code" {
		return ar, errText("unsupported response_type")
	}
	if ar.ClientID != s.clientID {
		return ar, errText("unknown client_id")
	}
	if !s.redirectURIs[ar.RedirectURI] {
		return ar, errText("redirect_uri not allowed; set OIDC_REDIRECT_URIS")
	}
	if !hasScope(ar.Scope, "openid") {
		return ar, errText("scope must include openid")
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
	}
}

// issueCodeAndRedirect stores an auth code for the user and redirects back to
// the RP with code + state.
func (s *Server) issueCodeAndRedirect(w http.ResponseWriter, r *http.Request, ar authorizeRequest, u user.User) {
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

// errText is a string that satisfies error, used for validation messages.
type errText string

func (e errText) Error() string { return string(e) }
