# Changelog

## 2026-07-07

- Initial workspace created


## 2026-07-07

Created production embeddable IdP research ticket, downloaded OIDC/OAuth/Fosite/OWASP sources, analyzed current tiny-idp behavior, and wrote design guide plus intern textbook.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/design-doc/01-production-embeddable-idp-design-and-implementation-guide.md — Primary implementation guide
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/reference/01-oidc-intern-textbook.md — Intern textbook
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/sources/README.md — Downloaded source index


## 2026-07-07

Validated ticket with docmgr doctor; all checks passed.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/index.md — Ticket passed docmgr doctor


## 2026-07-07

Uploaded design guide, intern textbook, and source index bundle to reMarkable at /ai/2026/07/07/TINYIDP-PROD-001 as TINYIDP PROD 001 Guides.pdf.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/design-doc/01-production-embeddable-idp-design-and-implementation-guide.md — Uploaded in bundle
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/07/TINYIDP-PROD-001--production-embeddable-idp-reorganization/reference/01-oidc-intern-textbook.md — Uploaded in bundle


## 2026-07-07

Implemented phases 1-3 foundation: domain models/validation, storage interfaces, memory store test suite, OIDC metadata helpers, and key/JWKS helpers (commit 05b7189).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/domain/types.go — Production domain model
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/keys/keys.go — RSA key and JWKS helpers
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/oidcmeta/discovery.go — Production discovery metadata
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/storage/interfaces.go — Store contracts
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/memory/store.go — Memory store implementation


## 2026-07-07

Implemented phases 4-7 scaffold: strict adapter seam and end-to-end code flow, embedded provider API and validation, SQLite store with migrations and restart key test, and tinyidp serve --engine mock|fosite wiring (commit 1a796cf).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve.go — CLI engine selection
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Strict production-like adapter seam
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/sqlite/store.go — SQLite persistent store
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/provider.go — Public embedded provider API


## 2026-07-07

Replaced strict handwritten adapter spike with real Ory Fosite composition for authorize, token, refresh, OIDC ID token generation, and UserInfo token introspection.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/go.mod — Added github.com/ory/fosite dependency
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Real Fosite-backed strict adapter


## 2026-07-07

Implemented durable SQLite-backed Fosite protocol store and restart tests for authorization-code exchange plus refresh-token use across provider restarts.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore.go — SQLite-backed Fosite protocol storage
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore_test.go — Restart durability test for code and refresh-token flows
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/sqlite/store.go — Exposes SQL handle for protocol-state adapter


## 2026-07-07

Implemented Phase 8 hardening foundation: strict login CSRF protection, security headers, no-store token behavior, structured audit sink/events, and consent policy interfaces/defaults.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/audit/audit.go — Structured audit sink
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/consent.go — Consent policy interface and implementations
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/csrf.go — CSRF cookie/token issue and validation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/hardening_test.go — Hardening tests
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Security headers, audit emissions, consent checks, no-store token handling


## 2026-07-07

Extended Phase 8 hardening with server-side IdP browser sessions, prompt=none login_required handling, silent authorization reuse, consent continuation from an existing session, and rate-limiting hooks/defaults.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Session-aware authorize flow and rate-limit enforcement
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/ratelimit.go — Rate-limiter interface and fixed-window implementation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/session.go — Server-side session cookie and lookup
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/session_test.go — Session reuse and prompt=none tests


## 2026-07-07

Added persistent consent records to domain/storage/memory/SQLite stores, made production strict provider default to stored consent, and added refresh-token reuse regression coverage for SQLite-backed Fosite state.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/domain/types.go — Consent domain model
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/consent.go — Stored consent policy
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore_test.go — Refresh-token reuse regression
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/storage/interfaces.go — Consent store contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/sqlite/store.go — SQLite consent persistence


## 2026-07-07

Completed remaining strict-engine hardening items: stable audit reason codes, Fosite schema ownership in SQLite migrations, RSA signing-key rotation with retired-key verification retention, ID Token JWKS validation coverage, and production security/storage/conformance runbooks.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/docs/conformance.md — Strict-engine conformance runbook
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/audit_reason.go — Stable audit reason normalization
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Audit normalization and ID Token kid header
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore.go — Fosite store no longer owns schema DDL
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/keys/rotation.go — Signing-key rotation helper
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/sqlite/migrations/001_schema.sql — Domain and Fosite schema ownership
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/scripts/run-conformance.sh — Local conformance validation script

