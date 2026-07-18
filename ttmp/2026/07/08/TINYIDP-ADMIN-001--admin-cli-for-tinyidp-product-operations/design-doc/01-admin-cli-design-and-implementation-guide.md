---
Title: Admin CLI Design and Implementation Guide
Ticket: TINYIDP-ADMIN-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp/main.go
      Note: Root command wiring where admin command tree should be added
    - Path: repo://internal/domain/types.go
      Note: Client, user, and key models administered by CLI
    - Path: repo://internal/storage/interfaces.go
      Note: Store capabilities the admin service should use
    - Path: repo://internal/store/sqlite/migrations/001_schema.sql
      Note: Schema managed by admin init/migrate/backup commands
    - Path: repo://pkg/embeddedidp/options.go
      Note: Validation behavior admin doctor should mirror
ExternalSources: []
Summary: Design for adding a safe operational admin CLI for tinyidp clients, users, keys, migrations, backups, and diagnostics.
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: Use this when implementing tinyidp admin commands or changing operational storage workflows.
WhenToUse: Read before adding init, migrate, client, user, key, backup, restore, doctor, or export commands.
---


# Admin CLI Design and Implementation Guide

## Executive Summary

`tinyidp` has the protocol core needed for a strict embedded IdP, but operational management is still missing. Operators need a way to initialize storage, inspect and migrate the database, create and disable clients, create and manage users, rotate signing keys, validate configuration, and produce sanitized diagnostics. This guide proposes a structured `tinyidp admin` command tree backed by a small internal admin service layer.

The admin CLI should not be a separate product bolted onto the side. It should use the same config loader, SQLite store, domain models, key rotation functions, and validation rules as the server. The result should be a command surface that can be scripted safely in CI, copied into runbooks, and used by interns without requiring them to know the details of Fosite protocol tables.

## Problem Statement

The current binary has a very small command surface. It can serve and print config, but it cannot manage production state.

Evidence from the current codebase:

- `cmd/tinyidp/main.go:51-88` wires `serve` and `print-config` as the only application commands beneath the root command.
- `internal/cmds/serve.go:52-79` describes `serve` as a mock/local development command and composes only the OIDC section plus command settings.
- `internal/storage/interfaces.go:19-92` already exposes stores for clients, users, grants, authorization codes, access tokens, refresh tokens, consents, sessions, and keys. The CLI does not yet expose operational commands for those stores.
- `internal/store/sqlite/migrations/001_schema.sql:1-20` already defines durable tables, including clients, users, signing keys, and Fosite protocol tables. Operators need migration and backup tools around those tables.
- `internal/domain/types.go:14-29` models OAuth clients with disabled state, secret hash, redirect URIs, scopes, TTLs, and PKCE flags. Creating or changing these by hand in SQLite would be unsafe.
- `pkg/embeddedidp/options.go:42-74` has startup validation, but operators need a preflight command that runs similar checks before they restart a server.

Without an admin CLI, the only way to manage product state is by editing config, writing ad hoc scripts, or opening SQLite directly. That is error-prone and makes incident response harder.

## Scope

In scope:

- `tinyidp admin` command tree design.
- Store-backed operations for clients, keys, users, migrations, backups, diagnostics, and sanitized exports.
- A service layer that keeps command handlers thin.
- JSON/table output modes suitable for automation and humans.
- Safety features: dry-run, confirmation, redaction, exact validation errors, idempotent create/update behavior.

Out of scope:

- Full user/password credential implementation details. Those are covered by `TINYIDP-USERS-001`.
- The full structured config loader. That is covered by `TINYIDP-PROD-CONFIG-001`.
- Web-based administration UI.

## Design Goals

1. Keep command handlers small and boring.
2. Reuse domain and storage validation instead of duplicating business rules in Cobra handlers.
3. Make every destructive operation explicit, confirmable, and scriptable.
4. Never print raw secrets by default.
5. Support both human table output and machine JSON output.
6. Make production initialization repeatable.
7. Keep mock/test fixtures separate from production admin state.

## Proposed Command Tree

```text
tinyidp admin
  init
  migrate
  doctor
  client
    list
    get
    create
    update
    disable
    enable
    rotate-secret
    delete-or-archive
  user
    list
    get
    create
    disable
    enable
    lock
    unlock
    set-password
    require-password-reset
  keys
    list
    generate
    import
    rotate
    retire
    jwks
  sessions
    revoke-user
    revoke-client
    cleanup-expired
  consents
    list
    revoke
  tokens
    revoke-grant
    cleanup-expired
  backup
    create
    restore
    verify
  export
    diagnostics
    conformance-evidence
```

The first implementation should be smaller:

```text
tinyidp admin init
tinyidp admin migrate
tinyidp admin doctor
tinyidp admin client create/list/get/disable/rotate-secret
tinyidp admin user create/list/get/disable/set-password
tinyidp admin keys generate/list/rotate/retire
tinyidp admin backup create/verify
```

## Runtime Architecture

```text
Cobra/Glazed command
      |
      v
admin command settings
(config path, output mode, dry-run, yes)
      |
      v
appconfig.Load + appconfig.BuildAdminRuntime
      |
      v
internal/admin.Service
      |
      v
storage.Store + domain validators + key/password helpers
      |
      v
rows / JSON / redacted diagnostics
```

The admin service should be a library-style package that can be tested without launching the CLI.

```go
package admin

type Service struct {
    Store storage.Store
    Clock func() time.Time
    SecretHasher SecretHasher
    Passwords PasswordService
    Keys KeyService
    Audit audit.Sink
}

type ClientCreateRequest struct {
    ID string
    Public bool
    PlainSecret []byte
    RedirectURIs []string
    PostLogoutRedirectURIs []string
    AllowedScopes []string
    RequirePKCE bool
    AccessTokenTTL time.Duration
    IDTokenTTL time.Duration
    RefreshTokenTTL time.Duration
}

func (s *Service) CreateClient(ctx context.Context, req ClientCreateRequest) (domain.Client, SecretResult, error)
func (s *Service) DisableClient(ctx context.Context, id string) error
func (s *Service) RotateClientSecret(ctx context.Context, id string) (SecretResult, error)
func (s *Service) GenerateSigningKey(ctx context.Context, req KeyGenerateRequest) (domain.SigningKey, error)
func (s *Service) RotateSigningKey(ctx context.Context, req KeyRotateRequest) error
func (s *Service) Doctor(ctx context.Context) (DoctorReport, error)
```

The CLI layer should map flags to request structs, call the service, and render the result.

## Command Design Details

### `admin init`

Purpose: create an empty production database, run migrations, create initial signing key, and optionally create an initial admin/user.

Example:

```bash
tinyidp admin init \
  --config-file /etc/tinyidp/config.yaml \
  --generate-signing-key \
  --admin-login admin@example.com \
  --password-from-stdin
```

Expected behavior:

- Fail if the target database exists unless `--if-not-exists` is set.
- Run SQLite migrations.
- Create exactly one active signing key unless the config already imports one.
- Create an initial user only through the user/password service once that exists.
- Print a redacted summary.

Pseudocode:

```go
func RunAdminInit(ctx context.Context, flags InitFlags) error {
    cfg, err := appconfig.Load(ctx, flags.ConfigFile, osEnv{})
    if err != nil { return err }
    rt, err := appconfig.BuildAdminRuntime(ctx, cfg)
    if err != nil { return err }
    svc := admin.NewService(rt.Store)
    if err := svc.Init(ctx, admin.InitRequest{GenerateSigningKey: flags.GenerateSigningKey}); err != nil { return err }
    return render(flags.Output, svc.Summary(ctx))
}
```

### `admin migrate`

Purpose: apply schema migrations and report current schema version.

The current SQLite schema is embedded in `internal/store/sqlite/migrations/001_schema.sql:1-20`. The migration layer should grow from single embedded SQL to versioned migrations with a metadata table.

Example:

```bash
tinyidp admin migrate --config-file /etc/tinyidp/config.yaml --dry-run
tinyidp admin migrate --config-file /etc/tinyidp/config.yaml
```

Rules:

- `--dry-run` lists pending migrations without applying them.
- Applying migrations must be transactional where SQLite allows it.
- Failed migrations must return a clear error and not start the server.

### `admin doctor`

Purpose: run preflight checks without serving traffic.

Checks:

- config loads and validates;
- database opens;
- migrations are current;
- active signing key exists;
- clients validate;
- no active key is expired;
- no disabled client is referenced by active grants, if that query is later available;
- redaction check passes for diagnostics;
- optional: issuer discovery URL is reachable from the local machine.

Example output:

```text
OK config.version: 1
OK server.public_issuer: https://idp.example.com
OK storage.sqlite.path: /var/lib/tinyidp/tinyidp.db
OK keys.active: 2026-07-08-rsa-1
WARN keys.retired: 1 retired key can be removed after 2026-10-08
OK clients: 3 active, 1 disabled
```

### `admin client create`

Purpose: create a relying-party/OAuth client safely.

Example:

```bash
tinyidp admin client create \
  --id web-app \
  --secret-from-stdin \
  --redirect-uri https://app.example.com/oauth/callback \
  --scope openid --scope profile --scope email \
  --scope offline_access \
  --require-pkce \
  --access-token-ttl 1h \
  --id-token-ttl 1h \
  --refresh-token-ttl 720h \
  --output json
```

Rules:

- Secret is read from stdin, env, or generated; never from shell history by default.
- Secret is hashed before storing because `domain.Client.SecretHash` exists for that purpose.
- Redirect URI validation is exact and mirrors production startup validation.
- Public clients always require PKCE.
- Existing client creation fails unless `--if-not-exists` or `--update` is set.

### `admin keys rotate`

Purpose: activate a new signing key while keeping old verification keys until their safe retirement window passes.

Existing implementation anchor:

- `internal/domain/types.go:133-142` stores key lifecycle metadata.
- `internal/storage/interfaces.go:67-72` defines active, verification, create, activate, and retire key operations.

Example:

```bash
tinyidp admin keys rotate --new-kid 2026-10-01-rsa-1 --algorithm RS256 --overlap 90d
```

Expected behavior:

1. Generate new key.
2. Persist it inactive.
3. Activate it.
4. Retire the previous active key but keep it available for verification.
5. Print the key IDs and retirement dates, not private key material.

## Output and Rendering

Every command should support:

- default table/text output for humans;
- `--output json` for scripts;
- `--output yaml` where useful;
- `--quiet` for commands that only need exit status;
- `--redacted=false` only for carefully gated commands that intentionally print generated one-time secrets.

Secret handling rule: generated client secrets and bootstrap passwords may be printed once. Stored secrets are never retrievable because only hashes should be stored.

## Safety Rules

- Destructive commands require `--yes` when stdin is not a TTY.
- Commands that revoke or delete state must support `--dry-run`.
- Commands must emit audit events when the server-side audit sink is configured.
- Error messages must name resources but not secrets.
- A command must not partially mutate multiple resources unless the mutation is transactional or explicitly documented.
- Backups must be verified before reporting success.

## Files to Change

Primary new files:

- `internal/admin/service.go` — admin service constructor and shared dependencies.
- `internal/admin/client.go` — client lifecycle operations.
- `internal/admin/key.go` — signing-key operations.
- `internal/admin/user.go` — user lifecycle operations; call password service from `TINYIDP-USERS-001`.
- `internal/admin/doctor.go` — preflight report.
- `internal/admin/backup.go` — backup/restore helpers.
- `internal/cmds/admin.go` — `tinyidp admin` parent command.
- `internal/cmds/admin_client.go` — client subcommands.
- `internal/cmds/admin_keys.go` — key subcommands.
- `internal/cmds/admin_user.go` — user subcommands.
- `internal/cmds/admin_migrate.go` — migration subcommand.
- `internal/cmds/admin_doctor.go` — doctor subcommand.

Existing files to integrate:

- `cmd/tinyidp/main.go:51-88` — add the admin command tree beside `serve` and `print-config`.
- `internal/storage/interfaces.go:19-92` — use store interfaces rather than concrete SQLite where possible.
- `internal/store/sqlite/migrations/001_schema.sql:1-20` — connect migrations to admin lifecycle.
- `internal/domain/types.go:14-29` — use client model and validation.
- `pkg/embeddedidp/options.go:42-74` — mirror production preflight validation.

## Implementation Plan

### Phase 1: Admin runtime and service skeleton

1. Add `internal/admin.Service` with dependencies and a fake clock for tests.
2. Add `internal/cmds/admin.go` parent command.
3. Wire the parent into `cmd/tinyidp/main.go`.
4. Add `--config-file`, `--output`, `--dry-run`, and `--yes` conventions.
5. Add tests that the command tree is discoverable.

### Phase 2: Migration and doctor

1. Expose SQLite migration status from the store package.
2. Implement `admin migrate --dry-run` and `admin migrate`.
3. Implement `admin doctor` using config validation plus store checks.
4. Add tests using temporary SQLite databases.

### Phase 3: Client operations

1. Implement create/list/get/disable/enable.
2. Implement client secret hashing and rotation.
3. Add redirect URI and scope validation.
4. Add JSON/table rendering tests.
5. Add idempotency tests for `--if-not-exists`.

### Phase 4: Key operations

1. Wrap existing key rotation helpers.
2. Add list/generate/import/rotate/retire.
3. Add JWKS preview command that prints public keys only.
4. Add overlap/retirement validation tests.

### Phase 5: User operations

1. Add user commands that call the real password credential service from `TINYIDP-USERS-001`.
2. Add user disable/enable/lock/unlock.
3. Add password reset command that never prints existing passwords.

### Phase 6: Backup and evidence export

1. Add SQLite backup create/verify.
2. Add sanitized diagnostics export.
3. Add sanitized conformance evidence export that excludes raw tokens, authorization codes, and client secrets.

## Pseudocode: Client Creation

```go
func (s *Service) CreateClient(ctx context.Context, req ClientCreateRequest) (domain.Client, SecretResult, error) {
    now := s.Clock().UTC()
    rawSecret := req.PlainSecret
    if !req.Public && len(rawSecret) == 0 {
        rawSecret = generateRandomSecret(32)
    }

    c := domain.Client{
        ID: req.ID,
        Public: req.Public,
        RedirectURIs: normalizeUnique(req.RedirectURIs),
        PostLogoutRedirectURIs: normalizeUnique(req.PostLogoutRedirectURIs),
        AllowedScopes: normalizeScopes(req.AllowedScopes),
        RequirePKCE: req.RequirePKCE || req.Public,
        AccessTokenTTL: defaultDuration(req.AccessTokenTTL, time.Hour),
        IDTokenTTL: defaultDuration(req.IDTokenTTL, time.Hour),
        RefreshTokenTTL: defaultDuration(req.RefreshTokenTTL, 30*24*time.Hour),
        CreatedAt: now,
        UpdatedAt: now,
    }
    if !req.Public {
        c.SecretHash = s.SecretHasher.Hash(rawSecret)
    }
    if err := c.Validate(domain.ProductionMode); err != nil { return domain.Client{}, SecretResult{}, err }
    if err := s.Store.PutClient(ctx, c); err != nil { return domain.Client{}, SecretResult{}, err }
    return c, SecretResult{Generated: req.PlainSecret == nil, Plain: rawSecret}, nil
}
```

## Decision Records

### Decision 1: Add an admin service layer between commands and stores

Status: proposed.

Decision: command handlers should call `internal/admin.Service` instead of mutating stores directly.

Rationale: this keeps validation and side effects testable without Cobra, prevents duplicated rules across commands, and makes future HTTP/admin API reuse possible.

### Decision 2: Use the same binary for serving and administration

Status: proposed.

Decision: add `tinyidp admin ...` under the existing binary.

Rationale: operators already install one binary. The admin CLI needs exactly the same config and store dependencies as the server. A separate binary would increase packaging and version skew risk.

### Decision 3: Prefer one-time secret generation over retrievable secrets

Status: proposed.

Decision: generated client secrets are displayed once at creation/rotation time, then only hashes are stored.

Rationale: `domain.Client` already models `SecretHash`; storing raw client secrets would be a regression from the production design.

## Testing Strategy

Unit tests:

- command flag parsing for each subcommand;
- service validation errors for invalid clients;
- secret redaction in every renderer;
- dry-run produces no store changes;
- destructive commands require confirmation in non-interactive mode;
- generated secrets are returned once and not persisted raw.

Integration tests:

- create temp SQLite DB with `admin init`;
- run `admin migrate` twice and verify idempotency;
- create a client and start strict provider using that DB;
- rotate a signing key and verify old tokens can still validate;
- create a user and authenticate through strict login after the password ticket lands;
- run `go test ./... -count=1` and `scripts/run-conformance.sh`.

Manual smoke examples:

```bash
tinyidp admin init --config-file ./examples/config/dev-sqlite.yaml --generate-signing-key
tinyidp admin client create --config-file ./examples/config/dev-sqlite.yaml --id web-app --generate-secret --redirect-uri http://localhost:3000/callback --output json
tinyidp admin keys list --config-file ./examples/config/dev-sqlite.yaml
tinyidp admin doctor --config-file ./examples/config/dev-sqlite.yaml
```

## Intern Implementation Notes

Start with `doctor` and `migrate` before client/user mutation commands. Those commands teach you how config opens the store and how errors should be rendered, but they are less risky because they either read state or apply known migrations.

When adding a new subcommand, write the service method first, then test it with memory and SQLite stores, then add the CLI adapter. If a validation rule appears in both the command and the service, move it into the service.

## References

- `cmd/tinyidp/main.go:51-88`
- `internal/cmds/serve.go:52-79`
- `internal/storage/interfaces.go:19-92`
- `internal/store/sqlite/migrations/001_schema.sql:1-20`
- `internal/domain/types.go:14-29`
- `pkg/embeddedidp/options.go:42-74`
