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
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md
      Note: Durable 90-task phase ledger
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/various/phase3-password-load.json
      Note: Production Argon2 load evidence
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
