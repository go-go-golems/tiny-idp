# tinyidp

A minimal mock [OpenID Connect](https://openid.net/specs/openid-connect-core-1_0.html) Identity Provider for **local development and integration testing**. It exists to replace Keycloak-in-Docker when all you need is a working OIDC provider that issues RS256-signed ID tokens.

> ⚠️ **Not production-grade.** No real account system, consent screen, persistent keys, token revocation, or TLS enforcement. Refresh tokens and RP-initiated logout are implemented for testing semantics only and are in-memory. Bind to loopback (`127.0.0.1`) and never expose to the internet. See the design doc in `ttmp/` (ticket `MOCK-OIDC-IDP`).

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

### Profiles (switch setups with one flag)

A profile is a named bundle of overrides stored in `~/.config/tinyidp/profiles.yaml`:

```yaml
dev:
  oidc:
    client-id: dev-profile-client
    addr: 127.0.0.1:6600
ci:
  oidc:
    client-id: ci-runner
    redirect-uris:
      - http://localhost:9090/callback
```

```bash
go run ./cmd/tinyidp serve --profile dev
go run ./cmd/tinyidp serve --profile ci --profile-file /path/to/profiles.yaml
TINYIDP_PROFILE=dev go run ./cmd/tinyidp serve   # or via env
```

Profiles sit above defaults and below config/env/flags in precedence, so a local override always wins. The default file missing + `default` profile = silent skip (works out of the box). See `tinyidp help reference`.

### Introspect the resolved config

```bash
go run ./cmd/tinyidp print-config                          # print resolved config (yaml)
go run ./cmd/tinyidp print-config --profile dev             # what serve would use with --profile dev
go run ./cmd/tinyidp print-config --output json            # json instead of yaml
go run ./cmd/tinyidp serve --print-parsed-fields            # show resolved values + sources (incl. profiles)
go run ./cmd/tinyidp serve --print-schema                   # show the command's schema
```

`print-config` composes the same reusable `oidc` section as `serve`, so its output is exactly what `serve` would use.

```bash
go run ./cmd/tinyidp help                # browse topics
go run ./cmd/tinyidp help getting-started  # install + first login
go run ./cmd/tinyidp help tutorial      # guided scenario walkthrough
go run ./cmd/tinyidp help scenarios      # the scenario catalog
go run ./cmd/tinyidp help reference      # config, clients, endpoints
```

## Configure your app (RP)

Point your OIDC client at:

```
issuer:        http://localhost:5556
client_id:     dev-client
client_secret: (empty)
scopes:        openid profile email
```

## Clients

The provider ships with three built-in clients, so a single running instance can test public (SPA), confidential (web app), and permissive (quick-test) relying parties:

| Client ID | Type | PKCE | Secret | Default redirect URI |
|-----------|------|------|--------|---------------------|
| `dev-client` | public | optional | (none) | `http://localhost:3000/callback`, `http://127.0.0.1:3000/callback` |
| `public-spa` | public | **required** | (none) | `http://localhost:8080/callback` |
| `web-app` | confidential | optional | `dev-secret` | `http://localhost:8080/callback` |

### Configuring a client (merge behavior)

The OIDC section's `--client-id` / `--client-secret` / `--redirect-uris` register a single configured client. When the configured `--client-id` **matches a builtin**, the configuration is **merged** into the builtin rather than replacing it:

- `RequirePKCE`, `Secret` (when not overridden), and `AllowedScopes` are preserved from the builtin.
- The configured `--redirect-uris` are **added** (deduplicated) to the builtin's.
- A non-empty `--client-secret` overrides the builtin's.

So `--client-id public-spa --redirect-uris http://localhost:9090/cb` yields a `public-spa` client that still requires PKCE but now also accepts `http://localhost:9090/cb`. A configured `--client-id` that does not match a builtin registers a new permissive client.

## Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /.well-known/openid-configuration` | Discovery metadata. |
| `GET /jwks` | Public signing keys (JWKS). |
| `GET /authorize` | Authorization endpoint (login form → code). |
| `POST /token` | Token endpoint (`authorization_code` and `refresh_token`). |
| `GET /userinfo` | UserInfo (bearer access token → claims). |
| `GET /end-session` | RP-initiated logout. |
| `GET /healthz` | Liveness. |
| `GET/POST /debug/*` | Loopback-only introspection, reset, and JWKS failure-mode controls. |

## Status

- **Phase 0–4** — baseline OIDC happy path, multiple synthetic users, scenario registry, self-documenting login page, failure scenarios (done).
- **Glazed CLI** — reusable `oidc` field section, layered config (flags/env/config), **profiles** (`--profile` resolves `profiles.yaml`), **`print-config`** command (done).
- **Phase 5–11** — multiple clients, sessions, claims, debug UI, refresh tokens, JWKS rotation, and RP-initiated logout (done).
- **Phase 12** — Go test helper package (deferred; see `ttmp/.../reference/02-implementation-phases-and-tasks.md`).
