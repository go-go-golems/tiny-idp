# Tasks

## TODO

- [x] Phase 0: Refresh remotes and merge current origin/main into the implementation branch <!-- t:pxdl -->
- [x] Phase 0: Record production hostnames, one-replica SQLite topology, signup policy, and deferred device/multi-app scope <!-- t:d7s0 -->
- [x] Phase 0: Inspect current cluster Traefik forwarding, NetworkPolicy, prod-apps, Vault/VSO, registry, and backup contracts <!-- t:k9g9 -->
- [x] Phase 0: Write the detailed implementation design, acceptance matrix, rollback plan, and implementation diary <!-- t:avj8 -->
- [x] Phase 1: Implement and verify the lambda-first Goja signup foundation in TINYIDP-GOJA-001 <!-- t:ev4b -->
- [x] Phase 1: Provide checked programs, explicit continuations, native secret handles, and atomic signup commits <!-- t:bumr -->
- [x] Phase 1: Provide open-signup and email-verified-signup reference programs <!-- t:g4u2 -->
- [x] Phase 1: Provide generation activation, readiness, audit, policy, invite, email, and identity seams <!-- t:uo24 -->
- [x] Phase 1: Add Message Desk /auth/register initiation without exposing the provider account store or password <!-- t:hpkb -->
- [x] Phase 1: Update Message Desk UI for external provider-owned signup <!-- t:0uf8 -->
## Phase 2: Compose scripted signup into the production processes

- [x] Inventory the current `serve-production` construction order and record the exact legacy registration references to remove <!-- t:p2i1 -->
- [x] Remove the `RegistrationEnabled` production setting and `--registration-enabled` Glazed field <!-- t:rlgb -->
- [x] Remove production construction of `embeddedidp.RegistrationConfig`; do not add an adapter or fallback <!-- t:p2r2 -->
- [x] Update production command help, examples, and tests so the removed registration option is no longer advertised or accepted <!-- t:p2r3 -->
- [x] Add a required `--signup-program-file` production setting with a documented non-secret file contract <!-- t:sgp1 -->
- [x] Read the signup program with bounded input, contextual errors, and no source dumping into logs <!-- t:p2s2 -->
- [x] Check/compile the program before bootstrap, handler construction, or listener startup <!-- t:p2s3 -->
- [x] Construct the durable continuation, identity, workflow, evidence, and audit services required by scripted signup <!-- t:p2s4 -->
- [x] Derive the program's declared native capability set from the checked artifact <!-- t:sgp2 -->
- [x] For the initial open-signup policy, bind no JavaScript capabilities and reject any declared unimplemented native capability/service at startup <!-- t:p2c2 -->
- [x] Document the initial protected-staging policy as the shipped open-signup program and defer its ConfigMap mount to Phase 5 <!-- t:p2c3 -->
- [x] Keep future email/invite credentials out of JavaScript; defer their secret-backed native services with the corresponding workflow binding <!-- t:p2c4 -->
- [x] Activate exactly one checked generation before constructing the public signup handler <!-- t:p2a1 -->
- [x] Pass the generation manager and native services through `embeddedidp.ScriptedSignupConfig` <!-- t:p2a2 -->
- [x] Preserve generation pinning across every browser continuation and reject missing/retired generations safely <!-- t:p2a3 -->
- [x] Keep startup schema/signing-key/client bootstrap as the sole exact-state owner and run it before readiness <!-- t:p2b1 -->
- [x] Confirm repeated exact-state bootstrap is a no-op and widening/mismatch errors remain non-secret <!-- t:p2b2 -->
- [x] Make active generation and required native-capability availability part of Tiny-IDP readiness <!-- t:sgp3 -->
- [x] Make continuation, identity, signing/client, audit, and other required persistent stores part of Tiny-IDP readiness <!-- t:p2h2 -->
- [x] Verify Message Desk readiness still covers its database and durable external audit sink <!-- t:p2h3 -->
- [x] Phase 2: Define explicit direct-TLS and trusted-Traefik listener modes without compatibility aliases <!-- t:jabz -->
- [x] Phase 2: Implement trusted proxy CIDR/hop validation and canonical HTTPS origin enforcement in Tiny-IDP <!-- t:cbhq -->
- [x] Phase 2: Implement trusted proxy CIDR/hop validation and canonical HTTPS origin enforcement in Message Desk <!-- t:fwvu -->
- [x] Phase 2: Preserve Secure cookies and reject untrusted forwarded identity in both processes <!-- t:8t2f -->
- [x] Phase 2: Add durable external-mode Message Desk audit output and readiness checks <!-- t:nj81 -->
- [x] Phase 2: Use process startup as the sole idempotent exact-state owner for signing-key and Message Desk browser-client bootstrap <!-- t:0b0p -->
- [x] Test missing, unreadable, oversized, syntactically invalid, and contract-invalid signup program startup failures <!-- t:p2t1 -->
- [x] Test the selected zero-capability program and fail-closed rejection of unimplemented native capability/service declarations <!-- t:p2t2 -->
- [x] Test activation ordering, active-generation readiness, and pinned-continuation generation behavior <!-- t:p2t3 -->
- [x] Test startup-bootstrap no-op, exact-state mismatch, and widening rejection <!-- t:p2t4 -->
- [x] Re-run focused listener, forwarding, Secure-cookie, audit, and readiness tests with scripted signup enabled <!-- t:4e1i -->
- [x] Run Phase 2 package tests and commit production composition in reviewable checkpoints <!-- t:p2g1 -->

## Phase 3: Prove the real two-process product locally

### Tracking contract

Work through this phase in the listed order. A checkbox is evidence, not an
intention: check it only after the harness assertion exists, its focused run
has passed, and the diary names the command and commit. Keep all transient
state, process logs, browser fixtures, and generated signup programs beneath
the ticket's `scripts/` harness temporary root; do not add a second module or
turn test credentials into repository files. The `p3d1` diary task is the
phase-close record and therefore remains open until every preceding Phase 3
item has concrete evidence.

- [x] Design the harness lifecycle, temporary directory layout, port allocation, and cleanup contract <!-- t:783q -->
- [x] Start the real Tiny-IDP production command with separate durable state and a checked signup program <!-- t:p3h2 -->
- [x] Start the real external Message Desk command with its own database/audit state and Tiny-IDP backchannel URL <!-- t:p3h3 -->
- [x] Wait on readiness endpoints and capture both process logs and durable audit output <!-- t:p3h4 -->
- [x] Drive `/auth/register` and assert the authorization request retains client, redirect URI, PKCE, nonce, scope, and signup intent <!-- t:p3f1 -->
- [x] Submit the scripted signup form and assert exactly one identity and provider session are created <!-- t:p3f2 -->
- [x] Complete the authorization callback and assert a Message Desk application session is created <!-- t:p3f3 -->
- [x] Create and read a message as the signed-in subject <!-- t:p3f4 -->
- [x] Prove missing/invalid message CSRF is rejected without durable mutation <!-- t:p3f5 -->
- [ ] Prove local logout revokes only the Message Desk session <!-- t:p3f6 -->
- [ ] Prove provider logout and subsequent re-login follow the registered redirect contract <!-- t:hv5c -->
- [ ] Test duplicate identity, weak password, malformed input, and generic non-enumerating public errors <!-- t:p3n1 -->
- [ ] Test expired, replayed, and concurrently submitted signup continuations <!-- t:p3n2 -->
- [ ] Restart Tiny-IDP during a pending continuation and prove pinned resume or an explicit safe failure <!-- t:p3r1 -->
- [ ] Restart Message Desk during a pending OIDC transaction and prove the declared durable-state behavior <!-- t:p3r2 -->
- [ ] Restart both processes after signup and prove users, key, client, sessions-by-policy, and messages persist <!-- t:mbvy -->
- [ ] Scan logs and audits for passwords, cookies, authorization codes, raw tokens, and secret material <!-- t:p3s1 -->
- [ ] Run generation and focused package tests once after the complete harness slice <!-- t:p3g1 -->
- [ ] Run `go test ./...`, race tests for changed packages, lint, build, and repository security checks <!-- t:5fm4 -->
- [ ] Record exact commands, results, failures, and review instructions in the diary <!-- t:p3d1 -->

## Phase 4: Build and publish immutable application images

- [ ] Define separate production image targets and runtime entrypoints for Tiny-IDP and Message Desk <!-- t:ftd8 -->
- [ ] Build Message Desk frontend assets deterministically before Go compilation <!-- t:p4i2 -->
- [ ] Use multi-stage builds, non-root users, fixed work/state paths, and minimal runtime contents <!-- t:p4i3 -->
- [ ] Add OCI source, revision, version, and creation metadata to both images <!-- t:p4i4 -->
- [ ] Mount the reviewed signup program as read-only non-secret input <!-- t:img1 -->
- [ ] Mount native-provider and token credentials through owner-only secret files <!-- t:p4s1 -->
- [ ] Verify read-only root filesystem compatibility and explicitly writable state/audit paths <!-- t:p4s2 -->
- [ ] Add image-level health/readiness and non-root ownership smoke tests <!-- t:6m3o -->
- [ ] Run both images together and repeat the essential signup/login/message smoke path <!-- t:p4t1 -->
- [ ] Publish Tiny-IDP and Message Desk with immutable `sha-<commit>` tags from the same source commit <!-- t:tbv9 -->
- [ ] Add `deploy/gitops-targets.json` for both image consumers <!-- t:cs2l -->
- [ ] Add and validate the shared GitOps update workflow caller <!-- t:p4c1 -->
- [ ] Open the source PR and record image names, tags/digests, CI jobs, and review URL <!-- t:p4p1 -->
- [ ] Resolve CI/review findings, obtain green checks, and merge the source PR <!-- t:7viy -->

## Phase 5: Submit complete k3s GitOps desired state

- [ ] Refresh the k3s repository and create an isolated branch/worktree from current `origin/main` <!-- t:5hhi -->
- [ ] Add `tiny-message-desk` namespace authorization to the `prod-apps` Argo project <!-- t:p5a1 -->
- [ ] Add namespace and distinct least-privilege ServiceAccounts <!-- t:p5k1 -->
- [ ] Add separate Tiny-IDP and Message Desk PVCs with documented storage sizes <!-- t:p5k2 -->
- [ ] Add Tiny-IDP Deployment with one replica, `Recreate`, probes, resources, security context, and immutable image <!-- t:p5k3 -->
- [ ] Add Message Desk Deployment with one replica, `Recreate`, probes, resources, security context, and immutable image <!-- t:p5k4 -->
- [ ] Add ClusterIP Services and Traefik Ingresses for the two canonical HTTPS origins <!-- t:p5k5 -->
- [ ] Add the reviewed signup program as a checksum-rollout ConfigMap mount <!-- t:mr4d -->
- [ ] Configure Message Desk's public issuer and internal Tiny-IDP backchannel without changing issuer identity <!-- t:p5k6 -->
- [ ] Add least-privilege Vault policy, Kubernetes auth role, VaultAuth, and VaultStaticSecret resources <!-- t:mmv8 -->
- [ ] Copy secret material into main-container-owned `0600` files without logging values <!-- t:p5v1 -->
- [ ] Add mail-provider credentials only if the selected public signup program declares that capability <!-- t:p5v2 -->
- [ ] Add GHCR image-pull wiring only if package visibility requires it <!-- t:p5v3 -->
- [ ] Add NetworkPolicy for Traefik ingress, probes, DNS, Message Desk-to-IdP backchannel, and required provider egress <!-- t:9q3v -->
- [ ] Confirm manifests contain no bootstrap Job competing with Tiny-IDP startup <!-- t:p5b1 -->
- [ ] Add online SQLite backup CronJobs, external destination, retention, and restore runbook <!-- t:p5d1 -->
- [ ] Add Kustomization and Argo Application resources with explicit sync ordering <!-- t:p5a2 -->
- [ ] Render with `kubectl kustomize` and validate schemas, immutable images, probes, selectors, mounts, and namespace scope <!-- t:wng5 -->
- [ ] Validate NetworkPolicy reachability and denial boundaries against the documented topology <!-- t:p5n1 -->
- [ ] Seed required Vault material through an approved non-printing operator path <!-- t:mnn9 -->
- [ ] Open the GitOps PR and record rendered-diff, policy, secret-shape, and rollback review instructions <!-- t:p5p1 -->
- [ ] Resolve review/CI findings and merge the GitOps PR <!-- t:jet5 -->

## Phase 6: Reconcile and accept production

- [ ] Bootstrap or refresh the Argo Application and wait for `Synced` and `Healthy` <!-- t:u2lt -->
- [ ] Verify Pods, Services, Endpoints, PVCs, certificates, Ingresses, and probe status <!-- t:p6k1 -->
- [ ] Verify public discovery metadata and exact HTTPS issuer equality <!-- t:6n23 -->
- [ ] Verify TLS, security headers, Secure cookies, canonical origins, and redirect URIs <!-- t:p6w1 -->
- [ ] Verify direct exposure and untrusted forwarding cannot rewrite address, issuer, host, or scheme <!-- t:p6w2 -->
- [ ] Run browser signup through the selected production program <!-- t:p6a1 -->
- [ ] Complete login, application session, message creation, negative CSRF, local logout, provider logout, and re-login <!-- t:dnw6 -->
- [ ] Restart Message Desk and verify messages plus the declared app-session policy persist <!-- t:p6r1 -->
- [ ] Restart Tiny-IDP and verify identities, active key, exact client, continuations-by-policy, and login persist <!-- t:txry -->
- [ ] Inspect application, Traefik, provider audit, and security logs for credential leakage <!-- t:lwxe -->
- [ ] Record production acceptance evidence and any intentionally unsupported behavior in the diary <!-- t:p6d1 -->

## Phase 7: Prove recovery and close the ticket

- [ ] Create online backups of both SQLite stores and required durable audit output <!-- t:bm3f -->
- [ ] Record matching secret/key versions, image digests, schema versions, and signup-program revision without exposing secrets <!-- t:p7b1 -->
- [ ] Provision isolated scratch PVCs and non-public restore workloads <!-- t:p7r1 -->
- [ ] Restore Tiny-IDP and Message Desk state into the scratch environment <!-- t:p7r2 -->
- [ ] Run database integrity, Tiny-IDP doctor, key/client, and Message Desk checks on restored state <!-- t:p7r3 -->
- [ ] Prove restored login and reading an existing message <!-- t:mhfj -->
- [ ] Document rollback triggers, commands, recovery ownership, retention, and alerting <!-- t:p7o1 -->
- [ ] Document accepted single-node limitations and deferred device/multi-app scope <!-- t:xx1x -->
- [ ] Complete diary, changelog, related-file notes, and final acceptance evidence <!-- t:p7d1 -->
- [ ] Run frontmatter validation and `docmgr doctor`; ensure no task remains unchecked <!-- t:4gwe -->
- [ ] Close the ticket only after source, GitOps, public acceptance, and restore evidence are complete <!-- t:p7c1 -->

## Re-baseline notes (2026-07-20)

- The completed `TINYIDP-GOJA-001` lambda-first workflow supersedes the legacy
  hardcoded provider-registration workflow for production.
- No backward-compatibility adapter will keep both signup paths in the
  production command.
- The first production policy is selected as a reviewed JavaScript program;
  staging may use open signup, while public production should use
  email-verified signup when mail delivery is operational.
- Tiny-IDP process startup is the sole exact-state owner for schema, one active
  signing key, and the Message Desk browser client. Kubernetes must not add a
  competing bootstrap Job.
- Device authorization, multiple applications, coding-agent bearer access,
  refresh tokens, and xgoja application routes remain out of scope.
