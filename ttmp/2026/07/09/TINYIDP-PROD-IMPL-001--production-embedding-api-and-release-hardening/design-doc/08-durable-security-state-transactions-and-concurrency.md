---
Title: Durable Security State, Transactions, and Concurrency
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
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: |-
        Concurrent interaction and refresh histories
        Concurrent histories
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: |-
        Fosite transaction context and protocol persistence
        Protocol transaction propagation
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: |-
        Authorization and token failpoint proofs
        Failpoint evidence
    - Path: repo://pkg/sqlitestore/store.go
      Note: |-
        Durable SQLite topology and transaction implementation
        SQLite ownership
ExternalSources: []
Summary: Theory and implementation of atomic capability mutation, SQLite transaction ownership, refresh rotation, fault injection, and linearizability.
LastUpdated: 2026-07-10T22:10:00-04:00
WhatFor: Teaching why protocol persistence must be designed around complete authority transitions rather than individual SQL statements.
WhenToUse: Before modifying store methods, Fosite adapters, token rotation, backup, recovery, or concurrent tests.
---


# Durable Security State, Transactions, and Concurrency

## Capabilities are durable state

Authorization codes, sessions, access tokens, refresh tokens, signing keys, and
consents grant authority. Their database rows are not ordinary application
records. Partial mutation can duplicate authority, lose legitimate authority,
or make audit and protocol responses disagree with durable state.

Atomicity must cover the complete protocol transition. For code redemption:

```text
BEGIN
  verify and deactivate exact active authorization code
  create access-token record
  create refresh-token record when requested
COMMIT
write token response
```

For refresh rotation:

```text
BEGIN
  conditionally deactivate exact active presented refresh token
  delete/revoke associated old access token
  create replacement access token
  create replacement refresh token
COMMIT
write token response
```

The conditional update is the rotation linearization point. If zero rows change,
another operation won or the token was already inactive; the adapter returns a
serialization failure instead of pretending to rotate it.

## Transaction propagation

Fosite orchestrates several storage calls while constructing one token response.
Its `storage.Transactional` extension begins a transaction and carries it in the
request context. `sqlFositeStore` selects the context transaction for every
participating mutation. Starting a private transaction inside only
`RotateRefreshToken` would omit replacement token creation from the atomic unit.

This is why transaction ownership follows the protocol orchestrator, not the
first repository method that happens to write SQL.

## Failure atomicity

Fault injection places named failures before and after each mutation and before
commit. A correct rollback test inspects all related rows, not only the method's
return value.

For failed code exchange, evidence requires:

```text
authorization code remains active
access-token count = 0
refresh-token count = 0
committed TokenLifecycleDone events = 0
```

For failed refresh rotation, the original token pair remains active, replacement
rows do not survive, and retry with the original refresh token succeeds. These
claims cover the named failpoints; they are not a proof against every possible
filesystem or kernel failure.

## Concurrency and linearizability

A data race is unsynchronized memory access. Linearizability is a semantic
property: each completed concurrent operation must appear to take effect at one
instant between invocation and response, producing a legal sequential history.
Race-detector success does not establish linearizability.

Porcupine receives operation intervals and a sequential model. Interaction
consumption permits one successful terminal operation. Refresh rotation permits
one successful use of an active token. Later requests can observe the inactive
token and trigger family reuse revocation. Thus a history can be linearizable
while the final active-family count is zero.

## SQLite topology and recovery

Production uses one active tiny-idp process, a local SQLite filesystem, WAL
mode, and one open connection. This topology is part of correctness. A network
filesystem or active/active writers would require new coordination semantics.

Backup uses SQLite's online backup API, verifies schema and migration checksums,
and records table counts. Restore is offline and preserves a rollback copy.
Newer schema versions are rejected by older binaries. These rules turn recovery
from an informal file-copy procedure into a tested state transition.

## Exercises

1. Identify the atomic unit if token response serialization fails after commit.
   Explain why database rollback can no longer retract an exposed response.
2. Explain why compensation is weaker than a transaction for capability
   issuance.
3. Write the sequential model for two concurrent interaction approvals.
4. Explain why legitimate concurrent refresh can cause family revocation.
5. Pick one failpoint and enumerate every row and event that must be absent or
   preserved.

## Retrospective: from entity methods to protocol transactions

The first store design exposed atomic operations for domain entities: consume an
authorization code, rotate a refresh token, update lockout state, rotate a
signing key, and persist consent. Those methods were necessary, but the later
authorization review found a larger atomicity boundary inside Fosite.

Fosite constructs an authorization response through multiple handlers. One
handler stores the authorization-code session, another stores PKCE state, and an
OpenID handler stores OIDC session state. A transaction inside any one method
cannot make the three-handler sequence atomic. The same structure exists at the
token endpoint: code invalidation and replacement token creation are separate
storage calls coordinated by Fosite.

Diary Steps 17 and 21 record the shift in reasoning. The question changed from
“is this repository method transactional?” to “which set of mutations makes one
protocol response true?” That question identifies the correct atomic unit.

## Capability interpretation of rows

An authorization code row grants the ability to exchange one validated browser
authorization for tokens. An active refresh row grants the ability to create a
new token generation. A session row grants browser continuity. A signing key
grants the process the ability to produce assertions accepted by relying
parties.

The database therefore holds a capability graph:

```text
user + client + authorization decision
                |
                v
        authorization code
                |
                v
      access token + refresh token
                         |
                         v
                 replacement generation
```

Edges describe derivation and revocation relationships. Partial updates can
leave an edge pointing to absent state, preserve two active generations, or
consume the only credential without producing its replacement.

## Store contract layers in the current project

`pkg/idpstore/interfaces.go` separates small interfaces for clients, users,
credentials, account security, grants, codes, access tokens, refresh tokens,
consents, sessions, interactions, and keys. `StoreOperations` composes ordinary
domain operations. `ReadStore` exposes the read-only view used inside
transactions. `TxStore` adds commit and rollback. `AtomicStore` exposes compound
security transitions. `Store` composes the public production contract.

This layered design serves three purposes:

- consumers can depend on the narrowest required authority;
- a transaction can expose the same domain operations without allowing nested
  transaction creation;
- memory and SQLite implementations can share the conformance suite.

The compile-time interface assertions document which concrete types implement
which contracts. They turn interface drift into a build failure.

## SQLite topology as a correctness assumption

The SQLite store configures one active process and one open connection. WAL
supports readers during writes and the online backup API, but it does not turn
the design into a distributed database. Active/active processes or a network
filesystem would introduce lock, cache, and failure semantics that the tests do
not model.

This assumption appears in production documentation and validation rather than
being hidden as a performance tuning choice. An intern should treat topology as
part of the proof boundary.

## Authorization artifact lifecycle

`beginAuthorizeLifecycle` starts a SQL transaction before Fosite constructs the
authorization response. The context carries `authorizeLifecycle`. Every
authorization artifact method calls `authorizeExec`, which refuses to write
without the lifecycle transaction.

The participating writes are:

- `CreateAuthorizeCodeSession`;
- `CreatePKCERequestSession`;
- `CreateOpenIDConnectSession`;
- terminal consumption of the associated interaction.

The lifecycle completion closure receives whether response construction
succeeded. On failure it rolls back. On success it executes the pre-commit
failpoint, consumes the interaction with the selected outcome, and commits.

The positive test proves one row exists in each protocol table and the
interaction is consumed. The negative table injects before and after every
mutation and proves all protocol counts and terminal state remain at baseline.

## Why interaction terminal state joins issuance

If interaction approval committed before code/PKCE/OIDC persistence, a storage
failure would consume the user's one-time decision without producing a usable
code. Retrying the form would be rejected as replay.

If code persistence committed before interaction consumption, two submits could
create multiple artifact sets or leave a pending interaction after issuance.

Joining terminal state and artifacts in one transaction establishes:

```text
committed approved interaction
    iff
complete committed authorization artifact set
```

This equivalence applies to the SQLite production path. The development memory
composition has different dependency behavior and remains a separate evidence
boundary.

## Fosite `storage.Transactional`

The saved `fosite-transactional-storage-api.md` documents three methods:
`BeginTX`, `Commit`, and `Rollback`. Fosite's token handlers call
`MaybeBeginTx`, `MaybeCommitTx`, and `MaybeRollbackTx` when the store implements
that interface.

`sqlFositeStore` declares:

```go
var _ storage.Transactional = (*sqlFositeStore)(nil)
```

`BeginTX` rejects nested token lifecycles, runs a named pre-begin hook, begins a
SQL transaction, and returns a derived context containing `tokenLifecycle`.
`Commit` resolves that context, runs the pre-commit hook, and commits.
`Rollback` resolves the same context and rolls back.

The context is the transaction capability. A method with only the base store
cannot accidentally use the transaction unless it receives the derived context.
Conversely, participating mutation helpers refuse missing transaction state.

## `tokenExec` versus `tokenOrDirectExec`

`tokenExec` requires an active token lifecycle. It is used for issuance
mutations that must join Fosite's transaction. Its named hook points let tests
fail before or after a specific write.

`tokenOrDirectExec` uses the context transaction when present and the base
database otherwise. It exists for revocation operations that Fosite can invoke
outside issuance, including reuse handling. This dual behavior is deliberately
narrow. Making every method fall back to direct execution would silently allow
future issuance calls to escape the transaction.

The `tinyidpprotocollifecycle` analyzer checks concrete adapter methods for
required helper use. Comments mark intentionally transaction-scoped methods so
the more general atomicity analyzer does not produce false positives.

## Code redemption in exact operations

Fosite validates the code, client, redirect, PKCE verifier, expiry, and session.
Within the token lifecycle, tiny-idp executes:

1. `InvalidateAuthorizeCodeSession` updates the exact signature to inactive.
2. `CreateAccessTokenSession` inserts the new access requester.
3. `CreateRefreshTokenSession` inserts the refresh requester when the grant
   includes a refresh token.
4. `Commit` makes the generation durable.

The code signature, not the raw token, is the store key. Persisted requester data
contains the normalized client, granted scopes/audience, session claims, expiry
map, subject, and request metadata needed by later introspection and refresh.

The transaction must finish before `WriteAccessResponse`. An HTTP client can
disconnect after commit; that creates an availability ambiguity but not partial
database state. Retrying the code is rejected because it was consumed, even if
the client never observed the response. No database design can retract a secret
that may already have crossed the network boundary.

## Code-redemption failpoint matrix

The eight injected positions are:

1. before transaction begin;
2. before code invalidation;
3. after code invalidation;
4. before access creation;
5. after access creation;
6. before refresh creation;
7. after refresh creation;
8. before commit.

For every point, the test begins from a freshly authorized code, arms the hook,
posts to `/token`, and then queries all three durable conditions.

Expected failure state:

```text
active authorization code count = 1
access-token row count = 0
refresh-token row count = 0
new token lifecycle commit events = 0
```

The test does not accept “token endpoint returned an error” as sufficient. It
checks the authority graph after the response.

## Refresh rotation in exact operations

`RotateRefreshToken` receives the request ID and presented refresh signature.
It conditionally updates the exact row:

```sql
UPDATE fosite_refresh_tokens
SET active = 0
WHERE request_id = ? AND signature = ? AND active = 1
```

The affected-row count is the semantic result. One row means this operation won
the active generation. Zero rows means the token is no longer rotatable. The
adapter maps zero rows to Fosite's serialization failure.

The same lifecycle removes the associated old access state and creates
replacement access and refresh rows. The old and new requester records preserve
request-family identity for Fosite's reuse response.

## Refresh failpoint matrix

The ten injected positions are:

1. before transaction begin;
2. before rotation;
3. after old-refresh deactivation;
4. after old-access deletion;
5. after rotation;
6. before replacement access creation;
7. after replacement access creation;
8. before replacement refresh creation;
9. after replacement refresh creation;
10. before commit.

For every failure, the original refresh and access rows remain active and no
replacement generation survives. The test disarms the hook and retries the
original refresh token. Retry must produce both replacement credentials.

Retryability is stronger than rollback row counts: it proves the old capability
still participates in the complete Fosite path.

## Research influence: lineage-driven fault injection

The lineage-driven fault-injection paper asks which failures can affect an
observed result and injects along those dependencies. tiny-idp's implementation
does not reproduce the full research system. It adopts the actionable principle:
name every mutation dependency in a security lifecycle and inject immediately
before and after it.

The pre/post distinction matters. A failure before a write proves no work was
performed. A failure after a write but before commit proves rollback erases a
real intermediate state.

The hook is test-only and synchronous. Production callers leave it nil. This
keeps instrumentation from becoming a runtime control plane.

## Atomicity versus durability versus isolation

Atomicity states that participating changes commit together or not at all.
Durability states that committed changes survive process failure according to
the database and filesystem contract. Isolation constrains what concurrent
transactions observe. Consistency is the set of application invariants preserved
by transitions.

The tests target each differently:

- failpoint rollback targets atomicity;
- restart and backup/restore tests target durability;
- conditional updates and concurrent histories target isolation/serialization;
- row, model, and trace assertions target application consistency.

Using “transactional” without naming the property hides important gaps.

## The interaction concurrent object

The abstract object supports `Create`, `Get`, and `Consume(outcome, now)`.
Consume has these sequential results:

```text
absent          -> not_found
pending+fresh   -> accepted and terminal
pending+expired -> expired
terminal        -> already_consumed
```

The memory implementation serializes access and copies maps/slices. SQLite uses
a conditional terminal update. The Porcupine model ignores storage mechanism and
checks observable accepted/rejected results.

Sixteen clients invoke consume concurrently. Call and return logical times come
from an atomic counter. A valid linearization must place exactly one accepted
operation before every already-consumed result while preserving real-time order
for non-overlapping calls.

## The refresh concurrent object

The refresh model initially treated the operation as a one-time rotate:

```text
active -> one success -> inactive
inactive -> failure
```

Eight concurrent HTTP clients presented the same token. Exactly one rotation
response succeeded, and Porcupine found a legal ordering. The initial database
assertion expected the winner's replacement refresh row to remain active. It
failed repeatedly: active refresh and access counts ended at zero.

Source and trace inspection showed a second protocol operation. Requests that
validated after the winner committed saw the old token inactive. Fosite treated
that as reuse and revoked the family, including the winner's replacement.

The corrected interpretation is:

```text
Rotate(old) -> new generation
Reuse(old)  -> revoke family containing new generation
```

The concurrent rotate history is linearizable. The later reuse response is an
intentional security transition outside the minimal rotation model. The final
database assertion documents the availability consequence instead of weakening
the one-time property.

## Research influence: linearizability

Herlihy and Wing define a concurrent history as linearizable when it can be
extended and reordered into a legal sequential history that preserves the
real-time order of non-overlapping operations. The property is local to objects
and composes, which makes it suitable for interaction and refresh abstractions.

Porcupine implements faster checking by partitioning models where possible. The
tiny-idp histories are intentionally small enough for deterministic CI and are
partitioned by one object/family in each test.

The model is part of the evidence. If it omits family reuse, a green verdict
does not prove reuse semantics. The failed final-state assertion exposed this
abstraction gap.

## Race detector, schedule control, and histories

The race detector instruments memory access. It can find unsynchronized map,
pointer, or field access in executed schedules. It does not know that exactly one
approval is legal.

Scheduling probes and barriers increase overlap around the operation of
interest. They do not enumerate every schedule. CHESS research motivates
systematic schedule exploration, but the current project uses bounded goroutine
coordination plus repeated/shuffled/race runs.

Operation histories preserve semantic evidence after execution. Call time,
return time, input, and output are sufficient for Porcupine's model. SQL traces
and security events provide additional diagnosis but are not inputs to the
current checker.

## SQLite migration ledger

Schema migrations have ordered version numbers and recorded checksums. Open
applies missing migrations transactionally. A failed migration is not recorded.
A checksum mismatch rejects open. A database with a newer schema than the binary
is refused.

These rules prevent three recovery hazards:

- partial schema presented as complete;
- edited historical migration silently accepted;
- older binary interpreting newer state.

Migration tests inject failure and corruption. Doctor reports the exact schema
and verifies clients and keys after open.

## Online backup

Copying the SQLite main file is unsafe in WAL mode because committed pages may
reside in the WAL. The review originally built a probe that demonstrated a
file-copy backup could open while silently omitting committed data.

The production backup uses SQLite's online backup API. It copies a consistent
snapshot while the source remains open, then verifies the resulting database.
Publication uses a temporary path and atomic replacement so cancellation or disk
failure does not destroy the last good backup.

The manifest records:

- schema version;
- migration checksums;
- table counts;
- active signing-key identifiers.

Verification opens read-only and compares the manifest. A backup is not trusted
because a file exists or SQLite accepts its header.

## Offline restore and rollback

Restore requires the active provider/store to be stopped. It verifies the backup
before replacement, preserves the current database under a rollback name,
publishes the restored file, and runs doctor afterward.

The release drill proves that a client created after backup disappears after
restore, while the backed-up client, credential, and active key remain. It then
runs downgrade refusal and signing/token rotation tests.

Recovery evidence is temporal:

```text
known-good backup
  -> later mutations
  -> stop service
  -> verify backup
  -> preserve rollback
  -> restore
  -> doctor
  -> verify expected state boundary
```

## Account lockout atomicity

Failed-login accounting is another compound transition. Concurrent failures must
not lose increments or compute lockout from stale values. The atomic store method
updates count, failure window, and locked-until state together.

The concurrency test launches failed logins and asserts the expected count. This
is not token issuance, but it applies the same principle: policy state derived
from multiple fields belongs in one transaction.

## Signing-key rotation atomicity

Normal rotation creates or activates the new key while retiring the prior active
key. The store refuses to retire the final active key. The compound result
returns both active and retired records so audit can describe the committed
transition.

Retired verification keys remain published through the maximum ID-token lifetime
plus clock skew. Maintenance later removes them. Emergency purge is a separate
operation with stricter preconditions and different availability semantics.

## Consent and transaction boundary

Consent persistence remains a deliberate question. Consent can be an idempotent
prerequisite outside Fosite issuance, or it can join a wider transaction through
an optional transactional policy contract. The current design treats protocol
artifact atomicity as proved and consent side effects as a separate review item.

An intern must not infer that every call inside `resumeAuthorize` joins the SQL
transaction. Follow the actual context and interface.

## Post-commit failures

Some failures occur after durable commit:

- audit delivery can fail;
- security-event delivery can fail;
- HTTP response write can fail;
- the client can disconnect;
- the process can crash before logging completion.

Rollback is no longer a valid response because durable authority may exist.
The system records delivery counters and readiness where appropriate. Callers
must reconcile ambiguous admin outcomes rather than blindly retry non-idempotent
mutations.

## Transaction review method

For every proposed multi-row change:

1. Name the protocol or policy fact that becomes true.
2. List every durable row that represents that fact.
3. Identify the coordinator that knows the complete mutation set.
4. Place begin/commit ownership at that coordinator.
5. Require participating methods to use the transaction capability.
6. Identify conditional updates and affected-row semantics.
7. Enumerate failures before and after every mutation.
8. Define retry behavior after rollback.
9. Define behavior after commit but before response delivery.
10. Record an event only after the authoritative commit.

## Common incorrect designs

### Per-method transactions

Each method can be atomic while the protocol response is partial. The unit must
be the complete authority transition.

### Best-effort compensation

Compensation introduces another mutation that can fail. It cannot retract a
credential already observed by a client. Use a transaction before exposure.

### Direct-execution fallback

Allowing issuance methods to write directly when context transaction state is
missing makes future callers silently unsafe. Fail closed and annotate narrow
cleanup exceptions.

### Status-only fault tests

An HTTP error does not establish rollback. Inspect every durable artifact and
the committed event stream.

### Race-free equals correct

Locks can protect an implementation that permits two semantic successes. Check
histories against an abstract object.

### Backup equals copied file

SQLite WAL invalidates ordinary copy assumptions. Use the database backup API
and verify the published result.

## Decision records

### DS-1: protocol coordinator owns transaction

- **Decision:** use Fosite lifecycle hooks and `storage.Transactional` rather
  than private repository transactions.
- **Reason:** Fosite knows the complete mutation sequence.
- **Consequence:** context propagation is a security boundary.

### DS-2: conditional update defines rotation

- **Decision:** update exact `(request_id, signature, active=1)` and require one
  affected row.
- **Reason:** concurrent use needs an explicit winner.
- **Consequence:** zero rows becomes a typed serialization failure.

### DS-3: enumerate pre/post mutation failpoints

- **Decision:** use named synchronous test hooks at lifecycle boundaries.
- **Reason:** rollback must be observed after real intermediate writes.
- **Consequence:** claim scope is the enumerated matrix.

### DS-4: model concurrent objects

- **Decision:** check operation histories with Porcupine in addition to race
  detection.
- **Reason:** semantic exactly-once behavior is not a memory-race property.
- **Consequence:** models and observation abstractions require review.

### DS-5: one supported SQLite topology

- **Decision:** one active process, local filesystem, one connection.
- **Reason:** current durability/isolation evidence assumes that topology.
- **Consequence:** active/active requires a new design, not configuration.

## Extended exercises

1. Draw the authorization artifact transaction with every hook and row.
2. Draw the code redemption transaction and mark response exposure.
3. Draw the refresh rotation transaction and the later reuse transition.
4. Explain why `tokenOrDirectExec` is safe only for specific methods.
5. Add a hypothetical ID-token persistence row and update the failpoint matrix.
6. Define a Porcupine model that includes refresh-family reuse.
7. Explain why a passing history can coexist with an incorrect final-state
   assertion.
8. Compare migration checksum mismatch with future-schema refusal.
9. Demonstrate why main-file copy can omit WAL commits.
10. Explain recovery behavior if doctor fails after restore publication.
11. Classify audit failure after commit as an atomicity or delivery problem.
12. Review whether consent persistence should join issuance.
13. State the topology assumptions required by each SQLite test.
14. Explain what disk-full failpoints do not simulate.
15. Design a crash test between commit and HTTP write.

## Chapter review checklist

- Can the reader identify capability rows and derivation edges?
- Can the reader distinguish entity atomicity from protocol atomicity?
- Can the reader explain Fosite transaction propagation?
- Can the reader enumerate all redemption and rotation failpoints?
- Can the reader state rollback and retry oracles?
- Can the reader define the two concurrent abstract objects?
- Can the reader explain refresh winner revocation?
- Can the reader distinguish race freedom and linearizability?
- Can the reader explain WAL-safe backup and offline restore?
- Can the reader identify post-commit ambiguity and remaining consent risk?

## Fosite SQL method lifecycle catalog

This catalog prevents review from treating all adapter methods as equivalent.

### Authorization-code session methods

`CreateAuthorizeCodeSession` persists requester state through `authorizeExec` and
therefore requires the authorization lifecycle. `GetAuthorizeCodeSession` reads
and restores active requester state. `InvalidateAuthorizeCodeSession` mutates
through `tokenExec` during code redemption.

The same logical code participates in two transactions at different times:
creation joins authorization issuance; invalidation joins token issuance.

### PKCE session methods

`CreatePKCERequestSession` joins authorization lifecycle. Get restores the
request for verifier validation. Delete participates in cleanup according to
Fosite sequencing. PKCE state must remain consistent with code state.

### OpenID Connect session methods

Create joins authorization lifecycle. Get restores ID-token claims/headers and
request state. Delete follows code lifecycle cleanup. OIDC state must not survive
as an independently usable capability when code creation rolls back.

### Access-token session methods

Create joins token lifecycle and records requester by signature. Get supports
introspection/UserInfo. Delete may join token rotation or direct revocation
depending on Fosite context.

### Refresh-token session methods

Create joins token lifecycle and records associated access signature. Get
requires active state. Delete/revoke may execute in transaction or direct reuse
cleanup. Rotate requires active lifecycle and conditional exact-row update.

### JWT replay methods

Client assertion and JTI methods enforce one-time JWT identifiers when those
features are used internally. They need their own expiry/cleanup and atomicity
review if public assertion profiles are enabled.

### Client authentication

`GetClient` loads public client metadata and confidential secret hashes.
`Authenticate` behavior is constrained by Fosite client handling. Persisted
client TTLs influence token lifetimes and maintenance derivation.

## Transaction-context threat model

The context value is an unexported typed key, so ordinary callers cannot collide
by string. The contained transaction is available only to code receiving the
derived context.

Threats and controls:

- **missing context:** lifecycle-required helper returns error;
- **nested begin:** `ErrNestedTransaction`/explicit rejection;
- **wrong helper:** lifecycle analyzer and tests;
- **commit ignored:** ignored-security-error analyzer;
- **rollback failure:** returned/recorded error path requires review;
- **context cancellation:** database call observes context and lifecycle rolls
  back;
- **reuse of committed transaction:** SQL driver returns error; lifecycle scope
  remains one request;
- **direct base DB access:** code review/analyzer search for mutation methods.

## Failure-state ledger

### Before begin

No transaction exists and no mutation occurs. The original credential and
interaction remain unchanged. Retry is safe subject to ordinary expiry/race.

### After first mutation

The SQL transaction contains a real intermediate state invisible after rollback.
This proves participating helper uses the transaction rather than base DB.

### Before commit

All intended writes may exist inside the transaction. Injected failure and
rollback must restore the complete baseline. This is the strongest named
rollback point.

### Commit failure

Database commit error is returned. Depending on driver failure, outcome can be
ambiguous; callers must not claim success. Tests cover controlled errors, while
platform recovery remains operational evidence.

### After commit, before event

Durable authority exists. Event delivery failure cannot roll it back. Delivery
counter and audit/reconciliation apply.

### After event, before HTTP write

Durable state and evidence exist; client may not receive artifact. One-time state
prevents blind retry from duplicating authority.

### Partial HTTP write

Client may receive incomplete or complete secret. Server must assume exposure.
Database compensation cannot retract possession.

## SQL invariant worksheet

For each capability table record:

```text
primary/signature key
request/family key
active predicate
subject/client binding
scope binding
expiry
creation transaction
consumption/revocation transaction
retention/deletion policy
secret or signature storage
indexes supporting conditional transition
backup manifest count
restart restoration test
```

An intern should complete this for authorize codes, PKCE, OIDC sessions, access
tokens, refresh tokens, interactions, browser sessions, consents, signing keys,
credentials, and account-security state.

## Concurrency history worksheet

```text
Object key:
Initial abstract state:
Operation input:
Operation output:
Legal transition:
Invocation timestamp:
Return timestamp:
Real-time constraints:
Linearization point:
Final concrete state:
Operations outside model:
Checker result:
Repeated-run result:
```

The refresh worksheet must list reuse revocation under operations outside the
minimal rotate model.

## Isolation scenarios

### Two different interactions

Operations should proceed independently and may commit concurrently subject to
SQLite serialization. One cannot consume the other because keys differ.

### Same interaction

Conditional terminal transition yields one winner. Every loser observes terminal
state or serialization result.

### Two different refresh families

Families should rotate independently. A broader request-ID revocation query must
not cross client/subject family boundaries.

### Same refresh generation

One rotation wins. Other uses become failures/reuse and may revoke family.

### Backup during writes

Online backup captures one consistent snapshot. It may represent state before or
after a concurrent transaction, never a torn mix.

### Maintenance during protocol operations

Retention predicates must not delete active/unexpired state. Transaction/locking
behavior should preserve current operations. Current scheduling and tests provide
bounded evidence.

## Recovery failure catalog

### Backup destination exists

Write to temporary file, verify, then atomically publish according to overwrite
policy. Do not truncate known-good backup first.

### Disk full

Temporary publication fails and known-good destination remains. Test injects
write failure.

### Context cancellation

Backup aborts, cleans temporary state, and preserves publication.

### Corrupt backup

Verification rejects before restore. Manifest/database mismatch is evidence of
invalid artifact.

### Migration checksum mismatch

Open rejects historical schema whose recorded content differs from binary.

### Future schema

Older binary refuses database version above supported migration.

### Restore publication failure

Rollback copy must remain. Operator follows runbook and does not start provider
on unverified state.

### Doctor failure after restore

Service remains stopped. Preserve evidence and either repair under reviewed
procedure or restore rollback copy.

## Research-to-implementation provenance

### Database transaction theory

Atomicity and isolation motivated grouping complete capability transitions.
The local inference was to use the protocol orchestrator's transaction extension
rather than entity-local transactions.

### Fosite source/API

Observed `MaybeBeginTx` calls proved upstream already offered the correct token
lifecycle hook. The implementation gap was adapter conformance.

### Lineage-driven fault injection

Motivated mutation-dependency enumeration and before/after hooks. The project
implements a manual named matrix, not automatic lineage discovery.

### Linearizability

Provided the correctness criterion for one-time concurrent objects. Porcupine
checks bounded recorded histories against project models.

### CHESS

Motivated schedule control and repeated overlap. The project does not implement
systematic preemption-bounded exploration.

### Model-based testing

Motivated separating compact abstract state from memory/SQLite behavior. The
model is reviewed as part of evidence.

## Store-change review questions

1. Does the change create, transfer, rotate, revoke, or delete authority?
2. Which rows collectively represent the fact?
3. Which coordinator knows the full set?
4. Does context carry the correct transaction?
5. Which direct-execution path exists and why?
6. What conditional predicate chooses a winner?
7. What does zero affected rows mean?
8. What are all pre/post mutation failpoints?
9. What remains retryable after rollback?
10. What is ambiguous after commit?
11. Which event follows commit?
12. Which concurrent history model changes?
13. Which migration/index/retention changes?
14. Which backup/restore evidence changes?
15. Which analyzer fixture should change?

## Final durable-state competence test

The reader passes when given a new multi-row security transition and can produce:

- complete mutation set;
- transaction owner and context path;
- conditional winner predicate;
- typed error mapping;
- pre/post failpoint matrix;
- row/event/retry oracle;
- concurrent abstract model;
- post-commit ambiguity analysis;
- migration and recovery impact;
- supported topology and residual gap.

## Research map

- Linearizability supplies the concurrent correctness definition; see
  `sources/paper-faster-linearizability-checking.md` and `porcupine-linearizability.md`.
- Lineage-driven fault injection motivates failures at mutation dependencies;
  see `paper-lineage-driven-fault-injection-context.md`.
- Fosite's actual transaction contract is captured in
  `sources/fosite-transactional-storage-api.md`.
