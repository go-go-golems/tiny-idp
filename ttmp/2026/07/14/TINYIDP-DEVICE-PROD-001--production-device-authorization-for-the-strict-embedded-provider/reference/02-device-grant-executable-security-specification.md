---
Title: Device Grant Executable Security Specification
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - architecture
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/idpstore/types.go
      Note: Future durable device-grant and client-capability types
    - Path: pkg/idpstore/interfaces.go
      Note: Named atomic transition contracts
    - Path: internal/fositeadapter/provider.go
      Note: Strict endpoint and token-handler composition
    - Path: internal/server/device.go
      Note: Mock protocol test reference only
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
Summary: "Frozen Phase 0 decisions and named executable security specifications for strict durable RFC 8628 implementation."
LastUpdated: 2026-07-15T08:00:00-04:00
WhatFor: "Turn production device-grant design into explicit implementation and verification contracts."
WhenToUse: "Read before changing client capability, device state, endpoint, verification UI, Fosite handler, audit, or test code."
---

# Device Grant Executable Security Specification

## Purpose and status

This document freezes Phase 0 decisions for the strict Fosite-backed provider.
It does not claim an endpoint is implemented or advertised.
Each requirement must become a named Go test, database constraint, typed store operation, bounded handler check, or secret-free audit/metric contract before discovery exposes device authorization.

## Protocol constants

```go
const (
    GrantAuthorizationCode = "authorization_code"
    GrantRefreshToken      = "refresh_token"
    GrantDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
)
```

| Decision | Fixed policy |
| --- | --- |
| device code | 32 random bytes, opaque URL-safe encoding |
| user code | eight ambiguity-reduced uppercase symbols and one separator |
| lifetime | 10 minutes |
| initial poll interval | 5 seconds |
| slow down | add 5 seconds to durable poll interval |
| terminal outcomes | approved, denied, consumed; expiry derives from time |
| refresh | only with explicit capability and `offline_access` approval |

Raw device codes, user codes, access tokens, refresh tokens, passwords, and browser handles are never persisted, logged, audited, used as metric labels, placed in errors, or exposed through debug endpoints.

## Client capability contract

Every client has a non-empty normalized allowed-grant list after migration.
No compatibility default grants device capability.

| Profile | Required grants | Forbidden shape |
| --- | --- | --- |
| browser | `authorization_code`, `refresh_token` | no redirect URI or no PKCE |
| device | device-code grant; optional refresh by policy | redirect/post-logout URI and authorization-code grant |
| generic | host-selected explicit grants | implicit or empty grant list |

Historical clients are backfilled as browser clients only when their stored configuration matches that profile; ambiguous records block production startup for an explicit operator decision.

## State-machine contract

```text
pending --permitted poll--> pending/authorization_pending
pending --early poll--> pending/slow_down
pending --explicit authenticated approve--> approved
pending --explicit authenticated deny--> denied
pending or approved --expiry--> expired
approved --atomic Fosite token issuance--> consumed
```

Only these named operations may mutate device state:

| Operation | Precondition | Required atomic result |
| --- | --- | --- |
| `CreateDeviceGrant` | valid device client and scope | pending record with keyed hashes |
| `PollDeviceGrant` | matching client and code hash | pending, slowdown, approved, denied, or expired result |
| `DecideDeviceGrant` | pending, unexpired, authenticated, CSRF-bound explicit action | exactly one approved or denied decision |
| `ConsumeDeviceGrant` | approved, unexpired, matching client | consumed state plus Fosite token persistence in one transaction |

Expiry takes precedence over decision and consumption, and all time comparisons use the provider clock inside the transaction.

## Hashing and generator tests

```text
device_hash = HMAC(secret, "tinyidp/device-code/v1\\x00" || raw_device_code)
user_hash   = HMAC(secret, "tinyidp/user-code/v1\\x00" || normalized_user_code)
```

- `TestDeviceCodeGeneratorProducesURLSafeHighEntropyValues`
- `TestUserCodeNormalizationIsCanonicalAndAmbiguitySafe`
- `TestDeviceAndUserCodeHashDomainsDiffer`
- `TestDeviceCodeRawValueNeverAppearsInStoreOrAudit`
- `TestUserCodeRawValueNeverAppearsInStoreOrAudit`
- `TestDeviceCodeCollisionRetriesWithinBound`

## Endpoint and verification tests

`POST {issuer}/device_authorization` accepts bounded form input, an identified device-capable client, and permitted scopes; success is a no-store RFC 8628 JSON response.
The typed browser verification model displays canonical client/scopes/user code, requires strict authentication, CSRF, and explicit approve/deny, and returns a generic response for invalid/expired/used codes.

- `TestDeviceAuthorizationRejectsWrongMethodMalformedFormAndDuplicateParameters`
- `TestDeviceAuthorizationRejectsClientWithoutDeviceGrantCapability`
- `TestDeviceAuthorizationRejectsDisallowedScope`
- `TestDeviceAuthorizationResponseHasNoStoreHeaders`
- `TestDeviceVerificationInvalidCodeIsGeneric`
- `TestDeviceVerificationRequiresCSRFAndExplicitDecision`
- `TestDeviceVerificationApproveDenyRaceHasOneWinner`

## Token and operational contract

The custom Fosite handler accepts only the exact device-code grant type and keeps existing Fosite client validation and token response behavior.

| Result state | OAuth response |
| --- | --- |
| unknown or wrong client | `invalid_grant` |
| pending permitted poll | `authorization_pending` |
| pending early poll | `slow_down` with durable interval increase |
| denied | `access_denied` |
| expired | `expired_token` |
| approved first poll | standard Fosite token response and atomic consumption |
| replay | `invalid_grant` |

- `TestDeviceTokenPendingAndSlowDownAreDurable`
- `TestDeviceTokenRejectsWrongClientAndReplayedCode`
- `TestDeviceTokenConsumesExactlyOnceUnderConcurrentPollers`
- `TestDeviceTokenRollsBackConsumptionWhenFositePersistenceFails`
- `TestDeviceTokenIssuesAccessIDAndEligibleRefreshTokens`

Audit names are `device.authorization.created`, `device.verification.decided`, `device.token.polled`, and `device.token.issued`; each has finite reasons and no raw secret fields.
Metrics may count event/result/reason and bounded client profile, never code, subject, username, client ID, IP, or arbitrary scope.

Discovery remains unchanged until unit, SQLite migration/rollback/restart, concurrency, Fosite, fuzz/race/reference-model, backup/restore, external CLI, readiness, audit, and operator-runbook gates are green.
