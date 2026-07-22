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
