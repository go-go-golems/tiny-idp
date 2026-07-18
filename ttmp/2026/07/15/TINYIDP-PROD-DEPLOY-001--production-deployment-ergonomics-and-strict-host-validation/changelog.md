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

## 2026-07-15

Added SQLite-backed device browser-continuation regression: approval form, token issuance, UserInfo, replay rejection, and bounded deadlock detection.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/device_authorization_test.go — Durable full-flow regression

## 2026-07-15 - Verified durable device-grant restart semantics

Added and passed the SQLite close/reopen redemption and replay regression gate; the prior strict-host post-approval deadlock blocker is resolved.

### Related Files

- internal/fositeadapter/sqlstore_test.go — Restart and replay coverage

## 2026-07-15 - Completed local static and dynamic release gates

Ran and recorded full tests, build, lint, project-specific audit AST analysis, and dependency vulnerability analysis; fixed the CSP verifier, lint findings, audit classifications, and an external-example private import.

### Related Files

- internal/fositeadapter/device_token_handler.go — Device transaction assurance
- ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint/main.go — Static-analysis assurance

## 2026-07-15 - Completed strict-host backup and restore drill

Added reusable online-backup and offline-restore scripts; verified a live backup, durable rollback preservation, restored-state doctor, and TLS readiness after restart.

### Related Files

- scripts/04-create-online-backup.sh — Online backup evidence
- scripts/05-offline-restore-drill.sh — Restore and rollback evidence
