package idp

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// PresentationKind identifies a provider-owned browser page. It is not a
// route, action, or protocol state transition; native code owns all of those.
type PresentationKind string

const (
	PresentationAccountSelection PresentationKind = "account_selection"
	PresentationConsent          PresentationKind = "consent"
	PresentationDeviceVerify     PresentationKind = "device_verification"
)

func (k PresentationKind) Valid() bool {
	switch k {
	case PresentationAccountSelection, PresentationConsent, PresentationDeviceVerify:
		return true
	default:
		return false
	}
}

// PresentationPolicy decorates an already chosen native browser page. It does
// not receive a handle, CSRF token, selected account, credential, device/user
// code, Fosite request, or response writer. A successful result can only set a
// bounded document title; page controls and all protocol decisions remain
// provider-owned.
type PresentationPolicy interface {
	Present(context.Context, PresentationInput) (PresentationOutput, error)
}

type PresentationInput struct {
	Kind           PresentationKind
	ClientID       string
	RequestedScope []string
	AccountCount   int
}

func (in PresentationInput) Clone() PresentationInput {
	in.RequestedScope = append([]string(nil), in.RequestedScope...)
	return in
}

func (in PresentationInput) Validate() error {
	if !in.Kind.Valid() || strings.TrimSpace(in.ClientID) == "" || len(in.RequestedScope) > 32 || in.AccountCount < 0 || in.AccountCount > 32 {
		return errors.New("presentation input is invalid")
	}
	for _, scope := range in.RequestedScope {
		if strings.TrimSpace(scope) == "" || utf8.RuneCountInString(scope) > 128 {
			return errors.New("presentation input contains an invalid scope")
		}
	}
	return nil
}

type PresentationOutput struct {
	DocumentTitle string
}

func (out PresentationOutput) Validate() error {
	if out.DocumentTitle == "" {
		return nil
	}
	if strings.TrimSpace(out.DocumentTitle) != out.DocumentTitle || utf8.RuneCountInString(out.DocumentTitle) > 120 {
		return errors.New("presentation document title is invalid")
	}
	return nil
}
