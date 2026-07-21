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
    - Path: repo://examples/tinyidp-message-app/ui/src/App.tsx
      Note: Separated account action navigation (commit 4b15802)
    - Path: repo://examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts
      Note: |-
        Playwright journeys (commit 34959ea)
        Playwright duplicate-email regression (commit 21456f9)
    - Path: repo://examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py
      Note: Live HTTPS rejection validation (commit 0ce1fa6)
    - Path: repo://internal/fositeadapter/provider.go
      Note: Null-origin guard and registration error routing (commits 924b575 and dffc6c4)
    - Path: repo://internal/fositeadapter/registration_test.go
      Note: |-
        Active-session second-signup regression coverage (commit 1a15439)
        Provider regression (commit c7a2cb7)
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Terminal browser error response boundary (commit dffc6c4)
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: |-
        Remembered-session continuation fix (commit c7a2cb7)
        Duplicate commit mapping (commit 21456f9)
    - Path: repo://internal/productionui/renderer.go
      Note: Per-client error theme selection (commit dffc6c4)
    - Path: repo://pkg/idpui/browser_error.go
      Note: Bounded terminal error model (commit dffc6c4)
    - Path: repo://pkg/idpui/workflow.go
      Note: Global duplicate-identity error model (commit 21456f9)
    - Path: repo://pkg/idpworkflow/descriptors.go
      Note: Production-aligned public password bounds (commit 2c136ee)
    - Path: repo://pkg/idpworkflow/submission.go
      Note: Native secret-field bounds enforcement (commit 2c136ee)
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

**User prompt (verbatim):** "Create a design document for this task, Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.\n\n\nthen work on it and get it up and running"

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

**User prompt (verbatim):** "https://message.localhost:8443/auth/callback?error=invalid_request&error_description=The+request+is+missing+a+required+parameter%2C+includes+an+invalid+parameter+value%2C+includes+a+parameter+more+than+once%2C+or+is+otherwise+malformed.+registration+requires+a+new+browser+session&state=c-Cub8zOZIkcGz9pJht0uSynae4A41daZ78fhR7yop8\n\nidentity login was not accepted\n\n---\n\nThis was after logging out of message-desk (not idp) and clicking \"sign up for a new account\""

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
