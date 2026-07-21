# Changelog

## 2026-07-21

- Initial workspace created


## 2026-07-21

Created evidence-backed two-application shared IdP design, GitOps theme boundary, implementation phases, and bounded Kubernetes operator research branch.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk — Existing deployment topology analyzed
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — One-client production host analyzed


## 2026-07-21

Validated the complete ticket and uploaded the ToC-enabled implementation bundle to reMarkable at /ai/2026/07/21/TINYIDP-MULTIAPP-THEMES-001.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/design-doc/01-shared-tiny-idp-theme-assets-and-a-second-application-on-k3s-analysis-design-and-implementation-guide.md — Delivered intern implementation guide

## 2026-07-21

Implemented and validated Phases 1-2 in commit 7d8a19d: strict production client catalogs, startup-loaded per-client mounted themes, signup workflow client attribution, same-origin asset serving, and updated production examples/harnesses.

### Related Files

- internal/cmds/serve_production.go — Production host catalog wiring
- internal/productionconfig/clients.go — Strict versioned multi-client production catalog
- internal/productionui/catalog.go — Mounted theme catalog and allowlisted CSS assets

## 2026-07-21 - Prepared the combined two-app GitOps rollout

Recorded source merges, image publishing repair, verified k3s/Argo topology, and GitOps commit fee8104 containing strict catalogs, mounted themes, the goja-auth repoint, bounded network policy, and an idempotent database migration.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/goja-auth-host-demo/bootstrap-configmap.yaml — Non-destructive OIDC schema migration
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/clients.json — Authoritative two-client catalog
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/hetzner-k3s-phase5/gitops/kustomize/tiny-message-desk/themes/themes.json — Authoritative per-client theme mapping
- ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/reference/01-investigation-diary.md — Step 6 contains the exact cross-repository rollout evidence

## 2026-07-21 - Deployed and accepted the shared two-app TinyIDP

Merged GitOps PR 191 at 68209d0, reached Synced/Healthy in Argo, and passed public MessageDesk signup plus shared-account goja-auth login with distinct client-selected themes, protected route, CSRF logout, stable PVCs, Ready certificates, and accepted audit events.

### Related Files

- ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/reference/02-production-client-and-theme-runbook.md — Client and theme operations and rollback procedures
- ttmp/2026/07/21/TINYIDP-MULTIAPP-THEMES-001--shared-tiny-idp-themes-and-second-application-on-k3s/reference/03-production-acceptance-evidence.md — Non-secret production delivery and runtime evidence
