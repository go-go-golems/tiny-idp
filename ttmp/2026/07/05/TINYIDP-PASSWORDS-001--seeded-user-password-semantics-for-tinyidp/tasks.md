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
- [x] Add small helper for configured-password validation
- [x] Reject wrong configured passwords with generic `invalid login or password` text
- [x] Return `401 Unauthorized` for wrong configured password
- [x] Reject missing configured passwords the same way as wrong passwords
- [x] Validate password before auth-error scenario redirects
- [x] Preserve no-password scenarios as permissive

## Phase 4 — Server-flow tests

- [x] Add test that password-protected seeded user succeeds with correct password
- [x] Add test that wrong password returns `401`
- [x] Add test that missing password returns `401`
- [x] Assert wrong password creates no session
- [x] Assert wrong password creates no authorization code
- [x] Add/keep test that seeded user without password accepts arbitrary submitted password
- [x] Add/keep test that built-in user without password accepts arbitrary submitted password
- [x] Run `go test ./internal/server -count=1`

## Phase 5 — Login UI and documentation

- [x] Update `internal/server/static/login.html` copy away from unconditional "password is ignored"
- [x] Update README seeded-user docs with optional password semantics
- [x] Update Glazed reference page seeded-user docs with optional password semantics
- [x] Update `examples/users/generic-claims-users.yaml` with fixture passwords if appropriate
- [x] Avoid implying production security or real account management

## Phase 6 — Full validation and bookkeeping

- [x] Run `GOWORK=off go test ./... -count=1`
- [x] Run `GOWORK=off go build ./cmd/tinyidp`
- [x] Update diary with exact command output
- [x] Relate changed implementation/docs files to the ticket docs
- [x] Update changelog with implementation summary
- [x] Run `docmgr doctor --ticket TINYIDP-PASSWORDS-001 --stale-after 30`
- [x] Commit implementation and docs
