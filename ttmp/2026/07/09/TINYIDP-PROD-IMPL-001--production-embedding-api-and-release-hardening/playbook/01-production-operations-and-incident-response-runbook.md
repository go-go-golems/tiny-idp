---
Title: Production operations and incident response runbook
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/admin_keys.go
      Note: Planned rotation and emergency retired-key purge commands
    - Path: repo://internal/cmds/serve_production.go
      Note: Normal production process lifecycle
    - Path: repo://pkg/sqlitestore/backup.go
      Note: Verified online backup and offline restore implementation
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/02-release-drills.sh
      Note: Executable recovery and rotation rehearsal
ExternalSources: []
Summary: Operator procedures for normal startup, backup and restore, corruption, signing or token key compromise, dependency emergencies, administrative lockout, and rollback.
LastUpdated: 2026-07-09T20:31:17.514815597-04:00
WhatFor: Operating the single-node SQLite deployment safely and responding to security or availability incidents without destroying evidence.
WhenToUse: Use for deployment preparation, on-call response, recovery drills, key rotation, dependency response, and release rollback.
---


# Production operations and incident response runbook

## Purpose and safety model

This playbook turns tiny-idp's production contracts into operator actions. It
is intentionally conservative. Preserve evidence before repair, take the node
out of traffic before changing durable state, and never infer “healthy” from a
live process alone.

The two health signals mean different things:

- liveness answers whether the provider process is functioning;
- readiness additionally requires the exact SQLite schema, a usable active
  signing key, acceptable token secret, durable audit delivery, production rate
  limiter, and recent successful maintenance;
- orchestrators may restart on sustained liveness failure;
- load balancers must remove an unready node from traffic without creating a
  restart loop for a transient database or audit outage.

The supported topology is one active process using one SQLite database on a
local filesystem. The store intentionally uses one database connection. Do not
place the database on NFS/SMB, share it between active writers, or treat a
shared-volume replica as an HA design.

## Required deployment inventory

Record these values in the deployment-specific secret/asset inventory, not in
this repository:

- release artifact SHA-256 and Sigstore bundle;
- issuer URL and listen address;
- local SQLite path and filesystem/mount identity;
- audit JSONL path and external retention/shipping owner;
- TLS certificate/key locations and renewal owner;
- token-secret file location, generation date, and rotation owner;
- trusted reverse-proxy CIDRs and maximum forwarding hops;
- client IDs, redirect URIs, and maximum ID/refresh token lifetimes;
- verified-backup location, encryption owner, RPO, and restore-test date;
- on-call, security incident commander, and release-owner contacts.

The token-secret file must be a regular owner-only file containing at least 32
random bytes. The production command does not accept its contents on the
command line and does not read it from an environment variable.

## Normal deployment and preflight

Provision or migrate while no provider is using the database:

```bash
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db migrate --dry-run
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db migrate
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db doctor
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db export diagnostics
```

The migration command refuses a database newer than the running binary. This
is the downgrade guard. `doctor` must report the exact supported schema, one
usable active key, valid clients, and verification keys.

Start the production host through the service manager:

```bash
go run ./cmd/tinyidp serve-production \
  --addr :8443 \
  --issuer https://idp.example.test \
  --db /var/lib/tinyidp/idp.db \
  --audit-path /var/log/tinyidp/audit.jsonl \
  --token-secret-file /run/secrets/tinyidp-token \
  --tls-cert /run/tls/tls.crt \
  --tls-key /run/tls/tls.key
```

The host runs maintenance before listening. Require HTTP 200 and `ready: true`
from the issuer's `/readyz`; require HTTP 200 from `/healthz`. Confirm the audit
stream contains `maintenance.completed`. Never send credentials with `curl` as
part of a generic health check.

Exit criteria:

- deployed binary hash equals the approved evidence packet;
- liveness and all readiness components are true;
- SQLite, token-secret, and audit files are owner-only;
- discovery issuer/endpoints and JWKS match the public URL;
- one strict Authorization Code + S256 PKCE flow succeeds from an external RP;
- the prior release remains available for rollback.

## Routine backup and restore drill

Create online backups while the provider is serving:

```bash
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
  backup create --out /secure-backups/tinyidp/idp-YYYYMMDD.db
go run ./cmd/tinyidp admin backup verify \
  --path /secure-backups/tinyidp/idp-YYYYMMDD.db
```

The backup path is published only after SQLite online backup, file and
directory fsync, read-only `integrity_check`, exact migration checksum checks,
and source/backup table-count and active-key comparison. A backup that merely
exists is not a verified backup.

Restore requires downtime:

1. remove the node from traffic and confirm readiness is no longer advertised;
2. stop the provider and confirm the process is gone;
3. preserve the current DB, `-wal`, `-shm`, audit log, service logs, and host
   metadata as incident evidence;
4. verify the selected backup read-only;
5. ensure no live destination WAL/SHM sidecar exists—the restore command
   refuses them;
6. restore, retaining the reported `.pre-restore-*` rollback path;
7. run `doctor`, start the same artifact, check readiness, and complete an OIDC
   smoke flow;
8. create and verify a fresh post-recovery backup before closing the incident.

```bash
go run ./cmd/tinyidp admin backup verify --path /secure-backups/tinyidp/known-good.db
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
  backup restore --path /secure-backups/tinyidp/known-good.db
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db doctor
```

The scripted rehearsal is:

```bash
bash ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/scripts/02-release-drills.sh
```

## SQLite corruption or I/O incident

Symptoms include readiness `store` or `schema` failure, SQLite corruption/I/O
errors, checksum mismatch, repeated busy deadlines beyond expected load, or a
filesystem remount/read-only event.

Immediate actions:

1. remove the node from traffic; do not repeatedly restart it;
2. stop the process gracefully, then use `lsof-who -p PORT -k` only if it did
   not exit by the shutdown deadline;
3. snapshot/copy the DB, WAL, SHM, audit, and service logs without opening the
   evidence DB through a migrating binary;
4. record filesystem type, free space/inodes, mount options, kernel storage
   errors, binary hash, and last successful backup/maintenance times;
5. verify backups newest-to-oldest without modifying them;
6. restore the newest verified backup to a new path or the stopped production
   destination;
7. do not merge an old WAL into a restored main database.

Escalate immediately if audit delivery also failed, because the event timeline
may be incomplete. Recovery is complete only after doctor/readiness/OIDC checks,
a fresh backup, evidence preservation, and a written estimate of possible data
loss bounded by the last verified backup.

## Signing private-key compromise

Normal rotation retains the old public JWK until the longest ID token plus
clock skew expires. That is correct for planned rotation and unsafe for a
compromised private key.

For suspected compromise:

1. declare an incident and preserve the current JWKS, audit log, key metadata,
   binary hash, and detection evidence;
2. generate/activate a new key atomically;
3. verify the new `kid` is active and appears in JWKS;
4. use the explicit emergency purge to remove the compromised retired key
   before normal overlap expires;
5. notify relying parties that tokens signed by the old `kid` must be rejected
   and may require session/logout intervention;
6. review audit and RP logs for tokens issued after the earliest possible
   compromise time.

```bash
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
  keys rotate --kid incident-YYYYMMDD-HHMM
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db keys list
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
  keys purge-retired --kid COMPROMISED_KID
```

`purge-retired` refuses the active key and a staged key that was never retired.
It intentionally breaks verification of otherwise unexpired tokens signed by
the purged key; that is the compromise-response tradeoff.

## Token-secret compromise or planned rotation

The host token secret protects Fosite opaque token signatures, CSRF values, and
browser-session handles. Rotating it invalidates existing access/refresh tokens
and browser sessions. ID-token signatures use the RSA signing key and follow a
separate lifecycle.

1. remove the node from traffic and stop it;
2. generate a new owner-only 32-byte-or-longer secret at a new atomic secret
   version/path;
3. update the service configuration and restart once;
4. verify readiness and a fresh OIDC flow;
5. verify a pre-rotation access and refresh token is rejected;
6. revoke/expire the old secret in the secret manager and document the forced
   reauthentication window.

Do not run old- and new-secret instances concurrently behind a load balancer;
tokens minted by one will fail on the other.

## Client-secret compromise

Rotate the named confidential client secret:

```bash
go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
  client rotate-secret --id CLIENT_ID
```

The command returns the generated secret once. Transfer it through the approved
secret channel; never put it in a ticket or logs. Coordinate the RP cutover.
Because this pre-release API does not retain two simultaneous client-secret
hashes, rotation is an immediate cutover, not a grace-period operation.

## Administrative lockout

The production host has no network administration endpoint. Administrative
authority is filesystem/process access to the SQLite CLI, which reduces remote
attack surface but makes host access recovery important.

If every interactive user is locked or a password is lost:

1. verify the operator is on the correct stopped/active single-node host and
   database path;
2. use the approved host-privileged channel to set a new password via stdin;
3. do not pass the password as a CLI flag outside a disposable drill;
4. password replacement atomically clears security state and revokes that
   user's sessions, grants, codes, access tokens, and refresh families;
5. inspect the durable admin audit event and require the user to authenticate
   through a new browser session.

```bash
printf '%s\n' 'new deployment-approved passphrase' | \
  go run ./cmd/tinyidp admin --db /var/lib/tinyidp/idp.db \
    user set-password --login alice --password-from-stdin
```

## Audit delivery incident

The production sink is synchronous append+fsync. There is no hidden buffer and
no intentional drop path. Disk-full, permission, filesystem, closed-handle, or
fsync errors therefore propagate to the triggering operation or fail
readiness through a delivery counter.

1. remove the node from traffic when readiness reports `audit` failure;
2. preserve the audit file and storage diagnostics;
3. determine whether an administrative operation returned
   `ErrAuditDelivery`. That error means the mutation committed but audit did
   not—reconcile state before retrying;
4. restore capacity/permissions or move to an approved durable audit volume;
5. restart only after the sink health check succeeds;
6. reconstruct and explicitly annotate any evidence gap from database and
   service logs.

## Dependency vulnerability emergency

1. freeze release/deployment and identify the exact artifact/module graph;
2. run `govulncheck ./...` on that commit to determine reachability;
3. read the upstream advisory and fixed release from primary sources;
4. patch the smallest compatible dependency edge; do not broad-upgrade the
   graph during incident pressure;
5. rerun build, unit, race, lint, custom analysis, fuzz seeds, external OIDC,
   backup/restore, and release drills;
6. generate a new artifact hash, SBOM, provenance, signatures, and evidence
   packet—never relabel the old artifact;
7. if no fix exists, require an owner, compensating control, expiry date, and
   rollback trigger before any exception.

## Rollback and downgrade

Application rollback is permitted only when the prior binary supports the
database schema in use. An old binary must fail closed on a newer migration
ledger. Never delete migration rows or edit checksums to force a downgrade.

Safe choices are:

- roll back code when no forward-only migration was applied and the exact old
  artifact passes doctor/readiness;
- restore a verified pre-upgrade backup, accepting and documenting the RPO;
- roll forward with a corrected binary/migration.

Rollback exit criteria are the same as deployment: exact artifact hash,
supported schema, readiness, OIDC smoke, audit continuity, and a fresh verified
backup. Preserve the failed candidate and database snapshot for analysis.

## Incident closure checklist

- timeline and scope recorded in UTC;
- exact binary/config/schema/key IDs recorded without secret contents;
- durable audit and storage evidence preserved;
- liveness/readiness and external OIDC flow pass;
- backup verifies and a restore rehearsal has a named date/result;
- forced token/session invalidation impact communicated;
- root cause, corrective actions, owners, and deadlines assigned;
- release evidence and residual-risk ledger updated;
- security and release owners explicitly approve return to service.
