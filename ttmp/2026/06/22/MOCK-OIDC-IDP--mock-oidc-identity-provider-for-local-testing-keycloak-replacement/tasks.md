# Tasks

## MVP (Phases 0–4)

### Phase 0 — Baseline OIDC happy path
- [x] P0.1 `go mod init`
- [x] P0.2 Scaffold `cmd/tinyidp/main.go` (env, key, ListenAndServe)
- [x] P0.3 `server` struct + state
- [x] P0.4 `discovery()`
- [x] P0.5 `jwks()`
- [x] P0.6 `authorize()` GET
- [x] P0.7 `token()` POST
- [x] P0.8 `userinfo()`
- [x] P0.9 Helpers (signJWT, verifyPKCE, etc.)
- [x] P0.10 `/healthz`
- [x] P0.11 `README.md`
- [x] P0.12 Validate build/vet/run + curl

### Phase 1 — Multiple synthetic users
- [x] P1.1 `internal/user/user.go` + `FromLogin`
- [x] P1.2 `/authorize` GET+POST with `html/template`
- [x] P1.3 `parseAuthorizeRequest` + hidden fields
- [x] P1.4 Remove fixed default user
- [x] P1.5 Validate distinct stable subs

### Phase 2 — Scenario registry
- [x] P2.1 `internal/scenario/scenario.go` struct
- [x] P2.2 `Registry` + built-ins
- [x] P2.3 Thread `*Scenario` through authCode/accessToken
- [x] P2.4 Handlers use `registry.Lookup`
- [x] P2.5 `redirectOAuthError` helper
- [x] P2.6 Validate one-file scenario add

### Phase 3 — Login page with selectable scenarios
- [x] P3.1 Group scenarios by category
- [x] P3.2 Render grouped buttons
- [x] P3.3 Keep manual input
- [x] P3.4 Validate one-click scenarios

### Phase 4 — High-value failure scenarios
- [x] P4.1 Auth-error scenarios
- [x] P4.2 Token-error scenarios
- [x] P4.3 ID-token mutation scenarios
- [x] P4.4 UserInfo-error scenarios
- [x] P4.5 Validate each end-to-end

## Deferred (Phases 5–12)
- [ ] P5 Multiple clients
- [ ] P6 Session cookie, prompt, max_age
- [ ] P7 Claims/authorization shapes
- [ ] P8 Debug UI
- [ ] P9 Refresh tokens
- [ ] P10 JWKS/key rotation
- [ ] P11 Logout
- [ ] P12 Go test helper

## Completed

### Phase 4
- [x] P4.1 auth-error scenarios
- [x] P4.2 token-error scenarios
- [x] P4.3 ID-token mutation scenarios
- [x] P4.4 UserInfo-error scenarios
- [x] P4.5 validated each end-to-end

### Phase 3
- [x] P3.1 group scenarios by category
- [x] P3.2 render grouped buttons
- [x] P3.3 keep manual input
- [x] P3.4 validated one-click scenarios

### Phase 2
- [x] P2.1 scenario.go struct
- [x] P2.2 Registry + builtins
- [x] P2.3 *Scenario threaded through authCode/accessToken
- [x] P2.4 handlers use registry.Lookup
- [x] P2.5 redirectOAuthError helper
- [x] P2.6 validated one-file scenario add

### Phase 1
- [x] P1.1 internal/user/user.go + FromLogin
- [x] P1.2 /authorize GET+POST with html/template
- [x] P1.3 parseAuthorizeRequest + hidden fields
- [x] P1.4 removed fixed default user
- [x] P1.5 validated distinct stable subs

### Phase 0
- [x] P0.1 go mod init
- [x] P0.2 main.go scaffold
- [x] P0.3 server struct
- [x] P0.4 discovery()
- [x] P0.5 jwks()
- [x] P0.6 authorize() GET
- [x] P0.7 token() POST
- [x] P0.8 userinfo()
- [x] P0.9 helpers
- [x] P0.10 /healthz
- [x] P0.11 README.md
- [x] P0.12 build/vet/test green

## Notes

- MVP cutoff = Phases 0–4. Detailed per-task breakdown in `reference/02-implementation-phases-and-tasks.md`.
- All work stays inside `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp`.
