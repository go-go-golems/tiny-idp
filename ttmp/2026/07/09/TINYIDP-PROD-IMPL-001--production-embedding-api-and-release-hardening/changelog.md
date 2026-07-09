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
