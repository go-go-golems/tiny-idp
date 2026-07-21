package idpui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

func TestBrowserErrorPageIsTerminalAndEscapesPublicText(t *testing.T) {
	renderer, err := idpui.NewDefaultRenderer()
	if err != nil {
		t.Fatal(err)
	}
	page := idpui.BrowserErrorPage{
		DocumentTitle: "Registration rejected",
		ClientID:      "message-desk",
		Heading:       "Registration could not be completed",
		Summary:       `<script>alert("unsafe")</script> Restart registration.`,
	}
	var output bytes.Buffer
	if err := renderer.RenderBrowserError(context.Background(), &output, page); err != nil {
		t.Fatal(err)
	}
	document := output.String()
	if strings.Contains(document, "<script>") || !strings.Contains(document, "&lt;script&gt;") {
		t.Fatalf("browser error output did not escape text: %s", document)
	}
	for _, forbidden := range []string{"<form", "csrf_token", "interaction", "redirect_uri"} {
		if strings.Contains(document, forbidden) {
			t.Fatalf("browser error output contains terminally forbidden %q: %s", forbidden, document)
		}
	}
}

func TestBrowserErrorPageRejectsInvalidModels(t *testing.T) {
	valid := idpui.BrowserErrorPage{DocumentTitle: "Registration rejected", ClientID: "message-desk", Heading: "Registration failed", Summary: "Restart registration."}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	valid.ClientID = ""
	if err := valid.Validate(); err == nil {
		t.Fatal("missing client ID was accepted")
	}
}
