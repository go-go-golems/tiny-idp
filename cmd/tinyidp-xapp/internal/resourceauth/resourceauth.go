// Package resourceauth validates tiny-idp opaque bearer access tokens for an
// xapp resource server. It deliberately owns the provider credential and
// returns only a constrained principal to callers.
package resourceauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultRequestTimeout = 10 * time.Second
	defaultPositiveTTL    = 30 * time.Second
	defaultNegativeTTL    = 3 * time.Second
	maxResponseBytes      = 16 << 10
)

// Outcome is the caller-visible security result. Reasons are intentionally
// coarse: callers must not turn token validation into an oracle.
type Outcome string

const (
	OutcomeAuthenticated Outcome = "authenticated"
	OutcomeUnauthorized  Outcome = "unauthorized"
	OutcomeForbidden     Outcome = "forbidden"
	OutcomeUnavailable   Outcome = "unavailable"
)

// Principal is the minimal authenticated identity an application may use for
// its own authorization decisions. It contains no provider credential.
type Principal struct {
	Subject   string
	ClientID  string
	Scopes    []string
	ExpiresAt time.Time
}

// Result is returned by Authenticate. Principal is non-nil only when Outcome
// is OutcomeAuthenticated.
type Result struct {
	Outcome   Outcome
	Principal Principal
}

// Config is entirely host-owned. ClientSecret must never enter JavaScript,
// browser configuration, logs, metrics, or error strings.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret []byte
	Audience     string

	PositiveCacheTTL time.Duration
	NegativeCacheTTL time.Duration
	HTTPClient       *http.Client
	Now              func() time.Time
}

// Authenticator validates bearer tokens using RFC 7662. It has an in-memory
// cache keyed by an HMAC digest, never the raw bearer token.
type Authenticator struct {
	issuer                string
	introspectionEndpoint string
	clientID              string
	clientSecret          []byte
	audience              string
	positiveTTL           time.Duration
	negativeTTL           time.Duration
	httpClient            *http.Client
	now                   func() time.Time
	cacheKey              []byte
	mu                    sync.Mutex
	cache                 map[string]cacheEntry
}

type cacheEntry struct {
	active    bool
	principal Principal
	expiresAt time.Time
}

type discoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint"`
	IntrospectionAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported"`
}

type introspectionResponse struct {
	Active    bool     `json:"active"`
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	ClientID  string   `json:"client_id"`
	Scope     string   `json:"scope"`
	Audience  []string `json:"aud"`
	Expires   int64    `json:"exp"`
	TokenType string   `json:"token_type"`
}

// New discovers and validates the issuer's introspection endpoint. The caller
// supplies a bounded transport appropriate to its deployment (in-process for
// the embedded development host; TLS for a deployed resource server).
func New(ctx context.Context, cfg Config) (*Authenticator, error) {
	if ctx == nil {
		return nil, errors.New("resource authentication context is required")
	}
	issuer, err := canonicalURL(cfg.IssuerURL)
	if err != nil {
		return nil, errors.Wrap(err, "validate issuer URL")
	}
	if strings.TrimSpace(cfg.ClientID) == "" || len(cfg.ClientSecret) == 0 || strings.TrimSpace(cfg.Audience) == "" {
		return nil, errors.New("issuer URL, client ID, client secret, and audience are required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultRequestTimeout}
	}
	positiveTTL := cfg.PositiveCacheTTL
	if positiveTTL == 0 {
		positiveTTL = defaultPositiveTTL
	}
	if positiveTTL <= 0 || positiveTTL > defaultPositiveTTL {
		return nil, fmt.Errorf("positive cache TTL must be between 1ns and %s", defaultPositiveTTL)
	}
	negativeTTL := cfg.NegativeCacheTTL
	if negativeTTL == 0 {
		negativeTTL = defaultNegativeTTL
	}
	if negativeTTL <= 0 || negativeTTL > defaultPositiveTTL {
		return nil, fmt.Errorf("negative cache TTL must be between 1ns and %s", defaultPositiveTTL)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	endpoint, err := discoverIntrospectionEndpoint(ctx, client, issuer)
	if err != nil {
		return nil, err
	}
	cacheKey := make([]byte, 32)
	if _, err := rand.Read(cacheKey); err != nil {
		return nil, errors.Wrap(err, "generate resource authentication cache key")
	}
	return &Authenticator{
		issuer: issuer, introspectionEndpoint: endpoint, clientID: strings.TrimSpace(cfg.ClientID),
		clientSecret: append([]byte(nil), cfg.ClientSecret...), audience: strings.TrimSpace(cfg.Audience),
		positiveTTL: positiveTTL, negativeTTL: negativeTTL, httpClient: client, now: now,
		cacheKey: cacheKey, cache: make(map[string]cacheEntry),
	}, nil
}

// Authenticate verifies one Authorization header value and then enforces all
// scopes required by the route. An unavailable provider fails closed.
func (a *Authenticator) Authenticate(ctx context.Context, authorization []string, requiredScopes []string) Result {
	if a == nil || ctx == nil {
		return Result{Outcome: OutcomeUnavailable}
	}
	token, ok := parseBearer(authorization)
	if !ok {
		return Result{Outcome: OutcomeUnauthorized}
	}
	key := a.tokenKey(token)
	if entry, ok := a.cached(key); ok {
		if !entry.active {
			return Result{Outcome: OutcomeUnauthorized}
		}
		if !containsAll(entry.principal.Scopes, requiredScopes) {
			return Result{Outcome: OutcomeForbidden}
		}
		return Result{Outcome: OutcomeAuthenticated, Principal: clonePrincipal(entry.principal)}
	}

	response, available := a.introspect(ctx, token)
	if !available {
		return Result{Outcome: OutcomeUnavailable}
	}
	if !response.Active {
		a.storeInactive(key)
		return Result{Outcome: OutcomeUnauthorized}
	}
	principal, ok := a.validPrincipal(response)
	if !ok {
		return Result{Outcome: OutcomeUnauthorized}
	}
	a.storeActive(key, principal)
	if !containsAll(principal.Scopes, requiredScopes) {
		return Result{Outcome: OutcomeForbidden}
	}
	return Result{Outcome: OutcomeAuthenticated, Principal: principal}
}

func discoverIntrospectionEndpoint(ctx context.Context, client *http.Client, issuer string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return "", errors.Wrap(err, "create issuer discovery request")
	}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "discover issuer metadata")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("issuer discovery returned status %d", response.StatusCode)
	}
	var document discoveryDocument
	if err := decodeJSON(response.Body, &document); err != nil {
		return "", errors.Wrap(err, "decode issuer discovery")
	}
	if document.Issuer != issuer {
		return "", errors.New("issuer discovery did not return the configured issuer")
	}
	if !contains(document.IntrospectionAuthMethodsSupported, "client_secret_basic") {
		return "", errors.New("issuer discovery does not support client_secret_basic introspection")
	}
	endpoint, err := canonicalURL(document.IntrospectionEndpoint)
	if err != nil {
		return "", errors.Wrap(err, "validate introspection endpoint")
	}
	issuerURL, _ := url.Parse(issuer)
	endpointURL, _ := url.Parse(endpoint)
	if issuerURL.Scheme != endpointURL.Scheme || issuerURL.Host != endpointURL.Host {
		return "", errors.New("introspection endpoint does not belong to configured issuer origin")
	}
	return endpoint, nil
}

func (a *Authenticator) introspect(ctx context.Context, token string) (introspectionResponse, bool) {
	form := url.Values{"token": []string{token}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.introspectionEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return introspectionResponse{}, false
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(a.clientID, string(a.clientSecret))
	response, err := a.httpClient.Do(request)
	if err != nil {
		return introspectionResponse{}, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return introspectionResponse{}, false
	}
	var result introspectionResponse
	if err := decodeJSON(response.Body, &result); err != nil {
		return introspectionResponse{}, false
	}
	return result, true
}

func (a *Authenticator) validPrincipal(response introspectionResponse) (Principal, bool) {
	if response.Issuer != a.issuer || !strings.EqualFold(response.TokenType, "Bearer") || strings.TrimSpace(response.Subject) == "" || response.Expires <= 0 || !contains(response.Audience, a.audience) {
		return Principal{}, false
	}
	expiresAt := time.Unix(response.Expires, 0).UTC()
	if !a.now().UTC().Before(expiresAt) {
		return Principal{}, false
	}
	return Principal{Subject: response.Subject, ClientID: response.ClientID, Scopes: splitScopes(response.Scope), ExpiresAt: expiresAt}, true
}

func (a *Authenticator) cached(key string) (cacheEntry, bool) {
	now := a.now().UTC()
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.cache[key]
	if !ok {
		return cacheEntry{}, false
	}
	if !now.Before(entry.expiresAt) {
		delete(a.cache, key)
		return cacheEntry{}, false
	}
	return entry, true
}

func (a *Authenticator) storeInactive(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache[key] = cacheEntry{expiresAt: a.now().UTC().Add(a.negativeTTL)}
}

func (a *Authenticator) storeActive(key string, principal Principal) {
	expiresAt := a.now().UTC().Add(a.positiveTTL)
	if principal.ExpiresAt.Before(expiresAt) {
		expiresAt = principal.ExpiresAt
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache[key] = cacheEntry{active: true, principal: clonePrincipal(principal), expiresAt: expiresAt}
}

func (a *Authenticator) tokenKey(token string) string {
	mac := hmac.New(sha256.New, a.cacheKey)
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseBearer(values []string) (string, bool) {
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" || strings.ContainsAny(parts[1], "\r\n") {
		return "", false
	}
	return parts[1], true
}

func canonicalURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("URL must be an absolute origin or path issuer without userinfo, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func decodeJSON(reader io.Reader, value any) error {
	decoder := json.NewDecoder(io.LimitReader(reader, maxResponseBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("response contains multiple JSON values")
	}
	return nil
}

func splitScopes(raw string) []string {
	return strings.Fields(raw)
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func containsAll(values, wanted []string) bool {
	for _, scope := range wanted {
		if !contains(values, scope) {
			return false
		}
	}
	return true
}

func clonePrincipal(principal Principal) Principal {
	principal.Scopes = append([]string(nil), principal.Scopes...)
	return principal
}
