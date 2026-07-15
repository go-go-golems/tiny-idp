# Tasks

## TODO

- [x] Phase 0: Record grant-type, code entropy, lifetime, polling, refresh, hashing, and UI decisions <!-- t:vkvx -->
- [x] Phase 0: Convert every documented device security invariant into a named test specification <!-- t:zzjv -->
- [x] Phase 0: Define secret-free audit events, reason codes, and low-cardinality metrics <!-- t:ky29 -->
- [x] Phase 1: Add explicit allowed grant types to the public client model and validation <!-- t:2epf -->
- [x] Phase 1: Add SQLite migration and deterministic backfill for existing browser clients <!-- t:6hxv -->
- [x] Phase 1: Update BrowserClient, DeviceClient, bootstrap drift, admin CLI, and Fosite client adaptation <!-- t:2ts2 -->
- [x] Phase 1: Add client capability migration, validation, and negative authorization tests <!-- t:3cml -->
- [ ] Phase 2: Define DeviceGrant records, statuses, poll/decision/consume request and result types <!-- t:u1bx -->
- [ ] Phase 2: Add named device store operations to Store and transaction-scoped TxStore <!-- t:6cl8 -->
- [ ] Phase 2: Implement memory-store device operations with invariant tests <!-- t:ulnd -->
- [ ] Phase 2: Add constrained SQLite device-grant schema, indexes, and atomic transitions <!-- t:klm8 -->
- [ ] Phase 2: Add transition concurrency, cancellation, rollback, restart, and expiry tests <!-- t:y2iq -->
- [ ] Phase 2: Integrate device records with maintenance, backup verification, and restore tests <!-- t:n2k6 -->
- [ ] Phase 3: Implement domain-separated keyed hashing and code generators with collision retry <!-- t:xdur -->
- [ ] Phase 3: Implement strict bounded POST device authorization parsing and client authentication <!-- t:16og -->
- [ ] Phase 3: Enforce device grant capability and requested-scope policy <!-- t:3cwn -->
- [ ] Phase 3: Persist grants, emit audit, return RFC 8628 response, and add endpoint tests <!-- t:gcwl -->
- [ ] Phase 4: Define typed bounded DeviceVerificationRenderer API and default pages <!-- t:ju8o -->
- [ ] Phase 4: Implement code entry, normalization, generic errors, and verification interaction binding <!-- t:r34p -->
- [ ] Phase 4: Integrate browser authentication, CSRF, client/scope display, and explicit decisions <!-- t:ubvd -->
- [ ] Phase 4: Add approve-deny races, replay, stale-session, renderer failure, and accessibility tests <!-- t:nrx9 -->
- [ ] Phase 5: Implement and register the custom Fosite device token endpoint handler <!-- t:szkc -->
- [ ] Phase 5: Map pending, slowdown, denial, expiry, wrong client, and replay to protocol errors <!-- t:akdz -->
- [ ] Phase 5: Atomically consume approved grants and persist access, ID, and optional refresh tokens <!-- t:9tn5 -->
- [ ] Phase 5: Verify device tokens through UserInfo, introspection, refresh, key rotation, and replay tests <!-- t:5jr3 -->
- [ ] Phase 5: Add transaction failpoints for every consumption and token persistence boundary <!-- t:i1k8 -->
- [ ] Phase 6: Add durable and general rate limits for creation, code entry, authentication, and polling <!-- t:tn4d -->
- [ ] Phase 6: Add readiness checks, retention reporting, metrics, dashboards, and operator runbook <!-- t:66u4 -->
- [ ] Phase 7: Add pure reference model and generated SQLite action-sequence comparison harness <!-- t:erz1 -->
- [ ] Phase 7: Add fuzzers, race suite, restart suite, backup-restore suite, and external CLI smoke client <!-- t:7lef -->
- [ ] Phase 7: Extend Go AST analyzers for secret fields, bounded parsing, named transitions, and handler assertions <!-- t:w7u4 -->
- [ ] Phase 8: Advertise device_authorization_endpoint only after the complete implementation passes gates <!-- t:g4gk -->
- [ ] Phase 8: Update embedding, admin, security profile, discovery, example, and release documentation <!-- t:ukf8 -->
- [ ] Phase 8: Obtain independent security review and complete production release checklist <!-- t:ue9c -->
