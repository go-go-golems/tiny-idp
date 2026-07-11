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
