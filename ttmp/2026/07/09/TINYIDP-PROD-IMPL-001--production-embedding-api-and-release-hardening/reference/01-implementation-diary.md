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
    - Path: repo://pkg/embeddedidp/options.go
      Note: First production API implementation target
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md
      Note: Implementation design this diary tracks
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md
      Note: Durable 90-task phase ledger
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
