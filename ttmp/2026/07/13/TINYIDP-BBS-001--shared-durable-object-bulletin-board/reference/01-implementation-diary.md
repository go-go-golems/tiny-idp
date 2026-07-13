---
Title: Implementation Diary
Ticket: TINYIDP-BBS-001
Status: active
Topics:
    - architecture
    - xgoja
    - identity
    - security
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/bbs.go
      Note: Uncommitted native Go facade that the accepted design removes
    - Path: repo://ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/reference/01-investigation-diary.md
      Note: Earlier BBS false start and original visual requirements
ExternalSources: []
Summary: Chronological implementation record for the shared Durable Object BBS, including the corrected Go-versus-xgoja boundary.
LastUpdated: 2026-07-13T16:27:18.315440162-04:00
WhatFor: Preserve decisions, commands, failures, commits, review risks, and continuation state for the BBS feature.
WhenToUse: Read before resuming or reviewing TINYIDP-BBS-001 and update after every meaningful implementation or verification interval.
---


# Implementation Diary

## Goal

This diary records the design, implementation, testing, delivery, and lessons
for the shared `BBS/community` Durable Object application. It is written so a
new contributor can resume from any completed step without reconstructing the
security boundary from conversation history.

## Step 1: Create the feature ticket and correct the application boundary

The work began with an uncommitted BBS prototype in the broader XAPP ticket.
That prototype correctly recognized that a shared board could not use the
actor-private dispatcher, but it incorrectly concluded that the BBS needed
native Go HTTP handlers. Source inspection showed that the xgoja Durable
Objects module already exposes trusted in-process named dispatch even while the
raw external gateway remains disabled.

This step created a dedicated feature ticket, converted the corrected boundary
into accepted design decisions, and divided the work into stable phases. The
old false start remains part of the evidence because the difference between a
disabled external gateway and an available trusted module call is easy to miss.

### Prompt Context

**User prompt (verbatim):** "ok, go ahead. Create a new docmgr ticket for the feature, and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Then iplement task by task, keep a detailed diary, commit at appropriate itnervals."

**Assistant interpretation:** Create a standalone intern-ready feature ticket,
document the corrected route and object architecture, implement each tracked
task, preserve a strict diary, commit coherent checkpoints, and deliver the
ticket bundle to reMarkable.

**Inferred user intent:** Make the BBS a maintainable product feature whose
code, security reasoning, test evidence, and history can be handed to a new
engineer without relying on prior chat context.

**Commit (code):** pending at the time this entry was created.

### What I did

- Read the ticket research, docmgr, diary, embedded frontend, textbook
  authoring, and reMarkable workflows and their required references.
- Created `TINYIDP-BBS-001` with a design guide, diary, verification playbook,
  task ledger, changelog, and standard directories.
- Added 25 tasks across contract, backend, frontend, verification, and delivery.
- Inspected the xgoja route, host composition, Durable Objects provider,
  manager, actor execution, SQLite storage, frontend source, and generator.
- Recorded file- and line-backed evidence in the design document.
- Accepted trusted xgoja routes plus literal `BBS/community` dispatch and
  superseded the native Go BBS facade.

### Why

- The feature needs an independent task and evidence trail rather than more
  unrelated work in the broad composition ticket.
- The architecture must be stable before generated code and frontend work make
  the change harder to review.
- Preserving the false start explains why the final design does not equate
  `enableRawGateway: false` with disabling `objects.fetch()`.

### What worked

- `docmgr ticket create-ticket` created the complete ticket skeleton.
- Provider source clearly separates named dispatch at lines 329-344 from raw
  gateway mounting at lines 381-400.
- Planned routes already provide authentication, CSRF, authorization, audit,
  actor context, parsed bodies, and path parameters.
- Exporting class `BBS` yields the required namespace without a manifest.

### What didn't work

- An earlier combined `apply_patch` intended to delete the Go facade and move
  routes to JavaScript failed before applying because its expected context did
  not match `development_app.go`:

  ```text
  apply_patch verification failed: Failed to find expected lines in
  /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/development_app.go
  ```

- A repository inspection was first run from `tiny-idp` while still prefixing
  paths with `tiny-idp/`, producing:

  ```text
  find: ‘tiny-idp/ttmp’: No such file or directory
  fatal: cannot change to 'tiny-idp': No such file or directory
  ```

- The first large documentation patch contained one unprefixed added line and
  was rejected atomically:

  ```text
  apply_patch verification failed: invalid hunk at line 1387,
  Expected update hunk to start with a @@ context marker,
  got: 'go vet ./cmd/tinyidp-xapp/...'
  ```

  It was split by document and applied successfully.

### What I learned

- `EnableRawGateway` controls whether `gateway()` is exposed as an HTTP
  handler. It does not remove trusted `rpc()` or `fetch()` module exports.
- The BBS API can remain in the xgoja route and Durable Object layers. Go only
  composes the existing services.
- A fixed shared object and an actor-bound private object represent different
  resource-sharing policies and should not share one dispatch helper.

### What was tricky to build

- The uncommitted object and old-ticket diary already encoded the obsolete Go
  facade decision. The correction must preserve history while making the new
  accepted architecture unambiguous.
- Trusted script can call arbitrary named objects, so safety depends on literal
  route coordinates and the absence of untrusted route plugins. This is a
  precise trust assumption, not a claim that named fetch is safe for a browser.
- The object must return 4xx responses for bad public input rather than throw,
  because an exception crosses the provider as an execution error.

### What warrants a second pair of eyes

- Verify every BBS route uses literal `"BBS"` and `"community"` coordinates.
- Verify fresh bodies select public fields before adding trusted actor fields.
- Review whether route bundles remain trusted when the proposed scripting layer
  is implemented.
- Review the single-document bounds against expected local usage.

### What should be done in the future

- If untrusted or tenant-authored route plugins are introduced, add a
  configured named shared-object capability for those plugins.
- Preserve the raw-gateway distinction in the go-go-objects API documentation.

### Code review instructions

- Begin with the accepted decisions in the main guide.
- Compare `durableobjects.go:329-344` with lines 381-400.
- Review `site.js` as trusted code and identify every untrusted field.
- Run `docmgr task list --ticket TINYIDP-BBS-001` for continuation status.

### Technical details

Ticket creation commands:

```bash
docmgr ticket create-ticket --ticket TINYIDP-BBS-001 \
  --title "Shared Durable Object Bulletin Board" \
  --topics architecture,xgoja,identity,security,testing
docmgr doc add --ticket TINYIDP-BBS-001 --doc-type design-doc \
  --title "Shared Durable Object BBS Analysis Design and Implementation Guide"
docmgr doc add --ticket TINYIDP-BBS-001 --doc-type reference \
  --title "Implementation Diary"
docmgr doc add --ticket TINYIDP-BBS-001 --doc-type playbook \
  --title "BBS Verification and Operations Playbook"
```

## Goal

<!-- What is the purpose of this reference document? -->

## Context

<!-- Provide background context needed to use this reference -->

## Quick Reference

<!-- Provide copy/paste-ready content, API contracts, or quick-look tables -->

## Usage Examples

<!-- Show how to use this reference in practice -->

## Related

<!-- Link to related documents or resources -->
