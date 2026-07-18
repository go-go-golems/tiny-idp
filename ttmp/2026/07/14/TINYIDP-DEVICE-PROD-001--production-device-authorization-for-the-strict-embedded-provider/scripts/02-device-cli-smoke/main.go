// Command 02-device-cli-smoke is an intentionally provider-independent RFC
// 8628 smoke client. It discovers a strict tiny-idp issuer, starts one device
// authorization, displays only the user-facing verification information, and
// polls until a terminal result. It never writes or prints bearer credentials.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type discovery struct {
	Issuer                      string   `json:"issuer"`
	DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
	TokenEndpoint               string   `json:"token_endpoint"`
	GrantTypesSupported         []string `json:"grant_types_supported"`
}

type deviceGrant struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

func main() {
	issuer := flag.String("issuer", "", "canonical tiny-idp issuer URL")
	clientID := flag.String("client-id", "", "registered public device client ID")
	scope := flag.String("scope", "openid", "space-separated requested scopes")
	audience := flag.String("audience", "", "optional resource indicator")
	timeout := flag.Duration("timeout", 12*time.Minute, "whole-flow timeout")
	flag.Parse()
	if *issuer == "" || *clientID == "" || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "-issuer, -client-id, and positive -timeout are required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := run(ctx, *issuer, *clientID, *scope, *audience, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "device smoke failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, rawIssuer, clientID, scope, audience string, out io.Writer) error {
	issuer := strings.TrimRight(strings.TrimSpace(rawIssuer), "/")
	client := &http.Client{Timeout: 15 * time.Second}
	metadata, err := getJSON[discovery](ctx, client, issuer+"/.well-known/openid-configuration")
	if err != nil {
		return fmt.Errorf("discover issuer: %w", err)
	}
	if metadata.Issuer != issuer || !sameOrigin(issuer, metadata.DeviceAuthorizationEndpoint) || !sameOrigin(issuer, metadata.TokenEndpoint) || !contains(metadata.GrantTypesSupported, "urn:ietf:params:oauth:grant-type:device_code") {
		return errors.New("discovery does not describe a compatible same-origin device grant")
	}
	form := url.Values{"client_id": {clientID}, "scope": {scope}}
	if audience != "" {
		form.Set("audience", audience)
	}
	grant, status, err := postFormJSON[deviceGrant](ctx, client, metadata.DeviceAuthorizationEndpoint, form)
	if err != nil {
		return fmt.Errorf("start device authorization: %w", err)
	}
	if status != http.StatusOK || grant.DeviceCode == "" || grant.UserCode == "" || grant.VerificationURI == "" || grant.ExpiresIn <= 0 {
		return fmt.Errorf("device authorization rejected with HTTP %d", status)
	}
	verification := grant.VerificationURIComplete
	if verification == "" {
		verification = grant.VerificationURI
	}
	fmt.Fprintf(out, "Open %s\nEnter code: %s\n", verification, grant.UserCode)
	interval := time.Duration(grant.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.NewTimer(time.Duration(grant.ExpiresIn) * time.Second)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return errors.New("device authorization expired before approval")
		case <-time.After(interval):
		}
		response, status, err := postFormJSON[tokenResponse](ctx, client, metadata.TokenEndpoint, url.Values{
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "client_id": {clientID}, "device_code": {grant.DeviceCode},
		})
		if err != nil {
			return fmt.Errorf("poll token endpoint: %w", err)
		}
		if status == http.StatusOK && response.AccessToken != "" && strings.EqualFold(response.TokenType, "bearer") {
			fmt.Fprintln(out, "Device authorization succeeded; bearer credentials were received and deliberately redacted.")
			return nil
		}
		switch response.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "access_denied", "expired_token", "invalid_grant":
			return fmt.Errorf("terminal token response %q", response.Error)
		default:
			return fmt.Errorf("unexpected token response HTTP %d error %q", status, response.Error)
		}
	}
}

func getJSON[T any](ctx context.Context, client *http.Client, endpoint string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return zero, err
	}
	response, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	return decodeJSON[T](response.Body)
}

func postFormJSON[T any](ctx context.Context, client *http.Client, endpoint string, form url.Values) (T, int, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return zero, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(req)
	if err != nil {
		return zero, 0, err
	}
	defer response.Body.Close()
	decoded, err := decodeJSON[T](response.Body)
	if err != nil {
		return zero, response.StatusCode, err
	}
	return decoded, response.StatusCode, nil
}

func decodeJSON[T any](body io.Reader) (T, error) {
	var value T
	limited := io.LimitReader(body, maxResponseBytes+1)
	encoded, err := io.ReadAll(limited)
	if err != nil {
		return value, err
	}
	if len(encoded) > maxResponseBytes {
		return value, errors.New("response exceeds 1 MiB limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return value, errors.New("response contains trailing JSON")
	}
	return value, nil
}

func sameOrigin(issuer, endpoint string) bool {
	left, leftErr := url.Parse(issuer)
	right, rightErr := url.Parse(endpoint)
	return leftErr == nil && rightErr == nil && left.Scheme == right.Scheme && left.Host == right.Host && right.User == nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
