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
    - Path: repo://examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py
      Note: Live HTTPS rejection validation (commit 0ce1fa6)
    - Path: repo://internal/fositeadapter/provider.go
      Note: Null-origin guard and registration error routing (commits 924b575 and dffc6c4)
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Terminal browser error response boundary (commit dffc6c4)
    - Path: repo://internal/productionui/renderer.go
      Note: Per-client error theme selection (commit dffc6c4)
    - Path: repo://pkg/idpui/browser_error.go
      Note: Bounded terminal error model (commit dffc6c4)
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

## Goal

<!-- What is the purpose of this reference document? -->

## Context

<!-- Provide background context needed to use this reference -->

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
