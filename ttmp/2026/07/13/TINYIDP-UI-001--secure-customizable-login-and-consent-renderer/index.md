---
Title: Secure Customizable Login and Consent Renderer
Ticket: TINYIDP-UI-001
Status: active
Topics:
    - oidc
    - identity
    - security
    - go
    - architecture
    - auth
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Design ticket for a host-supplied, writer-based login and consent renderer with provider-owned protocol state, security headers, CSP, and authorization decisions.
LastUpdated: 2026-07-13T17:38:38.771609633-04:00
WhatFor: Tracks design, implementation, assurance, and release work for secure HTML and CSS customization of strict tiny-idp browser interactions.
WhenToUse: Use when reviewing or implementing login/consent customization, xapp identity theming, interaction error pages, renderer analysis, or UI security testing.
---

# Secure Customizable Login and Consent Renderer

## Overview

tiny-idp's strict Fosite provider currently emits a minimal hard-coded login and
consent form. This ticket designs a public renderer contract that lets an
embedding host control HTML structure and same-origin CSS while tiny-idp retains
exclusive control of OAuth continuation, CSRF, authentication, consent,
security headers, cookies, response status, code issuance, and redirects.

The preferred API passes a typed `InteractionPage` and an `io.Writer` to trusted
host code. It intentionally does not pass `http.ResponseWriter`, `*http.Request`,
the original OAuth request, cookies, or stored interaction records. The xapp
will serve embedded theme assets under `/static/`; the provider will enforce a
fixed CSP and continue to reject scripts and framing.

The renderer, provider integration, xapp theme, conformance harness, static
analyzer, fuzzing, metrics, doctor check, and local browser canary are complete.
The remaining release gates are a production canary in its real TLS/proxy
topology and named human approval.

## Key Links

- [Primary analysis, design, and implementation guide](./design-doc/01-secure-interaction-rendering-analysis-design-and-implementation-guide.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Browser, accessibility, and local canary evidence](./reference/02-browser-accessibility-and-canary-evidence.md)
- [Release and rollback runbook](./reference/03-interaction-ui-release-and-rollback-runbook.md)
- [Detailed tasks](./tasks.md)
- [Preserved sources](./sources/)

## Status

Current status: **active**

- Phases 0–5: complete.
- Phase 6 implementation, documentation, doctor, and leakage review: complete.
- Phase 6 external production canary and named release approval: pending.
- Latest implementation checkpoint: `8e51f4b`.

## Topics

- oidc
- identity
- security
- go
- architecture
- auth

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
