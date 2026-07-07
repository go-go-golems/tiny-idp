---
Title: Investigation diary
Ticket: TINYIDP-PROD-001
Status: active
Topics:
    - auth
    - go
    - identity
    - oidc
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve.go
      Note: Phase 7 implementation recorded in diary
    - Path: repo://internal/domain/types.go
      Note: Phase 1 implementation recorded in diary
    - Path: repo://internal/fositeadapter/provider.go
      Note: Phase 4 implementation recorded in diary
    - Path: repo://internal/store/sqlite/store.go
      Note: Phase 6 implementation recorded in diary
    - Path: repo://ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/design-doc/01-production-embeddable-idp-design-and-implementation-guide.md
      Note: Primary design deliverable written during this diary
    - Path: repo://ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/reference/01-oidc-intern-textbook.md
      Note: Textbook deliverable written during this diary
    - Path: repo://ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/README.md
      Note: Downloaded source index for this investigation
ExternalSources: []
Summary: Chronological diary for creating the production embeddable IdP reorganization ticket and research deliverables.
LastUpdated: 2026-07-07T14:48:25.256086109-04:00
WhatFor: Use this to resume or review the research/design work for TINYIDP-PROD-001.
WhenToUse: Before continuing implementation, upload, validation, or ticket bookkeeping.
---



# Diary

## Goal

This diary captures the creation of the TINYIDP-PROD-001 docmgr ticket, the downloaded protocol/security sources, the repository investigation, and the two long-form deliverables: a production reorganization design guide and an OIDC intern textbook.

## Step 1: Create the ticket and define the deliverables

I created a new docmgr ticket for the production embeddable IdP reorganization. The ticket scope is intentionally documentation-first: preserve the current mock as a development/testing engine, design the strict production side, and give a new intern enough background to implement the work safely.

The ticket now contains a primary design document, an intern textbook, this diary, a `sources/` directory for downloaded references, and a task list that tracks source download, architecture analysis, writing, validation, and reMarkable upload.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to reorg the current local dev tiny-idp and implement the prod side of it.

Here is extensive research done by our colleague that should get you stsarted.

Make sure to download all the documents needed with defuddle and store in sources/ of the ticket, and look on the owasp best practices cheatsheet to validate we are doing things correctly, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

Also write a textbook for the new intern about all the internals and needed parts of an IDP first, in a textbook authoring style (see skill, no analogies). Upload to remarkable as well.

---

# Embeddable OIDC Identity Provider: Design and Implementation Guide

[The user prompt then included a long colleague research brief covering OIDC vocabulary, Authorization Code Flow with PKCE, security profile, current tiny-idp baseline, Fosite integration, package layout, domain model, endpoints, stores, key management, sessions, consent, claims, config, implementation phases, testing strategy, operations, intern checklist, and definition of done.]"

**Assistant interpretation:** Create a new docmgr ticket in the current `tiny-idp` repository, gather external OIDC/OAuth/Fosite/OWASP references into `sources/`, inspect current code, write a detailed design/implementation guide, write a textbook-style intern primer, validate with docmgr, and upload the deliverables to reMarkable.

**Inferred user intent:** The user wants a reviewable, source-backed implementation package that can guide a future intern or engineer through reorganizing the mock local IdP and building a production embeddable IdP without mixing test-only behavior into production.

### What I did

- Ran `docmgr status --summary-only` and confirmed docmgr uses `tiny-idp/ttmp` as the ticket root.
- Created ticket `TINYIDP-PROD-001` with title `Production embeddable IdP reorganization` and topics `auth,go,identity,oidc,testing`.
- Added docs:
  - `design-doc/01-production-embeddable-idp-design-and-implementation-guide.md`
  - `reference/01-oidc-intern-textbook.md`
  - `reference/02-investigation-diary.md`
- Added tasks for source download, current architecture analysis, guide writing, textbook writing, docmgr validation, and reMarkable upload.

### Why

- The work is large enough to need a ticket workspace, sources, references, and a diary.
- The user specifically asked for docmgr storage and reMarkable delivery.
- The production implementation needs evidence-backed design before code changes because the security boundary is subtle.

### What worked

- Ticket creation succeeded at `ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization`.
- Docmgr generated index/tasks/changelog plus the three requested documents.

### What didn't work

- N/A.

### What I learned

- The repository already has several completed `tinyidp` tickets, so the new ticket should follow the same ticket style and topics.

### What was tricky to build

- The user asked for both a production design and an intern textbook. These are related but different deliverables. I separated them so the design guide can be implementation-facing and the textbook can teach OIDC concepts directly.

### What warrants a second pair of eyes

- The exact scope of “implement the prod side” should be confirmed before code changes begin. In this step I created the ticket and deliverables rather than implementing the full Fosite/SQLite/embedded provider code, because the requested output centered on research, design, intern docs, sources, and upload.

### What should be done in the future

- Confirm the first code implementation phase after the user reviews the design guide.

### Code review instructions

- Start with `ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/index.md` and `tasks.md`.
- Validate with `docmgr ticket list --ticket TINYIDP-PROD-001` and `docmgr doc list --ticket TINYIDP-PROD-001`.

### Technical details

Commands run:

```bash
cd tiny-idp
docmgr ticket create-ticket --ticket TINYIDP-PROD-001 --title "Production embeddable IdP reorganization" --topics auth,go,identity,oidc,testing
docmgr doc add --ticket TINYIDP-PROD-001 --doc-type design-doc --title "Production embeddable IdP design and implementation guide"
docmgr doc add --ticket TINYIDP-PROD-001 --doc-type reference --title "OIDC intern textbook"
docmgr doc add --ticket TINYIDP-PROD-001 --doc-type reference --title "Investigation diary"
```

## Step 2: Download protocol and security references

I created a `sources/` directory under the ticket and downloaded OIDC, OAuth security, Fosite, OpenID Discovery, OpenID Certification, and OWASP cheat sheet references with Defuddle. These sources make the design reviewable without requiring the reader to fetch web pages again.

The OWASP set intentionally covers more than one cheat sheet because an IdP crosses OAuth, authentication, session management, CSRF, TLS, and logging boundaries.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Use Defuddle to collect all external references needed for the design and explicitly include OWASP best-practice material.

**Inferred user intent:** The user wants the ticket to be self-contained and source-backed.

### What I did

- Created `sources/00-source-urls.txt`.
- Downloaded 12 source documents into `sources/*.md` using `defuddle parse <url> --md | fold -w 100 -s`.
- Added `sources/README.md` explaining each downloaded source and why it matters.

### Why

- The design guide needs protocol and security references beyond the local codebase.
- Defuddle output keeps the ticket portable and reduces future web-fetch dependence.

### What worked

- All 12 downloads succeeded.
- File sizes looked plausible, including large OIDC Core and RFC 9700 pages.

### What didn't work

- N/A.

### What I learned

- `defuddle` output was wrapped with `fold` to avoid the known single-line markdown issue described by the defuddle skill.

### What was tricky to build

- The phrase “owasp best practices cheatsheet” is singular, but the IdP surface spans several OWASP cheat sheet domains. I downloaded OAuth2 plus supporting Authentication, Session Management, CSRF, TLS, and Logging cheat sheets so the design can validate the full browser-login and token lifecycle.

### What warrants a second pair of eyes

- Confirm whether the final implementation should also add an OWASP ASVS mapping document. The current design uses OWASP cheat sheets but does not create a full ASVS control matrix.

### What should be done in the future

- Add `docs/security-profile.md` with a formal OWASP/ASVS mapping during the hardening phase.

### Code review instructions

- Inspect `sources/README.md` first.
- Spot-check source files under `sources/` for successful markdown extraction.

### Technical details

Downloaded URLs:

```text
https://openid.net/specs/openid-connect-core-1_0.html
https://datatracker.ietf.org/doc/html/rfc9700
https://pkg.go.dev/github.com/ory/fosite
https://pkg.go.dev/github.com/ory/fosite/compose
https://openid.net/specs/openid-connect-discovery-1_0.html
https://openid.net/certification/
https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Security_Cheat_Sheet.html
https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html
```

## Step 3: Inspect current tiny-idp behavior

I inspected the current `tinyidp` codebase to anchor the design in actual files. The current server is a well-scoped local mock: it generates keys at startup, keeps all state in memory, exposes root and path-prefixed endpoints, includes loopback-only debug routes, and provides synthetic scenario behavior.

The important design conclusion is that this behavior should not be deleted. It should be isolated from production behavior behind an engine boundary.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Map the current architecture before proposing the production reorganization.

**Inferred user intent:** The user wants the implementation guide to reference concrete code and avoid speculative design.

### What I did

- Listed repository files with `rg --files`.
- Read line-numbered excerpts from:
  - `README.md`
  - `internal/server/server.go`
  - `internal/server/authorize.go`
  - `internal/server/token.go`
  - `internal/server/jwt.go`
  - `internal/server/session.go`
  - `internal/server/debug.go`
  - `internal/client/client.go`
  - `internal/user/user.go`
  - `internal/scenario/scenario.go`
  - `internal/cmds/serve.go`
  - `internal/sections/oidc/settings.go`
- Ran `go test ./...`.

### Why

- The design guide needed file-level evidence.
- The production plan depends on preserving known-good mock behavior while moving production semantics into new packages.

### What worked

- `go test ./...` passed.
- The source comments are unusually helpful and directly document mock-only assumptions such as in-memory state and per-process signing keys.

### What didn't work

- N/A.

### What I learned

- `internal/server/server.go` explicitly states that all state is in-memory and signing keys rotate on restart intentionally for a test tool.
- `internal/server/authorize.go` already has a good handwritten validation chokepoint.
- `internal/server/token.go` already deletes authorization codes atomically and rotates refresh tokens in memory.
- `internal/server/jwt.go` advertises `plain` PKCE and includes JWKS/key failure modes useful for RP tests.
- `internal/server/debug.go` is loopback-protected but still should not exist in production.

### What was tricky to build

- The current code has several behaviors that look insecure in isolation but are correct for a mock. The design guide had to classify these as mock-only strengths rather than implementation mistakes.

### What warrants a second pair of eyes

- The first code phase should be reviewed for accidental behavior changes in mock mode. The default CLI behavior should remain unchanged.

### What should be done in the future

- Add regression tests around `tinyidp serve` defaulting to mock mode once the `--engine` flag lands.

### Code review instructions

- Start with `internal/server/server.go` and `internal/server/jwt.go` to understand why the current engine is not production.
- Then read `internal/scenario/scenario.go` to understand why the mock engine is valuable.

### Technical details

Verification command:

```bash
cd tiny-idp && go test ./...
```

Result:

```text
ok  github.com/manuel/tinyidp/internal/server  7.052s
ok  github.com/manuel/tinyidp/internal/client  0.002s
ok  github.com/manuel/tinyidp/internal/cmds    0.012s
ok  github.com/manuel/tinyidp/internal/scenario 0.003s
ok  github.com/manuel/tinyidp/internal/sections/oidc 0.004s
ok  github.com/manuel/tinyidp/internal/user    0.002s
```

## Step 4: Write the design guide and intern textbook

I wrote two separate long-form documents. The design guide is implementation-facing: it maps current code, identifies production gaps, proposes package layout and APIs, records architecture decisions, and lays out phases and tests. The textbook is concept-facing: it explains the internals of an IdP in direct technical prose before the intern starts implementing code.

The two documents intentionally overlap on vocabulary but serve different review moments. The textbook should be read first by a new intern; the design guide should be used while implementing and reviewing the production engine.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce a clear technical implementation guide and a textbook-style OIDC primer with prose, bullets, pseudocode, diagrams, API references, and file references.

**Inferred user intent:** The user wants onboarding-quality documentation that turns the colleague research into project-specific implementation guidance.

### What I did

- Rewrote `design-doc/01-production-embeddable-idp-design-and-implementation-guide.md` with:
  - executive summary,
  - problem statement,
  - source list,
  - current-state analysis,
  - gap analysis,
  - proposed architecture,
  - API sketches,
  - Fosite adapter guidance,
  - HTTP routes,
  - security profile validated against OWASP categories,
  - decision records,
  - phased implementation plan,
  - testing strategy,
  - intern checklist,
  - review guide,
  - risks and references.
- Rewrote `reference/01-oidc-intern-textbook.md` in a textbook-authoring style with no analogies.

### Why

- The design guide and textbook answer different needs: implementation precision and conceptual onboarding.

### What worked

- The design guide now references current files and line ranges.
- The textbook directly explains OP/RP, Authorization Code Flow, clients, scopes, tokens, JWKS, refresh rotation, sessions, consent, discovery, audit, and the mock/production split.

### What didn't work

- N/A.

### What I learned

- The current codebase already encodes many useful semantics that the production design should keep conceptually: exact redirect matching, code one-time use, token response no-store headers, prompt/max_age behavior, and path-based issuer support.

### What was tricky to build

- The design needed to be strict without implying that the mock is bad. I handled this by naming mock-only behavior explicitly and assigning production behavior to new packages and engine boundaries.

### What warrants a second pair of eyes

- The proposed `pkg/embeddedidp.Options` API should be reviewed before implementation because it will become the public contract.
- The Fosite composition sketch should be checked against the exact pinned Fosite version when coding starts.

### What should be done in the future

- Convert the design guide phases into individual implementation tickets if the code work will be split across multiple PRs.

### Code review instructions

- Read the textbook first for concepts.
- Review the design guide decision records next.
- Then review package layout and public API sketches.

### Technical details

Primary docs:

```text
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/design-doc/01-production-embeddable-idp-design-and-implementation-guide.md
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/reference/01-oidc-intern-textbook.md
```

## Step 5: Add precise phase tasks and commit the ticket package

I expanded the ticket task list into phase-level tracking items before starting code. The tasks now break phases 1 through 7 into concrete substeps for domain modeling, validation, storage interfaces, memory persistence, metadata, keys, strict adapter work, embedded API, SQLite, and CLI engine selection.

This step also committed the ticket package that had been created earlier: downloaded sources, design guide, intern textbook, diary, tasks, changelog, and index files. That gives the implementation work a stable documentation baseline.

### Prompt Context

**User prompt (verbatim):** "ok, add etailed phases and tasks per phase so we can precisely keep track of what you are working on.
Then work on this task by task, keeping a detailed diary as you work, and do phases 1-7. I am going swimming for a while so you should really go at it and get it all done for when I come back, don't stop midway. 

commit at appropriate intervals."

**Assistant interpretation:** Expand the ticket with detailed implementation tasks, then implement phases 1 through 7 autonomously, updating the diary and committing at meaningful boundaries.

**Inferred user intent:** The user wants progress to be trackable while they are away and wants commits that separate documentation, foundation work, and implementation work.

**Commit (code):** 3039365092ed10adf5bc38b61d04492f6dbf8493 — "Docs: expand production IdP phase tasks"

### What I did

- Added 21 phase tasks to `tasks.md`:
  - Phase 1 domain model and validation tasks.
  - Phase 2 storage interfaces and memory store tasks.
  - Phase 3 OIDC metadata and key/JWKS tasks.
  - Phase 4 strict adapter seam and authorization-code-flow spike tasks.
  - Phase 5 embedded provider API tasks.
  - Phase 6 SQLite persistence tasks.
  - Phase 7 CLI engine-switch tasks.
- Committed the full ticket package and sources.

### Why

- The user asked for precise task tracking before implementation.
- Committing the ticket state first makes later code commits easier to review independently.

### What worked

- `docmgr task add` created stable task IDs for every phase subtask.
- The first commit captured the documentation baseline.

### What didn't work

- The initial commit was larger than ideal because the prior turn had created all source downloads and documents but had not committed them yet.

### What I learned

- Future ticket creation work should commit the ticket package immediately after upload/validation, before the next user asks for implementation.

### What was tricky to build

- The phase list needed to be detailed enough for tracking without exploding into dozens of tiny tasks that would make bookkeeping slower than implementation. I grouped each phase into three concrete tasks.

### What warrants a second pair of eyes

- Review whether the phase task names should become separate follow-up tickets once the production code leaves scaffold/spike status.

### What should be done in the future

- Keep phase tasks checked as code lands, not all at final handoff.

### Code review instructions

- Review `tasks.md` and confirm phase names match the design guide.
- Use `git show --stat 3039365` to see the documentation baseline.

### Technical details

Commands included:

```bash
docmgr task add --ticket TINYIDP-PROD-001 --text "Phase 1.1: ..."
git add ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization
git commit -m "Docs: expand production IdP phase tasks"
```

## Step 6: Implement phases 1 through 3 foundation packages

I implemented the production foundation before touching HTTP routing. This created project-owned domain types, validation rules, storage contracts, a concurrency-safe memory store, strict OIDC discovery helpers, and RSA/JWKS helpers. These packages give later phases a shared vocabulary that does not depend on the current mock server package.

The implementation deliberately keeps production semantics out of the mock engine. The current `internal/server` package remains unchanged; new production-facing packages live beside it.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Work through phases in order and commit at the foundation boundary.

**Inferred user intent:** The user wants durable progress that can be reviewed and continued even if later phases need refinement.

**Commit (code):** 05b7189c9fe75f3a9c48fecb894791b8a6c0a50c — "Add production IdP domain and stores"

### What I did

- Added `internal/domain`:
  - `Client`, `User`, `Grant`, `AuthorizationCode`, `AccessToken`, `RefreshToken`, `Session`, and `SigningKey`.
  - `Mode` with `DevMode` and `ProductionMode`.
  - Validation for empty client IDs, wildcard redirects, redirect fragments, production HTTP redirect URIs, public-client PKCE requirements, and subject/email mistakes.
  - Scope parsing/deduplication and claim filtering by scope.
  - HMAC-SHA256 secret hashing helper.
- Added `internal/storage`:
  - Store interfaces for clients, users, grants, authorization codes, access tokens, refresh tokens, sessions, and keys.
  - Shared store test suite covering one-time code consumption, parallel code consumption, expired code rejection, refresh-token rotation/reuse detection, and signing key activation.
- Added `internal/store/memory`:
  - Concurrency-safe maps for all store interfaces.
  - Refresh-token family revocation on reuse detection.
  - `Persistent() bool` reporting false for production validation.
- Added `internal/oidcmeta`:
  - Issuer parsing and validation.
  - Path-based discovery path construction.
  - Conservative production discovery metadata that advertises `code` and `S256` only.
- Added `internal/keys`:
  - RSA key generation.
  - PEM parsing.
  - Public JWKS construction.
  - Public-only JWKS tests.

### Why

- The production provider needs domain and persistence contracts before an HTTP engine can be meaningful.
- Shared tests force memory and future SQLite stores to preserve the same replay and rotation invariants.

### What worked

- Targeted tests passed:

```bash
go test ./internal/domain ./internal/storage ./internal/store/memory ./internal/oidcmeta ./internal/keys
```

- Store suite found and codified the expected one-winner behavior for concurrent authorization-code consumption.

### What didn't work

- N/A for this step.

### What I learned

- The memory store can model refresh-token family reuse detection more strictly than the original mock by retaining consumed refresh-token records instead of deleting them.
- Treating empty `AllowedScopes` as deny-by-default in the production domain prevents accidental permissive clients; the mock keeps its own legacy permissive behavior separately.

### What was tricky to build

- The store suite had to verify concurrency without making timing assumptions. I used concurrent goroutines racing to consume the same code and asserted exactly one success.
- Refresh-token reuse detection had to preserve the old token record after rotation so a second use can be identified as reuse instead of a generic unknown token.

### What warrants a second pair of eyes

- The production redirect URI validation currently permits HTTP loopback in dev mode and rejects non-loopback HTTP in production. Review whether production should also reject loopback HTTP clients unless a separate explicit local-client escape hatch is provided.
- The domain `AllowedScopes` deny-by-default behavior differs from the mock client's empty-allowed-scopes behavior intentionally; reviewers should confirm this split is acceptable.

### What should be done in the future

- Add fuzz tests for issuer parsing, redirect URI parsing, and scope parsing.
- Add password-hash utilities once real password login is implemented.

### Code review instructions

- Start at `internal/domain/types.go` and `internal/domain/validate.go`.
- Then review `internal/storage/testsuite.go`, because it defines behavioral contracts that every store must satisfy.
- Validate with:

```bash
go test ./internal/domain ./internal/storage ./internal/store/memory ./internal/oidcmeta ./internal/keys
```

### Technical details

Key files:

```text
internal/domain/types.go
internal/domain/validate.go
internal/storage/interfaces.go
internal/storage/testsuite.go
internal/store/memory/store.go
internal/oidcmeta/discovery.go
internal/keys/keys.go
```

## Step 7: Implement phases 4 through 7 strict engine scaffold

I implemented the strict engine scaffold, embedded provider API, SQLite persistence, and CLI engine switch. The result is a working production-like path that supports strict Authorization Code + S256 PKCE, discovery, JWKS, token exchange, refresh rotation, UserInfo, no debug routes, `pkg/embeddedidp.New`, a SQLite store, and `tinyidp serve --engine fosite` wiring.

There is an important caveat: the package is named `internal/fositeadapter` and exposes the intended Fosite adapter seam plus the explicit production handler list, but this commit does not yet bind to Ory Fosite itself. The handler is a strict in-repo spike that lets phases 5 through 7 compile and run while preserving the future adapter boundary. The next production-hardening pass should replace the handwritten strict protocol internals with actual Fosite composition behind the same package boundary.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Continue through phases 4-7 and commit after the strict engine, embedded API, SQLite, and CLI wiring compile and pass tests.

**Inferred user intent:** The user wants an end-to-end scaffold rather than isolated packages only.

**Commit (code):** 1a796cf2b12f28d1227f6549a88ab9411b51b0d3 — "Add strict embedded IdP engine scaffold"

### What I did

- Extended `internal/keys` with `SignJWT` for RS256 compact JWT signing.
- Added `internal/fositeadapter`:
  - Explicit `ProductionHandlerFactories` list matching the planned Fosite composition.
  - Strict discovery, JWKS, authorize, token, UserInfo, health, and readiness handlers.
  - S256 PKCE requirement.
  - One-time code consumption through the store.
  - ID Token signing from the active signing key.
  - Opaque access tokens and rotating refresh tokens.
  - Tests for end-to-end authorization-code flow and absence of `/debug`.
- Added `pkg/embeddedidp`:
  - Public `Options`, `Provider`, and `New` API.
  - Production validation for HTTPS issuer, secure cookies, persistent store reporting, configured clients, and active signing key.
  - Tests for production validation and no debug route.
- Added `internal/store/sqlite`:
  - Embedded migration schema.
  - JSON-backed SQLite implementation for every store interface.
  - Store-suite test against SQLite.
  - Restart-stable signing-key test.
- Added `examples/embedded`:
  - Minimal build-ignored example showing `embeddedidp.New` wiring.
- Added `--engine` to the OIDC settings section and print-config output.
- Updated `internal/cmds/serve.go`:
  - Default `engine=mock` preserves existing behavior.
  - `engine=fosite` builds the strict provider over a memory store seeded from current clients and scenarios.
  - Unknown engines fail with a clear error.
- Added `internal/cmds/strict_engine_test.go` for discovery and debug-route smoke coverage.

### Why

- Phases 4 through 7 needed a runnable path, not just APIs.
- The strict provider lets developers start compatibility testing with `tinyidp serve --engine fosite` while the mock remains the default.
- SQLite persistence proves the store contracts are not memory-only.

### What worked

- Full test suite passed:

```bash
go test ./...
```

- Strict adapter end-to-end test exercises:
  - authorize POST,
  - code callback,
  - token exchange,
  - ID/access/refresh token response,
  - UserInfo.
- SQLite store passed the same store suite as memory.
- Signing keys persisted across SQLite reopen.
- CLI strict-engine smoke test confirmed discovery works and `/debug` is absent.

### What didn't work

- I did not integrate the real Ory Fosite dependency in this pass. The adapter seam is present, and the strict handler follows the intended Authorization Code + PKCE behavior, but Fosite's `NewAuthorizeRequest`, `NewAccessRequest`, and response writers are not yet used.
- The SQLite implementation is intentionally simple and JSON-backed. It is suitable for exercising the persistence contract, but it is not yet the final relational schema described in the design guide.
- Login in the strict engine is a minimal form/post flow backed by seeded users. It is not a hardened login subsystem with CSRF, audit, rate limiting, or consent yet.

### What I learned

- The existing scenario registry can seed a strict dev engine well enough for compatibility tests, but scenario failure behavior is intentionally ignored in strict mode.
- Keeping `mock` as the default made CLI wiring low-risk: existing tests continued to pass, and strict behavior is opt-in.

### What was tricky to build

- The strict engine needed to share current client/user config without importing mock-only behavior into production packages. I solved this by converting `client.Client` and `scenario.Scenario` values into `domain.Client` and `domain.User` inside `internal/cmds/serve.go` only.
- SQLite refresh-token reuse detection had to update token-family records while avoiding assumptions about relational columns. I used JSON records plus indexed `grant_id` as an interim representation.
- The test for authorization redirect initially used `http.PostForm`, which followed the redirect to `rp.example` and failed DNS lookup. I fixed the test by using an HTTP client with `CheckRedirect: http.ErrUseLastResponse`.

### What warrants a second pair of eyes

- `internal/fositeadapter/provider.go` should be reviewed as a spike, not as the final Fosite integration. The next pass should replace handwritten strict protocol handling with actual Ory Fosite calls behind this boundary.
- `internal/store/sqlite/store.go` should be reviewed for transaction boundaries before production use. The current mutex-backed implementation passes the suite but is not a final multi-process concurrency design.
- `pkg/embeddedidp.Options` should be reviewed before external consumers rely on it.
- The strict engine currently uses a dev in-memory store when started from the CLI. Production embedding should use SQLite or another persistent store.

### What should be done in the future

- Replace the strict handwritten protocol internals with Ory Fosite composition.
- Add CSRF, consent, audit events, secure cookie sessions, and rate-limiting hooks in the Phase 8 hardening pass.
- Normalize SQLite schema from JSON-backed records into explicit columns before declaring storage stable.
- Add `tinyidp print-config` docs for the new `engine` field.

### Code review instructions

- Start with the public API in `pkg/embeddedidp/options.go` and `pkg/embeddedidp/provider.go`.
- Review strict protocol flow in `internal/fositeadapter/provider.go`.
- Review persistence behavior in `internal/store/sqlite/store.go` and `internal/storage/testsuite.go` together.
- Review CLI wiring in `internal/cmds/serve.go` and confirm `mock` remains default.
- Validate with:

```bash
go test ./...
go run ./cmd/tinyidp print-config
go run ./cmd/tinyidp print-config --engine fosite
```

### Technical details

Key files:

```text
internal/fositeadapter/provider.go
internal/fositeadapter/provider_test.go
pkg/embeddedidp/options.go
pkg/embeddedidp/provider.go
internal/store/sqlite/store.go
internal/store/sqlite/migrations/001_schema.sql
internal/cmds/serve.go
internal/sections/oidc/section.go
examples/embedded/main.go
```
