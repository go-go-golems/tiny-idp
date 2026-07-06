---
Title: Implementation Diary
Ticket: TINYIDP-DPOP-001
Status: active
Topics:
  - oidc
  - auth
  - identity
  - testing
  - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DPOP-001--dpop-sender-constrained-tokens-for-tinyidp/design-doc/01-dpop-design-and-implementation-guide.md
    Note: Primary DPoP design and implementation guide created in Step 1
ExternalSources:
  - "RFC 9449: OAuth 2.0 Demonstrating Proof of Possession (DPoP): https://www.rfc-editor.org/rfc/rfc9449.html"
Summary: Chronological diary for designing and implementing DPoP sender-constrained tokens in tinyidp.
LastUpdated: 2026-07-06T00:00:00-04:00
WhatFor: Read before resuming TINYIDP-DPOP-001 or reviewing DPoP implementation decisions.
WhenToUse: Use when implementing DPoP proof validation, token binding, replay protection, docs, or smoke tests.
---

# Diary

## Goal

Capture the design and implementation journey for adding DPoP sender-constrained token support to tinyidp.

## Step 1: Create the DPoP ticket and design the implementation

This step created a dedicated docmgr ticket for DPoP support and wrote the intern-facing design guide. The guide explains what DPoP changes in tinyidp's OAuth model: token issuance remains grant-driven, but access-token use becomes proof-of-possession-bound when a token request includes a valid `DPoP` proof JWT.

The design stays within tinyidp's local/test identity-provider scope. It keeps opaque tokens, stores the JWK thumbprint as in-memory token metadata, validates proof JWTs with Go standard-library crypto, and defers optional nonce support to a future enhancement.

### Prompt Context

**User prompt (verbatim):** "close the ticket, now implement dpop (I think there's a ticket already? But probably you need a full design / implementation guide: Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.
)"

**Assistant interpretation:** Confirm the previous device ticket is closed, create or locate a DPoP ticket, write a detailed intern-facing DPoP design/implementation guide, store it in the ticket, upload it to reMarkable, and then proceed toward implementation.

**Inferred user intent:** The user wants DPoP support handled with the same documentation discipline as device authorization: ticketed design, detailed implementation guide, diary, validation, and published review artifacts.

**Commit (code):** pending — design ticket creation step.

### What I did

- Confirmed `TINYIDP-DEVICE-001` is complete with zero open tasks.
- Searched existing docmgr tickets and found no DPoP ticket.
- Created `TINYIDP-DPOP-001` under the tinyidp repo `ttmp` root.
- Added `design-doc/01-dpop-design-and-implementation-guide.md`.
- Replaced the default task list with a phase-by-phase DPoP checklist.
- Wrote this diary entry.
- Used source evidence from:
  - `internal/server/server.go`,
  - `internal/server/token.go`,
  - `internal/server/userinfo.go`,
  - `internal/server/jwt.go`,
  - `internal/client/client.go`,
  - the recently implemented device authorization tests.

### Why

- DPoP has enough moving parts that implementation without a design would be easy to get subtly wrong: proof JWT parsing, JWK thumbprints, `htm`/`htu`, `iat`, replay protection, `ath`, token binding, refresh-token binding, and resource enforcement.
- tinyidp's opaque-token architecture means the design must choose between JWT access tokens and server-side metadata. The guide chooses server-side `DPoPJKT` metadata to preserve existing tinyidp behavior.

### What worked

- The existing tinyidp token helpers created during device authorization are a good insertion point for DPoP binding.
- `/userinfo` is a compact protected-resource endpoint, which makes it a good first enforcement surface for DPoP-bound access tokens.
- RFC 9449 maps cleanly onto tinyidp's in-memory state model when nonce support is deferred.

### What didn't work

- No command failures occurred in this step.

### What I learned

- The critical implementation boundary is not the access token string itself. It is the metadata stored with the opaque token: user, scenario, expiry, and now the DPoP JWK thumbprint.
- The replay cache should be keyed by proof key thumbprint and `jti`, not by token, because the proof JWT is the replayed object.

### What was tricky to build

- The design had to decide how to handle unbound refresh tokens when a refresh request includes a DPoP proof. The proposed behavior allows that request to upgrade the newly issued tokens to DPoP-bound tokens while still requiring matching proof for already-bound refresh tokens.
- The guide also had to distinguish DPoP from device authorization. Device authorization answers how a token is obtained. DPoP answers which private key may use that token.

### What warrants a second pair of eyes

- Review whether first-phase support for both `ES256` and `RS256` is worth the extra parser/test surface, or whether `ES256` alone is enough.
- Review whether unbound refresh tokens should be allowed to upgrade into DPoP-bound tokens or whether binding should only start at initial grant exchange.
- Review whether nonce support should remain deferred.

### What should be done in the future

- Implement the design phases.
- Upload the design bundle to reMarkable before or alongside implementation artifacts.
- Add a small manual DPoP smoke so reviewers can exercise proof generation and userinfo enforcement without writing a full RP.

### Code review instructions

- Start with `design-doc/01-dpop-design-and-implementation-guide.md`.
- Review the data model changes, proof validation algorithm, token endpoint behavior, and userinfo enforcement rules before reviewing code.

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DPOP-001--dpop-sender-constrained-tokens-for-tinyidp
```
