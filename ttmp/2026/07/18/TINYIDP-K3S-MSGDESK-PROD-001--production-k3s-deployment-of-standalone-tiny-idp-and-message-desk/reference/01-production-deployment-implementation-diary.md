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
    - Path: repo://internal/cmds/serve_production.go
      Note: |-
        Explicit listener modes and exact Message Desk bootstrap
        Production Goja signup program loading and lifecycle
    - Path: repo://internal/cmds/serve_production_test.go
      Note: Production program contract tests
    - Path: repo://internal/fositeadapter/provider.go
      Note: |-
        Provider-owned signup intent, action validation, account creation, session binding, and consent continuation in d5927e8
        Scripted signup account-service composition
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

## Step 5 — Establish the trusted-proxy primitive for listener modes

**Status:** in progress

The cluster terminates public TLS at Traefik and reaches the application Pods
over HTTP. The processes therefore need an explicit mode that accepts
forwarding metadata only from Traefik, not a switch that simply disables TLS.
The shared `idp.TrustedProxyResolver` is the foundation for that mode: it now
can answer whether the immediate TCP peer is trusted, which listener middleware
will use before accepting any transport-security header.

### What I did

- Added `TrustsRequestPeer` to the resolver. It intentionally examines only
  the immediate socket peer and does not turn an `X-Forwarded-For` value into a
  proxy-trust decision.
- Rejected `0.0.0.0/0` and `::/0` trusted-proxy CIDRs at construction time.
  The deployment will use the observed k3s Pod CIDR contract, not a broad
  internet-wide trust declaration.
- Added focused tests for catch-all rejection and trusted/untrusted immediate
  peer classification.

### What worked

- `go test ./pkg/idp -count=1` passed.

### Next

- Build the explicit `direct-tls` and `trusted-proxy-http` listener mode on
  top of this primitive, with canonical configured origins and no use of
  forwarded Host to rewrite OIDC identity.

## Step 6 — Add a reusable trusted-proxy HTTPS request contract

**Status:** complete foundation; command wiring follows

I added `idp.NewTrustedProxyHTTPHandler`, the boundary used by both production
processes when Traefik performs public TLS termination. It accepts a request
only when all of these facts agree: the immediate peer is in the configured
proxy CIDR, exactly one `X-Forwarded-Proto` says `https`, the request Host is
the configured public host, and any forwarded Host agrees with that value.
It does not derive identity from either forwarded header.

After verification, the wrapper gives downstream handlers a cloned request
with a TLS marker and HTTPS URL scheme. This is important for existing secure
cookie and same-origin behavior: the Go application can honestly operate as a
public HTTPS service even though its Pod socket is HTTP.

### What worked

- `go test ./pkg/idp -count=1` passed.
- The contract tests prove rejection for an untrusted immediate peer, an HTTP
  forwarded transport claim, a mismatched request Host, and a mismatched
  forwarded Host.

### Why this is not the final listener mode yet

The wrapper deliberately does not bind a port or choose TLS. Each command must
next make an explicit, mutually exclusive choice:

```text
direct-tls          => certificate/key required, ListenAndServeTLS
trusted-proxy-http  => certificate/key forbidden, wrapped ListenAndServe
```

Keeping that choice in command construction prevents a Deployment from
silently changing security semantics merely because an optional certificate
file is absent.

## Step 7 — Wire Tiny-IDP’s production command to explicit listener modes

**Status:** Tiny-IDP complete; Message Desk equivalent next

`tinyidp serve-production` now requires `--listener-mode`. `direct-tls` uses
`ListenAndServeTLS`, requires both certificate paths, and rejects proxy CIDRs.
`trusted-proxy-http` uses plain internal `ListenAndServe`, requires an HTTPS
issuer and narrow trusted-proxy CIDRs, rejects certificate paths, and wraps the
provider with the verified forwarding contract. The trusted resolver is also
the rate-limit client-address resolver in that mode.

Focused command tests cover absent/unknown modes and the mutually exclusive
direct-TLS and trusted-proxy configurations.

## Step 8 — Wire Message Desk to the same listener contract

**Status:** implementation checkpoint

Message Desk `serve` now requires the same `direct-tls` or
`trusted-proxy-http` selection. In trusted-proxy mode it requires its HTTPS
public base URL and narrow proxy CIDRs, forbids certificate flags, and wraps
the application handler in the shared verified-proxy contract. This means its
existing Secure cookie and same-origin checks see validated public HTTPS.

The focused Message Desk command test passes. The full hook will run with this
checkpoint; the next task is to expand listener-mode test cases and wire the
external audit/bootstrap configuration.

## Step 9: Backfill the production-listener and external-runtime checkpoints

This entry backfills the ledger after the user asked that the implementation
diary stay detailed. The code was committed in small, independently tested
checkpoints: shared proxy peer validation (`5a728f6`), verified HTTPS proxy
request handling (`56a6ac7`), Tiny-IDP mode wiring (`d707c74`), Message Desk
mode wiring (`17bbd11`), Message Desk mode tests (`d00ccc2`), and durable
external-runtime auditing (`f394dd5`).

Together these changes replace ambiguous "HTTPS public origin but HTTP Pod"
behavior with an explicit contract. Both services now distinguish direct TLS
from Traefik-terminated TLS, and the external Message Desk has a real audit
sink rather than a no-op sink.

### Prompt Context

**User prompt (verbatim):** "keep a diary, backfill whatever is necessary (look at skill for format)"

**Assistant interpretation:** Record the already-completed implementation
checkpoints in the ticket diary using the required structured format.

**Inferred user intent:** Another implementer must be able to reconstruct the
security decisions, test evidence, commits, and remaining deployment work.

**Commit (code):** `f394dd5` — "feat: audit external Message Desk runtime"

### What I did

- Recorded the listener/audit commit sequence above and the reason each piece
  was separated.
- Added `idp.TrustedProxyResolver.TrustsRequestPeer` and forbade catch-all
  proxy CIDRs.
- Added `idp.NewTrustedProxyHTTPHandler`, which verifies peer, forwarded HTTPS,
  canonical Host, and any forwarded Host before representing a request as TLS.
- Added required `direct-tls`/`trusted-proxy-http` modes to Tiny-IDP and
  Message Desk; each mode rejects the other mode's inputs.
- Made external Message Desk validate its state root and open its durable JSONL
  audit file; that sink participates in close/readiness behavior.

### Why

Traefik TLS termination must not make a Pod pretend that arbitrary local HTTP
is public HTTPS. The public origin remains configured; headers are evidence
accepted only from the narrow proxy network.

### What worked

- Focused tests passed for `./pkg/idp`, `./internal/cmds`, and
  `./examples/tinyidp-message-app`.
- Every listed commit's Lefthook ran repository-wide `go test ./...`,
  golangci-lint, Glazed lint, and the idpui analyzer successfully.

### What didn't work

- Sandboxed Go test commands repeatedly failed when the shared Go build cache
  or `httptest` loopback listener was inaccessible. Re-running the identical
  focused command with approved local build-cache/loopback access passed.
- During Message Desk listener wiring, compilation reported exactly
  `declared and not used: secureOrigin`; the obsolete inferred-TLS variable was
  removed, then the focused test passed.

### What I learned

- The existing `embeddedidp.Bootstrap` already performs exact client-state
  reconciliation and active-signing-key creation; the production command can
  call it rather than duplicating client comparison rules.

### What was tricky to build

- Browser origin checks see `r.TLS`, while a Traefik-to-Pod connection has no
  socket TLS. The wrapper clones only a request that has passed proxy/HTTPS/host
  validation and sets its TLS marker; it never trusts forwarded Host to rewrite
  the configured protocol identity.

### What warrants a second pair of eyes

- Review the chosen Traefik CIDR in GitOps against the cluster's actual stable
  Pod/network topology. The code correctly rejects broad trust, but deployment
  chooses the concrete allowed range.
- Review the exact browser-client identifier (`tinyidp-message-app`) before
  public deployment; it is the current compiled Message Desk OIDC client ID.

### What should be done in the future

- Complete and test the production command's bootstrap invocation, then mark
  listener/bootstrap tasks only when both processes have end-to-end coverage.
- Add containers, CI image publishing, GitOps manifests, Argo sync, and public
  acceptance evidence; none has been performed yet.

### Code review instructions

- Start at `pkg/idp/trusted_proxy_http.go`, then read
  `internal/cmds/serve_production.go` and
  `examples/tinyidp-message-app/commands.go` together.
- Validate with `go test ./pkg/idp ./internal/cmds
  ./examples/tinyidp-message-app -count=1`, then `go test ./...`.

### Technical details

```text
trusted-proxy-http request
  immediate peer ∈ configured CIDR
  X-Forwarded-Proto == https
  Host == configured public host
  X-Forwarded-Host absent or equal
  => cloned request is safe to treat as public HTTPS
```

## Step 10: Re-baseline production delivery on lambda-first scripted signup

The production plan had fallen behind the implementation. It still described
the original hardcoded provider-registration handlers as the intended public
workflow even though `TINYIDP-GOJA-001` had completed the lambda-first signup
system. It also left completed trusted-Traefik work unchecked and assigned
exact-state bootstrap to both process startup and a future Kubernetes Job.

I updated the design and task ledger rather than changing runtime code. The new
baseline gives the production command one signup path, gives bootstrap one
owner, and orders the remaining work from production composition through local
two-process proof, images, GitOps, public acceptance, and recovery.

### Prompt Context

**User prompt (verbatim):** "perfect, rebase / update the design doc / update the task list."

**Assistant interpretation:** Reconcile the existing Message Desk production
ticket with the completed Goja work and the implementation already present on
the branch, then rewrite its design and tasks to be executable from here.

**Inferred user intent:** Resume the original production deployment without
rebuilding obsolete signup primitives or losing track of completed listener,
audit, and bootstrap work.

### What I did

- Replaced the hardcoded production-registration plan with checked,
  generation-pinned scripted signup.
- Recorded the browser/runtime seam: Go renders allowlisted presentations and
  resumes fresh bounded invocations from durable one-use continuations.
- Made production command startup the sole exact-state bootstrap owner and
  removed the planned competing Kubernetes bootstrap Job.
- Checked off listener, trusted-proxy, Secure-cookie, Message Desk audit, and
  existing startup-bootstrap tasks supported by the implementation diary.
- Added remaining tasks for program loading, native capability binding,
  readiness, two-process acceptance, ConfigMap delivery, backchannel wiring,
  and provider secrets.

### Why

- The deployment plan must describe the architecture that will actually ship.
- Maintaining both the legacy registration path and scripted signup would
  create two security surfaces and an unrequested compatibility contract.
- Two bootstrap owners could race or disagree about client and signing state.

### What worked

- The ticket already contained commit-level evidence for the completed
  listener, proxy, audit, and startup-bootstrap checkpoints.
- The Goja ticket provides shipped open and email-verified programs plus the
  generation, continuation, commit, readiness, and native-provider primitives.

### What didn't work

- The first documentation patch failed with `apply_patch verification failed`
  because the target paragraph had a different line wrap. I split the change
  into smaller exact-context patches; the failed patch applied no changes.

### What I learned

- Phase 1 now represents completed Goja foundations plus Message Desk
  initiation. Production composition is the new front of Phase 2.
- The production command already owns exact client and active-key bootstrap,
  so Kubernetes should supply inputs and storage, not duplicate the mutation.

### What was tricky to build

- This was a semantic rebase, not merely checkbox maintenance. Older work
  remains useful infrastructure, but its hardcoded signup policy is superseded.
  The ledger distinguishes reusable completed mechanisms from the unimplemented
  production Goja composition.

### What warrants a second pair of eyes

- Confirm the launch policy before public rollout: protected staging can use
  open signup, while public production should wait for working email delivery.
- Review whether startup bootstrap diagnostics are operationally sufficient
  before treating that task as permanently closed.

### What should be done in the future

- Implement Phase 2 in task order, beginning by removing the legacy production
  registration option and requiring a checked signup program.
- Build the real two-process harness before adding containers or cluster state.

### Code review instructions

- Read the updated design sections 1 through 5, then compare `tasks.md` Phase 2
  and Phase 3 with `internal/cmds/serve_production.go` and
  `pkg/embeddedidp/options.go`.
- Validate with `docmgr validate frontmatter` and `docmgr doctor`.

### Technical details

```text
reviewed signup.js -> check -> activate generation -> readiness
browser POST -> consume continuation -> resume pinned generation
commit request -> native atomic account service -> resume OIDC

Tiny-IDP startup -> schema + signing key + exact browser client
Kubernetes       -> no competing bootstrap Job
```

## Step 11: Decompose the remaining delivery into precise checkpoints

The re-baselined task list identified the correct phases but still grouped
several days of implementation and validation into single checkboxes. That
would make progress ambiguous: for example, program loading, capability
binding, activation, readiness, and failure tests could not be tracked
independently.

I expanded every remaining phase into dependency-ordered checkpoints. Existing
stable IDs were retained for completed work and for the broad milestones they
still represent; new IDs identify individual construction, security,
validation, publishing, rollout, and recovery outcomes.

### Prompt Context

**User prompt (verbatim):** "Create detailed tasks in the ticket if you don't already, so we can precisely track your progress"

**Assistant interpretation:** Replace coarse remaining ticket entries with
small, objective tasks that can be checked immediately after their evidence is
committed.

**Inferred user intent:** Make future autonomous implementation observable and
prevent large phases from appearing stalled or complete without proof.

### What I did

- Split production composition into removal, file loading, checking, service
  construction, capability binding, activation, bootstrap, readiness, and
  focused-test tasks.
- Split the two-process proof into lifecycle, happy-path, negative, restart,
  leakage-scan, and repository-gate tasks.
- Split image delivery into construction, filesystem security, paired smoke,
  publishing, GitOps metadata, and source-PR tasks.
- Split GitOps into each Kubernetes, Vault, network, backup, validation, secret
  seeding, and PR result.
- Split live acceptance and recovery into individually observable checks.

### Why

- Each checkbox should correspond to one reviewable outcome and its evidence.
- Dependency order makes the next safe action obvious without inventing work
  outside the ticket.
- Separate negative tests and operational checks prevent a happy-path demo from
  being mistaken for production completion.

### What worked

- The existing phase gates mapped cleanly onto detailed tasks without changing
  the approved project scope.
- Existing task IDs could be preserved while adding stable IDs for the newly
  exposed substeps.

### What didn't work

- N/A.

### What I learned

- The most important tracking boundary is between constructing the Goja runtime
  and proving its behavior through the actual two-process OIDC flow.
- Cluster work also needs separate desired-state and live-acceptance tasks;
  manifest rendering is not production evidence.

### What was tricky to build

- Granularity had to improve without expanding scope. I therefore decomposed
  only work already implied by the approved design and acceptance matrix; I did
  not add device authorization, multi-app support, password recovery, or new
  product capabilities.

### What warrants a second pair of eyes

- Review Phase 2 ordering around bootstrap versus generation activation when
  implementation starts; both must complete before readiness, but constructors
  may impose a more precise internal order.
- Review the eventual production policy choice before the public acceptance
  tasks are run.

### What should be done in the future

- Check tasks immediately after the corresponding code/test commit and cite the
  commit in the diary and changelog.
- Begin with `t:p2i1`; do not start images or GitOps before Phase 3 passes.

### Code review instructions

- Read `tasks.md` from Phase 2 through Phase 7 and verify each entry has an
  observable completion criterion.
- Run `docmgr task list --ticket TINYIDP-K3S-MSGDESK-PROD-001` and
  `docmgr doctor --ticket TINYIDP-K3S-MSGDESK-PROD-001 --stale-after 30`.

### Technical details

```text
Phase 2  production composition and focused contracts
Phase 3  real two-process behavioral proof
Phase 4  immutable same-commit images
Phase 5  GitOps desired state and PR
Phase 6  public reconciliation and acceptance
Phase 7  restored-state proof and closure
```

## Step 12: Compose checked scripted signup into the production host

The production server now uses one signup path. It no longer accepts the
legacy `--registration-enabled` switch or constructs a production
`RegistrationConfig`; instead it requires a non-secret JavaScript program,
checks it, warms a generation, and passes that generation to the embedded
provider before the listener can start.

This first production policy is deliberately narrow: the shipped open-signup
program is accepted. A program requiring an unbound native capability, email
challenge, or another native effect fails during startup. That is the correct
initial fail-closed boundary while Phase 2 has not yet supplied deliberate
provider bindings for those workflows.

### Prompt Context

**User prompt (verbatim):** "phase 2"

**Assistant interpretation:** Implement the detailed Phase 2 production
composition tasks, with reviewable commits and ticket evidence.

**Inferred user intent:** Make the completed Goja workflow system the actual
Message Desk production signup mechanism before pursuing images or GitOps.

**Commit (code):** `5546ac5` — "Feat: activate scripted signup in production host"

### What I did

- Removed `RegistrationEnabled` and `--registration-enabled` from the
  production command.
- Added required `--signup-program-file` loading with a 256 KiB regular-file
  bound and no source-content logging.
- Checked the source before bootstrap/listening and warmed a one-worker,
  one-retained-generation manager audited through the durable production sink.
- Passed the manager through `embeddedidp.ScriptedSignupConfig` and closed it
  on all startup failures and normal shutdown.
- Changed the internal provider composition so scripted signup derives the
  canonical account service from the password authenticator without requiring
  legacy registration to be enabled.
- Added the program-contract tests and documentation in `241f928` and
  `936705e`.

### Why

- A production caller must have one explicit workflow owner, not a hidden
  legacy flag which happens to create a default executor.
- Checking and warming before listening converts source/configuration errors
  into a safe rollout failure rather than a browser-time failure.
- The account service remains Go-owned; JavaScript still receives only the
  atomic effect protocol and secret handles.

### What worked

- `go test ./internal/cmds ./internal/fositeadapter ./pkg/embeddedidp -count=1`
  passed during implementation.
- `go test ./pkg/idp ./internal/cmds ./examples/tinyidp-message-app
  ./internal/fositeadapter -count=1` passed after the production-manager
  composition was in place.
- The pre-commit hook for `241f928` and `936705e` passed repository lint,
  Glazed lint, the idpui analyzer, and `go test ./...`.
- `8a2f01f` adds the missing-program-source regression and its pre-commit hook
  passed the same repository-wide gates.
- The new integration regression proves a `tinyidp_signup=1` request reaches a
  scripted form with no legacy `RegistrationConfig` enabled.

### What didn't work

- The first native-service validator build failed with exactly
  `too many return values; have (nil, error) want (error)` in
  `validateProductionSignupProgram`. The helper returns one `error`; replacing
  `return nil, fmt.Errorf(...)` with `return fmt.Errorf(...)` fixed it. The
  immediately repeated focused test passed.

### What I learned

- The SQLite store already implements the durable workflow-continuation store;
  `fositeadapter.NewProvider` uses it when an activated scripted generation is
  supplied in production.
- `Provider.Readiness` already reports the generation manager as
  `scripted_signup`, alongside store/schema/signing-key/audit checks. The host
  composition therefore needed to supply the manager, not invent a second
  readiness endpoint.

### What was tricky to build

- The old registration option did more than advertise a UI choice: it supplied
  the account service used at the atomic commit boundary and gated signup
  intent. The replacement preserves that Go-owned account service by deriving
  it from the normal authenticator only when scripted signup is present; it
  does not make JavaScript an account-store owner.

### What warrants a second pair of eyes

- Review the intentional initial capability policy. It accepts the open-signup
  contract and rejects unbound capabilities/challenges/effects at startup.
  Email-verified public signup requires a subsequent deliberate mail-provider
  binding, not merely mounting `email_verified_signup.js`.
- Review the one-worker generation-pool choice against expected signup volume
  before public rollout.

### What should be done in the future

- Finish the remaining Phase 2 capability/provider-binding decision and
  add startup tests for every failure mode.
- Build the real two-process harness only after the production contract is
  settled; containers and GitOps remain out of scope until then.

### Code review instructions

- Start at `internal/cmds/serve_production.go`, then follow
  `pkg/embeddedidp/provider.go` into `internal/fositeadapter/provider.go` and
  `internal/fositeadapter/scripted_signup.go`.
- Run `go test ./internal/cmds ./internal/fositeadapter ./pkg/embeddedidp -count=1`;
  the repository hook additionally ran `go test ./...`.

### Technical details

```text
signup.js
  -> bounded read
  -> compile/check static program contract
  -> reject unbound capability/challenge/effect
  -> warm active GenerationManager
  -> Bootstrap exact client/key state
  -> embeddedidp.ScriptedSignupConfig
  -> /readyz includes scripted_signup
```

## Step 13 — Phase 3 tracking contract and two-process harness start

### Prompt Context

**User prompt (verbatim):** "Create detailed tasks in the ticket if you don't already, so we can precisely track your progress"

**Assistant interpretation:** Verify that the ticket contains an independently
checkable Phase 3 ledger before starting the real two-process assurance work,
and make the completion rule explicit rather than treating checkboxes as a
plan-only list.

**Inferred user intent:** Be able to see exactly what is done, what remains,
and what proof is required for every product-level behavior.

### What I did

- Verified the ticket already has dependency-ordered Phase 3 tasks covering
  process lifecycle, browser signup, OAuth/OIDC state propagation, application
  sessions/messages, negative cases, restarts, leak scanning, and final gates.
- Added the Phase 3 tracking contract to `tasks.md`: each check requires a
  real assertion, a passing focused run, and diary evidence naming the command
  and commit.
- Left every Phase 3 box open. No process-level behavior has yet been proven
  by a two-process harness, so marking a planning task complete early would be
  misleading.

### Why

The source tests prove individual construction paths, but Phase 3 must prove
the deployable product boundary: two separately started binaries, independent
durable state, public HTTPS origins behind trusted proxy listeners, and a
browser-equivalent redirect/cookie flow. The ledger distinguishes those claims
from implementation intent.

### What worked

- The current tasks enumerate each acceptance behavior independently and have
  stable `docmgr` task identifiers, so progress can be checked through the
  ticket rather than inferred from prose or commits.

### What didn't work

- Nothing was executed in this planning checkpoint; the first actual harness
  assertion is intentionally still pending.

### What I learned

- The appropriate first Phase 3 implementation boundary is a ticket-scoped Go
  harness which builds and launches `tinyidp serve-production` and the external
  Message Desk command as separate child processes. It must emulate the
  trusted Traefik request identity while retaining canonical public HTTPS
  origins.

### What was tricky to build

- A normal Go cookie jar will not retain `Secure` cookies received over the
  local plaintext test transport. The harness must model the reverse-proxy
  browser boundary deliberately instead of weakening either process into a
  development-origin listener.

### What warrants a second pair of eyes

- Review that the tracking rule is strict enough for product evidence but does
  not require committing temporary test state or secrets.

### What should be done in the future

- Implement and prove `t:783q` through `t:p3h4` first, then check them with
  `docmgr task check` as each focused assertion passes.

### Code review instructions

- Start with Phase 3 in `tasks.md`; its order is the review and execution
  order. The next source addition will be under this ticket's `scripts/`
  directory and will be related to the ticket when it exists.

### Technical details

```text
unchecked task
  -> focused harness assertion implemented
  -> command passes against two subprocesses
  -> diary records command + result + commit
  -> docmgr task check
```

## Step 14 — Phase 3 two-process lifecycle harness

### Prompt Context

**User prompt (verbatim):** "track it and phase 3"

**Assistant interpretation:** Begin Phase 3 product evidence, retaining the
ticket as a precise ledger rather than relying on in-process unit coverage.

**Inferred user intent:** Prove the real standalone IdP and external Message
Desk executable topology before moving to containers and GitOps.

**Commit:** `3cc6f38` — "Test: add two-process production lifecycle harness"

### What I did

- Added `scripts/01-two-process-harness/two_process_test.go`, a root-module Go
  integration test which builds `./cmd/tinyidp` and
  `./examples/tinyidp-message-app` into a test-only directory.
- Initialized a separate owner-only Message Desk state root, then started
  `tinyidp serve-production` with its own SQLite database, file audit sink,
  owner-only 32-byte test secret, and the checked embedded open-signup source.
- Started the external Message Desk process with its own application database
  and audit file, public issuer `https://idp.example.test/idp`, and an explicit
  backchannel destination.
- Added a deliberately narrow in-test Traefik-equivalent reverse proxy. It
  preserves the public IdP Host and supplies `X-Forwarded-Proto: https` and
  `X-Forwarded-Host` only while forwarding to the private IdP listener. Browser
  readiness requests use the same trusted-forwarding contract directly.
- Waited for `/idp/readyz` and `/readyz`, retained both child-process logs,
  checked Tiny-IDP's durable bootstrap audit, and gracefully terminates all
  child processes during test cleanup.

### Why

The external OIDC client correctly rewrites the network destination but keeps
the issuer identity. A direct service request cannot impersonate Traefik at a
listener that trusts only Traefik forwarding metadata. Modeling that proxy
boundary makes the harness honest: neither production binary is weakened to a
plain-development origin and neither shares process memory or SQLite state.

### What worked

- `go test ./ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness -count=1 -v`
  passed in 10.23 seconds.
- The `3cc6f38` pre-commit gate passed `golangci-lint`, the Glazed and IdP UI
  analyzers, and `go test ./...`; the new harness itself passed in 11.214
  seconds inside that complete suite.

### What didn't work

- The first external Message Desk startup attempted direct backchannel
  discovery against Tiny-IDP's trusted-proxy listener. Tiny-IDP rejected it
  with `400 Bad Request: trusted proxy must forward HTTPS transport`, which is
  the desired protection. The test now models the terminating proxy rather
  than bypassing or relaxing that guard.
- The initial assertion expected an activation audit event. Initial generation
  warming does not emit that event; the harness instead verifies the durable
  `identity.bootstrap.client_created` event and readiness, which are the
  observable production-startup contract.

### What I learned

- `external-backchannel-url` is a destination rewrite, not authority rewrite:
  discovery, JWKS, and token validation still see the canonical public issuer.
- A final k3s design must route the IdP backchannel through the same
  Traefik-style trusted boundary or introduce a separately designed and
  authenticated internal listener. Merely allowing the Message Desk Pod to
  send forwarded headers to the public listener would incorrectly grant it
  proxy authority.

### What was tricky to build

- Secure cookies cannot be accepted by Go's ordinary cookie jar over the
  plaintext loopback transport. The subsequent browser-flow slice therefore
  needs an explicit public-origin cookie model; this lifecycle slice avoids
  concealing the issue by turning off Secure cookies.

### What warrants a second pair of eyes

- Confirm the selected production routing choice for IdP backchannel traffic:
  service-to-Traefik-to-IdP versus a separate authenticated internal endpoint.
  The current ticket wording says direct Service backchannel, but the tested
  trusted-listener invariant demonstrates that this is not yet a valid direct
  route.

### What should be done in the future

- Add the browser-equivalent `/auth/register` PKCE/nonce/redirect assertion,
  then form submission, callback, session, message, CSRF, and restart cases.

### Code review instructions

- Review the harness source from `TestTwoProcessLifecycle` downward. The
  lifecycle contract is in `newHarness`, `startTinyIDP`, `startIDPProxy`,
  `startMessageDesk`, `waitReady`, and `startedProcess.stop`.
- Re-run the focused command above. It leaves no durable test state outside
  Go's test temporary directory.

### Technical details

```text
Browser test request                OIDC server-side request
  HTTPS public origin                 public issuer identity
      |                                      |
      v                                      v
test forwarded request             external-backchannel destination
      |                                      |
      +----------> trusted proxy <----------+
                           |
                           v
                    Tiny-IDP HTTP listener
                    (only forwarded HTTPS)
```
