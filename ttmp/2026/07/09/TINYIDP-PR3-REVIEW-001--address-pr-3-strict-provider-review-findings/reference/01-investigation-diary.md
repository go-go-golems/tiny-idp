---
Title: Investigation diary
Ticket: TINYIDP-PR3-REVIEW-001
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
    - Path: repo://internal/cmds/serve_test.go
      Note: Regression coverage for seeded strict password enforcement
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: Provider production secret regression coverage
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: Regression coverage for disabled persisted authorization code exchange
    - Path: repo://pkg/embeddedidp/provider_test.go
      Note: Embedded production secret regression coverage
ExternalSources:
    - https://github.com/go-go-golems/tiny-idp/pull/3
Summary: Chronological diary for addressing PR 3 strict provider review findings.
LastUpdated: 2026-07-09T12:00:00-04:00
WhatFor: Use this to review or resume PR 3 review-fix work.
WhenToUse: Read before changing the strict provider fixes for disabled clients, production secrets, seeded passwords, discovery metadata, or max_age behavior.
---


# Diary

## Goal

Capture the investigation and implementation work for addressing PR 3 code review findings and checking for similar strict-provider issues.

## Step 1: Create review ticket and fix strict-provider boundary issues

I created a dedicated review-fix ticket and addressed all five Codex PR review comments. The fixes tighten strict-provider boundaries around disabled clients, production token secrets, seeded password enforcement, discovery metadata, and `max_age=0` session reuse.

The main theme was revalidation. Values restored from storage, defaults inherited from development mode, and metadata exposed to clients all need to be checked at the point where they become security-relevant.

### Prompt Context

**User prompt (verbatim):** "Address code review issues in the PR"

**Assistant interpretation:** Fetch PR 3 review comments, create a ticket as requested by the user, fix the reported issues plus similar issues, document the reasoning, validate, and push the PR branch.

**Inferred user intent:** Resolve automated review findings without treating them as isolated patch requests; use the opportunity to audit adjacent strict-provider behavior and keep docmgr evidence.

**Commit (code):** d2664c8 — "Fix strict provider review findings"

### What I did
- Created ticket `TINYIDP-PR3-REVIEW-001`.
- Pulled PR review comments via `gh api repos/go-go-golems/tiny-idp/pulls/3/comments`.
- Fixed disabled-client handling in the SQL-backed strict Fosite store:
  - direct `GetClient` now rejects disabled clients;
  - restored persisted requesters now re-check the current domain client and reject disabled clients;
  - request-object redirect validation rejects disabled clients;
  - in-memory Fosite client construction skips disabled clients at startup.
- Required production token secrets of at least 32 bytes in both `fositeadapter.NewProvider` and `embeddedidp.Options.Validate`.
- Made strict dev startup create password credentials for seeded users that specify a password.
- Removed unimplemented `end_session_endpoint` from strict production discovery metadata.
- Changed `max_age=0` handling so existing browser sessions do not satisfy fresh-auth requests.
- Added focused regression tests for each finding.
- Wrote `design-doc/01-strict-provider-review-findings-and-fixes.md` in textbook style.
- Ran validation:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`

### Why
- Disabled clients must not remain usable after admin disable.
- Production mode must not share a hard-coded development secret.
- Seeded password fixtures should be meaningful in strict dev mode.
- Discovery metadata must match actual strict adapter routes.
- `max_age=0` is a relying-party request for fresh authentication.

### What worked
- The disabled-client fix became stronger by applying it to the shared persisted-request restore path instead of only one lookup method.
- Existing tests made it easy to add focused coverage around strict authorization, SQLite persistence, and browser session behavior.
- Full local conformance remained green after the stricter behavior.

### What didn't work
- N/A after implementation. The only expected behavior change was that production constructors now reject missing/short token secrets, so tests had to provide explicit production secrets when they were testing other production validations.

### What I learned
- Persisted OAuth requesters must not be treated as self-contained authorization. They are historical protocol state and still need current client-state validation.
- The strict adapter and embedded API are separate construction boundaries; production secret checks belong in both places.
- Discovery fields should be treated as executable contracts, not aspirational documentation.

### What was tricky to build
- The disabled-client issue had a direct path and a persisted-state path. Fixing only `GetClient` would have left authorization-code and refresh-token records carrying a previously active client snapshot. I added `restoreActiveRequester` so every SQL requester load checks the current project client.
- `max_age=0` required preserving tolerance for invalid/negative values while changing zero from unconstrained to fresh-auth required. The helper now treats `<0` as unconstrained but evaluates `0` normally.
- Seeded password enforcement needed to preserve dev passwordless compatibility when no fixture password exists. The fix only creates credentials when `sc.Password != ""`.

### What warrants a second pair of eyes
- Whether returning `fosite.ErrNotFound` is the desired external behavior for all disabled-client persisted-session paths, including refresh-token use after disable.
- Whether the production token secret minimum should remain byte-length based or eventually be modeled as entropy/encoding validation in structured configuration.
- Whether strict RP-initiated logout should be implemented soon now that discovery no longer advertises it.

### What should be done in the future
- Implement strict `/end-session` and reintroduce `end_session_endpoint` only in the same change.
- Extend structured production configuration work so missing token secrets fail early during config materialization with a targeted operator message.
- Consider documenting the dynamic-client-disable semantics for in-memory mode versus SQLite-backed production mode.

### Code review instructions
- Start with `internal/fositeadapter/sqlstore.go` and inspect `GetClient`, `GetAuthorizeCodeSession`, `getRequester`, and `restoreActiveRequester`.
- Review `internal/fositeadapter/provider.go` for production secret validation, disabled-client filtering, and redirect validation.
- Review `internal/cmds/serve.go` for seeded password credential creation.
- Review `internal/fositeadapter/session.go` for the `max_age=0` behavior change.
- Validate with:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`

### Technical details
- Focused tests:
  - `TestFositeSQLiteDisabledClientRejectsPersistedAuthorizationCode`
  - `TestProductionProviderRejectsMissingSecretKey`
  - `TestProductionValidationRejectsMissingTokenSecret`
  - `TestStrictProviderHonorsSeededUserPassword`
  - `TestProductionDiscoveryOmitsUnimplementedEndSessionEndpoint`
  - `TestBrowserSessionSilentAuthorizeAndPromptNone` (`max_age=0` branch)
- Validation commands passed on 2026-07-09:
  - `make lint`
  - `make logcopter-check`
  - `go test ./...`
  - `scripts/run-conformance.sh`
