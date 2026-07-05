# Tasks

## Phase 0 — Ticket and design package

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing portable config and smoke ergonomics guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-CONFIG-001 --stale-after 30`

## Phase 1 — Example config files

- [ ] Create `examples/configs/dev-root.yaml`
- [ ] Create `examples/configs/personal-inbox-root.yaml`
- [ ] Create `examples/configs/personal-inbox-realm.yaml`
- [ ] Create `examples/configs/public-spa-pkce.yaml`
- [ ] Create `examples/configs/confidential-web-app.yaml`
- [ ] Create `examples/users/personal-inbox-users.yaml`
- [ ] Document relative-path behavior for `users-file`

## Phase 2 — xgoja smoke snippets

- [ ] Add root-issuer xgoja Step 06 snippet
- [ ] Add realm-issuer xgoja Step 06 snippet
- [ ] Add Step 07 and Step 08 notes after realm validation
- [ ] Document common failure symptoms and fixes

## Phase 3 — Validation

- [ ] Validate `tinyidp print-config --config-file` for every example config
- [ ] Validate discovery for root and realm configs
- [ ] Validate xgoja Step 06 root issuer smoke
- [ ] Validate xgoja Step 06 realm issuer smoke
- [ ] Decide whether to add a Makefile target for config smoke validation

## Phase 4 — Docs and handoff

- [ ] Link examples from root README
- [ ] Update Glazed help reference/getting-started pages
- [ ] Update changelog and related file links after implementation
- [ ] Re-run docmgr doctor
