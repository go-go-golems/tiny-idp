---
Title: Production readiness review for tiny-idp
Ticket: TINYIDP-PROD-REVIEW-001
Status: active
Topics:
    - oidc
    - go
    - testing
    - auth
    - architecture
    - research
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Evidence-based architecture, security, operations, and code-quality review before tiny-idp is shipped to production."
LastUpdated: 2026-07-09T13:42:50.075172502-04:00
WhatFor: "Deciding whether tiny-idp is production-ready and planning the remediation required before release."
WhenToUse: "Use during release review, onboarding, implementation hardening, and operational readiness work."
---

# Production readiness review for tiny-idp

## Overview

This ticket is the cross-cutting production-readiness review for `tiny-idp`. It
combines code inspection, authoritative protocol and operations research,
purpose-built analysis scripts, and executable verification. The deliverable is
written to orient a new engineer while giving maintainers a prioritized release
decision and remediation plan.

## Key Links

- [Production readiness architecture and code review](./design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- `sources/`: locally captured external references
- `scripts/`: review automation and reproducible probes

## Status

Current status: **active**

### Review verdict

**No-go for production on the reviewed commit.** The strict engine has a sound
Fosite/PKCE/Argon2id/CSRF/persistence foundation, but release is blocked by the
unusable external embedding API, reachable dependency/runtime vulnerabilities,
WAL-unsafe backups, non-transactional security state, bypassable brute-force
controls, and unsafe database permission defaults. The primary review contains
the complete acceptance gate and phased remediation plan.

### Evidence snapshot

- Build, full tests, vet, race, pinned lint, Glazed lint, and rebuilt Staticcheck pass.
- Three five-second native fuzz campaigns completed without failures.
- Go 1.26.5 removes the active runtime's reachable standard-library findings; two reachable go-jose/v3 findings remain on v3.0.3.
- Live probes reproduce external API compilation failure, WAL backup data loss, lockout lost updates, one-character password acceptance, expired-key acceptance, optional controls, and `0644` SQLite creation under permissive umask.
- Runtime happy path completed 45 measured HTTP operations with zero request errors and no goroutine delta after connection cleanup.

### Review commits

- `bcca18c` — ticket and initial diary
- `54fcbcf` — research, analyzers, probes, and runtime instrumentation
- `c1da8d4` — research/instrumentation diary
- `ca40c40` — security invariant and scanner evidence
- `d282362` — full verification diary
- `c387926` — primary production-readiness review

## Topics

- oidc
- go
- testing
- auth
- architecture
- research

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
