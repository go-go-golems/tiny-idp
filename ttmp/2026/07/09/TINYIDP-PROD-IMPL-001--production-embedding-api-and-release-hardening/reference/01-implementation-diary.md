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
    - Path: repo://go.mod
      Note: Phase 0 dependency and toolchain baseline
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
