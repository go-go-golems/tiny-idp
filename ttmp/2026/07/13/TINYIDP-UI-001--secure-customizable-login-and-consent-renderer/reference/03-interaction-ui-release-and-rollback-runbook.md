---
Title: Interaction UI Release and Rollback Runbook
Ticket: TINYIDP-UI-001
Status: active
Topics:
    - oidc
    - identity
    - security
    - go
    - architecture
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp-xapp/doctor.go
      Note: Production doctor command described by this runbook
    - Path: internal/fositeadapter/rendering.go
      Note: Bounded rendering, metrics, and failure behavior operated by this runbook
    - Path: pkg/embeddedidp/options.go
      Note: Public rollback and renderer configuration boundary
ExternalSources: []
Summary: Operator checks, canary procedure, residual risks, and rollback steps for interaction UI customization.
LastUpdated: 2026-07-13T18:38:01.608802236-04:00
WhatFor: ""
WhenToUse: ""
---


# Interaction UI Release and Rollback Runbook

## Goal

Provide the operator procedure for qualifying, canarying, observing, approving,
and rolling back a custom tiny-idp login and consent renderer.

## Context

Renderer customization changes a security-sensitive browser surface but not the
provider's protocol state. The release unit nevertheless includes provider
integration changes: strict action validation, bounded output, retry pages,
same-origin style CSP, sanitized render observability, and 303 redirects after
credential POSTs.

The rollout must distinguish presentation rollback from protocol rollback. A
visual problem can be corrected by selecting the built-in renderer. Security
invariants must not be weakened to make a custom page work.

## Quick Reference

### Required preflight

Run from the repository root:

```bash
go test ./...
go test -race ./pkg/idpui/... ./pkg/embeddedidp ./internal/fositeadapter ./cmd/tinyidp-xapp -count=1
make idpui-analyzer
```

Run the three focused fuzz targets with an appropriate CI budget:

```bash
go test ./pkg/idpui -run '^$' -fuzz '^FuzzDefaultRendererEscapingAndConformance$' -fuzztime=30s
go test ./pkg/idpui/idpuitest -run '^$' -fuzz '^FuzzConformanceParserNeverPanics$' -fuzztime=30s
go test ./internal/fositeadapter -run '^$' -fuzz '^FuzzBoundedInteractionBuffer$' -fuzztime=30s
```

For an initialized product state, run the in-process doctor:

```bash
go run ./cmd/tinyidp-xapp doctor \
  --state-root /secure/path/to/state \
  --output table
```

The doctor must report `interaction-document` and
`interaction-stylesheet` as `ok`. It checks the public origin, status, CSP,
`no-store`, output bounds, declared `/static/` path, CSS status, and CSS media
type. It never emits the document or protocol values.

### Canary sequence

1. Deploy one instance or a bounded percentage using the new binary.
2. Do not change issuer, cookie, client, redirect, proxy, or key configuration in
   the same rollout.
3. Run the production doctor against the canary state before routing users.
4. Fetch `/auth/login` in a clean browser profile.
5. Test missing-session login, invalid credentials, successful login, denial,
   logout, `prompt=login`, and `max_age=0`.
6. Verify the page CSS loads from the same origin and no third-party request is
   emitted.
7. Verify a successful credential POST produces 303 before the OIDC callback.
8. Test two accounts in isolated browser profiles and confirm application
   subjects remain distinct.
9. Observe metrics and audit events for at least the agreed canary interval.
10. Expand traffic only after identity, host, frontend, accessibility, and
    security reviewers sign off.

### Metrics

Observe `InteractionRenderStats` as monotonically increasing process-local
counters:

- attempts;
- successes;
- failures;
- oversized documents;
- empty documents;
- response-write failures;
- total render latency;
- maximum render latency.

Alert on any oversized or empty document. Investigate render failures and
response-write failures immediately. Latency thresholds must be derived from the
deployment SLO; do not add client or user labels to obtain more detail.

### Audit review

Expected fixed render failure reasons are:

- `invalid_page`;
- `renderer_failed`;
- `document_too_large`;
- `empty_document`;
- `response_write_failed`.

Audit records must not contain template error strings, HTML, login identifiers,
passwords, CSRF values, cookies, interaction handles, OAuth state, nonce, PKCE
verifiers, authorization codes, or tokens.

### Header review

The interaction response must contain:

```text
Content-Type: text/html; charset=utf-8
Cache-Control: no-store
Pragma: no-cache
Content-Security-Policy: default-src 'none'; style-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

The CSS response must use a CSS media type. A proxy may cache the embedded CSS
according to its short public policy, but it must not cache interaction HTML.

### Page-source review

Inspect structure without copying values into the ticket. Confirm:

- one POST form with the provider authorize action;
- exactly one opaque interaction hidden field;
- exactly one CSRF hidden field;
- no other hidden input;
- no scripts, style blocks, inline styles, event attributes, frames, images,
  media, SVG, objects, or remote resources;
- exactly the server-required action buttons;
- `formnovalidate` on deny;
- no password `value` attribute;
- visible labels and correct autocomplete values;
- an alert role on retry errors.

Use `idpuitest` and the browser probe for repeatable checks instead of pasting
real page source.

### Rollback triggers

Rollback the custom renderer if any of the following occurs:

- CSP or stylesheet failures make authentication unusable;
- render failure, empty-document, oversized-document, or write-failure counters
  increase unexpectedly;
- authentication or consent outcome rates diverge materially from baseline;
- a hidden protocol field differs from the provider model;
- the password is retained in markup, logging, screenshots, or retry state;
- third-party requests, script, inline style, or frame exposure appears;
- keyboard, focus, contrast, reflow, labels, or error identification regress;
- the proxy caches authentication HTML or removes security headers.

### Presentation-only rollback

1. Configure `embeddedidp.UIConfig.Renderer` as nil, or deploy the previous host
   binary that used the built-in renderer.
2. Keep the same issuer, store, signing keys, token secret, clients, sessions,
   consent records, and interaction records.
3. Remove the host-specific CSS route only after the built-in renderer is active.
4. Re-run the interaction doctor and one clean-browser login.
5. Observe render metrics, login outcomes, and audit delivery.

No database migration or token revocation is required for this presentation-only
rollback.

Do not roll back strict POST actions, CSRF validation, forced reauthentication,
must-change-password enforcement, bounded rendering, generic credential errors,
303 redirect handling, provider-owned headers, or audit sanitization.

### Full binary rollback

If a non-presentation regression requires the previous binary:

1. Stop new traffic to the canary.
2. Gracefully stop the process.
3. Verify database schema compatibility before starting the previous binary.
4. Preserve audit logs and failed-canary evidence.
5. Start the approved previous binary with unchanged secrets and issuer.
6. Run readiness and a clean-browser login.
7. Record the rollback commit, binary digest, start time, and reason outside logs
   containing request data.

### Residual risks

- Same-origin CSS is trusted. Another host route capable of serving
  attacker-controlled CSS under `/static/` would weaken the page even though
  the template is safe.
- A host-supplied renderer is trusted code and can intentionally omit controls
  or misrepresent text. Provider checks prevent authorization bypass, but users
  can still be confused or phished by a malicious host.
- The conformance checker is deliberately conservative and does not prove every
  possible browser parsing differential.
- Process-local counters reset on restart and need an external collector for
  long-term trends.
- Automated accessibility evidence does not replace assistive-technology and
  human review.
- A local canary does not prove production reverse-proxy behavior.

### Release approval record

Before production expansion, record names and timestamps for:

- identity protocol review;
- embedding/host review;
- frontend review;
- accessibility review;
- security review;
- operations owner;
- rollback owner.

This ticket currently contains successful local canary evidence. Production
deployment approval is intentionally external to the code commit and must be
recorded by the responsible humans.

## Usage Examples

Use this document as a checklist during a release review. Link the exact Git
commit, binary digest, doctor output, browser-probe summary, metric observation
window, and reviewer approvals. Store no credentials or protocol values in that
record.

When a check fails, stop expansion. Presentation failures should first use the
nil-renderer rollback because it preserves protocol behavior and persisted state.
Do not loosen security policy to make an asset or template pass.

## Related

- `docs/interaction-rendering.md`
- `reference/02-browser-accessibility-and-canary-evidence.md`
- `reference/01-investigation-diary.md`
- `design-doc/01-secure-interaction-rendering-analysis-design-and-implementation-guide.md`
