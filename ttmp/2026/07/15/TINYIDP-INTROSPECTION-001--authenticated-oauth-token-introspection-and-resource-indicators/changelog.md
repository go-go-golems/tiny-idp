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


## 2026-07-16

Added SQLite-backed opaque-token lifecycle proof and corrected post-authentication inactive-token classification.

### Related Files

- internal/fositeadapter/provider.go — Authenticated resource callers before opaque-token classification
- internal/fositeadapter/provider_test.go — Invalid resource-client credential coverage
- internal/fositeadapter/sqlstore_test.go — Refresh-rotation and reuse-revocation introspection matrix


## 2026-07-16

Completed memory/SQLite introspection security matrix for caller authorization, token lifecycle, rate limiting, and audit redaction.

### Related Files

- internal/fositeadapter/provider_test.go — Memory verification matrix
- internal/fositeadapter/sqlstore_test.go — SQLite expiry fixture and regression


## 2026-07-16

Made bearer-only DPoP rejection and root/path introspection discovery executable.

### Related Files

- internal/fositeadapter/hardening_test.go — Verify root and path issuer discovery endpoints
- internal/fositeadapter/provider.go — Reject unsupported DPoP before code consumption

