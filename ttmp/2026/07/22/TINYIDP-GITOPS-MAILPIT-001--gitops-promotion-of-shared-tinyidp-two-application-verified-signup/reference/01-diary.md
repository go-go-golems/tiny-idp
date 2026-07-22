---
Title: Diary
Ticket: TINYIDP-GITOPS-MAILPIT-001
Status: active
Topics:
    - identity
    - oidc
    - security
    - operations
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/README.md
      Note: Operator acceptance runbook
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/mailpit.yaml
      Note: Private mail-catcher workload and service (commit 78308e6)
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/signup-program-configmap.yaml
      Note: Baseline deployed signup program
ExternalSources: []
Summary: Chronological implementation and validation record for the GitOps Mailpit and shared verified-signup rollout.
LastUpdated: 2026-07-22T19:09:53.547989483-04:00
WhatFor: Preserve commands, failures, decisions, commits, and review instructions for this rollout.
WhenToUse: Before resuming or reviewing TINYIDP-GITOPS-MAILPIT-001.
---



# Diary

## Goal

Record the promotion of the local two-application verified-signup topology into
the live Argo CD deployment, including security boundaries and operational
acceptance evidence.

## Step 1: Establish the live baseline and deployment boundary

I inspected the live cluster and compared it with the GitOps source and local
Compose acceptance stack. The public topology already contains Message Desk,
Goja Auth, and a shared TinyIDP, but production is still running the older
immediate-account-creation program and has no Mailpit or email-challenge flags.

This established that the task is a coordinated workflow promotion rather than
a new application topology. It also fixed the security boundary: Mailpit is a
private operator tool reached by port-forward, never a public application.

### Prompt Context

**User prompt (verbatim):** "ok, setup a new ticket to do that, add concise design doc, and then implement. Keep a detailed diary as you work, commit at appropriate intervals."

**Assistant interpretation:** Create a tracked design and implementation ticket, then promote the locally tested two-app invitation and email-verification stack into the live GitOps deployment.

**Inferred user intent:** Make the real deployment exercise the same robust signup behavior already proven locally, while retaining an auditable implementation record.

### What I did
- Queried the live `tiny-message-desk` and `goja-auth-host-demo` namespaces.
- Verified all three public HTTPS endpoints return HTTP 200.
- Inspected the live TinyIDP args and signup-program ConfigMap.
- Searched the GitOps tree and confirmed no Mailpit or SMTP configuration exists.
- Created this ticket, tasks, design document, and diary.

### Why
- Implementation must start from deployed reality rather than assumptions from the local Compose stack.

### What worked
- Read-only cluster access exposed the exact image, arguments, services, ingresses, and running program.

### What didn't work
- The first `apply_patch` attempt failed before changing files with `invalid hunk`; correcting every added-line prefix fixed the patch structure.

### What I learned
- Two-app OIDC routing is already live; only admission, email verification, and private delivery are missing.
- The live program currently commits an account directly with no challenge or invitation provider.

### What was tricky to build
- The local stack and live cluster were at different feature stages despite sharing hostnames and clients. Separating topology completion from workflow completion prevented an unnecessary redesign.

### What warrants a second pair of eyes
- Vault key naming and NetworkPolicy selectors must match live operator and namespace conventions exactly.

### What should be done in the future
- Replace Mailpit with a reviewed outbound SMTP transport after live workflow acceptance.

### Code review instructions
- Start with the design document, then compare `gitops/kustomize/tiny-message-desk` against `examples/tinyidp-shared-two-apps/compose.yaml`.
- Re-run read-only cluster inspection and public endpoint probes after Argo synchronization.

### Technical details
- Baseline image: `ghcr.io/go-go-golems/tiny-idp:sha-bdbad44`.
- Baseline program: `open-signup`, two effects, no challenge or provider.
- Local Mailpit version: `axllent/mailpit:v1.30.5`.

## Step 2: Implement the private delivery and verified-signup resources

I promoted the reviewed shared signup program into the GitOps ConfigMap, added
the two durable key files and SMTP flags to TinyIDP, and introduced an ephemeral
Mailpit Deployment and ClusterIP Service. NetworkPolicy permits only TinyIDP to
reach SMTP and the authenticated kubelet port-forward path to reach the UI.

The existing infrastructure branch had already been merged and deleted while
new image automation commits landed on `main`. I committed the isolated change,
fetched current state, and rebased cleanly onto `origin/main`; the rebased
infrastructure commit is `78308e6`.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement the design through immutable GitOps and prepare a safe operator acceptance path.

**Inferred user intent:** Reach a deployable, reviewable live configuration rather than leave the ticket at design stage.

**Commit (code):** `78308e6` — "Deploy private Mailpit verified signup flow"

### What I did
- Added `mailpit.yaml` with pinned Mailpit, restrictive security context, bounded retention, health probes, ephemeral storage, and no public ingress.
- Added TinyIDP SMTP egress and Mailpit ingress NetworkPolicies.
- Replaced the old two-step program with the reviewed open/invited, email-verified program.
- Added owner-only invitation and challenge key preparation and TinyIDP production flags.
- Seeded independent 64-byte `invitation-lookup-key` and `email-challenge-key` values into the existing Vault path without printing their values.
- Added the operator port-forward, invitation issue, and acceptance matrix to the infrastructure README.

### Why
- Program startup validation requires durable provider and email services to be present together.
- Mailpit must be useful to an operator without becoming a public verification-code disclosure surface.

### What worked
- `kubectl kustomize` rendered the full application.
- `kubectl apply --dry-run=client` accepted every rendered object.
- `tinyidp script validate --source ...` returned status `valid` and source fingerprint `1b320c3d...`.
- All four declarative program tests passed.
- `docker manifest inspect ghcr.io/go-go-golems/tiny-idp:sha-bdbad44` verified the published merge image.
- The infrastructure commit rebased onto current `origin/main` without conflicts.

### What didn't work
- `tinyidp script validate --file ...` failed with `Error: unknown flag: --file`; the documented flag is `--source`.
- `docker manifest inspect ...:sha-5aa8c4e` returned `manifest unknown`. PR image jobs intentionally set `push_image: false`; the merged `main` image is `sha-bdbad44`.

### What I learned
- A successful pull-request image workflow proves the build but does not publish a deployable tag.
- The Vault token policy currently allows the CLI patch through a read-modify-write fallback and warns that explicit HTTP PATCH capability should be added later.

### What was tricky to build
- The rollout has an ordering invariant: Vault data and Mailpit must exist before TinyIDP activates a program declaring challenge and invitation services. Argo sync waves, fail-closed init checks, and readiness keep partial configuration from serving traffic.
- Mailpit port-forward traffic is terminated by the node kubelet, so its UI policy admits only the current single node on TCP 8025 while SMTP uses pod selectors.

### What warrants a second pair of eyes
- Confirm the Mailpit image operates correctly under UID/GID 1000 and its `/livez` endpoint under the cluster restricted profile.
- Confirm the node-source assumption for port-forward with the active CNI; if it differs, refine only the UI rule rather than adding an Ingress.
- Review whether the Vault role should gain native `patch` capability.

### What should be done in the future
- Replace Mailpit with reviewed outbound SMTP and remove the operator relay step.

### Code review instructions
- Review `mailpit.yaml`, then the TinyIDP init/args in `deployment.yaml`, then the first NetworkPolicy document.
- Run `kubectl kustomize gitops/kustomize/tiny-message-desk` and client dry-run.
- Validate the source copy with `tinyidp script validate` and `tinyidp script test`.

### Technical details
- Vault path: `kv/apps/tiny-message-desk/prod/idp`.
- Published image: `ghcr.io/go-go-golems/tiny-idp:sha-bdbad44`.
- Mailpit ports: SMTP 1025, operator HTTP 8025.
