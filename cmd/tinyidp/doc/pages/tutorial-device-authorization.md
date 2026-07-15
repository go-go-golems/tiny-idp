---
Title: "Tutorial: OAuth Device Authorization Grant"
Slug: tutorial-device-authorization
Short: "Run a complete RFC 8628-style device-code login against tinyidp with curl."
Topics:
- oidc
- oauth2
- device-authorization
- testing
Commands:
- serve
Flags:
- issuer
- addr
- client-id
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial walks through tinyidp's native OAuth 2.0 Device Authorization Grant support. Use it when a command-line app, TV-style app, or constrained device needs the user to approve login in a browser while the app polls the token endpoint.

tinyidp is still a local/test identity provider. Device authorization state is in memory, approval uses seeded-user or synthetic-user login semantics, and all examples assume loopback HTTP.

## Start tinyidp

From the tinyidp repository root:

    go run ./cmd/tinyidp serve-dev \
      --issuer http://127.0.0.1:5556 \
      --addr 127.0.0.1:5556 \
      --client-id dev-client \
      --users-file examples/users/personal-inbox-users.yaml

The built-in `dev-client` is public and works for local device-flow experiments. The users file gives deterministic Alice/Bob subjects and fixture passwords.

## Start a device authorization request

In another terminal:

    curl -sS -X POST http://127.0.0.1:5556/device_authorization \
      -d client_id=dev-client \
      -d 'scope=openid profile email offline_access' | jq .

The response looks like this:

    {
      "device_code": "opaque-device-code",
      "user_code": "ABCD-EFGH",
      "verification_uri": "http://127.0.0.1:5556/device",
      "verification_uri_complete": "http://127.0.0.1:5556/device?user_code=ABCD-EFGH",
      "expires_in": 600,
      "interval": 5
    }

A real device displays `user_code` and `verification_uri` to the user. If it can show a QR code or clickable link, use `verification_uri_complete`.

## Poll before approval

The device polls `/token` with the standard device-code grant type:

    curl -sS -X POST http://127.0.0.1:5556/token \
      -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
      -d client_id=dev-client \
      -d device_code="$DEVICE_CODE" | jq .

Before approval, tinyidp returns:

    {
      "error": "authorization_pending",
      "error_description": "device authorization is pending"
    }

If the client polls faster than the returned `interval`, tinyidp returns `slow_down`. Back off before polling again.

## Approve in the browser

Open the complete verification URI, or open `/device` and type the user code manually:

    http://127.0.0.1:5556/device?user_code=ABCD-EFGH

For the checked-in personal-inbox fixture users:

- login: `alice`, password: `alice-password`
- login: `bob`, password: `bob-password`

Choose **Approve device**. Choosing **Deny device** makes token polling return `access_denied`.

## Poll after approval

After approval and after respecting the polling interval:

    curl -sS -X POST http://127.0.0.1:5556/token \
      -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
      -d client_id=dev-client \
      -d device_code="$DEVICE_CODE" | jq .

The successful response includes an access token. Because the requested scope includes `openid`, it also includes an ID token. Because the requested scope includes `offline_access`, it also includes a refresh token:

    {
      "access_token": "opaque-access-token",
      "token_type": "Bearer",
      "expires_in": 3600,
      "scope": "openid profile email offline_access",
      "id_token": "signed-jwt",
      "refresh_token": "opaque-refresh-token"
    }

The device code is one-time use. A second token request with the same `device_code` returns `invalid_grant`.

## Path-based issuer example

If an app expects a Keycloak-shaped issuer URL, start tinyidp with a path issuer:

    go run ./cmd/tinyidp serve-dev \
      --issuer http://127.0.0.1:5556/realms/personal-inbox \
      --addr 127.0.0.1:5556 \
      --client-id dev-client \
      --users-file examples/users/personal-inbox-users.yaml

The device endpoints are then available at both root and prefixed paths. Discovery for the prefixed issuer advertises:

    http://127.0.0.1:5556/realms/personal-inbox/device_authorization
    http://127.0.0.1:5556/realms/personal-inbox/token

Path issuers are URL-shape compatibility only. Device approval still uses tinyidp's generic seeded-user and scenario behavior.

## Debugging

Loopback-only debug endpoints help tests inspect state:

    curl -sS http://127.0.0.1:5556/debug/device-grants | jq .
    curl -sS -X POST http://127.0.0.1:5556/debug/reset | jq .

`/debug/device-grants` shows redacted device-code prefixes, user codes, status, client ID, scope, expiry, and slow-down count. It does not expose full device codes.
