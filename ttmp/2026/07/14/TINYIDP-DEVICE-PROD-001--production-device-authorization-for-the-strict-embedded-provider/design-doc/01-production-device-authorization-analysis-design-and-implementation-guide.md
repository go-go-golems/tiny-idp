---
Title: Production Device Authorization Analysis Design and Implementation Guide
Ticket: TINYIDP-DEVICE-PROD-001
Status: active
Topics:
    - identity
    - oidc
    - oauth2
    - security
    - architecture
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: internal/fositeadapter/provider.go
      Note: Strict route and Fosite handler composition point
    - Path: internal/fositeadapter/sqlstore.go
      Note: Fosite client, request, session, and token persistence adapter
    - Path: internal/server/device.go
      Note: Existing mock protocol sketch that must remain outside the production security boundary
    - Path: internal/server/token.go
      Note: Mock polling behavior and protocol error reference
    - Path: pkg/embeddedidp/bootstrap.go
      Note: Client capability bootstrap and drift surface
    - Path: pkg/embeddedidp/options.go
      Note: Production readiness and host configuration gate
    - Path: pkg/idpstore/interfaces.go
      Note: Public durable transaction and named-transition contract
    - Path: pkg/idpstore/types.go
      Note: Public client and future device grant domain types
    - Path: pkg/sqlitestore/store.go
      Note: Persistent schema and transaction implementation surface
ExternalSources:
    - sources/rfc-8628-oauth-device-authorization-grant.md
    - sources/rfc-9700-oauth-security-bcp.md
    - sources/rfc-8414-authorization-server-metadata.md
    - sources/openid-connect-discovery-1.0.md
Summary: Intern-ready design for adding durable, strict, auditable OAuth device authorization to the production embedded provider.
LastUpdated: 2026-07-14T18:20:00Z
WhatFor: Guides implementation and review of production device authorization without promoting the existing mock fixture into the security boundary.
WhenToUse: Use before changing device endpoints, client capabilities, Fosite token handlers, durable protocol state, verification UI, maintenance, backup, or conformance tests.
---



# Production Device Authorization

## 1. Executive summary

tiny-idp contains a useful device-flow simulation in the mock server. It does
not yet contain a production device authorization grant. The distinction is
important. The mock implementation demonstrates the protocol vocabulary and
supports scenario testing, but it stores raw codes in a process-local map,
authenticates fixture users, and does not share the strict provider's Fosite
token lifecycle, durable storage, audit guarantees, maintenance, or recovery
model.

This design adds RFC 8628 to the strict embedded provider as a first-class
grant. The implementation must preserve the properties already established by
the production browser flow:

- explicit client capabilities;
- durable protocol state;
- secret-free persistence and logging;
- atomic one-time transitions;
- Fosite-compatible access, refresh, ID token, UserInfo, revocation, and
  introspection behavior;
- strict request parsing and bounded responses;
- authenticated, CSRF-protected user decisions;
- production rate limiting and client-address resolution;
- retention, backup, restore, readiness, audit, and metrics integration;
- deterministic, race, failure-injection, fuzz, and end-to-end tests.

The core design decision is that device authorization is not a parallel token
system. It is a new way to establish an authorized grant, after which the
existing token subsystem remains authoritative. A custom Fosite token endpoint
handler recognizes the RFC 8628 grant type and issues the same token families
as the authorization-code handler. Durable device state and token persistence
must commit atomically.

## 2. Reader prerequisites

An intern should understand these protocol roles before editing code:

- **Authorization server:** tiny-idp. It authenticates the user, records the
  decision, and issues tokens.
- **Device client:** a CLI, television, appliance, or other client that cannot
  conveniently receive a browser redirect.
- **User agent:** the separate browser in which the user signs in and approves
  or denies the device request.
- **Resource server:** an API that accepts the resulting access token.
- **Relying party:** when `openid` is requested, the client also relies on the
  ID token as an authentication statement.

The device is not trusted merely because a user typed a displayed code. The
device must retain a high-entropy `device_code`; the browser receives only a
shorter `user_code`; and the server binds both to one durable grant.

Read these repository documents first:

- `docs/security-profile.md` defines the strict provider's enabled controls.
- `docs/storage.md` explains callback-scoped transactions and named atomic
  operations.
- `docs/embedding-foundations.md` explains bootstrap, host lifecycle, and the
  current device-client placeholder.
- `docs/interaction-rendering.md` defines the bounded presentation boundary.
- `internal/fositeadapter/provider.go` shows strict route and Fosite assembly.
- `internal/fositeadapter/sqlstore.go` adapts public durable state to Fosite.
- `pkg/idpstore/interfaces.go` is the public persistence contract.

## 3. Normative protocol flow

RFC 8628 defines a two-channel protocol:

```text
device client                 tiny-idp                    browser/user
     |                           |                              |
     | POST /device_authorization                              |
     | client_id, scope          |                              |
     |-------------------------->|                              |
     | device_code, user_code,   |                              |
     | verification_uri, expiry  |                              |
     |<--------------------------|                              |
     |                           |  GET /device                 |
     | display instructions      |<-----------------------------|
     |                           |  POST code + CSRF             |
     |                           |<-----------------------------|
     |                           |  authenticate and show grant  |
     |                           |------------------------------>|
     |                           |  POST explicit approve/deny   |
     |                           |<-----------------------------|
     | POST /token (poll)        |                              |
     |-------------------------->|                              |
     | authorization_pending OR  |                              |
     | slow_down OR tokens       |                              |
     |<--------------------------|                              |
```

The device authorization response contains:

- `device_code`: an opaque, high-entropy credential returned only to the
  device;
- `user_code`: a short code designed for manual entry;
- `verification_uri`: the browser entry page;
- optional `verification_uri_complete`: a convenience URI containing the user
  code;
- `expires_in`: remaining lifetime in seconds;
- `interval`: minimum polling interval in seconds.

The token request uses the exact grant type
`urn:ietf:params:oauth:grant-type:device_code` and includes `device_code` and
client identity. Before approval it receives `authorization_pending`. Polling
too quickly receives `slow_down`, and the client must increase its polling
interval by five seconds for that and all subsequent requests. Terminal errors
include `access_denied` and `expired_token`.

## 4. Current state

### 4.1 Mock implementation

`internal/server/device.go` and the device branch of
`internal/server/token.go` provide development behavior. They are valuable as
an executable protocol sketch and a source of negative test cases. They are
not a production implementation because:

- `deviceGrants` is an in-memory map, so restart loses every grant;
- raw device and user codes are stored as map keys or values;
- user-code lookup scans the map;
- decisions and token consumption do not participate in the durable Fosite
  transaction;
- users come from scenario fixtures;
- rate limits are not the strict provider's production policy;
- an empty verification action can become approval;
- backup, restore, maintenance, and readiness cannot observe the state;
- multi-process or concurrent transition semantics are undefined.

Do not refactor this map into the strict provider. Keep the mock as a protocol
test fixture and build the production implementation on public store contracts.

### 4.2 Strict provider

`internal/fositeadapter/provider.go` currently mounts discovery, JWKS,
authorization, token, UserInfo, end-session, health, and readiness handlers.
Its Fosite composition supports Authorization Code + PKCE and refresh tokens.
Fosite v0.49.0 does not ship an RFC 8628 handler, but it exposes
`fosite.TokenEndpointHandler`, so tiny-idp can register one without bypassing
Fosite's client-authentication and response pipeline.

### 4.3 Public storage

`pkg/idpstore/interfaces.go` already establishes the right design language:
durable records, callback-scoped `WithTx`, and named invariant-preserving
operations. It has no device-grant record or transition API. The SQLite store
therefore has no device table, indexes, migration, cleanup, backup assertion,
or restore test.

### 4.4 Client bootstrap

`embeddedidp.DeviceClient` creates a public client with no redirects, but
`idpstore.Client` does not encode allowed grant types. The Fosite adapter
currently returns authorization-code and refresh-token grants for clients.
Production device support therefore requires explicit grant capabilities; a
no-redirect client must not accidentally become a browser-flow client.

## 5. Security invariants

The implementation is complete only when these invariants are executable in
tests and observable in audit results.

### 5.1 Code secrecy

1. Raw `device_code` is returned once and never persisted, logged, audited, or
   included in an error.
2. Raw `user_code` is returned once and may be rendered back to the user during
   confirmation, but is never persisted or logged.
3. Database lookup uses keyed, domain-separated hashes, not unkeyed SHA-256.
4. Device and user code hash namespaces are distinct.

Conceptual pseudocode:

```text
device_raw = random(32 bytes)
user_raw   = generate_human_code()

device_hash = HMAC(protocol_key, "device-code\x00" || device_raw)
user_hash   = HMAC(protocol_key, "user-code\x00" || normalize(user_raw))

INSERT grant(device_hash, user_hash, ...)
RETURN device_raw, display(user_raw)
```

### 5.2 Client binding

Every create, poll, and consume operation is bound to one client ID. The token
endpoint must authenticate or identify the client according to its registered
type. A device code created for client A cannot be polled by client B, even if
the raw code is disclosed.

### 5.3 Explicit capability

Every client has explicit allowed grant types. Recommended constants are:

```go
const (
    GrantAuthorizationCode = "authorization_code"
    GrantRefreshToken      = "refresh_token"
    GrantDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
)
```

`BrowserClient` requests authorization-code and refresh-token capability.
`DeviceClient` requests device-code capability and, only when policy permits,
refresh-token capability. Empty grant lists are invalid after migration; do
not retain an implicit compatibility default.

### 5.4 One terminal decision

A pending grant may transition to approved, denied, or expired exactly once.
Approve and deny races cannot both succeed. A terminal decision cannot be
reversed by replaying the browser form.

### 5.5 One token issuance

An approved device grant can produce one initial token response. Concurrent
pollers cannot both consume it. A process crash cannot produce tokens while
leaving the grant reusable, or consume the grant while rolling back every
token record.

### 5.6 Polling discipline

The server stores the next permitted poll time. This state is durable and
updated atomically. `slow_down` increases the interval by five seconds. The
decision cannot depend only on a process-local limiter because restarts or a
second process would reset it.

### 5.7 Browser authorization

Typing a valid user code is not authorization. The browser user must:

- authenticate through the same password/session policy as browser OIDC;
- see the client identity and requested scopes;
- see and confirm the user code, including when using
  `verification_uri_complete`;
- explicitly approve or deny;
- submit a server-bound CSRF token;
- receive a generic response for invalid, expired, or already-used codes.

### 5.8 No authority expansion

Approved scopes are a subset of requested scopes, which are a subset of the
client's allowed scopes. The verification page cannot add scopes. Disabled
clients and disabled users cannot finish the grant. The user and client are
revalidated at the terminal transition and again inside token issuance where
necessary.

## 6. Durable domain model

Add `idpstore.DeviceGrant` with a deliberately protocol-oriented schema:

```go
type DeviceGrant struct {
    ID                 string
    DeviceCodeHash     []byte
    UserCodeHash       []byte
    ClientID           string
    RequestedScopes    []string
    ApprovedScopes     []string
    Status             DeviceGrantStatus
    UserID             string
    Subject            string
    AuthTime           time.Time
    AuthenticationMethods []string
    CreatedAt          time.Time
    ExpiresAt          time.Time
    PollInterval       time.Duration
    NextPollAt         time.Time
    SlowDownCount      uint32
    DecidedAt          time.Time
    ConsumedAt         time.Time
    Version            uint64
}
```

The exact subject and authentication context should be captured when the user
approves so the token response represents that authentication event. Storing
`UserID` permits revalidation; storing the stable pairwise/public subject used
by the provider prevents later login-name changes from changing the grant.

Recommended statuses are `pending`, `approved`, `denied`, and `consumed`.
Expiry is derived from `ExpiresAt`, not necessarily written as a status. This
keeps expiry checks authoritative under transaction time. Maintenance may
materialize or delete old expired records after the audit retention window.

SQLite constraints:

- primary key on internal grant ID;
- unique fixed-length device-code hash;
- unique fixed-length user-code hash;
- foreign key to client ID;
- status check constraint;
- nonnegative poll interval and slowdown count;
- approved rows require user, subject, and decision time;
- consumed rows require consumption time;
- indexes on expiry and status for maintenance.

## 7. Store API

Do not expose a general `UpdateDeviceGrant(func(*DeviceGrant))`. It makes the
security state machine dependent on every caller. Add named operations whose
SQL predicates encode the legal transition:

```go
type DeviceGrantStore interface {
    CreateDeviceGrant(ctx context.Context, grant DeviceGrant) error
    GetDeviceGrantByUserCodeHash(ctx context.Context, hash []byte) (DeviceGrant, error)
    InspectDeviceGrantByDeviceCodeHash(ctx context.Context, hash []byte, clientID string) (DeviceGrant, error)
    PollDeviceGrant(ctx context.Context, request DevicePollRequest) (DevicePollResult, error)
    DecideDeviceGrant(ctx context.Context, request DeviceDecisionRequest) (DeviceGrant, error)
    ConsumeDeviceGrant(ctx context.Context, request DeviceConsumeRequest) (DeviceGrant, error)
}
```

`PollDeviceGrant` must atomically compare `now` to `NextPollAt`, update the next
time, and return a typed result. `DecideDeviceGrant` uses a predicate equivalent
to `status = pending AND expires_at > now`. `ConsumeDeviceGrant` uses
`status = approved AND consumed_at IS NULL AND expires_at > now`.

These operations must also be available on callback-scoped `TxStore`, so the
device consumption and Fosite token writes share one `sql.Tx`.

## 8. Endpoint design

### 8.1 Device authorization endpoint

Mount `POST {issuer}/device_authorization`. Reject every other method. Require
`application/x-www-form-urlencoded`, cap the body before parsing, reject
duplicate security-sensitive parameters, authenticate or identify the client,
and enforce `GrantDeviceCode` capability.

Validation order:

```text
validate method and media type
resolve rate-limit client address
parse bounded form strictly
authenticate/identify client
check client enabled and device-grant capability
parse requested scopes once
require openid when an ID token is expected
enforce client allowed scopes
generate both codes and keyed hashes
insert with collision retry bounded to a small constant
emit durable audit event
return no-store JSON response
```

The endpoint advertises a stable `verification_uri`. It may include
`verification_uri_complete`, but the browser confirmation page still shows the
code and asks the user to compare it with the device.

### 8.2 Verification endpoint

Mount `GET` and `POST {issuer}/device`.

The GET page has two modes:

- code entry, when no valid code has been supplied;
- confirmation, after lookup and authentication establish a pending grant.

The POST action must be explicit. Missing or unknown actions are invalid and
must never default to approval. Prefer a short-lived, server-side verification
interaction record bound to browser CSRF state and the device grant ID. Do not
place scopes, subject, or the raw device code in hidden inputs.

The existing `idpui.InteractionRenderer` is authorization-request specific.
Add a sibling interface such as `DeviceVerificationRenderer` with typed,
bounded view data. The default renderer and custom renderers must share the
same security headers, output limits, panic containment, and safe fallback
policy documented in `docs/interaction-rendering.md`.

### 8.3 Token endpoint handler

Implement `fosite.TokenEndpointHandler` in `internal/fositeadapter`. Its
`CanHandleTokenEndpointRequest` recognizes only the RFC 8628 grant type.
`HandleTokenEndpointRequest` validates client grant capability, the device
code, requested scope rules, and current durable state. Pending, slowdown,
denied, and expired outcomes map to RFC error responses.

`PopulateTokenEndpointResponse` performs the terminal transaction:

```text
WITH durable transaction:
    grant = consumeApprovedDeviceGrant(device_hash, client_id, now)
    session = buildFositeSession(grant.subject, grant.auth_context)
    access = generate and persist access token through existing strategy/store
    if offline_access approved:
        refresh = generate and persist refresh token
    if openid approved:
        id_token = existing RS256 OpenID strategy.GenerateIDToken(session)
    commit
RETURN standard Fosite access response
```

The implementation should factor token construction shared with existing
flows rather than reproduce claim mapping. The generated access token must be
accepted by the existing UserInfo and introspection paths. Refresh tokens must
be accepted by the existing refresh handler.

Do not intercept `/token` before Fosite and construct a separate token format.
That would duplicate client authentication, error encoding, lifetimes,
rotation, introspection state, and audit behavior.

## 9. Fosite integration details

Fosite's extension surface is `fosite.TokenEndpointHandler`:

```go
type TokenEndpointHandler interface {
    HandleTokenEndpointRequest(context.Context, AccessRequester) error
    PopulateTokenEndpointResponse(context.Context, AccessRequester, AccessResponder) error
    CanHandleTokenEndpointRequest(context.Context, AccessRequester) bool
    CanSkipClientAuth(context.Context, AccessRequester) bool
}
```

`CanSkipClientAuth` should remain false. This does not prohibit public clients;
Fosite can identify a registered public client without a secret. It prevents
the extension handler from declaring that no client identity is necessary.

Use built-in handlers as implementation references:

- `handler/oauth2/flow_authorize_code_token.go` for access/refresh generation,
  request scope propagation, and storage ordering;
- `handler/openid/flow_explicit_token.go` and `handler/openid/helper.go` for ID
  token generation;
- `access_request_handler.go` for client authentication dispatch;
- `compose/compose.go` and the existing provider factory list for registration.

Compile-time assertions are required:

```go
var _ fosite.TokenEndpointHandler = (*DeviceGrantHandler)(nil)
```

## 10. Client metadata migration

Add `AllowedGrantTypes` to `idpstore.Client`, validation, JSON persistence,
admin output, bootstrap drift detection, and Fosite client adaptation.

Because existing databases contain clients without this field, add a schema
migration that deterministically assigns browser clients
`authorization_code,refresh_token`. This is a data migration, not a runtime
compatibility adapter. After migration, empty grant metadata is invalid.

Update:

- `pkg/idpstore/types.go` validation and constants;
- `pkg/embeddedidp/bootstrap.go` constructors and drift fields;
- `internal/admin/clients.go` and `internal/cmds/admin_client.go` input/output;
- `internal/fositeadapter/sqlstore.go` `GetGrantTypes` mapping;
- SQLite migrations and migration tests;
- backup/restore schema expectations;
- docs and examples.

## 11. Rate limiting

Use both the configured production limiter and durable protocol timing.
Recommended dimensions are:

- device authorization creation: client ID plus resolved address;
- verification code submission: resolved address plus a nonreversible hash
  bucket, with a global address ceiling;
- password authentication: existing account/address controls;
- token polling: client plus device hash, with durable `NextPollAt`;
- invalid-code traffic: address-only limits to prevent unlimited enumeration.

Never put a raw code in a limiter key. Production readiness must reject a
limiter or client-address resolver that does not implement the existing
`ProductionReadyReporter` contract.

## 12. Audit and operational telemetry

Recommended audit events:

- `device.authorization.created`;
- `device.authorization.rejected`;
- `device.verification.code_rejected`;
- `device.authorization.approved`;
- `device.authorization.denied`;
- `device.token.pending`;
- `device.token.slow_down`;
- `device.token.issued`;
- `device.token.rejected`;
- `device.grant.expired` when maintenance accounts for expiry.

Safe fields include client ID, subject after authentication, approved scope
names, result, stable reason code, and coarse request-address classification.
Forbidden fields include device code, user code, their hashes, passwords,
cookies, CSRF values, access tokens, refresh tokens, and ID tokens.

Metrics should be low-cardinality counters and histograms:

- grants created, approved, denied, expired, consumed;
- polling outcome counts;
- decision latency and issue latency;
- collision retry count;
- active pending grant gauge obtained through bounded store aggregation;
- renderer failure/overflow counters;
- durable transition conflict count.

Do not label metrics by user, device code, subject, or arbitrary client ID.

## 13. Maintenance, backup, and readiness

Maintenance deletes expired and consumed device grants only after configured
protocol-state retention. It reports those deletions in
`MaintenanceReport.ProtocolRecords`. Index expiry so the pass remains bounded.

SQLite online backup automatically copies the table, but backup verification
must explicitly confirm the device schema and integrity. Restore tests must
cover a pending grant, an approved unconsumed grant, and a consumed grant.

Readiness should fail in production when:

- the configured schema predates device support while a device client exists;
- the store lacks the device transition interface;
- required hashing key material is unavailable;
- audit or limiter readiness fails under existing policy.

Readiness should not fail merely because there are no pending grants.

## 14. Test architecture

### 14.1 Table tests

Cover every state and request combination:

| State | Poll at/after interval | Early poll | Verification decision |
|---|---|---|---|
| pending | `authorization_pending` | `slow_down` | approve or deny once |
| approved | issue once | `slow_down` if early | conflict |
| denied | `access_denied` | `access_denied` | conflict |
| consumed | invalid/terminal error | same | conflict |
| expired | `expired_token` | `expired_token` | conflict |

Also test wrong client, disabled client, disabled user, disallowed scope, missing
`openid`, unsupported grant, duplicate form keys, wrong content type, oversized
body, unknown action, missing CSRF, stale browser session, and renderer failure.

### 14.2 State-machine model

Define a small pure reference model:

```text
state Pending
Approve: Pending -> Approved
Deny:    Pending -> Denied
Expire:  Pending|Approved -> Expired (derived by time)
Poll:    Pending -> Pending + next_poll
Slow:    early poll -> interval + 5s + next_poll
Consume: Approved -> Consumed
all other terminal transitions -> conflict
```

Generate action sequences and compare the SQLite implementation after each
step. This is lightweight model-based testing and complements, rather than
replaces, a future TLA+/PlusCal model.

### 14.3 Concurrency tests

Run under `go test -race`:

- approve versus deny;
- approve versus expiry boundary;
- two valid pollers racing to consume;
- early poll versus on-time poll;
- maintenance versus approve;
- backup versus decision and token consumption.

Each test asserts legal result sets and final durable state, not scheduling.

### 14.4 Failure injection

Insert failpoints at:

- after grant consumption but before access token persistence;
- after access token but before refresh token persistence;
- after token persistence but before transaction commit;
- after decision update but before audit delivery;
- renderer panic and renderer overflow;
- SQLite busy, disk full, canceled context, and closing store.

The expected result is rollback or a fail-closed error. No failpoint may leave
both reusable grant state and committed tokens.

### 14.5 Fuzzing

Fuzz harnesses should target:

- user-code normalization and separator handling;
- bounded form parsing with duplicate keys and invalid UTF-8;
- device authorization response encoding;
- token polling error mapping;
- state-machine action sequences;
- renderer view construction;
- JWK/token claim construction shared with device ID tokens.

Seed every harness with valid, near-valid, empty, oversized, repeated, Unicode,
and delimiter-heavy inputs. Fuzz tests assert no panic, bounded allocation,
stable normalization, and invariant preservation.

### 14.6 End-to-end test

Build a small CLI test client that:

1. discovers `device_authorization_endpoint`;
2. requests `openid profile offline_access`;
3. polls once and observes pending;
4. drives the browser verification flow as a test user;
5. polls and receives tokens;
6. validates the ID token and calls UserInfo;
7. refreshes once if offline access was approved;
8. proves the original device code cannot issue again;
9. restarts the provider at pending and approved checkpoints;
10. repeats after backup and restore.

## 15. Static analysis and review gates

Extend the repository's Go AST analyzers with device-specific checks where
syntax provides a reliable signal:

- forbid audit/log fields named `device_code`, `user_code`, `access_token`, or
  token-like variants in device packages;
- require strict endpoint handlers to call bounded form parsing helpers;
- flag direct status assignment outside the named store transition files;
- flag map-backed device state in production adapter packages;
- verify all custom Fosite handlers have compile-time interface assertions;
- verify route registration and discovery metadata are changed together.

AST checks cannot prove atomicity or protocol correctness. SQL transition
tests, race tests, model-based sequences, and failure injection remain the
primary evidence for those properties.

Release gates:

```text
go test ./...
go test -race ./...
go vet ./...
make lint
make gosec
device model-sequence harness
device restart and backup/restore harness
external RFC 8628 smoke client
manual verification-page accessibility/security review
```

## 16. Implementation phases

### Phase 0 — Freeze the contract

Write decision records for grant metadata, code format and entropy, hashing key
derivation, lifetimes, polling interval, offline access, error mapping, audit
vocabulary, and UI extension. Turn every security invariant into a named test.

### Phase 1 — Client capabilities and schema

Add explicit grant types, migrations, admin/Bootstrap support, and drift tests.
This phase must ship without advertising device authorization.

### Phase 2 — Durable device state

Add records, table constraints, named operations, memory-store parity, SQLite
transaction tests, maintenance, backup, and restore coverage.

### Phase 3 — Device authorization endpoint

Implement strict client/scope validation, code generation, keyed hashing,
collision retry, durable creation, audit, response headers, and discovery
metadata behind a feature/configuration gate until token issuance is complete.

### Phase 4 — Verification browser flow

Add typed rendering, code entry, authentication/session integration, CSRF,
explicit confirmation, approve/deny atomic transitions, generic errors, audit,
and renderer containment tests.

### Phase 5 — Fosite token handler

Implement polling outcomes and atomic token issuance, including ID token,
UserInfo, optional refresh, introspection, replay prevention, failure injection,
and concurrency tests. Remove the temporary advertisement gate only when this
phase passes.

### Phase 6 — Operational integration

Add metrics, readiness, retention, backup verification, restore drills,
runbooks, and rate-limit tuning guidance.

### Phase 7 — Adversarial verification

Run fuzz, race, model sequences, restart tests, an external device client, and
applicable OIDC conformance profiles. Review audit output for secret absence.

### Phase 8 — Documentation and release

Update public embedding APIs, admin documentation, discovery examples, security
profile, changelog, and a third runnable demo that uses device authorization.
Require an independent security review before labeling the grant production
ready.

## 17. Alternatives considered

### Promote the mock device map

Rejected. It cannot meet restart, atomicity, secret persistence, backup,
maintenance, or production account requirements.

### Implement `/token` outside Fosite

Rejected. It would create a second token lifecycle and duplicate security
logic for client authentication, token strategies, storage, errors, scopes,
refresh, introspection, and UserInfo.

### Store raw codes encrypted

Rejected. Lookup needs only keyed hashes, and encryption preserves unnecessary
recoverability. Return raw codes once and persist only derived lookup values.

### Use only the general rate limiter for polling

Rejected. RFC polling interval is grant state and must survive restart and
concurrency. The general limiter remains a complementary abuse boundary.

### Reuse authorization interactions unchanged

Rejected. Authorization interactions encode redirect-based request semantics.
Device verification needs a smaller typed record and view. Shared rendering and
browser-session infrastructure should be factored, not hidden behind an
incorrect domain type.

## 18. Review checklist

- [ ] No raw device/user code appears in storage, audit, logs, metrics, errors,
      URLs other than the explicit complete verification URI, or tests that
      print failures.
- [ ] Every client has explicit grant capabilities after migration.
- [ ] Public clients are identified; confidential clients are authenticated.
- [ ] Approve, deny, expire, poll, and consume are named atomic operations.
- [ ] Token persistence and device consumption share a transaction.
- [ ] Empty browser actions fail and never approve.
- [ ] User code, client, and scopes are visibly confirmed.
- [ ] CSRF and browser authentication policies match strict browser login.
- [ ] Slowdown state survives restart.
- [ ] Access tokens work with UserInfo and introspection.
- [ ] Refresh is issued only under explicit client/scope policy.
- [ ] Maintenance, backup, restore, readiness, audit, and metrics cover grants.
- [ ] Race, fuzz, model, failpoint, restart, and end-to-end tests pass.
- [ ] Discovery advertises the endpoint only when the complete implementation
      is active.

## 19. References

Primary sources are preserved in this ticket's `sources/` directory:

- `sources/rfc-8628-oauth-device-authorization-grant.md` — normative device
  protocol, polling errors, usability, brute-force, and phishing guidance.
- `sources/rfc-9700-oauth-security-bcp.md` — current OAuth security practices.
- `sources/rfc-8414-authorization-server-metadata.md` — authorization server
  metadata framework.
- `sources/openid-connect-discovery-1.0.md` — OpenID Provider discovery and
  issuer consistency.

The implementation must use the published specifications as normative text;
this design records how their requirements map onto tiny-idp's architecture.
