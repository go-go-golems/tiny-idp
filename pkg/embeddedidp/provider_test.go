package embeddedidp_test

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-go-golems/tiny-idp/internal/keys"
	"github.com/go-go-golems/tiny-idp/internal/store/memory"
	"github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

type recordingInteractionRenderer struct {
	called   atomic.Bool
	delegate idpui.InteractionRenderer
}

func (r *recordingInteractionRenderer) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	r.called.Store(true)
	return r.delegate.RenderInteraction(ctx, dst, page)
}

func TestProductionValidationRejectsMissingTokenSecret(t *testing.T) {
	st := memory.New()
	_ = st.PutClient(context.Background(), idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	_, err := embeddedidp.New(context.Background(), embeddedidp.Options{Issuer: "https://example.com/idp", Mode: embeddedidp.ProductionMode, Store: st, Cookie: embeddedidp.CookieConfig{Secure: true}})
	if err == nil {
		t.Fatal("expected production token secret rejection")
	}
}

func TestProductionValidationRejectsHTTPAndMemory(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	_, err := embeddedidp.New(context.Background(), embeddedidp.Options{Issuer: "http://example.com/idp", Mode: embeddedidp.ProductionMode, Store: st, Cookie: embeddedidp.CookieConfig{Secure: true}})
	if err == nil {
		t.Fatal("expected production HTTP issuer rejection")
	}
	_, err = embeddedidp.New(context.Background(), embeddedidp.Options{Issuer: "https://example.com/idp", Mode: embeddedidp.ProductionMode, Store: st, Cookie: embeddedidp.CookieConfig{Secure: true}, Token: embeddedidp.TokenConfig{SecretKey: []byte("production-secret-key-32-bytes-min")}})
	if err == nil {
		t.Fatal("expected production memory store rejection")
	}
}

func TestDevProviderBuildsAndHasNoDebug(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost:3000/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	p, err := embeddedidp.New(context.Background(), embeddedidp.Options{Issuer: "http://127.0.0.1:5556", Mode: embeddedidp.DevMode, Store: st})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/debug")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("debug route status = %d", resp.StatusCode)
	}
}

func TestCustomRendererFlowsThroughPublicOptions(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("ui-key", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	delegate, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	renderer := &recordingInteractionRenderer{delegate: delegate}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: "http://127.0.0.1:5556/idp", Store: store, UI: embeddedidp.UIConfig{Renderer: renderer}})
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{
		"response_type":         {"code"},
		"client_id":             {"spa"},
		"redirect_uri":          {"http://localhost/callback"},
		"scope":                 {"openid"},
		"state":                 {"state-1234567890"},
		"nonce":                 {"nonce-1234567890"},
		"code_challenge":        {"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"},
		"code_challenge_method": {"S256"},
	}
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:5556/idp/authorize?"+values.Encode(), nil)
	recorder := httptest.NewRecorder()
	provider.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !renderer.called.Load() {
		t.Fatalf("status=%d rendererCalled=%v body=%s", recorder.Code, renderer.called.Load(), recorder.Body.String())
	}
}

func TestProviderReadinessAndIdempotentClose(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost:3000/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)

	p, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: "http://127.0.0.1:5556", Mode: embeddedidp.DevMode, Store: st})
	if err != nil {
		t.Fatal(err)
	}
	if report := p.Readiness(ctx); !report.Ready || len(report.Checks) < 8 {
		t.Fatalf("unexpected ready report: %#v", report)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if report := p.Readiness(ctx); report.Ready || len(report.Checks) != 1 || report.Checks[0].Reason != "provider_closed" {
		t.Fatalf("unexpected closed report: %#v", report)
	}

	req := httptest.NewRequest(http.MethodGet, "http://idp.test/healthz", nil)
	rec := httptest.NewRecorder()
	p.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed handler status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestIssuerPathOwnsOnlyPrefixedRoutesAndHealthIsStructured(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	key, err := keys.GenerateRSA("kid", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: "http://127.0.0.1:5556/idp", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{"/healthz": http.StatusNotFound, "/idp/healthz": http.StatusOK, "/idp/readyz": http.StatusOK} {
		recorder := httptest.NewRecorder()
		provider.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://idp.test"+path, nil))
		if recorder.Code != want {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, want)
		}
		if want == http.StatusOK {
			var report idp.ReadinessReport
			if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
				t.Fatalf("%s body: %v", path, err)
			}
			if !report.Ready {
				t.Fatalf("%s report = %#v", path, report)
			}
		}
	}
}

func TestProductionReadinessTransitionsOnAuditFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(dir, "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}, AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour}
	if err := store.PutClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit, err := idp.NewFileAuditSink(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = audit.Close() })
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: "https://issuer.example.test/idp", Mode: embeddedidp.ProductionMode, Store: store, Cookie: embeddedidp.CookieConfig{Secure: true}, Token: embeddedidp.TokenConfig{SecretKey: []byte("production-token-secret-at-least-32-bytes")}, Audit: audit, RateLimiter: idp.NewFixedWindowRateLimiter(100, time.Minute), ClientAddress: idp.DirectClientAddressResolver{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.RunMaintenance(ctx); err != nil {
		t.Fatal(err)
	}
	if report := provider.Readiness(ctx); !report.Ready {
		t.Fatalf("initial readiness = %#v", report)
	}
	if err := audit.Close(); err != nil {
		t.Fatal(err)
	}
	if report := provider.Readiness(ctx); report.Ready {
		t.Fatalf("readiness after audit close = %#v", report)
	}
	if report := provider.Liveness(ctx); !report.Ready {
		t.Fatalf("liveness should ignore dependency outage: %#v", report)
	}
}

func TestProductionTLSDiscoveryAndAuthenticatedIntrospectionSmoke(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(dir, "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	secret := "tls-smoke-resource-secret"
	secretHash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutClient(ctx, idpstore.Client{ID: "smoke-api", SecretHash: secretHash, CanIntrospect: true, AllowedAudiences: []string{"https://api.example.test/smoke"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode}}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("kid-tls-smoke", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	audit, err := idp.NewFileAuditSink(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = audit.Close() })
	provider, err := embeddedidp.New(ctx, embeddedidp.Options{Issuer: "https://issuer.example.test/idp", Mode: embeddedidp.ProductionMode, Store: store, Cookie: embeddedidp.CookieConfig{Secure: true}, Token: embeddedidp.TokenConfig{SecretKey: []byte("production-token-secret-for-tls-smoke-32")}, Audit: audit, RateLimiter: idp.NewFixedWindowRateLimiter(100, time.Minute), ClientAddress: idp.DirectClientAddressResolver{}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(provider.Handler())
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/idp/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.TLS == nil || response.TLS.Version < 0x0303 || response.StatusCode != http.StatusOK {
		t.Fatalf("TLS discovery status=%d tls=%#v", response.StatusCode, response.TLS)
	}
	var discovery struct {
		IntrospectionEndpoint       string   `json:"introspection_endpoint"`
		DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
		Methods                     []string `json:"introspection_endpoint_auth_methods_supported"`
	}
	if err := json.NewDecoder(response.Body).Decode(&discovery); err != nil {
		t.Fatal(err)
	}
	if discovery.IntrospectionEndpoint != "https://issuer.example.test/idp/introspect" || discovery.DeviceAuthorizationEndpoint != "https://issuer.example.test/idp/device_authorization" || len(discovery.Methods) != 1 || discovery.Methods[0] != "client_secret_basic" {
		t.Fatalf("unexpected TLS discovery contract: %#v", discovery)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/idp/introspect", strings.NewReader("token=unknown-opaque-token"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth("smoke-api", secret)
	introspection, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer introspection.Body.Close()
	body, err := io.ReadAll(introspection.Body)
	if err != nil {
		t.Fatal(err)
	}
	if introspection.TLS == nil || introspection.StatusCode != http.StatusOK || subtle.ConstantTimeCompare(body, []byte("{\"active\":false}\n")) != 1 {
		t.Fatalf("TLS introspection status=%d body=%q tls=%#v", introspection.StatusCode, body, introspection.TLS)
	}
}

func TestProductionStartupRejectsCorruptPublishedVerificationKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(dir, "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AllowedGrantTypes: []string{idpstore.GrantAuthorizationCode, idpstore.GrantRefreshToken}, AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour}); err != nil {
		t.Fatal(err)
	}
	active, err := keys.GenerateRSA("active", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, active); err != nil {
		t.Fatal(err)
	}
	retiredAt := time.Now().Add(-time.Minute)
	corrupt := idpstore.SigningKey{ID: "corrupt-retired", Algorithm: "RS256", PrivateKeyPEM: []byte("not a key"), CreatedAt: retiredAt.Add(-time.Hour), NotBefore: retiredAt.Add(-time.Hour), NotAfter: retiredAt}
	if err := store.CreateSigningKey(ctx, corrupt); err != nil {
		t.Fatal(err)
	}
	audit, err := idp.NewFileAuditSink(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = audit.Close() })
	_, err = embeddedidp.New(ctx, embeddedidp.Options{Issuer: "https://issuer.example.test", Mode: embeddedidp.ProductionMode, Store: store, Cookie: embeddedidp.CookieConfig{Secure: true}, Token: embeddedidp.TokenConfig{SecretKey: []byte("production-token-secret-at-least-32-bytes")}, Audit: audit, RateLimiter: idp.NewFixedWindowRateLimiter(100, time.Minute), ClientAddress: idp.DirectClientAddressResolver{}})
	if err == nil {
		t.Fatal("expected corrupt published verification key rejection")
	}
}

func TestNewRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := embeddedidp.New(ctx, embeddedidp.Options{})
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestCookieConfigurationValidation(t *testing.T) {
	store := memory.New()
	tests := []struct {
		name   string
		issuer string
		cookie embeddedidp.CookieConfig
	}{
		{name: "same names", issuer: "http://127.0.0.1:5556/idp", cookie: embeddedidp.CookieConfig{SessionName: "shared", CSRFName: "shared"}},
		{name: "invalid session name", issuer: "http://127.0.0.1:5556/idp", cookie: embeddedidp.CookieConfig{SessionName: "bad name"}},
		{name: "relative path", issuer: "http://127.0.0.1:5556/idp", cookie: embeddedidp.CookieConfig{Path: "idp"}},
		{name: "path does not cover issuer", issuer: "http://127.0.0.1:5556/idp", cookie: embeddedidp.CookieConfig{Path: "/other"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := embeddedidp.New(context.Background(), embeddedidp.Options{Issuer: tt.issuer, Store: store, Cookie: tt.cookie})
			if err == nil {
				t.Fatal("expected invalid cookie configuration to be rejected")
			}
		})
	}
}

func TestAccountChooserConfigurationValidation(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		cfg  embeddedidp.AccountChooserConfig
	}{
		{name: "remembering needs label policy", cfg: embeddedidp.AccountChooserConfig{Enabled: true, RememberOnPasswordLogin: true}},
		{name: "default session cookie collision", cfg: embeddedidp.AccountChooserConfig{Enabled: true, ContextCookieName: "tinyidp_session"}},
		{name: "negative TTL", cfg: embeddedidp.AccountChooserConfig{Enabled: true, ContextTTL: -time.Hour}},
		{name: "too many accounts", cfg: embeddedidp.AccountChooserConfig{Enabled: true, MaxRememberedAccounts: 21}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := embeddedidp.Options{Issuer: "http://127.0.0.1:5556", Store: memory.New(), AccountChooser: tt.cfg}
			if err := opts.Validate(ctx); err == nil {
				t.Fatal("expected account chooser configuration rejection")
			}
		})
	}
}

func TestCookiePathMayBroadenIssuerPathForCombinedHost(t *testing.T) {
	store := memory.New()
	p, err := embeddedidp.New(context.Background(), embeddedidp.Options{
		Issuer: "http://127.0.0.1:5556/idp",
		Store:  store,
		Cookie: embeddedidp.CookieConfig{SessionName: "xapp_idp_session", CSRFName: "xapp_idp_csrf", Path: "/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
