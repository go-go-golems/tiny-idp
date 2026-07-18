---
Title: Production Deployment Ergonomics and Strict Host Validation
Ticket: TINYIDP-PROD-DEPLOY-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - operations
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/interaction_doctor.go
      Note: Callback-aware CSP verification for xapp interaction health checks
    - Path: repo://docs/storage.md
      Note: Supported SQLite backup and restore contract
    - Path: repo://examples/tinyidp-external-message-desk/idp_seed_test.go
      Note: External example uses public durable-store surface
    - Path: repo://internal/fositeadapter/device_token_handler.go
      Note: Explicit transaction rollback paths for device-token persistence
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: SQLite close/reopen device-grant redemption and replay regression coverage
    - Path: repo://pkg/sqlitestore/store.go
      Note: Documented transaction-scoped account-selection activation helper
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Project-specific persistence and embedding AST analysis
    - Path: repo://ttmp/2026/07/15/TINYIDP-PROD-DEPLOY-001--production-deployment-ergonomics-and-strict-host-validation/scripts/04-create-online-backup.sh
      Note: Online SQLite backup and artifact verification operator primitive
    - Path: repo://ttmp/2026/07/15/TINYIDP-PROD-DEPLOY-001--production-deployment-ergonomics-and-strict-host-validation/scripts/05-offline-restore-drill.sh
      Note: Offline restore, rollback preservation, and durable-state recovery drill
ExternalSources: []
Summary: Repeatable strict-host provisioning, deployment boundaries, and smoke-validation evidence.
LastUpdated: 2026-07-15T18:05:00Z
WhatFor: Make the production host operable without conflating it with the local development server.
WhenToUse: Use before provisioning, deploying, or release-gating tinyidp's strict host.
---




# Production Deployment Ergonomics and Strict Host Validation

## Overview

This ticket turns the existing strict production host into a repeatable
operator workflow. It preserves explicit secret paths and pre-provisioned
identity state rather than adding insecure startup defaults.

The required local provisioning sequence and browser verification page have
passed on a direct-TLS host. A post-approval unresponsiveness was observed
before device-code redemption, so device authorization remains release-blocked
until the behavior is reproduced and fixed.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- [Design guide](design-doc/01-production-deployment-ergonomics-analysis-design-and-implementation-guide.md)
- [Diary](reference/01-investigation-and-implementation-diary.md)

## Status

Current status: **active**

## Topics

- identity
- oidc
- oauth2
- security
- operations
- testing

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
