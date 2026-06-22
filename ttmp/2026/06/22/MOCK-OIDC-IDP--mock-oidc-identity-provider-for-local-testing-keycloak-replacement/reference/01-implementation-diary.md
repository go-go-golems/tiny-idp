---
Title: Implementation Diary
Ticket: MOCK-OIDC-IDP
Status: active
Topics:
    - oidc
    - go
    - testing
    - identity
    - auth
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp/main.go
      Note: Phase 0 implementation (commit d473d513)
    - Path: cmd/tinyidp/main_test.go
      Note: httptest flow + ID token signature verification (commit d473d513)
ExternalSources: []
Summary: 'Chronological implementation diary for the mock OIDC IdP (ticket MOCK-OIDC-IDP): steps, failures, sharp edges, review notes.'
LastUpdated: 2026-06-22T15:10:00-04:00
WhatFor: Trace what was built, why, what failed, and how to review/continue the mock IdP work.
WhenToUse: Read before resuming work on MOCK-OIDC-IDP; append a new Step for each chunk of work.
---



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

## Step 2: Phase 0 — baseline OIDC happy path

This step made the mock IdP real: a single Go binary that speaks the OIDC happy path end to end. The design doc's research baseline was transcribed into the `cmd/tinyidp/main.go` layout with one small refactor (extracting `registerRoutes`) to make the handlers testable with `httptest`. No external dependencies were added; only the standard library is used. The exit criterion — "an ID token issued for `alice` whose signature verifies against `/jwks`" — is enforced by a Go test, not just a curl smoke test.

### Prompt Context

**User prompt (verbatim):** "Create a detailed set of phases and for each phases, tasks, and store in the ticket as well. Commit at appropriate intervals. Then work on the tasks step by step, committing at appropriate intervals (including ttmp) and keep a diary as you work."

**Assistant interpretation:** Author a phased task breakdown, commit docs, then implement Phase 0 onward with frequent commits and a live diary.

**Inferred user intent:** Move from planning to a working, tested mock IdP, with reviewable commit boundaries and a traceable diary.

**Commit (code):** d473d5130b78b2eff3c38eadf9c1a1329c8b4604 — "feat(tinyidp): Phase 0 baseline OIDC happy path"

### What I did
- `go mod init github.com/manuel/tinyidp`.
- Wrote `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go` (server struct, discovery, jwks, authorize GET, token POST, userinfo, /healthz, signJWT, verifyPKCE, helpers, withCORS).
- Refactored route registration into `func (s *server) registerRoutes(mux *http.ServeMux)` so tests can mount the server on `httptest.NewServer` without `ListenAndServe`.
- Wrote `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main_test.go` with `httptest`-based tests: discovery fields, bad client/redirect rejection, happy path (no PKCE), happy path (PKCE S256), PKCE mismatch rejection, one-time code use, bad bearer at userinfo, and full ID-token signature verification against the JWKS public key.
- Added `README.md` (env table, run instructions) and `.gitignore`.

### Why
- Standard-library-only keeps `go run .` instant and CI trivial (Decision: single binary, stdlib only).
- Extracting `registerRoutes` is a minimal, low-risk change that unlocks real signature-verification tests — the core Phase 0 exit criterion. Without it, the only way to test is a live port + curl, which can't assert signature validity cleanly.
- Atomic pop-and-delete of the auth code under the mutex (§main.go token handler) is the one concurrency sharp edge; the test `TestCodeIsOneTimeUse` pins it.

### What worked
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green on first run after cleanup (0.693s).
- Runtime smoke test: `curl` against discovery + JWKS returned correct JSON; `/healthz` returned `ok`.
- ID token signature verified via `rsa.VerifyPKCS1v15` against the JWK reconstructed from `n`/`e`.

### What didn't work
- First test-file draft had over-engineered helpers (`randReader`, `mustRSAKey`, dead `ts2`, unused `pem`/`x509`/`time` imports) that wouldn't compile. Rewrote the file cleanly using `crypto/rand` directly. Lesson: avoid speculative future-proofing (JWKS rotation PEM round-trip) in test helpers — add it when Phase 10 needs it.
- Initial smoke test set `OIDC_ADDR=127.0.0.1:5557` but left `OIDC_ISSUER` at the default `http://localhost:5556`, so discovery advertised port 5556 while the server listened on 5557. Not a bug (issuer and addr are independent by design), but a sharp edge for users. Documented in README; tests sidestep it by setting `issuer = ts.URL`.

### What I learned
- `httptest.NewServer` + a `registerRoutes` helper is the clean way to unit-test HTTP servers without touching the network. `CheckRedirect = http.ErrUseLastResponse` is the idiomatic way to capture the 302 Location instead of following it.
- Reconstructing an RSA public key from a JWK is just `big.Int.SetBytes(n)` + `E = int(SetBytes(e))`; no third-party JWT library needed for verification.

### What was tricky to build
- The authorize-redirect capture in tests: the default `http.Client` follows 302s, so the test must install a `CheckRedirect` that returns `http.ErrUseLastResponse` and then read `resp.Location()`. Getting this wrong yields a 200 from the (non-existent) redirect target instead of the code.
- Keeping `fullFlow` generic over optional PKCE params without leaking test-only knobs into production code: the test passes `code_verifier` via an `extra` `url.Values` that the helper forwards only to the token endpoint.

### What warrants a second pair of eyes
- The token handler's critical section: `Lock; ac = codes[code]; delete(codes, code); Unlock`. Confirm the delete is inside the same lock as the read (it is) — any future refactor that splits read and delete reintroduces a code-reuse race.
- `userinfo` reads the token under the lock but does not delete it; that's correct (tokens are reusable until expiry), but confirm no future "revoke" path forgets the lock.

### What should be done in the future
- Add an explicit test that a code past its 5-minute TTL is rejected (requires a fake clock or a short-TTL env knob). Currently TTL is only proven indirectly.
- Consider an integration test that points a real third-party OIDC client (e.g. `golang.org/x/oauth2` + `github.com/coreos/go-oidc`) at the mock to confirm spec compliance beyond hand-rolled assertions.

### Code review instructions
- Start at `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go`: read `token()` for the atomic code pop, then `signJWT()` for RS256, then `verifyPKCE()`.
- Validate: `go test ./... -count=1 -v` — confirm `TestHappyPathWithPKCE` and `TestPKCEVerifierMismatchRejected` both pass (these exercise the trickiest paths).
- Smoke: `go run ./cmd/tinyidp` then `curl -s http://127.0.0.1:5556/.well-known/openid-configuration | jq .`.

### Technical details
- Module: `github.com/manuel/tinyidp`. Go 1.25.5 linux/amd64.
- Key: RSA 2048, generated per startup, `kid="dev-key-1"`.
- Code TTL: 5 minutes. Access token TTL: 1 hour.
- Test count: 7 (`TestDiscoveryContainsRequiredFields`, `TestAuthorizeRejectsBadClient`, `TestAuthorizeRejectsDisallowedRedirectURI`, `TestHappyPathNoPKCE`, `TestHappyPathWithPKCE`, `TestPKCEVerifierMismatchRejected`, `TestCodeIsOneTimeUse`, `TestUserInfoRejectsBadToken` — 8 total).
