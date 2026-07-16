# Changelog

## 2026-07-16

- Initial workspace created


## 2026-07-16

Step 1: mapped xapp/device/introspection architecture and published implementation design

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/design-doc/01-xgoja-device-authorization-durable-object-api-analysis-design-and-implementation-guide.md — Initial architecture and phased design


## 2026-07-16

Step 2: implemented resource authentication, client bootstrap, and device-authorized BBS API (commits 5e6d279, 4699d40)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/device_api.go — Host bearer API security boundary


## 2026-07-16

Step 3: added device-login and cached bearer BBS CLI commands (commit d474d3f)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/device_cli.go — Terminal device authorization implementation


## 2026-07-16

Step 4: added deterministic CLI tests and a live tmux smoke harness (commit b92d907)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/device_cli_test.go — Phase 4 verification

## 2026-07-16

Step 5: completed browser, lifecycle, two-user, malformed-request, and TLS verification (commit 748fef8)

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp-xapp/phase5_test.go — Application-level bearer security matrix
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/16/TINYIDP-XAPP-DEVICE-001--xgoja-durable-object-api-device-authorization-cli-example/scripts/playwright_browser_smoke.py — Reproducible browser smoke test
