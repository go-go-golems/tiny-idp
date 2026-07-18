---
Title: Go go goja scripting layer analysis design and implementation guide
Ticket: TINYIDP-GOJA-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Current strict OAuth and OIDC request flow plus proposed authorization and claims insertion points
    - Path: repo://pkg/embeddedidp/options.go
      Note: Current public construction and production-validation boundary that graph materialization must reuse
    - Path: repo://pkg/idp/contracts.go
      Note: Current public policy and authentication contracts to extend with authorization and claims policies
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/sources/01-colleague-identity-microkernel-research.md
      Note: Colleague research that defines the identity-microkernel framing, graph model, API sketches, and security boundary
    - Path: ws://go-go-goja/pkg/engine/factory.go
      Note: Current explicit runtime factory, module selection, ownership, and lifecycle API
    - Path: ws://go-go-goja/pkg/runtimeowner/runner.go
      Note: Serialized runtime call behavior and context-cancellation semantics that shape policy execution
    - Path: ws://go-go-goja/pkg/xgoja/providerapi/module.go
      Note: Current xgoja provider module, host service, runtime owner, and closer contract
ExternalSources:
    - https://github.com/dop251/goja
    - https://openid.net/specs/openid-connect-core-1_0-final.html
    - https://www.rfc-editor.org/rfc/rfc9700.txt
Summary: Intern-oriented design for compiling JavaScript into a validated tiny-idp identity graph while keeping protocols, secrets, persistence, and process ownership in Go.
LastUpdated: 2026-07-10T11:11:55.352987794-04:00
WhatFor: Implementing a safe go-go-goja configuration and request-policy layer on top of tiny-idp without moving OAuth, OIDC, cryptography, or durable security state into JavaScript.
WhenToUse: Read before implementing the scripting module, graph compiler, policy runtime pool, xgoja provider, hot reload, or any script-visible identity primitive.
---








# Go-go-goja scripting layer analysis, design, and implementation guide

## Executive summary

The recommended product is an **identity microkernel**, not a JavaScript rewrite
of an identity provider and not a smaller Keycloak clone. Go remains the host
process and trusted computing base. Existing tiny-idp code continues to own
Fosite protocol validation, exact redirect matching, PKCE, token and session
state, key selection and signing, password verification, audit, rate limiting,
SQLite transactions, maintenance, and HTTP lifecycle. JavaScript describes how
trusted native blocks are assembled and may supply a small number of bounded,
synchronous policy functions.

The colleague research in
`sources/01-colleague-identity-microkernel-research.md:1-64` gives the right
strategic model: compile configuration into an immutable graph at startup, run
only registered lambdas at request time, use one Goja runtime per owner, and
keep blocking or security-sensitive work native. The present tiny-idp and
current go-go-goja APIs support that direction, but they do **not** support the
research's complete fluent API today. In particular, strict tiny-idp has one
password interaction, fixed claim construction, no authorization-policy hook,
no general challenge continuation, and no script runtime dependency.

This guide therefore recommends a staged architecture:

1. Add a pure-Go, serializable `idpgraph.Graph` representation and validator.
2. Add a compile-only `require("tinyidp")` native module that produces that
   graph without opening stores, reading secrets, or starting listeners.
3. Let a Go host resolve opaque resource references and materialize the graph
   into `embeddedidp.Options`.
4. Add narrow public Go policy contracts for authorization and computed claims.
5. Execute registered request-time callbacks in a bounded pool of single-owner
   Goja runtimes, with explicit interruption and worker replacement.
6. Package the module as an xgoja/v2 provider with TypeScript declarations.
7. Add the full typed outcome algebra and Go-owned challenge continuations only
   after the current password flow has been refactored behind native graph
   blocks.

The first production-capable release should be intentionally smaller than the
research vision. It should support a `localWeb` preset, the current strict OIDC
Authorization Code + S256 PKCE flow, password authentication, stored consent,
static and computed claims, allow/deny policy, named host capabilities, embedded
policy tests, graph inspection, and atomic configuration activation. Passkeys,
magic links, TOTP composition, step-up, device flow, token exchange, CIBA,
workload identity, and multi-actor workflows belong to later native-block
packages. Publishing JavaScript names for unimplemented security semantics would
create a misleading API and should be avoided.

## Recommendation in one diagram

```text
                               startup / control plane

  auth.js                 one-shot Goja compiler              immutable Go value
+----------------+      +--------------------------+         +--------------------+
| require(       |      | only tinyidp module      |         | idpgraph.Graph     |
|  "tinyidp")   +----->+ no fs/net/exec/db/env     +-------->+ callback IDs       |
| fluent builders|      | bounded source + deadline|         | capability refs    |
+----------------+      +--------------------------+         +---------+----------+
                                                                         |
                                                    validate + resolve    |
                                                                         v
+------------------+     +--------------------------+         +--------------------+
| host config and  +---->+ native materializer      +-------->+ embeddedidp.Options|
| secret/resource  |     | fail-closed preflight    |         | + PolicySet        |
| registry         |     +--------------------------+         +---------+----------+
+------------------+                                                    |
                                                                         v
                                 request / data plane

browser/RP ---> net/http ---> Fosite + tiny-idp native blocks ---> SQLite/keys/audit
                                  |
                                  | bounded policy input
                                  v
                       +---------------------------+
                       | single-owner Goja pool    |
                       | selected callbacks only   |
                       | explicit capabilities only|
                       +---------------------------+
                                  |
                                  v
                         structured decision/claims
```

The arrow from JavaScript ends at structured data. It never ends at key bytes,
raw OAuth messages, password hashes, SQL handles, or an unconstrained network
client.

## How an intern should read this guide

Read in this order:

1. Learn the terminology and the current package map.
2. Follow the current strict authorization-code flow.
3. Study the gap table before interpreting the colleague API examples as
   already implementable.
4. Understand the four target layers: graph, compiler, materializer, executor.
5. Read the security boundary and runtime ownership sections twice.
6. Implement phases in order and do not begin full challenge composition before
   the authorization and claims seams are covered by tests.
7. Read the ticket diary before continuing work; it records the exact baseline
   and investigation decisions.

### Terms

- **IdP:** identity provider; tiny-idp authenticates users and issues OIDC/OAuth
  artifacts.
- **RP:** relying party or OAuth client.
- **Strict engine:** the Fosite-backed, production-shaped tiny-idp path.
- **Mock engine:** the scenario and failure-injection server under
  `internal/server`; it is not the scripting implementation target.
- **Graph:** immutable, serializable description of identity flows and native
  block composition.
- **Block:** a native Go operation with declared input, output, effects, and
  capability requirements.
- **Callback:** a JavaScript function registered during compilation and invoked
  later by stable ID.
- **Capability:** a narrow host-owned service that a callback can invoke through
  an explicit interface.
- **Compiler runtime:** one-shot Goja runtime that executes configuration code.
- **Policy runtime:** pooled Goja runtime that has loaded the same script and
  executes registered callbacks.
- **TCB:** trusted computing base. In this design the TCB is Go, Fosite, the
  selected store, key provider, host configuration, and approved native blocks.
- **Activation:** atomic replacement of one validated graph and its warmed
  policy-runtime generation with another.

## Problem statement

Tiny-idp now has a credible production embedding boundary, but customization is
primarily Go-only. `embeddedidp.Options` accepts a store, audit sink, consent
policy, rate limiter, client-address resolver, authenticator, password policy,
and maintenance configuration
(`pkg/embeddedidp/options.go:40-53`). This is a useful dependency-injection
surface, yet adding application-specific claims or routing requires editing Go
or implementing Go interfaces and rebuilding the host.

The desired scripting layer should make common identity composition easier
without weakening protocol correctness. A user should be able to select a
trusted preset, reference host-provided stores and services, add a claims
function, add an allow/deny policy, validate the result, run policy examples,
and deploy it as an embedded handler, sidecar, or script-configured appliance.

The central design problem is therefore:

> How can tiny-idp accept expressive JavaScript configuration and bounded
> application policy while preserving Go ownership of every protocol,
> cryptographic, secret, durable-state, replay, and lifecycle invariant?

### Goals

- Compile script configuration into a serializable immutable graph.
- Keep the existing strict Fosite engine as the first protocol implementation.
- Expose a versioned JavaScript API with deterministic validation errors.
- Support static presets and named graph slots.
- Support small synchronous callbacks for authorization and claims.
- Expose only declared, typed, host-owned capabilities.
- Make scripts testable before activation.
- Make graph activation atomic and rollback-friendly.
- Support direct Go embedding and xgoja/v2 generated runtimes.
- Preserve tiny-idp's production validation, readiness, maintenance, and
  shutdown behavior.
- Give interns a package-by-package implementation path with executable gates.

### Non-goals for the first release

- Reimplementing OAuth, OIDC, JWT, PKCE, CSRF, WebAuthn, password hashing, or key
  management in JavaScript.
- Running third-party untrusted scripts in-process.
- Exposing filesystem, process, arbitrary SQL, environment, or network modules.
- Implementing every namespace in the colleague research at once.
- Replacing the existing admin CLI with script mutation.
- Hot-swapping in-flight authentication continuations before a versioned
  continuation schema exists.
- Active/active SQLite or distributed policy-runtime coordination.
- Preserving an unpublished scripting API; this is a new pre-release surface,
  so it should be designed directly rather than wrapped in compatibility shims.

## Research conclusions and required adaptations

The source research establishes five principles that this design accepts:

1. Go owns protocols and secrets; JavaScript assembles trusted primitives
   (`sources/01-colleague-identity-microkernel-research.md:3-8`).
2. Scripts compile into immutable graphs and are validated before activation
   (`:38-48`).
3. Request lambdas are short, synchronous, bounded, and run in single-owner
   runtimes (`:50-64`).
4. Blocks use explicit outcomes rather than overloaded booleans or exceptions.
5. Presets are ordinary named graphs with stable patch slots.

Three adaptations are required for the current repositories:

- **Use an explicit native module.** The research proposes an injected global
  `Auth.v1`. Current go-go-goja and xgoja/v2 treat selected CommonJS modules as
  an explicit capability boundary. The first API should therefore be
  `const A = require("tinyidp").v1`. An optional global alias can be added later
  by a runtime initializer, but it must not be the only API.
- **Compile, then let Go start.** The research allows `.run()` to enter native
  Go. For tiny-idp v1, `build()` should return or register a graph and the Go
  host should start the server after the compiler runtime closes. This prevents
  listener, store, and secret lifetimes from becoming side effects of a config
  VM.
- **Ship a narrow first graph.** The current strict engine can safely back OIDC,
  password login, consent, sessions, claims, keys, and issuance. It cannot yet
  honestly back general `choose`, `challenge`, step-up, passkey, magic-link, or
  multi-actor semantics. Those names should follow native implementation, not
  precede it.

## Current tiny-idp architecture

### Product split

The repository contains two engines:

| Engine | Main package | State | Purpose | Scripting target? |
|---|---|---|---|---|
| Mock | `internal/server` | in-memory scenario state | local tests and failure injection | No, except a future deterministic test preset |
| Strict | `internal/fositeadapter` via `pkg/embeddedidp` | public `idpstore.Store`, SQLite in production | production-shaped OAuth/OIDC | Yes |

The strict adapter explicitly states the ownership boundary: Fosite owns
protocol parsing, authorization codes, PKCE, token exchange, refresh handling,
and response writing; tiny-idp owns login, scope policy, discovery/JWKS, and
claims (`internal/fositeadapter/provider.go:1-5`). This is exactly the seam the
identity-microkernel model needs.

### Current package map

```text
cmd/tinyidp
  main.go                         Cobra/Glazed command assembly
  internal/cmds/serve_production.go
                                  hardened example host and lifecycle

pkg/embeddedidp
  options.go                      public construction and preflight contract
  provider.go                     Handler, readiness, maintenance, close

pkg/idp
  contracts.go                    policy/authenticator/limiter/address contracts
  audit.go                        audit sink and health
  password.go                     password acceptance/work contracts

pkg/idpstore
  types.go                        clients, users, grants, tokens, sessions, keys
  interfaces.go                   storage + atomic security transition contracts
  claims.go                       scope-filtered standard profile claims

pkg/sqlitestore                   durable single-writer SQLite implementation

internal/fositeadapter
  provider.go                     Fosite composition and HTTP request flow
  session.go                      browser session cookie + durable record
  consent.go                      stored and development consent policies
  sqlstore.go                     Fosite protocol state over SQLite

internal/authn                    password verification and lockout
internal/keys                     RSA parsing, JWKS, rotation
internal/oidcmeta                 issuer and discovery metadata
```

### Strict provider construction

`embeddedidp.New(ctx, opts)` validates options before creating the adapter
(`pkg/embeddedidp/provider.go:39-73`). In production mode validation requires,
among other things:

- HTTPS issuer and a store;
- valid clients;
- at least 32 bytes of token secret;
- secure cookies;
- durable audit, production-ready limiter and address resolver;
- production password policy and bounded password work;
- persistent, current-schema storage with maintenance;
- exactly one usable RS256 signing key and valid published verification keys.

These checks live at `pkg/embeddedidp/options.go:56-190`. A script materializer
must call this same validation. It must not duplicate or bypass it.

The production command demonstrates host responsibilities at
`internal/cmds/serve_production.go:106-205`: it reads an owner-only token
secret, opens SQLite and audit stores, creates the address resolver, constructs
the provider, runs maintenance, wraps the handler with a request-size limit,
configures TLS and timeouts, schedules maintenance, and gracefully shuts down.
The script layer should configure identity behavior; this host remains the
lifecycle template.

### Current authorization-code request flow

```text
GET /authorize
  -> Fosite validates client, redirect URI, response type, scopes, PKCE shape
  -> tiny-idp reads browser session
  -> stored consent check
  -> render login/consent or finish authorization

POST /authorize
  -> parse form
  -> resolve trusted client address
  -> rate limit
  -> validate CSRF
  -> Fosite re-validates authorization request
  -> password authenticator verifies login
  -> create opaque server-side browser session
  -> finish authorization
       -> load client
       -> require/record consent
       -> grant requested allowed scopes and audience
       -> build OIDC session and scope-derived claims
       -> Fosite creates authorization response/code

POST /token
  -> rate limit
  -> Fosite validates code/client/PKCE or refresh token
  -> grant only client-allowed requested scopes
  -> Fosite creates/writes token response

GET /userinfo
  -> Fosite introspects access token
  -> render claims persisted in the OIDC session
```

Concrete anchors:

- Fosite handlers are composed at
  `internal/fositeadapter/provider.go:180-220`.
- Strict routes are registered at `:282-307`.
- GET and POST authorization behavior is at `:340-442`.
- Password authentication is called at `:415-432`.
- Token processing is at `:461-499`.
- UserInfo copies persisted session claims at `:501-515`.
- Claim construction is fixed in `newOIDCSession` at `:547-571`.
- Final consent and code issuance occur at `:731-766`.

### Current durable state and secret separation

`idpstore.User` contains profile and account state, while
`PasswordCredential` contains the encoded password hash
(`pkg/idpstore/types.go:31-66`). Protocol records store hashes rather than raw
codes and tokens (`:103-146`). Browser sessions store a hash of the opaque
cookie and an expiry (`:161-172`; `internal/fositeadapter/session.go:14-41`).
Signing keys are Go records with private PEM and lifecycle metadata
(`pkg/idpstore/types.go:175-184`).

This separation defines script visibility:

- scripts may receive a redacted projection of `User`;
- scripts must never receive `PasswordCredential`, token hashes, session hashes,
  signing-key PEM, raw authorization codes, refresh tokens, or Fosite request
  objects;
- scripts should receive immutable plain objects, not reflected store records;
- scripts return decisions and claims, never mutations to durable protocol
  records.

### Existing extension seams

Useful public seams already exist:

- `idp.ConsentPolicy` (`pkg/idp/contracts.go:16-21`);
- `idp.RateLimiter` (`:23-27`);
- `idp.ClientAddressResolver` (`:29-33`);
- `idp.PasswordAuthenticator` (`:149-168`);
- `idp.Sink` and audit health;
- `idpstore.Store` and named atomic operations
  (`pkg/idpstore/interfaces.go:143-161`).

Missing seams required for scripting:

- authorization policy after native request validation and authentication;
- computed-claims policy before the OIDC session is persisted;
- a general authentication-block interface;
- challenge continuation persistence;
- hook/effect dispatch with explicit delivery semantics;
- graph generation and activation metadata;
- policy-runtime health and metrics.

### Important current limitations

1. Claims are hard-coded through `idpstore.ClaimsForScopes`
   (`pkg/idpstore/claims.go:3-35`).
2. `AuthResult.AMR` exists (`pkg/idp/contracts.go:158-162`) but the strict login
   path stores only the user and creates a browser session without that AMR
   (`internal/fositeadapter/provider.go:415-432` and
   `internal/fositeadapter/session.go:14-24`). Step-up cannot be correct until
   AMR/ACR propagation is fixed.
3. The interaction renderer is one HTML password/consent form
   (`internal/fositeadapter/provider.go:710-728`). There is no resumable typed
   challenge engine.
4. `finishAuthorize` has no application authorization decision before consent
   and code issuance (`:731-766`).
5. The strict engine exposes OIDC Authorization Code + refresh only. Research
   examples involving device, CIBA, token exchange, DPoP, WebAuthn, or workload
   identity need native protocol work before script exposure.

## Current go-go-goja architecture

### Runtime construction

The sibling repository provides the runtime ownership and provider machinery
needed here:

- `engine.NewRuntimeFactoryBuilder` collects explicit module registrars and
  runtime initializers (`../go-go-goja/pkg/engine/factory.go:32-105`).
- `Build()` freezes the plan and validates duplicate IDs (`:122-179`).
- `NewRuntime()` creates Goja, an event loop, a `runtimeowner.RuntimeOwner`, a
  require registry, runtime services, and close hooks (`:182-288`).
- `engine.NativeModuleRegistrar` registers one loader by explicit module name
  (`../go-go-goja/pkg/engine/module_specs.go:51-78`).
- `Runtime.Close` cancels lifecycle state, waits, interrupts active JavaScript
  when needed, runs closers in reverse order, and stops the owner/loop
  (`../go-go-goja/pkg/engine/runtime.go:66-151`).

### Default-module hazard

A plain runtime builder exposes all modules in the default registry
(`../go-go-goja/pkg/engine/factory.go:84-88,137-150`). That registry includes
filesystem, database, exec, OS, and other host-access modules through blank
imports in `pkg/engine/runtime.go`.

Therefore the compiler and policy factories **must** use:

```go
engine.NewRuntimeFactoryBuilder(
    engine.WithImplicitDefaultRegistryModules(false),
    engine.WithDataOnlyDefaultRegistryModules(false),
).WithModules(tinyIDPModule)
```

Do not rely only on `MiddlewareSafe()`. It selects data-oriented modules
(`../go-go-goja/pkg/engine/module_middleware.go:21-30`), which may be useful for
other products, but the identity compiler needs no ambient module besides its
own API. Every additional module must be an explicit reviewed decision.

xgoja/v2 follows the safer pattern: its app factory disables both implicit and
data-only defaults, then registers only selected provider modules
(`../go-go-goja/pkg/xgoja/app/factory.go:100-140`).

### Runtime ownership

Goja itself is not goroutine-safe. `runtimeowner.Call` schedules work onto the
runtime's event loop and returns a result
(`../go-go-goja/pkg/runtimeowner/runner.go:90-134`). A context deadline can
cause `Call` to return, but that code alone does not interrupt JavaScript that
is already running. Goja's `Runtime.Interrupt` must be used to stop JavaScript;
it does not interrupt native Go functions. Consequently:

- each worker must own exactly one runtime;
- all calls go through `Runtime.Owner.Call`;
- invocation timeout must schedule `VM.Interrupt`;
- native capability methods must accept contexts and enforce their own
  deadlines;
- after timeout or panic, discard and rebuild the worker rather than assuming
  its state is clean;
- `ClearInterrupt` may run only after execution has stopped and synchronization
  proves no concurrent interrupt remains.

Goja `Compile` produces a runtime-independent program that may be run in many
runtimes. Compile the script source once per activation generation, then load
that program independently into every worker.

### Native module and xgoja provider APIs

For direct embedding, `engine.NativeModuleRegistrar` is preferable to global
`modules.Register` because it makes module selection explicit and allows a
per-compiler collector instance.

For generated xgoja binaries, add a provider package. A
`providerapi.Module` declares `Name`, `DefaultAs`, description, JSON config
schema, TypeScript descriptor, and a per-runtime `NewModuleFactory`
(`../go-go-goja/pkg/xgoja/providerapi/module.go:41-50`). Setup receives host
services, runtime owner, and closer registration (`:13-23`). The xgoja
`hostauth` provider is a useful security pattern: it resolves narrow host-owned
services and exposes builder methods rather than database handles
(`../go-go-goja/pkg/xgoja/providers/hostauth/hostauth.go:30-105`).

## Gap analysis

| Desired capability | Current support | Gap | First implementation |
|---|---|---|---|
| Go-hosted process | Strong | none | preserve `serve-production` lifecycle |
| Serializable immutable graph | None | no graph DTO/compiler/validator | add `pkg/idpgraph` |
| Fluent versioned JS API | None | no Goja dependency/module | add `pkg/gojamodules/tinyidp` |
| Opinionated local web preset | Partial native strict path | no preset or slot metadata | compile preset to native graph |
| OIDC auth code + PKCE | Strong strict support | graph cannot select it | native `protocol.oidc` block |
| Password login | Strong native support | no block abstraction; AMR lost | native password block + AMR fix |
| Stored consent | Existing interface | not graph-addressable | native consent block/reference |
| Static claims | Scope-based fixed claims | no extension pipeline | claims policy seam |
| Computed claims callback | None | no callback runtime | policy pool + result validation |
| Allow/deny policy | None | no finish-authorization hook | authorization policy seam |
| Step-up | Not supported | AMR + challenge orchestration absent | later typed challenge phase |
| Passkey/magic link/TOTP | Not strict-native | protocol/storage/UI absent | later native blocks |
| Capabilities | Conceptual only | no registry/projection/permissions | host capability registry |
| Policy tests | None | no test collection/runner | compile-time test specs |
| Atomic hot reload | None | no generation manager | activation manager |
| xgoja generated host | go-go-goja supports providers | tiny-idp provider absent | provider + DTS + example |
| Untrusted plugins | Unsafe in-process | no sandbox | external process only |

## Target architecture

### Four layers

#### 1. Graph layer: pure Go data

`pkg/idpgraph` contains no Goja, Fosite, HTTP, SQLite, or key imports. It defines
serializable data structures, stable enum strings, graph schema versioning, and
validation diagnostics.

```go
type Graph struct {
    SchemaVersion string            `json:"schemaVersion"`
    APIID         string            `json:"apiId"`
    Name          string            `json:"name"`
    Mount         string            `json:"mount"`
    Protocols     []ProtocolSpec    `json:"protocols"`
    Flows         map[string]Flow   `json:"flows"`
    Nodes         map[string]Node   `json:"nodes"`
    Slots         map[string]string `json:"slots"`
    Callbacks     []CallbackSpec    `json:"callbacks"`
    Capabilities  []CapabilityRef   `json:"capabilities"`
    Tests         []PolicyTest      `json:"tests"`
    Source        SourceIdentity    `json:"source"`
}

type Node struct {
    ID           string          `json:"id"`
    Kind         string          `json:"kind"`
    InputType    ValueType       `json:"inputType"`
    OutputType   ValueType       `json:"outputType"`
    Children     []string        `json:"children,omitempty"`
    Config       json.RawMessage `json:"config,omitempty"`
    CallbackID   string          `json:"callbackId,omitempty"`
    RequiredCaps []string        `json:"requiredCapabilities,omitempty"`
    Effects      EffectClass     `json:"effects"`
}
```

A graph contains callback IDs and source hashes, never `goja.Value`, Go
closures, open files, store objects, or key material.

#### 2. Compiler layer: JS builders to graph

`pkg/idpscript.Compiler` creates a one-shot runtime with only the tiny-idp
module. The module's builder objects mutate a Go-owned draft. `CompileFile`
returns a deep-copied `idpgraph.Graph`, diagnostics, source hash, and registered
callback metadata. The runtime then closes.

```go
type Compiler interface {
    Compile(ctx context.Context, source Source) (Artifact, error)
}

type Artifact struct {
    Graph       idpgraph.Graph
    Program     *goja.Program       // process-local cache, not serialized
    Source      []byte              // bounded, immutable copy
    SourceHash  [32]byte
    Diagnostics []idpgraph.Diagnostic
}
```

The serialized deployment artifact is graph + source + hash + optional
signature. `*goja.Program` is only an in-memory optimization.

#### 3. Materializer layer: graph to trusted Go services

`pkg/idpscript.Materializer` resolves names against a host registry:

```go
type HostResources struct {
    Stores         map[string]idpstore.Store
    AuditSinks     map[string]idp.Sink
    RateLimiters   map[string]idp.RateLimiter
    AddressSources map[string]idp.ClientAddressResolver
    Authenticators map[string]idp.PasswordAuthenticator
    Capabilities   CapabilityRegistry
    Secrets        SecretResolver
}
```

Scripts say `A.ref.store("primary")`, not
`A.store.sqlite("./users.db")`, in production. The host opens SQLite, checks
permissions, reads secrets, and owns closure. Development convenience helpers
may exist in a separate explicitly unsafe package or command, never in the
production compiler module.

Materialization produces:

```go
type RuntimePlan struct {
    GraphID       string
    Provider      embeddedidp.Options
    Policies      idp.PolicySet
    PolicyFactory *PolicyRuntimeFactory
    Tests         []idpgraph.PolicyTest
}
```

The materializer validates capabilities, native block availability, production
mode, and `embeddedidp.Options.Validate(ctx)` before activation.

#### 4. Executor layer: native request flow plus bounded callbacks

The strict adapter invokes public policy interfaces. Static graph nodes execute
in Go. Only callback nodes enter the policy pool.

```go
type AuthorizationPolicy interface {
    DecideAuthorization(context.Context, AuthorizationContext) (Decision, error)
}

type ClaimsPolicy interface {
    ComputeClaims(context.Context, ClaimsContext) (map[string]any, error)
}

type Decision struct {
    Kind   DecisionKind // allow or deny in v1
    Code   string
    Reason string       // internal/audit-safe, not blindly returned to client
}
```

The adapter never passes Fosite objects to these interfaces. It builds immutable
plain contexts from validated data.

### Dependency direction

```text
pkg/idpstore  <---- pkg/idp
                      ^
                      |
pkg/idpgraph          | public policies
     ^                 |
     |                 |
pkg/idpscript --------+
     ^
     |
pkg/gojamodules/tinyidp ----> go-go-goja/goja
     ^
     |
pkg/xgoja/providers/tinyidp -> xgoja/providerapi

pkg/embeddedidp -> internal/fositeadapter -> Fosite
       ^                   |
       +----- PolicySet ----+
```

Rules:

- `pkg/idpgraph` stays runtime-agnostic.
- Fosite types stay in `internal/fositeadapter`.
- Goja types stay in `pkg/idpscript` and module/provider packages.
- `pkg/idp` owns stable policy contracts.
- No lower layer imports the JS fluent API.

## Proposed JavaScript API v1

### Entry point

Use CommonJS because it is the explicit module-selection contract of current
xgoja:

```js
const A = require("tinyidp").v1;

module.exports = A.idp("notes")
  .use(A.preset.localWeb({
    store: A.ref.store("primary"),
    audit: A.ref.audit("security"),
    rateLimiter: A.ref.rateLimiter("login"),
    login: A.authn.password(),
    sessions: {
      idle: "30m",
      absolute: "12h"
    }
  }))
  .mount("/auth")
  .validate("strict")
  .build();
```

`build()` is pure with respect to host resources. It finalizes the draft and
returns a branded graph handle that only the native compiler can export. It does
not start a listener.

### Computed claims and authorization

```js
const A = require("tinyidp").v1;

const app = A.idp("orders")
  .use(A.preset.localWeb({
    store: A.ref.store("primary"),
    login: A.authn.password()
  }))
  .capabilities({
    services: ["orders.membership"]
  })
  .append(
    A.slot.login.authorize,
    A.policy.decide("orders-member", ctx => {
      const membership = ctx.cap.orders.membership({
        subjectId: ctx.subject.id,
        tenantId: ctx.subject.tenant
      });
      return membership.active
        ? A.decision.allow()
        : A.decision.deny("not_an_orders_member");
    })
  )
  .append(
    A.slot.token.claims,
    A.claims.compute("orders-claims", ctx => {
      const membership = ctx.cap.orders.membership({
        subjectId: ctx.subject.id,
        tenantId: ctx.subject.tenant
      });
      return {
        tenant_id: ctx.subject.tenant,
        roles: membership.roles,
        plan: membership.plan
      };
    })
  );

A.test("disabled membership is denied", {
  callback: "orders-member",
  given: {
    subject: { id: "u-1", tenant: "acme" },
    client: { id: "orders-web" },
    scopes: ["openid", "profile"]
  },
  capabilities: {
    "orders.membership": { active: false, roles: [], plan: "free" }
  },
  expect: { decision: "deny", code: "not_an_orders_member" }
});

module.exports = app.validate("strict").build();
```

### Supported first-release namespaces

| Namespace | Initial members | Native behavior |
|---|---|---|
| `preset` | `localWeb`, `testLab` | expands to named graph nodes/slots |
| `ref` | `store`, `audit`, `rateLimiter`, `addressResolver`, `authenticator` | opaque host resource reference |
| `protocol` | `oidc` | current strict Fosite composition |
| `authn` | `password`, `existingSession` | current password/session behavior |
| `policy` | `allow`, `deny`, `rbac`, `decide` | Go-native or registered callback |
| `claims` | `oidcStandard`, `static`, `compute`, `merge` | validated claim pipeline |
| `consent` | `stored`, `always` (development only) | existing consent contracts |
| `issue` | `oidc` | current native Fosite issuance |
| `decision` | `allow`, `deny` | structured v1 decisions |
| `slot` | stable local-web slot constants | patch points |
| `test` | policy test registration | activation gate |

Do not expose `stepUp`, `challenge`, `choose`, `passkey`, `magicLink`, `totp`,
`tokenExchange`, or `workload` until their native semantics exist.

### Slot contract

The local-web preset should publish stable slots:

```text
login.identify
login.authenticate
login.authorize
login.consent
login.claims
token.claims
issue.authorizationCode
issue.idToken
issue.accessToken
issue.refreshToken
```

A slot has a declared input/output type and cardinality. `replace` requires one
compatible node. `append` is allowed only for ordered pipeline slots. `wrap`
requires an explicitly wrappable block. Unknown or incompatible slots fail
compilation.

### JavaScript data conventions

- lowerCamelCase keys only;
- plain objects and arrays at the JS boundary;
- RFC 3339 strings for times unless a TypeScript declaration says otherwise;
- durations as Go-style strings (`"30m"`, `"12h"`) parsed natively;
- no `undefined` in returned policy values;
- integers must remain within JavaScript's safe integer range;
- maximum depth, key count, array length, string length, and encoded-byte size
  enforced on every input and output;
- functions accepted only at registration points and immediately assigned a
  stable callback ID.

### Syntax profile

The pinned Goja dependency identifies itself as ECMAScript 5.1+ with much of
ES6. Do not promise arbitrary modern Node.js syntax. Phase 0 should create a
checked syntax profile. A safe first profile may allow `const`/`let`, arrow
functions, template strings, object shorthand, arrays, maps as plain objects,
and ordinary synchronous functions, while disallowing dynamic `import`, ESM,
workers, async I/O, and Node globals. TypeScript examples should be transpiled
to the checked profile before deployment.

## Pure-Go graph model

### Node descriptor registry

Every native block is registered with metadata:

```go
type BlockDescriptor struct {
    Kind            string
    Version         int
    Input           idpgraph.ValueType
    Output          idpgraph.ValueType
    AllowedChildren ChildRule
    Effect          idpgraph.EffectClass
    ConfigSchema    json.RawMessage
    RequiredCaps    []CapabilityRequirement
    Validate        func(Node, ValidationContext) []Diagnostic
    Build           func(Node, MaterializeContext) (NativeBlock, error)
}
```

Effect classes should be explicit:

- `pure`: deterministic data transformation;
- `read`: reads a named capability;
- `write`: writes through a named capability;
- `challenge`: persists native continuation state;
- `issue`: creates a security artifact;
- `protocol`: validates or advances protocol state.

The graph validator can then forbid, for example, a write effect in a pure
claims slot or a JavaScript callback in an issuance-critical slot unless a
specific profile permits it.

### Outcome algebra

The full graph engine should use the colleague research's five outcomes:

```go
type OutcomeKind string

const (
    OutcomeOK        OutcomeKind = "ok"
    OutcomeChallenge OutcomeKind = "challenge"
    OutcomeDeny      OutcomeKind = "deny"
    OutcomeSkip      OutcomeKind = "skip"
    OutcomeError     OutcomeKind = "error"
)

type Outcome struct {
    Kind         OutcomeKind
    Value        Value
    Evidence     []Evidence
    Challenge    *Challenge
    Denial       *Denial
    Failure      *Failure
}
```

Security semantics:

- `ok` advances;
- `challenge` suspends with a Go-owned continuation;
- `deny` terminates due to a valid negative decision;
- `skip` means not applicable and is the only result that permits
  `firstAvailable` to try another branch;
- `error` is infrastructure or internal failure and fails closed unless a
  native recovery rule explicitly handles it.

A rejected credential is `deny`, never `skip`. Otherwise `choose(password,
magicLink)` could silently fall through after a bad password, producing
factor-confusion vulnerabilities.

### Composition operators

Implement operators only after typed node execution exists:

```text
seq            ordered success pipeline
all            all children must return ok
choose         user-visible selection among eligible factors
firstAvailable only skip advances to the next child
when           native/policy conditional branch
switch         typed selector branch
map            pure successful-value transform
tap            explicit observation/effect
recover        explicit error-category recovery
timeout        native budget wrapper
cache          only deterministic, non-secret, explicitly cacheable results
```

Each operator must define behavior for all five outcomes in table-driven tests.

### Validation passes

Run validation in deterministic passes:

1. Schema/API version support.
2. Unique graph, flow, node, callback, test, and slot IDs.
3. Reference resolution.
4. Node config schema validation.
5. Input/output slot compatibility.
6. Cycle detection and entrypoint reachability.
7. Every path terminates in an allowed outcome.
8. Capability declaration and host availability.
9. Effect placement policy.
10. Protocol invariants: issuer/mount, clients, scopes, redirect URIs,
    audiences, PKCE profile, token lifetimes.
11. Production-profile bans: test keys, fixed clocks, allow-all consent,
    development stores, debug blocks.
12. Callback source and permission metadata.
13. Embedded policy test execution.
14. Native `embeddedidp.Options.Validate(ctx)` after materialization.

Diagnostics should contain code, severity, graph path, source location when
available, and remediation text.

## Policy contracts and strict-adapter insertion points

### Authorization context

```go
type AuthorizationContext struct {
    RequestID string
    Subject   SubjectView
    Client    ClientView
    Scopes    []string
    Audience  []string
    Auth      AuthenticationView
    Request   RequestView
}

type SubjectView struct {
    ID                string
    Sub               string
    Email             string
    EmailVerified     bool
    PreferredUsername string
    Groups            []string
    Roles             []string
    Tenant            string
    Locale            string
}

type AuthenticationView struct {
    AMR      []string
    ACR      string
    AuthTime time.Time
    Age      time.Duration
}
```

Do not include password hashes, lockout counters, cookie handles, token values,
raw headers, arbitrary form fields, or Fosite interfaces.

### Authorization insertion point

In `finishAuthorize`:

```text
native Fosite validation already succeeded
native user authentication/session lookup succeeded
load client
normalize requested scopes/audience
-> invoke AuthorizationPolicy
   allow: continue
   deny: audit stable reason and write an OAuth-safe access_denied response
   error/timeout: audit and fail closed with server_error
then evaluate/record consent
then grant native scopes/audience
then build session and ask Fosite to issue code
```

The policy must run on both fresh-login and existing-session paths because both
converge on `finishAuthorize` (`internal/fositeadapter/provider.go:356-366` and
`:438`). It must not run before Fosite validates the client and redirect URI;
otherwise policy errors could be redirected to an attacker-controlled URI.

### Claims insertion point

Change `newOIDCSession` to return `(*openid.DefaultSession, error)`. Build base
claims through `idpstore.ClaimsForScopes`, then invoke the claims policy with a
copy, validate and merge allowed additional claims, and only then persist the
Fosite session.

Protected claim names must never be script-overridable:

```text
iss sub aud exp iat nbf nonce auth_time acr amr azp at_hash c_hash jti
```

The native issuer controls those fields. v1 computed claims may add
application names, `roles`, `groups`, or namespaced claims subject to scope and
client release policy. The result becomes part of Fosite's persisted OIDC
session, so ID token and UserInfo remain consistent (`provider.go:501-515`).

### AMR/ACR correction

Before step-up or authentication-routing work:

1. pass `AuthResult.AMR` out of the authenticator call;
2. write it into the durable browser session (`idpstore.Session.AMR`);
3. preserve/reload it on existing-session paths;
4. add it to native OIDC claims when policy allows;
5. define ACR assignment natively;
6. test fresh, silent, forced reauthentication, refresh, and UserInfo behavior.

Without this work, a callback cannot reliably distinguish password, passkey,
or multi-factor authentication.

## Compiler design

### Compilation algorithm

```text
Compile(ctx, source):
  reject source larger than MaxSourceBytes
  hash exact source bytes
  compile Goja Program once
  create DraftCollector
  create runtime factory with:
    implicit defaults = false
    data-only defaults = false
    only tinyidp compiler module bound to DraftCollector
  create one runtime with bounded lifetime
  start deadline interrupter
  Owner.Call:
    run Program
    read module.exports or collector's single finalized graph
  stop interrupter safely
  export deep-copied draft to idpgraph.Graph
  close runtime
  validate graph structure
  return Artifact
```

Reject:

- no exported graph;
- more than one root graph unless multi-realm is explicitly enabled;
- unfinished builder;
- callback without a stable name;
- non-serializable config;
- undeclared capability;
- duplicate callback or node name;
- callback closure that cannot be re-registered when the script is loaded into
  policy workers.

### Callback registration

The script must register callbacks deterministically each time it loads. A
callback key can be derived from API version, graph name, explicit callback
name, and source location/hash:

```text
v1/orders/orders-member@sha256:...
```

Do not serialize functions. Every policy worker loads the exact same program,
collects its callback registry, and verifies that the registry fingerprint
matches the compiler artifact before joining the pool.

### Compiler isolation

- no `fs`, `database`, `exec`, `os`, `process`, `fetch`, timers, or environment;
- no module roots or arbitrary `require` loader;
- no mutable host service objects;
- one graph draft per compile;
- source, graph, node, callback, and diagnostic bounds;
- wall-clock deadline plus Goja interrupt;
- process boundary required for untrusted authors.

## Request-time runtime pool

### Pool model

```text
PolicyRuntimePool generation N
  worker 1 -> runtime 1 -> owner 1 -> callback registry fingerprint F
  worker 2 -> runtime 2 -> owner 2 -> callback registry fingerprint F
  worker 3 -> runtime 3 -> owner 3 -> callback registry fingerprint F

request -> bounded acquire -> Owner.Call -> validate result -> return worker
                                     timeout/panic -> discard + replace worker
```

A worker is never used concurrently. Pool capacity, acquire timeout, callback
execution timeout, maximum capability calls, and maximum result bytes are
configuration with production defaults.

### Invocation pseudocode

```go
func (p *Pool) Invoke(ctx context.Context, id string, input any) (any, error) {
    worker, err := p.acquire(ctx)
    if err != nil { return nil, ErrPolicySaturated }

    healthy := false
    defer func() {
        if healthy { p.release(worker) } else { p.replaceAsync(worker) }
    }()

    callCtx, cancel := context.WithTimeout(ctx, p.maxExecution)
    defer cancel()

    stopInterrupt := interruptOnDone(callCtx, worker.Runtime.VM)
    result, err := worker.Runtime.Owner.Call(callCtx, "policy:"+id,
        func(ownerCtx context.Context, vm *goja.Runtime) (any, error) {
            fn := worker.Callbacks[id]
            value, err := fn(goja.Undefined(), vm.ToValue(copyInput(input)))
            if err != nil { return nil, normalizeJSError(err) }
            return exportBounded(value)
        })
    stopped := stopInterrupt()

    if err != nil || !stopped {
        return nil, err
    }
    if err := validatePolicyResult(id, result); err != nil {
        return nil, err
    }
    healthy = true
    return result, nil
}
```

`interruptOnDone` is subtle. `Runtime.Interrupt` may be called from another
goroutine, but `ClearInterrupt` may only run after the runtime is stopped and no
concurrent interrupt can still fire. This helper requires dedicated race tests.
Because Goja interrupts do not stop native Go functions, capability functions
must be synchronous, context-aware, and independently bounded.

### Failure semantics

| Failure | Authorization behavior | Claims behavior | Worker behavior |
|---|---|---|---|
| pool acquire timeout | fail closed | fail closed | none available |
| script timeout | server error | server error | discard/replace |
| JS exception | server error | server error | discard by default |
| invalid result | server error | server error | discard |
| missing capability | activation should have failed; runtime fail closed | same | discard |
| capability timeout | server error | server error | discard |
| policy deny | OAuth-safe denial | not applicable | healthy |

Do not silently fall back to allow or base claims in production. A separately
named development mode may support diagnostic fallback, but it must not satisfy
production readiness.

### Pool readiness and metrics

Add readiness checks and metrics:

- active graph generation/hash;
- configured and ready workers;
- pool acquire latency/saturation;
- invocation count by callback and outcome;
- interruption, panic, exception, invalid-output, and worker-rebuild counts;
- capability call latency/error counts;
- last successful activation and last failed reload reason;
- policy test count and result.

Metric labels must use bounded callback names, not subjects, clients supplied by
attackers, or arbitrary error messages.

## Capability model

### Capability registry

A capability is registered by a stable versioned name and an adapter that
projects a narrow API into each runtime.

```go
type CapabilityDescriptor struct {
    Name         string
    Version      int
    Effect       idpgraph.EffectClass
    InputSchema  json.RawMessage
    OutputSchema json.RawMessage
    NewBinding   func(BindingContext) (Binding, error)
}
```

Example host registration:

```go
caps.Register("orders.membership", MembershipCapability{
    Service: appMembershipService,
    Timeout: 50 * time.Millisecond,
    MaxCallsPerInvocation: 2,
})
```

JS projection:

```js
ctx.cap.orders.membership({ subjectId, tenantId })
```

The binding validates input, calls a typed Go service with the current request
context, redacts output, and validates output. It does not expose the service
object itself.

### Permission checks

A graph must declare capabilities. A callback must declare or infer the subset
it uses. Activation succeeds only when:

- host registry contains a compatible version;
- effect class is allowed in the callback's slot;
- production profile permits it;
- per-invocation budgets are defined;
- policy tests provide fakes for required capabilities.

### Forbidden script access

JavaScript must never directly:

- parse or validate JWT signatures;
- choose cryptographic algorithms;
- read signing key bytes or token secrets;
- read password hashes or raw credential records;
- construct/consume authorization codes or refresh tokens;
- validate redirect URIs, PKCE, nonce, state, or replay markers;
- mutate raw OAuth requests after native validation;
- issue tokens or decide to ignore invalid signatures;
- open files, processes, sockets, or SQL connections;
- read process environment;
- mutate the active graph in place;
- retain request contexts or host capabilities after invocation.

## Graph activation and hot reload

### Generation manager

```go
type Generation struct {
    ID         uint64
    Artifact   Artifact
    Plan       RuntimePlan
    Pool       *PolicyRuntimePool
    Activated  time.Time
}

type Manager struct {
    active atomic.Pointer[Generation]
}
```

Reload algorithm:

```text
read bounded source
compile in isolated compiler runtime
validate graph
resolve host resources
materialize native plan
run embedded policy tests
warm entire policy pool
verify callback fingerprints
run provider preflight/readiness probe
atomically swap active generation
stop sending new calls to old generation
drain old in-flight calls with deadline
close old pool and generation-owned resources
record auditable activation event
```

Store, key, and audit resources should normally be host-owned and shared across
generations; the generation only owns resources it created explicitly. This
prevents reload from closing the live database.

### In-flight continuations

Before challenge flows exist, requests are short and generation-local. Once
continuations are added, persist:

- graph schema version;
- graph generation/source hash;
- flow ID and native node ID;
- opaque native challenge state;
- expiry, subject/client binding, replay marker;
- no JavaScript closures or heap snapshots.

On resume, either route to a retained compatible generation or run an explicit
native migration. Never resume against a structurally unrelated graph by
matching only node names.

## Go-owned challenge state for the full graph

The current browser session is not a general continuation. A future challenge
store needs a public contract separate from OAuth codes and sessions:

```go
type ChallengeRecord struct {
    IDHash         []byte
    GenerationHash []byte
    FlowID         string
    NodeID         string
    ClientID       string
    SubjectID      string
    Kind           string
    State          []byte // native, versioned, encrypted/authenticated as needed
    CreatedAt      time.Time
    ExpiresAt      time.Time
    ConsumedAt     *time.Time
}

type ChallengeStore interface {
    CreateChallenge(context.Context, ChallengeRecord) error
    ConsumeChallenge(context.Context, []byte, time.Time) (ChallengeRecord, error)
}
```

Raw continuation handles stay in the browser; storage keeps a keyed hash. The
store enforces one-time consumption and expiry atomically. JavaScript receives a
challenge descriptor suitable for rendering decisions, not the persisted
state.

## Deployment modes

### Embedded

A Go application owns resources and mounts the handler:

```go
artifact, err := compiler.CompileFile(ctx, "auth.js")
plan, err := materializer.Materialize(ctx, artifact, hostResources)
manager, err := idpscript.NewManager(ctx, plan)
provider, err := embeddedidp.New(ctx, plan.Provider.WithPolicies(manager))

mux.Handle("/auth/", provider.Handler())
```

This should be the first supported mode because it matches tiny-idp's existing
public API.

### Sidecar

A Go binary uses the same compiler/materializer but exposes HTTP or gRPC to
nearby applications. The sidecar owns all process and network concerns. Scripts
do not gain network access merely because the deployment is a sidecar.

### Script-configured appliance

Add commands such as:

```text
tinyidp script validate --file auth.js --host-config host.yaml
tinyidp script explain  --file auth.js --case request.json
tinyidp script test     --file auth.js --host-config host.yaml
tinyidp serve-script    --file auth.js --host-config host.yaml ...
```

`serve-script` remains a Go command analogous to `serve-production`. The JS
file does not call `ListenAndServe`.

## xgoja/v2 packaging

### Provider package

Add `pkg/xgoja/providers/tinyidp`:

```go
const PackageID = "tinyidp"

func Register(reg *providerapi.ProviderRegistry) error {
    return reg.Package(PackageID, providerapi.Module{
        Name:        "tinyidp",
        DefaultAs:   "tinyidp",
        Description: "Compile tiny-idp identity graphs and policies.",
        ConfigSchema: moduleConfigSchema,
        TypeScript:  tinyIDPTypeScriptModule(),
        NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (
            require.ModuleLoader, error,
        ) {
            resources, err := lookupTinyIDPHostServices(ctx.Host)
            if err != nil { return nil, err }
            collector := NewDraftCollector(resources.APIProfile)
            return gojamodule.NewLoader(collector), nil
        },
    })
}
```

For **compile-only** xgoja commands the provider needs no secret or store host
service. For a generated serving command, inject host-owned resources through a
stable key such as `tinyidp.host-resources.v1`. Follow the xgoja host-service
pattern: provider code resolves typed values from `HostServiceLookup`, and
resource owners register closers.

### Example `xgoja.yaml`

```yaml
schema: xgoja/v2
name: tinyidp-script

go:
  module: xgoja.generated/tinyidp-script
  version: "1.26"

providers:
  - id: tinyidp
    import: github.com/manuel/tinyidp/pkg/xgoja/providers/tinyidp
    register: Register

runtime:
  modules:
    - provider: tinyidp
      name: tinyidp
      as: tinyidp

commands:
  - id: validate
    type: builtin.run
    name: validate

artifacts:
  - id: binary
    type: binary
    output: dist/tinyidp-script
```

A serving appliance will probably need a provider-owned command set rather than
pretending `builtin.run` owns HTTP lifecycle. Keep runtime module and command-set
responsibilities separate.

### TypeScript contract

Generate declarations for all public builders, branded references, contexts,
decisions, claim values, diagnostics, and test specs. The graph handle should be
opaque:

```ts
export interface TinyIDPModule { readonly v1: AuthAPIv1 }
export interface GraphHandle { readonly __tinyidpGraph: unique symbol }

export interface AuthAPIv1 {
  idp(name: string): IDPBuilder;
  readonly preset: Presets;
  readonly authn: AuthenticationBlocks;
  readonly policy: PolicyBlocks;
  readonly claims: ClaimBlocks;
  readonly decision: Decisions;
  test(name: string, spec: PolicyTestSpec): void;
}
```

Run `xgoja doctor`, `xgoja gen-dts`, and a generated-binary smoke test in CI.

## File-by-file implementation map

### New files/packages

```text
pkg/idpgraph/
  graph.go               schema and stable enums
  node.go                node/slot/callback specs
  outcome.go             full outcome algebra
  diagnostics.go         validation result model
  validate.go            structural validation orchestration
  validate_test.go

pkg/idpscript/
  compiler.go            one-shot compile API
  artifact.go            source/hash/graph artifact
  materializer.go        host reference resolution
  capabilities.go        descriptor and registry
  policy_pool.go         worker pool and invocation
  interrupt.go           deadline/Interrupt synchronization
  manager.go             atomic generation activation/drain
  tests.go               embedded policy test runner

pkg/gojamodules/tinyidp/
  module.go              explicit native loader
  api.go                 versioned namespace assembly
  builders.go            fluent objects backed by DraftCollector
  callbacks.go           deterministic callback registration
  codec.go               bounded JS <-> Go conversion
  typescript.go          declaration descriptor
  module_test.go         require("tinyidp") integration

pkg/xgoja/providers/tinyidp/
  provider.go            providerapi registration
  host_services.go       stable typed service lookup
  provider_test.go

examples/scripting/local-web/
  auth.js
  host.yaml
  xgoja.yaml
  README.md
```

### Existing files to change

| File | Change |
|---|---|
| `go.mod` | add pinned go-go-goja dependency; deliberately reconcile Go directive |
| `pkg/idp/contracts.go` | add authorization/claims contexts, decisions, policies, and `PolicySet` |
| `pkg/embeddedidp/options.go` | accept `PolicySet`/policy runtime and validate production readiness |
| `pkg/embeddedidp/provider.go` | own policy health/closure in readiness and lifecycle |
| `internal/fositeadapter/provider.go` | invoke authorization and claims policies at the documented seams |
| `internal/fositeadapter/session.go` | persist AMR/ACR from authentication result |
| `pkg/idpstore/types.go` | only add challenge records when full challenge phase starts |
| `pkg/idpstore/interfaces.go` | challenge store/atomic consume in full challenge phase |
| `pkg/sqlitestore/migrations/*` | challenge schema only in that phase |
| `internal/cmds/*` | validate/test/explain/serve-script command surfaces |
| `README.md` and docs | security model, syntax profile, examples, operations |
| `.github/workflows/*` | unit/race/lint/xgoja doctor/DTS/generated smoke gates |

## Detailed phased implementation plan

### Phase 0: contract spike and dependency decision

Purpose: remove unknowns before adding a public JS API.

Tasks:

1. Decide whether tiny-idp raises its `go` directive from 1.25.11 to at least
   the go-go-goja module's 1.26.1. The workspace and active toolchain already
   use Go 1.26, but the module contract must be explicit.
2. Pin a released or exact pseudo-version of go-go-goja; do not depend on the
   sibling workspace implicitly.
3. Build a 50-line explicit `tinyidp` module spike with implicit/data-only
   defaults disabled.
4. Confirm `require("fs")`, `require("exec")`, `require("database")`,
   `require("os")`, and `require("process")` all fail.
5. Compile one source with `goja.Compile`, load it in multiple owned runtimes,
   and race-test concurrent workers.
6. Implement and race-test a deadline interruption helper, including late
   interrupts and `ClearInterrupt` ordering.
7. Publish the supported syntax profile.

Gate:

- direct and `GOWORK=off` tests pass;
- forbidden-module tests pass;
- race detector passes interruption and worker ownership tests;
- dependency/toolchain decision is recorded.

### Phase 1: graph schema, descriptors, and validator

Purpose: make the graph the primary abstraction before polishing builders.

Tasks:

1. Implement pure-Go graph/node/slot/callback/test DTOs.
2. Implement descriptor registry and local-web native descriptors.
3. Add deterministic IDs and canonical JSON/hash.
4. Implement validation passes and stable diagnostic codes.
5. Model the current strict OIDC/password/consent/claims/issue path.
6. Build the `localWeb` graph directly in Go and materialize it into the current
   embedded provider without JavaScript.
7. Add graph snapshot and malformed-graph tests.

Gate:

- a pure-Go local-web graph completes existing strict integration tests;
- canonical graph output is deterministic;
- invalid references, cycles, incompatible slots, missing capabilities, and
  production-banned blocks fail with precise diagnostics.

### Phase 2: compile-only native module and fluent API

Purpose: make JavaScript an ergonomic graph authoring language, not the graph
runtime itself.

Tasks:

1. Implement `DraftCollector` and branded builder objects.
2. Add `require("tinyidp").v1`.
3. Implement initial namespaces and stable slots.
4. Export one graph from `module.exports` or an explicit collector finalization.
5. Bound source/config/object sizes and reject non-serializable values.
6. Add API TypeScript declarations.
7. Add `tinyidp script validate` and graph JSON output.
8. Add examples equivalent to the Go local-web graph.

Gate:

- JavaScript and pure-Go versions produce equivalent canonical graphs;
- no host resources open during compilation;
- forbidden modules remain unavailable;
- malformed fluent usage reports source-associated diagnostics.

### Phase 3: public authorization and claims seams

Purpose: provide useful application customization before general challenges.

Tasks:

1. Add immutable public subject/client/request/auth context DTOs.
2. Add allow/deny `AuthorizationPolicy`.
3. Add `ClaimsPolicy` with protected-claim and size/type validation.
4. Thread policies through `embeddedidp.Options` into Fosite adapter options.
5. Invoke authorization policy in `finishAuthorize` after native validation and
   before consent/code issuance.
6. Make `newOIDCSession` return errors and invoke claims policy before session
   persistence.
7. Propagate AMR/ACR into browser and OIDC sessions.
8. Add static native RBAC and claims nodes first.
9. Add tests for fresh login, existing session, `prompt=none`, consent,
   authorization deny, policy error, claim collision, refresh, and UserInfo.

Gate:

- existing strict and hosted conformance behavior remains intact;
- denied policy cannot produce a code;
- protected claims cannot be overwritten;
- ID token, refresh behavior, and UserInfo use consistent claims.

### Phase 4: policy runtime pool and capabilities

Purpose: enable bounded registered JavaScript callbacks.

Tasks:

1. Load exact source into each worker and verify callback fingerprint.
2. Implement bounded acquire and single-owner worker lifecycle.
3. Implement interruption, panic/timeout discard, and worker replacement.
4. Implement bounded immutable input projection and output codec.
5. Implement capability descriptor/registry/bindings.
6. Add authorization and claims callback nodes.
7. Add pool health/readiness and metrics.
8. Add saturation, timeout, exception, invalid result, capability failure, and
   concurrent flow tests.
9. Run race tests and production-shaped mixed load.

Gate:

- callbacks cannot access undeclared capabilities or ambient host modules;
- timeout stops JavaScript and replaces the worker;
- native capability calls have their own deadlines;
- saturation fails closed;
- the strict flow survives concurrent policy calls under race testing.

### Phase 5: embedded policy tests, explain, and atomic activation

Purpose: make policies operable rather than merely executable.

Tasks:

1. Collect `A.test` cases during compilation.
2. Provide deterministic fake capabilities and fixed test clock only in test
   runner.
3. Add `script test` and `script explain` commands.
4. Implement generation manager, warmup, atomic swap, draining, and rollback.
5. Add file-watch reload as an opt-in host feature.
6. Audit every activation attempt with source/graph hashes and redacted
   diagnostics.
7. Extend readiness with active generation and pool health.

Gate:

- failing policy test prevents activation;
- failed compile/materialization/warmup leaves old generation active;
- in-flight requests complete on the old generation;
- no generation-owned resource leaks after repeated reloads.

### Phase 6: xgoja/v2 provider and generated host

Purpose: support selected modules and distributable generated appliances.

Tasks:

1. Implement provider registration and config schema.
2. Implement TypeScript descriptor and generated declarations.
3. Add stable host-service key and typed lookup for serving resources.
4. Add `xgoja.yaml` compile-only example.
5. Add a provider-owned serving command or generated runtime-package example
   that delegates lifecycle to Go.
6. Add provider registry, module factory, host service, `xgoja doctor`, DTS, and
   build tests.

Gate:

- `xgoja doctor` passes;
- generated DTS compiles an example TypeScript policy;
- generated binary validates/tests a graph;
- serving example starts and stops through Go lifecycle, not JS listener code.

### Phase 7: typed authentication graph and challenge continuation

Purpose: unlock the full block algebra safely.

Tasks:

1. Refactor password and existing-session behavior behind native block
   interfaces.
2. Implement all outcome transition tables.
3. Add challenge store with hashed handles, one-time atomic consume, expiry, and
   generation binding.
4. Implement native interaction renderer contract.
5. Add `seq`, `all`, `firstAvailable`, `when`, and `choose` in that order.
6. Add explicit evidence, AMR, ACR, and step-up propagation.
7. Add one new native factor (recommended passkey) end to end before exposing a
   generic plugin factor API.
8. Threat-model downgrade, retry, replay, and factor-confusion paths.

Gate:

- every operator has a complete outcome truth table;
- rejected factors never fall through as `skip`;
- challenge replay and cross-generation resume fail;
- step-up produces correct AMR/ACR and native protocol behavior.

### Phase 8: additional native protocol/block packages

Add features in evidence-backed slices: upstream OIDC, device authorization,
token exchange, transaction authorization, CIBA, workload identity, and
multi-actor workflows. Each slice requires native protocol/storage support,
graph descriptors, capability policy, conformance or interoperability tests,
operations docs, and explicit production-profile review.

## Testing strategy

### Unit tests

- canonical graph IDs/hashes and deep-copy immutability;
- every validation pass and diagnostic code;
- builder method argument validation;
- callback registration/fingerprint determinism;
- bounded codecs and protected claims;
- outcome algebra truth tables;
- capability permission/effect checks;
- interruption synchronization and generation lifecycle.

### Runtime integration tests

Use the go-go-goja module-authoring pattern:

```go
factory := engine.NewRuntimeFactoryBuilder(
    engine.WithImplicitDefaultRegistryModules(false),
    engine.WithDataOnlyDefaultRegistryModules(false),
).WithModules(tinyidpModule).Build()

rt, err := factory.NewRuntime(engine.WithStartupContext(ctx))
defer rt.Close(context.Background())

_, err = rt.Owner.Call(ctx, "compile-test", func(_ context.Context, vm *goja.Runtime) (any, error) {
    return vm.RunString(`
      const A = require("tinyidp").v1;
      module.exports = A.idp("test")
        .use(A.preset.localWeb({store: A.ref.store("primary")}))
        .build();
    `)
})
```

Also assert that real host modules cannot be required.

### Strict-flow tests

Reuse and extend `internal/fositeadapter/provider_test.go` and
`pkg/embeddedidp/provider_test.go`:

- Authorization Code + S256 PKCE happy path;
- static and callback authorization allow/deny;
- claims in ID token and UserInfo;
- refresh behavior;
- existing browser session and `prompt=none`;
- consent required/remembered;
- policy timeout, exception, saturation, and unavailable capability;
- readiness when policy pool is degraded;
- provider close drains policy calls.

### Security tests

- redirect URI is validated before any policy-controlled redirect;
- script cannot alter `iss`, `sub`, `aud`, signature, nonce, timestamps, or key
  selection;
- raw secrets and credential records never occur in JS projections;
- forbidden modules and globals are absent;
- outputs exceed bounds fail closed;
- callback denial is distinct from infrastructure failure;
- capability effects are slot-compatible;
- challenge handles are hashed, one-time, bound, and expiring;
- reload cannot downgrade a production graph to development blocks.

### Concurrency and load

- `go test -race ./... -count=1`;
- concurrent policy calls never share one VM;
- pool saturation behavior is deterministic;
- timeout/interrupt/replacement under load;
- reload while requests are in flight;
- close while callbacks are active;
- native capability call cancellation;
- repeated failed reloads leak neither goroutines nor resources.

### CI command gate

```bash
go test ./... -count=1
go test -race ./... -count=1
GOWORK=off go test ./... -count=1
GOWORK=off make lint
xgoja doctor -f examples/scripting/local-web/xgoja.yaml
xgoja gen-dts -f examples/scripting/local-web/xgoja.yaml --out /tmp/tinyidp.d.ts
xgoja build -f examples/scripting/local-web/xgoja.yaml
```

Run the existing local and hosted OIDC conformance gates after strict-adapter
policy seams are added.

## Security and operations checklist

### Startup

- Verify script size, content hash, optional signature, owner, and permissions.
- Compile with an isolated module set.
- Validate graph and all host references.
- Run embedded tests.
- Warm all workers and compare callback fingerprints.
- Run existing production option validation.
- Emit activation audit record without source contents or secrets.

### Runtime

- Bounded pool and acquire timeout.
- Per-callback execution timeout and native capability deadline.
- Maximum calls, input/output bytes, nesting, and claim count.
- Fail closed on every policy infrastructure error.
- Stable audit reason codes.
- Readiness includes pool/generation health.
- Never log policy input wholesale.

### Reload

- New generation is fully ready before swap.
- Old generation drains with a deadline.
- Failed reload leaves old generation active.
- Generation/source hashes appear in diagnostics and audit.
- Continuations remain version-bound.

### Incident response

Operators need commands to:

- print active generation/hash/API version;
- validate a candidate without activating it;
- list callbacks/capabilities/effect classes;
- show redacted validation diagnostics;
- disable reload;
- roll back to the previous signed artifact;
- drain/replace a policy pool;
- correlate policy errors with audit request IDs.

## Decision records

### Decision: Go remains process host and trusted computing base

- **Context:** The research permits a script entrypoint, but protocols, secrets,
  storage, and lifecycle are security-critical.
- **Options considered:** JavaScript-hosted server; Go-hosted server with native
  `run()`; compile then Go starts.
- **Decision:** Compile then let Go materialize and start the service.
- **Rationale:** This reuses the production host, makes lifetimes explicit, and
  closes the compiler runtime before network service begins.
- **Consequences:** `.build()` is the v1 terminal builder method; a future
  `.run()` can only be a command-owned convenience around the same Go path.
- **Status:** proposed.

### Decision: graph is serializable and contains callback IDs, not functions

- **Context:** Goja values belong to one runtime and cannot be safely shared or
  persisted.
- **Options considered:** retain one compile VM; serialize closures; reload exact
  source into each worker.
- **Decision:** Store callback IDs/source hash in the graph and load exact source
  independently per worker.
- **Rationale:** Supports pools, deterministic warmup, inspection, and restart.
- **Consequences:** Callback registration must be deterministic and fingerprinted.
- **Status:** proposed.

### Decision: explicit `require("tinyidp")` is the primary API

- **Context:** Research suggests `Auth.v1`, while current go-go-goja/xgoja uses
  selected CommonJS modules as capability boundaries.
- **Options considered:** global only; module only; both.
- **Decision:** Module first; optional global alias later.
- **Rationale:** Selection is explicit and matches provider packaging and DTS.
- **Consequences:** Examples differ slightly from the colleague sketches.
- **Status:** proposed.

### Decision: no ambient go-go-goja modules

- **Context:** A plain engine builder can expose host-access modules.
- **Options considered:** default modules; safe middleware; only tiny-idp module.
- **Decision:** Disable implicit and data-only defaults; add every module
  explicitly.
- **Rationale:** Least authority and simpler review.
- **Consequences:** Scripts cannot read YAML/files/env unless a later reviewed
  compile input mechanism supplies data explicitly.
- **Status:** proposed.

### Decision: production scripts use opaque resource references

- **Context:** Research examples open SQLite and key handles from JavaScript.
- **Options considered:** script opens resources; JS config paths resolved by
  module; host resource registry.
- **Decision:** Production graph uses named references resolved by Go.
- **Rationale:** Keeps filesystem, secret, and closure ownership in the host.
- **Consequences:** Deployment has a host config in addition to `auth.js`.
- **Status:** proposed.

### Decision: first release exposes allow/deny and claims, not general challenges

- **Context:** Current strict engine lacks resumable challenge state and AMR
  propagation.
- **Options considered:** implement entire algebra immediately; fake unsupported
  blocks; stage narrow policies first.
- **Decision:** Stage static/authz/claims policy before challenge composition.
- **Rationale:** Delivers useful customization without inventing unsafe semantics.
- **Consequences:** Research examples with step-up/passkeys are roadmap examples,
  not v1 acceptance examples.
- **Status:** proposed.

### Decision: policy failure fails closed and unhealthy workers are replaced

- **Context:** Timeouts, exceptions, and partial runtime mutation are possible.
- **Options considered:** fall back to allow/base claims; reuse worker; fail
  closed and replace.
- **Decision:** Fail closed in production and replace after abnormal execution.
- **Rationale:** Identity policy cannot safely degrade open.
- **Consequences:** Pool sizing and readiness become production concerns.
- **Status:** proposed.

### Decision: computed claims cannot override protocol claims

- **Context:** A script-controlled issuer, audience, subject, or token timestamp
  would break native security invariants.
- **Options considered:** unrestricted merge; last-write wins; protected-name
  denylist and typed release policy.
- **Decision:** Native protected claims are immutable; custom claims are bounded
  and scope/client-filtered.
- **Rationale:** Keeps issuance and validation native.
- **Consequences:** Applications use namespaced/custom claim names.
- **Status:** proposed.

### Decision: raise or isolate the Go version deliberately

- **Context:** tiny-idp declares Go 1.25.11; current go-go-goja declares 1.26.1.
- **Options considered:** raise tiny-idp minimum; put scripting in another module
  or repository; wait for compatible dependency.
- **Decision:** Preferred implementation is to raise tiny-idp to Go 1.26.1 after
  release-policy approval; isolation is the fallback if 1.25 support is required.
- **Rationale:** The active workspace/toolchain already uses Go 1.26 and a split
  module would complicate public contracts and CI.
- **Consequences:** This is a release compatibility change and must be approved
  in Phase 0, not hidden in `go mod tidy`.
- **Status:** proposed.

### Decision: xgoja provider is required for generated binaries

- **Context:** Native module registration alone does not make a module selectable
  by xgoja/v2 generated hosts.
- **Options considered:** direct embedding only; global registry only; provider
  package.
- **Decision:** Ship both explicit direct registrar and xgoja provider.
- **Rationale:** Supports generated DTS, runtime selection, host services, and
  command sets using current APIs.
- **Consequences:** Provider tests and an example `xgoja.yaml` are release gates.
- **Status:** proposed.

## Alternatives considered

### JavaScript configuration that directly fills `embeddedidp.Options`

This is simpler but too low-level. It exposes implementation details, does not
model flow composition or slots, and makes future protocols hard to validate as
a whole. Use graph compilation instead.

### Keep all request policy in Go

This is the safest current state but misses the product goal of application-
specific, distributable policy. The proposed narrow contracts preserve Go
invariants while allowing bounded customization.

### One global Goja runtime protected by a mutex

This avoids a pool but makes every identity request head-of-line block behind
one callback and creates a single poisoned-state failure domain. Use a bounded
single-owner runtime per worker.

### New runtime for every request

This offers isolation but repeats script load and callback registration on the
critical path. Use warmed workers and replace abnormal ones. For extremely
sensitive callbacks, an external process can provide stronger isolation.

### Expose `fs`, `database`, or `fetch` with allowlists

These modules are broader than the identity-policy contract and make timeout,
secret, SSRF, and transaction behavior harder to reason about. Expose narrow
business capabilities instead.

### Store JavaScript continuations

Heap snapshots/closures are runtime-specific, non-portable, unauditable, and
unsafe across reloads. Persist only native versioned challenge state.

### Implement the whole colleague API before integration

This creates attractive names with undefined security semantics. Add APIs only
with native implementation, validation, and end-to-end tests.

## Risks and mitigations

| Risk | Consequence | Mitigation |
|---|---|---|
| Goja callback hangs | request and worker exhaustion | interrupt, native deadlines, bounded pool, replacement |
| native capability blocks | Goja interrupt cannot stop it | context-aware service timeout and call budget |
| ambient module accidentally enabled | filesystem/process/network access | explicit factory settings + negative require tests |
| policy mutates shared JS state | cross-request influence | per-worker isolation; freeze API/input; replace abnormal workers; document state prohibition |
| claims leak sensitive data | token disclosure | redacted context, capability outputs, protected names, client/scope release policy, bounds |
| reload changes semantics mid-flow | inconsistent authentication | generation binding and atomic activation |
| callback fingerprints differ | non-deterministic policy | warmup fingerprint check; reject activation |
| graph/API evolves | stored artifact incompatibility | schema/API versions and explicit migrations |
| xgoja and tiny-idp dependency drift | build break or behavior mismatch | pinned versions, `GOWORK=off`, provider/DTS/generated smokes |
| product scope explodes | monolithic IdP clone | native block packages and staged release sequence |
| in-process untrusted code | host compromise/DoS | trusted scripts only; external process boundary for untrusted authors |

## Open questions requiring owner decisions

1. Is raising the minimum Go version to 1.26.1 acceptable for the next tiny-idp
   release?
2. Is the first supported authoring model trusted operators only, or must tenant
   administrators submit scripts? The latter requires a process sandbox and a
   different threat model.
3. Should computed claims be fixed at authorization time (recommended for v1)
   or recomputed at UserInfo? Recompute creates consistency and capability
   availability problems.
4. Which custom claim naming policy is required: arbitrary non-protected names,
   URI names, or product namespace prefixes?
5. What is the production callback timeout and pool-sizing envelope based on
   expected peak login traffic?
6. Which capabilities are needed for the first real application integration?
7. Should a graph configure clients, or should clients remain provisioned only
   through the admin/store plane? This guide recommends store/admin ownership
   for production and graph fixtures only for test mode.
8. Should `testLab` remain in the same module with production-profile bans, or a
   separate development module for stronger exclusion?
9. Which new native factor should prove the challenge model first? Passkey is
   strategically strongest but has the largest browser/storage surface.
10. How many previous generations may remain available for continuation resume,
    and for how long?

## Intern implementation checklist

Before coding:

- [ ] Read the source research and latest diary.
- [ ] Run the baseline tests.
- [ ] Confirm current go-go-goja commit/version and xgoja help docs.
- [ ] Obtain the Go-version decision.
- [ ] Write failing negative tests for forbidden modules.

For every phase:

- [ ] Keep domain logic out of module loaders.
- [ ] Use lowerCamelCase JS keys.
- [ ] Add compile-time interface assertions.
- [ ] Add pure-Go tests and `require("tinyidp")` integration tests.
- [ ] Run direct and `GOWORK=off` tests.
- [ ] Run race tests for runtime ownership changes.
- [ ] Update TypeScript declarations and examples with the same API.
- [ ] Update graph schema/API versions intentionally.
- [ ] Record failures and decisions in the diary.

Before production claim:

- [ ] Existing strict OIDC tests and conformance pass.
- [ ] No ambient host modules are available.
- [ ] Policy timeout, saturation, exception, and invalid outputs fail closed.
- [ ] Protected claims cannot be changed.
- [ ] Secrets/credentials never enter JS.
- [ ] Reload is atomic and rollback tested.
- [ ] Readiness includes graph/pool health.
- [ ] xgoja doctor, DTS, and generated binary smokes pass.
- [ ] Deployment and incident-response docs are reviewed.

## Acceptance criteria for the ticket implementation

The eventual implementation can be called complete when:

1. A checked-in `auth.js` compiles to a deterministic graph.
2. The graph materializes through public tiny-idp APIs into a strict provider.
3. A complete Authorization Code + S256 PKCE flow succeeds.
4. A JavaScript authorization callback can allow and deny without seeing raw
   protocol objects.
5. A JavaScript claims callback adds bounded custom claims consistently to ID
   token and UserInfo while protected claims remain native.
6. No filesystem, process, environment, database, exec, or network module is
   available unless explicitly added by a separate reviewed profile.
7. Timeouts, saturation, exceptions, and invalid outputs fail closed and do not
   reuse unhealthy workers.
8. Embedded policy tests gate activation.
9. A failed reload leaves the previous generation live.
10. Direct embedding and xgoja/v2 provider packaging both pass CI-style tests.
11. Baseline tiny-idp tests, race suite, lint, and conformance gates pass.
12. Security and operations documentation explains trust, limits, metrics,
    reload, rollback, and incident response.

## API and file references

### Current tiny-idp

- `pkg/embeddedidp/options.go:40-190` — public construction and production
  validation.
- `pkg/embeddedidp/provider.go:19-92` — provider lifecycle and handler.
- `pkg/idp/contracts.go:16-168` — current policy/authentication seams.
- `pkg/idpstore/types.go:14-184` — durable domain and secret separation.
- `pkg/idpstore/interfaces.go:20-198` — store and atomic-operation contracts.
- `pkg/idpstore/claims.go:3-35` — current fixed claim release.
- `internal/fositeadapter/provider.go:180-220` — Fosite composition.
- `internal/fositeadapter/provider.go:340-515` — authorization, token, UserInfo.
- `internal/fositeadapter/provider.go:547-571` — OIDC session claims.
- `internal/fositeadapter/provider.go:731-766` — consent and code issuance seam.
- `internal/fositeadapter/session.go:14-41` — browser-session persistence.
- `internal/cmds/serve_production.go:106-205` — production host lifecycle.

### Current go-go-goja and xgoja

- `../go-go-goja/pkg/engine/factory.go:32-288` — immutable runtime factory and
  owned runtime construction.
- `../go-go-goja/pkg/engine/module_specs.go:51-78` — explicit native registrar.
- `../go-go-goja/pkg/engine/module_middleware.go:21-47` — safe/only selection.
- `../go-go-goja/pkg/engine/runtime.go:32-151` — lifecycle, close, interruption.
- `../go-go-goja/pkg/runtimeowner/runner.go:90-211` — serialized runtime calls.
- `../go-go-goja/pkg/xgoja/providerapi/module.go:13-50` — provider module API.
- `../go-go-goja/pkg/xgoja/app/factory.go:34-140` — selected-module runtime
  creation with defaults disabled.
- `../go-go-goja/pkg/xgoja/providers/hostauth/hostauth.go:30-105` — narrow
  host-service provider pattern.
- `../go-go-goja/examples/xgoja/14-generated-runtime-package/` — generated
  runtime package and Go-owned execution example.

### Source research and standards

- `sources/01-colleague-identity-microkernel-research.md` — product framing,
  fluent API sketches, outcome algebra, examples, security boundaries, and
  release sequence.
- OpenID Connect Core 1.0 —
  <https://openid.net/specs/openid-connect-core-1_0-final.html>.
- OAuth 2.0 Security Best Current Practice (RFC 9700) —
  <https://www.rfc-editor.org/rfc/rfc9700.txt>.
- Goja repository and runtime notes — <https://github.com/dop251/goja>.
- xgoja installed help: `xgoja help xgoja-v2-reference` and
  `xgoja help provider-runtime-config-and-host-services`.

## Final implementation guidance

Start with the graph, not the fluent API. If the native graph cannot model and
validate the current strict flow in pure Go, JavaScript builders will only hide
that deficiency. Then add the compiler as a narrow adapter, add public
authorization and claims seams, and only then introduce pooled callbacks.

The most important invariant is simple: **JavaScript describes or decides; Go
validates, persists, challenges, signs, issues, and serves.** Every code review
should be able to point to which side of that sentence a new feature belongs.
