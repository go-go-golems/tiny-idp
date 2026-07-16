---
Title: Implementation diary
Ticket: TINYIDP-XAPP-DEVICE-001
Status: active
Topics:
    - auth
    - oidc
    - oauth2
    - xgoja
    - durable-objects
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/objects/objects.js
      Note: Durable object input and ownership invariants recorded in Step 1
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Observed composition seam recorded in Step 1
ExternalSources: []
Summary: Chronological implementation and validation diary for the xgoja device authorization API example.
LastUpdated: 2026-07-16T15:20:00-04:00
WhatFor: Preserve the reasoning, commands, outcomes, and review guidance needed to continue this security-sensitive implementation.
WhenToUse: Before changing the resource server, device client, xapp state, CLI, or end-to-end harness.
---


# Diary

## Goal

This diary records the construction of a complete tiny-idp device authorization
and xgoja durable-object API example. It is intentionally chronological: a new
engineer should be able to identify what was known at each decision point,
which files carried the relevant contract, and which tests demonstrate the
claimed behavior.

## Step 1: Establish the device-to-durable-object security contract

The ticket was created and the existing xapp composition was inspected before
any runtime behavior was changed. The result is a concrete vertical-slice
design rather than a vague request to “add device auth”: a public device client
obtains an audience-bound opaque token, the Go host introspects it with a
separate confidential resource client, and only a verified subject becomes the
durable-object actor.

This step also deliberately preserves the current browser model. Browser
cookie routes retain hostauth and CSRF protection. The new bearer API has its
own narrow route family, so a later endpoint cannot accidentally become a
cookie-or-bearer ambiguity merely because both paths need the BBS object.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new docmgr ticket to do exactly that. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Then add detailed phases and for each phase, detailed tasks, so that we can track our progress precisely, and then implement it, committing at appropriate intervals, keeping a detailed diary"

**Assistant interpretation:** Create an evidence-based ticket and detailed design for an end-to-end xgoja durable-object API accessed through tiny-idp device authorization, then carry out the implementation in reviewed phases.

**Inferred user intent:** Obtain a production-shaped reference application that proves the exact boundary between tiny-idp OAuth, a Go resource server, xgoja routes, and durable objects, while leaving a durable handoff trail.

**Commit (code):** N/A — design and baseline investigation only.

### What I did

- Created ticket `TINYIDP-XAPP-DEVICE-001` and its design, task ledger, and diary documents.
- Inspected `cmd/tinyidp-xapp/development_app.go`, `production_app.go`, `state.go`, `app/routes/site.js`, and `app/objects/objects.js`.
- Inspected the prior tiny-idp introspection/xgoja consumer contract.
- Ran focused repository discovery commands:
  - `find cmd/tinyidp-xapp -maxdepth 3 -type f -printf '%p\\n' | sort`
  - `sed -n '1,330p' cmd/tinyidp-xapp/development_app.go`
  - `sed -n '1,360p' cmd/tinyidp-xapp/app/routes/site.js`

### Why

- The desired feature crosses four authority boundaries. Writing a CLI first
  would risk validating tokens in the wrong layer or leaking a provider secret
  into JavaScript.
- Existing `BoundDispatcherService` already contains the desired ownership
  pattern, so the design should preserve it rather than invent a parallel
  object authorization mechanism.

### What worked

- `composeApplication` proved that the xapp has a clear Go-owned composition
  point for provider, native hostauth, and trusted xgoja routes.
- The previous introspection work already provides device discovery, a device
  grant, opaque bearer issuance, exact resource audiences, and constrained
  RFC 7662 responses. No protocol fork is needed.
- The new task ledger separates the security primitives, state bootstrap,
  routes, CLI, and end-to-end work in dependency order.

### What didn't work

- A broad `rg` search over frontend build output produced a truncated result:
  `Warning: truncated output (original token count: 30028)`. It was not useful
  for analysis because generated frontend output obscured the handwritten
  route files. Subsequent inspection used exact source files and bounded
  `sed` ranges instead.

### What I learned

- The current browser BBS path obtains identity from `gojahttp.ActorFromContext`
  and passes only host-derived actor data into the durable object. The bearer
  path must normalize to this same invariant.
- Initialized xapp state currently stores only the browser client identity;
  adding a device/client/resource topology requires an explicit state version
  and secret lifecycle decision.

### What was tricky to build

- The principal risk is confusing three OAuth registrations: the browser
  client, the public device client, and the confidential resource client. The
  symptom of that confusion would be a CLI secret or an API introspection
  credential entering the wrong process. The proposed registration table and
  state model make each role distinct before code is written.
- Browser and bearer requests both need BBS access, but only browser cookies
  are ambient credentials requiring CSRF. The solution is separate route
  families with a shared durable-object operation contract, not a dual-mode
  middleware shortcut.

### What warrants a second pair of eyes

- The exact `gojahttp` public API for attaching a native bearer principal must
  be checked before deciding whether bearer routes can be generated xgoja routes
  or should remain host-native for this ticket.
- State version migration/reinitialization needs operator review because an old
  secret/client topology must not be silently inferred.

### What should be done in the future

- Run and record the focused current test baseline, then begin the resource
  authentication core.

### Code review instructions

- Start at `cmd/tinyidp-xapp/development_app.go:composeApplication` and trace
  its `BoundDispatcherService.ActorID` closure into `app/routes/site.js`.
- Read the prior contract in `TINYIDP-INTROSPECTION-001/reference/03-xgoja-oidcresource-consumer-contract.md` before reviewing any bearer middleware.
- Validate this documentation step with `docmgr doctor --ticket TINYIDP-XAPP-DEVICE-001 --stale-after 30` after relations are added.

### Technical details

```text
device client -- opaque bearer --> Go resource middleware -- verified sub --> BBS
                                \-- Basic /introspect --> tiny-idp
```

The durable object continues to validate all public message fields. It treats
`actorId` and `actorName` as trusted only because the host, not the CLI body,
constructs them after authentication.
