---
Title: Research Foundations for Identity Protocol Invariants, Atomicity, and Runtime Verification
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
    - Path: repo://internal/fositeadapter/interaction.go
      Note: Opaque interaction implementation derived from OAuth web-attacker and cross-message invariants
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: Porcupine concurrent history evidence
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Fosite transaction propagation and token rotation linearization point
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: Rapid executable state-machine evidence
    - Path: repo://internal/securitytrace/trace.go
      Note: Versioned runtime-verification event model and monitor
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/sources/fosite-transactional-storage-api.md
      Note: Upstream transaction contract captured with Defuddle
ExternalSources: []
Summary: Research-to-code map connecting OAuth/OIDC standards, formal protocol analysis, model-based testing, fault injection, runtime verification, and linearizability checking to tiny-idp's concrete security invariants and evidence.
LastUpdated: 2026-07-10T20:25:01.747476086-04:00
WhatFor: Understanding why the authorization and token lifecycle implementation has its current structure, which claims are standards-derived, which are engineering inferences, and how tests support those claims.
WhenToUse: Before reviewing or changing authorization interactions, token persistence, refresh rotation, trace instrumentation, state-machine tests, or release assurance gates.
---


# Research Foundations for Identity Protocol Invariants, Atomicity, and Runtime Verification

## Executive Summary

tiny-idp's security work is organized around protocol invariants rather than
individual handler branches. OAuth and OpenID Connect specify message-level
requirements, but production correctness additionally depends on local state
transitions: a validated authorization request must not be replaced by browser
input; a one-time continuation must have one terminal outcome; authorization
code redemption and refresh rotation must not expose partially committed token
families; and runtime evidence must be interpretable without recording secrets.

This document records the academic and standards basis for those decisions. It
distinguishes three forms of knowledge:

- **Normative requirement:** directly required by an RFC or OpenID specification.
- **Observed implementation fact:** established by Fosite or tiny-idp source and
  tests.
- **Design inference:** a local mechanism selected to satisfy or make testable a
  normative property.

The implemented authorization interaction is primarily a design inference from
the OAuth attacker model and exact redirect/code binding requirements. The SQL
lifecycle transaction is an implementation of the atomic abstract operation
assumed by state-machine and linearizability reasoning. Property testing,
fault-injection, trace monitoring, and Porcupine are complementary: they explore
input sequences, storage failures, observed event histories, and concurrent
operation order respectively.

## Problem Statement

An identity provider composes several stateful mechanisms whose correctness
cannot be established by endpoint happy-path tests alone:

1. Browser interactions span multiple HTTP requests and cross a trust boundary.
2. OAuth artifacts are one-time or rotating capabilities whose storage mutations
   must be indivisible at externally visible boundaries.
3. The state space contains time, mutable clients/users, concurrency, replay,
   parsing failures, and injected storage faults.
4. Security audit logs can show what happened, but an event stream requires an
   explicit monitor semantics before it becomes verification evidence.
5. Static analysis can prevent known code shapes from recurring, but cannot prove
   temporal behavior or database linearizability.

The review comments about forced reauthentication and password lifecycle were
therefore treated as instances of a larger class: required security actions were
represented implicitly in control flow and could disappear between requests.

## Proposed Solution

The assurance architecture uses layered executable specifications:

```text
RFC / OIDC requirement
        |
        v
typed invariant and state-machine transition
        |
        +----------> Go AST analyzer (forbidden local code shapes)
        |
        +----------> generated/fuzz action sequences
        |
        +----------> fault-injected storage lifecycle
        |
        +----------> structured security-event trace monitor
        |
        +----------> concurrent history + linearizability checker
```

No single layer is treated as a proof of the entire system. Each layer has a
declared claim boundary and counterexample format.

## Research-to-code map: authorization interaction

### Standards and research basis

RFC 9700 defines a web attacker able to initiate attacker-chosen navigations and
operate arbitrary endpoints. It requires exact redirect URI matching and
describes authorization-code injection, leakage, and replay threats. RFC 6749
requires binding an authorization code to its client and redirect URI. OpenID
Connect adds `prompt`, `max_age`, nonce, authentication time, and consent error
semantics.

The formal OAuth and OpenID Connect analyses saved with this ticket model the
protocol as a transition system under a web attacker. Their useful engineering
lesson is not that the Go implementation has been formally verified; it is that
security properties concern relationships across messages and principals, not
the local validity of one parsed request.

### Engineering inference

Once Fosite validates the initial request, tiny-idp stores a canonical server-side
representation and returns only a random handle. The resumed browser POST carries
native user actions and credentials, not authoritative protocol parameters. This
mechanism is stricter than signing a bag of hidden fields because it also supports
expiry, mutable-state revalidation, exactly-once consumption, and operational
inspection without exposing a bearer continuation in storage.

```text
GET /authorize
  -> Fosite validation
  -> canonical request + required actions
  -> HMAC-hashed opaque interaction record
  -> browser receives handle + handle-bound CSRF MAC

POST /authorize
  -> validate browser binding and expiry
  -> reconstruct only from canonical record
  -> satisfy fresh login / consent
  -> revalidate client, user, redirect, scopes, key
  -> atomic terminal consume and artifact issuance
```

### Concrete evidence

- `pkg/idpstore/types.go`: typed required actions and terminal outcomes.
- `internal/fositeadapter/interaction.go`: canonicalization, digest, bindings,
  generation hash, and reconstruction.
- `internal/fositeadapter/provider.go`: begin/resume state transitions.
- `internal/fositeadapter/interaction_hardening_test.go`: mutation, replay,
  forced-login, `max_age`, consent, expiry, and concurrency counterexamples.

## Research-to-code map: authorization persistence atomicity

### Database and protocol basis

OAuth describes code redemption as a one-time exchange. Fosite implements code,
PKCE, and OIDC sessions through separate storage calls. A protocol response is
only valid if those calls and interaction consumption have one commit point.
Lineage-driven fault injection motivates placing failures before and after each
mutation rather than testing only a generic database error.

### Engineering inference

The SQL adapter propagates a transaction through the same context Fosite passes
to its storage handlers. The transaction first performs the conditional
interaction transition and then receives Fosite's code, PKCE, and OIDC writes.
The HTTP response is emitted only after commit. A write without the lifecycle
context fails closed.

The claim is deliberately scoped to the SQLite production path. Arbitrary
consent-policy side effects remain outside this SQL transaction and are recorded
as a residual review point.

### Concrete evidence

- `internal/fositeadapter/sqlstore.go`: lifecycle context, conditional consume,
  failpoints, commit, and rollback.
- `internal/fositeadapter/sqlstore_test.go`: seven rollback points and a positive
  one-row-per-artifact commit assertion.
- Commits `aedff3c` and `27c339e`: implementation and consent disclosure.

## Research-to-code map: token redemption and refresh rotation

### Standards and library basis

RFC 6749 defines authorization codes as short-lived, single-use credentials and
requires refresh tokens to remain bound to the client. RFC 9700 strengthens the
refresh-token requirement for public clients: deployments must sender-constrain
refresh tokens or use rotation so replay reveals a breach. The saved Fosite
`Transactional` API documents a concrete library contract: `BeginTX` returns a
context containing the transaction, storage methods recover it from that context,
and `Commit` or `Rollback` terminates it.

This directly shaped the implementation. Earlier tiny-idp code gave
`RotateRefreshToken` a private transaction but left Fosite's surrounding
`CreateAccessTokenSession` and `CreateRefreshTokenSession` calls outside it. That
was locally atomic but protocol-incomplete. Implementing Fosite's interface makes
the transaction boundary coincide with the library's complete abstract
operation.

### Code redemption transaction

```text
BeginTX
  invalidate authorization code
  create access-token session
  create refresh-token session (when offline_access applies)
Commit
```

Failures before and after every mutation and before commit must restore an active
authorization code and leave no access/refresh rows. This is stronger than merely
returning an OAuth error: it proves the client can retry a transient internal
failure without losing the one-time credential or receiving a partial token set.

### Refresh rotation transaction

```text
BeginTX
  conditionally deactivate exactly the presented active refresh token
  delete its prior access-token session
  create replacement access-token session
  create replacement refresh-token session
Commit
```

The conditional update is the rotation linearization point. If no row changes,
the operation returns Fosite's serialization failure and rolls back. Fault tests
show that every injected storage failure leaves the old refresh/access pair
active and permits a later retry.

### Concurrent replay result

A Porcupine history with eight simultaneous uses of one refresh token has one
successful rotation, so the rotation operation itself is linearizable. However,
requests that reach validation after the winning commit observe the old token as
inactive. Fosite interprets that as reuse and revokes the complete token family.
The final database state therefore has no active refresh or access token.

This is a security/availability tradeoff implied by rotation-based replay
detection: the authorization server cannot determine which concurrent caller is
legitimate. The result is consistent with RFC 9700's containment objective, but
clients must serialize refresh attempts and operators should recognize a burst of
concurrent refreshes as a family-revocation cause.

### Concrete evidence

- `sources/fosite-transactional-storage-api.md`: upstream context-transaction
  contract captured with Defuddle.
- `internal/fositeadapter/sqlstore.go`: `BeginTX`, `Commit`, `Rollback`,
  transaction-aware mutations, and conditional refresh rotation.
- `internal/fositeadapter/sqlstore_test.go`: eight redemption failpoints and ten
  rotation failpoints, including successful retry after every rotation failure.
- `internal/fositeadapter/linearizability_test.go`: Porcupine histories for
  interaction consume and refresh rotation/reuse.

## Research-to-code map: model-based testing and trace monitoring

QuickCheck-style state-machine testing separates a compact abstract state from
commands applied to the real system. The generator chooses operations based on
the model; a transition function predicts the next state; postconditions compare
observations with the system under test. This is appropriate for interactions
because create, get, expire, consume, replay, and mutation form a small abstract
machine despite many HTTP encodings.

Runtime verification complements generated tests. A versioned event stream is
partitioned by a parameter such as interaction ID, and a monitor checks each
slice against temporal rules. The monitor does not infer protocol secrets or
replay requests. It consumes transition facts emitted by authoritative Go code.

The first implementation uses:

- Rapid to generate up to 80 interaction-store commands per case and shrink any
  counterexample;
- a reference state containing `created`, `consumed`, and `expired`;
- a versioned secret-free event schema keyed by HMAC-derived interaction ID;
- an offline monitor that rejects missing required authentication/consent,
  duplicate terminals, artifacts before approval, and duplicate artifact commit.

This is executable specification, not a formal proof. It provides reproducible
counterexamples and makes the monitored invariant explicit enough for review.

## Design Decisions

### Decision RF-1: server-owned interaction instead of hidden continuation

- **Status:** implemented.
- **Decision:** store the validated request and required actions under a hashed
  random handle.
- **Rationale:** removes browser authority, enables expiry/replay control, and
  gives one explicit state-machine object.
- **Consequence:** all internal and external test drivers must parse the opaque
  handle; no compatibility fallback exists.

### Decision RF-2: transactional context propagation instead of compensation

- **Status:** implemented for authorization issuance; in progress for token
  issuance and refresh rotation.
- **Decision:** join Fosite storage operations to a real SQL transaction.
- **Rationale:** compensation cannot reliably retract a code or token already
  exposed, and failure during compensation creates a second fault path.
- **Consequence:** storage methods reject issuance outside the lifecycle.

### Decision RF-3: complementary executable assurance layers

- **Status:** partially implemented.
- **Decision:** use AST analysis, state-machine generation, fault injection,
  trace monitoring, and linearizability checking for distinct claim classes.
- **Rationale:** static rules do not express temporal properties; fuzzing does
  not determine legal concurrent order; logs without a monitor are not verdicts.
- **Consequence:** each tool must document its precision and evidence boundary.

## Alternatives Considered

- **Signed hidden fields:** prevents undetected mutation but does not naturally
  supply one-time consumption, mutable-state revalidation, or server inspection.
- **Handler-local booleans:** caused the original required-action disappearance
  and cannot survive multiple requests explicitly.
- **Best-effort cleanup after partial persistence:** rejected for capability
  issuance because cleanup can fail and an exposed token cannot be recalled from
  a client response.
- **One large end-to-end fuzz test:** insufficient because failures are hard to
  shrink and concurrency legality is not encoded.
- **Production scripting as verdict authority:** rejected because dynamically
  loaded code would enlarge the trusted computing base. Go owns verdicts; any
  future Goja layer should compile bounded verification plans or policy inputs.

## Implementation Plan

1. Complete Fosite `storage.Transactional` support for code redemption and refresh
   rotation, with named mutation and commit failpoints.
2. Define a small pure Go reference model for interactions and token families.
3. Generate legal and adversarial command sequences; preserve seeds and shrunk
   traces on failure.
4. Emit versioned, secret-free security events at state-transition boundaries.
5. Run an offline parametric monitor keyed by interaction, request, and token
   family IDs.
6. Record concurrent operation intervals and check them against sequential models
   using Porcupine.
7. Feed all local evidence into the exact-candidate release packet, then run
   hosted OpenID conformance.

## Open Questions

- Should consent policy persistence join the same transaction through an optional
  transactional policy contract, or remain an idempotent prerequisite?
- Which token-family identifier is stable, secret-free, and available at every
  refresh rotation event?
- Should trace monitoring ship in production, test binaries, or both with
  different sinks?
- What history size and partition key keep linearizability checks fast in CI?
- Which remaining Goja verification hooks add value without becoming protocol
  verdict authority?

## References

- `sources/rfc6749-oauth2-authorization-framework.md`
- `sources/rfc6750-bearer-token-usage.md`
- `sources/rfc9700-oauth-security-bcp.md`
- `sources/openid-connect-core-1.0-errata2.md`
- `sources/paper-formal-security-analysis-oauth2.pdf`
- `sources/paper-formal-security-analysis-openid-connect.pdf`
- `sources/paper-lineage-driven-fault-injection.pdf`
- `sources/paper-model-based-security-testing.pdf`
- `sources/paper-runtime-verification-brief-account.md`
- `sources/porcupine-linearizability.md`
- `sources/paper-faster-linearizability-checking.md`
- `sources/paper-faster-linearizability-checking.pdf`
- `sources/fosite-transactional-storage-api.md`
- `sources/rapid-property-testing.md`
- `design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`
- `reference/01-implementation-diary.md`
