---
Title: Investigation Diary
Ticket: TINYIDP-STATIC-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md
      Note: Prior design read during the investigation
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Inventory evidence and future implementation baseline
ExternalSources:
    - https://pkg.go.dev/golang.org/x/tools/go/analysis
    - https://codeql.github.com/docs/codeql-language-guides/analyzing-data-flow-in-go/
    - https://arxiv.org/abs/2105.13840
Summary: Chronological record of research, analyzer design, experiments, failures, implementation, evaluation, and release evidence for the tiny-idp static-analysis program.
LastUpdated: 2026-07-12T01:25:00Z
WhatFor: Preserves how static-analysis claims, abstractions, tools, and implementation decisions were derived and tested.
WhenToUse: Read before continuing TINYIDP-STATIC-001 and update after every meaningful research, implementation, evaluation, or publication interval.
---


# Investigation Diary

## Goal

Build a professional static-analysis and selected implementation-verification
program for tiny-idp. The program should connect security invariants to Go AST,
CFG, SSA, interprocedural, taint, typestate, mutation, and deductive evidence
without confusing absence of diagnostics with whole-program proof.

## Step 1: Establish the static-analysis research and design program

This step created a separate long-lived ticket because model checking and static
implementation analysis answer different questions. It inventoried the existing
custom analyzer prototype, preserved primary research and tool documentation,
and designed the path from dated ticket tooling to a maintained analyzer product.

The resulting design makes one vertical authorization property the first serious
milestone. It also keeps Gobra-style deductive verification as an explicit,
isolated experiment rather than attaching proof language to ordinary linter
results.

### Prompt Context

**User prompt (verbatim):** "alright, make the ticket, write out a detailed design doc."

**Assistant interpretation:** Create a dedicated docmgr ticket for serious
static analysis and implementation verification, perform enough research and
code inventory to produce an intern-ready architecture and implementation plan,
and preserve detailed tasks and diary evidence.

**Inferred user intent:** Give static analysis the same durable, theoretically
grounded program structure as model checking, while building on the substantial
custom analyzers already written for tiny-idp.

### What I did

- Read the complete docmgr, ticket research/design, full technology research,
  diary, and Defuddle skill instructions relevant to this deliverable.
- Created `TINYIDP-STATIC-001` with design doc, diary, index, tasks, changelog,
  sources, and scripts directories.
- Inspected the existing 772-line `auditlint` implementation, fifteen registered
  analyzers, eleven direct `analysistest` functions, and fixture tree.
- Re-read the existing 1,400-line security-invariant assurance architecture,
  especially its static-analysis program and analyzer engineering contract.
- Inspected the top-level module's Go and `golang.org/x/tools` versions.
- Searched for official Go analysis/SSA/callgraph material, IFDS research, Gobra,
  and CodeQL Go dataflow documentation.
- Saved nineteen primary and prior research artifacts under `sources/` using
  Defuddle, direct PDF download, and provenance-preserving copy.
- Wrote an 891-line design guide covering theory, current evidence, gaps,
  architecture, APIs, priority analyzers, tool choices, development lifecycle,
  mutation metrics, evidence semantics, eleven phases, testing, decisions,
  risks, open questions, onboarding, and references.
- Wrote 101 stable-ID tasks and marked only completed setup/inventory/source work
  complete.

### Why

- The existing analyzer prototype has demonstrated value but remains stored in a
  dated ticket and primarily recognizes local code shapes.
- Model checking explores abstract states; static analysis must separately check
  possible implementation structure and flows.
- Advanced analysis needs explicit may/must semantics, lattices, joins, summaries,
  callgraph assumptions, and unsupported-behavior policy to avoid false claims.
- Mutation and historical-defect benchmarks are necessary to show that a rule
  recognizes the defect class it claims to cover.

### What worked

- Existing work supplied a strong concrete rule inventory rather than a blank
  research exercise.
- Official Go APIs align with the requested native Go AST/analysis approach.
- The historical forced-reauthentication and ignored-`MustChangePassword` defects
  form an unusually strong first vertical benchmark.
- The prior assurance source packet already contained IFDS, typestate, security
  automata, static/dynamic invariant, and formal OIDC materials.
- Defuddle successfully captured all requested web documentation.

### What didn't work

The first Kagi search command incorrectly used an unsupported flag:

```text
surf kagi search --query 'site:pkg.go.dev/golang.org/x/tools/go/analysis buildssa ctrlflow facts official' --limit 10

Error: unknown flag: --limit
```

The same error occurred for the second chained query. I removed `--limit` and
reran all four searches successfully. No result or source capture was lost.

### What I learned

- The present analyzer program is broad but shallow in representation: fifteen
  useful rules exist, while general CFG/SSA/interprocedural foundations do not.
- The serious static-analysis milestone should not be “more rules.” It should be
  one precisely specified path-sensitive invariant with measured mutation
  sensitivity and repository precision.
- Trust must be purpose-specific. Parsing, hashing, CSRF, and syntactic validity
  do not create a universal trusted value.
- Deductive verification is plausible for a small pure kernel but is not a
  substitute for analyzing the provider and integration boundaries.

### What was tricky to build

- The design had to preserve the value of existing syntax rules without implying
  that higher-complexity SSA analysis is always superior.
- Static properties had to map to temporal model/runtime properties without
  claiming formal refinement between Go and TLA+.
- Tool comparisons had to distinguish complementary coverage from duplicated
  assumptions and duplicated green badges.
- The task ledger had to separate completed research scaffolding from all
  unimplemented analyzer, framework, mutation, proof, and CI work.

### What warrants a second pair of eyes

- Review the proposed `cmd/tinyidpvet` and internal package layout.
- Review the authorization obligation abstract domain before any SSA code exists.
- Review whether `analysis.Fact` can provide sufficient interprocedural precision
  or a whole-program driver is needed.
- Review the evidence outcome and suppression semantics for release use.
- Review the proposed Gobra kernel and adopt/no-adopt criteria.

### What should be done in the future

1. Complete Phase 0 rule/evidence semantics.
2. Run Phase 1 tutorial and baseline experiments with pinned commands.
3. Classify all fifteen current analyzers and close the fifteen-versus-eleven
   test inventory question.
4. Obtain architecture approval before moving ticket code into maintained
   product packages.
5. Implement the authorization vertical property only after its abstract domain
   and dependency summaries are reviewed.

### Code review instructions

- Start with the design guide's executive summary, current-state table, target
  architecture, priority analyzers, and decisions.
- Compare the table with `auditlint/main.go` lines 23–40 and each analyzer
  declaration.
- Inspect `tasks.md` and confirm only four setup/research tasks are checked.
- Inspect `sources/` for the official Go, CodeQL, IFDS, and Gobra foundations.
- Run `docmgr doctor --ticket TINYIDP-STATIC-001 --stale-after 30` after edits.

### Technical details

```text
ticket: TINYIDP-STATIC-001
design guide: 891 lines
task phases: 11
stable tasks: 101
source artifacts: 19
existing registered analyzers: 15
existing direct analysistest functions: 11
existing analyzer implementation: 772 lines
production code changes: none
```

## Step 2: Validate and commit the ticket baseline

This step made the research/design package a reproducible repository artifact.
It resolved the only docmgr vocabulary warning, checked Markdown hygiene, and
committed only the ticket plus its required vocabulary entry.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate and preserve the completed ticket baseline
without claiming that implementation phases are complete.

**Inferred user intent:** Make the design durable and ready for long-term
implementation tracking.

### What I did

- Ran frontmatter validation on the primary design document.
- Ran `docmgr doctor --ticket TINYIDP-STATIC-001 --stale-after 30`.
- Added the missing `security` topic to `ttmp/vocabulary.yaml` after doctor
  reported it as unknown on three documents.
- Reran doctor successfully and ran `git diff --check`.
- Confirmed four tasks checked and ninety-seven tasks open.
- Staged only the new ticket and vocabulary entry.
- Committed as `d8e9ef4` with message
  `Docs: design tiny-idp static analysis program`.

### Why

The ticket must be searchable, warning-free, and independently reviewable before
implementation begins. Open task state is part of the evidence boundary.

### What worked

`docmgr doctor` passed after registering the legitimate topic. The commit
contains 9,073 inserted lines across the design, diary, task ledger, source
packet, index, changelog, README, and vocabulary.

### What didn't work

The first doctor run reported:

```text
[WARNING] unknown_topics — unknown topics value(s): security (3 docs)
```

The documented fix succeeded:

```text
docmgr vocab add --category topics --slug security --description "Security architecture, analysis, verification, controls, and assurance evidence"
```

### What I learned

The repository vocabulary had security-adjacent topics but no general `security`
slug. Registering it is preferable to weakening the ticket metadata.

### What was tricky to build

The commit had to include the vocabulary change while excluding unrelated
untracked OIDF source trees elsewhere in `ttmp`.

### What warrants a second pair of eyes

Confirm that adding the general `security` topic is consistent with repository
taxonomy rather than preferring only `auth` and `oidc`.

### What should be done in the future

Begin Phase 0 and Phase 1. No production analyzer should be moved until the rule
catalog, evidence semantics, and migration decision are accepted.

### Code review instructions

- Review commit `d8e9ef4`.
- Run `docmgr doctor --ticket TINYIDP-STATIC-001 --stale-after 30`.
- Confirm `docmgr task list --ticket TINYIDP-STATIC-001` shows only baseline
  work complete.

### Technical details

```text
commit: d8e9ef4
doctor: all checks passed
checked tasks: 4
open tasks: 97
production code changes: none
```
