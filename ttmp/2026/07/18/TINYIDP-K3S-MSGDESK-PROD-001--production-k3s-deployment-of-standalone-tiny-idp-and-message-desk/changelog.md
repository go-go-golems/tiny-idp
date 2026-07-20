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

## 2026-07-20

Step 12: Completed Phase 2: replaced the legacy production registration flag with required checked Goja signup-program activation, an explicit initial native-service policy, scripted-provider composition, focused proxy/readiness coverage, and command/documentation regressions (`5546ac5`, `241f928`, `936705e`, `8a2f01f`).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — Required bounded program loading and activation before production listening
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Scripted signup derives its canonical account service without the legacy production flag
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production_test.go — Production startup-contract coverage

## 2026-07-20

Step 13: Confirmed the detailed Phase 3 ledger and added an evidence-based
tracking contract: a task is checked only after its real harness assertion,
focused passing command, diary entry, and commit are present.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/tasks.md — Phase 3 execution order and completion rule
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/reference/01-production-deployment-implementation-diary.md — Tracking rationale and harness boundary

## 2026-07-20

Step 14: Added and verified the real two-process Tiny-IDP/Message Desk
lifecycle harness (`3cc6f38`): isolated durable state, trusted-proxy boundary,
readiness, process logs, and durable audit evidence.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go — Real executable lifecycle and trusted-proxy topology proof

## 2026-07-20

Step 15: Extended the two-process harness through provider-owned scripted
signup, explicit consent, authorization-code callback, and Message Desk
session (`bfda128`), including exact durable IdP user/session row assertions.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go — Browser-equivalent registration and callback evidence

## 2026-07-20

Step 16: Added real two-process Message Desk message creation/listing and
missing-CSRF no-mutation proof (`4daf974`).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go — Authenticated application message and CSRF evidence

## 2026-07-20

Phase 3 behaviour evidence is complete: the two-process harness now proves
signup, callback, message/CSRF, local/provider logout, negative registration,
continuation replay/expiry/concurrency, restarts, and artifact secret scanning.
Focused and race harness gates pass. The repository-wide `make verify` target
remains red only at auditlint due to four current-main signup findings outside
the deployment-harness change set; task `5fm4` is deliberately left open.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go — Executable Phase 3 production-process evidence
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/reference/01-production-deployment-implementation-diary.md — Exact gate outcomes and auditlint boundary

## 2026-07-20

Phase 3 is fully closed. The narrow audit-contract and bounded-conversion
remediations (`95c81e5`, `bc74f06`) made the repository-defined final gate
green: build, full tests, lint, auditlint, gosec, and govulncheck all pass.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpsignup/executor.go — Explicit development audit default and non-negative latency metric
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpcontinuation/service.go — Per-success cleanup metric accounting
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Bounded email challenge limit conversion
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/reference/01-production-deployment-implementation-diary.md — Final Phase 3 verification evidence

## 2026-07-20

Phase 4 checkpoint: added separate hardened OCI build targets for the
production Tiny-IDP host and external Message Desk (`e6f558f`), including
pinned frontend construction, direct non-root entrypoints, fixed runtime
paths, `SIGTERM`, OCI metadata, and local Docker build/inspection evidence.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/images/Dockerfile.tinyidp — Provider production image contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/images/Dockerfile.message-desk — Deterministic Message Desk UI and runtime image contract

## 2026-07-20

Phase 4 image security/readiness checkpoint: added the repeatable Docker
smoke (`cc99a15`) that verifies non-root, read-only-root, owner-only tmpfs
secret ownership, fixed writable paths, TLS Tiny-IDP readiness, and initialized
Message Desk readiness.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/02-production-image-smoke.sh — Executable OCI runtime security and readiness gate
