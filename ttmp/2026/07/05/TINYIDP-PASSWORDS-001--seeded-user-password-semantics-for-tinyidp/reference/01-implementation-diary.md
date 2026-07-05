---
Title: Implementation Diary
Ticket: TINYIDP-PASSWORDS-001
Status: active
Topics:
    - oidc
    - testing
    - identity
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/05/TINYIDP-PASSWORDS-001--seeded-user-password-semantics-for-tinyidp/design-doc/01-seeded-user-password-semantics-guide.md
      Note: Primary design guide created in Step 1
ExternalSources: []
Summary: Chronological diary for the seeded-user password semantics ticket.
LastUpdated: 2026-07-05T17:45:00-04:00
WhatFor: Read before implementing optional password checks for seeded users.
WhenToUse: Use when resuming TINYIDP-PASSWORDS-001 or reviewing password behavior decisions.
---

# Diary

## Goal

Capture the design and delivery work for optional seeded-user password semantics in tinyidp.

## Step 1: Create the ticket and design optional seeded-user passwords

This step created a dedicated ticket for password-shaped behavior in tinyidp seeded users. The design explicitly keeps tinyidp as a test IdP, not a production account system: passwords are optional fixture selectors that make demos and negative login tests clearer.

The primary output is an intern-facing implementation guide covering the current seeded-user model, current authorize POST behavior, the proposed schema, validation semantics, tests, and docs updates. No code changed in this step.

### Prompt Context

**User prompt (verbatim):** (same as TINYIDP-CONFIG-001 Step 1)

**Assistant interpretation:** Create a separate ticket for the third previously listed tinyidp usability item: seeded-user password support.

**Inferred user intent:** The user wants password semantics documented as a separate, carefully scoped feature rather than mixed into config portability or claim preset work.

**Commit (code):** N/A — documentation-only ticket creation step.

### What I did

- Created `TINYIDP-PASSWORDS-001` under the tinyidp repo `ttmp` root.
- Added design doc `design-doc/01-seeded-user-password-semantics-guide.md`.
- Replaced the default task list with phased implementation tasks.
- Wrote this diary entry.
- Used existing source evidence from:
  - `internal/scenario/seeded_users.go`,
  - `internal/server/authorize.go`,
  - `internal/server/static/login.html`.

### Why

- Current tinyidp ignores passwords, which is fine for generic OIDC testing but awkward for Keycloak-style tutorials that document Alice/Bob passwords.
- Optional password checks let tests verify wrong-password behavior without changing the default scenario-selector workflow.

### What worked

- The existing architecture has a natural insertion point: seeded users convert into scenarios, and authorize POST already has the login and password form available.

### What didn't work

- No failures occurred in this step.

### What I learned

- The safest first design is seeded-user-only password validation. Built-in scenarios can remain permissive.
- Plain-text fixture passwords are clearer than hashes for this local test tool, as long as docs warn against real credentials.

### What was tricky to build

- The subtle design issue is not implementation complexity but messaging: the feature should improve tutorial realism without suggesting security guarantees.

### What warrants a second pair of eyes

- Review whether wrong passwords should return plain `401` or re-render the login form with an error message.
- Review whether scenario auth-error hooks should happen before or after password validation. The guide proposes password validation first.

### What should be done in the future

- Upload the bundle to reMarkable.
- Implement schema and server tests before changing xgoja smoke helpers.

### Code review instructions

- Start with `design-doc/01-seeded-user-password-semantics-guide.md`.
- Focus on the semantics section and decision records before implementation.

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/05/TINYIDP-PASSWORDS-001--seeded-user-password-semantics-for-tinyidp
```
