---
Title: Device Authorization Release Evidence and Independent Review Packet
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - oauth2
    - oidc
    - security
    - testing
    - operations
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/fositeadapter/device_token_handler.go
      Note: Custom Fosite device-code redemption and transaction boundary.
    - Path: internal/fositeadapter/device_authorization_test.go
      Note: Browser-to-token protocol tests.
    - Path: internal/fositeadapter/sqlstore_test.go
      Note: Token persistence failpoints and signing-key lifecycle evidence.
    - Path: pkg/sqlitestore/device_model_test.go
      Note: Generated public-API reference-model comparison harness.
    - Path: internal/fositeadapter/device_fuzz_test.go
      Note: Device code normalization and hashing fuzz targets.
    - Path: ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Repository AST device-invariant analyzers.
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
Summary: Exact evidence, command matrix, reviewer questions, and approval boundary for release of strict RFC 8628 device authorization.
LastUpdated: 2026-07-16T00:00:00Z
WhatFor: Gives an independent reviewer a bounded, reproducible packet rather than asking them to reconstruct the implementation from tickets and commit history.
WhenToUse: At release-candidate freeze, before enabling a device client, and when assigning the external security review.
---

# Device Authorization Release Evidence and Independent Review Packet

## Decision boundary

The strict provider implements RFC 8628 device authorization. That statement is
about source and executable evidence, not a self-issued production approval.
The remaining open ticket task is deliberately external: an independent reviewer
must examine the frozen candidate and sign the release checklist. This document
prepares that review; it does not silently replace it.

## Security architecture under review

```text
untrusted device                         authenticated browser
----------------                         ---------------------
POST /device_authorization               GET/POST /device
  authenticated client + scopes            opaque interaction + CSRF
  high-entropy device code                 fresh password
  human user code                          approve or deny
          |                                         |
          v                                         v
SQLite DeviceGrant: pending -- named decision --> approved/denied
          |
          v
POST /token (device_code)
  durable poll/backoff
  Fosite token construction
  one SQLite transaction: conditional consume + access/refresh persistence
  => consumed exactly once or no externally usable token family
```

The raw user and device codes are inputs only. Durable state contains
domain-separated keyed hashes. Audit and metrics must not carry either code or
token material.

## Evidence matrix

| Claim | Primary evidence | Reviewer should verify |
| --- | --- | --- |
| Client capability is explicit | client validation and bootstrap tests | a device client cannot use the flow without `GrantDeviceCode`. |
| Codes are durable and secret-safe | `DeviceGrant`, migration 009, hashing tests | no raw code reaches SQLite, audit, or Fosite persisted requester form. |
| Browser approval is bound and fresh | device verification tests | CSRF, interaction binding, fresh credentials, denial, and race behavior. |
| Token issuance is atomic | `device_token_handler.go`, `sqlstore.go`, failpoint tests | consume and Fosite token sessions share the SQL transaction. |
| Refresh policy is explicit | commit `704872f` tests | `offline_access` plus `GrantRefreshToken` is required; failure leaves retry possible. |
| Key rotation is safe | signing-key rotation test | old device ID token remains JWKS-verifiable; new one uses active `kid`. |
| State semantics agree with store | `device_model_test.go` | generated model sequences use public store API and include expiry/backoff/one-time use. |
| Malformed inputs and races are exercised | fuzz targets and SQLite `-race` command | fuzz corpus and concurrency test run on the frozen commit. |
| Static regression checks exist | `auditlint` device analyzers | analyzer fixtures and actual provider/store invocation pass. |
| Operations do not leak identity into labels | ticket metrics script/runbook | only allow-listed stage/outcome/reason labels are emitted. |

## Reproduction commands

Run at the exact candidate commit, with a normal Go build cache and no local
uncommitted source changes:

```sh
go test ./internal/fositeadapter -count=1
go test ./pkg/sqlitestore -run TestDeviceGrantGeneratedActionSequencesAgreeWithReferenceModel -count=1
go test -race ./pkg/sqlitestore -run 'TestDeviceGrant(SurvivesRestartAndConcurrentConsumptionHasOneWinner|GeneratedActionSequencesAgreeWithReferenceModel)' -count=1
go test ./internal/fositeadapter -run '^$' -fuzz 'Fuzz(NormalizeUserCode|DeviceAndUserCodeHashesAreDomainSeparated)' -fuzztime=10s -count=1
go test ./ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts -count=1
go test ./ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts/02-device-cli-smoke -count=1
go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -count=1
go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint ./internal/fositeadapter ./pkg/sqlitestore
```

For a release candidate, separately start the strict production host with
real TLS, run the independent smoke CLI, approve in a real browser, and retain
only sanitized command output plus the resulting durable audit records:

```sh
go run ./ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/scripts/02-device-cli-smoke \
  -issuer https://idp.example.test -client-id reviewed-device-client -scope 'openid profile'
```

The CLI prints the verification URI and human user code but deliberately never
prints or writes the device code, access token, refresh token, or ID token.

## Reviewer checklist

- [ ] Candidate commit, dependency lock state, and deployment configuration are
      recorded before review begins.
- [ ] `deviceTokenHandler` accepts exactly the device grant, requires client
      identification, rejects duplicate/missing device code and scope input,
      and maps every durable outcome to the RFC protocol error.
- [ ] Conditional consumption is in the same transaction as access and optional
      refresh session persistence; failure paths roll back before response.
- [ ] Browser decision is authenticated, CSRF-bound, interaction-bound, and
      does not turn a remembered browser session into device approval evidence.
- [ ] Client configuration grants `offline_access` only where a product owner
      explicitly accepts refresh-token risk.
- [ ] Signing-key retention exceeds all issued device ID-token verification
      lifetimes before retired keys are purged.
- [ ] Rate-limit address resolution is correct for the real proxy topology.
- [ ] Audit storage is durable, access-controlled, monitored by `/readyz`, and
      its collector/exporter does not expose high-cardinality identity labels.
- [ ] The production smoke run completes through real TLS and a real browser.
- [ ] The command matrix passes at the frozen candidate and the reviewer has
      inspected any skips, flakes, or environment differences.
- [ ] An independent reviewer records their name, date, candidate SHA, findings,
      and disposition below.

## External approval record

| Field | Required value |
| --- | --- |
| Candidate SHA | _pending release candidate_ |
| Deployment profile | _pending_ |
| Reviewer (not implementation author) | _pending_ |
| Review date | _pending_ |
| Findings / accepted risks | _pending_ |
| Smoke evidence location | _pending_ |
| Disposition | _pending: approve / reject / approve with constraints_ |

Until this table is completed by an independent reviewer, the ticket remains
active and the strict device grant must not be represented as release-approved.
