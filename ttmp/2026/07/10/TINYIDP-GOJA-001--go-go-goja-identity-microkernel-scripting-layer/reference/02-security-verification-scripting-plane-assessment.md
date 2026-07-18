---
Title: Security verification scripting plane assessment
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict authorization state machine that verification scenarios will exercise
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md
      Note: Full static, dynamic, instrumentation, and release assurance architecture
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Base graph, policy runtime, capability, and challenge design extended by this assessment
ExternalSources:
    - https://ecommons.cornell.edu/items/5a936aa1-8a4f-41df-bc17-f2db479cf33e
    - https://www.usenix.org/conference/usenixsecurity15/technical-sessions/presentation/de-ruiter
    - https://www.usenix.org/conference/usenixsecurity22/presentation/ba
Summary: Assessment and API proposal for adding declarative security scenarios and assertions to the Goja identity microkernel without extending the production policy trust boundary.
LastUpdated: 2026-07-10T12:00:00-04:00
WhatFor: Deciding whether and how tiny-idp validation plugins should compile security verification scenarios, invariants, failpoints, and trace assertions.
WhenToUse: Read before extending the identity scripting API with hooks, policy tests, validation plugins, scenario drivers, or test-only host capabilities.
---




# Security verification scripting plane assessment

## Conclusion

The proposed scripting layer can support useful validation plugins, but the
behavioral verification system should be a separate scripting plane. It should
not be implemented as general production request hooks and should not share the
production policy runtime's capability registry.

The correct extension has two parts:

1. Add structural assertions to the compile-time identity graph API. These
   inspect pure graph data and reject unsafe configuration before activation.
2. Add a compile-only `require("tinyidp/verify").v1` module. Verification scripts
   describe scenarios, mutations, schedules, failpoints, and named assertions.
   The module produces an immutable pure-Go `VerificationPlan`. A Go runner owns
   HTTP execution, fake time, store inspection, fault injection, trace analysis,
   and final verdicts.

This division preserves the main design's strongest decisions: Go remains the
trusted computing base; JavaScript receives no raw Fosite request, store, key,
password, token, or ambient host module; callbacks are bounded and single-owner;
protected protocol claims remain native; and challenge continuations are
Go-owned, versioned, one-time state.

## 1. Why language support is useful

Security test suites become difficult to review when every scenario is expressed
as low-level HTTP construction. A small declarative language can make the
essential state transitions visible:

```text
given an authenticated browser session
begin authorization with prompt=login
observe a required fresh-login action
submit an empty login
assert that no code was issued
assert that no accepted terminal event occurred
```

JavaScript is already the proposed identity composition language. Reusing Goja
for verification authoring gives the project versioned builders, named native
assertions, TypeScript declarations, graph/source hashes, and xgoja packaging.
It also lets product-specific configurations ship scenarios next to the graph
that they are meant to validate.

The language must remain a description language. If it receives a live database,
HTTP response writer, Fosite requester, signing service, or authorization result
mutator, it stops being a verification layer and becomes another protocol
implementation.

## 2. Relationship to the current design

The main design at
`design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md`
already establishes the required foundation:

- Go, Fosite, stores, keys, authentication, audit, rate limiting, and HTTP
  lifecycle remain the TCB (`:52-59`).
- Configuration compiles once into an immutable graph and request callbacks are
  selected and bounded (`:61-84`).
- Scripts never receive secrets or raw protocol/store objects (`:390-400`).
- Implicit and data-only module registries are disabled (`:460-479`).
- Each Goja VM has one owner and abnormal workers are interrupted and replaced
  (`:485-506`).
- `Tests []PolicyTest` is proposed for embedded callback examples (`:525-542` and
  the graph/API sections).
- Future challenge state is a Go-owned, hashed, expiring, atomically consumed
  record (`:1339-1368`).
- Security tests already require protected claims, forbidden-module checks,
  bounded outputs, one-time challenge handles, and safe reload (`:1819-1830`).

The verification plane extends these decisions. It does not weaken them. The
important change is scope: `PolicyTest` covers pure callback input/output, while
`VerificationPlan` covers multi-request protocol behavior and durable state.

## 3. Three distinct runtime profiles

Using one runtime profile for every purpose is unsafe because verification needs
authority that production scripts must never possess.

| Profile | Purpose | JS-visible authority | Output |
|---|---|---|---|
| Identity compiler | Build and validate immutable identity graph | fluent builders, refs, structural assertions | `idpgraph.Graph` |
| Production policy | Evaluate selected bounded policy callbacks | redacted request/user context and reviewed capabilities | allow/deny or bounded custom claims |
| Verification compiler | Describe tests and native assertions | scenario DSL and symbolic test capability references | `VerificationPlan` |

The verification compiler still receives no live fake clock, store, failpoint
controller, or server. Those are owned by the Go runner after the JS runtime has
closed. A symbolic step such as `V.clock.advance("5m")` becomes data in the plan;
it does not mutate a host clock while JavaScript runs.

Production binaries may include the verification compiler for an explicit
`tinyidp script test` command, but the serving runtime factory must not register
the module. Stronger isolation is to package it in a separate test command or
generated xgoja command set. In both cases, negative integration tests must prove
that a production callback cannot `require("tinyidp/verify")`.

## 4. Authority model

### 4.1 Permitted verification operations

A verification plan may request that the Go runner:

- create synthetic clients, users, browser sessions, and interaction fixtures;
- drive HTTP requests through an in-process strict provider;
- advance an injected fake clock;
- select a named failpoint defined by native code;
- release a named scheduling point;
- inspect a redacted typed store snapshot;
- collect a versioned security trace;
- evaluate a registered native invariant; and
- emit a reproducible counterexample and evidence bundle.

### 4.2 Forbidden operations

A verification script cannot:

- open an arbitrary database or file;
- read environment variables;
- use network, process, or exec modules;
- access real passwords, keys, cookies, codes, or bearer tokens;
- register a new production authorization decision callback;
- choose raw SQL or arbitrary store mutations;
- disable a native assertion;
- catch an assertion violation and convert it to pass;
- rewrite the observed trace;
- select an unregistered redirect target; or
- run automatically on a production request.

Test fixtures use synthetic secret material internally. JS refers to fixture IDs,
not the values.

## 5. API proposal

### 5.1 JavaScript

```js
const V = require("tinyidp/verify").v1;

module.exports = V.suite("authorization interaction")
  .tags("release", "oidc", "interaction")

  .scenario("prompt login requires a new authentication", s => s
    .given(V.fixture.browserSession({ user: "alice", authAge: "5m" }))
    .when(V.authorize.begin({
      client: "web-app",
      responseType: "code",
      scopes: ["openid", "email"],
      pkce: "S256",
      prompt: "login"
    }))
    .then(V.expect.interaction({ requires: ["fresh_login"] }))
    .when(V.interaction.submit({ login: "", password: "" }))
    .then(V.expect.notAccepted())
    .assert(V.invariant.freshAuthenticationBeforeIssuance())
    .assert(V.invariant.exactlyOneTerminalOutcome()))

  .scenario("authorization response writes are atomic", s => s
    .given(V.fixture.authenticatedInteraction({ consent: "approved" }))
    .forEachFailpoint(V.failpoints.group("authorization_response"))
    .when(V.interaction.consume())
    .assert(V.invariant.protocolStateAllOrNone())
    .assert(V.invariant.noOrphanMutation()))

  .scenario("concurrent consumption is one-time", s => s
    .given(V.fixture.readyInteraction())
    .parallel(
      V.interaction.consume(),
      V.interaction.consume()
    )
    .assert(V.invariant.exactlyOneConsumeSucceeds())
    .assert(V.invariant.linearizable("interaction.consume.v1")))

  .build();
```

The DSL should not accept arbitrary callbacks for authoritative expectations.
Named native assertions produce stable semantics, documentation, and evidence
IDs. A later `observe` callback may format redacted data, but its return value
must not determine pass/fail.

### 5.2 Go data model

```go
type VerificationPlan struct {
    SchemaVersion string       `json:"schemaVersion"`
    APIVersion    string       `json:"apiVersion"`
    SourceHash    [32]byte     `json:"sourceHash"`
    Suites        []Suite      `json:"suites"`
    Required      Requirements `json:"required"`
}

type Suite struct {
    ID        string
    Name      string
    Tags      []string
    Scenarios []Scenario
}

type Scenario struct {
    ID         string
    Fixtures   []FixtureRef
    Steps      []Step
    Assertions []AssertionRef
    Bounds     Bounds
}

type Step struct {
    Kind       StepKind
    Parameters json.RawMessage
    Parallel   []Step
}

type AssertionRef struct {
    ID      string
    Version string
    Config  json.RawMessage
}
```

`Validate` rejects duplicate IDs, unknown step/assertion versions, impossible
parallel nesting, excessive plans, forbidden fixture classes, unavailable runner
features, and assertions incompatible with the scenario's observation level.

### 5.3 Native assertion registry

```go
type Assertion interface {
    ID() string
    Version() string
    Evaluate(context.Context, Evidence) Result
}

type AssertionRegistry interface {
    Resolve(id, version string) (Assertion, bool)
}
```

`Evidence` is immutable and contains redacted HTTP observations, model state,
security events, and typed store snapshots. It has no methods that mutate the
provider. The runner always evaluates required baseline assertions in addition
to script-selected assertions.

## 6. Structural graph assertions

Simple activation-time checks belong in the existing identity compiler, not the
behavioral verification runner. Proposed builder operations:

```js
const A = require("tinyidp").v1;

module.exports = A.idp("notes")
  .use(A.preset.localWeb(/* ... */))
  .assert(A.invariant.pkceRequired("S256"))
  .assert(A.invariant.durableAudit())
  .assert(A.invariant.productionRateLimiter())
  .assert(A.invariant.noAmbientCapabilities())
  .assert(A.invariant.protectedClaimsNative())
  .build();
```

These methods add declarative requirements to the graph. The Go graph validator
owns their semantics. A script cannot implement `durableAudit` by returning true.

## 7. Hook design

The word "hook" is ambiguous. Define the supported categories explicitly.

### 7.1 Production decision callbacks

These are already part of the proposed policy design: bounded authorization and
computed-claims callbacks over redacted immutable input. They are security
policy, not verification.

### 7.2 Native instrumentation points

These are Go calls that emit security events immediately before or after typed
state transitions. The monitor depends on their completeness. JavaScript cannot
register, replace, or suppress them.

### 7.3 Offline verification lifecycle hooks

The runner may support `beforeScenario`, `afterStep`, `afterScenario`, and
`onCounterexample` for formatting or external harness integration. Hooks receive
bounded redacted values. They cannot mutate the provider or determine the native
verdict.

### 7.4 Test scheduling and failpoints

These are symbolic plan steps resolved by the Go runner to an allowlisted native
point. JavaScript cannot name an arbitrary Go function, SQL statement, or source
line.

## 8. Execution flow

```text
verify.js
   |
   v
isolated compile-only Goja runtime
   |  module: tinyidp/verify only
   |  bounded source/time/plan
   v
VerificationPlan ---- validate/hash/snapshot ---- close Goja
   |
   v
Go runner creates isolated fixtures and strict provider
   |
   +--> fake clock
   +--> typed failpoint adapters
   +--> controlled scheduling probes
   +--> in-memory audit/trace collector
   +--> redacted store inspector
   |
   v
execute steps --> native assertions --> shrink/replay if generated
   |
   v
evidence: plan hash, graph hash, seed, schedule, failpoints, trace, verdicts
```

Each scenario runs with a fresh store unless it explicitly tests restart or
upgrade behavior. Total plan execution is cancellable. Timeouts are classified
as harness failure, expected injected failure, or invariant failure; they are not
silently converted to denial/pass.

## 9. Runtime verification and research rationale

Security-automata research shows that an online monitor can enforce properties
only when it sees the complete action stream and controls the violating
transition. Therefore production issuance guards remain native. The verification
plane can evaluate richer trace properties offline because it observes complete
isolated scenarios, but it still cannot prove behavior outside its explored
inputs.

Protocol-state fuzzing and stateful greybox fuzzing justify exposing native state
and transition IDs to the test runner. The feedback signal should include model
state, required actions, terminal outcomes, store mutation classes, event
coverage, and Go code coverage. It must not include secret values.

Dynamic invariant mining can propose new named assertions from traces. A proposed
relation enters the native registry only after review and negative-test design.
The script DSL should not auto-promote mined relations.

## 10. Security tests for the verification plane

1. `require("fs")`, `require("os")`, `require("database")`, network, and exec
   modules fail in all three profiles unless separately and explicitly approved.
2. Production policy runtime cannot require `tinyidp/verify`.
3. Verification compiler cannot require production policy capabilities.
4. Verification source receives no live host object or secret.
5. Unknown assertion/failpoint/schedule IDs fail plan validation.
6. A script cannot omit mandatory baseline assertions.
7. A script exception, timeout, oversized plan, or nondeterministic registration
   fails closed and produces no runnable artifact.
8. Identical source and host feature set produce identical plan fingerprints.
9. A failed assertion cannot be caught or rewritten by JavaScript.
10. Plan and graph generation mismatch is rejected.
11. Redacted observations remain free of passwords, cookies, codes, tokens, raw
    CSRF, private keys, and full request objects.
12. Verification workers and fixtures are closed after cancellation and repeated
    failures without goroutine, file, or database leakage.

## 11. Implementation phases

### Phase A: contracts

- Define `VerificationPlan`, step kinds, assertion refs, bounds, and diagnostics
  in a pure Go package.
- Define runner features and compatibility/version rules.
- Define mandatory native assertions and evidence format.
- Decide whether the verification provider ships in production binaries or only
  dedicated test commands.

### Phase B: compile-only module

- Implement `tinyidp/verify` as an explicit native module registrar.
- Disable all implicit/data-only default modules.
- Add fluent builders, immutability, deterministic IDs, source hashing, and DTS.
- Add runtime integration tests using the actual engine and `require`.
- Add negative module-resolution tests.

### Phase C: Go runner

- Build fixture factory, in-process strict-provider driver, fake clock, trace
  collector, redacted store inspector, and assertion registry.
- Execute deterministic sequential scenarios first.
- Produce Markdown, JSON, and JUnit evidence without secret values.

### Phase D: adversarial control

- Add typed failpoint adapters and mutation-boundary enumeration.
- Add scheduling probes and concurrent steps.
- Add Porcupine assertions for one-time operations.
- Add generated/scaled action sequences, shrinking, and corpus persistence.

### Phase E: activation and release integration

- Link verification plan/source/graph hashes in activation artifacts.
- Run pure policy tests during warmup.
- Run behavioral verification in CI and explicit pre-release/pre-activation
  commands, not synchronously on ordinary process startup.
- Add evidence to the production candidate ledger and reMarkable bundle.

## 12. Decision records

### Decision: use a separate verification module and runtime profile

- **Context:** Verification needs fake time, failpoints, scheduling, traces, and
  store observations that production policy must never access.
- **Options considered:** extend `tinyidp`; use production hooks; separate
  `tinyidp/verify` compiler.
- **Decision:** Add a separate compile-only module and Go-owned runner.
- **Rationale:** Capability separation is clearer and can be tested negatively.
- **Consequences:** Provider packaging and generated commands need explicit
  profile selection.
- **Status:** proposed.

### Decision: scripts reference native assertions

- **Context:** Arbitrary JS predicates can redefine pass/fail and are difficult to
  version or audit.
- **Options considered:** arbitrary predicates; only hard-coded suites; named
  native assertions with declarative configuration.
- **Decision:** Use a native assertion registry; allow non-authoritative custom
  formatting later.
- **Rationale:** Stable semantics and evidence IDs remain in the Go TCB.
- **Consequences:** New authoritative assertions require Go code and tests.
- **Status:** proposed.

### Decision: no production mutation hooks for verification

- **Context:** Hooks can observe, alter, delay, or suppress the very transition
  being verified.
- **Options considered:** general before/after callbacks; observer-only production
  callbacks; native events plus offline lifecycle hooks.
- **Decision:** Native instrumentation in production; lifecycle hooks only in the
  offline runner.
- **Rationale:** Complete mediation and event integrity remain native.
- **Consequences:** Production experiments use shadow monitoring, not injected JS
  behavior.
- **Status:** proposed.

## 13. Immediate next work

The first implementation should not begin with the DSL. First repair and type the
authorization interaction in Go, define the event schema, and build three native
regressions for `prompt=login`, expired `max_age`, and invalid `max_age`. Then
implement a minimal `VerificationPlan` capable of compiling and running those
same scenarios. This order ensures that the language describes real stable
semantics rather than freezing the current hidden-field behavior into an API.

The full assurance design, research map, analyzer plan, trace schema, and release
phases are in the production implementation ticket at
`ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`.
