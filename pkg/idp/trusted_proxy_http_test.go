package idp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

func TestTrustedProxyHTTPHandlerAcceptsOnlyConfiguredHTTPSProxyContract(t *testing.T) {
	resolver, err := idp.NewTrustedProxyResolver(idp.TrustedProxyConfig{TrustedCIDRs: []string{"10.42.0.0/24"}})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := idp.NewTrustedProxyHTTPHandler(idp.TrustedProxyHTTPConfig{PublicOrigin: "https://idp.example.test", Resolver: resolver}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || r.URL.Scheme != "https" || r.URL.Host != "idp.example.test" {
			t.Fatalf("public HTTPS request view = %#v", r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://idp.example.test/authorize", nil)
	request.RemoteAddr = "10.42.0.193:8080"
	request.Host = "idp.example.test"
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("X-Forwarded-Host", "idp.example.test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("trusted request status=%d body=%s", response.Code, response.Body.String())
	}

	for name, mutate := range map[string]func(*http.Request){
		"untrusted peer":            func(r *http.Request) { r.RemoteAddr = "198.51.100.9:8080" },
		"plain forwarded transport": func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "http") },
		"wrong host":                func(r *http.Request) { r.Host = "attacker.example.test" },
		"wrong forwarded host":      func(r *http.Request) { r.Header.Set("X-Forwarded-Host", "attacker.example.test") },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request.Clone(request.Context())
			candidate.Header = request.Header.Clone()
			mutate(candidate)
			denied := httptest.NewRecorder()
			handler.ServeHTTP(denied, candidate)
			if denied.Code != http.StatusBadRequest {
				t.Fatalf("denied request status=%d body=%s", denied.Code, denied.Body.String())
			}
		})
	}
}

func TestTrustedProxyHTTPHandlerRejectsInvalidConfiguration(t *testing.T) {
	resolver, err := idp.NewTrustedProxyResolver(idp.TrustedProxyConfig{TrustedCIDRs: []string{"10.42.0.0/24"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, origin := range []string{"http://idp.example.test", "https://idp.example.test/idp", "https://idp.example.test?x=y"} {
		if _, err := idp.NewTrustedProxyHTTPHandler(idp.TrustedProxyHTTPConfig{PublicOrigin: origin, Resolver: resolver}, http.NotFoundHandler()); err == nil {
			t.Fatalf("invalid public origin %q was accepted", origin)
		}
	}
}
