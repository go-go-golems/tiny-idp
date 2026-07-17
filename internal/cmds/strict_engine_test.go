package cmds

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/tiny-idp/internal/sections/oidc"
)

func TestBuildStrictProviderSmoke(t *testing.T) {
	cfg := &oidc.Settings{Issuer: "http://127.0.0.1:5556", ClientID: "public-spa", RedirectURIs: []string{"http://localhost:8080/callback"}, Engine: "fosite"}
	registry, err := buildScenarioRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := buildStrictProvider(cfg, buildClientRegistry(cfg), registry)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(provider.Handler())
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("discovery status = %d", resp.StatusCode)
	}
	debug, err := http.Get(ts.URL + "/debug")
	if err != nil {
		t.Fatal(err)
	}
	defer debug.Body.Close()
	if debug.StatusCode != http.StatusNotFound {
		t.Fatalf("strict debug status = %d", debug.StatusCode)
	}
}
