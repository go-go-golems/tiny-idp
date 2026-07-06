---
Title: "User guide"
Slug: user-guide
Short: "Run tinyidp as a local OIDC provider, configure clients, define fixture users, and troubleshoot relying-party tests."
Topics:
- oidc
- testing
- identity
- config
Commands:
- serve
- print-config
Flags:
- issuer
- addr
- client-id
- client-secret
- redirect-uris
- users-file
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

This guide explains how to use tinyidp as a local OpenID Connect provider for applications that act as relying parties. It focuses on operational use: choosing an issuer, configuring clients and redirects, defining deterministic fixture users, and diagnosing common failures.

Use `tinyidp help developer-guide` if you want to modify tinyidp itself. Use `tinyidp help reference` when you need a complete field or endpoint lookup.

## What tinyidp provides

tinyidp implements the OIDC authorization-code flow and OAuth device-code flow for local development and integration tests. It serves discovery, JWKS, authorize, device authorization, token, userinfo, logout, health, and debug endpoints. It stores all runtime state in memory and generates signing keys at startup.

The default server is intentionally local:

    tinyidp serve

This starts an issuer at `http://localhost:5556` and listens on `127.0.0.1:5556`. The default client is `dev-client`, a public client with no secret and permissive local redirect URIs.

## Choose the issuer and listen address

`issuer` and `addr` are different values.

| Field | Meaning | Example |
|---|---|---|
| `issuer` | URL advertised in discovery and written into ID tokens as `iss`. | `http://127.0.0.1:19087/realms/personal-inbox` |
| `addr` | TCP address the server binds to. | `127.0.0.1:19087` |

For a root issuer, they usually share the same host and port:

    tinyidp serve \
      --issuer http://127.0.0.1:19087 \
      --addr 127.0.0.1:19087

For a path-based issuer, only `issuer` contains the path:

    tinyidp serve \
      --issuer http://127.0.0.1:19087/realms/personal-inbox \
      --addr 127.0.0.1:19087

Path-based issuers are URL-shape compatibility. They make discovery advertise endpoints under the issuer path and make the server mount those routes. They do not change claim semantics or enable provider-specific behavior.

## Configure clients and redirects

Every authorize request names a client and a redirect URI. tinyidp rejects unknown clients and redirects that are not allowlisted.

The built-in clients are:

| Client | Use | Secret | PKCE |
|---|---|---|---|
| `dev-client` | First local tests and permissive development flows. | none | optional |
| `public-spa` | Browser SPA tests that should require PKCE. | none | required |
| `web-app` | Confidential web-app tests. | `dev-secret` | optional |

You can configure one client through flags or config files:

    tinyidp serve \
      --client-id personal-inbox-local \
      --redirect-uris http://127.0.0.1:19794/auth/callback

If the configured client ID matches a built-in client, tinyidp merges your redirect URIs into the built-in while preserving its defining behavior. For example, adding a redirect to `public-spa` does not remove its PKCE requirement.

## Use config files for repeatable setups

Config files put OIDC settings under the `oidc` section:

    oidc:
      issuer: http://127.0.0.1:19087
      addr: 127.0.0.1:19087
      client-id: personal-inbox-local
      redirect-uris:
        - http://127.0.0.1:19794/auth/callback
      users-file: examples/users/personal-inbox-users.yaml

Run with:

    tinyidp serve --config-file examples/configs/personal-inbox-root.yaml

Inspect without starting the server:

    tinyidp print-config --config-file examples/configs/personal-inbox-root.yaml

Checked-in examples live under `examples/configs/`:

| File | Purpose |
|---|---|
| `dev-root.yaml` | Basic root-issuer development setup. |
| `personal-inbox-root.yaml` | xgoja personal-inbox setup with a root issuer. |
| `personal-inbox-realm.yaml` | xgoja personal-inbox setup with a path-based issuer. |
| `public-spa-pkce.yaml` | Built-in public SPA client with PKCE required. |
| `confidential-web-app.yaml` | Built-in confidential web app with `dev-secret`. |

`users-file` paths are currently resolved relative to the process working directory. Run checked-in examples from the tinyidp repository root, or pass an absolute path.

## Define deterministic fixture users

Without a users file, any login name derives a synthetic user. That is enough for many tests. Use a users file when you need fixed subjects, fixed emails, fixture passwords, or predictable authorization claims.

    users:
      - login: alice
        password: alice-password
        sub: user-alice-fixed
        email: alice@example.test
        name: Alice Inbox
        email_verified: true
        groups: [inbox-users]
        roles: [writer]
        tenant: personal
        preferred_username: alice
        locale: en-US

Start with:

    tinyidp serve --users-file examples/users/personal-inbox-users.yaml

Seeded users override built-ins with the same login. If you define `alice`, logging in as `alice` uses your deterministic fixture, not the built-in synthetic Alice.

## Understand fixture passwords

Passwords are optional local fixture values. If a seeded user has no `password`, any submitted password is accepted. If a seeded user has `password: alice-password`, authorize POST must submit that exact value.

Wrong or missing fixture passwords return:

    HTTP 401 invalid login or password

No session and no authorization code are created. Built-in and fallback users remain permissive because they have no configured password.

These passwords are not production credentials. They exist so tutorial flows and negative tests can exercise credential-shaped behavior without introducing a real account system.

## Use generic authorization claims

The top-level fields `groups`, `roles`, `tenant`, `preferred_username`, and `locale` expand into ordinary top-level ID token and userinfo claims. They are provider-neutral and are intended for common authorization fixtures.

Use the raw `claims` map for unusual shapes:

    users:
      - login: carol
        claims:
          feature_flags: [compact-inbox]
          app_role: maintainer

If the same claim name appears in a generic field and in `claims`, the explicit `claims` value wins. `omit_claims` deletes claims from ID token and userinfo after claims are assembled.

## Use device authorization for CLI or constrained-device tests

Device authorization lets a non-browser client ask the user to approve in a browser. Start the flow with:

    curl -sS -X POST http://localhost:5556/device_authorization \
      -d client_id=dev-client \
      -d 'scope=openid profile email offline_access' | jq .

Show the returned `verification_uri` and `user_code` to the user, or open `verification_uri_complete` directly. The approval page uses the same scenario registry and seeded-user fixture password behavior as browser login. For example, with `examples/users/personal-inbox-users.yaml`, approve as `alice` / `alice-password`.

The device polls the token endpoint with:

    curl -sS -X POST http://localhost:5556/token \
      -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
      -d client_id=dev-client \
      -d device_code="$DEVICE_CODE" | jq .

Before approval, the token endpoint returns `authorization_pending`. Too-fast polling returns `slow_down`. Approval returns bearer tokens; denial returns `access_denied`; expiry returns `expired_token`; reusing a consumed device code returns `invalid_grant`.

See `tinyidp help tutorial-device-authorization` for a complete walkthrough.

## Debug a running provider

The debug endpoints are loopback-only. They show enough state to diagnose a test without exposing full secrets:

    curl -s http://localhost:5556/debug | jq .
    curl -s http://localhost:5556/debug/sessions | jq .
    curl -s http://localhost:5556/debug/codes | jq .
    curl -s http://localhost:5556/debug/tokens | jq .
    curl -s http://localhost:5556/debug/device-grants | jq .

Use `/debug/reset` to clear in-memory state between tests:

    curl -s -X POST http://localhost:5556/debug/reset

Use `/debug/jwks-mode` to simulate JWKS failures:

    curl -s -X POST http://localhost:5556/debug/jwks-mode \
      -H 'Content-Type: application/json' -d '{"mode":"500"}'

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Discovery issuer does not match the client configuration. | A stale server is still listening, or the RP is configured with a different issuer URL. | Check `ss -ltnp`, kill stale tinyidp processes, and fetch discovery manually. |
| `redirect_uri not allowed for this client`. | The app callback URL is not in `redirect-uris`. | Add the exact callback URL, including scheme, host, port, and path. |
| Users file cannot be read. | The relative path was resolved against a different working directory. | Run from the tinyidp repo root or pass an absolute `--users-file`. |
| Password-backed user returns 401. | The form did not submit the configured fixture password. | Submit the exact fixture password or remove `password` from the seeded user. |
| PKCE client fails at authorize. | `public-spa` requires `code_challenge`. | Use a proper PKCE-capable RP or use `dev-client` for a first test. |
| Debug endpoint returns 403. | The request is not from loopback. | Call debug endpoints from the same host. |

## See also

- `tinyidp help getting-started` — first run and first login.
- `tinyidp help tutorial-first-rp-login` — walk through the full browser login flow.
- `tinyidp help tutorial-seeded-users-and-claims` — build deterministic users and claims.
- `tinyidp help tutorial-device-authorization` — walk through device-code approval and polling.
- `tinyidp help tutorial-xgoja-personal-inbox` — use tinyidp with the xgoja personal-inbox examples.
- `tinyidp help reference` — complete lookup reference.
