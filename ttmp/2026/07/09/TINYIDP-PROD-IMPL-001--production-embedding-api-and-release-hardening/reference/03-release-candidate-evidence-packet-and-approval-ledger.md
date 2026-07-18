---
Title: Release candidate evidence packet and approval ledger
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://.github/workflows/ci.yml
      Note: Always-on local-equivalent gates
    - Path: repo://.github/workflows/release-evidence.yml
      Note: SBOM, signatures, provenance, licenses, and checksums
    - Path: repo://.github/workflows/release-gates.yml
      Note: Exact candidate race, fuzz, recovery, and hosted OIDF
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/reference/02-phase5-runtime-load-summary.md
      Note: Exact-candidate mixed load analysis
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-IMPL-001--production-embedding-api-and-release-hardening/various/phase5-production-host-smoke.json
      Note: Exact-candidate TLS host evidence
ExternalSources: []
Summary: Evidence and explicit approval boundary for candidate 2930981, including local hashes, tests, drills, residual risks, and unresolved external gates.
LastUpdated: 2026-07-09T20:31:17.620769528-04:00
WhatFor: Determining exactly what has and has not passed before anyone labels tiny-idp production-approved.
WhenToUse: Use for release review, hosted conformance, security sign-off, artifact publication, deployment approval, and rollback decisions.
---


# Release candidate evidence packet and approval ledger

## Release decision

**Candidate status: NOT APPROVED FOR PRODUCTION.**

The locally actionable engineering work and local release review are complete.
The candidate still lacks hosted OpenID Foundation conformance bound to the
deployed binary hash, actual CI-generated signatures/SBOM/provenance, complete
license reconciliation, a production-environment deployment/restore drill, an
independent security/code review, and release-owner approval.

Do not mark the final ticket task, create a production tag, or describe this
candidate as approved until every blocking row below has attached evidence and
a named approver.

## Candidate identity

| Field | Value |
|---|---|
| Source commit | `29309814f1fcdad3a5134674fc27a8938cb39c6a` |
| Branch during review | `task/prod-tiny-idp` |
| Build command | `go build -trimpath -buildvcs=false -o tinyidp-linux-amd64 ./cmd/tinyidp` |
| Local linux/amd64 SHA-256 | `1df7b90b9365fb8ad0b55473db93a050a71e86c11b3156616f1f9388b102f2ae` |
| Go toolchain | `go1.26.5`, `GOTOOLCHAIN=auto` |
| Build environment | `CGO_ENABLED=1`, `GOOS=linux`, `GOARCH=amd64`, `GOAMD64=v1` |
| Source cleanliness | built from a clean `git archive 2930981` outside the documentation worktree |
| Schema | version 5, checksummed migrations 001–005 |

The release workflows use the same `-trimpath -buildvcs=false` command. Commit
identity remains in GitHub artifact metadata and provenance rather than inside
the binary. A manual release-gate run requires the expected SHA-256 as input and
fails if its locally built candidate differs.

## Gate ledger

| Gate | Status | Evidence | Blocking follow-up |
|---|---|---|---|
| Build and all Go tests | PASS | clean archive `go test ./... -count=1`; `go build ./...` | none |
| Vet and repository Go AST analysis | PASS | `go vet ./...`; audit multichecker on `./pkg/... ./internal/...` | none |
| Pinned lint | PASS | golangci-lint v2.12.2: 0 issues; Glazed lint passed | none |
| Reachable vulnerability scan | PASS | govulncheck v1.5.0: 0 reachable; 2 imported-package and 14 required-module findings not called | rerun in signed workflow |
| Full race detector | PASS | `go test -race ./... -count=1` | rerun in release workflow |
| Parser fuzzing | PASS (local bounded) | issuer 474,734 executions; redirect 514,796; Argon encoding 442,681; no failure | 30-second-per-target CI job must pass |
| Outside-module production OIDC | PASS | external module compiled and completed Authorization Code + S256 PKCE | rerun on published module/tag |
| Production TLS host | PASS (local) | candidate `2930981`; HTTPS/HTTP2; liveness and eight readiness checks green; owner-only files; graceful SIGINT | repeat on target production platform |
| Mixed login/token/read load | PASS (local) | 5,125 requests, 25 password flows, 129 audit events, zero HTTP errors | validate against target cgroup/disk/SLO |
| Password-work bound | PASS | capacity 2; completed 25; saturations 22; rejected 0; 8.00s aggregate wait; 3.46s Argon duration | choose deployment memory/latency budget |
| SQLite pool under load | OBSERVED | one connection; 8,847 waits; wait time captured in runtime summary | validate target-disk contention/SLO |
| Backup/restore/migration/rotation drills | PASS (local) | backup verify/restore, rollback preservation, newer-schema refusal, signing rotation, token-secret invalidation | repeat using production volume and exact signed artifact |
| Always-on CI definition | READY | build/test/vet/lint/analyzer/vulnerability/fuzz seeds/external consumer/recovery | required branch checks must be enabled |
| Release CI definition | READY | hash binding, race, longer fuzz, injected faults, recovery drills, hosted OIDF | workflow has not run for this candidate |
| Hosted OpenID Foundation conformance | **BLOCKED** | runner/workflow exists; no plan result attached | supply plan ID and GitHub secrets; deploy exact hash; run to pass |
| SBOM, signatures, provenance | **BLOCKED** | workflow exists; local candidate is unsigned and no SBOM was generated | run `release-evidence` with GitHub OIDC and retain artifacts |
| Dependency license inventory | **BLOCKED** | collector found 354 module directories with top-level notices; 8 modules lacked a top-level license file in module cache | manually reconcile the 8 rows before distribution |
| Independent security/code review | **BLOCKED** | no independent reviewer recorded | named reviewer signs findings/acceptance |
| Release-owner approval | **BLOCKED** | no owner signature recorded | owner accepts residual risk after all evidence |

## Runtime evidence

The exact-candidate rerun is stored as:

- `reference/02-phase5-runtime-load-summary.md` — latency, runtime, DB, audit,
  and password-work interpretation;
- `various/phase5-runtime-load.ndjson` — 5,130 structured events/snapshots;
- `various/phase5-runtime-load.cpu.pprof` and `.heap.pprof` — Go profiles;
- `various/phase5-production-host-smoke.json` — TLS host/readiness/permissions
  and shutdown evidence.

The measured request mix was:

| Operation | Count | Result |
|---|---:|---|
| discovery reads | 1,250 | all 200 |
| JWKS reads | 1,250 | all 200 |
| readiness reads | 1,250 | all 200 |
| load userinfo reads | 1,250 | all 200 |
| password authorize GET/POST | 25 each | all 200/303 |
| code exchange | 25 | all 200 |
| refresh exchange | 25 | all 200 |
| flow userinfo | 25 | all 200 |

This is a bounded in-process load on a temporary local SQLite file. It proves
the work semaphore, one-connection envelope, audit delivery, and mixed protocol
paths behave coherently. It is not a capacity forecast for the production
filesystem, reverse proxy, CPU quota, or cgroup memory limit.

## Recovery and incident evidence

`scripts/02-release-drills.sh` performed:

1. database initialization and migration 005;
2. user/client/key provisioning through durable admin audit;
3. doctor preflight;
4. online backup and independent read-only verification;
5. signing-key rotation with old-key overlap;
6. post-backup mutation;
7. offline restore with `.pre-restore-*` rollback preservation;
8. proof that the post-backup mutation disappeared after restore;
9. doctor after restore;
10. future-schema/downgrade refusal;
11. proof that token-secret rotation invalidates old opaque access and refresh
    tokens.

The operator runbook separately covers corruption, normal and emergency key
rotation, token/client secret compromise, admin lockout, audit failure,
dependency emergencies, and rollback.

## Residual risk register

| ID | Severity | Risk and consequence | Current control | Owner / expiry / acceptance |
|---|---|---|---|---|
| R1 | High/blocking | Hosted OIDC edge cases may still fail against the official suite. | Local strict/external flows and hosted runner. | Release owner (unassigned); expires only on passing exact-hash evidence; no release exception. |
| R2 | High/blocking | No independent review has challenged the implementation or threat model. | This guide, full diary, tests, analyzers. | Security reviewer (unassigned); expires only on signed disposition; no release exception. |
| R3 | High/blocking | Local artifact is unsigned and lacks CI-produced SBOM/provenance. | Keyless signing/attestation workflow wired. | Release engineer (unassigned); expires only on verified CI artifacts; no distribution exception. |
| R4 | Medium/blocking | Eight transitive modules lack a top-level license file in the downloaded module directory. | Collector records `MISSING.tsv`; full module graph retained. | Legal/release owner (unassigned); reconcile before distribution; no release exception. |
| R5 | Medium | Audit mutation and audit append are not one database transaction. A post-commit append failure creates an evidence gap. | Synchronous fsync, typed `ErrAuditDelivery`, readiness failure, operator reconciliation. | Security/release owner; accept/remediate by 2026-08-09 and before release. |
| R6 | Medium | Signing private keys are stored in owner-only SQLite, not KMS/HSM. DB compromise exposes them. | 0600 files, single-node boundary, rotation and emergency purge. | Deployment/security owner; decide by 2026-08-09 and before internet exposure. |
| R7 | Medium | In-process rate-limit buckets reset on restart and do not support active/active. | Single-node topology, global Argon semaphore, audit metrics. | Deployment owner; accept/inject alternative by 2026-08-09 and before release. |
| R8 | Medium | Audit log has no in-process rotation/shipping/retention. A full volume fails readiness. | Owner-only fsync file and explicit health. | Operations owner; configure before launch; review by 2026-08-09. |
| R9 | Medium | SQLite is a single active node and has no transparent HA. | Verified online backups, restart/failover envelope, restore drill. | Service owner; document RTO/RPO and target volume before launch; review by 2026-08-09. |
| R10 | Medium | Load evidence is local and shows substantial one-connection wait time under concurrency. | DB wait metrics and bounded password work. | Performance owner; target-environment SLO test before approval; review by 2026-08-09. |
| R11 | Low/medium | Token-secret rotation forces all browser/access/refresh sessions to reauthenticate. | Explicit runbook and drill. | Product/operations owner; accept and communicate before first rotation; review by 2026-08-09. |
| R12 | Low/medium | Immediate client-secret rotation has no two-secret overlap. | One-time generated secret and coordinated cutover. | Client owner; accept before first confidential-client rotation; review by 2026-08-09. |

No claim that “no unaccepted P1 exists” is made until R1–R4 and the independent
review are closed. R5–R12 need named acceptance, remediation, or deployment
controls with dates.

Rollback/remove-from-traffic criteria are any artifact hash mismatch, failed
signature/provenance verification, unsupported schema, any readiness component
remaining false beyond its incident threshold, audit delivery failure, failed
external OIDC smoke/conformance, integrity/checksum failure, unexplained token
acceptance after rotation, or a new P0/unaccepted P1. Roll back code only when
the prior binary supports the current schema; otherwise restore the verified
pre-upgrade backup or roll forward, as detailed in the operations runbook.

## License follow-up list

The local module-cache scan found no top-level license file for:

- `github.com/agnivade/levenshtein v1.2.1`;
- `github.com/aymanbagabas/go-udiff v0.3.1`;
- `github.com/chzyer/logex v1.1.10`;
- `github.com/josharian/intern v1.0.0`;
- `github.com/kr/pretty v0.3.1`;
- `github.com/kr/text v0.2.0`;
- `github.com/mattn/go-localereader v0.0.1`;
- `github.com/niemeyer/pretty` at pseudo-version
  `v0.0.0-20200227124842-a10e7caefd8e`.

This is not a statement that the projects are unlicensed. It means the release
collector could not find a conventional top-level notice in the downloaded Go
module directory. Review upstream authoritative repositories and record the
license/notice before release.

## Hosted conformance execution

The manual workflow requires:

- `plan_id`: hosted suite plan configured to the externally reachable issuer;
- `candidate_sha256`: the exact deployed linux/amd64 hash above;
- environment secrets `OIDF_API_TOKEN` and/or `OIDF_JSESSIONID`;
- `TINYIDP_LOGIN` and `TINYIDP_PASSWORD` for the test identity.

The local agent did not read these environment values. The job stores per-test
info/log JSON. A pass must record plan ID, module variants, test IDs, timestamps,
results, issuer, and artifact hash here.

## Approval signatures

### Independent security/code reviewer

- Name: **unassigned**
- Review date: **pending**
- Reviewed source commit: **pending**
- Findings disposition: **pending**
- Decision/signature: **pending**

### Release owner

- Name: **unassigned**
- Approval date: **pending**
- Signed artifact hash: **pending**
- Hosted plan ID/result: **pending**
- Accepted residual risks and expiry dates: **pending**
- Rollback owner/criteria: **pending**
- Decision/signature: **NOT APPROVED**

## Final approval algorithm

```text
if any blocking gate lacks exact-candidate evidence:
    release = NOT APPROVED
if independent reviewer has not signed:
    release = NOT APPROVED
if any P0 or unaccepted P1 remains:
    release = NOT APPROVED
if artifact hash differs from conformance/deployment/signature evidence:
    release = NOT APPROVED
otherwise:
    release owner records residual-risk acceptance and rollback criteria
    mark Phase 5 gate and final task complete
```

## Related documents

- `design-doc/01-production-embedding-api-and-release-implementation-guide.md`
- `reference/01-implementation-diary.md`
- `reference/02-phase5-runtime-load-summary.md`
- `playbook/01-production-operations-and-incident-response-runbook.md`
- `tasks.md`
