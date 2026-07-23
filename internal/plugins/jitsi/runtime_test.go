package jitsi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/go-go-golems/tiny-idp/internal/observability"
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
	metrics, err := observability.NewMetrics()
	if err != nil {
		t.Fatal(err)
	}
	defer metrics.Close(context.Background())
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	t.Cleanup(func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("shut down tracer provider: %v", err)
		}
	})
	services := pluginapi.RuntimeServices{
		OIDC: broker, Audit: audit, Logger: zerolog.Nop(), Clock: fixedClock{now},
		Random: bytes.NewReader(bytes.Repeat([]byte{7}, 64)),
		Meter:  metrics.Provider().Meter("tinyidp/plugins"),
		Tracer: tracerProvider.Tracer("tinyidp/plugins"),
	}
	signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "app", "meet.example.test", 5*time.Minute, fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := newRuntime(Settings{
		PublicOrigin: "https://meet.example.test", XMPPDomain: "meet.example.test",
		AppID: "app", OIDCClientID: "jitsi-client",
	}, services, signer, nil)
	if err != nil {
		t.Fatal(err)
	}
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
	if broker.started.Registration || broker.started.SelectAccount {
		t.Fatalf("default start intent = %#v", broker.started)
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
	if callback.Code != http.StatusOK {
		t.Fatalf("callback response = %d %s", callback.Code, callback.Body.String())
	}
	targetValue := firstMeetingTarget(t, callback.Body.String())
	target, err := url.Parse(targetValue)
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
	scrape := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if scrape.Code != http.StatusOK ||
		!strings.Contains(scrape.Body.String(), "tinyidp_plugin_requests_total") ||
		!strings.Contains(scrape.Body.String(), "tinyidp_jitsi_tokens_issued_total") {
		t.Fatalf("Jitsi metrics scrape = %d %s", scrape.Code, scrape.Body.String())
	}
	spans := spanRecorder.Ended()
	if len(spans) != 2 || spans[0].Name() != "tinyidp.jitsi.start" || spans[1].Name() != "tinyidp.jitsi.callback" {
		t.Fatalf("Jitsi spans = %#v", spans)
	}
}

func firstMeetingTarget(t *testing.T, body string) string {
	t.Helper()
	const marker = `<p><a href="`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("meeting transition link missing from %q", body)
	}
	start += len(marker)
	end := strings.Index(body[start:], `">Enter the meeting</a>`)
	if end < 0 {
		t.Fatalf("meeting transition link malformed in %q", body)
	}
	return strings.ReplaceAll(body[start:start+end], "&#43;", "+")
}

func TestRuntimePassesExplicitSignupAndAccountChooserIntents(t *testing.T) {
	now := time.Now()
	for _, testCase := range []struct {
		target        string
		registration  bool
		selectAccount bool
	}{
		{target: "/start?room=signup-room&intent=signup", registration: true},
		{target: "/start?room=chooser-room&prompt=select_account", selectAccount: true},
	} {
		broker := &fakeBroker{}
		signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "app", "meet.example.test", 5*time.Minute, fixedClock{now})
		if err != nil {
			t.Fatal(err)
		}
		runtime, err := newRuntime(
			Settings{OIDCClientID: "jitsi-client"},
			pluginapi.RuntimeServices{
				OIDC: broker, Audit: idp.NewMemorySink(), Logger: zerolog.Nop(), Clock: fixedClock{now},
			},
			signer,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, testCase.target, nil))
		if closeErr := runtime.Close(context.Background()); closeErr != nil {
			t.Fatal(closeErr)
		}
		if response.Code != http.StatusSeeOther ||
			broker.started.Registration != testCase.registration ||
			broker.started.SelectAccount != testCase.selectAccount {
			t.Fatalf("%s response=%d request=%#v", testCase.target, response.Code, broker.started)
		}
	}
}

func TestRuntimeRendersStableHTMLForInputAndCancellationErrors(t *testing.T) {
	now := time.Now()
	signer, err := NewSigner([]byte("0123456789abcdef0123456789abcdef"), "app", "meet.example.test", 5*time.Minute, fixedClock{now})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := newRuntime(
		Settings{OIDCClientID: "jitsi-client"},
		pluginapi.RuntimeServices{
			Audit: idp.NewMemorySink(), Logger: zerolog.Nop(), Clock: fixedClock{now},
		},
		signer,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close(context.Background())
	for _, target := range []string{
		"/start?room=../admin",
		"/start?room=engineering&intent=unknown",
		"/start?room=engineering&intent=signup&prompt=select_account",
		"/callback?error=access_denied",
	} {
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
