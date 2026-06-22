# Tasks

## MVP (Phases 0–4)

### Phase 0 — Baseline OIDC happy path
- [ ] P0.1 `go mod init`
- [ ] P0.2 Scaffold `cmd/tinyidp/main.go` (env, key, ListenAndServe)
- [ ] P0.3 `server` struct + state
- [ ] P0.4 `discovery()`
- [ ] P0.5 `jwks()`
- [ ] P0.6 `authorize()` GET
- [ ] P0.7 `token()` POST
- [ ] P0.8 `userinfo()`
- [ ] P0.9 Helpers (signJWT, verifyPKCE, etc.)
- [ ] P0.10 `/healthz`
- [ ] P0.11 `README.md`
- [ ] P0.12 Validate build/vet/run + curl

### Phase 1 — Multiple synthetic users
- [ ] P1.1 `internal/user/user.go` + `FromLogin`
- [ ] P1.2 `/authorize` GET+POST with `html/template`
- [ ] P1.3 `parseAuthorizeRequest` + hidden fields
- [ ] P1.4 Remove fixed default user
- [ ] P1.5 Validate distinct stable subs

### Phase 2 — Scenario registry
- [ ] P2.1 `internal/scenario/scenario.go` struct
- [ ] P2.2 `Registry` + built-ins
- [ ] P2.3 Thread `*Scenario` through authCode/accessToken
- [ ] P2.4 Handlers use `registry.Lookup`
- [ ] P2.5 `redirectOAuthError` helper
- [ ] P2.6 Validate one-file scenario add

### Phase 3 — Login page with selectable scenarios
- [ ] P3.1 Group scenarios by category
- [ ] P3.2 Render grouped buttons
- [ ] P3.3 Keep manual input
- [ ] P3.4 Validate one-click scenarios

### Phase 4 — High-value failure scenarios
- [ ] P4.1 Auth-error scenarios
- [ ] P4.2 Token-error scenarios
- [ ] P4.3 ID-token mutation scenarios
- [ ] P4.4 UserInfo-error scenarios
- [ ] P4.5 Validate each end-to-end

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

(none yet)

## Notes

- MVP cutoff = Phases 0–4. Detailed per-task breakdown in `reference/02-implementation-phases-and-tasks.md`.
- All work stays inside `/home/manuel/code/wesen/2026-06-22--mock-oidc-idp`.
