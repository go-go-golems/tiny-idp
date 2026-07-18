---
Title: Operator guide for OAuth resource servers and token introspection
Ticket: TINYIDP-INTROSPECTION-001
Status: active
Topics:
    - auth
    - oidc
    - security
    - operations
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/admin_client.go
      Note: CLI registration flags for resource audiences and introspection capability
    - Path: repo://internal/fositeadapter/provider.go
      Note: Protocol implementation and failure semantics
    - Path: repo://internal/oidcmeta/discovery.go
      Note: Published introspection endpoint and supported caller authentication method
ExternalSources:
    - https://www.rfc-editor.org/rfc/rfc7662.html
    - https://www.rfc-editor.org/rfc/rfc8707.html
Summary: Registration, request, response, failure, and rollout procedures for a tiny-idp protected resource.
LastUpdated: 2026-07-15T22:30:00-04:00
WhatFor: Safely configure an API that accepts tiny-idp opaque access tokens.
WhenToUse: Before deploying an API resource server or diagnosing its introspection responses.
---


# Operator guide: resource servers and token introspection

## Scope and safety boundary

This guide configures an API as a tiny-idp OAuth protected resource. It does
not configure browser login. The API never reads tiny-idp's SQLite tables and
never attempts to validate opaque access tokens locally. It authenticates to
the issuer's introspection endpoint and applies local authorization after the
issuer says the token is active for that API.

Use TLS for the issuer and API. This first release supports only confidential
resource servers using `client_secret_basic`; do not put the secret in a browser
bundle, a public xgoja script, source control, logs, or response messages.

## Mental model

```text
OAuth client                         Resource server              tiny-idp
------------                         ---------------              -------
request audience ------------------> authorization/token
<---------------- opaque token -----

Authorization: Bearer <token> ----> API handler
                                     POST /introspect Basic ----->
                                     <---- active, sub, scope, aud
                                     local ownership/scope decision
```

The OAuth **client** receives a token. The resource **server** is a separate
confidential client registered with `CanIntrospect`. They may be deployed in
the same organization, but their IDs and secrets should be distinct.

## Registration procedure

The examples use a binary built from this checkout. Replace paths and names
with deployment values. `--db` addresses the persistent production SQLite
database, and the administrative command outputs the generated secret exactly
once.

### 1. Register the issuing application/client

For a device CLI that may request the inbox API, grant the resource identifier
to the CLI. Device clients are usually public, so they do not receive a client
secret or introspection privilege.

```sh
go run ./cmd/tinyidp admin --db /srv/tinyidp/idp.sqlite client create \
  --id inbox-cli \
  --public \
  --require-pkce \
  --grant-type urn:ietf:params:oauth:grant-type:device_code \
  --grant-type refresh_token \
  --scope openid \
  --scope profile \
  --scope inbox.read \
  --scope inbox.write \
  --audience https://inbox.example.test/api
```

For an authorization-code web client, include its exact callback URI and the
same audience if it needs an API token:

```sh
go run ./cmd/tinyidp admin --db /srv/tinyidp/idp.sqlite client create \
  --id inbox-web \
  --public \
  --require-pkce \
  --redirect-uri https://app.example.test/oauth/callback \
  --grant-type authorization_code \
  --grant-type refresh_token \
  --scope openid \
  --scope profile \
  --scope inbox.read \
  --audience https://inbox.example.test/api
```

### 2. Register the API resource server

The API itself gets a confidential client. It must list the resource indicators
it is allowed to inspect and explicitly receive the introspection capability.

```sh
go run ./cmd/tinyidp admin --db /srv/tinyidp/idp.sqlite client create \
  --id inbox-api \
  --generate-secret \
  --can-introspect \
  --grant-type authorization_code \
  --audience https://inbox.example.test/api
```

Store the returned `secret` in the API's secret manager. The command's normal
safe output shows `has_secret`, `allowed_audiences`, and `can_introspect`, but
never the persisted BCrypt hash.

### 3. Verify discovery before configuring the API

```sh
curl --fail --silent --show-error https://idp.example.test/.well-known/openid-configuration \
  | jq '{issuer, introspection_endpoint, introspection_endpoint_auth_methods_supported}'
```

The expected method list is `client_secret_basic`. A missing endpoint means the
deployment is running an older provider binary or has a routing/prefix error.

## API request contract

The resource server sends exactly one HTTP `Authorization` header containing
Basic credentials and one form `token` parameter. The provider accepts only
`POST` and `application/x-www-form-urlencoded`, uses a small body limit, and
sends `Cache-Control: no-store` and `Pragma: no-cache`.

```sh
curl --fail --silent --show-error \
  --user "inbox-api:${INTROSPECTION_SECRET}" \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "token=${ACCESS_TOKEN}" \
  https://idp.example.test/introspect
```

For a valid bearer access token granted to the inbox resource, the meaningful
shape is:

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

An API must require all of the following before handling a request:

1. HTTP status is 200 and `active` is exactly `true`.
2. `iss` equals the configured issuer exactly; do not trust an issuer supplied
   by the browser request.
3. `aud` contains the API's exact configured resource identifier.
4. `token_type` is `Bearer`. DPoP tokens are not supported by this release.
5. `scope` contains the route's required scope.
6. The application applies ownership/tenant/record-level authorization using
   `sub`; a valid OAuth token is not an automatic right to every record.

Pseudocode:

```text
authorization = incoming Bearer token
metadata = cached-or-live-introspect(authorization)
if metadata.active != true: deny(401)
if metadata.iss != configuredIssuer: deny(401)
if apiAudience not in metadata.aud: deny(403)
if requiredScope not in split(metadata.scope): deny(403)
if !applicationAllows(metadata.sub, requestedObject): deny(403)
allow()
```

## Failure semantics and incident handling

| Condition | Provider result | API behavior |
| --- | --- | --- |
| Wrong/missing/duplicate Basic credentials | `401 {"error":"invalid_client"}` | Treat as API configuration/secret incident, not end-user denial. |
| Unknown, malformed, expired, revoked, or wrong-audience token | `200 {"active":false}` | Deny the bearer request without revealing why to the caller. |
| Invalid request shape | `400 {"error":"invalid_request"}` | Fix API integration; do not retry blindly. |
| Endpoint rate limited | `429 {"error":"temporarily_unavailable"}` | Fail closed; use bounded backoff and alert on sustained failures. |
| Provider unavailable/TLS failure | transport failure | Fail closed. Do not accept a locally cached positive decision beyond its explicitly bounded TTL. |

If a resource-server secret is suspected compromised, rotate it with
`tinyidp admin --db ... client rotate-secret --id inbox-api`, deploy the new
secret atomically, and verify old workers are drained. Disable the client as an
emergency containment measure. Token expiry/revocation behavior is still
provider controlled; no token is written to audit events by this endpoint.

## Rollout checklist

- [ ] The issuer and API use production TLS and a trusted deployment path.
- [ ] The OAuth client has only the exact scopes and resource audiences it
  needs.
- [ ] The resource-server client is confidential, enabled, and has
  `can_introspect: true`.
- [ ] The resource-server secret is in a secret manager and not emitted in
  application diagnostics.
- [ ] Discovery reports the intended HTTPS introspection endpoint.
- [ ] The API validates `active`, `iss`, `aud`, token type, scope, and local
  ownership for every request.
- [ ] Metrics and audit alerts cover `introspection.rejected`,
  `introspection.inactive`, and rate-limit events without storing token values.
- [ ] The strict HTTPS smoke in `scripts/01-introspection-smoke.sh` succeeds
  against a non-production token first.

## Current limits and follow-up boundary

- The endpoint is bearer-only. It deliberately does not accept DPoP-bound
  tokens as bearer tokens; implementing DPoP requires API-origin proof and
  replay checks described by RFC 9449.
- This guide does not make a positive provider result a replacement for local
  authorization. It is an authentication and delegated-grant input.
- The next verification phase must prove expiry, revocation, and refresh
  rotation at the endpoint against SQLite and a real strict-TLS deployment.
