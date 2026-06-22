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

	"github.com/manuel/tinyidp/internal/client"
	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// Server holds all IdP state.
type Server struct {
	issuer   string
	clients  *client.Registry

	key *rsa.PrivateKey
	kid string

	registry *scenario.Registry

	mu       sync.Mutex
	codes    map[string]authCode
	tokens   map[string]accessToken
	sessions map[string]*session
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
	AuthTime            time.Time
}

// accessToken is an opaque bearer token mapped to a user + expiry.
type accessToken struct {
	User     user.User
	Expires  time.Time
	Scenario *scenario.Scenario
}

// Options configures a Server at construction time.
type Options struct {
	Issuer   string
	Clients  *client.Registry
}

// New constructs a Server with a freshly generated RSA signing key. If
// opts.Clients is nil, the built-in client registry is used (dev-client,
// public-spa, web-app).
func New(opts Options) (*Server, error) {
	key, err := rsa.GenerateKey(cryptoRandReader, 2048)
	if err != nil {
		return nil, err
	}
	clients := opts.Clients
	if clients == nil {
		clients = client.NewRegistry()
	}
	return &Server{
		issuer:   opts.Issuer,
		clients:  clients,
		key:      key,
		kid:      "dev-key-1",
		registry: scenario.New(),
		codes:    map[string]authCode{},
		tokens:   map[string]accessToken{},
		sessions: map[string]*session{},
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
	s.debugRoutes(mux)
}

// Issuer returns the configured issuer URL.
func (s *Server) Issuer() string { return s.issuer }

// Clients returns the client registry.
func (s *Server) Clients() *client.Registry { return s.clients }
