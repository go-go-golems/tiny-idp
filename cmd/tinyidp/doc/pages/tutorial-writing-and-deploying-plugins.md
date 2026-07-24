---
Title: "Tutorial: writing and deploying TinyIDP plugins"
Slug: writing-and-deploying-plugins
Short: "Implement a compiled-in TinyIDP integration, configure it with Glazed, test its lifecycle, and deploy it through the production host."
Topics:
- plugins
- integrations
- production
- glazed
- goja
- kubernetes
Commands:
- serve-production
Flags:
- jitsi-enabled
- jitsi-public-origin
- jitsi-xmpp-domain
- jitsi-app-id
- jitsi-oidc-client-id
- jitsi-token-ttl
- jitsi-shared-secret-file
- jitsi-policy-program-file
- jitsi-policy-pool-size
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

TinyIDP plugins are compiled-in, first-party integrations hosted by
`tinyidp serve-production`. A plugin can add a scoped HTTP integration, declare
the OIDC client it requires, consume host-owned identity and operational
services, contribute readiness, emit audit and telemetry data, and shut down
cleanly with the provider.

This tutorial explains how to implement that lifecycle and deploy it safely. It
uses the Jitsi Meet token bridge in `internal/plugins/jitsi` as the complete
working example. By the end, you will know which responsibilities belong to
the plugin, which remain with the TinyIDP host, and which deployment contracts
must be validated before production.

## Understand the supported plugin model

The version 1 plugin API is not a dynamic module loader. A plugin is Go code
compiled into the TinyIDP binary and registered when the root command is
constructed. This keeps configuration schemas, routes, runtime dependencies,
and shutdown behavior visible to the host before it accepts traffic.

The API currently lives under `internal/pluginapi`. Go's `internal` import rule
means a separate repository cannot import it directly. Add first-party plugins
inside the TinyIDP repository, or deliberately promote and version the API in a
separate design before supporting external plugin modules.

Version 1 does not support:

- Loading Go shared objects at runtime.
- Installing plugins from a directory or URL.
- Treating arbitrary JavaScript as an HTTP plugin.
- Registering a plugin after `serve-production` starts.
- Giving plugin code direct access to the TinyIDP database.

JavaScript can still participate as a bounded policy inside a compiled plugin.
The Jitsi plugin does this for meeting authorization: Go owns HTTP, OIDC,
transactions, signing, auditing, and metrics; a reviewed Goja program returns
a typed allow or deny decision.

## Follow the complete lifecycle

The host moves every registered plugin through explicit phases. Understanding
the sequence is necessary because configuration validation happens before
runtime services exist, while secret reads and HTTP construction happen only
after the production host has initialized its durable services.

```text
main registers Definition
        |
        v
serve-production composes Definition.Section()
        |
        v
Glazed resolves defaults < profile < config < env < args < flags
        |
        v
Definition.Prepare(ctx, values)
        |
        +--> disabled: retain descriptor, request no runtime
        |
        +--> enabled: validate settings and declare Requirements
                         |
                         v
host loads production OIDC client catalog
host validates plugin client requirements
                         |
                         v
Prepared.Build(ctx, RuntimeServices)
                         |
                         v
host mounts /integrations/<plugin-id>/
host combines plugin readiness with core readiness
                         |
                         v
host closes runtimes in reverse order
```

The corresponding interfaces are in `internal/pluginapi/api.go`:

```go
type Definition interface {
    Descriptor() Descriptor
    Section() (schema.Section, error)
    Prepare(context.Context, *values.Values) (Prepared, error)
}

type Prepared interface {
    Descriptor() Descriptor
    Enabled() bool
    Requirements() Requirements
    Build(context.Context, RuntimeServices) (Runtime, error)
}

type Runtime interface {
    Descriptor() Descriptor
    Handler() http.Handler
    Readiness(context.Context) idp.ReadinessCheck
    Close(context.Context) error
}
```

Keep compile-time assertions next to each implementation:

```go
var _ pluginapi.Definition = Definition{}
var _ pluginapi.Prepared = (*prepared)(nil)
var _ pluginapi.Runtime = (*Runtime)(nil)
```

These assertions turn an accidental interface drift into a compile error at
the implementation site.

## Step 1: define a stable descriptor

The descriptor gives the plugin its canonical identity. The ID must contain
lowercase ASCII letters, digits, and hyphens. It determines the route prefix,
so changing it changes the public HTTP API.

The Jitsi definition uses:

```go
func (Definition) Descriptor() pluginapi.Descriptor {
    return pluginapi.Descriptor{
        ID:         "jitsi",
        APIVersion: pluginapi.APIVersion,
        Summary:    "Jitsi Meet token bridge",
    }
}
```

The host derives:

```text
/integrations/jitsi/
```

from that descriptor. A runtime handler registered at `/start` is therefore
publicly available at `/integrations/jitsi/start`.

Treat the plugin ID and its routes as an API contract. Validate untrusted path
and query inputs inside the runtime; do not place tenant names, usernames, or
configuration values into the descriptor.

## Step 2: expose configuration through a Glazed section

Each plugin contributes one Glazed section to `serve-production`. Use a unique
section slug and a unique field prefix. The prefix turns a field such as
`enabled` into the command-line flag `--jitsi-enabled` and the corresponding
TinyIDP environment/config key.

The Jitsi section begins like this:

```go
const (
    SectionSlug   = "plugin-jitsi"
    sectionPrefix = "jitsi-"
)

type Settings struct {
    Enabled           bool   `glazed:"enabled"`
    PublicOrigin      string `glazed:"public-origin"`
    XMPPDomain        string `glazed:"xmpp-domain"`
    AppID             string `glazed:"app-id"`
    OIDCClientID      string `glazed:"oidc-client-id"`
    TokenTTL          string `glazed:"token-ttl"`
    SharedSecretFile  string `glazed:"shared-secret-file"`
    PolicyProgramFile string `glazed:"policy-program-file"`
    PolicyPoolSize    int    `glazed:"policy-pool-size"`
}

func (Definition) Section() (schema.Section, error) {
    return schema.NewSection(
        SectionSlug,
        "Jitsi integration",
        schema.WithPrefix(sectionPrefix),
        schema.WithFields(
            fields.New(
                "enabled",
                fields.TypeBool,
                fields.WithDefault(false),
                fields.WithHelp("Enable the Jitsi token bridge"),
            ),
            // Add the remaining typed fields here.
        ),
    )
}
```

Follow these configuration rules:

- Use Glazed fields instead of reading environment variables in plugin code.
- Give every field a type and operationally useful help text.
- Use defaults only when a default is safe in every environment.
- Accept a secret **file path**, never secret bytes, in settings.
- Keep field decoding in `Prepare`; runtime handlers should receive typed,
  validated state.
- Keep section slugs and prefixes unique. `pluginapi.NewRegistry` rejects
  collisions during process construction.

The normal TinyIDP precedence applies:

```text
defaults < profiles < config files < environment < arguments < flags
```

This means the plugin does not need its own configuration loader. It receives
the same source provenance and precedence behavior as the production host.

## Step 3: prepare and validate configuration

`Prepare` runs after Glazed has resolved all configuration sources but before
the host opens runtime services. Decode settings, return early when disabled,
and validate every enabled setting that does not require a secret read.

```go
func (Definition) Prepare(
    ctx context.Context,
    vals *values.Values,
) (pluginapi.Prepared, error) {
    if ctx == nil || vals == nil {
        return nil, errors.New("plugin preparation context and values are required")
    }

    settings := Settings{}
    if err := vals.DecodeSectionInto(SectionSlug, &settings); err != nil {
        return nil, fmt.Errorf("decode plugin settings: %w", err)
    }

    production, err := productionsection.GetSettings(vals)
    if err != nil {
        return nil, err
    }

    value := &prepared{
        descriptor: (Definition{}).Descriptor(),
        settings:   settings,
        issuer:     strings.TrimSuffix(production.Issuer, "/"),
    }
    if !settings.Enabled {
        return value, nil
    }
    if err := value.validate(); err != nil {
        return nil, err
    }
    return value, nil
}
```

Validate origins, domains, identifiers, durations, pool sizes, and bounded
non-secret policy files here. Fail startup for invalid enabled configuration.
Do not defer a deterministic configuration error until the first HTTP request.

Do not read the signing secret in `Prepare`. The host has not yet provided its
bounded `SecretResolver`, and configuration inspection must remain
secret-free.

## Step 4: declare OIDC client requirements

An integration that starts an internal authorization flow must declare the
browser client it needs. The production host validates the requirement against
the reviewed client catalog before it builds the plugin.

The Jitsi plugin requires a public authorization-code client with PKCE:

```go
func (p *prepared) Requirements() pluginapi.Requirements {
    if !p.Enabled() {
        return pluginapi.Requirements{}
    }
    return pluginapi.Requirements{
        OIDCClients: []pluginapi.OIDCClientRequirement{{
            ID:          p.settings.OIDCClientID,
            RedirectURI: p.issuer + p.descriptor.RoutePrefix() + "callback",
            Scopes:      []string{"openid", "profile", "email"},
            Public:      true,
            RequirePKCE: true,
        }},
    }
}
```

The matching production client catalog must contain the exact callback:

```json
{
  "id": "tinyidp-jitsi-prod",
  "profile": "browser",
  "redirectURIs": [
    "https://idp-jitsi.example.test/integrations/jitsi/callback"
  ],
  "allowedScopes": ["email", "openid", "profile"]
}
```

The host rejects startup when the client is absent, the redirect URI differs,
PKCE is not required, the authorization-code grant is unavailable, or an
allowed scope is missing. Do not duplicate these checks in each plugin.

## Step 5: build only from host-provided services

`Build` receives the capabilities that the production host deliberately gives
plugins:

| Service | Purpose |
| --- | --- |
| `OIDC` | Start and complete a host-owned authorization-code/PKCE transaction. |
| `Secrets` | Read a bounded secret from a reviewed owner-private path. |
| `Audit` | Write security-relevant events through the host audit sink. |
| `Logger` | Emit structured logs under host logging policy. |
| `Meter` | Create OpenTelemetry metrics. |
| `Tracer` | Create OpenTelemetry spans. |
| `Clock` | Make time-dependent behavior testable. |
| `Random` | Obtain host-provided cryptographic randomness. |

Check every required service before constructing the runtime:

```go
func (p *prepared) Build(
    ctx context.Context,
    services pluginapi.RuntimeServices,
) (pluginapi.Runtime, error) {
    if services.OIDC == nil ||
        services.Secrets == nil ||
        services.Audit == nil ||
        services.Clock == nil ||
        services.Random == nil {
        return nil, errors.New("plugin requires OIDC, secrets, audit, clock, and random services")
    }

    secret, err := services.Secrets.Read(ctx, p.settings.SharedSecretFile, 32)
    if err != nil {
        return nil, fmt.Errorf("read plugin signing secret: %w", err)
    }
    defer zeroSecret(secret)

    // Construct bounded signers, policy runtimes, metrics, routes, and cleanup.
    return newRuntime(p.settings, services, secret)
}
```

Do not open the TinyIDP SQLite database from the plugin. Use the host broker
for identity transactions and the audit interface for security records. If a
new durable primitive is genuinely necessary, design it as a narrow host
service with explicit atomicity and privacy semantics.

## Step 6: implement routes under the scoped prefix

The runtime returns one `http.Handler`. The plugin host strips the descriptor
prefix before dispatching, so the Jitsi runtime registers relative paths:

```go
mux := http.NewServeMux()
mux.HandleFunc("/start", runtime.start)
mux.HandleFunc("/callback", runtime.callback)
runtime.handler = mux
```

The host mounts this handler at `/integrations/jitsi/` and adds common security
headers:

```text
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Content-Security-Policy: default-src 'none'; style-src 'self'; ...
```

Plugin handlers still own request-specific security:

- Restrict HTTP methods and return `Allow` when appropriate.
- Reject duplicate query parameters.
- Validate room, tenant, and callback values against bounded formats.
- Use host-owned OIDC state, nonce, PKCE, expiry, replay protection, and
  browser binding.
- Render safe diagnostic identifiers rather than internal errors.
- Never log codes, cookies, tokens, secrets, or complete plugin state.
- Audit accepted and rejected security decisions.
- Redirect only to a validated configured origin.

## Step 7: use the OIDC broker instead of implementing login

The relying-party broker lets a plugin authenticate a TinyIDP browser identity
without handling passwords, sessions, authorization codes, or token endpoint
calls itself.

Start a transaction:

```go
result, err := services.OIDC.Start(ctx, pluginapi.StartRequest{
    PluginID:       descriptor.ID,
    ClientID:       settings.OIDCClientID,
    CallbackPath:   descriptor.RoutePrefix() + "callback",
    Scopes:         []string{"openid", "profile", "email"},
    PluginState:    encodedBoundedState,
    BrowserBinding: bindingCookie,
    Registration:   requestAsksForSignup,
    SelectAccount:  requestAsksForAccountChooser,
    TTL:            10 * time.Minute,
})
```

Complete it from the callback:

```go
completion, err := services.OIDC.Complete(ctx, pluginapi.CompleteRequest{
    PluginID:       descriptor.ID,
    BrowserBinding: bindingCookie,
    State:          state,
    Code:           code,
})
```

`Completion.Identity` contains the trusted identity mapped by the host:

```text
subject, email, emailVerified, name, preferredUsername,
groups, roles, authTime
```

`Completion.PluginState` contains the encrypted, one-time state the plugin
supplied at start. Validate it again before using it. A successful decode does
not replace domain validation.

## Step 8: add Goja as a bounded policy, not a configuration language

Use Goja when an operator needs programmable authorization over a stable,
typed input and output contract. Go must still own I/O, cryptographic signing,
state transitions, HTTP, metrics, audit, deadlines, and cleanup.

The Jitsi policy declares one versioned lambda:

```javascript
const A = require("tinyidp").v1;

module.exports = A.program("jitsi-policy", program => {
  const decide = A.lambda("integration.jitsi.authorize@v1", {
    kind: "provider",
    input: "integration.jitsi.authorize.input.v1",
    output: "integration.jitsi.authorize.output.v1",
    outcomes: ["complete", "deny"],
    effects: [],
    capabilities: [],
    timeoutMs: 50,
    maxCapabilityCalls: 0,
    maxOutputBytes: 4096,
    run: ctx => {
      if (!ctx.input.identity.email) {
        return A.result.deny("verified_email_required");
      }
      return A.result.complete({
        kind: "complete",
        claims: {
          displayName: ctx.input.identity.displayName,
          includeEmail: true,
          moderator: false,
        },
      });
    },
  });

  program.provider("authorization", "jitsi", {
    version: 1,
    state: "virtual",
    replayProtection: "none",
    revocation: "none",
    handlers: {decide},
  });
});
```

The Go executor validates the exact lambda ID, schema IDs, allowed outcomes,
effects, capabilities, timeout, and output. It warms a bounded runtime pool
before readiness succeeds. Unknown denial codes and malformed completion data
fail closed.

Add capabilities only when the policy genuinely needs a host operation.
Version the capability and bind it explicitly at every invocation seam. A
declared but unbound capability is a startup or invocation defect, not a
request to let JavaScript access arbitrary Go state.

## Step 9: implement readiness, telemetry, audit, and shutdown

A plugin is not ready merely because its routes were mounted. Report whether
the runtime can perform its work:

```go
func (r *Runtime) Readiness(_ context.Context) idp.ReadinessCheck {
    ready := !r.closed.Load() &&
        r.signer != nil &&
        (r.policy == nil || r.policy.Ready())

    reason := ""
    if !ready {
        reason = "plugin_runtime_unavailable"
    }
    return idp.ReadinessCheck{
        Name:      "plugin." + r.descriptor.ID,
        Ready:     ready,
        Reason:    reason,
        CheckedAt: r.services.Clock.Now().UTC(),
    }
}
```

The host combines core and plugin readiness. A required enabled plugin can
therefore keep `/readyz` false when its signer, policy pool, or other runtime
dependency is unavailable.

Use bounded, low-cardinality metric attributes such as plugin, operation,
outcome, and reason class. Do not use email, subject, room, tenant, token, or
request IDs as metric labels.

Make `Close` idempotent and context-aware. Close policy pools, signers, and
other resources. `pluginhost.Close` runs plugins in reverse build order.

## Step 10: register the plugin in the binary

Registration is intentionally explicit. Import the definition in
`cmd/tinyidp/main.go`, construct the immutable registry, and pass it to the
production command:

```go
import (
    "github.com/go-go-golems/tiny-idp/internal/pluginapi"
    exampleplugin "github.com/go-go-golems/tiny-idp/internal/plugins/example"
)

registry, err := pluginapi.NewRegistry(
    exampleplugin.Definition{},
)
cobra.CheckErr(err)

productionCmd, err := cmds.NewServeProductionCommand(registry)
cobra.CheckErr(err)
```

When retaining existing plugins, list all definitions:

```go
registry, err := pluginapi.NewRegistry(
    exampleplugin.Definition{},
    jitsiplugin.Definition{},
)
```

The registry sorts definitions by ID and rejects duplicate IDs, section slugs,
and field prefixes. There is no load-order override.

## Step 11: test each boundary

Test the definition, host lifecycle, runtime, policy, protocol artifact, and
deployment independently. A single browser test cannot identify which
boundary failed.

Minimum Go test matrix:

- Descriptor validation and registry collisions.
- Disabled preparation without secret reads.
- Enabled setting validation.
- Exact OIDC client requirements.
- Missing host service rejection.
- Secret read bounds and secret clearing.
- Route method and input validation.
- Broker start, callback completion, browser binding, replay, and expiry.
- Audit failure behavior.
- Readiness before and after close.
- Idempotent shutdown.
- Metrics and traces without sensitive attributes.
- Goja allow, deny, malformed output, timeout, saturation, and interruption.
- Protocol-specific wrong secret, issuer, audience, domain, expiry, and scope.

Run the focused Jitsi packages:

```sh
go test ./internal/pluginapi ./internal/pluginhost/... ./internal/plugins/jitsi -count=1
```

Run the local production-shaped stack:

```sh
./examples/tinyidp-jitsi/scripts/00-init-secrets.sh

tmux new-session -s tinyidp-jitsi \
  'docker compose -f examples/tinyidp-jitsi/compose.yaml up --build'

./examples/tinyidp-jitsi/scripts/02-smoke.sh
./examples/tinyidp-jitsi/scripts/03-browser-tests.sh
```

The browser suite covers signup, login, account selection, cancellation,
policy denial, logout, malformed JWT rejection, and a two-browser JVB
conference.

## Step 12: configure `serve-production`

Enable a plugin only after its core production files are valid. The Jitsi
deployment passes:

```text
--jitsi-enabled
--jitsi-public-origin=https://meet.example.test
--jitsi-xmpp-domain=meet.example.test
--jitsi-app-id=tinyidp-jitsi-prod
--jitsi-oidc-client-id=tinyidp-jitsi-prod
--jitsi-token-ttl=5m
--jitsi-shared-secret-file=/run/tinyidp-runtime-secrets/jitsi-shared-secret
--jitsi-policy-program-file=/config/jitsi-policy.js
--jitsi-policy-pool-size=4
```

The host also requires its issuer, client catalog, theme catalog and CSS,
signup program, SQLite path, audit path, owner-private token secret, public
listener mode, and private administration listener.

Use `trusted-proxy-http` behind a reviewed Traefik proxy CIDR. Do not configure
an HTTPS public issuer while starting a plain local-development listener that
does not validate forwarded origin information.

## Step 13: deploy with immutable artifacts

The production deployment example is under:

```text
deploy/kubernetes/tinyidp-jitsi/
```

Its deployment boundary includes:

- An immutable TinyIDP image.
- Reviewed client, theme, CSS, signup, and policy ConfigMap assets.
- A `local-path` PVC for SQLite and audit state.
- A restricted root init container for state ownership and secret handoff.
- A UID/GID 65532 TinyIDP process with no capabilities.
- Vault Secrets Operator for source secret delivery.
- A memory-backed, owner-private runtime secret volume.
- Traefik ingress with an HTTPS issuer.
- A private administration Service and NetworkPolicy.
- A pinned Jitsi Helm chart with Prosody JWT validation.

Render both TinyIDP and Jitsi before changing GitOps:

```sh
kubectl kustomize deploy/kubernetes/tinyidp-jitsi \
  > /tmp/tinyidp-jitsi-kustomize.yaml

helm template jitsi oci://ghcr.io/jitsi-contrib/jitsi-meet \
  --version 2.22.0 \
  --namespace tinyidp-jitsi \
  --values deploy/kubernetes/tinyidp-jitsi/jitsi-values.yaml \
  > /tmp/tinyidp-jitsi-helm.yaml

./deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh \
  /tmp/tinyidp-jitsi-helm.yaml
```

Then prove lifecycle behavior, not only YAML shape:

1. Start against a fresh PVC.
2. Confirm readiness.
3. Restart the Pod without deleting the PVC.
4. Confirm readiness again.
5. Inspect prepared secret ownership and mode without printing values.
6. Run the browser and protocol rejection matrix.
7. Inspect audit event classes, metrics, and redacted logs.

Pin a full source revision and immutable image tag in Argo CD. After sync,
confirm the desired revision, active operation revision, new ReplicaSet, Pod
image, and container arguments all agree.

## Worked example: Jitsi responsibility map

The Jitsi plugin demonstrates the intended separation:

| Responsibility | Owner |
| --- | --- |
| Plugin identity and Glazed section | `internal/plugins/jitsi/definition.go` |
| Durable PKCE transaction | `internal/pluginhost/oidcbroker` |
| Meeting request and callback routes | `internal/plugins/jitsi/runtime.go` |
| Bounded Goja authorization | `internal/plugins/jitsi/policy.go` |
| HS256 claim construction and signing | `internal/plugins/jitsi/token.go` |
| Route mounting and common headers | `internal/pluginhost/host.go` |
| Production composition | `internal/cmds/serve_production.go` |
| Compiled registration | `cmd/tinyidp/main.go` |
| Kubernetes deployment | `deploy/kubernetes/tinyidp-jitsi` |
| Local end-to-end stack | `examples/tinyidp-jitsi` |

Prosody remains the Jitsi JWT verifier. TinyIDP does not replace XMPP,
conference control, or media transport.

## Review checklist

Before requesting review, verify:

- The descriptor ID, route prefix, section slug, and field prefix are unique.
- `Prepare` is secret-free and validates all deterministic enabled settings.
- Requirements declare the exact reviewed OIDC client.
- `Build` uses only explicit host services.
- Routes validate methods, duplicate inputs, lengths, and redirect targets.
- Security failures are audited and fail closed.
- Logs, metrics, and traces contain no sensitive or high-cardinality identity
  data.
- Optional Goja policy has versioned schemas, bounded resources, and explicit
  capabilities.
- Readiness represents actual plugin dependencies.
- Shutdown is idempotent.
- Tests cover disabled, accepted, denied, malformed, timeout, replay, restart,
  and wrong-secret/protocol cases.
- Deployment tests run the exact rendered configuration and retain state across
  a second start.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `duplicate plugin id` | Two definitions use the same descriptor ID. | Assign one canonical ID; the registry has no override order. |
| `section slug ... collides` | Two plugins use the same Glazed section slug. | Use `plugin-<id>` or another unique stable slug. |
| `section prefix ... collides` | Two plugins generate the same flag prefix. | Use a unique `<id>-` prefix and update settings tags/tests. |
| Plugin flags do not appear | The definition was not registered before constructing `serve-production`. | Add the definition to `pluginapi.NewRegistry` in `cmd/tinyidp/main.go`. |
| Plugin is configured but no routes exist | `Enabled()` returned false or the wrong values section was decoded. | Inspect resolved Glazed configuration and test `Prepare` directly. |
| Required OIDC client is missing | The client catalog does not match `Requirements()`. | Add the exact public PKCE client, callback, grant, and scopes. |
| Callback is rejected | State, code, browser binding, expiry, or plugin identity did not match the durable transaction. | Preserve the binding cookie and use the broker; do not implement callback state independently. |
| Secret file is rejected | The runtime file is not a bounded regular owner-private file. | Materialize it as the TinyIDP UID with mode 0400 or 0600; do not weaken validation. |
| Policy never becomes ready | Compilation, schema validation, or pool warming failed. | Run policy unit tests and inspect the startup error before serving traffic. |
| Policy reports an unbound capability | The lambda declares a capability that the invocation did not bind. | Bind the versioned capability at that invocation seam or remove the requirement. |
| Handler CSS is blocked | The response relies on inline style or an unregistered asset. | Serve reviewed CSS from the host theme route and comply with the host CSP. |
| First Pod works but restart fails | The init sequence was tested only on empty state. | Reproduce a UID-65532 mode-0700 retained volume and run the exact initializer twice. |
| Argo source changed but Pod did not | An older active operation is still retrying. | Compare desired and active revisions, then inspect ReplicaSet creation before diagnosing the new source. |

## See Also

- `tinyidp help developer-guide` — package layout, configuration sections, and
  contribution workflow.
- `tinyidp help reference` — general client, endpoint, and configuration
  reference.
- `examples/tinyidp-jitsi/README.md` — local complete Jitsi stack.
- `deploy/kubernetes/tinyidp-jitsi/README.md` — Kubernetes, Vault, and Jitsi
  deployment contract.
