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
