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

## Detailed evidence for Step 3 — Replace the placeholder page with the Local Loop React application

**Date:** 2026-07-13

**Goal:** Build the real browser client, connect it to the xgoja route surface,
and prove the design against the initialized TLS product before treating the
frontend as complete.

**Inferred user intent:** The demonstration must be a usable application, not a
static mockup. Identity, CSRF, shared storage, ownership, restart persistence,
and visual presentation must all meet at the same browser boundary.

### What I did

- Replaced the three-file placeholder frontend with a pnpm-managed Vite,
  React, TypeScript, Redux Toolkit, RTK Query, and Bootstrap application.
- Added typed session, current-user, board, post, and reply contracts.
- Added RTK Query operations for session bootstrap, current-user projection,
  board reads, post creation, replies, owner deletion, and logout.
- Kept the CSRF token only in Redux memory and installed it as
  `X-CSRF-Token` only on unsafe methods.
- Implemented loading, unauthenticated, logged-out, error, empty-board,
  composition, thread, reply, statistics, and ownership states.
- Implemented the requested early-Mac monochrome visual language with ordinary
  system sans and monospace fonts, strict line work, no menu bar, no title bar,
  no window chrome, no gradient, and no desktop simulation.
- Used blue, teal, and coral only as foreground accents. Structural surfaces
  remain black, white, and warm gray.
- Updated xgoja's asset source from `public` to Vite `dist`.
- Made `go generate` build the frontend and invoke xgoja generation with
  `--clean`. This makes the embedded asset tree a mirror rather than an
  append-only collection of old content hashes.
- Updated generated-asset tests to discover the hashed JS and CSS references
  in `index.html` and request both through the embedded application.
- Added an executable Playwright harness under the ticket's `scripts/`
  directory. It drives the system Chrome binary through the real TLS,
  password, OIDC authorization-code, PKCE, application-session, BBS, and
  logout paths.
- Ran the harness with separate Alice and Bob browser contexts. It created a
  hostile-markup checkpoint post, replied as Bob, denied Bob's deletion,
  restarted the process with the same state root, verified the post and reply,
  and deleted them as Alice.
- Inspected the full-page screenshot rather than relying only on DOM tests.

### Exact verification commands

```bash
pnpm --dir cmd/tinyidp-xapp/app/frontend run typecheck
pnpm --dir cmd/tinyidp-xapp/app/frontend run build
go generate ./cmd/tinyidp-xapp
go test ./cmd/tinyidp-xapp -count=1

PYENV_VERSION=3.11.4 python \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/scripts/01_real_browser_bbs.py \
  --base-url https://127.0.0.1:19443 \
  --alice-password-file /tmp/tinyidp-xapp-real-browser/operator/alice-password \
  --bob-password-file /tmp/tinyidp-xapp-real-browser/operator/bob-password \
  --mode create \
  --marker TINYIDP-BBS-001 \
  --screenshot /tmp/tinyidp-bbs-alice.png

lsof-who -p 19443 -k
# Restart serve-initialized in tmux with the same --state-root, then:
PYENV_VERSION=3.11.4 python \
  ttmp/2026/07/13/TINYIDP-BBS-001--shared-durable-object-bulletin-board/scripts/01_real_browser_bbs.py \
  --base-url https://127.0.0.1:19443 \
  --alice-password-file /tmp/tinyidp-xapp-real-browser/operator/alice-password \
  --bob-password-file /tmp/tinyidp-xapp-real-browser/operator/bob-password \
  --mode verify-restart \
  --marker TINYIDP-BBS-001
```

### What worked

- TypeScript typechecking, Vite production build, generation, and focused Go
  tests passed.
- Repeated generation produced the same three-file embedded asset set: one
  index, one hashed stylesheet, and one hashed JavaScript bundle.
- The unauthenticated page rendered a sign-in state and did not attempt board
  mutations.
- Both Alice and Bob completed the actual password and OIDC redirect flow.
- All three cookies were Secure, HttpOnly, SameSite=Lax, and had the intended
  `/` or `/idp` path boundary.
- A post without CSRF returned 403. The valid RTK Query mutation returned 201.
- The stored author matched the display name derived from the authenticated
  `/api/me` projection: `Alice Operator`.
- Alice and Bob had distinct application-user IDs; Bob saw the shared post and
  added a reply.
- Bob had no delete control, and a handcrafted authenticated DELETE returned
  403 at the object ownership boundary.
- `<img src=x onerror=alert(1)>` remained text. No image element was created.
- Logout without CSRF returned 403. UI logout returned 204, removed the
  application session, and rendered the explicit ended-session state.
- A full process stop and reopen preserved the post and reply. Alice's final
  deletion removed both.

### What did not work, in chronological order

1. Running the script with the shell's default Python failed because that
   interpreter did not contain Playwright. The installed and previously
   verified harness environment is selected explicitly with
   `PYENV_VERSION=3.11.4`.
2. The initial unauthenticated view displayed
   `Request failed (PARSING_ERROR)`. `fetchBaseQuery` attempted to parse the
   host's plain-text 401 response, so RTK Query represented it as
   `PARSING_ERROR` with `originalStatus: 401`. `isUnauthorized` now accepts
   both the direct 401 and this documented parse-error shape.
3. The first successful create pass compared the trusted author to the
   hard-coded string `Alice`. The persistent fixture was initialized with the
   name `Alice Operator`. The harness now computes the expected display label
   from the separately authenticated `/api/me` response and compares it to the
   BBS projection. This is a stronger provenance invariant than a fixture
   literal.
4. Regeneration initially left an older hashed JavaScript file beside the new
   one. xgoja already provides `generate --clean`; adding that flag to the
   generation directive removed stale executable assets without changing the
   generator.
5. The first audit aggregation query used the guessed column name
   `event_type`; the real schema calls it `event`. Inspecting `.schema
   auth_audit_records` supplied the correct query. A storage-size query
   similarly guessed `value`; the Durable Object table uses `value_json`.
   These were read-only inspection mistakes and did not affect application
   state.

### Audit and persistence evidence

The application audit database contained paired policy and completion records
for `bbs.read`, `bbs.post.created`, `bbs.reply.created`, and
`bbs.post.deleted`. It also contained 403 completion evidence for Bob's denied
delete and denied records for missing-CSRF creates. The inspection grouped by
event, outcome, and status and intentionally omitted actor and resource IDs.

Exactly one physical Durable Object database contained the logical `board`
key. The inspection reported only `BBS board key present`; it did not copy the
hashed physical filename into the application, documentation API, or UI.

### What I learned

- An HTTP status and a client-library error discriminant are not always the
  same value. Authentication bootstraps should normalize transport/parser
  representations before choosing the unauthenticated UI.
- Identity assertions in integration tests should compare two trusted server
  projections or compare a trusted projection with a user-visible result. A
  hard-coded display string tests fixture spelling rather than provenance.
- Content-hashed build output requires replacement semantics at the embed
  boundary. Copying a new tree over an old tree is not deterministic cleanup.
- Browser evidence is necessary here because it joins cookie flags, redirects,
  JavaScript escaping, CSRF headers, conditional controls, and state refreshes
  that direct object tests cannot observe.

### What warrants a second pair of eyes

- Review whether Bootstrap's complete CSS payload is acceptable for this small
  application or whether a later production optimization should compile only
  the used modules.
- Review the logged-out wording: the application session is definitely gone,
  while the IdP session may remain active by design.
- Review the display-name fallback (`preferredUsername`, then `Member`) and
  decide whether a production community should expose a configured profile
  name instead.
- Inspect the narrow-screen presentation with additional real devices even
  though the layout is responsive and uses ordinary document flow.

### Code review instructions

1. Start at `src/api.ts` and verify which requests receive CSRF.
2. Read `src/App.tsx` as a finite set of session and board states.
3. Inspect `src/styles.css` for the absence of decorative desktop chrome and
   for visible `:focus-visible` treatment.
4. Trace `dist/index.html` through `xgoja.yaml` into generated embedded assets.
5. Run both browser modes around a real tmux restart.
6. Query audit records by aggregate only; do not paste subject IDs into review
   notes.

## Chronological diary

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

## Step 3: Build and verify the Local Loop browser application

The full commands, browser evidence, failures, and review instructions for this
step are preserved in the detailed Step 3 evidence section at the beginning of
this document. Commit `c3f7b37` is the frontend checkpoint.

## Step 4: Harden validation, distinguish logout scopes, and run release gates

### Prompt context

The user observed that clicking “Sign in again” after logout did not request a
password. I explained that `/auth/logout` revokes only the xapp session while
the tiny-idp browser SSO session remains active. The user explicitly asked to
implement the two distinct operations.

### What I changed

- Added a table-driven object validation matrix covering type, empty-value,
  length, category, missing-post, malformed-route, and no-mutation invariants.
- Added an authenticated HTTP invalid-category case and proved rejected input
  does not consume the first post ID.
- Implemented strict `GET /end-session` in `internal/fositeadapter`.
- Added `end_session_endpoint` to strict discovery in the same change as the
  mounted endpoint.
- Required `client_id` when a post-logout redirect is requested and compared
  the URI exactly against that client's durable allowlist.
- Validated the redirect before revoking any session or emitting clearing
  cookies. This avoids turning error handling into either an open redirect or
  a surprising partial logout.
- Revoked the server-side IdP session represented by the opaque configured
  cookie handle, treating an already-absent session as idempotent success.
- Cleared both configured IdP session and CSRF cookies with the issuer path,
  HttpOnly, Secure, SameSite, expiration, and negative Max-Age attributes.
- Added stable `logout.success` and rejection audit events and `no-store` on
  the endpoint response.
- Added provider tests for valid redirect/revocation/cookie/audit behavior and
  invalid-redirect non-mutation behavior.
- Replaced one ambiguous “Log out” control with two precise actions:
  “Log out of Local Loop” and “Log out of Local Loop + tiny-idp.”
- Kept the local ended-session page explicit that SSO remains active and added
  a link to end the IdP session there too.
- Added an IdP-ended page that states the scope accurately: this browser has
  lost the Local Loop and tiny-idp sessions, but other applications and devices
  own independent sessions.
- Extended the browser harness to prove local SSO reuse followed by IdP logout
  and mandatory password entry on the next authorization.
- Added a 390-pixel layout assertion and keyboard-visible-focus assertion.
- Updated README and strict security-profile claims so discovery, routes, and
  documentation agree.
- Saved the official OpenID Connect RP-Initiated Logout 1.0 text with Defuddle
  under this ticket's `sources/` directory.

### Security model

The two operations are intentionally sequential for the larger scope:

```text
Local only:
  CSRF-protected POST /auth/logout
  -> revoke xapp session
  -> preserve tiny-idp SSO

Local + tiny-idp in this browser:
  CSRF-protected POST /auth/logout
  -> GET /idp/end-session with registered RP redirect
  -> validate client and exact redirect
  -> revoke durable IdP browser session
  -> clear both IdP cookies
  -> redirect to explicit ended state
```

This is not front-channel logout, back-channel logout, device-wide logout, or
revocation of other applications' already-established sessions. The UI and
guide now state that limit.

### Browser evidence

The final create-mode result proved all of the following in one initialized TLS
process:

- secure cookies at their intended paths;
- distinct Alice and Bob application identities;
- missing-CSRF create status 403;
- trusted author label `Alice Operator`;
- Bob delete status 403;
- literal hostile-markup rendering;
- local logout and SSO reuse;
- Local Loop plus tiny-idp logout and a password form on the next sign-in;
- zero horizontal overflow at 390 pixels;
- an interactive keyboard focus target with a visible outline.

The process was then killed with `lsof-who`, relaunched in tmux against the
same state root, and verify-restart mode proved that both checkpoint records
survived before owner deletion. The preexisting user post titled `Hello` was
preserved; aggregate storage inspection therefore correctly reported one
remaining post after checkpoint cleanup.

### Audit and storage evidence

Aggregate application audit queries showed allowed/completed BBS reads,
creates, replies, and deletes, missing-CSRF denials, and object-level 403 delete
completions. The identity audit JSONL contained accepted `logout.success`
records. Queries deliberately omitted actor IDs, resource IDs, cookie values,
and physical object hashes.

Exactly one Durable Object database contains the logical `board` key. The
final aggregate query reported the baseline post count without copying the
hashed physical database name into documentation or UI.

### Failures and corrections

1. Playwright's non-exact button lookup matched both logout labels. Adding
   `exact=True` to the local-only selector fixed the harness without changing
   UI semantics.
2. The initial narrow-layout check found a two-pixel overflow. Live DOM bounds
   identified Bootstrap row negative gutters as the cause. `mx-0` keeps the
   row inside the page shell; long thread headings also receive
   `overflow-wrap:anywhere`. The unchanged strict assertion then passed.
3. `make lint` failed before analysis because the wrapper forces `GOWORK=off`
   even though this checkout intentionally uses only the top-level workspace
   module. The pinned linter was run directly with `go.work` enabled.
4. The full lint run reported two preexisting `nonamedreturns` findings in
   `development_app.go` and `production_app.go`. Neither file was changed in
   this checkpoint. Running the same pinned linter with
   `--new-from-rev c3f7b37` from the tiny-idp Git root reported `0 issues`.

### Verification commands and results

```bash
pnpm --dir cmd/tinyidp-xapp/app/frontend run typecheck
go generate ./cmd/tinyidp-xapp
go test ./internal/fositeadapter ./internal/oidcmeta ./cmd/tinyidp-xapp -count=1
go test -race ./internal/fositeadapter -run '^TestEndSession' -count=1
go test -race ./cmd/tinyidp-xapp -run 'TestBBS|TestDevelopmentApplication' -count=1
go test ./... -count=1
go build ./...
/tmp/golangci-lint-v2.12.2-go1.26.5 run -c .golangci.yml --new-from-rev c3f7b37 ./...
go vet -vettool=/tmp/glazed-lint-v1.3.6-go1.26.5 \
  -glazedclilint.allow-paths=cmd/tinyidp/main.go,internal/cmds/admin.go,internal/cmds/admin_backup.go,internal/cmds/admin_client.go,internal/cmds/admin_export.go,internal/cmds/admin_keys.go,internal/cmds/admin_ops.go,internal/cmds/config.go,internal/cmds/profiles.go \
  ./cmd/... ./internal/... ./pkg/...
```

All commands above passed. The full unfiltered linter has only the two noted
preexisting findings.

### Review instructions

1. Review `end_session.go` before the frontend. Redirect validation must remain
   before session mutation.
2. Confirm discovery and route registration change together.
3. Confirm the larger logout always performs application CSRF logout first.
4. Run invalid-redirect tests and verify the session remains active.
5. Run both browser modes around a real process kill/restart.
6. Read the saved RP-Initiated Logout specification and compare the current
   current-browser subset with future `id_token_hint` and coordinated logout
   work.

### Residual work

- The strict endpoint does not yet process a verified `id_token_hint`.
- There is no front-channel or back-channel logout coordination.
- Other devices and other applications' already-established sessions are not
  revoked by current-browser logout.
- The Makefile's forced `GOWORK=off` targets should be repaired in a separate
  workspace-tooling change.
- BBS capacity branches are bounded and implemented but are not filled to 200
  posts/100 replies in this release test; configurable small limits would make
  those tests inexpensive.

## Step 5: Prepare the delivery bundle and final running handoff

### What I did

- Committed the validation, strict logout, browser hardening, official source,
  and verification documentation as `110b008` (`Security: add scoped browser
  logout`).
- Related the nine primary route, object, frontend, provider, test, and harness
  files to the ticket index with specific notes.
- Marked Phase 4 complete after the checkpoint commit and recorded exact
  reproduction commands above.
- Terminated the old listener with `lsof-who -p 19443 -k`, recreated tmux
  session `tinyidp-xapp-e2e`, and started the initialized TLS product from the
  committed tree with the preserved state root.
- Captured the startup line and queried readiness. Evidence was:

  ```text
  tinyidp-xapp initialized TLS server started listen=127.0.0.1:19443
  {"status":"ready"}
  ```

- Ran `docmgr doctor --ticket TINYIDP-BBS-001 --stale-after 30`; all checks
  passed before rendering the reMarkable bundle.

### Current running process

```text
tmux session: tinyidp-xapp-e2e
URL:          https://127.0.0.1:19443/
state root:   /tmp/tinyidp-xapp-real-browser/state
readiness:    https://127.0.0.1:19443/readyz
```

The local certificate is the preserved test fixture and is self-signed. The
process is suitable for handoff and demonstration, not a claim that the `/tmp`
fixture is production deployment state.

### Delivery target

The complete Markdown set, including the design guide, diary, playbook, ticket
metadata, task ledger, changelog, and saved OpenID standard, is bundled as one
PDF named `TINYIDP BBS 001 Shared Durable Object Bulletin Board` at:

```text
/ai/2026/07/13/TINYIDP-BBS-001
```

The dry run enumerated all eight Markdown documents and the subsequent upload
completed with:

```text
OK: uploaded TINYIDP BBS 001 Shared Durable Object Bulletin Board.pdf -> /ai/2026/07/13/TINYIDP-BBS-001
```
