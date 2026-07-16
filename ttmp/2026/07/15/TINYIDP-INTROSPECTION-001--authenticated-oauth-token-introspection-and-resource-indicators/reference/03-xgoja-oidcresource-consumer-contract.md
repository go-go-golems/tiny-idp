---
Title: xgoja oidcresource consumer contract and implementation handoff
Ticket: TINYIDP-INTROSPECTION-001
Status: active
Topics:
    - auth
    - oidc
    - xgoja
    - api-design
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        RFC 7662 provider contract and bearer-only issuance policy.
        Authenticated RFC 7662 bearer-only provider contract
    - Path: repo://internal/oidcmeta/discovery.go
      Note: |-
        Discovery fields consumed by resource servers.
        Issuer discovery and device metadata contract
    - Path: repo://pkg/embeddedidp/provider_test.go
      Note: |-
        Production TLS discovery and authenticated introspection smoke.
        Production TLS resource-server smoke
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
    - https://www.rfc-editor.org/rfc/rfc8628.html
Summary: Stable contract and implementation checklist for a future xgoja oidcresource API authenticator consuming tiny-idp opaque bearer tokens.
LastUpdated: 2026-07-16T13:50:00-04:00
WhatFor: Implement an xgoja resource-server middleware without coupling an application to tiny-idp storage or confusing authentication with local authorization.
WhenToUse: Before adding a provider-backed API authentication module to go-go-goja or an xapp.
---


# xgoja `oidcresource`: consumer contract and handoff

## Purpose

`oidcresource` is a resource-server component, not an OIDC browser-login
client. It receives an API request containing a tiny-idp opaque bearer token,
calls the issuer's authenticated RFC 7662 endpoint, validates the constrained
response, and supplies a verified principal to application code. It must not
read the identity provider's SQLite database, inspect opaque token bytes, or
use `/userinfo` as an authorization API.

The resulting split is deliberate:

```text
browser / device CLI -- bearer token --> xgoja API
                                      |
                                      v
                              oidcresource middleware
                                      |
                    Basic-authenticated POST /introspect
                                      |
                                      v
                                  tiny-idp decision
                                      |
                                      v
                       application scope + ownership decision
```

## Provider discovery and static configuration

At application startup, discover the issuer over HTTPS and validate:

- `issuer` is exactly the configured issuer URL.
- `introspection_endpoint` is HTTPS and belongs to the configured issuer
  deployment policy.
- `introspection_endpoint_auth_methods_supported` contains exactly the method
  the component supports: `client_secret_basic`.
- `device_authorization_endpoint` and the device-code grant may be used by a
  companion CLI/device-login module, but are not required by request middleware.

The xapp host provides the resource-client ID and secret through its secret
configuration, never JavaScript source or browser-delivered configuration.

```json
{
  "issuer": "https://idp.example.test/idp",
  "resourceClientID": "inbox-api",
  "resourceAudience": "https://inbox.example.test/api",
  "requiredScopes": ["inbox.read"],
  "introspectionCacheMaxAge": "30s"
}
```

The resource client is created in tiny-idp with `CanIntrospect: true`, a BCrypt
secret hash, and the same exact audience. The OAuth client that receives a
token is a different client and needs only permission to request that audience.

## Wire contract

For every uncached bearer token, make this request:

```http
POST /idp/introspect HTTP/1.1
Authorization: Basic base64(inbox-api:secret)
Content-Type: application/x-www-form-urlencoded

token=<incoming bearer token>
```

Interpret results as follows:

| Provider result | Meaning | Middleware action |
| --- | --- | --- |
| `200 {"active":false}` | Token unusable for this API, without a disclosed reason. | Return API `401`; do not retry. |
| `200 active:true` and valid fields | Current issuer decision. | Construct principal, then apply route and local policy. |
| `401 invalid_client` | xapp deployment secret/client is wrong, disabled, public, or lacks capability. | Fail closed; alert operator; normally return API `503`. |
| `429 temporarily_unavailable` | Provider rate limit. | Fail closed with `503`; apply bounded retry/backoff only outside request hot path. |
| TLS/network/5xx failure | Provider cannot decide. | Fail closed with `503`; never treat a stale positive cache entry as indefinitely valid. |

An active response contains only:

```ts
type IntrospectionSuccess = {
  active: true;
  iss: string;
  sub: string;
  client_id: string;
  scope: string;
  aud: string[];
  exp: number;
  iat: number;
  token_type: "Bearer";
};
```

Do not depend on profile fields, groups, roles, or tenant claims in this
response. If an xapp needs them, it should maintain application data keyed by
`sub`, or use a separately scoped identity-claims flow.

## Required validation algorithm

```text
authenticate(request):
  token = parse exactly one Authorization: Bearer header
  if missing or malformed: return unauthenticated

  decision = boundedCache.get(token) or introspectOverTLS(token)
  if decision.active is not true: return unauthenticated
  if decision.iss != configuredIssuer: return unauthenticated
  if decision.token_type != "Bearer": return unauthenticated
  if configuredAudience not in decision.aud: return unauthenticated
  if now >= decision.exp: return unauthenticated

  principal = { sub, clientID: decision.client_id,
                scopes: split(decision.scope), expiresAt: decision.exp }
  return principal

authorize(principal, route):
  if route.requiredScope not in principal.scopes: return forbidden
  if !applicationOwnsOrShares(principal.sub, route.object): return forbidden
  return allowed
```

`active:true` is authentication plus a delegated audience grant. It is never a
substitute for tenancy, object ownership, moderation, or application-specific
authorization.

## Cache policy

Cache only positive decisions under a token-derived key such as an HMAC of the
token with an application-local cache key. Never use the raw token as a metric
label, log field, or persistent cache key. The cache expiry is:

```text
min(provider exp, now + configured maximum positive TTL)
```

The default maximum should be short (for example 30 seconds) because
introspection is the revocation boundary. Negative results may be cached only
briefly to protect the provider from malformed-token floods; do not cache
`invalid_client`, rate-limit, TLS, or server failures as token decisions.

## xgoja module surface

The Go host should expose a narrow module rather than raw HTTP credentials to
application JavaScript:

```ts
const auth = require("xapp/oidcresource");

app.use(auth.requireBearer({ requiredScopes: ["inbox.read"] }));
app.get("/api/messages", async (req) => {
  const principal = auth.principal(req); // { sub, scopes, clientID, expiresAt }
  return messages.listVisibleTo(principal.sub);
});
```

The host owns discovery refresh, TLS transport, Basic secret handling,
timeouts, cache keying, audit/metric redaction, and principal attachment.
JavaScript supplies route scopes and application policy only. This boundary
keeps secrets out of scripts and makes the secure default reusable across
xapps.

## Implementation tasks for go-go-goja

1. Define an immutable Go configuration type and startup validation.
2. Implement HTTPS discovery with exact issuer/method checks.
3. Implement a timeout-bounded introspection client using Basic credentials.
4. Define a principal type and middleware/context attachment API.
5. Add token-HMAC cache keys, bounded TTLs, and redacted metrics.
6. Add table-driven tests for inactive, wrong issuer, audience, scope, expiry,
   invalid client, TLS failure, and local ownership denial.
7. Add an xgoja example with browser login, device CLI login, and one protected
   durable-object API route.

## Acceptance checklist

- [ ] The resource-client secret never reaches JS, browser bundles, logs, or metrics.
- [ ] Every protected route requires a bearer token and validates issuer,
  audience, token type, expiry, and scope.
- [ ] Every mutable/read-sensitive object operation has local ownership or
  tenant authorization after principal construction.
- [ ] API unavailability fails closed.
- [ ] Positive caching is bounded by both token expiry and a short local TTL.
- [ ] Device clients discover the same issuer metadata and request the API's
  registered audience.
