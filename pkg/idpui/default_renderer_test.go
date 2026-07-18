package idpui_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

func TestDefaultRendererPageShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		page             idpui.InteractionPage
		wantLogin        bool
		wantConsent      bool
		wantActions      []string
		wantFormNoValide bool
	}{
		{name: "login", page: loginPage(), wantLogin: true, wantActions: []string{"continue"}},
		{name: "consent", page: consentPage(), wantConsent: true, wantActions: []string{"approve", "deny"}, wantFormNoValide: true},
		{name: "combined", page: combinedPage(), wantLogin: true, wantConsent: true, wantActions: []string{"approve", "deny"}, wantFormNoValide: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			document := render(t, tt.page)
			form := findElement(t, document, "form", "", "")
			if got := attr(form, "method"); got != "post" {
				t.Fatalf("form method=%q", got)
			}
			if got := attr(form, "action"); got != "https://issuer.example.test/idp/authorize" {
				t.Fatalf("form action=%q", got)
			}
			assertInputValue(t, document, idpui.InteractionFieldName, "opaque-handle")
			assertInputValue(t, document, idpui.CSRFFieldName, "csrf-value")
			if got := findElements(document, "input", "name", idpui.LoginFieldName); (len(got) == 1) != tt.wantLogin {
				t.Fatalf("login inputs=%d wantLogin=%v", len(got), tt.wantLogin)
			}
			if got := findElements(document, "section", "aria-labelledby", "requested-access-heading"); (len(got) == 1) != tt.wantConsent {
				t.Fatalf("consent sections=%d wantConsent=%v", len(got), tt.wantConsent)
			}
			for _, action := range tt.wantActions {
				button := findElement(t, document, "button", "value", action)
				if action == "deny" && tt.wantFormNoValide && !hasAttr(button, "formnovalidate") {
					t.Fatal("deny button does not bypass credential constraint validation")
				}
			}
		})
	}
}

func TestDefaultRendererEscapesUntrustedData(t *testing.T) {
	t.Parallel()
	page := combinedPage()
	page.DocumentTitle = `Sign in </title><script>alert(1)</script>`
	page.Login.LoginValue = `alice" autofocus onfocus="alert(1)`
	page.Consent.ClientID = `</strong><script>alert(2)</script>`
	page.Consent.Scopes = []idpui.Scope{{Name: `openid"><img src=x onerror=alert(3)>`, Description: `<b>admin</b>`}}
	page.Error = &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: `<img src=x onerror=alert(4)> Invalid login or password.`}

	document := render(t, page)
	for _, tag := range []string{"script", "img", "style", "iframe", "object", "embed"} {
		if nodes := findElements(document, tag, "", ""); len(nodes) != 0 {
			t.Fatalf("untrusted data created <%s>", tag)
		}
	}
	walk(document, func(node *html.Node) {
		for _, attribute := range node.Attr {
			if strings.HasPrefix(strings.ToLower(attribute.Key), "on") {
				t.Fatalf("untrusted data created event handler %s", attribute.Key)
			}
		}
	})
}

func TestDefaultRendererNeverPopulatesPassword(t *testing.T) {
	t.Parallel()
	page := loginPage()
	page.Error = &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: "Invalid login or password."}
	document := render(t, page)
	password := findElement(t, document, "input", "name", idpui.PasswordFieldName)
	if hasAttr(password, "value") {
		t.Fatal("password input has a value attribute")
	}
	if attr(password, "aria-invalid") != "true" || attr(password, "aria-describedby") != "interaction-error" {
		t.Fatal("password error is not associated with the public error")
	}
}

func TestDefaultRendererRendersAccountChooser(t *testing.T) {
	t.Parallel()
	page := loginPage()
	page.DocumentTitle = "Choose an account"
	page.Login = nil
	page.AccountChooser = &idpui.AccountChooserPrompt{
		AccountField: idpui.AccountFieldName,
		Entries: []idpui.AccountChooserEntry{
			{Value: "opaque-entry-one", Label: "First account"},
			{Value: "opaque-entry-two", Label: "Second account"},
		},
	}
	document := render(t, page)
	choices := findElements(document, "input", "name", idpui.AccountFieldName)
	if len(choices) != 2 {
		t.Fatalf("account choices=%d", len(choices))
	}
	for _, choice := range choices {
		if attr(choice, "type") != "radio" || !hasAttr(choice, "required") {
			t.Fatalf("unsafe chooser input: %#v", choice.Attr)
		}
		if _, ok := findLabelFor(document, attr(choice, "id")); !ok {
			t.Fatalf("chooser input %q has no label", attr(choice, "id"))
		}
	}
}

// TestCombinedGoldenSemantics keeps the checked-in review example aligned with
// the contract without making whitespace or exact markup a compatibility API.
func TestCombinedGoldenSemantics(t *testing.T) {
	t.Parallel()
	contents, err := os.ReadFile("testdata/combined.golden.html")
	if err != nil {
		t.Fatal(err)
	}
	document, err := html.Parse(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	form := findElement(t, document, "form", "", "")
	if attr(form, "method") != "post" || attr(form, "action") != "https://issuer.example.test/idp/authorize" {
		t.Fatal("golden form no longer demonstrates the interaction contract")
	}
	assertInputValue(t, document, idpui.InteractionFieldName, "opaque-handle")
	assertInputValue(t, document, idpui.CSRFFieldName, "csrf-value")
	findElement(t, document, "button", "value", string(idpui.ActionApprove))
	deny := findElement(t, document, "button", "value", string(idpui.ActionDeny))
	if !hasAttr(deny, "formnovalidate") {
		t.Fatal("golden denial action is blocked by credential constraints")
	}
}

func TestInteractionPageValidateRejectsInvalidContract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(*idpui.InteractionPage)
	}{
		{name: "missing title", edit: func(page *idpui.InteractionPage) { page.DocumentTitle = "" }},
		{name: "unsafe action scheme", edit: func(page *idpui.InteractionPage) { page.Form.ActionURL = "javascript:alert(1)" }},
		{name: "wrong csrf field", edit: func(page *idpui.InteractionPage) { page.Form.CSRFField = "token" }},
		{name: "duplicate action", edit: func(page *idpui.InteractionPage) {
			page.Form.Actions = []idpui.Action{idpui.ActionContinue, idpui.ActionContinue}
		}},
		{name: "unknown action", edit: func(page *idpui.InteractionPage) { page.Form.Actions = []idpui.Action{"accept"} }},
		{name: "unknown reason", edit: func(page *idpui.InteractionPage) { page.Login.Reason = "maybe" }},
		{name: "no prompt", edit: func(page *idpui.InteractionPage) { page.Login = nil }},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page := loginPage()
			tt.edit(&page)
			if err := page.Validate(); err == nil {
				t.Fatal("Validate accepted invalid page")
			}
		})
	}
}

func TestInteractionPageCloneIsDefensive(t *testing.T) {
	t.Parallel()
	page := combinedPage()
	clone := page.Clone()
	clone.Form.Actions[0] = idpui.ActionContinue
	clone.Login.LoginValue = "mallory"
	clone.Consent.Scopes[0].Name = "admin"
	if page.Form.Actions[0] != idpui.ActionApprove || page.Login.LoginValue != "alice" || page.Consent.Scopes[0].Name != "openid" {
		t.Fatal("Clone shares mutable page state")
	}
}

func TestDefaultRendererHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := renderer.RenderInteraction(ctx, io.Discard, loginPage()); err == nil {
		t.Fatal("renderer accepted canceled context")
	}
}

func render(t *testing.T, page idpui.InteractionPage) *html.Node {
	t.Helper()
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := renderer.RenderInteraction(context.Background(), &output, page); err != nil {
		t.Fatal(err)
	}
	document, err := html.Parse(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func loginPage() idpui.InteractionPage {
	return idpui.InteractionPage{
		DocumentTitle: "Sign in",
		Form: idpui.InteractionForm{
			ActionURL:        "https://issuer.example.test/idp/authorize",
			InteractionField: idpui.InteractionFieldName,
			Interaction:      "opaque-handle",
			CSRFField:        idpui.CSRFFieldName,
			CSRFToken:        "csrf-value",
			ActionField:      idpui.ActionFieldName,
			Actions:          []idpui.Action{idpui.ActionContinue},
		},
		Login: &idpui.LoginPrompt{
			Reason:        idpui.LoginReasonSessionMissing,
			LoginField:    idpui.LoginFieldName,
			PasswordField: idpui.PasswordFieldName,
			LoginValue:    "alice",
			Autofocus:     true,
		},
	}
}

func consentPage() idpui.InteractionPage {
	page := loginPage()
	page.DocumentTitle = "Approve access"
	page.Login = nil
	page.Form.Actions = []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}
	page.Consent = &idpui.ConsentPrompt{ClientID: "example-client", Scopes: []idpui.Scope{{Name: "openid"}}}
	return page
}

func combinedPage() idpui.InteractionPage {
	page := loginPage()
	page.Form.Actions = []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}
	page.Consent = &idpui.ConsentPrompt{ClientID: "example-client", Scopes: []idpui.Scope{{Name: "openid"}, {Name: "email", Description: "Read your email address"}}}
	return page
}

func assertInputValue(t *testing.T, root *html.Node, name, value string) {
	t.Helper()
	input := findElement(t, root, "input", "name", name)
	if got := attr(input, "value"); got != value {
		t.Fatalf("input %s value=%q want=%q", name, got, value)
	}
}

func findElement(t *testing.T, root *html.Node, tag, attribute, value string) *html.Node {
	t.Helper()
	nodes := findElements(root, tag, attribute, value)
	if len(nodes) != 1 {
		t.Fatalf("found %d <%s> elements with %s=%q", len(nodes), tag, attribute, value)
	}
	return nodes[0]
}

func findElements(root *html.Node, tag, attribute, value string) []*html.Node {
	var found []*html.Node
	walk(root, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != tag {
			return
		}
		if attribute == "" || attr(node, attribute) == value {
			found = append(found, node)
		}
	})
	return found
}

func findLabelFor(root *html.Node, target string) (*html.Node, bool) {
	for _, label := range findElements(root, "label", "for", target) {
		return label, true
	}
	return nil, false
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}

func attr(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func hasAttr(node *html.Node, name string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return true
		}
	}
	return false
}
