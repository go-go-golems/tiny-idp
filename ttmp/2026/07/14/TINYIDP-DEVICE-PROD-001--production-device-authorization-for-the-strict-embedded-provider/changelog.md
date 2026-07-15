# Changelog

## 2026-07-15

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
