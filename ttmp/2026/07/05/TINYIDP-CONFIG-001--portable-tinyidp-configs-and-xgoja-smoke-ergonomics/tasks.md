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

- [x] Create `examples/configs/dev-root.yaml`
- [x] Include root issuer, loopback addr, dev-client, and default dev redirects in `dev-root.yaml`
- [x] Create `examples/configs/personal-inbox-root.yaml`
- [x] Include root issuer, personal-inbox client ID, redirects, and users file in `personal-inbox-root.yaml`
- [x] Create `examples/configs/personal-inbox-realm.yaml`
- [x] Include path-based issuer and matching loopback addr in `personal-inbox-realm.yaml`
- [x] Create `examples/configs/public-spa-pkce.yaml`
- [x] Use builtin `public-spa` client and document PKCE-required behavior in comments
- [x] Create `examples/configs/confidential-web-app.yaml`
- [x] Use builtin `web-app` client and configured `dev-secret` in comments/config
- [x] Create `examples/users/personal-inbox-users.yaml`
- [x] Include Alice and Bob with fixed subjects, emails, passwords, groups, roles, tenant, preferred usernames, and locale
- [x] Keep all example config files provider-neutral except optional path-based issuer compatibility notes

## Phase 2 — Relative path and smoke ergonomics docs

- [x] Document that `oidc.users-file` is currently resolved relative to the process working directory
- [x] Document the safest invocation pattern: run from repo root or use an absolute users-file path
- [x] Add root-issuer xgoja Step 06 snippet
- [x] Add path-issuer xgoja Step 06 snippet without requiring provider-specific claim shapes
- [x] Add Step 07 note for Alice/Bob isolation with the same users file
- [x] Add Step 08 note that device authorization remains xgoja-native, not tinyidp-hosted
- [x] Document common failure symptoms and fixes
- [x] Link examples from root README
- [x] Update Glazed getting-started page
- [x] Update Glazed reference page

## Phase 3 — Example config validation

- [x] Validate `tinyidp print-config --config-file examples/configs/dev-root.yaml`
- [x] Validate `tinyidp print-config --config-file examples/configs/personal-inbox-root.yaml`
- [x] Validate `tinyidp print-config --config-file examples/configs/personal-inbox-realm.yaml`
- [x] Validate `tinyidp print-config --config-file examples/configs/public-spa-pkce.yaml`
- [x] Validate `tinyidp print-config --config-file examples/configs/confidential-web-app.yaml`
- [x] Validate root discovery with `dev-root.yaml`
- [x] Validate path-based discovery with `personal-inbox-realm.yaml`
- [x] Capture exact validation output in diary

## Phase 4 — xgoja smoke decision and validation

- [x] Decide whether this ticket adds Makefile targets or documents manual overrides only
- [x] Validate xgoja Step 06 root issuer smoke if practical in this slice
- [x] Validate xgoja Step 06 path issuer smoke if practical in this slice
- [x] If xgoja validation is deferred, record exact reason and command to run later

## Phase 5 — Final validation and bookkeeping

- [x] Run `GOWORK=off go test ./... -count=1`
- [x] Run `GOWORK=off go build ./cmd/tinyidp`
- [x] Update diary with exact command output
- [x] Relate changed implementation/docs/example files to the ticket docs
- [x] Update changelog with implementation summary
- [x] Run `docmgr doctor --ticket TINYIDP-CONFIG-001 --stale-after 30`
- [x] Commit implementation and docs
