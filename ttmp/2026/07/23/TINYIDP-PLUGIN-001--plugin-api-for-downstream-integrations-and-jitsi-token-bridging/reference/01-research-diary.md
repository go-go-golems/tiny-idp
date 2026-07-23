---
Title: Research diary
Ticket: TINYIDP-PLUGIN-001
Status: active
Topics:
    - architecture
    - auth
    - jitsi
    - operations
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp/main.go
      Note: |-
        Production parser middleware composition introduced in commit 294e343
        Registers the compiled-in Jitsi plugin with the production command (commit c6f98bd)
    - Path: repo://internal/cmds/serve_production.go
      Note: |-
        Production host now decodes the reusable section in commit 294e343
        Builds mounts observes and closes plugin runtimes in production (commit c6f98bd)
        Runs the public and internal listeners in one production lifecycle (commit 91f81f5)
    - Path: repo://internal/observability/prometheus.go
      Note: Owns the isolated Prometheus exporter and internal administration handler (commit 91f81f5)
    - Path: repo://internal/pluginapi/api.go
      Note: Trusted plugin authority and lifecycle contracts from commit 513b7b9
    - Path: repo://internal/pluginapi/registry.go
      Note: Immutable registry validation from commit 513b7b9
    - Path: repo://internal/pluginhost/host.go
      Note: Preparation, client validation, routing, readiness, and cleanup from commit 513b7b9
    - Path: repo://internal/pluginhost/oidcbroker/broker.go
      Note: In-process OIDC relying-party broker from commit 4df3a9b
    - Path: repo://internal/pluginhost/oidcbroker/transaction.go
      Note: Encrypted durable one-time transaction manager from commit 4df3a9b
    - Path: repo://internal/plugins/jitsi/definition.go
      Note: Defines typed Glazed configuration and prepared runtime requirements for Phase 5 (commit a0437f6)
    - Path: repo://internal/plugins/jitsi/policy.go
      Note: Bounded versioned Jitsi policy executor from commit a5eecf1
    - Path: repo://internal/plugins/jitsi/runtime.go
      Note: |-
        Implements Jitsi start and callback browser paths for Phase 5 (commit a0437f6)
        Records bounded Jitsi metrics and trace spans (commit 91f81f5)
    - Path: repo://internal/plugins/jitsi/token.go
      Note: Implements exact bounded HS256 Jitsi token signing for Phase 5 (commit a0437f6)
    - Path: repo://internal/sections/production/section.go
      Note: |-
        Canonical production Glazed section introduced in commit 294e343
        Defines the internal administration listener through Glazed (commit 91f81f5)
    - Path: repo://pkg/idpprogram/schema.go
      Note: Array schema contract required by typed roles and groups in commit a5eecf1
    - Path: repo://pkg/sqlitestore/migrations/015_integration_transactions.sql
      Note: Durable integration transaction schema from commit 4df3a9b
ExternalSources: []
Summary: Chronological record of the preliminary TinyIDP plugin architecture research.
LastUpdated: 2026-07-23T16:32:32.222501884-04:00
WhatFor: ""
WhenToUse: ""
---














# Research diary

## Goal

Identify the configuration, browser identity, Goja, lifecycle, deployment, and
observability decisions required before writing the full plugin system design.

## 2026-07-23 — Protocol and prior-ticket review

- Reviewed `TINYIDP-JITSI-001` and `TINYIDP-JITSI-K3S-001`.
- Confirmed that an embedded plugin can replace the standalone OIDC adapter,
  but not Prosody.
- Rechecked current Jitsi token documentation. Prosody validates shared-secret
  or public-key JWTs on the XMPP connection and again against the MUC room.
- Rechecked the current `jitsi-contrib/jitsi-oidc-adapter`; it remains a useful
  behavioral reference and separately deployable fallback.

## 2026-07-23 — Configuration inspection

- Read the TinyIDP command construction, reusable OIDC section, profile
  middleware, config plan, and production server wiring.
- Confirmed the intended precedence:
  `defaults < profiles < config < environment < arguments < flags`.
- Verified from local Glazed source tests that a section prefix participates in
  environment names, allowing `TINYIDP_JITSI_*`.
- Found that `serve-production` does not currently use the parser/profile
  configuration used by `serve-dev` and `print-config`.
- Confirmed that production already uses protected secret-file references,
  which is the appropriate precedent for plugin secrets.

## 2026-07-23 — Scriptability and runtime inspection

- Read `pkg/idpscript`, `pkg/idpsignup`, `pkg/embeddedidp`, and the internal
  browser-session implementation.
- Confirmed that bounded, versioned Goja capabilities fit plugin policy.
- Identified browser identity as the principal missing host service: a handler
  mounted next to the provider cannot currently resolve the active TinyIDP
  browser identity or initiate/resume login through a public API.
- Recorded native browser-identity and embedded OIDC/PKCE approaches.

## 2026-07-23 — Loading and operations

- Compared compiled-in Go registration, standard-library shared objects,
  HashiCorp subprocess plugins, JavaScript-only plugins, and standalone
  adapters.
- Selected compiled-in registration as the exploratory recommendation.
- Confirmed structured logging, durable audit, readiness, and internal atomic
  metric snapshots, but no general production metrics exporter.
- Drafted scoped routing, readiness, secret resolution, Goja policy, and
  observability boundaries for review before the full design.

## Current conclusion

The first full-design decision should be the browser identity seam. The rest of
the plugin structure can be built cleanly around a compiled-in registry and
plugin-owned Glazed sections.

## 2026-07-23 — Full design

- Promoted the exploratory findings into an intern-facing analysis, design,
  and implementation guide.
- Selected a host-owned OIDC relying-party broker for version one instead of
  exposing browser session internals. The broker uses authorization code,
  PKCE S256, nonce, durable one-time transactions, ID-token validation, and
  userinfo.
- Selected an in-process provider-backed HTTP transport for server-side token
  and userinfo calls so the pod does not depend on its own public ingress.
- Defined the compiled-in descriptor, definition, prepared, runtime, registry,
  route, service, readiness, and lifecycle boundaries.
- Defined the Jitsi configuration, claim, Goja policy, secret, audit,
  observability, deployment, failure, and test contracts.
- Added task-level implementation phases through GitOps and media-connected
  browser validation.

## 2026-07-23 — Validation and delivery

- Validated the guide frontmatter with `docmgr validate frontmatter`.
- Validated the complete ticket with
  `docmgr doctor --ticket TINYIDP-PLUGIN-001`; all checks passed.
- The first upload attempt failed because the sandbox could not resolve the
  reMarkable cloud hosts. Repeated the same upload with approved network
  access.
- Uploaded the guide as
  `TINYIDP Plugin System and Jitsi Integration Guide.pdf` to
  `/ai/2026/07/23/TINYIDP-PLUGIN-001` on reMarkable.

## Step 1: Compose production configuration through Glazed

This step removed the production server's exceptional configuration path.
Production settings now live in a reusable `production` section, and
`serve-production` resolves profiles, configuration files, environment values,
arguments, and flags through the same ordered middleware chain as the other
primary commands.

The change also makes the security boundary inspectable: Glazed sees secret
file paths but has no fields capable of accepting the corresponding secret
contents. Focused tests prove value precedence and retain the winning source in
the field provenance log.

### Prompt Context

**User prompt (verbatim):** "Implement TINYIDP-PLUGIN-001, run locally with docker compose to confirm. Keep a detailed diary as you work."

**Assistant interpretation:** Implement every ticket phase in focused commits, preserve a detailed implementation record, and prove the resulting Jitsi integration in a local Compose environment.

**Inferred user intent:** Turn the reviewed plugin design into a professional, locally reproducible TinyIDP integration rather than leaving it as architectural documentation.

**Commit (code):** `294e343` — "feat(config): compose production through Glazed sources"

### What I did

- Added `internal/sections/production` with the canonical Glazed section,
  typed settings, decoder, and tests.
- Changed `serve-production` to compose and decode that section.
- Wired `serve-production` through `ProfileMiddlewaresFunc`,
  `ConfigFilePlanBuilder`, `AppName: "tinyidp"`, and the profile settings
  section.
- Changed the supported configuration-file shape to the documented
  `production:` section; no flat legacy adapter was added.
- Checked ticket tasks `p1s1` through `p1s4`.

### Why

- Plugin sections must exist before Glazed parses values, and production must
  use the same source chain for core and plugin configuration.
- Keeping secrets as file references prevents raw signing material from
  entering flags, environment values, parsed-field output, or provenance logs.

### What worked

- `go test ./internal/sections/production -count=1` passed.
- `go test ./internal/cmds -run 'Production|OwnerOnly' -count=1` passed.
- The pre-commit test, lint, Glazed analyzer, and UI analyzer gates passed.

### What didn't work

- The first focused test command failed before compilation:
  `open /home/manuel/.cache/go-build/...: read-only file system`. It succeeded
  after setting `GOCACHE=/tmp/tinyidp-plugin-go-cache`.
- The broad `go test ./internal/cmds` run reached an unrelated existing
  listener test and failed with
  `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`.
  The production-only test selection avoided claiming coverage from a test the
  sandbox could not execute.
- The first `git add` failed with
  `Unable to create '/home/manuel/code/wesen/go-go-golems/tiny-idp/.git/worktrees/tiny-idp/index.lock': Read-only file system`.
  The same scoped operation succeeded with approved repository access.

### What I learned

- The repository already had the correct source middleware; production simply
  bypassed it.
- Glazed records provenance on each `FieldValue.Log`, so source inspection does
  not require a parallel configuration system.

### What was tricky to build

- Section extraction changes configuration-file nesting while preserving
  command-line flag names. The decoder must use `production.Slug`, not the
  default section, or values silently decode as zero values.
- Required fields cannot be exercised by defaults-only tests, so the section
  tests focus on optional/defaulted fields and schema shape while production
  command tests retain the required-field assertions.

### What warrants a second pair of eyes

- Review any existing deployment configuration for flat production keys. The
  intended new contract is a `production:` mapping, with no compatibility
  loader.
- Confirm the Glazed built-in parsed-field output presents secret file paths
  at the desired operational sensitivity; it never reads file contents.

### What should be done in the future

- Plugin sections must be passed to the same command constructor before
  parsing; they must not read environment variables independently.

### Code review instructions

- Start at `internal/sections/production/section.go`, then inspect the
  `serve-production` construction in `cmd/tinyidp/main.go`.
- Run:
  `GOCACHE=/tmp/tinyidp-plugin-go-cache go test ./internal/sections/production ./internal/cmds -run 'Production|OwnerOnly' -count=1`.

### Technical details

```text
defaults < profiles < config < environment < arguments < flags

config key: production.addr
environment: TINYIDP_ADDR
flag:        --addr
```

## Step 2: Build the compiled-in plugin kernel

This step established the trusted Go boundary that integrations use. The
registry is immutable and validates identity, API version, section slug, and
field prefix collisions before parsing. Prepared plugins declare reviewed
OIDC client requirements, and runtime construction closes already-built
resources in reverse order on partial failure.

The host now also owns scoped route mounting, security headers, and readiness
aggregation. Plugins receive their derived path and cannot acquire the root
router.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement Phase 2 as an independently testable,
compiled-in host kernel before introducing protocol state.

**Inferred user intent:** Keep extension mechanics professional and bounded so
the Jitsi integration does not become special-case server wiring.

**Commit (code):** `513b7b9` — "feat(plugin): add compiled-in host kernel"

### What I did

- Added the descriptor, definition, prepared, runtime, runtime-service, OIDC
  broker, secret resolver, identity, and requirement contracts.
- Added deterministic registry validation and defensive slice copies.
- Added prepare/build/close lifecycle functions and reverse cleanup.
- Added exact public-client, authorization-code, PKCE, redirect, and scope
  requirement validation.
- Added derived `/integrations/<id>/` mounting, host security headers, and
  fail-closed readiness aggregation.
- Checked tasks `p2s1` through `p2s4`.

### Why

- Trusted mechanics and untrusted policy require different authority
  boundaries.
- Registry and client errors must fail before the public listener starts.

### What worked

- `go test ./internal/pluginapi ./internal/pluginhost -count=1` passed.
- The repository pre-commit tests, lint, Glazed analyzer, and UI analyzer
  passed.

### What didn't work

- N/A.

### What I learned

- Existing `idp.ReadinessCheck` and Glazed `Section.GetPrefix` provide the host
  primitives directly; no duplicate health or schema model is needed.

### What was tricky to build

- Partial construction must close only successfully built runtimes, in reverse
  order, while preserving both the build error and cleanup errors.
- Requirements are validated against exact reviewed client properties rather
  than merely checking that a client ID exists.

### What warrants a second pair of eyes

- Review the authority surface in `RuntimeServices`; every added field expands
  what all compiled integrations can do.
- Review whether the fixed CSP is sufficient for every host-rendered plugin
  error page before adding any script or remote asset allowance.

### What should be done in the future

- Keep the API internal until the Jitsi implementation proves the contracts.

### Code review instructions

- Start with `internal/pluginapi/api.go`, then
  `internal/pluginhost/host.go`.
- Run `go test ./internal/pluginapi ./internal/pluginhost -count=1`.

### Technical details

```text
Definition -> Prepared -> Runtime
                      failure
                         |
                         v
              close(runtime[n..0])
```

## Step 3: Add durable one-time OIDC brokerage

This step implemented the host-owned relying-party protocol boundary. Pending
transactions survive process restart in SQLite, but raw state, browser
bindings, nonce values, PKCE verifiers, and plugin state are not stored in
plaintext. Callback consumption is atomic and one-time.

The broker uses the existing fail-closed in-process issuer transport for
discovery, token exchange, JWKS retrieval, and userinfo. It validates the ID
token signature, issuer, audience, expiry, and nonce before mapping the public
identity contract.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the durable OIDC broker as the
security-critical core of Phase 3, retaining browser-path tests for the mounted
Jitsi runtime.

**Inferred user intent:** Reuse TinyIDP's real login/signup behavior without
giving plugins session-store access or adding an unreliable public-ingress
hairpin.

**Commit (code):** `4df3a9b` — "feat(plugin): add durable OIDC relying-party broker"

### What I did

- Added SQLite migration 15 for integration transactions.
- Added keyed state, nonce, and browser-binding hashes.
- Added domain-separated AES-GCM boxes for PKCE verifier and plugin state.
- Added expiry, plugin binding, browser binding, malformed-state, atomic
  consumption, replay, and restart behavior.
- Added OIDC discovery, authorization URL generation, PKCE S256, code
  exchange, ID-token verification, nonce validation, and userinfo mapping.
- Added stable external broker error codes with wrapped internal causes.
- Checked tasks `p3s1` through `p3s4`; `p3s5` remains open for mounted browser
  login/signup/session/cancellation validation.

### Why

- OAuth callback state is a durable security transaction, not process memory.
- Plugins need a stable identity result, not access to browser cookies or
  provider internals.

### What worked

- The complete broker, plugin host, plugin API, and SQLite suites passed.
- Restart persistence, replay, expiry, wrong binding, wrong plugin, ciphertext,
  PKCE, nonce, JWT verification, and userinfo tests passed.
- Repository pre-commit tests and lint passed.

### What didn't work

- The first broker test exposed only `id_token_rejected`, because public errors
  deliberately omit verifier details. After inspecting the wrapped cause, the
  exact failure was:
  `oidc: token is expired (Token Expiry: 2026-07-23 08:05:00 -0400 EDT)`.
- The transaction manager used an injected test clock while the OIDC verifier
  used `time.Now`. Passing the same injected clock into `oidc.Config.Now`
  fixed the inconsistency without disabling expiry checks.
- The first commit gate rejected a helper named `clear`:
  `function clear has same name as predeclared identifier (predeclared)`.
  It was renamed to `zeroBytes`.

### What I learned

- The repository already contained a bounded, exact-origin
  `InProcessIssuerTransport`; the broker did not need another handler
  transport.
- All components evaluating the same protocol transaction must share one clock.

### What was tricky to build

- State must be consumed before exchanging the authorization code to prohibit
  retries and replay. This means a transient token-exchange failure requires a
  fresh browser start, which is the intended fail-closed behavior.
- Error classification must remain stable for themed pages while wrapped
  causes remain available to logs and tests.

### What warrants a second pair of eyes

- Review the domain-separated key derivation and AEAD associated data.
- Review whether deriving transaction keys from the production token secret is
  operationally preferable to introducing a distinct mounted key before host
  integration.

### What should be done in the future

- Add transaction retention to the maintenance policy when the broker is wired
  into the production host.

### Code review instructions

- Start with `internal/pluginhost/oidcbroker/transaction.go`, then
  `broker.go`, and finally migration 15.
- Run:
  `go test ./internal/pluginhost/oidcbroker ./internal/pluginapi ./internal/pluginhost ./pkg/sqlitestore -count=1`.

### Technical details

```text
raw state -> keyed hash -> SQLite primary key
verifier  -> AES-GCM(state hash + "pkce")
app state -> AES-GCM(state hash + "plugin-state")
callback  -> SELECT + conditional UPDATE consumed_at -> commit once
```

## Step 4: Implement bounded Jitsi policy execution

This step added the versioned `integration.jitsi.authorize@v1` Goja contract.
The native boundary constructs identity input, validates nested objects and
bounded string arrays, accepts only typed allow/deny results, and exposes no
capabilities or effects in version one.

A warmed `idpscript.Pool` supplies isolation, deadlines, saturation behavior,
interruption, replacement, readiness, shutdown, and operational counters.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement and exercise all Phase 4 policy
contracts without giving JavaScript HTTP, signing, storage, or ambient module
authority.

**Inferred user intent:** Let deployments express meeting policy in JavaScript
while retaining a professional native mechanics and security core.

**Commit (code):** `a5eecf1` — "feat(jitsi): add bounded Goja authorization policy"

### What I did

- Added the TypeScript input/result declarations and Go codecs.
- Added bounded array schemas to `idpprogram` for roles and groups.
- Added a terminal `@vN` identifier suffix for versioned handlers.
- Added compile-time provider/handler validation, a warmed pool, readiness,
  counters, and shutdown.
- Tested allow, deny, malformed output, forbidden capabilities, timeout,
  interruption replacement, saturation, and declarations.
- Checked tasks `p4s1` through `p4s3`.

### Why

- The design requires typed `string[]` identity fields and the exact
  `integration.jitsi.authorize@v1` contract.
- JavaScript decides claims and access; it must not mint tokens or control
  protocol transitions.

### What worked

- Focused policy, program, scripting, broker, and repository commit gates
  passed.

### What didn't work

- The initial compile rejected `@v1` with
  `lambda.id_mismatch at lambdas.integration.jitsi.authorize@v1`.
  The identifier grammar now permits only one terminal positive version suffix.
- The first commit gate found exhaustive-switch, predeclared-name, and
  capitalized-error issues; correcting all enumerated outcomes, renaming
  `max`, and normalizing error strings resolved them.
- Go still requires a terminal return after the lint-exhaustive switch; that
  was added as an unreachable fail-closed guard.

### What I learned

- The existing scripting pool already supplies the operational behavior the
  plugin needs; a plugin-specific VM scheduler would duplicate mature code.

### What was tricky to build

- The schema model previously had no arrays. Adding them required runtime item
  validation, item counts, schema-reference validation, and cycle traversal.
- A deny outcome carries its stable diagnostic in the native outcome code,
  while an allow outcome carries typed claim data in its validated value.

### What warrants a second pair of eyes

- Review array-schema limits and whether 64 roles/groups is the correct
  production bound.
- Review the two-code public denial allowlist before deployment.

### What should be done in the future

- Add capabilities only as separately versioned, typed native services.

### Code review instructions

- Start with `internal/plugins/jitsi/policy.go`, then the array additions in
  `pkg/idpprogram`.
- Run `go test ./internal/plugins/jitsi ./pkg/idpprogram ./pkg/idpscript -count=1`.

### Technical details

```text
OIDC identity -> validated PolicyInput -> bounded Goja worker
    complete(claims) -> native Decision{Allowed:true}
    deny(code)       -> allowlisted public diagnostic
```

## Step 5: Implement the Jitsi token bridge runtime

This step turned the policy contract into a complete integration runtime. The
runtime owns the browser start and callback handlers, delegates OIDC protocol
mechanics to the host broker, invokes the bounded policy, signs an exact
short-lived Jitsi token, and redirects the browser only to the configured
Jitsi public origin.

The implementation keeps the shared signing secret out of Glazed values. The
configuration contains only an owner-only file path, and secret bytes are
resolved during runtime construction and cleared during shutdown.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the complete Phase 5 Jitsi plugin
configuration, signing, browser handlers, errors, audit, and adversarial tests.

**Inferred user intent:** Produce a usable first plugin whose JavaScript seam
controls policy without weakening the native protocol and signing boundary.

**Commit (code):** `a0437f6` — "feat(jitsi): add token bridge runtime"

### What I did

- Added the `plugin-jitsi` Glazed section with typed settings and strict
  cross-field validation.
- Added bounded owner-only secret-file resolution.
- Implemented exact HS256 Jitsi claims for issuer, audience, subject domain,
  room, timestamps, and `context.user`.
- Added `/start` and `/callback` handlers, secure browser-binding cookies,
  safe redirect construction, themed HTML errors, and non-secret audit.
- Added tests for invalid configuration, wrong secrets, expired tokens,
  incorrect application/domain/room claims, privacy, redaction, callback
  failures, policy denial, and closed runtime behavior.
- Checked Phase 5 tasks `p5s1` through `p5s4`.

### Why

- Jitsi and Prosody need a narrowly scoped application token rather than a
  general TinyIDP ID token.
- Token construction and secret access are privileged native mechanics; Goja
  should return only a validated authorization decision and claim inputs.

### What worked

- Jitsi runtime, signing, configuration, policy, plugin host, and broker tests
  passed.
- The runtime can use either the bounded Goja policy or the fail-closed native
  default without changing the HTTP and signing mechanics.
- The signer rejects closed, expired, wrong-secret, wrong-application,
  wrong-domain, and wrong-room cases.

### What didn't work

- The first lint pass rejected a local variable named `copy` because it
  shadows a predeclared identifier. Renaming it to `candidate` resolved the
  commit gate.

### What I learned

- A fixed maximum token lifetime of ten minutes makes policy output easier to
  reason about and limits the impact of a leaked browser URL.
- Treating the public origin as a parsed, canonical URL allows every redirect
  to be constructed from reviewed components instead of accepting a script- or
  request-provided destination.

### What was tricky to build

- Jitsi claim names overlap with ordinary JWT conventions but have
  deployment-specific meanings. The signer must bind `iss` and `aud` to the
  configured application ID, `sub` to the XMPP domain, and `room` to exactly
  one normalized room.
- Error rendering must preserve a useful browser experience without exposing
  OAuth state, tokens, secret paths, or wrapped internal errors.

### What warrants a second pair of eyes

- Compare the exact claim set with the Prosody token-auth module version used
  by the deployment.
- Review whether ten minutes is the desired maximum for every production
  deployment and whether coordinated HS256 rotation needs an overlap window.

### What should be done in the future

- Exercise the token against a real Prosody instance rather than only the
  independent verifier in the Go test suite.

### Code review instructions

- Start with `internal/plugins/jitsi/definition.go`, then `runtime.go`, and
  finally `token.go`.
- Run `go test ./internal/plugins/jitsi -count=1`.

### Technical details

```text
GET /start?room=R
  -> broker.Begin(plugin=jitsi, state={room:R})
  -> TinyIDP login/signup
  -> GET /callback?code=...&state=...
  -> broker.Complete()
  -> policy.authorize(identity, R)
  -> signer.Sign(identity, R, claims)
  -> 303 https://meet.example/R?jwt=<short-lived token>
```

## Step 6: Wire plugins into the production host

This step integrated the compiled registry into `tinyidp serve-production`.
Plugin sections now participate in command construction before source parsing,
prepared plugins declare their exact OIDC clients, and runtimes are built only
after the provider and durable transaction manager exist.

Lifecycle handling closes plugin runtimes in reverse order on partial startup
failure and normal shutdown. The public handler mounts only derived plugin
prefixes and composes plugin readiness with the provider readiness result.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete production composition so enabling the
Jitsi section creates a validated, mounted, lifecycle-managed plugin.

**Inferred user intent:** Make the plugin implementation reachable through the
real production binary rather than leaving it as an isolated package.

**Commit (code):** `c6f98bd` — "feat(plugin): wire Jitsi into production host"

### What I did

- Constructed an immutable registry containing the Jitsi definition in
  `cmd/tinyidp/main.go`.
- Required `serve-production` construction to receive that registry and
  compose its Glazed sections.
- Prepared plugins before opening the listener and validated every declared
  OIDC client against the reviewed production catalog.
- Constructed the durable transaction manager and in-process OIDC broker from
  host-owned provider services.
- Built and mounted plugin runtimes with scoped secret resolution, audit,
  logging, clock, randomness, meter, and tracer services.
- Added reverse cleanup on every failure path and combined readiness.
- Added production command tests for plugin section composition.

### Why

- Definitions must contribute fields before Glazed resolves profiles, config
  files, environment, arguments, and flags.
- Plugins must authenticate through public OIDC semantics while avoiding a
  fragile public-ingress hairpin inside the process.
- A runtime that fails after another plugin starts must not leak goroutines,
  handles, or secret-bearing objects.

### What worked

- The host integration commit passed the repository pre-commit test and lint
  gates.
- Focused `internal/plugins/jitsi` and `internal/pluginhost` tests passed again
  during the continuation audit.

### What didn't work

- Running the broader command suite inside the restricted workspace sandbox
  failed at an existing `httptest.NewServer` call:
  `httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted`.
  This is a sandbox socket restriction; the same package passed in the
  repository commit gate outside that restriction.

### What I learned

- The existing in-process issuer transport is also the correct production
  broker transport: it preserves full OAuth/OIDC validation without relying on
  cluster DNS or ingress.
- Plugin client requirements provide an early, exact contract between
  deployment configuration and runtime behavior.

### What was tricky to build

- The token secret is cleared after provider construction, while the durable
  integration transaction manager also needs key material. The manager must be
  constructed before clearing the host buffer and must derive domain-separated
  keys rather than retaining the raw slice.
- Every failure branch after runtime construction needs the same reverse-close
  semantics as graceful shutdown.

### What warrants a second pair of eyes

- Review the transaction-key derivation from the core token secret versus
  provisioning a distinct protected integration-transaction key.
- Audit every early return after `BuildAll` to ensure reverse cleanup remains
  complete as the production command evolves.

### What should be done in the future

- Add the dedicated internal observability listener and validate the mounted
  runtime through Docker Compose and a real Prosody token-auth deployment.

### Code review instructions

- Start with `cmd/tinyidp/main.go`, then
  `internal/cmds/serve_production.go`, especially plugin preparation,
  construction, mounting, readiness, and cleanup.
- Run
  `go test ./internal/cmds ./internal/pluginhost ./internal/plugins/jitsi -count=1`
  in an environment that permits loopback listeners.

### Technical details

```text
registry sections
    -> Glazed source resolution
    -> PrepareAll(values)
    -> validate required OIDC clients
    -> construct provider + broker
    -> BuildAll(host services)
    -> MountAll(/integrations/<id>/)
    -> serve
    -> CloseAll(reverse order)
```

## Step 7: Add internal administration and plugin telemetry

This step added the process-owned observability layer required to operate
plugins professionally. TinyIDP now creates an isolated OpenTelemetry meter
provider backed by a Prometheus exporter and serves liveness, aggregate
readiness, and metrics on a separate administration listener.

The Jitsi runtime records only bounded, low-cardinality dimensions and creates
spans for its start and callback operations. User identifiers, email
addresses, rooms, OAuth values, tokens, and error strings never become metric
labels or span attributes.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement Phase 6.1 with a real Prometheus
exporter, internal health/readiness endpoints, and instrumented Jitsi browser
operations.

**Inferred user intent:** Make the plugin deployable and diagnosable without
expanding the public attack surface or leaking identity and authorization data.

**Commit (code):** `91f81f5` — "feat(observability): expose plugin health and metrics"

### What I did

- Added an isolated Prometheus registry and OpenTelemetry meter provider in
  `internal/observability`.
- Added `/healthz`, `/readyz`, and `/metrics` to an internal-only handler.
- Added the Glazed `production.admin-addr` field with a loopback default.
- Started and stopped the public and administration servers in the same
  `errgroup` lifecycle.
- Passed the process meter to plugin construction.
- Instrumented Jitsi start, callback, OIDC completion, policy decisions, token
  issuance, and request duration.
- Added trace spans with stable operation/outcome/reason-class attributes.
- Added scrape, readiness degradation, metric-emission, and span tests.
- Checked task `p6s1`.

### Why

- A global no-op meter proved only API shape; it did not provide observable
  production behavior.
- Health and metrics should not be routed through public Traefik ingress.
- Exporter ownership belongs to the host, while plugins remain
  exporter-neutral.

### What worked

- Focused observability, Jitsi, plugin-host, production-section, and command
  tests passed.
- The Prometheus test scrape contained both request and token counters.
- Span tests observed one start span and one callback span.
- The repository pre-commit test, golangci-lint, Glazed lint, and UI analyzer
  gates passed.

### What didn't work

- The first `go mod tidy` used the default cache and failed repeatedly with
  `open /home/manuel/.cache/go-build/...: read-only file system`. Re-running
  with `GOCACHE=/tmp/tinyidp-plugin-go-cache` succeeded.
- A test that manually constructed `Runtime` bypassed telemetry initialization
  and panicked at `runtime.go:116`. Moving the fixture through `newRuntime`
  restored the constructor invariant.
- The first commit gate reported
  `Error return value of tracerProvider.Shutdown is not checked (errcheck)`.
  A cleanup closure now reports shutdown errors through the test.

### What I learned

- OpenTelemetry Prometheus exporter `v0.65.0` is the compatible release for
  the repository's OpenTelemetry `v1.43.0` dependencies.
- A private Prometheus registry avoids global collector collisions in tests and
  embedded hosts.

### What was tricky to build

- Runtime instrumentation must be created during construction so a broken
  instrument fails startup; direct struct literals must not bypass that
  invariant.
- Both HTTP servers must terminate when either listener or the parent context
  fails, while plugin runtimes remain alive until both servers stop.

### What warrants a second pair of eyes

- Confirm the Kubernetes Service and NetworkPolicy make the administration
  port reachable only by probes and monitoring.
- Review the allowed telemetry attributes whenever a new plugin operation is
  added.

### What should be done in the future

- Add trace exporter configuration only when the deployment has an explicit
  collector endpoint and retention policy.

### Code review instructions

- Start with `internal/observability/prometheus.go`, then the server lifecycle
  in `internal/cmds/serve_production.go`, and finally Jitsi instrumentation in
  `internal/plugins/jitsi/runtime.go`.
- Run
  `go test ./internal/observability ./internal/plugins/jitsi ./internal/pluginhost ./internal/sections/production -count=1`.

### Technical details

```text
public :8443                  internal :9090
  OIDC/provider routes          /healthz
  /integrations/jitsi/*         /readyz -> core + plugin checks
                                /metrics -> private Prometheus registry
```

## Step 8: Reconcile and publish the authoritative implementation guide

This step reconciled the ticket overview and the long-form guide with the
implementation that now exists. The guide remains useful as a design
specification, but it now distinguishes completed host/runtime work from the
Kubernetes, Prosody, and browser-validation tasks that remain.

The editorial pass also made the second design document explicitly
authoritative. This prevents a new intern from mistaking the earlier
option-selection note for the current system contract.

### Prompt Context

**User prompt (verbatim):** "ok, this is great i can already tell. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Finalize the detailed plugin/Jitsi guide in the
existing ticket, make it suitable for an unfamiliar implementer, and deliver
the resulting document to reMarkable.

**Inferred user intent:** Preserve the architecture and implementation
knowledge in a durable teaching document before continuing deployment work.

### What I did

- Reviewed the complete 1,300-line system guide and its section structure.
- Updated the ticket overview so it points readers to the authoritative guide.
- Added an explicit implementation-status note before the phased plan.
- Updated the changelog and this diary before validation and upload.

### Why

- The ticket overview still described the workspace as an option notebook,
  although the design had been selected and most runtime phases implemented.
- A new intern needs to know which document is normative and which tasks remain.

### What worked

- The existing guide already covered system context, API contracts,
  configuration, lifecycle, security, Goja policy, Jitsi claims, Prosody,
  observability, Kubernetes, testing, decisions, alternatives, and file
  references.
- The live `tasks.md` checklist provided a precise source for implementation
  status.

### What didn't work

- N/A

### What I learned

- The long-form guide had already evolved beyond its original research note,
  but the ticket landing page had not been updated to reflect that transition.

### What was tricky to build

- The document describes both target architecture and implemented behavior.
  The status paragraph had to distinguish those without rewriting historical
  phases as if no code existed.

### What warrants a second pair of eyes

- Confirm the remaining Kubernetes and Prosody deployment guidance against the
  final manifests when Phase 6 is completed.

### What should be done in the future

- Update the implementation-status paragraph after Phases 6 and 7 are checked.

### Code review instructions

- Begin with the executive summary and request-path diagram, then verify the
  API sketches against `internal/pluginapi`, `internal/pluginhost`, and
  `internal/plugins/jitsi`.
- Run `docmgr doctor --ticket TINYIDP-PLUGIN-001 --stale-after 30`.

### Technical details

```text
design-doc/01-*  initial option research
design-doc/02-*  authoritative system and implementation guide
tasks.md          live implementation status
reference/01-*   chronological evidence and continuation record
```
