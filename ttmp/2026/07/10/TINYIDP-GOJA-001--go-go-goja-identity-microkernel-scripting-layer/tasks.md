# Tasks

## Research and design deliverable

- [x] Create `TINYIDP-GOJA-001` with architecture/auth/Go/OIDC/research/testing/xgoja topics
- [x] Move `/tmp/idp-research.md` into the ticket `sources/` directory
- [x] Read the colleague identity-microkernel research completely
- [x] Map current tiny-idp engines, strict request flow, persistence, lifecycle, and extension seams
- [x] Map current go-go-goja runtime ownership, module selection, interruption, and xgoja provider APIs
- [x] Run the current tiny-idp test baseline
- [x] Write the intern-oriented analysis/design/implementation guide
- [x] Record compact architecture decisions, alternatives, risks, and open questions
- [x] Create a phased implementation and acceptance plan
- [x] Validate ticket frontmatter and docmgr health
- [x] Upload the design and diary bundle to reMarkable

## Phase 0: contract spike and dependency decision

- [ ] Decide whether tiny-idp may raise its minimum Go version from 1.25.11 to 1.26.1+
- [ ] Pin a released or exact go-go-goja version in `tiny-idp/go.mod`
- [ ] Build an explicit compile-only `require("tinyidp")` native module spike
- [ ] Disable implicit and data-only default modules in compiler and policy factories
- [ ] Add negative tests for `fs`, `exec`, `database`, `os`, `process`, network, and arbitrary loaders
- [ ] Compile one Goja program and load it independently into multiple owned runtimes
- [ ] Implement and race-test execution deadline interruption and `ClearInterrupt` ordering
- [ ] Publish the supported JavaScript syntax profile
- [ ] Pass direct, `GOWORK=off`, and race-test gates

## Phase 1: pure-Go graph and validation

- [ ] Add `pkg/idpgraph` schema, stable enums, nodes, slots, callbacks, tests, and diagnostics
- [ ] Add canonical JSON and deterministic graph/source hashing
- [ ] Add native block descriptor registry with input/output/effect/capability metadata
- [ ] Model the current strict OIDC/password/consent/claims/issuance flow
- [ ] Add a pure-Go `localWeb` preset
- [ ] Implement reference, cycle, reachability, termination, slot, capability, effect, and production-profile validation
- [ ] Materialize the pure-Go graph into the existing strict provider
- [ ] Add deterministic snapshots and malformed-graph tests

## Phase 2: fluent compiler module

- [ ] Add `pkg/idpscript` compiler/artifact contracts
- [ ] Add `pkg/gojamodules/tinyidp` with `v1` API and lowerCamelCase data
- [ ] Implement `DraftCollector` and branded fluent builder objects
- [ ] Implement `preset`, `ref`, `protocol`, `authn`, `policy`, `claims`, `consent`, `issue`, `decision`, and `slot` v1 subsets
- [ ] Require explicit callback names and deterministic registration IDs
- [ ] Bound source/config/object sizes and reject non-serializable values
- [ ] Export graph through `module.exports`/`build()` without starting services
- [ ] Add TypeScript declarations
- [ ] Add `tinyidp script validate` and canonical graph output
- [ ] Add checked-in local-web script example

## Phase 3: strict authorization and claims seams

- [ ] Add immutable public subject/client/request/authentication context DTOs
- [ ] Add allow/deny `AuthorizationPolicy` and `PolicySet`
- [ ] Add `ClaimsPolicy` with protected protocol claim names
- [ ] Thread policies through `embeddedidp.Options` and Fosite adapter options
- [ ] Invoke authorization policy after native validation and before consent/code issuance
- [ ] Invoke claims policy before OIDC session persistence
- [ ] Propagate AMR/ACR through browser and OIDC sessions
- [ ] Add static native RBAC and claims blocks
- [ ] Test fresh login, existing session, prompt none, consent, deny/error, refresh, and UserInfo
- [ ] Re-run strict conformance gates

## Phase 4: policy pool and capabilities

- [ ] Add same-source callback registry and fingerprint verification per worker
- [ ] Add bounded single-owner worker acquisition and lifecycle
- [ ] Add timeout/panic/invalid-output worker discard and replacement
- [ ] Add bounded immutable JS input/output codecs
- [ ] Add typed capability descriptors, registry, permissions, effects, and budgets
- [ ] Add JavaScript authorization and claims callback nodes
- [ ] Add pool/generation readiness and metrics
- [ ] Add saturation, exception, timeout, capability failure, and concurrent strict-flow tests
- [ ] Run race and production-shaped mixed-load gates

## Phase 5: tests, explain, and atomic activation

- [ ] Collect and run `A.test` cases with deterministic fake capabilities
- [ ] Add `tinyidp script test` and `tinyidp script explain`
- [ ] Add generation manager with warmup, atomic swap, drain, close, and rollback
- [ ] Add opt-in file-watch reload
- [ ] Audit activation attempts using source and graph hashes
- [ ] Keep previous generation active after compile/test/warmup failures
- [ ] Test repeated reloads for request consistency and resource/goroutine leaks

## Phase 6: xgoja/v2 packaging

- [ ] Add `pkg/xgoja/providers/tinyidp`
- [ ] Add provider module config schema and TypeScript descriptor
- [ ] Add stable typed host-service key and lookup
- [ ] Add compile-only `xgoja.yaml` example
- [ ] Add Go-owned provider command set or generated runtime-package serving example
- [ ] Test provider registry and module factory behavior
- [ ] Pass `xgoja doctor`, `xgoja gen-dts`, and `xgoja build`

## Phase 7: typed challenge graph

- [ ] Refactor password and existing-session behavior behind native block interfaces
- [ ] Implement complete five-outcome transition tables
- [ ] Add durable hashed one-time challenge continuation contract and SQLite schema
- [ ] Bind challenges to graph generation, flow, client, subject, and expiry
- [ ] Implement native interaction renderer contract
- [ ] Add `seq`, `all`, `firstAvailable`, `when`, and `choose` with table-driven tests
- [ ] Add native evidence, AMR, ACR, and step-up propagation
- [ ] Implement one complete native factor (recommended passkey) before plugin factors
- [ ] Threat-model downgrade, fallthrough, replay, retry, and cross-generation resume

## Phase 8: additional native blocks and protocols

- [ ] Prioritize upstream OIDC, production device flow, and dynamic step-up
- [ ] Add token exchange and transaction authorization as separate native slices
- [ ] Keep CIBA, workload identity, quorum, edge, and multi-actor flows experimental until abstractions stabilize
- [ ] Require native storage/protocol support, graph descriptors, interoperability tests, docs, and production review for each slice
- [ ] Define a separate tinyidp/verify module and pure-Go VerificationPlan schema for offline security scenarios <!-- t:e7z1 -->
- [ ] Specify verification runtime profiles and prove production policy runtimes cannot resolve test-only capabilities <!-- t:4phg -->
- [ ] Prototype scenario compilation and Go-owned execution against redacted traces, fake clock, and failpoint adapters <!-- t:7aql -->
- [ ] Add negative tests preventing verification plugins from overriding production authorization decisions or suppressing invariant failures <!-- t:4dw1 -->
