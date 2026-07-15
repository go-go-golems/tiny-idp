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
RelatedFiles: []
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
