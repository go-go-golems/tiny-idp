---
Title: Intern Accelerated Curriculum and Code Reading Map
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - identity
    - oidc
    - research
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Central strict OAuth and OpenID Connect request path
        Central strict protocol flow
    - Path: repo://internal/securitytrace/trace.go
      Note: |-
        Versioned security events and native temporal monitor
        Temporal evidence model
    - Path: repo://pkg/embeddedidp/options.go
      Note: |-
        Public construction, validation, and host ownership boundary
        Public production boundary
    - Path: repo://pkg/idpstore/interfaces.go
      Note: |-
        Durable security-state contracts and transaction boundary
        Durable security state contracts
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/01-implementation-diary.md
      Note: Chronological implementation evidence and corrected assumptions
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/04-validate-intern-textbook.sh
      Note: Reproducible corpus depth and structure validation
ExternalSources: []
Summary: A one-week theory-to-code curriculum that gives a new intern the protocol, state-machine, persistence, assurance, and production knowledge required to contribute safely to tiny-idp.
LastUpdated: 2026-07-10T22:00:00-04:00
WhatFor: Onboarding a capable engineer without requiring them to reconstruct the system or its security theory from commits and issue history.
WhenToUse: Start here on the first day, then follow the reading, execution, and review exercises in order.
---



# Intern Accelerated Curriculum and Code Reading Map

## Purpose

An identity provider is a protocol implementation, a security-state machine, a
credential verifier, a capability issuer, and an operational service. A change
that looks local to one HTTP handler can affect replay resistance, authentication
freshness, consent, token authority, transaction ordering, or recovery. The
onboarding program must therefore teach the model before asking an intern to
modify the implementation.

This curriculum compresses the necessary foundation into five ordered modules.
It does not ask the reader to memorize OAuth parameters or package names. It
teaches how to derive a security property, locate its authoritative state,
identify the transition that changes that state, and select evidence that can
actually support the claim.

By the end, the intern should be able to answer four questions for any proposed
change:

1. Which principal is supplying each input, and which inputs are authoritative?
2. Which state transition creates, consumes, rotates, or revokes authority?
3. Which invariant must hold before and after that transition?
4. Which test or analysis technique can falsify the implementation claim?

## The five-module sequence

| Module | Core question | Primary chapter | Demonstration |
|---|---|---|---|
| Protocol security | What must an authorization server preserve across messages? | `06-oauth-oidc-protocol-security-foundations.md` | Trace Authorization Code + PKCE |
| Temporal reasoning | What sequences are legal, not merely what inputs are valid? | `07-security-state-machines-and-temporal-invariants.md` | Replay and forced-login histories |
| Durable authority | Where is the atomic and concurrent linearization point? | `08-durable-security-state-transactions-and-concurrency.md` | Failpoint and Porcupine tests |
| Assurance science | What does each tool prove, miss, or merely suggest? | `09-assurance-methods-and-evidence-interpretation.md` | Analyzer, Rapid, fuzz, monitor |
| Production security | Which guarantees belong to the host and release process? | `10-production-trust-boundaries-and-release-security.md` | Readiness, recovery, proxy, release ledger |

The practical workbook in
`reference/07-intern-security-review-labs.md` turns these chapters into commands,
code-reading questions, and small review assignments.

## Evidence vocabulary

Every onboarding document uses four labels. They prevent an implementation
choice from being presented as if a standard required it.

- **Normative:** required by an RFC, OpenID specification, or documented public
  contract.
- **Observed:** established by the current code, dependency source, tests, or a
  recorded execution.
- **Inferred:** an engineering mechanism selected to realize or test a property.
- **Open:** a question or risk for which current evidence is incomplete.

Example:

```text
Normative: an authorization code must be short-lived and single-use.
Observed: Fosite invalidates the code during token response construction.
Inferred: code invalidation and replacement token creation must share one SQL
          transaction so a failed response cannot consume only half the state.
Open: hosted conformance for the exact candidate remains external evidence.
```

## System map before details

```text
browser / relying party
          |
          v
  net/http host boundary
          |
          v
  embeddedidp.Provider
          |
          v
  fositeadapter.Provider ------------------+
     |      |       |       |              |
     |      |       |       |              +--> audit / security events
     |      |       |       +-----------------> authentication / rate limit
     |      |       +-------------------------> consent policy
     |      +---------------------------------> Fosite protocol engine
     +----------------------------------------> idpstore.Store
                                                    |
                                       memory or SQLite implementation
```

The arrows represent authority and dependency, not simply function calls.
Fosite validates and constructs protocol messages. tiny-idp owns browser
interactions, authentication, consent, sessions, host policy, and durable store
integration. The store owns state transitions that must remain atomic across
concurrent requests and process restarts.

## First-day reading order

### 1. Read the public boundary

Start with `pkg/embeddedidp/options.go` and `provider.go`. Identify what a host
must provide in production mode: issuer, store, cookies, token secret, audit,
rate limiting, client-address resolution, authentication controls, and
maintenance. Then read `pkg/idp/contracts.go`, `pkg/idpstore/interfaces.go`, and
`pkg/idpstore/types.go`.

The goal is to understand ownership. Do not begin in `cmd/tinyidp`; the command
is one host of the library, not the definition of the provider contract.

### 2. Trace one successful flow

Read these symbols in order:

```text
fositeadapter.NewProvider
Provider.Handler
Provider.handleAuthorize
Provider.beginAuthorize
Provider.resumeAuthorize
Provider.finishAuthorize
Provider.handleToken
Provider.handleUserInfo
```

Keep a table with four columns: input origin, validation owner, durable mutation,
and emitted artifact. This prevents browser form data, validated OAuth request
data, and server-owned interaction state from being conflated.

### 3. Trace one rejected flow

Use `TestForcedPromptLoginCannotReuseExistingSession`. Follow the request with
an existing browser session, `prompt=login`, and an empty interaction submit.
Locate the stored required action and the branch that rejects missing login.
Then follow the security events and confirm no authorization artifact event is
emitted.

### 4. Trace one failure inside persistence

Use `TestSQLiteAuthorizationCodeRedemptionFailpointsAreAtomic`. Pick the
`after_create_access` failpoint. Locate the Fosite transaction context, the SQL
executor selected from that context, rollback, and the assertions showing that
the authorization code remains usable and no access or refresh row survives.

### 5. Trace one concurrent history

Read `linearizability_test.go` before reading its output. Write down the
sequential model first. Then inspect call/return intervals and Porcupine's
verdict. For refresh, distinguish the successful rotation from later reuse
detection that revokes the winning family.

## Reading existing documents without duplication

The new chapters teach foundations. Existing documents serve different roles:

| Document | Use it for |
|---|---|
| Design 01 | Complete public API, storage, host, and implementation plan |
| Design 02 | Full invariant catalog and assurance architecture |
| Design 03 | Primary research-to-code decisions for interactions and tokens |
| Design 04 | Goja verification-plan capability boundary |
| Reference 04 | Original authorization review findings |
| Reference 05 | Task-by-task implementation ledger |
| Reference 06 | Exact-candidate evidence and honest missing gates |
| Diary | Chronology, failures, corrected assumptions, and commits |

The intern should use the textbook chapters to form the model, then use the
older documents to inspect full detail and history.

## Expected competence after one week

The intern is ready for a small security-sensitive change when they can:

- explain `prompt=login`, `max_age`, `prompt=none`, consent, `auth_time`, PKCE,
  code consumption, refresh rotation, and UserInfo bearer transport precisely;
- draw the interaction state machine and identify its terminal transition;
- explain why CSRF protection does not make browser-carried protocol state
  authoritative;
- locate the SQL transaction spanning a complete protocol mutation;
- distinguish a race detector result from linearizability evidence;
- state a custom analyzer's false-negative boundary;
- design a deterministic test with an injected clock;
- describe what may and may not be exposed to a Goja verification script;
- interpret readiness, audit delivery, recovery, and conformance as separate
  release claims;
- leave a diary entry that separates requirement, observation, inference, and
  remaining uncertainty.

## Review assignment for the mentor

Give the intern one narrow proposal, such as adding a new authorization request
parameter or security event. Ask for a one-page review containing:

1. the protocol semantics and threat model;
2. the authoritative source of the value;
3. the state transitions it influences;
4. persistence and concurrency consequences;
5. static, deterministic, generated, fuzz, and external evidence choices;
6. expected audit and security events;
7. an explicit list of claims the proposed tests would not prove.

The assignment should be reviewed before code is written. The quality criterion
is not volume. It is whether each claim is connected to an authoritative source,
a concrete transition, and an appropriate falsification method.

## How this curriculum was reconstructed

The curriculum follows the order in which production risk became visible, not
the chronological order of source files. Diary Steps 5–6 established dependency
and release baselines. Steps 7–10 discovered that the public embedding boundary
was itself a security and operability concern. Steps 11–12 built atomic stores
and correct recovery. Steps 13–14 added bounded authentication work, abuse
control, audit, keys, readiness, and maintenance. Step 15 built the production
host and evidence ledger.

Steps 17–22 then exposed the deeper protocol issue: locally reasonable handlers
could violate cross-request invariants. Steps 23–25 built transactions, models,
linearizability checks, trace monitoring, analyzers, and programmable plans.
Step 26 reran exact-candidate evidence and preserved missing external gates.

This history determines the teaching order:

```text
protocol meaning
    -> temporal obligations
    -> durable/concurrent authority
    -> assurance method selection
    -> production and release authority
```

Starting with command-line flags or SQL tables would show implementation before
the properties that justify it.

## Curriculum contract for mentor and intern

The intern is expected to read source and run code. The mentor is expected to
challenge reasoning, not simply verify command completion.

Each module produces four artifacts:

1. a state/sequence diagram drawn by the intern;
2. a code map with exact symbols;
3. an evidence statement containing claim and limitation;
4. one proposed counterexample or negative test.

The mentor should reject answers that rely on vague phrases such as “more
secure,” “validated,” “atomic,” or “covered” without naming what is validated,
which mutations are atomic, or which property the evidence covers.

## Day 0: environment and evidence preservation

The first task is not feature work. The intern confirms the repository, branch,
toolchain, workspace, and working-tree ownership.

Run:

```bash
git status --short
git branch --show-current
go version
go env GOWORK GOVERSION GOTOOLCHAIN CGO_ENABLED
go test ./... -count=1
docmgr doctor --ticket TINYIDP-PROD-IMPL-001
```

The expected result is a green suite and only the already-known untracked hosted
evidence directories. The intern must not add, delete, or reinterpret those
directories.

Read the latest diary steps and commit log:

```bash
git log --oneline -20
rg -n '^## Step' ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/01-implementation-diary.md
```

Learning goal: evidence belongs to a specific commit and working-tree state.

Deliverable: a short baseline note containing branch, HEAD, toolchain, full-test
result, known untracked paths, and current release decision.

## Day 1 morning: principals, protocol, and artifacts

Read design 06 through the successful sequence, threat model, Fosite composition,
and parameter authority catalog. Read RFC 6749 roles and code flow, RFC 7636
PKCE, RFC 9700 recommendations, and OIDC Core sections on authorization, nonce,
prompt, max age, and auth time.

Then trace `TestStrictAuthorizationCodeFlow`.

Build this table:

| Value | Created by | Validated by | Stored as | Consumed by |
|---|---|---|---|---|
| client ID | client | Fosite/store | canonical requester | authorize/token |
| redirect URI | client | exact client policy | interaction/requester | redirect/token |
| state | client | opaque preservation | canonical request | callback |
| nonce | client | OIDC handler | OIDC session | RP ID-token check |
| challenge | client | PKCE handler | PKCE session | verifier check |
| interaction | server | store/browser binding | hash-indexed record | authorize POST |
| code | server | Fosite/token store | signature/requester | token endpoint |
| access token | server | Fosite introspection | signature/requester | UserInfo |
| refresh token | server | Fosite refresh handler | family requester | token endpoint |

The intern must explain why state, nonce, CSRF, and interaction handle solve
different binding problems.

## Day 1 afternoon: public architecture

Read:

```text
pkg/embeddedidp/options.go
pkg/embeddedidp/provider.go
pkg/idp/contracts.go
pkg/idpstore/interfaces.go
pkg/idpstore/types.go
pkg/sqlitestore/store.go
```

For each package, answer:

- Which authority does it expose?
- Which dependency does it intentionally hide?
- Which production checks belong here?
- Which interface could be narrower?
- Which implementation is replaceable?

The key distinction is between protocol engine, product policy, durable state,
and process host. Fosite is internal. The host sees public tiny-idp contracts.

Deliverable: redraw the package map with arrows labeled by interface names.

## Day 2 morning: authorization as a state machine

Read design 07 and these symbols:

```text
InteractionRequiredAction
InteractionOutcome
InteractionRecord
Provider.beginAuthorize
Provider.resumeAuthorize
Provider.createInteraction
Provider.reconstructAuthorizeRequest
Store.CreateInteraction
Store.GetInteraction
Store.ConsumeInteraction
```

Run the forced-login, max-age, prompt-none, consent, replay, mutation, tab, and
expiry tests. For each, write the pre-state, action, result, post-state, and
forbidden artifact.

The mentor should ask the intern to remove one stored field conceptually and
explain the resulting counterexample.

## Day 2 afternoon: authentication and consent

Trace password authentication from `resumeAuthorize` through limiter,
`PasswordAuthenticator`, password-work controller, credential verification,
account-security update, browser session creation, audit, and security event.

Inspect the typed result that includes `MustChangePassword`. Explain why a
successful hash comparison is not sufficient permission to issue tokens.

Trace consent policy and explicit action. Compare:

- development skip;
- stored prior consent;
- required explicit approval;
- denial;
- omitted decision;
- policy/storage error.

Deliverable: a decision table whose final column is “may call
`finishAuthorize`.”

## Day 3 morning: transaction ownership

Read design 08, Fosite's saved `Transactional` API, and `sqlstore.go`.

Draw two transactions:

- authorization artifacts plus interaction terminal state;
- code invalidation plus access/refresh creation.

Then draw refresh rotation separately, including conditional update and later
reuse revocation.

Run one before-mutation and one after-mutation failpoint from each matrix. Predict
row counts before running.

Deliverable: a mutation ledger with transaction coordinator, context capability,
SQL helper, failpoint, rollback oracle, event oracle, and retry behavior.

## Day 3 afternoon: concurrency and recovery

Read the two Porcupine tests. Define the sequential model before reading the test
assertions. Explain call time, return time, real-time precedence, and legal
linearization.

Run the tests repeatedly. Explain why refresh final state can be revoked despite
one successful rotation.

Then study migration checksums, online backup, manifest verification, offline
restore, rollback path, doctor, and future-schema refusal.

Deliverable: one page distinguishing atomicity, durability, isolation,
linearizability, and recovery.

## Day 4 morning: static and generated assurance

Read design 09 and every analyzer declaration. Pick four analyzers:

- one public API rule;
- one data/provenance rule;
- one lifecycle rule;
- one error-handling rule.

Read their fixtures. State false positives and false negatives. Run the
multichecker over the repository.

Then compare one example regression, Rapid property, fuzz target, and
metamorphic test. Explain each oracle.

Deliverable: a claim-to-tool matrix with no repeated generic claim.

## Day 4 afternoon: runtime verification and Goja

Read `internal/securitytrace`, its tests, `pkg/verifyplan`, the Goja compiler,
module, and strict scenario driver.

Enumerate every JavaScript-reachable capability. The correct live-capability
count is zero: the module constructs data. Explain source hash, deadline, source
size, output limits, ambient module rejection, native driver, and native
assertion registry.

Run the forced-login verification plan and ambient-module/infinite-loop tests.

Deliverable: an authority diagram showing where JavaScript stops and native Go
begins.

## Day 5 morning: production host

Read design 10, `serve_production.go`, `embeddedidp.Options.Validate`, readiness,
audit, proxy resolver, maintenance, and the production-host README.

Build a production configuration checklist. For every control, name:

- configuration source;
- validation site;
- runtime owner;
- readiness signal;
- failure response;
- operational evidence.

Run the release drills in a disposable environment and inspect their output.

## Day 5 afternoon: release decision

Read reference 06, release workflow files, and the incident runbook. Reconstruct
the exact candidate hash and local gate matrix. Identify which rows remain open
and why no local test can close them.

Deliverable: a written recommendation with three sections:

1. supported local claims;
2. missing external/human evidence;
3. current decision and next authorized action.

The correct current decision is NOT APPROVED.

## Package atlas

### `internal/fositeadapter`

Purpose: integrate Fosite protocol machinery with tiny-idp policy, browser,
store, audit, and security evidence.

Start with `provider.go`. Then read `interaction.go`, `session.go`, `consent.go`,
`ratelimit.go`, `csrf.go`, and `sqlstore.go`. Read tests by property rather than
alphabetically.

Change risk: highest. Most protocol/state changes cross multiple requests and
Fosite lifecycle calls.

### `internal/authn`

Purpose: password credential verification, typed outcomes, lockout, password
work admission, and audit reasons.

Change risk: high. CPU/memory cost, account enumeration, timing, lockout, and
lifecycle flags interact.

### `internal/securitytrace`

Purpose: secret-free versioned facts and native temporal verdicts.

Change risk: schema compatibility, missing instrumentation, event timing, and
false confidence from incomplete traces.

### `internal/store/memory`

Purpose: development/test implementation of public store contracts.

Change risk: copy isolation and semantic parity. It is not production durability
evidence.

### `pkg/sqlitestore`

Purpose: durable supported store, migrations, atomic operations, backup, restore,
maintenance, and operational reporting.

Change risk: capability duplication/loss, WAL assumptions, schema compatibility,
filesystem publication, and concurrency.

### `internal/cmds`

Purpose: Glazed CLI commands for administration, development serve, and
production host.

Change risk: secret input, lifecycle ownership, post-commit ambiguity, audit,
and unsafe defaults.

### `pkg/verifyplan` and `internal/gojaverify`

Purpose: data-only programmable verification and isolated compilation.

Change risk: capability expansion, resource limits, schema/version drift, and
script-defined verdict authority.

## Symbol atlas: construction and health

### `embeddedidp.Options.Validate`

Read this before adding any public option. A field is incomplete until validation,
runtime use, tests, help, and readiness implications are defined.

### `embeddedidp.New`

This is the public construction and initial maintenance boundary. It converts
public contracts into internal adapter options without exporting Fosite.

### `Provider.Readiness`

This is an operational aggregation, not a protocol endpoint test. Each component
needs a reason and owner.

### `Provider.Close`

This preserves lifecycle extensibility and host ownership. Do not hide process
termination inside it.

## Symbol atlas: authorization

### `Provider.beginAuthorize`

Review validation-before-credentials, session state, strict `max_age`, prompt
semantics, required actions, interaction persistence, and rendering.

### `Provider.resumeAuthorize`

Review CSRF, opaque lookup, canonical reconstruction, mutable-state revalidation,
typed authentication, explicit consent, atomic terminal state, and issuance.

### `canonicalAuthorizeForm`

Review every supported parameter and equivalence rule. Never add raw POST
authority here.

### `createInteraction`

Review entropy, hash storage, browser/session binding, client generation, TTL,
copy isolation, and creation event.

### `finishAuthorize`

Review granted scopes/audience, OIDC claims, transaction ownership, event timing,
and response exposure.

## Symbol atlas: tokens

### `sqlFositeStore.BeginTX`

Review nested transaction rejection, hook timing, and derived context.

### `tokenExec`

Review fail-closed transaction requirement and pre/post hook names.

### `RotateRefreshToken`

Review exact signature conditional update, affected rows, old access removal,
and serialization error mapping.

### `persistRequester` / `restoreRequester`

Review every security binding that must survive restart. Treat persistence shape
as a compatibility/security format.

## Symbol atlas: evidence

### `securitytrace.Event`

Review version, secret exclusion, partition identifiers, obligations, outcome,
and grant type.

### `Monitor.Observe`

Review unsupported versions, creation, prerequisites, terminal uniqueness, and
artifact ordering. Identify event kinds intentionally outside its partition.

### `interactionReferenceState.Apply`

Review model simplicity and correspondence with store errors.

### `strictScenarioDriver.Execute`

Review strict decoding, native state ownership, action allowlist, and observation
redaction.

## Test atlas by invariant

### Fresh authentication

Read forced prompt login, expired max age, malformed max age, prompt none, and
injected-clock expiry together. They share temporal semantics but different
inputs.

### Continuation authority

Read form field absence, state mutation, duplicate parameters, independent tabs,
sequential replay, concurrent replay, and generation mutation together.

### Consent

Read explicit approve, deny, omitted decision, stored skip, prompt none, displayed
client/scopes, and persistence error together.

### Token lifecycle

Read successful code flow, restart, code failpoints, refresh failpoints, reuse,
secret rotation, and linearizability together.

### UserInfo

Read GET/POST success, query/form/mixed transport, duplicate Authorization,
methods, cache, challenge, disabled user, and scope claims together.

### Operations

Read production validation, filesystem mode, backup/restore, migration failure,
checksum mismatch, future schema, maintenance, readiness, key rotation, audit,
and runtime load together.

## Source packet reading guide

### Standards first

Read RFC 6749, RFC 6750, RFC 9700, OIDC Core, and NIST 800-63B-4 for normative
language. Record exact section before calling a requirement normative.

### Formal protocol work

Read the OAuth and OIDC formal analyses for authorization, authentication, and
session-integrity definitions. Focus on attacker model and cross-message
properties rather than proof mechanics on first pass.

### Static analysis

Read IFDS, CodeQL dataflow, Go analysis docs, and Semgrep rules. Compare formal
interprocedural reachability with the project's narrow local analyzers.

### Stateful testing

Read model-based security testing, coverage-guided property testing, stateful
greybox fuzzing, protocol-state TLS fuzzing, and metamorphic cybersecurity. Map
each to current or planned harness capability.

### Concurrency and faults

Read CHESS, Herlihy/Wing linearizability context, faster Porcupine checking, and
lineage-driven fault injection. Compare abstract operation histories with SQL
failpoint dependencies.

### Runtime verification

Read the introductory chapter, brief account, monitoring-oriented programming,
security automata, typestate, Daikon, and static verification from dynamic
invariants. Distinguish enforcement, monitoring, and candidate discovery.

## Vocabulary the intern must use precisely

### Authentication

Evidence that a subject completed an accepted authentication mechanism under a
policy at a time.

### Authorization

Permission for a client/action/scope, not merely subject identity.

### Session

Server-recognized continuity state with authentication time, expiry, and
revocation. Not automatically fresh authentication.

### Consent

User or policy authorization of a client and scope set. Not implied by login.

### Capability

A value or reference whose possession enables an operation, such as code, token,
session handle, transaction context, or live host object.

### Invariant

A property required across all states or transitions in its scope.

### Obligation

A required future event that must occur before a later transition becomes legal.

### Linearization point

The instant within an operation interval at which it takes effect in a legal
sequential history.

### Fail closed

Reject or stop progress when required security evidence/control is absent or
invalid. It does not mean convert every availability fault into denial without
considering retry and ambiguity.

### Idempotent

Repeating an operation has the same effect as one application. One-time consume
is not idempotent success; later calls return a terminal error.

### Atomic

All participating mutations commit or none do. Always name participating
mutations.

### Durable

Committed state survives according to database/filesystem failure assumptions.

### Evidence

An observation connected to a claim and scope. Raw output without an oracle or
identity is data, not complete evidence.

### Provenance

Information linking an artifact to source/build process. A source hash alone is
identity, not trusted authorship.

## Mentor challenge questions

1. Which current type is the authoritative authorization interaction state?
2. Why is CSRF insufficient for protocol continuation?
3. Which event proves post-interaction authentication?
4. Where is current client state revalidated?
5. Which operation atomically decides one terminal outcome?
6. Who owns the authorization artifact transaction?
7. Who owns the token transaction?
8. What is refresh rotation's linearization point?
9. Why can the winning refresh response be revoked?
10. Which analyzer guards browser continuation?
11. What is its main false negative?
12. What does a Rapid shrink produce?
13. What does the monitor miss if instrumentation is absent?
14. Why may JavaScript select but not implement an assertion?
15. Which production checks cannot live in the library?
16. Why does proxy trust start at the immediate peer?
17. Which readiness check detects stopped maintenance?
18. Why is online backup required in WAL mode?
19. What does hosted OIDF not prove?
20. Who may approve release?

## Assessment rubric

### Protocol model: insufficient

The intern lists endpoints and parameters but cannot identify principals,
bindings, or temporal obligations.

### Protocol model: ready

The intern traces one success and one adversarial history, distinguishes all
correlation values, and connects them to validated/server-owned state.

### State reasoning: insufficient

The intern proposes handler booleans or status assertions without a durable
state transition.

### State reasoning: ready

The intern defines predecessor state, obligations, linearization point, terminal
behavior, replay, expiry, and emitted event.

### Persistence reasoning: insufficient

The intern says “inside a transaction” without listing the mutation set or
coordinator.

### Persistence reasoning: ready

The intern enumerates rows, context capability, failpoints, rollback oracle,
retry behavior, and post-commit ambiguity.

### Assurance reasoning: insufficient

The intern treats green tests, coverage, lint, or conformance as universal proof.

### Assurance reasoning: ready

The intern states a bounded claim, oracle, observed representation, blind spot,
and complementary method.

### Production reasoning: insufficient

The intern assumes a correct handler implies a secure deployment.

### Production reasoning: ready

The intern covers host, proxy, secrets, audit, readiness, recovery, exact
artifact, external evidence, review, and owner authority.

## First contribution constraints

The first intern change should be narrow, reversible, and supported by existing
models. Good candidates include:

- improve one analyzer fixture and precision note;
- add one negative UserInfo transport regression;
- add one monitor diagnostic test;
- add one strict-driver observation with no new live capability;
- improve one readiness reason;
- add one recovery drill assertion;
- document one unsupported authorization parameter precisely.

Poor first assignments include:

- new grant type;
- new authentication method;
- token format change;
- distributed storage;
- dynamic policy capability;
- signing-key architecture rewrite;
- release approval.

## Review template for the first change

```text
Problem:
Normative or local requirement:
Threat/counterexample:
Authoritative state:
Current symbol path:
Proposed transition:
Persistence/concurrency impact:
Audit/security-event impact:
Tests and analysis:
Evidence limitations:
Rollback/compatibility:
Open questions:
```

No backwards-compatibility adapter should be introduced for unpublished
pre-release behavior without explicit product need. State behavior should be
changed directly with tests and docs.

## Completion checklist

- [ ] Baseline and release decision recorded.
- [ ] Successful Authorization Code + PKCE traced.
- [ ] Public package ownership diagram completed.
- [ ] Forced-login and consent histories drawn.
- [ ] Interaction fields explained.
- [ ] Authorization and token transactions drawn.
- [ ] Failpoint predictions match tests.
- [ ] Porcupine models explained.
- [ ] Backup/restore boundary explained.
- [ ] Four analyzers reviewed with precision limits.
- [ ] Example/property/fuzz/metamorphic oracles compared.
- [ ] Security monitor observability limit stated.
- [ ] Goja reachable authority enumerated.
- [ ] Production configuration/control table completed.
- [ ] Exact-candidate evidence interpreted.
- [ ] Current NOT APPROVED decision defended.
- [ ] First-change review note accepted by mentor.

## Curriculum maintenance

After each intern cohort, record:

- sections that required verbal explanation;
- commands that no longer match the tree;
- symbols that moved;
- exercises with ambiguous expected results;
- incorrect assumptions repeated by multiple readers;
- missing research prerequisites;
- first-change defects caught or missed by the curriculum.

Update the textbook from observed onboarding outcomes. Do not expand breadth
without a learning failure or new system feature that justifies it.

## Provenance map by diary interval

| Diary interval | Knowledge captured | Curriculum destination |
|---|---|---|
| Steps 5–6 | dependencies/toolchain/release baseline | Day 0, assurance, production |
| Steps 7–10 | public API and external consumer | Day 1 architecture |
| Steps 11–12 | atomic store, migrations, backup/restore | Day 3 durable state |
| Step 13 | passwords, bounded work, abuse controls | Day 2 auth, Day 5 host |
| Step 14 | audit, keys, readiness, maintenance | Day 5 production |
| Step 15 | host, CI, evidence, incident response | Day 5 release |
| Step 17 | protocol findings and threat paths | Days 1–2 protocol/state |
| Step 18 | papers and assurance architecture | Day 4 tools/research |
| Steps 19–22 | interaction repair and analyzer precision | Days 2 and 4 |
| Step 23 | token transactions, models, monitor | Days 3–4 |
| Step 24 | analyzers, fuzz feeds, Goja boundary | Day 4 |
| Step 25 | strict driver and metamorphic relation | Day 4 labs |
| Step 26 | exact candidate and missing gates | Day 5 release |

## Key source-to-chapter routes

The formal OAuth/OIDC papers route to protocol and temporal chapters. IFDS and
Go analysis route to assurance. Model-based testing, stateful fuzzing, and
metamorphic testing route to state and assurance. CHESS and linearizability
route to durable/concurrent state. Fault injection routes to transactions and
recovery. Runtime verification, security automata, typestate, monitoring-oriented
programming, and Daikon route to temporal monitoring and assurance. NIST, RFC
9700 proxy guidance, OIDF suite docs, Sigstore, CI actions, and release evidence
route to production security.

The route is not one source to one implementation. A paper supplies a concept;
the diary records the local inference; code and tests supply observed behavior;
the chapter states the remaining gap.

## Key points

- Protocol correctness is a property of message sequences and state transitions,
  not a collection of independently valid handlers.
- Browser, relying-party, host, provider, store, and script inputs have different
  authority.
- Security-sensitive mutations require explicit atomic and concurrent semantics.
- Tests and tools support bounded claim classes; no result should be generalized
  beyond its evidence boundary.
- Production approval includes operational and human authority that code cannot
  self-attest.

## References

- `design-doc/01-production-embedding-api-and-release-implementation-guide.md`
- `design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`
- `design-doc/03-research-foundations-for-identity-protocol-invariants-atomicity-and-runtime-verification.md`
- `design-doc/04-programmable-verification-plans-research-boundary-and-implementation.md`
- `reference/01-implementation-diary.md`
- `reference/04-authorization-interaction-and-protocol-robustness-review.md`
- `reference/05-authorization-interaction-hardening-implementation-ledger.md`
- `reference/06-exact-candidate-assurance-evidence-5bb4dae.md`
