---
Title: Self-Contained xgoja tiny-idp Express and Durable Objects Analysis Design and Implementation Guide
Ticket: TINYIDP-XAPP-001
Status: active
Topics:
    - architecture
    - auth
    - go
    - identity
    - oidc
    - research
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp-xapp/app/objects/objects.js
      Note: Bounded USER_STATE Durable Object implementation
    - Path: repo://cmd/tinyidp-xapp/app/routes/site.js
      Note: Trusted planned routes for static frontend session identity and actor-bound object access
    - Path: repo://cmd/tinyidp-xapp/internal/xgojaruntime/xgoja_runtime.gen.go
      Note: Generated importable runtime package and HostServices injection seam
    - Path: repo://cmd/tinyidp-xapp/main.go
      Note: Glazed lifecycle-host command root and structured logging boundary
    - Path: repo://cmd/tinyidp-xapp/xgoja.yaml
      Note: Generated runtime package providers sources commands declarations and embedded assets
    - Path: repo://pkg/embeddedidp/options.go
      Note: Defines the embedded identity provider configuration boundary used by the custom host
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Constructs the path-prefixed OIDC handler and exposes readiness maintenance and shutdown
    - Path: ws:///go-go-goja/pkg/gojahttp/auth/oidcauth/inprocess_transport.go
      Note: Fail-closed same-process discovery token and JWKS transport
    - Path: ws:///go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go
      Note: Provider-neutral OIDC relying-party handlers and injectable back-channel client
    - Path: ws:///go-go-goja/pkg/gojahttp/planned_dispatch.go
      Note: Carries the authenticated actor into trusted native route services
    - Path: ws:///go-go-goja/pkg/xgoja/hostauth/builder.go
      Note: Builds app sessions native auth routes and issuer-scoped application users
    - Path: ws:///go-go-objects/pkg/durableobjects/bound_dispatcher.go
      Note: Derives HMAC actor-bound object IDs and rejects caller-selected IDs
    - Path: ws:///go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go
      Note: Exposes actor-bound xgoja calls and default-denied raw gateways
ExternalSources: []
Summary: Product assessment and cross-repository design for a self-contained xgoja-generated application that embeds tiny-idp, serves an Express-style JavaScript API and frontend, and binds every authenticated user to private SQLite-backed durable objects.
LastUpdated: 2026-07-12T02:45:00Z
WhatFor: Defines whether the current components are usable, how they should be composed, which APIs must change across tiny-idp/go-go-goja/go-go-objects, and how an intern can implement and validate the complete browser login-to-durable-state product.
WhenToUse: Read before implementing the self-contained app, adding a tiny-idp xgoja provider, changing OIDC hostauth, exposing durable objects to authenticated routes, or generating the product binary.
---



# Self-Contained xgoja Identity and Durable Object Application

## Executive summary

The three projects are usable building blocks today. They are not yet a usable
self-contained product when combined without additional integration work.

tiny-idp has a real embeddable provider, SQLite persistence, production-mode
preflight, key lifecycle, password hardening, server-owned authorization
interactions, maintenance, readiness, and extensive tests. go-go-goja has a
single-owner Goja runtime, an Express-style HTTP host, planned authentication and
authorization, OIDC relying-party handlers, server-side application sessions,
xgoja providers, generated runtime packages, embedded assets, and working
generated-host examples. go-go-objects has per-object Goja actors, serialized
dispatch, SQLite-backed object storage, Promise-aware RPC/fetch, alarms, idle
eviction, an HTTP gateway, and an xgoja provider.

All three complete `go test ./... -count=1` successfully in the shared workspace.
That is strong implementation evidence. It does not solve product composition.

The proposed first product is a single-process, single-node, self-hosted personal
application:

- the public origin is `https://app.example.test`;
- tiny-idp is the OIDC issuer at `https://app.example.test/idp`;
- the same process is an OIDC relying party with callback `/auth/callback`;
- a separate opaque application session authenticates Express planned routes;
- each application user maps deterministically to one private durable object;
- JavaScript declares trusted routes and durable-object behavior;
- HTML, CSS, and browser JavaScript are embedded into the generated binary;
- SQLite state and secrets live under one application state root; and
- one Go host owns startup, the outer `http.ServeMux`, maintenance, background
  loops, readiness, server shutdown, and resource closure.

```text
browser
  |
  v
single public origin / outer Go ServeMux
  |
  +-- /idp/* --------> embedded tiny-idp OIDC provider
  +-- /auth/* -------> native OIDC RP + application-session handlers
  +-- /static/* -----> embedded frontend assets
  +-- /api/* --------> gojahttp planned routes -> subject-bound durable objects
  +-- / -------------> embedded index.html
```

The recommended implementation is a hybrid xgoja package host, not a pure
generated binary and not a direct JavaScript embedding of tiny-idp.

xgoja should generate a reusable runtime package containing provider registration,
module selection, TypeScript declarations, JavaScript route sources, the durable
object bundle, and frontend assets. A small handwritten Go `cmd/app` imports that
generated package and constructs the identity, session, durable-state, HTTP, and
lifecycle services. This is already an intended xgoja pattern, demonstrated by
`examples/xgoja/14-generated-runtime-package`.

After the host contract works, tiny-idp can add an xgoja provider package. That
provider should primarily contribute host services, configuration sections,
mount hooks, health, and closers. It should not expose password verification,
stores, signing keys, token issuance, or raw provider objects to JavaScript.

Three cross-repository changes are necessary:

1. **go-go-goja:** generalize the `keycloakauth` package name and API into a
   provider-neutral OIDC RP adapter, and allow an injected `*http.Client` or
   `RoundTripper` for in-process discovery, JWKS, and token exchange.
2. **go-go-objects:** add a subject-bound dispatch service or provider helper that
   derives object identity from the authenticated actor. Do not publicly mount
   the raw `/fetch/:namespace/:name` gateway for private user objects.
3. **tiny-idp:** add an xgoja provider/host-service package only after the custom
   host proves lifecycle and configuration; add cookie-path configuration and a
   narrow in-process OIDC transport helper if it belongs with the issuer.

The result is suitable as a personal application, internal tool, local-first
service, or controlled single-node deployment. It is not initially a hostile
multi-tenant JavaScript platform or a distributed Durable Objects system.

## 1. Product definition

The first application is intentionally concrete. A user opens the embedded
frontend, logs in through tiny-idp, and receives an application session. The
frontend calls authenticated API routes. Those routes access one durable object
whose identity is derived from the authenticated subject. The object stores and
updates the user's application state across process restarts.

### 1.1 User-visible behavior

1. `GET /` returns the embedded application shell.
2. The frontend calls `GET /auth/session`.
3. An unauthenticated user selects Login.
4. `GET /auth/login` creates state, nonce, and PKCE verifier, then redirects to
   `/idp/authorize`.
5. tiny-idp validates the RP client and renders its login/consent interaction.
6. After authentication, tiny-idp returns an authorization code to
   `/auth/callback`.
7. The native RP handler exchanges the code, verifies the ID token, checks nonce,
   normalizes the subject, and creates an opaque application session.
8. `GET /api/me` returns the app actor projection.
9. `GET /api/object` and `POST /api/object` dispatch to that actor's durable
   object without accepting an object name from the browser.
10. A process restart preserves identity, application session when configured,
    and object state.

### 1.2 Explicit non-goals for v1

- Multiple application nodes sharing one active durable-object namespace.
- Untrusted customer-supplied JavaScript.
- Cross-region placement or migration of object actors.
- A public raw durable-object gateway.
- One cookie serving simultaneously as IdP and application session.
- Direct access from JavaScript to tiny-idp stores, credentials, keys, or tokens.
- A configurable identity workflow before the native login path is stable.
- Coordinated transactions across identity, application auth, and object stores.

## 2. Is tiny-idp usable?

### 2.1 What works now

`pkg/embeddedidp.Options` exposes issuer, mode, store, cookie, token secret,
audit, consent, limiter, address resolver, authenticator, password policy/work,
and maintenance configuration. `embeddedidp.New` validates before construction
and returns a normal `http.Handler` through `Provider.Handler()`.

The issuer path is already meaningful. `pkg/embeddedidp/provider.go:64-73`
derives health and readiness paths from the issuer URL path. An issuer of
`https://app.example.test/idp` therefore fits the intended mount topology.

Production validation checks:

- HTTPS issuer;
- persistent supported-schema store;
- client validity;
- token secret length;
- secure cookie policy;
- durable audit readiness;
- production limiter and client-address resolver;
- bounded password work and password policy;
- maintenance support;
- exactly one usable RS256 signing key; and
- retained verification-key validity.

The provider has synchronous maintenance, readiness, liveness, close, and
metrics/reporting APIs. This is a usable embedding surface.

### 2.2 What “usable” does not mean

The library does not bootstrap a complete application by itself. The host must
still own storage paths, secrets, client/user/key initialization, TLS or trusted
proxy configuration, maintenance scheduling, audit storage, backup, and graceful
shutdown. That is correct for an embedding library, but it means the proposed
product needs a host package and initialization command.

The project also has open release evidence outside unit/integration tests,
including hosted conformance and operational canary work tracked in earlier
production tickets. The application should be described as integration-ready,
not universally production-certified.

### 2.3 Current test evidence

On 2026-07-11, using the workspace's normal `go.work`:

```text
tiny-idp:      go test ./... -count=1 -> PASS
go-go-goja:    go test ./... -count=1 -> PASS
go-go-objects: go test ./... -count=1 -> PASS
```

The tiny-idp suite includes the strict Fosite adapter, SQLite store, embedding
API, custom static analyzers, security trace, state models, and verification
plans. The go-go-goja suite includes xgoja generation, runtime ownership,
Express, planned auth, OIDC host auth, assets, and provider infrastructure. The
go-go-objects suite includes manager, storage, actor dispatch, gateway, and xgoja
provider tests.

## 3. Current component architecture

### 3.1 go-go-goja Express host

`pkg/gojahttp.Host` owns route registration and dispatch. Planned routes declare
authentication, CSRF, resource, authorization, audit, and handler behavior.
`RejectRawRoutes: true` prevents accidental raw JavaScript routes from bypassing
the planned pipeline.

At dispatch, Go authenticates the request and projects a read-only actor into JS:

```js
app.get("/me")
  .auth(express.user().required())
  .allow("user.self.read")
  .audit("user.self.read")
  .handle((ctx, res) => res.json({ id: ctx.actor.id }));
```

The JavaScript handler sees `ctx.actor`, but the authentication mechanism and
application session remain Go-owned.

### 3.2 OIDC relying-party hostauth

`pkg/gojahttp/auth/keycloakauth` is named for its first deployment but implements
standard OIDC Authorization Code + PKCE behavior:

- discovery through `oidc.NewProvider`;
- state/nonce/PKCE transaction storage;
- authorization redirect;
- token exchange;
- ID-token signature, issuer, audience, and nonce verification;
- normalized subject/profile claims; and
- opaque application-session creation.

`pkg/xgoja/hostauth.Builder` creates stores, a session manager, app authorization,
audit, native auth routes, and closers. The generated HTTP serve command mounts
those routes before the JavaScript host fallback.

### 3.3 go-go-objects

`durableobjects.Manager` maps `(namespace,name)` to one live actor. Concurrent
startup is coalesced. Each actor owns one Goja runtime and private SQLite storage.
Dispatch is serialized per object. Actors can be evicted and recovered from
storage. `Server` and `Gateway` provide embedding and HTTP surfaces.

The xgoja provider already supports:

- runtime module `require("durableobjects")`;
- `rpc`, `fetch`, and mountable `gateway()`;
- embedded bundle and manifest assets;
- SQLite storage root;
- CPU and idle timeouts;
- alarm and eviction loops;
- HTTP provider host-service discovery;
- command sets and help; and
- TypeScript declarations.

The README calls the runtime experimental. Important limitations include no
storage quotas, no distributed ownership, trusted bundles only, and immature
schema migration policy.

### 3.4 xgoja generated runtime package

`target.kind: package` generates a Go package exposing provider registration,
decoded runtime plan, `NewBundle`, host-service injection, runtime construction,
TypeScript declarations, and optional command attachment. This is the correct
mechanism for a custom lifecycle host.

## 4. The integration problem

### 4.1 Two authentication sessions are required

The IdP browser session proves that the user authenticated to the issuer. The
application session proves that the browser completed the OIDC RP flow and is
authorized to call this application. They have different audiences, CSRF state,
expiry, revocation, and logout semantics.

```text
tinyidp_session
  owner: tiny-idp
  purpose: issuer login/consent continuity

app_session
  owner: application hostauth
  purpose: authenticate planned application routes
```

Do not pass the tiny-idp cookie into Express authentication or eliminate the OIDC
round trip through an internal user lookup. The real OIDC flow is the product's
integration test and preserves future separation of issuer and application.

### 4.2 Same-process discovery creates a startup cycle

`keycloakauth.New` performs discovery immediately. In the proposed topology, the
issuer URL points to the same HTTP server that is still being constructed. If it
uses the default network client, startup fails before the listener exists.

The correct solution is an injectable OIDC HTTP client:

```go
type OIDCConfig struct {
    IssuerURL       string
    ClientID        string
    RedirectURL     string
    HTTPClient      *http.Client
}
```

The client uses an in-process `RoundTripper` for requests whose origin matches the
configured public issuer and delegates all other origins to the normal transport.
It routes discovery, JWKS, and token requests directly to `tinyidp.Handler()`
while retaining the public HTTPS URLs in metadata and issuer validation.

```text
OIDC library GET https://app.example.test/idp/.well-known/...
  -> InProcessIssuerTransport
  -> httptest-style request to tiny-idp.Handler
  -> normal discovery JSON with public issuer URLs
```

The transport is host-only. It is not exposed to JavaScript. Production may
choose ordinary network loopback instead, but startup must not depend on ingress,
DNS, or TLS availability.

### 4.3 Raw durable-object gateways are not user authorization

The gateway path includes namespace and object name. Authentication alone does
not prove that a user may access the selected name. The private-user API must
derive object identity after authentication:

```text
authenticated actor ID
  -> canonical subject binding
  -> HMAC/encoded object name
  -> ObjectID(USER_STATE, derivedName)
  -> Manager.Dispatch
```

The browser supplies only the operation and bounded payload. It never supplies
the object name.

## 5. Recommended architecture

### 5.1 Hybrid generated package plus custom Go host

```text
xgoja.yaml
  -> xgoja generate
  -> internal/xgojaruntime/       generated provider/runtime package

cmd/selfcontained/main.go         handwritten Glazed host
  -> config and init
  -> tiny-idp store/provider
  -> in-process OIDC transport
  -> app auth stores/session/RP handlers
  -> durable-object server/manager
  -> generated xgoja bundle/runtime
  -> Express route registration
  -> outer ServeMux and HTTP server
```

The host is small but intentional. It owns infrastructure that cannot be safely
represented as JSON module configuration: database handles, audit sinks, issuer
transport, token secret, maintenance scheduler, listener, shutdown order, and
shared health.

### 5.2 URL layout

| Path | Owner | Authentication |
|---|---|---|
| `/idp/.well-known/openid-configuration` | tiny-idp | public |
| `/idp/authorize` | tiny-idp | IdP browser interaction |
| `/idp/token` | tiny-idp | OAuth client/PKCE |
| `/idp/jwks.json` | tiny-idp | public |
| `/auth/login` | native RP adapter | public entry |
| `/auth/callback` | native RP adapter | state/nonce/PKCE |
| `/auth/logout` | native app session | app session/CSRF for POST |
| `/auth/session` | native app session | app session |
| `/static/*` | embedded assets | public |
| `/api/me` | Express planned route | required user |
| `/api/object/*` | Express + subject-object bridge | required user + CSRF for writes |
| `/healthz` | aggregate host | public |
| `/readyz` | aggregate host | public |
| `/` | embedded index route | public |

Static files remain under `/static/`. The root route returns the embedded HTML
shell; functional API paths are never used as static-file mounts.

### 5.3 Startup sequence

```text
1. parse Glazed configuration; validate public origin and filesystem permissions
2. open identity SQLite store and durable audit sink
3. require initialized signing key, RP client, and user records
4. construct embedded tiny-idp for issuer <publicBase>/idp
5. construct in-process issuer HTTP client
6. open app-auth/session/audit SQLite stores
7. construct provider-neutral OIDC RP handlers with injected issuer client
8. construct durable-object manager from embedded bundle and storage root
9. create gojahttp Host using app AuthOptions
10. create generated xgoja Bundle with host services
11. create runtime; register Express routes; mount frontend assets
12. build outer ServeMux in fixed priority order
13. run initial tiny-idp maintenance and readiness checks
14. start maintenance, alarm, and idle-eviction goroutines under errgroup context
15. start HTTP server; on cancellation, shut down and close in reverse order
```

All goroutines use `errgroup.WithContext`. Background-loop errors become readiness
failures or terminate the host according to a documented policy.

## 6. Identity and durable-object binding

### 6.1 Stable subject mapping

tiny-idp emits a stable `sub`. The app-auth store currently creates deterministic
IDs as `user:<sub>`. That is sufficient for restart and restore stability when
the issuer is fixed. To support multiple issuers safely, define:

```go
type Subject struct {
    Issuer string
    Sub    string
}

func AppUserID(subject Subject) string {
    return "oidc:" + base64url(sha256(subject.Issuer + "\x00" + subject.Sub))
}
```

Do not use email as identity. Email can change and may be unverified.

### 6.2 Object-name derivation

The host owns a separate object-binding secret:

```go
func UserObjectName(bindingKey []byte, actorID string) string {
    mac := hmac.New(sha256.New, bindingKey)
    mac.Write([]byte("user-state@v1\x00"))
    mac.Write([]byte(actorID))
    return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
```

This prevents subject identifiers from appearing in gateway paths, filenames,
metrics, or routine object events. Rotation requires an explicit migration; the
binding key is durable application state.

### 6.3 Subject-bound service

Add to go-go-objects:

```go
type SubjectDispatcher interface {
    DispatchForActor(
        ctx context.Context,
        actorID string,
        namespace string,
        request FetchRequest,
    ) (FetchResponse, error)
}

type BoundDispatcher struct {
    Manager    *Manager
    BindingKey []byte
    AllowedNamespaces map[string]struct{}
}
```

The JavaScript adapter may expose:

```ts
interface UserObjects {
  fetch(actor: Readonly<Actor>, namespace: string, request: FetchRequest): FetchResponse;
  rpc(actor: Readonly<Actor>, namespace: string, method: string, args: unknown[]): unknown;
}
```

Because route JavaScript is trusted, passing the actor projection is acceptable
for v1. The stronger endpoint is a Go-owned planned-route integration that takes
the authenticated actor from the enforcer rather than a JS-supplied object. The
design should implement that before exposing route scripts to less-trusted
authors.

## 7. Cross-repository API changes

### 7.1 go-go-goja: provider-neutral OIDC adapter

Rename `keycloakauth` to `oidcauth` before a stable external API is declared.
Do not add a compatibility adapter unless an existing consumer requires it.

```go
type Config struct {
    IssuerURL       string
    ClientID        string
    ClientSecret    string
    RedirectURL     string
    Scopes          []string
    SessionManager  *sessionauth.Manager
    UserNormalizer  UserNormalizer
    TransactionStore TransactionStore
    HTTPClient      *http.Client
}
```

Discovery, token exchange, and remote-key verification must use the supplied
client. Tests require an in-process issuer transport and confirm that no network
dial occurs.

Rename `User.KeycloakSub`, `ByKeycloakSub`, and related schema concepts to
provider-neutral issuer/subject fields before product adoption. This is a direct
migration, not a dual-field compatibility layer.

### 7.2 go-go-goja: native mount contributions

Generalize HTTP serve host services so providers or custom hosts can contribute
native handler mounts with explicit patterns and priorities:

```go
type NativeMount struct {
    Pattern string
    Handler http.Handler
    Owner   string
}
```

Reject duplicates before server start. Auth handlers, tiny-idp, aggregate health,
and any future WebSocket server then use one mount contract.

### 7.3 go-go-objects: private subject binding

Add `BoundDispatcher` and a provider host-service key. Separate public gateway
configuration from private dispatch configuration:

```yaml
privateBindings:
  userState:
    namespace: USER_STATE
    actorSource: authenticated-user
```

The raw HTTP gateway is disabled by default in the self-contained product.
Storage quotas, maximum value size, and per-actor storage metrics should be added
before calling the product multi-user production-ready.

### 7.4 tiny-idp: xgoja provider package

After the custom host works, add:

```text
pkg/xgoja/providers/tinyidp/
  provider.go
  config.go
  services.go
  commands.go
  typescript.go       only if a safe JS module exists
  provider_test.go
```

The provider registers package ID `tinyidp`, contributes configuration sections,
creates or receives a host-owned `Service`, registers closers, and exposes a
native mount. It need not expose a JavaScript module in v1.

```go
type Service struct {
    Provider   *embeddedidp.Provider
    Handler    http.Handler
    Issuer     string
    HTTPClient *http.Client
}
```

The provider may later expose redacted discovery/readiness helpers. It must never
expose raw stores, password APIs, signing material, token issuance, or provider
mutation to JS.

### 7.5 tiny-idp: cookie path and host lifecycle

Extend cookie configuration with a path defaulting to the issuer path (`/idp` in
this product). This reduces unnecessary IdP-cookie exposure to application paths.
Add tests for issuer prefixes, cookie paths, and coexistence with `app_session`.

Implementation checkpoint: `embeddedidp.CookieConfig` now exposes distinct
session and CSRF names plus an optional path. An empty path retains the
issuer-derived least-exposure default. Both the embedding API and the lower
Fosite adapter reject invalid or colliding names, unsafe paths, and paths that do
not cover the issuer. The integration test carries an unrelated host session
through login and silent authorization, proving that the application and IdP
cookies can coexist without name interpretation or accidental replacement.

### 7.6 Development vertical-slice implementation checkpoint

The first custom host is now implemented in `cmd/tinyidp-xapp`. Its construction
order follows the ownership model in this guide: tiny-idp first, then the
restricted in-process OIDC client, application auth services, the external HTTP
host, the host-owned Durable Objects manager and bound dispatcher, the generated
xgoja runtime, trusted route registration, and finally the outer handler. The
listener is created only by the `serve` command after construction succeeds.

The end-to-end test crosses all intended trust boundaries rather than injecting
an actor directly. It completes the browser OIDC code flow, creates the opaque
application session, obtains host CSRF state, and performs an actor-bound SQLite
write and reread. Raw object gateways remain absent. The binding key is stable,
owner-only state under the chosen root; the IdP and application auth stores are
still deliberately ephemeral and therefore keep this command development-only.

### 7.7 Persistent initialization checkpoint

The `init` command now treats initialization as desired-state reconciliation,
not an imperative seed script. The manifest is committed only after migrations,
security roots, the RP client, first credential, and signing key are available.
Reruns validate immutable identity configuration and preserve credential/key
material. This allows a future production `serve` path to use `state.json` as a
strict completion marker while still validating every referenced file and
database before accepting traffic.

## 8. xgoja build design

### 8.1 Proposed specification

```yaml
schema: xgoja/v2
name: tinyidp-user-object-app

go:
  module: example.com/tinyidp-user-object-app
  version: "1.26"

target:
  kind: package
  output: internal/xgojaruntime

providers:
  - id: go-go-goja-core
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/core
    register: Register
  - id: go-go-goja-host
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/host
    register: Register
  - id: go-go-goja-http
    import: github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http
    register: Register
  - id: go-go-objects
    import: github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects
    register: Register

runtime:
  modules:
    - provider: go-go-goja-core
      name: timer
    - provider: go-go-goja-host
      name: fs
      as: fs:assets
      config:
        embedded:
          allow: true
          mounts:
            - asset: frontend
              mount: /app
    - provider: go-go-goja-http
      name: express
      config:
        reject-raw-routes: true
        dev-errors: false
    - provider: go-go-objects
      name: durableobjects
      config:
        storageRoot: ./var/durable-objects
        bundleAsset: object-bundle
        bundleAssetPath: objects.js

sources:
  - id: routes
    kind: jsverbs
    from: {dir: ./app/routes}
  - id: frontend
    kind: assets
    from: {dir: ./app/frontend}
  - id: object-bundle
    kind: assets
    from: {dir: ./app/objects}

artifacts:
  - id: runtime-package
    type: runtime-package
    output: internal/xgojaruntime
  - id: assets
    type: embedded-assets
    sources: [frontend, object-bundle]
```

The custom Go host is the final binary. xgoja owns generated composition, not the
application's security lifecycle.

### 8.2 Route script

```js
const express = require("express");
const objects = require("durableobjects");
const assets = require("fs:assets");

function site() {
  const app = express.app();
  app.staticFromAssetsModule("/static", assets, "/app/public");

  app.get("/").public().handle((_ctx, res) =>
    res.type("text/html").send(assets.readFileSync("/app/public/index.html", "utf8")));

  app.get("/api/object")
    .auth(express.user().required())
    .allow("user.self.read")
    .audit("user.object.read")
    .handle((ctx, res) => {
      const out = objects.fetchForActor(ctx.actor, "USER_STATE", {
        method: "GET", path: "/state"
      });
      res.status(out.status).json(out.body);
    });

  app.post("/api/object")
    .auth(express.user().required())
    .csrf()
    .allow("user.self.update")
    .audit("user.object.updated")
    .handle((ctx, res) => {
      const out = objects.fetchForActor(ctx.actor, "USER_STATE", {
        method: "POST", path: "/state", body: ctx.body
      });
      res.status(out.status).json(out.body);
    });
}
```

`fetchForActor` is proposed; the current provider exposes `fetch(namespace,
name, request)` and is not sufficient as the public private-object contract.

## 9. Durable object implementation

```js
class UserState {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  async fetch(req) {
    if (req.method === "GET" && req.path === "/state") {
      return { status: 200, body: this.state.storage.get("document") || {} };
    }
    if (req.method === "POST" && req.path === "/state") {
      const next = validateDocument(req.body);
      this.state.storage.put("document", next);
      return { status: 200, body: next };
    }
    return { status: 404, body: { error: "not_found" } };
  }
}

exports.objects = { UserState };
```

Input validation must bound object keys, nesting, string sizes, and total encoded
size. The current storage API has no quota; the bound must be enforced before
write and later by storage accounting.

## 10. Configuration and initialization

The CLI uses Glazed. Do not read deployment configuration directly with
`os.Getenv`; environment-backed values are resolved through Glazed sections.

```yaml
publicBaseURL: https://app.example.test
listen: 127.0.0.1:8787
stateRoot: ./var

identity:
  issuerPath: /idp
  database: ./var/idp.sqlite
  tokenSecretFile: ./var/secrets/idp-token.key
  objectBindingKeyFile: ./var/secrets/object-binding.key

application:
  database: ./var/app.sqlite
  clientId: self-app
  redirectPath: /auth/callback

objects:
  root: ./var/durable-objects
  cpuTimeout: 2s
  idleTimeout: 5m
```

### 10.1 Initialization command

`app init` performs explicit, idempotent initialization:

- create owner-only state/secrets directories;
- create identity and application schemas;
- generate token and object-binding secrets;
- generate an initial signing key;
- create the `self-app` public PKCE client with exact redirect URI;
- optionally create the first user through an interactive or file-backed input;
- run readiness validation; and
- print no secret values.

Normal `serve` refuses missing initialization. It does not silently generate a
new signing key or token secret on every start.

## 11. Storage, backup, and recovery

```text
var/
  idp.sqlite
  idp-audit.sqlite or durable audit log
  app.sqlite
  durable-objects/
    alarms.sqlite
    objects/.../*.sqlite
  secrets/
    idp-token.key
    object-binding.key
```

There is no cross-database transaction. Identity creation, application-user
upsert, and object creation occur at different times and are independently
retryable. Stable issuer+subject mapping reconnects restored identities to the
same object binding.

Back up the entire state root from one quiesced or snapshot-consistent boundary.
Restore validation checks schemas, signing key, secrets, client redirect,
application-user mapping, object index reconciliation, and readiness before
listening.

## 12. Security design

### 12.1 Trust boundaries

- Browser input is untrusted.
- Route and durable-object JavaScript are trusted deployment code in v1.
- tiny-idp, OIDC RP verification, session management, CSRF, object binding,
  persistence, and HTTP lifecycle are native Go trusted code.
- Embedded frontend assets are public data.
- Object storage is application data and may contain user secrets; it is not
  exposed to identity scripts.

### 12.2 Required controls

- Exact issuer and redirect URI validation.
- PKCE, state, nonce, and ID-token verification.
- Separate cookie names and scoped paths.
- Secure, HttpOnly, SameSite cookies in production.
- CSRF on every application-session-authenticated mutation.
- `RejectRawRoutes: true`.
- No raw public durable-object gateway.
- Object name derived only from authenticated actor.
- Bounded request bodies and object values.
- Trusted JS bundles only; runtime CPU deadlines and actor replacement.
- Durable audit for identity and planned-route security outcomes.
- Aggregate readiness for IdP, app stores, object manager, runtime, maintenance,
  and background schedulers.

### 12.3 Logout semantics

Application logout revokes the app session but initially leaves the IdP session.
The next login may complete without credentials. The UI must label this behavior.
A later phase can add OIDC RP-initiated logout or a “sign out of identity
provider” action. Do not pretend app logout clears the issuer session.

## 13. Operational model

The first production target is one active process. SQLite and in-memory actor
ownership make multiple replicas unsafe without routing or a coordination layer.
Use a single replica with persistent storage, backups, readiness, and controlled
rollout.

Aggregate health exposes component names without secrets:

```json
{
  "ready": true,
  "components": {
    "identity": "ready",
    "appAuth": "ready",
    "durableObjects": "ready",
    "javascriptRoutes": "ready",
    "maintenance": "ready"
  }
}
```

Metrics must avoid user IDs and object names as labels.

## 14. Implementation phases

### Phase 0: prove the integration seam

- Create a new application directory/module under the existing top-level
  workspace arrangement without nested experimental modules in tiny-idp.
- Generate an xgoja runtime package with Express, assets, and durable objects.
- Build a custom Go host with development tiny-idp and an in-process OIDC client.
- Complete one browser or HTTP-client login and one private object read/write.

Exit: one test proves login, subject mapping, object isolation, persistence after
restart, and logout behavior.

### Phase 1: provider-neutral OIDC and in-process transport

- Rename Keycloak-specific types/schema fields.
- Add injected HTTP client support for discovery/JWKS/exchange.
- Add no-network in-process issuer tests.
- Preserve standard public issuer validation.

Exit: RP services can be built before the listener starts.

### Phase 2: subject-bound durable objects

- Implement `BoundDispatcher` and object binding key.
- Add xgoja provider service and TypeScript contract.
- Disable raw gateway in product configuration.
- Test two users cannot address one another's object through any route input.

Exit: object isolation is a native invariant, not route convention.

### Phase 3: persistent product host

- Add Glazed `init`, `serve`, `doctor`, `backup`, and `restore` commands.
- Use persistent tiny-idp, app-auth/session/audit, and object stores.
- Add maintenance, alarm, eviction, readiness, and graceful shutdown.
- Add coordinated state-root backup and restore verification.

Exit: restart and restore preserve login identities and object state.

### Phase 4: frontend product loop

- Build the embedded HTML/TypeScript frontend.
- Add session bootstrap, login/logout, CSRF handling, object read/write, errors,
  and accessibility states.
- Serve assets only under `/static/` and use an explicit root route.

Exit: a new user can complete the product story without curl.

### Phase 5: xgoja tiny-idp provider

- Package the proven host service as an xgoja provider capability.
- Add config schema, Glazed sections, mount contribution, closers, help, doctor,
  and generated-host example.
- Keep JavaScript exposure absent or read-only/redacted.

Exit: a second app can select the provider declaratively without copying host
lifecycle code.

### Phase 6: production evidence

- Run race, fuzz, static analysis, model scenarios, hosted conformance, browser
  end-to-end, failure injection, backup/restore, and load tests.
- Add storage quotas and explicit object schema migration.
- Document single-replica deployment and residual risks.

Exit: an explicit release review approves the intended deployment class.

## 15. Test strategy

### 15.1 End-to-end browser state machine

```text
Unauthenticated
  -> LoginRedirected
  -> IdPInteraction
  -> RPCallback
  -> AppSession
  -> ObjectRead
  -> ObjectWriteWithCSRF
  -> Restart
  -> ObjectReadPreserved
  -> AppLogout
  -> ProtectedRouteRejected
```

Test denial, bad password, malformed state, nonce mismatch, reused code, missing
CSRF, object-name injection, second user isolation, expired sessions, disabled
user, signing-key rotation, object actor eviction, Promise timeout, storage
failure, and shutdown during dispatch.

### 15.2 Build validation

```text
go test ./... -count=1                         in all three repositories
xgoja doctor -f xgoja.yaml
xgoja gen-dts -f xgoja.yaml
xgoja generate -f xgoja.yaml
go test ./... -race -count=1                  in product module
```

### 15.3 Security evidence

- tiny-idp verification scenarios cover forced login and password lifecycle.
- Static analyzers enforce OIDC/client/object authority boundaries.
- Runtime traces correlate OIDC interaction, app session creation, and object
  dispatch using opaque request IDs, not user labels.
- Model checks cover login/callback/session/object access ordering at an abstract
  level after the first implementation works.

## 16. Decision records

### Decision: custom Go host plus generated runtime package

- **Context:** Identity and object lifecycle require Go resources and startup
  ordering beyond static module configuration.
- **Options considered:** pure generated binary; custom host without xgoja;
  generated runtime package imported by a custom host.
- **Decision:** Use the generated package plus custom host.
- **Rationale:** It retains xgoja composition and assets while making the trusted
  lifecycle explicit and testable.
- **Consequences:** A small handwritten main remains part of the product.
- **Status:** proposed

### Decision: integrate through standard OIDC

- **Context:** The IdP and app run in one process, but may separate later.
- **Options considered:** share the IdP session/store directly; internal login
  callback; real OIDC with in-process transport.
- **Decision:** Use real OIDC and separate app sessions.
- **Rationale:** It exercises protocol boundaries and preserves deployment
  flexibility without requiring network availability during startup.
- **Consequences:** Two sessions and explicit logout semantics are required.
- **Status:** proposed

### Decision: object identity is host-derived

- **Context:** The raw gateway lets callers select object names.
- **Decision:** Derive private object names from authenticated actor ID using a
  durable binding key.
- **Rationale:** User isolation becomes a native invariant.
- **Consequences:** Binding-key backup and migration are critical.
- **Status:** proposed

### Decision: tiny-idp provider follows proof of composition

- **Context:** Provider packaging before lifecycle behavior is proven would freeze
  the wrong API.
- **Decision:** Build the custom host first, then extract an xgoja provider.
- **Rationale:** The second implementation is the right time to identify reusable
  provider seams.
- **Consequences:** Phase 0 contains manual host wiring.
- **Status:** proposed

## 17. Risks and open questions

- Does the OIDC library use the injected client consistently for discovery,
  exchange, and later JWKS refresh? Tests must prove all three.
- Should app user IDs use public `issuer+sub` hashes or keyed derivation?
- Which database owns app user disablement when the IdP user is disabled?
- How does app logout optionally initiate IdP logout?
- Should object binding use app actor ID or raw verified issuer+subject?
- What storage quota is acceptable per personal object?
- How are durable-object schema migrations versioned and rolled back?
- Should the final provider contribute native mounts directly or a service that
  the HTTP provider mounts?
- Can identity and application audit share a physical sink while retaining event
  schemas and retention policy?
- What is the supported reverse-proxy and forwarded-address model?

## 18. Intern onboarding and implementation map

Read in this order:

1. tiny-idp `pkg/embeddedidp/options.go` and `provider.go`.
2. go-go-goja Express auth integration guide and generated-host example.
3. `pkg/xgoja/hostauth/builder.go` and `keycloakauth` callback flow.
4. go-go-objects README, `manager.go`, `server.go`, and xgoja provider.
5. xgoja generated runtime-package example.
6. This document's integration problem, startup sequence, and security sections.

The first implementation files should be:

```text
product/
  xgoja.yaml
  cmd/app/main.go
  internal/apphost/config.go
  internal/apphost/host.go
  internal/apphost/inprocess_oidc.go
  internal/apphost/init.go
  internal/apphost/subject_objects.go
  app/routes/site.js
  app/objects/objects.js
  app/frontend/public/index.html
  app/frontend/public/app.ts
  app/frontend/public/styles.css
```

The intern's first pull request should implement the in-process OIDC transport
and its isolated test. The second should produce the minimal custom host and
login smoke. The third should add subject-bound object dispatch and two-user
isolation. Frontend work follows only after the native invariants are tested.

## 19. Acceptance criteria

- One command initializes state without printing secrets.
- One command starts the self-contained server.
- The binary contains route JS, object JS, HTML, CSS, and browser JS assets.
- Login uses tiny-idp through Authorization Code + PKCE, state, and nonce.
- Express routes authenticate through a separate opaque app session.
- Object identity is derived from the authenticated subject in Go.
- Two users cannot read or mutate one another's durable object.
- State survives restart and verified backup/restore.
- Raw object gateway and raw Express routes are disabled.
- Production mode rejects insecure issuer, cookies, missing audit, ephemeral
  stores/secrets, missing keys, or missing initialization.
- All three upstream suites plus product race/e2e tests pass.
- xgoja doctor, DTS generation, runtime-package generation, and clean build pass.
- The intended deployment class and residual risks are documented.

## 20. Local API references

### tiny-idp

- `pkg/embeddedidp/options.go:40-197` — embedding and production preflight.
- `pkg/embeddedidp/provider.go:19-220` — handler, issuer path, readiness,
  maintenance, and lifecycle.
- `pkg/idpstore/types.go` and `interfaces.go` — domain and persistence contracts.
- `pkg/sqlitestore/store.go` — persistent implementation.
- `internal/fositeadapter/provider.go` — strict OIDC protocol behavior.

### go-go-goja

- `cmd/xgoja/doc/19-express-auth-host-integration-guide.md`.
- `pkg/gojahttp/host.go` and `enforcer.go` — route and security pipeline.
- `pkg/gojahttp/auth/keycloakauth/keycloakauth.go:20-237` — current generic OIDC
  behavior under Keycloak-specific naming.
- `pkg/xgoja/hostauth/builder.go:18-241` — stores, sessions, native handlers.
- `pkg/xgoja/providers/http/serve.go:405-487` — auth services and mux mounting.
- `examples/xgoja/14-generated-runtime-package` — custom-host generation pattern.
- `examples/xgoja/21-generated-host-auth` — embedded frontend + planned OIDC app.

### go-go-objects

- `pkg/durableobjects/manager.go` — object ownership and dispatch.
- `pkg/durableobjects/server.go` — embedding and background loops.
- `pkg/durableobjects/id.go` — namespace/name validation and storage identity.
- `pkg/xgoja/providers/durableobjects/durableobjects.go` — xgoja module,
  configuration, assets, gateway, host services, and TypeScript API.
- `examples/counter/xgoja-buildspec.yaml` — generated Express composition.

## 21. Final recommendation

Build this product. The foundational work is present and the integration is
small enough to prove with one vertical slice. Do not begin by making tiny-idp a
large JavaScript module. Begin with a custom Go host importing an xgoja-generated
runtime package, use the real OIDC boundary internally, and make subject-bound
durable-object dispatch a native service.

When that version is stable, extract tiny-idp lifecycle and mount behavior into
an xgoja provider. At that point the provider API will be based on a working
application rather than an imagined composition model.
