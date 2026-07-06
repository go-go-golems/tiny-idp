# Changelog

## 2026-07-06

- Initial workspace created


## 2026-07-06

Created device authorization grant design package with implementation phases, API design, tests, and file references.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/design-doc/01-device-authorization-grant-design-and-implementation-guide.md — Primary design guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/reference/01-implementation-diary.md — Creation diary
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/tasks.md — Detailed implementation checklist


## 2026-07-06

Uploaded device authorization design bundle to reMarkable at /ai/2026/07/06/TINYIDP-DEVICE-001 and verified docmgr doctor passes.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/design-doc/01-device-authorization-grant-design-and-implementation-guide.md — Uploaded design guide
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/reference/01-implementation-diary.md — Uploaded diary


## 2026-07-06

Implemented native device authorization runtime core with routes, polling, approval form, debug visibility, and server tests.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/device.go — Device authorization endpoint implementation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/token.go — Device-code token exchange implementation


## 2026-07-06

Documented native device authorization and validated the full repository with tests, build, help rendering, and a curl smoke.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/README.md — README device flow update
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/cmd/tinyidp/doc/pages/tutorial-device-authorization.md — Device authorization tutorial


## 2026-07-06

Ticket closed


## 2026-07-06

Uploaded final implementation bundle to reMarkable at /ai/2026/07/06/TINYIDP-DEVICE-001.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/07/06/TINYIDP-DEVICE-001--oauth-device-authorization-grant-for-tinyidp/reference/01-implementation-diary.md — Records final reMarkable upload


## 2026-07-06

Addressed PR review feedback: device authorization now reuses confidential-client authentication, approval rejects blank logins in tests/docs, and slow_down persists the RFC backoff interval increase.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/device.go — Device authorization client authentication
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/device_test.go — Regression tests for review feedback
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/token.go — Shared client authentication and slow_down interval persistence


## 2026-07-06

Addressed second PR review pass: invalid DPoP proofs no longer consume approved device grants, and completed approval decisions cannot be overwritten by duplicate browser submits.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/device.go — Guards completed grants before approval/denial mutation
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/device_test.go — Duplicate-submit regression coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/dpop_test.go — Invalid DPoP retry regression coverage
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/token.go — Moves approved grant deletion after DPoP validation

