---
Title: Production-shaped local Compose stack analysis design and implementation guide
Ticket: TINYIDP-LOCAL-COMPOSE-001
Status: active
Topics:
    - oidc
    - tiny-idp
    - kubernetes
    - local-development
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Design and implementation guide for running shared TinyIDP, Message Desk, and go-go-goja auth-host locally with production-shaped HTTPS, proxy trust, OIDC catalogs, and safe local-CA distribution."
LastUpdated: 2026-07-21T13:18:51.582701211-04:00
WhatFor: "Understand, operate, review, and extend the local two-application TinyIDP Compose topology."
WhenToUse: "Use before changing local TLS, trusted-proxy handling, OIDC client registration, per-client themes, signup behavior, or Compose startup ordering."
---

# Production-shaped local Compose stack analysis design and implementation guide

## Executive Summary

The local development environment now runs the same security-relevant shape as
the k3s deployment: one TinyIDP issuer, two relying parties, one TLS-terminating
proxy, exact HTTPS origins, PKCE browser clients, secure cookies, per-client
identity themes, and durable state. The environment is intentionally smaller
than production—it has no Vault, cert-manager, Argo CD, backup controller, or
Kubernetes NetworkPolicy—but it preserves the contracts most likely to break an
OIDC deployment during quick iteration.

The central design problem is local certificate trust. A browser and both
relying-party backchannels must trust the same local issuer certificate. Caddy
can generate that certificate automatically, but it stores the public root next
to the CA private key with owner-only permissions. Mounting Caddy's data volume
directly into non-root applications both fails and violates least privilege.
The implemented solution uses a one-shot `ca-export` service to copy only the
public root certificate into a second volume with mode `0444`. Every Go TLS
client reads that public root through `SSL_CERT_FILE`; the browser receives the
same public root through an explicit export script.

The running system is available at:

- `https://message.localhost:8443` for Message Desk and open signup.
- `https://goja.localhost:8443` for the generated go-go-goja auth-host demo.
- `https://idp.localhost:8443` for the canonical TinyIDP issuer.

The final smoke test verifies all three readiness endpoints and both OIDC login
redirects without disabling TLS verification.

## Problem Statement

The live deployment is intentionally onerous. It crosses source repositories,
container builds, GitOps review, Argo reconciliation, cert-manager issuance,
Vault materialization, ingress routing, and persistent production state. That
path is appropriate for production, but it is a poor inner development loop for
changing signup programs, client catalogs, themes, proxy policy, and OIDC
integration.

The older Compose example at
`examples/tinyidp-external-message-desk/compose.yaml` proves a single relying
party over loopback HTTP. It does not exercise:

- the strict `serve-production` host;
- HTTPS issuer identity and secure cookies;
- trusted reverse-proxy validation;
- two clients sharing one issuer;
- client-selected login/signup themes;
- the generated go-go-goja auth host and its PostgreSQL stores;
- certificate trust in application backchannels.

The new environment must therefore be convenient without becoming a second,
weaker interpretation of production. In particular, it must not solve local
TLS by setting `InsecureSkipVerify`, weakening cookies, accepting arbitrary
forwarded headers, or advertising a container DNS name as the issuer.

### Scope

This ticket includes:

- a Compose topology for Caddy, TinyIDP, Message Desk, goja auth-host, and
  PostgreSQL;
- strict HTTPS public origins under `*.localhost:8443`;
- safe distribution of Caddy's public local root;
- exact two-client and two-theme catalogs;
- open scripted signup;
- persistent local volumes and deterministic startup gates;
- a published goja image by default and an optional local-build overlay;
- operator documentation and executable smoke checks.

This ticket does not add new TinyIDP, Message Desk, or go-go-goja product
features. It does not add device authorization, multi-tenant routing, invite
administration, email delivery, Vault, production backups, or a Kubernetes
operator.

## System Foundations

### TinyIDP

TinyIDP is the OpenID Provider. It owns local accounts, password credentials,
authorization interactions, signup workflows, consent/account selection,
authorization codes, tokens, signing keys, provider sessions, audit records,
and OIDC discovery. Applications do not receive access to TinyIDP's account
database. They receive signed identity claims after a successful OAuth 2.0
authorization-code flow.

The strict host in `internal/cmds/serve_production.go` loads the client catalog,
theme catalog, signup JavaScript, SQLite store, audit sink, token secret, rate
limiter, and proxy resolver before it opens a listener. Lines 185–217 bootstrap
the reviewed clients and construct `embeddedidp.Provider` in production mode.
Lines 232–267 wrap the provider in the trusted-proxy handler and then serve
plain HTTP only inside the private network. Public transport remains HTTPS.

### Relying parties

Message Desk and goja auth-host are separate OAuth clients:

| Application | Client ID | Callback | Requested scopes |
| --- | --- | --- | --- |
| Message Desk | `tinyidp-message-app` | `https://message.localhost:8443/auth/callback` | `openid profile` |
| goja auth-host | `goja-auth-host-demo` | `https://goja.localhost:8443/auth/callback` | `openid profile email` |

Message Desk stores messages and its own opaque application session. The goja
host stores auth/session/application authorization data in PostgreSQL. Neither
application is an identity provider and neither should create TinyIDP accounts
directly.

### Canonical issuer versus network destination

OIDC gives an issuer URL identity semantics. The exact string
`https://idp.localhost:8443` appears in discovery metadata, ID-token `iss`
claims, relying-party validation, authorization URLs, and client
configuration. `idp:8081` is only a Compose transport destination used by
Caddy. Substituting it into any browser-visible or token-visible field changes
the issuer and breaks validation.

## Architecture

```text
                                      public local HTTPS
 browser ──────────────────────────────────────────────────────────┐
                                                                  │
                       :8443                                      ▼
                 ┌──────────────┐       private HTTP      ┌──────────────┐
                 │    Caddy     │ ───────────────────────▶│   TinyIDP    │
                 │ TLS + routes │                         │ :8081 strict │
                 └──────┬───────┘                         └──────────────┘
                        │
              ┌─────────┴──────────┐
              │ private HTTP       │ private HTTP
              ▼                    ▼
      ┌────────────────┐   ┌─────────────────┐      ┌────────────┐
      │  Message Desk  │   │ goja auth-host  │─────▶│ PostgreSQL │
      │     :8080      │   │      :8080      │      │   :5432    │
      └────────────────┘   └─────────────────┘      └────────────┘

 Caddy protected data volume
        │ root.crt only (read-only input)
        ▼
 ┌────────────────┐       mode 0444       ┌─────────────────────┐
 │ ca-export job  │──────────────────────▶│ public trust volume │
 └────────────────┘                       └──────────┬──────────┘
                                                   ├─▶ Message Desk
                                                   ├─▶ goja auth-host
                                                   └─▶ verified health checks
```

### Request flow

For Message Desk login, the sequence is:

```text
1. Browser -> GET https://message.localhost:8443/auth/login
2. Caddy -> Message Desk with forwarded HTTPS origin metadata
3. Message Desk generates state, nonce, and PKCE verifier/challenge
4. Message Desk -> 303 Location: https://idp.localhost:8443/authorize?...
5. Browser -> TinyIDP authorize endpoint through Caddy
6. TinyIDP validates client_id and exact redirect_uri
7. TinyIDP renders the Message Desk theme and login/signup workflow
8. Browser submits provider-owned forms
9. TinyIDP -> callback with one-time authorization code and state
10. Message Desk validates state, exchanges code over verified HTTPS,
    validates issuer/signature/nonce, then creates its local session
```

The goja flow is the same protocol with a different client ID, callback,
requested scope set, theme, and application-session implementation.

## Local Certificate Trust

### Observed failure

The first implementation mounted Caddy's data volume into both applications
and set:

```text
SSL_CERT_FILE=/caddy-data/caddy/pki/authorities/local/root.crt
```

Both applications exited during provider discovery with:

```text
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

Inspection showed the actual cause:

```text
-rw------- 1 0 0 631 ... /data/caddy/pki/authorities/local/root.crt
```

Message Desk ultimately runs as UID 10001 and the distroless goja image runs as
UID 65532. Neither could read a root-owned `0600` file. Go's Linux x509 source
confirms that `SSL_CERT_FILE` overrides the default certificate file, but
`os.ReadFile` must succeed before PEM roots are appended. The environment
variable was correct; the trust artifact boundary was not.

### Implemented trust publication contract

The `ca-export` service follows this pseudocode:

```text
wait up to 60 seconds for Caddy's public root certificate
if it exists and is non-empty:
    copy root.crt into the public trust volume
    chmod copied certificate to 0444
    exit successfully
otherwise:
    exit non-zero
```

TinyIDP depends on this job with `service_completed_successfully`. The two
relying parties depend on TinyIDP health. This ordering is supported directly
by the Compose dependency model and avoids a polling loop in application code.

Only these permissions cross the boundary:

- Caddy has read/write access to its complete PKI volume.
- `ca-export` has read-only access to Caddy PKI and write access to the public
  trust volume.
- applications have read-only access to the public trust volume.
- the browser export script copies only `caddy-local-root.crt` to an ignored
  runtime directory.

The root private key is never mounted into an application and is never copied
to the host working tree.

### Browser trust is intentionally explicit

Containers cannot safely or portably mutate the host operating system or
browser trust store. The export script produces
`runtime/caddy-local-root.crt`; the developer chooses whether to import it.
This is a security boundary, not missing automation. Installing a CA grants it
authority over TLS names trusted by that workstation.

## Network and Proxy Trust

Each backend listener accepts forwarded metadata only from a dedicated subnet:

| Backend | Proxy subnet trusted | Public origin |
| --- | --- | --- |
| TinyIDP | `172.31.0.0/24` | `https://idp.localhost:8443` |
| Message Desk | `172.32.0.0/24` | `https://message.localhost:8443` |
| goja auth-host | `172.33.0.0/24` | `https://goja.localhost:8443` |

The trusted handlers reject an untrusted immediate peer, the wrong forwarded
scheme, or the wrong forwarded host. Message Desk's listener validation in
`examples/tinyidp-message-app/commands.go:392` also requires an HTTPS public
origin and non-empty trusted CIDRs while forbidding direct TLS certificate
flags in proxy mode.

Message Desk and goja share the IdP network for verified issuer backchannel
traffic, and each also shares a distinct proxy network with Caddy. A bare
Compose service name may resolve on any shared network. The first smoke run
therefore produced `untrusted proxy peer` for Message Desk even though the
configured subnet was correct. The final Caddyfile uses network-specific
aliases (`message-proxy-backend` and `goja-proxy-backend`) so proxy traffic
cannot accidentally take the IdP network path.

## Configuration and Extension Points

### Client catalog

`clients.json` is the exact registration boundary. Adding an application means
adding a new public browser client with exact HTTPS callback and post-logout
URLs. Wildcard callbacks are not appropriate. Existing persisted client
registrations must match the desired catalog or bootstrap fails rather than
silently mutating security-sensitive fields.

### Theme catalog

`themes.json` maps the already-validated OAuth `client_id` to a reviewed,
same-origin stylesheet and product name. An application cannot supply a CSS URL
in an authorization request. This prevents arbitrary remote styling and keeps
the deployment/configuration layer responsible for approved assets.

### Signup program

`open-signup.js` uses the TinyIDP v1 scripting API. The start continuation
presents display name, email, password, and password-confirmation fields. The
submitted continuation commits the local identity and password credential.
The browser pause occurs between continuations; no Goja VM remains suspended
while waiting for user input.

Conceptually:

```javascript
start = lambda(ctx => ctx.present.form({
  fields: [displayName(), email(), password(), passwordConfirmation()],
  resume: "submitted"
}))

submitted = lambda(ctx => ctx.commit.signup({
  login: ctx.input.email,
  displayName: ctx.input.displayName,
  password: ctx.secret.password,
  passwordConfirmation: ctx.secret.passwordConfirmation
}))
```

## Design Decisions

### Decision: Preserve production-shaped HTTPS locally

- **Context:** HTTP loopback is convenient but skips secure-cookie, issuer,
  certificate, proxy, and forwarded-origin behavior.
- **Options considered:** HTTP development mode; direct TLS in every process;
  one local TLS reverse proxy.
- **Decision:** Use one Caddy TLS boundary and strict trusted-proxy listeners.
- **Rationale:** This matches the cluster topology while keeping certificate
  handling in one component.
- **Consequences:** Developers must trust one local CA; Compose needs explicit
  proxy networks and origin configuration.
- **Status:** accepted.

### Decision: Export only the public CA root

- **Context:** Non-root applications could not read Caddy's root, and mounting
  the full PKI volume would expose the CA private key.
- **Options considered:** chmod Caddy storage; mount the whole volume as root;
  disable verification; bake a repository CA; copy only the public root.
- **Decision:** A one-shot job copies only `root.crt` to a mode-0444 volume.
- **Rationale:** It gives TLS clients exactly the trust anchor they require and
  no signing capability.
- **Consequences:** Startup depends on a successful one-shot job. Destroying
  volumes rotates the CA and requires browser re-trust.
- **Status:** accepted.

### Decision: Keep browser trust manual

- **Context:** Installing a root CA mutates workstation security policy and is
  OS/browser-specific.
- **Options considered:** automatic sudo/trust-store mutation; browser TLS
  bypass; explicit export and documented import.
- **Decision:** Export the certificate, print its path, and require an explicit
  developer trust action.
- **Rationale:** The action is visible, reviewable, and reversible.
- **Consequences:** First-time setup has one manual step.
- **Status:** accepted.

### Decision: Use the published goja image by default

- **Context:** A TinyIDP clone should run without requiring a sibling goja
  checkout, while goja implementers still need local rebuilds.
- **Options considered:** mandatory cross-repository build; published image
  only; published base plus a local overlay.
- **Decision:** Pin the published image and provide
  `compose.goja-local.yaml.example` as an opt-in build overlay.
- **Rationale:** Default setup is reproducible; source iteration remains one
  explicit override.
- **Consequences:** The overlay requires the developer to choose a checkout
  path.
- **Status:** accepted.

### Decision: Do not add tools to the distroless goja runtime

- **Context:** Docker healthchecks normally run inside the target image, but
  the production goja image contains no shell, curl, or wget.
- **Options considered:** enlarge the runtime image; create a special dev
  image; omit verification; use an external smoke check.
- **Decision:** Keep the image distroless and let `02-smoke.sh` poll the public
  readiness endpoint through verified TLS.
- **Rationale:** Readiness is tested at the boundary users actually depend on,
  without changing the production artifact.
- **Consequences:** Compose reports `Up` rather than `healthy` for goja; the
  smoke script is the authoritative local readiness check.
- **Status:** accepted.

## Alternatives Considered

### mkcert-generated host certificates

mkcert is a strong alternative for a developer-wide local PKI. It installs one
local CA into supported system/browser stores and generates leaf certificates
for `*.localhost`. Caddy could mount the generated leaf certificate and key,
while applications mount only mkcert's public `rootCA.pem`.

It was not selected as the default because it is not installed on this host,
introduces an OS-specific prerequisite, and `mkcert -install` intentionally
mutates host trust. It remains attractive if several unrelated Compose stacks
need to share a single already-trusted developer CA. The private
`rootCA-key.pem` must never be committed or mounted into applications.

### Plain loopback HTTP

This remains suitable for isolated unit tests and the older one-app teaching
demo. It is inadequate as a pre-production acceptance topology because it does
not exercise secure cookies, HTTPS issuer enforcement, proxy trust, or CA
distribution.

### Disable TLS verification

`InsecureSkipVerify`, `curl -k`, and equivalent runtime switches were rejected
for application backchannels. They turn certificate mistakes into passing
tests. A diagnostic health probe may bypass validation only if a separate
verified acceptance check exists; the final IdP healthcheck and smoke script use
the exported CA.

### Commit a development CA private key

This would make setup deterministic but is unsafe. Anyone with the committed
key can issue certificates trusted by every developer who imported its root.
No CA private key belongs in source control.

## Implementation Plan and File Guide

### Phase 1: Compose topology

- `compose.yaml` declares services, volumes, startup gates, fixed subnets,
  trusted CIDRs, OIDC settings, and durable stores.
- `Caddyfile` owns the three HTTPS host routes.
- `Dockerfile.idp` builds `cmd/tinyidp` and retains the non-root entrypoint.
- `Dockerfile.message-desk` builds the existing Message Desk example.

### Phase 2: Identity configuration

- `clients.json` registers both exact browser clients.
- `themes.json`, `message-desk.css`, and `goja-auth.css` provide reviewed
  per-client presentation.
- `open-signup.js` activates the scripted open-signup workflow.

### Phase 3: Trust publication

- Caddy creates and protects its local CA.
- `ca-export` publishes only the public root into `local-ca`.
- Go processes use `SSL_CERT_FILE=/trust/caddy-local-root.crt`.
- `scripts/01-export-browser-ca.sh` exports the same public root for explicit
  workstation/browser trust.

### Phase 4: Validation and operation

- `scripts/02-smoke.sh` polls all readiness endpoints using `--cacert`.
- It asserts that each application's login route redirects to TinyIDP with the
  correct client ID.
- `README.md` documents startup, trust, reset, logs, local goja builds, and
  expected one-shot service state.

## Testing Strategy

### Static validation

```sh
docker compose -f examples/tinyidp-shared-two-apps/compose.yaml config
sh -n examples/tinyidp-shared-two-apps/scripts/*.sh
git diff --check
```

### Runtime validation

```sh
docker compose -f examples/tinyidp-shared-two-apps/compose.yaml up --build -d
examples/tinyidp-shared-two-apps/scripts/01-export-browser-ca.sh
examples/tinyidp-shared-two-apps/scripts/02-smoke.sh
docker compose -f examples/tinyidp-shared-two-apps/compose.yaml ps -a
```

The acceptance invariant is:

```text
ready(idp) && ready(message) && ready(goja)
&& message.login.client_id == "tinyidp-message-app"
&& goja.login.client_id == "goja-auth-host-demo"
&& every request verifies Caddy's exported CA
```

### Manual browser acceptance

After explicitly trusting the exported root:

1. Open Message Desk and choose account creation.
2. Confirm the provider page uses the Message Desk theme.
3. Create an account, complete the callback, and create/read a message.
4. Open goja auth-host and log in with the same TinyIDP account.
5. Confirm the provider page uses the goja theme.
6. Confirm `/me` succeeds after login.
7. Inspect the two application cookies and verify they are secure and scoped
   to their respective hosts.

## Risks and Operational Notes

- `docker compose down -v` rotates the local CA. An installed old root then
  becomes stale and the new root must be exported and trusted.
- Fixed RFC1918 subnets can collide with another local Docker network or VPN.
  If that occurs, change all four subnets and their matching trusted CIDRs as
  one atomic edit.
- The PostgreSQL password is a committed development fixture. The stack must
  never be exposed beyond the local Docker boundary.
- `*.localhost` normally resolves to loopback. A browser or resolver with
  unusual policy may require explicit host entries.
- A successful readiness smoke does not replace full browser signup and session
  acceptance; it proves topology, TLS, discovery, and authorization initiation.

## References

Repository evidence:

- `examples/tinyidp-shared-two-apps/compose.yaml`
- `examples/tinyidp-shared-two-apps/Caddyfile`
- `examples/tinyidp-shared-two-apps/README.md`
- `internal/cmds/serve_production.go`
- `examples/tinyidp-message-app/commands.go`
- `../go-go-goja/Dockerfile.auth-host`
- `../hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/clients.json`
- `../hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/themes.json`

Captured primary sources in this ticket:

- `sources/01-caddy-local-https.md` — Caddy local CA storage and trust model.
- `sources/02-mkcert-local-ca.md` — mkcert installation, trust stores, and CA
  private-key warning.
- `sources/03-docker-compose-startup-order.md` — Compose health and successful
  one-shot dependency conditions.
- `sources/04-go-system-roots-source.md` — Go's Unix `SSL_CERT_FILE` and
  `SSL_CERT_DIR` root-loading implementation.
