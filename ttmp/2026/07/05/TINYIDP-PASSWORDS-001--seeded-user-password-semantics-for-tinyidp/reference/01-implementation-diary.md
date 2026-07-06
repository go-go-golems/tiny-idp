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
    - Path: internal/scenario/scenario.go
      Note: Scenario password metadata
    - Path: internal/scenario/seeded_users.go
      Note: Seeded-user password schema and conversion
    - Path: internal/scenario/seeded_users_test.go
      Note: Password schema/load tests
    - Path: internal/server/authorize.go
      Note: Authorize POST password validation
    - Path: internal/server/server_test.go
      Note: Server-flow password validation tests
    - Path: ttmp/2026/07/05/TINYIDP-PASSWORDS-001--seeded-user-password-semantics-for-tinyidp/design-doc/01-seeded-user-password-semantics-guide.md
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

## Step 2: Resume password ticket with detailed implementation checklist

This step resumed `TINYIDP-PASSWORDS-001` after completing the generic claim preset ticket. I replaced the coarse task list with a precise phase-by-phase checklist so implementation progress can be tracked at the level of schema fields, validation behavior, tests, docs, and final repository validation.

No code changed in this step. The next step will implement the seeded-user and scenario schema changes with focused tests before touching authorize POST behavior.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the step-by-step tinyidp follow-up work, using the existing docmgr tickets, detailed task tracking, diary updates, validation, and focused commits.

**Inferred user intent:** The user wants the next ticket advanced incrementally without losing the documentation and review discipline established in the previous ticket.

**Commit (code):** pending — task/diary baseline only.

### What I did

- Chose `TINYIDP-PASSWORDS-001` as the next ticket after finishing generic claim presets.
- Replaced the coarse password task list with detailed phases:
  - ticket baseline and tracking,
  - seeded-user schema and conversion,
  - scenario tests,
  - authorize POST validation,
  - server-flow tests,
  - login UI/docs,
  - final validation/bookkeeping.
- Added this diary entry before code changes.

### Why

- The user asked for precise tracking in the docmgr ticket.
- Password semantics touch schema, request handling, UI copy, docs, and tests; splitting them prevents a hard-to-review batch.

### What worked

- Existing ticket state was clean and the previous design bundle had already passed `docmgr doctor`.

### What didn't work

- No failures occurred in this step.

### What I learned

- The current password ticket still had a coarse checklist. It needed the same task granularity as the generic claim preset ticket before implementation.

### What was tricky to build

- The main scoping choice is to keep passwords optional and seeded-user driven. Built-in and fallback users should continue to be permissive unless a configured seeded user supplies a password.

### What warrants a second pair of eyes

- Review the final validation behavior: wrong and missing configured passwords should both return the same generic `invalid login or password` message.

### What should be done in the future

- Implement Phase 1 and Phase 2 in a focused code commit.
- Then implement authorize POST validation and server-flow tests.

### Code review instructions

- Start with `tasks.md` for the execution checklist.
- Then inspect the next code commit for `internal/scenario/seeded_users.go` and `internal/scenario/seeded_users_test.go` only.

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/05/TINYIDP-PASSWORDS-001--seeded-user-password-semantics-for-tinyidp
```

## Step 3: Add password metadata to seeded users and scenarios

This step implemented the data-model part of optional password semantics. Seeded users can now carry a `password` fixture value, and conversion into a scenario preserves that password as optional scenario metadata.

No authorize behavior changed in this step. A seeded user with a configured password is now represented correctly, but login validation will be added in the next implementation slice.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue the password implementation in small reviewable slices, starting with schema and conversion tests.

**Inferred user intent:** The user wants incremental implementation with commits at meaningful boundaries rather than one large password-feature patch.

**Commit (code):** pending — schema/conversion slice.

### What I did

- Added `Password string` to `scenario.SeededUser` with JSON/YAML tags.
- Added `Password string` to `scenario.Scenario` as optional test-fixture metadata.
- Trimmed configured seeded-user passwords during conversion.
- Preserved empty password as the marker for "no password required".
- Added direct seeded-user conversion tests.
- Added YAML and JSON users-file password load tests.

### Why

- Authorize POST should validate against scenario metadata, not re-read users-file state or maintain a parallel map.
- Empty passwords must keep the current permissive behavior for builtins, fallback users, and seeded users that do not opt in.

### What worked

- `go test ./internal/scenario -count=1` passed.

### What didn't work

- No failures occurred in this step.

### What I learned

- The existing `SeededUsersToScenarios` conversion is the right boundary for password metadata, just like it is for deterministic identity and claims.

### What was tricky to build

- Password values should be trimmed at configuration-conversion time so accidental whitespace in YAML does not become part of the fixture password.
- Empty string remains the only sentinel for no password requirement.

### What warrants a second pair of eyes

- Review whether trimming passwords is desired. It is ergonomic for fixtures, but it means leading/trailing spaces cannot be intentional test passwords.

### What should be done in the future

- Add authorize POST validation and server-flow tests.
- Update login UI copy and docs after behavior exists.

### Code review instructions

- Start with `internal/scenario/seeded_users.go` and `internal/scenario/scenario.go`.
- Review `internal/scenario/seeded_users_test.go` for direct, YAML, and JSON load coverage.
- Validate with `go test ./internal/scenario -count=1`.

### Technical details

Validation command run:

```text
go test ./internal/scenario -count=1
ok  	github.com/manuel/tinyidp/internal/scenario	0.003s
```

## Step 4: Validate configured passwords in authorize POST

This step connected the seeded-user password metadata to the browser authorization flow. Authorize POST now checks a scenario's optional password before running auth-error scenarios or issuing a code.

The behavior remains permissive for scenarios with no configured password. Wrong and missing passwords for password-protected seeded users return the same generic `invalid login or password` error and do not create sessions or authorization codes.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue from schema metadata into request-time password validation with focused server tests.

**Inferred user intent:** The user wants test-backed behavior changes with exact validation output and task bookkeeping.

**Commit (code):** pending — authorize validation slice.

### What I did

- Added `passwordAccepted` helper in `internal/server/authorize.go`.
- Read submitted password from the authorize POST form.
- Rejected wrong or missing configured passwords with `401 Unauthorized` and generic error text.
- Kept no-password scenarios permissive.
- Added server-flow tests for:
  - correct configured password success,
  - wrong password failure,
  - missing password failure,
  - no session/code creation for failed attempts,
  - unprotected seeded user permissive behavior,
  - builtin user permissive behavior.

### Why

- Password validation belongs before auth-error scenario redirects and before session/code creation.
- Using generic error text avoids teaching user-enumeration-style behavior even in a local mock.

### What worked

- `go test ./internal/server -count=1` passed.

### What didn't work

- No failures occurred in this step.

### What I learned

- Failed authorize POST attempts can be verified by checking the in-memory `sessions` and `codes` maps directly in server tests.
- Existing test helpers could be extended with a no-redirect POST helper to inspect non-302 responses cleanly.

### What was tricky to build

- The state-count assertion must account for the successful correct-password login that intentionally creates one session and one code before the wrong/missing password checks.

### What warrants a second pair of eyes

- Review whether `401 Unauthorized` is preferable to re-rendering the login form with an error message.
- Review whether password validation should trim submitted passwords. Current behavior requires exact submitted password and only trims configured fixture passwords.

### What should be done in the future

- Update login form copy and public docs.
- Run full repository validation after docs/examples are updated.

### Code review instructions

- Start with the POST branch in `internal/server/authorize.go`.
- Then review `TestSeededUserPasswordValidation` and `postAuthorizeNoRedirect` in `internal/server/server_test.go`.
- Validate with `go test ./internal/server -count=1`.

### Technical details

Validation command run:

```text
go test ./internal/server -count=1
ok  	github.com/manuel/tinyidp/internal/server	9.577s
```

## Step 5: Document optional fixture passwords and run full validation

This step completed the user-facing portion of optional password semantics. The login page no longer says passwords are always ignored, and the README/reference docs now explain that passwords are optional seeded-user fixture values.

The full repository test suite and tinyidp build passed after the UI/docs changes, so the password ticket is implementation-complete.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Finish the password ticket by updating UI copy, public docs, examples, validation, and ticket bookkeeping.

**Inferred user intent:** The user wants the behavior change to be understandable to future users and reviewers, not only implemented in code.

**Commit (code):** pending — docs/UI/final validation slice.

### What I did

- Updated `internal/server/static/login.html`:
  - removed unconditional "password is ignored" copy,
  - changed password placeholder to `optional fixture password`.
- Updated `README.md` seeded-user examples and semantics.
- Updated `cmd/tinyidp/doc/pages/reference.md` seeded-user examples and semantics.
- Added fixture passwords to `examples/users/generic-claims-users.yaml`.
- Ran full repository validation.

### Why

- Once password-protected seeded users exist, the old login-page copy was misleading.
- Docs must make clear that these passwords are local test fixtures, not a production account/security system.

### What worked

- Full tests passed.
- The tinyidp binary built successfully.

### What didn't work

- No command failures occurred in this step.

### What I learned

- The example users file created for generic claims is also a good place to demonstrate optional fixture passwords without adding a separate example.

### What was tricky to build

- The docs need to balance two facts: passwords can be enforced for seeded users, but tinyidp remains non-production and passwords are plain local fixtures.

### What warrants a second pair of eyes

- Review whether examples should include passwords by default, or whether that makes the default path feel less permissive than tinyidp really is.

### What should be done in the future

- If xgoja smoke helpers adopt password-protected users, update them to submit the configured fixture passwords.
- Consider adding config examples under `TINYIDP-CONFIG-001` that reference `examples/users/generic-claims-users.yaml`.

### Code review instructions

- Review the seeded-user docs in `README.md` and `cmd/tinyidp/doc/pages/reference.md`.
- Review `internal/server/static/login.html` for clear local-test wording.
- Validate with the full commands below.

### Technical details

Validation commands run:

```text
GOWORK=off go test ./... -count=1
?   	github.com/manuel/tinyidp/cmd/tinyidp	[no test files]
?   	github.com/manuel/tinyidp/cmd/tinyidp/doc	[no test files]
ok  	github.com/manuel/tinyidp/internal/client	0.008s
ok  	github.com/manuel/tinyidp/internal/cmds	0.019s
ok  	github.com/manuel/tinyidp/internal/scenario	0.006s
ok  	github.com/manuel/tinyidp/internal/sections/oidc	0.007s
ok  	github.com/manuel/tinyidp/internal/server	9.656s
ok  	github.com/manuel/tinyidp/internal/user	0.005s

GOWORK=off go build ./cmd/tinyidp
```
