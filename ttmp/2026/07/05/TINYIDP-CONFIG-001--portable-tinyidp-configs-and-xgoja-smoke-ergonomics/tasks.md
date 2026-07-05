# Tasks

## Phase 0 — Ticket baseline and detailed tracking

- [x] Create docmgr ticket under the tinyidp repo `ttmp` root
- [x] Write intern-facing portable config and smoke ergonomics guide
- [x] Write implementation diary
- [x] Upload design bundle to reMarkable
- [x] Run `docmgr doctor --ticket TINYIDP-CONFIG-001 --stale-after 30`
- [x] Replace coarse task list with detailed implementation checklist
- [x] Record continuation prompt and execution plan in diary
- [x] Commit detailed task/diary baseline

## Phase 1 — Portable example config files

- [ ] Create `examples/configs/dev-root.yaml`
- [ ] Include root issuer, loopback addr, dev-client, and default dev redirects in `dev-root.yaml`
- [ ] Create `examples/configs/personal-inbox-root.yaml`
- [ ] Include root issuer, personal-inbox client ID, redirects, and users file in `personal-inbox-root.yaml`
- [ ] Create `examples/configs/personal-inbox-realm.yaml`
- [ ] Include path-based issuer and matching loopback addr in `personal-inbox-realm.yaml`
- [ ] Create `examples/configs/public-spa-pkce.yaml`
- [ ] Use builtin `public-spa` client and document PKCE-required behavior in comments
- [ ] Create `examples/configs/confidential-web-app.yaml`
- [ ] Use builtin `web-app` client and configured `dev-secret` in comments/config
- [ ] Create `examples/users/personal-inbox-users.yaml`
- [ ] Include Alice and Bob with fixed subjects, emails, passwords, groups, roles, tenant, preferred usernames, and locale
- [ ] Keep all example config files provider-neutral except optional path-based issuer compatibility notes

## Phase 2 — Relative path and smoke ergonomics docs

- [ ] Document that `oidc.users-file` is currently resolved relative to the process working directory
- [ ] Document the safest invocation pattern: run from repo root or use an absolute users-file path
- [ ] Add root-issuer xgoja Step 06 snippet
- [ ] Add path-issuer xgoja Step 06 snippet without requiring provider-specific claim shapes
- [ ] Add Step 07 note for Alice/Bob isolation with the same users file
- [ ] Add Step 08 note that device authorization remains xgoja-native, not tinyidp-hosted
- [ ] Document common failure symptoms and fixes
- [ ] Link examples from root README
- [ ] Update Glazed getting-started page
- [ ] Update Glazed reference page

## Phase 3 — Example config validation

- [ ] Validate `tinyidp print-config --config-file examples/configs/dev-root.yaml`
- [ ] Validate `tinyidp print-config --config-file examples/configs/personal-inbox-root.yaml`
- [ ] Validate `tinyidp print-config --config-file examples/configs/personal-inbox-realm.yaml`
- [ ] Validate `tinyidp print-config --config-file examples/configs/public-spa-pkce.yaml`
- [ ] Validate `tinyidp print-config --config-file examples/configs/confidential-web-app.yaml`
- [ ] Validate root discovery with `dev-root.yaml`
- [ ] Validate path-based discovery with `personal-inbox-realm.yaml`
- [ ] Capture exact validation output in diary

## Phase 4 — xgoja smoke decision and validation

- [ ] Decide whether this ticket adds Makefile targets or documents manual overrides only
- [ ] Validate xgoja Step 06 root issuer smoke if practical in this slice
- [ ] Validate xgoja Step 06 path issuer smoke if practical in this slice
- [ ] If xgoja validation is deferred, record exact reason and command to run later

## Phase 5 — Final validation and bookkeeping

- [ ] Run `GOWORK=off go test ./... -count=1`
- [ ] Run `GOWORK=off go build ./cmd/tinyidp`
- [ ] Update diary with exact command output
- [ ] Relate changed implementation/docs/example files to the ticket docs
- [ ] Update changelog with implementation summary
- [ ] Run `docmgr doctor --ticket TINYIDP-CONFIG-001 --stale-after 30`
- [ ] Commit implementation and docs
