---
Title: 'Plugin API research: configuration, Goja policy, runtime lifecycle, and observability'
Ticket: TINYIDP-PLUGIN-001
Status: active
Topics:
    - architecture
    - auth
    - jitsi
    - operations
    - security
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/tinyidp/main.go
      Note: Production and development Glazed command construction
    - Path: repo://internal/cmds/profiles.go
      Note: Established profile, config, environment, argument, and flag precedence
    - Path: repo://internal/cmds/serve_production.go
      Note: Production runtime composition and secret-file handling precedent
    - Path: repo://internal/sections/oidc/section.go
      Note: Reusable Glazed section pattern
    - Path: repo://pkg/embeddedidp/provider.go
      Note: HTTP handler, readiness, lifecycle, and runtime stats
    - Path: repo://pkg/idpscript/capabilities.go
      Note: Bounded Goja capability contract
ExternalSources:
    - https://jitsi.github.io/handbook/docs/devops-guide/token-authentication/
    - https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md
    - https://github.com/jitsi-contrib/jitsi-oidc-adapter
    - https://pkg.go.dev/plugin
    - https://github.com/hashicorp/go-plugin
Summary: Exploratory draft of a TinyIDP plugin structure; intended for option review before the full design.
LastUpdated: 2026-07-23T16:32:31.900687375-04:00
WhatFor: ""
WhenToUse: ""
---


# Plugin API research: configuration, Goja policy, runtime lifecycle, and observability

## Executive Summary

This is an exploratory draft, not the final implementation guide.

The strongest first design is a **compiled-in plugin registry**. A plugin is a
Go package linked into TinyIDP and registered before Glazed parses the command.
It contributes a typed Glazed section, prepares validated configuration, and
builds a runtime handler from narrow host services. Goja remains an optional
policy layer; it does not become the plugin loader or receive HTTP, filesystem,
database, or signing authority.

For Jitsi, TinyIDP would host the browser/token-translation behavior currently
provided by a standalone OIDC adapter. Prosody remains responsible for XMPP
signaling and local JWT/room enforcement.

## Problem Statement

We want integrations to reuse TinyIDP identity, signup, sessions, scripting,
audit, and deployment machinery without adding every application's protocol
directly to the OIDC core. A useful plugin boundary must cover four planes:

- Static configuration and secrets.
- Browser routes and runtime lifecycle.
- Optional Goja policy.
- Health, metrics, logs, and audit.

The current code has most primitives, but no object tying these planes
together.

## Proposed Solution

### 1. Definition phase: configuration exists before parsing

Glazed needs to know every section and flag before it can resolve profiles,
config files, environment variables, arguments, and flags. Plugin definitions
must therefore be registered when `serve-production` is constructed.

```go
type Descriptor struct {
    ID          string
    APIVersion  uint32
    RoutePrefix string
}

type Definition interface {
    Descriptor() Descriptor
    Section() (schema.Section, error)
    Prepare(ctx context.Context, vals *values.Values) (Prepared, error)
}

type Prepared interface {
    Build(ctx context.Context, host HostServices) (Runtime, error)
}

type Runtime interface {
    Handler() http.Handler
    Readiness(ctx context.Context) idp.ReadinessCheck
    Close(ctx context.Context) error
}
```

`Prepare` lets each implementation decode into its real typed settings without
putting `map[string]any` into the runtime API. All enabled plugins are prepared
and validated before the public listener opens.

### 2. Glazed configuration composition

The current reusable OIDC section and `ProfileMiddlewaresFunc` already establish
the correct precedence:

```text
defaults < profiles < config < environment < arguments < flags
```

A Jitsi plugin section should use:

```go
schema.NewSection(
    "plugin-jitsi",
    "Jitsi integration",
    schema.WithPrefix("jitsi-"),
    schema.WithFields(...),
)
```

That gives one typed setting three useful forms:

```yaml
plugin-jitsi:
  enabled: true
  public-origin: https://meet.example.test
  xmpp-domain: meet.example.test
  app-id: tinyidp-jitsi
  token-ttl: 5m
  shared-secret-file: /run/secrets/jitsi-token
  policy-program-file: /etc/tinyidp/jitsi-policy.js
```

```text
--jitsi-public-origin https://meet.example.test
TINYIDP_JITSI_PUBLIC_ORIGIN=https://meet.example.test
```

Plugins should never call `os.Getenv`. The host resolves all sources once and
passes `values.Values` to `Prepare`.

The immediate prerequisite is that `serve-production` must be built with the
same explicit parser configuration as `serve-dev` and `print-config`. It
currently uses a bare `cli.BuildCobraCommand`, so it does not compose the
existing profile/config/environment middleware consistently.

### 3. Secrets remain file-backed

Glazed should contain the secret reference, not the secret contents. This
matches the existing production options such as `--token-secret-file`.

For the first Jitsi deployment:

```text
Vault value
  +-- mounted read-only into TinyIDP
  +-- mounted read-only into Prosody
```

TinyIDP signs with the Jitsi-only HMAC secret. Prosody validates with the same
secret. Jitsi web never receives it. The TinyIDP OIDC signing key and cookie
secret are not reused.

A host-owned `SecretResolver` should enforce regular-file, ownership, mode, and
size requirements and return bytes only during runtime construction.

### 4. Routes are scoped

Each plugin receives exactly one immutable prefix:

```text
/integrations/jitsi/
```

The host wraps the handler with request-size limits, security headers, request
IDs, panic recovery, structured logging, and tracing. A plugin cannot register
`/authorize`, `/token`, `/.well-known`, `/metrics`, or another plugin's route.

### 5. Host services are capabilities, not internals

Candidate services:

```go
type HostServices struct {
    Identity      BrowserIdentityService
    Continuations BrowserContinuationService
    Secrets       SecretResolver
    Audit         idp.Sink
    Clock         Clock
    Random        RandomSource
    Observability Observability
}
```

Notably absent:

- The raw store.
- The embedded Fosite provider.
- Cookie and CSRF keys.
- The OIDC private signing key.
- An unrestricted router.

If a plugin needs more authority, the API gains a typed service after review.

### 6. Goja is a policy hook

For Jitsi, JavaScript should run only after Go has normalized the room and
resolved an authenticated TinyIDP identity, but before token issuance:

```text
validate request
  -> normalize room/tenant
  -> require browser identity
  -> invoke integration.jitsi.authorize@v1
  -> validate policy result
  -> construct and sign Jitsi JWT in Go
  -> audit
  -> redirect to fixed Jitsi origin
```

Possible input:

```json
{
  "room": "engineering",
  "tenant": "",
  "identity": {
    "subject": "user-123",
    "displayName": "Manuel",
    "emailVerified": true,
    "roles": ["meeting-organizer"]
  }
}
```

Possible result:

```json
{
  "kind": "complete",
  "claims": {
    "displayName": "Manuel",
    "moderator": true
  }
}
```

This is not a raw JWT claim map. Go still owns `iss`, `aud`, `sub`, `room`,
`iat`, `exp`, algorithm, key selection, and destination. JavaScript never gets
a generic signer.

If policy needs durable application data, the host binds a bounded versioned
capability such as `meeting.membership.lookup@v1`, following the existing
`pkg/idpscript` model.

### 7. Operations are part of the runtime contract

An enabled plugin contributes readiness. It is ready when its configuration,
secret, and optional warmed Goja generation are usable.

Logging should use stable low-cardinality fields:

```text
plugin_id=jitsi
operation=issue_token
result=accepted|rejected|failed
reason=<stable-code>
request_id=<opaque-id>
duration_ms=<number>
```

Tokens, secrets, authorization codes, cookies, continuations, email addresses,
and raw subjects must not appear in logs.

Security decisions additionally use the durable `idp.Sink` audit channel.
Logs and audit are intentionally different.

Metrics options:

1. Extend the current atomic snapshot approach and write a central Prometheus
   collector.
2. Pass a Prometheus registerer to plugins.
3. Pass host-created OpenTelemetry meters/tracers and choose the exporter in
   the host.

The third option is the cleanest long-term API. A Prometheus exporter can still
serve Kubernetes scraping. Metrics must not use users, rooms, emails, tenants,
request IDs, or arbitrary error strings as labels.

A separate internal administrative listener for readiness and `/metrics` is
preferable to placing metrics on the public issuer origin.

## Design Decisions

These are draft recommendations:

- Start with compiled-in, first-party plugins.
- Register schemas before Glazed parsing.
- Move `serve-production` onto the established Glazed source chain.
- Pass secret file references through configuration.
- Scope every plugin below `/integrations/<id>/`.
- Give plugins narrow services instead of raw provider internals.
- Keep Goja typed, bounded, and policy-only.
- Keep token construction, signing, and redirects in Go.
- Compose readiness and graceful shutdown centrally.
- Use structured logs, durable audit, and low-cardinality metrics.

## Alternatives Considered

| Model | Good part | Main cost | Draft position |
|---|---|---|---|
| Compiled-in Go registry | Best Glazed and type integration | Requires rebuilding TinyIDP | Recommended first |
| Go `plugin` shared object | Runtime loading | Fragile toolchain/build compatibility and process safety | Reject |
| HashiCorp subprocess plugin | Isolation and protocol versioning | RPC, manifests, binaries, lifecycle | Defer |
| JavaScript-only plugin | Very flexible policy | Wrong boundary for HTTP and key authority | Policy only |
| Separate adapter service | Strong isolation and standard OIDC | Extra deployment per integration | Keep as fallback |

Go's own `plugin` documentation warns about portability, race detection,
toolchain alignment, initialization, security, and deployment limitations.
HashiCorp's subprocess model is mature, but it requires a discovery/handshake
before Glazed parsing and is unjustified until third-party plugins are a real
requirement.

### Browser identity options

This is the largest unresolved seam.

**Native service:** extract a narrow browser-session/login/continuation service
from the Fosite adapter and expose it to plugins. This produces the cleanest
user flow but requires careful CSRF, fresh-login, account chooser, and
continuation design.

**Embedded OIDC relying party:** make the in-process Jitsi plugin use an
authorization-code/PKCE callback against TinyIDP just like the standalone
adapter. This preserves the public OIDC boundary and is easier to prototype,
but creates a self-referential browser OAuth flow and retains more adapter
machinery.

The full design should decide this first.

## Implementation Plan

No implementation is proposed in this exploratory pass. If the direction is
accepted, the full design should sequence:

1. Browser identity decision.
2. Exact descriptor, prepared/runtime, route, and shutdown interfaces.
3. `serve-production` Glazed section and parser refactor.
4. Redacted configuration inspection with source provenance.
5. Registry, scoped routing, readiness, and lifecycle.
6. Versioned Jitsi Goja policy schema.
7. Jitsi signing and redirect runtime.
8. Administrative listener and metrics exporter.
9. Unit, browser, restart, redaction, wrong-secret, expired-token, and
   room-mismatch tests.

## Open Questions

- Native browser identity or embedded OIDC/PKCE?
- One Jitsi integration per process initially, or named repeated instances?
- Prometheus-native metrics first, or OpenTelemetry from the start?
- Does a plugin runtime failure only fail readiness, or terminate the host?
- Startup-only Goja policies initially, or hot-swappable generations?
- HS256 only in version 1, or public-key Jitsi signing too?

## References

- `cmd/tinyidp/main.go`
- `internal/cmds/profiles.go`
- `internal/cmds/config.go`
- `internal/cmds/serve_production.go`
- `internal/sections/oidc/section.go`
- `pkg/embeddedidp/provider.go`
- `pkg/idpscript/capabilities.go`
- `pkg/idpsignup`
- [Jitsi token authentication](https://jitsi.github.io/handbook/docs/devops-guide/token-authentication/)
- [Jitsi JWT and Prosody contract](https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md)
- [Jitsi OIDC adapter](https://github.com/jitsi-contrib/jitsi-oidc-adapter)
- [Go shared-plugin warnings](https://pkg.go.dev/plugin)
- [HashiCorp subprocess plugin model](https://github.com/hashicorp/go-plugin)
