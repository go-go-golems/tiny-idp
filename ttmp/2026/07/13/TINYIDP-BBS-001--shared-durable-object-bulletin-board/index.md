---
Title: Shared Durable Object Bulletin Board
Ticket: TINYIDP-BBS-001
Status: complete
Topics:
    - architecture
    - xgoja
    - identity
    - security
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/frontend/src/App.tsx
      Note: Typed browser UX and distinct logout scopes
    - Path: repo://cmd/tinyidp-xapp/app/frontend/src/api.ts
      Note: RTK Query transport and CSRF header policy
    - Path: repo://cmd/tinyidp-xapp/app/frontend/src/styles.css
      Note: Responsive early-Mac monochrome visual system
    - Path: repo://cmd/tinyidp-xapp/app/objects/objects.js
      Note: Shared persistent BBS state machine
    - Path: repo://cmd/tinyidp-xapp/app/routes/site.js
      Note: Trusted fixed-object BBS HTTP API and actor projection
    - Path: repo://cmd/tinyidp-xapp/bbs_test.go
      Note: Object validation ownership and restart invariants
    - Path: repo://internal/fositeadapter/end_session.go
      Note: Strict current-browser RP-initiated logout
    - Path: repo://internal/fositeadapter/end_session_test.go
      Note: Redirect revocation cookie and audit tests
    - Path: repo://ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/scripts/01_real_browser_bbs.py
      Note: Two-user TLS browser restart and logout harness
ExternalSources: []
Summary: Design and implementation of a shared, identity-aware bulletin board backed by one go-go-objects Durable Object and served by trusted xgoja routes.
LastUpdated: 2026-07-13T17:29:32.242051901-04:00
WhatFor: Track the architecture, implementation, verification, and delivery of the tinyidp-xapp bulletin board feature.
WhenToUse: Use when changing the BBS API, shared-object boundary, board schema, React client, security tests, or deployment workflow.
---



# Shared Durable Object Bulletin Board

## Overview

This ticket turns `tinyidp-xapp` from an identity-plus-private-scratchpad
demonstration into a useful multi-user bulletin board. The application keeps
tiny-idp responsible for authentication, uses trusted xgoja routes for the
public HTTP API, stores board state in one `BBS/community` Durable Object, and
ships a React interface in the generated single binary.

The central security rule is that browser input never selects a Durable Object
namespace, object name, or actor identity. Trusted route code fixes the object
identity and derives authorship from the authenticated application session.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- [Analysis, design, and implementation guide](./design-doc/01-shared-durable-object-bbs-analysis-design-and-implementation-guide.md)
- [Implementation diary](./reference/01-implementation-diary.md)
- [Verification and operations playbook](./playbook/01-bbs-verification-and-operations-playbook.md)

## Status

Current status: **active**

## Topics

- architecture
- xgoja
- identity
- security
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
