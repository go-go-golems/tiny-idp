---
Title: TinyIDP plugin system and Jitsi integration analysis design and implementation guide
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
      Note: Glazed command construction that must compose the plugin registry
    - Path: repo://internal/cmds/profiles.go
      Note: Established configuration precedence middleware
    - Path: repo://internal/cmds/serve_production.go
      Note: Production runtime, secret loading, HTTP composition, and lifecycle integration point
    - Path: repo://internal/fositeadapter/session.go
      Note: Sensitive browser-session implementation deliberately kept outside the plugin API
    - Path: repo://pkg/embeddedidp/provider.go
      Note: Provider handler used by the proposed in-process OIDC broker and readiness composition
    - Path: repo://pkg/idpscript/capabilities.go
      Note: Bounded versioned Goja capability model
ExternalSources:
    - https://jitsi.github.io/handbook/docs/devops-guide/token-authentication/
    - https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md
    - https://github.com/jitsi-contrib/jitsi-oidc-adapter
    - https://pkg.go.dev/plugin
    - https://github.com/hashicorp/go-plugin
Summary: Intern-facing system guide for a compiled-in TinyIDP plugin API, Glazed configuration, OIDC browser brokerage, bounded Goja policy, Jitsi JWT issuance, observability, testing, and Kubernetes deployment.
LastUpdated: 2026-07-23T17:05:00-04:00
WhatFor: Use this guide to understand and implement the TinyIDP plugin host and its first Jitsi integration.
WhenToUse: Read before changing production command configuration, adding integration routes, exposing identity to plugins, adding Jitsi signing, or implementing plugin observability.
---


# TinyIDP plugin system and Jitsi integration analysis design and implementation guide

## Executive Summary

TinyIDP needs an extension boundary for application-specific authentication integrations that do not belong in its OIDC protocol core. Jitsi is the first concrete case. Jitsi can authenticate users through TinyIDP, but Prosody expects a Jitsi-specific JWT rather than a normal OIDC ID token. A bridge must therefore perform an OIDC browser flow, obtain a trusted identity, apply application policy, mint a short-lived room-bound Jitsi JWT, and redirect the browser into the meeting.

This guide specifies a version-one plugin system with these properties:

- Plugins are first-party Go packages compiled into the TinyIDP binary.
- Every plugin contributes a typed Glazed configuration section before command parsing.
- `serve-production` resolves core and plugin values through one documented source precedence chain.
- Secret contents remain in protected files mounted by the deployment.
- Plugins are mounted below fixed `/integrations/<plugin-id>/` route prefixes.
- A host-owned OIDC relying-party broker performs authorization code, PKCE, nonce, callback, token exchange, ID-token validation, and userinfo retrieval.
- Goja can decide application policy through a typed, bounded handler, but it cannot sign tokens or control HTTP.
- The host composes readiness, structured logs, durable audit, metrics, traces, and graceful shutdown.

The Jitsi plugin replaces the separately deployed OIDC-to-Jitsi adapter. It does not replace Prosody. Prosody continues to validate the Jitsi JWT locally, run XMPP signaling, enforce room membership, and coordinate Jitsi components.

The design is deliberately narrow. Version one supports one configured instance of each compiled plugin. It does not load third-party binaries, Go shared objects, arbitrary JavaScript plugins, or repeated dynamic plugin instances.

## 1. System Context

### 1.1 TinyIDP

TinyIDP is the identity provider. It owns:

- User accounts and password authentication.
- Signup, invitation, and email-verification workflows.
- Browser sessions and account selection.
- OAuth 2.0 and OpenID Connect endpoints.
- OIDC client registration and redirect allowlists.
- OIDC signing keys and token issuance.
- Durable protocol state, audit, and retention.

The production entry point is `internal/cmds/serve_production.go`. It loads reviewed catalogs and protected secret files, opens SQLite, compiles the signup program, constructs `embeddedidp.Provider`, starts the HTTP server, schedules maintenance, and performs graceful shutdown.

### 1.2 Jitsi Meet

Jitsi Meet is the browser application. It renders the meeting UI, obtains the Jitsi JWT, and supplies that token when establishing its XMPP connection. The browser does not know the signing secret.

### 1.3 Prosody

Prosody is the XMPP server used by Jitsi. Jitsi's Prosody modules:

- Validate the JWT when the browser connects through WebSocket or BOSH.
- Compare the token's `room` and `sub` claims with the requested conference.
- Propagate identity information from `context.user`.
- Enforce moderator, guest, lobby, and room behavior.
- Provide signaling used by the browser, Jicofo, and other Jitsi components.

Prosody supports shared-secret and public-key validation. Version one uses a Jitsi-only HS256 shared secret because it is the simplest documented contract.

### 1.4 Jicofo and Jitsi Videobridge

Jicofo coordinates conferences. Jitsi Videobridge routes media. They use Prosody and have their own internal XMPP credentials. TinyIDP does not receive those credentials.

### 1.5 Complete request path

```text
Browser
  |
  | 1. GET /integrations/jitsi/start?room=engineering
  v
TinyIDP Jitsi plugin
  |
  | 2. Redirect to TinyIDP /authorize with PKCE, state, and nonce
  v
TinyIDP OIDC provider
  |
  | 3. Login, signup, account selection, and policy
  | 4. Redirect authorization code to plugin callback
  v
Host-owned OIDC relying-party broker
  |
  | 5. Exchange code in-process; validate ID token; fetch userinfo
  v
Jitsi plugin
  |
  | 6. Invoke optional Goja policy
  | 7. Mint short-lived room-bound Jitsi JWT
  | 8. Redirect to https://meet.example/engineering?jwt=...
  v
Jitsi browser application
  |
  | 9. Pass JWT on XMPP WebSocket/BOSH connection
  v
Prosody
  |
  | 10. Validate signature, application, expiry, domain, and room
  v
Jitsi conference
```

TinyIDP and Prosody do not make synchronous requests to each other. Their shared contract consists of the Jitsi application ID, signing material, algorithm, domain, and claim rules.

## 2. Problem and Design Boundaries

TinyIDP currently has two kinds of extensibility.

The first is Go composition. `pkg/embeddedidp.Options` accepts concrete policies, stores, renderers, rate limiters, authenticators, and scripted signup components. This is a strong embedding API, but it is assembled directly by `serve-production`; it is not a plugin system.

The second is Goja workflow scripting. `pkg/idpscript` compiles JavaScript, limits runtime work, binds declared capabilities, and validates typed outcomes. This supports policy and workflow decisions without giving scripts ambient authority. It does not provide HTTP routes, configuration registration, secrets, lifecycle management, or token signing.

Application integrations need both forms without collapsing them together. The plugin supplies trusted Go mechanics. Goja supplies optional application policy. The host supplies identity, configuration, secrets, and operations.

Jitsi demonstrates why these boundaries matter. A normal TinyIDP ID token says that TinyIDP authenticated a subject for an OIDC client. A Jitsi token grants access to a particular Jitsi deployment and often to a particular room. It uses Jitsi-specific claims and is transported to Prosody through the browser's Jitsi connection. Reusing a normal ID token would combine two different audiences and security meanings.

### Requirements

The plugin system must:

- Fail startup before listening if an enabled plugin is misconfigured.
- Use the same Glazed configuration sources and precedence as the host.
- Keep secret bytes out of flags, help, parsed configuration, and configuration inspection.
- Prevent route collisions with OIDC endpoints and other plugins.
- Avoid exposing raw TinyIDP sessions, stores, and private signing keys.
- Preserve normal TinyIDP login, signup, and account-selection behavior.
- Support typed and bounded Goja authorization policy.
- Emit non-secret readiness, logs, audit events, metrics, and traces.
- Close resources deterministically during partial startup failure and normal shutdown.
- Allow the Jitsi bridge to be tested without running an entire Kubernetes cluster.

### Non-goals

Version one does not:

- Replace Prosody.
- Load untrusted or third-party plugin code.
- Use Go's `buildmode=plugin`.
- Start plugin subprocesses.
- Allow JavaScript to register HTTP routes.
- Give JavaScript a generic JWT signing primitive.
- Support arbitrary plugin dependencies or load order.
- Support multiple named instances of the same plugin.
- Hot-install or uninstall plugins in a running process.
- Add a backward-compatibility layer around the production command.

## 3. Package Structure

The first implementation should remain internal while the API is changing:

```text
internal/
  pluginapi/
    descriptor.go
    definition.go
    runtime.go
    services.go
    registry.go
  pluginhost/
    prepare.go
    build.go
    routing.go
    readiness.go
    lifecycle.go
    oidcbroker/
      broker.go
      transaction.go
      handler_transport.go
  plugins/
    jitsi/
      definition.go
      section.go
      settings.go
      runtime.go
      handler.go
      token.go
      policy.go
      metrics.go
  sections/
    production/
      section.go
      settings.go
```

Keeping the API in `internal/` avoids promising third-party compatibility before the Jitsi implementation proves the abstractions. If a later project needs external first-party modules in other repositories, the stable subset can be promoted to `pkg/pluginapi`.

## 4. The Plugin API

### 4.1 Descriptor

The descriptor is static metadata. It is available before parsing configuration.

```go
package pluginapi

type Descriptor struct {
    ID         string
    APIVersion uint32
    Summary    string
}

func (d Descriptor) Validate() error
func (d Descriptor) RoutePrefix() string
```

`ID` must contain lowercase ASCII letters, digits, and hyphens. The host derives the route prefix as `/integrations/<id>/`; plugins do not provide arbitrary paths.

`APIVersion` versions the Go host contract. Goja handler schemas are versioned separately.

### 4.2 Definition

A definition contributes configuration and converts parsed values into a validated prepared object.

```go
type Definition interface {
    Descriptor() Descriptor
    Section() (schema.Section, error)
    Prepare(
        ctx context.Context,
        vals *values.Values,
    ) (Prepared, error)
}
```

The definition must be deterministic and side-effect free except for bounded reads of reviewed non-secret program files. It must not start goroutines, open listeners, modify the database, or read secret contents.

### 4.3 Prepared plugin

Preparation separates configuration validation from runtime construction.

```go
type Requirements struct {
    OIDCClients []OIDCClientRequirement
}

type Prepared interface {
    Descriptor() Descriptor
    Enabled() bool
    Requirements() Requirements
    Build(
        ctx context.Context,
        services RuntimeServices,
    ) (Runtime, error)
}
```

`Requirements` lets the host verify that the reviewed TinyIDP client catalog contains the plugin's internal public client before constructing the provider. The Jitsi requirement specifies:

- A fixed client ID.
- Public-client authentication.
- Authorization code grant.
- PKCE S256.
- Exact callback URI under the plugin route prefix.
- Only the required `openid`, `profile`, and optional `email` scopes.
- No device grant or refresh token unless separately justified.

The plugin does not silently create a privileged client. The operator's reviewed client catalog remains authoritative.

### 4.4 Runtime

The runtime owns the handler, policy executor, signer, instruments, and loaded secret bytes.

```go
type Runtime interface {
    Descriptor() Descriptor
    Handler() http.Handler
    Readiness(ctx context.Context) idp.ReadinessCheck
    Close(ctx context.Context) error
}
```

Version one does not include a background `Run` method because the Jitsi bridge does not require background work. If a later plugin requires workers, that need should produce an API-versioned extension rather than an unused method in every first implementation.

Every implementation uses a compile-time assertion:

```go
var _ pluginapi.Runtime = (*Runtime)(nil)
```

### 4.5 Runtime services

```go
type RuntimeServices struct {
    OIDC    RelyingPartyBroker
    Secrets SecretResolver
    Audit   idp.Sink
    Logger  zerolog.Logger
    Meter   metric.Meter
    Tracer  trace.Tracer
    Clock   Clock
    Random  io.Reader
}
```

The service set is intentionally small. The plugin does not receive:

- `idpstore.Store`.
- `embeddedidp.Provider`.
- Fosite request objects.
- Browser cookie or CSRF keys.
- TinyIDP's OIDC private key.
- A root `http.ServeMux`.

This restriction makes authority visible in API review.

## 5. Glazed Configuration

### 5.1 Why plugins contribute sections

Glazed resolves typed fields only after a command schema exists. If a plugin tries to discover its configuration after command parsing, its flags and environment mappings cannot participate in the normal source chain.

Each compiled definition therefore contributes one section during `NewServeProductionCommand`.

```go
func NewSection() (schema.Section, error) {
    return schema.NewSection(
        "plugin-jitsi",
        "Jitsi integration",
        schema.WithPrefix("jitsi-"),
        schema.WithFields(
            fields.New("enabled", fields.TypeBool,
                fields.WithDefault(false)),
            fields.New("public-origin", fields.TypeString),
            fields.New("xmpp-domain", fields.TypeString),
            fields.New("app-id", fields.TypeString),
            fields.New("oidc-client-id", fields.TypeString),
            fields.New("token-ttl", fields.TypeString,
                fields.WithDefault("5m")),
            fields.New("shared-secret-file", fields.TypeString),
            fields.New("policy-program-file", fields.TypeString),
        ),
    )
}
```

The section slug organizes config and profiles. The section prefix organizes flags and environment keys.

| Source | Example |
|---|---|
| Default | `enabled=false`, `token-ttl=5m` |
| Profile | `plugin-jitsi.public-origin` inside the selected profile |
| Config file | `plugin-jitsi.public-origin` |
| Environment | `TINYIDP_JITSI_PUBLIC_ORIGIN` |
| Flag | `--jitsi-public-origin` |

The precedence remains:

```text
defaults < profiles < config < environment < arguments < flags
```

Plugins never call `os.Getenv`. The host resolves all sources once and passes `values.Values` to `Prepare`.

### 5.2 Required production-command refactor

`cmd/tinyidp/main.go` currently builds `serve-dev` and `print-config` with `ProfileMiddlewaresFunc`, `ConfigFilePlanBuilder`, `AppName: "tinyidp"`, and the profile settings section. `serve-production` is built with a bare `cli.BuildCobraCommand`.

The production command must use the same explicit parser configuration:

```go
productionCmd, err := cmds.NewServeProductionCommand(registry)
if err != nil {
    return err
}

productionCobraCmd, err := cli.BuildCobraCommand(
    productionCmd,
    cli.WithParserConfig(cli.CobraParserConfig{
        AppName:           "tinyidp",
        ConfigPlanBuilder: cmds.ConfigFilePlanBuilder,
        MiddlewaresFunc: cmds.ProfileMiddlewaresFunc(
            "tinyidp",
            cmds.ConfigFilePlanBuilder,
        ),
    }),
    cli.WithProfileSettingsSection(),
)
```

The flat production flags should move into a reusable `internal/sections/production` section. This makes production configuration available to a redacted `print-config` or future `config check` command without copying definitions.

No alternate loader or compatibility adapter should be added.

### 5.3 Configuration example

```yaml
production:
  addr: :8443
  listener-mode: trusted-proxy-http
  issuer: https://idp.example.test
  clients-file: /etc/tinyidp/clients.json
  theme-dir: /etc/tinyidp/themes
  theme-catalog-file: /etc/tinyidp/themes/themes.json
  signup-program-file: /etc/tinyidp/signup.js
  db: /var/lib/tinyidp/idp.db
  audit-path: /var/log/tinyidp/audit.jsonl
  token-secret-file: /run/secrets/tinyidp-token
  trusted-proxy-cidrs:
    - 10.42.0.0/16

plugin-jitsi:
  enabled: true
  public-origin: https://meet.example.test
  xmpp-domain: meet.example.test
  app-id: tinyidp-jitsi
  oidc-client-id: tinyidp-plugin-jitsi
  token-ttl: 5m
  shared-secret-file: /run/secrets/jitsi-token
  policy-program-file: /etc/tinyidp/jitsi-policy.js
```

### 5.4 Secret rules

Configuration carries only file paths. `SecretResolver` applies the common production rules already represented by `readOwnerOnlySecret`:

- The path is non-empty.
- The target is a regular file.
- Symbolic-link policy is explicit and tested.
- Ownership and mode are acceptable.
- Size is bounded.
- Content has a minimum length.
- Temporary byte slices are cleared after failed construction.

Raw secrets must not be supported as flags or environment variables.

### 5.5 Redacted inspection

An operator needs to see the final resolved values and their sources before starting the server. A production-aware configuration inspection command should:

- Compose the same core and plugin sections.
- Resolve the same precedence chain.
- Show whether each plugin is enabled.
- Show non-secret values and secret file paths.
- Never open or print secret contents.
- Show the winning source recorded by Glazed.
- Run preparation validation without starting services.

## 6. Registry and Host Construction

### 6.1 Registry validation

```go
registry, err := pluginapi.NewRegistry(
    jitsi.NewDefinition(),
)
```

`NewRegistry` rejects:

- Duplicate IDs.
- Unsupported API versions.
- Invalid IDs.
- Duplicate section slugs.
- Duplicate field prefixes.
- Derived route collisions.

The registry is immutable after command construction.

### 6.2 Construction sequence

The host validates as much as possible before opening durable or network resources.

```text
construct registry
    |
compose production and plugin sections
    |
parse Glazed sources
    |
decode production settings
    |
prepare every plugin
    |
load client and theme catalogs
    |
validate plugin OIDC client requirements
    |
load core secrets and programs
    |
open SQLite and audit
    |
bootstrap reviewed clients and signing keys
    |
construct embedded provider
    |
construct in-process OIDC broker
    |
build plugin runtimes and load plugin secrets
    |
compose public and administrative handlers
    |
run initial maintenance
    |
start listeners
```

If runtime construction fails, already built runtimes are closed in reverse order. The store, audit sink, script managers, and provider follow the same explicit cleanup path. Server and maintenance goroutines remain under the existing `errgroup`.

## 7. Browser Identity Through an OIDC Broker

### 7.1 Why version one uses OIDC internally

The preliminary research considered exposing TinyIDP's browser session directly to plugins. That would require extracting session-cookie reading, fresh-login semantics, account chooser, CSRF, signup continuation, and error rendering from `internal/fositeadapter`. It would also allow every plugin to depend on sensitive session internals.

Version one instead treats each integration plugin as a constrained OIDC relying party. This preserves the identity-provider boundary:

- TinyIDP owns login and signup.
- The plugin requests identity through standard authorization.
- The plugin receives only validated identity claims.
- The same bridge can later move to a separate process without changing its identity semantics.

The host provides an in-process transport for the server-side token and userinfo calls. This avoids requiring a pod to reach its own public ingress.

### 7.2 Broker API

```go
type BeginRequest struct {
    PluginID     string
    ClientID     string
    CallbackPath string
    Scopes       []string
    State        json.RawMessage
    TTL          time.Duration
}

type Identity struct {
    Subject           string
    DisplayName       string
    PreferredUsername string
    Email             string
    EmailVerified     bool
    Roles             []string
    Groups            []string
    AuthTime          time.Time
}

type Completion struct {
    Identity Identity
    State    json.RawMessage
}

type RelyingPartyBroker interface {
    Begin(
        w http.ResponseWriter,
        r *http.Request,
        request BeginRequest,
    ) error

    Complete(
        w http.ResponseWriter,
        r *http.Request,
        pluginID string,
    ) (Completion, error)
}
```

The broker, not the plugin, owns:

- Cryptographically random state, nonce, and PKCE verifier.
- Durable pending-transaction storage.
- Exact callback construction.
- Authorization request construction.
- Code exchange.
- ID-token signature, issuer, audience, expiry, and nonce validation.
- Userinfo retrieval.
- One-time transaction consumption.
- Browser binding and expiration.
- OAuth error normalization.

The plugin provides a small bounded state document, such as normalized room and tenant. It never stores access tokens or authorization codes.

### 7.3 In-process HTTP transport

The browser must use the public issuer URL for `/authorize`. The server-side code exchange should not traverse DNS, public ingress, and Traefik only to return to the same process.

An internal `http.RoundTripper` dispatches requests to `embeddedidp.Provider.Handler()`:

```go
type HandlerTransport struct {
    Handler http.Handler
    Issuer  *url.URL
}

func (t *HandlerTransport) RoundTrip(
    request *http.Request,
) (*http.Response, error) {
    // Validate that request origin equals the configured issuer.
    // Clone the request and dispatch it to the provider handler.
    // Capture status, headers, and body as an http.Response.
}
```

Only the configured issuer origin and protocol endpoints are accepted. Redirect following is disabled for server-side calls. Request and response bodies are bounded.

This transport preserves HTTP-level token endpoint behavior without creating a cluster network dependency.

### 7.4 Durable broker transactions

A pending transaction contains:

```go
type Transaction struct {
    StateHash       []byte
    PluginID        string
    ClientID        string
    CallbackPath    string
    NonceHash       []byte
    PKCEVerifierBox []byte
    PluginStateBox  []byte
    BrowserBinding  []byte
    CreatedAt       time.Time
    ExpiresAt       time.Time
    ConsumedAt      *time.Time
}
```

The raw state is returned to the browser; only a keyed hash is used for lookup. PKCE verifier and plugin state are encrypted at rest with a domain-separated key derived inside the host from protected runtime key material. The plugin does not receive that key.

Consumption is atomic and one-time. Expired, mismatched, already consumed, or wrong-plugin transactions fail closed and emit stable audit reasons.

## 8. Jitsi Plugin

### 8.1 Configuration validation

The Jitsi definition validates:

- `public-origin` is an HTTPS origin with no userinfo, query, or fragment.
- `xmpp-domain` is normalized and matches the expected Jitsi domain policy.
- `app-id` is non-empty and bounded.
- `oidc-client-id` names a public PKCE client in the reviewed catalog.
- `token-ttl` is positive and no greater than the compiled maximum.
- `shared-secret-file` is present when enabled.
- `policy-program-file`, when present, is a bounded regular file that compiles against the Jitsi policy schemas.

Redirect paths are derived, not configured:

```text
/integrations/jitsi/start
/integrations/jitsi/callback
```

### 8.2 Start handler

```go
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
    requireMethod(GET)

    room := normalizeAndValidateRoom(
        r.URL.Query().Get("room"),
    )
    tenant := normalizeAndValidateTenant(
        r.URL.Query().Get("tenant"),
    )
    state := encodeBoundedState(room, tenant)

    err := h.oidc.Begin(w, r, BeginRequest{
        PluginID:     "jitsi",
        ClientID:     h.settings.OIDCClientID,
        CallbackPath: "/integrations/jitsi/callback",
        Scopes:       []string{"openid", "profile", "email"},
        State:        state,
        TTL:          10 * time.Minute,
    })
    if err != nil {
        h.renderSafeError(
            w, r, "authentication_start_failed",
        )
    }
}
```

The input parser rejects duplicate parameters, oversized values, control characters, encoded separators, and room names that cannot be represented consistently in Jitsi and Prosody.

### 8.3 Callback handler

```go
func (h *Handler) Callback(
    w http.ResponseWriter,
    r *http.Request,
) {
    completion, err := h.oidc.Complete(
        w, r, "jitsi",
    )
    if err != nil {
        h.renderSafeError(
            w, r, classifyOIDCError(err),
        )
        return
    }

    state := decodeAndValidateState(completion.State)
    decision, err := h.policy.Authorize(
        r.Context(),
        completion.Identity,
        state.Room,
        state.Tenant,
    )
    if err != nil || !decision.Allowed {
        h.auditDecision(...)
        h.renderSafeError(
            w, r, "meeting_access_denied",
        )
        return
    }

    token, err := h.signer.Issue(IssueRequest{
        Identity: completion.Identity,
        Room:     state.Room,
        Tenant:   state.Tenant,
        Decision: decision,
    })
    if err != nil {
        h.renderSafeError(
            w, r, "token_issue_failed",
        )
        return
    }

    target := h.redirectURL(
        state.Room, state.Tenant, token,
    )
    http.Redirect(
        w, r, target, http.StatusSeeOther,
    )
}
```

### 8.4 Jitsi JWT

The Go signer owns every security-sensitive field:

```json
{
  "iss": "tinyidp-jitsi",
  "aud": "tinyidp-jitsi",
  "sub": "meet.example.test",
  "room": "engineering",
  "iat": 1784850000,
  "nbf": 1784850000,
  "exp": 1784850300,
  "context": {
    "user": {
      "id": "user-123",
      "name": "Manuel",
      "email": "wesen@example.test",
      "moderator": true
    }
  }
}
```

The signer enforces:

- Fixed HS256 algorithm in version one.
- Fixed application ID as issuer and audience.
- Configured domain or normalized tenant as subject.
- Exact normalized room; wildcard room issuance is not exposed.
- Short lifetime and bounded clock skew.
- No caller-supplied timestamps.
- A stable subject identifier.
- Filtered scalar `context.user` values accepted by Jitsi.
- No empty or null identity values that break Jitsi consumers.

The normal TinyIDP OIDC signing key is never used.

### 8.5 Prosody configuration

Prosody uses matching values:

```lua
VirtualHost "meet.example.test"
    authentication = "token"
    app_id = "tinyidp-jitsi"
    app_secret = "<same mounted Jitsi secret>"
    allow_empty_token = false

Component "conference.meet.example.test" "muc"
    modules_enabled = {
        "token_verification";
    }
```

If identity should appear in participant presence, enable the supported identity module documented by Jitsi. Guest and wait-for-host behavior is a separate Prosody/Jitsi policy decision and does not alter the TinyIDP plugin API.

## 9. Goja Policy

### 9.1 Contract

The plugin defines a versioned handler:

```text
integration.jitsi.authorize@v1
```

Input schema:

```typescript
interface JitsiAuthorizeInput {
  integrationId: string;
  room: string;
  tenant: string;
  identity: {
    subject: string;
    displayName: string;
    preferredUsername: string;
    email: string;
    emailVerified: boolean;
    roles: string[];
    groups: string[];
    authTime: string;
  };
}
```

Output schema:

```typescript
type JitsiAuthorizeResult =
  | {
      kind: "complete";
      claims: {
        displayName: string;
        includeEmail: boolean;
        moderator: boolean;
      };
    }
  | {
      kind: "deny";
      diagnosticId: string;
    };
```

The diagnostic ID comes from an allowlist of stable public-safe codes. The script cannot return arbitrary browser error text.

### 9.2 Example

```javascript
const A = require("tinyidp").v1;

module.exports = A.program("jitsi-policy", p => {
  p.lambda("integration.jitsi.authorize", {
    input: "integration.jitsi.authorize.input.v1",
    output: "integration.jitsi.authorize.output.v1",
    budget: {
      timeoutMs: 50,
      maxCapabilityCalls: 1,
      maxOutputBytes: 4096,
    },
    async run(ctx) {
      const i = ctx.input;

      if (!i.identity.emailVerified) {
        return {
          kind: "deny",
          diagnosticId: "verified_email_required",
        };
      }

      return {
        kind: "complete",
        claims: {
          displayName: i.identity.displayName,
          includeEmail: true,
          moderator:
            i.identity.roles.includes(
              "meeting-organizer",
            ),
        },
      };
    },
  });
});
```

### 9.3 Capabilities

If durable application membership is required later:

```javascript
const membership =
  await ctx.capabilities.meeting.membership.lookup({
    subject: ctx.input.identity.subject,
    room: ctx.input.room,
  });
```

The Go binding validates its input, limits calls and bytes, applies a context deadline, and returns a typed result. The script never receives the database.

### 9.4 Program lifecycle

The Jitsi plugin reuses the proven patterns in `pkg/idpsignup`:

- Compile and validate before listening.
- Warm a bounded worker pool.
- Expose pool readiness.
- Fingerprint executable generations.
- Count invocations, failures, interruptions, outcome kinds, and latency.
- Close the pool on shutdown.

Version one loads the policy at startup. Hot generation replacement is deferred until the baseline flow is proven.

## 10. HTTP Composition and Security

`productionHTTPHandler` currently mounts theme assets and delegates all other paths to the provider handler. It should be extended deliberately:

```go
func productionHTTPHandler(
    provider http.Handler,
    assets http.Handler,
    plugins []pluginapi.Runtime,
    limits Limits,
) http.Handler {
    mux := http.NewServeMux()
    mux.Handle("/static/themes/", assets)

    for _, plugin := range plugins {
        prefix :=
            plugin.Descriptor().RoutePrefix()
        mux.Handle(
            prefix,
            http.StripPrefix(
                prefix,
                plugin.Handler(),
            ),
        )
    }

    mux.Handle("/", provider)

    return requestLimits(
        securityHeaders(
            requestID(
                recoverPanics(mux),
            ),
        ),
        limits,
    )
}
```

Common security middleware should be extracted so plugin and provider routes receive consistent headers. Plugin browser errors use the production renderer, not `http.Error`, while protocol endpoints retain their required OAuth error formats.

Redirect construction uses the configured Jitsi origin and a normalized room path. Request data can select only the room and tenant components that pass validation; it cannot replace the scheme, authority, or base path.

## 11. Observability

### 11.1 Structured logs

Each runtime receives a logger already scoped with:

```text
component=tinyidp.plugin
plugin_id=jitsi
```

Operations add:

- `operation`.
- `result`.
- A stable `reason`.
- `request_id`.
- Duration.

Ordinary logs exclude tokens, codes, secrets, cookie values, continuation handles, raw subjects, email addresses, and unrestricted room names.

### 11.2 Durable audit

Recommended Jitsi audit events:

```text
integration.jitsi.authentication_started
integration.jitsi.authentication_completed
integration.jitsi.policy_denied
integration.jitsi.token_issued
integration.jitsi.request_rejected
```

Audit fields use stable values such as policy version, result, and reason. Whether room identifiers belong in durable audit must be an explicit privacy decision; they must not become metric labels.

If token issuance commits but audit delivery fails, the operation follows TinyIDP's existing `idp.ErrAuditDelivery` semantics. The safe ordering is to complete required audit delivery before emitting the redirect containing the token.

### 11.3 Metrics

Recommended instruments:

```text
tinyidp.plugin.requests
tinyidp.plugin.request.duration
tinyidp.jitsi.tokens.issued
tinyidp.jitsi.policy.invocations
tinyidp.jitsi.policy.duration
tinyidp.jitsi.oidc.transactions
```

Allowed attributes:

```text
plugin=jitsi
operation=start|callback|issue_token
outcome=accepted|denied|failed
reason_class=validation|oauth|policy|signing|audit
```

Users, rooms, tenants, request IDs, and raw error text are prohibited metric attributes.

The host creates OpenTelemetry meters and tracers and exposes Prometheus format from an exporter. This keeps the plugin API exporter-neutral.

### 11.4 Administrative listener

Add a separate internal listener:

```yaml
production:
  admin-addr: 127.0.0.1:9090
```

It serves:

```text
/healthz
/readyz
/metrics
```

Kubernetes probes and monitoring use an internal Service or pod port. Public Traefik ingress does not expose it.

Readiness aggregates core checks and one stable check per enabled plugin. Prosody reachability is not part of TinyIDP readiness because token validation is local to Prosody. A separate synthetic test validates the complete meeting join.

## 12. Kubernetes Deployment

```text
Namespace: jitsi

TinyIDP Deployment
  - core ConfigMap
  - Jitsi plugin config
  - signup.js
  - jitsi-policy.js
  - SQLite PVC
  - audit volume
  - TinyIDP core secrets
  - Jitsi HMAC secret mount

Jitsi Web Deployment

Prosody Deployment
  - matching app_id
  - matching Jitsi HMAC secret mount

Jicofo Deployment

JVB Deployment
  - public UDP/10000 media exposure
```

Vault Secrets Operator materializes the Jitsi signing value into the TinyIDP and Prosody Kubernetes secrets. The mounts are read-only. Rotation requires a coordinated rollout because HS256 does not identify multiple verification keys with `kid` in the simple shared-secret configuration.

The plugin code remains in the TinyIDP image. Non-secret configuration and reviewed JavaScript are GitOps-managed. Secret values remain in Vault and do not enter Git.

Resource limits must account for TinyIDP request concurrency, the Jitsi policy Goja pool, SQLite and audit I/O, and telemetry. JVB media capacity is independent and usually dominates Jitsi resource planning.

## 13. Failure Model

| Failure | Required behavior |
|---|---|
| Plugin config invalid | Process exits before listening |
| Required OIDC client missing | Process exits before provider construction completes |
| Jitsi secret unreadable | Process exits before listening |
| Goja program invalid | Process exits before listening |
| Goja pool saturated | Request gets themed retryable error; metric and audit emitted |
| OIDC state, nonce, or PKCE mismatch | Callback rejected; transaction revoked |
| OIDC transaction expired | Themed restart-login response |
| Policy denies | No Jitsi token is created |
| Signing fails | No redirect; audit and metric emitted |
| Audit delivery fails | No token-bearing redirect |
| Prosody has wrong secret | TinyIDP stays ready; end-to-end join test fails |
| Runtime closed | Plugin readiness fails and handler returns unavailable |

Failures shown to browsers use stable, non-sensitive explanations. Detailed errors remain in structured logs without secret material.

## 14. Testing Strategy

### 14.1 Configuration tests

Test every source independently and in precedence combinations:

```text
defaults
profiles
config
environment
flags
flag > environment > config > profile > default
```

Verify section prefix behavior, typed lists and booleans, invalid durations, unknown or missing plugin clients, redacted inspection, and absence of secret bytes in parsed values.

### 14.2 Registry tests

Test duplicate IDs, invalid IDs, unsupported API versions, duplicate prefixes, route collisions, disabled plugin behavior, preparation ordering, reverse cleanup, and partial build failure.

### 14.3 OIDC broker tests

Use the real embedded provider handler to test:

- Authorization code with PKCE S256.
- State and nonce validation.
- Exact callback binding.
- Public client behavior.
- Login, signup, and existing-session paths.
- Expired and replayed transactions.
- Browser binding mismatch.
- OAuth denial.
- Restart between authorization and callback.
- Bounded token and userinfo responses.

### 14.4 Goja tests

Test allow, deny, malformed output, timeout, saturation, capability failure, excess calls, oversized input/output, interruption, and deterministic declarative program tests.

### 14.5 Jitsi token tests

Parse and verify issued tokens independently:

- Correct HS256 signature.
- Correct issuer and audience.
- Exact domain and room.
- Short expiry.
- No wildcard room.
- Correct moderator mapping.
- Email omitted when policy requests omission.
- Invalid or null context values rejected.
- Token bytes absent from logs and audit.

Where practical, execute Prosody's token validation in a container test with a valid token, wrong secret, expired token, wrong application ID, wrong domain, and wrong room.

### 14.6 Browser tests

Playwright should cover:

- Anonymous meeting start, login, and room entry.
- Anonymous meeting start and new account signup.
- Existing TinyIDP session skipping password entry.
- Account chooser.
- Policy denial.
- Unverified-email denial.
- OIDC cancellation.
- Expired callback.
- Malformed room.
- Themed error rendering.
- Logout followed by a new meeting login.

The final production smoke test must confirm actual media connectivity, not only page navigation.

## 15. Design Decisions

The design makes these decisions:

- Version one uses compiled-in first-party Go plugins.
- The API remains under `internal/` until proven.
- Plugins contribute static Glazed sections.
- `serve-production` adopts the existing Glazed profile/config/environment chain.
- Secret values are file-backed and resolved by the host.
- Each plugin owns a derived scoped route prefix.
- Plugins use a host-owned OIDC relying-party broker instead of browser-session internals.
- The broker uses authorization code, PKCE S256, nonce, durable one-time state, and an in-process provider transport.
- The reviewed client catalog remains authoritative.
- Goja is optional typed policy, not mechanics or signing.
- Jitsi version one uses a separate HS256 secret and exact room tokens.
- Metrics and traces use OpenTelemetry abstractions; Prometheus is an exporter.
- Metrics, readiness, and probes use an internal administrative listener.
- Audit delivery precedes token-bearing redirects.

## 16. Alternatives Considered

### Dynamic Go shared objects

Rejected. Go documents strict toolchain, build-tag, and dependency alignment requirements, weak race-detector support, portability limitations, difficult container packaging, and process-wide crash and security exposure.

### HashiCorp subprocess plugins

Deferred. The subprocess and RPC model provides isolation and protocol versioning, but it requires binary discovery, integrity verification, startup handshakes, RPC schemas, log forwarding, and failure supervision. Those costs are justified only when third-party or separately released plugins exist.

### JavaScript-only plugins

Rejected as the mechanics layer. JavaScript remains valuable for policy, but HTTP parsing, OAuth transactions, secrets, token signing, redirects, and lifecycle remain in reviewed Go.

### Direct access to TinyIDP browser sessions

Rejected for version one. It would expose sensitive internals and require a new generic login and signup continuation API. The OIDC broker reuses the existing identity boundary.

### Network loopback to the public issuer

Rejected. It introduces DNS, ingress, TLS, proxy, and hairpin dependencies inside one process. The in-process RoundTripper preserves HTTP semantics without those deployment dependencies.

### Reusing TinyIDP ID tokens as Jitsi tokens

Rejected. OIDC and Jitsi tokens have different issuer, audience, claims, lifetimes, transports, and authorization meanings.

### Prosody introspection calls to TinyIDP

Rejected. Jitsi's supported token modules validate locally. Introspection would create synchronous latency and availability coupling.

## 17. Implementation Plan

### Phase 1: Production Glazed composition

1. Extract production fields and settings into `internal/sections/production`.
2. Change `NewServeProductionCommand` to accept a registry and compose all sections.
3. Build `serve-production` with `AppName`, config plan, profile middleware, and profile settings.
4. Add configuration precedence and source-provenance tests.
5. Add redacted `config check` or production-aware `print-config`.

### Phase 2: Plugin kernel

1. Implement descriptor validation and immutable registry.
2. Implement definition preparation and client requirements.
3. Implement runtime build ordering and reverse cleanup.
4. Implement derived scoped route mounting.
5. Extract common production HTTP security middleware.
6. Aggregate plugin readiness into the provider or host report.

### Phase 3: OIDC relying-party broker

1. Define broker request, identity, completion, and error contracts.
2. Add durable integration transaction storage and SQLite migration.
3. Implement state hashing, PKCE, nonce, browser binding, encryption, expiry, and atomic consumption.
4. Implement provider-backed `http.RoundTripper`.
5. Implement code exchange, ID-token verification, and userinfo mapping.
6. Validate plugin OIDC client requirements against the catalog.
7. Add login, signup, session, replay, expiry, and restart tests.

### Phase 4: Jitsi policy

1. Define versioned JSON schemas and TypeScript declarations.
2. Implement a Jitsi-specific executor and bounded worker pool.
3. Implement deterministic tests and a script check/explain command.
4. Add bounded application membership capabilities only when a concrete workflow requires them.
5. Expose readiness and operational metric snapshots.

### Phase 5: Jitsi runtime

1. Implement the Glazed section and strict settings validation.
2. Implement secret resolution and HS256 signer.
3. Implement start and callback handlers.
4. Implement exact claim construction and safe redirect construction.
5. Implement themed errors, stable logs, and durable audit.
6. Add token and handler test matrices.

### Phase 6: Observability and deployment

1. Add host OpenTelemetry meter and tracer setup.
2. Add the internal administrative listener and Prometheus exporter.
3. Add Kubernetes ports, probes, Service, and NetworkPolicy.
4. Add ConfigMap and Vault Secrets Operator resources.
5. Configure Prosody token mode with matching application and secret.
6. Add coordinated HS256 rotation documentation.

### Phase 7: End-to-end validation

1. Run local browser flows through login and signup.
2. Run Prosody valid and invalid token tests.
3. Deploy through GitOps.
4. Validate Argo CD health and logs.
5. Run Playwright against the production ingress.
6. Join with two browsers and verify media through JVB.
7. Verify no secrets, tokens, codes, or private identity values appear in logs, metrics, or configuration output.

## 18. Open Questions

- Whether version one needs email in Jitsi identity context by default, or only after explicit policy approval.
- Whether room identifiers belong in durable audit.
- Which existing core secret should derive broker transaction encryption keys, or whether a separate file-backed key is operationally preferable.
- Whether asymmetric Jitsi signing should immediately follow the HS256 baseline to improve independent rotation.

## 19. File and API Reference

### TinyIDP source

- `cmd/tinyidp/main.go` — Cobra and Glazed command construction.
- `internal/cmds/profiles.go` — configuration source precedence.
- `internal/cmds/config.go` — explicit config plan.
- `internal/cmds/serve_production.go` — production composition and lifecycle.
- `internal/sections/oidc/section.go` — reusable Glazed section precedent.
- `internal/sections/oidc/settings.go` — typed section decoding.
- `internal/productionconfig` — reviewed client catalog.
- `pkg/embeddedidp/options.go` — production validation boundary.
- `pkg/embeddedidp/provider.go` — handler, readiness, lifecycle, and stats.
- `internal/fositeadapter/session.go` — browser session internals deliberately not exposed to plugins.
- `pkg/idpscript/capabilities.go` — bounded capability bindings.
- `pkg/idpsignup` — script generation, pool, readiness, metrics, and audit precedent.
- `pkg/idp/audit.go` — durable audit and delivery-failure semantics.

### External APIs and protocol references

- [Jitsi token authentication](https://jitsi.github.io/handbook/docs/devops-guide/token-authentication/)
- [Jitsi JWT and Prosody token contract](https://github.com/jitsi/lib-jitsi-meet/blob/master/doc/tokens.md)
- [Jitsi OIDC adapter](https://github.com/jitsi-contrib/jitsi-oidc-adapter)
- [Go shared-plugin package warnings](https://pkg.go.dev/plugin)
- [HashiCorp subprocess plugin architecture](https://github.com/hashicorp/go-plugin)
- `github.com/go-go-golems/glazed/pkg/cmds/schema`
- `github.com/go-go-golems/glazed/pkg/cmds/values`
- `github.com/go-go-golems/glazed/pkg/cmds/sources`
- `github.com/coreos/go-oidc/v3/oidc`
- `golang.org/x/oauth2`
- `go.opentelemetry.io/otel/metric`
- `go.opentelemetry.io/otel/trace`

## 20. Key Points for the Implementer

- A plugin is trusted Go mechanics plus optional bounded JavaScript policy.
- Glazed configuration is composed before parsing; plugins never read the environment directly.
- Secret file paths are configuration, but secret bytes are runtime material.
- The OIDC broker preserves TinyIDP's login and signup behavior without exposing browser sessions.
- The Jitsi token is a separate authorization artifact, not a reused OIDC token.
- Prosody validates the Jitsi token and remains a required Jitsi component.
- Readiness, logs, audit, metrics, traces, and cleanup are part of the plugin contract.
- The first implementation should prove one Jitsi plugin before generalizing the API further.
