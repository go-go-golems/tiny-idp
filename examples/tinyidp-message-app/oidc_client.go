package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const loginAttemptLifetime = 5 * time.Minute

type oidcClient struct {
	config             oauth2.Config
	publicOrigin       string
	endSessionEndpoint string
	verifier           *oidc.IDTokenVerifier
	http               *http.Client
	now                func() time.Time
}

// issuerRewriteTransport keeps the issuer URL visible to oidc and oauth2
// while routing only back-channel traffic to a private container address.
// Browser redirects retain the public canonical issuer.
type issuerRewriteTransport struct {
	issuer      *url.URL
	backchannel *url.URL
	base        http.RoundTripper
}

func externalBackchannelClient(issuerRaw, backchannelRaw string) (*http.Client, error) {
	if strings.TrimSpace(backchannelRaw) == "" {
		return &http.Client{}, nil
	}
	issuer, err := url.Parse(issuerRaw)
	if err != nil {
		return nil, errors.Wrap(err, "parse external issuer")
	}
	backchannel, err := url.Parse(backchannelRaw)
	if err != nil {
		return nil, errors.Wrap(err, "parse external OIDC backchannel URL")
	}
	return &http.Client{Transport: &issuerRewriteTransport{issuer: issuer, backchannel: backchannel, base: http.DefaultTransport}}, nil
}

func (t *issuerRewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t == nil || t.issuer == nil || t.backchannel == nil || request == nil || request.URL == nil || request.URL.Scheme != t.issuer.Scheme || request.URL.Host != t.issuer.Host {
		return http.DefaultTransport.RoundTrip(request)
	}
	clone := request.Clone(request.Context())
	clone.URL = cloneURL(request.URL)
	clone.URL.Scheme = t.backchannel.Scheme
	clone.URL.Host = t.backchannel.Host
	issuerPath := strings.TrimRight(t.issuer.EscapedPath(), "/")
	requestPath := request.URL.EscapedPath()
	suffix := strings.TrimPrefix(requestPath, issuerPath)
	clone.URL.Path = strings.TrimRight(t.backchannel.EscapedPath(), "/") + suffix
	clone.URL.RawPath = ""
	clone.Host = t.issuer.Host
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func cloneURL(value *url.URL) *url.URL {
	clone := *value
	return &clone
}

type loginCompletion struct {
	SessionToken string
	ReturnTo     string
}

func newOIDCClient(ctx context.Context, issuer, publicBaseURL string, client *http.Client) (*oidcClient, error) {
	if ctx == nil || client == nil {
		return nil, errors.New("OIDC context and HTTP client are required")
	}
	issuer = strings.TrimSpace(issuer)
	publicBaseURL, err := normalizePublicBaseURL(publicBaseURL)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, errors.Wrap(err, "discover embedded OIDC provider")
	}
	config := oauth2.Config{
		ClientID: clientID, Endpoint: provider.Endpoint(), RedirectURL: publicBaseURL + callbackPath,
		Scopes: []string{oidc.ScopeOpenID, "profile"},
	}
	return &oidcClient{config: config, publicOrigin: publicBaseURL, endSessionEndpoint: strings.TrimSuffix(issuer, "/") + "/end-session", verifier: provider.Verifier(&oidc.Config{ClientID: clientID}), http: client, now: time.Now}, nil
}

func (c *oidcClient) beginLogin(ctx context.Context, store *appStore, rawReturnTo string) (string, error) {
	if c == nil || store == nil {
		return "", errors.New("OIDC client and application store are required")
	}
	returnTo, err := normalizeReturnTo(rawReturnTo)
	if err != nil {
		return "", err
	}
	state, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomURLToken(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomURLToken(48)
	if err != nil {
		return "", err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	now := c.now().UTC()
	if err := store.createLoginAttempt(ctx, state, loginAttempt{
		Nonce: nonce, PKCEVerifier: verifier, ReturnTo: returnTo,
		CreatedAt: now, ExpiresAt: now.Add(loginAttemptLifetime),
	}); err != nil {
		return "", err
	}
	// Message Desk is deliberately an account-chooser demonstration. A browser
	// with remembered tiny-idp accounts sees a provider-owned selection prompt;
	// one without them receives the ordinary password login page.
	return c.config.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "select_account")), nil
}

func (c *oidcClient) finishLogin(ctx context.Context, store *appStore, state, code string) (loginCompletion, error) {
	if c == nil || c.verifier == nil || store == nil || strings.TrimSpace(code) == "" || len(code) > 8192 {
		return loginCompletion{}, errors.New("OIDC callback is invalid")
	}
	// Discovery, the token exchange, and JWKS verification must use the same
	// explicit in-process transport. The browser reaches the IdP through the
	// public handler, while these back-channel requests must never silently
	// fall back to the host network.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	now := c.now().UTC()
	attempt, err := store.consumeLoginAttempt(ctx, state, now)
	if err != nil {
		return loginCompletion{}, err
	}
	token, err := c.config.Exchange(ctx, code, oauth2.VerifierOption(attempt.PKCEVerifier))
	if err != nil {
		return loginCompletion{}, errors.Wrap(err, "exchange OIDC authorization code")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return loginCompletion{}, errors.New("OIDC token response does not contain an ID token")
	}
	identity, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return loginCompletion{}, errors.Wrap(err, "verify OIDC ID token")
	}
	var claims struct {
		Nonce string `json:"nonce"`
		Name  string `json:"name"`
	}
	if err := identity.Claims(&claims); err != nil {
		return loginCompletion{}, errors.Wrap(err, "decode OIDC claims")
	}
	if identity.Subject == "" || claims.Nonce != attempt.Nonce {
		return loginCompletion{}, errors.New("OIDC identity does not match the login attempt")
	}
	sessionToken, err := randomURLToken(32)
	if err != nil {
		return loginCompletion{}, err
	}
	csrfSecret := make([]byte, 32)
	if _, err := rand.Read(csrfSecret); err != nil {
		return loginCompletion{}, errors.Wrap(err, "generate session CSRF secret")
	}
	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = identity.Subject
	}
	if err := store.createAppSession(ctx, sessionToken, appSession{
		Subject: identity.Subject, DisplayName: displayName, CSRFSecret: csrfSecret,
		CreatedAt: now, ExpiresAt: now.Add(8 * time.Hour),
	}); err != nil {
		return loginCompletion{}, err
	}
	return loginCompletion{SessionToken: sessionToken, ReturnTo: attempt.ReturnTo}, nil
}

func normalizeReturnTo(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		!strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return "", errors.New("return_to must be a local absolute path")
	}
	lower := strings.ToLower(parsed.EscapedPath())
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") || strings.Contains(lower, "%2e") || path.Clean(parsed.Path) != parsed.Path {
		return "", errors.New("return_to path is not canonical")
	}
	return parsed.Path, nil
}

func randomURLToken(size int) (string, error) {
	if size < 1 {
		return "", fmt.Errorf("random token size must be positive")
	}
	contents := make([]byte, size)
	if _, err := rand.Read(contents); err != nil {
		return "", errors.Wrap(err, "generate random token")
	}
	return base64.RawURLEncoding.EncodeToString(contents), nil
}
