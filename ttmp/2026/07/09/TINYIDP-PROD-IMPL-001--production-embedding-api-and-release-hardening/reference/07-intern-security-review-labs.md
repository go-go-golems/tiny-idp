---
Title: Intern Security Review Labs
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - auth
    - go
    - identity
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction_hardening_test.go
      Note: |-
        Browser protocol-state laboratory
        Browser security labs
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: |-
        Concurrency laboratory
        Concurrency labs
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: |-
        Transaction and failpoint laboratory
        Failure atomicity labs
    - Path: repo://internal/fositeadapter/verification_scenario_test.go
      Note: |-
        Typed programmable verification laboratory
        Programmable verification lab
ExternalSources: []
Summary: Executable first-week exercises that connect the intern textbook chapters to real tiny-idp code, tests, traces, and review artifacts.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Measuring whether a new contributor can reason about the system rather than merely repeat documentation.
WhenToUse: Complete in order during onboarding and review answers with a maintainer.
---


# Intern Security Review Labs

## Lab 1: Trace authority

Run:

```bash
go test ./internal/fositeadapter -run TestStrictAuthorizationCodeFlow -count=1 -v
```

Produce a table for every request value showing its origin, validator,
server-owned representation, mutation point, and final artifact. Explain why
`state`, nonce, interaction handle, code, and access token are not interchangeable.

## Lab 2: Reproduce temporal defenses

```bash
go test ./internal/fositeadapter -run 'TestForcedPromptLoginCannotReuseExistingSession|TestExpiredMaxAgeCannotReuseExistingSession|TestAuthorizationInteractionRejectsSequentialReplay|TestAuthorizationInteractionIsOneTimeUnderConcurrency' -count=1 -v
```

Draw each history. Identify the event whose absence would permit the defect and
the authoritative store transition that rejects replay.

## Lab 3: Audit browser authority

Read `resumeAuthorize` and the `tinyidpinteractioncontinuation` analyzer. List all
POST fields read by the handler, classify them as opaque/native user input or
forbidden protocol continuation, and explain the analyzer's false-negative
boundary.

## Lab 4: Break transactions safely

```bash
go test ./internal/fositeadapter -run 'TestSQLiteAuthorizationCodeRedemptionFailpointsAreAtomic|TestSQLiteRefreshRotationFailpointsAreAtomicAndRetryable' -count=1 -v
```

Choose two failpoints. Predict database rows and security events before running.
Compare the prediction with assertions and explain the transaction owner.

## Lab 5: Interpret concurrency

```bash
go test ./internal/fositeadapter -run 'TestInteractionConsumeHistoryIsLinearizable|TestRefreshRotationHistoryIsLinearizable' -count=10 -v
```

Write the sequential model, identify the linearization point, and explain why the
refresh family can end revoked after one successful response.

## Lab 6: Compare assurance methods

Run the custom analyzer, one Rapid test, one fuzz target, and the security trace
monitor. For each, write one supported claim and one unsupported claim.

```bash
go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...
go test ./internal/fositeadapter -run TestInteractionStoreStateMachine -count=1
go test ./internal/securitytrace -run '^$' -fuzz FuzzMonitorEventSequences -fuzztime 2s -parallel=1
```

## Lab 7: Review the scripting boundary

Read `pkg/verifyplan`, `internal/gojaverify`, and
`verification_scenario_test.go`. Enumerate every object reachable by JavaScript.
Explain why the plan can select an assertion but cannot implement its verdict.
State the in-process memory limitation.

## Lab 8: Make a release decision

Read reference 06 and the operations playbook. Write a release recommendation
that separates local technical evidence, hosted evidence, artifact provenance,
independent review, and owner authority. The correct current decision is not
approval; the exercise is to justify every row precisely.

## Detailed lab protocol

The short lab list above is the index. The following protocols define preparation,
procedure, expected evidence, interpretation limits, and mentor acceptance.

For every lab record:

```text
Commit:
Command:
Claim:
Threat or counterexample:
Observed representation:
Oracle:
Result:
Supported conclusion:
Unsupported conclusion:
Mentor question:
```

Do not edit expected results to match a failure. Investigate whether the cause is
a regression, an environment difference, an incorrect model, or stale text.

### Detailed Lab 1: Authorization authority trace

**Objective:** classify every value in one code + PKCE flow.

**Read first:** design 06; `provider.go`; `interaction.go`;
`TestStrictAuthorizationCodeFlow`.

**Procedure:**

1. List raw authorization query values.
2. Locate Fosite validation.
3. Locate canonical request creation.
4. List rendered form fields.
5. List POST fields read on resume.
6. Locate interaction consumption.
7. Locate code/PKCE/OIDC writes.
8. Parse callback code and state.
9. Locate verifier comparison.
10. Locate access-token introspection.

**Required rows:** client ID, redirect, response type, scopes, state, nonce,
challenge, method, prompt, max age, interaction, CSRF, login, password, consent,
code, verifier, access, refresh, session cookie, signing key ID.

**Expected evidence:** protocol values become authoritative only after validation
and server-owned persistence; the browser submit contains only native action
inputs and opaque handles.

**Questions:**

- Which values are correlation values?
- Which are bearer capabilities?
- Which may appear in evidence?
- Why are state, nonce, CSRF, and interaction distinct?

**Limit:** one success path does not cover rejection, concurrency, or hosted
interoperability.

**Acceptance:** the intern redraws the sequence and explains every binding.

### Detailed Lab 2: Temporal authorization histories

**Objective:** express requirements as event histories.

**Read first:** design 07; interaction state types; hardening tests.

**Run:** the four indexed tests plus prompt-none, consent denial, omitted
decision, expiry, client mutation, and user mutation.

For each write:

```text
initial session
request parameters
required actions
events before submit
submit action
consume result
events after submit
artifact result
```

**Expected evidence:** forced login and expired max age require post-creation
authentication; one handle has one terminal success; prompt none creates no
interactive path when obligations remain.

**Questions:**

- Why is a non-empty login field not the invariant?
- Why is expiry checked in consume?
- Why can independent tabs succeed?
- Which failures preserve retryability?

**Limit:** named histories do not establish every legal or illegal sequence.

**Acceptance:** properties are stated without coupling to HTML implementation.

### Detailed Lab 3: Browser authority and static guard

**Objective:** prove browser action is not OAuth continuation authority.

**Read first:** `renderInteraction`, `resumeAuthorize`, canonical reconstruction,
`tinyidpinteractioncontinuation`, and fixtures.

**Procedure:**

1. Inventory hidden and visible form inputs.
2. Inventory all POST reads.
3. Classify interaction, CSRF, login, password, action, and consent approval.
4. Search for client, redirect, scope, state, nonce, prompt, max age, PKCE,
   claims, audience, and resource reads.
5. Run analyzer tests.
6. Run repository analyzer scan.
7. Read the state-mutation regression.

**Expected evidence:** no protocol continuation fields are rendered/read; direct
forbidden reads in resume would be diagnosed.

**Questions:**

- What interprocedural helper could evade the rule?
- Why does CSRF not authenticate hidden protocol data?
- What would signed hidden fields still omit?

**Limit:** local AST analysis is complemented by render and mutation tests.

**Acceptance:** the intern states a concrete false negative.

### Detailed Lab 4: Authorization artifact transaction

**Objective:** enumerate the complete authorize mutation set.

**Read first:** `beginAuthorizeLifecycle`, `authorizeExec`, `finishAuthorize`,
authorization failpoint tests.

**Procedure:**

1. Draw transaction begin.
2. Place code-session write.
3. Place PKCE write.
4. Place OIDC-session write.
5. Place interaction terminal consume.
6. Place commit.
7. Place security events.
8. Place HTTP redirect write.
9. Run positive commit proof.
10. Run rollback table.

**Prediction columns:** transaction, mutation executed, rollback, interaction,
code count, PKCE count, OIDC count, event count, retry.

**Expected evidence:** approved terminal state exists iff the complete artifact
set commits in production SQLite.

**Questions:** why join interaction terminal state; why events after commit; what
happens if redirect delivery fails?

**Limit:** development memory composition has a different evidence boundary.

**Acceptance:** no mutation is hidden under the phrase “Fosite writes state.”

### Detailed Lab 5: Code redemption transaction

**Objective:** prove code invalidation and token creation are one lifecycle.

**Read first:** saved Fosite Transactional API; `BeginTX`, `Commit`, `Rollback`,
`tokenExec`, redemption test.

**Procedure:** select failpoints before invalidation, after invalidation, after
access creation, after refresh creation, and before commit. Predict state, run,
and compare.

**Expected failure state:** active code one; access zero; refresh zero; new token
commit events zero.

**Expected success state:** code inactive; access present; refresh present when
grant requests it.

**Questions:**

- Why does Fosite own transaction begin?
- Why is a private code-store transaction insufficient?
- What does response loss after commit mean for retry?

**Limit:** eight hooks cover enumerated dependencies, not every crash mode.

**Acceptance:** the intern identifies response exposure after commit.

### Detailed Lab 6: Refresh rotation and reuse

**Objective:** separate rotation from reuse-driven family revocation.

**Read first:** `RotateRefreshToken`, refresh failpoints, reuse test, Porcupine
history, diary Step 23.

**Procedure:**

1. Write the conditional SQL predicate.
2. Explain affected-row semantics.
3. Trace old access deletion.
4. Trace replacement access/refresh creation.
5. Run ten failpoints and retry.
6. Run eight-client concurrent history.
7. Inspect final active rows.

**Expected evidence:** failpoints restore original pair and retry; one concurrent
rotation wins; later old-token reuse can revoke replacement family.

**Questions:**

- What is the linearization point?
- Why did the first final-state assertion fail?
- How should clients serialize refresh?

**Limit:** current Porcupine model abstracts reuse as a later operation.

**Acceptance:** green linearizability and revoked final family are reconciled.

### Detailed Lab 7: Race detection versus linearizability

**Objective:** compare memory and semantic concurrency properties.

**Run:** focused race tests and both Porcupine tests ten times.

**Procedure:** define sequential model, identify call/return times, preserve
non-overlap order, locate winner operation, then compare final state.

**Expected evidence:** no observed Go data races in executed paths; histories
admit legal sequential explanations.

**Questions:**

- Can a mutex-protected implementation violate exactly once?
- Can a linearizable history contain operation failures?
- What does schedule repetition not enumerate?

**Limit:** bounded histories and models, not exhaustive schedules.

**Acceptance:** the intern never uses “race-free” as synonym for correct.

### Detailed Lab 8: Strict parsing and duplicates

**Objective:** design parsing that distinguishes absence, zero, error, and
conflicting duplicates.

**Read first:** `parseMaxAge`, malformed tests, fuzz target,
`tinyidpstrictparse`, diary Step 22.

**Inputs:** absent, zero, positive, plus sign, negative, whitespace, float,
overflow, duplicate equal, duplicate conflicting.

**Record:** parser value, presence, error, OAuth response, credential form, and
artifact absence.

**Expected evidence:** only bounded non-negative decimal syntax is accepted;
malformed input never becomes absence.

**Questions:** why did the first analyzer flag the tuple parser; what duplicate
policy is explicit versus accidental `url.Values` behavior?

**Limit:** the analyzer intentionally covers single-boolean predicates only.

**Acceptance:** propose a strict duplicate-aware scalar parser in pseudocode.

### Detailed Lab 9: Session-state distinctions

**Objective:** distinguish missing, inactive, invalid, unavailable, disabled, and
active session conditions.

**Read first:** `browserSessionState`, `readBrowserSession`, session tests, fault
store, user revalidation.

**Cases:** no cookie; malformed cookie; absent row; expired; revoked; store
error; disabled user; active user/session; stale cookie during interaction
creation.

**Expected evidence:** store error is service unavailable with no credential
form; inactive state creates login obligation; disabled user cannot authorize;
stale cookie is not bound as active session.

**Questions:** which are availability facts; why reload user; which cookie
attributes are host-owned?

**Limit:** local cookie/storage tests do not model browser implementation bugs.

**Acceptance:** no state is collapsed into a zero-value user shortcut.

### Detailed Lab 10: Authentication outcomes and bounded work

**Objective:** trace typed authentication beyond password equality.

**Read first:** authenticator contracts, password result, Argon parser/hash,
password-work controller, lockout, audit reasons.

**Cases:** valid; invalid; disabled credential; locked account; unavailable
store; saturated work; canceled context; must-change success.

**Expected evidence:** invalid and unavailable differ; work is bounded; stable
audit reasons exclude raw errors; must-change cannot issue artifacts.

**Questions:** how much memory can capacity N consume; why is verifier success
not always authorization; which account update must be atomic?

**Limit:** local work tests require target capacity validation.

**Acceptance:** the intern proposes password-change continuation as typed native
state, not a boolean exception.

### Detailed Lab 11: Consent and scope algebra

**Objective:** model consent over user, client, scope set, time, and revocation.

**Read first:** stored consent, consent tests, rendered disclosure, terminal
denial.

**Relations:** requested subset, equal, superset, overlap, disjoint; then add
expired and revoked prior consent.

**Expected evidence:** prior consent never grants added scopes; explicit denial
is terminal access denied; omitted action is not approval; POST does not return
authoritative scope/client.

**Questions:** does consent join issuance transaction; what retry ambiguity
remains; which event represents explicit approval versus policy skip?

**Limit:** consent transactional coupling remains an open design question.

**Acceptance:** the intern separates authentication, policy skip, and user
decision.

### Detailed Lab 12: UserInfo transport and claims

**Objective:** review bearer transport, cache, challenge, methods, current user,
and scope-controlled claims.

**Cases:** no credential; malformed scheme; duplicate header; query token; form
token; mixed transport; invalid token; valid GET; valid POST; unsupported method;
disabled user.

**Expected evidence:** one header accepted; query/form/mixed rejected; invalid
token challenged; all responses no-store; claims respect scopes/current state.

**Questions:** why not generic Fosite extraction; why reject duplicates; which
standard claims are protected; how should custom claims be namespaced?

**Limit:** tests cover current claim/profile set, not every RP interpretation.

**Acceptance:** the intern distinguishes invalid token and invalid request.

### Detailed Lab 13: Static analyzer precision

**Objective:** review four analyzers as small verification products.

Choose public API, bearer/continuation provenance, protocol lifecycle, and ignored
security error rules.

For each record:

```text
originating defect
AST/type pattern
positive fixture
negative fixture
directive/suppression
false positive
false negative
behavioral complement
```

**Expected evidence:** repository scan is clean; each rule supports a narrow
claim.

**Questions:** when would IFDS/CodeQL be justified; why narrow false-positive
rules; who updates call allowlists?

**Limit:** checked syntax and types are not whole-program semantic proof.

**Acceptance:** the intern can improve a fixture without broad suppression.

### Detailed Lab 14: Property, fuzz, and metamorphic oracles

**Objective:** compare generated techniques by semantic input and oracle.

**Run:** Rapid interaction model, action-sequence fuzz, max-age fuzz, monitor
fuzz, and `ui_locales` metamorphic test.

**Record:** generator, corpus, model, shrink, coverage, oracle, duration,
execution count, counterexample format.

**Expected evidence:** Rapid explores legal/adversarial commands and shrinks;
fuzz explores byte/action coverage; metamorphic compares a relation across
executions.

**Questions:** why not fuzz two targets in one package concurrently; what is a
valid versus invalid transform; what must be persisted for replay?

**Limit:** bounded runs and incomplete model/relations.

**Acceptance:** “no counterexample in this run” replaces “proved correct.”

### Detailed Lab 15: Runtime monitor

**Objective:** connect event instrumentation to finite bad-prefix verdicts.

**Read first:** event schema, recorder, monitor, provider emission sites, monitor
properties/fuzz, failpoint trace feeds.

**Procedure:** construct one valid forced-login trace, one missing-auth trace,
one duplicate terminal, one denied-with-artifact, one unsupported version, and
two interleaved interaction partitions.

**Expected evidence:** valid accepted; each invalid trace reports relevant
violation; partitions do not satisfy each other; token events are ignored by the
interaction automaton as designed.

**Questions:** what if an event is never emitted; why after commit; should
delivery failure affect readiness?

**Limit:** monitor correctness is conditional on instrumentation completeness.

**Acceptance:** the intern proposes an instrumentation-completeness test.

### Detailed Lab 16: Goja capability boundary

**Objective:** prove scripts author plans but do not execute or judge security.

**Read first:** plan types/limits, module loader, compiler, tests, strict driver,
native assertions.

**Run:** valid compile, native-run integration, ambient `fs` rejection, infinite
loop interruption, normal workspace and `GOWORK=off`.

**Reachability table:** filesystem, network, environment, process, store,
provider, key, clock, assertion, live observation.

**Expected evidence:** all live capabilities absent; only normalized plan data;
native driver and assertion own effects/verdicts.

**Questions:** what does source hash prove; what does deadline not limit; what
would one new native module expose?

**Limit:** no hard heap sandbox for hostile scripts.

**Acceptance:** the intern can threat-model a proposed new module.

### Detailed Lab 17: SQLite backup and recovery

**Objective:** execute WAL-safe backup, verification, restore, rollback, and
schema checks.

**Read first:** original file-copy probe, online backup, manifest, verification,
restore, migration ledger, release drill.

**Run:** online backup/restore, concurrent write, disk-full publication,
corruption, future schema, and release drill tests.

**Expected evidence:** WAL commits included; failed publication preserves prior
backup; manifest binds schema/checksums/counts/keys; restore is offline and keeps
rollback; doctor confirms expected boundary.

**Questions:** why can a bad copy open; what if doctor fails; which filesystem
faults remain; when create a new backup?

**Limit:** temporary local filesystem is not target-environment proof.

**Acceptance:** the intern can write the recovery sequence without “copy DB.”

### Detailed Lab 18: Proxy, limiter, and production host

**Objective:** review deployed address trust and host-owned controls.

**Read first:** resolver, limiter phases, production command, server settings,
readiness, production-host README.

**Cases:** untrusted peer forged XFF; trusted direct client; trusted intermediate;
malformed chain; excessive hops; claimed client pre-auth; verified client
post-auth.

**Host table:** TLS, timeouts, body/header bounds, secure cookies, secret file,
audit file, SQLite topology, maintenance, signals, shutdown.

**Expected evidence:** trust begins at immediate peer; normalized address excludes
port; attacker-claimed client does not shard pre-auth buckets; host controls are
explicit.

**Questions:** what must proxy sanitize; which controls cannot provider validate;
what target tests remain?

**Limit:** resolver unit and local tmux smoke are not deployed proxy evidence.

**Acceptance:** the intern can review an actual proxy topology.

### Detailed Lab 19: Keys, secrets, audit, and readiness

**Objective:** connect cryptographic/operational lifecycle to observable health.

**Read first:** key rotation/purge, token secret loading/rotation test, audit sink,
delivery counters, readiness, maintenance.

Draw planned key overlap, emergency purge, and token-secret immediate cutover.
List all eight readiness components and operator response.

**Expected evidence:** planned old key remains verifiable; emergency purge may
invalidate; old opaque tokens fail after secret rotation; audit fsync applies
backpressure; stale maintenance degrades readiness.

**Questions:** why separate audit and security events; what is post-commit audit
ambiguity; should monitor delivery gate readiness?

**Limit:** key/secret compromise prevention belongs to target secret management.

**Acceptance:** the intern distinguishes planned availability from emergency
revocation.

### Detailed Lab 20: Exact release decision

**Objective:** integrate all evidence without substituting authority classes.

**Read first:** exact-candidate reference, release workflows, hosted runner,
runbook, residual risks, approval algorithm.

**Procedure:** record code candidate/hash; list local gates; list harness failures
and fixes; list missing hosted/scanner/signature/reviewer/owner rows; state next
authorized actions.

**Expected recommendation:** strong local evidence; production release NOT
APPROVED; run exact-hash external gates and obtain human authority.

**Questions:** why can't signature prove protocol; why can't OIDF prove audit;
why can't old hosted results bind new bytes; who may sign approval?

**Limit:** the exercise cannot create external or human evidence.

**Acceptance:** evidence quantity is never used as a substitute for a missing
class.

## Extended design assignment

Choose password-change continuation, step-up authentication, token-family
monitor, consent expiry, logout/session revocation, or duplicate parameter
policy.

Submit:

1. normative/local requirement;
2. attacker and assumptions;
3. authoritative state;
4. legal transition model;
5. persistence/concurrency boundary;
6. audit/security events;
7. analyzer opportunity and precision;
8. deterministic tests;
9. property/fuzz/metamorphic tests;
10. failpoints/histories;
11. production/recovery impact;
12. evidence limitations;
13. phased files and commands.

Do not begin implementation until the proposal contains one minimal
counterexample and distinguishes observation, detection, and enforcement.

## Oral review bank

1. Trace `prompt=login` to ID-token `auth_time`.
2. Explain why browser POST is not protocol authority.
3. Explain state, nonce, CSRF, and interaction separately.
4. Define one terminal interaction outcome.
5. Define authorization artifact transaction.
6. Define token transaction.
7. Locate refresh linearization point.
8. Explain family revocation after a winner.
9. Compare race and Porcupine.
10. State one analyzer false negative.
11. Explain Rapid shrinking.
12. State one fuzz evidence limit.
13. Explain monitor observability.
14. Enumerate JavaScript capabilities.
15. Walk XFF right-to-left.
16. Explain WAL-safe backup.
17. Explain all readiness components.
18. Interpret zero reachable vulnerabilities.
19. Explain hosted evidence identity.
20. Defend NOT APPROVED.

## Command index

### Baseline

```bash
go test ./... -count=1
go vet ./...
make lint
```

### Race

```bash
GOWORK=off go test -race ./... -count=1
```

### Static analysis

```bash
go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...
```

### Local conformance

```bash
GOWORK=off bash scripts/run-conformance.sh
```

### Recovery

```bash
GOWORK=off bash ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/02-release-drills.sh
```

### Fuzz discipline

Run one fuzz target per package command. Record duration, executions, new
interesting inputs, seed/corpus, commit, and result. Do not run two fuzz targets
for the same package concurrently.

## Mentor scoring rubric

### Insufficient protocol reasoning

Lists endpoints and fields without principals, validation, bindings, or artifact
authority.

### Ready protocol reasoning

Traces success and rejection, classifies correlation/capability values, and
connects formal property to current code.

### Insufficient state reasoning

Uses handler booleans/status only.

### Ready state reasoning

Defines predecessors, obligations, terminal behavior, replay, expiry, and event.

### Insufficient persistence reasoning

Says “transactional” without mutation set or coordinator.

### Ready persistence reasoning

Lists rows, context capability, failpoints, rollback, retry, and post-commit
ambiguity.

### Insufficient evidence reasoning

Treats coverage, lint, fuzz, monitor, or conformance as universal proof.

### Ready evidence reasoning

States claim, oracle, observed representation, blind spot, and complementary
method.

### Insufficient production reasoning

Assumes correct handler equals secure deployment.

### Ready production reasoning

Covers host, proxy, secret, audit, readiness, recovery, artifact, external
evidence, reviewer, and owner.

## Lab submission review checklist

For each submission the mentor checks:

- command includes explicit package/target and count/duration where relevant;
- commit and toolchain are recorded;
- working-tree assumptions are stated;
- claim is one falsifiable sentence;
- attacker/counterexample is concrete;
- authoritative state is a named type/field/store row;
- transition is a named function/method;
- oracle inspects protected artifact or temporal fact;
- result includes actual status/count/diagnostic;
- limitation names a real blind spot;
- source or research concept is cited without normative inflation;
- proposed follow-up is scoped and authorized.

## Example passing evidence note

```text
Commit: 5bb4dae...
Command: go test ./internal/fositeadapter -run
  TestSQLiteAuthorizationCodeRedemptionFailpointsAreAtomic -count=1
Claim: Any named failure in code redemption leaves the original code active and
  creates no access or refresh rows.
Threat: Partial token transaction consumes a one-time code or creates incomplete
  authority.
Observed representation: fosite_authorize_codes, fosite_access_tokens,
  fosite_refresh_tokens, securitytrace recorder.
Oracle: active code count 1; access count 0; refresh count 0; new token commit
  events 0 at each failpoint.
Result: PASS for eight named points.
Supported conclusion: The enumerated adapter lifecycle failures roll back in the
  tested SQLite topology.
Unsupported conclusion: Every possible power/filesystem/kernel failure is safe.
Research link: transaction atomicity and lineage-oriented failure placement.
```

## Example insufficient evidence note

```text
The token tests pass, so refresh is safe.
```

This note omits commit, command, threat, state, oracle, concurrency, reuse,
failures, and limitation. The mentor returns it without approval.

## First-contribution gate

Before code assignment, the intern must complete Labs 1–8 and the oral questions
for protocol, temporal, transaction, and assurance reasoning. Before a
production-sensitive assignment, complete all twenty labs.

Good first contributions:

- analyzer fixture/precision improvement;
- negative transport/parser regression;
- monitor diagnostic test;
- redacted strict-driver observation;
- readiness reason improvement;
- recovery drill assertion;
- unsupported parameter documentation.

Changes requiring experienced pairing:

- new grant or authentication mechanism;
- token format or signing changes;
- transaction/lifecycle rewrite;
- distributed persistence;
- live Goja capability;
- production topology or release approval.

## Lab maintenance protocol

After each onboarding cohort record:

1. stale commands or symbol names;
2. ambiguous expected observations;
3. missing prerequisites;
4. repeated misconceptions;
5. labs completed mechanically without understanding;
6. useful counterexamples proposed by the intern;
7. code defects found during labs;
8. mentor explanations that should become prose;
9. runtime cost of the full workbook;
10. recommended ordering changes.

Update the workbook in a focused documentation commit with diary provenance.

## Completion report template

```text
Intern:
Mentor:
Dates:
Repository commit:
Go/toolchain/workspace:
Labs completed:
Commands and artifacts:
Protocol diagram:
Interaction state model:
Authorization/token transaction diagrams:
Concurrency model:
Analyzer precision reviews:
Generated/fuzz evidence reviews:
Monitor/Goja authority diagram:
Production control table:
Release recommendation:
Open questions:
Proposed first contribution:
Mentor assessment:
Scope authorized:
```

Completion authorizes only the recorded contribution scope. It is not an
independent security review or release approval.

## Expected final diagrams

The completion report embeds or links these diagrams:

1. Principal/channel diagram.
2. Authorization Code + PKCE sequence.
3. Interaction state machine.
4. Forced-login valid and invalid histories.
5. Authorization artifact transaction.
6. Code redemption transaction.
7. Refresh rotation and reuse family graph.
8. Interaction consume concurrent history.
9. Event instrumentation and monitor boundary.
10. Goja compile-plan/native-run authority boundary.
11. Production host and proxy trust boundary.
12. Release evidence dependency graph.

Every diagram labels authoritative state and trust direction. Decorative
components without security meaning should be omitted.

## Expected final source references

The completion report cites at least:

- one OAuth/OIDC normative source;
- one formal protocol paper;
- one state/model testing paper;
- one static-analysis source;
- one concurrency/linearizability source;
- one fault-injection source;
- one runtime-verification source;
- one production/authenticator source;
- one current design document;
- one diary interval;
- five concrete code symbols;
- five concrete tests or tools.

The citations explain influence. A bibliography without a claim-to-code link is
not sufficient.

## Expected final counterexamples

The intern can reproduce or explain these minimal histories:

```text
old session + prompt login + blank submit
create + approve + approve
create + expire + approve
consent required + omitted decision
code invalidated + access write + injected failure
refresh rotate winner + old-token reuse
trusted proxy omitted + forged forwarding header
valid-looking SQLite main-file copy missing WAL commit
```

For each, name the repair and the evidence that prevents recurrence.

## Workbook provenance

This workbook derives from diary Steps 17–26, designs 02–10, the saved standards
and papers under `sources/`, the custom analyzer and runtime tools under the
review ticket, and current production source/tests. It should be revised when any
of those contracts change.

## Final mentor declaration

The mentor records whether the intern can:

- reason from property to state transition;
- reason from transition to transaction/concurrency;
- reason from claim to bounded evidence;
- reason from local evidence to production authority;
- preserve uncertainty and ask for review before expanding scope.

If any item is not demonstrated, assign the relevant lab again with a new
counterexample. Completion is based on reasoning artifacts, not elapsed days.

The completed workbook becomes onboarding evidence for the contributor, not a
permanent credential. Major changes to grants, authentication methods, storage,
the scripting capability boundary, or production topology require focused
re-onboarding and review against the new implementation.

The mentor records which modules must be repeated.

## Completion artifact

Submit a short review note with:

- one protocol invariant;
- one temporal counterexample;
- one transaction boundary;
- one concurrent history;
- one analyzer precision limit;
- one fuzz or property evidence limit;
- one production trust boundary;
- one open release blocker.

The mentor should challenge any statement that lacks a code symbol, test,
standard, trace, or explicit inference.
