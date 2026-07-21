---
Title: Implementation diary
Ticket: TINYIDP-EMAIL-SIGNUP-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Chronological decisions and evidence for verified-email signup with a private first-deploy delivery outbox."
LastUpdated: 2026-07-21T16:09:43.745179461-04:00
WhatFor: "Record why fake delivery is allowed only as a private transport substitution and how the design reuses TinyIDP's durable native challenge workflow."
WhenToUse: "Read before implementing or resuming TINYIDP-EMAIL-SIGNUP-001."
---

# Implementation diary

## Goal

Activate durable, truthful email-verified signup for both applications while using a private operator outbox for the first deployment and deferring real SMTP credentials and cluster delivery work.

## Step 1: Separate fake delivery from fake verification

I created a separate ticket rather than reopening the completed invitation-core ticket. I inspected TinyIDP's challenge service, compiled email-verified signup program, SQLite persistence tests, browser continuation handler, account commit, and production validation.

### Prompt Context

**User prompt (verbatim):** "create a separate ticket + design doc for the email signup, for now, fake it for the first deploy."

**Assistant interpretation:** Design email-verified signup as an independent workstream, retain the real durable verification core, and use a temporary private delivery sink for the first deployment instead of integrating the personal SMTP server now.

**Inferred user intent:** Unblock the first small deployment without creating a dishonest `email_verified` claim or a disposable alternate signup implementation.

### What I did

- Created `TINYIDP-EMAIL-SIGNUP-001` with a design document, implementation diary, detailed tasks, and changelog.
- Located the existing typed `Mailer` interface and durable challenge service.
- Confirmed the SQLite store already owns challenge attempts, resends, expiry, evidence, and cleanup.
- Confirmed explicit continuations preserve pending and verified challenge references across browser waits and process restarts.
- Confirmed the native signup commit sets `EmailVerified` only when verified evidence is present and matches the account login.
- Confirmed `serve-production` intentionally rejects challenge outcomes because it constructs no mailer or challenge service.
- Selected a private operator SMTP catcher as the first-deploy transport substitution.
- Rejected displaying the code in the public browser or automatically marking first-deploy accounts verified.

### Why

- The invitation core and email-delivery activation have different scope, secrets, operational dependencies, and acceptance criteria.
- Reusing the real challenge path means later SMTP activation changes deployment binding rather than application semantics.
- A publicly visible code cannot prove email possession and must not produce a verified claim.

### What worked

- The relevant packages and restart tests make the boundary unusually clear: durable verification is complete, while production delivery construction is missing.
- The existing `Mailer` request is already narrow enough for a fixed-template SMTP adapter.

### What didn't work

- No code or experiment failed during this design step.

### What I learned

- The production blocker is smaller than a general email subsystem: one delivery adapter, one key/constructor path, one validator change, and one combined signup program.
- `CreateAndSend` persists before SMTP delivery, so delivery failure recovery must be tested through the existing bounded resend path.

### What was tricky to build

- The term “fake” could describe either transport or verification. Only fake transport is acceptable if TinyIDP continues to emit `email_verified=true`.
- The operator outbox contains live bearer codes and therefore needs private access and short retention even though it is temporary.

### What warrants a second pair of eyes

- Review whether operator-assisted code relay provides an acceptable first-deploy identity assurance statement for the actual invited user population.
- Review the proposed SMTP TLS modes and ensure plaintext is accepted only on an isolated private workload network.

### What should be done in the future

- Implement the phases in `tasks.md`.
- Replace the private mail catcher with the personal SMTP server through configuration after the separate deployment work is authorized.

### Code review instructions

- Begin with `pkg/idpemailchallenge/service.go` and `internal/fositeadapter/scripted_signup.go`.
- Compare `pkg/idpsignup/email_verified_signup.js` with the current two-application `open-signup.js`.
- Confirm `internal/cmds/serve_production.go` remains fail-closed until the native service is fully constructed.

### Technical details

```text
temporary: TinyIDP -> SMTP -> private operator outbox -> manual relay -> user
later:     TinyIDP -> SMTP -> personal mail server -> recipient mailbox

unchanged in both modes:
code generation -> keyed hash -> durable bindings -> attempts/resends ->
native verification -> evidence reference -> atomic verified account commit
```
