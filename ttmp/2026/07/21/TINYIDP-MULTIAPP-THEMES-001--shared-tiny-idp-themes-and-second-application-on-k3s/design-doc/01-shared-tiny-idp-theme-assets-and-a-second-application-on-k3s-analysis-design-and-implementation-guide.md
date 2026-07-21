---
Title: 'Shared tiny-idp theme assets and a second application on k3s: analysis, design, and implementation guide'
Ticket: TINYIDP-MULTIAPP-THEMES-001
Status: active
Topics:
    - oidc
    - identity
    - auth
    - architecture
    - operations
    - security
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/tinyidp-message-app/loginui/renderer.go
      Note: Existing embedded MessageDesk CSS to migrate
    - Path: repo://internal/cmds/serve_production.go
      Note: Current one-client production host to replace with catalog loading
    - Path: repo://internal/fositeadapter/rendering.go
      Note: Interaction CSP and rendering boundary
    - Path: repo://pkg/embeddedidp/bootstrap.go
      Note: Existing multi-client bootstrap and conflict validation
    - Path: repo://pkg/idpui/types.go
      Note: Safe provider-created interaction presentation model
    - Path: ws://hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk
      Note: Current k3s deployment topology to generalize
ExternalSources: []
Summary: 'Intern-ready design for a shared TinyIDP: declarative multi-client bootstrapping, GitOps-mounted per-client themes, a second independently deployed browser application, and a bounded Kubernetes-operator research branch.'
LastUpdated: 2026-07-21T10:52:49.15243738-04:00
WhatFor: Explain why the current deployment is single-app, how to make it safely shared, and how to deploy a second app without weakening OIDC or cluster boundaries.
WhenToUse: Use as the implementation and review guide for TINYIDP-MULTIAPP-THEMES-001.
---


# Shared tiny-idp theme assets and a second application on k3s: analysis, design, and implementation guide

## Executive Summary

TinyIDP is already capable of holding multiple OAuth clients. Its bootstrap API
accepts a slice of `ClientSpec` values, normalizes each client, and refuses a
startup configuration that conflicts with an existing durable client record.
The deployed production command is not yet multi-application: it accepts one
`--message-desk-origin`, creates one `tinyidp-message-app` client, selects one
MessageDesk renderer, and serves one stylesheet embedded in the TinyIDP image.
Those are host-wiring choices, not inherent protocol limits.

This ticket proposes a first shared deployment with two browser applications:
the existing MessageDesk and a deliberately separate second example application
whose concrete product is selected before Phase 4. A read-only, GitOps-mounted
client catalog declares the exact registered clients. A second read-only theme
catalog maps those already-validated client IDs to names, display labels, and
same-origin CSS files. The application can contribute its CSS source to the
reviewed GitOps change, but it cannot choose a CSS URL during OAuth, execute
scripts in the IdP, change the form protocol, or obtain write access to the
IdP Pod.

The recommended first implementation is ordinary Argo CD plus Kustomize. A
custom Kubernetes operator is worth researching because it could reduce
repetition as the number of apps grows, but it should not be introduced before
we have operated the two-app design. An operator would add a controller,
CRDs, RBAC, reconciliation races, and an additional privileged trust boundary.

## 1. What the intern is building

There are three different systems in this work. Keeping them distinct prevents
the most common mistake: treating a visual theme as if it were an OAuth client
configuration, or treating an application deployment as if it controlled the
identity provider.

| System | Owns | Does not own |
| --- | --- | --- |
| TinyIDP | accounts, sessions, OIDC protocol state, consent, tokens, client validation, audited signup effects | application pages and application data |
| Each relying-party application | its own HTTP server, session, data, public hostname, callback handler, application UI, and CSS source contribution | the IdP's redirect validation, login form protocol, or account database |
| k3s GitOps configuration | the reviewed desired deployment state: images, ConfigMaps, volumes, network policy, ingress, and client/theme catalogs | imperative protocol decisions made from a browser request |

An OAuth *client* is the registered identity of an application at the IdP. A
browser application requests authorization using `client_id`, `redirect_uri`,
scope, state, nonce, and a PKCE challenge. The provider loads that client from
durable storage before rendering an interaction. A *theme* is only an IdP
presentation choice associated with that validated client. It has no authority
to change client registration, an authorization decision, a user, a password,
or a redirect URI.

The target topology is:

```text
Browser
  | HTTPS                         | HTTPS
  v                               v
MessageDesk host              Second-app host
  | /auth/login                    | /auth/login
  |                                 |
  +---------- redirect to ----------+
                 public issuer
                       |
                       v
             idp-message-desk.yolo.scapegoat.dev
                       |
                 Traefik TLS edge
                       |
                       v
             TinyIDP Pod (one shared SQLite state)
              |                         |
              | validates client         +-- read-only theme ConfigMap
              | selects theme                  themes.json + CSS files
              v
        provider-owned login / signup / consent HTML
```

The initial deployment retains the existing issuer hostname. Renaming an
issuer changes the issuer claim, discovery address, cookies, and every relying
party configuration. That is a planned migration, not a prerequisite for
adding the second application.

## 2. Current state, with evidence

### 2.1 The reusable provider is multi-client capable

`embeddedidp.BrowserClient` constructs a public browser client that requires
PKCE and permits only authorization-code and refresh-token grants
([bootstrap.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/bootstrap.go:32)).
`BootstrapConfig` contains `Clients []ClientSpec`
([bootstrap.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/bootstrap.go:49)).
Bootstrap sorts the desired clients, rejects duplicate IDs, creates missing
clients, and compares existing records against the desired normalized
configuration ([bootstrap.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/embeddedidp/bootstrap.go:102)).

This gives the implementation an important invariant: adding a second client
is supported at the durable-provider level, and accidental widening of an
existing redirect URI list is a startup error rather than a silent change.

### 2.2 The production embedding is intentionally single-app today

`serve-production` accepts `--message-desk-origin` and turns it into exactly
one browser client named `tinyidp-message-app`
([serve_production.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go:165)).
The same command constructs the MessageDesk renderer and installs it as the
only `embeddedidp.UIConfig.Renderer`
([serve_production.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go:190)).
It gives the renderer a route before the provider handler
([serve_production.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go:226)).

The renderer embeds `static/login.css` in the Go binary and recognizes only
`/static/tinyidp/login.css`
([renderer.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-message-app/loginui/renderer.go:19)).
This is why the current public IdP looks like MessageDesk even when it is
conceptually a provider: the command is a purpose-built one-app host.

### 2.3 The provider gives a renderer a safe presentation model

`idpui.InteractionPage` deliberately omits passwords, cookies, redirect URIs,
raw authorization requests, and stored interaction records. Its public
`ClientID` is separate from the opaque interaction handle
([types.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpui/types.go:147)).
That is precisely the input a theme selector may use. The renderer must not
reconstruct authorization parameters or rely on a query string.

The provider's interaction CSP uses `default-src 'none'` and `style-src
'self'`, and permits form submission only to the IdP plus the already-validated
terminal redirect origin ([rendering.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/rendering.go:111)).
Same-origin mounted CSS fits this contract. A CSS URL supplied by an
application would not.

### 2.4 The existing k3s workload is a coupled pair

The Argo Application points at `gitops/kustomize/tiny-message-desk` and deploys
into namespace `tiny-message-desk`
([tiny-message-desk.yaml](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/applications/tiny-message-desk.yaml:1)).
The TinyIDP Deployment has a persistent SQLite state volume, read-only signup
program ConfigMap, Vault-delivered token secret, and trusted-proxy listener
([deployment.yaml](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/deployment.yaml:70)).

MessageDesk is a separate Deployment in the same namespace, and it preserves
the public HTTPS issuer name while routing the backchannel to Traefik through a
host alias ([deployment.yaml](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/deployment.yaml:202)).
The NetworkPolicy explicitly permits that application only to Traefik port
8443, rather than directly to the IdP Service
([network-policy.yaml](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/network-policy.yaml:72)).

The second app must repeat that *relationship* with its own namespace,
ServiceAccount, Service, Ingress, NetworkPolicy, and persistent data. It must
not share MessageDesk's PVC or identity-provider credentials.

## 3. Requirements and non-goals

### Requirements

- One durable TinyIDP instance must register and serve at least two public
  browser clients, each with its own HTTPS callback and logout URI.
- The IdP must choose a theme only after it has loaded the registered client.
- CSS must live outside the TinyIDP image and be delivered from the IdP's own
  HTTPS origin.
- Application teams may contribute reviewed CSS through GitOps, but no browser
  parameter, application endpoint, or app Pod may select or serve IdP assets.
- The second application must be an independent Argo CD deployment and retain
  the cluster's Traefik TLS topology.
- Existing MessageDesk users, client registration, issuer, and SQLite state
  must survive the change.

### Non-goals for this ticket

- This ticket does not define arbitrary application HTML or JavaScript inside
  the IdP.
- This ticket does not change device authorization, token audiences, or the
  shared account model.
- This ticket does not make client removal automatic. Revoking a client is a
  deliberate administrative operation because a missing declaration must not
  silently delete durable protocol state.
- This ticket does not require a Kubernetes operator for the first rollout.

## 4. Proposed architecture

### 4.1 Declarative client catalog

Replace the MessageDesk-specific production flag with a bounded file loaded at
startup. It is non-secret configuration mounted read-only from a ConfigMap.

```json
{
  "version": 1,
  "clients": [
    {
      "id": "tinyidp-message-app",
      "profile": "browser",
      "redirectURIs": ["https://message-desk.yolo.scapegoat.dev/auth/callback"],
      "postLogoutRedirectURIs": ["https://message-desk.yolo.scapegoat.dev/"],
      "allowedScopes": ["openid", "profile"]
    },
    {
      "id": "tinyidp-second-example-app",
      "profile": "browser",
      "redirectURIs": ["https://second-app.yolo.scapegoat.dev/auth/callback"],
      "postLogoutRedirectURIs": ["https://second-app.yolo.scapegoat.dev/"],
      "allowedScopes": ["openid", "profile"]
    }
  ]
}
```

The loader converts each entry to `embeddedidp.BrowserClient` and passes the
complete list to `embeddedidp.Bootstrap`. It accepts no inline secrets and no
unbounded external references. It rejects unknown JSON fields so a spelling
mistake cannot silently alter the intended contract.

```go
func loadClientCatalog(path string) ([]embeddedidp.ClientSpec, error) {
    raw := readRegularFileAtMost(path, 256<<10)
    catalog := decodeStrictJSON[ClientCatalog](raw)
    require(catalog.Version == 1)
    for _, c := range catalog.Clients {
        requireUnique(c.ID)
        requireHTTPSOrigins(c.RedirectURIs, c.PostLogoutRedirectURIs)
        require(c.Profile == "browser")
        specs = append(specs, embeddedidp.BrowserClient(
            c.ID, c.RedirectURIs, c.PostLogoutRedirectURIs, c.AllowedScopes))
    }
    return specs, nil
}
```

`Bootstrap` is additive for desired clients and validates clients it already
finds. Therefore, removing an entry from this catalog is not a client
revocation mechanism. The guide requires an explicit `tinyidp admin` command
or future managed lifecycle API to disable/revoke a client, audit that action,
and invalidate appropriate grants.

### 4.2 Theme catalog and asset directory

The theme catalog lives in the same ConfigMap as the CSS files, but its role is
not client registration. It only selects approved presentation.

```json
{
  "version": 1,
  "defaultTheme": "shared",
  "themes": {
    "shared": {"productName": "TinyIDP", "stylesheet": "shared.css"},
    "message-desk": {"productName": "Message Desk", "stylesheet": "message-desk.css"},
    "second-example": {"productName": "Second Example", "stylesheet": "second-example.css"}
  },
  "clientThemes": {
    "tinyidp-message-app": "message-desk",
    "tinyidp-second-example-app": "second-example"
  }
}
```

The proposed CLI boundary is:

```text
tinyidp serve-production \
  --clients-file=/etc/tinyidp/catalog/clients.json \
  --theme-dir=/etc/tinyidp/themes \
  --theme-catalog-file=/etc/tinyidp/themes/themes.json \
  ...existing listener, SQLite, audit, secret, and signup arguments...
```

The catalog loader receives a root directory, not arbitrary asset URLs. Its
rules are intentionally narrow:

- A stylesheet value is a basename ending in `.css`; it has no slash, query,
  fragment, backslash, or URL scheme.
- The resolved file must be a regular file under `--theme-dir`, be bounded in
  size, and be read before the server reports readiness.
- Every client-theme mapping must name a client in the loaded client catalog
  and a theme in the same theme catalog.
- The renderer produces `/static/themes/<theme-name>.css`; the asset handler
  maps that route to preloaded bytes. It never joins a browser path to the
  filesystem.
- A missing mapping selects the configured default theme. A missing file is a
  startup failure, not a runtime fallback.

The renderer can be expressed as a composition rather than a custom renderer
per application:

```go
type Theme struct {
    Name, ProductName, StylesheetRoute string
    CSS []byte
}
type ThemeResolver interface { Resolve(clientID string) (Theme, error) }

func (r *ThemedRenderer) RenderInteraction(ctx context.Context, out io.Writer, p idpui.InteractionPage) error {
    if err := p.Validate(); err != nil { return err }
    theme, err := r.themes.Resolve(p.ClientID) // provider-created page only
    if err != nil { return err }
    return r.template.Execute(out, view{Page: p.Clone(), Theme: theme})
}
```

The template remains owned by TinyIDP. It contains the exact provider-generated
interaction, CSRF, action, and error fields. The mounted theme may change
colors, spacing, typography, logo treatment, and the product name shown in the
page chrome. It cannot change POST targets or hidden protocol fields.

### 4.3 Request sequence

```text
1. Second app redirects browser to /authorize with its registered client_id.
2. TinyIDP validates the OAuth request and loads the durable client record.
3. TinyIDP creates an opaque, browser-bound interaction record.
4. TinyIDP constructs InteractionPage with the validated public ClientID.
5. ThemedRenderer resolves that ID in its preloaded IdP-owned catalog.
6. Browser receives HTML with /static/themes/second-example.css.
7. Browser fetches CSS from the same IdP origin; CSP permits style-src 'self'.
8. Browser posts only the provider's CSRF-bound interaction form to TinyIDP.
9. TinyIDP authenticates/consents/signs up, then redirects only to the
   validated registered callback for that client.
```

This is deliberately not:

```text
application -> /authorize?theme=https://application.example/theme.css
```

That alternative gives an untrusted request control of browser-visible IdP
content. It is incompatible with the existing `style-src 'self'` CSP and
allows a malicious or compromised client to make the IdP look like another
application.

### 4.4 GitOps packaging and ownership

MessageDesk may own the source of its theme, for example
`examples/tinyidp-message-app/idp-theme/message-desk.css`. A source release or
an explicit deployment change copies that file into the Hetzner GitOps
repository. The GitOps PR—not an application Pod at runtime—is the approval
boundary that binds a client ID to its theme.

Use a Kustomize generator rather than a fixed-name ConfigMap plus a manually
maintained checksum:

```yaml
configMapGenerator:
  - name: tinyidp-theme-catalog
    files:
      - themes.json=config/themes.json
      - message-desk.css=config/message-desk.css
      - second-example.css=config/second-example.css
generatorOptions:
  disableNameSuffixHash: false
```

Kustomize rewrites the Deployment's ConfigMap reference to the content-hashed
name. A CSS or manifest edit consequently changes the Pod template and rolls
TinyIDP. TinyIDP reads a coherent catalog at startup, rather than attempting
to watch ConfigMap's projected symlink updates while rendering requests.

```text
application source CSS -> reviewed GitOps PR -> generated ConfigMap hash
    -> Deployment template reference -> new TinyIDP Pod -> startup validation
    -> readiness -> same-origin asset serving
```

The ConfigMap uses `defaultMode: 0444`, the Pod mount is `readOnly: true`, and
the existing Pod security context remains non-root and read-only-root-filesystem.
CSS is not secret material and must not go through Vault. The token secret and
SQLite volume remain separate as in the current Deployment.

## 5. k3s deployment for a second app

The second app must be its own Argo CD Application, with a fresh namespace such
as `tiny-second-example`. The shared IdP remains a distinct workload for this
first increment. Keeping the existing issuer and SQLite PVC avoids an issuer
migration or account-data move.

```text
Argo Application: tiny-message-desk        Argo Application: tiny-second-example
  namespace: tiny-message-desk               namespace: tiny-second-example
  Deployment, Service, Ingress,              Deployment, Service, Ingress,
  NetworkPolicy, app PVC                     NetworkPolicy, app PVC
                 \                             /
                  \ OIDC over canonical HTTPS/
                   v                         v
           Argo Application: shared-tinyidp (initially refactor existing one)
             namespace: tiny-message-desk (then optionally tiny-identity)
             SQLite PVC, Vault secret, client catalog, theme catalog,
             Service, Ingress, NetworkPolicy
```

The second app's deployment follows the known-good backchannel pattern, not a
direct ClusterIP call. It keeps the canonical external issuer URL, uses a host
alias to Traefik's in-cluster Service IP, and allows egress to Traefik Pods on
8443. This preserves the trusted-proxy boundary that rejects direct requests
with forged forwarded headers. Its probes set its own public host and forwarded
headers, just as the existing workloads do.

Before adding the second app, choose and record these facts in the ticket:

| Fact | Example value | Why it is fixed before code |
| --- | --- | --- |
| Public hostname | `second-app.yolo.scapegoat.dev` | It appears in DNS, TLS, ingress, redirect URI, and cookie behavior. |
| Client ID | `tinyidp-second-example-app` | It is the durable key for registrations, consent, audit, theme selection, and Goja signup input. |
| Callback | `/auth/callback` | It must exactly match the registered redirect URI. |
| Logout target | `/` | It must exactly match the registered post-logout URI. |
| Theme owner | second app's source + reviewed GitOps copy | It identifies who changes the visual artifact without granting runtime authority. |
| App state | own PVC/DB/none | It prevents accidental coupling to MessageDesk data. |

The current Goja signup executor already receives `ClientID` in its start
input ([executor.go](/home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpsignup/executor.go:74)). For the first two-app deployment, retain a shared signup workflow unless the second app has a genuine policy difference. If it does, dispatch in the IdP-owned reviewed program by that validated client ID; do not let a browser select a workflow name.

## 6. Decision records

### Decision: Theme configuration is mounted GitOps data, not application-controlled runtime data

- **Context:** CSS should not be baked into the TinyIDP image, while applications need different visual identities.
- **Options considered:** Embed CSS in TinyIDP; allow a CSS URL in `/authorize`; have an application Pod serve CSS; mount reviewed data into TinyIDP.
- **Decision:** Mount a reviewed theme catalog and CSS files read-only into TinyIDP through GitOps.
- **Rationale:** It decouples release cadence from the binary while retaining same-origin CSP and a small, auditable authority boundary.
- **Consequences:** A theme edit triggers a TinyIDP rollout; application developers need a GitOps PR to publish a theme. This is intentional review friction.
- **Status:** proposed.

### Decision: Theme selection uses the validated `InteractionPage.ClientID`

- **Context:** A renderer must know which application's presentation is appropriate without trusting raw request data.
- **Options considered:** query parameter; requested redirect URI; client ID from `InteractionPage`; a cookie; an external lookup.
- **Decision:** Resolve from `InteractionPage.ClientID` only.
- **Rationale:** The provider constructs the page after client validation, and the type deliberately excludes secret protocol state.
- **Consequences:** A client must be in both the client catalog and the theme catalog or receive the explicit default theme.
- **Status:** proposed.

### Decision: Keep the existing issuer for the first shared rollout

- **Context:** Its hostname says "message-desk" but it is already a public issuer with durable data and active users.
- **Options considered:** rename now; retain it; run two IdPs.
- **Decision:** Retain it while adding the second client.
- **Rationale:** Issuer migration has account/session/client implications unrelated to proving the shared-client model.
- **Consequences:** Naming remains imperfect; a generic issuer migration must be separately planned and tested.
- **Status:** proposed.

### Decision: Use GitOps manifests before a Kubernetes operator

- **Context:** A second app adds repeated catalogs, ingress, policy, and secret wiring.
- **Options considered:** hand-written Kustomize only; Helm only; a TinyIDP operator now; an operator after operating the two-app design.
- **Decision:** Ship the GitOps-only design first and research an operator in parallel.
- **Rationale:** The current system has one cluster and two clients. An operator would introduce a privileged control loop before its durable API is known.
- **Consequences:** The first second-app PR contains repetitive manifests, but those manifests become the evidence for a minimal future CRD.
- **Status:** proposed.

## 7. Operator research branch

An operator is a controller that watches custom resources and reconciles them
into ordinary Kubernetes resources. It can simplify the human-facing desired
state, but it cannot remove the underlying responsibilities: client catalog
validation, safe config packaging, IdP rollout ordering, certificates, network
policies, secrets, and status reporting.

### 7.1 A deliberately small possible API

```yaml
apiVersion: identity.tinyidp.dev/v1alpha1
kind: TinyIDP
metadata:
  name: shared
  namespace: tiny-identity
spec:
  issuer: https://idp-message-desk.yolo.scapegoat.dev
  image: ghcr.io/go-go-golems/tiny-idp:sha-...
  state:
    existingClaim: tinyidp-data
  vaultRuntimeSecret:
    name: tinyidp-runtime
  themes:
    configMapRef: tinyidp-theme-catalog
  clients:
    - id: tinyidp-message-app
      browser:
        redirectURIs: [https://message-desk.yolo.scapegoat.dev/auth/callback]
        postLogoutRedirectURIs: [https://message-desk.yolo.scapegoat.dev/]
      theme: message-desk
status:
  conditions: []
  observedGeneration: 0
  clientCatalogHash: ""
```

The first operator should own only a ConfigMap containing the generated
catalog, a Deployment, Service, and perhaps a PodDisruptionBudget. It should
not initially create DNS records, issue Vault policies, read arbitrary Secrets,
or accept cross-namespace CSS references. Those capabilities multiply RBAC and
reconciliation failure modes.

### 7.2 Client resource question

`TinyIDPClient` resources could make each app independently declare its
callback and theme:

```yaml
kind: TinyIDPClient
metadata:
  name: message-desk
  namespace: tiny-message-desk
spec:
  providerRef:
    name: shared
    namespace: tiny-identity
  clientID: tinyidp-message-app
  redirectURIs: [https://message-desk.yolo.scapegoat.dev/auth/callback]
  themeRef: message-desk
```

This is attractive, but it creates the key hard problem: a namespaced app
object asks a controller in another namespace to alter central identity state.
The controller must decide who may claim a hostname, client ID, or theme name;
whether deletion disables a client; and how to avoid a partially reconciled
catalog. For two apps, one reviewed central catalog is simpler and safer.

### 7.3 Operator success threshold

Revisit the operator after all of the following are true:

- At least three independently deployed apps use one IdP.
- The common manifests have been stable through additions, theme changes, and
  one rollback.
- The desired CRD fields can be stated without exposing ad hoc execution or
  arbitrary URLs.
- The team accepts maintaining CRD conversion, controller upgrades, metrics,
  leader election, RBAC, and recovery documentation.

Until then, Kustomize plus Argo CD is the operator: it reconciles reviewed
declarative resources, and its failure surface is already understood in this
cluster.

## 8. Phased implementation guide

### Phase 1 — Production catalog API

Create a small package such as `pkg/productionconfig` or
`internal/productionconfig` with strict parsers for clients and themes. Keep
host filesystem parsing out of `pkg/embeddedidp`; that library remains an
embedding API, while `serve-production` owns command-line and filesystem
policy. Add `--clients-file`, remove `--message-desk-origin`, and retain the
existing bounded regular-file pattern used for `--signup-program-file` in
`internal/cmds/serve_production.go`.

Test startup using two browser client declarations. Test that a preexisting
client whose callback list changed causes `ErrBootstrapConflict`. Do not write
a compatibility adapter for the removed flag: update the one production
manifest in the same change.

### Phase 2 — Theme loader and renderer

Add a generic `ThemedRenderer` under `pkg/idpui` only if its API is useful to
embedders; otherwise keep it under the production host. It validates the
catalog at startup, preloads the CSS byte slices, and returns immutable theme
values to requests. Use `html/template`, keep `InteractionPage.Validate`, and
retain the provider's CSP headers.

Unit tests must prove that unknown clients use the default theme, known clients
select their own route, a path such as `../secret`, `https://evil.example/x`,
or `x.css?y=z` is rejected, and the handler returns 404 for a route not in the
preloaded set.

### Phase 3 — GitOps migration of MessageDesk theme

Move or copy the existing MessageDesk IdP CSS from
`examples/tinyidp-message-app/loginui/static/login.css` into the theme source
directory. Add the generated ConfigMap, read-only volume mount, command flags,
and hash-driven rollout. Change the production image reference only after the
TinyIDP code PR is merged and published.

Render the kustomization locally. Inspect the generated Deployment to ensure
the ConfigMap generated name is mounted and that the Pod template changes when
the CSS changes. Do not rely on in-place ConfigMap projection updates.

### Phase 4 — Second application

Choose a concrete existing or new example. Its implementation needs the normal
xgoja hostauth browser flow, but it must configure its public base URL, issuer,
and external backchannel to the shared IdP. Give it a distinct image-publishing
workflow and GitOps Application. Add its client and theme in the same IdP
catalog rollout, then deploy the app after the IdP reports ready.

### Phase 5 — acceptance and operations

Extend the existing public acceptance script rather than creating an unrelated
test harness. For each client, prove authorization request, signup or login,
consent, callback, PKCE token exchange, local/provider logout, and theme asset
selection. Scrub URLs and logs before committing evidence. Test rollback by
reverting the GitOps ConfigMap-generator input and confirming the old IdP Pod
serves the old catalog while the SQLite PVC is unchanged.

## 9. Test and review matrix

| Layer | Required evidence |
| --- | --- |
| Catalog parser | Strict JSON decoding; schema/version failure; duplicate ID; invalid URI; bounded file size. |
| Provider bootstrap | Two clients create/validate; modified existing redirect URI fails; no automatic deletion. |
| Theme selector | Known client, default client, unknown theme, invalid path, missing CSS, escaping. |
| HTTP | Only GET/HEAD allowlisted theme assets; correct CSS content type; 404 elsewhere; CSP remains same-origin. |
| GitOps | `kustomize build`; generated ConfigMap name changes with CSS; volume remains read-only; Pod rolls. |
| Cluster | Argo synced/healthy; both app Ingress certificates ready; NetworkPolicy permits each app through Traefik only. |
| Public OIDC | Independent PKCE flow per client; wrong redirect rejected; direct forged-forwarded traffic rejected. |
| Recovery | Revert theme/catalog; keep SQLite PVC; audit client-specific startup and interaction events. |

Code review should start at the parser and renderer rather than the manifests:
those components define the trust boundary. Then inspect the production command
to ensure it loads catalogs before opening the listener. Finally inspect the
Kustomize build and Deployment volume/args to ensure the intended catalog is
actually what the running process sees.

## 10. Risks, alternatives, and open questions

### Risks

- **Issuer naming:** The existing issuer hostname is MessageDesk-branded. It
  works technically for a second app but should not become permanent by
  accident.
- **Catalog drift:** The IdP client catalog, second app settings, DNS, ingress,
  and certificate names must be reviewed as one logical change. A correct CSS
  mapping cannot compensate for a wrong redirect URI.
- **Theme overreach:** CSS can make a login page misleading. GitOps review,
  same-origin CSP, and a visible client/application label are important even
  though CSS cannot change protocol processing.
- **Shared signup policy:** Different app signup requirements require a
  deliberate IdP-owned Goja decision by validated client ID. They must not be
  smuggled into a theme catalog.
- **Operator privilege:** A controller that edits identity state or reads Vault
  would be a high-value control-plane component.

### Alternatives rejected for the first rollout

- **External CSS URL configured by the app:** breaks the same-origin security
  model and gives an app request presentation authority.
- **A separate IdP per application:** avoids theming selection but duplicates
  users, signup, issuer operation, secrets, and recovery.
- **Application sidecar serving CSS into TinyIDP:** creates runtime coupling
  and an avoidable network trust path.
- **Live filesystem watching:** ConfigMap updates can present a changing
  directory view; restart with a content-hashed mount instead.
- **Operator first:** creates a new privileged platform before two-app
  operational requirements are proven.

### Open questions to resolve before Phase 4

1. Which concrete second example should be deployed, and what data durability
   does it need?
2. Is open signup intentionally shared across both apps, or should the Goja
   program branch on `clientId` from the beginning?
3. Should the second app receive its own visible consent descriptions beyond
   the client ID, and where should that display metadata live?
4. When should the issuer migrate from `idp-message-desk...` to a generic
   hostname, and what token/session invalidation window is acceptable?
5. Who is permitted to approve an app's CSS contribution and client catalog
   entry in the GitOps repository?

## 11. References

- Current deployment plan: [TINYIDP-K3S-MSGDESK-PROD-001](../../../../18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/design-doc/01-production-implementation-and-deployment-plan-for-standalone-tiny-idp-and-message-desk.md).
- `pkg/embeddedidp/bootstrap.go` — multi-client bootstrap, profiles, and
  conflict behavior.
- `pkg/idpui/types.go` — safe interaction presentation model.
- `internal/fositeadapter/rendering.go` — interaction rendering and CSP.
- `internal/cmds/serve_production.go` — current production host wiring.
- `examples/tinyidp-message-app/loginui/renderer.go` — current embedded theme
  and asset route.
- `gitops/kustomize/tiny-message-desk/` in
  `/home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5` — live
  deployment, TLS, Vault, PVC, and network-policy pattern.
