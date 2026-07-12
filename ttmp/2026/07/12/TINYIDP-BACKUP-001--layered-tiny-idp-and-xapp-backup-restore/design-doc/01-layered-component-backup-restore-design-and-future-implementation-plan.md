---
Title: Layered Component Backup Restore Design and Future Implementation Plan
Ticket: TINYIDP-BACKUP-001
Status: active
Topics:
    - identity
    - architecture
    - testing
    - go
    - security
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design a standard tiny-idp component archive, hostauth and Durable Object snapshots, and an XAPP coordinator that produces and restores one verified product backup."
LastUpdated: 2026-07-12T17:53:41.839674856-04:00
WhatFor: "Defines ownership, commands, archive contracts, consistency boundaries, and phased implementation tasks for reliable recovery."
WhenToUse: "Use when implementing any backup/restore API or CLI and when reviewing compatibility between a tiny-idp component archive and an XAPP product archive."
---

# Layered Component Backup Restore Design and Future Implementation Plan

## Executive Summary

Backup is layered. General-purpose tiny-idp must implement and version the
backup format for state that tiny-idp owns. go-go-goja hostauth and
go-go-objects must do the same for their stores. The combined XAPP must not copy
these implementations. It coordinates quiescence, invokes each component,
packages shared product secrets and configuration, verifies the complete set,
and restores into a staging root before an atomic final rename.

The initial release uses offline backup. A live backup flag must not exist until
all component snapshotters can participate in a coordinated snapshot protocol.

## Problem Statement

The initialized XAPP contains several separately consistent state domains:

- tiny-idp clients, users, credentials, grants, sessions, protocol artifacts,
  signing keys, schema history, and audit events;
- application users, OIDC identities, browser sessions, memberships,
  capabilities, and application audit events;
- Durable Object databases, alarms, object schema state, and actor bindings;
- the product manifest, token key, and object-binding key.

Copying files independently can produce a backup whose individual databases are
valid but represent different logical times. Reimplementing tiny-idp backup in
the XAPP would also create two archive formats and two sets of restore bugs.

The design must ensure internal consistency, global quiescence, path traversal
resistance, owner-only permissions, format versioning, checksum validation,
restore staging, schema compatibility checks, and an executable recovery drill.

## Proposed Solution

### Ownership

```text
pkg/idpbackup
  owns tiny-idp component manifest, snapshot, verify, and restore

go-go-goja hostauth backup package
  owns application auth/session/capability/audit snapshot and restore

go-go-objects backup package
  owns object databases, alarm state, schema inventory, and restore

tinyidp-xapp coordinator
  owns exclusive product lock, quiescence, component invocation,
  product secrets, cross-component validation, and final archive
```

### Intended commands

General-purpose tiny-idp:

```text
tinyidp backup --db PATH --audit PATH --output ARCHIVE
tinyidp backup-verify --input ARCHIVE
tinyidp restore --input ARCHIVE --output-dir EMPTY_DIRECTORY
```

Combined product:

```text
tinyidp-xapp backup --state-root ROOT --output ARCHIVE
tinyidp-xapp backup-verify --input ARCHIVE
tinyidp-xapp restore --input ARCHIVE --state-root NEW_EMPTY_ROOT
tinyidp-xapp doctor --state-root RESTORED_ROOT
```

### Product archive

```text
xapp-backup/
  backup.json
  components/
    tiny-idp/
      backup.json
      identity.sqlite
      audit.jsonl
    application-auth/
      backup.json
      auth.sqlite
    durable-objects/
      backup.json
      objects/...
  product/
    state.json
    secrets/token.key
    secrets/object-binding.key
```

The tiny-idp directory is a standard tiny-idp component archive. It can be
verified independently by general tiny-idp tooling.

### Global algorithm

```text
backup(root, output):
    acquire exclusive product lock
    refuse if a serving process owns the state root
    stop admission of new writes
    drain active requests and actors

    idp = idpbackup.snapshot(identity, idpAudit)
    app = hostauthbackup.snapshot(appStores)
    objects = objectsbackup.snapshot(objectManager)

    copy product manifest and security roots
    hash every file
    write product manifest last
    fsync archive and parent directory
    release lock
```

```text
restore(archive, destination):
    require destination absent or empty
    reject absolute paths, traversal, links, devices, and duplicate names
    verify product and component format versions
    verify every length and checksum

    create owner-only sibling staging root
    restore each component through its owning package
    restore product secrets with exact modes
    validate schemas and cross-component identities
    construct initialized application without listening
    require readiness
    atomically rename staging root to destination
```

### Backup manifest

The product manifest records format version, creation time, quiescence mode,
state-manifest version, application version, component format versions, and a
sorted list of file paths, lengths, modes, and SHA-256 checksums. It never
contains password material, cookies, raw authorization codes, PKCE values, or
command transcripts.

## Design Decisions

- Offline is the only initial consistency mode.
- Component archives remain independently verifiable.
- Restore targets a new root; in-place restore is forbidden.
- Restore never combines archive secrets with destination secrets.
- Archive extraction rejects all non-regular files and directories.
- Schema upgrade is a separate explicit operation, not a side effect of
  restore.
- The final recovery acceptance test uses the real application login flow and
  verifies an existing private object after restoration.

## Alternatives Considered

- Raw recursive copy: rejected because WAL databases and active actors can
  change during traversal.
- XAPP-specific tiny-idp copy logic: rejected because it duplicates ownership
  and breaks compatibility with other embedders.
- Immediate live backup: rejected until there is a coordinated snapshot
  protocol across all writers.
- In-place restore: rejected because partial extraction can destroy the only
  usable installation.
- Restore-time automatic migration: rejected because recovery and upgrade have
  different rollback and validation requirements.

## Implementation Plan

### Phase 0 — Contracts and fixtures

- Freeze component and product manifests.
- Inventory every owned file/table and WAL behavior.
- Create golden archives and malicious archive fixtures.
- Define exclusive state-root lock and quiescence interfaces.

### Phase 1 — General tiny-idp backup

- Implement SQLite snapshot and durable audit capture.
- Implement standard component manifest and verification.
- Add general `tinyidp backup`, `backup-verify`, and `restore` commands.
- Test grants, signing keys, interactions, credentials, and retired keys after
  restore.

### Phase 2 — Application-auth backup

- Add hostauth snapshot/verify/restore capability.
- Preserve issuer-plus-subject uniqueness and session/audit state.
- Prove disabled users and revoked sessions remain disabled/revoked.

### Phase 3 — Durable Objects backup

- Quiesce actors, alarms, and eviction.
- Snapshot object databases and alarm index.
- Inventory namespaces and object schema versions.
- Verify object identity and alarm recovery after restore.

### Phase 4 — XAPP coordinator

- Implement global lock and offline refusal.
- Invoke the three component snapshotters.
- Package product manifest and security roots.
- Implement safe staged restore and cross-component reconciliation.

### Phase 5 — Recovery proof

- Initialize and run the real TLS application.
- Login, write a private object, and stop the server.
- Back up and restore to a different root.
- Start the restored product, login, and read the same private object.
- Repeat with corruption, truncation, traversal, missing key, and unsupported
  format fixtures.

## Open Questions

- Whether audit logs are embedded in every archive or rotated at a defined
  checkpoint.
- Whether large object roots require a streaming tar format plus external
  encryption.
- Where deployment-specific encryption and key management live without making
  the application own an encryption master key.
- What coordinated protocol would be sufficient for a later live-backup mode.

## References

- `TINYIDP-XAPP-001` for the current product state-root and real application.
- `pkg/sqlitestore` for tiny-idp SQLite durability and existing backup support.
- go-go-goja hostauth SQL stores.
- go-go-objects Durable Objects manager, storage, alarms, and eviction.
