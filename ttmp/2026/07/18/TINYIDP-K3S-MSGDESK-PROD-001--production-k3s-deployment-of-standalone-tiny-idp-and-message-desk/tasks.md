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
- [ ] Phase 2: Remove the legacy production RegistrationConfig/--registration-enabled path rather than retain an adapter <!-- t:rlgb -->
- [ ] Phase 2: Add a required signup-program file contract and activate a checked generation before listening <!-- t:sgp1 -->
- [ ] Phase 2: Bind only the native capabilities declared by the selected signup program and fail closed when unavailable <!-- t:sgp2 -->
- [ ] Phase 2: Make active signup generation, persistent stores, signing state, and audit availability part of readiness <!-- t:sgp3 -->
- [x] Phase 2: Define explicit direct-TLS and trusted-Traefik listener modes without compatibility aliases <!-- t:jabz -->
- [x] Phase 2: Implement trusted proxy CIDR/hop validation and canonical HTTPS origin enforcement in Tiny-IDP <!-- t:cbhq -->
- [x] Phase 2: Implement trusted proxy CIDR/hop validation and canonical HTTPS origin enforcement in Message Desk <!-- t:fwvu -->
- [x] Phase 2: Preserve Secure cookies and reject untrusted forwarded identity in both processes <!-- t:8t2f -->
- [x] Phase 2: Add durable external-mode Message Desk audit output and readiness checks <!-- t:nj81 -->
- [x] Phase 2: Use process startup as the sole idempotent exact-state owner for signing-key and Message Desk browser-client bootstrap <!-- t:0b0p -->
- [ ] Phase 2: Add focused scripted-production, listener, forwarding, readiness, and bootstrap reconciliation tests <!-- t:4e1i -->
- [ ] Phase 3: Add a real two-process harness for scripted Tiny-IDP and external Message Desk <!-- t:783q -->
- [ ] Phase 3: Prove signup, OIDC completion, app session, message creation, CSRF rejection, and logout <!-- t:hv5c -->
- [ ] Phase 3: Prove continuation, identity, signing/client, app-session, and message behavior across process restarts <!-- t:mbvy -->
- [ ] Phase 3: Scan captured logs/audit for credentials and run focused, full, race, lint, build, and security checks <!-- t:5fm4 -->
- [ ] Phase 4: Add production multi-stage images for Tiny-IDP and Message Desk <!-- t:ftd8 -->
- [ ] Phase 4: Mount the reviewed signup program as non-secret input and keep native service credentials in owner-only secret files <!-- t:img1 -->
- [ ] Phase 4: Add paired image smoke tests, non-root users, owner-only paths, health checks, and OCI metadata <!-- t:6m3o -->
- [ ] Phase 4: Add CI publishing of two immutable same-commit GHCR image tags <!-- t:tbv9 -->
- [ ] Phase 4: Add deploy/gitops-targets.json and the shared GitOps update workflow <!-- t:cs2l -->
- [ ] Phase 4: Publish the source branch, obtain CI-green review, and merge the source PR <!-- t:7viy -->
- [ ] Phase 5: Create an isolated k3s GitOps branch from current origin/main <!-- t:5hhi -->
- [ ] Phase 5: Add namespace, service accounts, PVCs, Deployments, Services, Ingresses, probes, resources, Recreate rollout, and signup-program ConfigMap <!-- t:mr4d -->
- [ ] Phase 5: Add Vault policy/role, VaultAuth/VaultStaticSecret, owner-only secret copy, and optional image-pull wiring <!-- t:mmv8 -->
- [ ] Phase 5: Add NetworkPolicy, backchannel Service wiring, backup CronJobs/runbook, and Argo Application; do not add a second bootstrap owner <!-- t:9q3v -->
- [ ] Phase 5: Render and validate Kustomize, manifests, policy boundaries, and prod-apps namespace allowance <!-- t:wng5 -->
- [ ] Phase 5: Seed production Vault and selected signup-provider material without exposing secret values <!-- t:mnn9 -->
- [ ] Phase 5: Publish, review, and merge the GitOps PR <!-- t:jet5 -->
- [ ] Phase 6: Bootstrap the Argo Application and wait for Synced/Healthy production state <!-- t:u2lt -->
- [ ] Phase 6: Verify public discovery, issuer, TLS, health, readiness, cookies, headers, and direct-exposure controls <!-- t:6n23 -->
- [ ] Phase 6: Run browser signup/login/message/CSRF/logout/re-login acceptance against production hostnames <!-- t:dnw6 -->
- [ ] Phase 6: Restart each workload and confirm users, keys, clients, sessions by policy, and messages persist <!-- t:txry -->
- [ ] Phase 6: Inspect application, ingress, audit, and security logs for credential leakage <!-- t:lwxe -->
- [ ] Phase 7: Create online backups of both SQLite stores and record the matching secret/key recovery set <!-- t:bm3f -->
- [ ] Phase 7: Restore both stores into scratch PVCs and prove login plus existing-message read <!-- t:mhfj -->
- [ ] Phase 7: Document rollback, operational ownership, accepted single-node limitations, and deferred device/multi-app work <!-- t:xx1x -->
- [ ] Phase 7: Complete diary, changelog, file relationships, docmgr validation, and close the ticket <!-- t:4gwe -->

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
