---
Title: Secure Interaction Rendering Analysis Design and Implementation Guide
Ticket: TINYIDP-UI-001
Status: active
Topics:
    - oidc
    - identity
    - security
    - go
    - architecture
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Outer host composition demonstrating separate idp and static asset routing
    - Path: repo://internal/fositeadapter/csrf.go
      Note: Defines interaction-bound CSRF generation cookie binding and validation
    - Path: repo://internal/fositeadapter/interaction.go
      Note: Defines opaque server-owned continuation required actions bindings and reconstruction
    - Path: repo://internal/fositeadapter/provider.go
      Note: Owns strict authorize flow security headers and current hard-coded interaction rendering
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public configuration boundary where UIConfig is proposed
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Construction boundary that forwards public options into the strict adapter
    - Path: repo://pkg/idpstore/types.go
      Note: Defines clients sessions consent required actions and authorization interaction records
ExternalSources:
    - https://pkg.go.dev/html/template
    - https://www.w3.org/TR/CSP/
    - https://openid.net/specs/openid-connect-core-1_0-18.html
    - https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
    - https://www.rfc-editor.org/rfc/rfc9700.html
    - https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html
    - https://www.w3.org/WAI/WCAG22/Understanding/accessible-authentication-minimum.html
Summary: Evidence-backed design for a host-supplied HTML interaction renderer that permits branded login and consent pages while keeping OAuth state, CSRF, actions, headers, and terminal authorization decisions under tiny-idp control.
LastUpdated: 2026-07-13T17:38:38.914499982-04:00
WhatFor: Defines the public rendering API, trust boundary, CSP and asset policy, migration plan, implementation phases, and verification gates for customizable tiny-idp browser interactions.
WhenToUse: Read before changing login or consent HTML, exposing UI configuration through embeddedidp.Options, mounting theme assets, modifying interaction errors, or reviewing a custom renderer for production use.
---


# Secure Interaction Rendering Analysis Design and Implementation Guide

## Executive summary

tiny-idp's strict provider currently renders its login and consent interaction as
one HTML string inside `internal/fositeadapter.Provider`. The page is functional,
but the embedding API offers no way for an application host to replace the
markup or provide a stylesheet. The existing embedded `static/login.html` file
belongs to the synthetic test server and does not affect the production-oriented
Fosite provider used by `tinyidp-xapp`.

This document proposes a public, host-supplied interaction renderer. The renderer
may choose HTML structure, text, CSS classes, and same-origin static assets. It
does not receive the HTTP response writer, the original OAuth request, redirect
URI, signing material, browser cookies, stored interaction record, or any ability
to issue an authorization response. tiny-idp remains responsible for headers,
cookies, status codes, interaction creation, CSRF validation, authentication,
consent policy, replay prevention, redirect validation, and authorization-code
issuance.

The core interface is deliberately small:

```go
type InteractionRenderer interface {
    RenderInteraction(
        ctx context.Context,
        dst io.Writer,
        page InteractionPage,
    ) error
}
```

The interface belongs in a new `pkg/idpui` package. Placing it directly in
`pkg/embeddedidp` would create an import cycle because `pkg/embeddedidp` already
constructs `internal/fositeadapter`, and the adapter must invoke the renderer.
Both packages can safely depend on `pkg/idpui`.

The renderer writes into a bounded buffer. Only after rendering succeeds does
the provider commit an HTTP 200 response. The provider sets the content type,
cache policy, CSP, anti-framing headers, and all cookies. This means a rendering
failure cannot leave a partially committed authorization page, and a custom
renderer cannot weaken response headers through the supported API.

Styles are delivered as immutable, same-origin files under `/static/`, mounted
by the application host. Interaction pages use a fixed CSP that adds
`style-src 'self'` while retaining `default-src 'none'`, `frame-ancestors
'none'`, and `base-uri 'none'`. Scripts remain disabled. The first version does
not add inline-style, inline-script, remote-font, remote-image, or arbitrary CSP
configuration escape hatches.

The implementation should proceed in six substantive phases after this design
phase:

1. Introduce the public view model, renderer contract, and default
   `html/template` renderer.
2. Integrate the renderer into the strict provider and embedded API without
   exposing `http.ResponseWriter`.
3. Make recoverable interaction failures renderable and test the complete
   login/consent state matrix.
4. Add a concrete xapp theme and mount its embedded CSS under `/static/`.
5. Build renderer conformance, fuzz, static-analysis, browser, CSP, and
   accessibility gates.
6. Complete operational documentation, review, and staged release evidence.

The design is proposed, not yet implemented. This ticket establishes the
contract and the work plan; it does not modify runtime code.

## 1. How to use this document

This document is written for an engineer who has not worked on tiny-idp before.
Read Sections 2 through 6 before editing code. Those sections explain the OAuth
interaction boundary and why rendering is security-sensitive. Sections 7
through 13 define the proposed API and runtime behavior. Sections 14 through 17
turn the design into implementation and review work.

The terms **host**, **provider**, **renderer**, **interaction**, and **relying
party** have precise meanings here:

- The **host** is the Go application embedding tiny-idp, such as
  `cmd/tinyidp-xapp`.
- The **provider** is tiny-idp's OAuth 2.0 and OpenID Connect authorization
  server implementation.
- The **renderer** is trusted host code that converts a provider-owned view
  model into HTML.
- An **interaction** is a short-lived, server-owned authorization continuation
  that records required user actions and validated protocol state.
- A **relying party**, also called an OAuth client, initiates authorization and
  later redeems the returned code.

The word **trusted** does not mean “incapable of mistakes.” A host-supplied
template is trusted application code because it runs in the authorization
server's origin and presents credential and consent controls. The API still
removes unnecessary authority so ordinary template mistakes fail closed.

## 2. Problem statement

An embedded identity provider must allow its host application to present a
coherent product interface. Login pages need product typography, spacing,
accessible labels, error presentation, responsive layout, and sometimes
client-specific explanation. Hard-coding markup inside the protocol adapter
prevents that work and forces downstream users to fork internal code.

The direct solution—accept an arbitrary handler or response callback—is too
broad. A callback with `http.ResponseWriter` and `*http.Request` can modify
cookies, suppress security headers, redirect to attacker-selected locations,
reflect OAuth parameters, or commit a partial response before returning an
error. Those capabilities are not required to choose HTML and CSS.

The design problem is therefore:

> How can a host control the presentation of login and consent interactions
> while tiny-idp remains the sole owner of protocol continuation, authorization
> decisions, browser binding, CSRF, headers, and redirects?

### 2.1 In scope

- A public Go interface for rendering interactive authorization pages.
- A typed, immutable-by-convention page model with explicit login, consent,
  action, and public-error data.
- A secure default renderer based on `html/template`.
- Host-provided templates and same-origin external stylesheets.
- Integration through `pkg/embeddedidp.Options`.
- Recoverable login and consent error rendering.
- CSP and asset-delivery rules.
- Conformance tests, fuzzing, static analysis, accessibility checks, and browser
  scenarios.
- A concrete integration path for `tinyidp-xapp`.

### 2.2 Out of scope for the first implementation

- Executing untrusted templates uploaded by users or OAuth clients.
- Loading renderer code dynamically from Goja or another scripting runtime.
- Remote stylesheets, web fonts, analytics, tag managers, or third-party images.
- Client-supplied HTML, Markdown, CSS, logos, or arbitrary URLs.
- A general CMS for identity pages.
- Changing token, session, password, or consent policy semantics.
- Account chooser and password-reset implementation. The view model should not
  prevent those future pages, but this ticket does not build them.
- Maintaining the exact byte-for-byte legacy HTML output.

No backwards-compatibility adapter is proposed. The default renderer replaces
the hard-coded function while preserving the supported authorization behavior.

## 3. Current system architecture

### 3.1 The strict provider is the relevant implementation

The production-oriented browser interaction lives in
`internal/fositeadapter/provider.go`. `Provider.Handler` creates a `net/http`
multiplexer, registers the issuer endpoints, and wraps the result with security
headers. The registered paths are discovery, JWKS, authorize, token, UserInfo,
end-session, health, and readiness.

At lines 407–415, the authorize handler dispatches GET requests to
`beginAuthorize` and POST requests to `resumeAuthorize`. This is the primary
state boundary:

```text
GET  /authorize  -> validate protocol request -> decide required actions
POST /authorize  -> validate interaction       -> satisfy required actions
```

The synthetic server in `internal/server` is a different product surface. Its
`static/login.html` is parsed with `html/template`, includes development scenario
pickers, and reconstructs the request using hidden fields. That page is useful
for mock-server testing, but it is not used by `pkg/embeddedidp` or
`tinyidp-xapp`. A customization feature must not accidentally modify only this
mock path.

### 3.2 The embedding path

`pkg/embeddedidp.New` validates the public options and constructs the strict
adapter:

```text
cmd/tinyidp-xapp
    |
    | embeddedidp.New(options)
    v
pkg/embeddedidp
    |
    | fositeadapter.NewProvider(options)
    v
internal/fositeadapter
    |
    | Handler()
    v
net/http
```

`pkg/embeddedidp/options.go:45-59` currently exposes issuer, store, cookie,
token, audit, consent, rate limiting, client-address resolution, authentication,
password policy, password-work, and maintenance configuration. There is no UI
or renderer field.

`pkg/embeddedidp/provider.go:60` manually maps those options into
`fositeadapter.Options`. Any public renderer configuration must be forwarded at
this construction boundary.

Both development and production xapp constructors call `embeddedidp.New`.
`cmd/tinyidp-xapp/development_app.go:235-240` mounts the IdP at `/idp/`, native
application-auth handlers at their exact methods and paths, and the Goja HTTP
host as the fallback. Frontend assets already use `/static/assets/...`, which
demonstrates the correct outer-host ownership model for a login stylesheet.

### 3.3 Authorization interaction creation

`beginAuthorize` performs these operations in order:

1. It rejects unsupported request objects.
2. Fosite validates the OAuth/OIDC authorization request.
3. It parses `max_age` and reads the browser session.
4. It determines whether login, fresh login, or consent is required.
5. It handles `prompt=none` without displaying UI.
6. It immediately finishes authorization when no interaction is required.
7. Otherwise, it creates a server-owned interaction and renders the page.

The required actions are stored as a bit set in
`pkg/idpstore.InteractionRequiredAction`:

```go
const (
    InteractionRequireLogin InteractionRequiredAction = 1 << iota
    InteractionRequireFreshLogin
    InteractionRequireConsent
    InteractionRequireStepUp
)
```

This representation matters to UI design. A boolean named `NeedLogin` loses the
difference between an ordinary login and forced reauthentication caused by
`prompt=login` or `max_age`. The page model should retain that distinction so the
renderer can accurately tell the user why credentials are required.

### 3.4 Server-owned continuation

`internal/fositeadapter/interaction.go` converts the validated request into an
`idpstore.InteractionRecord`. The browser receives only two continuation values:

- an opaque `interaction` handle; and
- a `csrf_token` bound to the browser's CSRF cookie and that handle.

The server stores:

- the canonical validated request;
- a digest of that request;
- client identifier and exact redirect URI;
- required actions;
- browser-binding and optional session-binding hashes;
- a client-generation hash;
- creation and expiration times; and
- terminal consumption state.

The current hardening test
`TestInteractionFormContainsNoProtocolContinuation` asserts that the HTML does
not contain browser-editable `client_id`, `redirect_uri`, `state`, `scope`, or
`code_challenge` inputs. A renderer API must preserve this invariant. It may
display a client ID and scopes as escaped text, but it must not convert them into
protocol continuation fields.

```text
OAuth request parameters
        |
        | validate and canonicalize
        v
server-side InteractionRecord
        |
        +-- hashed browser/session bindings
        +-- client policy snapshot hash
        +-- required actions
        +-- expiration and terminal state
        |
        | expose only opaque handle + CSRF MAC
        v
browser form
```

### 3.5 Interaction resumption

`resumeAuthorize` does not trust the submitted form to describe the OAuth
request. It uses the opaque handle to load the record and then validates:

- the CSRF token;
- record existence, expiry, and unconsumed state;
- browser binding;
- session binding when one was recorded;
- reconstructed request digest;
- current client enabled state;
- current redirect and scope policy;
- current signing-key availability;
- forced-login requirements;
- password authentication and rate limits;
- current user enabled state;
- consent policy and explicit approval; and
- atomic terminal consumption.

Only after those checks does `finishAuthorize` create the Fosite authorization
response. This is the most important existing property for the renderer design:
HTML does not authorize. Server state and server-side policy authorize.

### 3.6 Current rendering implementation

`internal/fositeadapter/provider.go:941-959` builds HTML through string
concatenation and `fmt.Fprintf`. It conditionally adds login inputs, always
includes the requested-access disclosure for the call site at line 482, creates
approve/deny buttons, and emits the opaque interaction and CSRF hidden fields.

The current function has several limitations:

- There is no document title, language attribute, viewport metadata, visible
  labels, focus treatment, error region, or coherent semantic hierarchy.
- Layout and visual styling cannot be supplied.
- Rendering errors are discarded because the write result is ignored.
- Manual `htmlEscape` is narrower and easier to misuse than contextual
  autoescaping.
- The function accepts several loosely related scalar parameters rather than a
  typed page contract.
- Recoverable authentication errors use `http.Error`, so the response no longer
  contains a usable form.
- The reason for a login requirement is not available to the renderer.
- The page contract is implicit in string literals and regular-expression test
  helpers.

### 3.7 Current response policy

`Provider.securityHeaders` currently emits:

```text
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Content-Security-Policy: default-src 'none'; frame-ancestors 'none';
  form-action 'self' https:; base-uri 'none'
```

`renderInteraction` additionally sets `Content-Type: text/html; charset=utf-8`
and `Cache-Control: no-store`.

Because `default-src 'none'` is the fallback for style requests, both inline and
external CSS are currently blocked. Adding a `<style>` or `<link>` element
without changing the CSP would create markup that browsers refuse to apply.

## 4. Standards and research basis

The complete source captures are stored under this ticket's `sources/`
directory. This section states how each source affects the implementation.

### 4.1 Go `html/template`

The Go package documentation says that template authors are trusted while data
passed to `Execute` is untrusted. It applies contextual escaping for HTML, CSS,
JavaScript, URL, and attribute contexts. This precisely matches the proposed
boundary: the host owns the template, but client IDs, scopes, login values, and
public errors are data.

The implementation must not expose `template.HTML`, `template.CSS`,
`template.JS`, or similar trusted-content types in `InteractionPage`. Those types
bypass escaping and would turn data back into markup. The proposed static
analysis phase adds a repository analyzer to prevent those types in renderer
view models and default templates.

### 4.2 Content Security Policy Level 3

CSP Level 3 defines `style-src`, nonce and hash sources, inline checks, and
`default-src` fallback behavior. The current policy is an effective denial
baseline because unspecified resource types inherit `'none'`.

For v1, external same-origin CSS is preferred over inline CSS. The provider adds
`style-src 'self'`; the host mounts versioned CSS under `/static/`; scripts remain
disabled by `default-src 'none'`. This is easier to audit than per-response
nonces and does not require passing a nonce into host templates.

### 4.3 CSRF guidance

The OWASP CSRF guidance describes hidden form tokens and session-bound signed
double-submit patterns. tiny-idp already creates an HMAC token over the browser
CSRF cookie and opaque interaction handle. The renderer must emit the provided
token as a POST form field and must never put it in a URL, log, stylesheet URL,
or client-side script.

The renderer does not generate or validate CSRF tokens. It only places a
provider-generated value in the form. This avoids two independent
implementations of the same security mechanism.

### 4.4 OpenID Connect Core

OIDC Core requires support for `prompt`, including `none` and `login`, and ties
`max_age` to fresh authentication and `auth_time`. A branded page must not turn
forced reauthentication into ordinary session reuse. The page model therefore
contains a login reason, and tests exercise blank and crafted submissions for
both `prompt=login` and expired `max_age` interactions.

### 4.5 OAuth 2.0 Security Best Current Practice

RFC 9700 treats authorization-server UI as part of the security boundary. It
discusses clickjacking, consent presentation, redirect behavior after credential
POSTs, and phishing risks around registered redirect URIs. The existing
`frame-ancestors 'none'` and `X-Frame-Options: DENY` must remain invariant across
all renderers.

The RFC also warns against redirect status 307 after credential submission
because a user agent can preserve the POST body and send credentials to the
client. Renderer work must not introduce redirects; provider redirect behavior
needs a separate check that successful credential POSTs use an appropriate
status, preferably 303. Existing tests currently accept both 302 and 303, so the
release phase should resolve that ambiguity explicitly.

### 4.6 WCAG 2.2 guidance

WCAG guidance requires text identification of input errors and accessible
authentication paths. The default renderer should use explicit `<label>`
elements, permit password-manager and paste behavior, use the standard
autocomplete tokens, keep a logical focus order, and associate errors with the
affected input through `aria-describedby` and `aria-invalid`.

The design avoids JavaScript-only state changes. Each submission returns a new
document, so a visible error summary and correctly associated field errors are
more important than live-region behavior. If a future renderer performs dynamic
updates, status messages must be programmatically determinable.

## 5. Requirements

### 5.1 Functional requirements

The feature is complete only when:

- An embedding host can supply a custom renderer through
  `embeddedidp.Options`.
- The renderer can produce distinct login, forced-login, combined
  login/consent, and consent-only presentations.
- The renderer can display client and requested-scope disclosure.
- Recoverable login failures can return a usable, branded form without
  retaining the password.
- A nil renderer selects a secure built-in default.
- The xapp can embed its HTML template and stylesheet into the Go binary.
- The xapp mounts CSS under `/static/` and the login page can load it under the
  enforced CSP.
- Existing authorization behavior remains covered by integration tests.

### 5.2 Security requirements

- The renderer cannot set headers, cookies, or status codes through the public
  interface.
- The renderer cannot access the original request or stored continuation.
- The browser never receives protocol continuation fields beyond the opaque
  interaction handle and CSRF token.
- All dynamic strings are treated as untrusted plain text.
- Scripts and event-handler attributes are absent from the default and xapp
  renderers.
- External resources are same-origin and explicitly allowed by CSP.
- Login pages cannot be framed.
- Interaction documents are not cached.
- Renderer failure produces a generic fail-closed response and an audit event.
- Passwords never appear in a page model, render error, log, trace, or retry
  response.
- Forced authentication, consent, replay, expiry, browser binding, and client
  revalidation remain server decisions.

### 5.3 Accessibility requirements

- Every input has a visible programmatically associated label.
- Username and password inputs use `autocomplete="username"` and
  `autocomplete="current-password"`.
- Password managers and paste are not blocked.
- Keyboard order follows document order.
- Focus indication meets contrast requirements.
- Errors are visible text and associated with the corresponding input.
- Approve and deny actions have distinct accessible names and are not
  distinguishable by color alone.
- Responsive layout works at narrow widths and high zoom.
- The document declares a language and has a meaningful title and heading.

### 5.4 Operability requirements

- Theme assets are compiled into the host binary.
- A missing stylesheet does not make the form unusable.
- Static assets may be cached with content hashes; HTML remains `no-store`.
- Rendering time and failures are observable without logging page data.
- The default renderer provides a recovery path when a custom renderer cannot
  be constructed at process startup; production should fail startup rather than
  silently selecting a different theme after explicit configuration.

## 6. Threat model and authority map

### 6.1 Actors and inputs

The relevant actors are:

- The embedding host author, who is trusted to provide application code and
  templates.
- The OAuth client administrator, who configures client identifiers, redirects,
  and scopes.
- The browser user, who supplies credentials and decisions.
- A malicious website attempting CSRF, framing, login CSRF, or phishing.
- A malicious or compromised OAuth client attempting redirect, scope, or
  continuation manipulation.
- Untrusted strings stored in client or scope configuration.

### 6.2 Authority matrix

| Capability | Provider | Renderer | Browser |
|---|---:|---:|---:|
| Validate OAuth request | yes | no | no |
| Store canonical continuation | yes | no | no |
| Generate interaction handle | yes | no | no |
| Generate and validate CSRF | yes | no | returns token |
| Authenticate password | yes | no | supplies credential |
| Decide required login freshness | yes | displays reason | no |
| Decide consent requirement | yes | displays disclosure | submits choice |
| Set response headers and cookies | yes | no | no |
| Choose HTML structure and classes | no | yes | no |
| Choose same-origin stylesheet | host | yes | requests asset |
| Issue code or redirect | yes | no | follows redirect |

### 6.3 Failure principles

The implementation follows four failure principles:

1. A missing or malformed UI field must not create an authorization success.
2. A renderer error must happen before HTTP headers and body are committed.
3. A custom template may reduce usability, but it must not gain new protocol
   authority through the supported interface.
4. A security-header change requires provider code and tests; it cannot be a
   runtime template option.

## 7. Proposed package architecture

### 7.1 Package layout

```text
pkg/idpui/
    renderer.go             public interface and view model
    actions.go              closed action and reason constants
    default_renderer.go     html/template implementation
    templates/login.html    embedded default document
    renderer_test.go        escaping and page-shape tests

pkg/embeddedidp/
    options.go              UIConfig added to public Options
    provider.go             forwards renderer into adapter

internal/fositeadapter/
    provider.go             builds pages and invokes renderer
    rendering.go            bounded rendering and response policy
    rendering_test.go       provider integration and headers

cmd/tinyidp-xapp/internal/loginui/
    renderer.go             xapp-owned template renderer
    templates/login.html    branded markup
    static/login.css        branded stylesheet source
    renderer_test.go        host theme contract tests
```

The host's build pipeline may copy or embed the stylesheet into its existing
static asset bundle. It must remain available at a `/static/...` URL.

### 7.2 Dependency direction

```text
pkg/idpui                 no dependency on provider internals
   ^   ^
   |   |
   |   +---------------- internal/fositeadapter
   |
   +-------------------- pkg/embeddedidp
                              ^
                              |
                       cmd/tinyidp-xapp
```

`pkg/idpui` may depend only on small standard-library packages such as
`context`, `io`, and possibly `time`. It must not import Fosite, `idpstore`,
`net/http`, or xapp packages.

## 8. Public API design

### 8.1 Renderer interface

```go
package idpui

type InteractionRenderer interface {
    RenderInteraction(
        ctx context.Context,
        dst io.Writer,
        page InteractionPage,
    ) error
}
```

The interface receives `context.Context` so startup cancellation, request
cancellation, tracing, and bounded work can propagate. It receives `io.Writer`
so it can emit a document without controlling HTTP behavior. It receives the
page by value so the contract is explicit at the call site.

The provider should make defensive copies of slices before calling an external
renderer. Interface values can still contain mutable slices, so copying avoids a
renderer accidentally changing provider-owned data that remains in use.

### 8.2 Page model

```go
type InteractionPage struct {
    DocumentTitle string
    Form          InteractionForm
    Login         *LoginPrompt
    Consent       *ConsentPrompt
    Error         *PublicError
}

type InteractionForm struct {
    ActionURL         string
    InteractionField string
    Interaction      string
    CSRFField         string
    CSRFToken         string
    Actions           []Action
}

type LoginPrompt struct {
    Reason     LoginReason
    LoginValue string
    Autofocus  bool
}

type LoginReason string

const (
    LoginReasonSessionMissing LoginReason = "session_missing"
    LoginReasonPromptLogin    LoginReason = "prompt_login"
    LoginReasonMaxAge         LoginReason = "max_age"
    LoginReasonStepUp         LoginReason = "step_up"
)

type ConsentPrompt struct {
    ClientID string
    Scopes   []Scope
}

type Scope struct {
    Name        string
    Description string
}

type Action string

const (
    ActionContinue Action = "continue"
    ActionApprove  Action = "approve"
    ActionDeny     Action = "deny"
)

type PublicError struct {
    Code    ErrorCode
    Summary string
    Field   FieldName
}
```

The model contains values needed to construct the form, but none of them are
authorization evidence by themselves. `ActionURL` is provider-generated from
the validated issuer. Field names are constants passed explicitly so custom
templates do not duplicate private string literals.

`PublicError` contains a stable code and safe generic text. It never wraps or
exposes an internal error. Invalid username and invalid password use the same
public code and text to avoid account enumeration.

### 8.3 Configuration through `embeddedidp.Options`

```go
type UIConfig struct {
    Renderer idpui.InteractionRenderer
}

type Options struct {
    // Existing fields...
    UI UIConfig
}
```

`nil` selects `idpui.NewDefaultRenderer()`. The nested configuration leaves a
coherent location for future UI policy without adding unrelated fields to the
top-level options. It must not grow a raw CSP string or arbitrary security-header
map.

`fositeadapter.Options` receives the same interface internally:

```go
type Options struct {
    // Existing fields...
    InteractionRenderer idpui.InteractionRenderer
}
```

The embedded provider forwards `opts.UI.Renderer` directly. This is not a
compatibility adapter; it is dependency injection across the public/internal
construction boundary.

### 8.4 Why the renderer does not receive `http.ResponseWriter`

With an HTTP response writer, a renderer could:

- overwrite CSP or cache headers;
- set or clear authentication cookies;
- emit a redirect;
- send a 200 before discovering a template error;
- call `WriteHeader` with a misleading status;
- stream an unbounded response; or
- use HTTP hijacking or flushing interfaces.

None of those capabilities are required for HTML customization. `io.Writer`
supports every normal template engine while preserving provider ownership of
the response.

### 8.5 Why the renderer does not receive `*http.Request`

The request contains cookies, remote-address data, headers, and unvalidated
browser inputs. A template that reads query parameters directly can accidentally
reintroduce protocol fields or reflected injection. The provider should derive a
minimal view model after request validation and pass only that model.

If localization is added later, the provider may pass a normalized locale value.
It should not expose the entire request merely so a renderer can inspect
`Accept-Language`.

## 9. Rendering pipeline

### 9.1 Page construction

The adapter converts provider state into `InteractionPage` in one function. This
keeps the mapping reviewable and testable:

```text
buildInteractionPage(record, client, scopes, submittedForm, publicError):
    page.Form.ActionURL = issuer /authorize
    page.Form.Interaction = raw handle
    page.Form.CSRFToken = provider-issued token

    if record requires login or fresh login:
        page.Login = login prompt with exact reason

    if access disclosure is applicable:
        page.Consent = client ID plus copied scope list

    page.Form.Actions = actions permitted for this presentation
    page.Error = already-sanitized public error
    return page
```

The raw interaction handle and CSRF token exist only at the request/rendering
edge. They are not written to logs or audit fields.

### 9.2 Buffered output

```text
renderInteraction(page):
    buffer = bounded buffer with maximum document size
    err = renderer.RenderInteraction(ctx, buffer, page)
    if err != nil:
        audit interaction.render_failed without page data
        return generic 500
    set provider-owned headers
    write 200 and buffered document
```

A 256 KiB default maximum is sufficient for a login document and protects
availability if a custom renderer loops or emits unexpected data. The exact
limit should be a package constant in v1, not a runtime option. A future need for
larger documents should be justified with measurements.

### 9.3 Renderer errors

Renderer errors are internal. The browser receives a generic response such as
“authentication page unavailable.” Audit and metrics record:

- renderer implementation name if supplied as a static startup label;
- interaction stage, not the handle;
- client ID only if existing audit policy permits it;
- duration bucket; and
- failure category such as execute, size limit, or response write.

The error text itself must be cleaned before audit because template engines can
include values or template source locations.

### 9.4 Recoverable form errors

The current provider sends plain `http.Error` responses for a missing login,
invalid credentials, and missing explicit consent. The new pipeline should
rerender the still-pending interaction when retry is safe.

```text
POST interaction
    |
    +-- invalid CSRF/binding/expiry/replay -> terminal generic error, no form
    |
    +-- rate limited/unavailable ---------> bounded error response, Retry-After
    |
    +-- missing login/invalid password ---> same pending interaction form
    |                                       password field always empty
    |
    +-- valid login + consent decision ---> continue server state machine
```

Rerendering may reuse the submitted CSRF token only after the provider has
validated it and confirmed the interaction is pending and browser-bound. A
cleaner implementation can rotate the CSRF value, but that is not necessary for
v1 and must be assessed against browser Back-button behavior.

The renderer may receive the submitted normalized login for usability. It never
receives the password. Invalid user and invalid password remain one public
message.

## 10. HTML contract

### 10.1 Required structure

A conforming renderer produces a complete HTML document containing exactly one
authorization form:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{.DocumentTitle}}</title>
    <link rel="stylesheet" href="/static/tinyidp/login.css">
  </head>
  <body>
    <main>
      <h1>Sign in</h1>
      <form method="post" action="{{.Form.ActionURL}}">
        <!-- provider fields, conditional prompts, and allowed actions -->
      </form>
    </main>
  </body>
</html>
```

Required invariants include:

- The form method is POST.
- The action is exactly the provider-supplied same-origin authorization URL.
- Exactly one interaction field and one CSRF field are present.
- Login and password inputs are present exactly when `Login != nil`.
- The password input is never populated.
- Submitted action values are drawn from `Form.Actions`.
- OAuth protocol parameters are not hidden inputs.
- Dynamic values occur only through ordinary `html/template` actions.

### 10.2 Action semantics

The page model presents a closed action set. The renderer determines labels and
layout, but it must submit the exact values:

| Action | Meaning |
|---|---|
| `continue` | Submit credentials or continue a non-consent interaction. |
| `approve` | Explicitly approve the displayed client and scope request. |
| `deny` | Explicitly deny the authorization request. |

The server still validates whether an action is acceptable for the stored
required actions. A malicious browser can submit any string regardless of what
the HTML shows, so authorization correctness cannot rely on rendered buttons.

### 10.3 Consent disclosure

The renderer displays the bound client ID and every requested scope as text. A
scope description may be derived only from a trusted server-side registry. An
OAuth request cannot supply its own description.

`idpstore.Client` currently has no display-name or logo field. V1 therefore uses
the client ID as the canonical disclosure. A host renderer may map known IDs to
static labels, but unknown clients must fall back visibly to the ID. Adding
persisted display metadata requires a separate schema and administration design;
it should not be smuggled into this rendering change.

### 10.4 Login reason

Forced reauthentication should be visible. Suggested default text is:

- Session missing: “Sign in to continue.”
- `prompt=login`: “This application requested that you sign in again.”
- `max_age`: “Your previous authentication is too old for this request. Sign in
  again.”
- Step-up: “This request requires an additional authentication step.”

These messages explain provider state without disclosing sensitive policy
details.

## 11. CSS and asset delivery

### 11.1 Host-owned assets

The renderer belongs to the host, so its stylesheet should also belong to the
host. For xapp:

```go
//go:embed static/*
var staticFS embed.FS

mux.Handle(
    "GET /static/tinyidp/",
    http.StripPrefix(
        "/static/tinyidp/",
        http.FileServer(http.FS(themeAssets)),
    ),
)
mux.Handle("/idp/", idpProvider.Handler())
```

The actual project should reuse the existing asset host if it already provides
content hashing and correct media types. It should not create a second server or
filesystem root merely for the login page.

### 11.2 Cache policy

- HTML interaction responses use `Cache-Control: no-store`.
- Content-hashed CSS may use `Cache-Control: public, max-age=31536000,
  immutable`.
- Non-hashed theme paths should use revalidation or a short cache lifetime.
- Theme assets must not vary on cookies or contain user-specific data.

### 11.3 CSP

The proposed interaction policy is:

```text
default-src 'none';
style-src 'self';
img-src 'self' data:;
font-src 'self';
frame-ancestors 'none';
form-action 'self';
base-uri 'none'
```

The first xapp theme does not need images or custom fonts, but explicitly
limiting those types makes future behavior predictable. If the implementation
does not ship either resource type, the stricter choice is to leave them at
`'none'` until required.

There is no `script-src` directive because `default-src 'none'` blocks scripts.
There is no `'unsafe-inline'`. Templates must not use `<style>` elements,
`style` attributes, `<script>` elements, inline event handlers, or `javascript:`
URLs.

### 11.4 Why remote resources are excluded

Remote resources disclose that a user visited the authorization page, create a
new supply-chain dependency, complicate CSP, and can modify the credential UI
without a tiny-idp deployment. All production assets should be reviewed and
embedded into the host binary.

## 12. Default renderer

The built-in renderer ensures that existing embedding callers remain functional
when `Options.UI.Renderer` is nil. It is a real `html/template` implementation,
not a retained call to the string-building function.

Construction parses the template once:

```go
//go:embed templates/interaction.html
var defaultTemplate string

func NewDefaultRenderer() (*DefaultRenderer, error) {
    tmpl, err := template.New("interaction").Parse(defaultTemplate)
    if err != nil {
        return nil, errors.Wrap(err, "parse default interaction template")
    }
    return &DefaultRenderer{template: tmpl}, nil
}

var _ InteractionRenderer = (*DefaultRenderer)(nil)
```

Because an embedded constant template should always parse, package initialization
is a possible alternative. A constructor returning an error is preferred because
it keeps failure handling explicit and makes testing alternate templates easier.

The default renderer must remain intentionally plain but fully usable. Product
branding belongs in host renderers. It should contain no development accounts,
scenario pickers, or JavaScript.

## 13. xapp renderer example

The xapp implementation uses its own package and template:

```go
renderer, err := loginui.New(loginui.Options{
    ProductName:   "Local Loop",
    StylesheetURL: "/static/tinyidp/login.css",
})
if err != nil {
    return nil, errors.Wrap(err, "construct login renderer")
}

idpProvider, err := embeddedidp.New(ctx, embeddedidp.Options{
    Issuer: issuer,
    Store: store,
    UI: embeddedidp.UIConfig{
        Renderer: renderer,
    },
    // Existing security and lifecycle options...
})
```

`loginui.New` validates that `StylesheetURL` is a root-relative `/static/` URL
without a host, query, fragment, or path traversal. The template treats it as
data; the CSP supplies the second enforcement layer.

The early Macintosh visual language requested for the BBS can be reused through
shared design tokens, but the credential page should remain visually quieter
than the application. It should not simulate window chrome, menus, or desktop
controls that compete with the authentication decision.

## 14. Decision records

### Decision: use a narrow writer-based renderer

- **Context:** The host needs HTML control, while HTTP and authorization state
  must remain provider-owned.
- **Options considered:** Hard-coded templates, template path configuration,
  `http.Handler`, callback with `http.ResponseWriter`, and writer-based renderer.
- **Decision:** Use `RenderInteraction(context.Context, io.Writer,
  InteractionPage) error`.
- **Rationale:** It supplies enough authority to render HTML and no authority to
  alter HTTP state, cookies, redirects, or protocol continuation.
- **Consequences:** Rendering must be buffered and the page model must be
  complete. Streaming is intentionally unavailable.
- **Status:** proposed.

### Decision: place contracts in `pkg/idpui`

- **Context:** `pkg/embeddedidp` imports the internal adapter, which must also
  invoke the renderer.
- **Options considered:** Define the interface in `pkg/embeddedidp`, define it in
  the internal adapter, add it to the broad `pkg/idp`, or create `pkg/idpui`.
- **Decision:** Create dependency-light `pkg/idpui`.
- **Rationale:** Both construction layers can import it without a cycle, and the
  UI contract remains distinct from audit, authentication, and policy APIs.
- **Consequences:** A new public package must maintain semantic-versioning
  discipline.
- **Status:** proposed.

### Decision: use trusted Go templates, not runtime user templates

- **Context:** Identity pages handle credentials and authorization decisions.
- **Options considered:** Host-compiled Go renderers, filesystem templates,
  database templates, Goja templates, and OAuth-client-provided HTML.
- **Decision:** V1 accepts only host-constructed Go renderers compiled into the
  application.
- **Rationale:** Deployment review and Go's type system remain the trust and
  change-control boundaries. Runtime template upload adds an unnecessary code
  execution and content-governance problem.
- **Consequences:** Theme changes require a host build and deployment.
- **Status:** proposed.

### Decision: keep provider ownership of headers and CSP

- **Context:** A theme needs CSS but must not weaken anti-framing, caching, or
  content restrictions.
- **Options considered:** Renderer-supplied headers, configurable raw CSP,
  middleware outside the provider, and provider-fixed headers.
- **Decision:** The provider emits a fixed interaction CSP and all security
  headers.
- **Rationale:** Security invariants remain testable in one package and cannot
  drift between themes.
- **Consequences:** New resource types require an explicit provider change and
  security review.
- **Status:** proposed.

### Decision: use same-origin external CSS

- **Context:** Current CSP blocks all styles; themes require CSS.
- **Options considered:** `'unsafe-inline'`, CSP nonces, CSP hashes, data URLs,
  remote stylesheets, and same-origin external CSS.
- **Decision:** Permit `style-src 'self'` and serve embedded CSS under
  `/static/`.
- **Rationale:** It is simple, cacheable, compatible with host asset pipelines,
  and avoids inline-policy exceptions.
- **Consequences:** Same-origin asset routes become part of the deployment and
  must not serve attacker-controlled CSS at the referenced path.
- **Status:** proposed.

### Decision: preserve opaque server-owned continuation

- **Context:** Custom HTML could tempt implementers to echo OAuth fields as
  hidden inputs, as the mock server does.
- **Options considered:** Echo original parameters, sign a browser continuation,
  or retain the current opaque server-side interaction.
- **Decision:** Expose only the interaction handle and CSRF token.
- **Rationale:** Existing mutation, expiry, replay, and browser-binding checks
  remain effective and browser-editable protocol state does not expand.
- **Consequences:** Renderers cannot operate independently of the provider
  store, which is an intentional property.
- **Status:** proposed.

### Decision: keep client presentation minimal in v1

- **Context:** Consent pages benefit from a display name and logo, but current
  client storage contains only an ID and protocol/security fields.
- **Options considered:** Add schema fields now, allow renderer lookups, accept
  values from the request, or show the client ID.
- **Decision:** Always provide and display the client ID; allow static host
  labeling without adding persistence in this ticket.
- **Rationale:** Presentation customization does not justify an unreviewed client
  metadata and remote-logo subsystem.
- **Consequences:** Some clients have less friendly labels until a separate
  metadata design is implemented.
- **Status:** proposed.

## 15. Implementation phases and detailed tasks

### Phase 0: Evidence and contract design

This phase creates the reviewable specification before runtime changes.

1. Map strict provider rendering, authorize state transitions, interaction
   storage, CSRF, embedded construction, xapp mounts, and existing tests.
2. Distinguish the strict provider from the synthetic server template.
3. Capture primary sources for Go templating, CSP, CSRF, OIDC, OAuth security,
   and accessibility.
4. Define the trust boundary, public API, CSP, asset model, and failure policy.
5. Review this design with identity, application-host, frontend, and security
   owners.

Exit criterion: the API and security decisions are accepted before code is
written.

### Phase 1: Public contract and default renderer

1. Create `pkg/idpui` with action, login-reason, error-code, form, prompt, and
   page types.
2. Add the `InteractionRenderer` interface and compile-time implementation
   assertions.
3. Implement a default renderer with `html/template` and an embedded template.
4. Ensure every dynamic value is an ordinary string, never a trusted-content
   wrapper.
5. Add unit tests for every page shape and escaping context.
6. Add golden snapshots only for semantic review; do not make exact whitespace
   a compatibility contract.
7. Document the renderer contract with a minimal custom implementation.

Exit criterion: `pkg/idpui` is dependency-light, fully unit tested, and usable
without the provider.

### Phase 2: Provider and embedding integration

1. Add `UIConfig` to `pkg/embeddedidp.Options`.
2. Forward the renderer through `pkg/embeddedidp.New` into
   `fositeadapter.Options`.
3. Default nil configuration to the built-in renderer.
4. Replace scalar `renderInteraction` arguments with one page builder.
5. Render into a bounded buffer and propagate render/write errors.
6. Keep headers, status, cookies, and response commit in the provider.
7. Add renderer failure audit and metrics without recording page data.
8. Change the interaction CSP to permit only required same-origin style assets.
9. Verify discovery, token, UserInfo, health, and JSON error responses retain
   appropriate headers and media types.
10. Remove the old string renderer and manual `htmlEscape` if no other caller
    remains.

Exit criterion: existing embeddings work with the default renderer, and a test
renderer receives typed pages through the public option.

### Phase 3: Interaction error and state semantics

1. Classify resume failures as retryable, terminal user error, protocol error,
   or internal/unavailable error.
2. Rerender missing-login and invalid-credential failures with generic messages.
3. Never repopulate or expose the password.
4. Preserve or deliberately normalize the login field without leaking account
   existence.
5. Represent ordinary login, `prompt=login`, `max_age`, consent-only, and
   combined login/consent pages distinctly.
6. Validate the submitted action against server-required actions independent of
   rendered buttons.
7. Decide and test whether successful credential POST redirects use 303.
8. Preserve interaction expiry, browser binding, session binding, client
   generation, scope, redirect, and replay checks.
9. Add tests for renderer errors during both initial display and retry display.

Exit criterion: all recoverable failures return usable pages and no UI path can
satisfy a stored required action without the existing server checks.

### Phase 4: xapp theme and static assets

1. Create `cmd/tinyidp-xapp/internal/loginui` with a host-owned renderer.
2. Embed the template and CSS in the binary.
3. Mount the stylesheet under `/static/tinyidp/` using the existing outer host
   and static-asset conventions.
4. Validate theme URLs at renderer construction.
5. Implement the approved product typography, spacing, focus states, disclosure,
   actions, errors, and responsive layout.
6. Keep the document functional when CSS fails to load.
7. Use only same-origin assets permitted by the fixed CSP.
8. Wire the same renderer into development and production constructors.
9. Add an HTTP smoke test proving the HTML references a retrievable CSS asset
   with the correct content type.

Exit criterion: the real xapp login and consent flow is branded in development
and production composition without forking provider logic.

### Phase 5: Professional assurance tooling

1. Add a renderer conformance harness that parses output and verifies one POST
   form, expected action, opaque fields, conditional prompts, and allowed
   actions.
2. Reject or flag script elements, event-handler attributes, inline styles,
   dangerous URL schemes, protocol continuation inputs, and external origins.
3. Add a Go `analysis.Analyzer` that forbids `text/template` and trusted-content
   wrapper types in approved renderer packages.
4. Add an analyzer check for direct HTML `fmt.Fprintf` construction in provider
   interaction code.
5. Fuzz all public strings with markup, Unicode controls, long values, invalid
   UTF-8, URL-like strings, and template delimiters.
6. Fuzz the output conformance parser and bounded writer.
7. Add provider integration tests for missing session, existing session,
   `prompt=login`, `max_age`, `prompt=none`, consent approve/deny, invalid
   credentials, disabled user, expired interaction, replay, and concurrent
   terminal submissions.
8. Run browser tests with two accounts and password-manager-compatible fields.
9. Run automated accessibility checks and manual keyboard, focus, zoom, and
   screen-reader review.
10. Test CSP enforcement in a real browser, including blocked inline script,
    inline style, remote CSS, and framing.
11. Add render latency, failure count, and oversized-document observability.
12. Run `go test ./...`, race tests, fuzz budgets, repository static analysis,
    and existing model/verification scenarios.

Exit criterion: renderer customization has evidence at unit, integration,
browser, static-analysis, fuzz, accessibility, and security-header layers.

### Phase 6: Release and operations

1. Document the public API and host integration example.
2. Document the default CSP and the process for requesting a new resource type.
3. Document asset caching, reverse-proxy behavior, and `no-store` requirements.
4. Add a production doctor check that fetches the interaction page and declared
   stylesheet without logging tokens.
5. Capture screenshots for visual review without real credentials or secrets.
6. Run a canary deployment and observe render errors, login failures, consent
   outcomes, and asset failures.
7. Perform a security review of page source, headers, redirects, audit records,
   and browser network requests.
8. Record residual risks and obtain release approval.

Exit criterion: production rollout has an operator runbook, canary evidence,
review sign-off, and a rollback plan.

## 16. Verification strategy

### 16.1 Unit tests

The default renderer unit tests instantiate `InteractionPage` variants and
execute the template. They parse HTML structurally rather than searching with
regular expressions. Test data includes malicious client IDs and scopes:

```text
client ID: </strong><script>alert(1)</script>
scope:     openid"><img src=x onerror=alert(1)>
login:     alice@example.test" autofocus onfocus="alert(1)
```

The parsed document must contain those values only as text or escaped attribute
content, never as new elements or attributes.

### 16.2 Provider contract tests

A recording renderer captures a defensive copy of `InteractionPage` and then
delegates to the default renderer. This proves the public model for each state
without coupling provider tests to exact markup.

Key assertions include:

- `prompt=login` produces `LoginReasonPromptLogin` even with an active session.
- expired `max_age` produces `LoginReasonMaxAge`.
- `prompt=none` never invokes an interactive renderer.
- a consent-only interaction omits credential inputs.
- invalid credentials rerender with a generic error and empty password.
- renderer failure does not issue a code or redirect.
- custom markup cannot cause blank forced-login POSTs to reuse a session.

### 16.3 Conformance harness

The conformance harness should be reusable by downstream hosts:

```go
func TestRenderer(t *testing.T, renderer idpui.InteractionRenderer) {
    idpuitest.Conformance(t, renderer)
}
```

It renders a matrix of pages, parses each document, and reports contract
violations with element paths. It is a developer tool, not a claim that arbitrary
renderers are safe. Host code remains trusted and still requires review.

### 16.4 Static analysis

A project analyzer based on `go/analysis` should report:

- imports of `text/template` in renderer packages;
- conversions to `template.HTML`, `template.CSS`, `template.JS`,
  `template.HTMLAttr`, or `template.URL` from nonconstant values;
- calls to `fmt.Fprintf` or `io.WriteString` containing HTML literals inside the
  strict interaction-rendering package;
- renderer interfaces or implementations accepting `http.ResponseWriter` or
  `*http.Request`; and
- direct access to original authorization form fields from renderer packages.

The analyzer encodes architecture constraints that ordinary linters do not know.
It should include `analysistest` fixtures for positive and negative cases.

### 16.5 Fuzzing

Fuzz targets should cover:

```text
FuzzDefaultRendererEscaping
FuzzRendererConformanceParser
FuzzBoundedRenderWriter
FuzzInteractionPageActionMatrix
```

Useful properties are:

- rendering never panics;
- output stays within the configured bound;
- untrusted input never creates script, style, iframe, object, embed, or event
  handler nodes;
- the form action remains exactly provider supplied;
- password value attributes are absent; and
- unexpected action values are rejected server-side.

### 16.6 Browser and accessibility tests

The browser suite should run the real xapp in tmux, exercise the actual TLS or
loopback configuration, and capture response headers and network requests. It
should verify:

- CSS loads from `/static/` and no third-party request occurs.
- Inline script and styles are blocked under CSP.
- The page cannot be framed.
- Tab order is username, password, approve/continue, then deny when applicable.
- Visible focus remains clear at 200% and 400% zoom.
- Errors are announced and associated with fields.
- Password managers recognize the fields.
- Logout followed by forced login requires credentials.

### 16.7 Review commands

Expected implementation review commands include:

```bash
go fmt ./...
go test ./pkg/idpui ./internal/fositeadapter ./pkg/embeddedidp ./cmd/tinyidp-xapp -count=1
go test -race ./internal/fositeadapter ./pkg/embeddedidp -count=1
go test ./... -count=1
go vet ./...
go run ./cmd/tinyidp-static ./...
```

The exact analyzer command may change when the static-analysis ticket exposes a
stable driver. Use the top-level `go.work`; do not create a nested module or
private Go cache.

## 17. Risks and alternatives

### 17.1 Same-origin CSS risk

`style-src 'self'` allows styles from the entire origin, not just the intended
path. A host that serves attacker-controlled CSS from the same origin weakens
the assurance of the login page. Mitigations are immutable embedded assets,
reviewed static routing, no user-controlled uploads under executable CSS media
types, and browser CSP tests.

If that deployment assumption becomes false, a later version can use a CSP hash
for a fixed inline stylesheet or a dedicated identity origin. That is a policy
change, not a template option.

### 17.2 Trusted renderer risk

A trusted renderer can deliberately mislabel an approve button or omit important
disclosure. The narrow API prevents protocol compromise but cannot make a
malicious credential UI honest. Code review, conformance tests, visual review,
and controlled deployment remain required.

### 17.3 Public API stability

An exported struct is easy to use but difficult to extend if downstream code
constructs it directly. The provider is expected to construct `InteractionPage`;
renderers consume it. Additive fields are generally safe, while renaming actions
or changing meaning is not. Constants need precise documentation and tests.

### 17.4 Combined login and consent

The current initial page can combine credentials and consent disclosure because
consent policy may depend on the authenticated user. Splitting this into a
login-first then consent-only interaction would improve clarity but requires a
carefully modeled server-side interaction transition and session rebinding.

This design does not silently invent that state transition. Phase 3 must either
preserve the combined behavior with explicit page semantics or create a separate
model-checked design for advancing a pending interaction.

### 17.5 Renderer localization

Localization is not included in v1. Passing raw request headers to the renderer
is rejected. A future design can normalize supported locales in the provider,
select one trusted catalog, and pass a locale identifier or already resolved
messages.

### 17.6 Alternatives rejected for v1

| Alternative | Reason not selected |
|---|---|
| Edit `internal/server/static/login.html` | It changes only the synthetic server. |
| Fork `internal/fositeadapter` | It duplicates security-sensitive protocol code. |
| Filesystem template path | It adds runtime mutation and deployment ambiguity. |
| Raw `http.Handler` | It grants headers, cookies, redirects, request, and streaming authority. |
| Goja-rendered identity page | It expands VM and scripting trust into credential handling before a concrete need. |
| Inline CSS with `unsafe-inline` | It weakens CSP and is unnecessary. |
| Remote theme CDN | It leaks visits and adds a mutable external dependency. |
| Client-provided branding URL | It introduces phishing, tracking, validation, and persistence problems. |

## 18. Open questions for design review

1. Should successful credential POSTs be changed from 302 to an explicit 303 in
   the same implementation, given RFC 9700 guidance?
2. Should the first CSP permit same-origin images and fonts, or leave both at
   `'none'` until a reviewed theme requires them?
3. Is a 256 KiB rendered-document limit sufficient for every intended embedded
   host?
4. Should invalid credentials rerender with HTTP 401 or HTTP 200? The UX,
   password-manager, monitoring, and cache consequences should be tested.
5. Should normalized login text be repopulated after failure, or omitted for
   privacy on shared devices?
6. Should a renderer implementation label be a required startup option for
   metrics, or derived from its Go type?
7. Does the combined login/consent flow remain acceptable for v1, or should a
   separate ticket model a two-step interaction transition?

None of these questions authorizes a raw compatibility layer or a weaker
security default. They identify decisions that need explicit acceptance before
implementation.

## 19. Intern onboarding path

An intern should use this sequence:

1. Read `pkg/idpstore/types.go`, focusing on `InteractionRecord`, required
   actions, sessions, and consent.
2. Read `internal/fositeadapter/interaction.go` to understand opaque
   continuation and request reconstruction.
3. Read `internal/fositeadapter/csrf.go` to understand browser binding and the
   form token.
4. Trace `beginAuthorize`, `resumeAuthorize`, and `finishAuthorize` in
   `internal/fositeadapter/provider.go`.
5. Run the interaction hardening tests individually and change no code yet.
6. Read `pkg/embeddedidp/options.go` and `provider.go` to understand the public
   construction boundary.
7. Read xapp composition in `cmd/tinyidp-xapp/development_app.go` and identify
   `/idp/` and `/static/` ownership.
8. Read the locally preserved Go template, CSP, CSRF, OIDC, OAuth BCP, and WCAG
   sources.
9. Implement Phase 1 as an isolated package and request review before adapter
   integration.
10. Keep the ticket diary current with commands, failures, tests, and commits.

The first review should concentrate on authority boundaries, not visual design.
Once the provider cannot be influenced beyond the typed view model, frontend
work becomes substantially easier to review.

## 20. File and source reference map

### 20.1 Repository files

- `internal/fositeadapter/provider.go:348-374` constructs the strict handler,
  security headers, and endpoint registrations.
- `internal/fositeadapter/provider.go:407-615` contains authorization begin and
  resume behavior.
- `internal/fositeadapter/provider.go:941-959` is the current hard-coded renderer.
- `internal/fositeadapter/interaction.go:18-147` defines transient fields,
  server-owned interaction creation, bindings, and request reconstruction.
- `internal/fositeadapter/csrf.go:12-47` generates and validates interaction-bound
  CSRF values.
- `pkg/idpstore/types.go:14-29` shows that clients currently have no presentation
  metadata.
- `pkg/idpstore/types.go:175-224` defines required actions and interaction
  records.
- `pkg/embeddedidp/options.go:45-59` is the public configuration gap.
- `pkg/embeddedidp/provider.go:39-73` validates options and constructs the strict
  adapter.
- `internal/server/embed.go` and `internal/server/static/login.html` define the
  separate synthetic-server template.
- `internal/fositeadapter/interaction_hardening_test.go` contains forced-login,
  consent, replay, expiry, browser-binding, and no-protocol-continuation tests.
- `internal/fositeadapter/hardening_test.go` verifies security headers exist.
- `cmd/tinyidp-xapp/development_app.go:169-242` composes `/idp/`, native auth,
  static/frontend behavior, Goja routes, and durable objects.
- `cmd/tinyidp-xapp/production_app.go:58-73` constructs the production embedded
  provider.

### 20.2 Preserved external sources

- [`go-html-template-security-model.md`](../sources/go-html-template-security-model.md)
  explains contextual autoescaping and its trusted-template/untrusted-data model.
- [`w3c-content-security-policy-level-3.md`](../sources/w3c-content-security-policy-level-3.md)
  defines source lists, `style-src`, nonces, hashes, and fallback behavior.
- [`owasp-csrf-prevention-cheat-sheet.md`](../sources/owasp-csrf-prevention-cheat-sheet.md)
  documents hidden form tokens, signed binding, and token leakage constraints.
- [`openid-connect-core-1.0.md`](../sources/openid-connect-core-1.0.md)
  defines `prompt`, `max_age`, `auth_time`, and authorization endpoint behavior.
- [`rfc-9700-oauth-2-security-bcp.md`](../sources/rfc-9700-oauth-2-security-bcp.md)
  provides current OAuth security guidance for authorization UI, framing, and
  credential-post redirects.
- [`wcag-2.2-error-identification.md`](../sources/wcag-2.2-error-identification.md)
  explains visible, textual input error requirements.
- [`wcag-2.2-accessible-authentication-minimum.md`](../sources/wcag-2.2-accessible-authentication-minimum.md)
  explains accessible authentication requirements and password-manager-friendly
  behavior.

## 21. Final design statement

Custom login HTML is not a static-file substitution. It is a controlled view of
a server-owned authorization state machine. The preferred implementation makes
that control explicit: tiny-idp builds a minimal typed page, a trusted host
renderer converts it to HTML in a bounded buffer, and tiny-idp alone commits the
HTTP response and decides whether the interaction can advance.

This division gives application authors meaningful visual control without
duplicating the provider or turning templates into protocol handlers. It also
creates stable surfaces for conformance tests, Go static analysis, fuzzing,
browser CSP verification, accessibility review, and future interaction types.
