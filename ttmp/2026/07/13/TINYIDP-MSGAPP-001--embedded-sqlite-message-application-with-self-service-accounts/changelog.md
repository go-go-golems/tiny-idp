# Changelog

## 2026-07-13

- Initial workspace created


## 2026-07-13

Created the ticket, mapped current embedding and SQLite boundaries, and identified public account provisioning, bootstrap, and in-process OIDC transport gaps.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/admin/users.go — Current internal account creation logic that shaped the public API proposal.
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/options.go — Current public embedding contract reviewed for the design.


## 2026-07-13

Authored the intern-ready application analysis, design, schemas, API sketches, security invariants, test plan, and eight-phase implementation plan.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/design-doc/01-embedded-tiny-idp-sqlite-message-application-analysis-design-and-implementation-guide.md — Primary design deliverable.


## 2026-07-13

Stored current Go OIDC, OAuth2, SQLite, OWASP authentication, and OWASP CSRF references and documented the investigation chronologically.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/reference/01-investigation-diary.md — Chronological research and design evidence.


## 2026-07-13

Validated the complete ticket package with docmgr doctor; all checks passed before publication.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/reference/01-investigation-diary.md — Records validation commands and results.


## 2026-07-13

Rendered and uploaded the ticket index, design, diary, tasks, and changelog as a reMarkable bundle under /ai/2026/07/13/TINYIDP-MSGAPP-001.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/design-doc/01-embedded-tiny-idp-sqlite-message-application-analysis-design-and-implementation-guide.md — Primary document in the uploaded design bundle.

## 2026-07-14 - Phase 0 contract reconciliation

Reconciled the original plan with landed embedding foundations, added 36 implementation tasks, accepted the recommended design decisions, and added an executable external-import and route contract.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/contracts.go — Frozen application contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/13/TINYIDP-MSGAPP-001--embedded-sqlite-message-application-with-self-service-accounts/reference/02-implementation-contract-and-task-map.md — Accepted decisions and task reconciliation


## 2026-07-14 - Phase 3 secure state root

Added the versioned application manifest, deterministic two-database paths, loopback-aware origin validation, owner-only secrets, atomic writes, and focused tests (commit 9f4a4e2).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/state.go — State-root implementation


## 2026-07-14 - Phase 3 application schema

Added the application-owned SQLite store, checksummed migration history, WAL and durability pragmas, owner-only permissions, and focused schema tests (commit c41ba0b).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/appstore.go — Application SQLite boundary


## 2026-07-14 - Phase 3 OIDC login attempts

Persisted only hashed OAuth state and added an atomic one-time consume transition with replay, expiry, wrong-state, and concurrent winner tests (commit 2603c18).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/login_attempts.go — Durable callback replay boundary

