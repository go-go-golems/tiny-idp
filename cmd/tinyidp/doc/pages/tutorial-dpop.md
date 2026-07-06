---
Title: "Tutorial: DPoP Sender-Constrained Tokens"
Slug: tutorial-dpop
Short: "Use DPoP proof JWTs to obtain and call tinyidp with sender-constrained access tokens."
Topics:
- oidc
- oauth2
- dpop
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

This tutorial explains tinyidp's DPoP support. DPoP, Demonstrating Proof-of-Possession, binds an OAuth access token to the public key in a signed proof JWT. The token is no longer sufficient by itself. The caller must also sign a fresh proof for the endpoint it is calling.

tinyidp implements DPoP for local and integration tests. Tokens are still opaque in-memory strings. The DPoP binding is stored as server-side token metadata, and `/userinfo` enforces the binding for DPoP-bound access tokens.

## Start tinyidp

    go run ./cmd/tinyidp serve \
      --issuer http://127.0.0.1:5556 \
      --addr 127.0.0.1:5556 \
      --client-id dev-client

Discovery advertises DPoP proof algorithms:

    curl -sS http://127.0.0.1:5556/.well-known/openid-configuration \
      | jq .dpop_signing_alg_values_supported

The current implementation accepts `ES256` and `RS256` proof JWTs.

## What a DPoP proof contains

A DPoP proof is a compact JWT in the `DPoP` HTTP header. Its header contains the public JWK. Its payload binds the proof to one HTTP request.

Header:

    {
      "typ": "dpop+jwt",
      "alg": "ES256",
      "jwk": {"kty":"EC","crv":"P-256","x":"...","y":"..."}
    }

Payload for the token endpoint:

    {
      "jti": "unique-proof-id",
      "htm": "POST",
      "htu": "http://127.0.0.1:5556/token",
      "iat": 1783380000
    }

Payload for `/userinfo` also includes `ath`, the base64url SHA-256 hash of the access token:

    {
      "jti": "unique-userinfo-proof-id",
      "htm": "GET",
      "htu": "http://127.0.0.1:5556/userinfo",
      "iat": 1783380005,
      "ath": "base64url-sha256-access-token"
    }

## Obtain a DPoP-bound token

Run a normal authorization-code flow, but include the `DPoP` header on the `/token` request. tinyidp validates the proof, computes the JWK thumbprint, stores it with the opaque access token, and returns:

    {
      "access_token": "opaque-access-token",
      "token_type": "DPoP",
      "expires_in": 3600,
      "scope": "openid profile email",
      "id_token": "signed-id-token"
    }

The same binding works for device-code token exchange. If the token request has no `DPoP` header, tinyidp preserves existing bearer behavior and returns `token_type: Bearer`.

## Call userinfo with a DPoP token

A DPoP-bound access token must be used with both headers:

    Authorization: DPoP <access-token>
    DPoP: <proof-jwt-with-ath>

The proof must be signed by the same key that was used at `/token`, must target `GET http://127.0.0.1:5556/userinfo`, must contain a fresh `jti` and `iat`, and must include the correct `ath` value.

These calls fail:

- `Authorization: Bearer <dpop-bound-token>` fails because the token is not a bearer token.
- `Authorization: DPoP <token>` without a `DPoP` proof fails.
- A proof signed by a different key fails.
- A proof with the wrong `ath` fails.
- Reusing the same proof JWT fails because tinyidp stores proof `jti` values in an in-memory replay cache.

## Refresh tokens

If the original token response includes `offline_access`, the refresh token is bound to the same DPoP key. Refreshing it requires a new DPoP proof for `POST /token` signed by the same key. The rotated refresh token remains bound to that key.

Unbound refresh tokens preserve old behavior. If an unbound refresh token is refreshed with a valid DPoP proof, the newly issued access and refresh tokens become DPoP-bound.

## Implementation limits

The first tinyidp DPoP implementation deliberately does not implement nonce support. It relies on `iat` freshness and an in-memory `jti` replay cache. This is enough for deterministic local tests of sender-constrained token handling.

Because tinyidp is a local test IdP, all DPoP state is in memory. Restarting tinyidp clears access tokens, refresh tokens, and proof replay history.

## See also

- `tinyidp help reference` — endpoint and behavior reference.
- `tinyidp help developer-guide` — implementation notes for server internals.
- `tinyidp help tutorial-device-authorization` — device-code grant support, which can also issue DPoP-bound tokens.
