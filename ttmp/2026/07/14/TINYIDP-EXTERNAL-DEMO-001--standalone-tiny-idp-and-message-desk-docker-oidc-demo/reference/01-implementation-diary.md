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
