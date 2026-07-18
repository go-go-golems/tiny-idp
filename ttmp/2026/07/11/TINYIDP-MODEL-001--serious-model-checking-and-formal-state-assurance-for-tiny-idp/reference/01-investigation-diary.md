---
Title: Investigation Diary
Ticket: TINYIDP-MODEL-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/state_model_test.go
      Note: Existing generated interaction reference model
    - Path: repo://internal/fositeadapter/linearizability_test.go
      Note: Existing Porcupine history checking
    - Path: repo://internal/securitytrace/trace.go
      Note: Existing runtime temporal monitor
    - Path: repo://pkg/verifyplan/plan.go
      Note: Existing counterexample replay target
ExternalSources:
    - https://lamport.azurewebsites.net/tla/tutorial/intro.html
    - https://apalache-mc.org/docs
    - https://alloytools.org/book.html
Summary: Chronological record of the research, abstraction, design, experiments, implementation, counterexamples, and evidence produced by the tiny-idp model-checking program.
LastUpdated: 2026-07-12T00:40:00Z
WhatFor: Continuing the program without losing checker assumptions, failed models, counterexample interpretation, or implementation consequences.
WhenToUse: Read before resuming any TINYIDP-MODEL-001 task and update after every research or implementation interval.
---

# Investigation Diary

## Goal

Build a serious, reviewable model-checking program for tiny-idp that begins with
theory and literature, models small security-critical transition systems,
reproduces historical defects, exports counterexamples into Go regression tests,
and integrates bounded checker evidence without overstating what it proves.

## Step 1: Create the formal-state-assurance program and literature baseline

This step created a separate long-lived ticket rather than treating model
checking as another subsection of production hardening. It established the
program charter, copied the existing formal-methods research into a dedicated
source packet, captured current primary tool documentation, and converted the
prior model-checking assessment into an intern-ready research/design guide.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to improve model checking / get serious about it, including a proper research and literature and theoretical work phase. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Establish a dedicated research and implementation
program with formal-method theory as a gated first phase, a detailed intern
guide, precise phases/tasks, preserved sources, diary, validation, commits, and
reMarkable delivery.

**Inferred user intent:** Move from useful but narrow state-based tests to a
professional formal modeling discipline that can be maintained over time and
produce implementation-relevant counterexamples.

### What I did

- Created ticket `TINYIDP-MODEL-001` with design-doc, diary, index, tasks, and
  changelog.
- Inventoried the current model-oriented code and confirmed no `.tla`, `.als`,
  `.pml`, `.smv`, `.ivy`, or `.smt2` specifications exist.
- Reused the prior 1,536-line project report as evidence-backed starting
  material rather than rewriting the current-state inventory from memory.
- Copied twenty relevant saved sources and PDFs covering formal OAuth/OIDC,
  model-based security testing, runtime verification, linearizability, CHESS,
  typestate, stateful fuzzing, metamorphic testing, and fault injection.
- Captured eight additional primary pages using Defuddle: PlusCal/TLA+ tools,
  Apalache symbolic checking and installation, Alloy overview/book, Porcupine,
  and Quint documentation.
- Added a program charter, non-goals, intern reading order, six literature groups,
  and explicit theory deliverables to the design guide.
- Created ten implementation phases and 84 individually trackable tasks.
- Marked only completed ticket/scaffold/current-inventory/source tasks complete;
  checker experiments, models, integration, and governance remain open.

### Why

- A checker chosen before the abstraction and claim are understood produces
  formal-looking artifacts without a stable security meaning.
- The existing Rapid, Porcupine, monitor, failpoint, and VerificationPlan work
  already provides a domain vocabulary and regression bridge; the formal program
  should build on it.
- A separate ticket prevents research, model semantics, counterexamples, and CI
  evidence from being buried in a production-release ledger.

### What worked

- The prior source packet already contained the most important academic papers,
  including the primary linearizability, formal OAuth/OIDC, model-based testing,
  concurrency, and fault-injection material.
- Defuddle captured the official tool pages directly into the new ticket.
- The previous temporal textbook and project report supplied detailed current
  state, allowing this step to focus on program design and literature ordering.

### What didn't work

N/A. Some tool landing pages produce short Defuddle line counts because the
extractor preserves large logical lines; character counts confirm content was
captured. This known formatting behavior should be considered when reading the
source packet.

### What I learned

- The missing capability is not state vocabulary. It is exhaustive exploration
  of an implementation-independent transition system and a governed bridge from
  checker traces back to Go.
- Theory outputs need concrete review artifacts: glossary, annotated
  bibliography, assumption ledger, checker matrix, tutorial runs, and accepted
  model boundary.
- The first useful specification remains authorization interaction state; it can
  intentionally reproduce the exact historical forced-login counterexample.

### What was tricky to build

- The ticket must be exhaustive without implying that all 84 tasks are already
  authorized implementation changes. The ledger therefore separates completed
  research setup from open experiments, models, tooling, CI, and governance.
- Formal OAuth/OIDC research operates at a broader web-attacker level than the
  proposed local models. The guide repeatedly separates inherited Fosite/protocol
  assumptions from tiny-idp-owned interaction and persistence transitions.

### What warrants a second pair of eyes

- Review the literature ordering and add any foundational temporal-logic or
  refinement text required by the team.
- Review whether direct TLA+ actions or PlusCal should be the first authoring
  style for the intern.
- Review the first-model boundary before any model file is committed.

### What should be done in the future

1. Complete the glossary and annotated bibliography.
2. Pin and run TLC on the one-time-capability tutorial model.
3. Write and approve the authorization abstraction/assumption ledger.
4. Implement the first model only after those reviews.

### Code review instructions

- Read the design guide's program charter and literature phase first.
- Compare the current capability inventory with `state_model_test.go`,
  `linearizability_test.go`, `trace.go`, and `verification_scenario_test.go`.
- Review `tasks.md` for phase ordering and verify no implementation phase is
  marked complete.
- Inspect the `sources/` packet and confirm primary tool and paper provenance.

### Technical details

```text
ticket: TINYIDP-MODEL-001
source artifacts: 28
implementation phases: 10
tracked tasks: 84
existing general exhaustive formal specifications: 0
proposed first core specifications: 3
production code changes: none
```

## Step 2: Separate the system design from the theory reader and derive the tasks

The user challenged the structure directly: the task ledger appeared to have no
visible design source, while the primary guide appeared to concern mainly
literature. The criticism was correct. The original guide contained product
models and implementation phases, but its opening emphasized research and its
embedded six-phase outline no longer matched the ten phases in `tasks.md`.

### Prompt context

**User prompt (verbatim):** "ok but where do all the tasks come from, what is the design behind it? the design doc seems to only cncerne literature research?"

**Follow-up prompt (verbatim):** "read the skill on creating design docs for docmgr tickets"

**Assistant interpretation:** Stop publication, inspect the discrepancy, read
the governing docmgr and research/design skills, and create a document whose
architecture directly derives every phase and exit artifact.

### What I did

- Audited all headings in the original 2,000-line guide and compared its
  implementation section with all 84 tasks.
- Confirmed a concrete mismatch: the guide still exposed an older Phase 0–5
  outline, while the ledger used Phase 0–9.
- Read the complete `docmgr` skill and its 438-line reference.
- Read the complete `ticket-research-docmgr-remarkable` and
  `full-blown-tech-research-design` skills, plus the required writing-style and
  deliverable checklists.
- Added a 779-line primary system architecture document following the prescribed
  order: executive summary, scope, current evidence, gaps, architecture, data and
  APIs, model slices, flows, decision records, task derivation, file-level plan,
  validation, risks, open decisions, onboarding, and references.
- Reclassified the earlier guide in ticket navigation as the companion theory,
  literature, property-catalog, and case-study reader.
- Added an explicit five-input derivation for the backlog and mapped every phase
  to its design question and exit artifact.
- Related the new design doc to seven key implementation files using absolute
  paths and focused notes.

### Why

A task list is reviewable only when each task is necessary to produce a defined
system artifact. Research topics alone cannot justify parser, replay, CI,
governance, or crash-model tasks. The new document makes the evidence pipeline
the system and shows that literature qualification is one dependency within it.

### What worked

- The current code already exposes clean architecture anchors: interaction
  records, atomic consume, token transaction hooks, Rapid models, Porcupine
  histories, runtime monitor, and VerificationPlan driver.
- The exact skill structure forced the guide to distinguish current-state
  evidence, proposed APIs, decisions, flows, implementation phases, and testing.
- A separate focused design doc avoids destroying the extensive theory reader
  while giving an intern a clear entry point.

### What didn't work

The first design guide attempted to serve simultaneously as textbook, research
report, property catalog, case-study archive, and implementation RFC. All useful
content was present, but the navigation made the program appear research-led
rather than architecture-led. Adding more sections to that document alone did
not fully solve the information-architecture problem.

### What was tricky to build

- The new design needed to be independently understandable without duplicating
  all 2,000 lines of theory and case studies.
- The phase table had to explain not merely sequence but the evidence artifact
  that makes each phase complete.
- Proposed APIs had to remain designs rather than imply that `models/` or
  `internal/modelcheck/` already exists.

### What warrants a second pair of eyes

- Review the five backlog inputs and phase exit artifacts before accepting the
  84-task ledger.
- Review whether the evidence envelope outcome semantics are strict enough for
  release use.
- Review the proposed repository layout and whether model assets belong at
  top-level `models/`.
- Review the first toolchain decision, which remains explicitly proposed.

### What should be done in the future

1. Validate that every task maps to exactly one phase exit artifact; split or
   remove tasks that do not.
2. Complete Phase 0 identifiers and evidence schema before installing CI gates.
3. Perform Phase 1 tool qualification and seek maintainer approval.
4. Build the authorization abstraction ledger before writing the product model.

### Code review instructions

- Start with `design-doc/02-serious-model-checking-system-architecture-and-implementation-plan.md`.
- Check its task-derivation table against every Phase 0–9 heading in `tasks.md`.
- Use the original `design-doc/01-...md` as the theory and detailed property
  companion, not as the primary program map.
- Confirm the design doc's line-referenced claims against the seven related Go
  files.

### Technical details

```text
primary system design: 779 lines
companion theory/research guide: 2,066 lines
task phases: 10
tracked tasks: 84
related product files on primary design: 7
production code changes: none
```

## Step 3: Validate, commit, and publish the research/design bundle

### What I did

- Ran frontmatter validation on the primary design document.
- Ran `docmgr doctor --ticket TINYIDP-MODEL-001 --stale-after 30`; all checks
  passed.
- Ran `git diff --check` and staged only the new ticket, leaving unrelated OIDF
  source directories untouched.
- Committed the ticket as `87c8123` with message
  `Docs: design serious tiny-idp model checking program`.
- Dry-ran a six-document reMarkable bundle containing the index, system design,
  theory reader, diary, task ledger, and changelog.
- Uploaded the rendered bundle successfully.

### What worked

The dry run resolved all six inputs and the real command returned:

```text
OK: uploaded TINYIDP-MODEL-001 Serious Model Checking 87c8123.pdf -> /ai/2026/07/11/TINYIDP-MODEL-001
```

### What didn't work

N/A.

### What was tricky to build

The publication had to preserve both kinds of document without presenting them
as alternatives. The primary system design now appears before the companion
theory reader in the bundle.

### What warrants a second pair of eyes

Review the rendered diagrams and tables on the reMarkable, especially the wide
phase and gap-analysis tables.

### What should be done in the future

Begin with Phase 0 evidence semantics and Phase 1 qualification. Do not mark a
production model task complete merely because a tutorial checker run succeeds.

### Code review instructions

Review commit `87c8123`, starting with the ticket index and primary system
design. Confirm the open task state accurately distinguishes design completion
from formal-model implementation.

### Technical details

```text
docmgr doctor: all checks passed
git commit: 87c8123
bundle inputs: 6 Markdown documents
reMarkable destination: /ai/2026/07/11/TINYIDP-MODEL-001
bundle name: TINYIDP-MODEL-001 Serious Model Checking 87c8123.pdf
```
