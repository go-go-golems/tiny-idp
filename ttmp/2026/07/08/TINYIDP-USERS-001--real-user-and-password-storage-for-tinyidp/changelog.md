# Changelog

## 2026-07-08

- Initial workspace created


## 2026-07-08

Created real user/password storage ticket, implementation guide, investigation diary, and phase tasks.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/domain/types.go — User model evidence
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Strict login verification integration target
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/scenario/seeded_users.go — Dev seeded password compatibility evidence


## 2026-07-08

Uploaded user/password storage guide bundle to reMarkable at /ai/2026/07/08/TINYIDP-USERS-001.


## 2026-07-08

Step 2: added Argon2id password hashing, credential/account-security domain models, storage interfaces, memory store support, SQLite migration/store support, and store suite coverage (commit 24e0323).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/passwordhash/argon2id.go — Password hashing primitive
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/store/sqlite/migrations/002_password_credentials.sql — Credential persistence schema


## 2026-07-08

Step 3: added password authentication service, lockout/audit behavior, provider authenticator hook, strict login password verification, and embeddable authenticator option (commit ff5f30c).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/authn/password.go — Authentication service
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Login integration


## 2026-07-08

Step 4: added admin user/password service, tinyidp admin user commands, operator docs, and full validation with go test ./... plus scripts/run-conformance.sh (commit 3b3a155).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/docs/users-and-passwords.md — Documentation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/admin/users.go — Admin service
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/admin.go — Admin CLI


## 2026-07-08

Ticket closed

