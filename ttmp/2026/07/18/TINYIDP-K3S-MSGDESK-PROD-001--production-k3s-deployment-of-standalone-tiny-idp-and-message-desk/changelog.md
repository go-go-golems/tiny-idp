# Changelog

## 2026-07-18

- Initial workspace created


## 2026-07-18

Created the 43-task production delivery plan, implementation design, diary, live cluster inventory, acceptance matrix, and rollback contract.
### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/design-doc/01-production-implementation-and-deployment-plan-for-standalone-tiny-idp-and-message-desk.md — Production source, GitOps, rollout, and recovery design

## 2026-07-18

Phase 0 complete: merged current upstream in 4cce6b6 and verified the live cluster proxy, storage, VSO, Argo, and namespace-policy contracts.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/reference/01-production-deployment-implementation-diary.md — Phase 0 evidence and exact failures


## 2026-07-18

Phase 1 checkpoint: added opt-in provider-owned durable signup interactions and PKCE/replay tests in d5927e8.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Canonical registration continuation implementation


## 2026-07-18

Step 9: Backfilled listener/audit checkpoints and wired IdP bootstrap (pending commit)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — Production listener/bootstrap owner

## 2026-07-20

Step 10: Re-baselined the production design and task ledger on the completed lambda-first Goja signup architecture. Removed the planned dual signup path and competing Kubernetes bootstrap owner; recorded the remaining source, assurance, image, GitOps, rollout, and recovery sequence.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/design-doc/01-production-implementation-and-deployment-plan-for-standalone-tiny-idp-and-message-desk.md — Superseding production architecture and delivery phases
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/tasks.md — Reconciled implementation ledger

## 2026-07-20

Step 11: Expanded every remaining delivery phase into dependency-ordered, independently checkable implementation, security, test, publishing, GitOps, acceptance, and recovery tasks without expanding the approved scope.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/tasks.md — Precise progress ledger
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/reference/01-production-deployment-implementation-diary.md — Task decomposition rationale and continuation instructions
