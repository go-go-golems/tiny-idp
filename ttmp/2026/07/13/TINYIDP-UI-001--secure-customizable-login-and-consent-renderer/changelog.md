# Changelog

## 2026-07-13

- Initial workspace created


## 2026-07-13

Created an evidence-backed 1,500-line secure interaction renderer design, seven-source research packet, detailed diary, and 47-task implementation and assurance plan; runtime implementation remains pending approval.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Current strict rendering and authorization flow analyzed
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/options.go — Proposed public renderer injection boundary


## 2026-07-13

Validated the ticket cleanly and delivered the index, design guide, diary, tasks, and changelog as a bundled PDF to /ai/2026/07/13/TINYIDP-UI-001 on reMarkable.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/design-doc/01-secure-interaction-rendering-analysis-design-and-implementation-guide.md — Primary delivered design
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/reference/01-investigation-diary.md — Delivery evidence and continuation log


## 2026-07-13

Step 5: Recorded explicit implementation approval and prepared the design and research ticket as a documentation-only baseline before runtime changes.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/reference/01-investigation-diary.md — Approval gate and baseline preparation


## 2026-07-13

Step 6: Implemented and committed the dependency-light pkg/idpui contract, validated page model, contextual default renderer, semantic golden, and structural security tests (commit e77158f).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/default_renderer_test.go — Phase 1 verification evidence
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/renderer.go — Public rendering boundary


## 2026-07-13

Step 7: Integrated custom interaction rendering through the public embedding API, added bounded pre-commit rendering, strict action validation, recoverable retry pages, a reviewed same-origin-style CSP, and POST-to-303 redirect handling (commits 817fb15 and fdd008f).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/rendering.go — Typed page construction and bounded response rendering
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/options.go — Public UI configuration boundary


## 2026-07-13

Step 8: Added and wired the xapp-owned themed interaction renderer, embedded same-origin stylesheet, strict asset URL validation, page-shape coverage, and development/production HTTP smoke tests (commit fc16a87).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/internal/loginui/renderer.go — Host-owned renderer and asset handler
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/internal/loginui/static/login.css — Product interaction theme
