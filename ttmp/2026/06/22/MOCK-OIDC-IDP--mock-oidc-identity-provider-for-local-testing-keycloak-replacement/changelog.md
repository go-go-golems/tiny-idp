# Changelog

## 2026-06-22

- Initial workspace created


## 2026-06-22

Step 1: Created ticket MOCK-OIDC-IDP, design doc, phased task breakdown, and initial diary. No code yet.

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/design-doc/01-mock-oidc-idp-design-and-implementation-guide.md — Intern-ready design and implementation guide


## 2026-06-22

Step 2: Phase 0 baseline OIDC happy path — main.go + tests, go build/vet/test green (commit d473d513).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/cmd/tinyidp/main.go — server


## 2026-06-22

Step 3: Phase 1 multiple synthetic users + refactor into internal/server with go:embed login page (commit f9ece67).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/authorize.go — GET+POST login flow + parseAuthorizeRequest (commit f9ece67)


## 2026-06-22

Step 4: Phase 2 scenario registry — *Scenario threaded through handlers, one-file-add property tested (commit 6454cd3).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go — Scenario + Registry (commit 6454cd3)


## 2026-06-22

Step 5: Phase 3 self-documenting login page — scenarios rendered from registry.Grouped() (one-click buttons).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/embed.go — scenarioGroups bridges registry to template (Phase 3)


## 2026-06-22

Step 6: Phase 4 high-value failure scenarios — 17 scenarios registered as data, matrix test (37 tests green). MVP (0-4) complete.

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go — 17 failure scenarios as data entries (Phase 4)


## 2026-06-22

Step 7: delivery — docmgr doctor clean (vocab added), bundle uploaded + verified on reMarkable at /ai/2026/06/22/MOCK-OIDC-IDP.

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/reference/01-implementation-diary.md — delivery step


## 2026-06-22

Step 8: Adopted Glazed command framework — reusable oidc field section, layered config (defaults<config<env<args<flags), profile-ready, embedded help. stdlib-only decision superseded for CLI layer (commit 871eae0).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/sections/oidc/section.go — reusable OIDC field section (commit 871eae0)


## 2026-06-22

Step 9: Profiles (profiles.yaml + --profile, full precedence chain) + print-config command (second consumer of reusable oidc section). 48 tests green (commits ca2ada2, 0257f23).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/cmds/profiles.go — ProfileMiddlewaresFunc wires GatherFlagsFromProfiles (commit ca2ada2)


## 2026-06-22

Step 10: Phase 5 multiple clients — client registry (dev-client/public-spa/web-app), per-client redirect/PKCE/scope, cross-client code rejection. 60 tests green (commit 5fed666).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/client/client.go — Client + Registry + builtins (commit 5fed666)


## 2026-06-22

Step 11: Merge configured client into builtin (resolve Step 10 open question) — RequirePKCE/Secret/AllowedScopes preserved, redirect URIs unioned. 71 tests green (commit c9101d8).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/client/client.go — Merge(base


## 2026-06-22

Step 12: Phase 6 session layer — session cookie + prompt=none/login + max_age + login_hint + auth_time carried from login. 78 tests green (commit 20d210f).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/session.go — session store + cookie + prompt/max_age helpers (commit 20d210f)


## 2026-06-22

Step 13: Phase 7 claim variants — ExtraClaims/OmitClaims honored by ID token + userinfo; 9 claim-bearing scenarios (admin/viewer/tenant-a-admin/unicode-name/no-email/...). 86 tests green (commit b2099d0).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go — ExtraClaims + OmitClaims fields + 9 claim-variant scenarios (commit b2099d0)



## 2026-06-22

Step 14: Phase 10 JWKS/key rotation — multi-key JWKS (dev-key-1/rotated-key-2/bad-sig-key, shared via sync.Once), Scenario.SignKey (rotated/unknown-kid/bad-sig), server-level jwksMode (normal/500/slow/empty) + /debug/jwks-mode, 3 new scenarios, generalized test verify helper to look up kid. 105 tests green.

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/jwt.go — multi-key signJWT + jwks() + shared keys + jwksMode (commit d75aa44)
- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/scenario/scenario.go — SignKey field + 3 JWKS scenarios (commit d75aa44)
- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/debug.go — /debug/jwks-mode + reset restores normal (commit d75aa44)

Step 15: Phase 11 RP-initiated logout — /end-session (id_token_hint deletes by subject, post_logout_redirect_uri validated client-scoped, state forwarded, cookie cleared), client.PostLogoutRedirectURIs allowlist + Merge, end_session_endpoint in discovery. 113 tests green (commit d75aa44).

### Related Files

- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/server/logout.go — /end-session handler (commit d75aa44)
- /home/manuel/code/wesen/2026-06-22--mock-oidc-idp/internal/client/client.go — PostLogoutRedirectURIs + AllowsPostLogoutRedirectURI + Merge (commit d75aa44)

## 2026-07-04

Step 16: added Keycloak-style path-based issuer route support and validated xgoja Step 06 against /realms/personal-inbox (commit 64a6165).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/server.go — Registers root and issuer-path route sets
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/internal/server/server_test.go — Covers prefixed discovery/authorize/token/userinfo
- /home/manuel/workspaces/2026-06-12/goja-express-auth/2026-06-22--mock-oidc-idp/ttmp/2026/06/22/MOCK-OIDC-IDP--mock-oidc-identity-provider-for-local-testing-keycloak-replacement/reference/01-implementation-diary.md — Step 16 diary

