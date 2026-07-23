# Changelog

## 2026-07-23

- Initial workspace created


## 2026-07-23

Created an exploratory plugin architecture draft after auditing TinyIDP configuration, Goja, lifecycle, and observability seams and revalidating the Jitsi/Prosody token contract. Recorded compiled-in registration as the leading option and browser identity as the first full-design decision.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp/main.go — Shows the production Glazed parser-composition gap
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/pkg/idpscript/capabilities.go — Supplies the recommended bounded scripting model


## 2026-07-23

Added the full intern-facing plugin system and Jitsi integration analysis, design, and implementation guide. Selected a compiled-in registry, plugin-owned Glazed sections, a host-managed in-process OIDC/PKCE broker, bounded Goja policy, scoped routes, separate Jitsi signing material, and production observability/deployment contracts; expanded tasks through GitOps validation.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/23/TINYIDP-PLUGIN-001--plugin-api-for-downstream-integrations-and-jitsi-token-bridging/design-doc/02-tinyidp-plugin-system-and-jitsi-integration-analysis-design-and-implementation-guide.md — Full system guide


## 2026-07-23

Validated the full plugin-system and Jitsi integration guide and uploaded it to reMarkable at /ai/2026/07/23/TINYIDP-PLUGIN-001.


## 2026-07-23

Step 1: composed production settings through the shared Glazed source chain and added provenance/security tests (commit 294e343).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/sections/production/section.go — Reusable production section

## 2026-07-23

Step 2: implemented and tested the compiled-in plugin API, immutable registry, lifecycle, scoped routing, client requirements, and readiness aggregation (commit 513b7b9).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/pluginhost/host.go — Plugin host kernel

## 2026-07-23

Step 3: implemented the encrypted durable one-time OIDC broker, PKCE/nonce validation, in-process exchange, and SQLite migration (commit 4df3a9b).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/pluginhost/oidcbroker/broker.go — OIDC broker core

## 2026-07-23

Step 4: implemented the bounded versioned Jitsi Goja authorization policy, TypeScript contract, array schemas, pool lifecycle, and failure tests (commit a5eecf1).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/plugins/jitsi/policy.go — Jitsi Goja policy

## 2026-07-23

Step 5: implemented the complete Jitsi token bridge runtime (commit a0437f6)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/plugins/jitsi/runtime.go — Browser OIDC policy signing redirect and audit runtime


## 2026-07-23

Step 6: wired compiled plugins into serve-production with lifecycle and readiness composition (commit c6f98bd)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve_production.go — Production preparation build mounting readiness and cleanup

## 2026-07-23

Step 7: added the internal administration listener, Prometheus exporter, and bounded Jitsi OpenTelemetry instrumentation (commit 91f81f5)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/observability/prometheus.go — Internal health readiness and Prometheus surface

## 2026-07-23

Reconciled the ticket overview and authoritative intern-facing guide with the
implemented plugin phases, clarified the remaining deployment and end-to-end
validation work, and prepared the updated guide for reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/23/TINYIDP-PLUGIN-001--plugin-api-for-downstream-integrations-and-jitsi-token-bridging/design-doc/02-tinyidp-plugin-system-and-jitsi-integration-analysis-design-and-implementation-guide.md — Authoritative plugin system guide

## 2026-07-23

Step 9: repaired absent-collection policy input, added typed signup and account chooser intents, and made callback completion CSP-safe (commits ac161d3 and b4cedfa)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/plugins/jitsi/runtime.go — Browser completion boundary


## 2026-07-23

Step 10: added and validated the complete local Jitsi Compose stack, eight-case browser matrix, provider logout, Prosody enforcement, and two-browser connected media (commits e9c25b9, fe59277, and f552483)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/examples/tinyidp-jitsi/compose.yaml — Local deployment

## 2026-07-23

Step 11: added and live-API-validated the Kubernetes, Vault Secrets Operator, pinned Jitsi Helm, Prosody token-mode, NetworkPolicy, and coordinated HS256 rotation contract (commit 7f6425d)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/kubernetes/tinyidp-jitsi/README.md — Production deployment contract and validation procedure

## 2026-07-23

Step 12: merged current main, added generated Logcopter package areas, isolated parallel administration listeners, and passed the complete Go suite (commits 947c47c and 3a80254)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/18/TINYIDP-K3S-MSGDESK-PROD-001--production-k3s-deployment-of-standalone-tiny-idp-and-message-desk/scripts/01-two-process-harness/two_process_test.go — Parallel production-process listener isolation

## 2026-07-23

Step 13: made Jitsi rejection handling fail closed when durable audit delivery fails and added safe themed-response coverage (commit 574990b)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/plugins/jitsi/runtime.go — Explicit rejection-audit delivery semantics

## 2026-07-23

Step 14: published the merged plugin image, added and merged the multi-source
Argo/Vault/Hetzner deployment in infrastructure PR 200, provisioned scoped
Vault records, and applied the one-rule JVB firewall update (infrastructure
commits 60d11f2, 6527960, and 608c1b3).

### Related Files

- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/gitops/applications/tinyidp-jitsi.yaml — Pinned multi-source production application
- /home/manuel/code/wesen/2026-03-27--hetzner-k3s/main.tf — JVB UDP 10000 firewall rule

## 2026-07-23

Step 15: diagnosed the live local-path WaitForFirstConsumer/Argo wave deadlock
and moved the PVC beside its Deployment consumer (commit 9e9befb, PR 19).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/kubernetes/tinyidp-jitsi/persistent-volume-claim.yaml — PVC/consumer sync-wave invariant

## 2026-07-23

Step 16: fixed the local-path init-container ownership/mode ordering while
preserving the minimal `CAP_CHOWN` capability contract (commit 8f41210).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/kubernetes/tinyidp-jitsi/deployment.yaml — State directory ownership and mode initialization
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/deploy/kubernetes/tinyidp-jitsi/scripts/validate.sh — Ordering regression check
