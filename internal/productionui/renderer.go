package productionui

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/go-go-golems/tiny-idp/pkg/idpui"
)

//go:embed templates/interaction.html
var interactionTemplate string

//go:embed templates/workflow.html
var workflowTemplate string

type Renderer struct {
	catalog     *Catalog
	interaction *template.Template
	workflow    *template.Template
}

var _ idpui.InteractionRenderer = (*Renderer)(nil)
var _ idpui.WorkflowRenderer = (*Renderer)(nil)

func NewRenderer(catalog *Catalog) (*Renderer, error) {
	if catalog == nil {
		return nil, fmt.Errorf("theme catalog is required")
	}
	interaction, err := template.New("interaction").Parse(interactionTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse production interaction template: %w", err)
	}
	workflow, err := template.New("workflow").Parse(workflowTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse production workflow template: %w", err)
	}
	return &Renderer{catalog: catalog, interaction: interaction, workflow: workflow}, nil
}

func (r *Renderer) RenderInteraction(ctx context.Context, destination io.Writer, page idpui.InteractionPage) error {
	if r == nil || r.interaction == nil || destination == nil {
		return fmt.Errorf("production interaction renderer is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate interaction page: %w", err)
	}
	theme, err := r.catalog.Resolve(page.ClientID)
	if err != nil {
		return err
	}
	view := interactionView{
		Page:           page.Clone(),
		Theme:          theme,
		CredentialsBad: page.Error != nil && page.Error.Field == idpui.FieldCredentials,
		ConsentBad:     page.Error != nil && page.Error.Field == idpui.FieldConsent,
	}
	if err := r.interaction.ExecuteTemplate(destination, "interaction", view); err != nil {
		return fmt.Errorf("render production interaction: %w", err)
	}
	return nil
}

func (r *Renderer) RenderWorkflow(ctx context.Context, destination io.Writer, page idpui.WorkflowPage) error {
	if r == nil || r.workflow == nil || destination == nil {
		return fmt.Errorf("production workflow renderer is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return fmt.Errorf("validate workflow page: %w", err)
	}
	theme, err := r.catalog.Resolve(page.ClientID)
	if err != nil {
		return err
	}
	if err := r.workflow.ExecuteTemplate(destination, "workflow", workflowView{Page: page.Clone(), Theme: theme}); err != nil {
		return fmt.Errorf("render production workflow: %w", err)
	}
	return nil
}

type interactionView struct {
	Page           idpui.InteractionPage
	Theme          Theme
	CredentialsBad bool
	ConsentBad     bool
}

type workflowView struct {
	Page  idpui.WorkflowPage
	Theme Theme
}
