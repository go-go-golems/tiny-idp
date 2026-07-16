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
    - Path: repo://cmd/tinyidp-xapp/development_app_test.go
      Note: End-to-end device approval and BBS author proof
    - Path: repo://cmd/tinyidp-xapp/device_api.go
      Note: Host-owned bearer BBS API committed in 4699d40
    - Path: repo://cmd/tinyidp-xapp/device_cli.go
      Note: Glazed device login and cached bearer BBS commands committed in d474d3f
    - Path: repo://cmd/tinyidp-xapp/device_cli_test.go
      Note: Deterministic polling, cache, and bearer request tests
    - Path: repo://cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go
      Note: Go-only opaque bearer validation committed in 5e6d279 and 4699d40
    - Path: repo://ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/scripts/run-xapp-device-smoke.sh
      Note: Live tmux smoke harness
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

## Step 2: Build the host resource server and prove the device-to-object path

The implementation now establishes all three client roles and a state-owned
resource client secret, then mounts a separate host-owned bearer API. The
device client is public and requests an audience-bound token; the resource
client is confidential and introspects that opaque token. The BBS only receives
the verified subject as its actor. The browser BBS routes remain xgoja routes
with their original CSRF policy.

The integration test completes the actual provider device verification form,
polls `/idp/token`, verifies the token through RFC 7662 both over the mounted
provider and the xapp in-process transport, and posts a BBS entry. It also
demonstrates that a caller-supplied `actorId` is rejected before dispatch and
that the created post is authored as `dev-alice-subject`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the phased device-auth API vertical slice, commit coherent increments, and retain exact evidence in the diary.

**Inferred user intent:** Turn the design into a real, inspectable application boundary rather than a protocol-only demonstration.

**Commit (code):** `5e6d27968a2ea237e95cc630c6f084a254430379` — "feat(xapp): establish device resource authentication"; `4699d40981799c06919bd84f47ebd4b031765cdf` — "feat(xapp): expose device-authorized BBS API"

### What I did

- Added `internal/resourceauth`, which discovers issuer metadata, calls
  `/introspect` with Basic auth, validates a constrained response, and keeps an
  HMAC-keyed bounded cache.
- Added browser/device/resource registrations and `resource-client.key` to the
  xapp state model; state manifest version is now 2 and old state is rejected.
- Added `GET /api/device/bbs` and `POST /api/device/bbs/posts` in
  `device_api.go`, with route scopes, strict JSON fields, redacted audit events,
  and direct manager dispatch to BBS/community.
- Added state and end-to-end tests. Validation commands included:
  - `go test ./cmd/tinyidp-xapp/... -count=1`
  - `go test ./cmd/tinyidp-xapp -run TestDevelopmentApplicationDeviceTokenPostsToBearerBBSAPI -count=1`

### Why

- A native host endpoint is the correct initial boundary because it keeps the
  resource-client secret and raw bearer token outside generated JavaScript.
- Client IDs and state paths must be stable on restart; a random BCrypt hash
  cannot be passed through generic bootstrap idempotence, so the resource
  client is reconciled by verifying the stored hash against the stable secret.

### What worked

- Full focused suite: `ok github.com/manuel/tinyidp/cmd/tinyidp-xapp` and
  `ok github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/resourceauth`.
- The device token is active only when the device client requests the exact
  configured API audience, and the BBS author comes from introspection `sub`.
- Provider failures are represented as API `503`; malformed/missing bearer is
  `401`; valid bearer missing scope is `403`.

### What didn't work

- A first device request used `resource=...` and returned
  `{"error":"invalid_scope","error_description":"requested scope is not allowed"}`
  before the required `openid` scope was included. tiny-idp requires `openid`
  on device requests.
- After that, `resource=...` did not populate the current provider's Fosite
  audience representation, yielding `{"active":false}` from introspection.
  Existing provider tests use `audience=...`; the implementation and guide now
  use that established contract.
- Strict discovery decoding initially rejected valid standard metadata:
  `json: unknown field "authorization_endpoint"`. Discovery is extensible, so
  only introspection responses remain strict.
- Strict introspection decoding initially omitted the documented `iat` field,
  causing an in-process result of `OutcomeUnavailable`. Adding `iat` preserved
  strictness for truly unknown fields and fixed the result.

### What I learned

- `gojahttp.ActorFromContext` is not needed for the first bearer API because a
  native handler can pass verified principal data to the durable-object manager
  directly. This is a safe temporary architecture and a concrete input to the
  future reusable xgoja principal bridge.
- OIDC discovery and token introspection have different extension rules:
  discovery must tolerate unknown metadata; the constrained security response
  should not.

### What was tricky to build

- The bootstrap secret is persistent while BCrypt hashes are intentionally
  non-deterministic. Re-hashing each startup would falsely conflict with the
  stored client. `reconcileResourceClient` instead creates once, then validates
  immutable client fields and BCrypt-verifies the persisted secret on restart.
- A provider's public device client must include `openid` even when the
  application only cares about OAuth API scopes. This is an existing tiny-idp
  device-flow requirement, now explicit in the registration and guide.

### What warrants a second pair of eyes

- The current `audience` parameter is an implementation contract; evaluate
  RFC 8707 `resource` interoperability before publishing the CLI as a generic
  OAuth client example.
- Review whether `idp.Event.Subject` is acceptable for xapp API audit policy
  or should be a stable keyed hash in deployments with stricter log privacy.

### What should be done in the future

- Implement the Glazed device-login and BBS CLI commands, then add browser
  automation and initialized TLS tests.

### Code review instructions

- Review `internal/resourceauth/resourceauth.go` from `New` through
  `Authenticate`, then `device_api.go` from `authenticate` through `dispatch`.
- Run `go test ./cmd/tinyidp-xapp/... -count=1`.
- Start with `TestDevelopmentApplicationDeviceTokenPostsToBearerBBSAPI` for a
  readable end-to-end protocol trace.

### Technical details

```text
device CLI -> /idp/device_authorization (openid + bbs scopes + audience)
          -> browser /idp/device confirmation
          -> /idp/token opaque bearer
          -> /api/device/bbs/posts
          -> Go RFC 7662 -> principal{sub} -> BBS/community
```

## Step 3: Add the terminal-facing device and BBS commands

The xapp binary now includes the user-facing half of the vertical slice:
`device-login` starts and polls the device flow; `bbs-get` and `bbs-post` use
the resulting bearer token against the host-owned API. All connection values
are explicit Glazed flags. The token cache is a regular 0600 file and refuses
to be used when expired.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the planned implementation through the CLI phase with commit and diary discipline.

**Inferred user intent:** A person should be able to operate the demonstrated device-authenticated durable-object API without writing ad hoc HTTP requests.

**Commit (code):** `d474d3f9d400dd2017d3c59e8daec8eb71aa961f` — "feat(xapp): add device authorization BBS CLI"

### What I did

- Added `cmd/tinyidp-xapp/device_cli.go` and registered three Glazed commands
  from `main.go`.
- `device-login` discovers the issuer, prints the verification URL/code, polls
  on the provider interval, handles `authorization_pending`/`slow_down`, and
  writes an owner-only cache only after a complete success response.
- `bbs-get` and `bbs-post` load the same cache, refuse expired credentials, and
  call the bearer API with the token in `Authorization`.
- Ran `go test ./cmd/tinyidp-xapp/... -count=1`; all packages passed.

### Why

- Keeping these commands in the xapp binary makes the example self-contained
  and shows the precise device-client configuration rather than requiring a
  private curl recipe.

### What worked

- The complete xapp test suite compiled and passed after command registration.
- The cache has no environment-variable dependency and uses owner-only file
  permissions on creation and validation.

### What didn't work

- The initial compact implementation used an incorrect Glazed helper type,
  producing `undefined: fields.Field`. Local API documentation confirmed
  `fields.New` returns `*fields.Definition`; the command helpers were corrected.

### What I learned

- The command surface is sufficient for manual device authorization, but it
  still needs a fake-provider unit suite and a real browser/CLI harness before
  Phase 4 can be called fully verified.

### What was tricky to build

- Polling must preserve the provider's interval and react to `slow_down` while
  never persisting a token on a pending, denied, malformed, or expired result.
  The implementation keeps device code only in process memory and writes only
  the final access token/expiry cache.

### What warrants a second pair of eyes

- Review CLI output conventions and cache placement defaults for the intended
  distribution platform. The default is intentionally local and explicit;
  system-wide credential-store integration is out of scope.

### What should be done in the future

- Add deterministic fake-provider tests and a tmux/browser smoke harness that
  runs `device-login` and `bbs-post` against a live xapp.

### Code review instructions

- Read `deviceLogin`, `writeDeviceTokenCache`, and `callBBSAPI` in
  `cmd/tinyidp-xapp/device_cli.go`.
- Run `go test ./cmd/tinyidp-xapp/... -count=1` and inspect
  `go run ./cmd/tinyidp-xapp --help`.

### Technical details

```text
device-login -> cache(access token, expiry, issuer, audience; mode 0600)
bbs-get/post -> cache expiry check -> Authorization: Bearer -> /api/device/*
```

## Step 4: Execute deterministic CLI tests and a live tmux smoke startup

Phase 4 now has a deterministic fake-provider suite. It proves request
formation, pending polling, `slow_down` interval handling, final token parsing,
owner-only cache mode, and BBS bearer-header construction without sleeping or
requiring a live browser. The implementation uses an injectable poll wait only
for test timing; normal command operation still waits on a context-aware timer.

For Phase 5, the ticket now contains a tmux smoke harness. It starts a real
development xapp with two seeded accounts, prints the exact manual device-login
and BBS commands, retains the shell if startup fails, and uses OIDC discovery
as the correct readiness probe for `serve` mode. The live instance is currently
running in tmux session `tinyidp-xapp-device-smoke` on port 18878.

### Prompt Context

**User prompt (verbatim):** "phase 4 + phase 5"

**Assistant interpretation:** Complete the CLI verification phase and perform the planned end-to-end/regression verification work.

**Inferred user intent:** Move from an implemented demo to evidence that its device flow, CLI, and live process behavior are reliable.

**Commit (code):** `b92d907c511e84d8bd38432f5c54da11c6da633e` — "test(xapp): cover device CLI polling and cache"

### What I did

- Added `device_cli_test.go` for fake discovery/device/token polling and cache/API behavior.
- Added `scripts/run-xapp-device-smoke.sh` under this ticket.
- Started the script in tmux and verified discovery exposes the expected
  issuer, device authorization endpoint, and introspection endpoint.

### Why

- Device authorization has timing states that should not require a 5-second
  test sleep or real human approval to unit-test.
- A real server process remains necessary to catch lifecycle/startup mistakes
  that handler-level tests cannot observe.

### What worked

- `go test ./cmd/tinyidp-xapp -run 'Test(DeviceLoginPollsSlowDownThenCachesOnlySuccess|DeviceTokenCacheAndBBSRequest)$' -count=1` passed.
- The tmux server emitted `tinyidp-xapp development server started` and
  `curl -fsS http://127.0.0.1:18878/idp/.well-known/openid-configuration`
  returned the configured issuer/device/introspection URLs.

### What didn't work

- The first harness used port 8787, which was not a reliable dedicated test
  port. It was moved to 18878.
- The first rerun reused state bound to the old public URL; xapp correctly
  refused the conflicting persistent identity configuration. The state root is
  now port-specific.
- `/healthz` returned 404 because it is only mounted by `serve-initialized`;
  discovery is the correct `serve` readiness probe.

### What I learned

- The test suite can validate the browser-side provider form flow through Go
  HTTP integration, but no Playwright project or MCP runner is configured in
  this workspace. Browser automation and initialized TLS protocol smoke remain
  explicitly open Phase 5 tasks.

### What was tricky to build

- The smoke script must preserve diagnostic output after `go run` exits. It
  now executes `exec zsh` after the server command rather than silently closing
  the tmux pane.

### What warrants a second pair of eyes

- The remaining Playwright and production TLS tasks need a chosen browser
  runner/certificate fixture rather than an improvised headless workflow.

### What should be done in the future

- Complete the three unchecked Phase 5 tasks: Playwright UI regression,
  two-user browser/device switching assertion, and initialized TLS smoke.

### Code review instructions

- Run the focused CLI test command above.
- Run `scripts/run-xapp-device-smoke.sh`, inspect the tmux pane, and follow its
  printed device login/post/read commands with a real browser.

### Technical details

```text
fake provider: pending -> slow_down -> success
live tmux:     discovery ready -> human browser approval -> CLI cache -> BBS API
```

## Step 5: Complete the end-to-end security and browser regression matrix

Phase 5 now has direct evidence across both interfaces. The ticket-owned
Playwright script drives the live browser login form, creates a BBS post
through the unchanged CSRF-protected browser route, and logs out. Go
integration tests create device grants for Alice and Bob, verify that their
posts retain distinct verified subjects, prove that an insufficient-scope
request cannot mutate the durable object, reject malformed and wrong-audience
requests, and validate that changing Alice's password invalidates an unobserved
device token at the actual bearer API. A separate TLS test executes discovery,
device approval, token polling, introspection, and a BBS post in initialized
mode.

### Prompt Context

**User prompt (verbatim):** "DO it all"

**Assistant interpretation:** Finish the outstanding Phase 4 and Phase 5
implementation and verification tasks, keep the ticket evidence reproducible,
and use Playwright where available.

**Inferred user intent:** The example should be demonstrably usable rather
than merely compiling: it needs evidence for browser behavior, security
boundaries, user identity propagation, revocation, and production-shaped TLS
execution.

**Commit (code):** `748fef864efd032d277682291eb48f600f6cc074` —
"test(xapp): complete device authorization verification matrix"

### What I did

- Extended deterministic CLI coverage to reject issuer-mismatched discovery,
  immediately expired device grants, and terminal `access_denied` responses.
- Added `phase5_test.go`. It creates real device grants for Alice and Bob,
  writes their posts, checks the two server-derived subject values, and proves
  a read-only token cannot add the attempted `denied` post.
- Used the real password-change lifecycle (`idpaccounts.SetPassword`) to
  revoke an unobserved Alice token, then observed `401` from `/api/device/bbs`.
  This exercises the actual storage path which removes Fosite access-token
  sessions during password security-artifact revocation.
- Added a TLS initialized-state test and a ticket-owned Python Playwright
  script. The script uses the local Playwright 1.45 package and system
  Chromium; it does not depend on a missing MCP server.
- Started the app in tmux and ran the browser script against
  `http://127.0.0.1:18878`. Ran `go test ./...` after the changes.

### Why

- A resource server must not infer identity from request fields or retain a
  token after a credential-security transition. The test makes both statements
  observable from the application boundary.
- Device authorization spans different clients and different human/machine
  interactions. Browser automation and protocol integration tests cover
  complementary failures; neither replaces the other.

### What worked

- The initial Playwright run completed login and post creation. After choosing
  an exact accessible name for the app-only logout button, the complete browser
  smoke passed.
- `go test ./cmd/tinyidp-xapp -run 'Test(DeviceLogin|DeviceAPI|InitializedTLS)' -count=1`
  passed.
- `go test ./...` passed, including the xapp package in 10.779 seconds.
- The existing browser-route Go integration test continues to prove that a
  mutation without `X-CSRF-Token` returns `403`; the browser smoke proves the
  normal UI still performs the protected mutation successfully.

### What didn't work

- The first local Playwright launch failed inside the sandbox because the
  system Chromium is Snap-packaged and Snap requires a host capability that
  the sandbox intentionally withholds. The same local test was rerun with the
  narrowly scoped host permission and launched successfully.
- The first browser run used a partial accessible name for logout. Two buttons
  matched (`Log out of Local Loop` and `Log out of Local Loop + tiny-idp`), so
  Playwright correctly raised a strict-selector error. The script now uses
  `exact=True`, making the intended app logout explicit and stable.

### What I learned

- There is no Playwright MCP exposed to this agent session. A project-local
  script using the installed Playwright library remains a reviewable,
  repeatable substitute, and it belongs in the ticket's `scripts/` folder.
- Password replacement is a security lifecycle boundary, not just a credential
  update: the SQLite store invalidates Fosite access-token sessions for the
  subject. Testing it through the BBS API catches a class of failures that an
  isolated RFC 7662 decoder test cannot.

### What was tricky to build

- The device resource-authenticator caches positive decisions. To test
  revocation without a cache-timing ambiguity, the test issues a fresh token,
  changes the password before the token's first API request, and then observes
  the `401` outcome. This tests the provider lookup rather than cache expiry.
- TLS initialized mode is configured with the deployed public origin while
  `httptest.NewTLSServer` supplies the transport endpoint. The test correctly
  retains the public audience (`https://app.example.test/api`) and uses the
  test server's trusted TLS client only for transport.

### What warrants a second pair of eyes

- The development application now retains its private SQLite store pointer so
  package-local integration tests can exercise an authentic password-security
  transition. Confirm that keeping this internal test seam is preferable to a
  broader public revocation API; no new runtime endpoint was added.
- The current ticket validates password-change revocation. A future product
  feature exposing operator-initiated OAuth token revocation should add a
  corresponding public API contract and end-to-end test.

### What should be done in the future

- Execute the printed `device-login`, `bbs-post`, and `bbs-get` commands in a
  separate real terminal while the tmux server is running; the protocol pieces
  are covered automatically, but this remains a useful operator walkthrough.
- Complete Phase 6: operator runbook, extraction recommendation, final ticket
  relations/upload, and closure decision.

### Code review instructions

- Read `phase5_test.go` in this order: two-user/scopes, malformed request,
  wrong audience, password-change revocation, TLS initialized flow.
- Run the browser smoke against the tmux process:

  ```sh
  /home/manuel/.pyenv/versions/3.11.3/bin/python \
    ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/scripts/playwright_browser_smoke.py \
    --base-url http://127.0.0.1:18878
  ```

- Run `go test ./...` from the repository root.

### Technical details

```text
Alice device token -> post(author=dev-alice-subject)
Bob device token   -> post(author=dev-bob-subject)
Bob read-only      -> POST denied -> board unchanged
fresh Alice token  -> password change -> Fosite session removed -> API 401

browser form -> application session + CSRF -> /api/bbs/posts -> logout
initialized TLS -> device grant -> token -> RFC 7662 -> /api/device/bbs/posts
```
