# Tasks

## Phase 0 — Ticket baseline and detailed tracking

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing seeded-user password semantics guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-PASSWORDS-001 --stale-after 30`
- [x] Replace coarse task list with detailed implementation checklist
- [x] Record continuation prompt and execution plan in diary
- [x] Commit detailed task/diary baseline

## Phase 1 — Seeded-user schema and conversion

- [x] Add optional `Password string` field to `scenario.SeededUser` with JSON/YAML tags
- [x] Add optional `Password string` metadata to `scenario.Scenario`
- [x] Trim whitespace from configured seeded-user password during conversion
- [x] Copy trimmed seeded-user password into the resulting scenario
- [x] Preserve empty password as "no password required"
- [x] Keep existing seeded-user identity and generic claim preset behavior unchanged

## Phase 2 — Scenario tests

- [x] Add direct `SeededUsersToScenarios` test proving password is copied
- [x] Add YAML users-file test proving password is loaded
- [x] Add JSON users-file test proving password is loaded
- [x] Add test proving missing password remains empty/permissive
- [x] Run `go test ./internal/scenario -count=1`

## Phase 3 — Authorize POST validation

- [x] Read submitted password from authorize POST form
- [ ] Add small helper for configured-password validation
- [ ] Reject wrong configured passwords with generic `invalid login or password` text
- [ ] Return `401 Unauthorized` for wrong configured password
- [ ] Reject missing configured passwords the same way as wrong passwords
- [ ] Validate password before auth-error scenario redirects
- [ ] Preserve no-password scenarios as permissive

## Phase 4 — Server-flow tests

- [ ] Add test that password-protected seeded user succeeds with correct password
- [ ] Add test that wrong password returns `401`
- [ ] Add test that missing password returns `401`
- [ ] Assert wrong password creates no session
- [ ] Assert wrong password creates no authorization code
- [ ] Add/keep test that seeded user without password accepts arbitrary submitted password
- [ ] Add/keep test that built-in user without password accepts arbitrary submitted password
- [ ] Run `go test ./internal/server -count=1`

## Phase 5 — Login UI and documentation

- [ ] Update `internal/server/static/login.html` copy away from unconditional "password is ignored"
- [ ] Update README seeded-user docs with optional password semantics
- [ ] Update Glazed reference page seeded-user docs with optional password semantics
- [ ] Update `examples/users/generic-claims-users.yaml` with fixture passwords if appropriate
- [ ] Avoid implying production security or real account management

## Phase 6 — Full validation and bookkeeping

- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Update diary with exact command output
- [ ] Relate changed implementation/docs files to the ticket docs
- [ ] Update changelog with implementation summary
- [ ] Run `docmgr doctor --ticket TINYIDP-PASSWORDS-001 --stale-after 30`
- [ ] Commit implementation and docs
