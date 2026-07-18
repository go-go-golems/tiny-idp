# Changelog

## 2026-07-09

- Initial workspace created


## 2026-07-09

Fixed PR 3 strict provider review findings: disabled-client revalidation, production token-secret enforcement, seeded strict passwords, discovery metadata accuracy, and max_age=0 fresh-auth semantics (commit d2664c8).

### Related Files

- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/cmds/serve.go — Seeded passwords converted to credentials
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/provider.go — Production secret validation and disabled-client filtering
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/internal/fositeadapter/sqlstore.go — Disabled clients rejected for direct lookup and restored requester sessions
- /home/manuel/workspaces/2026-07-07/prod-tiny-idp/tiny-idp/ttmp/2026/07/09/TINYIDP-PR3-REVIEW-001--address-pr-3-strict-provider-review-findings/design-doc/01-strict-provider-review-findings-and-fixes.md — Textbook-style report for the review fixes


## 2026-07-09

Ticket closed

