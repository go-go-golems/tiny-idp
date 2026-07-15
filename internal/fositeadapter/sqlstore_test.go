package fositeadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/securitytrace"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestFositeSQLiteRefreshTokenReuseIsRejected(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-reuse", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts.URL, verifier)
	tokens := exchangeCode(t, ts.URL, code, verifier)
	oldRefresh := tokens["refresh_token"].(string)
	firstRefresh := refreshToken(t, ts.URL, oldRefresh)
	if firstRefresh["refresh_token"] == "" || firstRefresh["refresh_token"] == oldRefresh {
		t.Fatalf("refresh token was not rotated: %#v", firstRefresh)
	}
	refreshTokenMustFail(t, ts.URL, oldRefresh)
}

func TestSQLiteAuthorizePersistenceFailpointsAreAtomic(t *testing.T) {
	points := []string{
		"before_authorize_code",
		"after_authorize_code",
		"before_pkce",
		"after_pkce",
		"before_oidc",
		"after_oidc",
		"before_commit",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			ctx := context.Background()
			st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
				t.Fatal(err)
			}
			if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
				t.Fatal(err)
			}
			key, err := keys.GenerateRSA("kid-failpoint", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if err := st.CreateSigningKey(ctx, key); err != nil {
				t.Fatal(err)
			}
			provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
				Issuer:    "http://127.0.0.1:5556",
				Store:     st,
				SecretKey: []byte("sqlite-failpoint-secret-key-32"),
				AuthorizePersistenceHook: func(candidate string) error {
					if candidate == point {
						return errors.New("injected authorize persistence failure")
					}
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(provider.Handler())
			defer server.Close()

			form := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			csrf, cookie := fetchCSRF(t, server.URL, form)
			form.Set("csrf_token", csrf)
			req, _ := http.NewRequest(http.MethodPost, server.URL+"/authorize", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(cookie)
			client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			location, _ := url.Parse(resp.Header.Get("Location"))
			if location.Query().Get("code") != "" {
				t.Fatalf("failpoint %s issued code: %s", point, location)
			}
			for _, table := range []string{"fosite_authorize_codes", "fosite_pkces", "fosite_oidc_sessions"} {
				var count int
				if err := st.SQLDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatalf("failpoint %s left %d rows in %s", point, count, table)
				}
			}
			var consumed int
			if err := st.SQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM authorization_interactions WHERE consumed_at IS NOT NULL`).Scan(&consumed); err != nil {
				t.Fatal(err)
			}
			if consumed != 0 {
				t.Fatalf("failpoint %s consumed %d interactions", point, consumed)
			}
		})
	}
}

func TestSQLiteAuthorizePersistenceCommitsAllArtifactsAndInteraction(t *testing.T) {
	ctx := context.Background()
	st, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-atomic-success", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("sqlite-fosite-secret-key-32-bytes")})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()

	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if code := authorizeForCode(t, server.URL, verifier); code == "" {
		t.Fatal("authorization code missing")
	}
	for _, table := range []string{"fosite_authorize_codes", "fosite_pkces", "fosite_oidc_sessions"} {
		var count int
		if err := st.SQLDB().QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("successful authorization left %d rows in %s, want 1", count, table)
		}
	}
	var consumed int
	if err := st.SQLDB().QueryRowContext(ctx, `SELECT COUNT(*) FROM authorization_interactions WHERE consumed_at IS NOT NULL`).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed != 1 {
		t.Fatalf("successful authorization consumed %d interactions, want 1", consumed)
	}
}

func TestSQLiteAuthorizationCodeRedemptionFailpointsAreAtomic(t *testing.T) {
	points := []string{
		"before_begin_token",
		"before_invalidate_authorize_code",
		"after_invalidate_authorize_code",
		"before_create_access_token",
		"after_create_access_token",
		"before_create_refresh_token",
		"after_create_refresh_token",
		"before_commit_token",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			armed := false
			recorder := &securitytrace.Recorder{}
			store, server, verifier := newSQLiteTokenFixtureWithSecurityEvents(t, func(candidate string) error {
				if armed && candidate == point {
					return errors.New("injected token persistence failure")
				}
				return nil
			}, recorder)
			code := authorizeForCode(t, server.URL, verifier)
			armed = true
			status, _ := postTokenForm(t, server.URL, url.Values{
				"grant_type":    {"authorization_code"},
				"client_id":     {"spa"},
				"code":          {code},
				"redirect_uri":  {"http://localhost/callback"},
				"code_verifier": {verifier},
			})
			if status == http.StatusOK {
				t.Fatalf("failpoint %s returned success", point)
			}
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_authorize_codes WHERE active=1`, 1)
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_access_tokens`, 0)
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens`, 0)
			assertSecurityTrace(t, recorder.Events(), 0)
		})
	}
}

func TestSQLiteDeviceTokenRedemptionFailpointsRollbackGrantAndTokens(t *testing.T) {
	points := []string{
		"before_begin_token",
		"before_consume_device_grant",
		"after_consume_device_grant",
		"before_create_access_token",
		"after_create_access_token",
		"before_commit_token",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			ctx := context.Background()
			now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
			secret := []byte("sqlite-device-token-secret-key-32b")
			store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := store.PutClient(ctx, idpstore.Client{ID: "device-cli", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
				t.Fatal(err)
			}
			if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
				t.Fatal(err)
			}
			key, err := keys.GenerateRSA("kid-device-token", now)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.CreateSigningKey(ctx, key); err != nil {
				t.Fatal(err)
			}
			deviceCode := "sqlite-device-code"
			grant := idpstore.DeviceGrant{ID: "device-grant", DeviceCodeHash: idpstore.HashSecret(secret, "tinyidp/device-code/v1\x00"+deviceCode), UserCodeHash: idpstore.HashSecret(secret, "tinyidp/user-code/v1\x00ABCD-EFGH"), ClientID: "device-cli", RequestedScopes: []string{"openid"}, Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour), PollInterval: time.Second, NextPollAt: now}
			if err := store.CreateDeviceGrant(ctx, grant); err != nil {
				t.Fatal(err)
			}
			if _, err := store.DecideDeviceGrant(ctx, idpstore.DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: idpstore.DeviceGrantApprove, UserID: "u1", Subject: "user-alice", AuthTime: now, AuthenticationMethods: []string{"pwd"}, ApprovedScopes: []string{"openid"}, Now: now}); err != nil {
				t.Fatal(err)
			}
			provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: secret, Clock: func() time.Time { return now }, TokenPersistenceHook: func(candidate string) error {
				if candidate == point {
					return errors.New("injected device token persistence failure")
				}
				return nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(provider.Handler())
			defer server.Close()
			status, _ := postTokenForm(t, server.URL, url.Values{"grant_type": {idpstore.GrantDeviceCode}, "client_id": {"device-cli"}, "device_code": {deviceCode}})
			if status == http.StatusOK {
				t.Fatalf("failpoint %s returned success", point)
			}
			loaded, err := store.InspectDeviceGrantByDeviceCodeHash(ctx, grant.DeviceCodeHash, "device-cli")
			if err != nil || loaded.Status != idpstore.DeviceGrantApproved {
				t.Fatalf("failpoint %s consumed grant %#v %v", point, loaded, err)
			}
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_access_tokens`, 0)
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens`, 0)
		})
	}
}

func TestSQLiteDeviceTokenRedemptionSignsBeforeSingleConnectionTransaction(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	secret := []byte("sqlite-device-token-secret-key-32b")
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.PutClient(ctx, idpstore.Client{ID: "device-cli", Public: true, RequirePKCE: true, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantDeviceCode}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-device-token-success", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	deviceCode := "sqlite-device-code-success"
	grant := idpstore.DeviceGrant{ID: "device-grant-success", DeviceCodeHash: idpstore.HashSecret(secret, "tinyidp/device-code/v1\x00"+deviceCode), UserCodeHash: idpstore.HashSecret(secret, "tinyidp/user-code/v1\x00ABCD-EFGH"), ClientID: "device-cli", RequestedScopes: []string{"openid"}, Status: idpstore.DeviceGrantPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour), PollInterval: time.Second, NextPollAt: now}
	if err := store.CreateDeviceGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DecideDeviceGrant(ctx, idpstore.DeviceDecisionRequest{UserCodeHash: grant.UserCodeHash, Decision: idpstore.DeviceGrantApprove, UserID: "u1", Subject: "user-alice", AuthTime: now, AuthenticationMethods: []string{"pwd"}, ApprovedScopes: []string{"openid"}, Now: now}); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: secret, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/token", strings.NewReader(url.Values{"grant_type": {idpstore.GrantDeviceCode}, "client_id": {"device-cli"}, "device_code": {deviceCode}}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("device token redemption blocked: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("device token status = %d body=%q", response.StatusCode, body)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["access_token"] == "" || body["id_token"] == "" {
		t.Fatalf("device token response = %#v", body)
	}
}

func TestSQLiteRefreshRotationFailpointsAreAtomicAndRetryable(t *testing.T) {
	points := []string{
		"before_begin_token",
		"before_rotate_refresh",
		"after_revoke_refresh",
		"after_delete_old_access",
		"after_rotate_refresh",
		"before_create_access_token",
		"after_create_access_token",
		"before_create_refresh_token",
		"after_create_refresh_token",
		"before_commit_token",
	}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			armed := false
			recorder := &securitytrace.Recorder{}
			store, server, verifier := newSQLiteTokenFixtureWithSecurityEvents(t, func(candidate string) error {
				if armed && candidate == point {
					return errors.New("injected refresh persistence failure")
				}
				return nil
			}, recorder)
			code := authorizeForCode(t, server.URL, verifier)
			tokens := exchangeCode(t, server.URL, code, verifier)
			oldRefresh := tokens["refresh_token"].(string)
			armed = true
			status, _ := postTokenForm(t, server.URL, url.Values{
				"grant_type":    {"refresh_token"},
				"client_id":     {"spa"},
				"refresh_token": {oldRefresh},
			})
			if status == http.StatusOK {
				t.Fatalf("failpoint %s returned success", point)
			}
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens WHERE active=1`, 1)
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_refresh_tokens`, 1)
			assertSQLCount(t, store, `SELECT COUNT(*) FROM fosite_access_tokens`, 1)
			assertSecurityTrace(t, recorder.Events(), 1)
			armed = false
			retried := refreshToken(t, server.URL, oldRefresh)
			if retried["access_token"] == "" || retried["refresh_token"] == "" {
				t.Fatalf("retry after failpoint %s did not issue tokens: %#v", point, retried)
			}
		})
	}
}

func newSQLiteTokenFixture(t *testing.T, hook func(string) error) (*sqlitestore.Store, *httptest.Server, string) {
	return newSQLiteTokenFixtureWithSecurityEvents(t, hook, nil)
}

func newSQLiteTokenFixtureWithSecurityEvents(t *testing.T, hook func(string) error, securityEvents securitytrace.Sink) (*sqlitestore.Store, *httptest.Server, string) {
	t.Helper()
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-token-atomic", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{
		Issuer:               "http://127.0.0.1:5556",
		Store:                store,
		SecretKey:            []byte("sqlite-fosite-secret-key-32-bytes"),
		TokenPersistenceHook: hook,
		SecurityEvents:       securityEvents,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	t.Cleanup(server.Close)
	return store, server, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
}

func assertSecurityTrace(t *testing.T, events []securitytrace.Event, wantTokenCommits int) {
	t.Helper()
	monitor := securitytrace.NewMonitor()
	tokenCommits := 0
	for _, event := range events {
		monitor.Observe(event)
		if event.Kind == securitytrace.TokenLifecycleDone {
			tokenCommits++
		}
	}
	if violations := monitor.Violations(); len(violations) != 0 {
		t.Fatalf("security trace violations=%v events=%#v", violations, events)
	}
	if tokenCommits != wantTokenCommits {
		t.Fatalf("token lifecycle commits=%d, want %d", tokenCommits, wantTokenCommits)
	}
}

func postTokenForm(t *testing.T, baseURL string, form url.Values) (int, []byte) {
	t.Helper()
	response, err := http.PostForm(baseURL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, body
}

func assertSQLCount(t *testing.T, store *sqlitestore.Store, query string, want int) {
	t.Helper()
	var count int
	if err := store.SQLDB().QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("query %q count=%d, want %d", query, count, want)
	}
}

func TestFositeSQLiteClientWithEmptyScopesRejectsRequestedScope(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-empty-scopes", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: []byte("sqlite-empty-scopes-secret-32")})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/authorize?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status=%d body=%s", resp.StatusCode, b)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Query().Get("error") != "invalid_scope" {
		t.Fatalf("error=%q location=%s", loc.Query().Get("error"), loc.String())
	}
}

func TestFositeSQLiteDisabledClientRejectsPersistedAuthorizationCode(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")
	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}
	if err := st.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-disabled", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts.URL, verifier)
	client.Disabled = true
	if err := st.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	exchangeCodeMustFail(t, ts.URL, code, verifier)
}

func TestFositeSQLiteStoreSurvivesProviderRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "idp.db")
	secretKey := []byte("sqlite-fosite-secret-key-32-bytes")

	st, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	if err := st.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice", Email: "alice@example.test", EmailVerified: true, Name: "Alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-sqlite", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider1, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts1 := httptest.NewServer(provider1.Handler())

	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := authorizeForCode(t, ts1.URL, verifier)
	ts1.Close()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st2, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	provider2, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st2, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts2 := httptest.NewServer(provider2.Handler())
	tokens := exchangeCode(t, ts2.URL, code, verifier)
	ts2.Close()
	if err := st2.Close(); err != nil {
		t.Fatal(err)
	}

	st3, err := sqlitestore.Open(context.Background(), sqlitestore.DefaultConfig(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	defer st3.Close()
	provider3, err := fositeadapter.NewProvider(context.Background(), fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: st3, SecretKey: secretKey})
	if err != nil {
		t.Fatal(err)
	}
	ts3 := httptest.NewServer(provider3.Handler())
	defer ts3.Close()
	refreshed := refreshToken(t, ts3.URL, tokens["refresh_token"].(string))
	if refreshed["access_token"] == "" || refreshed["refresh_token"] == "" {
		t.Fatalf("missing refreshed token fields: %#v", refreshed)
	}
}

func TestTokenSecretRotationInvalidatesPriorOpaqueTokens(t *testing.T) {
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(t.TempDir(), "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid", "profile", "email", "offline_access"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}, AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour}
	if err := store.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}

	first, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("first-token-secret-is-at-least-32-bytes")})
	if err != nil {
		t.Fatal(err)
	}
	firstServer := httptest.NewServer(first.Handler())
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	tokens := exchangeCode(t, firstServer.URL, authorizeForCode(t, firstServer.URL, verifier), verifier)
	firstServer.Close()

	second, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("second-token-secret-is-at-least-32-byte")})
	if err != nil {
		t.Fatal(err)
	}
	secondServer := httptest.NewServer(second.Handler())
	defer secondServer.Close()
	req, err := http.NewRequest(http.MethodGet, secondServer.URL+"/userinfo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+tokens["access_token"].(string))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old access token status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	refreshResponse, err := http.PostForm(secondServer.URL+"/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {"spa"}, "refresh_token": {tokens["refresh_token"].(string)}})
	if err != nil {
		t.Fatal(err)
	}
	defer refreshResponse.Body.Close()
	if refreshResponse.StatusCode == http.StatusOK {
		t.Fatal("old refresh token survived token-secret rotation")
	}
}

func authorizeForCode(t *testing.T, baseURL, verifier string) string {
	t.Helper()
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {s256(verifier)},
		"code_challenge_method": {"S256"},
		"login":                 {"alice"},
	}
	csrfToken, csrfCookie := fetchCSRF(t, baseURL, form)
	form.Set("csrf_token", csrfToken)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	reqAuth, _ := http.NewRequest(http.MethodPost, baseURL+"/authorize", strings.NewReader(form.Encode()))
	reqAuth.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqAuth.AddCookie(csrfCookie)
	resp, err := client.Do(reqAuth)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status = %d body=%s", resp.StatusCode, b)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in location %s", loc.String())
	}
	return code
}

func exchangeCodeMustFail(t *testing.T, baseURL, code, verifier string) {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("token exchange unexpectedly succeeded: %s", b)
	}
}

func exchangeCode(t *testing.T, baseURL, code, verifier string) map[string]any {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"spa"},
		"code":          {code},
		"redirect_uri":  {"http://localhost/callback"},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("token status = %d body=%s", resp.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func refreshTokenMustFail(t *testing.T, baseURL, token string) {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {token},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("refresh reuse unexpectedly succeeded: %s", b)
	}
}

func refreshToken(t *testing.T, baseURL, token string) map[string]any {
	t.Helper()
	resp, err := http.PostForm(baseURL+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"spa"},
		"refresh_token": {token},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("refresh status = %d body=%s", resp.StatusCode, b)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}
