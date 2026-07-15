---
Title: Implementation diary
Ticket: TINYIDP-EXTERNAL-DEMO-001
Status: active
Topics:
    - architecture
    - go
    - identity
    - oidc
    - oauth2
    - docker
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-external-message-desk/compose.yaml
      Note: Compose deployment work
    - Path: repo://examples/tinyidp-external-message-desk/docker-entrypoint.sh
      Note: Container state ownership evidence
    - Path: repo://examples/tinyidp-message-app/external_runtime.go
      Note: External RP boundary
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Callback-origin CSP evidence
    - Path: repo://ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/01-compose-health-smoke.sh
      Note: Reproducible Compose health check
    - Path: repo://ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/02-external-demo.spec.mjs
      Note: Committed two-origin browser flow
    - Path: repo://ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/03-compose-durability-and-secret-check.sh
      Note: Restart and development-fixture exposure assurance
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T21:58:07.711626152-04:00
WhatFor: ""
WhenToUse: ""
---



# Implementation diary

## Goal

Record the design investigation for a standalone tiny-idp and Message Desk
Docker reference deployment, including its trust boundaries and implementation
sequence.

## Step 1: Establish the external-issuer reference architecture

The requested demo is not a cosmetic repackaging of the embedded example. It
must prove that Message Desk is a normal OIDC relying party when tiny-idp is a
separate process and origin. The design therefore preserves the browser and
back-channel protocol mechanics while removing all direct provider and account
service dependencies from the application container.

### Prompt Context

**User prompt (verbatim):** "Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create an intern-ready design package for a
two-container standalone tiny-idp and Message Desk demo and deliver it through
the ticket and reMarkable.

**Inferred user intent:** Provide a reusable, honest reference deployment for
external tiny-idp integration rather than requiring every application to embed
the provider binary.

### What I did

- Created `TINYIDP-EXTERNAL-DEMO-001`, its design document, diary, and seven
  implementation phases.
- Mapped current embedded seams in `commands.go`, `oidc_client.go`, and
  `app_http.go`.
- Wrote the two-origin topology, OIDC flow, configuration contracts, visual
  renderer boundary, seeded-account policy, logout scopes, validation plan,
  decisions, and intern onboarding checklist.

### Why

- A separate provider is useful only when its database, cookies, UI, and
  lifecycle remain provider-owned. Sharing embedded internals would hide the
  deployment boundary the example is intended to teach.

### What worked

- The existing Message Desk client already accepts an arbitrary issuer and
  `*http.Client`; its PKCE/state/nonce/ID-token verification path is reusable.

### What didn't work

- N/A. No implementation experiment was run in this design step.

### What I learned

- The essential new work is composition and operational validation, not a new
  OAuth client implementation. Registration is the one deliberate functional
  omission because it currently depends on direct embedded account APIs.

### What was tricky to build

- The browser needs public canonical origins while containers use private DNS.
  These must not be confused: issuer discovery, redirects, and token `iss`
  checks use public URLs; Docker service names are only internal transport
  addresses.

### What warrants a second pair of eyes

- Review development HTTP versus production HTTPS/cookie policy and the seed
  credential lifecycle before implementation begins.

### What should be done in the future

- Implement phases 1–7 in order; create a separate ticket if a public
  tiny-idp self-registration API is desired.

### Code review instructions

- Start with `examples/tinyidp-message-app/oidc_client.go` and compare its
  transport injection to the proposed standard HTTPS client.

### Technical details

```text
Message Desk owns: app SQLite, app session, PKCE/state/nonce, messages.
tiny-idp owns: identity SQLite, accounts, IdP cookies, login/consent/chooser.
```

## Step 2: Establish the external relying-party configuration boundary

The first implementation slice adds a standalone configuration type before any
container or handler change. An external deployment is valid only when the
Message Desk public origin and issuer are distinct, the issuer is canonical,
the client identifier is present, and cookie security agrees with the public
URL scheme. This makes an accidentally co-hosted or insecure configuration
fail at startup rather than behaving ambiguously at callback time.

### Prompt Context

**User prompt (verbatim):** "ok, now implement them, task by task, committing at appropriate intervals, keeping a detailed diary as you work"

**Assistant interpretation:** Execute the detailed implementation plan in
small verified commits while maintaining a continuation-ready diary.

**Inferred user intent:** Build the external IdP demonstration from explicit,
testable security contracts rather than an unstructured Docker prototype.

### What I did

- Added `externalOIDCConfig` and canonical external issuer normalization in
  `examples/tinyidp-message-app/external_config.go`.
- Added table-style negative tests for same-origin, missing-client, cookie,
  cross-origin logout endpoint, query, credentials, and noncanonical path
  configurations.
- Ran `gofmt` and the focused configuration test command.

### Why

- The embedded app derives issuer and logout paths from one origin. That is
  unsafe as an implicit assumption for independent services, so the external
  contract must name and validate those values before runtime composition.

### What worked

- `go test ./examples/tinyidp-message-app -run 'TestExternalOIDCConfigValidation|TestNormalizeExternalIssuer' -count=1` passed.

### What didn't work

- N/A.

### What I learned

- The existing `normalizePublicBaseURL` already encodes the development
  loopback-only HTTP policy. Reusing it makes external issuer validation obey
  the same policy without accepting arbitrary plaintext network origins.

### What was tricky to build

- An issuer may include a path, but an RP public origin may not. The
  normalization therefore validates the origin with the existing helper and
  preserves only one canonical issuer path suffix.

### What warrants a second pair of eyes

- Review whether the final production configuration should discover the
  end-session endpoint exclusively or allow the validated explicit override
  introduced here for operational compatibility.

### What should be done in the future

- Implement Phase 1.2 next: the IdP seed/client manifest and idempotent
  bootstrap command.

### Code review instructions

- Read `externalOIDCConfig.validate` beside `normalizePublicBaseURL`, then run
  the focused test command above.

### Technical details

```text
public origin != issuer
issuer and end-session endpoint share origin
https public origin <=> Secure app cookie
```

## Step 3: Implement idempotent standalone client and account seeding

The external demo needs a reproducible identity state without exposing its
database to Message Desk. `SeedManifest` now bootstraps the public browser
client and reconciles seed accounts exclusively through `embeddedidp.Bootstrap`
and `idpaccounts.Service`. A repeat run verifies persisted identity fields and
passwords; it never silently changes a client or account.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Implement the next planned bootstrap task and
record its security behavior and validation result.

**Inferred user intent:** Make the Docker demo restartable and safe to run
repeatedly while retaining clear identity-state ownership.

### What I did

- Added `examples/tinyidp-external-message-desk/idp_seed.go` with
  `SeedManifest`, `SeedAccount`, bootstrap, and duplicate reconciliation.
- Added a memory-store test proving repeat bootstrap succeeds and account
  identity drift fails.
- Corrected a compile failure by replacing unsupported `%w` use in
  `pkg/errors.Errorf` with `errors.Wrapf`.

### Why

- A seed process must make desired state explicit but must never act as a
  password-reset or account-mutation mechanism on every container start.

### What worked

- `go test ./examples/tinyidp-external-message-desk -count=1` passed.

### What didn't work

- The first compile failed with `github.com/pkg/errors.Errorf does not support
  error-wrapping directive %w`; using `errors.Wrapf` corrected the wrapping.

### What I learned

- Existing public bootstrap and account APIs are sufficient for standalone
  seeding; no direct SQLite access or internal package import is required.

### What was tricky to build

- Duplicate account creation alone is not idempotence. The reconciler checks
  ID, subject, email, name, and password authentication so a changed fixture
  fails instead of silently diverging from durable state.

### What warrants a second pair of eyes

- The future command must obtain seed passwords from a development-only secret
  file or operator input and must avoid storing them in image layers or logs.

### What should be done in the future

- Implement Phase 1.3 and then expose this seeder through the standalone IdP
  container command.

### Code review instructions

- Review `SeedManifest.Bootstrap` and run the package test command above.

### Technical details

```text
missing account -> create via idpaccounts.Service
duplicate account -> compare identity fields + authenticate configured password
any mismatch -> startup failure
```

## Step 4: Compose a standalone provider from public APIs

`NewStandaloneIDP` now composes a provider process from a store, account
service, seed manifest, token secret, renderer, and explicit cookie mode. It
does not return the store to the application and it does not require Message
Desk imports. This is the service boundary Docker will run.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing the independent provider
service after completing the seeding contract.

**Inferred user intent:** Make the identity service independently runnable
while preserving the same login, consent, chooser, and logout capabilities.

### What I did

- Added `NewStandaloneIDP` with public embedded/provider/account APIs.
- Added standalone issuer normalization because command-private Message Desk
  helpers are not an independent-service API.
- Compiled and tested the standalone package.

### Why

- Docker is a deployment wrapper, not an authorization architecture. The
  provider constructor needs a direct testable boundary before a container
  command can safely call it.

### What worked

- `go test ./examples/tinyidp-external-message-desk -count=1` passed.

### What didn't work

- The first compile referenced the Message Desk private issuer helper. The
  standalone package now owns equivalent canonical validation instead.

### What I learned

- The public packages are sufficient to construct an independent provider;
  the only remaining work is process configuration, durable storage opening,
  and wiring the external RP/container topology.

### What was tricky to build

- An independent package cannot import a `main` package's private helpers.
  Duplicating a small validation rule is preferable to coupling the provider
  service to an application command package.

### What warrants a second pair of eyes

- Consolidate issuer normalization into a public shared package only if more
  than this demo needs it; do not expose an API prematurely.

### What should be done in the future

- Add the standalone command, SQLite lifecycle, renderer asset package, and
  Compose files, then run the two-origin browser flow.

### Code review instructions

- Review `NewStandaloneIDP` and run the standalone package test.

### Technical details

```text
store + accounts + seed + token secret + renderer
  -> embeddedidp.New
  -> independent HTTP provider handler
```

## Step 5: Add the standalone tiny-idp process

The external demo now has a separate `cmd/idp` process. It opens only the
provider SQLite database, loads an operator-mounted seed manifest, retains a
durable owner-only token secret, composes the standalone provider, and serves
the provider handler with health and readiness endpoints. Message Desk is not
linked into this process.

### Prompt Context

**User prompt (verbatim):** "ok, build the whole ticket, don't stop, i'm going out for a swim and i want you to be done and tested when I come back (use playwright MCP to test it all)"

**Assistant interpretation:** Continue through service, container, application,
and browser assurance work without pausing at the design stage.

**Inferred user intent:** Return to a runnable, independently deployed demo
with evidence rather than only an architectural proposal.

### What I did

- Added `examples/tinyidp-external-message-desk/cmd/idp/main.go`.
- Added explicit flags for state root, issuer, listen address, seed file, and
  log level; no environment-variable configuration is required.
- Added separate SQLite/token-secret lifecycle and `/healthz`/`/readyz`.
- Fixed token-secret handling so permission or I/O failure cannot regenerate a
  secret; only an absent file permits initial creation.

### Why

- Separate containers require separate process ownership. The provider must
  persist its own state and expose ordinary HTTP endpoints before an external
  relying party can use it.

### What worked

- `go test ./examples/tinyidp-external-message-desk/... -count=1` compiled
  both the seed package and standalone command.

### What didn't work

- N/A.

### What I learned

- The standalone command needs no private tiny-idp implementation imports;
  public composition, account, and SQLite APIs are sufficient.

### What was tricky to build

- Secret initialization must distinguish missing-file initialization from an
  unreadable existing secret. Treating both as creation would break signing and
  session continuity after a filesystem fault.

### What warrants a second pair of eyes

- The first command intentionally supports development HTTP only. The
  production TLS/reverse-proxy profile remains a later task and must not be
  represented as ready by this binary.

### What should be done in the future

- Add the external relying-party process and Compose wiring, then run a live
  two-origin browser test.

### Code review instructions

- Inspect `cmd/idp/main.go`, then run the package test command above.

### Technical details

```text
idp container: seed file + idp SQLite + token key -> provider HTTP
app container: separate app SQLite -> ordinary OIDC client HTTP
```

## Step 6: Make Message Desk a real external relying party

Message Desk can now run without an embedded provider handler, identity SQLite
store, or account service. External mode opens only its application database,
performs OIDC discovery and verification against the configured issuer, and
uses the normal browser authorization-code flow with PKCE, nonce, and a local
opaque application session.

The live browser test uncovered a broader provider correctness issue: an
interaction page may POST to the provider and receive a redirect to the
validated RP callback. Chromium applies `form-action` to that terminal
redirect as well. A CSP that listed only the provider therefore blocked a
correct OIDC completion. The fix allows only the callback *origin* already
validated by the provider for that particular interaction; it does not allow
arbitrary form targets or callback URLs.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Continue from the standalone provider into an
independent relying-party runtime and prove a real browser authorization flow.

**Inferred user intent:** Ensure the demo's process split is a genuine security
boundary rather than merely two different server routes in one binary.

**Commit (code):** a15f51a — "feat(message-app): add external issuer runtime mode"; 09b556d — "fix(idpui): allow canonical form action in CSP"; 911aa11 — "fix(idpui): permit validated callback in form CSP"

### What I did

- Added `--external-issuer` and external runtime composition that opens only
  Message Desk state.
- Kept OIDC discovery, token exchange, JWKS lookup, and ID-token verification
  on one explicit HTTP client.
- Used Playwright to exercise `8080 -> 8081/authorize -> 8080/auth/callback`.
- Diagnosed Chromium's exact error: `Refused to send form data ... because it
  violates ... form-action 'self'`.
- Added `InteractionForm.RedirectOrigin` and render-time CSP specialization;
  the value originates from an already verified authorization request.

### Why

- The authorization response is a navigation across origins. CSP needs to
  preserve that protocol transition while retaining a deny-by-default policy.

### What worked

- `go test ./internal/fositeadapter ./pkg/idpui ./examples/tinyidp-message-app -count=1` passed.
- Playwright selected a remembered `Amelie` account and returned to an
  authenticated Message Desk session at `http://127.0.0.1:8080/`.

### What didn't work

- A first CSP adjustment that allowed only the canonical IdP action URL still
  failed. The browser followed the 303 authorization response to the RP and
  applied `form-action` to the final callback origin.

### What I learned

- Form-action is relevant to the whole form-navigation chain, not merely the
  literal HTML `action` attribute. OIDC login pages need a callback-aware CSP
  derived only from server-validated client registration state.

### What was tricky to build

- The rendered page needs policy data but the renderer must not decide protocol
  state. `newInteractionPage` derives the canonical redirect origin from the
  Fosite-validated request; `renderInteraction` only serializes the CSP. This
  preserves the separation between protocol ownership and host presentation.

### What warrants a second pair of eyes

- Review all future interaction render paths to ensure they pass canonical
  stored request data, not raw browser parameters, to `newInteractionPage`.

### What should be done in the future

- Add an integration assertion that follows the final 303 in a browser engine
  so this CSP/OIDC interaction cannot regress.

### Code review instructions

- Start with `internal/fositeadapter/rendering.go`, then review
  `internal/fositeadapter/provider.go` and `pkg/idpui/types.go`.
- Run the focused Go test command above and perform a browser sign-in against
  distinct local origins.

### Technical details

```text
interaction document CSP:
  form-action 'self' <registered callback origin>

POST /authorize -> 303 -> RP /auth/callback
```

## Step 7: Separate provider-owned provisioning from the external RP UI

The external demo now presents a truthful guest experience. It does not render
or expose Message Desk self-registration because account provisioning belongs
to the standalone provider. The provider mounts the same constrained
Message-Desk interaction renderer and its stylesheet, so chooser, login,
consent, error, and logout-facing pages share the intended retro visual system
without transferring authorization decisions into the application.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Finish the user-visible two-origin separation
and make the provider UI consistent with the application visual language.

**Inferred user intent:** A user should understand which service owns an
account and should never be offered a broken or misleading local workflow.

**Commit (code):** 14e0c4a — "feat(external-demo): separate seeded identity from registration"

### What I did

- Mounted `loginui.Renderer` and `/static/tinyidp/` in the standalone IdP
  command.
- Added an explicit `registrationEnabled` capability to `/api/session`.
- Disabled and returned 404 for registration endpoints in external runtime.
- Rebuilt the committed React asset and replaced the guest registration form
  with an operator-seeded-account explanation.
- Added `TestExternalModeDoesNotExposeSelfRegistration`.

### Why

- “Unavailable” registration endpoints still encourage users to place account
  data in the wrong service. A 404 and a capability-controlled UI make the
  ownership boundary explicit.

### What worked

- Focused standalone and Message Desk Go tests passed.
- Playwright verified local logout returns the guest UI with the new
  `Use a desk account` panel, and the provider chooser uses the shared style.

### What didn't work

- Initially registration routes were omitted whenever `accounts == nil`, which
  broke existing tests that intentionally construct a no-account test app.
  The final design makes registration an explicit runtime capability instead
  of inferring it from a dependency pointer.

### What I learned

- Feature availability is protocol-visible behavior and deserves an explicit
  capability rather than an accidental consequence of construction details.

### What was tricky to build

- React hooks cannot be conditionally invoked. The guest branch is a separate
  component: the external branch never calls the registration query hook, so
  it cannot create an unused pre-session registration cookie.

### What warrants a second pair of eyes

- Confirm that future external providers expose an intentional provisioning
  link or admin process rather than re-enabling application registration by
  default.

### What should be done in the future

- Add an operator UI or documented provisioning CLI only after defining its
  authorization and audit requirements.

### Code review instructions

- Review `messageApp.registrationEnabled`, `openExternalMessageApplication`,
  and `ui/src/App.tsx` together.
- Run `go test ./examples/tinyidp-message-app -count=1` and inspect the guest
  page through a browser.

### Technical details

```text
external runtime
  /api/session -> registrationEnabled: false
  /api/registration, /api/accounts -> 404
  guest panel -> provider-owned sign-in
```

## Step 8: Define the Docker topology and private OIDC backchannel

The ticket now includes two multi-stage Debian-based images, Compose service
definitions, named volumes, health checks, seed fixture, and an operator
runbook. A special but bounded Docker requirement is addressed explicitly:
the host browser must use `http://localhost:8081` as the issuer, whereas the
Message Desk container must connect to `http://idp:8081`. The rewrite transport
changes only outbound network routing and preserves the public issuer URL and
Host header seen by OIDC validation.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Complete a self-contained deployment artifact
and verify the implementation rather than leaving local-only commands.

**Inferred user intent:** Make the demo reproducible on a clean workstation
while keeping public browser trust and private container routing distinct.

**Commit (code):** 8739522 — "feat(external-demo): add compose deployment topology"

### What I did

- Added two Dockerfiles, `compose.yaml`, public development seed fixture, and
  a concise runbook.
- Added `--external-backchannel-url` and a path-preserving issuer rewrite
  transport for discovery, JWKS, and token calls.
- Validated configuration with `docker compose -f compose.yaml config`.
- Ran `go test ./examples/tinyidp-external-message-desk/... ./examples/tinyidp-message-app -count=1` successfully.
- Began an actual Compose image build after stopping the temporary tmux demo
  listeners on ports 8080 and 8081. At this diary point Docker had completed
  the IdP image but had not left a Message Desk image or running containers;
  this needs a final build-status follow-up before marking Compose assurance
  complete.

### Why

- DNS names inside a Compose network are not browser-visible hostnames. Simply
  configuring `idp:8081` as issuer would make browser redirects fail, while
  simply configuring `localhost:8081` would make container backchannel calls
  target the app container itself.

### What worked

- The Go code and Compose schema validation succeeded.
- The IdP image was successfully exported as
  `tinyidp-external-message-desk-idp:latest` during the build attempt.

### What didn't work

- The initial `docker compose up --build -d` and follow-up build command
  returned partial BuildKit output before the Message Desk image was exported;
  `docker compose ps` showed no containers. This is recorded as incomplete
  infrastructure validation, not as a passing Compose test.

### What I learned

- Public issuer identity and private reachability are different contracts.
  The transport must rewrite only the network destination, never the issuer
  string which `go-oidc` verifies in discovery and ID tokens.

### What was tricky to build

- With an issuer path, a private backchannel URL must preserve that path. The
  transport strips the canonical issuer prefix from the request and appends
  only the suffix to the private base, preventing doubled path prefixes.

### What warrants a second pair of eyes

- Add a direct `httptest` assertion for `issuerRewriteTransport` before relying
  on it for an issuer mounted below a non-root path.
- Investigate the incomplete BuildKit execution and then run Compose plus the
  full Playwright two-origin suite.

### What should be done in the future

- Finish Compose startup, test login, chooser, local logout, global logout,
  message creation, restart persistence, and invalid callback/CSRF failures.

### Code review instructions

- Read `compose.yaml` with `oidc_client.go` and `external_config.go`.
- Run the focused Go test command, `docker compose -f compose.yaml config`,
  then `docker compose up --build` and browser-test `localhost:8080`.

### Technical details

```text
browser: localhost:8080 <---- redirect ----> localhost:8081
container app: public issuer URL localhost:8081
container app transport: localhost:8081 -> idp:8081
container IdP: validates/mints tokens for issuer localhost:8081
```

## Step 9: Repair named-volume startup and verify the real Compose flow

The Compose test progressed from infrastructure failure to a full two-origin
browser flow. Docker initially mounted named volumes as root, while the images
deliberately ran as an unprivileged service user. A small entrypoint now
creates and chowns `/state` once as root, then uses `setpriv` to execute the
actual server as `tinyidp`. The private `idp:8081` destination also required
its own operator-only validation rule: it is not a browser public origin and
must not be rejected for failing a loopback-hostname constraint.

### Prompt Context

**User prompt (verbatim):** (see Step 5)

**Assistant interpretation:** Complete actual container startup and browser
assurance rather than accepting configuration-only evidence.

**Inferred user intent:** Have a self-contained runnable application a user
can open immediately, backed by evidence that identity and application state
are truly separated.

**Commit (code):** 8d040cb — "fix(external-demo): initialize durable container volumes"

### What I did

- Diagnosed the IdP startup log: `open /state/tinyidp.sqlite: permission denied`.
- Added `docker-entrypoint.sh`, verified `/usr/bin/setpriv` exists in the
  Debian image, and changed both images to drop privileges after chowning the
  named state volume.
- Diagnosed the next startup failure: private `http://idp:8081` was rejected
  as a browser public origin.
- Added `normalizeExternalBackchannelURL`, which keeps strict URL/path rules
  while allowing a private Compose DNS host.
- Rebuilt both images in tmux, started Compose successfully, and confirmed
  both health checks are healthy.
- Used Playwright against `http://localhost:8080` to verify:
  - guest external-mode capability UI and no registration form;
  - password login and requested `openid`, `profile` scopes;
  - callback back to the Message Desk RP;
  - persistent message creation;
  - chooser display and a switch from Amelie to Wesen;
  - global logout returning to guest mode; and
  - a subsequent sign-in opening password login rather than a remembered
    account chooser.
  - local logout returning to guest mode while a subsequent sign-in correctly
    showed the remembered-account chooser, proving the provider session stayed
    distinct from the application session.

### Why

- A container image that works only after manual volume permission repair is
  not a self-contained demo. Likewise, private network routing must not dilute
  public-issuer validation rules.

### What worked

- `go test ./examples/tinyidp-message-app -count=1` passed after adding the
  backchannel validation test.
- `docker compose -f compose.yaml ps` reported both `idp` and `message-desk`
  as healthy with public ports 8081 and 8080.
- The Playwright browser flow completed on the real Compose deployment.

### What didn't work

- The first Compose run failed because the `tinyidp` user could not write a
  fresh root-owned named volume.
- The second run failed because the private Docker DNS backchannel was passed
  through a validator intended for public browser origins.
- Docker BuildKit emitted partial output when run directly in this environment;
  running the long message-image build in tmux retained the build to completion.

### What I learned

- Container durable-state permissions are part of the deployment contract,
  not an operational afterthought. A temporary privileged initialization step
  is compatible with a permanently unprivileged server process.
- Public OIDC issuer identity, browser navigation, and private backchannel
  reachability require three separately named configuration concepts.

### What was tricky to build

- The entrypoint must not leave the application server running as root. It
  performs only `mkdir`/`chown`, then `exec`s `setpriv` with the configured
  UID, GID, and supplementary groups. Docker's named volume mount replaces
  image filesystem ownership, so a Dockerfile-only `chown` cannot solve it.

### What warrants a second pair of eyes

- Review whether the demo should use a one-shot volume-init service rather
  than an entrypoint once this pattern is generalized for production.
- Add source-level Playwright or Go integration harnesses for all tested
  scenarios; this ticket currently records MCP browser evidence but does not
  yet persist a reusable browser script.
- `scripts/01-compose-health-smoke.sh` now persists the repeatable container
  build/readiness portion of this verification; browser interaction remains a
  separate Playwright concern.

### What should be done in the future

- Add durable automated two-origin tests, including local logout and negative
  CSRF/callback cases, before promoting the example beyond development demo
  status.
- Add HTTPS/reverse-proxy deployment examples and replace the committed public
  fixture with an operator secret mount in any shared environment.

### Code review instructions

- Read `docker-entrypoint.sh`, both Dockerfiles, `compose.yaml`, and
  `external_config.go` as one deployment boundary.
- Run the focused Go test, `docker compose up --build`, then exercise the
  sequence listed under What I did from a browser.

### Technical details

```text
container start:
root entrypoint -> chown /state -> setpriv tinyidp -> server

browser test:
guest -> IdP login -> RP callback -> write -> chooser/switch
      -> global logout -> guest -> fresh login form
```

## Step 10: Make the development and production profile boundary explicit

The external demo now states plainly that its Docker Compose topology is a
loopback-only development profile. The current standalone command passes
`embeddedidp.DevMode`, serves public HTTP loopback origins, and uses a
committed public fixture; it must not be promoted by merely replacing
`localhost` with a deployed hostname. The runbook now gives a concrete
production contract and explains why private Docker routing must not redefine
the public issuer.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Continue the listed next steps: automate the
two-origin validation, add durability/failure assurance, and document the
HTTPS production boundary.

**Inferred user intent:** Turn the successful local demonstration into a
retraceable release-hardening path without overstating what is production
ready.

### What I did

- Added a `Deployment profiles and the HTTPS boundary` section to
  `examples/tinyidp-external-message-desk/README.md`.
- Classified the current Compose setup as development-only and documented its
  local HTTP, public seed, development-mode, named-volume, and no-TLS
  assumptions.
- Defined the production requirements for canonical HTTPS issuer/origin,
  secure explicit cookies, production-mode readiness prerequisites,
  trustworthy reverse-proxy origin handling, operator secret injection, and
  coherent database/signing-key backup.
- Marked ticket task 1.3 complete.

### Why

- The existing code intentionally supports a convenient local exercise, but
  its `DevMode` construction cannot provide the production readiness checks
  required by the public embedded provider API. Documentation that calls the
  Compose file a generic deployment would be misleading and unsafe.

### What worked

- The README now names the exact boundary and provides a reviewable checklist
  for a future production-specific host rather than leaving HTTPS and proxy
  behavior implicit.

### What didn't work

- The first patch targeted the workspace parent instead of the nested
  `tiny-idp` repository and failed before changing a file:
  `apply_patch verification failed: Failed to read file to update .../examples/tinyidp-external-message-desk/README.md: No such file or directory`.
  Retargeting the patch to `tiny-idp/examples/.../README.md` applied the
  intended change. No application behavior was affected by the failed patch.

### What I learned

- `embeddedidp.ProductionMode` already validates substantially more than an
  HTTPS URL: persistent/schema-capable store, durable audit, production-ready
  rate limiter and client-address resolver, password-work reporting, usable
  RSA signing material, and secure cookies. A production standalone host must
  deliberately supply those capabilities instead of inheriting this demo's
  defaults.

### What was tricky to build

- A reverse proxy has two separate effects: it may terminate TLS and it may
  provide a private network route. Neither effect changes the issuer that OIDC
  discovery and ID-token verification use. The runbook therefore distinguishes
  browser-visible canonical issuer, RP public origin, and private
  service-to-service backchannel route.

### What warrants a second pair of eyes

- Review the future production-host design for an explicit trusted-proxy
  policy. It must not construct issuer or redirect behavior from untrusted
  browser-controlled `Host` or forwarded headers.
- Review whether production account provisioning and seed replacement are
  addressed by the separate identity-management API work before a deployment
  guide is advertised as complete.

### What should be done in the future

- Add the committed two-origin Playwright integration harness (task 6.1).
- Add restart, bad-seed, unavailable-provider, and secret/persistence checks
  (task 6.3).
- Build a dedicated production host/deployment guide only after those checks
  and the required production dependencies are implemented.

### Code review instructions

- Read the new README section alongside
  `cmd/idp/main.go` and `pkg/embeddedidp/options.go`.
- Confirm that the documented Compose values are intentionally local HTTP and
  that the production list matches `embeddedidp.Options.Validate`.

### Technical details

```text
development demo:
  issuer=http://localhost:8081, DevMode, public fixture, local named volumes

production host:
  issuer=https://issuer.example, ProductionMode, secure cookies,
  durable audit/secrets/store, explicit trusted proxy and backchannel policy
```

## Step 11: Convert browser evidence into a committed two-origin assurance suite

The prior Playwright result was valuable but lived only in an interactive MCP
session. This step converts the same complete behavior into a version-pinned,
ticket-local Playwright suite that any developer or CI worker can execute
against the Compose topology. It also adds a source-level assertion that the
private Docker backchannel rewrites only the network destination while keeping
the public issuer Host and logical request URL intact.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Implement the remaining automated two-origin
integration and operational assurance work, not merely describe it.

**Inferred user intent:** Make the standalone example resistant to regression
by preserving real browser and durable-state checks in the repository.

**Commit (code):** e4a1a2d — "test(external-demo): automate two-origin assurance"

### What I did

- Added a ticket-local pinned `@playwright/test` package, lockfile, config,
  artifact `.gitignore`, browser spec, and executable runner at
  `scripts/02-run-playwright-browser-smoke.sh`.
- Implemented `02-external-demo.spec.mjs` to check both readiness endpoints,
  external-mode registration absence, unrecognized callback rejection,
  `openid`/`profile` consent display, password login, missing-CSRF rejection,
  message creation, provider chooser behavior after local logout, use-another
  account behavior, and global logout requiring a fresh password page.
- Added `TestIssuerRewriteTransportRoutesOnlyTheNetworkDestination` in
  `examples/tinyidp-message-app/oidc_client_test.go`. It asserts a public
  `https://issuer.example.test/idp/...` logical request is cloned to the
  private `http://idp:8081/private-idp/...` destination with the public Host
  preserved and the original request unchanged.
- Installed the ticket-local runner and ran the suite successfully against
  fresh live Compose services, then ran it again after controlled restarts.
- Ran focused Go packages and Compose schema validation successfully.

### Why

- OIDC redirect and cookie behavior, provider CSP, chooser ownership, and
  local-versus-global logout are browser properties. A Go handler test cannot
  establish that Chromium will accept the final redirect chain.
- The private backchannel transport carries a critical issuer invariant that
  should have a direct source-level test in addition to its Compose behavior.

### What worked

- `pnpm ... test:browser` passed twice against the real containers. The final
  run completed one complete scenario in roughly three seconds.
- `go test ./examples/tinyidp-external-message-desk/... ./examples/tinyidp-message-app ./internal/fositeadapter ./pkg/idpui -count=1` passed.
- `docker compose -f examples/tinyidp-external-message-desk/compose.yaml config`
  succeeded.

### What didn't work

- The initial `@playwright/test` 1.55.0 runner expected
  `chromium_headless_shell-1187`, which was absent even after an installer
  invocation that printed only ten-percent download progress. The workstation
  had a complete 1.60.0 Chromium/headless-shell pair, so the pinned runner was
  updated to 1.60.0 and lockfile regenerated rather than relying on an
  incomplete browser download.
- The first live selector expected `Approve access`, while the renderer's
  actual accessible button name is `Approve`. Playwright's snapshot exposed
  the exact semantic control and the selector was corrected.
- A subsequent broad `getByText("Amelie")` assertion matched both the signed-in
  status and a durable preexisting feed entry. The assertion was scoped to
  `header .status`, which tests the intended authenticated-session fact rather
  than an unrelated persisted message.

### What I learned

- Browser assertions must use stable, intentionally scoped accessible names.
  The selector errors were test-harness defects, not protocol failures, and
  the snapshots gave precise correction evidence.
- A pinned Playwright package and a compatible pinned browser build are one
  operational unit. The lockfile provides runner reproducibility; the runner
  script documents installation and never commits browser artifacts.

### What was tricky to build

- The flow is stateful by design. A local logout must lead to a chooser only
  because the same browser retains a provider cookie, while a global logout
  must later show the password fields. A fresh Playwright page/context and
  carefully ordered assertions make these two intended states distinguishable.
- The Message Desk feed is durable, so test text must be unique (`Date.now()`)
  and the identity assertion must not rely on a possibly repeated author name
  in historical messages.

### What warrants a second pair of eyes

- Review the browser package/version strategy when CI is introduced. CI should
  provision the exact Playwright browser build before calling the ticket
  runner and retain traces only on failure.
- Review whether another user-facing application needs a separate account
  fixture/client rather than sharing this intentionally development-only
  Message Desk client.

### What should be done in the future

- Wire `01`, `02`, and `03` into a CI job with Docker support.
- Add an HTTPS/reverse-proxy browser variant when a production host topology
  exists; do not reinterpret this HTTP loopback suite as that evidence.

### Code review instructions

- Read `02-external-demo.spec.mjs` in order; each assertion corresponds to a
  browser- or protocol-boundary invariant.
- Read the source transport test beside `issuerRewriteTransport.RoundTrip`.
- Start Compose, then run `scripts/02-run-playwright-browser-smoke.sh`.

### Technical details

```text
browser test path:
guest -> authorize -> password + consent -> callback -> app session
      -> CSRF-negative POST -> durable message -> local logout -> chooser
      -> use another account -> global logout -> fresh password login

transport test:
logical issuer URL and Host remain public; TCP destination becomes private DNS
```

## Step 12: Add restart, privilege, and development-fixture exposure checks

The browser suite proves the interactive protocol path. This step adds a
separate shell check for properties that need container introspection and
controlled restart behavior. It uses only the deliberately public demo
fixture, checks that the fixture is not emitted in rendered Compose
configuration or service logs, and proves that the provider signing secret,
Message Desk state manifest, and public message feed survive restart.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Finish the explicit failure, persistence, and
secret-leak assurance portion of the existing ticket.

**Inferred user intent:** Demonstrate that a running Docker example has basic
operational safety properties beyond a single happy-path login.

**Commit (code):** e4a1a2d — "test(external-demo): automate two-origin assurance"

### What I did

- Added executable `scripts/03-compose-durability-and-secret-check.sh`.
- Checked `/proc/1/status` in each container rather than the diagnostic
  `docker compose exec` user, proving that both actual server PID 1 processes
  run unprivileged after the entrypoint's `setpriv` transition.
- Checked rendered Compose config and accumulated service logs for the public
  fixture password.
- Captured hashes for `/state/token.key` and `/state/state.json`, captured the
  public feed, restarted IdP first, waited for its readiness, restarted the RP,
  waited for both readiness endpoints, then compared all persisted evidence.
- Added the three-runner sequence to the example README and completed ticket
  tasks 6.1 and 6.3 plus their parent assurance phase.

### Why

- Docker `USER`, an entrypoint privilege drop, volume persistence, and log
  hygiene are not established by a web request. They need direct operational
  assertions.
- Restarting the provider before its dependent RP exercises the intended
  startup ordering without inventing an unsupported simultaneous-restart
  requirement.

### What worked

- The corrected durability script passed. It reported
  `durability and development-fixture exposure checks passed` after both
  controlled service restarts.
- The subsequent complete Playwright run still passed, confirming browser
  behavior after the durability exercise.

### What didn't work

- The first privilege check ran `docker compose exec -T idp id -u` and reported
  `a service is running as root: idp=0 message-desk=0`. This measured Docker's
  diagnostic exec process, not the service. Inspecting PID 1 through
  `/proc/1/status` corrected the measurement and confirmed the actual servers
  are unprivileged.
- Readiness polling printed transient expected connection messages such as
  `curl: (52) Empty reply from server` and `curl: (56) Recv failure: Connection reset by peer`
  during controlled restart. The loop retried and succeeded before its bounded
  timeout; these are recorded as restart timing observations, not ignored
  failures.

### What I learned

- Container diagnostics have their own user identity. A meaningful privilege
  assertion must inspect the process that accepts traffic, not the user Docker
  gives to an ad-hoc exec command.
- A local fixture string can be used safely as a negative exposure sentinel
  only because it is explicitly public and development-only. This script is
  not a general secret scanner and must not receive real credentials.

### What was tricky to build

- State persistence needs stable evidence. Hashing the IdP token secret and RP
  manifest proves the essential bootstrap files survive, while byte-comparing
  the unauthenticated public feed proves the app data view survives. The script
  restarts the provider first and waits before RP restart because the RP
  performs discovery at initialization.

### What warrants a second pair of eyes

- Review whether the app readiness endpoint should eventually include an
  optional external-issuer dependency check. It currently reports its local
  store/audit readiness; changing that contract could affect availability
  semantics during provider maintenance.
- Review production secret scanning separately. Rendered config and logs are
  useful development checks but cannot replace image/SBOM/secret-manager
  policy or centralized log retention review.

### What should be done in the future

- Add Docker-capable CI for all three scripts and retain only failure
  artifacts.
- Add a production host test that exercises secure cookies and an explicitly
  trusted proxy boundary.

### Code review instructions

- Read `03-compose-durability-and-secret-check.sh` with
  `docker-entrypoint.sh` and `compose.yaml`.
- Run the script only against a local development topology; it intentionally
  restarts services.
- Confirm `scripts/.gitignore` excludes `node_modules`, reports, and test
  artifacts.

### Technical details

```text
PID 1 uid != 0
fixture password !in {compose config, service logs}
sha256(idp token.key) before == after
sha256(app state.json) before == after
GET /api/messages before == after
```

## Step 13: Complete security handoff and deliver the updated ticket bundle

The implementation work is now accompanied by a focused security review and
continuation guide. It states the demonstrated claims, their concrete source
and runtime evidence, the precise non-production boundaries, the acceptance
commands, and the next production-host and device-client work. The design,
diary, and review were validated and uploaded together to the ticket's
reMarkable folder.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Complete the remaining handoff/delivery phase
after implementing the test and operational assurance work.

**Inferred user intent:** Leave the work usable by a new contributor without
requiring them to infer what was tested, what failed, or why the example is
not yet a production deployment.

### What I did

- Added `reference/02-security-review-and-handoff.md` with a scope statement,
  architecture, claim/evidence review, reproduction procedure, production
  release gate, CI roadmap, device-client follow-up, and reviewer checklist.
- Validated the new document frontmatter and ran `docmgr doctor` successfully.
- Uploaded design guide, implementation diary, and security handoff as
  `TINYIDP External Demo 001 Assurance.pdf` to
  `/ai/2026/07/15/TINYIDP-EXTERNAL-DEMO-001`.
- Marked Phase 7 complete because the diary, security review, reMarkable
  bundle, and handoff are now present.

### Why

- A source change and passing test are insufficient handoff material for an
  identity-system deployment boundary. The next contributor needs clear
  security claims, exact evidence, explicit limits, and a known-safe next
  sequence.

### What worked

- `docmgr validate frontmatter --doc .../02-security-review-and-handoff.md`
  returned `Frontmatter OK`.
- `docmgr doctor --ticket TINYIDP-EXTERNAL-DEMO-001 --stale-after 30` reported
  all checks passed.
- `remarquee upload bundle ...` reported:
  `OK: uploaded TINYIDP External Demo 001 Assurance.pdf -> /ai/2026/07/15/TINYIDP-EXTERNAL-DEMO-001`.

### What didn't work

- N/A. The delivery validation and upload succeeded on the first attempt.

### What I learned

- The handoff is most useful when it separates development-demo completion
  from production readiness. The completed ticket proves an external OIDC
  integration and operational checks; it does not imply that TLS, trusted
  proxying, production audit/secrets, account lifecycle, backup recovery, or
  CI are solved.

### What was tricky to build

- The final review has to preserve the distinction between direct evidence and
  recommendation. Each positive claim is tied to code, a test, or a live
  assertion; each production requirement is labelled as an open release gate.

### What warrants a second pair of eyes

- Review the proposed CI isolation strategy before enabling it on a shared
  Docker runner; the scripts intentionally create durable demo state and the
  durability script restarts services.
- Review the production-host design separately from this ticket. Adding HTTPS
  URLs to this Compose file is not an adequate substitute for the production
  capabilities enforced by `embeddedidp.ProductionMode`.

### What should be done in the future

- Create/wire the Docker-capable CI job for scripts 01–03.
- Implement the production standalone host and deployment design.
- Continue the planned device-authorization example as a separate client with
  its own protocol and negative-path assurance.

### Code review instructions

- Read `reference/02-security-review-and-handoff.md` after the design guide
  and before changing deployment configuration.
- Run its acceptance command sequence locally; do not reuse the public fixture
  or scripts in production.

### Technical details

```text
bundle contents:
  design guide -> chronological implementation diary -> security handoff

delivery:
  /ai/2026/07/15/TINYIDP-EXTERNAL-DEMO-001/
  TINYIDP External Demo 001 Assurance.pdf
```
