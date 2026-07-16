# Changelog

## 2026-07-15

- Initial workspace created


## 2026-07-15

Implemented authenticated RFC 7662 endpoint core and device resource-indicator propagation (commits f718d36, d5c7647); recorded operator runbook and strict HTTPS smoke script.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_token_handler.go — Device token audience propagation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Endpoint and audience policy


## 2026-07-15

Proved refresh-token rotation preserves the original granted resource audience through authenticated introspection (commit 866e0bb).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider_test.go — Refresh introspection regression

