---
Title: Implementation diary
Ticket: TINYIDP-INTROSPECTION-001
Status: active
Topics:
    - auth
    - oidc
    - security
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/device_token_handler.go
      Note: Transfers approved device audiences into Fosite token request (d5c7647)
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Introspection endpoint, exact audience policy, and device resource indicator handling (f718d36, d5c7647)
        Preauthenticated caller classification before opaque-token validation (d196aeb)
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: |-
        Authorization-code, device, caller-authentication, wrong-audience, duplicate-credential, and refresh-audience regression evidence (f718d36, d5c7647, 866e0bb, d196aeb)
        Incorrect resource-client credential regression (d196aeb)
        Memory-backed caller, audit, rate-limit, expiry, and rotation matrix (df6b2be, d15f0f3, e4358ed)
    - Path: repo://internal/fositeadapter/sqlstore_test.go
      Note: |-
        Restart-backed SQLite opaque-token lifecycle, audience, rotation, reuse, and inactive-response regression evidence (d196aeb)
        SQLite lifecycle and inactive-response evidence (d196aeb)
        SQLite expiry and durable token-family lifecycle matrix (df6b2be)
    - Path: repo://pkg/idpstore/types.go
      Note: Durable resource-indicator fields and client capability model (f718d36, d5c7647)
    - Path: repo://pkg/sqlitestore/store.go
      Note: SQLite durable approval transition for device audiences (d5c7647)
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://www.rfc-editor.org/rfc/rfc9449.html
Summary: Chronological evidence for authenticated opaque-token introspection and resource-indicator implementation.
LastUpdated: 2026-07-16T13:05:00-04:00
WhatFor: Preserve implementation decisions, verification evidence, and remaining production work.
WhenToUse: Read before extending the endpoint, its resource-server contract, or xgoja integration.
---





# Diary

## Goal

Record how tiny-idp gained an authenticated, audience-constrained RFC 7662
introspection endpoint and how resource indicators were carried through both
authorization-code and device authorization lifecycles. The diary distinguishes
implemented, tested behavior from planned lifecycle and deployment work.

## Step 1: Establish the resource-server contract and provider core

The initial implementation kept tiny-idp's opaque access-token architecture.
Rather than teaching each application to inspect provider storage or misuse
UserInfo as an authorization API, it added a provider-owned RFC 7662 boundary.
The boundary authenticates a confidential resource server, asks Fosite to
validate the opaque token, and exposes only product-approved metadata.

This step also made resource authorization an explicit client capability. A
client may request only registered audiences; an introspection caller must be a
confidential, enabled client with the separate `CanIntrospect` capability. A
valid token is therefore insufficient by itself: the token must also have a
granted audience shared with the calling API.

### Prompt Context

**User prompt (verbatim):** "Create a detailed guide for implementation and the requisite tasks. then implement it, keeping a dtailed diaryas you work, committing at appropriate intervals."

**Assistant interpretation:** Produce the implementation design and task plan, then implement the production-safe core with traceable evidence and focused commits.

**Inferred user intent:** Build an interoperable tiny-idp/xgoja authorization path without weakening opaque-token or resource-separation invariants.

**Commit (code):** `f718d36` — "feat: add authenticated token introspection"

### What I did

- Created the ticket design guide and phase/task plan.
- Added `AllowedAudiences` and `CanIntrospect` to `idpstore.Client`, including
  production validation and sanitized admin output.
- Passed allowed audiences into Fosite clients in both memory and SQLite-backed
  Fosite stores, so issued and refreshed requester records retain the
  authorization decision.
- Mounted `POST /introspect` and published it through OIDC discovery with only
  `client_secret_basic` advertised.
- Implemented a constrained active response: `iss`, `sub`, `client_id`,
  `scope`, `aud`, `exp`, `iat`, and `token_type`. Inactive outcomes use exactly
  `{"active":false}` after successful caller authentication.
- Added authorization-code integration coverage for the permitted inbox API
  and a different API that receives only the inactive response.

### Why

RFC 7662 defines an authorization-server decision endpoint, while RFC 8707
defines resource indicators that prevent one access token from being silently
reused at unrelated APIs. The existing UserInfo endpoint does neither job: it
is an identity-claims endpoint and lacks resource-server authentication and
audience enforcement.

### What worked

- `go test ./pkg/idpstore ./internal/admin ./internal/cmds ./internal/oidcmeta ./internal/fositeadapter -count=1` passed after the implementation.
- The authorization-code test proved both sides of the policy: a token for
  `https://inbox.example.test/api` was active for `inbox-api` and inactive for
  `other-api`.
- Fosite's `NewIntrospectionRequest` was the correct shared primitive: it
  validates the opaque token against the issuer's existing persistence and
  strategies rather than duplicating token parsing logic.

### What didn't work

- The first focused test build failed because `pkg/idpstore/validate.go` used
  `fmt.Errorf` without importing `fmt`. The exact command was
  `go test ./pkg/idpstore ./internal/admin ./internal/cmds ./internal/oidcmeta ./internal/fositeadapter -count=1`; the compiler reported `undefined: fmt`.
  Adding the missing standard-library import fixed the build and the rerun
  passed.

### What I learned

- Fosite persists both requested and granted audiences in its requester
  representation, including the SQLite adapter, so token refresh already has
  the necessary continuity once the original authorization grant is correct.
- The resource caller's client record and the token's issuing client are
  intentionally different identities. The former proves the API's privilege;
  the latter is returned as `client_id` for authorization policy and audit.

### What was tricky to build

The token state should not become an oracle. An unknown, revoked, expired,
wrong-audience, or malformed token must not give the authenticated resource
server distinguishable reasons. The implementation therefore returns only
`active:false` for all such token outcomes, but uses HTTP 401 for a caller that
has not authenticated as an authorized resource server. This keeps provider
operations observable through redacted audit events without disclosing token
state over the protocol.

### What warrants a second pair of eyes

- Confirm that the chosen disclosure fields are sufficient for every intended
  xgoja resource policy without adding profile claims casually.
- Review whether the administrative capability should be called
  `CanIntrospect` or a more role-oriented public name before stabilizing an
  external management API.
- Verify that production deployment supplies client secrets to the active
  provider process in a rotation-safe way; the SQL record stores only BCrypt
  hashes, while Fosite's active confidential-client verifier needs the
  configured plaintext secret input at startup.

### What should be done in the future

- Add explicit revoked, expired, malformed, and refresh-rotation lifecycle
  cases for both memory and SQLite.
- Add a strict HTTPS deployment smoke and the xgoja resource-server consumer.

### Code review instructions

- Start with `internal/fositeadapter/provider.go`, especially `introspect`,
  `buildFositeStore`, and the discovery route registration.
- Review `pkg/idpstore/validate.go` before accepting the client-capability
  policy.
- Validate with `go test ./internal/fositeadapter ./internal/oidcmeta ./pkg/idpstore -count=1`.

### Technical details

```text
authorized resource server + opaque access token
        |
        v
POST /introspect (Basic client credentials)
        |
        +-- Fosite validation fails ------------> 200 {"active":false}
        |
        +-- no shared granted audience ----------> 200 {"active":false}
        |
        +-- valid bearer access token -----------> constrained active response
```

The governing sources are RFC 7662 sections 2.1 and 2.2, RFC 8707 resource
indicator semantics, and RFC 9449's warning that DPoP sender constraints need
resource-server-side proof verification.

## Step 2: Carry resource indicators through device authorization and harden callers

Device authorization was initially a gap: its durable grant remembered scopes
but not audiences. That would let a device client complete a valid OAuth flow
yet produce a token that no protected API could safely accept. This step gave
device grants requested and approved audience fields, copying them only at
fresh browser approval and granting them only at device-code redemption.

The step also found that Fosite's default audience strategy accepts URI-path
descendants. Tiny-idp's new domain model deliberately uses exact registered
resource identifiers. The provider now declares Fosite's exact strategy too,
so authorization-code and device flows use one auditable policy.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Continue the implementation rather than stopping after the initial endpoint, and validate the same security contract across alternate OAuth grants.

**Inferred user intent:** Ensure that the production device/CLI path is not a weaker exception to resource-server authorization.

**Commit (code):** `d5c7647` — "auth: bind device tokens to registered audiences"

### What I did

- Added `RequestedAudiences` and `ApprovedAudiences` to `DeviceGrant`, and
  `ApprovedAudiences` to the named device decision request.
- Updated memory cloning/transactional transitions and SQLite JSON persistence
  transitions so audiences survive polling, approval, restart, and one-use
  consumption.
- Parsed the RFC 8707-compatible repeated or space-separated `audience` form
  parameter at `/device_authorization`, rejected unregistered values with
  `invalid_target`, and stored only validated values.
- Copied requested audiences into the durable approval only after fresh
  password authentication, then set and granted Fosite audiences at token
  redemption.
- Added end-to-end coverage from device authorization through browser approval,
  token issuance, and authenticated introspection.
- Hardened `/introspect` to reject duplicate `Authorization` headers and rate
  limit by both normalized caller address and resource-client ID.

### Why

Resource indicators are part of the authorization grant, not a cosmetic token
attribute. A device code is an asynchronous authorization request and needs the
same request/approval/issuance provenance as an authorization-code request.
Likewise, allowing a proxy-dependent "first Authorization header wins" rule is
unnecessarily ambiguous for a credential-bearing endpoint.

### What worked

- `go test ./internal/fositeadapter ./internal/store/memory ./pkg/sqlitestore ./pkg/idpstore -count=1` passed after durable audience propagation.
- `go test ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestDeviceTokenExchangeIssuesOIDCTokensConsumesOnceAndSupportsUserInfo' -count=1` passed after caller hardening.
- The device regression verifies that an inbox audience is stored pending,
  copied on approval, granted into the Fosite requester, and visible to the
  correctly registered inbox API through introspection.
- Store-suite assertions exercise both memory and SQLite implementations,
  preventing shallow-copy bugs in in-memory transactions and JSON persistence
  omissions in SQLite.

### What didn't work

- Defuddle could not extract the RFC Editor HTML and returned:
  `Error: No content could be extracted from https://www.rfc-editor.org/rfc/rfc7662.html`.
  The ticket therefore preserves the RFC Editor's canonical `.txt` editions
  in `sources/` instead. This is a source-extraction limitation, not an
  implementation failure.

### What I learned

- `fosite.GetAudiences` supports the RFC ecosystem's two common encodings:
  repeated `audience` parameters, or one space-delimited parameter. It also
  removes empty values, so tiny-idp validates the resulting canonical set.
- Fosite's default audience strategy is deliberately URI-prefix aware. That is
  useful for some deployments, but it would conflict with `Client.AllowsAudience`
  exact matching. Explicit configuration is necessary; relying on a library
  default would make code review misleading.

### What was tricky to build

The device flow has two state machines. The project-owned `DeviceGrant` first
controls human approval and one-time consumption; after redemption, Fosite owns
the opaque access-token requester. The audience has to cross that boundary only
after approval. Setting it before the decision would violate the durable grant
model; setting it after Fosite persistence would create a token whose
introspection metadata did not match its authorization record.

### What warrants a second pair of eyes

- Confirm that exact URI matching is the intended product contract for all
  future APIs. Relaxing it later must be a documented semantic change, not a
  per-client exception.
- Review the rate-limit capacity/window configuration used by production hosts;
  this change supplies stable keys, not a universal quota.
- Verify reverse-proxy configuration for `ClientAddressResolver`; an incorrect
  trusted-proxy list can make address buckets either ineffective or unfair.

### What should be done in the future

- Add explicit test coverage for revocation, expiration, and refresh rotation
  at the introspection boundary.
- Add an HTTPS smoke against `serve-production` with real configured secrets.
- Add DPoP only as a coordinated provider-and-resource-server release with
  durable replay prevention; it remains explicitly unsupported here.

### Code review instructions

- Start at `internal/fositeadapter/provider.go:deviceAuthorization` and follow
  `DeviceGrant` through `completeDeviceVerification` and
  `deviceTokenHandler.HandleTokenEndpointRequest`.
- Review memory cloning and SQLite decision persistence before trusting the
  lifecycle guarantee.
- Validate with the two commands in **What worked**, then run `go test ./... -count=1` before release.

### Technical details

```text
POST /device_authorization audience=https://inbox.example.test/api
    -> DeviceGrant.RequestedAudiences
browser password approval
    -> DeviceGrant.ApprovedAudiences
POST /token grant_type=device_code
    -> Fosite Request.GrantedAudience
POST /introspect as inbox-api
    -> active=true only when audiences intersect
```

## Step 3: Prove refresh preserves the original resource decision

The initial implementation relied on Fosite's requester persistence to carry
the original granted audience into refresh-token exchange. That is an
appropriate implementation mechanism, but a production invariant needs a
regression test at the public boundary. This step added that test rather than
marking the task complete from code inspection alone.

The same authorization-code flow now issues a token for the inbox resource,
rotates the refresh token, and introspects the new access token as `inbox-api`.
The test requires an active response containing the exact inbox audience.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the concrete implementation evidence required by the task plan, including alternative token lifecycles.

**Inferred user intent:** Avoid a security design that is correct only for the first access token and silently changes during refresh.

**Commit (code):** `866e0bb` — "test: preserve audience across refresh introspection"

### What I did

- Extended `TestStrictAuthorizationCodeFlow` to introspect the access token
  returned by refresh-token rotation.
- Required HTTP 200, `active:true`, and the exact
  `https://inbox.example.test/api` audience.

### Why

Fosite's refresh handler copies requested and granted audiences from the
original requester. The provider must preserve that behavior because otherwise
a resource server could accept a refreshed token without a corresponding
resource grant, or applications could unexpectedly lose API access after a
normal refresh.

### What worked

- `go test ./internal/fositeadapter -run TestStrictAuthorizationCodeFlow -count=1` passed.
- The test starts at browser authorization and ends at the external
  introspection contract, covering the same boundary an xgoja resource server
  will use.

### What didn't work

- N/A. The focused test passed on its first run.

### What I learned

- The requester preservation already implemented by Fosite is observable and
  stable through tiny-idp's constrained response; no separate audience cache
  or refresh adapter is necessary.

### What was tricky to build

The refresh response does not itself expose audience. Testing only its HTTP 200
status would miss a persistence regression. The correct observation point is a
second authenticated introspection request, which also exercises the provider's
resource-audience intersection check.

### What warrants a second pair of eyes

- Review future changes to Fosite requester sanitization and SQL
  `persistRequester`/`restoreRequester`; this test detects a behavior change
  but not every possible data-minimization regression.

### What should be done in the future

- Add the same post-refresh assertion to a restart-backed SQLite fixture.
- Add revoked and expired access-token assertions before claiming full Phase C
  lifecycle coverage.

### Code review instructions

- Read the refresh section of `TestStrictAuthorizationCodeFlow` after the
  first access-token introspection assertion.
- Validate with `go test ./internal/fositeadapter -run TestStrictAuthorizationCodeFlow -count=1`.

### Technical details

```text
initial authorized request: granted_aud = [Inbox API]
refresh-token rotation:     granted_aud copied by Fosite
new access token:           active for Inbox API only
```

## Step 4: Make inactive opaque-token results unambiguous and prove SQLite token-family transitions

This step turned the remaining lifecycle claim into an end-to-end SQLite
regression. It starts from a resource-indicated authorization-code grant,
performs caller-authenticated introspection, rotates the refresh token, and
then deliberately reuses the old refresh token. The observable contract is
strict: an unknown token, an unparseable token, an old access token after a
normal rotation, and a rotated access token after refresh-token reuse are all
the same HTTP success response containing exactly `{"active":false}`.

The first version of this test uncovered a real endpoint classification defect.
Fosite can include `ErrRequestUnauthorized` in the error chain for malformed or
unknown opaque subject tokens. Tiny-idp had treated any error matching that
sentinel as a failed resource-client credential, returning `401 invalid_client`
despite the caller having already authenticated successfully. That makes the
protocol semantically wrong and creates a needless distinction between token
classes. The fix authenticates the registered resource client explicitly,
using its persisted BCrypt hash, before Fosite is asked to validate the subject
token. After that check succeeds, every Fosite token-validation error is an
inactive token at the external boundary.

### Prompt Context

**User prompt (verbatim):** "ok, continue"

**Assistant interpretation:** Continue the planned lifecycle/confidentiality
matrix rather than declaring the initial endpoint implementation complete.

**Inferred user intent:** Turn the designed RFC 7662 safety properties into
durable regression evidence, especially across real SQLite persistence and
refresh-family revocation.

**Commit (code):** `d196aeb` — "auth: harden opaque token introspection lifecycle"

### What I did

- Extended the SQLite token fixture with a public SPA that may request the
  inbox audience and a confidential `inbox-api` resource server that has a
  BCrypt secret hash, `CanIntrospect`, and that same allowed audience.
- Added a complete SQLite lifecycle test covering active introspection,
  unknown token, malformed token, refresh rotation, old access-token
  invalidation, refreshed access-token activation, refresh-token reuse, and
  token-family revocation after reuse.
- Compared the unknown and malformed responses byte-for-byte with
  `crypto/subtle.ConstantTimeCompare` and required the exact canonical body
  `{"active":false}\n`.
- Added a wrong-resource-secret assertion to the strict endpoint integration
  test: it must receive `401` and a `WWW-Authenticate` Basic challenge,
  distinguishing caller authentication from subject-token state.
- Made `/introspect` normalize Basic credentials with the same form-decoding
  convention Fosite uses, look up the resource client, and validate the
  supplied secret with `bcrypt.CompareHashAndPassword` before invoking
  `NewIntrospectionRequest`.
- Changed post-authentication Fosite failures to record an `inactive_token`
  audit event and return the canonical inactive response. No token value,
  digest, or Fosite error is included in the audit event or HTTP response.

### Why

RFC 7662 specifies that an authorization server returns `active:false` when a
token is invalid, expired, revoked, malformed, or otherwise unusable. The
resource server needs a clear distinction between *its own credential is
unacceptable* and *the presented bearer token is not usable*. It does not need,
and should not receive, a taxonomy of why the bearer token failed. Explicit
caller authentication establishes that distinction before token processing;
the constrained inactive response preserves it afterward.

Refresh reuse is especially important because it is a security transition, not
merely an exchange failure. The existing Fosite configuration revokes the
token family on detected reuse. The test proves that a previously valid
refreshed access token becomes unusable through the API a relying application
will actually call.

### What worked

- `go test ./internal/fositeadapter -run TestSQLiteIntrospectionLifecyclePreservesAudienceAndHidesInactiveTokens -count=1` passed after the correction.
- `go test ./internal/fositeadapter -run 'TestProviderStrictEndpoints|TestSQLiteIntrospectionLifecyclePreservesAudienceAndHidesInactiveTokens' -count=1` passed, proving valid caller authentication, invalid caller authentication, and opaque-token confidentiality together.
- `go test ./internal/fositeadapter -count=1` passed in 17.538 seconds before the code commit.
- The active half of the SQLite test proves that persisted BCrypt client hashes
  are accepted by the complete Fosite path, not only by a direct store lookup.

### What didn't work

- The initial lifecycle assertion failed with:

  ```text
  inactive responses unknown=(401,"{\"error\":\"invalid_client\"}\n") malformed=(401,"{\"error\":\"invalid_client\"}\n")
  ```

  A valid token had already introspected successfully with the same Basic
  credentials, which isolated the problem to error classification after caller
  authentication rather than fixture configuration or BCrypt persistence.
- The first test invocation inside the workspace sandbox could not compile
  because its existing Go build-cache path was mounted read-only:

  ```text
  open /home/manuel/.cache/go-build/...: read-only file system
  ```

  Re-running the exact focused test with ordinary host-cache access succeeded.
  This was an execution-environment limitation, not a product failure.

### What I learned

- A wrapped library sentinel is not necessarily a safe public error-classifier.
  Fosite's authorization error family is reused below the layer where
  tiny-idp's RFC 7662 response policy needs to make its decision.
- Authentication and token validity are two separate state machines. The
  endpoint must finish the resource-client state machine before interpreting
  errors from the bearer-token state machine.
- The right regression observation is not merely a failed old refresh request:
  it is the post-failure introspection state of a previously valid access
  token, because that verifies family revocation reaches a resource server.

### What was tricky to build

The endpoint still calls Fosite after tiny-idp authenticates the resource
client. That is intentional duplication of the credential verification at two
different abstraction levels. Tiny-idp owns the externally visible response
classification; Fosite owns token persistence and protocol request validation.
The initial pre-check makes the former deterministic, while Fosite's existing
check remains an internal defence in depth layer. The credential copy is
normalized with `url.QueryUnescape` to preserve Fosite's handling of
application/x-www-form-urlencoded Basic values rather than inventing a subtly
different authentication grammar.

### What warrants a second pair of eyes

- Review the claim that a post-preauthentication Fosite error should always be
  represented as inactive. This is correct for token confidentiality, but an
  operator may want a separately redacted health metric for infrastructure
  errors so availability problems do not hide in inactive-token counts.
- Review the client-secret normalization rule against the final supported
  administrative secret syntax. Generated secrets are base64url-safe; user
  supplied secrets containing `+` or percent escapes should be covered by a
  dedicated compatibility test before documenting that syntax.
- Confirm the intended refresh-reuse policy remains token-family revocation
  when Fosite is upgraded; the test protects the observed behavior.

### What should be done in the future

- Add deterministic expiry coverage using the provider clock, rather than
  waiting on wall-clock expiry in a test.
- Add the same inactive-response and wrong-client matrix to the memory store
  so the development path cannot accidentally drift from SQLite behavior.
- Test public, disabled, and capability-less resource clients, address/client
  rate-limit handling, root/path issuer discovery, and audit redaction before
  closing the negative-test task.

### Code review instructions

- Read `Provider.introspect` in `internal/fositeadapter/provider.go` from the
  Basic-header validation through the `NewIntrospectionRequest` error branch.
  Confirm the order is: transport validation, resource caller authorization,
  BCrypt credential verification, rate limiting, Fosite token validation,
  audience validation, response rendering.
- Read `TestSQLiteIntrospectionLifecyclePreservesAudienceAndHidesInactiveTokens`
  in `internal/fositeadapter/sqlstore_test.go` as the executable lifecycle
  table, then `TestProviderStrictEndpoints` for the incorrect-secret case.
- Run `go test ./internal/fositeadapter -count=1`; for a fast targeted review,
  run the two commands listed in **What worked**.

### Technical details

```text
POST /introspect
  Basic inbox-api:secret + token
       |
       +-- malformed / unknown Basic --------------------> 401 invalid_client
       |
       +-- disabled/public/non-introspecting client ------> 401 invalid_client
       |
       +-- valid resource-client credential
              |
              +-- unknown, malformed, expired, revoked access token
              |      -> 200 {"active":false}
              |
              +-- active access token, no shared audience
              |      -> 200 {"active":false}
              |
              +-- active access token, shared audience
                     -> 200 constrained active metadata
```

## Step 5: Complete the executable caller, audit, rate-limit, expiry, and refresh matrix

The previous SQLite test established the hardest persistence and refresh-reuse
transition. This step expanded the matrix so it is not a SQLite-only claim and
so the resource-client boundary itself has deliberate negative evidence. The
memory-backed strict authorization-code flow now exercises both caller
authentication and opaque-token state, while the SQLite fixture retains the
restart-backed token-family-revocation proof.

### Prompt Context

**User prompt (verbatim):** "ok, continue"

**Assistant interpretation:** Continue from the SQLite correction into the
remaining concrete security cases, record the work, and commit at coherent
verification intervals.

**Inferred user intent:** Treat a production authorization endpoint as a
matrix of invariants rather than a single happy-path integration test.

**Commits (code):**

- `df6b2be` — "test: extend introspection security matrix"
- `d15f0f3` — "test: cover memory introspection lifecycle"
- `e4358ed` — "test: prove memory refresh token rotation"

### What I did

- Added a configurable SQLite fixture constructor, then used a synchronized
  provider clock and a one-minute access-token lifetime to prove that an
  otherwise valid token becomes inactive after expiry without wall-clock
  sleeping.
- Extended the memory-backed strict authorization-code test with three
  resource-caller rejections: a public OAuth client, a disabled confidential
  resource client, and a confidential client that lacks `CanIntrospect`.
  Each must receive Basic-auth `401`, before subject-token handling.
- Added the bad-secret case for an otherwise authorized resource client.
- Added unknown and malformed opaque tokens to the memory flow and required
  the same exact `{"active":false}` body for both.
- Verified normal refresh rotation at the public introspection boundary: the
  refreshed token remains active for its audience, while the pre-rotation
  access token becomes inactive. The existing SQLite test additionally proves
  reuse of the old refresh token revokes the token family.
- Injected a test-only rate limiter through the public provider options,
  confirmed it observes both `introspection:address:` and
  `introspection:client:` keys, and forced the resource-client bucket to
  produce `429 temporarily_unavailable`.
- Injected `idp.MemorySink`, required `introspection.accepted`,
  `introspection.inactive`, and `introspection.rejected` events, marshalled
  the resulting event collection, and asserted that access tokens, refresh
  tokens, good secrets, and deliberately wrong secrets are absent.

### Why

The endpoint has two independent attacker-controlled inputs: bearer token
material and resource-client credentials. The matrix ensures each is handled
at its correct boundary. A public or disabled client must not reach token
validation; a valid confidential client with a bad or unauthorized token must
not learn why the token failed; an overloaded authorized client must receive a
retryable availability response rather than an inactive-token result.

Audit tests are part of the protocol design, not merely observability tests.
Opaque access and refresh tokens are bearer credentials. Persisting them in an
audit sink would turn routine diagnostics into a high-value credential store.
The endpoint records stable client IDs, bounded result/reason vocabularies,
and counts, which support operations without recording the presented secret.

### What worked

- `go test ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestSQLiteIntrospectionExpiresAccessTokenAtProviderClock' -count=1` passed after the capability, audit, rate, and SQLite expiry additions.
- `go test -race ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestSQLiteIntrospection(LifecyclePreservesAudienceAndHidesInactiveTokens|ExpiresAccessTokenAtProviderClock)' -count=1` passed for the focused matrix.
- `go test ./internal/fositeadapter -run 'TestStrictAuthorizationCodeFlow|TestSQLiteIntrospection(LifecyclePreservesAudienceAndHidesInactiveTokens|ExpiresAccessTokenAtProviderClock)' -count=1` passed after adding memory unknown/malformed/expiry coverage.
- `go test ./internal/fositeadapter -run TestStrictAuthorizationCodeFlow -count=1` passed after adding the normal memory refresh-rotation assertion.

### What didn't work

- Attempting to reuse the internal-package `recordingLimiter` from the
  external-package integration test failed at compile time:

  ```text
  internal/fositeadapter/provider_test.go:106:14: undefined: recordingLimiter
  ```

  The replacement is a small `strictRecordingLimiter` in the external test
  package. This keeps the test dependent only on public `Options.RateLimiter`
  behavior rather than unexported adapter implementation details.
- A first expiry advance of one minute plus one second was too close to the
  boundary. Fosite stamps expiry at the point of token issuance, a few seconds
  after the fixture's provider-clock snapshot. The test observed an active
  token with `exp` just after the advanced provider time. Advancing ten minutes
  beyond a one-minute TTL removes that scheduling edge while remaining fully
  deterministic and does not sleep.
- A combined full-suite-and-commit shell invocation did not reach staging and
  produced no diagnostic output. I separated full-suite verification and the
  commit afterward; the focused matrix commands above provide the explicit
  recorded evidence for the changed behavior.

### What I learned

- A rate-limit test must prove namespace selection as well as status code.
  Otherwise an implementation can return `429` while accidentally applying a
  global, token-derived, or client-only bucket.
- In a provider that injects a product clock but delegates token issuance to a
  library clock, expiry tests need a generous logical gap. The production
  behavior remains correct because the endpoint explicitly compares the
  session's access-token expiry against `p.now()`.
- Storage-independent invariant testing is valuable even when Fosite owns much
  of the token state. The memory test catches adapter wiring changes; the SQL
  test catches durable requester/transaction/restart changes.

### What was tricky to build

The strict test progresses through browser login, code redemption, UserInfo,
introspection, refresh rotation, and rate limiting. A test limiter that begins
rejecting too early would alter unrelated login and token steps. The limiter is
therefore permissive until the dedicated introspection request, then reset
before the expiry assertion. This keeps each observed response attributable to
one policy dimension.

The final memory flow has two relevant access tokens: the original token, made
inactive by normal rotation, and the refreshed token, made inactive by the
provider-clock expiry assertion. The SQLite flow adds the third transition:
refresh reuse revokes an already refreshed token family. Together they prevent
future refactors from accidentally treating rotation, expiry, and reuse as the
same state transition.

### What warrants a second pair of eyes

- Decide whether production deployments should expose the rate-limit reason
  only through the existing bounded audit event or also through a protected
  metrics counter. The HTTP response intentionally remains generic.
- Review the desired semantics of resource-client changes after provider
  startup for the memory store. The endpoint re-reads the product client record
  for authorization, while Fosite's in-memory client representation was built
  at startup; production SQLite reads dynamically. Admin mutation/rotation
  semantics should be specified before advertising hot reconfiguration.
- Add a dedicated memory-store refresh-reuse assertion if the project wants
  every store backend to prove every token-family transition separately. The
  current SQLite test is the durable source of truth for reuse revocation.

### What should be done in the future

- Add root-issuer and path-issuer discovery assertions for
  `introspection_endpoint`, then complete the remaining negative/discovery
  task.
- Run the documented strict-TLS smoke against a configured
  `serve-production` instance rather than only a local HTTP test server.
- Write the xgoja `oidcresource` consumer contract and example handoff using
  the now-tested resource-server behavior.

### Code review instructions

- Read `TestStrictAuthorizationCodeFlow` from provider construction through
  its final audit check. Its subcases now act as a compact memory-backed
  security matrix.
- Read `TestSQLiteIntrospectionLifecyclePreservesAudienceAndHidesInactiveTokens`
  and `TestSQLiteIntrospectionExpiresAccessTokenAtProviderClock` for durable
  lifecycle and clock behavior.
- Review `strictRecordingLimiter` as test infrastructure: it implements the
  public `idp.RateLimiter` interface and does not require access to provider
  internals.
- Start with the focused commands in **What worked**; run
  `go test ./internal/fositeadapter -count=1` as the package-level gate.

### Technical details

```text
resource caller state
  public | disabled | lacks CanIntrospect | bad BCrypt secret
       -> 401 invalid_client; do not inspect bearer token

authorized resource caller state
  unknown | malformed | wrong audience | rotated | expired | reused-family
       -> 200 {"active":false}
  client/address bucket exhausted
       -> 429 temporarily_unavailable
  granted exact audience + current access token
       -> 200 active metadata

audit event payload
  contains: event name, resource client ID, result/reason vocabulary, counts
  excludes: bearer token, refresh token, client secret, Fosite error detail
```
