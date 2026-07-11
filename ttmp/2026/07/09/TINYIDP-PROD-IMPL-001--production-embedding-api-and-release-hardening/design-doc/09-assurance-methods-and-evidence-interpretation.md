---
Title: Assurance Methods and Evidence Interpretation
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - research
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: |-
        Property and fuzz models
        Generated testing
    - Path: repo://internal/securitytrace/trace.go
      Note: |-
        Runtime verification monitor
        Runtime verification
    - Path: repo://pkg/verifyplan/plan.go
      Note: |-
        Native verification plan execution boundary
        Native verdict boundary
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: |-
        Repository-specific Go AST analyzers
        Go AST analyzers
ExternalSources: []
Summary: How static analysis, examples, property testing, fuzzing, fault injection, race detection, linearizability, runtime monitoring, conformance, and review support different claim classes.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Preventing overclaiming and helping contributors select the cheapest technique capable of falsifying a security hypothesis.
WhenToUse: During test design, security review, analyzer development, evidence collection, or release decisions.
---


# Assurance Methods and Evidence Interpretation

## Start with the claim

Tool selection follows the property. Static analysis is effective for forbidden
dataflow or missing error checks. A state model is effective for legal operation
sequences. Linearizability is effective for concurrent histories. Hosted
conformance is effective for standardized external protocol behavior. None is a
universal security oracle.

| Method | Strongest supported claim | Important limitation |
|---|---|---|
| Example test | named behavior works | sparse input/history coverage |
| `go/analysis` | syntactic/typed pattern absent or present | bounded by rule precision |
| Rapid property | generated sequences satisfy model relation | model may be incomplete |
| Native fuzzing | no counterexample in explored input space/time | weak semantic oracle |
| Metamorphic test | controlled transform preserves/changes relation | relation must be correct |
| Failpoints | named intermediate failures preserve invariant | unenumerated failures remain |
| Race detector | executed code has no observed Go data race | not semantic concurrency proof |
| Porcupine | history admits legal sequential explanation | model and observed output are abstractions |
| Trace monitor | emitted facts satisfy temporal property | missing instrumentation is invisible |
| OIDF conformance | external standardized cases pass | not application policy or operations |
| Human review | design assumptions and omissions challenged | reviewer scope and independence matter |

## Static analysis

The custom analyzers use Go AST, types, and local control structure. They detect
browser continuation reads, ignored security errors, insecure bearer transport,
fail-open parsing, direct security wall clocks, limiter identity risks, and
protocol writes outside lifecycle helpers.

Every analyzer needs positive fixtures, negative fixtures, a repository scan,
and a precision statement. The lifecycle analyzer is not a whole-program proof;
it checks concrete adapter methods for required helper calls. This limitation is
part of the tool's contract, not an embarrassment to omit.

## Generated testing and fuzzing

Property testing generates meaningful actions and compares them with a reference
model. Fuzzing mutates bytes and is most effective when the harness has a strong
oracle: no panic, strict parse result, terminal uniqueness, or monitor verdict.
Shrinking and committed replay histories convert discovery into maintainable
regressions.

The seed, plan hash, source commit, action sequence, observations, trace schema,
and assertion versions should eventually form one evidence envelope. A bare
statement that “fuzzing passed” cannot be reproduced.

## Runtime verification and scripting

Security events describe authoritative native transitions. The native monitor
owns verdicts. Goja scripts compile data-only `VerificationPlan` values and may
select scenarios, but they receive no provider, store, network, clock, or
assertion authority. Otherwise the same script could create, conceal, and judge
its own evidence.

The in-process compiler has source, time, and output limits, not a hard heap
quota. Trusted reviewed scripts are the accepted threat model. Hostile tenant
scripts would require process isolation.

## Evidence discipline

Use this template for every claim:

```text
Claim:
Threat/counterexample:
Authoritative state:
Falsification method:
Observed result:
Coverage boundary:
Residual uncertainty:
```

## Retrospective: why layered assurance was necessary

The assurance program began after two code-review findings: forced
reauthentication could be bypassed on POST, and a typed password result requiring
password change was ignored. Neither defect was a malformed string or a memory
race. Both were failures to carry a security fact through control flow.

Diary Step 17 generalized those findings into a review of continuation authority,
strict parsing, consent decisions, limiter identities, multi-handler persistence,
UserInfo bearer transport, error redirects, and storage failure semantics. Diary
Step 18 then asked which analysis technique could falsify each property.

The conclusion was a portfolio:

```text
source structure       -> Go AST/type analyzers
single examples        -> deterministic regression tests
operation sequences    -> reference models + Rapid
input robustness       -> native fuzzing
execution relations    -> metamorphic testing
intermediate failures  -> named failpoints
memory synchronization -> race detector
concurrent semantics   -> Porcupine histories
temporal execution     -> versioned events + monitor
scenario composition   -> data-only Goja plans + native verdicts
external protocol      -> local and hosted conformance
production claim       -> artifact, operations, review, owner ledger
```

The portfolio is not defense by tool count. Each layer observes a different
representation and has a different oracle.

## Claim taxonomy

Before choosing a method, classify the claim.

### Structural claim

A structural claim concerns source shape or typed calls. Examples:

- `resumeAuthorize` does not read browser POST protocol continuation fields;
- a security transition does not discard `ConsumeInteraction` errors;
- token persistence methods use the lifecycle executor;
- production construction does not silently install a no-op audit sink.

Go analysis is appropriate because the prohibited or required shape is visible
without executing the system.

### Functional claim

A functional claim maps a concrete input/state to an output/state. Examples:

- malformed negative `max_age` does not render credentials;
- UserInfo rejects query bearer transport;
- a disabled client cannot resume an interaction.

Deterministic examples are appropriate and document expected protocol details.

### Relational claim

A relational claim compares executions. Examples:

- adding `ui_locales` preserves code issuance and returned `state`;
- memory and SQLite stores should produce equivalent store-independent
  observations;
- retry after a rolled-back refresh failure succeeds.

Metamorphic or differential tests are appropriate.

### Temporal claim

A temporal claim constrains event order. Examples:

- required authentication precedes approval;
- denial excludes artifact commit;
- one interaction has one terminal outcome.

Reference models and runtime monitors are appropriate.

### Concurrent claim

A concurrent claim constrains overlapping operations. Examples:

- exactly one interaction consume succeeds;
- refresh rotation has one winner and a legal reuse outcome.

Histories and linearizability checking are appropriate.

### Failure-atomicity claim

A failure-atomicity claim constrains intermediate failures. Examples:

- code invalidation does not survive failed replacement-token creation;
- disk-full backup does not replace the last published backup.

Failpoints and state inspection are appropriate.

### Operational claim

An operational claim concerns deployed behavior: readiness, filesystem modes,
audit durability, TLS, shutdown, restore, or exact artifact identity. Runtime
probes, drills, platform tests, and human approval are appropriate.

## The custom Go analysis program

The analyzer binary is a `multichecker` under the production-review ticket. It
uses `golang.org/x/tools/go/analysis`, the same framework as vet-style analyzers.
Each analyzer receives syntax trees, type information, package facts, and a
reporting API.

Fixtures use `analysistest`. Source comments mark expected diagnostics. Positive
fixtures demonstrate detection; negative fixtures protect precision. The suite
also runs over `./pkg/...` and `./internal/...` in CI.

### `tinyidpinternalapi`

This analyzer finds exported public API declarations that mention types from an
`internal/` package. The original embedding review found that public construction
leaked internal interfaces, making external consumption impossible.

The analyzer walks exported type, function, and method signatures with type
information. Its supported claim is that checked declarations do not expose
forbidden package paths. It does not prove the API is usable, which is why the
external standalone-module flow remains necessary.

### `tinyidprand`

This analyzer reports ignored errors from cryptographic randomness. A random
handle or nonce created from a failed reader must not degrade into predictable
or empty output.

Its scope is known `crypto/rand` call shapes. It does not evaluate entropy quality
or operating-system RNG health.

### `tinyidphttpserver`

This analyzer reports zero-value or convenience HTTP server startup patterns
that omit explicit timeouts and limits. The production review used it to replace
plain `ListenAndServe` behavior with configured `http.Server` ownership.

It cannot inspect reverse-proxy or kernel configuration. It protects only source
construction patterns.

### `tinyidpsecuritydefault`

This analyzer reports implicit `NoopSink` and `AllowAllRateLimiter` installation
unless a development-default directive documents the boundary. Production
validation separately rejects missing or non-production-ready controls.

The directive is an explicit suppression and therefore review surface. It should
appear only where production mode is rejected or validated upstream.

### `tinyidpratelimitkey`

The initial rule found `RemoteAddr` including ephemeral port in limiter keys.
Later review found a different issue: unauthenticated claimed client IDs could
create unbounded buckets. The implementation now resolves a normalized address
before authentication and uses verified client identity afterward.

The analyzer detects direct shapes, not arbitrary interprocedural taint. The
CodeQL/IFDS research explains how a future whole-program dataflow rule could
track attacker-controlled values more completely.

### `tinyidpconfiguse`

This analyzer reports public configuration fields that are declared but never
read. Security configuration that appears supported but has no effect is more
dangerous than an absent field because operators may rely on it.

The rule proves source usage, not correct semantics. Tests must show that changing
the field changes behavior.

### `tinyidpauditdelivery`

This analyzer reports discarded audit delivery errors. The production design
uses synchronous fsync audit and exposes delivery failures. Some call sites
cannot roll back after commit, so they return typed ambiguity or increment
health counters rather than pretending delivery cannot fail.

The rule cannot prove event completeness. A successful call with the wrong or
missing event requires behavioral review.

### `tinyidpatomicity`

This analyzer looks for multi-mutation persistence functions without an explicit
transaction boundary. It helped identify broad candidates during the production
review.

Transaction-scoped Fosite methods carry a directive because their transaction
is inherited through context rather than begun in the method. This is a precision
compromise documented in source.

The analyzer cannot infer the complete protocol transaction. The later
`tinyidpprotocollifecycle` rule and failpoint tests cover the concrete adapter
more directly.

### `tinyidpbackupcopy`

This analyzer reports raw filesystem copy patterns used as SQLite backup. The
runtime probe demonstrated why: a copied main file can open successfully while
omitting committed WAL state.

The rule does not validate the online backup implementation or backup manifest.
Those are behavioral and recovery claims.

### `tinyidpbearertransport`

This analyzer guards explicit bearer-transport parsing. Fosite's generic helper
accepts query tokens; tiny-idp's UserInfo contract permits one Authorization
header and rejects query/form or mixed transport.

The rule protects known helper/source shapes. HTTP tests provide the exact status,
challenge, cache, and method semantics.

### `tinyidpsecurityclock`

This analyzer reports direct `time.Now` in named security transition code. The
provider injects a clock so freshness, expiry, lockout, and event timing can be
deterministic.

Not every time read is security-sensitive. The rule scopes functions and permits
documented infrastructure use. It does not establish distributed clock accuracy.

### `tinyidpstrictparse`

This analyzer detects boolean security predicates that accept on parse failure.
Its first implementation incorrectly reported `parseMaxAge` because the
function returns `(int64, bool, error)` and the boolean represented presence,
not acceptance. Diary Step 22 records the correction: the rule now limits itself
to single-boolean predicate results.

This episode is an important analyzer lesson. A noisy security rule trains
maintainers to ignore diagnostics. Narrow explainable precision is preferable to
an ambitious but misleading approximation.

### `tinyidpinteractioncontinuation`

This analyzer walks `resumeAuthorize` and reports reads of browser POST fields
that belong to the OAuth continuation. It permits only the native interaction
inputs required by the UI.

Its precision boundary is intentionally local. A helper that reads a forbidden
field interprocedurally could evade it. Tests and review of the rendered form
provide complementary evidence.

### `tinyidpprotocollifecycle`

This analyzer checks concrete SQL Fosite persistence methods for the required
authorization or token lifecycle executor. It encodes the codebase-specific
rule discovered after inspecting Fosite handler ordering.

It is not a general transactional analysis. New storage implementations or
renamed methods require fixture and rule updates.

### `tinyidpignoredsecurityerror`

This analyzer reports ignored errors from high-value transitions such as
`ConsumeInteraction`, `CreateBrowserSession`, `RecordConsent`,
`ActiveSigningKey`, and transaction `Commit`.

The call list is an explicit policy. A new security-returning method is not
covered until added. Code review and error inventories remain necessary.

## Analyzer engineering workflow

For every new rule:

1. Start with a real defect and minimal source pattern.
2. State the supported claim in one sentence.
3. State obvious false positives and false negatives.
4. Create one positive fixture per intended shape.
5. Create negative fixtures for similar legal code.
6. Run `analysistest` before repository scan.
7. Inspect every repository diagnostic manually.
8. Narrow the rule rather than suppressing broad false positives.
9. Document directives as reviewable exceptions.
10. Keep a behavioral test for the original defect.

## Research influence: IFDS and dataflow

The IFDS framework expresses interprocedural distributive dataflow problems as
graph reachability over exploded supergraphs. CodeQL and Semgrep provide
different practical abstractions for taint and patterns.

tiny-idp's current analyzer suite deliberately stays mostly intraprocedural. The
research matters because it identifies where local checks stop being adequate:
limiter identity taint, unverified redirect provenance, and secrets reaching
audit/log sinks are candidates for interprocedural dataflow.

The design document proposes advancing only when a concrete defect justifies the
complexity and fixture burden.

## Deterministic regression tests

Example tests remain the most readable specification for named behavior. The
hardening suite contains forced login, `max_age`, malformed parsing,
`prompt=none`, consent denial, request mutation, concurrent tabs, replay,
expiry, disabled client/user, missing key, CSRF, UserInfo, and storage failures.

A security regression test should assert the protected artifact, not only the
surface response. Useful negative oracles include:

- no authorization code in redirect;
- no token rows in SQL;
- interaction remains pending or has exactly one terminal outcome;
- no credential form for invalid request;
- no forbidden hidden fields;
- no committed security event;
- stable typed error reason.

## Property-based state-machine testing

Rapid generates a sequence length and operations. The reference state predicts
duplicate create, absent get/consume, accepted terminal, expired terminal, and
already-consumed results. The real memory store executes the corresponding
operation.

Generation explores combinations that table tests may omit. Shrinking minimizes
the failing sequence. Labels record drawn values. Reproducibility requires the
seed and final shrunk actions.

The model is intentionally smaller than HTTP. This makes disagreements
diagnosable. It also means provider reconstruction, CSRF, and Fosite behavior are
outside the claim.

## Native fuzzing

Go fuzz targets preserve seed corpora and use coverage feedback. tiny-idp has
targets for issuer parsing, redirect validation, Argon2 hash parsing, bounded
`max_age`, event sequences, and interaction model actions.

Parser targets have strong local oracles: accepted values satisfy strict
normalization and invalid inputs do not panic or escape bounds. State fuzzers
need an invariant oracle such as terminal uniqueness or monitor totality.

The exact-candidate evidence records duration, executions, and interesting
inputs. A bounded campaign supports “no counterexample found in this run,” not
“parser is correct for all strings.”

Two fuzz commands for the same package were accidentally launched concurrently
and stalled. They were rerun sequentially. The diary preserves this because
orchestration is part of reproducibility.

## Coverage-guided property testing

The saved coverage-guided property-testing paper motivates combining semantic
generators and coverage feedback. Ordinary fuzzing reaches code paths but may
generate semantically meaningless protocol sequences. Ordinary property testing
generates meaningful actions but may repeatedly cover the same states.

The current project has not implemented a combined engine. The design inference
is to preserve typed actions and observations now so coverage guidance can be
added later without redefining security semantics.

## Metamorphic testing

Metamorphic testing is useful when outputs contain randomness or time. The
oracle compares a relation, not exact bytes.

The first strict-provider relation varies `ui_locales` and requires unchanged
successful issuance and returned `state`. Future relations should cover query
ordering, irrelevant extension parameters, storage backends, and equivalent
scope encodings where the standard permits equivalence.

Security-sensitive mutation needs an inequality oracle: duplicate redirect,
changed PKCE challenge, or changed interaction handle must not preserve success.

The cybersecurity metamorphic-testing source influenced this separation between
invariance transforms and adversarial transforms.

## Protocol-state fuzzing

The TLS protocol-state fuzzing and stateful greybox papers show why message
sequences reveal logical vulnerabilities that packet mutation misses. A stateful
fuzzer maintains or learns protocol state and chooses messages that reach deeper
transitions.

tiny-idp's typed verification actions are the beginning of such an alphabet:
session login, authorize begin, interaction submit, and time advance. Token,
refresh, UserInfo, fault, and administrative mutation actions remain future work.

Native Go owns decoding and execution so generated scripts cannot bypass the
protocol harness.

## Fault injection

Named failpoints test failures around actual durable mutations. Authorization
has seven lifecycle points, code exchange eight, and refresh rotation ten. Backup
tests cover cancellation and disk-full publication.

Every failpoint needs a before-state, injected error, response observation,
durable after-state, event after-state, and retry expectation where applicable.

Failpoint coverage is enumerable and reviewable. It does not simulate arbitrary
process, kernel, power, or storage-controller failure. Crash testing and platform
faults are additional layers.

## Race detection and schedule exploration

`go test -race` instruments executed memory access. Shuffled and repeated tests
vary order. Barriers create overlap around targeted operations. The CHESS paper
provides the research context for systematic schedule exploration and preemption
bounding.

tiny-idp does not claim exhaustive schedule coverage. The race run, controlled
histories, and Porcupine models support complementary bounded claims.

## Linearizability checking

Porcupine consumes operation call/return times, inputs, outputs, and an abstract
model. It searches for a legal sequential order preserving real-time precedence.

Interaction consume and refresh rotation use different models. The refresh
test's failed final-state assumption demonstrated that model correctness is as
important as checker correctness. Rotation was linearizable, while later reuse
revoked the family.

## Runtime verification

The new introductory runtime-verification source organizes the field around
instrumentation, specification, monitor synthesis/execution, and verdict. The
project implements a deliberately small manual automaton.

The event schema is versioned and excludes raw handles, codes, credentials, and
tokens. The interaction ID is derived for correlation. The monitor partitions
state, records obligations, and reports finite bad prefixes.

Instrumentation completeness is the central limitation. A monitor cannot reject
an event that native code failed to emit. Provider tests, failpoint feeds, and
source review check important emission sites.

## Monitoring-oriented programming and dynamic invariants

Monitoring-oriented programming advocates treating monitors as explicit program
components. Daikon-style dynamic invariant detection infers likely properties
from executions. The project separates these roles:

- reviewed native invariants may fail tests or readiness;
- discovered correlations may suggest candidates;
- no mined invariant becomes authorization policy automatically.

This prevents training data or incomplete traces from silently defining
security semantics.

## eBPF and host instrumentation

The research/design phase considered eBPF for syscall, network, file, and
scheduler evidence. It was not used for protocol verdicts because kernel traces
do not contain validated OAuth request meaning or consent obligations.

Application instrumentation proved more direct: HTTP timing, Go runtime metrics,
SQLite pool stats, password-work counters, audit counts, and security events.
eBPF remains useful for corroborating filesystem and network assumptions in a
target deployment.

## Runtime load probe

The runtime probe provisions a production-mode SQLite provider and executes full
login, code, token, UserInfo, and refresh flows under bounded concurrency. It
emits NDJSON HTTP events, runtime metrics, SQL pool snapshots, password-work
statistics, audit counts, and optional CPU/heap profiles.

The analyzer computes route status distributions and latency percentiles. The
recorded candidate performed 5,125 HTTP operations with zero HTTP errors and 25
bounded password operations. This is performance/operations evidence, not a
proof of protocol safety.

## Data-only verification plans

`pkg/verifyplan` defines a versioned plan, suites, scenarios, steps, assertion
references, limits, source hash, driver, observations, and results. It has no
Goja or provider dependency.

`internal/gojaverify` creates a fresh runtime, rejects every ambient module,
registers only `tinyidp/verify`, enforces source/time/output limits, and binds
the result to source SHA-256. Tests reject `fs` and interrupt an infinite loop.

The strict driver decodes known actions with unknown fields rejected. Native
assertion functions own verdicts. A plan hash provides provenance, not author
authentication.

The compiler is in-process and has no hard heap quota. Hostile multi-tenant
scripts require a separate process and authenticated plan handoff.

## Conformance

The local conformance script runs the full suite, selected strict Fosite cases,
durable store/key tests, AST analysis, and an external public-API consumer flow.
It proves the repository-selected profile in the local environment.

Hosted OIDF drives standardized external cases against a deployed issuer. It
requires exact artifact binding, plan metadata, suite authority, and preserved
logs. Older hosted result directories cannot be reused for a new binary hash.

Conformance does not test durable audit, password admission, backup, custom
consent policy, or every local temporal invariant.

## Vulnerability scanning, SBOM, and provenance

Govulncheck uses the module graph and call analysis to report reachable known
vulnerabilities. The exact candidate had zero called vulnerabilities while
dependencies contained additional known entries not reached by current code.

SBOM and module graph describe composition. Checksums identify bytes. Signatures
and provenance bind an artifact to build identity and process. None prove runtime
correctness, but each answers a release question tests cannot.

## Human review

Independent review can challenge threat model, missing properties, model
abstraction, analyzer blind spots, operational assumptions, and residual risk.
Release-owner approval accepts organizational risk and deployment scope.

The software cannot self-assign either authority. The ledger keeps those rows
open despite extensive local evidence.

## Paper-to-tool provenance matrix

| Research/source | Concept used | Concrete project result | Not claimed |
|---|---|---|---|
| Formal OAuth analysis | authorization/session integrity | canonical interaction and review properties | full formal verification |
| Formal OIDC analysis | authentication/session binding | freshness, nonce/auth-time reasoning | proof of all OIDC profiles |
| Security automata | finite bad prefixes | interaction monitor | production enforcement completeness |
| Typestate | state-dependent operations | pending/terminal consume contract | compile-time durable typestate |
| IFDS | interprocedural dataflow | future taint boundary; local AST precision | whole-program taint today |
| Model-based security testing | abstract transition systems | Rapid store model | model completeness |
| Coverage-guided PBT | semantic generation + coverage | typed action design direction | implemented hybrid engine |
| Stateful greybox fuzzing | protocol sequence exploration | action-sequence fuzz target | deep coverage guidance |
| Protocol-state TLS fuzzing | logical state vulnerabilities | separate action alphabet | TLS-state model reuse |
| Metamorphic cybersecurity | relational oracle | `ui_locales` relation | exhaustive relations |
| CHESS | controlled schedules | barriers/repetition/history capture | systematic schedule enumeration |
| Linearizability | legal concurrent histories | Porcupine consume/refresh checks | full token-family model |
| Lineage fault injection | dependency-oriented faults | pre/post mutation matrices | kernel/storage exhaustive faults |
| Runtime verification | event/specification/verdict separation | native parametric monitor | observability completeness |
| Daikon | candidate invariant mining | research direction only | mined policy enforcement |
| Monitoring-oriented programming | first-class monitors | separate securitytrace package | universal runtime monitor framework |
| Go analysis docs | typed AST tooling | auditlint multichecker | semantic proof |
| Go fuzz docs | corpus and coverage fuzzing | six exact-candidate campaigns | exhaustive input proof |

## Evidence review questions

For a green result, ask:

1. What exact property was the oracle checking?
2. What representation did the tool observe?
3. Which code paths or schedules executed?
4. Which model or rule encoded legality?
5. What could be absent from the observation?
6. Is the result reproducible from commit, seed, plan, and command?
7. Did a failure occur before or after durable commit?
8. Does this claim belong to local code, deployment, or human authority?
9. What counterexample would falsify the claim?
10. Which complementary method covers the largest blind spot?

## Common evidence errors

### Counting tests

Test count says little about property diversity or oracle quality. Map tests to
invariants and failure modes.

### Treating no panic as security

No panic is valuable robustness evidence but usually says nothing about
authorization correctness.

### Treating coverage as correctness

Executing a branch does not prove the assertion checked its security effect.

### Treating lint as proof

Static rules encode selected source patterns and have explicit blind spots.

### Treating monitor silence as success

Missing events and disabled sinks can produce silent traces. Delivery and
instrumentation completeness need separate evidence.

### Treating conformance as certification

A passing run is tied to suite, plan, configuration, deployment, and artifact.
It does not automatically grant formal certification or release approval.

### Treating reproducibility as authenticity

A deterministic hash identifies bytes. Signatures and provenance add origin and
build claims. Review and owner approval add authority.

## Decision records

### AM-1: choose tools by claim

- **Decision:** maintain complementary assurance layers.
- **Reason:** structural, temporal, concurrent, failure, and operational
  properties require different observations and oracles.
- **Consequence:** release evidence is a matrix, not one score.

### AM-2: narrow blocking analyzers

- **Decision:** prefer precise local rules with fixtures and documented gaps.
- **Reason:** false positives degrade trust in CI diagnostics.
- **Consequence:** behavioral and future dataflow tools cover broader claims.

### AM-3: native verdict authority

- **Decision:** Go owns monitor and scenario assertions.
- **Reason:** tested scripts must not define their own success.
- **Consequence:** JavaScript remains a plan-authoring surface.

### AM-4: record failures and corrected models

- **Decision:** diary preserves unsuccessful tests and mistaken assumptions.
- **Reason:** corrected abstractions are part of the research result.
- **Consequence:** textbook provenance includes failure, not only final design.

## Extended exercises

1. Map every auditlint analyzer to its originating defect or risk.
2. Propose an interprocedural limiter-taint analysis and state sources/sinks.
3. Write positive and negative fixtures for a new secret-logging analyzer.
4. Design a Rapid model for consent expiry and revocation.
5. Design a fuzz oracle for duplicated authorization parameters.
6. Create one valid and one invalid metamorphic transform for scopes.
7. Add a failpoint after consent persistence and predict ambiguity.
8. Explain why the race detector cannot prove token rotation uniqueness.
9. Extend the Porcupine refresh model with reuse.
10. Identify one security event whose absence current monitor tests might miss.
11. Design an instrumentation-completeness test.
12. Define an evidence envelope schema for plan/seed/commit/observations.
13. Explain when eBPF adds value to a target deployment.
14. Compare local conformance with hosted OIDF claim scope.
15. Review a govulncheck result containing unreachable vulnerabilities.
16. Explain why SBOM, signature, and provenance are separate artifacts.
17. Write a release statement that remains honest after all local gates pass.
18. Identify one paper concept intentionally not implemented.
19. Find a diary failure that changed a test oracle.
20. State the cheapest falsification method for a proposed invariant.

## Chapter review checklist

- Can the reader classify a claim before selecting a tool?
- Can the reader explain all current custom analyzers and their gaps?
- Can the reader distinguish examples, properties, fuzz, and metamorphic tests?
- Can the reader build a complete failpoint oracle?
- Can the reader distinguish race, scheduling, and linearizability evidence?
- Can the reader explain event instrumentation and monitor limitations?
- Can the reader enumerate JavaScript's reachable capabilities?
- Can the reader interpret runtime load evidence without security overclaiming?
- Can the reader distinguish conformance, vulnerability, SBOM, signature, and
  human approval?
- Can the reader trace every implemented assurance layer to saved research and
  current code?

## Exercises

1. Choose a technique for “browser fields cannot replace canonical state” and
   explain why fuzzing alone is insufficient.
2. State the false-negative boundary of one auditlint analyzer.
3. Explain what a monitor cannot detect when an event is never emitted.
4. Distinguish a passing race run from a passing Porcupine history.
5. Write an evidence statement for a 30-second fuzz campaign without claiming
   exhaustive correctness.

## Research packet

Read `paper-ifds-dataflow-analysis`, `paper-model-based-security-testing`,
`paper-stateful-greybox-fuzzing`, `paper-chess-systematic-concurrency-testing`,
`paper-lineage-driven-fault-injection`, and
`paper-runtime-verification-brief-account` under `sources/`.
