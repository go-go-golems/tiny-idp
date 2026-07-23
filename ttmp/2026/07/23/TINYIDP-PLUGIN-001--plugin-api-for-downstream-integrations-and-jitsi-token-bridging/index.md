---
Title: Plugin API for downstream integrations and Jitsi token bridging
Ticket: TINYIDP-PLUGIN-001
Status: active
Topics:
    - architecture
    - auth
    - jitsi
    - operations
    - security
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/07/09/TINYIDP-JITSI-001--use-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet/design-doc/01-tiny-idp-as-an-oidc-identity-provider-for-jitsi-meet-analysis-design-and-implementation-guide.md
      Note: Prior accepted protocol and adapter research
    - Path: repo://ttmp/2026/07/23/TINYIDP-JITSI-K3S-001--deploy-jitsi-meet-on-hetzner-k3s-with-tinyidp-authentication/analysis/01-jitsi-meet-on-hetzner-k3s-prior-research-cluster-fit-and-deployment-boundary.md
      Note: Current deployment boundary and GitOps context
ExternalSources:
    - https://jitsi.github.io/handbook/docs/devops-guide/token-authentication/
    - https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md
    - https://github.com/jitsi-contrib/jitsi-oidc-adapter
    - https://pkg.go.dev/plugin
    - https://github.com/hashicorp/go-plugin
Summary: Exploratory architecture research for compiled-in TinyIDP plugins, Glazed configuration, bounded Goja policy, and production operations.
LastUpdated: 2026-07-23T16:32:24.782740888-04:00
WhatFor: ""
WhenToUse: ""
---




# Plugin API for downstream integrations and Jitsi token bridging

## Overview

This ticket explores how TinyIDP could host optional downstream integrations,
using a Jitsi token bridge as the first concrete plugin. It is currently an
option-selection notebook, not the final implementation guide.

The plugin would authenticate a TinyIDP browser identity, optionally invoke a
bounded Goja policy, mint a narrowly scoped Jitsi JWT, and redirect the browser
to Jitsi. Prosody remains deployed and validates the token.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- architecture
- auth
- jitsi
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
