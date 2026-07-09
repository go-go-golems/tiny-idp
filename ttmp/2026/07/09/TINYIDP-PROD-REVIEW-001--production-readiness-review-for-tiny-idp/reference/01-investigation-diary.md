---
Title: Investigation diary
Ticket: TINYIDP-PROD-REVIEW-001
Status: complete
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
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/security-invariants-probe/main.go
      Note: Reproduces password, lockout, key-validation, control-default, and database-permission gaps (commit ca40c40)
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/gosec-product.json
      Note: Product-only gosec findings used for manual triage
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/govulncheck-go1.26.1.txt
      Note: Reachable vulnerability report for the active Go 1.26.1 toolchain
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/govulncheck-go1.26.5.txt
      Note: Patched-toolchain report isolating remaining go-jose vulnerabilities
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/various/runtime/summary.md
      Note: Measured runtime baseline for the strict happy path (commit 54fcbcf)
ExternalSources: []
Summary: Chronological evidence, commands, failures, decisions, and verification for the tiny-idp production-readiness review.
LastUpdated: 2026-07-09T17:31:17-04:00
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

## Step 4: Run the full verification and security gate

This step moved from targeted evidence to broad release-candidate validation.
The normal and race-enabled test suites, build, vet, pinned project lint,
Glazed lint, rebuilt Staticcheck, gosec, govulncheck, short native fuzz
campaigns, and a patched-Go compatibility run were all exercised. Results were
separated into tool failures, scanner false positives, and confirmed ship
blockers.

The step also added a dedicated security-invariant probe after static review
showed several gaps that deserved runtime proof. It reproduced permissive
database permissions, ineffective minimum-password policy, expired-key
acceptance, optional production controls, and lost concurrent lockout updates.

### Prompt Context

**User prompt (verbatim):**

> continue

> keep a frequent and detailed diary, btw. Backfill all that is necessary (read the skill for how to format it)

**Assistant interpretation:** Resume the interrupted verification work, keep
the diary current using the strict diary format, and backfill every material
command, result, failure, and conclusion before proceeding.

**Inferred user intent:** Preserve enough chronological evidence that another
engineer can reproduce the ship decision and understand how each finding was
confirmed or dismissed.

**Commit (code):** `ca40c40` — "audit(tinyidp): add security invariant verification"

### What I did

- Recovered the interrupted parallel verification call instead of rerunning completed work.
- Ran `go build ./...`, `go test ./... -count=1`, `go vet ./...`, and `go test -race ./... -count=1` through the normal workspace.
- Ran pinned `make lint`, including golangci-lint v2.12.2 and the Glazed analyzer.
- Rebuilt Staticcheck v0.6.1 with the active Go toolchain and ran it across `./...`.
- Ran gosec twice: an initial broad scan and a product-only `cmd/internal/pkg` scan, preserving both JSON reports.
- Ran govulncheck with Go 1.26.1 and again with Go 1.26.5, preserving both reports.
- Ran the complete tests with `GOTOOLCHAIN=go1.26.5` without changing repository metadata.
- Ran each parser fuzz target for five seconds with two workers.
- Added and ran `scripts/security-invariants-probe`, then brought all review tooling back through tests and pinned lint.

### Why

- Passing unit tests is necessary but insufficient for an internet-facing authorization server.
- Race, fuzz, dependency-reachability, typed lint, and runtime invariant checks each cover different failure classes.
- Testing Go 1.26.5 independently distinguishes outdated-runtime vulnerabilities from remaining dependency vulnerabilities.
- Reproducing lockout and filesystem behavior prevents ambiguous severity judgments based only on source patterns.

### What worked

- `go build ./...`, full tests, vet, the full race suite, rebuilt Staticcheck, and final `make lint` all passed.
- The complete suite passed under both the active Go 1.26.1 toolchain and patched Go 1.26.5.
- Fuzzing completed without crashes:
  - issuer parser: 125,089 executions, 118 total interesting inputs;
  - production redirect validator: 110,502 executions, 119 total interesting inputs;
  - Argon2id hash parser: 154,305 executions, 38 total interesting inputs.
- Product-only gosec inspected 80 files/9,362 lines and emitted 15 findings for human triage.
- The Go 1.26.1 govulncheck found 14 reachable vulnerabilities: 12 standard-library issues and two go-jose/v3 issues.
- Re-running on Go 1.26.5 removed all reachable standard-library results; the two go-jose/v3 findings remained.
- `go mod why` traced `go-jose/v3` through `internal/fositeadapter -> ory/fosite -> go-jose/v3`; v3.0.5 is available while the graph selects v3.0.3.
- The invariant probe produced these exact confirmations:

```text
OBSERVED: SQLite database mode under umask 000 is 0644
CONFIRMED: SQLite store creation does not enforce owner-only permissions
CONFIRMED: one-character password is accepted; MustChangePassword=true
CONFIRMED: production construction accepts an expired active key plus nil audit and rate-limit controls
CONFIRMED: concurrent failed-login accounting lost updates in round 0: attempts=5 stored_count=4 locked=false
```

### What didn't work

- The previous parallel tool wrapper was interrupted after running for an implausible reported duration. `functions.wait` recovered completed exit codes: build, test, and vet were all successful. The full test result claimed `wall_time_seconds=7760.75`, while individual package timings totaled seconds; this was treated as a wrapper timing anomaly.
- The installed Staticcheck binary failed before analysis with:

```text
file requires newer Go version go1.26 (application built with go1.25)
module requires at least go1.25.11, but Staticcheck was built with go1.25.3
```

  Running the same Staticcheck version through `go run ...@v0.6.1` rebuilt it with the active toolchain and passed cleanly.
- The initial gosec scan included auditlint `testdata` and reported an SSA type error for the deliberately internal-package fixture. A second product-only scan removed this fixture contamination.
- The first `make lint` run found four issues only in review tooling: two unchecked temporary-directory removals, an incomplete `runtime/metrics.ValueKind` switch, and an untagged error switch. All four were fixed; the second run reported `0 issues` and Glazed lint passed.
- Adding the security probe as a second `main` file in the scripts root caused `main redeclared in this block`. Moving it into `scripts/security-invariants-probe/` resolved the package collision; tests and lint then passed.
- Parallel fuzz commands appeared to exceed their five-second windows at the wrapper level, but each process's own output reported a normal 5.0–5.2 second fuzz campaign and clean exit.

### What I learned

- The repository is functionally healthy under its current tests but is not shippable on the active dependency/toolchain graph.
- Go 1.26.5 is source-compatible and removes the reachable standard-library vulnerability set; release automation should select at least that patch level.
- `github.com/go-jose/go-jose/v3@v3.0.3` remains reachable through Fosite on patched Go. Govulncheck identifies a parsing DoS fixed in v3.0.4 and a JWE-related panic fixed in v3.0.5.
- Most gosec cookie/open-redirect findings are false positives after manual review: strict cookie security is supplied through a runtime boolean, and both redirect sites first perform exact registered-URI checks. The zero-timeout server finding is confirmed, although `serve` is documented as non-production.
- The gosec hardcoded-credential report points to mock debug output and is not a strict-engine credential exposure.
- The Argon2id int-to-uint32 warnings require defensive length bounds for corrupted/untrusted encoded hashes, but they are not realistic integer wrap paths without multi-gigabyte decoded inputs.
- Lockout counters need a transactional/atomic store operation; a mutex around each independent SQLite method cannot protect the read-modify-write sequence in `PasswordService`.

### What was tricky to build

- Toolchain versioning had three layers: the module's `go 1.25.11` directive, the active auto-selected Go 1.26.1 toolchain, and pinned linters built with other patch releases. Scanner failures had to be attributed to the scanner binary rather than the code.
- Gosec taint diagnostics required control-flow review. Both reported redirects are guarded by exact client registration checks, so treating them as exploitable open redirects would have been incorrect.
- The lockout race is timing-sensitive in theory. The probe uses a simultaneous start barrier and fast test-only Argon2id parameters; it reproduced lost updates in the first round while leaving the normal runtime probe on production password parameters.
- A permissive umask test temporarily changes process-wide state. The probe restores the prior umask immediately after opening its isolated temporary database and runs no concurrent goroutines during that interval.

### What warrants a second pair of eyes

- Confirm the acceptable patched toolchain floor and whether release builds should pin exactly Go 1.26.5 or a later supported security patch.
- Validate that directly requiring `go-jose/v3@v3.0.5` is compatible with Fosite v0.49.0, or select another Fosite release/strategy after tests and conformance.
- Re-review all gosec suppressions before codifying them in configuration; false positives should have narrow, documented rationale rather than broad rule disablement.
- Decide whether SQLite file permissions are enforced by the library, CLI, deployment umask, or a pre-created secret volume. The current code does not own the invariant.

### What should be done in the future

- Make build/test/race/lint/govulncheck/auditlint/external-consumer/backup probes explicit CI jobs.
- Add persistent fuzz corpora or scheduled longer fuzzing after the remediation work begins.
- Add a release manifest/SBOM recording Go toolchain, CGO/SQLite, Fosite, and go-jose versions.
- After atomic lockout remediation, keep the invariant probe as a positive test requiring all five simultaneous failures to be counted and the account locked.

### Code review instructions

- Start with the raw scanner reports under `various/govulncheck-*.txt` and `various/gosec-product.json`.
- Confirm dependency selection with `go mod why -m github.com/go-jose/go-jose/v3` and `go list -m -u github.com/go-jose/go-jose/v3`.
- Run the invariant probe command from `scripts/README.md`; it should reproduce all four confirmed gaps until fixes land.
- Validate release-toolchain compatibility with `GOTOOLCHAIN=go1.26.5 go test ./... -count=1`.
- Run `make lint` and `go test -race ./... -count=1` after modifying any review tool or production concurrency path.

### Technical details

Verification commands:

```text
go build ./...
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
make lint
go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
gosec -fmt=json -out .../various/gosec-product.json ./cmd/... ./internal/... ./pkg/...
govulncheck ./...
GOTOOLCHAIN=go1.26.5 go test ./... -count=1
GOTOOLCHAIN=go1.26.5 govulncheck ./...
```

## Step 5: Synthesize the intern-oriented production review

This step converted the source, standards, tests, scanners, and probe evidence
into one coherent architecture and production-readiness document. The report is
designed to work at two altitudes: maintainers can act on its priority/acceptance
tables, while a new intern can read from the package map through complete
authorization, storage, authentication, refresh, key, and admin flows before
encountering the findings.

The report makes the ship decision explicit: the current commit is a no-go, but
the strict engine has enough sound protocol and security primitives that a
phased remediation is practical. Findings are separated from gosec false
positives and documented mock behavior so the release decision is not inflated
by irrelevant diagnostics.

### Prompt Context

**User prompt (verbatim):** (see Step 4)

**Assistant interpretation:** Keep the detailed diary current while producing
the requested comprehensive, technically clear review for an unfamiliar
engineer.

**Inferred user intent:** Deliver a durable engineering reference, not merely a
short audit list, so a new contributor can understand and safely implement the
production-hardening work.

**Commit (code):** `c387926` — "docs(ticket): add tiny-idp production readiness review"

### What I did

- Wrote a 1,277-line / 7,734-word primary review with stable sections and line-level repository evidence.
- Led with a no-go decision, strengths, six P0 blocker families, and a concrete definition of ready.
- Explained the engine split, trust boundaries, package ownership, endpoint surface, startup, Authorization Code + PKCE, sessions/CSRF, passwords, consent/claims, refresh, keys, admin operations, and SQLite model.
- Added an ASCII topology diagram, flow pseudocode, package/endpoint/gap/runtime tables, public API sketches, transaction/backup/login algorithms, and external API references.
- Documented P0/P1/P2 findings with observed evidence, impact, recommended fix, and acceptance conditions.
- Reconciled gosec output so false-positive cookie/open-redirect/debug findings were not mislabeled as production vulnerabilities.
- Added six compact decision records and a five-phase, file-oriented implementation plan.
- Added always-on/release test matrices, operational ship checklist, alternatives, open questions, and repository/ticket/external references.

### Why

- Findings without system orientation force an intern to rediscover control flow and trust boundaries before fixing anything.
- Severity is useful only when tied to executable acceptance criteria.
- The current pre-release API is already unusable externally, so the design needs to say explicitly that direct replacement is preferable to a compatibility layer.
- Production readiness includes release engineering, restore, proxy/server, key/secret, monitoring, and incident-response responsibilities—not only OAuth endpoint correctness.

### What worked

- The report combines prose, bullets, tables, diagrams, pseudocode, API sketches, decision records, and file references as requested.
- Major claims are anchored in the same source/probe evidence recorded in earlier diary steps.
- The design preserves existing strengths such as Fosite ownership, exact redirects, S256 PKCE, Argon2id, hashed session handles, CSRF, and strict discovery rather than proposing a rewrite.
- The proposed package design exposes no `internal/` types and explicitly avoids an unnecessary compatibility shim.
- `git diff --check` passed and no template placeholders remain.

### What didn't work

- N/A. This step produced documentation only and introduced no command or render failure.

### What I learned

- The highest-leverage remediation order is dependency/toolchain patching, public API repair, persistence/backup atomicity, authentication controls, then operational/release proof.
- SQLite remains viable for a one-active-process product if the supported envelope is explicit and tested; it should not be presented as transparent active/active storage.
- The 64 MiB Argon2id setting is a security strength and a capacity constraint. The report therefore recommends bounding concurrent password work rather than weakening the hash.
- Several current configuration fields (`SameSite`, per-client TTLs, must-change) are false contracts: they can be set or displayed without changing effective behavior.

### What was tricky to build

- The review needed to be unequivocal about blockers without obscuring the healthy protocol foundation. Separating confirmed production findings, mock-only behavior, scanner limitations, and unsupported-but-honestly-omitted features kept the decision precise.
- Fosite and domain persistence overlap. The report calls out that domain refresh-token invariants do not automatically prove the strict Fosite SQL path, preventing a misleading conclusion from the shared store suite.
- The host owns TLS and `http.Server`, while the library owns identity semantics. The proposed contract keeps that split but makes proxy trust, timeouts, limits, shutdown, and preflight executable release requirements.

### What warrants a second pair of eyes

- Verify that the product owner agrees with the proposed single-active-node SQLite envelope.
- Confirm whether per-client TTLs, MFA/recovery, logout/revocation/introspection, and KMS/HSM custody are v1 requirements or intentionally deferred scope.
- Review the direct public-package reorganization and atomic store API before implementation begins.
- Validate severity and acceptance criteria for audit delivery semantics and database permission enforcement against the actual deployment platform.

### What should be done in the future

- Update this report as remediation decisions are accepted or superseded; do not let it become a stale one-time audit.
- Convert each accepted P0/P1 acceptance criterion into a ticket task and automated regression.
- Record the final release artifact hashes and conformance/restore evidence in a follow-up release record.

### Code review instructions

- Read the executive summary and finding register first for release scope.
- New engineers should then read `What the System Is`, `Architecture for a New Engineer`, `Core Runtime Flows`, and `Storage Model` in order.
- Maintainers planning work should focus on `Proposed Production Architecture`, decision records, phased implementation, and test strategy.
- Cross-check raw evidence in the diary, `various/`, and source files linked in the report.
- Run `rg -n '^#{1,4} ' <design-doc>` to inspect the complete stable outline.

### Technical details

Document checks:

```text
wc -lw .../design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md
1277 lines, 7734 words

git diff --check
clean

template placeholder search
no TODO/FIXME/HTML template placeholders
```

## Related

- [Production readiness architecture and code review](../design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)
- [Ticket task list](../tasks.md)

## Step 6: Validate ticket metadata and prepare the delivery artifact

This step closed the documentation bookkeeping and reran the affected quality
gates after making generated review artifacts valid docmgr documents. It also
records the first validation failure instead of silently presenting only the
eventual clean result.

### Prompt Context

**User prompt (verbatim):** “keep a frequent and detailed diary, btw. Backfill
all that is necessary (read the skill for how to format it)”

**Assistant interpretation:** Preserve a tiny-step account of final validation,
including exact failures, fixes, commands, and intentional warnings before the
review is delivered.

**Inferred user intent:** Make the production decision reproducible and leave a
continuation point that does not require reverse-engineering what happened at
the end of the audit.

**Commit (code):** pending at the start of this step.

### What I did

- Related the primary report and diary to the principal production files and
  raw evidence with `docmgr doc relate`.
- Updated the ticket index, task list, and changelog with the no-go verdict,
  evidence snapshot, review commits, and completed work.
- Ran `docmgr doctor --ticket TINYIDP-PROD-REVIEW-001` and preserved both the
  initial failure and the post-fix result below.
- Added full docmgr frontmatter to `scripts/README.md`.
- Changed `scripts/runtime-analyze` so every regenerated `summary.md` receives
  valid reference-document frontmatter and an updated timestamp.
- Regenerated the runtime summary, tested every ticket Go package, ran the
  complete repository lint target, and checked the diff for whitespace errors.

### Why

- A ticket is not complete if its primary artifacts fail the repository's own
  documentation validator.
- Fixing only the generated `summary.md` would be temporary; the analyzer must
  preserve the invariant on every rerun.
- The analyzer changed after the earlier verification checkpoint, so targeted
  tests and the full lint gate needed to be repeated before delivery.

### What worked

- The second doctor run exited zero and reported no errors.
- The affected package and all ticket script packages pass `go test`.
- `git diff --check` is clean.
- golangci-lint reports `0 issues`, and Glazed lint passes.
- The regenerated runtime summary retains the same measured evidence while now
  participating correctly in docmgr validation.

### What didn't work

The first doctor run found two errors because Markdown artifacts inside the
ticket were missing YAML frontmatter:

```text
scripts/README.md: missing frontmatter
various/runtime/summary.md: missing frontmatter
```

After adding frontmatter and regenerating the summary, doctor exited zero with
two warnings:

```text
missing_numeric_prefix — scripts/README.md
missing_numeric_prefix — various/runtime/summary.md
```

These are intentional names. `README.md` is the conventional scripts entry
point, and `summary.md` is the stable generated output consumed by the runtime
probe workflow. Renaming either would reduce discoverability or make the tool's
documented output path misleading. There are no remaining doctor errors.

### What I learned

- `docmgr doctor` validates incidental Markdown under `scripts/` and
  `various/`, not only documents under `design-doc/` and `reference/`.
- Generated Markdown must own its metadata in the generator; a one-time manual
  repair is not durable.
- The ticket validator distinguishes actionable errors from convention
  warnings, allowing stable artifact names where their role justifies them.

### What was tricky to build

- The runtime summary timestamp must be generated at analysis time without
  changing the deterministic metrics calculations or source event stream.
- The generated YAML needed quoted prose fields and stable topic/doc-type
  values while remaining readable as a standalone report.

### What warrants a second pair of eyes

- Confirm the team prefers retaining stable `README.md` and `summary.md` names
  over eliminating the two non-blocking numeric-prefix warnings.
- Review the doc relations when remediation moves or replaces the public API,
  storage, backup, or password-service files.

### What should be done in the future

- Keep the generated frontmatter contract covered when changing
  `runtime-analyze`.
- Rerun doctor after adding any future Markdown evidence to this ticket.
- Update the runtime summary and report baseline after each blocker family is
  remediated.

### Code review instructions

- Run `docmgr doctor --ticket TINYIDP-PROD-REVIEW-001`; accept only the two
  documented numeric-prefix warnings.
- Run `go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/...`.
- Run `make lint` from the repository root.
- Inspect the beginning of `various/runtime/summary.md` and confirm it contains
  a complete `TINYIDP-PROD-REVIEW-001` reference frontmatter block.

### Technical details

Final pre-delivery commands:

```text
git diff --check
go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/runtime-analyze \
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/...
make lint
docmgr doctor --ticket TINYIDP-PROD-REVIEW-001
```

Observed result:

```text
ticket script tests: PASS
golangci-lint: 0 issues
Glazed lint: PASS
docmgr doctor: exit 0, 0 errors, 2 documented naming warnings
```

## Related

- [Production readiness architecture and code review](../design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)
- [Runtime probe summary](../various/runtime/summary.md)
- [Ticket task list](../tasks.md)

## Step 7: Deliver the review bundle to reMarkable

This step performed the required safe delivery sequence: first resolve the
bundle and destination without rendering or uploading, then render and upload
the same inputs non-interactively. The local ticket remains the evidence system
of record; the reMarkable PDF is the reading copy.

### Prompt Context

**User prompt (verbatim):** “Store in the ticket and the nupload to remarkable.”

**Assistant interpretation:** Preserve the complete review locally and publish
a readable PDF bundle to a dated, ticket-specific reMarkable folder.

**Inferred user intent:** Make the technical production review available for
focused offline reading without losing the repository-native source and raw
evidence.

**Commit (code):** `d3c5e26` — "docs(ticket): finalize production review evidence"

### What I did

- Selected the primary architecture/code/implementation review and the measured
  runtime summary as the reading bundle.
- Ran the exact upload command with `--dry-run`, `--non-interactive`, a stable
  document name, and a dated ticket destination.
- Verified that both source paths, the intermediate PDF name, and remote folder
  resolved as intended.
- Repeated the command without `--dry-run` and captured the uploader's success
  response.
- Did not invoke reMarkable account, status, or post-upload listing commands.

### Why

- The primary report is the coherent intern-oriented reading path; the runtime
  summary is a compact empirical appendix.
- Scanner JSON, profiles, source captures, scripts, and the diary are more
  useful in their native ticket form and would make the device bundle noisy.
- A dry run prevents destination/name/input mistakes before external state is
  changed.

### What worked

- The dry run exited zero and included exactly the two intended Markdown files.
- Pandoc rendered the combined bundle successfully.
- The uploader returned an explicit `OK: uploaded` result for the requested
  destination.

### What didn't work

- N/A. Both the dry run and actual upload succeeded on the first attempt.

### What I learned

- The two-document bundle preserves the explanatory narrative and measured
  runtime baseline without overwhelming the reading copy with raw output.
- The stable ticket path maps cleanly to a matching dated reMarkable directory.

### What was tricky to build

- Delivery needed to happen after final report/frontmatter validation but before
  recording the upload result in this local-only diary. Excluding the diary from
  the device bundle avoids a circular re-upload solely to include its own
  delivery receipt.

### What warrants a second pair of eyes

- On-device, verify the table of contents, wide finding tables, ASCII diagrams,
  and long code/file references are comfortable to read in the default layout.

### What should be done in the future

- Upload a superseding review rather than overwriting this production decision
  after remediation; preserving the reviewed-state record makes deltas clear.
- If annotations are made on the device, do not use a forced overwrite workflow
  without first preserving them.

### Code review instructions

- Treat the repository ticket as authoritative for scripts, sources, raw
  scanner output, profiles, and the complete diary.
- Treat the reMarkable bundle as a convenience reading copy of the primary
  report and runtime appendix.

### Technical details

Dry-run result:

```text
DRY: bundle name=TINYIDP PROD REVIEW 001 Production Readiness Review
DRY: remote-dir=/ai/2026/07/09/TINYIDP-PROD-REVIEW-001
DRY: include design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md
DRY: include various/runtime/summary.md
DRY: pandoc <bundle> -> <tmp>/TINYIDP PROD REVIEW 001 Production Readiness Review.pdf
DRY: upload TINYIDP PROD REVIEW 001 Production Readiness Review.pdf -> /ai/2026/07/09/TINYIDP-PROD-REVIEW-001
```

Actual upload result:

```text
OK: uploaded TINYIDP PROD REVIEW 001 Production Readiness Review.pdf -> /ai/2026/07/09/TINYIDP-PROD-REVIEW-001
```

## Related

- [Production readiness architecture and code review](../design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)
- [Runtime probe summary](../various/runtime/summary.md)
- [Ticket task list](../tasks.md)
