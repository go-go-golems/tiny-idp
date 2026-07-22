---
Title: Implementation diary
Ticket: TINYIDP-LOCAL-COMPOSE-001
Status: active
Topics:
    - oidc
    - tiny-idp
    - kubernetes
    - local-development
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://Makefile
      Note: Defines fast, Fosite-only, two-process, and full test gates (commit a99b0ed)
    - Path: repo://README.md
      Note: Documents the explicit test-level contract (commit a99b0ed)
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Safe non-reflective Message Desk callback recovery page (commit 9c70f31)
    - Path: repo://examples/tinyidp-message-app/app_http_test.go
      Note: Callback recovery page contract (commit 9c70f31)
    - Path: repo://examples/tinyidp-message-app/ui/src/App.tsx
      Note: Separated account action navigation (commit 4b15802)
    - Path: repo://examples/tinyidp-shared-two-apps/README.md
      Note: Fresh-reset topology contract (commit 69f3283)
    - Path: repo://examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts
      Note: |-
        Playwright journeys (commit 34959ea)
        Playwright duplicate-email regression (commit 21456f9)
        Themed unknown-login and wrong-password browser coverage (commit 882790a)
        Goja Auth invalid-credential and client-theme browser coverage (commit 98821fa)
        Native display-name and password browser validation coverage (commit f5f9eaf)
        Email-code resend browser recovery coverage (commit 137ebd3)
        Browser provider-logout coverage (commit 9d25a40)
        Two-account Chromium switch regression (commit fadfc08)
        Two-account switching and removal regression (commits fadfc08 and 492a659)
        Email-limit and Goja invitation browser coverage (commits cd93fec and 2403443)
        Message Desk callback browser evidence (commit cb5d2ca)
        Same-origin replayed-form Chromium evidence (commit 10190ba)
        Live Goja callback recovery regression (uncommitted matrix work)
        Live Goja callback recovery regression (commit a62d319)
        Chromium recovery journey through the trusted local stack
        Complete Chromium new-account journey
        Password and invitation browser matrix (commit 647d540)
    - Path: repo://examples/tinyidp-shared-two-apps/compose.yaml
      Note: |-
        Local shared IdP enables reviewed chooser policy (commit d940253)
        Configures Goja local callback stylesheet (commit 9c37b66)
        Finite local exhaustive-test rate budget (commit 595742b)
        Deterministic proxy address and dynamic pool (commit 69f3283)
    - Path: repo://examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py
      Note: Live HTTPS rejection validation (commit 0ce1fa6)
    - Path: repo://internal/cmds/serve_production.go
      Note: Opt-in production account chooser host configuration (commit d940253)
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Null-origin guard and registration error routing (commits 924b575 and dffc6c4)
        Authorization rate-limit call sites (commit 595742b)
    - Path: repo://internal/fositeadapter/registration_test.go
      Note: |-
        Active-session second-signup regression coverage (commit 1a15439)
        Provider regression (commit c7a2cb7)
        Provider replay terminal-page regression (commit 73b0c0d)
        Provider regression for callback-aware consent CSP and approval redirect
    - Path: repo://internal/fositeadapter/rendering.go
      Note: |-
        Terminal browser error response boundary (commit dffc6c4)
        Provider-owned browser throttling presentation (commit 595742b)
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: |-
        Remembered-session continuation fix (commit c7a2cb7)
        Duplicate commit mapping (commit 21456f9)
        Native email-code verification and closed error mapping (commit cd93fec)
        Themed terminal handling for unavailable signup continuations (commit 73b0c0d)
        Preserves the validated RP origin across the signup-to-consent handoff
        Native password mismatch boundary (commit 647d540)
    - Path: repo://internal/productionui/renderer.go
      Note: Per-client error theme selection (commit dffc6c4)
    - Path: repo://lefthook.yml
      Note: |-
        Fast pre-commit and full pre-push validation policy (commit bd4c424)
        Runs the complete suite only before pushes (commit a99b0ed)
    - Path: repo://pkg/idpemailchallenge/memory.go
      Note: Keeps development/test challenge semantics aligned with SQLite
    - Path: repo://pkg/idpemailchallenge/service_test.go
      Note: Deterministic exhaustion resend and replacement-code contract
    - Path: repo://pkg/idpui/browser_error.go
      Note: Bounded terminal error model (commit dffc6c4)
    - Path: repo://pkg/idpui/workflow.go
      Note: Global duplicate-identity error model (commit 21456f9)
    - Path: repo://pkg/idpworkflow/descriptors.go
      Note: |-
        Production-aligned public password bounds (commit 2c136ee)
        Non-redisplayable one-time email verification code contract (commit bd4c424)
    - Path: repo://pkg/idpworkflow/submission.go
      Note: |-
        Native secret-field bounds enforcement (commit 2c136ee)
        Request-scoped non-redisplayable verification values (commit cd93fec)
    - Path: repo://pkg/sqlitestore/email_challenge.go
      Note: Durably commits rejected verification attempts and resets only a replacement code generation
    - Path: ws://go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Safe Goja OIDC callback recovery renderer (commit f8ff1af)
    - Path: ws://go-go-goja/pkg/xgoja/hostauth/builder.go
      Note: Wires the validated application stylesheet path to the native handler (commit f8ff1af)
ExternalSources: []
Summary: Chronological implementation and debugging record for the shared TinyIDP local HTTPS Compose stack.
LastUpdated: 2026-07-21T13:18:51.810017936-04:00
WhatFor: Review what changed, why the local trust boundary is shaped this way, which failures occurred, and how to validate the result.
WhenToUse: Read before resuming or reviewing TINYIDP-LOCAL-COMPOSE-001.
---

























# Implementation diary

## Goal

Build and explain a repeatable local HTTPS environment for one strict TinyIDP
serving Message Desk and goja auth-host, while preserving the cluster's issuer,
proxy, client-registration, theme, and certificate-validation contracts.

## Step 1: Establish the ticket and production-shaped target

The existing one-app HTTP Compose demo was useful evidence but insufficient as
a deployment rehearsal. This step defined the local environment as a strict
HTTPS proxy topology with two relying parties, rather than expanding the old
demo until its development assumptions became ambiguous.

### Prompt Context

**User prompt (verbatim):**

> Create a design document for this task, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
>
>
> then work on it and get it up and running

**Assistant interpretation:** Create a ticketed intern guide, implement the
local multi-app Compose topology, run it, and publish the documentation.

**Inferred user intent:** Shorten the feedback loop for TinyIDP changes while
testing the same integration contracts that otherwise fail only during a live
k3s rollout.

**Commit (code):** `988049a` — "feat: add shared TinyIDP local Compose scaffold"

### What I did

- Compared the older one-app Compose example with the current production
  GitOps manifests and goja auth-host image.
- Created `TINYIDP-LOCAL-COMPOSE-001` and five phase-level tasks.
- Added the initial Caddy, TinyIDP, Message Desk, PostgreSQL, and goja services.
- Added local client/theme catalogs and open-signup program.

### Why

- The local stack must detect issuer, redirect, secure-cookie, proxy, and theme
  errors before a GitOps deployment.

### What worked

- `docker compose config` accepted the initial topology.
- TinyIDP built and became healthy behind Caddy.
- Its discovery document advertised the exact issuer
  `https://idp.localhost:8443`.

### What didn't work

- The first two direct `docker compose up --build -d` commands outlived the
  command-output window and left overlapping build processes. Moving the
  long-running launch into tmux made lifecycle and output observable.
- Message Desk and goja exited during discovery with:

  `tls: failed to verify certificate: x509: certificate signed by unknown authority`

### What I learned

- The cluster's TLS topology can be represented locally without Kubernetes,
  but certificate trust is a first-class application dependency.

### What was tricky to build

- Issuer identity had to include the host port `8443` everywhere, including
  internal backchannel discovery. Using Caddy's internal port 443 would have
  produced issuer mismatch during discovery.

### What warrants a second pair of eyes

- Review every issuer, callback, and post-logout URL as an exact string.

### What should be done in the future

- Keep the older loopback HTTP demo as a teaching profile; do not silently
  change it into this stricter acceptance profile.

### Code review instructions

- Start with `compose.yaml`, `clients.json`, and `themes.json`.
- Run `docker compose config` before reviewing runtime behavior.

### Technical details

- Public origins use `idp.localhost`, `message.localhost`, and
  `goja.localhost`, all on host port 8443.

## Step 2: Research the certificate failure at the trust boundary

The failure initially looked like Go ignoring `SSL_CERT_FILE`. Direct evidence
showed that the variable was present but its target file was unreadable to the
actual non-root processes. This distinction determined the design: publish a
readable public trust anchor rather than weakening TLS or changing application
code.

### Prompt Context

**User prompt (verbatim):** "Ok, let's take a step back and figure out what to do about that, because that will help us for all idp local setups in the future. Feel free to reserach online as well, and savinginteresting resourcesin the sources/ folder of the ticket using defuddle."

**Assistant interpretation:** Pause speculative fixes, establish a reusable
local PKI pattern from primary sources and runtime evidence, and save the
research in the ticket.

**Inferred user intent:** Solve local IdP certificate trust once in a way future
Compose environments can copy safely.

### What I did

- Captured Caddy, mkcert, Compose startup-order, and Go x509 primary sources
  with Defuddle under `sources/`.
- Inspected the actual Caddy volume and container UIDs.
- Compared Caddy public-root export with mkcert-generated host certificates.

### Why

- A local CA is security-sensitive. The solution must be based on actual file
  ownership, runtime behavior, and supported trust APIs.

### What worked

- Runtime inspection produced decisive evidence:

  `-rw------- 1 0 0 631 ... /data/caddy/pki/authorities/local/root.crt`

- The goja container reported UID `65532`; Message Desk drops to UID `10001`.
- Go's `root_unix.go` confirmed that `SSL_CERT_FILE` is supported and read with
  `os.ReadFile`.

### What didn't work

- Mounting Caddy's data volume directly into applications did not make the
  owner-only root readable.
- That mount also exposed a directory containing `root.key` and
  `intermediate.key`, which was too broad even if permissions were changed.

### What I learned

- The public root certificate is distributable trust material; the CA private
  key is signing authority. They require different volumes and principals.
- Caddy cannot install its trust root into a host browser from inside a
  container, and should be configured with `skip_install_trust`.

### What was tricky to build

- `root.crt` appears public by name but inherits owner-only protection because
  it lives beside CA private keys. Treating the entire storage directory as a
  trust bundle confuses two different security roles.

### What warrants a second pair of eyes

- Confirm that no application volume mount exposes `caddy-data` after the fix.

### What should be done in the future

- If many unrelated local stacks need one workstation-wide CA, evaluate a
  documented mkcert prerequisite. Never commit `rootCA-key.pem`.

### Code review instructions

- Read the four captured sources, then inspect `ca-export` and volume mounts in
  `compose.yaml`.

### Technical details

- Caddy documents its local PKI under `pki/authorities/local`.
- Compose documents `service_completed_successfully` for one-shot dependency
  gates.

## Step 3: Implement public-root publication and deterministic routing

This step converted the research into two narrow contracts. `ca-export` waits
for Caddy, copies only the public root to a separate volume, and exits zero.
Network-specific aliases ensure Caddy reaches each backend through the subnet
that backend explicitly trusts.

### Prompt Context

**User prompt (verbatim):** (see Step 2)

**Assistant interpretation:** Apply the researched general solution and get the
stack running.

**Inferred user intent:** Produce a usable implementation, not only a design
recommendation.

**Commit (code):** `567a1eb` — "fix: distribute local Caddy trust safely"

### What I did

- Added the successful one-shot `ca-export` service and `local-ca` volume.
- Set the exported public root to mode `0444`.
- Mounted only `local-ca` read-only into TLS clients.
- Added explicit browser-root export and acceptance-smoke scripts.
- Added network-specific Caddy aliases.
- Documented startup, trust, reset, logs, and local goja builds.

### Why

- Applications need a readable root certificate, not access to CA storage.
- Narrow trusted CIDRs are useful only if proxy traffic deterministically uses
  the matching network.

### What worked

- `ca-export` completed with exit code 0.
- Message Desk and goja completed verified HTTPS discovery and stayed running.
- All three public readiness endpoints returned 200 with `curl --cacert`.
- Both application login endpoints redirected to TinyIDP with the correct
  client ID, exact callback, nonce, state, and S256 PKCE challenge.

### What didn't work

- The first Message Desk request returned `400 untrusted proxy peer` even
  though `172.32.0.0/24` was trusted. Caddy and Message Desk shared two
  networks; the bare service name resolved through `172.31.0.0/24`.
- A shell command used zsh's read-only variable name `status` and failed with
  `zsh:1: read-only variable: status`. The retry used `response_code`.
- The first external goja readiness check raced container startup and returned
  502. The smoke script now polls for up to 30 seconds.

### What I learned

- Compose service discovery across multiple shared networks is not an adequate
  routing policy. A per-network alias makes the intended source/destination
  interface explicit.
- A distroless image should not be enlarged just to host a Docker healthcheck.
  An external verified readiness probe better exercises the user-visible path.

### What was tricky to build

- There are two independent trust decisions: the relying party verifies the
  IdP's TLS certificate, while each backend verifies the immediate reverse
  proxy and forwarded origin. Passing one does not imply passing the other.

### What warrants a second pair of eyes

- Review the fixed subnet allocation for conflicts with common VPN or Docker
  networks.
- Verify browser CA removal instructions on non-Debian platforms before adding
  platform-specific automation.

### What should be done in the future

- Add a browser-level signup/login acceptance test if this local topology
  becomes a required CI gate.

### Code review instructions

- Run:

  `docker compose -f examples/tinyidp-shared-two-apps/compose.yaml up --build -d`

- Then run both scripts under `examples/tinyidp-shared-two-apps/scripts/`.
- Confirm no request uses `-k` in the acceptance script and both Location
  headers carry the expected client IDs.

### Technical details

- Successful acceptance output:

  ```text
  OK https://idp.localhost:8443/readyz
  OK https://message.localhost:8443/readyz
  OK https://goja.localhost:8443/auth/readyz
  OK login redirect tinyidp-message-app
  OK login redirect goja-auth-host-demo
  ```

## Step 4: Diagnose Firefox registration rejection and preserve the CSRF boundary

The browser supplied decisive evidence that the rejected form was a genuine
same-origin navigation whose `Origin` header had been serialized as `null`.
TinyIDP itself caused that representation by serving the form with
`Referrer-Policy: no-referrer`. This step corrected the guard without treating
every opaque origin as trustworthy.

### Prompt Context

**User prompt (verbatim):** "Add new concise implementation guide to handle the error page and add tasks to the ticket. and then implement it."

**Assistant interpretation:** Document and implement a themed, safe HTML page
for the registration rejection that was exposed while debugging Firefox.

**Inferred user intent:** Make security failures understandable and visually
consistent with each relying application without weakening the registration
guard or leaking submitted account data.

**Commit (code):** `924b575` — "fix: accept verified null-origin registration posts"

### What I did

- Compared the captured Firefox headers with `sameOriginBrowserPost` and the
  provider's `Referrer-Policy` header.
- Added a narrow `Origin: null` acceptance branch requiring `same-origin`,
  `navigate`, `document`, and `?1` Fetch Metadata.
- Added positive and adversarial tests for explicit and null origins.
- Rebuilt the local IDP and confirmed every readiness check passed.

### Why

- A basic form POST under `no-referrer` can legitimately carry `Origin: null`.
  Rejecting it made normal Firefox signup impossible.
- A sandboxed hostile document can also have an opaque origin, so `null` alone
  is never evidence of same-origin provenance.

### What worked

- `go test ./internal/fositeadapter -count=1` passed.
- The pre-commit full suite and both lint analyzers passed.
- The local IDP was healthy after the image rebuild.

### What didn't work

- The first focused test used `Origin: http://idp.localhost:8080` with request
  host `idp.localhost:8443` and correctly failed with:

  `sameOriginBrowserPost() = false, want true`

  The fixture was corrected to use the same port.
- The first manual readiness probe referenced the wrong CA path and failed:

  `curl: (77) error setting certificate file: examples/tinyidp-shared-two-apps/.local-ca/caddy-local-root.crt`

  The documented file is
  `examples/tinyidp-shared-two-apps/runtime/caddy-local-root.crt`; retrying with
  it returned the complete ready response.

### What I learned

- `Origin: null` is not synonymous with cross-site. It is an opaque
  serialization that must be interpreted with independently supplied browser
  context.
- Fetch Metadata permits a substantially narrower exception than removing the
  origin guard or accepting all null-origin posts.

### What was tricky to build

- The same header value can result from a legitimate privacy policy or a
  hostile opaque-origin document. The solution therefore required all four
  browser-controlled navigation properties and retained the cryptographic CSRF
  and durable interaction checks.

### What warrants a second pair of eyes

- Review the Fetch Metadata conjunction in `sameOriginBrowserPost`; weakening
  any member expands which opaque-origin requests can reach account creation.

### What should be done in the future

- Keep a real-browser test for Firefox's `no-referrer` form behavior in the
  broader browser automation layer when one is introduced.

### Code review instructions

- Start with `sameOriginBrowserPost` and `browser_post_test.go`.
- Run `go test ./internal/fositeadapter -run TestSameOriginBrowserPost -count=1`.

### Technical details

- Accepted null-origin shape: `same-origin` + `navigate` + `document` + `?1`.
- The exact-origin compatibility path continues to reject contradictory
  `Sec-Fetch-Site: cross-site` metadata.

## Step 5: Add a terminal client-themed registration error page

The origin rejection still used `http.Error`, so a correct security decision
produced an unstyled text document. This step introduced a terminal browser
error presentation contract that reuses approved per-client CSS while carrying
none of the authority or sensitive values present in an interaction or signup
workflow page.

### Prompt Context

**User prompt (verbatim):** "continue keeping a diary and committing at a ppropriate intervals (see skill for diary format in case you forgt)."

**Assistant interpretation:** Continue through implementation and deployment,
recording the work in the ticket's established detailed diary format and using
focused commits.

**Inferred user intent:** Leave a reviewable technical history alongside the
working UI change, rather than only an unexplained patch.

**Commit (code):** `dffc6c4` — "feat: render themed registration rejection pages"

**Commit (acceptance):** `0ce1fa6` — "test: cover themed registration rejection in local stack"

**Commit (acceptance hardening):** `c26aa26` — "test: assert rejection response security headers"

### What I did

- Added the concise implementation guide and five concrete ticket tasks in
  commit `2991296`.
- Added `BrowserErrorPage` validation and `BrowserErrorRenderer` under
  `pkg/idpui`.
- Implemented default and production templates using `html/template`.
- Made `productionui.Renderer` select the stylesheet through
  `Catalog.Resolve(page.ClientID)`.
- Added a bounded, no-store provider rendering boundary with a plain-text
  last-resort fallback and renderer-failure audit event.
- Replaced only the registration-origin rejection response; Fosite OAuth error
  redirects and unrelated `http.Error` paths remain unchanged.
- Added model, escaping, theme-selection, provider-integration, and live HTTPS
  acceptance coverage.

### Why

- Reusing `InteractionPage` would require a fake live form, CSRF token, action,
  and continuation. A terminal page needs none of those capabilities.
- A separate bounded model makes it mechanically difficult to leak the posted
  display name, email, password, invite code, or continuation.

### What worked

- Focused tests passed for `pkg/idpui`, `internal/productionui`, and
  `internal/fositeadapter`.
- The full pre-commit lint and `go test ./...` suite passed.
- After rebuilding the IDP, the HTTPS acceptance probe received `403`,
  `text/html`, and `/static/themes/message-desk.css`.
- The probe verified that submitted identity data, `<form`, and `csrf_token`
  were absent.
- The hardened probe also verified `Cache-Control: no-store` and the exact
  restrictive Content Security Policy over the live Caddy route.

### What didn't work

- N/A. The implementation and deployment probe passed on the first run.

### What I learned

- The client ID is sufficient public context for theme selection once the
  stored interaction and current client generation have been validated.
- Terminal error presentation deserves a different model from resumable
  workflow errors; their security capabilities are different.

### What was tricky to build

- Renderer failures must not recursively render another error page. The
  boundary buffers once, audits the failure, and falls back directly to a
  minimal plain-text response with the original failure status.
- A callback URI is not necessarily an application landing page. The initial
  error model deliberately contains no return URL rather than inventing a new
  redirect policy.

### What warrants a second pair of eyes

- Review the terminal model for accidental future addition of URLs, HTML, or
  continuation fields.
- Confirm that the fixed public copy is suitable for all configured clients;
  only presentation, not error semantics, is client-specific.

### What should be done in the future

- Classify other browser-owned direct error responses individually before
  migrating them. Do not mechanically replace OAuth protocol error handling.

### Code review instructions

- Read the implementation guide first.
- Review `pkg/idpui/browser_error.go`, then
  `internal/fositeadapter/rendering.go`, then the production template.
- Run `go test ./pkg/idpui ./internal/productionui ./internal/fositeadapter -count=1`.
- Rebuild the `idp` Compose service and invoke
  `reject_cross_site_registration_with_themed_page` from the browser acceptance
  script.

### Technical details

- Public status remains `403 Forbidden`.
- Headers include `Cache-Control: no-store`, `Pragma: no-cache`, the existing
  strict CSP, and `Content-Type: text/html; charset=utf-8`.
- The last-resort response is still plain text by design because renderer
  failure cannot safely recurse.

## Step 6: Separate Message Desk account actions

The screenshot confirmed that the two account actions were adjacent inline
anchors. Their source formatting was not a layout contract, so the browser
wrapped both labels as one continuous run of linked text.

### Prompt Context

**User prompt (verbatim):** "were you able to figure out the html issues with the mangled links : [Image #1] at https://message.localhost:8443/"

**Assistant interpretation:** Diagnose and correct the visibly merged signup
and login links in the deployed Message Desk welcome panel.

**Inferred user intent:** Present account creation and existing-account login
as two clearly distinct, accessible actions.

**Commit (code):** `4b15802` — "fix: separate Message Desk account actions"

### What I did

- Wrapped the anchors in a navigation region labelled `Account actions`.
- Used the already-loaded Bootstrap flex utilities to create a vertical,
  start-aligned action list with an explicit gap.
- Rebuilt the embedded Vite assets and the Message Desk container.

### Why

- Adjacent inline anchors have no reliable visual separation. A semantic
  navigation region and explicit layout describe both meaning and geometry.

### What worked

- `pnpm run build` and `go test ./examples/tinyidp-message-app -count=1`
  passed.
- The live JavaScript bundle contains
  `d-flex flex-column align-items-start gap-2`, and Message Desk is healthy.

### What didn't work

- The first patch attempted to insert a custom rule into the one-line minified
  stylesheet and could not match a partial line. The implementation instead
  reused Bootstrap utilities already imported by the application.

### What I learned

- The issue was neither CSP nor React whitespace handling; it was normal inline
  anchor layout with no intervening layout container.

### What was tricky to build

- Source CSS and generated CSS are deliberately compact. Reusing checked-in
  Bootstrap utilities avoided editing generated presentation text by hand.

### What warrants a second pair of eyes

- Confirm the two actions remain visually distinct at narrow viewport widths.

### What should be done in the future

- N/A

### Code review instructions

- Review the `Welcome` component in `ui/src/App.tsx`.
- Run `pnpm run build` from the UI directory and load the local HTTPS page.

### Technical details

- The navigation uses `flex-column`, `align-items-start`, and `gap-2`; it does
  not depend on JSX whitespace or `<br>` elements.

## Step 7: Expose and enforce signup password requirements

The password descriptor already carried length bounds, but the workflow HTML
omitted `minlength` and the secret-submission branch checked only the maximum.
This created three inconsistent layers: browser presentation, native workflow
validation, and the 15-character production account policy.

### Prompt Context

**User prompt (verbatim):** "also, when filling the passwor page with a too short password for example, it would be good to have a proper error message (or validation upfront in the HTML / JS), because we just get a \"password could not be accepted\" kind of error (don't remember the exact phrasing)"

**Assistant interpretation:** Add proactive browser constraints and useful safe
feedback for short passwords while keeping server-side validation authoritative.

**Inferred user intent:** Let users correct ordinary signup mistakes without
guessing at an opaque policy rejection.

**Commit (code):** `2c136ee` — "fix: expose signup password requirements"

**Commit (acceptance):** `7ebecc3` — "test: verify live password guidance"

### What I did

- Aligned the built-in password and confirmation descriptors with TinyIDP's
  default production minimum of 15 characters.
- Added `minlength` to default and production workflow templates.
- Added visible `Use at least 15 characters.` guidance on secret fields.
- Enforced minimum and maximum descriptor lengths for secrets in the native
  submission parser.
- Replaced generic password-rejection text with safe actionable guidance.
- Added unit and live HTTPS acceptance checks, rebuilt TinyIDP, and completed a
  real email-verified signup through Message Desk.

### Why

- Browser validation shortens the feedback loop, but direct HTTP clients can
  bypass it. Native parsing and account policy must still reject invalid input.

### What worked

- Focused workflow, UI, production renderer, and provider tests passed.
- The full lint and `go test ./...` pre-commit suite passed.
- The live password page contains `minlength="15"` and visible guidance, and a
  valid signup completed successfully.

### What didn't work

- N/A

### What I learned

- Sensitive workflow fields followed a separate parser branch that enforced
  required and maximum length but accidentally omitted minimum length.

### What was tricky to build

- Client validation must mirror the public portion of server policy without
  replacing it. Blocklist and context-derived rejection remain server-only;
  the UI describes only the stable minimum and gives conservative guidance.

### What warrants a second pair of eyes

- The built-in descriptor and default production policy now both use 15.
  Review future configurable password policies to ensure their public workflow
  descriptors remain synchronized.

### What should be done in the future

- If deployments receive configurable minimum lengths, construct the workflow
  registry from the validated password policy rather than changing one default.

### Code review instructions

- Start with `pkg/idpworkflow/descriptors.go` and `submission.go`, then review
  both workflow templates and `WorkflowFieldError.Summary`.
- Run `go test ./pkg/idpworkflow ./pkg/idpui ./internal/productionui ./internal/fositeadapter -count=1`.

### Technical details

- The password remains a secret handle and is never redisplayed.
- HTML validation is advisory; native parsing and account acceptance remain
  authoritative.

## Step 8: Permit explicit signup from a remembered provider session

Message Desk local logout correctly revoked only the relying-party session.
TinyIDP still remembered the identity, but its authorization handler rejected
every explicit registration intent when any provider session existed. That
made the advertised multi-account workflow internally inconsistent.

### Prompt Context

**User prompt (verbatim):**

> https://message.localhost:8443/auth/callback?error=invalid_request&error_description=The+request+is+missing+a+required+parameter%2C+includes+an+invalid+parameter+value%2C+includes+a+parameter+more+than+once%2C+or+is+otherwise+malformed.+registration+requires+a+new+browser+session&state=c-Cub8zOZIkcGz9pJht0uSynae4A41daZ78fhR7yop8
>
> identity login was not accepted
>
> ---
>
> This was after logging out of message-desk (not idp) and clicking "sign up for a new account"

**Assistant interpretation:** Reproduce and fix signup after relying-party-only
logout while preserving TinyIDP's remembered-account behavior.

**Inferred user intent:** Let one browser create and switch between multiple
TinyIDP identities without requiring a confusing provider-wide logout first.

**Commit (code):** `1a15439` — "fix: allow signup from remembered provider sessions"

### What I did

- Traced the callback error to the explicit `registrationRequested &&
  hasSession` rejection in `beginAuthorize`.
- Removed that contradictory precondition while retaining interaction, CSRF,
  origin, PKCE, client-generation, rate-limit, and native commit validation.
- Documented the transition: signup is an explicit request to create and
  switch to a new identity.
- Added provider coverage proving a second registration page can begin with an
  active provider session.
- Added and ran the exact live regression journey: TinyIDP login, Message
  Desk-only logout, then new-account signup entry.

### Why

- Local RP logout is intentionally not global logout. Requiring a new IdP
  session made that supported distinction unusable for multi-account signup.
- Preemptively ending the old session would discard useful remembered-account
  context even if the user abandoned signup.

### What worked

- The focused registration test passed.
- The full lint and `go test ./...` pre-commit suite passed.
- The rebuilt local stack returned the signup identity form after the exact
  user-reported logout sequence.

### What didn't work

- N/A

### What I learned

- The relying app's `identity login was not accepted` text was only a generic
  projection of the upstream OAuth error. The actionable cause was entirely in
  TinyIDP's registration/session precondition.

### What was tricky to build

- The previous identity must remain current while signup is pending so an
  abandoned attempt does not log the browser out. Only a successful atomic
  signup commit emits the new current-session cookie; the old durable session
  remains eligible for the account chooser.

### What warrants a second pair of eyes

- Review the intended product language for “create and switch identity.” The
  security mechanics now support it, but the signup page may eventually state
  explicitly which remembered account is currently active.

### What should be done in the future

- Give Message Desk's OAuth callback a themed, safe error page for genuinely
  unrecoverable provider errors rather than its current generic plain text.

### Code review instructions

- Review the registration branch in `beginAuthorize` and the new second-signup
  assertion in `registration_test.go`.
- Run the `begin_signup_after_local_logout` acceptance function against the
  local Compose stack.

### Technical details

- Starting signup does not revoke the old provider session.
- Successful signup creates a new provider session and replaces the browser's
  current session cookie.
- Remembered-account state remains provider-owned and available to
  `prompt=select_account`.

## Step 9: Define the browser-state authentication UX matrix

The raw duplicate-email report showed that protocol tests alone were not a
sufficient acceptance boundary. I defined a browser-level matrix that makes
both state transitions and user presentation observable requirements across
TinyIDP, Message Desk, Goja Auth, Caddy, and Mailpit.

The guide records the browser, provider, remembered-identity, relying-party,
and workflow dimensions; expected outcomes; phased implementation work; test
artifact policy; and an initial defect ledger. It deliberately distinguishes
recoverable field mistakes from terminal stale or replayed authority.

### Prompt Context

**User prompt (verbatim):** "can you do playwright tests for all kinds of potential error cases and such to test the different paths. multiple account, sining out , using same email, bad password, bad username, etc... We want to provide a high quality navigation and UI experiences and not these error pages without anything. 

Crate a design / analysis / implementation guide to keep track of all thecombos and how to exercise them and then to keep track of the things to do and errors encountered."

**Assistant interpretation:** Define and begin implementing a real-browser
test program for authentication state, validation, error presentation, and
cross-application navigation.

**Inferred user intent:** Turn authentication UX quality into a repeatable
release gate instead of discovering raw error responses through manual use.

**Commit (code):** `3deca0d` — "docs: define authentication browser test matrix"

### What I did

- Added `design-doc/03-playwright-browser-state-and-authentication-ux-test-matrix.md`.
- Added seven ticket tasks covering the matrix, harness, validation, sessions,
  cross-client behavior, defect repair, and final retained-state validation.
- Recorded UX-001 through UX-003 in the initial defect ledger.

### Why

- Cookie and navigation state crosses process boundaries that isolated Go
  tests do not render.
- The expected result of a rejection includes HTML, theme, accessibility,
  safe copy, preserved non-secret values, and a valid next action.

### What worked

- Docmgr frontmatter validation passed.
- The guide maps each phase to concrete exit criteria and file locations.

### What didn't work

- N/A

### What I learned

- The relevant matrix is not simply inputs by endpoints. Current identity,
  remembered identities, RP sessions, workflow stage, and client policy each
  change the correct browser outcome.

### What was tricky to build

- The matrix must cover pairwise state boundaries without becoming an
  unmaintainable Cartesian product. Scenarios were selected where ownership or
  recovery semantics change.

### What warrants a second pair of eyes

- Review the duplicate-email disclosure policy. The guide recommends explicit
  copy only inside a signup attempt, while login remains non-enumerating.

### What should be done in the future

- Complete each phase and update the defect ledger with test names and fix
  commits.

### Code review instructions

- Read the state model and matrix before the implementation phases.
- Compare each browser test name to one or more matrix rows.

### Technical details

- Playwright artifacts are retained on failure and ignored by Git.
- Stable conclusions belong in the diary; traces must not be copied into the
  ticket because they can contain transient workflow authority.

## Step 10: Fix active-session continuation binding and add Playwright journeys

The audit event at `2026-07-21T23:03:35Z` classified the reported request as
`workflow.signup.resume_rejected` with `continuation_unavailable`. The failure
occurred before duplicate-account validation: a signup interaction correctly
cleared its active-session binding, but continuation loading incorrectly
reintroduced the cookies from the submitting request.

I changed continuation loading to preserve the interaction's binding contract
and added a provider regression that completes one signup, starts another in
the remembered browser, and submits its first form. I also added the initial
Playwright project and four browser journeys.

### Prompt Context

**User prompt (verbatim):** (see Step 9)

**Assistant interpretation:** Implement the first testable slice and use it to
diagnose the current raw error rather than merely documenting future work.

**Inferred user intent:** Make the reported journey work and establish a suite
that prevents equivalent UX regressions.

**Commit (code):** `c7a2cb7` — "fix: preserve unbound signup continuation sessions"

**Commit (code):** `34959ea` — "test: add authentication UX browser journeys"

### What I did

- Changed `signupLoadBindings` to use the validated interaction record's
  browser, session, and chooser bindings.
- Extended provider integration coverage through the first POST of a second
  signup in a browser with an active TinyIDP session.
- Added a pinned pnpm Playwright project with Chromium, single-worker stateful
  execution, retained failure traces, screenshots, video, and ignored output.
- Added journeys for Message Desk account-action layout, malformed email,
  remembered-session add-account submission, and duplicate email.

### Why

- A continuation must not strengthen an interaction's intentionally empty
  session binding by sampling unrelated current cookies.
- The real browser is required to test native form validation, CSS loading,
  layout, navigation, and browser error behavior.

### What worked

- The focused Go regression passed.
- Full `go test ./...`, golangci-lint, Glazed lint, and the UI analyzer passed
  through the pre-commit hook.
- Playwright discovered all four tests.
- The layout and malformed-email tests passed against the live HTTPS stack.

### What didn't work

- The first Playwright run could not launch: `Executable doesn't exist at
  /home/manuel/.cache/ms-playwright/chromium_headless_shell-1155/chrome-linux/headless_shell`.
  `pnpm exec playwright install chromium` installed the pinned revision, after
  which the two executed journeys passed.
- The first regression-test draft expected password mismatch copy, but the
  existing implementation rendered a generic display-name rejection. The
  assertion was narrowed to the invariant under repair: a live themed workflow
  response rather than continuation rejection. Password taxonomy remains a
  separate matrix defect.

### What I learned

- The form CSRF value and CSRF cookie in the browser capture were expected to
  differ: the form value is a MAC over the cookie nonce and interaction. The
  decisive evidence was the continuation audit event.
- Signup interactions clear `SessionIDHash`, while the former load binding
  sampled a non-empty active session cookie. `validateBindings` correctly
  rejected that mismatch.

### What was tricky to build

- Signup must remain independent of the active identity so abandoning signup
  does not destroy or replace it. The repair therefore preserves the empty
  session binding; it does not bind signup back to the old session or weaken
  the CSRF browser binding.

### What warrants a second pair of eyes

- Review that every future continuation constructor and loader uses the same
  authoritative binding source. Current request cookies are appropriate only
  when the interaction contract requires them.
- Review the test-only `ignoreHTTPSErrors` scope. Human browsers and backend
  clients still use the persistent exported CA; Playwright's disposable
  Chromium profile does not inherit that trust store.

### What should be done in the future

- Implement the closed duplicate-account and password error taxonomy.
- Complete email-code, invitation, session, logout, stale-form, replay, and
  cross-client journeys.

### Code review instructions

- Start with `signupLoadBindings` and the expanded
  `TestScriptedSignupDoesNotRequireLegacyRegistrationOption`.
- Run `go test ./internal/fositeadapter -run
  TestScriptedSignupDoesNotRequireLegacyRegistrationOption -count=1`.
- From `examples/tinyidp-shared-two-apps/browser-tests`, run `pnpm test` and
  inspect retained traces with `pnpm show-report`.

### Technical details

```text
interaction.SessionIDHash = empty (intentional add-account behavior)
old loader expectation    = hash(current tinyidp_session) (incorrect)
new loader expectation    = interaction.SessionIDHash (empty, authoritative)
browser CSRF binding      = interaction.BrowserBindingHash (still required)
```

## Step 11: Distinguish duplicate identity from password rejection

The reported password submission used a 19-character value and the browser
received a correctly themed retry page. The TinyIDP audit record established
that the password was not rejected: the native commit failed with
`duplicate_login` because the submitted email already belongs to an account.

The workflow had already advanced past its email page, so attaching that
failure to the visible password field would mislead the user. I added a small,
closed global workflow-error model for this specific case. It renders fixed
public copy and a return link to the already-validated relying-application
origin, without echoing an account record or exposing a backend error.

### Prompt Context

**User prompt (verbatim):** A password submission was reported as rejected;
the supplied password value is deliberately omitted from the diary because it
is credential material.

**Assistant interpretation:** Determine whether the password policy rejected
the submitted value and fix the misleading UX if another validation boundary
was responsible.

**Inferred user intent:** Signup feedback must identify the actual correction
the user can make rather than applying generic password copy to every failed
commit.

**Commit (code):** `21456f9` — "fix: explain duplicate signup identities"

### What I did

- Inspected the local IDP audit log for the exact request time.
- Confirmed `account.self_registration` was rejected with reason
  `duplicate_login`, not `password_rejected`.
- Added `WorkflowGlobalError` with the closed
  `duplicate_identity` public error code.
- Rendered the global alert in the default and themed production workflow
  templates, with an application-return link only when a validated redirect
  origin is available.
- Mapped `idpstore.ErrDuplicate` at the native signup commit boundary to that
  global error, and updated the Playwright duplicate-email assertion.

### Why

- The password fields are the only fields on the final signup page. A field
  error there would falsely say that a valid password was invalid.
- The email was explicitly submitted as part of signup, so direct
  duplicate-email guidance is appropriate in this context; login errors remain
  generic.

### What worked

- Focused `pkg/idpui`, `internal/productionui`, and
  `internal/fositeadapter` test packages passed.
- The full pre-commit suite, including `go test ./...`, lint, Glazed lint, and
  the UI analyzer, passed before the code commit.

### What didn't work

- The first global-error renderer test omitted a redirect origin, so it could
  not legitimately expect a return link. Adding the validated test origin made
  the intended conditional behavior explicit.

### What I learned

- The `password_rejected` UI copy was accurate for actual password policy
  failures, but this request never reached that branch.
- A workflow can discover a correctable error about data collected on an
  earlier page. Such an error needs page-level presentation rather than an
  invented field association.

### What was tricky to build

- Error copy must be both useful and bounded. The global model exposes only a
  closed provider-owned code and fixed text; neither store errors nor account
  metadata enter the template.

### What warrants a second pair of eyes

- Review the explicit-signup duplicate disclosure boundary and the wording of
  the return action for applications whose landing page does not offer login.

### What should be done in the future

- Persist and render the script-selected workflow title, so the password stage
  says `Choose a password` rather than the current provider default
  `Create an account`.
- Complete the remaining Playwright matrix scenarios.

### Code review instructions

- Review `WorkflowGlobalError`, its template use, and the
  `idpstore.ErrDuplicate` branch in `resumeScriptedSignup`.
- Run `go test ./pkg/idpui ./internal/productionui ./internal/fositeadapter -count=1`.
- Rebuild local `idp` and run `pnpm exec playwright test -g 'duplicate email'`
  from `examples/tinyidp-shared-two-apps/browser-tests`.

### Technical details

```text
password input --> native commit --> idpstore.ErrDuplicate
                                  --> audit: duplicate_login
                                  --> WorkflowGlobalError{duplicate_identity}
                                  --> themed HTML + validated application return link
```

## Step 12: Cover invalid-credential retries and classify the password mismatch defect

This step added browser coverage for the login boundary that must not disclose
whether an account exists. Both an unknown login and a known login with the
wrong password now retain the non-secret login field, clear the password field,
and render the relying application's themed generic error.

I also investigated the password-confirmation browser row without weakening its
security requirements. The local audit proved the request was rejected before
the password comparison branch. An experimental mismatch-specific error mapping
therefore had no effect and was deliberately removed rather than committed as
an unverified repair.

### Prompt Context

**User prompt (verbatim):** "Continue keeping a diary bruh"

**Assistant interpretation:** Continue implementation while recording focused
test evidence and failures in the ticket diary.

**Inferred user intent:** Make matrix progress auditable and avoid claiming
coverage where the observed browser behavior does not meet the documented UX
contract.

**Commit (code):** `882790a` — "test: cover themed invalid credential login"

### What I did

- Added two real-HTTPS Playwright cases to
  `authentication-ux.spec.ts`: unknown login and wrong password.
- Asserted the fixed public message `Invalid login or password.`, retained login
  value, cleared password value, and Message Desk stylesheet.
- Ran `pnpm exec playwright test -g 'retains the login name'`; both cases
  passed.
- Inspected the local durable audit with:

  ```sh
  docker compose -f examples/tinyidp-shared-two-apps/compose.yaml \
    exec -T idp sh -c 'tail -100 /state/audit/audit.jsonl'
  ```

- Observed the two password-mismatch attempts at `2026-07-22T00:25:29Z` and
  `2026-07-22T00:26:22Z` as
  `account.self_registration result=rejected reason=registration_rejected`.
- Removed the unproven mismatch-specific native change and its failing browser
  assertion. The matrix row remains open.

### Why

- Login must use one non-enumerating result for unknown accounts and bad
  credentials, while still giving a user enough information to retry safely.
- A generic precondition failure must not be relabeled as a password mismatch;
  that would make the matrix look green while leaving the actual defect hidden.

### What worked

- `pnpm exec playwright test -g 'retains the login name'` reported:

  ```text
  2 passed (2.3s)
  ```

- The two tests observed the themed Message Desk workflow and no product page
  errors.

### What didn't work

- The attempted password-confirmation journey did not reach the introduced
  `errSignupPasswordMismatch` branch. Its audit reason remained
  `registration_rejected`, which means one of the prior native effect checks
  rejected the request.
- The browser page attached that generic failure to the password field and
  showed password-policy wording. That is misleading and remains an explicit
  UX defect rather than validated coverage.

### What I learned

- The audit reason is the authoritative way to distinguish password-policy
  rejection, duplicate identity, state conflict, and a broader commit
  precondition at this boundary.
- The existing generic renderer picks the first visible field. On the password
  page that produces password-policy copy even when the cause is not the
  password policy.

### What was tricky to build

- The login tests run against a persisted local stack whose cookies and
  provider rate limits are meaningful state. Each test starts a fresh
  Playwright context, but service-side audit evidence is still needed to avoid
  mistaking a generic provider refusal for the intended validation branch.
- Password secrets cannot appear in the diary, audit assertion, or retained
  screenshots. The test asserts only post-submit field clearing and public
  copy.

### What warrants a second pair of eyes

- Review the native commit preconditions before changing error taxonomy. The
  repair must classify each safe rejection without exposing verified-email,
  secret-handle, rate-limit, or continuation details.
- Review whether the default generic workflow renderer should use a global
  recovery alert instead of assigning `ErrorRejected` to the first field.

### What should be done in the future

- Add a deterministic provider-level regression that identifies the exact
  mismatch precondition before implementing its user-facing classification.
- Continue the independent email-code, invitation, session, replay, and
  cross-client matrix rows.

### Code review instructions

- Start with the loop of credential cases in
  `examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts`.
- Run `pnpm exec playwright test -g 'retains the login name'` from
  `examples/tinyidp-shared-two-apps/browser-tests`.
- Compare the audit records only by event name and public reason category; do
  not copy credentials, cookies, codes, or continuation handles into review
  notes.

### Technical details

```text
unknown login / wrong password
  -> POST TinyIDP login workflow
  -> generic credential error (no account disclosure)
  -> login redisplayed; password cleared; Message Desk CSS loaded

password mismatch experiment
  -> native commit precondition
  -> audit reason registration_rejected
  -> generic first-field error (defect; row remains open)
```

## Step 13: Separate protocol completion from the Playwright navigation failure

The new-account browser journey reached the consent page and submitted its
approval action, but Playwright did not observe a response or navigation away
from `/authorize`. The TinyIDP audit nevertheless recorded a successful
account creation, consent grant, and authorization issuance. This required a
separate protocol control rather than an immediate product change.

The existing Python HTTPS acceptance harness completed all nine of its
cross-service journeys, including Message Desk signup/callback, Goja signup,
TinyIDP signup invitations, application membership invitations, and replay
rejections. The browser-specific failure therefore remains open as a
Playwright/Caddy/Chromium navigation defect; it is not evidence that the OIDC
or callback contract is broken.

### Prompt Context

**User prompt (verbatim):** (same as Step 12)

**Assistant interpretation:** Continue investigation with durable evidence and
do not claim an incomplete browser navigation journey as covered.

**Inferred user intent:** The matrix must distinguish actual protocol defects
from test-runner defects while still insisting on browser-level evidence for
the final UX claim.

**Commit (code):** N/A — the temporary failing Playwright journey was removed
instead of being committed as coverage.

### What I did

- Added a temporary Playwright new-identity signup-to-callback journey.
- Confirmed it created a new account and displayed the themed `Approve access`
  form, then observed the page remain at `https://idp.localhost:8443/authorize`
  after approval.
- Inspected the retained trace. The consent approval POST was reported as
  `net::ERR_ABORTED` with no browser-visible response.
- Inspected IDP audit evidence for both reproductions. Each recorded accepted
  signup and consent, followed by `authorize.request.accepted`.
- Ran `python3 scripts/03-browser-acceptance.py`; all nine real-HTTPS
  acceptance journeys passed.
- Removed the temporary test so the committed Playwright suite does not
  contain an unverified or hanging matrix claim.

### Why

- A successful backend audit event is insufficient to mark a browser
  navigation row complete.
- Conversely, changing issuer, consent, callback, or redirect behavior based
  only on a Playwright abort would risk breaking a protocol sequence that the
  independent HTTPS harness has verified end to end.

### What worked

- The control command completed in 4.9 seconds and ended with:

  ```text
  PASS: shared TinyIDP Phase 5 browser acceptance completed
  ```

- It explicitly validated Message Desk open signup and OIDC callback, plus
  invite-gated Goja signup and callback.

### What didn't work

- The temporary Chromium journey did not receive a response for the final
  consent POST. Adding `page.waitForResponse` made the test process wait until
  it was terminated; that confirms the missing response observation but does
  not identify a safe product-side repair.
- The trace recorded `net::ERR_ABORTED` for the approval POST. No TinyIDP or
  Caddy application log contained a matching server exception.

### What I learned

- The acceptance harness and Chromium differ materially in the final approval
  navigation even though both use the same trusted local TLS endpoints.
- `authorize.request.accepted` currently records an `internal_error` reason
  when the supplied reason is empty because `cleanAuditReason("")` maps to its
  fallback. That audit wording is misleading but is not the cause of the
  Chromium abort.

### What was tricky to build

- The final interaction is a cross-origin 303 from TinyIDP to Message Desk.
  Capturing it in a browser test must preserve browser cookies, navigation,
  fetch metadata, and Caddy TLS behavior; replaying it with a server-side
  request alone cannot prove the browser experience.

### What warrants a second pair of eyes

- Review Playwright 1.50 Chromium network behavior with Caddy's HTTP/2/HTTP/3
  listener and the local CA configuration before changing the OAuth response
  writer.
- Review whether an access log with redacted request IDs should be enabled for
  this local Compose profile to make browser/proxy correlation observable.

### What should be done in the future

- Reintroduce the full Playwright callback journey only after its final
  approval navigation is reliably observable.
- Keep the passing Python HTTPS sequence as protocol coverage, but do not use
  it as a substitute for the matrix's browser row.

### Code review instructions

- Run `python3 scripts/03-browser-acceptance.py` from
  `examples/tinyidp-shared-two-apps` as the protocol control.
- Inspect retained Playwright traces before accepting a browser-only fix.
- Start with `issueApprovedAuthorizationArtifacts` and Caddy's three host
  routes if analyzing the final redirect path.

### Technical details

```text
Chromium temporary journey:
  signup -> email code -> password -> consent page -> approval POST
  browser observation: net::ERR_ABORTED, page remains /authorize

Independent HTTPS control:
  same logical sequence -> 303 callback -> RP token exchange -> signed-in app
  result: PASS (both Message Desk and Goja Auth)
```

## Step 14: Cover equivalent invalid-credential UX in Goja Auth

This step made the cross-client error requirement concrete for the credential
boundary. An unknown Goja Auth login now has a real Chromium assertion for the
same generic, non-enumerating error used by Message Desk, while verifying that
TinyIDP selects Goja Auth's own approved stylesheet.

### Prompt Context

**User prompt (verbatim):** (same as Step 12)

**Assistant interpretation:** Continue matrix coverage with independent,
repeatable browser evidence.

**Inferred user intent:** A shared IdP must not accidentally give every relying
party the same visual treatment or leak account existence through a
client-specific error path.

**Commit (code):** `98821fa` — "test: cover Goja invalid credential theme"

### What I did

- Added a `gojaOrigin` browser fixture and a strict Goja theme assertion.
- Added an unknown-login journey starting from
  `https://goja.localhost:8443/auth/login?return_to=/`.
- Asserted generic public copy, retained login, cleared password, and the
  `/static/themes/goja-auth-lab.css` stylesheet route.
- Ran `pnpm exec playwright test -g 'Goja Auth invalid credentials'`.

### Why

- Error-equivalence means both applications use the same safe identity policy,
  not that they share the same CSS asset.
- A server-owned per-client theme route is the tested boundary: application
  code does not submit arbitrary stylesheet URLs to TinyIDP.

### What worked

- The focused Chromium test passed in 2.6 seconds.
- The login value remained visible for correction, while the sensitive password
  field was empty after rejection.

### What didn't work

- The initial assertion expected `/static/themes/goja-auth.css`. Runtime HTML
  correctly served `/static/themes/goja-auth-lab.css`; the test expectation was
  updated to the configured catalog route.

### What I learned

- The runtime theme catalog is authoritative over a filename inferred from the
  Compose bind-mount name.
- The error copy and field-clearing behavior are shared provider policy; the
  theme route is client-specific presentation policy.

### What was tricky to build

- The test must begin at the relying-party URL, not directly at TinyIDP, so the
  client ID and resulting theme choice come from a real OIDC authorization
  request.

### What warrants a second pair of eyes

- Confirm the theme catalog continues to expose the exact `goja-auth-lab.css`
  route expected by this local deployment profile.

### What should be done in the future

- Add matching cross-client coverage for recoverable signup and terminal
  callback failures after the Playwright final-navigation defect is resolved.

### Code review instructions

- Review `expectGojaAuthTheme` and the Goja invalid-credential test.
- Run `pnpm exec playwright test -g 'Goja Auth invalid credentials'` from the
  browser-tests directory.

### Technical details

```text
Goja Auth login URL -> TinyIDP authorize(client_id=goja-auth-host-demo)
                   -> invalid credentials
                   -> generic public error; login retained; password cleared
                   -> /static/themes/goja-auth-lab.css
```

## Step 15: Cover browser-native signup validation boundaries

This step covered two validation rows that the browser can reject before a
workflow request reaches TinyIDP. The identity form requires a display name and
publishes the durable 120-character boundary. After email verification, the
password form rejects a 14-character secret using its public 15-character
minimum without posting the password page.

### Prompt Context

**User prompt (verbatim):** (same as Step 12)

**Assistant interpretation:** Continue filling the browser UX matrix with
focused tests rather than repeatedly exercising unrelated full suites.

**Inferred user intent:** Form validation should be immediately understandable
and should avoid creating misleading server-side error pages for values the
browser can reject safely.

**Commit (code):** `f5f9eaf` — "test: cover native signup validation UX"

### What I did

- Added a display-name validation journey that asserts `maxlength="120"` and
  native required-field focus behavior.
- Added a verified-email password journey that submits a 14-character password
  and asserts the browser's `validity.tooShort` result and password focus.
- Ran `pnpm exec playwright test -g 'display names|short password'`.

### Why

- The browser's built-in constraint validation gives instant, accessible
  feedback and prevents a needless request containing invalid user input.
- The server still enforces the same descriptor limits; the tests cover the UI
  boundary, not a replacement for native validation.

### What worked

- Both Chromium tests passed in 2.0 seconds.
- Both assertions remained on the client-themed TinyIDP page, and the short
  password flow reached the real email-code/password stage before validation.

### What didn't work

- N/A.

### What I learned

- A browser cannot type beyond the input's `maxlength`; testing the rendered
  bound is the correct end-user assertion for the over-limit display-name
  case.
- `minlength` on a password input is observable through `validity.tooShort`
  without retaining or rendering the secret.

### What was tricky to build

- The password validation case must first pass the real email challenge; a
  shallow DOM-only test would not prove that the dynamic password presentation
  carries the descriptor constraints.

### What warrants a second pair of eyes

- Verify any future custom workflow renderer preserves `required`, `minlength`,
  and `maxlength` attributes from the provider-owned descriptors.

### What should be done in the future

- Cover server-side malformed/replayed submissions separately, because browser
  constraint validation cannot protect non-browser clients or crafted POSTs.

### Code review instructions

- Review the two native-validation tests in `authentication-ux.spec.ts`.
- Run `pnpm exec playwright test -g 'display names|short password'` from the
  browser-tests directory.

### Technical details

```text
display name: required + maxlength=120 -> browser prevents empty submit
password: minlength=15                  -> browser prevents 14-character POST

The provider-owned workflow descriptor remains the single source of both
HTML constraints and server-side ParseSubmission bounds.
```

## Step 16: Never redisplay rejected email codes and split fast from full commit gates

The wrong-email-code browser test showed that the retry HTML contained the
previous one-time code. This is not acceptable retry state: although the code
is text rather than a password, it is authentication evidence and must not be
rendered after a failed verification attempt. The descriptor now declares that
field non-redisplayable, so the existing public-value projection omits it.

This Go change also exposed why each commit felt slow. The old pre-commit hook
ran the entire repository suite, including ticket-local, two-process k3s
deployment harnesses. The hook now runs a fast reusable-package and Message
Desk suite; `make test` and pre-push retain the complete Fosite and production
harness coverage.

### Prompt Context

**User prompt (verbatim):** "how long does the entire test process / commit gate take?"

**Assistant interpretation:** Explain the latency source and make the normal
commit feedback loop proportionate without weakening the full validation gate.

**Inferred user intent:** Keep frequent, safe commits practical while retaining
the expensive deployment-shaped tests at a sensible boundary.

**Commit (code):** `bd4c424` — "fix: never redisplay rejected email codes"

### What I did

- Added a browser journey for a wrong email code, expecting a themed retry form
  and resend action.
- Observed that the rejected code was rendered back into the input.
- Allowed non-secret fields to opt into `RedisplayNever` and assigned that
  policy to `FieldEmailCode`.
- Added a unit regression proving email codes do not enter `PublicValues`.
- Rebuilt only the local IDP and reran the focused browser test successfully.
- Updated a stale k3s two-process harness expectation from its old generic
  duplicate-email copy to the current actionable themed duplicate-identity
  guidance; its focused test passed in 22.099 seconds.
- Added `make test-fast` and changed the pre-commit test hook to use it.
  Full `make test` remains the pre-push command.

### Why

- Retry pages are part of the authentication boundary. They must not retain a
  one-time code in HTML, DOM values, screenshots, browser history, or shared
  screen captures.
- The full suite includes deliberately expensive integration harnesses. They
  are essential before push but are disproportionate for every small Go edit.

### What worked

- `go test ./pkg/idpworkflow ./internal/fositeadapter -count=1` passed before
  the local rebuild.
- `pnpm exec playwright test -g 'wrong email verification code'` passed after
  rebuilding the IDP.
- The focused two-process k3s harness passed after its expected public copy was
  updated.
- The new pre-commit gate passed lint and `make test-fast` in 7.90 seconds.

### What didn't work

- The original full suite failed after 1–2 minutes because
  `TestTwoProcessRegistrationRedirectAndSignup` still asserted the obsolete
  generic duplicate-email wording. Its failure was a stale test expectation,
  not a regression from non-redisplayable email codes.

### What I learned

- `RedisplayNever` is a distinct concern from `Sensitive`: a value can be
  safely parsed as text for a current native verifier while still being unsafe
  to carry into a later rendered presentation.
- The k3s harness is a Go test under the production deployment ticket. It
  launches TinyIDP and Message Desk as separate local processes and checks the
  production-shaped signup/redirect boundary; it is not Kubernetes itself.

### What was tricky to build

- The validation implementation cannot simply mark email codes as password
  secrets, because that would change the input kind and presentation contract.
  Allowing non-secret `RedisplayNever` preserves the one-time-code input while
  closing the redisplay channel.

### What warrants a second pair of eyes

- Review all future short-lived proof fields—recovery, device, or invitation
  values—to decide explicitly whether they may be redisplayed.
- Review the fast test set periodically so it remains meaningful without
  silently becoming a second full suite.

### What should be done in the future

- Complete the other email-code state rows: exhausted, expired, replayed, and
  resend-limit behavior.
- Keep full `make test` green before pushing; do not use `test-fast` as a
  release or deployment substitute.

### Code review instructions

- Start with `FieldEmailCode` in `pkg/idpworkflow/descriptors.go` and the
  `ParseSubmission` projection test.
- Run `make test-fast` for the pre-commit suite, then run the focused
  Playwright wrong-code test from the browser-tests directory.
- Run the two-process harness package or full `make test` before push when
  reviewing production-shape changes.

### Technical details

```text
wrong code POST
  -> ParseSubmission
  -> email_code excluded from PublicValues (RedisplayNever)
  -> verifier rejects code
  -> themed retry page with blank input + resend action

pre-commit: make test-fast + make lint  (~8 seconds observed)
pre-push:   make test + make lint       (includes Fosite and k3s harnesses)
```

## Step 17: Cover resend as browser recovery state

The Mailpit convenience endpoint cannot safely establish message order when two
codes are sent to the same recipient in the same short interval. The browser
test therefore verifies the durable user-facing resend contract directly: the
same workflow remains available, the code field is blank, the resend action is
still available, and the client theme remains loaded.

### Prompt Context

**User prompt (verbatim):** (same as Step 16)

**Assistant interpretation:** Continue adding reliable matrix evidence without
making a mail-test-helper ordering assumption part of the product contract.

**Inferred user intent:** Recovery paths should be proven in the browser while
mail delivery ordering remains a service-level concern.

**Commit (code):** `137ebd3` — "test: cover email code resend recovery"

### What I did

- Replaced a failing attempt to compare Mailpit's `view/latest.txt` values.
- Added a Playwright resend journey that asserts the blank retry field, resend
  action, and Message Desk theme after the real submit.
- Ran `pnpm exec playwright test -g 'email-code resend keeps'`.

### Why

- The provider promises retry-safe browser state, not a particular Mailpit UI
  ordering implementation.

### What worked

- The focused Chromium test passed in 3.6 seconds.

### What didn't work

- Comparing two Mailpit “latest” reads produced a different code but a later
  read could return the earlier message, so the test submitted an invalid code.

### What I learned

- Mail delivery replacement should be covered through the email-challenge
  service tests; browser tests should assert observable recovery UX.

### What was tricky to build

- The resend action intentionally skips browser field validation, so the test
  must confirm the provider returns a valid continuation page rather than
  merely that a button was clickable.

### What warrants a second pair of eyes

- Review any future outbox test helper for a stable message identifier or
  timestamp contract before using it to assert sequence ordering.

### What should be done in the future

- Add explicit exhaustion and replay browser journeys with deterministic
  provider-controlled fixtures.

### Code review instructions

- Run `pnpm exec playwright test -g 'email-code resend keeps'` from the
  browser-tests directory.

### Technical details

```text
Send another code -> continuation advance -> themed code form
                                         -> blank input
                                         -> resend remains available
```

## Step 18: Cover RP-initiated provider logout

This browser journey verifies the distinction between the relying-party session
and the TinyIDP browser session at the full logout boundary. Message Desk first
revokes its own application session, then navigates the browser to TinyIDP's
validated end-session endpoint; TinyIDP clears its session cookies and returns
the browser to Message Desk's registered post-logout origin.

### Prompt Context

**User prompt (verbatim):** (same as Step 16)

**Assistant interpretation:** Continue the session and logout portions of the
matrix with actual browser cookie and navigation evidence.

**Inferred user intent:** Users must be able to understand and trust the scope
of “log out everywhere,” especially in a shared identity-provider topology.

**Commit (code):** `9d25a40` — "test: cover provider logout browser flow"

### What I did

- Logged into Message Desk using the seeded local administrator.
- Clicked the application’s `Log out everywhere` control.
- Asserted the registered Message Desk post-logout URL, guest UI, and absence
  of a non-empty `tinyidp_session` cookie in the Playwright browser context.

### Why

- An RP-only logout must not be mistaken for provider logout. This row proves
  the stronger user action removes both layers of browser authentication.

### What worked

- `pnpm exec playwright test -g 'logging out everywhere'` passed in 2.6
  seconds.

### What didn't work

- A temporary one-account chooser assertion did not show `Choose an account`.
  With only one remembered identity, TinyIDP correctly completed the
  `prompt=select_account` request without a selection UI. The temporary test
  was removed; the chooser row remains open and requires two identities.

### What I learned

- The app emits a real `prompt=select_account` request for change-account.
  Whether a chooser renders depends on remembered-account cardinality, not
  merely the prompt value.

### What was tricky to build

- The browser test must inspect the IdP-origin cookie jar after a cross-origin
  redirect; the Message Desk guest page alone would not prove provider logout.

### What warrants a second pair of eyes

- Review whether the product should visually acknowledge provider logout in
  addition to its guest-mode state.

### What should be done in the future

- Create two remembered identities deterministically, then cover chooser
  selection and account removal without relying on the flaky final-signup
  Playwright navigation path.

### Code review instructions

- Run `pnpm exec playwright test -g 'logging out everywhere'` from the
  browser-tests directory.

### Technical details

```text
Message Desk logout everywhere
  -> revoke RP session
  -> browser GET TinyIDP end-session with registered post-logout URL
  -> TinyIDP revokes browser session and clears cookie
  -> Message Desk guest mode
```

## Step 19: Enable and exercise the production account chooser

The local host had a complete provider-side chooser implementation, but
`serve-production` never opted into it. As a result, an RP's
`prompt=select_account` request could finish with the current TinyIDP session
instead of presenting the account-choice interaction that the Message Desk UI
advertises. This step adds a deliberately opt-in host flag and enables it only
in the local shared-two-app Compose deployment.

The browser journey now proves the entire useful path with seeded identities:
the administrator is remembered, the user chooses another account and signs in
as the invitee, then the next chooser page presents both identities and
switches back to the administrator. This is production-shaped configuration,
not a test-only provider shortcut.

### Prompt Context

**User prompt (verbatim):** "yes thanks"

**Assistant interpretation:** Continue the narrowed browser UX work after the
test-gate discussion, using the local stack to close the remaining session and
account-selection coverage.

**Inferred user intent:** The development feedback loop should stay quick, but
the browser matrix must still exercise the real multi-account identity-provider
behavior before deployment work resumes.

**Commit (code):** `d940253` — "feat: enable opt-in production account chooser"

**Commit (code):** `fadfc08` — "test: cover production account chooser browser flow"

### What I did

- Added the optional `--account-chooser` `serve-production` flag.
- Bound the opt-in to TinyIDP's existing `AccountChooserConfig`, including
  password-login remembering and a deliberate display-label policy: trimmed
  account name first, then preferred username.
- Enabled that flag only in
  `examples/tinyidp-shared-two-apps/compose.yaml`.
- Rebuilt the local `idp` Compose service in tmux.
- Added and ran a real Chromium test that remembers the two seeded users,
  renders both labels, and switches between them.

### Why

- A shared IdP needs account selection to be a conscious host policy because
  remembered names reveal earlier use of that browser. The production command
  must not enable it accidentally.
- The local two-app environment is specifically intended to expose shared-IdP
  session behavior before a cluster rollout.

### What worked

- `GOWORK=off go test ./internal/cmds -run
  'TestProductionCommandRequiresSignupProgramAndDropsLegacyRegistrationFlag'
  -count=1` passed.
- `docker compose -f examples/tinyidp-shared-two-apps/compose.yaml config`
  passed.
- The normal pre-commit `make test-fast` and lint gate passed in 9.07 seconds
  while committing `d940253`.
- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'account chooser remembers two password'` passed in 7.2 seconds.

### What didn't work

- Rebuilding the CGO-enabled TinyIDP Docker image spent roughly two minutes
  compiling `github.com/mattn/go-sqlite3`'s `sqlite3-binding.c`. This was build
  time, not a product or test failure.

### What I learned

- The provider already creates the chooser whenever an RP sends
  `prompt=select_account` and a remembered browser context exists; the missing
  seam was host configuration, not protocol or renderer support.
- Account selection can require a separate consent continuation after a switch,
  so the browser test deliberately accepts a consent page if it is shown.

### What was tricky to build

- The chooser needs a safe browser-visible label policy. Reusing an arbitrary
  user record field would make the privacy decision implicit. The flag both
  enables remembering and selects the reviewed name/preferred-username policy,
  while the zero value leaves all remembered-account state disabled.
- The test must create both remembered sessions in one browser context. It does
  so through the actual `Use another account` action, rather than adding
  provider cookies or store rows directly.

### What warrants a second pair of eyes

- Confirm that production operators who enable `--account-chooser` accept that
  remembered account display names are visible to later users of the same
  browser profile.
- Review whether the final production deployment should enable the flag now or
  leave it disabled until its product privacy policy is documented.

### What should be done in the future

- Add the explicit remembered-account removal journey; the provider supports
  it, but this matrix increment verifies selection and switching first.
- Finish the separate signup failure rows (password mismatch, expired,
  exhausted, and replayed email codes) before closing the whole matrix.

### Code review instructions

- Start with `serveProductionSettings` and `runProductionHost` in
  `internal/cmds/serve_production.go`.
- Inspect the exact local opt-in in
  `examples/tinyidp-shared-two-apps/compose.yaml`.
- Run the focused command test, rebuild `idp`, then run the named Playwright
  chooser test from the browser-tests directory.

### Technical details

```text
Message Desk Change account
  -> OIDC authorize?prompt=select_account
  -> TinyIDP browser-context lookup
  -> Choose an account [Local Administrator]
  -> Use another account
  -> password login [Local Invitee]
  -> remembered context: Administrator + Invitee
  -> Choose an account -> Administrator -> Message Desk
```

## Step 20: Cover remembered-account removal

Account selection is not complete if the browser can only accumulate remembered
identities. The existing chooser contract permits a user to remove one opaque
remembered entry without deleting the TinyIDP account or the other remembered
entries. This step extends the same real browser context after two-account
switching and verifies that distinction.

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Complete the remaining account-chooser state
transition while the stack and two seeded identities are available.

**Inferred user intent:** A browser identity-management UI must be reversible
and unsurprising, not merely capable of selecting an account once.

**Commit (code):** `492a659` — "test: cover remembered account removal"

### What I did

- Extended the two-identity Chromium journey after it switched back to Local
  Administrator.
- Re-opened account selection, selected Local Invitee, and submitted the real
  `Remove account` action.
- Asserted that Local Invitee disappears while Local Administrator remains
  available.

### Why

- Removing a remembered browser choice must not delete the durable user or
  accidentally leave a stale chooser entry that can later be activated.

### What worked

- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'account chooser remembers two password'` passed in 4.6 seconds.

### What didn't work

- N/A

### What I learned

- The production renderer exposes removal as a normal server-owned form action;
  no client-side cookie or database manipulation is needed to test it.

### What was tricky to build

- The test keeps the administrator selected as the active browser identity,
  then removes the invitee from the remembered set. This proves the correct
  scope of removal: remembered membership changes while the current provider
  session and durable accounts remain intact.

### What warrants a second pair of eyes

- Confirm the product wording makes clear that removal only forgets an account
  on this browser and does not delete the identity.

### What should be done in the future

- Add a visual/help-copy review for the removal scope if the chooser becomes
  enabled in the public production deployment.

### Code review instructions

- Read the final section of the named chooser test.
- Run the focused Playwright command above against the Compose stack.

### Technical details

```text
chooser [Administrator, Invitee]
  -> select Invitee + Remove account
  -> chooser [Administrator]
  -> durable Invitee account unchanged
```

## Step 21: Restore request-scoped email-code verification

The non-redisplay policy for email codes correctly stopped a rejected code from
returning in HTML, but a provider regression test showed that it had also
removed the code from the native verifier input. A valid code consequently
could never advance from the email-code page. This step restores the intended
three-way separation: public fields may render, secrets and evidence may be
resolved only during the current native request, and neither becomes durable
workflow presentation state.

The same implementation adds closed, provider-owned messages for expired,
attempt-exhausted, and resend-limited email-code states. The error code, not an
underlying storage error, crosses the native/rendering boundary.

### Prompt Context

**User prompt (verbatim):** (same as Step 19)

**Assistant interpretation:** Continue covering the browser UX matrix while
preserving the security rule that one-time codes are never redisplayed.

**Inferred user intent:** Authentication error handling must be both secure and
actionable; fixing an information-flow issue must not quietly break signup.

**Commit (code):** `cd93fec` — "fix: preserve verifiable email codes and explain limits"

### What I did

- Changed `ParseSubmission` so a `RedisplayNever` non-secret value is stored
  in the request-scoped secret set instead of being discarded.
- Resolved the email-code handle only in `resumeScriptedSignup`, passed a
  transient clone to the native email challenge verifier, and cleared it.
- Added closed field-error codes and public copy for expiry, exhausted attempts,
  and resend limits.
- Added focused Go regressions for the projection and safe public copy.
- Added Chromium tests for attempt exhaustion and resend-limit presentation.

### Why

- An email code is authentication evidence: it must reach the native verifier
  for this request, but must not enter `PublicValues`, JavaScript input, a
  continuation, a rendered retry form, or audit data.

### What worked

- `GOWORK=off go test ./pkg/idpworkflow ./pkg/idpui ./internal/fositeadapter
  -run 'Test(ParseSubmissionDoesNotRedisplayEmailVerificationCodes|WorkflowEmailCodeFailureHasSpecificPublicCopy|EmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification|EmailVerifiedScriptedSignupSurvivesSQLiteRestart)' -count=1`
  passed.
- The normal fast pre-commit gate and lint passed while committing `cd93fec`.
- The real Chromium resend-limit journey passed and displayed the closed,
  themed recovery message.

### What didn't work

- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'email-code (attempt exhaustion|resend limit)'` still failed the
  exhaustion assertion after a rebuilt image: the final page showed
  `This value could not be accepted.` rather than the new attempt-limit text.
- The initial failed run was made against an image built before the
  request-scoped verifier correction; its durable challenge record had
  `Attempts:0`. A second rebuild still reproduced the generic final page, so
  this remaining discrepancy is recorded as a defect rather than receiving a
  third speculative implementation change.

### What I learned

- `RedisplayNever` must mean “private to the current verifier,” not “drop the
  submitted value.” It is not equivalent to a display-only omission.
- The resend path is independent of verification and therefore proved the new
  error taxonomy even while attempt-exhaustion remains unresolved.

### What was tricky to build

- The parser has to normalize a non-secret code before placing it in the
  request-scoped set, while password descriptors must remain unnormalized
  secrets. Treating every non-redisplayed field as a password would change
  its input and validation contract.

### What warrants a second pair of eyes

- Trace the deployed exhaustion POST through `ParseSubmission`,
  `emailChallenges.Verify`, and `renderScriptedSignupEmailCodeError` before
  changing it again. Confirm the runtime error class and attempt counter rather
  than inferring it from the generic browser page.

### What should be done in the future

- Resolve the attempt-exhaustion discrepancy with one instrumented,
  deterministic trace, then retain the Chromium regression.
- Add expiry and replay fixtures after the error-class trace is understood.

### Code review instructions

- Start with `Submission` in `pkg/idpworkflow/submission.go`, then inspect the
  email-code branch in `internal/fositeadapter/scripted_signup.go`.
- Run the focused Go command above. Run the resend-limit test separately; do
  not treat the currently failing exhaustion assertion as passing evidence.

### Technical details

```text
email_code POST
  -> normalize (trim)
  -> request-scoped SecretHandle (never PublicValues)
  -> native email challenge Verify(code)
  -> clear clone + destroy handle set
  -> themed field error only when verification rejects
```

## Step 22: Cover Goja Auth invitation rejection

The shared local stack has a different policy per relying party: Message Desk
is open signup, while Goja Auth starts the same TinyIDP workflow with an
invite-code field. This step proves that the real Goja `/auth/register` route
preserves that policy and gets Goja's client-specific theme when an unknown
invite is rejected.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Continue an independent browser-matrix row while
the email-code exhaustion defect remains explicitly bounded.

**Inferred user intent:** Two applications may share the provider without
silently sharing signup policy or visual presentation.

**Commit (code):** `2403443` — "test: cover Goja invitation rejection UX"

### What I did

- Added a Goja signup helper that enters through the actual RP route and
  verifies the resulting TinyIDP authorize origin and Goja CSS.
- Submitted a unique identity plus an unknown invitation code.
- Asserted that the invite value remains available for correction, the provider
  renders a field error, and the `goja-auth-lab.css` theme remains active.

### Why

- Invitation policy and visual client identity are both provider-selected by
  the validated OAuth client. A Message Desk result cannot prove Goja's path.

### What worked

- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'Goja signup rejects an unknown invitation'` passed in 2.3 seconds.

### What didn't work

- N/A

### What I learned

- Goja's `/auth/register` forwards the standard PKCE/OIDC request with
  `tinyidp_signup=1`; TinyIDP adds the invite field only after validating the
  `goja-auth-host-demo` client identifier.

### What was tricky to build

- The test deliberately uses an unknown code rather than creating an invitation
  through storage internals. It exercises the normal browser policy boundary
  and avoids turning a test fixture into authorization evidence.

### What warrants a second pair of eyes

- Review the generic invite rejection copy before public rollout: it avoids
  leaking invitation existence, but the product may want a more helpful
  provider-owned generic sentence.

### What should be done in the future

- Add durable invitation lifecycle fixtures—valid, consumed, revoked, expired,
  and wrong-audience—using supported administrative setup rather than direct
  database mutation.

### Code review instructions

- Run the named focused Goja Playwright test with the Compose stack running.
- Inspect `open-signup.js` to see the client-ID-gated invite descriptor.

### Technical details

```text
Goja /auth/register
  -> OIDC authorize (client_id=goja-auth-host-demo, tinyidp_signup=1)
  -> TinyIDP client policy adds inviteCode
  -> provider lookup rejects unknown code
  -> Goja-themed field error; no email challenge sent
```

## Step 23: Render Message Desk callback errors as safe recovery pages

The callback matrix probe found that Message Desk returned `text/plain`
`identity login was not accepted` for an OIDC error redirect. This is an
application-owned failure page, so provider themes cannot repair it. The
handler now returns CSP-bound HTML using the application’s existing stylesheet,
fixed copy, and two safe navigation choices.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Complete the RP callback-error row with
browser-visible evidence instead of accepting a provider-only themed flow.

**Inferred user intent:** OAuth errors must remain understandable and safe at
the relying-party boundary as well as inside TinyIDP.

**Commit (code):** `9c70f31` — "fix: render Message Desk callback recovery page"

**Commit (code):** `cb5d2ca` — "test: cover Message Desk callback recovery UX"

### What I did

- Replaced raw callback `http.Error` responses with a same-origin HTML recovery
  page for provider errors and failed callback completion.
- Used only fixed provider-owned copy, escaped the internal title/summary
  arguments defensively, did not reflect `error` or `error_description`, and
  linked only to local Message Desk routes.
- Added unit and Chromium coverage for content type, CSS, recovery links, and
  non-reflection of attacker-controlled query text.

### Why

- The RP owns the last page users see after OAuth returns. A raw protocol error
  is not an acceptable user experience and could accidentally reflect
  untrusted query input in a future change.

### What worked

- Focused Message Desk Go tests passed.
- The fast pre-commit test and lint gate passed in 13.88 seconds.
- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'Message Desk OIDC callback error'` passed in 2.5 seconds after the
  rebuilt service.

### What didn't work

- The same direct probe still finds Goja Auth’s separate repository handler
  returns `401 text/plain: oidc error: access_denied`. It is recorded as an
  external cross-client defect; this ticket did not silently edit that other
  checkout.

### What I learned

- TinyIDP’s client theme selects provider interaction pages, not callback pages
  rendered by a relying application. Each RP must own its own callback-error
  presentation contract.

### What was tricky to build

- The callback page must preserve strict CSP without relying on inline style or
  JavaScript. Reusing the existing same-origin compiled application stylesheet
  gives a coherent page while retaining `default-src 'none'`.

### What warrants a second pair of eyes

- Review whether Message Desk should distinguish a user cancellation from a
  provider failure more visibly. Both current pages deliberately avoid
  provider-supplied details.

### What should be done in the future

- Open or update the corresponding Goja Auth work item to replace its raw
  callback errors with an equivalent application-owned recovery page.

### Code review instructions

- Read `handleCallback` and `renderCallbackError` in
  `examples/tinyidp-message-app/app_http.go`.
- Run the focused Go test, rebuild Message Desk, and run the named Playwright
  test.

### Technical details

```text
OIDC callback?error=...
  -> Message Desk fixed error class
  -> HTML + /static/app/assets/index.css + strict CSP
  -> Try signing in again | Return to Message Desk
  -> no provider error query reflected
```

## Step 24: Render replayed signup continuations as themed terminal pages

The signup workflow can correctly reject a continuation that was consumed by a
previous successful code verification, but its early load failure still used a
raw `http.Error`. This step sends continuation-load, retired-generation, and
invalid-presentation failures through the existing client-themed terminal
renderer. It does not weaken replay protection or turn the replay into a live
form retry.

### Prompt Context

**User prompt (verbatim):** (same as Step 21)

**Assistant interpretation:** Cover stale and replayed browser authority with
the same safe, actionable terminal experience required for ordinary mistakes.

**Inferred user intent:** A security rejection must remain fail-closed while
giving a browser user a clear route back to a fresh registration flow.

**Commit (code):** `73b0c0d` — "fix: render stale signup continuations safely"

**Commit (code):** `10190ba` — "test: cover replayed signup terminal UX"

### What I did

- Added `renderSignupTerminalError`, a bounded `BrowserErrorPage` with fixed
  restart copy and the already-validated client ID.
- Routed unavailable continuations, unavailable program generations, and
  invalid persisted workflow descriptors through it.
- Extended the provider replay test to require themed HTML and non-reflection
  of the former raw fallback.
- Added a Chromium journey that captures a real email-code form, consumes it
  normally, then submits the captured POST again in the same browser origin.

### Why

- A consumed continuation is terminal authority, not a correctable field
  value. The user must restart, but should not receive raw text or protocol
  details.

### What worked

- `GOWORK=off go test ./internal/fositeadapter -run
  TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification
  -count=1` passed.
- The fast test and lint gate passed while committing `73b0c0d`.
- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'replayed signup form returns'` passed in 2.2 seconds.

### What didn't work

- A first back-navigation scenario was not a true stale continuation: the
  identity-page continuation legitimately advanced to a new email challenge.
- `page.goBack()` after the email-code POST produced Chromium
  `net::ERR_CACHE_MISS`, so the browser test now captures the original form
  fields before consuming them and submits a new same-origin HTML form.
- The form contains a hidden input named `action`, which shadows DOM
  `form.action`; reading `form.getAttribute("action")` avoids an accidental
  navigation to `/[object Object]`.

### What I learned

- A stale UX test must target a consumed/retired authority, not merely an
  earlier presentation that the workflow intentionally allows to advance.

### What was tricky to build

- The browser test cannot use `fetch` under the provider’s deliberate
  `default-src 'none'` CSP. Creating and submitting an ordinary hidden form
  preserves the real navigation and CSP contract.

### What warrants a second pair of eyes

- Confirm the terminal wording correctly distinguishes “restart” from a
  recoverable wrong code, where the existing workflow retry remains valid.

### What should be done in the future

- Add an expiry fixture that exercises the same terminal renderer after a
  continuation’s actual TTL elapses or is operator-expired.

### Code review instructions

- Start at `resumeScriptedSignup` and `renderSignupTerminalError`.
- Run the focused provider test and named Chromium replay test.

### Technical details

```text
valid email-code POST -> consume continuation -> password page
captured old email-code POST -> continuation load rejects
                           -> BrowserErrorPage(client theme)
                           -> restart registration guidance
```

## Step 25: Split the ordinary test loop from the strict provider and process proofs

The repository had already moved its pre-commit hook to `make test-fast`, but
the public `make test` target still meant `go test ./...`. That collected the
strict Fosite adapter suite and every executable experiment below `ttmp/`,
including a production-shaped two-process harness. This step makes the cost and
purpose of each level explicit without removing release coverage.

The result is intentionally a routing change, not a reduction in assurance:
ordinary edits get a short feedback loop; a local push and CI still execute the
full module package graph. A developer can now request exactly the Fosite or
two-process proof that is relevant to their change.

### Prompt Context

**User prompt (verbatim):** "yes thanks"

**Assistant interpretation:** Apply the proposed separation between the fast
daily gate and the expensive protocol/production-topology checks, and explain
the harness by naming it accurately in the developer interface.

**Inferred user intent:** Keep the high-assurance checks available and required
before sharing work, but do not make unrelated local commits wait on them.

**Commit (code):** `a99b0ed` — "build: split fast and production test gates"

### What I did

- Made `make test` delegate to the existing fast package and Message Desk loop.
- Added `make test-fosite` for `./internal/fositeadapter` and `make
  test-k3s-harness` for the deployment ticket's two-process test package.
- Added `make test-full` for `go test ./... -count=1` and wired Lefthook
  pre-push to that target.
- Kept `verify` conservative by making it depend on `test-full`.
- Documented the commands and corrected the misleading shorthand: the harness
  is stored under the k3s ticket, but does not start or require a k3s cluster.

### Why

- Fosite tests exercise the strict OAuth/OIDC provider boundary and are slower
  than package-local checks.
- The two-process harness builds real deployable binaries and proves trusted
  reverse-proxy behavior; it is production evidence, not an edit-time unit
  test.
- Explicit target names let an implementer select the smallest adequate check
  while preserving a full pre-push and CI gate.

### What worked

- `make test` passed in 9.3 seconds.
- `make -n test-fosite test-k3s-harness test-full` showed the intended,
  distinct commands without running the expensive variants.
- `git diff --check` passed before commit.

### What didn't work

- N/A. This was a build-contract refactor; no test failures were observed.

### What I learned

- The expensive test had been entering the normal developer workflow through
  the ambiguous `make test` name, despite the pre-commit hook already using
  `test-fast`.
- A ticket directory name is not an execution environment: the so-called k3s
  harness is a local Go process/proxy integration test.

### What was tricky to build

- The required assurance split has two independent controls: developer targets
  and hooks. Changing only `make test` would make a push fast by accident;
  changing only Lefthook would leave everyday manual commands surprising. The
  implementation therefore adds named targets, points pre-push at `test-full`,
  and keeps CI's direct `go test ./... -count=1` unchanged.

### What warrants a second pair of eyes

- Confirm that projects invoking `make test` in external automation want the
  new fast semantics. This repository's GitHub CI uses direct full `go test
  ./... -count=1`, and `verify`/pre-push remain full, so release coverage is
  not reduced.

### What should be done in the future

- If the two-process proof remains confusing, move it from the dated ticket
  `ttmp/` directory to a stable `test/integration/` location in a separately
  scoped cleanup. Do not do that as part of the browser UX ticket.

### Code review instructions

- Read the test-target comments and commands in `Makefile`, then the matching
  `pre-push` command in `lefthook.yml`.
- Run `make test` for the fast loop; inspect `make -n test-fosite
  test-k3s-harness test-full` to verify target selection. Run `make test-full`
  deliberately before a release or when validating the entire repository.

### Technical details

```text
ordinary edit / pre-commit
  -> make test (= test-fast) + lint

provider change (explicit)
  -> make test-fosite

production process/proxy change (explicit)
  -> make test-k3s-harness

git push / CI-equivalent local check
  -> make test-full (= go test ./... -count=1)
```

## Step 26: Render Goja Auth callback failures as safe, application-styled recovery pages

Goja Auth previously allowed the browser-facing callback handler to return a
raw `401 text/plain` protocol string when TinyIDP returned an OAuth error.
This step gives the reusable Goja OIDC handler a deliberately bounded recovery
page contract and wires the local generated-host example to its own static
stylesheet. The provider still owns authorization; the relying application now
owns the final browser page it presents after an unsuccessful callback.

The result is not a TinyIDP theme change. TinyIDP renders its own interaction
pages, while Goja Auth renders `/auth/callback`; each process uses a
same-origin stylesheet and a separate, restrictive CSP. This preserves clear
ownership when one TinyIDP serves several applications.

### Prompt Context

**User prompt (verbatim):** "ok, let's focus on the callback part first."

**Assistant interpretation:** Resolve the raw Goja Auth OIDC callback failure
before returning to the remaining browser-matrix paths.

**Inferred user intent:** A user who cancels or cannot complete sign-in must
receive a polished, safe application page rather than a raw OAuth diagnostic.

**Commit (code):** `f8ff1af` in the sibling Goja checkout — "auth: render safe
OIDC callback recovery pages"

**Commit (Compose):** `9c37b66` — "compose: style Goja callback recovery pages"

**Commit (browser test):** `a62d319` — "test: cover Goja callback recovery UX"

### What I did

- Located the actual Compose build source at
  `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja`; its current
  reusable package is `pkg/gojahttp/auth/oidcauth`.
- Added `CallbackErrorPage` to the Goja OIDC handler configuration, accepting
  only absolute same-origin paths for its optional stylesheet and recovery
  links.
- Replaced every raw callback error boundary with fixed HTML recovery copy:
  cancellation, restart-required authority failure, and generic retry.
- Set `no-store`, strict CSP, `no-referrer`, no-sniff, and frame-denial headers;
  OAuth `error` and `error_description` are not reflected.
- Added a generated-host setting and configured this local Goja application to
  use `/static/styles.css`.
- Added focused Go tests, a direct trusted-HTTPS response probe, and a
  Playwright regression test for `access_denied`.

### Why

- Callback errors cross a trust boundary. Provider-controlled query values and
  internal token-validation details must not be rendered to the browser.
- The generic auth library cannot know an application's CSS contents, so it
  accepts a validated same-origin stylesheet path; the host chooses that path.
- A dedicated recovery page gives users an actionable retry while retaining
  opaque state/nonce/token validation failures.

### What worked

- `go test ./pkg/gojahttp/auth/oidcauth ./pkg/xgoja/hostauth -count=1` passed.
- The Goja pre-commit lint and vet gate passed before `f8ff1af`.
- The rebuilt local endpoint returned `401 text/html; charset=utf-8` with the
  expected CSP, `/static/styles.css`, cancellation copy, and no reflected
  `<script>` payload.
- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'Goja Auth OIDC callback error' --reporter=list` passed in 1.8
  seconds.

### What didn't work

- The first source search inspected a different, older Goja checkout using the
  former `keycloakauth` package name. It was not committed. Compose actually
  builds the sibling checkout in this workspace, which uses `oidcauth`; the
  implementation was repeated there before any deployment or verification.
- The first pre-commit lint pass found an omitted explicit `callbackErrorRetry`
  branch in the closed error-kind switch. Adding that branch made the
  exhaustiveness gate pass.

### What I learned

- “Application-styled callback” does not mean an identity-provider theme. The
  RP has to render and secure this page itself because it owns the callback
  route and static asset mount.
- A small presentation capability—fixed copy plus validated local paths—is
  enough. It does not require exposing protocol errors, templates, or a Goja
  runtime to the callback error boundary.

### What was tricky to build

- The recovery page needs enough configuration to use each application's
  stylesheet but must reject external URLs, authority-style `//` URLs, and
  backslash paths. The configuration is normalized once at handler creation;
  request query values never participate in renderer selection.
- The local Compose image uses a sibling repository as its Docker build
  context. Checking that path before rebuilding was essential; otherwise a
  green unit test could have validated code the running service never used.

### What warrants a second pair of eyes

- Review the public `CallbackErrorPage` API as a reusable host-auth surface:
  its intentionally small field set is designed to prevent it becoming a
  general template or redirect configuration language.
- Confirm other generated-host examples that want styled callback errors set a
  real same-origin stylesheet path rather than relying on the unstyled safe
  default.

### What should be done in the future

- Continue the separate happy-path Chromium approval-navigation investigation.
- Resolve the email-code attempt-exhaustion copy before closing the overall
  matrix task.

### Code review instructions

- Start at `oidcauth.handleCallback`, `normalizeCallbackErrorPage`, and
  `renderCallbackError` in the sibling Goja checkout.
- Review `hostauth.BuildNativeHandlers` and the new generated-host setting to
  see the safe application-to-library handoff.
- Run the focused Go tests, rebuild `goja-auth`, then run the named Playwright
  test above.

### Technical details

```text
TinyIDP callback redirect ?error=access_denied&error_description=...
  -> Goja Auth native /auth/callback handler
  -> closed error kind: canceled
  -> fixed HTML + same-origin /static/styles.css
  -> strict CSP; query values never rendered
  -> Try signing in again | Return to the application
```

## Step 27: Make exhausted email-code recovery durable and test it through Chromium

The email-code screen already had a closed `attempts_exceeded` presentation
code, but the real SQLite-backed Compose instance did not reach it. A wrong
code updated the record in a transaction and then returned its typed error
before committing; the deferred rollback erased the counter. This step makes
the rejection itself a durable security transition and turns the visible
resend button into a real recovery operation instead of a dead end.

A permitted resend continues to preserve the challenge's workflow, browser,
and recipient bindings, but it rotates the code and resets the incorrect-code
counter for that new code generation. The browser test proves the complete
contract: it exhausts five attempts, observes a styled actionable page with no
password field, requests another code, confirms that the original code fails,
and confirms the replacement reaches the password page.

### Prompt Context

**User prompt (verbatim):** "ok, implement that"

**Assistant interpretation:** Implement the previously specified
email-code-exhaustion test and recovery contract, including the durable store
behavior required for it to be truthful in the local production-shaped stack.

**Inferred user intent:** An ordinary verification mistake must not strand a
user on vague copy or create a security bypass; the IDP should provide a
professional, testable recovery route.

**Commit (native contract):** `a41087c` — "fix: recover email verification
after attempt exhaustion"

**Commit (browser regression):** `263603a` — "test: cover email code
exhaustion recovery"

### What I did

- Inspected both `MemoryStore` and the production `sqlitestore.Store` email
  challenge transitions rather than assuming the public error mapping was
  wrong.
- Added a transaction commit for a rejected SQLite verification after saving
  the incremented attempt counter, before returning `ErrConflict` or
  `ErrAttemptsExceeded`.
- Changed both stores so a successful, policy-authorized resend rotates the
  code hash and resets `Attempts` to zero while retaining the durable
  challenge ID, bindings, expiry, and resend count.
- Updated the closed public workflow copy to instruct the user to request a
  new code, rather than incorrectly telling them to restart registration.
- Added deterministic service and SQLite regression tests for exhaustion,
  old-code invalidation, resend, and replacement-code verification.
- Rebuilt only the local `idp` Compose image and ran the named Chromium
  journey against `https://idp.localhost:8443` through Caddy.

### Why

- Incorrect-code limits are security state. They must survive a request that
  returns an error; otherwise every browser POST silently resets the budget.
- The old UI offered `Send another code` after exhaustion, but the stores kept
  the exhausted count. The replacement code therefore could never succeed.
- A code rotation is a clear generation boundary: it invalidates the prior
  secret and gives only the replacement code a fresh, bounded attempt budget.

### What worked

- `go test ./pkg/idpemailchallenge ./pkg/sqlitestore ./pkg/idpui -count=1`
  passed after the transaction and recovery changes.
- `go test ./...` passed.
- The native commit's pre-commit gate passed package tests, `golangci-lint`,
  and both custom `go vet` analyzers.
- `docker compose -f examples/tinyidp-shared-two-apps/compose.yaml up -d
  --build --no-deps idp` rebuilt the service; the resulting health check was
  healthy.
- `pnpm --dir examples/tinyidp-shared-two-apps/browser-tests exec playwright
  test -g 'email-code exhaustion stays themed' --reporter=list` passed: one
  Chromium test in 1.7 seconds.

### What didn't work

- The first focused test run failed with:
  `expected: "email challenge attempts exceeded"; in chain: "email challenge
  conflict"` in `TestSQLiteChallengeResendRestoresAttemptBudgetAndInvalidatesOldCode`.
  This was the diagnostic signal that revealed the SQLite transaction was
  rolling back rejected attempt increments.
- The first test run also showed the expected-copy failure after changing the
  recovery wording. Updating `TestWorkflowEmailCodeFailureHasSpecificPublicCopy`
  made the public copy contract explicit.

### What I learned

- A typed error return does not make a state transition durable. With a
  transaction-scoped store, the commit point must be placed before a rejected
  result is returned whenever the rejection updates security state.
- The `attempts` count belongs to the delivered-code generation, not to the
  overall workflow. The workflow-level controls that remain monotonic are the
  challenge bindings, expiry, and resend budget.

### What was tricky to build

- The original symptom looked like a presentation-taxonomy failure because
  Chromium displayed generic `rejected` copy. The provider already mapped
  `ErrAttemptsExceeded` correctly; the storage layer never emitted that error
  in the deployed SQLite path. The focused SQLite regression was necessary to
  identify the actual boundary.
- The browser assertion has to follow newly rendered forms and retrieve the
  latest Mailpit message. It deliberately asserts a blank code field after
  every rejection so secrets do not accidentally become redisplayed as part of
  improving recovery UX.

### What warrants a second pair of eyes

- Review the chosen generation semantics: a resend after exhaustion consumes
  one of the finite resend budget and resets only `Attempts`; it does not
  extend expiry or relax browser/workflow binding checks.
- Confirm deployment operators want the existing script-configured limits
  (five attempts and two resends) for their risk model. This change does not
  alter those policy values.

### What should be done in the future

- Finish the remaining browser matrix rows, especially password mismatch,
  all invitation states, and the separately paused Chromium happy-path
  approval-navigation investigation.
- Run the whole Playwright matrix on fresh and retained Compose volumes once
  the remaining rows are complete.

### Code review instructions

- Start at `Store.VerifyEmailChallenge` and `Store.ResendEmailChallenge` in
  `pkg/sqlitestore/email_challenge.go`; verify the rejected-attempt commit
  precedes its typed error and resend resets only `Attempts`.
- Compare `pkg/idpemailchallenge/memory.go` to ensure the test/development
  store has the same contract.
- Run `go test ./pkg/idpemailchallenge ./pkg/sqlitestore ./pkg/idpui -count=1`.
- Rebuild the `idp` service, then run the named Playwright command above.

### Technical details

```text
wrong code (SQLite)
  -> Attempts += 1
  -> save + COMMIT
  -> ErrConflict | ErrAttemptsExceeded

permitted resend
  -> fresh random code -> new HMAC hash
  -> Attempts = 0; Resends += 1; old hash is unreachable
  -> send the new code

old code -> rejected; replacement code -> verified evidence -> password page
```

## Step 28: Preserve the relying-party origin across signup and consent

The complete new-account journey stopped on an otherwise valid consent page.
The browser trace showed that account creation, the TinyIDP session cookie,
and the consent interaction were all successful. Chromium then refused the
Approve submission because the consent document's CSP allowed only a
same-origin form action, while CSP evaluates the entire resulting redirect
chain and the authorization response terminates at Message Desk.

The signup completion path constructed the consent presentation with a nil
request even though the newly stored interaction retained the validated
canonical OAuth request. The fix passes that canonical request into the same
bounded presentation builder used by other consent paths. The builder extracts
only the already-validated callback origin for `form-action`; it does not expose
the redirect URI or make browser input authoritative.

### Prompt Context

**User prompt (verbatim):** "Go ahead, figure it out and fix it."

**Assistant interpretation:** Diagnose the committed full-signup browser
failure, repair its root cause, and prove that a new account can approve
consent and reach Message Desk.

**Inferred user intent:** Restore a complete production-shaped signup path so
the browser matrix tests real product behavior rather than stopping at the
last authorization step.

**Commit (code):** `cfc1d08` — "fix: allow post-signup consent redirect"

### What I did

- Reproduced the committed Playwright journey and retained its screenshot,
  video, and trace.
- Read the trace's final request and console event. Chromium reported:
  `Refused to send form data to 'https://idp.localhost:8443/authorize'
  because it violates ... "form-action 'self'".`
- Confirmed that the prior password POST created the account and returned both
  `tinyidp_session` and `tinyidp_csrf` cookies.
- Traced the consent page to `completeScriptedSignup`, where
  `newInteractionPage` received a nil canonical request.
- Passed `record.CanonicalRequest` into the consent presentation builder so it
  can derive the validated Message Desk origin for the response CSP.
- Extended the email-verified signup provider test to require stored consent,
  assert the exact callback-aware CSP, submit approval, and observe the OAuth
  redirect.
- Rebuilt the local IDP container and reran the full Chromium signup journey.

### Why

- `form-action` constrains redirects caused by form submission as well as the
  initial form target. A same-origin `/authorize` POST can therefore be blocked
  when it returns the expected cross-origin RP callback redirect.
- The RP callback origin was already authenticated by client registration and
  persisted in the canonical interaction. Reusing it preserves the security
  model; inventing an origin from the current POST would not.

### What worked

- `go test ./internal/fositeadapter -run
  TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification
  -count=1` passed.
- The pre-commit package, lint, custom analyzer, and vet gates passed.
- The rebuilt IDP became healthy.
- The focused Playwright journey passed in Chromium and reached signed-in
  Message Desk in 2.6 seconds.

### What didn't work

- The first test edit changed the consent policy in a different signup test
  because several provider fixtures used the same option line. The focused
  run still returned `303` immediately and reported
  `verified signup consent status=303 body=`. The unrelated fixture change was
  reverted, and the intended email-verified fixture was updated using its
  surrounding configuration as context.

### What I learned

- The `net::ERR_ABORTED` network entry was a browser policy decision, not a
  failed TinyIDP request. The decisive evidence was the associated console
  event in the Playwright trace.
- A presentation field can be security-critical without being authorization
  authority. `RedirectOrigin` carries a reduced, validated value solely to
  compile an accurate CSP.

### What was tricky to build

- The blocked POST appeared in the network archive with no cookies and no
  response, which initially resembled a session or CSRF handoff bug. The cookie
  snapshot immediately before the click proved both cookies existed; the
  console trace then identified CSP as the actual enforcement layer.
- The policy must include only the canonical origin, not the complete callback
  URI. `newInteractionPage` already enforces this reduction through
  `interactionRedirectOrigin`, so the fix reuses that established seam.

### What warrants a second pair of eyes

- Review that every interaction page capable of ending in an OAuth redirect is
  constructed with the stored canonical request. The signup and
  account-selection consent transitions now follow the same pattern.
- Confirm the exact-CSP regression remains intentional if future directives are
  added; it currently protects both the RP origin and the otherwise restrictive
  policy.

### What should be done in the future

- Finish the remaining validation rows and run the complete matrix against
  fresh and retained Compose state.

### Code review instructions

- Start at `completeScriptedSignup` in
  `internal/fositeadapter/scripted_signup.go` and compare it with the
  account-selection consent transition in `provider.go`.
- Review the final section of
  `TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification`.
- Run the focused Go and Playwright commands listed above.

### Technical details

```text
password commit
  -> new TinyIDP session
  -> new consent interaction stores canonical OAuth request
  -> consent page derives https://message.localhost:8443
  -> CSP form-action 'self' https://message.localhost:8443
  -> Approve POST /authorize
  -> 303 Message Desk callback
  -> signed-in application
```

## Step 29: Separate browser throttling UX from the exhaustive local test budget

The retained-stack matrix initially appeared to expose six unrelated failures,
but every artifact contained the same plain-text `rate limited` response. The
default production limiter permits 30 authorization submissions per address
and minute, while the sequential matrix deliberately performs substantially
more submissions from one Compose proxy address.

The provider now renders address-wide and account-aware authorization
throttling through the terminal browser-error contract. Production throttling
remains enabled. The local exhaustive stack raises its finite budget to 500 so
the matrix reaches the validation behavior under test instead of accidentally
becoming a rate-limiter load test.

### Prompt Context

**User prompt (verbatim):** "do it"

**Assistant interpretation:** Continue the agreed browser-validation work,
diagnose the full-matrix failures, fix real UX defects, and finish validation.

**Inferred user intent:** Make the complete local authentication experience
reliably testable while preserving production-safe security behavior.

**Commit (code):** `595742b` — "fix: render browser rate limits safely"

### What I did

- Ran all 21 retained-stack Playwright scenarios and inspected each failure
  artifact rather than treating the first downstream symptom as independent.
- Identified `authorize:<client-address>` as the exhausted shared limiter
  namespace.
- Added a provider-owned `renderRateLimited` terminal error document that does
  not reflect an untrusted client identifier at the early address boundary.
- Routed both authorization address throttling and per-login throttling through
  the safe HTML renderer.
- Added a focused renderer regression for status, media type, cache policy, and
  public copy.
- Set `--rate-limit=500` only in the local two-application Compose command.

### Why

- A security control should remain visible as a coherent authentication page;
  raw provider text is not an acceptable browser failure mode.
- The local matrix uses one network source for many intentionally invalid
  requests. Its workload is qualitatively different from a human production
  session, so it needs a documented test budget without disabling throttling.

### What worked

- `go test ./internal/fositeadapter ./internal/productionui ./pkg/idpui
  -count=1` passed.
- The pre-commit package tests, golangci-lint, glazed analyzer, custom UI
  analyzer, and vet gates passed.
- The browser-error response is a 429 HTML document with `no-store` and
  `no-cache` and contains no raw `rate limited` provider text.

### What didn't work

- The first focused Go run failed to compile with
  `internal/fositeadapter/rendering.go:169:5: undefined: strings` because the
  new normalization guard lacked its import. Adding the import resolved the
  failure; the repeated focused run passed.
- The first complete browser run could not exercise its last six scenarios
  because all were masked by the shared limiter.

### What I learned

- The browser matrix itself is a form of state: retained limiter windows can
  affect later cases even when database and browser contexts are isolated.
- Early address throttling occurs before a client record can be trusted. A
  provider-owned presentation context is required there; a submitted
  `client_id` must not select arbitrary themed content.

### What was tricky to build

- Raising a test limit alone would hide the raw 429 defect, while styling the
  429 alone would still prevent later tests from reaching their intended
  branches. Both changes are necessary and serve different purposes.
- A truthful `Retry-After` value was intentionally not invented because the
  renderer does not own or expose the configured limiter window.

### What warrants a second pair of eyes

- Review whether device-verification browser throttling should adopt the same
  terminal renderer in its own scoped follow-up; this change is limited to the
  `/authorize` matrix.
- Confirm 500 remains comfortably above the exhaustive local suite while still
  catching accidental request loops.

### What should be done in the future

- Rebuild the local IDP, complete password mismatch and invitation validation
  coverage, and rerun retained and fresh-stack matrices.

### Code review instructions

- Start with `renderRateLimited` in `internal/fositeadapter/rendering.go` and
  its two call sites in `provider.go`.
- Review the local-only `--rate-limit=500` in the Compose service command.
- Run `go test ./internal/fositeadapter ./internal/productionui ./pkg/idpui
  -count=1`.

### Technical details

```text
production/default: 30 requests / configured window / limiter key
local matrix:       500 requests / configured window / limiter key

early address rejection -> provider-owned "tinyidp" presentation context
known login rejection   -> validated client presentation context
```

## Step 30: Complete password-mismatch and invitation-policy coverage

The native signup commit already refused unequal passwords, but the rejection
occurred after the JavaScript handler and returned the generic password-policy
copy. TinyIDP now compares the two secret capabilities immediately after the
active form is parsed, clears both resolved byte slices, and renders the closed
`mismatch` error on the confirmation field without invoking JavaScript.

The browser matrix now exercises missing, unknown, expired, and wrong-audience
signup invitations against Goja Auth and requires identical public field copy.
This complements the existing real-stack acceptance flow for valid consumption
and consumed-code replay and the lower-level revoked-state provider matrix.

### Prompt Context

**User prompt (verbatim):** (same as Step 29)

**Assistant interpretation:** Finish the remaining signup-validation rows and
make their browser behavior specific, safe, and testable.

**Inferred user intent:** Ensure ordinary validation and security-policy
failures remain within a polished themed workflow instead of producing vague
or terminal errors.

**Commit (code):** `647d540` — "test: complete signup validation matrix"

### What I did

- Added native cross-field password comparison before signup lambda invocation.
- Attached `ErrorMismatch` to `passwordConfirmation` and preserved the existing
  secret-destruction lifecycle.
- Strengthened the provider integration test to assert exact mismatch copy and
  `aria-invalid` on the confirmation input.
- Added a real Chromium password-mismatch journey that verifies both password
  fields return blank and the Message Desk theme remains loaded.
- Added Goja invitation cases for native required validation, expired durable
  evidence, and wrong audience in addition to the existing unknown-code case.
- Added a test-only helper that issues opaque invitations through the actual
  administrative CLI inside the running IDP container.

### Why

- Password equality is a native field relationship, not application policy.
  JavaScript should receive only structurally valid submissions.
- Invitation failure classes intentionally collapse to one public response,
  but each important policy boundary still needs an executable regression.

### What worked

- `TestScriptedSignupDoesNotRequireLegacyRegistrationOption` passed with the
  new field-specific mismatch assertions.
- Playwright collected all 25 tests after the helper's module path was fixed.
- The pre-commit package, lint, analyzer, and vet gates passed.

### What didn't work

- The first `playwright test --list` failed with
  `ReferenceError: require is not defined in ES module scope` at the Node
  imports. `import.meta.url` forced mixed ESM/CommonJS evaluation in this
  package. Replacing it with `resolve(process.cwd(), "../compose.yaml")`
  matched the documented pnpm working directory and collection then passed.

### What I learned

- The test suite's invitation coverage is deliberately layered: Playwright
  proves browser presentation for representative durable-policy classes, the
  HTTP acceptance script proves one-time consume and replay over the real
  stack, and package tests exhaust internal state classifications.
- Resolving a secret handle returns a byte slice that must be cleared even for
  validation-only comparisons; `submission.DestroySecrets` remains a second
  lifecycle safeguard for the owned handles.

### What was tricky to build

- Invitation codes are one-time operator secrets. The helper must parse the
  administrative JSON only in memory and must not print or copy codes into the
  diary or committed fixtures.
- Cross-field validation must happen after deny and resend actions, which are
  explicitly allowed to bypass ordinary form validation, but before any
  lambda capability is invoked.

### What warrants a second pair of eyes

- Confirm `submissionPasswordsMatch` is correctly ordered before JavaScript
  while the native commit retains its independent equality revalidation.
- Review that browser artifacts remain gitignored because retained traces for
  failed invitation tests can contain the ephemeral input value.

### What should be done in the future

- Run the focused new browser cases, then the complete retained and fresh-stack
  matrices and update the defect ledger with final evidence.

### Code review instructions

- Start in `resumeScriptedSignup` immediately after the deny/resend branches.
- Review the invitation cases and `issueSignupInvitation` helper in
  `authentication-ux.spec.ts`.
- Run the focused Go test and `pnpm --dir
  examples/tinyidp-shared-two-apps/browser-tests exec playwright test --list`.

### Technical details

```text
password form POST
  -> parse secret handles
  -> deny/resend bypass (when applicable)
  -> resolve password + confirmation
  -> constant-time equality check
  -> clear resolved copies
  -> mismatch: themed 400 on confirmation field
  -> match: invoke signup lambda, then independently revalidate at commit
```

## Step 31: Prove retained and fresh stacks and reserve the proxy address

All 25 Chromium scenarios passed against the retained local stack. A subsequent
application-volume reset uncovered that the documented container-side
`idp.localhost` resolution depended on Caddy receiving `172.31.0.2`, but the
Compose network did not reserve that address. Mailpit could receive `.2` first
during concurrent startup, leaving TinyIDP's TLS readiness probe unable to
reach the proxy.

Caddy now receives an explicit `.2` endpoint and dynamic IDP-backend addresses
come only from `.128/25`. After recreating the network, Caddy owned `.2`, the
one-shot CA and application bootstrap jobs exited successfully, all long-lived
services reached their intended states, and the fresh-stack suite passed 25 of
25 scenarios.

### Prompt Context

**User prompt (verbatim):** (same as Step 29)

**Assistant interpretation:** Finish validation to completion, including both
retained and truly recreated local application state, and repair discovered
defects.

**Inferred user intent:** Leave a reproducible local system that another
implementer can reset and validate without relying on historical Docker
allocation order.

**Commit (code):** `69f3283` — "fix: make fresh local topology deterministic"

### What I did

- Ran the complete 25-test matrix against retained volumes: 25 passed in 33.2
  seconds.
- Deleted only project-owned application volumes with `docker compose down
  -v`; the external protected CA volume was retained by design.
- Reinitialized local secrets and recreated the complete service graph.
- Diagnosed failed IDP readiness through Docker health history and network
  endpoint inspection.
- Assigned Caddy `172.31.0.2` and confined dynamic IDP-backend allocation to
  `172.31.0.128/25`.
- Recreated the network, verified endpoint addresses and service/bootstrap
  states, and reran the complete matrix: 25 passed in 54.3 seconds.
- Updated the operator README, defect ledger, final evidence, and task states.

### Why

- A fresh-stack proof must validate orchestration assumptions, not merely
  application behavior against already allocated networks and initialized
  databases.
- Container TLS clients intentionally resolve the public issuer through Caddy.
  Therefore its backend address is a real declared topology contract and must
  not be left to dynamic allocation order.

### What worked

- The retained-state suite passed every validation, session, theme, callback,
  replay, and signup scenario.
- After the IPAM correction, Docker reported Caddy at `172.31.0.2` and dynamic
  services at `.129` through `.132`.
- `ca-export` and `goja-bootstrap` exited with status 0; TinyIDP, Mailpit,
  PostgreSQL, and Message Desk were healthy.
- The fresh-state suite passed all 25 Chromium cases.

### What didn't work

- The first fresh startup left TinyIDP in `health: starting`. Docker recorded:
  `curl: (7) Failed to connect to idp.localhost port 8443 after 0 ms: Couldn't
  connect to server`. Inspection showed Mailpit at `172.31.0.2` and Caddy at
  `172.31.0.3`.
- Assigning only a static `.2` address to Caddy was insufficient because
  independent services start concurrently. Mailpit dynamically claimed `.2`
  before Caddy attached, and Docker returned `Error response from daemon:
  Address already in use`.
- One validation command used a repository-relative Compose path after its
  working directory had already changed into the example and returned `stat
  .../examples/tinyidp-shared-two-apps/examples/tinyidp-shared-two-apps/compose.yaml:
  no such file or directory`. Reissuing it as `docker compose -f compose.yaml
  config --quiet` succeeded.

### What I learned

- Docker's dynamic allocator does not reserve a service's requested static
  address before that endpoint is attached. A disjoint `ip_range` is required
  when concurrent dynamic endpoints share the subnet.
- Retained-stack and fresh-stack runs answer different questions. Both are
  required for a stateful authentication topology.

### What was tricky to build

- The TLS topology uses the public hostname inside containers on purpose, so
  replacing the health check with a direct `idp:8081` request would have
  bypassed the listener, certificate, proxy headers, and issuer path under
  test. The correction had to preserve the proxy traversal.
- The subnet must contain both the fixed `.2` address and the dynamic range,
  while TinyIDP continues trusting the complete explicitly private `/24` as
  its proxy network.

### What warrants a second pair of eyes

- Review whether future fixed infrastructure endpoints need addresses below
  `.128`; that range is now intentionally excluded from dynamic allocation.
- Confirm the external CA volume remains outside `docker compose down -v` in
  future Compose changes.

### What should be done in the future

- N/A for the scoped browser matrix. The published guide should be refreshed
  on reMarkable after final ticket bookkeeping.

### Code review instructions

- Review the proxy service network mapping and IDP-backend IPAM config in
  `compose.yaml` together with every `.2` resolution site.
- Run `docker compose config --quiet`, recreate the project network, inspect
  its endpoints, then run the complete Playwright command.

### Technical details

```text
idp-backend subnet:       172.31.0.0/24
Caddy fixed endpoint:     172.31.0.2
dynamic allocation pool: 172.31.0.128/25
TinyIDP trusted proxies:  172.31.0.0/24

container TLS client
  -> idp.localhost:8443 resolves to 172.31.0.2
  -> Caddy terminates TLS
  -> trusted-proxy HTTP to TinyIDP
```
