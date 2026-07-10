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

## 2026-07-09

Published the committed Phase 1 and Phase 2 guide, storage reference, diary, and ledger bundle to reMarkable at /ai/2026/07/09/TINYIDP-PROD-IMPL-001

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md — Uploaded Phase 1 and Phase 2 completion guide

## 2026-07-09

Step 13: closed Phase 3 with NIST-aligned password policy, bounded observable Argon2 work, trusted-address and three-dimensional throttling, fail-closed storage, and password-change revocation (commit 7022e7d)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/authn/password.go — Bounded fail-closed password authentication
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/tasks.md — Checked Phase 3 gate

## 2026-07-09

Step 14: closed Phase 4 with validated/JWKS-retained signing keys, synchronous durable audit, structured liveness/readiness, atomic retention maintenance, effective cookie/TTL/route contracts, and transition tests (commit f8c35bb)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/provider.go — Structured lifecycle and maintenance health
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idp/audit.go — Synchronous durable audit contract
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/sqlitestore/maintenance.go — Atomic retention implementation

## 2026-07-09

Step 15: implemented the production TLS host, emergency key purge, release/recovery workflows, exact-candidate load and drill evidence, operator runbook, and explicit not-approved release ledger (commits 2a0b287, 5e23978, 2930981)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — Production host implementation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/.github/workflows/release-gates.yml — Exact-hash race/fuzz/fault/recovery/hosted gate
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/03-release-candidate-evidence-packet-and-approval-ledger.md — Release decision and blockers

## 2026-07-09

Step 16: dry-ran and uploaded the committed guide, runbook, evidence ledger, runtime summary, diary, and task bundle to reMarkable at /ai/2026/07/09/TINYIDP-PROD-IMPL-001

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/01-production-embedding-api-and-release-implementation-guide.md — Primary uploaded guide
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/03-release-candidate-evidence-packet-and-approval-ledger.md — Uploaded not-approved decision ledger

## 2026-07-10

Step 18: researched CS foundations and designed layered static analysis, model-based testing, fuzzing, deterministic concurrency, fault injection, trace monitoring, and isolated scripting verification architecture

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md — Primary intern-ready assurance guide

## 2026-07-10

Step 18 delivery: doctor passed for both assurance tickets and the five-document research bundle was rendered and uploaded to reMarkable at /ai/2026/07/10/TINYIDP-PROD-IMPL-001

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/02-security-invariant-assurance-architecture-static-analysis-runtime-verification-and-stateful-testing.md — Uploaded primary assurance guide
