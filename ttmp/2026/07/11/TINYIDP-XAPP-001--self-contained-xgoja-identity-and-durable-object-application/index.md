---
Title: Self-Contained xgoja Identity and Durable Object Application
Ticket: TINYIDP-XAPP-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - identity
    - oidc
    - research
    - testing
    - xgoja
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/xgoja.yaml
      Note: Generated runtime and separate embedded-asset marker contract
    - Path: repo://internal/fositeadapter/csrf.go
      Note: Configured CSRF cookie reads, writes, and issuer-derived path
    - Path: repo://internal/fositeadapter/provider.go
      Note: Provider-local cookie configuration and direct-caller validation
    - Path: repo://internal/fositeadapter/session.go
      Note: Configured browser-session cookie reads and writes
    - Path: repo://internal/fositeadapter/session_test.go
      Note: End-to-end host-session and IdP-cookie coexistence proof
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public cookie ownership and issuer-path validation for combined hosts
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Transfers the embedding cookie contract into the protocol adapter
ExternalSources: []
Summary: Design and implementation program for a self-contained xgoja application combining tiny-idp, Express planned routes, an embedded frontend, and authenticated actor-bound Durable Objects.
LastUpdated: 2026-07-11T18:45:00-04:00
WhatFor: Track architecture, cross-repository implementation, security invariants, operational work, validation, and production release evidence.
WhenToUse: Use when implementing or reviewing the integrated product host, OIDC boundary, app session, actor/object binding, xgoja runtime, frontend, persistence, or release gates.
---


# Self-Contained xgoja Identity and Durable Object Application

## Overview

This ticket turns three tested building-block repositories into one self-contained single-node product. The browser authenticates through an embedded tiny-idp issuer, receives a separate opaque application session, and accesses only the durable object derived by the Go host from its authenticated application actor.

The design is complete enough to implement. The first cross-repository security foundations are committed: provider-neutral OIDC with same-process back-channel transport, actor propagation for trusted native services, issuer-scoped application identities, HMAC actor-bound Durable Objects, xgoja actor-bound calls, and default-denied raw object gateways. Product-host and persistence work remains.

## Key Links

- [Intern-ready analysis, design, and implementation guide](./design-doc/01-self-contained-xgoja-tiny-idp-express-and-durable-objects-analysis-design-and-implementation-guide.md)
- [Investigation and implementation diary](./reference/01-investigation-diary.md)
- [Detailed phased tasks](./tasks.md)
- [Source packet](./sources/)

## Status

Current status: **active**

## Topics

- architecture
- auth
- go
- identity
- oidc
- research
- testing
- xgoja

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design-doc/ - Architecture, analysis, and implementation guide
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- sources/ - Preserved primary local API and design references
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
