---
Title: 2026-07-16 Device Production Ticket Reconciliation
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - oauth2
    - oidc
    - security
    - testing
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp-xapp/phase5_test.go
      Note: Application-level device token and TLS resource-server evidence.
    - Path: internal/fositeadapter/device_authorization_test.go
      Note: Endpoint, browser decision, token/UserInfo/introspection/replay security evidence.
    - Path: internal/fositeadapter/device_token_handler.go
      Note: Custom Fosite RFC 8628 handler and transactional redemption boundary.
    - Path: internal/fositeadapter/sqlstore_test.go
      Note: SQLite failpoint and restart-safe redemption evidence.
    - Path: pkg/sqlitestore/backup_test.go
      Note: Device-grant backup/restore coverage.
    - Path: pkg/sqlitestore/store.go
      Note: Durable device poll/decision/consume state transitions.
ExternalSources: []
Summary: Evidence-based reconciliation of the July 14 device-production phase ledger with the implemented strict provider, xapp device client, and remaining release gaps.
LastUpdated: 2026-07-16T00:00:00Z
WhatFor: Prevent a stale task list from understating implemented device-grant security work or overstating release readiness.
WhenToUse: Before resuming this ticket, prioritizing release work, or deciding whether a device feature is already covered by another ticket.
---


# 2026-07-16 Device Production Ticket Reconciliation

## Result

The previous ledger showed all Phase 5 work as open. That was obsolete. The
strict provider now has the durable RFC 8628 path described by the original
design: explicit client grant capability, SQLite-backed hashed device grants,
browser verification with fresh credentials and CSRF, a custom Fosite device
token handler, transactional consumption/persistence, discovery metadata, and
an xapp device client that uses the provider through real HTTP/TLS flows.

This reconciliation marks five completed tasks and narrows five broad pending
tasks. It does **not** declare the feature production-release-ready. The
remaining work is release/assurance work, not absence of the core grant.

## Evidence matrix

| Original task | Reconciled state | Evidence |
| --- | --- | --- |
| `szkc` custom Fosite token handler | Complete | `internal/fositeadapter/device_token_handler.go`; registered in `Provider` construction. Commit `8e41d42`. |
| `akdz` polling and terminal error mapping | Complete | `deviceTokenHandler.HandleTokenEndpointRequest` maps pending, slow-down, denial, expiry, consumed/replay, malformed forms, and wrong client; durable `PollDeviceGrant` tests establish the state outcomes. |
| `9tn5` atomic consumption and normal token persistence | Complete | `PopulateTokenEndpointResponse` starts Fosite storage transaction, conditionally consumes SQLite grant, persists access/eligible refresh sessions, and commits once. `TestSQLiteDeviceTokenRedemptionFailpointsRollbackGrantAndTokens`. |
| `5jr3` token lifecycle verification | Partially complete | Device flow has UserInfo, authenticated introspection, audience, token replay, and restart tests. It lacks explicit device refresh-token and signing-key-rotation test coverage. |
| `i1k8` persistence failpoints | Partially complete | Device flow tests begin/consume/access/commit failure points. It lacks device-flow refresh-token persistence and retry-after-failure coverage. |
| `tn4d` durable/general limiting | Complete | Device grant `NextPollAt` makes poll backoff durable; provider enforces limiter dimensions for creation, code entry, verification authentication, and token requests. |
| `66u4` operations/observability | Partially complete | `embeddedidp` readiness and maintenance include durable grants; the completed xapp ticket has an operator runbook. Device-specific metrics/dashboard evidence and strict-provider runbook are still missing. |
| `7lef` adversarial suite | Partially complete | SQLite restart/replay and backup/restore coverage exist. Dedicated device fuzz/race suites and a provider-independent external CLI smoke client remain. |
| `w7u4` AST analyzers | Open | Existing `auditlint` validates generic lifecycle/parse/secret-adjacent invariants but has no device-specific analyzer suite. |
| `g4gk` discovery advertisement gate | Complete | Production discovery advertises the endpoint and device-code grant; hardening and embedded-provider discovery tests cover it. Core exchange is no longer a stub. |
| `ukf8` public documentation | Open | README and several user-facing pages still describe strict device authorization as mock-only or in-progress; these claims conflict with current implementation and must be corrected with appropriate release caveats. |
| `ue9c` independent review/release checklist | Open | No independent security review or exact-release-candidate sign-off evidence was found. |

## What actually changed after the old ledger

The ticket's own diary already records the decisive Phase 5 implementation,
but its Markdown task checkboxes were never updated. The relevant committed
sequence is:

```text
a3ec9e1  explicit per-client grant capabilities
27e45ad  durable DeviceGrant state machine
8a1153c  strict device-authorization endpoint
fe6a230  browser verification flow
8e41d42  transactional Fosite device token exchange
542b417  SQLite single-connection signing-deadlock correction
8405da8  SQLite browser-continuation coverage
6b819f1  restart-safe redemption and replay rejection
d5c7647  registered-audience binding
d196aeb  opaque-token introspection lifecycle hardening
748fef8  xapp two-user/TLS/lifecycle device resource-server matrix
```

The provider's decisive security boundary is:

```text
POST /device_authorization
  -> HMAC(device_code), HMAC(user_code), durable pending grant

GET/POST /device
  -> opaque browser-bound interaction + CSRF + fresh password
  -> atomic grant approval or denial

POST /token (device_code)
  -> durable polling outcome
  -> Fosite token values/session
  -> one SQLite transaction:
       conditional approved -> consumed transition
       access token session (+ eligible refresh session)
       commit
```

The xapp work is complementary rather than a replacement for strict-provider
verification. It proves a public device client can obtain a token and use it
against a host-owned bearer resource API with exact audience/scope enforcement,
two distinct subjects, password-change invalidation, and initialized TLS.

## Remaining plan

1. **Close Phase 5 coverage gaps.** Add explicit successful device refresh and
   signing-key rotation tests, then exercise refresh-session creation failpoints
   and a successful retry after each injected token persistence failure.
2. **Finish strict-provider operations.** Define device-specific low-cardinality
   metrics, alert conditions, and dashboard/runbook requirements. Reuse generic
   provider readiness/maintenance, but do not pretend they substitute for a
   device abuse/replay operating procedure.
3. **Run adversarial evidence.** Add stateful device fuzz/race coverage and a
   black-box CLI that discovers the strict provider, drives browser approval,
   polls, validates results, and is runnable against a release candidate.
4. **Extend static analysis.** Add only checks with a reliable syntactic
   signal: prohibited secret-like audit fields in device packages, bounded form
   parsing at device handlers, named device transitions, and handler/metadata
   registration parity.
5. **Correct public documentation and release evidence.** Make current docs
   distinguish the mock CLI from the strict embedded provider, then require
   independent review and signed-release gates before claiming production
   readiness.

## Verification run during reconciliation

The following focused suites passed on 2026-07-16:

```sh
go test ./internal/fositeadapter ./pkg/idpstore ./internal/store/memory \
  ./pkg/sqlitestore ./pkg/embeddedidp ./cmd/tinyidp-xapp -count=1
```

This is evidence of current correctness, not an external security review or
OIDF device-profile certification.
