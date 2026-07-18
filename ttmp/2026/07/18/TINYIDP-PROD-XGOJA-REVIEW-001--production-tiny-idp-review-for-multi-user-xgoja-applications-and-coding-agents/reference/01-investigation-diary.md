---
Title: Investigation diary
Ticket: TINYIDP-PROD-XGOJA-REVIEW-001
Status: active
Topics:
    - architecture
    - auth
    - identity
    - oauth2
    - oidc
    - operations
    - research
    - security
    - testing
    - xgoja
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/production_app.go
      Note: Current production composition across IdP, hostauth, resource auth, and xgoja
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Existing controlled self-registration workflow
    - Path: repo://ttmp/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001--production-tiny-idp-review-for-multi-user-xgoja-applications-and-coding-agents/scripts/01-evidence-map.sh
      Note: Reproducible symbol and route evidence map
    - Path: ws://go-go-goja/modules/express/auth_builders.go
      Note: Express planned-auth syntax and route-plan compilation
    - Path: ws://go-go-goja/pkg/gojahttp/enforcer.go
      Note: Host-owned enforcement order before JavaScript dispatch
ExternalSources: []
Summary: Chronological evidence, commands, findings, decisions, validation results, and delivery record for the production IdP review.
LastUpdated: 2026-07-18T13:58:42.125849394-04:00
WhatFor: Preserve how the review was conducted and make its evidence and validation reproducible.
WhenToUse: When reviewing the final design, reproducing findings, or continuing the investigation.
---


# Diary

## Goal

This diary records the evidence-first review of `tiny-idp` as the production
identity plane for multi-user xgoja applications and coding agents. It preserves
the commands, conclusions, failures, review risks, and delivery steps needed for
another engineer to reproduce or continue the work.

## Step 1: Establish the ticket and research contract

The first step created a dedicated docmgr workspace and fixed the scope before
reading implementation details. The review must connect the identity provider,
browser-facing user lifecycle, xgoja Express authentication syntax, resource
server enforcement, and device-authorized coding-agent access without treating
them as unrelated features.

The deliverable is documentation and review guidance, not a compatibility layer
or an implementation patch. Existing focused tickets are evidence inputs; this
ticket synthesizes the end-to-end production model and names unresolved gaps.

### Prompt Context

**User prompt (verbatim):** "Creat ea new docmgr ticket to do a review of tiny-idp/ in the axis of using it as a production idp for allowing a set of multi-user xgoja apps using the go-go-goja express auth syntax to be deployed, allowing users to signup, use the site, but also use the device auth path to register coding agents to use the API.

Create  a detailed analysis / design / code review guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.

You can write and run experiments in the docmgr ticket's scripts/ directory. 

Keep a detailed diary as you work, commit at appropriat eintervals."

**Assistant interpretation:** Create a new ticket containing an evidence-backed,
intern-facing production-readiness review and target design for tiny-idp-backed
multi-user xgoja applications, including browser signup/login and coding-agent
OAuth device authorization, then validate, commit, and upload it to reMarkable.

**Inferred user intent:** Establish a technically precise shared model that a new
engineer can use to understand the system, review its security and production
gaps, and implement the remaining work without rediscovering the architecture.

### What I did

- Read the ticket research, docmgr, diary, textbook-authoring, and reMarkable upload instructions.
- Confirmed the `tiny-idp` worktree was clean on `task/prod-tiny-idp`.
- Inspected the existing ticket registry and vocabulary.
- Created `TINYIDP-PROD-XGOJA-REVIEW-001`, its design document, this diary, and eight research/delivery tasks.
- Defined the writing contract: direct technical prose, conceptual foundations first, no analogies, complete-sentence bullets, and explicit observed-versus-proposed labels.

### Why

- A separate synthesis ticket preserves the boundaries of existing focused tickets while providing one end-to-end production review.
- Recording the scope first prevents later code reading from narrowing the analysis to whichever package is easiest to inspect.

### What worked

- `docmgr status --summary-only` located the workspace at `tiny-idp/ttmp` and reported a healthy existing documentation corpus.
- `docmgr ticket create-ticket` and both `docmgr doc add` calls created the expected index, task list, changelog, design document, and diary.
- Existing vocabulary already covered every required topic.

### What didn't work

- The first `git status --short --branch` ran from the multi-repository workspace root and returned exactly: `fatal: not a git repository (or any of the parent directories): .git`. Re-running it from `tiny-idp/` succeeded.
- The first ticket-file search targeted `ttmp` at the workspace root, but `.ttmp.yaml` points docmgr at `tiny-idp/ttmp`; the corrected search used that repository root.

### What I learned

- The workspace contains neighboring checkouts for `tiny-idp`, `go-go-goja`, and `go-go-objects`; identity-provider documentation belongs in the `tiny-idp` worktree even when it cites integration code from a neighboring repository.
- Existing tickets cover device authorization, browser account selection, deployment validation, token introspection, and an xgoja device client. This ticket must synthesize them and verify current code rather than copy their conclusions.

### What was tricky to build

- The scope crosses repository boundaries but only one repository owns the ticket. The solution is to store the review in `tiny-idp/ttmp`, cite neighboring files by absolute path where they materially shape the design, and avoid editing those repositories during a documentation-only review.

### What warrants a second pair of eyes

- Review whether the final target architecture draws the issuer/resource-server boundary correctly when several xgoja applications share one IdP but retain distinct clients, audiences, scopes, and application data stores.
- Check every security conclusion against current code rather than assuming a focused ticket was fully implemented.

### What should be done in the future

- Keep focused implementation work in follow-up tickets after this review establishes priorities and ownership boundaries.

### Code review instructions

- Start with this diary, then the ticket index and primary design document.
- Reproduce ticket discovery with `docmgr ticket list --ticket TINYIDP-PROD-XGOJA-REVIEW-001`.
- Confirm no unrelated source changes with `git status --short` from `tiny-idp/`.

### Technical details

Initial commands:

```text
docmgr status --summary-only
docmgr ticket create-ticket --ticket TINYIDP-PROD-XGOJA-REVIEW-001 ...
docmgr doc add --ticket TINYIDP-PROD-XGOJA-REVIEW-001 --doc-type design-doc ...
docmgr doc add --ticket TINYIDP-PROD-XGOJA-REVIEW-001 --doc-type reference --title "Investigation diary"
```

## Step 2: Map the identity, browser-session, Express, and agent boundaries

This step traced the current product from durable account creation through OIDC
browser login and into xgoja planned-route enforcement. It then traced the
separate OAuth device grant and bearer-token introspection path used by the
coding-agent example. The result is a boundary map: tiny-idp authenticates and
issues credentials; go-go-goja creates application sessions and enforces
browser route plans; the current agent API uses a Go-owned resource-server
handler instead of the Express `user` security mode.

The review also located a reusable self-registration implementation in Message
Desk. That code demonstrates the controls required around the public
`idpaccounts.Service.Create` primitive, but the production `tinyidp-xapp` host
does not mount an equivalent signup route. Signup is therefore an observed gap,
not an undocumented feature.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Verify the current implementation boundaries and
collect line-addressable evidence for the intern-facing production design.

**Inferred user intent:** Make the final review concrete enough that a new
engineer can follow actual call paths and distinguish shipped behavior from
proposed work.

### What I did

- Read the current `tiny-idp` README, production host, embedding, storage, conformance, and security-profile documentation.
- Traced `embeddedidp.Options.Validate`, `Bootstrap`, `Provider`, Fosite route registration, device authorization, browser verification, and device token redemption.
- Traced `idpaccounts.Service.Create` and `AuthenticatePassword`, including durable lockout, dummy password work, bounded Argon2id concurrency, and post-commit audit semantics.
- Traced the production xapp state initializer, combined application composition, xgoja planned routes, Durable Object actor binding, resource-token introspection, and device API handler.
- Traced go-go-goja's Express builder into `RoutePlan`, `Enforcer`, hostauth OIDC callback/session creation, resource resolution, authorization, CSRF, and audit.
- Read the Message Desk public registration workflow as an existing security pattern.
- Checked the primary protocol references for RFC 8628, RFC 7662, RFC 8707, RFC 9700, RFC 7636, and OpenID Connect.
- Added and ran `scripts/01-evidence-map.sh` as a compact reproduction aid.

### Why

- A production review must model trust boundaries and state ownership, not just list endpoints.
- The browser and coding-agent paths use different credentials and authenticators; combining them conceptually without preserving that distinction would create an unsafe design.
- Public account creation needs controls that are not part of the low-level account service itself.

### What worked

- Current source provides a complete strict device grant with durable hashed codes, fresh password verification, transactional consumption/token persistence, audience propagation, and resource-server introspection.
- The xapp integration test `TestDeviceAPITwoUsersScopesAndMalformedCredentials` covers two subjects, scope denial, ambiguous Authorization headers, audience rejection, and password-triggered token invalidation.
- Express planned routes fail registration unless they declare `.public()` or `.auth(...)`; authenticated routes require `.allow(action)` before handler registration.
- `gojahttp.Enforcer` completes authentication, CSRF, resource resolution, and authorization before it creates the JavaScript secure context.
- The Message Desk signup pattern uses a one-use pre-session attempt, CSRF, same-origin validation, rate limiting by address and normalized-login hash, generic errors, and audit.

### What didn't work

- The first broad repository `rg` emitted 244,098 tokens of matches and was truncated. Subsequent searches were narrowed by package, symbol, and exact file.
- A shell glob intended to enumerate several prior ticket design documents failed with exactly: `zsh:1: no matches found: ttmp/2026/07/13/TINYIDP-ACCOUNT-CHOOSER-001*/design-doc/*.md`. Direct paths and `find` were used afterward.
- `nl -ba cmd/tinyidp-xapp/app/verbs/site.js` failed with exactly: `nl: cmd/tinyidp-xapp/app/verbs/site.js: No such file or directory`. The generated source is under `cmd/tinyidp-xapp/app/routes/site.js`.

### What I learned

- One tiny-idp issuer can serve multiple relying parties, but every browser app, device client, and introspecting resource server needs a distinct registered client role and narrowly declared grants/scopes/audiences.
- OIDC identity is keyed by the pair `(issuer, subject)`. go-go-goja correctly normalizes that pair into an application-owned user, then projects memberships into its own session.
- The planned Express DSL currently models browser `user` authentication. It does not model bearer tokens, device clients, route scopes, or OAuth audiences.
- The production xapp proves one safe agent path by keeping bearer tokens and introspection credentials in Go and dispatching a constrained actor into Durable Objects. It does not make those agent endpoints Express-auth planned routes.
- The xapp initializes one first user but no public signup HTTP surface. The library primitive exists, and Message Desk provides a pattern, but product integration remains.
- Several high-level docs contain historical wording that no longer matches current code, especially around device-flow availability and TLS/proxy topology. Final conclusions must cite source and tests first.

### What was tricky to build

- The same process holds three sessions with different meanings: the IdP browser session, the relying-party/xgoja application session, and the coding agent's bearer credential. The final guide must name each cookie/token and owner explicitly so an intern does not treat them as interchangeable.
- The xapp has two API surfaces over the same Durable Object: browser Express routes and host-owned agent routes. Both derive an actor, but through different authenticated paths and with different authorization vocabularies.

### What warrants a second pair of eyes

- Decide whether the desired product should extend go-go-goja's planned-auth model with a bearer security mode or keep agent APIs as separate Go-owned handlers. The current source supports only the latter.
- Review whether opaque-token positive caching for up to 30 seconds meets the desired password-change/disable revocation latency.
- Review whether a single issuer shared by many independently deployed apps should remain embedded per app or become one standalone identity service; SQLite permits only one active writer per database.

### What should be done in the future

- Add an explicit self-registration product contract and decide whether account verification/recovery is in the initial production scope.
- Add an explicit multi-application client/audience/scope provisioning model.
- Resolve documentation drift as part of any implementation ticket derived from this review.

### Code review instructions

- Run `ttmp/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001--production-tiny-idp-review-for-multi-user-xgoja-applications-and-coding-agents/scripts/01-evidence-map.sh` from `tiny-idp/`.
- Start the browser trace at `cmd/tinyidp-xapp/production_app.go:NewInitializedApplication`, then follow hostauth into `pkg/gojahttp/auth/oidcauth` and `pkg/gojahttp/enforcer.go` in the neighboring go-go-goja checkout.
- Start the agent trace at `cmd/tinyidp-xapp/device_cli.go`, then follow `/idp/device_authorization`, `/idp/device`, `/idp/token`, `resourceauth.Authenticator`, and `deviceAPIHandler`.
- Start the signup trace at `examples/tinyidp-message-app/app_http.go:handleCreateAccount` and end at `pkg/idpaccounts/accounts.go:Service.Create`.

### Technical details

Observed credential ownership:

~~~text
IdP browser cookie       -> tiny-idp Fosite adapter
xapp session cookie      -> go-go-goja hostauth/sessionauth
authorization code       -> browser RP callback, one-use exchange
device code              -> coding-agent client, one-use polling credential
access token             -> coding agent and Go resource server only
introspection secret     -> Go resource server only
ctx.actor                -> minimal post-enforcement projection exposed to JS
~~~

## Step 3: Verify protocol details and establish the test baseline

This step tested the current repository and checked the coding-agent protocol
against the primary standards. The complete Go test suite, build, and focused
race tests pass. The device implementation is therefore a well-tested current
mechanism, while signup, generic multi-app provisioning, and Express bearer
authentication remain absent product capabilities rather than unexplained test
failures.

The protocol comparison found one interoperability issue: the current device
authorization endpoint and CLI use the form parameter audience. RFC 8707
defines resource as the request parameter for targeting an access token. The
implementation propagates the requested value correctly into token audience
and introspection, but a generic standards-aware device client will not know
the private request vocabulary.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate current code in proportion to the
security risk and turn any product or standards gap into a concrete review item.

**Inferred user intent:** Give the next engineer trustworthy evidence and
acceptance criteria, not an architecture proposal detached from executable
behavior.

### What I did

- Read device authorization parsing, device token redemption, discovery
  metadata, introspection response construction, xapp state provisioning, and
  the device CLI token cache.
- Compared those paths with RFC 8628, RFC 7662, RFC 8707, RFC 7636, RFC 9700,
  OpenID Connect Core, and RFC 7009.
- Ran the complete repository test suite.
- Built every package.
- Ran the Fosite adapter and production xapp under the Go race detector without
  using cached results.
- Converted the findings into a readiness matrix, P0/P1/P2 gap list, source
  map, API reference, threat model, and phased implementation plan.

### Why

- Passing focused tests alone would miss repository-wide build or integration
  regressions.
- Standards comparison is necessary because an internally consistent client
  and server can share a private extension while remaining incompatible with
  other clients.
- Race validation is proportionate for one-use grant consumption, token
  persistence, session state, and the integrated multi-user xapp.

### What worked

- go test ./... completed successfully across every repository package and the
  ticket script packages.
- go build ./... completed successfully.
- go test -race ./internal/fositeadapter ./cmd/tinyidp-xapp -count=1 completed
  successfully. The Fosite adapter took 91.935 seconds and tinyidp-xapp took
  37.591 seconds.
- Device grant creation enforces method, media type, body size, client
  capability, client authentication, openid scope, exact allowed audiences,
  rate limits, expiry, and polling interval.
- Device redemption can issue a refresh token only when the client has the
  refresh grant and the approved scopes contain offline_access. The current
  xapp intentionally provisions neither.
- Introspection authenticates a confidential resource client before parsing the
  token, returns inactive without a token oracle, and requires a shared allowed
  audience.

### What didn't work

- The first combined symbol search was broader than needed and its 13,103-token
  output was truncated. Exact line reads were used for the device, discovery,
  token, and introspection handlers.

No build, test, or race command failed in this step.

### What I learned

- The current discovery document advertises the device grant and introspection
  endpoint, but it does not describe the private audience request convention.
- The current xapp CLI stores one opaque access token and expiry. It has no
  refresh-token cache or disconnect operation, so humans must authorize again
  after expiry.
- tiny-idp has no RFC 7009 endpoint. Password changes revoke server-side
  artifacts, while the resource server may retain a positive introspection
  result for up to its configured cache bound.
- A central issuer is a better match than one embedded issuer per application:
  it preserves shared identities and SSO while each app keeps its own local
  user, memberships, session store, resource, and confidential credential.
- go-go-goja's default OIDC transaction store remains in memory even when its
  other auth stores are durable. Application login callbacks therefore impose
  a one-replica or shared-transaction-state constraint.

### What was tricky to build

- “Device auth works” and “agent lifecycle is production complete” are
  different claims. The protocol exchange is implemented, but token refresh,
  user-visible disconnection, resource-parameter interoperability, and
  revocation latency are policy and product work.
- Extending planned auth must not make a bearer credential look like a browser
  session. The guide therefore proposes a new explicit OAuth mode and separate
  browser and agent namespaces instead of an adapter.

### What warrants a second pair of eyes

- Confirm whether standards migration must retain the existing audience form
  field. That would be backwards-compatibility work and needs an explicit user
  decision.
- Decide whether access-token-only agents meet the product's operating model or
  whether reviewed refresh rotation and disconnect are release blockers.
- Verify the proposed actor includes enough client and scope context for policy
  without leaking bearer material.
- Review whether the positive introspection cache bound is acceptable for
  account disable, password change, and future per-agent revoke.

### What should be done in the future

- Create separate implementation tickets for registration, multi-app client
  provisioning, RFC 8707 resource indicators, planned bearer auth, and agent
  credential lifecycle after product decisions are approved.
- Add an independent threat-model review before a production label.

### Code review instructions

- Re-run the three validation commands from the tiny-idp repository root.
- Review internal/fositeadapter/provider.go:438-556 beside RFC 8628 and RFC
  8707, paying special attention to form field names and duplicate handling.
- Review internal/fositeadapter/provider.go:1139-1282 beside RFC 7662.
- Review cmd/tinyidp-xapp/state.go:147-160 to see the access-token-only device
  client and separate confidential resource client.
- Review cmd/tinyidp-xapp/device_cli.go:156-253 to see the cache and polling
  policy actually shipped to agents.

### Technical details

Validation transcript:

~~~text
$ go test ./...
PASS (all packages)

$ go build ./...
PASS

$ go test -race ./internal/fositeadapter ./cmd/tinyidp-xapp -count=1
ok github.com/go-go-golems/tiny-idp/internal/fositeadapter 91.935s
ok github.com/go-go-golems/tiny-idp/cmd/tinyidp-xapp 37.591s
~~~

Standards conclusion:

~~~text
Observed request:  audience=https://app.example.test/api
RFC 8707 request: resource=https://app.example.test/api
Observed token:    aud includes the requested API
Observed check:    introspection client must share that allowed audience
Required action:   adopt resource as the canonical request contract
~~~

## Step 4: Validate, publish, and close the review

This step validated the finished ticket, committed the long-form review, and
published the design plus diary as one reMarkable PDF bundle. The upload used
the required non-interactive dry run before the real operation. The uploader's
success receipt is the delivery verification.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the documentation artifact, preserve a
reviewable Git history, and deliver it to the requested device.

**Inferred user intent:** Leave both a durable repository record and a readable
offline copy, with enough audit detail to reproduce the publication.

### What I did

- Ran docmgr doctor with error failure enabled and git diff --check.
- Marked the seven research, design, and validation tasks complete.
- Committed the finished guide and test diary as commit 2a3005e.
- Dry-ran a two-document reMarkable bundle with a depth-two table of contents.
- Uploaded the actual bundle non-interactively.
- Marked the publication task complete and closed the documentation ticket.

### Why

- The dry run verifies document selection, name, destination, and rendering
  plan without changing the device.
- Bundling the design and diary keeps conclusions beside their provenance.
- Closing tasks and ticket metadata makes the deliverable discoverable as
  completed work rather than an abandoned investigation.

### What worked

- docmgr doctor reported all checks passed.
- git diff --check reported no whitespace errors.
- The dry run selected the intended design and diary and targeted the dated
  ticket directory.
- The actual uploader returned:

~~~text
OK: uploaded TINYIDP Production XGOJA Review.pdf -> /ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001
~~~

### What didn't work

No validation, render, authentication, or upload step failed. The uploader
printed a remote-tree refresh warning before its success receipt; no retry was
needed.

### What I learned

- The final guide contains 1,240 lines and the diary contained 400 lines before
  this delivery entry.
- The deliverable is intentionally a completed review while its P0 findings are
  future implementation work. Closing this ticket does not imply that the
  production gaps are implemented.

### What was tricky to build

- Ticket completion had to mean “review delivered,” not “production product
  shipped.” The index, readiness matrix, and priority list preserve that
  distinction.

### What warrants a second pair of eyes

- Review the P0 ordering and the explicit open decisions before creating
  implementation tickets.
- Confirm whether the existing audience request field must be supported during
  migration; the review deliberately does not assume backwards compatibility.

### What should be done in the future

- Create focused implementation tickets only after the open product decisions
  have owners and acceptance criteria.
- Re-run the validation and production exercise phases after those changes,
  then issue a new readiness decision.

### Code review instructions

- Review commits e31bceb, 9d227de, 2a3005e, and the final delivery commit.
- Open the ticket index, then the design guide, then this diary.
- Confirm tasks.md has no unchecked delivery tasks and docmgr doctor still
  passes.
- Treat the reMarkable success receipt above as the upload verification; do not
  delete or overwrite the remote document because annotations may later exist.

### Technical details

Publication commands:

~~~text
remarquee upload bundle <design> <diary>
  --name "TINYIDP Production XGOJA Review"
  --remote-dir "/ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001"
  --toc-depth 2 --dry-run --non-interactive

remarquee upload bundle <design> <diary>
  --name "TINYIDP Production XGOJA Review"
  --remote-dir "/ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001"
  --toc-depth 2 --non-interactive
~~~
