# tinyidp

A minimal mock [OpenID Connect](https://openid.net/specs/openid-connect-core-1_0.html) Identity Provider for **local development and integration testing**. It exists to replace Keycloak-in-Docker when all you need is a working OIDC provider that issues RS256-signed ID tokens.

> ⚠️ **Not production-grade.** No real login, consent, persistent keys, refresh tokens, revocation, logout, or TLS enforcement. Bind to loopback (`127.0.0.1`) and never expose to the internet. See the design doc in `ttmp/` (ticket `MOCK-OIDC-IDP`).

## Run

```bash
go run ./cmd/tinyidp
```

Defaults:

| Variable | Default | Meaning |
|----------|---------|---------|
| `OIDC_ISSUER` | `http://localhost:5556` | Issuer URL; endpoints are derived from it. |
| `OIDC_ADDR` | `127.0.0.1:5556` | Listen address (loopback by default). |
| `OIDC_CLIENT_ID` | `dev-client` | Single client ID. |
| `OIDC_CLIENT_SECRET` | (empty) | If set, token endpoint enforces it. |
| `OIDC_REDIRECT_URIS` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | CSV allowlist. |
| `OIDC_USER_SUB` | `user-123` | Fixed user subject (Phase 0 only). |
| `OIDC_USER_EMAIL` | `dev@example.test` | Fixed user email (Phase 0 only). |
| `OIDC_USER_NAME` | `Dev User` | Fixed user name (Phase 0 only). |

## Configure your app (RP)

Point your OIDC client at:

```
issuer:        http://localhost:5556
client_id:     dev-client
client_secret: (empty)
scopes:        openid profile email
```

For a different callback URL:

```bash
OIDC_REDIRECT_URIS=http://localhost:8080/auth/callback go run ./cmd/tinyidp
```

For a confidential-client-style test:

```bash
OIDC_CLIENT_ID=my-app \
OIDC_CLIENT_SECRET=dev-secret \
OIDC_REDIRECT_URIS=http://localhost:8080/callback \
go run ./cmd/tinyidp
```

## Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /.well-known/openid-configuration` | Discovery metadata. |
| `GET /jwks` | Public signing keys (JWKS). |
| `GET /authorize` | Authorization endpoint (issues a code). |
| `POST /token` | Token endpoint (code → ID + access token). |
| `GET /userinfo` | UserInfo (bearer access token → claims). |
| `GET /healthz` | Liveness. |

## Status

- **Phase 0** — baseline OIDC happy path (done).
- **Phase 1–4** — multiple synthetic users, scenario registry, self-documenting login page, failure scenarios (see `ttmp/.../reference/02-implementation-phases-and-tasks.md`).
