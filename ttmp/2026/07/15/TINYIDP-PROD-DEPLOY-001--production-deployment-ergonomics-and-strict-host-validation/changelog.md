# Changelog

## 2026-07-15

- Initial workspace created


## 2026-07-15

Established explicit serve-dev versus serve-production command boundary, repeatable local strict-host provisioning, systemd template, and honest browser smoke evidence; device redemption responsiveness remains a release blocker.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve.go — Renamed local-only command
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/15/TINYIDP-PROD-DEPLOY-001--production-deployment-ergonomics-and-strict-host-validation/reference/01-investigation-and-implementation-diary.md — Evidence and release blocker

## 2026-07-15

Fixed strict device-token self-deadlock: ID-token signing now precedes the one-connection SQLite transaction; added regression coverage and verified real browser approval, token redemption, replay rejection, and readiness.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_token_handler.go — Transaction/signing ordering fix
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore_test.go — SQLite regression test
