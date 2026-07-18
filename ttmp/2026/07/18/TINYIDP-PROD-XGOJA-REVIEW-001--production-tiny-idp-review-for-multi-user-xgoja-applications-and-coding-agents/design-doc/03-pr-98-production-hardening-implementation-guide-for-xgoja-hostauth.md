---
Title: PR 98 production hardening implementation guide for xgoja hostauth
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
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/device_handlers.go
      Note: |-
        Native device, refresh, revoke, approval, audit, and security-event HTTP boundary
        Native device HTTP boundary
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/oauth_token.go
      Note: |-
        Access and rotating refresh-token lifecycle and revocation semantics
        Token lifecycle and revocation semantics
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/ratelimit.go
      Note: |-
        Planned-route rate-limit contract and current RemoteAddr client identity
        Current planned-route limiter and client IP behavior
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/preflight.go
      Note: |-
        Current single-node production configuration contract
        Single-node preflight contract
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/xgoja/hostauth/readiness.go
      Note: |-
        Current static topology report that must become dependency-aware readiness
        Static readiness topology report
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Current tiny-idp device and authenticated introspection endpoint implementation
        Current device and introspection handlers
    - Path: repo://internal/oidcmeta/discovery.go
      Note: |-
        Current tiny-idp discovery contract already advertising device authorization and RFC 7662 introspection
        Current tiny-idp discovery and introspection contract
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/pull/98
    - https://www.rfc-editor.org/rfc/rfc8628
    - https://www.rfc-editor.org/rfc/rfc7662
    - https://www.rfc-editor.org/rfc/rfc7009
    - https://www.rfc-editor.org/rfc/rfc8707
Summary: Intern-facing implementation chapter for finishing the production safety boundaries around PR 98 without confusing application-owned programauth credentials with tiny-idp-issued OAuth credentials.
LastUpdated: 2026-07-18T16:45:00-04:00
WhatFor: Explain the remaining security and operations work in PR 98, why each invariant matters, where to implement it, and how to prove it with tests.
WhenToUse: When implementing, reviewing, or deploying PR 98 and its follow-up changes for an internet-facing single-replica xgoja host behind Traefik.
---


# PR 98 production hardening implementation guide for xgoja hostauth

## Purpose and expected outcome

With merged PR 95 as its baseline, PR 98 gives generated xgoja hosts durable
OIDC login transactions, end-to-end durable auth-store wiring, rotating refresh
tokens, refresh-family revocation, configuration preflight, a single-node
deployment profile, and security-event hooks. Those changes are substantial.
They make restarts and token lifecycle operations much more predictable than
they were at the PR 95 merge point.

Durability is not the final property a public authentication service needs. A
service can preserve every record correctly and still make an unsafe decision
about the caller's IP address, accept an unauthorized action name, report
itself ready while its database is unavailable, or leave users without a way
to disconnect an agent. This guide explains the remaining boundaries and gives
the implementer a concrete order in which to finish them.

By the end of the work described here, one xgoja process should be safe to run
behind the cluster's Traefik ingress with durable SQL storage. The deployment
will still be intentionally single-replica. High availability and shared
tiny-idp-issued device credentials remain separate projects.

The central result is:

~~~text
public request
  -> trusted ingress interpretation
  -> native endpoint request budget
  -> application-owned device policy
  -> durable state transition
  -> auditable, non-secret outcome
  -> dependency-aware readiness
~~~

Each arrow is a security boundary. None can be replaced by documentation about
another arrow.

## 1. Begin with the ownership model

The easiest mistake in this subsystem is to use the phrase “device token”
without naming its issuer. PR 98 contains two identity systems that cooperate
but do not share bearer credentials.

Tiny-idp authenticates the human. The xgoja application authenticates the
agent. A browser session created after OIDC login authorizes an application
device request; the resulting `ggat_` access token is created by xgoja
`programauth`, stored in the application's database, and validated by the
application's `CompositeAuthenticator`.

~~~text
Human browser                         Coding agent
     |                                    |
     | OIDC authorization code + PKCE     | app device_code
     v                                    v
  tiny-idp                         generated xgoja host
     |                                    |
     | verified ID token                  | ggat_ / ggrt_
     v                                    v
local app session  -- approves --> local programauth agent
~~~

This division is a valid design for a standalone application. It avoids a
runtime introspection call for every agent request and lets the application use
its own action vocabulary. It also has a direct consequence: tiny-idp cannot
revoke a `ggat_` token because tiny-idp did not issue it.

Keep this statement visible during implementation:

> PR 98 hardens application-owned device authorization. It does not turn the
> xgoja host into a resource server for tiny-idp-issued access tokens.

The distinction prevents several implementation errors:

- Do not send `ggat_` tokens to tiny-idp introspection.
- Do not accept a tiny-idp opaque access token in `programauth` merely because
  both credentials use the HTTP Bearer scheme.
- Do not assume that disabling a tiny-idp account automatically disables an
  application agent.
- Do not describe the application refresh endpoint as a tiny-idp OAuth
  endpoint.

## 2. Preserve what PR 98 already gets right

The work below should extend PR 98, not rewrite its sound foundations. Before
changing code, trace and test the following paths.

### 2.1 Durable OIDC login transactions

An OIDC login transaction contains state, nonce, a PKCE verifier, the safe
return path, creation time, and expiry. It exists before the browser leaves for
tiny-idp and is consumed when the callback returns. It is not an application
session and not an OAuth token cache.

The SQL transaction store correctly makes `Take` a delete-and-return operation.
Two callbacks presenting the same state cannot both consume it. Preserve that
one-use property.

~~~text
GET /auth/login
  generate state, nonce, verifier
  persist transaction
  redirect to tiny-idp

GET /auth/callback?state=S&code=C
  atomically take S
  exchange C with verifier
  verify ID-token issuer, audience, nonce
  create local session
~~~

### 2.2 Atomic programauth token pairs

An access/refresh pair must be returned only if both token hashes were stored.
Refresh rotation must create the next access token, create the next refresh
token, and consume the current refresh token in one SQL transaction. PR 98's
`OAuthTokenPairStore` is the correct capability boundary for this operation.

Do not weaken it into several independent writes. A partial rotation can either
lose the user's only valid refresh credential or create an access token the
server never returned to the legitimate caller.

### 2.3 Explicit single-node production profile

The `single-node` profile tells the operator exactly what is supported:

- one serving xgoja process;
- durable SQLite or PostgreSQL stores;
- no runtime schema application;
- secure browser cookies;
- HTTPS issuer and public callback URLs; and
- a process-local memory rate limiter.

It is good engineering to reject an unsupported topology instead of silently
running it. Keep the profile name and its fail-closed checks explicit. Do not
rename it to `production`, because that would imply multi-replica behavior the
implementation does not provide.

### 2.4 Application refresh and revocation semantics

PR 98 mounts:

| Method | Path | Meaning |
|---|---|---|
| `POST` | `/auth/device/refresh` | Rotate one application refresh credential and issue a new pair. |
| `POST` | `/auth/device/revoke` | Revoke the refresh-token family identified by a presented refresh token. |

The revocation endpoint does not revoke already-issued access tokens. They
remain valid until their short expiry. This is a defensible bounded behavior,
but the user-facing disconnect operation must state it accurately.

## 3. Priority zero: establish one trustworthy request identity

### 3.1 Why the proxy boundary must be explicit

In k3s, Traefik terminates public TLS and forwards private HTTP to the pod. The
application needs two different pieces of information:

- Its public origin is `https://app.example.test`.
- Its listener receives HTTP from a Traefik address on the cluster network.

PR 98's `auth.oidc.public-base-url` handles the first fact. It lets the host
construct an exact HTTPS callback without inspecting forwarding headers. That
is an important improvement.

The second fact is not yet modeled. Today, audit code trusts the first
`X-Forwarded-For` value unconditionally, while planned route rate limiting uses
`RemoteAddr`. Behind Traefik, these two subsystems can assign different client
addresses to the same request.

~~~text
request                       audit IP             limiter IP
-------                       --------             ----------
direct with forged XFF        attacker-chosen      direct peer
through Traefik               original client      Traefik pod
~~~

The first row makes audit attribution spoofable when the service is directly
reachable. The second row places every public user into the same IP rate-limit
bucket. Fixing only one consumer would preserve the disagreement.

### 3.2 Add a host-level proxy policy

Add configuration that defines how network identity is resolved. Use an
explicit mode; do not infer trust merely because an `X-Forwarded-For` header is
present.

~~~go
type ProxyMode string

const (
    ProxyModeDirect           ProxyMode = "direct"
    ProxyModeTrustedForwarded ProxyMode = "trusted-forwarded"
)

type ProxyConfig struct {
    Mode         ProxyMode `yaml:"mode"`
    TrustedCIDRs []string  `yaml:"trusted-cidrs"`
}
~~~

The production configuration should resemble:

~~~yaml
auth:
  deployment:
    profile: single-node
  proxy:
    mode: trusted-forwarded
    trusted-cidrs:
      - 10.42.0.0/16
~~~

Use the narrowest stable Traefik source range the cluster can guarantee. If the
live cluster cannot provide a dedicated source range, enforce a NetworkPolicy
that permits ingress only from Traefik and document the selected pod or
namespace CIDR. A broad cluster CIDR is a temporary operational compromise,
not an invisible default.

### 3.3 Resolve the address once

Create one Go-owned resolver and put its result in request context before
authentication, rate limiting, audit, access logging, or JavaScript request
projection runs.

~~~go
type RequestIdentity struct {
    PeerIP   netip.Addr
    ClientIP netip.Addr
    ViaProxy bool
}

type RequestIdentityResolver interface {
    ResolveRequestIdentity(ctx context.Context, r *http.Request) (RequestIdentity, error)
}

var _ RequestIdentityResolver = (*TrustedProxyResolver)(nil)
~~~

The resolution algorithm should be easy to review:

~~~text
resolve(request):
  peer = parse request.RemoteAddr

  if mode == direct:
    reject or ignore forwarding headers
    return client = peer, viaProxy = false

  if peer is not in trusted proxy CIDRs:
    ignore forwarding headers
    return client = peer, viaProxy = false

  parse standardized Forwarded or configured X-Forwarded-For chain
  reject malformed addresses and cap header size / hop count
  walk right-to-left across trusted proxy hops
  return first untrusted address as client, viaProxy = true
~~~

Walking from the right matters because a trusted proxy appends information to
an existing chain. The leftmost value can already have been supplied by the
caller. The exact Traefik forwarding configuration and resolver algorithm must
be tested together.

### 3.4 Feed every consumer from the same value

After resolution:

- `gojahttp` rate-limit keys use `RequestIdentity.ClientIP`.
- Audit hashes `RequestIdentity.ClientIP`.
- Access logs record both peer and client addresses, subject to the existing
  privacy policy.
- JavaScript receives only the intended normalized client value.
- Native device handlers use the same identity when applying their budgets.

Do not let each package parse forwarding headers independently.

### 3.5 Proxy acceptance tests

Write table-driven tests for at least these cases:

| Peer | Forwarding header | Policy | Expected client |
|---|---|---|---|
| `192.0.2.10` | absent | direct | `192.0.2.10` |
| `192.0.2.10` | forged value | direct | `192.0.2.10` |
| trusted Traefik | one client | trusted | forwarded client |
| trusted Traefik | client plus trusted hops | trusted | first untrusted hop from right |
| untrusted peer | forwarded value | trusted | peer address |
| trusted Traefik | malformed or oversized chain | trusted | fail closed or documented peer fallback |

Then run one ingress-level test against k3s. Unit tests prove parsing;
ingress-level tests prove that the assumed Traefik header shape is real.

## 4. Priority zero: put policy and request budgets around native device endpoints

### 4.1 Planned route limits do not protect native handlers

Express routes can declare `.rateLimit(...)`, but `/auth/device/start`,
`/auth/device/token`, `/auth/device/refresh`, `/auth/device/revoke`, and
`/auth/device/approve` are Go-owned native handlers mounted before the
JavaScript application. They do not pass through the planned-route enforcer.

This distinction matters because the most attackable endpoints are public:

- Device start allocates durable state and generates codes.
- Device poll performs secret lookup and persistent timing updates.
- Refresh performs credential lookup and token rotation.
- User-code approval performs a short-code lookup before creating an agent.

The protocol-level `slow_down` response protects one known device code from
over-polling. It is not a general request budget for unknown codes, device-start
floods, or distributed guessing.

### 4.2 Add a native auth-endpoint policy object

Keep policy in host configuration, not in request JSON and not in tutorial
JavaScript.

~~~go
type DeviceEndpointPolicy struct {
    AllowedActions        map[string]struct{}
    MaxActionsPerRequest  int
    DeviceTTL             time.Duration
    InitialPollInterval   time.Duration
    VerificationPath      string
    StartBudget           RateBudget
    PollBudget            RateBudget
    ApprovalBudget        RateBudget
    RefreshBudget         RateBudget
    RevokeBudget          RateBudget
}
~~~

The request may choose a subset of `AllowedActions`. It may never invent a new
action. Reject an unknown action rather than silently dropping it; a CLI that
misspells a permission should receive a deterministic error instead of a token
that later fails mysteriously.

~~~text
requested = normalize(request.actions)

if requested is empty:
  reject invalid_scope

if count(requested) > MaxActionsPerRequest:
  reject invalid_scope

if any requested action is not in AllowedActions:
  reject invalid_scope

store exactly requested
~~~

For the personal inbox example, the allowlist might contain only
`user.self.read` initially. Production applications must define their own local
action vocabulary.

### 4.3 Do not accept a caller-selected verification origin

`deviceStartRequest` currently includes `verificationUri`. A public client
should not decide which browser location the server advertises as its
verification endpoint. Build the URI from the configured public base URL and a
fixed application path.

This prevents a legitimate-looking device response from directing a user to an
attacker-controlled verification page.

~~~text
verification_uri = PublicBaseURL + DevicePolicy.VerificationPath
verification_uri_complete = verification_uri + "?user_code=" + user_code
~~~

The server may return a relative URI in local development, but the
`single-node` profile should return an absolute HTTPS URI derived from trusted
configuration.

### 4.4 Apply budgets before expensive or revealing work

Inject a limiter into `DeviceHandlersConfig`. Give native endpoint budgets
stable policy names and low-cardinality keys.

Recommended first keys:

| Endpoint | Pre-authentication key | Additional behavior |
|---|---|---|
| start | client IP | Global ceiling to prevent durable-state exhaustion. |
| token poll | client IP | Keep per-device `slow_down`; do not put the raw device code in metrics. |
| approve/inspect | client IP, then session actor | Bound user-code guessing and authenticated abuse. |
| refresh | client IP | Refresh reuse logic remains the credential-level defense. |
| revoke | client IP | Preserve the non-oracle response for unknown credentials. |

Return `429 Too Many Requests` and a correct `Retry-After` header for transport
budgets. Preserve RFC 8628 `slow_down` for a valid device request polled faster
than its stored interval. These are related but different signals.

### 4.5 Add approval inspection and denial

An approval screen cannot make an informed decision if it only accepts a code
and posts a hard-coded action list. Add a session-protected, rate-limited
inspection endpoint that returns a redacted pending request:

Use a POST with a JSON body even though inspection does not mutate state. User
codes should not enter access-log query strings, browser history, or proxy URL
metrics.

~~~http
POST /auth/device/request
Cookie: app_session=...
Content-Type: application/json

{"user_code":"ABCD-EFGH"}
~~~

~~~json
{
  "clientName": "personal-inbox-cli",
  "requestedActions": ["user.self.read"],
  "expiresIn": 418,
  "status": "pending"
}
~~~

Never return the device code, device-code hash, token-family information, or
another user's identity.

Add a denial endpoint using the existing service transition:

~~~http
POST /auth/device/deny
Cookie: app_session=...
X-CSRF-Token: ...
Content-Type: application/json

{"user_code":"ABCD-EFGH"}
~~~

Approval and denial must both require a fresh local session and CSRF. A denied
request is terminal, and subsequent polling returns `access_denied`.

### 4.6 Device endpoint tests

Tests should prove policy, not only successful issuance:

- Unknown actions fail before a device record is inserted.
- An empty action set fails when the application requires an explicit grant.
- A client-supplied verification URI is ignored or rejected.
- Start floods return 429 without growing the device table beyond the budget.
- Unknown user codes are rate-limited and do not reveal whether a close code
  exists.
- Inspection never returns the raw device code.
- Approval cannot broaden the stored request.
- Denial is terminal.
- Two concurrent redemptions produce exactly one token pair.
- Audit and metrics contain no raw code or token.

## 5. Priority zero: make readiness report dependencies, not intentions

### 5.1 The current endpoint is a topology declaration

`BuildReadinessReport` currently sets `ready: true` and lists configured store
drivers. It does not test the database. Also, `sql.Open` is lazy; constructing a
`*sql.DB` does not prove a connection can be established.

The current endpoint answers:

> “Did configuration resolution produce a supported topology?”

Kubernetes readiness must answer:

> “Can this process safely receive a new login, callback, session request, or
> token transition now?”

Both answers are useful, but they should not share one unconditional boolean.

### 5.2 Separate liveness, readiness, and topology

Use three concepts:

| Signal | Question | Dependency behavior |
|---|---|---|
| Liveness | Is the process event loop responsive? | Do not fail only because SQL or tiny-idp is temporarily unavailable. |
| Readiness | Can the process safely serve auth traffic? | Fail when required SQL stores cannot complete a bounded probe. |
| Topology | What mode and drivers were configured? | Return non-secret diagnostics; do not claim live health. |

One possible HTTP shape is:

- `/healthz` for process liveness.
- `/auth/readyz` for dependency readiness.
- Include the redacted topology inside the readiness response or expose it at
  `/auth/configz` if operators need it separately.

### 5.3 Add a health capability at the store boundary

Because store interfaces are intentionally domain-specific, do not add `Ping`
to every domain interface. Add an optional health capability owned by the host
store bundle.

~~~go
type DependencyHealth interface {
    Name() string
    CheckHealth(ctx context.Context) error
}

type SQLHealth struct {
    name string
    db   *sql.DB
}

var _ DependencyHealth = (*SQLHealth)(nil)

func (h *SQLHealth) CheckHealth(ctx context.Context) error {
    return h.db.PingContext(ctx)
}
~~~

Deduplicate probes when several logical stores share one `*sql.DB`. A single
database handle should produce one bounded network round trip, not six
sequential pings.

### 5.4 Bound the readiness operation

Readiness itself must not hang the HTTP server.

~~~text
readiness(request):
  context timeout = 2 seconds
  check each unique required dependency, preferably in parallel
  collect safe component name and outcome
  return 200 only if every required dependency passes
  otherwise return 503
~~~

Do not return DSNs, SQL errors containing credentials, issuer response bodies,
client secrets, or schema contents. Log a redacted internal error separately
if operators need more detail.

### 5.5 Decide what to do with tiny-idp availability

OIDC discovery is loaded while building handlers, so startup already fails if
the issuer cannot be discovered. At runtime:

- Existing local sessions and app-owned access tokens do not require tiny-idp.
- Starting or completing browser login does require tiny-idp.
- Device approval requires a local session, but token polling does not call
  tiny-idp.

For the single-node profile, make SQL a hard readiness dependency. Treat issuer
availability as a separate degraded signal unless the product requires all
browser login operations to stop receiving traffic immediately. This avoids
evicting a healthy application merely because the IdP has a short outage.

### 5.6 Readiness tests

- A configured but unreachable PostgreSQL DSN prevents readiness.
- A database that becomes unavailable changes readiness from 200 to 503.
- Recovery changes readiness back to 200 without restarting the process.
- The response contains driver and component names but no DSN.
- A slow dependency is bounded by the readiness timeout.
- `/healthz` remains healthy during a simulated SQL outage.

## 6. Priority one: define account, agent, and credential lifecycle

### 6.1 Authentication creates two durable identities

After browser login, the application has a local user keyed by the verified
OIDC subject. After device approval, it also has an application agent whose
`OwnerUserID` refers to that local user.

~~~text
tiny-idp subject
      |
      v
local application user
      |
      +---- owns ----> programauth agent
                          |
                          +---- access tokens
                          +---- refresh-token families
~~~

The owner relationship must be enforced by every management query. A user must
never be able to enumerate, disable, or revoke another user's agents by
supplying an agent ID.

### 6.2 Define four distinct operations

Do not collapse these into one vague “logout” action:

| Operation | Effect |
|---|---|
| Browser logout | Deletes one local browser session and optionally ends the IdP session. |
| Revoke token family | Prevents further refresh from one installation; existing access tokens expire naturally. |
| Disable agent | Immediately rejects all access tokens because authentication reloads the agent; refresh also fails when it checks the disabled agent. |
| Disable application user | Rejects browser sessions and disables or rejects all owned agents according to explicit policy. |

The fourth operation needs a product decision. For a personal-agent product,
the safe default is to disable every owned agent when the local application
user is disabled. Tiny-idp account disablement is not automatically propagated
to the application today, so document whether propagation happens on next
browser login, through an administrative action, or through a future event or
back-channel mechanism.

### 6.3 Add an owner-scoped management surface

Provide session-only, CSRF-protected application routes for:

- listing the current user's agents;
- listing redacted credential families and last-used timestamps;
- revoking one refresh family;
- disabling one agent; and
- renaming an agent so the user can recognize it.

These can be planned Express routes backed by Go-owned auth services. The route
must derive the owner from `ctx.actor.id`; it must not accept `ownerUserId` from
the body.

~~~javascript
app.post("/api/me/agents/:agentId/disable")
  .auth(express.sessionUser())
  .csrf()
  .allow("agent.self.manage")
  .handle((ctx, res) => {
    // Service receives both ctx.actor.id and ctx.params.agentId.
    // It disables only when agent.OwnerUserID == ctx.actor.id.
  })
~~~

The service-side predicate is the security boundary. Hiding another user's
agent in the UI is not authorization.

### 6.4 Give disconnect an honest result

When a user revokes a refresh family, return or display:

~~~text
New access tokens: blocked
Current access token: may remain valid for at most 15 minutes
Immediate stop: disable the agent
~~~

If product requirements demand immediate per-installation disconnect without
disabling the entire agent, add access-token family revocation and check it on
every bearer authentication. Do not claim immediate revocation before that
check exists.

## 7. Priority one: finish operations and observability

### 7.1 A hook is not an exporter

PR 98 adds `SecurityEventObserver` and defaults to `MemorySecurityMetrics`.
That makes event production testable, but an in-memory counter that no
monitoring system reads is not production telemetry.

Add a production integration point that exports low-cardinality counters. Keep
the event keys bounded:

~~~text
name:    programauth.device.poll
outcome: rejected
reason:  slow_down
~~~

Never use user IDs, client names, action names, route parameters, IP addresses,
device codes, token prefixes, or raw error strings as metric labels.

### 7.2 Keep audit and metrics different

Metrics answer aggregate questions such as “Did rejected refresh attempts rise
sharply?” Audit records answer event questions such as “Which local actor
disabled this agent?”

| Property | Metrics | Audit |
|---|---|---|
| Cardinality | Strictly bounded | Higher, but controlled and indexed |
| Retention | Monitoring policy | Security/operations policy |
| User or agent identity | No | Redacted stable IDs when necessary |
| Raw credentials | Never | Never |

Keep the existing recursive redaction tests and add representative event tests
for every new endpoint.

### 7.3 Add cleanup and retention jobs

Durable auth tables grow even when correctness ignores expired rows. Define
operator-run cleanup for:

- expired OIDC login transactions;
- expired and consumed device authorizations;
- expired access tokens;
- old used/revoked refresh-token generations; and
- audit records beyond the approved retention window.

Cleanup must not delete a live refresh-family record needed for reuse detection
or incident investigation. Write the retention rule down before writing the
DELETE statement.

### 7.4 Treat migrations as release artifacts

The single-node profile correctly requires `apply-schema: false`. Complete the
contract by publishing ordered migration files and a repeatable migration
command. A production operator should be able to answer:

- Which schema version does this binary require?
- Which command upgrades the database?
- Can the migration run before the new pod starts?
- How is a failed migration detected and recovered?

Do not ask the serving process to discover and mutate its schema during
startup.

## 8. Correct the tiny-idp resource-server documentation

PR 98 includes a useful future design for accepting tiny-idp-issued device
tokens, but its current-state premise is stale. Current tiny-idp `main`
advertises:

~~~json
{
  "device_authorization_endpoint": ".../device_authorization",
  "introspection_endpoint": ".../introspect",
  "introspection_endpoint_auth_methods_supported": ["client_secret_basic"]
}
~~~

It also implements authenticated introspection with resource-client audience
checks and returns issuer, subject, client ID, scopes, audience, expiry, issued
time, and token type for active tokens.

Update
`reference/05-native-tinyidp-resource-server-contract.md` so it distinguishes:

- provider capabilities that already exist in tiny-idp; and
- the missing reusable go-go-goja adapter that converts a successful
  introspection response into `gojahttp.AuthResult` and local grants.

This correction changes the future work estimate. The IdP-owned path does not
need a new introspection protocol. It needs a go-go-goja resource-server
adapter, configuration, cache/revocation policy, and planned-route integration.

Do not add that adapter to PR 98 unless the PR scope is deliberately expanded.
The immediate requirement is to correct the document so the next implementer
starts from the actual provider contract.

## 9. Keep high availability as an explicit later profile

The single-node profile is compatible with a k3s Deployment using:

- `replicas: 1`;
- `strategy: Recreate` when using a local SQLite PVC;
- one durable PVC or one PostgreSQL database;
- pre-applied migrations; and
- Traefik TLS termination with explicit trusted-proxy policy.

Do not set `replicas: 2` merely because the auth records are in PostgreSQL. The
rate limiter is still process-local, and any other process-local coordination
must be audited first.

A future `production-ha` profile should require:

- PostgreSQL or another shared transactional store;
- a distributed rate limiter with atomic counters and expiry;
- shared or deterministic request identity semantics;
- multi-replica OIDC callback tests;
- concurrent device approval and token rotation tests; and
- rolling-deployment tests proving old and new schema compatibility when that
  compatibility is actually required by the release plan.

Do not add an adapter or compatibility layer preemptively. Define the supported
upgrade window when the HA deployment is designed.

## 10. Implementation sequence

The order below keeps each commit reviewable and makes failures attributable to
one boundary.

### Phase 1: request identity and trusted proxy

1. Add proxy configuration and validation.
2. Implement the canonical request identity resolver.
3. Store the resolved identity in request context.
4. Migrate rate limiting, audit, and access logging to the canonical result.
5. Add unit and Traefik integration tests.

Exit criterion: a forged forwarding header from an untrusted peer cannot change
audit or limiter identity, and a real Traefik request uses the same original
client address in both systems.

### Phase 2: native endpoint policy and budgets

1. Add `DeviceEndpointPolicy` and an action allowlist.
2. Remove caller control over the production verification URI.
3. Inject rate limiting into native device handlers.
4. Add request inspection and denial handlers.
5. Update the approval UI to display server-returned client, actions, and
   expiry.
6. Add negative and concurrency tests.

Exit criterion: every public native endpoint has a documented budget, every
issued grant is an allowed application action, and a user can inspect and deny
a pending request without exposing secrets.

### Phase 3: dependency-aware health

1. Add deduplicated SQL health capabilities to `StoreBundle`.
2. Implement bounded readiness checks and safe component results.
3. Keep process liveness separate.
4. Add outage and recovery tests.

Exit criterion: Kubernetes removes the pod from service when required SQL is
unavailable and restores it automatically after recovery, while the response
leaks no DSN or credential.

### Phase 4: owner lifecycle and operations

1. Add owner-scoped agent listing and disable operations.
2. Add refresh-family listing and revocation views.
3. Define local-user disable behavior.
4. Wire a real metrics observer.
5. Publish migrations, cleanup commands, and retention policy.

Exit criterion: a user can identify and disconnect their agents, an operator
can observe security outcomes without secrets, and database growth has a tested
maintenance procedure.

### Phase 5: documentation correction and deployment proof

1. Correct the tiny-idp introspection capability description.
2. Update the single-node runbook with proxy CIDRs, NetworkPolicy, migration,
   readiness, backup, and restore steps.
3. Run the strict tiny-idp smoke through the actual Traefik topology.
4. Capture exact validation commands and results in the PR diary.

Exit criterion: the documentation describes the shipped code, and the same
configuration shape used in k3s passes browser login, device approval, refresh,
revocation, restart, and negative security tests.

## 11. Validation matrix

### 11.1 Package tests

Run focused tests while implementing:

~~~bash
go test ./pkg/gojahttp/auth/programauth -count=1
go test ./pkg/gojahttp/auth/programauth/sqlstore -count=1
go test ./pkg/gojahttp/auth/keycloakauth/... -count=1
go test ./pkg/xgoja/hostauth -count=1
go test ./pkg/gojahttp -count=1
~~~

Run the race detector for the state-transition and limiter packages:

~~~bash
go test -race ./pkg/gojahttp/auth/programauth/... ./pkg/xgoja/hostauth ./pkg/gojahttp -count=1
~~~

Then follow the repository contract:

~~~bash
go fmt ./...
go test ./...
go build ./...
golangci-lint run -v
~~~

### 11.2 Protocol and security cases

| Area | Case | Required result |
|---|---|---|
| OIDC | callback state replay | exactly one callback succeeds |
| OIDC | restart between login and callback | callback succeeds with durable transaction |
| Device | unknown requested action | no device record; `invalid_scope` |
| Device | approval attempts broader actions | rejected or strict intersection; never broader |
| Device | concurrent redemption | exactly one token pair |
| Device | explicit denial | all later polls return `access_denied` |
| Device | rapid valid polling | RFC 8628 `slow_down` and increased interval |
| Device | start or unknown-code flood | HTTP 429 according to native budget |
| Refresh | successful rotation | old credential becomes used; new pair works |
| Refresh | reuse old credential | family revoked; no orphan access token |
| Revoke | unknown credential | non-oracle success response |
| Revoke | valid family | no future refresh; current access expires within bound |
| Agent | owner disables agent | all its access tokens fail immediately |
| Ownership | another user supplies agent ID | no information leak; operation denied |
| Proxy | forged XFF from direct peer | client identity remains direct peer |
| Proxy | request from trusted Traefik | audit and limiter agree on original client |
| Readiness | SQL outage | 503 within timeout; no secret in response |
| Readiness | SQL recovery | returns to 200 without restart |

### 11.3 Deployment smoke

Use `tmux` for every server process, capture panes for logs, and terminate test
servers deterministically. The full smoke should execute:

~~~text
1. Apply migrations.
2. Start tiny-idp with strict HTTPS issuer configuration.
3. Start one xgoja host behind the same proxy topology used in k3s.
4. Begin browser login.
5. Restart xgoja before callback and complete login.
6. Start a device request for an allowed action.
7. Inspect the request in the browser and approve it.
8. Poll and call an express.agent() route.
9. Rotate refresh credentials and reject reuse.
10. Revoke the family and verify bounded access-token behavior.
11. Disable the agent and verify immediate access denial.
12. Stop SQL and verify readiness becomes 503.
13. Restore SQL and verify readiness returns to 200.
14. Attempt forged forwarding headers and verify canonical IP behavior.
15. Scan captured logs and audit rows for credential material.
~~~

## 12. Review map for a new implementer

Read the code in this order. The order follows one request from configuration
to protocol transition.

1. `pkg/xgoja/hostauth/config.go` defines the configuration vocabulary.
2. `pkg/xgoja/hostauth/resolve.go` parses URLs, cookies, stores, and defaults.
3. `pkg/xgoja/hostauth/preflight.go` states what single-node production means.
4. `pkg/xgoja/hostauth/stores.go` builds and shares SQL handles.
5. `pkg/xgoja/hostauth/builder.go` composes services and mounts native paths.
6. `pkg/gojahttp/auth/programauth/device_handlers.go` owns the public device
   HTTP contract.
7. `pkg/gojahttp/auth/programauth/device.go` owns start, approval, denial,
   polling, expiry, and one-use consumption.
8. `pkg/gojahttp/auth/programauth/oauth_token.go` owns token pairs, refresh,
   reuse, and revocation semantics.
9. `pkg/gojahttp/auth/programauth/sqlstore/sqlstore.go` owns transactional SQL
   transitions.
10. `pkg/gojahttp/ratelimit.go` owns planned-route budgets and current client
    identity.
11. `pkg/gojahttp/auth/audit/audit.go` owns redaction and current forwarded-IP
    interpretation.
12. `pkg/xgoja/hostauth/readiness.go` shows why readiness is currently static.

For the future IdP-owned path, then read tiny-idp:

1. `internal/oidcmeta/discovery.go` for advertised endpoints.
2. `internal/fositeadapter/provider.go` for device authorization and
   introspection.
3. `cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go` for the existing
   application-specific introspection client pattern.

## 13. Common misunderstandings

### “The database is durable, so multi-replica is safe.”

False. The limiter remains local to one process, and request coordination must
be reviewed subsystem by subsystem. Shared SQL is necessary for HA but not
sufficient.

### “Secure cookies mean the backend must listen with TLS.”

False. A Secure cookie describes the browser-visible connection. Traefik may
terminate HTTPS and forward private HTTP, provided the public origin is
configured explicitly and the backend trusts forwarding information only from
known proxies.

### “RFC 8628 slow_down rate-limits the device API.”

False. It governs polling cadence for a known device authorization. It does not
bound device creation, unknown-code guessing, or refresh traffic.

### “Revoking a refresh token logs the agent out immediately.”

False. It prevents future access-token creation. Existing access tokens remain
valid until expiry unless the agent or access-token family is checked and
disabled separately.

### “Tiny-idp account disablement revokes application tokens.”

False in the app-owned model. The application must define how provider account
state reaches local users and owned agents.

### “The readiness JSON says ready, so the database is connected.”

False in the current PR. The report describes resolved configuration and does
not call `PingContext`.

## 14. Completion checklist

PR 98 or its immediate follow-up is ready for a public single-node xgoja
deployment only when all of the following are true:

- [ ] One canonical request identity resolver is used by audit, rate limiting,
      access logging, native handlers, and JavaScript projection.
- [ ] Forwarded headers are trusted only from configured Traefik addresses.
- [ ] NetworkPolicy prevents an unintended direct path to the pod.
- [ ] Every public native device endpoint has a bounded request budget.
- [ ] Device actions come from an application allowlist.
- [ ] The verification URI comes from trusted public-origin configuration.
- [ ] The approval UI displays server-returned client, actions, and expiry.
- [ ] A CSRF-protected denial path exists and is terminal.
- [ ] Readiness probes required SQL dependencies with a timeout and returns
      503 on failure.
- [ ] Liveness remains separate from dependency readiness.
- [ ] Users can list and disconnect only their own agents.
- [ ] Revocation UI states the remaining access-token lifetime accurately.
- [ ] A production metrics observer is wired, or metrics are explicitly
      declared unavailable rather than silently retained in memory.
- [ ] Cleanup, audit retention, backup, restore, and migrations have tested
      operator commands.
- [ ] The tiny-idp resource-server reference reflects current introspection
      support.
- [ ] The strict smoke passes through the real Traefik trust topology.
- [ ] `go test ./...`, `go build ./...`, lint, race-focused tests, and secret
      scans pass.

## 15. Standards and API references

- [RFC 8628](https://www.rfc-editor.org/rfc/rfc8628) defines device codes, user
  codes, verification, polling, `authorization_pending`, and `slow_down`.
- [RFC 7009](https://www.rfc-editor.org/rfc/rfc7009) defines OAuth token
  revocation and the non-oracle behavior expected for invalid credentials.
- [RFC 7662](https://www.rfc-editor.org/rfc/rfc7662) defines authenticated token
  introspection for the future tiny-idp-issued resource-server path.
- [RFC 8707](https://www.rfc-editor.org/rfc/rfc8707) defines resource indicators
  relevant to choosing and enforcing the intended API audience.
- PR 98's native application API is rooted at `/auth/device/*`; it is
  OAuth-shaped but remains an application-owned hostauth contract.

## Closing perspective

PR 98 establishes durable protocol state and honest single-node configuration.
The remaining work is not a second authentication system. It is the set of
boundaries that make the existing system safe to expose: trustworthy request
identity, bounded native endpoints, application-owned action policy, truthful
readiness, owner-scoped credential lifecycle, and observable operations.

Implement these boundaries in that order. Request identity comes first because
rate limiting and audit depend on it. Native policy comes next because public
device endpoints should not issue grants before their vocabulary and budgets
are fixed. Readiness follows because Kubernetes needs a truthful serving
signal. User lifecycle and operations complete the product behavior around the
protocol.

Once those properties are proven, a one-process xgoja host behind Traefik is a
coherent production target. Multi-replica hostauth and shared tiny-idp-issued
agent credentials can then be designed as explicit extensions rather than
assumptions hidden inside the first deployment.
