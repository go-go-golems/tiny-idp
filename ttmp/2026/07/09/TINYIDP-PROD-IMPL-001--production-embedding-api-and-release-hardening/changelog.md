# Changelog

## 2026-07-09

- Initial workspace created

## 2026-07-09

Step 2: wrote the 5,611-word architecture and implementation guide and related its source evidence

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md — Primary implementation guide

## 2026-07-09

Step 4: dry-ran, rendered, and uploaded the implementation guide and phase ledger to reMarkable

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md — Primary reMarkable guide document

## 2026-07-09

Step 5: established the Go 1.26.5 and go-jose/v3 3.0.5 release baseline with CI, SBOM, provenance, and zero reachable vulnerabilities (commit a2c86a9)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/.github/workflows/ci.yml — Phase 0 CI enforcement
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/go.mod — Release dependency and toolchain baseline

## 2026-07-09

Step 6: closed Phase 0 from committed state and advanced to Phase 1 (code commit a2c86a9)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/go.mod — Verified Phase 0 release graph
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md — Phase 0 gate and Phase 1 progress state

## 2026-07-09

Step 7: inventoried the full transitive internal-type leak at the Phase 1 embedding boundary

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/storage/interfaces.go — Transitive public store type surface
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/options.go — Current public options leaking internal contracts

## 2026-07-09

Steps 8-9: published store, policy, and SQLite packages and replaced embedding construction with context, readiness, close, and positive external compilation (commits e042a15, 0bcbf24, 24c9a92, e65ff53)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/provider.go — Replacement embedding lifecycle
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/store.go — Public durable store

## 2026-07-09

Step 10: closed production construction and the outside-module TLS Authorization Code plus S256 PKCE gate (commit 88e29fd)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/options.go — Production preflight requirements


## 2026-07-09

Step 11: implemented public transactions, named security invariants, checksummed migrations, and concurrent rollback evidence (commit df72fdd)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/store.go — Transactional SQLite and migration implementation


## 2026-07-09

Step 12: completed and verified online backup, atomic publication, offline restore, Phase 1 gate, and Phase 2 gate (commit 7cd13b4)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/backup.go — Verified recovery implementation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md — Checked Phase 1 and Phase 2 gates
