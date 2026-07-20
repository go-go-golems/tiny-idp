---
Title: Production implementation and deployment plan for standalone tiny-idp and Message Desk
Ticket: TINYIDP-K3S-MSGDESK-PROD-001
Status: active
Topics:
    - auth
    - identity
    - oidc
    - security
    - operations
    - docker
    - testing
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/projects/prod-apps.yaml
      Note: Argo namespace authorization for the production deployment
    - Path: repo://examples/tinyidp-message-app/app_http.go
      Note: Message Desk routes, sessions, signup controls, and health surface
    - Path: repo://examples/tinyidp-message-app/external_runtime.go
      Note: External relying-party separation boundary
    - Path: repo://internal/cmds/serve_production.go
      Note: Strict standalone provider process and listener topology
    - Path: repo://internal/fositeadapter/provider.go
      Note: Owns the OIDC authorization interaction that scripted signup resumes
    - Path: repo://pkg/idpsignup/manager.go
      Note: Owns checked signup-program generations and activation
    - Path: repo://pkg/embeddedidp/options.go
      Note: Public composition seam for the scripted signup runtime
ExternalSources: []
Summary: 'Implement and operate the first public Tiny-IDP product slice: lambda-first scripted signup, standalone OIDC, Message Desk, immutable images, GitOps deployment, browser acceptance, and verified recovery on the Hetzner k3s cluster.'
LastUpdated: 2026-07-20T00:00:00-04:00
WhatFor: Drive the source, packaging, GitOps, production rollout, and disaster-recovery work from one gated plan.
WhenToUse: Use while implementing or reviewing TINYIDP-K3S-MSGDESK-PROD-001 and before declaring the public deployment production-ready.
---


# Production implementation and deployment plan for standalone tiny-idp and Message Desk

## Executive summary

This ticket turns the existing Tiny-IDP and Message Desk components into the
first public production slice on the Hetzner k3s cluster. The product has two
independent processes and two independent SQLite stores. Tiny-IDP owns identity,
passwords, browser authentication, OIDC protocol state, signing keys, clients,
and provider audit. Its lambda-first Goja layer selects the signup workflow
while Go retains every protocol, persistence, evidence, and atomic mutation
boundary. Message Desk owns only its application sessions and message data.
Users register at the provider and reach Message Desk through an OIDC
authorization-code flow with PKCE.

The work is complete only when immutable source artifacts have been merged,
GitOps desired state has been merged, Argo reports the application healthy, the
public browser journey passes, restarts preserve state, logs contain no
credentials, and a scratch restore proves that the recovery set works. A
successful image build or a running Pod is an intermediate result, not the
definition of production.

The initial public contract is deliberately narrow:

- issuer: `https://idp-message-desk.yolo.scapegoat.dev`;
- relying party: `https://message-desk.yolo.scapegoat.dev`;
- one public client named `message-desk`;
- `authorization_code`, PKCE S256, `openid`, and `profile` only;
- one replica for each process, each with a separate local-path PVC;
- TLS terminated by Traefik with an explicit trusted-proxy listener mode;
- a reviewed signup program mounted as non-secret deployment input;
- open signup only for protected staging; email-verified signup is the public
  production target once mail delivery is operational;
- operator-assisted recovery and an emergency registration-disable control;
- device authorization, coding agents, bearer APIs, xgoja routes, refresh
  tokens, and multiple applications are deferred.

## 1. System and trust boundaries

### 1.1 Runtime topology

```text
                                 public HTTPS
 Browser ---------------------------------------------------------------+
    |                                                                  |
    | idp-message-desk.yolo.scapegoat.dev                              |
    v                                                                  v
+------------------+        ClusterIP HTTP       +-------------------------+
| Traefik          |---------------------------->| Tiny-IDP Deployment      |
| TLS + routing    |                              | scripted signup          |
+--------+---------+                              | OIDC + identity SQLite   |
         |                                        +------------+------------+
         | message-desk.yolo.scapegoat.dev                     |
         +-----------------------------+                        | OIDC back channel
                                       v                        |
                              +-------------------------+       |
                              | Message Desk Deployment |-------+
                              | PKCE relying party      |
                              | app-session/message DB  |
                              +-------------------------+
```

Traefik is the only public listener. The two Go servers listen on unencrypted
ClusterIP HTTP only when explicitly configured for the trusted-Traefik mode.
They continue to treat their public origins as HTTPS, issue Secure cookies, and
trust forwarding metadata only from configured proxy peers.

### 1.2 Identity boundary

Tiny-IDP owns:

- usernames, display names, password hashes, subjects, and disabled state;
- OIDC clients, exact redirect/logout URIs, scopes, and grant capabilities;
- browser sessions, authorization interactions, authorization codes, and keys;
- the checked signup-program generation, explicit continuations, native
  evidence providers, atomic account commit, and provider audit events.

The Goja program is application logic, not an alternate HTTP or OAuth server.
It chooses forms, challenges, decisions, and commits through narrow host
capabilities. Go owns request parsing, CSRF, continuation lifecycle, secret
handles, password hashing, transactions, OAuth resumption, and durable audit.

Message Desk owns:

- the `(issuer, subject)` projection used by application sessions;
- local hashed session tokens and CSRF material;
- messages and their author subject/display snapshot;
- application audit, health, readiness, and cleanup.

Message Desk must never receive a provider account-store handle, administrative
credential, plaintext password, token secret, or signing key. Registration is
not implemented by proxying a password to an admin API.

### 1.3 Deployment boundary

The Tiny-IDP repository owns source, tests, Dockerfiles, GHCR publishing, and
deployment-target metadata. The k3s repository owns Kubernetes objects, Vault
and VSO shape, exact image pins, ingress, storage, probes, and the Argo
Application. The cluster owns runtime state, issued certificates, and controller
status. Production changes are durable only after the corresponding source and
GitOps commits are merged.

## 2. Security invariants

The implementation and review must preserve all of the following:

1. A signup workflow exists only inside a validated OIDC authorization
   interaction for the registered Message Desk client and an active checked
   program generation.
2. The provider stores only a hash of every browser capability handle. A handle
   is bounded, expires, is bound to browser context and client generation, and
   can be consumed once.
3. JavaScript can request only declared native operations. Atomic account
   creation ultimately uses the canonical Go service; scripts cannot
   reimplement hashing, uniqueness, rollback, or audit semantics.
4. Duplicate and invalid registrations return the same public failure shape.
5. CSRF, Origin, Fetch Metadata, address, login-key, and password-work limits
   execute before expensive or durable account mutation.
6. The successful account becomes the subject of the original authorization
   flow. Registration alone grants no application role.
7. External Message Desk exposes no local `POST /api/accounts` route and never
   executes the signup program.
8. Only configured public origins are used for issuer, redirect, logout, cookie,
   and origin validation. Forwarded `Host` cannot rewrite protocol identity.
9. Forwarding headers from an untrusted peer are ignored or rejected; a catch-
   all trusted CIDR is forbidden.
10. Both processes run as non-root, with read-only root filesystems where
    practical and owner-only mutable/secret paths.
11. Every production image is pinned by immutable `sha-<commit>` tag and is
    traceable to a CI-green source commit.
12. A backup is not accepted until restored identity and application state can
    complete a login and read an existing message.
13. Production has one exact-state bootstrap owner: Tiny-IDP process startup.
    Kubernetes does not race it with a second bootstrap Job.

## 3. Public flow contracts

### 3.1 Signup

```text
GET https://message-desk.../auth/register?return_to=/
  validate local return path
  create PKCE verifier + state + nonce in Message Desk SQLite
  redirect to Tiny-IDP /authorize with provider registration intent

GET Tiny-IDP /authorize?...registration intent...
  validate client, redirect URI, PKCE, scope, and prompt
  pin the active checked signup-program generation
  invoke the program until it presents a form or native challenge

POST Tiny-IDP scripted signup continuation
  check media type, size, browser binding, CSRF, Origin, Fetch Metadata
  consume the one-use continuation and recover only native secret handles
  resume the pinned program generation with normalized evidence
  program requests an atomic native signup commit
  establish authenticated provider session
  resume the canonical authorization request

GET Message Desk /auth/callback?code=...&state=...
  atomically consume state
  exchange code with PKCE
  verify issuer, signature, audience, expiry, nonce, and subject
  create hashed application session
  redirect to local return path
```

The signup-intent representation must be explicit and covered by tests. It may
be a provider-specific authorization parameter because registration is not an
OIDC standard endpoint, but unknown or malformed values must fail closed and it
must not weaken normal client/redirect validation. Browser waits are explicit
durable continuations: no Goja VM, goroutine, or Promise is suspended across an
HTTP round trip. Every continuation is bound to a checked program generation so
activation cannot change the meaning of an in-flight workflow.

### 3.2 Script and browser boundary

The deployment supplies a reviewed JavaScript source file. At startup Tiny-IDP
checks it, activates a generation, verifies that all declared native
capabilities are bound, and only then becomes ready. A form expression such as
the following produces an internal presentation command rather than directly
controlling the browser:

```javascript
const form = await ctx.present.form({
  fields: ["displayName", "email", "password"],
});
```

Go converts that command to an allowlisted form, stores a hashed one-use
continuation, and sends HTML. The next browser POST starts a fresh bounded
runtime invocation with normalized public values and opaque native secret
handles. Kubernetes may replace the program ConfigMap only through review and a
rollout that successfully activates the new generation.

### 3.3 Login and logout

Existing users use the same authorization endpoint without signup intent.
Application-local logout revokes only the Message Desk session. Provider logout
revokes the provider browser session through the registered end-session flow.
Both application mutations remain POST plus CSRF; navigating a GET must not
silently revoke local state.

### 3.4 Health

| Process | Liveness | Readiness |
| --- | --- | --- |
| Tiny-IDP | `/healthz` | `/readyz` checks schema, signing key, token secret, audit, maintenance, and password work |
| Message Desk | `/healthz` | `/readyz` checks the application database and durable audit output |

OIDC discovery is tested by deployment smoke, not by a frequent kubelet probe.

## 4. Persistence and recovery model

Each Deployment has `replicas: 1`, `strategy.type: Recreate`, and its own
local-path PVC. This prevents two SQLite writers and accepts that the current
single-node cluster is not highly available. PVC and first consumer remain in
the same Argo sync wave because the storage class uses `WaitForFirstConsumer`.

The recovery set consists of:

- Tiny-IDP SQLite database, including signing private keys and clients;
- Tiny-IDP audit files required by retention policy;
- the versioned Vault token secret corresponding to the backup;
- Message Desk SQLite database;
- Message Desk durable audit files;
- the source/GitOps image pins and schema versions used at backup time.

Online SQLite backups go to storage outside the k3s node. A raw copy of a live
database is not an acceptable backup. Restore validation uses scratch PVCs and
non-public Services before any recovered state replaces production.

## 5. Detailed delivery phases

### Phase 0: baseline, decisions, and evidence

Purpose: start from current upstream state and freeze the deployable contract
before changing security-sensitive code.

Tasks:

- fetch Tiny-IDP and k3s remotes and record exact starting commits;
- merge current Tiny-IDP `origin/main` without discarding the existing review
  documents;
- inspect the live cluster through its kubeconfig: Traefik Pod CIDR/address,
  forwarded-header behavior, kube-router NetworkPolicy enforcement, local-path
  storage, prod-apps allowlist, VSO CRDs, ingress naming, and Argo bootstrap;
- confirm the two hostnames and DNS/cert-manager path;
- record the no-email-verification limitation and operator recovery contract;
- record single-replica SQLite and deferred device/multi-app boundaries;
- create this plan, the task ledger, and the chronological diary.

Exit gate: `docmgr doctor` passes, the branch is based on current upstream, and
no implementation decision depends on an unverified proxy or cluster fact.

### Phase 1: lambda-first signup foundation and Message Desk initiation

Purpose: use the completed `TINYIDP-GOJA-001` foundation to let a new user
create an identity without placing provider credentials, policy wiring, or
passwords in Message Desk.

Tasks:

- reuse checked artifacts, pinned generations, explicit continuations, native
  secret handles, and atomic commit operations from `TINYIDP-GOJA-001`;
- use the shipped open-signup and email-verified-signup programs as executable
  references rather than rebuilding registration policy in handlers;
- add `/auth/register` to Message Desk as OIDC initiation only;
- render an external-mode create-account link without enabling local account
  endpoints;
- keep the legacy registration implementation out of the production command;
  it is not a compatibility contract.

Exit gate: the lambda-first ticket is complete, Message Desk can initiate
provider signup, external Message Desk has no provider store, and direct
`/api/accounts` remains 404.

### Phase 2: production listeners, audit, and bootstrap

Purpose: make the two processes honest about the cluster TLS topology and make
their desired identity state reproducible.

Tasks:

- remove `RegistrationConfig` and `--registration-enabled` from the production
  command rather than preserving a dual-path adapter;
- require a signup-program source file, check it, bind exactly its declared
  native capabilities, activate it, and fail before listening on any error;
- include active generation, required stores, native providers, signing state,
  and audit availability in readiness;
- add a required listener-mode enum with `direct-tls` and
  `trusted-proxy-http`; do not retain an ambiguous compatibility behavior;
- make direct TLS require certificate/key and trusted proxy HTTP forbid them;
- require HTTPS issuer/public origin in production proxy mode;
- configure trusted proxy CIDRs and bounded hop count through Glazed flags;
- pass the resulting resolver to provider and registration rate limiting;
- preserve Secure cookies and canonical-origin validation;
- add a durable external Message Desk audit sink and fail readiness when it is
  unavailable;
- use Tiny-IDP process startup as the sole idempotent owner of schema, one
  active signing key, and the exact public Message Desk client;
- show a non-secret bootstrap diff, allow exact/narrow reconciliation, reject
  unexpected grant/scope/redirect widening, and do not add a Kubernetes Job
  that competes for the same state;
- add validation and focused tests for script activation, native capability
  binding, listener, readiness, and bootstrap states.

Exit gate: only Traefik-trusted forwarding changes client identity, direct HTTP
misconfiguration fails before listening, invalid signup programs never become
ready, and repeated bootstrap is a no-op.

### Phase 3: local production-shaped assurance

Purpose: prove behavior across a real two-process HTTP boundary before
container and cluster variables are introduced.

Tasks:

- create a Go integration harness that launches both real servers with separate
  temporary durable state; use tmux for manual server interaction;
- test scripted signup, authorization callback, application session, feed, message
  mutation, negative CSRF, local logout, provider logout, and re-login;
- interrupt each process during a pending continuation and document whether the
  pinned flow resumes or fails cleanly;
- restart both processes and prove users, clients, signing key, messages, and
  the declared session policy persist;
- scan captured logs and audit for passwords, cookies, codes, raw tokens, and
  secrets;
- run generation, focused tests, full tests, race detector, lint, build,
  security analysis, and existing release drills.

Exit gate: the production-shaped harness and repository gates pass from a clean
checkout. Any intentionally unsupported restart behavior is explicit and safe.

### Phase 4: immutable images and source release

Purpose: produce deployable artifacts through CI rather than operator-local
builds.

Tasks:

- add two multi-stage image targets: Tiny-IDP production host and external
  Message Desk;
- mount the reviewed signup source as non-secret input while keeping mail,
  invite, token, and other native-provider credentials in owner-only files;
- build the Message Desk React assets deterministically before Go compilation;
- use non-root runtime users, minimal images, fixed working/state paths,
  stop signals, and OCI source/revision labels;
- add image-level health and owner-only-path smoke tests;
- publish `ghcr.io/go-go-golems/tiny-idp:sha-<commit>` and
  `ghcr.io/go-go-golems/tiny-idp-message-desk:sha-<commit>` from the same commit;
- add GitOps target metadata and the shared workflow caller;
- open the source PR, wait for CI, address failures within the two-attempt
  debugging rule, and merge only when green.

Exit gate: both immutable images are pullable by the cluster, report the same
source revision, and the source commit is merged.

### Phase 5: Vault and GitOps desired state

Purpose: declare all production resources and secrets without placing secret
values in Git.

Tasks:

- create an isolated k3s branch/worktree from current `origin/main`;
- add `tiny-message-desk` to the `prod-apps` AppProject;
- add namespace, distinct ServiceAccounts, two PVCs, signup-program ConfigMap,
  two Deployments, Services, Ingresses, resource requests/limits, probes, and
  `Recreate` strategies;
- create least-privilege Vault policy/role, `VaultAuth`, and
  `VaultStaticSecret` for `kv/apps/tiny-message-desk/prod/idp`;
- copy VSO material through an init container to a main-container-owned `0600`
  file without weakening `readOwnerOnlySecret`;
- wire a private-image pull secret only if GHCR package visibility requires it;
- add a NetworkPolicy allowing application ingress from Traefik and necessary
  probe/controller traffic, and allowing Message Desk egress to Tiny-IDP;
- configure Message Desk with the public HTTPS issuer and the internal
  Tiny-IDP Service as its backchannel without changing issuer identity;
- add backup scheduling/runbook and the Argo Application;
- render with `kubectl kustomize`, run schema/policy validation, open the GitOps
  PR, obtain review, and merge.

Exit gate: manifests render deterministically, contain immutable images and no
secret values, and the merged AppProject permits exactly the target namespace.

### Phase 6: production rollout and acceptance

Purpose: reconcile Git into the live cluster and test the public security
contract rather than merely controller status.

Tasks:

- seed the Vault token secret through an approved operator path without
  printing it;
- bootstrap the Argo Application once, request a hard refresh, and wait for
  `Synced` and `Healthy`;
- verify Pods, Services, PVC binding, certificates, Ingresses, discovery,
  issuer equality, headers, Secure cookies, liveness, and readiness;
- run a real browser test for signup, login, application session, message
  creation, negative CSRF, local logout, provider logout, and re-login;
- restart each Deployment and re-run persistence checks;
- inspect workload, Traefik, audit, and security logs for credential leakage;
- verify that direct node/Service exposure and untrusted forwarded headers do
  not alter the public identity contract.

Exit gate: the acceptance matrix passes against public hostnames and Argo shows
no drift. A failed mandatory scenario triggers rollback, not a production-ready
label.

### Phase 7: recovery proof and closure

Purpose: prove that accepted single-node risk has an executable recovery path.

Tasks:

- create online backups of identity and application SQLite plus required audit;
- record, without disclosing, the matching token-secret version and image pins;
- restore into scratch PVCs using non-public workloads;
- run Tiny-IDP backup verification/admin doctor and Message Desk integrity
  checks;
- complete login and read an existing message from restored state;
- document rollback, recovery ownership, retention, alerting, and the accepted
  single-node/single-replica service limitation;
- record the next device/multi-app project boundaries;
- complete diary, changelog, file relationships, doctor, and ticket closure.

Exit gate: recovery evidence is reproducible by another operator and the ticket
has no unchecked task.

## 6. Acceptance matrix

| Scenario | Required result |
| --- | --- |
| Anonymous Message Desk load | 200; public feed readable |
| Create account | Redirects into the active provider signup program inside the OIDC interaction |
| Valid scripted signup | One IdP user, one subject, one OIDC completion, one app session |
| Invalid program/capability | Tiny-IDP fails closed before readiness |
| Duplicate/invalid account | Generic rejection; no existence oracle |
| Replayed signup capability | Rejected; no second account mutation |
| Direct external `/api/accounts` | 404 |
| Existing-user login | Valid code, nonce, ID token, and app session |
| Message without session | 401 |
| Message without valid CSRF | 403; no database row |
| Message with session + CSRF | 201; durable row |
| Local logout | App session revoked; provider session may remain |
| Provider logout | Provider session ends through registered redirect |
| Wrong client/redirect/scope | Provider rejects before signup/login |
| Untrusted forwarding headers | Cannot change address, issuer, origin, or redirect behavior |
| IdP unavailable | Login fails closed; no local bypass |
| Message Desk restart | Messages and declared session policy persist |
| Tiny-IDP restart | Users, keys, client, and provider policy persist |
| Second replica | GitOps keeps replica count at one |
| Secret/log scan | No password, cookie, code, raw token, or token secret |
| Backup restore | Restored pair supports login and existing-message read |

## 7. Rollback plan

Source rollback and runtime rollback are separate operations. If the new source
image fails before a schema migration, pin both Deployments back to the previous
known-good immutable tags in a GitOps PR. If a migration has executed, restore
the matching pre-deployment recovery set or use an explicitly tested downgrade
path; never point older code at a newer database merely because Kubernetes can
start the Pod.

During initial rollout, signup can be disabled without disabling login by
deploying the reviewed disabled policy and rolling Tiny-IDP through GitOps. If
signup security fails but existing login remains safe, disable signup, preserve
evidence, and roll forward or back through Git. If identity integrity, key
material, or protocol validation is suspect, remove public ingress and stop the
issuer before attempting repair.

Argo remains the desired-state owner. Emergency `kubectl` changes must be
recorded, followed by a Git change or deliberate rollback, because self-heal
will otherwise erase them.

## 8. Operational decisions and open risks

- The two documented hostnames are accepted as the initial target unless DNS
  or certificate issuance proves they are unavailable.
- Open signup is permitted only on protected staging. Public production targets
  the email-verified reference workflow once outbound mail and its secret
  material are operational; automated password recovery remains deferred.
- Signup limits and emergency disablement require concrete defaults before
  public exposure.
- The live Traefik Pod currently uses cluster address `10.42.0.193`, but Pod
  addresses are not stable. Trust must be expressed as the narrow cluster Pod
  CIDR/hop contract and constrained by NetworkPolicy, not as one observed IP.
- The cluster has a single node and local-path storage. Backups outside that
  node are mandatory.
- The production cluster is reachable through the Tailscale kubeconfig. The
  older SSH target `89.167.52.236` presented a changed host key and is not used
  until independently verified.
- `prod-apps` does not yet allow namespace `tiny-message-desk`; GitOps must add
  it before Application sync.

## 9. Review map

Start review in these files and follow the named ownership boundaries:

- `internal/fositeadapter/provider.go` — authorization interactions and browser
  sessions;
- `pkg/embeddedidp/options.go` and `pkg/embeddedidp/provider.go` — public
  provider composition;
- `pkg/idpsignup/manager.go`, `pkg/idpsignup/open_signup.js`, and
  `pkg/idpsignup/email_verified_signup.js` — generation activation and the
  deployable reference policies;
- `pkg/idpcontinuation/` and `pkg/idpworkflow/` — durable browser/runtime seam;
- `pkg/idpaccounts/accounts.go` — the only account-creation service;
- `internal/cmds/serve_production.go` — strict provider process;
- `examples/tinyidp-message-app/oidc_client.go` — PKCE client and callback;
- `examples/tinyidp-message-app/app_http.go` — application routes and CSRF;
- `examples/tinyidp-message-app/external_runtime.go` — provider separation;
- `examples/tinyidp-message-app/commands.go` — application process listener;
- `gitops/projects/prod-apps.yaml` — namespace authorization;
- future `gitops/kustomize/tiny-message-desk/` — complete runtime topology.

## 10. References

- `TINYIDP-PROD-XGOJA-REVIEW-001/design-doc/02-initial-k3s-deployment-design-for-standalone-tiny-idp-and-message-desk.md`
- `TINYIDP-PROD-DEPLOY-001/design-doc/01-production-deployment-ergonomics-analysis-design-and-implementation-guide.md`
- `TINYIDP-GOJA-001/design-doc/02-lambda-first-goja-identity-workflow-api.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-deployment-pipeline.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-runtime-secrets-and-identity-provisioning-playbook.md`
- `/home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/argocd-app-setup.md`
