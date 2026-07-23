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
