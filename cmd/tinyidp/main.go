// Package main implements tinyidp, a minimal mock OpenID Connect Identity
// Provider for local development and integration testing. It is NOT production
// grade: see docs/ttmp design doc MOCK-OIDC-IDP.
//
// Phase 0 scope: baseline OIDC happy path (discovery, JWKS, authorize, token,
// userinfo) with a single fixed client and a single fixed user. Multiple
// synthetic users, scenarios, and a login page arrive in later phases.
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// server holds all IdP state. Everything is in-memory and per-process; the
// signing key is generated at startup, so a restart invalidates outstanding
// codes/tokens and rotates JWKS (acceptable for a test tool).
type server struct {
	issuer       string
	clientID     string
	clientSecret string
	redirectURIs map[string]bool

	key *rsa.PrivateKey
	kid string

	mu     sync.Mutex
	codes  map[string]authCode
	tokens map[string]accessToken

	user user
}

type authCode struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Expires             time.Time
	User                user
}

type accessToken struct {
	User    user
	Expires time.Time
}

type user struct {
	Sub   string
	Email string
	Name  string
}

func main() {
	issuer := strings.TrimRight(env("OIDC_ISSUER", "http://localhost:5556"), "/")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}

	s := &server{
		issuer:       issuer,
		clientID:     env("OIDC_CLIENT_ID", "dev-client"),
		clientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		redirectURIs: parseCSV(env("OIDC_REDIRECT_URIS", "http://localhost:3000/callback,http://127.0.0.1:3000/callback")),
		key:          key,
		kid:          "dev-key-1",
		codes:        map[string]authCode{},
		tokens:       map[string]accessToken{},
		user: user{
			Sub:   env("OIDC_USER_SUB", "user-123"),
			Email: env("OIDC_USER_EMAIL", "dev@example.test"),
			Name:  env("OIDC_USER_NAME", "Dev User"),
		},
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := env("OIDC_ADDR", "127.0.0.1:5556")
	log.Printf("tinyidp listening on %s; issuer=%s client_id=%s", addr, s.issuer, s.clientID)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

// registerRoutes wires all IdP handlers onto the given mux. Extracted from main
// so tests can mount the server on an httptest.Server without ListenAndServe.
func (s *server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/openid-configuration", s.discovery)
	mux.HandleFunc("/jwks", s.jwks)
	mux.HandleFunc("/authorize", s.authorize)
	mux.HandleFunc("/token", s.token)
	mux.HandleFunc("/userinfo", s.userinfo)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
}

func (s *server) discovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/authorize",
		"token_endpoint":                        s.issuer + "/token",
		"userinfo_endpoint":                     s.issuer + "/userinfo",
		"jwks_uri":                              s.issuer + "/jwks",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "email", "email_verified", "name"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
	})
}

func (s *server) jwks(w http.ResponseWriter, r *http.Request) {
	pub := s.key.PublicKey
	e := big.NewInt(int64(pub.E)).Bytes()

	writeJSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"use": "sig",
			"kid": s.kid,
			"alg": "RS256",
			"n":   b64(pub.N.Bytes()),
			"e":   b64(e),
		}},
	})
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")

	if q.Get("response_type") != "code" {
		http.Error(w, "unsupported response_type", http.StatusBadRequest)
		return
	}
	if clientID != s.clientID {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	if !s.redirectURIs[redirectURI] {
		http.Error(w, "redirect_uri not allowed; set OIDC_REDIRECT_URIS", http.StatusBadRequest)
		return
	}
	if !hasScope(q.Get("scope"), "openid") {
		http.Error(w, "scope must include openid", http.StatusBadRequest)
		return
	}

	code := randomB64(32)

	s.mu.Lock()
	s.codes[code] = authCode{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               q.Get("scope"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		Expires:             time.Now().Add(5 * time.Minute),
		User:                s.user,
	}
	s.mu.Unlock()

	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}

	v := u.Query()
	v.Set("code", code)
	if state := q.Get("state"); state != "" {
		v.Set("state", state)
	}
	u.RawQuery = v.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		tokenError(w, http.StatusMethodNotAllowed, "invalid_request", "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		tokenError(w, http.StatusBadRequest, "invalid_request", "invalid form")
		return
	}
	if r.Form.Get("grant_type") != "authorization_code" {
		tokenError(w, http.StatusBadRequest, "unsupported_grant_type", "only authorization_code is supported")
		return
	}

	clientID, basicSecret, hasBasic := r.BasicAuth()
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	if clientID != s.clientID {
		tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_id")
		return
	}

	if s.clientSecret != "" {
		secret := r.Form.Get("client_secret")
		if hasBasic {
			secret = basicSecret
		}
		if secret != s.clientSecret {
			tokenError(w, http.StatusUnauthorized, "invalid_client", "bad client_secret")
			return
		}
	}

	code := r.Form.Get("code")

	// Pop the code atomically under the lock: authorization codes are
	// one-time use, so read+delete must be a single critical section.
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
	access := randomB64(32)

	s.mu.Lock()
	s.tokens[access] = accessToken{
		User:    ac.User,
		Expires: now.Add(time.Hour),
	}
	s.mu.Unlock()

	claims := map[string]any{
		"iss":            s.issuer,
		"sub":            ac.User.Sub,
		"aud":            ac.ClientID,
		"exp":            now.Add(time.Hour).Unix(),
		"iat":            now.Unix(),
		"auth_time":      now.Unix(),
		"email":          ac.User.Email,
		"email_verified": true,
		"name":           ac.User.Name,
	}
	if ac.Nonce != "" {
		claims["nonce"] = ac.Nonce
	}

	idToken, err := s.signJWT(claims)
	if err != nil {
		tokenError(w, http.StatusInternalServerError, "server_error", "could not sign token")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        ac.Scope,
		"id_token":     idToken,
	})
}

func (s *server) userinfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}

	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))

	s.mu.Lock()
	at, ok := s.tokens[token]
	s.mu.Unlock()

	if !ok || time.Now().After(at.Expires) {
		http.Error(w, "invalid bearer token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sub":            at.User.Sub,
		"email":          at.User.Email,
		"email_verified": true,
		"name":           at.User.Name,
	})
}

func (s *server) signJWT(claims map[string]any) (string, error) {
	header := map[string]any{
		"typ": "JWT",
		"alg": "RS256",
		"kid": s.kid,
	}

	h, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	c, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	input := b64(h) + "." + b64(c)
	sum := sha256.Sum256([]byte(input))

	sig, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return input + "." + b64(sig), nil
}

func verifyPKCE(challenge, method, verifier string) bool {
	if challenge == "" {
		return true
	}
	if verifier == "" {
		return false
	}

	switch method {
	case "", "plain":
		return verifier == challenge
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		return b64(sum[:]) == challenge
	default:
		return false
	}
}

func hasScope(scope, wanted string) bool {
	for _, s := range strings.Fields(scope) {
		if s == wanted {
			return true
		}
	}
	return false
}

func randomB64(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b64(b)
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func tokenError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": desc,
	})
}

func parseCSV(s string) map[string]bool {
	out := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
