---
Title: Initial k3s deployment design for standalone tiny-idp and Message Desk
Ticket: TINYIDP-PROD-XGOJA-REVIEW-001
Status: active
Topics:
    - architecture
    - auth
    - identity
    - oauth2
    - oidc
    - operations
    - research
    - security
    - testing
    - xgoja
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-deployment-pipeline.md
      Note: Canonical source-image-GitOps-Argo deployment contract
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-kanban/deployment.yaml
      Note: Existing one-replica Recreate and local-path PVC workload pattern
    - Path: abs:///home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/infrastructure-and-release.md
      Note: User-supplied infrastructure and release system map
    - Path: repo://examples/tinyidp-external-message-desk/README.md
      Note: Existing two-process external OIDC topology and production caveats
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Current hardened signup controls, OIDC callback, sessions, health, and message API
    - Path: repo://examples/tinyidp-message-app/external_runtime.go
      Note: Existing external-OIDC Message Desk composition and disabled registration seam
    - Path: repo://examples/tinyidp-message-app/oidc_client.go
      Note: Persistent PKCE login transaction and external issuer routing
    - Path: repo://internal/cmds/serve_production.go
      Note: Standalone production tiny-idp command and current direct-TLS boundary
ExternalSources: []
Summary: Narrow first-release design for one standalone tiny-idp and one signup-enabled Message Desk deployment on the existing Hetzner k3s platform.
LastUpdated: 2026-07-18T19:45:00-04:00
WhatFor: Define the shortest implementation and GitOps path to a public two-service identity and messaging deployment while deferring device authorization and multiple relying parties.
WhenToUse: When implementing, packaging, deploying, or reviewing the initial tiny-idp and Message Desk k3s release.
---


# Initial k3s deployment design for standalone tiny-idp and Message Desk

## Executive Summary

The initial project will deploy exactly two public processes on the existing
single-node Hetzner k3s cluster:

1. A standalone tiny-idp instance that owns user accounts, passwords, browser
   login, OIDC authorization, signing keys, and self-registration.
2. A standalone Message Desk instance built from
   examples/tinyidp-message-app that trusts tiny-idp over OIDC, owns its own
   application sessions and messages, and exposes a signup entry point that
   starts the provider-owned registration flow.

This is intentionally not the broader xgoja platform described in
[the first design document](01-production-idp-architecture-and-code-review-guide-for-xgoja-applications-and-coding-agents.md).
Device authorization, coding-agent API access, bearer-token introspection,
multiple applications, dynamic client provisioning, and active-active
operation are deferred.

The existing code is close to this topology. Message Desk already supports an
external issuer, PKCE, durable login attempts, secure application sessions,
logout, health checks, and a persistent SQLite message store. Its external mode
currently disables signup because account creation still depends on an
in-process idpaccounts.Service. The first source change is therefore to move
self-registration to tiny-idp and make Message Desk initiate that OIDC-bound
signup flow.

The deployment follows the k3s repository's established contracts:

- immutable images published by the source repository;
- a Kustomize package in 2026-03-27--hetzner-k3s;
- one Argo CD Application in the prod-apps project;
- one namespace containing two one-replica Deployments;
- separate ServiceAccounts, Services, Ingresses, and PVCs;
- Traefik and cert-manager for public HTTPS;
- Vault and Vault Secrets Operator for the tiny-idp token secret;
- local-path storage with Recreate rollouts;
- and explicit backup and restore validation before calling the deployment
  production-ready.

The recommended hostnames below are placeholders to confirm before manifests
are written:

~~~text
Identity provider: https://idp-message-desk.yolo.scapegoat.dev
Application:       https://message-desk.yolo.scapegoat.dev
OIDC client ID:    message-desk
Redirect URI:      https://message-desk.yolo.scapegoat.dev/auth/callback
Logout redirect:   https://message-desk.yolo.scapegoat.dev/
~~~

## 1. Scope

### Included

- One tiny-idp issuer and identity database.
- One public OIDC client for Message Desk.
- Public account signup with login, display name, password, confirmation,
  CSRF protection, generic errors, rate limits, audit, and immediate
  continuation through OIDC.
- Browser sign-in, local application logout, provider logout, and account
  switching.
- Public reading of messages and authenticated, CSRF-protected message
  creation.
- Independent durable stores for identity state and Message Desk state.
- k3s packaging, Vault secret delivery, ingress, probes, resource limits,
  rollout strategy, backups, and a browser smoke test.
- One immutable release pipeline from the tiny-idp source repository into a
  GitOps pull request and Argo CD rollout.

### Deferred

- OAuth device authorization and coding-agent registration.
- Bearer APIs, token introspection, resource indicators, API scopes, refresh
  tokens, and agent revocation.
- More than one relying party or application.
- xgoja Express planned routes and Durable Objects.
- Active-active IdP or Message Desk replicas.
- Dynamic client registration.
- Email verification, email delivery, and password recovery.
- Social login, federation, MFA, WebAuthn, DPoP, and enterprise provisioning.

The deferred items must not leave placeholder credentials, unused public
endpoints, or permissive client grants in the initial deployment. In
particular, the Message Desk client needs only authorization_code, PKCE,
openid, and profile. Do not provision device or resource-server clients.

## 2. Current implementation baseline

### Standalone tiny-idp

internal/cmds/serve_production.go:51-205 already provides the production
process. It opens durable SQLite, a synchronous file audit sink, an owner-only
token-secret file, strict embeddedidp production mode, scheduled maintenance,
request bounds, timeouts, direct TLS, readiness, liveness, and graceful
shutdown.

The process requires a database that already has a valid active signing key and
registered browser client. docs/admin-cli.md documents the current
initialization, client, key, user, backup, and diagnostic commands.

Missing for this scope:

- A public provider-owned self-registration interaction.
- An idempotent one-client bootstrap/reconciliation command suitable for a
  Kubernetes initialization Job.
- A deliberate trusted-Traefik listener mode, because the current command
  requires its own certificate and key even when Traefik owns public TLS.

### External Message Desk

examples/tinyidp-message-app already contains the application required for this
release.

- commands.go:64-112 defines init and serve commands, including
  external-issuer and external-backchannel-url.
- external_runtime.go:11-40 opens only the application SQLite store and an
  external OIDC client. It deliberately has no identity store, account service,
  or provider handler.
- oidc_client.go:85-194 performs discovery, S256 PKCE, durable one-use state,
  nonce validation, token exchange, ID-token validation, and application
  session creation.
- app_http.go:73-92 registers login, callback, session, message, logout,
  health, readiness, and optional embedded registration routes.
- app_http.go:343-400 contains the existing hardened local-registration
  handler.
- ui/src/App.tsx:82-128 renders either the registration form or a sign-in-only
  state.

The exact seam is external_runtime.go:35-39: the handler is constructed without
accounts or a provider, and registrationEnabled is set to false. That is the
correct current behavior; an external relying party must not receive direct
access to the provider's account store merely to make signup work.

Missing for this scope:

- A Message Desk /auth/register endpoint that starts an OIDC transaction with a
  signup intent.
- A UI contract that displays “Create account” as a provider flow rather than
  trying POST /api/accounts in external mode.
- A trusted-Traefik HTTP listener mode for an HTTPS public origin.
- A durable application audit sink in external mode.

## 3. Target architecture

~~~text
                                    Existing Hetzner k3s node

Internet
   |
   | HTTPS
   v
+--------------------+    ClusterIP HTTP     +----------------------------+
| Traefik            |---------------------->| tiny-idp Deployment         |
| cert-manager cert  |                       | replicas: 1                 |
| idp-message-desk.* |                       | strict production mode      |
+--------------------+                       | /data/identity.sqlite       |
                                             | /data/audit.jsonl           |
                                             +-------------+--------------+
                                                           ^
                                                           | OIDC discovery,
                                                           | authorize, token
                                                           |
+--------------------+    ClusterIP HTTP     +-------------+--------------+
| Traefik            |---------------------->| Message Desk Deployment     |
| cert-manager cert  |                       | replicas: 1                 |
| message-desk.*     |                       | external OIDC mode          |
+--------------------+                       | /data/application.sqlite    |
        ^                                    | /data/audit.jsonl           |
        |                                    +----------------------------+
        |
      Browser

Vault -> VSO -> Kubernetes Secret -> init copy with 0600 -> tiny-idp token secret
Argo CD -> Kustomize package -> Namespace, PVCs, Jobs, Deployments, Services, Ingresses
~~~

Both public origins are HTTPS. Traefik terminates public TLS. The backends
listen on plain HTTP only in a new explicit trusted-proxy mode; normal direct
HTTP must remain rejected for production public origins.

The issuer value is always the canonical public URL. It is not a Kubernetes
Service DNS name. Message Desk may initially use the public issuer for
discovery, JWKS, token exchange, and logout to avoid inventing a certificate or
Host-rewrite boundary. A private backchannel can be introduced later if it
preserves the public issuer in every validation step.

## 4. Signup and login contract

### 4.1 Ownership

tiny-idp owns registration because it owns the identity. Message Desk owns only
the link that initiates registration, the OIDC transaction state, the resulting
application session, and messages.

Do not add an account-creation admin API credential to Message Desk. Do not let
Message Desk write the identity SQLite database. Do not proxy password JSON
through the relying party to the provider.

### 4.2 Proposed browser flow

~~~text
Browser          Message Desk              tiny-idp               stores
   |                  |                         |                     |
   | GET /            |                         |                     |
   |----------------->| session says guest + signup available         |
   | Create account   |                         |                     |
   |----------------->| GET /auth/register      |                     |
   |                  | create state, nonce, PKCE in app SQLite       |
   |                  | 302 /authorize?screen_hint=signup ----------->|
   |                  |                         | persist interaction  |
   |<-------------------------------------------| registration form   |
   | POST registration + bound CSRF ---------->|                     |
   |                  |                         | rate limit           |
   |                  |                         | accounts.Create ---->|
   |                  |                         | create IdP session    |
   |                  |                         | resume authorization  |
   |<-------------------------------------------| code to callback     |
   | GET /auth/callback                         |                     |
   |----------------->| POST /token ----------->|                     |
   |                  | verify nonce/ID token   |                     |
   |                  | create app session -------------------------->|
   |<-----------------| app cookie, redirect /  |                     |
~~~

screen_hint=signup is a proposed tiny-idp extension carried only on the
authorization request. It is not a claim of an OpenID Connect standard
parameter. The important property is not the spelling: the signup continuation
is an already-validated OIDC authorization interaction with a registered
client, redirect URI, state, nonce, and PKCE verifier.

### 4.3 Provider registration controls

Move the security properties of the current Message Desk registration handler
to the provider interaction:

- one-use, expiring registration attempt bound to the pending authorization
  interaction;
- provider-owned CSRF token and HttpOnly SameSite cookie;
- POST-only mutation with exact media type and bounded body;
- same-origin Origin and Sec-Fetch-Site checks;
- rate limits by trusted client address and normalized-login hash;
- the existing idpaccounts password policy and bounded Argon2id work;
- generic duplicate/password rejection response;
- atomic user and password-credential creation;
- durable audit for accepted, rejected, and unavailable outcomes;
- no email verification claim unless verification actually occurred;
- and immediate provider login only for the account just created.

The interaction must survive a pod restart because the identity store is
durable. A process-local registration attempt would create avoidable failures
during a rollout.

### 4.4 Message Desk UI contract

Replace the ambiguous registrationEnabled boolean with an explicit
registration mode in the session response:

~~~json
{
  "authenticated": false,
  "registration": {
    "mode": "provider",
    "url": "/auth/register?return_to=/"
  }
}
~~~

Embedded development mode may use mode local and the existing form endpoints.
The k3s external deployment uses mode provider. Disabled signup uses mode none.
This is a product contract, not a backwards-compatibility shim; update the
frontend and tests together.

The new /auth/register handler should call the same internal OIDC attempt
builder as /auth/login but request signup presentation. It must still normalize
return_to to a local absolute path.

## 5. Kubernetes and GitOps design

### 5.1 Source repository delivery

The tiny-idp repository owns:

- both Go binaries;
- the Message Desk React build embedded by go generate;
- Dockerfiles or one multi-target Dockerfile;
- go test ./..., go build ./..., and frontend build validation;
- immutable GHCR image publication;
- deploy/gitops-targets.json;
- and the shared workflow caller that opens a GitOps image-pin PR.

Publish two minimal images from the same source commit:

~~~text
ghcr.io/go-go-golems/tiny-idp:sha-<commit>
ghcr.io/go-go-golems/tiny-idp-message-desk:sha-<commit>
~~~

Two images keep entry points and runtime contents explicit. Pin both immutable
tags in the GitOps PR. Convenience main/latest tags are not deployment inputs.

### 5.2 GitOps repository layout

Add:

~~~text
gitops/applications/tiny-message-desk.yaml
gitops/kustomize/tiny-message-desk/
  namespace.yaml
  serviceaccount-idp.yaml
  serviceaccount-app.yaml
  vault-connection.yaml
  vault-auth-idp.yaml
  token-secret.yaml
  pvc-idp.yaml
  pvc-app.yaml
  bootstrap-configmap.yaml
  bootstrap-job.yaml
  deployment-idp.yaml
  deployment-app.yaml
  service-idp.yaml
  service-app.yaml
  ingress-idp.yaml
  ingress-app.yaml
  networkpolicy.yaml
  kustomization.yaml
~~~

One Argo Application owns the initial two-process product. Assign it to
prod-apps, set the destination namespace to tiny-message-desk, and add that
namespace to the project allowlist if necessary. Required labels include:

~~~yaml
scapegoat.dev/tier: app
scapegoat.dev/source-type: kustomize
scapegoat.dev/has-database: "true"
scapegoat.dev/has-persistent-storage: "true"
scapegoat.dev/has-ingress: "true"
scapegoat.dev/database-type: embedded
~~~

Apply gitops/applications/tiny-message-desk.yaml once. This cluster does not
currently auto-create new Application objects merely because their YAML was
merged.

### 5.3 Workloads and storage

Use two Deployments with:

- replicas: 1;
- strategy.type: Recreate;
- enableServiceLinks: false;
- non-root containers;
- read-only root filesystem where the binaries permit it;
- explicit CPU/memory requests and memory limits;
- terminationGracePeriodSeconds greater than the server shutdown timeout;
- and separate local-path ReadWriteOnce PVCs.

Put each PVC in the same Argo sync wave as its first consuming Job or
Deployment. The cluster's local-path class uses WaitForFirstConsumer; an earlier
PVC wave can deadlock while Argo waits for a pod that it has not created.

Suggested mounts:

~~~text
tiny-idp:
  /var/lib/tinyidp/idp.sqlite
  /var/lib/tinyidp/audit.jsonl
  /run/tinyidp/token-secret        ephemeral owner-only copy

Message Desk:
  /var/lib/message-desk/state.json
  /var/lib/message-desk/application.sqlite
  /var/lib/message-desk/audit.jsonl
~~~

The identity and application databases must never share a file. No second pod
may mount either database as an active writer.

### 5.4 Initialization

Add an idempotent tiny-idp bootstrap CLI backed by embeddedidp.Bootstrap. The
initialization Job should:

1. Open/migrate the identity database.
2. Ensure one active production signing key exists.
3. Reconcile exactly one public Message Desk browser client.
4. Validate the exact redirect URI, logout URI, scopes, grant type, and PKCE
   requirement.
5. Run admin doctor.
6. Exit successfully when the desired state already exists.
7. Fail on an unexpected widening or conflicting client definition.

Proposed invocation:

~~~text
tinyidp admin --db /var/lib/tinyidp/idp.sqlite bootstrap-browser-client
  --id message-desk
  --redirect-uri https://message-desk.yolo.scapegoat.dev/auth/callback
  --post-logout-redirect-uri https://message-desk.yolo.scapegoat.dev/
  --scope openid
  --scope profile
  --require-pkce
  --signing-key-id message-desk-initial
~~~

The Message Desk initialization Job runs:

~~~text
tinyidp-message-desk init
  --state-root /var/lib/message-desk
  --public-base-url https://message-desk.yolo.scapegoat.dev
~~~

It must be safe on an existing state root. External mode then runs:

~~~text
tinyidp-message-desk serve
  --state-root /var/lib/message-desk
  --addr :8080
  --external-issuer https://idp-message-desk.yolo.scapegoat.dev
  --listener-mode trusted-proxy-http
~~~

### 5.5 Secrets

Store the token secret at:

~~~text
Vault KV: kv/apps/tiny-message-desk/prod/idp
key:      token_secret
~~~

Use a tiny-idp-specific Vault policy, Kubernetes auth role, ServiceAccount,
VaultAuth, and VaultStaticSecret. The policy must not read other application
paths.

Kubernetes Secret volumes are commonly group/world-readable according to their
mounted mode, while serve-production rejects token-secret files with any group
or other permission bits. Use an init container to copy the VSO source into an
emptyDir, set ownership to the main process UID, and chmod 0600. Do not weaken
readOwnerOnlySecret.

Message Desk uses a public PKCE client and therefore has no OIDC client secret.
Do not create one merely because other cluster applications use confidential
Keycloak clients.

If GHCR images are private, add the existing Vault-backed image pull secret
pattern and separate that credential from runtime secrets.

### 5.6 Ingress and trusted proxy mode

The cluster's standard public path is Traefik TLS termination. Add an explicit
production listener mode to tiny-idp and Message Desk rather than silently
allowing HTTP for any HTTPS public URL.

The trusted-proxy mode must:

- require a canonical HTTPS issuer/public base URL;
- keep Secure cookies enabled;
- require configured trusted proxy CIDRs and a bounded hop count;
- accept client-address forwarding only from trusted peers;
- reject direct public exposure by design and documentation;
- keep issuer, redirect, and origin validation configuration-driven rather than
  deriving them from Host;
- and be paired with a ClusterIP Service and NetworkPolicy allowing ingress
  from Traefik plus the required probe path.

The application code should receive a Glazed field/flag. It must not infer the
mode from an environment variable.

### 5.7 Probes

Use:

| Workload | Liveness | Readiness |
| --- | --- | --- |
| tiny-idp | /healthz | /readyz |
| Message Desk | /healthz | /readyz |

Message Desk readiness must fail when application SQLite is unavailable. The
standalone provider readiness must fail for schema, signing key, token secret,
audit, maintenance, or password-work failures. OIDC discovery should be
checked by the post-deployment smoke test, not by a high-frequency kubelet
probe.

## 6. Persistence, backup, and recovery

The cluster is single-node and local-path volumes are tied to that node. Git and
Vault cannot recreate SQLite user/message data. A node loss without a verified
backup loses the product.

Before production labeling:

1. Schedule tinyidp admin backup create against the live identity database.
2. Back up the Message Desk SQLite database using an online SQLite-safe
   mechanism, not a raw copy during writes.
3. Include audit files according to the selected retention policy.
4. Store backups in the cluster object-storage backup account, not on the same
   local disk.
5. Record the Vault token-secret version associated with the backup.
6. Restore both databases into scratch PVCs.
7. Verify the tiny-idp backup with admin backup verify and admin doctor.
8. Start both scratch processes on non-public endpoints and complete login plus
   message reads.

The signing private key is stored with identity state. Restoring only a new
empty identity database while retaining an old application database produces
orphaned subjects and unusable login. Restoring identity state without the
matching token secret invalidates server-side security artifacts. Treat these
as one recovery set even though Vault and object storage hold separate pieces.

## 7. Delivery phases

### Phase 1: source separation and provider signup

- Add provider-owned registration interaction and durable tests.
- Add Message Desk provider-registration mode and /auth/register.
- Keep POST /api/accounts unavailable in external mode.
- Add explicit trusted-proxy listener fields to both commands.
- Add a durable external-mode application audit sink.
- Add the idempotent one-client bootstrap command.

Exit criteria:

- A two-process integration test signs up a new account through the provider,
  returns through OIDC, creates a Message Desk session, and posts a message.
- Restart either process during a pending interaction and document the expected
  recovery behavior.
- No password or provider account-store handle reaches Message Desk.

### Phase 2: images and CI

- Add reproducible multi-stage images for both binaries.
- Run go generate ./..., go test ./..., go build ./..., and the Message Desk
  frontend build in CI.
- Publish two immutable same-commit image tags.
- Add the GitOps target file and shared workflow caller.

Exit criteria:

- A pull request builds both images without publishing.
- A main-branch release publishes both exact tags.
- The updater produces a GitOps PR changing only the two image pins.

### Phase 3: Vault and GitOps

- Add the namespace, service accounts, Vault policy/role/resources, PVCs,
  bootstrap Job, Deployments, Services, Ingresses, NetworkPolicy, and Argo
  Application.
- Seed the token secret without printing its value.
- Render with kubectl kustomize.
- Bootstrap the Argo Application once.

Exit criteria:

- Argo reports Synced and Healthy.
- Both pods are ready with one replica and Recreate strategy.
- The public issuer discovery document reports the exact HTTPS issuer.

### Phase 4: end-to-end validation

- Complete browser signup.
- Confirm provider login, Message Desk app session, public feed, authenticated
  message creation, CSRF rejection, local logout, provider logout, and
  re-login.
- Restart each pod and confirm identities, sessions according to policy, and
  messages persist.
- Exercise backup and scratch restore.
- Confirm Loki/container logs and audit files contain no passwords, cookies,
  authorization codes, token secrets, or session tokens.

Exit criteria:

- The complete acceptance matrix below passes against the public hostnames.
- The restore drill produces a readable message and a successful login.

## 8. Acceptance matrix

| Scenario | Expected result |
| --- | --- |
| Anonymous GET / | Message Desk loads and public messages are readable |
| Create account link | Starts a PKCE OIDC request with signup intent |
| Valid signup | Creates one IdP user, completes OIDC, creates one app session |
| Duplicate login | Generic rejection without confirming account existence |
| Weak password | Generic rejection; no user or credential row |
| Reused registration CSRF/attempt | Rejected; exactly one account mutation |
| Direct POST to Message Desk /api/accounts in external mode | 404 |
| Login with existing user | Valid code, nonce, ID token, and app session |
| Post message without session | 401 |
| Post message without/wrong CSRF | 403 and no row |
| Post message with session and CSRF | 201 and durable row |
| Local logout | App cookie revoked; IdP browser session may remain |
| Provider logout | App cookie revoked and IdP end-session completed |
| Wrong redirect URI/client | Provider rejects before signup/login |
| IdP unavailable during login | Message Desk fails closed without local login |
| Message Desk restart | Message and application session policy persist |
| IdP restart | User, key, client, and provider session policy persist |
| Second replica attempted | GitOps policy keeps replicas at one |
| Backup restore | Restored pair supports login and existing message read |

Validation commands include:

~~~text
go generate ./...
go test ./...
go build ./...
kubectl kustomize gitops/kustomize/tiny-message-desk
kubectl -n argocd get application tiny-message-desk
kubectl -n tiny-message-desk get deploy,pod,svc,ingress,pvc,job
curl -fsS https://idp-message-desk.yolo.scapegoat.dev/.well-known/openid-configuration
curl -fsS https://message-desk.yolo.scapegoat.dev/healthz
curl -fsS https://message-desk.yolo.scapegoat.dev/readyz
~~~

Use a real browser automation test for cookies, redirects, CSP, signup, and
logout. Curl alone cannot validate the browser security contract.

## 9. Decisions

### Decision: deploy Message Desk, not the xgoja BBS prototype

- **Context:** The initial request was clarified to mean
  examples/tinyidp-message-app.
- **Options considered:** Deploy cmd/tinyidp-xapp, deploy Message Desk, or build
  a new app.
- **Decision:** Deploy Message Desk in external-OIDC mode.
- **Rationale:** It already contains the required message model, React UI,
  PKCE client, sessions, signup controls, external issuer support, health
  endpoints, and two-container development proof.
- **Consequences:** xgoja, Durable Objects, device APIs, and Express auth syntax
  are outside this release.
- **Status:** accepted.

### Decision: provider-owned signup

- **Context:** External Message Desk has no safe access to the provider's
  account service and intentionally disables local signup.
- **Options considered:** Add an IdP admin API credential to the app, proxy
  passwords through the app, or implement provider-owned signup.
- **Decision:** Implement provider-owned signup bound to the OIDC authorization
  interaction.
- **Rationale:** Passwords and identity mutations stay within the IdP trust
  boundary, while OIDC already provides the safe continuation to the app.
- **Consequences:** tiny-idp needs new UI/interaction/state tests; Message Desk
  needs only an initiation route and presentation contract.
- **Status:** proposed.

### Decision: one namespace and one Argo Application

- **Context:** The product has two processes but is one initial deployment.
- **Options considered:** Two namespaces/Applications or one package with
  separate ServiceAccounts.
- **Decision:** Use one namespace and Argo Application, with distinct
  Deployments, PVCs, Services, and service accounts.
- **Rationale:** This minimizes first-rollout GitOps machinery while retaining
  process, storage, and secret separation.
- **Consequences:** Namespace administrators can see both workloads. Split them
  later only for a demonstrated policy or ownership requirement.
- **Status:** proposed.

### Decision: explicit trusted-Traefik listener mode

- **Context:** Both current commands require direct TLS for HTTPS public origins,
  while the cluster standard terminates public TLS at Traefik.
- **Options considered:** Mount certificates and re-encrypt to each pod, weaken
  the HTTPS check, or implement an explicit trusted-proxy mode.
- **Decision:** Add an explicit production proxy-listener mode with secure
  cookies, canonical HTTPS origins, trusted address parsing, ClusterIP-only
  Services, and NetworkPolicy.
- **Rationale:** It matches the existing cluster topology without silently
  turning local HTTP into a production configuration.
- **Consequences:** Proxy trust and source CIDRs require live cluster
  validation. Direct TLS mode remains available for non-proxy deployments.
- **Status:** proposed.

### Decision: SQLite and one replica for both processes

- **Context:** Both current stores are SQLite and the k3s node provides
  local-path PVCs.
- **Options considered:** PostgreSQL, shared network storage, or single-writer
  local SQLite.
- **Decision:** Use distinct local-path PVCs, replicas: 1, and Recreate.
- **Rationale:** This is supported by current code and is the shortest first
  deployment.
- **Consequences:** No HA; node recovery depends on verified object-storage
  backups.
- **Status:** accepted for the initial release.

## 10. Risks and open questions

- Confirm the two public hostnames before creating the client or certificates.
- Determine the actual Traefik source CIDRs/hop behavior before setting trusted
  proxies; do not use an unreviewed catch-all CIDR.
- Confirm the cluster NetworkPolicy controller enforces the intended ingress
  restrictions.
- Decide whether “initial deployment” may launch without email verification or
  password recovery. This design permits that only if the limitation is
  explicit to users and operator-assisted recovery exists.
- Decide signup abuse thresholds and how operators disable registration during
  an incident.
- Decide audit retention and whether file audit must be shipped to Loki or
  separately archived.
- Define the object-storage prefix, schedule, retention, encryption, and alert
  policy for the two SQLite backups.
- Verify that the prod-apps AppProject allows the new namespace.
- Decide whether both GHCR packages are public; private packages require the
  established Vault-backed pull-secret path.
- The cluster and both workloads are single-node/single-replica. This is an
  accepted availability limit, not an accidental omission.

## 11. File references

### tiny-idp repository

- internal/cmds/serve_production.go:51-205 — standalone strict server.
- docs/admin-cli.md — database, clients, users, keys, backup, and diagnostics.
- examples/production-host/README.md — production single-writer contract.
- examples/tinyidp-message-app/commands.go:64-112,281-355 — init/serve and
  external mode selection.
- examples/tinyidp-message-app/external_runtime.go:11-40 — external separation
  and signup-disabled seam.
- examples/tinyidp-message-app/external_config.go:10-108 — issuer, public
  origin, cookie, and backchannel validation.
- examples/tinyidp-message-app/oidc_client.go:85-194 — PKCE and callback.
- examples/tinyidp-message-app/app_http.go:59-242,343-480 — routes, health,
  signup controls, sessions, and audit.
- examples/tinyidp-message-app/ui/src/App.tsx:82-128 — signup/sign-in UI.
- examples/tinyidp-external-message-desk/README.md — existing two-container
  development topology and production caveats.

### k3s and knowledge base

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/cluster-architecture-overview.md
  — single-node k3s, Argo, Traefik, Vault/VSO, and local-path topology.
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-deployment-pipeline.md
  — source image, GitOps PR, immutable tags, and first Application bootstrap.
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md
  — Vault/VSO and rollout ordering.
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/argocd-app-setup.md
  — project labels, Kustomize layout, sync waves, and validation.
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-kanban/
  — current one-replica, Recreate, local-path workload example.
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/kustomize/goja-auth-host-demo/
  — current OIDC/Vault/Ingress workload example.
- /home/manuel/code/wesen/go-go-golems/go-go-parc/Research/KB/Projects/infrastructure-and-release.md
  — cross-repository release and platform map.
