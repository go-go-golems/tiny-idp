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

## 2026-07-10

Step 19: expanded authorization hardening into 31 detailed tasks with accepted semantics, phase exits, validation commands, and commit boundaries

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/05-authorization-interaction-hardening-implementation-ledger.md — Task-by-task implementation ledger

## 2026-07-10

Added red authorization-interaction regressions and implemented server-owned interaction persistence with atomic one-time consume in memory and SQLite.

## 2026-07-10

Steps 21-22: completed detailed authorization hardening Phases 1-5 with opaque interactions, strict reauthentication and consent, adjacent endpoint hardening, atomic SQLite issuance, failpoints, AST analyzers, and full verification (commits aedff3c, 34580db, 27c339e).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Authorization hardening implementation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore.go — Atomic protocol persistence

## 2026-07-10

Step 23: implemented atomic token lifecycles, 18 token failpoints, Rapid state-machine testing, Porcupine histories, and versioned runtime trace monitoring; added research-to-code design context (commit 26fa7db).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore.go — Token lifecycle transaction implementation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/design-doc/03-research-foundations-for-identity-protocol-invariants-atomicity-and-runtime-verification.md — Academic and standards context

## 2026-07-10

Completed protocol invariant analyzers, multi-source security monitor feeds, and the data-only Goja VerificationPlan compiler; added a research boundary decision record and detailed diary.

### Related Files

- design-doc/04-programmable-verification-plans-research-boundary-and-implementation.md — Academic and implementation traceability
- internal/gojaverify/compiler.go — Isolated plan compilation
- pkg/verifyplan/plan.go — Native plan runner

## 2026-07-10

Added the native strict-provider VerificationPlan driver, typed action observations, deterministic clock, metamorphic relation, minimized replay histories, and action-sequence fuzzing.

### Related Files

- internal/fositeadapter/state_model_test.go — Pure model, replay histories, and fuzz seeds
- internal/fositeadapter/verification_scenario_test.go — Strict HTTP scenario driver and native assertions

## 2026-07-10

Froze local exact-candidate evidence for code commit 5bb4dae and binary cf43cae...f43dd: race, static, lint, vuln, fuzz, failpoint, recovery, external-module, local conformance, proxy, and production-host gates passed; hosted OIDF and generic scanner remain explicit gaps.

### Related Files

- reference/06-exact-candidate-assurance-evidence-5bb4dae.md — Exact command/results ledger and release blockers
- ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-api-smoke.sh — Failure visibility and Go 1.26.1 consumer probe

## 2026-07-10

Rendered and uploaded the complete assurance, research, diary, operations, and exact-candidate reading bundle to reMarkable without overwriting the earlier packet.

### Related Files

- reference/01-implementation-diary.md — Render repair and upload record

## 2026-07-10

Backfilled a seven-document intern curriculum connecting OAuth/OIDC theory, temporal invariants, durable/concurrent security state, assurance science, production trust, and executable labs to the implemented code and saved research.

### Related Files

- design-doc/05-intern-accelerated-curriculum-and-code-reading-map.md — Ordered onboarding path and competence criteria
- design-doc/06-oauth-oidc-protocol-security-foundations.md — Protocol and attacker-model foundations
- design-doc/07-security-state-machines-and-temporal-invariants.md — Temporal reasoning
- design-doc/08-durable-security-state-transactions-and-concurrency.md — Atomicity and concurrency
- design-doc/09-assurance-methods-and-evidence-interpretation.md — Evidence epistemology
- design-doc/10-production-trust-boundaries-and-release-security.md — Operational and release security
- reference/07-intern-security-review-labs.md — Executable onboarding labs

## 2026-07-10

Expanded temporal-invariant and durable-security-state intern chapters into code-led research treatments; captured FAPI attacker-model and runtime-verification teaching sources; recorded the in-progress textbook provenance pass.

### Related Files

- design-doc/07-security-state-machines-and-temporal-invariants.md — Current interaction state, monitor, model, analyzer, and case-study analysis
- design-doc/08-durable-security-state-transactions-and-concurrency.md — Current Fosite transactions, failpoints, linearizability, and recovery analysis
- sources/fapi-2-attacker-model-final.md — Additional explicit attacker vocabulary
- sources/introduction-to-runtime-verification.md — Runtime verification teaching context

## 2026-07-10

Expanded assurance-method and production-security intern chapters into concrete analyses of every current analyzer, test/instrumentation layer, host control, workflow, exact-candidate result, and research influence.

### Related Files

- design-doc/09-assurance-methods-and-evidence-interpretation.md — Tool-by-tool assurance and paper provenance
- design-doc/10-production-trust-boundaries-and-release-security.md — Code-led production and release trust analysis
