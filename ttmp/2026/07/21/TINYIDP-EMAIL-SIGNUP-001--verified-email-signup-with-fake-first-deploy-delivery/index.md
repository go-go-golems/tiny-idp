---
Title: Verified email signup with fake first-deploy delivery
Ticket: TINYIDP-EMAIL-SIGNUP-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - security
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation plan for durable verified-email signup, initially delivered through a private operator outbox and later switched to the personal SMTP server without changing workflow semantics."
LastUpdated: 2026-07-21T16:09:42.923879149-04:00
WhatFor: "Track the minimum work needed to activate TinyIDP's existing durable email-challenge workflow in production while using an explicitly temporary first-deploy delivery mechanism."
WhenToUse: "Use before changing production signup validation, mail delivery, email challenge construction, the shared two-application signup program, or acceptance tests."
---

# Verified email signup with fake first-deploy delivery

## Overview

This ticket activates TinyIDP's existing durable email-code challenge for account creation. Message Desk remains open admission and goja-auth remains invitation-gated, but both paths prove email-code possession before the native account transaction sets `email_verified=true`.

The first deployment uses a private operator-only mail catcher. This is a delivery substitution, not a second verification implementation: TinyIDP still generates, hashes, binds, expires, attempts, resends, verifies, and consumes the real challenge. An operator retrieves the message from the protected outbox and relays the code to the intended user. A later deployment changes only the SMTP destination to the personal mail server.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **designed; implementation not started**

## Topics

- oidc
- identity
- auth
- security
- testing

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
