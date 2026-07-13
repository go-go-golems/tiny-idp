package idpui

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/interaction.html
var defaultInteractionTemplate string

// DefaultRenderer is the dependency-free built-in interaction renderer.
type DefaultRenderer struct {
	template *template.Template
}

var _ InteractionRenderer = (*DefaultRenderer)(nil)

// NewDefaultRenderer parses the embedded template. Callers should construct a
// renderer once and reuse it for every request.
func NewDefaultRenderer() (*DefaultRenderer, error) {
	tmpl, err := template.New("interaction").Parse(defaultInteractionTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse default interaction template: %w", err)
	}
	return &DefaultRenderer{template: tmpl}, nil
}

func (r *DefaultRenderer) RenderInteraction(ctx context.Context, dst io.Writer, page InteractionPage) error {
	if r == nil || r.template == nil {
		return fmt.Errorf("default renderer is not initialized")
	}
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if dst == nil {
		return fmt.Errorf("destination writer is required")
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate interaction page: %w", err)
	}
	view := defaultTemplateView{
		Page:           page.Clone(),
		CredentialsBad: page.Error != nil && page.Error.Field == FieldCredentials,
		ConsentBad:     page.Error != nil && page.Error.Field == FieldConsent,
	}
	if err := r.template.ExecuteTemplate(dst, "interaction", view); err != nil {
		return fmt.Errorf("execute default interaction template: %w", err)
	}
	return nil
}

type defaultTemplateView struct {
	Page           InteractionPage
	CredentialsBad bool
	ConsentBad     bool
}
