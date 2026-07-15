---
Title: Production Deployment Ergonomics Analysis Design and Implementation Guide
Ticket: TINYIDP-PROD-DEPLOY-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - operations
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/production-host/systemd/tinyidp.service
      Note: Least-privilege service-manager template
    - Path: repo://internal/cmds/serve.go
      Note: Local-only command renamed to serve-dev
    - Path: repo://internal/cmds/serve_production.go
      Note: Strict durable HTTPS host prerequisites
    - Path: repo://pkg/embeddedidp/options.go
      Note: Final strict production validation boundary
    - Path: repo://pkg/sqlitestore/store.go
      Note: SQLite single-active-node durability contract
ExternalSources: []
Summary: Operator-facing design for provisioning and running tinyidp's durable strict host safely and repeatably.
LastUpdated: 2026-07-15T18:05:00Z
WhatFor: Explain the strict-host operational contract, tooling, and release gates to an engineer new to tinyidp.
WhenToUse: Read before modifying server commands, provisioning, deployment assets, readiness checks, or release gates.
---


# Production Deployment Ergonomics Analysis Design and Implementation Guide

## Executive Summary

`tinyidp` now has two deliberately different server commands. `serve-dev` is
local-only development infrastructure; `serve-production` is a durable direct
TLS host assembled through the public `pkg/embeddedidp` API. This document
turns the latter command's prerequisites into a reviewable operator workflow.

## Problem Statement

The repository already had durable SQLite, an administrative lifecycle,
owner-only token-secret validation, signing-key validation, audit delivery,
bounded HTTP requests, and readiness. What it lacked was a compact path that
showed how these pieces must be assembled before binding a public identity
endpoint. The old `serve` name also obscured the fact that it defaulted to a
mock, in-memory development server.

The workflow has to prove more than a rendered login page. For device
authorization, the browser decision, token redemption, UserInfo validation,
replay handling, and continued host responsiveness are one security contract.

## Proposed Solution

The resulting command and state model is:

```text
tinyidp serve-dev
  local mock or in-memory strict preview; loopback only

tinyidp admin --db <database> init / user / client / doctor
  persistent schema, signing key, accounts, grants, preflight

tinyidp serve-production --issuer https://idp.example ...
  durable strict host with direct TLS, audit, maintenance, readiness
```

The local smoke scripts use the same public commands as a real operator. They
create a fresh owner-only directory, a 32-byte token secret, a one-day
localhost certificate, a SQLite database, an active RS256 key, one password
account, and a public device client with an explicit device-code capability.
They refuse to reuse existing fixture files. The launcher runs in the
foreground so a human or CI harness chooses the supervisor; for interactive
work, start it in tmux.

An Internet-facing host must use a certificate whose SAN covers the canonical
HTTPS issuer. Direct TLS is supported today. A reverse proxy may only forward
to the host over TLS until a separate trusted forwarded-origin design exists;
the current proxy option resolves client addresses, not request origin.

## Design Decisions

### Command naming

- `serve-dev` makes the development-only contract explicit.
- `serve-production` remains explicit because production identity operations
  benefit from a hard-to-mistake command.
- No `serve` compatibility alias is retained. A silent compatibility layer
  would preserve the unsafe ambiguity the rename removes.

### State and secrets

- The host accepts a token-secret file path, never a token secret CLI value.
- A regular owner-only file with at least 32 bytes is required.
- `admin init --generate-signing-key` owns initial signing-key creation.
- Clients declare every allowed grant. Browser clients need
  `authorization_code` (and normally `refresh_token`); device clients declare
  `urn:ietf:params:oauth:grant-type:device_code`.
- SQLite is one active process on a local filesystem, not a shared-volume HA
  datastore.

### Operational signals

`/healthz` indicates process liveness. `/readyz` is a traffic-admission signal
and includes store/key/audit policy checks. Operators should restart for
liveness failure, not a transient readiness failure. The audit JSONL file is
a security record and needs collection, rotation, and retention.

## Alternatives Considered

1. **Add more flags to `serve`.** Rejected: a mock default and development
   state must not share an ambiguous production command.
2. **Generate users, clients, and secrets at startup.** Rejected: identity
   state would change across restarts and be hard to audit.
3. **Trust generic forwarded headers.** Rejected: proxy termination requires
   an explicit trust-boundary design, not a convenience toggle.
4. **Call browser approval a successful device smoke.** Rejected: the current
   observed post-approval hang shows why redemption and responsiveness must be
   part of the gate.

## Implementation Plan

1. Rename development serving and update maintained docs.
2. Ship finite local provisioning, launch, and TLS-verifying probe scripts.
3. Ship a least-privilege service-manager template.
4. Run strict-host browser and token smoke tests and retain redacted evidence.
5. Reproduce and fix the observed post-approval unresponsiveness before
   advertising device authorization as production-ready.
6. Add locally trusted browser automation, backup/restore drills, and external
   review before release.

## Open Questions

The post-approval strict-host unresponsiveness is a release blocker. It needs
a retained goroutine dump, a regression test using the production SQLite
configuration, and a narrow fix. Container manifests are deferred until the
supported TLS/proxy topology is settled; a sample must not imply unsupported
active/active operation.

## References

- `internal/cmds/serve_production.go` — strict host and required flags.
- `internal/cmds/serve.go` — renamed local development command.
- `internal/cmds/admin*.go` — durable provisioning lifecycle.
- `pkg/embeddedidp/options.go` — final strict production validation boundary.
- `pkg/sqlitestore/store.go` — SQLite single-connection contract.
- `docs/security-profile.md` and `examples/production-host/README.md`.
- `TINYIDP-DEVICE-PROD-001` — device state-machine work and evidence.
