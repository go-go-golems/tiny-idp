---
Title: Public Embedding Foundations for Browser and Device Applications
Ticket: TINYIDP-EMBED-FOUND-001
Status: complete
Topics:
    - go
    - identity
    - oidc
    - architecture
    - security
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://.github/workflows/ci.yml
      Note: CI execution of repository-specific analyzers
    - Path: repo://Makefile
      Note: Workspace-aware verification and analyzer entry point
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Persistent public SQLite embedding consumer and seed reconciliation
    - Path: repo://cmd/tinyidp-xapp/development_app_test.go
      Note: Restart idempotency and credential drift regression
    - Path: repo://docs/embedding-foundations.md
      Note: Supported host composition and production handoff guide
    - Path: repo://examples/embedded/main.go
      Note: Runnable public API browser composition
    - Path: repo://pkg/embeddedidp/example_test.go
      Note: Executable external-package browser and device examples
    - Path: repo://ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go
      Note: Go analysis public embedding import guard
ExternalSources: []
Summary: Implement reusable public account provisioning, browser/device client bootstrap, signing-key provisioning, and bounded in-process issuer transport APIs for tiny-idp embedding hosts.
LastUpdated: 2026-07-14T12:50:01.044606501-04:00
WhatFor: Track design, implementation, verification, commits, and delivery for tiny-idp's shared application embedding foundations.
WhenToUse: Start here when reviewing or resuming TINYIDP-EMBED-FOUND-001.
---



# Public Embedding Foundations for Browser and Device Applications

## Overview

This ticket moves password-backed account creation and authentication into a
supported public package, adds conservative provider bootstrap for browser and
device-shaped clients, provisions the initial signing key without exposing key
representation, and adds an exact-origin bounded in-process HTTP transport for
OIDC back-channel requests.

The work is shared by the existing xapp, the planned SQLite message
application, and a later device-authorization example. The device example's
strict grant endpoint remains later work, but this ticket ensures client
bootstrap does not require a browser callback.

## Key Links

- [Primary design and implementation guide](./design-doc/01-public-account-bootstrap-and-in-process-issuer-apis-analysis-design-and-implementation-guide.md)
- [Implementation diary](./reference/01-implementation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **complete**. Public accounts, declarative bootstrap, bounded
in-process transport, consumer migration, documentation, static import guards,
release-quality verification, and delivery bookkeeping are complete.

## Topics

- go
- identity
- oidc
- architecture
- security

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and implementation design
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
