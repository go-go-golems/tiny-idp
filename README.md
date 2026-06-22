# tinyidp

A minimal mock [OpenID Connect](https://openid.net/specs/openid-connect-core-1_0.html) Identity Provider for **local development and integration testing**. It exists to replace Keycloak-in-Docker when all you need is a working OIDC provider that issues RS256-signed ID tokens.

> ⚠️ **Not production-grade.** No real login, consent, persistent keys, refresh tokens, revocation, logout, or TLS enforcement. Bind to loopback (`127.0.0.1`) and never expose to the internet. See the design doc in `ttmp/` (ticket `MOCK-OIDC-IDP`).

## Run

```bash
go run ./cmd/tinyidp serve
```

The CLI is built on the [Glazed](https://github.com/go-go-golems/glazed) command framework. Configuration is layered with predictable precedence (low → high):

1. **Section defaults**
2. **Config files** (`--config-file`)
3. **Environment variables** (`TINYIDP_*`)
4. **Positional arguments**
5. **CLI flags**

## Configuration

The OIDC provider config is a **reusable Glazed field section** (`internal/sections/oidc`). The same flags are available as CLI flags, env vars, and config-file keys:

| Flag | Env | Config key | Default | Meaning |
|------|-----|------------|---------|---------|
| `--issuer` | `TINYIDP_ISSUER` | `oidc.issuer` | `http://localhost:5556` | Issuer URL; endpoints derived from it. |
| `--addr` | `TINYIDP_ADDR` | `oidc.addr` | `127.0.0.1:5556` | Listen address (loopback by default). |
| `--client-id` | `TINYIDP_CLIENT_ID` | `oidc.client-id` | `dev-client` | Accepted client ID. |
| `--client-secret` | `TINYIDP_CLIENT_SECRET` | `oidc.client-secret` | (empty) | If set, `/token` enforces it; if empty, client is public. |
| `--redirect-uris` | `TINYIDP_REDIRECT_URIS` | `oidc.redirect-uris` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | Allowlist (repeatable flag / list in config). |

### Examples

Flags:

```bash
go run ./cmd/tinyidp serve \
  --issuer http://localhost:5556 \
  --client-id dev-client \
  --redirect-uris http://localhost:8080/callback
```

Env vars:

```bash
TINYIDP_CLIENT_ID=my-app \
TINYIDP_CLIENT_SECRET=dev-secret \
TINYIDP_REDIRECT_URIS=http://localhost:8080/callback \
go run ./cmd/tinyidp serve
```

Config file (`tinyidp.yaml`):

```yaml
oidc:
  client-id: my-app
  client-secret: dev-secret
  redirect-uris:
    - http://localhost:8080/callback
```

```bash
go run ./cmd/tinyidp serve --config-file tinyidp.yaml
```

### Introspect the resolved config

```bash
go run ./cmd/tinyidp serve --print-parsed-fields   # show resolved values + sources
go run ./cmd/tinyidp serve --print-schema          # show the command's schema
```

### Profiles (ready for future use)

`--profile` / `--profile-file` (and `TINYIDP_PROFILE` / `TINYIDP_PROFILE_FILE`) are wired. Loading a `profiles.yaml` is a future step; see `tinyidp help profiles`.

```bash
go run ./cmd/tinyidp help              # browse topics
go run ./cmd/tinyidp help oidc-config  # the OIDC section explained
```

## Configure your app (RP)

Point your OIDC client at:

```
issuer:        http://localhost:5556
client_id:     dev-client
client_secret: (empty)
scopes:        openid profile email
```

## Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /.well-known/openid-configuration` | Discovery metadata. |
| `GET /jwks` | Public signing keys (JWKS). |
| `GET /authorize` | Authorization endpoint (login form → code). |
| `POST /token` | Token endpoint (code → ID + access token). |
| `GET /userinfo` | UserInfo (bearer access token → claims). |
| `GET /healthz` | Liveness. |

## Status

- **Phase 0–4** — baseline OIDC happy path, multiple synthetic users, scenario registry, self-documenting login page, failure scenarios (done).
- **Glazed CLI** — reusable `oidc` field section, layered config (flags/env/config), profile-ready (done).
- **Phase 5–12** — multiple clients, sessions, claims, debug UI, refresh tokens, JWKS rotation, logout, Go test helper (deferred; see `ttmp/.../reference/02-implementation-phases-and-tasks.md`).
