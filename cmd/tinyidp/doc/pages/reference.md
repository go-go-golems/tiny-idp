---
Title: "Reference"
Slug: reference
Short: "Configuration, clients, endpoints, and behaviors — the lookup reference for everything tinyidp does."
Topics:
- oidc
- config
- reference
- clients
Commands:
- serve
- print-config
Flags:
- issuer
- addr
- client-id
- client-secret
- redirect-uris
- profile
- profile-file
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This is the lookup reference for tinyidp: how it is configured, which
clients it ships, which endpoints it exposes, and the behaviors those
endpoints implement. It is organized for finding a specific fact, not for
reading top to bottom. For a guided introduction, start with
`tinyidp help getting-started`; for the scenario catalog, see
`tinyidp help scenarios`.

## Configuration

tinyidp is configured through the Glazed command framework. A reusable
`oidc` field section defines the provider settings; it is composed into
the `serve` and `print-config` commands so flags, environment variables,
and config-file schema never drift between them.

### Fields

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--issuer` | `TINYIDP_ISSUER` | `http://localhost:5556` | Issuer URL; endpoints are derived from it. Path-based issuers such as `http://localhost:5556/realms/demo` are supported. |
| `--addr` | `TINYIDP_ADDR` | `127.0.0.1:5556` | Listen address (loopback by default; set `0.0.0.0:5556` for LAN). |
| `--client-id` | `TINYIDP_CLIENT_ID` | `dev-client` | Client ID. If it matches a builtin, the config is merged into it. |
| `--client-secret` | `TINYIDP_CLIENT_SECRET` | (empty) | If set, `/token` enforces it; if empty, the client is public. |
| `--redirect-uris` | `TINYIDP_REDIRECT_URIS` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | Allowlist of redirect URIs (repeat the flag or pass a list). |
| `--users-file` | `TINYIDP_USERS_FILE` | (empty) | Optional YAML/JSON file with seeded users and claims. |

### Precedence

From lowest to highest:

1. Section defaults
2. Profiles
3. Config files (`--config-file`)
4. Environment variables (`TINYIDP_*`)
5. Positional arguments
6. CLI flags

A local override always wins. Use `tinyidp serve --print-parsed-fields`
to see which source won each field, or `tinyidp print-config` to print
the resolved configuration without starting the server.

### Config files

A YAML config file layers above profiles and below env/flags. The fields
live under the `oidc` section slug:

    oidc:
      issuer: http://localhost:5556
      addr: 127.0.0.1:5556
      client-id: dev-client
      client-secret: dev-secret
      redirect-uris:
        - http://localhost:8080/callback
      users-file: ./users.yaml

Pass it with `tinyidp serve --config-file config.yaml`.

### Profiles

Profiles are named bundles of overrides stored in a YAML file, selected
with `--profile`. They sit above defaults and below config/env/flags, so
they are a convenient baseline that local overrides always win against.

The file is a YAML map: profile name, then section slug, then fields.

    dev:
      oidc:
        client-id: dev-profile-client
        addr: 127.0.0.1:6600
    ci:
      oidc:
        client-id: ci-runner
        addr: 127.0.0.1:6601

tinyidp looks for a default file at `~/.config/tinyidp/profiles.yaml`
(`$XDG_CONFIG_HOME/tinyidp/profiles.yaml` on Linux). If the default file
is missing and the requested profile is `default`, loading is skipped
silently — tinyidp works out of the box with no `profiles.yaml`. A
non-default profile with no file is an error, never a silent fallback.

    tinyidp serve --profile dev
    TINYIDP_PROFILE=ci tinyidp serve

## Seeded users

By default, tinyidp derives a stable synthetic user from whatever login is typed. For integration tests that need fixed subjects or app-specific claims, pass `--users-file` (or `oidc.users-file` in config). The file may be YAML or JSON:

    users:
      - login: alice
        sub: user-alice-fixed
        email: alice@example.test
        name: Alice Inbox
        password: alice-password
        email_verified: true
        groups: [inbox-users]
        roles: [writer]
        tenant: personal
        preferred_username: alice
        locale: en-US
      - login: bob
        sub: user-bob-fixed
        email: bob@example.test
        name: Bob Inbox
        password: bob-password
        email-verified: true
        groups: [inbox-users]
        roles: [reader]
        tenant: personal
        claims:
          feature_flags: [compact-inbox]

Seeded users are registered as normal scenarios. They override builtins with the same login and appear on the login page under "Seeded users" by default.

`password` is optional. If it is omitted or empty, the seeded user remains permissive and any submitted password is accepted. If it is set, authorize POST must submit the exact fixture password; wrong or missing passwords return `401 invalid login or password` and no session or authorization code is created. Passwords are plain local test fixture values, not production credentials.

Generic top-level claim helpers are available for common authorization fixtures: `groups`, `roles`, `tenant`, `preferred_username`, and `locale`. These expand into ordinary top-level claims in both the ID token and userinfo response. The raw `claims` map remains available for provider-specific or unusual shapes; explicit `claims` entries override generic helper fields with the same claim name. Use `omit_claims` when a seeded user should deliberately omit a base claim such as `email`.

## Clients

tinyidp ships a client registry with three builtins, so one running
instance can serve a public SPA, a confidential web app, and a
permissive dev client simultaneously.

| Client | Type | PKCE | Secret | Post-logout redirect |
|--------|------|------|--------|---------------------|
| `dev-client` | public | optional | (none) | `http://localhost:3000` |
| `public-spa` | public | **required** | (none) | `http://localhost:8080` |
| `web-app` | confidential | optional | `dev-secret` | `http://localhost:8080` |

Each client owns its own redirect-URI allowlist, PKCE requirement, scope
allowlist, and post-logout redirect allowlist. A redirect URI is valid
for a specific client, not globally, and an authorization code issued to
one client cannot be redeemed by another.

When the configured `--client-id` matches a builtin, the configuration is
**merged** into the builtin rather than replacing it: the builtin's
class-defining properties (`RequirePKCE`, `Secret`, `AllowedScopes`) are
preserved, and redirect URIs are unioned. A non-matching `--client-id`
registers a new permissive client.

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/.well-known/openid-configuration` | GET | Discovery; advertises every endpoint and supported grant/scope/claim. |
| `/jwks` | GET | Public signing keys (three kids). See JWKS rotation below. |
| `/authorize` | GET/POST | Authorization endpoint; GET decides silent vs interactive login, POST submits credentials. |
| `/token` | POST | Token endpoint; `authorization_code` and `refresh_token` grants. |
| `/userinfo` | GET | UserInfo; bearer-token-protected claims. |
| `/end-session` | GET | RP-initiated logout. See Logout below. |
| `/healthz` | GET | Liveness (`ok`). |
| `/debug/*` | GET/POST | Loopback-only introspection. See Debug UI below. |

If `--issuer` includes a path component, tinyidp registers the same endpoints under that prefix as well as at the root. For example, issuer `http://localhost:5556/realms/demo` serves discovery at `/realms/demo/.well-known/openid-configuration` and advertises endpoint URLs under `/realms/demo`. This is useful when replacing Keycloak realm URLs in tests while keeping the root issuer workflow available for simple local runs.

## Behaviors

### Sessions

After a login, tinyidp sets an IdP session cookie and remembers the
authentication time. The authorize GET endpoint implements the OIDC
session rules:

- `prompt=none` forbids any UI. If a valid, fresh-enough session exists,
  a code is issued silently; otherwise the RP receives `login_required`.
- `prompt=login` forces the login form even when a session exists.
- `max_age` requires the session's `auth_time` to be within the given
  seconds; otherwise re-authentication is forced.
- `login_hint` prefills the login form.

The `auth_time` claim in the ID token is the original login time, not
the token-issuance time. This is what makes `max_age` checks honest: a
silent re-issue minutes after login still reports the true age of the
session. The cookie is `HttpOnly` and `SameSite=Lax` but not `Secure`,
because tinyidp serves plain HTTP on loopback and a `Secure` cookie
would never be sent.

### Refresh tokens

When the RP requests the `offline_access` scope, the token response
includes a `refresh_token`. Refresh tokens rotate: each use deletes the
presented token and issues a new one. Reusing a rotated token fails
with `invalid_grant` — the standard reuse signal. A refresh token cannot
be redeemed by a different client than the one it was issued to.

### JWKS rotation

`/jwks` publishes three keys: `dev-key-1` (the active signing key),
`rotated-key-2`, and `bad-sig-key`. The `SignKey` scenario field selects
which key signs the ID token (see `tinyidp help scenarios`). Because
`/jwks` is global rather than tied to a login, its failure modes are
server-level, toggled through the debug UI:

    curl -X POST http://localhost:5556/debug/jwks-mode \
      -H 'Content-Type: application/json' -d '{"mode":"500"}'

Valid modes are `normal` (default), `500`, `slow` (sleeps 10s), and
`empty` (returns `{"keys":[]}`). The debug reset restores `normal`.

### Logout

`/end-session` implements RP-initiated logout. Parameters:

- `id_token_hint` — a previously issued ID token. Its payload is decoded
  (signature not re-verified) to find the subject, and any session for
  that subject is deleted — even from a client that holds no cookie.
- `post_logout_redirect_uri` — a URI to redirect to after logout. It must
  be registered for the client; when `client_id` is given, it is checked
  against that client only, otherwise against all clients.
- `state` — forwarded on the redirect as a query parameter.

The session cookie is always cleared. Without a `post_logout_redirect_uri`,
tinyidp returns a logged-out page.

### Debug UI

The `/debug/*` endpoints are read-only views of in-memory state plus a
reset, guarded to loopback so a LAN bind does not leak state.

| Endpoint | Method | Returns |
|----------|--------|---------|
| `/debug` | GET | Counts and the current JWKS mode. |
| `/debug/sessions` | GET | Active sessions (login, sub, auth_time, expires). |
| `/debug/codes` | GET | Outstanding authorization codes. |
| `/debug/tokens` | GET | Issued access tokens. |
| `/debug/jwks-mode` | GET/POST | Read or set the JWKS failure mode. |
| `/debug/reset` | POST | Clear all sessions, codes, tokens, and refresh tokens. |

Secrets are shown as 8-character prefixes — enough to correlate a flow
against a log without exposing the full token in a listing.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `prompt=none` always returns `login_required`. | The session cookie is not being sent. | tinyidp serves plain HTTP; ensure the cookie is not marked `Secure` by a proxy, and that the RP and IdP share a context. |
| Config flag has no effect. | A higher-precedence source overrode it. | Run `tinyidp serve --print-parsed-fields` to see which source won. |
| `--redirect-uris` replaced the builtin's URIs. | It does not — it unions them. | The merge preserves builtin properties; the configured URI is added, not swapped. |
| Debug endpoints return 403. | The request is not from loopback. | Call them from the same host; they are loopback-only by design. |

## See also

- `tinyidp help getting-started` — install and first login.
- `tinyidp help tutorial` — a guided walkthrough of scenarios.
- `tinyidp help scenarios` — the full scenario catalog and model.
