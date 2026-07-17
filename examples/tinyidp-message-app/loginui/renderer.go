// Package loginui renders the provider-owned login and consent interaction.
// It intentionally owns presentation only; tiny-idp still owns protocol state.
package loginui

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

const StylesheetURL = "/static/tinyidp/login.css"

//go:embed templates/interaction.html
var interactionTemplate string

//go:embed static/login.css
var stylesheet []byte

type Renderer struct{ template *template.Template }

var _ idpui.InteractionRenderer = (*Renderer)(nil)

func New() (*Renderer, error) {
	t, err := template.New("interaction").Parse(interactionTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse message desk interaction template: %w", err)
	}
	return &Renderer{template: t}, nil
}
func (r *Renderer) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	if r == nil || r.template == nil || dst == nil {
		return fmt.Errorf("message desk interaction renderer is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate interaction page: %w", err)
	}
	return r.template.ExecuteTemplate(dst, "interaction", struct {
		Page           idpui.InteractionPage
		CredentialsBad bool
		ConsentBad     bool
	}{Page: page.Clone(), CredentialsBad: page.Error != nil && page.Error.Field == idpui.FieldCredentials, ConsentBad: page.Error != nil && page.Error.Field == idpui.FieldConsent})
}
func (r *Renderer) AssetsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		if q.URL.Path != StylesheetURL || (q.Method != http.MethodGet && q.Method != http.MethodHead) {
			http.NotFound(w, q)
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeContent(w, q, path.Base(StylesheetURL), time.Time{}, bytes.NewReader(stylesheet))
	})
}
