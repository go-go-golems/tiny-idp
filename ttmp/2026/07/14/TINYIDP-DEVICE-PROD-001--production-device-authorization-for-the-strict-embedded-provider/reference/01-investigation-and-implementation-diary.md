---
Title: Investigation and Implementation Diary
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - architecture
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: examples/embedded/app.go
      Note: PKCE browser flow and application sessions
    - Path: examples/embedded/app_test.go
      Note: Complete relying-party callback harness
    - Path: examples/embedded/main.go
      Note: One-origin IdP and relying-party composition
    - Path: examples/embedded/oidc.go
      Note: Back-channel exchange and ID-token verification
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
Summary: Chronological evidence log for the production device design and self-contained embedded relying-party implementation.
LastUpdated: 2026-07-14T18:25:00Z
WhatFor: Preserves commands, findings, mistakes, decisions, and validation so another engineer can continue without reconstructing the investigation.
WhenToUse: Read at handoff, before implementing a task, or when reviewing why the design chose durable atomic state and Fosite integration.
---


# Investigation and Implementation Diary

## 2026-07-14 13:45 EDT — Request decomposition

### Goal

The request contained two work streams:

1. create a durable ticket and intern-facing design for production RFC 8628
   device authorization in the strict provider;
2. correct `examples/embedded`, which registered a callback on a different
   application origin without implementing that relying party.

I treated the first as research and design, not authorization to implement the
large security feature. I treated the second as an explicit implementation
request. This separation prevents an incomplete device grant from being
advertised while still producing a complete runnable embedding example.

### Workspace state

Repository: `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp`

Branch: `task/prod-tiny-idp`

The worktree already contained two unrelated untracked OIDF source directories
under `TINYIDP-PROD-001`. They were not read, edited, staged, or removed.

## 2026-07-14 13:55 EDT — Ticket creation

Created ticket `TINYIDP-DEVICE-PROD-001`, titled **Production Device
Authorization for the Strict Embedded Provider**.

Created:

- `design-doc/01-production-device-authorization-analysis-design-and-implementation-guide.md`;
- `reference/01-investigation-and-implementation-diary.md`;
- standard ticket index, task, changelog, and README files.

The initial generated documents contained templates. I kept them only long
enough to inspect the expected frontmatter, then replaced the bodies with the
design and this diary.

## 2026-07-14 14:00 EDT — Primary-source collection

I used Defuddle because the user explicitly requested retraceable sources and
the workflow calls for cleaned Markdown under `sources/`.

Preserved:

- RFC 8628, OAuth 2.0 Device Authorization Grant;
- RFC 9700, Best Current Practice for OAuth 2.0 Security;
- RFC 8414, OAuth 2.0 Authorization Server Metadata;
- OpenID Connect Discovery 1.0.

The first attempt to parse the RFC Editor presentation of RFC 8628 failed with
`No content could be extracted`. I retried using the IETF Datatracker HTML
publication, which Defuddle successfully converted. This failure matters
because it explains why the saved source URL differs from the first URL tried.

Key requirements extracted from RFC 8628:

- the authorization server exposes a device authorization endpoint;
- the response returns device code, user code, verification URI, expiry, and
  optionally complete verification URI and polling interval;
- polling before approval returns `authorization_pending`;
- overly frequent polling returns `slow_down` and adds five seconds to the
  polling interval;
- denial and expiry are terminal;
- user-code brute force requires finite lifetime and rate limiting;
- device codes require high entropy;
- complete verification URIs do not remove the need to show and confirm the
  code because remote phishing remains relevant.

## 2026-07-14 14:05 EDT — Existing device code inspection

Inspected `internal/server/device.go`, `internal/server/token.go`, discovery,
tests, and client fixtures.

Findings:

- The mock server already models pending, approved, and denied grants.
- It returns appropriate pending, slowdown, expiry, denial, and invalid-grant
  errors for development tests.
- It uses an eight-character human code with an ambiguity-reduced alphabet.
- State is a process-local `deviceGrants` map.
- Raw codes remain in memory and user-code lookup scans the map.
- Decisions do not participate in a durable token transaction.
- Authentication uses scenario users rather than production accounts.
- Empty verification action behavior is too permissive for the strict flow.

Conclusion: retain it as a mock and source of test cases; do not promote it into
`internal/fositeadapter`.

## 2026-07-14 14:10 EDT — Strict-provider extension analysis

Inspected:

- `internal/fositeadapter/provider.go` route and factory assembly;
- `internal/fositeadapter/sqlstore.go` token/client persistence;
- Fosite v0.49.0 `TokenEndpointHandler`;
- Fosite authorization-code, refresh, and OpenID handlers;
- `pkg/idpstore/interfaces.go` and `types.go`;
- SQLite migrations, maintenance, backup, and verification;
- `pkg/embeddedidp/bootstrap.go` and `options.go`.

Fosite has no built-in RFC 8628 handler in this version. It does provide the
correct extension interface. `CanSkipClientAuth` must remain false: public
client identification is not equivalent to omitting client identity.

The public store already encourages named transitions through transactional
interfaces. That pattern led directly to the design for `PollDeviceGrant`,
`DecideDeviceGrant`, and `ConsumeDeviceGrant` instead of a callback-based
generic update.

The central atomicity problem found during this step was:

```text
consume grant, then fail token writes  => approved grant is lost
write tokens, then fail grant consume  => one device code can replay
```

Therefore consumption and Fosite token persistence must share one SQLite
transaction. A standalone token endpoint was rejected because it would create
a second token format and lifecycle.

## 2026-07-14 14:25 EDT — Embedded example gap

The old `examples/embedded/main.go`:

- served only `provider.Handler()` on `127.0.0.1:5556`;
- registered `http://127.0.0.1:8080/auth/callback`;
- implemented nothing on port 8080;
- demonstrated discovery through an in-process transport only in a separate
  package example.

This meant a reader could start the example and view an IdP, but could not run
the registered OIDC application.

Chosen composition:

```text
one process / one public origin: http://127.0.0.1:5556
    /                    relying-party home
    /login               begin Authorization Code + PKCE
    /auth/callback       implemented callback
    /logout              local + RP-initiated logout
    /idp/*               embedded tiny-idp provider
```

Browser redirects use normal HTTP paths. Discovery, token exchange, JWKS, and
UserInfo are dispatched by `NewInProcessIssuerTransport`, which admits only the
exact issuer origin and has no network fallback.

## 2026-07-14 14:35 EDT — Relying-party implementation

Replaced `examples/embedded/main.go` and added:

- `app.go`: handlers, PKCE/state/nonce generation, transient login flows,
  application sessions, CSRF-protected logout, and HTML;
- `oidc.go`: discovery, code exchange, bounded JSON, RS256/JWKS verification,
  issuer/audience/expiry/nonce checks, and UserInfo retrieval;
- `app_test.go`: a complete callback harness with a generated RSA key and fake
  exact endpoint transport.

Security properties implemented in the example:

- 256-bit random state and nonce;
- high-entropy PKCE verifier and S256 challenge;
- opaque HttpOnly SameSite application cookies;
- one-time, five-minute login-flow consumption;
- exact state check before back-channel activity;
- bounded one-megabyte OIDC responses;
- discovery issuer equality;
- JWT compact structure, `alg=RS256`, `kid`, JWK type/use/algorithm checks;
- RSA signature, issuer, subject, audience, expiration, issued-at, and nonce
  validation;
- UserInfo subject equality with the ID token;
- independent eight-hour application session;
- POST + CSRF logout and RP-initiated IdP logout;
- CSP, no-referrer, and nosniff response headers.

This is intentionally still a development example. Its session store is
in-memory and its public URL is loopback HTTP. The production xapp demonstrates
durable application sessions and HTTPS validation.

## 2026-07-14 14:50 EDT — Build and test corrections

First test command:

```text
gofmt -w examples/embedded/main.go examples/embedded/app.go examples/embedded/oidc.go
go test ./examples/embedded
go test ./pkg/embeddedidp ./internal/fositeadapter
```

Observed issues:

1. A constant expression used to guard conversion of the JWK exponent
   overflowed `uint32`. Replaced it with the explicit `int32` upper bound.
2. Broader packages attempted to create `httptest` listeners, which the
   filesystem/network sandbox prohibited. This was an environment restriction,
   not a product failure.
3. The shared Go build cache was read-only inside the sandbox. Retried the
   example test with approved normal cache access, per the repository's
   `go.work` guidance.
4. The new test initially assigned a string to a byte slice while trimming
   leading exponent zeroes. Replaced it with an in-place slice loop.
5. The deterministic JWT test timestamp was in the future relative to the
   verifier's real clock. Aligned the fixture to current UTC time.

Final focused result:

```text
ok github.com/manuel/tinyidp/examples/embedded
```

The callback test also proves a state-mismatch consumes the transient flow and
does not initiate a back-channel request.

## 2026-07-14 15:05 EDT — Real-process smoke test

Per repository instructions, attempted to start the server in tmux. The first
sandboxed attempt failed:

```text
error connecting to /tmp/tmux-1000/default (Operation not permitted)
```

Retried with approval and started session `tinyidp-embedded-smoke`.
Captured output:

```text
self-contained app listening at http://127.0.0.1:5556
```

Loopback checks established:

- `GET /` returned 200 and the relying-party home page;
- `GET /login` returned 302 to `/idp/authorize` on the same origin;
- the authorization request included exact callback, state, nonce, and S256
  challenge;
- the IdP rendered its login/consent form;
- submitting `alice / correct horse battery staple` completed the real code
  exchange and established the application session;
- authenticated `/` displayed `Alice Example`, email, and the stable subject.

One curl invocation explicitly set `-X POST` while following redirects. Curl
therefore retained POST at the final `/` redirect and received `Method Not
Allowed`. This was a smoke-driver mistake, not an application failure. The
cookie jar already contained the successfully established session; a normal
GET verified it.

Stopped the process with the required command:

```text
lsof-who -p 5556 -k
```

## 2026-07-14 15:20 EDT — Design synthesis

Wrote the design from the primary specifications and repository evidence. The
design connects protocol requirements to concrete public types, Fosite
extension points, SQL constraints, transitions, rendering boundaries,
operational controls, and test harnesses. It includes phased work and a release
checklist but deliberately does not claim that device authorization is already
implemented.

## Review instructions

For the embedded example:

```bash
go test ./examples/embedded
go run ./examples/embedded
```

Open `http://127.0.0.1:5556`, sign in with the documented development account,
verify the application profile, then use the POST logout button. Confirm that a
new sign-in requires credentials after coordinated logout.

For the device ticket, begin with Phase 0 and Phase 1. Do not add discovery
metadata until durable creation, verification, polling, and token issuance are
complete behind the strict provider.

## 2026-07-14 15:35 EDT — Final validation, commits, and publication

The complete repository suite passed with normal loopback and shared-cache
access:

```text
go test ./...
```

Created two scoped commits with `LEFTHOOK=0`:

- `b2f31d3 feat(examples): make embedded host a complete OIDC app`
- `7d46268 docs(device): design production authorization grant`

The first sandboxed staging attempt could not create the linked-worktree
`index.lock` because the actual Git directory is outside the writable workspace.
Staging and committing were repeated with the required approval. Only the five
example files, the new ticket, and its vocabulary entry were staged. The two
pre-existing untracked OIDF directories remained untouched.

Docmgr frontmatter validation passed, and `docmgr doctor` reported all checks
passed after adding the `oauth2` topic vocabulary and notes for every related
file.

Uploaded one combined PDF containing the ticket index, design, diary, tasks,
and four preserved specifications:

```text
/ai/2026/07/14/TINYIDP-DEVICE-PROD-001/
TINYIDP Device Production Design Bundle.pdf
```

### Remaining implementation work

- Implement the device phases tracked in `tasks.md`.
- Keep device discovery advertisement disabled until the endpoint, browser
  verification, and atomic Fosite token issuance phases pass together.
- Obtain an independent security review before declaring RFC 8628 production
  ready.

## Step 1: Freeze executable device-grant security contracts

The existing design describes the production device authorization architecture,
but Phase 0 was deliberately left open until every important security choice
had a concrete test, transition, or observability consequence. This step adds
that compact specification before any client field, migration, or endpoint.

### Prompt Context

**User prompt (verbatim):** "ok, implement the ticket, task by task, commit at appropriate intervals."

**Assistant interpretation:** Execute the production device-authorization
ticket in dependency order with focused commits and continuation-quality docs.

**Inferred user intent:** Turn the researched strict-provider device design
into a production-quality implementation rather than reuse the mock server.

### What I did

- Added `reference/02-device-grant-executable-security-specification.md`.
- Fixed grant constants, code entropy/lifetime, poll behavior, refresh
  capability, client-profile, state-machine, keyed-hash, endpoint, token,
  audit, metric, and release-gate contracts.
- Named the tests later phases must add before discovery can advertise device
  authorization.
- Completed the three Phase 0 task entries.

### Why

- Device authorization combines a bearer credential, human code, browser
  authentication, polling, and token issuance. Implicit contracts risk replay,
  logging, or authority expansion in an otherwise valid-looking endpoint.

### What worked

- The existing production design, preserved RFC sources, mock behavior, and
  strict Fosite/store analysis gave sufficient evidence for a test-oriented
  specification without changing runtime behavior.

### What didn't work

- Two initial document patches were rejected with `invalid hunk ... is not a
  valid hunk header` because wrapped lines lacked an add marker. No file was
  changed; a compact line-oriented patch applied successfully.

### What I learned

- The mock endpoint behavior is a useful test source, but its map/raw-code
  state cannot influence strict implementation. The strict provider needs
  keyed hashes, named durable transitions, and one Fosite token lifecycle.

### What was tricky to build

- Atomic consumption is the key invariant: consume-first can lose an approved
  grant after token persistence fails, while persist-first can leave a replayable
  device code. The specification requires one transaction boundary.

### What warrants a second pair of eyes

- Review the fixed ten-minute lifetime and five-second initial interval before
  they become external compatibility commitments.
- Review the migration rule: ambiguous historical clients must fail closed.

### What should be done in the future

- Implement Phase 1: explicit client grant capabilities, SQLite migration,
  bootstrap profiles, Fosite adaptation, and negative tests.

### Code review instructions

- Read the new specification with design sections 5 through 12.
- Confirm this increment changes no discovery, endpoint, token, or DB behavior.

### Technical details

```text
raw device/user codes -> domain-separated HMAC hashes -> durable grant
pending -> approved|denied|expired -> consumed once with Fosite tokens
```

## Step 2: Make client grant capability explicit, validated, and durable

Phase 0 established that no endpoint may infer authority from a client shape.
This step implements that principle before device-grant records or public
routes exist. Every real strict-provider client now carries a finite list of
OAuth grants, and every provisioning surface must supply it.

### What I did

- Added `AllowedGrantTypes` and the three currently supported grant constants
  to `pkg/idpstore.Client`.
- Made empty, unsupported, and duplicate declarations fail validation; added
  `AllowsGrantType` as the future endpoint's policy predicate.
- Added SQLite migration `008_client_grant_capabilities.sql`. It classifies a
  legacy client with a redirect URI as browser (`authorization_code`,
  `refresh_token`), a redirect-less public PKCE client as device
  (`device_code`), and all ambiguous records as an empty capability list.
- Added a restart migration test that removes the new JSON field from three
  stored clients and proves the database receives those exact results.
- Made `BrowserClient` and `DeviceClient` set and enforce their exact profile
  grants, and added the field to bootstrap drift detection.
- Propagated the field into both memory and SQLite Fosite `DefaultClient`
  projections. The adapter has no fallback grant list.
- Extended the admin service and `tinyidp admin client create` with a required,
  repeatable `--grant-type`; created clients are validated in production mode.
- Updated strict-provider fixtures, review probes, the external-consumer test,
  and the legacy strict development adapter to use explicit browser grants.

### Why

- Fosite evaluates a client's grant list at token issuance. Leaving an adapter
  default would silently grant a newly introduced client more authority than
  its persisted configuration declares.
- Migration ambiguity is an operational security decision. A historical
  confidential or atypical client cannot safely be guessed, so it must fail
  closed and be repaired by an operator.

### What worked

- `go test ./pkg/idpstore ./pkg/sqlitestore ./pkg/embeddedidp ./internal/admin
  ./internal/cmds ./internal/fositeadapter
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts
  ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/external-consumer`
  passed after the changes.
- The migration test exercised close/reopen, migration-ledger replay, browser
  classification, device classification, and ambiguous-client fail-closed
  validation.
- The Fosite adapter test proves a device-only configuration remains
  device-only at the protocol library boundary.

### What didn't work

- The first affected-suite run exposed direct embedded-provider test clients
  with no grant declaration. Updating the fixtures explicitly fixed the
  expected consequence of removing the implicit adapter list.
- A first Fosite projection assertion compared `fosite.Arguments` directly
  with `[]string`; the values were correct but Go's distinct slice types are
  not deeply equal. Converting to `[]string` made the intended assertion exact.
- `go test ./...` still fails only in `cmd/tinyidp-xapp` on existing interaction
  CSP/header expectations (`form-action` includes its test-server origin).
  Phase 1 did not edit that application or its UI/header code; its own package
  had already been failing independently of grant-type work. All packages
  touched by this increment pass.
- A focused `docmgr validate frontmatter` invocation initially used positional
  arguments rather than its required `--doc` flag; the corrected diary
  validation passed. `changelog.md` intentionally has no frontmatter, so its
  focused validation reports missing delimiters while ticket-level
  `docmgr doctor` correctly passes.

### What I learned

- The former hard-coded Fosite list appeared in both in-memory and SQLite
  projection paths. Updating only one would create storage-dependent OAuth
  behavior, which is precisely the class of split-brain policy this phase
  prevents.
- JSON data migrations can preserve the schema while still requiring an
  explicit transition for semantic fields. The migration conditions handle both
  absent fields from old binaries and JSON `null` written by transitional code.

### What was tricky to build

- Client profiles are normalised before their exact grant sets are compared.
  This permits harmless ordering/whitespace differences while rejecting extra
  authority, omitted refresh authority, or device authority on a browser
  profile.

### What warrants a second pair of eyes

- Verify the migration classification rules against every production database
  before rollout; ambiguous records intentionally stop strict startup until
  repaired.
- Confirm whether a future device client should opt into refresh tokens with a
  separate explicit profile rather than silently inheriting it.
- Review CLI/documentation migration communications: `--grant-type` is now
  required and existing operational examples need an explicit browser list.

### What should be done in the future

- Begin Phase 2: introduce the durable `DeviceGrant` state model and named
  memory/SQLite operations. Do not expose device endpoints or discovery yet.

### Code review instructions

- Review `pkg/idpstore/validate.go` first: empty grant lists have no authority.
- Read migration `008` together with `TestClientGrantCapabilityMigrationBackfillsKnownLegacyProfiles`.
- Check both Fosite projections in `provider.go` and `sqlstore.go`; neither may
  restore a hard-coded authorization-code/refresh default.
- Invoke `tinyidp admin client create --help` and confirm `--grant-type` is
  documented as required.

### Technical details

```text
admin/bootstrap input
  -> AllowedGrantTypes (sorted, validated, persisted)
  -> Fosite DefaultClient.GrantTypes
  -> later /device_authorization and /token capability checks

legacy SQLite client JSON
  redirect URI          -> [authorization_code, refresh_token]
  public + PKCE/no URI  -> [device_code]
  anything ambiguous    -> [] -> strict production validation error
```

## Step 3: Build the durable device-grant state machine

This step implements the state substrate that later public endpoints and the
Fosite extension will use. It deliberately contains no raw-code generation,
HTTP parsing, rendering, discovery, or token issuance. The result is a small
durable protocol machine with no caller-exposed mutable-record API.

### What I did

- Added `DeviceGrant`, four stored statuses, typed poll outcomes, and typed
  poll/decision/consume requests to `pkg/idpstore`.
- Added `ValidateForCreate`; only complete pending records can enter storage.
  Named operations own every subsequent status transition.
- Extended `Store`, `ReadStore`, and transaction-scoped `TxStore` with
  creation, code lookup, client-bound inspection, polling, decision, and
  consumption operations.
- Implemented both in-memory and SQLite versions. The memory implementation
  clones all secret-hash and slice fields across transaction snapshots; SQLite
  migration `009` stores fixed hash lookup columns, client binding, status,
  expiry, next-poll time, a JSON payload, check constraints, and maintenance
  indexes.
- Implemented durable poll discipline: a permitted pending poll advances the
  next poll time; an early pending or approved poll adds five seconds,
  increments the slowdown count, and stores the new interval.
- Implemented exactly-once decisions and consumption. SQLite uses conditional
  `UPDATE` predicates for pending/approved status and unexpired time, while a
  transaction callback lets later Fosite persistence share one commit.
- Added generic store-suite tests for client binding, cancellation, polling,
  approval, rollback, replay, and expiry. Added SQLite restart plus concurrent
  consume and decision-race tests. Added maintenance and online backup/restore
  tests for device records.

### Why

- A device code is a bearer credential. A generic update method would let an
  endpoint accidentally bypass client binding, expiry, polling, or one-time
  consumption. Named operations put those predicates beside their durable data.
- The device poll interval must survive a process restart. A memory-only rate
  limiter cannot enforce RFC 8628 slowdown semantics across failures.
- Token issuance later needs a single database transaction. Making consume
  available on `TxStore` is the prerequisite for all-or-nothing device-code
  consumption plus Fosite token persistence.

### What worked

- `go test ./pkg/idpstore ./internal/store/memory ./pkg/sqlitestore` passed,
  including the cross-store state-machine suite.
- The SQLite restart test proves approval context survives reopening the DB;
  its 16 concurrent consumers and 16 concurrent decision attempts each have
  exactly one winner.
- Backup manifests now count `device_grants`, and restore reopens the copy and
  retrieves the pending grant by its device-code hash.

### What didn't work

- A repository-wide test run caught two Phase 1 probe files that used the new
  `idpstore` grant constants without importing the package. Adding the missing
  imports made both probes compile and their focused package tests pass.
- The same broad run continued to report the unrelated existing xapp CSP/header
  expectations. No Phase 2 source is in that application; it remains an
  external test-health issue rather than a device-state-machine defect.

### What I learned

- The existing store architecture already supports the required atomic shape:
  `Update` passes a `TxStore` to a callback. The device API can therefore avoid
  a special transaction abstraction while still sharing SQL state with Fosite.
- Deriving expiry from `ExpiresAt` avoids a background-job race. At the exact
  expiry instant, decide and consume reject before status predicates run;
  maintenance later removes the retained terminal record.

### What was tricky to build

- Transaction snapshots in the memory store need deep copies for hashes,
  slices, and optional timestamps. Shallow map copies would allow a caller to
  mutate a snapshot and corrupt committed state even when its transaction
  rolls back.
- SQLite has both JSON payload state and searchable constrained columns. Each
  transition updates the payload while predicates use the columns, which keeps
  lookup efficient and legal transitions visible in SQL.

### What warrants a second pair of eyes

- Review the exact `expires_at > now` semantics and fixed five-second slowdown
  increment against the RFC before it becomes compatibility behavior.
- Review whether denied-grant maintenance should retain by decision time or
  solely by expiry; the current conservative policy removes only after the
  configured retention criterion is met.
- Check the transaction boundary again when Phase 5 adds Fosite writes; a
  standalone `ConsumeDeviceGrant` call must never be used outside that shared
  callback for the final token path.

### What should be done in the future

- Begin Phase 3: domain-separated code hashing and generators, then bounded
  device-authorization request parsing and creation. The store must receive
  hashes only; raw code handling belongs at that endpoint boundary.

### Code review instructions

- Start with `DeviceGrantStore` in `pkg/idpstore/interfaces.go`; confirm there
  is no `UpdateDeviceGrant` escape hatch.
- Review `009_device_grants.sql` and the three SQLite transition predicates in
  `pkg/sqlitestore/store.go` together.
- Run `go test ./pkg/sqlitestore -run DeviceGrant -count=1` to exercise restart
  and concurrent one-winner behavior.
- Read the generic `RunStoreSuite` device tests to compare memory and SQLite
  observable behavior.

### Technical details

```text
create -> pending
pending + permitted poll -> pending; next_poll += interval
pending|approved + early poll -> same status; interval += 5s
pending + approve/deny -> approved|denied
approved + consume (same tx as token persistence later) -> consumed
any active status at expires_at <= now -> derived expired outcome
```

## Step 4: Add the bounded device-authorization creation boundary

The durable store accepts hashes, never raw user-facing credentials. This step
therefore introduces the only creation boundary that handles raw device and
user codes. It does not advertise discovery metadata yet, and verification and
token issuance are still separate phases.

### What I did

- Added `POST /device_authorization` to the strict provider route set.
- Enforced POST-only, form media type, a 16 KiB body cap before parsing, and
  duplicate rejection for `client_id` and `scope`.
- Added cryptographic 32-byte URL-safe device codes and canonical 8-character
  ambiguity-safe user codes (`ABCD-EFGH`).
- Added domain-separated HMAC inputs `tinyidp/device-code/v1\x00` and
  `tinyidp/user-code/v1\x00`. Hash helpers receive only the provider secret
  and raw code; store records receive their output only.
- Added client resolution: public clients use their registered form client ID;
  confidential clients must use HTTP Basic authentication and bcrypt
  verification. Conflicting Basic and form identities fail closed.
- Enforced `GrantDeviceCode`, enabled-client status, `openid`, and the existing
  exact allowed-scope policy before generating any durable grant.
- Added address and client creation-rate-limit keys, bounded five-attempt
  collision retry, secret-free audit events, and no-store JSON responses with
  `device_code`, `user_code`, verification URI, complete URI, expiry, and
  interval.
- Added focused tests for malformed input, unauthorized capability, scope
  rejection, public and confidential clients, raw-code non-persistence/audit,
  collision retry, user-code normalization, and hash-domain separation.

### Why

- Device codes are bearer credentials and user codes are verification secrets.
  Treating either as ordinary request logging or persistence material would
  turn storage, audit, and metrics into credential disclosure surfaces.
- Form parsing and client identity must occur before code generation. Otherwise
  a malformed request can allocate durable state or induce an oracle.
- RFC 8628 creation has different client authentication needs from browser
  authorization: a public device can identify itself, while a confidential one
  must prove its client secret.

### What worked

- `go test ./internal/fositeadapter -run 'TestDevice' -count=1` passed after
  both public and confidential client paths were covered.
- `go test ./internal/fositeadapter ./pkg/idpstore ./internal/store/memory
  ./pkg/sqlitestore` passed, confirming existing strict authorization-code and
  refresh flows remain intact.

### What didn't work

- The first focused build used `idp.ParseScopes`; scope parsing belongs to
  `idpstore.ParseScopes`. Replacing the package reference fixed the compile
  failure without changing endpoint behavior.

### What I learned

- The strict provider already has correctly scoped security headers and a
  reusable `writeJSON` path. The endpoint only needed to add cache controls and
  keep its own error body protocol-compatible.
- The public client model is now enough to decide whether Basic is mandatory:
  `Public` plus explicit device capability selects registered-public identity;
  a non-public device-capable client selects bcrypt-backed Basic verification.

### What was tricky to build

- Collision safety applies to both hashes. A collision on either unique SQLite
  column returns the same storage duplicate signal, so the endpoint regenerates
  the entire device/user-code pair rather than attempting partial reuse.
- The verification URI complete value must include the human code for user
  convenience while the persisted grant must still contain only its HMAC.

### What warrants a second pair of eyes

- Review whether the fixed 16 KiB request limit and five collision attempts
  match deployment requirements before release.
- Confirm external integrators expect strict `openid` for this OIDC-focused
  device endpoint; a pure OAuth resource-only profile would need a separately
  designed policy.
- Confirm audit retention and downstream sinks never serialize raw HTTP bodies
  for this route; the provider event fields themselves are secret-free.

### What should be done in the future

- Begin Phase 4: add a typed verification renderer and browser UI bound to the
  stored user-code hash, authenticated session, CSRF token, and explicit
  approve/deny decision.

### Code review instructions

- Read `deviceAuthorization` from method/media checks through `CreateDeviceGrant`.
- Verify no raw code appears in `DeviceGrant`, audit fields, rate-limit keys,
  or store parameters.
- Run the focused `TestDeviceAuthorizationRetriesHashCollisionsWithinBound` and
  `TestDeviceAuthorizationAuthenticatesConfidentialDeviceClients` tests.
- Confirm discovery intentionally remains unchanged until Phases 4–7 pass.

### Technical details

```text
bounded form + authenticated/identified client + allowed openid scope
  -> random (device_code, user_code)
  -> HMAC(domain || raw code)
  -> CreateDeviceGrant(pending, hashes only)
  -> audit(event/reason only) + no-store RFC 8628 JSON
```

## Step 5: Build the browser verification continuation and decision boundary

This step turns the durable pending grant into a browser interaction without
turning the browser into an authority. The browser may enter a public user
code, but it receives neither a device code nor a database key. It can only
submit a provider-generated opaque interaction handle that is independently
bound to the CSRF cookie and then consumed together with the irreversible
grant decision.

The implementation deliberately requires fresh password authentication for
both approve and deny. It does not treat an ambient `tinyidp_session` as proof
for this RFC 8628 decision and it does not create a browser session as a side
effect. This avoids repeating the forced-reauthentication/session-reuse class
of error already identified in the main browser authorization flow.

### Prompt Context

`continue` — same short continuation prompt as the immediately preceding
implementation turn. The active ticket objective remained to implement the
production device-authorization phases incrementally with a detailed diary,
tests, and commits.

### What I did

- Added a public `idpui.DeviceVerificationRenderer` interface and a typed
  `DeviceVerificationPage` contract. Its three exclusive states are code entry,
  authenticated decision, and terminal notice.
- Added a dependency-free default HTML renderer at
  `pkg/idpui/templates/device_verification.html`. The entry form is GET-only;
  the confirmation form is POST-only and carries the opaque interaction and
  CSRF fields. Labels, autocomplete values, explicit headings, status/error
  roles, and a `formnovalidate` deny button are part of the template contract.
- Exposed the new renderer through `fositeadapter.Options` and
  `embeddedidp.UIConfig`. If a host provides one default renderer implementing
  both interfaces, the provider reuses it; otherwise it constructs the safe
  built-in device renderer.
- Added `GET|POST /device`. GET normalizes the user code, rate-limits code
  entry, and renders a generic error for malformed, unknown, expired, denied,
  consumed, or otherwise unavailable codes. For a pending grant it creates a
  durable `InteractionRecord` with an independently domain-separated opaque
  handle hash, the existing browser-binding hash, the client generation hash,
  and `DeviceUserCodeHash` only.
- Extended `InteractionRecord` and the memory clone path with
  `DeviceUserCodeHash`. SQLite already stores the record as an encoded payload,
  so the existing interaction migration, transaction, backup, and maintenance
  machinery preserve the new hash field without a raw-code column or a second
  mutable state table.
- Made POST reject oversized/duplicated forms, rate-limit decisions, validate
  CSRF and browser binding, re-check the pending grant and current client
  policy, authenticate the entered credentials, and atomically consume the
  interaction plus decide the grant through `Store.Update`.
- Added focused renderer tests (page shapes, HTML escaping, invalid-contract
  rejection, cancellation) and endpoint tests for approval, denial, retry
  after bad credentials, cross-browser rejection, replay, equal generic error
  pages, concurrent decisions, renderer failure, no browser-session creation,
  and audit records.
- Tightened `clientGenerationHash` to include `AllowedGrantTypes`, so removal
  of the device capability invalidates a pending browser continuation rather
  than allowing it to complete against a materially changed client policy.

### Why

- A raw `user_code` is intentionally human-entered and observable in a device
  screen/complete URI; it is not a durable capability to authorize tokens. The
  server must translate it once into a keyed hash and then continue with an
  opaque, browser-bound interaction identifier.
- Reusing the durable interaction primitive retains existing one-time,
  expiry, browser-binding, maintenance, and transaction semantics. A new
  generic verification-session subsystem would duplicate state transitions and
  risk divergent expiry or replay behavior.
- The decision transaction consumes the interaction first and decides the
  grant second, but the `Update` callback rolls both changes back if either
  operation fails. Thus an attacker cannot consume a UI continuation without
  producing a legal decision, nor can two submissions produce two decisions.
- A generic unavailable-code response avoids exposing whether a guessed code
  is unknown, expired, previously denied, already approved, or consumed.

### What worked

- `go test ./pkg/idpui ./internal/fositeadapter ./pkg/embeddedidp
  ./internal/store/memory ./pkg/sqlitestore` passed after the Phase 4 changes.
- `go test ./internal/fositeadapter -run
  'TestDevice(Authorization|Verification|Code)' -count=1` passed the focused
  endpoint suite, including the concurrent one-winner decision test.
- The renderer test parses generated HTML and confirms the entry form uses GET,
  the decision form uses POST, the hidden values are present only on the
  decision page, labels exist for credentials and code entry, and untrusted
  client/login/scope/error values cannot create active HTML nodes or handlers.
- The success test demonstrates that an approved grant stores subject,
  `auth_time`, AMR, and requested/approved scopes while the response body never
  contains the device bearer code and no browser session cookie is issued.

### What didn't work

- The first focused compile after adding the handler failed because the new
  file still imported `time` after the implementation stopped using it. The
  exact compiler error was `"time" imported and not used`. Removing that
  unused import restored the build.
- The initial endpoint-test compilation omitted imports for `idpui`,
  `idpaccounts`, `io`, and `net/http/cookiejar`. The compiler reported each as
  undefined; importing the test dependencies fixed the test harness without
  changing production code.
- While wiring the embedded option I briefly wrote the nonexistent field name
  `opts.Cookie.CSRFCookieName`. The real public field is `opts.Cookie.CSRFName`;
  the typo was corrected before any test or commit.

### What I learned

- The existing `InteractionRecord` is already the correct durable shape for a
  device verification continuation: it is keyed by a hash, has explicit expiry
  and terminal outcome, records a browser binding, participates in `TxStore`,
  and is included in the established maintenance/backup path. Adding one
  explicitly documented hash field is less risky than another store interface.
- CSRF validation alone is insufficient if an interaction can be copied to a
  browser profile with a matching form token. Storing and comparing the hash of
  the HttpOnly CSRF cookie binds the interaction to the issuing profile.
- Browser convenience does not require creating a session. A device page can
  authenticate a password solely to produce a durable device-grant decision;
  that choice reduces session lifecycle coupling and makes fresh-auth behavior
  obvious in review.

### What was tricky to build

- The GET entry page cannot need an interaction handle yet, because it has no
  valid code to bind. Making it a no-state GET avoids a pre-code CSRF/session
  record. Only the valid-code path emits a cookie, CSRF MAC, and durable
  continuation.
- Error rerenders after a wrong password must preserve the original opaque
  handle and valid CSRF MAC without consuming the record. The code therefore
  reuses the submitted MAC only after it has validated the cookie and loaded
  the server-owned record.
- Client policy changes must invalidate the continuation. The grant itself
  captures the original scopes, but the record also captures a generation hash
  that now includes allowed grant types; the POST re-loads and re-validates the
  enabled client before authenticating and deciding.

### What warrants a second pair of eyes

- Confirm the product requirement that even **deny** requires password entry.
  It is the conservative interpretation of authenticated explicit decision,
  but a future UX policy might use an existing recently authenticated session
  with a separate freshness proof. That would need its own invariant design;
  it must not silently accept a stale session.
- Review the wording of generic invalid-code messages with UX and support.
  The equal-body test intentionally protects against a status oracle, so more
  detailed messages must not distinguish unknown from terminal codes.
- Review host custom-renderer integration. A custom OAuth interaction renderer
  that does not implement `DeviceVerificationRenderer` gets the safe default
  page; a product that requires a fully unified visual design should explicitly
  supply both bounded renderer interfaces.

### What should be done in the future

- Begin Phase 5: implement the Fosite token endpoint support for
  `urn:ietf:params:oauth:grant-type:device_code`. It must map every durable
  polling/terminal result to RFC 8628 errors and consume an approved grant in
  the exact same SQL transaction as Fosite token persistence.
- Do not advertise `device_authorization_endpoint` or device grant support in
  discovery until Phase 5 token exchange and its failure-path tests are
  complete.
- Add the remaining Phase 6 durable rate-limit and operator-readiness work;
  today the endpoint invokes the configured limiter, but a process-local
  limiter alone is not a restart-safe production abuse-control guarantee.

### Code review instructions

- Start at `internal/fositeadapter/device_verification.go`. Trace GET from
  normalization through `createDeviceVerificationInteraction`, then trace POST
  from `validateCSRF` through the `Store.Update` callback. Confirm no raw
  user/device code reaches `InteractionRecord`, audit events, or rendering.
- Compare `deviceVerificationHandleHash` with existing interaction and device
  code hash domains. The prefix must remain unique and versioned.
- Read `DeviceVerificationPage.Validate` and the default template together.
  The renderer contract must reject malformed host page models before output;
  renderers must not receive response writers, cookies, or redirect authority.
- Run `go test ./internal/fositeadapter -run TestDeviceVerification -count=1`
  and inspect the one-winner race plus renderer-failure assertions.
- Review the added `AllowedGrantTypes` input to `clientGenerationHash`; it is
  security-relevant policy material, not incidental client metadata.

### Technical details

```text
GET /device?user_code=ABCD-EFGH
  normalize + HMAC(user-code domain)
  -> pending, unexpired DeviceGrant?
  -> random opaque handle + CSRF(cookie, handle)
  -> InteractionRecord{
       IDHash=HMAC(device-verification domain, handle),
       DeviceUserCodeHash=HMAC(user-code domain, normalized code),
       BrowserBindingHash=HMAC(csrf-cookie),
       ClientID, GenerationHash, expires=min(interaction TTL, grant TTL)
     }
  -> typed confirmation page

POST /device
  bounded form + rate limit + CSRF + interaction/binding/client validation
  -> fresh AuthenticatePassword
  -> Store.Update:
       require pending and current grant/client
       ConsumeInteraction(approved|denied)
       DecideDeviceGrant(approved identity/scopes | denied)
     // any error rolls back both state changes
  -> secret-free audit + terminal no-store page
```
