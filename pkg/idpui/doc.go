// Package idpui defines the presentation-only contract for tiny-idp browser
// interactions.
//
// A renderer is trusted host code, but it is deliberately given only an
// io.Writer and a typed page model. The OAuth provider retains ownership of the
// HTTP response, cookies, security headers, interaction state, authentication,
// consent policy, redirects, and token issuance.
package idpui
