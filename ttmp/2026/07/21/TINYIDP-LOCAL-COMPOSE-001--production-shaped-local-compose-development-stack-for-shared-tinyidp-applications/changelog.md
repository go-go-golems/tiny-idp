# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Added a bounded terminal browser-error contract, client-themed production rendering, and HTTPS acceptance coverage for registration-origin rejection (commits dffc6c4 and 0ce1fa6).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py — Live HTTPS themed rejection probe
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/rendering.go — Safe buffered browser-error HTTP boundary
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/productionui/renderer.go — Client theme selection for terminal errors


## 2026-07-21

Separated Message Desk account actions vertically and added explicit 15-character password constraints, visible guidance, server-side secret length enforcement, and live HTTPS acceptance coverage (commits 4b15802, 2c136ee, 7ebecc3).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/ui/src/App.tsx — Semantic vertically spaced account navigation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/productionui/templates/workflow.html — Production password constraints and guidance
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpworkflow/submission.go — Server-side sensitive-field length enforcement


## 2026-07-21

Allowed explicit signup to start from an active remembered TinyIDP session and added a live Message Desk-only logout regression journey (commit 1a15439).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/scripts/03-browser-acceptance.py — Exact local-logout then signup regression journey
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Explicit registration now creates and switches identity without preemptive logout


## 2026-07-21

Defined the Playwright browser-state and authentication UX matrix, phased harness plan, and initial defect ledger.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/21/TINYIDP-LOCAL-COMPOSE-001--production-shaped-local-compose-development-stack-for-shared-tinyidp-applications/design-doc/03-playwright-browser-state-and-authentication-ux-test-matrix.md — Intern implementation and review guide


## 2026-07-21

Fixed remembered-session signup continuation loading (c7a2cb7) and added the initial Playwright authentication UX harness and journeys (34959ea).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Initial real-browser UX journeys
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Preserves authoritative interaction bindings


## 2026-07-21

Explained duplicate-email signup failures with a themed workflow-level recovery error and verified it through the Playwright duplicate-email journey (21456f9).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Live duplicate-email browser regression
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Maps duplicate commit failures to safe user guidance
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/workflow.go — Closed global workflow-error contract


## 2026-07-21

Covered themed invalid credential browser retries and recorded the unresolved pre-comparison password mismatch defect (commit 882790a).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Focused real-HTTPS login UX regressions


## 2026-07-21

Recorded protocol-control evidence for the outstanding Playwright approval-navigation defect and added Goja Auth invalid-credential theme coverage (commit 98821fa).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Cross-client invalid credential browser journey


## 2026-07-21

Added focused Chromium coverage for native display-name and password validation boundaries (commit f5f9eaf).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Native signup validation journeys


## 2026-07-21

Prevented rejected one-time email-code redisplay, verified the themed retry journey, and made the pre-commit test gate fast while retaining full pre-push coverage (commit bd4c424).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/lefthook.yml — Commit gate scope
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpworkflow/descriptors.go — One-time code redisplay policy


## 2026-07-21

Added a reliable Playwright resend recovery journey that verifies blank themed retry state without coupling to Mailpit message ordering (commit 137ebd3).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Resend UX coverage


## 2026-07-21

Added real-browser RP-initiated provider logout coverage, including Message Desk guest state and retired TinyIDP session cookie (commit 9d25a40).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Provider logout journey

