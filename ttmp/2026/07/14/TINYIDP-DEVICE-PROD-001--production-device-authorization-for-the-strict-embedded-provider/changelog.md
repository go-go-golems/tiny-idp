# Changelog

## 2026-07-15

Completed Phase 4 with a server-owned, browser-bound RFC 8628 verification
interaction at `GET|POST /device`. A typed and bounded renderer receives only
presentation data. A valid public user code creates an opaque continuation
whose stored reference is the hash, not the raw code. Both approval and denial
require fresh password authentication, CSRF, same-browser binding, an enabled
device-capable client, and an atomic consume-interaction plus decide-grant
transaction. Tests cover rendering, escaping, accessibility-relevant labels,
invalid-code non-oracles, invalid-credential retry, CSRF/browser binding,
replay, concurrent one-winner decisions, and renderer failure.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_verification.go — browser entry and decision protocol boundary
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/types.go — typed bounded device renderer contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/templates/device_verification.html — dependency-free default page
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_authorization_test.go — browser and adversarial flow tests

Completed Phase 0 by freezing grant capability, code secrecy, state transition,
polling, audit/metric vocabulary, and named verification contracts.

Completed Phase 1 by making OAuth grant permissions an explicit, validated,
durably migrated client property. Browser and device bootstrap profiles, the
admin client command, strict Fosite clients, review probes, and external
consumer fixtures now declare capabilities deliberately.

Completed Phase 2 with a durable, secret-free `DeviceGrant` state machine.
Named store transitions now cover creation, code lookup, polling, decision,
and one-time consumption in memory and SQLite, with constrained schema,
maintenance, backup/restore, restart, rollback, cancellation, and concurrency
coverage.

Completed Phase 3 with the bounded `POST /device_authorization` boundary.
Raw codes exist only while producing the response, then become domain-separated
keyed hashes in a durable pending grant. The endpoint authenticates confidential
clients, enforces capability and scope policy, retries collisions, emits
secret-free audit events, and returns RFC 8628 fields with no-store headers.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_codes.go — generation, canonicalization, and hash domains
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — strict request boundary and route registration
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_authorization_test.go — endpoint and secret-handling tests

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpstore/types.go — device state and typed requests/results
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpstore/interfaces.go — named transition contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/migrations/009_device_grants.sql — constrained durable schema
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/store.go — atomic SQLite transition predicates

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpstore/types.go — public grant-capability model
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/migrations/008_client_grant_capabilities.sql — deterministic legacy classification
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/bootstrap.go — browser and device profiles
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — strict adapter grant propagation

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/reference/02-device-grant-executable-security-specification.md — Phase 0 security/test contract

## 2026-07-14

- Initial workspace created


## 2026-07-14 - Production device design and complete embedded application

Preserved primary specifications, mapped RFC 8628 onto strict Fosite and durable-store architecture, added phased tasks, and replaced the incomplete embedded provider host with a same-origin Authorization Code plus PKCE relying party.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/embedded/main.go — Self-contained process composition
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/design-doc/01-production-device-authorization-analysis-design-and-implementation-guide.md — Intern-facing production device plan


## 2026-07-14 - Validated and published

Recorded the passing repository suite, scoped commits, clean docmgr audit, and successful bundled reMarkable publication.

## 2026-07-16

Reconciled stale Phase 5–8 device ledger with current strict-provider, durable-store, and xapp evidence; marked core implementation complete and narrowed remaining release gaps.
### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_token_handler.go — Core completed device token extension.
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-DEVICE-PROD-001--production-device-authorization-for-the-strict-embedded-provider/reference/03-2026-07-16-reconciliation.md — Evidence matrix and remaining plan.

## 2026-07-16

Close device token lifecycle coverage: refresh-token persistence failpoints now roll back and retry, and device ID tokens are verified through signing-key rotation (commit 704872f).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider_test.go — Reusable JWKS verifier supports no-nonce device ID tokens
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore_test.go — Device refresh failpoint/retry and signing-key-rotation evidence

