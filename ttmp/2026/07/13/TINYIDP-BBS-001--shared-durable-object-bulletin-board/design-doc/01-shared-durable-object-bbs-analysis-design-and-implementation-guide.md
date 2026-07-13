---
Title: Shared Durable Object BBS Analysis Design and Implementation Guide
Ticket: TINYIDP-BBS-001
Status: active
Topics:
    - architecture
    - xgoja
    - identity
    - security
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/objects/objects.js
      Note: Shared BBS state machine and public projection
    - Path: repo://cmd/tinyidp-xapp/app/routes/site.js
      Note: Trusted planned BBS HTTP routes and actor boundary
    - Path: repo://cmd/tinyidp-xapp/bbs_test.go
      Note: Two-actor object ownership and restart verification
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: Go composition and host-service boundary
    - Path: repo://cmd/tinyidp-xapp/xgoja.yaml
      Note: Embedded route object and frontend source contract
    - Path: ws://go-go-goja/pkg/gojahttp/auth/appauth/appauth.go
      Note: Deny-by-default BBS action policy
    - Path: ws://go-go-objects/pkg/durableobjects/storage_sqlite.go
      Note: Durable object persistence implementation
    - Path: ws://go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go
      Note: Evidence separating in-process fetch from raw gateway mounting
ExternalSources: []
Summary: Intern-ready architecture, API, security, state, frontend, implementation, and verification guide for the shared Durable Object bulletin board.
LastUpdated: 2026-07-13T16:27:18.167445462-04:00
WhatFor: Explain why the BBS belongs in trusted xgoja routes and a shared Durable Object, then define the contracts required to implement and review it.
WhenToUse: Read before implementing, reviewing, extending, debugging, or operating the BBS feature.
---



# Shared Durable Object BBS Analysis Design and Implementation Guide

## Executive Summary

The bulletin board is the first shared-state application built into
`tinyidp-xapp`. It proves that the current identity provider, xgoja HTTP host,
and go-go-objects runtime can form one self-contained application. An
authenticated user can create a categorized post, another user can read and
reply to it, and only the original author can delete it. State survives process
restart because the board is stored by the Durable Objects SQLite backend.

The implementation has four responsibility boundaries:

- tiny-idp authenticates credentials and produces the OIDC identity consumed
  by the application session.
- Host authentication validates the application session and supplies a trusted
  `ctx.actor` to planned xgoja routes.
- Trusted xgoja routes implement `/api/bbs`, enforce authentication, CSRF,
  authorization policy, and audit declarations, then dispatch to the fixed
  `BBS/community` object.
- The `BBS` Durable Object owns the persisted schema, input validation,
  identifiers, timestamps, capacity limits, ownership checks, mutation order,
  and public response projection.

The public API is JavaScript, not a native Go facade. The existing
`require("durableobjects").fetch(namespace, name, request)` function permits
trusted in-process code to call a named object. `enableRawGateway: false`
prevents mounting the caller-selected raw HTTP gateway; it does not disable the
in-process module call. Route code uses literal object coordinates and never
copies object identity from a browser request.

The frontend is a pnpm/Vite React application written in TypeScript. Redux
Toolkit stores authenticated session state, RTK Query implements the API client
and cache invalidation, and Bootstrap supplies layout and accessible form
primitives. Custom CSS implements the requested early-Mac monochrome
presentation without Chicago, menu bars, window title bars, fake chrome,
gradients, or desktop simulation. Pastel 1950s colors appear only as readable
foreground accents.

## Problem Statement

The current application demonstrates two independent properties: a browser can
authenticate through the embedded tiny-idp, and each authenticated actor can
read and write a private Durable Object. The frontend exposes that private
object as a JSON scratchpad. This is useful as a composition test, but it does
not demonstrate shared application state, multi-user authorization, or a user
interface that supports an actual workflow.

A bulletin board changes the object topology. Private state uses one physical
object per authenticated actor:

```text
Alice ─────► USER_STATE/<opaque Alice binding>
Bob   ─────► USER_STATE/<opaque Bob binding>
```

A shared board requires several authenticated actors to reach the same object:

```text
Alice ──┐
Bob   ──┼────► BBS/community
Carol ──┘
```

This topology creates new security questions. The browser must not choose an
arbitrary namespace or object name. The system must not accept author identity
from JSON. Every cookie-authenticated mutation must verify CSRF. Authorization
to use the board must remain distinct from authorization to delete a particular
post. Public JSON must omit OIDC subjects and physical storage identifiers.

### Goals

- Provide one shared local bulletin board named `Local Loop`.
- Let authenticated users list posts, create posts, reply, and delete their own
  posts.
- Derive authorship only from the authenticated application session.
- Keep the raw Durable Objects HTTP gateway disabled.
- Persist board data across actor eviction and process restart.
- Return a public projection that omits stored actor identifiers.
- Ship the frontend and API in the existing single application binary.
- Produce executable evidence for CSRF, validation, sharing, ownership, and
  restart behavior.

### Non-goals

- Multiple boards or browser-selected object names.
- Anonymous reading or posting.
- Editing posts or replies.
- Moderator or administrator deletion.
- Attachments, Markdown rendering, HTML input, or rich text.
- Search, pagination, reactions, messaging, or notification delivery.
- A backwards-compatibility UI for the private JSON scratchpad.

## Current-State Analysis

### Application composition

`cmd/tinyidp-xapp/development_app.go:173-219` constructs the goja HTTP host and
supplies the generated bundle with its HTTP, host-auth, Durable Object manager,
and actor-bound dispatcher services. Lines 231-243 load trusted routes and
mount tiny-idp, native authentication handlers, and the xgoja host on one
`http.ServeMux`.

The Go host owns process configuration and lifecycle. It does not need to own
BBS business behavior. A native Go BBS handler would duplicate route planning,
audit, CSRF, and JavaScript dispatch facilities already present in the runtime.

### Planned routes

`cmd/tinyidp-xapp/app/routes/site.js` is trusted application code embedded in
the generated binary. Its current private-object mutation demonstrates the
security builder:

```javascript
app.post("/api/object")
  .auth(express.user().required())
  .csrf()
  .allow("user.self.update")
  .audit("user.object.updated")
  .handle((ctx, res) => { /* trusted handler */ });
```

Authentication establishes `ctx.actor`; CSRF rejects cross-site mutations;
authorization evaluates the declared action; audit records the planned
operation; and the handler runs after these gates. The body and path parameters
remain untrusted even though the handler itself is trusted.

### Private and shared object calls

The current route calls `objects.fetchForActor("USER_STATE", request)`. The
host-created `BoundDispatcher` derives an opaque physical name from the actor
and an allowlisted namespace. That is correct for private state and cannot
produce one board shared across users.

The same module exports `objects.fetch(namespace, name, input)`.
`go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go:329-344`
constructs an `ObjectID` and calls `Manager.Dispatch`. The gateway flag is
checked separately at lines 381-400 when code asks for an externally mountable
handler. Therefore:

```text
enableRawGateway = false

disables: browser/caller-selected Durable Object HTTP gateway
does not disable: trusted in-process objects.fetch(...) calls
```

### Execution and persistence

`go-go-objects/pkg/durableobjects/manager.go:70-107` validates an object ID,
locates or starts the matching actor, and dispatches the operation.
`actor.go:142-167` evaluates the object bundle and constructs the exported
class. Operations for one object run through that actor, allowing a synchronous
read-modify-write operation without an additional application mutex.

`storage_sqlite.go:31-73` maps an object hash to a database below the configured
storage root. Lines 119-149 enable WAL and initialize the key/value schema. The
BBS stores one versioned document under key `board`. Actor eviction discards
the runtime but not the SQLite state. A process restart reads the same document
when the application keeps the same state root.

### Generated assets

`cmd/tinyidp-xapp/xgoja.yaml` embeds trusted routes, frontend assets, and the
object bundle. `cmd/tinyidp-xapp/generate.go` produces TypeScript declarations
and the checked-in runtime package. The Vite build must precede xgoja generation
so generated Go assets contain `dist/index.html` and its hashed static files.

## Proposed Solution

### Component architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│ Browser: React + Redux Toolkit + RTK Query                       │
└──────────────┬──────────────────────────────────────────────────┘
               │ HTTPS, app-session cookie, X-CSRF-Token
               ▼
┌─────────────────────────────────────────────────────────────────┐
│ One Go HTTP server                                              │
│ /idp/* tiny-idp       /auth/* host-auth                         │
│ /api/bbs* xgoja       /static/* embedded Vite assets            │
└──────────────┬──────────────────────────────────────────────────┘
               │ objects.fetch("BBS", "community", request)
               ▼
┌─────────────────────────────────────────────────────────────────┐
│ BBS/community Durable Object                                    │
│ schema, validation, ordering, ownership, public projection      │
└──────────────┬──────────────────────────────────────────────────┘
               │ state.storage.get/put("board")
               ▼
┌─────────────────────────────────────────────────────────────────┐
│ SQLite Durable Object storage                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Trust boundary

```text
UNTRUSTED                     TRUSTED APPLICATION CODE

request.body.title ───────┐
request.body.body  ───────┼──► selected fields ───────► validation
request.body.category ────┘

request.body.actorId ─────X    never copied
request.body.object  ─────X    never copied
request.query.namespace ──X    never copied

validated session ───────────► ctx.actor.id ──────────► actorId
OIDC claims ─────────────────► display-name snapshot ► actorName
route constants ─────────────► BBS / community ──────► ObjectID
```

The handler constructs a new request body. It never merges arbitrary browser
JSON after security-sensitive values:

```javascript
// Wrong: the last spread can replace actorId.
body: { actorId: ctx.actor.id, ...ctx.body }

// Correct: public fields are selected, identity is session-derived.
body: {
  title: ctx.body && ctx.body.title,
  body: ctx.body && ctx.body.body,
  category: ctx.body && ctx.body.category,
  actorId: ctx.actor.id,
  actorName: actorDisplayName(ctx.actor)
}
```

### Why `/api/bbs` belongs in xgoja

The URL is a public application contract. Its implementation belongs in the
trusted xgoja route layer because that layer already owns application-route
security. Go provides the route runtime and host services. The Durable Object
owns domain behavior.

A Go handler would repeat actor extraction, CSRF verification, audit creation,
and fetch-envelope construction. Enabling the raw HTTP gateway would instead
grant the browser excessive object-selection authority. Trusted named dispatch
with fixed literals avoids both problems.

## HTTP API Reference

Every endpoint requires an authenticated application session. Mutations require
the `X-CSRF-Token` returned by `/auth/session`.

### Public types

```typescript
type BBSCategory = "general" | "projects" | "questions" | "notes";

interface BBSReply {
  id: string;
  body: string;
  author: string;
  createdAt: string;
}

interface BBSPost {
  id: string;
  title: string;
  body: string;
  category: BBSCategory;
  author: string;
  createdAt: string;
  canDelete: boolean;
  replies: BBSReply[];
}

interface BBSBoard {
  name: "Local Loop";
  description: string;
  posts: BBSPost[];
  stats: { posts: number; replies: number };
}
```

No public type contains an `authorId`, OIDC subject, object name, object hash,
SQLite path, or storage key.

### `GET /api/bbs`

Reads the board and calculates viewer-relative deletion permission. Success is
`200` with `BBSBoard`.

### `POST /api/bbs/posts`

```json
{
  "title": "Release checklist",
  "body": "Please review the browser restart test.",
  "category": "projects"
}
```

Success is `201` with the updated board. The route ignores additional
security-sensitive fields and injects trusted actor data.

### `POST /api/bbs/posts/:postId/replies`

```json
{ "body": "I ran the restart case." }
```

Success is `201` with the updated board. A well-formed missing post returns
`404`.

### `DELETE /api/bbs/posts/:postId`

The object compares the post's stored `authorId` with the trusted current
`actorId`. Success is `200` with the updated board. Another authenticated user
receives `403 {"error":"not_post_author"}`.

### Expected errors

Expected client and domain failures return a stable JSON code:

| Status | Code | Meaning |
|---:|---|---|
| 400 | `title_must_be_text` | Title is not a string. |
| 400 | `title_required` | Normalized title is empty. |
| 400 | `title_too_long` | Title exceeds 100 JS characters. |
| 400 | `body_required` | Post or reply body is empty. |
| 400 | `body_too_long` | Body exceeds its limit. |
| 400 | `invalid_category` | Category is outside the fixed set. |
| 401 | host-auth response | Session is absent or invalid. |
| 403 | host CSRF response | CSRF token is missing or invalid. |
| 403 | `not_post_author` | Actor does not own the post. |
| 404 | `post_not_found` | Post does not exist. |
| 409 | `board_capacity_reached` | Board already has 200 posts. |
| 409 | `reply_capacity_reached` | Post already has 100 replies. |

Untrusted validation returns 4xx from the object instead of throwing. Throws
are reserved for corrupted stored state or missing trusted fields because a
throw crosses the provider as an execution failure.

## Durable Object Design

### Namespace and stored schema

The bundle exports class `BBS`, which derives namespace `BBS` through
go-go-objects namespace conversion:

```javascript
exports.objects = { UserState, BBS };
```

The persisted document is internal and contains identity fields used for
authorization:

```typescript
interface StoredBoardV1 {
  version: 1;
  nextPostId: number;
  nextReplyId: number;
  posts: StoredPostV1[];
}

interface StoredPostV1 {
  id: string;
  title: string;
  body: string;
  category: BBSCategory;
  authorId: string;
  authorName: string;
  createdAt: string;
  replies: StoredReplyV1[];
}

interface StoredReplyV1 {
  id: string;
  body: string;
  authorId: string;
  authorName: string;
  createdAt: string;
}
```

The object generates monotonically increasing identifiers:

```text
post_000000000001
reply_000000000001
```

They must be unique within one board, not unpredictable. Generating them inside
the serialized object eliminates collisions and removes identifier authority
from the browser.

### Mutation pseudocode

```text
function createPost(request):
    board = loadAndValidateStoredBoard()
    if length(board.posts) >= 200:
        return conflict("board_capacity_reached")

    title = validatePublicText(request.body.title, 100)
    content = validatePublicText(request.body.body, 4000)
    category = validateCategory(request.body.category)
    actorID = requireTrustedText(request.body.actorId, 256)
    actorName = requireTrustedText(request.body.actorName, 80)

    post = {
        id: nextSequenceID("post", board.nextPostId),
        title, body: content, category,
        authorId: actorID, authorName,
        createdAt: currentUTCISOString(), replies: []
    }

    board.nextPostId += 1
    board.posts.append(post)
    storage.put("board", board)
    return created(publicBoard(board, actorID))
```

Deletion performs authorization and mutation in the same dispatch:

```text
function deletePost(postID, actorID):
    board = loadAndValidateStoredBoard()
    post = findPost(board, postID)
    if post is absent: return notFound("post_not_found")
    if post.authorId != actorID: return forbidden("not_post_author")
    remove post and its replies
    storage.put("board", board)
    return ok(publicBoard(board, actorID))
```

### Projection and limits

Stored posts are oldest first; the public projection returns newest first.
Replies remain chronological. Projection constructs fresh objects and omits
internal actor IDs.

The V1 limits are:

- Title: 100 characters.
- Post body: 4,000 characters.
- Reply body: 2,000 characters.
- Board: 200 posts.
- Replies per post: 100.

These limits bound the single board document. They do not replace production
rate limiting, moderation, monitoring, or retention policy.

## Identity and Security

### Authentication sequence

```text
User       Browser      tiny-idp      host-auth      xgoja route      BBS
 │            │             │              │              │            │
 ├───────────►│ login       │              │              │            │
 │            ├────────────►│ credentials  │              │            │
 │            │◄────────────┤ OIDC code    │              │            │
 │            ├───────────────────────────►│ callback     │            │
 │            │◄───────────────────────────┤ app session  │            │
 │            ├────────────────────────────► validate     │            │
 │            │             │              ├─────────────►│ ctx.actor  │
 │            │             │              │              ├───────────►│
 │            │◄───────────────────────────────────────────────────────┤
```

Display names never authorize an operation. The object stores the stable actor
ID for authorization and a name snapshot for presentation.

### CSRF and authorization

Every POST and DELETE route declares `.csrf()`. RTK Query adds the session CSRF
token only to unsafe methods. A missing token must fail before object dispatch.

Route-level `.allow("bbs.post.delete")` answers whether a principal may invoke
that operation class. Object-level comparison answers whether the principal
owns this specific post. Both checks are necessary.

### Stored XSS

Content is plain text. React renders strings as escaped text nodes. The BBS UI
must not use `dangerouslySetInnerHTML`, a Markdown renderer, or direct DOM HTML
insertion. Browser coverage submits hostile markup and verifies that no element
is created.

### Audit events

| Route | Event |
|---|---|
| `GET /api/bbs` | `bbs.read` |
| `POST /api/bbs/posts` | `bbs.post.created` |
| `POST /api/bbs/posts/:postId/replies` | `bbs.reply.created` |
| `DELETE /api/bbs/posts/:postId` | `bbs.post.deleted` |

Audit output must not include post bodies, reply bodies, cookies, CSRF tokens,
or physical object hashes.

## Frontend Design

Redux contains a small auth slice and the RTK Query API slice. Form drafts stay
in component-local state.

```text
Redux store
├── auth: status, csrfToken, user display fields
└── bbsApi
    ├── getBoard
    ├── createPost
    ├── createReply
    └── deletePost
```

The UI hierarchy is:

```text
App
├── Masthead: wordmark, signed-in user, logout
├── BoardIntroduction: description and counters
├── PostComposer: title, category, body, status
└── ThreadList
    └── Thread: metadata, content, delete, replies, reply form
```

### Visual system

The visual design references early monochrome Macintosh documents through
typography, rules, spacing, compact controls, and black-on-white contrast. It
does not reproduce an operating-system screen.

- Use `-apple-system`, `BlinkMacSystemFont`, `Helvetica Neue`, Arial, and
  sans-serif. Do not load or imitate Chicago.
- Do not render a menu bar, title bar, close box, fake window frame, desktop,
  gradient, bevel, or ornamental shadow.
- Use black rules, white surfaces, clear focus outlines, and compact metadata.
- Use monospaced system fonts only for timestamps, counters, and small labels.
- Use Bootstrap for layout and accessible form semantics, with custom CSS for
  the monochrome geometry.

Pastel colors are foreground accents on white:

| Token | Use | Value |
|---|---|---|
| `--accent-coral` | destructive action, questions | `#9d3f52` |
| `--accent-teal` | projects, success | `#236e69` |
| `--accent-blue` | notes, information | `#49658f` |
| `--accent-gold` | general, warnings | `#795f18` |

Desktop uses two columns, collapsing to one below the Bootstrap large
breakpoint. Every control has a label and visible focus; no behavior depends on
hover.

## Design Decisions

### Decision: Implement the API in trusted xgoja routes

- **Context:** The shared board cannot use actor-private dispatch. An initial
  implementation added Go handlers after misreading the disabled raw gateway.
- **Options considered:** Native Go handlers; raw HTTP gateway; trusted xgoja
  named dispatch; a new configured named capability.
- **Decision:** Use planned xgoja routes and
  `objects.fetch("BBS", "community", request)`.
- **Rationale:** Named module dispatch already exists, route security is
  declarative, and literal coordinates are not influenced by browser input.
- **Consequences:** No BBS-specific Go handler is needed. If untrusted route
  plugins are introduced, they require a narrower capability.
- **Status:** accepted.

### Decision: Use one fixed shared object

- **Context:** Sharing is required, but direct browser object selection is not.
- **Options considered:** One global object; one object per actor; path-selected
  boards; a host registry.
- **Decision:** Use literal namespace `BBS` and name `community`.
- **Rationale:** This gives one intentional sharing and serialization domain.
- **Consequences:** Multiple boards require a registry or configured binding.
- **Status:** accepted.

### Decision: Generate IDs and timestamps inside the object

- **Context:** IDs and times must not be forgeable browser fields.
- **Options considered:** Browser UUIDs; route randomness; object sequences;
  host-provided services.
- **Decision:** Use object-local sequence IDs and UTC object time.
- **Rationale:** Serialized dispatch is collision-free for one board and keeps
  domain authority inside the object.
- **Consequences:** IDs expose relative order and are not secrets.
- **Status:** accepted.

### Decision: Project permissions, not actor IDs

- **Context:** Deletion needs a stable ID while the UI does not.
- **Options considered:** Authorize by display name; expose IDs; store IDs and
  return `canDelete`.
- **Decision:** Store actor IDs internally and project display names plus
  viewer-relative `canDelete`.
- **Rationale:** Names may collide or change and cannot authorize deletion.
- **Consequences:** Old content retains its name snapshot.
- **Status:** accepted.

### Decision: Plain text only

- **Context:** HTML and Markdown add a stored-rendering security boundary.
- **Options considered:** Raw HTML; sanitized Markdown; React-rendered text.
- **Decision:** Accept and render plain text.
- **Rationale:** React escapes strings and no parser or sanitizer is needed.
- **Consequences:** Formatting is limited to preserved line breaks.
- **Status:** accepted.

## Alternatives Considered

### Native Go facade

The uncommitted prototype added `cmd/tinyidp-xapp/bbs.go` and registered it in
`development_app.go`. It securely authenticated, checked CSRF, generated IDs,
and dispatched a fixed object. It was rejected because it duplicates xgoja
route facilities and splits BBS knowledge between Go and JavaScript.

### Raw Durable Objects gateway

The gateway accepts caller-controlled object coordinates. It is appropriate for
development tooling but grants too much authority to a product browser and
remains disabled.

### Actor-bound board

`fetchForActor("BBS", ...)` produces one private board per actor and fails the
defining cross-user requirement.

### New named-capability provider first

A binding such as `community-board -> BBS/community` would further reduce the
trusted script capability. It is worthwhile if plugins become less trusted,
but it would expand this feature into a provider redesign without changing the
current browser boundary.

## Implementation Plan

### Phase 1: Contract and documentation

- Inspect all boundaries with file-backed evidence.
- Record the corrected raw-gateway interpretation.
- Specify APIs, schema, limits, errors, audits, and visual constraints.
- Create the guide, task ledger, diary, and playbook.

### Phase 2: Shared object and routes

- Delete the Go facade and mux registration.
- Add four planned routes to `app/routes/site.js`.
- Build fresh request bodies from selected public fields and `ctx.actor`.
- Implement the BBS schema and operations in `app/objects/objects.js`.
- Add direct object and route-level security tests.
- Commit the backend checkpoint.

### Phase 3: React application

- Replace the placeholder with Vite, React, TypeScript, Redux Toolkit, RTK
  Query, Bootstrap, and pnpm.
- Implement session bootstrap, CSRF-aware mutations, cache invalidation,
  logout, responsive UI states, and the visual system.
- Point xgoja assets at `dist`, build first, regenerate, and commit.

### Phase 4: Verification

- Add HTTP and two-user browser harnesses.
- Verify hostile markup, audit behavior, and restart persistence.
- Run focused, frontend, generation, and repository-wide checks.
- Commit verification evidence.

### Phase 5: Delivery

- Start the final TLS application in tmux.
- Complete tasks, relationships, changelog, diary, and doctor checks.
- Upload one ticket bundle to `/ai/2026/07/13/TINYIDP-BBS-001`.
- Commit final documentation and hand off residual risks.

## Testing Strategy

Direct object tests cover empty projection, validation, sequence IDs, reply
ordering, capacity, author deletion, public omission of IDs, and SQLite reload.

Route integration tests prove unauthenticated rejection, CSRF rejection, fixed
object selection, identity-spoofing resistance, status preservation, and raw
gateway confinement.

The browser scenario uses separate Alice and Bob contexts:

```text
Alice creates post
Bob sees Alice's post and replies
Bob cannot delete Alice's post
server restarts with same state root
Alice sees post and Bob's reply
Alice deletes thread
both contexts observe its absence
```

Build and static checks include:

```bash
pnpm --dir cmd/tinyidp-xapp/app/frontend run typecheck
pnpm --dir cmd/tinyidp-xapp/app/frontend run build
go generate ./cmd/tinyidp-xapp
go test ./cmd/tinyidp-xapp -count=1
go test ./... -count=1
go build ./...
```

## Open Questions

- General named fetch remains available to every trusted script in the bundle.
  A future untrusted plugin model requires a named capability boundary.
- One bounded document is suitable for V1, but pagination or per-thread objects
  require a new consistency and schema design.
- JavaScript length counts UTF-16 code units rather than grapheme clusters.
- Deleting a post deletes other users' replies; the UI must state this.
- Rate limiting, moderation, backup/restore drills, and production retention
  remain follow-ups in the broader production work.

## Intern Review Order

1. Read `app/routes/site.js` and separate public body fields from trusted actor
   fields and literal object coordinates.
2. Read `app/objects/objects.js` and trace one mutation through validation,
   storage, ownership, and projection.
3. Read `development_app.go` and confirm it supplies services without BBS
   handlers.
4. Read `xgoja.yaml` and `generate.go` to understand embedding.
5. Read the RTK Query slice before UI components to locate CSRF and caching.
6. Run direct, route, browser, and restart checks from the playbook.

Review questions:

- Can browser input select an object or actor?
- Can a mutation dispatch without CSRF verification?
- Can public JSON contain an actor ID or physical object identifier?
- Can concurrent dispatches allocate the same ID?
- Can a non-author delete by changing UI state?
- Does hostile markup remain text?
- Does state survive process restart?

## References

### tiny-idp

- `cmd/tinyidp-xapp/development_app.go` — application composition.
- `cmd/tinyidp-xapp/app/routes/site.js` — trusted HTTP routes.
- `cmd/tinyidp-xapp/app/objects/objects.js` — object classes and state.
- `cmd/tinyidp-xapp/xgoja.yaml` — source and artifact specification.
- `cmd/tinyidp-xapp/generate.go` — generated runtime entrypoint.
- `cmd/tinyidp-xapp/development_app_test.go` — login and route integration.
- `cmd/tinyidp-xapp/app/frontend/` — frontend source and build.

### go-go-goja

- `pkg/gojahttp/host.go` — route matching and request construction.
- `pkg/gojahttp/planned_dispatch.go` — actor-aware invocation.
- `pkg/gojahttp/auth_plan.go` — planned security contract.
- `modules/express/typescript.go` — route API declarations.
- `pkg/xgoja/hostauth/` — session, CSRF, and native auth services.

### go-go-objects

- `pkg/xgoja/providers/durableobjects/durableobjects.go` — module exports and
  gateway configuration.
- `pkg/durableobjects/bound_dispatcher.go` — actor-private dispatch.
- `pkg/durableobjects/manager.go` — actor startup and dispatch.
- `pkg/durableobjects/actor.go` — JS object execution.
- `pkg/durableobjects/manifest.go` — namespace derivation.
- `pkg/durableobjects/storage_sqlite.go` — persistence.

### Ticket

- `reference/01-implementation-diary.md` — chronological evidence and commits.
- `playbook/01-bbs-verification-and-operations-playbook.md` — executable checks.
