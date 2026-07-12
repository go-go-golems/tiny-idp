# Tasks

## Phase 0 — Contracts and fixtures

- [ ] Inventory all tiny-idp, hostauth, Durable Object, and product-owned state. <!-- t:bkinv -->
- [ ] Freeze versioned component and product backup manifest schemas. <!-- t:bkfmt -->
- [ ] Define exclusive state-root lock and offline quiescence contract. <!-- t:bklock -->
- [ ] Create golden, corrupt, truncated, traversal, link, and duplicate-path archive fixtures. <!-- t:bkfix -->

## Phase 1 — General tiny-idp component backup

- [ ] Implement tiny-idp SQLite and audit snapshot API in the general library. <!-- t:bkidp -->
- [ ] Implement tiny-idp component verification and staged restore. <!-- t:bkivr -->
- [ ] Add general tinyidp backup, backup-verify, and restore Glazed commands. <!-- t:bkicli -->
- [ ] Test clients, users, credentials, grants, interactions, signing-key overlap, and audit restoration. <!-- t:bkitst -->

## Phase 2 — Hostauth component backup

- [ ] Implement application auth/session/capability/audit snapshot API. <!-- t:bkauth -->
- [ ] Implement verification and staged restore with schema checks. <!-- t:bkavr -->
- [ ] Prove issuer-subject uniqueness, disabled users, and revoked sessions survive restore. <!-- t:bkatst -->

## Phase 3 — Durable Objects component backup

- [ ] Implement actor/alarm quiescence and object storage snapshot API. <!-- t:bkobj -->
- [ ] Record namespace, object database, alarm, and schema inventory. <!-- t:bkomf -->
- [ ] Implement verification, staged restore, and alarm-index reconciliation. <!-- t:bkovr -->
- [ ] Prove actor identity, private data, alarms, and schema state survive restore. <!-- t:bkotst -->

## Phase 4 — XAPP product coordinator

- [ ] Implement offline product lock and running-server refusal. <!-- t:bkxlock -->
- [ ] Implement XAPP backup by invoking all component snapshotters. <!-- t:bkx -->
- [ ] Package and verify state manifest, token key, and object-binding key. <!-- t:bksec -->
- [ ] Implement path-safe staged XAPP restore into a new root. <!-- t:bkxr -->
- [ ] Add backup, backup-verify, restore, and state doctor commands. <!-- t:bkxcli -->

## Phase 5 — Recovery proof and operations

- [ ] Complete real-app login/write/backup/restore/login/read round-trip. <!-- t:bke2e -->
- [ ] Test corruption, missing keys, unsupported versions, disk-full, and interrupted restore. <!-- t:bkfail -->
- [ ] Measure archive size, duration, memory, and large-object-root behavior. <!-- t:bkperf -->
- [ ] Write operator backup, verification, restore, rollback, and recovery-drill playbooks. <!-- t:bkrun -->
- [ ] Review encryption-at-rest integration and residual risks. <!-- t:bkenc -->
