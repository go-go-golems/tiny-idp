---
Title: Professional signup invitations and application membership invitations
Ticket: TINYIDP-INVITES-001
Status: complete
Topics:
    - oidc
    - identity
    - auth
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design ticket for a minimal production-grade TinyIDP signup-invitation core and a separate atomic go-go-goja organization-membership invitation flow."
LastUpdated: 2026-07-21T15:15:38-04:00
WhatFor: "Track the design and implementation phases needed to ship invite-gated identities and application memberships without coupling the two authority domains."
WhenToUse: "Use to orient implementation work and find the detailed design, diary, tasks, and change history."
---

# Professional signup invitations and application membership invitations

## Overview

This ticket defines two deliberately separate invitation layers: TinyIDP controls who may create an identity, while each application controls who may join its organizations and with which role. The user journey may pass through both layers, but their data and commits remain owned by separate services.

The design reuses existing durable TinyIDP invitation and go-go-goja capability primitives. Work is concentrated on production activation, safe operator commands, atomic application membership acceptance, OIDC pending-state orchestration, and local browser validation. Generic storage APIs, raw JavaScript database access, and a broad administration UI are explicitly outside the initial scope.

## Key documents

- [Professional invitation core and application membership invitation design and implementation guide](./design-doc/01-professional-invitation-core-and-application-membership-invitation-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Implementation tasks](./tasks.md)
- [Changelog](./changelog.md)

## Current status

All ticket tasks through local Phase 5 are complete. TinyIDP production invitation activation, the concrete per-client signup workflow, atomic go-go-goja membership acceptance, safe OIDC continuation, deterministic local bootstrap, and the full browser acceptance suite are implemented and passing. k3s and GitOps remain deliberately outside this ticket's completed scope.

## Core decision

```text
TinyIDP signup invitation              Application organization invitation
--------------------------------      -----------------------------------
permits identity creation             permits membership creation
owned by TinyIDP database             owned by application database
consumed with account creation        consumed with membership insertion
audience is OIDC client               resource is application organization
contains no app role                  contains app-owned role
```

## Topics

- OIDC authentication and registration intent
- TinyIDP Goja workflows and explicit continuations
- durable, hashed, expiring, revocable single-use invitations
- go-go-goja capabilities and application membership
- transaction boundaries and retryable cross-service sagas
- local Compose acceptance before production deployment

## Structure

- `design-doc/` — architecture, APIs, pseudocode, diagrams, phases, and tests
- `reference/` — chronological research diary and continuation evidence
- `tasks.md` — precise phase and task tracking
- `changelog.md` — completed documentation and implementation milestones
