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

//go:embed templates/device_verification.html
var defaultDeviceVerificationTemplate string

//go:embed templates/workflow.html
var defaultWorkflowTemplate string

//go:embed templates/browser_error.html
var defaultBrowserErrorTemplate string

// DefaultRenderer is the dependency-free built-in interaction renderer.
type DefaultRenderer struct {
	template                   *template.Template
	deviceVerificationTemplate *template.Template
	workflowTemplate           *template.Template
	browserErrorTemplate       *template.Template
}

var _ InteractionRenderer = (*DefaultRenderer)(nil)
var _ DeviceVerificationRenderer = (*DefaultRenderer)(nil)
var _ WorkflowRenderer = (*DefaultRenderer)(nil)
var _ BrowserErrorRenderer = (*DefaultRenderer)(nil)

// NewDefaultRenderer parses the embedded template. Callers should construct a
// renderer once and reuse it for every request.
func NewDefaultRenderer() (*DefaultRenderer, error) {
	tmpl, err := template.New("interaction").Parse(defaultInteractionTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse default interaction template: %w", err)
	}
	deviceTemplate, err := template.New("device-verification").Parse(defaultDeviceVerificationTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse default device verification template: %w", err)
	}
	workflowTemplate, err := template.New("workflow").Parse(defaultWorkflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse default workflow template: %w", err)
	}
	browserErrorTemplate, err := template.New("browser-error").Parse(defaultBrowserErrorTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse default browser error template: %w", err)
	}
	return &DefaultRenderer{template: tmpl, deviceVerificationTemplate: deviceTemplate, workflowTemplate: workflowTemplate, browserErrorTemplate: browserErrorTemplate}, nil
}

func (r *DefaultRenderer) RenderBrowserError(ctx context.Context, dst io.Writer, page BrowserErrorPage) error {
	if r == nil || r.browserErrorTemplate == nil || dst == nil {
		return fmt.Errorf("default browser error renderer is not initialized")
	}
	if ctx == nil {
		return fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate browser error page: %w", err)
	}
	if err := r.browserErrorTemplate.ExecuteTemplate(dst, "browser-error", page); err != nil {
		return fmt.Errorf("execute default browser error template: %w", err)
	}
	return nil
}

func (r *DefaultRenderer) RenderWorkflow(ctx context.Context, dst io.Writer, page WorkflowPage) error {
	if r == nil || r.workflowTemplate == nil {
		return fmt.Errorf("default workflow renderer is not initialized")
	}
	if ctx == nil || dst == nil {
		return fmt.Errorf("context and destination writer are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate workflow page: %w", err)
	}
	if err := r.workflowTemplate.ExecuteTemplate(dst, "workflow", page.Clone()); err != nil {
		return fmt.Errorf("execute default workflow template: %w", err)
	}
	return nil
}

func (r *DefaultRenderer) RenderDeviceVerification(ctx context.Context, dst io.Writer, page DeviceVerificationPage) error {
	if r == nil || r.deviceVerificationTemplate == nil {
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
		return fmt.Errorf("validate device verification page: %w", err)
	}
	if err := r.deviceVerificationTemplate.ExecuteTemplate(dst, "device-verification", page.Clone()); err != nil {
		return fmt.Errorf("execute default device verification template: %w", err)
	}
	return nil
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
