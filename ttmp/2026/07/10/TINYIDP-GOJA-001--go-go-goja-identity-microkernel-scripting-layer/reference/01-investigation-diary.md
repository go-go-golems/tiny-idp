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
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/01-go-go-goja-scripting-layer-analysis-design-and-implementation-guide.md
      Note: Primary design produced by the investigation
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/design-doc/03-lambda-first-tiny-idp-javascript-api-with-explicit-browser-continuations.md
      Note: Normative design produced and recorded in Step 7
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/reference/02-security-verification-scripting-plane-assessment.md
      Note: Step 4 focused verification-plane design
    - Path: repo://ttmp/2026/07/10/TINYIDP-GOJA-001--go-go-goja-identity-microkernel-scripting-layer/sources/01-colleague-identity-microkernel-research.md
      Note: Verbatim source moved from /tmp and analyzed in Steps 1 and 2
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
