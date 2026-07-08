---
Title: Structured Production Configuration Design and Implementation Guide
Ticket: TINYIDP-PROD-CONFIG-001
Status: active
Topics:
    - go
    - identity
    - oidc
    - auth
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://internal/cmds/serve.go
      Note: Runtime wiring point for mock and strict engines
    - Path: repo://internal/domain/types.go
      Note: Domain clients, users, sessions, consents, and keys that config materializes
    - Path: repo://internal/sections/oidc/section.go
      Note: Current flat OIDC configuration fields that structured production config should not overload
    - Path: repo://internal/sections/oidc/settings.go
      Note: Current type-safe flat settings decode target
    - Path: repo://internal/store/sqlite/migrations/001_schema.sql
      Note: Durable schema available to product config runtime
    - Path: repo://pkg/embeddedidp/options.go
      Note: Existing production validation boundary for strict embedded provider
ExternalSources: []
Summary: Design for replacing flat dev-oriented tinyidp flags with a typed, validated, production configuration system.
LastUpdated: 2026-07-08T01:05:00-04:00
WhatFor: Use this when implementing structured production configuration for tinyidp.
WhenToUse: Read before changing CLI flags, config loading, production startup validation, storage/key/client bootstrapping, or environment/profile behavior.
---


# Structured Production Configuration Design and Implementation Guide

## Executive Summary

`tinyidp` now has a strict Fosite-backed engine, durable protocol storage, consent, sessions, audit hooks, rate limiting hooks, and key rotation. The remaining product gap is that production runtime configuration is still assembled through a flat development section originally designed for a mock IdP. This guide proposes a typed production configuration package that can load YAML, environment overrides, and optional secret references into one validated `Config` model before any server starts.

The target outcome is a startup path where an operator can run `tinyidp serve --config-file /etc/tinyidp/config.yaml`, receive deterministic validation errors, and know that the configured issuer, TLS/cookie policy, clients, storage, keys, audit sink, rate limiter, and user source are production-safe. The mock engine keeps its convenient flat dev defaults; the strict engine gains an explicit product-grade configuration contract.

## Problem Statement

The current CLI surface is good for local testing and hosted conformance runs, but it is not yet a production configuration system.

Evidence from the current codebase:

- `internal/sections/oidc/section.go:31-78` defines one reusable Glazed section named `oidc` with flat fields such as `issuer`, `addr`, `client-id`, `client-secret`, `redirect-uris`, `extra-clients`, `users-file`, and `engine`.
- `internal/sections/oidc/settings.go:13-22` decodes those fields into a single `Settings` struct. This is type-safe, but it is not expressive enough for production clients, storage, cookies, signing key policy, audit, or secret references.
- `internal/cmds/profiles.go:38-45` already defines a reliable precedence chain: defaults < profiles < config < env < args < flags. This should be preserved instead of inventing another ad hoc parser.
- `internal/cmds/config.go:9-30` makes `--config-file` load a Glazed explicit config layer. That is a useful foundation, but the loaded shape is still field-section oriented rather than product oriented.
- `internal/cmds/serve.go:215-255` builds strict mode from an in-memory store, generated development key, fixed secret key, dev mode, and clients/users translated from the local test registries.
- `pkg/embeddedidp/options.go:42-74` already has production validation for issuer, store presence, secure cookies, persistent store, and active signing key. Structured configuration should feed this API instead of bypassing it.

The problem is not that configuration is absent. The problem is that production-critical configuration is distributed across local defaults, command-line flags, development registries, and embedding options. That makes it hard to review, audit, document, test, and run safely.

## Scope

This ticket covers the design and implementation plan for structured production runtime configuration.

In scope:

- A stable YAML/JSON-compatible configuration schema.
- Typed Go structs for loading, normalization, validation, and redacted rendering.
- Secret reference handling for client secrets, cookie/token keys, database credentials, and optional admin bootstrap secrets.
- Compatibility with the existing Glazed precedence chain.
- Strict-mode startup integration with durable SQLite storage and `pkg/embeddedidp.Options`.
- Config validation tests and examples.

Out of scope for this ticket:

- Full admin CLI implementation. That is covered by `TINYIDP-ADMIN-001`.
- Real password credential storage. That is covered by `TINYIDP-USERS-001`.
- Docker/systemd packaging. The config should be designed so those can be added later.

## Current-State Architecture

### Existing config flow

```text
CLI flags / env / config file / profile
              |
              v
Glazed oidc section
internal/sections/oidc.Settings
              |
              v
internal/cmds.ServeCommand.Run
              |
     +--------+---------+
     |                  |
 mock engine        fosite engine
 local registry     memory store + dev key + dev secret key
```

The existing flow is intentionally simple. The strict engine path currently adapts the same development inputs into a strict provider. That was appropriate for conformance work, but product use needs a separate production flow.

### Existing validation boundary

`pkg/embeddedidp/options.go:42-74` is the best current validation boundary. It checks:

- issuer validity through `oidcmeta.ValidateIssuer`;
- store presence;
- all configured clients through `domain.Client.Validate`;
- secure cookies in production;
- persistent store in production;
- active signing key in production.

The new configuration package should build values that reach this boundary. It should not duplicate all protocol validation if the domain and embedded provider already know how to validate it.

### Existing domain support

`internal/domain/types.go:14-29` already defines production-oriented clients with hashed secrets, exact redirect URIs, allowed scopes, PKCE requirement, token TTLs, timestamps, and disabled state. `internal/domain/types.go:133-142` defines persisted signing keys with lifecycle fields. `internal/storage/interfaces.go:19-92` defines store capabilities and persistent-store reporting.

This means structured config does not need to invent new runtime concepts for clients and keys. It needs to describe how operators express them, validate them, and load them into existing domain/store structures.

## Proposed Configuration Shape

The product config should be nested by operational concern, not by historical CLI flag. A minimal production file should look like this:

```yaml
version: 1
mode: production

server:
  listen_addr: 127.0.0.1:5556
  public_issuer: https://idp.example.com
  behind_proxy: true
  trusted_proxy_cidrs:
    - 127.0.0.1/32

storage:
  driver: sqlite
  sqlite:
    path: /var/lib/tinyidp/tinyidp.db
    busy_timeout: 5s
    wal: true

security:
  cookies:
    secure: true
    same_site: Lax
  token_secret:
    from_file: /etc/tinyidp/token-secret
  csrf_secret:
    from_file: /etc/tinyidp/csrf-secret
  require_pkce_for_public_clients: true

keys:
  active_kid: 2026-07-08-rsa-1
  load:
    - kid: 2026-07-08-rsa-1
      algorithm: RS256
      private_key_pem_file: /etc/tinyidp/keys/2026-07-08-rsa-1.pem
      active: true
      not_before: 2026-07-08T00:00:00Z
      not_after: 2027-07-08T00:00:00Z

clients:
  - id: web-app
    public: false
    secret:
      from_file: /etc/tinyidp/clients/web-app.secret
    redirect_uris:
      - https://app.example.com/oauth/callback
    post_logout_redirect_uris:
      - https://app.example.com/logout/callback
    allowed_scopes: [openid, profile, email, offline_access]
    require_pkce: true
    access_token_ttl: 1h
    id_token_ttl: 1h
    refresh_token_ttl: 720h

audit:
  sink: stderr-json
  redact: true

rate_limit:
  login:
    window: 1m
    max: 10
  token:
    window: 1m
    max: 60
```

Development fixtures can still use the existing `oidc:` section or a shorter product config:

```yaml
version: 1
mode: dev
server:
  listen_addr: 127.0.0.1:5556
  public_issuer: http://localhost:5556
storage:
  driver: memory
clients:
  - id: dev-client
    public: true
    redirect_uris: [http://localhost:3000/callback]
    allowed_scopes: [openid, profile, email]
```

## Proposed Go API

Create a new package, preferably `internal/appconfig`, so the product config is separate from the Glazed field section.

```go
package appconfig

type Config struct {
    Version int
    Mode domain.Mode
    Server ServerConfig
    Storage StorageConfig
    Security SecurityConfig
    Keys KeyConfig
    Clients []ClientConfig
    Audit AuditConfig
    RateLimit RateLimitConfig
    Users UserBootstrapConfig // only bootstrap/source pointers; not credential internals
}

type ServerConfig struct {
    ListenAddr string
    PublicIssuer string
    BehindProxy bool
    TrustedProxyCIDRs []string
}

type StorageConfig struct {
    Driver string // memory, sqlite
    SQLite SQLiteConfig
}

type SecretRef struct {
    Value string
    FromEnv string
    FromFile string
}

type ClientConfig struct {
    ID string
    Public bool
    Secret SecretRef
    RedirectURIs []string
    PostLogoutRedirectURIs []string
    AllowedScopes []string
    RequirePKCE *bool
    AccessTokenTTL time.Duration
    IDTokenTTL time.Duration
    RefreshTokenTTL time.Duration
    Disabled bool
}

func Load(ctx context.Context, path string, env Env) (Config, error)
func (c *Config) Normalize() error
func (c Config) Validate(ctx context.Context) error
func (c Config) Redacted() Config
func (c Config) BuildRuntime(ctx context.Context) (*Runtime, error)
```

`BuildRuntime` should be intentionally boring: open the configured store, apply migrations, upsert configured clients and keys, and return objects that `serve` can pass into `embeddedidp.New`.

```go
type Runtime struct {
    Options embeddedidp.Options
    Store storage.Store
    Close func(context.Context) error
}
```

## Loading and Validation Pipeline

```text
read config file
      |
      v
unmarshal YAML/JSON into appconfig.Config
      |
      v
normalize defaults that are safe and explicit
      |
      v
resolve secret references
      |
      v
validate syntax and production policy
      |
      v
open/migrate store
      |
      v
materialize clients and keys
      |
      v
validate embeddedidp.Options
      |
      v
start HTTP server
```

### Pseudocode: loader

```go
func Load(ctx context.Context, path string, env Env) (Config, error) {
    raw, err := os.ReadFile(path)
    if err != nil { return Config{}, err }

    var cfg Config
    if err := yaml.Unmarshal(raw, &cfg); err != nil { return Config{}, err }

    if err := cfg.Normalize(); err != nil { return Config{}, err }
    if err := cfg.ResolveSecrets(ctx, env); err != nil { return Config{}, err }
    if err := cfg.Validate(ctx); err != nil { return Config{}, err }
    return cfg, nil
}
```

### Pseudocode: secret resolution

```go
func (s SecretRef) Resolve(env Env) ([]byte, error) {
    set := countNonEmpty(s.Value, s.FromEnv, s.FromFile)
    if set != 1 { return nil, fmt.Errorf("exactly one secret source is required") }
    switch {
    case s.Value != "":
        return []byte(s.Value), nil // reject in production unless AllowInlineSecrets is true
    case s.FromEnv != "":
        return []byte(env.Getenv(s.FromEnv)), nil
    case s.FromFile != "":
        b, err := os.ReadFile(s.FromFile)
        return bytes.TrimSpace(b), err
    }
}
```

### Validation rules

Production validation should fail closed:

- `mode` must be `production` or `dev`.
- `server.public_issuer` must pass existing issuer validation.
- production issuer must be HTTPS and must not be loopback.
- production cookies must be secure.
- storage driver must be persistent in production unless an explicit unsafe test flag is set.
- inline secrets are rejected in production by default.
- client IDs must be unique.
- redirect URIs must be absolute, exact, and must not contain fragments.
- confidential clients require a secret reference.
- public clients require PKCE.
- at least one active signing key must exist before serving production.
- token/cookie/CSRF secrets must meet minimum entropy and length requirements.
- unsupported fields should be rejected by using a strict YAML decoder.

## Integration with Existing CLI

Do not remove the existing `oidc` section immediately. Instead, split `serve` into two startup paths:

```text
serve --engine mock
  uses current oidc.Settings and scenario registry

serve --engine fosite --config-file production.yaml
  if product config exists: appconfig runtime path
  else: legacy dev strict path for conformance/dev use
```

This preserves mock behavior while giving production strict mode a real contract.

Implementation sketch in `internal/cmds/serve.go`:

```go
func (c *ServeCommand) Run(ctx context.Context, vals *values.Values) error {
    cfg, _ := oidc.GetSettings(vals)
    if cfg.Engine == "fosite" && hasProductConfig(vals) {
        runtime, err := appconfig.LoadRuntime(ctx, productConfigPath(vals))
        if err != nil { return err }
        defer runtime.Close(ctx)
        provider, err := embeddedidp.New(runtime.Options)
        if err != nil { return err }
        return listen(ctx, runtime.ListenAddr, provider.Handler())
    }
    return runLegacyServePath(ctx, cfg)
}
```

The exact command flag can be `--config-file`, but the parser should distinguish between the old `oidc:` shape and the new `version/server/storage/...` shape. During the migration period, `tinyidp print-config` should also learn how to print the redacted product config.

## Files to Change

Primary files:

- `internal/appconfig/config.go` — config structs, defaults, validation.
- `internal/appconfig/load.go` — strict YAML/JSON loading and secret resolution.
- `internal/appconfig/runtime.go` — store opening, migrations, client/key materialization, embedded options.
- `internal/appconfig/config_test.go` — validation unit tests.
- `internal/appconfig/testdata/*.yaml` — valid and invalid examples.
- `internal/cmds/serve.go` — product strict startup path.
- `internal/cmds/print_config.go` — redacted config printing and validation diagnostics.
- `docs/configuration.md` — operator-facing reference.

Existing reference points:

- `internal/sections/oidc/section.go:31-78` for the legacy flat field section.
- `internal/cmds/profiles.go:38-45` for precedence that should remain documented.
- `internal/cmds/serve.go:215-255` for the dev strict path that should not become production config.
- `pkg/embeddedidp/options.go:42-74` for final runtime validation.
- `internal/domain/types.go:14-29` for client fields.
- `internal/store/sqlite/migrations/001_schema.sql:1-20` for currently available durable tables.

## Implementation Plan

### Phase 1: Add schema and parser

1. Create `internal/appconfig`.
2. Define config structs and custom duration parsing if YAML does not map directly to `time.Duration`.
3. Add strict YAML decode that rejects unknown fields.
4. Add `SecretRef` with redacted rendering.
5. Add tests for minimal dev config, minimal production config, duplicate clients, missing issuer, missing secrets, and unknown fields.

### Phase 2: Add product validation

1. Reuse `oidcmeta.ValidateIssuer` for issuer policy.
2. Convert `ClientConfig` into `domain.Client` and call `domain.Client.Validate`.
3. Validate secrets before constructing runtime values.
4. Validate storage driver and SQLite path.
5. Validate that exactly one signing key is active.

### Phase 3: Build runtime materialization

1. Open SQLite when `storage.driver=sqlite`.
2. Run existing SQLite migrations.
3. Upsert configured clients with hashed secrets.
4. Load or create signing keys according to policy.
5. Construct `embeddedidp.Options` and call `Validate`.

### Phase 4: Wire strict serve path

1. Update `tinyidp serve` to prefer product config when running `--engine fosite` with a product config file.
2. Keep mock and legacy strict dev path intact.
3. Add smoke tests that start strict mode from config.
4. Add `tinyidp print-config --redacted` coverage.

### Phase 5: Document operations

1. Add `docs/configuration.md`.
2. Add examples under `examples/config/`.
3. Update `docs/security-profile.md` to point to production config validation.
4. Add a conformance example config that replaces the long ad hoc hosted-suite command.

## Decision Records

### Decision 1: Use a new product config package, not more `oidc` fields

Status: proposed.

Options considered:

- Extend `internal/sections/oidc` with dozens of production fields.
- Create a new `internal/appconfig` model and keep the old section as a dev compatibility layer.

Decision: create `internal/appconfig`.

Rationale: production configuration has nested objects, secret references, validation phases, and redaction needs. Encoding all of that as flat Glazed fields would make the CLI harder to use and the config harder to review.

Consequence: there will temporarily be two config paths. This is acceptable because mock/local behavior must remain stable.

### Decision 2: Reject inline secrets in production by default

Status: proposed.

Decision: allow `value:` for dev and tests; require `from_file:` or `from_env:` for production unless an explicit unsafe escape hatch is used.

Rationale: product config files are often committed, templated, attached to tickets, or printed for support. Inline secrets create unnecessary leakage risk.

### Decision 3: Materialize clients and keys into the durable store at startup

Status: proposed.

Decision: config describes desired startup state; runtime opens the store and upserts clients/keys before validation.

Rationale: the strict provider already reads clients and keys from `storage.Store`. Feeding that store keeps the provider simple and makes admin commands operate on the same data model.

## Testing Strategy

Unit tests:

- strict YAML rejects unknown fields;
- valid dev config loads;
- valid production config loads with file/env secrets;
- duplicate client IDs fail;
- public client without PKCE fails in production;
- confidential client without secret fails;
- inline secret fails in production;
- invalid redirect URI fails;
- multiple active keys fail;
- redacted rendering never contains raw secret bytes.

Integration tests:

- start strict provider from a temporary SQLite config;
- complete Authorization Code + PKCE flow;
- restart with the same DB and confirm keys/clients persist;
- run `scripts/run-conformance.sh` with a config-backed strict server;
- verify `go test ./...` remains green.

Operator smoke test:

```bash
tinyidp serve --engine fosite --config-file examples/config/production.sqlite.yaml --print-parsed-fields
tinyidp print-config --config-file examples/config/production.sqlite.yaml --redacted
go test ./... -count=1
scripts/run-conformance.sh
```

## Risks and Mitigations

- Risk: breaking existing mock users who rely on simple flags. Mitigation: keep the current `oidc` section and mock path unchanged.
- Risk: silently accepting misspelled production keys. Mitigation: strict YAML decode and tests for unknown fields.
- Risk: leaking secrets through diagnostics. Mitigation: all config display uses `Redacted()`; tests assert raw secrets are absent.
- Risk: startup mutates store unexpectedly. Mitigation: log planned materialization, support dry-run validation later through admin CLI, and make upserts idempotent.
- Risk: config and admin CLI compete for ownership. Mitigation: document that config can bootstrap desired state, while admin CLI manages operational state after initialization.

## Intern Implementation Notes

When implementing this, start with parsing and validation before touching server startup. A configuration system is easiest to review when the first PR has no networking behavior. Make invalid states unrepresentable where possible; otherwise, make validation errors precise and include the field path, for example `clients[0].redirect_uris[1]: fragments are not allowed`.

Do not store raw client secrets in `domain.Client`. The domain model already has `SecretHash` at `internal/domain/types.go:16-18`. The loader should resolve the secret just long enough to hash it, then discard the raw bytes.

## References

- `internal/sections/oidc/section.go:31-78`
- `internal/sections/oidc/settings.go:13-22`
- `internal/cmds/config.go:9-30`
- `internal/cmds/profiles.go:38-45`
- `internal/cmds/serve.go:215-255`
- `pkg/embeddedidp/options.go:42-74`
- `internal/domain/types.go:14-29`
- `internal/storage/interfaces.go:19-92`
- `internal/store/sqlite/migrations/001_schema.sql:1-20`
