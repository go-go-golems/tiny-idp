// Package server implements the HTTP layer of the mock OIDC IdP.
//
// It is split across one file per endpoint (discovery, jwks, authorize,
// token, userinfo) plus shared helpers (jwt signing/PKCE, response writers,
// CORS). The login page is an embedded HTML template (static/login.html)
// rendered via html/template.
//
// All state is in-memory and per-process; the RSA signing key is generated
// at construction time, so a restart invalidates outstanding codes/tokens
// and rotates JWKS. This is intentional for a test tool.
package server

import (
	"crypto/rsa"
	"net/http"
	"sync"
	"time"

	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// Server holds all IdP state.
type Server struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURIs map[string]bool

	key *rsa.PrivateKey
	kid string

	registry *scenario.Registry

	mu     sync.Mutex
	codes  map[string]authCode
	tokens map[string]accessToken
}

// authCode is a one-time authorization code awaiting exchange at /token.
type authCode struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Expires             time.Time
	User                user.User
	Scenario            *scenario.Scenario
}

// accessToken is an opaque bearer token mapped to a user + expiry.
type accessToken struct {
	User     user.User
	Expires  time.Time
	Scenario *scenario.Scenario
}

// Options configures a Server at construction time.
type Options struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURIs []string
}

// New constructs a Server with a freshly generated RSA signing key.
func New(opts Options) (*Server, error) {
	key, err := rsa.GenerateKey(cryptoRandReader, 2048)
	if err != nil {
		return nil, err
	}
	redirs := make(map[string]bool, len(opts.RedirectURIs))
	for _, u := range opts.RedirectURIs {
		if u != "" {
			redirs[u] = true
		}
	}
	return &Server{
		issuer:       opts.Issuer,
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		redirectURIs: redirs,
		key:          key,
		kid:          "dev-key-1",
		registry:     scenario.New(),
		codes:        map[string]authCode{},
		tokens:       map[string]accessToken{},
	}, nil
}

// RegisterRoutes wires all IdP handlers onto the given mux. Tests use this to
// mount the server on an httptest.Server without ListenAndServe.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/openid-configuration", s.discovery)
	mux.HandleFunc("/jwks", s.jwks)
	mux.HandleFunc("/authorize", s.authorize)
	mux.HandleFunc("/token", s.token)
	mux.HandleFunc("/userinfo", s.userinfo)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
}

// Issuer returns the configured issuer URL.
func (s *Server) Issuer() string { return s.issuer }

// ClientID returns the configured client ID.
func (s *Server) ClientID() string { return s.clientID }

// Registry returns the scenario registry (used by tests and, in Phase 3, by
// the login page to list selectable scenarios).
func (s *Server) Registry() *scenario.Registry { return s.registry }
