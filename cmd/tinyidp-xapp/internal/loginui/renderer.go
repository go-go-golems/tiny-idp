// Package loginui contains the tinyidp-xapp-owned interaction presentation.
// Protocol state and authorization decisions remain owned by tiny-idp.
package loginui

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

const DefaultStylesheetURL = "/static/tinyidp/login.css"

//go:embed templates/interaction.html
var interactionTemplate string

//go:embed static/login.css
var stylesheet []byte

type Options struct {
	ProductName   string
	StylesheetURL string
}

type Renderer struct {
	template       *template.Template
	productName    string
	stylesheetPath string
}

var _ idpui.InteractionRenderer = (*Renderer)(nil)

func New(opts Options) (*Renderer, error) {
	if strings.TrimSpace(opts.ProductName) == "" {
		opts.ProductName = "Tiny BBS"
	}
	if opts.StylesheetURL == "" {
		opts.StylesheetURL = DefaultStylesheetURL
	}
	stylesheetPath, err := validateStylesheetURL(opts.StylesheetURL)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("interaction").Parse(interactionTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse xapp interaction template: %w", err)
	}
	return &Renderer{
		template:       tmpl,
		productName:    strings.TrimSpace(opts.ProductName),
		stylesheetPath: stylesheetPath,
	}, nil
}

func (r *Renderer) RenderInteraction(ctx context.Context, dst io.Writer, page idpui.InteractionPage) error {
	if r == nil || r.template == nil {
		return fmt.Errorf("xapp interaction renderer is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate interaction page: %w", err)
	}
	view := templateView{
		Page:           page.Clone(),
		ProductName:    r.productName,
		StylesheetURL:  r.stylesheetPath,
		CredentialsBad: page.Error != nil && page.Error.Field == idpui.FieldCredentials,
		ConsentBad:     page.Error != nil && page.Error.Field == idpui.FieldConsent,
	}
	if err := r.template.ExecuteTemplate(dst, "interaction", view); err != nil {
		return fmt.Errorf("render xapp interaction: %w", err)
	}
	return nil
}

func (r *Renderer) AssetsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if r == nil || request.URL.Path != r.stylesheetPath || (request.Method != http.MethodGet && request.Method != http.MethodHead) {
			http.NotFound(w, request)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeContent(w, request, path.Base(r.stylesheetPath), time.Time{}, bytes.NewReader(stylesheet))
	})
}

type templateView struct {
	Page           idpui.InteractionPage
	ProductName    string
	StylesheetURL  string
	CredentialsBad bool
	ConsentBad     bool
}

func validateStylesheetURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse stylesheet URL: %w", err)
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || parsed.IsAbs() || parsed.Host != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("stylesheet URL must be a root-relative same-origin path")
	}
	if strings.Contains(raw, "\\") || path.Clean(parsed.Path) != parsed.Path || !strings.HasPrefix(parsed.Path, "/static/") || parsed.Path == "/static/" {
		return "", fmt.Errorf("stylesheet URL must be a clean path below /static/")
	}
	return parsed.Path, nil
}
