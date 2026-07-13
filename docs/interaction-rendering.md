# Custom interaction rendering

tiny-idp lets an embedding host customize the browser page used for login and
consent. The host controls presentation; tiny-idp continues to control OAuth,
OpenID Connect, credentials, consent, CSRF, interaction state, cookies, headers,
status codes, redirects, audit events, and token issuance.

This boundary is intentionally narrower than an HTTP handler:

```go
type InteractionRenderer interface {
    RenderInteraction(
        ctx context.Context,
        dst io.Writer,
        page InteractionPage,
    ) error
}
```

The renderer does not receive `http.ResponseWriter`, `*http.Request`, the stored
interaction record, raw OAuth parameters, cookies, or a password value. A host
cannot authorize a request by changing HTML. The provider validates every POST
against its server-side pending interaction.

## Configuring a renderer

Supply the implementation when constructing the embedded provider:

```go
renderer, err := myloginui.New(myloginui.Options{
    ProductName:   "Example Product",
    StylesheetURL: "/static/identity/login.css",
})
if err != nil {
    return err
}

provider, err := embeddedidp.New(ctx, embeddedidp.Options{
    Issuer:        "https://app.example.test/idp",
    Mode:          embeddedidp.ProductionMode,
    Store:         store,
    Authenticator: authenticator,
    Cookie: embeddedidp.CookieConfig{
        Secure: true,
    },
    Token: embeddedidp.TokenConfig{SecretKey: tokenSecret},
    UI:    embeddedidp.UIConfig{Renderer: renderer},
})
```

A nil renderer selects tiny-idp's built-in `html/template` implementation. The
built-in markup is semantically stable, but its exact whitespace and element
layout are not a compatibility API.

## Rendering contract

`InteractionPage` contains a complete, already-derived presentation model:

- `DocumentTitle` is a nonempty display title.
- `Form.ActionURL` is the provider-owned absolute authorize endpoint.
- `Form.Interaction` is an opaque handle. It is not an encoded OAuth request.
- `Form.CSRFToken` is a browser-bound form token.
- `Form.Actions` is a closed list of `continue`, `approve`, or `deny` values.
- `Login` is present only when credentials are required.
- `Consent` is present only when access disclosure is required.
- `Error` is a stable, non-sensitive public error category and summary.

Renderers must call `page.Validate()` before executing a template. The provider
also validates the page before invoking custom code and passes a defensive clone.

The form must preserve these exact field names:

| Meaning | Field |
| --- | --- |
| Opaque pending interaction | `interaction` |
| CSRF token | `csrf_token` |
| Submitted decision | `action` |
| Login identifier | `login` |
| Password | `password` |

Do not echo `client_id`, `redirect_uri`, `scope`, `state`, `nonce`, PKCE values,
or other OAuth parameters as hidden inputs. The provider reconstructs and
revalidates them from server-side state.

## Safe template example

Use `html/template`, not `text/template`, and pass every string as ordinary
data:

```go
type Renderer struct {
    template *template.Template
}

var _ idpui.InteractionRenderer = (*Renderer)(nil)

func (r *Renderer) RenderInteraction(
    ctx context.Context,
    dst io.Writer,
    page idpui.InteractionPage,
) error {
    if err := ctx.Err(); err != nil {
        return err
    }
    if err := page.Validate(); err != nil {
        return errors.Wrap(err, "validate interaction page")
    }
    return r.template.ExecuteTemplate(dst, "interaction", page.Clone())
}
```

Do not convert dynamic strings to `template.HTML`, `template.CSS`,
`template.JS`, `template.URL`, `template.HTMLAttr`, or related trusted-content
types. Such a conversion suppresses contextual escaping and expands the host's
trusted computing base.

The renderer may return an error. tiny-idp buffers its output, enforces a 256 KiB
document limit, and commits HTML only after rendering succeeds. Failure produces
a generic 500 response and a sanitized `interaction.render_failed` audit event.

## Actions and browser validation

Render one submit button for every value in `page.Form.Actions`. Submit the
original value even if the visible label is localized.

The deny button must carry `formnovalidate`. Otherwise HTML `required`
credential fields can prevent a user from denying a combined login and consent
request:

```html
<button
  type="submit"
  name="action"
  value="deny"
  formnovalidate>Denied</button>
```

Browser validation is a usability feature, not authorization evidence. The
provider rejects missing, unknown, replayed, expired, or context-inappropriate
actions independently of rendered controls.

## Credential fields

Use visible labels and password-manager-compatible autocomplete values:

```html
<label for="tinyidp-login">Username</label>
<input id="tinyidp-login" name="login" autocomplete="username" required>

<label for="tinyidp-password">Password</label>
<input id="tinyidp-password" name="password"
       type="password" autocomplete="current-password" required>
```

The password input must never have a `value` attribute. `InteractionPage` has no
password value member. A retry page may retain the normalized login identifier,
but invalid users and invalid passwords receive the same public error category.

Forced login states are explicit:

- `session_missing`: no accepted browser session is available.
- `prompt_login`: the client supplied `prompt=login`.
- `max_age`: the accepted session authentication time is too old.
- `step_up`: the server requires an additional authentication step.

Hosts may vary explanatory text, but must not hide the fact that fresh
credentials are required.

## Static assets and CSP

The reviewed interaction CSP is:

```text
default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'
```

It permits one class of optional resource: same-origin external stylesheets.
Scripts, inline styles, images, fonts, frames, objects, media, connections, and
third-party resources remain denied.

Mount identity assets on the outer application host under `/static/`:

```go
mux := http.NewServeMux()
mux.Handle("/idp/", provider.Handler())
mux.Handle("GET /static/identity/", renderer.AssetsHandler())
mux.Handle("/", applicationHandler)
```

Validate configured asset paths at startup. Accept only clean root-relative
paths below `/static/`. Reject:

- absolute and scheme-relative URLs;
- query strings and fragments;
- `..` traversal and encoded traversal;
- backslashes;
- user information or authority components.

Serve CSS with `text/css; charset=utf-8`. The xapp reference renderer uses a
five-minute public cache. The interaction HTML itself is always `Cache-Control:
no-store` and `Pragma: no-cache`.

If a reverse proxy rewrites `/idp/` or `/static/`, it must preserve:

- the public issuer and exact form action origin;
- CSP, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, and
  `X-Content-Type-Options: nosniff`;
- `no-store` on the interaction document;
- the CSS media type;
- secure cookie attributes and paths;
- 303 redirects emitted after successful credential POSTs.

Do not inject scripts, analytics, remote fonts, consent managers, debugging
overlays, or HTML-rewriting middleware into the identity path.

## Conformance checks

Downstream renderers can reuse `pkg/idpui/idpuitest`:

```go
document, violations, err := idpuitest.RenderAndCheck(ctx, renderer, page)
if err != nil {
    t.Fatal(err)
}
if len(violations) != 0 {
    t.Fatalf("renderer violations: %v", violations)
}
_ = document
```

The parsed-DOM checker reports:

- script, style, frame, object, media, image, SVG, and other active elements;
- event-handler and inline-style attributes;
- `javascript:`, `data:`, and `vbscript:` URLs;
- external origins other than the exact provider form action;
- missing, duplicated, changed, or unexpected protocol hidden fields;
- unknown, missing, or duplicated action controls;
- password value retention;
- missing labels, autocomplete metadata, alert semantics, and deny bypass.

Run the Go analyzer for implementation-level checks:

```bash
make idpui-analyzer
```

It rejects `text/template`, dynamic trusted-content conversions, raw HTML writes,
and renderer interfaces coupled to `http.ResponseWriter` or `*http.Request`.
`make lint` includes this analyzer.

## Observability

The embedded provider exposes bounded process-local metrics:

```go
stats := provider.InteractionRenderStats()
```

`idpui.RenderStats` includes attempts, successes, failures, oversized documents,
empty documents, response-write failures, total latency, and maximum latency.
It intentionally has no client, user, login, interaction, route, or error-text
labels.

Render failure audit reasons are from a fixed set such as `invalid_page`,
`renderer_failed`, `document_too_large`, `empty_document`, and
`response_write_failed`. Never add raw template errors or page values to an
audit record.

## Production verification

The xapp doctor can inspect initialized state and exercise the interaction page
and CSS entirely in process:

```bash
go run ./cmd/tinyidp-xapp doctor \
  --state-root /secure/path/to/state \
  --output table
```

The check validates status, origin, CSP, cache policy, document bounds, declared
stylesheet path, CSS response status, and media type. It never prints or stores
cookies, CSRF tokens, interaction handles, HTML, or credentials.

Before rollout, also run:

```bash
go test ./...
go test -race ./pkg/idpui/... ./pkg/embeddedidp ./internal/fositeadapter ./cmd/tinyidp-xapp -count=1
make idpui-analyzer
```

Use the ticket browser probe for real Chromium checks. Supply only dedicated
development credentials and do not save its command line or shell history in a
production environment.

## Rollback

The safest rendering-only rollback is to omit `UI.Renderer`, which selects the
built-in renderer while preserving all provider state and protocol behavior.
If the host also introduced an asset route, remove that route after switching
the renderer. No database migration or token invalidation is required for this
presentation-only rollback.

Do not roll back the strict action validation, bounded rendering, forced-login
checks, CSRF binding, 303 behavior, or provider-owned security headers as part
of a visual rollback.
