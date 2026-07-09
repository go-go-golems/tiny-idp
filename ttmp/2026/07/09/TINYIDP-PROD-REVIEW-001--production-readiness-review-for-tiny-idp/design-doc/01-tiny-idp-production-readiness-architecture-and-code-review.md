---
Title: Tiny IDP production readiness architecture and code review
Ticket: TINYIDP-PROD-REVIEW-001
Status: complete
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
    - Path: repo://README.md
      Note: Documented engine split and intended production embedding path
    - Path: repo://go.mod
      Note: Toolchain and vulnerable transitive dependency graph
    - Path: repo://internal/admin/backup.go
      Note: Live backup data-loss blocker
    - Path: repo://internal/admin/users.go
      Note: Password lifecycle and non-transactional admin operations
    - Path: repo://internal/authn/password.go
      Note: Password verification, policy, lockout, audit, and concurrency findings
    - Path: repo://internal/cmds/serve.go
      Note: Dev-only server boundary and HTTP hardening ownership
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict provider composition, endpoints, controls, login, token, and claims
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Fosite durable state and non-transactional refresh rotation
    - Path: repo://internal/keys/rotation.go
      Note: Signing key rotation invariant and crash window
    - Path: repo://internal/passwordhash/argon2id.go
      Note: Argon2id implementation and runtime capacity analysis
    - Path: repo://internal/storage/interfaces.go
      Note: Current entity-oriented store contract and proposed transaction boundary
    - Path: repo://internal/store/sqlite/migrations/001_schema.sql
      Note: Domain and Fosite schema reviewed for constraints/versioning/retention
    - Path: repo://internal/store/sqlite/store.go
      Note: SQLite persistence, key lifecycle, refresh operations, migrations, and permissions
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public production contract and startup validation reviewed in P0-1/P1-2
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/reference/01-investigation-diary.md
      Note: Chronological evidence supporting the final review
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/summary.md
      Note: Measured strict-flow runtime baseline
ExternalSources: []
Summary: Intern-oriented architecture, security, operations, code-quality, and implementation review for deciding whether tiny-idp can ship to production.
LastUpdated: 2026-07-09T17:31:17-04:00
WhatFor: Understanding tiny-idp end to end, making the production ship decision, and implementing the prioritized remediation plan.
WhenToUse: Use for onboarding, security review, release planning, remediation implementation, and final production acceptance.
---

















# Tiny IDP production readiness architecture and code review

## Executive Summary

### Release decision: no-go

`tiny-idp` should **not ship as a production identity provider in its current
state**. The strict Fosite engine has a solid protocol foundation and a broad
test suite, but the assembled product still has release-blocking problems at
the public API, dependency, persistence, authentication-control, secret-file,
and operations boundaries.

This is not a judgment that the system is a failed prototype. Important pieces
are already well designed:

- the production-shaped path uses Ory Fosite rather than handwritten OAuth;
- the strict profile permits Authorization Code plus PKCE S256 and omits
  implicit/hybrid grants;
- redirect URIs use exact matching;
- clients, users, consent, sessions, signing keys, and Fosite protocol state can
  live in SQLite;
- password credentials use salted Argon2id with a 64 MiB default work factor;
- raw browser-session handles are not persisted;
- CSRF, `HttpOnly`, Secure-cookie validation, security headers, opaque access
  tokens, refresh rotation, stable audit reason codes, and key rotation helpers
  are present;
- build, unit/integration tests, race detection, vet, pinned lint, rebuilt
  Staticcheck, and short parser fuzz campaigns pass;
- the strict happy-path runtime probe completed login, token exchange,
  UserInfo, refresh, and 40 bounded concurrent reads without request errors.

Those strengths make remediation tractable. They do not compensate for the
following blocker families.

| Priority | Blocker | Evidence | Required outcome |
|---|---|---|---|
| P0 | The documented production embedding API cannot be consumed by an external Go module because its public types require `internal/` packages. | `pkg/embeddedidp/options.go:30-40`; external compile probe | Move the production contracts and durable store to importable public packages; replace the pre-release API directly. |
| P0 | The selected release stack contains reachable vulnerabilities. Go 1.26.1 produced 12 reachable standard-library results; `go-jose/v3@v3.0.3` produced two reachable results even on Go 1.26.5. | `go.mod`; `various/govulncheck-go1.26.1.txt`; `various/govulncheck-go1.26.5.txt` | Build with a patched Go release and select go-jose v3.0.5 or later; rerun tests, conformance, and govulncheck. |
| P0 | The backup command can produce a readable backup that silently omits committed WAL data. | `internal/admin/backup.go:20-51`; live WAL probe | Use SQLite's online backup mechanism, verify read-only with integrity and content checks, and test under active writes/WAL. |
| P0 | Multi-write security transitions are not transactional. | `internal/store/sqlite/store.go:319-344,487-527`; `internal/fositeadapter/sqlstore.go:279-284`; `internal/admin/users.go:68-110`; auditlint | Make refresh rotation/reuse, key activation, user+credential creation, password reset, migrations, and related operations atomic. |
| P0 | Brute-force controls fail open: production accepts no limiter, limiter keys include ephemeral ports, and concurrent failed logins lose updates. | `internal/fositeadapter/provider.go:101-120,350,407`; `internal/authn/password.go:181-199`; invariant probe | Require a production limiter; normalize trusted client identity; atomically increment/lock accounts; test concurrent and distributed behavior. |
| P0 | SQLite creation does not enforce confidentiality for password hashes and private signing keys. | `internal/store/sqlite/store.go:30-40`; invariant probe observed `0644` under umask `000` | Create/preflight the database as owner-only, reject unsafe permissions, document storage encryption and secret-volume requirements. |

Several P1 issues must also be resolved before a real release: minimum-password
policy is declared but not applied; `MustChangeAtLogin` is not enforced; expired
active signing keys pass provider validation; audit delivery can be silently
disabled and every audit error is ignored; retention/cleanup and schema-version
operations are missing; public per-client TTLs are stored but unused; the
`SameSite` option does nothing; release CI/SBOM/provenance are absent; and the
supported reverse-proxy/single-node SQLite deployment envelope is not an
executable contract.

### What “ready” means

The recommended ship gate is not “all findings closed.” It is:

1. every P0/P1 invariant below has an implementation and automated acceptance
   test;
2. the external-consumer probe is a positive compile-and-flow test;
3. online backup and restore are tested with WAL and concurrent writes;
4. patched-toolchain govulncheck has zero reachable vulnerabilities or a
   reviewed, time-bounded exception;
5. the race suite, pinned lint, auditlint, storage concurrency tests, fuzz seed
   corpus, and local conformance suite run in CI;
6. a fresh hosted OpenID Foundation run passes the intended profile;
7. the deployment runbook proves TLS/proxy trust, file ownership, backup,
   restore, key/secret rotation, monitoring, incident response, and rollback.

## Review Scope and Method

### In scope

This review covers the `tiny-idp/` repository as a product:

- CLI and configuration;
- mock/strict engine selection;
- `pkg/embeddedidp` public API;
- strict Fosite request handling;
- password authentication and account lockout;
- domain model and storage contracts;
- memory and SQLite storage;
- Fosite's SQLite adapter;
- consent, browser sessions, CSRF, discovery, JWKS, and signing keys;
- admin CLI, migrations, backup, diagnostics, and readiness;
- tests, conformance material, build/lint/security tooling, and documentation;
- dependency/toolchain reachability and bounded runtime behavior.

The sibling `glazed/` and `go-go-goja/` repositories are workspace build inputs,
not primary code-review scope. The mock engine is reviewed for separation and
exposure risks, not held to production authentication semantics.

### Evidence method

The conclusions use four evidence layers:

1. **Line-level source inspection.** Major claims cite concrete files and
   symbols.
2. **Existing tests.** The suite contains 168 product tests before the ticket
   fuzz/analyzer additions.
3. **Purpose-built probes.** Typed analyzers, an external module compile test,
   WAL backup reproduction, runtime flow instrumentation, and security invariant
   checks live under this ticket's `scripts/` directory.
4. **Authoritative guidance.** The ticket captures OpenID Connect Core, RFC
   9700, SQLite backup/WAL guidance, Go security/HTTP documentation, and OWASP
   authentication/password-storage guidance under `sources/`.

Observed behavior is labeled as such. Recommendations that depend on deployment
policy are called out as decisions or open questions.

## What the System Is

`tiny-idp` is two related identity providers behind one CLI:

| Engine | Construction | Persistence | Purpose |
|---|---|---|---|
| Mock | `internal/server` through `tinyidp serve` | In-memory | Local RP development, scenario injection, device/DPoP experiments, intentionally malformed tokens/JWKS, debug state. |
| Strict/Fosite | `internal/fositeadapter` through dev `serve --engine fosite` or `pkg/embeddedidp` | Memory in CLI preview; SQLite through embedding | Production-shaped OAuth/OIDC validation, real users/passwords, consent, durable protocol state, keys, and sessions. |

The distinction matters. `tinyidp serve` always uses a local development
construction. Even its Fosite option creates a memory store, a fixed development
secret, scenario users, and `DevMode` (`internal/cmds/serve.go:213-268`). The
README correctly says production means embedding the strict handler with a
durable store (`README.md:204-287`). The public API defect means that intended
production path currently cannot be used by a normal external module.

## Architecture for a New Engineer

### High-level topology

```text
                         administrative trust boundary
                         +---------------------------+
                         | tinyidp admin CLI          |
                         | clients/users/keys/backup  |
                         +-------------+-------------+
                                       |
                                       v
+-------------+   TLS   +--------------+--------------+   Go calls   +------------------+
| Browser /   +-------->+ reverse proxy + host service +------------>+ embeddedidp      |
| OIDC client |         | trusted forwarded metadata  |              | http.Handler     |
+------+------+         +-----------------------------+              +--------+---------+
       |                                                                    |
       | redirects, token calls                                             v
       |                                                          +---------+----------+
       +----------------------------------------------------------+ Fosite + product   |
                                                                  | login/consent/UI  |
                                                                  +----+----------+---+
                                                                       |          |
                                                          protocol SQL |          | domain SQL
                                                                       v          v
                                                                  +------------------+
                                                                  | SQLite database  |
                                                                  | clients/users    |
                                                                  | hashes/keys      |
                                                                  | sessions/tokens  |
                                                                  +------------------+
```

The reverse proxy and host service are part of the security system even though
they are outside this repository. The handler itself does not terminate TLS,
configure `http.Server` timeouts, authenticate proxy headers, choose request
body limits, or perform graceful shutdown. A production contract must assign
those responsibilities explicitly.

### Package map

| Package | Responsibility | Start here |
|---|---|---|
| `cmd/tinyidp` | Cobra/Glazed root, help, logging, command registration | `cmd/tinyidp/main.go:26-91` |
| `internal/sections/oidc` | Shared dev CLI section and config decoding | `section.go:36-77`, `settings.go:13-33` |
| `internal/cmds` | `serve`, `print-config`, admin command adapters | `serve.go:82-145`, `admin.go` |
| `internal/server` | Mock engine and scenario/debug/device/DPoP behavior | `server.go`, `authorize.go`, `token.go` |
| `pkg/embeddedidp` | Intended public strict provider API | `options.go`, `provider.go` |
| `internal/fositeadapter` | Fosite composition plus login, consent, sessions, CSRF, discovery, JWKS, rate/audit hooks | `provider.go`, `sqlstore.go` |
| `internal/domain` | Product entities and validation | `types.go`, `validate.go`, `claims.go` |
| `internal/storage` | Store interfaces and cross-store contract suite | `interfaces.go`, `testsuite.go` |
| `internal/store/memory` | Dev/test store | `store.go` |
| `internal/store/sqlite` | Durable store, migrations, raw SQL handle for Fosite | `store.go`, `migrations/*.sql` |
| `internal/authn` | Password verification, dummy hashing, account state, lockout | `password.go` |
| `internal/passwordhash` | Encoded Argon2id hash implementation | `argon2id.go` |
| `internal/keys` | RSA generation, parsing, JWKS, signing, rotation | `keys.go`, `rotation.go` |
| `internal/admin` | Product operations on the domain store | `clients.go`, `users.go`, `keys.go`, `backup.go`, `doctor.go` |
| `internal/oidcmeta` | Issuer parsing and production discovery document | `issuer.go`, `discovery.go` |
| `internal/audit` | Audit event contract and no-op/memory sinks | `audit.go` |

### Public HTTP surface in strict mode

`internal/fositeadapter.Provider.registerAt` installs the following routes
(`internal/fositeadapter/provider.go:264-272`):

| Endpoint | Method | Purpose | Persistent dependencies |
|---|---|---|---|
| `/.well-known/openid-configuration` | GET | Advertise issuer, endpoints, grants, scopes, auth methods, PKCE | Issuer configuration |
| `/jwks` | GET | Publish active and retained RSA verification keys | Key store |
| `/authorize` | GET/POST | Validate authorization request; login/consent; create code | Clients, users, credentials, security state, consent, session, Fosite code state |
| `/token` | POST | Exchange code or rotate refresh token | Fosite code/PKCE/OIDC/access/refresh state, clients, signing key |
| `/userinfo` | GET/POST via Fosite token extraction | Return claims from the stored access-token session | Fosite access-token state |
| `/healthz` | GET | Process liveness | None |
| `/readyz` | GET | Check that an active signing key can be loaded | Key store only |

When the issuer has a path prefix, the same strict routes are registered both at
the root and under the prefix (`provider.go:244-251`). Discovery advertises only
the prefixed routes. The duplicate root surface should be removed or explicitly
specified; it is surprising and expands the reachable surface.

Discovery advertises only implemented strict capabilities: code response type,
authorization-code/refresh grants, RS256, scopes, basic/post/none client auth,
and S256 (`internal/oidcmeta/discovery.go:20-39`). Strict mode intentionally does
not advertise mock device, DPoP, logout, debug, or dynamic-registration
features.

## Core Runtime Flows

### Startup

The intended production startup sequence is:

```text
open durable store
apply/verify schema
load all clients
validate each client under ProductionMode
require HTTPS issuer
require Secure cookies
require >=32-byte Fosite/HMAC/session secret
require persistent store marker
require an active signing key
compose Fosite strategies and handlers
return http.Handler to host
```

That is implemented across `pkg/embeddedidp.Options.Validate`
(`options.go:43-77`) and `fositeadapter.NewProvider`
(`internal/fositeadapter/provider.go:84-182`). Validation currently checks only
the existence of an active key—not algorithm, parseability, `NotBefore`, or
`NotAfter`—and silently installs no-op audit and allow-all rate controls.

### Authorization Code + PKCE

The strict happy path is conceptually:

```text
GET /authorize
    Fosite validates client, exact redirect, response_type, scopes, PKCE
    if valid browser session and policy says no consent:
        mint one-time authorization code
    else:
        issue CSRF cookie/token and render login/consent form

POST /authorize
    parse form
    rate-limit request
    verify CSRF cookie == form token and HMAC
    Fosite re-validates authorization request
    authenticate normalized login against Argon2id credential
    update account-security state
    create opaque browser session; persist only keyed handle hash
    enforce/record consent
    build OIDC claims based on granted scopes
    Fosite persists code/PKCE/OIDC state and redirects to exact client URI

POST /token
    rate-limit request
    Fosite authenticates client and validates code + verifier
    invalidate one-time code
    create opaque access token and optional rotating refresh token
    sign ID token with active RSA key and kid
```

The relevant source is `internal/fositeadapter/provider.go:305-427`. Fosite owns
protocol parsing and response writing. Product code owns login, consent, audit,
scope granting, claims, CSRF, and browser sessions.

### Browser session and CSRF

`createBrowserSession` generates a 32-byte random handle, derives a keyed hash
using the provider secret, persists that hash, and sends only the raw handle in
an `HttpOnly` cookie (`internal/fositeadapter/session.go:14-22`). Reads hash the
cookie and reject revoked/expired sessions or disabled users (`session.go:25-38`).

The CSRF token is a random nonce plus HMAC, double-submitted in an `HttpOnly`
cookie and hidden form field (`internal/fositeadapter/csrf.go:13-40`). Cookie
expiry is 10 minutes. Cookies are issuer-path scoped and hard-code
`SameSite=Lax` (`csrf.go:43-52`; `session.go:21`).

### Password authentication

The data is intentionally separated:

- `domain.User` carries profile/account information;
- `domain.PasswordCredential` carries the encoded hash and lifecycle flags;
- `domain.AccountSecurityState` carries failed attempts, lockout, and last
  successful login (`internal/domain/types.go:31-90`).

Default Argon2id parameters are 64 MiB, three iterations, two lanes, a 16-byte
salt, and 32-byte key (`internal/passwordhash/argon2id.go:34-40`). This exceeds
the captured OWASP minimum. Unknown-user and malformed login paths perform a
dummy verification to reduce account-enumeration timing differences
(`internal/authn/password.go:95-137`). Successful verification opportunistically
rehashes when parameters change (`password.go:139-146`).

The policy and state update implementation is incomplete: `MinLength` is never
checked, `recordFailure` is a non-atomic read-modify-write, its error is ignored,
and the authorization handler ignores `AuthResult.MustChangePassword`.

### Consent and claims

Production defaults to durable consent (`provider.go:104-110`). The consent key
normalizes and sorts scopes, preventing order/duplication mismatch. Consent is
exact-set based: a previously approved subset does not automatically authorize
a later superset (`internal/fositeadapter/consent.go:27-60`).

Claims follow granted scopes:

- `sub` is always present;
- `email` and `email_verified` require `email`;
- name, preferred username, groups, roles, tenant, and locale require `profile`
  (`internal/domain/claims.go:6-34`).

The `sub` validation rejects using the email itself as subject
(`internal/domain/validate.go:83-90`), which supports stable, non-reassignable
subject identifiers.

### Refresh tokens

Fosite issues refresh tokens only when `offline_access` is granted
(`provider.go:143-156`). The SQL adapter persists Fosite request/session state
and marks refresh tokens inactive during rotation. The product domain store has
a separate refresh-token family model and reuse-detection contract, but the
strict Fosite path primarily uses `sqlFositeStore`'s tables. Engineers must not
assume every invariant in `domain.RefreshToken` automatically governs Fosite
tokens.

`sqlFositeStore.RotateRefreshToken` revokes refresh state, then deletes the
access-token row as two independent writes (`internal/fositeadapter/sqlstore.go:279-284`).
The domain store's `RotateRefreshToken` also performs a multi-step update/insert
and optional family revocation without a transaction
(`internal/store/sqlite/store.go:319-381`). These need explicit atomicity tests
on the actual Fosite path.

### Signing keys

Keys are 2048-bit RSA private keys serialized as PKCS#1 PEM inside
`domain.SigningKey`; public JWKS entries expose RS256, kid, modulus, and exponent
(`internal/keys/keys.go:19-67`). ID token creation asks the store for the active
key and includes its `kid` (`internal/fositeadapter/provider.go:475-499`).

Rotation does:

```text
load old active key
generate new RSA key
insert new inactive key
activate new key (deactivate every other key)
retire old key (set NotAfter)
read active key back
```

See `internal/keys/rotation.go:17-53`. The desired invariant—new tokens use the
new key while old tokens remain verifiable—is correct. The sequence is not a
transaction. A crash after activation but before retirement makes the old key
inactive with zero `NotAfter`; `VerificationKeys` omits it because it publishes
only active keys or keys with non-zero `NotAfter`
(`internal/store/sqlite/store.go:465-485`). That creates a validation outage for
already issued ID tokens.

### Admin operations

The Cobra admin tree opens SQLite directly and supports initialization,
migration, doctor, client/user/key lifecycle, backup, and sanitized diagnostics
(`internal/cmds/admin.go:15-36`; `README.md:212-240`). Passwords can come from
stdin so they do not enter shell history (`internal/cmds/admin.go:185-200`).
Generated client secrets are printed once and bcrypt-hashed for storage
(`internal/admin/clients.go:38-81`).

The admin service is useful but not yet transaction-safe. For example,
`CreateUser` writes the user then credential; credential failure leaves a
partial account (`internal/admin/users.go:68-110`). `SetPassword` writes the
credential and resets security state separately (`users.go:120-143`). Key
generation inserts and activates separately (`internal/admin/keys.go:18-38`).

## Storage Model

### Store contract

`internal/storage.Store` embeds ten interfaces for clients, users, credentials,
security state, grants, authorization codes, access/refresh tokens, consent,
sessions, and keys (`internal/storage/interfaces.go:19-100`). All methods take a
context, which is good. The abstraction is entity-oriented, but it lacks a
transaction/unit-of-work contract for cross-entity security operations.

### SQLite representation

The two migrations create domain tables and Fosite protocol tables
(`internal/store/sqlite/migrations/001_schema.sql` and
`002_password_credentials.sql`). Most domain rows store a JSON blob plus a small
number of indexed columns. This keeps domain evolution simple but gives SQLite
few enforceable constraints: relationships, expiry, active-key uniqueness, and
several identity invariants live only in Go.

Important current properties:

- `sqlite.Open` calls `sql.Open` and runs all embedded migrations every time
  (`internal/store/sqlite/store.go:30-40,67-82`).
- There is no schema-version table, checksum, down/forward policy, or
  transaction covering a migration set.
- No busy timeout, journal policy, foreign-key policy, connection limit, or
  synchronous durability setting is established by the store.
- A process-local mutex protects selected compound methods, but not other
  processes/instances and not service-layer read-modify-write sequences.
- No scheduled purge removes expired codes, sessions, consents, tokens, JTI
  rows, or retired keys.
- The database stores password hashes, client-secret hashes, browser state,
  access/refresh protocol sessions, and private signing-key PEM. File access is
  therefore equivalent to compromise of important authentication material.

### Deployment envelope

SQLite can be an appropriate store for a compact single-node embedded IdP, but
the supported envelope must be explicit:

```text
one active process
local durable filesystem (not an unsafe network filesystem)
owner-only database and directory
defined WAL/rollback-journal policy
bounded connection pool and busy timeout
online backup with restore drills
process-level high availability handled by restart/failover, not simultaneous writers
```

If multi-instance active/active service is a requirement, the design should
move to a transactional network database rather than hiding coordination behind
the same interface.

## Security Controls That Are Already Good

The following should be preserved during refactoring:

- Fosite owns OAuth/OIDC parsing, code/refresh behavior, and response writing.
- Strict mode enforces PKCE for all flows and S256 only
  (`provider.go:143-159`). This aligns with RFC 9700's preference for PKCE on
  public and confidential clients.
- Redirect validation is exact and production rejects non-HTTPS redirects except
  loopback (`internal/domain/validate.go:37-80`).
- Strict discovery does not advertise unsupported mock features.
- Authorization and token interactions set no-store/no-cache where relevant.
- HTML values written into hidden form fields are escaped
  (`provider.go:502-517,601-604`).
- Security middleware sets nosniff, deny framing, no-referrer, and restrictive
  CSP (`provider.go:254-261`).
- Browser session handles are random, opaque, `HttpOnly`, secure in production,
  path-scoped, and stored only as keyed hashes.
- CSRF uses random nonces, HMAC, constant-time comparison, and double submission.
- Unknown-account password paths perform dummy Argon2id verification.
- Client secrets are bcrypt-hashed; user passwords are Argon2id-hashed; raw
  passwords are not placed on `domain.User`.
- Stable audit reason codes avoid sending raw library error strings as the
  security event taxonomy.
- The strict engine has dedicated tests for CSRF, headers, consent, prompt,
  sessions, persistence restart, disabled clients, refresh reuse, and debug
  route absence.
- The local strict conformance script focuses on the production-shaped engine,
  not the intentionally permissive mock (`scripts/run-conformance.sh`).

## Detailed Finding Register

### P0-1: Public production API is not externally consumable

**Observed.** `embeddedidp.Options` exposes `storage.Store`, `audit.Sink`,
`fositeadapter.ConsentPolicy`, `fositeadapter.RateLimiter`, and
`fositeadapter.PasswordAuthenticator`, all from `internal/` packages
(`pkg/embeddedidp/options.go:30-40`). The only durable implementation is
`internal/store/sqlite`.

Go forbids an external module from importing those packages. The ticket's
external probe reproduced:

```text
use of internal package github.com/manuel/tinyidp/internal/store/sqlite not allowed
```

**Impact.** The README's production example cannot compile for the intended
consumer. There is no supported production entrypoint: `serve --engine fosite`
is explicitly dev/memory mode, and external embedding is blocked.

**Fix.** This is a pre-release API. Make a direct breaking reorganization—no
compatibility adapter:

```text
pkg/embeddedidp       public provider/options/policy contracts
pkg/idpstore          public domain and Store/Tx contracts
pkg/sqlitestore       public durable SQLite implementation
internal/fositeadapter remains implementation detail behind public contracts
```

**Acceptance.** A separate temporary module imports only public packages,
creates SQLite, provisions client/user/key, constructs ProductionMode, and
completes Authorization Code + PKCE.

### P0-2: Reachable vulnerable runtime and JOSE dependency

**Observed.** The active Go 1.26.1 govulncheck report contains 12 reachable
standard-library vulnerabilities and two reachable `go-jose/v3` issues. Running
the complete suite and govulncheck with Go 1.26.5 removed the standard-library
results but retained:

- GO-2025-3485, go-jose parsing denial of service, fixed in v3.0.4;
- GO-2026-4945, go-jose JWE decryption panic, fixed in v3.0.5.

`go mod why` traces the module through Fosite, and the graph selects v3.0.3.

**Impact.** Untrusted OAuth/OIDC inputs reach Fosite/JOSE parsing. A known
reachable parsing DoS is incompatible with an internet-facing release.

**Fix.** Pin the release builder to a supported patched Go toolchain (at least
the verified 1.26.5 in this review) and select go-jose/v3 v3.0.5 or later,
directly or via a compatible Fosite upgrade. Do not rely on workstation
`GOTOOLCHAIN=auto` to choose the release patch.

**Acceptance.** Tests, race, fuzz, conformance, and govulncheck pass on the exact
release toolchain/graph; the binary SBOM records both.

### P0-3: Backup can silently lose committed WAL data

**Observed.** `CreateSQLiteBackup` opens the main DB path and copies bytes with
`io.Copy` (`internal/admin/backup.go:20-51`). The live probe enabled WAL,
committed a client, observed an 8,272-byte WAL, copied the main file, then opened
the backup successfully. The committed client was absent:

```text
CONFIRMED: backup opens successfully but omits a committed client stored in the source WAL
```

`VerifySQLiteBackup` opens the backup with `sqlite.Open`, which applies
migrations and then only lists clients (`backup.go:80-87`). Verification is
therefore mutating and does not run `integrity_check` or compare expected data.

**Impact.** Operators can receive a green “verified” backup that loses clients,
users, signing keys, or live grants. Restore can cause an authentication outage
or security-state rollback.

**Fix.** Use SQLite's online backup API (preferred for online operation) or a
carefully controlled `VACUUM INTO`. Open verification read-only; run
`PRAGMA integrity_check`; validate schema version; assert essential counts and
an active parseable key; optionally compare a source snapshot manifest. Fsync
the file and directory before reporting success.

**Acceptance.** Concurrent writer + WAL tests repeatedly back up and restore
without missing committed sentinel rows. Failure injection proves no successful
result is emitted for partial output.

### P0-4: Security state transitions lack transactions

**Observed.** Auditlint identifies compound mutations with no `Begin`/`BeginTx`:

- domain refresh rotation and family revocation;
- Fosite refresh/access rotation;
- signing-key activation and retirement;
- user plus credential creation;
- password update plus lockout reset;
- key generation plus activation;
- migration application.

Process-local mutexes do not make independent SQL statements atomic and do not
coordinate another process. Admin service operations span store methods and
therefore cannot share the store's mutex transactionally.

**Impact.** Crash, cancellation, disk-full, busy/locked errors, or concurrency
can leave partial users, multiple/no active keys, lost old JWKS verification
keys, mismatched refresh/access state, or incomplete schema.

**Fix.** Put transaction ownership in the public store contract. Prefer explicit
high-level atomic operations for security-critical invariants, implemented on a
shared `*sql.Tx`, rather than expecting every caller to compose raw entity
methods correctly.

**Acceptance.** Failure-injection tests abort each statement boundary and prove
the old or new state is complete—never intermediate. Add multi-connection
concurrency tests, not just goroutine tests guarded by one store mutex.

### P0-5: Brute-force protection is bypassable

**Observed.** Production defaults to `AllowAllRateLimiter` when nil
(`provider.go:111-113`). When the fixed-window limiter is supplied, the request
key includes `r.RemoteAddr`, which contains IP plus ephemeral source port
(`provider.go:350,407`). New connections can receive fresh buckets. The map
never deletes expired keys (`ratelimit.go:17-50`).

Account lockout reads state, increments it, and writes it as separate calls
(`internal/authn/password.go:181-199`). The write error is ignored by
authentication (`password.go:133-137`). Five simultaneous bad passwords
produced `stored_count=4 locked=false` in the first invariant-probe round.

**Impact.** Attackers can bypass both network and account controls through
connection churn/concurrency. The unbounded rate map also permits memory growth.

**Fix.** Require a production limiter or provide a secure deployment-owned
implementation. Define trusted-proxy parsing with an explicit allowlist, strip
the source port using `net.SplitHostPort`, and combine account, client, and
trusted-IP dimensions. Replace lockout read-modify-write with one atomic SQL
operation/transaction. Propagate storage failures so authentication does not
silently continue without recording the failure. Define cleanup/distributed
semantics.

**Acceptance.** Concurrent attempts always reach the threshold; new TCP ports
do not evade limits; untrusted forwarding headers do not spoof identity; stale
buckets are bounded; database failure follows an explicit fail-open/fail-closed
policy and emits operational alerts.

### P0-6: Sensitive SQLite file permissions are not enforced

**Observed.** `sqlite.Open` passes the path directly to the driver and does not
pre-create or validate the file/directory (`internal/store/sqlite/store.go:30-40`).
With process umask set to `000`, the invariant probe observed mode `0644`.

The same file includes Argon2id hashes, bcrypt client-secret hashes, private RSA
PEM, browser sessions, and OAuth protocol state.

**Impact.** Another local user/container process with filesystem access can read
private signing keys and credential material. Database theft permits offline
password guessing and token signing.

**Fix.** Require an owner-only directory; create a new file with `0600`; reject
or repair unsafe existing modes according to explicit operator policy; document
UID/GID and Kubernetes secret-volume expectations; support encrypted storage or
external key custody where threat models require it. Backups must inherit the
same confidentiality.

**Acceptance.** Permission tests run under permissive umask and on existing
unsafe files. Admin doctor reports owner/mode/path and fails production preflight
when policy is not met.

### P1-1: Password lifecycle policy is only partially implemented

`DefaultPasswordPolicy` says minimum eight characters
(`internal/authn/password.go:43-45`), but create/set-password only reject empty
input (`internal/admin/users.go:68-75,120-127`). The probe created and
authenticated a one-character password. `MustChangeAtLogin` is persisted and
returned in `AuthResult` (`password.go:146`), but the authorization handler uses
only `result.User` (`provider.go:369-388`).

**Required outcome.** Centralize password acceptance policy in the password
service and call it from every credential-write path. Enforce minimum/maximum
length and any breached/common-password policy. Define a password-change flow;
until it exists, reject `--must-change` rather than setting a flag that does
nothing. Revoke sessions/refresh grants on password change according to policy.

### P1-2: Signing-key validity and lifecycle can fail open

Production validation requires only that `ActiveSigningKey` returns a row
(`pkg/embeddedidp/options.go:73-75`). The probe constructed a provider with an
active key whose `NotAfter` was yesterday. The signing callback parses the PEM
but does not enforce time/algorithm metadata (`provider.go:232-242`). Admin can
retire an active key without requiring a replacement (`internal/admin/keys.go:54-63`).

**Required outcome.** Validate algorithm, key size, PEM parse, public/private
match, `NotBefore <= now < NotAfter` when dates exist, exactly one active key,
and retained-key horizon. Make activation/retirement atomic. Prohibit retiring
the last usable active key. Make `/readyz` perform the same checks, not existence
only.

### P1-3: Audit is optional and delivery errors disappear

`audit.Sink` is a good boundary, but nil becomes `NoopSink` in provider,
password, and admin constructors. Every call discards `Emit` errors. The runtime
probe did observe nine events for its flow, showing useful coverage when a sink
is present; there is no durable sink, queue, backpressure, health, or drop
metric in the product.

**Required outcome.** Production validation requires an explicit sink. Define
which events must be durable, whether request handling fails open, buffer bounds,
redaction, PII retention, correlation/request IDs, delivery health, and dropped
event metrics. Admin operations should use the same operational audit policy.

### P1-4: Schema, retention, and maintenance are not production operations

Migrations are a sorted set of `CREATE IF NOT EXISTS` scripts executed every
open. There is no version/checksum record or transactional plan. Expired rows
are generally rejected at lookup but not purged. JTI cleanup happens only when
that JTI is checked. Retired key cleanup is documented but not implemented.

**Required outcome.** Add versioned transactional migrations, startup
compatibility checks, dry-run plan, downgrade policy, and backup-before-migrate.
Add bounded maintenance for expired codes/sessions/tokens/consents/JTIs and key
retention based on maximum issued token lifetime plus skew. Publish row counts,
DB size, cleanup age, and last successful maintenance metrics.

### P1-5: Deployment/HTTP boundary is not executable

The strict embedding API returns only `http.Handler`, which is reasonable if
the host contract is strong. The repo does not provide a compilable production
host. `serve` uses package-level `http.ListenAndServe`, permissive mock CORS, no
timeouts, and no graceful `Shutdown` (`internal/cmds/serve.go:95-145`). Gosec
and auditlint both report this surface; documentation correctly says `serve` is
not production.

**Required outcome.** Provide a public, compiling production-host example and a
deployment checklist that owns `ReadHeaderTimeout`, read/write/idle policy,
header/body limits, max concurrent password work, graceful shutdown, TLS
termination, host validation, trusted proxies, HSTS at the TLS edge, and
end-to-end protection between proxy and application. Do not silently turn the
dev command into production by adding flags around its memory/scenario setup.

### P1-6: Release automation is missing

The repository has strong Make targets but no checked-in `.github` workflows.
The documented release gate is mostly `go test` plus local/hosted conformance.
It does not pin a patched toolchain, run race/govulncheck/auditlint/backup/API
probes, produce SBOM/provenance, or test restore.

**Required outcome.** CI and release workflows must reproduce the acceptance
matrix in this document. Hosted OIDC conformance evidence must be tied to the
exact commit and dependency graph, not reused from an earlier run.

### P2 implementation-quality findings

These are not independent ship blockers once the P0/P1 work is complete, but
they should be handled deliberately:

- `CookieConfig.SameSite` is public and ignored; cookies always use Lax.
- Per-client access/ID/refresh TTL fields are stored and printed by admin, but
  Fosite uses provider-global TTLs; the public contract is misleading.
- `randomB64` in strict code and admin user ID generation ignore CSPRNG errors
  (`provider.go:606-609`; `internal/admin/users.go:172-175`). Fail closed.
- `Issuer.ParseIssuer` silently removes query and fragment rather than rejecting
  invalid issuer input (`internal/oidcmeta/issuer.go:14-22`). Prefer validation.
- Path issuers expose root aliases in addition to advertised prefixed routes.
- `LastSeenAt` is set on browser session creation but never refreshed.
- `FixedWindowRateLimiter` has no bucket cleanup or multi-process semantics.
- Readiness validates only active-key existence, not SQLite writeability,
  migration compatibility, key validity, or audit/maintenance health.
- The CSP `form-action 'self' https:` is broader than necessary; after the
  canonical public URL contract is fixed, restrict it to the required origin.
- The login/consent HTML is intentionally minimal. A production UI needs
  accessibility, localization, branded error paths, and secure template tests,
  without adding third-party resources to authorization responses.
- Private key material is plaintext inside the database. File permissions are
  the minimum; high-assurance deployments may need envelope encryption or
  external KMS/HSM signing.
- No password reset/recovery, MFA, user self-service, session listing/revocation,
  `/revoke`, `/introspect`, or strict logout exists. These are product-scope
  decisions; discovery must continue to omit unsupported endpoints.

## Scanner Triage

The product-only gosec report contains 15 findings. They are not all confirmed
vulnerabilities:

| Rule | Review result |
|---|---|
| G114 zero-timeout HTTP server | Confirmed on dev `serve`; production host contract still missing. |
| G710 open redirects | False positive after manual flow review: both sites require exact registered URI matching before redirect. Keep tests. |
| G124 cookie attributes | Mostly analysis limitation: strict cookies set `HttpOnly`, `SameSite=Lax`, and a runtime `Secure` boolean that ProductionMode requires true. Mock cookies are intentionally insecure/loopback. |
| G101 hardcoded credentials | Mock debug/scenario data, not strict stored credentials. Keep mock isolated. |
| G304 variable file paths | CLI/operator-selected users/DB/backup paths. Not arbitrary remote inclusion; still require safe privileges, symlink policy, and owner-only creation. |
| G301 backup directory `0755` | Confirmed confidentiality concern combined with DB content and observed file mode. |
| G115 int to uint32 in Argon2 parser | Defensive robustness issue for corrupted enormous encoded hashes; add maximum decoded salt/key and work-factor bounds. |

Narrow documented suppressions may be added after fixes. Do not globally disable
the rules.

## Runtime Evidence

The probe is a correctness/diagnostic baseline, not a capacity benchmark. It
uses `httptest`, local SQLite, one process, and 40 bounded concurrent reads.

| Operation | Observation |
|---|---:|
| GET authorize interaction | ~0.53 ms |
| POST authorize/password/consent | ~82 ms |
| Token exchange | ~6.3 ms |
| Refresh | ~4.8 ms |
| Discovery p95 | ~0.15 ms |
| JWKS p95 | ~0.38 ms |
| Ready p95 | ~0.22 ms |
| UserInfo load p95 | ~0.49 ms |
| Go heap allocations across flow/load | ~67.9 MB |
| Live heap delta | ~0.15 MB |
| Goroutine delta after closing idle connections | 0 |
| Audit events | 9 |
| SQLite pool waits | 0 |

The allocation is consistent with one production Argon2id verification at 64
MiB plus overhead. Capacity planning must bound simultaneous password hashes.
For example, 100 concurrent login verifications can demand roughly 6.4 GiB of
working memory before normal process/database overhead. Use a semaphore or
worker budget plus rate limiting; do not weaken hashing just to survive
unbounded concurrency.

CPU and heap profiles are stored under `various/runtime/`. This machine's Go
distribution lacks the `pprof` tool, so profile interpretation remains a review
follow-up.

## Gap Matrix Against Guidance

| Area | Guidance baseline | Current state | Assessment |
|---|---|---|---|
| Redirects | RFC 9700 exact matching | Exact registered string matching | Meets baseline |
| Authorization flow | Code; PKCE supported/required; S256 | Code only; PKCE all production; S256 only | Strong |
| Implicit grant | Avoid | Not composed/advertised | Meets baseline |
| Refresh replay | Rotation or sender constraint for public clients | Fosite rotation tests exist; atomicity incomplete | Blocked on transaction proof |
| Token privilege | Restrict scopes/audience | Exact client scope grant; requested audience granted without a product allowlist | Review resource/audience model |
| Authentication throttling | Account-associated throttling plus defense in depth | Account lockout race; optional/bypassable IP limiter | Fails |
| Authentication logging | Log success, failure, lockout; monitor | Events exist; optional/no-op/error loss; lockout-specific operations incomplete | Fails operationally |
| Password storage | Modern salted adaptive hash | Argon2id 64 MiB, t=3, p=2 | Strong primitive |
| Password acceptance | Enforce policy consistently | Minimum and must-change not enforced | Fails lifecycle |
| SQLite live backup | Online backup or controlled snapshot | Raw main-file copy | Fails |
| Go secure development | govulncheck, race, fuzz, vet | Manually run in ticket; no CI; reachable dependency issues | Fails release gate |
| HTTP server hardening | Timeouts, limits, shutdown | Owned externally but no production host contract | Incomplete |

## Proposed Production Architecture

### Public packages and ownership

```text
pkg/idp
    Mode, Client, User/profile DTOs needed by consumers
    AuditEvent/AuditSink
    RateLimiter/ClientAddressResolver
    Authenticator/ConsentPolicy contracts

pkg/idpstore
    Store, TxStore, atomic security operations

pkg/sqlitestore
    Config{Path, BusyTimeout, JournalMode, MaxOpenConns, PermissionPolicy}
    Open / Migrate / Backup / Verify / Maintenance

pkg/embeddedidp
    Options using only public or standard-library types
    New(ctx, Options) (*Provider, error)
    Handler(), Readiness(ctx), Close(ctx)

internal/fositeadapter
    maps public contracts to Fosite; no external consumer sees Fosite types
```

Do not create aliases/adapters solely to preserve the current public surface.
It has not shipped and is unusable externally; direct replacement is simpler
and safer.

### Suggested API sketch

```go
type Options struct {
    Issuer        string
    Mode          Mode
    Store         idpstore.Store
    TokenSecret   SecretSource
    Cookies       CookiePolicy
    Audit         idp.AuditSink
    RateLimiter   idp.RateLimiter
    ClientAddress idp.ClientAddressResolver
    Authenticator idp.PasswordAuthenticator
    Consent       idp.ConsentPolicy
    Clock         func() time.Time
}

func New(ctx context.Context, opts Options) (*Provider, error)

type Provider interface {
    Handler() http.Handler
    Readiness(ctx context.Context) ReadinessReport
    Close(ctx context.Context) error
}
```

Production validation should fail closed on missing audit/limiter/address
policy, unsafe store permissions, invalid/expired signing key, unsupported
schema, or unavailable secret material.

### Atomic store sketch

```go
type Store interface {
    View(ctx context.Context, fn func(ReadStore) error) error
    Update(ctx context.Context, fn func(TxStore) error) error

    RecordFailedLogin(ctx context.Context, userID string, now time.Time, policy LockoutPolicy) (AccountSecurityState, error)
    CreateUserWithCredential(ctx context.Context, user User, credential PasswordCredential) error
    ReplacePasswordAndSecurityState(ctx context.Context, credential PasswordCredential, state AccountSecurityState) error
    RotateSigningKey(ctx context.Context, next SigningKey, now time.Time) (RotationResult, error)
}
```

The high-level methods encode invariants. `Update` remains useful for admin
workflows but should not be the only protection.

### Online backup sketch

```text
create temp file in owner-only backup directory
start SQLite online backup from live source connection
copy pages with retry/busy handling
finish backup and close destination
fsync destination file and parent directory
open destination read-only (never migrate)
run integrity_check
read schema version
validate active key/client/user counts against source manifest
atomically rename temp file to requested backup name
emit audit event and result only after every check succeeds
```

### Login control sketch

```text
resolve trusted client address using explicit proxy allowlist
acquire global password-work semaphore
check account/client/IP limiter
perform constant-cost user lookup + Argon2id verify
if invalid:
    atomically increment account failure state
    derive lockout result in same transaction
    emit required audit event or delivery-health signal
    return generic credentials error
if MustChangeAtLogin:
    create restricted password-change transaction, not OIDC grant
else:
    atomically reset security state
    issue browser session
```

## Decision Records

### Decision: Do not ship the current commit

- **Context:** Protocol happy paths pass, but production construction, backup,
  dependencies, security-state atomicity, and file confidentiality have proven
  blockers.
- **Options considered:** Ship with operational warnings; ship only embedded;
  postpone until P0/P1 acceptance.
- **Decision:** No-go until the defined acceptance gate passes.
- **Rationale:** Several issues can cause credential compromise, brute-force
  bypass, data loss, token validation outage, or inability to deploy at all.
- **Consequences:** The next work is remediation, not release packaging.
- **Status:** accepted for this review.

### Decision: Replace the public API directly

- **Context:** Public `embeddedidp.Options` exposes internal types and cannot be
  consumed externally.
- **Options considered:** Compatibility aliases/adapters; a public facade over
  internal stores; direct move/redesign of contracts.
- **Decision:** Directly replace the pre-release API with public contracts and a
  public SQLite package.
- **Rationale:** Compatibility adds complexity without preserving a working
  external contract.
- **Consequences:** Examples/tests/docs update together; no shim.
- **Status:** proposed.

### Decision: Store owns atomic security operations

- **Context:** Entity interfaces cannot guarantee cross-row/cross-table
  invariants.
- **Options considered:** Service mutexes; expose raw `*sql.Tx`; generic
  `WithTx`; high-level atomic store methods plus transactions.
- **Decision:** Add high-level atomic operations backed by explicit transaction
  support.
- **Rationale:** Invariants remain testable and work across implementations;
  process mutexes are insufficient.
- **Consequences:** Store interfaces change and memory/SQLite suites expand.
- **Status:** proposed.

### Decision: Use SQLite online backup

- **Context:** Raw file copy loses WAL commits.
- **Options considered:** Stop server and copy; force checkpoint then copy;
  `VACUUM INTO`; online backup API.
- **Decision:** Online backup for normal operation, with optional offline backup
  only under an explicit shutdown lock.
- **Rationale:** Correct snapshot semantics during live writes and official
  SQLite support.
- **Consequences:** Driver-specific implementation belongs in public
  `sqlitestore`; restore validation becomes first-class.
- **Status:** proposed.

### Decision: Keep HTTP ownership with host, publish a strict contract

- **Context:** An embedded handler cannot choose every host's TLS/proxy/server
  settings.
- **Options considered:** Provider starts its own server; handler only with
  prose; handler plus production host example/preflight contract.
- **Decision:** Keep handler ownership with the host and provide a compiling,
  tested production host example plus readiness/preflight APIs.
- **Rationale:** Embedding remains flexible without hiding essential controls.
- **Consequences:** The contract is part of release acceptance.
- **Status:** proposed.

### Decision: Support SQLite as single-active-node first

- **Context:** SQLite is attractive for a tiny embedded IdP but does not provide
  transparent distributed coordination.
- **Options considered:** Claim multi-instance support; add process locks;
  single-node documented envelope; network database immediately.
- **Decision:** Ship SQLite only for one active process on local durable storage.
- **Rationale:** Honest, testable envelope; avoids unsafe pseudo-HA.
- **Consequences:** HA means fast restart/volume failover until another store is
  designed.
- **Status:** proposed.

## Phased Implementation Plan

### Phase 0: Patch the release graph

Files: `go.mod`, `go.sum`, release workflow/toolchain configuration.

1. Select patched Go >=1.26.5.
2. Select `go-jose/v3` >=3.0.5 and verify Fosite compatibility.
3. Run build, full tests, race, lint, fuzz seeds, conformance, and govulncheck.
4. Add SBOM/provenance output.

Exit: zero reachable known vulnerabilities or approved expiry-bound exception.

### Phase 1: Make production construction real

Files/packages: `pkg/embeddedidp`, new `pkg/idp`, `pkg/idpstore`,
`pkg/sqlitestore`, README/examples.

1. Move public domain/policy/store types out of `internal`.
2. Redesign `Options` with public types only and a context-aware constructor.
3. Publicly expose the durable SQLite implementation.
4. Delete the unusable pre-release interface; do not add compatibility shims.
5. Convert the external API probe to a positive integration test.

Exit: an external module compiles and completes a strict flow.

### Phase 2: Repair persistence and backup

Files: SQLite store/migrations, Fosite SQL store, admin users/keys/backup.

1. Add transaction support and high-level atomic methods.
2. Make user+credential, password+state, refresh rotation/reuse, key rotation,
   and migrations atomic.
3. Add active-key uniqueness at the schema/application transaction level.
4. Add schema versions/checksums and transactional migration plans.
5. Implement online backup, read-only verify, integrity/schema/content checks,
   fsync, and atomic destination replacement.
6. Enforce DB/backup directory and file permissions.
7. Configure/document busy timeout, journaling, synchronous policy, pool limits,
   and supported filesystem.

Exit: failure-injection, concurrency, WAL backup, and restore suites pass.

### Phase 3: Repair authentication controls

Files: `internal/authn`, admin user commands, strict provider/rate limiter.

1. Enforce password acceptance policy on create/reset.
2. Implement or remove must-change until a real flow exists.
3. Make failed-login counting atomic and propagate storage errors.
4. Require production rate limiting and trusted-client-address policy.
5. Bound concurrent Argon2id work.
6. Define password-change session/refresh revocation.

Exit: invariant probe becomes positive; concurrent abuse tests pass.

### Phase 4: Harden keys, audit, readiness, and maintenance

Files: `internal/keys`, store key operations, audit implementations,
`embeddedidp.Options`, readiness, maintenance command/service.

1. Validate active-key time, algorithm, size, parseability, uniqueness.
2. Make rotation atomic and prevent last-key retirement.
3. Require audit sink and define delivery/drop/health semantics.
4. Add maintenance/retention operations and metrics.
5. Expand readiness to schema, key, store, audit, and maintenance checks.
6. Resolve `SameSite`, per-client TTL, issuer validation, RNG error, and path
   route contracts.

Exit: production preflight fails on every known unsafe configuration.

### Phase 5: Release engineering and deployment proof

1. Add CI for build/test/race/lint/auditlint/gosec/govulncheck/fuzz seeds.
2. Add external consumer and live backup/restore jobs.
3. Build a production host example with timeouts, shutdown, request limits, and
   proxy policy.
4. Run sustained login/token/read load with realistic Argon2id concurrency and
   observe DB/runtime/audit metrics.
5. Run a new hosted OpenID Foundation profile on the exact release artifact.
6. Perform backup restore, key rotation, token-secret rotation plan, downgrade,
   and incident-response drills.

Exit: release checklist signed with artifact hashes and evidence links.

## Testing and Validation Strategy

### Always-on CI

```text
go build ./...
go test ./... -count=1
go vet ./...
make lint
go run ./ticket/scripts/auditlint ./cmd/... ./internal/... ./pkg/...
govulncheck ./...
external consumer integration
SQLite permission/backup/restore tests
local strict conformance
```

### Scheduled or release CI

- `go test -race ./... -count=1`;
- longer fuzzing for issuer, redirect, Argon2 hash, request-object/JWT, form, and
  persisted-request decoding;
- multi-connection SQLite concurrency and fault injection;
- sustained password/login load with memory budget checks;
- hosted OIDC conformance;
- restore drill and migration from previous released schema;
- dependency license/SBOM/provenance/signature verification.

### New required tests by finding

| Finding | Required regression |
|---|---|
| Public API | External module positive compile and complete flow |
| Backup | WAL + concurrent writer snapshot/restore sentinel preservation |
| Transactions | Fail each statement boundary; assert no intermediate state |
| Lockout | Five simultaneous failures always count and lock |
| Rate limit | Port churn and spoofed proxy headers do not reset identity |
| Permissions | New/existing DB and backup owner/mode policy |
| Password policy | Empty, one-char, max, Unicode, must-change behavior |
| Keys | Expired/not-yet-valid/bad PEM/multiple active/last retire |
| Audit | Sink outage, buffer overflow, redaction, correlation, health |
| Retention | Expiry purge safety and retired-key horizon |
| Host | Slow headers/body, graceful shutdown, proxy trust, canonical host |

## Operational Release Checklist

Before production traffic:

1. Record exact binary hash, Go version, module graph, SBOM, and source commit.
2. Run admin preflight against a copy of production configuration/data.
3. Verify database directory/file ownership, `0600` policy, local filesystem,
   free space, journal settings, and backup destination permissions.
4. Verify issuer/canonical public URL, TLS certificate, HSTS, trusted proxies,
   forwarded-header sanitization, and end-to-end proxy-to-app protection.
5. Verify `http.Server` timeouts, body/header limits, shutdown deadline, and
   Argon2id concurrency budget.
6. Verify every client redirect, public/confidential type, PKCE, allowed scopes,
   secret rotation process, and token TTL behavior.
7. Verify active signing key validity, retained keys, JWKS cache behavior, and
   rotation/rollback drill.
8. Verify durable audit ingestion, alerts for auth failure/lockout/rate limit,
   dropped events, readiness, DB growth, backup, cleanup, and key expiry.
9. Create an online backup, restore it into isolation, run integrity/preflight,
   and perform a real login/token flow against the restore.
10. Run local and hosted conformance on the final artifact.
11. Define rollback boundaries: binary-only rollback is unsafe after an
    irreversible schema migration.
12. Document incident actions for stolen DB/signing key/token secret, including
    key replacement, session/grant revocation, client notification, and audit
    preservation.

## Risks and Alternatives

### Alternative: ship mock/serve behind a private network

Rejected as production. Network isolation does not convert scenario users,
memory state, fixed secrets, permissive CORS, or dev mode into a durable IdP.

### Alternative: stop writes and copy SQLite

Viable only as an explicit offline backup with a proven exclusive shutdown lock
and fsync. It should not be the normal online backup implementation.

### Alternative: replace SQLite immediately

Not required for a single-active-node product. It becomes the appropriate
choice if active/active, remote transactions, or stronger operational tooling
are requirements.

### Alternative: make audit/rate limiting the reverse proxy's job

The proxy can provide valuable layers, but account-associated lockout and
protocol/security event audit require application context. Define both layers;
do not let each assume the other owns the control.

## Open Questions

1. Is the supported production topology one process/one SQLite volume, or is
   active/active required?
2. Is tiny-idp intended as a general library, a standalone product, or both?
   The public package and operational host should reflect one explicit answer.
3. Which strict features are required for v1: logout, revocation,
   introspection, MFA, recovery, DPoP, device flow?
4. What are the maximum user/client/session counts and login/token rates?
5. What are the required RPO/RTO, backup retention, restore time, and key
   compromise response?
6. Must private signing keys use KMS/HSM, encrypted SQLite, or is owner-only
   local storage acceptable?
7. What PII/audit retention and deletion rules apply?
8. Are per-client token TTLs a real requirement? If not, remove the fields and
   CLI flags rather than keeping a false contract.
9. What trusted proxy products/topologies must client-address resolution
   support?
10. Which exact Go release policy and supported OS/CGO/SQLite matrix will the
    product publish?

## References

### Key repository files

- `README.md`: product intent, engine split, documented production path.
- `pkg/embeddedidp/options.go`: public API and production validation.
- `pkg/embeddedidp/provider.go`: adapter construction.
- `internal/fositeadapter/provider.go`: strict runtime and endpoint flows.
- `internal/fositeadapter/sqlstore.go`: durable Fosite protocol state.
- `internal/fositeadapter/session.go`, `csrf.go`, `consent.go`, `ratelimit.go`:
  browser/product controls.
- `internal/storage/interfaces.go`: current persistence contracts.
- `internal/store/sqlite/store.go`: durable domain store.
- `internal/store/sqlite/migrations/*.sql`: schema.
- `internal/authn/password.go`: authentication and lockout.
- `internal/passwordhash/argon2id.go`: hash format/work factors.
- `internal/keys/keys.go`, `rotation.go`: signing/JWKS lifecycle.
- `internal/admin/*.go`, `internal/cmds/admin*.go`: operations.
- `internal/oidcmeta/*.go`: issuer/discovery metadata.
- `docs/security-profile.md`, `docs/storage.md`, `docs/conformance.md`: existing
  intended contracts.

### Ticket evidence

- `reference/01-investigation-diary.md`: chronological commands, failures, and
  decisions.
- `scripts/README.md`: reproduction commands.
- `scripts/auditlint`: typed repository-specific analyzers.
- `scripts/external-api-smoke.sh`: external visibility reproduction.
- `scripts/sqlite-backup-probe.go`: WAL backup data-loss reproduction.
- `scripts/security-invariants-probe`: runtime security checks.
- `scripts/runtime-probe`, `scripts/runtime-analyze`: bounded flow metrics.
- `various/auditlint.txt`: typed diagnostics.
- `various/runtime/summary.md`: runtime summary.
- `various/gosec-product.json`: scoped scanner output.
- `various/govulncheck-go1.26.1.txt` and
  `govulncheck-go1.26.5.txt`: reachability comparison.

### External guidance captured under `sources/`

- OpenID Connect Core 1.0 incorporating errata set 2:
  <https://openid.net/specs/openid-connect-core-1_0.html>
- RFC 9700, OAuth 2.0 Security Best Current Practice:
  <https://www.rfc-editor.org/rfc/rfc9700.html>
- SQLite Online Backup API: <https://www.sqlite.org/backup.html>
- SQLite Write-Ahead Logging: <https://www.sqlite.org/wal.html>
- SQLite How To Corrupt Your Database: <https://www.sqlite.org/howtocorrupt.html>
- Go Security Best Practices: <https://go.dev/doc/security/best-practices>
- Go `net/http` API: <https://pkg.go.dev/net/http>
- OWASP Authentication Cheat Sheet:
  <https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html>
- OWASP Password Storage Cheat Sheet:
  <https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html>
