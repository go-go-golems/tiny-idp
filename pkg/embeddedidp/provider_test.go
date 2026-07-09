package embeddedidp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
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
	if report := p.Readiness(ctx); !report.Ready || len(report.Checks) != 3 {
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

func TestNewRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := embeddedidp.New(ctx, embeddedidp.Options{})
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}
