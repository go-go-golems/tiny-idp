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
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/manuel/tinyidp/internal/client"
	"github.com/manuel/tinyidp/internal/scenario"
	"github.com/manuel/tinyidp/internal/user"
)

// Server holds all IdP state.
type Server struct {
	issuer  string
	clients *client.Registry

	key *rsa.PrivateKey
	kid string

	// jwksMode selects a failure mode for the /jwks endpoint (Phase 10):
	// "normal" (default), "500", "slow", or "empty". JWKS failures are
	// server-level (not per-user) because /jwks is global and fetched
	// independently of any login.
	jwksMode string

	registry *scenario.Registry

	mu            sync.Mutex
	codes         map[string]authCode
	tokens        map[string]accessToken
	sessions      map[string]*session
	refreshTokens map[string]refreshToken
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

// refreshToken is an opaque token mapped to a user + expiry, used to obtain
// new access tokens without re-authentication. Rotation deletes the
// presented refresh token and issues a fresh one.
type refreshToken struct {
	User     user.User
	Scenario *scenario.Scenario
	ClientID string
	Scope    string
	Expires  time.Time
}

// Options configures a Server at construction time.
type Options struct {
	Issuer   string
	Clients  *client.Registry
	Registry *scenario.Registry
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
	registry := opts.Registry
	if registry == nil {
		registry = scenario.New()
	}
	return &Server{
		issuer:        opts.Issuer,
		clients:       clients,
		key:           key,
		kid:           "dev-key-1",
		registry:      registry,
		codes:         map[string]authCode{},
		tokens:        map[string]accessToken{},
		sessions:      map[string]*session{},
		refreshTokens: map[string]refreshToken{},
	}, nil
}

// RegisterRoutes wires all IdP handlers onto the given mux. Tests use this to
// mount the server on an httptest.Server without ListenAndServe.
//
// When the configured issuer has a path component, routes are registered both
// at the root paths and at the issuer path prefix. For example, issuer
// "http://127.0.0.1:5556/realms/demo" serves discovery at both
// "/.well-known/openid-configuration" and
// "/realms/demo/.well-known/openid-configuration". This keeps the simple root
// issuer workflow while allowing Keycloak-shaped realm issuer URLs in tests.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	s.registerRoutesAt(mux, "")
	if prefix := s.issuerPathPrefix(); prefix != "" {
		s.registerRoutesAt(mux, prefix)
	}
}

func (s *Server) registerRoutesAt(mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/.well-known/openid-configuration", s.discovery)
	mux.HandleFunc(prefix+"/jwks", s.jwks)
	mux.HandleFunc(prefix+"/authorize", s.authorize)
	mux.HandleFunc(prefix+"/token", s.token)
	mux.HandleFunc(prefix+"/userinfo", s.userinfo)
	mux.HandleFunc(prefix+"/end-session", s.endSession)
	mux.HandleFunc(prefix+"/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	s.debugRoutesAt(mux, prefix)
}

func (s *Server) issuerPathPrefix() string {
	u, err := url.Parse(s.issuer)
	if err != nil {
		return ""
	}
	prefix := strings.TrimRight(u.EscapedPath(), "/")
	if prefix == "/" {
		return ""
	}
	return prefix
}

// Issuer returns the configured issuer URL.
func (s *Server) Issuer() string { return s.issuer }

// Clients returns the client registry.
func (s *Server) Clients() *client.Registry { return s.clients }

// SetJWKSMode sets a failure mode for the /jwks endpoint (Phase 10). Valid
// values are "normal", "500", "slow", and "empty". It is intended for test
// setup and the debug UI; the value is not part of the serialized config.
func (s *Server) SetJWKSMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jwksMode = mode
}

// JWKSMode returns the current /jwks failure mode.
func (s *Server) JWKSMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.jwksMode == "" {
		return "normal"
	}
	return s.jwksMode
}
