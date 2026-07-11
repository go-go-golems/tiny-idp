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

## Research map

- Security automata motivate explicit permitted transition histories.
- Typestate motivates making legal operations depend on the current state.
- Model-based testing connects an abstract transition system to real execution.
- Runtime verification checks emitted histories against executable temporal
  properties.
- See `sources/paper-enforceable-security-policies.md`,
  `paper-typestate.md`, `paper-model-based-security-testing.md`, and
  `paper-runtime-verification-brief-account.md`.
