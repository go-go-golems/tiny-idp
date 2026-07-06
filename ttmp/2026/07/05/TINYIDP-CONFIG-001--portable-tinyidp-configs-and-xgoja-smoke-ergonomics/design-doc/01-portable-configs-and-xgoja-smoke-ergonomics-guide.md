---
Title: Portable Configs and xgoja Smoke Ergonomics Guide
Ticket: TINYIDP-CONFIG-001
Status: active
Topics:
    - oidc
    - testing
    - go
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile
      Note: |-
        Step 06 tinyidp smoke target to make portable and documented
        Existing tinyidp smoke variables and Step 06 target
    - Path: ../../../../../../../go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/scripts/tinyidp_login_smoke.py
      Note: xgoja helper updated to submit fixture password (commit eccfa5b)
    - Path: ../../../../../../../go-go-goja/examples/xgoja/23-personal-knowledge-inbox/Makefile
      Note: Aggregate personal-inbox smoke and tinyidp-smoke entrypoint
    - Path: README.md
      Note: Public config example and xgoja smoke docs
    - Path: cmd/tinyidp/doc/pages/getting-started.md
      Note: Getting-started config example docs
    - Path: cmd/tinyidp/doc/pages/reference.md
      Note: Reference docs for config examples and users-file paths
    - Path: examples/configs/confidential-web-app.yaml
      Note: Confidential web app config example
    - Path: examples/configs/dev-root.yaml
      Note: Basic root-issuer config example
    - Path: examples/configs/personal-inbox-realm.yaml
      Note: Personal-inbox path issuer config example
    - Path: examples/configs/personal-inbox-root.yaml
      Note: Personal-inbox root issuer config example
    - Path: examples/configs/public-spa-pkce.yaml
      Note: Public SPA PKCE config example
    - Path: examples/users/personal-inbox-users.yaml
      Note: Personal-inbox seeded users fixture
    - Path: internal/cmds/serve.go
      Note: |-
        Serve command wiring from OIDC settings to server/client registry
        serve command consumes OIDC settings and users-file
    - Path: internal/sections/oidc/section.go
      Note: |-
        Current Glazed OIDC config section and defaults
        Current OIDC config section and config-file keys
    - Path: internal/sections/oidc/settings.go
      Note: |-
        Current typed settings decode and issuer normalization
        Typed OIDC settings decode and issuer normalization
ExternalSources: []
Summary: Design and implementation guide for making tinyidp configs portable across local checkouts, CI, and xgoja smoke tests.
LastUpdated: 2026-07-05T17:45:00-04:00
WhatFor: Use when implementing example config files, xgoja smoke snippets, profile conventions, and config-debugging ergonomics for tinyidp.
WhenToUse: Read before changing tinyidp CLI config fields, example config layout, xgoja Makefile smoke targets, or docs for root vs realm issuers.
---



# Portable Configs and xgoja Smoke Ergonomics Guide

## Executive summary

`tinyidp` already has the OIDC server behavior needed by the current xgoja smokes: discovery, authorize, token, userinfo, JWKS, sessions, seeded users, and path-based issuer routes. The next usability gap is not another protocol endpoint. It is making the tool easy to run in repeatable local and CI contexts without copying long command lines between tickets.

This ticket designs a portable configuration layer and smoke-test ergonomics package for `tinyidp`. The target outcome is that an intern can clone the repository, pick an example config, run `tinyidp serve --config-file examples/configs/personal-inbox.yaml`, and point an xgoja smoke test at it without reverse-engineering issuer URLs, redirect URI allowlists, seeded user files, or path-prefix behavior.

The main design is intentionally conservative:

1. Keep the existing Glazed OIDC section as the single source of CLI/config/env fields.
2. Add checked-in example configs and user files under an `examples/` tree.
3. Add copy/paste smoke snippets for the xgoja examples that already use `tinyidp`.
4. Add a small machine-readable config inspection mode only if `print-config` is not sufficient.
5. Keep root issuer and realm-path issuer examples side by side so tests can choose clarity or Keycloak-shaped compatibility.

This work makes `tinyidp` usable as a daily integration-test dependency. It does not turn it into a production identity provider.

## Problem statement and scope

The current smokes work, but they require the caller to know a lot of details:

- `--issuer` and `--addr` are independent values.
- The issuer may be root-shaped (`http://127.0.0.1:19087`) or realm-shaped (`http://127.0.0.1:19087/realms/personal-inbox`).
- Redirect URIs must match the generated app callback exactly.
- Seeded users live in a separate YAML file.
- `GOWORK=off` is often required when running `tinyidp` from the larger workspace.
- xgoja examples may need different ports to avoid collisions.
- Step-level Makefiles use local assumptions such as `TINYIDP_ROOT ?= $(REPO_ROOT)/../2026-06-22--mock-oidc-idp`.

The goal of this ticket is to remove that cognitive overhead for local and CI users. It covers configuration files, examples, docs, smoke command shape, and debugging ergonomics. It does not implement passwords, Keycloak claim presets, or `tinyidptest`; those are separate tickets.

## Current-state analysis

### Existing Glazed config section

`internal/sections/oidc/section.go` already defines the reusable config surface. It provides fields for:

- `issuer`
- `addr`
- `client-id`
- `client-secret`
- `redirect-uris`
- `users-file`

The comments in that file correctly state that the section maps to flags, `TINYIDP_*` environment variables, and config-file keys under `oidc:`. This means portable YAML configs can be added without inventing a second parser.

`internal/sections/oidc/settings.go` decodes the section into `Settings` and trims a trailing slash from `Issuer`. This is important for example docs: all examples should use issuer URLs without trailing slashes, and the code should continue to normalize accidental trailing slashes.

### Existing serve command

`internal/cmds/serve.go` builds the scenario registry from `UsersFile`, builds the client registry from the OIDC config, constructs `server.New`, registers routes, logs the chosen address and issuer, and runs `http.ListenAndServe`.

That makes `serve` the right integration point for portable examples. Example configs do not need a new command; they only need stable files and documentation.

### Existing xgoja smoke pattern

The current personal-inbox Step 06/07/08 smokes define variables such as:

```make
TINYIDP_ROOT ?= $(REPO_ROOT)/../2026-06-22--mock-oidc-idp
TINYIDP_ADDR ?= 127.0.0.1:19087
TINYIDP_APP_ADDR ?= 127.0.0.1:19794
TINYIDP_ISSUER := http://$(TINYIDP_ADDR)
TINYIDP_USERS_FILE ?= $(EXAMPLE_DIR)/../tinyidp-users.yaml
```

The pattern is functional but still example-local. The same concepts should be documented as a reusable smoke recipe and backed by checked-in `tinyidp` configs.

## Proposed repository layout

Add an `examples/` tree to the `tinyidp` repo:

```text
examples/
  README.md
  configs/
    dev-root.yaml
    personal-inbox-root.yaml
    personal-inbox-realm.yaml
    public-spa-pkce.yaml
    confidential-web-app.yaml
  users/
    personal-inbox-users.yaml
    roles-demo-users.yaml
  snippets/
    xgoja-personal-inbox-step06.md
    xgoja-personal-inbox-step07.md
    xgoja-personal-inbox-step08.md
```

### `examples/configs/dev-root.yaml`

A minimal default that mirrors current defaults but makes the shape explicit:

```yaml
oidc:
  issuer: http://127.0.0.1:5556
  addr: 127.0.0.1:5556
  client-id: dev-client
  redirect-uris:
    - http://localhost:3000/callback
    - http://127.0.0.1:3000/callback
```

### `examples/configs/personal-inbox-root.yaml`

A root-issuer xgoja config:

```yaml
oidc:
  issuer: http://127.0.0.1:19087
  addr: 127.0.0.1:19087
  client-id: personal-inbox-local
  redirect-uris:
    - http://127.0.0.1:19794/auth/callback
  users-file: examples/users/personal-inbox-users.yaml
```

### `examples/configs/personal-inbox-realm.yaml`

A Keycloak-shaped issuer config:

```yaml
oidc:
  issuer: http://127.0.0.1:19087/realms/personal-inbox
  addr: 127.0.0.1:19087
  client-id: personal-inbox-local
  redirect-uris:
    - http://127.0.0.1:19794/auth/callback
  users-file: examples/users/personal-inbox-users.yaml
```

The `addr` is still root bind address. The `issuer` carries the realm path. That distinction should be called out prominently in docs.

### `examples/users/personal-inbox-users.yaml`

```yaml
users:
  - login: alice
    sub: user-alice-fixed
    email: alice@example.test
    name: Alice Inbox
    email_verified: true
    claims:
      groups: [inbox-users]
      tenant: personal
  - login: bob
    sub: user-bob-fixed
    email: bob@example.test
    name: Bob Inbox
    email_verified: true
    claims:
      groups: [inbox-users]
      tenant: personal
```

This mirrors the already-working seeded-user model. The password semantics ticket may later add optional `password` fields; this ticket should not add them.

## User-facing workflows

### Workflow 1: run a root issuer

```bash
cd /path/to/2026-06-22--mock-oidc-idp
GOWORK=off go run ./cmd/tinyidp serve \
  --config-file examples/configs/personal-inbox-root.yaml
```

Then configure the app under test with:

```text
issuer:    http://127.0.0.1:19087
client_id: personal-inbox-local
callback:  http://127.0.0.1:19794/auth/callback
```

### Workflow 2: run a Keycloak-shaped realm issuer

```bash
cd /path/to/2026-06-22--mock-oidc-idp
GOWORK=off go run ./cmd/tinyidp serve \
  --config-file examples/configs/personal-inbox-realm.yaml
```

Discovery is available at:

```text
http://127.0.0.1:19087/realms/personal-inbox/.well-known/openid-configuration
```

The advertised authorize/token/userinfo/JWKS URLs should also include `/realms/personal-inbox`.

### Workflow 3: xgoja Makefile override

From an xgoja example:

```bash
make tinyidp-smoke \
  TINYIDP_ROOT=/path/to/2026-06-22--mock-oidc-idp \
  TINYIDP_ADDR=127.0.0.1:19087 \
  TINYIDP_APP_ADDR=127.0.0.1:19794 \
  TINYIDP_ISSUER=http://127.0.0.1:19087/realms/personal-inbox
```

The examples should document this, but the preferred long-term path is for example Makefiles to point at checked-in config files where possible.

## Proposed implementation phases

### Phase 1: example config files

1. Create `examples/configs/`.
2. Add root and realm configs for personal-inbox.
3. Add generic public SPA and confidential web-app configs.
4. Add `examples/users/personal-inbox-users.yaml`.
5. Add a README explaining the difference between `issuer`, `addr`, and redirect URIs.

### Phase 2: xgoja snippets

1. Add `examples/snippets/xgoja-personal-inbox-step06.md`.
2. Include root and realm issuer commands.
3. Include common failure symptoms:
   - discovery 404 means issuer path is wrong or old tinyidp build is running;
   - invalid redirect URI means config callback does not match app public base URL;
   - stale session means restart or `/debug/reset`.

### Phase 3: config validation smoke

Add a script or Make target that starts `tinyidp` with each checked-in config and verifies discovery:

```bash
for cfg in examples/configs/*.yaml; do
  tinyidp serve --config-file "$cfg" &
  curl -fsS "$issuer/.well-known/openid-configuration"
  kill $pid
done
```

Prefer a Go test if config parsing and port allocation become complex.

### Phase 4: docs and help integration

1. Link `examples/README.md` from the root README.
2. Add a short `tinyidp help examples` page if the help system supports it cleanly.
3. Update `tinyidp help reference` with the example config directory.

## API and file references

Implementation files:

- `internal/sections/oidc/section.go` — add fields only if examples reveal missing config.
- `internal/sections/oidc/settings.go` — keep issuer normalization centralized.
- `internal/cmds/serve.go` — no changes expected unless adding config-derived logging.
- `README.md` — link example configs.
- `cmd/tinyidp/doc/pages/getting-started.md` — add example config path.
- `cmd/tinyidp/doc/pages/reference.md` — add config-file examples.

Consumer files:

- `go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile`
- `go-go-goja/examples/xgoja/23-personal-knowledge-inbox/07-user-scoped-inbox/Makefile`
- `go-go-goja/examples/xgoja/23-personal-knowledge-inbox/08-device-authorization/Makefile`

## Decision records

### Decision: Use checked-in YAML configs rather than generating configs first

- **Context:** The tool already supports config files through Glazed. The immediate pain is discoverability, not parser capability.
- **Options considered:** Add `tinyidp init-config`; add checked-in examples; only document flags.
- **Decision:** Start with checked-in examples.
- **Rationale:** Examples are reviewable, versionable, and easy for xgoja docs to reference.
- **Consequences:** A future `init-config` command can copy these examples, but is not required for first usability.
- **Status:** proposed

### Decision: Keep `issuer` and `addr` separate

- **Context:** Realm-path issuers use the same bind address but a path-prefixed public issuer.
- **Options considered:** Derive `addr` from `issuer`; keep separate fields; add a separate public-base-url field.
- **Decision:** Keep existing separate `issuer` and `addr` fields.
- **Rationale:** This matches current implementation and supports reverse proxy or path-prefix use.
- **Consequences:** Docs must explain the distinction clearly.
- **Status:** proposed

### Decision: Do not add password or claim-preset config in this ticket

- **Context:** Password semantics and Keycloak claim presets are separate requested tickets.
- **Options considered:** Combine everything into one usability ticket; keep separate tickets.
- **Decision:** Keep this ticket focused on portable config and smoke ergonomics.
- **Rationale:** Combining them would make review and implementation harder.
- **Consequences:** Example files may later gain password or preset fields in follow-up tickets.
- **Status:** proposed

## Testing strategy

1. Config parse tests:
   - run `tinyidp print-config --config-file examples/configs/personal-inbox-root.yaml`;
   - assert issuer, addr, client id, redirect URIs, and users file.
2. Runtime discovery tests:
   - start root config;
   - fetch root discovery;
   - assert endpoint URLs.
3. Runtime realm tests:
   - start realm config;
   - fetch prefixed discovery;
   - assert endpoint URLs include `/realms/personal-inbox`.
4. xgoja integration smoke:
   - Step 06 root issuer;
   - Step 06 realm issuer;
   - optionally Step 07 and Step 08 after ports are stable.

## Risks and open questions

- Example ports may collide on developer machines. The docs should recommend override variables and high ports.
- Config files with relative `users-file` paths need clear resolution behavior. If Glazed resolves them relative to the current working directory, docs must instruct users to run from the repo root or pass absolute paths.
- CI may need installed `tinyidp` binaries rather than `go run` for speed.
- If xgoja examples live in a different workspace layout, `TINYIDP_ROOT` defaults must remain overrideable.

## References

- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/sections/oidc/section.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/sections/oidc/settings.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/cmds/serve.go`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/06-browser-login-keycloak/Makefile`
- `/home/manuel/workspaces/2026-06-12/goja-express-auth/go-go-goja/examples/xgoja/23-personal-knowledge-inbox/README.md`
