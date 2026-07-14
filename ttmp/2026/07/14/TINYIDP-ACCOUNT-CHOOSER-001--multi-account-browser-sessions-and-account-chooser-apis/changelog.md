# Changelog

## 2026-07-14

- Initial workspace created


## 2026-07-14

Created evidence-backed multi-account account chooser design, stored official OIDC sources, and added six implementation phases.


## 2026-07-14

Implemented phased storage foundation: opaque browser contexts, remembered sessions, atomic fresh-handle activation, SQLite migration, backup/maintenance support, and cross-store invariant tests (commit 7b4fa5ec347ce088b32468381eca4a0e4471cdb5).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpstore/interfaces.go — Public persistence contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/migrations/007_browser_contexts.sql — Schema migration


## 2026-07-14

Implemented provider lifecycle slice: opt-in opaque browser contexts, host label policy, deduplicated bounded remembered membership, public embedded configuration, and atomic global browser-context logout (commit 7b19d58014e60a92fa3a7d49a9de62a374e9dd7c).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/end_session.go — Global logout
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/session.go — Atomic lifecycle


## 2026-07-14

Added typed, renderer-safe account chooser prompt contract with opaque entry values, validation, default accessible radio rendering, and focused UI tests (commit 01e96adf026aa972e854702bcd860b554baf09b5).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/types.go — Public chooser UI model

