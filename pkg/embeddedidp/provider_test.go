package embeddedidp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/domain"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/embeddedidp"
)

func TestProductionValidationRejectsHTTPAndMemory(t *testing.T) {
	st := memory.New()
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(context.Background(), key)
	_, err := embeddedidp.New(embeddedidp.Options{Issuer: "http://example.com/idp", Mode: embeddedidp.ProductionMode, Store: st, Cookie: embeddedidp.CookieConfig{Secure: true}})
	if err == nil {
		t.Fatal("expected production HTTP issuer rejection")
	}
	_, err = embeddedidp.New(embeddedidp.Options{Issuer: "https://example.com/idp", Mode: embeddedidp.ProductionMode, Store: st, Cookie: embeddedidp.CookieConfig{Secure: true}})
	if err == nil {
		t.Fatal("expected production memory store rejection")
	}
}

func TestDevProviderBuildsAndHasNoDebug(t *testing.T) {
	ctx := context.Background()
	st := memory.New()
	_ = st.PutClient(ctx, domain.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost:3000/callback"}, AllowedScopes: []string{"openid"}})
	key, _ := keys.GenerateRSA("kid-1", time.Now())
	_ = st.CreateSigningKey(ctx, key)
	p, err := embeddedidp.New(embeddedidp.Options{Issuer: "http://127.0.0.1:5556", Mode: embeddedidp.DevMode, Store: st})
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
