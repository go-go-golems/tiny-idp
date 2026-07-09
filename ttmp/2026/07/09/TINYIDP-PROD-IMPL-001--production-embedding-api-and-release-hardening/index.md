---
Title: Production embedding API and release hardening
Ticket: TINYIDP-PROD-IMPL-001
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
Summary: "Long-lived implementation program for a consumable embedding API, transactional SQLite security invariants, mandatory authentication controls, operational hardening, and release proof."
LastUpdated: 2026-07-09T17:37:00.868784708-04:00
WhatFor: "Implementing and verifying every remediation required before tiny-idp can be considered for production release."
WhenToUse: "Use for implementation sequencing, onboarding, code review, release gates, and continuation across working sessions."
---

# Production embedding API and release hardening

## Overview

This ticket executes the production-readiness findings from
`TINYIDP-PROD-REVIEW-001`. It is deliberately long-lived: work is divided into
six ordered phases with independently reviewable tasks, acceptance gates, code
commits, and diary checkpoints.

The program starts by repairing the release dependency graph, then replaces the
currently unusable public embedding API. Persistence and authentication
invariants come next, followed by operational hardening and final release proof.
Passing an earlier phase does not imply the product is releasable; production
release remains blocked until every phase gate and the final evidence packet
are complete.

## Key Links

- [Implementation guide](./design-doc/01-production-embedding-api-and-release-implementation-guide.md)
- [Implementation diary](./reference/01-implementation-diary.md)
- [Detailed phase ledger](./tasks.md)
- [Source production review](../TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/design-doc/01-tiny-idp-production-readiness-architecture-and-code-review.md)

## Status

Current status: **active**

Current phase: **Phase 0 — dependency and toolchain security baseline**

Release status: **blocked until all phase gates pass**

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
