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
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: SQLite close/reopen device-grant redemption and replay regression coverage
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
