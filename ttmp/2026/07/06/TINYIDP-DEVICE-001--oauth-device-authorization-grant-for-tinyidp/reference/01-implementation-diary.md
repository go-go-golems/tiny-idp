---
Title: Implementation Diary
Ticket: TINYIDP-DEVICE-001
Status: active
Topics:
    - oidc
    - testing
    - identity
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/design-doc/01-device-authorization-grant-design-and-implementation-guide.md
      Note: Primary device authorization design guide created in Step 1
ExternalSources:
    - "OAuth 2.0 Device Authorization Grant RFC 8628: https://www.rfc-editor.org/rfc/rfc8628"
Summary: Chronological diary for designing native OAuth device authorization grant support in tinyidp.
LastUpdated: 2026-07-06T00:00:00-04:00
WhatFor: Read before implementing tinyidp-native device authorization grant endpoints.
WhenToUse: Use when resuming TINYIDP-DEVICE-001 or reviewing device authorization design decisions.
---

# Diary

## Goal

Capture the design and delivery work for adding native OAuth 2.0 Device Authorization Grant support to tinyidp.

## Step 1: Create the ticket and design native device authorization

This step created a dedicated docmgr ticket for tinyidp-native device authorization grant support. The design distinguishes this new feature from the existing xgoja Step 08 flow: xgoja currently owns its own device authorization endpoints, while tinyidp supplies browser OIDC login. This ticket is for adding the OAuth device grant to tinyidp itself.

The output is an intern-facing implementation guide that explains RFC 8628 terminology, tinyidp's current server state and token dispatch architecture, the proposed device grant data model, new endpoints, polling states, approval UI, tests, docs, and validation plan. No runtime code changed in this step.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for implementing device authorization grant, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new ticket in the tinyidp repository for implementing OAuth 2.0 Device Authorization Grant support, write a detailed intern-friendly design and implementation guide, store it in the ticket, and upload the design package to reMarkable.

**Inferred user intent:** The user wants the device-login gap turned into an executable design package before implementation begins, with enough context that an intern can understand tinyidp, RFC 8628, and the affected files.

**Commit (code):** N/A — documentation-only ticket creation step.

### What I did

- Created `TINYIDP-DEVICE-001` under the tinyidp repo `ttmp` root.
- Added `design-doc/01-device-authorization-grant-design-and-implementation-guide.md`.
- Replaced the default task list with a detailed phase-by-phase implementation checklist.
- Wrote this diary entry.
- Used existing source evidence from:
  - `internal/server/server.go`,
  - `internal/server/token.go`,
  - `internal/server/jwt.go`,
  - `internal/server/authorize.go`,
  - `internal/scenario/seeded_users.go`,
  - xgoja Step 08 docs and smoke helpers.

### Why

- tinyidp currently supports browser OIDC flows but not OAuth Device Authorization Grant.
- xgoja Step 08 proves a device-style flow can coexist with tinyidp login, but that flow is app-owned and should not be mistaken for tinyidp-native device support.
- A design ticket prevents device support from being implemented as ad hoc token-endpoint branches without approval state, polling semantics, discovery metadata, docs, or tests.

### What worked

- The existing tinyidp architecture has natural insertion points:
  - `Server` can hold `deviceGrants` beside codes, tokens, sessions, and refresh tokens.
  - `registerRoutesAt` can mount `/device_authorization` and `/device` under both root and path issuers.
  - `/token` already dispatches on grant type.
  - seeded-user fixture passwords can authenticate approval form submissions.

### What didn't work

- No command failures occurred in this step.

### What I learned

- The important design distinction is not whether a flow has a device code; it is which component owns the device authorization server. xgoja Step 08 owns one today. This ticket would add one to tinyidp.
- The existing scenario registry should remain the identity and claim boundary for approving users.

### What was tricky to build

- The tricky part is deciding how much browser-session behavior to reuse for approval. The guide proposes a direct `/device` login/password approval form first because it is deterministic and testable. Reusing existing IdP sessions can be a later enhancement.
- Another subtle point is token issuance. Device polling should return an ID token when the approved scope includes `openid`, and should reuse existing refresh-token behavior when the scope includes `offline_access`.

### What warrants a second pair of eyes

- Review whether `/device_authorization` and `/device` are the preferred endpoint paths.
- Review whether direct login/password approval is acceptable for the first implementation or whether it should reuse the existing IdP session cookie immediately.
- Review the polling policy for `slow_down`: the guide proposes returning `slow_down` when the client polls more frequently than the grant interval.
- Review whether device grants should be deleted immediately on denial/expiry or retained briefly for debug visibility.

### What should be done in the future

- Upload the design bundle to reMarkable.
- Run `docmgr doctor` and commit the ticket package.
- Implement Phase 1 data model and route skeleton before changing token issuance.

### Code review instructions

- Start with `design-doc/01-device-authorization-grant-design-and-implementation-guide.md`.
- Pay special attention to the endpoint design, token polling error matrix, implementation phases, and testing plan.

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp
```
