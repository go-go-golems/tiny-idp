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
    - Path: internal/cmds/config.go
      Note: ConfigFilePlanBuilder wiring --config-file (commit 871eae0)
    - Path: internal/cmds/print_config.go
      Note: second consumer of reusable oidc section (commit 0257f23)
    - Path: internal/cmds/serve.go
      Note: |-
        Glazed BareCommand composing the oidc section (commit 871eae0)
        buildClientRegistry Lookup-or-merge (commit c9101d8)
    - Path: internal/scenario/scenario.go
      Note: Phase 2 scenario registry (commit 6454cd3)
    - Path: internal/server/authorize.go
      Note: |-
        Phase 1 authorize flow (commit f9ece67)
        per-client validation in parseAuthorizeRequest (commit 5fed666)
    - Path: internal/server/server_test.go
      Note: Phase 4 matrix tests (Phase 4)
    - Path: internal/server/static/login.html
      Note: embedded login form with quick-pick buttons (Phase 3)
    - Path: internal/server/token.go
      Note: |-
        scenario TokenError + MutateClaims threading (commit 6454cd3)
        client auth from registry + cross-client rejection (commit 5fed666)
    - Path: internal/user/user.go
      Note: FromLogin stable sub derivation (commit f9ece67)
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

## Step 3: Phase 1 + refactor into internal/server (go:embed login page)

This step added multiple synthetic users and, at the user's invitation, refactored the single `main.go` into a multi-file `internal/server` package with the login form embedded via `go:embed`. The result is far easier to extend in Phase 2: each endpoint is one file, the scenario registry can live in its own package, and the login page is a real HTML file instead of a Go string literal. All existing tests still pass after the move, and Phase 1 exit criteria (distinct stable subjects per login) are pinned by tests.

### Prompt Context

**User prompt (verbatim):** "feel free to break the software into smaller files, use go:embed to embed assets, etc..."

**Assistant interpretation:** Refactor freely — split into smaller files, embed the login HTML as an asset rather than inlining it.

**Inferred user intent:** Keep the codebase maintainable as phases grow; don't let a monolithic main.go accumulate.

**Commit (code):** f9ece670b67a6e676ed441cbdfa3a78bb56086fe — "feat(tinyidp): Phase 1 multiple synthetic users + package refactor"

### What I did
- Created `internal/user` package (`user.go`): `User`, `FromLogin` (stable `sub = user-<b64(sha256("tinyidp:user:"+login)[:16])>`, synthetic email, name = local part), `Normalize`.
- Refactored `cmd/tinyidp/main.go` (single ~400-line file) into `internal/server` package split by concern:
  - `server.go` — `Server`, `Options`, `New`, `RegisterRoutes`, `Issuer()`/`ClientID()` accessors.
  - `discovery.go` + `jwt.go` — discovery, jwks, `signJWT`, `verifyPKCE`, `b64`.
  - `authorize.go` — `authorizeRequest`, `authorize` (GET+POST), `parseAuthorizeRequest`, `hiddenAuthorizeFields`, `issueCodeAndRedirect`, `errText`.
  - `token.go`, `userinfo.go` — endpoints.
  - `helpers.go` — `writeJSON`, `tokenError`, `randomB64`, `hasScope`, `WithCORS`.
  - `embed.go` + `static/login.html` — login form embedded via `//go:embed`.
- Slimmed `cmd/tinyidp/main.go` to `main()` + `env`/`parseCSV`.
- Converted `/authorize` from immediate redirect to GET (render form) + POST (submit login); echo authorize params as hidden fields.
- Moved tests to `internal/server/server_test.go` (package `server`); added `internal/user/user_test.go`.
- Updated tests for the new contract (login arg on `fullFlow`, POST-based `authorizePostRedirect`, derived-sub assertions).

### Why
- One file per endpoint keeps Phase 2's scenario threading localized: `authorize.go`/`token.go`/`userinfo.go` each become the single place a scenario hook lands.
- `go:embed` makes the login page a real editable HTML file (with future scenario buttons in Phase 3) instead of an escaped Go string; rebuilding is the only step to change the page.
- Extracting `New(Options)` + `RegisterRoutes` keeps `main()` thin and makes the Phase 12 Go test helper (`Start(t, opts)`) almost trivial later.

### What worked
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green: `internal/server` (1.687s) and `internal/user` (0.003s).
- Runtime smoke test against the refactored binary: `/healthz` ok; `GET /authorize` renders `<input id="login">`; `POST /authorize` as `alice` returns `302 -> ...callback?code=...&state=zz`.
- git correctly detected the test file move (`R cmd/tinyidp/main_test.go -> internal/server/server_test.go`).

### What didn't work
- An early multi-edit on the test file failed (`oldText` whitespace mismatch) because tabs vs. the edit's content diverged. Recovered by rewriting the whole test file with `write` instead of patching. Lesson: for large structural test changes, a clean rewrite is faster and less error-prone than many small edits.

### What I learned
- `//go:embed static/login.html` requires the file to live under the package dir; `internal/server/static/login.html` is the right home. The embedded string is parsed once into `template.Must` at package init — safe for concurrent `Execute`.
- Tests in `package server` (internal) can touch unexported fields directly, so `newTestServer` can build a `*Server` with a known RSA key without exporting every field. `New()` is only needed by `main()` and the future public helper.

### What was tricky to build
- The login form must echo *all* authorize params as hidden fields (including `code_challenge`, `nonce`, `state`), otherwise the POST loses context and the flow breaks. `hiddenAuthorizeFields` is the single source of truth for which params survive the round-trip — adding a new param later means updating both it and `parseAuthorizeRequest`.
- `issuer` is set to the test server's URL *after* constructing the `*Server`, because `httptest.NewServer` picks a random port. This is why `newTestServer` mutates `s.issuer` post-construction rather than passing it in.

### What warrants a second pair of eyes
- `parseAuthorizeRequest` is called on both GET and POST. Confirm the POST path re-validates everything (it does) — a future refactor that only validates on GET would let a forged POST bypass client/redirect checks.
- The hidden-field round-trip trusts the browser to echo params back. For a test tool this is fine, but confirm no production assumption leaks in (e.g., re-validating `redirect_uri` against the allowlist on POST — done).

### What should be done in the future
- Phase 2 will replace the loose `user.FromLogin(login)` call in `authorize` POST with `registry.Lookup(login)`, returning a `*Scenario` that carries both the user and any failure hooks. The `issueCodeAndRedirect` signature will gain a scenario param.
- Consider a `Server.SetIssuer(string)` test helper (or a `NewWithKey` constructor) so tests don't poke `s.issuer` directly; acceptable as-is only because tests are in-package.

### Code review instructions
- Start at `internal/server/authorize.go`: read `authorize` (GET/POST switch), then `parseAuthorizeRequest` (validation chokepoint), then `issueCodeAndRedirect` (code storage + redirect).
- Then `internal/user/user.go`: `FromLogin` (subject derivation) + `Normalize`.
- Validate: `go test ./... -count=1 -v -run 'TestDistinct|TestSubIsStable|TestArbitraryEmail|TestAuthorizePOSTRequiresLogin'`.

### Technical details
- Package layout now: `cmd/tinyidp/main.go`; `internal/server/{server,embed,discovery,jwt,authorize,token,userinfo,helpers}.go` + `static/login.html`; `internal/user/user.go`.
- Test counts: server (12 tests), user (5 tests).
- Login page embedded size: ~2 KB.

## Step 4: Phase 2 — scenario registry

This step replaced ad-hoc, login-string switches with a `Scenario` data model and a `Registry`. A scenario bundles a synthetic user with optional failure hooks for each OIDC stage (authorization, token, ID-token claims, userinfo). Handlers now resolve a login to a `*Scenario` once and branch on the scenario's fields, so adding a new failure case (Phase 4) is one registry entry, not edits in three handlers. The "one-file add" property is pinned by a test that injects a custom-claim scenario and asserts the claim surfaces in the issued ID token with zero handler changes.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Continue the phased plan; Phase 2 is the scenario registry that decouples failure behavior from handler code.

**Inferred user intent:** Make the mock extensible so Phase 4's failure scenarios are trivial to add and self-documenting.

**Commit (code):** 6454cd3ab379ed4bab8d6f1db9f911a7784ec882 — "feat(tinyidp): Phase 2 scenario registry"

### What I did
- Created `internal/scenario/scenario.go`: `Scenario` struct (`Name`, `Description`, `User`, `AuthError`, `TokenError`, `UserInfoError`, `MutateClaims`); `Registry` with `New` (preloads `alice`/`bob`), `Lookup` (with fallback deriving a normal user from any login), `All`, `Register`.
- Threaded `*scenario.Scenario` through `authCode` and `accessToken` (replacing the placeholder `FailureMode string`).
- `authorize` POST: `registry.Lookup(login)`, then `AuthError` → `redirectOAuthError` (no code); else `issueCodeAndRedirect(..., sc.User, &sc)`.
- `token`: applied `TokenError` switch (`invalid_grant`/`server_error`/`slow` 10s sleep) before issuing; applied `MutateClaims` to the claims map after building it; stored `Scenario` on the access token.
- `userinfo`: applied `UserInfoError` switch (`401`/`500`/`sub_mismatch` returning a different `sub`).
- Added `Server.Registry()` accessor and a `Registry.Register` method (the single extension point).
- Tests: `scenario_test.go` (preload, fallback, normalize, All, MutateClaims contract); `TestScenarioHookIsThreadedThroughFlow` in server tests (the one-file-add guarantee).

### Why
- The Phase 1 design had `user.FromLogin` called directly in `authorize`; without a registry, Phase 4 would require per-handler `switch login` blocks that drift. The registry makes the login→behavior mapping a single source of truth.
- `MutateClaims` as a `func(claims, now)` keeps ID-token mutations composable and testable in isolation (scenario_test.go calls it directly without an HTTP round-trip).
- `Register` is deliberately the only mutation entry point so Phase 3 (login page from `All()`) and future config-driven scenarios go through one API.

### What worked
- `go build/vet/test` green across all four packages (`cmd/tinyidp`, `internal/server`, `internal/scenario`, `internal/user`).
- `TestScenarioHookIsThreadedThroughFlow` passed first try after fixing the `Register` access — clean evidence that the scenario threads end-to-end through authorize→token→ID-token.

### What didn't work
- First server test draft reached into `s.registry.m` directly (`vet: cannot refer to unexported field m`) because the test is in `package server` but `m` belongs to `internal/scenario`. Fixed by adding `Registry.Register` and using it — which is actually the better API anyway (tests use the same extension point users will).
- The scenario unit test's `exp` assertion was initially `v >= 0`, which is always true for a Unix timestamp. Rewrote to compare against `time.Now().Unix()`. Lesson: assert against a meaningful reference, not a sign check.

### What I learned
- Exposing `Register` (rather than letting callers poke the map) keeps the registry's invariants (keyed by `Name`, fallback preserved) in one place. This matters for Phase 3, which reads `All()` and must stay consistent with what `Lookup` returns.
- The `*Scenario` pointer is stored on both `authCode` and `accessToken` so the same scenario object drives failures at token time *and* userinfo time from a single login. No re-lookup needed.

### What was tricky to build
- Ordering in `token`: the `TokenError` switch must run *before* storing the access token and signing the ID token (otherwise a `server_error` scenario would still issue a token). `MutateClaims` must run *after* the base claims are built (including the nonce echo) so mutators can delete/override any field. Both are now in the right order, with comments.
- `time.Sleep(10 * time.Second)` for `token-slow` is a real 10s block in a handler; acceptable for manual testing but will need a shorter duration or a knob if it lands in CI. Documented as a Phase 4 open item.

### What warrants a second pair of eyes
- `MutateClaims` can overwrite `iss`/`aud`/`sub`/`nonce` arbitrarily (that's the point — `id-wrong-iss`, `id-bad-nonce`). Confirm the token still *signs* correctly after mutation (it does: signing happens after `MutateClaims`). A reviewer should confirm no mutator accidentally breaks JSON marshalling (e.g., a non-serializable value).
- `token-slow` sleeps with the mutex *not* held (correct — it's before the lock), but confirm no future refactor moves it inside the critical section.

### What should be done in the future
- Phase 4: register the actual failure scenarios (`id-expired`, `id-wrong-aud`, `userinfo-401`, etc.) in `builtinScenarios()`. Each is ~5 lines.
- Consider a `Scenario.Category` field so Phase 3 can group buttons without re-deriving categories from name prefixes.
- `token-slow` duration should be configurable (env or scenario field) before CI use.

### Code review instructions
- Start at `internal/scenario/scenario.go`: read `Scenario`, then `Registry.Lookup` (fallback), then `Register`.
- Then `internal/server/token.go`: the `TokenError` switch + `MutateClaims` call ordering.
- Validate: `go test ./internal/scenario/ ./internal/server/ -count=1 -run 'Scenario|MutateClaims|Hook' -v`.

### Technical details
- Packages: 4 (`cmd/tinyidp`, `internal/server`, `internal/scenario`, `internal/user`).
- Test counts: scenario (5), server (13), user (5).
- Registry preloads 2 normal scenarios (`alice`, `bob`); fallback covers all other logins.

## Step 5: Phase 3 — self-documenting login page

This step wired the scenario registry into the login page so the page is self-documenting: it lists every registered scenario as a one-click quick-pick button, grouped by category. The page and the registry can no longer drift, because the page is rendered from `registry.Grouped()`. A developer opening `/authorize` now sees `alice`, `bob` (and, after Phase 4, every failure mode) as buttons rather than having to memorize magic usernames.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement Phase 3 — make the login page list selectable scenarios grouped by category.

**Inferred user intent:** Reduce friction and memorization; make the mock IdP discoverable for new users.

**Commit (code):** (this step) — "feat(tinyidp): Phase 3 self-documenting login page"

### What I did
- Added `Category` field to `scenario.Scenario`; categorized `alice`/`bob` as "Normal users".
- Added `scenario.Registry.Grouped()` returning `[]CategoryGroup` (first-seen category order, skips uncategorized).
- Added `Server.scenarioGroups()` bridging `scenario.CategoryGroup` → the template's `scenarioGroup`/`scenarioItem` shape.
- Wired `Scenarios: s.scenarioGroups()` into the GET authorize handler's template data (previously nil).
- The embedded `static/login.html` already had the `{{range .Scenarios}}` markup with `data-login` buttons and a small JS handler to fill the login input on click.
- Test `TestLoginPageListsBuiltinScenarios` asserts the page renders "Quick picks", "Normal users", and a `data-login="alice"`/`"bob"` button.

### Why
- Rendering the page from `registry.Grouped()` is a single source of truth: adding a scenario in Phase 4 automatically surfaces it on the page. No separate "update the docs" step.
- The `Category` field (rather than name-prefix parsing) keeps grouping explicit and stable.

### What worked
- `go build/vet/test` green; live smoke test (port 5561, after killing stale processes) shows "Quick picks" + `alice`/`bob` buttons.
- `TestLoginPageListsBuiltinScenarios` passes in-package.

### What didn't work
- **Stale-process trap:** earlier `go run` smoke tests left `tinyidp` processes bound to :5556, so later `curl` calls hit a pre-Phase-3 binary and appeared to show the page broken. The code was correct all along (the in-package test passed). Fix: `pkill -9 -f 'exe/tinyidp'` and use unique ports (`OIDC_ADDR=127.0.0.1:55XX`) per smoke test. Lesson: always confirm no leftover listener before trusting a curl result, and prefer `httptest`-based assertions over live-port curls.
- A first test draft had a stray `for _, sc := range newTestServer(t).0` line (autocomplete noise) that I removed before committing.

### What I learned
- `go:embed` + `html/template` renders server-side; the page content is purely a function of the template + the `loginPageData` model, so testing the rendered HTML string is a valid, fast assertion.
- `pkill -f tinyidp` is dangerous because it can match the agent's own subprocess tree; matching `exe/tinyidp` is precise.

### What was tricky to build
- Distinguishing "the code is wrong" from "the curl hit a stale process". The in-package `httptest` test was the decisive signal — it rendered scenarios correctly, which proved the live binary would too once the port was free.
- Template truthiness: `{{if .Scenarios}}` is falsy for a nil slice but truthy for a non-nil (even empty) slice. `Grouped()` returns a non-nil slice, so the section renders even with zero scenarios (correct: the header just has no items).

### What warrants a second pair of eyes
- Confirm `Grouped()`'s first-seen ordering is deterministic enough for the page. It iterates a Go map, so category order within `Grouped()` is *not* stable across runs (Go randomizes map iteration). The *categories* are stable (alice/bob both "Normal users" → one group), but if Phase 4 adds many categories, their on-page order may shuffle per restart. Acceptable for a test tool; flag for review.

### What should be done in the future
- If category order matters (e.g. Phase 4 wants "Normal users" first, then "ID token failures"), either sort `Grouped()` by a defined category priority or store categories in an ordered slice rather than deriving from map iteration.
- Add a "show advanced" toggle once scenario count grows past ~15 to keep the page scannable.

### Code review instructions
- Start at `internal/scenario/scenario.go`: `Grouped()`; then `internal/server/embed.go`: `scenarioGroups()`; then `authorize.go` GET branch.
- Validate: `go test ./internal/server/ -run TestLoginPageListsBuiltinScenarios -v`; then `go run ./cmd/tinyidp` and open `/authorize?...` in a browser.

### Technical details
- Login page now ~2.6 KB rendered; 2 quick-pick buttons (alice, bob).
- Category field added without breaking existing scenario registrations (zero value = uncategorized, skipped by Grouped).

## Step 6: Phase 4 — high-value failure scenarios

This step added the actual failure scenarios as pure data entries in `builtinScenarios()` — the payoff of the Phase 2 registry design. Each scenario is ~5 lines and required zero handler edits. A matrix test asserts every failure surfaces where the spec says it should: auth errors redirect with the OAuth error code, token errors return the right status + error, ID-token mutations produce invalid claims, and userinfo errors return the right status (or a mismatched sub). This completes the MVP cutoff (Phases 0–4).

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement Phase 4 — register the high-value failure scenarios and verify each end-to-end.

**Inferred user intent:** Deliver the MVP: a mock IdP that reproduces real OIDC client bugs (auth denial, token exchange failure, invalid ID tokens, broken userinfo).

**Commit (code):** (this step) — "feat(tinyidp): Phase 4 high-value failure scenarios"

### What I did
- Registered 17 scenarios in `internal/scenario/scenario.go` `builtinScenarios()`, grouped by Category:
  - Normal users: `alice`, `bob`.
  - Authorization failures: `fail-access-denied`, `fail-login-required`, `fail-consent-required`, `fail-server-error`.
  - Token failures: `token-invalid-grant`, `token-server-error`, `token-slow` (uncategorized — 10s sleep, not a quick-pick).
  - ID token failures: `id-expired`, `id-wrong-aud`, `id-wrong-iss`, `id-missing-email`, `id-email-unverified`, `id-bad-nonce`, `id-future-iat`.
  - UserInfo failures: `userinfo-401`, `userinfo-500`, `userinfo-sub-mismatch`.
- Added a Phase 4 matrix test in `internal/server/server_test.go`:
  - `TestPhase4_AuthErrorScenariosRedirectWithError` (4 subtests) — each redirects with the right `error` + `state`, no code issued.
  - `TestPhase4_TokenErrorScenarios` (2 subtests) — 400 `invalid_grant`, 500 `server_error`.
  - `TestPhase4_IDTokenMutations` — 7 ID-token checks (past exp, wrong aud, wrong iss, missing email, unverified email, wrong nonce, future iat).
  - `TestPhase4_UserInfoFailures` (2 subtests) — 401, 500.
  - `TestPhase4_UserInfoSubMismatch` — userinfo sub differs from ID token sub (suffix `-different`).
- Added test helpers `runAuthorizeLogin` and `exchangeCode` to reduce matrix boilerplate.

### Why
- The whole point of the scenario registry was to make these failures trivial to add. Phase 4 proves it: 17 scenarios, ~120 lines of data, zero handler changes.
- A matrix test (one per failure) is the only way to confidently say "the mock reproduces this bug." Hand-testing 17 flows manually would drift.

### What worked
- `go build/vet/test` green; 37 tests total (was 23 before Phase 4).
- Every ID-token mutation test verified the signature *after* mutation, proving mutations don't break signing (they only change claim values).
- `id-bad-nonce` correctly only mutates when a nonce was sent (the test always sends one, but the guard keeps it safe when an RP omits nonce).

### What didn't work
- First draft of `TestPhase4_UserInfoFailures` had a dead `fullFlow("userinfo-401")` call at the top that failed because `fullFlow` asserts userinfo returns 200 (and `userinfo-401` returns 401). Removed the dead call; the real assertions are in the loop below. Lesson: don't reuse a happy-path helper for failure cases — it has incompatible success assumptions.

### What I learned
- JSON numbers unmarshal to `float64` in Go, so `claims["exp"].(float64)` is the right assertion type, not `int64`. Caught this when the `id-expired` assertion would have panicked on a wrong type.
- `verifyIDTokenSignature` returns the *parsed claims*, so mutation tests can assert on claim values without a second decode — a nice reuse of the Phase 0 helper.

### What was tricky to build
- `id-wrong-iss` mutates `iss` by appending `/wrong`, which requires reading the current value: `claims["iss"].(string) + "/wrong"`. If a future mutator runs before `iss` is set, this panics; confirmed the base claims set `iss` first.
- `userinfo-sub-mismatch` returns 200 (not an error status) but with a *different* `sub`. The test must not treat "200" as success — it asserts the sub differs. This is the subtle one reviewers should eyeball.

### What warrants a second pair of eyes
- `id-bad-nonce` only mutates `if _, ok := claims["nonce"]; ok`. If an RP sends no nonce, the scenario becomes a no-op (normal token). Confirm that's the desired behavior (vs. always setting a wrong nonce) — I chose "no-op when no nonce" because echoing a nonce the RP didn't send would itself be a different bug class.
- `token-slow` sleeps a real 10s. It's intentionally not a quick-pick (no Category), but it *is* resolvable via the text input. Confirm no CI test accidentally logs in as `token-slow` (none do).

### What should be done in the future
- Add a `token-slow` test with a configurable (short) duration, or extract the sleep duration to a scenario field so tests can override it.
- Phase 5+ (multiple clients, sessions, refresh tokens, etc.) are now the next targets; the MVP (0–4) is complete and shippable.

### Code review instructions
- Start at `internal/scenario/scenario.go`: `builtinScenarios()` — read each scenario's failure field.
- Then `internal/server/server_test.go`: the `TestPhase4_*` matrix; focus on `TestPhase4_IDTokenMutations` (7 assertions) and `TestPhase4_UserInfoSubMismatch`.
- Validate: `go test ./... -count=1 -run Phase4 -v`.

### Technical details
- Scenarios: 19 total (2 normal + 17 failure; `token-slow` uncategorized).
- Tests: 37 total across 4 packages (server 26, scenario 5, user 5, cmd 0).
- MVP (Phases 0–4) exit criteria all met: happy path, multiple users, scenario registry, self-documenting login page, high-value failures.

## Step 7: Delivery — validation + reMarkable upload

This final step validated the ticket with `docmgr doctor`, uploaded the bundled design + phases + diary documents to reMarkable, and verified the remote listing. The MVP (Phases 0–4) is complete, tested (37 tests green), and delivered. Later phases (5–12) remain documented in the phases doc but are intentionally deferred.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the deliverable: validate docs, upload to reMarkable, verify.

**Inferred user intent:** Close the loop on the original request — store the design in the ticket and upload to reMarkable.

### What I did
- Ran `docmgr doctor --ticket MOCK-OIDC-IDP --stale-after 30`; resolved the single `unknown_topics` warning by adding `oidc`, `go`, `testing`, `identity`, `auth` to the vocabulary. Re-ran doctor: ✅ All checks passed.
- Dry-run bundle upload of the design doc + phases doc + diary → `Mock OIDC IdP — Design, Phases, Diary.pdf`.
- Real upload to `/ai/2026/06/22/MOCK-OIDC-IDP`.
- Verified: `remarquee cloud ls /ai/2026/06/22/MOCK-OIDC-IDP --long --non-interactive` shows the PDF.

### Why
- The deliverable checklist requires doctor-clean docs and a verified reMarkable upload before handoff.

### What worked
- Doctor clean after vocabulary addition.
- Bundle upload succeeded on first real run (dry-run caught no issues).
- Remote listing confirms the PDF landed at the expected path.

### What didn't work
- Nothing failed in this step.

### What I learned
- The vocabulary is project-wide; adding topics once benefits all future tickets in this repo.

### What was tricky to build
- None.

### What warrants a second pair of eyes
- The bundle contains the diary as-is (long, with all the "what didn't work" honesty). Confirm that's desired for the reMarkable audience vs. a trimmed handoff. I kept the full diary because the user asked to "keep a detailed diary."

### What should be done in the future
- When Phase 5+ work begins, append new diary steps here (do not rewrite history).
- Re-upload the bundle after major doc revisions.

### Code review instructions
- Validate delivery: `docmgr doctor --ticket MOCK-OIDC-IDP --stale-after 30` (expect all-pass); `remarquee cloud ls /ai/2026/06/22/MOCK-OIDC-IDP --long --non-interactive` (expect the PDF).
- Validate code: `go build ./... && go vet ./... && go test ./... -count=1`.

### Technical details
- reMarkable path: `/ai/2026/06/22/MOCK-OIDC-IDP/Mock OIDC IdP — Design, Phases, Diary`.
- Final test count: 37 (server 26, scenario 5, user 5, cmd 0).
- Commits: 12 total (1 initial + 11 from this work).

## Step 8: Adopt the Glazed command framework

This step replaced the hand-rolled env-var-only CLI with a Glazed-based CLI, per the user's explicit request. The OIDC provider config is now a **reusable field section** (`internal/sections/oidc`) composed into the `serve` command; the root is wired like `glaze` (logging section, embedded help, `help_cmd.SetupCobraRootCommand`); and the full config precedence chain (defaults < config files < env < args < flags) is wired and verified. Profile *flags* are ready (`--profile`/`--profile-file`); profile *file* loading remains "later down the road" as the user phrased it. This supersedes the original "stdlib only" decision for the CLI layer (the HTTP/JWT/JWKS/PKCE layer stays stdlib-only).

### Prompt Context

**User prompt (verbatim):** "use the glazed command framework and update the current setup. We want proper schema sections we can reuse, and later down the road use profiles and such. (read the skill, potentially use glaze help --all to get more information)"

**Assistant interpretation:** Adopt Glazed for the tinyidp CLI; define the OIDC config as a reusable schema section; leave profiles as a ready-but-future path.

**Inferred user intent:** Stop hand-rolling flags/env vars; get reusable, composable config sections and a documented path to profiles/config files, with proper introspection (`--print-schema`).

**Commit (code):** 871eae01b98f38b7acec57a875f883e86fa815ef — "feat(tinyidp): adopt Glazed command framework with reusable OIDC section"

### What I did
- Read the `glazed-command-authoring` and `glazed-help-page-authoring` skills; surveyed `glaze help --all` and the key topics (`custom-section-tutorial`, `profiles`, `config-files`, `sections-guide`). Confirmed API details against the glazed v1.3.6 source (`schema.NewSection`, `fields.New`+`TypeStringList`, `cli.WithProfileSettingsSection`, `ConfigPlanBuilder`, `BareCommand`).
- Added `github.com/go-go-golems/glazed@v1.3.6`.
- `internal/sections/oidc/{section,settings}.go`: reusable OIDC section (`NewSection` declares issuer/addr/client-id/client-secret/redirect-uris with defaults) + `Settings` decode target + `GetSettings` (normalizes issuer). Decoupled from `internal/server` so it is genuinely reusable.
- `internal/cmds/serve.go`: `ServeCommand` as a `cmds.BareCommand` (Run, no row emission — a server has no tabular output) composing the OIDC section + Glazed command-settings section.
- `internal/cmds/config.go`: `ConfigFilePlanBuilder` so `--config-file` actually loads (the flag is otherwise a no-op, confirmed by glazed's own `TestCobraParserDoesNotImplicitlyLoadConfigFileWithoutPlanBuilder`).
- `cmd/tinyidp/main.go`: Glazed root init (logging section, embedded help, `help_cmd.SetupCobraRootCommand`); `serve` built with `AppName: "tinyidp"` (env loading) + `cli.WithProfileSettingsSection()` + the config-file plan builder.
- `cmd/tinyidp/doc/{doc.go,pages/*.md}`: embedded help via `go:embed all:pages` (tinyidp overview + oidc-config topic).
- `internal/sections/oidc/section_test.go`: section shape, defaults round-trip via `sources.Execute`+`FromDefaults`, env override (`TINYIDP_*`), issuer normalization.
- Updated README (flags/env/config table, precedence, introspection, profiles-ready) and the design doc (superseded "stdlib only" decision; added "Glazed CLI + reusable field sections" decision).

### Why
- A reusable section means the OIDC config (flags + env + config-file schema) is defined **once**; any future command (`print-config`, `gen-key`, the Phase 12 test helper) composes `oidc.NewSection()` and gets identical config without redefinition. This is the "proper schema sections we can reuse" the user asked for.
- Glazed's built-in parser chain gives the full precedence (defaults < config < env < args < flags) for free, with a parse log (`--print-parsed-fields`) that proves which source won — far better than the previous opaque env-var resolution.
- Wiring the root like `glaze` (logging + help) is the canonical pattern from both skills; deviating would leave the CLI in an "incomplete initialization state" per the help skill.

### What worked
- `go build/vet/test` green across all packages (added 4 OIDC section tests).
- `tinyidp serve --help` renders grouped sections: OIDC Provider Configuration, General purpose command options (--config-file/--print-parsed-fields/--print-schema/--print-yaml), Profile settings, and inherited Logging flags.
- `--print-parsed-fields` shows the full parse log with `source:` per value (defaults/env/config) — the introspection the user wanted.
- **Precedence verified end-to-end:** config-only → `cfg-client`; config+env → `env-cid`; config+env+flag → `flag-cid`. Exactly the documented chain.
- Full authorize→token→userinfo flow works under env-var-only configuration (`TINYIDP_ADDR/CLIENT_ID/REDIRECT_URIS`), issuing an RS256 ID token verifiable against JWKS.

### What didn't work
- First `ConfigFilePlanBuilder` draft had the wrong signature (returned a builder-of-builders). The type is `func(*values.Values, *cobra.Command, []string) (*config.Plan, error)`; fixed by matching glazed's own test pattern.
- Initial live smoke-test appeared to show env vars not loading, but that was a misread: I hadn't actually set `TINYIDP_CLIENT_ID` in that run, so the server correctly rejected `client_id=env-app` as unknown. `--print-parsed-fields` confirmed env loading works. (Recurring lesson from Step 5: trust the in-process assertion over the live-port curl.)
- `--config-file` did nothing until I wired `ConfigPlanBuilder` — by design, glazed does not implicitly load config files without a builder (there's a dedicated test asserting this). Documented in the decision record.

### What I learned
- `BareCommand` (Run returns error, no processor/writer) is the right interface for a long-running server command; `GlazeCommand`/`RunIntoGlazeProcessor` is for row-emitting commands. `BuildCobraCommand` detects the interface via type assertion.
- The env-var prefix is `strings.ToUpper(AppName)` → `TINYIDP_*`. This is a breaking change from the previous `OIDC_*` (documented in the decision record; acceptable pre-release).
- `go:embed all:pages` (with `all:`) is the safe way to embed a help subdirectory without also embedding the `.go` source file in the same package dir.

### What was tricky to build
- The `ConfigPlanBuilder` returns a `*config.Plan`. Returning `nil` is risky (unclear how `FromConfigPlanBuilder` handles it); returning an empty `config.NewPlan(config.WithLayerOrder(config.LayerExplicit))` with no `Add` is a clean no-op when `--config-file` is unset.
- Layer ordering: the built-in chain appends in reverse precedence (last = highest). The resulting order is flags > args > env > config > defaults, matching the documented chain. Getting this wrong would invert precedence silently; verified with the cfg/env/flag triple test.
- The OIDC section is decoupled from `internal/server` to avoid an import cycle and keep it reusable — `serve.go` bridges section → `server.Options`.

### What warrants a second pair of eyes
- The env-prefix change (`OIDC_*` → `TINYIDP_*`) is a breaking change for anyone following the pre-Glazed diary smoke-test commands. Confirm this is acceptable (MVP unreleased; I believe it is).
- `ConfigFilePlanBuilder` reads `--config-file` from `cmd.Flags()`; confirm the flag is registered (by the command-settings section) before the builder runs. It is (ParseCommandSettingsSection runs first), but a future refactor that skips the command-settings section would break config loading silently.
- Profile *file* loading (`profiles.yaml` resolution via `middlewares.GatherFlagsFromProfiles`) is NOT wired — only the `--profile`/`--profile-file` flags exist. Anyone expecting `--profile dev` to actually apply a profile today will be surprised; the help page says "requires profiles.yaml".

### What should be done in the future
- Wire `middlewares.GatherFlagsFromProfiles` so `--profile` resolves a `profiles.yaml` (the documented "later down the road" profiles feature). The flag plumbing is already in place.
- Consider a `print-config` subcommand that composes the same OIDC section and emits the resolved config as rows (a natural use of the reusable section + Glazed output).
- Re-upload the bundle to reMarkable after this docs update (Step 9 / delivery refresh).

### Code review instructions
- Start at `internal/sections/oidc/section.go`: the reusable section. Then `internal/cmds/serve.go`: how a command composes it. Then `internal/cmds/config.go`: the `--config-file` loader. Then `cmd/tinyidp/main.go`: the root init.
- Validate: `go test ./... -count=1`; `go run ./cmd/tinyidp serve --print-parsed-fields`; `go run ./cmd/tinyidp help oidc-config`.
- Precedence triple-check: the three `--print-parsed-fields` runs in "What worked" (cfg/env/flag).

### Technical details
- Dependency: `github.com/go-go-golems/glazed v1.3.6` (+ cobra v1.10.2, zerolog v1.35.1).
- New packages: `internal/sections/oidc`, `internal/cmds`, `cmd/tinyidp/doc`.
- Test count: 41 (server 26, scenario 5, user 5, oidc 4, + 1 unused).
- Env prefix: `TINYIDP_*`. Config-file key: `oidc:`. Profile flags: `--profile`/`--profile-file`.

## Step 9: Profiles + print-config

This step delivered the two Glazed CLI extras the user asked for: working profile resolution (so setups are switchable with `--profile`) and a `print-config` command (to debug the resolved configuration). Profiles are wired through a custom `MiddlewaresFunc` that inserts `GatherFlagsFromProfiles` at the correct precedence layer (above defaults, below config/env/flags), with a bootstrap resolution of `--profile`/`--profile-file` that honors `TINYIDP_PROFILE` env. `print-config` is the second consumer of the reusable `oidc` section, which validates that the section is genuinely decoupled from `serve`.

### Prompt Context

**User prompt (verbatim):** "go ahead. I want profiles so we can quickly switch setups, print-config is nice to debug things, and then the phases. Continue committing at appropriate intervals, and keeping a diary, and checking off tasks"

**Assistant interpretation:** Implement profiles and print-config now, then resume the deferred phases (5+), with continued commits/diary/task-checking.

**Inferred user intent:** Make configuration switchable and debuggable before adding more provider features.

**Commit (code):** ca2ada2 (profiles) + 0257f23 (print-config)

### What I did
- `internal/cmds/profiles.go`: `ProfileMiddlewaresFunc(appName, configPlanBuilder)` — a `cli.CobraMiddlewaresFunc` that builds the full source chain with `GatherFlagsFromProfiles` inserted above defaults and below config/env/flags. Bootstrap-resolves `--profile`/`--profile-file` from the parsed command-settings section + `TINYIDP_PROFILE`/`TINYIDP_PROFILE_FILE` env before constructing the profile middleware (per the Glazed implementing-profile-middleware pattern, to avoid capturing defaults before env is applied).
- `cmd/tinyidp/main.go`: `serve` now uses `MiddlewaresFunc: cmds.ProfileMiddlewaresFunc(...)` (replacing the default chain), keeping `WithProfileSettingsSection()` for the flags. Default profile file: `~/.config/tinyidp/profiles.yaml` (XDG-aware via `os.UserConfigDir`).
- `internal/cmds/print_config.go`: `PrintConfigCommand` as a `cmds.GlazeCommand` composing the same reusable `oidc` section as `serve` + the Glazed output section (default `yaml`). Emits one row with the resolved config.
- `cmd/tinyidp/doc/pages/profiles.md`: help page (file format, default location, precedence, error behavior, introspection).
- Tests: 4 profile tests (`TestProfileOverridesDefaults`, `TestProfileEnvOverridesProfile`, `TestProfileMissingDefaultFileSkipsSilently`, `TestProfileExplicitMissingFileErrors`) + 3 print-config tests (`TestPrintConfigEmitsResolvedDefaults`, `TestPrintConfigReflectsEnvOverride`, `TestPrintConfigReflectsProfileOverride`).

### Why
- Profiles sit above defaults and below config/env/flags by design: a profile is a convenient baseline (e.g. a `dev` issuer vs a `ci` issuer), but a local override (a flag, an env var) must always win. This is the Glazed-recommended placement for environment presets.
- `print-config` is both a debugging tool and a proof of reusability. If `print-config` could not compose the same `oidc` section and resolve it identically to `serve`, the "reusable section" claim from Step 8 would be false. The `TestPrintConfigReflectsProfileOverride` test pins that the full chain (profile → row) works through the same section.

### What worked
- Live verification: `--profile dev` → `dev-profile-client`; `--profile ci` → `ci-runner` with custom redirect; `profile+env` → env wins; `profile+env+flag` → flag wins. The `--print-parsed-fields` log shows `source: profiles` winning over `source: defaults`.
- `print-config` outputs yaml by default and `--output json` works; profile and env overrides surface in the emitted row.
- 48 tests green across 5 packages.

### What didn't work
- First profiles_test draft used `parsed.Set(slug, key, val)` which doesn't exist; `Values.Set` takes `(slug, *SectionValues)`. Fixed by using `GetOrCreate(section)` + `sectionValues.Fields.Set(key, *FieldValue)` with a `setField` helper that builds the `FieldValue` from the section's definition. This mirrors how glazed's own tests populate values.
- `FromCobra(nil, ...)` panics (it dereferences the cobra command). The middleware now guards `if cmd != nil` before adding `FromCobra`, mirroring `ParseCommandSettingsSection`'s pattern. Tests pass `nil` cmd; production always passes a real cmd.
- `print-config` first used `cli.NewGlazedSchemasWithDefaults` (doesn't exist) and a `fields` default key (rejected: "unknown field"). Fixed to `settings.NewGlazedSchema` with `WithOutputSectionOptions(schema.WithDefaults({"output":"yaml"}))`. The `fields` key belongs to a different sub-section.
- `captureProcessor` needed a `Close` method to satisfy `middlewares.Processor`; `Row.Get` returns `(value, ok)` (it's an orderedmap). Added both.

### What I learned
- The Glazed profile middleware's 4th argument (`defaultProfileName`, defaulting to `"default"`) is what makes the no-profiles.yaml case work: if the default file is missing AND the requested profile is the default name, loading skips silently. An explicitly-named profile with a missing default file errors. This is the right UX: out-of-the-box works, typos are loud.
- The "bootstrap parse then build chain" pattern exists because profile *selection* can come from env/config — if you constructed the profile middleware before env was applied, you'd capture the default profile name and load the wrong profile. The bootstrap parse resolves selection first, then the chain applies profiles with the resolved selection.

### What was tricky to build
- Getting the precedence exactly right. The chain is built in *reverse* precedence (last applied wins): flags, args, env, config, profiles, defaults. If profiles were appended after env, env would not override profiles. The `TestProfileEnvOverridesProfile` test pins this; without it, a future refactor could invert precedence silently.
- `print-config`'s glazed section: the output default must use a valid field name (`output`, not `fields`). The glazed section is composed of sub-sections, so `WithDefaults` keys must match a real field in one of them.

### What warrants a second pair of eyes
- The `ProfileMiddlewaresFunc` guards `FromCobra` on `cmd != nil`. Confirm production always passes a real cmd (it does, via `BuildCobraCommand`). A future code path that calls the middleware with a nil cmd in production would silently skip flag parsing — the guard prevents a panic but not silent flag loss.
- Profile *file* loading now errors on an explicitly missing file. Confirm this matches expectations: `--profile-file /typo.yaml` should error (it does), not silently fall back.

### What should be done in the future
- The `token-slow` 10s sleep is still a fixed duration; consider a scenario field or env knob before CI use (carried from Step 6).
- A `--print-profile` or showing the active profile in `print-config` output would help debug which profile was selected (currently only visible via `--print-parsed-fields`).

### Code review instructions
- Start at `internal/cmds/profiles.go`: `ProfileMiddlewaresFunc` (the chain construction) + `resolveProfileSelection` (bootstrap). Then `internal/cmds/print_config.go`: how a second command composes the same `oidc` section.
- Validate: `go test ./internal/cmds/ -count=1 -v -run 'Profile|PrintConfig'`; then `go run ./cmd/tinyidp print-config --profile dev --profile-file <profiles.yaml>`.

### Technical details
- Default profile file: `~/.config/tinyidp/profiles.yaml` (XDG).
- Profile precedence: defaults < profiles < config < env < args < flags.
- New commands: `tinyidp print-config` (GlazeCommand, emits rows), `tinyidp serve` (BareCommand).
- Test count: 48 (cmds 7, server 26, scenario 5, user 5, oidc 4, +1 unused).

## Step 10: Phase 5 — multiple clients

This step replaced the single hardcoded client with a client registry, so a single running provider can serve multiple relying parties. Three built-in clients ship: `dev-client` (permissive, the quick-test default), `public-spa` (a public client with PKCE required), and `web-app` (a confidential client with a secret required). The Phase 0-4 single-client UX is preserved: setting `--redirect-uris` still works and overrides `dev-client`'s allowlist. The key new invariants are per-client redirect URI allowlists, per-client PKCE requirement, and cross-client code rejection (a code issued to one client cannot be redeemed by another).

### Prompt Context

**User prompt (verbatim):** (see Step 9)

**Assistant interpretation:** After profiles and print-config, resume the deferred phases beginning with Phase 5 (multiple clients).

**Inferred user intent:** Let a single mock IdP test public (SPA), confidential (web-app), and permissive (dev) clients against the same instance — the common real-world shape.

**Commit (code):** 5fed666 — "feat(tinyidp): Phase 5 multiple clients (client registry)"

### What I did
- Created `internal/client/client.go`: `Client` struct (`ID`, `Secret`, `RedirectURIs`, `RequirePKCE`, `AllowedScopes`) with `AllowsRedirectURI`/`AllowsScope` helpers; `Registry` (`Lookup`/`Register`/`All`); `NewRegistry` preloads `dev-client`, `public-spa`, `web-app`.
- `Server` now holds `*client.Registry` instead of single-client fields (`clientID`/`clientSecret`/`redirectURIs` removed). `New(Options)` defaults to `client.NewRegistry()` when `opts.Clients` is nil.
- `parseAuthorizeRequest`: looks up the client, checks the redirect URI against *that client's* allowlist, enforces `RequirePKCE` (rejects an authorize with no `code_challenge`), and checks `AllowedScopes` (empty = permissive, the dev-client default).
- `token()`: authenticates the client from the registry; confidential clients (non-empty `Secret`) require it via Basic or post, public clients skip. The existing `ac.ClientID != clientID` check now also serves as cross-client code rejection (the code's `ClientID` was set at authorize time from the authorize request's `client_id`).
- `serve.go` builds the registry from builtins + the OIDC-section-configured client (`buildClientRegistry`): the configured `--client-id`/`--client-secret`/`--redirect-uris` register a single permissive client (default `dev-client`), overriding the builtin. So `--redirect-uris http://localhost:8080/cb` still "just works" as in Phase 0-4.
- Tests: 5 server (`TestPhase5_PublicSpaRequiresPKCE`, `AcceptsWithPKCE`, `WebAppRequiresSecret`, `WebAppSucceedsWithSecret`, `CrossClientCodeRejection`) + 7 client unit tests.

### Why
- A single-client provider can't test the most common real-world shape: an app with a public SPA frontend (PKCE-only, no secret) and a confidential backend (secret required). The registry lets one running provider serve both, plus a permissive `dev-client` for quick manual testing.
- Per-client redirect URI allowlists are the correct OIDC behavior — a redirect URI is valid for a specific client, not globally. The single-client model collapsed this; the registry makes it explicit.
- `RequirePKCE` is enforced at `/authorize` (reject before issuing a code) rather than only at `/token`, so a public-spa client never issues a code that lacks a challenge. The stored `ac.CodeChallenge` is non-empty for such clients, so the existing `verifyPKCE` (which rejects an empty verifier when a challenge exists) handles the token-side enforcement for free.

### What worked
- 60 tests green (was 48); live smoke shows `clients=3` in the log, `public-spa` no-PKCE → 400, `dev-client` renders the login form.
- Cross-client rejection: a code issued to `dev-client` returns 400 `invalid_grant` when redeemed as `web-app`. This is the single most important new security property.
- The OIDC section's `--client-id`/`--client-secret`/`--redirect-uris` still work unchanged, so no breaking change for existing users of the Phase 0-4 single-client flow.

### What didn't work
- First Phase 5 test draft used GET `/authorize` expecting a 302 redirect for `public-spa`/`web-app`, but GET renders the login form (200), not a redirect. The redirect happens on POST. Fixed by using the existing `authorizePostRedirect` helper (which POSTs the login form). Recurring lesson from Step 2/5: the authorize flow is GET-form-then-POST-submit, not GET-redirect.
- Removing `Server.ClientID()` and `Server.Registry()` accessors broke `serve.go` (logging) and the in-package `TestScenarioHookIsThreadedThroughFlow` (`s.registry.Register`). The test is in-package so it accesses `s.registry` directly (no accessor needed); `serve.go` logging switched to `len(srv.Clients().All())`.

### What I learned
- The cross-client code rejection property comes for free from the existing `ac.ClientID != clientID` check, because the code's `ClientID` is set at authorize time from the *authorize request's* `client_id`, and the token handler's `clientID` is the *token request's* `client_id`. If they differ, the code was issued to a different client. No new check needed — the existing one already does the right thing once multiple clients exist.
- `AllowedScopes` empty = permissive is a deliberate choice: it preserves the Phase 0-4 behavior (any scope including `openid` works) for `dev-client`, while a client with a non-empty `AllowedScopes` enforces a real allowlist. This makes the field opt-in rather than mandatory.

### What was tricky to build
- Preserving backward compatibility. The OIDC section still has `client-id`/`client-secret`/`redirect-uris` fields (removing them would break profiles/config files). The `buildClientRegistry` helper registers the configured client on top of the builtins, so the configured client (default `dev-client`) overrides the builtin `dev-client`'s redirect URIs. A developer who only sets `--redirect-uris` gets the same behavior as before.
- The `RequirePKCE` enforcement split: authorize rejects no-challenge requests for such clients, and the stored code's challenge is non-empty, so token-side `verifyPKCE` rejects a missing verifier. Both sides must agree; the test `TestPhase5_PublicSpaRequiresPKCE` pins the authorize side, and `TestPhase5_PublicSpaAcceptsWithPKCE` pins the happy path.

### What warrants a second pair of eyes
- `buildClientRegistry` registers the configured client as permissive (PKCE optional, all scopes). If a user sets `--client-id public-spa` via the OIDC section, they get a *permissive* `public-spa` (overriding the builtin PKCE-required one). This is probably not what they want. Confirm whether the configured client should inherit the builtin's properties when the ID matches, or whether the current "configured overrides builtin" is acceptable. Documented as an open question.
- Cross-client rejection relies on `ac.ClientID != clientID`. Confirm no future refactor sets `ac.ClientID` from the token request instead of the authorize request (that would break the property).

### What should be done in the future
- Resolve the open question above: when the configured `--client-id` matches a builtin (e.g. `public-spa`), should the configured redirect URIs be *added* to the builtin's properties (keeping RequirePKCE) or *replace* them (current behavior)? The current behavior is simpler but potentially surprising.
- Consider exposing client registration via config file (a `clients:` section in `tinyidp.yaml`) so users can define arbitrary clients without code. The registry's `Register` method already supports this; only the config wiring is missing.
- Phase 6 (session cookie, `prompt`, `max_age`) is the next phase; it builds on the client registry (sessions are per-client in real OIDC, though the mock may not need that distinction yet).

### Code review instructions
- Start at `internal/client/client.go`: `Client`, `AllowsRedirectURI`, `AllowsScope`, `Registry`, `BuiltinClients`. Then `internal/server/authorize.go` `parseAuthorizeRequest` (per-client validation) and `internal/server/token.go` (client auth from registry). Then `internal/cmds/serve.go` `buildClientRegistry`.
- Validate: `go test ./internal/client/ ./internal/server/ -count=1 -run 'Phase5|Client' -v`.

### Technical details
- Built-in clients: `dev-client` (public, permissive), `public-spa` (public, PKCE required, redirect `http://localhost:8080/callback`), `web-app` (confidential, secret `dev-secret`).
- Test count: 60 (client 7, cmds 7, server 31, scenario 5, user 5, oidc 4, +1).
- No new dependencies; client package is stdlib-only.

## Step 11: Merge configured client into builtin (resolve open question)

This step resolves the open question from Step 10: when the OIDC-section `--client-id` matches a builtin, the configured client is now **merged** into the builtin instead of replacing it. The builtin's `RequirePKCE`, `Secret`, and `AllowedScopes` are preserved; the configured redirect URIs are unioned (deduplicated) onto the builtin's; a non-empty configured `--client-secret` overrides the builtin's. A configured ID that does not match any builtin still registers a new permissive client (the Phase 0-4 behavior for custom IDs).

### Prompt Context

**User prompt (verbatim):** "I want merging."

**Assistant interpretation:** When the configured client_id matches a builtin, merge rather than replace — keep builtin properties (RequirePKCE, Secret) and add the configured redirect URIs.

**Inferred user intent:** Configure `--client-id public-spa --redirect-uris X` and get a public-spa client that still requires PKCE but also accepts X, rather than a permissive client that lost the PKCE requirement.

**Commit (code):** c9101d8 — "feat(tinyidp): merge configured client into builtin"

### What I did
- Added `client.Merge(base, override Client) Client` + `unionStrings` helper to `internal/client/client.go`. Merge starts from `base`, takes `override.ID`, overrides `Secret` only when non-empty, unions `RedirectURIs` (deduplicated, base's order first), and keeps `RequirePKCE`/`AllowedScopes` from base (no override fields exist in the OIDC section).
- Updated `buildClientRegistry` in `internal/cmds/serve.go` to `Lookup` the configured ID in the builtin registry; if present, `Register(Merge(base, configured))`; else `Register(configured)` as a new permissive client.
- Tests: 5 in `internal/client/merge_test.go` (preserve properties, dedup, non-empty secret overrides, empty secret keeps builtin, override ID wins) + 3 in `internal/cmds/serve_test.go` (merge builtin across public-spa/web-app, register new permissive, default keeps builtins).

### Why
- The replace behavior was surprising and dangerous: `--client-id public-spa --redirect-uris X` silently produced a `public-spa` client with `RequirePKCE=false`, which is the opposite of what the builtin name promises. A test relying on "public-spa requires PKCE" would pass with the builtin and fail (or worse, silently pass insecurely) once a developer added a custom redirect URI.
- Merge makes builtin properties sticky: the builtin defines the client's *class* (public/confidential, PKCE required or not), and configuration only extends the client's *allowlists* (redirect URIs) or overrides the secret. This matches how a developer thinks about "I want to use the public-spa client with my callback URL".

### What worked
- 71 tests green (was 60). Live smoke: configuring `public-spa` with a custom redirect URI via env keeps `RequirePKCE` (both the custom URI `http://localhost:9090/cb` and the builtin `http://localhost:8080/callback` return 400 without PKCE), and an unknown redirect URI is still rejected. The merge is observable end-to-end.
- `unionStrings` dedup matters for the default case: `dev-client`'s builtin redirect URIs exactly equal the OIDC section's defaults, so merge + dedup produces the same set (no spurious duplicates).

### What didn't work
- Nothing failed. The `Merge` design (non-empty overrides Secret; union RedirectURIs; keep RequirePKCE/AllowedScopes) was unambiguous because the OIDC section has no fields for RequirePKCE or AllowedScopes, so there was nothing to "merge" for those — they're always taken from base.

### What I learned
- A clean merge rule is: **configured non-empty scalar values override; list values union; fields absent from the config schema are taken from the base**. This generalizes — if a future config field for `RequirePKCE` is added, it would follow the same "non-empty overrides" rule.
- The merge happens in `buildClientRegistry` (the serve command's wiring), not in `Registry.Register`. `Register` stays a plain replace, which is the right primitive for callers that want full control (e.g. tests, a future config-file clients section that defines a client from scratch). Merge is a higher-level policy layered on top.

### What was tricky to build
- Deciding whether `RedirectURIs` should union or replace. Union is the literal meaning of "merge" and is more useful (the user's callback works without losing the builtin defaults), but it is a behavior change from Phase 5 (where `--redirect-uris` replaced the builtin's set for `dev-client`). For `dev-client` the change is invisible in the default case (dedup), and for an explicit `--client-id dev-client --redirect-uris X` the user now gets both X and the builtin defaults — which is the intuitive "extend" behavior. Chose union; documented in the function comment.

### What warrants a second pair of eyes
- The union-of-redirect-URIs behavior for `dev-client`: a user who previously relied on `--redirect-uris X` *replacing* the builtin defaults (so only X was accepted) now gets X plus the builtin defaults. This is a minor backward-incompatible change within the Phase 5 release cycle. Confirm this is acceptable — I believe it is, because the merge is the requested behavior and the builtin defaults are harmless (they're localhost callbacks).
- `Merge` takes `override.ID`. In our usage `base.ID == override.ID` (we looked up base by override.ID), so this is safe. A general-purpose caller passing mismatched IDs would get `override.ID` on a `base`-shaped client, which could be confusing. Documented in the function comment; not a concern for the single caller.

### What should be done in the future
- If a config-file `clients:` section is added (mentioned in Step 10's future work), decide whether clients defined there merge into builtins (by ID match) or always replace. The current `buildClientRegistry` policy is specific to the OIDC-section single client; a multi-client config section would need its own policy, likely "replace unless an explicit merge flag is set".
- Consider exposing `RequirePKCE` and `AllowedScopes` as OIDC-section fields if users want to configure them per-client without code. Currently they're only settable via builtins.

### Code review instructions
- Start at `internal/client/client.go`: `Merge` + `unionStrings`. Then `internal/cmds/serve.go`: `buildClientRegistry` (the Lookup-or-merge).
- Validate: `go test ./internal/client/ ./internal/cmds/ -count=1 -run 'Merge|BuildClient' -v`.

### Technical details
- Merge rule: `out = base; out.ID = override.ID; if override.Secret != "" { out.Secret = override.Secret }; out.RedirectURIs = union(base, override); RequirePKCE/AllowedScopes from base`.
- Test count: 71 (client 12, cmds 10, server 31, scenario 5, user 5, oidc 4, +4).
