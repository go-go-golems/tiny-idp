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

## Step 5: Narrow the initial release to tiny-idp and Message Desk on k3s

This step reopened the completed review ticket for a focused initial-release
design. The broad architecture remains useful background, but the first
implementation now contains one standalone identity provider and one standalone
Message Desk relying party on the existing Hetzner k3s cluster. Device
authorization, agent APIs, xgoja, and multiple applications are explicitly
deferred.

The application target was corrected during the investigation from the xgoja
BBS prototype to examples/tinyidp-message-app. That correction reduces the
required source work because Message Desk already has an external OIDC mode,
durable PKCE transactions, application sessions, messages, logout, probes, and
a hardened local signup implementation.

### Prompt Context

**User prompt (verbatim):** "I want to refine the scope for the initial project to be the following:

- deploy to k3s (see ~/code/wesen/2026-03-27--hetzner-k3s and /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/infrastructure-and-release.md) a tiny-idp instance and an instance of the BBS messaging app that has user signup.
- wedefer the device auth and all that for a second step, multi-app support as well. For now, we want a standalone app using a standalone idp and allow user signup.

Create a separate design doc for that (that can reference things in the first one, it doesn't need to be all that detailed), and upload to remarkable."

**User correction (verbatim):** "or like i think i meant examples/tinyidp-message-app as the app to deploy"

**User follow-up (verbatim):** "Explain that part to me? nd an explicit trusted-Traefik listener mode for both processes so the k3s deployment
  follows the cluster’s normal TLS topology without pretending an HTTPS public origin is a plain local-development
  server."

**Assistant interpretation:** Add a shorter companion design for deploying one
strict tiny-idp and the existing Message Desk as independent processes on k3s,
with provider-owned public signup, while moving the broader device and
multi-application work to later phases.

**Inferred user intent:** Establish a small first release that can reach real
users quickly without importing future agent or platform complexity into the
deployment.

### What I did

- Reopened TINYIDP-PROD-XGOJA-REVIEW-001 and added three focused tasks.
- Read the cluster architecture, application pipeline, Argo CD setup,
  Vault/VSO provisioning playbook, deployment examples, and infrastructure
  knowledge map supplied by the user.
- Inspected the live goja-kanban one-replica/PVC pattern and the
  goja-auth-host-demo OIDC/Vault pattern.
- Traced Message Desk external mode, OIDC client, application store, health,
  signup endpoints, React presentation, and TLS listener checks.
- Confirmed that external_runtime.go deliberately disables registration because
  the relying party has no identity store or account service.
- Wrote a separate 786-line initial deployment design with scope, topology,
  signup sequence, GitOps layout, phases, acceptance tests, decisions, risks,
  and file references.
- Explained the proposed trusted-Traefik listener boundary: public HTTPS and
  Secure cookies remain mandatory while explicitly trusted Traefik terminates
  TLS and forwards only on the private cluster network.

### Why

- Provider-owned signup keeps passwords and identity mutation inside tiny-idp.
- Reusing Message Desk avoids building a new application and preserves its
  existing browser/session/message tests.
- The cluster already has a repeatable image-to-GitOps-to-Argo delivery path;
  the design should use it rather than invent another deployment mechanism.
- A separate narrow document prevents the initial release from inheriting the
  first guide's device, resource-server, and multi-app requirements.

### What worked

- Existing cluster documentation defines the required source/GHCR/GitOps/Argo
  ownership and the one-time Application bootstrap step.
- goja-kanban provides the exact replicas: 1, Recreate, local-path PVC pattern
  needed by SQLite workloads.
- Message Desk already validates that its public origin and external issuer are
  distinct and canonical.
- Its login attempt persists state, nonce, PKCE verifier, return path, and
  expiry before redirect.
- The current embedded signup handler supplies a concrete set of controls to
  move to the provider-owned interaction.
- docmgr created the companion design cleanly and doctor accepted its anchored
  local and cross-repository references.

### What didn't work

- A read attempted examples/tinyidp-message-app/README.md, which does not exist,
  and returned exactly:

~~~text
sed: can't read examples/tinyidp-message-app/README.md: No such file or directory
~~~

  The relevant deployment documentation is
  examples/tinyidp-external-message-desk/README.md.
- An official Traefik documentation search returned no result content. The
  design therefore does not depend on an unverified ingress annotation; it
  specifies the application trust contract and leaves live proxy CIDR/hop
  validation as an acceptance item.

### What I learned

- The corrected Message Desk target already proves the standalone IdP/RP
  boundary in its external Docker development profile.
- External signup cannot be enabled by flipping registrationEnabled: the
  handler would have no accounts service and would correctly return unavailable.
- The safe continuation is an OIDC-bound provider signup flow, not an
  application credential for an account-creation API.
- Both production commands currently equate an HTTPS public origin with direct
  TLS in the pod. The cluster normally terminates public TLS at Traefik, so that
  trust mode must be explicit rather than inferred or weakened.
- The two SQLite stores require separate PVCs, one replica, and Recreate.

### What was tricky to build

- “Standalone” has two meanings that had to be preserved together: separate
  processes and stores, but one browser flow. Binding registration to the
  existing OIDC interaction lets the provider own the password while Message
  Desk still owns state, nonce, PKCE, and its application session.
- Kubernetes Secret volume modes do not satisfy tiny-idp's owner-only token
  file check for a non-root process by default. The design uses an init copy
  into an emptyDir with process ownership and mode 0600 instead of weakening
  the check.
- The local-path PVC must be applied with its first consumer to avoid the
  WaitForFirstConsumer Argo deadlock recorded in the infrastructure docs.

### What warrants a second pair of eyes

- Review the exact provider registration interaction state and restart
  behavior before implementation.
- Review the trusted Traefik CIDRs, forwarded-address hop limit, and
  NetworkPolicy against the live cluster.
- Confirm the proposed public hostnames and whether the images are public.
- Confirm that deferring email verification and recovery is acceptable for the
  first public signup release.
- Review the online backup mechanism for the Message Desk SQLite database.

### What should be done in the future

- Implement the focused phases before returning to device or multi-app work.
- Create the device/agent phase as a separate ticket after browser signup,
  login, messaging, backup, and restore are proven live.

### Code review instructions

- Begin with design-doc/02-initial-k3s-deployment-design-for-standalone-tiny-idp-and-message-desk.md.
- Compare its current-state claims with external_runtime.go, oidc_client.go,
  app_http.go, and commands.go.
- Compare its manifests with goja-kanban and goja-auth-host-demo in the k3s
  repository.
- Run docmgr doctor for the ticket and git diff --check before publication.

### Technical details

Focused dependency chain:

~~~text
provider signup
  -> external OIDC Message Desk
  -> two production listener modes
  -> two immutable images
  -> Vault token secret + two PVCs
  -> Kustomize + Argo Application
  -> browser smoke + backup/restore
~~~

## Step 6: Publish the focused initial-release design

This step published the Message Desk design as a separate PDF. It did not
replace the earlier broad production review, so existing reMarkable annotations
remain safe and the initial scope can be read independently.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Deliver the narrower design as its own
reMarkable artifact and close the refinement tasks.

**Inferred user intent:** Make the first implementation plan easy to review
without requiring readers to navigate the larger future-platform analysis.

### What I did

- Committed the focused design and diary checkpoint as e66ce9a.
- Ran a non-interactive dry run for a single-document Markdown upload.
- Uploaded the rendered PDF into the existing dated ticket directory.
- Completed the publication task, reran docmgr doctor, and closed the ticket
  again after the refinement.

### Why

- A separate PDF matches the user's request for a separate design.
- Reusing the ticket directory preserves the relationship to the first review.
- Avoiding force protects the earlier document and any annotations.

### What worked

The uploader returned the explicit success receipt:

~~~text
OK: uploaded TINYIDP_Message_Desk_K3S_Initial_Design.pdf -> /ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001
~~~

### What didn't work

No dry-run, render, authentication, or upload step failed.

### What I learned

- The dedicated initial-scope document renders successfully as a standalone
  reMarkable artifact.

### What was tricky to build

- The new PDF needed a distinct name so publication would not overwrite the
  earlier broad bundle. The upload intentionally omitted force.

### What warrants a second pair of eyes

- Review the trusted-proxy proposal and the decision to defer email
  verification/recovery before implementation begins.

### What should be done in the future

- Use the focused design as the source for implementation tickets and leave the
  first review as the roadmap for later device and multi-app phases.

### Code review instructions

- Review e66ce9a, then the final publication bookkeeping commit.
- Run docmgr doctor --ticket TINYIDP-PROD-XGOJA-REVIEW-001.
- Confirm tasks.md reports zero open tasks.

### Technical details

Publication shape:

~~~text
source: design-doc/02-initial-k3s-deployment-design-for-standalone-tiny-idp-and-message-desk.md
name:   TINYIDP Message Desk K3S Initial Design
target: /ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001
mode:   dry-run, then non-interactive upload
~~~

## Step 7: Review PR 98 and write the remaining-hardening implementation chapter

This step revisited the device-auth assessment after PR 95 merged and PR 98
opened. PR 98 materially improves durability and application credential
lifecycle, but it does not by itself establish trustworthy ingress identity,
budgets for native device endpoints, live dependency readiness, or an
owner-facing agent lifecycle.

The resulting chapter teaches those boundaries before prescribing code. It
keeps the application-owned programauth flow separate from the future
tiny-idp-issued resource-server flow, and it corrects the assumption that
current tiny-idp lacks authenticated introspection.

### Prompt Context

**User prompt (verbatim):** "make a document for the implementer of #98 about what they should add and why. Upload to remarkable. Explain why in detail in a textbook authoring style, they are an intern."

**Assistant interpretation:** Write a detailed, technically precise teaching
chapter for the PR 98 implementer, integrate it into the existing ticket, and
publish it as a standalone reMarkable document.

**Inferred user intent:** Give a less-experienced implementer enough conceptual
foundation, code landmarks, pseudocode, sequencing, and validation criteria to
finish the production boundary without cargo-culting a checklist.

### What I did

- Re-read the merged PR 95 baseline and inspected all 52 files changed by PR
  98 in the referenced go-go-goja checkout.
- Traced hostauth configuration, single-node preflight, store construction,
  static readiness, device handlers, token-pair transactions, security events,
  rate-limit IP selection, audit forwarded-IP selection, and the strict
  tiny-idp documentation.
- Verified against tiny-idp `origin/main` that production discovery advertises
  device authorization and authenticated RFC 7662 introspection.
- Added a 1,080-line textbook-style guide with conceptual foundations,
  sequence diagrams, comparison tables, Go and JavaScript sketches,
  pseudocode, implementation phases, tests, a review map, and a completion
  checklist.
- Related the seven primary implementation files through docmgr.
- Validated frontmatter, ran docmgr doctor, and ran `git diff --check`.

### Why

- PR 98 solves durable state and refresh lifecycle, but those properties do
  not determine whether forwarding headers are trustworthy or native public
  endpoints are bounded.
- An intern needs to understand the difference between public origin, private
  listener, proxy peer, and original client before implementing rate limits or
  audit attribution.
- The device flow needs application policy around requested actions; protocol
  mechanics alone cannot decide which application permissions may be issued.
- Kubernetes requires dependency-aware readiness, not an unconditional
  topology declaration.
- Credential management must be derived from the authenticated owner rather
  than from user-supplied owner IDs.

### What worked

- The PR 98 checkout contained the complete pushed head and its ticket
  documentation, allowing every recommendation to be tied to a current symbol
  and behavior.
- The existing production review ticket was a natural home for the follow-up
  chapter and already contained the broader and phase-one contexts.
- Docmgr accepted all local and cross-workspace file relationships and reported
  all checks passed.
- The textbook structure made it possible to explain why each security
  boundary exists before presenting APIs and tests.

### What didn't work

No document creation, relationship, frontmatter, or doctor command failed.
The PR status inspection initially requested an unsupported `gh pr view` JSON
field in the prior analysis step and returned:

~~~text
Unknown JSON field: "baseRefOid"
~~~

The query was rerun with supported fields and returned the complete PR state.

### What I learned

- `auth.oidc.public-base-url` correctly separates the browser-visible HTTPS
  origin from the private HTTP listener, but PR 98 has no trusted-proxy CIDR or
  canonical client-address resolver.
- Audit trusts the first `X-Forwarded-For` value while the rate limiter uses
  `RemoteAddr`; the two systems disagree behind Traefik.
- Native `/auth/device/*` handlers do not traverse planned Express route rate
  limits.
- `/auth/readyz` reports resolved topology and deliberately does not ping SQL.
- PR 98 supplies a metrics observer interface, but its default implementation
  is an in-memory test counter rather than a production exporter.
- Tiny-idp `main` already advertises and implements the introspection contract
  that PR 98's resource-server reference describes as missing.

### What was tricky to build

- The guide had to distinguish a real production blocker from a deliberate
  bounded semantic. For example, refresh-family revocation intentionally
  leaves existing access tokens valid until their short expiry; the missing
  part is honest disconnect UX and an immediate agent-disable operation, not a
  claim that every revocation must be instantaneous.
- Trusted proxy processing cannot be reduced to reading the leftmost forwarded
  address. The document specifies peer validation and right-to-left trusted-hop
  processing so the implementer understands the attack surface.
- Readiness needed three separate concepts: process liveness, required
  dependency health, and non-secret topology reporting.
- The native device approval UI needs a server-owned inspection API. The guide
  uses a POST body rather than a query parameter so short user codes do not
  enter URL logs and browser history.

### What warrants a second pair of eyes

- Review the exact Traefik source CIDRs and forwarded-header behavior against
  the live k3s cluster before finalizing `ProxyConfig`.
- Review whether unknown requested actions should be rejected or whether any
  application requires a documented narrowing behavior.
- Review whether local user disablement should automatically disable all owned
  agents and how tiny-idp account state reaches the application.
- Review whether SQL alone is a hard readiness dependency or whether issuer
  availability should also gate traffic.
- Review the proposed retention periods before implementing cleanup SQL.

### What should be done in the future

- Implement the chapter's phases in order: request identity, native endpoint
  policy, readiness, owner lifecycle, then operations and deployment proof.
- Keep `production-ha` and the reusable tiny-idp introspection authenticator as
  separate, explicitly scoped follow-up work.

### Code review instructions

- Begin with
  `design-doc/03-pr-98-production-hardening-implementation-guide-for-xgoja-hostauth.md`.
- Follow its Section 12 review map from hostauth config through native handlers,
  token services, SQL transitions, limiter, audit, and readiness.
- Run `docmgr doctor --ticket TINYIDP-PROD-XGOJA-REVIEW-001 --stale-after 30`
  and `git diff --check` for this documentation change.
- For implementation, run the focused Go package tests, race tests, full build,
  full test suite, lint, and the Traefik-shaped strict smoke listed in Section
  11.

### Technical details

Recommended dependency order:

~~~text
canonical request identity
  -> native endpoint budgets
  -> allowed action policy
  -> durable device transition
  -> dependency-aware readiness
  -> owner-scoped disconnect
  -> metrics, retention, migration, deployment proof
~~~

## Step 8: Publish the PR 98 implementation guide

This step committed the validated implementation chapter and uploaded it as a
standalone PDF. The separate artifact preserves the earlier broad architecture
review and phase-one k3s design while giving the PR 98 implementer a focused
document to annotate.

### Prompt Context

**User prompt (verbatim):** (same as Step 7)

**Assistant interpretation:** Publish the completed intern-facing chapter to
the ticket's reMarkable directory and retain an exact receipt.

**Inferred user intent:** Make the implementation guidance available for
offline review and annotation without replacing prior documents.

**Commit (documentation):** 45f0fdc56acc23210dde803804be6430ac7a9d0c —
"Docs: add PR 98 production hardening guide"

### What I did

- Reviewed the staged file list and statistics before committing.
- Committed only the guide, index link, tasks, changelog, and diary checkpoint.
- Rendered the single Markdown guide with a depth-two table of contents.
- Uploaded it non-interactively to the existing dated ticket directory without
  force.

### Why

- A standalone PDF keeps the implementation sequence easy to navigate.
- Avoiding force protects existing documents and annotations.
- Recording the commit and upload receipt makes the published source exact.

### What worked

The uploader returned:

~~~text
OK: uploaded PR 98 Production Hardening Implementation Guide.pdf -> /ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001
~~~

### What didn't work

No render, authentication, or upload step failed.

### What I learned

- The 1,100-line chapter renders successfully as one reMarkable PDF with its
  code blocks, tables, diagrams, and checklist intact.

### What was tricky to build

- The document needed a distinct upload name to avoid replacing the previous
  two ticket artifacts. The upload intentionally omitted `--force`.

### What warrants a second pair of eyes

- Review the rendered tables and long Go signatures on-device; the Markdown
  source is authoritative if a narrow page wraps a code line.

### What should be done in the future

- Use annotations from the implementer review to refine the implementation
  phases, not to blur the app-owned and IdP-owned token models.

### Code review instructions

- Review commit 45f0fdc and this publication bookkeeping commit.
- Confirm docmgr reports no open ticket tasks and doctor reports all checks
  passed.

### Technical details

~~~text
source: design-doc/03-pr-98-production-hardening-implementation-guide-for-xgoja-hostauth.md
name:   PR 98 Production Hardening Implementation Guide
target: /ai/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001
mode:   bundle, ToC depth 2, non-interactive, no force
~~~

## Step 9: Add the complete project and deployment context

The first edition assumed the reader already understood why hostauth existed.
That was the wrong starting point for the intended implementer. This revision
adds the project context before the security work so an intern can place every
change in the correct repository and ownership boundary.

The guide now explains tiny-idp, Message Desk, go-go-goja, xgoja, hostauth, the
Personal Knowledge Inbox reference application, the Hetzner k3s cluster, and
the staged product plan. It explicitly states that go-go-goja is a
general-purpose runtime and module repository rather than the product or IdP.

### Prompt Context

**User prompt (verbatim):** "explain what tiny-idp / the cluster / all that is. go-go-goja is a general purpose set of goja modules + runtime stuff. The implementor doesn't know anything about what we are trying to do."

**Assistant interpretation:** Rewrite the beginning of the PR 98 guide for a
reader with no project history, while preserving the detailed implementation
plan after the new foundation.

**Inferred user intent:** Prevent an intern from treating tutorial composition,
product policy, identity-provider behavior, reusable runtime code, and cluster
manifests as one undifferentiated system.

### What I did

- Added a system-responsibility table distinguishing tiny-idp, Message Desk,
  go-go-goja, xgoja, hostauth, the reference inbox, and the k3s cluster.
- Explained tiny-idp account ownership, browser OIDC flow, native device flow,
  and authenticated introspection.
- Explained the narrower first production phase and why device auth is not on
  its critical path.
- Explained go-go-goja as reusable Goja modules and runtime infrastructure,
  xgoja generated composition, planned Express routes, and optional hostauth.
- Explained why the Personal Knowledge Inbox is a progressive reference rather
  than a production policy definition.
- Added the source-image-GitOps-Argo-Traefik deployment path and the
  responsibilities of cert-manager, Vault/VSO, local-path storage, Services,
  Deployments, and NetworkPolicy.
- Added the full later xgoja topology and a working-directory/repository map.
- Revalidated frontmatter, docmgr health, and whitespace.

### Why

- File placement decisions depend on understanding ownership. A proxy resolver
  belongs in reusable host infrastructure; an inbox action allowlist belongs
  to the application; passwords belong to tiny-idp; NetworkPolicy belongs to
  GitOps.
- The intern must know that PR 98 prepares an optional later capability and
  does not define all of go-go-goja or block the first Message Desk release.
- Explaining the cluster makes the trusted-Traefik and single-replica
  requirements concrete rather than abstract web-server recommendations.

### What worked

- The existing phase-one design and PR 98 code review supplied a consistent
  system map without requiring new experiments.
- The document grew from 1,100 to 1,527 lines while preserving the detailed
  hardening sections unchanged.
- Frontmatter validation and docmgr doctor both passed after the expansion.

### What didn't work

No patch, validation, or documentation command failed during this revision.

### What I learned

- The implementer's most important first distinction is not session versus
  bearer token; it is reusable runtime infrastructure versus product-specific
  policy.
- The first Message Desk deployment and the later xgoja agent platform share
  tiny-idp and cluster concepts but have different release criteria.
- `public-base-url`, trusted proxy identity, and private listener address become
  easier to understand once the actual Traefik topology is drawn.

### What was tricky to build

- The orientation needed enough detail to be self-contained without turning
  the PR guide into a second copy of the full architecture review. The revision
  focuses on responsibilities, flows, and repository placement, then points
  into the existing hardening sequence.
- Tiny-idp has its own device flow while PR 98 has an application-owned device
  flow. Both had to be introduced early without implying that their tokens are
  interchangeable.

### What warrants a second pair of eyes

- Confirm that the distinction between the first Message Desk phase and the
  later xgoja phase matches the current implementation schedule.
- Confirm that the listed working directories remain the canonical checkouts
  for the intern.
- Review whether the cluster overview should name concrete public domains once
  they are selected.

### What should be done in the future

- Keep the orientation current when the first Message Desk deployment lands or
  a production xgoja application replaces the Personal Knowledge Inbox as the
  reference target.

### Code review instructions

- Read the guide from the beginning through “What the PR 98 implementer is
  responsible for” before reviewing Section 1.
- Verify the responsibility table against the three repositories rather than
  moving application or cluster policy into hostauth.
- Run docmgr frontmatter validation, doctor, and `git diff --check`.

### Technical details

The revised teaching sequence is:

~~~text
product goal
  -> systems and ownership
  -> tiny-idp browser and device roles
  -> first Message Desk phase
  -> general go-go-goja/xgoja runtime
  -> reference inbox
  -> k3s/GitOps/Traefik deployment
  -> PR 98 scope
  -> detailed hardening plan
~~~
