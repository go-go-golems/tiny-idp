---
Title: Shared tiny-idp themes and second application on k3s
Ticket: TINYIDP-MULTIAPP-THEMES-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - architecture
    - operations
    - security
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Evidence-backed design for turning the current one-client production host into a shared IdP with GitOps-mounted, per-client same-origin themes and a separately deployed second relying party. Includes an operator research branch."
LastUpdated: 2026-07-21T12:53:00-04:00
WhatFor: "Plan the first shared tiny-idp deployment and its second browser application without giving applications control over IdP protocol behavior or asset origins."
WhenToUse: "Use before implementing multi-client production configuration, theme assets, a second k3s application, or a TinyIDP Kubernetes operator."
---

# Shared tiny-idp themes and second application on k3s

## Overview

This ticket turns the successful standalone MessageDesk deployment into the first
shared TinyIDP deployment. It defines two linked changes: an IdP-owned theme
catalog mounted from reviewed GitOps configuration, and a separately deployed
second browser application registered with the same issuer.

The primary design document is written for an intern who has not worked with
TinyIDP, OpenID Connect, k3s, Argo CD, or the current deployment. It also
contains a deliberately separate investigation of a TinyIDP Kubernetes
operator. The operator is not a prerequisite for the first two-app rollout.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **Primary guide**: [shared IdP and themes design](./design-doc/01-shared-tiny-idp-theme-assets-and-a-second-application-on-k3s-analysis-design-and-implementation-guide.md)
- **Investigation record**: [diary](./reference/01-investigation-diary.md)
- **Production runbook**: [client and theme operations](./reference/02-production-client-and-theme-runbook.md)
- **Acceptance evidence**: [two-app production evidence](./reference/03-production-acceptance-evidence.md)
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active** — the GitOps-only two-app rollout is complete; the
separate Kubernetes-operator research branch remains an explicit follow-up.

## Topics

- oidc
- identity
- auth
- architecture
- operations
- security

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
