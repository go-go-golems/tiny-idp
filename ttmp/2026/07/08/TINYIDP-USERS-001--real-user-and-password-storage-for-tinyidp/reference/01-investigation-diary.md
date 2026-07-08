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
RelatedFiles:
    - Path: repo://internal/authn/password.go
      Note: Password authentication service with lockout and audit behavior (commit ff5f30c)
    - Path: repo://internal/domain/types.go
      Note: Password credential and account-security domain models (commit 24e0323)
    - Path: repo://internal/fositeadapter/provider.go
      Note: Strict login POST now calls password authenticator (commit ff5f30c)
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: Strict password login regression coverage (commit ff5f30c)
    - Path: repo://internal/passwordhash/argon2id.go
      Note: Argon2id encoded password hashing implementation (commit 24e0323)
    - Path: repo://internal/storage/interfaces.go
      Note: Credential and account-security store contracts (commit 24e0323)
    - Path: repo://internal/store/memory/store.go
      Note: Memory credential/security-state store implementation (commit 24e0323)
    - Path: repo://internal/store/sqlite/migrations/002_password_credentials.sql
      Note: SQLite credential/security-state schema (commit 24e0323)
    - Path: repo://internal/store/sqlite/store.go
      Note: SQLite credential/security-state store implementation and migration runner (commit 24e0323)
    - Path: repo://pkg/embeddedidp/options.go
      Note: Embeddable authenticator option (commit ff5f30c)
ExternalSources: []
Summary: Chronological notes for the real user/password storage design ticket.
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: Use this to resume or review the password-storage design work.
WhenToUse: Read before implementing TINYIDP-USERS-001 or updating its design guide.
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

## Step 2: Add Password Hashing and Credential Stores

I implemented the first production credential layer: a self-describing Argon2id password hash package, domain models for password credentials and account-security state, storage interfaces, and memory/SQLite persistence. This gives the rest of the ticket a durable place to store password verifiers without putting hashes on `domain.User`.

The key architectural result is that stores now own two new concepts: `PasswordCredential` for the verifier and credential lifecycle, and `AccountSecurityState` for failure counters and lockout timestamps. The SQLite store also now runs all embedded migrations in sorted order, so future credential/admin migrations can be added incrementally.

### Prompt Context

**User prompt (verbatim):** "Build TINYIDP-USERS-001 and commit at appropriate intervals, keep a detailed diary."

**Assistant interpretation:** Implement the real user/password storage ticket in focused commits while maintaining the docmgr diary and ticket bookkeeping.

**Inferred user intent:** Move from design-only productization docs to working production user/password infrastructure with reviewable incremental commits.

**Commit (code):** 24e0323 — "Add password credential storage primitives"

### What I did
- Added `internal/passwordhash` with Argon2id hash, parse, verify, mismatch, malformed-hash, and rehash detection tests.
- Added `domain.PasswordCredential`, `domain.PasswordHashParams`, and `domain.AccountSecurityState`.
- Added `storage.PasswordCredentialStore` and `storage.AccountSecurityStore` to the aggregate store contract.
- Implemented credential/security-state methods in the memory store.
- Added SQLite migration `002_password_credentials.sql`.
- Updated SQLite migrations to apply all embedded `*.sql` files in sorted order.
- Implemented credential/security-state methods in the SQLite store.
- Extended the storage suite to verify credential lookup, duplicate login rejection, delete behavior, and security-state reset.
- Ran `gofmt` and targeted tests:
  - `go test ./internal/passwordhash ./internal/store/memory ./internal/store/sqlite ./internal/storage`

### Why
- Password hashes must not live on `domain.User`, because user records feed OIDC profile and userinfo behavior.
- Authentication service and admin commands need durable credential and lockout state before provider login can be made strict.
- SQLite needs versioned migrations before adding new tables.

### What worked
- The existing store suite pattern made it easy to test memory and SQLite implementations with one set of invariants.
- The encoded Argon2id string keeps hash parameters next to the derived key, so policy upgrades can detect `needsRehash` later.

### What didn't work
- No blocking failures. One design detail needed correction during implementation: SQLite `INSERT OR REPLACE` would have allowed a duplicate login to move from one user ID to another, unlike memory store behavior. I added an explicit existing-login check so a login already owned by another user returns `storage.ErrDuplicate`.

### What I learned
- The store aggregate is central enough that adding credential interfaces immediately surfaces all implementations that need to be updated.
- Migration ordering needed to become generic before adding `002_password_credentials.sql`; hard-coding only `001_schema.sql` would make later product schema work brittle.

### What was tricky to build
- Duplicate-login semantics were the sharpest edge. SQLite's `INSERT OR REPLACE` is not an upsert in the safety sense; it can delete and replace the row that violates a unique constraint. The fix was to query by login first and reject if the existing credential belongs to a different user ID.
- Argon2id tests need low-cost parameters. The production default is stronger, but the package exposes `TestParams()` so unit tests stay fast without changing the encoded hash format.

### What warrants a second pair of eyes
- Review the Argon2id default parameters and whether 64 MiB / 3 iterations / parallelism 2 is the desired production baseline for tinyidp's deployment targets.
- Review whether the SQLite credential schema should later add explicit searchable columns beyond `user_id`, `login`, and serialized credential data.

### What should be done in the future
- Build the authentication service on top of the new credential/security stores.
- Wire strict provider login to that service in production mode.
- Add admin commands that create users and write credentials transactionally.

### Code review instructions
- Start with `internal/passwordhash/argon2id.go`, then read `internal/domain/types.go` around the new credential models.
- Review store behavior in `internal/store/memory/store.go` and `internal/store/sqlite/store.go`.
- Validate with `go test ./internal/passwordhash ./internal/store/memory ./internal/store/sqlite ./internal/storage`.

### Technical details
- Encoded hash format: `$argon2id$v=19$m=<KiB>,t=<iterations>,p=<parallelism>$<salt>$<key>`.
- New SQLite migration: `internal/store/sqlite/migrations/002_password_credentials.sql`.

## Step 3: Add Password Authentication Service and Strict Login Wiring

I added the password authentication service and wired the strict Fosite login POST path through it. The provider now uses an authenticator abstraction, so development strict runs can preserve passwordless seeded-user compatibility while production and configured authenticators verify stored password credentials before creating an IdP browser session.

This step moves the ticket from storage-only infrastructure to behavior change: a login can now be rejected because of a wrong password, disabled account, or lockout state. The public browser error remains generic, while audit reasons use stable internal codes.

### Prompt Context

**User prompt (verbatim):** (same as Step 2)

**Assistant interpretation:** Continue implementing TINYIDP-USERS-001 with a focused commit for authentication behavior and provider integration.

**Inferred user intent:** Make strict login actually depend on real password credentials while preserving existing mock/dev test ergonomics.

**Commit (code):** ff5f30c — "Enforce password authentication in strict login"

### What I did
- Added `internal/authn.PasswordService` with login normalization, dummy verification for missing users, disabled/locked checks, failure recording, lockout, success reset, rehash-on-login support, and audit events.
- Added password-service tests for success, lockout, wrong-password failure, and dev-only passwordless compatibility.
- Added `fositeadapter.PasswordAuthenticator` and `Options.Authenticator`.
- Updated strict provider login POST handling to call `AuthenticatePassword` before browser-session creation.
- Kept default dev-mode behavior compatible by allowing passwordless login only when no credential exists and the provider is not in production mode.
- Added `embeddedidp.Options.Authenticator` so embedding applications can supply their own authenticator.
- Added provider test coverage proving an authenticator-backed strict login rejects a wrong password and issues an authorization code for the correct password.
- Updated the production consent test to provide a real credential/authenticator because that test is about consent behavior, not password bypass.
- Ran targeted tests:
  - `go test ./internal/authn ./internal/fositeadapter ./pkg/embeddedidp`

### Why
- Production strict mode must not accept a login name alone.
- The login handler is the security boundary where credentials become an authenticated browser session.
- Development and conformance workflows still need the old seeded-user ergonomics unless credentials are explicitly configured.

### What worked
- The authenticator abstraction kept provider changes small.
- Existing browser-session and consent tests exposed the compatibility edges immediately.
- The same memory store credential implementation from Step 2 was enough for fast provider tests.

### What didn't work
- Initial targeted tests failed because the first `PasswordService` version rejected empty passwords before checking the dev-mode passwordless policy. Existing dev strict tests submit only `login`, so they returned `401 invalid login or password`.
- `TestProductionProviderDefaultsToStoredConsent` also failed with `401` because production default authentication correctly requires a credential. I updated that test to create a real credential and submit the password.
- The new provider password test initially got `403` for the correct password because production mode defaults to stored consent. I set `Consent: AlwaysSkipConsent{}` in that focused authentication test.

### What I learned
- The mode boundary is important: dev compatibility should be explicit and narrow. The implemented rule is: dev may accept passwordless login only when no credential is stored; if a credential exists, password verification is used.
- Existing consent tests are useful canaries because production defaults interact: authentication must succeed before consent can be tested.

### What was tricky to build
- The main tricky part was sequencing policy checks. Missing password should be fatal in production, but in dev it must be allowed only if the user has no credential. I fixed this by allowing empty passwords through the initial syntax gate only when the passwordless policy is enabled, then making the missing-credential branch decide whether that is acceptable.
- Another subtle edge is audit duplication. The authentication service emits `password.login.*` events, while the provider still emits high-level `login.*` events. This preserves existing audit names while adding credential-specific evidence.

### What warrants a second pair of eyes
- Review the default dev passwordless behavior. It preserves compatibility, but future product config should make production credential requirements even more explicit.
- Review whether password rehash-on-login should remain inline or move to a bounded background path for high-traffic deployments.

### What should be done in the future
- Add admin/service commands that create users and write password credentials.
- Add documentation describing strict production authentication semantics.
- Consider adding AMR propagation into ID Tokens in a future OIDC claims enhancement.

### Code review instructions
- Start with `internal/authn/password.go`, especially `AuthenticatePassword`.
- Then review `internal/fositeadapter/provider.go` around the POST `/authorize` login branch.
- Validate with `go test ./internal/authn ./internal/fositeadapter ./pkg/embeddedidp`.

### Technical details
- Public login failure response: `invalid login or password` with HTTP 401.
- Stable audit reasons: `invalid_credentials`, `account_disabled`, `account_locked`.
- Provider default: production uses credential-backed authentication; dev permits missing credentials for compatibility.
