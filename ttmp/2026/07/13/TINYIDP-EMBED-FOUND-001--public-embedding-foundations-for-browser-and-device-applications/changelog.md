# Changelog

## 2026-07-13

- Initial workspace created


## 2026-07-13

Phase 0: mapped account, bootstrap, transport, browser, and device-client foundations; accepted the public API and no-adapter migration design.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-EMBED-FOUND-001--public-embedding-foundations-for-browser-and-device-applications/design-doc/01-public-account-bootstrap-and-in-process-issuer-apis-analysis-design-and-implementation-guide.md — Primary accepted foundation design.

## 2026-07-13

Implemented Phase 1 public account lifecycle API, removed internal/authn, migrated provider, CLI, xapp, tests, and executable probes, and passed focused, race, and full-repository tests.

### Related Files

- internal/admin/service.go — Administration boundary after extraction
- pkg/idpaccounts/accounts.go — Public atomic account lifecycle
- pkg/idpaccounts/password.go — Public password authenticator and work reporter

## 2026-07-13

Implemented Phase 2 declarative browser/device/generic client bootstrap, semantic drift detection, initial signing-key provisioning, and xapp initialization migration; focused, race, and full tests pass.

### Related Files

- cmd/tinyidp-xapp/state.go — Xapp consumes bootstrap API
- pkg/embeddedidp/bootstrap.go — Public bootstrap API

## 2026-07-13

Implemented Phase 3 bounded fail-closed in-process issuer transport, comprehensive path/origin/overflow/cancellation tests, and xapp development/production migration; race and full tests pass.

### Related Files

- cmd/tinyidp-xapp/development_app.go — Xapp development transport consumer
- pkg/embeddedidp/inprocess_transport.go — Public exact-issuer RoundTripper

## 2026-07-14 - Phase 4 public embedding assurance complete

Published the supported composition guide and executable browser/device examples; added a Go analysis import boundary; migrated xapp development identity state to public SQLite APIs with restart drift tests; aligned verification with go.work; passed full build, tests, race-selected tests, lint, custom analyzers, formatting, logging generation checks, patched-toolchain vulnerability analysis, and a live tmux HTTP smoke. Implementation commit: 519a4cf.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/development_app.go — Public persistent embedding migration
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/docs/embedding-foundations.md — Public embedding guide
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/embedded/main.go — Live-smoked runnable example
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go — Static import boundary


## 2026-07-14 - Ticket completed and prepared for reMarkable delivery

All phases and 20 tracked tasks are complete. Docmgr validation passes with warnings treated as failures. The committed six-document ticket bundle is prepared for one non-interactive upload to /ai/2026/07/14/TINYIDP-EMBED-FOUND-001.


## 2026-07-14

Ticket closed
