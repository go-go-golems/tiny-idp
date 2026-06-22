# Implementation Diary

## Goal

Capture the step-by-step implementation of the mock OIDC IdP (ticket `MOCK-OIDC-IDP`), from ticket setup through Phase 0–4 implementation, including failures and sharp edges.

## Step 1: Ticket setup, design doc, and phased task breakdown

This step established the documentation foundation for the mock OIDC IdP work. The goal was to have a single source of truth (design doc), an executable checklist (phases/tasks), and a live diary before writing any code. No Go code was written yet; this step is entirely documentation and ticket bookkeeping.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to create a mock IDP for testing. Here's the research I've done. [...] Keep a detailed diary as you work. Create a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet points and pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and then upload to remarkable."

**Assistant interpretation:** Create a docmgr ticket `MOCK-OIDC-IDP`, write an intern-ready design/implementation guide from the provided research (minimal Go OIDC IdP replacing Keycloak for local testing), keep a diary, and upload to reMarkable.

**Inferred user intent:** Have a well-documented, phased plan and design for building a mock OIDC IdP that replaces Keycloak-in-Docker for local dev/testing, with the research consolidated into a reviewable artifact.

### What I did
- Loaded skill references: `ticket-research-docmgr-remarkable/references/writing-style.md`, `deliverable-checklist.md`, `diary/references/diary.md`.
- Confirmed `docmgr status --summary-only`: fresh workspace, 0 tickets.
- Confirmed `remarquee status` (ok) and `remarquee cloud account --non-interactive` (user=wesen@ruinwesen.com).
- Created ticket `MOCK-OIDC-IDP` (title: "Mock OIDC Identity Provider for local testing (Keycloak replacement)", topics: oidc,go,testing,identity,auth).
- Added design doc `design-doc/01-mock-oidc-idp-design-and-implementation-guide.md` (intern-ready: exec summary, problem/scope, OIDC background, current state, gap analysis, architecture, API reference, decision records, pseudocode, phased plan, test strategy, risks).
- Added reference doc `reference/02-implementation-phases-and-tasks.md` (checkbox per-task breakdown for Phases 0–12, MVP cutoff at Phase 4).
- Populated `tasks.md` with the MVP task checklist.

### Why
- The design doc is the "why and what"; the phases doc is the "when and how". Keeping them separate makes the design doc stable while tasks churn.
- Writing the intern guide before coding forces explicit decisions (scenario model, RS256-only, deterministic sub, loopback binding) that would otherwise be re-litigated during implementation.

### What worked
- docmgr created the full ticket workspace (9 dirs, 4 files) in one command.
- reMarkable account was already authenticated.
- Design doc consolidated the user's research (single-file reference impl + multi-user + failure modes + roadmap) into a coherent architecture with decision records.

### What didn't work
- Nothing failed. One minor surprise: docmgr created the design doc under `design-doc/` (not `design/`); confirmed via `docmgr doc list`.

### What I learned
- The vocabulary had no OIDC/auth/identity topics, but docmgr accepted the ticket topics anyway (they're free-form at ticket creation; vocab is checked by `doctor`). Will run `doctor` before upload.
- The research already contains a complete baseline implementation; Phase 0 is mostly transcribing it into the `cmd/` + `internal/` layout, not designing from scratch.

### What was tricky to build
- None yet (documentation only). The main subtlety carried forward: `sub` must be deterministic *and* not equal to the raw login, so `sub = "user-" + b64(sha256("tinyidp:user:"+login)[:16])`.

### What warrants a second pair of eyes
- The phased MVP cutoff (0–4) vs. deferred (5–12). Confirm multiple-clients/sessions/refresh are genuinely acceptable to defer for a first usable release.

### What should be done in the future
- Add `oidc`, `identity`, `auth`, `testing` to docmgr vocabulary if `doctor` flags them before reMarkable upload.

### Code review instructions
- Read `design-doc/01-mock-oidc-idp-design-and-implementation-guide.md` end to end.
- Cross-check §6.4 scenario table and §9 phases against `reference/02-implementation-phases-and-tasks.md`.
- Validate frontmatter: `docmgr validate frontmatter --doc <design-doc-path> --suggest-fixes`.

### Technical details
- Ticket path: `ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/`
- Design doc: 46 KB; phases doc: ~8 KB.
