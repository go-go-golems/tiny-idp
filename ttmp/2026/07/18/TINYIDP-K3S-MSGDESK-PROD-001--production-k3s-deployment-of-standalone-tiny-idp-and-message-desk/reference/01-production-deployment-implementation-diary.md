---
Title: Production deployment implementation diary
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/app-deployment-pipeline.md
      Note: Canonical source-to-GHCR-to-GitOps-to-Argo release path
    - Path: abs:///home/manuel/code/wesen/2026-03-27--hetzner-k3s/docs/argocd-app-setup.md
      Note: Argo project, sync-wave, and first-bootstrap requirements
    - Path: repo://internal/fositeadapter/provider.go
      Note: Provider-owned signup intent, action validation, account creation, session binding, and consent continuation in d5927e8
    - Path: repo://internal/fositeadapter/registration_test.go
      Note: End-to-end PKCE signup and replay evidence in d5927e8
    - Path: repo://pkg/idpui/types.go
      Note: Typed registration presentation contract in d5927e8
    - Path: repo://ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/tasks.md
      Note: Authoritative phased production task ledger
ExternalSources: []
Summary: Chronological implementation, release, rollout, failure, and recovery evidence for the standalone Tiny-IDP and Message Desk production deployment.
LastUpdated: 2026-07-18T20:32:21.050406937-04:00
WhatFor: Preserve exact decisions, commands, failures, commits, review instructions, and production receipts across the multi-repository rollout.
WhenToUse: Read before resuming the ticket, reviewing a checkpoint, changing production, or executing rollback/recovery.
---



# Production deployment implementation diary

## Goal

This diary records the complete path from the existing Tiny-IDP review branch
to a public, verified, recoverable Tiny-IDP and Message Desk deployment on the
Hetzner k3s cluster. It covers both repositories, external writes, production
receipts, failures, and the reasoning behind security-sensitive decisions.

## Step 1: Establish the production project and inspect reality

The first step converted the broad instruction to “push it all the way to prod”
into a 43-task gated delivery plan. I refreshed the source and GitOps remotes,
verified that the implementation branches were clean, created a dedicated
docmgr ticket, and inspected the actual cluster through its Tailscale
kubeconfig. This established that the target is a single-node k3s cluster with
Traefik, cert-manager, Argo CD, Vault Secrets Operator, local-path
`WaitForFirstConsumer` storage, and an explicit prod-apps namespace allowlist.

The direct SSH inventory did not proceed because the recorded host key for the
older IP differs from the key presented today. I preserved strict host checking
and used the existing authenticated Kubernetes control-plane path instead. No
known-host entry was deleted and no SSH security check was bypassed.

### Prompt Context

**User prompt (verbatim):** “Create a docmgr ticket, and add detailed phases + tasks for each phase in the ticket. Then implement it step by step, committing at appropriate intervals, keeping a detailed diary. Push it all the way to prod.”

**Assistant interpretation:** Create a durable implementation project, implement the standalone provider-owned-signup deployment in phased source and GitOps commits, merge and publish both repositories, operate the production rollout, and stop short of completion until browser and recovery evidence pass.

**Inferred user intent:** Turn the accumulated Tiny-IDP research and parallel authentication work into one small real product that users can join and use, while retaining enough evidence to review and operate it safely.

### What I did

- Read the repository agent instructions and the docmgr, diary, and commit-hygiene workflows.
- Refreshed `go-go-golems/tiny-idp` and `wesen/2026-03-27--hetzner-k3s` remotes.
- Confirmed Tiny-IDP `task/prod-tiny-idp` was clean and contained twelve local review-document commits above the source already merged as PR 6.
- Confirmed the go-go-goja integration branch was clean and already pushed to `wesen`; it is not required for this initial deployment.
- Created ticket `TINYIDP-K3S-MSGDESK-PROD-001`, this diary, the implementation design, and 43 phase tasks.
- Queried the live cluster using `kubeconfig-k3s-demo-1.tail879302.ts.net.yaml`.
- Recorded the current node (`91.98.46.169`), Traefik Pod (`10.42.0.193` at observation time), local-path storage, VSO/Argo/cert-manager CRDs, ingresses, and prod-apps policy.
- Wrote the trust boundaries, invariants, phases, acceptance matrix, rollback contract, open risks, and file-review map.
- Merged refreshed Tiny-IDP `origin/main` in commit `4cce6b6` without conflicts.
- Verified the node Pod CIDR is `10.42.0.0/24`; Traefik is a single non-root Pod and records forwarded headers while dropping other headers from access-log output.
- Verified existing NetworkPolicy objects are installed and the new policy can select Traefik by its stable labels rather than an ephemeral Pod IP.

### Why

- A production rollout spans source, images, GitOps, secrets, controllers,
  runtime state, and recovery. A task ledger prevents one layer from being
  mistaken for the final outcome.
- Live cluster facts supersede examples in old documentation.
- Host-key changes are a security event until independently verified.

### What worked

- Remote refresh showed no competing source implementation.
- The authenticated Tailscale kubeconfig reached a Ready Kubernetes node and
  returned the required read-only inventory.
- The current code has a strong baseline: strict production serving, durable
  authorization interactions, account service, external PKCE relying party,
  and hardened embedded registration controls to reuse.

### What didn't work

The first remote inventory command failed exactly as follows:

```text
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
The fingerprint for the ED25519 key sent by the remote host is
SHA256:RxBj19fc/v1yXS288yXMTgpPM1DW0t3cXkq0Ucef5ps.
Offending ECDSA key in /home/manuel/.ssh/known_hosts:174
Host key verification failed.
```

The first sandboxed kubeconfig query also failed because DNS/network access was
restricted:

```text
Unable to connect to the server: dial tcp: lookup k3s-demo-1.tail879302.ts.net on 127.0.0.53:53: dial udp 127.0.0.53:53: socket: operation not permitted
```

Running the same read-only `kubectl --kubeconfig ...` query with approved
network access succeeded.

### What I learned

- The current production node is `91.98.46.169`, not the older approved SSH
  target at `89.167.52.236`.
- `prod-apps` must be extended before Argo may deploy the new namespace.
- The local-path storage class uses `WaitForFirstConsumer`, so PVC and consumer
  cannot be separated into deadlocking sync waves.
- Traefik Pod IPs are ephemeral. Proxy trust must use a verified narrow network
  contract plus NetworkPolicy rather than pinning the observed Pod address.
- The first-release proxy resolver can trust one hop from `10.42.0.0/24`, while
  the Kubernetes policy admits only the labeled Traefik Pod to public workload
  ports. Kubelet probes carry no forwarded identity and therefore do not gain
  proxy authority.

### What was tricky to build

- “Production” required separating external actions into two sources of truth:
  source artifacts in Tiny-IDP and runtime desired state in the k3s GitOps
  repository. The plan must also preserve a one-time Argo Application bootstrap
  step because this cluster does not automatically create new Applications.
- The design must allow Traefik to terminate TLS without letting arbitrary
  internal callers forge public origin or client address.

### What warrants a second pair of eyes

- Verify the registration-intent protocol shape before it becomes public API.
- Verify the actual cluster Pod CIDR, kube-router NetworkPolicy enforcement,
  and Traefik forwarded-header behavior before choosing trusted CIDRs/hops.
- Independently verify the changed SSH host key before any SSH-only operation.

### What should be done in the future

- Complete Phase 0 by merging current Tiny-IDP upstream and recording the
  verified proxy/network contract.
- Do not start the deferred device/multi-app work inside this ticket.

### Code review instructions

- Start with the design document beside this diary and `tasks.md`.
- Re-run `docmgr doctor --ticket TINYIDP-K3S-MSGDESK-PROD-001`.
- Compare the cluster observations with `gitops/projects/prod-apps.yaml` and
  the current `kubectl get` output before approving manifests.

### Technical details

```text
Tiny-IDP branch: task/prod-tiny-idp
Tiny-IDP upstream: origin/main at bbc8500 after refresh
Phase 0 merge: 4cce6b6
k3s upstream: origin/main at 8d1db2d after refresh
cluster node: k3s-demo-1 / 91.98.46.169 / k3s v1.34.5+k3s1
storage: local-path / WaitForFirstConsumer
Pod CIDR: 10.42.0.0/24; trusted proxy hop target: 1
public ingress: Traefik 3.6.9 + cert-manager letsencrypt-prod
```

## Step 2: Add provider-owned registration to durable authorization interactions

This step added account creation as a first-class required action on Tiny-IDP's
existing server-owned authorization interaction. A valid `tinyidp_signup=1`
request now produces the same kind of hashed, one-use, expiring,
browser-bound, client-generation-bound continuation that login and consent use.
The provider renders the account form, calls the existing account service,
creates the provider session, and resumes the original PKCE authorization
request. There is no application-owned password endpoint or separate redirect
state to attack.

The implementation deliberately keeps registration opt-in. Supplying an
account-service pointer without explicitly enabling registration does not open
the feature. Existing provider consumers retain their existing login-only
behavior until a host selects `Registration.Enabled`.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Implement the first source phase of the public
product while keeping security-critical lifecycle and audit behavior explicit.

**Inferred user intent:** New users should be able to join through the identity
provider and continue directly into the standalone Message Desk OIDC flow.

**Commit (code):** `d5927e8` — "feat: add provider-owned signup interactions"

### What I did

- Added `InteractionRequireRegistration` to the durable interaction action set.
- Added a strict `tinyidp_signup=1` authorization request parameter. Missing is
  normal login; duplicate or values other than exactly `1` fail with an OAuth
  `invalid_request` redirect.
- Added typed registration fields, action, public error category, page contract,
  defensive cloning, and the default HTML renderer form.
- Added opt-in registration composition in `embeddedidp.Options` and
  `fositeadapter.Options`.
- Made registration require an `idpaccounts.Service`; a custom password
  authenticator cannot accidentally enable account creation without an explicit
  account service.
- Made `createBrowserSession` return only the hashed session identifier needed
  to bind a follow-up consent interaction; the raw session value remains only in
  the outgoing HttpOnly cookie.
- Added registration-specific client/address/login-key rate-limit namespaces,
  duplicate-field rejection, password confirmation, generic public rejection,
  audit events, and best-effort byte-slice clearing.
- Added an end-to-end test covering PKCE signup, code issuance, persistent
  user creation, replay rejection, and audit field inspection.
- Added a malformed-intent test and verified the provider's normal `303 See
  Other` OAuth error redirect.
- Ran focused provider/UI/embedded tests, then the full pre-commit repository
  test and lint gates.

### Why

- The authorization interaction already owns canonical redirect/client/PKCE
  validation and has atomic consumption behavior. Reusing it avoids duplicating
  an easy-to-forget part of the OIDC security boundary.
- Keeping registration opt-in prevents a new public endpoint from appearing in
  arbitrary embedded-provider users.
- A post-registration consent page, if the configured policy requires it, is a
  new interaction bound to the just-created browser session; registration does
  not silently approve consent.

### What worked

- `go test ./internal/fositeadapter -run 'TestProviderOwnedRegistration' -count=1`
  passed.
- `go test ./internal/fositeadapter ./pkg/idpui ./pkg/embeddedidp -count=1`
  passed.
- The commit hook ran `go test ./...`, golangci-lint, Glazed CLI lint, and the
  idpui analyzer successfully.

### What didn't work

- The first sandboxed focused test could not create entries under the shared Go
  build cache; it failed with `read-only file system`. The same command passed
  with approved build-cache access.
- The initial malformed-intent test expected `302 Found`; Fosite correctly uses
  `303 See Other` for this authorization error redirect. The test now accepts
  either redirect status while asserting the stable `invalid_request` code.
- The first commit-hook process finished after its execution cell ceased
  reporting output. A status check confirmed that it had created `d5927e8`; the
  retry found a clean index and made no second commit.

### What I learned

- `idpaccounts.Service.Create` can report audit delivery failure after durable
  account mutation. The provider therefore treats any creation error as a
  generic rejection and never exposes whether a login exists. Recovery/audit
  policy still needs a product-level decision so a user is not stranded after a
  post-commit audit outage.
- A registration flow and a consent flow must not share an action vocabulary:
  `register` creates an identity, while `approve` grants a requesting client.

### What was tricky to build

- The registered session is created during the POST, but that new raw cookie is
  not available from the request object when creating a follow-up consent
  continuation. Returning only its hash from `createBrowserSession` lets the
  next interaction bind correctly without retaining the raw credential.
- An account-service pointer alone must not be the enable switch. The code
  explicitly clears the internal registration service unless
  `Registration.Enabled` is true.

### What warrants a second pair of eyes

- Review the public `tinyidp_signup` parameter name and whether it should be
  documented as a stable product extension before third parties depend on it.
- Review behavior after `idp.ErrAuditDelivery`: state can be committed before
  audit delivery fails, which is safe against credential disclosure but needs an
  operator recovery runbook.
- Review the exact registration rate limits before public exposure; current
  keys provide the correct isolation but not final production thresholds.

### What should be done in the future

- Wire `Registration.Enabled` in the standalone production host only after the
  listener, UI, abuse-control, and audit/readiness tasks are complete.
- Add explicit Origin and Fetch Metadata checks for registration POST before
  marking the full abuse-control task complete.
- Add SQLite-backed restart/race tests in addition to the memory-store protocol
  test.

### Code review instructions

- Read `beginAuthorize`, `resumeAuthorize`, and `createInteractionForSession`
  together in `internal/fositeadapter`.
- Confirm the typed `idpui.RegistrationPrompt` permits no protocol continuation
  fields other than the opaque interaction and CSRF inputs.
- Run the focused command above, then `go test ./...`.

### Technical details

```text
GET /authorize?...&tinyidp_signup=1
  -> InteractionRequireRegistration
  -> hashed interaction + CSRF + browser binding
POST /authorize action=register
  -> idpaccounts.Service.Create
  -> hashed browser session
  -> original PKCE authorization response
```

## Step 3 — Add the Message Desk provider-registration handoff

**Status:** complete

The external Message Desk deployment now gives an unauthenticated visitor a
clear “Create an account with Tiny-IDP” action. That action is a plain
top-level browser navigation to `GET /auth/register`; it contains no login,
display name, password, or account-service API. Message Desk creates its
ordinary one-use OIDC state/nonce/PKCE continuation and redirects to Tiny-IDP
with `tinyidp_signup=1`. Tiny-IDP owns the account form and, after successful
registration, returns to the normal callback, where Message Desk verifies the
ID token and establishes its own application session.

### What I did

- Refactored the OIDC client’s common authorization setup into
  `beginAuthorization`; `beginLogin` and the new `beginRegistration` differ
  only in their deliberate authorization-request parameter.
- Added `GET /auth/register`, protected by a separate
  `providerRegistrationEnabled` capability. It is not a replacement for the
  embedded demo’s local registration endpoints.
- Enabled that capability only in `openExternalMessageApplication`. External
  mode therefore keeps `/api/registration` and `/api/accounts` absent while
  exposing only the safe OIDC handoff.
- Added the new capability to the session API and updated the React view. The
  external screen explains that Tiny-IDP owns credentials and offers separate
  create-account and sign-in links.
- Rebuilt the checked-in, Go-embedded UI assets with the pinned pnpm project.
- Added unit coverage for the signup authorization URL and an HTTP-level test
  that validates the externally visible capability and redirect without
  providing an application account service.

### Why

An application should never become a second identity provider merely because
it wants a signup link. The browser needs an OIDC transaction before it can
end up at the provider registration page; the application is responsible for
the transaction's state, nonce, verifier, callback URL, and local return path.
It is *not* responsible for storing or validating the new credential. This
division leaves Tiny-IDP as the single password and account-record boundary.

```text
Message Desk browser               Tiny-IDP
--------------------               -------
GET /auth/register
  create {state, nonce, PKCE} ----> GET /authorize?...&tinyidp_signup=1
  <--- 303 Location ---------------- registration interaction
                                      POST account form
  <--- callback?code&state ---------- issue authorization code
verify ID token + create app session
```

### What worked

- `go test ./examples/tinyidp-message-app -count=1` passed when allowed to
  bind the local loopback listener used by the pre-existing browser integration
  test.
- `pnpm build` in `examples/tinyidp-message-app/ui` passed and regenerated the
  embedded static bundle.
- `git diff --check` passed before the checkpoint commit.

### What didn't work

- The first sandboxed Go test run failed only because `httptest` attempted to
  bind `[::1]` and the workspace sandbox disallows local sockets. The identical
  test passed with narrowly approved local-listener access; it did not contact
  the cluster or an external service.

### What warrants a second pair of eyes

- Confirm the copy distinguishes the standalone deployment from a general
  multi-application account portal: it intentionally says "desk account" and
  delegates identity semantics to Tiny-IDP.
- Review the final production configuration to ensure the new external-mode
  capability is paired with `tinyidp_signup` being enabled in Tiny-IDP; either
  side alone is intentionally insufficient.

### Files to review

- `examples/tinyidp-message-app/oidc_client.go` — common PKCE continuation and
  the provider-registration request parameter.
- `examples/tinyidp-message-app/app_http.go` — route boundary and exported
  capability state.
- `examples/tinyidp-message-app/external_runtime.go` — external-only feature
  selection.
- `examples/tinyidp-message-app/ui/src/App.tsx` — credential-boundary copy and
  navigation.

## Step 4 — Harden the provider registration POST boundary

**Status:** complete

Provider-owned registration already used a one-time server-side interaction,
an HttpOnly browser binding, and a CSRF token. This step makes the browser
context check explicit and bounds the form body before Go parses it. A
registration POST must now carry an `Origin` matching the request's public
scheme and host; if the browser supplies Fetch Metadata, `Sec-Fetch-Site:
cross-site` is rejected as well. The handler records a generic rejection and
does not consume the valid interaction, so a legitimate same-origin retry is
still possible.

### What I did

- Applied a 64 KiB `http.MaxBytesReader` cap to every `/authorize` POST before
  `ParseForm`; registration fields are therefore never parsed from an
  unbounded body.
- Added the provider-local `sameOriginBrowserPost` predicate for registration
  interactions. It requires `Origin`, rejects an origin different from the
  current public request origin, and rejects explicit cross-site Fetch
  Metadata.
- Kept the existing cryptographic CSRF validation in front of interaction
  lookup. The new check is defense in depth, not a replacement for the
  server-side one-use secret.
- Audited rejected provider registrations as `origin_rejected` without storing
  any submitted account fields.
- Extended the full PKCE registration test with a forged cross-site submission;
  it receives 403, creates no user, and is followed successfully by the same
  valid form submission.

### Why

CSRF tokens protect the authorization interaction even if a hostile page can
submit a form, but browser-origin validation eliminates the request earlier and
makes the policy auditable. It is deliberately scoped to the public account
creation action: OIDC client back-channel requests are not browser forms, and
the pre-existing login/consent interaction compatibility behavior remains
unchanged.

### What worked

- `go test ./internal/fositeadapter -run 'TestProviderOwnedRegistration' -count=1`
  passed with approved local loopback-listener access.
- The test confirms a cross-site request does not consume the legitimate
  registration continuation: the immediately following same-origin submission
  still completes PKCE authorization.

### What didn't work

- As with the Message Desk browser test, the initial sandboxed run could not
  bind `httptest`'s IPv6 loopback listener. It was an environment restriction,
  not a source failure; the same focused command passed with local-listener
  permission.

### What warrants a second pair of eyes

- The check derives the expected origin from the request's public Host rather
  than from a new proxy configuration field. This is correct only while the
  next listener-mode phase enforces canonical Host handling at the trusted
  proxy boundary. That dependency is explicit in the design and should be
  retained in deployment review.
- Decide whether legacy browsers without an `Origin` header must be supported.
  The first production scope rejects them for registration; Fetch Metadata is
  optional because not every browser sends it.
