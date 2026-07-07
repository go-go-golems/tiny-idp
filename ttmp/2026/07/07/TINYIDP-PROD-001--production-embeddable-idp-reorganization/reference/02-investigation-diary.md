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
