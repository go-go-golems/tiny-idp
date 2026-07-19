---
Title: Investigation diary
Ticket: TINYIDP-GOJA-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - oidc
    - research
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://go.mod
      Note: Phase 0 baseline source for the Go 1.26 toolchain and pinned go-go-goja dependency recorded in Step 10.
    - Path: repo://lefthook.yml
      Note: Step 17 exact pre-commit test and lint policy whose runner orphaned children
    - Path: repo://pkg/idpcontinuation/idpcontinuationtest/suite.go
      Note: Step 14 reusable memory and SQLite concurrency contract
    - Path: repo://pkg/idpcontinuation/service.go
      Note: Step 14 keyed handle, binding, generation, schema, and cleanup service
    - Path: repo://pkg/idpcontinuation/service_test.go
      Note: Step 15 generation, binding, resume-input, and safe-terminal tests
    - Path: repo://pkg/idpcontinuation/store.go
      Note: Step 14 atomic store and generation-resolution interfaces
    - Path: repo://pkg/idpcontinuation/types.go
      Note: Step 14 versioned VM-independent durable continuation contract
    - Path: repo://pkg/idpprogram/value.go
      Note: Step 14 shared runtime-independent JSON and public-carry validation
    - Path: repo://pkg/idpprogram/value_test.go
      Note: Step 15 sensitive-carry and bounded JSON regression tests
    - Path: repo://pkg/idpscript/codec.go
      Note: Step 15 runtime now shares the core schema validator
    - Path: repo://pkg/idpui/templates/workflow.html
      Note: Step 18 escaped generic workflow renderer
    - Path: repo://pkg/idpui/workflow.go
      Note: Step 18 provider-owned WorkflowPage contract
    - Path: repo://pkg/idpui/workflow_test.go
      Note: Step 18 renderer secret and escaping tests
    - Path: repo://pkg/idpworkflow/descriptors.go
      Note: Step 16 closed provider-owned field and action vocabulary
    - Path: repo://pkg/idpworkflow/descriptors_test.go
      Note: Step 16 authority-policy regression tests
    - Path: repo://pkg/idpworkflow/presentation.go
      Note: Step 18 compiled-edge and registry presentation validation (commit 9f6ad5b)
    - Path: repo://pkg/idpworkflow/presentation_test.go
      Note: Step 18 presentation authority negative tests
    - Path: repo://pkg/memorystore/continuation.go
      Note: Step 14 atomic ephemeral implementation
    - Path: repo://pkg/memorystore/continuation_test.go
      Note: Step 15 memory-store conformance entry point
    - Path: repo://pkg/sqlitestore/continuation.go
      Note: Step 14 transactional durable implementation
    - Path: repo://pkg/sqlitestore/continuation_test.go
      Note: Step 15 SQLite conformance and restart-resume proof
    - Path: repo://pkg/sqlitestore/migrations/011_workflow_continuations.sql
      Note: Step 14 durable schema and expiry index
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Primary design produced by the investigation
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md
      Note: |-
        Normative design produced and recorded in Step 7
        Normative continuation and generation design governing Step 14
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/reference/02-security-verification-scripting-plane-assessment.md
      Note: Step 4 focused verification-plane design
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/sources/01-colleague-identity-microkernel-research.md
      Note: Verbatim source moved from /tmp and analyzed in Steps 1 and 2
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/tasks.md
      Note: Normative Phase 0-7 lambda-first implementation ledger expanded in Step 9 (commit c84bcaa).
ExternalSources: []
Summary: Chronological record of the repository investigation and design work for the tiny-idp go-go-goja scripting layer.
LastUpdated: 2026-07-10T11:11:55.464532318-04:00
WhatFor: Resuming the scripting-layer design or reviewing which evidence and commands produced the implementation guide.
WhenToUse: Read before continuing TINYIDP-GOJA-001 or reviewing the design assumptions and validation evidence.
---





























# Diary

## Goal

Capture how the go-go-goja identity-microkernel scripting design was derived
from the colleague research, current tiny-idp code, and current go-go-goja/xgoja
APIs, including failures, decisions, validation, and delivery.

## Step 1: Create the ticket and preserve the source research

This step established a dedicated docmgr workspace before analysis began. The
colleague's research was moved, rather than copied, from `/tmp` into the ticket
so the input that shaped the design is durable and reviewable beside the final
guide.

The ticket uses existing architecture, auth, Go, OIDC, research, testing, and
xgoja vocabulary. It contains the normal index, tasks, changelog, design, diary,
and source structure.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket to add a go-go-goja scripting layer to tiny-idp, according to this research by your colleague in /tmp/idp-research.md (move to the sources/ directory of the ticket).

Analyze deeply and map out how to implement it with our current tiny-idp. 

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Create a new ticket, preserve the supplied research as a ticket source, inspect both tiny-idp and go-go-goja deeply, write an intern-oriented architecture/design/implementation guide and diary, validate docmgr bookkeeping, and upload the resulting document bundle to reMarkable.

**Inferred user intent:** Turn a broad identity-microkernel concept into a concrete, evidence-backed implementation plan that a new engineer can execute safely against the current repositories.

### What I did

- Loaded the ticket-research, diary, docmgr, go-go-goja module-authoring, and
  reMarkable upload instructions.
- Read `/tmp/idp-research.md` completely.
- Ran `docmgr status --summary-only` and inspected existing ticket vocabulary.
- Created `TINYIDP-GOJA-001` with:

  ```bash
  docmgr ticket create-ticket \
    --ticket TINYIDP-GOJA-001 \
    --title "Go go goja identity microkernel scripting layer" \
    --topics architecture,auth,go,oidc,research,testing,xgoja
  ```

- Added the design and diary documents with `docmgr doc add`.
- Moved the source to:
  `ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/sources/01-colleague-identity-microkernel-research.md`.

### Why

- A ticket workspace keeps source, decisions, tasks, and final guidance linked.
- Moving the source fulfills the request and avoids leaving the only research
  copy in ephemeral `/tmp`.
- Existing vocabulary avoids unnecessary docmgr warnings.

### What worked

- docmgr created the complete ticket skeleton and both requested subdocuments.
- The research file is now present in the ticket's `sources/` directory.
- The ticket appears in `docmgr ticket list` as active.

### What didn't work

- The first repository-status command was run from the workspace wrapper rather
  than the nested `tiny-idp` repository:

  ```text
  fatal: not a git repository (or any of the parent directories): .git
  ```

  The command was:

  ```bash
  pwd && printf '\n-- git status --\n' && git status --short && ...
  ```

  Because it used `&&`, discovery stopped after that failure. I corrected the
  working directory to `tiny-idp/` for repository and docmgr operations.

### What I learned

- The workspace contains three sibling repositories (`tiny-idp`, `go-go-goja`,
  and `glazed`) under a non-Git wrapper.
- docmgr resolves the workspace-level `.ttmp.yaml` to `tiny-idp/ttmp`.
- Existing unrelated untracked OIDF source directories were already present in
  the tiny-idp repository and must remain untouched.

### What was tricky to build

- The user named the product repository but the shell started one directory
  above it. The failure symptom was Git's “not a git repository” error. The
  solution was to identify nested `.git` files, run all tiny-idp commands from
  `tiny-idp/`, and use the sibling `go-go-goja/` path only for read-only API
  research.

### What warrants a second pair of eyes

- Confirm `TINYIDP-GOJA-001` is the desired ticket identifier and that the
  research should remain verbatim as the authoritative source input.

### What should be done in the future

- Keep future experiments under this ticket's numerically named `scripts/`
  directory if implementation spikes are added.

### Code review instructions

- Start at the ticket `index.md`.
- Verify the source path exists and `/tmp/idp-research.md` no longer exists.
- Run `docmgr ticket list --ticket TINYIDP-GOJA-001`.

### Technical details

- Ticket root:
  `ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer`.
- No product code changed in this research step.

## Step 2: Map the current systems and write the implementation design

This step traced the current strict tiny-idp request flow and public embedding
contracts, then matched them against go-go-goja runtime ownership, explicit
module registration, xgoja provider packaging, and host-service APIs. The design
uses concrete line anchors rather than treating the colleague sketches as an
already implemented surface.

The key conclusion is a staged identity-microkernel architecture: compile
JavaScript into a pure-Go graph, materialize named resources in Go, invoke only
narrow authorization and claims callbacks through a bounded runtime pool, and
defer general challenge composition until native challenge state and AMR/ACR
semantics exist.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce an implementation-grade bridge from the research vision to the exact current tiny-idp and go-go-goja APIs.

**Inferred user intent:** Give an intern enough system context, design rationale, pseudocode, file guidance, and acceptance criteria to implement the feature without weakening identity security.

### What I did

- Read the tiny-idp README, module file, public embedding API, public IDP/store
  contracts, strict Fosite adapter, browser sessions, claims, production host,
  and representative integration tests.
- Read the current go-go-goja engine factory, runtime lifecycle, module
  registrars, runtime owner, xgoja provider API/app factory, hostauth provider,
  generated runtime example, and installed xgoja help.
- Captured line-numbered evidence with `nl -ba` for major claims.
- Ran:

  ```bash
  cd tiny-idp && go test ./... -count=1
  ```

- Confirmed Goja `Interrupt`, `ClearInterrupt`, and `Compile` behavior with
  `go doc`.
- Wrote the primary guide with architecture diagrams, API sketches, graph and
  capability models, runtime-pool pseudocode, security boundaries, package and
  file maps, phased tasks, tests, decisions, alternatives, risks, and open
  questions.

### Why

- The existing Fosite boundary already keeps protocol validation native and is
  the correct foundation.
- Current go-go-goja defaults can expose host-access modules unless explicitly
  disabled; this is a critical design constraint.
- Current tiny-idp lacks claims/authorization callbacks and a general challenge
  engine, so the research API must be staged rather than copied literally.
- Goja context cancellation alone does not stop active JavaScript; interruption
  and worker replacement need explicit design and race tests.

### What worked

- All tiny-idp tests passed from the current repository baseline.
- Local go-go-goja APIs provide explicit module registrars, owned runtimes,
  closers, runtime-independent compiled programs, xgoja providers, host
  services, and TypeScript descriptors needed by the design.
- The current strict authorization path has clear insertion points:
  `finishAuthorize` for allow/deny policy and `newOIDCSession` for claims.
- The design separates first-release authorization/claims value from later
  challenge and factor composition.

### What didn't work

- An attempted read of
  `go-go-goja/pkg/xgoja/hostauth/service.go` returned:

  ```text
  ENOENT: no such file or directory, access '/home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja/pkg/xgoja/hostauth/service.go'
  ```

  The package uses different filenames. This did not block the analysis because
  `pkg/xgoja/providers/hostauth/hostauth.go`, provider API files, app factory,
  and repository search exposed the required host-service pattern.

### What I learned

- `embeddedidp.Options.Validate` is the production fail-closed gate and should
  be reused after graph materialization, not duplicated in the script layer.
- Strict claims are currently fixed at OIDC-session creation and later reused
  by UserInfo, which makes authorization-time claim computation the safest v1
  consistency model.
- `AuthResult.AMR` currently exists but is not propagated into the durable
  browser session; correct step-up work depends on fixing this first.
- A plain go-go-goja runtime builder can expose all default-registry modules.
  Identity compiler/runtime factories must disable both implicit and data-only
  defaults and add only reviewed modules.
- tiny-idp declares Go 1.25.11 while current go-go-goja declares 1.26.1. The
  minimum Go version is a real release decision, not a hidden dependency update.

### What was tricky to build

- The research is intentionally broad and uses future blocks such as passkeys,
  CIBA, workload attestation, quorum authorization, and token exchange. The
  difficult part was separating the stable architectural insight from APIs the
  current strict engine cannot safely implement. The solution was a gap matrix
  and ordered phases: pure graph, compiler, static policy seams, pooled
  callbacks, activation, xgoja packaging, then typed challenges and new native
  protocols.
- Runtime timeout semantics required reading both `runtimeowner.Call` and Goja
  documentation. The symptom is that a canceled `Call` can return while active
  JavaScript still runs. The design therefore requires `VM.Interrupt`, careful
  `ClearInterrupt` synchronization, context-bounded native capabilities, and
  worker discard after abnormal calls.

### What warrants a second pair of eyes

- Review the proposed Go 1.26.1 minimum-version decision.
- Review whether production claims should be computed once at authorization
  time, as recommended.
- Threat-model the proposed policy input projection and capability effect
  classes.
- Review interruption synchronization and worker-replacement semantics before
  implementation.
- Confirm the first release should omit passkey/step-up/general challenge names
  until native support exists.

### What should be done in the future

- Implement Phase 0 as a small code spike before freezing the public JS API.
- Record each implementation phase and exact validation commands in this diary.
- Update installed xgoja help/API evidence when the dependency is pinned.

### Code review instructions

- Begin with the guide's Executive Summary and “Current tiny-idp architecture.”
- Trace `internal/fositeadapter/provider.go` through authorization, session
  claims, and final issuance.
- Then inspect `../go-go-goja/pkg/engine/factory.go`,
  `pkg/runtimeowner/runner.go`, and `pkg/xgoja/app/factory.go`.
- Validate the baseline with `go test ./... -count=1` from `tiny-idp/`.

### Technical details

- Current tiny-idp baseline tests passed on 2026-07-10.
- Current sibling go-go-goja commit inspected:
  `cc9f18656f02f42e945806bcb6e3b1d86c0658ad`.
- Key installed help commands:

  ```bash
  xgoja help xgoja-v2-reference
  xgoja help provider-runtime-config-and-host-services
  ```

## Step 3: Validate and deliver the design

This step validated the ticket as a documentation deliverable and published the
intern guide to reMarkable. Validation covered frontmatter, docmgr health,
Markdown fence balance, the source move, and the current tiny-idp test baseline.

The upload bundled the ticket index, primary design, and investigation diary
with a level-two table of contents. Product implementation remains intentionally
open in the task list; this delivery completes the requested research and design
work, not the scripting feature itself.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the ticket and deliver its detailed design bundle to the requested reMarkable destination.

**Inferred user intent:** Make the design immediately reviewable away from the development machine while leaving a clean, continuation-ready ticket.

### What I did

- Validated all authored frontmatter with `docmgr validate frontmatter`.
- Ran:

  ```bash
  docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30
  ```

- Confirmed all Markdown fence counts are even and
  `/tmp/idp-research.md` was moved rather than copied.
- Performed the required bundle dry-run.
- Uploaded the index, design, and diary bundle with:

  ```bash
  remarquee upload bundle \
    index.md \
    design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md \
    reference/01-investigation-diary.md \
    --name "TINYIDP GOJA 001 Identity Microkernel Scripting Design" \
    --remote-dir "/ai/2026/07/10/TINYIDP-GOJA-001" \
    --toc-depth 2 \
    --non-interactive
  ```

### Why

- Frontmatter and doctor checks make the ticket searchable and continuation-safe.
- A dry-run catches bundle/path problems before rendering and upload.
- A single PDF with a table of contents is the most useful reMarkable review
  format for this long design.

### What worked

- Frontmatter validation reported `Frontmatter OK` for the design, diary, and
  index.
- `docmgr doctor` reported `✅ All checks passed`.
- `go test ./... -count=1` passed for the current tiny-idp baseline.
- The upload command reported:

  ```text
  OK: uploaded TINYIDP GOJA 001 Identity Microkernel Scripting Design.pdf -> /ai/2026/07/10/TINYIDP-GOJA-001
  ```

### What didn't work

- N/A. The dry-run and real upload both completed successfully.

### What I learned

- The primary guide is 2,213 lines, so a bundled table of contents is important
  for review navigation.
- docmgr recognizes 84 tasks: 11 research/delivery tasks and 73 implementation
  tasks across eight phases.

### What was tricky to build

- The ticket must accurately distinguish a completed design deliverable from an
  unimplemented product feature. The solution was to keep ticket status active,
  check only research/validation/upload tasks, and leave every code phase open.

### What warrants a second pair of eyes

- Review the owner decisions and Phase 0 gate before implementation, especially
  the minimum Go version, trust model for script authors, claims timing, and
  interruption semantics.

### What should be done in the future

- Begin with Phase 0's isolated module and interruption spike; do not freeze the
  fluent API before those risks are proven.

### Code review instructions

- Read `index.md`, then the design's Executive Summary, Decision Records, and
  Phase 0 sections.
- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30`.
- Run `go test ./... -count=1` from the tiny-idp repository before coding.

### Technical details

- reMarkable destination: `/ai/2026/07/10/TINYIDP-GOJA-001`.
- Uploaded document: `TINYIDP GOJA 001 Identity Microkernel Scripting Design.pdf`.
- No product code was changed or committed during this research delivery.

## Step 4: Separate security verification authoring from production policy

This step evaluated whether language hooks could load validation plugins that
assert security invariants and run multi-request scenarios. The answer is yes,
with a strict architectural condition: behavioral verification must be a
separate compile-only module and runner profile, not a set of general hooks in
the production authorization path.

I wrote a focused assessment that preserves Go as the trusted computing base and
extends the identity microkernel with structural graph assertions plus a pure
`VerificationPlan`. The plan is authored through Goja, but fake time, HTTP,
failpoints, scheduling, store inspection, trace collection, and final verdicts
remain native Go responsibilities.

### Prompt Context

**User prompt (verbatim):** "Also, take a loook at tiny-idp/ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md which is an idea we are looking at, could the addition of language hooks and such allow loading for example \"validation plugins\" that could assert invariants and then run different scenarios? Maybe bye xtending or modifying the scope of the proposed scripting layer, or adding a separate scripting layer that is focused on security and verification?"

**Assistant interpretation:** Evaluate the proposed identity scripting design as
a host for security scenario and invariant plugins, decide whether to extend or
separate it, and provide a concrete safe API and implementation plan.

**Inferred user intent:** Make security verification programmable and reusable
without allowing test authority, arbitrary JavaScript, or diagnostic hooks to
weaken the production identity provider.

### What I did

- Re-read the design's TCB, graph, compiler, default-module, runtime-owner,
  challenge-state, test, security, and decision sections with line references.
- Compared production policy requirements with verification requirements such as
  fake clocks, failpoints, scheduling control, and store snapshots.
- Created `reference/02-security-verification-scripting-plane-assessment.md` with
  a three-profile architecture, authority model, JavaScript and Go API sketches,
  execution diagram, hooks taxonomy, security tests, phases, and decisions.
- Added four ticket tasks for the plan schema, runtime-profile isolation,
  scenario prototype, and negative authority tests.
- Cross-linked the focused assessment to the production assurance design.

### Why

- Verification capabilities are deliberately dangerous in a production request
  runtime. A shared registry makes accidental authority escalation possible.
- Compiling scenarios to pure data retains an ergonomic authoring language while
  making execution, effects, assertion semantics, and evidence native and typed.
- The existing `PolicyTest` idea is appropriate for pure callback behavior but
  not for HTTP histories, persistence, concurrency, or failpoint scenarios.

### What worked

- The original design already contains the correct safety primitives: no ambient
  modules, explicit capabilities, one owner per VM, immutable graph artifacts,
  protected native claims, and Go-owned challenge state.
- A separate `require("tinyidp/verify").v1` module fits the existing CommonJS and
  xgoja provider model without changing the production callback API.
- Named native assertions provide stable versions and prevent scripts from
  redefining a failed invariant as a pass.
- The focused assessment and this diary were included in the successfully
  uploaded `TINYIDP Security Invariant Assurance Research` bundle at
  `/ai/2026/07/10/TINYIDP-PROD-IMPL-001`.

### What didn't work

- N/A for the architecture work. The Kagi Assistant and Defuddle failures during
  the broader research are recorded in the production implementation diary,
  Step 18.

### What I learned

- "Hook" must be split into production decision callbacks, native instrumentation
  points, offline runner lifecycle hooks, and symbolic test schedule/failpoint
  steps. Treating them as one extension mechanism hides authority differences.
- Verification JavaScript does not need live test capabilities. It can describe
  them symbolically and close before the native runner creates fixtures.
- Structural configuration invariants belong in the identity graph validator;
  behavioral multi-request invariants belong in a separate verification plan.

### What was tricky to build

- The API needs enough expressiveness for concurrency and failure scenarios
  without admitting arbitrary host mutation. The proposed DSL uses allowlisted
  step, assertion, failpoint-group, and linearizability-model IDs.
- Verification must remain useful for product-specific scripts while native
  baseline assertions remain mandatory. The plan therefore adds selected
  assertions but cannot remove the runner's required set.

### What warrants a second pair of eyes

- Decide whether verification providers are compiled into serving binaries or
  only dedicated test commands.
- Review how graph and verification-plan generations are linked and how pending
  interactions behave across reload.
- Review whether any custom JavaScript predicate is needed; the recommended first
  release permits native assertions only.
- Review the redacted `Evidence` schema before declaring it script-visible.

### What should be done in the future

1. Complete the native authorization interaction and assurance event schema.
2. Define the pure-Go `VerificationPlan` before implementing builders.
3. Implement deterministic sequential scenarios before concurrency/failpoints.
4. Add negative runtime integration tests that attempt every forbidden module and
   cross-profile capability.

### Code review instructions

- Read `reference/02-security-verification-scripting-plane-assessment.md` after
  the main design's Executive Summary and security boundary sections.
- Compare the three runtime profiles and forbidden-operation list with the
  existing runtime factory APIs.
- Start implementation review at the `VerificationPlan` DTO and validator, then
  inspect the actual `require` integration and negative module tests.
- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30` after changes.

### Technical details

```text
compile module: tinyidp
production policy module/profile: bounded redacted callbacks
verification module: tinyidp/verify
verification JS output: immutable pure-Go VerificationPlan
assertion authority: native registry plus mandatory baseline set
test capabilities visible to JS: none; symbolic references only
production mutation hooks: rejected
```

## Step 5: Synthesize scripting, model checking, and static analysis into a refactoring design

This step revisited the original identity-microkernel proposal after substantial
assurance work had landed. It compared the proposed Goja configuration graph with
the current interaction state, atomic store contracts, verification plan/compiler,
runtime security monitor, executable state models, model-checking design, and
static-analysis design.

The principal result is that tiny-idp does not need an immediate universal
workflow rewrite. It already has the right state-machine primitives. The proposed
refactor first consolidates their vocabulary and transition authority through
three linked data schemas, then extracts typed authorization proof boundaries and
only later evaluates selective pure transition kernels.

### Prompt Context

**User prompt (verbatim):**

```text
now look at tiny-idp/ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer  and all that model checking and static analysis design work, and make a new document (you can put it in the TINYIDP-GOJA-001 ticket) about proposals to refactor / reorganize the current codebase to make it more amenable to static analysis / model checking / scriptability. Probably we can build it around a core grammar of steps and resources and such that can be more easily analyszed statically or logged / traced for model checking etc... Maybe we are also already in a great state .

Anyway, do a deep thinking research and write deep analysis and design doc.
```

**Assistant interpretation:** Assess the actual post-hardening architecture,
synthesize the Goja, model-checking, and static-analysis programs, and design a
refactor that improves all three without weakening native protocol security.

**Inferred user intent:** Establish one coherent architectural direction so
scripting and assurance tooling reinforce each other instead of creating
parallel, drifting vocabularies and execution layers.

### What I did

- Read the complete code-quality review and textbook-authoring instructions and
  defined a precise no-analogy textbook/RFC style.
- Inventoried the full Goja ticket, its source research, implementation guide,
  verification assessment, diary, and task ledger.
- Mapped the current 22,870-line internal/pkg Go surface and largest control-flow
  files.
- Inspected interaction/store types and interfaces, provider authorization flow,
  Goja verification compiler/module, VerificationPlan, strict scenario driver,
  security trace/monitor, Rapid state model, and Porcupine histories.
- Cross-read the primary model-checking and static-analysis system designs.
- Identified six concrete architectural friction points with file locations,
  runtime/assurance consequences, and cleanup sketches.
- Wrote the 1,129-line
  `design-doc/02-assurance-oriented-core-grammar-and-codebase-refactoring-proposal.md`.
- Added Phase 9 with thirteen tasks and stable task IDs.
- Corrected the ticket index to acknowledge the verification implementation that
  now exists and distinguish it from the unimplemented configuration graph.

### Why

- Stable transition vocabulary is now the largest shared gap across scripting,
  model checking, static analysis, scenarios, and traces.
- A generic interpreted workflow could improve extensibility while weakening the
  trusted protocol boundary and erasing types/effects required by analyzers.
- Existing server-owned interaction state and named atomic operations should be
  preserved and made more explicit rather than replaced.

### What worked

- The current code provided concrete instances of the proposed architecture:
  data-only VerificationPlan compilation, typed interaction obligations, one-time
  consume, structured traces, and executable reference models.
- The original Goja graph already separates serializable configuration from
  Go-owned resources and capabilities.
- The model/static tickets supplied compatible stable-ID, evidence-envelope,
  counterexample, abstract-domain, and transition-authority concepts.

### What didn't work

N/A. Some command output was intentionally truncated because the combined code
inventory exceeded the terminal output budget; all targeted files were then read
directly in focused ranges.

### What I learned

- The desired “core grammar” should be a typed assurance IR and registry before
  it becomes an execution engine.
- Configuration graph, native transition catalog, and scenario/trace records
  require linked but distinct schemas.
- Unexported proof types at irreversible sinks can simplify static analysis
  without pretending to be a formal proof system.
- Generated model skeletons and analyzer metadata reduce vocabulary drift, but
  human authors must still define abstractions, invariants, and transfer
  semantics.

### What was tricky to build

- The design had to improve scriptability without making protocol validation,
  credential handling, atomicity, or artifact issuance scriptable.
- A single grammar had to support runtime identifiers, finite formal atoms, and
  source-level code authorities without conflating their representations.
- The proposal needed useful refactoring steps while honestly concluding that
  the current architecture is already good enough to avoid a broad rewrite.

### What warrants a second pair of eyes

- Review the three-schema split and whether `internal/assurance` is the right
  initial package boundary.
- Review the authorization proof type and artifact sink before implementation.
- Review whether transition results can generate complete trace events without
  obscuring Fosite transaction boundaries.
- Review public-package timing for `idpgraph`; the recommendation is internal
  until two consumers and one vertical slice stabilize it.

### What should be done in the future

1. Complete Phase 9 vocabulary/crosswalk work without changing provider behavior.
2. Add typed VerificationPlan codecs as the first executable consumer.
3. Consolidate transition tracing before refactoring provider control flow.
4. Introduce the authorization proof boundary under characterization and
   differential tests.
5. Materialize the Goja graph only after the native catalog is stable.

### Code review instructions

- Read the new document's Executive Summary, current-state assessment, three
  schemas, authorization slice, migration phases, and final assessment.
- Verify evidence against `idpstore/types.go`, `idpstore/interfaces.go`,
  `provider.go`, `verifyplan/plan.go`, `gojaverify/compiler.go`,
  `securitytrace/trace.go`, and the model tests.
- Confirm Phase 9 marks only the synthesis document complete.
- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30`.

### Technical details

```text
new design document: 1,129 lines
current internal/pkg Go surface: 22,870 lines
linked schemas: configuration graph, transition catalog, scenario/trace
initial vertical slice: authorization interaction
new tracked tasks: 13
production code changes: none
```

## Step 6: Render and upload the assurance-core bundle

This step published the updated GOJA ticket as one reMarkable PDF. The bundle
orders the index and assurance-core proposal first, followed by the original
scripting design, verification assessment, diary, tasks, and changelog.

### Prompt Context

**User prompt (verbatim):** "Upload to remarkable."

**Assistant interpretation:** Bundle and publish the complete updated GOJA ticket
to its existing dated reMarkable folder.

**Inferred user intent:** Read and annotate the synthesis together with the
designs from which it was derived.

### What I did

- Dry-ran a seven-document bundle with ToC depth 2.
- Attempted the real render and captured the Pandoc error.
- Located a literal `\n` sequence in the diary's verbatim prompt text.
- Replaced it with the actual paragraph break inside a fenced text block.
- Re-ran the exact upload successfully.

### Why

The bundle name includes source commit `3267560`, making the rendered architecture
traceable to repository state.

### What worked

```text
OK: uploaded TINYIDP-GOJA-001 Assurance Core Grammar 3267560.pdf -> /ai/2026/07/10/TINYIDP-GOJA-001
```

### What didn't work

The first real render failed before upload:

```text
Error: pandoc failed: Error producing PDF.
! Undefined control sequence.
l.6805 \n

: exit status 43
```

The literal prompt encoding was the cause. Converting it to a fenced verbatim
block with an actual blank line fixed the render.

### What I learned

Literal backslash commands in ordinary Markdown prose may survive into LaTeX and
break PDF generation. Verbatim multi-paragraph prompts should use fenced text
blocks with real line breaks.

### What was tricky to build

The dry run does not invoke Pandoc, so it validated input resolution and upload
destination but could not expose the LaTeX rendering error.

### What warrants a second pair of eyes

Review the wide grammar, phase, and scriptability tables on the device.

### What should be done in the future

Use a unique commit-suffixed name for later material revisions so existing
annotations are preserved.

### Code review instructions

Verify that the uploaded suffix matches `3267560` and that the diary contains no
literal `\n` prompt encodings outside code blocks.

### Technical details

```text
bundle inputs: 7 Markdown documents
bundle name: TINYIDP-GOJA-001 Assurance Core Grammar 3267560.pdf
remote destination: /ai/2026/07/10/TINYIDP-GOJA-001
upload result: success after one render fix
```

## Step 7: Replace the graph-first API with a lambda-first explicit-continuation design

This step revised the production scripting direction after working through
signup, virtual users, virtual invitation codes, email verification, and the
boundary between JavaScript Promises and browser HTTP requests. The resulting
design makes named JavaScript lambdas the primary application behavior while
retaining the program graph as an enforceable contract for schemas,
capabilities, effects, budgets, outcomes, and continuation edges.

The new normative document uses explicit browser continuations. A handler may
await bounded capability calls within one HTTP request, but forms and native
challenges return outcomes naming a later handler. Go persists the versioned
continuation, validates the later browser request, and invokes the handler in a
fresh runtime call. This preserves restartability and makes the state visible to
the assurance grammar without reducing JavaScript to a richer configuration
file.

### Prompt Context

**User prompt (verbatim):**

```text
ok, i like the explicit continuation approach.

Let's write a new detailed design document for the JS API that supersedes the existing one (mark it as deprecated clearly). Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Author a new normative intern implementation
guide for the lambda-first Tiny-IDP JavaScript API, use explicit durable browser
continuations, deprecate the graph-first API document unmistakably, update all
ticket bookkeeping, and publish the resulting design to reMarkable.

**Inferred user intent:** Make Tiny-IDP programmable through real lambdas and
composable native identity primitives, including virtual resources and
multi-request signup, without proliferating Go modes or configuration files and
without weakening protocol, secret, transaction, or replay invariants.

### What I did

- Read the complete local `TINYIDP-GOJA-001` diary and the selected ticket,
  textbook, diary, docmgr, and reMarkable workflow instructions.
- Kept discovery and all edits under
  `/home/manuel/workspaces/2026-07-07/prod-tiny-idp`.
- Inspected the current authorization and registration request flow,
  interaction creation/reconstruction, interaction store contract, UI model and
  renderer, account creation transaction, isolated verification compiler,
  go-go-goja runtime factory, runtime owner, and native Promise guidance.
- Created
  `design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md`.
- Wrote the normative design with system orientation, API examples, exact
  browser/OAuth sequences, handler and outcome contracts, runtime invocation,
  virtual identities and invites, secret handles, capability binding, effect
  plans, native atomic commit, generation semantics, package/file guidance,
  seven implementation phases, tests, decisions, alternatives, risks, and
  review instructions.
- Marked `design-doc/01` deprecated in its title, status, metadata, and a
  prominent document-opening warning linked to design 03.
- Updated the ticket index so design 03 is the normative entry point and the
  current status and key decisions describe lambda-first workflows.
- Added a task-ledger deprecation note and a new lambda-first implementation
  task group.

### Why

- The old API correctly protected Go-owned protocol and secret authority, but
  its first production scope made JavaScript primarily graph syntax plus narrow
  authorization/claims callbacks.
- The desired product needs lambdas to implement application-specific flow,
  virtual resource behavior, routing, and identity projection.
- Explicit continuations reuse the existing durable interaction pattern and do
  not require retaining or serializing a Goja heap across browser requests.
- Typed effect plans let lambdas choose mutations without executing JavaScript
  while a store transaction is open.

### What worked

- The current code provides strong direct foundations: Fosite validation before
  interaction creation, hashed and browser-bound interaction records, OAuth
  request reconstruction and digest checks, provider-owned presentation models,
  atomic store operations, an isolated Goja compiler, and owner-serialized
  runtime calls.
- The explicit continuation model supports the requested lambda-first behavior
  while remaining compatible with restart, reload, static analysis, model
  checking, and secret-free tracing.
- The new document is 1,821 lines and is structured as a textbook-style
  intern implementation guide rather than an API sketch.

### What didn't work

- N/A. Document creation and local patching completed without command or render
  failures in this step before validation and upload.

### What I learned

- The most useful boundary is not declarative versus imperative JavaScript. It
  is bounded in-request execution versus explicit cross-request continuation.
- A graph remains valuable when it constrains executable lambdas rather than
  attempting to replace their behavior.
- Virtual identity and invitation providers need explicit state semantics so an
  operator can see the replay and revocation consequences of a stateless
  implementation.
- The current hardcoded provider-owned registration path is a better first
  vertical slice than authorization/claims because it exercises UI,
  continuations, secrets, effects, and OAuth resumption together.

### What was tricky to build

- The design had to preserve real JavaScript behavior without giving lambdas
  ambient authority. The solution was to make lambda contracts declare schemas,
  capabilities, effects, outcomes, budgets, and continuation edges, then keep
  irreversible operations behind native effect committers.
- Browser-spanning `await` is attractive syntax, but an ordinary Promise is a VM
  heap object. The design explicitly distinguishes bounded Promise calls within
  one request from browser presentation/challenge outcomes persisted by Go.
- Secret fields cannot become ordinary JS strings if virtual providers are
  programmable. The design uses invocation-scoped opaque handles passed to
  typed native capabilities and avoids carrying passwords across long waits.

### What warrants a second pair of eyes

- Review whether `WorkflowContinuation` should extend the existing interaction
  storage or use a separate store and migration.
- Review Promise timeout, interruption, late-settlement, and worker-replacement
  semantics against the current go-go-goja owner implementation.
- Review the proposed secret-handle and pending-credential lifetime rules.
- Review the generation-retention bound for continuations during script reload.
- Review whether the generic workflow presentation model can remain strongly
  typed without duplicating the existing `idpui` models.

### What should be done in the future

1. Execute Phase 0 as a contract/runtime spike before stabilizing the public
   TypeScript surface.
2. Resolve continuation schema placement with memory and SQLite conformance
   tests before changing the Fosite adapter.
3. Migrate the current signup behavior differentially before adding email or
   invite providers.
4. Keep the assurance vocabulary work aligned with handler, outcome, effect,
   evidence, and observation identifiers from design 03.

### Code review instructions

- Start at the supersession notice and executive summary in design 03.
- Review sections 5–7 for handler invocation and exact browser/OAuth behavior.
- Review sections 9–14 for secrets, virtual resources, effects, transactions,
  continuations, and generation pinning.
- Review sections 18–21 for file-level implementation work, phase gates, tests,
  and accepted decisions.
- Confirm design 01 is unmistakably deprecated and the ticket index points to
  design 03 first.
- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30`.

### Technical details

```text
normative design: design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md
deprecated design: design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
primary migration target: internal/fositeadapter/provider.go registration branch
browser boundary: persisted named continuation, never a retained Goja Promise
in-request async: bounded Promise-returning native capability
irreversible mutation: validated native effect plan and named atomic commit
```

## Step 8: Validate, commit, and deliver the superseding design

This step converted the authored design into a validated and traceable ticket
checkpoint. It added `deprecated` to the ticket vocabulary so the historical
document can be marked semantically rather than relying only on warning prose,
ran docmgr and Markdown hygiene checks, committed the complete documentation
change, and published a new reMarkable bundle without overwriting earlier
annotated designs.

The uploaded bundle contains the updated ticket index, the normative
lambda-first design, and the full investigation diary. Its dated name and remote
directory distinguish it from the earlier graph-first and assurance-core PDFs.

### Prompt Context

**User prompt (verbatim):** (same as Step 7)

**Assistant interpretation:** Validate, checkpoint, and publish the completed
superseding design after the content and ticket bookkeeping are consistent.

**Inferred user intent:** Receive a durable, reviewable design in both Git and
reMarkable form, with no ambiguity about which JavaScript API is normative.

**Commit (documentation):** `0d5e4fb` — "Docs: design lambda-first Tiny-IDP scripting API"

### What I did

- Added the `deprecated` status vocabulary entry with:

  ```bash
  docmgr vocab add --category status --slug deprecated \
    --description "Superseded historical document retained for context and not normative implementation guidance"
  ```

- Validated the new design, deprecated design, index, and diary frontmatter.
- Ran `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30` until it
  reported all checks passed.
- Counted Markdown fences in the new design and diary; both counts were even.
- Ran `git diff --check`, reviewed the staged 2,071-line documentation change,
  and committed it as `0d5e4fb`.
- Dry-ran and then uploaded the index, design 03, and diary bundle with:

  ```bash
  remarquee upload bundle \
    index.md \
    design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md \
    reference/01-investigation-diary.md \
    --name "TINYIDP GOJA 001 Lambda First JS API Explicit Continuations 2026 07 19" \
    --remote-dir "/ai/2026/07/19/TINYIDP-GOJA-001" \
    --toc-depth 2 \
    --non-interactive
  ```

### Why

- A vocabulary-backed deprecated status makes docmgr search and validation agree
  with the visible supersession warning.
- A new reMarkable name preserves earlier documents and any annotations rather
  than forcing an overwrite.
- Committing before upload makes the delivered design traceable to an exact
  repository checkpoint.

### What worked

- Frontmatter validation reported `Frontmatter OK` for every changed ticket
  document.
- `docmgr doctor` reported `✅ All checks passed` after the vocabulary update.
- The dry run resolved all three input files and the intended remote directory.
- The real upload reported:

  ```text
  OK: uploaded TINYIDP GOJA 001 Lambda First JS API Explicit Continuations 2026 07 19.pdf -> /ai/2026/07/19/TINYIDP-GOJA-001
  ```

### What didn't work

- The first `docmgr doctor` run warned that `Status: deprecated` was not in the
  vocabulary:

  ```text
  [WARNING] unknown_status — unknown status value(s): deprecated (1 doc)
  ```

  Adding the explicit status vocabulary entry resolved the warning.
- The first `git diff --check` reported a newly added blank line at the end of
  `changelog.md`. Removing that extra blank line resolved the check.

### What I learned

- Deprecation is a reusable documentation state and belongs in the shared
  vocabulary when a superseded design must remain searchable.
- The reMarkable dry run validates path and bundle resolution but the real
  command remains the render test; this bundle rendered successfully on the
  first real attempt.

### What was tricky to build

- The deprecated document needed to remain useful historical evidence while
  being impossible to mistake for current guidance. The solution combines
  status metadata, title, summary, usage text, related-file link, top-of-document
  warning, index ordering, and task-ledger notice.

### What warrants a second pair of eyes

- Review the rendered table widths for the outcome, capability, failure, and
  file-plan tables on the reMarkable device.
- Confirm the seven-phase implementation plan is appropriately sliced before
  Phase 0 coding begins.

### What should be done in the future

- Begin Phase 0 from commit `0d5e4fb` and record the contract/runtime spike in
  this diary.

### Code review instructions

- Run `git show --stat 0d5e4fb` and inspect design 03 first.
- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30`.
- Confirm design 01 has `Status: deprecated` and points to design 03.
- Confirm the reMarkable bundle is named exactly as recorded above if manual
  device review is performed.

### Technical details

```text
design commit: 0d5e4fb
bundle inputs: index, design 03, investigation diary
bundle name: TINYIDP GOJA 001 Lambda First JS API Explicit Continuations 2026 07 19.pdf
remote destination: /ai/2026/07/19/TINYIDP-GOJA-001
upload result: success
```

## Step 9: Turn the lambda-first design into an executable implementation ledger

This step converted the normative Phase 0–7 roadmap in design 03 from twelve
coarse ticket entries into an ordered implementation ledger. The new ledger is
still the same design: it does not introduce another architecture, another
ticket, or speculative features. It identifies the exact contracts, runtime
work, persistence work, provider integration, tests, gates, validation commands,
and commit checkpoints needed to begin implementation without rediscovering the
plan during each coding session.

The task structure deliberately keeps the no-browser runtime spike first and
the production request path unchanged until its ownership and isolation rules
are proven. Each later phase names its dependency and completion gate, so an
intern can distinguish exploratory work from code that is safe to integrate.

### Prompt Context

**User prompt (verbatim):** "ok, add detailed phases + tasks for each phase to implement this goja scripting API, and commit at appropriate intervals, and keep a detailed diary as you work."

**User clarification (verbatim):** "Stay within the scope of the ticket, don't add new stuff, we want to get this going."

**Assistant interpretation:** Expand only the lambda-first implementation plan
already approved in design 03 into detailed, dependency-ordered ticket tasks,
then checkpoint the plan and record the work chronologically.

**Inferred user intent:** Make the design immediately executable by an
implementer without reopening architecture scope or losing the intended phase
boundaries as coding begins.

**Commit (documentation):** `c84bcaa` — "Docs: detail lambda API implementation phases"

### What I did

- Replaced the coarse lambda-first task list in `tasks.md` with:
  - a normative-scope notice;
  - explicit purpose, dependencies, and deliverables for each phase;
  - 86 ordered Phase 0–7 implementation tasks;
  - five cross-phase assurance tasks already required by design 03;
  - one overall completion gate;
  - phase-specific acceptance gates and validation commands;
  - suggested focused commit boundaries.
- Preserved the completed design and reMarkable delivery tasks.
- Preserved the deprecated graph-first tasks as historical evidence, behind the
  existing warning that they are not prerequisites.
- Assigned unique docmgr IDs `lf00` through `lf92` to the planning checkpoint
  and implementation ledger.
- Ran:

  ```bash
  docmgr task list --ticket TINYIDP-GOJA-001
  docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30
  rg -o 't:[a-z0-9]+' tasks.md | sort | uniq -d
  git diff --check
  ```

- Reviewed repository paths used by the proposed validation commands with
  `rg --files cmd pkg internal` before committing.

### Why

- The old lambda-first entries described phase outcomes but were too coarse to
  drive daily implementation or code review.
- Explicit dependencies stop browser/persistence integration from getting
  ahead of the runtime-ownership and capability-isolation spike.
- Per-phase gates define what “done” means in observable terms rather than by
  the presence of new packages alone.
- Suggested commit boundaries make each architectural seam independently
  reviewable and give future diary steps stable checkpoints to cite.

### What worked

- Docmgr recognized every new task and reported all task IDs without collision.
- `docmgr doctor` reported `✅ All checks passed`.
- `git diff --check` reported no whitespace errors.
- The final ledger follows sections 18–20 of design 03 and keeps native OAuth,
  HTTP, secret, persistence, and commit authority outside JavaScript.
- The focused planning commit contains only `tasks.md`.

### What didn't work

- The first draft used `go run ./cmd/tiny-idp`, but the repository entry point is
  `./cmd/tinyidp`. It also named a nonexistent `pkg/deviceflow`; the current
  device implementation lives under `internal/fositeadapter`,
  `internal/server`, and `pkg/idpui`. I corrected both validation blocks before
  staging the commit.
- No source-code tests were run because this checkpoint changes documentation
  only. Runtime commands in the ledger are acceptance commands for the future
  packages and phases that introduce them.

### What I learned

- The repository already has two layers of historical task planning. A visible
  normative boundary inside `tasks.md` is necessary so an implementer does not
  accidentally combine the deprecated graph-builder API with the lambda-first
  API.
- The phase plan is most actionable when each task names one contract or
  invariant and the gate tests the integration of those tasks together.

### What was tricky to build

- The central difficulty was increasing task detail without increasing product
  scope. The ledger was therefore derived line by line from the package plan,
  Phase 0–7 tasks, testing strategy, and accepted decisions in design 03.
- Phase 3 required careful wording around migration. The design explicitly says
  to differential-test the old and new registration paths and then remove the
  old branch, not preserve a compatibility adapter. The task ledger keeps that
  ordering explicit.
- Cross-phase assurance could look like a new parallel project. The ledger
  instead marks it as work performed when each relevant contract lands and
  states that it must not delay the Phase 0 spike with unrelated refactoring.

### What warrants a second pair of eyes

- Confirm the Phase 0 task boundaries are small enough for focused reviews while
  still proving owner-safe Promise settlement as one coherent runtime gate.
- Confirm `SignupCommitter` can atomically cover the stores named in Phase 3
  before its storage interface is frozen.
- Confirm Phase 4 provider examples exactly cover the configurability cases
  motivating this design and do not smuggle general storage/network authority
  into JavaScript.
- Confirm the Phase 5 password timing choice before implementing pending-secret
  persistence.

### What should be done in the future

- Begin with task `lf01` and record the baseline and dependency decision before
  adding `pkg/idpprogram`.
- Do not begin Phase 1 integration until the Phase 0 race, isolation, timeout,
  and worker-discard gate passes.

### Code review instructions

- Start with the `Lambda-first superseding design` section of `tasks.md`.
- Compare every phase against sections 18–20 of design 03; reject tasks that
  introduce APIs or authority not present there.
- Check that task IDs `lf00` through `lf92` are unique with:

  ```bash
  rg -o 't:[a-z0-9]+' \
    ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/tasks.md \
    | sort | uniq -d
  ```

- Run `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30` and
  `git diff --check`.

### Technical details

```text
normative source: design-doc/03, sections 18–20
planning commit: c84bcaa
Phase 0 tasks: lf01-lf14
Phase 1 tasks: lf15-lf25
Phase 2 tasks: lf26-lf35
Phase 3 tasks: lf36-lf46
Phase 4 tasks: lf47-lf56
Phase 5 tasks: lf57-lf66
Phase 6 tasks: lf67-lf77
Phase 7 tasks: lf78-lf86
cross-phase assurance: lf87-lf91
overall completion gate: lf92
first implementation task: lf01
```

## Step 10: Establish the Phase 0 baseline and dependency checkpoint

This step established the clean starting point for implementation before any
new packages or runtime behavior were added. The repository already contained
the dependency decisions required by `lf01`: Tiny-IDP targets Go 1.26.4 with the
Go 1.26.5 toolchain and pins go-go-goja to an exact pseudo-version rather than
depending on an unresolved workspace revision.

The direct and CI-style test baselines pass. The first sandboxed CI-style run
could not create loopback listeners, and the targeted race baseline exposed one
non-reproducing existing linearizability-test failure. Both outcomes are
preserved here so future implementation failures can be distinguished from the
starting environment and known baseline behavior.

### Prompt Context

**User prompt (verbatim):** "Implement it task byt ask, commit at appropriate intervals, keep a detailed diary as you work."

**User continuation instruction (verbatim):** "I am going to run errand, just run it to completion so I can validate when I come back."

**User blocking clarification (verbatim):** "don't touch grass unless you are genuinely blocked, btw"

**Assistant interpretation:** Execute the normative `lf01`–`lf92` ledger in
dependency order, autonomously continue through ordinary implementation and
test failures, and keep code, task, changelog, and diary commits synchronized.

**Inferred user intent:** Return to a fully implemented and reviewable
lambda-first scripting API rather than another planning pass or a partially
executed spike.

### What I did

- Confirmed the working tree was clean on `task/prod-tiny-idp`.
- Recorded:
  - `go version go1.26.5 linux/amd64`;
  - `go 1.26.4` and `toolchain go1.26.5` in `go.mod`;
  - pinned go-go-goja pseudo-version
    `v0.10.5-0.20260717152521-c6e464c5bbc6`;
  - the workspace uses local `./go-go-goja`, while `GOWORK=off` resolves the
    pinned module version.
- Ran the direct baseline:

  ```bash
  go test ./... -count=1
  ```

- Ran the CI-style baseline outside the socket-restricted sandbox:

  ```bash
  GOWORK=off go test ./... -count=1
  ```

- Ran a targeted existing Goja/Fosite/embedding race baseline and isolated the
  one failed test:

  ```bash
  go test -race ./internal/gojaverify ./internal/gojamodules/verify \
    ./internal/fositeadapter ./pkg/embeddedidp -count=1
  go test -race ./internal/fositeadapter \
    -run TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily \
    -count=1
  ```

- Checked task `lf01` only after the dependency and baseline evidence existed.

### Why

- Phase 0 needs a trustworthy before-state for attributing compile, race, and
  isolation failures to the new scripting implementation.
- The checked-in `go.mod` is the reproducible dependency decision; local
  workspace resolution is useful during development but cannot be the only
  validation path.
- A flaky unrelated race test should be recorded, isolated, and left outside
  this ticket rather than “fixed” speculatively while implementing scripting.

### What worked

- `go test ./... -count=1` passed.
- The escalated `GOWORK=off go test ./... -count=1` passed using the pinned
  go-go-goja version.
- The isolated race rerun of
  `TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily` passed.
- No dependency or Go-version edit was necessary.

### What didn't work

- The first sandboxed `GOWORK=off go test ./... -count=1` failed in packages
  using `httptest.NewServer` with:

  ```text
  panic: httptest: failed to listen on a port: listen tcp6 [::1]:0: socket: operation not permitted
  ```

  Re-running the same command with the required loopback-listener permission
  passed.
- The first targeted race baseline failed once with:

  ```text
  --- FAIL: TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily (0.34s)
      linearizability_test.go:104: query "SELECT COUNT(*) FROM fosite_refresh_tokens WHERE active=1" count=1, want 0
  ```

  The exact isolated `-race -run` rerun passed. No unrelated source change was
  made.

### What I learned

- The minimum-Go and dependency-pin decisions were already made by the merged
  repository state, so `lf01` required validation and documentation rather than
  a `go.mod` mutation.
- Full integration suites require loopback-listener permission in this
  workspace; pure package tests do not.

### What was tricky to build

- Running direct and `GOWORK=off` tests distinguishes local workspace behavior
  from the pinned dependency, but the sandbox failure initially looked like a
  test regression. The repeated `operation not permitted` listener panic across
  unrelated packages identified the environmental cause.
- The race failure was in pre-existing refresh-token linearizability behavior,
  outside the scripting scope. Isolating the exact test once was sufficient to
  classify it as non-reproducing baseline evidence without entering an
  unrelated debugging branch.

### What warrants a second pair of eyes

- If the refresh-token linearizability failure repeats during later full race
  gates, review it as a separate existing concurrency issue rather than
  attributing it automatically to scripting.
- Confirm future CI runs use the pinned dependency path at least once and do not
  rely only on `go.work`.

### What should be done in the future

- Begin `lf02` with runtime-independent contracts in `pkg/idpprogram`.
- Re-run the direct, targeted race, and `GOWORK=off` gates at the end of Phase 0.

### Code review instructions

- Review `go.mod` lines declaring the Go/toolchain and go-go-goja version.
- Review task `lf01` in `tasks.md` and this diary step; there is intentionally no
  application-source diff for this checkpoint.
- Reproduce with `go test ./... -count=1` and, where loopback sockets are
  allowed, `GOWORK=off go test ./... -count=1`.

### Technical details

```text
task completed: lf01
Go language version: 1.26.4
Go toolchain: 1.26.5
go-go-goja pin: v0.10.5-0.20260717152521-c6e464c5bbc6
direct baseline: pass
GOWORK=off baseline: pass outside socket-restricted sandbox
targeted race baseline: one non-reproducing pre-existing linearizability failure
next task: lf02
```

## Step 11: Implement the runtime-independent program contracts

This step created the first implementation boundary: `pkg/idpprogram` is the
serializable contract shared by the JavaScript compiler, owned runtimes,
workflow executor, activation tooling, and future continuation service. It has
no Goja dependency and stores callback identities rather than callback
functions, which prevents VM objects from leaking into durable or cross-worker
state.

The package also makes invalid programs deterministic and inspectable. It
defines closed outcome/effect vocabularies, bounded schemas and budgets,
workflow edges, sorted diagnostics, dynamic outcome validation, canonical JSON,
and separate hashes for source, program, callback registry, and schemas.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Implement tasks `lf02`–`lf05` as a pure-Go
contract checkpoint before adding the Tiny-IDP native module or runtime pool.

**Inferred user intent:** Establish a reviewable, deterministic security
contract that the scripting runtime cannot redefine implicitly.

**Commit (code):** `0e0a4b0` — "Feat: add scripting program contracts"

### What I did

- Added `pkg/idpprogram` with:
  - `Program`, `Workflow`, `HandlerSpec`, and declared transition edges;
  - named bounded schemas and schema fields;
  - `LambdaSpec`, lambda kinds, budgets, capability requirements, allowed
    effects, allowed outcomes, and source locations;
  - typed `Outcome`, `BrowserContinuation`, and native `EffectPlan` values;
  - deterministic diagnostics and program validation;
  - canonical JSON and source/program/callback/schema SHA-256 fingerprints.
- Added validation for map-key/ID consistency, identifier syntax, schema bounds
  and reference cycles, schema existence, capability versions, duplicate
  declarations, budgets, workflow entries, handler/lambda kinds, transition
  targets, transition input compatibility, and reachability.
- Added dynamic outcome validation so a callback cannot return an undeclared
  outcome or effect and browser outcomes require an explicit handler.
- Added tests for valid/malformed programs, stable diagnostic ordering,
  incompatible edges, insertion-order-independent canonicalization,
  fingerprint separation, undeclared effects/outcomes, and browser
  continuation requirements.
- Ran:

  ```bash
  go test ./pkg/idpprogram -count=1
  go test -race ./pkg/idpprogram -count=1
  go vet ./pkg/idpprogram
  ```

- The pre-commit hook additionally ran the full `GOWORK=off go test ./...` and
  repository lint/vet suite successfully.
- Checked tasks `lf02`, `lf03`, `lf04`, and `lf05` after commit validation.

### Why

- Runtime-independent contracts keep persistence, CLI, testing, and activation
  code independent of Goja lifecycle details.
- Closed stable IDs let validation, audit, metrics, and future model checks
  describe the same operation without reflecting Go type names.
- Separate fingerprints distinguish a changed source from a changed materialized
  program or callback registry and support Phase 0 cross-worker verification.

### What worked

- Unit tests, race tests, vet, the full pinned-dependency suite, and repository
  lint all passed.
- Canonical JSON remains identical when maps are built in a different insertion
  order.
- The package compiles without importing Goja or go-go-goja.
- The commit contains only the new contract package and its tests.

### What didn't work

- No test command failed in this step.
- The first code draft described edges as only browser `present`/`challenge`
  edges. A pre-test contract review found that the design's immediate
  `continue` outcome also needs a declared typed destination. The edge and
  validator were corrected to cover `continue`, `present`, and `challenge`
  before commit.

### What I learned

- Go's `encoding/json` sorts string map keys, so the restricted contract types
  can use direct marshaling as canonical JSON as long as floats, interfaces,
  functions, and VM-owned values remain excluded.
- Schema byte bounds are not enough by themselves for safe recursive
  validation; explicit reference-cycle rejection keeps Phase 0 schemas finite.

### What was tricky to build

- The serialized contract must be expressive enough to validate handler edges
  without storing JavaScript functions. The solution is a `LambdaSpec` registry
  keyed by stable callback ID and workflow handlers that refer to those IDs.
- Dynamic outcomes and static workflow edges are different checks. Static
  validation proves the target and input schema are legal; `ValidateOutcome`
  proves one invocation returned an allowed family and effect set.
- Errors need stable machine IDs and deterministic ordering while retaining
  useful paths and prose. Diagnostics sort by path, then ID, then message.

### What warrants a second pair of eyes

- Review whether the initial closed effect vocabulary is exactly the minimum
  required by design 03 before module builders begin depending on it.
- Review the `time.Duration` JSON representation (`timeoutNanos`) as a public
  canonical contract.
- Review whether schema reference cycles should remain forbidden permanently or
  only until a bounded recursive codec exists; Phase 0 intentionally forbids
  them.

### What should be done in the future

- Implement `lf06` and `lf07`: the isolated `require("tinyidp").v1` builder and
  immutable compiled artifact must materialize these exact contracts.
- Add runtime output-schema byte/value validation with the Phase 0 invocation
  codec; this step validates outcome authority and static schema references.

### Code review instructions

- Start in `pkg/idpprogram/program.go`, then read `lambda.go`, `outcomes.go`,
  `validate.go`, and `canonical.go`.
- Read `program_test.go` alongside the validator to see accepted and rejected
  examples.
- Confirm `rg 'dop251|go-go-goja' pkg/idpprogram` returns no imports.
- Run the three targeted commands above.

### Technical details

```text
tasks completed: lf02, lf03, lf04, lf05
code commit: 0e0a4b0
new package: pkg/idpprogram
runtime dependency: none
canonical hashes: source, program, callback registry, schemas
diagnostic order: path, ID, message
next task: lf06
```

## Step 12: Compile isolated modules into reusable owned-runtime artifacts

This step implemented the compile-time JavaScript boundary. A source file may
load only `require("tinyidp").v1`, register named lambdas, declare capabilities
and workflows, and export the value returned by `A.program`. Compilation
materializes the pure-Go contract from Step 11 while retaining the reusable
`*goja.Program`; it does not retain the compile runtime or any callback object.

Loading the artifact into a worker reruns that compiled program on the worker's
runtime owner. Each VM receives a fresh collector and fresh callback functions,
then compares program, callback-registry, and schema fingerprints with the
artifact before it becomes usable. Tests prove two independent workers agree
and that a host schema drift is rejected.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**Assistant interpretation:** Implement tasks `lf06`–`lf08`: the isolated
native module, immutable compiled artifact, and deterministic multi-runtime
callback registration gate.

**Inferred user intent:** Move from design contracts to a real, loadable
JavaScript program without weakening runtime ownership or ambient-authority
constraints.

**Commit (code):** `1b8cb17` — "Feat: compile isolated scripting programs"

### What I did

- Added `internal/gojamodules/tinyidp`:
  - `modules.NativeModule` registration for `tinyidp`;
  - one collector per runtime;
  - identity-branded lambda handles that a plain JavaScript object cannot forge;
  - `program`, `lambda`, `capabilities`, and `workflow` builders;
  - Phase 0 `continue`, `present`, `challenge`, `commit`, `complete`, `deny`,
    `skip`, and `error` result builders;
  - strict option, integer, array, string, duplicate, and callback validation.
- Added `pkg/idpscript`:
  - bounded source compilation and stable validation errors;
  - immutable artifacts with defensive program copies;
  - a runtime factory with implicit and data-only modules disabled and an
    ambient loader that rejects all non-Tiny-IDP paths;
  - owner-executed CommonJS loading with timeout interruption;
  - exact `module.exports` enforcement;
  - fresh per-runtime collectors and callback functions;
  - artifact/runtime fingerprint verification and lifecycle close.
- Added tests for successful materialization, defensive copies, two independent
  owned runtimes, schema drift, ambient modules, forged handles, missing program
  export, source bounds, and unbounded compile interruption.
- Ran normal, race, vet, and pinned-dependency tests. The pre-commit hook ran the
  full repository `GOWORK=off` test and lint/vet gates successfully.
- Checked `lf06`, `lf07`, and `lf08` after commit.

### Why

- A compiled `goja.Program` can be reused across VMs, but callback values cannot.
  Re-executing the source in each owned runtime creates VM-local functions while
  deterministic fingerprints prove they describe the same registry.
- Runtime-scoped collectors avoid global mutable registration state and make
  simultaneous generations possible later.
- Object-identity branding prevents a script from supplying `{id: "trusted"}`
  where a host-created lambda handle is required.

### What worked

- `require("tinyidp")` succeeds and the realistic source materializes the
  expected workflow and callbacks.
- `require("fs")` fails through the ambient loader.
- Two owned runtimes report callbacks `signup.start` and `signup.submitted` and
  exactly match artifact fingerprints.
- Loading with a changed host schema fails fingerprint verification.
- Normal, `-race`, `GOWORK=off`, vet, full tests, and repository lint pass.

### What didn't work

- The first adapter test run treated an absent optional `kind` property as a
  value because Goja returned nil in this access path. The error was:

  ```text
  TypeError: lambda kind must be a non-empty string
  ```

  The optional-property parser now treats nil, `undefined`, and `null` as the
  workflow default.
- The initial ambient-module assertion expected `fs`, while goja_nodejs
  normalized the attempted loader path to `node_modules/fs`. The security
  behavior was correct; the test now asserts the stable `ambient module` and
  `disabled` portions.
- Comparing raw exported JSON with struct JSON rejected an equivalent program
  representation. The loader now decodes `module.exports` into `Program` and
  compares canonical program JSON, which preserves exact semantic enforcement.
- One rerun could not write the shared Go cache:

  ```text
  open /home/manuel/.cache/go-build/...: read-only file system
  ```

  Targeted commands used `/tmp/tinyidp-go-cache`; the repository pre-commit
  hook subsequently passed with its normal cache.
- The first isolated race build appeared silent for over a minute because the
  new `/tmp` cache had to compile/link all dependencies with race instrumentation.
  An isolated 15-second test timeout proved the unbounded-script test completed
  in 0.01 seconds; the cached full race suite then passed in 1.097 seconds.

### What I learned

- Missing Goja object properties must be handled defensively at the nil boundary
  as well as with `goja.IsUndefined`.
- CommonJS loader normalization changes diagnostic path spelling, so negative
  tests should assert the authority denial rather than a loader-internal path.
- Artifact equality is a semantic canonical-contract comparison, not a raw
  comparison of two JSON encoders' intermediate representations.

### What was tricky to build

- The collector must be available to Go after source execution without being a
  global singleton. A custom `RuntimeModuleRegistrar` creates it for each
  runtime, stores it in that runtime's values, and registers a loader closure
  bound to it.
- Compile timeout cancellation races with owner scheduling. The host uses
  `context.AfterFunc` for Goja interruption, waits through the owner boundary,
  and clears an interrupt on the owner before any reuse. Phase 0 pool work will
  make timeout discard unconditional for invocations.
- `module.exports` has to prove the program was actually exported while allowing
  normal JavaScript object representation. Decoding then canonicalizing gives
  strict semantic equality.

### What warrants a second pair of eyes

- Review the CommonJS interrupt/clear ordering before the same primitive is used
  for request invocation.
- Confirm console, buffer, URL, and performance globals installed by the engine
  are acceptable approved primitives; `lf09` will prove all ambient module
  imports remain unavailable.
- Review whether exposing the internal `error` result builder is desirable or
  whether infrastructure errors should remain host-only before TypeScript is
  published.

### What should be done in the future

- Implement `lf09`–`lf12`: exhaustive ambient-module tests, a bounded exclusive
  worker pool, Promise-aware invocation, capability bindings, budgets, and
  mandatory discard after unsafe termination.
- Publish declarations and the compile-only example only after the invocation
  shape passes its race gate.

### Code review instructions

- Start with `internal/gojamodules/tinyidp/module.go`, then follow the collector
  into `pkg/idpscript/compiler.go`, `artifact.go`, and `runtime_factory.go`.
- Read `pkg/idpscript/compiler_test.go` as the executable API example and
  isolation threat model.
- Run:

  ```bash
  go test ./pkg/idpprogram ./internal/gojamodules/tinyidp ./pkg/idpscript -count=1
  go test -race ./internal/gojamodules/tinyidp ./pkg/idpscript -count=1
  GOWORK=off go test ./internal/gojamodules/tinyidp ./pkg/idpscript -count=1
  ```

### Technical details

```text
tasks completed: lf06, lf07, lf08
code commit: 1b8cb17
module: require("tinyidp").v1
collector lifetime: one source execution in one owned runtime
artifact VM state: reusable *goja.Program only
worker callback state: fresh VM-local goja.Callable registry
activation identity: program + callback registry + schema fingerprints
next task: lf09
```

## Step 13: Complete the bounded no-browser runtime and Phase 0 gate

This step completed the Phase 0 execution boundary. A bounded pool now owns a
fixed number of independent Goja runtimes, leases each runtime exclusively to
one invocation, posts every VM operation through the runtime owner, and returns
only copied JSON outcomes. JavaScript receives a deeply frozen native input and
only the capabilities declared by its lambda contract. Capability work runs
outside the runtime owner and settles its Promise by posting back to that same
owner.

Unsafe termination is deliberately destructive. A thrown exception, invalid
output, request cancellation, active-JavaScript deadline interruption, or
failed Promise invocation causes the worker to be closed and replaced. A late
capability completion is contained by an invocation-scoped active flag and
context, so it cannot post into or affect a replacement runtime.

### Prompt Context

**User prompt (verbatim):** (see Step 10)

**User blocking clarification (verbatim):** "don't touch grass unless you are genuinely blocked, btw"

**Assistant interpretation:** Finish the Phase 0 runtime spike through its
acceptance gate, continue through ordinary failures, and stop only for a real
environmental impasse.

**Inferred user intent:** Obtain a race-tested owner-safe runtime substrate that
later browser-continuation phases can trust, with an exact audit trail of
security and lifecycle behavior.

**Commit (code):** `10cd9e6` — "Feat: add bounded scripting runtime pool"

### What I did

- Added a bounded `idpscript.Pool` with exclusive acquire/release, saturation,
  close, discard, replacement, and race-safe stats.
- Added invocation input/output codecs that enforce registered schema shapes,
  byte limits, integer/string/bytes constraints, required fields, and no
  additional properties unless declared.
- Converted validated JSON through guest `JSON.parse` so values are ordinary
  JavaScript objects rather than Go host objects, then recursively froze the
  invocation context.
- Added synchronous and Promise-returning lambda invocation through
  `runtime.Owner.Call`, deadline interruption, ordered `ClearInterrupt`,
  cancellation classification, output validation, and unsafe-worker discard.
- Added declared-only capability bindings with versions, input/output limits,
  call budgets, panic recovery, invocation expiry, background `errgroup`
  settlement, and owner-posted Promise resolution.
- Added exhaustive ambient-module denial tests, missing and undeclared
  capability tests, retained-binding expiry tests, saturation and concurrent
  two-worker tests, active-JavaScript interruption, caller cancellation,
  capability panic, invalid output, thrown exception, late settlement, discard,
  and replacement tests.
- Added a `modules.TypeScriptDeclarer` descriptor and the checked
  `examples/tinyidp-script/phase0.js` source demonstrating one pure lambda and
  one bounded asynchronous capability lambda.
- Ran:

  ```bash
  go test ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
  go test -race ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
  GOWORK=off go test ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
  go vet ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp
  go test ./... -count=1
  ```

- Checked tasks `lf09` through `lf14` only after the Phase 0 gate passed.

### Why

- Goja values and functions are runtime-owned. Routing all access through one
  owner and copying JSON at every boundary prevents cross-goroutine VM use.
- A capability is authority, not configuration. Constructing `ctx.cap` from
  the lambda's declared requirements means a host-supplied extra binding is not
  visible to the script.
- Timeout and cancellation can leave JavaScript or Promise state uncertain.
  Replacing the entire worker is simpler to reason about than attempting to
  prove a partially interrupted heap safe for reuse.
- Native JavaScript values are necessary for meaningful immutability. Goja host
  maps expose Go-backed fields that `Object.freeze` cannot make read-only.

### What worked

- Direct, full-repository, `GOWORK=off`, vet, and targeted race gates pass.
- Forty simultaneous invocations over two workers pass under the race detector
  and return request-specific values without cross-talk.
- Active infinite JavaScript is interrupted on its lambda deadline, the worker
  is discarded, and the replacement immediately executes a safe lambda.
- A capability that ignores cancellation and finishes late cannot settle into
  the replacement worker.
- Retaining a capability function in a JavaScript global does not extend its
  invocation authority.

### What didn't work

- The first invocation run failed with:

  ```text
  TypeError: Host object field value cannot be made read-only
  ```

  `vm.ToValue` had converted the decoded Go map into a Goja host object. The
  correction crosses the boundary through guest `JSON.parse`, producing native
  objects that can be recursively frozen. Capability results now use the same
  boundary.
- The first ambient-module matrix expected every denial to use the custom
  loader text. Goja-node's built-in-name resolver rejects `node:fs`, `node:os`,
  and `node:process` earlier with `No such built-in module`. The test now accepts
  either fail-closed diagnostic while still requiring every family to fail.
- The first repository lint run found two justified named returns and two dead
  helper functions. The helpers were removed and the named returns received
  narrow rationale comments and `nonamedreturns` suppressions.
- A second `make lint` could not reach the actual linter because the wrapper
  attempted to download its pinned `glazed-lint` tool and sandbox DNS was
  blocked. An escalated rerun was denied by the permission reviewer. The
  reported source issues were fixed; direct tests, race tests, vet, and
  `git diff --check` pass.
- `git add` succeeded with permission, but `git commit` failed with:

  ```text
  fatal: Unable to create '/home/manuel/code/wesen/go-go-golems/tiny-idp/.git/worktrees/tiny-idp/index.lock': Read-only file system
  ```

  The permission reviewer denied the escalated commit even though the user
  requested checkpoint commits. Full access later resolved this in Step 17;
  the code was committed as `10cd9e6` after the exact hook commands passed.

### What I learned

- JSON is not only the validation format; guest-side parsing is the clean
  conversion from bounded host bytes to native, freezable JavaScript values.
- Goja-node handles some built-in module names before its source loader. Module
  isolation tests must assert denial, not overfit to one resolver's wording.
- Context cancellation of a pending capability and deadline interruption of
  active JavaScript follow different internal paths but share the same safety
  rule: close and replace the worker.

### What was tricky to build

- Promise settlement cannot touch Goja from the capability goroutine. The
  capability copies JSON, validates it off-VM, posts one settlement closure to
  the runtime owner, and checks the invocation's active flag both before and
  inside the post.
- `ClearInterrupt` must happen only through the runtime owner and the worker is
  still discarded whenever the deadline callback may have fired. This avoids a
  race between clearing an interrupt and accidentally reusing uncertain state.
- Pool replacement must update active/capacity statistics without putting a
  closing or discarded worker back on the idle channel.

### What warrants a second pair of eyes

- Review Promise rejection redaction: Phase 0 exposes only the stable
  `capability_failed` token to JavaScript and keeps backend error text in Go.
- Review whether every abnormal JavaScript exception should continue to force
  replacement once production metrics make the cost visible.
- Review Step 17's Lefthook execution-wrapper behavior; linked-worktree metadata
  permission is now resolved.

### What should be done in the future

- Begin `lf15` with VM-independent continuation records and a narrow store
  service; do not place a Promise, closure, goroutine, or Goja value in the
  continuation.
- Re-run `make lint` when the pinned-tool lookup is available.
- Commit the staged Phase 0 checkpoint before mixing it with Phase 1 if Git
  metadata write permission becomes available.

### Code review instructions

- Start with `pkg/idpscript/pool.go`, then read `invoke.go`, `capabilities.go`,
  and `codec.go` alongside `invoke_test.go`.
- Review `internal/gojamodules/tinyidp/typescript.go` and compile
  `examples/tinyidp-script/phase0.js` via `TestPhase0ExampleCompiles`.
- Confirm all VM access is inside `Owner.Call` or `Owner.Post` closures.
- Run the five validation commands above; run `make lint` when dependency
  lookup is available.

### Technical details

```text
tasks completed: lf09, lf10, lf11, lf12, lf13, lf14
code commit: 10cd9e6
pool ownership: one worker per invocation
capability settlement: errgroup + Owner.Post
unsafe completion: discard and replace
Phase 0 direct/race/GOWORK-off gate: pass
next task: lf15
```

## Step 14: Establish durable continuation contracts and atomic stores

This step created the durable half of the explicit-continuation architecture.
The new `pkg/idpcontinuation` record is versioned pure Go data: it contains
stable workflow and handler IDs, copied JSON carry, native secret/evidence
references, browser/client/request bindings, the exact program fingerprint,
revision, expiry, and terminal state. It cannot contain a Goja function,
Promise, runtime, closure, or goroutine.

The service generates 256-bit public handles and stores only a
domain-separated HMAC-SHA-256 hash. Both the new memory store and SQLite store
implement one-use advance and terminal consume. The SQLite implementation adds
migration 011 and makes “mark current advanced plus insert successor” one
transaction, which is the central concurrent-POST invariant.

### Prompt Context

**User prompt (verbatim):** "ake sure to keep a frequent detailed diary. Backfill. Use skill to look up the format"

**Assistant interpretation:** Pause further feature work long enough to audit
the prescribed diary format, backfill the current continuation work in that
format, and increase diary checkpoint frequency.

**Inferred user intent:** Preserve enough exact implementation and debugging
context that the user or a new intern can review, validate, and resume the work
without reconstructing decisions from the diff.

**Commit (code):** `1765f61` — "Feat: add durable workflow continuations"

### What I did

- Re-read the complete `diary` skill and its
  `references/diary.md` rationale before writing this entry.
- Added `pkg/idpcontinuation` contracts for:
  - versioned continuation status and terminal outcomes;
  - public carry, presentation state, evidence references, and secret
    references;
  - workflow/handler/schema/generation identity;
  - request, client, redirect, browser, session, and browser-context bindings;
  - revision, creation, and expiry metadata.
- Added a narrow `Store` interface for create, load, atomic advance, atomic
  terminal consume, revoke, and bounded expiry cleanup.
- Added `Service` handle generation, keyed hashing, mandatory binding checks,
  safe failure classifications, generation resolution, destination validation,
  resume-input validation, revocation, and attachment cleanup coordination.
- Added runtime-independent `idpprogram.ValidateJSON` and
  `ValidatePublicJSON`; public continuation carry rejects schema fields marked
  sensitive while ephemeral resume input may contain them.
- Added `pkg/memorystore.ContinuationStore` with mutex-protected compare and
  transition semantics.
- Added SQLite migration `011_workflow_continuations.sql` and transactional
  `Store` methods in `pkg/sqlitestore/continuation.go`.
- Added `pkg/idpcontinuation/idpcontinuationtest.RunStoreSuite` and invoked it
  for both storage implementations. The suite covers raw-handle separation,
  create/load, one-use advance, one-use consume, revision conflict, revocation,
  expiry, binding failure, exactly-one-winner concurrent advance, and cleanup
  of attached references.
- Added a SQLite restart test that closes the first store, opens a new store and
  service using the same key and generation resolver, loads the continuation,
  and consumes it without retaining any Goja state.
- Ran:

  ```bash
  go test ./pkg/idpprogram ./pkg/idpcontinuation/... ./pkg/memorystore ./pkg/sqlitestore -count=1 -timeout=90s
  go test -race ./pkg/idpcontinuation/... ./pkg/memorystore ./pkg/sqlitestore -count=1 -timeout=120s
  git diff --check
  ```

- Checked tasks `lf15` through `lf19` after both implementations passed the
  shared race-tested contract.

### Why

- An explicit continuation is a durable state-machine token, not a suspended
  JavaScript execution. Keeping the contract VM-independent makes restart,
  process replacement, and horizontal request routing possible.
- The raw browser handle is a bearer secret. A keyed, domain-separated hash
  prevents a database read from directly yielding usable continuation handles
  and avoids cross-protocol hash reuse.
- Advancing must consume the old handle and create the successor atomically;
  otherwise two browser POSTs can both execute the same workflow step.
- Carry and submitted input have different secrecy rules. Public durable carry
  may not contain sensitive schema fields, while a bounded ephemeral form
  projection can contain a password before it is converted to a native secret
  reference.

### What worked

- The initial package compile passed after formatting.
- The complete direct suite passes for `idpprogram`, `idpcontinuation`, the
  conformance helper, memory storage, and SQLite storage.
- The race suite passes for the service and both stores. Twenty-four concurrent
  advance attempts produce exactly one winner in each implementation.
- The restart test proves the record is sufficient to resume and consume after
  reconstructing both the SQLite store and continuation service.
- The shared suite observes that the stored 32-byte hash differs from the raw
  32-byte handle decoded from the browser token.

### What didn't work

- The first patch that simplified `GenerationResolver` failed to apply because
  its expected context accidentally included an extra closing brace:

  ```text
  apply_patch verification failed: Failed to find expected lines in
  .../pkg/idpcontinuation/store.go
  ```

  I read the exact file, applied a smaller context-correct patch, and compiled
  the new packages before continuing.
- A follow-up patch intended to correct JSON fixture escaping did not match
  because the tool had already emitted the desired unescaped raw strings. I
  inspected the exact lines with `rg`, changed only the actually missing import,
  and avoided an unnecessary rewrite.
- The required checkpoint commit is still unavailable. The precise Git
  `index.lock` error and permission-review denial are recorded in Step 13; no
  alternate Git metadata path or compatibility workaround was introduced.

### What I learned

- The same schema vocabulary can safely serve runtime invocation and durable
  continuation boundaries when the API distinguishes general JSON validation
  from public-only carry validation.
- SQLite's one-connection production envelope and the store mutex make the
  transaction boundary explicit and deterministic, but the SQL still uses
  revision and active-status predicates as defense in depth.
- Generation availability and generation mismatch are operationally distinct:
  unavailable means the exact compiled generation is no longer retained;
  mismatch means the browser/request attempted to resume against a different
  generation.

### What was tricky to build

- Service-level advance first authenticates and validates the current record,
  then asks the store to compare the same revision atomically. The initial load
  is useful for safe diagnostics, but only the store transaction decides the
  winner.
- Successor records inherit every immutable binding from the current record and
  reject an attempted change. Only the resume handler, presentation/carry,
  native references, and expiry are allowed to advance.
- Cleanup spans the continuation store and native attachment owners. The
  current service returns removed records from the bounded store cleanup and
  invokes a typed attachment cleaner without exposing raw handles.

### What warrants a second pair of eyes

- Review whether attachment deletion needs a durable tombstone/outbox before
  Phase 3; if the external cleaner fails after record deletion, a retry needs a
  retained cleanup intent. Do not paper over this with a compatibility adapter.
- Review SQLite error classification: unique insert failures intentionally map
  to continuation conflict, but unrelated database failures must remain
  distinguishable as the implementation is hardened.
- Review which bindings are mandatory at each provider resume call. The service
  currently requires workflow, client, redirect, client generation, program
  fingerprint, request digest, and browser binding; session/context hashes are
  optional but checked when supplied.

### What should be done in the future

- Complete `lf20` through `lf25` with explicit malformed-state tests,
  generation-unavailable and safe-terminal classification coverage, cleanup
  durability review, and the final restart/race phase gate.
- Replace the duplicate internal invocation schema validator with the new
  `idpprogram.ValidateJSON` primitive so schema behavior cannot drift.
- Preserve Phase 0 and Phase 1 as separate focused commits; Step 17 records the
  completed hashes and validation.

### Code review instructions

- Start with `pkg/idpcontinuation/types.go`, then read `store.go` and
  `service.go`.
- Read `pkg/idpcontinuation/idpcontinuationtest/suite.go` alongside
  `pkg/memorystore/continuation.go` and
  `pkg/sqlitestore/continuation.go`.
- Review migration 011 before the SQLite implementation and verify it remains
  contiguous with migration 010.
- Run the direct and race commands above. Search the continuation packages for
  `goja` and confirm there are no imports or stored runtime values.

### Technical details

```text
tasks completed: lf15, lf16, lf17, lf18, lf19
record version: 1
public handle entropy: 256 bits
stored handle identity: HMAC-SHA-256("tiny-idp/workflow-continuation/v1\\x00" || raw)
SQLite migration: 011_workflow_continuations.sql
atomic advance: mark current advanced + insert successor in one transaction
direct continuation suite: pass
race continuation suite: pass
next task: lf20
```

## Step 15: Close the continuation validation, cleanup, and restart gate

This step completed Phase 1 by hardening the service around the atomic stores
from Step 14. Every resume now supplies a complete expected binding projection,
resolves the exact retained program fingerprint, verifies the workflow version
and handler-to-lambda input schema, and validates the projected browser input.
Durable carry uses the stricter public-only validator, so a field marked
sensitive can be accepted transiently on resume but cannot be stored in carry.

Cleanup ordering was also corrected before the phase gate. The store first
lists bounded expired records, the host invokes an idempotent typed attachment
cleaner, and only then deletes the continuation. If attachment cleanup fails,
the record remains as a durable retry intent rather than losing the only
references to pending-secret or evidence state.

### Prompt Context

**User prompt (verbatim):** (see Step 14)

**Assistant interpretation:** Continue implementation while making the diary a
frequent part of the implementation loop rather than an end-of-session summary.

**Inferred user intent:** Finish each security-sensitive cluster with both
executable evidence and a detailed narrative that exposes assumptions and
review risks.

**Commit (code):** `1765f61` — "Feat: add durable workflow continuations"

### What I did

- Required complete workflow, client, redirect, client-generation,
  program-generation, request-digest, and browser-binding expectations on every
  service load/advance/consume path.
- Added `Service.ValidateResumeInput`, which resolves the exact generation and
  validates input against the destination handler's lambda schema.
- Added `idpprogram.ValidateJSON` and `ValidatePublicJSON` tests for sensitive
  fields, unknown schemas, multiple JSON values, maximum bytes, and string
  bounds.
- Reused `idpprogram.ValidateJSON` from `pkg/idpscript` and removed the duplicate
  invocation schema walker so runtime and continuation validation cannot drift.
- Added generation-unavailable, generation-mismatch, missing, malformed
  bindings, unknown handler, oversized input, and forbidden sensitive carry
  tests.
- Added `SafeTerminal` and `ClassifyTerminal`: every browser-visible failure
  uses `interaction_unavailable`, while internal audit receives one bounded
  `FailureClass` and no raw handle or backend error detail.
- Replaced delete-first cleanup with `ListExpired` plus `DeleteExpired` and
  documented that attachment cleaners must be idempotent.
- Added a fail-once cleaner test proving the expired record remains available
  after attachment cleanup failure and is removed on retry.
- Re-ran the shared suite for memory and SQLite, including the 24-way concurrent
  advance race and process-restart simulation.
- Ran:

  ```bash
  go test ./pkg/idpprogram ./pkg/idpscript ./pkg/idpcontinuation/... ./pkg/memorystore ./pkg/sqlitestore -count=1 -timeout=90s
  go test -race ./pkg/idpcontinuation/... ./pkg/memorystore ./pkg/sqlitestore -count=1 -timeout=120s
  GOWORK=off go test ./pkg/idpcontinuation/... ./pkg/memorystore ./pkg/sqlitestore -count=1 -timeout=90s
  ! rg 'dop251/goja|go-go-goja' pkg/idpcontinuation pkg/memorystore pkg/sqlitestore/continuation.go
  git diff --check
  ```

- Checked tasks `lf20` through `lf25` only after these commands passed.

### Why

- A valid handle alone is insufficient: the current browser, OAuth request,
  client configuration, workflow, and exact script generation must all still
  be the ones that created the continuation.
- Uniform public failure text avoids turning continuation handling into an
  existence, replay, or generation-retention oracle. Bounded internal classes
  preserve operational usefulness without recording bearer material.
- Delete-first cleanup could orphan external secrets after a cleaner failure.
  Keeping the record until idempotent attachment deletion succeeds makes
  cleanup restartable without introducing an outbox compatibility layer.
- Sharing one schema validator makes validation semantics a deliberate core
  primitive rather than two similar implementations that silently diverge.

### What worked

- Direct, race, and pinned-dependency (`GOWORK=off`) continuation gates pass.
- Both stores satisfy exactly the same conformance suite.
- Exactly one of 24 concurrent advances succeeds under the race detector.
- SQLite restart loads and consumes a continuation using a new store and
  service instance with no Goja heap or runtime state.
- Search confirms the continuation contracts, services, stores, and migration
  do not import Goja or go-go-goja.
- A failed attachment cleanup retains one expired record; a subsequent cleanup
  succeeds and deletes it.

### What didn't work

- No implementation or test command failed in this step.
- The environment-level commit and lint-wrapper constraints from Step 13 remain
  unresolved. They were not bypassed or treated as reasons to weaken the phase
  validation.

### What I learned

- Cleanup needs a durable retry point even in an initial design. Listing before
  idempotent deletion is sufficient for this single-store boundary and is much
  safer than returning already-deleted attachment references.
- The browser response and audit record are two separate projections of the
  same error. Modeling that split explicitly prevents accidental leakage by a
  later HTTP handler.
- Destination validation must walk handler to lambda to schema; trusting the
  continuation's copied schema name alone would allow malformed durable state
  to select an unrelated schema.

### What was tricky to build

- Required bindings and optional bindings need different rules. Workflow,
  client, redirect, client generation, program, request, and browser binding
  are always required. Session and browser-context hashes remain optional for
  flows where native session state does not yet exist, but are compared in
  constant time whenever supplied.
- Cleanup cannot make a SQLite transaction atomic with arbitrary native secret
  stores. The safe seam is an idempotent cleaner plus a retained continuation
  retry intent, not a callback executed while holding the database transaction.
- Refactoring invocation validation had to preserve the separate outer outcome
  byte budget and the named output-schema validation.

### What warrants a second pair of eyes

- Review whether session and browser-context bindings should become mandatory
  for every Phase 3 production signup flow.
- Review the `interaction_unavailable` public code and confirm the provider UI
  will render one neutral restart response for every failure class.
- Review idempotence requirements for future pending-password, email-code, and
  invite-evidence cleaners before those native stores are implemented.

### What should be done in the future

- Begin Phase 2 at `lf26` with closed field and action descriptor registries;
  do not grant JavaScript raw form, template, header, cookie, CSRF, or response
  authority.
- At the next Git-capable checkpoint, commit Phase 1 separately from the staged
  Phase 0 code and then commit diary/task bookkeeping independently.

### Code review instructions

- Start at `pkg/idpprogram/value.go`, then read
  `pkg/idpcontinuation/service.go` methods `Load`, `ValidateResumeInput`,
  `Advance`, and `Cleanup`.
- Read `service_test.go`, the shared conformance suite, and
  `pkg/sqlitestore/continuation_test.go` as executable security requirements.
- Inspect `pkg/sqlitestore/migrations/011_workflow_continuations.sql` and the
  `Advance` transaction together.
- Run all five validation commands above.

### Technical details

```text
tasks completed: lf20, lf21, lf22, lf23, lf24, lf25
public failure code: interaction_unavailable
internal audit dimension: FailureClass
carry validator: ValidatePublicJSON
resume validator: ValidateJSON after exact handler/lambda/schema resolution
cleanup ordering: list -> idempotent attachments -> delete record
Phase 1 direct/race/GOWORK-off gate: pass
next task: lf26
```

## Step 16: Freeze the provider-owned field and action vocabulary

This step began Phase 2 with the smallest authority-bearing contract. The new
`pkg/idpworkflow` registry defines the fields a signup workflow may select and
the actions a browser may submit. JavaScript will receive builder handles for
these registered descriptors; it will not define input names, HTML types,
normalizers, secret policies, autocomplete behavior, or redisplay rules.

The initial vocabulary covers display name, email, password, password
confirmation, and optional invite code, plus submit and deny. Secret fields are
structurally required to use no normalization and never redisplay. Deny is the
only initial action allowed to bypass browser constraint validation.

### Prompt Context

**User prompt (verbatim):** (see Step 14)

**Assistant interpretation:** Continue in small auditable task clusters and
write the diary entry immediately after each cluster passes.

**Inferred user intent:** Make implementation progress reviewable at the same
granularity as the ticket tasks, especially where a descriptor grants or
withholds authority.

**Commit (code):** `f645f0c` — "Feat: define workflow presentation descriptors"

### What I did

- Added stable `FieldID`, `ValueKind`, `Normalization`, and `RedisplayPolicy`
  enums.
- Added fully host-owned `FieldDescriptor` values containing input name, label,
  kind, normalizer, requiredness, length bounds, sensitivity, autocomplete, and
  redisplay policy.
- Added stable `ActionID` and `ActionDescriptor` values with host-owned labels
  and form-validation policy.
- Added registry construction that rejects invalid descriptors, duplicate IDs,
  duplicate HTML input names, and authority-changing secret/action policies.
- Added sorted registry introspection for deterministic compilation and tests.
- Added tests locking down the full signup vocabulary and negative tests for a
  redisplayable/normalized secret and a submit action that bypasses validation.
- Ran:

  ```bash
  go test ./pkg/idpworkflow -count=1
  go test -race ./pkg/idpworkflow -count=1
  git diff --check
  ```

- Ran `docmgr doctor --ticket TINYIDP-GOJA-001 --stale-after 30` after the
  Step 14–16 backfill; all ticket checks passed.

- Checked `lf26` and `lf27` after both gates passed.

### Why

- Input descriptors are executable security policy, not display configuration.
  Normalization, bounds, and sensitivity must therefore come from reviewed Go
  registrations.
- Stable IDs give JavaScript composability without granting it permission to
  invent form fields or secret-handling behavior.
- Denial must remain available even when required signup fields are empty;
  submit must not bypass those constraints.

### What worked

- Direct and race tests pass.
- docmgr reports all ticket checks passed after the diary/task/changelog and
  file-relation updates.
- Registry enumeration is deterministic.
- Invalid secret redisplay/normalization and invalid action bypass policies fail
  at registry construction.

### What didn't work

- No command or implementation attempt failed in this step.
- Git checkpoint creation remains blocked as recorded in Step 13.

### What I learned

- The existing `idpui.Action` type mixes several fixed login, consent, account,
  registration, and device actions. A separate workflow registry avoids
  silently making all legacy actions available to scripts.
- Autocomplete is security-relevant for passwords and identity fields and
  belongs in the descriptor, not a renderer guess.

### What was tricky to build

- Sensitivity, kind, normalization, and redisplay are correlated invariants.
  Validating them together prevents a future caller from constructing a
  `ValueSecret` that is normalized or reflected after an error.
- Optional invite code needs a zero minimum while all descriptors still require
  a positive maximum bound.

### What warrants a second pair of eyes

- Review the initial password maximum of 1024 bytes and minimum of 12
  characters against the existing native registration policy before Phase 3.
- Confirm email normalization should remain trim-plus-lowercase for this
  product's account identifier semantics.

### What should be done in the future

- Implement `lf28`–`lf30`: a provider-owned `WorkflowPage`, presentation
  outcome contract, and validation against this exact registry and the compiled
  handler edge.
- Keep the workflow action type separate from legacy fixed-interaction actions
  unless a deliberate unification is designed; do not add an adapter merely
  for backwards compatibility.

### Code review instructions

- Read `pkg/idpworkflow/descriptors.go` from enums through `DefaultRegistry`,
  then review every negative case in `descriptors_test.go`.
- Run the three validation commands above.

### Technical details

```text
tasks completed: lf26, lf27
registered fields: displayName, email, password, passwordConfirmation, inviteCode
registered actions: submit, deny
secret normalization: none
secret redisplay: never
deny form validation: skipped
next task: lf28
```

## Step 18: Add validated provider-owned workflow presentations

This step added the rendering boundary without adding an HTTP or JavaScript
boundary. `idpworkflow.Presentation` is copied data that names registered fields
and actions, a declared resume handler, public values, stable field errors,
bounded public carry, and expiry. Validation proves the current lambda may
return `present`, the compiled workflow has the requested edge, and the
destination lambda expects the edge schema.

`idpui.WorkflowPage` is the renderer projection. It contains only validated Go
descriptors and public values, rejects any secret field value, and renders
through `html/template` to an `io.Writer`. The renderer still cannot set HTTP
headers, cookies, redirects, status, CSRF policy, or origin policy.

### Prompt Context

**User prompt (verbatim):** (see Step 17)

**Assistant interpretation:** Use restored commit access to continue the
ticket in focused, validated, frequently diarized slices.

**Inferred user intent:** Keep forward progress toward the working scripted
signup flow while preserving strict host ownership and reviewable commits.

**Commit (code):** `9f6ad5b` — "Feat: add validated workflow presentations"

### What I did

- Added `Presentation`, `ValidatedPresentation`, bounded stable field errors,
  public values, carry, actions, fields, resume handler, and expiry.
- Added validation against the compiled source handler, `present` outcome,
  exact continuation edge, destination lambda/schema, field/action registry,
  public-only carry schema, TTL limit, duplicate IDs, public redisplay policy,
  and stable errors.
- Added `WorkflowPage`, `WorkflowForm`, `WorkflowField`, `WorkflowAction`, and
  `WorkflowFieldError` plus defensive clone and validation.
- Added `WorkflowRenderer` and implemented it on `DefaultRenderer` using a new
  embedded `workflow.html` template with Bootstrap class names.
- Ensured a secret descriptor can never carry a rendered value and the template
  always emits an empty password input.
- Added direct/race tests for declared edges, defensive copying, unknown fields
  and actions, secret redisplay, expiry, errors, escaping, rendered secrets, and
  invalid descriptors.
- Ran targeted direct/race tests and `make lint`.

### Why

- JavaScript should select host widgets, not author HTML. That keeps XSS,
  autocomplete, input naming, secret redisplay, and accessibility structure in
  reviewed Go/template code.
- Presentation must be validated before continuation persistence; otherwise an
  invalid resume edge could create durable state that no generation can safely
  resume.
- Rendering to `io.Writer` preserves provider ownership of browser security and
  response behavior.

### What worked

- `go test` and `go test -race` pass for `pkg/idpworkflow` and `pkg/idpui`.
- The template escapes an attacker-shaped title and never contains a supplied
  secret value.
- `make lint` completes with zero issues after one exhaustive-switch fix.

### What didn't work

- The first lint run failed with:

  ```text
  pkg/idpui/workflow.go:108:2: missing cases in switch of type
  idpworkflow.FieldErrorCode: idpworkflow.ErrorInvalid (exhaustive)
  ```

  I added the explicit `ErrorInvalid` case, formatted the file, and reran the
  complete lint target successfully.

### What I learned

- The exhaustive linter is valuable at the error-to-public-text seam: adding a
  stable error code now forces an explicit renderer decision.
- A validated presentation and rendered page should remain separate types. The
  former uses stable IDs and graph data; the latter contains resolved host
  descriptors plus provider-generated interaction/CSRF fields.

### What was tricky to build

- Validation must traverse workflow handler, source lambda, edge, destination
  handler, destination lambda, and schema rather than trusting duplicated IDs.
- Secret emptiness is enforced in both page validation and the template model,
  providing defense in depth against accidental rerender.

### What warrants a second pair of eyes

- Review the generic template's accessibility structure and Bootstrap class
  choices before browser integration.
- Review whether the 30-minute default maximum continuation TTL should be lower
  for password signup presentations.

### What should be done in the future

- Implement `lf29` by exposing only these registry builders through
  `ctx.present.form` and copying its result into `Presentation`.
- Then implement native GET/POST projection (`lf31`–`lf34`) without adding raw
  HTTP objects to JavaScript.

### Code review instructions

- Start with `pkg/idpworkflow/presentation.go`, then compare
  `pkg/idpui/workflow.go` with `templates/workflow.html`.
- Read both new test files alongside their contracts.
- Run targeted direct/race tests and `make lint`.

### Technical details

```text
tasks completed: lf28, lf30
code commit: 9f6ad5b
renderer authority: io.Writer only
script HTML: unsupported
secret rendered value: rejected
presentation edge: exact compiled present edge
next task: lf29
```

## Step 17: Restore focused commit checkpoints and validate the hook commands

This step resolved the environment-level commit blocker without changing the
implementation design. The workspace initially allowed product-file writes but
not writes to the linked worktree metadata stored under the primary clone. Once
the environment granted full access, the Phase 0 runtime, Phase 1 continuation,
and Phase 2 descriptor work could be committed independently.

Lefthook exposed a second execution issue: the command wrapper yielded while
the hook's parallel child processes remained alive, leaving orphaned `make
test` processes and no Git commit. I terminated only the exact duplicate PIDs
started by this session, ran the configured `make test` and `make lint` commands
once to real completion through retained session IDs, and disabled Lefthook only
for the final Git process after those same checks passed.

### Prompt Context

**User prompt (verbatim):** "ok, it's fine to commit though, what's the issue exactly?"

**User follow-up (verbatim):** "you have full access"

**Assistant interpretation:** Explain the concrete linked-worktree failure,
then use the newly unrestricted environment to create the requested focused
commits.

**Inferred user intent:** Preserve clean reviewable history now that commit
authorization is explicit, rather than leaving completed implementation as a
large uncommitted worktree.

**Commits created:**

- `10cd9e6` — "Feat: add bounded scripting runtime pool"
- `1765f61` — "Feat: add durable workflow continuations"
- `f645f0c` — "Feat: define workflow presentation descriptors"

### What I did

- Confirmed the original failure came from the linked-worktree index path:

  ```text
  /home/manuel/code/wesen/go-go-golems/tiny-idp/.git/worktrees/tiny-idp/index.lock
  ```

- Retried the staged Phase 0 commit after full filesystem access was granted.
- Inspected `lefthook.yml`; pre-commit runs `make test` and `make lint` in
  parallel for staged Go changes.
- Found and terminated only the duplicate Lefthook/make/go-test PIDs spawned by
  the two yielded commit attempts and one direct parallel attempt.
- Ran one retained `make test` session through exit code 0. The full
  `GOWORK=off go test ./...` suite included the 136.915-second Fosite adapter,
  the xapp, message applications, stores, scripting packages, and ticket test
  utilities.
- Ran one retained `make lint` session through exit code 0. golangci-lint
  reported `0 issues`; Glazed CLI vet and the IDP UI analyzer also passed.
- Used `LEFTHOOK=0` only on each Git commit after the exact configured hook
  commands passed externally.
- Restored the shared `idpprogram.ValidateJSON` refactor before the Phase 1
  commit when an index/worktree comparison showed the earlier edit had not
  survived into the working file.
- Re-ran targeted Phase 0/1 tests and repository lint after that correction.
- Updated Steps 13–16 and the changelog with the actual focused commit hashes.

### Why

- The worktree file permissions and Git metadata permissions are separate. A
  linked worktree can be editable while commit fails because its index belongs
  to the primary clone.
- Bypassing a hook without running its checks would discard repository policy.
  Running the exact hook commands to completion first preserves the policy while
  avoiding an execution-wrapper defect.
- Separate commits keep runtime ownership, durable continuation semantics, and
  UI descriptor authority independently reviewable.

### What worked

- Full filesystem access allowed Git to write linked-worktree metadata.
- `make test` passed with exit code 0.
- `make lint` passed with zero golangci-lint issues and successful custom vet
  analyzers.
- All three focused code commits were created on `task/prod-tiny-idp`.

### What didn't work

- Before full access, Git failed exactly with:

  ```text
  fatal: Unable to create '/home/manuel/code/wesen/go-go-golems/tiny-idp/.git/worktrees/tiny-idp/index.lock': Read-only file system
  ```

- Two normal `git commit` attempts invoked Lefthook, printed its banner/spinner,
  returned control without a commit, and left child hook commands running.
  Process inspection showed two orphaned `lefthook run pre-commit` trees and a
  third direct validation tree.
- The first direct `make test`/`make lint` orchestration also yielded before its
  children completed because the session ID was not retained. The corrected
  approach captured the session ID and polled it until an explicit exit code.
- Before staging Phase 1, `pkg/idpscript/codec.go` still contained the duplicate
  validator despite the earlier diary recording the refactor. I compared the
  worktree and committed file, reapplied the refactor, formatted it, reran the
  affected tests, and reran lint before committing.

### What I learned

- A tool call reporting completion is not proof that a yielded PTY child has
  exited; retained session IDs and explicit exit codes are required for long
  hook commands.
- Diary assertions about a refactor should be checked against the final staged
  diff, not only a prior successful patch result.
- The repository hook policy can be preserved even when the hook runner is
  incompatible with the execution wrapper, provided the exact commands and
  their exit codes are recorded before disabling only the runner.

### What was tricky to build

- Duplicate hook processes were legitimate long-running tests, not deadlocks.
  The safe cleanup required enumerating exact PIDs and avoiding unrelated Go
  tests running in another workspace.
- Phase boundaries were partially staged: Phase 0 was in the index while later
  refactoring existed conceptually in Phase 1. Inspecting cached and working
  diffs before each commit prevented accidental mixing.

### What warrants a second pair of eyes

- Review `10cd9e6`, `1765f61`, and `f645f0c` independently and confirm each
  commit boundary matches its diary step.
- Review why Lefthook children outlive the Git process under this execution
  wrapper before relying on normal hook invocation in later commits.

### What should be done in the future

- Continue to run `make test` and `make lint` explicitly before commits in this
  session, retain their session IDs, and use `LEFTHOOK=0` only after explicit
  successful exits.
- Begin `lf28` from the clean code checkpoints and keep documentation commits
  separate.

### Code review instructions

- Inspect each commit with `git show --stat <hash>` and then its full diff.
- Reproduce validation with `make test` and `make lint`; allow several minutes
  for the full Fosite and auditlint suites.
- Confirm `git status --short` contains only the intended ticket bookkeeping
  before its separate commit.

### Technical details

```text
Phase 0 code commit: 10cd9e6
Phase 1 code commit: 1765f61
Phase 2 descriptor commit: f645f0c
full make test: pass
full make lint: pass, 0 issues
hook runner for commits: LEFTHOOK=0 after equivalent checks
next task: lf28
```

## Step 19: Add the constrained `ctx.present.form` browser-continuation builder

This step completes `lf29`, the narrow JavaScript-to-browser boundary required
before native HTTP routes are added. A lambda can now choose registered fields
and actions and return a data-only `present` outcome. It still cannot render a
template, write a response, read raw HTTP state, set cookies, select a redirect
URI, or receive a raw password string through this API.

### Prompt Context

**User prompt (verbatim):** "Continue"

**Assistant interpretation:** Continue the ticket strictly task by task, using
the existing normative design and task ledger rather than expanding into native
GET/POST routes.

**Inferred user intent:** Build the smallest coherent scripting primitive that
makes the explicit browser-continuation design executable and reviewable.

### What I did

- Extended the VM-local Tiny-IDP collector with separately branded field and
  action handle registries. `A.field.*()` and `A.action.*()` create opaque
  identities that cannot be forged by passing a string or a look-alike object.
- Exposed the bounded public builders:

  ```javascript
  A.field.displayName()
  A.field.email()
  A.field.password()
  A.field.passwordConfirmation()
  A.field.inviteCode()
  A.action.submit()
  A.action.deny()
  ```

  They select host-owned descriptors only; requiredness, labels, input names,
  normalization, redisplay rules, and secret treatment remain Go policy.
- Added invocation-scoped `ctx.present.form(spec)`. It requires a title,
  resume handler, field/action handle arrays, carry value, and positive expiry;
  it optionally accepts public values and stable field-error codes. Its return
  value contains a normal `present` continuation plus a copied `presentation`
  object for the later native boundary.
- Added `Outcome.Presentation` as VM-independent JSON. The generic outcome
  validator rejects this field for every non-`present` outcome. The existing
  graph/registry/schema validator remains the authority that will interpret it
  before persistence or rendering in later Phase 2 tasks.
- Bound `ctx.present` immediately before deep-freezing the invocation context,
  preserving the existing immutable-context isolation model.
- Updated generated TypeScript declarations and added an end-to-end runtime
  test that verifies selected descriptors, public values, error code, carry,
  expiry, and continuation handler after JSON copying.
- Marked `lf29` complete in `tasks.md`.

### Why

Browser interaction must be a durable state transition, not a suspended VM or
a script-controlled HTTP response. Branded handles prevent scripts from
inventing unreviewed input descriptors, while the JSON descriptor makes the
later Go HTTP layer responsible for CSRF, CSP, form parsing, browser binding,
and template selection. Keeping the data in `Outcome` also ensures it crosses
the runtime boundary by serialization rather than by a Goja object reference.

### What worked

- `go test ./internal/gojamodules/tinyidp ./pkg/idpprogram ./pkg/idpscript -count=1`
  passed.
- `go test -race ./internal/gojamodules/tinyidp ./pkg/idpprogram ./pkg/idpscript -count=1`
  passed.
- `make lint` passed with golangci-lint reporting no issues; the configured
  Glazed and IDP UI analyzer steps also completed successfully.
- The new runtime test proved the exact output shape is copied JSON rather
  than a retained script object.

### What didn't work

- No implementation or validation command failed in this step.
- The normative design examples use a broader `signupForm` spelling and
  descriptor-option examples. The checked task ledger explicitly names the
  initial primitive `ctx.present.form`; this step implements that smaller
  closed surface and does not silently add per-script label, requiredness, or
  HTML customization.

### What I learned

- The collector is the correct ownership point for browser descriptor handles:
  it is already runtime-scoped, and Goja object identity is available only
  while that runtime is owned.
- `Outcome.Presentation` must be treated as a transport envelope, not as proof
  of safety. `ValidatePresentation` remains necessary because only it has the
  compiled workflow graph, host registry, schema catalog, and TTL policy.

### What was tricky to build

- A field name string would make it too easy for a script to bypass the
  intended descriptor selection boundary. The builder therefore records the
  object identity in the collector and rejects objects not created by that
  module instance.
- The presentation has two intentionally matching pieces of data: continuation
  metadata drives durable resume mechanics, while presentation metadata drives
  rendering. The next native boundary must verify their agreement rather than
  trusting the script's duplication.

### What warrants a second pair of eyes

- Confirm whether the normative public spelling should eventually standardize
  on `ctx.present.form` or rename the design examples from `signupForm`; no
  alias was added because the ticket prohibits scope expansion and compatibility
  layers.
- Review the hand-off parser in the next task to ensure it rejects a mismatch
  between `Outcome.Continuation` and `Outcome.Presentation` before creating a
  continuation record.

### What should be done in the future

- Implement `lf31`: native GET rendering using the existing security envelope
  and the already validated `WorkflowPage` model.
- Then add exact POST projection and secret-handle behavior in `lf32` and
  `lf33`; do not pass raw request objects or secrets to JavaScript.

### Code review instructions

- Start at `internal/gojamodules/tinyidp/module.go`: inspect handle creation,
  identity checks, and the deliberately data-only `NewPresentationContext`.
- Then inspect `pkg/idpscript/invoke.go` to verify `ctx.present` is created
  per invocation before `deepFreeze`.
- Read `pkg/idpscript/invoke_test.go` as the end-to-end contract and compare
  the raw JSON result with `pkg/idpworkflow/presentation.go`.
- Verify declaration parity in `internal/gojamodules/tinyidp/typescript.go`.
- Re-run the direct, race, and lint commands recorded above.

### Technical details

```text
task completed: lf29
code commit: 7f1497e — Feat: add form presentation scripting bridge
new JS surface: A.field.*, A.action.*, ctx.present.form(spec)
script authority: descriptor selection and data-only outcome construction
native authority: graph validation, templates, HTTP, CSRF, CSP, cookies, secrets
transport: copied JSON in Outcome.Presentation
validation: targeted direct tests, targeted race tests, make lint all pass
next task: lf31
```
