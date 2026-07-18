package main

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxOIDCResponseBytes int64 = 1 << 20

type discoveryDocument struct {
	Issuer           string `json:"issuer"`
	TokenEndpoint    string `json:"token_endpoint"`
	UserInfoEndpoint string `json:"userinfo_endpoint"`
	JWKSURI          string `json:"jwks_uri"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	IDToken     string `json:"id_token"`
}

type userInfo struct {
	Subject string `json:"sub"`
	Name    string `json:"name"`
	Email   string `json:"email"`
}

type idTokenClaims struct {
	Issuer   string          `json:"iss"`
	Subject  string          `json:"sub"`
	Audience json.RawMessage `json:"aud"`
	Expires  int64           `json:"exp"`
	IssuedAt int64           `json:"iat"`
	Nonce    string          `json:"nonce"`
}

type jwkSet struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KeyType string `json:"kty"`
	KeyID   string `json:"kid"`
	Use     string `json:"use"`
	Alg     string `json:"alg"`
	N       string `json:"n"`
	E       string `json:"e"`
}

func exchangeCode(ctx context.Context, opts rpOptions, code, verifier string) (tokenResponse, error) {
	discovery, err := fetchDiscovery(ctx, opts)
	if err != nil {
		return tokenResponse{}, err
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {opts.ClientID},
		"redirect_uri":  {opts.PublicBaseURL + "/auth/callback"},
		"code":          {code},
		"code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var tokens tokenResponse
	if err := getJSON(opts.HTTPClient, request, &tokens); err != nil {
		return tokenResponse{}, err
	}
	if tokens.AccessToken == "" || !strings.EqualFold(tokens.TokenType, "Bearer") || tokens.IDToken == "" {
		return tokenResponse{}, fmt.Errorf("token response is incomplete")
	}
	return tokens, nil
}

func fetchUserInfo(ctx context.Context, opts rpOptions, accessToken string) (userInfo, error) {
	discovery, err := fetchDiscovery(ctx, opts)
	if err != nil {
		return userInfo{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.UserInfoEndpoint, nil)
	if err != nil {
		return userInfo{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	var profile userInfo
	if err := getJSON(opts.HTTPClient, request, &profile); err != nil {
		return userInfo{}, err
	}
	return profile, nil
}

func fetchDiscovery(ctx context.Context, opts rpOptions) (discoveryDocument, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.Issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return discoveryDocument{}, err
	}
	var discovery discoveryDocument
	if err := getJSON(opts.HTTPClient, request, &discovery); err != nil {
		return discoveryDocument{}, err
	}
	if discovery.Issuer != opts.Issuer || discovery.TokenEndpoint == "" || discovery.UserInfoEndpoint == "" || discovery.JWKSURI == "" {
		return discoveryDocument{}, fmt.Errorf("discovery document is incomplete or has an unexpected issuer")
	}
	return discovery, nil
}

func verifyIDToken(ctx context.Context, opts rpOptions, rawToken, expectedNonce string) (idTokenClaims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return idTokenClaims{}, fmt.Errorf("ID token is not a compact JWT")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return idTokenClaims{}, fmt.Errorf("decode ID token header: %w", err)
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "RS256" || header.KeyID == "" {
		return idTokenClaims{}, fmt.Errorf("ID token must select an RS256 signing key")
	}
	discovery, err := fetchDiscovery(ctx, opts)
	if err != nil {
		return idTokenClaims{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.JWKSURI, nil)
	if err != nil {
		return idTokenClaims{}, err
	}
	var set jwkSet
	if err := getJSON(opts.HTTPClient, request, &set); err != nil {
		return idTokenClaims{}, err
	}
	key, err := rsaKey(set, header.KeyID)
	if err != nil {
		return idTokenClaims{}, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return idTokenClaims{}, fmt.Errorf("decode ID token signature: %w", err)
	}
	digest := crypto.SHA256.New()
	_, _ = digest.Write([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest.Sum(nil), signature); err != nil {
		return idTokenClaims{}, fmt.Errorf("verify ID token signature: %w", err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return idTokenClaims{}, fmt.Errorf("decode ID token claims: %w", err)
	}
	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return idTokenClaims{}, fmt.Errorf("decode ID token claims: %w", err)
	}
	now := time.Now().UTC().Unix()
	if claims.Issuer != opts.Issuer || claims.Subject == "" || claims.Nonce != expectedNonce || claims.Expires <= now || claims.IssuedAt > now+60 || !audienceContains(claims.Audience, opts.ClientID) {
		return idTokenClaims{}, fmt.Errorf("ID token claims do not match the login request")
	}
	return claims, nil
}

func rsaKey(set jwkSet, keyID string) (*rsa.PublicKey, error) {
	for _, candidate := range set.Keys {
		if candidate.KeyID != keyID || candidate.KeyType != "RSA" || candidate.Alg != "RS256" || (candidate.Use != "" && candidate.Use != "sig") {
			continue
		}
		modulus, err := base64.RawURLEncoding.DecodeString(candidate.N)
		if err != nil || len(modulus) == 0 {
			return nil, fmt.Errorf("selected JWK has an invalid modulus")
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(candidate.E)
		if err != nil || len(exponentBytes) == 0 || len(exponentBytes) > 4 {
			return nil, fmt.Errorf("selected JWK has an invalid exponent")
		}
		padded := make([]byte, 4)
		copy(padded[4-len(exponentBytes):], exponentBytes)
		exponent := binary.BigEndian.Uint32(padded)
		if exponent < 3 || exponent > uint32(1<<31-1) {
			return nil, fmt.Errorf("selected JWK exponent is outside the supported range")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(exponent)}, nil
	}
	return nil, fmt.Errorf("ID token signing key is unavailable")
}

func audienceContains(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) != nil {
		return false
	}
	for _, audience := range multiple {
		if audience == expected {
			return true
		}
	}
	return false
}

func getJSON(client *http.Client, request *http.Request, destination any) error {
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	reader := io.LimitReader(response.Body, maxOIDCResponseBytes+1)
	contents, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	if int64(len(contents)) > maxOIDCResponseBytes {
		return fmt.Errorf("OIDC response exceeds size limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("OIDC endpoint returned HTTP %d", response.StatusCode)
	}
	if err := json.Unmarshal(contents, destination); err != nil {
		return fmt.Errorf("decode OIDC response: %w", err)
	}
	return nil
}
