---
Title: OAuth and OpenID Connect Protocol Security Foundations
Ticket: TINYIDP-PROD-IMPL-001
Status: active
Topics:
    - auth
    - authentication
    - identity
    - oidc
    - research
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/interaction.go
      Note: |-
        Canonical authorization request and opaque interaction binding
        Canonical request and opaque interaction
    - Path: repo://internal/fositeadapter/interaction_hardening_test.go
      Note: |-
        Fresh authentication, consent, mutation, replay, and expiry regressions
        Protocol regressions
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Authorization, token, UserInfo, session, and response integration
        Authorization token and UserInfo behavior
    - Path: repo://internal/fositeadapter/provider_test.go
      Note: Complete strict Authorization Code and PKCE flow
    - Path: repo://internal/fositeadapter/session.go
      Note: |-
        Browser-session validation and authentication time
        Session and authentication time
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc6749
    - https://www.rfc-editor.org/rfc/rfc7636
    - https://www.rfc-editor.org/rfc/rfc9700
    - https://openid.net/specs/openid-connect-core-1_0.html
Summary: Protocol-first explanation of OAuth authorization, OpenID authentication, PKCE, browser interactions, freshness, consent, tokens, and UserInfo as implemented by tiny-idp.
LastUpdated: 2026-07-10T22:00:00-04:00
WhatFor: Giving new contributors enough protocol and attacker-model knowledge to review authorization and token changes safely.
WhenToUse: Read before modifying authorization parsing, sessions, login, consent, redirects, token issuance, refresh, claims, or UserInfo.
---


# OAuth and OpenID Connect Protocol Security Foundations

## Purpose

OAuth 2.0 delegates authorization. OpenID Connect adds an authenticated identity
statement to an OAuth flow. tiny-idp implements the Authorization Code flow with
S256 PKCE, browser authentication, consent, ID tokens, access tokens, refresh
tokens, and UserInfo. Each artifact has a different audience and security
meaning. Treating them as interchangeable bearer strings is the first mistake
this chapter is designed to prevent.

The essential protocol property is binding. An issued artifact must remain
bound to the validated client, redirect URI, requested scopes, authenticated
subject, authentication time, nonce, code challenge, and server-side decision
that authorized it. Multi-request browser interactions must preserve those
bindings without allowing the browser to become their authority.

## Principals and channels

| Principal | Role | Inputs it controls |
|---|---|---|
| Resource owner | authenticates and grants/denies consent | credentials and explicit decision |
| Browser | transports front-channel requests and cookies | all form/query bytes, including attacker mutations |
| Relying party/client | requests authorization and redeems code | client ID, redirect, state, nonce, scopes, PKCE values |
| Authorization server | validates requests and issues artifacts | interaction state, sessions, codes, tokens, claims |
| Resource server/UserInfo | accepts access-token authority | bearer credential transport and token scopes |

The browser is not trusted merely because it has a CSRF cookie. CSRF protection
binds a request to a browser context. It does not prove that hidden fields are
the same fields the authorization server validated earlier.

## The successful Authorization Code + PKCE sequence

```text
Client                  Browser                 tiny-idp                 Store
  |                        |                        |                       |
  | authorization request  |                        |                       |
  +----------------------->+----------------------->| validate with Fosite  |
  |                        |                        | create interaction ---->|
  |                        |<-----------------------+ login/consent form     |
  |                        | credentials/decision   |                       |
  |                        +----------------------->| authenticate/revalidate|
  |                        |                        | consume interaction ---->|
  |                        |                        | create code+PKCE+OIDC -->|
  |                        |<-----------------------+ redirect(code,state)   |
  |<-----------------------+                        |                       |
  | token request(code, verifier)                  |                       |
  +------------------------------------------------>| validate + transaction|
  |                                                 | invalidate code ------>|
  |                                                 | create access/refresh >|
  |<------------------------------------------------+ token response         |
```

`state` is returned to the client for client-side request/callback correlation.
It is not tiny-idp's interaction handle. `nonce` binds an ID token to the
client's authentication request. PKCE binds code redemption to possession of a
high-entropy verifier whose S256 digest was in the authorization request.

## Authorization request validation

The authorization endpoint does not begin by collecting credentials. It first
asks Fosite to parse and validate the protocol request. Validation covers the
client, exact redirect URI, response type, scopes, PKCE requirements, and OIDC
parameters. Only after successful protocol validation may tiny-idp decide
whether authentication or consent is required.

The safe order is:

```text
raw request
  -> protocol validation
  -> current client/session/policy evaluation
  -> server-owned interaction creation
  -> credential or consent UI
```

Rendering credentials after an invalid request would create a phishing and
confused-deputy surface. `interaction_hardening_test.go` contains regressions
ensuring malformed `max_age` and other invalid requests do not become credential
forms.

## Canonical request and opaque interaction

An authorization interaction spans at least two browser requests. The first
request is a client-controlled OAuth message. The second is a user action. The
server must carry forward the validated protocol request while accepting only
the new user input required at the second step.

tiny-idp stores an `InteractionRecord` under a hash of a random opaque handle.
The record contains:

- a canonical copy of the validated authorization request;
- the validated client and redirect binding;
- a digest of client generation/configuration;
- required actions such as login, fresh login, and consent;
- session binding when appropriate;
- creation and expiry times;
- terminal outcome metadata.

The browser receives only the opaque interaction handle, CSRF token, and fields
needed for login or consent. It does not receive authoritative hidden copies of
`client_id`, `redirect_uri`, `scope`, `state`, `nonce`, or PKCE parameters.

On resume, `interaction.go` reconstructs the Fosite request from the stored
canonical values. The provider revalidates mutable client/user state before
issuance. This two-stage validation matters because a client or user can be
disabled while a browser form is open.

## Authentication freshness

A valid browser session proves that authentication occurred in the past. It
does not automatically satisfy a request for new authentication.

Three inputs control the relevant behavior:

- `prompt=login` requires the authorization server to prompt for
  reauthentication.
- `max_age=N` requires the elapsed time since authentication to be no greater
  than `N` seconds.
- `prompt=none` prohibits interactive UI; if login or consent is required, the
  server must return an OAuth error rather than display a form.

The temporal property is:

```text
if interaction.required_actions contains fresh_login
then an AuthenticationSatisfied event must occur after InteractionCreated
and before an approved terminal outcome and artifact issuance
```

This explains the original review defect. The GET path correctly rendered a
fresh-login form, but the browser did not carry `prompt` into the POST and an
empty login could fall back to the old session. Checking only whether a session
exists erased the required action. Persisting the action in the interaction
makes it impossible for the second request to recompute a weaker requirement.

`auth_time` is the instant associated with successful authentication. A forced
login must update it to the new authentication event; reusing the old session
would produce a semantically false freshness claim even if the ID token were
cryptographically valid.

## Password results are typed security outcomes

Authentication is not a boolean. The authenticator can return:

- success with a user and metadata;
- invalid credentials;
- temporary authentication unavailability;
- admission rejection because bounded password work is saturated;
- success accompanied by a required password-change state.

The provider must handle every outcome before creating a browser session or
issuing artifacts. The earlier code-review comment about `MustChangePassword`
identified exactly this class: successful password verification is not always
authorization to continue. Lifecycle flags are part of the authentication
result's security semantics.

## Consent

Authentication identifies the subject. Consent authorizes a client and scope
set. These transitions must not be collapsed.

When consent is required, the form displays the validated client ID and exact
requested scopes. The browser sends only an explicit approve or deny action.
Approval is bound to server-owned request data; denial consumes the interaction
with a denied terminal outcome and produces OAuth `access_denied`.

An omitted decision is not approval. Stored historical consent can allow policy
to skip a new prompt, but that is an explicit server-side policy result rather
than a missing browser field interpreted favorably.

## Codes and tokens

| Artifact | Presented by | Accepted at | Core binding |
|---|---|---|---|
| Authorization code | client | token endpoint | client, redirect, PKCE, request/session |
| ID token | client/RP | RP | issuer, audience, subject, nonce, auth time |
| Access token | client | UserInfo/resource | subject, client, scopes, expiry |
| Refresh token | client | token endpoint | token family, client, subject, scopes |

An authorization code is a one-time capability. The token transaction must
invalidate it exactly once while creating the resulting tokens atomically. If
the code were consumed but token creation rolled back incompletely, a transient
storage failure would permanently deny the legitimate client. If a token row
committed while code invalidation failed, replay could create multiple grants.

Refresh rotation replaces one active refresh capability with another. Reuse of
an inactive old token is evidence that a token may have been copied, so Fosite
revokes the family. This creates an important availability consequence: two
legitimate concurrent refresh requests can produce one successful response and
then family revocation when the loser is recognized as reuse. Clients should
serialize refresh operations.

## UserInfo and bearer transport

UserInfo accepts an access token and returns claims permitted by its scopes.
Bearer transport must be unambiguous. tiny-idp accepts the Authorization header
and rejects query or form bearer tokens, including mixed transports. Duplicate
Authorization headers are rejected rather than resolved by precedence.

The endpoint distinguishes:

- missing or invalid credentials, which map to `invalid_token` semantics;
- multiple or forbidden transport methods, which map to `invalid_request`;
- unsupported HTTP methods;
- successful GET or POST with no-store cache controls.

This is both protocol and web security. Query tokens leak through histories,
logs, referrers, and intermediaries more easily than Authorization headers.

## Redirect and error discipline

An authorization server may redirect an OAuth error only when it has validated
the client and redirect URI sufficiently to trust the destination. Raw or
unverified request-object claims must never supply an error redirect target.
Otherwise the authorization endpoint becomes an open redirect or sends
security-sensitive errors to an attacker-controlled location.

The review test for unsupported request objects preserves `state` only after the
ordinary request has established a valid redirect. This is a useful code-review
question for every new early-return path: does it write an HTTP error locally,
or is it safe to redirect?

## Security properties to preserve

- Invalid authorization requests never collect credentials.
- Browser-controlled fields never replace the stored canonical request.
- Required authentication and consent actions cannot disappear between requests.
- `prompt=none` never produces interactive UI.
- Fresh authentication occurs after the interaction that required it.
- One interaction has at most one terminal outcome.
- Approval precedes authorization artifact creation.
- Code redemption and refresh rotation are atomic.
- Access and refresh credentials are never written to audit or security traces.
- UserInfo accepts one explicit bearer transport.
- Error redirects use only validated redirect destinations.

## Guided code trace

Read and annotate this path:

```text
Provider.beginAuthorize
  Fosite.NewAuthorizeRequest
  parseMaxAge
  readBrowserSession
  determineRequiredActions
  createAuthorizationInteraction
  renderInteraction

Provider.resumeAuthorize
  parse form and CSRF
  loadAuthorizationInteraction
  reconstructAuthorizeRequest
  revalidate client/key/session/user
  AuthenticatePassword when required
  evaluate explicit consent
  consume interaction
  finishAuthorize
```

For every call, write down whether it reads raw request data, validated protocol
data, server-owned state, or current mutable state. If one value changes category
without an explicit validation step, investigate it.

## Exercises

1. Explain why signing the old hidden continuation would detect mutation but
   would not by itself provide expiry, one-time consumption, mutable-state
   revalidation, or operator inspection.
2. Construct a sequence in which a session is valid but `max_age=0` requires new
   authentication.
3. Explain why `prompt=none` plus required consent returns an error rather than a
   consent form.
4. Locate the exact point at which an authorization code becomes externally
   visible and compare it with the SQL commit point.
5. Explain why two concurrent refresh requests can revoke the winning response's
   family without violating the one-time rotation model.
6. Review one error path and prove whether redirecting is safe.

## Research provenance: from standards to local properties

RFC 6749 defines the authorization framework, roles, endpoints, grants, codes,
tokens, redirects, and error responses. RFC 6750 defines bearer-token transport
and its disclosure risks. RFC 7636 adds proof key for code exchange. RFC 9700
updates the security baseline and emphasizes exact redirect handling, PKCE,
browser and proxy threats, and deprecated flow choices. OpenID Connect Core adds
identity claims, ID tokens, nonce, `prompt`, `max_age`, `auth_time`, and UserInfo.

The formal OAuth and OIDC papers in the source packet go beyond message syntax.
They define authorization, authentication, and session-integrity goals in a web
attacker model. That research influenced the review in three concrete ways:

1. A cryptographically valid response can still violate session integrity if it
   represents the wrong browser action or authentication event.
2. Cross-message binding must be reviewed as one protocol trace rather than as
   independently validated endpoints.
3. Redirect, browser, cookie, and attacker-controlled web behavior belong in the
   model, not outside it as generic HTTP concerns.

The final FAPI 2.0 attacker model is included as additional vocabulary for
attacker roles and capabilities. tiny-idp does not claim FAPI implementation or
FAPI security. The document helps an intern state whether an attacker controls a
browser, client, endpoint configuration, network segment, or token rather than
using the undifferentiated word “attacker.”

## Threat model for the strict tiny-idp profile

The relevant attacker may:

- send arbitrary authorization and token requests;
- operate an unregistered or registered malicious client subject to configured
  client validation;
- control browser query and form bytes;
- replay observed opaque handles, codes, or tokens if they are disclosed;
- submit duplicate parameters and headers;
- open concurrent browser tabs and concurrent refresh requests;
- induce malformed, oversized, or interrupted HTTP traffic;
- exploit stale browser cookies;
- race administrative disable/revocation with pending interactions;
- send forged forwarding headers through an untrusted peer;
- trigger storage and audit failures in the fault model;
- read ordinary application logs, which is why secrets are excluded.

The base model assumes:

- TLS protects external browser/client traffic;
- cryptographic primitives and the Go runtime behave according to their
  contracts;
- protected key/secret files are not disclosed;
- the configured host and immediate trusted proxy are not malicious;
- SQLite runs in the documented single-writer local-filesystem topology;
- registered redirect URIs and client metadata are administered correctly.

Compromise of process memory, root, signing private key, or token HMAC secret is
handled by incident response and rotation, not prevented by protocol parsing.

## Fosite composition in the current project

`NewProvider` builds a Fosite configuration with code, access, ID-token, and
refresh lifetimes. It composes a core HMAC strategy for opaque OAuth artifacts
and an OpenID strategy backed by the current active RSA signing key.

The production handler factories are explicitly listed:

- `OAuth2AuthorizeExplicitFactory` implements authorization-code issuance;
- `OAuth2PKCEFactory` binds code exchange to the verifier;
- `OAuth2RefreshTokenGrantFactory` implements refresh grants and reuse behavior;
- `OpenIDConnectExplicitFactory` adds OIDC authorization and ID-token state;
- `OpenIDConnectRefreshFactory` carries OIDC semantics through refresh;
- `OAuth2TokenIntrospectionFactory` supplies internal token introspection used by
  UserInfo.

The list is an allowlist. Unsupported implicit, password, client-credentials,
device, CIBA, token-exchange, and DPoP semantics are not accidentally enabled by
`ComposeAllEnabled`.

Fosite owns protocol request parsing and response construction. tiny-idp owns
product behavior around it: browser authentication, consent, persistent client
and user state, cookies, limiter identity, audit, security events, and host
readiness.

## Endpoint map

`Provider.Handler` mounts the strict handler at the configured issuer path and
applies security headers.

| Endpoint | Method | Authority transition |
|---|---|---|
| discovery | GET | publishes metadata, no user authority |
| JWKS | GET | publishes verification keys |
| authorize | GET/POST | validates request, authenticates/consents, issues code |
| token | POST | consumes code or refresh and creates token generation |
| UserInfo | GET/POST | discloses scoped claims for access token |
| health | GET | reports lifecycle |
| readiness | GET | reports production dependencies |

There is no public debug route. Token introspection is composed for internal
UserInfo validation but not necessarily exposed as an HTTP endpoint.

## Discovery

Discovery publishes issuer, authorization endpoint, token endpoint, UserInfo,
JWKS URI, supported response and grant types, subject type, signing algorithm,
scopes, claims, and PKCE method.

Metadata is a contract with clients. Advertising a grant, algorithm, or claim
that the provider cannot implement causes interoperability and security errors.
The issuer and endpoint URLs must be derived consistently from the canonical
issuer, including any path prefix.

The security-header test confirms discovery uses JSON content type, CSP,
no-referrer, nosniff, and frame denial as configured. Cache behavior must be
reviewed separately for metadata and keys.

## JWKS

JWKS publishes public verification material for active and retained retired
signing keys. It never includes private PEM. Each key has an identifier used in
ID-token headers.

Planned rotation relies on overlap: a token signed before rotation remains
verifiable while its key remains in JWKS. Emergency purge intentionally ends
that property after compromise.

The active key is loaded again before authorization issuance. A valid pending
interaction cannot continue if signing capability disappears.

## Authorization parameter authority catalog

### `client_id`

The client selects the identifier in the initial request. Fosite loads the
registered client and validates its policy. Resume uses the stored client ID and
current store record. A POST field cannot replace it.

### `redirect_uri`

The client supplies an exact URI registered for that client. Fosite validates it
before any safe error redirect. The value is stored in the interaction and
rechecked against current client metadata on resume.

Prefix, host-only, or pattern matching would create redirect confusion. Client
validation uses exact configured values.

### `response_type`

The strict profile supports code response. Unsupported types are rejected before
interaction. Browser POST does not select a response type.

### `scope`

The client requests a space-delimited set. Fosite and tiny-idp verify allowed
scopes. The server grants requested scopes through the requester. Consent, ID
token claims, UserInfo claims, and refresh behavior depend on the granted set.

The interaction displays exact bound scopes when consent is required. POST
cannot expand them.

### `state`

The client uses `state` to correlate request and callback and defend its own
browser session. tiny-idp preserves the original validated value through the
canonical request and returns it. The provider does not interpret it as its own
CSRF or interaction identifier.

The mutation regression submits attacker state values and proves the callback
contains the original.

### `nonce`

OIDC clients use nonce to bind the ID token to the authentication request. It is
stored in the requester/session state and included in ID-token claims according
to Fosite's OIDC handler. Browser POST cannot replace it.

### `code_challenge` and `code_challenge_method`

Public clients require PKCE. tiny-idp clients can declare `RequirePKCE`; public
client validation sets the correct grant policy. The strict profile supports
S256, not plain.

The authorization code stores the challenge binding. Token exchange supplies
the verifier. Fosite hashes it and compares with the challenge before code
consumption succeeds.

### `prompt`

`prompt=login` creates a fresh-login obligation. `prompt=none` excludes
interaction and returns protocol errors when an action is needed. Other prompt
semantics must be evaluated against OIDC Core before support is advertised.

The value belongs to the canonical request and derived obligation. The browser
form does not need to return it because the obligation is stored explicitly.

### `max_age`

The value is a non-negative decimal integer in seconds. Parsing distinguishes
absence from zero and error. The provider compares current clock with session
`AuthTime`. Exceeded age creates fresh-login obligation.

Malformed, negative, and overflowing values are invalid requests and never
render credentials.

### `claims`

OIDC claims requests can carry structured requirements. The current strict
profile must not silently lose or misrepresent them across interaction. Any
future support needs canonical preservation, explicit validation, and claim
construction semantics. Absence of full support should be documented rather than
approximated from POST.

### `acr_values`

Requested authentication context affects which authentication method may satisfy
the client. Current password flow does not implement a general ACR selection
engine. Publishing support before native semantics would be misleading.

### `login_hint`

A hint can prefill UX but is not authenticated identity. It must never bypass
credential verification or become an authoritative subject.

### `id_token_hint`

An ID-token hint carries prior authentication context and requires validation.
It must not be decoded without signature/audience/issuer checks to choose a user
or redirect.

### `ui_locales`

Locale preference may affect presentation, not authorization binding. The first
metamorphic test varies it and requires equivalent issuance/state outcome.

### `request`

Signed request objects are not supported in the strict profile. The provider
returns `request_not_supported`. It may parse unverified payload only for limited
error routing after validating ordinary client/redirect inputs; unverified claims
must not create a redirect destination.

### `response_mode`, `audience`, `resource`, and extensions

Each parameter needs an explicit classification: supported and canonicalized,
rejected, or ignored only where the governing specification permits. Browser
continuation must not drop a supported security-relevant extension. The opaque
canonical record makes preservation possible, but implementation still needs
native semantics.

## Duplicate parameters

HTTP query/form APIs can return first, last, or all values. Security parameters
must not be resolved by accidental library precedence. Duplicate client,
redirect, response type, PKCE, state, or interaction values need explicit
fail-closed behavior or standards-defined processing.

The hardening tests include duplicate and mutation cases. An intern adding a
parameter should add duplicates to the test table before implementing success.

## Canonicalization

`canonicalAuthorizeForm` builds a fresh `url.Values` from the validated Fosite
requester rather than retaining the raw request object. It includes client ID,
redirect URI, response type, requested scopes, state, nonce, PKCE, prompt,
`max_age`, and supported form values available through the requester.

The map is deep-copied, hashed, stored, and reconstructed into a new synthetic
request on resume. Fosite validates the reconstructed request again.

Canonicalization is not arbitrary normalization. Changing URI encoding, scope
order, or duplicated values can change semantics. Every transformation needs a
specified equivalence relation.

## Request digest and client generation

`authorizeRequestDigest` hashes deterministic encoded canonical values. It
supports integrity comparison and traceability without storing a signature in
the browser.

`clientGenerationHash` hashes security-relevant registered metadata. Resume
compares it with the current client. The hash includes updated timestamp so an
administrative rewrite invalidates pending interactions even when a subset of
fields returns to the same values.

These digests are local integrity mechanisms, not public signatures or proof of
who authored the request.

## Browser binding and CSRF

CSRF uses a browser cookie and form token derived/validated by the provider. The
cookie lifetime is refreshed for each interaction while retaining the nonce so
concurrent tabs remain valid.

Browser binding hashes selected browser/session context into the interaction.
It narrows stolen-handle replay. Neither mechanism makes hidden OAuth fields
authoritative.

The interaction form contains only opaque handle, CSRF, login/password when
needed, and consent action. Tests search the rendered HTML for forbidden
protocol field names.

## Session states

`readBrowserSession` returns user, session, state, and error. The explicit state
distinguishes missing, active, inactive, and storage failure.

An active session requires:

- valid signed/opaque cookie format;
- stored session exists and is unexpired/unrevoked;
- referenced user exists and is enabled;
- current time satisfies session policy.

Storage failure is not “no session.” It returns service unavailable and does not
collect credentials. This prevents fail-open authentication and avoids asking a
user to type a password when the server cannot persist the result.

## Authentication time

`Session.AuthTime` records the authentication event, while `CreatedAt` records
session object creation and `LastSeenAt` records activity. Freshness compares
against `AuthTime`.

When existing session is accepted, ID-token `auth_time` uses session auth time.
When password authentication succeeds, the provider uses injected current time,
creates a new session, and emits authentication success.

Using last-seen time would allow ordinary browsing to extend authentication
freshness without a new authentication event.

## Password authentication path

The provider normalizes login and first evaluates the stored required action. An
empty login is rejected when login or fresh login is mandatory.

The limiter receives verified client ID, resolved client address, and normalized
login. The authenticator receives context, login, password, and metadata. It
returns a typed result or typed error.

Unavailable authentication and saturated work map to temporary service failure.
Invalid credentials map to unauthorized with stable audit reason. Success
returns the user and lifecycle flags.

`MustChangePassword` must stop token issuance or route to a Go-owned password
change continuation. Ignoring it would make a temporary credential sufficient
for tokens. This remains a canonical example of why typed success is not a
boolean.

## Browser session creation

Session creation generates a random handle, stores only its hash, records user
and auth time, and sets a Secure/HttpOnly/SameSite cookie according to host
policy. Storage failure returns an error before login success is treated as a
complete browser state.

The session cookie is a bearer reference. TLS and browser cookie attributes
protect transport; durable revocation and expiry protect server acceptance.

## Consent policy path

After authentication, the provider reloads the user and client and evaluates
`ConsentPolicy.RequireConsent` over exact requested scopes.

Development can use `AlwaysSkipConsent`; production defaults to stored consent
when no explicit policy is supplied. Production behavior remains explicit in
validation and docs.

If consent is required, only action `approve` satisfies it. `deny` consumes a
denied outcome and writes OAuth access denied. Missing action returns a bad
request and leaves no artifact.

Recorded consent is bound to user, client, scopes, grant time, expiry, and
revocation. Scope comparison must avoid treating a subset grant as approval for
additional scopes.

## Authorization issuance path

`finishAuthorize` grants the requested scopes/audience, constructs the OIDC
session, and asks Fosite for an authorize response. The OIDC session includes
issuer, subject, audience, nonce, authentication time, requested claims, and
user claims permitted by scopes.

SQLite begins the authorization lifecycle before Fosite storage handlers run.
Interaction terminal consume and code/PKCE/OIDC rows commit together. Only after
commit does the provider emit terminal/artifact security evidence and write the
redirect response.

Development memory has a different persistence implementation and must not be
used as evidence for production SQL atomicity.

## ID-token claim construction

`newOIDCSession` constructs standard claims from the stored user and request.
Subject is stable `User.Sub`, not login or email. Audience derives from client.
Issuer derives from canonical issuer. Nonce derives from authorization request.
`auth_time` derives from actual session authentication.

Claims such as email and profile fields depend on granted scopes. Generic groups,
roles, tenant, preferred username, and locale are provider-specific top-level
claims supported by the domain model. Explicit custom claim design must prevent
overwriting protected standard claims.

The signing key ID and algorithm are represented in headers. Private key
selection occurs through `activePrivateKey` and current store state.

## Authorization response and redirect

Fosite creates the code and redirect parameters. The provider writes only to the
validated redirect URI. `state` is returned unchanged from canonical request.

Response write failure after commit is ambiguous to the client: the code may
exist although the browser did not receive it. Repeating the form is rejected by
one-time interaction state. This is secure but can reduce availability.

## Token endpoint request processing

The token endpoint accepts POST form and applies a pre-authentication address
rate limit. Fosite parses grant type and client authentication. Public PKCE
clients do not use a client secret; confidential clients use stored BCrypt secret
hashes or explicitly supplied migration secrets.

After successful client authentication, verified client identity can be used for
post-authentication limiting and audit.

`NewAccessRequest` validates code or refresh credential, client, redirect,
verifier, expiry, scopes, and stored session. `NewAccessResponse` performs
transactional mutations and constructs output.

Errors are returned through Fosite's OAuth token error writer with JSON and
appropriate status. Raw credentials are never included in error description.

## Persisted requester

The SQL Fosite adapter serializes a normalized requester rather than Go interface
internals. `persistedRequest` contains request ID/time, client, requested/granted
scope, form, session, and requested/granted audience.

`persistedClient` contains ID, secret hash where required, redirect URIs, grant
and response types, scopes, audience, public flag, and per-client token TTLs.

`persistedSession` contains ID-token claims/headers, expiry map, username, and
subject. Restore reconstructs Fosite request/client/session values.

Persistence format is security-sensitive because later token validation trusts
these bindings. Tests cover provider restart and restoration.

## Code exchange and PKCE

The token request supplies code, redirect URI, client ID/authentication, and
`code_verifier`. Fosite locates the code requester and PKCE session, compares
S256 verifier, and enforces one-time use.

The SQL transaction invalidates the code while creating access and refresh
sessions. Eight failpoints prove rollback and retryability.

A stolen code without verifier is insufficient for a public client. A stolen
verifier without code is insufficient. Disclosure of both remains a bearer
threat.

## Access tokens

The strict Fosite strategy creates opaque HMAC-protected access tokens. The
database indexes signature and stores requester state. UserInfo introspects the
token through Fosite and reconstructs subject/client/scope.

Access tokens are bearer capabilities. Storage and logs use signatures/hashes or
metadata, not raw values. TLS and Authorization header transport are mandatory.

Expiry and revocation are checked at introspection. Token-secret rotation makes
previous opaque tokens invalid even if rows remain.

## Refresh tokens and families

Refresh tokens are opaque bearer capabilities with longer lifetime. Rotation
deactivates the exact presented generation and creates a new generation in one
transaction.

Presenting an inactive old token triggers reuse behavior and family revocation.
This assumes old-token reuse may mean theft. Legitimate clients must serialize
refresh to avoid self-triggered revocation.

Refresh can narrow scopes according to Fosite configuration but must not expand
beyond the original grant. OIDC refresh preserves subject and session claim
semantics.

## UserInfo exact processing

The handler accepts GET or POST. It reads raw Authorization header instances and
rejects duplicates. It rejects any `access_token` query or form parameter,
including when a valid header is also present.

The header must use one Bearer scheme and token. Fosite introspects it as an
access token into an OIDC session. The provider loads current user state and
constructs claims allowed by granted scopes.

Every response uses no-store/no-cache protections. Invalid credentials return a
Bearer challenge with `invalid_token`. Multiple transport methods return
`invalid_request`. Unsupported methods do not process credentials.

Current-user disable after token issuance can therefore block UserInfo even
while the token is cryptographically valid, according to local policy.

## Error classes

### Local HTTP error

Use when no validated redirect exists or the failure is a server/browser
interaction problem. Examples include malformed form, CSRF, storage unavailable,
or invalid interaction.

### Authorization redirect error

Use only after validating client and redirect. Preserve original state. Examples
include `login_required`, `consent_required`, `access_denied`, and supported
Fosite authorization errors.

### Token JSON error

Token endpoint never redirects. It returns OAuth error JSON and appropriate
status.

### UserInfo challenge

Invalid bearer credentials use `WWW-Authenticate`. Ambiguous transport is an
invalid request. Cache remains disabled on errors.

## Unsupported request object case

The provider detects the `request` parameter before ordinary processing and
returns `request_not_supported`. A JWT payload can be base64-decoded without
verification only to recover candidate display/error context; it cannot establish
client or redirect authority.

`clientAllowsRedirect` loads registered metadata before any redirect. If
validation fails, the error stays local. The regression checks stable error and
state behavior.

## Audit and security-event mapping

Protocol transitions produce two evidence streams.

Audit examples:

- login success/failure/rate-limited/unavailable;
- authorization accepted/denied/rejected;
- token lifecycle outcomes;
- administrative client/user/key/backup changes.

Security events:

- interaction created with required actions;
- authentication satisfied;
- consent approved or denied;
- interaction terminal outcome;
- authorization artifacts committed;
- token lifecycle committed.

Events are emitted after authoritative transitions and exclude raw artifacts.
Audit delivery and security-event delivery have separate failure counters.

## Formal property map

### Authorization

Only the authenticated and consenting resource owner can cause a code bound to
the validated client/request, subject to configured policy.

Concrete mechanisms: Fosite validation, password authenticator, explicit
consent, canonical interaction, mutable-state revalidation, atomic artifact
commit.

### Authentication

An RP receiving a valid ID token can attribute its subject/authentication claims
to the issuer and request, subject to key and client validation.

Concrete mechanisms: signed ID token, issuer/audience/nonce, stable subject,
actual auth time, current key.

### Session integrity

The authorization response corresponds to the browser/user action and client
request that initiated it.

Concrete mechanisms: state preservation, browser/CSRF binding, required actions,
one-time opaque interaction, exact redirect, revalidation.

### Token confidentiality

Bearer artifacts are not disclosed through forbidden transports, logs, traces,
or untrusted redirects under the stated host assumptions.

Concrete mechanisms: TLS requirement, header-only UserInfo, secret-free evidence,
validated redirects, protected files.

### Replay resistance

Interactions and codes are one-time; refresh reuse revokes the family.

Concrete mechanisms: atomic consume, transactional invalidation, conditional
rotation, expiry, PKCE.

## Test map by protocol concept

| Concept | Representative test |
|---|---|
| Complete code flow | `TestStrictAuthorizationCodeFlow` |
| Session + silent authorize | `TestBrowserSessionSilentAuthorizeAndPromptNone` |
| Forced fresh login | `TestForcedPromptLoginCannotReuseExistingSession` |
| Authentication age | `TestExpiredMaxAgeCannotReuseExistingSession` |
| Strict parse | `TestMalformedMaxAgeNeverRendersCredentialForm` |
| Consent denial | `TestConsentDenialUsesOAuthAccessDenied` |
| Canonical state | `TestAuthorizationStateCannotBeMutatedOnResume` |
| Independent tabs | `TestConcurrentTabsKeepIndependentInteractions` |
| One-time interaction | concurrent/sequential replay tests |
| Current client/user | mutation/disable tests |
| Interaction expiry | injected-clock expiry test |
| Form authority | `TestInteractionFormContainsNoProtocolContinuation` |
| Code transaction | redemption failpoint table |
| Refresh reuse | SQLite refresh reuse and linearizability tests |
| UserInfo transport | phase-3 rate-limit/UserInfo tests |
| Restart persistence | Fosite SQLite restart test |
| Secret rotation | prior opaque token invalidation test |

## Code review method for a new parameter

1. Identify governing RFC/OIDC section.
2. State syntax, duplicates, and invalid-value behavior.
3. Identify which principal controls it.
4. State whether it changes validation, authentication, consent, claims, or
   response construction.
5. Add it to canonical request only after validation.
6. Decide whether it creates a stored required action.
7. Decide whether mutable state must be revalidated.
8. Ensure resume never accepts it from browser POST.
9. Add success, malformed, duplicate, mutation, replay, prompt-none, and storage
   failure tests as applicable.
10. Update discovery only after support is complete.

## Common protocol mistakes

### Trusting a decoded JWT

Base64 decoding gives claims, not authenticity. Validate signature, issuer,
audience, expiry, nonce, and intended use before authority.

### Reconstructing from hidden fields

Even signed fields omit mutable-state revalidation and one-time lifecycle unless
the server represents those separately.

### Equating authentication and authorization

A known user has not necessarily granted this client and scope set.

### Equating session and freshness

An active session may be too old or explicitly disallowed by `prompt=login`.

### Consuming before complete issuance

One-time state and resulting artifacts must commit atomically.

### Accepting convenient bearer transport

Query/form bearer credentials expand leakage and ambiguity.

### Redirecting before validation

Raw client input cannot select where security errors are sent.

### Advertising partial support

Discovery is an interoperability promise. Do not publish unimplemented grants,
claims, algorithms, or prompt semantics.

## Decision records

### PS-1: strict code + S256 profile

- **Decision:** compose only explicit code, PKCE, refresh, OIDC, and internal
  introspection handlers.
- **Reason:** reduce enabled protocol surface and follow modern browser-client
  guidance.
- **Consequence:** other grants require explicit design and tests.

### PS-2: server-owned canonical continuation

- **Decision:** persist validated request and obligations behind opaque handle.
- **Reason:** browser must not carry protocol authority across login/consent.
- **Consequence:** store and lifecycle are part of authorization correctness.

### PS-3: fresh authentication as stored obligation

- **Decision:** distinguish active session from post-interaction authentication.
- **Reason:** `prompt=login` and `max_age` are temporal semantics.
- **Consequence:** blank submit cannot fall back to old session.

### PS-4: explicit consent outcome

- **Decision:** approval/denial is native server-bound action.
- **Reason:** omission is not authorization and browser scopes are not authority.
- **Consequence:** denied interaction has terminal state and OAuth error.

### PS-5: header-only UserInfo bearer

- **Decision:** reject query/form/mixed credentials.
- **Reason:** reduce disclosure and ambiguity.
- **Consequence:** Fosite generic extraction is not used directly.

### PS-6: typed authentication outcomes

- **Decision:** handle unavailable, saturated, invalid, success, and lifecycle
  flags explicitly.
- **Reason:** password verification success may still prohibit issuance.
- **Consequence:** future password-change continuation remains Go-owned.

## Extended exercises

1. Annotate one complete authorization request with principal ownership.
2. Show how exact redirect validation prevents an open redirect.
3. Explain state versus nonce versus interaction handle versus CSRF token.
4. Write canonicalization rules for one new extension parameter.
5. Design duplicate handling for `max_age`.
6. Explain why login hint is not authenticated identity.
7. Define the validation required for an ID-token hint.
8. Trace `auth_time` from password event to ID-token claim.
9. Explain a stored-consent subset/superset case.
10. Trace persisted requester through process restart.
11. Predict all rows after a PKCE verifier mismatch.
12. Explain why a code may be consumed when response delivery fails.
13. Draw refresh rotation followed by reuse.
14. Review UserInfo response for an access token lacking email scope.
15. Classify each error path as local, redirect, token JSON, or challenge.
16. Explain why unsupported request-object payload cannot select redirect.
17. Map security events to the formal property they support.
18. Identify a protocol property not represented by current monitor.
19. Review discovery for an advertised feature and locate its implementation.
20. Propose safe first-class support for one currently unsupported parameter.

## Chapter review checklist

- Can the reader state the attacker capabilities and assumptions?
- Can the reader explain every composed Fosite handler?
- Can the reader classify authorization parameter authority?
- Can the reader explain canonical request, digest, generation, CSRF, and browser
  binding separately?
- Can the reader distinguish session existence, validity, and freshness?
- Can the reader handle typed authentication and consent outcomes?
- Can the reader trace code exchange, access, refresh, and UserInfo?
- Can the reader choose safe error delivery?
- Can the reader map formal authorization/authentication/session integrity to
  concrete code and tests?
- Can the reader review a new parameter without weakening cross-message binding?

## Key points

- OAuth and OpenID Connect security depends on preserving bindings across
  messages, actors, and time.
- A browser session and fresh authentication are different protocol facts.
- CSRF and protocol-continuation integrity solve different problems.
- Authentication, consent, code issuance, token rotation, and UserInfo are
  distinct authority transitions.
- Cryptographic validity does not repair incorrect state semantics.

## Research and standards packet

- `sources/rfc6749-oauth2-authorization-framework.md`
- `sources/rfc6750-bearer-token-usage.md`
- `sources/rfc9700-oauth-security-bcp.md`
- `sources/openid-connect-core-1.0-errata2.md`
- `sources/paper-formal-security-analysis-oauth2.md`
- `sources/paper-formal-security-analysis-openid-connect.md`
- `reference/04-authorization-interaction-and-protocol-robustness-review.md`
