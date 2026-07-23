package jitsi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idp"
)

type fakeBroker struct {
	started pluginapi.StartRequest
	result  pluginapi.Completion
}

func (b *fakeBroker) Start(_ context.Context, request pluginapi.StartRequest) (pluginapi.StartResult, error) {
	b.started = request
	return pluginapi.StartResult{AuthorizationURL: "https://idp.example.test/authorize?state=opaque"}, nil
}
func (b *fakeBroker) Complete(_ context.Context, _ pluginapi.CompleteRequest) (pluginapi.Completion, error) {
	return b.result, nil
}

func TestRuntimeStartCallbackIssuesTokenAndAuditsWithoutSecrets(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	broker := &fakeBroker{}
	audit := idp.NewMemorySink()
	services := pluginapi.RuntimeServices{
		OIDC: broker, Audit: audit, Logger: zerolog.Nop(), Clock: fixedClock{now},
		Random: bytes.NewReader(bytes.Repeat([]byte{7}, 64)),
	}
	signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "app", "meet.example.test", 5*time.Minute, fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	runtime := newRuntime(Settings{
		PublicOrigin: "https://meet.example.test", XMPPDomain: "meet.example.test",
		AppID: "app", OIDCClientID: "jitsi-client",
	}, services, signer, nil)
	defer runtime.Close(context.Background())

	start := httptest.NewRecorder()
	runtime.Handler().ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/start?room=engineering", nil))
	if start.Code != http.StatusSeeOther || !strings.HasPrefix(start.Header().Get("Location"), "https://idp.example.test/authorize") {
		t.Fatalf("start response = %d %q", start.Code, start.Header().Get("Location"))
	}
	cookies := start.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Path != "/integrations/jitsi/" {
		t.Fatalf("binding cookies = %#v", cookies)
	}
	broker.result = pluginapi.Completion{
		Identity: pluginapi.Identity{
			Subject: "user-123", Name: "Test User", Email: "user@example.test", EmailVerified: true,
		},
		PluginState: broker.started.PluginState,
	}
	callbackRequest := httptest.NewRequest(http.MethodGet, "/callback?state=opaque&code=authorization-code", nil)
	callbackRequest.AddCookie(cookies[0])
	callback := httptest.NewRecorder()
	runtime.Handler().ServeHTTP(callback, callbackRequest)
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("callback response = %d %s", callback.Code, callback.Body.String())
	}
	target, err := url.Parse(callback.Header().Get("Location"))
	if err != nil || target.Scheme != "https" || target.Host != "meet.example.test" ||
		target.Path != "/engineering" || target.Query().Get("jwt") == "" {
		t.Fatalf("meeting target = %s, %v", target, err)
	}
	events := audit.Events()
	if len(events) != 1 || events[0].Name != "integration.jitsi.token_issued" ||
		events[0].Subject != "user-123" || events[0].Fields["room"] != "engineering" {
		t.Fatalf("audit events = %#v", events)
	}
	encoded := events[0].Name + events[0].Reason + events[0].Fields["room"]
	if strings.Contains(encoded, target.Query().Get("jwt")) || strings.Contains(encoded, "user@example.test") {
		t.Fatal("audit event leaked token or email")
	}
}

func TestRuntimeRendersStableHTMLForInputAndCancellationErrors(t *testing.T) {
	runtime := &Runtime{
		descriptor: (Definition{}).Descriptor(),
		settings:   Settings{OIDCClientID: "jitsi-client"},
		services: pluginapi.RuntimeServices{
			Audit: idp.NewMemorySink(), Logger: zerolog.New(io.Discard), Clock: fixedClock{time.Now()},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/start", runtime.start)
	mux.HandleFunc("/callback", runtime.callback)
	runtime.handler = mux
	for _, target := range []string{"/start?room=../admin", "/callback?error=access_denied"} {
		response := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest ||
			response.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
			!strings.Contains(response.Body.String(), "<!doctype html>") ||
			strings.Contains(response.Body.String(), "access_denied") {
			t.Fatalf("%s response = %d %q", target, response.Code, response.Body.String())
		}
	}
}
