# Changelog

## 2026-07-14

- Initial workspace created


## 2026-07-14

Added intern-ready standalone Docker OIDC design, external boundary analysis, and initial implementation diary.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/oidc_client.go — External-mode reuse evidence.

## 2026-07-15

Defined the explicit development HTTP versus production HTTPS/cookie boundary,
then added committed two-origin Playwright, transport, restart, privilege, and
development-fixture exposure assurance (commits 10edf79, e4a1a2d).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-external-message-desk/README.md — Deployment profile and assurance runbook
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/oidc_client_test.go — Public issuer/private route invariant
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/02-external-demo.spec.mjs — Browser integration flow
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/03-compose-durability-and-secret-check.sh — Restart and exposure assurance

## 2026-07-15

Completed security review, continuation handoff, and reMarkable delivery of the
design, diary, and assurance bundle.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/reference/02-security-review-and-handoff.md — Security evidence, release gate, and continuation roadmap

## 2026-07-14

Implemented external RP runtime, callback-aware interaction CSP, provider UI reuse, registration boundary, and Compose deployment topology (commits a15f51a, 911aa11, 14e0c4a, 8739522).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-external-message-desk/compose.yaml — Two-container topology
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/external_runtime.go — External relying-party composition
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/rendering.go — Validated callback-aware CSP


## 2026-07-14

Completed Compose startup repair and real two-origin Playwright verification: login, scopes, message creation, chooser/switch, local logout, global logout, and fresh-login behavior (commit 8d040cb).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-external-message-desk/docker-entrypoint.sh — Named-volume ownership then privilege drop
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/external_config.go — Private backchannel validation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/14/TINYIDP-EXTERNAL-DEMO-001--standalone-tiny-idp-and-message-desk-docker-oidc-demo/scripts/01-compose-health-smoke.sh — Repeatable Compose health smoke
