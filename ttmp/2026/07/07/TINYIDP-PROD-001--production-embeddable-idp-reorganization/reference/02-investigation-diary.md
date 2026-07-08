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
    - Path: repo://docs/conformance.md
      Note: |-
        Step 13 conformance documentation
        Step 14 hosted runner usage
        Step 16 distinct-client hosted runbook
    - Path: repo://go.mod
      Note: Step 8 dependency change
    - Path: repo://internal/audit/audit.go
      Note: Step 10 audit implementation
    - Path: repo://internal/cmds/serve.go
      Note: |-
        Phase 7 implementation recorded in diary
        Step 16 extra-client registry parsing
    - Path: repo://internal/cmds/serve_test.go
      Note: Step 16 extra-client regression
    - Path: repo://internal/domain/types.go
      Note: Phase 1 implementation recorded in diary
    - Path: repo://internal/fositeadapter/audit_reason.go
      Note: Step 13 audit hardening
    - Path: repo://internal/fositeadapter/consent.go
      Note: |-
        Step 10 consent policy implementation
        Step 12 stored-consent implementation
    - Path: repo://internal/fositeadapter/consent_test.go
      Note: Step 12 consent policy tests
    - Path: repo://internal/fositeadapter/csrf.go
      Note: Step 10 hardening implementation
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Phase 4 implementation recorded in diary
        Step 8 real Fosite implementation
        Step 15 prompt/max-age/request-object hosted OIDF fixes
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: Step 15 request-object regression coverage
    - Path: repo://internal/fositeadapter/ratelimit.go
      Note: Step 11 rate-limit hardening implementation
    - Path: repo://internal/fositeadapter/session.go
      Note: |-
        Step 11 session hardening implementation
        Step 15 max-age session helper
    - Path: repo://internal/fositeadapter/session_test.go
      Note: Step 15 prompt/max-age regression coverage
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Step 9 durable Fosite storage implementation
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: Step 12 refresh-token reuse coverage
    - Path: repo://internal/keys/rotation.go
      Note: Step 13 key rotation
    - Path: repo://internal/sections/oidc/section.go
      Note: Step 16 extra-client CLI flag
    - Path: repo://internal/sections/oidc/settings.go
      Note: Step 16 extra-client settings
    - Path: repo://internal/store/sqlite/store.go
      Note: Phase 6 implementation recorded in diary
    - Path: repo://scripts/oidf_hosted_runner.py
      Note: Step 14 hosted OIDF Python automation
    - Path: repo://scripts/run-conformance.sh
      Note: Step 13 validation script
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

## Step 8: Replace the strict adapter spike with real Ory Fosite

I replaced the handwritten strict protocol internals in `internal/fositeadapter` with a real Ory Fosite composition. The package boundary stayed the same, but `/authorize`, `/token`, refresh-token handling, authorization-code persistence, PKCE validation, ID Token generation, token response writing, and token introspection now go through Fosite's provider APIs.

This closes the caveat from Step 7. The adapter is still intentionally narrow: tinyidp owns login lookup, scope-grant policy, discovery, JWKS, and UserInfo claim rendering, while Fosite owns OAuth/OIDC protocol machinery.

### Prompt Context

**User prompt (verbatim):** "ok, do it."

**Assistant interpretation:** Replace the previous strict in-repo spike with real Ory Fosite integration now, rather than leaving it as a future hardening task.

**Inferred user intent:** The user wants the Phase 4 implementation to match the design promise: a genuinely Fosite-backed strict engine, not just a compatible scaffold.

**Commit (code):** 572901a2ca8d6aacd6051ec0df0215ff3060bc3b — "Use Ory Fosite for strict IdP adapter"

### What I did

- Added `github.com/ory/fosite v0.49.0` to `go.mod` and `go.sum`.
- Reworked `internal/fositeadapter/provider.go` to:
  - Build a `fosite.Config` with Authorization Code lifetimes, token lifetimes, issuer, `EnforcePKCE`, `EnforcePKCEForPublicClients`, `EnablePKCEPlainChallengeMethod=false`, `ExactScopeStrategy`, and `offline_access` refresh-token scope.
  - Seed a Fosite `storage.MemoryStore` from the project domain client store.
  - Compose a real `fosite.OAuth2Provider` with:
    - `compose.OAuth2AuthorizeExplicitFactory`
    - `compose.OAuth2PKCEFactory`
    - `compose.OAuth2RefreshTokenGrantFactory`
    - `compose.OpenIDConnectExplicitFactory`
    - `compose.OpenIDConnectRefreshFactory`
    - `compose.OAuth2TokenIntrospectionFactory`
  - Use `compose.NewOAuth2HMACStrategy` for OAuth2 opaque tokens/codes.
  - Use `compose.NewOpenIDConnectStrategy` with the active project signing key for ID Tokens.
  - Use `NewAuthorizeRequest`, `NewAuthorizeResponse`, `WriteAuthorizeResponse`, `NewAccessRequest`, `NewAccessResponse`, and `WriteAccessResponse` in the handlers.
  - Use `IntrospectToken` for UserInfo bearer-token validation.
- Updated the strict CLI conversion to provide plaintext dev client secrets only to the adapter so Fosite can BCrypt-hash legacy dev secrets for its own client store.
- Adjusted the embedded example to use a public PKCE client, because production callers should not pass HMAC-hashed secrets to Fosite as if they were BCrypt hashes.
- Updated strict authorization-code-flow tests for Fosite's redirect status and entropy requirements.

### Why

- Fosite is the planned owner of OAuth/OIDC protocol validation.
- Keeping handwritten protocol internals would leave the strict engine with the same long-term maintenance problem as the mock.
- The previous scaffold proved package shape, but this step makes the adapter real.

### What worked

- Full test suite passes:

```bash
go test ./...
```

- `go run ./cmd/tinyidp print-config --engine fosite` works and shows `engine: fosite`.
- The strict end-to-end test still covers authorize → code → token → ID/access/refresh token → UserInfo.
- `/debug` remains absent from the strict provider.

### What didn't work

- The first Fosite run returned `invalid_state` because Fosite enforces state entropy. I updated the test to use a longer state and nonce instead of weakening the production configuration.
- The first authorization response returned an insecure-redirect error for `http://rp.example/callback`. Fosite's secure redirect checker permits localhost-style development HTTP, so I changed the test redirect URI to `http://localhost/callback`.
- The first Fosite authorization response returned a server error because the test `GlobalSecret` was shorter than the HMAC strategy requires. I changed the test secret to a 32+ byte value.
- The first UserInfo attempt failed with `request_unauthorized` because the Fosite composition did not include a token introspection handler. I added `compose.OAuth2TokenIntrospectionFactory` so `IntrospectToken` can validate access tokens for UserInfo.

### What I learned

- Fosite's defaults are useful security pressure: entropy and redirect-scheme checks caught weak test inputs immediately.
- Fosite stores client secrets as BCrypt hashes. Our domain-level HMAC secret hash is useful for opaque token lookup, but it is not a Fosite client-secret hash. The adapter must either receive BCrypt hashes or receive plaintext only at config-conversion time and hash them internally.

### What was tricky to build

- The adapter has two stores now: the project `storage.Store` for users, clients, signing keys, and JWKS, and Fosite's protocol store for authorization codes, PKCE sessions, access tokens, refresh tokens, and OIDC sessions. This is acceptable for the current strict dev engine, but persistent production Fosite storage still needs a domain-backed implementation later.
- UserInfo needed Fosite token introspection rather than direct access-token hash lookup because Fosite now owns token issuance.
- ID Token claims need to be placed into `openid.DefaultSession` before `NewAuthorizeResponse`, so Fosite can carry them through code exchange and token generation.

### What warrants a second pair of eyes

- Review `buildFositeStore`: production embedding should prefer BCrypt client-secret hashes in `domain.Client.SecretHash`; the CLI path uses plaintext dev secrets only during conversion.
- Review whether `OAuth2TokenIntrospectionFactory` should be documented as required for UserInfo even if the public introspection endpoint remains optional.
- Review persistence expectations: Fosite's memory store is fine for strict dev compatibility, but a production durable Fosite store is still a future storage-hardening task.

### What should be done in the future

- Implement a Fosite storage adapter backed by the project SQLite store so production embedded deployments do not rely on Fosite memory storage.
- Add CSRF, consent, audit, and secure cookie sessions in the hardening phase.
- Add ID Token validation tests that parse the JWT header/claims and verify against JWKS.

### Code review instructions

- Start with `internal/fositeadapter/provider.go` and look for the `compose.Compose` call.
- Confirm only code, PKCE, refresh, OpenID explicit/refresh, and introspection handlers are composed.
- Review `internal/fositeadapter/provider_test.go` for the end-to-end strict flow.
- Validate with:

```bash
go test ./...
go run ./cmd/tinyidp print-config --engine fosite
```

### Technical details

Key files:

```text
go.mod
go.sum
internal/fositeadapter/provider.go
internal/fositeadapter/provider_test.go
internal/cmds/serve.go
examples/embedded/main.go
```

## Step 9: Add durable SQLite-backed Fosite protocol storage

I continued from the real Fosite integration by removing the most important remaining shortcut: Fosite protocol state no longer has to live in Fosite's memory store when the project store is SQLite-backed. The adapter now detects stores that expose the underlying SQLite handle and uses a dedicated SQL-backed Fosite store for authorization codes, PKCE requests, OIDC sessions, access tokens, refresh tokens, and JWT replay records.

This matters because a production embedded IdP cannot lose outstanding authorization codes, access-token introspection state, or refresh-token state merely because the provider object is rebuilt. The new test proves the critical restart behavior by issuing a code with one provider instance, exchanging it after reopening the SQLite database with a second provider instance, and using the refresh token after reopening with a third provider instance.

### Prompt Context

**User prompt (verbatim):** "don't take any shortcuts"

**Assistant interpretation:** Continue beyond the real Fosite composition and remove the in-memory Fosite protocol-state shortcut by implementing durable protocol storage for SQLite.

**Inferred user intent:** The user wants the production side to be real enough to support restart-stable embedded deployments, not only a dev-mode strict engine.

**Commit (code):** 949ec1d808d5675adcf37c8e86ca6d158e9694ed — "Persist Fosite protocol state in SQLite"

### What I did

- Added `internal/fositeadapter/sqlstore.go`.
- Added a SQLite-backed Fosite store wrapper that implements the Fosite storage methods used by the composed handlers:
  - `GetClient`
  - `CreateAuthorizeCodeSession`
  - `GetAuthorizeCodeSession`
  - `InvalidateAuthorizeCodeSession`
  - `CreatePKCERequestSession`
  - `GetPKCERequestSession`
  - `DeletePKCERequestSession`
  - `CreateOpenIDConnectSession`
  - `GetOpenIDConnectSession`
  - `DeleteOpenIDConnectSession`
  - `CreateAccessTokenSession`
  - `GetAccessTokenSession`
  - `DeleteAccessTokenSession`
  - `CreateRefreshTokenSession`
  - `GetRefreshTokenSession`
  - `DeleteRefreshTokenSession`
  - `RevokeRefreshToken`
  - `RevokeAccessToken`
  - `RotateRefreshToken`
  - JWT replay methods used by Fosite extension paths.
- Added `SQLDB() *sql.DB` to `internal/store/sqlite.Store` so the Fosite adapter can share the same SQLite file without exposing ownership of closing the handle.
- Changed `internal/fositeadapter.NewProvider` so:
  - SQLite-backed stores use the durable SQL Fosite store.
  - Non-SQL dev stores still use Fosite's memory store for local strict-engine tests.
- Added `internal/fositeadapter/sqlstore_test.go` to prove restart durability:
  - provider 1 creates an authorization code;
  - provider 2 exchanges that code after reopening SQLite;
  - provider 3 uses the refresh token after reopening SQLite again.

### Why

- Real Fosite composition was necessary but insufficient for production if all Fosite state remained in memory.
- Authorization codes and refresh tokens are protocol state. They must survive provider object restart when the host selected a persistent SQLite store.

### What worked

- The new restart-durability test passed:

```bash
go test ./internal/fositeadapter -run TestFositeSQLiteStoreSurvivesProviderRestart -v
```

- The full test suite passed:

```bash
go test ./...
```

### What didn't work

- My first attempt to inspect multiple Fosite docs in one `go doc` command failed because `go doc` expects one package/symbol target at a time. I reran the relevant docs separately and then inspected Fosite's `storage.MemoryStore` implementation as the behavioral reference.

### What I learned

- Fosite's memory store persists several distinct state classes: authorization code state, PKCE state, OIDC session state, access-token state, refresh-token state, request-id indexes, and JTI replay records. A durable adapter has to model those separately rather than dumping a single token table into SQLite.
- Fosite's `fosite.Request` can be reconstructed if we persist the request ID, request time, client, requested/granted scopes, request form, requested/granted audience, and the `openid.DefaultSession` fields used for ID Token generation and UserInfo.

### What was tricky to build

- `fosite.Requester` contains interfaces for `Client` and `Session`, so it cannot be safely JSON-marshaled directly and restored without type information. I implemented explicit `persistedRequest`, `persistedClient`, and `persistedSession` structs and convert them back to `fosite.Request`, `fosite.DefaultClient`, and `openid.DefaultSession`.
- Refresh-token rotation requires request-id based revocation because Fosite associates access/refresh tokens belonging to the same request. The SQL store mirrors Fosite's memory behavior by indexing access and refresh tokens by `request_id` and marking refresh tokens inactive on rotation.
- The adapter has to avoid a method-name collision with the project store's `GetClient(ctx,id) (domain.Client,error)`. I solved this with a wrapper type rather than trying to make `internal/store/sqlite.Store` directly implement Fosite's `GetClient(ctx,id) (fosite.Client,error)`.

### What warrants a second pair of eyes

- Review the explicit requester serialization format in `internal/fositeadapter/sqlstore.go`. It is intentionally narrow and supports the session/client types this adapter creates.
- Review whether token-family reuse handling should be stricter than Fosite's memory-store-compatible inactive-token behavior. The project domain refresh-token store has family revocation semantics; the Fosite SQL adapter currently mirrors Fosite's request-id revocation semantics.
- Review the `SQLDB()` exposure on the SQLite store. It is scoped and documented, but it is still a boundary decision.

### What should be done in the future

- Add migrations for the Fosite protocol tables into `internal/store/sqlite/migrations` instead of creating them from the adapter at provider startup.
- Add database transaction wrappers around multi-step revocation paths if multi-process SQLite use becomes a target.
- Add tests for refresh-token reuse after rotation with the SQL Fosite store.

### Code review instructions

- Start with `internal/fositeadapter/sqlstore.go` and compare method behavior to Fosite's `storage.MemoryStore`.
- Review `internal/fositeadapter/sqlstore_test.go` to understand the restart-durability guarantee.
- Validate with:

```bash
go test ./internal/fositeadapter -run TestFositeSQLiteStoreSurvivesProviderRestart -v
go test ./...
```

### Technical details

Key files:

```text
internal/fositeadapter/sqlstore.go
internal/fositeadapter/sqlstore_test.go
internal/fositeadapter/provider.go
internal/store/sqlite/store.go
```

## Step 10: Implement Phase 8 hardening foundation

I implemented the first hardening layer for the strict Fosite-backed engine. This pass focused on controls that belong directly on the current browser/protocol boundary: CSRF protection for login form POSTs, security headers on strict engine responses, no-store token response behavior, structured audit events, and a consent policy boundary.

The implementation avoids moving mock-only behavior into production. The mock server remains unchanged. The hardening code lives in the strict adapter and the embedded provider API, so callers can supply audit sinks and consent policies when embedding the provider.

### Prompt Context

**User prompt (verbatim):** "Ok, hardening."

**Assistant interpretation:** Proceed to Phase 8 and implement concrete security hardening rather than stopping at the Fosite/storage integration.

**Inferred user intent:** The user wants the production side to move toward secure-by-default behavior with browser-request protections, auditability, and policy hooks.

**Commit (code):** 60a94df3ba5da7fe989702023876c5ebdac41e40 — "Harden strict IdP browser and audit paths"

### What I did

- Added `internal/audit`:
  - `Event` with time, name, client ID, subject, request ID, result, reason, and safe structured fields.
  - `Sink` interface.
  - `NoopSink` default.
  - `MemorySink` for tests.
- Added strict login CSRF protection in `internal/fositeadapter/csrf.go`:
  - GET `/authorize` issues a signed CSRF token and `HttpOnly`, `SameSite=Lax` cookie.
  - POST `/authorize` validates the submitted token against the cookie and HMAC.
  - Successful authorization clears the CSRF cookie.
  - Cookie `Secure` follows adapter/embedded cookie configuration.
- Added security headers around the strict provider handler:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Referrer-Policy: no-referrer`
  - `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'`
- Added `Cache-Control: no-store` on login form rendering and token endpoint responses.
- Added audit emission for:
  - `authorize.request.rejected`
  - `authorize.request.accepted`
  - `login.csrf_rejected`
  - `login.failure`
  - `login.success`
  - `consent.required`
  - `consent.granted`
  - `token.request.rejected`
  - `token.request.accepted`
- Added `internal/fositeadapter/consent.go`:
  - `ConsentPolicy` interface.
  - `AlwaysSkipConsent` default.
  - `RememberConsent` in-memory implementation.
- Exposed `Audit` and `Consent` through `pkg/embeddedidp.Options` and passed them into the strict adapter.
- Updated tests to obtain CSRF tokens from GET `/authorize` before POSTing login.
- Added hardening tests for CSRF rejection/audit and security headers.

### Why

- Login form POSTs are browser-originating state-changing requests and need CSRF protection independent of OAuth `state`.
- Security headers reduce clickjacking, MIME-sniffing, referrer leakage, and unintended embedding of IdP pages.
- Token responses should explicitly avoid caching.
- Production identity flows need structured audit events without logging raw codes, tokens, passwords, or client secrets.
- Consent policy must be an explicit extension point before a real consent UI/storage flow is added.

### What worked

- Strict adapter hardening tests passed:

```bash
go test ./internal/fositeadapter -v
```

- Full suite passed:

```bash
go test ./...
```

### What didn't work

- Existing strict authorization tests initially posted directly to `/authorize`, which is exactly what CSRF protection should reject. I updated the tests to first GET `/authorize`, extract the hidden CSRF token and cookie, then POST with both values.

### What I learned

- Adding CSRF changed test shape in a useful way: tests now exercise the same browser sequence that a real login page uses.
- Keeping the audit sink as a small interface avoided introducing logging dependencies into protocol code and kept tests deterministic.

### What was tricky to build

- The strict engine does not yet have a durable IdP browser session layer. That means full multi-page consent UX is not safe to implement as a separate POST without adding authenticated server-side session state. I added the consent policy boundary and explicit approval field handling but kept `AlwaysSkipConsent` as the default until the session/consent UI is implemented properly.
- The CSRF cookie path must respect path-based issuers, so `cookiePath()` uses the issuer path and falls back to `/`.
- Audit events must avoid raw secrets. I emitted client IDs, subjects, request IDs, grant type, results, and reason categories, but not raw authorization codes, access tokens, refresh tokens, passwords, or client secrets.

### What warrants a second pair of eyes

- Review the CSP. It is intentionally strict and may need nonce-based relaxation when real templates include styles or scripts.
- Review the consent boundary. It is not a full consent UX yet because that requires persistent IdP browser sessions.
- Review audit reason strings. Some currently use error text from Fosite; if those strings are too verbose for production logs, map them to stable reason codes.

### What should be done in the future

- Add server-side IdP sessions with secure cookies and persistent session hashes.
- Add a real consent page backed by server-side authenticated session state.
- Add rate-limiting hooks for login and token endpoints.
- Add audit event documentation and log redaction tests.

### Code review instructions

- Start with `internal/fositeadapter/csrf.go` for the browser POST protection.
- Review `internal/fositeadapter/provider.go` around `authorize`, `token`, and `securityHeaders`.
- Review `internal/audit/audit.go` for event shape and test sink.
- Validate with:

```bash
go test ./internal/fositeadapter -v
go test ./...
```

### Technical details

Key files:

```text
internal/audit/audit.go
internal/fositeadapter/csrf.go
internal/fositeadapter/consent.go
internal/fositeadapter/provider.go
internal/fositeadapter/hardening_test.go
pkg/embeddedidp/options.go
pkg/embeddedidp/provider.go
```

## Step 11: Add server-side IdP sessions and rate-limiting hooks

I continued hardening by adding the server-side IdP browser session layer that the previous CSRF/consent pass deliberately left as future work. The strict provider can now create an opaque browser session after login, store only a keyed hash of the session handle, silently issue authorization codes for later authorize requests with a valid session, and return `login_required` for `prompt=none` requests when no valid session is present.

I also added a rate-limiting abstraction and a fixed-window implementation. The default remains allow-all so existing tests and embeddings are not broken, but production callers can now wire a limiter through `embeddedidp.Options` and the strict adapter enforces it on login and token paths.

### Prompt Context

**User prompt (verbatim):** (same as Step 10)

**Assistant interpretation:** Continue production hardening beyond the first browser controls and avoid leaving the provider without sessions or rate-limiting hooks.

**Inferred user intent:** The user wants the production side to be materially closer to a real embeddable IdP, not just a protocol/token service.

**Commit (code):** 958232d3e68b2e43fb65479d2ab5703276927890 — "Add strict IdP sessions and rate limiting hooks"

### What I did

- Added `internal/fositeadapter/session.go`:
  - `tinyidp_session` opaque cookie.
  - keyed HMAC hash storage using the existing `domain.Session` record.
  - server-side session creation after login.
  - session lookup, expiry, revocation, and disabled-user checks.
  - issuer-path-scoped, `HttpOnly`, `SameSite=Lax`, optionally `Secure` cookie settings.
- Updated strict `/authorize` behavior:
  - GET with valid session and no `prompt=login` can silently issue an authorization code.
  - GET with `prompt=none` and no session redirects with `login_required` via Fosite error handling.
  - GET with valid session but required consent renders a continuation form instead of asking for credentials again.
  - POST can continue from an existing session without a login field, which is needed for consent continuation.
  - POST login creates a server-side IdP session before completing authorization.
- Added `internal/fositeadapter/session_test.go`:
  - verifies session cookie issuance;
  - verifies silent authorization code issuance from an existing session;
  - verifies `prompt=none` without a session returns `login_required`.
- Added `internal/fositeadapter/ratelimit.go`:
  - `RateLimiter` interface;
  - `AllowAllRateLimiter` default;
  - `FixedWindowRateLimiter` implementation.
- Wired `RateLimiter` through `fositeadapter.Options` and `embeddedidp.Options`.
- Enforced rate limiting on login POST and token POST paths with audit events for rejections.

### Why

- The IdP session is distinct from RP application sessions and is required for OIDC semantics such as silent authorization, `prompt=none`, `prompt=login`, and consent continuation.
- Rate-limiting needs to be a hook because embedded deployments may use in-process, reverse-proxy, Redis, or host-application limiters.

### What worked

- Focused session test passed:

```bash
go test ./internal/fositeadapter -run TestBrowserSessionSilentAuthorizeAndPromptNone -v
```

- Full suite passed:

```bash
go test ./...
```

### What didn't work

- The first session test failed because the test secret was 31 bytes, and Fosite requires a 32-byte HMAC secret. I fixed the test secret length.
- The session test initially exposed the generic Fosite `server_error` message; enabling debug responses only for dev mode made local test failures diagnosable without enabling debug messages in production mode.

### What I learned

- Fosite's OIDC prompt/max_age validation depends on `auth_time`, so the strict provider needs to preserve the original browser-session authentication time and pass it into the OIDC session when silently authorizing.
- Consent continuation needs an authenticated browser session; otherwise the provider cannot know which user is approving scopes without asking for credentials again.

### What was tricky to build

- The authorize handler now has three distinct branches: unauthenticated login, authenticated silent authorization, and authenticated consent continuation. Keeping those branches explicit made it easier to avoid accidentally bypassing consent or CSRF.
- Session cookies need to share the same path logic as CSRF cookies so path-based issuers work correctly.

### What warrants a second pair of eyes

- Review the authorize flow for prompt semantics. `prompt=login` currently forces the login form despite an existing session; `prompt=none` without a session uses Fosite's `login_required` error.
- Review the rate-limit keys. They currently use path category plus `RemoteAddr` and client ID for token requests. Deployments behind proxies may need trusted-proxy normalization.

### What should be done in the future

- Persist consent records instead of using only in-memory `RememberConsent`.
- Add trusted proxy handling for remote address normalization.
- Add tests for `prompt=login` and `max_age` once the session UX is expanded.

### Code review instructions

- Start with `internal/fositeadapter/session.go`.
- Then review the GET/POST branches in `internal/fositeadapter/provider.go`.
- Review `internal/fositeadapter/session_test.go` for session behavior.
- Review `internal/fositeadapter/ratelimit.go` and decide how production deployments should supply limiter keys.

### Technical details

Key files:

```text
internal/fositeadapter/session.go
internal/fositeadapter/ratelimit.go
internal/fositeadapter/provider.go
internal/fositeadapter/session_test.go
pkg/embeddedidp/options.go
pkg/embeddedidp/provider.go
```

## Step 12: Persist consent and add refresh-token reuse coverage

I continued the hardening work by replacing the remaining in-memory-only consent path with a real domain/storage concept. Production-mode strict providers now default to a stored consent policy, which means user approvals can survive process restarts when the embedding application uses the SQLite store.

I also added a Fosite-level SQLite refresh-token reuse regression test. The lower-level store suite already covered refresh reuse detection, but this new test proves the actual strict token endpoint rejects reuse after rotation when Fosite is backed by the durable SQLite protocol store.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the production-readiness implementation from the previous hardening increment without stopping at session/rate-limit scaffolding.

**Inferred user intent:** The user wants the remaining production gaps to be closed incrementally, with validation, docs, and commits after coherent slices.

**Commit (code):** 7167fd4f548e5b61480d4df35ef99758fd0756a9 — "Persist strict IdP consent grants"

### What I did

- Added `domain.Consent` with user ID, client ID, normalized scopes, grant time, optional expiry, and optional revocation time.
- Added `storage.ConsentStore` to the shared store contract.
- Implemented consent persistence in:
  - `internal/store/memory.Store`;
  - `internal/store/sqlite.Store`;
  - SQLite schema migration DDL.
- Added `domain.NormalizeScopes` so consent keys are stable across scope ordering and duplicate input.
- Added `fositeadapter.StoredConsent` and `NewStoredConsent`.
- Changed `fositeadapter.NewProvider` so production mode defaults to stored consent when no custom `ConsentPolicy` is supplied; development mode keeps the previous skip-consent default.
- Added consent tests for:
  - normalized stored-consent approval;
  - revocation requiring consent again;
  - production-mode default consent requirement.
- Added `TestFositeSQLiteRefreshTokenReuseIsRejected`, which exercises refresh-token rotation through the real `/token` endpoint and confirms old refresh-token reuse fails.

### Why

- Consent is security-relevant authorization state and cannot remain process-local for production deployments.
- Production defaults should fail toward explicit user approval, while development defaults can remain convenient for local testing.
- Refresh-token reuse is one of the most important durable-state correctness properties because reuse can indicate token theft.

### What worked

- Focused tests passed:

```bash
go test ./internal/fositeadapter -run 'TestStoredConsent|TestProductionProviderDefaults' -v
go test ./internal/fositeadapter -run 'TestFositeSQLiteRefreshTokenReuseIsRejected|TestFositeSQLiteStoreSurvivesProviderRestart' -v
```

- Full suite passed:

```bash
go test ./...
```

### What didn't work

- The first refresh-token reuse test attempt returned a 404 HTML page while fetching CSRF. The cause was that the helper requested `profile email` scopes but the test client initially allowed only `openid offline_access`; Fosite redirected to the callback with an error, and the default client followed the redirect. I fixed the client fixture to allow the same scopes used by the helper.

### What I learned

- Scope normalization belongs in the domain layer, not just the consent policy, because both memory and SQLite stores need the same key semantics.
- The production default can be stricter without breaking existing tests by keying the behavior on `domain.ProductionMode`; dev mode remains unchanged.

### What was tricky to build

- Consent records need stable keys even when requests present scopes in different orders or with duplicates. The solution was `domain.NormalizeScopes`, used both before storing records and before lookup/revocation.
- The refresh-token reuse endpoint test depended on the shared authorization helper, so the client fixture had to match that helper's requested scopes exactly.

### What warrants a second pair of eyes

- Review whether consent lookup should accept prior consent for a superset of scopes. This implementation requires an exact normalized scope set, which is strict and simple but can prompt again for smaller/larger variants.
- Review migration strategy before long-lived deployments. The current SQLite migration file uses `CREATE TABLE IF NOT EXISTS`, which is acceptable for current development but should evolve into numbered additive migrations.

### What should be done in the future

- Add consent expiry policy configuration to the embedded API.
- Add a consent-management/revocation API for embedding applications.
- Move Fosite SQL table creation out of adapter startup DDL and into migrations.

### Code review instructions

- Start with `internal/domain/types.go` and `internal/domain/scopes.go`.
- Then review `internal/storage/interfaces.go` and the memory/SQLite store implementations.
- Review `internal/fositeadapter/consent.go` for policy semantics and default production behavior in `provider.go`.
- Validate with `go test ./...`.

### Technical details

Key files:

```text
internal/domain/types.go
internal/domain/scopes.go
internal/storage/interfaces.go
internal/store/memory/store.go
internal/store/sqlite/store.go
internal/store/sqlite/migrations/001_schema.sql
internal/fositeadapter/consent.go
internal/fositeadapter/consent_test.go
internal/fositeadapter/sqlstore_test.go
```

## Step 13: Close the remaining strict-engine hardening loop

I treated the user's instruction to “do it all” as permission to finish the remaining production-readiness items that had been repeatedly called out as next steps. This pass focused on things that make the strict engine reviewable and operable: stable audit reason codes, schema ownership, key rotation, ID Token/JWKS validation, and runbooks.

The result is not a claim of external OpenID Foundation certification, but it does turn the remaining local gaps into code, tests, and explicit operational documentation. The strict engine now has a CI-friendly conformance script plus manual instructions for the external suite.

### Prompt Context

**User prompt (verbatim):** "do it all."

**Assistant interpretation:** Complete the remaining hardening targets rather than stopping after one more small increment.

**Inferred user intent:** The user wants the strict production engine to be as close as practical to production-ready in this session, with tests, docs, and bookkeeping updated.

**Commit (code):** 8005ed3f2df2f12b3f93be91e298098cc6bf41e6 — "Complete strict IdP hardening loop"

### What I did

- Added audit reason normalization:
  - `internal/fositeadapter/audit_reason.go` maps Fosite RFC6749 errors to stable snake-case reason codes.
  - strict provider token/authorize audit paths now use normalized reasons instead of raw error text.
  - added `TestAuditReasonsUseStableCodes`.
- Moved Fosite SQLite schema ownership:
  - Fosite protocol tables and indexes now live in `internal/store/sqlite/migrations/001_schema.sql`.
  - `internal/fositeadapter/sqlstore.go` no longer runs adapter-owned startup DDL.
- Added signing-key rotation support:
  - `internal/keys.RotateRSA` creates a new key, activates it, and retires the previous active key while leaving it in `VerificationKeys`.
  - memory and SQLite tests verify old/new key visibility and durable retired-key retention.
- Strengthened ID Token validation coverage:
  - strict authorization-code flow test now verifies the ID Token signature against JWKS.
  - the test validates `kid`, `alg`, issuer, audience, nonce, and numeric expiry.
  - strict OIDC sessions now set the active signing key ID as the JWT `kid` header.
- Added production runbooks:
  - `docs/security-profile.md`
  - `docs/storage.md`
  - `docs/key-rotation.md`
  - `docs/conformance.md`
  - `scripts/run-conformance.sh`

### Why

- Stable audit reason codes are necessary for alerting, dashboards, and tests; raw Fosite error strings are too verbose and version-sensitive.
- Schema belongs to the store/migration layer, not adapter construction.
- Key rotation must retain old verification keys long enough for relying parties to validate already-issued ID Tokens.
- A signed ID Token is not enough; relying parties validate by selecting the JWKS key matching the `kid` header.
- Production readiness needs written operational gates, not only code.

### What worked

- Focused tests passed:

```bash
go test ./internal/keys ./internal/store/sqlite ./internal/fositeadapter
```

- The local conformance script passed:

```bash
scripts/run-conformance.sh
```

### What didn't work

- The first ID Token JWKS validation test failed because Fosite emitted a JWT header without `kid`:

```text
bad token header: map[string]interface {}{"alg":"RS256", "typ":"JWT"}
```

  I fixed this by adding the active signing key ID to the OIDC session headers in `newOIDCSession`.

- The next ID Token validation attempt assumed `aud` was a string. Fosite emits `aud` as an array for this flow:

```text
"aud": []interface {}{"spa"}
```

  I fixed the test helper to accept both string and array audience encodings.

### What I learned

- Fosite's signer does not automatically infer a `kid` from the private key returned by the key getter; the adapter must set the JWT header explicitly.
- Keeping Fosite schema in SQLite migrations makes the durable store much easier to reason about because all tables are visible in one place.

### What was tricky to build

- Key rotation had to avoid a moment where the old key disappeared from JWKS. The helper creates the new key, activates it, then retires the old key so `VerificationKeys` includes both the active new key and the retired old key.
- ID Token validation needed to avoid overfitting to one JSON representation of `aud`; OIDC consumers commonly need to handle both string and array forms.

### What warrants a second pair of eyes

- Review the key-retention model. Retired keys currently remain visible through `VerificationKeys`; a future cleanup job should remove them only after token lifetime plus skew.
- Review whether `internal/keys.RotateRSA` should be exposed through a public embedded API command or left as an internal helper for now.
- Review the conformance runbook against the exact OpenID Foundation test plan selected for certification.

### What should be done in the future

- Run the external OpenID Foundation conformance suite against a public HTTPS deployment and store the export in the ticket source bundle.
- Add a cleanup job for expired protocol rows, sessions, consents, and old retired keys.
- Add explicit public API affordances for key rotation and consent revocation if external embedders need them outside this module.

### Code review instructions

- Start with `internal/fositeadapter/audit_reason.go` and the modified audit calls in `provider.go`.
- Review `internal/store/sqlite/migrations/001_schema.sql` against `internal/fositeadapter/sqlstore.go`.
- Review `internal/keys/rotation.go` and the memory/SQLite rotation tests.
- Review the ID Token/JWKS helper in `internal/fositeadapter/provider_test.go`.
- Run `scripts/run-conformance.sh`.

### Technical details

Key files:

```text
internal/fositeadapter/audit_reason.go
internal/fositeadapter/provider.go
internal/fositeadapter/hardening_test.go
internal/fositeadapter/provider_test.go
internal/fositeadapter/sqlstore.go
internal/store/sqlite/migrations/001_schema.sql
internal/keys/rotation.go
internal/keys/rotation_test.go
internal/store/sqlite/store_test.go
docs/security-profile.md
docs/storage.md
docs/key-rotation.md
docs/conformance.md
scripts/run-conformance.sh
```

## Step 14: Add Python automation for the hosted OIDF suite

The hosted OpenID Foundation suite still needs a real browser-like actor even when the test lifecycle is API-driven. I added a Python runner that uses the authenticated suite session cookie for API access, starts or resumes plan modules, follows exported authorization URLs, submits the tiny-idp login and consent form, and saves hosted-suite JSON evidence as it polls.

The important implementation detail is that this is hybrid automation, not a fake protocol shortcut. The script lets the suite generate each test's authorization request and then drives the same redirects a user agent would drive, including the suite's callback-page implicit POST that normally runs in JavaScript.

### Prompt Context

**User prompt (verbatim):** "automate using python."

**Assistant interpretation:** Implement a Python-based automation path for interacting with the hosted OpenID conformance suite instead of continuing manual browser clicks.

**Inferred user intent:** The user wants the Basic OP hosted-suite run to be repeatable, evidence-producing, and less dependent on manual Playwright operation.

**Commit (code):** d1ce6d8cedd1186eeff5c7010f6af942e69e1943 — "Automate hosted OIDF conformance runner"

### What I did

- Added `scripts/oidf_hosted_runner.py`.
- The script authenticates to the hosted suite API with `OIDF_JSESSIONID` or `--cookie`.
- It supports selecting modules with `--only`, running unfinished modules with `--remaining`, polling existing instances with `--resume`, and writing `/api/info` plus `/api/log` artifacts with `--artifacts`.
- It uses Python `requests` to follow suite-exported browser actions through tiny-idp's authorization endpoint.
- It parses the tiny-idp login/consent HTML form, fills `login=alice`, preserves hidden request fields and CSRF, and posts the form back to `/authorize`.
- It reproduces the suite callback page's JavaScript implicit-submission POST so the suite continues processing after redirects.
- Updated `docs/conformance.md` with hosted-runner usage.
- Smoke-tested the runner against the live hosted plan:
  - `oidcc-userinfo-get` eventually passed with test id `lgPKQCYB0mrr2r5` after fixing the implicit-submit handling.
  - `oidcc-userinfo-post-header` passed with test id `gBXryAAXqlfNv7F`.
  - A later `--remaining` run advanced additional modules: userinfo POST body and nonce-less code flow passed; optional scope modules produced `WARNING` or `SKIPPED` results; display page/popup passed; `oidcc-prompt-login` stopped at manual screenshot review as expected.

### Why

- The suite API can create tests and expose browser URLs, but tests do not progress unless those browser URLs are visited and callbacks are submitted.
- Using Python keeps the automation script inspectable and easy to run in a shell with a copied session cookie.
- Saving JSON artifacts gives the ticket reproducible evidence without requiring a full manual suite export after each module.

### What worked

- Authenticated API calls with the browser's active `JSESSIONID` worked for:
  - `/api/currentuser`
  - `/api/plan/Ko612tVrEHxTT`
  - `/api/runner`
  - `/api/info/<test-id>`
  - `/api/log/<test-id>`
- The runner successfully drove tiny-idp login/consent via plain HTTP form submission.
- The fixed implicit-submit parser posted to the suite's `/implicit/<id>` URL and allowed hosted tests to finish.
- Validation command passed:

```bash
python3 -m py_compile scripts/oidf_hosted_runner.py
```

### What didn't work

- The first `oidcc-userinfo-get` automation attempt timed out because the script followed the authorization callback but did not reproduce the suite page's JavaScript POST to the generated implicit-submit URL:

```text
FAILED/STOP: oidcc-userinfo-get Uwrwv4HOwkvMgsQ status=TIMEOUT result=None
```

- A second attempt detected an implicit URL but submitted a malformed escaped URL:

```text
[browser] implicit submit POST https://www.certification.openid.net/test/a/tinyidp-basic-20260708/\/\/www.certification.openid.net\/test\/a\/tinyidp-basic-20260708\/implicit\/FTmYg7weTVRpCHhA1vLZ
[browser] browser response 400 ...
FAILED/STOP: oidcc-userinfo-get lgPKQCYB0mrr2r5 status=TIMEOUT result=None
```

- I fixed this by sharing the suite `JSESSIONID` with the browser session and normalizing JavaScript-escaped slashes (`\/`) after regex extraction.
- The first `--remaining` run stopped on `oidcc-scope-address` because the suite returned `SKIPPED` for an unsupported optional scope. I updated the runner to treat `SKIPPED` as non-fatal while still preserving the artifact for review.
- The `oidcc-prompt-login` module reached a suite `REVIEW` checkpoint requiring a screenshot upload:

```text
manual review required: The server must ask the user to login for a second time; a screenshot of this must be uploaded.
```

  I updated the runner to stop immediately when no undriven browser actions remain and the log contains a review checkpoint.

### What I learned

- The hosted suite separates API control from browser continuation: `/api/runner/<id>` exposes URLs, but the callback page still expects a browser-side POST to `/implicit/<id>`.
- The suite's callback processing page may embed absolute URLs with JavaScript-escaped slashes, so URL extraction must unescape before `urljoin`.
- The browser automation session should carry the same suite cookie as the API session to avoid being treated as an unauthenticated user when interacting with suite callback pages.

### What was tricky to build

- The subtle part was that a successful HTTP 200 from the callback page did not mean the test had progressed. The callback page was an intermediate HTML page whose JavaScript performs the actual continuation POST. The symptom was an indefinitely `WAITING` test even though tiny-idp had redirected back with a code. The solution was to parse the callback HTML, extract the generated implicit-submit URL, unescape it correctly, and POST to it with the suite cookie.
- The script also needs to be conservative around manual review states. It currently stops when the suite reports review and there are no actionable browser URLs, instead of pretending the certification decision is automatic.

### What warrants a second pair of eyes

- Review `pending_browser_actions` against future suite modules that may use `browserApiRequests` or POST browser actions with structured bodies.
- Review whether artifacts under `sources/oidf-hosted-python` should be committed wholesale or periodically pruned to only final passing instances.
- Review whether `WARNING` should be treated as acceptable for all certification phases or only for exploratory hosted runs.

### What should be done in the future

- Continue the hosted Basic OP plan with `--remaining` and capture final passing artifacts.
- Add a small helper to extract `JSESSIONID` automatically from the local browser profile if that proves reliable and safe.
- Add an optional public/plan sharing step to capture read-only links for reviewers.

### Code review instructions

- Start with `scripts/oidf_hosted_runner.py`, especially `SuiteClient`, `BrowserDriver`, `_find_implicit_submit_url`, and `run_one`.
- Review `docs/conformance.md` for the intended operator workflow.
- Validate with:

```bash
python3 -m py_compile scripts/oidf_hosted_runner.py
scripts/oidf_hosted_runner.py --plan Ko612tVrEHxTT --cookie 'JSESSIONID=<value>' --remaining --dry-run
```

### Technical details

Key files and artifacts:

```text
scripts/oidf_hosted_runner.py
docs/conformance.md
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python/lgPKQCYB0mrr2r5.info.json
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python/lgPKQCYB0mrr2r5.log.json
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python/gBXryAAXqlfNv7F.info.json
ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/oidf-hosted-python/gBXryAAXqlfNv7F.log.json
```

## Step 15: Continue hosted OIDF run and fix prompt/max-age/request-object edge cases

The Python runner progressed the hosted Basic OP plan through the remaining browser-heavy modules. During that run, the suite exposed three strict-engine gaps: `prompt=none` with an existing session needed Fosite-compatible `RequestedAt`, expired `max_age` needed to force a fresh login page instead of returning `login_required`, and unsupported unsigned request objects needed browser-visible redirect/error-page behavior instead of a bare JSON response.

I fixed those behaviors and reran the affected hosted modules. The plan is now down to one known configuration blocker: the refresh-token module expects distinct `client` and `client2`, but the existing hosted plan was created with both set to `web-app`/`dev-secret`, so the cross-client refresh-token misuse check cannot fail as intended on that plan.

### Prompt Context

**User prompt (verbatim):** "ok continue"

**Assistant interpretation:** Continue the hosted OIDF suite run, using the automation and fixing blockers as they appear.

**Inferred user intent:** The user wants the certification run advanced as far as possible and wants implementation fixes committed when they are clearly needed.

**Commit (code):** 4a0179405938d7ffc13b7559b2d51c7690d4e46c — "Fix strict OIDC prompt and request object edges"

### What I did

- Continued `scripts/oidf_hosted_runner.py --remaining` against plan `Ko612tVrEHxTT`.
- Uploaded required review screenshots for:
  - `oidcc-prompt-login` (`JL0IqvBrBLRVbpB`)
  - `oidcc-max-age-1` (`oSwhtwEwi1ieXCm`)
  - `oidcc-ensure-registered-redirect-uri` (`xZC6nQZBEr1OeMR`)
  - `oidcc-ensure-request-object-with-redirect-uri` (`9YRMjvJqTC37icW`)
- Fixed prompt/max-age session handling:
  - `prompt=none` with a valid existing session now sets OIDC `RequestedAt` so Fosite accepts silent reuse.
  - expired `max_age` now renders a login page instead of silently reusing the browser session or returning an immediate authorization error.
- Fixed unsupported request-object handling:
  - valid redirect URI receives an authorization error redirect with `request_not_supported` and `state`.
  - invalid query `redirect_uri` produces a local error page, satisfying redirect URI safety tests.
- Added regression tests for prompt-none session reuse, expired max-age login prompt, and request-object unsupported redirects.

### Why

- OIDC prompt and max-age semantics are not only token-claim details; they affect whether the OP may silently reuse an existing browser session.
- Fosite validates `prompt=none`, `prompt=login`, and `max_age` against `AuthTime` and `RequestedAt`; our strict adapter must set those values consistently when reusing or refreshing sessions.
- Unsupported request objects must still be handled safely: redirect only to an allowed redirect URI, preserve `state`, and never fall back to an unregistered/default redirect URI.

### What worked

- Local validation passed:

```bash
go test ./...
scripts/run-conformance.sh
docmgr doctor --ticket TINYIDP-PROD-001 --stale-after 30
```

- Hosted modules fixed or advanced after changes:
  - `oidcc-prompt-none-logged-in`: `MtW9l7G0x25fJ7w` passed.
  - `oidcc-max-age-1`: `oSwhtwEwi1ieXCm` finished `REVIEW` after screenshot upload.
  - `oidcc-unsigned-request-object-supported-correctly-or-rejected-as-unsupported`: `W6bLCRwlS1YePL3` finished `SKIPPED`.
  - `oidcc-ensure-request-object-with-redirect-uri`: `9YRMjvJqTC37icW` finished `REVIEW` after screenshot upload.
  - `oidcc-ensure-request-with-valid-pkce-succeeds`: `E8VgG05dfIpf1Fb` passed.

### What didn't work

- `oidcc-refresh-token` failed on this existing plan because both configured suite clients are the same logical client:

```text
ValidateErrorFromTokenEndpointResponseError FAILURE The authorization server was expected to return an error, but the 'error' field in the response is either null or empty
```

  The suite attempted to use a refresh token “issued to client 2” with “client 1”, but this plan has `client` and `client2` both configured as `web-app` with `dev-secret`, so the server correctly sees the same client.

### What I learned

- The hosted Basic OP suite's static-client profile still needs distinct client credentials for the refresh-token cross-client misuse module.
- Some OIDF `REVIEW` outcomes are successful terminal outcomes after screenshot upload, not code failures.
- Request-object tests can be satisfied by an explicit `request_not_supported` redirect when the redirect URI is valid; invalid redirect URI cases must stay on a local error page.

### What was tricky to build

- `max_age` was tricky because Fosite returns `login_required` during request creation when the current session no longer satisfies max-age. The adapter must detect this case and render the login page for interactive requests, while still returning `login_required` for `prompt=none`.
- The request-object redirect-uri case was subtle: the unsigned request object contained a valid redirect URI in the JWT payload, while the outer query had an invalid redirect URI. Redirecting to the payload URI looked convenient but violated the suite's “do not use a default redirect URI” safety check.

### What warrants a second pair of eyes

- Review the `rejectUnsupportedRequestObject` helper to ensure it rejects request objects narrowly and safely without accidentally advertising request-object support.
- Review whether the strict CLI should accept multiple explicit client definitions so future hosted plans can configure distinct `client` and `client2` without code changes.

### What should be done in the future

- Create a fresh hosted Basic OP plan with distinct static clients, e.g. `web-app` and `web-app-2`, and start the strict server with both clients registered.
- Decide whether to retain review screenshots in the repo or only keep sanitized metadata/log summaries.

### Code review instructions

- Review `internal/fositeadapter/provider.go` around `authorize`, `newOIDCSession`, and `rejectUnsupportedRequestObject`.
- Review `internal/fositeadapter/session.go` for `sessionSatisfiesMaxAge`.
- Review `internal/fositeadapter/session_test.go` and `internal/fositeadapter/provider_test.go` for regression coverage.
- Validate with `go test ./...` and `scripts/run-conformance.sh`.

### Technical details

Key files:

```text
internal/fositeadapter/provider.go
internal/fositeadapter/session.go
internal/fositeadapter/session_test.go
internal/fositeadapter/provider_test.go
```

## Step 16: Add extra strict CLI clients and pass hosted refresh-token cross-client test

The remaining hosted refresh-token failure was not a Fosite storage bug. The existing hosted plan configured both `client` and `client2` as the same static client, so the suite's “use client 2 refresh token with client 1” misuse check could not fail. I added a small strict-CLI affordance for extra static clients, created a fresh hosted plan with distinct clients, restarted the strict server with `web-app-2`, and reran the refresh-token module successfully.

This keeps the production API direction clean: the embeddable provider already accepts a store with arbitrary clients; the new CLI flag is only a development/conformance convenience for registering additional clients without writing a custom storage bootstrapper.

### Prompt Context

**User prompt (verbatim):** "go ahead"

**Assistant interpretation:** Proceed with the next concrete step to resolve the refresh-token hosted-suite blocker.

**Inferred user intent:** The user wants the conformance run completed rather than stopping at the known client-configuration limitation.

### What I did

- Added `--extra-clients` to the OIDC serve section.
- Added strict CLI parsing for extra clients in the form `client-id|secret|redirect-uri[|redirect-uri...]`.
- Added registry test coverage for extra-client registration.
- Updated `docs/conformance.md` with the distinct-client hosted-suite command pattern.
- Created hosted plan `Geeb9MBn659ah` with alias `tinyidp-basic-20260708b` and distinct static clients:
  - `client`: `web-app` / `dev-secret`
  - `client2`: `web-app-2` / `dev-secret-2`
- Restarted strict CLI with:

```bash
--extra-clients "web-app-2|dev-secret-2|https://www.certification.openid.net/test/a/tinyidp-basic-20260708b/callback|https://www.certification.openid.net/test/a/tinyidp-basic-20260708b/callback?dummy1=lorem&dummy2=ipsum"
```

- Ran the hosted refresh-token module against the new plan.

### Why

- The suite's refresh-token test intentionally checks client binding by using a refresh token issued to `client2` at `client1`'s token endpoint authentication.
- A plan where both clients are the same client cannot validate that behavior.
- The CLI needed a way to register a second confidential client with the suite's generated redirect URI.

### What worked

- Local tests for command/client registry changes passed.
- Hosted refresh-token module passed on the distinct-client plan:

```text
oidcc-refresh-token: s6Wy9BgOnvhsEG5 FINISHED PASSED
```

### What didn't work

- The original plan `Ko612tVrEHxTT` remains unable to prove refresh-token cross-client rejection because its `client` and `client2` config are identical. I left that plan's raw artifacts uncommitted because they may contain transient protocol tokens/codes.

### What I learned

- The hosted suite configuration is part of the test's correctness. Passing the same client twice can make a server look wrong even when its refresh-token binding logic is correct.
- A small CLI-only multi-client affordance is enough for hosted certification without changing the embeddable provider's storage abstractions.

### What was tricky to build

- The extra-client flag needed a shell-friendly syntax that preserves URLs containing `://` and query strings. A pipe-separated format avoids splitting on URL colons.
- The hosted suite appends alias-specific callback paths, so the strict server must be restarted with redirect URIs that match the newly created plan alias.

### What warrants a second pair of eyes

- Review whether `--extra-clients` should remain a simple conformance/dev flag or grow into structured config-file support with validation errors for malformed specs.
- Review whether temporary hosted plan `Geeb9MBn659ah` should become the canonical Basic OP plan for final evidence, or whether a fresh final plan should be created after all code is committed.

### What should be done in the future

- Run a fresh full Basic OP plan using the distinct-client setup so every module belongs to one final plan/export.
- Sanitize and summarize hosted artifacts before committing any evidence bundle.

### Code review instructions

- Review `internal/sections/oidc/section.go` and `settings.go` for the new flag.
- Review `internal/cmds/serve.go` for `parseExtraClientSpec` and registry registration.
- Review `internal/cmds/serve_test.go` for the extra-client regression.

### Technical details

Key files:

```text
internal/sections/oidc/section.go
internal/sections/oidc/settings.go
internal/cmds/serve.go
internal/cmds/serve_test.go
docs/conformance.md
```
