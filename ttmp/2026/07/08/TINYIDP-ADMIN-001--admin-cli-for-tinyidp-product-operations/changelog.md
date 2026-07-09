# Changelog

## 2026-07-08

- Initial workspace created


## 2026-07-08

Created admin CLI ticket, implementation guide, investigation diary, and phase tasks.

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/cmd/tinyidp/main.go — CLI command tree integration target
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/domain/types.go — Administered domain models
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/storage/interfaces.go — Admin service storage boundary


## 2026-07-08

Uploaded admin CLI guide bundle to reMarkable at /ai/2026/07/08/TINYIDP-ADMIN-001.


## 2026-07-08

Implemented admin command tree for init, migrate, doctor, clients, users, keys, backups, and sanitized diagnostics; validation passed with go test ./... and scripts/run-conformance.sh (commits 3b3a155, 6b974d8, 167b444).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/docs/admin-cli.md — Admin CLI documentation
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/admin/clients.go — Client lifecycle operations
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/admin/keys.go — Key lifecycle operations
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/admin_export.go — Diagnostics export
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/admin_ops.go — Init/migrate/doctor commands


## 2026-07-08

Ticket closed

