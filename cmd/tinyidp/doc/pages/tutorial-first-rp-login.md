---
Title: "Tutorial: first relying-party login"
Slug: tutorial-first-rp-login
Short: "Run tinyidp, point a relying party at it, complete an authorization-code login, and inspect the issued session state."
Topics:
- oidc
- testing
- login
Commands:
- serve
- print-config
Flags:
- issuer
- addr
- client-id
- redirect-uris
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial walks through the first complete relying-party login against tinyidp. It uses the default root issuer and the permissive `dev-client` so the protocol sequence is visible without extra client setup.

By the end, you will know which URLs must agree, what the browser login form does, what the token exchange produces, and how to inspect tinyidp's in-memory state after the flow.

## Prerequisites

You need:

- a tinyidp checkout or installed `tinyidp` binary;
- an OIDC relying party that can use the authorization-code flow;
- a redirect URI you can configure in that relying party.

The examples below assume the relying party callback is:

    http://localhost:3000/callback

That URI is accepted by the built-in `dev-client`.

## Step 1 — start tinyidp

Start the provider with the basic root-issuer config:

    tinyidp serve --config-file examples/configs/dev-root.yaml

If you are running from source:

    go run ./cmd/tinyidp serve --config-file examples/configs/dev-root.yaml

The provider listens on `127.0.0.1:5556` and advertises the issuer `http://localhost:5556`.

## Step 2 — confirm discovery

Fetch the discovery document:

    curl -s http://localhost:5556/.well-known/openid-configuration | jq '{issuer, authorization_endpoint, token_endpoint, jwks_uri}'

Expected shape:

    {
      "issuer": "http://localhost:5556",
      "authorization_endpoint": "http://localhost:5556/authorize",
      "token_endpoint": "http://localhost:5556/token",
      "jwks_uri": "http://localhost:5556/jwks"
    }

The relying party validates that the discovery `issuer` equals the issuer URL it was configured with. If these values differ, the client should reject the provider.

## Step 3 — configure the relying party

Configure your RP with:

    issuer:        http://localhost:5556
    client_id:     dev-client
    client_secret: (empty)
    redirect_uri:  http://localhost:3000/callback
    scope:         openid profile email

For a first login, use `dev-client`. It is public, does not require a secret, and does not require PKCE. After the first flow works, switch to `public-spa` or `web-app` if you need to test stricter client behavior.

## Step 4 — trigger login

Start a login from the RP. The browser is redirected to tinyidp's authorize endpoint. The authorize GET validates the request and renders a login form. Hidden fields carry the original OIDC request parameters into the POST.

At the login page, type:

    alice

Leave the password empty unless you are using a seeded users file that defines a password for Alice. Submit the form.

The authorize POST does four things:

1. It validates the same OIDC request parameters again.
2. It normalizes `login=alice` and resolves the scenario.
3. It creates an IdP session and a one-time authorization code.
4. It redirects the browser to the RP callback with `code` and `state`.

## Step 5 — let the RP exchange the code

The RP sends the authorization code to `/token`. tinyidp checks the client, redirect URI, code expiry, and optional PKCE verifier. It then returns an ID token and access token.

The ID token contains claims derived from the selected login:

    sub:             deterministic synthetic subject
    email:           alice@example.test
    email_verified:  true
    name:            alice
    iss:             http://localhost:5556
    aud:             dev-client

The access token is opaque. tinyidp stores it in memory and uses it to answer `/userinfo`.

## Step 6 — inspect provider state

After the login, inspect tinyidp's state from another terminal:

    curl -s http://localhost:5556/debug | jq .
    curl -s http://localhost:5556/debug/sessions | jq .
    curl -s http://localhost:5556/debug/codes | jq .
    curl -s http://localhost:5556/debug/tokens | jq .

The debug views show active sessions, outstanding codes, and issued tokens. Secret values are shortened so the output can be used for diagnosis without exposing full bearer tokens.

## Step 7 — reset state between tests

Clear in-memory state when you want a clean run:

    curl -s -X POST http://localhost:5556/debug/reset

This does not change the signing key or configured clients. It clears sessions, codes, access tokens, and refresh tokens.

## What you learned

The first login establishes the invariant every other tutorial builds on: the RP's configured issuer, the discovery issuer, token `iss`, client ID, and redirect URI must agree. Once that agreement holds, tinyidp can derive users from login names, issue signed ID tokens, and expose enough debug state to make relying-party tests understandable.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| RP rejects discovery. | Configured issuer does not equal discovery `issuer`. | Configure the RP with `http://localhost:5556` for this tutorial. |
| `redirect_uri not allowed for this client`. | The RP callback does not match the allowlist. | Use `http://localhost:3000/callback` or configure `--redirect-uris`. |
| Browser login succeeds but RP rejects token. | RP expects a different client ID, issuer, or JWKS. | Compare RP config with discovery and ID token claims. |
| Debug endpoints return 403. | Request is not from loopback. | Call debug endpoints from the same host. |

## See also

- `tinyidp help user-guide` — operational guide for everyday use.
- `tinyidp help tutorial` — scenario and failure-mode tutorial.
- `tinyidp help tutorial-seeded-users-and-claims` — deterministic users and claims.
- `tinyidp help tutorial-device-authorization` — device-code approval and polling.
- `tinyidp help reference` — full endpoint and config reference.
