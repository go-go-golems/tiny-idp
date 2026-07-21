---
Title: Investigation diary
Ticket: TINYIDP-MULTIAPP-THEMES-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - architecture
    - operations
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve_production.go
      Note: Evidence for one-client host gap
    - Path: repo://internal/productionconfig/clients.go
      Note: Strict versioned multi-client production catalog
    - Path: repo://internal/productionui/catalog.go
      Note: Mounted theme catalog validation and allowlisted assets
    - Path: repo://internal/productionui/renderer.go
      Note: Per-client interaction and workflow rendering
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk
      Note: Evidence for current GitOps and trusted-Traefik pattern
ExternalSources: []
Summary: Chronological evidence and decisions for the shared TinyIDP themes and second-application design.
LastUpdated: 2026-07-21T12:16:07-04:00
WhatFor: Record the evidence used to design the first two-client TinyIDP deployment and operator research branch.
WhenToUse: Read before continuing implementation or reconsidering the GitOps-only versus operator decision.
---



# Diary

## Goal

This diary records the evidence-first analysis behind the shared TinyIDP theme
and second-application design. It is written so the next implementer can tell
which facts were observed in source and manifests, which conclusions follow
from them, and which decisions remain proposed.

## Context

The starting point was a deployed standalone TinyIDP and MessageDesk pair on
k3s. The user asked to move CSS out of the TinyIDP binary while allowing a
shared IdP to present different looks for two separately deployed applications.
They then asked to explore a custom Kubernetes operator as a research branch.

## Quick Reference

## Step 1: Map the current provider and deployment boundaries

The first pass established whether a second app is blocked by the OAuth
provider or only by the production embedding. The answer is important because
it changes the work from a protocol redesign to a configuration-and-deployment
design. The code shows that the reusable bootstrap API accepts multiple client
specifications, while the production command hard-codes one MessageDesk
client and renderer.

### Prompt Context

**User prompt (verbatim):** "ok, do that. that way we can have message-desk provide CSS to style idp, but have the app deployment actually set it up safely. Createa new docmgr ticket for that and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


In fact, the ticket should be geared towards hosting now a second application next to message-desk, so that tiny-idp can serve two apps that have different looks and needs.

I'm not sure if this is already possible either. It would be a separate k3s deployment too, as a different example and more complex setup"

**Assistant interpretation:** Create an intern-ready research/design ticket for a shared two-application TinyIDP deployment, with GitOps-provided themes rather than CSS embedded in the IdP image.

**Inferred user intent:** Retain robust IdP-owned authentication boundaries while allowing different app identities and independently deployable applications.

### What I did

- Created ticket `TINYIDP-MULTIAPP-THEMES-001` with a design document and diary.
- Inspected `pkg/embeddedidp/bootstrap.go`, `internal/cmds/serve_production.go`, `pkg/idpui/types.go`, `internal/fositeadapter/rendering.go`, and the deployed MessageDesk Kustomize resources.
- Compared the reusable multi-client bootstrap boundary against the current one-client CLI and embedded CSS route.

### Why

- The design must not assume that a client can safely select its own browser assets.
- The second app must preserve the existing trusted-Traefik topology rather
  than taking a shortcut directly to the IdP Service.

### What worked

- `embeddedidp.Bootstrap` takes multiple client specs, deduplicates them, and
  rejects conflicts with durable registration state.
- `InteractionPage.ClientID` is a public, provider-created input suitable for
  theme selection; it is not an opaque interaction handle or credential.
- The existing CSP permits same-origin CSS and rejects all other default
  sources.
- The live GitOps deployment already demonstrates read-only ConfigMap mounts,
  Vault secrets, content-triggered rollout annotations, Argo sync waves, and
  a restricted Traefik backchannel policy.

### What didn't work

- `docmgr ticket list --limit 100` failed because this CLI version does not
  support `--limit`: `Error: unknown flag: --limit`. I used ticket-targeted
  listing and document search instead.

### What I learned

- The provider is multi-client capable today; the gap is the production host
  configuration and deployment model.
- The Goja signup start input contains `clientId`, so future per-app signup
  policy can remain inside IdP-owned reviewed code instead of leaking into CSS
  configuration.
- The current issuer name is MessageDesk-specific, but changing an issuer is a
  separate identity migration and should not be bundled with the first
  second-app rollout.

### What was tricky to build

- The deployment uses the canonical public IdP hostname for application
  backchannels but resolves it to Traefik's ClusterIP through `hostAliases`.
  Direct Service traffic would fail the TinyIDP trusted-proxy boundary. The
  design therefore repeats the canonical-hostname-through-Traefik pattern for
  every relying party, not merely its ingress manifest.
- CSS must be supplied by application ownership without being supplied by the
  application at runtime. The solution is a reviewed GitOps copy mounted into
  the IdP, which separates source ownership from runtime authority.

### What warrants a second pair of eyes

- The exact second example, public hostname, data needs, and per-app signup
  policy are intentionally unresolved and must be selected before Phase 4.
- Review the future client disable/revocation contract before a catalog entry
  is treated as a lifecycle authority.

### What should be done in the future

- Implement Phases 1–5 in `tasks.md` after choosing the second app.
- Revisit the Kubernetes operator only after the GitOps-only two-app design
  has been operated.

### Code review instructions

- Start at `embeddedidp.Bootstrap` and `idpui.InteractionPage`, then inspect
  the production command's hard-coded client and renderer.
- Compare the TinyIDP and MessageDesk deployments, ingress, and network policy
  before creating the second application's analogous resources.
- Validate future work with `go test ./...`, `kustomize build`, Argo health,
  and one public PKCE flow per client.

### Technical details

```text
validated OAuth client_id -> InteractionPage.ClientID -> IdP theme catalog
  -> same-origin /static/themes/<name>.css

The application never provides an asset URL in /authorize.
```

## Step 2: Add the Kubernetes-operator research branch

The follow-up requested an exploration of a custom operator. The design treats
this as a separate branch because an operator changes the control plane rather
than merely simplifying YAML. It should be justified by stable repeated
patterns, not by the first occurrence of repetition.

### Prompt Context

**User prompt (verbatim):** "also, explore the idea that we could provide custom k8s operator to manage tiny-idp and make all of this a bit easier too, as a side branch of the research."

**Assistant interpretation:** Add a bounded comparison and a minimal future
operator API to the ticket without making an operator mandatory for the
two-application delivery.

**Inferred user intent:** Learn whether a higher-level deployment abstraction
will reduce future application-onboarding work without prematurely adding a
privileged platform component.

### What I did

- Added a dedicated operator chapter, minimal `TinyIDP` CR sketch,
  `TinyIDPClient` research question, RBAC boundary, and adoption threshold to
  the design document.
- Added operator research tasks separately from the implementation phases.

### Why

- A controller could generate client catalogs, ConfigMaps, and common IdP
  resources, but it must also own reconciliation, status, leader election,
  CRD versions, and privileged RBAC.

### What worked

- The current Kustomize deployment makes the repeated unit of configuration
  concrete: catalog, themes, Deployment, Service, Ingress, and policy.
- This evidence is enough to design a small hypothetical CR without claiming
  that an operator is presently necessary.

### What didn't work

- No implementation experiment was run. Writing an operator before selecting
  the second app would make its CRD speculative and expand scope beyond this
  research ticket.

### What I learned

- Cross-namespace `TinyIDPClient` objects are the difficult part of an
  operator: they require an explicit policy for who may claim client IDs,
  callback hosts, and theme names.
- GitOps already reconciles desired resources. An operator becomes valuable
  only when it captures stable, policy-rich behavior that Kustomize cannot
  express safely or ergonomically.

### What was tricky to build

- The tempting operator design grants it access to Secrets, Vault, Ingress,
  and central identity configuration at once. The proposed research API avoids
  that by limiting initial ownership to a generated catalog and ordinary
  workload objects.

### What warrants a second pair of eyes

- Any future CRD that accepts cross-namespace references or handles client
  deletion needs security and lifecycle review before implementation.

### What should be done in the future

- Reassess after three applications and at least one complete GitOps rollback,
  following the threshold in the primary guide.

### Code review instructions

- Review the operator section against the real Kustomize objects it proposes
  to replace. Reject controller authority that is not required for a concrete
  reconciliation invariant.

### Technical details

```text
TinyIDP CR -> controller validates desired client/theme catalog
             -> ConfigMap + Deployment + status conditions

Not in the first controller: arbitrary URLs, automatic client deletion,
cross-namespace theme reads, Vault policy creation, or DNS management.
```

## Step 3: Validate and deliver the design bundle

The completed ticket was checked as structured docmgr content before delivery,
not merely as Markdown that happens to render. The validation confirmed the
frontmatter and related-file records, and the bundle upload produced one
reMarkable PDF with a table of contents for the index, primary guide, tasks,
and diary.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the ticket deliverables and upload them to reMarkable.

**Inferred user intent:** Receive a durable, readable implementation guide that can be reviewed away from the development machine.

### What I did

- Ran frontmatter validation on the index, design document, and diary.
- Ran `docmgr doctor --ticket TINYIDP-MULTIAPP-THEMES-001 --stale-after 30`.
- Ran a dry-run bundle upload, then uploaded the ticket bundle with
  `remarquee upload bundle`.

### Why

- The ticket must remain navigable and structurally valid as implementation
  continues, and the reMarkable copy must include the task checklist and the
  evidence record rather than an isolated design chapter.

### What worked

- All frontmatter validations reported `Frontmatter OK`.
- Doctor reported one ticket with `✅ All checks passed`.
- Dry run selected `/ai/2026/07/21/TINYIDP-MULTIAPP-THEMES-001` and included
  the four intended Markdown documents.
- Upload completed with: `OK: uploaded TinyIDP Shared Themes and Second App
  Guide.pdf -> /ai/2026/07/21/TINYIDP-MULTIAPP-THEMES-001`.

### What didn't work

- N/A.

### What I learned

- The document bundle is compact enough to be read as one guide while still
  retaining the implementation checklist and research diary.

### What was tricky to build

- N/A; the prior design work made the bundle boundaries clear.

### What warrants a second pair of eyes

- Review the unresolved second-application choice and issuer-migration timing
  before beginning the Phase 4 manifests.

### What should be done in the future

- Begin Phase 1 after choosing whether to keep the second app as a generic
  placeholder or name a concrete example.

### Code review instructions

- Read the primary guide's decision records before reviewing the task list.
- Confirm any implementation PR updates both `tasks.md` and this diary with
  command/test evidence.

### Technical details

```text
Bundle destination:
/ai/2026/07/21/TINYIDP-MULTIAPP-THEMES-001/
  TinyIDP Shared Themes and Second App Guide.pdf
```

## Step 4: Implement the production client and theme boundaries

This step turns the first two design phases into executable code. The
production host no longer knows that MessageDesk is special. It accepts a
strict client catalog and a strict theme catalog, loads both before opening its
listener, and uses the provider-validated client ID to choose presentation for
both login/consent pages and scripted-signup continuation pages.

### Prompt Context

**User prompt (verbatim):** "implement and deploy the two app example setup. use existing apps if possible, this is more abut configuration and deployment not the actual apps themselves."

**Assistant interpretation:** Implement the ticket through public production
deployment, reusing MessageDesk and the existing `goja-auth-host-demo` instead
of creating another application.

**Inferred user intent:** Concentrate engineering effort on the shared IdP,
GitOps configuration, isolation, and rollout rather than application feature
development.

### What I did

- Confirmed the active goal with the goal tracker and selected the already
  deployed `goja-auth-host-demo` as the second relying party.
- Replaced the MessageDesk-specific production flag with required
  `--clients-file`, `--theme-dir`, and `--theme-catalog-file` inputs.
- Added `internal/productionconfig`, which strictly decodes a bounded,
  versioned browser-client catalog and produces existing
  `embeddedidp.ClientSpec` values.
- Added `internal/productionui`, which preloads reviewed CSS, validates the
  catalog and its client mappings, renders login/consent/signup pages, and
  serves only exact same-origin routes below `/static/themes/`.
- Carried `InteractionRecord.ClientID` into `idpui.WorkflowPage.ClientID`, so a
  signup continuation cannot lose the original client presentation context.
- Added checked-in production example catalogs and updated the systemd,
  two-process, Docker smoke, and production image-flow examples to use the new
  explicit contract.
- Added tests for two clients, unsafe URLs and scopes, strict/bounded JSON,
  missing and escaping theme declarations, duplicate JSON theme keys,
  per-client interaction/workflow selection, HTML escaping, and asset routes.
- Committed the implementation as `7d8a19d` (`Add production client and theme
  catalogs`).

### Why

- TinyIDP's reusable bootstrap layer was already multi-client. The hard-coded
  single-client production command was the actual blocker.
- The browser cannot be allowed to supply a stylesheet URL: that would turn an
  OAuth request parameter into executable presentation authority. The safe
  selector is the registered client ID, and the safe assets are immutable
  files reviewed and mounted at deployment time.
- Signup is a continuation-based workflow. Adding the client ID to its public
  renderer model preserves the same trusted selection seam across requests
  without retaining a suspended JavaScript VM.

### What worked

- `go build ./...` passed.
- `go test ./...` passed, including the real two-process signup lifecycle.
- The pre-commit hook independently ran the full suite, `golangci-lint`,
  `glazed-lint`, and the project-specific IDP UI analyzer; all passed.
- Existing `embeddedidp.Bootstrap` conflict tests still prove that a catalog
  cannot silently widen an existing durable registration.
- Strict duplicate-key detection prevents `encoding/json` map overwrite
  behavior from making a reviewed theme declaration ambiguous.

### What didn't work

- The first patch against
  `internal/fositeadapter/workflow_rendering_internal_test.go` used stale
  surrounding text and failed with:
  `apply_patch verification failed: Failed to find expected lines ...`.
  No partial change was made; I read the current fixture and applied the
  one-line client ID addition against its actual layout.
- An initial inspection command used the old `2026/07/07` ticket path and
  `sed` reported `No such file or directory`. Repository search established
  that the production deployment ticket lives under `2026/07/18`, after which
  all three harnesses were updated.
- One `rg` invocation placed `--glob` after the pattern and treated the options
  as file paths. This was an inspection-only failure and did not alter files.

### What I learned

- The existing second app already has an independent namespace, Deployment,
  Service, Ingress, and PostgreSQL state, which makes it the right reuse target.
- Its currently deployed image predates public-client PKCE support from merged
  go-go-goja PR #98. Deployment configuration can be reused, but the image must
  be refreshed before repointing it to TinyIDP.
- A theme catalog must cover scripted signup as well as ordinary interaction
  rendering; otherwise a branded login page falls back to an unstyled signup
  page during the most important onboarding path.

### What was tricky to build

- Kubernetes ConfigMap projected files are implemented with symlinks. The
  loader therefore constrains catalog references to CSS basenames and reads
  resolved regular files, instead of rejecting all symlinks and thereby
  rejecting normal Kubernetes mounts.
- Unknown client IDs use the declared default theme defensively, but mappings
  themselves may name only registered clients. Protocol validation still owns
  whether the client exists; presentation selection does not become a second
  client registry.

### What warrants a second pair of eyes

- Review the new HTML templates against the previous MessageDesk presentation
  for semantic parity of every provider-owned hidden field and submit action.
- Review the exact goja-auth callback/logout URLs in the GitOps catalog before
  merging the deployment PR.

### What should be done in the future

- Package the MessageDesk and goja-auth themes with Kustomize, register both
  clients, and repoint the existing goja-auth deployment.
- Publish an image containing this commit and complete independent public PKCE
  flows through both applications.

### Code review instructions

- Start with `internal/productionconfig/clients.go` and its tests.
- Follow `runProductionHost` through bootstrap, UI construction, and the
  `/static/themes/` route.
- Trace signup from `workflowPage` into `productionui.RenderWorkflow` and
  verify that only `ClientID` affects theme selection.
- Run `go test ./...`; do not validate only the new packages because the
  two-process harness is what proves the changed CLI contract.

### Technical details

```text
clients.json                         themes.json + *.css
     |                                      |
     v                                      v
productionconfig.ClientCatalog       productionui.Catalog
     |                                      |
     +--> embeddedidp.Bootstrap              +--> exact static assets
                                                |
validated OAuth client_id ---------------------+--> renderer selection
```

## Step 5: Make progress and evidence persistent at short intervals

### Prompt Context

**User prompt (verbatim):** "keep a detailed diary as you work, update your goal or so, commit at appropriate intervals (read diary skill)"

**Assistant interpretation:** Keep the existing deployment goal active, follow
the diary skill's exact step structure, and create reviewable commits at phase
boundaries rather than accumulating one opaque change.

**Inferred user intent:** Make a long autonomous production rollout auditable
and recoverable without interrupting it for routine status questions.

### What I did

- Reconfirmed that the active goal remains the complete two-app production
  deployment; it is not complete merely because the source implementation
  passes tests.
- Applied the diary skill's required headings and recorded exact prompt,
  command, failure, test, and commit evidence in Step 4.
- Migrated the task list to stable IDs and checked all Phase 1 and Phase 2
  tasks after commit `7d8a19d` and successful validation.
- Related the new configuration and renderer source files to this diary and
  added the implementation checkpoint to the ticket changelog.

### Why

- The next work crosses repositories and delivery systems. A source commit is
  the clean rollback boundary before GitOps manifests and image publication.

### What worked

- The first implementation checkpoint is self-contained and its hooks passed.
- `docmgr task migrate` stamped 30 stable task IDs, allowing later checks to
  remain precise even if prose or ordering changes.

### What didn't work

- N/A.

### What I learned

- The ticket began as a design deliverable, so task IDs had not yet been
  stamped. Migrating them before implementation tracking avoided fragile
  numeric checkbox references.

### What was tricky to build

- The commit hash can only be recorded after the implementation commit exists;
  this diary update is therefore a separate documentation checkpoint.

### What warrants a second pair of eyes

- Confirm Phase 1 and Phase 2 checkbox evidence against `7d8a19d` during PR
  review.

### What should be done in the future

- Continue with the GitOps and second-app phases without closing the goal until
  Argo health and both public flows are captured.

### Code review instructions

- Review the preceding implementation commit first, then this ticket update;
  the separation is intentional.

### Technical details

```text
source boundary: 7d8a19d
validation: go build ./... && go test ./...
hook validation: golangci-lint + glazed-lint + IDP UI analyzer + go test ./...
```

## Usage Examples

Use the primary design doc for implementation order and `tasks.md` as the
authoritative work checklist. Add a new diary step immediately after each
implementation phase, preserving exact commands, test results, and any failed
assumptions.

## Related

- [Primary guide](../design-doc/01-shared-tiny-idp-theme-assets-and-a-second-application-on-k3s-analysis-design-and-implementation-guide.md)
- [Task list](../tasks.md)
- Existing production deployment ticket:
  `TINYIDP-K3S-MSGDESK-PROD-001`
