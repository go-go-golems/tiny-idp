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

