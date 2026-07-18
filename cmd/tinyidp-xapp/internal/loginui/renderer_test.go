package loginui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpui/idpuitest"
)

func TestRendererProducesThemedAccessibleInteraction(t *testing.T) {
	renderer, err := New(Options{ProductName: "Tiny BBS"})
	if err != nil {
		t.Fatal(err)
	}
	page := testPage()
	page.Login = &idpui.LoginPrompt{Reason: idpui.LoginReasonPromptLogin, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName, LoginValue: `alice"><script>`, Autofocus: true}
	page.Error = &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: `Nope <img src=x>`}

	var output bytes.Buffer
	if err := renderer.RenderInteraction(context.Background(), &output, page); err != nil {
		t.Fatal(err)
	}
	html := output.String()
	for _, fragment := range []string{
		`href="/static/tinyidp/login.css"`,
		`aria-labelledby="interaction-heading"`,
		`autocomplete="username"`,
		`autocomplete="current-password"`,
		`role="alert"`,
		`formnovalidate`,
	} {
		if !strings.Contains(html, fragment) {
			t.Errorf("rendered HTML missing %q", fragment)
		}
	}
	if strings.Contains(html, `<script>`) || strings.Contains(html, `<img src=x>`) {
		t.Fatalf("dynamic HTML was not escaped: %s", html)
	}
	if strings.Contains(html, `type="password" value=`) {
		t.Fatal("password input retained a value")
	}
}

func TestRendererSupportsEveryInteractionShape(t *testing.T) {
	renderer, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		mutate      func(*idpui.InteractionPage)
		want        []string
		wantMissing []string
	}{
		{
			name: "missing session login only",
			mutate: func(page *idpui.InteractionPage) {
				page.Consent = nil
				page.Form.Actions = []idpui.Action{idpui.ActionContinue}
				page.Login = &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName}
			},
			want:        []string{"Sign in to continue.", `value="continue"`, `type="password"`},
			wantMissing: []string{"Requested access", `value="deny"`},
		},
		{
			name: "max age forced login",
			mutate: func(page *idpui.InteractionPage) {
				page.Login = &idpui.LoginPrompt{Reason: idpui.LoginReasonMaxAge, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName}
			},
			want: []string{"previous authentication is too old", `value="approve"`, `formnovalidate`},
		},
		{
			name: "consent only",
			mutate: func(page *idpui.InteractionPage) {
				page.Login = nil
			},
			want:        []string{"Requested access", "tinyidp-xapp", `value="deny"`},
			wantMissing: []string{`type="password"`, "Sign in to continue."},
		},
		{
			name: "consent error",
			mutate: func(page *idpui.InteractionPage) {
				page.Login = nil
				page.Error = &idpui.PublicError{Code: idpui.ErrorConsentRequired, Field: idpui.FieldConsent, Summary: "Choose approve or deny."}
			},
			want: []string{`role="alert"`, `aria-describedby="interaction-error"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := testPage()
			tt.mutate(&page)
			var output bytes.Buffer
			if err := renderer.RenderInteraction(context.Background(), &output, page); err != nil {
				t.Fatal(err)
			}
			for _, fragment := range tt.want {
				if !strings.Contains(output.String(), fragment) {
					t.Errorf("rendered HTML missing %q: %s", fragment, output.String())
				}
			}
			for _, fragment := range tt.wantMissing {
				if strings.Contains(output.String(), fragment) {
					t.Errorf("rendered HTML unexpectedly contains %q: %s", fragment, output.String())
				}
			}
		})
	}
}

func TestStylesheetHandler(t *testing.T) {
	renderer, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	renderer.AssetsHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://x.test/static/tinyidp/login.css", nil))
	if recorder.Code != http.StatusOK || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/css") || !strings.Contains(recorder.Body.String(), "--mint") {
		t.Fatalf("stylesheet response status=%d headers=%v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
	}

	notFound := httptest.NewRecorder()
	renderer.AssetsHandler().ServeHTTP(notFound, httptest.NewRequest(http.MethodGet, "http://x.test/static/tinyidp/missing.css", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("missing asset status=%d", notFound.Code)
	}
}

func TestStylesheetURLValidation(t *testing.T) {
	for _, invalid := range []string{
		"https://cdn.example.test/login.css",
		"//cdn.example.test/login.css",
		"static/login.css",
		"/assets/login.css",
		"/static/../login.css",
		"/static/login.css?version=1",
		"/static/login.css#theme",
		`/static\\login.css`,
	} {
		t.Run(invalid, func(t *testing.T) {
			if _, err := New(Options{StylesheetURL: invalid}); err == nil {
				t.Fatalf("New accepted unsafe stylesheet URL %q", invalid)
			}
		})
	}
	if _, err := New(Options{StylesheetURL: "/static/brand/login.css"}); err != nil {
		t.Fatalf("New rejected safe stylesheet URL: %v", err)
	}
}

func TestRendererConformsToTinyIDPTrustBoundary(t *testing.T) {
	renderer, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	page := testPage()
	page.Login = &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName}
	_, violations, err := idpuitest.RenderAndCheck(context.Background(), renderer, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("xapp renderer violations: %v", violations)
	}
}

func testPage() idpui.InteractionPage {
	return idpui.InteractionPage{
		DocumentTitle: "Sign in and approve access",
		Form: idpui.InteractionForm{
			ActionURL:        "https://app.example.test/idp/authorize",
			InteractionField: idpui.InteractionFieldName,
			Interaction:      "interaction-value",
			CSRFField:        idpui.CSRFFieldName,
			CSRFToken:        "csrf-value",
			ActionField:      idpui.ActionFieldName,
			Actions:          []idpui.Action{idpui.ActionApprove, idpui.ActionDeny},
		},
		Consent: &idpui.ConsentPrompt{ClientID: "tinyidp-xapp", Scopes: []idpui.Scope{{Name: "openid"}, {Name: "profile"}}},
	}
}
