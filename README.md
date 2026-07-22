# tinyidp

A compact, self-contained [OpenID Connect](https://openid.net/specs/openid-connect-core-1_0.html) Identity Provider written in Go. It started life as a mock IdP to replace Keycloak-in-Docker for local development, and it still does that job well — but it has since grown a **strict, Fosite-backed engine**, **durable SQLite storage**, **Argon2id password credentials**, **persistent signing keys with rotation**, an **operational `admin` CLI**, and an **embeddable Go package** (`pkg/embeddedidp`) so you can run a real OIDC provider inside your own binary.

There are two engines, and picking the right one matters:

| Engine | Flag | Store | Intended use | Notes |
|--------|------|-------|--------------|-------|
| **mock** | `--engine mock` (default) | in-memory | Local dev, integration tests, **failure simulation** | Rich scenario catalog, debug routes, device grant, DPoP, JWKS failure modes. Not for production. |
| **strict** | `--engine fosite` | in-memory (via `serve-dev`) or a persistent `idpstore.Store` (via `pkg/embeddedidp` or `serve-production`) | Production-like OAuth/OIDC behavior | Fosite validation, Auth-Code + PKCE, CSRF, security headers, persistent consent/keys, Argon2id login. |

> **Maturity, read this.** The `mock` engine and **`tinyidp serve-dev` are for local/testing use** — bind to loopback (`127.0.0.1`) and never expose them to the internet. `tinyidp serve-production` is the durable strict host: it requires an HTTPS issuer, SQLite database, owner-only token-secret file, audit sink, a reviewed non-secret `--signup-program-file`, and a pre-provisioned active signing key. It supports either direct TLS or explicit trusted-proxy HTTP; the latter preserves the canonical HTTPS origin and Secure cookies only for configured proxy peers. A custom production deployment may instead embed the strict engine through `pkg/embeddedidp` with those same requirements; `embeddedidp.Options.Validate(ctx)` enforces them. `serve-dev --engine fosite` remains an in-memory development preview, not a production server. See [`docs/security-profile.md`](docs/security-profile.md) and [`examples/production-host/README.md`](examples/production-host/README.md).
>
> **Honest caveats.** The strict engine has passed a **hosted OpenID Foundation Basic OP conformance run with zero hard failures** (suite 5.2.0; discovery + static clients) — this is *not* a claim of formal certification. Strict RFC 8628 device authorization is implemented with durable grants, browser verification, transactional token issuance, and discovery metadata. It remains subject to the release evidence and independent-review gate in `TINYIDP-DEVICE-PROD-001`; implementation availability is not a blanket production approval. Still missing/in progress: token `/revoke`, DPoP, and additional production-native capability bindings for more elaborate signup programs.

---

## Quick start (mock engine, local dev)

```bash
go run ./cmd/tinyidp serve-dev
```

Point your OIDC client at:

```
issuer:        http://localhost:5556
client_id:     dev-client
client_secret: (empty)
scopes:        openid profile email
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
| `--engine` | `TINYIDP_ENGINE` | `oidc.engine` | `mock` | Provider engine: `mock` (local failure simulation) or `fosite` (strict production-like behavior). |
| `--issuer` | `TINYIDP_ISSUER` | `oidc.issuer` | `http://localhost:5556` | Issuer URL; endpoints derived from it. Path-based issuers such as `http://localhost:5556/realms/demo` are supported. |
| `--addr` | `TINYIDP_ADDR` | `oidc.addr` | `127.0.0.1:5556` | Listen address (loopback by default). |
| `--client-id` | `TINYIDP_CLIENT_ID` | `oidc.client-id` | `dev-client` | Accepted client ID. |
| `--client-secret` | `TINYIDP_CLIENT_SECRET` | `oidc.client-secret` | (empty) | If set, `/token` enforces it; if empty, client is public. |
| `--redirect-uris` | `TINYIDP_REDIRECT_URIS` | `oidc.redirect-uris` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | Allowlist (repeatable flag / list in config). |
| `--extra-clients` | `TINYIDP_EXTRA_CLIENTS` | `oidc.extra-clients` | (empty) | Extra clients, one per entry, pipe-separated `id\|secret\|redirect[\|redirect...]`. |
| `--users-file` | `TINYIDP_USERS_FILE` | `oidc.users-file` | (empty) | Optional YAML/JSON file with seeded users and claims (mock engine). |

### Examples

Flags:

```bash
go run ./cmd/tinyidp serve-dev \
  --issuer http://localhost:5556 \
  --client-id dev-client \
  --redirect-uris http://localhost:8080/callback
```

Env vars:

```bash
TINYIDP_CLIENT_ID=my-app \
TINYIDP_CLIENT_SECRET=dev-secret \
TINYIDP_REDIRECT_URIS=http://localhost:8080/callback \
go run ./cmd/tinyidp serve-dev
```

Config file (`tinyidp.yaml`):

```yaml
oidc:
  client-id: my-app
  client-secret: dev-secret
  redirect-uris:
    - http://localhost:8080/callback
  users-file: ./users.yaml
```

```bash
go run ./cmd/tinyidp serve-dev --config-file tinyidp.yaml
```

Checked-in portable examples live under `examples/configs/`:

| File | Use |
|------|-----|
| `examples/configs/dev-root.yaml` | Basic root-issuer dev setup on `localhost:5556`. |
| `examples/configs/personal-inbox-root.yaml` | xgoja personal-inbox smoke setup with a root issuer. |
| `examples/configs/personal-inbox-realm.yaml` | xgoja personal-inbox smoke setup with a path-based issuer URL. |
| `examples/configs/public-spa-pkce.yaml` | Builtin public SPA client that preserves PKCE-required behavior. |
| `examples/configs/confidential-web-app.yaml` | Builtin confidential web-app client with local `dev-secret`. |

`oidc.users-file` is currently resolved relative to the process working directory. For portable examples, run tinyidp from the repository root or use an absolute users-file path.

### Engines: mock vs strict (fosite)

- **mock** (default) is the local-testing engine. It backs everything in-memory, ships a **scenario registry** (synthetic users, malformed-token and JWKS failure modes), exposes loopback-only **`/debug/*`** routes, and implements the device grant and DPoP for integration tests. Choose it for reproducing OIDC edge cases cheaply.
- **fosite** (strict) runs the production-shaped code path: [ory/fosite](https://github.com/ory/fosite) for OAuth/OIDC validation and response writing, **Authorization Code + PKCE (`S256`) and RFC 8628 device authorization**, exact redirect-URI allow-listing, server-side browser sessions with hashed opaque cookies, CSRF on login/consent POSTs, security headers, `Cache-Control: no-store`, persistent consent, and **Argon2id password login** via a `PasswordAuthenticator`. Select it with `--engine fosite`. Features explicitly **not** available in strict mode today: debug routes, scenario failure injection, implicit/hybrid flows, production DPoP, and dynamic client registration ([`docs/security-profile.md`](docs/security-profile.md)).

### Seeded users (mock engine)

By default, any login derives a stable synthetic user. For tests that need fixed subjects or app-specific claim shapes, pass a users file:

```yaml
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
    # Raw claims remain available for provider-specific or unusual shapes.
    claims:
      feature_flags: [compact-inbox]
```

```bash
go run ./cmd/tinyidp serve-dev --users-file ./users.yaml
```

A ready-to-copy personal-inbox fixture is available at `examples/users/personal-inbox-users.yaml`.

Seeded users override builtins with the same login, so you can keep using `alice` and `bob` while making their `sub`, `email`, `name`, and claims deterministic for a test suite.

`password` is optional. When it is omitted or empty, that seeded user keeps the default local-test behavior and any submitted password is accepted. When `password` is set, the authorize form must submit the exact fixture password or tinyidp returns `401 invalid login or password` without creating a session or authorization code. These are plain local test fixtures, not production credentials. For **durable** users with real hashed credentials (used by the strict engine and production embedding), see [Users and passwords](#users-and-passwords-durable) below.

Common authorization claims can be written as top-level generic fields: `groups`, `roles`, `tenant`, `preferred_username`, and `locale`. These expand into ordinary top-level ID token and userinfo claims. The raw `claims` map is still the escape hatch for provider-specific or unusual claim shapes; explicit `claims` values override the generic fields when the same claim name appears in both places.

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
go run ./cmd/tinyidp serve-dev --profile dev
go run ./cmd/tinyidp serve-dev --profile ci --profile-file /path/to/profiles.yaml
TINYIDP_PROFILE=dev go run ./cmd/tinyidp serve-dev   # or via env
```

Profiles sit above defaults and below config/env/flags in precedence, so a local override always wins. The default file missing + `default` profile = silent skip (works out of the box). See `tinyidp help reference`.

### Introspect the resolved config

```bash
go run ./cmd/tinyidp print-config                          # print resolved config (yaml)
go run ./cmd/tinyidp print-config --profile dev             # what serve would use with --profile dev
go run ./cmd/tinyidp print-config --output json            # json instead of yaml
go run ./cmd/tinyidp serve-dev --print-parsed-fields            # show resolved values + sources (incl. profiles)
go run ./cmd/tinyidp serve-dev --print-schema                   # show the command's schema
```

`print-config` composes the same reusable `oidc` section as `serve-dev`, so its output is exactly what `serve-dev` would use.

```bash
go run ./cmd/tinyidp help                # browse topics
go run ./cmd/tinyidp help getting-started                  # install + first login
go run ./cmd/tinyidp help user-guide                       # operational guide
go run ./cmd/tinyidp help developer-guide                  # implementation guide
go run ./cmd/tinyidp help tutorial-first-rp-login          # first RP login
go run ./cmd/tinyidp help tutorial-seeded-users-and-claims # users, passwords, claims
go run ./cmd/tinyidp help tutorial-device-authorization    # OAuth device-code login
go run ./cmd/tinyidp help tutorial-dpop                    # DPoP sender-constrained tokens
go run ./cmd/tinyidp help tutorial-xgoja-personal-inbox    # xgoja Steps 06/07/08
go run ./cmd/tinyidp help tutorial                         # guided scenario walkthrough
go run ./cmd/tinyidp help scenarios                        # the scenario catalog
go run ./cmd/tinyidp help reference                        # config, clients, endpoints
```

---

## Production path (strict engine)

The strict engine is designed to be embedded in your own service with a durable store. The moving parts are: a persistent **store**, the **admin CLI** to provision it, **Argon2id** user credentials, **persistent signing keys** with a safe rotation invariant, and the **`pkg/embeddedidp`** package that wires them into an `http.Handler`.

### Storage and persistence

Strict-engine domain state flows through `idpstore.Store`. `pkg/sqlitestore` is the durable single-active-node implementation; its checksummed migrations own all domain and Fosite schema. **Production mode requires a persistent store** and has no in-memory override. Named store operations atomically protect user/credential creation, password/security-state replacement, login counters, authorization-code use, refresh rotation/reuse revocation, and signing-key rotation. SQLite defaults to WAL, `synchronous=FULL`, a five-second busy timeout, and exactly one connection. See [`docs/storage.md`](docs/storage.md) for the deployment, filesystem, transaction, backup, and restore contract.

### Admin CLI

`tinyidp admin` is the operational surface for SQLite-backed deployments. All commands take an explicit `--db` path:

```
tinyidp admin --db ./tinyidp.db
├── init [--generate-signing-key --kid <kid>]   # create DB, apply migrations, optional first key
├── migrate [--dry-run]                          # apply embedded migrations
├── doctor                                        # validate clients + signing keys (production rules)
├── client  create | list | get | disable | enable | rotate-secret
├── keys    generate | rotate | list | retire
├── user    create | set-password | get | disable | enable
├── backup  create --out <file> | verify --path <file> | restore --path <file>
└── export  diagnostics                           # sanitized (no secret hashes, no private PEM)
```

```bash
# Bootstrap a database and an initial signing key.
tinyidp admin --db ./tinyidp.db init --generate-signing-key --kid initial-rsa-1

# Register a confidential client (generated secret printed once).
tinyidp admin --db ./tinyidp.db client create \
  --id web-app --generate-secret \
  --redirect-uri https://app.example.test/callback \
  --scope openid --scope profile --scope email --scope offline_access \
  --require-pkce
```

Client/keys/diagnostics output redacts secret hashes and never prints private key PEM. See [`docs/admin-cli.md`](docs/admin-cli.md).

### Users and passwords (durable)

Strict login uses durable public records split across three types: `idpstore.User` (subject/profile/account state), `idpstore.PasswordCredential` (the **encoded Argon2id hash** and lifecycle flags — never stored on `User`), and `idpstore.AccountSecurityState` (failed-login counters, lockout, last-successful-login). The strict adapter authenticates `POST /authorize` through an `idp.PasswordAuthenticator`, returns the generic `invalid login or password` on failure, and emits stable audit reason codes (`invalid_credentials`, `account_disabled`, `account_locked`). Provision credentials with the admin CLI, preferring stdin so secrets stay out of shell history:

```bash
printf '%s\n' 'alice-password' | \
  tinyidp admin --db ./tinyidp.db user create \
    --login alice --email alice@example.test --email-verified \
    --name 'Alice Example' --password-from-stdin
```

See [`docs/users-and-passwords.md`](docs/users-and-passwords.md).

### Signing keys and rotation

The strict engine signs ID tokens with the active key in the store's `KeyStore` and publishes `VerificationKeys` at `/jwks`. `internal/keys.RotateRSA` implements a safe three-state rotation: generate a new key → make it active for new tokens → retire the previous key but keep it in JWKS so relying parties can validate old ID tokens until they expire. Keep retired keys published for at least the maximum ID-token lifetime plus clock skew. See [`docs/key-rotation.md`](docs/key-rotation.md).

### Embedding the provider (`pkg/embeddedidp`)

Run the strict IdP inside your own binary. `embeddedidp.New(ctx, Options)` returns a `*Provider` whose `Handler()` you mount on any `*http.ServeMux`:

The supported composition also includes `pkg/idpaccounts` for account/password lifecycle, `embeddedidp.Bootstrap` for browser or device-shaped clients and the initial signing key, and `embeddedidp.NewInProcessIssuerTransport` for bounded same-process discovery and token exchange. See [`docs/embedding-foundations.md`](docs/embedding-foundations.md) for the complete construction order, failure semantics, executable examples, and the strict-mode device authorization contract.

```go
import (
    "context"
    "net/http"

    "github.com/go-go-golems/tiny-idp/pkg/embeddedidp"
    "github.com/go-go-golems/tiny-idp/pkg/sqlitestore"
)

ctx := context.Background()
store, err := sqlitestore.Open(ctx, sqlitestore.DefaultConfig("tinyidp.db"))
if err != nil { /* handle */ }
defer store.Close()

provider, err := embeddedidp.New(ctx, embeddedidp.Options{
    Issuer: "https://id.example.com",
    Mode:   embeddedidp.ProductionMode,        // enforces the invariants below
    Store:  store,                              // persistent idpstore.Store (SQLite)
    Cookie: embeddedidp.CookieConfig{Secure: true},
    Token:  embeddedidp.TokenConfig{SecretKey: secret /* >= 32 bytes */},
    Audit: auditSink,                           // required in production
    RateLimiter: limiter,                       // required in production
    ClientAddress: resolver,                    // required in production
    Authenticator: authenticator,
    PasswordPolicy: idp.DefaultPasswordAcceptancePolicy(),
    PasswordWork: idp.PasswordWorkConfig{MaxConcurrent: 2},
})
if err != nil { /* Validate() failed */ }
defer provider.Close(context.Background())

mux := http.NewServeMux()
mux.Handle("/", provider.Handler())
// The handler speaks plain HTTP; terminate TLS at a reverse proxy in front.
http.ListenAndServe("127.0.0.1:5556", mux)
```

`Options.Validate(ctx)` enforces the production contract: a valid HTTPS issuer, valid clients, a ≥32-byte token secret, secure cookies, explicit audit/limiter/client-address implementations, a persistent current-schema store, and exactly one currently usable RS256 RSA key of at least 2048 bits. `Readiness(ctx)` reports lifecycle/store/key checks and `Close(ctx)` is idempotent. A runnable dev example is in [`examples/embedded/main.go`](examples/embedded/main.go). See [`docs/security-profile.md`](docs/security-profile.md) for the full enabled-controls list and the release gate.

---

## DPoP sender-constrained tokens (mock engine)

The mock engine supports DPoP-bound access tokens for local and integration tests. If a `/token` request includes a valid `DPoP` proof JWT, tinyidp stores the proof key thumbprint with the issued opaque access token and returns `token_type: DPoP`. Calling `/userinfo` with that token then requires `Authorization: DPoP <token>` plus a fresh proof signed by the same key and containing the correct `ath` hash of the access token.

Bearer behavior remains unchanged when no `DPoP` header is present. Refresh tokens issued from a DPoP-bound flow are bound to the same key and require matching DPoP proofs during rotation. See `tinyidp help tutorial-dpop`. (DPoP is not part of the strict production profile.)

## Device authorization grant

Both engines implement the OAuth 2.0 Device Authorization Grant. The mock engine remains useful for local failure simulation; the strict provider persists keyed hashes of device/user codes, binds the browser verification interaction with CSRF and fresh password authentication, and consumes an approved grant in the same SQLite transaction that persists normal Fosite tokens. A device starts with `POST /device_authorization`, shows the returned `user_code` and `verification_uri` to the user, and polls `/token` with `grant_type=urn:ietf:params:oauth:grant-type:device_code` until the browser approval form at `/device` approves or denies the request.

Quick start:

```bash
DEVICE_JSON=$(curl -sS -X POST http://localhost:5556/device_authorization \
  -d client_id=dev-client \
  -d 'scope=openid profile email offline_access')

echo "$DEVICE_JSON" | jq .
# Open verification_uri_complete, approve as alice/alice-password when using examples/users/personal-inbox-users.yaml.

curl -sS -X POST http://localhost:5556/token \
  -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
  -d client_id=dev-client \
  -d device_code="$(echo "$DEVICE_JSON" | jq -r .device_code)" | jq .
```

Polling before approval returns `authorization_pending`; polling too quickly returns `slow_down`; denied, expired, mismatched, unknown, or already-used device codes return the corresponding OAuth error. Strict discovery advertises `device_authorization_endpoint` and the device grant type only because this full path is implemented. See `tinyidp help tutorial-device-authorization` and the device ticket's operator runbook before enabling a client in production.

## Clients (mock engine builtins)

The mock engine ships three built-in clients, so a single running instance can test public (SPA), confidential (web app), and permissive (quick-test) relying parties:

| Client ID | Type | PKCE | Secret | Default redirect URI |
|-----------|------|------|--------|---------------------|
| `dev-client` | public | optional | (none) | `http://localhost:3000/callback`, `http://127.0.0.1:3000/callback` |
| `public-spa` | public | **required** | (none) | `http://localhost:8080/callback` |
| `web-app` | confidential | optional | `dev-secret` | `http://localhost:8080/callback` |

(In the strict/production path, clients are provisioned in the store via `tinyidp admin client create`.)

### Configuring a client (merge behavior)

The OIDC section's `--client-id` / `--client-secret` / `--redirect-uris` register a single configured client. When the configured `--client-id` **matches a builtin**, the configuration is **merged** into the builtin rather than replacing it:

- `RequirePKCE`, `Secret` (when not overridden), and `AllowedScopes` are preserved from the builtin.
- The configured `--redirect-uris` are **added** (deduplicated) to the builtin's.
- A non-empty `--client-secret` overrides the builtin's.

So `--client-id public-spa --redirect-uris http://localhost:9090/cb` yields a `public-spa` client that still requires PKCE but now also accepts `http://localhost:9090/cb`. A configured `--client-id` that does not match a builtin registers a new permissive client.

## Endpoints

The two engines expose **different route sets**. Shared by both:

| Endpoint | Purpose |
|----------|---------|
| `GET /.well-known/openid-configuration` | Discovery metadata. |
| `GET /jwks` | Public signing keys (JWKS). |
| `GET /authorize` | Authorization endpoint (login form → code). |
| `POST /token` | Token endpoint (`authorization_code`, `refresh_token`; device-code in mock). |
| `GET /userinfo` | UserInfo (bearer access token → claims). |
| `GET /end-session` | RP-initiated current-browser logout with exact registered post-logout redirect validation. |
| `GET /healthz` | Liveness. |

Mock engine only:

| Endpoint | Purpose |
|----------|---------|
| `POST /device_authorization` | OAuth device-code start endpoint. |
| `GET/POST /device` | Browser approval/denial form for device-code requests. |
| `GET/POST /debug/*` | Loopback-only introspection, reset, and JWKS failure-mode controls. |

Strict (fosite) engine only:

| Endpoint | Purpose |
|----------|---------|
| `GET /readyz` | Readiness (store/migrations/active-key checks). |

The strict engine deliberately does **not** serve `/debug/*`; it does serve `/device_authorization` and `/device` as the RFC 8628 flow. Strict `/introspect` is available to configured confidential resource servers. There is no token `/revoke` HTTP route yet (see [`docs/security-profile.md`](docs/security-profile.md)). Strict `/end-session` revokes the session represented by the current browser cookie; it does not yet accept `id_token_hint` for subject-wide revocation or perform front-channel/back-channel logout at other relying parties.

When `--issuer` contains a path, tinyidp also serves the same routes under that path. For example, `--issuer http://localhost:5556/realms/personal-inbox` serves discovery at `/realms/personal-inbox/.well-known/openid-configuration` and advertises `/realms/personal-inbox/authorize`, `/token`, `/userinfo`, and `/jwks` endpoint URLs. Root routes remain available for simple local testing. Path-based issuers are URL-shape compatibility only; seeded-user claims stay provider-neutral.

## xgoja personal-inbox smoke ergonomics

From the xgoja Step 06 example directory, you can continue using the existing Makefile variables while pointing them at tinyidp:

```bash
TINYIDP_ROOT=/path/to/2026-06-22--mock-oidc-idp \
TINYIDP_ISSUER=http://127.0.0.1:19087 \
TINYIDP_USERS_FILE=/path/to/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml \
make tinyidp-smoke
```

For a path-based issuer:

```bash
TINYIDP_ROOT=/path/to/2026-06-22--mock-oidc-idp \
TINYIDP_ISSUER=http://127.0.0.1:19087/realms/personal-inbox \
TINYIDP_USERS_FILE=/path/to/2026-06-22--mock-oidc-idp/examples/users/personal-inbox-users.yaml \
make tinyidp-smoke
```

Step 07 can reuse the same users file to exercise Alice/Bob inbox isolation. Step 08's device authorization flow uses tiny-idp's external strict device endpoint; the generated xgoja host supplies the protected resource API and browser application behavior.

Common symptoms:

- `users file ... no such file`: run from the tinyidp repo root or pass an absolute `TINYIDP_USERS_FILE`.
- `redirect_uri not allowed for this client`: make the app public base URL match the config's redirect URI.
- discovery works at `/` but not under a path: use a path-based issuer and fetch discovery under that same prefix.

## Documentation

Deep-dive docs for the productized strict engine live under `docs/`:

- [`docs/security-profile.md`](docs/security-profile.md) — strict-engine security baseline, enabled controls, unsupported features, release gate.
- [`docs/storage.md`](docs/storage.md) — the `idpstore.Store` profile, transactions, SQLite durability, backup, and restore.
- [`docs/users-and-passwords.md`](docs/users-and-passwords.md) — Argon2id credentials and strict login behavior.
- [`docs/key-rotation.md`](docs/key-rotation.md) — safe signing-key rotation.
- [`docs/admin-cli.md`](docs/admin-cli.md) — the `tinyidp admin` command surface.
- [`docs/conformance.md`](docs/conformance.md) — conformance runner and coverage.

## Project and release workflow

tinyidp follows the Go-Go-Golems release contract. Development commands execute
with `GOWORK=off` so that the same versioned module graph used by CI and release
artifacts is tested locally. The checked-in xapp generator likewise resolves
`xgoja` from `go.mod`; it does not depend on a sibling checkout or a `replace`
directive in `xgoja.yaml`.

```bash
make build             # compile every repository package
make test              # fast ordinary feedback: reusable packages + Message Desk
make test-fosite       # focused strict OAuth/OIDC provider suite (explicit)
make test-k3s-harness  # real two-process trusted-proxy topology proof (explicit)
make test-full         # every repository package, including ttmp harnesses
make lint              # pinned golangci-lint + Glazed + repository analyzers
make verify            # build + test + lint + auditlint + gosec + govulncheck
make docs-export       # write .docsctl/tinyidp-help.sqlite
make goreleaser        # local, unsigned, single-target snapshot
```

The tracked `lefthook.yml` runs the fast test loop and lint before commits. A
push runs `make test-full` plus a local GoReleaser snapshot; this is where the
strict Fosite adapter suite and the production-shaped two-process harness run.
A `v*` tag starts the release workflow:
Linux (including arm64 CGO) and Darwin artifacts are built separately, merged
and signed in the protected `release` environment, then published to GitHub and
the Go-Go-Golems Homebrew tap. The same successful tag exports the canonical
Glazed help corpus as SQLite and calls the shared `publish-docsctl` workflow.
That workflow authenticates to Vault with GitHub OIDC and mints a short-lived,
package-scoped `docsctl-tinyidp-publisher` credential; no registry token is
stored in this repository.

The Go module itself is published by the public Go module proxy from semantic
version tags; it does not require a separate registry credential or publish
job. After a release tag has been pushed, verify that the proxy can resolve the
module with:

```bash
GOWORK=off GOPROXY=https://proxy.golang.org \
  go list -m github.com/go-go-golems/tiny-idp@v0.0.2
```

The `release` Make target performs the tag push and this proxy lookup together.
Use a new tag for each release; an existing tag such as `v0.0.1` must not be
reused after a failed binary publication.

Before the first tag, repository administrators must provision the `release`
environment, signing/Homebrew release secrets, and the Vault role named
`docsctl-tinyidp-publisher`. The project deliberately has no committed
distribution license yet, so package formats that require a license declaration
remain outside the GoReleaser configuration until that product decision is made.

## Status

- **Mock engine (Phases 0–11)** — baseline OIDC happy path, synthetic/scenario users, self-documenting login page, failure scenarios, multiple clients, sessions, claims, debug UI, refresh tokens, JWKS rotation, RP-initiated logout, device grant, and DPoP (done).
- **Glazed CLI** — reusable `oidc` field section, layered config (flags/env/config), **profiles**, **`print-config`**, and `--engine` selection (done).
- **Strict/production engine** — Fosite-backed Auth-Code + PKCE, durable SQLite storage, persistent consent, Argon2id user/password storage (bcrypt for client secrets), persistent signing keys + rotation, `pkg/embeddedidp` embeddable provider, and the `tinyidp admin` operational CLI (done; see the `TINYIDP-PROD-001`, `TINYIDP-USERS-001`, and `TINYIDP-ADMIN-001` tickets under `ttmp/`).
- **Conformance** — the strict engine passed a hosted OpenID Foundation **Basic OP** run with zero hard failures (not formal certification); see [`docs/conformance.md`](docs/conformance.md).
- **Validated integration** — verified live as the OIDC provider for **Jitsi Meet** via an OIDC→Jitsi-JWT adapter (`TINYIDP-JITSI-001`); no new IdP features were required.
- **In progress / next** — structured, config-backed runtime configuration and store loading for `serve` (design-only in `TINYIDP-PROD-CONFIG-001`; today the durable path is `admin` + `pkg/embeddedidp`), a token `/revoke`/`/introspect` route, and broader strict-mode protocol coverage.
