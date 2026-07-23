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
