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
RelatedFiles: []
ExternalSources: []
Summary: "Chronological notes for the admin CLI design ticket."
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: "Use this to resume or review the admin CLI design work."
WhenToUse: "Read before implementing TINYIDP-ADMIN-001 or updating its design guide."
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
