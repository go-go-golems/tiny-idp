package embeddedidp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/sqlitestore"
)

func TestProductionValidationRejectsMissingTokenSecret(t *testing.T) {
	st := memory.New()
	_ = st.PutClient(context.Background(), idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}})
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
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost:3000/callback"}, AllowedScopes: []string{"openid"}})
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

func TestProviderReadinessAndIdempotentClose(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost:3000/callback"}, AllowedScopes: []string{"openid"}})
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
	client := idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour}
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

func TestProductionStartupRejectsCorruptPublishedVerificationKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig(filepath.Join(dir, "idp.db")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"https://app.example.test/callback"}, AllowedScopes: []string{"openid"}, AccessTokenTTL: time.Hour, IDTokenTTL: time.Hour, RefreshTokenTTL: 24 * time.Hour}); err != nil {
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
