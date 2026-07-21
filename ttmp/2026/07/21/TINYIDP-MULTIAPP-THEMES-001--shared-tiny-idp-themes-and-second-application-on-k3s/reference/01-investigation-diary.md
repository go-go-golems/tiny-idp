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
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk
      Note: Evidence for current GitOps and trusted-Traefik pattern
ExternalSources: []
Summary: Chronological evidence and decisions for the shared TinyIDP themes and second-application design.
LastUpdated: 2026-07-21T10:52:49.653859502-04:00
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
