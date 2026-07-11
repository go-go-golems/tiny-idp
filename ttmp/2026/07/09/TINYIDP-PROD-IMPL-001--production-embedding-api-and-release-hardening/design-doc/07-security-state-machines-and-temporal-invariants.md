---
Title: Security State Machines and Temporal Invariants
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - architecture
    - auth
    - identity
    - research
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction.go
      Note: |-
        Canonical request reconstruction and interaction lifecycle
        Lifecycle implementation
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: |-
        Pure model, Rapid properties, minimized histories, and fuzz actions
        Executable reference model
    - Path: repo://internal/securitytrace/trace.go
      Note: |-
        Executable temporal monitor
        Parametric monitor
    - Path: repo://pkg/idpstore/types.go
      Note: |-
        Interaction states, required actions, outcomes, and store contracts
        Interaction state vocabulary
ExternalSources: []
Summary: A precise method for expressing authentication and authorization requirements as legal histories rather than isolated input checks.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Teaching contributors to reason about replay, freshness, consent, expiry, and terminal outcomes across requests.
WhenToUse: Before changing multi-request flows, sessions, interaction persistence, security events, or stateful tests.
---


# Security State Machines and Temporal Invariants

## Why state is the unit of reasoning

Input validation answers whether one message is well formed. A temporal
invariant answers whether a sequence of messages and mutations is legal. Forced
reauthentication, one-time interaction consumption, consent-before-issuance, and
refresh rotation are temporal properties. No regular expression or handler-local
boolean can express their full meaning.

An interaction has this abstract state:

```text
Absent --create--> Pending --approve--> Approved
                       |       |
                       |       +--> authorization artifacts committed
                       +--deny--> Denied
                       +--time--> Expired

Approved, Denied, Expired --consume/replay--> rejected
```

Required actions refine `Pending`. They are obligations that must be discharged
by later events, not hints for rendering:

```text
required = {fresh_login, consent}
satisfied = {}

AuthenticationSatisfied -> satisfied += fresh_login
ConsentApproved          -> satisfied += consent

approve is legal iff required subset-of satisfied
```

## Safety and liveness

A safety property states that a bad event never occurs. “At most one terminal
outcome” and “no artifacts before approval” are safety properties. A finite
counterexample can falsify them.

A liveness property states that a desired event eventually occurs. “Every valid
approved request eventually receives a response” is liveness. Timeouts, crashes,
and client abandonment complicate liveness, so tiny-idp's current monitor focuses
primarily on safety and records operational availability separately.

This distinction prevents an availability failure from being mislabeled as an
authorization bypass, and prevents a secure rejection from being mislabeled as
successful protocol progress.

## Linear-time properties used by tiny-idp

For one interaction identifier `i`:

```text
Created(i) occurs at most once.
Terminal(i, outcome) occurs at most once.
Artifacts(i) implies a prior Terminal(i, approved).
Terminal(i, approved) with require_login implies prior Authenticated(i).
Terminal(i, approved) with require_consent implies prior ConsentApproved(i).
Denied(i) excludes Artifacts(i).
Expired(i) excludes later successful terminal consumption.
```

These are implemented as a parametric monitor: one state machine is instantiated
per opaque interaction identifier. The event stream is versioned and secret-free.
Partitioning is essential; events from two browser tabs must not satisfy each
other's obligations.

## Why the opaque record matters

The stored interaction is a materialized protocol state. It makes required
actions, expiry, canonical request data, and terminal status observable to one
atomic store operation. Without it, the POST handler would have to infer past
facts from browser fields and current session state. That reconstruction is
weaker because required actions can disappear and mutable configuration cannot
be compared with the original generation.

## Executable models

`state_model_test.go` defines a smaller model than the provider. Its purpose is
not to reproduce HTTP or Fosite. It predicts the legal results of create, get,
approve, deny, expire, replay, and returned-copy mutation.

```text
model.Apply(action) -> expected observation
store.Execute(action) -> actual observation
assert relation(expected, actual)
```

Rapid generates action sequences and shrinks failures. Committed minimized
histories make important counterexamples readable without a random generator:
create/approve/approve, create/deny/approve, create/expire/approve, and approve
without create.

## Metamorphic relations

Some protocol outputs contain random codes and signatures, so exact equality is
the wrong oracle. A metamorphic test defines how a controlled input transform
may affect observations. The current `ui_locales` relation states that changing
presentation locale must not change successful issuance or returned server-owned
`state`.

Security-relevant transforms need explicit non-equivalence rules. Duplicating
`redirect_uri`, changing PKCE challenge, or changing the opaque interaction is
not an irrelevant transformation and must fail closed.

## Exercises

1. Draw the history for forced login with an old session and blank POST. Mark the
   precise missing event that made the old implementation unsafe.
2. Explain why two concurrent tabs require separate monitor partitions.
3. Add a proposed `step_up` action to the abstract state without writing code.
   Define its obligation and terminal rule.
4. Classify each property above as safety or liveness.
5. Design one valid and one invalid metamorphic relation for authorization.

## Retrospective case study: how the temporal model emerged

The temporal model was not selected before the code review. It was derived from
a concrete discrepancy between two requests in one authorization interaction.
Diary Step 17 records the original trace. The GET request evaluated
`prompt=login` or `max_age`, rendered a credential form, and then discarded the
reason that made credentials mandatory. The POST request saw an existing
session, an empty login, and enough reconstructed fields to call
`finishAuthorize`. Each handler branch appeared locally plausible. The combined
history violated freshness.

The first useful abstraction was not “password must be non-empty.” That rule
would couple protocol semantics to one UI field and would not generalize to
passkeys, step-up authentication, or future challenge methods. The useful
abstraction was an obligation:

```text
interaction created with fresh-authentication required
    -> authentication event after creation is mandatory
    -> approval is illegal before that event
```

This is why `InteractionRequiredAction` is a bit set in
`pkg/idpstore/types.go`. The record stores obligations independently from the
mechanism used to satisfy them. `InteractionRequireLogin` describes the absence
of an authenticated session. `InteractionRequireFreshLogin` describes a request
whose semantics reject reuse of an otherwise valid session.

The distinction is visible in `Provider.resumeAuthorize`. The handler computes
`requiresLogin` from the stored record, not from the current POST. If login is
required and the normalized login is empty, the handler emits a stable audit
reason and rejects the request before session fallback. When authentication
succeeds, the handler updates `authTime` from `p.now()` and emits
`AuthenticationSatisfied` for the same interaction trace identifier.

### Research influence

The formal OAuth and OpenID Connect analyses in the source packet treat session
integrity as a protocol security goal. That framing changed the review question.
Instead of checking only whether an attacker can steal a token, the analysis
checks whether the authorization response corresponds to the user's intended
session and action. The local forced-login defect is a session-integrity defect:
the response is cryptographically valid but represents the wrong authentication
event.

The typestate paper supplied useful API vocabulary. A pending interaction and a
consumed interaction should not accept the same operations. Go does not encode
the full state in separate compile-time types here because the state is durable
and loaded dynamically. The store contract implements the same discipline at
runtime: `ConsumeInteraction` accepts only a pending, unexpired record and
returns typed errors for absent, expired, or already-consumed state.

The runtime-verification literature supplied the separation between
instrumentation and verdict. `recordSecurity` emits a fact after the native
transition. `Monitor.Observe` owns the temporal rule. The provider does not ask
the monitor for permission to authorize. This keeps request correctness from
depending on optional evidence delivery while making trace violations
independently testable.

## Concrete state representation

`InteractionRecord` is the durable representation of the abstract state. Each
field supports a specific invariant.

### `IDHash`

The browser receives a random handle. The store indexes its cryptographic hash.
Database disclosure therefore does not directly reveal a live browser handle.
The raw handle remains a bearer continuation and must be protected by TLS,
expiry, CSRF, and browser binding.

### `CanonicalRequest`

This map holds the protocol continuation reconstructed from the validated
Fosite requester. It is not copied from the POST. Values are deep-copied by
store implementations so a caller cannot mutate durable state through a map or
slice alias. The Rapid model includes a returned-copy mutation operation because
copy isolation is part of the store contract.

### `RequestDigest`

The digest makes the canonical request comparable without logging its full
contents. It is evidence of equality, not authenticity by itself. Authenticity
comes from server ownership of record creation and lookup.

### `ClientID` and `RedirectURI`

These fields identify the principal and response destination that were accepted
at creation. Resume loads the current client and verifies that redirect and
scope remain allowed. The stored values prevent browser substitution; current
lookup prevents stale configuration from remaining authorized indefinitely.

### `RequiredActions`

This is the obligation set. It survives the request boundary and is included in
the `InteractionCreated` security event. The monitor therefore knows which
later events are prerequisites for approval.

### `BrowserBindingHash`

The binding ties resume to browser context without storing raw browser material.
It narrows theft and cross-browser replay. It does not replace CSRF or handle
entropy, and it must not be described as device identity.

### `SessionIDHash`

This field is present only when creation observed an active session that is
relevant to the interaction. The implementation deliberately stopped binding a
stale cookie when no active session existed; otherwise an invalid cookie could
become accidental continuation authority.

### `GenerationHash`

The client generation hash covers security-relevant client configuration:
redirects, allowed scopes, PKCE requirement, disabled state, and update time.
Resume rejects a changed generation. This is stricter than checking only client
ID and protects a pending interaction from configuration changes.

### `CreatedAt` and `ExpiresAt`

These timestamps use the injected provider clock. Expiry is evaluated by the
atomic store transition. A deterministic clock lets tests advance beyond the
interaction TTL without sleeping and removes scheduler timing from the result.

### `ConsumedAt` and `Outcome`

These fields represent the terminal transition. They are written atomically and
are never inferred from the existence of downstream artifacts. This permits
clear replay errors and operations inspection.

## Concrete transition: begin authorization

`Provider.beginAuthorize` is the first transition coordinator.

Its input is a raw HTTP request. Its first security boundary is
`p.oauth2.NewAuthorizeRequest`, which validates client and protocol parameters.
No interaction exists before this succeeds.

The handler then parses `max_age` strictly. The parser returns value, presence,
and error separately. Negative, non-decimal, and overflow values are rejected.
The former fail-open behavior treated malformed input as if the constraint were
absent; the custom `tinyidpstrictparse` analyzer now searches for this family of
mistake in boolean security predicates.

The handler reads the browser session with a four-state result rather than a
user-or-empty shortcut:

```text
missing
active
inactive or expired
storage error
```

Storage error returns service unavailable and never renders credentials. This
prevents an availability failure from being interpreted as ordinary absence.

The handler derives required actions from session state and request semantics.
If `prompt=none` conflicts with any required action, it returns the appropriate
OAuth error and creates no interactive form. Otherwise it creates the durable
interaction before rendering.

The ordering matters:

```text
validate -> derive obligations -> persist -> render
```

Rendering before persistence would show a form that cannot be resumed safely.
Persisting before validation would store attacker-selected invalid protocol
state.

## Concrete transition: resume authorization

`Provider.resumeAuthorize` accepts only POST. It parses the form, verifies CSRF,
loads the opaque interaction, and reconstructs the Fosite request from
`CanonicalRequest`.

The `tinyidpinteractioncontinuation` analyzer exists because this boundary is
easy to regress. It reports reads of browser POST protocol fields from
`resumeAuthorize`. The rule intentionally permits native user-input fields such
as login, password, action, interaction, and CSRF. It is a local structural
guard, not interprocedural proof.

Resume revalidates the client generation, redirect, scopes, active signing key,
session, and user. Revalidation is a second temporal obligation:

```text
valid_at_creation does not imply valid_at_issuance
```

An administrator can disable a client or user while the browser form is open.
Tests mutate each state between begin and submit and assert that no code is
issued.

Denial consumes the interaction with a denied outcome before writing the OAuth
`access_denied` response. Approval requires satisfaction of stored login and
consent obligations. The terminal consume and authorization persistence share
the appropriate lifecycle boundary in the SQL path.

## Case file: forced `prompt=login`

Initial conditions:

- the browser has an active session;
- the new request contains `prompt=login`;
- client, redirect, scope, nonce, and PKCE are valid.

Expected history:

```text
InteractionCreated(required=fresh_login)
AuthenticationSatisfied
InteractionTerminal(approved)
AuthorizationArtifactsDone
```

Invalid history formerly possible:

```text
InteractionCreated(required=fresh_login)
InteractionTerminal(approved)
AuthorizationArtifactsDone
```

`TestForcedPromptLoginCannotReuseExistingSession` submits the opaque form
without login. Its oracle is absence of an authorization code, not merely a
particular status. This is important because an error page and a redirect can
both vary while capability non-issuance remains the invariant.

## Case file: expired `max_age`

`max_age` compares the current security clock with session `AuthTime`. A present
zero value requires authentication unless the timestamp is effectively current.
The parser rejects negative, malformed, and overflowing decimal values before
rendering credentials.

`TestExpiredMaxAgeCannotReuseExistingSession` proves that an existing session
does not satisfy a zero-age request after time has advanced. The injected clock
test proves interaction expiry independently from authentication age.

The two clocks are conceptually related but distinct:

- authentication age constrains which session event may satisfy the request;
- interaction TTL constrains how long a pending decision remains actionable.

## Case file: `prompt=none`

Non-interactive authorization permits no login or consent UI. If either action
is required, the server returns `login_required` or `consent_required` through a
validated redirect. It must not create a form and wait for later user input.

This is a transition exclusion:

```text
prompt_none AND required_actions != empty
    -> terminal protocol error
    -> no Pending interactive state
```

## Case file: explicit consent

Consent is represented as an obligation separate from authentication. The form
displays the bound client and requested scopes but does not return them as
authoritative inputs. Approval and denial are explicit action values.

An omitted decision does not satisfy consent. Denial creates a terminal denied
outcome. Stored prior consent can remove the obligation before interaction
creation, but only through the native consent policy.

The event model distinguishes `ConsentApproved` and `ConsentDenied`. Approval is
required before approved terminal state when the corresponding bit is set.
Denial cannot be followed by artifacts.

## Case file: sequential replay

The first valid submit atomically consumes the interaction. A second submit with
the same handle reaches an already-consumed record and cannot issue another
code.

The minimal history is committed in model tests:

```text
create -> approve(accepted) -> approve(already_consumed)
```

Sequential replay tests store the exact browser form and submit it twice. This
tests the HTTP boundary, CSRF state, store behavior, and artifact absence
together.

## Case file: concurrent replay

Two goroutines submit the same interaction with the same browser cookies. The
store's atomic consume permits exactly one terminal success. Mutex protection in
memory and conditional SQL update in SQLite implement the same abstract object.

The test counts issued codes and requires exactly one. The Porcupine history
records invocation and return intervals for sixteen concurrent consumers and
checks that the results admit a legal sequential order.

Race detection would answer whether memory was unsafely accessed. It would not
answer whether two synchronized consumers both succeeded. Porcupine answers the
second question relative to the supplied model.

## Case file: concurrent tabs

Two tabs share a browser cookie jar but receive different random interaction
handles and state values. Each can complete independently. This case prevents an
overcorrection in which CSRF or session binding permits only one outstanding
interaction per browser.

The relevant property is isolation, not global serialization:

```text
events(i1) satisfy only obligations(i1)
events(i2) satisfy only obligations(i2)
```

Parametric monitor state keyed by interaction trace ID implements the same
partitioning.

## Case file: request mutation

The test adds attacker-controlled `state` values to the POST. Resume ignores
them and returns the original stored state. Equivalent tests can target client,
redirect, scope, nonce, and PKCE fields. The form itself contains none of these
hidden protocol inputs.

Mutation resistance has two layers:

1. authority reduction: protocol fields are absent from the browser form and
   are never read on resume;
2. integrity verification: canonical request digest and client generation are
   checked inside server-owned state.

## Case file: expiry

The test creates an interaction at a deterministic time, advances the clock by
eleven minutes, and submits the original form. Atomic consume observes expiry
and issues no code.

Expiry must be checked at transition time, not only when loading. Otherwise two
operations could load a pending record before expiry and both act after expiry.
The store method receives `now` as an explicit argument for this reason.

## Case file: disabled client or user

Client generation and current user status are reloaded on resume. Disabling
either principal between begin and submit causes rejection. This case shows why
the canonical record is necessary but not sufficient: immutable request binding
must be combined with current mutable security state.

## Case file: signing-key unavailability

Authorization must not collect credentials or consume the interaction if no
active signing key can support the eventual ID token. The provider checks the
key before irreversible terminal progress. Error handling must preserve a
retryable state where appropriate and must not claim artifact commit.

## From model to monitor

The pure model predicts store operation results. The runtime monitor checks
provider event histories. They overlap but observe different abstractions.

| Layer | Input | State | Verdict |
|---|---|---|---|
| Pure model | action enum | created/consumed/expired | accepted + reason |
| Store contract test | method calls | concrete memory/SQLite | typed error/result |
| Strict driver | HTTP steps | cookies/form/clock | typed observations |
| Security monitor | events | obligations and terminal state | violations |

A defect can appear in only one layer. For example, a provider might fail to
emit an event even though store state is correct. Cross-feeding real-provider and
failpoint traces into the monitor reduces this gap but does not eliminate the
need to review instrumentation completeness.

## The monitor implementation

`Monitor.Observe` first rejects unsupported schema versions. Events without an
interaction identifier are ignored by the interaction monitor; token lifecycle
events currently belong to a different future partition.

`InteractionCreated` allocates state and records the required-action bit set.
A second create for the same key is a violation.

Authentication and consent events require existing state. An event before
creation is a violation because it could otherwise satisfy a later interaction.

Terminal processing checks duplicate outcomes and required prerequisites.
Artifact processing requires approved terminal state and rejects duplicates.

The monitor returns all accumulated violations so offline analysis can report
more than the first failure. It does not mutate provider behavior.

## Event timing rules

An event must be emitted after the authoritative transition it claims.

- `InteractionCreated` follows successful durable create.
- `AuthenticationSatisfied` follows authenticator success and session creation.
- consent events follow explicit native policy/user decisions.
- `InteractionTerminal` follows successful atomic consume.
- `AuthorizationArtifactsDone` follows lifecycle commit.
- `TokenLifecycleDone` follows Fosite token transaction commit.

Emitting before commit would create false evidence on rollback. Emitting only
after HTTP response write could miss a committed transition when client delivery
fails. The chosen event position describes durable authority, while delivery
failure is counted separately.

## Generated valid traces

The property test chooses required authentication and consent bits, then
constructs a history that satisfies exactly those obligations before terminal
approval. Denied histories contain consent denial and no artifacts. The monitor
must accept every generated valid trace.

This is a positive property. The fuzz target supplies arbitrary event sequences,
versions, outcomes, and partition identifiers to test total handling and discover
unexpected monitor states. Hand-authored negative tests assert specific
violations. All three are necessary:

- examples document intended diagnostics;
- generated valid traces protect against false positives;
- fuzzing protects parser/state handling under malformed sequences.

## Typed verification-plan driver

`verification_scenario_test.go` connects data-only Goja plans to the real strict
provider. The driver owns cookies, forms, and clock. JavaScript selects only
named actions and native assertion IDs.

The initial actions are `session.login`, `authorize.begin`,
`interaction.submit`, and `clock.advance`. Parameter decoding rejects unknown
fields and trailing JSON. Observations report code presence rather than code
values, preserving evidence usefulness without recording bearer artifacts.

The forced-login plan establishes a session, begins with `prompt=login`, submits
blank input, and invokes native assertions for credential-form display and no
authorization code. This is the executable form of the temporal case study.

## Review heuristics for new transitions

When adding a transition, answer these questions in order:

1. What durable object owns the state?
2. What are the legal predecessor states?
3. Is the transition terminal, repeatable, or idempotent?
4. Which obligations must already be satisfied?
5. Which mutable state must be revalidated?
6. What is the atomic linearization point?
7. Which event is emitted, and after which authoritative mutation?
8. What happens on retry, timeout, cancellation, and duplicate delivery?
9. Can two browser tabs proceed independently?
10. Which counterexample should be committed as a minimized history?

## Common incorrect models

### “A valid cookie means login is satisfied”

This ignores freshness requests and session revocation. The correct predicate
depends on request obligations, session state, authentication time, and current
user status.

### “CSRF makes hidden fields safe”

CSRF authenticates browser context for the form submission. It does not prove
that protocol fields are unchanged from the validated client request.

### “A 400 response proves no capability was issued”

Persistence may have committed before response construction failed. Negative
tests must inspect durable artifacts and events.

### “One mutex proves exactly-once semantics”

A mutex can remove data races while the protected logic still accepts two
semantic transitions. Exactly-once must be stated and tested at the object
model.

### “A complete event trace proves implementation correctness”

The monitor proves only its properties over emitted facts. Incorrect or missing
instrumentation can make the trace incomplete.

## Decision records

### TI-1: Persist obligations

- **Context:** required login and consent vanished between GET and POST.
- **Decision:** store required actions in the interaction record.
- **Consequence:** resume cannot weaken requirements by recomputation.

### TI-2: One atomic terminal transition

- **Context:** replay and concurrent submit must not issue multiple codes.
- **Decision:** expose `ConsumeInteraction(now, outcome)` as an atomic store
  method with typed terminal errors.
- **Consequence:** memory and SQLite implementations share one abstract model.

### TI-3: Native event verdicts

- **Context:** programmable scenarios are useful, but script-defined verdicts
  would not be independent assurance.
- **Decision:** native code emits events and owns monitor/assertion verdicts.
- **Consequence:** scripts select experiments without gaining protocol authority.

### TI-4: Deterministic security time

- **Context:** freshness and expiry tests were otherwise scheduler-dependent.
- **Decision:** inject the provider clock and pass transition time explicitly to
  stores.
- **Consequence:** tests advance time without sleeping; the clock analyzer guards
  direct wall-clock regression in named security code.

## Extended exercises

1. Write the event trace for stored-consent skip and explain why no
   `ConsentApproved` event is required for a newly absent obligation.
2. Decide whether authentication failure should consume the interaction. State
   the retry and brute-force consequences of each choice.
3. Extend the monitor model with `InteractionExpired`. Decide whether expiry is
   an event, an observed rejection, or both.
4. Design a token-family parametric monitor keyed without logging token values.
5. Define the legal history for password-change-required authentication.
6. Explain where a WebAuthn challenge would live and what binds its response.
7. Add a hypothetical administrator client-disable event to the history. Explain
   why issuance must still query current state.
8. Compare terminal denial with protocol validation error before interaction
   creation.
9. Determine whether artifact delivery failure is safety, liveness, or both.
10. Review `Monitor.Observe` for a property it intentionally does not express.

## Chapter review checklist

- Can the reader distinguish validation from temporal legality?
- Can the reader name every `InteractionRecord` field's invariant?
- Can the reader reconstruct begin and resume from authoritative sources?
- Can the reader state forced-login and consent properties as histories?
- Can the reader explain sequential and concurrent replay?
- Can the reader distinguish model, store, driver, and monitor evidence?
- Can the reader locate event emission after authoritative transitions?
- Can the reader state the monitor's observability limitation?
- Can the reader design a new action without exposing provider authority to JS?
- Can the reader identify the remaining token-family monitoring gap?

## Research map

- Security automata motivate explicit permitted transition histories.
- Typestate motivates making legal operations depend on the current state.
- Model-based testing connects an abstract transition system to real execution.
- Runtime verification checks emitted histories against executable temporal
  properties.
- See `sources/paper-enforceable-security-policies.md`,
  `paper-typestate.md`, `paper-model-based-security-testing.md`, and
  `paper-runtime-verification-brief-account.md`.
