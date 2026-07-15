package loginui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/manuel/tinyidp/pkg/idpui"
	"github.com/manuel/tinyidp/pkg/idpui/idpuitest"
)

func TestRendererConformsToInteractionContract(t *testing.T) {
	renderer, err := New()
	if err != nil {
		t.Fatal(err)
	}
	page := idpui.InteractionPage{DocumentTitle: "Sign in", Form: idpui.InteractionForm{ActionURL: "https://issuer.example.test/idp/authorize", InteractionField: idpui.InteractionFieldName, Interaction: "opaque", CSRFField: idpui.CSRFFieldName, CSRFToken: "csrf", ActionField: idpui.ActionFieldName, Actions: []idpui.Action{idpui.ActionApprove, idpui.ActionDeny}}, Login: &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName, Autofocus: true}, Consent: &idpui.ConsentPrompt{ClientID: "message-desk", Scopes: []idpui.Scope{{Name: "openid"}}}}
	_, violations, err := idpuitest.RenderAndCheck(context.Background(), renderer, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("renderer violations = %#v", violations)
	}
}

func TestRendererRendersAccountChooserControls(t *testing.T) {
	renderer, err := New()
	if err != nil {
		t.Fatal(err)
	}
	page := idpui.InteractionPage{
		DocumentTitle: "Choose an account",
		Form: idpui.InteractionForm{
			ActionURL:        "https://issuer.example.test/idp/authorize",
			InteractionField: idpui.InteractionFieldName,
			Interaction:      "opaque",
			CSRFField:        idpui.CSRFFieldName,
			CSRFToken:        "csrf",
			ActionField:      idpui.ActionFieldName,
			Actions:          []idpui.Action{idpui.ActionContinue, idpui.ActionUseAnotherAccount, idpui.ActionDeny},
		},
		AccountChooser: &idpui.AccountChooserPrompt{AccountField: idpui.AccountFieldName, Entries: []idpui.AccountChooserEntry{{Value: "opaque-account-handle", Label: "Amelie"}}},
	}
	var rendered bytes.Buffer
	if err := renderer.RenderInteraction(context.Background(), &rendered, page); err != nil {
		t.Fatal(err)
	}
	html := rendered.String()
	for _, required := range []string{`type="radio"`, `name="account"`, `value="opaque-account-handle"`, "Amelie", "CHOOSE AN ACCOUNT", `value="use_another_account"`, "Use another account"} {
		if !strings.Contains(html, required) {
			t.Fatalf("account chooser is missing %q: %s", required, html)
		}
	}
}
