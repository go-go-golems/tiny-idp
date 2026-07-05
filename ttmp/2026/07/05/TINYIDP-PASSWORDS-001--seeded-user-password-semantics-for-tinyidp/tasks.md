# Tasks

## Phase 0 — Ticket and design package

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing seeded-user password semantics guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-PASSWORDS-001 --stale-after 30`

## Phase 1 — Schema and registry

- [ ] Add optional `password` field to `scenario.SeededUser`
- [ ] Add optional password metadata to `scenario.Scenario`
- [ ] Copy seeded-user password into the scenario during conversion
- [ ] Add JSON/YAML seeded-user password tests
- [ ] Confirm users without passwords preserve current permissive behavior

## Phase 2 — Authorize POST validation

- [ ] Read submitted password from the authorize POST form
- [ ] Reject wrong configured passwords with generic error text
- [ ] Ensure wrong password creates no session
- [ ] Ensure wrong password creates no authorization code
- [ ] Preserve built-in scenario behavior when no password is configured

## Phase 3 — Docs and examples

- [ ] Update login page copy away from unconditional "ignored"
- [ ] Update README seeded users section
- [ ] Update Glazed help reference page
- [ ] Update example users once portable config examples exist

## Phase 4 — Integration validation

- [ ] Add server tests for correct, wrong, missing, and unconfigured password cases
- [ ] Update xgoja smoke helpers to submit configured passwords if users file gains passwords
- [ ] Add negative smoke only if it remains fast and clear
- [ ] Update changelog and related file links after implementation
