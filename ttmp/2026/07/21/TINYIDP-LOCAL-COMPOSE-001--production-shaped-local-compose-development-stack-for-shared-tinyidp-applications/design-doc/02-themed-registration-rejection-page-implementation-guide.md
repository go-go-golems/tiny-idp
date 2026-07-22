---
Title: Themed registration rejection page implementation guide
Ticket: TINYIDP-LOCAL-COMPOSE-001
Status: active
Topics:
    - oidc
    - tiny-idp
    - kubernetes
    - local-development
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Registration rejection decision and validated client context
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Bounded browser rendering boundary
    - Path: repo://internal/productionui/renderer.go
      Note: Per-client approved theme selection
    - Path: repo://pkg/idpui/types.go
      Note: Provider-owned safe presentation models
ExternalSources: []
Summary: A narrow design for replacing raw registration rejection responses with bounded, client-themed HTML while preserving OAuth and CSRF security boundaries.
LastUpdated: 2026-07-21T18:11:24.986730978-04:00
WhatFor: Implement and review the browser error presentation contract used when TinyIDP refuses an account-creation request.
WhenToUse: Read before changing registration rejection responses, production UI templates, or per-client error-page theming.
---


# Themed registration rejection page implementation guide

## Executive Summary

TinyIDP currently uses `http.Error` when the security guard rejects a
registration POST. That response is technically correct but bypasses the
production renderer, so the browser receives unstyled plain text. This change
adds a small provider-owned browser-error model and renderer interface. The
default renderer emits safe standalone HTML; the production renderer selects
the same approved stylesheet as login and signup pages using the interaction's
already-validated client ID.

The first consumer is the `origin_rejected` registration path. OAuth error
redirects are not changed, and the error renderer receives no request object,
cookie, CSRF token, continuation, redirect URI, exception, or arbitrary HTML.

## Problem Statement

The failing request reaches `resumeAuthorize`, passes CSRF and durable
interaction validation, reconstructs the authorization request, and resolves
the registered client. The registration-specific origin check can then reject
the request. Its current response is:

```text
HTTP/2 403
Content-Type: text/plain; charset=utf-8

registration request was not accepted
```

Because `http.Error` never crosses `internal/productionui.Renderer`, the client
theme in `themes.json` is not selected. The result is also inconsistent with
the surrounding signup workflow, which is rendered as a complete HTML page.

The provider must not solve this by sending the submitted signup form back to
the browser. A rejected-origin request is untrusted, and echoing fields would
increase disclosure and injection risk. It should render only fixed public
copy and the public client identifier already stored in the interaction.

## Proposed Solution

Add `idpui.BrowserErrorPage` and `idpui.BrowserErrorRenderer`:

```go
type BrowserErrorPage struct {
    DocumentTitle string
    ClientID      string
    Heading       string
    Summary       string
}

type BrowserErrorRenderer interface {
    RenderBrowserError(context.Context, io.Writer, BrowserErrorPage) error
}
```

`Validate` enforces non-empty, bounded plain-text fields. `ClientID` is used
only as the key into the server-loaded theme catalog. The templates use
`html/template`, contain no form or script, and load only the approved
same-origin stylesheet route.

```text
registration POST
       |
       v
CSRF + interaction + client validation
       |
       v
same-origin browser guard ---- accepted ----> signup continuation
       |
    rejected
       |
       v
BrowserErrorPage{ClientID, fixed public copy}
       |
       +--> DefaultRenderer ----> standalone safe HTML
       |
       `--> productionui.Renderer
                 |
                 `--> Catalog.Resolve(ClientID) --> approved local CSS
```

The provider buffers and size-limits the rendered document before writing it,
sets `Cache-Control: no-store`, `Pragma: no-cache`, the existing restrictive
CSP, and the original `403 Forbidden` status. If validation or rendering
fails, a minimal `http.Error` response remains the last-resort fallback; this
prevents an error renderer failure from recursively invoking itself.

## Design Decisions

- Use a separate terminal error model instead of manufacturing an
  `InteractionPage`. An interaction page requires a live form, actions,
  interaction handle, and CSRF token; a terminal rejection must expose none of
  those capabilities.
- Keep copy provider-owned. Callers choose a closed error case, not arbitrary
  HTML. The initial helper constructs fixed registration-rejection text.
- Theme by validated client ID. At the target branch, the interaction record
  and current client generation have already been checked.
- Preserve HTTP status semantics. Styling does not turn a rejected request
  into `200 OK`.
- Preserve OAuth response ownership. Valid OAuth errors that belong at a
  registered redirect URI continue through Fosite.
- Do not add a return link in this first increment. A callback URI is not an
  application home page, and inventing a destination would create a new
  redirect/navigation policy.

## Alternatives Considered

- Re-render the signup workflow: rejected because an origin failure makes the
  submitted browser context unsuitable for continuation and field replay.
- Reuse `InteractionPage`: rejected because it would require dummy form
  authority and weaken that model's validation invariants.
- Put inline CSS in `http.Error`: rejected because it bypasses the catalog,
  conflicts with CSP, and duplicates application styling.
- Change all `http.Error` calls at once: rejected as too broad. Different
  failures occur before client validation or belong to OAuth redirect rules.

## Implementation Plan

1. Add and test the bounded browser-error model and renderer interface.
2. Add embedded default and production HTML templates.
3. Configure the provider with a browser-error renderer, defaulting safely.
4. Add a buffered `renderBrowserError` boundary with no-store headers, strict
   CSP, status preservation, metrics, audit on renderer failure, and plain-text
   fallback.
5. Replace only the registration origin rejection's `http.Error` call.
6. Assert `403`, HTML content, safe public copy, absence of submitted identity
   fields, and production theme stylesheet selection.
7. Rebuild the local IDP and probe the rejection path through Caddy.

## Open Questions

Later work may classify other direct browser failures and add safe navigation
back to an application-owned landing URL. Neither is required for this fix.
They should not be inferred from OAuth callback URLs.

## References

- `internal/fositeadapter/provider.go`: registration security decision.
- `internal/fositeadapter/rendering.go`: bounded HTTP rendering boundary.
- `pkg/idpui/types.go`: provider-owned presentation models.
- `pkg/idpui/default_renderer.go`: dependency-free renderer.
- `internal/productionui/renderer.go`: client-specific theme selection.
- `internal/productionui/catalog.go`: approved local CSS catalog.
