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
      Note: Introspection endpoint, exact audience policy, and device resource indicator handling (f718d36, d5c7647)
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: Authorization-code, device, wrong-audience, duplicate-credential, and refresh-audience regression evidence (f718d36, d5c7647, 866e0bb)
    - Path: repo://pkg/idpstore/types.go
      Note: Durable resource-indicator fields and client capability model (f718d36, d5c7647)
    - Path: repo://pkg/sqlitestore/store.go
      Note: SQLite durable approval transition for device audiences (d5c7647)
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://www.rfc-editor.org/rfc/rfc9449.html
Summary: Chronological evidence for authenticated opaque-token introspection and resource-indicator implementation.
LastUpdated: 2026-07-15T22:30:00-04:00
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
