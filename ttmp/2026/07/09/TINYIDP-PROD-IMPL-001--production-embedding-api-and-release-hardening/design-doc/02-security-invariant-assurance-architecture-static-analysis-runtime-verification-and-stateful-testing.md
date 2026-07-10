---
Title: Security invariant assurance architecture static analysis runtime verification and stateful testing
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Authorization, token, UserInfo, continuation, consent, and protocol-artifact control paths analyzed by the assurance design
    - Path: repo://internal/fositeadapter/session.go
      Note: Browser-session classification and max_age policy boundary
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Target location for typed atomic interaction and protocol lifecycle contracts
    - Path: repo://pkg/sqlitestore/store.go
      Note: Durable implementation and failpoint/atomicity test target
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Existing custom Go analysis multichecker to extend
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Identity scripting TCB and runtime architecture evaluated for verification authoring
ExternalSources:
    - https://arxiv.org/abs/1601.01229
    - https://arxiv.org/abs/1704.08539
    - https://www.usenix.org/conference/usenixsecurity15/technical-sessions/presentation/de-ruiter
    - https://www.usenix.org/conference/usenixsecurity22/presentation/ba
    - https://ecommons.cornell.edu/items/5a936aa1-8a4f-41df-bc17-f2db479cf33e
Summary: A research-backed assurance architecture for tiny-idp using custom Go analysis, typed protocol models, stateful fuzzing, controlled concurrency, fault injection, structured traces, runtime monitors, and an isolated Goja verification plane.
LastUpdated: 2026-07-10T12:00:00-04:00
WhatFor: Designing and implementing repeatable evidence that tiny-idp preserves authentication, authorization, session-integrity, storage, audit, and scripting invariants before production release.
WhenToUse: Read before changing authorization interactions, adding security analyzers or fuzzers, instrumenting protocol flows, or extending the Goja scripting design with validation plugins.
---







# Security invariant assurance architecture: static analysis, runtime verification, and stateful testing

## Executive summary

Tiny-idp needs an assurance system that tests security properties across complete
request histories, not only individual handler calls. The forced-reauthentication
bug is a concrete example. The GET request correctly decides that a password is
required, but the POST request reconstructs a weaker request from browser fields
and accepts an existing session. Each local operation looks plausible. The
history violates the intended property: an authorization code was issued without
the required authentication transition.

The recommended system has six cooperating layers:

1. A typed Go reference model defines authorization-interaction states, actions,
   required actions, and terminal outcomes.
2. Repository-specific `go/analysis` analyzers reject code shapes that commonly
   weaken the model, including fail-open security parsing, mutable browser-owned
   continuations, unstable limiter keys, implicit bearer-token transports, direct
   wall-clock reads, ignored security errors, and partial protocol mutations.
3. Example, table-driven, property-based, metamorphic, and native fuzz tests drive
   the same model through normal and adversarial request sequences.
4. Controlled clocks, scheduling points, store failpoints, and concurrent-history
   tests make expiry, replay, cancellation, and partial-commit behavior
   reproducible.
5. Structured security events form a versioned trace. Offline parametric monitors
   evaluate temporal invariants per interaction, authorization code, session,
   refresh-token family, client, and activation generation.
6. An optional Goja verification plane compiles human-authored scenarios and pure
   assertions into a Go-owned `VerificationPlan`. It is separate from the
   production policy runtime and cannot authorize requests, suppress failures, or
   resolve production secrets.

This is not a proposal to prove the entire implementation correct. It is a
proposal to make important properties explicit, make violations observable, and
systematically search the state space in which implementation defects occur.
Formal OAuth and OpenID Connect work supplies the property vocabulary and attacker
model. Security-automata and runtime-verification research explains which finite
bad prefixes a monitor can reject or report. Typestate, IFDS-style dataflow,
model-based testing, stateful greybox fuzzing, controlled concurrency, fault
injection, and dynamic invariant mining supply complementary implementation
techniques.

The immediate release conclusion is unchanged: tiny-idp should not be released
until the authorization interaction is server-owned and one-time, forced login
and `max_age` are tested as required transitions, malformed security parameters
fail closed, consent denial uses protocol semantics, token limiting cannot be
sharded by attacker-controlled identity, UserInfo transport is explicit, and
Fosite's multi-handler writes are shown to be atomic or compensating under every
injected failure point.

## 1. Purpose, scope, and reader contract

This guide is written for an intern who can read Go but is new to identity
protocols and program analysis. By the end, the reader should be able to:

- trace an authorization request from browser input to Fosite validation,
  authentication, consent, code creation, and durable protocol state;
- state the security properties without referring to a particular test case;
- choose the correct assurance technique for a property;
- implement and test a repository-specific Go analyzer;
- add model actions, generators, failpoints, and runtime events without exposing
  secrets;
- understand why verification scripts require a separate trust profile; and
- produce release evidence that another engineer can rerun from a clean checkout.

The scope is the strict Fosite-backed provider in `internal/fositeadapter`, its
public embedding boundary in `pkg/embeddedidp`, the stores in `pkg/idpstore` and
`pkg/sqlitestore`, and the proposed Goja identity microkernel. The older mock
engine under `internal/server` remains useful as a scenario oracle and source of
test ideas, but it is not production proof for the strict provider.

This guide does not propose eBPF as an enforcement mechanism, JavaScript as a
replacement for native protocol validation, or audit logs as proof of behavior
that was never instrumented. It also does not treat a passing conformance suite as
evidence for application-specific invariants such as durable audit delivery or
one-time interaction consumption.

## 2. Current system and the defect pattern

### 2.1 Request path

`Provider.Handler` mounts discovery, JWKS, authorization, token, UserInfo, health,
and readiness endpoints in `internal/fositeadapter/provider.go:300-306`. The
authorization handler parses a Fosite request on GET, reads a browser session,
decides whether login and consent are needed, and either finishes or renders an
HTML interaction (`provider.go:340-379`). On POST it parses the browser form,
checks rate limiting and CSRF, asks Fosite to rebuild the authorization request,
optionally authenticates a submitted login, and calls `finishAuthorize`
(`provider.go:380-438`).

The current form continuation is generated by `hidden` at
`provider.go:574-589`. It preserves eight fields. It does not preserve `prompt`,
`max_age`, `claims`, `acr_values`, `id_token_hint`, `login_hint`, `ui_locales`,
`response_mode`, `audience`, `resource`, or extension parameters. More
fundamentally, the browser is asked to carry the protocol continuation. CSRF
proves that a form came from a browser holding a cookie; it does not prove that
the client request, required actions, or validation result are immutable.

### 2.2 Forced reauthentication as a state-machine violation

On GET, an existing session does not bypass `prompt=login`; the handler renders a
login form. On POST, `prompt` has disappeared. If `login` is blank and a browser
session exists, the branch at `provider.go:433-437` does not reject the request.
The handler uses the old session and issues a response at line 438.

The useful property is not "the password field must be non-empty." It is:

```text
For every accepted authorization interaction I:
  if required_actions(I) contains fresh_authentication,
  then I contains a successful authentication event after I was created,
  and the issued ID token's auth_time is the time of that event.
```

The blank-password example, a crafted form, a concurrent browser tab, a replay,
or a changed continuation all violate the same property. A state model exposes
that common cause and avoids accumulating one regression per symptom.

### 2.3 Other observed paths with the same structure

The focused review in
`reference/04-authorization-interaction-and-protocol-robustness-review.md`
documents related issues:

- `sessionSatisfiesMaxAge` returns true on parse failure or a negative value and
  performs overflow-prone duration arithmetic
  (`internal/fositeadapter/session.go:53-61`).
- The GET error branch renders credentials for a broad class of Fosite errors
  whenever `max_age` is present (`provider.go:346-354`).
- Consent rejection is a raw HTTP 403, while the browser session may already have
  been created.
- The token limiter includes untrusted form `client_id` in its only bucket key
  (`provider.go:474-483`).
- UserInfo obtains a token through Fosite's generic `AccessTokenFromRequest`
  helper (`provider.go:501-514`), making transport and method policy implicit.
- Request-object rejection decodes unverified claims to choose error-routing
  values (`provider.go:592-620`).
- Browser-session store failures and absent sessions are collapsed into the same
  boolean result (`internal/fositeadapter/session.go:28-41`).
- Fosite invokes authorization-code, PKCE, and OIDC response handlers in sequence,
  while the current SQL store exposes individually atomic mutations rather than
  one transaction spanning the group.

These examples motivate the architecture: security behavior lives in relations
between parsing, control flow, durable mutations, time, and prior requests.

## 3. What an invariant is

An invariant is a predicate expected to hold over every relevant state or trace.
The word is used at several levels, which must not be confused.

| Level | Example | Appropriate evidence |
|---|---|---|
| Structural | Production Goja factories expose no ambient modules. | Typed static analyzer plus negative integration test |
| State | A consumed interaction cannot become pending again. | Store unit tests and model-based tests |
| Transition | Only `pending -> consumed` can issue a code. | Reference model, stateful properties, runtime monitor |
| Temporal | Required fresh authentication occurs after interaction creation and before issuance. | Trace monitor with timestamps and event order |
| Relational | Changing `state` changes only the returned state, not whether authentication is required. | Metamorphic test |
| Information-flow | Passwords and bearer tokens do not reach logs or script projections. | Static taint analysis, redaction tests, trace scans |
| Atomicity | Code, PKCE, and OIDC session records are all committed or all absent. | Failpoint matrix and storage inspection |
| Concurrency | At most one concurrent consume succeeds. | Parallel history plus linearizability check |
| Liveness | A healthy request eventually completes within its deadline. | Bounded timing tests and operational SLOs |

Security automata are strongest for safety properties: once a finite bad prefix
has occurred, no future event can repair it. Issuing a code before required login,
accepting the same interaction twice, or logging a token are safety violations.
"Every request eventually succeeds" is a liveness property and cannot generally
be established by observing a finite prefix. For production release, convert
operational liveness into bounded safety where possible: "no request remains in
an in-progress state beyond deadline D."

This distinction determines whether a runtime hook may enforce a property. A
monitor placed immediately before issuance can prevent a known bad transition.
An offline monitor can only report it after the fact. A trace with missing events
cannot establish the absence of the transition at all.

## 4. Research foundations and their engineering consequences

### 4.1 Formal OAuth and OpenID Connect analysis

Fett, Küsters, and Schmitz analyze OAuth and OpenID Connect in an expressive web
model containing browsers, windows, cookies, redirects, network attackers,
malicious clients, and multiple simultaneous protocol runs. Their important
contribution for this project is the separation of three property families:

- authorization: an attacker does not obtain access to resources or grants that
  the user did not authorize;
- authentication: a relying party does not accept an attacker as the user; and
- session integrity: the identity and authorization associated with the
  completed session correspond to the user's initiating action.

Tiny-idp cannot reproduce the full proof inside Go tests. It can use the same
discipline: name the principals and sessions, define the attacker-controlled
values, bind every continuation to its initiating request, and test multiple
simultaneous runs. The concurrent-tab case is not UI trivia. It is a session-
integrity test.

### 4.2 Security automata and runtime verification

Schneider's security-automata work characterizes execution monitoring in terms
of prefixes. A monitor must see a correct and complete action stream, maintain
the state needed to classify the prefix, and have control at the point where it
enforces a transition. The concrete consequences are:

- event instrumentation must be complete at security-relevant boundaries;
- the monitor must be isolated from the script or component it checks;
- an offline monitor detects but does not prevent;
- a production guard must run before the irreversible effect; and
- a script callback cannot be allowed to suppress the guard's verdict.

Monitoring-oriented programming and parametric trace monitoring add one useful
idea: instantiate a logical monitor for each parameter tuple. Tiny-idp should
slice traces by opaque interaction ID, authorization-code fingerprint, refresh-
token family, session ID hash, and graph generation. A single global automaton
would conflate independent browser flows.

### 4.3 Typestate

Typestate refines "what operations does this type support?" into "what operations
are permitted in the object's present state?" An interaction handle has one Go
type but multiple semantic states:

```text
pending --satisfy login--> pending
pending --approve consent--> pending
pending --consume--> consumed
pending --expire--> expired
pending --deny--> denied
consumed --consume--> ERROR(replay)
expired  --consume--> ERROR(expired)
denied   --consume--> ERROR(terminal)
```

Go's type system will not encode every persistent state. The store API can still
make the legal transition explicit with `ConsumeInteraction`, rather than
`GetInteraction` followed by `UpdateInteraction`. Static analysis can then flag
mutation sequences that bypass the atomic transition.

### 4.4 Interprocedural dataflow and taint

The Reps–Horwitz–Sagiv IFDS result shows that a useful class of finite,
distributive interprocedural dataflow problems can be reduced to graph
reachability. Modern security query systems use related source-to-sink models.
For this repository the relevant facts include:

- browser-controlled authorization fields;
- validated client and redirect identities;
- password, cookie, token, and code secrets;
- redacted or hashed derivatives;
- durable mutation capabilities; and
- authorization acceptance sinks.

Simple AST matching is preferable when a rule is local and exact. SSA/dataflow
is needed when values pass through helpers or interfaces. CodeQL is appropriate
for cross-package taint experiments. A custom Go analyzer is appropriate when
the rule depends on project types and should run in normal Go CI. Semgrep is a
fast secondary guard for syntax patterns, not the authoritative checker for
typed protocol properties.

### 4.5 Model-based and property-based security testing

Model-based security testing separates the abstract security model from the
concrete driver. A model action such as `SubmitBlankLogin` has a precondition, a
state transition, and an expected observation. The driver translates the action
into HTTP, cookies, form fields, and store inspection.

Property-based testing generates many action sequences and shrinks a failing
sequence to a smaller counterexample. The model must generate semantically valid
requests often enough to reach deep states. Coverage-guided property testing
addresses sparse preconditions by retaining inputs that reach new branches. For
tiny-idp, a structured generator should produce valid client, redirect, scope,
PKCE, and CSRF data first, then mutate one semantic dimension at a time. Random
bytes alone will spend most executions in the first parser error.

### 4.6 Protocol-state and stateful greybox fuzzing

Protocol-state fuzzing demonstrated that learning the state machine of real TLS
implementations exposes unintended transitions, including security-sensitive
messages accepted in the wrong state. Stateful greybox fuzzing adds
instrumentation feedback about state variables and transition sequences.

Tiny-idp has a stronger starting position than a black-box TLS implementation:
we can define the intended model and instrument native state IDs. The fuzzer's
feedback vector should include:

```text
(model_state, required_action_bits, terminal_outcome,
 HTTP_status_class, OAuth_error, store_mutation_bits,
 audit_event_bits, code_coverage_hash)
```

Inputs that discover a new vector enter the corpus. The fuzzer mutates action
sequences, not only bytes.

### 4.7 Controlled concurrency and linearizability

CHESS controls scheduling decisions and records them for deterministic replay.
Go does not provide a drop-in CHESS equivalent, but the design principle applies:
put explicit test scheduling points around load, validate, mutation, and commit;
record the chosen schedule; and reproduce the same schedule after a failure.

`testing/synctest` can provide a fake clock and controlled concurrency bubble for
pure Go components. The race detector detects unsynchronized memory access but
does not prove correct atomic state transitions. Porcupine can check whether a
history of concurrent one-time operations is equivalent to some legal serial
history. Apply it to interaction consumption, refresh-token rotation, lockout
counters, and key activation.

### 4.8 Fault injection and lineage

Random failure injection spends effort on failures that do not influence the
outcome. Lineage-driven fault injection reasons backward from a good result to
the messages and facts that supported it. Tiny-idp can use a smaller, explicit
version: every accepted authorization result records the durable writes and
security decisions in its test trace; the harness injects a failure before and
after each contributing step and verifies that the accepted outcome disappears
or remains complete.

For a Fosite authorization response, the first matrix is:

```text
fail before code write
fail after code write, before PKCE write
fail after PKCE write, before OIDC session write
fail after OIDC write, before transaction commit
fail commit
fail response serialization after commit
```

For every point, inspect code, PKCE, OIDC, audit, and browser output. The desired
storage postcondition is "all or none"; the HTTP postcondition may require a
separate recovery design when commit succeeds but response delivery fails.

### 4.9 Dynamic invariant mining

Daikon infers likely relations from observed values. Nimmer and Ernst combine
dynamic candidates with static verification, which captures the right division
of labor: observed relations are hypotheses, not proofs.

Tiny-idp should start with a small domain-specific miner over redacted trace
fields. It can discover candidates such as:

- `required_login => auth_success_count == 1`;
- `outcome == accepted => consume_count == 1`;
- `oauth_error != "" => code_write_count == 0`;
- `max_age_present => auth_time >= interaction_created_at - max_age`;
- `consent_denied => granted_scope_count == 0`.

Engineers review useful candidates, promote them into named model predicates,
and then seek static or exhaustive bounded evidence. Automatically promoting a
candidate from a limited trace corpus would encode the test suite's blind spots.

## 5. Canonical invariant catalog

### 5.1 Authorization request and interaction

`AUTH-01 Validated continuation`: The interaction references one server-stored,
validated authorization request. Browser fields cannot replace the client,
redirect, response type, PKCE binding, nonce, requested scopes, audience,
resource, or required actions.

`AUTH-02 One-time terminal transition`: Exactly one terminal outcome is possible:
accepted, denied, expired, or rejected. Replays cannot create another outcome.

`AUTH-03 Forced fresh authentication`: If `prompt=login`, expired `max_age`,
step-up, or native policy requires fresh authentication, acceptance requires a
successful authentication after interaction creation.

`AUTH-04 Non-interactive behavior`: `prompt=none` never renders UI. If login or
consent is required, the response is the appropriate OAuth/OIDC error.

`AUTH-05 Strict parameter semantics`: Malformed, negative, overflowing, unknown,
or contradictory security parameters are rejected. Parser error never weakens a
requirement.

`AUTH-06 Request immutability`: POST and resume operations use the stored request
digest. Any browser-supplied request parameters are absent or must match the
stored canonical values exactly.

`AUTH-07 Current-state revalidation`: Resume rechecks mutable server facts such
as client enabled state, exact redirect registration, user disabled state,
session revocation, signing-key readiness, and graph-generation compatibility.

### 5.2 Authentication and session

`SESS-01 Authentication time provenance`: `auth_time` comes from the successful
native authentication event, never from request time or a reused session when
fresh authentication was required.

`SESS-02 Session error classification`: Store unavailable, corrupt record,
expired, revoked, disabled user, and absent cookie remain distinguishable for
control flow and audit, even if the browser receives a uniform response.

`SESS-03 No premature session side effect`: A denied consent or rejected
authorization does not establish a browser session unless an explicit product
decision and test require it.

`SESS-04 Authentication lifecycle enforcement`: Every native authenticator
result field that denotes a required action is either enforced before issuance
or unsupported and rejected. The former `MustChangePassword` bug is one instance.

### 5.3 Consent and authorization

`CONSENT-01 Explicit decision`: Consent approval and denial are typed actions.
Absence of an approval field is not approval.

`CONSENT-02 Bound display`: The form displays and binds the validated client and
requested scopes. A changed request requires a new interaction.

`CONSENT-03 Protocol denial`: Denial produces `access_denied` through the
validated redirect and preserves state when redirecting is safe.

`CONSENT-04 Grant subset`: Granted scopes and audiences are an allowed subset of
client registration, requested values, consent, and policy results.

### 5.4 Protocol artifacts and persistence

`PROTO-01 Artifact preconditions`: A code or token is created only after request
validation, required authentication, consent, policy approval, and readiness.

`PROTO-02 Group atomicity`: Authorization-code, PKCE, and OIDC session state is
complete or absent. Refresh rotation invalidates the old member and creates the
new family member atomically.

`PROTO-03 Single use`: Authorization codes, refresh rotation members, device
codes, and interaction handles obey their specified one-time semantics under
concurrency.

`PROTO-04 Error routing`: Unvalidated redirect or request-object claims never
select a redirect target. Safe errors remain local.

`PROTO-05 Explicit bearer transport`: UserInfo accepts only documented methods
and bearer locations, rejects query tokens, sets cache and challenge headers, and
does not echo credentials.

### 5.5 Abuse resistance

`ABUSE-01 Stable limiter identity`: An attacker cannot create fresh buckets by
changing an unauthenticated form, header, port, case, or whitespace value.

`ABUSE-02 Layered buckets`: Expensive operations use global/address/client/
account dimensions where applicable; denial in any required dimension denies the
operation.

`ABUSE-03 Cost before proof`: The service bounds parsing, password work, script
execution, storage work, response size, and audit pressure before performing
attacker-amplifiable work.

### 5.6 Audit and scripting

`AUDIT-01 Complete terminal evidence`: Every security interaction has one
terminal audit outcome, including infrastructure failure and replay rejection.

`AUDIT-02 Secret exclusion`: Passwords, cookies, codes, bearer tokens, private
keys, raw CSRF tokens, and unverified JWT bodies never enter audit or traces.

`SCRIPT-01 Native TCB`: JavaScript cannot select redirect targets, sign tokens,
verify passwords, mutate raw protocol state, or override protected claims.

`SCRIPT-02 Least authority`: Compiler, production policy, and verification
runtime profiles expose separate explicit modules and capabilities.

`SCRIPT-03 Verification non-authority`: A validation plugin may generate actions
and assertions. It cannot make a failed invariant pass, authorize a production
request, or suppress a native release gate.

## 6. Target assurance architecture

```text
                         specification plane

 RFC/OIDC + threat model ----> invariant catalog ----> typed Go model
                                      |                     |
                                      |                     v
                                      |              generators/shrinkers
                                      v                     |
 source code ---> go/analysis + CodeQL/Semgrep              |
      |                 |                                   |
      |                 +------------ diagnostics ----------+
      v                                                     v
 instrumented tiny-idp <---- scenario driver/fuzzer ---- observations
      |              ^             |                        |
      |              |             +-- fake clock/scheduler |
      |              |             +-- store failpoints     |
      |              |                                      v
      |              +---------------- counterexample -- trace monitors
      |                                                     |
      +-- native guards before irreversible effects         v
                                                    release evidence

       optional authoring plane: tinyidp/verify JS -> VerificationPlan
       (compile only; Go owns execution, verdicts, secrets, and effects)
```

The typed model and event schema are the shared contracts. Static analyzers know
the source-level operations. Drivers know how model actions map to HTTP. Runtime
monitors know how observations map back to properties. Goja supplies an optional
authoring syntax, not a second source of truth.

## 7. Static analysis program

### 7.1 Existing foundation

The completed production-review ticket already includes
`ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go`.
It uses `analysis.Analyzer`, `inspect.Analyzer`, typed object resolution,
`analysistest`, and `multichecker`. Existing rules cover:

- exported public APIs depending on `internal` types;
- ignored `crypto/rand.Read` errors;
- package-level `http.ListenAndServe` without explicit server lifecycle;
- silent no-op audit and allow-all limiter defaults;
- raw `RemoteAddr` in rate-limit keys;
- public config fields that are never read;
- ignored audit-delivery errors;
- multi-mutation functions without visible transaction boundaries; and
- unsafe database backup copying.

Move this tool into a stable top-level internal tooling package only when the
project decides it is a maintained CI product. Until then, extend it in place or
copy it into this ticket with provenance; do not create a second incompatible
diagnostic vocabulary.

### 7.2 Analyzer: `tinyidpinteractioncontinuation`

Purpose: find authorization handlers that validate on one request and reconstruct
security state from a later browser form.

First implementation:

1. Identify functions taking `*http.Request` that branch on `r.Method`.
2. Record calls to Fosite `NewAuthorizeRequest` in GET and POST branches.
3. Record calls to interaction renderers and helpers emitting hidden fields.
4. Report a POST branch that calls `NewAuthorizeRequest` directly without a call
   to an approved `LoadInteraction`/`ConsumeInteraction` API.
5. Report browser fields used to derive required actions after a stored
   interaction has been loaded.

This is intentionally repository-specific. The diagnostic should name the
required API, not claim that every two-request form is insecure.

### 7.3 Analyzer: `tinyidpsecurityparse`

Purpose: reject parse errors that return an allow/satisfied/default-zero result
for security parameters.

Start with syntactic and control-flow patterns around `strconv.ParseInt`,
`ParseUint`, `ParseBool`, duration parsers, URL parsing, and JWT claim conversion.
Use `ctrlflow.Analyzer` or SSA when the result is returned through a helper.

```text
source fact: parse operation for named security parameter
bad sink: true / allow / satisfied / empty requirement / zero age
exception: explicit optional parameter absent before parse
required diagnostic: parse failure weakens <parameter>; reject or return error
```

The analyzer should catch `sessionSatisfiesMaxAge` returning true on `err != nil`
and on negative input. Unit fixtures must include safe rejection, optional absence,
and unrelated numeric parsing to control false positives.

### 7.4 Analyzer: `tinyidplimiteridentity`

Purpose: find attacker-controlled values that shard the only limiter key.

Sources include `Form.Get`, `PostForm.Get`, `URL.Query().Get`, selected headers,
raw `RemoteAddr`, and unauthenticated Basic username. Sinks are `RateLimiter.Allow`
keys. Sanitizers are not generic hashing; hashing attacker-controlled arbitrary
identity still permits unbounded buckets. Approved transformations normalize a
trusted proxy address or use a validated client identity after authentication.

The analyzer needs a small intra-procedural taint lattice first:

```text
Unknown < AttackerControlled < ValidatedIdentity
Unknown < StableNetworkIdentity
```

Report when every limiter dimension for an expensive pre-authentication operation
contains `AttackerControlled`, or when no stable/global dimension is present.
The current token key at `provider.go:479` is the initial regression.

### 7.5 Analyzer: `tinyidpbearertransport`

Purpose: make bearer-token extraction explicit at resource endpoints.

Report use of generic helpers such as `fosite.AccessTokenFromRequest` in UserInfo
unless the enclosing path has already enforced the method and rejected query
transport. A safer API is a tiny-idp-owned `BearerFromAuthorizationHeader` that
returns typed errors and owns the `WWW-Authenticate`, cache, and method contract.

### 7.6 Analyzer: `tinyidpclocksource`

Purpose: require deterministic time in authentication, session, interaction,
token, maintenance, and script-policy decisions.

Report direct `time.Now` in selected packages except approved constructors or
clock adapters. This is a design constraint, not a universal Go style rule. The
injected `Clock` interface should return one request-consistent instant where
the property requires it.

### 7.7 Analyzer: `tinyidpsecurityerror`

Purpose: find errors collapsed into absence or ignored at security boundaries.

Use typed callees and named result flows. Initial targets are store reads for
sessions, clients, users, keys, interactions, consents, and protocol artifacts;
audit emission; random generation; policy execution; and transaction commit.
Diagnostics should distinguish an intentionally uniform external response from
an internally discarded cause. Returning the same HTTP message is allowed;
returning `false` for both missing and database unavailable is not.

### 7.8 Analyzer: `tinyidpprotocollifecycle`

Purpose: flag protocol operations whose grouped state mutations lack a transaction
or compensation contract.

Extend the existing atomicity analyzer with named lifecycle groups:

```text
authorization response: code + PKCE + OIDC session
refresh rotation: revoke old + create replacement + family metadata
key activation: insert key + deactivate previous + activation audit
interaction consume: mark consumed + create terminal artifact
```

Static analysis cannot prove database atomicity through arbitrary interfaces.
The rule should require one of:

- an approved atomic store method;
- an explicit transaction-scoped annotation that is tested by failpoints; or
- a documented compensation method with idempotence tests.

### 7.9 Scripting analyzers

When the Goja layer exists, add:

- `tinyidpambientmodule`: production/compile/verify runtime builders must disable
  implicit registries and use an allowlisted module set;
- `tinyidpprotectedclaims`: script output merge paths cannot assign `iss`, `sub`,
  `aud`, `exp`, `iat`, `auth_time`, `nonce`, `azp`, signature headers, or key ID;
- `tinyidpruntimeowner`: every Goja runtime is called through its single owner;
- `tinyidpverifycapability`: test-only capability providers are unreachable from
  production build profiles and provider registries.

### 7.10 Analyzer engineering contract

Every analyzer ships with:

- positive and negative `analysistest` fixtures;
- stable diagnostic text and a rule ID;
- a scope statement and known false-negative list;
- no automatic source rewrite for security-sensitive behavior;
- a repository baseline file only if each existing finding has an owner and
  expiry; and
- a CI command that analyzes production packages and the scripting packages.

Use `inspect` for local syntax, `buildssa` for value flow, `ctrlflow` for branch
reachability, and analysis facts only when cross-package summaries materially
improve precision. A complex analyzer is not automatically better. Prefer the
smallest sound-enough rule that catches a demonstrated bug with acceptable noise.

## 8. Reference model and test harness

### 8.1 Pure model

The model must not import Fosite, HTTP, SQL, or Goja.

```go
type InteractionState string
const (
    Pending  InteractionState = "pending"
    Accepted InteractionState = "accepted"
    Denied   InteractionState = "denied"
    Expired  InteractionState = "expired"
    Rejected InteractionState = "rejected"
)

type RequiredAction uint32
const (
    RequireLogin RequiredAction = 1 << iota
    RequireFreshLogin
    RequireConsent
    RequireStepUp
)

type Model struct {
    State       InteractionState
    Required    RequiredAction
    CreatedAt   time.Time
    AuthTime    *time.Time
    AuthEvents  int
    Consent     ConsentState
    Consumed    bool
    RequestHash [32]byte
}

type Action interface {
    Precondition(Model) bool
    Apply(Model) Model
    Expected() ObservationPredicate
}
```

The concrete driver owns an `http.Client`, cookie jars or explicit cookies,
registered clients, users, fake clock, store inspector, audit collector, and
failpoint controller. After every action it compares HTTP and durable observations
with the model.

### 8.2 Action vocabulary

Initial actions:

- begin authorization with parameter vector;
- advance/rewind fake time;
- submit correct, wrong, blank, or mismatched login;
- approve, deny, omit, or replay consent;
- mutate one continuation field;
- revoke session, disable user, disable client, or rotate key between steps;
- consume the same interaction sequentially or concurrently;
- exchange code with correct, wrong, absent, or replayed verifier;
- rotate refresh token concurrently;
- call UserInfo with header, form, query, wrong method, expired token, or wrong
  token type;
- inject a named store/audit/policy failure before a transition; and
- reload a graph generation while an interaction is pending.

### 8.3 Example-based tests

Write the smallest failing regressions first. They establish intended behavior
and make later generators easier to debug:

1. valid session + `prompt=login` + blank POST does not issue a code;
2. valid session + expired `max_age` + blank POST does not issue a code;
3. invalid/negative/overflowing `max_age` is rejected and never renders login;
4. `prompt=none` with required login/consent never renders UI;
5. consent denial returns `access_denied` and produces no code;
6. changed client/redirect/scope/PKCE/nonce on resume is rejected;
7. two consumes yield exactly one success;
8. a token limiter cannot be reset by changing form `client_id`; and
9. UserInfo query bearer tokens are rejected.

### 8.4 Property-based tests

Use Rapid or an equivalent typed generator. Generate a valid base request, then
derive mutations. Properties include:

```text
accepted => exactly_one_terminal && exactly_one_consume
required_fresh_login => auth_success_after_creation
denied_or_rejected => no_code && no_pkce && no_oidc_session
request_mutation => rejection_or_same_canonical_digest
advance_time(delta >= max_age) => cannot_reuse_old_auth
replay(any_terminal_handle) => never_accept
```

Persist the random seed and shrunk action list with every failure. A counterexample
must be runnable as a named Go test without the generator.

### 8.5 Metamorphic tests

Metamorphic relations test controlled changes when a full oracle is expensive:

- changing `state` changes only returned state and request digest, not login or
  consent requirements;
- reordering scopes does not change the set decision;
- changing case/whitespace in an account identifier does not create a fresh
  limiter identity;
- adding an unknown security parameter does not weaken a known requirement;
- advancing time cannot make an expired session become valid;
- replacing a valid PKCE verifier with another equal-length verifier cannot
  succeed; and
- retrying an infrastructure failure cannot duplicate a terminal artifact.

### 8.6 Native fuzz harnesses

Maintain separate fuzz targets for:

1. pure parsers and canonicalizers;
2. parameter-vector validation;
3. interaction action sequences;
4. token transport and error response formatting;
5. audit-event serialization/redaction;
6. Goja graph and verification-plan decoding; and
7. trace-monitor event sequences.

Run short fuzz budgets in PR CI and longer corpus campaigns on a schedule or
OSS-Fuzz. Do not fuzz a real production database or network endpoint. Use an
in-process provider and disposable store.

## 9. Determinism, concurrency, and failpoints

### 9.1 Clock

Introduce one `Clock` contract at the public/native policy boundary:

```go
type Clock interface { Now() time.Time }
```

Pass a captured request instant into operations that must agree. Tests use a fake
clock; pure concurrency components may use `testing/synctest`. Avoid environment
variables or hidden global overrides.

### 9.2 Scheduling points

Test-only scheduling hooks belong behind an injected interface, not sleeps:

```go
type SchedulerProbe interface {
    Reach(ctx context.Context, point Point, objectID string) error
}
```

Points include interaction loaded, requirements checked, before consume, after
consume, before artifact write, before commit, and after commit. The no-op
production implementation has no goroutine or channel behavior. The test
implementation blocks until the driver releases a named point and records the
schedule.

### 9.3 Linearizability history

For a one-time consume operation record:

```go
type ConsumeCall struct {
    Client  int
    Start   int64
    End     int64
    Handle  string // test-only opaque ID, never a real secret
    Outcome string
}
```

The sequential specification permits one success for each pending handle and
replay errors afterward. Porcupine checks whether the concurrent history admits
that serial interpretation. Repeat with `-race`, shuffled tests, multiple CPU
counts, and injected store latency.

### 9.4 Failpoint contract

Prefer explicit adapter failpoints at security boundaries. `gofail` is useful
when source-level failpoint rewriting is acceptable; a typed wrapper is often
clearer for stores and audit sinks.

Every failpoint has:

- stable name and lifecycle operation;
- `before` or `after` position;
- single-fire/count/probability mode;
- injected error class;
- trace event; and
- cleanup assertion.

The release matrix must enumerate all mutation boundaries. "We tested one SQL
error" is not evidence for group atomicity.

## 10. Instrumentation and runtime verification

### 10.1 Audit, trace, metrics, and logs are different products

Audit records answer who attempted or completed a security action and must have
defined delivery semantics. Verification traces expose transition structure for
tests and offline analysis. Metrics aggregate counts and latency. Debug logs
explain local implementation behavior. Using one unbounded event body for all
four creates secret and cardinality risk.

### 10.2 Versioned security event

```go
type SecurityEvent struct {
    SchemaVersion   string
    Sequence        uint64
    Time            time.Time
    TraceID         string
    InteractionID   string // opaque or keyed fingerprint
    SessionID       string // keyed fingerprint
    ArtifactID      string // keyed fingerprint
    GenerationHash  string
    RequestDigest   string
    Name            EventName
    ClientID        string
    SubjectID       string // stable internal/audit ID according to policy
    RequiredActions uint32
    StateBefore     string
    StateAfter      string
    Outcome         string
    Reason          string
    Fields          map[string]string // schema-checked bounded keys
}
```

Event names start with:

```text
interaction.created
interaction.loaded
interaction.requirements_evaluated
authentication.attempted
authentication.succeeded
authentication.failed
consent.approved
consent.denied
interaction.consume_attempted
interaction.consumed
interaction.replay_rejected
interaction.expired
protocol_state.write_started
protocol_state.write_committed
protocol_state.write_rolled_back
authorization.accepted
authorization.rejected
```

Never include raw password, cookie, CSRF token, authorization code, access or
refresh token, private key, request object, full form, or full JWT. Test traces
may use synthetic IDs, but production code should still enforce the schema.

### 10.3 Parametric monitor

```text
for each event e:
  validate schema and monotonically increasing sequence
  key = e.interaction_id
  m = monitors.getOrCreate(key)
  m.transition(e)

  if e.name == authorization.accepted:
    assert m.created == 1
    assert m.consumed == 1
    assert m.terminal == 0
    assert e.request_digest == m.original_request_digest
    if m.required_fresh_login:
      assert m.auth_success_after_created == 1
    if m.required_consent:
      assert m.consent_approved == 1

  if m.terminal_event(e):
    assert m.terminal == 0
    m.terminal = 1
```

An end-of-trace pass reports incomplete monitors, duplicate terminals, missing
events, timestamp inversions, and orphan protocol writes. A live shadow monitor
may alert on the same properties, but native pre-effect guards remain the
enforcement boundary.

### 10.4 Dynamic invariant discovery

Add an offline command that groups redacted NDJSON by event and computes a small
catalog of candidate relations. Store support counts and counterexamples. Promote
only reviewed candidates. DuckDB is convenient for exploratory queries; the
release monitor should be a typed Go program with versioned semantics.

### 10.5 eBPF's role

eBPF is useful for host-level corroboration:

- syscall latency and errors for SQLite files;
- unexpected outbound network calls from a scripting-enabled process;
- process, file, and socket activity during sandbox tests;
- scheduler and CPU evidence during password or policy saturation; and
- TLS/listener behavior at deployment boundaries.

eBPF cannot see semantic facts such as required login, consent, canonical request
digest, or interaction consumption unless application code exports them. It is
therefore a deployment and performance instrument, not the primary invariant
monitor. Begin with native structured events and Go runtime metrics. Add eBPF
only for a named question that cannot be answered at the application boundary.

## 11. Goja security verification plane

### 11.1 Answer to the design question

Language hooks can make security scenario authoring substantially more effective,
but the verification system must be separate from the production policy runtime.
The existing scripting design correctly keeps Go as the TCB, disables ambient
modules, uses immutable graphs, exposes explicit capabilities, and proposes
Go-owned challenge state. Those constraints should be preserved.

Two additions are appropriate:

1. Structural assertions in the compile-time `tinyidp` module. These inspect the
   graph being built and can reject configurations that omit rate limiting,
   durable audit, PKCE, protected-claim rules, or required tests.
2. A separate `tinyidp/verify` module and runtime profile. It compiles behavioral
   suites into pure `VerificationPlan` data. A Go runner executes the plan against
   an in-process provider with test-only capabilities.

Do not add general before/after hooks to the production request path for the
purpose of verification. Mutation hooks expand the TCB, can change the behavior
being observed, and create an alternate authorization mechanism.

### 11.2 Three runtime profiles

| Profile | Module | Capabilities | Effects |
|---|---|---|---|
| Compile | `tinyidp` | graph builders and structural assertions | produces immutable graph |
| Production policy | `tinyidp/policy` or registered callbacks | redacted context and narrow business capabilities | bounded allow/deny/custom claims only |
| Verification | `tinyidp/verify` | scenario DSL only at compile time | produces immutable verification plan |

The Go verification runner, not JavaScript, owns fake time, HTTP driving, store
snapshots, failpoint selection, schedule control, audit collection, invariant
evaluation, and verdict output. The JS source receives no live store or provider
object.

### 11.3 Verification API sketch

```js
const V = require("tinyidp/verify").v1;

module.exports = V.suite("authorization interaction")
  .scenario("forced login cannot reuse browser session", s => s
    .given(V.fixture.validBrowserSession({ authAge: "5m" }))
    .when(V.authorize.begin({ prompt: "login" }))
    .then(V.expect.interaction({ requires: ["fresh_login"] }))
    .when(V.interaction.submit({ login: "", password: "" }))
    .then(V.expect.oauthNotAccepted())
    .assert(V.invariant.freshAuthenticationBeforeIssuance())
    .assert(V.invariant.exactlyOneTerminalOutcome()))
  .scenario("protocol writes are atomic", s => s
    .forEachFailpoint(V.failpoints.authorizationResponseWrites())
    .run(V.flow.authorizationCodePKCE())
    .assert(V.invariant.protocolStateAllOrNone()))
  .build();
```

The compiled Go value contains no JS closures:

```go
type VerificationPlan struct {
    SchemaVersion string
    SourceHash    [32]byte
    Suites        []Suite
}

type Step struct {
    Kind       StepKind
    Parameters json.RawMessage
}

type AssertionRef struct {
    ID      string
    Version string
    Config  json.RawMessage
}
```

Assertion IDs resolve only to native Go implementations. A future custom pure
predicate may run over a bounded redacted snapshot, but it cannot replace native
release assertions and its failure cannot be caught and converted to pass.

### 11.4 Hooks

Safe verification hooks are observer/driver lifecycle events:

- `beforeScenario` resets isolated fixtures;
- `beforeStep` selects a failpoint or scheduling release;
- `afterStep` receives a redacted immutable observation;
- `afterScenario` receives native assertion results; and
- `onCounterexample` formats additional diagnostic context.

They execute in the offline verification process. They do not run in production
authorization. The runner caps source size, compilation time, plan size, step
count, observation bytes, predicate time, and total suite time.

### 11.5 Relationship to embedded policy tests

The scripting guide proposes `Tests []PolicyTest` and embedded policy examples.
Retain those for pure callback input/output behavior. Add a separate
`VerificationPlanRef` to the activation artifact for multi-request protocol
scenarios. Policy tests run during worker warmup; verification suites run in CI,
pre-release environments, and explicit pre-activation gates. They should not
silently run expensive HTTP/failpoint scenarios during normal process startup.

## 12. Existing and recommended tools

| Tool | Role | Use in tiny-idp | Limit |
|---|---|---|---|
| Go `analysis` | typed repository rules | authoritative custom CI lint | needs hand-written summaries for deep interprocedural flow |
| CodeQL | cross-package dataflow/taint | browser/secret sources to redirect/log/script sinks | separate query/toolchain maintenance |
| Semgrep | fast syntax/taint experiments | prototype patterns and configuration guardrails | weaker Go type and interface semantics |
| staticcheck/gosec/govulncheck | baseline defects/dependencies | mandatory general gate | unaware of tiny-idp protocol model |
| Native Go fuzzing | coverage-guided input search | parsers, action sequences, event monitors | corpus design determines reachable states |
| Rapid | typed properties and shrinking | model action sequences | not coverage-guided by default |
| `testing/synctest` | fake time and controlled Go concurrency | pure interaction/store components | external I/O does not belong in the bubble |
| gofail or typed adapters | failure injection | mutation-boundary matrix | failpoint coverage must be enumerated |
| Porcupine | linearizability | one-time consumption and rotation histories | needs a correct sequential specification |
| OIDF conformance | protocol interoperability | final external gate | not an application invariant proof |
| OWASP ZAP | HTTP surface checks | headers, cookies, generic web findings | weak semantic knowledge of OIDC histories |
| eBPF tooling | host corroboration | syscall/network/scheduler questions | cannot infer application protocol state |

Go Flow Levee is useful prior art for Go taint analysis, but it was archived in
April 2026. Do not make an archived project a new release dependency. Study its
source/sink and propagation design where useful.

## 13. Phased implementation plan

### Phase 0: freeze vocabulary and evidence format

1. Review and assign owners to every invariant in Section 5.
2. Mark each property enforce, statically reject, test, monitor, or operationally
   observe.
3. Define `InteractionID`, request digest, event schema v1, reason codes, and
   secret-classification rules.
4. Decide whether consent denial establishes a browser session.
5. Decide supported UserInfo methods and bearer transports.
6. Record the first release profile and exact OIDF plan.

Exit: invariant catalog, threat model, and event schema are accepted; no code
uses ambiguous terms such as "valid session" without a defined state.

### Phase 1: repair the authorization interaction

1. Add a store-owned `InteractionRecord` with hashed random handle, canonical
   validated request data, required actions, timestamps, browser/session binding,
   graph generation, and terminal state.
2. Add atomic create/load/consume/deny/expire methods.
3. Replace hidden request continuation with one opaque handle and CSRF proof.
4. Revalidate mutable server state on resume.
5. Reject malformed `max_age` with overflow-safe arithmetic.
6. Implement explicit consent approve/deny transitions.
7. Add the nine example regressions in Section 8.3.

Exit: forced login and `max_age` cannot be bypassed by blank, crafted, replayed,
mutated, or concurrent POST requests.

### Phase 2: deterministic model harness

1. Introduce the pure Go model and concrete strict-provider driver.
2. Inject the clock and remove policy-relevant direct `time.Now` calls.
3. Implement typed actions and observations.
4. Add Rapid sequential state-machine tests and seed persistence.
5. Add metamorphic relations.
6. Add fuzz adapters for serialized action sequences.

Exit: every named authorization invariant has a model predicate and at least one
positive and negative generated path.

### Phase 3: static analyzer expansion

1. Preserve the earlier `auditlint` tests as the baseline.
2. Implement `securityparse`, `bearertransport`, and `clocksource` first; they are
   local and have concrete current regressions.
3. Implement `interactioncontinuation` after the new store API names are stable.
4. Implement limiter taint with typed fixtures.
5. Extend protocol lifecycle and ignored-security-error rules.
6. Add CodeQL prototypes for cross-package secret and redirect flows.
7. Gate new production diffs in CI; burn down existing findings explicitly.

Exit: each confirmed defect family has a static guard where source structure can
reliably express it.

### Phase 4: instrumentation and monitors

1. Add versioned `SecurityEvent` and an in-memory test collector.
2. Instrument interaction, authentication, consent, protocol mutation, and
   terminal boundaries.
3. Implement redaction/schema validation tests.
4. Build the typed offline trace monitor.
5. Feed every model/fuzz execution through the monitor.
6. Add production metrics with bounded labels and optional shadow alerts.

Exit: the monitor rejects synthetic violations and all deterministic tests
produce complete, secret-free traces.

### Phase 5: concurrency and fault injection

1. Add scheduling probes around one-time and grouped mutations.
2. Build Porcupine models for interaction consume and refresh rotation.
3. Enumerate Fosite handler mutation failpoints.
4. Inject store, commit, audit, serialization, cancellation, and timeout faults.
5. Verify all-or-none storage and one terminal audit outcome.
6. Run `-race`, shuffle, repeated, and CPU-count matrices.

Exit: exact failing schedules and failpoint names are reproducible; no partial
protocol group or duplicate success is observed.

### Phase 6: Goja verification plane

1. Add pure `VerificationPlan` DTOs and validator.
2. Implement compile-only `tinyidp/verify` with no ambient modules.
3. Implement native assertion registry and bounded plan runner.
4. Expose only declarative failpoint/schedule references in the DSL.
5. Add negative tests proving production factories cannot require verification
   modules or resolve test capabilities.
6. Compile the core release scenarios from JS and compare their plan snapshots.

Exit: verification scripts improve authoring without adding production request
authority or secret access.

### Phase 7: external and continuous validation

1. Run the OIDF conformance plan against the exact candidate artifact.
2. Run hosted reverse-proxy/TLS tests with production cookie and client-address
   configuration.
3. Run scheduled fuzz campaigns and retain corpora/counterexamples.
4. Run ZAP or equivalent generic web scanning as a secondary gate.
5. Add targeted eBPF capture only for unresolved syscall/network questions.
6. Produce a signed/hashed release evidence packet.

Exit: local, hosted, external, race, fuzz, failure, and static evidence all name
the same candidate identity.

### Phase 8: release and feedback

1. Canary with shadow monitors and strict alert routing.
2. Verify audit delivery, metric cardinality, and trace sampling behavior.
3. Exercise rollback, key, store, and interaction cleanup runbooks.
4. Review monitor violations and near misses daily during canary.
5. Promote useful dynamically observed relations into explicit tests or static
   rules.

Exit: production approval is a signed decision with residual risks, not an
inference from green CI alone.

## 14. Release gate

A professional release gate requires all of the following:

- no open P1/P2 interaction or lifecycle findings;
- all example and generated invariant suites pass;
- custom analyzers, baseline linters, `govulncheck`, and secret scans pass;
- race, concurrency-history, and failpoint matrices pass;
- fuzz corpora run for the agreed budget with no unresolved crash or invariant
  failure;
- OIDF local and hosted plans pass against the candidate artifact;
- backup/restore, key rotation, audit failure, rollback, and incident drills are
  recorded;
- the candidate image/binary/config/script hashes appear in the evidence packet;
  and
- an accountable owner signs the residual-risk ledger.

Useful command skeleton:

```bash
go test ./... -count=1
go test -race ./... -count=1
go test ./... -shuffle=on -count=20
go run ./ttmp/.../scripts/auditlint ./...
go test ./internal/fositeadapter -run 'Interaction|Prompt|MaxAge|Consent' -count=1
go test ./pkg/sqlitestore -run 'Interaction|Refresh|Atomic|Failpoint' -count=1
go test ./internal/assurance -run 'Model|Linearizable|Trace' -count=1
go test ./internal/assurance -fuzz FuzzInteractionSequence -fuzztime 10m
```

## 15. Decision records

### Decision: authoritative invariants are native and typed

- **Context:** Tests, analyzers, runtime monitors, and scripts need one vocabulary.
- **Options considered:** prose-only rules; JavaScript predicates; pure Go model.
- **Decision:** Define invariant IDs, model state, event schema, and release
  verdicts in Go. Scripts reference native assertions.
- **Rationale:** The production TCB and protocol implementation are Go; typed
  native semantics are reviewable and cannot be redefined by a plugin.
- **Consequences:** Adding a custom invariant requires a Go implementation before
  it can become an authoritative release gate.
- **Status:** proposed.

### Decision: server-owned opaque authorization interactions

- **Context:** Browser-carried continuation loses required actions and permits
  mutation/replay.
- **Options considered:** preserve more hidden fields; sign the whole browser
  request; store an opaque interaction server-side.
- **Decision:** Store canonical state server-side and expose one random one-time
  handle.
- **Rationale:** This makes required actions, original request, expiry, replay,
  mutable-state revalidation, and audit correlation explicit.
- **Consequences:** A new store lifecycle, cleanup job, and concurrency contract
  are required. No backwards compatibility layer should retain the hidden-field
  path.
- **Status:** proposed.

### Decision: use layered assurance rather than one universal tool

- **Context:** AST checks, taint, state histories, concurrency, and runtime traces
  expose different defect classes.
- **Options considered:** only conformance; only fuzzing; only static analysis;
  layered program.
- **Decision:** Use the six-layer architecture in the executive summary.
- **Rationale:** Each layer supplies evidence unavailable to the others and can
  cross-check instrumentation blind spots.
- **Consequences:** CI and release evidence are more complex and require named
  ownership.
- **Status:** proposed.

### Decision: verification scripts compile plans and do not mutate production

- **Context:** Language hooks make scenarios approachable but can enlarge the
  trust boundary.
- **Options considered:** production before/after hooks; same runtime with test
  capabilities; isolated compile-only verification module.
- **Decision:** Use a separate `tinyidp/verify` profile that emits a pure plan
  executed by Go.
- **Rationale:** Verification needs fake clocks, failpoints, and store inspection,
  which are categorically forbidden production policy capabilities.
- **Consequences:** There are separate module/provider registries and negative
  isolation tests.
- **Status:** proposed.

### Decision: runtime traces are evidence, not authorization

- **Context:** A post-hoc monitor sees violations after effects and can miss
  uninstrumented actions.
- **Options considered:** rely on audit; let monitor authorize; native guard plus
  trace monitor.
- **Decision:** Native code guards irreversible transitions; traces support tests,
  shadow detection, investigation, and invariant discovery.
- **Rationale:** Enforcement requires complete mediation before the effect.
- **Consequences:** Important properties may exist both as pre-effect assertions
  and post-effect trace checks.
- **Status:** proposed.

## 16. Alternatives and rejected shortcuts

### Preserve every authorization parameter in hidden inputs

This reduces accidental loss but leaves the browser as mutable protocol storage,
does not solve one-time consumption, and makes extension-parameter review
perpetual. Reject it.

### Sign or encrypt the entire continuation without server state

Integrity protection prevents mutation but does not by itself provide atomic
one-time use, current-state revalidation, revocation, cleanup, or simple
concurrency semantics. It may be appropriate for a different architecture, but
the current store already supports server state and the security requirements
favor an opaque handle.

### Run arbitrary JavaScript assertions in production

This adds mutable, potentially hanging code to the security path and makes the
monitor part of the behavior it observes. Keep production guards native. Optional
script shadow checks must be bounded, non-authoritative, and unable to suppress
native events or verdicts.

### Depend on eBPF for protocol invariants

Kernel instrumentation sees syscalls and scheduling, not OIDC semantics. It can
corroborate file/network/process behavior but cannot replace application events.

### Infer all invariants from traces

Observed relations depend on workload coverage. Use mining to propose and
prioritize properties, then make them explicit and seek stronger evidence.

## 17. Risks and open questions

1. What exact browser/session binding should an interaction use before and after
   login? Account switching and pre-login flows need deliberate semantics.
2. How should a committed authorization response recover when HTTP delivery fails?
   Retrying must not duplicate artifacts, but the client needs a defined result.
3. Can Fosite handler writes share a transaction without forking Fosite or
   exposing SQL transactions through public store interfaces?
4. Which audit events are synchronous release-critical evidence, and which may be
   buffered? Backpressure behavior must be explicit.
5. Should production shadow monitors sample? Sampling is incompatible with
   "exactly once" conclusions unless completeness is separately known.
6. Is Goja verification authoring trusted-operator only? Untrusted authors require
   a process boundary even for compile-time scripts.
7. Which CodeQL queries are worth maintaining after custom analyzers cover local
   rules?
8. What scheduled fuzz budget and corpus retention policy can the team sustain?
9. Which graph-generation changes invalidate pending interactions, and which may
   drain against the old generation?

## 18. Code review and implementation entry points

Read in this order:

1. `internal/fositeadapter/provider.go:340-438` for the interaction state split.
2. `internal/fositeadapter/provider.go:574-620` for mutable continuation and
   request-object error routing.
3. `internal/fositeadapter/session.go:28-61` for session error collapse and
   `max_age` semantics.
4. `internal/fositeadapter/provider.go:461-514` for token limiting and UserInfo.
5. `pkg/idpstore/interfaces.go` and `pkg/sqlitestore/store.go` for atomic API
   boundaries.
6. The earlier `scripts/auditlint/main.go` for analyzer conventions.
7. `reference/04-authorization-interaction-and-protocol-robustness-review.md` for
   the finding and test matrix.
8. The Goja scripting design at
   `ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md`.
9. Its companion assessment in
   `reference/02-security-verification-scripting-plane-assessment.md`.

## 19. Research and source packet

The ticket `sources/` directory contains Defuddle Markdown plus full PDFs where
appropriate. Primary research includes:

- Fett, Küsters, and Schmitz, *A Comprehensive Formal Security Analysis of OAuth
  2.0*.
- Fett, Küsters, and Schmitz, *The Web SSO Standard OpenID Connect: In-Depth
  Formal Security Analysis and Security Guidelines*.
- Schneider, *Enforceable Security Policies*.
- Reps, Horwitz, and Sagiv, *Precise Interprocedural Dataflow Analysis via Graph
  Reachability*.
- Schieferdecker, Grossmann, and Schneider, *Model-Based Security Testing*.
- de Ruiter and Poll, *Protocol State Fuzzing of TLS Implementations*.
- Ba, Böhme, Mirzamomen, and Roychoudhury, *Stateful Greybox Fuzzing*.
- Lampropoulos, Hicks, and Pierce, *Coverage-Guided Property-Based Testing*.
- Musuvathi et al., *Finding and Reproducing Heisenbugs in Concurrent Programs*.
- Alvaro, Rosen, and Hellerstein, *Lineage-Driven Fault Injection*.
- Ernst et al., *The Daikon System for Dynamic Detection of Likely Invariants*.
- Nimmer and Ernst, *Static Verification of Dynamically Detected Program
  Invariants*.
- Strom and Yemini, *Typestate: A Programming Language Concept for Enhancing
  Software Reliability*.
- Chen and Roşu, *MOP: An Efficient and Generic Runtime Verification Framework*.
- Manès et al., *The Art, Science, and Engineering of Fuzzing*.

Standards and implementation references in the same packet include OpenID
Connect Core, RFC 6749, RFC 6750, RFC 9700, the Go analysis framework, native Go
fuzzing, `testing/synctest`, CodeQL Go dataflow, Semgrep rules, gofail, Rapid,
Porcupine, and the OpenID Foundation conformance suite.
