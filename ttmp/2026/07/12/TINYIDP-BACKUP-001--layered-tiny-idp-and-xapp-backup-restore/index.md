---
Title: Layered Tiny-IDP and XAPP Backup Restore
Ticket: TINYIDP-BACKUP-001
Status: active
Topics:
    - identity
    - architecture
    - testing
    - go
    - security
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/state.go
      Note: Current product state-root manifest, paths, permissions, and initialization contract
    - Path: repo://pkg/sqlitestore/store.go
      Note: General tiny-idp persistent SQLite implementation and backup boundary
    - Path: ws://go-go-goja/pkg/xgoja/hostauth/stores.go
      Note: Application auth/session/audit/capability store ownership
    - Path: ws://go-go-objects/pkg/durableobjects/manager.go
      Note: Durable Object actor lifecycle and quiescence boundary
ExternalSources: []
Summary: Track the layered component backup APIs and whole-product XAPP backup, verification, restore, and recovery drills without duplicating subsystem logic.
LastUpdated: 2026-07-12T17:53:41.683660441-04:00
WhatFor: Planning and implementing recoverable backups across tiny-idp, hostauth, Durable Objects, and the combined XAPP state root.
WhenToUse: Use before changing snapshot formats, adding backup commands, implementing restore, or running a recovery drill.
---


# Layered Tiny-IDP and XAPP Backup Restore

## Overview

This ticket records future work for layered backup and restore. Each subsystem
owns its snapshot format and validation. `tinyidp-xapp` owns the global
quiescence boundary, product manifest, shared secrets, and cross-component
reconciliation.

This ticket is intentionally planning-only while real-application browser,
lifecycle, and fault-injection work proceeds in `TINYIDP-XAPP-001`.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

Design baseline written; implementation deferred.

## Topics

- identity
- architecture
- testing
- go
- security
- xgoja

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
