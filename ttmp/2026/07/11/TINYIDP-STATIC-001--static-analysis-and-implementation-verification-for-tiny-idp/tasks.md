# Tasks

## Phase 0 — Baseline and evidence semantics

- [x] Create TINYIDP-STATIC-001 with design guide, diary, source packet, task ledger, and changelog. <!-- t:lmtd -->
- [x] Inventory the existing auditlint registry, implementation size, and direct analysistest entry points. <!-- t:bp21 -->
- [x] Identify the existing static-analysis assurance design and model-checking integration points. <!-- t:pi5u -->
- [ ] Assign stable property, rule, diagnostic, mutation, and evidence identifiers. <!-- t:ihra -->
- [ ] Classify every existing analyzer by AST, types, CFG, SSA, interprocedural, taint, or heuristic technique. <!-- t:jyft -->
- [ ] Record scope, confidence, severity, known false positives, and known false negatives for every existing rule. <!-- t:3k8m -->
- [ ] Define PASS, FINDINGS, INCONCLUSIVE, TOOL_ERROR, and partial-package semantics. <!-- t:03t7 -->
- [ ] Define the versioned analysis evidence envelope and diagnostic fingerprint contract. <!-- t:b9uj -->
- [ ] Define suppression ownership, reason, expiry, and residual-risk requirements. <!-- t:bv3u -->
- [ ] Obtain maintainer approval of the invariant/rule catalog and evidence contract. <!-- t:a9yr -->

## Phase 1 — Literature and tool qualification

- [x] Preserve primary Go analysis, SSA, callgraph, CodeQL, Semgrep, gosec, Staticcheck, IFDS, typestate, and Gobra sources. <!-- t:ba2l -->
- [ ] Write an annotated bibliography distinguishing source claims from tiny-idp design inferences. <!-- t:ov03 -->
- [ ] Complete a glossary for soundness, completeness, precision, recall, may/must analysis, lattice, join, fixed point, context sensitivity, and path sensitivity. <!-- t:73xm -->
- [ ] Reproduce an inspect/types analysistest tutorial with exact tool and Go versions. <!-- t:oqq4 -->
- [ ] Reproduce a ctrlflow/buildssa tutorial and preserve CFG/SSA output for a branching fixture. <!-- t:kg23 -->
- [ ] Implement a small finite forward dataflow worklist and test lattice/termination properties. <!-- t:twsg -->
- [ ] Implement the same small source-to-sink question in native Go analysis and CodeQL. <!-- t:x0uz -->
- [ ] Evaluate Semgrep taint mode for fast local variants without overstating interprocedural coverage. <!-- t:rtma -->
- [ ] Run gosec, vet, Staticcheck, and CodeQL baseline scans and document overlap with custom rules. <!-- t:n4pv -->
- [ ] Complete a Gobra tutorial and record supported Go subset, annotations, solver/runtime, and reproducibility. <!-- t:wkik -->
- [ ] Approve the initial native analysis and comparison-tool strategy. <!-- t:tutl -->

## Phase 2 — Maintained analyzer product architecture

- [ ] Decide the maintained package and command names without creating a nested Go module. <!-- t:v3x9 -->
- [ ] Create registry, rule metadata, diagnostic, configuration, and evidence packages. <!-- t:8pdj -->
- [ ] Define stable YAML/JSON schemas for catalog, diagnostics, evidence, and suppressions. <!-- t:ztlm -->
- [ ] Implement deterministic rule registration and duplicate-ID rejection. <!-- t:p8sj -->
- [ ] Implement package coverage and skipped-package accounting. <!-- t:78ub -->
- [ ] Implement fail-closed analyzer panic, package-load, cancellation, and unsupported-IR handling. <!-- t:mgq0 -->
- [ ] Implement text, JSON, and SARIF-compatible output or document why SARIF is deferred. <!-- t:rdpn -->
- [ ] Create shared analysistest helpers and machine-readable fixture metadata. <!-- t:vqqb -->
- [ ] Write a deliberate migration plan from the dated ticket prototype; do not add a compatibility adapter without approval. <!-- t:0olz -->
- [ ] Add developer commands and documentation for local, CI, and focused-rule execution. <!-- t:byz1 -->

## Phase 3 — Structural analyzer promotion

- [ ] Promote the injected security-clock rule with positive, negative, and near-miss fixtures. <!-- t:inps -->
- [ ] Promote ignored randomness and security-error rules with typed callee identity. <!-- t:biey -->
- [ ] Promote strict parsing with explicit optional-absence handling. <!-- t:rb4f -->
- [ ] Promote explicit bearer-transport enforcement. <!-- t:dtnk -->
- [ ] Promote server-owned continuation structural checks. <!-- t:im8j -->
- [ ] Promote protocol-lifecycle and atomicity heuristics as advisory rules. <!-- t:qwe7 -->
- [ ] Classify security-default, rate-limit identity, audit delivery, HTTP server, backup, config-use, and internal-API rules. <!-- t:bwk6 -->
- [ ] Add a direct analysistest entry point for every registered rule. <!-- t:528e -->
- [ ] Add mutation cases and whole-repository golden results for each promoted rule. <!-- t:hh9l -->
- [ ] Measure runtime, memory, findings, and repository noise before blocking promotion. <!-- t:hclg -->

## Phase 4 — CFG and SSA foundation

- [ ] Define reusable finite fact sets, lattices, joins, transfer functions, and worklist APIs. <!-- t:av36 -->
- [ ] Test idempotent/commutative/associative joins and monotone transfers. <!-- t:92bl -->
- [ ] Add CFG block, dominance, post-dominance, and reachability utilities. <!-- t:rr5y -->
- [ ] Add SSA value provenance, phi, call, return, closure, interface, and alias handling policy. <!-- t:ikc5 -->
- [ ] Render diagnostic flow traces from source or obligation to sink. <!-- t:yu89 -->
- [ ] Define explicit behavior for reflection, unsafe, cgo, generics, goroutines, and unknown calls. <!-- t:xxod -->
- [ ] Add pathological-loop and large-function termination/performance fixtures. <!-- t:sc9a -->
- [ ] Add visualization/debug output usable during analyzer review. <!-- t:uhh6 -->
- [ ] Review the framework before implementing security rules on top of it. <!-- t:ksqt -->

## Phase 5 — Authorization vertical property

- [ ] Define the exact abstract domain for required login, authentication time, password-change requirement, consent, expiry, and terminal state. <!-- t:8bmq -->
- [ ] Map sources, dischargers, blockers, sinks, and dependency summaries to current Go symbols. <!-- t:7481 -->
- [ ] Implement the intra-provider CFG/SSA version of STATIC-AUTH-004. <!-- t:74on -->
- [ ] Seed and detect the historical forced-reauthentication blank/crafted POST mutation. <!-- t:ss64 -->
- [ ] Seed and detect ignored MustChangePassword at authorization issuance. <!-- t:ttqt -->
- [ ] Add safe, denial, expiry, old-session, new-login, and password-change fixtures. <!-- t:nlen -->
- [ ] Add helper, interface, closure, alias, and branch-merge variants within declared scope. <!-- t:m9i7 -->
- [ ] Compare diagnostics with formal property, runtime event rule, and native regressions. <!-- t:mius -->
- [ ] Run the complete repository and classify every finding and blind spot. <!-- t:cye1 -->
- [ ] Obtain security review before advisory or blocking deployment. <!-- t:bz7s -->

## Phase 6 — Interprocedural facts and taint

- [ ] Define versioned function-effect, obligation, trust, and propagation facts. <!-- t:iapi -->
- [ ] Define callgraph algorithm, context sensitivity, and interface dispatch assumptions. <!-- t:dugo -->
- [ ] Add explicit Fosite, store, HTTP, logging, audit, and Goja dependency summaries. <!-- t:x9g4 -->
- [ ] Implement cross-package fact export/import and cache/version invalidation tests. <!-- t:gbo9 -->
- [ ] Implement browser-controlled continuation taint to canonical-state and decision sinks. <!-- t:nxpx -->
- [ ] Implement secret taint from passwords/tokens/cookies/client secrets to logs, errors, metrics, audit, scripts, redirects, and persistence. <!-- t:bq3p -->
- [ ] Define sink-specific sanitizers and prove hashing is not a universal sanitizer. <!-- t:tlsr -->
- [ ] Compare one native global flow with CodeQL and document divergent paths. <!-- t:kifc -->
- [ ] Measure precision/runtime effects of context and callgraph choices. <!-- t:iolv -->
- [ ] Review whether IFDS/IDE infrastructure is justified by the implemented fact domains. <!-- t:yliv -->

## Phase 7 — Protocol lifecycle and model integration

- [ ] Define grouped effects for authorization response, code redemption, refresh rotation, key activation, and interaction consume. <!-- t:v7re -->
- [ ] Implement transaction/effect summaries and terminal-success sink checks. <!-- t:jdk9 -->
- [ ] Tie lifecycle diagnostics to SQLite failpoint boundaries and tests. <!-- t:byi6 -->
- [ ] Import stable invariant/action IDs from TINYIDP-MODEL-001 without duplicating authority. <!-- t:eg1w -->
- [ ] Map model counterexamples into static mutation cases when the defect has a source-level signature. <!-- t:1urq -->
- [ ] Classify every invariant as statically enforceable, approximable, model-checkable, runtime-monitorable, testable, or operational. <!-- t:boxd -->
- [ ] Add a coverage report showing which evidence layers support each high-risk invariant. <!-- t:gk1s -->
- [ ] Review mismatches among static, model, runtime, and implementation observations. <!-- t:h086 -->

## Phase 8 — Deductive verification experiment

- [ ] Select a small pure interaction, refresh-family, or transaction-planning kernel. <!-- t:sos4 -->
- [ ] Write precise preconditions, postconditions, invariants, frame/permission assumptions, and ghost state. <!-- t:gm58 -->
- [ ] Pin Gobra/Viper/solver versions and create reproducible invocation scripts under the ticket first. <!-- t:9kdc -->
- [ ] Verify the baseline kernel and preserve proof output and runtime statistics. <!-- t:vpov -->
- [ ] Seed contract-breaking, race/permission, and state-transition mutations. <!-- t:gxag -->
- [ ] Document unsupported Go features and required code reshaping. <!-- t:r7cn -->
- [ ] Compare proof strength and maintenance cost with tests and model checking. <!-- t:g0ga -->
- [ ] Obtain a second trained reviewer and make an adopt/no-adopt decision. <!-- t:8xcr -->

## Phase 9 — CI, governance, and release evidence

- [ ] Add fast structural and focused-flow analysis on relevant changes. <!-- t:usar -->
- [ ] Add scheduled/release whole-repository and comparison-tool profiles. <!-- t:ilyf -->
- [ ] Pin Go, x/tools, external tool, query/model, and schema versions. <!-- t:86u4 -->
- [ ] Fail on blocking findings and tool errors; expose inconclusive coverage explicitly. <!-- t:4qlz -->
- [ ] Archive diagnostics, evidence envelopes, coverage, runtime, and mutation results. <!-- t:mprd -->
- [ ] Add analyzer/catalog ownership and required security review for weakened rules. <!-- t:3rkw -->
- [ ] Enforce suppression owner, reason, expiry, and review metadata. <!-- t:gfpf -->
- [ ] Bind source, catalog, analyzer, configuration, and dependency-summary hashes into release evidence. <!-- t:tcuk -->
- [ ] Run independent review and approve the static-analysis release gate. <!-- t:gxd1 -->

## Phase 10 — Long-term evaluation

- [ ] Track escaped defects, false positives, suppressions, runtime, and package coverage by rule/version. <!-- t:a4ad -->
- [ ] Periodically rerun historical and seeded mutation corpora. <!-- t:6x5w -->
- [ ] Audit analyzer behavior after Go, x/tools, Fosite, SQLite, or Goja upgrades. <!-- t:63kp -->
- [ ] Remove or redesign noisy rules rather than normalizing broad suppressions. <!-- t:g40z -->
- [ ] Expand invariant coverage only when a precise claim and owner exist. <!-- t:zuvm -->
- [ ] Reassess deductive verification and whole-program techniques after the first vertical property is stable. <!-- t:mqdt -->
