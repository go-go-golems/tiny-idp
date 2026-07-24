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
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/README.md
      Note: Production Kubernetes, VSO, and pinned Jitsi Helm deployment contract
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/deployment.yaml
      Note: |-
        Local-path permission ordering repair commit d2ec228
        Existing-volume CAP_CHOWN and CAP_FOWNER repair commit 1c0e0e6
        Restart-safe existing-state ownership repair commit 8144cfe
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/jitsi-values.yaml
      Note: Prosody token mode and Jitsi runtime configuration
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/rotation-runbook.md
      Note: Ordered HS256 rotation and rollback procedure
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/runtime-secret.yaml
      Note: Shared Vault-backed runtime Secret and coordinated restart targets
    - Path: repo://deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh
      Note: |-
        Regression guard for ownership and mode ordering commit d2ec228
        Existing-volume capability invariant commit 1c0e0e6
        Restart-safe order invariant commit 8144cfe
    - Path: repo://examples/tinyidp-jitsi/browser-tests/tests/jitsi-plugin.spec.ts
      Note: Eight-case browser, provider-logout, and media-connected conference matrix (commits e9c25b9, fe59277, and f552483)
    - Path: repo://examples/tinyidp-jitsi/compose.yaml
      Note: Complete validated local TinyIDP Jitsi Prosody Jicofo JVB stack (commit e9c25b9)
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
      Note: |-
        Bounded versioned Jitsi policy executor from commit a5eecf1
        Canonical empty role and group arrays for bounded policy input (commit ac161d3)
    - Path: repo://internal/plugins/jitsi/runtime.go
      Note: |-
        Implements Jitsi start and callback browser paths for Phase 5 (commit a0437f6)
        Records bounded Jitsi metrics and trace spans (commit 91f81f5)
        Typed identity intents and CSP-safe completion transition (commit b4cedfa)
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
    - Path: repo://ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go
      Note: Allocates an isolated administration listener for each parallel process harness (commit 3a80254)
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

## Step 9: Repair real-browser callback completion

The first live browser login reached the plugin callback but failed before
policy execution because an OIDC identity with no `groups` or `roles` encoded
those fields as JSON `null`. The versioned Goja input schema correctly requires
arrays. The identity conversion now canonicalizes absent collections to `[]`.

The next browser run exposed a second boundary defect in Chromium 133. The
authorization form POST completed on the server, but CSP `form-action` blocked
the callback's cross-origin redirect to Jitsi. The callback now returns a
cache-disabled, themed transition document with a meta refresh and fallback
link. This preserves the strict CSP instead of adding the Jitsi origin to every
provider form.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Exercise the actual plugin in a browser and
repair integration defects until the OIDC, policy, signing, and Jitsi return
path works.

**Inferred user intent:** Prove that the implemented abstractions compose into
a usable login and signup flow rather than relying only on unit tests.

**Commit (code):** `ac161d3` — "fix(jitsi): encode absent identity groups as arrays"

**Commit (code):** `b4cedfa` — "feat(jitsi): support identity intents and safe completion"

### What I did

- Ran the Jitsi prejoin and TinyIDP login flow with Playwright.
- Traced the first callback failure to `PolicyInputFromIdentity`.
- Added an empty-array regression test and canonical conversion helper.
- Added typed `Registration` and `SelectAccount` broker intents.
- Strictly mapped `intent=signup` and `prompt=select_account`; arbitrary OAuth
  parameters remain forbidden.
- Replaced the callback's cross-origin 303 with a no-store transition page.
- Added runtime and broker tests for all new contracts.

### Why

- JSON `null` is not interchangeable with an array in a versioned schema.
- Signup and account selection are host-owned OIDC intents and must not be
  represented as an unreviewed parameter map.
- Weakening CSP would broaden every authorization page. Terminating the form
  navigation at a same-origin 200 document keeps the existing policy intact.

### What worked

- The focused plugin, broker, and policy suites passed.
- The repository commit gate passed unit tests, golangci-lint, Glazed lint, and
  the UI analyzer.
- A real administrator login produced a room-bound token accepted by Prosody.

### What didn't work

- The first live callback failed with:
  `validate lambda input: field "identity": field "groups": schema "jitsi.strings" requires an array`.
- The first Playwright matrix showed:
  `Refused to send form data to 'https://idp.localhost:8443/authorize' because it violates the following Content Security Policy directive: "form-action 'self' https://idp.localhost:8443".`
- The first commit attempt failed lint with:
  `internal/plugins/jitsi/runtime.go:168:1: named return "registration" with type "bool" found (nonamedreturns)`.
  The helper now uses unnamed return values.

### What I learned

- Chromium evaluates the authorization form's navigation across redirects.
  A same-origin callback that immediately redirects to another origin can
  still be rejected by `form-action`.
- Server audit success does not prove browser navigation success; both are
  required evidence.

### What was tricky to build

- The completion page contains a short-lived JWT in its target URL. It must be
  no-store, rendered through `html/template`, excluded from logs and audit, and
  immediately leave the page while retaining a manual fallback.
- Registration and account selection are mutually exclusive starts. The
  plugin rejects duplicates, unknown values, and combinations before creating
  a durable broker transaction.

### What warrants a second pair of eyes

- Review the transition document's handling of the token-bearing URL and
  confirm the production reverse proxy preserves `Cache-Control: no-store` and
  `Referrer-Policy: no-referrer`.
- Confirm that future plugins use typed broker intents rather than adding a
  generic authorization-parameter escape hatch.

### What should be done in the future

- If a future browser standard provides a cleaner navigation primitive under
  the same CSP, reevaluate the meta-refresh transition with equivalent
  cross-browser tests.

### Code review instructions

- Start with `internal/plugins/jitsi/runtime.go`, then
  `internal/pluginapi/api.go` and `internal/pluginhost/oidcbroker/broker.go`.
- Run
  `go test ./internal/pluginapi ./internal/pluginhost/oidcbroker ./internal/plugins/jitsi -count=1`.

### Technical details

```text
POST /authorize
  -> 303 /integrations/jitsi/callback
  -> 200 no-store transition document
  -> meta refresh / explicit link
  -> https://meet.localhost:8443/<room>?jwt=<token>
```

## Step 10: Run the complete local Jitsi deployment

This step added and validated the complete local deployment requested by the
ticket. The Compose project runs Caddy, TinyIDP, Jitsi Web, Prosody, Jicofo,
and JVB with persistent state, protected local secrets, a reusable development
CA, private service networks, and only the intended HTTPS and UDP host ports.

The acceptance suite exercises failure and success paths through real browser
navigation. Its final test creates two isolated browser contexts, authenticates
both through TinyIDP, waits for both Jitsi conferences to report `connected`,
and confirms both clients see two participants.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Provide a reproducible local Compose deployment
and prove TinyIDP, Goja policy, Prosody token validation, Jicofo, and JVB work
together.

**Inferred user intent:** Leave an environment the user can start and validate
without deploying Kubernetes.

**Commit (code):** `e9c25b9` — "feat(example): add local TinyIDP Jitsi stack"

**Commit (code):** `fe59277` — "test(jitsi): require connected media transports"

**Commit (code):** `f552483` — "test(jitsi): verify provider logout reauthentication"

### What I did

- Added `examples/tinyidp-jitsi/compose.yaml` using pinned Jitsi
  `stable-10978` images.
- Added Caddy TLS, a persistent development CA, protected secret
  initialization, and public-CA export.
- Added deterministic normal and policy-denied local identities.
- Added open signup and bounded Jitsi policy programs.
- Added repeatable Compose smoke checks for service state, HTTPS, readiness,
  plugin metrics, token-auth URL generation, and Prosody token mode.
- Added eight Playwright scenarios: empty token, cancellation, policy denial,
  malformed JWT, signup, account chooser, provider logout, and a two-browser
  conference.
- Verified the two-browser peer connections reached Jitsi's `connected` media
  state and JVB created both endpoints.
- Checked tasks `p3s5`, `p7s1`, and `p7s2`.

### Why

- A successful token unit test cannot prove that Jitsi transmits the token,
  Prosody accepts it, or JVB establishes media transports.
- Named volumes preserve iterative local state while the external CA volume
  avoids repeatedly retrusting development certificates.
- Jicofo and JVB passwords use printable 256-bit hex because arbitrary binary
  bytes made their generated HOCON files unparsable.

### What worked

- `./examples/tinyidp-jitsi/scripts/02-smoke.sh` reported:
  `OK: TinyIDP, Jitsi, Prosody token mode, HTTPS, readiness, and metrics are available`.
- The final Playwright run reported `8 passed (14.4s)`.
- Prosody rejected missing tokens and malformed tokens, then authenticated both
  token-bearing browser sessions.
- Jicofo created the conference, admitted two participants, and selected the
  local videobridge for both.
- JVB created two endpoints and both browser APIs reported media state
  `connected`.

### What didn't work

- The first Compose start failed because TinyIDP attempted to read
  `/root/.config/tinyidp/profiles.yaml`; setting the container `HOME` to the
  unprivileged TinyIDP home fixed it.
- The first generated Jicofo and JVB passwords contained binary control
  characters. Jicofo reported:
  `Expecting a value but got wrong token: 'control character 0x1d'`.
  The initializer preserved the originals as
  `.pre-printable-backup` files and generated 64-character hex replacements.
- The initial browser matrix had four failed cases. Retained traces showed two
  test assumptions and the CSP product defect recorded in Step 9. After the
  product fix and assertion corrections, the original seven passed.
- The first logout test requested `/end_session`, which is not the registered
  provider endpoint and returned 404. It was corrected to `/end-session`; the
  complete eight-case matrix then passed.

### What I learned

- Official Jitsi containers generate HOCON and shell-derived configuration, so
  service credentials must be cryptographically strong and printable.
- Jitsi conference `isJoined()` proves XMPP membership; the stronger
  `APP.conference.getConnectionState() === "connected"` assertion is required
  to prove the media transport.

### What was tricky to build

- Caddy must share two otherwise separated Docker networks while TinyIDP must
  trust only the proxy subnet.
- Prosody and TinyIDP must receive byte-for-byte identical Jitsi signing
  material, while Jicofo and JVB must receive independent service passwords.
- JVB must advertise `127.0.0.1` for a workstation browser while retaining its
  container address internally.

### What warrants a second pair of eyes

- Review the local `127.0.0.1:10000/udp` candidate assumptions on non-Linux
  Docker hosts.
- Confirm the production manifests never reuse the local deterministic
  identities or passwords.

### What should be done in the future

- Complete the Kubernetes and Vault Secrets Operator resources and coordinated
  HS256 rotation runbook in Phase 6.
- Run the same browser matrix against the GitOps deployment in Phase 7.3.

### Code review instructions

- Start with `examples/tinyidp-jitsi/README.md` and `compose.yaml`.
- Run:
  `./examples/tinyidp-jitsi/scripts/00-init-secrets.sh`
  followed by the documented tmux Compose command,
  `./examples/tinyidp-jitsi/scripts/02-smoke.sh`, and
  `./examples/tinyidp-jitsi/scripts/03-browser-tests.sh`.

### Technical details

```text
HTTPS :8443 -> Caddy
  idp.localhost  -> TinyIDP :8081
  meet.localhost -> Jitsi Web :80

Jitsi Web -> Prosody :5280 / :5222
Jicofo    -> Prosody :5222
JVB       -> Prosody :5222
Browser   -> JVB 127.0.0.1:10000/udp
```

## Step 11: Encode the Kubernetes, Vault, and Prosody deployment contract

This step translates the locally proven Compose topology into production-shaped
Kubernetes artifacts without placing any credential bytes in Git. TinyIDP is
managed with Kustomize. Jitsi remains on the upstream
`jitsi-contrib/jitsi-helm` chart at version 2.22.0. Vault Secrets Operator
materializes one runtime Secret that is mounted by TinyIDP and consumed as
environment variables by Prosody, Jicofo, and JVB.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete the remaining deployment phase after
the local Compose proof, preserving the ticket's security and operational
contracts.

**Inferred user intent:** Make the plugin implementation deployable through the
existing k3s, Argo CD, and Vault platform without weakening secret handling or
reimplementing Jitsi packaging.

**Commit (code):** `7f6425d` — "feat(deploy): add TinyIDP Jitsi Kubernetes contract"

### What I did

- Inspected the Hetzner k3s repository's Argo CD, VaultConnection, VaultAuth,
  VaultStaticSecret, image-pull transformation, and workload conventions.
- Inspected pinned upstream Jitsi Helm chart 2.22.0 and rendered it locally.
- Added Kustomize resources for the namespace, service account, VSO resources,
  TinyIDP PVC, deployment, public and administration Services, ingress, and
  NetworkPolicies.
- Added reviewed production OIDC client, signup workflow, and Jitsi policy
  ConfigMap inputs.
- Added Jitsi Helm values for token-only authentication, matching issuer,
  audience, application ID, XMPP domain, host UDP/10000, metrics, probes, and
  resource bounds.
- Added an explicit five-key Vault contract and separate transformed GHCR
  image-pull Secret.
- Added a coordinated HS256 rotation runbook and a repeatable cross-manifest
  validator.

### Why

- TinyIDP and Prosody must receive byte-identical `JWT_APP_SECRET` material,
  while the TinyIDP continuation key and XMPP service credentials must remain
  independent.
- The upstream chart already has `existingSecretName` seams for Prosody,
  Jicofo, and JVB. Using those seams avoids forking the chart and avoids
  secret-bearing Helm values.
- HS256 Prosody configuration accepts one active verification secret. Rotation
  therefore requires an ordered admission pause rather than an unsafe rolling
  overlap.

### What worked

- `kubectl kustomize deploy/kubernetes/tinyidp-jitsi` rendered 557 lines.
- Helm 3.18.4 rendered chart 2.22.0 into 660 lines.
- The rendered chart contains the expected external Secret references,
  `hostPort: 10000`, exact JWT issuer/audience/application values, and the
  TinyIDP token-auth URL.
- `scripts/validate.sh` reported:
  `OK: TinyIDP Kubernetes, VSO, and Jitsi shared-secret contracts are coherent`.
- A read-only cluster call confirmed the configured target is the healthy
  single-node `k3s-demo-1` cluster running Kubernetes 1.34.5+k3s1.
- The repeatable server-side dry-run helper validated every rendered TinyIDP,
  VSO, NetworkPolicy, Jitsi, Prosody, ServiceMonitor, and ingress object against
  the live API without creating resources.
- Live selectors confirm the pod CIDR is within `10.42.0.0/16`, Traefik carries
  `app.kubernetes.io/name=traefik`, and the monitoring namespace carries its
  standard metadata name label.

### What didn't work

- The first server-side dry-run could not resolve the Tailscale Kubernetes API
  hostname inside the restricted sandbox:
  `dial udp 127.0.0.53:53: socket: operation not permitted`.
- Retrying read-only with approved network access reached the cluster, but
  `kubectl apply --dry-run=server` does not persist the Namespace object before
  validating later documents. Every namespaced resource therefore reported:
  `namespaces "tinyidp-jitsi" not found`.
- After those two consecutive failures, I stopped that debugging path as
  required by the repository guidelines instead of attempting another ad hoc
  workaround.
- The Hetzner Terraform firewall currently allows UDP/TCP 40000–40100 for the
  earlier MiroTalk work but does not allow Jitsi's configured UDP/10000.
  Deployment cannot claim working media until the GitOps/infrastructure change
  opens that port and the external two-browser test passes.

### What I learned

- VSO `rolloutRestartTargets` can coordinate all four secret consumers, but
  target names must match the Helm release's rendered resource names exactly.
- With release name `jitsi`, chart 2.22.0 renders
  `jitsi-jitsi-meet-prosody`, `jitsi-jitsi-meet-jicofo`, and
  `jitsi-jitsi-meet-jvb-0`.
- Kubernetes server-side dry-run over a multi-document file does not model
  Argo CD sync-wave persistence for a Namespace prerequisite.

### What was tricky to build

- The TinyIDP image runs as UID/GID 65532 with a read-only root filesystem.
  Persistent SQLite and audit state therefore need an explicit ownership
  initialization step without broad runtime privileges.
- The runtime Secret is intentionally shared as an object but its keys have
  different consumers. TinyIDP mounts only its two allowed keys as files;
  Jitsi components receive only the environment keys their upstream templates
  read.
- VSO restart target names are coupled to the Helm release name. The validator
  cross-checks them against the pinned rendered chart.

### What warrants a second pair of eyes

- Confirm `10.42.0.0/16` is still the production pod CIDR and that Traefik's
  live pod labels match the NetworkPolicy selector.
- Confirm host UDP/10000 is permitted by both the Hetzner firewall and host
  firewall before deployment.
- Review whether the monitoring namespace uses the exact
  `kubernetes.io/metadata.name: monitoring` selector assumed by the admin-port
  NetworkPolicy.

### What should be done in the future

- Validate the resources against the live API after creating a temporary
  dry-run-independent Namespace validation context. (Completed in this step.)
- Move the proven resources into the GitOps repository, provision the Vault
  policy and records, and replace the placeholder TinyIDP image with the
  immutable post-merge SHA.
- Run the local browser matrix against the resulting public deployment.

### Code review instructions

- Start with `deploy/kubernetes/tinyidp-jitsi/README.md`, then inspect
  `runtime-secret.yaml`, `deployment.yaml`, `jitsi-values.yaml`, and
  `rotation-runbook.md`.
- Render the Kustomize resources and pinned upstream chart exactly as shown in
  the README.
- Run `deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh` with the rendered
  Helm file path.
- Verify that neither rendered file contains secret values.

### Technical details

```text
Vault KV v2
  -> VaultStaticSecret/tinyidp-jitsi-runtime
      -> TinyIDP file: TINYIDP_TOKEN_SECRET
      -> TinyIDP file: JWT_APP_SECRET
      -> Prosody env:  JWT_APP_SECRET
      -> Jicofo env:   JICOFO_AUTH_PASSWORD
      -> JVB env:      JVB_AUTH_USER + JVB_AUTH_PASSWORD

Traefik -> TinyIDP :8081
Monitoring -> TinyIDP admin :9090
Browser -> Jitsi Web HTTPS
Browser -> JVB host UDP/10000
```

## Step 12: Rebase the post-merge work and repair the release gate

The original branch was merged as PR 17 before the later plugin implementation
commits existed. I merged current `origin/main`, created the fresh
`feat/tinyidp-plugin-jitsi` branch, and ran the repository's actual release and
test gates rather than assuming the earlier local acceptance run covered the
new package graph.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Make the completed plugin implementation
publishable and ensure all repository-wide consumers remain safe.

**Inferred user intent:** Leave a branch that can pass CI, publish an immutable
TinyIDP image, and become the input to the GitOps deployment.

**Commit (code):** `947c47c` — "chore(plugin): add generated package loggers"

**Commit (code):** `3a80254` — "test(production): allocate isolated admin listener"

### What I did

- Fetched and merged current `origin/main` after confirming PR 17 was already
  merged.
- Created `feat/tinyidp-plugin-jitsi` for the complete post-merge change set.
- Ran the pre-push gate and inspected the generated source it left behind.
- Added the six missing Logcopter package-area files for the new packages.
- Ran `make logcopter-check`.
- Gave every two-process production harness instance its own allocated
  administration listener and passed it through `--admin-addr`.
- Reran the focused process harness and provider-clock introspection tests
  outside the restricted socket sandbox.

### Why

- GoReleaser runs `go generate ./...`; generated package loggers must be
  committed or the release gate dirties the tree and CI's generated-source
  check fails.
- `serve-production` now owns a second listener. A fixed default port is useful
  for an operator but cannot be shared by independently running parallel test
  processes.

### What worked

- `make logcopter-check` passed with all six new generated package loggers.
- Pre-commit fast tests, golangci-lint, Glazed lint, and the IDP UI analyzer
  passed for both repair commits.
- The focused process-harness and introspection tests passed:
  `ok .../01-two-process-harness 11.427s` and
  `ok .../internal/fositeadapter 0.493s`.
- The subsequent repository-wide `go test ./...` completed successfully,
  including `internal/fositeadapter` in 25.053s and the two-process harness in
  10.062s.

### What didn't work

- The first repository-wide run observed a provider-clock expiry test still
  active at the exact boundary. It passed immediately in the focused rerun and
  is unrelated to the plugin packages.
- More importantly, two parallel two-process tests failed after TinyIDP logged:
  `serve production administration listener: listen tcp 127.0.0.1:9090: bind: address already in use`.
- The first focused rerun was executed in the restricted sandbox and every
  `net.Listen` failed with `socket: operation not permitted`. The same exact
  command passed with the required socket permission.
- The first push did not create a remote ref because its pre-push release/test
  gate found these issues and generated the missing source files.

### What I learned

- Adding an administration listener changes the concurrency contract even when
  the public listener already uses an ephemeral port.
- Generated-source checks must be run after introducing packages, not only
  after modifying existing packages.
- A passing local Compose stack and focused plugin suites do not replace the
  repository-wide process harness.

### What was tricky to build

- The process harness restarts the same TinyIDP during a test, so the admin
  address must be stable within one harness while remaining unique across
  parallel harnesses.
- The pre-push commands run in parallel. A truncated terminal view can show the
  successful lint job while the release or full-test job prevents the push.

### What warrants a second pair of eyes

- Review whether `admin-addr` should remain a fixed operator default or require
  explicit configuration in all production invocations.
- The provider-clock test uses an artificial clock but Fosite may compare a
  persisted timestamp at second precision. Its isolated pass suggests timing
  sensitivity worth a separate focused review if CI reproduces it.

### What should be done in the future

- Complete one clean pre-push run. (The full suite completed in this step.)
- Push the new branch and open the image-producing PR.
- Do not deploy until the merged immutable image tag is available.

### Code review instructions

- Review the generated `logcopter.go` files as mechanical outputs.
- Review the harness change at
  `ttmp/2026/07/18/.../scripts/01-two-process-harness/two_process_test.go`.
- Run `make logcopter-check` and the focused command recorded above.

### Technical details

```text
parallel harness A:
  public 127.0.0.1:<ephemeral-A>
  admin  127.0.0.1:<ephemeral-B>

parallel harness B:
  public 127.0.0.1:<ephemeral-C>
  admin  127.0.0.1:<ephemeral-D>
```

## Step 13: Fail closed when rejection audit delivery fails

PR 18's repository-specific audit analyzer identified one discarded audit
error in the Jitsi rejection path. Successful token issuance already failed
closed on audit delivery. Rejections now follow an equally explicit contract:
if the required rejection record cannot be delivered, the browser receives a
themed 503 response and no token or redirect is produced.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Address the PR gate as a production correctness
issue, not as an analyzer suppression.

**Inferred user intent:** Preserve durable security evidence and safe browser
behavior across both successful and rejected plugin operations.

**Commit (code):** `574990b` — "fix(jitsi): fail closed on rejection audit errors"

### What I did

- Inspected CI run 30051961501, job 89355740063.
- Replaced the discarded `Audit.Emit` result in `Runtime.reject`.
- Logged a structured delivery failure without including token or identity
  material.
- Returned HTTP 503 through the existing themed renderer when delivery fails.
- Added a failing audit sink and a regression test proving the response is
  fail-closed and does not expose either internal reason string.
- Ran `make auditlint` and the focused Jitsi suite.

### Why

- A rejection is security-relevant evidence. Silently losing it makes the
  audit record incomplete and hides storage failure from readiness operators.
- Returning the original 400 after audit failure would falsely imply that the
  request was handled normally.
- The browser must not receive raw `audit_delivery_failed` or `invalid_room`
  identifiers.

### What worked

- The repository audit analyzer completed with no findings.
- `go test ./internal/plugins/jitsi -count=1` passed in 0.118s.
- The pre-commit fast suite, golangci-lint, Glazed lint, and UI analyzer passed.

### What didn't work

- The first sandboxed `make auditlint` invocation reported
  `auditlint: ./pkg/... matched no packages`; the same repository command ran
  correctly with the permissions used by CI.
- The first regression assertion expected the internal
  `audit_delivery_failed` code in HTML. The renderer correctly produced the
  stable sentence `No meeting token was issued.` instead. The test now asserts
  that user-facing sentence and explicitly forbids both internal reason codes.

### What I learned

- The themed error renderer is part of the security boundary: tests should
  assert stable public language and the absence of internal identifiers.
- The repository-specific analyzer covers semantic production requirements
  that the normal lint target does not.

### What was tricky to build

- The rejection path has already failed its primary operation. The secondary
  audit failure must change status and operator evidence without recursively
  attempting another audit record.

### What warrants a second pair of eyes

- Confirm HTTP 503 is the desired public status for all rejection-audit
  failures and that reverse proxies do not replace the themed body.

### What should be done in the future

- Push the repair and obtain a completely green PR 18 CI run.

### Code review instructions

- Review `Runtime.reject` and
  `TestRuntimeFailsClosedWhenRejectionAuditCannotBeDelivered`.
- Run `make auditlint` and `go test ./internal/plugins/jitsi -count=1`.

### Technical details

```text
request rejected
  -> emit integration.jitsi.rejected
     -> success: themed 400
     -> failure: structured error log + themed 503
        -> no token
        -> no meeting redirect
        -> no recursive audit attempt
```

## Step 14: Publish the plugin host and prepare the GitOps deployment

PR 18 passed every required repository check and merged as
`2cc95aa7454bd67cc07c29c55d526513b04de700`. The main-branch publication
workflow then produced the immutable image
`ghcr.io/go-go-golems/tiny-idp:sha-2cc95aa`. I used those immutable identifiers
to build the production Argo, Vault, and Hetzner deployment changes.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Carry the locally proven plugin and Jitsi stack
through release, GitOps review, secret provisioning, and production rollout.

**Inferred user intent:** Finish with a reproducible production deployment,
not only source code and a local Compose demonstration.

**Commit (infrastructure):** `60d11f2` — "Deploy TinyIDP-backed Jitsi through Argo CD"

**Commit (infrastructure):** `6527960` — "Use the Vault CLI token store for Jitsi bootstrap"

**Commit (infrastructure):** `608c1b3` — "Use the existing Goja Auth GHCR credential"

### What I did

- Waited for PR 18 CI and the main-branch image publication workflow.
- Created an isolated infrastructure worktree from current `origin/main`.
- Added a multi-source Argo Application that pins the TinyIDP source commit,
  TinyIDP image tag, and Jitsi chart version independently.
- Extended the `prod-apps` project with only the two required source
  repositories and the `tinyidp-jitsi` destination namespace.
- Added a least-privilege Vault policy and Kubernetes auth role.
- Added an idempotent bootstrap script that uses the Vault CLI token store,
  generates independent 256-bit credentials, and never prints secret bytes.
- Added `10000/udp` to the Hetzner firewall for JVB media.
- Rendered the exact inline Helm values and Kustomize source, then submitted
  both renders and the Argo CRs to the live API with server-side dry-run.
- Provisioned the production Vault policy, Kubernetes role, runtime record,
  and image-pull record.
- Opened and merged infrastructure PR 200 and applied the reviewed Terraform
  plan.

### Why

- Argo must render immutable reviewed inputs; a mutable image tag or chart
  revision would make production state non-reproducible.
- Jitsi's browser media path requires a host UDP port that HTTP ingress cannot
  proxy.
- TinyIDP and Prosody must receive the same Jitsi signing secret while all
  other credentials remain independent.

### What worked

- Terraform validated after installing the pinned provider.
- The filtered Terraform plan showed one in-place firewall update: adding
  `udp/10000`; no resource was created or destroyed.
- The exact Argo values rendered 660 Helm lines; Kustomize rendered 557 lines.
- The live API accepted the AppProject, Application, VSO resources, workloads,
  ingress, NetworkPolicy, and ServiceMonitor in server-side dry-run.
- Vault created runtime and image-pull records at version 1 without logging
  their values.
- Infrastructure PR 200 merged cleanly and Terraform reported
  `0 added, 1 changed, 0 destroyed`.

### What didn't work

- The first `terraform validate` lacked the provider in the isolated worktree:
  `Missing required provider registry.terraform.io/hetznercloud/hcloud`.
  `terraform init -backend=false -input=false` installed the locked provider,
  after which validation passed.
- The first secret bootstrap assumed Message Desk had a Vault-managed image
  pull record at `kv/apps/tiny-message-desk/prod/image-pull`. Vault returned
  `No value found at kv/data/apps/tiny-message-desk/prod/image-pull`.
  The existing deployment uses public images. The bootstrap now copies the
  established Goja Auth GHCR record instead and the retry succeeded.

### What I learned

- This cluster intentionally has no app-of-apps discovery. A newly merged
  `gitops/applications/*.yaml` file requires one initial `kubectl apply`; Argo
  owns it thereafter.
- The source application's existing image access model must be checked before
  choosing it as a credential-copy source.

### What was tricky to build

- Multi-source Argo combines a Git Kustomize source and a Helm repository into
  one ownership boundary. The AppProject must authorize both repositories
  before the Application can render.
- Secret creation had to be both idempotent and opaque: an existing record is
  never rotated implicitly, and generated values never appear in stdout.

### What warrants a second pair of eyes

- Review the explicit external source repositories added to `prod-apps`.
- Confirm the JVB host-port model remains appropriate if the cluster gains
  additional nodes or JVB replicas.

### What should be done in the future

- Add an app-of-apps or ApplicationSet only as a separate infrastructure
  project; it is outside this ticket.
- Add a documented credential-copy source selector if several GHCR identities
  become available.

### Code review instructions

- Review infrastructure PR 200, starting with
  `gitops/applications/tinyidp-jitsi.yaml`, `main.tf`, and the scoped Vault
  policy/role.
- Re-run the filtered Terraform plan and both manifest render commands.
- Confirm Vault metadata versions without reading secret data.

### Technical details

```text
Argo Application/tinyidp-jitsi
  source 1: tiny-idp.git @ 2cc95aa... (Kustomize)
    image override: tiny-idp:sha-2cc95aa
  source 2: jitsi-contrib Helm repository
    chart: jitsi-meet 2.22.0

Vault role tinyidp-jitsi
  service account: tinyidp-jitsi
  namespace:       tinyidp-jitsi
  readable paths:  runtime, image-pull
```

## Step 15: Resolve the live WaitForFirstConsumer sync-wave deadlock

The first live Argo reconciliation made Vault, Prosody, Jicofo, JVB, and the
Jitsi web pod healthy, but TinyIDP did not start. Argo waited for the
`tinyidp-state` PVC in wave 0 before applying the TinyIDP Deployment in wave 1.
The cluster's `local-path` storage class deliberately leaves claims Pending
until a consuming pod is scheduled, so those two correct behaviors formed an
ordering deadlock.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Diagnose the live rollout from Argo and
Kubernetes evidence and repair the durable manifest rather than patching the
workload by hand.

**Inferred user intent:** Ensure the production deployment can recreate itself
from Git without undocumented manual ordering.

**Commit (code):** `9e9befb` — "Fix Jitsi PVC sync wave deadlock"

### What I did

- Inspected the Argo operation state and all namespace workloads.
- Confirmed Argo's exact message:
  `waiting for healthy state of /PersistentVolumeClaim/tinyidp-state and 1 more resources`.
- Confirmed the PVC remained Pending while all Jitsi workloads and both
  VaultStaticSecrets were healthy.
- Moved the PVC from wave 0 to wave 1, matching its TinyIDP consumer.
- Rendered the updated Kustomize package and opened TinyIDP PR 19.

### Why

- A `WaitForFirstConsumer` claim and its first consumer must be created in the
  same Argo wave. Otherwise Argo health ordering prevents the scheduling event
  Kubernetes needs to bind the claim.
- Manually applying the Deployment would unblock this one cluster but leave
  disaster recovery and namespace recreation broken.

### What worked

- The updated Kustomize package renders successfully.
- PR 19's reachable-vulnerability check passed while build/smoke and lint
  continued.
- The already-applied Jitsi and Vault resources remained healthy during the
  source fix.

### What didn't work

- The first pre-push release gate in the clean worktree failed because frontend
  dependencies were absent: `sh: 1: tsc: not found`. Installing the locked
  pnpm dependencies fixed that environment prerequisite.
- `go generate` changed an unrelated generated TypeScript declaration in the
  temporary worktree. It was intentionally not staged in the focused fix.

### What I learned

- Server-side dry-run validates schemas and admission, but it cannot prove
  storage-class binding or Argo health-wave liveness.
- Sync waves encode runtime dependencies, not merely preferred presentation
  order.

### What was tricky to build

- The Jitsi Helm objects have the default wave 0 and were allowed to start,
  which made the overall namespace look mostly healthy. The missing TinyIDP
  Deployment and Argo operation message were the decisive evidence.

### What warrants a second pair of eyes

- Verify the PVC and Deployment both carry wave 1 in the rendered source.
- Confirm no future pre-TinyIDP migration Job requires the claim in an earlier
  wave.

### What should be done in the future

- Complete PR 19 CI, merge it, update the Argo source pin, and verify the claim
  binds and TinyIDP becomes ready.

### Code review instructions

- Review `deploy/kubernetes/tinyidp-jitsi/persistent-volume-claim.yaml`.
- Run `kubectl kustomize deploy/kubernetes/tinyidp-jitsi`.
- In a fresh namespace, observe that PVC and Deployment are applied together
  and the local-path provisioner binds the claim.

### Technical details

```text
broken:
  wave 0: PVC (Pending, waits for Pod)
  wave 1: Deployment (not applied, waits for PVC health)

fixed:
  wave 1: PVC + Deployment
          -> scheduler selects node
          -> local-path provisions PV
          -> PVC Bound
          -> TinyIDP starts
```

## Step 16: Correct local-path ownership and mode ordering

After the PVC bound, the TinyIDP Pod reached its init container but could not
start the application. The container first handed `/state` to the unprivileged
TinyIDP UID and then attempted chmod with only `CAP_CHOWN`, which correctly
failed on the now non-owned directory.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Correct and validate the production
volume-permission initializer without weakening its least-privilege contract.

**Inferred user intent:** TinyIDP must recreate cleanly from Git on the
cluster's local-path storage.

**Commit (code):** `d2ec228` — "Fix TinyIDP local-path state permissions"

### What I did

- Inspected the live event and captured `chmod: /state: Operation not permitted`.
- Moved `chmod 0700 /state /state/audit` before `chown -R 65532:65532 /state`.
- Kept the init container limited to `CAP_CHOWN`; no `CAP_FOWNER` was added.
- Added a validator assertion that fails if a future manifest reverses this
  ordering.
- Ran the Kubernetes contract validator and rendered the Kustomize package.

### Why

- Mode restriction is required before handing the state tree to TinyIDP.
- Capability expansion would conceal an ordering error and widen the
  privileged init-container boundary.

### What worked

- The Kubernetes contract validator passed.
- The local-path claim is Bound, establishing that the preceding wave fix
  allowed the volume to be provisioned and mounted.

### What didn't work

- The prior live Pod entered `Init:CrashLoopBackOff` after its successful
  ownership transfer, because `chmod` was no longer authorized.

### What I learned

- Filesystem capability requirements depend on the state created by prior
  commands within the same init container.

### What was tricky to build

- This is a runtime interaction between a real PVC and a restricted Linux
  capability set; manifest schema validation alone cannot expose it.

### What warrants a second pair of eyes

- Review that all state files restored from backup remain safe under the
  recursive ownership transfer.

### What should be done in the future

- Merge this fix, advance Argo's immutable source pin, and prove TinyIDP
  readiness plus the public authentication/browser paths.

### Code review instructions

- Review `deploy/kubernetes/tinyidp-jitsi/deployment.yaml` and its validator.
- Run `bash deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh` with the
  rendered Helm file.

### Technical details

```text
old: chmod after chown -> EPERM with CAP_CHOWN only
new: chmod before chown -> private mode, then safe unprivileged hand-off
```

## Step 17: Repair permissions on already-owned persistent state

The reordered initializer was correct for a fresh volume but the live PVC had
already been changed to UID 65532 by the earlier failed startup. The root init
container retained `CAP_CHOWN` but lacked the separate `CAP_FOWNER` permission
needed to repair mode bits on that pre-existing non-owned directory.

### Prompt Context

**User prompt (verbatim):** "how do we do the CAP_FOWNER / CAP_CHOWN thiing?"

**Assistant interpretation:** Explain and implement the minimal Kubernetes
capability configuration needed to recover the existing TinyIDP state volume.

**Inferred user intent:** Recover production safely without deleting identity
state and without giving the long-running IDP process privileges.

**Commit (code):** `1c0e0e6` — "Repair TinyIDP permissions on existing PVCs"

### What I did

- Kept the fresh-volume `chmod` before `chown` invariant.
- Added only `FOWNER` alongside the existing `CHOWN` capability to the
  root-only, short-lived init container.
- Updated the validator to require both capabilities and the safe command
  ordering.
- Rendered the Kustomize package and ran the shared Kubernetes/Jitsi contract
  validator.

### Why

- `CHOWN` changes ownership but does not grant authority to chmod an inode
  owned by UID 65532. `FOWNER` supplies exactly that repair authority.
- Recreating the PVC would discard the TinyIDP database and audit state.
- The main TinyIDP container remains non-root with all capabilities dropped.

### What worked

- The deployment contract validator passed and the rendered init container
  contains only `CHOWN` and `FOWNER` in addition to `drop: [ALL]`.

### What didn't work

- The live container logs from the prior terminated instance had already been
  removed by containerd, but the live Pod specification still proved it had
  only `CHOWN`, and the preceding failure plus persisted ownership established
  the required capability boundary.

### What I learned

- Recovery behavior for a persistent volume must account for partial mutation
  by a failed previous initializer, not only the fresh-volume case.

### What was tricky to build

- The desired minimal authority changes across the two idempotence states:
  fresh directories need ordering; restored or previously initialized
  directories also need FOWNER to repair modes.

### What warrants a second pair of eyes

- Confirm the init container remains the only process receiving either Linux
  capability and that its mounted volume remains limited to `/state`.

### What should be done in the future

- Merge this focused repair, advance Argo's pinned TinyIDP source, then prove
  readiness and browser admission against the currently Bound PVC.

### Code review instructions

- Review the init-container `securityContext` in the Jitsi deployment.
- Run the deployment validator and inspect the rendered capability list.

### Technical details

```text
new PVC:            chmod -> chown
previously owned PVC: FOWNER permits chmod, CHOWN restores UID/GID
main TinyIDP:       UID 65532, no capabilities
```

## Step 18: Make initialization restart-safe after a successful run

The review of PR 21 identified the remaining state-transition case: after a
successful run, `/state` is UID 65532 and mode 0700. CHOWN and FOWNER alone do
not grant directory search permission, so an initializer that starts with
`mkdir -p /state/audit` would fail before it can perform any repair.

### Prompt Context

**User prompt (verbatim):** "you need to take care of proper sync waves for argo.  or are we not blocked on the PVC?"

**Assistant interpretation:** Confirm the PVC wave is fixed, then address the
new review finding so repeated Argo rollouts and Pod restarts remain safe.

**Inferred user intent:** The live deployment must be idempotent, not merely
able to bootstrap once from an empty persistent volume.

**Commit (code):** `8144cfe` — "Make TinyIDP state initialization restart-safe"

### What I did

- Retrieved the unresolved inline review thread from PR 21.
- Confirmed the PVC is no longer blocked: it is Bound and shares sync wave 1
  with the TinyIDP Deployment.
- Changed the initialization sequence to take ownership of `/state` itself
  before traversing it, restore mode, create/repair the audit directory, and
  only then hand the full tree back to UID/GID 65532.
- Updated the static contract validator to assert the complete ordering.
- Rendered Kustomize and ran the Kubernetes/Jitsi contract validator.

### Why

- `CAP_FOWNER` permits mode repair but does not bypass DAC search permission on
  a 0700 directory owned by UID 65532.
- Reacquiring ownership of the final `/state` path requires no traversal of
  that directory; once root owns it, normal traversal and repair are possible.

### What worked

- The rendered initializer has the explicit order:
  `chown 0:0 /state`, `chmod /state`, `mkdir audit`, `chmod audit`, recursive
  hand-off.
- The deployment contract validator passed.

### What didn't work

- The prior PR 21 sequence would have crash-looped on its next restart even
  though it could repair the currently mounted volume's mode bits.

### What I learned

- Persistent-volume initialization must model both fresh and previously
  successful states. Linux ownership capabilities and directory traversal are
  separate authorization checks.

### What was tricky to build

- The operation on `/state` must be deliberately non-recursive first. A
  recursive operation or child creation would need traversal before the repair
  has restored it.

### What warrants a second pair of eyes

- Review the ordering validator and confirm no future state-preparation command
  is inserted before `chown 0:0 /state`.

### What should be done in the future

- Merge, advance the immutable Argo source pin, and prove the real Pod starts;
  then perform a second rollout to exercise the already-owned state case.

### Code review instructions

- Review the `initialize-state-permissions` script in the deployment.
- Run the validator and inspect the rendered command sequence.

### Technical details

```text
previous state: /state owner=65532 mode=0700
chown 0:0 /state: final-path metadata repair; no /state traversal
chmod 0700 /state: root is now owner; traversal restored
prepare audit child
chown -R 65532:65532 /state: final TinyIDP hand-off
```

## Step 19: Repair a private audit child during live persistent-volume recovery

**Date:** 2026-07-23

### Prompt Context

**User prompt (verbatim):** Review comments reporting that a persistent
`/state` directory and its audit child can prevent later init-container runs.

**Assistant interpretation:** Validate the merged parent-directory repair on
the live local-path PVC, then correct any remaining restart path using the
least privilege already granted to the initializer.

### What I did

- Advanced the Argo Application to the parent-directory repair revision and
  observed the real init-container failure rather than assuming the static
  ordering test covered it.
- Read the terminating container output: `chown: /state/audit: Permission
  denied`.
- Identified the missing state: after a successful run, both `/state` and the
  existing `/state/audit` child may be UID/GID 65532 and mode 0700.
- Added a guarded, non-recursive `chown 0:0 /state/audit` after reclaiming
  `/state` and before `mkdir -p` plus the final recursive handoff.
- Extended the manifest validator to require this direct-child reclaim in the
  correct order.
- Ran the validator and a disposable two-pass BusyBox experiment with exactly
  `CAP_CHOWN` and `CAP_FOWNER`; it demonstrated both paths can be reclaimed
  and then handed back as `65532:65532`, mode 0700.

### Why

Changing the metadata of the `/state` path itself does not require entering
that directory. Once the initializer owns `/state`, it can name its direct
`audit` child, but a recursive `chown -R` still cannot enter a private child
owned by UID 65532. Reclaiming that known child first makes the final recursive
handoff safe without adding broad DAC-override capability.

### What worked

- Live cluster log isolated the exact failed path.
- `deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh` passed with the new
  ordering contract.
- Capability-matched two-pass experiment showed the pre-handoff state as:
  `0:0 700 /state` and `0:0 700 /state/audit`.

### What didn't work

- The previous parent-only repair still left an existing private audit child
  inaccessible to the final recursive ownership operation.
- The first validator expression matched both parent and child `chown` lines;
  anchoring the expressions made the ordering assertion unambiguous.

### What I learned

Persistent-volume repair must account for every private directory the script
will recursively traverse. `CAP_CHOWN` permits ownership changes to a named
path, but it does not confer permission to search a descendant directory.

### What should be done in the future

- Merge this focused correction, advance the immutable Argo source pin, and
  verify both the initial ready rollout and a deliberate second Pod restart.

### Code review instructions

- Check that the script reclaims only `/state` and the known `/state/audit`
  path before its final handoff.
- Confirm no `CAP_DAC_OVERRIDE` or broader privilege was added.
- Run the Kubernetes/Jitsi manifest validator.

## Step 20: Materialize owner-private runtime secrets for an unprivileged pod

**Date:** 2026-07-23

### What I did

- Followed the successful persistent-volume init fix through the live Pod.
- Captured the subsequent server startup rejection: the projected Kubernetes
  Secret did not satisfy TinyIDP's owner-private secret-file contract.
- Changed the existing root-only init container to copy both runtime secrets
  into a memory-backed `emptyDir`, then `chown` them to UID/GID 65532 and
  `chmod` them 0400.
- Mounted only the prepared copy into the unprivileged TinyIDP container.
- Added static validation for the copy, ownership, mode, and new token path.

### Why

Kubernetes Secret projection cannot set the file UID to the application user.
With pod `fsGroup`, the source projection is group-readable, which TinyIDP
correctly rejects. A root-only initializer can read that source and make an
owner-private copy without relaxing TinyIDP's secret validation or granting
the long-running process extra privilege.

### Validation

`deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh` passed, and rendered
Kustomize output confirms the source Secret is init-container-only while the
server receives only the memory-backed prepared-secret mount.
