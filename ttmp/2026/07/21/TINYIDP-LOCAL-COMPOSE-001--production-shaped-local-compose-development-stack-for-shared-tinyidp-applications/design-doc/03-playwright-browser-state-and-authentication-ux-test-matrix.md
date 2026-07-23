---
Title: Playwright browser-state and authentication UX test matrix
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
    - Path: repo://examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts
      Note: |-
        Matrix Goja invitation and email-limit evidence
        Browser evidence for UX-004 resolution
    - Path: repo://examples/tinyidp-shared-two-apps/compose.yaml
      Note: Production-shaped local topology exercised by the browser suite
    - Path: repo://examples/tinyidp-shared-two-apps/open-signup.js
      Note: Per-client signup workflow under test
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Authorization, sessions, account selection, and logout transitions
        Authorization session and account-selection transitions
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: |-
        Signup continuation and recoverable validation rendering
        Signup continuation and validation presentation boundary
    - Path: repo://internal/productionui/templates/workflow.html
      Note: |-
        Themed workflow form and accessible field errors
        Themed accessible workflow form
    - Path: repo://pkg/sqlitestore/email_challenge.go
      Note: Root cause and durable resolution of UX-004
ExternalSources: []
Summary: A browser-level verification strategy for every supported signup, login, remembered-account, logout, and validation transition in the shared TinyIDP local stack.
LastUpdated: 2026-07-21T19:06:06.136211008-04:00
WhatFor: Implement and review a Playwright suite that treats navigation quality and themed error recovery as observable authentication requirements.
WhenToUse: Read before adding browser tests, changing authentication navigation, or deciding how a TinyIDP failure should appear to a user.
---




# Playwright browser-state and authentication UX test matrix

## Executive Summary

TinyIDP's unit and HTTP integration tests establish protocol and storage
correctness, but they do not prove that a browser can complete a coherent
journey across Message Desk, Goja Auth, Caddy, TinyIDP, and Mailpit. The
reported duplicate-email failure demonstrates the gap: each subsystem could
accept or reject its local operation correctly while the browser still ended
on an unstyled `text/plain` response with no recovery instruction.

This guide defines a Playwright suite for the real local HTTPS Compose stack.
The suite models browser cookies, the active TinyIDP identity, remembered
identities, each relying party's independent session, signup workflow state,
and validation outcomes. Every test asserts both the state transition and the
page presented to the user. A rejection is successful only when it preserves
security invariants *and* produces safe, themed, actionable HTML.

The first defect already has concrete evidence. At `2026-07-21T23:03:35Z`,
the audit log recorded `workflow.signup.resume_rejected` with reason
`continuation_unavailable`. The provider returned the literal fallback
`registration request was not accepted`. The browser suite must reproduce and
classify that transition before its repair is considered complete.

## Problem Statement

Authentication behavior is a product of several independent state machines:

- The browser has host-scoped cookies for Message Desk, Goja Auth, and
  TinyIDP. Logging out of one relying party does not necessarily log out of
  TinyIDP or the other relying party.
- TinyIDP has a current session plus durable remembered sessions. A user may
  add a new identity, select an old identity, remove one from the chooser, or
  request fresh authentication.
- Signup is a durable multi-request workflow: identity fields, email
  challenge, password selection, native account commit, consent, and OAuth
  callback.
- Each relying party has different signup policy and theming. Message Desk is
  open signup; Goja Auth requires an invitation.
- Browser requests carry Origin, Fetch Metadata, CSRF, interaction, and
  continuation bindings. A stale tab, replay, cookie change, or duplicate
  submission can invalidate one of these bindings.

Testing individual handlers cannot cover the cross-product. Ad hoc manual
testing also loses exact evidence, tends to exercise only the happy path, and
does not prevent a raw response from returning later.

The quality rule for this project is therefore explicit:

> No expected user mistake or ordinary browser-state transition may terminate
> on an unstyled response, raw OAuth error, unexplained rejection, or page
> without a safe next action.

Security attacks and irrecoverable stale state may terminate the workflow,
but their response must still be bounded, non-sensitive, themed HTML when a
validated client context is available.

## Proposed Solution

### 1. Test the deployed system boundary

Place a small Playwright project under
`examples/tinyidp-shared-two-apps/browser-tests/`. It drives the externally
visible origins through Caddy:

```text
Playwright browser context
        |
        | HTTPS, trusted persistent local CA
        v
 Caddy :8443
   |             |                 |
   v             v                 v
Message Desk   Goja Auth         TinyIDP ------> Mailpit
 RP session     RP session       IDP cookies     email code
```

Tests must not bypass Caddy, inject database rows for ordinary journeys, or
call internal handler methods. Administrative fixture setup may use supported
operator commands when a scenario specifically requires a pre-existing user
or invitation.

### 2. Represent state explicitly

Each test description names its starting state and transition. Reusable
fixtures expose semantic operations rather than CSS-selector sequences:

```ts
type Identity = { email: string; password: string; displayName: string };

await messageDesk.open();
await messageDesk.beginSignup();
await tinyIdp.submitIdentity(identity);
const code = await mailpit.latestCode(identity.email);
await tinyIdp.submitEmailCode(code);
await tinyIdp.submitPassword(identity.password);
await messageDesk.expectSignedInAs(identity.email);
```

The fixture layer owns stable role-, label-, and URL-based selectors. Tests
must not depend on incidental DOM nesting. Generated identities use a unique
run identifier so retained volumes can be tested without collisions; tests
that require a collision deliberately reuse a known fixture address.

### 3. Assert presentation as a contract

For every failed action, assert all of the following:

- The response is HTML, not `text/plain`.
- The expected client stylesheet is loaded from the approved same-origin
  `/static/` route.
- A visible heading identifies the operation that failed.
- A field-level message names a correctable value when correction is possible.
- Focus or `aria-describedby` connects the message to its input.
- Non-secret values needed for correction remain populated; password and code
  fields never do.
- The page offers a valid retry, restart, return, or login action.
- The response contains no exception, database detail, account identifier,
  continuation secret, CSRF token outside hidden form authority, or email-code
  value.
- CSP produces no application-owned console error. Browser-extension console
  messages are filtered from the product assertion but retained in artifacts.

### 4. Test matrix

The initial matrix is intentionally pairwise rather than a brute-force
Cartesian product. Each row selects a state boundary with distinct behavior.

| Area | Starting state | Action | Required user-visible result |
|---|---|---|---|
| Open signup | No RP or IdP session | Register a unique valid address | Email-code page, password page, callback, signed-in app |
| Duplicate identity | Existing account, optional active IdP session | Complete signup with the same normalized email | Themed signup page identifies that the email is already registered and offers login |
| Email syntax | Signup identity page | Empty, malformed, or surrounding-whitespace email | Native browser or themed field error; no challenge is sent |
| Display name | Signup identity page | Empty or over-limit name | Specific field error; email value remains present |
| Email code | Pending challenge | Empty, wrong, expired, exhausted, or replayed code | Specific themed error with allowed resend/restart action |
| Password | Verified email | Too short, mismatched, or rejected password | Specific password field error; secrets are cleared |
| Invitation | Goja signup | Empty, unknown, expired, revoked, consumed, or wrong-audience code | Invite-code field error without revealing which policy check failed |
| Login | No IdP session | Unknown login or wrong password | Themed generic credential error; login retained, password cleared |
| Remembered account | Two remembered identities | `prompt=select_account`, select each identity | Correct identity reaches requesting RP |
| Add account | One current remembered identity | Start explicit signup | Signup begins without destroying the current identity; success switches current identity |
| RP-local logout | Both IdP and RP sessions | Log out of Message Desk only | Message Desk is anonymous; IdP and Goja sessions retain their defined state |
| Provider logout | Active IdP and RP sessions | Complete TinyIDP logout | Provider cookie is retired; RP behavior follows its independent session contract |
| Account removal | Multiple remembered identities | Remove a chooser entry | Entry disappears; another valid identity remains selectable |
| Stale form | Open signup form | Restart flow, then submit old tab | Themed terminal stale-request page with restart guidance |
| Replay | Successfully consumed form | Submit it again | Themed terminal completed-request page; no duplicate side effect |
| Cross-client theme | Equivalent failure in both RPs | Trigger recoverable and terminal errors | Message Desk and Goja Auth use their own approved CSS and policy copy |
| OAuth callback | Provider returns a valid OAuth error | Follow redirect to each RP | RP renders a useful themed callback page instead of raw text |

### 5. Browser contexts and cleanup

Use a fresh Playwright browser context for isolated tests. Use a single
context only for tests whose purpose is cookie/session evolution. Save
`storageState` at named boundaries when debugging, not as a committed source
of test authority.

Each failing test retains:

- a screenshot of the final page;
- Playwright trace with DOM snapshots and network events;
- console and page-error log;
- response status and content type for navigation requests;
- the relevant redacted TinyIDP audit tail; and
- a compact state label such as `idp=current:A,remembered:[A,B];message=none`.

Artifacts go under a gitignored test-results directory. The ticket diary and
defect ledger record stable conclusions, not cookies, passwords, invite codes,
email codes, or raw traces containing secrets.

### 6. Defect classification

Failures are recorded in this document's ledger before repair:

| ID | Journey | Observed failure | Layer | Status |
|---|---|---|---|---|
| UX-001 | Consumed or unavailable signup continuation | Former `400 text/plain`: `registration request was not accepted`; now a client-themed terminal restart page (`73b0c0d`, Chromium replay `10190ba`) | Continuation binding / provider presentation | Resolved |
| UX-002 | Duplicate-email commit | Current generic field copy is `This value could not be accepted.` | Signup error taxonomy | Planned |
| UX-003 | RP OAuth callback error | Message Desk previously returned `400 text/plain`; it now renders a CSP-bound recovery page (`9c70f31`, Chromium `cb5d2ca`). Goja Auth still returns `401 text/plain: oidc error: access_denied` from its separate repository handler. | RP callback presentation | Message Desk resolved; Goja follow-up required |
| UX-004 | Email-code attempt exhaustion | SQLite rolled back each rejected verification counter, so the live attempt limit was never reached. A resend also rotated a code without restoring an exhausted attempt budget. Both transitions are now durable: rejected attempts commit, a permitted resend resets the new generation's counter, and Chromium verifies old-code rejection plus replacement-code success (`a41087c`, `263603a`). | SQLite email challenge transition / recovery presentation | Resolved |
| UX-005 | New-account consent approval | The post-signup consent page discarded the canonical authorization request while constructing its presentation model. Its CSP therefore contained only `form-action 'self'`; Chromium applied that directive to the form's redirect chain and blocked the validated Message Desk callback with `net::ERR_ABORTED`. The page now derives the RP origin from the stored canonical request (`cfc1d08`), and the complete Chromium signup reaches Message Desk. | Signup-to-consent presentation / CSP | Resolved |
| UX-006 | Long sequential browser matrix | The default 30-request address window masked later cases with `429 text/plain: rate limited`. Authorization throttling now uses the terminal browser-error renderer, while the local exhaustive stack has a finite 500-request budget (`595742b`). | Provider browser-error presentation / local harness policy | Resolved |
| UX-007 | Password confirmation mismatch | Native commit rejected unequal secrets only after lambda invocation and rendered generic password-policy copy. Cross-field equality is now checked before JavaScript, resolved secret copies are cleared, and the confirmation field receives the closed mismatch error (`647d540`). | Native workflow submission validation | Resolved |
| UX-008 | Fresh Compose recreation | Health checks and backend TLS clients resolved `idp.localhost` to documented Caddy address `172.31.0.2`, but dynamic allocation could give that address to the first concurrently started service. Caddy now has an explicit `.2` endpoint and dynamic allocation is confined to `.128/25`. | Compose network address contract | Resolved |

New rows must include the first failing trace, expected behavior, owner layer,
fix commit, and passing test name.

### Final execution evidence

The single-worker Chromium suite contains 28 scenarios. The 2026-07-23
retained-state run passed with `28 passed (17.7s)`. It adds two direct browser
proofs that were absent from the earlier 25-scenario claim:

- **RP-local logout isolation:** after one browser context establishes both
  Message Desk and Goja Auth sessions through the same TinyIDP identity,
  `Log out of Message Desk` leaves Message Desk in guest mode while Goja's
  `/auth/session` endpoint and rendered dashboard remain authenticated.
- **Consumed invitation replay:** a real Goja signup consumes an
  operator-issued one-time invitation, then a second browser signup presents
  the exact same code. The second submission remains on the Goja-themed
  identity form, marks the invite field invalid, shows only the closed
  `This value could not be accepted.` copy, and never sends an email code.

The fresh-state run was also green on 2026-07-23: `28 passed (16.2s)`. It used
`docker compose down -v`, which destroyed only Compose-project application
volumes. The explicitly external `tinyidp-local-caddy-pki` CA volume remained
protected and was reused by the subsequent stack startup. The post-reset smoke
check proved readiness at all three public origins and the two OAuth login
redirects before Chromium began. Retained-state success alone is not evidence
that bootstrap, operator fixtures, or fresh database behavior still work; both
execution modes are required whenever the suite or local topology changes.

The fresh run also proved that the one-shot CA export and goja administrator
bootstrap exited successfully, TinyIDP and Message Desk became healthy, and
Caddy owned `172.31.0.2` while dynamic IDP-backend services received addresses
from the reserved `.128/25` pool.

Invitation coverage is layered intentionally. Chromium exercises missing,
unknown, expired, wrong-audience, and consumed codes as indistinguishable
themed field errors; it also proves a valid code starts and completes a real
signup. The `pkg/idpinvite` provider matrix exhausts revoked and remaining
internal state classifications without exposing those distinctions in browser
copy. Revocation is not a normal browser-created state because issuing and
revoking a bearer capability are explicit operator actions, but its externally
visible error class remains the same closed invite-field response.

## Design Decisions

- Use Playwright rather than extending the standard-library acceptance client.
  The existing script verifies HTTP flows efficiently, but it cannot assert
  layout, focus, accessibility relationships, loaded CSS, browser validation,
  history behavior, or console policy errors.
- Keep the existing HTTP acceptance script. It remains a fast protocol smoke
  test and helps distinguish browser rendering failures from backend failures.
- Run against real TLS. Cookie security, origin classification, Fetch Metadata,
  callback URLs, CSP, and stylesheet loading behave differently over an HTTP
  shortcut.
- Prefer semantic fixtures. Page objects express TinyIDP operations and keep
  selectors centralized, while assertions remain in tests so the intended
  behavior is visible at the call site.
- Preserve negative security semantics. A UX fix changes presentation and
  recovery, not whether CSRF, origin, continuation, replay, audience, or
  credential checks accept a request.
- Use closed public error codes. Go and JavaScript may classify failures, but
  templates map approved codes to fixed public text. Backend error strings do
  not cross into HTML.
- Do not use visual snapshots as the primary oracle. Structural and accessible
  assertions define correctness; screenshots aid diagnosis and permit a small
  number of intentional layout checks.

## Alternatives Considered

- Selenium was rejected because this repository has no existing Selenium
  infrastructure and Playwright provides stronger tracing, browser contexts,
  auto-waiting, and network inspection for this test shape.
- A synthetic in-process browser was rejected because it would bypass Caddy,
  trusted-proxy handling, secure cookies, and application callback behavior.
- Testing every possible state combination was rejected as expensive and
  redundant. Pairwise boundary scenarios plus focused unit tests give better
  diagnostic value.
- Rendering every error as one terminal page was rejected because recoverable
  validation failures should preserve the live workflow and field values.
- Returning exact storage errors was rejected because account enumeration and
  implementation details must not become public. Copy may say an email is
  already registered only in the explicit signup context, where the submitted
  address is already known to the browser.

## Implementation Plan

### Phase 1 — Harness and observability

- Add a pnpm Playwright project without creating a second Go module.
- Configure the local CA for Chromium, Firefox, and WebKit where supported;
  do not disable TLS verification globally.
- Add health preconditions for all three HTTPS origins and Mailpit.
- Implement fixtures for unique identities, email-code lookup, invitations,
  RP pages, TinyIDP pages, response capture, and redacted diagnostics.
- Add one happy-path Message Desk signup as the harness proof.

Exit criterion: a clean-stack run completes signup and produces a trace only
on failure.

### Phase 2 — Validation and error taxonomy

- Add table-driven browser tests for identity, email challenge, password,
  invitation, and duplicate-account errors.
- Add closed field-error codes where the current `rejected` code is too vague.
- Ensure recoverable failures re-render the same live continuation and do not
  consume or advance it.
- Ensure terminal continuation failures use a themed browser-error page with
  restart guidance.

Exit criterion: every ordinary invalid input produces a precise accessible
field error; stale/replayed authority produces a precise terminal page.

### Phase 3 — Sessions and multi-account navigation

- Exercise first login, automatic current-session reuse, forced account
  chooser, adding a second identity, switching, removing, RP-local logout, and
  provider logout.
- Assert cookie effects indirectly through the visible state of all three
  origins.
- Cover abandoned signup and confirm it does not destroy the prior identity.

Exit criterion: the state-transition table is executable and no journey
depends on clearing all browser storage manually.

### Phase 4 — Cross-client policy and theming

- Repeat representative success, recoverable-error, and terminal-error paths
  for Message Desk and Goja Auth.
- Verify open signup versus invitation-required signup.
- Verify each response loads only its registered theme stylesheet.
- Verify application callback errors are styled and actionable.

Exit criterion: both clients preserve independent policy and presentation
while sharing the same provider.

### Phase 5 — Stability and operational use

- Run once after a full reset and once against retained volumes.
- Add bounded retries only for asynchronous email delivery and container
  readiness; never retry assertions that may hide a race.
- Document commands for headed debugging, one-test execution, trace viewing,
  and safe fixture cleanup.
- Attach the final matrix, defect resolution table, and evidence summary to
  the ticket diary.

Exit criterion: the suite is deterministic enough to run before local or k3s
promotion and every ledger item is closed or explicitly deferred.

### Suggested file layout

```text
examples/tinyidp-shared-two-apps/browser-tests/
├── package.json
├── playwright.config.ts
├── fixtures/
│   ├── stack.ts
│   ├── identity.ts
│   ├── mailpit.ts
│   └── diagnostics.ts
├── pages/
│   ├── message-desk.ts
│   ├── goja-auth.ts
│   └── tiny-idp.ts
└── tests/
    ├── signup-validation.spec.ts
    ├── login.spec.ts
    ├── remembered-accounts.spec.ts
    ├── logout.spec.ts
    ├── stale-and-replay.spec.ts
    └── cross-client-theme.spec.ts
```

### Review pseudocode

```text
for each scenario:
    establish named browser/application/identity state
    perform exactly one boundary action
    assert HTTP classification and visible page semantics
    assert sensitive fields were not replayed
    assert next action is possible
    assert audit event classification
    on failure:
        retain trace, screenshot, console, and redacted audit tail
        add or update defect-ledger row
```

## Open Questions

- Firefox trusts the CA in the user's profile, but automated Playwright browser
  binaries may use separate trust behavior. The harness must prove a scoped CA
  configuration before choosing any browser-specific fallback.
- Duplicate-email disclosure policy should be explicit. This guide recommends
  direct copy during an explicit signup attempt while retaining generic login
  errors; security review should confirm that boundary.
- Provider logout and RP logout are intentionally separate. Product copy and
  test names must state which session is affected rather than promising a
  global logout that the protocol does not perform.
- Email challenge expiry tests need a supported clock or short-lived test
  policy. Production sleeps are not acceptable in the suite.

## References

- `examples/tinyidp-shared-two-apps/compose.yaml`: local production-shaped
  service and TLS topology.
- `examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py`: current
  fast HTTP-level acceptance coverage.
- `examples/tinyidp-shared-two-apps/open-signup.js`: open and invitation-based
  signup workflow definitions.
- `internal/fositeadapter/provider.go`: authorization and session state
  transitions.
- `internal/fositeadapter/scripted_signup.go`: continuation loading, workflow
  invocation, and native signup commit boundary.
- `pkg/idpcontinuation/service.go`: continuation binding and replay semantics.
- `pkg/idpui/workflow.go`: closed field-error presentation contract.
- `internal/productionui/templates/workflow.html`: accessible themed workflow
  template.
- `design-doc/02-themed-registration-rejection-page-implementation-guide.md`:
  terminal browser-error model and renderer boundary.
