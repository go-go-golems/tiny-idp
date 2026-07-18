---
Title: OAuth introspection and resource-server contract implementation guide
Ticket: TINYIDP-INTROSPECTION-001
Status: active
Topics:
    - auth
    - oidc
    - security
    - api-design
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: Fosite composition, endpoint routing, token issuance, and current UserInfo behavior.
    - Path: repo://internal/oidcmeta/discovery.go
      Note: Discovery metadata extended with the introspection endpoint.
    - Path: repo://pkg/idpstore/types.go
      Note: Persistent client model and opaque access-token model.
    - Path: repo://internal/fositeadapter/sqlstore.go
      Note: Durable requester persistence including granted audiences.
    - Path: repo://internal/admin/clients.go
      Note: Administrative client registration boundary.
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://www.rfc-editor.org/rfc/rfc9449.html
Summary: Intern-ready design and phased implementation plan for authenticated RFC 7662 introspection, resource indicators, audience policy, and the future xgoja consumer.
LastUpdated: 2026-07-15T21:53:44.921806453-04:00
WhatFor: ""
WhenToUse: ""
---

# OAuth introspection and resource-server contract implementation guide

## Executive Summary

Tiny-idp currently issues opaque OAuth access tokens. This is intentional:
token validity, revocation, refresh rotation, and DPoP binding are server-side
state. An xgoja API cannot verify such a token with JWKS, and `/userinfo` is an
identity endpoint rather than an API-authorization contract. This ticket adds
the missing protected-resource protocol: authenticated OAuth 2.0 Token
Introspection (RFC 7662), published through OIDC discovery, with deliberate
resource/audience policy and an implementation path for a later xgoja
`oidcresource` authenticator.

The endpoint is provider work. It does not make every tiny-idp client a token
inspector. A registered confidential resource server authenticates to
`POST /introspect`; tiny-idp validates the presented opaque access token with
Fosite and returns only the metadata that resource server may use. The response
is `active: false` for unknown, expired, revoked, malformed, or
not-authorized-for-this-resource tokens. This both prevents a token oracle and
lets callers fail closed.

```text
client -- token request (resource=Inbox API) --> tiny-idp
client <-- opaque access token ---------------- tiny-idp

client -- Authorization: Bearer token --------> Inbox API
Inbox API -- authenticated POST /introspect --> tiny-idp
Inbox API <-- active + sub + scope + aud ----- tiny-idp
Inbox API -- local subject/resource policy --> application action
```

## Problem Statement

The existing personal-inbox example correctly uses tiny-idp only for browser
OIDC login and uses application-owned `programauth` credentials for its CLI.
That works today. A platform-level API additionally needs to accept a
tiny-idp-issued device or authorization-code access token directly. The current
provider cannot safely support that because:

- access tokens are opaque; JWKS validates signed ID tokens, not access tokens;
- discovery publishes no introspection endpoint;
- `/userinfo` returns profile claims but not token audience, client, expiry,
  scope, token type, or confirmation key;
- clients do not yet declare the audiences they may request or whether they are
  privileged resource-server introspection clients;
- DPoP is enforced at tiny-idp’s UserInfo endpoint today, but an independent
  API needs its own proof/replay validation policy.

The security invariant is: a resource server accepts a token only after an
authenticated provider decision says that the token is active and authorized
for that resource, then after its own local authorization checks pass.

## Proposed Solution

### Public protocol

Discovery gains two standard metadata members:

```json
{
  "introspection_endpoint": "https://idp.example.test/introspect",
  "introspection_endpoint_auth_methods_supported": ["client_secret_basic"]
}
```

The endpoint accepts only TLS-protected `POST` form requests. It accepts only
HTTP Basic authentication for this first release; the caller is a confidential
client with a stored BCrypt secret. It rejects public clients, disabled clients,
missing/duplicate credentials, bearer authorization, and non-POST requests.
`private_key_jwt` is a separately designed extension, not a guessed fallback.

```http
POST /introspect HTTP/1.1
Authorization: Basic base64(resource-server-id:secret)
Content-Type: application/x-www-form-urlencoded

token=<opaque access token>&token_type_hint=access_token
```

For active access tokens the response has:

```json
{
  "active": true,
  "iss": "https://idp.example.test",
  "sub": "user-alice-fixed",
  "client_id": "inbox-cli",
  "scope": "openid inbox.read",
  "aud": ["https://inbox.example.test/api"],
  "exp": 1780000000,
  "iat": 1779996400,
  "token_type": "Bearer"
}
```

Inactive is exactly `{"active":false}` after a successfully authenticated
inspection request. Authentication failures remain HTTP 401 and invalid request
shape remains HTTP 400. Token strings, hashes, and detailed storage errors never
enter response descriptions, audit fields, metrics, or logs.

### Client and audience model

`idpstore.Client` gains two persistent, administrative fields:

```go
AllowedAudiences []string // resource identifiers this client may request
CanIntrospect    bool     // confidential resource server privilege
```

The authorization request accepts OAuth resource indicators through Fosite’s
existing `audience` parsing. Fosite validates the requested audience against the
issuing client’s `AllowedAudiences`, and tiny-idp records granted audiences in
the durable requester record. Token refresh preserves granted audience. The
introspection caller must have `CanIntrospect`, must be confidential, and may
receive an active result only when the token’s granted audiences intersect that
resource server’s `AllowedAudiences`.

No audience means no API resource authorization. Browser-only OIDC clients can
continue receiving ID tokens and sessions, but their access token must not be
accepted as an API credential by the new endpoint.

### Endpoint algorithm

```pseudocode
handleIntrospect(request):
    require POST + form token parameter
    authenticate HTTP Basic confidential client
    reject disabled/public/not-CanIntrospect client

    result = Fosite.NewIntrospectionRequest(request)
    if invalid token: return 200 { active: false }
    if token use != access token: return 200 { active: false }

    metadata = requester(result)
    if grantedAudience(metadata) ∩ caller.AllowedAudiences is empty:
        return 200 { active: false }

    return 200 active response built from metadata
```

`Fosite.NewIntrospectionRequest` already validates the opaque token through the
same storage/strategy used by token issuance. Tiny-idp wraps its response rather
than trusting Fosite’s generic writer so it can guarantee issuer, audience,
resource policy, and the exact redaction contract.

### DPoP policy

The first endpoint supports bearer tokens only. A DPoP-bound token must return
inactive until tiny-idp can supply trusted `cnf.jkt` metadata and the resource
server validates DPoP proof method, public URL, access-token hash, time, and
single-use `jti`. This explicit rejection avoids bearer downgrade. The future
response will add `token_type: "DPoP"` and `cnf: {"jkt":"..."}`; the xgoja
consumer will then own replay storage for its API origin.

## Design Decisions

| Decision | Rationale |
| --- | --- |
| RFC 7662 introspection, not JWT access tokens | Matches tiny-idp’s opaque, revocable server-side token model. |
| Authenticated confidential resource client | Prevents public token scanning and allows per-resource disclosure policy. |
| Audience intersection required | A valid token for one API is not automatically valid for another API. |
| Custom response writer | Fosite validates the token; tiny-idp owns product fields and redaction policy. |
| Bearer-only first release | DPoP requires resource-server-side proof/replay verification; accepting it as bearer is unsafe. |
| Separate from `programauth` | App-issued credentials remain valid for self-contained apps; provider-issued credentials are optional interoperability. |

## Alternatives Considered

- **Use `/userinfo` from xgoja.** Rejected: it is not audience/scope/resource
  authorization, and makes an API delegate token transport semantics to an
  identity endpoint.
- **Share tiny-idp SQL tables with xgoja.** Rejected: breaks deployment,
  encapsulation, migrations, and audit boundaries.
- **Issue JWT access tokens immediately.** Rejected for this release: solves
  remote validation but creates key/cache/revocation design work while
  contradicting the provider’s current opaque token model.
- **Let every confidential client introspect every token.** Rejected: client
  authentication alone prevents anonymous scanning but does not establish a
  resource authorization relationship.
- **Accept DPoP response metadata without proof validation.** Rejected:
  sender-constrained tokens must not silently become bearer tokens.

## Implementation Plan

### Phase A — Freeze the external contract

1. Add discovery fields and endpoint documentation.
2. Add client `AllowedAudiences` and `CanIntrospect` to memory/SQLite/SQL
   persistence, config/bootstrap/admin surfaces, validation, and safe display.
3. Configure Fosite clients with allowed audiences; add audience request tests.

### Phase B — Implement the endpoint

4. Mount `POST /introspect` under root and path issuers.
5. Authenticate resource clients; reject public/disabled/unprivileged callers.
6. Call Fosite introspection, apply access-token-only/audience checks, emit the
   constrained response, and add redacted audit/security events.
7. Add rate limiting separate from `/token` and UserInfo limits.

### Phase C — Prove lifecycle and confidentiality

8. Test active, unknown, malformed, revoked, expired, refreshed, and wrong
   audience tokens for memory and SQL stores.
9. Test public/missing/wrong resource-client authentication and ensure inactive
   responses reveal no token state.
10. Test root/path issuer discovery and strict TLS fixture behavior.

### Phase D — Consumer follow-up

11. Implement xgoja `oidcresource` against the stable endpoint.
12. Add subject mapping, scope-to-grant configuration, local ownership checks,
    cache bounds, and a strict cross-project smoke.
13. Add DPoP response/consumer support only with API-side proof replay storage.

## Open Questions

- Should `CanIntrospect` be named `ResourceServer` in the public admin API?
  This guide uses the explicit capability name because it describes the
  authorization decision rather than a deployment role.
- Is Basic authentication enough for the first production deployment, or should
  `private_key_jwt` ship before external hosting? Basic is adequate only with
  TLS, secret rotation, and restricted resource-server deployment.
- Do device clients need an explicit `resource` parameter exposed in all CLI
  helpers, or is audience setup initially limited to code-flow clients?
- What storage and retention policy will xgoja use for DPoP proof `jti` values?
  This must be shared/durable for multi-replica resource servers.

## References

- [RFC 7662: OAuth 2.0 Token Introspection](https://www.rfc-editor.org/rfc/rfc7662.html)
- [RFC 8707: Resource Indicators for OAuth 2.0](https://www.rfc-editor.org/rfc/rfc8707.html)
- [RFC 9449: OAuth 2.0 DPoP](https://www.rfc-editor.org/rfc/rfc9449.html)
- `internal/fositeadapter/provider.go`: current provider composition and
  server-side introspection primitive.
