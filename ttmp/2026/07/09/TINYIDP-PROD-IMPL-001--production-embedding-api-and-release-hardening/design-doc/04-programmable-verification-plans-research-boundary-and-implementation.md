---
Title: 'Programmable Verification Plans: Research Boundary and Implementation'
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - auth
    - architecture
    - xgoja
    - research
    - identity
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/gojamodules/verify/module.go
      Note: |-
        Only JavaScript-visible module and lower-camel plan decoder
        Only JavaScript-visible compile-time capability
    - Path: repo://internal/gojaverify/compiler.go
      Note: |-
        Time- and source-bounded isolated Goja compiler
        Isolated compiler and resource controls
    - Path: repo://internal/securitytrace/trace.go
      Note: |-
        Native reference monitor whose authority is intentionally not delegated to scripts
        Native reference monitor kept outside script authority
    - Path: repo://pkg/verifyplan/plan.go
      Note: |-
        Versioned plain-data plan, native driver contract, assertion registry, and runner
        Versioned plan and native execution boundary
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Parent scripting-layer design whose compile-then-run boundary this verifier reuses
ExternalSources:
    - https://www.usenix.org/legacy/events/sec2000/full_papers/evans/evans.pdf
    - https://www.cs.utexas.edu/~isil/cs389L/monitoring.pdf
Summary: Research-to-code decision record for using Goja as a bounded verification-plan authoring language while keeping scenario effects, observations, and security verdicts native in Go.
LastUpdated: 2026-07-10T20:49:15-04:00
WhatFor: Reviewing or extending programmable tiny-idp security scenarios without turning JavaScript into protocol or invariant authority.
WhenToUse: Before adding verification steps, assertion types, JavaScript modules, live capabilities, scenario drivers, or production monitor hooks.
---


# Programmable Verification Plans: Research Boundary and Implementation

## Executive summary

tiny-idp now has a small programmable verification-plan compiler. JavaScript is
used because it is concise for assembling suites, scenarios, steps, and assertion
references. It is not used to execute provider operations or decide whether a
security property holds. The compiler returns a versioned, source-bound Go data
structure. A native driver performs each step, records typed observations, and
dispatches assertions from a Go-owned registry.

This division follows two bodies of work. Runtime verification treats an event
trace and a monitor as explicit objects; the monitor's verdict has meaning only
relative to a defined property. Capability-oriented systems constrain authority
by controlling which references a component receives. Applied here, these ideas
produce a narrow rule:

> The script may select and parameterize predeclared experiments. It receives no
> reference that can perform an experiment or declare a security verdict.

The implementation is useful today for reproducible scenario authoring and will
later support a strict-provider driver. It is not an operating-system sandbox,
does not make in-process untrusted JavaScript safe, and does not replace native
tests, static analysis, fault injection, or the security trace monitor.

## The system in one diagram

```text
verification.js
    |
    | require("tinyidp/verify").v1.plan(plainData)
    v
+--------------------------+
| isolated compiler runtime|
| source limit + deadline  |
| no fs/net/env/db/provider|
+------------+-------------+
             |
             | JSON normalization, schema validation,
             | SHA-256 source binding
             v
+--------------------------+        +-------------------------+
| verifyplan.Plan          |------->| native Go Runner        |
| suites/scenarios/steps   |        |                         |
| assertion IDs + config   |        | Driver.Execute(step)    |
+--------------------------+        | AssertionRegistry[id@v] |
                                    +-----------+-------------+
                                                |
                              typed observations| native verdicts
                                                v
                                    +-------------------------+
                                    | results and evidence    |
                                    +-------------------------+
```

There is no arrow from JavaScript to the provider, store, key material, network,
clock, or assertion function.

## Academic concepts and local interpretation

### Runtime verification and monitor authority

Runtime verification checks a trace against a formal or executable property.
The important engineering consequence is separation: instrumentation emits
facts; a monitor consumes facts; the monitor implementation defines the verdict.
`internal/securitytrace/trace.go` follows this structure for authorization
transitions. It is versioned, partitions state by opaque interaction identifier,
and rejects illegal temporal orders.

A JavaScript callback that receives the provider and returns `true` or `false`
would collapse all three roles. It could omit facts, mutate the system, and
define its own success. Such a callback is useful as an application policy hook,
but it is not independent security verification. The verification-plan API
therefore refers to native assertions by `(id, version)` and supplies only data
configuration.

### Capability discipline

Authority follows references. A runtime with a SQL handle, HTTP client, raw
provider pointer, secret accessor, or assertion registration API can exercise
that authority regardless of naming conventions. Conversely, a CommonJS runtime
whose only module constructs plain data has no direct path to those effects.

`internal/gojaverify/compiler.go` creates a fresh Goja runtime and a module
registry whose fallback loader rejects every ambient module. It registers only
`tinyidp/verify`. The module returns normalized plan data and exposes no host
object with methods. Negative tests prove that `require("fs")` fails.

This is defense by object graph reduction, not process isolation. Goja shares the
host address space. A future module that exposes a live Go object would expand
the trusted computing base and must be treated as a security design change.

### Typestate and explicit action alphabets

Protocol tests become easier to reason about when their operations form a typed
alphabet and their observations are distinct from commands. `verifyplan.Step`
currently carries a version-independent `kind` plus JSON parameters so the first
schema can be introduced before the complete strict-provider action algebra is
stable. The driver is responsible for decoding each known kind into a typed Go
request and rejecting unknown kinds.

The intended execution shape is:

```text
state := NewNativeScenarioState()
for step in scenario.steps:
    action := NativeDecoder[step.kind].Decode(step.parameters)
    observation := StrictProviderDriver.Execute(state, action)
    observations.append(observation)

for assertion in scenario.assertions:
    check := NativeAssertions[assertion.id + "@" + assertion.version]
    check(assertion.config, observations)
```

The dynamic outer envelope enables schema evolution; typed decoding at the
effect boundary prevents scripts from invoking arbitrary methods.

### Provenance and reproducibility

`Compile` binds the plan to the SHA-256 digest of its complete source. A stored
result can therefore identify the script that selected the scenario, even when
two scripts compile to the same plan. This digest is provenance, not a signature:
it detects accidental or unrecorded source differences but does not establish
an author or trusted origin.

Property tests and fuzzers require additional provenance: random seed, shrunk
counterexample, binary commit, plan schema, driver version, and assertion
versions. Those fields belong in the future evidence envelope rather than in the
immutable plan itself.

## Concrete implementation map

### `pkg/verifyplan`

This package has no Goja or provider dependency. It defines:

- schema `tinyidp.verify/v1`;
- `Plan`, `Suite`, `Scenario`, `Step`, and `Assertion` data;
- count and serialized-size limits;
- source digest binding;
- the native `Driver` interface;
- versioned native assertion functions;
- per-scenario results and observations.

The runner fails closed when its driver is absent, the plan is invalid, or an
assertion ID/version is not registered. A failed step stops that scenario before
assertions run. JavaScript functions cannot appear in the normalized plan
because the Goja value must survive JSON encoding and typed decoding.

### `internal/gojamodules/verify`

The module implements the go-go-goja `modules.NativeModule` contract and is
registered under `tinyidp/verify`. Its `v1.plan(spec)` function:

1. requires exactly one argument;
2. JSON-encodes the exported Goja value;
3. decodes it into `verifyplan.Plan`;
4. supplies the v1 schema when omitted;
5. validates default limits;
6. returns a normalized plain object.

It does not retain a runtime, register callbacks, execute assertions, or expose
a host service.

### `internal/gojaverify`

The compiler supplies three resource controls:

- source size, default 64 KiB;
- wall-clock/context deadline, default 250 ms;
- output shape and JSON size limits.

The deadline calls `Runtime.Interrupt`, which stops non-terminating JavaScript in
the tested loop case. Package tests run with the repository workspace and with
`GOWORK=off`, proving that the tagged go-go-goja dependency is sufficient and
the sibling checkout is not accidentally required.

## What is proved and what is not

The current tests establish:

- a valid script can load only the named module and compile a plan;
- ambient CommonJS module loading is rejected;
- an unbounded loop is interrupted by the configured deadline;
- the plan is source-bound and schema-validated;
- a compiled plan is executed by a native driver;
- assertion lookup and verdict execution occur in Go.

They do not establish:

- hard heap or CPU quotas against hostile in-process code;
- process isolation;
- absence of all Goja implementation vulnerabilities;
- completeness of a strict-provider scenario driver;
- soundness of every future native assertion;
- deterministic JavaScript in the presence of built-in time or randomness;
- authenticity of the source hash.

The memory caveat is important. A deadline is not a heap limit, and an in-process
runtime can allocate aggressively before interruption. Production use should
accept only trusted, reviewed plan sources. If hostile multi-tenant scripts ever
become a requirement, compilation belongs in a separately resource-limited
process with an authenticated plan handoff.

## Relationship to the identity microkernel proposal

The broader `TINYIDP-GOJA-001` design compiles identity configuration into an
immutable graph and recommends that Go start the system only after compilation.
The verifier reuses this compile-then-run pattern but has a different authority
profile:

| Layer | Script output | Native effect | Acceptable script authority |
|---|---|---|---|
| Identity configuration | immutable identity graph | materialize provider | selected trusted policy/configuration |
| Verification authoring | immutable test plan | execute scenarios/assertions | select and parameterize tests only |
| Runtime monitor | none | observe facts and issue verdict | no script authority in v1 |

Combining all three into one live scripting layer would make the same code able
to configure behavior, exercise it, and judge it. Keeping separate modules and
data types supports independent review and prevents a policy callback from being
mistaken for an assurance oracle.

## Extension rules

Adding a step requires:

1. a stable kind and typed native parameter decoder;
2. a documented precondition and possible observations;
3. strict rejection of unknown or surplus security-relevant fields;
4. deterministic tests, including invalid input;
5. a trace/evidence mapping when the step mutates security state.

Adding an assertion requires:

1. a globally documented ID and explicit version;
2. a typed native config decoder;
3. positive and negative fixtures;
4. a statement of soundness and completeness limits;
5. a decision on whether it consumes driver observations, security events, or
   persistent-state snapshots.

Adding any JavaScript-visible capability requires a threat-model revision. The
review must enumerate reachable methods and data, concurrency ownership,
cancellation, secret exposure, persistence effects, and failure behavior.

## Next implementation phases

1. Define typed strict-provider actions for authorization begin, interaction
   submit, consent decision, token exchange, refresh, UserInfo, and time advance.
2. Define observations that retain HTTP/protocol semantics without credentials
   or token values.
3. Build a native driver around an in-memory strict provider and injected clock.
4. Register native assertions for fresh authentication, consent before issuance,
   terminal uniqueness, continuation opacity, and token-family rotation.
5. Add metamorphic transforms such as query ordering, unrelated parameters, and
   duplicate security parameters with explicit expected relations.
6. Persist plan hash, commit, seed, shrunk action sequence, observations, trace
   schema, and assertion versions in an evidence envelope.
7. Run the same accepted plans against memory and SQLite stores and compare only
   store-independent observations.
8. Keep production shadow monitoring native; do not load verification scripts in
   the request-serving process.

## Decision record PV-1

- **Decision:** JavaScript compiles data-only verification plans; native Go owns
  actions, observations, and verdicts.
- **Status:** implemented for the generic compiler and runner; strict-provider
  driver remains open.
- **Rationale:** preserves an auditable capability boundary and keeps the
  reference monitor independent of tested scripts.
- **Consequence:** adding new behaviors requires native driver and assertion
  code, but scripts cannot silently redefine security success.

## References

- `design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`
- `design-doc/03-research-foundations-for-identity-protocol-invariants-atomicity-and-runtime-verification.md`
- `sources/paper-enforceable-security-policies.md`
- `sources/paper-monitoring-oriented-programming.md`
- `sources/paper-runtime-verification-brief-account.md`
- `sources/paper-typestate.md`
- `sources/paper-model-based-security-testing.md`
- `sources/paper-metamorphic-testing-cybersecurity.md`
- `../10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md`
