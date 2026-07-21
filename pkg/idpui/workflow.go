package idpui

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-go-golems/tiny-idp/pkg/idpworkflow"
)

type WorkflowRenderer interface {
	RenderWorkflow(context.Context, io.Writer, WorkflowPage) error
}

type WorkflowPage struct {
	DocumentTitle string
	// ClientID is the validated public OAuth client that owns the pending
	// workflow. Renderers may use it for presentation selection only.
	ClientID string
	Form     WorkflowForm
	Fields   []WorkflowField
	Actions  []WorkflowAction
	Errors   []WorkflowFieldError
	Error    *WorkflowGlobalError
}

type WorkflowForm struct {
	ActionURL         string
	RedirectOrigin    string
	InteractionField  string
	Interaction       string
	CSRFField         string
	CSRFToken         string
	ActionField       string
	ContinuationField string
	Continuation      string
}

type WorkflowField struct {
	Descriptor idpworkflow.FieldDescriptor
	Value      string
}

type WorkflowAction struct {
	Descriptor idpworkflow.ActionDescriptor
}

type WorkflowFieldError struct {
	Field idpworkflow.FieldID
	Code  idpworkflow.FieldErrorCode
}

// WorkflowGlobalErrorCode identifies a provider-owned error that applies to a
// live workflow but cannot truthfully be attached to a field on the current
// page. A duplicate email is discovered after this workflow has advanced to
// password selection, where no email input is rendered.
type WorkflowGlobalErrorCode string

const WorkflowErrorDuplicateIdentity WorkflowGlobalErrorCode = "duplicate_identity"

func (c WorkflowGlobalErrorCode) Valid() bool {
	return c == WorkflowErrorDuplicateIdentity
}

type WorkflowGlobalError struct {
	Code WorkflowGlobalErrorCode
}

func (e WorkflowGlobalError) Summary() string {
	if e.Code == WorkflowErrorDuplicateIdentity {
		return "An account already uses this email address. Return to the application to sign in, or restart signup with a different email address."
	}
	return "This request could not be completed."
}

func (p WorkflowPage) Validate() error {
	if strings.TrimSpace(p.DocumentTitle) == "" {
		return fmt.Errorf("workflow document title is required")
	}
	if strings.TrimSpace(p.ClientID) == "" {
		return fmt.Errorf("workflow client ID is required")
	}
	actionURL, err := url.Parse(p.Form.ActionURL)
	if err != nil || actionURL.Scheme == "" || actionURL.Host == "" || actionURL.User != nil || actionURL.Fragment != "" ||
		(actionURL.Scheme != "http" && actionURL.Scheme != "https") {
		return fmt.Errorf("workflow action URL must be an absolute HTTP(S) URL without user info or fragment")
	}
	if p.Form.RedirectOrigin != "" {
		redirectOrigin, err := url.Parse(p.Form.RedirectOrigin)
		if err != nil || redirectOrigin.Scheme == "" || redirectOrigin.Host == "" || redirectOrigin.User != nil || redirectOrigin.Path != "" || redirectOrigin.RawQuery != "" || redirectOrigin.Fragment != "" ||
			(redirectOrigin.Scheme != "http" && redirectOrigin.Scheme != "https") {
			return fmt.Errorf("workflow redirect origin must be an absolute HTTP(S) origin")
		}
	}
	if p.Form.InteractionField != InteractionFieldName || p.Form.CSRFField != CSRFFieldName || p.Form.ActionField != ActionFieldName || p.Form.ContinuationField != WorkflowContinuationFieldName {
		return fmt.Errorf("workflow form fields must use the provider contract")
	}
	if p.Form.Interaction == "" || p.Form.CSRFToken == "" || p.Form.Continuation == "" {
		return fmt.Errorf("workflow interaction, continuation, and CSRF values are required")
	}
	if len(p.Fields) == 0 || len(p.Actions) == 0 {
		return fmt.Errorf("workflow page requires fields and actions")
	}
	fieldIDs := map[idpworkflow.FieldID]bool{}
	inputNames := map[string]bool{}
	for _, field := range p.Fields {
		if err := field.Descriptor.Validate(); err != nil {
			return fmt.Errorf("invalid workflow field: %w", err)
		}
		if fieldIDs[field.Descriptor.ID] || inputNames[field.Descriptor.InputName] {
			return fmt.Errorf("duplicate workflow field")
		}
		fieldIDs[field.Descriptor.ID] = true
		inputNames[field.Descriptor.InputName] = true
		if field.Descriptor.Sensitive && field.Value != "" {
			return fmt.Errorf("secret workflow field %q may not contain a rendered value", field.Descriptor.ID)
		}
	}
	actionIDs := map[idpworkflow.ActionID]bool{}
	for _, action := range p.Actions {
		if err := action.Descriptor.Validate(); err != nil {
			return fmt.Errorf("invalid workflow action: %w", err)
		}
		if actionIDs[action.Descriptor.ID] {
			return fmt.Errorf("duplicate workflow action")
		}
		actionIDs[action.Descriptor.ID] = true
	}
	for _, fieldError := range p.Errors {
		if !fieldIDs[fieldError.Field] || !fieldError.Code.Valid() {
			return fmt.Errorf("invalid workflow field error")
		}
	}
	if p.Error != nil && !p.Error.Code.Valid() {
		return fmt.Errorf("invalid workflow global error")
	}
	return nil
}

func (p WorkflowPage) Clone() WorkflowPage {
	clone := p
	clone.Fields = append([]WorkflowField(nil), p.Fields...)
	clone.Actions = append([]WorkflowAction(nil), p.Actions...)
	clone.Errors = append([]WorkflowFieldError(nil), p.Errors...)
	if p.Error != nil {
		errorCopy := *p.Error
		clone.Error = &errorCopy
	}
	return clone
}

func (e WorkflowFieldError) Summary() string {
	switch e.Code {
	case idpworkflow.ErrorRequired:
		return "This field is required."
	case idpworkflow.ErrorMismatch:
		return "The values do not match."
	case idpworkflow.ErrorRejected:
		if e.Field == idpworkflow.FieldPassword || e.Field == idpworkflow.FieldPasswordConfirmation {
			return "Use at least 15 characters and choose a password that is difficult to guess."
		}
		return "This value could not be accepted."
	case idpworkflow.ErrorInvalid:
		return "Enter a valid value."
	default:
		return "Invalid field value."
	}
}

func (p WorkflowPage) ErrorFor(id idpworkflow.FieldID) *WorkflowFieldError {
	for index := range p.Errors {
		if p.Errors[index].Field == id {
			ret := p.Errors[index]
			return &ret
		}
	}
	return nil
}
