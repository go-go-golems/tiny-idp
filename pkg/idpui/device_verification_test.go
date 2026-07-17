package idpui_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

func TestDefaultRendererRendersDeviceEntryConfirmationAndNotice(t *testing.T) {
	t.Parallel()
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		page  idpui.DeviceVerificationPage
		check func(*testing.T, *html.Node)
	}{
		{name: "entry", page: deviceEntryPage(), check: func(t *testing.T, document *html.Node) {
			form := findElement(t, document, "form", "", "")
			if attr(form, "method") != "get" || attr(form, "action") != "https://issuer.example.test/device" {
				t.Fatalf("entry form %#v", form.Attr)
			}
			findElement(t, document, "input", "name", idpui.UserCodeFieldName)
		}},
		{name: "confirmation", page: deviceConfirmationPage(), check: func(t *testing.T, document *html.Node) {
			form := findElement(t, document, "form", "", "")
			if attr(form, "method") != "post" {
				t.Fatalf("confirmation method=%q", attr(form, "method"))
			}
			assertInputValue(t, document, idpui.InteractionFieldName, "device-handle")
			assertInputValue(t, document, idpui.CSRFFieldName, "csrf-value")
			deny := findElement(t, document, "button", "value", string(idpui.ActionDeny))
			if !hasAttr(deny, "formnovalidate") {
				t.Fatal("deny button must be available even if credentials are empty")
			}
		}},
		{name: "notice", page: deviceNoticePage(), check: func(t *testing.T, document *html.Node) {
			if len(findElements(document, "form", "", "")) != 0 {
				t.Fatal("terminal device notice unexpectedly has a form")
			}
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			if err := renderer.RenderDeviceVerification(context.Background(), &output, test.page); err != nil {
				t.Fatal(err)
			}
			document, err := html.Parse(bytes.NewReader(output.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			test.check(t, document)
		})
	}
}

func TestDefaultRendererEscapesDeviceVerificationValues(t *testing.T) {
	t.Parallel()
	page := deviceConfirmationPage()
	page.Confirmation.ClientID = `</strong><script>alert(1)</script>`
	page.Confirmation.Login.LoginValue = `alice" autofocus onfocus="alert(2)`
	page.Confirmation.Scopes = []idpui.Scope{{Name: `openid"><img src=x onerror=alert(3)>`}}
	page.Error = &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: `<img src=x onerror=alert(4)>`}
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := renderer.RenderDeviceVerification(context.Background(), &output, page); err != nil {
		t.Fatal(err)
	}
	document, err := html.Parse(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
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

func TestDeviceVerificationPageValidateRejectsInvalidContracts(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		edit func(*idpui.DeviceVerificationPage)
	}{
		{name: "multiple prompts", edit: func(page *idpui.DeviceVerificationPage) {
			page.Entry = &idpui.DeviceCodeEntryPrompt{UserCodeField: idpui.UserCodeFieldName}
		}},
		{name: "wrong code field", edit: func(page *idpui.DeviceVerificationPage) {
			page.Confirmation = nil
			page.Entry = &idpui.DeviceCodeEntryPrompt{UserCodeField: "code"}
		}},
		{name: "missing csrf", edit: func(page *idpui.DeviceVerificationPage) { page.Form.CSRFToken = "" }},
		{name: "continue action", edit: func(page *idpui.DeviceVerificationPage) { page.Form.Actions = []idpui.Action{idpui.ActionContinue} }},
		{name: "unsafe action URL", edit: func(page *idpui.DeviceVerificationPage) { page.Form.ActionURL = "javascript:alert(1)" }},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			page := deviceConfirmationPage()
			test.edit(&page)
			if err := page.Validate(); err == nil {
				t.Fatal("Validate accepted invalid device verification page")
			}
		})
	}
}

func TestDefaultDeviceRendererHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := renderer.RenderDeviceVerification(ctx, io.Discard, deviceEntryPage()); err == nil {
		t.Fatal("renderer accepted a canceled context")
	}
}

func deviceEntryPage() idpui.DeviceVerificationPage {
	return idpui.DeviceVerificationPage{DocumentTitle: "Verify your device", Form: idpui.DeviceVerificationForm{ActionURL: "https://issuer.example.test/device"}, Entry: &idpui.DeviceCodeEntryPrompt{UserCodeField: idpui.UserCodeFieldName}}
}

func deviceConfirmationPage() idpui.DeviceVerificationPage {
	return idpui.DeviceVerificationPage{DocumentTitle: "Approve device access", Form: idpui.DeviceVerificationForm{ActionURL: "https://issuer.example.test/device", InteractionField: idpui.InteractionFieldName, Interaction: "device-handle", CSRFField: idpui.CSRFFieldName, CSRFToken: "csrf-value", ActionField: idpui.ActionFieldName, Actions: []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}}, Confirmation: &idpui.DeviceConfirmationPrompt{ClientID: "device-cli", Scopes: []idpui.Scope{{Name: "openid"}}, Login: idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName}}}
}

func deviceNoticePage() idpui.DeviceVerificationPage {
	return idpui.DeviceVerificationPage{DocumentTitle: "Device verification complete", Form: idpui.DeviceVerificationForm{ActionURL: "https://issuer.example.test/device"}, Notice: &idpui.DeviceVerificationNotice{Summary: "The device request was approved."}}
}
