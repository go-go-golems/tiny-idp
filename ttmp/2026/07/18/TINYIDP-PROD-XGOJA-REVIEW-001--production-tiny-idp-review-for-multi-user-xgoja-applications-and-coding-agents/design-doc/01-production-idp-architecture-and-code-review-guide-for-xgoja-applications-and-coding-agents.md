---
Title: Production IdP architecture and code review guide for xgoja applications and coding agents
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
    - Path: repo://pkg/embeddedidp/options.go
      Note: Strict production configuration invariants
    - Path: repo://internal/fositeadapter/provider.go
      Note: OIDC, device authorization, token, introspection, and browser interaction endpoints
    - Path: repo://cmd/tinyidp-xapp/production_app.go
      Note: Current one-application production composition
    - Path: repo://cmd/tinyidp-xapp/device_api.go
      Note: Current Go-owned bearer API boundary
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Existing controlled public signup pattern
    - Path: ws://go-go-goja/modules/express/auth_builders.go
      Note: Express planned-auth DSL
    - Path: ws://go-go-goja/pkg/gojahttp/enforcer.go
      Note: Host enforcement before JavaScript dispatch
    - Path: ws://go-go-goja/pkg/xgoja/hostauth/builder.go
      Note: Application OIDC login and session composition
ExternalSources:
    - https://openid.net/specs/openid-connect-core-1_0.html
    - https://www.rfc-editor.org/info/rfc7636/
    - https://www.rfc-editor.org/info/rfc7662/
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://www.rfc-editor.org/info/rfc8628/
    - https://www.rfc-editor.org/info/rfc9700/
Summary: Evidence-backed current-state review and target production design for one tiny-idp issuer serving multi-user xgoja browser applications and device-authorized coding agents.
LastUpdated: 2026-07-18T17:00:00-04:00
WhatFor: Teach a new engineer how the complete identity system works, identify production gaps, and provide file-level implementation and review guidance.
WhenToUse: Before implementing signup, multi-application provisioning, planned bearer authentication, coding-agent lifecycle, or a production deployment.
---

# Production IdP architecture and code review guide for xgoja applications and coding agents

## Executive Summary

The repository contains a production-shaped identity-provider core, not yet a
complete multi-application identity product. Its strict mode has durable SQLite
state, Argon2id password handling, lockout, rotating RS256 keys, OIDC
authorization-code flow with PKCE, refresh-token rotation, OAuth device
authorization, token introspection, audit events, readiness checks, backup
support, and direct TLS serving. The most security-sensitive packages have
substantial unit, integration, race, and failure-injection coverage.

The tinyidp-xapp command is an especially useful vertical slice. It proves that
one process can combine the strict issuer, an xgoja browser application, a
host-owned OIDC application session, Durable Objects, a device client, and an
opaque-bearer resource server. It also proves that two users receive distinct
actors and that scopes, audience, password-change revocation, ambiguous
Authorization headers, and token secrecy can be enforced.

It does not prove the requested production product. The current state
initializer hard-codes one browser client, one device client, one introspection
client, one audience, and one initial user. There is no public signup surface in
the production xapp. Coding-agent routes are separate Go handlers rather than
routes protected by the go-go-goja Express auth syntax. The device client does
not request refresh tokens. The device authorization endpoint uses the private
form parameter audience where RFC 8707 specifies resource. SQLite and several
in-memory coordination components constrain the first deployment to one active
IdP process and one application replica unless additional work is performed.

The recommended initial production architecture is:

- Run one central tiny-idp issuer as a separately deployed, single-active
  identity service.
- Give every browser application its own public OIDC client, redirect URI set,
  logout URI set, and application session database.
- Give every protected API audience its own confidential introspection client.
- Give each coding-agent distribution or trust domain an explicit public device
  client with narrowly allowed scopes and resources.
- Make signup an issuer-owned native workflow that creates an identity only.
  Each xgoja application remains responsible for its own membership, role, and
  tenant onboarding.
- Keep browser session routes and agent bearer routes in distinct URL
  namespaces. Add a first-class OAuth bearer security mode to planned Express
  routes before claiming that coding agents use the Express auth syntax.
- Continue using opaque access tokens and RFC 7662 introspection for the first
  release. Do not expose access tokens or the resource-server client secret to
  JavaScript.

This is a conditional production recommendation. The strict issuer can be the
foundation, and the current xapp can be the reference implementation. Public
signup, generic multi-app provisioning, standards-aligned resource indicators,
agent credential lifecycle, and planned bearer auth are release blockers for
the product described in this ticket.

### Readiness at a glance

| Capability | Current state | Production conclusion |
| --- | --- | --- |
| Durable identities and credentials | Implemented and tested | Suitable for a single-active deployment after operational validation |
| Browser OIDC login | Implemented with PKCE and app-owned sessions | Suitable as the browser authentication foundation |
| Express session-route authorization | Implemented and fail-closed | Suitable for browser application routes |
| OAuth device authorization | Implemented with durable one-use grants | Strong foundation; finish interoperability and lifecycle policy |
| Opaque-token introspection | Implemented with resource-client authentication | Suitable if cache/revocation latency is accepted |
| Public signup | Pattern exists in Message Desk, absent from production xapp | Release blocker |
| Multiple deployed applications | Protocol primitives exist, provisioning product does not | Release blocker |
| Agent auth through Express DSL | Not modeled; current agent API is Go-owned | Release blocker if this is a firm requirement |
| Coding-agent refresh/revocation | Device client is access-token-only; no revocation endpoint | Release blocker or explicitly accepted short-session limitation |
| Active-active IdP | Unsupported with the current SQLite topology | Do not deploy active-active |
| Production DPoP | Explicitly unsupported | Deferred hardening, not required for initial opaque bearer release |

## 1. How to read this guide

This guide is written for an engineer who is new to the codebase but is expected
to make security-sensitive changes. It separates three kinds of statement:

- **Observed** means the behavior exists in current source or tests.
- **Proposed** means this ticket recommends a target design that is not yet
  implemented.
- **Decision required** means product or operational policy must be fixed before
  implementation can be considered complete.

Read the component map first, then follow one browser request and one device
request through the source. Only after those paths are clear should you review
the proposed multi-application shape. Authentication defects often arise from
confusing a credential owner, a session owner, or an authorization boundary.

The important repository names are:

- tiny-idp owns identities, passwords, OIDC interactions, OAuth grants, tokens,
  signing keys, and provider audit.
- go-go-goja owns the host-side Express route plan and enforcement machinery.
- An xgoja application owns its local user row, memberships, roles, tenant
  relationships, application session, and business data.
- A coding agent owns an opaque access token in a local token cache.
- A resource server owns the confidential introspection credential and converts
  a valid token into a minimal actor before application code runs.

## 2. Problem statement and scope

The target product must allow a set of independently deployed, multi-user xgoja
applications to trust one production identity plane. A person must be able to
create an account, sign in through a browser, receive an application session,
and use an Express-planned site. The same person must be able to authorize a
coding agent through the OAuth device flow and let that agent call a
scope-limited API as that person.

The system must preserve these boundaries:

- Identity proves who the human is. It does not automatically grant application
  membership, tenant access, or an administrator role.
- A browser OIDC login results in an application cookie, not in a bearer token
  being exposed to JavaScript.
- A device authorization results in an OAuth access token for a declared
  resource and scope set. It does not create a browser application session.
- Resource authorization happens in the Go host before xgoja JavaScript or
  Durable Object code receives an actor.
- Each relying party and resource server receives only the credentials and
  protocol capabilities required for its role.

This review does not implement the missing work. It does not add compatibility
adapters, dynamic client registration, social login, federation, SCIM, WebAuthn,
active-active storage, or production DPoP. Those are separate product choices.

## 3. The three identities and three credential families

A new engineer should begin by distinguishing the human identity, the
application identity, and the OAuth client identity.

### 3.1 Human identity

The tiny-idp user has a stable opaque subject. Password credentials and account
status are stored beside that user in the identity store. OIDC emits the subject
as sub. Application code must use the pair of issuer and subject as the stable
external identity key; a subject from a different issuer is a different
identity even if the text is identical.

Primary files:

- pkg/idpstore/types.go defines users, clients, grants, sessions, and device
  grants.
- pkg/idpaccounts/accounts.go:16-95 performs normalized, atomic account
  creation.
- pkg/idpaccounts/password.go:120 onward authenticates with dummy password work,
  durable lockout, and bounded Argon2id.

### 3.2 Application identity

go-go-goja hostauth normalizes an OIDC identity into an application-owned user.
The default normalizer upserts by issuer and subject, then projects existing
memberships into the application session. It deliberately does not grant roles
or tenant membership merely because the IdP authenticated a user.

Primary files in the neighboring go-go-goja checkout:

- pkg/xgoja/hostauth/builder.go:168-209 normalizes identity and projects
  membership.
- pkg/gojahttp/auth/sessionauth/sessionauth.go stores opaque application
  sessions and CSRF state.
- pkg/gojahttp/auth/oidcauth/oidcauth.go:156-252 completes OIDC callback
  verification and creates the application session.

### 3.3 OAuth client identity

An OAuth client identifies software, not the human. The browser client
identifies one relying party. The device client identifies a public client
distribution or trust domain. The introspection client identifies one
confidential resource server. A device token therefore needs both a human sub
and a client_id; policy may need to consider both.

Never treat a public device client ID as a secret or as a unique installation
identity. Coding agents cannot safely hold a client secret. If installation
identity or attestation is required, it needs a separate design.

### 3.4 Credential ownership table

| Credential | Owner and transport | Purpose | Must not be exposed to |
| --- | --- | --- | --- |
| IdP browser cookie | Browser and tiny-idp | Resume provider interaction and SSO | Relying-party JavaScript |
| Authorization code | Browser redirect, then relying-party backend | One-use OIDC exchange | Logs and application JavaScript |
| OIDC transaction state | Relying-party backend | Bind state, nonce, PKCE verifier | Other replicas without shared storage |
| Application session cookie | Browser and xgoja hostauth | Authenticate browser routes | Other applications |
| Device code | Coding agent and token endpoint | Poll one pending device grant | Browser page, logs, JavaScript |
| User code | Human and verification page | Select one pending grant | Long-term storage |
| Access token | Coding agent and resource server | Authenticate an API request | xgoja JavaScript and logs |
| Refresh token | Coding agent if later enabled | Obtain rotated access tokens | Browser and application JavaScript |
| Introspection client secret | Resource server only | Authenticate RFC 7662 calls | Coding agents and JavaScript |
| Signing private key | tiny-idp only | Sign ID tokens | Every relying party |

## 4. Current system architecture

The following diagram is observed in tinyidp-xapp. It is a single-process
reference composition, not the recommended multi-deployment topology.

~~~text
Browser
  |  /idp/authorize, /auth/callback, xapp cookie
  v
+------------------------------------------------------------------+
| tinyidp-xapp process                                              |
|                                                                  |
|  +--------------------+       +-------------------------------+  |
|  | embedded tiny-idp  |       | go-go-goja hostauth           |  |
|  | /idp/*             |<----->| /auth/login, callback, logout |  |
|  +---------+----------+       +---------------+---------------+  |
|            |                                  |                  |
|  identity.sqlite                    application/auth.sqlite       |
|            |                                  |                  |
|            +------------+---------------------+                  |
|                         v                                        |
|                 Express planned routes                           |
|                 /api/me, /api/bbs, /api/object                   |
|                         | ctx.actor                               |
|                         v                                        |
|                    Durable Objects                               |
|                                                                  |
|  /api/device/* -> Go bearer handler -> RFC 7662 introspection     |
|                         | constrained actor                       |
|                         +---------------------> Durable Objects   |
+------------------------------------------------------------------+
                 ^
                 | device grant, opaque token, Bearer API
             Coding agent
~~~

### 4.1 Production composition

Observed composition starts in cmd/tinyidp-xapp/production_app.go:21-149. It:

- Opens the strict SQLite identity store.
- Opens a durable file audit sink.
- loads owner-only token and resource-client secrets.
- constructs embeddedidp in ProductionMode.
- constructs resourceauth with issuer, confidential resource client, audience,
  positive cache TTL, and negative cache TTL.
- opens persistent hostauth application storage.
- passes all components to the common application composer.

cmd/tinyidp-xapp/development_app.go:244-323 mounts:

- the provider at /idp/,
- static assets under /static/tinyidp/,
- Go-owned agent endpoints under /api/device/,
- native hostauth endpoints under /auth/,
- and the xgoja application at /.

Raw xgoja HTTP routes are rejected. Durable Object actor binding derives from
gojahttp.ActorFromContext, so objects receive the actor created by the
host-enforced route or the constrained Go device handler.

### 4.2 Strict provider gates

pkg/embeddedidp/options.go:82-235 refuses production construction unless the
issuer is HTTPS, cookies are secure, durable audit and rate limiting are marked
production-ready, password work is bounded and observable, the store reports a
current durable schema, and exactly one usable active RS256 signing key exists.
The token secret must contain at least 32 bytes.

These are construction invariants, not deployment evidence. A production
release still needs to prove certificate renewal, filesystem permissions,
backup restore, audit collection, disk exhaustion behavior, clock accuracy,
monitoring, and incident response in its actual environment.

### 4.3 Storage topology

pkg/sqlitestore/store.go opens SQLite with WAL, FULL synchronous behavior, a
five-second busy timeout, and one open connection. Migrations cover identity,
credentials, browser interactions, Fosite artifacts, device grants, remembered
sessions, key material, and related security state. Password change revocation
invalidates the user's active browser and token artifacts.

The supported first topology is one active process per SQLite identity
database. examples/production-host/README.md:54-58 explicitly rejects shared
network filesystems and concurrent active writers. A verified online backup and
restore drill is part of production readiness.

## 5. Browser signup, login, and Express route flow

### 5.1 Current browser login sequence

~~~text
Browser         xgoja hostauth       tiny-idp          app store       JS route
   | GET /private     |                  |                 |              |
   |----------------->| no app session   |                 |              |
   | 302 /auth/login  |                  |                 |              |
   |----------------->| create state, nonce, PKCE          |              |
   | 302 /authorize -------------------->|                 |              |
   |                   |   IdP login + consent + cookie    |              |
   |<------------------------------------| authorization code              |
   | GET /auth/callback|                  |                 |              |
   |----------------->| POST /token ---->|                 |              |
   |                   | verify issuer, signature, nonce, sub              |
   |                   | normalize (issuer, sub) ---------->| upsert user   |
   |                   | create opaque app session -------->|              |
   | app cookie        |                  |                 |              |
   | GET /private ---->| authenticate, CSRF, resolve, allow |              |
   |                   |----------------------------------------------->   |
   |                   |                         ctx.actor, no OIDC token   |
~~~

go-go-goja/pkg/gojahttp/auth/oidcauth/oidcauth.go performs discovery and uses
authorization code plus S256 PKCE. It verifies the ID token and nonce, then
normalizes issuer and subject. Tokens remain on the Go side.

One multi-replica caveat is easy to miss: the default OIDC transaction store in
oidcauth is memory-backed. Even when user, membership, and session stores are
SQLite or PostgreSQL, a login callback must reach a replica that can retrieve
the initiating state, nonce, and verifier. Keep an application at one replica
until this state is shared or routing affinity and failure behavior are
deliberately designed.

### 5.2 Express planned-auth syntax

Observed browser routes look like:

~~~javascript
app.post("/api/bbs/posts")
  .auth(express.user().required())
  .csrf()
  .resource("board", { id: "main" })
  .allow("bbs.post.create")
  .audit({ event: "bbs.post.create" })
  .handle(function (req, res, ctx) {
    // ctx.actor exists only after host enforcement.
  });
~~~

modules/express/auth_builders.go:20-204 compiles this chain into a RoutePlan.
Registration fails unless a route explicitly declares public or authenticated
security. An authenticated route must declare allow(action) before handle.

pkg/gojahttp/enforcer.go:62-156 validates and enforces the plan in this order:

~~~text
validate route plan
    -> authenticate
    -> enforce CSRF for unsafe session requests
    -> resolve declared resources
    -> authorize actor + action + resources
    -> create secure context
    -> dispatch to the VM owner
    -> emit audit outcome
~~~

pkg/gojahttp/planned_dispatch.go:33-70 calls JavaScript only after that envelope
succeeds. Production errors are not copied directly into responses. This is the
correct boundary: JavaScript consumes an authorization decision; it does not
parse cookies, validate OIDC, or hold provider credentials.

### 5.3 Public signup is not current xapp behavior

pkg/idpaccounts.Service.Create is the correct low-level identity mutation. It
normalizes a login, checks duplicates, generates an ID and subject, validates
the user, hashes the password, and atomically inserts user plus credential.
That method is intentionally not a safe public HTTP endpoint by itself.

examples/tinyidp-message-app/app_http.go:216-480 demonstrates the missing HTTP
controls:

- a one-use pre-session registration attempt,
- a CSRF value bound to an HttpOnly cookie,
- same-origin validation using Origin and Sec-Fetch-Site,
- body size and media-type validation,
- rate limits by client address and normalized-login hash,
- generic error responses that resist account enumeration,
- durable account creation and audit,
- and an application session created only after commit.

That example explicitly does not mint a tiny-idp browser SSO cookie. The
production xapp mounts no equivalent registration route. Therefore “users can
sign up” is a missing product surface.

### 5.4 Proposed signup ownership

The central issuer should own a native /register workflow because the created
object is an issuer identity shared across applications. Successful signup
should redirect into the normal OIDC authorization flow. It should not create
sessions in every relying party and should not grant any application role,
tenant, or membership.

~~~text
POST /register
  validate one-use attempt + CSRF + same-origin
  apply address and normalized-login rate limits
  validate password policy and required profile fields
  account = accounts.Create(...)
  optionally create verification challenge
  audit identity.registration.accepted
  redirect to the pending OIDC authorization interaction

application callback
  verify OIDC result
  upsert external identity by (issuer, sub)
  evaluate app-specific onboarding policy
  create app session with only existing or explicitly granted memberships
~~~

Decision required: whether email ownership verification is mandatory before
login, before sensitive actions, or not in the first release. Password recovery,
email changes, account deletion, disabled-user support, abuse escalation, and
privacy retention must be specified with the signup release. Public signup
without recovery and abuse policy is not a complete account lifecycle.

## 6. Coding-agent device authorization flow

### 6.1 Current sequence

~~~text
Coding agent          tiny-idp            Browser/human       Resource server
    | POST /device_authorization                |                    |
    | client_id, scope, audience                 |                    |
    |------------------------------------------->|                    |
    | device_code, user_code, URI                |                    |
    |<-------------------------------------------|                    |
    |                                            | GET /device       |
    |                                            | login + code      |
    |                                            | review scopes     |
    |                                            | approve/deny      |
    | POST /token device_code                    |                    |
    |---------------------> transactional consume + token persist     |
    | access_token, ID token                     |                    |
    |<---------------------|                     |                    |
    | Authorization: Bearer opaque-token --------------------------->|
    |                                            |     POST /introspect
    |                                            |<-------------------|
    |                                            | active, sub, client,
    |                                            | scopes, aud, exp   |
    |                                            |------------------->|
    |<---------------- API response after scope + actor enforcement --|
~~~

Observed details:

- internal/fositeadapter/provider.go:438-556 accepts a form POST, authenticates
  the declared client as public or confidential, requires openid scope, checks
  client scopes and audiences, applies rate limits, and persists a ten-minute
  grant with a five-second poll interval.
- internal/fositeadapter/device_codes.go creates random device material and an
  eight-character human code excluding ambiguous characters. Only
  domain-separated HMAC hashes are stored.
- internal/fositeadapter/device_verification.go requires fresh password
  authentication and CSRF before an atomic approve or deny decision. It does
  not trust an existing provider session cookie as proof of fresh presence.
- internal/fositeadapter/device_token_handler.go maps pending, slow_down,
  denied, expired, and consumed states to OAuth errors. SQLite consumes the
  approved device grant and persists issued token state in one transaction.
- cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go authenticates the
  resource server to /introspect, verifies issuer, Bearer type, subject, expiry,
  audience, and scopes, and never sends the raw token to JavaScript.
- cmd/tinyidp-xapp/device_api.go maps the validated subject to a constrained
  actor and calls Durable Objects from Go.

### 6.2 Current lifecycle limitations

The xapp device client allows the device grant but not the refresh-token grant
and does not request offline_access. cmd/tinyidp-xapp/device_cli.go stores an
access token and expiry only. When it expires, the human repeats device
authorization. That can be an acceptable initial security posture, but it is
not suitable for unattended long-running agents unless interruption and
reauthorization are expected product behavior.

tiny-idp does not expose an RFC 7009 revocation endpoint. Password changes
revoke server-side user artifacts, but users and agents lack a standard
“disconnect this agent” operation. The resource server positively caches
introspection for at most 30 seconds by default, so disable or password-change
enforcement may lag by that amount at the API.

Decision required:

- Keep access-token-only agents with short lifetimes and explicit human
  reauthorization, or enable refresh tokens with rotation, reuse detection,
  bounded lifetime, secure local storage, and disconnect/revoke support.
- Define the acceptable revocation propagation time and configure the positive
  introspection cache accordingly.
- Decide whether users can list authorized clients or sessions and revoke one
  agent without changing their password.

### 6.3 Standards interoperability gap

RFC 8628 defines the device flow. RFC 8707 defines resource indicators using the
request parameter resource. The current provider calls fosite.GetAudiences and
the xapp sends audience. This private extension is internally consistent and
tested, but generic OAuth device clients will not infer it from the standard.
Discovery also does not advertise resource indicator behavior.

The production design should accept and validate resource as the canonical
parameter, carry its value into the token aud claim and introspection response,
and reject ambiguous requests. If audience remains temporarily supported, that
would be compatibility work and needs an explicit product decision; this ticket
does not recommend silently adding an adapter.

Acceptance cases include:

- one resource parameter produces the exact allowed audience,
- repeated resource parameters follow RFC 8707 semantics only if deliberately
  supported,
- a resource not registered for the device client returns invalid_target,
- mixed resource and audience inputs fail closed,
- the approved resource is immutable between device creation and token issue,
- and the introspecting resource client sees active only for a shared allowed
  audience.

## 7. Proposed multi-application production architecture

The recommended deployment separates the issuer from the applications. This
keeps one identity database and one issuer namespace while allowing application
code, sessions, membership, and business data to remain independently owned.

~~~text
                         +----------------------------------+
                         | Central tiny-idp                 |
                         | https://id.example.com           |
                         |                                  |
       Browser OIDC ---->| /authorize /token /userinfo     |
       Device OAuth ---->| /device_authorization /device   |
       Introspection --->| /introspect                      |
                         | /register                        | proposed
                         | identity.sqlite + keys + audit   |
                         +----------------+-----------------+
                                          |
                        issuer/sub, tokens, introspection
             +----------------------------+---------------------------+
             |                            |                           |
  +----------v-----------+     +----------v-----------+    +----------v-----------+
  | xgoja app A          |     | xgoja app B          |    | xgoja app C          |
  | browser client A     |     | browser client B     |    | browser client C     |
  | resource client A    |     | resource client B    |    | resource client C    |
  | app session store A  |     | app session store B  |    | app session store C  |
  | app users/roles A    |     | app users/roles B    |    | app users/roles C    |
  | audience /apps/a/api |     | audience /apps/b/api |    | audience /apps/c/api |
  +----------+-----------+     +----------+-----------+    +----------+-----------+
             ^                            ^                           ^
             | Bearer                     | Bearer                    | Bearer
       coding agent A               coding agent B              coding agent C
~~~

### 7.1 Client inventory

For every application X, provision explicit records:

| Client role | Public | Grants | Important restrictions |
| --- | --- | --- | --- |
| browser-X | yes | authorization_code, optionally refresh_token | exact HTTPS callbacks, S256 PKCE, browser scopes only |
| device-X | yes | device_code, optionally refresh_token | exact API resources and agent scopes |
| resource-X | no | no end-user grant needed for introspection role | owner-only secret, CanIntrospect, exact allowed resources |

Do not reuse the browser client as the device client. Do not put an
introspection secret into a public client or xgoja script. Do not let one
resource client introspect all applications unless it is intentionally a shared
gateway and its blast radius is accepted.

### 7.2 Provisioning contract

The current xapp state manifest is version 2 and has scalar fields for one
browser client, device client, resource client, and audience. A multi-app
product needs an explicit declarative inventory or an operator workflow. Dynamic
client registration is explicitly unsupported and is not necessary for the
first release.

Proposed declarative shape:

~~~yaml
issuer: https://id.example.com
applications:
  - id: notes
    browserClient:
      id: notes-web
      redirectURIs:
        - https://notes.example.com/auth/callback
      postLogoutRedirectURIs:
        - https://notes.example.com/
      scopes: [openid, profile, email]
    api:
      resource: https://notes.example.com/api
      introspectionClient: notes-api
    deviceClient:
      id: notes-agent
      scopes: [openid, notes.read, notes.write]
      resources: [https://notes.example.com/api]
~~~

Reconciliation must be idempotent, reject widening changes unless an operator
explicitly approves them, avoid printing secrets, and audit client changes.
Secret generation and rotation must use owner-only files or a production secret
manager. No source in the xgoja application should know the raw resource
client secret.

## 8. Proposed planned bearer authentication for Express

### 8.1 Why current express.user is insufficient

The current SecurityMode has public and user semantics. Its Authenticator
returns a session-backed Actor. Route plans have no bearer credential kind,
OAuth issuer, audience/resource, scope requirement, token client identity, or
token expiry. Reusing express.user for device tokens would hide materially
different CSRF, credential transport, revocation, and audit semantics.

Do not add a compatibility adapter that makes a bearer token look like a browser
session. Add a first-class security mode if agent endpoints must be expressed in
the planned DSL.

### 8.2 Proposed route syntax

Keep browser and agent routes separate even when they invoke the same domain
service:

~~~javascript
app.post("/api/browser/bbs/posts")
  .auth(express.user().required())
  .csrf()
  .resource("board", { id: "main" })
  .allow("bbs.post.create")
  .audit({ event: "bbs.browser.post.create" })
  .handle(createPost);

app.post("/api/agent/bbs/posts")
  .auth(express.oauth()
    .issuer("https://id.example.com")
    .resource("https://bbs.example.com/api")
    .scopes("bbs.post.create"))
  .resource("board", { id: "main" })
  .allow("bbs.post.create")
  .audit({ event: "bbs.agent.post.create" })
  .handle(createPost);
~~~

This syntax is a design sketch, not an existing API. Exact builder naming should
be settled in the go-go-goja implementation ticket.

The agent route intentionally has no CSRF declaration. Bearer Authorization
headers are not cookie authentication. It must reject cookies as a substitute
for the token. Conversely, the browser route must not accept a bearer token as a
substitute for the application session.

### 8.3 Proposed host interfaces

~~~go
type OAuthRequirement struct {
    Issuer   string
    Resource string
    Scopes   []string
}

type TokenPrincipal struct {
    Issuer    string
    Subject   string
    ClientID  string
    Scopes    []string
    Resources []string
    ExpiresAt time.Time
}

type TokenAuthenticator interface {
    AuthenticateToken(
        ctx context.Context,
        req *http.Request,
        requirement OAuthRequirement,
    ) (TokenPrincipal, error)
}
~~~

The host implementation should wrap the existing resourceauth logic or a
generalized equivalent. It must:

- reject zero or multiple Authorization headers,
- accept one strict Bearer scheme and forbid query/form token transport,
- introspect over TLS with a Go-owned secret,
- verify exact issuer, positive expiry, resource, scopes, and Bearer token type,
- represent unavailable provider separately from invalid credentials,
- keep raw tokens out of actors, errors, metrics, cache keys, and audit,
- bind the token client ID into actor claims or a typed principal,
- and run the normal resource resolver and authorizer before JavaScript.

Add a compile-time interface assertion for each Go implementation.

### 8.4 Actor semantics

The actor should identify both the user and credential context:

~~~json
{
  "kind": "oauth-user",
  "id": "stable-idp-subject",
  "claims": {
    "issuer": "https://id.example.com",
    "clientId": "bbs-agent",
    "scopes": ["bbs.post.create"],
    "resource": "https://bbs.example.com/api",
    "credential": "bearer"
  }
}
~~~

The raw access token is never a claim. Application authorization can then
express rules such as “this user may post to this board and this client is an
approved automation client.” The OAuth scope is a coarse grant ceiling; the
application authorizer still checks the user, action, and resolved resource.

## 9. Authorization model

Authentication answers who presented a valid credential. OAuth scope answers
which coarse API capabilities the human approved for the client. Application
authorization answers whether this application user may perform one action on
one resource now.

The required order for an agent route is:

~~~text
valid opaque access token
  AND token issuer == configured issuer
  AND token audience includes exact route resource
  AND token scopes include every route-required scope
  AND token subject maps to an enabled application user
  AND token client is accepted for this route if client policy is configured
  AND application authorizer allows actor + action + resolved resources
~~~

Do not copy IdP groups or roles directly into permanent application
authorization without a synchronization and revocation contract. Current
hostauth correctly projects application-owned memberships. Keep that ownership.

For Durable Objects, derive the owner or actor binding only after all checks.
The current xapp binds browser objects from gojahttp.ActorFromContext and agent
objects from a Go-constructed subject actor. A generalized planned bearer mode
must preserve the same invariant.

## 10. API reference

### 10.1 Current tiny-idp endpoints

Routes are registered in internal/fositeadapter/provider.go:401-436.

| Endpoint | Method | Caller | Purpose |
| --- | --- | --- | --- |
| /.well-known/openid-configuration | GET | RP or agent | OIDC/OAuth metadata |
| /jwks | GET | RP | Public RS256 verification keys |
| /authorize | GET, POST | Browser | OIDC authorization interaction |
| /device_authorization | POST form | Device client | Start device grant |
| /device | GET, POST | Browser | Enter code, authenticate, approve/deny |
| /token | POST form | RP or device client | Exchange code, refresh, or device grant |
| /userinfo | GET, POST | RP backend | Resolve OIDC subject claims |
| /introspect | POST form + Basic | Confidential resource server | Validate opaque access token |
| /end-session | GET, POST | Browser | Revoke current browser session and redirect |
| /healthz | GET | Probe | Process liveness |
| /readyz | GET | Probe | Signing-key readiness |

Discovery advertises authorization code, refresh token, and device code grants;
RS256; S256; public subjects; Basic introspection authentication; and the
openid, profile, email, and offline_access scopes
(internal/oidcmeta/discovery.go:23-46).

### 10.2 Current xapp native endpoints

| Endpoint | Security owner | Notes |
| --- | --- | --- |
| /auth/login | go-go-goja hostauth | Starts OIDC state, nonce, and PKCE |
| /auth/callback | go-go-goja hostauth | Verifies result and creates app session |
| /auth/logout | go-go-goja hostauth | POST, CSRF-protected local logout |
| /auth/session | go-go-goja hostauth | Returns app session projection |
| /api/device/bbs | tinyidp-xapp Go handler | Requires bbs.read |
| /api/device/bbs/posts | tinyidp-xapp Go handler | Requires bbs.post.create |
| /api/me, /api/bbs, /api/object | Express plan | Browser application session |

### 10.3 Important Go APIs

- embeddedidp.Options and Options.Validate construct strict or mock providers.
- embeddedidp.Bootstrap reconciles client records and signing keys.
- idpaccounts.Service.Create is the durable account-creation primitive.
- idpaccounts.Service.AuthenticatePassword owns password and lockout policy.
- idpstore.Store is the identity security-state boundary.
- resourceauth.Authenticator is the current xapp RFC 7662 client.
- gojahttp.Authenticator, ResourceResolver, Authorizer, CSRFProtector, and
  AuditSink are the host-owned Express enforcement services.

## 11. Threat model and trust boundaries

### 11.1 Assets

- Password hashes and account recovery channels.
- Stable user subjects and profile data.
- Signing private keys and token hashing secret.
- Browser sessions and OIDC transaction state.
- Authorization codes, device codes, access tokens, and refresh tokens.
- Confidential introspection client secrets.
- Application memberships, roles, tenant relationships, and Durable Object
  data.
- Audit records needed for incident reconstruction.

### 11.2 Trust boundaries

- The public internet to the direct TLS listener or approved reverse proxy.
- Browser to IdP login and consent forms.
- Browser to application callback and session endpoints.
- Coding agent to device and token endpoints.
- Resource server to introspection using confidential Basic credentials.
- Go host to xgoja VM.
- Application actor to Durable Object instance.
- Process to SQLite, secret files, backups, and audit files.

### 11.3 Required abuse cases

Review and test at least:

- account enumeration through signup, login, recovery, or registration timing;
- password spraying across accounts, clients, and source addresses;
- device user-code brute force and approval race/replay;
- phishing through untrusted verification URIs or misleading client names;
- device polling faster than the required interval;
- redirect URI, post-logout URI, issuer-path, and proxy-host confusion;
- CSRF on signup, login continuation, consent, logout, and browser mutations;
- authorization code interception or replay without the PKCE verifier;
- two Authorization headers or mixed token transports;
- token substitution across issuers, resources, clients, or users;
- stale positive introspection cache after disable, password change, or revoke;
- leakage of tokens or secrets into JavaScript, audit, logs, URLs, crash dumps,
  or metrics;
- application auto-provisioning that accidentally grants a tenant or role;
- one application's resource client introspecting another application's token;
- a malicious xgoja handler trying to fabricate or overwrite ctx.actor;
- backup restore with missing signing keys, stale schema, or permissive file
  ownership;
- and audit-sink failure after an account transaction commits.

## 12. Prioritized production gaps

### P0: required before the described product launches

1. **Issuer-owned public signup and lifecycle contract.**
   Implement the controlled registration workflow, define verification and
   recovery policy, and test enumeration, CSRF, replay, rate limiting, duplicate
   races, and post-commit audit behavior.

2. **Multi-application client/resource provisioning.**
   Replace the one-xapp scalar manifest as the product boundary with an explicit
   inventory and reconciliation workflow. Test least privilege and rejected
   widening changes.

3. **Standards-aligned resource indicator.**
   Use RFC 8707 resource for device authorization, discovery/documentation, and
   tests. Make any support for the existing audience parameter an explicit
   compatibility decision.

4. **First-class planned OAuth bearer auth, if Express syntax is mandatory.**
   Add a distinct security mode and host service. Keep the confidential
   credential and raw token in Go. Do not disguise it as express.user.

5. **Coding-agent lifecycle policy.**
   Choose access-token-only reauthorization or reviewed refresh tokens. Define
   disconnect/revocation, secure local token-cache requirements, maximum
   lifetimes, and cache propagation.

6. **Production deployment evidence.**
   Exercise TLS, owner-only state, backups and restore, maintenance, rate
   limits, audit collection, key rotation, resource-client rotation, disk
   failure, process restart, and clock behavior in the real environment.

### P1: required for reliable operations or broader rollout

1. Add an RFC 7009-style revocation or equivalent user-visible disconnect
   operation.
2. Add user/admin views for active grants, device clients, and relevant audit
   history.
3. Resolve application OIDC transaction state before scaling an xgoja app past
   one replica.
4. Specify cross-application logout behavior. Local app logout, IdP browser
   logout, and agent token revocation are different operations.
5. Make documentation match current direct-TLS and implemented-device behavior.
6. Define secret/key rotation runbooks with overlap, rollback, and verification.
7. Define subject deletion, retention, and application orphan-handling policy.

### P2: deliberate later hardening

1. Sender-constrained tokens such as DPoP after the resource servers can enforce
   them end to end.
2. Active-active identity storage after selecting and validating a different
   concurrency topology.
3. Dynamic client registration only if operator-managed provisioning becomes a
   demonstrated bottleneck.
4. Additional authentication factors, WebAuthn, federation, or enterprise
   lifecycle protocols as separate product features.

## 13. Implementation plan

### Phase 0: freeze product decisions

- Assign the issuer URL and one resource URI per application.
- Decide whether signup requires verified email.
- Decide agent token lifetime, refresh, and disconnect semantics.
- Decide whether agent endpoints must be Express planned routes in release one.
- Fix the single-active IdP and single-replica app assumptions in the deployment
  contract.

Exit criterion: a reviewed decision record answers every item and defines
measurable acceptance tests.

### Phase 1: registration

Primary tiny-idp files:

- Add a native registration handler and route near provider interaction
  composition, using idpui for rendering.
- Reuse idpaccounts.Service.Create rather than duplicating account mutation.
- Extract or reproduce the controls demonstrated in
  examples/tinyidp-message-app/app_http.go without coupling the provider to that
  example.
- Add persistent one-use attempt state if production restart/replay semantics
  require it.
- Add focused HTTP, account-store, race, and audit failure tests.

Exit criterion: two unrelated users can sign up without enumeration, obtain
separate subjects, enter normal OIDC, and receive no application grant merely
from registration.

### Phase 2: multi-app provisioning

- Define a versioned application/client inventory.
- Reconcile browser, device, and resource client roles.
- Generate and store resource secrets without stdout exposure.
- Add admin diff, dry-run, narrowing, rotation, and audit behavior.
- Verify exact callbacks, logout redirects, resources, scopes, and grant types.

Exit criterion: two applications have distinct clients, app stores, resources,
and secrets; cross-app callback, introspection, and token substitution fail.

### Phase 3: resource indicator and agent lifecycle

- Change the device authorization contract to RFC 8707 resource.
- Update CLI settings, tests, examples, and discovery documentation.
- Implement the chosen refresh or reauthorization policy.
- Add disconnect/revoke behavior and cache invalidation expectations.
- Test device denial, expiry, slow_down, replay, client mismatch, resource
  mismatch, refresh rotation/reuse, and revocation.

Exit criterion: a standards-aware CLI can discover the issuer, authorize a
human, and call only its approved resource and scopes.

### Phase 4: planned bearer routes

Primary go-go-goja files:

- modules/express/auth_builders.go for the explicit OAuth builder.
- pkg/gojahttp/auth_plan.go for the typed security requirement.
- pkg/gojahttp/enforcer.go for token authentication before resources and JS.
- pkg/gojahttp/planned_dispatch.go to preserve minimal actor exposure.
- pkg/xgoja/hostauth or a new host resource-auth composer for Go-owned
  introspection.

Primary tiny-idp-xapp files:

- Generalize cmd/tinyidp-xapp/internal/resourceauth without leaking its secret.
- Replace or complement device_api.go with planned agent routes.
- Keep browser and agent namespaces separate.
- Add integration tests showing both routes invoke the same domain object with
  distinct credential enforcement.

Exit criterion: an agent route cannot register without resource, scopes,
allow(action), and audit policy; invalid credentials never invoke JavaScript.

### Phase 5: production exercise

- Deploy the standalone issuer and at least two applications on staging names.
- Run signup, OIDC, logout, device, introspection, password-change, disable,
  revoke, backup/restore, restart, key rotation, and secret rotation scenarios.
- Capture audit events and verify no credential material is recorded.
- Run go test ./..., go build ./..., focused race tests, static analysis, and the
  repository's conformance plan.
- Review the final threat model with someone other than the implementer.

Exit criterion: the release evidence links each production claim to a passing
scenario, code location, operational artifact, and owner.

## 14. Test and review matrix

| Area | Essential test | Expected invariant |
| --- | --- | --- |
| Signup | simultaneous same-login registration | one account commits; both responses do not enumerate |
| Signup | reused attempt/CSRF | second request fails without account mutation |
| Browser OIDC | wrong state, nonce, issuer, callback | no app session |
| Browser OIDC | two issuers with same textual sub | two distinct app identities |
| Express | missing public/auth or missing allow | route registration fails |
| Express | failed auth/resource/policy | JavaScript invocation count remains zero |
| Device | wrong client, code, resource, scope | no grant or inactive token |
| Device | parallel approved-code redemption | exactly one token transaction commits |
| Polling | early repeated polls | slow_down and increased interval |
| Introspection | wrong resource-client secret | 401 without token oracle |
| Introspection | client lacks shared resource | active false |
| API | duplicate Authorization headers | 401; no JS or object dispatch |
| API | IdP unavailable | 503, never downgrade to anonymous |
| Revocation | password change/disable/disconnect | denied within stated cache bound |
| Multi-app | token for A sent to B | denied by exact resource |
| Multi-app | app A secret sent to app B | authentication or resource check fails |
| Durable Objects | two user subjects | isolated actor-bound object behavior |
| Operations | restore verified backup | ready only with current schema and usable key |
| Audit | sink failure around commits | documented post-commit result; no duplicate mutation |

Current evidence already includes:

~~~text
go test ./...                                                    PASS
go build ./...                                                   PASS
go test -race ./internal/fositeadapter ./cmd/tinyidp-xapp -count=1 PASS
~~~

These commands were run on 2026-07-18 from the tiny-idp repository. They are a
baseline, not evidence for unimplemented signup or generic multi-app behavior.

## 15. Intern code-review walkthrough

Use this order for a first review:

1. Run the ticket script scripts/01-evidence-map.sh to locate the symbols.
2. Read pkg/idpstore/types.go and validate.go to learn the durable domain and
   client capability checks.
3. Read pkg/idpaccounts/accounts.go and password.go. Write down transaction
   boundaries and every path that changes revocation state.
4. Read pkg/embeddedidp/options.go and bootstrap.go. Separate construction gates
   from runtime behavior.
5. Read internal/fositeadapter/provider.go route registration, device creation,
   token endpoint, and introspection in that order.
6. Read device_verification.go and device_token_handler.go. Confirm that fresh
   human authentication, approval, one-use consumption, and token persistence
   occur at the expected boundaries.
7. Read cmd/tinyidp-xapp/state.go, production_app.go, development_app.go, and
   device_api.go to see how the vertical slice is composed.
8. Switch to go-go-goja and trace auth_builders.go to auth_plan.go, enforcer.go,
   and planned_dispatch.go.
9. Trace oidcauth.go and hostauth/builder.go to find the application-session and
   identity-normalization boundary.
10. Read examples/tinyidp-message-app/app_http.go registration code and list
    which controls belong at the future issuer endpoint.
11. Review tests before proposing changes. Existing negative tests often encode
    security decisions not obvious in public docs.
12. Compare every proposed endpoint or builder with the P0 acceptance cases in
    this document.

For each pull request, ask:

- What credential is accepted, and which component owns its validation?
- What exact issuer, client, resource, scope, subject, and expiry checks occur?
- Can invalid input reach JavaScript or a Durable Object?
- Is state one-use, durable, atomic, and safe across restart?
- Which secret could enter logs, URLs, errors, audit, or actors?
- Does this change widen a client, redirect, resource, scope, or role?
- Are browser cookie and bearer-token CSRF semantics kept separate?
- Is application membership still application-owned?
- What happens when SQLite, audit, introspection, DNS, TLS, or the clock fails?
- Is the failure response generic without becoming an authentication oracle?
- Does the test prove the negative path and zero downstream invocation?

## 16. Alternatives considered

### Embed an issuer in every xgoja app

This matches the current vertical slice but creates one issuer and identity
database per app, fragments user accounts and SSO, duplicates key and account
operations, and makes agent trust configuration harder. It is useful for demos
and isolated products, not the proposed shared identity plane.

### Put all applications behind one giant xgoja process

This reduces provisioning but couples deployments, secrets, failures, sessions,
and business data. It creates a larger privilege and incident radius. Separate
applications with one central issuer provide a clearer boundary.

### Use self-contained JWT access tokens

JWT validation removes introspection availability and latency but makes
immediate revocation, resource-client isolation, and key/audience policy more
complex. The current opaque-token implementation is already strong and keeps
token status centralized. Retain it for the first release.

### Let JavaScript introspect tokens

This exposes a confidential resource secret and raw bearer material to a
dynamic runtime and duplicates security logic across apps. It violates the
existing host-enforcement design. Reject it.

### Reuse express.user for bearer tokens

This hides different transport, CSRF, cache, audience, and client semantics and
would be a compatibility adapter. Add a first-class mode instead.

### Accept session and bearer credentials on one route

This creates precedence and confused-deputy questions, especially around CSRF
and audit. Separate browser and agent URL namespaces can share the same domain
service without sharing a credential parser. Prefer separation.

### Enable refresh tokens immediately

Refresh tokens improve unattended operation but create long-lived credential
storage, rotation, reuse, revocation, and user-control obligations. Access-token
only operation is safer until those obligations are implemented. Product
requirements may still make refresh a P0 item.

## 17. Open decisions

- Is verified email required, and what recovery channel is trusted?
- Can any registered identity enter every app, or does each app require an
  invitation, membership approval, or tenant join?
- Are coding agents expected to run unattended beyond one access-token
  lifetime?
- What is the maximum acceptable revocation delay at a resource server?
- Is one device client defined per app, per CLI distribution, or per trust
  domain?
- Must coding-agent routes use the Express DSL in release one, or is the current
  Go-owned handler boundary acceptable temporarily?
- Is support for the private audience parameter required during a migration to
  RFC 8707 resource? This is a compatibility decision and must not be assumed.
- Is the single-active IdP and single-replica app topology acceptable for the
  initial production service-level objective?
- Which system owns application invitations and tenant membership?
- Which audit events and retention periods are required for user support and
  incident response?

## 18. Source map

### tiny-idp

- README.md:1-14 distinguishes mock and strict provider profiles.
- pkg/embeddedidp/options.go:82-235 defines strict construction gates.
- pkg/embeddedidp/bootstrap.go:32-47 defines browser and device client shapes.
- pkg/idpaccounts/accounts.go:16-95 owns account creation.
- pkg/idpaccounts/password.go:120 onward owns password authentication and
  bounded hashing.
- pkg/sqlitestore/store.go:57 onward opens and migrates the production store.
- internal/fositeadapter/provider.go:285-330 configures strict Fosite behavior.
- internal/fositeadapter/provider.go:401-436 registers provider endpoints.
- internal/fositeadapter/provider.go:438-556 starts device authorization.
- internal/fositeadapter/provider.go:1023-1080 issues tokens.
- internal/fositeadapter/provider.go:1139-1282 implements introspection.
- internal/fositeadapter/device_codes.go protects device and user codes.
- internal/fositeadapter/device_verification.go owns fresh human approval.
- internal/fositeadapter/device_token_handler.go owns one-use redemption.
- internal/oidcmeta/discovery.go:23-46 publishes discovery metadata.
- internal/cmds/serve_production.go:51-205 composes the direct TLS server.
- cmd/tinyidp-xapp/state.go:49-172 owns the current one-app manifest.
- cmd/tinyidp-xapp/production_app.go:21-149 composes the production xapp.
- cmd/tinyidp-xapp/development_app.go:244-323 mounts all xapp boundaries.
- cmd/tinyidp-xapp/app/routes/site.js declares browser planned routes.
- cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go validates opaque
  bearer tokens.
- cmd/tinyidp-xapp/device_api.go exposes the current Go agent API.
- cmd/tinyidp-xapp/device_cli.go:156-253 performs discovery and device polling.
- cmd/tinyidp-xapp/phase5_test.go:20-97 covers users, scopes, audience, headers,
  and password revocation.
- examples/tinyidp-message-app/app_http.go:216-480 demonstrates signup controls.
- docs/security-profile.md records supported and unsupported strict features.
- examples/production-host/README.md records single-writer deployment limits.

### go-go-goja

- modules/express/auth_builders.go:20-204 builds public and user route security.
- pkg/gojahttp/auth_plan.go:38-155 defines plans, actors, and host services.
- pkg/gojahttp/enforcer.go:62-156 enforces before JavaScript.
- pkg/gojahttp/planned_dispatch.go:33-70 performs post-enforcement VM dispatch.
- pkg/gojahttp/auth/oidcauth/oidcauth.go:21-252 owns OIDC RP behavior.
- pkg/gojahttp/auth/sessionauth/sessionauth.go owns application sessions/CSRF.
- pkg/xgoja/hostauth/builder.go:111-246 composes native auth and normalizes
  identities.
- examples/xgoja/21-generated-host-auth/verbs/sites.js is the concrete planned
  auth example.

## 19. Protocol references

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
  defines ID tokens, UserInfo, nonce, and relying-party validation.
- [RFC 7636: PKCE](https://www.rfc-editor.org/info/rfc7636/) defines the code
  challenge and verifier used by public browser clients.
- [RFC 8628: OAuth Device Authorization Grant](https://www.rfc-editor.org/info/rfc8628/)
  defines device codes, user codes, verification, polling, and slow_down.
- [RFC 7662: Token Introspection](https://www.rfc-editor.org/info/rfc7662/)
  defines authenticated resource-server token status queries.
- [RFC 8707: Resource Indicators](https://www.rfc-editor.org/rfc/rfc8707.html)
  defines the resource request parameter and target binding.
- [RFC 9700: OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/info/rfc9700/)
  is the current OAuth security baseline.
- [RFC 7009: Token Revocation](https://www.rfc-editor.org/info/rfc7009/)
  defines the standard revocation endpoint relevant to agent disconnect.

## 20. Final recommendation

Proceed with tiny-idp as the identity foundation, with the current xapp treated
as a security reference and integration test. Do not label the result a generic
production multi-user xgoja identity platform until the P0 items are complete.

The shortest credible route is one central, single-active issuer; operator-
managed per-app clients and resources; issuer-owned signup; one replica per
xgoja app initially; opaque token introspection; short-lived access-token-only
agents unless refresh is explicitly required; and separate browser/agent route
namespaces. If agent routes must use go-go-goja Express auth syntax, implement a
new first-class planned OAuth mode in Go and preserve the rule that no
credential-bearing request reaches JavaScript before host enforcement.
