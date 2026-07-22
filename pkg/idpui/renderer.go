package idpui

import (
	"context"
	"io"
)

// InteractionRenderer converts a provider-owned page model into one complete
// HTML document. Implementations must be safe for concurrent use.
//
// The writer is not an http.ResponseWriter. Renderers cannot set headers,
// cookies, status codes, or redirects through this contract.
type InteractionRenderer interface {
	RenderInteraction(ctx context.Context, dst io.Writer, page InteractionPage) error
}

// BrowserErrorRenderer renders a terminal, provider-owned browser error page.
// The model contains no form authority, continuation, credential, or redirect
// target. Implementations must be safe for concurrent use.
type BrowserErrorRenderer interface {
	RenderBrowserError(ctx context.Context, dst io.Writer, page BrowserErrorPage) error
}

// DeviceVerificationRenderer renders the browser verification pages used by
// the device authorization grant. It receives a bounded, provider-owned
// model and cannot set headers, cookies, status codes, or redirects.
type DeviceVerificationRenderer interface {
	RenderDeviceVerification(ctx context.Context, dst io.Writer, page DeviceVerificationPage) error
}
