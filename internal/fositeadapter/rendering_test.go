package fositeadapter_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/manuel/tinyidp/internal/fositeadapter"
	"github.com/manuel/tinyidp/internal/keys"
	"github.com/manuel/tinyidp/internal/store/memory"
	"github.com/manuel/tinyidp/pkg/idp"
	idpstore "github.com/manuel/tinyidp/pkg/idpstore"
	"github.com/manuel/tinyidp/pkg/idpui"
)

type rendererFunc func(context.Context, io.Writer, idpui.InteractionPage) error

func (f rendererFunc) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	return f(ctx, dst, page)
}

type recordingRenderer struct {
	mu       sync.Mutex
	delegate idpui.InteractionRenderer
	pages    []idpui.InteractionPage
}

func newRecordingRenderer(t *testing.T) *recordingRenderer {
	t.Helper()
	delegate, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	return &recordingRenderer{delegate: delegate}
}

func (r *recordingRenderer) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	r.mu.Lock()
	r.pages = append(r.pages, page.Clone())
	r.mu.Unlock()
	return r.delegate.RenderInteraction(ctx, dst, page)
}

func (r *recordingRenderer) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pages = nil
}

func (r *recordingRenderer) last(t *testing.T) idpui.InteractionPage {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.pages) == 0 {
		t.Fatal("renderer was not invoked")
	}
	return r.pages[len(r.pages)-1].Clone()
}

func TestInteractionRendererReceivesTypedLoginReasons(t *testing.T) {
	renderer := newRecordingRenderer(t)
	fixture := newInteractionFixtureWithRenderer(t, nil, renderer)

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("login_hint", `Alice <alice@example.test>`)
	_, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("initial status=%d", status)
	}
	page := renderer.last(t)
	if page.Login == nil || page.Login.Reason != idpui.LoginReasonSessionMissing || page.Login.LoginValue != `Alice <alice@example.test>` {
		t.Fatalf("initial page=%#v", page)
	}
	if page.Consent == nil || page.Consent.ClientID != "spa" || len(page.Consent.Scopes) != 1 {
		t.Fatalf("initial consent=%#v", page.Consent)
	}

	fixture.login()
	renderer.reset()
	request = authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("prompt", "login")
	_, _, status = fixture.begin(request)
	if status != http.StatusOK || renderer.last(t).Login.Reason != idpui.LoginReasonPromptLogin {
		t.Fatalf("prompt=login status=%d page=%#v", status, renderer.last(t))
	}

	renderer.reset()
	request.Del("prompt")
	request.Set("max_age", "0")
	_, _, status = fixture.begin(request)
	if status != http.StatusOK || renderer.last(t).Login.Reason != idpui.LoginReasonMaxAge {
		t.Fatalf("max_age status=%d page=%#v", status, renderer.last(t))
	}
}

func TestInteractionRendererReceivesConsentOnlyPage(t *testing.T) {
	renderer := newRecordingRenderer(t)
	fixture := newInteractionFixtureWithRenderer(t, func(store *memory.Store) idp.ConsentPolicy {
		return fositeadapter.NewStoredConsent(store, 0)
	}, renderer)
	fixture.login()
	renderer.reset()

	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	request.Set("scope", "openid email")
	_, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("consent begin status=%d", status)
	}
	page := renderer.last(t)
	if page.Login != nil || page.Consent == nil || len(page.Consent.Scopes) != 2 {
		t.Fatalf("consent-only page=%#v", page)
	}
}

func TestRecoverableLoginErrorRendersPendingInteraction(t *testing.T) {
	renderer := newRecordingRenderer(t)
	fixture := newInteractionFixtureWithRenderer(t, nil, renderer)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	renderer.reset()
	resp := fixture.submit(form)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), `name="password"`) || !strings.Contains(string(body), `role="alert"`) {
		t.Fatalf("retry status=%d body=%s", resp.StatusCode, body)
	}
	page := renderer.last(t)
	if page.Error == nil || page.Error.Code != idpui.ErrorMissingLogin || page.Form.Interaction == "" || page.Form.CSRFToken == "" {
		t.Fatalf("retry page=%#v", page)
	}
}

func TestCredentialPostUsesSeeOther(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, status := fixture.begin(request)
	if status != http.StatusOK {
		t.Fatalf("begin status=%d", status)
	}
	form.Set("login", "alice")
	resp := fixture.submit(form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("credential POST status=%d, want 303", resp.StatusCode)
	}
}

func TestUnknownInteractionActionFailsClosed(t *testing.T) {
	fixture := newInteractionFixture(t, nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	form, _, _ := fixture.begin(request)
	form.Set("login", "alice")
	form.Set("action", "continue")
	resp := fixture.submit(form)
	defer resp.Body.Close()
	assertNoAuthorizationCode(t, resp)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown action status=%d", resp.StatusCode)
	}
}

func TestRendererFailureAndSizeLimitFailBeforeHTMLCommit(t *testing.T) {
	tests := []struct {
		name     string
		renderer idpui.InteractionRenderer
		reason   string
	}{
		{name: "failure", renderer: rendererFunc(func(context.Context, io.Writer, idpui.InteractionPage) error { return errors.New("template exploded") }), reason: "renderer_failed"},
		{name: "empty", renderer: rendererFunc(func(context.Context, io.Writer, idpui.InteractionPage) error { return nil }), reason: "empty_document"},
		{name: "oversize", renderer: rendererFunc(func(_ context.Context, dst io.Writer, _ idpui.InteractionPage) error {
			// A custom renderer may accidentally discard the writer error. The
			// provider must still observe that the fixed document bound was hit.
			_, _ = dst.Write(bytes.Repeat([]byte("x"), 300<<10))
			return nil
		}), reason: "document_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audit := idp.NewMemorySink()
			provider := newRendererTestProvider(t, audit, tt.renderer)
			request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			request.Del("login")
			recorder := doAuthorizeBegin(t, provider.Handler(), request)
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Fatalf("failure content type=%q", got)
			}
			if strings.Contains(recorder.Body.String(), "template exploded") {
				t.Fatal("internal renderer error leaked")
			}
			events := audit.Events()
			if len(events) == 0 || events[len(events)-1].Name != "interaction.render_failed" || events[len(events)-1].Reason != tt.reason {
				t.Fatalf("audit events=%#v", events)
			}
			stats := provider.InteractionRenderStats()
			if stats.Attempts != 1 || stats.Successes != 0 || stats.Failures != 1 || stats.TotalLatency <= 0 || stats.MaxLatency <= 0 {
				t.Fatalf("render stats=%#v", stats)
			}
			if tt.reason == "document_too_large" && stats.OversizedDocuments != 1 {
				t.Fatalf("oversized render stats=%#v", stats)
			}
			if tt.reason == "empty_document" && stats.EmptyDocuments != 1 {
				t.Fatalf("empty render stats=%#v", stats)
			}
		})
	}
}

func TestInteractionCSPAllowsOnlySameOriginStyles(t *testing.T) {
	provider := newRendererTestProvider(t, idp.NewMemorySink(), nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	recorder := doAuthorizeBegin(t, provider.Handler(), request)
	want := "default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self' http://127.0.0.1:5556; base-uri 'none'"
	if got := recorder.Header().Get("Content-Security-Policy"); got != want {
		t.Fatalf("CSP=%q want=%q", got, want)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("cache headers=%v", recorder.Header())
	}
	stats := provider.InteractionRenderStats()
	if stats.Attempts != 1 || stats.Successes != 1 || stats.Failures != 0 || stats.TotalLatency <= 0 {
		t.Fatalf("successful render stats=%#v", stats)
	}
}

func TestInteractionRenderResponseWriteFailureIsCounted(t *testing.T) {
	provider := newRendererTestProvider(t, idp.NewMemorySink(), nil)
	request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	request.Del("login")
	httpRequest := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:5556/authorize?"+request.Encode(), nil)
	writer := &failingResponseWriter{header: make(http.Header)}
	provider.Handler().ServeHTTP(writer, httpRequest)
	stats := provider.InteractionRenderStats()
	if stats.Attempts != 1 || stats.Successes != 0 || stats.Failures != 1 || stats.ResponseWriteFailures != 1 {
		t.Fatalf("response write render stats=%#v", stats)
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header { return w.header }
func (*failingResponseWriter) WriteHeader(int)       {}
func (*failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("response disconnected")
}

func newRendererTestProvider(t *testing.T, audit idp.Sink, renderer idpui.InteractionRenderer) *fositeadapter.Provider {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	if err := store.PutClient(ctx, idpstore.Client{ID: "spa", Public: true, RequirePKCE: true, RedirectURIs: []string{"http://localhost/callback"}, AllowedScopes: []string{"openid"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.PutUser(ctx, "alice", idpstore.User{ID: "u1", Sub: "user-alice"}); err != nil {
		t.Fatal(err)
	}
	key, err := keys.GenerateRSA("renderer-test", timeNow())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateSigningKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	provider, err := fositeadapter.NewProvider(ctx, fositeadapter.Options{Issuer: "http://127.0.0.1:5556", Store: store, SecretKey: []byte("renderer-test-secret-key-32-bytes"), Audit: audit, InteractionRenderer: renderer})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func doAuthorizeBegin(t *testing.T, handler http.Handler, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:5556/authorize?"+values.Encode(), nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func timeNow() time.Time { return time.Now().UTC() }
