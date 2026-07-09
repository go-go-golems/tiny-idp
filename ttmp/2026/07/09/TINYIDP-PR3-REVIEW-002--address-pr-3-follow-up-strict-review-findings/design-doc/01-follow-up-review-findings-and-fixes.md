---
Title: Follow-up Review Findings and Fixes
Ticket: TINYIDP-PR3-REVIEW-002
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/admin/backup.go
      Note: Rejects same-file SQLite backup destinations before truncation
    - Path: repo://internal/admin/users.go
      Note: Rejects duplicate explicit user IDs before writes
    - Path: repo://internal/fositeadapter/provider.go
      Note: Preserves empty strict scopes and returns consent_required for prompt none
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Preserves empty scope allowlists in SQL-backed strict clients
ExternalSources:
    - https://github.com/go-go-golems/tiny-idp/pull/3
Summary: 'Textbook-style report for the second PR 3 review pass: backup safety, empty scope allowlists, prompt=none consent errors, and duplicate user IDs.'
LastUpdated: 2026-07-09T12:00:00-04:00
WhatFor: Use this report to understand the second round of PR 3 review fixes and the invariants they protect.
WhenToUse: Read when reviewing admin backup behavior, strict Fosite client scope handling, silent authorization, or admin user creation uniqueness.
---


# Follow-up Review Findings and Fixes

## 1. Purpose

The second review pass found four issues that all come from the same productization theme: once `tiny-idp` has production-shaped storage and admin operations, defensive checks must move from “the happy path usually does the right thing” to “dangerous states are impossible to enter.”

This document explains each fix as an invariant, then shows how the code and tests enforce it.

## 2. Findings at a Glance

| Finding | Invariant | Fixed in |
| --- | --- | --- |
| Backup could target the source DB path | A backup command must never truncate the live database it is trying to copy | `internal/admin/backup.go` |
| Empty strict scope allowlists became permissive defaults | In strict stores, an empty allowlist means no scopes are allowed | `internal/fositeadapter/sqlstore.go`, `provider.go` |
| `prompt=none` with missing consent rendered HTML | Silent authorization must return an OAuth error redirect, not interaction UI | `internal/fositeadapter/provider.go` |
| Duplicate explicit user IDs overwrote existing records | User IDs are durable primary identifiers and must be unique across logins | `internal/admin/users.go` |

## 3. Backup Safety: Refuse Same-File Destinations Before Truncation

The backup command opened the source database for reading, then opened the destination with `O_TRUNC`. If `--out` pointed to the same file as `--db`, opening the destination truncated the live SQLite database before `io.Copy` had anything useful to copy.

The fix is to compare the source and destination before opening the output file:

```go
same, err := sameFile(source, dest)
if err != nil {
    return BackupResult{}, err
}
if same {
    return BackupResult{}, fmt.Errorf("backup destination must differ from source database")
}
```

The helper handles both lexical equality and filesystem identity:

- `filepath.Abs` catches `db.sqlite` versus `./db.sqlite`.
- `os.Stat` + `os.SameFile` catches hard links and symlinks to the same inode.
- A non-existent destination is allowed after proving it is not the source path.

The regression test attempts a same-file backup and then checks that the original database size is unchanged and non-zero.

## 4. Strict Scope Allowlists: Empty Means Empty

The domain model deliberately defines an empty client `AllowedScopes` list as “no scopes allowed.” That is a safe production default: an operator or API caller that forgets scopes should not accidentally grant `profile`, `email`, or `offline_access`.

The strict Fosite adapters were still applying a development default when the allowlist was empty:

```go
if len(fc.Scopes) == 0 {
    fc.Scopes = []string{"openid", "profile", "email", "offline_access"}
}
```

That fallback is wrong for strict stores. It turns omission into authority. The fix removes the fallback from both SQL-backed strict clients and in-memory strict client construction. Mock/dev legacy behavior can remain in mock-specific paths; strict Fosite should preserve the domain rule exactly.

The regression test creates a SQLite strict client with no allowed scopes and verifies that requesting `openid` produces `invalid_scope`.

## 5. Silent Authorization: `prompt=none` Cannot Render Consent UI

`prompt=none` means the relying party is asking whether authorization can complete without user interaction. If login is missing, the provider returns `login_required`. If consent is missing for requested scopes, the provider must return `consent_required`. It must not render an HTML consent form, because rendering UI violates the silent-auth contract.

The provider already handled the no-session case. The missing branch was: session exists, but requested scopes still require consent. The fix checks `prompt=none` before rendering the consent form:

```go
if requireConsent {
    if promptHas(ar.GetRequestForm().Get("prompt"), "none") {
        p.oauth2.WriteAuthorizeError(r.Context(), w, ar, fosite.ErrConsentRequired)
        return
    }
    p.renderInteraction(w, ar, false, true)
    return
}
```

The regression test first grants consent for `openid`, then sends a silent request for `openid email`. Because `email` has not been approved, the provider redirects with `error=consent_required`.

## 6. User Creation: Explicit IDs Must Be Globally Unique

The admin user service already rejected duplicate logins. It did not reject a second login with the same explicit `ID`. Both memory and SQLite stores key user records and credentials by user ID, so allowing duplicate IDs could overwrite the original user row or leave stale login aliases.

The fix checks ID uniqueness after resolving the requested/generated ID and before constructing credentials:

```go
if _, err := s.Store.GetUser(ctx, id); err == nil {
    return domain.User{}, storage.ErrDuplicate
} else if !errors.Is(err, storage.ErrNotFound) {
    return domain.User{}, err
}
```

The regression test runs against both memory and SQLite stores. It creates `alice` with `fixed-user-id`, then attempts `alice-alias` with the same ID and expects `storage.ErrDuplicate`. It also verifies that the rejected alias was not written.

## 7. Validation

The following commands passed after the fixes:

```bash
make lint
make logcopter-check
go test ./...
scripts/run-conformance.sh
```

Focused regression coverage:

- `TestServiceKeysDoctorAndBackup` now verifies same-file backup rejection.
- `TestFositeSQLiteClientWithEmptyScopesRejectsRequestedScope` verifies empty strict allowlists remain empty.
- `TestPromptNoneReturnsConsentRequiredWhenNewScopesNeedConsent` verifies silent consent errors.
- `TestServiceCreateUserRejectsDuplicateExplicitID` verifies duplicate IDs are rejected for memory and SQLite stores.

## 8. Review Notes

These fixes intentionally fail closed:

- A backup destination that might be the source is rejected before truncation.
- A client with no scopes gets no scopes.
- A silent request that needs consent gets an OAuth error, not UI.
- A duplicate user ID is a duplicate even when the login is different.

The result is stricter behavior, but stricter in places that protect operator intent and OAuth/OIDC protocol semantics.
