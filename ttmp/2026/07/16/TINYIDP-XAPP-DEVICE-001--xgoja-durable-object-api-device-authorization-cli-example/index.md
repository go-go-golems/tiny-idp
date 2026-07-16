---
Title: xgoja Durable Object API Device Authorization CLI Example
Ticket: TINYIDP-XAPP-DEVICE-001
Status: complete
Topics:
    - auth
    - oidc
    - oauth2
    - xgoja
    - durable-objects
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Development-only identity-store seam used to test real password-security token invalidation.
    - Path: repo://cmd/tinyidp-xapp/device_cli.go
      Note: Device authorization CLI, polling state machine, and owner-only token cache.
    - Path: repo://cmd/tinyidp-xapp/device_cli_test.go
      Note: Deterministic protocol error, cache, and request-formation verification.
    - Path: repo://cmd/tinyidp-xapp/phase5_test.go
      Note: Two-subject, denied-dispatch, password-revocation, malformed-request, and TLS application matrix.
    - Path: repo://ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/scripts/playwright_browser_smoke.py
      Note: Local Playwright browser login/post/logout regression harness.
    - Path: repo://ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/scripts/run-xapp-device-smoke.sh
      Note: tmux live-server operator harness.
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-16T15:37:41.04965256-04:00
WhatFor: ""
WhenToUse: ""
---

# xgoja Durable Object API Device Authorization CLI Example

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **complete** — all planned phases have implementation and verification evidence. The runbook records the deployment limits that must be addressed by a deployment-specific operations design.

## Topics

- auth
- oidc
- oauth2
- xgoja
- durable-objects
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
