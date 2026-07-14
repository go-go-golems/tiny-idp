---
Title: Investigation Diary
Ticket: TINYIDP-ACCOUNT-CHOOSER-001
Status: active
Topics:
    - identity
    - oidc
    - security
    - architecture
    - go
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-14T17:53:02.926941303-04:00
WhatFor: ""
WhenToUse: ""
---

# Investigation Diary

## Goal

Record the evidence and design decisions for adding safe, standard OIDC
multi-account selection to tiny-idp.

## Step 1: Map the single-session model and standards boundary

The investigation began from an observed Message Desk behavior: creating a
second account after local logout still authorized the original IdP account.
Repository inspection showed that this is not a random application bug.
Message Desk revoked only its RP session; tiny-idp correctly retained the
provider browser session and reused it on the next authorization request.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket for that, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


We want tiny-idp to be well designed and robust, abd provide a full toolbox of apis and tools and packages to build higih quality identity experiences"

**Assistant interpretation:** Create a dedicated research/design ticket for a reusable multi-account account-chooser capability, rather than implementing an ad hoc Message Desk switch.

**Inferred user intent:** tiny-idp should offer hosts standard, auditable, extensible identity UX primitives while retaining provider control of all authentication decisions.

### What I did

- Created `TINYIDP-ACCOUNT-CHOOSER-001` with an analysis document and diary.
- Inspected `session.go`, `provider.go`, `interaction.go`, `end_session.go`,
  `idpstore`, `idpui`, and SQLite migration boundaries.
- Retrieved and stored official OpenID Core, RP-Initiated Logout, Session
  Management, and Prompt Create specifications with Defuddle.
- Confirmed Core defines `prompt=select_account` and
  `account_selection_required`; confirmed Fosite already allows the prompt.

### What worked

```text
docmgr ticket create --ticket TINYIDP-ACCOUNT-CHOOSER-001 ...
defuddle parse https://openid.net/specs/openid-connect-core-1_0-18.html --md ...
```

The source bundle contains 320,655 bytes of retraceable specification text.

### What didn't work

The first Defuddle attempt failed in the restricted sandbox with `Error: fetch
failed`. A normal networked retry wrote all four official source files. An
initial documentation patch used an incorrect absolute workspace path and
removed the generated template before its add operation failed; the design
document was then recreated at the confirmed ticket path.

### What I learned

Account choice is already standardized. The engineering problem is not a URL
parameter; it is representing multiple valid sessions and activating a selected
one without reissuing an old raw cookie handle or trusting browser claims.

### What was tricky to build

The current store retains only hashes of session handles. That is correct for
security, but it means a chooser cannot simply retrieve and resend a selected
session cookie. The design therefore requires an atomic fresh-handle activation
operation that verifies server-side membership/session/user state.

### What warrants a second pair of eyes

- The privacy policy for remembered account labels needs product review.
- The semantics of per-entry removal versus global session revocation need an
  explicit decision before storage work begins.

### What should be done in the future

Implement the phased task plan after review of the proposed public store and
embedding APIs.

### Code review instructions

- Read the design’s current-state section against `session.go` and
  `provider.go`.
- Verify each protocol claim against the stored sources.

### Technical details

```text
single active IdP cookie -> existing correct SSO behavior
multiple remembered sessions -> provider-owned context/entry records
prompt=select_account -> context-bound chooser interaction -> fresh activation
```
