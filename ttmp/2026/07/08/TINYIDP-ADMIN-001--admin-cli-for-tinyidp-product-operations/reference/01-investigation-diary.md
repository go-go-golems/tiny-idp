---
Title: Investigation diary
Ticket: TINYIDP-ADMIN-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://docs/admin-cli.md
      Note: Operator documentation for admin commands (commit 167b444)
    - Path: repo://internal/admin/backup.go
      Note: SQLite backup helpers (commit 6b974d8)
    - Path: repo://internal/admin/clients.go
      Note: Admin client lifecycle service operations (commit 6b974d8)
    - Path: repo://internal/admin/doctor.go
      Note: Admin preflight checks (commit 6b974d8)
    - Path: repo://internal/admin/keys.go
      Note: Admin signing-key generation/rotation/redaction operations (commit 6b974d8)
    - Path: repo://internal/admin/users.go
      Note: Existing admin user/password service integrated into this ticket (commit 3b3a155)
    - Path: repo://internal/cmds/admin.go
      Note: Admin command parent and user command helpers
    - Path: repo://internal/cmds/admin_backup.go
      Note: Backup subcommands (commit 6b974d8)
    - Path: repo://internal/cmds/admin_client.go
      Note: Client subcommands (commit 6b974d8)
    - Path: repo://internal/cmds/admin_export.go
      Note: Sanitized diagnostics export (commit 167b444)
    - Path: repo://internal/cmds/admin_keys.go
      Note: Key subcommands (commit 6b974d8)
    - Path: repo://internal/cmds/admin_ops.go
      Note: Init/migrate/doctor subcommands (commit 6b974d8)
ExternalSources: []
Summary: Chronological notes for the admin CLI design ticket.
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: Use this to resume or review the admin CLI design work.
WhenToUse: Read before implementing TINYIDP-ADMIN-001 or updating its design guide.
---


# Diary

## Goal

Capture the investigation and design-writing work for adding a safe operational `tinyidp admin` CLI.

## Step 1: Create the ticket and write the admin CLI guide

I created a dedicated docmgr ticket for the admin CLI and wrote a design guide that turns product operations into a command tree and service layer. The guide focuses on initialization, migrations, doctor checks, client lifecycle, user lifecycle, key rotation, backups, and sanitized exports.

This step produced an implementation plan only. It did not add commands to the binary yet.

### Prompt Context

**User prompt (verbatim):** "Create three docmgr tickets for:
1. structured configuration,
2. admin CLI,
3. real user/password storage.

For each ticket, write a detailed intern-oriented analysis/design/implementation guide with prose, bullets, pseudocode, diagrams/API/file references, store it in the ticket, and upload it to reMarkable."

**Assistant interpretation:** Create a separate docmgr ticket and long-form design guide for a production admin CLI, grounded in the current command wiring and storage/domain capabilities.

**Inferred user intent:** Convert the productization roadmap into concrete implementation tickets that an intern can pick up and execute safely.

### What I did
- Created ticket `TINYIDP-ADMIN-001`.
- Added `design-doc/01-admin-cli-design-and-implementation-guide.md`.
- Added this investigation diary.
- Inspected root command wiring, storage interfaces, SQLite schema, and domain client/key support.

### Why
- A production IdP needs operational commands for initialization, migrations, clients, users, keys, backups, and diagnostics.
- Direct SQLite edits or ad hoc scripts would be unsafe for client secrets, signing keys, and user lifecycle operations.

### What worked
- Existing store interfaces already expose the resources the admin service needs.
- Existing key lifecycle storage and domain client fields make the first admin commands feasible.

### What didn't work
- N/A; this was documentation and ticket setup only.

### What I learned
- `cmd/tinyidp/main.go` currently wires only `serve` and `print-config`.
- The admin CLI should be implemented through an internal service layer so validation is not trapped in Cobra handlers.

### What was tricky to build
- The design has to coordinate with two sibling tickets: structured config supplies the admin runtime, and real password storage supplies user credential operations. The guide keeps those boundaries explicit.

### What warrants a second pair of eyes
- Review the proposed command tree for MVP scope; it may be sensible to cut backup/export commands into a later phase if implementation needs to be smaller.

### What should be done in the future
- Implement `admin doctor` and `admin migrate` first.
- Then add client, key, user, backup, and export subcommands in phases.

### Code review instructions
- Start with `design-doc/01-admin-cli-design-and-implementation-guide.md`.
- Cross-check references to `cmd/tinyidp/main.go`, `internal/storage/interfaces.go`, `internal/domain/types.go`, and SQLite migrations.
- Validate docs with `docmgr doctor --ticket TINYIDP-ADMIN-001 --stale-after 30`.

### Technical details
- No code changed in this step.
- Proposed package boundary: `internal/admin` service plus `internal/cmds/admin*.go` CLI adapters.

## Step 2: Build Admin Runtime, Client, Key, Migration, Doctor, Backup, and Diagnostics Commands

I implemented the operational admin CLI surface around the existing SQLite store and the user/password work that had just landed. The command tree now has `init`, `migrate`, `doctor`, `client`, `keys`, `user`, `backup`, and `export diagnostics` branches, with the mutation logic kept in `internal/admin.Service` where it can be tested without Cobra.

This step turns the admin ticket from a design document into a usable product-operations workflow. Operators can initialize a database, create clients, generate and rotate signing keys, create users and passwords, run preflight checks, create and verify backups, and export sanitized diagnostics without manually editing SQLite rows.

### Prompt Context

**User prompt (verbatim):** "do TINYIDP-ADMIN-001"

**Assistant interpretation:** Implement the active admin CLI ticket, commit at sensible points, keep docmgr ticket state current, and validate the result.

**Inferred user intent:** Complete the productization admin surface so tinyidp has operational commands for clients, keys, users, migrations, backups, and diagnostics.

**Commit (code):** 6b974d8 — "Add admin client key migration and backup commands"

**Commit (docs/code):** 167b444 — "Document admin CLI diagnostics workflow"

**Related prior code commit:** 3b3a155 — "Add user password admin commands"

### What I did
- Extended the existing `tinyidp admin` command parent with:
  - `init`
  - `migrate`
  - `doctor`
  - `client create/list/get/disable/enable/rotate-secret`
  - `keys generate/list/rotate/retire`
  - `backup create/verify`
  - `export diagnostics`
- Added admin service methods for client lifecycle operations, including bcrypt client-secret hashing and one-time generated secret output.
- Added admin service methods for signing key generation, rotation, listing, retirement, and PEM redaction.
- Added `Doctor` checks for client production validation, active signing key presence, and verification key state.
- Added SQLite migration listing so `admin migrate --dry-run` can show embedded migrations.
- Added backup create/verify helpers for SQLite database files.
- Added sanitized diagnostics export that redacts client secret hashes and private key PEM material.
- Added `docs/admin-cli.md` with command examples and safety notes.
- Added tests for client lifecycle, key generation/rotation, doctor behavior, and backup verification.
- Ran validation:
  - `go test ./internal/admin ./internal/cmds ./internal/store/sqlite ./cmd/tinyidp`
  - `go test ./...`
  - `scripts/run-conformance.sh`

### Why
- A production-like IdP needs repeatable operational commands around clients, keys, users, migrations, and backups.
- Command handlers should remain adapters; the service layer should own domain validation and mutations.
- Diagnostics and command output must avoid leaking client secrets or private signing-key material.

### What worked
- The `internal/admin.Service` introduced by the user/password implementation was a good place to add client/key/doctor operations.
- Existing store interfaces already exposed enough client and key operations for a useful MVP.
- Existing key rotation helper `keys.RotateRSA` could be reused directly.
- Full conformance checks remained green after adding the admin CLI surface.

### What didn't work
- There was no full structured production config yet, so the admin CLI cannot load product config files. I kept the explicit `--db` runtime for now and documented that `TINYIDP-PROD-CONFIG-001` should later replace or supplement it.
- SQLite migrations did not expose a migration list initially. I added `sqlite.MigrationNames()` so dry-run output can report embedded migration filenames.

### What I learned
- The existing `storage.Store` contract is enough for an MVP admin CLI, but list-style user management is limited because there is no `ListUsers` method yet.
- Backup and diagnostics are useful even in simple form, as long as private key PEM and secret hashes are redacted from output.

### What was tricky to build
- Client secret handling needed two paths: confidential clients require a bcrypt hash in `domain.Client.SecretHash`, while generated secrets must be printed exactly once for operators. I made command output include a one-time generated secret but redacted stored hashes everywhere else.
- Key management needed to avoid leaking `PrivateKeyPEM`. Service methods return normal domain keys for storage correctness, but command output and diagnostics pass through `RedactSigningKey` / `RedactSigningKeys`.
- `doctor` intentionally uses production validation even though the admin runtime can be pointed at a local database. This makes preflight checks stricter and catches unsafe clients before a production server starts.

### What warrants a second pair of eyes
- Review whether the explicit `--db` runtime should remain after structured config exists.
- Review whether backup creation should use SQLite online backup/VACUUM INTO instead of file copy for live high-write deployments.
- Review command output schemas before scripts depend on them as stable APIs.
- Review whether `ListUsers` should be added to the storage interface for a fuller `admin user list` command.

### What should be done in the future
- Replace/supplement `--db` with config-backed admin runtime from `TINYIDP-PROD-CONFIG-001`.
- Add transaction support for multi-row operations such as user+credential creation.
- Add session/grant revocation commands for disabled users and password resets.
- Add richer migration metadata rather than only idempotent embedded SQL filenames.

### Code review instructions
- Start with `internal/admin/clients.go`, `internal/admin/keys.go`, `internal/admin/doctor.go`, and `internal/admin/backup.go`.
- Then review CLI adapters in `internal/cmds/admin_*.go`.
- Confirm output redaction in `redactClient`, `admin.RedactSigningKey`, and `admin export diagnostics`.
- Validate with `go test ./...` and `scripts/run-conformance.sh`.

### Technical details
- Admin runtime: `tinyidp admin --db <sqlite-path> ...`.
- Client secret hashing: bcrypt into `domain.Client.SecretHash`.
- Key output redaction: private key PEM omitted from command/diagnostic output.
- Backup verification: opens the backup through the SQLite store and performs basic store checks.
