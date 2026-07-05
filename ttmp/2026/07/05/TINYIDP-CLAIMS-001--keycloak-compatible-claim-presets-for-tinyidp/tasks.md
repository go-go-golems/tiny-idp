# Tasks

## Phase 0 — Scope correction and bookkeeping

- [x] Replace Keycloak-specific design language with generic claim preset language
- [x] Record the user correction in the implementation diary
- [x] Update task list to be detailed enough for step-by-step tracking
- [x] Commit the scope-correction docs

## Phase 1 — Seeded-user schema

- [ ] Add `Groups []string` to `SeededUser` with JSON/YAML tags
- [ ] Add `Roles []string` to `SeededUser` with JSON/YAML tags
- [ ] Add `Tenant string` to `SeededUser` with JSON/YAML tags
- [ ] Add `PreferredUsername string` to `SeededUser` with JSON/YAML tags
- [ ] Add `Locale string` to `SeededUser` with JSON/YAML tags
- [ ] Keep `Claims map[string]any` and `OmitClaims []string` unchanged

## Phase 2 — Claim expansion helper

- [ ] Add helper to trim scalar string claim values
- [ ] Add helper to trim string-list values and drop empty entries while preserving order
- [ ] Expand non-empty `groups` into `extra["groups"]`
- [ ] Expand non-empty `roles` into `extra["roles"]`
- [ ] Expand non-empty `tenant` into `extra["tenant"]`
- [ ] Expand non-empty `preferred_username` into `extra["preferred_username"]`
- [ ] Expand non-empty `locale` into `extra["locale"]`

## Phase 3 — Merge semantics

- [ ] Apply convenience fields before explicit `Claims`
- [ ] Preserve explicit `Claims` override behavior
- [ ] Preserve `email_verified` handling after explicit `Claims`
- [ ] Preserve `OmitClaims` behavior without changing token/userinfo code

## Phase 4 — Unit tests

- [ ] Add test for top-level groups/roles/tenant/preferred_username/locale
- [ ] Add test proving explicit `claims` override top-level groups/roles
- [ ] Add test proving empty/whitespace list entries are dropped
- [ ] Add YAML load test covering generic top-level fields
- [ ] Run `go test ./internal/scenario -count=1`

## Phase 5 — Server-flow tests

- [ ] Add or update server flow test proving generic preset claims appear in ID token
- [ ] Assert the same claims appear in `/userinfo`
- [ ] Run `go test ./internal/server -count=1`

## Phase 6 — Docs and examples

- [ ] Update README seeded-user documentation
- [ ] Update Glazed reference page seeded-user documentation
- [ ] Add or update example users file with generic top-level fields
- [ ] Avoid provider-specific realm/client role examples

## Phase 7 — Final validation and diary

- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Update diary with exact command output
- [ ] Update changelog and doc relations
- [ ] Run `docmgr doctor --ticket TINYIDP-CLAIMS-001 --stale-after 30`
- [ ] Commit implementation and docs
