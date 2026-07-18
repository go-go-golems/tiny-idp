---
Title: Embedded SQLite Message Application with Self-Service Accounts
Ticket: TINYIDP-MSGAPP-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - architecture
    - security
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Design a copyable single-process Go application that embeds tiny-idp, uses separate SQLite identity and message stores, supports secure self-registration and OIDC login, and lets authenticated users publish plain-text messages.
LastUpdated: 2026-07-13T20:27:13.676018275-04:00
WhatFor: Track the architecture, public API prerequisites, implementation phases, research, and delivery evidence for the embedded message application.
WhenToUse: Start here when reviewing or implementing TINYIDP-MSGAPP-001.
---

# Embedded SQLite Message Application with Self-Service Accounts

## Overview

This ticket specifies a complete but intentionally small web application that
demonstrates professional tiny-idp embedding. The application runs one Go
process and one public origin, mounts tiny-idp below `/idp`, performs an
ordinary OIDC authorization-code flow with PKCE, persists application sessions
and messages in a separate SQLite database, and permits visitors to create a
durable tiny-idp account before signing in.

The investigation found that provider construction, durable storage, and
stylable interactions are already public. Account creation, signing-key
bootstrap, and the current in-process OIDC transport are not yet suitable for a
copyable external example. The design includes narrow public APIs for those
gaps and prohibits the showcase from importing `github.com/manuel/tinyidp/internal/...`.

## Key Links

- [Primary analysis, design, and implementation guide](./design-doc/01-embedded-tiny-idp-sqlite-message-application-analysis-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)
- [Stored research sources](./sources/)

## Status

Current status: **active**. Research and design are complete; implementation is
not part of this ticket-creation turn and should begin with Phase 0 and Phase 1
after architecture review.

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
- sources/ - Preserved package, SQLite, and OWASP source material
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
