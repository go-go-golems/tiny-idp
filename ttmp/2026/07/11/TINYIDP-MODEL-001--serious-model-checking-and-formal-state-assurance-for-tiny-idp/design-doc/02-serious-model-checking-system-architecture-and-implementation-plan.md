---
Title: Serious Model Checking System Architecture and Implementation Plan
Ticket: TINYIDP-MODEL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: Existing Porcupine one-time capability model and concurrent histories
    - Path: repo://internal/fositeadapter/provider.go
      Note: Authorization begin, resume, terminal, and token endpoint implementation mapped by the models
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Transaction, code redemption, and refresh persistence boundaries for Models B and C
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: Existing Rapid reference model that seeds the authorization abstraction
    - Path: repo://internal/securitytrace/trace.go
      Note: Runtime temporal monitor and observable event vocabulary
    - Path: repo://pkg/idpstore/types.go
      Note: Authoritative interaction state and required-action representation for Model A
    - Path: repo://pkg/verifyplan/plan.go
      Note: Typed native scenario and driver API for counterexample replay
ExternalSources: []
Summary: System design for turning tiny-idp security claims into checked finite-state models, reproducible evidence, and native Go regressions.
LastUpdated: 2026-07-12T00:30:00Z
WhatFor: Defines the model-checking architecture, work products, APIs, phase gates, task provenance, and implementation order.
WhenToUse: Use when planning, implementing, reviewing, or interpreting any TINYIDP-MODEL-001 task.
---


# Serious Model Checking System Architecture and Implementation Plan

## Executive summary

This document designs a model-checking subsystem for tiny-idp. The subsystem is
not one TLA+ file and it is not a replacement for Go tests. It is a controlled
pipeline that turns a security claim into a finite transition system, explores
that system, preserves the checker result, and replays discovered failures
against the implementation.

The first increment will build three deliberately separate models:

1. authorization interaction and persisted browser obligations;
2. authorization-code redemption across transaction and crash boundaries; and
3. refresh-token family rotation, reuse, concurrency, loss, and retry.

TLA+/PlusCal with TLC is the proposed primary notation and checker. That choice
is provisional until Phase 1 reproduces tutorial results and records a reviewed
comparison. Apalache is a later symbolic checker for supported stable models.
Alloy is optional and must answer a concrete relational question before it is
added. Porcupine, Rapid, native runtime monitors, failpoint tests, and typed
verification plans remain part of the system because they connect formal state
to actual Go behavior.

The most important acceptance criterion is a vertical result: deliberately
remove one historical forced-reauthentication protection, obtain a formal
counterexample, normalize it, replay it through the strict-provider driver, and
commit the replay as a Go regression. Until that loop works, the project has a
formal model but not an implementation assurance system.

`tasks.md` is derived from this architecture. Its phases establish evidence
semantics, qualify the theory and tools, review abstraction boundaries, build
the three model slices, connect counterexamples to Go, and govern the result in
CI. The literature work is Phase 1 of ten; it supports the design but does not
define the backlog.

## Problem statement and scope

tiny-idp controls security-relevant state that evolves over time. Examples
include a browser interaction that retains a fresh-login requirement, an
authorization code that may be redeemed once, and a refresh family that must
never reactivate an old generation. Individual branches can look correct while
an interleaving violates the intended property. Ordinary example tests cover
chosen paths; fuzzers sample paths; runtime monitors examine paths that ran.
A model checker instead enumerates or symbolically searches all behaviors in a
declared finite abstraction.

The present repository already has strong ingredients but no general exhaustive
specification. The current evidence includes:

- a Rapid state machine for interaction-store histories in
  `internal/fositeadapter/state_model_test.go:14-155`;
- Porcupine sequential specifications and observed concurrent histories in
  `internal/fositeadapter/linearizability_test.go:17-101`;
- a runtime trace monitor in `internal/securitytrace/trace.go:74-155`;
- typed scenario plans and drivers in `pkg/verifyplan/plan.go:17-158` and
  `internal/fositeadapter/verification_scenario_test.go:37-105`;
- authorization and token transaction failpoints exercised by
  `internal/fositeadapter/sqlstore_test.go:63-291`; and
- deterministic security time in the strict scenario driver at
  `internal/fositeadapter/verification_scenario_test.go:20-35`.

Those tools are valuable, but they answer different questions. Rapid generates
and shrinks sampled executions. Porcupine decides whether a supplied concurrent
history is linearizable. The monitor judges emitted events. VerificationPlan
executes a supplied scenario. None explores every reachable state of a declared
authorization, redemption, or refresh transition system.

### In scope

- finite-state safety checking for the three initial slices;
- carefully selected liveness questions with explicit fairness assumptions;
- explicit attacker, scheduler, crash, retry, and administrative-mutation
  actions where they affect the selected claims;
- evidence envelopes that preserve configuration, tool version, result, and
  counterexample;
- normalization and replay of counterexamples against native Go drivers;
- mutation experiments proving that models detect known classes of defect;
- bounded per-change and larger scheduled/release checker configurations; and
- governance that prevents inconclusive results from being reported as passes.

### Out of scope for the first increment

- proof that Go machine code refines a formal specification;
- re-proving cryptographic primitives, TLS, SQLite, or all of Fosite;
- reproducing the complete web attacker model of academic OAuth analyses;
- generating production Go code from the model;
- adopting several formal languages merely to claim tool diversity; and
- replacing conformance, race, fuzz, recovery, operational, or human gates.

## Current-state architecture

### Authorization interaction state

`pkg/idpstore/types.go:177-238` defines required-action flags and
`InteractionRecord`. The record is the server-owned continuation capability. It
retains the canonical request, browser/session binding, client and user
generations, timestamps, required actions, and terminal outcome. The storage
interface at `pkg/idpstore/interfaces.go:86-88` exposes create, get, and atomic
consume operations.

`internal/fositeadapter/interaction.go:66-109` creates that record after Fosite
validates the authorization request. `provider.go:395-460` begins authorization;
`provider.go:462-616` resumes it; and `provider.go:938-1000` completes it. These
are the primary code mappings for the first formal model.

The historical forced-reauthentication defect demonstrates why persistence
matters. A requirement derived from `prompt=login` or `max_age` must survive the
GET-to-POST boundary. A valid old cookie is state, but it is not proof that the
new interaction's authentication obligation was discharged.

### Transactional token state

`internal/fositeadapter/sqlstore.go:111-190` wraps Fosite token mutations in an
explicit transaction lifecycle and exposes named failpoint boundaries.
Authorization-code state begins at `sqlstore.go:336`; access and refresh rows
begin at `sqlstore.go:400`; refresh rotation is implemented at
`sqlstore.go:455-494`. These APIs supply concrete action names and durable
observations for the redemption and refresh models.

The crucial distinction is among transaction-local state, committed durable
state, and HTTP response state. A crash after commit but before response
delivery is not equivalent to rollback. The model must expose that ambiguity
rather than flattening the whole token endpoint into a single successful call.

### Existing executable models and monitors

The Rapid interaction model defines a small reference state and actions at
`state_model_test.go:14-73`, then compares model observations with store calls.
This provides vocabulary and generators, but not exhaustive reachability.

The Porcupine model at `linearizability_test.go:17-35` describes a one-time
boolean capability. Concurrent consume histories are supplied at lines 37-101.
This checks whether each observed history admits a valid sequential ordering. It
does not model browser obligations, refresh generations, crash, or response
loss.

The monitor at `internal/securitytrace/trace.go:74-155` keeps per-interaction
state and reports temporal violations. It can detect a bad executed trace. A
model checker can search traces the program has not emitted. The two should
share stable property and action identifiers but must retain different evidence
claims.

## Gap analysis

| Needed capability | Present evidence | Missing work |
|---|---|---|
| Exhaustive finite reachability | No general specification | `Init`, `Next`, finite constants, invariants, TLC configuration |
| Explicit assumptions | Scattered in code/tests/docs | reviewed Fosite, SQLite, clock, actor, crash, and mutation ledgers |
| Reproducible result semantics | normal Go test output | versioned evidence envelope and strict outcome vocabulary |
| Known-defect sensitivity | historical Go regressions | formal mutation models that rediscover the defects |
| Counterexample portability | manual interpretation | normalized trace schema, exporter, adapter, corpus, renderer |
| Abstract/implementation comparison | partial scenario assertions | differential model, HTTP, row-count, and event observations |
| Governed CI | ordinary tests only | pinned checkers, budgets, artifacts, ownership, release binding |
| Broader family policy | boolean Porcupine oracle | refresh generation/family transition model |

The largest risk is not lack of notation. It is an abstraction that silently
omits the behavior responsible for a product defect. The second risk is an
overstated result: a bounded successful run described as proof of the unbounded
implementation. The design therefore makes abstraction ledgers and evidence
semantics prerequisites, not post-processing documentation.

## Proposed system architecture

### Components and responsibilities

```text
security claims and historical defects
                 |
                 v
+-------------------------------+
| property catalog              |
| stable IDs, threat, verdict   |
+-------------------------------+
          |             |
          v             v
+----------------+  +----------------+
| abstraction    |  | assumptions    |
| state/actions  |  | Fosite/SQLite  |
+----------------+  +----------------+
          \             /
           v           v
      +---------------------+
      | TLA+/PlusCal models |
      | configs + mutations |
      +---------------------+
          |             |
          v             v
+----------------+  +----------------+
| run envelope   |  | raw/normalized |
| stats + result |  | counterexample |
+----------------+  +----------------+
                           |
                           v
                 +--------------------+
                 | VerificationPlan   |
                 | native Go adapter  |
                 +--------------------+
                           |
                           v
                 +--------------------+
                 | differential       |
                 | observations       |
                 +--------------------+
                           |
                           v
                  CI and release record
```

The property catalog identifies the claim independently of checker syntax. The
abstraction ledger identifies which product state implements the claim. The
assumption ledger identifies behavior accepted from dependencies. The formal
model searches the finite abstraction. The evidence envelope makes the result
reproducible. The trace adapter connects a failing behavior to the Go provider.

### Repository layout

The proposed implementation layout uses the repository's top-level Go module
and does not create nested modules:

```text
models/
  README.md
  catalog.yaml
  authorization/
    Authorization.tla
    Authorization.cfg
    AuthorizationMutationRequiredLogin.tla
  redemption/
    Redemption.tla
    Redemption.cfg
  refresh/
    RefreshFamily.tla
    RefreshFamily.cfg
  schemas/
    evidence-envelope.schema.json
    normalized-trace.schema.json
  corpus/
    authorization/
    redemption/
    refresh/

internal/modelcheck/
  evidence.go
  trace.go
  tlcparse.go
  render.go

internal/fositeadapter/
  modelcheck_verification_test.go
```

Models are product assets and belong outside the ticket once implemented. The
ticket's `scripts/` directory holds temporary probes, qualification commands,
and research tooling until an artifact is mature enough to move into product
code.

## Data and API design

### Stable identifiers

Identifiers must survive file renames and checker changes:

```go
type ModelID string
type ActionID string
type PropertyID string
type ObservationID string

const (
    ModelAuthorization ModelID = "AUTH-INTERACTION"
    ActionAuthBegin ActionID = "AUTH.BEGIN"
    PropertyRequiredLoginPersists PropertyID = "AUTH.S-004"
)
```

The catalog maps every ID to a description, owning model, product symbols,
security events, and maturity. TLA+ operators should include the same ID in a
comment or operator name. Runtime monitors and regressions should report it.

### Abstraction ledger

Every model receives a table with these columns:

| Product field or behavior | Formal representation | Treatment | Reason | Code evidence |
|---|---|---|---|---|
| required actions | finite set/bitset | modeled | central security obligation | `pkg/idpstore/types.go:177-238` |
| raw password | none | omitted | success/failure is sufficient | authenticator boundary |
| canonical request | abstract digest and bindings | abstracted | contents matter through equality and selected fields | `interaction.go:128-170` |
| wall clock | bounded logical integer | abstracted | only order and expiry boundary matter | provider clock |
| RSA signature bytes | none | assumed | cryptography is outside model | signing capability becomes boolean/generation |

Treatment must be one of `modeled`, `derived`, `abstracted`, or `omitted`. An
omission is not automatically safe; it requires an argument explaining why the
selected properties cannot depend on it.

### Assumption ledger

Assumptions use stable IDs and carry a validation link:

```yaml
- id: SQL-A-001
  statement: A committed transaction survives modeled process restart.
  relied_on_by: [REDEEM-S-002, REFRESH-S-003]
  implementation_evidence:
    - internal/fositeadapter/sqlstore_test.go
  external_basis: SQLite transaction documentation
  failure_if_false: durable state may contradict the model
  status: proposed
```

Important groups are Fosite authorization/token behavior, SQLite atomicity and
isolation, logical time, scheduler fairness, browser isolation, and
administrative mutation visibility.

### Evidence envelope API

```go
type Outcome string

const (
    OutcomePass         Outcome = "PASS"
    OutcomeFail         Outcome = "FAIL"
    OutcomeInconclusive Outcome = "INCONCLUSIVE"
    OutcomeToolError    Outcome = "TOOL_ERROR"
)

type EvidenceEnvelope struct {
    SchemaVersion   int
    ModelID         ModelID
    SourceSHA256    string
    PropertyIDs     []PropertyID
    Checker         CheckerIdentity
    Configuration   json.RawMessage
    Outcome         Outcome
    Statistics      RunStatistics
    Counterexample  *CounterexampleReference
    SourceCommit    string
}
```

`PASS` is permitted only after the declared finite state space completes.
Invariant violation is `FAIL`. Timeout, state-space exhaustion, and an explicit
search budget ending are `INCONCLUSIVE`. Installation, parsing, and checker
crashes are `TOOL_ERROR`. CI fails on `FAIL` and `TOOL_ERROR`; release policy
also refuses to treat `INCONCLUSIVE` as evidence of the property.

### Normalized trace API

```go
type Trace struct {
    SchemaVersion int
    ModelID       ModelID
    PropertyID    PropertyID
    Constants     map[string]json.RawMessage
    Steps         []TraceStep
}

type TraceStep struct {
    Index       int
    ActionID    ActionID
    Actor       string
    Arguments   json.RawMessage
    Before      json.RawMessage
    After       json.RawMessage
    Observation json.RawMessage
}
```

The normalized form contains abstract identifiers, hashes, enumerations, and
logical times, never passwords, tokens, cookies, or raw authorization codes.
The original raw checker trace is preserved beside it for forensic review.

### VerificationPlan adapter

`verifyplan.Driver` is defined at `pkg/verifyplan/plan.go:111-113`. The adapter
will translate normalized action IDs into existing strict-provider driver steps:

```text
AUTH.BEGIN              -> begin_authorization
AUTH.SUBMIT_LOGIN       -> submit_interaction(login,password)
AUTH.SUBMIT_CONSENT     -> submit_interaction(consent)
TIME.ADVANCE            -> advance_clock
AUTH.MUTATE_CLIENT_GEN  -> administrative test fixture operation
CRASH.RESTART           -> restartable store/provider harness operation
```

Translation fails closed on an unknown action or missing argument. It does not
silently skip abstract steps. The result contains HTTP status/redirect,
interaction outcome, durable row counts, and security-event verdicts.

## The three initial formal models

### Model A: authorization interaction

State includes two bounded interactions, browser/tab binding, optional session,
required actions, authentication/consent satisfaction, client/user/key
generations, logical time, and terminal outcome.

Representative actions are:

```text
Begin(tab, request)
Resume(tab, handle)
Authenticate(tab, credentialsResult)
ApproveConsent(tab)
Deny(tab)
Expire(interaction)
MutateClientGeneration(client)
DisableUser(user)
RotateOrRemoveSigningKey()
CommitApproval(interaction)
```

Initial safety properties include validation before credentials, server-owned
canonical request, persisted forced login and fresh-login requirements, consent
before approval, expiry exclusion, tab isolation, one terminal outcome, and no
artifact before approved commit.

The model must include mutation variants. Removing persisted required actions
must produce the historical blank/crafted POST bypass. Splitting terminal check
from consume must produce a concurrent approve/deny or replay trace. A model
that cannot find those seeded defects has not yet demonstrated useful fidelity.

### Model B: authorization-code redemption

State includes code status, transaction-local mutations, durable token rows,
transaction phase, response state, crash/recovery, and client retry.

```text
PresentCode(client)
BeginTransaction(request)
ReadActiveCode()
CreateReplacementRows()
InvalidateCode()
Commit()
Rollback()
Crash()
Recover()
DeliverResponse()
RetryRequest(client)
```

Properties require at-most-one committed redemption, all-or-none durable
replacement state, rollback preserving the active code, and committed
replacement tokens implying invalid code. The model explicitly does not promise
that a client receives a response after commit. Phase 4 must make the recovery
policy for that ambiguity a product decision.

### Model C: refresh-token family

State includes family status, current generation, presented generation,
replacement rows, reuse marker, response state, crash, loss, and retry.

Properties include at-most-one successful rotation per generation, no active
token in a revoked family, no reactivation of an old generation, and consistent
reuse policy. The model must expose the legitimate concurrent-refresh sequence
in which one request wins and the loser presents the now-old token, potentially
revoking the winner's family. That is both a security policy and an availability
result; it leads to a documented client singleflight requirement and operational
indicator.

## Core end-to-end flows

### Successful finite check

```text
engineer selects model + configuration + property IDs
    -> wrapper hashes source and records checker identity
    -> checker explores declared state space
    -> parser reads completion statistics
    -> wrapper verifies completion, not only process exit code
    -> evidence envelope records PASS and finite constants
    -> CI archives model, config, stdout/stderr, and envelope
```

### Counterexample-to-regression flow

```text
checker reports invariant AUTH-S-004 failure
    -> preserve raw trace
    -> parse state transitions and infer stable action IDs
    -> write normalized trace JSON
    -> validate JSON schema and secret policy
    -> translate into VerificationPlan steps
    -> execute native strict-provider driver
    -> compare abstract and concrete observations
    -> classify product defect, model over-approximation, or adapter gap
    -> commit minimal trace and Go regression
```

### Model change review flow

```text
review specification diff
    -> review property and assumption changes separately
    -> inspect abstraction-ledger changes
    -> rerun mutation models
    -> compare state counts and depths
    -> replay retained counterexample corpus
    -> require owner approval for weakened/removed property
```

## Decision records

### Decision: TLA+/PlusCal with TLC is the first toolchain

- **Context:** The first problems are concurrent transition systems with safety
  properties, explicit nondeterminism, crashes, and small finite domains.
- **Options considered:** TLA+/TLC, Quint, Alloy 6, SPIN, Murphi, direct Go
  enumeration, and immediate symbolic checking.
- **Decision:** Qualify TLA+/PlusCal and TLC first; approve it only after Phase 1.
- **Rationale:** It directly expresses actions and temporal properties, provides
  explicit counterexamples, and has mature finite-state exploration.
- **Consequences:** The team must learn TLA+ semantics and control state growth.
  This is a proposed decision, not a completed tool qualification.
- **Status:** proposed

### Decision: three small models rather than one OAuth model

- **Context:** Interaction obligations, SQL redemption, and refresh-family
  policy have different authoritative state and atomicity boundaries.
- **Options considered:** one end-to-end model; three slices; only interaction.
- **Decision:** Build three slices in dependency order.
- **Rationale:** Smaller models make counterexamples and assumptions reviewable
  and allow each model to align with existing native harnesses.
- **Consequences:** Cross-slice claims are deferred and interfaces between models
  must be documented.
- **Status:** proposed

### Decision: replay, not claimed formal refinement

- **Context:** A checked abstraction does not prove the Go implementation follows
  it, while full refinement would substantially expand scope.
- **Options considered:** informal comparison; counterexample replay; generated
  code; machine-checked refinement.
- **Decision:** Normalize counterexamples and replay them through native drivers.
- **Rationale:** It creates concrete implementation regressions with manageable
  complexity and exposes abstraction gaps.
- **Consequences:** Successful abstract checks remain model-level evidence, and
  unobserved implementation deviations remain possible.
- **Status:** proposed

### Decision: one evidence contract across checkers

- **Context:** TLC, Apalache, and future tools report bounds and failures
  differently.
- **Options considered:** raw logs only; tool-specific metadata; common envelope.
- **Decision:** Preserve raw output and also emit a common versioned envelope.
- **Rationale:** CI and reviewers need consistent outcome semantics without
  erasing checker-specific evidence.
- **Consequences:** Parsers become security-relevant assurance tooling and need
  fixtures, failure tests, and version pinning.
- **Status:** proposed

### Decision: secondary tools require a question

- **Context:** Multiple successful tools do not automatically constitute
  independent proof and increase maintenance cost.
- **Options considered:** mandate TLA+, Apalache, and Alloy for every model; use
  only TLC forever; conditional adoption.
- **Decision:** Evaluate Apalache on stable TLA+ models and Alloy only for a
  concrete relational question.
- **Rationale:** Each tool must add coverage or usability, not badges.
- **Consequences:** A written no-adopt decision can complete Phase 6.
- **Status:** proposed

## Task derivation and implementation phases

The task ledger is generated from five evidence gaps:

1. the absence of explicit checker-result semantics;
2. the need for reviewer competence in the selected formal semantics;
3. the absence of reviewed product abstractions and dependency assumptions;
4. the three product state/atomicity boundaries; and
5. the absence of a counterexample-to-Go and checker-to-release pipeline.

| Phase | Why it exists | Exit artifact |
|---|---|---|
| 0. Baseline/evidence | prevents undefined or overstated results | IDs, schema, outcome rules, charter approval |
| 1. Literature/tooling | prevents semantic/tool misuse | annotated bibliography, tutorial evidence, comparison and decision |
| 2. Abstraction | prevents hidden omissions and dependency claims | field/action and assumption ledgers |
| 3. Authorization | proves the first vertical loop and historical sensitivity | checked model, mutation traces, Go regression |
| 4. Redemption | models durable atomicity, crash, loss, retry | checked model, recovery decision, SQL regressions |
| 5. Refresh | adds generations and family policy beyond boolean linearizability | checked model, Porcupine comparison, client/ops requirements |
| 6. Secondary analysis | controls Alloy/Apalache adoption and reductions | justified result or no-adopt decision |
| 7. Integration | makes failures portable and durable | schema, parser, adapter, corpus, renderer |
| 8. CI/governance | makes evidence reproducible and reviewable | pinned jobs, budgets, ownership, release binding |
| 9. Refinement | prevents premature scope expansion | measured follow-on plan |

Every checkbox in `tasks.md` is an acceptance criterion for one of these exit
artifacts. For example, Phase 3 is intentionally decomposed into writing the
model, encoding properties, isolating tabs, seeding two defect families,
checking the corrected design, handling fairness, exporting a trace, replaying
Go, and reviewing divergences. “Create a TLA+ file” cannot complete the phase.

### File-level implementation order

1. Add `models/README.md`, `models/catalog.yaml`, and JSON schemas. Implement the
   evidence types and validators under `internal/modelcheck/` with table tests.
2. Complete the Phase 1 tutorial models under the ticket `scripts/` directory;
   preserve exact commands and checker metadata. Do not promote tutorials into
   product `models/`.
3. Write and review the authorization abstraction ledger. Map actions to
   `interaction.go`, `provider.go`, `idpstore`, events, and existing tests.
4. Add the authorization model/config and mutation variant. Check a deliberately
   tiny configuration first, then two interactions/tabs/generations.
5. Manually normalize the first counterexample before automating parsing. Add a
   native replay in `modelcheck_verification_test.go`.
6. Implement the TLC parser only after at least two real output fixtures exist.
   Fail closed on unknown completion or trace formats.
7. Repeat the abstraction/model/replay loop for redemption, aligning every model
   failure point with `sqlstore.go` hooks and `sqlstore_test.go` assertions.
8. Repeat for refresh, then compare its histories with Porcupine and expand the
   oracle only where the model demonstrates missing family semantics.
9. Add fast CI only after local commands are reproducible. Add scheduled/release
   bounds separately and archive evidence envelopes plus raw logs.
10. Evaluate secondary tools and later models after the complete vertical loop
    has passed an independent review.

## Testing and validation strategy

### Model tests

- syntax and type checks for every specification;
- `TypeOK` invariant in every finite configuration;
- seeded mutation models that must fail the intended property;
- corrected models that must complete within declared budgets;
- retained counterexamples replayed after specification changes;
- state-count/depth comparison to expose accidental state-space collapse; and
- separate configurations for safety and any fairness-dependent liveness claim.

### Assurance-tool tests

- JSON schema acceptance and rejection fixtures;
- parser fixtures for pass, invariant violation, timeout, state exhaustion,
  malformed output, checker crash, and unknown checker version;
- secret scanners over raw and normalized artifacts;
- deterministic rendering of timelines;
- adapter rejection of unknown or incomplete actions; and
- round-trip preservation of IDs, constants, property, and action order.

### Implementation conformance tests

- compare abstract terminal state with HTTP status/redirect;
- compare artifact counts with durable SQL rows;
- compare action order with security events and monitor verdicts;
- run counterexamples repeatedly under `go test -race` where applicable;
- retain historical and generated scenarios in a versioned corpus; and
- add failpoints only when the model identifies an unobservable boundary.

### CI profiles

| Profile | Trigger | Purpose | Outcome policy |
|---|---|---|---|
| smoke | model/tool changes | syntax, tiny bounds, parser and mutation checks | fail on any non-expected result |
| per-change | every relevant PR | complete small finite configurations | fail on violation/tool error; disclose inconclusive |
| scheduled | nightly/weekly | larger actors, time, generations, failures | archive statistics and traces |
| release | candidate commit | approved configurations and full corpus replay | required evidence; no inconclusive substitution |

## Risks, alternatives, and open questions

### Principal risks

- **Unsound abstraction:** a removed field changes the property. Mitigation:
  field-by-field ledger, mutation cases, and native differential replay.
- **State explosion:** realistic cardinalities become intractable. Mitigation:
  start small, measure counts, justify symmetry/constraints, and preserve a
  baseline before reductions.
- **False confidence:** finite pass is reported as product proof. Mitigation:
  evidence envelope and explicit claim language.
- **Model drift:** product behavior changes without specification updates.
  Mitigation: code relations, ownership, corpus replay, and release source hashes.
- **Parser fragility:** checker output changes and is misclassified. Mitigation:
  pinned versions, fixtures, raw output, and fail-closed parsing.
- **Instrumentation gap:** implementation violates a property without an
  observable difference. Mitigation: row/event observations and targeted
  failpoints rather than speculative instrumentation.

### Alternatives retained for evaluation

Quint may improve authoring and simulation ergonomics. Apalache may improve
bounded symbolic search. Alloy may better express binding and lineage relations.
SPIN or Murphi may be useful if process/channel or protocol-state questions
dominate later. Direct Go state enumeration remains useful for implementation
adjacency. None is rejected permanently; none is added without a precise
question and maintenance owner.

### Open decisions

1. Which exact TLC distribution and checksum policy will CI pin?
2. Should initial specifications use direct TLA+ actions or PlusCal for the
   transition algorithm and generated TLA+ for checking?
3. Which abstraction of Fosite code invalidation is supported by upstream
   contract versus only observed implementation behavior?
4. What client-facing recovery behavior is acceptable after commit with lost
   token response?
5. Is automatic TLC trace parsing stable enough after the first two manual
   traces, or should an intermediate structured exporter be used?
6. Which model files require security-owner approval in CODEOWNERS or equivalent
   review policy?

## Intern onboarding and first contribution

The intern should not begin by editing TLA+. They should proceed in this order:

1. Read `pkg/idpstore/types.go:177-238`, `interaction.go:66-170`, and
   `provider.go:384-616` to understand server-owned continuation state.
2. Run the Rapid, Porcupine, security-trace, strict-scenario, and SQL failpoint
   tests. Record what each can and cannot claim.
3. Read the companion research/theory guide and reproduce the Phase 1 TLC
   one-time-capability tutorial with exact tool metadata.
4. Draft one row-complete authorization abstraction ledger and obtain review.
5. Add the smallest authorization model with one interaction and no concurrency.
6. Add a second tab and persisted login obligation; then run the required-action
   mutation experiment.
7. Manually convert the first counterexample to a VerificationPlan scenario.
8. Stop and review the full loop before adding code redemption or automation.

The first contribution is successful when another engineer can reproduce the
checker result, understand every modeled and omitted field, replay the failure
against Go, and state the evidence boundary without reading terminal history.

## References

### Local implementation

- `pkg/idpstore/types.go:177-238` — interaction actions and record.
- `pkg/idpstore/interfaces.go:86-88` — interaction-store contract.
- `internal/fositeadapter/interaction.go:66-170` — create and reconstruct flow.
- `internal/fositeadapter/provider.go:384-616` — begin/resume authorization.
- `internal/fositeadapter/provider.go:938-1000` — terminal authorization flow.
- `internal/fositeadapter/state_model_test.go:14-155` — Rapid reference model.
- `internal/fositeadapter/linearizability_test.go:17-101` — Porcupine model.
- `internal/securitytrace/trace.go:74-155` — temporal runtime monitor.
- `pkg/verifyplan/plan.go:17-158` — typed plan and driver API.
- `internal/fositeadapter/verification_scenario_test.go:20-105` — native driver.
- `internal/fositeadapter/sqlstore.go:111-190` — token transaction lifecycle.
- `internal/fositeadapter/sqlstore.go:336-494` — code/token/refresh persistence.
- `internal/fositeadapter/sqlstore_test.go:63-291` — failpoint atomicity tests.

### Ticket material

- `design-doc/01-model-checking-research-analysis-design-and-implementation-guide.md`
  — companion theory, literature, property catalog, case studies, and proposed
  model details.
- `tasks.md` — authoritative fine-grained phase ledger.
- `reference/01-investigation-diary.md` — chronological evidence and decisions.
- `sources/` — preserved primary tool documentation and research papers.

### Primary external material

The ticket source packet preserves official TLA+/PlusCal/TLC, Apalache, Alloy,
Quint, and Porcupine documentation, together with papers on formal OAuth/OIDC
analysis, model-based testing, linearizability, runtime verification,
concurrency exploration, fault injection, stateful fuzzing, and typestate. The
companion guide separates source-derived claims from tiny-idp design inferences.
