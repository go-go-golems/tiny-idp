---
Title: Investigation diary
Ticket: TINYIDP-PR3-REVIEW-002
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/admin/admin_ops_test.go
      Note: Regression coverage for same-file backup rejection
    - Path: repo://internal/admin/users_test.go
      Note: Regression coverage for duplicate explicit user IDs
    - Path: repo://internal/fositeadapter/consent_test.go
      Note: Regression coverage for prompt none consent_required
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: Regression coverage for empty strict scope allowlists
ExternalSources:
    - https://github.com/go-go-golems/tiny-idp/pull/3
Summary: Chronological diary for the second PR 3 review-fix pass.
LastUpdated: 2026-07-09T12:00:00-04:00
WhatFor: Use this to review or resume the second PR 3 review-fix pass.
WhenToUse: Read before changing backup safety, strict scope handling, prompt=none consent behavior, or admin user ID uniqueness.
---


# Diary

## Goal

Capture the second pass of PR 3 review fixes: backup safety, strict scope allowlists, silent consent errors, and duplicate user ID rejection.

## Step 1: Address second-round PR 3 review findings

I fetched the latest PR 3 review comments and found four new actionable findings. The fixes all make product/admin behavior fail closed: backup refuses to overwrite the source database, strict clients no longer gain default scopes from an empty allowlist, silent authorization returns the proper OAuth consent error, and admin user creation rejects duplicate explicit IDs.

The implementation was small but correctness-critical. Each fix prevents an operator or client from observing surprising behavior after moving from mock-only development semantics to strict, durable, admin-managed state.

### Prompt Context

**User prompt (verbatim):** "Repeat https://github.com/go-go-golems/tiny-idp/pull/3"

**Assistant interpretation:** Repeat the previous PR review-fix workflow for PR 3: inspect new review comments, fix them, add regression tests, validate, document, commit, push, and update the PR.

**Inferred user intent:** Keep iterating on PR 3 automated review feedback until the productization branch is clean and reviewable.

**Commit (code):** e8253b1 — "Fix strict review follow-up findings"

### What I did
- Queried PR 3 state and inline review comments with `gh pr view` and `gh api repos/go-go-golems/tiny-idp/pulls/3/comments --paginate`.
- Fixed `internal/admin/backup.go` so backup rejects same-file source/destination paths before opening the output with `O_TRUNC`.
- Fixed strict Fosite scope handling by removing permissive default scopes from SQL and memory strict client construction.
- Fixed `prompt=none` consent handling so an existing session with missing consent receives `consent_required` instead of an HTML consent form.
- Fixed admin user creation so a caller-supplied duplicate `ID` returns `storage.ErrDuplicate` before writing user or credential records.
- Added regression tests in admin and Fosite adapter packages.
- Ran validation:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`
- Created ticket `TINYIDP-PR3-REVIEW-002` and wrote the design/report document.

### Why
- A backup command must never truncate the live source database.
- Strict production stores should preserve the domain rule that an empty scope allowlist grants no scopes.
- `prompt=none` must not show UI; it must return OAuth errors for interaction requirements.
- User IDs are primary identifiers and must be unique independently from login names.

### What worked
- The focused tests reproduced each review concern cleanly.
- Removing strict default scopes did not break conformance because all strict tests and seeded clients that need scopes now declare them explicitly.
- Full lint, logcopter, unit, and local conformance validation passed.

### What didn't work
- N/A. No failed validation command occurred during this pass.

### What I learned
- Productization review is surfacing places where old mock/dev permissiveness leaked into strict paths.
- Backup commands need path identity checks before any destructive open mode, not after opening both files.
- `prompt=none` requires checking every interaction branch, not only the missing-login branch.

### What was tricky to build
- The backup fix needed to handle both lexical equality (`db.sqlite` versus `./db.sqlite`) and filesystem identity (symlinks/hard links). I used absolute path comparison first, then `os.Stat` and `os.SameFile` when the destination exists.
- Scope handling needed care because permissive defaults are useful in mock/dev client setup but wrong in strict store adapters. The fix removes defaults only from strict Fosite client construction.
- The consent test needed an authenticated browser session with partial consent. It first grants `openid`, then sends `prompt=none` for `openid email` and expects `consent_required`.

### What warrants a second pair of eyes
- Confirm that strict in-memory clients should also preserve empty scope allowlists, not only SQLite-backed clients.
- Confirm that `fosite.ErrConsentRequired` is the preferred error for missing consent with `prompt=none` in all strict silent-auth branches.
- Confirm whether backup should also reject destinations inside a live SQLite WAL/shm file set in a future hardening pass.

### What should be done in the future
- Consider documenting the empty-scope production behavior in the admin client command documentation.
- Consider adding backup path checks for `.wal`/`.shm` companion files if the backup implementation grows beyond this simple copy helper.

### Code review instructions
- Start with `internal/admin/backup.go:sameFile` and `CreateSQLiteBackup`.
- Review `internal/fositeadapter/sqlstore.go` and `internal/fositeadapter/provider.go` for empty scope handling.
- Review `internal/fositeadapter/provider.go:authorize` for the `prompt=none` consent branch.
- Review `internal/admin/users.go:CreateUser` for duplicate ID detection.
- Validate with:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`

### Technical details
- Focused tests:
  - `TestServiceKeysDoctorAndBackup`
  - `TestFositeSQLiteClientWithEmptyScopesRejectsRequestedScope`
  - `TestPromptNoneReturnsConsentRequiredWhenNewScopesNeedConsent`
  - `TestServiceCreateUserRejectsDuplicateExplicitID`
- Validation commands passed on 2026-07-09:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`
