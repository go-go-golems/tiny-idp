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

## Step 2: Implement trusted routes and the shared object state machine

This step replaced the uncommitted native Go facade with four planned xgoja
routes and completed the `BBS/community` Durable Object. The routes establish
the coarse security boundary; the object establishes domain validation,
mutation order, persistence, ownership, and public projection.

The implementation tests the boundary twice. Direct object tests exercise two
actors and a real SQLite restart. Full application tests perform the embedded
OIDC login, prove CSRF rejection, send spoofed actor and object fields, and
confirm the route still attributes content to the application session.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Implement Phase 2 exactly as specified, retain
the fixed shared-object boundary, and commit the reusable policy change and BBS
vertical slice separately.

**Inferred user intent:** Obtain a secure and reviewable shared-state backend
before frontend presentation work begins.

**Commit (go-go-goja):** `f9dbf36` — "HTTP: authorize shared BBS route actions"

**Commit (tiny-idp):** `0f5b907` — "Xapp: add shared durable object BBS"

### What I did

- Deleted the uncommitted native `bbs.go` facade and removed its mux
  registration, returning `development_app.go` to composition-only behavior.
- Added `GET /api/bbs`, post creation, reply creation, and deletion routes to
  `app/routes/site.js`.
- Declared authentication and audit on every route and CSRF on every mutation.
- Selected `BBS/community` with literals and derived `actorId` and `actorName`
  from `ctx.actor`.
- Implemented schema V1, sequence IDs, UTC timestamps, validation, category and
  capacity limits, immutable persistence, public projection, and author-only
  deletion in `app/objects/objects.js`.
- Added BBS actions to the deny-by-default appauth authorizer. These actions
  allow an authenticated route only when it has no caller-selected resource;
  post-level ownership remains in the object.
- Added a direct two-actor object test with close/reopen persistence.
- Extended the embedded application vertical slice with unauthenticated, raw
  gateway, CSRF, spoofing, create, reply, and delete assertions.
- Regenerated the embedded route and object assets.

### Why

- Trusted planned routes remove duplicated Go security and dispatch code.
- Literal object coordinates preserve confinement without enabling the raw
  gateway.
- Object-generated IDs and timestamps prevent browser authority over domain
  metadata.
- Immutable value replacement works consistently for fresh JavaScript objects
  and values decoded from SQLite through Go.
- Explicit appauth actions preserve deny-by-default policy rather than reusing
  semantically incorrect self-update actions.

### What worked

- Direct dispatch derived namespace `BBS` from the exported class and persisted
  one shared board in SQLite.
- Alice and Bob received different `canDelete` projections for the same post.
- Bob's object-level deletion returned `403 not_post_author`.
- Closing and reconstructing the object server with the same root preserved the
  post and reply.
- The application route ignored attacker-supplied actor, namespace, and object
  fields and returned author `Alice` from the session.
- Missing CSRF failed with 403 before object mutation.
- The focused appauth and tiny-idp test suites passed.

### What didn't work

- The first direct reply test returned `201` but the projected reply list was
  still empty:

  ```text
  development_app_test equivalent: replied board ... Replies:[] ...
  --- FAIL: TestBBSSharedStateOwnershipAndRestart
  ```

  The stored board had been decoded through Go-backed values. Mutating the
  nested `post.replies` array in place did not update the document subsequently
  written by the test runtime. The implementation now builds a new post array
  and a new board value for every mutation.

- The first valid route create returned 403 after CSRF had succeeded:

  ```text
  development_app_test.go:246: create post status=403 body=Forbidden
  ```

  The host authorizer denies unknown action strings. Adding the four explicit
  BBS action cases, plus allow and denial tests, resolved the failure.

### What I learned

- Durable Object JavaScript should treat values returned from storage as values
  to replace, not mutable object graphs whose nested mutations are guaranteed
  to propagate through Goja wrappers.
- Planned `.allow()` names are executable policy inputs, not audit-only labels.
  Adding a route action requires adding the corresponding deny-by-default host
  policy decision.
- Resource ownership can remain inside the object while the outer authorizer
  controls the operation class. The absence of a caller-selected resource is a
  useful policy invariant for this fixed board.

### What was tricky to build

- The route must accept title, body, and category while discarding forged
  actor and object fields. The route selects only those public fields and adds
  trusted identity last.
- Expected validation errors must remain object responses. Trusted field or
  stored schema failures throw and become execution failures.
- The public-projection leak test must inspect the raw response body. Marshaling
  a reduced Go test struct would hide leaked fields during unmarshal and create
  a false assurance result.

### What warrants a second pair of eyes

- Review `fetchBoard` and confirm its only callers construct selected request
  bodies; consider replacing the internal spread with explicit fields if this
  helper grows.
- Review the appauth action placement. The package already represents a small
  explicit monolith policy, but a future general extension mechanism may be
  preferable to adding application action constants.
- Review stored schema validation depth and corruption behavior.
- Review sequence exhaustion and document size assumptions even though V1
  capacity makes practical exhaustion unreachable.

### What should be done in the future

- Add a test that fills configured small capacity limits if those constants are
  made injectable for testing.
- Consider an application-owned authorizer extension point before adding many
  unrelated action families to the shared appauth package.

### Code review instructions

- Start with `app/routes/site.js`, then trace into class `BBS`.
- Read `bbs_test.go` before the implementation to see the invariants expressed
  as executable behavior.
- Review the appauth switch and its positive and negative tests in go-go-goja.
- Reproduce with:

  ```bash
  go test ../go-go-goja/pkg/gojahttp/auth/appauth -count=1
  go test ./cmd/tinyidp-xapp -run 'TestBBS|TestDevelopmentApplicationLoginToApplicationVerticalSlice' -count=1 -v
  ```

### Technical details

- Object identity: namespace `BBS`, logical name `community`.
- IDs: `post_` or `reply_` plus a 12-digit sequence.
- Storage key: `board`.
- Mutations write newly constructed board documents.
- Public projection omits `authorId` and returns `canDelete` for each post.

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
