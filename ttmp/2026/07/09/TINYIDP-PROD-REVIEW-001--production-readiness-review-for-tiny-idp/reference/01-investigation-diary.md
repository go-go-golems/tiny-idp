---
Title: Investigation diary
Ticket: TINYIDP-PROD-REVIEW-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/admin/backup.go
      Note: Contains the raw file-copy backup reproduced as incomplete in WAL mode
    - Path: repo://internal/authn/password.go
      Note: Defines Argon2id login, lockout state, and password-policy behavior
    - Path: repo://internal/fositeadapter/provider.go
      Note: Owns strict OAuth/OIDC construction, routes, defaults, rate keys, cookies, and audit behavior
    - Path: repo://internal/store/sqlite/store.go
      Note: Implements durable domain state and the non-transactional mutations under review
    - Path: repo://pkg/embeddedidp/options.go
      Note: Defines the advertised production API and the internal-type boundary found by the review
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Typed repository-specific production analyzer (commit 54fcbcf)
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/runtime-probe/main.go
      Note: Strict-flow runtime instrumentation and profile capture (commit 54fcbcf)
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/summary.md
      Note: Measured runtime baseline for the strict happy path (commit 54fcbcf)
ExternalSources: []
Summary: Chronological evidence, commands, failures, decisions, and verification for the tiny-idp production-readiness review.
LastUpdated: 2026-07-09T13:42:50.339918882-04:00
WhatFor: Making the audit reproducible and allowing another engineer to continue or challenge the review.
WhenToUse: Use while reviewing findings, rerunning checks, or continuing production-hardening work.
---









# Investigation diary

## Goal

Record the complete investigation path for the production-readiness review:
what was inspected, why each check was chosen, what succeeded or failed, and
how conclusions were validated.

## Step 1: Establish the review workspace and scope

The review began by locating the actual Git repository inside the shared
multi-repository workspace, checking the existing ticket history, and creating
a dedicated cross-cutting audit ticket. Prior tickets already cover focused
features, while this ticket evaluates the assembled system as a production IdP.

Two untracked source directories already exist under the older
`TINYIDP-PROD-001` ticket. They predate this audit and are being preserved as
user-owned work; they will not be staged in this ticket's commits.

### Prompt Context

**User prompt (verbatim):**

> We have been working on tiny-idp/ and we want to do a full review because we want to ship it to production.
>
> Create a new docmgr ticket, and do  web research and get whatever you need to properly review and inspect this code.
>
> Store interesting doucments in the sources/ folder of the ticket, using defuddle. Keep a detailed diary of your work. Store scripts and tools you build (don't hesistate writing static analysis tooling and instrumentation and smoke testing and fuzzing and such) in the scripts/ folder of the ticket.
>
> Keep a detailed diary as you work, commit at appropriate intervals.
>
> Create  a detailed analysis / design + code review / implementation review that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

**Assistant interpretation:** Create a new docmgr ticket and conduct a broad,
evidence-backed production review of the `tiny-idp` repository, including
external research, reproducible analysis tooling, detailed documentation,
incremental commits, validation, and reMarkable delivery.

**Inferred user intent:** Obtain a trustworthy ship/no-ship assessment and an
intern-friendly technical map that can guide the work required to operate this
identity provider safely in production.

### What I did

- Read the ticket-research, diary, Defuddle, and reMarkable workflow instructions.
- Ran `docmgr status --summary-only` and found 13 existing tickets and 30 documents.
- Ran `git status --short --branch` from the workspace root, then corrected to the nested `tiny-idp/` Git repository.
- Reviewed recent history, the Go module, existing ticket names, and repository size.
- Created `TINYIDP-PROD-REVIEW-001` plus its design and diary documents.
- Added an explicit audit task list and ticket overview.

### Why

- A production review needs a separate evidence trail from the implementation tickets it evaluates.
- Establishing repository ownership and dirty state first prevents unrelated user work from being committed.
- Explicit task boundaries keep the review broad without losing reproducibility.

### What worked

- `docmgr` resolved its root to `tiny-idp/ttmp` from the workspace configuration.
- The new ticket was created with existing vocabulary slugs: `oidc`, `go`, `testing`, `auth`, `architecture`, and `research`.
- The repository history exposed the earlier component tickets and recent strict-provider review work.

### What didn't work

- `git status --short --branch` from `/home/manuel/workspaces/2026-07-07/prod-tiny-idp` failed with `fatal: not a git repository (or any of the parent directories): .git`.
- `sed -n '1,240p' go.mod` from that same directory failed with `sed: can't read go.mod: No such file or directory`.
- Both commands succeeded after changing the working directory to `tiny-idp/`.

### What I learned

- The checkout is a Go workspace containing `tiny-idp`, `glazed`, and `go-go-goja`; only `tiny-idp/` is in scope for commits.
- The production review must account for approximately 15,481 lines of Go under `cmd/`, `internal/`, and `pkg/`, plus documentation and examples.
- A previous ticket already uses `TINYIDP-PROD-001`, so the new cross-cutting audit is `TINYIDP-PROD-REVIEW-001`.

### What was tricky to build

- The workspace root contains a read-only `.git` directory but is not itself a usable worktree. The symptom was Git's repository error despite the apparent directory. Inspecting nested contents and rerunning Git in `tiny-idp/` established the correct boundary.

### What warrants a second pair of eyes

- Confirm that the audit should cover only `tiny-idp/` runtime behavior, while treating workspace replacements for `glazed` and `go-go-goja` as build inputs rather than code under review.

### What should be done in the future

- Preserve the two pre-existing untracked source directories in the older production ticket unless their owner explicitly decides how they should be handled.

### Code review instructions

- Start at this ticket's `index.md` and `tasks.md` to confirm scope.
- Run `git status --short --branch` from `tiny-idp/` and verify that only this ticket's files are added by review commits.

### Technical details

Initial commands:

```text
docmgr status --summary-only
git status --short --branch
git log -12 --oneline --decorate
find ttmp/2026/07 -mindepth 2 -maxdepth 2 -type d | sort
```

## Step 2: Map the system and capture authoritative production guidance

This step traced the production-shaped system from its public embedding API
through Fosite protocol handling, domain storage, SQLite, password hashing,
signing keys, and operational commands. The review deliberately separated the
mock engine from the strict path because the README explicitly limits the mock
engine and `serve` command to development use.

The code evidence was then compared with primary or authoritative guidance:
OpenID Connect Core, OAuth 2.0 Security BCP (RFC 9700), official SQLite backup
and WAL documentation, official Go security and `net/http` documentation, and
OWASP authentication/password-storage guidance. Local Markdown captures make
the review reproducible if the web pages later move.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Inventory the complete production system before
judging it, then ground findings in current protocol, runtime, persistence, and
authentication guidance.

**Inferred user intent:** Avoid shipping based only on passing unit tests or
feature claims; establish a defensible production baseline tied to actual code
and recognized requirements.

### What I did

- Counted 115 Go source files and approximately 15,481 Go lines under `cmd/`, `internal/`, and `pkg/`.
- Mapped the CLI entrypoint, mock and strict engines, public embedding package, Fosite adapter, storage interfaces, memory/SQLite stores, admin service, authentication, signing keys, consent, sessions, discovery, and health routes.
- Inspected all SQLite migrations and both domain and Fosite persistence paths.
- Searched for routes, server construction, randomness, process exits, environment access, secrets, TODO markers, and test/fuzz/benchmark coverage.
- Used web search to locate authoritative sources, then captured nine documents with `defuddle parse ... --md -o ...` under `sources/`.
- Compared Argon2id parameters, OAuth redirect/PKCE/refresh requirements, live SQLite backup semantics, Go testing/security guidance, and HTTP server controls with the implementation.

### Why

- An identity provider crosses browser, protocol, credential, cryptographic, persistence, and operational trust boundaries; no single test suite covers all of them.
- RFC 9700 is the current OAuth Security Best Current Practice and changes the appropriate baseline from older OAuth-only assumptions.
- SQLite's WAL behavior directly determines whether a file-copy backup is complete.
- A production review needs source snapshots in the ticket so future engineers can reproduce the reasoning without repeating discovery.

### What worked

- Defuddle produced readable Markdown captures totaling about 674 KB and preserved useful headings/anchors.
- Existing code clearly separates the mock and Fosite paths and already includes strong primitives: exact redirect matching, S256 PKCE, Argon2id, opaque hashed handles, server-side consent, persistent signing keys, CSRF, and stable audit reason codes.
- The existing 168 tests provide broad behavior coverage for both engines and the two storage implementations.

### What didn't work

- The first sandboxed Defuddle call failed exactly with `Error: fetch failed`. Re-running with approved Defuddle network access succeeded.
- `go list ./...` initially failed with `error obtaining VCS status: exit status 128`; `GOWORK=off`/`-buildvcs=false` isolated that first inventory run. Later Go commands used the normal `go.work` as requested.
- The repository contains no `.github` workflow files, so there was no CI definition to compare with the documented release gate.

### What I learned

- The advertised production API is `pkg/embeddedidp`, but its `Options` type embeds interfaces and policy types from four `internal/` packages. The only durable store is also internal. This makes the documented external embedding flow uncompilable from another Go module.
- Production construction checks HTTPS issuer, secure cookies, a 32-byte secret, persistent storage, clients, and an active key, but does not require a real audit sink or rate limiter.
- `CookieConfig.SameSite` is public but never read; cookies are hard-coded to Lax.
- The product's backup command copies the main database file with `io.Copy`, which is not a consistent live backup in WAL mode.
- Multi-row/token/key/admin state transitions are assembled from independent SQLite writes without transactions.
- Password policy defines a minimum length, but user creation/password reset only reject empty passwords; `MustChangeAtLogin` is returned by authentication but ignored by the authorization handler.
- The fixed-window limiter uses `RemoteAddr` including its ephemeral TCP port and never evicts stale buckets.

### What was tricky to build

- The repository presents two very different products under one binary: a deliberately permissive fault-injection mock and a strict provider. Findings had to be attributed to the correct engine so mock behavior was not mislabeled as a production vulnerability.
- The Go workspace replaces Glazed and go-go-goja locally while the module Makefile uses `GOWORK=off`. I treated the workspace dependencies as build inputs but kept the code review and commits scoped to `tiny-idp/`.

### What warrants a second pair of eyes

- Validate the intended distribution model. If tiny-idp will only ever be compiled inside this repository, the `internal/` public-API blocker has a different product impact; it still contradicts the current README's external embedding example.
- Confirm expected reverse-proxy topology and whether the application, proxy, or tiny-idp owns trusted-client-IP extraction, TLS policy, request limits, and server timeouts.
- Review whether the product requires per-client token TTLs: the domain stores them, but Fosite configuration currently uses global TTLs.

### What should be done in the future

- Keep the captured standards synchronized when changing OAuth/OIDC behavior or release gates.
- Add explicit CI workflows after deciding the supported Go version, CGO platform matrix, security scans, race cadence, fuzz cadence, and conformance policy.

### Code review instructions

- Start at `pkg/embeddedidp/options.go`, then follow `pkg/embeddedidp/provider.go` into `internal/fositeadapter/provider.go`.
- Continue through `internal/storage/interfaces.go`, `internal/store/sqlite/store.go`, `internal/fositeadapter/sqlstore.go`, and the embedded migrations.
- Read `sources/rfc-9700-oauth-security-bcp.md`, `sources/sqlite-online-backup-api.md`, and `sources/owasp-authentication-cheat-sheet.md` alongside the relevant findings.

### Technical details

Representative evidence commands:

```text
rg -n 'HandleFunc|ListenAndServe|http.Server|Shutdown' cmd internal pkg -S
rg -n 'NoopSink|AllowAllRateLimiter|RemoteAddr|SameSite|SecretKey' cmd internal pkg -S
rg -n 'BeginTx|io.Copy|INSERT OR REPLACE|ExecContext' internal/store internal/fositeadapter internal/admin -S
```

## Step 3: Build AST analysis, live probes, fuzz harnesses, and runtime instrumentation

This step converted the highest-risk hypotheses into repeatable tools. The
ticket now contains a typed `go/analysis` multichecker with analysistest
fixtures, a live-WAL backup reproduction, an isolated external-module compile
probe, parser fuzz targets, a static surface inventory, and a production-mode
runtime probe plus NDJSON analyzer.

The runtime probe provisions a temporary SQLite provider, creates a real client
and Argon2id user, performs Authorization Code + PKCE login, token exchange,
userinfo, and refresh, then sends 40 bounded concurrent reads. This creates a
useful regression baseline without changing production code or requiring a
public network listener.

### Prompt Context

**User prompt (verbatim):**

> write static analysis tools with go ast/analysis since you have high quality tooling, might as well use it.

> also you should now be able to operate with go.work normally, no need to make your own cache

> you can of course also write instrumentation code (or even ebpf hooks or stuff like that) to get runtime metrics and logs out of tiny-idp and then write tooling to analyze the results too, to see what's going on

**Assistant interpretation:** Upgrade the audit from text searches to typed
AST analysis, use the normal workspace, and collect/analyze real runtime
evidence rather than relying only on static inspection.

**Inferred user intent:** Make the review technically deep, reproducible, and
useful after this one session, with tooling capable of catching regressions and
explaining actual behavior.

**Commit (code):** `54fcbcf` — "audit(tinyidp): add production review research and probes"

### What I did

- Added `scripts/auditlint`, a `multichecker.Main` command with nine repository-specific analyzers.
- Added analysistest fixtures for internal-API leakage, ignored CSPRNG errors, zero-value HTTP serving, unused public configuration, and missing transaction boundaries.
- Added `external-api-smoke.sh`, which creates an isolated temporary module and compiles the documented production import path.
- Added `sqlite-backup-probe.go`, which forces WAL mode, commits a client in the WAL, invokes `admin.CreateSQLiteBackup`, and checks the copied database.
- Added three native Go fuzz targets for issuer parsing, production redirect validation, and Argon2id encoded-hash parsing.
- Added `runtime-probe` and `runtime-analyze`, including optional CPU/heap profiles, serialized NDJSON events, Go runtime metrics, SQL pool stats, route latency distributions, audit counts, and bounded concurrency using `errgroup`.
- Captured analyzer diagnostics in `various/auditlint.txt` and runtime evidence in `various/runtime/`.

### Why

- Type-aware analysis can prove that public signatures depend on internal types and distinguish actual `crypto/rand.Read` calls from similarly named functions.
- A deterministic WAL reproduction is stronger than a theoretical warning about file-copy backups.
- An external-module compilation test exercises the Go visibility rule exactly as a consumer experiences it.
- Runtime measurements identify capacity-sensitive behavior such as Argon2id memory cost and distinguish transient HTTP goroutines from leaks.

### What worked

- The final analyzer test suite passed for every tooling package.
- The analyzer reported the `embeddedidp.Options` internal-package leak, two ignored CSPRNG errors, zero-value `ListenAndServe`, silent no-op/allow-all defaults, port-bearing rate keys, unused `SameSite`, ignored audit delivery errors, non-transactional state transitions, and raw SQLite copying.
- The external module failed with `use of internal package github.com/manuel/tinyidp/internal/store/sqlite not allowed`, exactly confirming the packaging blocker.
- The backup probe printed `CONFIRMED: backup opens successfully but omits a committed client stored in the source WAL` after observing an 8,272-byte WAL.
- The runtime probe completed 45 measured HTTP operations with zero errors and emitted nine audit events.
- After explicitly closing idle connections, goroutines returned from 19 to 19, so the earlier +7 delta was an instrumentation artifact rather than evidence of a leak.
- A full strict login allocated about 67.9 MB and took about 82 ms, consistent with the configured 64 MiB Argon2id cost. Token exchange was about 6.3 ms, refresh about 4.8 ms, and bounded read routes remained sub-millisecond in this local probe.

### What didn't work

- The first normal-cache test inside the restricted shell failed with `open /home/manuel/.cache/go-build/...: read-only file system`. Running approved Go commands with the normal cache outside the filesystem sandbox resolved it; no custom cache remains part of the workflow.
- The first analyzer analysistest run failed because two analyzers shared a fixture containing each other's `// want` directives. Splitting the fixture into `randcheck` and `httpcheck` made expectations analyzer-specific.
- The external probe first surfaced missing `go.sum` entries, then `go: updates to go.mod needed`; copying the repository sums and using `-mod=mod` in the temporary isolated module exposed the intended internal-package error.
- `go tool pprof` failed with `go: no such tool "pprof"`. CPU and heap profiles were still captured, but this Go installation cannot render them without adding a pprof tool.
- The first runtime snapshot showed seven additional goroutines because the probe sampled before closing idle HTTP connections. The probe now closes them and waits briefly; the refreshed result has zero goroutine delta.

### What I learned

- The strongest current runtime cost is intentionally password hashing, not OIDC token/JWKS processing. Production concurrency limits must account for roughly 64 MiB of Argon2id working memory per simultaneous verification, plus process overhead.
- SQLite opened two idle connections under the bounded read workload with zero wait count. This is a smoke result, not evidence about multi-process or write contention.
- The strict flow is functionally healthy under the exercised happy path, but happy-path performance does not reduce the severity of the packaging, backup, atomicity, audit, and operational-control blockers.

### What was tricky to build

- A production provider requires an HTTPS issuer and secure cookies, while `httptest.Server` itself is HTTP. The probe directly targets the handler and manually returns the Secure CSRF cookie, preserving production-mode validation without pretending the test transport is the public issuer.
- The analyzer's atomicity rule is intentionally structural and repository-specific. It reports multiple mutating calls or looped mutations without `Begin`/`BeginTx`; each diagnostic still requires human confirmation of whether the operation is one logical transaction.
- Profiles are binary evidence and useful for future pprof analysis, but the current Go distribution lacks the pprof subtool. The Markdown runtime summary therefore relies on runtime metrics and explicit request timing.

### What warrants a second pair of eyes

- Review the analyzer's audit-delivery policy: production may choose fail-open asynchronous delivery, bounded buffering, or fail-closed behavior for selected events, but silently ignoring every sink error needs an explicit decision.
- Review the runtime probe's load mix and add deployment-specific write/login concurrency after fixing SQLite transaction and backup behavior.
- Inspect the captured CPU and heap profiles on a workstation with pprof before setting capacity targets.

### What should be done in the future

- Promote the analyzers that correspond to accepted invariants into CI, with narrowly documented suppressions where needed.
- Convert the external-consumer probe from an expected failure to a positive compile-and-flow test after the public API is redesigned.
- Run longer fuzz campaigns and preserve any minimized crashing corpus as ordinary regression tests.
- Add eBPF or system-level tracing only if in-process profiles and SQL/runtime metrics cannot explain production-like contention or syscall latency.

### Code review instructions

- Read `scripts/README.md`, then inspect `scripts/auditlint/main.go` and its analysistest fixtures.
- Run `go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/...`.
- Run the external API and SQLite backup probes; both currently confirm known blockers while exiting successfully as audit tools.
- Inspect `various/runtime/summary.md` and rerun the documented runtime probe after any authentication, SQLite, or Fosite change.

### Technical details

Key outputs:

```text
EXPECTED: external production embedding is blocked by Go internal-package visibility
CONFIRMED: backup opens successfully but omits a committed client stored in the source WAL
Requests observed: 45
Audit events emitted: 9
/sched/goroutines:goroutines delta: 0
/gc/heap/allocs:bytes delta: ~67.9 MB
```

## Related

- [Production readiness architecture and code review](../design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)
- [Ticket task list](../tasks.md)
