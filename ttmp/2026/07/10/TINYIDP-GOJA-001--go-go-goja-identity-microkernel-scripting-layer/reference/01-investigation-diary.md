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
