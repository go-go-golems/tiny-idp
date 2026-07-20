---
Title: Production tiny-idp review for multi-user xgoja applications and coding agents
Ticket: TINYIDP-PROD-XGOJA-REVIEW-001
Status: complete
Topics:
    - architecture
    - auth
    - identity
    - oauth2
    - oidc
    - operations
    - research
    - security
    - testing
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Evidence-backed production-readiness review of tiny-idp as the identity plane for multi-user xgoja web applications and OAuth device-authorized coding agents.
LastUpdated: 2026-07-18T15:28:59.357516579-04:00
WhatFor: Orient implementers and reviewers, identify production gaps, and define an actionable target architecture.
WhenToUse: When evaluating, implementing, deploying, or reviewing tiny-idp-backed xgoja applications and coding-agent API access.
---




# Production tiny-idp review for multi-user xgoja applications and coding agents

## Overview

This ticket reviews the current `tiny-idp` implementation and its neighboring
`go-go-goja`/xgoja application surfaces as one production identity system. It
connects interactive browser signup and login, OIDC authorization, application
sessions, API access-token validation, and OAuth device authorization for coding
agents. The primary deliverable is an intern-facing architecture and code review
guide that separates observed implementation behavior from proposed production
design. A second design narrows the initial implementation to one standalone
tiny-idp and one signup-enabled Message Desk deployment on the existing k3s
platform; device authorization, xgoja, and multiple applications are deferred.

## Key Links

- [Production IdP architecture and code review guide](design-doc/01-production-idp-architecture-and-code-review-guide-for-xgoja-applications-and-coding-agents.md)
- [Initial k3s design for standalone tiny-idp and Message Desk](design-doc/02-initial-k3s-deployment-design-for-standalone-tiny-idp-and-message-desk.md)
- [PR 98 production hardening implementation guide](design-doc/03-pr-98-production-hardening-implementation-guide-for-xgoja-hostauth.md)
- [Investigation diary](reference/01-investigation-diary.md)
- **Related Files**: See the frontmatter `RelatedFiles` field.
- **External Sources**: See the frontmatter `ExternalSources` field.

## Status

Current status: **active**

## Topics

- architecture
- auth
- identity
- oauth2
- oidc
- operations
- research
- security
- testing
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
