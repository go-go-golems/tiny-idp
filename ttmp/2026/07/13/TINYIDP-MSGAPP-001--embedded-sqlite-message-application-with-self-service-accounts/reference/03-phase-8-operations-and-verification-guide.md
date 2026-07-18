---
Title: Phase 8 Operations and Verification Guide
Ticket: TINYIDP-MSGAPP-001
Status: active
Topics:
    - go
    - identity
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-message-app/main.go
      Note: Glazed and Cobra executable composition, signal context, and zerolog initialization.
    - Path: repo://examples/tinyidp-message-app/commands.go
      Note: init, serve, doctor, construction, startup maintenance, health checks, and shutdown ownership.
    - Path: repo://examples/tinyidp-message-app/state.go
      Note: owner-only state layout, manifest, and atomic secret publication.
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: application health and readiness HTTP endpoints.
    - Path: repo://pkg/embeddedidp/provider.go
      Note: provider readiness, maintenance, and close semantics delegated by the host.
    - Path: repo://pkg/sqlitestore/backup.go
      Note: supported online-backup implementation for the identity database when a host schedules one.
ExternalSources:
    - https://www.sqlite.org/backup.html
    - https://www.sqlite.org/wal.html
Summary: Operator-facing guide for initializing, serving, diagnosing, backing up, restoring, upgrading, rolling back, and verifying the self-contained message application.
LastUpdated: 2026-07-14T21:10:00Z
WhatFor: Lets an operator safely run the example and understand its current single-node operational boundary.
WhenToUse: Read before local testing, staging deployment, backup work, upgrade work, or incident recovery.
---

# Phase 8 Operations and Verification Guide

## 1. Purpose and deployment boundary

`tinyidp-message-app` is a single Go executable built from
`examples/tinyidp-message-app`. It hosts a relying-party message application
and an embedded tiny-idp on one canonical browser origin. The browser reaches
the application at `/`, application protocol routes at `/auth/*` and `/api/*`,
static application assets at `/static/app/*`, the provider interaction
stylesheet at `/static/tinyidp/*`, and the provider at `/idp/*`.

The executable is intentionally a single-node SQLite application. A state root
belongs to one active process on a local filesystem. It is not a replicated,
multi-writer, or shared-network-filesystem deployment model. SQLite WAL gives
good local durability and reader concurrency; it does not turn the state root
into a distributed database.

The host constructs the following durable dependencies in this order:

```text
state manifest + owner-only secrets
              |
              v
identity SQLite ----> account service ----> embedded tiny-idp
              |                                  |
application SQLite <---- OIDC RP client <--------+
              |
              v
one public HTTP handler: /, /auth, /api, /idp, /static, /healthz, /readyz
```

This ordering is security-relevant. The provider is bootstrapped before
discovery, token exchange, or browser login are available. The OIDC RP client
uses `embeddedidp.NewInProcessIssuerTransport`, so discovery, token exchange,
and JWKS verification can only address the configured issuer; they cannot
fall back to a host network route.

## 2. Commands and the state root

The executable uses Glazed commands and the root logging section supplied by
Glazed. Every command accepts `--log-level`; no command reads secrets or core
configuration from environment variables.

```text
tinyidp-message-app init   --state-root DIR --public-base-url ORIGIN
tinyidp-message-app serve  --state-root DIR [operational flags]
tinyidp-message-app doctor --state-root DIR
```

`init` is idempotent only for the same canonical origin. It creates the state
directories at mode `0700`, creates owner-only 32-byte secrets at mode `0600`,
and atomically writes `state.json`. It intentionally does not create an
administrator account. The application demonstrates self-service registration,
so the first browser user is created through the public registration flow.

The state root has this contract:

```text
STATE/
  state.json                         canonical origin, issuer, client ID, version
  identity/tinyidp.sqlite             provider users, clients, keys, sessions, protocol state
  application/messages.sqlite         RP login attempts, app sessions, registration attempts, messages
  secrets/token.key                   provider token secret, 32 bytes, 0600
  secrets/app-session.key             reserved application secret material, 32 bytes, 0600
  audit/events.jsonl                  synchronous append-and-fsync audit stream, 0600
```

Do not hand-edit `state.json`, change the canonical origin in place, replace
only one database, or copy a live SQLite main file while ignoring its `-wal`
sidecar. Each operation can break the exact browser client registration,
invalidate a signing-key history, or create an inconsistent database image.

## 3. Local browser canary

The supported local mode is plain HTTP on `localhost` or a loopback IP. The
state-root validator rejects non-loopback HTTP origins. This is a deliberate
development exception; it permits a real browser canary without teaching
operators to deploy cleartext public authentication.

Initialize a fresh local state root:

```sh
go run ./examples/tinyidp-message-app init \
  --state-root /tmp/tinyidp-message-demo \
  --public-base-url http://127.0.0.1:8090
```

Start it in tmux, so that the process is observable and has a clear lifecycle:

```sh
tmux new-session -d -s tinyidp-message-demo \
  'cd /path/to/tiny-idp && go run ./examples/tinyidp-message-app serve \
  --state-root /tmp/tinyidp-message-demo --addr 127.0.0.1:8090 --log-level info'

tmux capture-pane -pt tinyidp-message-demo:0.0 -S -100
```

Then visit `http://127.0.0.1:8090/` and perform this canary:

1. Confirm the public message feed loads and the guest registration form is
   shown.
2. Create an account with a unique login, display name, and a policy-compliant
   password. Registration does not log the user in automatically.
3. Select **Sign in**. Confirm that the browser redirects to `/idp/authorize`
   and that the styled tiny-idp login form appears.
4. Enter the new credentials, approve the request if consent is shown, and
   confirm that the browser returns to `/` authenticated.
5. Post a message. Refresh the page and verify the message persists and its
   author is the authenticated account display name.
6. Select **Log out**, confirm the guest state returns, and verify a message
   POST without an application session is rejected.

Useful non-secret checks are:

```sh
curl -fsS http://127.0.0.1:8090/healthz
curl -fsS http://127.0.0.1:8090/readyz
go run ./examples/tinyidp-message-app doctor \
  --state-root /tmp/tinyidp-message-demo --log-level info
```

`/healthz` asks only whether the provider process is live. `/readyz` combines
the provider readiness report with an application SQLite ping. It is the
correct endpoint for admission to a load balancer or deployment controller.
Neither endpoint returns secret bytes, passwords, bearer tokens, or session
identifiers.

## 4. HTTPS staging and production-like mode

When `state.json` names an HTTPS origin, `serve` requires both `--tls-cert`
and `--tls-key`; the listener uses TLS 1.2 or newer. The host selects tiny-idp
production mode, secure cookies, a synchronous durable audit sink, a persistent
SQLite store, an explicit direct-peer address resolver, bounded password work,
and an in-process fixed-window login limiter. If TLS is missing for an HTTPS
origin, or if TLS arguments are supplied for an HTTP origin, startup fails.

Example shape (replace all paths and use a real certificate):

```sh
tinyidp-message-app init \
  --state-root /var/lib/tinyidp-message-app \
  --public-base-url https://messages.example.test

tinyidp-message-app serve \
  --state-root /var/lib/tinyidp-message-app \
  --addr :8443 \
  --tls-cert /etc/tinyidp-message-app/tls.crt \
  --tls-key /etc/tinyidp-message-app/tls.key \
  --log-level info
```

The current host trusts only its immediate TCP peer for rate-limit address
resolution. Deploy it directly, or add a separately reviewed trusted-proxy
configuration before placing it behind a reverse proxy. Do not assume that an
arbitrary `X-Forwarded-For` header changes the security decision.

## 5. Lifecycle, maintenance, and shutdown

`serve` performs these operations before it opens the listener:

```text
validate state manifest and secret file modes
open and migrate both databases
open durable audit stream
bootstrap exact browser client and active signing key
construct provider, renderer, in-process issuer transport, and RP client
run identity retention maintenance
clean expired or terminal application protocol state
evaluate readiness
bind listener
```

An `errgroup` owns three concurrent activities: the HTTP server, the periodic
maintenance/cleanup loop, and context-driven graceful shutdown. SIGINT and
SIGTERM cancel the root context. Shutdown calls `http.Server.Shutdown` with a
bounded timeout and closes the provider, audit stream, application database,
and identity database. A failed maintenance pass is logged and turns provider
readiness unhealthy rather than silently claiming the application is ready.

Stop a local process with Ctrl-C in its tmux pane, or deliberately terminate
the tmux session. Before reusing a local port, follow repository practice:

```sh
lsof-who -p 8090 -k
```

## 6. Backup procedure

### 6.1 What must be preserved

A restorable product snapshot contains all of the following:

- the identity SQLite database and its migration state;
- the application SQLite database and its migration state;
- `secrets/token.key`, because it protects tokens and changing it invalidates
  outstanding token material;
- `secrets/app-session.key`, reserved application secret material;
- `state.json`, which binds the databases to the canonical origin and client;
- `audit/events.jsonl`, retained as operational evidence.

The two databases have different transactional domains. A message references
an immutable subject string and does not require a cross-database SQL join, so
a documented short skew is tolerable for the current append-only message
model. It is still not acceptable to take a raw copy of a live WAL-mode SQLite
main file without a SQLite-aware backup operation.

### 6.2 Current supported coordinated backup

The first-release operational procedure is a short controlled offline backup.
This is preferable to claiming that two independent database snapshots are
atomically consistent when they are not. Schedule a maintenance window, stop
the single writer, check that it exited, and then copy the complete state root
with owner-only modes preserved into a protected backup target. Verify the
restored copy with `doctor` before relying on it.

```text
1. Announce maintenance and stop the only serve process.
2. Confirm no process still has the state root's SQLite files open.
3. Copy state.json, identity/, application/, secrets/, and audit/ together.
4. Preserve modes: directories 0700; secret, database, WAL, SHM, and audit files 0600.
5. Encrypt and store the backup outside the running host.
6. Start the original state root, then run doctor against the backup in an isolated restore location.
```

For future online backup automation, use `pkg/sqlitestore.Store.Backup` for
the identity database rather than filesystem copy. Application-database online
backup and a signed cross-database backup manifest are deliberately not claimed
as implemented in this example. They are the next operational hardening task,
not a hidden guarantee.

## 7. Restore, upgrade, and rollback

### Restore

Restore only to a private, owner-only directory. Never start the original and
restored state roots simultaneously against the same public HTTPS origin;
doing so can issue conflicting browser sessions and create ambiguous recovery
evidence.

```text
1. Stop the application and retain the failed state root untouched.
2. Restore the complete backup to a new state directory with original modes.
3. Run doctor against the new directory.
4. Start it on an isolated loopback/staging origin for a browser canary.
5. After the canary, make the recovered instance the sole owner of the public origin.
```

If the restore has the same public origin, the token secret and signing keys
must move with it. Replacing only the databases can cause token or cookie
validation failures and is not a supported recovery action.

### Upgrade

Both SQLite stores have migration ledgers. Startup opens each database and
applies only new, checksummed migrations. Before an upgrade:

1. Run the complete verification gate on the candidate build.
2. Take the coordinated offline backup described above.
3. Run `doctor` on the current state root and record its success.
4. Stop the current executable and start the candidate with the same root.
5. Wait for `/readyz`, inspect the audit stream and server log, then run the
   browser canary.

Treat a migration checksum or filename mismatch as a stop condition. It means
the recorded migration history and executable disagree; do not edit the
ledger to force startup.

### Rollback

Rollback is safe only when the previous executable can understand the schema
currently on disk. If the new release added migrations, restoring the backup
made immediately before the upgrade is the reliable rollback procedure. Keep
that backup until the new release has completed its post-deployment observation
window. Do not downgrade binaries against a newer database merely because the
process starts; semantic incompatibilities may appear later in protocol state.

## 8. Verification gate and known follow-up work

Before merging or releasing an application change, run at least:

```sh
pnpm -C examples/tinyidp-message-app/ui run build
go test ./examples/tinyidp-message-app
go test -race ./examples/tinyidp-message-app
go test ./pkg/embeddedidp ./pkg/idpaccounts ./pkg/sqlitestore
go vet ./examples/tinyidp-message-app
```

Then perform `init`, `doctor`, `/healthz`, `/readyz`, and the manual browser
canary. The repository-wide static analyzers and fuzz-smoke suites belong in
the release gate as their ticketed harnesses mature; their current result must
be recorded in the implementation diary rather than inferred from a unit-test
green result.

The remaining Phase 7 component/accessibility task and Phase 4 explicit
callback mismatch/expiry test task remain separately tracked. They are not
silently closed by the runtime host work. This guide is also deliberately
explicit that coordinated *online* backup of the two database domains remains
future implementation work.
