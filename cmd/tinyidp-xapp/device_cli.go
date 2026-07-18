package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/pkg/errors"
)

type DeviceLoginCommand struct{ *cmds.CommandDescription }
type DeviceLoginSettings struct {
	Issuer, ClientID, Audience, Scopes, TokenCache string
}

var _ cmds.BareCommand = (*DeviceLoginCommand)(nil)

var deviceLoginPollWait = func(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func NewDeviceLoginCommand() (*DeviceLoginCommand, error) {
	return &DeviceLoginCommand{cmds.NewCommandDescription("device-login", cmds.WithShort("Authorize this terminal through tiny-idp device login"), cmds.WithFlags(
		fields.New("issuer", fields.TypeString, fields.WithRequired(true), fields.WithHelp("tiny-idp issuer URL, including /idp")),
		fields.New("client-id", fields.TypeString, fields.WithDefault(deviceClientID), fields.WithHelp("Registered public device client ID")),
		fields.New("audience", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Exact API audience")),
		fields.New("scopes", fields.TypeString, fields.WithDefault("openid bbs.read bbs.post.create"), fields.WithHelp("Requested device scopes")),
		fields.New("token-cache", fields.TypeString, fields.WithDefault("./tinyidp-xapp-device-token.json"), fields.WithHelp("Owner-only local token cache")),
	))}, nil
}
func (c *DeviceLoginCommand) Run(ctx context.Context, vals *values.Values) error {
	var s DeviceLoginSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return errors.Wrap(err, "decode device-login settings")
	}
	token, expiry, err := deviceLogin(ctx, s)
	if err != nil {
		return err
	}
	if err := writeDeviceTokenCache(s.TokenCache, deviceTokenCache{AccessToken: token, ExpiresAt: expiry, Issuer: s.Issuer, Audience: s.Audience}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Device authorization complete. Token cache: %s (expires %s)\n", s.TokenCache, expiry.UTC().Format(time.RFC3339))
	return nil
}

type BBSGetCommand struct{ *cmds.CommandDescription }
type BBSPostCommand struct{ *cmds.CommandDescription }
type BBSSettings struct {
	APIBaseURL string `glazed:"api-base-url"`
	TokenCache string `glazed:"token-cache"`
	Title      string `glazed:"title"`
	Body       string `glazed:"body"`
	Category   string `glazed:"category"`
}

var _ cmds.BareCommand = (*BBSGetCommand)(nil)
var _ cmds.BareCommand = (*BBSPostCommand)(nil)

func NewBBSGetCommand() (*BBSGetCommand, error) {
	return &BBSGetCommand{cmds.NewCommandDescription("bbs-get", cmds.WithShort("Read the BBS using a cached device token"), cmds.WithFlags(apiBaseFlag(), tokenCacheFlag()))}, nil
}
func NewBBSPostCommand() (*BBSPostCommand, error) {
	return &BBSPostCommand{cmds.NewCommandDescription("bbs-post", cmds.WithShort("Post to the BBS using a cached device token"), cmds.WithFlags(
		apiBaseFlag(), tokenCacheFlag(),
		fields.New("title", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Post title")),
		fields.New("body", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Post body")),
		fields.New("category", fields.TypeString, fields.WithDefault("notes"), fields.WithHelp("BBS category")),
	))}, nil
}
func apiBaseFlag() *fields.Definition {
	return fields.New("api-base-url", fields.TypeString, fields.WithRequired(true), fields.WithHelp("xapp public base URL"))
}
func tokenCacheFlag() *fields.Definition {
	return fields.New("token-cache", fields.TypeString, fields.WithDefault("./tinyidp-xapp-device-token.json"), fields.WithHelp("Owner-only local token cache"))
}
func (c *BBSGetCommand) Run(ctx context.Context, vals *values.Values) error {
	var s BBSSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	return callBBSAPI(ctx, s, http.MethodGet, "/api/device/bbs", nil)
}
func (c *BBSPostCommand) Run(ctx context.Context, vals *values.Values) error {
	var s BBSSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil {
		return err
	}
	return callBBSAPI(ctx, s, http.MethodPost, "/api/device/bbs/posts", map[string]string{"title": s.Title, "body": s.Body, "category": s.Category})
}

type deviceTokenCache struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Issuer      string    `json:"issuer"`
	Audience    string    `json:"audience"`
}

func loadDeviceTokenCache(file string) (deviceTokenCache, error) {
	info, err := os.Stat(file)
	if err != nil {
		return deviceTokenCache{}, errors.Wrap(err, "stat token cache")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return deviceTokenCache{}, errors.New("token cache must be a regular mode-0600 file")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return deviceTokenCache{}, errors.Wrap(err, "read token cache")
	}
	var c deviceTokenCache
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, errors.Wrap(err, "decode token cache")
	}
	if c.AccessToken == "" || !time.Now().UTC().Before(c.ExpiresAt) {
		return c, errors.New("device token cache is missing or expired; run device-login")
	}
	return c, nil
}
func writeDeviceTokenCache(file string, c deviceTokenCache) error {
	if c.AccessToken == "" || c.ExpiresAt.IsZero() {
		return errors.New("complete device token is required")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return errors.Wrap(err, "create token-cache directory")
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.Wrap(err, "encode token cache")
	}
	tmp := file + ".tmp"
	if err = os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return errors.Wrap(err, "write token cache")
	}
	return errors.Wrap(os.Rename(tmp, file), "publish token cache")
}

func deviceLogin(ctx context.Context, s DeviceLoginSettings) (string, time.Time, error) {
	issuer := strings.TrimRight(strings.TrimSpace(s.Issuer), "/")
	if issuer == "" || s.ClientID == "" || s.Audience == "" {
		return "", time.Time{}, errors.New("issuer, client ID, and audience are required")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	var d struct {
		Issuer, DeviceAuthorizationEndpoint, TokenEndpoint string `json:"-"`
	}
	var raw map[string]any
	r, err := client.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "discover issuer")
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("issuer discovery returned status %d", r.StatusCode)
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return "", time.Time{}, errors.Wrap(err, "decode issuer discovery")
	}
	d.Issuer, _ = raw["issuer"].(string)
	d.DeviceAuthorizationEndpoint, _ = raw["device_authorization_endpoint"].(string)
	d.TokenEndpoint, _ = raw["token_endpoint"].(string)
	if d.Issuer != issuer || d.DeviceAuthorizationEndpoint == "" || d.TokenEndpoint == "" {
		return "", time.Time{}, errors.New("issuer discovery is incomplete or does not match configured issuer")
	}
	start, err := client.PostForm(d.DeviceAuthorizationEndpoint, url.Values{"client_id": {s.ClientID}, "scope": {s.Scopes}, "audience": {s.Audience}})
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "start device authorization")
	}
	defer start.Body.Close()
	var grant struct {
		DeviceCode, UserCode, VerificationURI, VerificationURIComplete string `json:"-"`
		ExpiresIn, Interval                                            int64  `json:"-"`
	}
	var graw map[string]any
	if err := json.NewDecoder(start.Body).Decode(&graw); err != nil {
		return "", time.Time{}, errors.Wrap(err, "decode device authorization")
	}
	grant.DeviceCode, _ = graw["device_code"].(string)
	grant.UserCode, _ = graw["user_code"].(string)
	grant.VerificationURI, _ = graw["verification_uri"].(string)
	grant.VerificationURIComplete, _ = graw["verification_uri_complete"].(string)
	if v, ok := graw["expires_in"].(float64); ok {
		grant.ExpiresIn = int64(v)
	}
	if v, ok := graw["interval"].(float64); ok {
		grant.Interval = int64(v)
	}
	if start.StatusCode != http.StatusOK || grant.DeviceCode == "" {
		return "", time.Time{}, fmt.Errorf("device authorization returned status %d", start.StatusCode)
	}
	verification := grant.VerificationURIComplete
	if verification == "" {
		verification = grant.VerificationURI
	}
	fmt.Fprintf(os.Stdout, "Open %s\nCode: %s\n", verification, grant.UserCode)
	deadline := time.Now().Add(time.Duration(grant.ExpiresIn) * time.Second)
	interval := time.Duration(grant.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	for {
		if !time.Now().Before(deadline) {
			return "", time.Time{}, errors.New("device authorization expired")
		}
		if err := deviceLoginPollWait(ctx, interval); err != nil {
			return "", time.Time{}, err
		}
		resp, err := client.PostForm(d.TokenEndpoint, url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "client_id": {s.ClientID}, "device_code": {grant.DeviceCode}})
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "poll device token")
		}
		var token map[string]any
		err = json.NewDecoder(resp.Body).Decode(&token)
		_ = resp.Body.Close()
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "decode device token")
		}
		if resp.StatusCode == http.StatusOK {
			access, _ := token["access_token"].(string)
			expires, _ := token["expires_in"].(float64)
			if access == "" || expires <= 0 {
				return "", time.Time{}, errors.New("token response is incomplete")
			}
			return access, time.Now().UTC().Add(time.Duration(expires) * time.Second), nil
		}
		code, _ := token["error"].(string)
		if code == "authorization_pending" {
			continue
		}
		if code == "slow_down" {
			interval += 5 * time.Second
			continue
		}
		return "", time.Time{}, fmt.Errorf("device token request failed: %s", code)
	}
}
func callBBSAPI(ctx context.Context, s BBSSettings, method, path string, body any) error {
	cache, err := loadDeviceTokenCache(s.TokenCache)
	if err != nil {
		return err
	}
	var reader *strings.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return marshalErr
		}
		reader = strings.NewReader(string(raw))
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.APIBaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cache.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("BBS API returned status %d", resp.StatusCode)
	}
	_, err = fmt.Fprintln(os.Stdout, func() string { raw, _ := io.ReadAll(resp.Body); return string(raw) }())
	return err
}
