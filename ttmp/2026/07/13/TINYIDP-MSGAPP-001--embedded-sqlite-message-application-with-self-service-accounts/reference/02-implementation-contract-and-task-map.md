---
Title: Implementation Contract and Task Map
Ticket: TINYIDP-MSGAPP-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - architecture
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-message-app/contracts.go
      Note: Executable names, route contract, and security invariant inventory.
    - Path: repo://examples/tinyidp-message-app/contracts_test.go
      Note: Enforces the public-package boundary and unique contract inventory.
ExternalSources: []
Summary: Accepted implementation decisions and task reconciliation for building the message application after the embedding foundations landed.
LastUpdated: 2026-07-14T19:00:00Z
WhatFor: Keeps the original design, current repository state, and implementation task order aligned.
WhenToUse: Read before starting or checking an implementation task.
---

# Implementation Contract and Task Map

## Accepted Phase 0 decisions

- Example directory and Go package: `examples/tinyidp-message-app`, `package main`.
- Application and client identifier: `tinyidp-message-app`.
- One public origin. The IdP is mounted at `/idp/`; the relying party owns `/`,
  `/auth/*`, `/api/*`, `/healthz`, `/readyz`, and `/static/app/*`.
- Public Authorization Code client with mandatory PKCE S256. There is no client
  secret and no refresh token in the first message-app release.
- State root contains separate `identity/tinyidp.sqlite` and
  `application/messages.sqlite` databases plus owner-only secrets and a
  versioned manifest.
- Application sessions are opaque, server-side, absolute eight-hour sessions.
  Only SHA-256 token hashes are stored.
- Registration returns stable public error shapes. Duplicate usernames remain
  behaviorally observable because accounts activate immediately; the
  limitation is documented rather than hidden.
- Messages are append-only in the first release. Owner deletion remains out of
  scope until the base authorization path is complete.
- The issuer transport has no fallback. Non-issuer origins fail closed.
- Unknown functional paths return 404. Static assets exist only below
  `/static/app/` and `/static/tinyidp/`; there is no unrestricted SPA fallback.
- The frontend uses pnpm, React, TypeScript, Redux Toolkit, RTK Query, Bootstrap,
  Vite, and `go:embed` as required by repository conventions.

## Inherited implementation

The original design's Phases 1 and 2 were implemented by
`TINYIDP-EMBED-FOUND-001` after this ticket was authored:

- `pkg/idpaccounts` provides atomic account creation, password authentication,
  replacement, policy enforcement, bounded password work, and audit behavior;
- `embeddedidp.Bootstrap` reconciles public browser clients and an active
  signing key without exposing private-key representation;
- `embeddedidp.NewInProcessIssuerTransport` dispatches only exact issuer URLs,
  bounds responses, propagates cancellation, and has no network fallback.

MSGAPP treats these APIs as prerequisites and verifies their focused packages.
It does not duplicate them or restore deleted internal adapters.

## Task tracking rule

The initial `tasks.md` tracked research only. The implementation tasks appended
on 2026-07-14 mirror Phases 0 through 8 in Section 29 of the design. A task is
checked only after its focused tests pass and the diary records the evidence.

## Executable contract inventory

`contracts.go` freezes route names, cookie names, and named invariant tests.
`contracts_test.go` parses every Go file in the example and fails if it imports
`github.com/manuel/tinyidp/internal/...`. This is intentionally an external
consumer boundary even though the example is stored in the same module.

The invariant names are implementation obligations, not claims of current test
coverage. Each later phase must add the actual test with the corresponding
name, or replace the inventory entry with an equivalently explicit test name
and update this document.
