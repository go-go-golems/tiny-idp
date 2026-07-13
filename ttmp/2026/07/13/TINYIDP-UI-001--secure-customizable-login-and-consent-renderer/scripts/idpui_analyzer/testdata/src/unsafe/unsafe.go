package unsafe

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	texttemplate "text/template" // want "use html/template, not text/template, for interaction HTML"
)

var _ = texttemplate.HTMLEscapeString

func trusted(value string) template.HTML {
	return template.HTML(value) // want "trusted-content conversion template.HTML is forbidden"
}

func direct(dst io.Writer) {
	fmt.Fprintf(dst, `<form><input name="password"></form>`) // want "direct HTML string construction is forbidden"
}

func RenderInteraction(_ context.Context, w http.ResponseWriter, _ *http.Request) error { // want "RenderInteraction must not accept http.ResponseWriter or \*http.Request"
	_, _ = w.Write([]byte(`<html></html>`)) // want "direct HTML string construction is forbidden"
	return nil
}
