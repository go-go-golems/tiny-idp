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

## Research map

- Linearizability supplies the concurrent correctness definition; see
  `sources/paper-faster-linearizability-checking.md` and `porcupine-linearizability.md`.
- Lineage-driven fault injection motivates failures at mutation dependencies;
  see `paper-lineage-driven-fault-injection-context.md`.
- Fosite's actual transaction contract is captured in
  `sources/fosite-transactional-storage-api.md`.
