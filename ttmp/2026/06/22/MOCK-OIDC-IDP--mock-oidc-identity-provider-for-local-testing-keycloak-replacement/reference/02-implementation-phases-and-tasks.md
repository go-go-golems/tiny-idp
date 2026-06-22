---
Title: Implementation Phases and Tasks
Ticket: MOCK-OIDC-IDP
Status: active
Topics:
    - oidc
    - go
    - testing
    - identity
    - auth
DocType: reference
Intent: short-term
Owners: []
RelatedFiles:
    - Path: cmd/tinyidp/main.go:Phase 0 entrypoint
    - Path: internal/client/client.go:Client registry (Phase 5)
    - Path: internal/jwt/jwt.go:JWT/JWKS/PKCE
    - Path: internal/scenario/scenario.go
      Note: Phase 2 scenario registry
    - Path: internal/scenario/scenario.go:Scenario registry
    - Path: internal/server/server.go:HTTP handlers
    - Path: internal/user/user.go:User derivation
ExternalSources: []
Summary: Concrete, checkbox-tracked breakdown of every implementation phase and the tasks within each phase.
LastUpdated: 2026-06-22T15:00:00-04:00
WhatFor: Track exact work items per phase; mark tasks complete as implementation progresses.
WhenToUse: Use as the live checklist while implementing; mirror completion into tasks.md.
---


# Implementation Phases and Tasks

> This is the executable checklist. The design doc (`design-doc/01-...`) explains *why*; this doc tracks *what* and *when*.
> MVP cutoff = Phases 0–4. Later phases are documented but deferred.

## Legend

Each task is a single, reviewable commit unit where reasonable. A task is `[x]` only when its validation step passes.

## Phase 0 — Baseline OIDC happy path

**Goal:** a normal OIDC client can log in as `alice` and receive an ID token + access token.

- [x] P0.1 `go mod init github.com/manuel/tinyidp`
- [x] P0.2 Scaffold `cmd/tinyidp/main.go`: env parsing (`OIDC_ISSUER`, `OIDC_ADDR`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URIS`), RSA key generation, `ListenAndServe` with `withCORS`.
- [x] P0.3 Implement `server` struct with `issuer`, `clientID`, `clientSecret`, `redirectURIs`, `key`, `kid`, `codes`, `tokens`, `sync.Mutex`.
- [x] P0.4 Implement `discovery()` returning all required metadata fields.
- [x] P0.5 Implement `jwks()` returning the public RSA key as a JWK (`kty/use/kid/alg/n/e`).
- [x] P0.6 Implement `authorize()` GET: validate `response_type`, `client_id`, `redirect_uri` allowlist, `scope` includes `openid`; store code; redirect with `code` + `state`.
- [x] P0.7 Implement `token()` POST: client auth (Basic + post), code pop + expiry, `client_id`/`redirect_uri` match, PKCE verify, issue access + RS256 ID token; echo `nonce`.
- [x] P0.8 Implement `userinfo()` POST/GET: bearer token lookup + expiry, return `sub`/`email`/`email_verified`/`name`.
- [x] P0.9 Implement helpers: `signJWT`, `verifyPKCE`, `randomB64`, `b64`, `writeJSON`, `tokenError`, `parseCSV`, `env`, `hasScope`.
- [x] P0.10 Add `/healthz` returning `ok`.
- [x] P0.11 Write `README.md` with run/config instructions and env var table.
- [x] P0.12 Validate: `go build ./...`, `go vet ./...`, `go run .`; `curl` discovery + JWKS; manual authorize→token→userinfo.

**Exit criteria:** ID token issued for `alice`, signature verifiable against `/jwks`, `iss`/`aud`/`exp`/`nonce` correct.

## Phase 1 — Multiple synthetic users

**Goal:** log in as any username; distinct stable `sub`s.

- [x] P1.1 Extract `internal/user/user.go` with `User` + `FromLogin(login)` (stable `sub = user-<b64(sha16)>`, email/name derivation, normalization).
- [x] P1.2 Convert `/authorize` to GET (render login form) + POST (submit login) using `html/template`.
- [x] P1.3 Add `parseAuthorizeRequest` + `authorizeRequest` struct; echo authorize params as hidden fields.
- [x] P1.4 Add `hiddenAuthorizeFields`, `normalizeLogin`, `errText`; remove fixed default user from `server`.
- [x] P1.5 Validate: log in as `alice` then `bob`; confirm distinct stable `sub`s; confirm arbitrary usernames work.

**Exit criteria:** distinct stable subjects per login; login page round-trips authorize params.

## Phase 2 — Scenario registry

**Goal:** adding a failure case is one scenario, not three handler edits.

- [x] P2.1 Create `internal/scenario/scenario.go` with `Scenario` struct (`Name`, `Description`, `User`, `AuthError`, `TokenError`, `UserInfoError`, `MutateClaims`).
- [x] P2.2 Implement `Registry` (`New`, `Lookup`, `All`) backed by a map; pre-register normal scenarios (`alice`, `bob`).
- [x] P2.3 Replace `FailureMode string` on `authCode`/`accessToken` with `*scenario.Scenario`; thread scenario through authorize→token→userinfo.
- [x] P2.4 Refactor handlers to call `registry.Lookup(login)` once and branch on scenario fields (no per-handler login switches).
- [x] P2.5 Add `redirectOAuthError` helper for auth-error scenarios.
- [x] P2.6 Validate: add a throwaway scenario, confirm only `scenario.go` changed; unknown logins fall back to derived normal user.

**Exit criteria:** scenario added in one place; all handlers read from the registry.

## Phase 3 — Login page with selectable scenarios

**Goal:** the login page is self-documenting.

- [x] P3.1 Group `registry.All()` into categories (normal / auth-failure / token-failure / id-token-failure / userinfo-failure).
- [x] P3.2 Render grouped scenario buttons that fill+submit the `login` field.
- [x] P3.3 Keep manual login input for arbitrary usernames.
- [x] P3.4 Validate: open `/authorize?...` in a browser; every listed scenario reachable in one click and matching the registry.

**Exit criteria:** login page lists every supported scenario; one-click login works.

## Phase 4 — High-value failure scenarios

**Goal:** reproduce real OIDC client bugs.

- [x] P4.1 Auth-error scenarios: `fail-access-denied`, `fail-login-required`, `fail-consent-required`, `fail-server-error`.
- [x] P4.2 Token-error scenarios: `token-invalid-grant`, `token-server-error`, `token-slow` (10s sleep).
- [x] P4.3 ID-token mutation scenarios: `id-expired`, `id-wrong-aud`, `id-wrong-iss`, `id-missing-email`, `id-email-unverified`, `id-bad-nonce`, `id-future-iat`.
- [x] P4.4 UserInfo-error scenarios: `userinfo-401`, `userinfo-500`, `userinfo-sub-mismatch`.
- [x] P4.5 Validate each scenario end-to-end against a sample RP flow; confirm failure surfaces where expected.

**Exit criteria:** all listed scenarios produce their documented failure.

## Phase 5 — Multiple clients

- [x] P5.1 `internal/client/client.go` with `Client{ID, Secret, RedirectURIs, RequirePKCE, AllowedScopes}`.
- [x] P5.2 Client registry; replace single-client fields on `server`.
- [x] P5.3 Predefined clients: `public-spa` (no secret, PKCE required), `web-app` (secret required), `dev-client` (permissive).
- [x] P5.4 Discovery advertises `token_endpoint_auth_methods_supported` accurately (already correct from Phase 0).

## Phase 6 — Session cookie, prompt, max_age

- [x] P6.1 IdP session cookie + store.
- [x] P6.2 `prompt=none` (→ `login_required` if no session), `prompt=login` (force form), valid session (skip form).
- [x] P6.3 `max_age` handling + `auth_time` claim.
- [x] P6.4 `login_hint` prefill.

## Phase 7 — Claims and authorization shapes

- [x] P7.1 Claim-bearing scenarios: `admin`, `viewer`, `no-email`, `unverified-email`, `no-groups`, `many-groups`, `tenant-a-admin`, `tenant-b-viewer`, `unicode-name`.
- [x] P7.2 Emit `groups`, `roles`, `tenant`, `preferred_username`, `locale` (via ExtraClaims, honored by ID token + userinfo).

## Phase 8 — Debug UI (deferred)

- [ ] P8.1 `/debug`, `/debug/sessions`, `/debug/codes`, `/debug/tokens`, `/debug/reset` (loopback only).

## Phase 9 — Refresh tokens (deferred)

- [ ] P9.1 `offline_access`, `refresh_token` grant, rotation, reuse detection.

## Phase 10 — JWKS/key rotation (deferred)

- [ ] P10.1 Multiple kids, `kid-not-found`, bad signature, JWKS 500/slow/empty.

## Phase 11 — Logout (deferred)

- [ ] P11.1 `/end-session`, `id_token_hint`, `post_logout_redirect_uri`, `state`.

## Phase 12 — Go test helper (deferred)

- [ ] P12.1 Public `Start(t testing.TB, opts Options) *Provider` returning `Issuer()`; `t.Cleanup(Close)`.
