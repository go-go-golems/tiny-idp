# Changelog

## 2026-07-11

- Initial workspace created
- Added a 1,122-line intern-ready architecture and implementation guide, 69 stable-ID phased tasks, detailed diary, and nine-document primary source packet.
- Verified complete baseline test suites in tiny-idp, go-go-goja, and go-go-objects.
- Implemented provider-neutral OIDC naming and origin/path-restricted in-process discovery, token, and JWKS transport in go-go-goja (`ebc5600`).
- Exposed authenticated planned-route actors to trusted native services in go-go-goja (`ce36bf8`).
- Added HMAC actor-bound Durable Object dispatch, xgoja actor-bound APIs, JSON codecs, and default-denied raw gateways in go-go-objects (`ec73ddd`).
- Scoped application identities and SQL uniqueness to the OIDC issuer-and-subject tuple in go-go-goja (`43a69d7`).
- Published the index, design, implementation diary, tasks, and changelog as `TINYIDP-XAPP-001 Self Contained Identity Objects e93d304.pdf` under `/ai/2026/07/11/TINYIDP-XAPP-001` on reMarkable.
- Added and generated the `cmd/tinyidp-xapp` Glazed command, xgoja runtime package, trusted routes, bounded USER_STATE object, pnpm/Bootstrap frontend shell, embedded assets, declarations, and generation tests (`5176052`).
- Added configurable issuer-aware IdP cookie names and paths with host-session coexistence tests; corrected and documented the xgoja embedded-assets generation contract (`577f253`).
- Added the working development custom host and complete login-to-private-object vertical test (`3ca71e5`), supported by composed HTTP/OIDC fixes in go-go-goja (`2d7878d`) and side-effect-free external manager precedence in go-go-objects (`46ba195`).
- Published the refreshed five-document bundle as `TINYIDP-XAPP-001 Vertical Slice 86595c6.pdf` under the ticket's reMarkable directory.
- Added idempotent persistent product-state initialization, owner-only secret/password files, exact RP/user/key reconciliation, and completion-manifest validation (`acbf207`).
- Added initialized-state refusal and persistent production construction for identity, application auth/session/audit, and actor-bound object stores (`a9b562e`).
- Added initialized-state TLS serving with aggregate health/readiness, global body limits, periodic maintenance, bounded timeouts, and graceful shutdown (`568367b`).
- Published `TINYIDP-XAPP-001 Initialized TLS b1b22e3.pdf` to the ticket's reMarkable directory.

## 2026-07-12

Recorded the real TLS application handoff: successful initialization/readiness, two-user fixture, draft Chromium harness, exact Playwright artifact failures, cleanup state, security review questions, and restart procedure.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/scripts/01_real_browser_e2e.py — Uncommitted browser checkpoint handed to the next engineer

## 2026-07-12

Added an intern-ready ownership map, security model, executable continuation procedure, scenario matrix, first-contribution sequence, and review boundaries; recorded Step 37 in the diary.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/11/TINYIDP-XAPP-001--self-contained-xgoja-identity-and-durable-object-application/playbook/01-new-intern-handoff-and-continuation-playbook.md — Primary onboarding and continuation entry point
