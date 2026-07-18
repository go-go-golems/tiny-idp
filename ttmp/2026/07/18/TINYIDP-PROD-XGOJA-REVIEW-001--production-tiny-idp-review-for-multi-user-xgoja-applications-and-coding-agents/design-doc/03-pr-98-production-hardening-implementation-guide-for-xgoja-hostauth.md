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

## The project around PR 98

PR 98 lives in a general-purpose runtime repository, but it was created to
support a concrete product direction. The implementer needs both views. The
runtime view explains where reusable code belongs. The product view explains
why the optional authentication subsystem has these requirements.

The product direction is to let people use small multi-user web applications
and later authorize local coding agents or command-line tools to call those
applications' APIs. A person should be able to create an account, sign in with
a browser, use an application, approve a device shown by a CLI, and later
disconnect that device. The first deployment is intentionally smaller; the
device and multi-application pieces come after browser signup and login are
running reliably.

This section introduces every system named in the rest of the guide. Read it
before opening `device_handlers.go`.

### The systems in one table

| System | What it is | What it owns | What it does not own |
|---|---|---|---|
| tiny-idp | A small Go OIDC/OAuth identity provider. | Accounts, passwords, signup, browser authentication, OIDC authorization, signing keys, provider sessions, native device grants, and opaque OAuth tokens. | Application messages, application sessions, xgoja route policy, or `ggat_` credentials. |
| Message Desk | The initial standalone multi-user messaging example in the tiny-idp repository. | Messages, application users, local browser sessions, and its OIDC relying-party behavior. | Passwords in the external-IdP design and general xgoja runtime behavior. |
| go-go-goja | A general-purpose collection of Goja modules, runtime infrastructure, generated-host tooling, and examples. | Reusable JavaScript-to-Go modules, xgoja runtime composition, planned HTTP route enforcement, and optional host capabilities such as `hostauth`. | The identity of every application using it, the cluster, or the tiny-idp provider database. |
| xgoja | The generated-runtime layer built from go-go-goja providers and specifications. | Selection and composition of modules into concrete binaries and commands. | A universal product policy; each generated application selects its own modules and routes. |
| hostauth | An optional go-go-goja/xgoja subsystem used by generated HTTP hosts. | Local application sessions, OIDC relying-party state, local users, agents, application API tokens, device approvals, and auth enforcement inputs. | tiny-idp accounts and tiny-idp-issued OAuth token state. |
| Personal Knowledge Inbox | A progressive xgoja reference application used to exercise the runtime. | Example inbox records, example browser routes, and an example programmatic capture API. | Cluster-wide identity policy or production infrastructure. |
| Hetzner k3s cluster | The deployment platform. | Scheduling pods, networking Services, Traefik ingress, certificates, persistent volumes, secret delivery, and GitOps reconciliation. | OAuth semantics and application authorization decisions. |

The table is not merely vocabulary. It tells you where a proposed change
belongs. A trusted-proxy resolver belongs in reusable host HTTP infrastructure.
An inbox action such as `user.self.read` belongs to an application policy. A
password-reset flow belongs to tiny-idp. A Traefik NetworkPolicy belongs in the
cluster GitOps repository.

## What tiny-idp is

Tiny-idp is an identity provider implemented in Go. In OIDC terminology it is
the OpenID Provider. An application that sends a browser to tiny-idp for login
is a Relying Party. In OAuth terminology tiny-idp can also act as the
authorization server that issues tokens to public or confidential clients.

For the project, tiny-idp has three distinct jobs.

### Tiny-idp owns human accounts

Tiny-idp stores a user's stable subject identifier, login name, password hash,
profile attributes, account state, and provider-side sessions. Public signup
belongs here because the password should enter the identity provider, not each
application that happens to use the identity.

~~~text
signup form
  -> tiny-idp validates CSRF and request budget
  -> tiny-idp validates username and password policy
  -> tiny-idp hashes the password
  -> tiny-idp stores the account
  -> tiny-idp continues the pending OIDC interaction
~~~

The stable OIDC `sub` claim is the important application identifier. An email
address or display name can change. A local application user should be linked
to the verified issuer and subject, not recreated whenever profile text
changes.

### Tiny-idp authenticates browser users for applications

An application does not receive the user's password. It redirects the browser
to tiny-idp with an authorization request. Tiny-idp authenticates the user and
returns an authorization code to the application's exact registered callback.
The application exchanges that code using PKCE and verifies the returned ID
token.

~~~text
Browser             Application                 tiny-idp
   | GET /auth/login     |                          |
   |-------------------->|                          |
   |                     | state + nonce + PKCE     |
   |                     |------------------------->|
   |<-------------------- browser login ------------|
   |--------------------- credentials ------------->|
   |<---------------- callback with code -----------|
   |                     | exchange code + verifier |
   |                     |------------------------->|
   |                     |<------ verified tokens --|
   |<---- local application session cookie ---------|
~~~

The last cookie is owned by the application. Logging out of one application
session and ending the provider session are related but separate operations.

### Tiny-idp also has its own device and introspection protocols

Current tiny-idp can issue opaque OAuth access tokens through its native device
authorization flow. It advertises an authenticated RFC 7662 introspection
endpoint so a registered resource server can ask whether such a token is
active and receive its issuer, subject, client, scopes, audience, and expiry.

This capability matters for the future multi-application design. One
tiny-idp-issued credential could be intended for a declared API audience and
validated by a separate application. PR 98 does not implement that integration
inside go-go-goja. Its application-owned `programauth` path is a different
credential system.

## What the initial Message Desk project is

The first production project is not the complete xgoja device platform. It is
a deliberately small deployment containing:

1. One standalone tiny-idp process with public signup.
2. One standalone Message Desk process using tiny-idp for browser OIDC login.

Message Desk is the BBS-like messaging application under
`examples/tinyidp-message-app` in the tiny-idp repository. It owns messages and
application sessions. In external OIDC mode it trusts tiny-idp for identity.
The first release proves that a new user can sign up, sign in, and use a real
multi-user application through the public cluster.

~~~text
Internet
   |
   +-- https://idp.example.test ------> tiny-idp
   |                                      |
   |                                      +--> identity SQLite/PVC
   |
   +-- https://messages.example.test -> Message Desk
                                          |
                                          +--> message/session SQLite/PVC
~~~

Device authorization, coding-agent access, multiple relying parties, and xgoja
applications are deferred from this first release. That scope decision is
important for the PR 98 implementer: PR 98 is preparing a later reusable xgoja
host capability. It must not block the initial Message Desk deployment, and the
initial deployment must not be used as proof that PR 98's device endpoints are
production-safe.

## What go-go-goja and xgoja are

Goja is a JavaScript implementation written in Go. Go-go-goja builds a larger
runtime system around it. The repository supplies native modules such as HTTP,
fetch, filesystem, database, and Express-like server APIs; runtime ownership
and provider infrastructure; command integration; generated-host support; and
examples showing how those pieces compose.

Go-go-goja is therefore not one server. It is reusable source code from which
many different binaries and applications can be built.

~~~text
go-go-goja repository
  |
  +-- runtime and event-loop infrastructure
  +-- native modules exposed through require(...)
  +-- provider registry and xgoja generation
  +-- HTTP route planning and Go-owned enforcement
  +-- optional hostauth provider
  +-- examples and tutorials
~~~

Xgoja specifications select providers and modules and generate a Go binary.
One specification might build a CLI with database and fetch modules. Another
might build a web application with Express routes, embedded assets, and
hostauth. Adding code to `hostauth` must not make every go-go-goja consumer an
OIDC application.

### Planned Express routes

The Express-like JavaScript syntax is a declaration layer. JavaScript declares
that a route accepts a browser session, an agent, or an explicit alternative,
and names the required local action. The Go host authenticates the request and
enforces that plan before invoking JavaScript.

~~~javascript
app.post("/api/programmatic/capture")
  .auth(express.agent())
  .allow("user.self.read")
  .audit("inbox.programmatic.capture")
  .handle((ctx, res) => {
    // Authentication and the action check already happened in Go.
    // ctx.auth is redacted; ctx.actor identifies the local agent.
  })
~~~

This division is intentional. Application authors can describe route policy in
the same file as the handler without receiving raw credential stores or
reimplementing Bearer parsing in JavaScript.

### Hostauth is optional infrastructure inside go-go-goja

`pkg/xgoja/hostauth` builds concrete auth services for a generated host. It can
create:

- local browser session services;
- OIDC login and callback handlers;
- local application user normalization;
- audit storage;
- capability-token storage;
- local programmatic agents and API tokens;
- application-owned device authorization; and
- the authenticator and authorizer inputs used by planned routes.

The provider is selected by an xgoja specification. A consumer that does not
select it should not pay its operational or policy cost.

This gives the implementer a placement rule:

- Generic request identity, auth-store health, and native handler composition
  belong in reusable Go host infrastructure.
- The list of actions that one application permits belongs in that
  application's hostauth configuration.
- UI wording and domain behavior belong in the application example or product.
- Tiny-idp provider changes belong in the tiny-idp repository.
- Traefik, PVC, Secret, and Deployment changes belong in the k3s GitOps
  repository.

## Why the Personal Knowledge Inbox exists

`examples/xgoja/23-personal-knowledge-inbox` is a progressive reference
application. Its steps begin with a small JavaScript verb and add HTTP serving,
SQLite, an API client, embedded UI assets, browser OIDC login, per-user data
isolation, and finally application-owned device authorization.

Step 08 demonstrates the product interaction PR 95 and PR 98 are intended to
support:

~~~text
1. Alice signs in through tiny-idp.
2. A CLI asks the xgoja application for a device code.
3. Alice enters the user code in the application's browser UI.
4. The local application session proves which user is approving.
5. The xgoja host creates an application agent owned by Alice.
6. The CLI receives ggat_ and ggrt_ credentials.
7. The CLI calls an express.agent() route.
8. The application stores the captured item under Alice's owner ID.
~~~

The example proves composition and account isolation. It is not itself the
production policy. Hard-coded actions, tutorial ports, memory defaults, and
test users must not silently become production configuration.

## What the cluster is

The deployment target is an existing single-node Hetzner k3s cluster. K3s is a
lightweight Kubernetes distribution. Kubernetes schedules containerized
processes, connects them through Services, restarts failed pods, mounts durable
storage, and evaluates health probes. It does not understand OIDC, refresh
tokens, or application ownership; those remain application responsibilities.

The platform uses a GitOps flow:

~~~text
source repository commit
  -> CI builds immutable container image
  -> image is published to GHCR
  -> GitOps repository updates the image reference
  -> Argo CD observes the Git change
  -> Argo CD reconciles k3s resources
  -> Traefik routes public HTTPS to the Service
~~~

The important cluster components are:

| Component | Responsibility |
|---|---|
| Traefik | Public HTTP ingress and TLS termination. |
| cert-manager | Obtaining and renewing public certificates used by ingress. |
| Argo CD | Reconciling the declared Git manifests into the live cluster. |
| Kustomize | Organizing and composing Kubernetes manifests. |
| Vault and Vault Secrets Operator | Delivering secrets without committing them to Git. |
| local-path storage | Providing node-local persistent volumes for single-node SQLite workloads. |
| Kubernetes Deployment | Running and restarting a declared number of application pods. |
| Kubernetes Service | Giving pods a stable private network endpoint. |
| NetworkPolicy | Restricting which cluster sources may connect to a pod. |

### Why TLS terminates at Traefik

The browser connects to an HTTPS public origin. Inside the cluster, Traefik can
forward private HTTP to the pod. This is the cluster's normal topology:

~~~text
browser -- HTTPS --> Traefik -- private HTTP --> application pod
~~~

The application must still know that its public origin is HTTPS so it creates
exact callback URLs and Secure cookies. It must also know whether Traefik is a
trusted source of the original client address. Those are two different
configuration facts:

~~~text
public-base-url: https://app.example.test
trusted proxy:   Traefik source CIDR(s)
listener:        private pod HTTP address
~~~

PR 98 adds the first fact. This guide asks the implementer to add the second.
The listener remains private HTTP; no development-mode exception is needed.

### Why the first supported topology has one replica

SQLite files are mounted from node-local persistent volumes, and PR 98's only
rate limiter is process-local memory. The supported first topology is therefore
one serving process. Kubernetes should use one replica and a Recreate rollout
when a SQLite PVC cannot be mounted safely by overlapping old and new pods.

~~~yaml
spec:
  replicas: 1
  strategy:
    type: Recreate
~~~

Moving auth records to PostgreSQL does not automatically make the service
multi-replica. Shared limiting, concurrency, migrations, and rolling upgrades
must also be designed and tested.

## How all the pieces cooperate in the later xgoja deployment

The future standalone xgoja application topology has two public origins and at
least two durable state owners:

~~~text
                            Hetzner k3s cluster

Internet browser
      |
      +-- HTTPS --> Traefik --> tiny-idp Service --> tiny-idp pod
      |                                             | accounts
      |                                             | passwords
      |                                             | provider sessions
      |                                             +--> IdP database
      |
      +-- HTTPS --> Traefik --> xgoja Service ----> xgoja pod
                                                    | local users
Coding agent                                       | app sessions
      |                                             | app agents/tokens
      +-- HTTPS --> Traefik ------------------------+--> app database
                                                    |
                                                    +--> JavaScript routes
~~~

A browser login crosses both services. An app-owned agent API request reaches
only the xgoja service after the credentials have been issued. A tiny-idp
account administration action reaches the provider, but its effect on existing
application-owned agents must be defined explicitly.

## Why PR 98 exists

PR 95 introduced the broad programmatic-auth capability: agent principals,
API and access tokens, device authorization, Express principal requirements,
guarded fetch, SQL stores, and the progressive inbox example. That established
the application programming model.

PR 98 addresses what happens when that model is run as a long-lived service:

- OIDC state must survive restarts between redirect and callback.
- Refresh-token rotation must be atomic.
- Users need a revocation path.
- Production configuration must reject memory-only auth state.
- Schema changes must be applied deliberately.
- Operators need readiness, audit, and security outcomes.

The changes in PR 98 are optional runtime infrastructure. They should be
implemented generically enough for different xgoja applications, while leaving
each application's action vocabulary and domain authorization local.

## What the PR 98 implementer is responsible for

The implementer should finish the reusable host boundary and the reference
application proof.

They are responsible for:

- a safe configuration model for hostauth;
- canonical interpretation of requests arriving through a trusted proxy;
- rate limits and policy checks on Go-owned auth endpoints;
- durable and atomic auth state transitions;
- truthful health and readiness behavior;
- owner-scoped application agent lifecycle services;
- redacted audit and metric integration points;
- tests that exercise restart, concurrency, failure, and abuse cases; and
- documentation that matches the actual tiny-idp and cluster contracts.

They are not responsible in this PR for:

- implementing the first Message Desk signup deployment;
- turning all go-go-goja applications into authenticated servers;
- adding a second general-purpose identity provider;
- implementing multi-replica hostauth;
- silently accepting tiny-idp-issued tokens as `programauth` tokens; or
- deciding every future application's permission vocabulary.

This boundary keeps the work reviewable. The remaining sections now explain
the concrete additions to PR 98 and why each is required.

### Working-directory map

The development machine contains several neighboring checkouts. Confirm the
current directory before running a command or committing a change.

| Path | Use |
|---|---|
| `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja` | PR 98 implementation checkout. Hostauth code changes and tests belong here. |
| `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp` | Tiny-idp checkout and this review ticket. Use it to verify provider behavior, not to implement go-go-goja hostauth. |
| `/home/manuel/code/wesen/2026-03-27--hetzner-k3s` | GitOps and live-cluster design repository. Kubernetes, Traefik, PVC, Argo, and NetworkPolicy work belongs here. |
| `/home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/infrastructure-and-release.md` | Infrastructure and release-system orientation. |

Start PR 98 code review with these go-go-goja paths:

~~~text
pkg/xgoja/hostauth/
pkg/gojahttp/auth/programauth/
pkg/gojahttp/auth/keycloakauth/
pkg/gojahttp/ratelimit.go
pkg/gojahttp/auth/audit/
examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/
~~~

Start product-context review with these tiny-idp paths:

~~~text
examples/tinyidp-message-app/
internal/oidcmeta/discovery.go
internal/fositeadapter/provider.go
cmd/tinyidp-xapp/internal/resourceauth/
~~~

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
