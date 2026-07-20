# Tasks

> **Planning status:** The implementation tasks in Phases 0–8 below belong to
> the deprecated graph-first design in `design-doc/01`. They are retained as
> historical planning evidence and must not be treated as the normative API
> sequence. The current implementation sequence is the lambda-first plan in
> `design-doc/03` and the `Lambda-first superseding design` tasks appended to
> this file. Phase 9's assurance-vocabulary work remains complementary.

## Research and design deliverable

- [x] Create `TINYIDP-GOJA-001` with architecture/auth/Go/OIDC/research/testing/xgoja topics <!-- t:d3bd -->
- [x] Move `/tmp/idp-research.md` into the ticket `sources/` directory <!-- t:3njt -->
- [x] Read the colleague identity-microkernel research completely <!-- t:fvvu -->
- [x] Map current tiny-idp engines, strict request flow, persistence, lifecycle, and extension seams <!-- t:v6tr -->
- [x] Map current go-go-goja runtime ownership, module selection, interruption, and xgoja provider APIs <!-- t:o0qm -->
- [x] Run the current tiny-idp test baseline <!-- t:rcdh -->
- [x] Write the intern-oriented analysis/design/implementation guide <!-- t:o5sn -->
- [x] Record compact architecture decisions, alternatives, risks, and open questions <!-- t:4hin -->
- [x] Create a phased implementation and acceptance plan <!-- t:y3cp -->
- [x] Validate ticket frontmatter and docmgr health <!-- t:zvi5 -->
- [x] Upload the design and diary bundle to reMarkable <!-- t:b21a -->

## Phase 0: contract spike and dependency decision

- [ ] Decide whether tiny-idp may raise its minimum Go version from 1.25.11 to 1.26.1+ <!-- t:c22l -->
- [ ] Pin a released or exact go-go-goja version in `tiny-idp/go.mod` <!-- t:0868 -->
- [ ] Build an explicit compile-only `require("tinyidp")` native module spike <!-- t:c1j3 -->
- [ ] Disable implicit and data-only default modules in compiler and policy factories <!-- t:3wqm -->
- [ ] Add negative tests for `fs`, `exec`, `database`, `os`, `process`, network, and arbitrary loaders <!-- t:yk2r -->
- [ ] Compile one Goja program and load it independently into multiple owned runtimes <!-- t:yloc -->
- [ ] Implement and race-test execution deadline interruption and `ClearInterrupt` ordering <!-- t:u075 -->
- [ ] Publish the supported JavaScript syntax profile <!-- t:1nuj -->
- [ ] Pass direct, `GOWORK=off`, and race-test gates <!-- t:ahgp -->

## Phase 1: pure-Go graph and validation

- [ ] Add `pkg/idpgraph` schema, stable enums, nodes, slots, callbacks, tests, and diagnostics <!-- t:o8no -->
- [ ] Add canonical JSON and deterministic graph/source hashing <!-- t:559d -->
- [ ] Add native block descriptor registry with input/output/effect/capability metadata <!-- t:l2ut -->
- [ ] Model the current strict OIDC/password/consent/claims/issuance flow <!-- t:h745 -->
- [ ] Add a pure-Go `localWeb` preset <!-- t:kzwd -->
- [ ] Implement reference, cycle, reachability, termination, slot, capability, effect, and production-profile validation <!-- t:of6d -->
- [ ] Materialize the pure-Go graph into the existing strict provider <!-- t:szms -->
- [ ] Add deterministic snapshots and malformed-graph tests <!-- t:nqv1 -->

## Phase 2: fluent compiler module

- [ ] Add `pkg/idpscript` compiler/artifact contracts <!-- t:scjs -->
- [ ] Add `pkg/gojamodules/tinyidp` with `v1` API and lowerCamelCase data <!-- t:9guv -->
- [ ] Implement `DraftCollector` and branded fluent builder objects <!-- t:uks3 -->
- [ ] Implement `preset`, `ref`, `protocol`, `authn`, `policy`, `claims`, `consent`, `issue`, `decision`, and `slot` v1 subsets <!-- t:e19z -->
- [ ] Require explicit callback names and deterministic registration IDs <!-- t:kq80 -->
- [ ] Bound source/config/object sizes and reject non-serializable values <!-- t:k2ye -->
- [ ] Export graph through `module.exports`/`build()` without starting services <!-- t:0tli -->
- [ ] Add TypeScript declarations <!-- t:6ucc -->
- [ ] Add `tinyidp script validate` and canonical graph output <!-- t:81cm -->
- [ ] Add checked-in local-web script example <!-- t:2drr -->

## Phase 3: strict authorization and claims seams

- [ ] Add immutable public subject/client/request/authentication context DTOs <!-- t:q89w -->
- [ ] Add allow/deny `AuthorizationPolicy` and `PolicySet` <!-- t:jcgl -->
- [ ] Add `ClaimsPolicy` with protected protocol claim names <!-- t:4nnp -->
- [ ] Thread policies through `embeddedidp.Options` and Fosite adapter options <!-- t:8xhf -->
- [ ] Invoke authorization policy after native validation and before consent/code issuance <!-- t:tdqa -->
- [ ] Invoke claims policy before OIDC session persistence <!-- t:ujyy -->
- [ ] Propagate AMR/ACR through browser and OIDC sessions <!-- t:m8da -->
- [ ] Add static native RBAC and claims blocks <!-- t:7oob -->
- [ ] Test fresh login, existing session, prompt none, consent, deny/error, refresh, and UserInfo <!-- t:vakp -->
- [ ] Re-run strict conformance gates <!-- t:fozg -->

## Phase 4: policy pool and capabilities

- [ ] Add same-source callback registry and fingerprint verification per worker <!-- t:nie1 -->
- [ ] Add bounded single-owner worker acquisition and lifecycle <!-- t:ydej -->
- [ ] Add timeout/panic/invalid-output worker discard and replacement <!-- t:f04k -->
- [ ] Add bounded immutable JS input/output codecs <!-- t:x7cg -->
- [ ] Add typed capability descriptors, registry, permissions, effects, and budgets <!-- t:h46y -->
- [ ] Add JavaScript authorization and claims callback nodes <!-- t:evgu -->
- [ ] Add pool/generation readiness and metrics <!-- t:57os -->
- [ ] Add saturation, exception, timeout, capability failure, and concurrent strict-flow tests <!-- t:g1nx -->
- [ ] Run race and production-shaped mixed-load gates <!-- t:thxp -->

## Phase 5: tests, explain, and atomic activation

- [ ] Collect and run `A.test` cases with deterministic fake capabilities <!-- t:na1p -->
- [ ] Add `tinyidp script test` and `tinyidp script explain` <!-- t:2620 -->
- [ ] Add generation manager with warmup, atomic swap, drain, close, and rollback <!-- t:6z3f -->
- [ ] Add opt-in file-watch reload <!-- t:cf1x -->
- [ ] Audit activation attempts using source and graph hashes <!-- t:sn2g -->
- [ ] Keep previous generation active after compile/test/warmup failures <!-- t:4f1h -->
- [ ] Test repeated reloads for request consistency and resource/goroutine leaks <!-- t:9mgk -->

## Phase 6: xgoja/v2 packaging

- [ ] Add `pkg/xgoja/providers/tinyidp` <!-- t:7n5g -->
- [ ] Add provider module config schema and TypeScript descriptor <!-- t:cu9d -->
- [ ] Add stable typed host-service key and lookup <!-- t:vo2x -->
- [ ] Add compile-only `xgoja.yaml` example <!-- t:wlf7 -->
- [ ] Add Go-owned provider command set or generated runtime-package serving example <!-- t:cqwx -->
- [ ] Test provider registry and module factory behavior <!-- t:nmib -->
- [ ] Pass `xgoja doctor`, `xgoja gen-dts`, and `xgoja build` <!-- t:tawc -->

## Phase 7: typed challenge graph

- [ ] Refactor password and existing-session behavior behind native block interfaces <!-- t:l5et -->
- [ ] Implement complete five-outcome transition tables <!-- t:zf2d -->
- [ ] Add durable hashed one-time challenge continuation contract and SQLite schema <!-- t:j5pb -->
- [ ] Bind challenges to graph generation, flow, client, subject, and expiry <!-- t:rc1p -->
- [ ] Implement native interaction renderer contract <!-- t:rufv -->
- [ ] Add `seq`, `all`, `firstAvailable`, `when`, and `choose` with table-driven tests <!-- t:0v2f -->
- [ ] Add native evidence, AMR, ACR, and step-up propagation <!-- t:v219 -->
- [ ] Implement one complete native factor (recommended passkey) before plugin factors <!-- t:zzfd -->
- [ ] Threat-model downgrade, fallthrough, replay, retry, and cross-generation resume <!-- t:r3j6 -->

## Phase 8: additional native blocks and protocols

- [ ] Prioritize upstream OIDC, production device flow, and dynamic step-up <!-- t:6msg -->
- [ ] Add token exchange and transaction authorization as separate native slices <!-- t:4h93 -->
- [ ] Keep CIBA, workload identity, quorum, edge, and multi-actor flows experimental until abstractions stabilize <!-- t:sblx -->
- [ ] Require native storage/protocol support, graph descriptors, interoperability tests, docs, and production review for each slice <!-- t:tpda -->
- [ ] Define a separate tinyidp/verify module and pure-Go VerificationPlan schema for offline security scenarios <!-- t:e7z1 -->
- [ ] Specify verification runtime profiles and prove production policy runtimes cannot resolve test-only capabilities <!-- t:4phg -->
- [ ] Prototype scenario compilation and Go-owned execution against redacted traces, fake clock, and failpoint adapters <!-- t:7aql -->
- [ ] Add negative tests preventing verification plugins from overriding production authorization decisions or suppressing invariant failures <!-- t:4dw1 -->

## Phase 9: assurance-oriented core grammar and staged refactoring

- [x] Synthesize the Goja graph, model-checking, static-analysis, runtime-monitoring, and current-code designs into one assurance-oriented refactoring proposal. <!-- t:d10n -->
- [x] Inventory and crosswalk every current required-action bit, state-model action, VerificationPlan step, assertion ID, security event, model action, and static property ID. <!-- t:agxk -->
- [ ] Define stable versioned identifiers for resources, facts, obligations, steps, effects, outcomes, observations, and properties in a dependency-neutral internal package. <!-- t:buxx -->
- [ ] Define and validate the three-schema boundary: configuration graph, native transition catalog, and scenario/trace records. <!-- t:7r8d -->
- [ ] Add lossless fail-closed codecs between InteractionRequiredAction bits and obligation IDs. <!-- t:mdhf -->
- [ ] Add typed VerificationPlan step codecs and reject unknown kinds/parameters during plan materialization. <!-- t:vpdm -->
- [ ] Map current authorization events to transition results and prove complete secret-free terminal-path instrumentation. <!-- t:0zlh -->
- [ ] Introduce unexported authorization proof types and one approved artifact-issuance sink without changing Fosite protocol behavior. <!-- t:uopp -->
- [ ] Generate static-analysis authority/effect metadata and a formal-model vocabulary skeleton from the transition catalog. <!-- t:9zhn -->
- [ ] Replay one normalized model counterexample through registered VerificationPlan codecs without a handwritten action-name adapter. <!-- t:e9p7 -->
- [ ] Materialize the local-web Goja configuration graph solely from registered native block and policy descriptors. <!-- t:t0q3 -->
- [ ] Evaluate selective pure transition kernels only after the vocabulary, tracing, proof boundary, analyzers, and model replay are stable. <!-- t:qh8f -->
- [ ] Run differential provider, persistence, trace, monitor, conformance, race, fuzz, failpoint, and performance gates after each refactoring phase. <!-- t:kjlg -->

## Lambda-first superseding design

This is the normative implementation ledger for
`design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md`.
Execute the phases in order. A later phase may be prototyped for discovery, but
it must not be integrated until every earlier phase gate passes. The historical
graph-first tasks above are not prerequisites for this plan.

### Planning and design checkpoints

- [x] Publish the normative intern implementation guide and clearly deprecate design 01 <!-- t:vnp5 -->
- [x] Validate docmgr bookkeeping and upload the superseding design bundle to reMarkable <!-- t:d8yk -->
- [x] Expand design 03 into this phase-by-phase implementation ledger with dependencies, acceptance gates, validation commands, and commit boundaries <!-- t:lf00 -->

### Phase 0: freeze contracts with a no-browser runtime spike

**Purpose:** Prove the lambda registration and invocation model without changing
browser, persistence, Fosite, or account-creation behavior.

**Depends on:** Design 03 only.

**Deliverables:** Pure-Go contracts in `pkg/idpprogram`, an isolated
`require("tinyidp").v1` compiler in `pkg/idpscript` and
`internal/gojamodules/tinyidp`, and an owner-safe bounded worker spike.

#### Phase 0 tasks

- [x] Record the current Go version, go-go-goja dependency source, and direct/GOWORK-off/race baseline before adding packages; resolve the minimum-Go-version and pinned-dependency decisions without adding a compatibility shim <!-- t:lf01 -->
- [x] Add runtime-independent `Program`, `Workflow`, `WorkflowHandlers`, `LambdaSpec`, schema, capability-requirement, effect-declaration, budget, and diagnostic types under `pkg/idpprogram` <!-- t:lf02 -->
- [x] Add stable identifiers and typed handler outcomes for `present`, `challenge`, `commit`, `complete`, `deny`, and `skip`, including validation that a handler returns only declared outcomes <!-- t:lf03 -->
- [x] Implement deterministic program validation for duplicate identifiers, missing handlers, incompatible continuation edges, undeclared capabilities/effects, invalid budgets, unbounded carry schemas, and unreachable workflow entries <!-- t:lf04 -->
- [x] Implement canonical program serialization plus deterministic source, program, callback-registry, and schema fingerprints <!-- t:lf05 -->
- [x] Implement the isolated `require("tinyidp").v1` module with only `program`, `workflow`, `lambda`, and the Phase 0 result builders needed by the spike <!-- t:lf06 -->
- [x] Compile source into an immutable artifact containing the canonical program, callback metadata, compiled source, required capabilities, and fingerprints without starting a server or mutating global registries <!-- t:lf07 -->
- [x] Load one artifact into at least two separately owned runtimes and reject activation if their callback IDs or fingerprints differ <!-- t:lf08 -->
- [x] Build an explicit runtime factory whose module resolver exposes only the Tiny-IDP module and approved language/runtime primitives; add negative resolution tests for filesystem, process, execution, database, network, OS, and arbitrary loaders <!-- t:lf09 -->
- [x] Implement single-owner worker acquisition, invocation, release, discard, and replacement so no request goroutine touches a Goja runtime directly <!-- t:lf10 -->
- [x] Implement bounded invocation of synchronous and Promise-returning lambdas, with request cancellation, deadline interruption, `ClearInterrupt` ordering, late-settlement containment, panic recovery, output validation, and mandatory worker discard after unsafe termination <!-- t:lf11 -->
- [x] Bind only declared capabilities to an invocation, enforce call/effect/time/output budgets, and prove retained globals cannot reuse an expired binding <!-- t:lf12 -->
- [x] Publish TypeScript declarations and one compile-only no-browser example covering a pure lambda and a bounded Promise-returning capability lambda <!-- t:lf13 -->
- [x] Add unit, concurrency, and race tests that exercise two workers, simultaneous invocations, forbidden modules, undeclared capabilities, invalid outcomes, timeout, cancellation, panic, late settlement, discard, and replacement <!-- t:lf14 -->

**Phase 0 gate:** Two owned workers load the same callback registry and execute
concurrent calls under the race detector; forbidden modules and undeclared
capabilities fail closed; a timed-out, interrupted, panicked, or otherwise
unsafe worker is never reused.

**Validation:**

```bash
go test ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
go test -race ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
GOWORK=off go test ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
```

**Suggested commits:** contracts and validation; compiler/module/fingerprints;
owned runtime and Promise invocation; Phase 0 declarations, examples, and
acceptance tests.

### Phase 1: explicit continuation domain

**Purpose:** Persist browser-spanning workflow state as native versioned data,
never as a suspended VM, goroutine, closure, or Promise.

**Depends on:** Phase 0 artifact, handler identity, schema, and fingerprint
contracts.

**Deliverables:** `pkg/idpcontinuation`, memory and SQLite implementations, and
generation-aware create/load/advance/consume/cleanup semantics.

#### Phase 1 tasks

- [x] Define versioned `WorkflowContinuation`, public carry, native evidence references, native secret references, presentation state, generation binding, expiry, and terminal outcome types without embedding Goja values <!-- t:lf15 -->
- [x] Define a narrow continuation store interface and service contract for create, load, advance, terminal consume, revoke, and cleanup operations <!-- t:lf16 -->
- [x] Generate high-entropy public handles, store only keyed handle hashes, and bind every record to workflow, handler, client, browser, generation, revision, and expiry <!-- t:lf17 -->
- [x] Implement memory storage with atomic compare-and-advance and terminal consume semantics <!-- t:lf18 -->
- [x] Add the SQLite migration and SQLite store implementation with the same atomicity, conflict, expiry, and one-use behavior <!-- t:lf19 -->
- [x] Validate resumed input and carry against the destination handler schema and reject unknown handlers, incompatible schema versions, oversized state, and forbidden secret values <!-- t:lf20 -->
- [x] Implement generation lookup and pinning so a continuation resumes only against its compatible compiled program generation <!-- t:lf21 -->
- [x] Define the safe terminal response and audit classification for missing, expired, replayed, browser-mismatched, client-mismatched, and unavailable-generation continuations <!-- t:lf22 -->
- [x] Implement cleanup that expires continuations and removes attached native pending-secret or challenge state without exposing raw handles <!-- t:lf23 -->
- [x] Add a reusable store conformance suite covering create/load, one-use advance, one-use consume, conflict, replay, expiry, revocation, cleanup, and concurrent POST races <!-- t:lf24 -->
- [x] Add a restart integration test that creates with one service/store instance and resumes with another while proving no runtime or Goja heap object is retained <!-- t:lf25 -->

**Phase 1 gate:** Memory and SQLite pass the same conformance suite, exactly one
concurrent advance succeeds, restart/resume works without a Goja object, and
every mismatch or replay fails safely.

**Validation:**

```bash
go test ./pkg/idpcontinuation ./pkg/memorystore ./pkg/sqlitestore -count=1
go test -race ./pkg/idpcontinuation ./pkg/memorystore ./pkg/sqlitestore -count=1
```

**Suggested commits:** continuation contracts and service; memory store and
conformance suite; SQLite migration/store; generation routing, cleanup, and
restart tests.

### Phase 2: generic provider-owned presentation

**Purpose:** Let a handler select a native form and its continuation without
granting JavaScript HTTP parsing, response-writing, template, header, cookie,
CSRF, or origin authority.

**Depends on:** Phase 1 continuation create/advance semantics.

**Deliverables:** Typed field/action registries, generic workflow page rendering,
presentation outcome builders, and exact POST projection.

#### Phase 2 tasks

- [x] Define stable typed field descriptors for the design-03 signup surface, including value kind, normalization, requiredness, bounds, sensitivity, autocomplete, and public redisplay policy <!-- t:lf26 -->
- [x] Define stable action descriptors and require every submitted action to match an action declared by the active presentation <!-- t:lf27 -->
- [x] Generalize `pkg/idpui` with a `WorkflowPage` or compatible `InteractionPage` extension that renders only host-validated descriptors and never script-supplied HTML <!-- t:lf28 -->
- [x] Implement `ctx.present.form` and the Phase 2 field/action builders so an outcome names the resume handler, allowed edge, fields, actions, public values, errors, carry, and expiry <!-- t:lf29 -->
- [x] Validate presentation outcomes against the compiled handler graph, field/action registries, schema bounds, capability declarations, and continuation expiry limit before persistence or rendering <!-- t:lf30 -->
- [x] Implement native GET rendering with the current CSP, security headers, CSRF token, origin policy, browser binding, body limit, and cache behavior unchanged <!-- t:lf31 -->
- [x] Implement native POST parsing that rejects duplicate singleton fields, missing or extra fields, unknown actions, invalid encodings, oversized values, and normalization failures before invoking JavaScript <!-- t:lf32 -->
- [x] Project sensitive inputs as invocation-scoped opaque handles and project only explicitly public normalized values as ordinary JavaScript data <!-- t:lf33 -->
- [x] Implement rerender behavior using stable public error codes and field errors without reflecting secrets, raw exceptions, or attacker-controlled HTML <!-- t:lf34 -->
- [x] Add renderer, request-validation, CSRF, origin, browser-binding, field-set, normalization, secret-redaction, and browser smoke tests for the current signup form <!-- t:lf35 -->

**Phase 2 gate:** A checked script can present, submit, reject, and rerender the
current signup form through native HTTP handling, with existing browser security
tests intact and no secret exposed to JavaScript as a normal string.

**Validation:**

```bash
go test ./pkg/idpui ./pkg/idpworkflow ./internal/fositeadapter -count=1
go test -race ./pkg/idpui ./pkg/idpworkflow ./internal/fositeadapter -count=1
```

**Suggested commits:** field/action and presentation contracts; native renderer;
exact POST projection and secret handles; Phase 2 integration and smoke tests.

### Phase 3: signup workflow vertical slice

**Purpose:** Replace the hardcoded open-signup branch with named lambdas while
preserving the existing OAuth, PKCE, account, password, consent, session, audit,
and callback behavior.

**Depends on:** Phase 0 invocation, Phase 1 continuations, and Phase 2
presentation.

**Deliverables:** `signup.start`, `signup.submitted`, native signup effects, one
named atomic commit operation, and a checked-in open-signup JavaScript program.

#### Phase 3 tasks

- [x] Add immutable workflow input projection for the validated client, authorization request, browser session, interaction, presentation, carry, and existing native evidence views needed by signup <!-- t:lf36 -->
- [x] Route only natively validated `tinyidp_signup=1` authorization interactions into the configured `signup.start` handler <!-- t:lf37 -->
- [x] Persist the `signup.start` presentation as a continuation and route a natively validated POST into the declared `signup.submitted` handler <!-- t:lf38 -->
- [x] Implement invocation-scoped password secret handles and prevent their serialization into carry, continuation rows, logs, traces, metrics, or JavaScript strings <!-- t:lf39 -->
- [x] Define and validate the native password-credential and local-identity effect plans emitted by `signup.submitted` <!-- t:lf40 -->
- [x] Add `SignupCommitter` as the single named atomic operation for identity creation, credential creation, continuation consumption, interaction update, and session effects <!-- t:lf41 -->
- [x] Map duplicate login, password-policy rejection, invalid input, store conflict, and internal failure to stable non-enumerating workflow outcomes and audit events <!-- t:lf42 -->
- [x] Express the current open-signup behavior entirely in a checked-in JavaScript program using `signup.start`, `signup.submitted`, `ctx.present.form`, and `ctx.commit.signup` <!-- t:lf43 -->
- [x] Thread the activated program/executor through `pkg/embeddedidp/options.go`, the Fosite adapter, and production serving without transferring listener, TLS, OAuth, key, cookie, or store ownership to JavaScript <!-- t:lf44 -->
- [x] Differential-test the hardcoded and scripted paths for equivalent successful and failing behavior, then delete the hardcoded registration branch instead of retaining an adapter or fallback <!-- t:lf45 -->
- [x] Run the existing PKCE registration, replay, CSRF, origin, duplicate-login, password-policy, consent, session, audit, callback, and SQLite test suites through the scripted path <!-- t:lf46 -->

**Phase 3 gate:** Open signup is implemented by the checked-in JavaScript
workflow, the former hardcoded registration branch is gone, and all existing
security and protocol behavior passes without a compatibility path.

**Validation:**

```bash
go test ./internal/fositeadapter ./pkg/embeddedidp ./pkg/idpaccounts ./pkg/memorystore ./pkg/sqlitestore -count=1
go test -race ./internal/fositeadapter ./pkg/idpworkflow ./pkg/idpscript -count=1
go test ./... -count=1
```

**Suggested commits:** signup projections and routing; secret/effect contracts;
atomic signup committer; checked-in JS program and wiring; differential removal
of the hardcoded branch and full regression tests.

### Phase 4: virtual identity and invitation providers

**Purpose:** Prove that identity and eligibility resources may be computed or
cryptographically verified while preserving the same typed workflow and native
commit boundaries.

**Depends on:** Stable Phase 3 signup semantics.

**Deliverables:** Identity and invite provider contracts plus virtual identity,
signed stateless invite, capability-computed invite, and durable one-time invite
implementations.

#### Phase 4 tasks

- [x] Define typed identity-provider and invite-provider contracts with stable provider IDs, declared capabilities/effects, bounded inputs, normalized outputs, and explicit replay/revocation semantics <!-- t:lf47 -->
- [x] Add JavaScript provider registration and invocation through the same artifact registry, worker ownership, budgets, and schema validation used by workflow lambdas <!-- t:lf48 -->
- [x] Implement deterministic virtual subject derivation and protected claim projection without requiring a local user row <!-- t:lf49 -->
- [x] Implement signed stateless invitation verification with audience, issuer, expiry, policy-version, and subject/email constraints expressed as native validated output <!-- t:lf50 -->
- [x] Implement a capability-backed computed invitation example whose eligibility decision is bounded and whose result contains no ambient database or network authority <!-- t:lf51 -->
- [x] Implement a durable one-time invitation provider with hashed lookup, expiry, revocation, atomic redemption, and replay-safe evidence <!-- t:lf52 -->
- [x] Integrate provider results into the Phase 3 signup workflow without changing the native signup commit authority or duplicating account/session logic <!-- t:lf53 -->
- [x] Add checked-in examples for open signup, allowed email domain, signed invitation, computed eligibility, durable one-time invitation, virtual identity, and local stored identity <!-- t:lf54 -->
- [x] Extend explain output to state for each provider whether state exists, where replay is prevented, how revocation works, what identity is materialized, and which native effects may occur <!-- t:lf55 -->
- [x] Add a table-driven provider matrix covering success, denial, malformed data, expiry, revocation, replay, capability failure, timeout, virtual subject stability, claim protection, and atomic one-time redemption <!-- t:lf56 -->

**Phase 4 gate:** All design-03 signup policies run through the same workflow
API, virtual identities require no local row, and durable one-time invitation
redemption remains atomic with signup completion.

**Validation:**

```bash
go test ./pkg/idpprogram ./pkg/idpworkflow ./pkg/idpscript ./pkg/idpaccounts -count=1
go test ./pkg/memorystore ./pkg/sqlitestore ./internal/fositeadapter -count=1
go test -race ./pkg/idpworkflow ./pkg/idpscript -count=1
```

**Suggested commits:** provider contracts; virtual identity and signed invite;
computed and durable invite providers; examples, explain output, and provider
matrix tests.

### Phase 5: email challenge and multi-request signup

**Purpose:** Exercise explicit browser continuations with native, restartable,
one-use verified-email evidence.

**Depends on:** Phase 3 signup commit semantics and Phase 4 provider contracts.

**Deliverables:** Native pending-email challenge state, mailer capability,
email-code presentation, atomic verification evidence, and a restartable signup
workflow.

#### Phase 5 tasks

- [x] Define versioned pending-email challenge, challenge reference, verified-email evidence, attempt, resend, expiry, and terminal-consumption contracts <!-- t:lf57 -->
- [x] Add memory and SQLite challenge persistence with hashed codes, one-use atomic consumption, continuation binding, cleanup, and conformance tests <!-- t:lf58 -->
- [x] Add a narrowly typed mailer capability that accepts a native challenge reference and approved template data rather than arbitrary SMTP, network, or message authority <!-- t:lf59 -->
- [x] Implement native challenge creation and bounded email dispatch with redacted audit events and retry classification <!-- t:lf60 -->
- [x] Add typed email-code field/action descriptors and `ctx.challenge` outcomes that name the resume handler and declared evidence <!-- t:lf61 -->
- [x] Validate and atomically consume a submitted code before producing unforgeable verified-email evidence for the resumed lambda <!-- t:lf62 -->
- [x] Enforce attempt limits, resend policy, expiry, browser/client/workflow/generation binding, cleanup, and generic non-enumerating public errors <!-- t:lf63 -->
- [x] Update the signup workflow to collect the password only after email verification, or use the approved native pending-credential reference if collection must precede suspension <!-- t:lf64 -->
- [x] Bind verified-email evidence into `SignupCommitter` and reject any script-created or stale substitute <!-- t:lf65 -->
- [x] Add restart, replay, concurrent-submit, wrong-code, expired-code, resend, attempt-limit, browser mismatch, client mismatch, generation mismatch, mailer failure, and cleanup integration tests <!-- t:lf66 -->

**Phase 5 gate:** A process restart between send and verify preserves the signup
flow, exactly one correct submission yields native verified-email evidence, and
all replay or binding failures terminate safely without leaking account state.

**Validation:**

```bash
go test ./pkg/idpcontinuation ./pkg/idpworkflow ./pkg/memorystore ./pkg/sqlitestore -count=1
go test ./internal/fositeadapter ./pkg/embeddedidp -count=1
go test -race ./pkg/idpcontinuation ./pkg/idpworkflow ./pkg/sqlitestore -count=1
```

**Suggested commits:** challenge contracts and stores; mailer capability and
dispatch; code presentation/verification; workflow integration; restart and
adversarial tests.

### Phase 6: validation, tests, explanation, activation, and operations

**Purpose:** Make a compiled program safe to inspect, test, activate, observe,
retain for live continuations, replace, and roll back in production.

**Depends on:** Stable Phase 0–5 artifact, runtime, continuation, provider, and
effect contracts.

**Deliverables:** Script CLI commands, embedded tests, atomic generation
manager, operational health/metrics/audit surfaces, and rollback behavior.

#### Phase 6 tasks

- [x] Implement `tinyidp script validate` to compile, canonicalize, validate, bind a named production profile, and report stable source/program/schema/callback fingerprints without executing requests <!-- t:lf67 -->
- [x] Implement collection and execution of embedded program tests with deterministic fake clock, random, mailer, identity, invitation, and store capabilities <!-- t:lf68 -->
- [x] Implement `tinyidp script test` with stable per-case diagnostics and nonzero exit behavior for compilation, binding, assertion, timeout, or leak failures <!-- t:lf69 -->
- [x] Implement `tinyidp script explain` for workflows, handler edges, schemas, capabilities, effects, budgets, continuations, providers, state/replay/revocation semantics, and production-profile violations <!-- t:lf70 -->
- [x] Implement a generation manager that compiles, validates, binds capabilities, creates workers, verifies fingerprints, runs embedded tests, and warms the pool before activation <!-- t:lf71 -->
- [x] Atomically activate only a fully ready generation and leave the previous generation active after compile, validation, binding, test, fingerprint, warmup, or readiness failure <!-- t:lf72 -->
- [x] Retain a bounded set of old generations required by compatible continuations, route resumes by generation, drain unused workers, and close generations without goroutine or runtime leaks <!-- t:lf73 -->
- [x] Expose readiness failure when the active generation or required native bindings are unavailable, without reporting a merely compiled artifact as ready <!-- t:lf74 -->
- [x] Add bounded-cardinality metrics for pool saturation, invocation outcome/latency, interruption/discard, continuation create/resume/replay/expiry, activation, retained generations, and cleanup <!-- t:lf75 -->
- [x] Add redacted activation and runtime audit records containing stable program/source hashes and diagnostic IDs but no subjects, emails, invite codes, passwords, raw exceptions, or unbounded callback labels <!-- t:lf76 -->
- [x] Add failure-matrix and repeated-reload tests covering every pre-activation failure, atomic swap, in-flight request consistency, continuation generation routing, rollback, draining, resource cleanup, and goroutine/runtime leaks <!-- t:lf77 -->

**Phase 6 gate:** Every activation prerequisite succeeds before swap; any failure
leaves the previous generation serving; live compatible continuations resume on
their retained generation; repeated reload tests show no request inconsistency
or resource leak.

**Validation:**

```bash
go run ./cmd/tinyidp script validate --help
go run ./cmd/tinyidp script test --help
go run ./cmd/tinyidp script explain --help
go test ./pkg/idpprogram ./pkg/idpscript ./pkg/idpworkflow -count=1
go test -race ./pkg/idpscript ./pkg/idpworkflow -count=1
go test ./... -count=1
```

**Suggested commits:** validate command; embedded tests and test command;
explain command; generation manager and atomic activation; observability,
retention, rollback, and repeated-reload tests.

### Phase 7: authorization, claims, and additional existing workflows

**Purpose:** Reuse the proven lambda contracts beyond signup while keeping
native OAuth validation, protocol transitions, token issuance, credential
effects, and RFC 8628 state ownership in Go.

**Depends on:** Phase 3 signup semantics must be stable and Phase 6 activation
and operational gates must pass.

**Deliverables:** Authorization and claims lambdas plus carefully bounded
presentation integration for existing account-selection, consent,
password-recovery, and device-verification flows described by design 03.

#### Phase 7 tasks

- [x] Define immutable authorization input, allowed allow/deny/skip outcomes, declared evidence, and stable denial diagnostics without exposing Fosite or token mutation authority <!-- t:lf78 -->
- [x] Invoke authorization lambdas only after native client, redirect URI, scope, PKCE, prompt, session, authentication, and request validation and before native consent/code issuance <!-- t:lf79 -->
- [x] Define immutable claims input and bounded claims output with native protection for issuer, subject, audience, expiry, nonce, authentication time, and other protocol-owned claims <!-- t:lf80 -->
- [x] Invoke claims lambdas before native OIDC session persistence and preserve refresh-token and UserInfo consistency <!-- t:lf81 -->
- [x] Express account selection and consent as workflow handlers only where the Phase 0–6 presentation/continuation contracts fit without moving OAuth decisions or response writing into JavaScript <!-- t:lf82 -->
- [x] Add password recovery only through existing native credential and challenge effects; do not permit scripts to write password hashes, credentials, or recovery state directly <!-- t:lf83 -->
- [x] Integrate device-verification presentation through workflow handlers while leaving RFC 8628 device/user code generation, polling, expiry, authorization, denial, and token transitions native <!-- t:lf84 -->
- [x] Add strict tests for fresh login, existing session, `prompt=none`, account selection, consent, allow/deny/error, protected claims, refresh, UserInfo, recovery replay, and device verification <!-- t:lf85 -->
- [x] Run the complete conformance, security-model, race, browser, memory-store, and SQLite-store suites before enabling any Phase 7 handler in a production profile <!-- t:lf86 -->

**Phase 7 gate:** Authorization and claims customization cannot bypass native
protocol checks or protected claims, and all current browser, refresh/UserInfo,
device-flow, conformance, and security-model tests remain intact.

**Validation:**

```bash
go test ./internal/fositeadapter ./internal/server ./pkg/embeddedidp ./pkg/idpui -count=1
go test -race ./internal/fositeadapter ./pkg/idpscript ./pkg/idpworkflow -count=1
go test ./... -count=1
```

**Suggested commits:** authorization contracts and seam; claims contracts and
seam; account/consent/recovery presentation integration; device-verification
presentation; full Phase 7 regression and conformance gate.

### Cross-phase assurance required by design 03

These checks apply to the implementation above; they are not a separate feature
track and must not delay the Phase 0 vertical spike with unrelated refactoring.

- [ ] Assign stable IDs to handlers, outcomes, schemas, capabilities, effects, evidence, observations, and validation diagnostics as each Phase 0–7 contract lands <!-- t:lf87 -->
- [ ] Emit secret-free traces at native invocation, continuation, evidence, effect-validation, commit, and terminal boundaries using only bounded stable dimensions <!-- t:lf88 -->
- [ ] Model declared lambda outcomes as constrained nondeterministic transitions and prove that native evidence plus an approved atomic commit is required before identity/session/token-relevant completion <!-- t:lf89 -->
- [ ] Add property, fuzz, race, replay, and normalized-trace tests at the phase that introduces each codec, transition, store, worker lifecycle, or effect boundary <!-- t:lf90 -->
- [ ] Keep production scripting and `tinyidp/verify` runtime/module/capability profiles separate; do not make verification tooling a prerequisite for request-path execution <!-- t:lf91 -->

### Overall completion gate

- [ ] Every Phase 0–7 gate passes in order, all normative examples and TypeScript declarations match the implementation, `go test ./...`, targeted race suites, and `GOWORK=off` checks pass, and the implementation diary identifies the exact commits and validation evidence for each phase <!-- t:lf92 -->
