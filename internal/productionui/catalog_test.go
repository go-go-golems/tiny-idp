package productionui

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/internal/productionconfig"
	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

func TestCatalogSelectsPerClientThemesAndServesOnlyApprovedCSS(t *testing.T) {
	catalog := testCatalog(t)
	message, err := catalog.Resolve("message-desk")
	if err != nil {
		t.Fatal(err)
	}
	if message.Name != "message" || message.ProductName != "Message Desk" || message.StylesheetRoute != "/static/themes/message.css" {
		t.Fatalf("message theme = %#v", message)
	}
	fallback, err := catalog.Resolve("unmapped-client")
	if err != nil {
		t.Fatal(err)
	}
	if fallback.Name != "shared" {
		t.Fatalf("fallback theme = %#v", fallback)
	}

	response := httptest.NewRecorder()
	catalog.AssetsHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "https://idp.example/static/themes/message.css", nil))
	if response.Code != http.StatusOK || response.Body.String() != "body{color:navy}" {
		t.Fatalf("asset response = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "text/css; charset=utf-8" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("asset headers = %#v", response.Header())
	}
	missing := httptest.NewRecorder()
	catalog.AssetsHandler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "https://idp.example/static/themes/unknown.css", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.Code)
	}
	post := httptest.NewRecorder()
	catalog.AssetsHandler().ServeHTTP(post, httptest.NewRequest(http.MethodPost, "https://idp.example/static/themes/message.css", nil))
	if post.Code != http.StatusNotFound {
		t.Fatalf("POST status = %d", post.Code)
	}
}

func TestCatalogRejectsUnsafeThemeDeclarations(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "safe.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	clients := testClients(t)
	tests := []struct {
		name     string
		document CatalogDocument
		contains string
	}{
		{name: "traversal", document: CatalogDocument{Version: 1, DefaultTheme: "shared", Themes: map[string]ThemeConfig{"shared": {ProductName: "Shared", Stylesheet: "../safe.css"}}}, contains: "CSS basename"},
		{name: "external URL", document: CatalogDocument{Version: 1, DefaultTheme: "shared", Themes: map[string]ThemeConfig{"shared": {ProductName: "Shared", Stylesheet: "https://evil.example/x.css"}}}, contains: "CSS basename"},
		{name: "unknown client", document: CatalogDocument{Version: 1, DefaultTheme: "shared", Themes: map[string]ThemeConfig{"shared": {ProductName: "Shared", Stylesheet: "safe.css"}}, ClientThemes: map[string]string{"evil": "shared"}}, contains: "undeclared client"},
		{name: "unknown theme", document: CatalogDocument{Version: 1, DefaultTheme: "shared", Themes: map[string]ThemeConfig{"shared": {ProductName: "Shared", Stylesheet: "safe.css"}}, ClientThemes: map[string]string{"message-desk": "missing"}}, contains: "undeclared theme"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCatalog(directory, tt.document, clients)
			if err == nil || !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error = %v, want %q", err, tt.contains)
			}
		})
	}
}

func TestLoadCatalogRejectsDuplicateThemeNames(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "safe.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(directory, "themes.json")
	source := `{"version":1,"defaultTheme":"shared","themes":{"shared":{"productName":"First","stylesheet":"safe.css"},"shared":{"productName":"Second","stylesheet":"safe.css"}},"clientThemes":{}}`
	if err := os.WriteFile(catalogPath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCatalog(directory, catalogPath, testClients(t))
	if err == nil || !strings.Contains(err.Error(), `duplicate object key "shared"`) {
		t.Fatalf("duplicate theme error = %v", err)
	}
}

func TestRendererUsesClientThemeForInteractionAndWorkflow(t *testing.T) {
	renderer, err := NewRenderer(testCatalog(t))
	if err != nil {
		t.Fatal(err)
	}
	interaction := idpui.InteractionPage{
		DocumentTitle: "Sign in",
		ClientID:      "message-desk",
		Form: idpui.InteractionForm{
			ActionURL: "https://idp.example/authorize", RedirectOrigin: "https://message.example",
			InteractionField: idpui.InteractionFieldName, Interaction: strings.Repeat("i", 32),
			CSRFField: idpui.CSRFFieldName, CSRFToken: strings.Repeat("c", 32),
			ActionField: idpui.ActionFieldName, Actions: []idpui.Action{idpui.ActionContinue},
		},
		Login: &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName},
	}
	var interactionOutput bytes.Buffer
	if err := renderer.RenderInteraction(context.Background(), &interactionOutput, interaction); err != nil {
		t.Fatal(err)
	}
	if output := interactionOutput.String(); !strings.Contains(output, "Message Desk") || !strings.Contains(output, `/static/themes/message.css`) {
		t.Fatalf("interaction output = %s", output)
	}

	registry := idpworkflow.DefaultRegistry()
	email, _ := registry.Field(idpworkflow.FieldEmail)
	password, _ := registry.Field(idpworkflow.FieldPassword)
	submit, _ := registry.Action(idpworkflow.ActionSubmit)
	workflow := idpui.WorkflowPage{
		DocumentTitle: "Create an account", ClientID: "goja-auth",
		Form:   idpui.WorkflowForm{ActionURL: "https://idp.example/authorize", RedirectOrigin: "https://goja.example", InteractionField: idpui.InteractionFieldName, Interaction: strings.Repeat("i", 32), ContinuationField: idpui.WorkflowContinuationFieldName, Continuation: strings.Repeat("n", 32), CSRFField: idpui.CSRFFieldName, CSRFToken: strings.Repeat("c", 32), ActionField: idpui.ActionFieldName},
		Fields: []idpui.WorkflowField{{Descriptor: email}, {Descriptor: password}}, Actions: []idpui.WorkflowAction{{Descriptor: submit}},
	}
	var workflowOutput bytes.Buffer
	if err := renderer.RenderWorkflow(context.Background(), &workflowOutput, workflow); err != nil {
		t.Fatal(err)
	}
	if output := workflowOutput.String(); !strings.Contains(output, "Goja Auth Lab") || !strings.Contains(output, `/static/themes/goja.css`) || !strings.Contains(output, `name="password" type="password" value=""`) {
		t.Fatalf("workflow output = %s", output)
	}

	errorPage := idpui.BrowserErrorPage{DocumentTitle: "Registration rejected", ClientID: "message-desk", Heading: "Registration could not be completed", Summary: "Restart registration from the application."}
	var errorOutput bytes.Buffer
	if err := renderer.RenderBrowserError(context.Background(), &errorOutput, errorPage); err != nil {
		t.Fatal(err)
	}
	if output := errorOutput.String(); !strings.Contains(output, "Message Desk") || !strings.Contains(output, `/static/themes/message.css`) || strings.Contains(output, "<form") {
		t.Fatalf("browser error output = %s", output)
	}
}

func TestRendererEscapesCatalogPresentation(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "safe.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(directory, CatalogDocument{
		Version: 1, DefaultTheme: "shared",
		Themes: map[string]ThemeConfig{"shared": {ProductName: `<script>alert("theme")</script>`, Stylesheet: "safe.css"}},
	}, testClients(t))
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := NewRenderer(catalog)
	if err != nil {
		t.Fatal(err)
	}
	page := idpui.InteractionPage{
		DocumentTitle: "Sign in", ClientID: "message-desk",
		Form:  idpui.InteractionForm{ActionURL: "https://idp.example/authorize", RedirectOrigin: "https://message.example", InteractionField: idpui.InteractionFieldName, Interaction: strings.Repeat("i", 32), CSRFField: idpui.CSRFFieldName, CSRFToken: strings.Repeat("c", 32), ActionField: idpui.ActionFieldName, Actions: []idpui.Action{idpui.ActionContinue}},
		Login: &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName},
	}
	var output bytes.Buffer
	if err := renderer.RenderInteraction(context.Background(), &output, page); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), `<script>`) || !strings.Contains(output.String(), `&lt;script&gt;`) {
		t.Fatalf("unescaped output = %s", output.String())
	}
}

func testCatalog(t *testing.T) *Catalog {
	t.Helper()
	directory := t.TempDir()
	for name, css := range map[string]string{"shared.css": "body{color:black}", "message.css": "body{color:navy}", "goja.css": "body{color:purple}"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(css), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := NewCatalog(directory, CatalogDocument{
		Version: 1, DefaultTheme: "shared",
		Themes: map[string]ThemeConfig{
			"shared":  {ProductName: "TinyIDP", Stylesheet: "shared.css"},
			"message": {ProductName: "Message Desk", Stylesheet: "message.css"},
			"goja":    {ProductName: "Goja Auth Lab", Stylesheet: "goja.css"},
		},
		ClientThemes: map[string]string{"message-desk": "message", "goja-auth": "goja"},
	}, testClients(t))
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func testClients(t *testing.T) *productionconfig.ClientCatalog {
	t.Helper()
	catalog, err := productionconfig.NewClientCatalog(productionconfig.ClientCatalogDocument{Version: 1, Clients: []productionconfig.BrowserClientConfig{
		{ID: "message-desk", Profile: "browser", RedirectURIs: []string{"https://message.example/auth/callback"}, PostLogoutRedirectURIs: []string{"https://message.example/"}, AllowedScopes: []string{"openid", "profile"}},
		{ID: "goja-auth", Profile: "browser", RedirectURIs: []string{"https://goja.example/auth/callback"}, PostLogoutRedirectURIs: []string{"https://goja.example/"}, AllowedScopes: []string{"openid", "profile", "email"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
