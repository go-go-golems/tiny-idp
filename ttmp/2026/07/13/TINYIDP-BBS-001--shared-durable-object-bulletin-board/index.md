---
Title: Shared Durable Object Bulletin Board
Ticket: TINYIDP-BBS-001
Status: active
Topics:
    - architecture
    - xgoja
    - identity
    - security
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation of a shared, identity-aware bulletin board backed by one go-go-objects Durable Object and served by trusted xgoja routes."
LastUpdated: 2026-07-13T16:27:18.015615966-04:00
WhatFor: "Track the architecture, implementation, verification, and delivery of the tinyidp-xapp bulletin board feature."
WhenToUse: "Use when changing the BBS API, shared-object boundary, board schema, React client, security tests, or deployment workflow."
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
