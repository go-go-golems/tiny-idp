# Tasks

## Phase 0 — Program baseline and evidence contract

- [x] Create the dedicated TINYIDP-MODEL-001 ticket, design guide, diary, source packet, task ledger, and changelog. <!-- t:15tm -->
- [x] Inventory current Rapid models, Porcupine histories, trace monitors, verification plans, failpoints, clocks, fuzzers, and replay tests. <!-- t:exax -->
- [x] Confirm which formal-specification file formats and general model checkers are absent from the repository. <!-- t:82k0 -->
- [x] Record the current maturity boundary between model-based testing, linearizability checking, runtime verification, and exhaustive model checking. <!-- t:5bdv -->
- [ ] Define stable model, action, invariant, observation, assumption, and counterexample identifiers. <!-- t:mkbe -->
- [ ] Define a versioned model-checking evidence envelope including checker version, constants, state count, depth, duration, result, and trace digest. <!-- t:2ryg -->
- [ ] Define rules for PASS, FAIL, INCONCLUSIVE, TIMEOUT, STATE-SPACE-EXHAUSTED, and TOOL-ERROR outcomes. <!-- t:1yix -->
- [ ] Obtain maintainer acceptance of the program charter and evidence contract. <!-- t:234s -->

## Phase 1 — Literature, theory, and tool qualification

- [x] Assemble primary TLA+/PlusCal/TLC, Apalache, Alloy, Porcupine, runtime-verification, model-based testing, concurrency, fault-injection, and OAuth/OIDC sources. <!-- t:bqbr -->
- [ ] Write an annotated bibliography that states the exact concept imported from each source and what is only a tiny-idp inference. <!-- t:oar4 -->
- [ ] Complete a glossary for state, behavior, action, invariant, safety, liveness, fairness, refinement, stuttering, trace, linearizability, and bounded scope. <!-- t:sxua -->
- [ ] Complete a TLC tutorial model for a one-time capability and preserve commands, configuration, state count, and counterexample output. <!-- t:p7s6 -->
- [ ] Complete an Apalache tutorial model and compare its bounded symbolic result with TLC's explicit-state result. <!-- t:qzu6 -->
- [ ] Complete an Alloy 6 tutorial exercise covering relations, scopes, assertions, counterexamples, and temporal instances. <!-- t:f3cx -->
- [ ] Evaluate Quint as an authoring or simulation frontend without changing the initial TLA+/TLC decision prematurely. <!-- t:0i2j -->
- [ ] Write the checker comparison matrix covering semantics, bounds, counterexamples, CI packaging, maintenance, and team learning cost. <!-- t:3j7q -->
- [ ] Review formal OAuth/OIDC attacker-model assumptions and identify the subset inherited rather than re-modeled locally. <!-- t:f760 -->
- [ ] Hold a theory review and approve the first model language/toolchain. <!-- t:agyy -->

## Phase 2 — Domain abstraction and assumption ledger

- [ ] Create a field-by-field abstraction table for InteractionRecord and mark modeled, derived, abstracted, and omitted fields. <!-- t:yi4l -->
- [ ] Create a Fosite assumption ledger for authorization validation, code semantics, token response orchestration, and reuse behavior. <!-- t:dvt3 -->
- [ ] Create a SQLite assumption ledger for atomic commit, rollback, isolation, crash persistence, and single-writer topology. <!-- t:i6sb -->
- [ ] Define the bounded logical clock and expiry semantics used by every model. <!-- t:f7dd -->
- [ ] Define browser, relying-party, user, administrator, crash, and scheduler actors. <!-- t:0j9a -->
- [ ] Define mutable client/user/key generation changes and their allowed observation points. <!-- t:ct2v -->
- [ ] Map every model action to current Go methods, tests, failpoints, and security events. <!-- t:odj7 -->
- [ ] Review abstraction soundness and explicitly list implementation behaviors outside the model. <!-- t:znkp -->

## Phase 3 — Authorization interaction model

- [ ] Write the authorization interaction TLA+/PlusCal specification with two tabs, bounded time, required actions, mutable generations, and terminal outcomes. <!-- t:ak3o -->
- [ ] Encode validation-before-credentials, server-owned canonical request, required login, fresh login, consent, expiry, denial, and artifact-order invariants. <!-- t:m452 -->
- [ ] Encode independent interaction partitions so one tab cannot satisfy another tab's obligations. <!-- t:cv1b -->
- [ ] Reproduce the historical forced-reauthentication POST bypass by intentionally removing required-action persistence. <!-- t:j7fn -->
- [ ] Reproduce replay and concurrent approve/deny counterexamples with a deliberately non-atomic terminal transition. <!-- t:73or -->
- [ ] Restore the intended design and check all safety invariants under the approved finite constants. <!-- t:xmfi -->
- [ ] Evaluate fairness only for explicitly selected liveness questions; keep safety checking independent of unjustified fairness. <!-- t:bizy -->
- [ ] Export at least one TLC counterexample into a normalized action trace. <!-- t:0758 -->
- [ ] Replay the trace against the native strict-provider VerificationPlan driver and commit it as a Go regression. <!-- t:s8v2 -->
- [ ] Review model/implementation divergences and update either the abstraction or product design explicitly. <!-- t:rdfs -->

## Phase 4 — Authorization-code redemption and crash model

- [ ] Model code state, token rows, transaction state, response state, crash, recovery, and client retry. <!-- t:sm0t -->
- [ ] Align every pre-commit model action with an existing token persistence failpoint. <!-- t:odo4 -->
- [ ] Check that rollback preserves the active code and removes partial replacement tokens. <!-- t:46zb -->
- [ ] Check that committed replacement tokens imply code invalidation. <!-- t:dzj8 -->
- [ ] Check at-most-one committed redemption under concurrent requests. <!-- t:jiks -->
- [ ] Explore crash after commit and before response delivery as an explicit ambiguous outcome. <!-- t:ka63 -->
- [ ] Decide product/client recovery semantics for the post-commit lost-response case. <!-- t:m7j2 -->
- [ ] Add missing implementation failpoints or tests revealed by the model. <!-- t:xw8a -->
- [ ] Convert counterexamples into deterministic SQL/provider regression tests. <!-- t:5slt -->

## Phase 5 — Refresh-token family and reuse model

- [ ] Model family status, generations, concurrent presentations, validation, rotation, reuse detection, revocation, delivery, loss, and retry. <!-- t:oasl -->
- [ ] Check at-most-one successful rotation per generation. <!-- t:74pq -->
- [ ] Check that revoked families have no active refresh or access token. <!-- t:y3tb -->
- [ ] Check that an old generation never becomes active again. <!-- t:fxds -->
- [ ] Model the observed legitimate concurrent refresh sequence that revokes the winning replacement family. <!-- t:5bhk -->
- [ ] Compare TLA+ traces with current Porcupine histories and database end-state assertions. <!-- t:grbg -->
- [ ] Extend the executable sequential model or add a separate family-state oracle where the current boolean model is insufficient. <!-- t:95bm -->
- [ ] Document the client singleflight requirement and operational indicators for concurrent refresh reuse. <!-- t:38cs -->
- [ ] Add a token-family security-event vocabulary and native monitor if justified by model results. <!-- t:263p -->

## Phase 6 — Relational and symbolic expansion

- [ ] Reassess whether Alloy materially improves canonical-request, session, client-generation, and token-lineage analysis. <!-- t:j1h4 -->
- [ ] If justified, write one bounded Alloy model for lineage/binding relationships and link its assertions to TLA+ actions. <!-- t:u4au -->
- [ ] Run Apalache on stable supported TLA+ specifications and record bounded execution lengths and solver results. <!-- t:icvo -->
- [ ] Compare TLC and Apalache counterexamples for semantic consistency without treating duplicate success as independent proof. <!-- t:url3 -->
- [ ] Evaluate symmetry sets, state constraints, view functions, and decomposition only after baseline models are understandable. <!-- t:jk40 -->
- [ ] Record every state-space reduction and the behaviors it may merge or exclude. <!-- t:84yx -->

## Phase 7 — Counterexample and implementation integration

- [ ] Define JSON schema for normalized model traces and version it under the ticket. <!-- t:9w3a -->
- [ ] Implement a parser/exporter for TLC traces into normalized actions. <!-- t:h1o6 -->
- [ ] Implement a VerificationPlan adapter that maps normalized actions to native strict-provider steps. <!-- t:ikvg -->
- [ ] Preserve model constants, invariant ID, checker version, source hash, and original trace with every regression. <!-- t:h3zk -->
- [ ] Add differential checks between abstract observations, provider observations, security events, and durable row counts. <!-- t:9yft -->
- [ ] Add shrinking or minimization only if it preserves the violated formal property. <!-- t:cd9w -->
- [ ] Create a corpus directory for historical and model-generated counterexamples. <!-- t:oa9g -->
- [ ] Add review tooling that renders counterexample traces as human-readable timelines. <!-- t:ey1g -->

## Phase 8 — CI, governance, and release evidence

- [ ] Pin TLC and any selected checker versions with reproducible installation checksums. <!-- t:ac1i -->
- [ ] Add fast per-change models with explicit state-space budgets. <!-- t:euq3 -->
- [ ] Add scheduled/release configurations with larger bounds and archived statistics. <!-- t:o8ru -->
- [ ] Fail CI on invariant violation and tool error; report timeout/state exhaustion as inconclusive. <!-- t:6ry7 -->
- [ ] Publish counterexample artifacts and checker logs without secrets. <!-- t:zlgl -->
- [ ] Add specification review ownership and required reviewers for model or abstraction changes. <!-- t:kxa1 -->
- [ ] Bind model source hashes and invariant versions into the release evidence packet. <!-- t:2qcx -->
- [ ] Document how model evidence complements race, fuzz, failpoint, monitor, conformance, recovery, and human gates. <!-- t:s4z7 -->
- [ ] Run an independent formal-methods review of the three core models. <!-- t:uuj1 -->
- [ ] Approve the model-checking gate only after it reproduces historical defects and produces implementation regressions. <!-- t:8m8g -->

## Phase 9 — Long-term refinement and expansion

- [ ] Evaluate implementation refinement techniques only after the model-to-test loop is stable. <!-- t:g0p7 -->
- [ ] Consider logout/session revocation, forced password change, signing-key lifecycle, and administrative mutation models. <!-- t:dqql -->
- [ ] Revisit full token-family monitoring and production shadow verification. <!-- t:i3hi -->
- [ ] Track state-space growth and model maintenance cost across product changes. <!-- t:jogm -->
- [ ] Conduct periodic mutation exercises that intentionally break invariants and confirm checker/test/monitor detection. <!-- t:24rb -->
- [ ] Revise the literature packet and assumptions when OAuth/OIDC/FAPI profiles or Fosite behavior change. <!-- t:sgks -->
