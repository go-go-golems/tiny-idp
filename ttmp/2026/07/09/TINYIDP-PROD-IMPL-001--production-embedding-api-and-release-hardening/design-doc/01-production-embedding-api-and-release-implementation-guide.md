---
Title: Production embedding API and release implementation guide
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/admin/backup.go
      Note: Admin wrapper over verified online backup and offline restore
    - Path: repo://internal/authn/password.go
      Note: Password verification and lockout transition behavior
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict OAuth and OIDC composition and route ownership
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public context-aware construction and validation boundary
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Public durable record and persistence contracts introduced in Phase 1
    - Path: repo://pkg/sqlitestore/backup.go
      Note: Implemented Phase 2 online backup verification atomic publication and offline restore (commit 7cd13b4)
    - Path: repo://pkg/sqlitestore/backup_test.go
      Note: Phase 2 WAL concurrency corruption interruption permission and restore evidence
    - Path: repo://pkg/sqlitestore/store.go
      Note: Public durable SQLite implementation and migrations introduced in Phase 1
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md
      Note: Source findings and release gate
ExternalSources: []
Summary: Intern-oriented architecture, public API, persistence, authentication, operations, and phased implementation guide for making tiny-idp releasable.
LastUpdated: 2026-07-09T17:37:01.014365676-04:00
WhatFor: Guiding implementation and review of the production embedding API and every release-hardening phase.
WhenToUse: Read before implementing a phase, reviewing a hardening change, onboarding to tiny-idp, or assessing release evidence.
---



# Production embedding API and release implementation guide

## Executive Summary

`tiny-idp` is a Go OpenID Connect identity provider with two engines. The mock
engine is intentionally convenient for development. The strict engine uses
Fosite, durable domain state, password authentication, browser sessions,
consent, signing keys, and OAuth/OIDC protocol storage. Production deployment is
supposed to embed the strict engine's `http.Handler` in a host Go process.

The strict engine had a sound foundation, but the reviewed baseline could not
ship. Its exported construction API mentioned Go `internal/` types, so an
external module could not construct the only production-shaped provider. The
same review also reproduced reachable dependency vulnerabilities, WAL-unsafe
backup, non-atomic security transitions, bypassable abuse controls, lost
concurrent lockout updates, and permissive SQLite file creation.

As of 2026-07-09, Phases 0, 1, and 2 are implemented. The selected dependency
graph is vulnerability-clean; the public embedding API is consumable from an
outside module and passes a TLS Authorization Code + S256 PKCE flow; and SQLite
now has explicit transactions, named invariant operations, checksummed
migrations, online backup, read-only verification, atomic publication, and
verified offline restore. Sections labeled “current state” describe the review
baseline unless an “implemented state” subsection says otherwise. Phases 3–5
remain release blockers.

This guide turns those findings into an ordered implementation program:

1. establish a reproducible vulnerability-clean release graph;
2. replace the unusable public API with importable contracts and lifecycle;
3. implement atomic persistence, correct online backup, and restore proof;
4. make password and abuse controls mandatory, bounded, and observable;
5. harden signing keys, audit, readiness, and maintenance;
6. prove the exact release candidate in CI and a production-like environment.

This is a direct pre-release redesign. Do not add compatibility aliases or
adapters for the current API: it cannot be used by its intended external
consumer, so preserving it would add complexity without preserving working
behavior.

The durable phase and task state lives in `tasks.md`. This document explains the
system and the reasoning behind those tasks; the diary records what actually
happened. A checked phase gate means its evidence exists, not merely that code
was written.

## How an Intern Should Read This Guide

Read the document in this order:

1. Learn the vocabulary and package map.
2. Follow one Authorization Code + PKCE request through the diagrams and
   pseudocode.
3. Understand why the current public API is not actually public.
4. Study the proposed API and lifecycle before editing packages.
5. Read transaction and backup sections before changing SQLite.
6. Read the phase you are implementing and its acceptance gate.
7. Read the latest diary entry before running or changing anything.

Important terms:

- **IdP:** the identity provider authenticating a user and issuing tokens.
- **RP:** relying party, normally an OIDC client application.
- **Issuer:** canonical HTTPS base URL identifying this IdP.
- **Authorization Code + PKCE:** browser authorization flow where a one-time
  code is bound to a verifier, preventing intercepted-code use.
- **Fosite:** the OAuth2/OIDC protocol library used by the strict engine.
- **Domain store:** tiny-idp records such as users, credentials, clients,
  consent, sessions, keys, and token families.
- **Protocol store:** Fosite request/code/token state needed to enforce OAuth
  lifecycle rules.
- **Embedding API:** exported Go packages used by another application to
  construct and host tiny-idp in its own `http.Server`.
- **Phase gate:** executable evidence required before the next phase can be
  considered complete.

## Problem Statement

### Intended deployment model

The production-shaped model is not “run the demo CLI and expose its port.” A
host program should import tiny-idp, open a durable store, provide security
policies and secret sources, obtain an `http.Handler`, and mount it in a hardened
HTTP server. The host owns network and process concerns; tiny-idp owns identity
and protocol correctness.

```text
                    host application
        ┌───────────────────────────────────────┐
        │ TLS termination and trusted proxies  │
        │ http.Server limits and timeouts       │
        │ process lifecycle and shutdown        │
        │                                       │
browser ├──► embeddedidp.Provider.Handler() ───┐│
client  │                                      ││
        └──────────────────────────────────────┼┘
                                               ▼
          ┌──────────── strict tiny-idp ─────────────┐
          │ Fosite protocol validation              │
          │ login, consent, sessions, CSRF           │
          │ claims, signing, discovery, JWKS         │
          │ audit, rate limiting, readiness          │
          └──────────────────┬───────────────────────┘
                             ▼
                    SQLite durable state
```

### The exported boundary is unusable

`pkg/embeddedidp.Options` is exported, but fields at
`pkg/embeddedidp/options.go:30-40` have types from `internal/storage`,
`internal/audit`, and `internal/fositeadapter`. Go only permits packages inside
the parent module tree to import an `internal` package. An application in a
different module therefore cannot import the durable SQLite store or name the
policy contracts required to fill the options.

The external compile probe produced:

```text
use of internal package github.com/manuel/tinyidp/internal/store/sqlite not allowed
```

The public wrapper at `pkg/embeddedidp/provider.go:10-26` only exposes
`Handler()`. It has no context-aware startup, structured readiness, or shutdown
contract. This leaves production lifecycle behavior implicit.

### Production risk is broader than API shape

Repairing imports alone would make an unsafe system deployable. The previous
review demonstrated six blocker families:

- the embedding API cannot be consumed externally;
- the selected runtime/dependency graph has reachable known vulnerabilities;
- raw copying the SQLite main file can omit committed WAL data while producing
  a backup that opens successfully;
- multi-write security state is not consistently transactional;
- production audit and rate limiting can silently fall back to no-op/allow-all,
  and concurrent lockout accounting loses updates;
- database creation does not enforce owner-only permissions.

The release program must therefore repair construction, state transitions,
operations, and release evidence together.

## Scope and Non-Goals

In scope:

- the strict Fosite-backed engine;
- exported embedding, store, policy, and lifecycle APIs;
- SQLite as the first supported durable implementation;
- passwords, sessions, consent, grants, tokens, and signing keys;
- audit, rate limiting, client-address trust, readiness, maintenance, backup,
  restore, migration, CI, conformance, and artifact evidence.

Not automatically in scope:

- turning the mock engine into a production engine;
- active/active SQLite across hosts;
- a PostgreSQL implementation;
- MFA, account recovery, SCIM, federation, or a full operator UI;
- compatibility with the current unusable pre-release embedding surface.

These can become later tickets. They must not expand the release boundary while
the core invariants remain incomplete.

## Current-State Architecture

### Engine split

```text
cmd/tinyidp
  └─ internal/cmds/serve.go
       ├─ mock engine: internal/server
       │    development scenarios and extra debug/device behavior
       └─ strict engine: pkg/embeddedidp -> internal/fositeadapter
            Fosite OAuth/OIDC + durable domain/Fosite state
```

The mock engine is useful for tests and local integrations, but production work
must be evaluated against the strict engine. The strict route registration at
`internal/fositeadapter/provider.go:265-271` exposes:

| Route | Role |
|---|---|
| `/.well-known/openid-configuration` | issuer metadata and endpoint discovery |
| `/jwks` | public verification keys |
| `/authorize` | browser authorization, login, consent, session reuse |
| `/token` | code exchange and refresh |
| `/userinfo` | claims for an access token |
| `/healthz` | process liveness-style response |
| `/readyz` | current minimal readiness response |

### Package responsibility map

| Package/file | Current responsibility | Intended direction |
|---|---|---|
| `pkg/embeddedidp` | nominal exported constructor and handler | stable public construction/lifecycle only |
| `internal/fositeadapter` | Fosite composition, HTTP endpoints, sessions, CSRF, claims | remain internal behind public contracts |
| `internal/domain/types.go` | clients, users, credentials, grants, tokens, consent, sessions, keys | move stable consumer-visible records to `pkg/idpstore` |
| `internal/storage/interfaces.go` | entity-oriented store capabilities | replace with public read/transaction/invariant contracts |
| `internal/store/sqlite` | durable domain and protocol storage | become supported `pkg/sqlitestore` implementation |
| `internal/authn/password.go` | Argon2id verification, lockout, audit | depend on atomic public store operations |
| `internal/passwordhash` | Argon2id encoding/parsing | remain internal unless external administration requires it |
| `internal/keys` | generation and rotation workflow | use atomic store operation and public lifecycle policy |
| `internal/admin` | user/client/key/backup operations | call public stores and invariant services |
| `internal/oidcmeta` | issuer parsing and discovery | remain internal protocol support |

### State model

`internal/domain/types.go` separates profile and credential data deliberately:

- `Client` stores registered redirect URIs, scopes, secret hash, PKCE and TTL
  policy.
- `User` stores OIDC subject/profile and account disable/lock state.
- `PasswordCredential` stores only an encoded password verifier and lifecycle
  metadata; plaintext passwords never belong on `User`.
- `AccountSecurityState` stores failure windows, counters, lockout, and last
  successful login.
- `Grant`, `AuthorizationCode`, `AccessToken`, and `RefreshToken` represent
  authorization and token lifecycle.
- `Consent` is server-side remembered approval.
- `Session` stores only a hash of the browser's random session handle.
- `SigningKey` stores private key material plus activation dates.

The separation is good, but it creates cross-record invariants. Creating a user
and credential, updating a password and security state, rotating a refresh
token, or switching active signing keys must succeed or fail as one unit.

### Current store contract

`internal/storage/interfaces.go:19-100` composes many entity-level interfaces
into one large `Store`. Methods such as `GetAccountSecurityState` followed by
`PutAccountSecurityState` do not express atomic read-modify-write semantics.
The password service performs exactly that sequence at
`internal/authn/password.go:181-199`, allowing concurrent failures to overwrite
each other.

The SQLite implementation has a process mutex (`internal/store/sqlite/store.go:
25-28`), but locking individual method calls cannot protect a service workflow
spanning multiple calls. Transaction ownership must move to an operation that
sees the entire invariant.

### SQLite representation and migrations

The store opens `database/sql`, runs every embedded migration, and serializes
many records as JSON blobs (`internal/store/sqlite/store.go:30-86`). This is
simple and useful for a small IdP, but production behavior needs:

- explicit schema versions and checksums;
- transactionally applied migrations;
- constrained/indexed columns for invariants queried by SQL;
- connection/journal/busy/synchronous configuration;
- a documented local-filesystem, single-active-process support envelope;
- permission enforcement independent of ambient umask.

### Authorization Code + PKCE flow

```text
Browser/RP       strict handler       authn/consent        Fosite       SQLite
    │ GET /authorize   │                    │                  │             │
    ├─────────────────►│ parse request ──────────────────────►│             │
    │                  │ validate client, redirect, PKCE      │             │
    │ login form       │                    │                  │             │
    │◄─────────────────┤                    │                  │             │
    │ credentials      │ authenticate ─────►│ read user/cred ──────────────►│
    ├─────────────────►│                    │ Argon2id + lockout             │
    │                  │ consent decision ────────────────────────────────►│
    │                  │ issue code ────────────────────────►│ persist ────►│
    │ redirect + code  │                    │                  │             │
    │◄─────────────────┤                    │                  │             │
    │ POST /token + verifier                │                  │             │
    ├───────────────────────────────────────►│ Fosite consumes code          │
    │                  │                    │ validates PKCE, issues tokens  │
    │ tokens           │                    │ persists token/request state ─►│
    │◄───────────────────────────────────────┤                  │             │
```

Simplified authorization pseudocode:

```text
parse and validate Fosite authorization request
if reusable browser session satisfies prompt/max_age:
    load client and determine consent requirement
    if consent not required:
        finish authorization and issue one-time code
render login/consent interaction

on POST:
    validate CSRF
    apply client/account/address rate limits
    authenticate password using constant-cost unknown-account path
    atomically update security state
    persist server-side session using hashed random handle
    persist consent when granted
    finish Fosite authorization response
```

### Trust boundaries

```text
untrusted network input
    ├─ issuer/Host/proxy headers
    ├─ OAuth parameters and redirect URIs
    ├─ credentials and cookies
    └─ JWT/JWK/request-object material
             │
             ▼
host boundary: TLS, proxy allowlist, HTTP limits, shutdown
             │
             ▼
provider boundary: validation, CSRF, rate limits, authn, Fosite
             │
             ▼
secret/state boundary: token secret, signing key, SQLite, backups, audit
```

Never trust `X-Forwarded-For` merely because it exists. The host and provider
must share an explicit trusted-proxy policy; otherwise an attacker chooses the
rate-limit key.

## Proposed Solution

## Proposed Package Architecture

```text
pkg/idp
  Mode, AuditSink, RateLimiter, ClientAddressResolver,
  PasswordAuthenticator, ConsentPolicy, readiness and policy types

pkg/idpstore
  stable records, sentinel errors, Store/ReadStore/TxStore,
  high-level atomic security operations

pkg/sqlitestore
  Open(ctx, Config), migrations, transactions, online backup,
  read-only verification, restore support, maintenance, diagnostics

pkg/embeddedidp
  Options, New(ctx, Options), Provider.Handler,
  Provider.Readiness, Provider.Close

internal/fositeadapter
  maps public contracts to Fosite and HTTP behavior
```

Dependencies point inward toward contracts:

```text
host app ─► embeddedidp ─► idp + idpstore
   │              │
   └─► sqlitestore┘
                  ▼
       internal/fositeadapter ─► Fosite
```

`pkg/idp` must not import Fosite. Public consumers should not need to understand
Fosite request/session types to implement audit, rate limiting, consent, or
authentication.

### Public construction API

```go
package embeddedidp

type Options struct {
    Issuer        string
    Mode          idp.Mode
    Store         idpstore.Store
    TokenSecret   idp.SecretSource
    Cookies       idp.CookiePolicy
    Audit         idp.AuditSink
    RateLimiter   idp.RateLimiter
    ClientAddress idp.ClientAddressResolver
    Authenticator idp.PasswordAuthenticator // optional only if safe default built
    Consent       idp.ConsentPolicy          // optional only if safe default built
    Clock         func() time.Time
}

func New(ctx context.Context, opts Options) (*Provider, error)

type Provider struct { /* unexported implementation */ }

func (p *Provider) Handler() http.Handler
func (p *Provider) Readiness(ctx context.Context) idp.ReadinessReport
func (p *Provider) Close(ctx context.Context) error
```

All methods that perform I/O accept `context.Context`. `New` performs bounded
startup checks and returns only after the provider is safe to serve. `Close` is
idempotent, stops internal background work, waits through `errgroup`, and does
not close externally owned dependencies unless ownership is explicit.

### Ownership table

| Resource | Default owner | Rule |
|---|---|---|
| `http.Server` and listener | host | host configures TLS, timeouts, limits, proxy trust, shutdown |
| `Provider` | host | host calls `Close(ctx)` after server shutdown begins |
| injected `Store` | host | provider does not close it unless constructor explicitly takes ownership |
| provider-created background workers | provider | stopped and joined by `Close` using context/errgroup |
| token/signing secret source | host integration | provider reads through interface and never logs material |
| SQLite online backup destination | `sqlitestore` operation | created owner-only, verified, fsynced, atomically published |

### Production validation

Production mode must reject:

- non-HTTPS issuer except explicitly supported loopback development mode;
- missing persistent store or unsupported schema;
- missing/weak token secret;
- absent, expired, malformed, weak, or non-unique active signing key;
- insecure cookie policy;
- nil audit sink, limiter, or address resolver;
- in-memory stores or allow-all/no-op production defaults;
- unsafe SQLite permissions or unsupported filesystem/configuration;
- inconsistent client redirect/scope/PKCE registration.

Validation pseudocode:

```text
parse canonical issuer
validate public options without side effects
inspect store capabilities, schema, configuration and permissions
load and cryptographically parse active signing key
validate time window and verification-key publication set
probe secret source without exposing bytes in diagnostics
probe audit/limiter/address-policy health
construct Fosite adapter
run structured readiness
if any required component is not ready: close partial resources and fail
return provider
```

### Readiness API

```go
type Check struct {
    Name      string
    Ready     bool
    Degraded  bool
    Reason    string // stable, non-secret reason code
    CheckedAt time.Time
}

type ReadinessReport struct {
    Ready  bool
    Checks []Check
}
```

Readiness is operational state, not just “the process started.” Required checks
include store connectivity, schema support, active signing key usability, secret
source availability, audit health, limiter health, and overdue maintenance.
`/healthz` remains liveness and should avoid dependent I/O.

## Public Store and Transaction Design

Avoid one enormous public interface that every test double must implement.
Expose capability-oriented read/transaction contracts plus high-level methods
for security invariants.

```go
type Store interface {
    View(ctx context.Context, fn func(ReadStore) error) error
    Update(ctx context.Context, fn func(TxStore) error) error

    RecordFailedLogin(
        ctx context.Context,
        userID string,
        now time.Time,
        policy LockoutPolicy,
    ) (AccountSecurityState, error)

    CreateUserWithCredential(
        ctx context.Context,
        user User,
        credential PasswordCredential,
    ) error

    ReplacePasswordAndSecurityState(
        ctx context.Context,
        credential PasswordCredential,
        state AccountSecurityState,
    ) error

    RotateSigningKey(
        ctx context.Context,
        next SigningKey,
        now time.Time,
    ) (RotationResult, error)
}
```

High-level methods make the invariant explicit and testable across memory and
SQLite implementations. `Update` supports administrative compositions, but raw
`*sql.Tx` must not leak from `pkg/sqlitestore`.

Transaction rules:

- a callback cannot retain a transaction object after return;
- the first callback error rolls back and is preserved with context;
- commit failure is returned even when callback work succeeded;
- nested transactions are rejected or explicitly implemented with documented
  savepoint semantics—never silently opened as unrelated transactions;
- context cancellation aborts waits and SQL work;
- all entity methods invoked through `TxStore` use that transaction;
- no audit event claims success until commit succeeds.

Atomic failed-login pseudocode:

```text
BEGIN IMMEDIATE
load security row for user
if row absent: initialize count/window
if previous window expired: reset count and first-failure timestamp
increment count
if count reaches threshold: calculate locked-until
write row with monotonic version/update condition
COMMIT
return committed state
```

The lockout decision and stored counter must be derived inside the same
transaction. A Go mutex cannot replace this guarantee because multiple database
connections or future processes do not share it.

## Correct SQLite Backup and Restore

The current `internal/admin/backup.go:20-51` copies only the main database file.
In WAL mode, recent committed pages may exist only in `-wal`. The destination
can open successfully while silently missing committed identities or keys.

Required online backup algorithm:

```text
validate source and destination are different
create owner-only destination directory
create owner-only temporary database on same destination filesystem
start SQLite online backup from live source connection
copy pages in bounded batches with context cancellation and busy retry
finish backup and close destination
fsync destination file
open destination read-only without running migrations
run PRAGMA integrity_check
read schema version and migration checksums
compare source manifest with destination:
    critical row counts
    active key identifiers
    configured client/user counts
fsync destination directory
atomically rename temporary file to final path
fsync destination directory again
emit success audit only after publication
```

On any error, remove the temporary file and leave an existing final backup
untouched. Verification must never call the normal `Open` path if that path can
migrate or mutate the file.

Restore is a separate operation:

1. verify artifact integrity and supported schema read-only;
2. require the provider to be stopped or hold an exclusive restore lock;
3. preserve the current database for rollback;
4. atomically install the verified artifact;
5. reopen, run readiness, and complete a strict smoke flow;
6. retain the rollback copy until operator acceptance.

## Implemented Phase 1 and Phase 2 Reference

This section records what the code does after Phases 1 and 2. It is the bridge
between the earlier proposed design and the concrete API an intern will review.
The public boundary is intentionally small: host applications import
`pkg/embeddedidp`, `pkg/idp`, `pkg/idpstore`, and `pkg/sqlitestore`; Fosite and
`database/sql` remain implementation details.

### Phase 1 host-facing API

Construction is context-aware and accepts only public or standard-library
types (`pkg/embeddedidp/options.go`). In production mode, validation rejects a
missing or unsafe component before any handler is returned:

- HTTPS issuer and production-valid clients;
- secure cookies and at least 32 bytes of token secret;
- explicit audit sink, rate limiter, and client-address resolver;
- a store reporting persistence and a positive schema version;
- exactly one currently usable RS256 key;
- parseable RSA private key with a modulus of at least 2048 bits.

The lifecycle is:

```text
host startup context
  -> sqlitestore.Open(ctx, Config)
  -> provisioned clients/users/key already present
  -> embeddedidp.New(ctx, Options)
  -> provider.Readiness(ctx)
  -> mount provider.Handler() in host HTTP server
  -> host stops accepting traffic
  -> provider.Close(shutdownCtx) (idempotent)
  -> store.Close()
```

The host owns the store and HTTP server. `Provider.Close` does not close the
injected store. The external fixture in the production-review ticket imports
only public packages, starts a TLS test server, submits the CSRF login form,
exchanges an authorization code with S256 PKCE, checks access and ID tokens,
checks readiness, and closes.

### Phase 2 public transaction API

The transaction interfaces are defined at
`pkg/idpstore/interfaces.go:105-160`. `ReadStore` exposes reads, while `TxStore`
exposes domain operations within one implementation transaction. Neither
mentions `*sql.DB`, `*sql.Tx`, nor SQLite driver types.

```go
err := store.Update(ctx, func(tx idpstore.TxStore) error {
    if err := tx.PutUser(ctx, login, user); err != nil {
        return err
    }
    return tx.PutPasswordCredential(ctx, credential)
})
```

The named `CreateUserWithCredential` method packages this example so callers do
not need to reproduce it. The SQLite callback implementation begins one
`sql.Tx`, constructs a scoped store whose query runner is that transaction,
preserves the callback error, and commits only after success
(`pkg/sqlitestore/store.go:787-837`). The memory implementation clones every
map, runs the callback against the clone, and replaces live maps only after
success. Both reject a nested transaction with
`idpstore.ErrNestedTransaction`.

An important review rule is that “a method uses a mutex” does not imply “the
method is transactional.” The mutex can serialize goroutines in one process;
only the database transaction gives rollback and database-level isolation.

### Complete multi-record transition inventory

| Security transition | Before Phase 2 risk | Implemented boundary | Primary call site |
|---|---|---|---|
| User plus password credential | credential failure left an orphan user | `CreateUserWithCredential` | `internal/admin/users.go:67-107` |
| Password plus security reset | reset failure left stale lockout state | `ReplacePasswordAndSecurityState` | `internal/admin/users.go:116-139` |
| Failed-login increment | read/modify/write lost concurrent increments | `RecordFailedLogin` | `internal/authn/password.go:170-176` |
| Successful-login reset plus optional session | partial success state | `RecordSuccessfulLogin` | password service and future session callers |
| Authorization-code consumption | two consumers could pass the read | root call opens `Update`; scoped call validates and marks | `pkg/sqlitestore/store.go` |
| Refresh-token replacement | old link could commit without new token | root call opens `Update`; scoped call writes both | `pkg/sqlitestore/store.go` |
| Refresh reuse and family revocation | some family rows could remain live | reuse outcome commits detection and all revocations before returning the sentinel | `pkg/sqlitestore/store.go:557-596` |
| Signing-key rotation | create/activate/retire could stop midway | `RotateSigningKey` | `internal/admin/keys.go:18-47` |
| Fosite refresh/access revocation | one protocol token class could remain | one direct `sql.Tx` | `internal/fositeadapter/sqlstore.go:278-290` |

The refresh-reuse case has subtle control flow. Reuse is an expected security
outcome, but returning `ErrRefreshReuseDetected` directly from an `Update`
callback would roll back the evidence and revocations. The root method therefore
captures that outcome, lets the callback return `nil` so the security changes
commit, and returns the sentinel only after commit:

```text
Update callback:
    result, outcome = scoped.RotateRefreshToken(...)
    if outcome == reuse-detected:
        return nil             # commit detection + family revocation
    return outcome             # real storage failures roll back
after Update:
    return result, outcome
```

### Key uniqueness and retirement

Migration `003_signing_key_invariants.sql` adds a partial unique index for rows
whose `active` column is one. Activation runs inside a transaction and first
sets every row's scalar active column to zero, then updates the serialized key
records and activates the target (`pkg/sqlitestore/store.go:702-752`). This
ordering avoids a transient unique-index violation.

Direct retirement of the active key returns `idpstore.ErrLastSigningKey`.
Rotation generates the replacement, creates it inactive, makes it the unique
active key, and retires the previous key within one transaction. The retired
key remains a verification key until later retention work.

### Migration ledger

`MigrationNames` requires contiguous numeric prefixes beginning at one
(`pkg/sqlitestore/store.go:185-207`). `Migrate` creates
`schema_migrations(version, name, checksum, applied_at)`, computes SHA-256 over
the exact embedded SQL, and either verifies the existing checksum or applies
and records the migration in one transaction
(`pkg/sqlitestore/store.go:212-268`).

This makes three failure classes explicit:

1. a missing or misordered migration fails before SQL execution;
2. a modified historical migration fails checksum verification;
3. failed new SQL rolls back and does not receive a ledger row.

The schema version reported to production preflight is the maximum committed
ledger version, not the number of files currently embedded in the binary.

### SQLite configuration and topology

The public configuration is `pkg/sqlitestore.Config`:

```go
type Config struct {
    Path               string
    BusyTimeout        time.Duration
    JournalMode        string
    Synchronous        string
    MaxOpenConnections int
}
```

`DefaultConfig` chooses a five-second busy timeout, WAL, `FULL`, foreign keys,
and one connection. The implementation rejects any connection count other than
one. That is a supported-topology assertion, not a tuning suggestion: it keeps
connection-local PRAGMAs deterministic and matches the single-active-node
design decision.

The supported filesystem must provide SQLite locking, atomic same-filesystem
rename, file fsync, and directory fsync. Local POSIX filesystems are in scope.
NFS, SMB/CIFS, distributed filesystems, object-store mounts, and multiple active
processes are not supported without separate qualification. The main DB, WAL,
and SHM are forced to `0600`.

### Online backup data flow

The implementation lives in `pkg/sqlitestore/backup.go`. It reserves the sole
source connection, which gives a stable in-process ordering point, captures a
logical manifest, and uses `github.com/mattn/go-sqlite3.SQLiteConn.Backup` in
bounded page batches:

```text
source DB (possibly committed pages in WAL)
     |
     | sqlite3_backup, 128 pages/step, context checked
     v
0600 temp DB in destination directory
     |
     | read-only immutable open
     | integrity_check + schema + checksums + row counts + active key IDs
     v
fsync(temp) -> fsync(directory) -> rename(temp, final) -> fsync(directory)
```

The manifest covers every domain and Fosite table, every migration checksum,
the highest committed schema version, and the ordered active-key IDs. A backup
whose integrity is valid but whose logical manifest differs from the source
snapshot is not published.

Temporary and final backup files are `0600`; their dedicated directory is
`0700`. The function refuses `/` and the shared system temp directory because
changing either directory's mode would be unsafe. Before atomic rename, every
error removes only the temporary path and preserves an existing published
backup. This includes cancellation and injected `ENOSPC`.

### Read-only verification and offline restore

`VerifyBackup` uses a `file:` URI with `mode=ro&immutable=1`. It does not call
`Open`, run migrations, create a journal, or rewrite metadata. Verification
requires:

1. `PRAGMA integrity_check` equals `ok`;
2. schema version equals the current supported version;
3. stored migration checksums equal the embedded SQL;
4. optional expected manifest equals the artifact manifest.

Restore is intentionally not an online operation. It first verifies the source,
refuses a destination with `-wal` or `-shm`, copies with context checks into a
same-directory `0600` temporary file, verifies that staged file again, preserves
an existing main DB as `.pre-restore-<timestamp>`, atomically installs the
staged file, and fsyncs the directory. The operator keeps the rollback copy
until reopening, readiness, and a strict OIDC smoke flow succeed.

### Phase 2 failure evidence

The test suite is designed around the old-or-new-state rule:

- callback injection and a credential conflict prove no orphan user or
  credential (`pkg/sqlitestore/transaction_test.go:18-55`);
- 24 simultaneous failed-login writers produce exactly 24 increments and a
  lock (`pkg/sqlitestore/transaction_test.go:57-85`);
- invalid active-key state proves a failed migration is not recorded, and a
  tampered checksum prevents reopen
  (`pkg/sqlitestore/transaction_test.go:87-134`);
- a non-empty committed WAL is present before backup, and its client record is
  present after restore (`pkg/sqlitestore/backup_test.go:16-82`);
- restore preserves rollback data and refuses corruption
  (`pkg/sqlitestore/backup_test.go:84-124`);
- cancellation preserves the last published artifact and leaves no temp file
  (`pkg/sqlitestore/backup_test.go:126-155`);
- a held connection obeys context deadline rather than hanging
  (`pkg/sqlitestore/backup_test.go:157-171`);
- concurrent writers produce a self-consistent verifiable backup
  (`pkg/sqlitestore/backup_test.go:173-204`);
- an injected `syscall.ENOSPC` preserves the last good backup
  (`pkg/sqlitestore/backup_fault_test.go`).

The repository-specific Go analyzer `tinyidpatomicity` is implemented with
`go/ast`, `go/types`, and `golang.org/x/tools/go/analysis`. It scans the admin,
Fosite adapter, and public SQLite packages for functions containing multiple
persistence mutations without `Begin`, `BeginTx`, `Update`, or a named atomic
boundary. Transaction-scoped helpers require an explicit source directive, so
future open-coded multi-write paths become review diagnostics.

## Authentication and Abuse-Control Design

### Password work

Argon2id intentionally consumes memory and CPU. The current default is a
security strength, but an attacker can turn unconstrained parallel attempts into
memory exhaustion. Introduce a context-aware semaphore around real and dummy
verification.

```text
resolve trusted client address
check cheap account/client/address limiter
try to acquire password-work permit with request context
if saturated or deadline exceeded:
    record bounded rejection metric and generic login failure
perform constant-cost lookup and Argon2id verification
release permit
atomically record failure or reset success state
emit stable audit outcome
```

Do not skip dummy verification for unknown accounts, disabled accounts, or
malformed logins in a way that creates a user-enumeration timing signal.

### Password acceptance versus hashing

Hash parameters answer “how is a password verified?” Password policy answers
“what new password is accepted?” Keep them separate. Creation/reset/change must
share one policy validator so a one-character password cannot be provisioned
while login claims an eight-character minimum.

Must-change-password is a control-flow state, not metadata to ignore. If it is
supported, a successful password verification must enter a restricted flow that
can only change the password and cannot complete the OIDC grant until the change
commits. Otherwise remove the flag until the flow exists.

### Rate limiting and client addresses

Production mode requires layered keys:

- normalized account/login key;
- OAuth client identifier;
- trusted client network address;
- optionally global password-work capacity.

Responses remain generic. The limiter must not reveal whether an account exists.
Client addresses come from `RemoteAddr` unless the immediate peer matches an
explicit trusted-proxy network, in which case a documented forwarded-header
algorithm is applied.

## Signing Keys, Audit, and Maintenance

### Signing keys

Startup and rotation validate:

- supported algorithm;
- cryptographic key size;
- private/public parseability and match;
- `NotBefore <= now < NotAfter` for active signing;
- exactly one active signing key;
- enough retained verification keys to validate unexpired issued tokens.

Rotation must create the next key, make it active, retire the old signer, and
preserve verification publication atomically. Never permit retiring the last
usable key.

### Audit

`audit.NoopSink` is acceptable for tests and development but not production.
The public audit contract needs an explicit policy:

- delivery deadline;
- buffering and maximum queue;
- whether backpressure blocks sensitive operations;
- behavior when the sink is unavailable;
- dropped-event counters and readiness degradation;
- redaction rules for credentials, secrets, codes, tokens, and cookies.

Audit uses stable event and reason codes. It must not include raw library errors
as externally consumed security taxonomy.

### Maintenance

Expired sessions, Fosite requests, codes, tokens, consent, and retired keys
accumulate unless maintenance owns retention. A provider-managed worker may run
under `errgroup`, but single-active-node ownership must be explicit. Maintenance
reports last-success time, duration, rows removed, errors, and overdue state to
readiness/metrics.

## Host Application Contract

The embedded provider returns a handler; it cannot enforce every network
property. The supported production example must demonstrate:

```go
srv := &http.Server{
    Addr:              listenAddress,
    Handler:           requestLimits(provider.Handler()),
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       60 * time.Second,
    MaxHeaderBytes:    1 << 20,
}

g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return serveTLSOrTrustedProxy(srv) })
g.Go(func() error {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return srv.Shutdown(shutdownCtx)
})
if err := g.Wait(); err != nil { /* classify expected server close */ }
if err := provider.Close(closeCtx); err != nil { /* fail shutdown evidence */ }
```

The host contract documents TLS location, trusted proxies, allowed hosts,
request/body limits, timeouts, graceful shutdown order, readiness exposure, log
redaction, and secret injection.

## Design Decisions

### Decision: Replace the pre-release API directly

- **Context:** Exported options depend on internal types and cannot be consumed
  from the intended external module.
- **Options considered:** Compatibility aliases; adapters around internal
  stores; direct public contract redesign.
- **Decision:** Replace the current API and update all callers together.
- **Rationale:** There is no working external behavior to preserve, and the user
  explicitly does not want unnecessary compatibility layers.
- **Consequences:** A clean break now; examples and tests must migrate in the
  same commit series.
- **Status:** accepted.

### Decision: Keep Fosite internal

- **Context:** Fosite is central to protocol behavior but its concrete types are
  not product contracts.
- **Options considered:** Export Fosite options/interfaces; wrap each type;
  define product-level contracts.
- **Decision:** Public APIs use tiny-idp and standard-library types only.
- **Rationale:** Consumers implement identity policy rather than depending on a
  specific protocol library's request/session representation.
- **Consequences:** `internal/fositeadapter` owns mapping and requires focused
  adapter tests.
- **Status:** accepted.

### Decision: Store owns atomic security operations

- **Context:** Service-level sequences currently lose updates and can commit
  partial state.
- **Options considered:** Go mutexes; raw SQL transactions in services; generic
  transaction callback only; high-level invariant methods plus callbacks.
- **Decision:** Public store contracts include transaction callbacks and named
  atomic invariant operations.
- **Rationale:** Named operations are portable, testable, and cannot be bypassed
  accidentally by implementations using multiple connections.
- **Consequences:** Memory and SQLite stores share an invariant suite.
- **Status:** accepted.

### Decision: SQLite is single-active-node storage

- **Context:** SQLite is an excellent small embedded database but does not
  provide transparent distributed coordination.
- **Options considered:** Claim multi-instance support; add process locking;
  require a network database; document one active process.
- **Decision:** Support one active process on a local durable filesystem first.
- **Rationale:** This is honest, useful, and testable without unsafe pseudo-HA.
- **Consequences:** Availability comes from restart/volume failover; active/active
  requires another store design.
- **Status:** accepted.

### Decision: Use SQLite online backup

- **Context:** Raw main-file copy loses committed WAL pages.
- **Options considered:** Server shutdown and copy; checkpoint and copy;
  `VACUUM INTO`; SQLite online backup.
- **Decision:** Use online backup for live operation and verify read-only before
  atomic publication.
- **Rationale:** It provides documented snapshot semantics under live writes.
- **Consequences:** Backup becomes driver-specific `pkg/sqlitestore` behavior.
- **Status:** accepted.

### Decision: Host owns HTTP; provider publishes a strict contract

- **Context:** Embedded code cannot choose every application's listener, TLS,
  proxy, or orchestration model.
- **Options considered:** Provider starts a server; handler with prose only;
  handler plus executable host example/readiness contract.
- **Decision:** Keep `http.Server` ownership with the host and test a supported
  production example.
- **Rationale:** Retains embedding flexibility without hiding essential controls.
- **Consequences:** Host conformance is part of release evidence.
- **Status:** accepted.

### Decision: Fail closed in production mode

- **Context:** Nil controls currently become no-op audit and allow-all rate
  limiting.
- **Options considered:** Safe implicit defaults; warnings; required options.
- **Decision:** Require controls unless the library can construct a genuinely
  production-safe default from complete configuration.
- **Rationale:** Missing controls should prevent readiness, not silently weaken
  the deployment.
- **Consequences:** Production setup is more explicit and diagnostics must be
  actionable without exposing secrets.
- **Status:** accepted.

## Alternatives Considered

- **Ship the CLI as the production server:** rejected because the current serve
  path is development-oriented and cannot own every deployment's TLS/proxy
  requirements without a separate hardening design.
- **Move only SQLite public and keep internal field types:** rejected because
  audit, policy, and store method signatures would remain unnameable externally.
- **Expose a single configuration callback hiding all types:** rejected because
  it makes durable provisioning, testing, and custom policy integration opaque.
- **Use a global mutex for lockout and key rotation:** rejected because it does
  not protect multiple DB connections/processes or crash atomicity.
- **Checkpoint WAL and copy files:** rejected as the primary live backup because
  checkpoint coordination, sidecar handling, and writer races are easier to get
  wrong than the supported backup API.
- **Switch to PostgreSQL immediately:** deferred. It could support future HA but
  would substantially widen the release surface before current invariants are
  correct.
- **Lower Argon2id cost to handle load:** rejected as the default response.
  Bound concurrency and capacity-plan first; tune parameters only through an
  explicit security/performance decision.

## Implementation Plan

The authoritative checkboxes are in `tasks.md`. The descriptions below explain
ordering, main files, and exit evidence.

### Phase 0 — Dependency and toolchain security baseline

Primary files: `go.mod`, `go.sum`, `Makefile`, future CI/release configuration.

Work:

1. Record exact Go, Fosite, go-jose, SQLite/CGO selections.
2. Pin a patched supported Go toolchain rather than depending on workstation
   auto-selection.
3. Upgrade go-jose to a non-vulnerable compatible version and review the graph.
4. Run build, test, vet, race, lint, Staticcheck, custom audit analyzers, fuzz
   seeds, conformance smoke, and `govulncheck`.
5. Add missing CI gates and artifact SBOM/provenance generation.

Gate: the exact reproducible graph has no reachable known vulnerabilities or a
written exception with owner and expiry. Record command output and hashes.

### Phase 1 — Consumable public embedding API

Primary files: `pkg/embeddedidp`, new `pkg/idp`, `pkg/idpstore`,
`pkg/sqlitestore`, README/examples, external-module fixture.

Work:

1. Inventory every internal type leaking through the exported package.
2. Define stable public records and policies without Fosite types.
3. Move the durable SQLite implementation behind public contracts.
4. add context-aware construction, readiness, close, and ownership docs;
5. make production validation fail closed;
6. update all in-repo callers directly;
7. convert the negative external compile probe into a positive strict-flow test.

Gate: a separate module imports only public packages, provisions durable state,
starts production mode, completes Authorization Code + PKCE, observes readiness,
and shuts down cleanly.

### Phase 2 — Transactional persistence, backup, and restore

Primary files: store contracts/implementations, migrations, Fosite SQL store,
admin user/key/backup operations.

Work:

1. Map every multi-statement invariant.
2. Implement transaction callbacks and named atomic operations.
3. make user, password, lockout, refresh, and key transitions atomic;
4. introduce schema version/checksum history and transactional migrations;
5. implement online backup, read-only verification, fsync, atomic publish, and
   restore;
6. enforce permissions and supported SQLite pragmas/topology;
7. add concurrency and failure-injection suites.

Gate: forced failures at every operation boundary leave either the old or new
valid state; live WAL backup/restore preserves committed manifests.

### Phase 3 — Mandatory authentication and abuse controls

Primary files: `internal/authn`, strict provider/limiter, admin user workflows,
public policy contracts, metrics.

Work:

1. Enforce one password acceptance policy everywhere.
2. implement or remove must-change behavior;
3. require production rate limiting and trusted address resolution;
4. atomically count failures and reset success state;
5. bound Argon2id concurrency and expose capacity signals;
6. define session/token revocation after password change;
7. run simultaneous failure and realistic abuse/load tests.

Gate: the former negative invariant probe now asserts protection, and sustained
abuse cannot bypass counters or exceed the documented memory/concurrency budget.

### Phase 4 — Keys, audit, readiness, and maintenance

Primary files: key lifecycle, audit packages, embedded provider, maintenance
worker/command, readiness handlers.

Work:

1. Validate and atomically rotate signing keys with verification overlap.
2. require audit and implement explicit delivery/health semantics;
3. make readiness structured and dependency-aware;
4. separate liveness from readiness;
5. implement retention and expose maintenance health;
6. resolve currently ineffective configuration contracts.

Gate: every known unsafe configuration fails startup/readiness, and lifecycle
failure tests demonstrate correct health transitions and audit signals.

### Phase 5 — Release engineering and deployment proof

Primary files: CI, release configuration, production example, operator docs,
release evidence under this ticket.

Work:

1. Add always-on and release-only CI matrices.
2. build/test the production host example;
3. run sustained production-parameter load and inspect profiles/metrics;
4. run hosted OIDC conformance on the exact release artifact;
5. perform backup, restore, migration, rotation, rollback, and incident drills;
6. produce signed artifacts, checksums, SBOM, provenance, license inventory;
7. obtain independent review and release-owner sign-off.

Gate: a production-like deployment of the signed candidate passes protocol,
security, recovery, and operations evidence with recorded artifact hashes.

### Commit and diary rhythm

For each bounded task or tightly coupled task group:

```text
read latest diary and relevant phase section
implement the smallest coherent change
format and run targeted tests
run proportional repository gates
review diff and commit only related files
record commit hash, commands, errors, findings in diary
relate files and update changelog/tasks
commit documentation checkpoint
```

Do not check a phase gate in the same instant the implementation is first
written. Run the gate from a clean worktree and record its evidence.

## Testing and Verification Strategy

### Always-on checks

```text
go build ./...
go test ./... -count=1
go vet ./...
make lint
govulncheck ./...
custom Go analysis multichecker
external consumer integration
SQLite permission/backup/restore tests
strict local conformance smoke
```

### Release/scheduled checks

- race detector on all packages;
- longer native fuzzing of issuer, redirect, Argon hash, JWT/JWK/request object,
  form, and persisted-request parsers;
- multi-connection concurrency and SQL fault injection;
- sustained password load at production Argon2id parameters;
- leak, goroutine, file-descriptor, heap, CPU, DB-pool, and audit-queue analysis;
- hosted OpenID Foundation conformance;
- migration from every supported released schema;
- restore and incident-response drills;
- artifact signature, SBOM, provenance, license, and vulnerability verification.

### Required failure tests

```text
cancel startup during store/key/audit initialization
cancel shutdown while background maintenance is active
fail commit after successful callback
fail each statement in user/password/key/refresh transitions
run five simultaneous failed logins and require count=5 + locked
backup while WAL contains uncheckpointed committed pages
interrupt backup before publish and preserve old destination
open with umask 000 and require owner-only database/sidecars
start with expired/malformed/multiple active signing keys and reject
make audit/limiter/secret source unhealthy and require not-ready
```

### Release evidence packet

The final packet should identify:

- source commit and clean-tree status;
- Go toolchain and module graph;
- build flags, CGO and SQLite versions;
- artifact hashes and signatures;
- SBOM, provenance, license inventory, and vulnerability result;
- unit/race/fuzz/static/conformance results;
- backup/restore/migration/rotation/load drill results;
- deployment configuration and supported topology;
- residual risks with owner, expiry, monitoring, and rollback criteria;
- independent reviewer and release owner approval.

## Security Review Checklist

- Redirect URIs are exact registered matches and never suffix/prefix matches.
- Production enforces Authorization Code + S256 PKCE.
- State, nonce, code, token, cookie, and CSRF values use cryptographic randomness.
- Raw codes, access tokens, refresh tokens, passwords, and session handles are
  not stored or logged.
- Cookies are Secure, HttpOnly, correctly scoped, and use effective SameSite.
- Unknown-account login performs equivalent expensive work.
- Password work has a documented concurrency/memory budget.
- Lockout and refresh reuse operations are atomic.
- Active signing keys are valid and verification overlap covers token lifetime.
- Audit cannot silently disappear in production.
- Trusted proxy and address resolution cannot be spoofed by arbitrary clients.
- Database, WAL/SHM, backup, and secret material are owner-only.
- Backup verification is read-only and restore is rehearsed.
- Readiness fails closed when critical dependencies or invariants fail.

## Open Questions

These require explicit product/operations decisions before their affected phase
gate can close:

1. Is single-active-node SQLite on local durable storage the accepted v1
   production topology?
2. Which proxy/load balancer implementations and forwarded-header conventions
   must the address resolver support?
3. What is the required audit durability model: synchronous, bounded buffered,
   or external transactional outbox?
4. Where do token secrets and signing keys come from in the first deployment:
   files, secret manager, KMS/HSM, or encrypted database material?
5. What maximum concurrent login rate and memory budget should Argon2id support?
6. Is must-change-password required for v1? If yes, what UX hosts the restricted
   flow?
7. Which token/session families must password change revoke?
8. What schema versions and downgrade paths will be supported after the first
   release?
9. What retention periods apply to sessions, token/request records, consent,
   retired verification keys, and audit buffers?
10. Which OpenID Foundation conformance profile is the release gate?
11. Who is the independent security reviewer and final release owner?

## References

Repository evidence:

- `pkg/embeddedidp/options.go:30-77` — current exported options and validation.
- `pkg/embeddedidp/provider.go:10-26` — current handler-only wrapper.
- `internal/fositeadapter/provider.go:34-182` — Fosite composition, defaults,
  and production protocol configuration.
- `internal/fositeadapter/provider.go:265-271` — strict route surface.
- `internal/domain/types.go:14-185` — current durable domain records.
- `internal/storage/interfaces.go:19-107` — entity-oriented store contract.
- `internal/store/sqlite/store.go:25-82` — SQLite open and migration behavior.
- `internal/authn/password.go:95-225` — password, lockout, reset, and audit flow.
- `internal/admin/backup.go:20-87` — unsafe copy and mutating verification paths.
- `internal/store/sqlite/migrations/001_schema.sql` and
  `002_password_credentials.sql` — current schema.
- `internal/fositeadapter/sqlstore.go` — durable Fosite protocol state.
- `internal/keys/rotation.go` — current non-atomic key lifecycle.
- `internal/cmds/serve.go` — development server and current handler mounting.

Ticket evidence:

- `TINYIDP-PROD-REVIEW-001/design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md`
  — complete observed production review.
- `TINYIDP-PROD-REVIEW-001/reference/01-investigation-diary.md` — commands,
  scanners, probes, failures, and measurements.
- `TINYIDP-PROD-REVIEW-001/scripts/` — Go analysis, external consumer, backup,
  fuzz, runtime, and security-invariant probes.
- `TINYIDP-PROD-IMPL-001/tasks.md` — authoritative phase ledger.
- `TINYIDP-PROD-IMPL-001/reference/01-implementation-diary.md` — chronological
  implementation record.

External API/standards references captured in the source review ticket:

- OAuth 2.0 Security Best Current Practice, RFC 9700.
- OpenID Connect Core 1.0.
- SQLite Online Backup API, WAL documentation, and corruption guidance.
- Go secure development guidance and `net/http` API documentation.
- OWASP Authentication and Password Storage Cheat Sheets.
