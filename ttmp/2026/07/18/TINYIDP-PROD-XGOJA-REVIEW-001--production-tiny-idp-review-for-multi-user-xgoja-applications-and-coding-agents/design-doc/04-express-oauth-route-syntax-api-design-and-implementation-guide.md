---
Title: Express OAuth route syntax API design and implementation guide
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
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/modules/express/auth_builders.go
      Note: Current fluent auth builders and RoutePlan compilation point
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/appauth/appauth.go
      Note: Current single-issuer local identity and application authorization model
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth/programauth/composite.go
      Note: Current app-owned bearer and browser-session credential selection
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/auth_plan.go
      Note: Current SecuritySpec AuthRequirement AuthResult and route validation contracts
    - Path: abs:///home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/pkg/gojahttp/enforcer.go
      Note: Host-owned enforcement order and auth requirement matching
    - Path: repo://cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go
      Note: Reference strict RFC 7662 verifier and bounded cache
    - Path: repo://ttmp/2026/07/18/TINYIDP-PROD-XGOJA-REVIEW-001--production-tiny-idp-review-for-multi-user-xgoja-applications-and-coding-agents/design-doc/01-production-idp-architecture-and-code-review-guide-for-xgoja-applications-and-coding-agents.md
      Note: Source review proposing the new Express OAuth syntax
ExternalSources:
    - https://github.com/go-go-golems/go-go-goja/pull/95
    - https://github.com/go-go-golems/go-go-goja/pull/98
    - https://www.rfc-editor.org/rfc/rfc6750
    - https://www.rfc-editor.org/rfc/rfc7662
    - https://www.rfc-editor.org/rfc/rfc8707
    - https://www.rfc-editor.org/rfc/rfc8628
Summary: Intern-facing API and implementation guide for adding issuer-issued OAuth bearer requirements to go-go-goja planned Express routes without conflating them with application-owned agents or browser sessions.
LastUpdated: 2026-07-18T17:15:00-04:00
WhatFor: Explain the proposed express.oauth().issuer().resource().scopes() contract, the Go route-plan and host-verifier changes behind it, and the tests required to prove fail-closed enforcement before JavaScript runs.
WhenToUse: When designing, implementing, or reviewing the go-go-goja follow-up that makes tiny-idp-issued device tokens usable through planned Express routes.
---


# Express OAuth route syntax API design and implementation guide

## Purpose and expected outcome

This guide describes a new authentication declaration for go-go-goja's
Express-like JavaScript API:

~~~javascript
app.post("/api/agent/boards/:boardId/posts")
  .auth(express.oauth()
    .issuer("https://id.example.com")
    .resource("https://bbs.example.com/api")
    .scopes("bbs.post.create"))
  .resource(express.resource("board").idFromParam("boardId"))
  .allow("bbs.post.create")
  .audit("bbs.agent.post.create")
  .handle(createPost);
~~~

The syntax is short because JavaScript is only declaring a route contract.
The difficult work remains in Go: strict bearer parsing, issuer discovery,
authenticated token introspection, resource and scope checks, identity
mapping, application authorization, redacted context construction, audit, and
error handling. None of those operations belongs in a JavaScript handler.

This document is written for an implementer who knows Go and JavaScript but
does not know the history of tiny-idp, xgoja, PR 95, or PR 98. It begins with
the system boundary, then derives the API rather than asking the reader to
copy a builder mechanically.

The intended result is a reusable go-go-goja capability. Applications may opt
into it when their host supplies an OAuth resource-server verifier. An
application that does not select hostauth or OAuth verification must continue
to use the general-purpose runtime without inheriting identity-provider
assumptions.

## 1. The project this API serves

### 1.1 go-go-goja is general-purpose runtime infrastructure

go-go-goja is not the Message Desk application, not tiny-idp, and not a
deployment. It is a collection of Goja runtime infrastructure and native
modules that Go programs can compose. The repository provides such things as
runtime ownership, module registration, generated xgoja binaries, HTTP and
Express-like APIs, filesystem and fetch modules, and optional host services.

That means the OAuth feature must remain optional and interface-driven. The
core Express module can understand an OAuth route requirement without knowing
which identity provider issues the token. A host may satisfy the requirement
with tiny-idp introspection, another RFC 7662 provider, or a test fake. The
Express module must not import tiny-idp.

### 1.2 xgoja composes a concrete executable

xgoja is the generated-host layer in go-go-goja. A project selects providers
and modules in configuration, generates a Go executable, and runs JavaScript
inside that host. One generated executable might include Express and
hostauth. Another might be a command-line tool with no HTTP server. The new
route syntax must therefore have two halves:

- A provider-neutral route-plan type and JavaScript builder in go-go-goja.
- Optional host wiring that supplies a configured OAuth verifier.

The builder can always exist. A route that declares OAuth must fail startup if
the selected host cannot enforce it. It must never silently become public,
session-authenticated, or handler-owned authentication.

### 1.3 tiny-idp is the issuer in our target deployment

tiny-idp is a small Go OpenID Connect and OAuth authorization server. In the
target system it owns user accounts and passwords, browser login, device
authorization, opaque access-token issuance, and RFC 7662 introspection.

The later coding-agent flow is:

~~~text
coding agent                  tiny-idp                    xgoja application
     |                           |                               |
     | POST device_authorization|                               |
     |-------------------------->|                               |
     | device_code + user_code   |                               |
     |<--------------------------|                               |
     |                           |                               |
     |       user opens verification URI and approves            |
     |                           |                               |
     | poll token endpoint       |                               |
     |-------------------------->|                               |
     | opaque access token       |                               |
     |<--------------------------|                               |
     |                                                           |
     | Authorization: Bearer opaque-token                        |
     |---------------------------------------------------------->|
     |                           |     authenticated introspection|
     |                           |<-------------------------------|
     |                           | active, iss, sub, client_id,   |
     |                           | scope, aud, exp, token_type    |
     |                           |------------------------------->|
     |                           |                               |
     |                     host authorization, then JS handler    |
~~~

The application owns a confidential introspection client. Its client secret
stays in Go-owned configuration and secret storage. The coding agent possesses
only the access token. JavaScript receives neither value.

### 1.4 The initial cluster deployment is narrower

The first production phase deploys one tiny-idp and one Message Desk instance
to the Hetzner k3s cluster. It needs public signup and browser OIDC login. The
native tiny-idp device flow, multiple applications, and coding agents are a
later phase.

This distinction affects scheduling, not the API's correctness. The OAuth
route work should be designed now so it can be added later without weakening
the browser path. It is not a release blocker for the first standalone
Message Desk deployment.

## 2. What PR 95 and PR 98 already provide

The proposed API starts from merged PR 95 and the current PR 98 branch. Do not
design as though the old `express.user()` route model were still the entire
system.

| Existing element | Meaning today | What the OAuth follow-up adds |
| --- | --- | --- |
| `express.user()` | Any authenticated user accepted by the configured authenticator. | No change; it remains a broad legacy/general declaration. |
| `express.sessionUser()` | A browser-session user, specifically `method=session` and `principalKind=user`. | No change; browser-only routes remain explicit. |
| `express.agent()` | An application-owned automation principal, independent of the exact API-token or app-owned access-token credential. | No change; do not reinterpret it as an IdP-issued device token. |
| `express.anyOf(...)` | Explicit alternatives among current user/agent auth requirements. | Do not add OAuth alternatives in the first implementation. Keep browser and IdP-bearer URLs separate. |
| `AuthResult` and `ctx.auth` | Redacted method, principal, credential hint, grants, and scopes. | Add typed OAuth verification context: issuer, subject, client, resource, expiry, and verified scopes. |
| `CompositeAuthenticator` | Selects app-owned API/access-token authentication when a Bearer header exists, otherwise a browser session. | Add route-directed selection of an external OAuth verifier. |
| `hostauth` | Optional generated-host sessions, OIDC login, app-owned agents, device flow, tokens, stores, authorization, and audit. | Add optional resource-server profiles and secret-owned introspection clients. |
| PR 98 | Makes hostauth transactions and token lifecycle more durable and operable. | Supplies a stronger base but does not implement the proposed `express.oauth()` requirement. |

PR 95's application-owned device flow issues `ggat_` access tokens for durable
application agent records. Tiny-idp's native device flow issues IdP tokens for
a human subject and OAuth client. Both use Bearer transport. They do not have
the same issuer, persistence, revocation boundary, or principal model.

That is the central design fact for this work:

~~~text
express.agent()
  asks: "Did the host authenticate an application-owned agent principal?"

express.oauth().issuer(...).resource(...).scopes(...)
  asks: "Did the host verify an issuer-issued OAuth access token satisfying
         this exact issuer, resource, and scope contract?"
~~~

One declaration is about a principal kind. The other is about a credential
and its authorization-server assertions. Do not implement the second as an
alias for the first.

## 3. The four decisions made for every protected request

Authentication code becomes easier to reason about when its questions remain
separate.

### 3.1 Credential authentication

Credential authentication answers whether the request presented one valid
credential. For the new route this is one strict `Authorization: Bearer ...`
header followed by authenticated introspection. It checks token activity,
issuer, token type, and expiry.

### 3.2 OAuth grant ceiling

The token's resource and scopes describe the maximum API authority approved
for that client. The route requires an exact resource indicator and a set of
scopes. Every required scope must be present.

### 3.3 Application identity and authorization

The token subject is an issuer identity, not automatically the application's
database key. The host resolves `(issuer, subject)` to an enabled local user.
The application authorizer then checks that user's current membership and the
requested action against resolved application resources.

### 3.4 JavaScript execution

The handler runs only after the first three decisions succeed. It receives a
minimal actor, verified non-secret OAuth context, resolved resources, and the
declared action. It never receives the bearer token or introspection secret.

The complete conjunction is:

~~~text
one syntactically valid Bearer credential
AND introspection service is reachable and authentic
AND token active == true
AND token_type == Bearer
AND token iss == route issuer
AND token exp is in the future
AND token aud contains route resource
AND token scopes contain every route-required scope
AND (issuer, sub) maps to an enabled application user
AND application authorizer allows actor + action + resources
THEN invoke JavaScript
~~~

No earlier success can compensate for a later failure.

## 4. The proposed JavaScript API

### 4.1 Canonical syntax

The first implementation should support this exact shape:

~~~javascript
const express = require("express");
const app = express.app();

app.post("/api/agent/boards/:boardId/posts")
  .auth(express.oauth()
    .issuer("https://id.example.com")
    .resource("https://bbs.example.com/api")
    .scopes("bbs.post.create"))
  .resource(express.resource("board").idFromParam("boardId"))
  .allow("bbs.post.create")
  .audit("bbs.agent.post.create")
  .handle(createPost);
~~~

The OAuth builder is required authentication. It does not need `.required()`.
All three OAuth methods are mandatory:

- `.issuer(value)` declares the exact authorization-server issuer.
- `.resource(value)` declares the exact RFC 8707 resource indicator expected
  in the access token's audience.
- `.scopes(...values)` declares the complete set of scopes the route requires.

The order of these three builder calls should not matter. Each may be called
once. Repeated calls should fail with a useful registration error rather than
silently replacing or accumulating security policy.

### 4.2 Browser and agent routes remain separate

The browser version of the same domain operation remains session-specific:

~~~javascript
app.post("/api/browser/boards/:boardId/posts")
  .auth(express.sessionUser())
  .csrf()
  .resource(express.resource("board").idFromParam("boardId"))
  .allow("bbs.post.create")
  .audit("bbs.browser.post.create")
  .handle(createPost);
~~~

Both routes may call `createPost`. They do not share credential parsing. The
browser route requires the application cookie and CSRF protection. The agent
route requires an IdP-issued bearer token and does not use cookie CSRF.

For the first OAuth implementation, reject this form:

~~~javascript
// Deliberately unsupported in the first implementation.
.auth(express.anyOf(
  express.sessionUser(),
  express.oauth().issuer(...).resource(...).scopes(...)
))
~~~

Mixed-credential routes create precedence, CSRF, status-code, and audit
questions. Separate URL namespaces are easier to review and test. This is an
intentional API boundary, not a missing compatibility shim.

### 4.3 Two different meanings of `resource`

The example contains two calls named `resource`, and they describe different
objects:

| Declaration | Meaning | Verified by |
| --- | --- | --- |
| `express.oauth().resource("https://bbs.example.com/api")` | OAuth resource indicator or token audience. It says which API the token was minted for. | OAuth token verifier and route auth-requirement checker. |
| `.resource(express.resource("board")...)` | Application domain object touched by this request. It might be a board, post, project, or tenant. | Application resource resolver and authorizer. |

Do not merge these fields in `RoutePlan`. The OAuth resource is part of the
credential requirement. Application resources are inputs to domain policy.

### 4.4 Scopes and actions are not aliases

The example uses `bbs.post.create` as both a scope and action because that
vocabulary is convenient. The enforcement layers remain independent:

~~~javascript
.scopes("bbs.write")       // coarse authorization-server grant
.allow("bbs.post.create")  // precise application policy action
~~~

The implementation must not automatically turn OAuth scopes into
`programauth.GrantSet`, and it must not infer scopes from `.allow(...)`.
Explicit duplication is preferable to an invisible security mapping.

### 4.5 Audit is mandatory on OAuth routes

Current planned routes permit an empty audit event. The new OAuth route type
should fail route registration unless `.audit("event.name")` is present. The
reason is operational: an externally issued credential crosses a trust
boundary and must leave a stable, redacted decision record.

The minimum OAuth route contract is therefore:

~~~text
auth(oauth with issuer + resource + scopes)
  + allow(non-empty application action)
  + audit(non-empty event name)
  + handle(function)
~~~

An application-domain `.resource(...)` declaration is optional when an action
does not operate on a particular object. The OAuth resource indicator is never
optional.

## 5. Proposed TypeScript declarations

The generated declarations currently call every auth value `UserAuthSpec`.
That name becomes inaccurate once an OAuth credential requirement exists.
Introduce a common `AuthSpec` only where the route's `.auth(...)` method needs
the union. Keep the concrete builders distinct.

~~~typescript
export function user(): UserAuthBuilder;
export function agent(): UserAuthBuilder;
export function sessionUser(): UserAuthBuilder;
export function anyOf(...specs: UserAuthSpec[]): UserAuthBuilder;

export function oauth(): OAuthAuthBuilder;

export interface OAuthAuthBuilder {
  issuer(value: string): OAuthAuthBuilder;
  resource(value: string): OAuthAuthBuilder;
  scopes(...values: string[]): OAuthAuthSpec;
}

export type OAuthAuthSpec = OAuthAuthBuilder;
export type AuthSpec = UserAuthSpec | OAuthAuthSpec;

export interface RouteNeedsSecurity {
  name(name: string): RouteNeedsSecurity;
  public(): RouteNeedsHandler;
  auth(spec: AuthSpec): RouteNeedsPolicy;
}
~~~

`anyOf` deliberately continues to accept only `UserAuthSpec`. The runtime must
enforce the same restriction because JavaScript callers are not constrained
by TypeScript.

The first version should accept scope strings as separate arguments. Do not
also add array overloads, `.audience(...)` aliases, object-form builders, or
legacy adapters. A smaller surface produces better error messages and tests.

Update `AuthInfo` with a typed optional OAuth view:

~~~typescript
export interface OAuthAuthInfo {
  issuer: string;
  subject: string;
  clientId: string;
  resources: string[];
  scopes: string[];
  expiresAt: string;
  tokenType: "Bearer";
}

export interface AuthInfo {
  method: "none" | "session" | "apiToken" | "accessToken" | string;
  principalKind?: "user" | "agent" | "service" | string;
  principalId?: string;
  credentialId?: string;
  credentialHint?: string;
  scopes: string[];
  oauth?: OAuthAuthInfo;
}
~~~

The raw token is absent. A hash of the raw token is also absent. The
introspection client secret is absent.

## 6. Proposed Go route-plan model

### 6.1 Add a typed OAuth requirement

Do not add issuer, resource, and scope fields directly to `SecuritySpec`.
`SecuritySpec` can contain alternative `AuthRequirement` values, and the
OAuth fields belong to one requirement.

~~~go
type OAuthRequirement struct {
    Issuer   string
    Resource string
    Scopes   []string
}

type AuthRequirement struct {
    Method        AuthMethod
    PrincipalKind PrincipalKind
    OAuth         *OAuthRequirement
}
~~~

The builder should compile `express.oauth()` approximately as:

~~~go
SecuritySpec{
    Mode:     SecurityModeUser,
    Required: true,
    AuthRequirements: []AuthRequirement{{
        Method: AuthMethodAccessToken,
        OAuth: &OAuthRequirement{
            Issuer:   "https://id.example.com",
            Resource: "https://bbs.example.com/api",
            Scopes:   []string{"bbs.post.create"},
        },
    }},
}
~~~

`SecurityModeUser` is an old name, but changing it is not necessary to deliver
this feature. Treat it as the existing authenticated-mode branch. A future
cleanup may rename it in a deliberate breaking change.

Do not set `PrincipalKindAgent` in this requirement. A tiny-idp device token
represents the approving human subject and the OAuth client that requested
access. The process is an agent in product language, but the authenticated
principal is still the user. `express.agent()` remains the declaration for a
durable application-owned agent principal.

### 6.2 Add typed verified OAuth context to AuthResult

The enforcer needs verified values to compare against the route requirement
and expose safely to audit and handlers.

~~~go
type OAuthAuthContext struct {
    Issuer    string
    Subject   string
    ClientID  string
    Resources []string
    Scopes    []string
    ExpiresAt time.Time
    TokenType string
}

type AuthResult struct {
    Actor          *Actor
    Method         AuthMethod
    PrincipalKind  PrincipalKind
    PrincipalID    string
    CredentialID   string
    CredentialHint string
    Grants         GrantSet
    Scopes         []string
    CSRFRequired   bool
    OAuth          *OAuthAuthContext
}
~~~

`OAuthAuthContext` means the host verifier has checked these assertions. It is
not a general claims bag. Keep the fields typed so later code cannot mistake
an arbitrary string in `Actor.Claims` for a verified issuer or audience.

Clone every slice during normalization and secure-context projection. A route
handler must not be able to mutate verifier-owned state that is also referenced
by a cache entry or audit event.

### 6.3 Identity in Actor and AuthResult

For an issuer-issued device token, use this division:

~~~json
{
  "actor": {
    "id": "user:01J...",
    "kind": "user",
    "tenantIds": ["community"]
  },
  "auth": {
    "method": "accessToken",
    "principalKind": "user",
    "principalId": "user:01J...",
    "scopes": ["bbs.post.create"],
    "oauth": {
      "issuer": "https://id.example.com",
      "subject": "tiny-idp-stable-subject",
      "clientId": "bbs-coding-agent",
      "resources": ["https://bbs.example.com/api"],
      "scopes": ["bbs.post.create"],
      "expiresAt": "2026-07-18T22:00:00Z",
      "tokenType": "Bearer"
    }
  }
}
~~~

`actor.id` is the local application user ID used by membership and resource
authorization. `auth.oauth.subject` is the stable issuer subject. They may be
different strings and serve different purposes.

## 7. Route-plan validation

Validation happens before a route becomes reachable. A malformed OAuth policy
is a startup error, not a request-time 500.

### 7.1 Syntactic validation

Extend `ValidateRoutePlan` and `normalizeAuthRequirements` with these rules:

- An OAuth requirement must use `AuthMethodAccessToken`.
- It must not use `PrincipalKindAgent` as an implicit alias.
- Issuer must be an absolute URL without userinfo, query, or fragment.
- Resource must be an absolute URI without a fragment.
- Issuer and resource are compared exactly after one documented canonicalization
  pass. Do not guess that trailing-slash variants are equivalent.
- Scopes must contain at least one value.
- Each scope must be non-empty and contain no whitespace.
- Duplicate scopes are rejected or deterministically deduplicated. Prefer
  rejection in the builder and normalization in defensive Go validation.
- An OAuth route must have a non-empty `.allow(action)`.
- An OAuth route must have a non-empty `.audit(event)`.
- An OAuth requirement may not be combined with non-OAuth requirements in the
  first implementation.

Rejecting ambiguous combinations is better than inventing semantics inside
the enforcer.

### 7.2 Capability validation

Syntactic validity does not prove the host can enforce the declaration. At
route registration or immediately after JavaScript startup, validate every
OAuth requirement against host-owned verifier profiles.

~~~go
type AuthRequirementValidator interface {
    ValidateAuthRequirement(ctx context.Context, req AuthRequirement) error
}
~~~

The concrete host validator should answer these questions without making a
request:

- Is the exact issuer configured?
- Is the exact resource allowed for this resource-server profile?
- Is an introspection client ID configured?
- Is its secret available through a Go-owned secret source?
- Was discovery completed and was a same-origin introspection endpoint found?
- Does the provider advertise the required introspection authentication
  method?

A generated host containing an OAuth route and no matching profile must not
start listening.

## 8. Authentication and enforcement pipeline

### 8.1 Route-directed credential selection

The current `CompositeAuthenticator` sees a Bearer header and tries
application-owned API/access-token services, using token prefixes to choose an
order. That behavior is appropriate for PR 95 credentials. It is insufficient
for an arbitrary tiny-idp opaque token.

Use the route requirement to select the verifier:

~~~text
if route has OAuthRequirement:
    require exactly one Bearer header
    call configured external OAuth verifier for requirement.Issuer
    never fall back to app-owned tokens or a session
else:
    preserve current app-owned bearer/session selection
~~~

This prevents a token rejected by one trust domain from being tried in another
as a fallback. It also prevents a valid browser cookie from rescuing an invalid
Bearer credential on an OAuth route.

A focused interface is:

~~~go
type OAuthBearerAuthenticator interface {
    AuthenticateOAuthBearer(
        ctx context.Context,
        raw string,
        requirement OAuthRequirement,
    ) (AuthResult, error)
}
~~~

The composite authenticator may receive this as a third dependency. The
generic `gojahttp` package owns only the interface and selection logic. The
networked RFC 7662 implementation belongs in an auth subpackage or optional
hostauth provider.

### 8.2 Strict Bearer parsing

RFC 6750 bearer transport is intentionally narrow for this feature:

- Read `Header.Values("Authorization")`.
- Require exactly one header value.
- Require exactly two whitespace-separated fields.
- Compare the scheme to `Bearer` case-insensitively.
- Reject an empty credential or control characters.
- Reject tokens in query parameters, form bodies, and cookies.
- Never log the parsed value.

Duplicate Authorization headers are not a list of alternatives. Return 401.

### 8.3 Authenticated introspection

The reference verifier should follow the already implemented tiny-idp xapp
resource authenticator:

~~~text
GET {issuer}/.well-known/openid-configuration
  verify discovery.issuer == configured issuer
  require introspection_endpoint_auth_methods_supported contains
          client_secret_basic
  require introspection endpoint origin == issuer origin

POST introspection_endpoint
  Authorization: Basic base64(resource-client-id:secret)
  Content-Type: application/x-www-form-urlencoded
  body: token=<opaque access token>
~~~

Bound request time, response size, and JSON decoding. Discovery documents are
extensible, so ignore unknown discovery fields. Introspection responses should
be decoded into a bounded typed struct containing only fields used by policy.

An active response must still pass all local checks:

~~~go
response.Active == true
response.Issuer == requirement.Issuer
strings.EqualFold(response.TokenType, "Bearer")
response.Subject != ""
response.ExpiresAt.After(now)
contains(response.Audience, requirement.Resource)
containsAll(response.Scopes, requirement.Scopes)
~~~

The confidential introspection client must itself be allowed by tiny-idp to
introspect the declared resource. That server-side restriction is defense in
depth; the application still verifies the returned `aud` value.

### 8.4 Enforcer-side requirement check

After authentication, extend `checkAuthRequirements` so the generic enforcer
independently matches the returned result:

~~~text
method matches
AND principal kind matches when specified
AND OAuth context exists
AND OAuth issuer equals requirement issuer
AND OAuth resources contain requirement resource
AND OAuth scopes contain every required scope
AND OAuth expiry is still in the future
AND token type is Bearer
~~~

This is not redundant. The verifier translates an external protocol into an
`AuthResult`; the enforcer owns the route contract. Testing both sides prevents
a future verifier implementation from accidentally treating route fields as
advisory.

### 8.5 Application authorization remains last

After the OAuth requirement passes, the existing resource resolver and
authorizer run normally:

~~~text
OAuth verifier
  -> map issuer subject to local actor
  -> check OAuth requirement
  -> resolve board/project/post resources
  -> check current app membership and action
  -> post-auth rate limit
  -> record allowed audit
  -> invoke JavaScript
~~~

An OAuth scope never bypasses a disabled application user, revoked membership,
tenant boundary, or resource ownership rule.

## 9. Mapping an issuer subject to an application user

### 9.1 Use `(issuer, subject)`, not subject alone

OpenID Connect defines `sub` within an issuer. Two issuers may legally use the
same text for different people. A stable external identity key is therefore:

~~~text
(canonical issuer, exact subject)
~~~

The current hostauth `appauth.User` model uses a field named `KeycloakSub` and
indexes only that string. The OAuth work exposes why that model is too narrow.
Do not add a compatibility adapter that keeps pretending every issuer is
Keycloak or that `sub` is globally unique.

The reusable interface should become issuer-aware:

~~~go
type ExternalIdentityResolver interface {
    ByExternalIdentity(
        ctx context.Context,
        issuer string,
        subject string,
    ) (*Actor, error)
}
~~~

For the default appauth store, use either an external-identity table or user
columns constrained by a unique `(issuer, subject)` index. Browser OIDC login
and bearer-token authentication must call the same mapping.

### 9.2 Do not create membership during bearer authentication

If the external identity has never entered the application, product policy
must decide whether bearer authentication may create a local user. The safe
first rule is:

~~~text
browser OIDC login may upsert the external identity and run onboarding
bearer authentication may resolve an existing enabled identity only
bearer authentication never grants tenant membership or roles
~~~

This keeps signup and onboarding visible to the application. If later product
requirements permit just-in-time bearer onboarding, implement it as an
explicit policy with tests and audit rather than an incidental lookup side
effect.

### 9.3 Preserve client identity separately

The introspection `client_id` identifies the OAuth client that obtained the
token. It is not the user and not the local Actor ID. Preserve it in
`AuthResult.OAuth.ClientID` so an application can audit or optionally restrict
known coding-agent clients.

Do not make `client_id` an application role. If route-level client allowlists
become necessary, add a separately reviewed builder method such as
`.clients(...)`; do not overload `.scopes(...)` or `.allow(...)`.

## 10. Host configuration and secrets

### 10.1 Resource-server profiles

The host needs a finite set of configured issuer/resource profiles. JavaScript
must not cause arbitrary discovery or choose an introspection secret.

A conceptual configuration is:

~~~yaml
auth:
  oauthResources:
    - issuer: https://id.example.com
      resources:
        - https://bbs.example.com/api
      introspection:
        clientId: bbs-resource-server
        clientSecretFile: /run/secrets/bbs-introspection-client
      cache:
        positiveTtl: 30s
        negativeTtl: 3s
      requestTimeout: 10s
~~~

Use the repository's Glazed configuration sections and values rather than
direct `os.Getenv` calls. The secret should enter through a mounted file or an
equivalent Go-owned secret provider. Do not place it in xgoja YAML committed to
source, generated JavaScript, command output, or error strings.

### 10.2 Startup behavior

Build and validate profiles before the listener becomes ready:

1. Parse and canonicalize the issuer and allowed resources.
2. Read the client secret into Go-owned memory.
3. Discover the issuer with a bounded HTTP client.
4. Validate the introspection endpoint and supported auth method.
5. Register the verifier profile by exact issuer.
6. Load JavaScript and validate every planned OAuth route against profiles.
7. Report ready only after the database and required verifier configuration
   are usable.

If discovery is temporarily unavailable at process startup, fail startup or
remain unready according to an explicit operator policy. Do not start a public
listener and downgrade OAuth routes.

### 10.3 Caching and revocation

Introspection on every request gives immediate provider state but adds latency
and availability coupling. A small in-memory cache is reasonable for the
single-replica target when its revocation bound is explicit.

Use an HMAC of the token as the cache key with a process-random secret. Never
use the raw token. Bound a positive entry by all of:

~~~text
configured positive TTL
token expiration
operator's documented maximum revocation delay
~~~

Use a much shorter negative TTL. Do not turn provider-unavailable results into
negative inactive-token cache entries; availability failure and invalid
credential are different states.

## 11. Errors, status codes, and response headers

The current error set needs a typed unavailable outcome so an introspection
outage does not become an unexplained 500 or a 401 token oracle.

~~~go
var ErrAuthUnavailable = errors.New("authentication service unavailable")
~~~

Map outcomes as follows:

| Failure | Status | Handler invoked? | Notes |
| --- | ---: | --- | --- |
| Missing, duplicate, or malformed Authorization header | 401 | No | Add a bounded `WWW-Authenticate: Bearer` challenge. |
| Inactive, expired, wrong issuer, wrong resource, or wrong token type | 401 | No | Do not reveal which token assertion failed. |
| Valid token missing route scope | 403 | No | Credential is valid but lacks required grant ceiling. |
| Valid token maps to no enabled local user | 403 | No | Avoid account-existence detail in the response. |
| Application authorizer denies action/resource | 403 or existing not-found concealment | No | Preserve current application policy. |
| Introspection discovery/network/5xx/invalid response | 503 | No | Never downgrade to anonymous or session. |
| Host has no matching verifier profile | Startup failure | No listener | This is configuration, not a request error. |

Production bodies should use generic codes. Detailed reasons belong in
redacted server audit and metrics.

## 12. Audit and observability

An OAuth route audit record should contain enough data to reconstruct the
decision without containing credentials.

Safe attributes include:

- route name, method, pattern, and declared action;
- allow/deny/unavailable outcome and coarse reason;
- local actor ID after successful identity mapping;
- issuer, OAuth subject, and client ID;
- required scopes and verified scope names;
- required resource indicator;
- token expiry rounded or formatted as a timestamp;
- request correlation ID and canonical client address from trusted proxy
  resolution;
- resolved application resource IDs when the existing audit policy permits.

Never record:

- the access token;
- a refresh token or device code;
- the Authorization header;
- the introspection client secret;
- a reversible token cache key;
- full introspection JSON;
- raw network error bodies from the provider.

Metrics should distinguish invalid credentials, insufficient scope,
application denial, and provider unavailability without placing issuer
subjects, client IDs, tokens, or route parameters in high-cardinality labels.

## 13. The builder implementation

### 13.1 Extend builderStore with a distinct OAuth builder

`modules/express/auth_builders.go` currently stores auth specs by Goja object.
Add `newOAuthBuilder` that owns a typed builder state and ultimately stores a
`SecuritySpec` containing one OAuth requirement.

Pseudocode:

~~~go
type oauthBuilderState struct {
    spec        *SecuritySpec
    issuerSet   bool
    resourceSet bool
    scopesSet   bool
}

func (s *builderStore) newOAuthBuilder(vm *goja.Runtime) goja.Value {
    state := &oauthBuilderState{
        spec: &SecuritySpec{
            Mode: SecurityModeUser,
            Required: true,
            AuthRequirements: []AuthRequirement{{
                Method: AuthMethodAccessToken,
                OAuth: &OAuthRequirement{},
            }},
        },
    }
    obj := vm.NewObject()
    s.authSpecs.Store(obj, state.spec)

    obj.Set("issuer", func(raw string) (goja.Value, error) {
        if state.issuerSet { return nil, error("issuer called twice") }
        state.issuerSet = true
        state.spec.AuthRequirements[0].OAuth.Issuer = raw
        return obj, nil
    })

    obj.Set("resource", func(raw string) (goja.Value, error) {
        if state.resourceSet { return nil, error("resource called twice") }
        state.resourceSet = true
        state.spec.AuthRequirements[0].OAuth.Resource = raw
        return obj, nil
    })

    obj.Set("scopes", func(call goja.FunctionCall) (goja.Value, error) {
        if state.scopesSet { return nil, error("scopes called twice") }
        state.scopesSet = true
        state.spec.AuthRequirements[0].OAuth.Scopes = parseStrings(call.Arguments)
        return obj, nil
    })
    return obj
}
~~~

The real code must wrap `obj.Set` errors consistently and copy slices before a
plan is stored. `ValidateRoutePlan` remains the final defensive check for
missing methods and malformed values.

### 13.2 Register the export

In `modules/express/express.go`:

~~~go
_ = exports.Set("oauth", func() goja.Value {
    return builders.newOAuthBuilder(vm)
})
~~~

Keep the Express registrar provider-neutral. The export constructs data; it
does not perform discovery or read configuration.

### 13.3 Improve authSpec errors

The current error names the accepted builders explicitly. Add
`express.oauth()` to both nil and wrong-object errors. Verify that a plain
object imitating the fields is rejected:

~~~javascript
.auth({ issuer: "...", resource: "...", scopes: ["..."] })
~~~

Only objects created by the registered builder store may become security
specifications.

## 14. File-by-file implementation map

### 14.1 Core route API

| File | Required change |
| --- | --- |
| `modules/express/auth_builders.go` | Add the OAuth builder, builder-state validation, defensive copies, and runtime rejection from `anyOf`. |
| `modules/express/express.go` | Export `oauth()`. |
| `modules/express/typescript.go` | Add `OAuthAuthBuilder`, `OAuthAuthSpec`, `AuthSpec`, and typed `AuthInfo.oauth`. Keep `anyOf` restricted. |
| `modules/express/auth_builders_integration_test.go` | Prove compilation, missing fields, duplicates, plain-object rejection, `anyOf` rejection, and handler non-invocation. |

### 14.2 Generic Go enforcement

| File | Required change |
| --- | --- |
| `pkg/gojahttp/auth_plan.go` | Add `OAuthRequirement`, `OAuthAuthContext`, `AuthResult.OAuth`, normalization, validation, cloning, and unavailable error semantics. |
| `pkg/gojahttp/auth_plan_test.go` | Test canonical valid plans and every malformed OAuth requirement. |
| `pkg/gojahttp/enforcer.go` | Match verified OAuth results against route requirements before resources and authorization; map unavailable to 503. |
| `pkg/gojahttp/enforcer_test.go` | Test issuer, resource, scope, expiry, token type, actor mapping, CSRF separation, and zero handler calls on failure. |
| `pkg/gojahttp/planned_dispatch.go` | Project only the new redacted OAuth context into JavaScript. Do not expose request credentials. |
| `pkg/gojahttp/planned_dispatch_test.go` | Assert the exact JavaScript context shape and absence of secrets. |

### 14.3 Credential composition

| File or package | Required change |
| --- | --- |
| `pkg/gojahttp/auth/programauth/composite.go` or a generalized replacement | Select external OAuth verification only when the route has an OAuth requirement. Preserve existing PR 95 agent-token behavior otherwise. |
| New `pkg/gojahttp/auth/oauthresource` package | Implement provider-neutral discovery, RFC 7662 introspection, strict validation, bounded caching, and identity mapping interfaces. |
| Focused unit tests beside the new package | Cover transport, discovery pinning, Basic auth, response limits, cache bounds, status classification, and redaction. |

Do not put tiny-idp imports in this package. Its protocol contract is standard
OAuth metadata plus RFC 7662 fields.

### 14.4 Optional generated-host wiring

| File or package | Required change |
| --- | --- |
| `pkg/xgoja/hostauth/config.go` | Add Glazed-backed resource-server profiles and secret-file references. |
| `pkg/xgoja/hostauth/preflight.go` | Fail on incomplete, duplicate, unsafe, or unreferenced profiles according to the final config policy. |
| `pkg/xgoja/hostauth/builder.go` | Build verifiers, identity resolvers, and the route requirement validator; inject them into `AuthOptions`. |
| `pkg/xgoja/hostauth/readiness.go` | Include live dependency checks for required verifier configuration and the application identity store. |
| `pkg/xgoja/hostauth/services.go` | Expose typed verifier services to generated/custom hosts without exposing secrets to JavaScript. |

### 14.5 Application identity storage

The current `appauth` `KeycloakSub` field and lookup are single-issuer. If the
default hostauth implementation will map external OAuth subjects, update the
store contract and schema to use `(issuer, subject)`. This is a deliberate data
model change. Write a migration; do not retain a second hidden lookup path that
sometimes ignores issuer.

## 15. Implementation phases

### Phase A: Pure data model and validation

Implement `OAuthRequirement`, `OAuthAuthContext`, cloning, normalization, and
route validation. Add Go-native constructors if the package maintains parity
with JavaScript builders:

~~~go
func OAuth(issuer, resource string, scopes ...string) SecuritySpec
~~~

Exit criterion: malformed OAuth plans cannot pass `ValidateRoutePlan`, and no
network or Goja runtime is required to test this phase.

### Phase B: JavaScript and TypeScript API

Add `express.oauth()` and generated declarations. Compile scripts into route
plans and inspect the plans in integration tests.

Exit criterion: the canonical JavaScript produces the exact normalized Go
structure; missing or repeated fields and mixed `anyOf` fail registration.

### Phase C: Enforcer contract

Teach the enforcer to compare an already verified `AuthResult.OAuth` with the
route requirement. Use test authenticators; do not begin with live HTTP.

Exit criterion: every mismatch prevents resource resolution, authorization,
and handler execution. A matching result follows the existing pipeline.

### Phase D: RFC 7662 verifier

Build the provider-neutral resource verifier with a fake TLS server. Add
strict discovery, Basic-authenticated introspection, bounded decoding,
classification, and HMAC-keyed caching.

Exit criterion: the verifier accepts one valid fixture and rejects or marks
unavailable every negative fixture with the correct typed outcome.

### Phase E: Identity mapping and hostauth composition

Wire configured profiles and issuer-aware app identity lookup into generated
hostauth. Add startup capability validation.

Exit criterion: a generated host with an OAuth route fails startup without a
matching profile and runs with no secret visible to JavaScript.

### Phase F: tiny-idp end-to-end example

Extend the Personal Knowledge Inbox or create a focused example that uses
tiny-idp's native device flow, not the PR 95 app-owned device endpoints. Keep
the existing example path so both architectures can be compared.

Exit criterion: a CLI obtains a tiny-idp token, calls an `express.oauth()`
route, reaches the same domain service as a browser user, and fails on wrong
issuer, resource, scope, user, or client configuration.

### Phase G: documentation and generated checks

Update help pages, TypeScript snapshots, examples, diagrams, and generated
artifacts. Run repository generation and the complete test suite.

Exit criterion: `go generate ./...`, `go test ./...`, `go build ./...`, and the
focused example smoke all pass from the top-level module.

## 16. Test matrix

### 16.1 Builder and plan tests

| Scenario | Expected result |
| --- | --- |
| Canonical issuer/resource/one scope | Route registers with one typed OAuth requirement. |
| Multiple scopes | All are preserved in deterministic order. |
| Missing issuer | Registration fails before listener readiness. |
| Missing resource | Registration fails. |
| Empty scopes | Registration fails. |
| Scope containing whitespace | Registration fails with field-specific error. |
| Duplicate issuer/resource/scopes method call | Builder fails rather than replacing policy. |
| Plain JavaScript object passed to `.auth` | Rejected as not builder-owned. |
| OAuth builder passed to `express.anyOf` | Rejected in v1. |
| OAuth route missing `.allow` | Registration fails. |
| OAuth route missing `.audit` | Registration fails. |
| HTTP issuer in production profile | Preflight fails. |
| Route issuer/resource absent from host profiles | Startup fails. |

### 16.2 Request enforcement tests

For every denial test, count calls to the resolver, authorizer, and handler.
The handler count must remain zero. Earlier stages should also remain zero when
the failure precedes them.

| Request or introspection result | Status | Resolver | JS handler |
| --- | ---: | ---: | ---: |
| No Authorization header | 401 | 0 | 0 |
| Duplicate Authorization headers | 401 | 0 | 0 |
| Basic instead of Bearer | 401 | 0 | 0 |
| Query token only | 401 | 0 | 0 |
| Inactive token | 401 | 0 | 0 |
| Wrong issuer | 401 | 0 | 0 |
| Wrong resource/audience | 401 | 0 | 0 |
| Expired token | 401 | 0 | 0 |
| Wrong token type | 401 | 0 | 0 |
| Missing required scope | 403 | 0 | 0 |
| Provider timeout or 5xx | 503 | 0 | 0 |
| Unknown or disabled local user | 403 | 0 | 0 |
| Valid OAuth token, app policy denies | 403 | 1 when needed | 0 |
| Valid token and app policy allows | 2xx | expected | 1 |

### 16.3 Credential separation tests

These tests guard the distinction introduced in Section 2:

- A PR 95 `ggat_` application token cannot enter an `express.oauth()` route.
- A tiny-idp token cannot enter an `express.agent()` route merely because a
  coding-agent process presented it.
- A browser cookie cannot enter an OAuth route.
- A tiny-idp bearer token cannot enter a `sessionUser()` route.
- An invalid Bearer header plus a valid browser cookie does not fall back to
  the cookie.
- A valid tiny-idp token for application A cannot enter application B because
  its resource indicator differs.
- The same tiny-idp subject maps to the same local user for browser login and
  device-token API access.
- Equal textual subjects from different issuers map to distinct identities.

### 16.4 Secret and redaction tests

Use sentinel strings for the access token and introspection secret. Search
response bodies, audit records, structured logs, metrics snapshots, errors,
`ctx.auth`, and `ctx.actor`. Neither sentinel may appear.

The introspection HTTP fixture should assert that the secret appears only in
the request's Basic authentication and that the raw access token appears only
in the introspection form body.

### 16.5 Cache tests

- A positive cache hit skips a second introspection request.
- A positive entry expires no later than token `exp`.
- A negative inactive result uses the short negative TTL.
- Provider-unavailable is not cached as inactive.
- Cache keys do not contain the raw token.
- Two process instances do not claim shared cache behavior.
- Revocation takes effect within the documented positive TTL.

## 17. A concrete end-to-end trace

Assume the route requires issuer `https://id.example.com`, resource
`https://bbs.example.com/api`, scope `bbs.post.create`, and action
`bbs.post.create`.

~~~text
1. POST /api/agent/bbs/posts arrives with one Bearer header.
2. Route registry selects the planned route.
3. Pre-auth IP/route rate limits run.
4. Composite authenticator sees an OAuthRequirement and selects only the
   configured external OAuth verifier.
5. Verifier introspects the opaque token using the BBS resource secret.
6. tiny-idp reports active token, exact issuer, subject Alice, coding-agent
   client, BBS audience, required scope, Bearer type, and future expiry.
7. Verifier resolves (issuer, Alice-subject) to local app user user:alice.
8. Verifier returns AuthResult with Actor user:alice and typed OAuth context.
9. Enforcer independently checks the result against the route requirement.
10. CSRF is skipped because this credential is not ambient cookie auth.
11. Board resource "main" is resolved.
12. Application authorizer checks user:alice may create posts on board main.
13. Post-auth actor/resource rate limits run.
14. Allowed audit event is recorded without credential material.
15. JavaScript handler receives ctx.actor, ctx.auth.oauth, ctx.resource,
    ctx.action, params, and body.
16. Handler invokes the domain service and writes the response.
17. Completion audit records the final status.
~~~

If Step 6 lacks the scope, Steps 7 through 17 do not run. If Step 12 denies
membership, Step 15 does not run. This ordering is a testable contract.

## 18. Common implementation mistakes

### Treating `express.oauth()` as `express.agent()`

The process is a coding agent, but tiny-idp authenticated a user subject and
client grant. Principal kind and credential kind are different dimensions.

### Reusing the current app-owned access-token verifier

PR 95 access tokens are records in the application's token store. Tiny-idp
tokens are external opaque values verified through authenticated
introspection. Sharing the name `accessToken` does not make their trust roots
identical.

### Letting JavaScript call introspection

That exposes the resource-server secret and raw token to the dynamic runtime,
duplicates enforcement, and makes handler invocation part of authentication.
Keep introspection in Go.

### Looking up users by `sub` alone

Subjects are issuer-scoped. Use `(issuer, subject)` even if the first
deployment configures only one issuer.

### Translating scopes into application roles

Scopes are a coarse client grant. Membership, ownership, and current app state
still decide the action. Preserve both checks.

### Falling back on provider failure

An introspection timeout is 503, not anonymous access, session fallback, or an
inactive-token cache entry.

### Supporting every syntax variant at once

Do not add object forms, aliases, arrays, mixed `anyOf`, JWT validation, and
multiple client policies in the first patch. Each extra surface multiplies
security cases.

### Auditing full introspection responses

Responses can contain sensitive identifiers and future provider extensions.
Audit only the explicitly selected redacted fields.

## 19. Alternatives considered

### Keep the current Go-owned device API outside Express

The tiny-idp xapp already demonstrates a safe Go-owned bearer handler. Keeping
that boundary is viable and simpler. It does not satisfy the product goal that
application authors declare browser and coding-agent API policy through the
same planned route system. The proposed API generalizes that safe Go boundary
without moving token handling into JavaScript.

### Use `express.agent()` for tiny-idp tokens

Rejected because it loses issuer, resource, scope, client, and expiry
requirements and conflates an app-owned agent record with an OAuth client
acting for a user.

### Accept both sessions and OAuth on one route

Rejected for v1 because credential precedence, CSRF, error challenges, and
audit semantics become less obvious. Separate routes may share the domain
handler.

### Use self-contained JWT access tokens

JWT validation would require issuer key discovery, algorithm policy, audience
rules, clock handling, and a revocation contract. tiny-idp already issues
opaque tokens and supports authenticated introspection. Use the existing
central status model first.

### Put only scope strings on `.allow(...)`

Rejected because it merges authorization-server grant vocabulary with
application policy. Keep `.scopes(...)` and `.allow(...)` explicit.

### Add an OAuth-specific SecurityMode

This can work, but the current model already represents credential and
principal restrictions as `AuthRequirement`. A typed nested OAuth requirement
fits that model and leaves room for future explicit alternatives. Adding a
third mode would duplicate much of the authenticated enforcer branch.

## 20. Code review order

Review the implementation in dependency order rather than starting with the
JavaScript example:

1. `pkg/gojahttp/auth_plan.go`: verify the data model can express the contract
   without raw credentials or ambiguous fields.
2. Route-plan tests: verify invalid combinations fail before runtime.
3. `modules/express/auth_builders.go` and TypeScript declarations: verify the
   public API compiles exactly into that model.
4. Enforcer tests and `checkAuthRequirements`: verify mismatches stop before
   resource resolution and JavaScript.
5. OAuth resource verifier: verify strict transport, discovery pinning,
   introspection validation, error classification, and caching.
6. Identity resolver and appauth migration: verify issuer-qualified identity
   and disabled-user behavior.
7. hostauth builder/preflight/readiness: verify optional composition and
   startup failure for unsupported route requirements.
8. JavaScript context projection: verify redaction and defensive copies.
9. End-to-end tiny-idp smoke: verify real discovery, introspection, subject
   mapping, app authorization, and cross-resource denial.
10. Documentation and generated artifacts: verify the implementation and help
    pages describe the same API.

For every layer, ask:

- What exact input is trusted here?
- Which previous layer verified it?
- What happens if this dependency is unavailable?
- Can JavaScript run before this check?
- Can a raw credential enter a response, log, metric, audit record, or cache
  key?
- Does a test prove the negative path, not only the happy path?

## 21. Definition of done

The feature is complete when all of these statements are true:

- `express.oauth().issuer().resource().scopes()` is documented and represented
  by a typed Go route plan.
- Browser sessions, app-owned agents, and issuer-issued OAuth tokens remain
  distinct authentication contracts.
- OAuth routes cannot mix with `anyOf` in the first version.
- Missing issuer, resource, scopes, action, audit, or host profile prevents
  startup.
- The host selects an external verifier from the route requirement and never
  falls back to session or app-owned bearer authentication.
- tiny-idp tokens are introspected with a Go-owned confidential client over a
  bounded transport.
- Exact issuer, resource, expiry, token type, and required scopes are checked.
- `(issuer, subject)` resolves to an enabled local application user before
  application authorization.
- OAuth scopes do not replace application actions, membership, or resource
  policy.
- `ctx.auth.oauth` contains only typed, verified, non-secret fields.
- Invalid, forbidden, and unavailable outcomes map consistently to 401, 403,
  and 503.
- Every denial occurs before JavaScript invocation.
- Audit and metrics contain no token or introspection secret.
- Cache entries are HMAC-keyed and bounded by expiry and the revocation SLO.
- Go API, JavaScript API, TypeScript declarations, help pages, and examples
  agree.
- `go generate ./...`, `go test ./...`, `go build ./...`, focused race tests,
  and the tiny-idp device-to-Express smoke pass.

## 22. Source map and protocol references

### Existing go-go-goja code

- `modules/express/auth_builders.go` compiles fluent JavaScript declarations
  into `gojahttp.RoutePlan` security, resource, rate-limit, action, and audit
  fields.
- `modules/express/express.go` exports the module builders.
- `modules/express/typescript.go` defines the generated JavaScript API types.
- `pkg/gojahttp/auth_plan.go` owns `SecuritySpec`, `AuthRequirement`,
  `AuthResult`, route validation, and host interfaces.
- `pkg/gojahttp/enforcer.go` runs authentication, CSRF, resource resolution,
  grants, application authorization, rate limits, and audit before handler
  dispatch.
- `pkg/gojahttp/auth/programauth/composite.go` currently selects app-owned
  bearer or browser-session authentication.
- `pkg/gojahttp/auth/programauth/oauth_token.go` validates PR 95 application-
  owned access tokens. It is not a tiny-idp introspection adapter.
- `pkg/xgoja/hostauth/builder.go` composes optional generated-host auth
  services.
- `pkg/gojahttp/auth/appauth` owns the default local user, membership,
  resource, and authorization model.

### Existing tiny-idp reference code

- `cmd/tinyidp-xapp/internal/resourceauth/resourceauth.go` is the strongest
  existing reference for strict header parsing, discovery, authenticated RFC
  7662 introspection, audience/scope checks, HMAC-keyed caching, and coarse
  outcomes.
- `cmd/tinyidp-xapp/device_api.go` demonstrates Go-owned bearer enforcement
  before Durable Object dispatch.
- The production architecture review, Section 8, introduces the route syntax
  and explains why `express.user()` is insufficient.
- The PR 98 hardening guide explains hostauth, the cluster, and the distinction
  between app-owned and IdP-owned device authorization.

### Standards

- RFC 6750 defines Bearer token use in HTTP.
- RFC 7662 defines OAuth token introspection.
- RFC 8707 defines OAuth resource indicators.
- RFC 8628 defines the OAuth device authorization grant used by coding agents.

The standards define protocol fields and endpoint behavior. This guide adds
the go-go-goja route-plan, host-composition, identity-mapping, and application-
authorization rules needed to use those protocols safely.
