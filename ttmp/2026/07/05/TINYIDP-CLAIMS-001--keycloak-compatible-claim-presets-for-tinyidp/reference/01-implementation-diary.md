---
Title: Implementation Diary
Ticket: TINYIDP-CLAIMS-001
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
    - Path: /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/05/TINYIDP-CLAIMS-001--keycloak-compatible-claim-presets-for-tinyidp/design-doc/01-keycloak-compatible-claim-presets-guide.md
      Note: Primary design guide created in Step 1
ExternalSources: []
Summary: Chronological diary for the Keycloak-compatible claim presets ticket.
LastUpdated: 2026-07-05T17:45:00-04:00
WhatFor: Read before implementing optional Keycloak-shaped claim preset expansion.
WhenToUse: Use when resuming TINYIDP-CLAIMS-001 or reviewing authorization claim compatibility decisions.
---

# Diary

## Goal

Capture the design and delivery work for optional Keycloak-shaped authorization claim presets in tinyidp.

## Step 1: Create the ticket and design optional Keycloak-shaped claims

This step created a dedicated ticket for Keycloak-compatible claim presets. The design is intentionally opt-in and narrow: it helps tests emit `realm_access`, `resource_access`, groups, and preferred username claims without claiming full Keycloak emulation.

The primary output is an intern-facing guide that explains when presets are needed, why simple login tests do not need them, how to model the `keycloak:` block, how preset and explicit claims should merge, and how to test ID token and userinfo output.

### Prompt Context

**User prompt (verbatim):** (same as TINYIDP-CONFIG-001 Step 1)

**Assistant interpretation:** Create a separate ticket for the fourth previously listed tinyidp usability item: optional Keycloak-shaped claim presets.

**Inferred user intent:** The user wants claim presets treated as optional authorization compatibility work, not as a prerequisite for basic OIDC smokes.

**Commit (code):** N/A — documentation-only ticket creation step.

### What I did

- Created `TINYIDP-CLAIMS-001` under the tinyidp repo `ttmp` root.
- Added design doc `design-doc/01-keycloak-compatible-claim-presets-guide.md`.
- Replaced the default task list with phased implementation tasks.
- Wrote this diary entry.
- Used existing source evidence from:
  - `internal/scenario/seeded_users.go`,
  - `internal/scenario/scenario.go`,
  - `internal/server/jwt.go`,
  - `internal/server/userinfo.go`.

### Why

- Current seeded users can already emit arbitrary nested claims through `claims`, but repeated Keycloak-shaped YAML is error-prone.
- Optional presets make role/group fixtures easier for apps that test Keycloak claim mapping.

### What worked

- The existing `Claims map[string]any` surface is flexible enough that presets can be implemented as expansion into `ExtraClaims`, not as a token-layer special case.

### What didn't work

- No failures occurred in this step.

### What I learned

- Claim presets are not needed for current personal-inbox login/session smokes. They are for authorization mapping tests.
- The most important guardrail is wording: document “Keycloak-shaped claims,” not “Keycloak emulation.”

### What was tricky to build

- The tricky part was defining merge semantics. The guide proposes preset expansion first, explicit `claims` second, and `omit_claims` last so fixture authors retain final control.

### What warrants a second pair of eyes

- Review whether explicit `claims` should override or deep-merge preset fields. The guide proposes override for simplicity.
- Review whether `groups` should normalize to path-like values or preserve author input exactly. The guide proposes preserving author input.

### What should be done in the future

- Upload the bundle to reMarkable.
- Implement expansion helper tests before changing seeded-user conversion.

### Code review instructions

- Start with `design-doc/01-keycloak-compatible-claim-presets-guide.md`.
- Focus on the proposed `keycloak:` schema, merge rules, and decision records.

### Technical details

Ticket path:

```text
/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/05/TINYIDP-CLAIMS-001--keycloak-compatible-claim-presets-for-tinyidp
```

## Step 2: Correct claim presets away from Keycloak-specific realm semantics

This step corrected the scope of the claim-preset ticket. The user explicitly clarified that Keycloak realm-specific claim structures are not needed and that tinyidp should stay generic. The design now focuses on provider-neutral top-level authorization claims such as `groups`, `roles`, `tenant`, `preferred_username`, and `locale`.

The result is a more precise implementation plan: convenience fields expand into the existing `Scenario.ExtraClaims` map, explicit `claims` still acts as the escape hatch, and no nested realm/client role structures are added.

### Prompt Context

**User prompt (verbatim):** "No need for the keycloak realm stuff, we want to keep this generic. 

Add detailed list of tasks so we can precisely keep track of where we are in the docmgr ticket, and then implement step by step, committing at appropriate intervals, and keeping a detailed diary as you work."

**Assistant interpretation:** Remove the Keycloak-specific realm/client-role design from the claim-preset ticket, replace it with a generic claim-preset design, add precise task tracking, then proceed incrementally with commits and diary updates.

**Inferred user intent:** The user wants tinyidp to remain a generic OIDC/local-test IdP rather than accumulating provider-specific compatibility concepts.

**Commit (code):** pending — documentation scope correction before implementation.

### What I did

- Rewrote the claim preset design doc to remove Keycloak-specific semantics.
- Replaced the task list with detailed phases for generic claim presets.
- Defined generic seeded-user fields: `groups`, `roles`, `tenant`, `preferred_username`, and `locale`.
- Preserved the existing raw `claims` map as the escape hatch for provider-specific or unusual shapes.

### Why

- The prior design overfit the ticket to Keycloak-compatible nested claims.
- Generic top-level fields are enough for common authorization tests and keep tinyidp provider-neutral.

### What worked

- The existing `Scenario.ExtraClaims` model fits generic presets naturally.
- No token/userinfo special-case should be necessary.

### What didn't work

- The previous design contained Keycloak realm/client terminology that the user rejected for this scope.

### What I learned

- The important product boundary is generic OIDC test ergonomics, not provider emulation.
- Provider-specific shapes should remain possible through raw `claims`, but not promoted as first-class fields.

### What was tricky to build

- The tricky part is preserving power without teaching provider-specific defaults. The solution is to add only generic convenience fields and keep `claims` as the explicit escape hatch.

### What warrants a second pair of eyes

- Review the merge rule where explicit `claims` override convenience fields.
- Review whether `preferred_username` should default from login when omitted. The current corrected design only emits it when explicitly configured.

### What should be done in the future

- Commit this scope correction.
- Implement Phase 1 and Phase 2 in `internal/scenario/seeded_users.go`.
- Add unit tests before changing docs/examples.

### Code review instructions

- Start with `design-doc/01-keycloak-compatible-claim-presets-guide.md`; despite the historic filename, the content is now generic.
- Then inspect `tasks.md` for the precise execution checklist.

### Technical details

The corrected implementation should not add `realm_access`, `resource_access`, or any Keycloak-named schema fields.
