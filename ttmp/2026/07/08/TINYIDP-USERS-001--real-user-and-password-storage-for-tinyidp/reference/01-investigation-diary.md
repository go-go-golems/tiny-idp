---
Title: Investigation diary
Ticket: TINYIDP-USERS-001
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
Summary: "Chronological notes for the real user/password storage design ticket."
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: "Use this to resume or review the password-storage design work."
WhenToUse: "Read before implementing TINYIDP-USERS-001 or updating its design guide."
---

# Diary

## Goal

Capture the investigation and design-writing work for replacing dev-only login behavior with durable user accounts and password credential verification.

## Step 1: Create the ticket and write the user/password guide

I created a dedicated docmgr ticket for real user and password storage and wrote a design guide for credentials, hashing, account state, lockout, strict provider integration, admin integration, and tests. The guide preserves mock seeded-user behavior while making production strict mode require password verification.

This step produced an implementation plan only. It did not change the login handler or storage interfaces yet.

### Prompt Context

**User prompt (verbatim):** "Create three docmgr tickets for:
1. structured configuration,
2. admin CLI,
3. real user/password storage.

For each ticket, write a detailed intern-oriented analysis/design/implementation guide with prose, bullets, pseudocode, diagrams/API/file references, store it in the ticket, and upload it to reMarkable."

**Assistant interpretation:** Create a separate docmgr ticket and long-form design guide for production user/password storage and strict login verification.

**Inferred user intent:** Turn the known user/password product gap into a concrete design that can be implemented safely without breaking mock testing behavior.

### What I did
- Created ticket `TINYIDP-USERS-001`.
- Added `design-doc/01-real-user-and-password-storage-design-and-implementation-guide.md`.
- Added this investigation diary.
- Inspected domain user fields, user store interfaces, SQLite schema, seeded-user password fixtures, scenario semantics, and strict provider login handling.

### Why
- Production strict mode cannot authenticate a user by login name alone.
- Password hashes and credential lifecycle data should not be mixed into OIDC profile/userinfo structures.

### What worked
- Existing `domain.User` already has disabled/locked fields that can participate in account policy.
- Existing strict login form already renders a password field, so the UI boundary exists.

### What didn't work
- N/A; this was documentation and ticket setup only.

### What I learned
- Strict provider login currently calls `GetUserByLogin` and creates a browser session without password verification.
- Seeded-user password behavior is intentionally a mock/test fixture and must remain separate from production authentication.

### What was tricky to build
- The design must avoid breaking conformance and mock scenarios while closing the production authentication gap. The guide resolves this with mode-specific authenticators: scenario/permissive for mock/dev fixtures and password service for production.

### What warrants a second pair of eyes
- Review Argon2id parameter defaults and lockout policy before implementation.
- Review whether account lockout should live in credential tables or a separate security-state table.

### What should be done in the future
- Implement password hashing first, then credential storage, then authentication service, then provider wiring.
- Add admin user commands after the password service exists.

### Code review instructions
- Start with `design-doc/01-real-user-and-password-storage-design-and-implementation-guide.md`.
- Cross-check references to `internal/fositeadapter/provider.go`, `internal/scenario`, `internal/domain/types.go`, and `internal/storage/interfaces.go`.
- Validate docs with `docmgr doctor --ticket TINYIDP-USERS-001 --stale-after 30`.

### Technical details
- No code changed in this step.
- Proposed packages: `internal/passwordhash`, `internal/authn`, credential storage interfaces, SQLite migration `002_password_credentials.sql`.
