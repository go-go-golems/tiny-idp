---
Title: xgoja Device Authorization Durable Object API Analysis Design and Implementation Guide
Ticket: TINYIDP-XAPP-DEVICE-001
Status: active
Topics:
    - auth
    - oidc
    - oauth2
    - xgoja
    - durable-objects
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/routes/site.js
      Note: |-
        Current trusted JavaScript API routes and durable-object calls.
        Existing browser-only BBS routes and actor convention
    - Path: repo://cmd/tinyidp-xapp/development_app.go
      Note: |-
        Current Go host composition boundary for browser identity, xgoja, and durable objects.
        Go composition point for hostauth, durable objects, and embedded provider
    - Path: repo://cmd/tinyidp-xapp/state.go
      Note: |-
        Production state manifest, bootstrap, and owner-only secret lifecycle.
        Initialized state and secret lifecycle baseline
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        tiny-idp device authorization, token, and authenticated introspection endpoints.
        Existing device authorization and introspection provider endpoints
    - Path: repo://internal/oidcmeta/discovery.go
      Note: OIDC discovery metadata including the device authorization endpoint.
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc8628.html
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://openid.net/specs/openid-connect-discovery-1_0.html
Summary: Design and implementation plan for a complete xgoja durable-object API example where a CLI obtains a tiny-idp device token and accesses the same application through authenticated RFC 7662 introspection.
LastUpdated: 2026-07-16T15:20:00-04:00
WhatFor: Build and explain a production-shaped, end-to-end device authorization and resource-server example without exposing provider credentials to JavaScript.
WhenToUse: When extending tinyidp-xapp with programmatic access or extracting reusable xgoja resource-server primitives.
---


# xgoja Device Authorization Durable Object API

## Executive summary

`tinyidp-xapp` is already a useful self-contained vertical slice: one Go
process hosts an embedded tiny-idp issuer, an xgoja Express application, and
durable objects. Browser requests authenticate through authorization-code OIDC
and xgoja's `hostauth` service. The application route then receives an actor
from Go context and calls a durable object with that actor identity. This is a
good security boundary for browser traffic, but it does not let a non-browser
client use the application API.

This ticket extends that vertical slice with a deliberately separate API
authentication path. A CLI uses OAuth 2.0 Device Authorization Grant to obtain
an opaque bearer access token for a named application API audience. The Go host
validates each bearer token through tiny-idp's authenticated RFC 7662
introspection endpoint, checks issuer, token type, audience, expiry, and
route-required scope, then attaches a small verified principal to the request.
The xgoja route receives the same kind of actor information that it already
uses for browser sessions; it never receives a bearer token or an
introspection-client secret.

The first implementation is intentionally local to `cmd/tinyidp-xapp`. It is
an executable reference, not a premature general-purpose xgoja package. Its
configuration, tests, error model, and seams are designed so the stable parts
can later move to `go-go-goja/pkg/xgoja/oidcresource` with evidence from a real
application.

```text
                    +---------------------------+
                    | tinyidp-xapp Go host       |
                    |                           |
browser --------->  | /auth/* -> hostauth        |
                    |                 |         |
                    |                 v         |
                    | xgoja route -> actor ----+|----> Durable Object
                    |                           ||        BBS/community
CLI -- device flow  | /api/device/* -> resource ||
  |                 | bearer middleware          ||
  |                 |       |                    ||
  v                 +-------|--------------------+|
tiny-idp /idp               | RFC 7662 Basic       |
  device endpoint           v                     |
  token endpoint       /idp/introspect <----------+
```

## Reader prerequisites and terminology

An intern should distinguish four identities which are often incorrectly
collapsed into one term:

| Term | Created by | Used for | Example in this design |
| --- | --- | --- | --- |
| End user / subject | tiny-idp account store | who completed login | `dev-alice-subject` |
| OAuth client | tiny-idp client registration | software asking for a token | `tinyidp-xapp-cli` |
| Resource server client | tiny-idp client registration | confidential caller of `/introspect` | `tinyidp-xapp-api` |
| Application actor | Go host authentication adapter | route and object ownership | `actor.id == subject` |

The device CLI is *not* the resource server. It is a public OAuth client and
therefore has no confidential secret. The API is a confidential resource-server
client and owns an introspection secret. The API secret is stored in the xapp
state root and is only read by Go. A durable object is application state and
must never decide whether a raw OAuth token is valid.

## Scope, non-goals, and acceptance criteria

### In scope

- Register a device client and an audience-bound confidential resource client
  when development or initialized xapp state is bootstrapped.
- Provide a CLI command that starts RFC 8628 device authorization, displays
  verification instructions, polls under the server-provided interval, and
  stores an expiring token in an owner-only local cache.
- Provide authenticated API routes that read and create BBS posts through the
  existing shared durable object.
- Validate opaque bearer tokens with RFC 7662 over the existing in-process
  transport in development and through the actual TLS server in production.
- Keep browser session routes unchanged and keep their CSRF protection.
- Add unit, integration, CLI, and browser regression tests with explicit
  negative-security cases.

### Explicit non-goals

- General token exchange, refresh-token persistence, multi-tenant policy,
  remote durable-object dispatch, and a JavaScript-accessible identity-provider
  administration API.
- Treating a device token as an application `programauth` token. They are
  independently issued credentials with different lifecycle owners.
- Making opaque tokens self-validating. The provider remains the revocation
  authority.
- Publishing a reusable `go-go-goja` library before this application proves
  the minimal API. The later extraction is a follow-up, not compatibility work
  required by this ticket.

### Acceptance criteria

1. A seeded operator can run one command to begin device login, approve it in
   the browser, and run one command to post a second BBS message.
2. The resulting message's author is the approver's tiny-idp subject/display
   name, not a caller supplied CLI field.
3. Missing, malformed, expired, revoked, wrong-issuer, wrong-audience, and
   insufficient-scope bearer credentials fail before JavaScript or a durable
   object is called.
4. The resource-client secret and raw bearer token do not appear in JS,
   frontend assets, normal logs, audit values, diagnostics, or cache keys.
5. Browser BBS behavior remains covered and does not accidentally start
   accepting bearer tokens on cookie/CSRF routes.

## Current architecture: evidence-based walkthrough

### Process and routing composition

`cmd/tinyidp-xapp/main.go` is a Glazed/Cobra command root. It exposes
development `serve`, persistent `init`, production-shaped `serve-initialized`,
and `doctor` commands. `serve-initialized.go` adds `/healthz` and `/readyz` and
wraps the application handler with a request-body size limit; the application
itself owns the security-sensitive route composition.

`NewDevelopmentApplication` in `development_app.go` creates an SQLite identity
store, password account service, embedded provider, xgoja `hostauth` service,
durable-object server, and generated xgoja runtime. `composeApplication`
creates the top-level mux in this order:

```go
mux.Handle("/idp/", app.idp.Handler())
mux.Handle("GET /static/tinyidp/", app.loginUI.AssetsHandler())
for _, native := range app.auth.NativeHandlers {
    mux.Handle(native.Method+" "+native.Path, native.Handler)
}
mux.Handle("/", httpHost)
```

That order matters. Provider endpoints are host-owned; native hostauth paths
are intercepted before the xgoja HTTP host; only trusted generated routes run
under `httpHost`. `gojahttp.NewHost` is constructed with
`RejectRawRoutes: true`, so application scripts cannot bypass the generated
route/middleware model with arbitrary net/http registration.

`NewInitializedApplication` in `production_app.go` reopens the initialized
SQLite state, owner-only token secret, audit sink, and object storage. It uses
production embedded-idp mode, secure cookies, a rate limiter, and an initial
maintenance run. `state.go` writes a versioned manifest with a public base URL,
issuer, and browser client ID. At present that manifest has no representation
for API audience, device client, or resource-client secret; those are the
specific state-model gaps this ticket closes.

### Browser authentication and the actor boundary

The host's existing browser client is registered using
`embeddedidp.BrowserClient` in `development_app.go` and `state.go`. The client
uses authorization code with PKCE through xgoja `hostauth`. It receives its
issuer traffic through `embeddedidp.NewInProcessIssuerTransport` in
development/embedded operation. The transport makes a single-host demo work
without pretending that browser and issuer traffic are untrusted network
calls; production validation still exercises actual HTTPS.

In `composeApplication`, a `durableobjects.BoundDispatcherService` resolves
the actor from `gojahttp.ActorFromContext`. Its `ActorID` function returns an
error when the actor is absent. The service therefore establishes a valuable
invariant: `objects.fetchForActor("USER_STATE", ...)` cannot pick an arbitrary
object key from JavaScript; the Go host derives it from authenticated context.

The route file `app/routes/site.js` illustrates the convention. Browser BBS
routes call `express.user().required()`, declare an application permission with
`.allow(...)`, emit an audit event, and add `.csrf()` for mutations. `fetchBoard`
passes `ctx.actor.id` and a normalized display name to BBS. The `BBS` durable
object in `app/objects/objects.js` treats these fields as *trusted host input*;
all user-controlled message fields are independently length/type validated.

### Provider capabilities already available

tiny-idp already exposes the protocol components needed here:

- `internal/fositeadapter/provider.go` mounts `/device_authorization`,
  `/device`, `/token`, and authenticated `/introspect`.
- `internal/oidcmeta/discovery.go` advertises the device authorization endpoint
  and device-code grant in discovery metadata.
- RFC 7662 introspection accepts only a confidential client with explicit
  introspection capability. An active result is constrained to issuer,
  subject, client ID, scope, audience, expiry, issue time, and Bearer token
  type.
- Resource indicators are represented by per-client allowed audiences and are
  enforced at issuance/introspection. The prior contract is recorded in
  `TINYIDP-INTROSPECTION-001/reference/03-xgoja-oidcresource-consumer-contract.md`.

The implementation does not need to invent device authorization or decode
opaque tokens. Its responsibility is correct application composition.

## Gap analysis

| Required capability | Present today | Gap | Ticket response |
| --- | --- | --- | --- |
| Browser OIDC login | Yes, through `hostauth` | Cookie actor only | Preserve unchanged |
| Device authorization endpoint | Yes, in tiny-idp | No xapp device client or CLI | Bootstrap and CLI module |
| Resource indicators | Yes, in tiny-idp | No xapp API audience | Register one exact API audience |
| Authenticated introspection | Yes, in tiny-idp | No xapp client/middleware | Go-only resource authenticator |
| Durable-object BBS | Yes | Only browser actor can reach it | Principal-to-actor adapter |
| Production state root | Yes | No API client secret/device metadata | Version state manifest and paths |
| API regressions | Browser tests exist | No device/API security matrix | Add table and end-to-end tests |

## Proposed solution

### Identity and client registrations

The xapp has three registrations. Each has exactly one role.

```text
browser client:  tinyidp-xapp
  redirect URI:  https://app.example/auth/callback
  scopes:        openid profile email

device client:   tinyidp-xapp-cli
  public:         yes; no secret
  grant:          urn:ietf:params:oauth:grant-type:device_code
  scopes:         bbs.read bbs.post.create
  audience:       https://app.example/api

resource client: tinyidp-xapp-api
  confidential:   BCrypt-hashed generated secret
  capability:     CanIntrospect
  audience:       https://app.example/api
```

The exact audience is derived from the canonical public base URL as
`<public-base-url>/api`. It is an identifier, not a routing prefix permission.
The resource server requires exact equality with a value in the introspection
`aud` array. The device client and browser client remain separate even if their
scopes overlap: browser redirect security and device approval security differ.

### State and configuration model

The initialized state must be explicit, reproducible, and owner-only. Add
fields to `StateManifest` for the device and resource client IDs plus the API
audience. Add a `ResourceClientSecret` path under `secrets/`, generated once
with the same permissions as the existing token/binding keys. The IDP database
stores only a BCrypt hash through normal bootstrap/client persistence; the file
stores the usable secret for this single-process resource server.

```go
type StateManifest struct {
    Version          int
    PublicBaseURL    string
    Issuer           string
    ClientID         string // Browser OAuth client
    DeviceClientID   string
    ResourceClientID string
    ResourceAudience string
    CreatedAt        time.Time
}

type ResourceAuthConfig struct {
    IssuerURL        string
    ClientID         string
    ClientSecret     []byte       // Go only; zero after setup where possible
    Audience         string
    CacheMaxAge      time.Duration // max 30 seconds
    RequiredScopes   []string
}
```

The state version must increase. This is a purposeful incompatible product
state change: old initialized state is rejected with a clear re-initialization
or migration message. We will not add a silent backwards-compatibility adapter
because existing secret/client state cannot prove it has the new audience and
introspection capability.

### Resource authentication pipeline

The resource middleware lives in Go, near xapp composition. It accepts a
request's `Authorization` header and returns one of three classes:

| Result | HTTP response | Meaning |
| --- | --- | --- |
| unauthenticated | `401` plus `WWW-Authenticate: Bearer` | client supplied no usable bearer token |
| forbidden | `403` | token is valid but lacks the route's scope/object authority |
| unavailable | `503` | provider could not make a trustworthy decision |

The implementation must not report whether a malformed, revoked, expired, or
foreign token was the reason for `401`. The response body can contain a stable
machine-readable code such as `unauthorized`, while structured server audit
events name only a failure category—not token bytes.

```text
requireBearer(request, routeScopes):
    token = parseSingleBearerHeader(request.Authorization)
    if token absent or malformed:
        return 401

    decision = positiveCache.get(hmac(cacheKey, token))
    if decision absent:
        decision = introspectWithBasicOverBoundedTransport(token)
        if provider unavailable or invalid resource client:
            audit("api.auth.unavailable", category)
            return 503
        if decision inactive:
            return 401
        cache until min(decision.exp, now + 30 seconds)

    if decision.issuer != config.issuer or
       decision.tokenType != "Bearer" or
       config.audience not in decision.audience or
       now >= decision.exp:
        return 401

    principal = { subject: decision.sub, clientID: decision.clientID,
                  scopes: splitASCIIWhitespace(decision.scope),
                  expiresAt: decision.exp, kind: "oidc_bearer" }
    if !principal.scopes contains every routeScopes:
        return 403
    attach principal as actor to request context
    continue
```

The cache is an availability/performance optimization, never an indefinite
revocation bypass. It uses an HMAC digest as its in-memory key. Positive cache
entries expire no later than `exp` and no later than the configured short
maximum (30 seconds). Negative results may be cached only a few seconds after
we have a syntactically valid definitive inactive response. Transport failures,
429, and invalid resource client responses are not token decisions and are not
cached.

### Principal adapter and durable-object authorization

The existing BBS expects `ctx.actor`. Browser actor and bearer principal must
be normalized at one host boundary, rather than teaching every JS route about
two authentication mechanisms. Define an adapter that writes a `gojahttp`
actor equivalent using:

```text
actor.id       = principal.subject
actor.kind     = "oidc_bearer"
actor.claims   = {}    // no attempt to synthesize profile claims
actor.scopes   = principal.scopes, if the host representation supports it
```

Routes that accept bearer API access must use a dedicated native/host middleware
or an xgoja module bridge. They do **not** add `.csrf()` because a bearer token
sent by an explicit CLI `Authorization` header is not ambient browser
credential state. Conversely, existing cookie routes keep `.csrf()` and do not
become dual-mode implicitly.

The initial API contract is intentionally narrow:

```http
GET /api/device/bbs
Authorization: Bearer <opaque-token>
Accept: application/json

POST /api/device/bbs/posts
Authorization: Bearer <opaque-token>
Content-Type: application/json

{"title":"From the terminal","body":"approved through device login","category":"notes"}
```

Both handlers call the existing BBS durable object through a Go-controlled
adapter that passes `sub` and a display representation. The first version uses
the subject as its author label, because RFC 7662 intentionally does not expose
profile claims. A later product could resolve a local profile keyed by `sub`.
It must not call `/userinfo` on every API request merely to render a name.

### CLI command and local token cache

The CLI belongs in `cmd/tinyidp-xapp` to make the example one binary. Its
commands use Glazed, follow existing logging behavior, and receive all values
via flags. No environment variables are introduced.

```text
tinyidp-xapp device-login \
  --issuer https://localhost:9443/idp \
  --client-id tinyidp-xapp-cli \
  --audience https://localhost:9443/api \
  --token-cache ~/.local/.../tinyidp-xapp-device.json

tinyidp-xapp bbs get --api-base-url https://localhost:9443 \
  --token-cache ...

tinyidp-xapp bbs post --title ... --body ... --category notes \
  --api-base-url https://localhost:9443 --token-cache ...
```

`device-login` discovers metadata from the issuer, sends a form POST to
`device_authorization_endpoint` with `client_id`, requested scopes, and
`resource`, prints `verification_uri_complete` when available, and polls the
token endpoint no faster than the advertised `interval`. It increases its
interval after `slow_down`, stops at `expires_in`, and only writes an
owner-only token cache after a complete successful token response. The cache
contains a token and an expiry; it is not sent to the frontend and uses mode
0600. API commands reject missing/expired cache entries locally before making
a request.

### Observability

The existing IDP audit sink records protocol events. The API host will add
high-value, redacted events at its boundary:

- `xapp.api.auth.accepted`: subject hash/identifier policy, client ID, route,
  audience match, scopes requested; never token.
- `xapp.api.auth.rejected`: route and stable category (`missing_bearer`,
  `inactive`, `wrong_audience`, `missing_scope`); no raw provider detail.
- `xapp.api.auth.unavailable`: route and provider failure class.
- `xapp.api.bbs.posted`: subject and object operation, with the same semantics
  as browser route audit rather than duplicate secrets.

Counters may distinguish cache hit/miss and response category. Their labels
must not include subject, token, URL query, or body values. This is enough to
detect a provider outage or authorization regression without building a token
tracking database.

## API reference

### Discovery inputs

The CLI uses OIDC discovery at:

```text
{issuer}/.well-known/openid-configuration
```

It validates exact `issuer`, HTTPS endpoints outside explicit local-test
transport, `device_authorization_endpoint`, `token_endpoint`, and advertised
device-code grant support. This follows OpenID Connect Discovery; it does not
use the OAuth authorization-server discovery path form by accident.

### Device authorization request

```http
POST /idp/device_authorization
Content-Type: application/x-www-form-urlencoded

client_id=tinyidp-xapp-cli&scope=bbs.read%20bbs.post.create&resource=https%3A%2F%2Fapp.example%2Fapi
```

Successful response fields used: `device_code`, `user_code`,
`verification_uri`, optional `verification_uri_complete`, `expires_in`, and
`interval`. `device_code` is secret and only exists in process memory while
polling. `user_code` can be displayed; it is intended for human entry.

### Token polling request

```http
POST /idp/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=<secret>&client_id=tinyidp-xapp-cli
```

`authorization_pending` means continue after the current interval;
`slow_down` means add at least five seconds; `access_denied`, `expired_token`,
or any terminal error fails without cache write.

### Introspection request

```http
POST /idp/introspect
Authorization: Basic base64(tinyidp-xapp-api:<owner-only-secret>)
Content-Type: application/x-www-form-urlencoded

token=<opaque-bearer-token>
```

Only the Go host makes this request. The result schema and validation rules are
normative in the earlier `oidcresource` consumer contract.

## Design decisions

### ADR-1: host-owned resource authentication

- **Context:** JavaScript routes and frontend assets are application code, but
  introspection credentials are deployment secrets.
- **Options:** expose `/introspect` to JavaScript; decode token locally; Go
  host middleware with a constrained principal bridge.
- **Decision:** Go host middleware and constrained bridge.
- **Rationale:** opaque tokens cannot be safely decoded; Basic credentials
  cannot be trusted to JS; one Go policy point is testable and auditable.
- **Consequences:** the first implementation has a small amount of native Go
  routing. That is desirable: the security boundary is visible.
- **Status:** accepted.

### ADR-2: separate browser and bearer route families

- **Context:** browser cookies need CSRF defense; explicit bearer API requests
  have a different threat model.
- **Options:** one dual-mode route; separate `/api/device/*` native endpoints;
  convert all routes to bearer endpoints.
- **Decision:** separate bearer API endpoints sharing the BBS object service.
- **Rationale:** avoids a subtle route accepting the wrong credential type and
  permits focused API contracts/tests.
- **Consequences:** limited endpoint duplication is accepted until xgoja gets a
  proven reusable route-auth abstraction.
- **Status:** accepted.

### ADR-3: subject is the actor key

- **Context:** durable object ownership requires a stable identifier. Profile
  display claims are optional and not part of the introspection response.
- **Options:** user-supplied CLI author; call userinfo; use `sub`.
- **Decision:** use `sub`, with subject-derived first-version display output.
- **Rationale:** prevents impersonation and avoids an additional online
  identity dependency on every API call.
- **Consequences:** API-authored messages may display a stable subject rather
  than browser profile name until an application profile resolver is designed.
- **Status:** accepted.

### ADR-4: state migration is explicit

- **Context:** initialized xapp state lacks resource-client data and secret.
- **Options:** silently infer/default; auto-edit old state; fail clearly and
  provide an explicit migration/reinitialization command.
- **Decision:** versioned explicit migration path.
- **Rationale:** silently inventing secret/client registrations makes a
  security migration unreproducible.
- **Consequences:** operators must perform a documented state upgrade.
- **Status:** proposed; finalize after inspecting production init lifecycle.

## Alternatives considered

### Put the API entirely in JavaScript

Rejected. It would require a JavaScript module to hold a resource-client
secret, and one accidental bundle/export could disclose it. It also turns
transport timeout, cache, and redaction behavior into unreviewable script
configuration rather than host policy.

### Access tiny-idp SQLite directly

Rejected. It bypasses expiry/revocation/provider authorization semantics,
couples application state to provider schema, and makes a future remote IDP
deployment impossible without an application rewrite.

### Use `/userinfo` as a token validator

Rejected. Userinfo has a distinct OpenID Connect purpose and response shape.
RFC 7662 is the endpoint whose contract expresses whether an opaque access
token is active for this resource audience.

### Use application `programauth` credentials

Rejected for this example. `programauth` can be useful for a service-owned
automation identity, but it cannot demonstrate the user-approved device flow
or provider token revocation. The application may later accept both only under
an explicit principal-kind and authorization policy.

### Extract an xgoja library before implementing the example

Deferred. The cross-repository design work identified desirable primitives,
but a single real consumer should establish the exact Go/xgoja actor bridge,
error semantics, and tests before freezing a public API.

## Phased implementation plan

The companion `tasks.md` is the execution ledger. The phases below explain the
dependency graph; do not skip an earlier security boundary merely to make a
demo command run.

```text
P0 inventory/contract
   -> P1 resource authentication core
      -> P2 xapp state and IDP registrations
         -> P3 native bearer API and object adapter
            -> P4 device CLI
               -> P5 verification and Playwright regression
                  -> P6 operator handoff/extraction decision
```

### Phase 0 — establish contract and baseline

Read the existing xapp code and previous introspection contract, record the
route/object invariant, create an executable task ledger, and identify the
current test baseline. No runtime behavior changes occur here.

### Phase 1 — resource authentication core

Implement a package-local `resourceauth` component with immutable config,
discovery validation, Basic-authenticated introspection, strict response
validation, bounded cache, redacted errors/events, and a `Principal` result.
Tests use `httptest` to prove every validation branch independently of xgoja.

### Phase 2 — xapp state and provider registrations

Add device/resource client constants, exact API audience derivation, generated
owner-only resource secret, bootstrap registrations, state validation, and
production/development construction. Tests prove state consistency and client
capabilities without placing secrets in manifests.

### Phase 3 — bearer API routes and durable-object bridge

Add host-owned `/api/device/bbs` endpoints. They invoke Phase 1, enforce each
scope, translate the principal to trusted BBS actor input, and preserve object
ownership rules. Add application-level tests that prove rejected requests do
not call the object and accepted posts use the token subject.

### Phase 4 — device CLI

Implement Glazed commands for login, BBS read, and BBS post. Add owner-only
token-cache handling, discovery/poll semantics, concise human output, and
machine-readable error behavior. Unit-test terminal polling cases with a fake
provider.

### Phase 5 — full-system verification

Build a deterministic test harness that starts xapp in tmux, runs real device
authorization, completes the approval using browser automation, invokes the
CLI against the API, and asserts a second BBS message from the device subject.
Run Playwright browser regressions to ensure regular login and CSRF routes
still work. Add negative matrix cases for revoked/wrong-audience/scope tokens.

### Phase 6 — documentation, operations, and extraction recommendation

Publish runbooks, threat model, test scripts, and results. Compare the proven
local component against the cross-repository xgoja design ticket and write a
specific extraction proposal only for interfaces now exercised by tests.

## Test strategy

| Layer | Test target | Required assertions |
| --- | --- | --- |
| Unit | `resourceauth` parser/client | exact bearer grammar, issuer/audience/expiry/type, cache TTL, no token in errors |
| Unit | state bootstrap | unique clients, secret mode, exact audience, manifest validation |
| Integration | embedded IDP + API handler | active device token reaches BBS; inactive/foreign tokens do not |
| CLI | fake discovery/device/token server | interval/slow_down/expiry/cache permissions/terminal errors |
| E2E | actual xapp + browser | approval in browser then CLI get/post; message author changes across users |
| Regression | Playwright browser app | browser login/logout/CSRF/BBS still work |
| Production smoke | initialized TLS server | discovery, device flow, introspection and API use production transport |

Negative cases are as important as the happy path. A route test must inspect
the durable-object call counter, not merely receive a `401`, so it proves
failure occurred at the intended boundary. CLI test fixtures must check that
device codes and access tokens never appear in captured logging/output except
where the user intentionally receives an access-token cache file.

## Security review checklist

- [ ] Resource client uses a distinct ID from browser/device clients.
- [ ] Resource client secret is 0600, omitted from manifests, frontend, and logs.
- [ ] Bearer parser rejects multiple/ambiguous headers and non-Bearer schemes.
- [ ] Introspection validates `active`, exact issuer, token type, audience,
  nonempty subject, numeric unexpired expiry, and every route scope.
- [ ] Provider and transport failure fail closed as `503`; stale cache lifetime
  is bounded and never exceeds token expiry.
- [ ] API endpoints do not accept cookie session as a fallback, and browser
  mutation endpoints still require CSRF.
- [ ] Durable object receives actor identity only from a host-verified
  principal/session; CLI body cannot choose author/owner.
- [ ] Tests exercise post-issuance revocation and wrong-audience denial.

## Risks and open questions

1. **xgoja host actor API:** the exact public method for attaching a native
   bearer principal to `gojahttp` context may not be exported by the current
   dependency. If not, keep native endpoints in Go for this ticket rather than
   adding a reflection/adaptor layer; record the needed upstream primitive.
2. **Initialized-state migration:** current state version is one. We must
   decide whether a one-time `migrate` command is needed or whether this
   example's unreleased state can require clean initialization. The decision is
   security-sensitive and cannot be hidden behind defaults.
3. **Device approval automation:** RFC 8628 requires real user interaction.
   The E2E harness may use a seeded test user and browser automation, but it
   must not create a protocol backdoor to auto-approve a device code.
4. **Display names:** using a subject is secure but not ideal presentation.
   A later profile mapping must have its own consistency and privacy design.
5. **Reusable extraction:** success here does not automatically mean the
   package belongs in `go-go-goja`; first gather the actual host interfaces and
   at least one second consumer need.

## Intern implementation map

Start with these files in order:

1. `cmd/tinyidp-xapp/development_app.go`: dependency composition and current
   browser actor binding.
2. `cmd/tinyidp-xapp/state.go`: persistent state invariants and bootstrap.
3. `cmd/tinyidp-xapp/app/routes/site.js`: browser route authorization and BBS
   object invocation conventions.
4. `cmd/tinyidp-xapp/app/objects/objects.js`: data validation and author/owner
   enforcement inside durable-object state.
5. `internal/fositeadapter/provider.go`: device, token, and introspection
   endpoint contracts already provided by tiny-idp.
6. `TINYIDP-INTROSPECTION-001/reference/03-xgoja-oidcresource-consumer-contract.md`:
   normative resource-server requirements.

Use `go test ./cmd/tinyidp-xapp/... -count=1` before widening scope. Keep
new executable harnesses under this ticket's `scripts/` directory and record
their inputs/outputs (without credentials) in the diary. Commit code and its
diary/changelog update separately so future reviewers can map a behavioral
change to evidence.

## References

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/rfc/rfc8628.html)
- [RFC 7662: OAuth 2.0 Token Introspection](https://www.rfc-editor.org/rfc/rfc7662.html)
- [RFC 8707: Resource Indicators for OAuth 2.0](https://www.rfc-editor.org/rfc/rfc8707.html)
- [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html)
- `internal/fositeadapter/provider.go`
- `internal/oidcmeta/discovery.go`
- `pkg/embeddedidp/provider.go`
- `cmd/tinyidp-xapp/development_app.go`
- `cmd/tinyidp-xapp/production_app.go`
- `cmd/tinyidp-xapp/state.go`
- `cmd/tinyidp-xapp/app/routes/site.js`
- `cmd/tinyidp-xapp/app/objects/objects.js`
- `ttmp/2026/07/15/TINYIDP-INTROSPECTION-001--authenticated-oauth-token-introspection-and-resource-indicators/reference/03-xgoja-oidcresource-consumer-contract.md`
