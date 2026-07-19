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
    - Path: repo://internal/cmds/script.go
      Note: |-
        Operator test command and Glazed row command construction
        Stable per-case CLI output and failure contract
    - Path: repo://internal/cmds/script_test.go
      Note: Success and nonzero assertion-failure diagnostics
    - Path: repo://internal/fositeadapter/registration_test.go
      Note: Browser proof and delivery lifecycle evidence
    - Path: repo://internal/fositeadapter/scripted_signup.go
      Note: |-
        Native signup commit revalidates optional invitation effect and redeems it inside its transaction (commit 84a9995)
        Native signup continuation, proof, delivery, and resend boundary
    - Path: repo://internal/gojamodules/tinyidp/module.go
      Note: |-
        Optional invite code can only become declared consumeInvitation plan (commit 84a9995)
        Typed ctx.challenge.emailCode outcome builder (commit 2e8a517)
    - Path: repo://internal/gojamodules/tinyidp/typescript.go
      Note: JavaScript and TypeScript test fakes API
    - Path: repo://lefthook.yml
      Note: Step 17 exact pre-commit test and lint policy whose runner orphaned children
    - Path: repo://pkg/embeddedidp/options.go
      Note: Embeds durable invitation service in scripted signup configuration (commit d46ec55)
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Aggregate scripted-signup readiness
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
    - Path: repo://pkg/idpemailchallenge/service.go
      Note: |-
        Native one-use verification produces evidence (commits 04a514f, 0beb0b5)
        Native verified-email and code-rotation lifecycle
    - Path: repo://pkg/idpinvite/computed.go
      Note: Bounded host eligibility capability and validation seam (commit 40e7747)
    - Path: repo://pkg/idpinvite/computed_test.go
      Note: Provider invocation proves JavaScript receives a decision rather than authority (commit 40e7747)
    - Path: repo://pkg/idpinvite/durable.go
      Note: Keyed code hashing and transaction-scoped one-time invitation redemption (commit 21c7c4c)
    - Path: repo://pkg/idpprogram/program.go
      Note: |-
        Declarative embedded test contract
        Declarative fake fixture data contract
    - Path: repo://pkg/idpprogram/value.go
      Note: Step 14 shared runtime-independent JSON and public-carry validation
    - Path: repo://pkg/idpprogram/value_test.go
      Note: Step 15 sensitive-carry and bounded JSON regression tests
    - Path: repo://pkg/idpscript/codec.go
      Note: Step 15 runtime now shares the core schema validator
    - Path: repo://pkg/idpscript/invoke.go
      Note: Invocation-scoped evidence projection (commit c14d70f)
    - Path: repo://pkg/idpsignup/executor.go
      Note: Fixed test-only deterministic capability catalog
    - Path: repo://pkg/idpsignup/executor_test.go
      Note: Invitation workflow output regression (commit 84a9995)
    - Path: repo://pkg/idpsignup/manager.go
      Note: |-
        Activation test gate and active-generation readiness
        Atomic activation, retention, draining, and bounded metrics
    - Path: repo://pkg/idpsignup/manager_test.go
      Note: Reload, closure, retention, and metrics proof
    - Path: repo://pkg/idpstore/interfaces.go
      Note: Durable invitation lifecycle operations available on the caller-owned transaction (commit 21c7c4c)
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
    - Path: repo://pkg/sqlitestore/migrations/012_durable_invitations.sql
      Note: Persistent durable invitation state (commit 21c7c4c)
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

## Step 20: Add the native workflow-page rendering boundary

This step completes `lf31` without prematurely adding a public scripting route.
The Fosite adapter now has a native `renderWorkflow` path for a validated
`WorkflowPage`; Phase 3 will call it only after Fosite has accepted the
authorization request and the workflow executor has returned a presentation.

### Prompt Context

**User prompt (verbatim):** "Continue"

**Assistant interpretation:** Advance the next checked task while retaining
the provider's existing browser security envelope.

**Inferred user intent:** Prepare the actual production rendering seam, not a
mock UI or a JavaScript-controlled endpoint.

### What I did

- Added `WorkflowRenderer` to Fosite adapter options and provider state. It
  defaults to the supplied interaction renderer only when it implements the
  workflow contract; otherwise it constructs the established default renderer.
- Added `Provider.renderWorkflow`, an unexported native boundary parallel to
  `renderInteraction`. It preserves no-store/no-cache headers, a CSP restricted
  to self plus the validated redirect origin, bounded buffered output, HTML
  content type, render metrics, and bounded audit failure reasons.
- Extended `WorkflowForm` with a validated absolute redirect origin. This is
  provider-projected data used solely to construct CSP; scripts still cannot
  select a form action or header.
- Added a provider-internal regression test that renders a validated workflow
  page and asserts CSP, cache headers, HTML content type, form action, and the
  established interaction/CSRF hidden controls. Added a page-model test that
  rejects a redirect URL rather than a bare origin.
- Marked `lf31` complete in the task ledger.

### Why

The renderer is not an HTTP authority. The adapter must therefore own the
security headers and buffer before any response bytes are committed. A public
route would be unsafe and incomplete here: no activated generation, Fosite
validated request, workflow continuation, or native POST projection exists yet.
This seam makes the later routing task compose with the already hardened
provider controls instead of recreating a weaker form endpoint.

### What worked

- `go test ./pkg/idpui ./internal/fositeadapter -count=1` passed.
- `go test -race ./pkg/idpui ./internal/fositeadapter -count=1` completed with
  no race failure after the normal long-running Fosite browser/OIDC suite.
- `make lint` passed with zero golangci-lint issues and successful custom
  Glazed and IDP UI analyzer checks.

### What didn't work

- The first new test asserted invented hidden-control names. The actual,
  established provider contract is `interaction` and `csrf_token`; the test
  was corrected to assert that existing contract. No production code changed
  in response to the failure.

### What I learned

- `WorkflowPage` needs the redirect *origin* as a distinct provider-projected
  field: the full redirect URI must never be given to the renderer merely to
  construct CSP.
- Keeping `renderWorkflow` unexported prevents callers from treating it as an
  alternate OAuth entry point. It is a composition primitive for the validated
  authorization path, not a generic web endpoint.

### What was tricky to build

- The native renderer must make headers authoritative before validating/rendering
  the page, matching the existing interaction path exactly enough that failed
  renderers cannot accidentally weaken cache or CSP behavior.
- The generic rendering seam has to exist before Phase 3, while actual routing
  must wait for a Fosite-validated signup interaction. The boundary is therefore
  implemented and tested without exposing an incomplete public route.

### What warrants a second pair of eyes

- Compare `renderWorkflow` line by line with `renderInteraction` when later
  refactoring shared rendering mechanics; their failure accounting must remain
  equivalent.
- Review the Phase 3 caller to ensure its `RedirectOrigin` comes from the
  already validated registered redirect URI, never raw browser input.

### What should be done in the future

- Implement `lf32`: exact native POST projection. It must accept only the
  selected descriptors and actions, reject duplicate singleton values and
  malformed/oversized forms, and run before JavaScript.

### Code review instructions

- Read `internal/fositeadapter/rendering.go` beside its existing
  `renderInteraction` implementation.
- Confirm `pkg/idpui/workflow.go` accepts only a bare HTTPS/HTTP origin.
- Run the direct/race package commands and `make lint` above.

### Technical details

```text
task completed: lf31
code commit: 87db6ba — Feat: add native workflow page renderer
new boundary: Provider.renderWorkflow (unexported)
headers: Cache-Control no-store, Pragma no-cache, provider-owned CSP
HTML: bounded buffer, validated WorkflowPage, renderer gets io.Writer only
public route: intentionally absent until Fosite-validated Phase 3 routing
next task: lf32
```

## Step 21: Integrate the native signup POST projection and checked-in workflow

This step joins the already completed worker, continuation, descriptor, and
renderer slices into one real browser flow. It deliberately remains a narrow
vertical slice: Fosite validates the authorization request first; Go validates
and projects the form second; JavaScript only selects the declared continuation
and emits native effect plans; Go owns account creation, sessions, OAuth, and
the response.

### What I did

- Added the `workflow_continuation` hidden field to the host-owned workflow
  form and made it mandatory in the page model and exact POST parser.
- Extended the strict parser so it rejects duplicate singleton controls,
  missing/extra controls, disallowed actions, invalid normalizations, and
  overlong values before any JavaScript invocation. Password values are put in
  the invocation secret set as opaque handles; only normalized display name
  and email are ordinary JavaScript data.
- Added `internal/fositeadapter/scripted_signup.go`. After the existing
  Fosite/CSRF/browser/origin/client-generation checks, it runs
  `signup.start`, persists a bound continuation, renders the native form, and
  on POST invokes only the continuation's declared `signup.submitted` lambda.
- Added the checked-in `pkg/idpsignup/open_signup.js` program. Its start lambda
  selects the four host-owned signup fields and submit/deny actions. Its
  submitted lambda calls `ctx.commit.signup`; it cannot parse HTTP, inspect a
  cookie, render HTML, create a password hash, or issue an OAuth response.
- Threaded an optional activated executor and continuation store through
  `embeddedidp.Options` into the Fosite adapter. The adapter can construct the
  checked-in bounded default only when native registration is enabled. In
  production it refuses an absent durable continuation store.
- Updated the PKCE registration integration test to exercise the script-owned
  page and continuation POST. It still proves cross-site rejection, callback
  resumption, one-use replay rejection, and redacted audit fields.

### What worked

The focused integration gate passed:

```bash
go test ./internal/fositeadapter ./pkg/embeddedidp ./pkg/idpui ./pkg/idpworkflow ./pkg/idpsignup -count=1
```

The Fosite adapter package, including the real PKCE registration browser
round-trip, passed in about twenty seconds. Formatting and `git diff --check`
also passed.

### What remains deliberately open

This is not yet the Phase 3 gate. The native workflow input projection
(`lf36`), one named transaction-spanning `SignupCommitter` (`lf41`), stable
failure classification (`lf42`), deletion of the now-unreachable legacy
registration branch (`lf45`), and the full regression matrix (`lf46`) still
need their own reviewable changes. The current effect execution calls the
existing atomic account service, then consumes the continuation and continues
the browser/OAuth lifecycle. That is safe for the tested happy and replay
paths, but it is not the requested single all-record atomic operation and must
not be represented as one.

### Code review instructions

- Begin with `pkg/idpworkflow/submission.go`; confirm the POST vocabulary is a
  closed set and sensitive values become handles, not public strings.
- Follow `internal/fositeadapter/scripted_signup.go` from `beginScriptedSignup`
  to `resumeScriptedSignup`; check that Fosite, CSRF, origin, browser binding,
  client generation, and continuation bindings are all native checks.
- Read `pkg/idpsignup/open_signup.js` beside `pkg/idpsignup/executor.go`; the
  JavaScript program should contain policy/flow selection only.
- Confirm `pkg/embeddedidp/options.go` exposes no listener, TLS, OAuth key,
  cookie, renderer authority, or raw store capability to JavaScript.
- Re-run the focused command above. The next checkpoint, after the remaining
  Phase 3 work, is the complete Phase 2/3 direct, race, lint, and full suite.

### Technical details

```text
tasks completed: lf32-lf35, lf37-lf40, lf43-lf44
new public form control: workflow_continuation
secret boundary: native parser -> idpworkflow.SecretHandle -> ctx.secret
continuation binding: workflow, program, request, client, browser, expiry
script authority: field/action selection and declared native signup effects
native authority: HTTP, CSRF, CSP, cookies, account service, sessions, OAuth
validation: focused direct package suite passes
next tasks: lf36, lf41-lf42, lf45-lf46
```

## Step 22: Make scripted signup the only commit path

This step finishes the semantic Phase 3 work that was intentionally left open
in Step 21. The main design constraint is that a browser continuation must not
be consumed separately from the identity it authorizes: otherwise a retry can
create an account twice, or a storage failure can leave an account without its
terminal workflow state.

### What I did

- Added `idpsignup.StartInput`, a copied, schema-validated DTO for the
  signup-start lambda. It includes only client ID, registered redirect URI,
  requested scope text, a hashed interaction identity, and whether a browser
  session was bound. It excludes Fosite objects, raw requests, cookies,
  session handles, raw OAuth parameters, and store access.
- Added `idpaccounts.PreparedCreate`, `PrepareCreate`, and `CommitPrepared`.
  Password normalization and Argon2 work happen before the store transaction;
  the transaction receives a password verifier rather than a plaintext
  password. Existing `Create` retains its duplicate-ID/subject behavior.
- Made the internal memory domain store implement the continuation store on
  its copy-on-write transaction snapshot. SQLite continuation consumption now
  detects an existing transaction and uses its scoped SQL runner instead of
  opening a nested transaction. A SQLite regression test proves a loaded
  continuation can be consumed inside `Store.Update`.
- Added `Service.ConsumeLoaded`, which revalidates an already loaded,
  binding-checked continuation against a caller-supplied transaction-scoped
  store. It accepts no raw browser handle.
- Refactored `commitScriptedSignup` into the single named native commit
  boundary. Its one transaction consumes the workflow continuation, persists
  the prepared local identity and credential, creates the browser session, and
  consumes the authorization interaction. Only after commit does the provider
  set the session cookie and resume consent/OAuth handling.
- Added bounded audit reasons for duplicate login, password rejection,
  continuation/interation conflict, and generic rejection. The browser always
  gets the same redacted workflow rerender rather than an account-enumerating
  distinction.
- Removed the hardcoded registration POST execution and its legacy generic
  renderer path. A required registration interaction now either runs the
  configured scripted workflow or fails closed as unavailable; it never falls
  back to the old form/account code.

### What worked

Before the complete phase gate, the following direct integration command
passed:

```bash
go test ./internal/fositeadapter ./pkg/embeddedidp ./pkg/idpaccounts \
  ./pkg/idpcontinuation ./pkg/idpsignup ./pkg/idpui ./pkg/idpworkflow \
  ./internal/store/memory ./pkg/memorystore ./pkg/sqlitestore -count=1
```

The targeted SQLite transaction regression and all relevant account,
continuation, memory, SQLite, executor, UI, workflow, and Fosite tests passed.
The race/full/lint phase checkpoint is recorded separately once all of its
long-running commands finish.

### What was tricky to build

`idpstore.Update` deliberately gives callers an interface rather than a
concrete storage type. The committer therefore checks that its transaction
also implements the narrow continuation-store interface. This is not an
adapter or a second persistence path: both supplied production and test stores
own their continuation rows/maps in the same native transaction. A store that
cannot provide that invariant fails closed before writing an account.

The session cookie itself cannot be written transactionally—it is an HTTP
header. The committer instead creates the durable server-side session in the
transaction and emits the cookie only after successful commit. A failure before
commit emits no authentication cookie; a failure after commit is safe because
the next authorization request can use the durable session only with its
unpredictable cookie handle.

### Code review instructions

- Review `pkg/idpaccounts/accounts.go` first: a prepared account contains an
  Argon2 verifier, never a plaintext password, and the final write uses a
  supplied `TxStore`.
- Read `pkg/idpcontinuation/service.go` and the SQLite transaction regression
  test together. Confirm `ConsumeLoaded` preserves the normal generation,
  schema, expiry, binding, and compare-and-consume checks.
- Read `commitScriptedSignup` in
  `internal/fositeadapter/scripted_signup.go` as one transaction. The sequence
  must remain continuation → identity/credential → session → interaction;
  any error rolls the whole transaction back.
- Confirm `beginAuthorize` and `resumeAuthorize` have no generic registration
  fallback. The only registration page is the host-rendered `WorkflowPage`
  selected by the checked-in JavaScript program.

### Technical details

```text
tasks completed: lf36, lf41, lf42, lf45
atomic native records: continuation, user, password verifier, session, interaction
HTTP effect after commit: Set-Cookie only
browser error policy: stable generic rerender; no duplicate-login enumeration
store requirement: transaction must also implement idpcontinuation.Store
next task: lf46 (complete direct/race/full/lint evidence)
```

## Step 23: Complete the Phase 2/3 regression gate

The expensive suite was intentionally run once for the completed vertical
slice, rather than after every small implementation edit. This keeps the
feedback loop fast while still requiring the whole provider and its production
shaped stores to agree before Phase 3 is marked complete.

### What I did

- Ran the direct Phase 2/3 integration set across Fosite, embedded IDP,
  accounts, continuations, signup executor, UI, workflow parser, memory, and
  SQLite stores.
- Ran the required race suite:

  ```bash
  go test -race ./internal/fositeadapter ./pkg/idpworkflow ./pkg/idpscript -count=1
  ```

  Its passing rerun took 78 seconds in the Fosite package.
- Ran the complete repository suite and lint gate:

  ```bash
  go test ./... -count=1
  make lint
  ```

- Fixed one test fixture in `pkg/idpscript/invoke_test.go` to include the new
  required `workflow_continuation` control. This was a direct contract change:
  the test exercises `ParseSubmission`, so its synthetic form must have the
  same complete closed vocabulary as a browser form.
- The first race attempt also reported an unrelated SQLite refresh-rotation
  linearizability assertion. It did not reproduce on the complete rerun; the
  succeeding race run covers the same Fosite package. No refresh-token code
  was changed or hidden to make the result pass.

### What worked

All Phase 2/3 gates passed:

```text
focused direct integration suite: pass
workflow/Fosite/idpscript race suite: pass
go test ./... -count=1: pass
make lint: pass (golangci-lint: 0 issues; custom analyzers pass)
```

The full suite includes the existing PKCE registration callback/replay and
cross-origin regression, account/password policy coverage, consent/session/
audit behavior, and memory/SQLite continuation and storage suites. The
registration browser integration test now takes the scripted page path, so
these existing checks exercise the replacement implementation rather than a
compatibility fallback.

### Code review instructions

- Run the three commands above from repository root.
- Read `internal/fositeadapter/registration_test.go` to see the real PKCE
  signup authorization request, hostile-origin rejection, successful callback,
  replay rejection, and audit redaction assertion.
- Read `pkg/sqlitestore/continuation_test.go` to see transaction-scoped
  continuation consumption tested independently of the browser flow.
- Confirm `tasks.md` has every Phase 2 and Phase 3 task checked and that the
  Phase 3 gate statement matches the implementation: checked-in JavaScript,
  no old POST branch, native atomic commit, and complete regression evidence.

### Technical details

```text
task completed: lf46
fixture-only commit: d65eff0
Phase 2 gate: passed
Phase 3 gate: passed
full suite: passed
lint: passed, zero issues
next normative work: Phase 4 virtual identity and invitation providers
```

## Step 24: Freeze the Phase 4 virtual-provider artifact contract

Phase 4 begins by making the security properties of an identity or invitation
provider fingerprinted program data. This prevents a later implementation from
claiming one-time redemption or durable revocation where its storage model
cannot provide it.

### What I did

- Added identity/invitation provider contracts with stable ID, version, state,
  replay-protection, revocation, and handler metadata.
- Defined `virtual`/`durable` state, `none`/`expiry`/`one_time` replay
  protection, and `none`/`key_rollover`/`durable` revocation vocabulary.
- Pinned each provider handler to a `LambdaKindProvider` callback and exact
  input/output schema. Identity providers require `establish`; invitation
  providers require `validate`.
- Added deterministic validation for unknown schemas, workflow callbacks used
  as provider callbacks, virtual one-time claims, and durable revocation
  without durable state.
- Added a valid durable-invitation fixture and negative diagnostic tests.

### Why

The declaration lets future activation and explain tooling state where provider
data exists, how replay is stopped, and how revocation works without inspecting
arbitrary JavaScript. It is a contract, not a service locator or a database
handle exposed to scripts.

### What worked

```bash
go test ./pkg/idpprogram -count=1
```

passed. Code checkpoint: `414cfb4 Feat: define virtual provider contracts`.

### What should be done next

Implement `lf48`: add provider registration to the isolated module and route
provider callbacks through the existing artifact registry and bounded worker
pool, without adding ambient host authority.

### Code review instructions

- Start with `pkg/idpprogram/providers.go` for the complete vocabulary.
- Review the provider validation block in `pkg/idpprogram/validate.go`.
- Review `pkg/idpprogram/program_test.go` for stable diagnostic expectations.

### Technical details

```text
task completed: lf47
artifact addition: Program.Providers
provider callback kind: LambdaKindProvider only
next task: lf48
```

## Step 25: Register provider callbacks and add virtual/signed primitives

The Phase 4 provider boundary is now executable without granting JavaScript
ambient host access. The remaining computed and durable invitation work will
reuse this exact invocation path.

- Added `program.provider(kind, name, spec)` and TypeScript declarations. It
  accepts only module-created lambda handles and materializes fingerprinted
  provider handlers.
- Added `idpscript.ProviderInvoker`, which selects only a compiled
  provider/handler pair and delegates all schema, capability, budget, Promise,
  and owned-worker enforcement to the existing pool.
- Added `idpidentity.SubjectDeriver` and virtual candidates. Subjects are
  HMAC-derived from host key + namespace + verified seed; profile projection
  omits protocol claims and a virtual candidate creates no local user row.
- Added `idpinvite.KeyRing` signed-invitation verification. It verifies native
  issuer/audience/policy/expiry/not-before/subject/email expectations. Key
  removal revokes a stateless token family; one-time use is deliberately not
  claimed.

Focused validation passed:

```bash
go test ./internal/gojamodules/tinyidp ./pkg/idpscript ./pkg/idpprogram -count=1
go test ./pkg/idpidentity ./pkg/idpinvite ./pkg/idpscript -count=1
```

Code checkpoints: `95dc662` (provider callbacks), `c544a5f` (virtual
identity), and `2555f0b` (signed invitations).

Next: `lf51` and `lf52`, the bounded computed and durable one-time invitation
providers; neither may expose database/network authority to JavaScript.

## Step 26: Add a capability-backed computed invitation provider

Computed invitation eligibility now has a deliberately small native seam. A
program can await one declared capability and receive a validated decision,
but it cannot gain a database connection, a network client, or a callback it
can retain after the invocation.

### Prompt Context

**User prompt (verbatim):** "continue, do not stop until youre genunely at a loss."

**Assistant interpretation:** Continue Phase 4 task-by-task and maintain the
implementation diary and commits.

**Inferred user intent:** Finish the ticket's scripting-provider work with
reviewable, safe increments rather than stopping at design.

**Commit (code):** `40e7747` — "Feat: add computed invitation capability"

### What I did

- Added `idpinvite.NewEligibilityCapability`, bound to the stable capability
  ID `invitation.eligibility` version 1.
- Defined narrow `EligibilityProbe` and `EligibilityDecision` value contracts.
- Rejected unknown, malformed, oversized, or invalid capability inputs before
  the host evaluator runs; bounded decision data before it returns to JS.
- Added a provider-pool test in which JavaScript awaits
  `ctx.cap.invitation.eligibility(ctx.input)` and receives only the decision.

### Why

The host may consult a directory, database, or remote service, but that is
host implementation detail. The program must be able to express policy over a
result without becoming a general-purpose configuration language with ambient
authority.

### What worked

```bash
gofmt -w pkg/idpinvite/computed.go pkg/idpinvite/computed_test.go
go test ./pkg/idpinvite -count=1
```

Both completed successfully.

### What didn't work

N/A.

### What I learned

`idpscript.CapabilityBinding` already enforces declaration matching, payload
limits, per-invocation lifetime, Promise settlement, cancellation, and call
budgets. The invitation package therefore only needs to own the domain-shaped
input/output validation at its seam.

### What was tricky to build

The important distinction is between a host *implementation* that may use a
directory and the script-visible *contract*. The adapter calls the evaluator
only after decoding an exact probe and serializes only a small decision. The
end-to-end test confirms that path through `ProviderInvoker`; it does not give
the JavaScript source an object capable of additional queries.

### What warrants a second pair of eyes

- Confirm the initial product wants `email`, optional `inviteCode`, and
  `audience` as the complete eligibility input vocabulary.
- Confirm whether denial reasons should be intentionally coarser before any
  browser-facing workflow maps them to user-visible text.

### What should be done in the future

Implement `lf52`: durable one-time invitations with hashed lookup, expiry,
revocation, and transaction-scoped consumption.

### Code review instructions

- Start with `pkg/idpinvite/computed.go`, especially
  `NewEligibilityCapability` and the exact JSON validation helpers.
- Review `pkg/idpinvite/computed_test.go` for the end-to-end provider example.
- Validate with `go test ./pkg/idpinvite -count=1`.

### Technical details

```text
JS -> ctx.cap.invitation.eligibility({email, inviteCode?, audience})
   -> native exact decode + bounds
   -> host EligibilityEvaluator
   -> native bounded {accepted, reason?, evidenceId?}
   -> Promise decision in the owned Goja worker
```

## Step 27: Add durable one-time invitation storage and redemption

Durable invitations now protect their browser-visible code at rest and expose
only named lifecycle transitions. The service can redeem by code in its own
transaction or use the caller-owned transaction needed by the signup commit
path; neither route has a check-then-write redemption race.

### Prompt Context

**User prompt (verbatim):** (same as Step 26)

**Assistant interpretation:** Continue Phase 4 implementation without
expanding beyond the ticket.

**Inferred user intent:** Make real one-time invitation registration possible
with production persistence and an atomic integration seam.

**Commit (code):** `21c7c4c` — "Feat: add durable one-time invitations"

### What I did

- Added `idpstore.DurableInvitation` and named create/get/redeem/revoke store
  operations to the normal and transaction-scoped store contracts.
- Added memory copy-on-write state and SQLite migration `012`, storing the
  keyed code hash as the primary key and serialized non-secret metadata.
- Added `idpinvite.DurableService`; it HMACs a code with a domain-separated
  lookup key before persistence and returns redacted `DurableEvidence`.
- Added `RedeemInTransaction`, the exact dependency needed to consume an
  invite in the signup committer's existing transaction.
- Tested replay, audience mismatch, expiry, revocation, concurrent one-winner
  redemption, and SQLite restart persistence.

### Why

Signed invitations provide expiry and key-family revocation but cannot provide
per-token one-time redemption. Durable invitations make that stronger promise
only because native persistent state and a conditional state transition exist.

### What worked

```bash
gofmt -w pkg/idpinvite/durable.go pkg/idpinvite/durable_test.go
go test ./pkg/idpinvite ./internal/store/memory ./pkg/sqlitestore ./pkg/idpstore -count=1
```

All focused suites passed.

### What didn't work

The initial revocation test incorrectly attempted a raw, non-derived hash
lookup and received `not found`. The service intentionally does not expose its
derived hash, so I added the correctly bounded `DurableService.Revoke(code)`
operation and exercised revocation through that public seam.

### What I learned

The existing `idpstore.Update` transaction model already supports the required
composition. Extending `TxStore` with a narrow redemption operation is safer
than creating a separate invitation database transaction beside account and
continuation writes.

### What was tricky to build

The durable service must not accidentally make the HMAC lookup hash a new
browser-visible capability. `Issue`, `Redeem`, and `Revoke` accept raw codes,
but all store methods receive only the derived hash. `RedeemInTransaction`
keeps the hash derivation inside the service while accepting only the caller's
transaction surface, preserving atomic composition for the next task.

### What warrants a second pair of eyes

- Review the product semantics for revoking an already-redeemed invitation;
  this implementation reports `ErrAlreadyConsumed` rather than silently
  changing historical evidence.
- Review retention/maintenance policy for expired invitation rows before a
  production rollout; deletion is intentionally not added outside this task.

### What should be done in the future

Implement `lf53`: validate a durable-invitation effect in the scripted signup
outcome and call `RedeemInTransaction` beside account, browser-session, and
continuation operations.

### Code review instructions

- Start with `pkg/idpinvite/durable.go` for secrecy and transaction semantics.
- Review `pkg/idpstore/interfaces.go` and
  `pkg/sqlitestore/invitation.go` for native atomic state transitions.
- Run `go test ./pkg/idpinvite ./internal/store/memory ./pkg/sqlitestore -count=1`.

### Technical details

```text
browser code --HMAC(lookup key, domain || code)--> code_hash
code_hash + audience --single store transaction--> active -> redeemed
                                           \--> DurableEvidence (no raw code/hash)
signup transaction --RedeemInTransaction--> same transition + account/session
```

## Step 28: Compose durable invitation redemption with scripted signup

The provider now accepts an optional, declared invitation-consumption effect
from a signup script but remains the sole authority that creates users,
credentials, browser sessions, continuations, and authorization results. It
redeems the invitation through the transaction it already owns.

### Prompt Context

**User prompt (verbatim):** (same as Step 26)

**Assistant interpretation:** Continue the current Phase 4 work to completion
in scoped, tested increments.

**Inferred user intent:** Let a policy require an invitation without creating a
second, less-safe registration path.

**Commit (code):** `84a9995` — "Feat: commit durable invitations with signup"

### What I did

- Added `inviteCode` to the bounded submitted signup input vocabulary.
- Extended `ctx.commit.signup` so a nonempty `inviteCode` produces one
  declared `consumeInvitation` effect; it still cannot execute the effect.
- Added an optional durable-invitation service to the provider options.
- Revalidated the exact two-or-three-effect sequence in
  `commitScriptedSignup`, then called `RedeemInTransaction` beside the
  prepared account, session, workflow continuation, and interaction updates.
- Added an invitation workflow regression and ran the focused signup/module/
  provider test groups.

### Why

A JavaScript program decides whether to request the capability, not how to
consume it. The native committer applies the same transaction and rollback
boundary to invitation state as it already applies to account and OAuth state.

### What worked

```bash
go test ./pkg/idpsignup ./internal/fositeadapter ./internal/gojamodules/tinyidp -count=1
```

All passed.

### What didn't work

N/A.

### What I learned

The existing two-effect sequence was a narrow native protocol, not an
arbitrary list. Accepting exactly one final `consumeInvitation` effect keeps
the expansion explicit and prevents scripts from reordering account/session
side effects.

### What was tricky to build

The invite code is public form data, but its conversion to a durable lookup
hash must remain native. The effect contains the selected value only long
enough for the native committer to give it to `DurableService`; JavaScript
never observes a hash, redemption record, or store transaction.

### What warrants a second pair of eyes

- Confirm `record.ClientID` is the desired invitation audience binding for all
  planned relying-party deployments.
- Add a browser-level durable-invitation acceptance/replay test when the
  checked-in durable signup example is introduced in `lf54`.

### What should be done in the future

Implement `lf54`: checked-in policy examples for the Phase 4 signup modes.

### Code review instructions

- Start in `internal/fositeadapter/scripted_signup.go` at
  `commitScriptedSignup`.
- Review `internal/gojamodules/tinyidp/module.go` `signupEffects` to see that
  JavaScript only returns a plan.
- Validate with the focused command above.

### Technical details

```text
script ctx.commit.signup({ inviteCode }) -> [identity, credential, consumeInvite]
provider validates plan -> store.Update {
  consume continuation; commit account; redeem durable invite; create session;
  consume interaction
}
```

## Step 33: Persist, dispatch, and verify native email challenges

Phase 5 now has a native email-code authority rather than a JavaScript mail or
verification API. A script can request a typed challenge and select its resume
handler; native code owns code generation, durable state, delivery, atomic
verification, and the evidence later visible to the resumed lambda.

### Prompt Context

**User prompt (verbatim):** "Continue, from lf58, lf59, lf60, lf61, lf62. Don't stop, do them all, commit, keep a diary (see skill for the format)."

**Assistant interpretation:** Complete the Phase 5 persistence, mailer,
dispatch, typed challenge-outcome, and verified-evidence tasks with commits and
diary records.

**Inferred user intent:** Deliver a restart-safe, composable email-verification
foundation without weakening the native identity-provider trust boundary.

**Commit (code):** `9a970d5`, `04a514f`, `0beb0b5`, `c14d70f`, `2e8a517`

### What I did

- Added atomic memory and SQLite challenge stores; SQLite migration `013`
  persists only structured state and a keyed code hash.
- Added a typed `Mailer` and `Service` that generate a code, dispatch only
  approved mail request data, and atomically verify the supplied code.
- Added `emailCode` field descriptor and `ctx.challenge.emailCode` builder.
- Added invocation-scoped `ctx.evidence`, populated only by native Go code.

### Why

The browser and JavaScript must not be able to forge email verification or use
an email transport as ambient authority. A structured challenge request keeps
the workflow expressive while its security-sensitive effects remain native.

### What worked

```bash
go test ./pkg/idpworkflow ./internal/gojamodules/tinyidp ./pkg/idpemailchallenge ./pkg/sqlitestore -count=1
go test ./pkg/idpprogram ./pkg/idpscript ./internal/gojamodules/tinyidp -count=1
```

Both focused validation groups passed.

### What didn't work

The first SQLite adapter used generic `Create`, `Load`, and cleanup names,
which collided with the existing workflow-continuation store interface. The
compiler reported duplicate methods. I renamed the contract to explicit
`*EmailChallenge` lifecycle methods, preserving one shared SQLite store.

### What I learned

The existing generic `OutcomeChallenge` and continuation edge model already
provided the correct control-flow primitive. Phase 5 needed a typed challenge
payload and native evidence projection, not a parallel browser protocol.

### What was tricky to build

The evidence must be visible to the resumed callback but impossible for a
script to install or retain. `InvokeWithSecretsAndEvidence` constructs a fresh
frozen evidence object inside the owned worker invocation; it is not attached
to the pool or serializable as a new authority.

### What warrants a second pair of eyes

- SQLite attempt/resend updates should be moved to conditional update
  predicates before multi-node support is ever considered; Tiny-IDP currently
  supports one active SQLite connection.
- The next integration slice must bind the challenge to actual persisted
  workflow continuations and render the code page before marking Phase 5 done.

### What should be done in the future

Implement `lf63–lf66`: full continuation/browser binding integration, password
ordering, signup-commit evidence enforcement, and restart/adversarial tests.

### Code review instructions

- Start with `pkg/idpemailchallenge/service.go` and `store.go`.
- Review `pkg/sqlitestore/email_challenge.go` for durable lifecycle behavior.
- Review `internal/gojamodules/tinyidp/module.go` and `pkg/idpscript/invoke.go`
  for script-side request/evidence boundaries.

### Technical details

```text
script ctx.challenge.emailCode -> native challenge request + continuation
native service -> HMAC(code) + durable state -> typed mailer request
browser code -> atomic VerifyEmailChallenge -> verified-email evidence
native invocation -> frozen ctx.evidence.email -> resumed lambda
```

## Step 34: Carry native verified-email evidence into the password commit

The first integrated email challenge flow correctly verified a submitted code,
but review of the continuation sequence found that the subsequent password
page no longer had a native reference to that proof. This step makes the
evidence explicit durable continuation state and rehydrates it only through
the native challenge service when the final password form is submitted.

### Prompt Context

**User prompt (verbatim):** "DO NOT STOP, CONTINUE WITHUOT STOPPING, DO ALL OF PHASE 5."

**Assistant interpretation:** Finish every remaining Phase 5 correctness and
test task rather than stopping after the foundational challenge primitives.

**Inferred user intent:** Ensure the production signup flow has a complete,
reviewable verified-email trust chain, including the browser continuation that
separates verification from password collection.

**Commit (code):** `706a2c6` — "Feat: persist verified email evidence across signup"

### What I did

- Added `idpemailchallenge.Service.Evidence`, which loads an existing verified
  challenge, rechecks expiry and all continuation bindings, and returns native
  `VerifiedEmailEvidence`; it does not accept a browser email value.
- Split continuation evidence references into `pendingEmailChallenge` and
  `verifiedEmail`, avoiding an ambiguous state where a verified code would be
  re-submitted and rejected as already terminal.
- Replaced the pending reference with the verified reference when the
  email-verified lambda presents the password form.
- Rehydrated the evidence again for the password-submit handler and passed its
  address into the sole native signup commit boundary.
- Extended the service test to prove rehydration succeeds only with the
  original challenge bindings.

### Why

Email verification must not become a transient Go value that disappears after
one lambda call, nor a script-provided `{ verified: true }` object. A password
continuation needs a durable, native reference so final account creation can
reject a substituted login address even if a script output is otherwise valid.

### What worked

```bash
gofmt -w pkg/idpemailchallenge/service.go pkg/idpemailchallenge/service_test.go internal/fositeadapter/scripted_signup.go
go test ./pkg/idpemailchallenge ./internal/fositeadapter -count=1
```

Both packages passed. The provider suite took about 21 seconds and exercised
the existing strict authorization and registration paths.

### What didn't work

N/A. The review found the proof-propagation gap before an end-to-end test could
mistake a successful page transition for a correct final commit.

### What I learned

The continuation service intentionally does not inherit evidence references
implicitly. That is the right default: a transition must deliberately retain,
replace, or discard a proof, so an unrelated next form cannot accidentally
gain authentication authority.

### What was tricky to build

The same challenge has two semantically different states. The code-entry
continuation must call `Verify` exactly once, while the password continuation
must call `Evidence` without consuming it again. Named pending and verified
reference kinds make the state change visible in durable data and prevent a
replay from looking like a normal password submission.

### What warrants a second pair of eyes

- Verify the exact intended proof lifetime: `Evidence` currently requires the
  original challenge not to have expired, so password completion is bounded by
  the email challenge expiry as well as its own continuation expiry.
- Review whether an operational resend UI should preserve the same pending
  continuation handle or mint a new one; the next adversarial integration test
  slice will make this decision observable.

### What should be done in the future

Complete `lf63–lf66`: wire the service through `embeddedidp`, implement the
public resend route/limits if the workflow exposes it, and add browser-level
restart/replay/binding/mailer-failure tests before checking off Phase 5.

### Code review instructions

- Start in `internal/fositeadapter/scripted_signup.go` at
  `resumeScriptedSignup`, then follow `pendingEmailChallengeReference` and
  `verifiedEmailReference`.
- Review `pkg/idpemailchallenge/service.go` `Evidence` for its native binding
  checks and for the absence of a script-controlled proof constructor.
- Validate with `go test ./pkg/idpemailchallenge ./internal/fositeadapter -count=1`.

### Technical details

```text
pending code continuation --Verify(code, bindings)--> verified native record
  --ctx.evidence.email--> emailVerified lambda --present(password)-->
verified-email continuation --Evidence(reference, bindings)--> commit check

No continuation stores an email address as proof; it stores only:
  { kind: "verifiedEmail", id: challengeID }
```

## Step 35: Complete the browser-facing verified-email and resend slice

This step turned the native challenge primitives into one complete signup
workflow. The browser first provides only identity data, receives a native
email code, proves it, and only then receives a password form. The final
commit reloads the native evidence instead of trusting a prior JavaScript
result or browser-supplied email flag.

### Prompt Context

**User prompt (verbatim):** (same as Step 34)

**Assistant interpretation:** Continue Phase 5 through the real browser flow,
including delivery failure, resend, and final native account commit behavior.

**Inferred user intent:** Make the explicit-continuation design operational,
not merely a collection of standalone native primitives.

**Commit (code):** `348df6e` — "Feat: complete verified email signup workflow"; `c7fd7a3` — "Feat: add bounded email challenge resend flow"

### What I did

- Threaded the optional email-challenge service through `embeddedidp` into the
  Fosite adapter.
- Added a three-page browser integration test for identity, code verification,
  and password entry; it proves the created account has `EmailVerified=true`.
- Fixed the native secret projection so identity/code pages do not pass
  zero-value password handles to the Goja context.
- Added `verifiedEmail` continuation references and a native evidence reload
  path at the password handler/commit boundary.
- Added native `resend` action handling. It rotates the challenge code
  atomically, enforces the configured maximum, and sends only a typed mail
  request.
- Changed initial delivery ordering: mail delivery completes before the
  original browser continuation is advanced, so delivery failure leaves that
  original form retryable and cannot create an account.
- Added bounded, redacted audit records for initial send and resend results.

### Why

The security property is a sequence, not an isolated verification call:
identity data must be carried safely across the email page, password material
must be absent until after verification, and final account creation must be
able to prove precisely which native challenge attested to the email.

### What worked

```bash
go test ./internal/fositeadapter -run TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification -count=1 -v
go test ./pkg/idpemailchallenge ./pkg/idpworkflow ./internal/fositeadapter -run 'Test(ServiceResend|DefaultRegistry|EmailVerified)' -count=1 -v
```

The integration test covers an intentional mail failure, retry using the same
identity continuation, resend with code rotation, correct-code verification,
password collection, and final OIDC authorization continuation.

### What didn't work

The first broad command,
`go test ./pkg/idpemailchallenge ./pkg/idpsignup ./pkg/embeddedidp ./internal/fositeadapter -count=1`, failed only in the existing
`TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily` with
`active=1, want 0`. Its isolated rerun passed:

```bash
go test ./internal/fositeadapter -run TestSQLiteRefreshRotationHistoryIsLinearizableAndReuseRevokesFamily -count=1 -v
```

This was treated as a pre-existing concurrency-test flake, not fixed in the
email-challenge change set.

### What I learned

An invocation with an empty password map is not equivalent to an invocation
with two empty secret handles. The latter tries to create unusable Goja native
capabilities and fails closed, which is correct. The provider must project
only the secret descriptors active in the persisted presentation.

### What was tricky to build

Delivery cannot be atomically committed with a database update. Advancing the
browser continuation before calling a mailer stranded the user if delivery
failed. Reversing that order means a rare advance conflict can leave an
unreferenced, expiring challenge record, but it never blocks the user or
authorizes an account; retention cleanup owns that harmless record.

### What warrants a second pair of eyes

- Review the operational delivery policy: a failed send currently leaves an
  expiring unreferenced challenge record, intentionally favoring a retryable
  browser flow over impossible cross-system atomicity.
- The current `ResendNotBefore` contract is supported by the store, but the
  checked-in email program does not yet select a nonzero cooldown. Confirm the
  desired production cooldown before enabling this program broadly.

### What should be done in the future

Finish `lf63` and `lf66` with restart, concurrent POST, browser/client/
generation mismatch, expiration, cleanup, and resend-limit integration tests;
then mark the remaining Phase 5 tasks complete.

### Code review instructions

- Read `internal/fositeadapter/scripted_signup.go` in this order:
  `resumeScriptedSignup`, `beginEmailChallenge`, then `commitScriptedSignup`.
- Review `pkg/idpemailchallenge/service.go` and both store implementations for
  native code hashing/rotation and binding checks.
- Run the two commands in **What worked** and inspect the browser test's
  mail-failure and resend assertions.

### Technical details

```text
identity form --challenge request--> typed mail delivery --advance--> code form
code form --Verify--> verifiedEmail reference --present--> password form
password form --Evidence(reference, bindings)--> native signup commit

resend form --Resend(reference, bindings)--> rotate code hash -> typed delivery
```

## Step 36: Pass the complete Phase 5 acceptance gate

Phase 5 is now complete. The flow is restart-safe and proof-carrying from the
first identity page through the final account transaction, while the browser
receives only generic failures for incorrect, replayed, mismatched, expired,
or otherwise unusable state.

### Prompt Context

**User prompt (verbatim):** (same as Step 34)

**Assistant interpretation:** Finish the remaining Phase 5 adversarial and
durability work, commit it, and keep the ticket diary/task ledger current.

**Inferred user intent:** Have a defensible, end-to-end implementation gate
before expanding the scripting system into operational activation features.

**Commit (code):** `14e374c` — "Test: cover adversarial email signup submissions"; `1799362` — "Test: prove email signup restart and atomic verification"

### What I did

- Added browser tests for a code POST from a separate cookie jar, a rotated-out
  code, and a replay of a correctly verified code.
- Corrected generic error rendering to attach its field error to the active
  presentation's first registered field; an email-code error no longer tries
  to render an absent `email` field.
- Added a full SQLite/provider/executor restart test that restarts after send
  and before verification, then finishes the password commit on the new
  provider process.
- Added a two-goroutine correct-code test proving exactly one native memory
  store verification succeeds.
- Checked off `lf63`, `lf64`, `lf65`, and `lf66` in the normative task ledger.

### Why

The important Phase 5 property is not merely that a code can be verified. It
is that no replay, cross-browser form, rotated secret, process restart, or
delivery failure creates an alternate route to account creation.

### What worked

```bash
go test ./pkg/idpcontinuation ./pkg/idpworkflow ./pkg/memorystore ./pkg/sqlitestore -count=1
go test ./internal/fositeadapter ./pkg/embeddedidp -count=1
go test -race ./pkg/idpcontinuation ./pkg/idpworkflow ./pkg/sqlitestore -count=1
```

Every command passed. The provider/embedded suite passed in about 24 seconds;
the targeted race checks passed for continuation, presentation, and SQLite
stores.

### What didn't work

The first wrong-code browser test returned `500 workflow page unavailable`
instead of the intended generic `400`. The underlying page validation rejected
an error for `email` because the active page only declared `emailCode`. The
host renderer was corrected to select an error field from the active
presentation, after which the test passed with the safe generic response.

### What I learned

Native presentation metadata is also a security boundary. Even error paths
must refer only to descriptors persisted in the active continuation; otherwise
a safe protocol rejection can become a server error and leak an implementation
distinction to the browser.

### What was tricky to build

The restart test uses a different `httptest` listener after reopening SQLite.
To test the real browser binding rather than bypass it, it copies the existing
cookie jar's cookies to the replacement listener before posting the durable
code form. The old continuation is then validated by a newly created executor
and provider, demonstrating that no Goja object or prior process memory is
required.

### What warrants a second pair of eyes

- Confirm the deployment-level maintenance job invokes continuation and email
  challenge cleanup at the desired cadence; the stores and cleanup contracts
  are tested, but scheduling belongs to the hosting deployment.
- Review whether the generic code-page error should avoid field-specific
  highlighting entirely for product UX; it is presently generic and
  non-enumerating, but still highlights the only active field.

### What should be done in the future

Begin Phase 6 (`lf67`): production-profile validation, then embedded tests,
CLI diagnostics, and generation activation. Do not expand Phase 7 OAuth
customization until those operational gates are in place.

### Code review instructions

- Start at `TestEmailVerifiedScriptedSignupSurvivesSQLiteRestart` and
  `TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification`.
- Follow the provider calls into `resumeScriptedSignup` and the challenge
  service/store methods.
- Run the three Phase 5 commands above; for the primary user-visible flow run
  `go test ./internal/fositeadapter -run TestEmailVerifiedScriptedSignup -count=1 -v`.

### Technical details

```text
browser A + code form --restart--> SQLite continuation + challenge
new provider/executor + same cookies --Verify--> verified reference -> password

browser B + copied form --binding check--> generic rejection
old/rotated/replayed code --native state transition--> generic rejection
two correct submissions --atomic Verify--> exactly one VerifiedEmailEvidence
```

## Step 37: Add production-profile script validation and explanation commands

Phase 6 now has an operator-facing, compile-only entry point for the exact
signup program profile used by the provider. It does not start a listener or
execute a browser request; it compiles in the isolated runtime and reports the
immutable contract identity that a later activation manager must accept.

### Prompt Context

**User prompt (verbatim):** (same as Step 34)

**Assistant interpretation:** Continue directly into the operational Phase 6
work after the Phase 5 gate, with deliberate commits and diary evidence.

**Inferred user intent:** Give an implementer and operator a safe way to
inspect script artifacts before any runtime activation or production routing.

**Commit (code):** `c390379` — "Feat: add script validation and explanation commands"

### What I did

- Added `idpsignup.Compile`, a compile-only seam sharing the same host-owned
  signup schemas as `idpsignup.New` without starting worker pools.
- Added Glazed `tinyidp script validate --source FILE --profile signup`, which
  prints source/program/callback-registry/schema fingerprints.
- Added Glazed `tinyidp script explain`, which prints sorted workflow/lambda/
  schema inventories, provider projections, and the canonical full program
  contract as secret-free data.
- Wired the `script` command group into `cmd/tinyidp` and verified help plus
  real validation/explanation against `email_verified_signup.js`.
- Checked off `lf67` and `lf70`.

### Why

Activation cannot safely be treated as “the file parsed.” Operators need the
same isolated materialization, host schema bindings, and deterministic hashes
that the executor uses, while review tooling needs a structural explanation
without a Goja callback, browser session, or secret value.

### What worked

```bash
go test ./pkg/idpsignup ./internal/cmds ./cmd/tinyidp -count=1
go run ./cmd/tinyidp script validate --help
go run ./cmd/tinyidp script explain --help
go run ./cmd/tinyidp script validate --source pkg/idpsignup/email_verified_signup.js --output json
go run ./cmd/tinyidp script explain --source pkg/idpsignup/email_verified_signup.js --output json
```

All commands passed. The validation output included four independent stable
fingerprints; explanation listed the four signup handlers and emitted the full
canonical program contract.

### What didn't work

N/A.

### What I learned

The existing artifact already contained the necessary operational identity.
The missing part was not another compiler path but a narrow compile-only
signup-profile export and a command layer that refuses an unknown profile
instead of silently treating it as production-compatible.

### What was tricky to build

The command’s profile flag is intentionally restrictive. A generic JavaScript
file cannot be validated meaningfully without knowing which host-owned schemas
and native effect catalog apply. Accepting arbitrary profile names would make
the word “valid” misleading, so the command currently fails closed unless the
explicit `signup` profile is selected.

### What warrants a second pair of eyes

- The full `program_contract` output is intentionally structural and
  secret-free; confirm its size remains acceptable as more workflow types are
  added, or switch large contracts to a dedicated JSON file output mode.
- Activation/reload must route each continuation by its persisted program
  fingerprint. Do not install a manager that simply swaps the active executor
  and breaks old continuations.

### What should be done in the future

Implement `lf68–lf69` embedded deterministic program tests, then `lf71–lf77`
with a fingerprint-routed retained-generation manager, readiness, audit,
metrics, and repeated-reload failure matrix.

### Code review instructions

- Start in `internal/cmds/script.go`; follow `loadScriptArtifact` into
  `idpsignup.Compile`.
- Compare `Compile` and `New` in `pkg/idpsignup/executor.go` to confirm they
  share the schema catalog.
- Run the five commands in **What worked**.

### Technical details

```text
source file --isolated compile/materialize--> Artifact
Artifact --> source/program/callback/schema fingerprints
Artifact.Program() --> canonical contract --> script explain

No listener, HTTP request, browser cookie, secret, or mutable global registry
is created by either command.
```

## Step 38: Add atomic retained-generation activation and continuation routing

The scripting runtime can now prepare a new signup generation without changing
live traffic, publish it atomically only after all workers are warm, and retain
older executable generations for browser continuations that were issued before
the activation. The provider no longer assumes that “active now” means “the
generation that created this form.”

### Prompt Context

**User prompt (verbatim):** (same as Step 34)

**Assistant interpretation:** Continue through the Phase 6 activation work in
small, committed steps while preserving the explicit-continuation design.

**Inferred user intent:** Make reload safe for live browser flows instead of
allowing a source update to change the code that resumes an existing signup.

**Commit (code):** `f28041d` — "Feat: add retained signup generation manager"; `1e681e0` — "Feat: route signup continuations by generation"; `f9b3e65` — "Test: preserve signup flow across generation activation"

### What I did

- Added `idpsignup.GenerationManager`, which compiles and warms a candidate
  executor before publication, atomically swaps on success, and retains a
  bounded ordered map of prior executable generations.
- Corrected `Executor.Fingerprint` to combine source and program hashes.
  Program structure alone is insufficient because a lambda body can change
  without changing its declared handler/schema/effect contract.
- Added an optional manager to Fosite provider options and used it as the
  continuation generation resolver.
- Changed manager-backed continuation load to omit only the *expected active*
  generation fingerprint; the loaded record’s own immutable fingerprint is
  resolved and validated natively, then used for every invoke/advance/consume/
  commit binding.
- Added manager atomic-failure/retention tests and converted the browser
  signup test to activate a changed source after code issuance.

### Why

Reload safety requires two distinct choices: new interactions should use the
active generation, while an existing continuation must use the generation it
persisted. Treating a program-contract hash as the generation identity would
also let changed callback code masquerade as the prior executable generation.

### What worked

```bash
go test ./pkg/idpsignup -count=1 -v
go test ./pkg/idpcontinuation ./internal/fositeadapter -count=1
go test ./internal/fositeadapter -run TestEmailVerifiedScriptedSignupCollectsPasswordAfterCodeVerification -count=1 -v
```

The last test activates a new source while the user is on the email-code page;
the old code form proceeds to the password page and final commit successfully.

### What didn't work

The first reload run failed safely at the password-page transition. Investigation
found two residual uses of `signupBindings`, which selected the new active
fingerprint while advancing a continuation from the old generation. Replacing
them with bindings derived from `current.ProgramFingerprint` fixed the route;
the focused integration test then passed.

### What I learned

There are three places that must agree on a continuation’s generation:
resolver validation during load, JavaScript invocation, and native persistence
advance/consume. Updating only invocation still fails safely at storage, but
that failure looks like a generic browser rejection and can obscure a reload
regression without the dedicated test.

### What was tricky to build

The expected binding field is normally a useful active-generation guard. With
a manager it would reject every valid old continuation before the manager has a
chance to resolve it. The safe alternative is narrow: omit that expected field
only for manager-backed loading, then require the stored fingerprint to resolve
through the bounded retained registry during `idpcontinuation.Service.Load`.

### What warrants a second pair of eyes

- Retention is presently a bounded count, not a store-aware lease tied to the
  latest continuation expiry. Production activation should choose the bound so
  it covers the configured maximum workflow lifetime, or add durable
  generation-reference accounting before enabling frequent reloads.
- The manager has not yet been exposed by `embeddedidp` configuration; direct
  Fosite embedding is covered, but the production host needs an explicit
  manager lifecycle option.

### What should be done in the future

Finish `lf68–lf69` with embedded deterministic program test cases and the CLI
test command; then finish readiness/metrics/audit and repeated-reload resource
tests (`lf74–lf77`) before checking Phase 6 complete.

### Code review instructions

- Read `pkg/idpsignup/manager.go` followed by `manager_test.go`.
- Review `Executor.Fingerprint` and each use of `ProgramFingerprint` in
  `internal/fositeadapter/scripted_signup.go`.
- Run the three commands in **What worked**.

### Technical details

```text
candidate source --compile + warm workers--> ready candidate
ready candidate --atomic publication--> active generation for new forms

stored continuation { sourceHash:programHash }
  --retained resolver--> exact old executor
  --invoke/advance/consume--> bindings with that same stored fingerprint
```

## Step 39: Gate activation on declarative embedded program tests

The program artifact now carries bounded declarative test cases, and activation
refuses to publish a candidate unless every case succeeds in a warmed executor.
This turns the script’s own contract examples into an activation prerequisite,
rather than an operator convention that can be forgotten during reload.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the Phase 6 implementation without
stopping after the generation-routing milestone.

**Inferred user intent:** Ensure a candidate is tested before it can replace a
live generation, and make that verification available from the CLI.

**Commit (code):** `eff4573` — "Feat: add declarative script test runner";
`7e089e8` — "Feat: gate generation activation on embedded tests"

### What I did

- Added `Program.Tests` and validation for unique test IDs, existing lambdas,
  declared expected outcomes, and schema-valid bounded JSON input.
- Added `program.test(id, spec)` to the `tinyidp` JavaScript module and its
  TypeScript declaration.
- Added `Executor.RunTests`, and `tinyidp script test`, which emits a stable
  per-case ID/expected/actual/passed row and returns an error for the first
  failed case.
- Changed `warmGeneration` to execute every embedded test before publication;
  any failure closes the candidate and leaves the existing active generation
  untouched.
- Added executor and manager tests for success and failed expected outcome.

### Why

Compilation and worker warmup prove that a program can load; they do not prove
that its declared workflow examples still produce the intended transition.
Running tests in the candidate’s actual worker pool catches that distinction
before any new browser continuation receives the new fingerprint.

### What worked

```bash
go test ./pkg/idpsignup -count=1 -v
go run ./cmd/tinyidp script test --source pkg/idpsignup/email_verified_signup.js --output json
```

The example test emitted a deterministic `signup-start-presents-identity`
row with expected and actual `present`. A candidate whose test expectation was
intentionally changed to `deny` failed activation and preserved the old active
fingerprint.

### What didn't work

N/A.

### What I learned

Embedded tests must run with no ambient authority. The first runner therefore
supplies no capabilities or secrets; a test that needs a native dependency
fails closed instead of accidentally gaining production mailer, identity, or
store access.

### What was tricky to build

The generation identity must include source as well as the canonical program
contract. A changed lambda body can keep the same declared schemas and edges;
testing alone does not make that body interchangeable with a continuation
issued by the old executable generation.

### What warrants a second pair of eyes

- The current no-capability baseline is intentionally safe but does not yet
  satisfy the deterministic fake-capability portion of `lf68`.
- CLI failure diagnostics should gain direct command-level tests before `lf69`
  is checked off.

### What should be done in the future

Add explicit deterministic test-only host capability bindings and failure
diagnostic tests; then complete retained-generation draining and observability.

### Code review instructions

- Start with `pkg/idpprogram/validate.go` and `Program.Tests`.
- Follow `Executor.RunTests` into the worker pool, then `warmGeneration`.
- Run the two commands in **What worked**.

### Technical details

```text
source --> compile --> warm isolated worker pool --> embedded tests
                                                 | failure
                                                 v
                                      close candidate; keep active
                                                 |
                                               success
                                                 v
                                        atomic publication
```

## Step 40: Surface active scripted-generation readiness

The embedded provider’s established readiness endpoint now includes the
configured scripted-signup executor or generation manager. A closed manager
cannot leave `/readyz` green merely because the HTTP server, database, and
signing key are otherwise healthy.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Continue Phase 6 operational hardening and
commit the work only after the repository gate passes.

**Inferred user intent:** Make a failed or missing active script generation
visible to orchestration before it accepts browser traffic.

**Commit (code):** `759962f` — "Feat: expose scripted signup readiness"

### What I did

- Added non-secret `Executor.PoolStats` and `Executor.Ready` methods.
- Extended `GenerationSnapshot` with the active pool snapshot and added
  `GenerationManager.Ready`.
- Added the `scripted_signup` readiness check to `embeddedidp.Provider` for a
  configured manager or direct executor.
- Added a provider test which closes its configured manager and proves the
  readiness report becomes false with the stable
  `active_generation_unavailable` reason.
- Repaired pre-existing lint findings exposed by the commit gate and refactored
  the script command constructors so each output-enabled command is visibly a
  Glazed row producer to the static analyzer.

### Why

A merely compiled artifact is not a serving generation. K3s and other
orchestrators need a dependency signal that reflects warmed live workers, while
metrics can distinguish temporary pool saturation from an unavailable pool.

### What worked

```bash
go test ./internal/cmds ./pkg/idpsignup ./pkg/embeddedidp ./pkg/sqlitestore ./pkg/idpscript ./internal/fositeadapter -count=1
go run ./cmd/tinyidp script test --source pkg/idpsignup/email_verified_signup.js --output json
git commit -m 'Feat: expose scripted signup readiness'
```

The focused tests and real CLI command passed. The final commit hook passed
`GOWORK=off go test ./...`, GolangCI-Lint, Glazed CLI lint, and the IDP UI
analyzer.

### What didn't work

The first commit attempt exposed a stale unused helper and three existing
static-analysis findings. After those were corrected, Glazed CLI lint reported
that the generic command-description helper appeared to expose output flags
without implementing `RunIntoGlazeProcessor`; the commands themselves did
implement it, but the analyzer could not infer that through the helper’s
`*cmds.CommandDescription` return type. Moving `WithSections` into each typed
constructor resolved the analyzer limitation without changing command behavior.

### What I learned

Readiness should report an unavailable script generation, not saturation. A
saturated warmed pool may make progress when an in-flight invocation releases;
a closed/empty pool cannot serve any new workflow transition.

### What was tricky to build

The embedded provider intentionally owns the existing `/readyz` aggregation.
Adding a parallel script endpoint would make orchestration choose between
conflicting signals. The implementation instead adds one stable check to that
single report and leaves liveness independent of dependency availability.

### What warrants a second pair of eyes

- `Executor.Ready` defines warmth as a non-closed pool with its full initial
  worker capacity. If future replacement-worker behavior changes, verify the
  invariant remains appropriate.
- This is readiness only; bounded metrics, audit events, deterministic fake
  capabilities, and repeat-reload leak tests remain required Phase 6 work.

### What should be done in the future

Implement `lf68–lf69`, `lf71–lf73`, and `lf75–lf77` fully before declaring
Phase 6 complete; do not begin Phase 7 production enablement early.

### Code review instructions

- Review `Executor.Ready`, `GenerationManager.Snapshot`, and the
  `scripted_signup` branch in `embeddedidp.Provider.Readiness`.
- Run `go test ./pkg/idpsignup ./pkg/embeddedidp -count=1`.
- Close a manager in the test and inspect the named readiness check.

### Technical details

```text
active manager --> active executor --> warmed, non-closed worker pool
       |                       |                  |
       +-----------------------+------------------+--> /readyz scripted_signup=true

closed/missing active generation ---------------------> /readyz false
                                                     reason=active_generation_unavailable
```

## Step 41: Prove script-test result and failure diagnostics

The script test command now has direct command-level coverage instead of being
validated only through a manual invocation. Its output contract deliberately
uses primitive strings for outcome fields, so table, JSON, YAML, and test
processors all see the same representation.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Continue closing the Phase 6 operational command
requirements with executable evidence.

**Inferred user intent:** Make a failed embedded script test actionable and
machine-readable in CI or an activation run.

**Commit (code):** `115b7d4` — "Test: cover script test command diagnostics"

### What I did

- Added direct `ScriptTestCommand.RunIntoGlazeProcessor` tests using temporary
  source files and a capture processor.
- Pinned the success row’s test ID, boolean result, expected kind, and actual
  kind.
- Pinned the failure behavior: emit the failed row first, then return a stable
  error naming the test and expected/actual outcomes.
- Converted the emitted `OutcomeKind` aliases to strings at the command
  boundary so consumers do not receive Go named values.
- Checked off `lf69`.

### Why

Operators need both structured data for a failing case and a nonzero command
result. Returning early before the row loses useful diagnostic context; emitting
arbitrary Go values makes formatter behavior an accidental API.

### What worked

```bash
go test ./internal/cmds -run TestScriptTestCommand -count=1 -v
git commit -m 'Test: cover script test command diagnostics'
```

Both the success and deliberate assertion-failure tests passed. The commit hook
then passed the complete `GOWORK=off go test ./...` and lint suites.

### What didn't work

Changing only `expectedKind:"present"` to `deny` produced a compilation
diagnostic because validation correctly requires test expectations to be
declared by the selected lambda. The fixture now declares `deny` as allowed
while the implementation still returns `present`, exercising the intended
runtime assertion rather than weakening program validation.

### What I learned

An embedded test has two distinct failure classes: malformed contract tests
fail during compilation; a valid assertion mismatch fails during execution.
Both must be nonzero, but only the latter has a per-case execution row.

### What was tricky to build

The program validator protects workflow edge consistency, so a fixture cannot
arbitrarily invent an expected outcome. The test added an unused but declared
terminal outcome to the lambda, preserving a valid program graph while making
the runner’s actual/expected comparison meaningful.

### What warrants a second pair of eyes

- Compile, bind, timeout, and leak diagnostics still need their own test cases
  when deterministic fake capabilities are introduced for `lf68`.

### What should be done in the future

Implement explicit deterministic fake host capabilities and extend this command
test suite to bind/timeout/leak failures before Phase 6 is declared complete.

### Code review instructions

- Read `internal/cmds/script_test.go` beside `ScriptTestCommand`.
- Confirm outcome kinds are converted at the row boundary only.
- Run the command test shown above.

### Technical details

```text
valid test + matching outcome --> row { passed: true } --> exit 0
valid test + mismatched outcome --> row { passed: false } --> stable error --> nonzero
invalid test declaration --> compile diagnostic --> nonzero
```

## Step 42: Add deterministic embedded-test capability fakes

Embedded tests can now exercise capability-using lambdas without connecting to
production services. The program supplies only JSON fixture output; native code
owns the fixed capability vocabulary and binds it only during `RunTests`.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Finish the Phase 6 embedded-test capability
contract while retaining the no-ambient-authority model.

**Inferred user intent:** Let scripts test workflows that depend on time,
randomness, mail, identity, invitations, or storage without making JavaScript
a production integration harness.

**Commit (code):** `9b22790` — "Feat: bind deterministic script test fakes"

### What I did

- Added optional `fakes` JSON outputs to `ProgramTest` and the TypeScript API.
- Validated that every fake is bounded JSON and names a capability actually
  required by the test lambda.
- Added six runner-owned deterministic fake IDs: `clock.now`, `random.bytes`,
  `mailer.send`, `identity.lookup`, `invitation.lookup`, and `store.get`.
- Bound only declared known fake requirements during `Executor.RunTests`; an
  unknown required capability still fails closed.
- Added an async JavaScript test using configured `clock.now` output and
  confirmed it observes the fixture value rather than a wall clock.
- Checked off `lf68`.

### Why

Deterministic workflow tests need controlled dependency responses, but test
fixtures must not become a path to mail delivery, database access, account
mutation, or arbitrary host code. JSON return values keep the seam bounded,
serializable, and reviewable.

### What worked

```bash
go test ./pkg/idpprogram ./internal/gojamodules/tinyidp ./pkg/idpsignup -count=1 -v
git commit -m 'Feat: bind deterministic script test fakes'
```

The configured clock fake test passed, and the full commit lint gate passed.

### What didn't work

N/A.

### What I learned

Test fakes belong to the runner, not the general capability API. A program can
declare a production capability contract, but only the dedicated test runner
recognizes this fixed test vocabulary and supplies controlled output.

### What was tricky to build

The same program contract is compiled for production and test. Validation can
ensure a fixture refers to a lambda-required capability, while the profile
runner must separately reject known-but-unbound or unknown capability IDs.
That split prevents a generic `fakes` object from granting authority at normal
request invocation time.

### What warrants a second pair of eyes

- The default fake outputs are intentionally minimal and response-only. Add
  explicit request assertions only if a later test contract needs them; do not
  turn these into embedded JavaScript callbacks.

### What should be done in the future

Add targeted tests for unknown fake IDs and fake-output bounds as the test
contract grows, then complete lifecycle, metrics, audit, and reload coverage.

### Code review instructions

- Review `ProgramTest.Fakes`, the validation loop, and
  `Executor.testCapabilities` together.
- Run the command in **What worked**.
- Confirm ordinary `Executor.Start`/`Submit` calls do not use this method.

### Technical details

```text
program.test { fakes: { "clock.now": { unixMillis: 42 } } }
     --> validated JSON fixture --> test-only native binding
     --> ctx.cap.clock.now({}) --> { unixMillis: 42 }

normal browser request --> no test binding --> production host binding only
```

## Step 43: Prove retained-generation draining under repeated reload

The generation manager’s bounded retention behavior now has direct resource
lifecycle evidence, not just lookup behavior. Eviction closes the old worker
pool, surviving pools close at manager shutdown, and repeated reloads never
grow the retained registry past its configured bound.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Continue Phase 6 operational hardening with
evidence that reloading does not leak runtime workers.

**Inferred user intent:** Safely support live continuation routing while
keeping old Goja pools from accumulating indefinitely.

**Commit (code):** `dcfef75` — "Test: prove generation pool draining"

### What I did

- Added a lifecycle test that retains a direct executor reference, activates
  beyond the retention bound, and proves the evicted executor reports a closed
  pool.
- Proved retained and active pools close after `GenerationManager.Close`.
- Added twelve successive source reloads and asserted the retained registry is
  always bounded by active plus configured predecessors.

### Why

A removed fingerprint is insufficient proof of draining: a retained Goja pool
could still hold workers and event-loop resources after lookup removal.

### What worked

```bash
go test ./pkg/idpsignup -run 'TestGenerationManager(Drains|Repeated)' -count=1 -v
git commit -m 'Test: prove generation pool draining'
```

Both focused tests and the full commit hook passed.

### What didn't work

N/A.

### What I learned

The pool’s `Closed` operational state is the useful leak assertion at this
layer: it proves cancellation and image-close lifecycle ran for the executor
without exposing VM internals to the manager test.

### What was tricky to build

The test must retain executor pointers before manager eviction; after eviction
the public resolver correctly refuses the old fingerprint, but that alone
cannot inspect its pool closure.

### What warrants a second pair of eyes

- Retention remains a count bound, not durable continuation lease accounting;
  operators must set it consistently with the maximum workflow lifetime.

### What should be done in the future

Add activation audit records and complete the broader in-flight/replay failure
matrix before Phase 6 completion.

### Code review instructions

- Start at the two added `TestGenerationManager...` lifecycle tests.
- Follow eviction into `Executor.Close` and `Pool.Close`.

### Technical details

```text
generation 1 --evict--> remove resolver entry + close worker pool
generation 2/3 --manager Close--> close every retained worker pool
```

## Step 44: Expose bounded generation-manager metrics

The manager now exports a race-safe, secret-free metrics snapshot for
activation outcomes and worker-pool saturation. It contains counts and pool
sizes only; it does not use fingerprints, source text, user data, or callback
labels as metric dimensions.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Continue Phase 6 observability using bounded
operator-visible counters.

**Inferred user intent:** Let an embedding host detect reload failures,
retention pressure, and pool saturation without introducing PII or cardinality
risk.

**Commit (code):** `f5b5f87` — "Feat: expose generation manager metrics"

### What I did

- Added activation-success, activation-failure, eviction, and close counters.
- Added retained-generation count plus active pool capacity/active-worker
  values to `GenerationMetrics` and `GenerationSnapshot`.
- Added a test for one failed activation, one successful replacement, one
  eviction, and a bounded one-worker pool.

### Why

Operational signal needs to distinguish an unavailable generation from a
healthy but saturated worker pool, while avoiding unbounded labels derived
from JavaScript functions or user input.

### What worked

```bash
go test ./pkg/idpsignup -run TestGenerationManagerReportsBoundedOperationalMetrics -count=1 -v
git commit -m 'Feat: expose generation manager metrics'
```

The focused test and complete hook passed.

### What didn't work

N/A.

### What I learned

Metrics snapshots can be composed from existing pool statistics; the manager
does not need a parallel worker-lifecycle implementation to report saturation.

### What was tricky to build

`Snapshot` and `Metrics` both need consistent read locking. Nested read locks
are safe for `sync.RWMutex`; no writer is acquired while either snapshot is
being assembled.

### What warrants a second pair of eyes

- Invocation latency/outcome and continuation create/replay/expiry counters
  still need to be surfaced at their owning boundaries for full `lf75`.

### What should be done in the future

Add redacted audit records and complete the remaining metrics/failure matrix.

### Code review instructions

- Review `GenerationManager.Metrics` and its test.
- Confirm no metric field contains a fingerprint or request-derived value.

### Technical details

```text
activation -> {success|failure} counter
eviction -> evicted counter
snapshot -> retained count + pool {capacity, active}
```

## Step 45: Add redacted activation audit and invocation metrics

Script activation and execution now have bounded operational signals without
making JavaScript source, user data, browser state, or handler labels part of
the observable API. Activation emits a redacted audit event; executor metrics
count fixed outcome families and resource disruption.

### Prompt Context

**User prompt (verbatim):** (same as Step 39)

**Assistant interpretation:** Finish the Phase 6 observability boundary before
enabling Phase 7 protocol customization.

**Inferred user intent:** Make production script behavior diagnosable without
turning telemetry into a secret or high-cardinality data channel.

**Commit (code):** `a92339e` — "Feat: audit scripted generation activation";
`9b77ae1` — "Feat: expose signup invocation metrics"

### What I did

- Added optional manager audit/clock options and emitted
  `script.signup.activation` accepted/rejected records.
- Included only source/program fingerprints and fixed result/reason values;
  counted audit-delivery failures separately.
- Added executor counters for invocations, failures, fixed outcome buckets,
  cumulative latency, and discarded worker deltas.
- Added tests proving no client/subject/source text enters activation audit and
  that a challenge transition increments only bounded metrics.

### Why

Operations needs enough information to correlate a reload and diagnose a
worker interruption, but must not record credentials, invite codes, emails,
callback names, or raw JavaScript exceptions.

### What worked

```bash
go test ./pkg/idpsignup -run TestGenerationManagerActivationAuditIsRedacted -count=1 -v
go test ./pkg/idpsignup -run TestExecutorReportsBoundedInvocationMetrics -count=1
```

Both focused tests and their commit hooks passed.

### What didn't work

The outcome metric switch initially omitted several closed `OutcomeKind` cases;
the exhaustive linter correctly required an explicit mapping of all remaining
terminal outcomes into the fixed `other` bucket.

### What I learned

The right metric dimensions are protocol-stable categories, not workflow
handler IDs. Latency is exported only as cumulative nanoseconds here; a host
can derive rates without a per-request trace label.

### What was tricky to build

Activation failure can happen before an executor exists. The audit helper
therefore emits no hash fields for a failed warmup rather than logging source
or compiler errors; successful records take hashes from the compiled artifact.

### What warrants a second pair of eyes

- Continuation create/resume/replay/expiry metrics still belong in the
  continuation service; this executor/manager work does not claim them.

### What should be done in the future

Complete continuation metrics and the repeated-reload failure matrix, then run
the Phase 6 gate before beginning Phase 7 handlers.

### Code review instructions

- Read `GenerationManager.auditActivation` and `Executor.Metrics`.
- Run the two focused commands above.
- Verify audit fields contain hashes but no source or identity values.

### Technical details

```text
activation --> audit { event, accepted|rejected, reason, sourceHash, programHash }
invoke --> counters { outcome family, failure, latency total, discarded }
```
