---
Title: "OIDC configuration section"
Slug: oidc-config
Short: "The reusable `oidc` field section that configures the mock IdP provider."
Topics:
- oidc
- config
Commands:
- serve
Flags:
- issuer
- addr
- client-id
- client-secret
- redirect-uris
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `oidc` section is a reusable Glazed field section that configures the
mock OIDC Identity Provider. It is composed into the `serve` command and
can be reused by any future command that needs provider configuration.

## Fields

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--issuer` | `TINYIDP_ISSUER` | `http://localhost:5556` | Issuer URL; endpoints are derived from it. |
| `--addr` | `TINYIDP_ADDR` | `127.0.0.1:5556` | Listen address (loopback by default). |
| `--client-id` | `TINYIDP_CLIENT_ID` | `dev-client` | Accepted client ID. |
| `--client-secret` | `TINYIDP_CLIENT_SECRET` | (empty) | If set, `/token` enforces it; if empty, the client is public. |
| `--redirect-uris` | `TINYIDP_REDIRECT_URIS` | `http://localhost:3000/callback,http://127.0.0.1:3000/callback` | Allowlist of redirect URIs. |

## Config file form

In a YAML config file, these fields live under the `oidc` section slug:

    oidc:
      issuer: http://localhost:5556
      addr: 127.0.0.1:5556
      client-id: dev-client
      client-secret: dev-secret
      redirect-uris:
        - http://localhost:8080/callback

## Precedence

From lowest to highest:

1. Section defaults
2. Profiles (`tinyidp help profiles`)
3. Config files
4. Environment variables (`TINYIDP_*`)
5. CLI flags

## Introspection

To see the resolved configuration before running, use the Glazed
command-settings flags:

    tinyidp serve --print-parsed-fields

## Reuse

The section is defined once in `internal/sections/oidc` and composed into
commands via `cmds.WithSections(oidc.NewSection(), ...)`. Any future
command (for example a `print-config` or `gen-key` verb) can compose the
same section to get identical flags, env vars, and config-file schema
without redefining them.
