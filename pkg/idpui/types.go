package idpui

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	InteractionFieldName = "interaction"
	CSRFFieldName        = "csrf_token"
	ActionFieldName      = "action"
	LoginFieldName       = "login"
	PasswordFieldName    = "password"
	AccountFieldName     = "account"
)

// Action is a browser-submitted interaction decision. The provider validates
// submitted actions against its server-owned interaction state; rendered
// buttons are never authorization evidence.
type Action string

const (
	ActionContinue          Action = "continue"
	ActionUseAnotherAccount Action = "use_another_account"
	ActionRemoveAccount     Action = "remove_account"
	ActionApprove           Action = "approve"
	ActionDeny              Action = "deny"
)

func (a Action) Valid() bool {
	switch a {
	case ActionContinue, ActionUseAnotherAccount, ActionRemoveAccount, ActionApprove, ActionDeny:
		return true
	default:
		return false
	}
}

// Label returns the default English label. Host renderers may choose different
// visible text but must submit the original action value.
func (a Action) Label() string {
	switch a {
	case ActionContinue:
		return "Continue"
	case ActionUseAnotherAccount:
		return "Use another account"
	case ActionRemoveAccount:
		return "Remove account"
	case ActionApprove:
		return "Approve"
	case ActionDeny:
		return "Deny"
	default:
		return ""
	}
}

// SkipsConstraintValidation reports whether an action must remain available
// when required credential fields are empty. Denial must never be blocked by
// browser-side form validation.
func (a Action) SkipsConstraintValidation() bool {
	return a == ActionDeny || a == ActionUseAnotherAccount
}

// LoginReason explains why credentials are required for this interaction.
type LoginReason string

const (
	LoginReasonSessionMissing LoginReason = "session_missing"
	LoginReasonPromptLogin    LoginReason = "prompt_login"
	LoginReasonMaxAge         LoginReason = "max_age"
	LoginReasonStepUp         LoginReason = "step_up"
)

func (r LoginReason) Valid() bool {
	switch r {
	case LoginReasonSessionMissing, LoginReasonPromptLogin, LoginReasonMaxAge, LoginReasonStepUp:
		return true
	default:
		return false
	}
}

// Explanation returns a conservative default explanation for the login state.
func (r LoginReason) Explanation() string {
	switch r {
	case LoginReasonSessionMissing:
		return "Sign in to continue."
	case LoginReasonPromptLogin:
		return "This application requested that you sign in again."
	case LoginReasonMaxAge:
		return "Your previous authentication is too old for this request. Sign in again."
	case LoginReasonStepUp:
		return "This request requires an additional authentication step."
	default:
		return "Sign in to continue."
	}
}

// ErrorCode is a stable, non-sensitive public interaction error category.
type ErrorCode string

const (
	ErrorMissingLogin       ErrorCode = "missing_login"
	ErrorInvalidCredentials ErrorCode = "invalid_credentials"
	ErrorConsentRequired    ErrorCode = "consent_required"
)

func (c ErrorCode) Valid() bool {
	switch c {
	case ErrorMissingLogin, ErrorInvalidCredentials, ErrorConsentRequired:
		return true
	default:
		return false
	}
}

// FieldName identifies the control associated with a public error.
type FieldName string

const (
	FieldCredentials FieldName = "credentials"
	FieldConsent     FieldName = "consent"
)

func (f FieldName) Valid() bool {
	switch f {
	case FieldCredentials, FieldConsent:
		return true
	default:
		return false
	}
}

// InteractionPage is the complete presentation model for one pending browser
// interaction. It contains no password, cookie, redirect URI, authorization
// code, original OAuth request, or stored interaction record.
type InteractionPage struct {
	DocumentTitle  string
	Form           InteractionForm
	Login          *LoginPrompt
	Consent        *ConsentPrompt
	AccountChooser *AccountChooserPrompt
	Error          *PublicError
}

type InteractionForm struct {
	ActionURL        string
	InteractionField string
	Interaction      string
	CSRFField        string
	CSRFToken        string
	ActionField      string
	Actions          []Action
}

type LoginPrompt struct {
	Reason        LoginReason
	LoginField    string
	PasswordField string
	LoginValue    string
	Autofocus     bool
}

type ConsentPrompt struct {
	ClientID string
	Scopes   []Scope
}

// AccountChooserPrompt is a presentation-only list of server-selected opaque
// entry selectors. An entry value is never user identity or session evidence;
// the provider validates it against the context-bound interaction on POST.
type AccountChooserPrompt struct {
	AccountField string
	Entries      []AccountChooserEntry
}

type AccountChooserEntry struct {
	Value string
	Label string
}

type Scope struct {
	Name        string
	Description string
}

type PublicError struct {
	Code    ErrorCode
	Summary string
	Field   FieldName
}

// Validate checks the presentation contract before a renderer is invoked.
func (p InteractionPage) Validate() error {
	if strings.TrimSpace(p.DocumentTitle) == "" {
		return fmt.Errorf("document title is required")
	}
	if err := p.Form.validate(); err != nil {
		return err
	}
	if p.Login != nil {
		if !p.Login.Reason.Valid() {
			return fmt.Errorf("invalid login reason %q", p.Login.Reason)
		}
		if p.Login.LoginField != LoginFieldName || p.Login.PasswordField != PasswordFieldName {
			return fmt.Errorf("login fields must use the provider contract")
		}
	}
	if p.Consent != nil && strings.TrimSpace(p.Consent.ClientID) == "" {
		return fmt.Errorf("consent client ID is required")
	}
	if p.AccountChooser != nil {
		if p.AccountChooser.AccountField != AccountFieldName {
			return fmt.Errorf("account chooser field must use the provider contract")
		}
		if len(p.AccountChooser.Entries) == 0 {
			return fmt.Errorf("account chooser requires at least one entry")
		}
		seen := make(map[string]struct{}, len(p.AccountChooser.Entries))
		for _, entry := range p.AccountChooser.Entries {
			if strings.TrimSpace(entry.Value) == "" || strings.TrimSpace(entry.Label) == "" {
				return fmt.Errorf("account chooser entries require value and label")
			}
			if _, ok := seen[entry.Value]; ok {
				return fmt.Errorf("duplicate account chooser entry")
			}
			seen[entry.Value] = struct{}{}
		}
	}
	if p.Login == nil && p.Consent == nil && p.AccountChooser == nil {
		return fmt.Errorf("at least one interaction prompt is required")
	}
	if p.Error != nil {
		if !p.Error.Code.Valid() {
			return fmt.Errorf("invalid public error code %q", p.Error.Code)
		}
		if !p.Error.Field.Valid() {
			return fmt.Errorf("invalid public error field %q", p.Error.Field)
		}
		if strings.TrimSpace(p.Error.Summary) == "" {
			return fmt.Errorf("public error summary is required")
		}
	}
	return nil
}

func (f InteractionForm) validate() error {
	actionURL, err := url.Parse(f.ActionURL)
	if err != nil || actionURL.Scheme == "" || actionURL.Host == "" || actionURL.User != nil || actionURL.Fragment != "" {
		return fmt.Errorf("action URL must be an absolute HTTP(S) URL without user info or fragment")
	}
	if actionURL.Scheme != "http" && actionURL.Scheme != "https" {
		return fmt.Errorf("action URL scheme must be HTTP or HTTPS")
	}
	if f.InteractionField != InteractionFieldName || f.CSRFField != CSRFFieldName || f.ActionField != ActionFieldName {
		return fmt.Errorf("form fields must use the provider contract")
	}
	if f.Interaction == "" || f.CSRFToken == "" {
		return fmt.Errorf("interaction and CSRF values are required")
	}
	if len(f.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}
	seen := make(map[Action]struct{}, len(f.Actions))
	for _, action := range f.Actions {
		if !action.Valid() {
			return fmt.Errorf("invalid interaction action %q", action)
		}
		if _, ok := seen[action]; ok {
			return fmt.Errorf("duplicate interaction action %q", action)
		}
		seen[action] = struct{}{}
	}
	return nil
}

// Clone makes a defensive copy suitable for crossing into host renderer code.
func (p InteractionPage) Clone() InteractionPage {
	clone := p
	clone.Form.Actions = append([]Action(nil), p.Form.Actions...)
	if p.Login != nil {
		login := *p.Login
		clone.Login = &login
	}
	if p.Consent != nil {
		consent := *p.Consent
		consent.Scopes = append([]Scope(nil), p.Consent.Scopes...)
		clone.Consent = &consent
	}
	if p.AccountChooser != nil {
		chooser := *p.AccountChooser
		chooser.Entries = append([]AccountChooserEntry(nil), p.AccountChooser.Entries...)
		clone.AccountChooser = &chooser
	}
	if p.Error != nil {
		publicError := *p.Error
		clone.Error = &publicError
	}
	return clone
}
