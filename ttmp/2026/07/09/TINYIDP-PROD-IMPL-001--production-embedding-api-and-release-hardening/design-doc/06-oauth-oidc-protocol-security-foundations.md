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
