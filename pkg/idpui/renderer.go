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
