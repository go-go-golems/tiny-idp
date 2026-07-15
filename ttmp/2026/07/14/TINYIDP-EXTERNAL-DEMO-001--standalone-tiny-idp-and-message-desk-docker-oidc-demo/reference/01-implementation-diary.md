---
Title: Implementation diary
Ticket: TINYIDP-EXTERNAL-DEMO-001
Status: active
Topics:
    - architecture
    - go
    - identity
    - oidc
    - oauth2
    - docker
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T21:58:07.711626152-04:00
WhatFor: ""
WhenToUse: ""
---

# Implementation diary

## Goal

Record the design investigation for a standalone tiny-idp and Message Desk
Docker reference deployment, including its trust boundaries and implementation
sequence.

## Step 1: Establish the external-issuer reference architecture

The requested demo is not a cosmetic repackaging of the embedded example. It
must prove that Message Desk is a normal OIDC relying party when tiny-idp is a
separate process and origin. The design therefore preserves the browser and
back-channel protocol mechanics while removing all direct provider and account
service dependencies from the application container.

### Prompt Context

**User prompt (verbatim):** "Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create an intern-ready design package for a
two-container standalone tiny-idp and Message Desk demo and deliver it through
the ticket and reMarkable.

**Inferred user intent:** Provide a reusable, honest reference deployment for
external tiny-idp integration rather than requiring every application to embed
the provider binary.

### What I did

- Created `TINYIDP-EXTERNAL-DEMO-001`, its design document, diary, and seven
  implementation phases.
- Mapped current embedded seams in `commands.go`, `oidc_client.go`, and
  `app_http.go`.
- Wrote the two-origin topology, OIDC flow, configuration contracts, visual
  renderer boundary, seeded-account policy, logout scopes, validation plan,
  decisions, and intern onboarding checklist.

### Why

- A separate provider is useful only when its database, cookies, UI, and
  lifecycle remain provider-owned. Sharing embedded internals would hide the
  deployment boundary the example is intended to teach.

### What worked

- The existing Message Desk client already accepts an arbitrary issuer and
  `*http.Client`; its PKCE/state/nonce/ID-token verification path is reusable.

### What didn't work

- N/A. No implementation experiment was run in this design step.

### What I learned

- The essential new work is composition and operational validation, not a new
  OAuth client implementation. Registration is the one deliberate functional
  omission because it currently depends on direct embedded account APIs.

### What was tricky to build

- The browser needs public canonical origins while containers use private DNS.
  These must not be confused: issuer discovery, redirects, and token `iss`
  checks use public URLs; Docker service names are only internal transport
  addresses.

### What warrants a second pair of eyes

- Review development HTTP versus production HTTPS/cookie policy and the seed
  credential lifecycle before implementation begins.

### What should be done in the future

- Implement phases 1–7 in order; create a separate ticket if a public
  tiny-idp self-registration API is desired.

### Code review instructions

- Start with `examples/tinyidp-message-app/oidc_client.go` and compare its
  transport injection to the proposed standard HTTPS client.

### Technical details

```text
Message Desk owns: app SQLite, app session, PKCE/state/nonce, messages.
tiny-idp owns: identity SQLite, accounts, IdP cookies, login/consent/chooser.
```
