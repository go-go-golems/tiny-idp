---
Title: Static Analysis and Implementation Verification Research Design and Implementation Guide
Ticket: TINYIDP-STATIC-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - security
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Primary authorization and token control flow for path-sensitive rules
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Transaction and protocol lifecycle effects for static verification
    - Path: repo://pkg/idpstore/types.go
      Note: Authoritative interaction states and required actions
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md
      Note: Prior assurance architecture and research foundation
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Existing fifteen-rule go/analysis prototype to inventory and deliberately promote
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main_test.go
      Note: Existing analysistest coverage baseline
    - Path: repo://ttmp/2026/07/11/TINYIDP-MODEL-001--serious-model-checking-and-formal-state-assurance-for-tiny-idp/design-doc/02-serious-model-checking-system-architecture-and-implementation-plan.md
      Note: Shared invariant catalog and distinct model-checking evidence boundary
ExternalSources:
    - https://pkg.go.dev/golang.org/x/tools/go/analysis
    - https://pkg.go.dev/golang.org/x/tools/go/ssa
    - https://codeql.github.com/docs/codeql-language-guides/analyzing-data-flow-in-go/
    - https://github.com/viperproject/gobra
Summary: Research-backed system design for repository-specific Go static analysis, dataflow verification, taint tracking, mutation evaluation, and selected deductive verification of tiny-idp security invariants.
LastUpdated: 2026-07-12T01:15:00Z
WhatFor: Defines how tiny-idp turns security invariants into maintained AST, CFG, SSA, interprocedural, taint, typestate, and verification evidence.
WhenToUse: Read before implementing, promoting, reviewing, suppressing, or interpreting any tiny-idp security analyzer or implementation-verification experiment.
---


# Static Analysis and Implementation Verification

## Executive summary

tiny-idp already has a useful repository-specific `go/analysis` prototype. Its
fifteen registered analyzers detect unsafe API exposure, ignored randomness and
security errors, implicit HTTP server construction, weak security defaults,
unstable limiter identities, unused configuration, ignored audit delivery,
suspicious multi-mutation functions, unsafe backup copying, implicit bearer
transport, direct wall-clock access, fail-open security parsing, browser-owned
interaction continuation, and incomplete protocol lifecycles.

That prototype is valuable but it is not yet an implementation-verification
system. Most rules recognize local syntax and typed call sites. They do not
generally prove that all paths to authorization-code issuance satisfy a prior
authentication obligation, that `MustChangePassword` cannot flow to success, or
that unverified browser data cannot influence a security decision through helper
functions and interfaces. The prototype also lacks a stable rule catalog,
precision measurements, mutation benchmarks, interprocedural summaries, release
evidence, and a stated boundary between linting and proof.

This design creates six cooperating layers:

1. a canonical invariant and transition-authority catalog shared with
   `TINYIDP-MODEL-001`;
2. high-confidence AST and type-aware structural analyzers;
3. CFG and SSA analyzers for dominance, path state, values, and effects;
4. interprocedural facts and taint summaries for security-relevant flows;
5. mutation and historical-defect benchmarks that measure sensitivity and
   noise; and
6. optional deductive verification experiments for small pure or transactional
   kernels, explicitly separated from ordinary static diagnostics.

The first serious milestone is one vertical property. The recommended property
is authorization issuance under forced reauthentication:

```text
If an interaction requires fresh authentication, every path that can create an
authorization artifact must contain a successful authentication transition for
that interaction after its creation and must not bypass a required password
change.
```

The milestone is complete only when a documented abstract interpretation is
implemented, the analyzer finds seeded versions of both historical defects,
safe and near-miss fixtures remain quiet, whole-repository runtime is bounded,
and the rule maps to the same invariant used by models, runtime monitors, and Go
regressions.

## 1. Problem statement and scope

Security failures in identity systems frequently span several operations. A GET
request validates an authorization request and establishes obligations. A later
POST authenticates, obtains consent, and completes authorization. Token storage
may cross several Fosite callbacks. A locally plausible branch can violate the
global protocol transition.

Static analysis searches program representations without executing every input.
It can reject dangerous code shapes, approximate possible control/data flows,
and enforce architectural constraints across the repository. It cannot, by
itself, prove arbitrary semantic properties of Go, dependencies, databases, or
deployed infrastructure. A credible program must state the abstraction, its
soundness goal, and known false-positive and false-negative boundaries.

### 1.1 In scope

- Go AST, type information, control-flow graphs, SSA, call graphs, pointer/value
  approximation, and `analysis.Fact` summaries;
- repository-specific rules derived from concrete tiny-idp invariants;
- taint and information-flow approximations for secrets and untrusted inputs;
- typestate-like analysis for interaction and protocol lifecycles;
- mutation testing and historical-defect replay for analyzers;
- comparisons with CodeQL, Semgrep, gosec, Staticcheck, and Go vet;
- limited deductive verification experiments for well-isolated kernels;
- reproducible diagnostics, SARIF or equivalent output, CI, suppressions,
  ownership, and release evidence; and
- a shared mapping among static rules, formal properties, runtime events, and
  regression tests.

### 1.2 Out of scope initially

- claiming whole-program correctness;
- proving Fosite, SQLite, TLS, Go runtime, or cryptographic implementations;
- making every analyzer globally sound for arbitrary Go repositories;
- using regex-only tools as proof of control or dataflow properties;
- silently accepting analyzer crashes, skipped packages, or unknown IR;
- automatic fixes for security-sensitive behavior;
- requiring deductive annotations throughout the HTTP provider; and
- treating absence of diagnostics as evidence beyond each rule's declared
  scope.

## 2. Theory primer for an intern

### 2.1 Syntax, semantics, and approximation

An AST preserves source structure. It is ideal for rules such as “production
security packages may not call `time.Now` directly.” It is insufficient for
properties where a value travels through assignments, branches, helper calls,
interfaces, or closures.

A control-flow graph represents basic blocks and possible transfers of control.
It supports reachability, dominance, and must/may path reasoning. SSA gives each
value definition a unique identity and introduces phi nodes where control-flow
paths merge. It makes value provenance more explicit but does not eliminate
aliasing, dynamic dispatch, reflection, goroutines, or dependency modeling.

Static analysis computes an approximation. A may-analysis over-approximates what
might happen and is useful for finding possible unsafe flows, but may report
impossible paths. A must-analysis under-approximates what is guaranteed on all
paths and is useful for checking required gates, but joins often lose facts.

```text
concrete executions       possibly infinite
        | abstraction
        v
abstract states           finite lattice
        | transfer functions
        v
fixed point               conservative result under declared assumptions
```

### 2.2 Lattices and transfer functions

An analyzer state must have an order and a join operation. For a simple
authorization obligation:

```text
Unknown
  |
  +-- NoFreshAuthRequired
  |
  +-- FreshAuthRequired
          |
          +-- FreshAuthSatisfied
          |
          +-- MustChangePassword
```

This picture is only intuitive; the implementation needs a precise product
lattice. Conflicting paths usually join to uncertainty, and uncertainty at an
artifact sink must produce a diagnostic when the property requires a fact on
all paths.

```text
transfer(Begin(requiredLogin=true)) = add RequiresFreshAuth(interaction)
transfer(Authenticate(success=true)) = add AuthenticatedAfterCreation(interaction)
transfer(Authenticate(mustChange=true)) = add PasswordChangeRequired(subject)
transfer(CompletePasswordChange) = remove PasswordChangeRequired(subject)
transfer(IssueArtifact) = assert required facts and absence of blocking facts
```

### 2.3 Intra- versus interprocedural analysis

An intraprocedural analysis stops at function calls or uses handwritten call
summaries. An interprocedural analysis follows calls and returns. Context
sensitivity distinguishes different call sites or receiver states; it improves
precision but increases cost.

The IFDS framework represents suitable finite distributive dataflow problems as
graph reachability over valid interprocedural paths. It is relevant for finite
facts such as “value derived from browser input,” “interaction requires login,”
or “error from security operation remains unchecked.” It is not automatically
the right engine for every rule; the ticket requires a small prototype and
measured comparison before adopting a general framework.

### 2.4 Taint analysis

Taint analysis defines sources, propagators, barriers/sanitizers, and sinks.
Cryptographic hashing is not automatically a sanitizer. Hashing an
attacker-selected limiter key still permits attacker-selected buckets. Parsing
is not validation. Reading an unverified JWT claim remains untrusted even if the
JSON shape is valid.

### 2.5 Typestate and protocol state

Typestate associates allowed operations with abstract object state. tiny-idp
examples include `pending -> approved|denied|expired`, transaction
`none -> active -> committed|rolledBack`, and token-family
`active(generation) -> rotated(g+1)|revoked`.

Go types can encode some state transitions structurally, but persisted records,
interfaces, HTTP phases, and Fosite callbacks prevent a complete compile-time
encoding. Static typestate analysis can still check local transition authority
and required operation ordering.

### 2.6 Deductive verification

Deductive verification uses specifications such as preconditions,
postconditions, invariants, permissions, and ghost state to generate proof
obligations, commonly discharged by SMT solvers. Gobra targets Go and supports a
substantial language subset through the Viper infrastructure. This is a
different evidence class from `go/analysis` diagnostics.

A successful Gobra experiment can support a property of the annotated kernel
under its verifier semantics and assumptions. It does not prove the HTTP
provider, Fosite, SQLite, or glue code unless those are included and specified.

## 3. Current-state evidence

### 3.1 Existing analyzer registry

The prototype is located at
`ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go`.
Lines 23–40 register fifteen analyzers:

| Analyzer | Present technique | Intended property |
|---|---|---|
| `tinyidpinternalapi` | types/packages | public APIs do not expose internal types |
| `tinyidprand` | AST + typed call | cryptographic randomness errors are checked |
| `tinyidphttpserver` | AST | production server lifecycle is explicit |
| `tinyidpsecuritydefault` | AST | security defaults do not silently allow/no-op |
| `tinyidpratelimitkey` | AST | limiter identity is stable and not raw address |
| `tinyidpconfiguse` | AST/types | public configuration actually affects runtime |
| `tinyidpauditdelivery` | AST | audit delivery results are handled |
| `tinyidpatomicity` | AST heuristics | grouped mutations expose transaction boundary |
| `tinyidpbackupcopy` | AST | backups use safe database mechanisms |
| `tinyidpbearertransport` | AST + object identity | bearer transport is explicit |
| `tinyidpsecurityclock` | AST + package scope | security time is injected |
| `tinyidpstrictparse` | AST branch patterns | security parsing fails closed |
| `tinyidpinteractioncontinuation` | AST call patterns | continuation is server-owned |
| `tinyidpprotocollifecycle` | AST call sets | grouped protocol mutations are atomic |
| `tinyidpignoredsecurityerror` | AST + named methods | critical operation errors are not ignored |

The prototype is 772 lines and has fifteen files including fixtures. Eleven
analyzers have direct `analysistest` entry points in `main_test.go`. The mismatch
between fifteen registered rules and eleven suites is an immediate inventory
task, not proof that four rules are untested: some may share fixtures or lack
direct test functions and must be classified precisely.

### 3.2 Existing strengths

- It uses the official `analysis.Analyzer` API and `multichecker`.
- It resolves typed objects instead of matching only printed call names.
- It uses `inspect.Analyzer` to avoid repeated traversal.
- It contains repository-specific knowledge that general linters cannot infer.
- Several rules directly encode defects or review findings.
- `analysistest` fixtures make diagnostics executable and reviewable.

### 3.3 Existing limitations

- The tool lives under a dated ticket rather than maintained product tooling.
- Rule IDs, severity, confidence, scope, and evidence version are not a stable
  public contract.
- Most rules are syntax-local and not path-sensitive.
- Cross-function value and obligation flow is not generally represented.
- Interface dispatch and dependency summaries are ad hoc.
- There is no mutation score or historical-defect benchmark.
- False-positive and false-negative boundaries are not measured per rule.
- There is no parser-independent machine-readable result envelope.
- Analyzer crash, skipped package, or partial load semantics are not release
  policy.
- There is no shared mapping to model-checking properties or runtime monitors.

## 4. Target assurance architecture

```text
canonical invariant catalog
      | property IDs, code authorities, threats
      v
+-----------------------+       +------------------------+
| structural rules      |       | flow/state rules       |
| AST + types            |       | CFG + SSA + summaries  |
+-----------------------+       +------------------------+
          \                         /
           v                       v
             normalized diagnostics
                      |
        +-------------+-------------+
        |                           |
        v                           v
 mutation/historical corpus   whole-repo evaluation
        |                           |
        +-------------+-------------+
                      v
           versioned evidence envelope
                      |
          +-----------+-----------+
          |                       |
          v                       v
   CI/release gate      models/tests/monitors map

optional isolated branch:
small kernel + contracts -> Gobra/Viper/SMT -> proof artifact
```

### 4.1 Shared invariant catalog

Each property has a stable identifier independent of a tool:

```yaml
id: AUTH-S-004
title: Required fresh authentication persists to issuance
kind: temporal-safety
authoritative_state: InteractionRecord.RequiredActions
transition_authorities:
  - Provider.createInteraction
  - Authenticator.AuthenticatePassword
  - Store.ConsumeInteraction
  - Provider.finishAuthorize
static_rules:
  - STATIC-AUTH-004
model_properties:
  - TLA-AUTH-S-004
runtime_rules:
  - RV-AUTH-004
regressions:
  - TestPromptLoginBlankPOSTCannotReuseSession
```

The catalog prevents analyzer names from becoming the security specification.
It also exposes properties that cannot be checked statically and therefore need
other evidence.

### 4.2 Analyzer maturity levels

| Level | Representation | Claim form |
|---|---|---|
| L0 | textual/regex | candidate location only |
| L1 | AST | forbidden/required syntax in declared scope |
| L2 | AST + types | operation identity and local structural relation |
| L3 | CFG | possible/all paths within one function |
| L4 | SSA | local value/effect provenance across blocks |
| L5 | summaries/facts | bounded interprocedural property |
| L6 | whole-program call/value approximation | entrypoint-to-sink reachability under declared dispatch assumptions |
| L7 | deductive proof | annotated kernel satisfies stated contract under verifier assumptions |

Higher levels are not inherently better. A direct-clock rule is clearer and
more reliable at L2 than through a whole-program solver. Every rule declares the
lowest sufficient level.

## 5. Repository and package design

No nested Go module is created. Maintained tooling uses the top-level module:

```text
cmd/tinyidpvet/
  main.go

internal/staticcheck/
  registry.go
  diagnostic.go
  evidence.go
  config.go
  catalog/
  analyzers/
    securityclock/
    ignoredsecurityerror/
    strictparse/
    interactionflow/
    protocollifecycle/
    secrettaint/
  facts/
    trust.go
    effects.go
    obligations.go
  ir/
    authority.go
    lattice.go
    summary.go
  testutil/

staticcheck/
  catalog.yaml
  config.yaml
  baseline.yaml
  mutations/
  corpus/
  schemas/
```

The package name `staticcheck` may conflict conceptually with the external tool.
Before implementation, maintainers should choose `staticanalysis` if clarity
outweighs brevity. This design does not create a compatibility adapter for the
ticket prototype; useful analyzers are moved deliberately with stable new rule
IDs.

## 6. Core APIs

### 6.1 Rule metadata

```go
type Rule struct {
    ID              string
    PropertyIDs     []string
    Title           string
    Severity        Severity
    Confidence      Confidence
    Maturity        Maturity
    AnalysisLevel   Level
    Scope           Scope
    KnownFalsePos   []string
    KnownFalseNeg   []string
    Owner           string
    Analyzer        *analysis.Analyzer
}
```

An analyzer cannot register without metadata. `Maturity` distinguishes
`experimental`, `advisory`, and `blocking`. Promotion to blocking requires
mutation and corpus thresholds plus owner approval.

### 6.2 Diagnostic schema

```go
type Diagnostic struct {
    SchemaVersion string
    RuleID        string
    PropertyIDs   []string
    Severity      string
    Confidence    string
    Position      Position
    Message       string
    Trace         []FlowStep
    Fingerprint   string
    ToolVersion   string
    SourceCommit  string
}
```

Fingerprints identify the semantic site using rule, package, symbol, and a
normalized local anchor. Line number alone is unstable. Security-sensitive
rules provide remediation prose but no automatic edit.

### 6.3 Interprocedural facts

```go
type FunctionEffects struct {
    Produces       EffectSet
    Requires       ObligationSet
    Discharges     ObligationSet
    PropagatesArgs BitSet
    ReturnsTrust   []TrustClass
    MayIssue       ArtifactSet
}

func (*FunctionEffects) AFact() {}
```

Facts must be versioned because cached exported facts become part of analyzer
semantics. Dependency functions without source receive explicit summaries whose
provenance and version are reviewable.

### 6.4 Trust lattice

```text
Bottom
  -> Constant
  -> ServerOwned
  -> ValidatedProtocolValue
  -> AuthenticatedPrincipal

Bottom
  -> BrowserControlled
  -> ParsedBrowserValue

Unknown joins with either branch to Unknown.
Hash(BrowserControlled) remains BrowserControlled for identity selection.
```

Trust is purpose-specific. A syntactically valid redirect URI is not necessarily
authorized. A verified client ID is not necessarily a stable network limiter
identity. The implementation should prefer separate fact domains over one
universal “safe” bit.

## 7. Priority analyzers

### 7.1 `STATIC-AUTH-004`: authorization obligation dominance

Goal: all artifact-issuing paths satisfy stored required actions.

```text
sources:
  InteractionRecord.RequiredActions
  session freshness classification

dischargers:
  successful AuthenticatePassword after interaction creation
  password-change completion when MustChangePassword was returned
  explicit consent approval for consent obligation

sinks:
  WriteAuthorizeResponse
  CreateAuthorizeCodeSession
  terminal approved ConsumeInteraction

blocking states:
  RequiredLogin not discharged
  MustChangePassword
  interaction expired/consumed
```

The first implementation should analyze the strict provider and approved helper
summaries, not arbitrary HTTP code. It uses CFG/SSA within provider methods and
hand-reviewed summaries across authenticator/store/Fosite boundaries. A later
iteration may export facts across packages.

### 7.2 `STATIC-TRUST-001`: continuation trust

Track values from `Form`, `PostForm`, query, and selected headers. Report when
they reconstruct canonical protocol state or required actions after a stored
interaction exists. Permit opaque handle lookup and CSRF verification, but do
not treat either as authorization-request validation.

### 7.3 `STATIC-ERROR-001`: security error consumption

Extend ignored-call syntax to value flow. An error from randomness, audit,
security event delivery, interaction consumption, transaction commit,
authentication, key lookup, or policy evaluation must reach an approved handler.
Uniform external errors are allowed; collapsing internal unavailable and absent
states before recording cause is not.

### 7.4 `STATIC-PARSE-001`: fail-closed security parsing

Use CFG to identify parse-failure successors. A failure path may return an
error/rejection or preserve “parameter absent” only when absence was established
before parsing. It may not return satisfied/allowed/default-zero when parsing a
present security parameter.

### 7.5 `STATIC-TX-001`: lifecycle effects

Summaries identify grouped effects:

```text
authorization = create code + PKCE + OIDC session + consume interaction
redemption    = invalidate code + create access/refresh rows
rotation      = revoke/rotate old + create replacements + family policy
```

Report a reachable terminal success when required effects are split across no
approved atomic boundary. Static analysis checks structure and declared effects;
failpoint tests remain evidence for actual storage atomicity.

### 7.6 `STATIC-SECRET-001`: secret-to-sink taint

Sources include passwords, cookies, authorization codes, access/refresh tokens,
client secrets, private keys, request-object bytes, and recovery materials.
Sinks include logs, errors, metrics labels, audit attributes, JavaScript values,
redirects, and unapproved persistence. Sanitization is sink-specific. A token
hash may be allowed in correlation logs but not used as a public identifier
without separate review.

## 8. Tool strategy

### 8.1 Native `go/analysis`

This is the primary implementation platform because it shares Go types and
versioning with the repository, integrates with `go vet`-style drivers, supports
required analyzers and facts, and is testable with `analysistest`. Use
`inspect.Analyzer` for syntax, `ctrlflow` for syntactic CFGs, and `buildssa` for
typed SSA. Use callgraph algorithms only when the rule states its dispatch
assumptions.

### 8.2 CodeQL

CodeQL is the primary comparison for global dataflow and taint. It supports Go
dataflow configurations and dependency model packs. The ticket should implement
one equivalent secret or continuation flow in both systems and compare results,
maintenance, CI availability, explanation quality, and proprietary-service
constraints. Duplicate green results are not independent proofs when they share
the same source/sink assumptions.

### 8.3 Semgrep, gosec, Staticcheck, and vet

These tools remain valuable for commodity and fast structural checks. They
should not be stretched into repository-specific temporal proof. The design
records overlap so tiny-idp does not maintain custom rules for well-covered
general defects.

### 8.4 Gobra

Gobra is evaluated only after selecting a small kernel with a valuable contract
and manageable dependencies. Candidate experiments are a pure interaction
transition function, refresh-family state transition, or transaction effect
planner. Success criteria include understandable annotations, reproducible
solver/tool versions, proof runtime, mutation sensitivity, and maintainability by
more than one engineer.

## 9. Analyzer development lifecycle

```text
security defect or invariant
  -> stable property and rule IDs
  -> minimal unsafe/safe/near-miss examples
  -> choose lowest sufficient representation
  -> specify abstract domain and transfer rules
  -> implement analyzer and diagnostics
  -> run analysistest fixtures
  -> run seeded mutations and historical defect
  -> scan whole repository and classify findings
  -> record precision/runtime/coverage
  -> advisory deployment
  -> blocking promotion review
```

Every analyzer document includes scope, threat, sources/sinks or state domain,
transfer semantics, summaries, unsupported constructs, known false positives,
known false negatives, examples, test corpus, runtime budget, owner, and maturity.

## 10. Mutation and empirical evaluation

The benchmark corpus contains real and synthetic mutations:

- remove the required-login POST rejection;
- ignore `MustChangePassword`;
- reconstruct protocol fields from browser form values;
- bypass or split `ConsumeInteraction`;
- issue before durable terminal commit;
- replace injected time with `time.Now`;
- return satisfied on malformed `max_age`;
- ignore audit, security event, randomness, or commit error;
- use unverified `client_id` as the only limiter identity;
- accept bearer tokens from a query parameter;
- log tokens through direct and helper calls;
- split protocol row mutations across functions/interfaces;
- wrap unsafe calls in helpers to test interprocedural sensitivity; and
- add safe near-misses that must not diagnose.

Metrics are per rule:

```text
historical sensitivity = detected historical mutations / applicable historical mutations
mutation sensitivity   = detected seeded mutations / valid seeded mutations
precision sample       = true findings / reviewed findings
repository noise       = unowned diagnostics on clean baseline
runtime                = wall time, CPU, peak memory, packages analyzed
coverage               = loaded packages, skipped files, unsupported constructs
```

Mutation score is not proof of soundness. It is evidence that the implementation
matches its intended defect class.

## 11. CI and evidence contract

```yaml
schema_version: 1
tool: tinyidpvet
tool_version: "..."
source_commit: "..."
go_version: "..."
catalog_hash: "..."
config_hash: "..."
packages_requested: 0
packages_analyzed: 0
packages_skipped: []
rules:
  - id: STATIC-AUTH-004
    implementation_hash: "..."
    maturity: blocking
    findings: 0
result: PASS | FINDINGS | INCONCLUSIVE | TOOL_ERROR
diagnostics_artifact: "..."
```

`PASS` means every requested package loaded and every blocking rule completed
under the recorded configuration. Analyzer panic, package load failure, unknown
IR, canceled fixed point, or missing dependency summary is not pass. Depending
on declared policy it is `INCONCLUSIVE` or `TOOL_ERROR`.

Suppressions require rule ID, diagnostic fingerprint, reason, owner, creation
date, expiry, and review link. Broad path or rule disablement is not accepted for
blocking security rules without explicit residual-risk approval.

## 12. Phased implementation plan

### Phase 0: baseline and evidence semantics

Inventory the prototype, fixtures, current diagnostics, and existing reports.
Define property/rule IDs, maturity, outcomes, evidence envelope, and suppression
policy. Exit with an approved catalog/evidence contract.

### Phase 1: literature and tool qualification

Complete annotated readings on Go analysis APIs, abstract interpretation,
dataflow fixed points, IFDS/IDE, SSA/callgraphs, taint, typestate, CodeQL, and
Gobra. Reproduce small experiments and decide which tools answer which questions.

### Phase 2: analyzer product architecture

Create the maintained package/command layout, registry, metadata, diagnostics,
configuration, evidence output, test utilities, and migration plan. Do not retain
two analyzer registries or a backwards-compatibility adapter unless explicitly
approved.

### Phase 3: structural rule promotion

Move and harden high-confidence L1/L2 rules. Add direct fixtures for every rule,
stable IDs, near-miss cases, whole-repository baselines, and mutation cases.

### Phase 4: CFG and SSA foundation

Implement reusable CFG/SSA utilities, abstract domains, joins, worklists,
diagnostic traces, and fixture visualization. Prove termination on finite domains
and fail explicitly on unsupported constructs.

### Phase 5: authorization vertical property

Implement `STATIC-AUTH-004`, including `MustChangePassword`. Detect both seeded
historical defects, compare against native regressions, and map results to the
formal/runtime property catalog.

### Phase 6: interprocedural and taint analysis

Add versioned summaries/facts and one secret or continuation taint rule. Compare
native results with CodeQL and evaluate callgraph/context choices.

### Phase 7: protocol lifecycle and atomicity

Model grouped effects for authorization, redemption, refresh, and interaction
consume. Tie diagnostics to failpoint tests and model-checking counterexamples.

### Phase 8: deductive verification experiment

Select one small kernel, specify contracts, run Gobra or the approved alternative,
seed contract-breaking mutations, and make an adopt/no-adopt decision.

### Phase 9: CI, governance, and release integration

Pin tools, add fast and deep profiles, publish machine-readable artifacts,
enforce coverage and crash semantics, assign owners, govern suppressions, and
bind rule/catalog hashes into release evidence.

### Phase 10: long-term evaluation

Track defect escapes, false positives, analysis time, model drift, Go/x-tools
changes, dependency summaries, and new invariant coverage. Periodically rerun
mutations and independent review.

## 13. Testing strategy

Each rule requires:

- unsafe fixture with exact `// want` diagnostic;
- safe fixture;
- near-miss fixture;
- alias/assignment variants;
- helper and interface variants when in scope;
- generics, closures, methods, and error wrapping where relevant;
- historical mutation when available;
- whole-repository golden result;
- analyzer panic/load-failure tests; and
- runtime/coverage measurement.

Framework tests validate lattice laws, monotonic transfer functions, worklist
termination, deterministic diagnostics, summary serialization, fingerprint
stability, and evidence result classification.

## 14. Decision records

### Decision: separate static-analysis ticket

- **Context:** Model checking and static implementation analysis answer different
  questions and have different artifacts and maintenance.
- **Options considered:** merge into `TINYIDP-MODEL-001`; keep only the old
  production ticket; create a dedicated program.
- **Decision:** Use `TINYIDP-STATIC-001` and share invariant IDs.
- **Rationale:** This preserves evidence boundaries while allowing deliberate
  cross-tool traceability.
- **Consequences:** Two tickets require explicit catalog ownership and links.
- **Status:** accepted

### Decision: native Go analysis is primary

- **Context:** Rules need precise Go type identity and repository-specific APIs.
- **Options considered:** native `go/analysis`, CodeQL-only, Semgrep-only, or a
  custom parser.
- **Decision:** Build maintained rules on `go/analysis`; compare selected global
  flows with CodeQL.
- **Rationale:** It fits the module, tooling, test ecosystem, and existing work.
- **Consequences:** We own dataflow infrastructure and must control x/tools
  version changes.
- **Status:** proposed pending qualification

### Decision: lowest sufficient representation

- **Context:** Complex whole-program analyses can be slower and noisier than
  direct structural rules.
- **Decision:** Each rule declares and uses the lowest sufficient maturity level.
- **Consequences:** Shared framework remains modular; rule claims remain narrow.
- **Status:** proposed

### Decision: no silent partial analysis

- **Context:** A green result with skipped packages or analyzer failures is
  misleading for release.
- **Decision:** Record coverage; classify incomplete execution as inconclusive or
  tool error, never pass.
- **Consequences:** CI may fail during Go/tool upgrades and requires prompt owner
  response.
- **Status:** proposed

### Decision: deductive verification is an isolated experiment

- **Context:** Gobra can prove stronger contracts but requires annotations,
  supported language features, solver infrastructure, and specialized review.
- **Decision:** Begin with one small kernel and explicit adopt/no-adopt criteria.
- **Consequences:** No whole-provider proof claim; successful artifacts remain
  valuable local evidence.
- **Status:** proposed

## 15. Risks and alternatives

- **False confidence:** mitigate with per-rule claims and known gaps.
- **False positives:** measure on reviewed corpora; do not hide with broad
  suppressions.
- **False negatives:** maintain mutation/historical corpus and unsupported list.
- **Analysis unsoundness:** specify may/must direction, joins, dispatch, summaries,
  reflection, concurrency, and native-code assumptions.
- **Tool drift:** pin Go/x-tools and test output/evidence compatibility.
- **Framework overengineering:** first vertical rule must precede a generalized
  IFDS engine.
- **Analyzer monoculture:** compare selected flows with CodeQL and implementation
  tests, while avoiding duplicate badge counting.
- **Proof maintenance:** require at least two reviewers able to understand any
  deductive experiment before adoption.

## 16. Open questions

1. Should the maintained package be named `staticanalysis` to avoid confusion
   with the external Staticcheck project?
2. Which current prototype rules are immediately blocking-quality?
3. Is `analysis.Fact` sufficient for the first interprocedural property, or is a
   whole-program driver required?
4. Which callgraph approximation offers acceptable precision for Fosite/store
   interfaces?
5. Should CodeQL be required in release CI or remain scheduled comparison?
6. What exact coverage threshold promotes an advisory rule to blocking?
7. Which pure kernel is the best Gobra experiment?
8. Who owns dependency summaries and suppression expiry review?

## 17. Intern onboarding

1. Read the invariant catalog and the authorization interaction design.
2. Run all existing `auditlint` tests from the top-level module.
3. Read `analysis.Analyzer`, `Pass`, `Fact`, `inspect`, `ctrlflow`, `buildssa`, and
   SSA documentation in the ticket sources.
4. Draw the CFG for the historical blank-login and `MustChangePassword` paths.
5. Implement one safe/unsafe local fixture before designing interprocedural flow.
6. Write the abstract domain and transfer table in prose and tests.
7. Seed both historical mutations and demonstrate expected diagnostics.
8. Run the full repository, classify every finding, and record runtime/coverage.
9. Request review of semantics before promoting the rule to blocking.

## 18. References

### Local evidence

- `ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go`
- `ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main_test.go`
- `ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`
- `internal/fositeadapter/provider.go`
- `internal/fositeadapter/interaction.go`
- `internal/fositeadapter/sqlstore.go`
- `internal/securitytrace/trace.go`
- `pkg/idpstore/types.go`
- `pkg/verifyplan/plan.go`
- `ttmp/2026/07/11/TINYIDP-MODEL-001--serious-model-checking-and-formal-state-assurance-for-tiny-idp/design-doc/02-serious-model-checking-system-architecture-and-implementation-plan.md`

### Preserved research

The ticket `sources/` directory contains official Go analysis, SSA, passes, and
callgraph references; CodeQL Go dataflow and dependency-model documentation;
Semgrep taint, gosec, and Staticcheck documentation; IFDS, typestate,
security-policy, invariant-discovery, and formal OIDC research; and the Gobra
paper and upstream project page.
