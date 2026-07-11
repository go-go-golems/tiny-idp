---
Title: Implementation diary
Ticket: TINYIDP-PROD-IMPL-001
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
    - Path: repo://.github/workflows/ci.yml
      Note: Exact-toolchain build test lint race and vulnerability gates (commit a2c86a9)
    - Path: repo://.github/workflows/release-evidence.yml
      Note: SBOM checksum environment and provenance workflow (commit a2c86a9)
    - Path: repo://Makefile
      Note: Pinned vulnerability gate and toolchain-aware linter caches (commit a2c86a9)
    - Path: repo://go.mod
      Note: |-
        Phase 0 dependency and toolchain baseline
        Phase 0 Go toolchain and go-jose dependency baseline (commit a2c86a9)
    - Path: repo://internal/authn/password.go
      Note: Step 13 bounded fail-closed authentication (commit 7022e7d)
    - Path: repo://internal/fositeadapter/interaction.go
      Note: Server-owned canonical interaction implementation (commit aedff3c)
    - Path: repo://internal/fositeadapter/interaction_hardening_test.go
      Note: Mutation, replay, concurrency, consent, and revalidation evidence
    - Path: repo://internal/fositeadapter/provider.go
      Note: Steps 21-22 authorization state machine, endpoint hardening, and lifecycle coordination (commits aedff3c and 27c339e)
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Atomic SQLite Fosite authorization lifecycle and failpoints (commit aedff3c)
    - Path: repo://pkg/embeddedidp/options.go
      Note: First production API implementation target
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Context lifecycle and readiness API (commit e65ff53)
    - Path: repo://pkg/idp/contracts.go
      Note: Public policy authentication and readiness contracts (commit 0bcbf24)
    - Path: repo://pkg/idp/password.go
      Note: Step 13 public password policy and work metrics (commit 7022e7d)
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Public durable store contracts (commit e042a15)
    - Path: repo://pkg/sqlitestore/backup.go
      Note: Step 12 verified recovery implementation (commit 7cd13b4)
    - Path: repo://pkg/sqlitestore/backup_test.go
      Note: Step 12 recovery and fault gate evidence
    - Path: repo://pkg/sqlitestore/store.go
      Note: Public SQLite implementation (commit 24c9a92)
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md
      Note: Implementation design this diary tracks
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md
      Note: Step 18 research and assurance architecture deliverable
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/05-authorization-interaction-hardening-implementation-ledger.md
      Note: Step 19 detailed phase and task plan
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md
      Note: Durable 90-task phase ledger
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/various/phase3-password-load.json
      Note: Production Argon2 load evidence
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Strict parse, bearer transport, and injected clock analyzers (commit 34580db)
ExternalSources: []
Summary: Chronological implementation record for the production embedding API and release-hardening program.
LastUpdated: 2026-07-09T17:37:01.160328301-04:00
WhatFor: Preserving decisions, exact commands, failures, commits, verification, and continuation instructions across the multi-phase hardening effort.
WhenToUse: Read before resuming work, reviewing a phase, investigating a regression, or assembling release evidence.
---









# Implementation diary

## Goal

Record the implementation of every production-hardening phase in enough detail
that a new engineer can understand what changed, reproduce verification, and
continue from the last safe checkpoint without reconstructing prior reasoning.

## Step 1: Establish the long-lived implementation program

This step created a new execution ticket distinct from the completed production
review. The ticket turns the review's recommendations into ordered, checkable
work with explicit phase gates so progress can be tracked honestly across many
sessions and commits.

No production code changed in this step. The purpose was to establish the source
of truth for design, sequencing, diary evidence, and the final release decision
before dependency or API work begins.

### Prompt Context

**User prompt (verbatim):**

> cratea new docmgr ticket for that and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
>
>
> Then add detailed phases and for each phase, detailed tasks, so that we can keekp track of our progress over long periods of time, and then implement them one by one, and keep a detailed diarya s you work, committing ata appropriate intervals

**Assistant interpretation:** Create a durable implementation ticket, write and
deliver an intern-oriented technical guide, decompose the work into detailed
phase tasks, and then execute phases sequentially with verified code and diary
commits.

**Inferred user intent:** Move from a one-time production assessment to a
professional, reviewable hardening program that can survive interruptions and
eventually produce defensible release evidence.

**Commit (code):** N/A; this step creates documentation and tracking only.

### What I did

- Read the ticket-research, docmgr, diary, and reMarkable delivery workflows.
- Confirmed the working branch and preserved two unrelated untracked source
  directories belonging to `TINYIDP-PROD-001`.
- Created ticket `TINYIDP-PROD-IMPL-001` with design and diary documents.
- Added six ordered implementation phases, detailed tasks, explicit phase gates,
  and a final production-decision gate to `tasks.md`.
- Set Phase 0 as the current phase while keeping the release status blocked.

### Why

- The completed review explains why the current commit cannot ship; a separate
  ticket is needed to manage code changes and evidence over time.
- Phase gates prevent a long checklist from implying readiness while critical
  cross-cutting work remains unfinished.
- A diary established before code changes captures failures and decisions at the
  moment they occur rather than reconstructing them later.

### What worked

- `docmgr` created the ticket and both requested subdocuments successfully.
- Existing vocabulary accepted all selected topics.
- The repository was clean apart from the two known, unrelated untracked source
  directories, which remain untouched.

### What didn't work

- The first documentation checkpoint stopped before committing because the
  generated changelog ended with a blank line:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:6: new blank line at EOF.
  ```

  The failing command was `git diff --cached --check`. Removing only that final
  blank line made the generated file satisfy the repository whitespace gate.

### What I learned

- The production review already provides enough observed evidence to define the
  phase order without repeating discovery work.
- The implementation program needs separate gates for dependency safety, public
  construction, persistence invariants, authentication controls, operational
  health, and release proof; collapsing these would hide release risk.

### What was tricky to build

- Tasks must be granular enough to resume independently but not so small that
  bookkeeping overwhelms implementation. Each phase therefore combines concrete
  file-level tasks with one outcome-oriented acceptance gate.
- The public API and persistence work are coupled, but sequencing the API before
  transaction implementation provides a stable public package boundary while
  Phase 2 supplies the stronger semantics behind it.

### What warrants a second pair of eyes

- Confirm the organization agrees with the six-phase order and the proposed
  single-active-node SQLite support envelope before Phase 2 is finalized.
- Confirm whether hosted OpenID Foundation conformance and independent security
  review are mandatory release gates; this ticket currently treats both as
  mandatory.

### What should be done in the future

- Update this diary immediately after each meaningful implementation or
  verification step.
- Check tasks only when their acceptance evidence exists.
- Keep code commits focused and follow each with a documentation/bookkeeping
  checkpoint when a phase-level decision or result changes.

### Code review instructions

- Start with `tasks.md` and verify every phase has concrete work plus a distinct
  gate.
- Compare the task order with the source review's detailed finding register and
  phased plan.
- Run `docmgr task list --ticket TINYIDP-PROD-IMPL-001` to inspect parsed task
  state.

### Technical details

Ticket creation commands:

```text
docmgr ticket create-ticket --ticket TINYIDP-PROD-IMPL-001 --title "Production embedding API and release hardening" --topics oidc,go,testing,auth,architecture,research
docmgr doc add --ticket TINYIDP-PROD-IMPL-001 --doc-type design-doc --title "Production embedding API and release implementation guide"
docmgr doc add --ticket TINYIDP-PROD-IMPL-001 --doc-type reference --title "Implementation diary"
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- [Source production review](../../TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)

## Step 2: Write the architecture and implementation guide

This step converted the completed production review into a forward-looking
implementation manual. It explains the strict engine from the host HTTP boundary
through Fosite, authentication, domain/protocol storage, SQLite, keys, audit,
backup, and operations before specifying the replacement public API.

The document is written for an intern but remains precise enough to drive code
review. It distinguishes observed current behavior from proposed contracts and
ties each phase to files, pseudocode, failure cases, and executable exit
evidence.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Produce a detailed, clear, technical onboarding
and implementation guide with prose, bullets, diagrams, pseudocode, API sketches,
file references, phases, and long-term progress tracking.

**Inferred user intent:** Give a new contributor enough context to implement
production hardening safely without repeatedly rediscovering architecture or
the reasoning behind phase order.

**Commit (code):** N/A; this step writes design documentation only.

### What I did

- Re-read the current exported embedding options/provider, domain model, store
  interfaces, strict Fosite provider, password service, SQLite store, backup
  code, and prior production review.
- Wrote a 1,083-line / 5,611-word implementation guide.
- Added an intern reading path and glossary before introducing subsystem detail.
- Documented the engine split, route surface, package responsibilities, state
  model, Authorization Code + PKCE flow, and trust boundaries.
- Designed public `idp`, `idpstore`, `sqlitestore`, and `embeddedidp` boundaries.
- Added API sketches for construction, lifecycle, readiness, transactions, and
  atomic security operations.
- Added pseudocode for startup validation, failed-login atomicity, online
  backup, restore, password-work admission, and graceful hosting.
- Added seven accepted decision records, alternatives, security checklist,
  testing matrix, failure cases, release evidence packet, and open questions.
- Expanded all six phases with primary files, ordered work, and outcome gates.

### Why

- The production review explains defects; implementation needs a separate
  coherent target architecture and execution model.
- Moving types out of `internal/` without defining lifecycle, transaction, and
  ownership semantics would create a compilable but still unsafe API.
- New engineers need the flow and trust model before individual findings make
  sense.

### What worked

- The document contains no template placeholders, TODOs, or FIXMEs.
- `git diff --check` passes.
- Major recommendations are anchored to concrete repository files and the raw
  evidence in `TINYIDP-PROD-REVIEW-001`.
- The API sketches avoid Fosite and driver-specific types at the public boundary.
- Phase descriptions align with the 90-task execution ledger.

### What didn't work

- The first inventory command included a presumed `.github` directory that does
  not exist in this repository. `rg` reported the error twice because it was
  present in both path lists:

  ```text
  rg: .github: No such file or directory (os error 2)
  ```

  The useful repository output was still returned. CI configuration is therefore
  a Phase 0 addition rather than an existing workflow to modify.

- The first changelog update used `$ROOT/$D:Primary implementation guide` in
  zsh. The colon was interpreted as parameter-modifier syntax instead of the
  `path:note` separator, and docmgr rejected the corrupted argument:

  ```text
  Error: malformed --file-note value "/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp//home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.mdrimary implementation guide": expected 'path:note' (or 'path=note')
  ```

  Bracing both variables as `${ROOT}/${D}:...` preserves the literal colon and
  avoids zsh parameter modifiers.

- `docmgr changelog update` again wrote one blank line at EOF. `git diff
  --check` reported:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:14: new blank line at EOF.
  ```

  As in Step 1, removing the generated trailing blank line preserves the
  changelog content and restores the whitespace gate.

### What I learned

- The current repository has no `.github/` workflow baseline, so release CI and
  artifact provenance need explicit initial design rather than incremental
  edits to an existing pipeline.
- Public construction, store transactions, and production readiness form one
  contract: treating them as unrelated refactors would permit invalid partially
  hardened configurations.
- The current domain separation between profiles, password credentials, and
  account security state is valuable; the missing property is atomic transition
  ownership, not record consolidation.

### What was tricky to build

- The guide must explain Fosite's role without exporting Fosite as the product
  API. The package diagram and adapter decision make that boundary explicit.
- Phase 1 needs public store contracts before Phase 2 strengthens their
  implementation. The guide addresses this by defining final transaction and
  invariant shapes early while allowing implementation depth to land in Phase 2.
- Backup safety crosses SQLite snapshot semantics, filesystem permissions,
  `fsync`, read-only verification, and operator restore behavior; describing only
  file copying would miss most of the real contract.

### What warrants a second pair of eyes

- Review whether `Store` should expose both generic `Update` and named atomic
  methods, or whether named service interfaces should be narrower.
- Review dependency ownership and close semantics: the proposed default leaves
  an injected store owned by the host.
- Decide the outstanding audit durability, secret custody, Argon2id capacity,
  must-change-password, proxy, and revocation questions before their phases.

### What should be done in the future

- Keep the guide synchronized with accepted implementation decisions; mark a
  decision superseded rather than silently rewriting history.
- Add concrete public Go doc examples when Phase 1 types exist.
- Add links to phase-specific test evidence as each gate closes.

### Code review instructions

- Read `Current-State Architecture` before `Proposed Solution` and verify file
  references against the current code.
- Review every decision record for an unstated compatibility or ownership cost.
- Compare each phase with `tasks.md`; every guide action should have a durable
  checkbox and gate.
- Run `rg -n '^#{1,4} ' <guide>` to inspect the stable outline and
  `rg -n '<!--|TODO|FIXME' <guide>` to detect unfinished template content.

### Technical details

Document checks:

```text
wc -lw design-doc/01-production-embedding-api-and-release-implementation-guide.md
1083 lines, 5611 words

placeholder search
no matches

git diff --check
clean
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- [Source production review](../../TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)

## Step 3: Validate the implementation documentation

This step applied the repository's documentation quality gate after the guide,
relations, tasks, diary, and changelog were populated. Validation occurred
before delivery so the reMarkable reading copy is derived from a ticket whose
metadata and structure are internally consistent.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Make the implementation package durable and
validated before publishing or beginning code changes.

**Inferred user intent:** Avoid accumulating a long-running plan whose source
documents cannot be reliably found, related, or resumed through docmgr.

**Commit (code):** N/A; documentation validation only.

### What I did

- Ran the repository whitespace gate after correcting generated changelog EOF
  formatting.
- Ran `docmgr doctor --ticket TINYIDP-PROD-IMPL-001 --stale-after 30`.
- Checked the ticket validation task only after doctor returned successfully.

### Why

- reMarkable delivery should not be the first place malformed Markdown or
  metadata is discovered.
- A clean doctor result establishes a stable baseline before Phase 0 begins.

### What worked

- `git diff --check` returned no output and exited zero.
- Doctor returned one success finding and no errors or warnings:

  ```text
  ## Doctor Report (1 findings)

  ### TINYIDP-PROD-IMPL-001

  - ✅ All checks passed
  ```

### What didn't work

- N/A. The validator passed on the first run after the already-recorded
  changelog whitespace correction.

### What I learned

- The selected topics, document types, frontmatter, relations, and task syntax
  all match the repository vocabulary and docmgr conventions.

### What was tricky to build

- Validation itself was straightforward; the only sharp edge was ensuring the
  helper-generated changelog did not reintroduce the known trailing blank line.

### What warrants a second pair of eyes

- N/A for validator mechanics. Architecture decisions still require the review
  identified in Step 2.

### What should be done in the future

- Rerun doctor after every documentation checkpoint that adds Markdown or
  changes relations/frontmatter.

### Code review instructions

- Run the exact doctor command above and require `All checks passed` before
  publishing future guide revisions.

### Technical details

```text
git diff --check
docmgr doctor --ticket TINYIDP-PROD-IMPL-001 --stale-after 30
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)

## Step 4: Deliver the guide and phase ledger to reMarkable

This step published the stable reading copy after the documentation checkpoint
was committed and validated. The bundle contains the full intern-oriented guide
followed by the live 90-task phase ledger; raw implementation evidence and the
diary remain repository-native.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Upload the implementation guide to a dated,
ticket-specific reMarkable location using a safe preview before external state
changes.

**Inferred user intent:** Make the long-form design and execution plan available
for focused offline reading while preserving docmgr as the source of truth.

**Commit (code):** `f5fc49a` — "docs(ticket): add production implementation guide"

### What I did

- Selected the implementation guide and `tasks.md` as the reading bundle.
- Ran `remarquee upload bundle --dry-run` with a stable name, dated ticket
  folder, ToC depth 2, and non-interactive mode.
- Verified both input paths, render target, and remote destination in dry-run
  output.
- Ran the identical upload without `--dry-run` and captured the uploader's
  explicit success response.
- Checked the reMarkable delivery task after success.

### Why

- The guide explains why and how; the task ledger makes the long-term sequence
  usable during review.
- A dry run prevents an incorrect name, input set, or destination from mutating
  external state.
- Excluding the diary avoids a circular re-upload merely to include the diary's
  own upload receipt.

### What worked

- Dry run exited zero and selected exactly the two intended Markdown files.
- PDF rendering and upload succeeded on the first attempt.
- The uploader reported the exact expected dated destination.

### What didn't work

- Dry run and actual upload both succeeded without authentication retry.
  Afterwards, the changelog helper repeated its known trailing-newline behavior,
  and `git diff --check` reported:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:22: new blank line at EOF.
  ```

  Removing the final blank line restored the whitespace gate without changing
  the delivery record.

### What I learned

- The phase ledger renders cleanly as a second bundle document and keeps the
  implementation guide's narrative separate from live checkbox state.

### What was tricky to build

- The upload process temporarily returned a long-running execution session
  after the wrapper wait completed. Polling that existing session produced the
  final success line; no duplicate upload command was issued.

### What warrants a second pair of eyes

- Verify on-device readability of the wide package/ownership tables and ASCII
  sequence diagrams in the default layout.

### What should be done in the future

- Publish a superseding bundle when accepted architecture changes materially;
  do not force-overwrite a document that may contain annotations.

### Code review instructions

- Treat the local ticket as authoritative; the device PDF is a reading copy.
- Compare the uploaded bundle name/destination below with the requested ticket.

### Technical details

```text
DRY: bundle name=TINYIDP PROD IMPL 001 Implementation Guide
DRY: remote-dir=/ai/2026/07/09/TINYIDP-PROD-IMPL-001
DRY: include design-doc/01-production-embedding-api-and-release-implementation-guide.md
DRY: include tasks.md
DRY: pandoc <bundle> -> <tmp>/TINYIDP PROD IMPL 001 Implementation Guide.pdf
DRY: upload TINYIDP PROD IMPL 001 Implementation Guide.pdf -> /ai/2026/07/09/TINYIDP-PROD-IMPL-001

OK: uploaded TINYIDP PROD IMPL 001 Implementation Guide.pdf -> /ai/2026/07/09/TINYIDP-PROD-IMPL-001
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)

## Step 5: Implement the Phase 0 secure release baseline

This step repaired the known vulnerable release graph before changing the public
API. It kept Fosite at 0.49.0, upgraded only go-jose/v3, selected an exact patched
Go toolchain for module/release mode, and made those choices executable in local
targets and GitHub Actions.

It also wired first-class release evidence: a tag/manual workflow builds the
CGO-enabled Linux artifact on the exact toolchain, records build/module/Go
environment metadata, generates an SPDX JSON SBOM, calculates a checksum,
attests build provenance, and uploads the complete evidence directory.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Begin implementing the ordered production phases,
starting with a reproducible vulnerability-clean dependency/toolchain baseline,
and preserve detailed verification and failures.

**Inferred user intent:** Remove known reachable vulnerabilities and ensure they
cannot silently return because developer machines, CI, or cached tools use a
different Go patch level or dependency graph.

**Commit (code):** `a2c86a9` — "build: establish secure release baseline"

### What I did

- Recorded the starting graph:
  - workspace toolchain: Go 1.26.1 from the shared `go.work`;
  - module language floor: Go 1.25.11;
  - Fosite 0.49.0;
  - go-jose/v3 3.0.3;
  - go-sqlite3 1.14.32;
  - CGO enabled on linux/amd64.
- Confirmed the go-jose/v3 path is `internal/fositeadapter -> Fosite ->
  go-jose/v3`.
- Upgraded go-jose/v3 from 3.0.3 to 3.0.5 and ran `go mod tidy`.
- Added `toolchain go1.26.5` while retaining the Go 1.25.11 language floor.
- Added pinned `make vuln` (`govulncheck` v1.5.0) and composite `make verify`.
- Made golangci-lint and Glazed-lint cache names include the exact Go toolchain
  so stale binaries cannot target an older language version.
- Added `.github/workflows/ci.yml` with exact Go 1.26.5 build, test, vet, race,
  smoke, lint, and reachable-vulnerability jobs.
- Added `.github/workflows/release-evidence.yml` with checksum, build info, Go
  environment, SPDX JSON SBOM, GitHub provenance attestation, and artifact
  upload.
- Validated workflows with Ruby YAML parsing and actionlint v1.7.7.
- Ran build, full tests, vet, race, lint, rebuilt Staticcheck, the custom Go
  analyzer, govulncheck, three fuzz campaigns, strict conformance, and module
  checksum verification.

### Why

- Untrusted OAuth/JWT inputs reach go-jose through Fosite; reachable known
  parser/panic vulnerabilities are release blockers even when happy-path tests
  pass.
- `go` language and `toolchain` directives solve different problems. Retaining
  the language floor avoids an unnecessary source-language bump while exact
  release mode selects the verified security patch.
- CI must set `GOTOOLCHAIN=local` after installing Go 1.26.5 so automatic
  selection cannot hide a mismatched runner image.
- SBOM and provenance belong in the release path, not as a manual afterthought.

### What worked

- `go get` upgraded only go-jose/v3; Fosite remained at 0.49.0.
- `GOWORK=off go version` selects Go 1.26.5 from the new toolchain directive.
- Build, unit/integration tests, vet, and race all pass on Go 1.26.5.
- Rebuilt Staticcheck v0.6.1 passes.
- The corrected lint run reports zero issues and Glazed lint passes.
- govulncheck v1.5.0 reports:

  ```text
  No vulnerabilities found.

  Your code is affected by 0 vulnerabilities.
  This scan also found 2 vulnerabilities in packages you import and 14
  vulnerabilities in modules you require, but your code doesn't appear to call
  these vulnerabilities.
  ```

- Issuer fuzzing executed 365,956 inputs with 179 interesting cases.
- Production redirect fuzzing executed 347,323 inputs with 171 interesting
  cases.
- Argon2id hash fuzzing executed 353,848 inputs with 51 interesting cases.
- The strict local conformance script completed all full-suite, strict Fosite,
  durable storage, and key checks and printed its `OK` result.
- `go mod verify`, YAML parsing, actionlint, and `git diff --check` pass.

### What didn't work

- The first lint run reused `/tmp/golangci-lint-v2.12.2`, which had been built
  with Go 1.25.11. Once `toolchain go1.26.5` changed the target, golangci-lint
  correctly rejected the mismatch:

  ```text
  Error: can't load config: the Go language version (go1.25) used to build golangci-lint is lower than the targeted Go version (1.26.5)
  The command is terminated due to an error: can't load config: the Go language version (go1.25) used to build golangci-lint is lower than the targeted Go version (1.26.5)
  make: *** [Makefile:45: lint] Error 3
  ```

  Adding `$(GO_TOOLCHAIN_VERSION)` to both linter paths forced rebuilds with Go
  1.26.5. The retry reported the correct builder and passed:

  ```text
  golangci-lint has version 2.12.2 built with go1.26.5
  0 issues.
  ```

- The custom audit analyzer exited 1/status 3 because it intentionally detects
  unresolved later-phase release blockers. Findings included the external
  internal-type API, ignored CSPRNG errors, no-op audit, allow-all limiting,
  raw `RemoteAddr`, discarded audit errors, non-transactional persistence, and
  raw SQLite copy. This is expected negative-gate evidence, not a Phase 0
  regression. It must become clean as Phases 1–4 land.

- The Step 5 changelog update again appended a blank line at EOF. The exact
  whitespace diagnostic was:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:31: new blank line at EOF.
  ```

  Removing only that final blank line restored `git diff --check`.

### What I learned

- A versioned tool binary path is insufficient when the tool validates the
  target language version; the builder Go version is also part of the cache key.
- The shared workspace remains on Go 1.26.1, but module/release mode correctly
  selects 1.26.5. CI explicitly uses module mode, so the release graph is not
  coupled to the developer workspace.
- `go mod tidy` promoted zerolog, x/sync, and x/tools to direct requirements
  because tracked ticket instrumentation/analyzers import them. These are real
  module packages under `./...`, not unexplained dependency churn.
- Dependency removal is not the goal of govulncheck: two imported-package and 14
  required-module advisories remain unreachable. The release gate is zero
  reachable vulnerabilities, with verbose evidence available for review.

### What was tricky to build

- The workspace and module intentionally select different Go patch versions.
  Verification had to use `GOWORK=off` to prove the release directive while
  normal workspace commands continue to work as requested.
- tiny-idp uses go-sqlite3 and therefore CGO. Generic project templates assuming
  `CGO_ENABLED=0` would produce a nonfunctional release design, so the evidence
  workflow builds natively on Ubuntu and records CGO state instead of pretending
  to offer cross-platform static artifacts.
- The local project-setup skill's workflow conventions were useful, but its
  force-scaffold path would overwrite project-specific Makefile/README content.
  Only the relevant CI/security/release patterns were adapted.

### What warrants a second pair of eyes

- Review whether Go 1.26.5 is the desired fixed release patch or whether policy
  should automatically advance to later security patches after explicit
  qualification.
- Review GitHub permission availability for provenance attestations on the
  repository/account plan.
- Review whether unreachable vulnerability findings require a documented
  inventory/exception even though the executable call graph is clean.
- Review the CGO/SQLite target matrix before Phase 5 expands beyond Linux amd64.

### What should be done in the future

- Keep govulncheck v1.5.0 and action versions on an explicit update cadence.
- Add the custom audit analyzer as a required CI job only when its current
  finding set has been remediated or baselined with narrow expiring exceptions.
- Expand release artifacts and signing only after the supported OS/architecture
  and SQLite build strategy are decided in Phase 5.

### Code review instructions

- Start with `go.mod` and confirm the language floor, toolchain directive, and
  go-jose selection.
- Inspect `Makefile` cache keys and run `make vuln` followed by `make lint`.
- Inspect both workflows for exact `go-version: 1.26.5` and
  `GOTOOLCHAIN: local`.
- Run `GOWORK=off go mod verify`, `GOWORK=off go test ./... -count=1`, and
  `GOWORK=off go test -race ./... -count=1`.
- Run actionlint against `.github/workflows/*.yml`.
- Expect the ticket audit analyzer to remain red only on documented later-phase
  findings; investigate any new category immediately.

### Technical details

Key commands:

```text
go get github.com/go-jose/go-jose/v3@v3.0.5
go mod tidy
GOWORK=off go build ./...
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go test -race ./... -count=1
make lint
GOWORK=off go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...
make vuln
GOWORK=off go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/*.yml
GOWORK=off bash scripts/run-conformance.sh
GOWORK=off go mod verify
```

Selected graph after the change:

```text
Go release toolchain: go1.26.5
github.com/ory/fosite v0.49.0
github.com/go-jose/go-jose/v3 v3.0.5
github.com/mattn/go-sqlite3 v1.14.32
govulncheck v1.5.0
golangci-lint v2.12.2 built with go1.26.5
CGO_ENABLED=1 linux/amd64
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- `go.mod`
- `Makefile`
- `.github/workflows/ci.yml`
- `.github/workflows/release-evidence.yml`

## Step 6: Close the Phase 0 gate from committed state

This step reran the Phase 0 acceptance gate after both the implementation and
its evidence were committed. It separates “the change passed while being
developed” from “the recorded commit is reproducibly green,” then advances the
ticket to the public embedding API phase.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement phases sequentially and close each only
after a clean, committed verification checkpoint.

**Inferred user intent:** Prevent optimistic phase completion based on an
uncommitted or partially verified workspace.

**Commit (code):** `a2c86a9` — "build: establish secure release baseline"

### What I did

- Started from committed code and committed Step 5 evidence.
- Ran `make verify`, which executes build, full tests, pinned lint, Glazed lint,
  and pinned govulncheck in module mode.
- Ran actionlint v1.7.7 against both workflows.
- Required `git diff --exit-code` and `git diff --cached --exit-code` to pass.
- Checked the Phase 0 gate task and changed the ticket's current phase to Phase
  1.

### Why

- A phase gate should describe a repository commit another engineer can check
  out, not transient working state.
- Advancing the index only after the gate keeps the ticket overview honest.

### What worked

- Build and all tests passed.
- golangci-lint reported zero issues; Glazed lint passed.
- govulncheck reported zero reachable vulnerabilities.
- Both GitHub Actions workflows passed actionlint.
- Tracked and staged diffs were empty. Only the two pre-existing untracked
  source directories belonging to `TINYIDP-PROD-001` remain, and they were not
  included in any commit.

### What didn't work

- The clean-commit gate passed on its first run. The subsequent docmgr
  changelog update repeated the known helper formatting issue:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:40: new blank line at EOF.
  ```

  Removing the final blank line restored the documentation whitespace gate.

### What I learned

- Toolchain-qualified linter caches make the second verification both correct
  and fast: the exact Go 1.26.5 binaries were reused without the stale-builder
  error.

### What was tricky to build

- “Clean” had to be defined as no tracked or staged changes because unrelated
  user-owned untracked directories predate this ticket. Deleting or staging them
  would violate scope; the explicit diff checks prove the Phase 0 commit itself
  is stable.

### What warrants a second pair of eyes

- Confirm the release policy accepts zero *reachable* vulnerabilities while
  retaining visibility into unreachable advisories.

### What should be done in the future

- Use the same committed-state gate pattern at the end of every later phase.

### Code review instructions

- Check out `a2c86a9` or later and run `make verify` with network access for any
  missing pinned tools.
- Run actionlint v1.7.7 and confirm both tracked diff commands exit zero.

### Technical details

```text
make verify
  build: PASS
  tests: PASS
  golangci-lint: 0 issues
  Glazed lint: PASS
  govulncheck: 0 reachable vulnerabilities

actionlint: PASS
git diff --exit-code: PASS
git diff --cached --exit-code: PASS
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- `go.mod`
- `.github/workflows/ci.yml`

## Step 7: Inventory the Phase 1 public boundary

This step mapped every type an external embedding consumer must currently name
or implement. The result confirms that moving only the SQLite constructor would
not fix the API: store methods, policy methods, authentication results, audit
events, and mode values all transitively expose internal packages.

No source packages moved in this step. The inventory is a deliberate checkpoint
before a direct package reorganization that will touch many imports at once.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Start Phase 1 by proving the complete public API
surface before replacing it, with no compatibility adapters.

**Inferred user intent:** Avoid shipping a superficially public constructor that
still forces consumers through unimportable or implementation-specific types.

**Commit (code):** N/A; read-only boundary inventory.

### What I did

- Inspected exported documentation for `pkg/embeddedidp` and
  `internal/storage.Store`.
- Read audit, consent, rate-limiter, authenticator, domain, and store signatures.
- Enumerated every Go file importing the internal domain, storage, audit,
  authentication, or SQLite packages.
- Identified files importing both domain and storage, which require a coordinated
  package move rather than independent path replacement.

### Why

- Go's internal-package rule applies transitively to interface method types, not
  merely to constructors.
- A direct reorganization needs an accurate blast-radius list to avoid leaving
  duplicate record types or accidental adapters.

### What worked

- `go doc` confirms the exported package currently presents `Mode` as an alias
  and `Options` as a public type while hiding the internal field types.
- The import inventory found 11 files depending on both domain and storage,
  including the provider, password service, admin services, memory/SQLite stores,
  key rotation, and review probes.

### What didn't work

- The inventory commands completed successfully. The changelog helper then
  repeated its known EOF formatting issue:

  ```text
  ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/changelog.md:49: new blank line at EOF.
  ```

  Removing that final blank line restored `git diff --check`.

### What I learned

- The public boundary consists of these type families:
  - mode and cookie/token construction policy;
  - all durable records used by the composed store interfaces;
  - store sentinel errors and persistence capabilities;
  - `AuditEvent`/`AuditSink`;
  - consent policy using public user/client records;
  - rate limiter;
  - login metadata, authentication result, and password authenticator.
- Keeping Fosite internal is straightforward; none of these product contracts
  needs a Fosite request or session type.

### What was tricky to build

- Domain records and storage interfaces currently live in separate internal
  packages and are often imported together. Moving both to one public package
  requires symbol/import consolidation in those files; naive global path
  replacement would create duplicate imports.

### What warrants a second pair of eyes

- Confirm whether public durable records and store interfaces should share
  `pkg/idpstore` as designed, or whether records warrant a separate public model
  package. The current accepted guide chooses one `idpstore` package.

### What should be done in the future

- Move records and store contracts directly, update all callers in one coherent
  commit series, and delete the internal packages rather than aliasing them.
- Add compile-time interface assertions for every public implementation.

### Code review instructions

- Start at `pkg/embeddedidp/options.go:30-40`.
- Follow `storage.Store` through every embedded capability in
  `internal/storage/interfaces.go`.
- Inspect consent and authenticator signatures for their transitive domain and
  authn types.

### Technical details

```text
direct leaks:
  internal/domain.Mode
  internal/storage.Store
  internal/audit.Sink
  internal/fositeadapter.ConsentPolicy
  internal/fositeadapter.RateLimiter
  internal/fositeadapter.PasswordAuthenticator

transitive leaks:
  domain.Client, User, PasswordCredential, AccountSecurityState
  Grant, AuthorizationCode, AccessToken, RefreshToken, Consent, Session,
  SigningKey, audit.Event, authn.LoginMetadata, authn.AuthResult
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- `pkg/embeddedidp/options.go`
- `internal/storage/interfaces.go`

## Step 8: Move identity, policy, and SQLite contracts into public packages

This step removed the internal-package barrier rather than wrapping it. Durable
records and store interfaces moved to `pkg/idpstore`; audit and runtime policy
contracts moved to `pkg/idp`; and the complete SQLite implementation, migrations,
and tests moved to `pkg/sqlitestore`.

All callers were migrated directly. The old internal packages were deleted, and
compile-time interface assertions now protect the public store, audit, consent,
limiter, and password-authenticator contracts.

### Prompt Context

**User prompt (verbatim):** “continue”

**Assistant interpretation:** Resume the active Phase 1 implementation from the
latest diary checkpoint and continue committing verified slices.

**Inferred user intent:** Complete the production API work without re-planning
or losing the detailed implementation record.

**Commit (code):** `e042a15` — "refactor: publish identity store contracts";
`0bcbf24` — "refactor: publish identity policy contracts"; `24c9a92` —
"refactor: publish sqlite store"

### What I did

- Moved domain records, validation/helpers, sentinel errors, store capabilities,
  and the shared store suite into `pkg/idpstore`.
- Updated every production, test, example, and review-tool import directly and
  removed `internal/domain` and `internal/storage`.
- Added `var _ idpstore.Store = (*Store)(nil)` for memory and SQLite stores.
- Moved audit event/sink plus development implementations into `pkg/idp`.
- Added public consent, limiter, login metadata/result, password authenticator,
  and readiness contracts without Fosite types.
- Updated services and Fosite adapter implementations to consume those public
  interfaces and added compile-time assertions.
- Updated auditlint so public `idp.NoopSink` remains detectable.
- Moved the SQLite store, embedded migrations, logger, and tests to
  `pkg/sqlitestore` and updated all callers.

### Why

- Aliases or facades would preserve the unusable structure and violate the
  explicit no-compatibility requirement.
- Moving tests and migrations with the implementation makes the public package
  the actual supported store, not a wrapper over an internal implementation.

### What worked

- The record/store move compiled on its first attempt; no Go source imports
  `internal/domain` or `internal/storage` remain.
- Full build, tests, vet, lint, and race passed after the store and policy moves.
- Auditlint stopped reporting the domain/store portion of the exported leak,
  then stopped reporting the leak entirely after policy contracts moved.
- Public SQLite full tests, lint, and targeted race tests pass.

### What didn't work

- The audit-package mechanical rewrite changed receiver expressions such as
  `s.audit.Emit` into `s.idp.Emit`. Compilation reported:

  ```text
  internal/authn/password.go:212:8: s.idp undefined (type *PasswordService has no field or method idp)
  ```

  Searching for `.idp` found the same mistake on provider receiver fields.
  Restoring only those selectors to `.audit.Emit` fixed the build.
- The SQLite package rewrite matched `package sqlite` but not the external test
  declaration `package sqlite_test`. Go reported:

  ```text
  found packages sqlitestore (logcopter.go) and sqlite (store_test.go)
  ```

  Renaming it to `sqlitestore_test` resolved the package split; the retry passed.

### What I learned

- Public exposure is now structural: consumers name only `pkg/idp`,
  `pkg/idpstore`, `pkg/sqlitestore`, and standard-library types.
- AST/type analyzers are useful migration witnesses: the exported-internal-type
  diagnostic disappeared only after the complete transitive boundary moved.

### What was tricky to build

- Eleven files imported both former domain and storage packages. Mechanical
  migration had to consolidate duplicate imports while preserving a single
  `idpstore` type identity.
- Generated logger files and external-test package names needed explicit
  treatment during directory moves.

### What warrants a second pair of eyes

- Review whether all helpers now in `pkg/idpstore` deserve long-term public API
  stability or whether some should move back behind unexported implementation
  before the first release.
- Review the public contract names before external users depend on them.

### What should be done in the future

- Add Phase 2 transaction and named invariant operations to `pkg/idpstore`
  before checking its final contract task.
- Keep compile-time assertions for every implementation added later.

### Code review instructions

- Review commits in order: `e042a15`, `0bcbf24`, then `24c9a92`.
- Run `rg 'internal/(domain|storage|store/sqlite|audit)' --glob '*.go'` and
  require no production import matches.
- Run `go test -race ./pkg/idpstore ./pkg/idp ./pkg/sqlitestore`.

### Technical details

```text
pkg/idp        audit and runtime policy contracts
pkg/idpstore   durable records, validation, errors, store capabilities/suite
pkg/sqlitestore SQLite implementation, migrations, and tests

full tests: PASS
race: PASS
golangci-lint: 0 issues
Glazed lint: PASS
```

## Step 9: Replace construction with a context-aware lifecycle API

This step replaced the pre-release constructor directly with
`embeddedidp.New(ctx, Options)`, propagated context through startup store/client
work, added structured readiness and idempotent close, and made a closed handler
return 503. Repository documentation and the example now use only public paths.

The old negative external probe was converted into a positive temporary-module
compile test. It proves an outside module can open public SQLite, construct
production options, access the handler/readiness API, and close the provider
without importing any internal package.

### Prompt Context

**User prompt (verbatim):** (see Step 8)

**Assistant interpretation:** Continue Phase 1 through the actual embedding
constructor, lifecycle, documentation, and external consumer acceptance test.

**Inferred user intent:** Deliver a usable host-facing API rather than only
publicly relocated types.

**Commit (code):** `e65ff53` — "refactor: replace embedding constructor API"

### What I did

- Changed `Options.Validate` and `embeddedidp.New` to require context.
- Threaded context through Fosite construction, client listing, and bcrypt work.
- Added `Provider.Readiness(ctx)` with stable lifecycle/store/key reason codes.
- Added idempotent `Provider.Close(ctx)` and 503 behavior after close.
- Added lifecycle, canceled-context, readiness, and close regression tests.
- Removed the unused `CookieConfig.SameSite` field rather than retaining a false
  contract.
- Updated README, storage/key/password docs, and the embedded example to public
  package paths and lifecycle calls.
- Converted `external-api-smoke.sh` from expected compiler failure to positive
  external-module compilation using only public packages.

### Why

- Startup I/O must be cancelable and lifecycle ownership must be explicit for a
  production host.
- Readiness reasons are stable non-secret codes, not raw database/key errors.
- The current SameSite behavior is fixed Lax internally; a settable but ignored
  field is worse than an honest API.

### What worked

- Constructor/provider/Fosite/cmd tests and vet pass.
- Lifecycle tests prove ready state, two successful close calls, closed
  readiness, and a 503 closed handler.
- The external consumer prints:

  ```text
  OK: external production embedding imports only public tiny-idp packages and compiles
  ```

- Targeted race, full build/tests/vet, and lint pass after the final correction.

### What didn't work

- Removing `SameSite` exposed one stale runtime-probe initializer. Build and
  lint reported:

  ```text
  unknown field SameSite in struct literal of type embeddedidp.CookieConfig
  ```

  `rg 'SameSite:' --glob '*.go'` showed only the intended internal cookie
  settings plus this stale configuration use. Removing the stale field fixed
  both gates on the first retry.
- The changelog helper again added a final blank line after recording Steps
  8–9. `git diff --check` reported `changelog.md:58: new blank line at EOF.`;
  removing only that line restored the documentation gate.

### What I learned

- The external compile boundary is fixed, but the Phase 1 runtime gate still
  needs a complete outside-module Authorization Code + PKCE flow.
- Readiness currently covers lifecycle, basic store access, and active-key
  presence; richer production dependency health remains a Phase 4 task.

### What was tricky to build

- The host owns the injected store, so `Provider.Close` must not close it.
  Current close has no background resources; it records lifecycle state and is
  deliberately idempotent.
- Constructor signature replacement touched tests, probes, internal dev wiring,
  examples, and docs simultaneously because no compatibility overload was added.

### What warrants a second pair of eyes

- Review whether handler-after-close should return 503 or whether the host alone
  should guarantee it is no longer reachable.
- Review readiness check names/reason codes before operators consume them.

### What should be done in the future

- Require production audit, limiter, trusted address, secret, schema, and valid
  key controls before closing Phase 1.
- Extend the external fixture from compile-only to a complete strict flow.

### Code review instructions

- Start at `pkg/embeddedidp/options.go` and `provider.go`.
- Run the positive external API smoke script.
- Run `go test -race ./pkg/embeddedidp ./internal/fositeadapter ./pkg/sqlitestore`.

### Technical details

```text
New(ctx, Options) (*Provider, error)
Provider.Handler() http.Handler
Provider.Readiness(ctx) idp.ReadinessReport
Provider.Close(ctx) error

full build/tests/vet: PASS
targeted race: PASS
lint: PASS, 0 issues
external module compile: PASS
```

## Related

- [Implementation guide](../design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Phase ledger](../tasks.md)
- `pkg/embeddedidp/options.go`
- `pkg/embeddedidp/provider.go`
- `pkg/idp/contracts.go`
- `pkg/idpstore/interfaces.go`
- `pkg/sqlitestore/store.go`

## Step 10: Close the production embedding preflight and external flow

This step converted Phase 1's production-mode expectations into constructor
invariants and extended the outside-module test from compilation to a complete
Authorization Code + PKCE exchange. The fixture deliberately imports only the
public packages, provisions public SQLite, serves the provider over test TLS,
checks readiness, follows the CSRF-protected login form, exchanges the code,
checks both tokens, and closes cleanly.

### Prompt Context

**User prompt (verbatim):** "continue, do ll of phase 1 and phase 2"

**Assistant interpretation:** Finish every unchecked Phase 1 acceptance item,
then continue directly into all Phase 2 durability work.

**Inferred user intent:** Require executable release evidence rather than
checking tasks based only on implementation inspection.

**Commit (code):** `88e29fd` — "feat: enforce production embedding preflight"

### What I did

- Added the public client-address resolver contract and a direct-address
  implementation for non-proxy deployments.
- Required production audit, limiter, client-address resolution, durable store,
  current schema, token secret, and a valid active RS256 key.
- Parsed the active private key at startup and rejected non-RSA or sub-2048-bit
  keys, expired keys, and multiple active verification keys.
- Forwarded the resolved address into login metadata and limiter keys.
- Added an external-module TLS flow using public SQLite, public provisioning
  records, a public authenticator, a public limiter, and S256 PKCE.

### Why

- A production mode that silently supplies permissive defaults is a label, not
  a security boundary.
- A compile-only consumer test cannot prove that the exported contracts compose
  into an operational provider.

### What worked

- The external fixture completed Authorization Code + PKCE and returned access
  and ID tokens.
- Production constructor tests and the complete repository suite passed.
- The external script reported:

  ```text
  OK: external production embedding compiles and completes Authorization Code + PKCE
  ```

### What didn't work

- No failed implementation attempt occurred in this slice.

### What I learned

- The public boundary is now sufficient for a real host without importing
  Fosite or any `internal/` package.
- Trusted-proxy interpretation remains deliberately host-owned; the provider
  consumes the resolved address through a narrow interface.

### What warrants a second pair of eyes

- Confirm the production key floor and accepted algorithm remain RS256 with RSA
  2048 bits for the first release.
- Review the external fixture whenever the login form or public construction
  contract changes.

### Code review instructions

- Start with `pkg/embeddedidp/options.go`, then follow resolver use through
  `pkg/embeddedidp/provider.go` and `internal/fositeadapter/provider.go`.
- Run the external API smoke script from the production-review ticket.

### Technical details

```text
production preflight: PASS
external module compile: PASS
external TLS Authorization Code + S256 PKCE: PASS
targeted race: PASS
```

## Step 11: Introduce atomic store operations and a migration ledger

This step made transaction boundaries part of the public storage contract and
implemented them in memory and SQLite. It then moved the security-sensitive
call sites onto named invariant operations, added active-key schema protection,
and replaced implicit migration counting with ordered checksummed records.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Implement the transactional half of Phase 2 first
so online backup and restore can rely on a deterministic database policy.

**Inferred user intent:** Eliminate partial security state and lost updates
under concurrency before calling the SQLite deployment production-ready.

**Commit (code):** `df72fdd` — "feat: make identity persistence transitions atomic"

### What I did

- Added `ReadStore`, `TxStore`, `View`, `Update`, and named atomic operations to
  `pkg/idpstore` without exposing `database/sql` or driver types.
- Implemented copy-on-commit transactions in memory and `sql.Tx`-scoped store
  callbacks in SQLite.
- Made user-plus-credential creation, password/security reset, failed-login
  update, successful-login reset/session creation, and signing-key rotation
  atomic.
- Made Fosite refresh/access revocation commit as one SQLite transaction.
- Added the partial unique active-signing-key index and final-active-key
  retirement rejection.
- Added a schema migration ledger with numeric ordering, SHA-256 checksums, and
  one transaction per migration.
- Replaced the public SQLite opening surface directly with
  `Open(ctx, Config)`, secure defaults, WAL, `FULL` synchronous mode, a five
  second busy timeout, one connection, and owner-only DB/WAL/SHM modes.
- Added rollback, conflict, concurrent lockout-counter, and failed-migration
  tests.

### Why

- A process mutex cannot provide rollback and cannot protect another process.
- Named operations document invariants at the call boundary and are easier to
  fault-test than open-coded sequences.
- Migration names alone do not prove what SQL was applied; stored checksums do.

### What worked

- Twenty-four simultaneous failed-login writers produced a count of 24 with no
  lost update and the expected lock.
- Callback failure and credential conflict left no partially committed user.
- A deliberately invalid active-key state made migration 003 fail without
  recording migration 003.
- Targeted race tests and the updated external production flow passed.

### What didn't work

- The first compile after adding `AtomicStore` correctly exposed missing SQLite
  methods and a missing `errors` import:

  ```text
  *Store does not implement idpstore.Store (missing method CreateUserWithCredential)
  internal/store/memory/store.go:493:10: undefined: errors
  ```

  Implementing the SQLite transaction surface and importing `errors` resolved
  both; the next targeted test passed.
- The direct `Open(ctx, Config)` API rewrite exposed missing `context` imports in
  two command files. Adding those imports fixed the only build failure; the
  full suite then passed.

### What I learned

- A one-connection pool is an intentional part of the supported single-active-
  node envelope: it serializes security writers and prevents connection-local
  PRAGMA drift.
- Active-key uniqueness requires deactivating every row before activating the
  target, otherwise the partial unique index can reject a valid rotation.

### What was tricky to build

- Transaction-scoped store methods must dispatch every query through the
  injected runner while lifecycle, migration, and raw adapter access still use
  the owning `*sql.DB`.
- Memory transactions need a complete snapshot so callback failure cannot leak
  writes into the live maps.

### What warrants a second pair of eyes

- Review the store interface size before the first public release.
- Review whether the single-connection default should ever be configurable
  above one for this SQLite implementation; current production guidance should
  keep it at one.

### What should be done in the future

- Add online backup, read-only verification, atomic publication, and verified
  restore before closing Phase 2.
- Add nested-transaction misuse detection if future call sites accept both a
  root store and a transaction-scoped store.

### Code review instructions

- Review `pkg/idpstore/interfaces.go`, then the two implementations, then
  `pkg/sqlitestore/transaction_test.go`.
- Run `go test -race ./pkg/sqlitestore ./internal/store/memory ./internal/authn`.

### Technical details

```text
SQLite: WAL, synchronous=FULL, busy_timeout=5000ms, max connections=1
migrations: ordered integer version + SHA-256 checksum + transactional apply
rollback tests: PASS
24-writer lockout test: PASS under race detector
external production flow: PASS
```

## Step 12: Finish verified online backup, offline restore, and Phase 1–2 gates

This step completed the recovery half of Phase 2 and then audited the entire
Phase 1/2 surface as one release gate. The main database is no longer copied as
a regular file: the store now takes a SQLite online snapshot, proves its
logical identity, and publishes it only after durability operations succeed.
Restore is a separate stopped-provider operation with two verification passes
and a preserved rollback database.

The resumed review also found and corrected two edge cases that the first green
tests did not expose: a successful backup object could be finalized twice, and
SQLite BUSY can appear as “not done, no error,” which would otherwise spin.
This is why the phase gate includes source review and repository-specific AST
analysis in addition to tests.

### Prompt Context

**User prompt (verbatim):** "continue, phase 1 and phase 2"

**Assistant interpretation:** Resume after the repository-mandated debugging
pause and finish every implementation, evidence, documentation, and delivery
item in Phases 1 and 2.

**Inferred user intent:** Reach durable long-running checkpoints where both
phases are genuinely reviewable and checked, without treating a temporary
compile error or green happy-path test as completion.

**Commit (code):** `7cd13b4` — "feat: add verified sqlite backup and restore"

### What I did

- Added `Store.Backup`, `VerifyBackup`, and `Restore` in
  `pkg/sqlitestore/backup.go`.
- Used `SQLiteConn.Backup` in 128-page steps with context checks and bounded
  retry pacing.
- Captured and compared schema version, every migration checksum, every domain
  and Fosite table count, and active signing-key IDs.
- Verified artifacts with a read-only immutable connection and
  `PRAGMA integrity_check` without invoking migrations.
- Added mode-`0600` staging/final files, mode-`0700` dedicated directories,
  file/directory fsync, same-filesystem rename, and pre-rename cancellation
  checks.
- Added offline restore with sidecar refusal, cancellation-aware staging,
  second verification, timestamped rollback copy, atomic install, and directory
  fsync.
- Added the `tinyidp admin backup restore --path ...` command.
- Made root authorization-code consumption, refresh rotation/family revocation,
  signing activation, and signing retirement open their own SQLite transaction.
- Preserved refresh-reuse detection and family revocation by committing that
  expected security outcome before returning `ErrRefreshReuseDetected`.
- Rejected nested transactions in both SQLite and memory implementations.
- Required contiguous migration versions and added checksum-tamper refusal.
- Made active-key generation use the named atomic rotation boundary.
- Updated the Go AST/analysis `tinyidpatomicity` pass for public SQLite,
  high-level atomic boundaries, and explicitly transaction-scoped helpers.
- Rewrote `docs/storage.md` as the transaction/recovery/operator contract and
  updated the intern guide with the implemented Phase 1/2 architecture,
  transition inventory, pseudocode, diagrams, APIs, file references, and tests.
- Checked ticket tasks 19, 25, and 29–49 after the evidence passed.

### Why

- WAL may contain committed identities and signing keys absent from the main DB
  file; `io.Copy` of that file can create a valid-looking but incomplete backup.
- A backup is not usable evidence until its integrity, schema, historical SQL,
  and logical contents match the source snapshot.
- Publication must be old-or-new across cancellation, ENOSPC, corruption, and
  process interruption; the final pathname must never name a partial artifact.
- Restore changes the live trust database and therefore must reject an open
  destination, preserve rollback state, and verify before and after staging.

### What worked

- The WAL test proves a non-empty committed `-wal` exists before backup and the
  committed client exists after restore.
- Concurrent writers complete without errors and the produced snapshot verifies
  against its captured manifest.
- Corrupt input, live sidecars, migration checksum mismatch, and failed
  migration state are rejected.
- Canceled and injected-ENOSPC backups preserve the existing final artifact and
  leave no staging file.
- Database, WAL, SHM, backup, and restore artifacts have the expected owner-only
  modes.
- `go test ./... -count=1`, `go build ./...`, and `go vet ./...` passed.
- `go test -race ./... -count=1` passed for every package.
- `golangci-lint run` completed with `0 issues`.
- The focused `tinyidpatomicity` Go analyzer passed the SQLite/admin/Fosite
  packages.
- The outside-module production flow again reported:

  ```text
  OK: external production embedding compiles and completes Authorization Code + PKCE
  ```
- The required reMarkable dry-run listed the implementation guide, storage
  reference, diary, and phase ledger. The real upload then reported:

  ```text
  OK: uploaded TINYIDP PROD IMPL 001 Phase 1 and 2 Completion.pdf -> /ai/2026/07/09/TINYIDP-PROD-IMPL-001
  ```

### What didn't work

- The first backup compile exposed two ordinary import mistakes:

  ```text
  pkg/sqlitestore/backup.go:14:2: "sort" imported and not used
  pkg/sqlitestore/backup.go:346:9: undefined: sha256
  ```

  Replacing `sort` with `crypto/sha256` fixed the build.
- The first admin backup test found that this environment's fresh temporary
  directory was not owner-only:

  ```text
  directory /tmp/TestServiceKeysDoctorAndBackup24089809/002 must be owner-only (0700)
  ```

  Backup/restore now correct a dedicated directory to `0700` while explicitly
  refusing `/` and the shared system temp directory.
- The ENOSPC fault test initially omitted `database/sql`:

  ```text
  pkg/sqlitestore/backup_fault_test.go:28:41: undefined: sql
  ```

  This was the third consecutive implementation correction, so work stopped
  exactly as required by `AGENTS.md` with “I think I'm stuck, let's TOUCH
  GRASS.” On the user's next continuation, adding the single import made the
  targeted suite pass.
- The full lint gate found four closure-driven named returns and one unchecked
  test rollback:

  ```text
  pkg/sqlitestore/backup_test.go:164:19: Error return value of `tx.Rollback` is not checked (errcheck)
  internal/store/memory/store.go:547:1: named return "state" with type "idpstore.AccountSecurityState" found (nonamedreturns)
  internal/store/memory/store.go:583:1: named return "result" with type "idpstore.RotationResult" found (nonamedreturns)
  pkg/sqlitestore/store.go:849:1: named return "state" with type "idpstore.AccountSecurityState" found (nonamedreturns)
  pkg/sqlitestore/store.go:885:1: named return "result" with type "idpstore.RotationResult" found (nonamedreturns)
  ```

  Local result variables preserve closure behavior without named returns; an
  explicit deferred closure checks/ignores the rollback result. The rerun
  reported zero issues.

### What I learned

- `go-sqlite3.SQLiteBackup.Finish` delegates to `Close`; a successful path must
  not also run an unconditional deferred `Close` on the finalized handle.
- The driver converts SQLite BUSY/LOCKED during backup into `done=false,
  err=nil`. A small context-aware delay on every incomplete step prevents a hot
  spin even though an error-string retry branch rarely runs.
- Reserving the sole source connection orders the manifest and backup against
  every in-process writer. This relies on the explicitly enforced one-connection
  and single-active-process deployment envelope.
- A security sentinel is sometimes an expected committed outcome. Refresh reuse
  needs outcome/error separation so the transaction commits revocation evidence
  before the API returns the sentinel.
- Fsyncing the rollback file alone is insufficient; the directory entry must be
  fsynced before replacing the destination.

### What was tricky to build

- The online backup requires both raw driver connections to remain reserved for
  the lifetime of `SQLiteBackup`, while all driver types stay out of the public
  API.
- Manifest comparison must use one stable source ordering point and include the
  protocol tables as well as identity tables; otherwise an apparently valid
  backup can omit in-flight OAuth state.
- Restore cannot safely incorporate a stale WAL. The implementation refuses any
  destination sidecar instead of guessing whether the provider is actually
  stopped.
- The atomicity analyzer needed to distinguish a real transaction boundary from
  method names that merely mutate. Named invariant methods count as boundaries;
  private helpers require a visible `tinyidp:transaction-scoped` directive.

### What warrants a second pair of eyes

- Review the raw SQLite backup handle lifetime and cleanup paths in
  `onlineBackup` against the exact pinned `go-sqlite3` implementation.
- Review the assumption that supported production filesystems implement durable
  directory fsync and atomic replacement as documented.
- Review whether every Fosite table belongs in the manifest and keep the list in
  sync with future migrations.
- Exercise an operator restore drill on the target production volume before the
  release-candidate gate; unit fault injection cannot prove a storage platform's
  crash semantics.

### What should be done in the future

- Phase 3 must make password acceptance, storage failure propagation, Argon2id
  capacity, and abuse metrics complete; the general audit analyzer still
  reports ignored audit and RNG errors assigned to later phases.
- Phase 5 must run a real stop/restore/reopen/readiness/OIDC drill on the signed
  release candidate and retain artifact hashes.

### Code review instructions

- Start at `pkg/sqlitestore/backup.go:50` and trace success, cancellation,
  verification failure, ENOSPC, and restore branches.
- Review `pkg/sqlitestore/store.go:787` for transaction scoping and named
  invariant methods, then `internal/fositeadapter/sqlstore.go:278`.
- Review `pkg/sqlitestore/backup_test.go`, `backup_fault_test.go`, and
  `transaction_test.go` as the executable failure specification.
- Run:

  ```bash
  go test ./... -count=1
  go test -race ./... -count=1
  go build ./...
  go vet ./...
  golangci-lint run
  bash ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-api-smoke.sh
  ```

### Technical details

```text
source DB policy: WAL, synchronous=FULL, busy_timeout=5000ms, one connection
backup batch: 128 SQLite pages with cancellation checks
backup verification: read-only immutable + integrity_check + schema/checksums/manifest
publication: 0600 temp + fsync(file/dir) + same-filesystem rename + fsync(dir)
restore: stopped provider + no sidecars + verify/copy/verify/rollback/rename/fsync
full tests/build/vet: PASS
full race: PASS
lint: PASS, 0 issues
focused atomicity analyzer: PASS
external production OIDC flow: PASS
reMarkable completion bundle: UPLOADED
```

## Step 13: Make password authentication bounded, fail-closed, and abuse-aware

This step completed Phase 3 as one security boundary rather than a collection
of independent checks. Password establishment now follows one public policy;
every Argon2 operation uses a bounded context-aware capacity gate; storage
failures cannot silently accept or reset a login; and the HTTP adapter consumes
account, client, and trusted-address limiter buckets before password work.

The unsupported must-change flag was removed directly. Password replacement
now revokes browser, domain OAuth, and Fosite protocol state in the same SQLite
transaction that installs the credential, so a password change does not leave
old sessions or refresh capability usable.

### Prompt Context

**User prompt (verbatim):** "continue with all the phases"

**Assistant interpretation:** Continue the long-lived ticket through every
remaining implementation and release-evidence phase, keeping the same diary,
task, commit, and reMarkable discipline.

**Inferred user intent:** Turn the reviewed pre-release service into a genuinely
production-assessable artifact, while distinguishing locally completed work
from external gates that need real services or human sign-off.

**Commit (code):** `7022e7d` — "feat: bound password authentication and abuse controls"

### What I did

- Researched the final July 2025 NIST SP 800-63B-4 password requirements and
  saved the official authenticator section under the ticket `sources/` folder
  using Defuddle.
- Added public `PasswordAcceptancePolicy`, `PasswordBlocklist`, NFC
  normalization, 15-character production minimum, 64+-character allowance,
  byte ceiling, common/context blocklisting, and non-secret rejection reasons.
- Applied the policy inside `HashCredential(ctx, ...)`, covering admin create
  and set-password paths without coupling acceptance to Argon2 parameters.
- Deleted `MustChangeAtLogin`, `MustChangePassword`, and their CLI flags because
  no complete password-change interaction existed.
- Added a context-aware semaphore around hash, verify, rehash, and dummy work.
- Exported capacity, in-flight, waiting, saturation, context rejection,
  completion, total wait, and total duration metrics through
  `Provider.PasswordWorkStats`.
- Made user/credential/account-state read failures, failed-login writes,
  success resets, malformed stored hashes, and rehash persistence failures
  return `ErrAuthenticationUnavailable`; the adapter maps those to HTTP 503.
- Added public `FixedWindowRateLimiter` with production readiness and counters.
- Added hashed-account, client, and resolved-address keys, evaluating all three
  without embedding the login in the key.
- Added `TrustedProxyResolver` with explicit CIDRs, right-to-left XFF walking,
  hop bounds, and untrusted-peer header rejection.
- Added migration 004 subject columns/indexes and atomic password-change
  revocation for domain grants/codes/tokens/sessions plus Fosite code, PKCE,
  OIDC, access, and refresh rows.
- Converted the negative security-invariants probe into a positive assertion
  tool.
- Added and ran the production-parameter password load tool; stored its JSON
  evidence under `various/phase3-password-load.json`.

### Why

- NIST SP 800-63B-4 now requires 15 characters for single-factor passwords,
  recommends allowing at least 64, forbids composition rules, requires a
  blocklist, and requires account throttling.
- Argon2id's 64 MiB default makes unconstrained parallel verification a memory
  denial-of-service primitive.
- Ignoring a lockout/reset storage failure turns storage degradation into a
  security-policy bypass.
- A password change is incomplete if old sessions and refresh state survive.
- Forwarded addresses are trustworthy only when every accepted forwarding hop
  is explicitly inside the host's trust policy.

### What worked

- Full tests, build, vet, lint, and full race passed after Phase 3.
- The outside-module production OIDC flow still passed with the new production
  readiness contracts.
- The positive invariant probe reported owner-only storage, short-password
  rejection, unsafe production-construction rejection, and ten rounds of five
  concurrent failures without a lost update.
- The 64 MiB Argon2 load completed 24 attempts with eight workers and capacity
  two in 1.047 seconds (22.9 attempts/second). It recorded 22 saturations, zero
  rejections, 25 completed operations including setup, 5.27 seconds aggregate
  wait, and 2.14 seconds aggregate Argon2 duration.
- The load's observed post-run allocation was 134,547,360 bytes with
  349,899,096 bytes of Go runtime system memory, consistent with bounded rather
  than eight-way 64 MiB work.

### What didn't work

- Defuddle extracted the complete 87,761-byte NIST page but its metadata pass
  emitted a relative-URL exception:

  ```text
  Failed to parse URL: TypeError: Invalid URL
  input: '/800-63-4/sp800-63b/authenticators/'
  ```

  `wc` and the extracted heading/body confirmed the output file was complete,
  so the valid extraction was retained and the metadata warning recorded.
- The first full compile after moving the fixed-window limiter public found one
  stale runtime-probe import:

  ```text
  scripts/runtime-probe/main.go:30:2: "github.com/manuel/tinyidp/internal/fositeadapter" imported and not used
  ```

  Removing that import made the next full test pass.
- Staticcheck rejected a test expression that intentionally called the same
  limiter twice:

  ```text
  pkg/idp/security_test.go:56:5: SA4000: identical expressions on the left and right side of the '||' operator
  ```

  Expressing the two attempts as a loop removed the ambiguity; lint then
  reported zero issues.

### What I learned

- Password acceptance and password hashing cost are separate contracts. One
  decides what may be established; the other decides how accepted bytes are
  encoded and how much work verification consumes.
- The account limiter key can remain stable without disclosing the account by
  hashing the normalized login; all keys must be consumed even if one already
  rejects to avoid bucket-dependent behavior.
- Refresh/token revocation needs an indexed subject in protocol storage. JSON
  request blobs are sufficient for restoration but are the wrong operational
  index for security transitions.
- Correct-password verification is not complete until the successful-login
  reset commits; otherwise a storage outage can leave contradictory state.

### What was tricky to build

- Unknown-account dummy verification, real verification, hashing, and rehashing
  all need the same semaphore or one path can bypass the capacity limit.
- Refreshing an old Argon2 encoding after successful verification must fail
  closed if either fresh salt generation or persistence fails, without
  misclassifying a wrong password as storage corruption.
- Password revocation spans JSON domain rows and Fosite's protocol-specific
  tables. Migration 004 backfills the subject from existing request JSON, then
  future writes persist it directly.
- A reverse proxy chain must be walked from the immediate peer leftward; taking
  the first XFF item without proving the trusted suffix lets clients spoof it.

### What warrants a second pair of eyes

- Review whether the bundled baseline blocklist is sufficient for the intended
  deployment or whether production policy must inject a maintained breach list.
- Review the production Argon2 capacity of two against the actual container
  memory limit and desired login latency.
- Review every Fosite table populated by future grant types so password-change
  revocation remains complete.
- Review whether the in-process fixed-window limiter meets the intended
  single-node availability model; it is deliberately not a distributed limit.

### What should be done in the future

- Phase 4 readiness should surface password saturation, limiter health, audit
  delivery, schema, key expiry, and maintenance state together.
- Phase 5 should repeat the load with the production host's real memory/cgroup
  limits and capture runtime/database/audit metrics over a sustained interval.

### Code review instructions

- Start at `pkg/idp/password.go`, then follow policy/work through
  `internal/authn/password.go` and `pkg/embeddedidp/options.go`.
- Review limiter/address behavior in `pkg/idp/ratelimit.go`,
  `pkg/idp/contracts.go`, and `Provider.allowLogin`.
- Review migration 004, Fosite inserts, and both store implementations' password
  artifact revocation.
- Run the full race suite, external API smoke, positive security probe, and
  password load command recorded above.

### Technical details

```text
password establishment: NFC, min 15 chars, max 1024 chars/4096 bytes, blocklist
argon2id: 64 MiB, iterations=3, parallelism=2, capacity=2 by default
limiter dimensions: sha256(account), client ID, trusted client IP
password-change revocation: domain + browser + Fosite protocol rows in one tx
full test/build/vet/lint/race: PASS
external OIDC flow: PASS
positive invariant probe: PASS
production Argon2 load: PASS
```

## Step 14: Make key, audit, readiness, and retention lifecycles observable

This step implemented the Phase 4 lifecycle boundary. The important design
choice was to avoid pretending that audit, maintenance, and readiness are
independent utilities: a production provider is ready only while its durable
store/schema, active signing key, token-secret policy, audit delivery, rate
limiter, and maintenance schedule are all safe.

### Prompt Context

**User prompt (verbatim):** "continue with all the phases"

**Assistant interpretation:** Complete every locally actionable Phase 4 and
Phase 5 task, retain exact evidence, and leave external/human gates open unless
they actually run.

**Inferred user intent:** Reach a professional release-candidate boundary with
failure semantics an operator can trust, not merely a larger unit-test count.

**Commit (code):** `f8c35bb` — "feat: harden provider lifecycle and maintenance"

### What I did

- Added `FileAuditSink`, which serializes one JSON event, appends it, calls
  `fsync`, and only then returns success. It has no memory queue and no drop
  policy: callers provide backpressure. Health reports delivered, failed, and
  dropped counters plus the stable `synchronous-fsync` policy name.
- Added `ErrAuditDelivery`. Administrative methods now return their committed
  result plus this typed error when mutation succeeds but the audit record does
  not. The error explicitly warns callers not to retry a non-idempotent
  mutation blindly.
- Replaced every ignored adapter audit emission with a checked recorder.
  Request-path failures increment a monotonic counter and make production
  readiness fail; they do not attempt to rewrite an OAuth response that may
  already have been sent.
- Made the admin CLI open `<database>.audit.jsonl` as a durable audit sink so
  production mutations no longer use the library's development fallback.
- Added explicit `CookieConfig.SameSite`; Lax is the documented default and
  Lax, Strict, or Secure+None are the production-supported policies. CSRF and
  browser-session cookies consume the effective setting.
- Removed duplicate root route registration for path-based issuers. An issuer
  at `/idp` now owns `/idp/...` only; `/authorize` and `/healthz` at the host
  root are not accidental aliases.
- Applied each client's access, ID, and refresh token TTL through Fosite's
  `DefaultClientWithCustomTokenLifespans`. Persisted request state now retains
  those values so refresh grants do not silently fall back to global TTLs after
  database restoration.
- Propagated cryptographic randomness failures from user-ID, CSRF-token, and
  browser-session handle generation. The repository Go AST analyzer verifies
  that `crypto/rand.Read` errors are not assigned to `_`.
- Added schema migration 005 with creation timestamps for Fosite protocol rows.
  Added atomic maintenance for expired/terminal domain records, old protocol
  state, expired JTIs, and signing keys whose verification overlap elapsed.
- Derived default protocol retention from the maximum configured refresh-token
  TTL plus post-expiry retention. Derived signing-key overlap from the maximum
  ID-token TTL plus five minutes of skew. Production rejects shorter values.
- Added host-owned `RunMaintenance(ctx)`, `MaintenanceStatus`, and an overdue
  schedule rule. Maintenance is synchronous and serialized, but readiness reads
  its status without blocking behind a running pass.
- Expanded readiness to stable checks for lifecycle, store, exact supported
  schema, parsed/current 2048-bit+ RS256 key and active uniqueness, token
  secret, audit health/failure counters, production limiter, and maintenance.
  Added liveness that depends only on process/provider lifecycle.
- Made `/healthz` and `/readyz` return structured JSON and distinct HTTP status
  codes. A transient dependency outage makes readiness 503 without making
  liveness fail.
- Made SQLite refuse a database whose migration ledger is newer than the
  binary's embedded schema. This is the fail-closed downgrade contract.
- Tightened the Go analyzer: it now examines only exported struct fields when
  judging public API leaks, distinguishes package functions from methods, and
  uses explicit `tinyidp:development-default` and
  `tinyidp:transaction-scoped` directives instead of permanent false-positive
  allowlists.

### Why

- Security audit evidence is useful only if “delivered” has a durable meaning.
  A hidden lossy buffer or ignored `Emit` error creates false confidence.
- A key removed from JWKS before the longest ID token expires breaks valid
  relying-party verification. Keeping every retired key forever creates
  unbounded sensitive state. A derived overlap window resolves both risks.
- Protocol tables contain heterogeneous state. Creation-time retention bounded
  by the longest possible refresh lifetime is conservative and inspectable;
  deleting solely by an authorization-code TTL would destroy valid refresh
  state.
- Liveness and readiness serve different orchestration decisions. Restarting a
  healthy process because SQLite is briefly locked usually worsens an outage.
- Configuration fields that are accepted but ineffective are worse than absent
  fields because operators believe they changed security behavior.

### What worked

- Targeted audit, admin, Fosite, embedded-provider, SQLite, and memory-store
  suites pass.
- Audit tests prove the file contains a decodable event before success is
  reported, closure transitions health to failed, and canceled contexts write
  nothing.
- Maintenance tests delete only records beyond post-expiry retention, keep
  recent retired signing keys published, and remove keys only after overlap.
- A production provider becomes unready when its audit sink closes while its
  liveness remains healthy.
- Issuer-path tests prove the root aliases are gone and both health endpoints
  return structured reports.
- Client contract tests prove per-client TTLs survive Fosite request
  serialization/restoration.
- The refined repository analyzer completes with no finding across
  `./pkg/... ./internal/...`.

### What didn't work

- The first compile after changing `newID` to return an error failed because a
  previously block-scoped `err` variable was reused outside its scope:

  ```text
  internal/admin/users.go:84:7: undefined: err
  internal/admin/users.go:85:6: undefined: err
  internal/admin/users.go:86:63: undefined: err
  ```

  A local `generatedID, err := newID(...)` fixed the scope; the next targeted
  suite passed.
- The first provider suite correctly failed its old exact-three-readiness-check
  expectation after readiness grew to eight dependency checks. The assertion
  now checks the expanded contract and dedicated transition tests inspect its
  semantics.
- The first analyzer run reported deliberate development fallbacks, private
  fields on public structs, a transaction-scoped generic helper, and
  `httpServer.ListenAndServe()` as if it were package-level
  `http.ListenAndServe`. These were analyzer defects, not suppressions:
  exported-field traversal, explicit directives, and package-qualifier type
  checks removed them. The real zero-value server finding in the development
  CLI was fixed with timeouts and graceful shutdown.

### What I learned

- Fosite's per-client lifespan interface must survive serialized request state;
  wrapping only `GetClient` fixes authorization-code issuance but not refresh
  issuance after persistence.
- A production readiness check must verify the exact supported schema version,
  not merely `version > 0`; otherwise an old binary can report ready on a newer
  database.
- Holding the maintenance status mutex for an entire cleanup would make
  readiness itself hang. Separate run serialization and short status locks are
  necessary.
- A useful project analyzer needs explicit, reviewable exception semantics.
  Reporting every intentional development fallback makes the tool easy to
  ignore.

### What was tricky to build

- Maintenance must read candidate JSON rows before deleting them, then perform
  all deletes in the same SQLite transaction. Rows are closed before mutation
  because the production store deliberately uses one connection.
- Audit failures after a committed mutation cannot be rolled back generically.
  Returning the committed value and a typed post-commit error makes the
  ambiguity explicit without claiming transactional coupling that does not
  exist.
- A recent retired key has `NotAfter` set to retirement time, but remains in
  `VerificationKeys` until the separately derived overlap passes and
  maintenance removes it.

### What warrants a second pair of eyes

- Review whether synchronous `fsync` latency is acceptable for the target
  audit volume or whether a future durable outbox is needed. Do not replace it
  with an in-memory queue without a new loss/backpressure design.
- Review whether five minutes is sufficient clock skew for all relying parties.
- Review the protocol-state retention derivation whenever new Fosite grant
  types or token types are enabled.
- Review post-commit audit errors in every external admin caller so callers
  reconcile rather than blindly retry.

### Code review instructions

- Start with `pkg/idp/audit.go`, then trace failures through `internal/admin`,
  `internal/authn`, `internal/fositeadapter`, and `Provider.Readiness`.
- Review `pkg/sqlitestore/maintenance.go` together with migration 005 and the
  retention derivation in `pkg/embeddedidp/options.go`.
- Review cookie/route/TTL contracts in `internal/fositeadapter/{csrf,session,sqlstore,provider}.go`.
- Run the targeted suites and repository analyzer recorded above, then run the
  full Phase 4 gate from a clean committed tree.

### Technical details

```text
audit: synchronous append + fsync; no buffer; no intentional drops; caller backpressure
maintenance default: every 15m; unready after >2 intervals without success
protocol retention: max refresh TTL + expired retention (default expired retention 24h)
JWKS overlap: max ID token TTL + 5m skew
health: liveness=lifecycle only; readiness=store/schema/key/secret/audit/limiter/maintenance
targeted tests: PASS
custom Go analysis gate: PASS
full clean-tree Phase 4 gate: PASS as part of candidate 2930981
```

## Step 15: Build and review the production candidate without claiming external approval

This step implemented every locally actionable Phase 5 task and assembled the
release boundary. The outcome is intentionally not “shipped”: candidate
`2930981` has strong local evidence, but hosted conformance, signed CI artifacts,
license reconciliation, target-environment proof, independent review, and
release-owner sign-off remain blocking.

### Prompt Context

**User prompts (verbatim):**

- "continue with all the phases"
- "write static analysis tools with go ast/analysis since you have high quality tooling, might as well use it."
- "also you should now be able to operate with go.work normally, no need to make your own cache"
- "you can of course also write instrumentation code (or even ebpf hooks or stuff like that) to get runtime metrics and logs out of tiny-idp and then write tooling to analyze the results too, to see what's going on"
- "keep a frequent and detailed diary, btw. Backfill all that is necessary (read the skill for how to format it)"

**Assistant interpretation:** Finish the local implementation and evidence for
all remaining phases, use Go-native static/runtime tooling, keep normal workspace
behavior, and maintain an exact chronological record. Do not self-attest hosted
or human gates.

**Inferred user intent:** Produce something a professional release team can
review and continue over time, with a hard distinction between technical work
the agent can prove and organizational/external evidence it cannot fabricate.

**Commits:**

- `2a0b287` — "feat: add production host and release gates"
- `5e23978` — "ci: make release candidate builds reproducible"
- `2930981` — "fix: collect release dependency licenses"

### What I did

- Used the Glazed command-authoring workflow to implement
  `tinyidp serve-production` with decoded typed fields, long help, existing root
  logging/help integration, and no token-secret literal/environment value.
- Added TLS 1.2 minimum, certificate/key requirements, `http.MaxBytesHandler`,
  read-header/read/write/idle timeouts, header limits, direct or trusted-proxy
  address policy, fixed-window login limiting, initial/scheduled maintenance,
  signal cancellation, `errgroup`, and graceful `http.Server.Shutdown`.
- Updated the development server to use a configured `http.Server` and graceful
  shutdown, removing the analyzer's real zero-value server finding.
- Made the admin CLI open a synchronous `<db>.audit.jsonl` sink. Schema
  migrations, backup creation, restore, user/client/password/key mutation, and
  emergency key purge now emit durable admin records.
- Added normal key overlap and a separate emergency `keys purge-retired`
  command. It refuses active and never-retired staged keys, but lets compromise
  response remove old trust immediately after atomic rotation.
- Added future-schema refusal to SQLite so an older binary cannot silently open
  a newer database. Doctor reports exact schema support.
- Added token-secret rotation evidence: a second provider with a new secret
  rejects opaque access and refresh tokens minted with the old secret.
- Added a scripted release drill for migration, doctor, online backup,
  verification, signing rotation, post-backup mutation, offline restore,
  rollback preservation, downgrade refusal, and token-secret invalidation.
- Extended runtime instrumentation with configurable concurrent full password
  login/token/refresh flows. The analyzer now includes password-work capacity,
  completions, saturation, rejections, total wait, and Argon duration.
- Ran the exact-candidate mixed load: 5,125 HTTP operations, 129 durable audit
  events, zero HTTP errors, 25 bounded password operations, capacity two, 22
  saturations, zero rejections, 8.00 seconds aggregate admission wait, and 3.46
  seconds aggregate Argon work.
- Captured CPU/heap profiles, runtime deltas, DB pool snapshots, NDJSON request
  events, and a generated Markdown analysis under the ticket.
- Ran the actual production host in tmux with an ephemeral RSA certificate and
  owner-only token secret/SQLite/audit files. Candidate `2930981` served HTTPS
  (HTTP/2), returned green structured liveness and eight-component readiness,
  wrote the maintenance audit event, and stopped cleanly on SIGINT. The port
  was checked unreachable afterward.
- Added always-on CI for build, tests, vet, CLI, AST analysis, fuzz seeds,
  external-module OIDC, backup/restore, lint, and vulnerabilities.
- Added a manual release workflow that binds the build to an expected SHA-256,
  runs race/longer fuzz/fault/recovery drills, and then runs the hosted OIDF
  plan using GitHub environment secrets.
- Added release evidence automation for binary/checksum, toolchain data, SPDX
  SBOM, module graph, dependency notices, GitHub provenance, and Sigstore
  keyless signatures.
- Verified the current official setup-go/setup-python/cosign-installer action
  lines using their primary GitHub repositories and saved Defuddle extracts in
  `sources/`.
- Created the production incident/recovery playbook and the explicit
  not-approved evidence/approval ledger.

### Why

- A library cannot configure the host's listener, TLS, request limits, signal
  handling, or proxy deployment. A production-shaped executable makes that
  ownership executable and reviewable.
- The exact binary used for conformance must be the binary signed and deployed.
  The manual workflow therefore requires the expected candidate hash and fails
  on mismatch.
- Planned signing rotation and compromise response have opposite availability
  goals. Planned overlap preserves valid tokens; emergency purge revokes trust
  immediately.
- Release evidence without explicit missing rows invites social pressure to
  reinterpret “mostly passed” as approved. The ledger makes that impossible.
- Instrumentation is more useful than an eBPF hook here: public password-work
  counters, Go runtime metrics, HTTP timing, SQLite pool stats, and durable
  audit counts observe the application invariants directly and are portable in
  CI. Kernel hooks would add privilege/platform complexity without answering
  the key policy questions more precisely.

### What worked

- Full tests passed from the working tree and again from a clean archive of the
  code candidate.
- Build, vet, custom AST analysis, and whitespace checks passed.
- Pinned golangci-lint v2.12.2 reported zero issues; Glazed lint passed.
- Govulncheck v1.5.0 found zero reachable vulnerabilities. It reported two
  vulnerabilities in imported packages and fourteen in required modules that
  current code does not call.
- Full `go test -race ./... -count=1` passed.
- Ten-second native fuzz campaigns passed: issuer 474,734 executions, redirect
  514,796, and Argon parser 442,681.
- The external-module production flow compiled and completed Authorization Code
  plus S256 PKCE.
- The release drill passed twice, including after durable admin auditing.
- A clean archive of `2930981` built twice with
  `-trimpath -buildvcs=false` to the same linux/amd64 SHA-256:
  `1df7b90b9365fb8ad0b55473db93a050a71e86c11b3156616f1f9388b102f2ae`.
- The corrected license collector found top-level notices for 354 module
  directories and explicitly listed eight unresolved module-cache entries.

### What didn't work

- The first clean-archive command set its working directory to a path that the
  same command was supposed to create. Process creation failed before the shell
  ran:

  ```text
  CreateProcess: No such file or directory
  ```

  Running from the repository, creating/extracting the archive, then `cd`-ing
  inside fixed the procedure. The next clean candidate gate passed.
- The first license collection produced zero directories. The Go template text
  used literal `\t` characters, so the tab-delimited reader never split fields.
  Replacing it with `{{printf "%s\t%s\t%s" ...}}` emitted real tabs; the second
  run collected 354 directories and eight explicit missing rows.
- The first candidate hash exposed a reproducibility mismatch: archive builds
  lack `.git`, while Actions checkouts embed VCS build metadata by default.
  Adding `-buildvcs=false` to both release workflows made two archive builds
  match while provenance retains the commit identity.
- The exact-candidate TLS smoke's first curl connection occurred before the Go
  process finished compiling and printed one retryable connection refusal. The
  configured retry then received green liveness/readiness; this was startup
  timing, not a server failure.
- The first task-ledger loop passed each numeric ID as a positional argument.
  This docmgr version requires `--id`, so every call returned `Too many
  arguments` and no task changed. One command with
  `--id 64,65,...,89` applied the reviewed set; tasks 80, 83, 85, 86, 88, and
  90 remained open by design.

### What I learned

- “Reproducible build” includes Go's VCS metadata policy, not just `-trimpath`.
- License collection must treat a missing conventional file as an explicit
  review item, not silently omit the module or declare it unlicensed.
- A one-connection SQLite design remains coherent under mixed load but exposes
  queueing directly: the exact run recorded 8,847 waits. That supports the
  single-node contract but must be compared with production SLOs and disks.
- Readiness should degrade for audit/maintenance failures while liveness stays
  green. The tmux smoke and transition tests demonstrated that distinction.
- Running every local test cannot replace hosted protocol conformance or an
  independent reviewer. Those are different evidence sources, not extra unit
  tests.

### What warrants a second pair of eyes

- Review `serve_production.go` secret lifetime, TLS/proxy defaults, shutdown
  ordering, and maintenance error policy.
- Review the emergency key purge runbook and whether all relying parties can
  react quickly enough to a compromised `kid`.
- Review the synchronous audit post-commit gap. A durable outbox could couple
  DB mutation and audit intent in a future design.
- Review the eight license follow-ups against authoritative upstream sources.
- Run the target filesystem/proxy/cgroup load and restore drill.
- Run hosted OIDF on the deployed `1df7...f2ae` binary and retain every test ID.

### What should be done next

1. Configure required branch checks for the always-on workflow.
2. Deploy the exact candidate hash to a production-like reachable environment.
3. Run `release-gates` with the matching hash and hosted plan ID. The workflow
   reads GitHub secrets; no local environment values were inspected.
4. Run `release-evidence` to create actual signatures, SBOM, provenance, module
   graph, and license bundle.
5. Reconcile the eight license rows.
6. Obtain independent security/code review and disposition every finding.
7. Record deployment owners, resource budgets, audit shipping, RTO/RPO, and
   residual-risk expiry dates.
8. Obtain release-owner signature. Only then check Phase 5 and task 90.

### Code review instructions

- Begin with the public lifecycle in `pkg/embeddedidp`, then the production host
  and public/store contracts.
- Follow signing transitions through admin, both stores, maintenance, JWKS, and
  the compromise runbook.
- Follow audit from every admin/auth/request operation to FileAuditSink and
  readiness.
- Inspect CI expressions and exact build commands; confirm hosted plan config
  points to the same hash.
- Use the evidence packet as the release checklist; blank signatures and
  blocked rows are intentional stop signs.

### Technical details

```text
candidate source: 29309814f1fcdad3a5134674fc27a8938cb39c6a
linux/amd64 sha256: 1df7b90b9365fb8ad0b55473db93a050a71e86c11b3156616f1f9388b102f2ae
toolchain: go1.26.5, CGO_ENABLED=1
full test/build/vet/race/lint/analyzer/vulnerability: PASS
three 10s fuzz campaigns: PASS
external production OIDC: PASS
TLS production host: PASS locally on exact candidate
mixed load and profiles: PASS locally on exact candidate
recovery/rotation drills: PASS locally
hosted OIDF: NOT RUN
signed artifact/SBOM/provenance: WORKFLOW READY, NOT PRODUCED
independent review/release approval: NOT OBTAINED
release decision: NOT APPROVED
```

## Step 16: Publish the committed review bundle to reMarkable

The final local documentation bundle was rendered only after the implementation
guide, operator runbook, runtime analysis, evidence/approval ledger, diary, and
task state passed `docmgr doctor` and were committed as `babf302`.

The required dry run listed all six Markdown inputs, ToC depth two, and remote
directory `/ai/2026/07/09/TINYIDP-PROD-IMPL-001`. The actual non-interactive
upload then returned:

```text
OK: uploaded TINYIDP PROD IMPL 001 Release Candidate Review.pdf -> /ai/2026/07/09/TINYIDP-PROD-IMPL-001
```

No account query or post-upload cloud listing was performed. The uploader's
success response is the delivery evidence. This publication does not alter the
release decision; the PDF prominently records the candidate as not approved
and leaves the external/human gates unchecked.

## Step 17: Audit authorization interaction and protocol continuation paths

This step reopened the strict authorization path after an older code-review
comment showed that `prompt=login` and expired `max_age` were enforced while
rendering GET but not while consuming POST. The review deliberately expanded
from that branch to the entire browser interaction boundary, token and UserInfo
request identity, session error classification, unsupported request objects,
and Fosite's multi-handler persistence ordering.

The principal conclusion is architectural: the browser currently carries an
incomplete mutable reconstruction of the authorization request. Fixing only
the empty-login condition would leave parameter loss, mutation, replay,
concurrent-tab, and future required-action regressions. The focused review
therefore recommends an opaque, expiring, one-time, server-side interaction
record and adds eight unchecked tasks before Phase 5 can close.

**Commit (code):** N/A; this was a read-only code review plus ticket research
and planning update.

### Prompt Context

**User prompts (verbatim):**

> Did you address these older code review comments: [forced re-auth POST and
> forced password-change findings]

> Ok, can you think of similar issues / paths that we need to test and
> robustify, and how to best test them?

> save useful resources with defuddle into the sources/ folder of the ticket,
> so that we can retrace your steps.

**Assistant interpretation:** Verify both old comments against current source,
then conduct a broader invariant review around the same class of lost or
fail-open state. Persist authoritative sources, findings, tests, and follow-up
tasks in `TINYIDP-PROD-IMPL-001` without changing production code.

**Inferred user intent:** Avoid closing the release on isolated point fixes;
build a systematic adversarial test plan that catches equivalent control-flow,
browser-continuation, and partial-persistence defects.

### What I did

- Traced current GET and POST authorization logic, hidden form generation,
  session reads, CSRF, consent, token, UserInfo, request-object rejection, and
  Fosite SQL persistence.
- Used `git log`, `git show`, `git blame`, and `git -S` searches to determine
  that the forced password-change finding was resolved by removal in commit
  `7022e7d`, while the forced-reauthentication POST finding remains open.
- Inspected strict test names and bodies. Existing tests prove that GET renders
  a form for `prompt=login` and stale `max_age`, but no test submits a crafted
  empty-login POST with a valid session.
- Inspected Fosite v0.49.0 source under the selected module cache. Its prompt
  validator maps invalid `max_age` parsing to zero, while the tiny-idp helper
  also treats invalid/negative values as session-satisfying. Its authorize
  response handlers persist code, PKCE, and OIDC state sequentially.
- Verified that Fosite's `AccessTokenFromRequest` accepts query parameters,
  while tiny-idp's UserInfo handler does not define a method/cache/challenge
  contract.
- Used official web results to verify exact OIDC Core, RFC 6749, RFC 9700, and
  RFC 6750 requirements before capturing them locally.
- Saved four clean Defuddle extracts under this ticket's `sources/` directory.
- Created `reference/04-authorization-interaction-and-protocol-robustness-review.md`
  with nine prioritized findings, state-machine/property/fuzz/concurrency/fault
  tests, implementation ordering, and release implications.
- Added eight unchecked docmgr tasks for regression characterization,
  server-side interactions, consent semantics, deterministic state testing,
  token/UserInfo hardening, Fosite fault injection, exact-artifact reruns, and
  the new interaction gate.

Key inspection commands included:

```bash
rg -n -C 12 'reauth|prompt=login|max_age|AuthenticatePassword|finishAuthorize' internal/fositeadapter/provider.go internal/fositeadapter/*.go
git log --all -S'MustChangePassword' --oneline -- internal/fositeadapter/provider.go internal/authn/password.go pkg internal
git show e8253b1 -- internal/fositeadapter/provider.go internal/fositeadapter/provider_test.go
git blame -L 400,438 internal/fositeadapter/provider.go
rg -n -i -C 10 'empty login|missing_login|forced reauth|prompt.*post|max_age.*post' internal/fositeadapter/*_test.go
rg -n 'func AccessTokenFromRequest|max_age|prompt.*none|CreateAuthorizeCodeSession' /home/manuel/go/pkg/mod/github.com/ory/fosite@v0.49.0
```

Defuddle captures:

```bash
defuddle parse https://openid.net/specs/openid-connect-core-1_0.html --md -o sources/openid-connect-core-1.0-errata2.md
defuddle parse https://www.rfc-editor.org/rfc/rfc9700.html --md -o sources/rfc9700-oauth-security-bcp.md
defuddle parse https://datatracker.ietf.org/doc/html/rfc6749 --md -o sources/rfc6749-oauth2-authorization-framework.md
defuddle parse https://datatracker.ietf.org/doc/html/rfc6750 --md -o sources/rfc6750-bearer-token-usage.md
```

### Why

- OpenID Core requires `prompt=login` and exceeded `max_age` to cause active
  reauthentication; displaying a form without enforcing the action on POST is
  insufficient.
- Authorization validation, authentication, and consent form one state
  machine. Hidden-field reconstruction spreads the state across two requests
  without a durable invariant.
- Negative protocol paths need artifact assertions. A response status alone
  does not prove that no code, PKCE, OIDC, consent, session, access, or refresh
  row was partially created.
- The release is already not approved. Newly confirmed control-flow findings
  must become explicit tasks rather than informal notes that can disappear
  during final sign-off.

### What worked

- The code and history clearly distinguished the two old findings: forced
  password change was removed end-to-end, while forced reauthentication was
  not patched or covered by a POST regression.
- Defuddle produced complete, multi-line extracts for OpenID Core and RFC 9700.
- The IETF Datatracker HTML representations produced complete RFC 6749 and RFC
  6750 extracts after the RFC Editor pages failed extraction.
- The focused report links every finding to concrete functions and gives
  executable test names, state dimensions, properties, and artifact checks.
- Docmgr accepted the focused reference and eight new durable tasks.

### What didn't work

- Defuddle could not extract the RFC Editor HTML representations for RFC 6749
  and RFC 6750:

  ```text
  Error: No content could be extracted from https://www.rfc-editor.org/rfc/rfc6749.html
  Error: No content could be extracted from https://www.rfc-editor.org/rfc/rfc6750.html
  ```

  Retrying the authoritative IETF Datatracker HTML URLs succeeded. The stored
  files contain 3,776 lines/163,391 bytes and 891 lines/38,919 bytes,
  respectively.
- The first `apply_patch` for the focused review used a ticket path relative to
  the workspace root rather than the `tiny-idp/` repository subdirectory:

  ```text
  apply_patch verification failed: Failed to read file to update
  /home/manuel/workspaces/2026-07-07/prod-tiny-idp/ttmp/.../04-authorization-interaction-and-protocol-robustness-review.md
  ```

  No file was changed. Prefixing the path with `tiny-idp/` applied the review
  successfully on the next attempt.

### What I learned

- The `hidden(ar)` helper does not merely omit `prompt` and `max_age`; it drops
  every authorization parameter not in its eight-field allowlist, including
  ID-token hints, locales, claims, ACR, response mode, audience/resource, and
  extension parameters.
- `sessionSatisfiesMaxAge` and Fosite both treat parse failure as effectively
  zero/no restriction. Security-policy parsing needs an application-level
  strict contract and overflow-safe time arithmetic.
- `NewAuthorizeResponse` calls separate handlers. Atomic entity methods in the
  SQL adapter do not prove atomicity across code, PKCE, and OIDC handler writes.
- The token endpoint's pre-authentication limiter uses an attacker-controlled
  client ID in the only bucket key, unlike the login path's account/client/
  address layers.
- UserInfo inherits Fosite's query-token acceptance unless tiny-idp explicitly
  rejects it.

### What was tricky to build

- Severity needs to separate confirmed control flow from consequences that
  require a reproduction. The report labels invalid-client/redirect plus
  `max_age` and partial Fosite storage consequences as tests to confirm rather
  than overstating them.
- A server-side interaction must still revalidate mutable security state—client
  disablement, user disablement, session revocation—without trusting browser
  fields or silently changing the original client request.
- Consent denial, session establishment, and OAuth error delivery require an
  explicit product decision. Their current ordering is observable behavior but
  was never documented as policy.

### What warrants a second pair of eyes

- Review whether pending interactions should bind to a browser session hash or
  only to their own random handle and CSRF proof; account switching and
  pre-login flows need deliberate semantics.
- Review how to make Fosite's multi-handler writes atomic without forking
  Fosite or leaking raw transactions through public store contracts.
- Review supported UserInfo body transport against the exact conformance
  profile before rejecting query transport and unsupported methods.
- Reproduce invalid client/redirect/scope plus `max_age` before assigning final
  severity to credential-form rendering.
- Review every authorization parameter and explicitly classify it as
  supported, ignored without semantic effect, rejected, or preserved.

### What should be done in the future

1. Implement the first three failing regressions before production changes.
2. Replace hidden continuation directly with the server-side interaction; add
   no backwards-compatibility fallback.
3. Run the complete state, mutation, replay, concurrency, fuzz, and failpoint
   program from the focused reference.
4. Replace candidate identity and rerun all release evidence after behavior
   changes.

### Code review instructions

- Start with `reference/04-authorization-interaction-and-protocol-robustness-review.md`.
- Trace `Provider.authorize`, `hidden`, `renderInteraction`, and
  `finishAuthorize` together rather than reviewing branches independently.
- Read `sessionSatisfiesMaxAge`, `readBrowserSession`, and `validateCSRF` as the
  browser-state boundary.
- Follow Fosite handler ordering into `sqlFositeStore` create/delete methods.
- Run targeted tests from the review, then full race/fuzz/external/hosted gates.

### Technical details

```text
confirmed high-leverage families: interaction required-action loss, max_age fail-open parsing/error recovery, mutable incomplete continuation, token limiter sharding
confirmed secondary families: consent denial semantics, implicit UserInfo transport, unverified request-object error claims, session/store error collapse
fault investigation: authorize code -> PKCE -> OIDC writes are separate handler calls
stored sources: OIDC Core, RFC 6749, RFC 9700, RFC 6750
production code changes: none
release status: NOT APPROVED; eight new tasks remain unchecked
```

## Step 18: Design a research-backed security assurance architecture

This step turned the interaction findings into a complete assurance program. I
researched formal OAuth/OIDC analysis, security automata, runtime verification,
typestate, interprocedural dataflow, model-based security testing, stateful and
coverage-guided fuzzing, deterministic concurrency, fault injection, and dynamic
invariant discovery. I then mapped each research technique to concrete tiny-idp
invariants and implementation entry points.

The resulting guide does not propose one universal scanner. It defines a shared
typed model and event vocabulary, then assigns static analyzers, generated tests,
fuzzers, failpoints, concurrent-history checkers, runtime monitors, and external
conformance tests to the properties each can actually evaluate. It also records
the Goja verification-plane boundary developed in the companion scripting ticket.

### Prompt Context

**User prompt (verbatim):** "what would be useful static analysis or test harnesses or fuzz harnesses or audit logs that would make it effective to approach such invariants and security relevant issues from a static analysis / high quality professional point of view, and are there existing tools out there that could assist this work?"

**User prompt (verbatim):** "Also, take a loook at tiny-idp/ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md which is an idea we are looking at, could the addition of language hooks and such allow loading for example \"validation plugins\" that could assert invariants and then run different scenarios? Maybe bye xtending or modifying the scope of the proposed scripting layer, or adding a separate scripting layer that is focused on security and verification?"

**User prompt (verbatim):** "also look at CS research for these topics, and the kind of static + runtime analysis assisted by instrumentation and such. We are doing some serious research here.

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**User prompt (verbatim):** "are there papers you want me to look for? you can btw use `surf kagi search --query ...` and `surf kagi assistant ...` to do additional research."

**User prompt (verbatim):** "can you give me the links for the papers you need, and i'll downlaod them for you. Feel free to download the pdfs when needed to, no need to do everything through defuddle either."

**Assistant interpretation:** Research the CS foundations of static and dynamic
security assurance, inspect the Goja proposal, design an intern-ready layered
architecture with concrete APIs and phases, preserve reproducible sources, and
publish the result through the ticket workflow.

**Inferred user intent:** Build a professional, long-lived verification program
that can discover entire defect families, explain its scientific basis, and make
future release decisions depend on reproducible evidence rather than ad hoc code
review.

### What I did

- Read the complete ticket research, textbook-authoring, diary, Defuddle, and
  reMarkable workflow instructions relevant to this delivery.
- Re-read the production authorization code and the Goja design's trust boundary,
  graph, runtime ownership, challenge state, test, and decision sections with
  line-numbered evidence.
- Inspected the earlier production-review `auditlint` multichecker and runtime
  analysis tools instead of designing a duplicate static-analysis foundation.
- Used primary-source web search and these additional commands:

  ```bash
  surf kagi search --query "stateful security protocol fuzzing instrumentation runtime verification authentication authorization OAuth OpenID Connect papers"
  surf kagi search --query "static dynamic hybrid invariant inference runtime traces security protocol implementation testing research"
  surf kagi assistant "For a Go OpenID Connect identity provider, identify ..."
  ```

- Captured Defuddle Markdown for protocol-state fuzzing, stateful greybox
  fuzzing, model-based security testing, security automata, typestate, IFDS,
  runtime verification, monitoring-oriented programming, metamorphic security
  testing, Daikon, CHESS, and related tool documentation.
- Downloaded and verified 11 primary PDFs, totaling approximately 7.6 MiB, into
  the implementation ticket's `sources/` directory.
- Created
  `design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md`.
- Created a companion Goja assessment and four implementation tasks in
  `TINYIDP-GOJA-001`.

### Why

- The forced-reauthentication defect is a trace property spanning two HTTP
  requests, browser state, time, and artifact issuance. A handler-only test or
  AST rule cannot establish it alone.
- Static analysis is still valuable for recurring source shapes such as
  fail-open parsing, browser-owned continuations, implicit bearer extraction,
  unstable limiter identities, and bypassed atomic store APIs.
- Structured events create a common observation contract for deterministic
  tests, fuzzing, fault injection, runtime monitoring, and incident analysis.
- Verification scripts need capabilities such as fake time and failpoints that
  must never be available to production policy callbacks.

### What worked

- Kagi Search found current protocol-fuzzing surveys and reinforced the value of
  state-transition feedback rather than raw message coverage alone.
- All 11 downloaded files were identified as PDFs. `pdftotext` made the primary
  research searchable for safety-prefix, session-integrity, instrumentation,
  concurrency, lineage, and state-feedback claims.
- The existing `auditlint` already provides high-quality `analysis.Analyzer`,
  `inspect`, typed-object, `analysistest`, and `multichecker` patterns.
- The new guide is 1,446 lines and 8,553 words, with an invariant catalog,
  architecture diagram, analyzer algorithms, Go/JS API sketches, pseudocode,
  instrumentation schema, tool matrix, eight phases, release gate, decisions,
  risks, and file references.
- `docmgr doctor` reported `✅ All checks passed` for both
  `TINYIDP-PROD-IMPL-001` and `TINYIDP-GOJA-001`.
- The required reMarkable dry-run enumerated all five bundle inputs. After the
  Markdown repair below, the real command reported:

  ```text
  OK: uploaded TINYIDP Security Invariant Assurance Research.pdf -> /ai/2026/07/10/TINYIDP-PROD-IMPL-001
  ```

### What didn't work

- Defuddle could not extract the Lübeck runtime-verification page because its
  metadata contained an invalid relative URL:

  ```text
  Failed to parse URL: TypeError: Invalid URL
  input: '/research/publications/brief-account-runtime-verification'
  ```

- Defuddle could not fetch the ACM DOI for coverage-guided property testing:

  ```text
  Error: Failed to fetch: 403 Forbidden
  ```

- Defuddle correctly rejected direct PDF input:

  ```text
  Error: Not an HTML page (content-type: application/pdf)
  ```

  The full PDF was downloaded after the user explicitly allowed PDF collection.

- The CMU dynamic-taint landing page produced:

  ```text
  Error: No content could be extracted from https://kilthub.cmu.edu/articles/journal_contribution/All_You_Ever_Wanted_to_Know_About_Dynamic_Taint_Analysis_and_Forward_Symbolic_Execution_but_might_have_been_afraid_to_ask_/6468308
  ```

- `surf kagi assistant` failed twice with the same browser readiness error:

  ```text
  Error: tab 441399001 not ready: href="https://assistant.kagi.com/" title="Kagi Assistant" readyState="complete"
  Error: tab 441399002 not ready: href="https://assistant.kagi.com/" title="Kagi Assistant" readyState="complete"
  ```

  I stopped retrying after the second failure. Kagi Search, primary web sources,
  and the downloaded papers supplied the needed evidence.

- My first combined validation command contained literal Markdown fences inside
  the JavaScript orchestration string and failed before the shell ran:

  ```text
  Script error:
  SyntaxError: Invalid or unexpected token
  ```

  I rebuilt the command from ordinary string elements, after which both doctor
  checks and the fence counts completed.

- The reMarkable dry-run does not invoke Pandoc. The first real upload found a
  literal `\n\n` inside the diary's verbatim multi-paragraph prompt:

  ```text
  Error: pandoc failed: Error producing PDF.
  ! Undefined control sequence.
  l.7865 here.\n
  ```

  I rendered each input independently, isolated the failure to this diary,
  inspected Pandoc's generated LaTeX, replaced the escaped characters with the
  prompt's actual blank line, rerendered successfully, and retried the bundle
  once. The second upload succeeded.

### What I learned

- Runtime enforcement is appropriate for finite bad prefixes only when the
  monitor observes a complete event stream and controls the transition before
  the irreversible effect. This supports native issuance guards plus offline
  trace monitors, not script-controlled authorization monitors.
- Formal OAuth/OIDC work makes session integrity a first-class property. The
  concurrent-tab and mutable-continuation cases belong in that category.
- Typestate provides the right API vocabulary for pending/consumed/denied/
  expired interactions even when persistent state cannot be encoded entirely in
  Go's type system.
- Coverage-guided property testing and stateful fuzzing address the sparse-
  precondition problem: arbitrary bytes rarely reach deep valid OIDC states.
- Dynamic invariant discovery should propose candidate properties; reviewed
  native tests/static checks must establish them more strongly.
- eBPF can corroborate host file/network/scheduler behavior but cannot infer
  required login, consent, request binding, or one-time consumption.

### What was tricky to build

- The architecture had to distinguish observation, detection, and enforcement.
  Calling every assertion a "runtime invariant" would hide whether it runs before
  or after a security effect and whether missing instrumentation invalidates it.
- The static-analysis plan had to stay precise enough for CI. The guide uses
  local AST/typed rules for exact source patterns, SSA/dataflow only where values
  cross helpers, and dynamic tests for properties that source inspection cannot
  prove.
- The Goja proposal needed to offer useful language-level scenarios without
  exposing fake clocks, failpoints, or store snapshots to a reusable production
  runtime. The solution compiles symbolic steps to pure Go data, closes Goja, and
  lets a native runner execute and judge the plan.

### What warrants a second pair of eyes

- Review the invariant catalog for missing authentication, authorization,
  session-integrity, and availability properties before API names are frozen.
- Review which store transition groups need one transaction versus explicit
  compensation, especially response-delivery failure after commit.
- Review production audit completeness and sampling: a sampled trace cannot prove
  exactly-once properties without a separate completeness signal.
- Review whether verification tooling ships in production binaries or only a
  dedicated generated test command.
- Review analyzer false-positive contracts before making new rules blocking CI.

### What should be done in the future

1. Complete Phase 0 vocabulary decisions.
2. Repair the server-owned one-time authorization interaction before freezing the
   verification DSL.
3. Implement the first three static rules: strict security parsing, explicit
   bearer transport, and injected security clock.
4. Build the typed interaction model and trace schema, then feed example, Rapid,
   fuzz, failpoint, and concurrent tests through the same monitor.
5. Implement the Goja verification plan only after native semantics are stable.

### Code review instructions

- Start with the new design's Executive Summary, Sections 5-11, implementation
  phases, and decision records.
- Cross-check every observed code claim against
  `internal/fositeadapter/provider.go` and `session.go`.
- Review the existing earlier ticket's `scripts/auditlint/main.go` before adding a
  new analyzer.
- Inspect all `sources/paper-*.md` and `.pdf` files to retrace research.
- Run frontmatter/doctor checks, then render the Markdown before reMarkable upload.

### Technical details

```text
assurance layers: typed model, custom static analysis, generated/stateful tests, controlled time/concurrency/faults, structured trace monitors, isolated Goja verification authoring
authoritative verdict owner: Go
production script authority: bounded allow/deny and protected custom claims only
verification script output: immutable VerificationPlan
source packet: Defuddle Markdown plus 11 primary PDFs
doctor: both tickets passed
remarkable: /ai/2026/07/10/TINYIDP-PROD-IMPL-001/TINYIDP Security Invariant Assurance Research.pdf
production code changes: none
release status: NOT APPROVED
```

## Step 26: Run and freeze the post-assurance exact-candidate evidence

This interval ran the complete locally executable release matrix against the
post-assurance code and froze code commit `5bb4dae` with deterministic binary
SHA-256 `cf43cae64de3c1ac9610eb2bd723eb09189df751a6da422b2f8b80dbf86f43dd`.
It corrected two test-gate defects discovered by the matrix, but it does not
claim hosted OIDF, generic scanner, signed artifact, independent review, or
release-owner approval.

### What I did

- Ran `GOWORK=off go test -race ./... -count=1` successfully.
- Ran vet and every custom AST analyzer successfully.
- Built the deterministic candidate and recorded commit and SHA-256.
- Ran pinned golangci-lint and Glazed lint.
- Fixed explicit exhaustive handling for the new security event and one error
  string; repeated lint reported zero issues.
- Ran govulncheck and observed zero reachable vulnerabilities.
- Ran three 30-second parser fuzz campaigns and three 10-second invariant fuzz
  campaigns.
- Ran authorization, code-redemption, and refresh-rotation failpoint suites.
- Ran SQLite concurrency, migration, checksum, cancellation, busy, backup, and
  disk-full fault tests.
- Ran the complete migration/backup/restore/downgrade/key/token rotation drill.
- Diagnosed the external-consumer gate's hidden output and stale Go version,
  improved failure reporting, raised the generated consumer minimum to 1.26.1,
  and reran the complete external OIDC flow successfully.
- Ran `scripts/run-conformance.sh` successfully.
- Ran the production host in tmux with TLS, HTTP/2, eight readiness checks,
  owner-only secret files, trusted-proxy configuration, body bounds, generic
  method/transport probes, and graceful shutdown.
- Verified the trusted-proxy resolver's trusted, untrusted, and malformed cases.
- Wrote `reference/06-exact-candidate-assurance-evidence-5bb4dae.md` with an
  evidence hierarchy, commands, counts, failures, fixes, and missing rows.
- Left prior untracked hosted evidence directories untouched.

### What worked

- Full race, vet, AST analysis, repeated lint, vulnerability scan, all fuzz
  campaigns, failpoints, recovery, external module, local conformance, proxy
  resolver, and production-host smoke passed.
- The deterministic binary hash was reproducible after the final code/tooling
  commit.
- The production host served HTTP/2 discovery, liveness, and readiness and
  stopped cleanly under tmux.

### What didn't work

- Initial lint found missing exhaustive cases and a capitalized error. Commit
  `68945c7` fixed all three, and the repeated gate was green.
- Two fuzz targets from the same package were launched concurrently and stalled;
  they exited and were rerun sequentially. Only sequential results are evidence.
- The external probe hid its error and declared Go 1.25.11. Commit `5bb4dae`
  prints failures and declares the actual 1.26.1 minimum.
- The first oversized-body probe exceeded the shell argument limit. Streaming
  the same size through stdin exercised the host bound.
- Hosted OIDF could not be run without an exact deployed issuer, plan, and suite
  authority. No environment credentials were read.
- No installed generic web scanner was available. Manual probes were recorded
  without claiming scanner equivalence.

### What I learned

- Release gates test the test harness as well as the product. Silent captured
  output and stale consumer toolchain declarations were evidence-quality bugs.
- Exhaustive event switches are useful schema-evolution alarms even when a
  default return would have the same current runtime behavior.
- Fuzz targets in one Go package should run sequentially in the local evidence
  runner.
- A local HTTP smoke, a generic web scan, and hosted OIDC conformance are
  complementary and must remain separate ledger rows.

### What warrants a second pair of eyes

- Review the Go 1.26.1 minimum as an intentional release change caused by the
  go-go-goja dependency.
- Review whether a 400 response for an oversized parsed form is acceptable or
  whether the host should normalize it to 413.
- Review the exact generic scanner selection and false-positive adjudication
  process before making it mandatory.
- Review and authorize the hosted OIDF plan against the exact deployed hash.

### Technical details

```text
code candidate: 5bb4dae6961b23c5bb9e40678316cf15dd3d07b7
binary SHA-256: cf43cae64de3c1ac9610eb2bd723eb09189df751a6da422b2f8b80dbf86f43dd
full race: PASS
vet/custom analysis/pinned lint: PASS
reachable vulnerabilities: 0
parser fuzz executions: 892448 + 860573 + 685423
invariant fuzz executions: 44544 + 44309 + 5021
fault and recovery drills: PASS
external module/local conformance: PASS
production TLS host/manual HTTP probes: PASS
hosted OIDF: NOT RUN
generic scanner: NOT RUN
release status: NOT APPROVED
```

## Step 24: Close the analyzer and monitor loops, then isolate programmable verification

This interval completed three previously separate assurance layers. First, the
custom `go/analysis` suite now checks browser continuation authority, ignored
security-transition errors, and required protocol lifecycle helpers. Second,
generated and fuzzed event sequences plus persistence failpoint executions now
feed the same offline security monitor. Third, the first Goja verification
surface compiles scripts into immutable data-only plans that are executed and
judged entirely by native Go.

The corresponding academic record is
`design-doc/04-programmable-verification-plans-research-boundary-and-implementation.md`.
It explains why a scenario authoring language must not become a reference
monitor, maps capability discipline and runtime-verification separation to the
code, and states the unresolved in-process memory isolation limit explicitly.

### Prompt Context

**User prompt (verbatim):** "also, as you go, write documents that tie the research to the work you did, at appropriate \"intervals\" / topics. Do it retroactively too if appropriate, I basically want to be able to tie back some of the code you wrote to its underlying research, and what you inferred from it. This is valuable work that needs to be properly put into academic context."

**Assistant interpretation:** Treat every coherent assurance mechanism as both
an implementation interval and a research-accountability interval. Record which
claims came from standards or research, which facts were observed in code, and
which mechanisms were local design inferences.

**Inferred user intent:** Make the security engineering independently reviewable
and reproducible by future maintainers, rather than leaving the rationale in an
agent transcript or commit message.

**Commit (static analysis):** `9bfa7584621c2869f750e6dcddfbd8f3963f02e3` — "Add protocol invariant analyzers"

**Commit (monitor feeds):** `333d21f` — "Test security monitor across persistence failures"

**Commit (verification compiler):** `f4298f6` — "Add isolated verification plan compiler"

### What I did

- Added `tinyidpinteractioncontinuation`, which reports browser POST protocol
  parameters read by `resumeAuthorize` instead of server-owned interaction data.
- Added `tinyidpprotocollifecycle`, which checks that concrete SQL Fosite
  persistence methods use the required authorization or token lifecycle helper.
- Added `tinyidpignoredsecurityerror`, which reports discarded errors from
  interaction consumption, browser-session creation, consent recording, active
  signing-key selection, and transaction commit.
- Added positive and negative `analysistest` fixtures and ran the analyzers over
  repository production packages.
- Added generated valid trace properties to the security monitor.
- Added a native fuzz target that sends arbitrary, bounded, version-varying event
  sequences through the monitor and verifies total handling without panic.
- Fed all code-redemption and refresh-rotation failure points through a recorder
  and monitor.
- Asserted that failed code redemption emits no committed token lifecycle and a
  failed refresh rotation emits no lifecycle beyond the baseline successful
  code exchange.
- Added `pkg/verifyplan`, a Goja-independent package containing the versioned
  plan schema, limits, source binding, driver contract, native assertion
  registry, and scenario runner.
- Added the compile-only `tinyidp/verify` CommonJS module. It converts
  lower-camel plain JavaScript data into a validated normalized plan and exposes
  no provider or ambient capability.
- Added an isolated compiler with a 64 KiB default source bound, 250 ms default
  deadline, rejecting fallback module loader, JSON/output limits, and SHA-256
  source provenance.
- Added negative tests for ambient `fs` loading and unbounded loops.
- Added an integration test that compiles a JavaScript plan, executes its step
  through a native driver, and resolves its assertion through a native
  `(id,version)` registry.
- Verified the module both with the normal parent `go.work` and `GOWORK=off`.
- Wrote a module README with the JavaScript and Go ownership boundary.
- Wrote the focused research-to-implementation document and linked it from the
  broader research foundations.
- Preserved the two unrelated untracked hosted-conformance source directories.

### Why

- The earlier defects were not isolated syntax mistakes; they were authority and
  temporal-order failures. The analyzers encode local structural rules that
  make those defect classes harder to reintroduce during ordinary refactoring.
- A trace monitor tested only with hand-authored examples can contain blind spots
  in malformed or unexpected event order. Generated valid traces test
  acceptance, while fuzz sequences test total failure handling.
- Persistence fault tests become stronger when they check both database state
  and emitted transition evidence. A transaction rollback must not claim a
  committed token lifecycle.
- Loading JavaScript directly into provider execution would enlarge the trusted
  computing base. Compiling into plain data permits expressive scenario
  authoring while native code retains effects and verdict authority.
- A source digest makes evidence refer back to the exact scenario source, but it
  is intentionally documented as provenance rather than authentication.

### What worked

- All analyzer fixtures, analyzer repository scans, and `go vet ./...` passed.
- Both persistence failpoint tables produced monitor-valid traces with the exact
  expected committed-token counts.
- The monitor fuzz target passed 9,057 executions and added 37 interesting
  corpus inputs during the bounded local run.
- The Goja loop was interrupted by the configured deadline.
- The rejecting loader returned an explicit ambient-module error for `fs`.
- The compiled plan ran successfully through only native driver and assertion
  functions.
- Targeted Goja/plan tests passed under normal workspace resolution and with
  `GOWORK=off`.
- The complete repository suite and `go vet ./...` passed after dependency
  integration.

### What didn't work

- No unexpected implementation failure occurred in this interval. Dependency
  resolution updated direct and indirect Goja-related entries in `go.mod` and
  `go.sum`; the resulting module graph was validated both with and without the
  workspace.
- The current compiler deadline is not a heap quota. This is not treated as a
  fixed sandbox property: hostile in-process scripts remain outside the accepted
  threat model.

### What I learned

- The same native monitor can consume deterministic, property-generated, fuzzed,
  real-provider, and failpoint traces when instrumentation uses one versioned
  event alphabet.
- Static analysis is most credible when each rule states its syntactic boundary.
  The new lifecycle rule inspects concrete adapter methods; it is not a whole
  program proof that every future implementation is transactional.
- The useful Goja boundary is stronger than “scripts should not call dangerous
  methods.” The runtime receives no references to those methods at all.
- A programmable validator that authors, executes, and judges its own scenarios
  cannot supply independent assurance. Script-selected plans plus native
  execution and verdicts preserve reviewable role separation.
- Normal `go.work` support and standalone module support test different packaging
  claims; both matter for an embeddable library.

### What was tricky to build

- The lifecycle analyzer needed to recognize transaction-scoped helper calls
  without reporting legitimate cleanup methods that Fosite invokes outside an
  issuance transaction.
- Monitor assertions had to distinguish baseline successful code exchange from
  the subsequent failed refresh attempt.
- CommonJS module registration had to use the go-go-goja native module registry
  while rejecting every source-loaded fallback module.
- Goja interruption limits execution time but cannot promise hard memory
  isolation. The design document therefore separates implemented controls from
  requirements that would need a subprocess boundary.

### What warrants a second pair of eyes

- Review each analyzer's documented false-negative boundary before making it a
  mandatory release gate.
- Review whether failed security-event delivery should make production readiness
  false rather than only increment a counter.
- Review plan limit defaults and whether nested parameter objects require an
  independent depth limit beyond total serialized size.
- Review Goja built-ins such as time and randomness if deterministic compilation
  becomes a signed-build requirement.
- Review the strict-provider driver design before any live provider capability is
  exposed; the current compiler intentionally has none.

### What should be done in the future

1. Implement the typed strict-provider action and observation algebra with an
   injected clock.
2. Add native assertion packages for fresh authentication, consent-before-
   issuance, terminal uniqueness, continuation opacity, and refresh families.
3. Persist Rapid seeds, shrunk actions, plan hash, commit, driver version, and
   assertion versions in one evidence envelope.
4. Add metamorphic relations over query ordering, irrelevant parameters,
   duplicates, and memory-versus-SQLite behavior.
5. Run the exact-candidate race, fuzz, recovery, external consumer, proxy, and
   hosted conformance matrix before release approval.

### Code review instructions

- Read `design-doc/04-programmable-verification-plans-research-boundary-and-implementation.md`
  before reviewing the Goja packages.
- Review `pkg/verifyplan` first; it defines the trusted native execution model
  without JavaScript dependencies.
- Review the module exports and rejecting loader together to enumerate the
  reachable JavaScript object graph.
- Confirm the integration test's driver and assertion functions are native Go.
- Read each new analyzer beside its fixture and its precision statement.
- Compare failpoint database assertions with the corresponding security-event
  counts.
- Validate with:

  ```bash
  go test ./...
  go vet ./...
  GOWORK=off go test ./pkg/verifyplan ./internal/gojamodules/verify ./internal/gojaverify
  go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...
  ```

### Technical details

```text
new analyzer families: 3
monitor fuzz executions: 9,057
new interesting monitor inputs: 37
verification plan schema: tinyidp.verify/v1
default compiler source bound: 65,536 bytes
default compiler deadline: 250 ms
JavaScript-visible modules: tinyidp/verify only
live JavaScript capabilities: none
native driver integration: PASS
normal go.work tests: PASS
GOWORK=off verifier tests: PASS
full repository suite: PASS
go vet: PASS
strict-provider verification driver: OPEN
hard hostile-script memory isolation: NOT PROVIDED
release status: NOT APPROVED
```

## Step 23: Make token lifecycles atomic and add executable assurance layers

This step extended the authorization work into code redemption and refresh-token
rotation. The SQL Fosite store now implements the upstream `storage.Transactional`
contract, so Fosite—not a handler-specific wrapper—defines the complete token
transaction boundary. Eighteen named failpoints prove rollback behavior, and
every failed refresh rotation remains retryable with the original token.

The step also established the first executable assurance stack above example
tests. Rapid drives a reference interaction state machine, Porcupine checks
concurrent histories, and a versioned secret-free security event stream feeds an
offline parametric monitor. A new research-foundations document ties these
mechanisms to OAuth/OIDC standards, formal protocol analysis, model-based
testing, fault injection, runtime verification, and linearizability literature.

### Prompt Context

**User prompt (verbatim):** "go ahead. do it all, i'm going running."

**Additional user prompt (verbatim):** "also, as you go, write documents that tie the research to the work you did, at appropriate \"intervals\" / topics. Do it retroactively too if appropriate, I basically want to be able to tie back some of the code you wrote to its underlying research, and what you inferred from it. This is valuable work that needs to be properly put into academic context."

**Assistant interpretation:** Continue every locally actionable assurance phase
autonomously, and publish topic-oriented research-to-code documents at coherent
implementation intervals rather than leaving academic context implicit.

**Inferred user intent:** Produce durable engineering evidence that a future
reviewer or researcher can trace from primary sources and abstract properties to
specific implementation choices, counterexamples, tests, and residual risks.

**Commit (code):** `26fa7dbff131b7862431b223a7f0d55aedf98fb6` — "Verify token lifecycles and security traces"

### What I did

- Inspected Fosite 0.49.0 token handlers and its `storage.Transactional`
  interface directly from module source.
- Captured the official Fosite storage API with Defuddle in
  `sources/fosite-transactional-storage-api.md`.
- Created
  `design-doc/03-research-foundations-for-identity-protocol-invariants-atomicity-and-runtime-verification.md`.
- Wrote a retrospective research-to-code map for:
  - the opaque authorization interaction;
  - authorization artifact atomicity;
  - code redemption and refresh rotation;
  - state-machine testing and runtime trace monitoring.
- Added `TokenPersistenceHook` and implemented Fosite's `BeginTX`, `Commit`, and
  `Rollback` contract in `sqlFositeStore`.
- Made authorization-code invalidation, access-token creation, refresh-token
  creation, and refresh rotation transaction-aware.
- Changed refresh rotation's linearization point to a conditional update of the
  exact active presented token. Zero rows returns `fosite.ErrSerializationFailure`.
- Added eight code-redemption failpoints:
  - before transaction begin;
  - before/after code invalidation;
  - before/after access-token creation;
  - before/after refresh-token creation;
  - before commit.
- Added ten refresh-rotation failpoints:
  - before transaction begin;
  - before rotation;
  - after old-refresh deactivation;
  - after old-access deletion;
  - after rotation;
  - before/after replacement access creation;
  - before/after replacement refresh creation;
  - before commit.
- Proved after each redemption failure that the code remains active and no token
  rows exist.
- Proved after each refresh failure that the original refresh/access pair remains
  active and a subsequent retry succeeds.
- Added Rapid and Porcupine as explicit test dependencies.
- Added a Rapid interaction-store state machine with create, get, consume,
  expire, replay, duplicate-create, and returned-copy mutation commands.
- Added a Porcupine model for exactly-once consumption and a 16-client memory
  store history.
- Added an eight-client SQLite concurrent refresh history.
- Added `internal/securitytrace` with:
  - schema version 1;
  - secret-free typed events;
  - a concurrency-safe recorder;
  - a parametric interaction monitor;
  - violations for missing required actions, duplicate terminal outcomes,
    artifacts before approval, and duplicate artifact commits.
- Instrumented real provider transitions for interaction creation,
  authentication, consent, terminal outcome, authorization artifact commit, and
  token lifecycle commit.
- Exposed a separate security-event delivery-failure counter rather than silently
  treating monitor evidence as delivered.
- Fed a real browser authorization trace into the offline monitor and asserted no
  violations.
- Ran the complete repository suite successfully.

### Why

- Fosite already exposes the correct transaction extension point. Implementing
  that contract keeps tiny-idp aligned with upstream handler sequencing and
  avoids duplicating token orchestration in the HTTP provider.
- A private transaction inside `RotateRefreshToken` covered only two mutations;
  it did not include replacement token creation or code invalidation.
- Fault injection before and after each mutation establishes which intermediate
  states are externally impossible.
- State-machine generation explores legal and adversarial sequences that example
  tests do not enumerate and produces shrinkable counterexamples.
- Linearizability checking answers a different question from the race detector:
  whether completed concurrent results admit a legal sequential explanation.
- A log becomes verification evidence only after its schema and monitor verdict
  are explicit. The security trace therefore remains separate from human audit
  prose and excludes raw handles, credentials, codes, and tokens.

### What worked

- All eight redemption failpoints rolled back code invalidation and token rows.
- All ten refresh failpoints restored the old token pair and allowed retry.
- Rapid passed 100 generated state-machine cases on its first run.
- The 16-client interaction consumption history was linearizable.
- The eight-client refresh history had exactly one successful rotation and was
  linearizable with the one-time operation model.
- The real provider event trace satisfied the offline monitor.
- `go test ./... -count=1` passed, including the new securitytrace package and
  external consumer.

### What didn't work

- The first Porcupine test compared `CheckOperations` with `porcupine.Ok`, but
  version 1.3.0 returns `bool` from that convenience function:

  ```text
  invalid operation: result != porcupine.Ok (mismatched types bool and porcupine.CheckResult)
  ```

  Switching to the documented boolean result fixed the build.
- The first concurrent refresh assertion expected the winner's replacement token
  family to remain active. All ten repetitions instead ended with zero active
  refresh tokens. Investigation showed the sequence was correct Fosite reuse
  handling: requests that validate after the winner commits see the old token as
  inactive and revoke the whole family.

### What I learned

- Fosite's token handlers already call `MaybeBeginTx`, `MaybeCommitTx`, and
  `MaybeRollbackTx`; the missing atomicity was entirely in the adapter contract.
- Refresh rotation and refresh reuse are distinct abstract operations. Rotation
  is linearizable, but a subsequent replay intentionally transitions the family
  to revoked.
- Concurrent refresh by a legitimate client can therefore invalidate the winning
  response. Clients must serialize refresh and operators should treat concurrent
  refresh bursts as a likely family-revocation cause.
- A conditional SQL update on `(request_id, signature, active=1)` provides a clear
  rotation linearization point and maps contention to a typed serialization
  failure.
- Research context is most useful when it distinguishes normative requirements,
  observed library behavior, and local design inference; conflating them would
  overstate the strength of the evidence.

### What was tricky to build

- Some revocation methods are called both inside Fosite transactions and directly
  during code-reuse handling. They must use the context transaction when present
  but preserve direct revocation behavior outside issuance.
- Failpoint hooks must fail after a mutation but before commit to prove rollback,
  without adding an artificial compensating mutation.
- The refresh Porcupine history observed only HTTP success/failure, while the
  final database state included Fosite's separate reuse response. The test now
  checks both the operation history and the intentional terminal family state.
- Security events must be emitted after authoritative transitions. SQL terminal
  and artifact events occur only after commit; memory terminal events occur after
  atomic consume.

### What warrants a second pair of eyes

- Review whether `fosite.ErrSerializationFailure` is the best external mapping
  for a zero-row conditional refresh rotation.
- Review direct code-reuse revocation methods that operate outside a transaction;
  Fosite performs two best-effort revocations in that branch.
- Review the event schema before long-term compatibility is promised; it is
  versioned but still internal.
- Review whether security-event delivery failure should degrade readiness rather
  than only increment a counter.
- Review the operational policy for concurrent refresh family revocation and
  whether client documentation should require singleflight behavior.

### What should be done in the future

1. Extend the monitor to token-family events and feed failpoint traces through it.
2. Add analyzer rules preventing browser protocol continuation and issuance
   outside lifecycle helpers.
3. Preserve failing Rapid seeds and add metamorphic HTTP relations.
4. Capture Porcupine's primary paper and document the model's claim boundary.
5. Run race, longer fuzz/property schedules, recovery, and exact-candidate gates.

### Code review instructions

- Start with Fosite's saved `Transactional` source and compare it with
  `sqlFositeStore.BeginTX`, `Commit`, and `Rollback`.
- Trace code exchange and refresh mutations through `tokenExec` and
  `RotateRefreshToken`.
- Run the two failpoint tables independently.
- Read the Rapid reference state beside memory store semantics.
- Read the Porcupine model before interpreting either concurrent history.
- Check `securitytrace.Event` for secrets and then follow every
  `recordSecurity` call in the provider.

### Technical details

```text
code redemption failpoints: 8/8 PASS
refresh rotation failpoints: 10/10 PASS and retryable
interaction Rapid cases: 100 PASS
interaction Porcupine clients: 16, linearizable
refresh Porcupine clients: 8, one rotation winner
refresh final state after concurrent reuse: active refresh=0, access=0
security event schema: version 1
real authorization trace monitor: PASS
full repository suite: PASS
release status: NOT APPROVED
```

## Step 19: Freeze the authorization-hardening phase and task ledger

This step converted the broad assurance phases into 31 small implementation
tasks with stable docmgr IDs, explicit exit criteria, and five commit boundaries.
It also established the pre-change test baseline and accepted the product
semantics that the tests and implementation will enforce.

The ledger deliberately requires removal of the browser-hidden continuation with
no compatibility fallback. Fresh authentication, consent, one-time consumption,
mutable-state revalidation, UserInfo transport, limiter identity, and Fosite
persistence are tracked as separate review units so progress remains auditable
while the implementation spans multiple commits.

### Prompt Context

**User prompt (verbatim):** "Create a detailed set of phases each with detailed tasks (so we can keep precise track of where we are in the implementation work), and then implement them task by task, keeping a detailed diary as you work. I am going on a run, and I want you to do all of 1. - 5. at the very least."

**Assistant interpretation:** Expand the existing production assurance plan into
precise tasks, then implement the previously listed regression, interaction,
provider migration, mutable-state, and adjacent hardening work autonomously with
frequent diary and commit checkpoints.

**Inferred user intent:** Return from the run to a substantial, reviewable
security-hardening implementation whose decisions, failures, tests, and remaining
work can be followed without reconstructing the session.

### What I did

- Read the docmgr and diary workflows before changing the ticket.
- Inspected the existing task ledger, recent commit history, working tree, focused
  robustness review, assurance design, and Step 18 diary entry.
- Created
  `reference/05-authorization-interaction-hardening-implementation-ledger.md`.
- Added tasks `omhr` through `ah32`, organized into Phases 0 through 5.
- Defined the non-negotiable behavior contract and commit boundaries.
- Ran the baseline commands:

  ```bash
  go test ./internal/fositeadapter -count=1
  go test ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -count=1
  git status --short
  ```

### Why

- The earlier phase tasks described outcomes but were too coarse to identify
  whether persistence, handler migration, consent, replay, or adjacent endpoints
  were actually complete.
- Stable IDs let diary and changelog entries refer to exact review units.
- Accepted semantics must precede tests; otherwise tests would merely codify the
  current implementation.

### What worked

- The focused Fosite adapter baseline passed in 3.937 seconds.
- The public store, memory store, and SQLite store baselines all passed.
- The new ledger names every requested item 1 through 5 and assigns an explicit
  exit condition.
- Existing unrelated untracked hosted-conformance directories remained untouched.

### What didn't work

- N/A. Ticket creation, task expansion, and baseline tests completed on the first
  attempt.

### What I learned

- The repository already has individually atomic domain operations and a general
  `Update` transaction boundary, but no interaction record.
- The current production consent form represents approval as an optional checkbox;
  absence falls into a raw HTTP 403 rather than a typed denial action.
- The existing memory store transaction implementation clones every map, so the
  new interaction map must be added to both clone and replace paths.
- The current migration ledger is embedded and checksummed, making a new
  interaction table a normal versioned migration rather than ad hoc startup SQL.

### What was tricky to build

- "All of 1-5" spans behavior, persistence, HTTP, and Fosite internals. The ledger
  had to split work finely enough for precise tracking without creating tasks that
  could pass while the end-to-end invariant remained broken. Each phase therefore
  has both component tasks and a behavioral exit condition.

### What warrants a second pair of eyes

- Review the choice to accept GET and POST UserInfo methods but require the
  Authorization header for both.
- Review interaction consumption timing relative to Fosite artifact persistence;
  Phase 5 must ensure a consumed interaction cannot leave misleading partial
  protocol state.
- Review browser binding semantics for pre-login versus existing-session flows.

### What should be done in the future

1. Implement Phase 1 failing regressions before changing provider behavior.
2. Add interaction contracts and both store implementations.
3. Migrate the provider without retaining the hidden-field path.
4. Complete adjacent endpoint and lifecycle atomicity tasks before candidate
   approval.

### Code review instructions

- Start with `reference/05-authorization-interaction-hardening-implementation-ledger.md`.
- Compare its task IDs with `tasks.md`.
- Verify the baseline with the commands above.
- Confirm `git status --short` lists only ticket changes plus the two pre-existing
  unrelated hosted-conformance directories.

### Technical details

```text
phase count: 6 (Phase 0 through Phase 5)
detailed task count: 31
first code target: internal/fositeadapter interaction regressions
accepted UserInfo transport: Authorization header only
accepted consent denial: OAuth access_denied
compatibility fallback: prohibited
baseline result: PASS
```

## Step 20: Capture red interaction regressions and implement the persistence substrate

This step translated the authorization findings into executable HTTP regressions,
confirmed each defect against the unmodified provider, and implemented the
server-owned interaction record in the public store contract, memory store, and
SQLite store. The record stores only a keyed handle hash, a canonical request,
required-action bits, binding hashes, lifecycle timestamps, and a typed terminal
outcome.

The common store suite treats terminal consumption as the security boundary. A
record can be read without mutation, but only one caller may move it from pending
to an approved, denied, or rejected terminal outcome. Expired records and invalid
outcomes fail closed.

### Prompt Context

**User prompt (verbatim):** "Create a detailed set of phases each with detailed tasks (so we can keep precise track of where we are in the implementation work), and then implement them task by task, keeping a detailed diary as you work. I am going on a run, and I want you to do all of 1. - 5. at the very least."

**Assistant interpretation:** Implement the task ledger continuously, preserving
failing-first evidence and committing the interaction persistence layer as an
independently reviewable unit.

**Inferred user intent:** Make the eventual security claims traceable to tests
that failed before the implementation and to storage behavior shared by every
supported backend.

### What I did

- Added a reusable cookie-jar browser harness in
  `internal/fositeadapter/interaction_hardening_test.go`.
- Added regressions for forced `prompt=login`, expired `max_age`, malformed
  `max_age`, explicit consent denial, browser mutation of `state`, and concurrent
  duplicate submission.
- Ran the focused regressions before changing provider behavior.
- Added `InteractionRecord`, `InteractionRequiredAction`, and
  `InteractionOutcome` to `pkg/idpstore`.
- Added create, get, and atomic consume operations to `StoreOperations`,
  `ReadStore`, and the complete `Store` contract.
- Implemented copy-isolated interaction persistence in the memory store,
  including transaction clone/replace behavior.
- Added SQLite migration `006_authorization_interactions.sql`, conditional
  terminal consumption, indexes, and maintenance deletion.
- Added common store tests for copy isolation, expiry, invalid outcomes, and 16
  concurrent terminal attempts.
- Ran:

  ```bash
  go test ./internal/fositeadapter -run 'TestForcedPromptLogin|TestExpiredMaxAge|TestMalformedMaxAge|TestConsentDenial|TestAuthorizationState|TestAuthorizationInteraction' -count=1
  gofmt -w pkg/idpstore/testsuite.go
  go test ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -count=1
  ```

### Why

- Browser-hidden OAuth parameters cannot serve as an authorization continuation;
  they are attacker-controlled on the resume request.
- A one-time state machine requires an atomic backend primitive. A handler-level
  get followed by update would permit concurrent duplicate terminal outcomes.
- Common contract tests prevent memory and SQLite behavior from drifting at the
  precise boundary on which replay prevention depends.

### What worked

- The failing-first run reproduced the forced-login and `max_age=0` session-reuse
  bypasses: a blank resumed POST issued an authorization code.
- Invalid, negative, and overflowing `max_age` values rendered a credential form.
- Browser replacement of the original state returned `attacker-state`.
- Two concurrent submissions produced two authorization codes.
- After correcting the test parser to read hidden inputs only, consent decisions
  are no longer inferred from the visible checkbox by the harness.
- The memory and SQLite store suites passed, including exactly one successful
  consume among 16 concurrent callers.

### What didn't work

- The first consent-denial observation was invalid because the initial HTML parser
  collected every input, including the visible checked-value checkbox. Restricting
  the parser to `type="hidden"` fixed the harness before provider implementation.
- The HTTP regression suite remains intentionally red at this boundary because
  the current provider still trusts hidden continuation parameters and has no
  one-time interaction handle.

### What I learned

- SQLite's conditional `UPDATE ... WHERE consumed_at IS NULL AND expires_at > ?`
  gives the terminal transition a single database linearization point.
- The memory store already clones state for transaction callbacks, so omitting the
  interaction map from clone or replace would have silently lost transactions.
- The interaction schema can remain generic and secret-free; raw interaction and
  browser handles never need durable storage.

### What was tricky to build

- Returning the pre-transition record from `ConsumeInteraction` is useful to the
  handler, but the stored copy must receive both `ConsumedAt` and `Outcome` while
  preserving copy isolation for maps and byte slices.
- Top-level SQLite calls must use the store transaction boundary, while calls on
  an already scoped transaction must execute directly to avoid prohibited nested
  transactions.

### What warrants a second pair of eyes

- Confirm that the interaction outcome vocabulary is sufficient for operational
  reporting without exposing protocol secrets.
- Review whether rejected validation attempts should terminally consume an
  interaction or remain retryable by class of error.
- Review the ten-minute interaction retention target when the provider option is
  added.

### What should be done in the future

1. Replace the current CSRF double-submit token with a browser nonce and an HMAC
   bound to the opaque interaction handle.
2. Create records only after Fosite validates the original GET request.
3. Reconstruct resumed Fosite requests exclusively from canonical stored values.
4. Enforce required fresh authentication and explicit consent before atomic
   consume.

### Code review instructions

- Run `go test ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -count=1`.
- Inspect `ConsumeInteraction` in both backends for a single terminal transition.
- Verify migration 006 stores the serialized public record but indexes lifecycle
  columns separately for atomic consume and maintenance.
- Do not treat the red provider regressions as a release failure until the next
  handler-migration boundary; they are the preserved before-state evidence.

### Technical details

```text
raw interaction handle persisted: no
terminal outcomes: approved, denied, rejected
atomic consume predicate: pending AND unexpired
memory concurrent winners: 1/16
sqlite concurrent winners: 1/16
focused store suites: PASS
provider regressions before migration: FAIL as expected
```

## Step 21: Replace browser continuation, harden adjacent endpoints, and make SQLite issuance atomic

This step implemented the complete authorization-interaction hardening path built
on the persistence substrate from Step 20. The provider now treats the browser as
an untrusted transport: the initial validated authorization request becomes a
server-owned record, the browser receives only a random opaque handle and an
interaction-bound CSRF token, and the POST reconstructs Fosite input exclusively
from stored canonical values. Required fresh authentication, explicit consent,
expiry, browser/session binding, mutable-state revalidation, and exactly-once
terminal transitions are enforced before a code can be issued.

The step also hardened the neighboring token and UserInfo endpoints, added an
injected provider clock, and introduced an explicit SQLite authorization
lifecycle transaction. Fosite's authorization-code, PKCE, and OIDC writes now
share one SQL transaction with interaction consumption. Seven named failpoints
prove that every pre-write, post-write, and pre-commit failure rolls back all
protocol artifacts and leaves the interaction pending for a safe retry.

### Prompt Context

**User prompt (verbatim):** (see Step 19)

**Assistant interpretation:** Complete phases 1 through 5 of the detailed
authorization-hardening ledger, preserve a detailed chronological record, and
verify the implementation in proportion to production security risk.

**Inferred user intent:** Replace the two reported authorization defects with a
systematic state-machine repair that resists the adjacent mutation, replay,
concurrency, parsing, transport, and partial-persistence failure classes.

### What I did

- Added `internal/fositeadapter/interaction_hardening_test.go` with a reusable
  cookie-jar browser harness that parses only hidden opaque interaction fields.
- Preserved failing-first evidence for:
  - blank forced-reauthentication POSTs issuing codes;
  - `max_age=0` reusing an existing session;
  - invalid, negative, and overflowing `max_age` rendering credentials;
  - browser mutation of `state` changing the callback;
  - two concurrent submissions issuing two codes.
- Added `internal/fositeadapter/interaction.go` with:
  - canonical Fosite request copying;
  - a stable SHA-256 request digest;
  - current-client generation hashing;
  - hashed opaque interaction handles;
  - browser and existing-session bindings;
  - reconstruction of a fresh HTTP/Fosite request from stored values only.
- Replaced the old CSRF double-submit value with a stable HttpOnly browser nonce
  and an HMAC over both the nonce and opaque interaction handle. This permits
  independent concurrent tabs without one form invalidating another.
- Rewrote `/authorize` into explicit `beginAuthorize` and `resumeAuthorize`
  state transitions.
- Removed all browser-hidden OAuth/OIDC continuation fields. There is no legacy
  fallback accepting `client_id`, `redirect_uri`, `scope`, `state`, nonce, or
  PKCE values from the resumed POST.
- Implemented strict decimal `max_age` parsing and an overflow-free duration
  comparison, including exact-boundary and subsecond behavior.
- Added an injected `Clock` and `InteractionTTL` to the internal provider
  options. Authorization, session, audit, and OIDC claim times on the changed
  path use `p.now()`.
- Added explicit consent `approve` and `deny` actions. Denial atomically consumes
  the interaction with a denied outcome and returns OAuth `access_denied`.
  Omitted consent decisions fail without terminal consumption.
- Revalidated current client enablement, redirect URI, scopes, client generation,
  user enablement, browser session binding, and active signing-key readiness.
- Classified browser session results as absent, active, not found, expired,
  revoked, disabled-user, or storage-unavailable. Infrastructure failures now
  return 503 instead of collapsing into an anonymous session.
- Changed token rate limiting to use:
  - a normalized address bucket;
  - a claimed-client-plus-address pre-authentication bucket;
  - an authenticated-client post-authentication bucket.
- Hardened UserInfo to:
  - allow only GET and POST;
  - accept bearer tokens only from the `Authorization` header;
  - reject query and form bearer transport;
  - emit `Cache-Control: no-store`, `Pragma: no-cache`, and an RFC bearer
    `WWW-Authenticate` challenge on 401.
- Added a SQLite authorization lifecycle context in
  `internal/fositeadapter/sqlstore.go`. The provider starts the transaction,
  Fosite's code/PKCE/OIDC storage methods discover the transaction through the
  supplied context, and the provider commits only after response construction
  succeeds.
- Added failpoints:
  - `before_authorize_code` and `after_authorize_code`;
  - `before_pkce` and `after_pkce`;
  - `before_oidc` and `after_oidc`;
  - `before_commit`.
- Migrated the CLI serve test, external-consumer test, and runtime probe to the
  opaque interaction protocol. The provider intentionally has no compatibility
  adapter for their former hidden-field POST shape.
- Added regression coverage for:
  - opaque-form field minimization;
  - mutated and duplicate resumed parameters;
  - sequential replay;
  - concurrent replay;
  - independent concurrent tabs;
  - client mutation and disablement;
  - disabled session users;
  - interaction expiry under an injected clock;
  - strict `max_age` boundaries and a native fuzz target;
  - valid GET and POST UserInfo plus rejected query/form bearer transport;
  - SQLite all-or-none failpoint behavior.
- Extended the existing Go `analysis` multichecker with working analyzers for:
  - permissive `fosite.AccessTokenFromRequest` bearer extraction;
  - direct `time.Now` calls in named authorization/session transitions.
- Began a third analyzer for fail-open numeric parsing and recorded its current
  failing fixture below instead of hiding the incomplete state.

### Why

- A valid request on GET does not make browser-returned hidden fields trustworthy
  on POST. Only a server-owned canonical record preserves the validated request.
- Fresh-authentication requirements are properties of the interaction, not of
  whether the POST happened to include a login field.
- Replay prevention needs one atomic terminal transition shared by every storage
  backend; a handler-level read followed by write is insufficient.
- Fosite creates three related durable objects through separate storage calls.
  Without an outer transaction, an error after the first or second write leaves
  inconsistent protocol state.
- Query/form access tokens leak through URLs, logs, history, and referrers. The
  UserInfo contract intentionally accepts only the Authorization header.
- Security-time boundary tests require a deterministic clock; wall-clock sleeps
  are slow and cannot prove exact `max_age` semantics.
- Repository-specific analyzers make these contracts reviewable in CI instead of
  depending only on reviewer memory.

### What worked

- The original focused regression command failed before implementation and
  passed after migration:

  ```bash
  go test ./internal/fositeadapter -run 'TestForcedPromptLogin|TestExpiredMaxAge|TestMalformedMaxAge|TestConsentDenial|TestAuthorizationState|TestAuthorizationInteraction' -count=1
  ```

- The full adapter suite passed after updating old form drivers:

  ```text
  ok github.com/manuel/tinyidp/internal/fositeadapter 12.536s
  ```

- The first repository-wide run correctly identified two remaining old clients:
  `internal/cmds/serve_test.go` and the external-consumer test. After migrating
  them, `go test ./... -count=1` passed across the repository.
- Focused race validation passed:

  ```text
  ok github.com/manuel/tinyidp/internal/fositeadapter 13.367s
  ok github.com/manuel/tinyidp/pkg/idpstore 1.011s
  ok github.com/manuel/tinyidp/internal/store/memory 1.013s
  ok github.com/manuel/tinyidp/pkg/sqlitestore 2.586s
  ```

- All seven SQL failpoint cases left zero rows in
  `fosite_authorize_codes`, `fosite_pkces`, and `fosite_oidc_sessions`, and left
  zero consumed interactions.
- The max-age fuzz target completed 2,912 executions in one second with 14 seed
  cases and no failure.
- The bearer-transport and injected-clock `analysis.Analyzer` fixtures passed,
  and the multichecker reported no findings over `./pkg/... ./internal/...`.

### What didn't work

- The initial interaction HTML parser accidentally collected the visible consent
  checkbox because it matched every `<input>`. That made the first denial result
  invalid. Restricting it to `type="hidden"` corrected the test harness before
  provider changes.
- The first full adapter run failed because the shared `fetchCSRF` helper copied
  only the CSRF token and omitted the new interaction handle. Updating the helper
  to mutate the submitted form with `interaction` fixed all older adapter tests.
- The first concurrent-tab test used `state=tab-one` and `state=tab-two`. Fosite's
  minimum entropy policy rejected both with 303 responses. Longer state values
  corrected the fixture; this was a test-data failure, not a provider failure.
- The first repository-wide run produced these expected migration failures:

  ```text
  internal/cmds: expected 401, actual 400
  external-consumer: authorization response status=400 location=""
  ```

  Both callers were still posting the old browser-controlled continuation. They
  now parse the opaque handle and use explicit `action=approve`.
- The initial fail-open parsing analyzer fixture did not receive its expected
  diagnostic:

  ```text
  --- FAIL: TestStrictSecurityParseAnalyzer
      analysistest.go:713: fixture/checks/strictparsecheck/strictparse.go:8:
      no diagnostic was reported matching `numeric parse failure returns true in unsafeMaxAge`
  ```

  A second attempt replacing object identity with lexical identifier matching
  produced the same error. Work stopped at that point under the then-understood
  debugging limit; the user subsequently clarified that expected, diagnosable
  failures are exempt.

### What I learned

- The correct browser concurrency model is a stable browser nonce plus a form MAC
  bound to each independent interaction, not one CSRF cookie value per form.
- Fosite passes the same context through its authorization response handlers. A
  transaction pointer in that context lets the existing storage interface join
  code, PKCE, and OIDC writes without a fork or public raw-SQL API.
- SQLite's configured single-connection envelope makes it essential to complete
  all project-store reads before opening the Fosite lifecycle transaction. A
  project-store lookup while the transaction holds the sole connection could
  deadlock.
- `time.Duration(maxAge) * time.Second` can overflow before comparison. Comparing
  integer seconds and the subsecond remainder avoids multiplication entirely.
- A form-driver failure returning 400 is positive evidence that the removed
  hidden-field protocol has no accidental compatibility path.
- Rollback-only fault tests are necessary but not sufficient; the successful
  commit path should also assert one row in each protocol table and one consumed
  interaction.

### What was tricky to build

- Consumption must happen before response exposure but share fate with Fosite's
  durable artifacts. On SQLite, the interaction update is executed inside the
  same transaction before Fosite writes; rollback restores it to pending. On the
  development memory backend, atomic interaction consume is still guaranteed,
  but there is no injected persistent-write failure surface equivalent to SQL.
- The POST needs login and password fields from the user while refusing every
  browser-returned protocol parameter. Reconstruction therefore creates a new
  internal POST body from the canonical record and passes that request—not the
  browser request—to `NewAuthorizeRequest`.
- Current mutable state is checked twice: once explicitly before consumption and
  again in `finishAuthorize` where relevant. The checks must occur before opening
  the single-connection SQL transaction.
- Consent recording is supplied by an arbitrary policy interface and is not part
  of the SQL/Fosite transaction. The current ordering records consent before code
  issuance; a protocol rollback may therefore leave an idempotent consent grant
  while the interaction remains retryable. This needs explicit review and
  documentation rather than being mistaken for full cross-policy atomicity.
- The unfinished strict-parsing analyzer is attempting to connect a `strconv`
  parse assignment with an error-dependent branch that returns `true`. The absent
  fixture diagnostic indicates the call or assignment discovery stage is not
  populating the tracked parse-error set; the next step is to test those stages
  independently instead of modifying both simultaneously.

### What warrants a second pair of eyes

- Review whether explicit denial should consume before or after any additional
  audit delivery boundary; it currently consumes then writes OAuth
  `access_denied`.
- Review client generation hashing. It includes identity, redirects, scopes,
  PKCE requirement, disabled state, and `UpdatedAt`; slice order changes are
  deliberately considered a generation change.
- Review the browser/session binding rule when a fresh login replaces the session
  cookie. The original session is verified before authentication; the new cookie
  is issued only after successful credential verification.
- Review the SQLite lifecycle context and verify Fosite cannot perform an
  unexpected project-store read after the transaction starts.
- Review consent persistence as the remaining non-transactional policy side
  effect.
- Review whether production should expose the failpoint hook through any public
  embedding option. It is currently confined to the internal adapter options.
- Review exact OAuth error choices for expired/replayed opaque interactions;
  current responses intentionally avoid redirecting an untrusted reconstruction.

### What should be done in the future

1. Repair or remove the incomplete strict-parsing analyzer; do not leave a broken
   fixture in the committed tree.
2. Add the successful SQL lifecycle assertion complementing rollback tests.
3. Re-run full, race, shuffle, fuzz, vet, multichecker, and external-consumer
   validation after the analyzer is stable.
4. Reconcile Phase 1 through Phase 5 task IDs and coarse parent tasks.
5. Commit handler/endpoint changes, lifecycle transaction changes, and analyzer
   changes as coherent review units.
6. Run docmgr doctor and record exact commit hashes in the next diary step.

### Code review instructions

- Start with `internal/fositeadapter/interaction.go`, then read
  `beginAuthorize`, `resumeAuthorize`, and `finishAuthorize` in `provider.go` as
  one state machine.
- Review `csrf.go` together with the concurrent-tab test.
- Review `sqlFositeStore.beginAuthorizeLifecycle` and the three
  `Create*Session` methods together with
  `TestSQLiteAuthorizePersistenceFailpointsAreAtomic`.
- Inspect the old-client migrations in `internal/cmds/serve_test.go`,
  `scripts/external-consumer/flow_test.go`, and `scripts/runtime-probe/main.go` to
  confirm they carry only interaction/CSRF plus native user actions.
- Run:

  ```bash
  go test ./internal/fositeadapter -count=1
  go test ./... -count=1
  go test -race ./internal/fositeadapter ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -count=1
  go test ./internal/fositeadapter -run '^$' -fuzz=FuzzParseMaxAgeAcceptsOnlyBoundedDecimal -fuzztime=1s -parallel=1
  go test ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -count=1
  ```

### Technical details

```text
browser POST protocol authority: none
browser POST native fields: interaction, csrf_token, login, password, action
interaction storage key: HMAC-derived hash of 32-byte random handle
browser binding: HMAC-derived hash of stable CSRF browser nonce
CSRF form token: HMAC(browser nonce || opaque interaction handle)
interaction default TTL: 10 minutes
required-action bits: login, fresh_login, consent, step_up
terminal outcomes: approved, denied, rejected
SQLite lifecycle transaction: interaction + authorization code + PKCE + OIDC
SQL failpoint cases: 7/7 rollback cleanly
UserInfo methods: GET, POST
UserInfo bearer source: Authorization header only
max_age fuzz executions: 2912 in 1 second
working new analyzers: bearer transport, injected security clock
incomplete analyzer: fail-open strict numeric parsing
last green full-suite state before incomplete analyzer: PASS
```

## Step 22: Repair analyzer precision, close verification, and land Phases 1–5

This step resumed from the recorded analyzer failure, corrected the AST walk,
completed positive and negative lifecycle proofs, and performed a line-by-line
security review before committing the work. The detailed authorization-hardening
tasks for Phases 1 through 5 are complete. Broader assurance tasks requiring a
Rapid model, trace monitor, Porcupine history, token-endpoint failpoints, hosted
conformance, or release-owner approval remain open.

The final review tightened three behaviors beyond the first green version: the
CSRF browser cookie lifetime is refreshed for each new interaction, stale session
cookies are not bound when GET had no active session, and SQL authorization
writes refuse to execute without the outer transaction. Consent now displays the
exact bound client ID and requested scopes while keeping protocol fields out of
the browser POST.

### Prompt Context

**User prompt (verbatim):**

```text
write a detailed diary of your work so far, then continue. You only should touch grass when you don't know how to continue, errors you kind of expect or know how to fix are exempt.

Continue
```

**Assistant interpretation:** Backfill the missing diary immediately, then
continue through expected implementation and test failures until the phase work
is verified, documented, and committed.

**Inferred user intent:** Preserve the debugging rule for genuine rabbit holes
without allowing routine, understood failures to interrupt autonomous production
hardening work.

**Commit (code):** `aedff3c6d5a83007ef78a7c38ae8b5c6b5143193` — "Harden authorization interaction lifecycle"

**Commit (tooling):** `34580db400ed5e36b3798ccac0c54bcf661cdfc2` — "Add authorization security analyzers"

**Commit (consent disclosure):** `27c339e13d8a8181b24c1f8a7c48e56e0e926a58` — "Display bound authorization consent details"

### What I did

- Added the complete Step 21 backfill before resuming code changes.
- Diagnosed the strict-parsing analyzer in independent stages:
  - confirmed parse-assignment discovery;
  - found that the condition walker matched `err` and then overwrote the match
    when visiting `value` later in the expression;
  - made the match monotonic;
  - restricted the rule to predicates whose sole result is `bool`.
- Verified three new analyzer rules for fail-open parsing, permissive bearer
  transport, and direct wall-clock reads in named security transitions.
- Added `TestSQLiteAuthorizePersistenceCommitsAllArtifactsAndInteraction` to
  complement the rollback matrix with a successful-commit proof.
- Corrected the success test's 31-byte HMAC secret after Fosite rejected it; the
  test now uses a known valid key.
- Removed the claimed-client pre-authentication rate-limit bucket. The token path
  now uses a normalized address before authentication and verified client ID
  afterward, avoiding attacker-controlled limiter-key cardinality.
- Strengthened UserInfo against mixed transports and duplicate credentials:
  - duplicate Authorization headers return `invalid_request`;
  - query/form bearer values return `invalid_request`, including when mixed with
    a valid Authorization header;
  - missing/malformed header credentials return `invalid_token`.
- Added tests for valid POST UserInfo, mixed query/header, mixed form/header,
  duplicate Authorization headers, cache headers, challenges, and methods.
- Added `TestBrowserSessionStorageFailureDoesNotRenderLogin` with an
  interface-asserted fault store. It proves a 503 response without credentials.
- Added `TestSQLAuthorizeWritesRequireLifecycleTransaction` so a future direct
  storage call cannot silently reintroduce partial writes.
- Refreshed the stable CSRF nonce cookie on every interaction render while
  retaining the nonce, aligning lifetimes and preserving concurrent tabs.
- Bound `SessionIDHash` only when the initial request had an active session.
- Changed `authorizeExec` to require a lifecycle transaction instead of falling
  back to direct database execution.
- Added interaction TTL validation and removed unused CSRF clearing.
- Added an escaped consent disclosure of the validated client and each requested
  scope while retaining only opaque/native POST fields.
- Reconciled detailed AH Phase 1–5 tasks and demonstrated parent tasks.
- Reopened coarse task `3uhz` because it also requires token-endpoint lifecycle
  fault injection, which this implementation did not add.
- Committed implementation, analyzer, and consent-disclosure changes separately,
  leaving unrelated hosted-conformance directories untouched.

### Why

- An analyzer that reports typed parser tuples as fail-open policy decisions
  would create false positives and train maintainers to ignore it.
- Rollback tests alone cannot show that the normal transaction commits every
  required record; positive and negative proofs are both necessary.
- Pre-authentication client IDs are attacker-controlled and unsuitable as
  unbounded limiter storage keys.
- Mixed bearer transports should be rejected rather than resolved by precedence.
- Optional SQL transaction fallback would allow a future caller to bypass the
  all-or-none invariant accidentally.
- Consent is meaningful only when the user sees the client and scopes bound to
  the decision.

### What worked

- Analyzer fixtures and the repository scan passed after the monotonic walker and
  boolean-predicate restriction.
- The successful SQLite test observed one consumed interaction and one row in
  each authorization-code, PKCE, and OIDC table.
- All seven failure injections observed zero consumed interactions and zero rows
  across the three protocol tables.
- The full repository suite passed after all refinements.
- Focused race tests passed for adapter, store contract, memory, and SQLite.
- A randomized adapter run passed in 22.639 seconds.
- The final fuzz run completed 1,988 executions in one second without failure.
- `go vet ./...` and the custom analyzer scan passed without diagnostics.
- Consent tests prove client/scope text is displayed while hidden protocol field
  names remain absent.

### What didn't work

- The first analyzer repair still produced the missing-diagnostic error because
  assignment recognition was not the root cause.
- Once condition recognition was fixed, the repository scan produced:

  ```text
  internal/fositeadapter/session.go:90:3: numeric parse failure returns true in
  parseMaxAge; security-sensitive parsing must fail closed
  ```

  Here `true` meant parameter presence in `(int64, bool, error)`, not acceptance.
  Restricting the rule to single-boolean predicates fixed the precision error.
- The first positive transaction test failed with:

  ```text
  secret for signing HMAC-SHA512/256 is expected to be 32 byte long, got 31 byte
  ```

  Reusing a validated SQLite test key corrected the fixture.
- Initial shuffled commands appeared to complete without a final exit record. A
  20-second verbose timeout showed tests still progressing and exited 124. A
  90-second run completed in 22.639 seconds, proving randomized RSA-heavy test
  order rather than deadlock.

### What I learned

- AST search state must be monotonic when walking compound expressions.
- Function result shape is a useful analyzer precision boundary: a boolean
  predicate expresses a decision, while a parser tuple does not.
- Fosite enforces the HMAC key size during code generation in development mode,
  so success fixtures need production-valid secrets even if construction passes.
- Random order can group RSA generation and make this suite several times slower;
  timeout diagnostics must show the active test before classifying a hang.
- Task reconciliation is a review method: "displayed scopes" exposed a missing
  consent disclosure, and "authorize/token lifecycle" prevented overclaiming.

### What was tricky to build

- The parser analyzer needed useful semantic detection without becoming an
  interprocedural dataflow engine. Its single-boolean-predicate boundary is
  explainable and covered by fixtures.
- UserInfo must distinguish missing credentials (`invalid_token`) from multiple
  transport methods (`invalid_request`) while protecting every response cache.
- Consent must display configured strings without returning them as authoritative
  form inputs. Escaped text is rendered, but POST authority remains server-side.
- Shuffled verification required distinguishing orchestration output loss from a
  Go test hang using bounded verbose and longer non-verbose executions.

### What warrants a second pair of eyes

- Review the strict-parser analyzer's deliberate lack of interprocedural taint
  analysis and multi-result parser reasoning.
- Review bearer challenges on UserInfo `invalid_request` responses.
- Review whether client/scopes require richer human-readable labels before UX
  sign-off; the current disclosure is technically exact.
- Review consent persistence relative to protocol rollback; it remains an
  idempotent policy side effect outside the Fosite SQL transaction.
- Review development-memory behavior under hypothetical Fosite storage failure;
  production SQLite has the proved all-or-none path.

### What should be done in the future

1. Implement token-endpoint storage failpoints before checking `3uhz`.
2. Build the Rapid state machine, trace monitor, and Porcupine history tracked by
   the remaining assurance tasks.
3. Run hosted OIDC conformance against the exact later candidate hash.
4. Obtain independent security review and release-owner approval.
5. Re-run the complete evidence workflow after those changes.

### Code review instructions

- Review commits in order: `aedff3c`, `34580db`, then `27c339e`.
- Start with `beginAuthorize`/`resumeAuthorize`, then `interaction.go`, `csrf.go`,
  and `session.go`.
- For atomicity, trace `finishAuthorize` into `beginAuthorizeLifecycle` and
  `authorizeExec`, then read both lifecycle tests.
- Read each analyzer beside its one-purpose fixture.
- Confirm coarse task `3uhz` remains open.
- Validate with:

  ```bash
  go test ./... -count=1
  go test -race ./internal/fositeadapter ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore -count=1
  timeout 90s go test ./internal/fositeadapter -shuffle=on -count=1
  go test ./internal/fositeadapter -run '^$' -fuzz=FuzzParseMaxAgeAcceptsOnlyBoundedDecimal -fuzztime=1s -parallel=1
  go vet ./...
  go run ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint -- ./pkg/... ./internal/...
  ```

### Technical details

```text
implementation commit: aedff3c6d5a83007ef78a7c38ae8b5c6b5143193
analyzer commit: 34580db400ed5e36b3798ccac0c54bcf661cdfc2
consent disclosure commit: 27c339e13d8a8181b24c1f8a7c48e56e0e926a58
detailed AH phases complete: 1, 2, 3, 4, 5
SQL rollback failpoints passing: 7/7
SQL successful commit counts: interaction=1, code=1, pkce=1, oidc=1
final full suite: PASS
focused race: PASS
shuffle: PASS (22.639s)
fuzz: PASS (1988 executions, 14 seeds)
go vet: PASS
custom analyzers: PASS
coarse token lifecycle task 3uhz: OPEN
hosted OIDC exact-candidate gate: OPEN
release status: NOT APPROVED
```

## Step 25: Execute typed verification plans against the strict provider

This interval connected the data-only verification compiler to the actual
Fosite-backed HTTP provider without exposing a live provider object to
JavaScript. A native test driver owns browser state, opaque interaction forms,
the injected clock, requests, and observations. A compiled plan can select
actions and assertions, but Go decodes every action and owns every verdict.

### Prompt Context

**User prompt (verbatim):** "go ahead. do it all, i'm going running."

**Assistant interpretation:** Continue through every locally executable
assurance phase, using commits and detailed diary intervals rather than stopping
after the generic scripting substrate.

**Commit:** `ef50a86` — "Add typed strict provider verification scenarios"

### What I did

- Added a mutex-protected deterministic `securityClock` and injected its `Now`
  method into the real strict provider fixture.
- Added a `verifyplan.Driver` implementation in the adapter's external test
  package.
- Defined native parameter types for authorization begin, interaction submit,
  and time advance.
- Rejected unknown JSON action fields and trailing JSON before any provider
  effect.
- Kept the current opaque interaction form in native driver state.
- Emitted typed observations describing status, credential-form presence,
  interaction opacity, code presence, OAuth error, and returned state without
  exposing code values.
- Compiled a JavaScript plan that establishes a session, requests forced login,
  and submits the interaction without credentials.
- Ran that plan against the real HTTP provider with native assertions proving a
  credential form was displayed and no authorization code was issued.
- Added negative driver tests for an unknown action and an unknown field.
- Added a real-provider metamorphic test proving `ui_locales` changes do not
  change code issuance or the server-owned returned state.
- Added a pure typed interaction action/observation model.
- Added four explicit minimized replay histories for duplicate approval,
  deny/approve competition, expiration, and absent consumption.
- Added those minimized histories as committed fuzz seeds and fuzzed arbitrary
  bounded action sequences while checking terminal uniqueness.
- Updated the focused research document to distinguish implemented driver
  actions from future token and UserInfo actions.
- Checked Assurance Phase 2 tasks `qru9` and `tlmk` only after the full repository
  suite passed.

### Why

- A generic runner does not prove that its boundary survives contact with the
  real provider. The integration test demonstrates that JavaScript selects data
  while native Go retains HTTP and security authority.
- Strict decoding makes the action schema an allowlist. Silently ignored fields
  would make misspelled or future security parameters appear tested when they
  were not.
- Metamorphic testing checks relations between executions when a complete output
  oracle would be costly. `ui_locales` is presentation metadata and must not
  change the authorization state binding.
- Committed minimized histories turn discovered or anticipated state-machine
  counterexamples into stable reviewable regression inputs.

### What worked

- The compiled forced-login scenario passed against the real strict provider.
- Unknown actions and fields failed before provider execution.
- All three `ui_locales` variants preserved the expected authorization relation.
- The action-model fuzz target completed 8,560 executions in two seconds with
  four committed minimized seeds and no invariant violation.
- The complete repository suite passed after the driver and model were added.

### What didn't work

- The first build found `cloneValues` already declared in another external
  adapter test file:

  ```text
  cloneValues redeclared in this block
  ```

  Renaming the helper to `cloneScenarioValues` resolved the expected package-wide
  test helper collision. No production behavior changed.

### What I learned

- An external test package is the right initial location for the strict driver:
  it exercises public HTTP behavior and avoids creating a privileged production
  testing endpoint.
- Typed observations should report capability presence, such as whether a code
  was issued, rather than storing credential values in evidence.
- The compile-only boundary remains intact even when plans control realistic
  multi-request browser sequences.
- A minimized seed needs a named replay test as well as a fuzz corpus entry so a
  reviewer can understand the security history without decoding bytes.

### What warrants a second pair of eyes

- Review whether `ui_locales` is the best first metamorphic transform and add
  duplicate-parameter relations with explicit fail-closed expectations.
- Review the observation schema before persisting it as a compatibility promise.
- Review whether scenario driver code should become a reusable internal package
  after token and UserInfo actions are added.
- Review the pure model against the durable SQLite interaction semantics; the
  current strict driver uses the memory store.

### What should be done in the future

1. Add typed consent, token exchange, refresh, UserInfo, and store-fault actions.
2. Run store-independent plans against both memory and SQLite.
3. Add a versioned evidence envelope for commit, plan hash, seed, observations,
   security events, and native assertion versions.
4. Begin the exact-candidate static, race, fuzz, recovery, proxy, and hosted OIDC
   verification matrix.

### Technical details

```text
strict driver action kinds: 4
strict driver live script capabilities: 0
native forced-login plan: PASS
unknown action/field rejection: PASS
metamorphic ui_locales variants: 3/3 PASS
committed minimized histories: 4
action fuzz executions: 8,560
full repository suite: PASS
Assurance Phase 2 qru9: COMPLETE
Assurance Phase 2 tlmk: COMPLETE
release status: NOT APPROVED
```

## Step 27: Render and publish the complete assurance reading bundle

This interval published the current ticket deliverables to reMarkable as one
table-of-contents PDF. The bundle includes the implementation guide, all four
design/research documents, the full diary, runtime and release references, the
authorization review/ledger, the new exact-candidate evidence, the operations
runbook, tasks, and changelog. Research source captures remain in the ticket and
were not duplicated into the reading bundle.

### What I did

- Used the ticket-aware reMarkable bundle workflow with ToC depth two.
- Selected a new document name so an older annotated packet would not be
  overwritten.
- Attempted the complete render and captured the Pandoc/LaTeX failure.
- Located a literal `backslash-n` sequence in a verbatim diary prompt.
- Converted only that prompt to a fenced text block with real line breaks,
  retaining its wording while making it render-safe.
- Repeated the same bundle command successfully.

### What didn't work

The first render failed before upload:

```text
Undefined control sequence ... exempt.\n
```

Pandoc treated the literal prompt escape as LaTeX input. The fenced text form
removed the ambiguity. This was a documentation rendering defect; no code,
evidence result, or prompt meaning changed.

### Result

```text
document: TINYIDP Assurance Research and Exact Candidate 5bb4dae.pdf
destination: /ai/2026/07/10/TINYIDP-PROD-IMPL-001
upload: PASS
overwrite: no
release status: NOT APPROVED
```

## Step 28: Backfill an intern theory-to-code security curriculum

This interval reorganized the accumulated protocol research, implementation
experience, counterexamples, and release evidence into a first-week curriculum
for a new intern. The existing documents remain the detailed design and history;
seven new chapters provide the prerequisite conceptual sequence, direct code
maps, exercises, and explicit evidence boundaries needed to use them effectively.

### Prompt Context

**User prompt (verbatim):** "do it retroactively. What documents would you write for the intern who joins us tomorrow to get them turbocharged on necessary theoretical and security knowledge"

**Assistant interpretation:** Create the missing retrospective educational
documents now, using textbook structure and direct links from theory and research
to tiny-idp code, tests, traces, and release decisions.

**Inferred user intent:** Let a capable new contributor become productive quickly
without learning security assumptions accidentally through code review failures.

### What I did

- Audited all existing ticket documents by heading and purpose to avoid rewriting
  the implementation guide, assurance architecture, research map, or ledgers.
- Defined a textbook writing contract: foundational model first, no analogies,
  normative/observed/inferred/open claim labels, concrete code and traces,
  falsifiable exercises, and explicit evidence limitations.
- Wrote an accelerated curriculum and code-reading map with a five-module order,
  first-day symbol trace, competence criteria, and mentor review assignment.
- Wrote a protocol-security chapter covering principals, channels,
  Authorization Code + S256 PKCE, canonical interaction state, authentication
  freshness, typed authentication outcomes, consent, codes/tokens, UserInfo, and
  safe error redirects.
- Wrote a temporal-invariant chapter covering safety/liveness, required-action
  obligations, parametric monitoring, reference models, shrinking, replay, and
  metamorphic relations.
- Wrote a durable-state chapter covering capability rows, transaction ownership,
  Fosite transaction propagation, failpoint evidence, linearizability, refresh
  family revocation, SQLite topology, backup, and restore.
- Wrote an assurance-methods chapter comparing AST analysis, examples, Rapid,
  fuzzing, metamorphic testing, failpoints, race detection, Porcupine, monitors,
  conformance, scripting, and human review by supported claim and limitation.
- Wrote a production-security chapter covering host ownership, proxy trust,
  audit versus trace/log/metrics, readiness, key rotation, recovery, artifact
  provenance, hosted evidence, and release authority.
- Wrote eight executable intern labs tied to real tests, analyzers, failpoints,
  concurrent histories, the Goja boundary, and the release ledger.
- Related each focused document to four concrete implementation files using
  docmgr.

### Why

- The existing corpus is extensive but organized around implementation phases
  and investigations. A new contributor needs prerequisite ordering and explicit
  learning outcomes before navigating that detail.
- OAuth tutorials commonly explain messages but omit temporal obligations,
  transaction ownership, concurrent semantics, and evidence epistemology. Those
  omissions correspond directly to the defects found during this review.
- Exercises require the intern to produce models and bounded claims, preventing
  passive familiarity from being mistaken for review competence.

### What worked

- Existing sources and code were sufficient; no speculative external material
  was needed.
- The chapters reuse primary-source captures already stored in `sources/` and
  distinguish research conclusions from local mechanisms.
- The code-reading path now progresses from public ownership through one success,
  one rejection, one persistence failure, and one concurrent history.

### What didn't work

- The first docmgr validation found two references to a nonexistent
  `pkg/idpstore/contracts.go`. The actual contracts are split between
  `interfaces.go` and `types.go`. The links and reading text were corrected
  before commit.

### What I learned

- The missing onboarding layer was not more implementation detail. It was a
  structured bridge from protocol semantics to temporal logic, durable
  authority, evidence interpretation, and production ownership.
- “What does this test not prove?” is a useful recurring exercise because it
  teaches both security reasoning and scientific restraint.
- The original forced-login and token-atomicity defects form effective teaching
  cases because each connects an abstract property to a minimal counterexample
  and a concrete repair.

### What warrants a second pair of eyes

- Review the one-week competence criteria against the actual first assignment we
  intend to give the intern.
- Review whether WebAuthn, federation, logout, DPoP, and device authorization
  need a second curriculum after the strict core is mastered.
- Verify that mentor answers for every lab are recorded without turning the labs
  into rote checklists.

### What should be done in the future

1. Have the first intern complete the labs and record confusing or missing
   prerequisites.
2. Add short captured traces to labs where reading test assertions is
   insufficient.
3. Revise the curriculum from observed onboarding outcomes rather than adding
   speculative breadth.

### Code review instructions

- Begin with design 05 and confirm its reading order matches package ownership.
- Review design 06 against RFC/OIDC source captures and provider symbols.
- Review designs 07–09 against the executable models and tool limitations.
- Review design 10 against the production host, playbook, and release ledger.
- Execute the commands in reference 07 from a clean tree.

### Technical details

```text
new onboarding documents: 7
curriculum modules: 5
guided labs: 8
claim labels: normative, observed, inferred, open
writing style: foundational prose, direct models, no analogies
production code changes: none
release status: NOT APPROVED
```
