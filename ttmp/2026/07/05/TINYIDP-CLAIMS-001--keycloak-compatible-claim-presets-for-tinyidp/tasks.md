# Tasks

## Phase 0 — Ticket and design package

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing Keycloak-compatible claim presets guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-CLAIMS-001 --stale-after 30`

## Phase 1 — Preset model and expansion helper

- [ ] Add `KeycloakClaims` / `KeycloakClaimPreset` type
- [ ] Add optional `keycloak` block to `SeededUser`
- [ ] Implement expansion for `preferred_username`, `groups`, `realm_access.roles`, and `resource_access.<client>.roles`
- [ ] Add unit tests for every claim shape

## Phase 2 — Seeded-user integration

- [ ] Expand `keycloak` preset before explicit `claims`
- [ ] Make explicit `claims` override preset-generated fields
- [ ] Apply `omit_claims` after preset and explicit claims
- [ ] Preserve current email/email_verified behavior

## Phase 3 — Docs and examples

- [ ] Add `examples/users/roles-demo-users.yaml`
- [ ] Document exact emitted JSON for each preset field
- [ ] Update README seeded-user section
- [ ] Update Glazed help reference page

## Phase 4 — Integration validation

- [ ] Add server flow test proving ID token includes preset claims
- [ ] Add userinfo test proving userinfo includes preset claims
- [ ] Add override test proving explicit claims win
- [ ] Add omit test proving preset claims can be removed
- [ ] Decide whether any xgoja/appauth example should consume the preset
- [ ] Update changelog and related file links after implementation
