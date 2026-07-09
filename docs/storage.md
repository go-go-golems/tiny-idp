# tiny-idp Strict Engine Storage Profile

The strict engine stores identity and protocol state through the public
`pkg/idpstore.Store` contract. `pkg/sqlitestore` is the supported production
implementation for a single active tiny-idp process. This document defines the
transaction, migration, durability, backup, restore, permission, and filesystem
contract an operator must understand before deploying it.

## Supported deployment envelope

Use SQLite only when one tiny-idp process is actively writing one database on a
local filesystem. A standby may exist, but it must not open the live file until
the active process has stopped and ownership has moved.

Supported:

- one process and one configured SQLite connection;
- a local POSIX filesystem with working advisory locks, atomic same-filesystem
  rename, file `fsync`, and directory `fsync`;
- WAL (the default), DELETE, or TRUNCATE journal mode;
- `FULL`, `NORMAL`, or `EXTRA` synchronous policy (`FULL` is the default);
- online backup while the source provider remains active;
- restore only while the destination provider is stopped.

Not supported without separate qualification:

- two active tiny-idp processes sharing one database;
- NFS, SMB/CIFS, distributed filesystems, object-store mounts, or a volume whose
  rename/fsync/locking semantics are unknown;
- copying only the main `.db` file while WAL mode is active;
- restoring over an open destination or a destination with `-wal`/`-shm`
  sidecars.

## Opening SQLite

Opening is context-aware and uses an explicit configuration:

```go
cfg := sqlitestore.DefaultConfig("/var/lib/tinyidp/idp.db")
store, err := sqlitestore.Open(ctx, cfg)
if err != nil {
    return err
}
defer store.Close()
```

`DefaultConfig` selects:

```text
busy_timeout         5 seconds
journal_mode         WAL
synchronous          FULL
foreign_keys         ON
max open connections 1 (required, not merely a default)
```

The single connection is part of the correctness envelope. It keeps PRAGMA
state deterministic and serializes in-process writers. `Open` creates the main
file as mode `0600`, applies ordered migrations, and forces the main, WAL, and
SHM files to mode `0600` when present.

## Public transaction model

`pkg/idpstore` separates transaction shape from the SQLite driver:

```go
type AtomicStore interface {
    View(ctx context.Context, fn func(ReadStore) error) error
    Update(ctx context.Context, fn func(TxStore) error) error

    CreateUserWithCredential(...)
    ReplacePasswordAndSecurityState(...)
    RecordFailedLogin(...)
    RecordSuccessfulLogin(...)
    RotateSigningKey(...)
}
```

No `*sql.Tx` crosses the public boundary. The SQLite implementation routes every
operation on a callback-scoped `TxStore` through the same `sql.Tx`. The memory
implementation mutates a snapshot and swaps it into the live store only after a
successful callback.

Transaction rules:

- callback error rolls back all writes and is returned unchanged;
- commit errors are returned;
- context cancellation aborts pool waits and SQL calls;
- nested `View`/`Update` calls fail with `idpstore.ErrNestedTransaction`;
- callers must not retain callback-scoped stores after return;
- named invariant operations are preferred to open-coded `Update` callbacks.

## Atomic security transitions

These transitions must commit completely or leave the previous valid state:

| Transition | State changed together | Entry point |
|---|---|---|
| Provision user | user plus password credential | `CreateUserWithCredential` |
| Replace password | credential plus reset security state | `ReplacePasswordAndSecurityState` |
| Failed login | window, count, timestamps, derived lock | `RecordFailedLogin` |
| Successful login | reset security state plus optional session | `RecordSuccessfulLogin` |
| Authorization-code use | validate unconsumed/unexpired plus mark consumed | `ConsumeAuthorizationCode` |
| Refresh rotation | link old token plus insert replacement | `RotateRefreshToken` |
| Refresh reuse | mark reuse plus revoke every token in the grant family | `RotateRefreshToken` |
| Signing rotation | create inactive key, switch active key, retire prior key | `RotateSigningKey` |
| Fosite refresh rotation | revoke refresh request plus its access tokens | internal Fosite transaction |

SQLite also enforces at most one row with `signing_keys.active = 1` through a
partial unique index. The store refuses to retire the active key directly; key
rotation first establishes a replacement.

## Schema and migration ledger

Migrations are embedded under `pkg/sqlitestore/migrations` and named with
contiguous numeric versions (`001_...`, `002_...`). Startup rejects gaps,
duplicate versions, non-numeric prefixes, and modified historical migrations.

For each migration, `Migrate` performs:

```text
read embedded SQL
compute SHA-256
if version exists:
    require stored checksum == embedded checksum
else:
    BEGIN
    execute migration SQL
    insert version, filename, checksum, applied_at
    COMMIT
```

A failed migration is rolled back and is not written to `schema_migrations`.
Never edit a shipped migration; add the next numeric file.

## Online backup

`Store.Backup` uses `sqlite3_backup`, so committed pages still resident in WAL
are included. The publication sequence is:

```text
live Store
  -> reserve the sole source connection
  -> capture schema/checksum/count/key manifest
  -> SQLite online backup into 0600 temp DB
  -> open temp DB read-only and immutable
  -> PRAGMA integrity_check
  -> compare schema, checksums, all table counts, active key IDs
  -> fsync temp file and destination directory
  -> atomic same-filesystem rename to final name
  -> chmod 0600 and fsync destination directory
```

The destination directory is created or corrected to mode `0700`. The command
refuses `/` and the shared system temporary directory because changing those
permissions would be unsafe. On cancellation, disk-full, verification failure,
or any pre-rename error, the temporary file is removed and an existing final
backup remains untouched.

CLI examples:

```bash
tinyidp admin --db /var/lib/tinyidp/idp.db \
  backup create --out /var/backups/tinyidp/idp-2026-07-09.db

tinyidp admin backup verify \
  --path /var/backups/tinyidp/idp-2026-07-09.db
```

## Restore

Restore is deliberately offline:

1. Stop the provider and ensure no process has the destination open.
2. Confirm destination `-wal` and `-shm` files do not exist.
3. Run the restore command.
4. Open the restored database and run provider readiness plus an OIDC smoke
   flow before deleting the rollback copy.

```bash
tinyidp admin --db /var/lib/tinyidp/idp.db \
  backup restore --path /var/backups/tinyidp/idp-2026-07-09.db
```

`Restore` verifies the artifact read-only, copies it into a mode-`0600`
same-directory temporary file with cancellation checks, verifies the staged
copy again, preserves an existing destination as a timestamped
`.pre-restore-*` rollback file, atomically renames the staged database, and
fsyncs the directory. It rejects corrupt artifacts, unsupported schema
versions, migration checksum mismatches, and live SQLite sidecars.

## Stored data and secrets

The schema contains clients, users, password credentials, account-security
state, grants, authorization codes, access and refresh tokens, consents,
sessions, signing keys, and Fosite authorize-code/PKCE/OIDC/token/JTI records.

The store keeps hashes for bearer-style handles where the domain owns the
token/session value. Fosite tables contain request/session metadata required to
complete grants and refresh rotation. Backups contain password hashes, client
secret hashes, and private signing-key PEM and must therefore be treated as
production secrets even though raw bearer values are not stored.

## Verification tests

The Phase 2 suite covers:

- transaction rollback and nested-transaction rejection;
- parallel one-time authorization-code consumption;
- simultaneous failed-login writers with no lost increments;
- refresh reuse and family revocation;
- failed and checksum-mismatched migrations;
- committed WAL content in online backup;
- read-only manifest verification and restore;
- concurrent writers during backup;
- corrupt backup refusal;
- context cancellation and busy-connection deadlines;
- injected `ENOSPC` with preservation of the last published backup;
- `0600` database/WAL/SHM/backup files and a `0700` backup directory.
