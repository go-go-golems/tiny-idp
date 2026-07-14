package loginui

import (
	"context"
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
