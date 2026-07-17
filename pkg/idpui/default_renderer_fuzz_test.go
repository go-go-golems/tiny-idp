package idpui_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
	"github.com/go-go-golems/tiny-idp/pkg/idpui/idpuitest"
)

func FuzzDefaultRendererEscapingAndConformance(f *testing.F) {
	f.Add("Alice", "tinyidp-xapp", "openid", "Invalid login or password.")
	f.Add(`<script>alert(1)</script>`, `client"><img src=x>`, `scope onmouseover=x`, `<style>body{display:none}</style>`)
	f.Add("Joséphine 👩🏽‍💻", "掲示板", "résumé:read", "Пароль недействителен")
	f.Add(string([]byte{0xff, 0xfe, '<', 'x', '>'}), "client\x00name", strings.Repeat("界", 4096), "error")
	f.Fuzz(func(t *testing.T, login, clientID, scope, summary string) {
		if strings.TrimSpace(clientID) == "" {
			clientID = "client"
		}
		if strings.TrimSpace(summary) == "" {
			summary = "Authentication failed."
		}
		renderer, err := idpui.NewDefaultRenderer()
		if err != nil {
			t.Fatal(err)
		}
		page := fuzzPage(login, clientID, scope, summary)
		document, violations, err := idpuitest.RenderAndCheck(context.Background(), renderer, page)
		if err != nil {
			t.Fatal(err)
		}
		if len(document) == 0 || len(violations) != 0 {
			t.Fatalf("document=%d violations=%v", len(document), violations)
		}
	})
}

func fuzzPage(login, clientID, scope, summary string) idpui.InteractionPage {
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
		Login:   &idpui.LoginPrompt{Reason: idpui.LoginReasonPromptLogin, LoginField: idpui.LoginFieldName, PasswordField: idpui.PasswordFieldName, LoginValue: login},
		Consent: &idpui.ConsentPrompt{ClientID: clientID, Scopes: []idpui.Scope{{Name: scope}}},
		Error:   &idpui.PublicError{Code: idpui.ErrorInvalidCredentials, Field: idpui.FieldCredentials, Summary: summary},
	}
}
