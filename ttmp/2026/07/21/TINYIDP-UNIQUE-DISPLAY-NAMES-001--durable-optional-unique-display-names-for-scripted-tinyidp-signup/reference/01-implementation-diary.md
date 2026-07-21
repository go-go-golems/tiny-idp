---
Title: Implementation diary
Ticket: TINYIDP-UNIQUE-DISPLAY-NAMES-001
Status: active
Topics:
    - tiny-idp
    - oidc
    - goja
    - identity
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological implementation record for durable optional unique display names."
LastUpdated: 2026-07-21T19:39:02.485697806-04:00
WhatFor: "Review decisions, validation, and remaining operational work for the feature."
WhenToUse: "Read before modifying the signup policy, display-name claims, or related browser UX."
---

# Implementation diary

## Goal

Record each meaningful implementation step for the durable unique-display-name
feature, including evidence, failures, and review guidance.

## Context

The shared TinyIDP demo uses a Goja workflow to collect a display name, verify
email, select a password, and atomically create an identity. A display name is
currently not unique. This ticket adds an optional script-selected policy with
a native durable reservation as the final authority.

## Quick Reference

The target flow is: `lookup (advisory) -> present/challenge -> commit effect
(policy) -> atomic reservation (authoritative) -> themed recovery`.

## Usage Examples

Run the focused package tests while changing each layer, then the shared
browser journey before declaring the user experience complete.

## Related

See `../design-doc/01-durable-unique-display-names-analysis-design-and-implementation-guide.md`.

## Step 1 — map the existing boundary and create the implementation plan

### Prompt Context

> ok, Create a new ticket with docmgr and add a design/implementation guide and upload to remarkable, then work on those tasks, commit at appropriate interval, keep a detailed diary as you work.

### Assistant interpretation

The request authorizes a new focused ticket and implementation, not merely an
answer about feasibility. The intended product rule is an optional but durable
unique display-name policy for scripted signup. No commit exists yet for this
step.

### What I did

- Created ticket `TINYIDP-UNIQUE-DISPLAY-NAMES-001`.
- Added the analysis/design/implementation guide and a task list with seven
  traceable tasks.
- Mapped the current browser-to-script-to-native path:
  `open-signup.js` -> `ctx.commit.signup` -> `createLocalIdentity` effect ->
  `commitScriptedSignup` -> `idpaccounts` -> `idpstore` transaction.
- Confirmed that the current stores index login and subject only; `User.Name`
  is stored in JSON and has no normalized uniqueness invariant.

### Why

The map identifies the only trustworthy place to prevent a race: the existing
native store transaction. It also identifies why a simple JavaScript lookup
cannot be the solution by itself.

### What worked

- `docmgr ticket create-ticket`, document creation, and task creation
  completed successfully.
- Code inspection confirmed a narrow extension point in the closed signup
  effect API and an existing capability-binding pattern to follow.

### What didn't work

Nothing failed in this planning step. The prior worktree contains unrelated,
untracked Local Compose ticket material; it will be preserved and excluded
from every commit.

### What I learned

- `idpsignup.Executor.InvokeSubmission` currently supplies no runtime
  capabilities, so the identity lookup requires a deliberate injection seam.
- The shared program already uses a bounded durable invitation lookup; it is a
  suitable model for an identity lookup capability.
- The provider already has a themed global duplicate-identity error path;
  display-name collision needs its own truthful code and message.

### What was tricky

“Optional policy” conflicts with a global unique `users.name` index. A
separate claim/reservation model permits one script to require uniqueness
without changing applications that treat display names as cosmetic.

### What warrants a second pair of eyes

Review the proposed normalization and reconciliation semantics before enabling
the policy on a production database containing historical identities.

### What should be done in the future

Consider a separate immutable public handle only if product requirements
outgrow the current profile display-name concept.

### Code review instructions

Start with the design guide, then inspect the listed native effect and store
boundaries. Verify later commits never make Goja the final authority.

### Technical details

The planned capability is `identity.displayName.lookup@v1` with a coarse
boolean result. The planned effect flag is `uniqueDisplayName: true`; the
native transaction must reserve a canonical key before writing the user and
credential.
