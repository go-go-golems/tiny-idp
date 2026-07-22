package fositeadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

func TestRenderRateLimitedProducesTerminalBrowserDocument(t *testing.T) {
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	provider := &Provider{browserErrorUI: renderer}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://idp.example.test/authorize", nil)

	provider.renderRateLimited(recorder, request, "")

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("content type=%q", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "Please wait before trying again") || strings.Contains(body, "rate limited") {
		t.Fatalf("unexpected browser error body=%q", body)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("cache headers=%v", recorder.Header())
	}
}
