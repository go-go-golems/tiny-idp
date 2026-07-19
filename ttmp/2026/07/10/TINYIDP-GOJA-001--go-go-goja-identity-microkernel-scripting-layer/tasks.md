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
- [ ] Inventory and crosswalk every current required-action bit, state-model action, VerificationPlan step, assertion ID, security event, model action, and static property ID. <!-- t:agxk -->
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

- [x] Lambda-first design: publish the normative intern implementation guide and clearly deprecate design-doc/01 <!-- t:vnp5 -->
- [ ] Lambda API Phase 0: define pure-Go program, workflow, lambda, outcome, effect, capability, schema, and validation contracts <!-- t:6h0x -->
- [ ] Lambda API Phase 0: implement isolated require("tinyidp").v1 compiler and deterministic callback fingerprints <!-- t:6hu5 -->
- [ ] Lambda API Phase 0: implement owner-safe Promise-aware worker invocation, interruption, discard, and negative ambient-module tests <!-- t:nj6d -->
- [ ] Lambda API Phase 1: implement versioned explicit workflow continuations with memory and SQLite conformance tests <!-- t:v4n3 -->
- [ ] Lambda API Phase 2: implement generic provider-owned presentation fields, actions, native rendering, and exact POST validation <!-- t:ctai -->
- [ ] Lambda API Phase 3: replace hardcoded provider registration with the scripted signup workflow and named native commit operation <!-- t:xv3m -->
- [ ] Lambda API Phase 4: implement virtual identity, signed virtual invite, computed invite, and durable one-time invite providers <!-- t:2gvn -->
- [ ] Lambda API Phase 5: implement native email-code challenges and restartable verified-email signup continuation <!-- t:lre3 -->
- [ ] Lambda API Phase 6: implement program validate, test, explain, activation, generation retention, readiness, metrics, and rollback <!-- t:7xil -->
- [ ] Lambda API Phase 7: add authorization and claims lambdas after signup semantics stabilize <!-- t:p90i -->
- [ ] Lambda API assurance: model lambda contracts as constrained nondeterministic transitions and emit secret-free runtime traces <!-- t:77za -->
- [ ] Lambda API delivery: validate docmgr bookkeeping and upload the superseding design bundle to reMarkable <!-- t:d8yk -->
