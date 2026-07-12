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
