---
Title: Production deployment implementation diary
Ticket: TINYIDP-K3S-MSGDESK-PROD-001
Status: active
Topics:
    - auth
    - identity
    - oidc
    - security
    - operations
    - docker
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-deployment-pipeline.md
      Note: Canonical source-to-GHCR-to-GitOps-to-Argo release path
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/argocd-app-setup.md
      Note: Argo project, sync-wave, and first-bootstrap requirements
    - Path: repo://ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/tasks.md
      Note: Authoritative phased production task ledger
ExternalSources: []
Summary: Chronological implementation, release, rollout, failure, and recovery evidence for the standalone Tiny-IDP and Message Desk production deployment.
LastUpdated: 2026-07-18T20:32:21.050406937-04:00
WhatFor: Preserve exact decisions, commands, failures, commits, review instructions, and production receipts across the multi-repository rollout.
WhenToUse: Read before resuming the ticket, reviewing a checkpoint, changing production, or executing rollback/recovery.
---


# Production deployment implementation diary

## Goal

This diary records the complete path from the existing Tiny-IDP review branch
to a public, verified, recoverable Tiny-IDP and Message Desk deployment on the
Hetzner k3s cluster. It covers both repositories, external writes, production
receipts, failures, and the reasoning behind security-sensitive decisions.

## Step 1: Establish the production project and inspect reality

The first step converted the broad instruction to “push it all the way to prod”
into a 43-task gated delivery plan. I refreshed the source and GitOps remotes,
verified that the implementation branches were clean, created a dedicated
docmgr ticket, and inspected the actual cluster through its Tailscale
kubeconfig. This established that the target is a single-node k3s cluster with
Traefik, cert-manager, Argo CD, Vault Secrets Operator, local-path
`WaitForFirstConsumer` storage, and an explicit prod-apps namespace allowlist.

The direct SSH inventory did not proceed because the recorded host key for the
older IP differs from the key presented today. I preserved strict host checking
and used the existing authenticated Kubernetes control-plane path instead. No
known-host entry was deleted and no SSH security check was bypassed.

### Prompt Context

**User prompt (verbatim):** “Create a docmgr ticket, and add detailed phases + tasks for each phase in the ticket. Then implement it step by step, committing at appropriate intervals, keeping a detailed diary. Push it all the way to prod.”

**Assistant interpretation:** Create a durable implementation project, implement the standalone provider-owned-signup deployment in phased source and GitOps commits, merge and publish both repositories, operate the production rollout, and stop short of completion until browser and recovery evidence pass.

**Inferred user intent:** Turn the accumulated Tiny-IDP research and parallel authentication work into one small real product that users can join and use, while retaining enough evidence to review and operate it safely.

### What I did

- Read the repository agent instructions and the docmgr, diary, and commit-hygiene workflows.
- Refreshed `go-go-golems/tiny-idp` and `wesen/2026-03-27--hetzner-k3s` remotes.
- Confirmed Tiny-IDP `task/prod-tiny-idp` was clean and contained twelve local review-document commits above the source already merged as PR 6.
- Confirmed the go-go-goja integration branch was clean and already pushed to `wesen`; it is not required for this initial deployment.
- Created ticket `TINYIDP-K3S-MSGDESK-PROD-001`, this diary, the implementation design, and 43 phase tasks.
- Queried the live cluster using `kubeconfig-k3s-demo-1.tail879302.ts.net.yaml`.
- Recorded the current node (`91.98.46.169`), Traefik Pod (`10.42.0.193` at observation time), local-path storage, VSO/Argo/cert-manager CRDs, ingresses, and prod-apps policy.
- Wrote the trust boundaries, invariants, phases, acceptance matrix, rollback contract, open risks, and file-review map.

### Why

- A production rollout spans source, images, GitOps, secrets, controllers,
  runtime state, and recovery. A task ledger prevents one layer from being
  mistaken for the final outcome.
- Live cluster facts supersede examples in old documentation.
- Host-key changes are a security event until independently verified.

### What worked

- Remote refresh showed no competing source implementation.
- The authenticated Tailscale kubeconfig reached a Ready Kubernetes node and
  returned the required read-only inventory.
- The current code has a strong baseline: strict production serving, durable
  authorization interactions, account service, external PKCE relying party,
  and hardened embedded registration controls to reuse.

### What didn't work

The first remote inventory command failed exactly as follows:

```text
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
The fingerprint for the ED25519 key sent by the remote host is
SHA256:RxBj19fc/v1yXS288yXMTgpPM1DW0t3cXkq0Ucef5ps.
Offending ECDSA key in /home/manuel/.ssh/known_hosts:174
Host key verification failed.
```

The first sandboxed kubeconfig query also failed because DNS/network access was
restricted:

```text
Unable to connect to the server: dial tcp: lookup k3s-demo-1.tail879302.ts.net on 127.0.0.53:53: dial udp 127.0.0.53:53: socket: operation not permitted
```

Running the same read-only `kubectl --kubeconfig ...` query with approved
network access succeeded.

### What I learned

- The current production node is `91.98.46.169`, not the older approved SSH
  target at `89.167.52.236`.
- `prod-apps` must be extended before Argo may deploy the new namespace.
- The local-path storage class uses `WaitForFirstConsumer`, so PVC and consumer
  cannot be separated into deadlocking sync waves.
- Traefik Pod IPs are ephemeral. Proxy trust must use a verified narrow network
  contract plus NetworkPolicy rather than pinning the observed Pod address.

### What was tricky to build

- “Production” required separating external actions into two sources of truth:
  source artifacts in Tiny-IDP and runtime desired state in the k3s GitOps
  repository. The plan must also preserve a one-time Argo Application bootstrap
  step because this cluster does not automatically create new Applications.
- The design must allow Traefik to terminate TLS without letting arbitrary
  internal callers forge public origin or client address.

### What warrants a second pair of eyes

- Verify the registration-intent protocol shape before it becomes public API.
- Verify the actual cluster Pod CIDR, kube-router NetworkPolicy enforcement,
  and Traefik forwarded-header behavior before choosing trusted CIDRs/hops.
- Independently verify the changed SSH host key before any SSH-only operation.

### What should be done in the future

- Complete Phase 0 by merging current Tiny-IDP upstream and recording the
  verified proxy/network contract.
- Do not start the deferred device/multi-app work inside this ticket.

### Code review instructions

- Start with the design document beside this diary and `tasks.md`.
- Re-run `docmgr doctor --ticket TINYIDP-K3S-MSGDESK-PROD-001`.
- Compare the cluster observations with `gitops/projects/prod-apps.yaml` and
  the current `kubectl get` output before approving manifests.

### Technical details

```text
Tiny-IDP branch: task/prod-tiny-idp
Tiny-IDP upstream: origin/main at bbc8500 after refresh
k3s upstream: origin/main at 8d1db2d after refresh
cluster node: k3s-demo-1 / 91.98.46.169 / k3s v1.34.5+k3s1
storage: local-path / WaitForFirstConsumer
public ingress: Traefik 3.6.9 + cert-manager letsencrypt-prod
```
