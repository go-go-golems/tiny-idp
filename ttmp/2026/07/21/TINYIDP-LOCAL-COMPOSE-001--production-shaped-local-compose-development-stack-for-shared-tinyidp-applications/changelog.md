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


## 2026-07-21

Enabled an opt-in production account chooser, wired it into the shared local Compose IdP, and covered two-account switching in Chromium (commits d940253 and fadfc08).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Live two-account chooser coverage
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — Reviewed browser-visible remembered-account policy


## 2026-07-21

Extended the real Chromium chooser journey to remove a remembered account while retaining another selectable identity (commit 492a659).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Remembered-account removal coverage


## 2026-07-21

Restored request-scoped email-code verification without redisplay, added closed email-limit UX copy, and covered Goja's themed unknown-invitation journey (commits cd93fec and 2403443).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Real Chromium evidence
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpworkflow/submission.go — Private verifier input boundary


## 2026-07-21

Replaced Message Desk's raw OIDC callback error with a non-reflective CSP-safe recovery page and verified it in Chromium (commits 9c70f31 and cb5d2ca).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/app_http.go — RP callback error boundary


## 2026-07-21

Replaced raw consumed-signup-continuation responses with themed terminal restart pages and covered a real browser replay POST (commits 73b0c0d and 10190ba).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Terminal signup replay boundary


## 2026-07-21

Step 25: Split fast, Fosite, two-process, and full test gates; preserve full pre-push coverage (commit a99b0ed).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/Makefile — Test target contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/lefthook.yml — Pre-push full test gate


## 2026-07-21

Step 26: Rendered Goja Auth OAuth callback failures as safe same-origin recovery pages (Goja f8ff1af; Compose 9c37b66).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go — Callback renderer
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/compose.yaml — Local stylesheet wiring


## 2026-07-21

Step 26 follow-up: Committed Chromium coverage for Goja callback recovery (commit a62d319).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Cross-client callback UX proof


## 2026-07-22

Step 27: Made SQLite email-code attempt exhaustion durable, made resend rotate and reset the replacement-code budget, and verified the complete recovery journey in Chromium (commits a41087c and 263603a).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Real HTTPS exhaustion and resend regression
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/email_challenge.go — Commit rejected verification counters before returning errors


## 2026-07-22

Step 28: Fixed the post-signup consent CSP by preserving the canonical RP origin and verified the full new-account Chromium journey (commit cfc1d08).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/registration_test.go — Consent CSP and redirect regression
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Signup-to-consent canonical request handoff


## 2026-07-22

Step 29: Rendered authorization throttling as safe terminal HTML and separated the local exhaustive matrix budget from production defaults (commit 595742b).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/compose.yaml — Local test budget
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/rendering.go — Browser throttling UX


## 2026-07-22

Step 30: Added native password-mismatch presentation and expanded the Goja invitation browser matrix (commit 647d540).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-shared-two-apps/browser-tests/tests/authentication-ux.spec.ts — Chromium coverage
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/scripted_signup.go — Cross-field secret validation

